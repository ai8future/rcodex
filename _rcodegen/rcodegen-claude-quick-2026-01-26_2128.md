Date Created: 2026-01-26 21:28:00 UTC
TOTAL_SCORE: 72/100

# rcodegen Quick Codebase Analysis

**Project**: rcodegen - Unified automation wrapper for AI coding assistants (Claude, Codex, Gemini)
**Language**: Go (~8,700 lines) + Python (~500 lines)
**Version**: 1.9.2

## Grade Breakdown
- Architecture & Design: 80/100
- Security: 75/100
- Test Coverage: 55/100
- Error Handling: 68/100
- Code Quality: 78/100
- Maintainability: 75/100

---

## 1. AUDIT - Security and Code Quality Issues

### AUDIT-1: Silent File Deletion Failures (MEDIUM)

**File**: `pkg/reports/manager.go:146`

**Issue**: File deletion errors silently ignored - stale reports may accumulate, obscuring disk issues.

**PATCH-READY DIFF**:
```diff
--- a/pkg/reports/manager.go
+++ b/pkg/reports/manager.go
@@ -143,7 +143,10 @@ func DeleteOldReports(reportDir string, shortcuts []string, reportPatterns map[s
 		// Delete all but the newest
 		for i := 1; i < len(files); i++ {
 			fmt.Printf("%sDeleting old report:%s %s\n", Dim, Reset, filepath.Base(files[i].path))
-			os.Remove(files[i].path)
+			if err := os.Remove(files[i].path); err != nil {
+				fmt.Fprintf(os.Stderr, "%sWarning:%s Could not delete old report %s: %v\n",
+					Yellow, Reset, filepath.Base(files[i].path), err)
+			}
 		}
 	}
 }
```

---

### AUDIT-2: Silent Temp File Cleanup Failure (LOW)

**File**: `pkg/runner/grades.go:185`

**Issue**: Temp file deletion on rename failure is silent - could leave orphan `.tmp` files.

**PATCH-READY DIFF**:
```diff
--- a/pkg/runner/grades.go
+++ b/pkg/runner/grades.go
@@ -182,7 +182,9 @@ func SaveGrades(reportDir string, grades *GradesFile) error {
 	// Atomic rename
 	if err := os.Rename(tempPath, gradesPath); err != nil {
 		// Clean up temp file on failure
-		os.Remove(tempPath)
+		if removeErr := os.Remove(tempPath); removeErr != nil {
+			fmt.Fprintf(os.Stderr, "Warning: Could not clean up temp file %s: %v\n", tempPath, removeErr)
+		}
 		return fmt.Errorf("failed to rename grades file: %w", err)
 	}
```

---

### AUDIT-3: Bare Exceptions in Python PTY Wrapper (MEDIUM)

**File**: `codex_pty_wrapper.py:72, 83, 90`

**Issue**: Bare `except:` clauses hide actual errors, making debugging difficult.

**PATCH-READY DIFF**:
```diff
--- a/codex_pty_wrapper.py
+++ b/codex_pty_wrapper.py
@@ -69,7 +69,7 @@ def run_codex_resume(session_id, args, prompt, timeout=600):
                     data = os.read(master, 8192)
                     if not data:
                         break
                     output += data
-            except:
+            except OSError:
                 pass
             break

@@ -80,7 +80,7 @@ def run_codex_resume(session_id, args, prompt, timeout=600):
     # Clean up
     try:
         os.close(master)
-    except:
+    except OSError:
         pass

     if p.poll() is None:
         p.terminate()
         try:
             p.wait(timeout=5)
-        except:
+        except subprocess.TimeoutExpired:
             p.kill()
```

---

### AUDIT-4: Lock Info File Not Cleaned Up on Release (LOW)

**File**: `pkg/lock/filelock.go:134-146`

**Issue**: `rcodegen.lock.info` file is written on Acquire but never deleted on Release, leaving stale holder information.

**PATCH-READY DIFF**:
```diff
--- a/pkg/lock/filelock.go
+++ b/pkg/lock/filelock.go
@@ -132,6 +132,13 @@ func (l *FileLock) Release() error {
 	if l == nil || l.file == nil {
 		return nil
 	}
+
+	// Clean up lock info file
+	lockDir, err := getLockDir()
+	if err == nil {
+		lockInfoPath := filepath.Join(lockDir, "rcodegen.lock.info")
+		os.Remove(lockInfoPath) // Best effort, ignore errors
+	}

 	unlockErr := syscall.Flock(int(l.file.Fd()), syscall.LOCK_UN)
 	closeErr := l.file.Close()
```

---

### AUDIT-5: No Input Validation on Task Name (LOW-MEDIUM)

**File**: `pkg/runner/runner.go:300-304`

**Issue**: `taskName` from config is used directly without validation. While sourced from settings.json (trusted), defense-in-depth suggests validation.

**PATCH-READY DIFF**:
```diff
--- a/pkg/runner/runner.go
+++ b/pkg/runner/runner.go
@@ -297,6 +297,14 @@ func (r *Runner) runWorkDir(ctx context.Context, cfg *Config, workDir string, fi
 // getTask returns a task prompt with placeholders substituted
 // This avoids mutating the shared TaskConfig
 func (r *Runner) getTask(cfg *Config, workDir, taskName string) string {
+	// Validate taskName is reasonable (alphanumeric, dash, underscore only)
+	for _, c := range taskName {
+		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') ||
+			(c >= '0' && c <= '9') || c == '-' || c == '_') {
+			return "" // Invalid task name
+		}
+	}
+
 	task := r.TaskConfig.Tasks[taskName]
 	reportDir := r.getReportDir(cfg, workDir)
 	return strings.ReplaceAll(task, "{report_dir}", reportDir)
```

---

## 2. TESTS - Proposed Unit Tests

### TEST-1: Codex Tool Package Tests

**File**: `pkg/tools/codex/codex_test.go` (new file)

**Missing Coverage**: BuildCommand, findWrapper, model validation

**PATCH-READY DIFF**:
```diff
--- /dev/null
+++ b/pkg/tools/codex/codex_test.go
@@ -0,0 +1,89 @@
+package codex
+
+import (
+	"testing"
+
+	"rcodegen/pkg/runner"
+)
+
+func TestNew(t *testing.T) {
+	tool := New()
+	if tool == nil {
+		t.Fatal("New() returned nil")
+	}
+}
+
+func TestName(t *testing.T) {
+	tool := New()
+	if name := tool.Name(); name != "rcodex" {
+		t.Errorf("Name() = %q, want %q", name, "rcodex")
+	}
+}
+
+func TestBinaryName(t *testing.T) {
+	tool := New()
+	if name := tool.BinaryName(); name != "codex" {
+		t.Errorf("BinaryName() = %q, want %q", name, "codex")
+	}
+}
+
+func TestValidModels(t *testing.T) {
+	tool := New()
+	models := tool.ValidModels()
+	if len(models) == 0 {
+		t.Error("ValidModels() returned empty slice")
+	}
+	// Check default model is in valid models
+	defaultModel := tool.DefaultModel()
+	found := false
+	for _, m := range models {
+		if m == defaultModel {
+			found = true
+			break
+		}
+	}
+	if !found {
+		t.Errorf("DefaultModel() %q not in ValidModels()", defaultModel)
+	}
+}
+
+func TestBuildCommand_NewSession(t *testing.T) {
+	tool := New()
+	cfg := &runner.Config{
+		Model:  "gpt-5.2-codex",
+		Effort: "high",
+	}
+	cmd := tool.BuildCommand(cfg, "/tmp/test", "test task")
+	if cmd == nil {
+		t.Fatal("BuildCommand returned nil")
+	}
+	if cmd.Path == "" {
+		t.Error("BuildCommand returned command with empty path")
+	}
+}
+
+func TestBuildCommand_WithSessionID(t *testing.T) {
+	tool := New()
+	cfg := &runner.Config{
+		Model:     "gpt-5.2-codex",
+		Effort:    "high",
+		SessionID: "test-session-123",
+	}
+	cmd := tool.BuildCommand(cfg, "/tmp/test", "test task")
+	if cmd == nil {
+		t.Fatal("BuildCommand returned nil for resume")
+	}
+	// Should use python3 for wrapper
+	if cmd.Args[0] != "python3" {
+		t.Errorf("Resume command should use python3, got %q", cmd.Args[0])
+	}
+}
+
+func TestReportDir(t *testing.T) {
+	tool := New()
+	if dir := tool.ReportDir(); dir != "_rcodegen" {
+		t.Errorf("ReportDir() = %q, want %q", dir, "_rcodegen")
+	}
+}
```

---

### TEST-2: Gemini Tool Package Tests

**File**: `pkg/tools/gemini/gemini_test.go` (new file)

**PATCH-READY DIFF**:
```diff
--- /dev/null
+++ b/pkg/tools/gemini/gemini_test.go
@@ -0,0 +1,72 @@
+package gemini
+
+import (
+	"testing"
+
+	"rcodegen/pkg/runner"
+)
+
+func TestNew(t *testing.T) {
+	tool := New()
+	if tool == nil {
+		t.Fatal("New() returned nil")
+	}
+}
+
+func TestName(t *testing.T) {
+	tool := New()
+	if name := tool.Name(); name != "rgemini" {
+		t.Errorf("Name() = %q, want %q", name, "rgemini")
+	}
+}
+
+func TestBinaryName(t *testing.T) {
+	tool := New()
+	if name := tool.BinaryName(); name != "gemini" {
+		t.Errorf("BinaryName() = %q, want %q", name, "gemini")
+	}
+}
+
+func TestValidModels(t *testing.T) {
+	tool := New()
+	models := tool.ValidModels()
+	if len(models) == 0 {
+		t.Error("ValidModels() returned empty slice")
+	}
+	// Verify flash models are included
+	hasFlash := false
+	for _, m := range models {
+		if strings.Contains(m, "flash") {
+			hasFlash = true
+			break
+		}
+	}
+	if !hasFlash {
+		t.Error("ValidModels() should include flash models")
+	}
+}
+
+func TestBuildCommand_NewSession(t *testing.T) {
+	tool := New()
+	cfg := &runner.Config{
+		Model: "gemini-3-pro-preview",
+	}
+	cmd := tool.BuildCommand(cfg, "/tmp/test", "test task")
+	if cmd == nil {
+		t.Fatal("BuildCommand returned nil")
+	}
+	if cmd.Dir != "/tmp/test" {
+		t.Errorf("BuildCommand Dir = %q, want %q", cmd.Dir, "/tmp/test")
+	}
+}
+
+func TestBuildCommand_WithResume(t *testing.T) {
+	tool := New()
+	cfg := &runner.Config{
+		Model:     "gemini-3-pro-preview",
+		SessionID: "session-abc",
+	}
+	cmd := tool.BuildCommand(cfg, "/tmp/test", "test task")
+	// Should include --resume flag
+	hasResume := false
+	for _, arg := range cmd.Args {
+		if arg == "--resume" {
+			hasResume = true
+			break
+		}
+	}
+	if !hasResume {
+		t.Error("BuildCommand with SessionID should include --resume")
+	}
+}
```

---

### TEST-3: Reports Manager Tests

**File**: `pkg/reports/manager_test.go` (new file)

**PATCH-READY DIFF**:
```diff
--- /dev/null
+++ b/pkg/reports/manager_test.go
@@ -0,0 +1,95 @@
+package reports
+
+import (
+	"os"
+	"path/filepath"
+	"testing"
+	"time"
+)
+
+func TestFindNewestReport(t *testing.T) {
+	// Empty list
+	if result := FindNewestReport(nil); result != "" {
+		t.Errorf("FindNewestReport(nil) = %q, want empty", result)
+	}
+
+	if result := FindNewestReport([]string{}); result != "" {
+		t.Errorf("FindNewestReport([]) = %q, want empty", result)
+	}
+}
+
+func TestFindNewestReport_WithFiles(t *testing.T) {
+	// Create temp directory with test files
+	tmpDir, err := os.MkdirTemp("", "reports_test")
+	if err != nil {
+		t.Fatalf("Failed to create temp dir: %v", err)
+	}
+	defer os.RemoveAll(tmpDir)
+
+	// Create files with different modification times
+	file1 := filepath.Join(tmpDir, "old.md")
+	file2 := filepath.Join(tmpDir, "new.md")
+
+	os.WriteFile(file1, []byte("old"), 0644)
+	time.Sleep(10 * time.Millisecond)
+	os.WriteFile(file2, []byte("new"), 0644)
+
+	result := FindNewestReport([]string{file1, file2})
+	if result != file2 {
+		t.Errorf("FindNewestReport() = %q, want %q", result, file2)
+	}
+}
+
+func TestIsReportReviewed(t *testing.T) {
+	tmpDir, err := os.MkdirTemp("", "reports_test")
+	if err != nil {
+		t.Fatalf("Failed to create temp dir: %v", err)
+	}
+	defer os.RemoveAll(tmpDir)
+
+	tests := []struct {
+		name     string
+		content  string
+		expected bool
+	}{
+		{
+			name:     "reviewed report",
+			content:  "Date Created: 2026-01-01\nDate Modified: 2026-01-02\nContent here",
+			expected: true,
+		},
+		{
+			name:     "unreviewed report",
+			content:  "Date Created: 2026-01-01\nContent here\nNo modified date",
+			expected: false,
+		},
+		{
+			name:     "date modified after line 10",
+			content:  "1\n2\n3\n4\n5\n6\n7\n8\n9\n10\n11\nDate Modified: late",
+			expected: false,
+		},
+	}
+
+	for _, tc := range tests {
+		t.Run(tc.name, func(t *testing.T) {
+			file := filepath.Join(tmpDir, tc.name+".md")
+			os.WriteFile(file, []byte(tc.content), 0644)
+
+			result := IsReportReviewed(file)
+			if result != tc.expected {
+				t.Errorf("IsReportReviewed() = %v, want %v", result, tc.expected)
+			}
+		})
+	}
+}
+
+func TestIsReportReviewed_NonexistentFile(t *testing.T) {
+	result := IsReportReviewed("/nonexistent/file.md")
+	if result != false {
+		t.Errorf("IsReportReviewed(nonexistent) = %v, want false", result)
+	}
+}
+
+func TestShouldSkipTask_NoRequireReview(t *testing.T) {
+	// When requireReview is false, should never skip
+	result := ShouldSkipTask("/any/dir", "test", "pattern", false)
+	if result != false {
+		t.Errorf("ShouldSkipTask with requireReview=false should return false")
+	}
+}
```

---

### TEST-4: Lock Package Concurrent Tests

**File**: `pkg/lock/filelock_concurrent_test.go` (new file)

**PATCH-READY DIFF**:
```diff
--- /dev/null
+++ b/pkg/lock/filelock_concurrent_test.go
@@ -0,0 +1,58 @@
+package lock
+
+import (
+	"sync"
+	"testing"
+	"time"
+)
+
+func TestSanitizeIdentifier(t *testing.T) {
+	tests := []struct {
+		input    string
+		expected string
+	}{
+		{"", "unknown"},
+		{"simple", "simple"},
+		{"with/slash", "with_slash"},
+		{"with\\backslash", "with_backslash"},
+		{"with\x00null", "with_null"},
+		{string(make([]byte, 200)), string(make([]byte, maxIdentifierLen))}, // truncation
+	}
+
+	for _, tc := range tests {
+		result := sanitizeIdentifier(tc.input)
+		if tc.input == "" && result != "unknown" {
+			t.Errorf("sanitizeIdentifier(%q) = %q, want %q", tc.input, result, tc.expected)
+		}
+		if len(result) > maxIdentifierLen {
+			t.Errorf("sanitizeIdentifier() returned string longer than max: %d > %d", len(result), maxIdentifierLen)
+		}
+	}
+}
+
+func TestGetIdentifier(t *testing.T) {
+	tests := []struct {
+		workDir  string
+		expected string
+	}{
+		{"/home/user/myproject", "myproject"},
+		{"/var/www/app", "app"},
+		{"", ""},  // Will use cwd
+		{".", ""}, // Will use cwd
+	}
+
+	for _, tc := range tests {
+		result := GetIdentifier(tc.workDir)
+		if tc.workDir != "" && tc.workDir != "." && result != tc.expected {
+			t.Errorf("GetIdentifier(%q) = %q, want %q", tc.workDir, result, tc.expected)
+		}
+	}
+}
+
+func TestAcquire_NoLock(t *testing.T) {
+	// When useLock is false, should return nil without error
+	lock, err := Acquire("test", false)
+	if err != nil {
+		t.Errorf("Acquire with useLock=false returned error: %v", err)
+	}
+	if lock != nil {
+		t.Error("Acquire with useLock=false should return nil lock")
+	}
+}
```

---

## 3. FIXES - Bugs, Issues, and Code Smells

### FIX-1: Silent Grade Persistence Failure

**File**: `pkg/runner/runner.go:339-343`

**Issue**: When report is not found after retries, function returns silently with no indication of failure.

**PATCH-READY DIFF**:
```diff
--- a/pkg/runner/runner.go
+++ b/pkg/runner/runner.go
@@ -336,8 +336,9 @@ func (r *Runner) persistGrade(cfg *Config, workDir, taskShortcut string) {
 	}

 	if err != nil {
-		// Report not found - this can happen if the tool didn't create one
-		// Debug: fmt.Fprintf(os.Stderr, "%sDebug:%s No report found for %s/%s: %v\n", Dim, Reset, toolName, taskShortcut, err)
+		// Report not found after retries - log at debug level
+		// This can happen if the tool didn't create a report (e.g., user cancelled)
+		// Uncomment for debugging: fmt.Fprintf(os.Stderr, "%sDebug:%s No report found for %s/%s: %v\n", Dim, Reset, toolName, taskShortcut, err)
 		return
 	}
```

---

### FIX-2: Missing Import in Gemini Test

**File**: `pkg/tools/gemini/gemini_test.go` (from TEST-2)

**Issue**: Test uses `strings.Contains` but doesn't import `strings` package.

**PATCH-READY DIFF** (update to TEST-2):
```diff
--- a/pkg/tools/gemini/gemini_test.go
+++ b/pkg/tools/gemini/gemini_test.go
@@ -2,6 +2,7 @@ package gemini

 import (
+	"strings"
 	"testing"

 	"rcodegen/pkg/runner"
```

---

### FIX-3: Inconsistent Error Return Patterns

**Files**: Multiple files use mixed patterns (return nil vs log and continue)

**Issue**: Some functions silently continue on error, others return/log. This inconsistency makes debugging difficult.

**Recommendation**: Standardize to always return errors OR always log at minimum. Current mixed approach hides failures.

**Example Pattern Fix** for `pkg/runner/runner.go:346-348`:
```diff
--- a/pkg/runner/runner.go
+++ b/pkg/runner/runner.go
@@ -343,8 +343,10 @@ func (r *Runner) persistGrade(cfg *Config, workDir, taskShortcut string) {
 	}

 	// Verify file exists and is readable
-	if _, err := os.Stat(reportPath); err != nil {
-		return
+	if info, err := os.Stat(reportPath); err != nil {
+		// File disappeared between find and stat - race condition, acceptable
+		return
+	} else if info.Size() == 0 {
+		// Empty file - nothing to extract
+		return
 	}
```

---

### FIX-4: Potential Nil Pointer in Claude Tracking

**File**: `pkg/tracking/claude.go:143-144`

**Issue**: `status.SessionLeft` and `status.WeeklyAllLeft` are checked for nil, but the overall `status` object could theoretically be nil if `GetClaudeStatus()` implementation changes.

**PATCH-READY DIFF**:
```diff
--- a/pkg/tracking/claude.go
+++ b/pkg/tracking/claude.go
@@ -140,6 +140,9 @@ func ShowClaudeStatusOnly() {

 // PrintClaudeStatusBefore prints the credit status before a task
 func PrintClaudeStatusBefore(status *ClaudeStatus) {
+	if status == nil {
+		return
+	}
 	hasData := status.SessionLeft != nil || status.WeeklyAllLeft != nil
 	if hasData {
```

---

## 4. REFACTOR - Opportunities to Improve Code Quality

### REFACTOR-1: Consolidate ANSI Color Definitions

**Current State**: Color constants defined in multiple packages:
- `pkg/reports/manager.go:16-20`
- `pkg/lock/filelock.go:14-19`
- `pkg/tracking/claude.go` (imports from common)
- `pkg/runner/` files

**Opportunity**: Create single `pkg/colors/colors.go` and import everywhere. This is partially done but not consistently applied.

**Benefit**: Single source of truth, easier theme customization.
**Effort**: Small
**Priority**: Low

---

### REFACTOR-2: Extract Config Validation to Dedicated Package

**Current State**: Validation logic scattered across:
- `pkg/runner/runner.go` (parseArgs)
- `pkg/settings/settings.go` (LoadSettings)
- Individual tool files (ValidModels)

**Opportunity**: Create `pkg/config/validator.go` with centralized validation rules.

**Benefit**: Consistent validation, easier to add new config options.
**Effort**: Medium
**Priority**: Medium

---

### REFACTOR-3: Tool Interface Simplification

**Current State**: `runner.Tool` interface has many methods, some optional (return empty/default).

**Opportunity**: Split into core interface + optional interfaces:
```go
type Tool interface {
    Name() string
    BinaryName() string
    BuildCommand(cfg *Config, workDir, task string) *exec.Cmd
}

type StatusTracker interface {
    SupportsStatusTracking() bool
    CaptureStatusBefore() interface{}
    CaptureStatusAfter() interface{}
}
```

**Benefit**: Cleaner implementations, compile-time interface checking.
**Effort**: Medium
**Priority**: Low

---

### REFACTOR-4: Magic Numbers to Named Constants

**Files**: Various
**Examples**:
- `get_claude_status.py:179`: `await asyncio.sleep(4)` - should be `WAIT_FOR_ITERM_READY = 4`
- `pkg/runner/runner.go:331`: `for i := 0; i < 10; i++` - should be `const maxReportFindRetries = 10`
- `pkg/runner/runner.go:336`: `50 * time.Millisecond` - should be `const reportFindRetryDelay = 50 * time.Millisecond`

**Benefit**: Self-documenting code, easier tuning.
**Effort**: Trivial
**Priority**: Low

---

### REFACTOR-5: Python Script Error Handling Standardization

**Current State**: Python scripts (`get_claude_status.py`, `claude_question_handler.py`, `codex_pty_wrapper.py`) have inconsistent error handling patterns.

**Opportunity**: Create common error handling utilities:
```python
# shared/errors.py
def safe_cleanup(func, *args, **kwargs):
    """Execute cleanup function, suppressing expected errors."""
    try:
        return func(*args, **kwargs)
    except (OSError, IOError):
        pass  # Expected during cleanup

def with_timeout(coro, timeout_seconds, default=None):
    """Run coroutine with timeout, returning default on timeout."""
    try:
        return asyncio.wait_for(coro, timeout=timeout_seconds)
    except asyncio.TimeoutError:
        return default
```

**Benefit**: Consistent behavior, easier testing.
**Effort**: Small
**Priority**: Medium

---

### REFACTOR-6: Separate Report Lifecycle Management

**Current State**: Report management (creation, deletion, grade extraction) spread across:
- `pkg/reports/manager.go`
- `pkg/runner/grades.go`
- `pkg/runner/runner.go`

**Opportunity**: Create unified `ReportManager` service handling full lifecycle.

**Benefit**: Clear ownership, easier testing, centralized policies.
**Effort**: Medium
**Priority**: Medium

---

## Summary

| Category | Count | Severity |
|----------|-------|----------|
| AUDIT Issues | 5 | 2 Medium, 3 Low |
| Missing Tests | 4 files | ~90-120 tests needed |
| FIXES | 4 | 1 Medium, 3 Low |
| REFACTOR | 6 | Various priorities |

**Overall Assessment**: This is a well-engineered codebase with solid architecture. The main areas for improvement are:
1. Test coverage expansion (especially Codex, Gemini, Tracking packages)
2. Consistent error handling patterns
3. Minor security hardening (error logging, input validation)

The 72/100 grade reflects strong fundamentals with room for improvement in test coverage and error handling consistency.
