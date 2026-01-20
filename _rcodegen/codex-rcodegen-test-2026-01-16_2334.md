Date Created: 2026-01-16 23:34:03 +0100
TOTAL_SCORE: 68/100

## Score Rationale
- Strengths: clear package boundaries, many pure helpers, cohesive tool abstractions.
- Gaps: limited automated tests outside a few packages; core orchestrator/executor paths, parsing, and concurrency are untested; command-building logic and status parsing lack validation; heavy I/O paths are unverified.

## Untested Areas (high risk)
- pkg/reports: review gating, newest report selection, cleanup.
- pkg/envelope: builder correctness and metrics wiring.
- pkg/executor: dispatcher routing, parallel aggregation, merge/vote outputs, cost/session parsing.
- pkg/orchestrator: context resolution, condition evaluation, display parsing, report helpers, file categorization.
- pkg/tracking: status script parsing and error handling.
- pkg/tools/codex and pkg/tools/gemini: command assembly, defaults, validation, status summary glue.
- cmd/*: thin main wrappers (lower priority for unit tests).

## Proposed Unit Tests (concise)
- reports: ensure newest report selection, review detection window, skip logic, deletion retains newest.
- envelope: builder success/failure with metrics and output refs.
- executor: cost parsing for claude/codex/gemini, session ID extraction, merge/vote outputs, parallel partial aggregation, dispatcher unknown step.
- orchestrator (context/conditions): input/step resolution, stdout streaming extraction, condition comparisons/AND/OR.
- orchestrator (display): duration formatting, tool/state mappings, ANSI stripping, meaningful content extraction, log tail selection, task truncation.
- orchestrator (report helpers): categorizeFile, scanOutputFiles stats, grade JSON extraction, overview extraction, article discovery, helper utilities.
- tracking: FormatCredit, iTerm2 error classifier, status script JSON parsing (success + invalid JSON).
- tools/codex: BuildCommand for new/resume, defaults from settings, default model selection.
- tools/gemini: BuildCommand (resume vs default), default model behavior, flash override, model validation.

## Patch-ready diffs

### pkg/reports/manager_test.go
```diff
diff --git a/pkg/reports/manager_test.go b/pkg/reports/manager_test.go
new file mode 100644
index 0000000..1111111
--- /dev/null
+++ b/pkg/reports/manager_test.go
@@
+package reports
+
+import (
+    "os"
+    "path/filepath"
+    "strings"
+    "testing"
+    "time"
+)
+
+func writeFile(t *testing.T, path, content string) {
+    t.Helper()
+    if err := os.WriteFile(path, []byte(content), 0644); err != nil {
+        t.Fatalf("write file: %v", err)
+    }
+}
+
+func TestFindNewestReport(t *testing.T) {
+    dir := t.TempDir()
+    oldPath := filepath.Join(dir, "old.md")
+    newPath := filepath.Join(dir, "new.md")
+
+    writeFile(t, oldPath, "old")
+    writeFile(t, newPath, "new")
+
+    oldTime := time.Now().Add(-2 * time.Hour)
+    newTime := time.Now().Add(-1 * time.Hour)
+    if err := os.Chtimes(oldPath, oldTime, oldTime); err != nil {
+        t.Fatalf("chtimes old: %v", err)
+    }
+    if err := os.Chtimes(newPath, newTime, newTime); err != nil {
+        t.Fatalf("chtimes new: %v", err)
+    }
+
+    if got := FindNewestReport([]string{oldPath, newPath}); got != newPath {
+        t.Fatalf("FindNewestReport returned %q, want %q", got, newPath)
+    }
+}
+
+func TestIsReportReviewed(t *testing.T) {
+    dir := t.TempDir()
+
+    reviewed := filepath.Join(dir, "reviewed.md")
+    writeFile(t, reviewed, "Date Modified: today\n")
+    if !IsReportReviewed(reviewed) {
+        t.Fatal("IsReportReviewed returned false for reviewed report")
+    }
+
+    unreviewed := filepath.Join(dir, "unreviewed.md")
+    lines := strings.Repeat("line\n", reviewScanLines)
+    writeFile(t, unreviewed, lines+"Date Modified: late\n")
+    if IsReportReviewed(unreviewed) {
+        t.Fatal("IsReportReviewed returned true for marker after scan window")
+    }
+
+    if IsReportReviewed(filepath.Join(dir, "missing.md")) {
+        t.Fatal("IsReportReviewed returned true for missing file")
+    }
+}
+
+func TestShouldSkipTask(t *testing.T) {
+    dir := t.TempDir()
+
+    if ShouldSkipTask(dir, "c", "codex-", false) {
+        t.Fatal("ShouldSkipTask returned true when review not required")
+    }
+    if ShouldSkipTask(dir, "c", "", true) {
+        t.Fatal("ShouldSkipTask returned true with empty pattern")
+    }
+
+    missingDir := filepath.Join(dir, "missing")
+    if ShouldSkipTask(missingDir, "c", "codex-", true) {
+        t.Fatal("ShouldSkipTask returned true for missing directory")
+    }
+
+    unreviewed := filepath.Join(dir, "codex-1.md")
+    writeFile(t, unreviewed, "Title\n")
+    oldTime := time.Now().Add(-2 * time.Hour)
+    if err := os.Chtimes(unreviewed, oldTime, oldTime); err != nil {
+        t.Fatalf("chtimes unreviewed: %v", err)
+    }
+    if !ShouldSkipTask(dir, "c", "codex-", true) {
+        t.Fatal("ShouldSkipTask returned false for unreviewed report")
+    }
+
+    reviewed := filepath.Join(dir, "codex-2.md")
+    writeFile(t, reviewed, "Date Modified: today\n")
+    newTime := time.Now().Add(-1 * time.Hour)
+    if err := os.Chtimes(reviewed, newTime, newTime); err != nil {
+        t.Fatalf("chtimes reviewed: %v", err)
+    }
+    if ShouldSkipTask(dir, "c", "codex-", true) {
+        t.Fatal("ShouldSkipTask returned true for reviewed report")
+    }
+}
+
+func TestDeleteOldReports(t *testing.T) {
+    dir := t.TempDir()
+    reportPatterns := map[string]string{"c": "codex-"}
+    shortcuts := []string{"c"}
+
+    oldPath := filepath.Join(dir, "codex-1.md")
+    newPath := filepath.Join(dir, "codex-2.md")
+    writeFile(t, oldPath, "old")
+    writeFile(t, newPath, "new")
+
+    oldTime := time.Now().Add(-2 * time.Hour)
+    newTime := time.Now().Add(-1 * time.Hour)
+    if err := os.Chtimes(oldPath, oldTime, oldTime); err != nil {
+        t.Fatalf("chtimes old: %v", err)
+    }
+    if err := os.Chtimes(newPath, newTime, newTime); err != nil {
+        t.Fatalf("chtimes new: %v", err)
+    }
+
+    DeleteOldReports(dir, shortcuts, reportPatterns)
+
+    if _, err := os.Stat(oldPath); !os.IsNotExist(err) {
+        t.Fatalf("expected old report to be deleted, stat err: %v", err)
+    }
+    if _, err := os.Stat(newPath); err != nil {
+        t.Fatalf("expected newest report to remain, stat err: %v", err)
+    }
+}
```

### pkg/envelope/envelope_test.go
```diff
diff --git a/pkg/envelope/envelope_test.go b/pkg/envelope/envelope_test.go
new file mode 100644
index 0000000..1111111
--- /dev/null
+++ b/pkg/envelope/envelope_test.go
@@
+package envelope
+
+import "testing"
+
+func TestBuilderSuccess(t *testing.T) {
+    env := New().
+        WithTool("codex").
+        WithDuration(123).
+        Success().
+        WithResult("foo", "bar").
+        WithOutputRef("out.json").
+        Build()
+
+    if env.Status != StatusSuccess {
+        t.Fatalf("Status = %q, want %q", env.Status, StatusSuccess)
+    }
+    if env.Metrics == nil {
+        t.Fatal("Metrics is nil")
+    }
+    if env.Metrics.Tool != "codex" {
+        t.Fatalf("Metrics.Tool = %q, want %q", env.Metrics.Tool, "codex")
+    }
+    if env.Metrics.DurationMs != 123 {
+        t.Fatalf("Metrics.DurationMs = %d, want %d", env.Metrics.DurationMs, 123)
+    }
+    if env.Result["foo"] != "bar" {
+        t.Fatalf("Result[\"foo\"] = %v, want %q", env.Result["foo"], "bar")
+    }
+    if env.OutputRef != "out.json" {
+        t.Fatalf("OutputRef = %q, want %q", env.OutputRef, "out.json")
+    }
+}
+
+func TestBuilderFailure(t *testing.T) {
+    env := New().Failure("E_CODE", "boom").Build()
+
+    if env.Status != StatusFailure {
+        t.Fatalf("Status = %q, want %q", env.Status, StatusFailure)
+    }
+    if env.Error == nil {
+        t.Fatal("Error is nil")
+    }
+    if env.Error.Code != "E_CODE" {
+        t.Fatalf("Error.Code = %q, want %q", env.Error.Code, "E_CODE")
+    }
+    if env.Error.Message != "boom" {
+        t.Fatalf("Error.Message = %q, want %q", env.Error.Message, "boom")
+    }
+}
```

### pkg/executor/executor_test.go
```diff
diff --git a/pkg/executor/executor_test.go b/pkg/executor/executor_test.go
new file mode 100644
index 0000000..1111111
--- /dev/null
+++ b/pkg/executor/executor_test.go
@@
+package executor
+
+import (
+    "encoding/json"
+    "math"
+    "os"
+    "path/filepath"
+    "strings"
+    "testing"
+
+    "rcodegen/pkg/bundle"
+    "rcodegen/pkg/envelope"
+    "rcodegen/pkg/orchestrator"
+    "rcodegen/pkg/runner"
+    "rcodegen/pkg/workspace"
+)
+
+func TestExtractCostInfoClaude(t *testing.T) {
+    stdout := strings.Join([]string{
+        `{"type":"system","session_id":"abc"}`,
+        `{"type":"result","total_cost_usd":0.42,"usage":{"input_tokens":100,"output_tokens":50,"cache_read_input_tokens":5,"cache_creation_input_tokens":2}}`,
+    }, "\n")
+
+    usage := extractCostInfo("claude", stdout, "")
+    if usage.CostUSD != 0.42 {
+        t.Fatalf("CostUSD = %f, want %f", usage.CostUSD, 0.42)
+    }
+    if usage.InputTokens != 100 {
+        t.Fatalf("InputTokens = %d, want %d", usage.InputTokens, 100)
+    }
+    if usage.OutputTokens != 50 {
+        t.Fatalf("OutputTokens = %d, want %d", usage.OutputTokens, 50)
+    }
+    if usage.CacheReadTokens != 5 {
+        t.Fatalf("CacheReadTokens = %d, want %d", usage.CacheReadTokens, 5)
+    }
+    if usage.CacheWriteTokens != 2 {
+        t.Fatalf("CacheWriteTokens = %d, want %d", usage.CacheWriteTokens, 2)
+    }
+}
+
+func TestExtractCostInfoCodex(t *testing.T) {
+    stderr := "tokens used\n1,000\n"
+    usage := extractCostInfo("codex", "", stderr)
+
+    if usage.InputTokens != 700 {
+        t.Fatalf("InputTokens = %d, want %d", usage.InputTokens, 700)
+    }
+    if usage.OutputTokens != 300 {
+        t.Fatalf("OutputTokens = %d, want %d", usage.OutputTokens, 300)
+    }
+    if diff := math.Abs(usage.CostUSD - 0.016); diff > 0.000001 {
+        t.Fatalf("CostUSD = %f, want %f (diff %f)", usage.CostUSD, 0.016, diff)
+    }
+}
+
+func TestExtractCostInfoGemini(t *testing.T) {
+    stdout := `{"type":"result","stats":{"input_tokens":200,"output_tokens":50,"cached":10}}`
+    usage := extractCostInfo("gemini", stdout, "")
+
+    if usage.InputTokens != 200 {
+        t.Fatalf("InputTokens = %d, want %d", usage.InputTokens, 200)
+    }
+    if usage.OutputTokens != 50 {
+        t.Fatalf("OutputTokens = %d, want %d", usage.OutputTokens, 50)
+    }
+    if usage.CacheReadTokens != 10 {
+        t.Fatalf("CacheReadTokens = %d, want %d", usage.CacheReadTokens, 10)
+    }
+    if diff := math.Abs(usage.CostUSD - 0.000175); diff > 0.0000001 {
+        t.Fatalf("CostUSD = %f, want %f (diff %f)", usage.CostUSD, 0.000175, diff)
+    }
+}
+
+func TestExtractSessionID(t *testing.T) {
+    if got := extractSessionID("claude", `{"type":"system","session_id":"abc"}`, ""); got != "abc" {
+        t.Fatalf("claude session = %q, want %q", got, "abc")
+    }
+    if got := extractSessionID("gemini", `{"type":"init","session_id":"xyz"}`, ""); got != "xyz" {
+        t.Fatalf("gemini session = %q, want %q", got, "xyz")
+    }
+    if got := extractSessionID("codex", "", "session id: 1234-abcd"); got != "1234-abcd" {
+        t.Fatalf("codex session = %q, want %q", got, "1234-abcd")
+    }
+    if got := extractSessionID("codex", "", "no session"); got != "" {
+        t.Fatalf("expected empty session, got %q", got)
+    }
+}
+
+func TestExtractStepName(t *testing.T) {
+    if got := extractStepName("${steps.build.output_ref}"); got != "build" {
+        t.Fatalf("extractStepName = %q, want %q", got, "build")
+    }
+    if got := extractStepName("plain"); got != "plain" {
+        t.Fatalf("extractStepName = %q, want %q", got, "plain")
+    }
+}
+
+func TestVoteExecutorMajority(t *testing.T) {
+    ctx := orchestrator.NewContext(nil)
+    ctx.SetResult("a", &envelope.Envelope{Status: envelope.StatusSuccess})
+    ctx.SetResult("b", &envelope.Envelope{Status: envelope.StatusFailure})
+    ctx.SetResult("c", &envelope.Envelope{Status: envelope.StatusSuccess})
+
+    step := &bundle.Step{
+        Name: "vote",
+        Vote: &bundle.VoteDef{
+            Inputs:   []string{"${steps.a.output_ref}", "${steps.b.output_ref}", "${steps.c.output_ref}"},
+            Strategy: "majority",
+        },
+    }
+
+    ws, err := workspace.New(t.TempDir())
+    if err != nil {
+        t.Fatalf("workspace.New: %v", err)
+    }
+
+    env, err := (&VoteExecutor{}).Execute(step, ctx, ws)
+    if err != nil {
+        t.Fatalf("VoteExecutor.Execute: %v", err)
+    }
+
+    if env.Result["decision"] != "approved" {
+        t.Fatalf("decision = %v, want approved", env.Result["decision"])
+    }
+    votes, ok := env.Result["votes"].(map[string]int)
+    if !ok {
+        t.Fatalf("votes type = %T, want map[string]int", env.Result["votes"])
+    }
+    if votes["success"] != 2 || votes["failure"] != 1 {
+        t.Fatalf("votes = %+v, want success=2 failure=1", votes)
+    }
+}
+
+func TestVoteExecutorUnanimous(t *testing.T) {
+    ctx := orchestrator.NewContext(nil)
+    ctx.SetResult("a", &envelope.Envelope{Status: envelope.StatusSuccess})
+    ctx.SetResult("b", &envelope.Envelope{Status: envelope.StatusFailure})
+
+    step := &bundle.Step{
+        Name: "vote",
+        Vote: &bundle.VoteDef{
+            Inputs:   []string{"${steps.a.output_ref}", "${steps.b.output_ref}"},
+            Strategy: "unanimous",
+        },
+    }
+
+    ws, err := workspace.New(t.TempDir())
+    if err != nil {
+        t.Fatalf("workspace.New: %v", err)
+    }
+
+    env, err := (&VoteExecutor{}).Execute(step, ctx, ws)
+    if err != nil {
+        t.Fatalf("VoteExecutor.Execute: %v", err)
+    }
+
+    if env.Result["decision"] != "rejected" {
+        t.Fatalf("decision = %v, want rejected", env.Result["decision"])
+    }
+}
+
+func TestMergeExecutorConcat(t *testing.T) {
+    dir := t.TempDir()
+    input1 := filepath.Join(dir, "a.txt")
+    input2 := filepath.Join(dir, "b.txt")
+    if err := os.WriteFile(input1, []byte("alpha"), 0644); err != nil {
+        t.Fatalf("write input1: %v", err)
+    }
+    if err := os.WriteFile(input2, []byte("beta"), 0644); err != nil {
+        t.Fatalf("write input2: %v", err)
+    }
+
+    step := &bundle.Step{
+        Name: "merge",
+        Merge: &bundle.MergeDef{
+            Inputs:   []string{input1, input2},
+            Strategy: "concat",
+        },
+    }
+
+    ws, err := workspace.New(t.TempDir())
+    if err != nil {
+        t.Fatalf("workspace.New: %v", err)
+    }
+
+    env, err := (&MergeExecutor{}).Execute(step, orchestrator.NewContext(nil), ws)
+    if err != nil {
+        t.Fatalf("MergeExecutor.Execute: %v", err)
+    }
+    if env.Status != envelope.StatusSuccess {
+        t.Fatalf("Status = %q, want %q", env.Status, envelope.StatusSuccess)
+    }
+
+    data, err := os.ReadFile(env.OutputRef)
+    if err != nil {
+        t.Fatalf("read output: %v", err)
+    }
+    var out struct {
+        Merged     string `json:"merged"`
+        InputCount int    `json:"input_count"`
+    }
+    if err := json.Unmarshal(data, &out); err != nil {
+        t.Fatalf("unmarshal output: %v", err)
+    }
+
+    if out.InputCount != 2 {
+        t.Fatalf("InputCount = %d, want %d", out.InputCount, 2)
+    }
+    if out.Merged != "alpha\n\n---\n\nbeta" {
+        t.Fatalf("Merged = %q, want %q", out.Merged, "alpha\n\n---\n\nbeta")
+    }
+}
+
+func TestParallelExecutorPartial(t *testing.T) {
+    ws, err := workspace.New(t.TempDir())
+    if err != nil {
+        t.Fatalf("workspace.New: %v", err)
+    }
+    ctx := orchestrator.NewContext(nil)
+
+    inputDir := t.TempDir()
+    inputPath := filepath.Join(inputDir, "input.txt")
+    if err := os.WriteFile(inputPath, []byte("alpha"), 0644); err != nil {
+        t.Fatalf("write input: %v", err)
+    }
+
+    step := &bundle.Step{
+        Name: "parent",
+        Parallel: []bundle.Step{
+            {
+                Name: "merge-ok",
+                Merge: &bundle.MergeDef{
+                    Inputs:   []string{inputPath},
+                    Strategy: "concat",
+                },
+            },
+            {
+                Name: "missing-tool",
+                Tool: "unknown",
+            },
+        },
+    }
+
+    d := NewDispatcher(map[string]runner.Tool{})
+    env, err := d.Execute(step, ctx, ws)
+    if err != nil {
+        t.Fatalf("Dispatcher.Execute: %v", err)
+    }
+
+    if env.Status != envelope.StatusPartial {
+        t.Fatalf("Status = %q, want %q", env.Status, envelope.StatusPartial)
+    }
+    steps, ok := env.Result["steps"].(int)
+    if !ok || steps != 2 {
+        t.Fatalf("steps = %v, want 2", env.Result["steps"])
+    }
+}
+
+func TestDispatcherUnknownStep(t *testing.T) {
+    d := NewDispatcher(map[string]runner.Tool{})
+    ws, err := workspace.New(t.TempDir())
+    if err != nil {
+        t.Fatalf("workspace.New: %v", err)
+    }
+
+    env, err := d.Execute(&bundle.Step{Name: "noop"}, orchestrator.NewContext(nil), ws)
+    if err != nil {
+        t.Fatalf("Dispatcher.Execute: %v", err)
+    }
+    if env.Status != envelope.StatusFailure {
+        t.Fatalf("Status = %q, want %q", env.Status, envelope.StatusFailure)
+    }
+}
```

### pkg/orchestrator/context_test.go
```diff
diff --git a/pkg/orchestrator/context_test.go b/pkg/orchestrator/context_test.go
new file mode 100644
index 0000000..1111111
--- /dev/null
+++ b/pkg/orchestrator/context_test.go
@@
+package orchestrator
+
+import (
+    "encoding/json"
+    "os"
+    "path/filepath"
+    "strings"
+    "testing"
+
+    "rcodegen/pkg/envelope"
+)
+
+func TestContextResolveInputsAndResults(t *testing.T) {
+    dir := t.TempDir()
+    outputPath := filepath.Join(dir, "step.json")
+
+    payload := map[string]interface{}{
+        "stdout": "{\"type\":\"result\",\"result\":\"hello\"}\n",
+        "stderr": "oops",
+    }
+    data, err := json.Marshal(payload)
+    if err != nil {
+        t.Fatalf("marshal payload: %v", err)
+    }
+    if err := os.WriteFile(outputPath, data, 0644); err != nil {
+        t.Fatalf("write output: %v", err)
+    }
+
+    ctx := NewContext(map[string]string{"name": "World"})
+    ctx.SetResult("step1", &envelope.Envelope{
+        Status:    envelope.StatusSuccess,
+        OutputRef: outputPath,
+        Result:    map[string]interface{}{"foo": "bar"},
+    })
+
+    if got := ctx.Resolve("Hello ${inputs.name}"); got != "Hello World" {
+        t.Fatalf("Resolve inputs = %q, want %q", got, "Hello World")
+    }
+    if got := ctx.Resolve("${steps.step1.output_ref}"); got != outputPath {
+        t.Fatalf("Resolve output_ref = %q, want %q", got, outputPath)
+    }
+    if got := ctx.Resolve("${steps.step1.status}"); got != "success" {
+        t.Fatalf("Resolve status = %q, want %q", got, "success")
+    }
+    if got := ctx.Resolve("${steps.step1.result.foo}"); got != "bar" {
+        t.Fatalf("Resolve result.foo = %q, want %q", got, "bar")
+    }
+    if got := ctx.Resolve("${steps.step1.result}"); !strings.Contains(got, "\"foo\":\"bar\"") {
+        t.Fatalf("Resolve result = %q, want JSON containing foo", got)
+    }
+    if got := ctx.Resolve("${steps.step1.stdout}"); got != "hello" {
+        t.Fatalf("Resolve stdout = %q, want %q", got, "hello")
+    }
+    if got := ctx.Resolve("${steps.step1.stderr}"); got != "oops" {
+        t.Fatalf("Resolve stderr = %q, want %q", got, "oops")
+    }
+    if got := ctx.Resolve("${inputs.missing}"); got != "${inputs.missing}" {
+        t.Fatalf("Resolve missing = %q, want unresolved", got)
+    }
+}
+
+func TestExtractStreamingResult(t *testing.T) {
+    content := "{\"type\":\"assistant\"}\n{\"type\":\"result\",\"result\":\"final\"}"
+    if got := extractStreamingResult(content); got != "final" {
+        t.Fatalf("extractStreamingResult = %q, want %q", got, "final")
+    }
+
+    plain := "plain output"
+    if got := extractStreamingResult(plain); got != plain {
+        t.Fatalf("extractStreamingResult = %q, want %q", got, plain)
+    }
+}
+
+func TestEvaluateCondition(t *testing.T) {
+    ctx := NewContext(map[string]string{"count": "5", "name": "alpha"})
+    tests := []struct {
+        condition string
+        want      bool
+    }{
+        {"", true},
+        {"${inputs.count} >= 5", true},
+        {"${inputs.count} < 5", false},
+        {"'alpha' contains 'ph'", true},
+        {"\"alpha\" == \"alpha\"", true},
+        {"true AND false", false},
+        {"true OR false", true},
+        {"x > 1", false},
+    }
+
+    for _, tt := range tests {
+        if got := EvaluateCondition(tt.condition, ctx); got != tt.want {
+            t.Fatalf("EvaluateCondition(%q) = %v, want %v", tt.condition, got, tt.want)
+        }
+    }
+}
```

### pkg/orchestrator/display_test.go
```diff
diff --git a/pkg/orchestrator/display_test.go b/pkg/orchestrator/display_test.go
new file mode 100644
index 0000000..1111111
--- /dev/null
+++ b/pkg/orchestrator/display_test.go
@@
+package orchestrator
+
+import (
+    "os"
+    "path/filepath"
+    "strings"
+    "testing"
+    "time"
+
+    "rcodegen/pkg/bundle"
+)
+
+func TestFormatDuration(t *testing.T) {
+    if got := formatDuration(30 * time.Second); got != "30s" {
+        t.Fatalf("formatDuration = %q, want %q", got, "30s")
+    }
+    if got := formatDuration(2 * time.Minute); got != "2m" {
+        t.Fatalf("formatDuration = %q, want %q", got, "2m")
+    }
+    if got := formatDuration(65 * time.Second); got != "1m 5s" {
+        t.Fatalf("formatDuration = %q, want %q", got, "1m 5s")
+    }
+}
+
+func TestToolAndStateHelpers(t *testing.T) {
+    if toolColor("claude") != colorMagenta {
+        t.Fatalf("toolColor claude mismatch")
+    }
+    if toolColor("unknown") != colorWhite {
+        t.Fatalf("toolColor unknown mismatch")
+    }
+    if stateIcon(StepSuccess) != iconSuccess {
+        t.Fatalf("stateIcon success mismatch")
+    }
+    if stateColor(StepFailure) != colorRed {
+        t.Fatalf("stateColor failure mismatch")
+    }
+}
+
+func TestNewProgressDisplayTruncation(t *testing.T) {
+    b := &bundle.Bundle{
+        Name: "demo",
+        Steps: []bundle.Step{
+            {Name: "one", Tool: "claude"},
+            {Name: "two", Parallel: []bundle.Step{{Name: "sub"}}},
+        },
+    }
+    inputs := map[string]string{"task": strings.Repeat("a", 61)}
+    p := NewProgressDisplay(b, "job", inputs)
+
+    if p.task != strings.Repeat("a", 57)+"..." {
+        t.Fatalf("task = %q, want truncation", p.task)
+    }
+    if p.steps[1].Tool != "parallel" {
+        t.Fatalf("parallel tool = %q, want %q", p.steps[1].Tool, "parallel")
+    }
+}
+
+func TestNewLiveDisplayTruncation(t *testing.T) {
+    b := &bundle.Bundle{
+        Name:  "demo",
+        Steps: []bundle.Step{{Name: "one", Tool: "claude"}},
+    }
+    inputs := map[string]string{"task": strings.Repeat("b", 56)}
+    d := NewLiveDisplay(b, "job", inputs)
+
+    if d.task != strings.Repeat("b", 52)+"..." {
+        t.Fatalf("task = %q, want truncation", d.task)
+    }
+}
+
+func TestStripAnsi(t *testing.T) {
+    input := "\033[31mred\033[0m"
+    if got := stripAnsi(input); got != "red" {
+        t.Fatalf("stripAnsi = %q, want %q", got, "red")
+    }
+}
+
+func TestExtractMeaningfulContent(t *testing.T) {
+    tests := []struct {
+        name         string
+        line         string
+        want         string
+        wantContains string
+    }{
+        {"system", `{"type":"system"}`, "", ""},
+        {"tool-read", `{"type":"tool_use","name":"Read"}`, "", "Reading files"},
+        {"assistant-text", `{"type":"assistant","message":{"content":[{"type":"text","text":"Hello\\nWorld"}]}}`, "Hello World", ""},
+        {"tool-result", `{"type":"tool_result"}`, "", "Processing result"},
+        {"plain", "Status: running", "Status: running", ""},
+    }
+
+    for _, tt := range tests {
+        got := extractMeaningfulContent(tt.line)
+        if tt.want != "" && got != tt.want {
+            t.Fatalf("%s: got %q, want %q", tt.name, got, tt.want)
+        }
+        if tt.want == "" && tt.wantContains == "" && got != "" {
+            t.Fatalf("%s: got %q, want empty", tt.name, got)
+        }
+        if tt.wantContains != "" && !strings.Contains(got, tt.wantContains) {
+            t.Fatalf("%s: got %q, want contains %q", tt.name, got, tt.wantContains)
+        }
+    }
+}
+
+func TestReadLastMeaningfulLine(t *testing.T) {
+    dir := t.TempDir()
+    logPath := filepath.Join(dir, "step.log")
+    content := "\n{\"type\":\"system\"}\nStatus: running\n"
+    if err := os.WriteFile(logPath, []byte(content), 0644); err != nil {
+        t.Fatalf("write log: %v", err)
+    }
+
+    d := &LiveDisplay{logDir: dir}
+    if got := d.readLastMeaningfulLine("step"); got != "Status: running" {
+        t.Fatalf("readLastMeaningfulLine = %q, want %q", got, "Status: running")
+    }
+}
```

### pkg/orchestrator/report_helpers_test.go
```diff
diff --git a/pkg/orchestrator/report_helpers_test.go b/pkg/orchestrator/report_helpers_test.go
new file mode 100644
index 0000000..1111111
--- /dev/null
+++ b/pkg/orchestrator/report_helpers_test.go
@@
+package orchestrator
+
+import (
+    "os"
+    "path/filepath"
+    "reflect"
+    "strings"
+    "testing"
+)
+
+func TestCategorizeFile(t *testing.T) {
+    tests := map[string]string{
+        "src/main.go":         "source",
+        "lib/utils.ts":        "source",
+        "samples/example.txt": "sample",
+        "testdata/input.md":   "sample",
+        "docs/readme.md":      "docs",
+        "final-report.md":     "report",
+        "config/settings.json": "config",
+        "output/file.pdf":     "output",
+        "misc/other.bin":      "other",
+    }
+
+    for path, want := range tests {
+        if got := categorizeFile(path); got != want {
+            t.Fatalf("categorizeFile(%q) = %q, want %q", path, got, want)
+        }
+    }
+}
+
+func TestScanOutputFiles(t *testing.T) {
+    dir := t.TempDir()
+    srcDir := filepath.Join(dir, "src")
+    docsDir := filepath.Join(dir, "docs")
+
+    if err := os.MkdirAll(srcDir, 0755); err != nil {
+        t.Fatalf("mkdir src: %v", err)
+    }
+    if err := os.MkdirAll(docsDir, 0755); err != nil {
+        t.Fatalf("mkdir docs: %v", err)
+    }
+
+    if err := os.WriteFile(filepath.Join(srcDir, "main.go"), []byte("line1\nline2"), 0644); err != nil {
+        t.Fatalf("write source: %v", err)
+    }
+    if err := os.WriteFile(filepath.Join(docsDir, "readme.md"), []byte("hello world"), 0644); err != nil {
+        t.Fatalf("write docs: %v", err)
+    }
+    if err := os.WriteFile(filepath.Join(dir, "final-report.md"), []byte("grade here"), 0644); err != nil {
+        t.Fatalf("write report: %v", err)
+    }
+
+    files, stats := scanOutputFiles(dir)
+    if stats.TotalSourceFiles != 1 {
+        t.Fatalf("TotalSourceFiles = %d, want %d", stats.TotalSourceFiles, 1)
+    }
+    if stats.TotalSourceLines != 2 {
+        t.Fatalf("TotalSourceLines = %d, want %d", stats.TotalSourceLines, 2)
+    }
+    if stats.TotalDocWords != 4 {
+        t.Fatalf("TotalDocWords = %d, want %d", stats.TotalDocWords, 4)
+    }
+
+    wantTypes := map[string]string{
+        filepath.Join("src", "main.go"):     "source",
+        filepath.Join("docs", "readme.md"):  "docs",
+        "final-report.md":                    "report",
+    }
+    for _, f := range files {
+        if want, ok := wantTypes[f.Path]; ok && f.Type != want {
+            t.Fatalf("file %q type = %q, want %q", f.Path, f.Type, want)
+        }
+    }
+}
+
+func TestExtractGradeFromReport(t *testing.T) {
+    dir := t.TempDir()
+    path := filepath.Join(dir, "final-report.md")
+    content := "text\n```json\n{\"grade\":{\"score\":88,\"letter\":\"B\"}}\n```\n"
+    if err := os.WriteFile(path, []byte(content), 0644); err != nil {
+        t.Fatalf("write report: %v", err)
+    }
+
+    grade := extractGradeFromReport(path)
+    if grade == nil || grade.Score != 88 {
+        t.Fatalf("grade = %+v, want score 88", grade)
+    }
+}
+
+func TestExtractOverviewFromSummary(t *testing.T) {
+    dir := t.TempDir()
+    path := filepath.Join(dir, "IMPLEMENTATION_SUMMARY.md")
+    content := "# Title\n\n## Overview\nThis is the overview.\n\n## Next\nLater"
+    if err := os.WriteFile(path, []byte(content), 0644); err != nil {
+        t.Fatalf("write summary: %v", err)
+    }
+
+    if got := extractOverviewFromSummary(path); got != "This is the overview." {
+        t.Fatalf("overview = %q, want %q", got, "This is the overview.")
+    }
+}
+
+func TestExtractOpeningSummary(t *testing.T) {
+    dir := t.TempDir()
+    path := filepath.Join(dir, "article.md")
+    content := "# Title\n\nJane, engineer, remote worker.\nMore text"
+    if err := os.WriteFile(path, []byte(content), 0644); err != nil {
+        t.Fatalf("write article: %v", err)
+    }
+
+    if got := extractOpeningSummary(path); got != "Jane, engineer" {
+        t.Fatalf("opening summary = %q, want %q", got, "Jane, engineer")
+    }
+}
+
+func TestExtractAngle(t *testing.T) {
+    dir := t.TempDir()
+    path := filepath.Join(dir, "angle.md")
+    content := "Systemic issues and economic constraints."
+    if err := os.WriteFile(path, []byte(content), 0644); err != nil {
+        t.Fatalf("write angle: %v", err)
+    }
+
+    if got := extractAngle(path); got != "Systemic critique, Economic analysis" {
+        t.Fatalf("angle = %q, want %q", got, "Systemic critique, Economic analysis")
+    }
+}
+
+func TestExtractDataPoint(t *testing.T) {
+    dir := t.TempDir()
+    path := filepath.Join(dir, "data.md")
+    content := "10% growth and 20% decline"
+    if err := os.WriteFile(path, []byte(content), 0644); err != nil {
+        t.Fatalf("write data: %v", err)
+    }
+
+    if got := extractDataPoint(path); got != "Statistics (10%, 20%)" {
+        t.Fatalf("data point = %q, want %q", got, "Statistics (10%, 20%)")
+    }
+}
+
+func TestExtractTone(t *testing.T) {
+    dir := t.TempDir()
+    path := filepath.Join(dir, "tone.md")
+    content := "Builder focused and political debates."
+    if err := os.WriteFile(path, []byte(content), 0644); err != nil {
+        t.Fatalf("write tone: %v", err)
+    }
+
+    if got := extractTone(path); got != "Builder-focused, Political" {
+        t.Fatalf("tone = %q, want %q", got, "Builder-focused, Political")
+    }
+}
+
+func TestFindArticleFilesInDir(t *testing.T) {
+    dir := t.TempDir()
+    files := []string{"draft-codex.md", "Run Report.md", "outline.md", "Article One.md"}
+    for _, name := range files {
+        if err := os.WriteFile(filepath.Join(dir, name), []byte("content"), 0644); err != nil {
+            t.Fatalf("write %s: %v", name, err)
+        }
+    }
+
+    articles := findArticleFilesInDir(dir)
+    if len(articles) != 1 || !strings.Contains(articles[0], "Article One.md") {
+        t.Fatalf("articles = %v, want Article One.md", articles)
+    }
+}
+
+func TestFindArticleByTool(t *testing.T) {
+    articles := []string{"/tmp/Title - Codex.md", "/tmp/Other - Gemini.md"}
+    if got := findArticleByTool(articles, "gemini"); !strings.Contains(got, "Gemini") {
+        t.Fatalf("findArticleByTool = %q, want Gemini path", got)
+    }
+}
+
+func TestGetArticleNames(t *testing.T) {
+    paths := []string{"/tmp/Title - Codex.md", "/tmp/Other - Gemini.md", "/tmp/Plain.md"}
+    got := getArticleNames(paths)
+    want := []string{"Codex", "Gemini", "Plain"}
+    if !reflect.DeepEqual(got, want) {
+        t.Fatalf("getArticleNames = %v, want %v", got, want)
+    }
+}
+
+func TestHelperFunctions(t *testing.T) {
+    if got := truncate("abcdef", 5); got != "ab..." {
+        t.Fatalf("truncate = %q, want %q", got, "ab...")
+    }
+    if got := truncate("abc", 5); got != "abc" {
+        t.Fatalf("truncate = %q, want %q", got, "abc")
+    }
+    if got := capitalize("hello"); got != "Hello" {
+        t.Fatalf("capitalize = %q, want %q", got, "Hello")
+    }
+    if got := max(1, 3, 2); got != 3 {
+        t.Fatalf("max = %d, want %d", got, 3)
+    }
+    if got := findStepOutput("draft-codex"); got != "`docs/draft-codex.md`" {
+        t.Fatalf("findStepOutput = %q, want %q", got, "`docs/draft-codex.md`")
+    }
+}
```

### pkg/tracking/tracking_test.go
```diff
diff --git a/pkg/tracking/tracking_test.go b/pkg/tracking/tracking_test.go
new file mode 100644
index 0000000..1111111
--- /dev/null
+++ b/pkg/tracking/tracking_test.go
@@
+package tracking
+
+import (
+    "os/exec"
+    "runtime"
+    "testing"
+)
+
+func TestFormatCredit(t *testing.T) {
+    if got := FormatCredit(nil); got != "N/A" {
+        t.Fatalf("FormatCredit(nil) = %q, want %q", got, "N/A")
+    }
+    val := 42
+    if got := FormatCredit(&val); got != "42" {
+        t.Fatalf("FormatCredit(42) = %q, want %q", got, "42")
+    }
+}
+
+func TestClaudeStatusIsITerm2Error(t *testing.T) {
+    status := &ClaudeStatus{Error: "not_iterm2"}
+    if !status.IsITerm2Error() {
+        t.Fatal("IsITerm2Error returned false for not_iterm2")
+    }
+    status.Error = "no_iterm2_package"
+    if !status.IsITerm2Error() {
+        t.Fatal("IsITerm2Error returned false for no_iterm2_package")
+    }
+    status.Error = "other"
+    if status.IsITerm2Error() {
+        t.Fatal("IsITerm2Error returned true for other error")
+    }
+}
+
+func TestRunStatusScript(t *testing.T) {
+    if runtime.GOOS == "windows" {
+        t.Skip("requires sh")
+    }
+
+    cmd := exec.Command("sh", "-c", "printf '{\"5h_left\":10,\"weekly_left\":20}'")
+    status := runStatusScript(cmd)
+    if status.Error != "" {
+        t.Fatalf("status error = %q", status.Error)
+    }
+    if status.FiveHourLeft == nil || *status.FiveHourLeft != 10 {
+        t.Fatalf("FiveHourLeft = %v, want 10", status.FiveHourLeft)
+    }
+
+    cmd = exec.Command("sh", "-c", "printf 'not-json'")
+    status = runStatusScript(cmd)
+    if status.Error == "" {
+        t.Fatal("expected error for invalid JSON")
+    }
+}
+
+func TestRunClaudeStatusScript(t *testing.T) {
+    if runtime.GOOS == "windows" {
+        t.Skip("requires sh")
+    }
+
+    cmd := exec.Command("sh", "-c", "printf '{\"session_left\":5}'")
+    status := runClaudeStatusScript(cmd)
+    if status.Error != "" {
+        t.Fatalf("status error = %q", status.Error)
+    }
+    if status.SessionLeft == nil || *status.SessionLeft != 5 {
+        t.Fatalf("SessionLeft = %v, want 5", status.SessionLeft)
+    }
+}
```

### pkg/tools/codex/codex_test.go
```diff
diff --git a/pkg/tools/codex/codex_test.go b/pkg/tools/codex/codex_test.go
new file mode 100644
index 0000000..1111111
--- /dev/null
+++ b/pkg/tools/codex/codex_test.go
@@
+package codex
+
+import (
+    "os"
+    "path/filepath"
+    "testing"
+
+    "rcodegen/pkg/runner"
+    "rcodegen/pkg/settings"
+)
+
+func containsArg(args []string, target string) bool {
+    for _, arg := range args {
+        if arg == target {
+            return true
+        }
+    }
+    return false
+}
+
+func argValue(args []string, flag string) string {
+    for i := 0; i < len(args)-1; i++ {
+        if args[i] == flag {
+            return args[i+1]
+        }
+    }
+    return ""
+}
+
+func TestBuildCommandNewSession(t *testing.T) {
+    tool := New()
+    cfg := &runner.Config{Model: "gpt-5.2-codex", Effort: "low", OutputJSON: true}
+
+    cmd := tool.BuildCommand(cfg, "/tmp/work", "do it")
+    if cmd.Path != "codex" {
+        t.Fatalf("Path = %q, want %q", cmd.Path, "codex")
+    }
+    if !containsArg(cmd.Args, "exec") {
+        t.Fatalf("args missing exec: %v", cmd.Args)
+    }
+    if !containsArg(cmd.Args, "--json") {
+        t.Fatalf("args missing --json: %v", cmd.Args)
+    }
+    if argValue(cmd.Args, "-C") != "/tmp/work" {
+        t.Fatalf("-C value = %q, want %q", argValue(cmd.Args, "-C"), "/tmp/work")
+    }
+    if argValue(cmd.Args, "--model") != "gpt-5.2-codex" {
+        t.Fatalf("--model value = %q, want %q", argValue(cmd.Args, "--model"), "gpt-5.2-codex")
+    }
+}
+
+func TestBuildCommandResumeSession(t *testing.T) {
+    tool := New()
+    cfg := &runner.Config{SessionID: "sess-1", Model: "gpt-5.2-codex", Effort: "high"}
+
+    dir := t.TempDir()
+    wrapper := filepath.Join(dir, "codex_pty_wrapper.py")
+    if err := os.WriteFile(wrapper, []byte(""), 0644); err != nil {
+        t.Fatalf("write wrapper: %v", err)
+    }
+
+    cwd, err := os.Getwd()
+    if err != nil {
+        t.Fatalf("getwd: %v", err)
+    }
+    if err := os.Chdir(dir); err != nil {
+        t.Fatalf("chdir: %v", err)
+    }
+    t.Cleanup(func() {
+        _ = os.Chdir(cwd)
+    })
+
+    cmd := tool.BuildCommand(cfg, "/tmp/work", "task")
+    if cmd.Path != "python3" {
+        t.Fatalf("Path = %q, want %q", cmd.Path, "python3")
+    }
+    if len(cmd.Args) < 2 || cmd.Args[1] != wrapper {
+        t.Fatalf("wrapper arg = %v, want %q", cmd.Args, wrapper)
+    }
+    if !containsArg(cmd.Args, "sess-1") {
+        t.Fatalf("args missing session: %v", cmd.Args)
+    }
+    if argValue(cmd.Args, "-C") != "/tmp/work" {
+        t.Fatalf("-C value = %q, want %q", argValue(cmd.Args, "-C"), "/tmp/work")
+    }
+}
+
+func TestApplyToolDefaults(t *testing.T) {
+    tool := New()
+    cfg := &runner.Config{}
+    tool.ApplyToolDefaults(cfg)
+
+    if cfg.Effort != "xhigh" {
+        t.Fatalf("Effort = %q, want %q", cfg.Effort, "xhigh")
+    }
+    if !cfg.TrackStatus {
+        t.Fatal("TrackStatus should be true")
+    }
+
+    tool.SetSettings(&settings.Settings{Defaults: settings.Defaults{Codex: settings.CodexDefaults{Model: "gpt-4.1-codex", Effort: "low"}}})
+    cfg = &runner.Config{}
+    tool.ApplyToolDefaults(cfg)
+    if cfg.Model != "gpt-4.1-codex" {
+        t.Fatalf("Model = %q, want %q", cfg.Model, "gpt-4.1-codex")
+    }
+    if cfg.Effort != "low" {
+        t.Fatalf("Effort = %q, want %q", cfg.Effort, "low")
+    }
+}
+
+func TestDefaultModelSetting(t *testing.T) {
+    tool := New()
+    if tool.DefaultModelSetting() != tool.DefaultModel() {
+        t.Fatalf("DefaultModelSetting = %q, want %q", tool.DefaultModelSetting(), tool.DefaultModel())
+    }
+
+    tool.SetSettings(&settings.Settings{Defaults: settings.Defaults{Codex: settings.CodexDefaults{Model: "gpt-4.1-codex"}}})
+    if tool.DefaultModelSetting() != "gpt-4.1-codex" {
+        t.Fatalf("DefaultModelSetting = %q, want %q", tool.DefaultModelSetting(), "gpt-4.1-codex")
+    }
+}
```

### pkg/tools/gemini/gemini_test.go
```diff
diff --git a/pkg/tools/gemini/gemini_test.go b/pkg/tools/gemini/gemini_test.go
new file mode 100644
index 0000000..1111111
--- /dev/null
+++ b/pkg/tools/gemini/gemini_test.go
@@
+package gemini
+
+import (
+    "testing"
+
+    "rcodegen/pkg/runner"
+)
+
+func containsArg(args []string, target string) bool {
+    for _, arg := range args {
+        if arg == target {
+            return true
+        }
+    }
+    return false
+}
+
+func argValue(args []string, flag string) string {
+    for i := 0; i < len(args)-1; i++ {
+        if args[i] == flag {
+            return args[i+1]
+        }
+    }
+    return ""
+}
+
+func TestBuildCommandDefaultModel(t *testing.T) {
+    tool := New()
+    cfg := &runner.Config{Model: tool.DefaultModel()}
+
+    cmd := tool.BuildCommand(cfg, "/tmp/work", "task")
+    if cmd.Path != "gemini" {
+        t.Fatalf("Path = %q, want %q", cmd.Path, "gemini")
+    }
+    if containsArg(cmd.Args, "-m") {
+        t.Fatalf("did not expect -m for default model: %v", cmd.Args)
+    }
+    if cmd.Dir != "/tmp/work" {
+        t.Fatalf("Dir = %q, want %q", cmd.Dir, "/tmp/work")
+    }
+}
+
+func TestBuildCommandResume(t *testing.T) {
+    tool := New()
+    cfg := &runner.Config{SessionID: "sess", Model: "gemini-2.5-pro"}
+
+    cmd := tool.BuildCommand(cfg, "", "task")
+    if !containsArg(cmd.Args, "--resume") {
+        t.Fatalf("args missing --resume: %v", cmd.Args)
+    }
+    if argValue(cmd.Args, "--resume") != "sess" {
+        t.Fatalf("--resume value = %q, want %q", argValue(cmd.Args, "--resume"), "sess")
+    }
+    if argValue(cmd.Args, "-m") != "gemini-2.5-pro" {
+        t.Fatalf("-m value = %q, want %q", argValue(cmd.Args, "-m"), "gemini-2.5-pro")
+    }
+}
+
+func TestApplyToolDefaults(t *testing.T) {
+    tool := New()
+    cfg := &runner.Config{}
+    tool.ApplyToolDefaults(cfg)
+    if cfg.Model != tool.DefaultModel() {
+        t.Fatalf("Model = %q, want %q", cfg.Model, tool.DefaultModel())
+    }
+}
+
+func TestPrepareForExecutionFlash(t *testing.T) {
+    tool := New()
+    cfg := &runner.Config{Model: tool.DefaultModel(), Flash: true}
+    tool.PrepareForExecution(cfg)
+    if cfg.Model != "gemini-3-flash-preview" {
+        t.Fatalf("Model = %q, want %q", cfg.Model, "gemini-3-flash-preview")
+    }
+}
+
+func TestValidateConfig(t *testing.T) {
+    tool := New()
+    cfg := &runner.Config{Model: "gemini-2.5-pro"}
+    if err := tool.ValidateConfig(cfg); err != nil {
+        t.Fatalf("ValidateConfig valid = %v", err)
+    }
+
+    cfg.Model = "invalid-model"
+    if err := tool.ValidateConfig(cfg); err == nil {
+        t.Fatal("expected error for invalid model")
+    }
+}
+
+func TestUsesStreamOutput(t *testing.T) {
+    tool := New()
+    if !tool.UsesStreamOutput() {
+        t.Fatal("UsesStreamOutput should be true")
+    }
+}
```
