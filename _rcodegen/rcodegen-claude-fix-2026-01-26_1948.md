Date Created: 2026-01-26 19:48:00 UTC
TOTAL_SCORE: 78/100

# rcodegen Codebase Audit Report

**Auditor:** Claude:Opus 4.5
**Date:** 2026-01-26
**Codebase:** rcodegen
**Purpose:** Bug detection, code smell identification, and patch-ready fixes

---

## Executive Summary

The rcodegen codebase is a well-structured Go monorepo providing automation wrappers for multiple AI coding assistants (Claude, Codex, Gemini). The code demonstrates solid architecture with good separation of concerns, but several issues warrant attention:

- **Critical Issues (0):** None found
- **High Severity (3):** Race condition in status caching, error handling in file operations, security concern in script path resolution
- **Medium Severity (9):** Ignored errors, missing cleanup, input validation gaps
- **Low Severity (6):** Hardcoded values, minor code smells

**Overall Grade: 78/100** - Good quality codebase with room for improvement in error handling and resource management.

---

## Issues Found

### HIGH-1: Race Condition in Claude Tool Status Caching

**File:** `pkg/tools/claude/claude.go`
**Lines:** 20-41, 149-155
**Severity:** HIGH

**Description:** The `cachedStatus` field is accessed without synchronization. While `checkOnce.Do()` protects the initial write, the cached value is later read and cleared in `CaptureStatusBefore()` without synchronization, creating a data race.

**Current Code:**
```go
// Lines 20-24
type Tool struct {
    settings     *settings.Settings
    currentModel string
    checkOnce    sync.Once
    isClaudeMax  bool
    cachedStatus *tracking.ClaudeStatus // Race condition: unsynchronized access
}

// Lines 149-155 - Unsynchronized read and clear
func (t *Tool) CaptureStatusBefore() interface{} {
    if t.cachedStatus != nil {
        status := t.cachedStatus
        t.cachedStatus = nil // RACE: another goroutine could be reading
        return status
    }
    // ...
}
```

**Patch-Ready Diff:**
```diff
--- a/pkg/tools/claude/claude.go
+++ b/pkg/tools/claude/claude.go
@@ -17,11 +17,12 @@ type Tool struct {
 	settings     *settings.Settings
 	currentModel string // Track current model for status calculations

-	// Thread-safe status caching using sync.Once
+	// Thread-safe status caching
 	checkOnce    sync.Once
+	statusMu     sync.Mutex
 	isClaudeMax  bool                   // True if user has Claude Max subscription
 	cachedStatus *tracking.ClaudeStatus // Cached status from initial check
 }

 // New creates a new Claude tool
@@ -32,6 +33,8 @@ func New() *Tool {
 // checkClaudeMax checks if user has Claude Max subscription and caches the result
 func (t *Tool) checkClaudeMax() {
 	t.checkOnce.Do(func() {
+		t.statusMu.Lock()
+		defer t.statusMu.Unlock()
 		// Try to get status - if successful, user has Claude Max
 		status := tracking.GetClaudeStatus()
 		if status.Error == "" && (status.SessionLeft != nil || status.WeeklyAllLeft != nil) {
@@ -147,9 +150,12 @@ func (t *Tool) SupportsStatusTracking() bool {

 // CaptureStatusBefore captures Claude Max credit status before tasks
 func (t *Tool) CaptureStatusBefore() interface{} {
-	// Use cached status if we already checked for Claude Max
+	// Use cached status if we already checked for Claude Max (thread-safe)
+	t.statusMu.Lock()
 	if t.cachedStatus != nil {
 		status := t.cachedStatus
 		t.cachedStatus = nil // Clear cache so we fetch fresh after
+		t.statusMu.Unlock()
 		return status
 	}
+	t.statusMu.Unlock()

 	status := tracking.GetClaudeStatus()
```

---

### HIGH-2: Ignored os.Remove() Errors in Critical Paths

**File:** `pkg/runner/grades.go`
**Line:** 185
**Severity:** HIGH

**Description:** When atomic rename fails, the temporary file cleanup error is silently ignored. This could lead to disk space exhaustion if temp files accumulate.

**Current Code:**
```go
// Lines 182-187
if err := os.Rename(tempPath, gradesPath); err != nil {
    // Clean up temp file on failure
    os.Remove(tempPath) // ERROR IGNORED
    return fmt.Errorf("failed to rename grades file: %w", err)
}
```

**Patch-Ready Diff:**
```diff
--- a/pkg/runner/grades.go
+++ b/pkg/runner/grades.go
@@ -181,8 +181,10 @@ func SaveGrades(reportDir string, grades *GradesFile) error {
 	// Atomic rename
 	if err := os.Rename(tempPath, gradesPath); err != nil {
 		// Clean up temp file on failure
-		os.Remove(tempPath)
-		return fmt.Errorf("failed to rename grades file: %w", err)
+		if rmErr := os.Remove(tempPath); rmErr != nil {
+			return fmt.Errorf("failed to rename grades file: %w (also failed to remove temp file: %v)", err, rmErr)
+		}
+		return fmt.Errorf("failed to rename grades file: %w", err)
 	}

 	return nil
```

---

### HIGH-3: Security Risk in Script Path Resolution

**File:** `pkg/tools/codex/codex.go`
**Lines:** 122-128
**Severity:** HIGH (Security)

**Description:** The `findWrapper()` function searches for `codex_pty_wrapper.py` in the current working directory. An attacker could place a malicious script in a project directory that would be executed with the user's permissions.

**Current Code:**
```go
// Lines 122-128
// 2. Check current working directory
if cwd, err := os.Getwd(); err == nil {
    path := filepath.Join(cwd, wrapperName)
    if _, err := os.Stat(path); err == nil {
        return path // Returns potentially attacker-controlled path
    }
}
```

**Patch-Ready Diff:**
```diff
--- a/pkg/tools/codex/codex.go
+++ b/pkg/tools/codex/codex.go
@@ -110,21 +110,15 @@ func (t *Tool) BuildCommand(cfg *runner.Config, workDir, task string) *exec.Cmd
 func (t *Tool) findWrapper() string {
 	const wrapperName = "codex_pty_wrapper.py"

-	// 1. Check same directory as executable
+	// 1. Check same directory as executable (trusted location)
 	exe, err := os.Executable()
 	if err == nil {
 		path := filepath.Join(filepath.Dir(exe), wrapperName)
 		if _, err := os.Stat(path); err == nil {
 			return path
 		}
 	}

-	// 2. Check current working directory
-	if cwd, err := os.Getwd(); err == nil {
-		path := filepath.Join(cwd, wrapperName)
-		if _, err := os.Stat(path); err == nil {
-			return path
-		}
-	}
+	// REMOVED: CWD check is a security risk - attackers could place malicious scripts

-	// 3. Check ~/.rcodegen/
+	// 2. Check ~/.rcodegen/ (trusted user config location)
 	home := os.Getenv("HOME")
+	if home == "" {
+		if h, err := os.UserHomeDir(); err == nil {
+			home = h
+		}
+	}
+	if home == "" {
+		// Fallback for dev environment
+		return wrapperName
+	}
 	path := filepath.Join(home, ".rcodegen", wrapperName)
 	if _, err := os.Stat(path); err == nil {
 		return path
 	}

-	// Fallback (mostly for dev environment if CWD check failed)
+	// Fallback: assume it's in PATH (for dev environment)
 	return wrapperName
 }
```

---

### MEDIUM-1: Ignored Error in Report Deletion

**File:** `pkg/reports/manager.go`
**Lines:** 144-147
**Severity:** MEDIUM

**Description:** Errors from deleting old reports are silently ignored. Users should be notified if cleanup fails.

**Current Code:**
```go
// Lines 143-147
for i := 1; i < len(files); i++ {
    fmt.Printf("%sDeleting old report:%s %s\n", Dim, Reset, filepath.Base(files[i].path))
    os.Remove(files[i].path) // ERROR IGNORED
}
```

**Patch-Ready Diff:**
```diff
--- a/pkg/reports/manager.go
+++ b/pkg/reports/manager.go
@@ -141,8 +141,10 @@ func DeleteOldReports(reportDir string, shortcuts []string, reportPatterns map[s

 		// Delete all but the newest
 		for i := 1; i < len(files); i++ {
-			fmt.Printf("%sDeleting old report:%s %s\n", Dim, Reset, filepath.Base(files[i].path))
-			os.Remove(files[i].path)
+			if err := os.Remove(files[i].path); err != nil {
+				fmt.Printf("%sWarning: could not delete old report:%s %s: %v\n", Yellow, Reset, filepath.Base(files[i].path), err)
+			} else {
+				fmt.Printf("%sDeleting old report:%s %s\n", Dim, Reset, filepath.Base(files[i].path))
+			}
 		}
 	}
 }
```

---

### MEDIUM-2: Unchecked defer logFile.Close() Error

**File:** `pkg/executor/tool.go`
**Lines:** 74-78
**Severity:** MEDIUM

**Description:** The deferred `logFile.Close()` error is not checked, potentially losing data if the file system is full or the file is corrupted.

**Current Code:**
```go
// Lines 74-78
if logErr == nil {
    cmd.Stdout = io.MultiWriter(&stdout, logFile)
    cmd.Stderr = io.MultiWriter(&stderr, logFile)
    defer logFile.Close() // Error from Close() is ignored
}
```

**Patch-Ready Diff:**
```diff
--- a/pkg/executor/tool.go
+++ b/pkg/executor/tool.go
@@ -71,10 +71,17 @@ func (e *ToolExecutor) Execute(step *bundle.Step, ctx *orchestrator.Context, ws
 	logPath := filepath.Join(logDir, step.Name+".log")
 	logFile, logErr := os.Create(logPath)

+	var closeLogFile func()
 	var stdout, stderr bytes.Buffer
 	if logErr == nil {
 		// Write to both buffer and log file simultaneously
 		cmd.Stdout = io.MultiWriter(&stdout, logFile)
 		cmd.Stderr = io.MultiWriter(&stderr, logFile)
-		defer logFile.Close()
+		closeLogFile = func() {
+			if err := logFile.Close(); err != nil {
+				fmt.Fprintf(os.Stderr, "Warning: failed to close log file %s: %v\n", logPath, err)
+			}
+		}
+		defer closeLogFile()
 	} else {
 		// Fallback to buffer only
 		cmd.Stdout = &stdout
```

---

### MEDIUM-3: Lock Info File Write Error Ignored

**File:** `pkg/lock/filelock.go`
**Line:** 127
**Severity:** MEDIUM

**Description:** The WriteFile for lock info only prints a warning to stderr but continues execution. While the current behavior is acceptable, the error message should be more informative.

**Current Code:**
```go
// Lines 126-129
if err := os.WriteFile(lockInfoPath, []byte(identifier), 0600); err != nil {
    fmt.Fprintf(os.Stderr, "%sWarning: could not write lock info: %v%s\n", Dim, err, Reset)
}
```

**Assessment:** The current implementation is actually reasonable - it warns but doesn't fail. No change needed.

---

### MEDIUM-4: Workspace Cleanup Not Implemented

**File:** `pkg/workspace/workspace.go`
**Lines:** 32-47
**Severity:** MEDIUM

**Description:** Workspace directories are created but never cleaned up, potentially leading to disk space issues over time as job directories accumulate in `~/.rcodegen/workspace/jobs/`.

**Patch-Ready Diff:**
```diff
--- a/pkg/workspace/workspace.go
+++ b/pkg/workspace/workspace.go
@@ -66,3 +66,28 @@ func (w *Workspace) WriteOutput(stepName string, data interface{}) (string, erro
 	}
 	return path, nil
 }
+
+// Cleanup removes the workspace directory and all its contents.
+// Call this when done with a job to free disk space.
+func (w *Workspace) Cleanup() error {
+	if w.JobDir == "" {
+		return nil
+	}
+	return os.RemoveAll(w.JobDir)
+}
+
+// CleanupOldJobs removes job directories older than the specified duration.
+// This can be called periodically to prevent disk space exhaustion.
+func CleanupOldJobs(baseDir string, maxAge time.Duration) error {
+	jobsDir := filepath.Join(baseDir, "jobs")
+	entries, err := os.ReadDir(jobsDir)
+	if err != nil {
+		return err
+	}
+	cutoff := time.Now().Add(-maxAge)
+	for _, entry := range entries {
+		if info, err := entry.Info(); err == nil && info.ModTime().Before(cutoff) {
+			os.RemoveAll(filepath.Join(jobsDir, entry.Name()))
+		}
+	}
+	return nil
+}
```

---

### MEDIUM-5: Empty HOME Environment Variable Not Validated

**File:** `pkg/settings/settings.go`
**Lines:** 77-103
**Severity:** MEDIUM

**Description:** When `os.UserHomeDir()` fails, the code falls back to `os.Getenv("HOME")` without checking if it's empty. This could result in malformed paths.

**Current Code:**
```go
// Lines 77-91
func expandTilde(path string) string {
    if path == "" {
        return path
    }
    if strings.HasPrefix(path, "~/") {
        home, err := os.UserHomeDir()
        if err != nil {
            home = os.Getenv("HOME") // fallback for legacy systems
            if home == "" {
                return path // Returns "~/something" literally
            }
        }
        return filepath.Join(home, path[2:])
    }
    // ...
}
```

**Assessment:** The current implementation actually handles the empty HOME case correctly by returning the path unchanged. The behavior is correct, though a log message would help debugging. No change strictly needed.

---

### MEDIUM-6: Bare except in Python PTY Wrapper

**File:** `codex_pty_wrapper.py`
**Lines:** 72, 83, 90
**Severity:** MEDIUM

**Description:** Bare `except:` clauses catch all exceptions including `SystemExit` and `KeyboardInterrupt`, which can make the script harder to terminate cleanly.

**Current Code:**
```python
# Line 72
except:
    pass

# Line 83
except:
    pass

# Line 90
except:
    p.kill()
```

**Patch-Ready Diff:**
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
@@ -87,7 +87,7 @@ def run_codex_resume(session_id, args, prompt, timeout=600):
         p.terminate()
         try:
             p.wait(timeout=5)
-        except:
+        except subprocess.TimeoutExpired:
             p.kill()

     # Decode and clean output
```

---

### MEDIUM-7: Output Path Error Ignored in Executor

**File:** `pkg/executor/tool.go`
**Lines:** 94-97
**Severity:** MEDIUM

**Description:** The `WriteOutput` error is silently ignored, which could cause downstream issues if the output file wasn't created properly.

**Current Code:**
```go
// Lines 94-97
outputPath, _ := ws.WriteOutput(step.Name, map[string]interface{}{
    "stdout": stdout.String(),
    "stderr": stderr.String(),
})
```

**Patch-Ready Diff:**
```diff
--- a/pkg/executor/tool.go
+++ b/pkg/executor/tool.go
@@ -91,9 +91,12 @@ func (e *ToolExecutor) Execute(step *bundle.Step, ctx *orchestrator.Context, ws
 	}

 	// Write output
-	outputPath, _ := ws.WriteOutput(step.Name, map[string]interface{}{
+	outputPath, writeErr := ws.WriteOutput(step.Name, map[string]interface{}{
 		"stdout": stdout.String(),
 		"stderr": stderr.String(),
 	})
+	if writeErr != nil {
+		fmt.Fprintf(os.Stderr, "Warning: failed to write step output: %v\n", writeErr)
+	}

 	// Build envelope
 	builder := envelope.New().
```

---

### MEDIUM-8: MkdirAll Error Ignored in Executor

**File:** `pkg/executor/tool.go`
**Line:** 69
**Severity:** MEDIUM

**Description:** The `os.MkdirAll` error is ignored when creating the log directory.

**Current Code:**
```go
// Lines 68-69
logDir := filepath.Join(ws.JobDir, "logs")
os.MkdirAll(logDir, 0755) // ERROR IGNORED
```

**Patch-Ready Diff:**
```diff
--- a/pkg/executor/tool.go
+++ b/pkg/executor/tool.go
@@ -66,7 +66,9 @@ func (e *ToolExecutor) Execute(step *bundle.Step, ctx *orchestrator.Context, ws

 	// Create log file for real-time output
 	logDir := filepath.Join(ws.JobDir, "logs")
-	os.MkdirAll(logDir, 0755)
+	if err := os.MkdirAll(logDir, 0755); err != nil {
+		fmt.Fprintf(os.Stderr, "Warning: failed to create log directory: %v\n", err)
+	}
 	logPath := filepath.Join(logDir, step.Name+".log")
 	logFile, logErr := os.Create(logPath)
```

---

### MEDIUM-9: Config Mutation During Execution

**File:** `pkg/runner/runner.go`
**Lines:** 106-111
**Severity:** MEDIUM

**Description:** The `cfg.Task` string is mutated in-place multiple times with string replacements. This makes debugging difficult and could cause issues if the config is used elsewhere.

**Assessment:** This is a code smell but not a bug. The pattern is intentional - the config is only used within this run. Consider documenting this behavior rather than changing it. No immediate fix needed.

---

### LOW-1: Hardcoded Lock Timeout

**File:** `pkg/lock/filelock.go`
**Line:** 24
**Severity:** LOW

**Description:** Lock timeout is hardcoded to 5 minutes with no way to configure it.

**Current Code:**
```go
const (
    lockTimeout      = 5 * time.Minute
    lockPollInterval = 5 * time.Second
    maxIdentifierLen = 100
)
```

**Recommendation:** Consider making this configurable via settings.json for users with long-running tasks. No immediate fix needed.

---

### LOW-2: Hardcoded Review Scan Lines

**File:** `pkg/reports/manager.go`
**Line:** 23
**Severity:** LOW

**Description:** The number of lines scanned for the review marker is hardcoded to 10.

**Assessment:** Reasonable default. No change needed.

---

### LOW-3: Empty Lines Dropped in Stream Parser

**File:** `pkg/runner/stream.go`
**Severity:** LOW

**Description:** Empty lines are silently discarded, which might affect output formatting.

**Assessment:** This is intentional behavior to clean up stream output. No change needed.

---

### LOW-4: Version File Search Uses Relative Paths

**File:** `pkg/orchestrator/orchestrator.go`
**Lines:** 1288-1303
**Severity:** LOW

**Description:** The `getVersion()` function searches for VERSION file using relative paths, which may not work correctly depending on working directory.

**Current Code:**
```go
func getVersion() string {
    candidates := []string{
        "VERSION",
        "../VERSION",
        "../../VERSION",
    }
    // ...
}
```

**Patch-Ready Diff:**
```diff
--- a/pkg/orchestrator/orchestrator.go
+++ b/pkg/orchestrator/orchestrator.go
@@ -1285,12 +1285,20 @@ func extractOverviewFromSummary(path string) string {
 // getVersion returns the rcodegen version from the VERSION file
 func getVersion() string {
-	// Try common locations
-	candidates := []string{
-		"VERSION",
-		"../VERSION",
-		"../../VERSION",
+	// Try to find VERSION relative to executable
+	exe, err := os.Executable()
+	if err == nil {
+		exeDir := filepath.Dir(exe)
+		candidates := []string{
+			filepath.Join(exeDir, "VERSION"),
+			filepath.Join(exeDir, "..", "VERSION"),
+		}
+		for _, path := range candidates {
+			if data, err := os.ReadFile(path); err == nil {
+				return strings.TrimSpace(string(data))
+			}
+		}
 	}

+	// Fallback to relative paths from working directory
+	candidates := []string{"VERSION", "../VERSION", "../../VERSION"}
 	for _, path := range candidates {
 		if data, err := os.ReadFile(path); err == nil {
 			return strings.TrimSpace(string(data))
```

---

### LOW-5: Inconsistent Use of os.UserHomeDir vs os.Getenv("HOME")

**File:** Multiple files
**Severity:** LOW

**Description:** Some code uses `os.UserHomeDir()` with fallback to `os.Getenv("HOME")`, while other code uses only `os.Getenv("HOME")`. This inconsistency could lead to different behavior on edge cases.

**Assessment:** The current approach is reasonable - `os.UserHomeDir()` is preferred with `os.Getenv("HOME")` as fallback. No change needed, but consistency should be maintained in future code.

---

### LOW-6: Unused Function getArticleNames

**File:** `pkg/orchestrator/orchestrator.go`
**Lines:** 853-868
**Severity:** LOW

**Description:** The function `getArticleNames` appears to be unused dead code.

**Patch-Ready Diff:**
```diff
--- a/pkg/orchestrator/orchestrator.go
+++ b/pkg/orchestrator/orchestrator.go
@@ -850,19 +850,6 @@ func findArticleByTool(articles []string, tool string) string {
 	return ""
 }

-func getArticleNames(paths []string) []string {
-	var names []string
-	for _, p := range paths {
-		name := filepath.Base(p)
-		name = strings.TrimSuffix(name, ".md")
-		// Shorten for table
-		if strings.Contains(name, "Codex") {
-			names = append(names, "Codex")
-		} else if strings.Contains(name, "Gemini") {
-			names = append(names, "Gemini")
-		} else {
-			names = append(names, name)
-		}
-	}
-	return names
-}
-
 func extractTitle(path string) string {
```

---

## Summary of Grades by Category

| Category | Score | Max | Notes |
|----------|-------|-----|-------|
| **Architecture & Design** | 22 | 25 | Clean separation of concerns, good interfaces |
| **Error Handling** | 12 | 20 | Multiple ignored errors, some race conditions |
| **Security** | 13 | 15 | One path traversal concern in script loading |
| **Code Quality** | 15 | 20 | Some dead code, minor inconsistencies |
| **Testing** | 6 | 10 | Limited test coverage evident from structure |
| **Documentation** | 10 | 10 | Good inline comments and README |
| **TOTAL** | **78** | **100** | |

---

## Recommended Priority

1. **Immediate (HIGH):** Fix race condition in Claude tool, add error checking for os.Remove in grades.go, remove CWD from script search path
2. **Soon (MEDIUM):** Add error handling for ignored errors, implement workspace cleanup
3. **Later (LOW):** Make timeouts configurable, remove dead code, standardize HOME resolution

---

## Conclusion

The rcodegen codebase is well-architected and functional. The main areas for improvement are:
1. Error handling discipline (several ignored errors)
2. Resource management (workspace cleanup)
3. Thread safety (one race condition identified)

The security concern in `findWrapper()` should be addressed promptly as it allows code execution from untrusted directories. Overall, this is a solid codebase with professional-quality structure that would benefit from more rigorous error handling.
