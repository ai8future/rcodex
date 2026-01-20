Date Created: 2026-01-16 23:38:20 +0100
Date Updated: 2026-01-17
TOTAL_SCORE: 45/100

## Tests Implemented (2026-01-17)
- ~~pkg/orchestrator/condition_test.go~~ - IMPLEMENTED: Condition parsing and evaluation
- ~~pkg/orchestrator/context_test.go~~ - IMPLEMENTED: Variable resolution, thread safety
- ~~pkg/executor/vote_test.go~~ - IMPLEMENTED: Vote decisions, step name extraction
- ~~pkg/envelope/envelope_test.go~~ - IMPLEMENTED: Builder integrity tests
- ~~pkg/runner/flags_test.go~~ - IMPLEMENTED: Flag parsing, duplicate detection, var extraction
**Score Rationale**
- Core orchestration/executor/tracking/reporting code has no unit coverage, so regressions in cost parsing, step conditions, and report output are likely.
- Only 7 test files exist and focus on a narrow slice (bundle, runner, settings, workspace, lock, claude tool).
- File parsing/report helpers and live/progress rendering are untested despite being logic-heavy and error-prone.
- No automated tests for `dashboard/` or `scheduler/`, leaving UI and daemon routes unvalidated.
**Untested Areas**
- `pkg/reports/*`: review gating and cleanup logic.
- `pkg/orchestrator/*`: condition evaluation, context resolution, report helpers, live/progress output, and final report generation.
- `pkg/executor/*`: cost/session parsing, merge/vote aggregation.
- `pkg/envelope/*` and `pkg/tracking/*`: builder/formatting and status parsing.
- `pkg/tools/codex`, `pkg/tools/gemini`: command construction and defaults (not covered by tests below).
**Proposed Unit Tests**
- Reports: verify review detection within first 10 lines, newest selection, cleanup retention, and skip/run gating.
- Orchestrator: condition parsing (AND/OR, numeric compares, contains), context resolve for inputs/steps/output refs, streaming-result extraction, and report helper parsing (titles, openings, grade/overview extraction, categorization, output stats).
- Display helpers: `stripAnsi` and `extractMeaningfulContent` output mapping.
- Executor: cost/session parsing for Claude/Codex/Gemini; merge/vote aggregation and decision logic.
- Envelope: builder integrity for success/failure paths.
- Tracking: FormatCredit, iTerm error classification, JSON parsing via helper process.
- Follow-up (not in diff): integration tests for `Orchestrator.Run` and `ToolExecutor.Execute` using test fakes or injected command runners; UI tests for `dashboard/` routes and table rendering.
**Patch-Ready Diffs**
```diff
diff --git a/pkg/reports/manager_test.go b/pkg/reports/manager_test.go
new file mode 100644
index 0000000..add1a39
--- /dev/null
+++ b/pkg/reports/manager_test.go
@@ -0,0 +1,107 @@
+package reports
+
+import (
+	"os"
+	"path/filepath"
+	"strings"
+	"testing"
+	"time"
+)
+
+func writeReport(t *testing.T, dir, name, contents string, modTime time.Time) string {
+	t.Helper()
+	path := filepath.Join(dir, name)
+	if err := os.WriteFile(path, []byte(contents), 0644); err != nil {
+		t.Fatalf("write report: %v", err)
+	}
+	if !modTime.IsZero() {
+		if err := os.Chtimes(path, modTime, modTime); err != nil {
+			t.Fatalf("chtimes: %v", err)
+		}
+	}
+	return path
+}
+
+func TestShouldSkipTask(t *testing.T) {
+	if got := ShouldSkipTask(filepath.Join(t.TempDir(), "missing"), "r", "report-", true); got {
+		t.Fatalf("expected no skip when report dir missing")
+	}
+	if got := ShouldSkipTask(t.TempDir(), "r", "", true); got {
+		t.Fatalf("expected no skip when pattern empty")
+	}
+	if got := ShouldSkipTask(t.TempDir(), "r", "report-", false); got {
+		t.Fatalf("expected no skip when review not required")
+	}
+
+	t.Run("unreviewed", func(t *testing.T) {
+		dir := t.TempDir()
+		writeReport(t, dir, "report-1.md", "Header\nBody\n", time.Now())
+		if got := ShouldSkipTask(dir, "r", "report-", true); !got {
+			t.Fatalf("expected skip for unreviewed report")
+		}
+	})
+
+	t.Run("reviewed", func(t *testing.T) {
+		dir := t.TempDir()
+		writeReport(t, dir, "report-1.md", "Date Modified: 2025-01-01\n", time.Now())
+		if got := ShouldSkipTask(dir, "r", "report-", true); got {
+			t.Fatalf("expected no skip for reviewed report")
+		}
+	})
+}
+
+func TestFindNewestReport(t *testing.T) {
+	dir := t.TempDir()
+	oldTime := time.Now().Add(-2 * time.Hour)
+	newTime := time.Now().Add(-1 * time.Hour)
+
+	oldPath := writeReport(t, dir, "report-old.md", "old", oldTime)
+	newPath := writeReport(t, dir, "report-new.md", "new", newTime)
+
+	got := FindNewestReport([]string{oldPath, newPath})
+	if got != newPath {
+		t.Fatalf("expected newest %q, got %q", newPath, got)
+	}
+}
+
+func TestIsReportReviewed(t *testing.T) {
+	dir := t.TempDir()
+	reviewed := writeReport(t, dir, "reviewed.md", "Date Modified: 2024-01-01\n", time.Now())
+	if !IsReportReviewed(reviewed) {
+		t.Fatalf("expected report to be reviewed")
+	}
+
+	lines := make([]string, 0, 11)
+	for i := 0; i < 10; i++ {
+		lines = append(lines, "Line")
+	}
+	lines = append(lines, "Date Modified: 2024-01-01")
+	unreviewed := writeReport(t, dir, "late.md", strings.Join(lines, "\n"), time.Now().Add(time.Second))
+	if IsReportReviewed(unreviewed) {
+		t.Fatalf("expected report with late marker to be unreviewed")
+	}
+}
+
+func TestDeleteOldReports(t *testing.T) {
+	dir := t.TempDir()
+	t1 := time.Now().Add(-3 * time.Hour)
+	t2 := time.Now().Add(-2 * time.Hour)
+	t3 := time.Now().Add(-1 * time.Hour)
+
+	writeReport(t, dir, "report-1.md", "one", t1)
+	writeReport(t, dir, "report-2.md", "two", t2)
+	newest := writeReport(t, dir, "report-3.md", "three", t3)
+
+	DeleteOldReports(dir, []string{"r"}, map[string]string{"r": "report-"})
+
+	matches, err := filepath.Glob(filepath.Join(dir, "report-*.md"))
+	if err != nil {
+		t.Fatalf("glob: %v", err)
+	}
+	if len(matches) != 1 {
+		t.Fatalf("expected 1 report remaining, got %d", len(matches))
+	}
+	if matches[0] != newest {
+		t.Fatalf("expected newest report %q, got %q", newest, matches[0])
+	}
+}
+
diff --git a/pkg/orchestrator/condition_test.go b/pkg/orchestrator/condition_test.go
new file mode 100644
index 0000000..8415585
--- /dev/null
+++ b/pkg/orchestrator/condition_test.go
@@ -0,0 +1,35 @@
+package orchestrator
+
+import "testing"
+
+func TestEvaluateCondition(t *testing.T) {
+	ctx := NewContext(map[string]string{
+		"num":  "10",
+		"word": "hello",
+	})
+
+	cases := []struct {
+		expr string
+		want bool
+	}{
+		{"", true},
+		{"true", true},
+		{"false", false},
+		{"${inputs.num} > 5", true},
+		{"${inputs.num} < 5", false},
+		{"${inputs.num} >= 10", true},
+		{"${inputs.num} <= 9", false},
+		{"${inputs.word} == hello", true},
+		{"${inputs.word} != world", true},
+		{"${inputs.word} contains ell", true},
+		{"true AND false", false},
+		{"true OR false", true},
+		{"abc > 1", false},
+	}
+
+	for _, tc := range cases {
+		if got := EvaluateCondition(tc.expr, ctx); got != tc.want {
+			t.Fatalf("EvaluateCondition(%q) = %v, want %v", tc.expr, got, tc.want)
+		}
+	}
+}
+
diff --git a/pkg/orchestrator/context_test.go b/pkg/orchestrator/context_test.go
new file mode 100644
index 0000000..5ff4bcf
--- /dev/null
+++ b/pkg/orchestrator/context_test.go
@@ -0,0 +1,84 @@
+package orchestrator
+
+import (
+	"encoding/json"
+	"os"
+	"path/filepath"
+	"strings"
+	"testing"
+
+	"rcodegen/pkg/envelope"
+)
+
+func writeOutputFile(t *testing.T, dir string, payload map[string]interface{}) string {
+	t.Helper()
+	path := filepath.Join(dir, "output.json")
+	data, err := json.Marshal(payload)
+	if err != nil {
+		t.Fatalf("marshal: %v", err)
+	}
+	if err := os.WriteFile(path, data, 0644); err != nil {
+		t.Fatalf("write output: %v", err)
+	}
+	return path
+}
+
+func TestResolveInputsAndSteps(t *testing.T) {
+	dir := t.TempDir()
+	stream := `{"type":"result","result":"done"}`
+	outputPath := writeOutputFile(t, dir, map[string]interface{}{
+		"stdout": stream,
+		"stderr": "warn",
+	})
+
+	ctx := NewContext(map[string]string{"name": "Ada"})
+	ctx.SetResult("step1", &envelope.Envelope{
+		Status:    envelope.StatusSuccess,
+		OutputRef: outputPath,
+		Result:    map[string]interface{}{"answer": "42"},
+	})
+
+	if got := ctx.Resolve("Hello ${inputs.name}"); got != "Hello Ada" {
+		t.Fatalf("resolve inputs: %q", got)
+	}
+	if got := ctx.Resolve("${steps.step1.output_ref}"); got != outputPath {
+		t.Fatalf("resolve output_ref: %q", got)
+	}
+	if got := ctx.Resolve("${steps.step1.stdout}"); got != "done" {
+		t.Fatalf("resolve stdout: %q", got)
+	}
+	if got := ctx.Resolve("${steps.step1.stderr}"); got != "warn" {
+		t.Fatalf("resolve stderr: %q", got)
+	}
+
+	gotResult := ctx.Resolve("${steps.step1.result}")
+	var decoded map[string]interface{}
+	if err := json.Unmarshal([]byte(gotResult), &decoded); err != nil {
+		t.Fatalf("unmarshal result: %v", err)
+	}
+	if decoded["answer"] != "42" {
+		t.Fatalf("expected result.answer 42, got %v", decoded["answer"])
+	}
+
+	if got := ctx.Resolve("${steps.step1.result.answer}"); got != "42" {
+		t.Fatalf("resolve result field: %q", got)
+	}
+	if got := ctx.Resolve("${inputs.missing}"); got != "${inputs.missing}" {
+		t.Fatalf("expected unresolved placeholder, got %q", got)
+	}
+}
+
+func TestExtractStreamingResult(t *testing.T) {
+	content := strings.Join([]string{
+		`{"type":"message","text":"ignore"}`,
+		`{"type":"result","result":"final"}`,
+	}, "\n")
+	if got := extractStreamingResult(content); got != "final" {
+		t.Fatalf("expected final result, got %q", got)
+	}
+
+	plain := "plain output"
+	if got := extractStreamingResult(plain); got != plain {
+		t.Fatalf("expected plain output, got %q", got)
+	}
+}
+
diff --git a/pkg/orchestrator/progress_test.go b/pkg/orchestrator/progress_test.go
new file mode 100644
index 0000000..df23953
--- /dev/null
+++ b/pkg/orchestrator/progress_test.go
@@ -0,0 +1,18 @@
+package orchestrator
+
+import (
+	"testing"
+	"time"
+)
+
+func TestFormatDuration(t *testing.T) {
+	if got := formatDuration(45 * time.Second); got != "45s" {
+		t.Fatalf("expected 45s, got %q", got)
+	}
+	if got := formatDuration(2 * time.Minute); got != "2m" {
+		t.Fatalf("expected 2m, got %q", got)
+	}
+	if got := formatDuration(2*time.Minute+5*time.Second); got != "2m 5s" {
+		t.Fatalf("expected 2m 5s, got %q", got)
+	}
+}
+
diff --git a/pkg/orchestrator/live_display_test.go b/pkg/orchestrator/live_display_test.go
new file mode 100644
index 0000000..491f794
--- /dev/null
+++ b/pkg/orchestrator/live_display_test.go
@@ -0,0 +1,31 @@
+package orchestrator
+
+import "testing"
+
+func TestStripAnsi(t *testing.T) {
+	input := "\033[31mred\033[0m"
+	if got := stripAnsi(input); got != "red" {
+		t.Fatalf("expected red, got %q", got)
+	}
+}
+
+func TestExtractMeaningfulContent(t *testing.T) {
+	cases := []struct {
+		name string
+		line string
+		want string
+	}{
+		{"system", `{"type":"system"}`, ""},
+		{"result", `{"type":"result","result":"x"}`, ""},
+		{"tool_use", `{"type":"tool_use","name":"Read"}`, "\U0001F4D6 Reading files..."},
+		{"assistant_text", `{"type":"assistant","message":{"content":[{"type":"text","text":"Hello\\nworld"}]}}`, "Hello world"},
+		{"tool_result", `{"type":"tool_result"}`, "\U0001F4CB Processing result..."},
+		{"status_line", "Working on it", "Working on it"},
+	}
+
+	for _, tc := range cases {
+		if got := extractMeaningfulContent(tc.line); got != tc.want {
+			t.Fatalf("%s: got %q, want %q", tc.name, got, tc.want)
+		}
+	}
+}
+
diff --git a/pkg/orchestrator/report_helpers_test.go b/pkg/orchestrator/report_helpers_test.go
new file mode 100644
index 0000000..369bfbe
--- /dev/null
+++ b/pkg/orchestrator/report_helpers_test.go
@@ -0,0 +1,263 @@
+package orchestrator
+
+import (
+	"os"
+	"path/filepath"
+	"reflect"
+	"strings"
+	"testing"
+)
+
+func writeFile(t *testing.T, dir, name, content string) string {
+	t.Helper()
+	path := filepath.Join(dir, name)
+	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
+		t.Fatalf("mkdir: %v", err)
+	}
+	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
+		t.Fatalf("write file: %v", err)
+	}
+	return path
+}
+
+func TestBasicHelpers(t *testing.T) {
+	if got := capitalize(""); got != "" {
+		t.Fatalf("capitalize empty: %q", got)
+	}
+	if got := capitalize("hello"); got != "Hello" {
+		t.Fatalf("capitalize: %q", got)
+	}
+	if got := truncate("short", 10); got != "short" {
+		t.Fatalf("truncate short: %q", got)
+	}
+	if got := truncate("hello world", 8); got != "hello..." {
+		t.Fatalf("truncate long: %q", got)
+	}
+	if got := max(1, 5, 3); got != 5 {
+		t.Fatalf("max: %d", got)
+	}
+}
+
+func TestExtractOpeningSummary(t *testing.T) {
+	dir := t.TempDir()
+	path := writeFile(t, dir, "summary.md", "Alice, engineer, morning.\nSecond line")
+	if got := extractOpeningSummary(path); got != "Alice, engineer" {
+		t.Fatalf("opening summary: %q", got)
+	}
+
+	longLine := "This is a very long opening line that exceeds forty characters."
+	path = writeFile(t, dir, "long.md", longLine)
+	if got := extractOpeningSummary(path); got != "This is a very long opening line that..." {
+		t.Fatalf("opening summary long: %q", got)
+	}
+}
+
+func TestExtractAngleDataTone(t *testing.T) {
+	dir := t.TempDir()
+	anglePath := writeFile(t, dir, "angle.md", "Systemic optimization issue with economic costs.")
+	if got := extractAngle(anglePath); got != "Systemic critique, optimization trap" {
+		t.Fatalf("angle: %q", got)
+	}
+
+	dataPath := writeFile(t, dir, "data.md", "We saw 10% gains and 20% losses.")
+	if got := extractDataPoint(dataPath); got != "Statistics (10%, 20%)" {
+		t.Fatalf("data point: %q", got)
+	}
+
+	studyPath := writeFile(t, dir, "study.md", "Recent research shows improvements.")
+	if got := extractDataPoint(studyPath); got != "Research-backed" {
+		t.Fatalf("data point fallback: %q", got)
+	}
+
+	tonePath := writeFile(t, dir, "tone.md", "Builder critical perspective.")
+	if got := extractTone(tonePath); got != "Builder-focused, Critical" {
+		t.Fatalf("tone: %q", got)
+	}
+}
+
+func TestArticleHelpers(t *testing.T) {
+	dir := t.TempDir()
+	writeFile(t, dir, "style-guide.md", "skip")
+	writeFile(t, dir, "draft-codex.md", "skip")
+	writeFile(t, dir, "Run Report.md", "skip")
+	codexPath := writeFile(t, dir, "My Article - Codex.md", "content")
+	geminiPath := writeFile(t, dir, "My Article - Gemini.md", "content")
+
+	articles := findArticleFilesInDir(dir)
+	if len(articles) != 2 {
+		t.Fatalf("expected 2 articles, got %d", len(articles))
+	}
+
+	if got := findArticleByTool(articles, "codex"); got != codexPath {
+		t.Fatalf("findArticleByTool codex: %q", got)
+	}
+	if got := findArticleByTool(articles, "gemini"); got != geminiPath {
+		t.Fatalf("findArticleByTool gemini: %q", got)
+	}
+
+	names := getArticleNames([]string{codexPath, geminiPath, filepath.Join(dir, "Other.md")})
+	expected := []string{"Codex", "Gemini", "Other"}
+	if !reflect.DeepEqual(names, expected) {
+		t.Fatalf("article names: %v", names)
+	}
+}
+
+func TestExtractTitleAndOpening(t *testing.T) {
+	dir := t.TempDir()
+	path := writeFile(t, dir, "article.md", "# Hello World\n\nThis is a sentence. Another one.")
+	if got := extractTitle(path); got != "Hello World" {
+		t.Fatalf("extractTitle: %q", got)
+	}
+	if got := extractOpening(path); got != "This is a sentence." {
+		t.Fatalf("extractOpening: %q", got)
+	}
+
+	plainPath := writeFile(t, dir, "plain.md", "No title here")
+	if got := extractTitle(plainPath); got != "plain.md" {
+		t.Fatalf("extractTitle fallback: %q", got)
+	}
+}
+
+func TestExtractOverviewAndGrade(t *testing.T) {
+	dir := t.TempDir()
+	summary := "# Title\n\n## Overview\nFirst line.\nSecond line.\n\n## Next\nSkip"
+	summaryPath := writeFile(t, dir, "IMPLEMENTATION_SUMMARY.md", summary)
+	if got := extractOverviewFromSummary(summaryPath); got != "First line. Second line." {
+		t.Fatalf("overview: %q", got)
+	}
+
+	report := "Text\n```json\n{\"grade\":{\"score\":88,\"letter\":\"B\"}}\n```"
+	reportPath := writeFile(t, dir, "final-report.md", report)
+	grade := extractGradeFromReport(reportPath)
+	if grade == nil || grade.Score != 88 || grade.Letter != "B" {
+		t.Fatalf("grade: %+v", grade)
+	}
+
+	report2 := "```json\n{\"score\":77,\"letter\":\"C\"}\n```"
+	reportPath2 := writeFile(t, dir, "final-report-2.md", report2)
+	grade2 := extractGradeFromReport(reportPath2)
+	if grade2 == nil || grade2.Score != 77 || grade2.Letter != "C" {
+		t.Fatalf("grade2: %+v", grade2)
+	}
+}
+
+func TestCategorizeAndScanOutputFiles(t *testing.T) {
+	cases := map[string]string{
+		"src/main.go":          "source",
+		"lib/util.ts":          "source",
+		"samples/example.txt":  "sample",
+		"final-report.md":      "report",
+		"README.md":            "docs",
+		"config.json":          "config",
+		"notes.txt":            "other",
+		"output.pdf":           "output",
+	}
+
+	for path, want := range cases {
+		if got := categorizeFile(path); got != want {
+			t.Fatalf("categorizeFile(%q) = %q, want %q", path, got, want)
+		}
+	}
+
+	dir := t.TempDir()
+	writeFile(t, dir, filepath.Join("src", "main.go"), "line1\nline2")
+	writeFile(t, dir, filepath.Join("docs", "readme.md"), "hello world")
+	writeFile(t, dir, "final-report.md", "score 100")
+	writeFile(t, dir, "config.json", "{}")
+
+	files, stats := scanOutputFiles(dir)
+	if stats.TotalSourceFiles != 1 || stats.TotalSourceLines != 2 {
+		t.Fatalf("source stats: %+v", stats)
+	}
+	if stats.TotalDocWords != 4 {
+		t.Fatalf("doc stats: %+v", stats)
+	}
+
+	fileTypes := map[string]string{}
+	for _, f := range files {
+		fileTypes[f.Path] = f.Type
+	}
+	if fileTypes[filepath.Join("src", "main.go")] != "source" {
+		t.Fatalf("file type src/main.go: %q", fileTypes[filepath.Join("src", "main.go")])
+	}
+	if fileTypes["final-report.md"] != "report" {
+		t.Fatalf("file type final-report.md: %q", fileTypes["final-report.md"])
+	}
+}
+
+func TestFindStepOutput(t *testing.T) {
+	if got := findStepOutput("draft"); got != "`docs/draft.md`" {
+		t.Fatalf("findStepOutput draft: %q", got)
+	}
+	if got := findStepOutput("unknown"); got != "-" {
+		t.Fatalf("findStepOutput unknown: %q", got)
+	}
+}
+
+func TestGetVersion(t *testing.T) {
+	dir := t.TempDir()
+	if err := os.WriteFile(filepath.Join(dir, "VERSION"), []byte("9.9.9\n"), 0644); err != nil {
+		t.Fatalf("write version: %v", err)
+	}
+	wd, err := os.Getwd()
+	if err != nil {
+		t.Fatalf("getwd: %v", err)
+	}
+	defer os.Chdir(wd)
+	if err := os.Chdir(dir); err != nil {
+		t.Fatalf("chdir: %v", err)
+	}
+	if got := getVersion(); got != "9.9.9" {
+		t.Fatalf("getVersion: %q", got)
+	}
+}
+
+func TestCountWords(t *testing.T) {
+	dir := t.TempDir()
+	path := writeFile(t, dir, "words.md", "one two\nthree")
+	if got := countWords(path); got != 3 {
+		t.Fatalf("countWords: %d", got)
+	}
+}
+
+func TestExtractOpeningTruncate(t *testing.T) {
+	dir := t.TempDir()
+	line := "This line is quite long and has no sentence break for truncation"
+	path := writeFile(t, dir, "opening.md", "# Title\n\n"+line)
+	expected := line[:47] + "..."
+	if got := extractOpening(path); got != expected {
+		t.Fatalf("extractOpening truncate: %q", got)
+	}
+}
+
+func TestGetArticleNames(t *testing.T) {
+	dir := t.TempDir()
+	paths := []string{
+		filepath.Join(dir, "Cool - Codex.md"),
+		filepath.Join(dir, "Nice - Gemini.md"),
+		filepath.Join(dir, "Other.md"),
+	}
+	got := getArticleNames(paths)
+	want := []string{"Codex", "Gemini", "Other"}
+	if !reflect.DeepEqual(got, want) {
+		t.Fatalf("getArticleNames: %v", got)
+	}
+}
+
+func TestExtractOpeningSummaryFallback(t *testing.T) {
+	dir := t.TempDir()
+	path := writeFile(t, dir, "fallback.md", "Short line")
+	if got := extractOpeningSummary(path); got != "Short line" {
+		t.Fatalf("opening summary fallback: %q", got)
+	}
+}
+
+func TestExtractOverviewTruncate(t *testing.T) {
+	dir := t.TempDir()
+	parts := []string{"## Overview", strings.Repeat("word ", 30), "## End"}
+	path := writeFile(t, dir, "summary-long.md", strings.Join(parts, "\n"))
+	overview := extractOverviewFromSummary(path)
+	if len(overview) > 100 {
+		t.Fatalf("overview length: %d", len(overview))
+	}
+}
+
diff --git a/pkg/executor/tool_test.go b/pkg/executor/tool_test.go
new file mode 100644
index 0000000..b98e052
--- /dev/null
+++ b/pkg/executor/tool_test.go
@@ -0,0 +1,65 @@
+package executor
+
+import (
+	"math"
+	"testing"
+)
+
+func TestExtractCostInfoClaude(t *testing.T) {
+	stdout := "{\"type\":\"message\"}\n{\"type\":\"result\",\"total_cost_usd\":1.23,\"usage\":{\"input_tokens\":10,\"output_tokens\":5,\"cache_read_input_tokens\":2,\"cache_creation_input_tokens\":3}}"
+	usage := extractCostInfo("claude", stdout, "")
+	if usage.CostUSD != 1.23 {
+		t.Fatalf("cost: %f", usage.CostUSD)
+	}
+	if usage.InputTokens != 10 || usage.OutputTokens != 5 {
+		t.Fatalf("tokens: in=%d out=%d", usage.InputTokens, usage.OutputTokens)
+	}
+	if usage.CacheReadTokens != 2 || usage.CacheWriteTokens != 3 {
+		t.Fatalf("cache tokens: read=%d write=%d", usage.CacheReadTokens, usage.CacheWriteTokens)
+	}
+}
+
+func TestExtractCostInfoCodex(t *testing.T) {
+	stderr := "tokens used\n7,476\n"
+	usage := extractCostInfo("codex", "", stderr)
+	if usage.InputTokens != 5233 || usage.OutputTokens != 2242 {
+		t.Fatalf("tokens: in=%d out=%d", usage.InputTokens, usage.OutputTokens)
+	}
+	expected := 0.11959
+	if diff := math.Abs(usage.CostUSD - expected); diff > 1e-6 {
+		t.Fatalf("cost: got=%f want=%f", usage.CostUSD, expected)
+	}
+}
+
+func TestExtractCostInfoGemini(t *testing.T) {
+	stdout := "{\"type\":\"result\",\"stats\":{\"input_tokens\":12,\"output_tokens\":4,\"cached\":3}}"
+	usage := extractCostInfo("gemini", stdout, "")
+	if usage.InputTokens != 12 || usage.OutputTokens != 4 {
+		t.Fatalf("tokens: in=%d out=%d", usage.InputTokens, usage.OutputTokens)
+	}
+	if usage.CacheReadTokens != 3 {
+		t.Fatalf("cache read: %d", usage.CacheReadTokens)
+	}
+	expected := 0.000012
+	if diff := math.Abs(usage.CostUSD - expected); diff > 1e-9 {
+		t.Fatalf("cost: got=%f want=%f", usage.CostUSD, expected)
+	}
+}
+
+func TestExtractSessionID(t *testing.T) {
+	stdoutClaude := "{\"type\":\"system\",\"session_id\":\"abc\"}"
+	if got := extractSessionID("claude", stdoutClaude, ""); got != "abc" {
+		t.Fatalf("claude session id: %q", got)
+	}
+	stdoutGemini := "{\"type\":\"init\",\"session_id\":\"xyz\"}"
+	if got := extractSessionID("gemini", stdoutGemini, ""); got != "xyz" {
+		t.Fatalf("gemini session id: %q", got)
+	}
+	stderrCodex := "session id: 1234-5678"
+	if got := extractSessionID("codex", "", stderrCodex); got != "1234-5678" {
+		t.Fatalf("codex session id: %q", got)
+	}
+	if got := extractSessionID("codex", "", "no session"); got != "" {
+		t.Fatalf("expected empty session id, got %q", got)
+	}
+}
+
diff --git a/pkg/executor/merge_vote_test.go b/pkg/executor/merge_vote_test.go
new file mode 100644
index 0000000..932e6cc
--- /dev/null
+++ b/pkg/executor/merge_vote_test.go
@@ -0,0 +1,163 @@
+package executor
+
+import (
+	"encoding/json"
+	"os"
+	"path/filepath"
+	"testing"
+
+	"rcodegen/pkg/bundle"
+	"rcodegen/pkg/envelope"
+	"rcodegen/pkg/orchestrator"
+	"rcodegen/pkg/workspace"
+)
+
+func readOutputMap(t *testing.T, path string) map[string]interface{} {
+	t.Helper()
+	data, err := os.ReadFile(path)
+	if err != nil {
+		t.Fatalf("read output: %v", err)
+	}
+	var out map[string]interface{}
+	if err := json.Unmarshal(data, &out); err != nil {
+		t.Fatalf("unmarshal output: %v", err)
+	}
+	return out
+}
+
+func TestMergeExecutorConcat(t *testing.T) {
+	dir := t.TempDir()
+	fileA := filepath.Join(dir, "a.txt")
+	fileB := filepath.Join(dir, "b.txt")
+	if err := os.WriteFile(fileA, []byte("alpha"), 0644); err != nil {
+		t.Fatalf("write a: %v", err)
+	}
+	if err := os.WriteFile(fileB, []byte("beta"), 0644); err != nil {
+		t.Fatalf("write b: %v", err)
+	}
+
+	ws, err := workspace.New(t.TempDir())
+	if err != nil {
+		t.Fatalf("workspace: %v", err)
+	}
+	ctx := orchestrator.NewContext(nil)
+	step := &bundle.Step{
+		Name: "merge",
+		Merge: &bundle.MergeDef{
+			Inputs:   []string{fileA, fileB},
+			Strategy: "concat",
+		},
+	}
+
+	exec := &MergeExecutor{}
+	env, err := exec.Execute(step, ctx, ws)
+	if err != nil {
+		t.Fatalf("execute: %v", err)
+	}
+	out := readOutputMap(t, env.OutputRef)
+	if got := out["merged"].(string); got != "alpha\n\n---\n\nbeta" {
+		t.Fatalf("merged content: %q", got)
+	}
+	if got := int(out["input_count"].(float64)); got != 2 {
+		t.Fatalf("input_count: %d", got)
+	}
+}
+
+func TestMergeExecutorUnionSkipsMissing(t *testing.T) {
+	dir := t.TempDir()
+	fileA := filepath.Join(dir, "a.txt")
+	if err := os.WriteFile(fileA, []byte("alpha"), 0644); err != nil {
+		t.Fatalf("write a: %v", err)
+	}
+
+	ws, err := workspace.New(t.TempDir())
+	if err != nil {
+		t.Fatalf("workspace: %v", err)
+	}
+	ctx := orchestrator.NewContext(nil)
+	step := &bundle.Step{
+		Name: "merge",
+		Merge: &bundle.MergeDef{
+			Inputs:   []string{fileA, filepath.Join(dir, "missing.txt")},
+			Strategy: "union",
+		},
+	}
+
+	exec := &MergeExecutor{}
+	env, err := exec.Execute(step, ctx, ws)
+	if err != nil {
+		t.Fatalf("execute: %v", err)
+	}
+	out := readOutputMap(t, env.OutputRef)
+	if got := out["merged"].(string); got != "alpha" {
+		t.Fatalf("merged content: %q", got)
+	}
+	if got := int(out["input_count"].(float64)); got != 1 {
+		t.Fatalf("input_count: %d", got)
+	}
+}
+
+func TestVoteExecutorDecisions(t *testing.T) {
+	ws, err := workspace.New(t.TempDir())
+	if err != nil {
+		t.Fatalf("workspace: %v", err)
+	}
+
+	ctx := orchestrator.NewContext(nil)
+	ctx.SetResult("step1", &envelope.Envelope{Status: envelope.StatusSuccess})
+	ctx.SetResult("step2", &envelope.Envelope{Status: envelope.StatusFailure})
+
+	step := &bundle.Step{
+		Name: "vote",
+		Vote: &bundle.VoteDef{
+			Inputs:   []string{"${steps.step1.output_ref}", "${steps.step2.output_ref}"},
+			Strategy: "majority",
+		},
+	}
+
+	env, err := (&VoteExecutor{}).Execute(step, ctx, ws)
+	if err != nil {
+		t.Fatalf("execute: %v", err)
+	}
+	if env.Result["decision"] != "rejected" {
+		t.Fatalf("decision: %v", env.Result["decision"])
+	}
+	votes, ok := env.Result["votes"].(map[string]int)
+	if !ok {
+		t.Fatalf("votes type: %T", env.Result["votes"])
+	}
+	if votes["success"] != 1 || votes["failure"] != 1 {
+		t.Fatalf("votes: %+v", votes)
+	}
+
+	ctx2 := orchestrator.NewContext(nil)
+	ctx2.SetResult("step1", &envelope.Envelope{Status: envelope.StatusSuccess})
+	ctx2.SetResult("step2", &envelope.Envelope{Status: envelope.StatusSuccess})
+	step.Vote.Strategy = "unanimous"
+	step.Vote.Inputs = []string{"${steps.step1.output_ref}", "${steps.step2.output_ref}"}
+	env2, err := (&VoteExecutor{}).Execute(step, ctx2, ws)
+	if err != nil {
+		t.Fatalf("execute: %v", err)
+	}
+	if env2.Result["decision"] != "approved" {
+		t.Fatalf("decision unanimous: %v", env2.Result["decision"])
+	}
+
+	step.Vote.Strategy = "unknown"
+	env3, err := (&VoteExecutor{}).Execute(step, ctx2, ws)
+	if err != nil {
+		t.Fatalf("execute: %v", err)
+	}
+	if env3.Result["decision"] != "unknown" {
+		t.Fatalf("decision unknown: %v", env3.Result["decision"])
+	}
+}
+
+func TestExtractStepName(t *testing.T) {
+	if got := extractStepName("${steps.alpha.output_ref}"); got != "alpha" {
+		t.Fatalf("extractStepName: %q", got)
+	}
+	if got := extractStepName("plain"); got != "plain" {
+		t.Fatalf("extractStepName plain: %q", got)
+	}
+}
+
diff --git a/pkg/envelope/envelope_test.go b/pkg/envelope/envelope_test.go
new file mode 100644
index 0000000..8ae17bc
--- /dev/null
+++ b/pkg/envelope/envelope_test.go
@@ -0,0 +1,39 @@
+package envelope
+
+import "testing"
+
+func TestBuilderSuccess(t *testing.T) {
+	env := New().
+		WithTool("claude").
+		WithOutputRef("/tmp/out.json").
+		WithDuration(123).
+		WithResult("answer", "ok").
+		Success().
+		Build()
+
+	if env.Status != StatusSuccess {
+		t.Fatalf("status: %s", env.Status)
+	}
+	if env.OutputRef != "/tmp/out.json" {
+		t.Fatalf("output ref: %s", env.OutputRef)
+	}
+	if env.Metrics == nil || env.Metrics.Tool != "claude" || env.Metrics.DurationMs != 123 {
+		t.Fatalf("metrics: %+v", env.Metrics)
+	}
+	if env.Result["answer"] != "ok" {
+		t.Fatalf("result: %v", env.Result["answer"])
+	}
+}
+
+func TestBuilderFailure(t *testing.T) {
+	env := New().
+		Failure("CODE", "message").
+		Build()
+
+	if env.Status != StatusFailure {
+		t.Fatalf("status: %s", env.Status)
+	}
+	if env.Error == nil || env.Error.Code != "CODE" || env.Error.Message != "message" {
+		t.Fatalf("error: %+v", env.Error)
+	}
+}
+
diff --git a/pkg/tracking/tracking_test.go b/pkg/tracking/tracking_test.go
new file mode 100644
index 0000000..3cdd420
--- /dev/null
+++ b/pkg/tracking/tracking_test.go
@@ -0,0 +1,75 @@
+package tracking
+
+import (
+	"fmt"
+	"os"
+	"os/exec"
+	"testing"
+)
+
+func TestFormatCredit(t *testing.T) {
+	val := 12
+	if got := FormatCredit(&val); got != "12" {
+		t.Fatalf("FormatCredit: %q", got)
+	}
+	if got := FormatCredit(nil); got != "N/A" {
+		t.Fatalf("FormatCredit nil: %q", got)
+	}
+}
+
+func TestIsITerm2Error(t *testing.T) {
+	if !(&ClaudeStatus{Error: "not_iterm2"}).IsITerm2Error() {
+		t.Fatalf("expected not_iterm2 to be true")
+	}
+	if !(&ClaudeStatus{Error: "no_iterm2_package"}).IsITerm2Error() {
+		t.Fatalf("expected no_iterm2_package to be true")
+	}
+	if (&ClaudeStatus{Error: "other"}).IsITerm2Error() {
+		t.Fatalf("expected other to be false")
+	}
+}
+
+func TestRunStatusScript(t *testing.T) {
+	payload := `{"5h_left":10,"weekly_left":50}`
+	cmd := exec.Command(os.Args[0], "-test.run=TestHelperProcess", "--", payload)
+	cmd.Env = append(os.Environ(), "GO_WANT_HELPER_PROCESS=1")
+	status := runStatusScript(cmd)
+	if status.Error != "" {
+		t.Fatalf("status error: %s", status.Error)
+	}
+	if status.FiveHourLeft == nil || *status.FiveHourLeft != 10 {
+		t.Fatalf("five hour left: %v", status.FiveHourLeft)
+	}
+	if status.WeeklyLeft == nil || *status.WeeklyLeft != 50 {
+		t.Fatalf("weekly left: %v", status.WeeklyLeft)
+	}
+}
+
+func TestRunClaudeStatusScript(t *testing.T) {
+	payload := `{"session_left":5,"weekly_all_left":10}`
+	cmd := exec.Command(os.Args[0], "-test.run=TestHelperProcess", "--", payload)
+	cmd.Env = append(os.Environ(), "GO_WANT_HELPER_PROCESS=1")
+	status := runClaudeStatusScript(cmd)
+	if status.Error != "" {
+		t.Fatalf("status error: %s", status.Error)
+	}
+	if status.SessionLeft == nil || *status.SessionLeft != 5 {
+		t.Fatalf("session left: %v", status.SessionLeft)
+	}
+	if status.WeeklyAllLeft == nil || *status.WeeklyAllLeft != 10 {
+		t.Fatalf("weekly all left: %v", status.WeeklyAllLeft)
+	}
+}
+
+func TestHelperProcess(t *testing.T) {
+	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
+		return
+	}
+	for i, arg := range os.Args {
+		if arg == "--" && i+1 < len(os.Args) {
+			fmt.Fprint(os.Stdout, os.Args[i+1])
+			os.Exit(0)
+		}
+	}
+	os.Exit(0)
+}
+```
