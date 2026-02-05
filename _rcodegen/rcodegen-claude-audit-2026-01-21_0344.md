# rcodegen Security and Code Audit Report

**Date Created:** 2026-01-21 03:44 UTC
**TOTAL_SCORE:** 78/100

**Auditor:** Claude Opus 4.5
**Version Audited:** 1.8.14

---

## Executive Summary

rcodegen is a well-structured, production-grade CLI automation framework for AI-powered code analysis. The codebase demonstrates solid engineering practices with good separation of concerns, defensive coding, and attention to security. However, there are several areas requiring attention:

- **High Priority:** 4 issues (2 security, 2 code quality)
- **Medium Priority:** 8 issues (3 security, 5 code quality)
- **Low Priority:** 6 issues (style and minor improvements)

---

## Table of Contents

1. [Security Issues](#security-issues)
2. [Code Quality Issues](#code-quality-issues)
3. [Bug Risks](#bug-risks)
4. [Error Handling Issues](#error-handling-issues)
5. [Performance Concerns](#performance-concerns)
6. [Testing Gaps](#testing-gaps)
7. [Patch-Ready Diffs](#patch-ready-diffs)

---

## Security Issues

### HIGH-SEC-1: Command Injection Risk in Shell Wrappers

**File:** `claude_wrapper.sh:11-18`, `codex_wrapper.sh:8-14`

**Severity:** HIGH

**Description:** The shell wrappers use unquoted `$latest` variable from `ls` output, which could cause issues with unusual directory names. Additionally, `ls -v` piped to `tail` can be manipulated if an attacker creates directories with special characters in `~/.nvm/versions/node`.

**Current Code:**
```bash
if [ -d "$HOME/.nvm/versions/node" ]; then
    # nvm - use latest installed version
    latest=$(ls -v "$HOME/.nvm/versions/node" 2>/dev/null | tail -1)
    if [ -n "$latest" ]; then
        export PATH="$HOME/.nvm/versions/node/$latest/bin:$PATH"
    fi
fi
```

**Risk:** While exploitation requires write access to user's nvm directory, this could be leveraged in multi-user environments or if the user clones a malicious repo that creates symlinks.

---

### HIGH-SEC-2: TOCTOU Race in Settings File Permission Check

**File:** `pkg/settings/settings.go:111-124`

**Severity:** HIGH

**Description:** The code checks file permissions then reads the file, creating a Time-Of-Check-Time-Of-Use (TOCTOU) race condition. An attacker could swap the file between the stat() and ReadFile() calls.

**Current Code:**
```go
info, err := os.Stat(configPath)
if err != nil {
    // ...
}
// Warn if settings file is world-writable (security risk)
mode := info.Mode().Perm()
if mode&0002 != 0 { // world-writable
    fmt.Fprintf(os.Stderr, "Warning: settings file %s is world-writable...")
}

data, err := os.ReadFile(configPath)  // Race window here
```

**Risk:** On shared systems, another process could replace the file between permission check and file read.

---

### MED-SEC-3: Debug Files Written to /tmp in Python Scripts

**File:** `get_codex_status.py:176-191`, `get_claude_status.py:206-217`

**Severity:** MEDIUM

**Description:** When DEBUG_MODE is enabled, scripts write debug output to temp files. While `tempfile.mkstemp` is used (good), if another user can predict the filename pattern, they could pre-create symlinks to arbitrary files.

**Mitigation Already Present:** Uses `tempfile.mkstemp()` with restricted permissions.

**Remaining Risk:** The temp file path is returned in JSON output which could leak the path to other processes.

---

### MED-SEC-4: Missing Input Sanitization in Dashboard API

**File:** `dashboard/src/app/api/repos/route.ts:288-293`

**Severity:** MEDIUM

**Description:** The `scanRepo` function constructs paths using `path.join()` but doesn't validate that the constructed path is still within the expected directory. A malicious directory name like `../../../etc` in the CODE_DIR could cause issues.

**Current Code:**
```typescript
const rcodgenDir = path.join(repoPath, '_rcodegen')

if (!fsSync.existsSync(rcodgenDir)) {
    return null
}
```

**Risk:** Path traversal if CODE_DIR environment variable is attacker-controlled or contains unexpected values.

---

### MED-SEC-5: Potential Command Line Argument Exposure

**File:** `pkg/runner/runner.go:893-907`

**Severity:** MEDIUM

**Description:** The runlog file writes full command arguments including potentially sensitive data passed via `-x` variable flags. This file is written with 0644 permissions, making it world-readable.

**Current Code:**
```go
lines = append(lines, fmt.Sprintf("Command:   %s %s", r.Tool.Name(), cfg.OriginalCmd))
// ...
if err := os.WriteFile(filepath, []byte(content), 0644); err != nil {
```

**Risk:** If users pass secrets via `-x secret=value`, they'll be written to a world-readable file.

---

## Code Quality Issues

### HIGH-CQ-1: Mutex Not Properly Protecting File Operations in Grades

**File:** `pkg/runner/grades.go:194-222`

**Severity:** HIGH

**Description:** `AppendGrade` uses a process-local mutex (`gradesFileMutex`) to protect concurrent writes, but this doesn't protect against multiple processes writing to the same file simultaneously.

**Current Code:**
```go
func AppendGrade(reportDir, reportFile, tool, task string, grade float64, date time.Time) error {
    // Lock to prevent race conditions
    gradesFileMutex.Lock()
    defer gradesFileMutex.Unlock()

    grades, err := LoadGrades(reportDir)  // Another process may modify here
    // ...
    return SaveGrades(reportDir, grades)
}
```

**Risk:** When multiple rclaude/rcodex/rgemini processes run simultaneously, they can overwrite each other's grade entries.

---

### HIGH-CQ-2: Unbounded String Allocation in Stream Parser

**File:** `pkg/runner/stream.go:307-316`

**Severity:** HIGH

**Description:** The stream parser allocates a 1MB buffer for line scanning, but doesn't limit memory for parsed JSON objects. A malicious or malfunctioning AI tool could send extremely large JSON objects.

**Current Code:**
```go
buf := make([]byte, 0, 64*1024)
scanner.Buffer(buf, 1024*1024) // 1MB max line size

for scanner.Scan() {
    p.ProcessLine(scanner.Text())  // No size check before JSON parsing
}
```

**Risk:** Memory exhaustion if receiving very large JSON responses.

---

### MED-CQ-3: Incorrect Token Ratio Estimation for Codex

**File:** `pkg/executor/tool.go:173-179`

**Severity:** MEDIUM

**Description:** The code estimates Codex token split as 70% input / 30% output, which is hardcoded and may not reflect actual usage patterns. This affects cost calculations.

**Current Code:**
```go
// Codex doesn't break down input/output, estimate 70% input, 30% output
usage.InputTokens = tokens * 7 / 10
usage.OutputTokens = tokens * 3 / 10
// Estimate cost: GPT-5.2 Codex pricing
usage.CostUSD = float64(usage.InputTokens)*0.00001 + float64(usage.OutputTokens)*0.00003
```

**Issue:** Hardcoded pricing and token ratios will become inaccurate as models change.

---

### MED-CQ-4: Potential Panic in Date Parsing

**File:** `dashboard/src/app/api/repos/route.ts:222-251`

**Severity:** MEDIUM

**Description:** The `parseDate` function uses array destructuring with fixed indices that could cause runtime errors if regex matches don't have expected groups.

**Current Code:**
```typescript
const compactMatch = dateStr.match(/^(\d{4})(\d{2})(\d{2})-(\d{2})(\d{2})(\d{2})?$/)
if (compactMatch) {
    const [, year, month, day, hours, minutes, seconds = '00'] = compactMatch
    return new Date(`${year}-${month}-${day}T${hours}:${minutes}:${seconds}Z`)
}
```

**Risk:** While this is TypeScript and has some safety, malformed date strings could still cause unexpected behavior.

---

### MED-CQ-5: Orphaned Temp File on Error

**File:** `pkg/runner/grades.go:167-187`

**Severity:** MEDIUM

**Description:** `SaveGrades` creates a temp file and attempts to rename it atomically. However, if `json.MarshalIndent` fails after the temp file path is constructed but before writing, there's no cleanup.

**Current Code:**
```go
func SaveGrades(reportDir string, grades *GradesFile) error {
    gradesPath := filepath.Join(reportDir, ".grades.json")
    tempPath := gradesPath + ".tmp"

    data, err := json.MarshalIndent(grades, "", "  ")
    if err != nil {
        return fmt.Errorf("failed to marshal grades: %w", err)
        // tempPath not created yet, but pattern could lead to orphans
    }
    // ...
}
```

**Note:** Current implementation is actually safe since temp file isn't created until WriteFile. However, pattern could be risky if code is modified.

---

### MED-CQ-6: Missing Context Cancellation in Orchestrator

**File:** `pkg/orchestrator/orchestrator.go:200-305`

**Severity:** MEDIUM

**Description:** The orchestrator's step execution loop doesn't support context cancellation. If a user wants to cancel a long-running multi-step bundle, there's no clean way to stop intermediate steps.

**Issue:** No graceful shutdown when running complex bundles.

---

### MED-CQ-7: Duplicate JSON Parsing in extractCostInfo

**File:** `pkg/executor/tool.go:137-167`

**Severity:** MEDIUM

**Description:** The `extractCostInfo` function for Claude parses JSON lines from the end, but parses each line fully even if it doesn't contain the expected type. This is inefficient.

**Current Code:**
```go
for i := len(lines) - 1; i >= 0; i-- {
    line := strings.TrimSpace(lines[i])
    if line == "" {
        continue
    }
    var obj map[string]interface{}
    if err := json.Unmarshal([]byte(line), &obj); err != nil {
        continue
    }
    if objType, _ := obj["type"].(string); objType == "result" {
        // ...
    }
}
```

**Improvement:** Could do a quick string check for `"type":"result"` before full JSON parsing.

---

## Bug Risks

### BUG-1: Integer Overflow in Credit Percentage Calculation

**File:** `pkg/tools/claude/claude.go:192-194`

**Severity:** LOW

**Description:** Credit percentage subtraction could produce negative values if credits reset during a run. The code handles this but assigns 0 which may be misleading.

**Current Code:**
```go
sessionCost = *statusBefore.SessionLeft - *statusAfter.SessionLeft
if sessionCost < 0 {
    sessionCost = 0 // Reset happened during run
}
```

**Issue:** Users won't know a reset occurred; they'll just see 0% cost.

---

### BUG-2: Glob Pattern Escaping May Miss Some Characters

**File:** `pkg/runner/grades.go:224-235`

**Severity:** LOW

**Description:** The `escapeGlobPattern` function escapes common glob characters but misses `{` and `}` which are valid in some glob implementations (e.g., brace expansion).

**Current Code:**
```go
func escapeGlobPattern(s string) string {
    replacer := strings.NewReplacer(
        "*", "\\*",
        "?", "\\?",
        "[", "\\[",
        "]", "\\]",
        "\\", "\\\\",
    )
    return replacer.Replace(s)
}
```

**Issue:** Tool or task names containing `{` or `}` could cause unexpected glob matches.

---

### BUG-3: Race Condition in Workspace Job ID Generation

**File:** `pkg/workspace/workspace.go:23-30`

**Severity:** LOW

**Description:** `GenerateJobID` uses timestamp plus random bytes. In extremely rare cases (same second, same random bytes), two workspaces could get the same ID.

**Current Code:**
```go
func GenerateJobID() string {
    now := time.Now()
    b := make([]byte, 4)
    if _, err := rand.Read(b); err != nil {
        // Fallback: use nanoseconds if crypto/rand fails
        return fmt.Sprintf("%s-%08x", now.Format("20060102-150405"), now.UnixNano()&0xFFFFFFFF)
    }
    return fmt.Sprintf("%s-%s", now.Format("20060102-150405"), hex.EncodeToString(b))
}
```

**Risk:** Extremely low probability collision (1 in 4 billion per second), but worth noting.

---

## Error Handling Issues

### ERR-1: Silent Failure in Dashboard Grade Migration

**File:** `dashboard/src/app/api/repos/route.ts:139-188`

**Severity:** MEDIUM

**Description:** `migrateRepoGrades` silently continues on read errors, which could cause incomplete migrations without user awareness.

**Current Code:**
```typescript
let content: string
try {
    content = fsSync.readFileSync(filePath, 'utf-8')
} catch {
    continue  // Silent skip
}
```

---

### ERR-2: Error Swallowed in Lock Release

**File:** `pkg/lock/filelock.go:140-146`

**Severity:** LOW

**Description:** If both unlock and close fail, only the unlock error is returned; the close error is lost.

**Current Code:**
```go
func (l *FileLock) Release() error {
    unlockErr := syscall.Flock(int(l.file.Fd()), syscall.LOCK_UN)
    closeErr := l.file.Close()
    if unlockErr != nil {
        return fmt.Errorf("failed to unlock: %w", unlockErr)
    }
    return closeErr  // closeErr only returned if unlockErr is nil
}
```

---

## Performance Concerns

### PERF-1: Repeated File Reads in Report Scanning

**File:** `dashboard/src/app/api/repos/route.ts:313-365`

**Severity:** LOW

**Description:** For each report file, the code may read the file twice - once to check grades cache, once to check for "Date Updated". Files should be read once and cached.

---

### PERF-2: Synchronous File Operations in Dashboard API

**File:** `dashboard/src/app/api/repos/route.ts`

**Severity:** LOW

**Description:** The API route uses `fsSync` synchronous operations which block the event loop. Should use async fs operations throughout.

---

## Testing Gaps

1. **No tests for edge cases in date parsing** - Various date formats could fail
2. **No integration tests for orchestrator** - Complex bundle execution untested
3. **Missing error path tests** - Many error conditions lack test coverage
4. **No tests for concurrent grade writes** - Race conditions untested

---

## Patch-Ready Diffs

### Patch 1: Fix TOCTOU Race in Settings File Read

```diff
--- a/pkg/settings/settings.go
+++ b/pkg/settings/settings.go
@@ -105,23 +105,21 @@ func expandTilde(path string) string {
 // Load reads settings from ~/.rcodegen/settings.json
 // Returns nil and an error if the file doesn't exist or is invalid
 func Load() (*Settings, error) {
 	configPath := GetConfigPath()

-	// Check file permissions for security
-	info, err := os.Stat(configPath)
+	// Read file first, then check permissions on the same data
+	data, err := os.ReadFile(configPath)
 	if err != nil {
 		if os.IsNotExist(err) {
 			return nil, fmt.Errorf("settings file not found: %s", configPath)
 		}
-		return nil, fmt.Errorf("failed to stat settings file: %w", err)
+		return nil, fmt.Errorf("failed to read %s: %w", configPath, err)
 	}

-	// Warn if settings file is world-writable (security risk)
-	mode := info.Mode().Perm()
-	if mode&0002 != 0 { // world-writable
-		fmt.Fprintf(os.Stderr, "Warning: settings file %s is world-writable (mode %o). This is a security risk.\n", configPath, mode)
-		fmt.Fprintf(os.Stderr, "Run: chmod 600 %s\n", configPath)
-	}
-
-	data, err := os.ReadFile(configPath)
-	if err != nil {
-		return nil, fmt.Errorf("failed to read %s: %w", configPath, err)
+	// Check file permissions for security (after read to avoid TOCTOU)
+	if info, err := os.Stat(configPath); err == nil {
+		mode := info.Mode().Perm()
+		if mode&0002 != 0 { // world-writable
+			fmt.Fprintf(os.Stderr, "Warning: settings file %s is world-writable (mode %o). This is a security risk.\n", configPath, mode)
+			fmt.Fprintf(os.Stderr, "Run: chmod 600 %s\n", configPath)
+		}
 	}

 	var settings Settings
```

### Patch 2: Use File Locking for Grades File

```diff
--- a/pkg/runner/grades.go
+++ b/pkg/runner/grades.go
@@ -8,6 +8,7 @@ import (
 	"regexp"
 	"strings"
 	"sync"
+	"syscall"
 	"time"
 )

@@ -165,12 +166,28 @@ func LoadGrades(reportDir string) (*GradesFile, error) {
 // SaveGrades writes the .grades.json file to a report directory atomically
 func SaveGrades(reportDir string, grades *GradesFile) error {
 	gradesPath := filepath.Join(reportDir, ".grades.json")
-	tempPath := gradesPath + ".tmp"
+
+	// Open file with exclusive lock for cross-process safety
+	f, err := os.OpenFile(gradesPath, os.O_RDWR|os.O_CREATE, 0644)
+	if err != nil {
+		return fmt.Errorf("failed to open grades file: %w", err)
+	}
+	defer f.Close()
+
+	// Acquire exclusive lock
+	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX); err != nil {
+		return fmt.Errorf("failed to lock grades file: %w", err)
+	}
+	defer syscall.Flock(int(f.Fd()), syscall.LOCK_UN)

 	data, err := json.MarshalIndent(grades, "", "  ")
 	if err != nil {
 		return fmt.Errorf("failed to marshal grades: %w", err)
 	}
+
+	tempPath := gradesPath + ".tmp"

 	// Write to temp file first
 	if err := os.WriteFile(tempPath, data, 0644); err != nil {
```

### Patch 3: Fix Runlog Permissions to Prevent Secret Exposure

```diff
--- a/pkg/runner/runner.go
+++ b/pkg/runner/runner.go
@@ -929,8 +929,11 @@ func (r *Runner) writeRunLog(cfg *Config, workDir string, startTime, endTime tim

 	content := strings.Join(lines, "\n") + "\n"

-	// Write file
-	if err := os.WriteFile(filepath, []byte(content), 0644); err != nil {
+	// Write file with restricted permissions (owner read/write only)
+	// This prevents exposure of any sensitive data passed via -x flags
+	if err := os.WriteFile(filepath, []byte(content), 0600); err != nil {
 		fmt.Fprintf(os.Stderr, "%sWarning:%s Could not write runlog: %v\n", Yellow, Reset, err)
 		return
 	}
```

### Patch 4: Add JSON Size Limit to Stream Parser

```diff
--- a/pkg/runner/stream.go
+++ b/pkg/runner/stream.go
@@ -11,6 +11,9 @@ import (
 	"strings"
 )

+// maxJSONLineSize limits the maximum size of a single JSON line to prevent memory exhaustion
+const maxJSONLineSize = 10 * 1024 * 1024 // 10MB
+
 // StreamEvent represents a parsed stream-json event from Claude or Gemini
 type StreamEvent struct {
 	Type         string          `json:"type"`
@@ -73,6 +76,12 @@ func NewStreamParser(w io.Writer) *StreamParser {
 // ProcessLine processes a single JSON line from stream output
 func (p *StreamParser) ProcessLine(line string) {
 	line = strings.TrimSpace(line)
+
+	// Protect against extremely large JSON lines
+	if len(line) > maxJSONLineSize {
+		fmt.Fprintf(p.writer, "%sWarning: Skipping oversized JSON line (%d bytes)%s\n", Yellow, len(line), Reset)
+		return
+	}
 	if line == "" {
 		return
 	}
```

### Patch 5: Add Brace Escaping to Glob Pattern Function

```diff
--- a/pkg/runner/grades.go
+++ b/pkg/runner/grades.go
@@ -222,11 +222,13 @@ func AppendGrade(reportDir, reportFile, tool, task string, grade float64, date t

 // escapeGlobPattern escapes special glob characters in a string
 func escapeGlobPattern(s string) string {
-	// Escape glob metacharacters: * ? [ ] \
+	// Escape glob metacharacters: * ? [ ] { } \
 	replacer := strings.NewReplacer(
 		"*", "\\*",
 		"?", "\\?",
 		"[", "\\[",
 		"]", "\\]",
+		"{", "\\{",
+		"}", "\\}",
 		"\\", "\\\\",
 	)
 	return replacer.Replace(s)
```

### Patch 6: Safe Shell Variable Quoting in Wrappers

```diff
--- a/claude_wrapper.sh
+++ b/claude_wrapper.sh
@@ -10,8 +10,12 @@ else

     if [ -d "$HOME/.nvm/versions/node" ]; then
         # nvm - use latest installed version
-        latest=$(ls -v "$HOME/.nvm/versions/node" 2>/dev/null | tail -1)
-        if [ -n "$latest" ]; then
+        # Use find to safely get the latest version without shell expansion issues
+        latest=$(find "$HOME/.nvm/versions/node" -maxdepth 1 -type d -name 'v*' 2>/dev/null | sort -V | tail -1)
+        if [ -n "$latest" ] && [ -d "$latest/bin" ]; then
+            export PATH="$latest/bin:$PATH"
+        elif latest=$(ls -v "$HOME/.nvm/versions/node" 2>/dev/null | tail -1); then
+            # Fallback for systems without find -V support
             export PATH="$HOME/.nvm/versions/node/$latest/bin:$PATH"
         fi
     fi
```

---

## Scoring Breakdown

| Category | Weight | Score | Notes |
|----------|--------|-------|-------|
| Security | 25% | 72/100 | TOCTOU race, shell risks, permission issues |
| Code Quality | 25% | 80/100 | Good structure, some mutex and memory issues |
| Error Handling | 15% | 75/100 | Many silent failures, some errors swallowed |
| Testing | 15% | 70/100 | Good test coverage exists but gaps in edge cases |
| Performance | 10% | 85/100 | Generally efficient, some unnecessary re-reads |
| Documentation | 10% | 90/100 | Good inline docs, clear package descriptions |

**TOTAL_SCORE: 78/100**

---

## Recommendations Summary

### Immediate Actions (Critical)
1. Fix TOCTOU race in settings file loading
2. Add file-level locking to grades.json operations
3. Restrict runlog file permissions to 0600
4. Add JSON line size limits in stream parser

### Short-term Actions (Important)
1. Improve shell wrapper safety with quoted variables
2. Add context cancellation support to orchestrator
3. Improve error logging in dashboard API
4. Add missing test coverage for edge cases

### Long-term Actions (Nice to Have)
1. Make token ratio and pricing configurable
2. Add async file operations in dashboard
3. Consider using a proper database for grades instead of JSON file
4. Add integration test suite for multi-tool scenarios

---

*Report generated by Claude Opus 4.5 via rcodegen audit framework*
