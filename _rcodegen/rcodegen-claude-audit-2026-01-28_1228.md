Date Created: 2026-01-28 12:28:00
TOTAL_SCORE: 81/100

# rcodegen Comprehensive Code Audit Report

## Executive Summary

rcodegen is a well-architected Go-based automation framework for running AI coding assistants (Claude, Codex, Gemini) in unattended mode. The codebase demonstrates solid design patterns, good security awareness, and reasonable code quality. However, there are areas for improvement in test coverage, error handling, and a few security considerations.

---

## Grading Breakdown

| Category | Score | Weight | Weighted |
|----------|-------|--------|----------|
| Architecture & Design | 88/100 | 25% | 22.0 |
| Security Practices | 78/100 | 20% | 15.6 |
| Error Handling | 75/100 | 15% | 11.25 |
| Testing | 65/100 | 15% | 9.75 |
| Idioms & Style | 85/100 | 15% | 12.75 |
| Documentation | 95/100 | 10% | 9.5 |
| **TOTAL** | | | **80.85** |

**Final Grade: 81/100 (B)**

---

## 1. Architecture & Design (88/100)

### Strengths
- **Clean interface-based design**: The `runner.Tool` interface allows easy addition of new AI tools
- **Plugin-like architecture**: Tools (claude, codex, gemini) are decoupled from the runner framework
- **Builder pattern**: Well-implemented in `envelope.Envelope` for constructing results
- **Separation of concerns**: Clear package boundaries (runner, tools, settings, bundle, orchestrator)
- **Factory pattern for dependency injection**: `DispatcherFactory` breaks circular dependencies elegantly

### Issues

**ISSUE-A1: Global mutable state with `noTrackStatus` variable**
- Location: `/Users/cliff/Desktop/_code/rcodegen/pkg/runner/runner.go:22`
- Severity: Low
- The package-level variable is modified during flag parsing, which could cause issues in concurrent tests.

```diff
--- a/pkg/runner/runner.go
+++ b/pkg/runner/runner.go
@@ -19,9 +19,6 @@ import (
 	"rcodegen/pkg/settings"
 )

-// noTrackStatus is a package-level variable used by defineToolSpecificFlags
-// to capture the --no-status flag value, which is then applied after flag.Parse()
-var noTrackStatus bool
-
 // Runner orchestrates the execution of a tool
 type Runner struct {
 	Tool         Tool
 	Settings     *settings.Settings
 	TaskConfig   *settings.TaskConfig
 	SettingsOK   bool
+	noTrackStatus bool  // Move to Runner struct
 }
```

---

## 2. Security Practices (78/100)

### Strengths
- **Path traversal protection**: Bundle loading validates names with regex to prevent `../` attacks
- **Trusted script locations**: Python scripts are only loaded from executable dir or `~/.rcodegen/scripts/`
- **Settings file permissions**: Warns if settings.json is world-writable, writes with 0600 permissions
- **Lock directory security**: Created with 0700 permissions (owner only)
- **Input sanitization**: Bundle names and lock identifiers are sanitized before use
- **Glob pattern escaping**: `escapeGlobPattern()` prevents glob injection in grade lookup

### Issues

**ISSUE-S1: Command injection risk in codex tool**
- Location: `/Users/cliff/Desktop/_code/rcodegen/pkg/tools/codex/codex.go:80-85`
- Severity: Medium
- The `cfg.Effort` value is interpolated directly into a shell command argument

```go
"-c", fmt.Sprintf("model_reasoning_effort=\"%s\"", cfg.Effort),
```

If `cfg.Effort` contains special characters like `"; rm -rf /; "`, this could be exploited.

```diff
--- a/pkg/tools/codex/codex.go
+++ b/pkg/tools/codex/codex.go
@@ -77,7 +77,12 @@ func (t *Tool) BuildCommand(cfg *runner.Config, workDir, task string) *exec.Cmd
 		args := []string{
 			wrapperPath,
 			cfg.SessionID,
 			task,
 			"--dangerously-bypass-approvals-and-sandbox",
 			"--model", cfg.Model,
-			"-c", fmt.Sprintf("model_reasoning_effort=\"%s\"", cfg.Effort),
+			"-c", fmt.Sprintf("model_reasoning_effort=%q", sanitizeEffort(cfg.Effort)),
 		}
+
+// Add function to validate effort level:
+func sanitizeEffort(effort string) string {
+	valid := map[string]bool{"low": true, "medium": true, "high": true, "xhigh": true}
+	if valid[effort] {
+		return effort
+	}
+	return "xhigh" // Default fallback
+}
```

**ISSUE-S2: Missing validation on Codex model names**
- Location: `/Users/cliff/Desktop/_code/rcodegen/pkg/tools/codex/codex.go:261-263`
- Severity: Low
- Unlike Claude and Gemini, Codex accepts any model name without validation

```go
func (t *Tool) ValidateConfig(cfg *runner.Config) error {
	// Codex accepts any model name, so no validation needed
	return nil
}
```

```diff
--- a/pkg/tools/codex/codex.go
+++ b/pkg/tools/codex/codex.go
@@ -259,8 +259,18 @@ func (t *Tool) PrepareForExecution(cfg *runner.Config) {

 // ValidateConfig validates Codex-specific configuration
 func (t *Tool) ValidateConfig(cfg *runner.Config) error {
-	// Codex accepts any model name, so no validation needed
+	// Validate effort level
+	validEfforts := []string{"low", "medium", "high", "xhigh"}
+	effortValid := false
+	for _, e := range validEfforts {
+		if cfg.Effort == e {
+			effortValid = true
+			break
+		}
+	}
+	if !effortValid {
+		return fmt.Errorf("invalid effort level '%s'. Valid options: low, medium, high, xhigh", cfg.Effort)
+	}
 	return nil
 }
```

**ISSUE-S3: Python import before iTerm2 check in get_codex_status.py**
- Location: `/Users/cliff/Desktop/_code/rcodegen/get_codex_status.py:15`
- Severity: Low
- The script imports `iterm2` before checking if running in iTerm2, unlike `get_claude_status.py` which checks first

```diff
--- a/get_codex_status.py
+++ b/get_codex_status.py
@@ -12,13 +12,21 @@ Output (JSON to stdout):
     {"5h_left": 64, "weekly_left": 89, "context_left": 52}
 """

-import iterm2
 import asyncio
 import json
 import re
 import sys
 import os
 import tempfile
 from datetime import datetime, timedelta
+
+# Check for iTerm2 environment before importing iterm2 package
+if not os.environ.get('ITERM_SESSION_ID'):
+    print(json.dumps({
+        "error": "not_iterm2",
+        "message": "Not running in iTerm2. Credit tracking requires iTerm2."
+    }))
+    sys.exit(0)
+
+import iterm2
```

---

## 3. Error Handling (75/100)

### Strengths
- **Structured error types**: Uses `RunResult` with exit codes and errors
- **Graceful degradation**: Missing settings trigger interactive setup
- **Retry logic**: Grade extraction retries file reads up to 10 times
- **Atomic file writes**: Grades file uses temp file + rename pattern

### Issues

**ISSUE-E1: Silent error swallowing in orchestrator**
- Location: `/Users/cliff/Desktop/_code/rcodegen/pkg/orchestrator/orchestrator.go:355-359`
- Severity: Medium
- Errors from copying bundle file are logged but not propagated

```go
if bundleData, err := os.ReadFile(b.SourcePath); err == nil {
    if err := os.WriteFile(bundleDest, bundleData, 0644); err != nil {
        fmt.Fprintf(os.Stderr, "Warning: failed to copy bundle to %s: %v\n", bundleDest, err)
    }
}
```

**ISSUE-E2: Panic potential in slice access**
- Location: `/Users/cliff/Desktop/_code/rcodegen/pkg/runner/flags.go:87-93`
- Severity: Low
- If `-x=` is passed without a key, `kv[:idx]` could panic (though `idx > 0` check prevents this)

**ISSUE-E3: Missing error handling for Getwd()**
- Location: Multiple locations
- Severity: Low
- Several places use `os.Getwd()` and ignore the error with `_`

```go
cwd, _ := os.Getwd()  // Error ignored
```

---

## 4. Testing (65/100)

### Coverage Analysis

| Package | Coverage | Notes |
|---------|----------|-------|
| pkg/envelope | 100% | Excellent |
| pkg/workspace | 82.6% | Good |
| pkg/bundle | 54.7% | Acceptable |
| pkg/lock | 45.3% | Needs improvement |
| pkg/runner | 14.6% | Poor - critical code |
| pkg/executor | 14.2% | Poor |
| pkg/orchestrator | 8.7% | Very poor |
| pkg/settings | 5.6% | Very poor |
| pkg/tools/claude | 5.1% | Very poor |
| pkg/tracking | 0% | No tests |
| pkg/reports | 0% | No tests |
| cmd/* | 0% | No tests |

### Issues

**ISSUE-T1: Critical paths untested**
- The main `Runner.Run()` function has minimal test coverage
- No integration tests for the full execution path
- Report generation logic completely untested

**ISSUE-T2: Missing edge case tests**
- No tests for error conditions in file operations
- No tests for concurrent lock acquisition
- No tests for malformed JSON parsing

---

## 5. Idioms & Style (85/100)

### Strengths
- **Consistent formatting**: Code is gofmt compliant
- **Good naming**: Functions and variables have clear, descriptive names
- **Constants for magic values**: Colors, task types, display lengths are constants
- **Comments**: Public functions and types are documented

### Issues

**ISSUE-I1: Color code duplication**
- Location: Multiple packages define their own color constants
- Already partially addressed with `pkg/colors/colors.go`, but not consistently used

**ISSUE-I2: Long functions**
- Location: `/Users/cliff/Desktop/_code/rcodegen/pkg/orchestrator/orchestrator.go`
- `generateRunReport()` is 200+ lines and should be broken down

**ISSUE-I3: Inconsistent error wrapping**
- Some places use `fmt.Errorf("...: %w", err)` and others use `fmt.Errorf("...: %v", err)`

---

## 6. Documentation (95/100)

### Strengths
- **Comprehensive README**: Installation, usage, examples, architecture explained
- **CHANGELOG**: Detailed version history with agent attribution
- **Inline comments**: Complex logic is well-documented
- **Package documentation**: Package-level comments explain purpose
- **Examples in help text**: CLI help includes practical examples

### Issues

**ISSUE-D1: Missing architecture diagram**
- The relationship between packages (runner, tools, orchestrator, executor) could benefit from a diagram

---

## Additional Bug Risks

**ISSUE-B1: Potential race condition in grade file mutex**
- Location: `/Users/cliff/Desktop/_code/rcodegen/pkg/runner/grades.go:52`
- The in-memory mutex doesn't protect against concurrent processes

```go
var gradesFileMutex sync.Mutex  // Only protects within single process
```

Consider using file-based locking for the grades file similar to how locks work:

```diff
--- a/pkg/runner/grades.go
+++ b/pkg/runner/grades.go
@@ -193,9 +193,19 @@ func AppendGrade(reportDir, reportFile, tool, task string, grade float64, date t
 	// Lock to prevent race conditions
 	gradesFileMutex.Lock()
 	defer gradesFileMutex.Unlock()
+
+	// Also use file-based lock for cross-process safety
+	lockPath := filepath.Join(reportDir, ".grades.lock")
+	lockFile, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0600)
+	if err != nil {
+		return fmt.Errorf("could not open grades lock: %w", err)
+	}
+	defer lockFile.Close()
+	if err := syscall.Flock(int(lockFile.Fd()), syscall.LOCK_EX); err != nil {
+		return fmt.Errorf("could not acquire grades lock: %w", err)
+	}
+	defer syscall.Flock(int(lockFile.Fd()), syscall.LOCK_UN)
```

**ISSUE-B2: Potential nil pointer in printDetailedSummary**
- Location: `/Users/cliff/Desktop/_code/rcodegen/pkg/runner/runner.go:544-555`
- If `cfg.TokenUsage` is set but individual fields are accessed without nil checks

---

## Dependency Management

The project uses Go modules with minimal dependencies:
- `go 1.25.5` specified in `go.mod`
- No external dependencies (standard library only)
- Python scripts require `iterm2` package

**ISSUE-DM1: Future Go version in go.mod**
- `go 1.25.5` doesn't exist yet (current is 1.22). This appears to be intentional for future compatibility.

---

## Summary of Recommendations

### High Priority
1. Add validation for Codex effort level to prevent command injection (ISSUE-S1)
2. Increase test coverage for critical paths (runner, settings, tools)
3. Add file-based locking for grades.json (ISSUE-B1)

### Medium Priority
4. Move `noTrackStatus` from package-level to struct field (ISSUE-A1)
5. Fix Python iTerm2 import order in get_codex_status.py (ISSUE-S3)
6. Improve error propagation in orchestrator (ISSUE-E1)

### Low Priority
7. Refactor long functions in orchestrator
8. Standardize error wrapping with `%w` format specifier
9. Add architecture diagram to documentation

---

## Files Reviewed

- `/Users/cliff/Desktop/_code/rcodegen/cmd/rclaude/main.go`
- `/Users/cliff/Desktop/_code/rcodegen/cmd/rcodex/main.go`
- `/Users/cliff/Desktop/_code/rcodegen/cmd/rgemini/main.go`
- `/Users/cliff/Desktop/_code/rcodegen/cmd/rcodegen/main.go`
- `/Users/cliff/Desktop/_code/rcodegen/pkg/runner/*.go`
- `/Users/cliff/Desktop/_code/rcodegen/pkg/tools/claude/claude.go`
- `/Users/cliff/Desktop/_code/rcodegen/pkg/tools/codex/codex.go`
- `/Users/cliff/Desktop/_code/rcodegen/pkg/tools/gemini/gemini.go`
- `/Users/cliff/Desktop/_code/rcodegen/pkg/settings/settings.go`
- `/Users/cliff/Desktop/_code/rcodegen/pkg/bundle/*.go`
- `/Users/cliff/Desktop/_code/rcodegen/pkg/orchestrator/*.go`
- `/Users/cliff/Desktop/_code/rcodegen/pkg/executor/*.go`
- `/Users/cliff/Desktop/_code/rcodegen/pkg/envelope/envelope.go`
- `/Users/cliff/Desktop/_code/rcodegen/pkg/lock/filelock.go`
- `/Users/cliff/Desktop/_code/rcodegen/pkg/workspace/workspace.go`
- `/Users/cliff/Desktop/_code/rcodegen/pkg/reports/manager.go`
- `/Users/cliff/Desktop/_code/rcodegen/pkg/tracking/*.go`
- `/Users/cliff/Desktop/_code/rcodegen/get_claude_status.py`
- `/Users/cliff/Desktop/_code/rcodegen/get_codex_status.py`
- All *_test.go files

---

*Report generated by Claude:Opus 4.5 on 2026-01-28*
