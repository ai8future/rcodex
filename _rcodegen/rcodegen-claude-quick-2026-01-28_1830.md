Date Created: 2026-01-28 18:30:00
TOTAL_SCORE: 82/100

# rcodegen Quick Analysis Report

**Tool:** Claude Code (Opus 4.5)
**Codebase:** rcodegen
**Analysis Type:** Quick Combined Report

---

## Executive Summary

rcodegen is a well-engineered unified automation framework for AI-powered code analysis. It wraps Claude Code, OpenAI Codex, and Google Gemini CLIs with a consistent interface for unattended code analysis tasks. The codebase demonstrates solid Go practices, good test coverage of critical paths, and thoughtful security considerations. Main improvement areas are reducing function complexity in runner.go and expanding test coverage to tool implementations.

---

## (1) AUDIT - Security and Code Quality Issues

### A1. Potential Command Injection in Task Prompts (LOW RISK)

**Location:** `pkg/runner/runner.go:106-111`
**Severity:** Low
**Description:** Task prompts are passed directly to CLI tools without sanitization. While prompts come from trusted settings.json, if a malicious prompt were crafted, it could potentially include shell metacharacters.

**Risk Assessment:** Low - prompts come from user-controlled settings.json which requires file system access to modify.

```diff
--- a/pkg/runner/runner.go
+++ b/pkg/runner/runner.go
@@ -103,6 +103,13 @@ func (r *Runner) Run() *RunResult {
 	cfg.Task = strings.ReplaceAll(cfg.Task, "{report_dir}", reportDir)

 	// Substitute {timestamp} with current time in YYYY-MM-DD_HHMM format
+	// NOTE: Tasks come from settings.json which is user-controlled.
+	// If this were to accept external input, sanitization would be needed:
+	// - Escape shell metacharacters (;, |, &, $, `, etc.)
+	// - Validate against allowed character set
+	// Current design is safe because settings.json requires local file access.
 	timestamp := time.Now().Format("2006-01-02_1504")
 	cfg.Task = strings.ReplaceAll(cfg.Task, "{timestamp}", timestamp)
```

### A2. No Validation of Codex Model Names

**Location:** `pkg/tools/codex/codex.go:261-264`
**Severity:** Low
**Description:** Codex ValidateConfig() accepts any model name without validation, unlike Claude which validates against a known list.

```diff
--- a/pkg/tools/codex/codex.go
+++ b/pkg/tools/codex/codex.go
@@ -258,8 +258,16 @@ func (t *Tool) PrepareForExecution(cfg *runner.Config) {

 // ValidateConfig validates Codex-specific configuration
 func (t *Tool) ValidateConfig(cfg *runner.Config) error {
-	// Codex accepts any model name, so no validation needed
-	return nil
+	// Validate model against known models
+	if err := runner.ValidateModel(t, cfg.Model); err != nil {
+		return err
+	}
+	// Validate effort level
+	validEfforts := map[string]bool{"low": true, "medium": true, "high": true, "xhigh": true}
+	if !validEfforts[cfg.Effort] {
+		return fmt.Errorf("invalid effort '%s': must be one of low, medium, high, xhigh", cfg.Effort)
+	}
+	return nil
 }
```

### A3. os.Exit() Calls in Settings Package

**Location:** `pkg/settings/settings.go:196-197`, `pkg/settings/settings.go:611-612`
**Severity:** Low
**Description:** Direct os.Exit() calls make the code harder to test and don't allow cleanup handlers to run.

```diff
--- a/pkg/settings/settings.go
+++ b/pkg/settings/settings.go
@@ -193,8 +193,7 @@ func LoadWithFallback() (*Settings, bool) {
 	// Check for reserved task name overrides before merging
 	if err := ValidateNoReservedTaskOverrides(settings.Tasks); err != nil {
 		fmt.Fprintf(os.Stderr, "%sError:%s %v\n", yellow, reset, err)
-		os.Exit(1)
+		return nil, false
 	}
 	// Merge default tasks - custom user tasks with non-reserved names are allowed
```

### A4. File Permissions Not Set Restrictively on Reports

**Location:** `pkg/runner/runner.go:1031`
**Severity:** Info
**Description:** Run log files are created with 0644 permissions. While generally fine, sensitive information about task execution could be exposed.

```diff
--- a/pkg/runner/runner.go
+++ b/pkg/runner/runner.go
@@ -1028,7 +1028,7 @@ func (r *Runner) writeRunLog(cfg *Config, workDir string, startTime, endTime tim
 	content := strings.Join(lines, "\n") + "\n"

 	// Write file
-	if err := os.WriteFile(filepath, []byte(content), 0644); err != nil {
+	if err := os.WriteFile(filepath, []byte(content), 0600); err != nil {
 		fmt.Fprintf(os.Stderr, "%sWarning:%s Could not write runlog: %v\n", Yellow, Reset, err)
 		return
 	}
```

---

## (2) TESTS - Proposed Unit Tests

### T1. Missing Tests for Claude Tool Command Building

**Location:** `pkg/tools/claude/claude_test.go` (new file)
**Priority:** High
**Description:** The Claude tool has no unit tests. BuildCommand() constructs complex CLI arguments that should be verified.

```diff
--- /dev/null
+++ b/pkg/tools/claude/claude_test.go
@@ -0,0 +1,89 @@
+package claude
+
+import (
+	"strings"
+	"testing"
+
+	"rcodegen/pkg/runner"
+)
+
+func TestBuildCommand_NewSession(t *testing.T) {
+	tool := New()
+	cfg := &runner.Config{
+		Model:     "sonnet",
+		MaxBudget: "10.00",
+	}
+
+	cmd := tool.BuildCommand(cfg, "/test/dir", "test task")
+
+	// Verify command structure
+	if cmd.Path == "" {
+		t.Error("Expected command path to be set")
+	}
+
+	args := strings.Join(cmd.Args, " ")
+
+	// Check required flags are present
+	if !strings.Contains(args, "-p") {
+		t.Error("Expected -p flag for prompt")
+	}
+	if !strings.Contains(args, "--dangerously-skip-permissions") {
+		t.Error("Expected --dangerously-skip-permissions flag")
+	}
+	if !strings.Contains(args, "--model sonnet") {
+		t.Error("Expected --model sonnet flag")
+	}
+	if !strings.Contains(args, "--max-budget-usd 10.00") {
+		t.Error("Expected --max-budget-usd flag")
+	}
+
+	// Verify working directory
+	if cmd.Dir != "/test/dir" {
+		t.Errorf("Expected Dir=/test/dir, got %s", cmd.Dir)
+	}
+}
+
+func TestBuildCommand_ResumeSession(t *testing.T) {
+	tool := New()
+	cfg := &runner.Config{
+		Model:     "opus",
+		MaxBudget: "5.00",
+		SessionID: "abc123",
+	}
+
+	cmd := tool.BuildCommand(cfg, "", "continue task")
+	args := strings.Join(cmd.Args, " ")
+
+	if !strings.Contains(args, "--resume abc123") {
+		t.Error("Expected --resume flag with session ID")
+	}
+}
+
+func TestBuildCommand_JSONOutput(t *testing.T) {
+	tool := New()
+	cfg := &runner.Config{
+		Model:      "haiku",
+		MaxBudget:  "1.00",
+		OutputJSON: true,
+	}
+
+	cmd := tool.BuildCommand(cfg, "", "task")
+	args := strings.Join(cmd.Args, " ")
+
+	if !strings.Contains(args, "--output-format json") {
+		t.Error("Expected JSON output format")
+	}
+	if strings.Contains(args, "stream-json") {
+		t.Error("Should not have stream-json when OutputJSON is true")
+	}
+}
+
+func TestValidateConfig_InvalidBudget(t *testing.T) {
+	tool := New()
+	cfg := &runner.Config{Model: "sonnet", MaxBudget: "-5.00"}
+
+	if err := tool.ValidateConfig(cfg); err == nil {
+		t.Error("Expected error for negative budget")
+	}
+}
```

### T2. Missing Tests for Codex Tool Command Building

**Location:** `pkg/tools/codex/codex_test.go` (new file)
**Priority:** High

```diff
--- /dev/null
+++ b/pkg/tools/codex/codex_test.go
@@ -0,0 +1,67 @@
+package codex
+
+import (
+	"strings"
+	"testing"
+
+	"rcodegen/pkg/runner"
+)
+
+func TestBuildCommand_NewSession(t *testing.T) {
+	tool := New()
+	cfg := &runner.Config{
+		Model:  "gpt-5.2-codex",
+		Effort: "xhigh",
+	}
+
+	cmd := tool.BuildCommand(cfg, "/work", "analyze code")
+
+	// Should start with "codex" and "exec"
+	if len(cmd.Args) < 2 || cmd.Args[0] != "codex" {
+		t.Error("Expected command to be 'codex'")
+	}
+
+	args := strings.Join(cmd.Args, " ")
+
+	if !strings.Contains(args, "exec") {
+		t.Error("Expected 'exec' subcommand")
+	}
+	if !strings.Contains(args, "--dangerously-bypass-approvals-and-sandbox") {
+		t.Error("Expected bypass flag")
+	}
+	if !strings.Contains(args, "--model gpt-5.2-codex") {
+		t.Error("Expected model flag")
+	}
+	if !strings.Contains(args, `model_reasoning_effort="xhigh"`) {
+		t.Error("Expected effort config")
+	}
+	if !strings.Contains(args, "-C /work") {
+		t.Error("Expected working directory flag")
+	}
+}
+
+func TestBuildCommand_ResumeWithPTY(t *testing.T) {
+	tool := New()
+	cfg := &runner.Config{
+		Model:     "gpt-5.2-codex",
+		Effort:    "high",
+		SessionID: "session-xyz",
+	}
+
+	cmd := tool.BuildCommand(cfg, "", "continue")
+
+	// Should use python3 for PTY wrapper
+	if cmd.Args[0] != "python3" {
+		t.Errorf("Expected python3 for resume, got %s", cmd.Args[0])
+	}
+
+	args := strings.Join(cmd.Args, " ")
+	if !strings.Contains(args, "session-xyz") {
+		t.Error("Expected session ID in args")
+	}
+}
```

### T3. Missing Tests for Stream Parser Edge Cases

**Location:** `pkg/runner/stream_test.go` (new file)
**Priority:** Medium

```diff
--- /dev/null
+++ b/pkg/runner/stream_test.go
@@ -0,0 +1,95 @@
+package runner
+
+import (
+	"bytes"
+	"strings"
+	"testing"
+)
+
+func TestStreamParser_InvalidJSON(t *testing.T) {
+	var buf bytes.Buffer
+	p := NewStreamParser(&buf)
+
+	// Should not panic on invalid JSON
+	p.ProcessLine("not json at all")
+	p.ProcessLine("{incomplete")
+	p.ProcessLine("")
+
+	output := buf.String()
+	if !strings.Contains(output, "not json at all") {
+		t.Error("Invalid JSON should be passed through as-is")
+	}
+}
+
+func TestStreamParser_ToolUseIcons(t *testing.T) {
+	var buf bytes.Buffer
+	p := NewStreamParser(&buf)
+
+	// Test Read tool
+	p.ProcessLine(`{"type":"assistant","message":{"content":[{"type":"tool_use","name":"Read","input":{"file_path":"/test/file.go"}}]}}`)
+
+	output := buf.String()
+	if !strings.Contains(output, "Reading file") {
+		t.Errorf("Expected 'Reading file' in output, got: %s", output)
+	}
+}
+
+func TestStreamParser_CapturesUsage(t *testing.T) {
+	var buf bytes.Buffer
+	p := NewStreamParser(&buf)
+
+	// Send result event with usage
+	p.ProcessLine(`{"type":"result","usage":{"input_tokens":1000,"output_tokens":500},"total_cost_usd":0.05}`)
+
+	if p.Usage == nil {
+		t.Fatal("Expected Usage to be captured")
+	}
+	if p.Usage.InputTokens != 1000 {
+		t.Errorf("Expected InputTokens=1000, got %d", p.Usage.InputTokens)
+	}
+	if p.Usage.OutputTokens != 500 {
+		t.Errorf("Expected OutputTokens=500, got %d", p.Usage.OutputTokens)
+	}
+	if p.TotalCostUSD != 0.05 {
+		t.Errorf("Expected TotalCostUSD=0.05, got %f", p.TotalCostUSD)
+	}
+}
+
+func TestStreamParser_GeminiStats(t *testing.T) {
+	var buf bytes.Buffer
+	p := NewStreamParser(&buf)
+
+	// Gemini format has stats instead of usage
+	p.ProcessLine(`{"type":"result","stats":{"input_tokens":2000,"output_tokens":800,"cached":100}}`)
+
+	if p.Usage == nil {
+		t.Fatal("Expected Usage to be captured from Gemini stats")
+	}
+	if p.Usage.InputTokens != 2000 {
+		t.Errorf("Expected InputTokens=2000, got %d", p.Usage.InputTokens)
+	}
+	if p.Usage.CacheReadInputTokens != 100 {
+		t.Errorf("Expected CacheReadInputTokens=100, got %d", p.Usage.CacheReadInputTokens)
+	}
+}
+
+func TestShortenPath(t *testing.T) {
+	tests := []struct {
+		input    string
+		contains string
+	}{
+		{"/some/absolute/path", "/some/absolute/path"},
+		{"", ""},
+	}
+
+	for _, tt := range tests {
+		result := shortenPath(tt.input)
+		if tt.contains != "" && !strings.Contains(result, tt.contains) {
+			t.Errorf("shortenPath(%q) = %q, expected to contain %q", tt.input, result, tt.contains)
+		}
+	}
+}
```

### T4. Missing Tests for Multi-Codebase Run Logic

**Location:** `pkg/runner/runner_multicodebase_test.go` (new file)
**Priority:** Medium

```diff
--- /dev/null
+++ b/pkg/runner/runner_multicodebase_test.go
@@ -0,0 +1,52 @@
+package runner
+
+import (
+	"os"
+	"path/filepath"
+	"testing"
+)
+
+func TestDiscoverDirectories_FindsGitRepos(t *testing.T) {
+	// Create temp directory structure
+	tmpDir := t.TempDir()
+
+	// Create a git repo
+	repo1 := filepath.Join(tmpDir, "repo1")
+	os.MkdirAll(filepath.Join(repo1, ".git"), 0755)
+
+	// Create a non-git directory
+	os.MkdirAll(filepath.Join(tmpDir, "notarepo"), 0755)
+
+	// Create nested git repo
+	nested := filepath.Join(tmpDir, "nested", "repo2")
+	os.MkdirAll(filepath.Join(nested, ".git"), 0755)
+
+	dirs, err := discoverDirectories(tmpDir, 2)
+	if err != nil {
+		t.Fatalf("Unexpected error: %v", err)
+	}
+
+	// Should find repo1 and nested/repo2, but not notarepo
+	if len(dirs) != 2 {
+		t.Errorf("Expected 2 directories, got %d: %v", len(dirs), dirs)
+	}
+}
+
+func TestDiscoverDirectories_SkipsHiddenDirs(t *testing.T) {
+	tmpDir := t.TempDir()
+
+	// Create hidden directory with .git
+	hidden := filepath.Join(tmpDir, ".hidden")
+	os.MkdirAll(filepath.Join(hidden, ".git"), 0755)
+
+	// Create node_modules with .git
+	nm := filepath.Join(tmpDir, "node_modules")
+	os.MkdirAll(filepath.Join(nm, ".git"), 0755)
+
+	dirs, err := discoverDirectories(tmpDir, 1)
+	if err != nil {
+		t.Fatalf("Unexpected error: %v", err)
+	}
+
+	if len(dirs) != 0 {
+		t.Errorf("Expected 0 directories (hidden/node_modules skipped), got %d: %v", len(dirs), dirs)
+	}
+}
```

---

## (3) FIXES - Bugs, Issues, and Code Smells

### F1. Variable Shadowing in filepath Assignment

**Location:** `pkg/runner/runner.go:992`
**Severity:** Low (Code Smell)
**Description:** The variable `filepath` shadows the imported `path/filepath` package. This works but is confusing and could lead to bugs.

```diff
--- a/pkg/runner/runner.go
+++ b/pkg/runner/runner.go
@@ -989,8 +989,8 @@ func (r *Runner) writeRunLog(cfg *Config, workDir string, startTime, endTime tim
 	}
 	timestamp := startTime.Format("2006-01-02_1504")
 	filename := fmt.Sprintf("%s-%s-%s.runlog", codebaseName, taskName, timestamp)
-	filepath := filepath.Join(outputDir, filename)
+	logPath := filepath.Join(outputDir, filename)

 	// Build content
 	var lines []string
@@ -1028,7 +1028,7 @@ func (r *Runner) writeRunLog(cfg *Config, workDir string, startTime, endTime tim
 	content := strings.Join(lines, "\n") + "\n"

 	// Write file
-	if err := os.WriteFile(filepath, []byte(content), 0644); err != nil {
+	if err := os.WriteFile(logPath, []byte(content), 0600); err != nil {
 		fmt.Fprintf(os.Stderr, "%sWarning:%s Could not write runlog: %v\n", Yellow, Reset, err)
 		return
 	}
```

### F2. Unused Duration Variable

**Location:** `pkg/runner/runner.go:310`
**Severity:** Info
**Description:** The `duration` variable is computed but explicitly discarded with `_ = duration`. This is intentional per comment but adds dead code.

```diff
--- a/pkg/runner/runner.go
+++ b/pkg/runner/runner.go
@@ -282,11 +282,8 @@ func (r *Runner) runForWorkDir(cfg *Config, workDir string) int {
 		}
 	}

-	// Record end time
-	duration := time.Since(startTime)
-
 	// Delete old reports if requested
-	if cfg.DeleteOld && exitCode == 0 {
+	if cfg.DeleteOld && exitCode == 0 && cfg.TaskShortcut != "" {
 		var shortcuts []string
 		switch cfg.TaskShortcut {
 		case TaskSuite:
@@ -305,8 +302,6 @@ func (r *Runner) runForWorkDir(cfg *Config, workDir string) int {
 		// Summary will be printed at the end
 	}

-	_ = duration // Used in multi-codebase mode
 	return exitCode
 }
```

### F3. Inconsistent Model Defaults Between Files

**Location:** `pkg/settings/settings.go:159` vs `pkg/settings/settings.go:188`
**Severity:** Low
**Description:** GetDefaultSettings() uses "gemini-3" while LoadWithFallback() uses "gemini-3-pro-preview" for Gemini defaults.

```diff
--- a/pkg/settings/settings.go
+++ b/pkg/settings/settings.go
@@ -156,7 +156,7 @@ func GetDefaultSettings() *Settings {
 			Budget: "10.00",
 		},
 		Gemini: GeminiDefaults{
-			Model: "gemini-3",
+			Model: "gemini-3-pro-preview",
 		},
 	},
 	Tasks: make(map[string]TaskDef),
```

### F4. Budget Validation Allows Very Small Values

**Location:** `pkg/tools/claude/claude.go:314-317`
**Severity:** Low
**Description:** Budget validation allows any positive number including impractically small values like 0.001.

```diff
--- a/pkg/tools/claude/claude.go
+++ b/pkg/tools/claude/claude.go
@@ -312,6 +312,9 @@ func (t *Tool) ValidateConfig(cfg *runner.Config) error {
 	if budget <= 0 {
 		return fmt.Errorf("invalid budget '%s': must be greater than 0", cfg.MaxBudget)
 	}
+	if budget < 0.10 {
+		return fmt.Errorf("invalid budget '%s': minimum is 0.10", cfg.MaxBudget)
+	}
 	if budget > 1000 {
 		return fmt.Errorf("invalid budget '%s': maximum is 1000.00", cfg.MaxBudget)
 	}
```

### F5. Error Message Typo in Grade Persistence

**Location:** `pkg/runner/runner.go:383`
**Severity:** Info
**Description:** Error message formatting could be improved for consistency.

```diff
--- a/pkg/runner/runner.go
+++ b/pkg/runner/runner.go
@@ -380,7 +380,7 @@ func (r *Runner) persistGrade(cfg *Config, workDir, taskShortcut string) {

 	// Append to .grades.json
 	if err := AppendGrade(reportDir, filename, toolName, taskShortcut, grade, date); err != nil {
-		fmt.Fprintf(os.Stderr, "%sWarning:%s Could not save grade: %v\n", Yellow, Reset, err)
+		fmt.Fprintf(os.Stderr, "%sWarning:%s Failed to save grade to .grades.json: %v\n", Yellow, Reset, err)
 	}
 }
```

---

## (4) REFACTOR - Opportunities to Improve Code Quality

### R1. Split runner.go into Smaller Files

**File:** `pkg/runner/runner.go` (1,107 lines)
**Recommendation:** Extract logical sections into separate files:

- `runner_parse.go` - parseArgs() and related helpers (~250 lines)
- `runner_exec.go` - execution methods (runSingleTask, runMultipleReports) (~150 lines)
- `runner_summary.go` - printDetailedSummary, writeRunLog (~150 lines)
- `runner_discovery.go` - discoverDirectories, setWorkingDirectories (~100 lines)

Keep `runner.go` focused on Run() and the core Runner struct (~400 lines).

### R2. Consolidate Status Tracking Pattern

**Files:** `pkg/tools/claude/claude.go`, `pkg/tools/codex/codex.go`
**Recommendation:** Both tools have similar CaptureStatusBefore/After patterns. Consider:

```go
// pkg/runner/status.go
type StatusTracker interface {
    CaptureStatus() (interface{}, error)
    FormatDelta(before, after interface{}) string
}
```

This would reduce duplication and make the pattern consistent.

### R3. Centralize Color Constants

**Files:** `pkg/runner/output.go`, `pkg/settings/settings.go`
**Recommendation:** Both packages define ANSI color constants. Move all colors to `pkg/colors/colors.go` and import from there.

Current duplication:
- `pkg/runner/output.go`: Bold, Dim, Green, Yellow, Cyan, Magenta, Red, White, Reset
- `pkg/settings/settings.go`: bold, dim, green, cyan, yellow, magenta, reset

### R4. Extract Tool Interface into Separate Package

**File:** `pkg/runner/tool.go` (implied)
**Recommendation:** The Tool interface and related types could be in their own package (`pkg/tool/interface.go`) to avoid circular dependency concerns and make the contract clearer.

### R5. Add Structured Logging

**Recommendation:** Currently debugging is done via commented-out fmt.Fprintf calls. Consider adding a structured logger with levels:

```go
// pkg/logging/logger.go
type Logger interface {
    Debug(msg string, fields ...any)
    Info(msg string, fields ...any)
    Warn(msg string, fields ...any)
    Error(msg string, fields ...any)
}
```

Enable via `--verbose` or `RCODEGEN_DEBUG=1` environment variable.

### R6. Reduce Nesting in parseArgs()

**File:** `pkg/runner/runner.go:708-907`
**Recommendation:** The parseArgs() function is 200 lines with multiple nested conditionals. Consider:

1. Extract flag definition to a separate function
2. Use early returns for special cases (help, tasks, migrate)
3. Group related validation into helper functions

### R7. Add Context Support for Cancellation

**Recommendation:** The runner currently doesn't support graceful cancellation. Adding `context.Context` would allow:
- Timeout-based cancellation
- User interrupt handling (Ctrl+C)
- Coordinated shutdown of multi-codebase runs

```go
func (r *Runner) RunWithContext(ctx context.Context) *RunResult {
    // Check ctx.Done() at key points
}
```

### R8. Consider Using Table-Driven Flag Definitions

**Recommendation:** Instead of individual flag.StringVar/BoolVar calls, use a table-driven approach:

```go
type FlagSpec struct {
    Short, Long string
    Target      *string  // or interface{} for type flexibility
    Default     string
    Help        string
}

func defineFlags(specs []FlagSpec) {
    for _, s := range specs {
        flag.StringVar(s.Target, s.Short, s.Default, s.Help)
        flag.StringVar(s.Target, s.Long, s.Default, s.Help)
    }
}
```

---

## Grade Breakdown

| Category | Weight | Score | Notes |
|----------|--------|-------|-------|
| Architecture & Design | 25% | 85/100 | Clean Tool interface, good separation of concerns |
| Security Practices | 20% | 85/100 | Permission warnings, path validation, minimal deps |
| Error Handling | 15% | 80/100 | Generally good, some os.Exit() calls |
| Testing | 15% | 75/100 | Good core coverage, missing tool tests |
| Idioms & Style | 15% | 82/100 | Standard Go, some large functions |
| Documentation | 10% | 90/100 | Excellent README, CHANGELOG, comments |

**Weighted Score: 82/100**

---

## Summary

rcodegen is a mature, well-designed automation framework with solid engineering fundamentals. The main areas for improvement are:

1. **Testing** - Add unit tests for tool implementations (claude, codex, gemini packages)
2. **Code Organization** - Split runner.go into smaller, focused files
3. **Validation** - Add model and effort validation to Codex tool
4. **Consistency** - Unify color constants and status tracking patterns

The codebase is production-ready and actively maintained. The architecture is extensible, making it straightforward to add new AI tool integrations.
