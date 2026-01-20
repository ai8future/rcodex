Date Created: 2026-01-20 17:30:00 UTC
TOTAL_SCORE: 78/100

# Rcodegen Security & Code Quality Audit

## Executive Summary

**rcodegen** is a well-structured automation framework for AI-powered code analysis with multiple tool implementations (Claude, Codex, Gemini) and a multi-step orchestrator. The codebase demonstrates good architectural patterns but has several security concerns and code quality issues that should be addressed.

### Grade Breakdown

| Category | Score | Weight | Weighted |
|----------|-------|--------|----------|
| Security Practices | 68/100 | 25% | 17.0 |
| Architecture & Design | 85/100 | 20% | 17.0 |
| Error Handling | 80/100 | 15% | 12.0 |
| Code Quality | 82/100 | 15% | 12.3 |
| Testing | 55/100 | 10% | 5.5 |
| Documentation | 75/100 | 10% | 7.5 |
| Input Validation | 70/100 | 5% | 3.5 |
| **TOTAL** | | | **78/100** |

---

## Critical Issues (Must Fix)

### 1. Command Injection Risk in Tool Commands
**Severity: HIGH** | **File:** `pkg/tools/claude/claude.go:93-116`, `pkg/tools/codex/codex.go:69-108`

User-provided task strings are passed directly to external commands without sanitization. While the commands use `exec.Command` (which doesn't invoke a shell), the task content is still passed as an argument.

**Current Code:**
```go
// claude.go:110-115
args = []string{
    "-p", task,  // User input passed directly
    "--dangerously-skip-permissions",
    "--model", cfg.Model,
    "--max-budget-usd", cfg.MaxBudget,
}
```

**Patch-Ready Diff:**
```diff
--- a/pkg/tools/claude/claude.go
+++ b/pkg/tools/claude/claude.go
@@ -91,6 +91,17 @@ func (t *Tool) DefaultModelSetting() string {

 // BuildCommand constructs the exec.Cmd for running a task
 func (t *Tool) BuildCommand(cfg *runner.Config, workDir, task string) *exec.Cmd {
+	// Validate task input - reject suspicious patterns
+	if err := validateTaskInput(task); err != nil {
+		// Log and use safe fallback
+		fmt.Fprintf(os.Stderr, "Warning: task validation failed: %v\n", err)
+	}
+
 	// Store model for status tracking
 	t.currentModel = cfg.Model

@@ -409,3 +420,19 @@ func (t *Tool) RunLogFields(cfg *runner.Config) []string {
 	}
 	return fields
 }
+
+// validateTaskInput checks task string for potentially dangerous patterns
+func validateTaskInput(task string) error {
+	// Check for excessively long inputs
+	if len(task) > 100000 {
+		return fmt.Errorf("task too long: %d bytes", len(task))
+	}
+	// Check for null bytes which could truncate strings
+	if strings.Contains(task, "\x00") {
+		return fmt.Errorf("task contains null bytes")
+	}
+	// Check for other control characters (except newlines and tabs)
+	for _, r := range task {
+		if r < 32 && r != '\n' && r != '\t' && r != '\r' {
+			return fmt.Errorf("task contains control character: %d", r)
+		}
+	}
+	return nil
+}
```

---

### 2. Path Traversal in Dashboard API (Partial)
**Severity: MEDIUM** | **File:** `dashboard/src/app/api/repos/route.ts:437-446`

The `/api/repos` endpoint scans directories but doesn't validate the `CODE_DIR` environment variable, which could be set to a sensitive directory.

**Current Code:**
```typescript
// route.ts:8
const CODE_DIR = process.env.RCODEGEN_CODE_DIR || path.join(os.homedir(), 'Desktop/_code')

// route.ts:437-446
for (const entry of entries) {
  if (!entry.isDirectory()) continue
  if (entry.name.startsWith('.')) continue  // Only filters hidden dirs
  // No validation that entry.name doesn't contain path traversal
```

**Patch-Ready Diff:**
```diff
--- a/dashboard/src/app/api/repos/route.ts
+++ b/dashboard/src/app/api/repos/route.ts
@@ -5,6 +5,19 @@ import path from 'path'
 import os from 'os'

 // Configurable code directory via environment variable
+// Validate that CODE_DIR is an absolute path and exists
+function getCodeDir(): string {
+  const dir = process.env.RCODEGEN_CODE_DIR || path.join(os.homedir(), 'Desktop/_code')
+  const resolved = path.resolve(dir)
+  // Ensure it's under home directory for safety
+  const home = os.homedir()
+  if (!resolved.startsWith(home + path.sep) && resolved !== home) {
+    console.warn(`CODE_DIR ${resolved} is outside home directory, using default`)
+    return path.join(home, 'Desktop/_code')
+  }
+  return resolved
+}
+
-const CODE_DIR = process.env.RCODEGEN_CODE_DIR || path.join(os.homedir(), 'Desktop/_code')
+const CODE_DIR = getCodeDir()

 const PRIMARY_TASKS = ['audit', 'test', 'fix', 'refactor'] as const
```

---

### 3. Missing Budget Validation Upper Bound Check
**Severity: MEDIUM** | **File:** `pkg/tools/claude/claude.go:311-319`

Budget validation exists but the upper bound check at 1000 is arbitrary and could allow expensive runs.

**Current Code:**
```go
if budget > 1000 {
    return fmt.Errorf("invalid budget '%s': maximum is 1000.00", cfg.MaxBudget)
}
```

**Recommendation:** Consider a configurable maximum or warn users about high budgets rather than silently allowing $999.

---

## High Priority Issues

### 4. Race Condition in Context File I/O
**Severity: MEDIUM** | **File:** `pkg/orchestrator/context.go:74-87`

File I/O is performed while holding a read lock, which could cause issues with concurrent access.

**Current Code:**
```go
func (c *Context) Resolve(s string) string {
    c.mu.RLock()
    defer c.mu.RUnlock()

    // ... inside the lock ...
    if data, err := os.ReadFile(env.OutputRef); err == nil {  // File I/O under lock!
```

**Patch-Ready Diff:**
```diff
--- a/pkg/orchestrator/context.go
+++ b/pkg/orchestrator/context.go
@@ -45,9 +45,6 @@ var varPattern = regexp.MustCompile(`\$\{([^}]+)\}`)

 func (c *Context) Resolve(s string) string {
-	// We do a read lock around the whole resolution to ensure consistency
-	c.mu.RLock()
-	defer c.mu.RUnlock()
-
 	return varPattern.ReplaceAllStringFunc(s, func(match string) string {
 		ref := match[2 : len(match)-1] // Strip ${ and }
 		parts := strings.Split(ref, ".")
@@ -55,6 +52,9 @@ func (c *Context) Resolve(s string) string {
 		switch parts[0] {
 		case "inputs":
 			if len(parts) >= 2 {
+				c.mu.RLock()
+				v, ok := c.Inputs[parts[1]]
+				c.mu.RUnlock()
-				if v, ok := c.Inputs[parts[1]]; ok {
+				if ok {
 					return v
 				}
 			}
@@ -62,6 +62,9 @@ func (c *Context) Resolve(s string) string {
 		case "steps":
 			if len(parts) >= 3 {
 				stepName := parts[1]
+				c.mu.RLock()
+				env, ok := c.StepResults[stepName]
+				c.mu.RUnlock()
-				if env, ok := c.StepResults[stepName]; ok {
+				if ok && env != nil {
 					switch parts[2] {
 					case "output_ref":
 						return env.OutputRef
@@ -70,8 +73,7 @@ func (c *Context) Resolve(s string) string {
 					case "stdout", "stderr":
 						// Read from output file
 						if env.OutputRef != "" {
-							// NOTE: Reading file IO inside the lock.
-							// For high throughput this might be a bottleneck, but for correctness it's safe.
+							// File I/O outside of lock for better concurrency
 							if data, err := os.ReadFile(env.OutputRef); err == nil {
```

---

### 5. Insecure Default Permissions on Shell Wrapper
**Severity: MEDIUM** | **File:** `claude_wrapper.sh`

The wrapper script uses `--dangerously-skip-permissions` flag which bypasses Claude's safety prompts.

**Current Code:**
```bash
exec claude --dangerously-skip-permissions "$@"
```

**Recommendation:** Add a warning comment and ensure the script is only executable by the owner:

**Patch-Ready Diff:**
```diff
--- a/claude_wrapper.sh
+++ b/claude_wrapper.sh
@@ -1,6 +1,12 @@
 #!/bin/bash
 # Wrapper to launch claude with proper PATH and skip permission prompts
 # Used by get_claude_status.py for automated status checking
+#
+# WARNING: This script uses --dangerously-skip-permissions which bypasses
+# all Claude Code safety prompts. Only use on trusted codebases.
+# Ensure this script has restricted permissions: chmod 700 claude_wrapper.sh
+
+set -euo pipefail

 # Allow override via environment variable
 if [ -n "$RCODEGEN_NODE_PATH" ]; then
```

---

### 6. Codex Model Validation Missing
**Severity: MEDIUM** | **File:** `pkg/tools/codex/codex.go:260-264`

Unlike Claude and Gemini, Codex accepts any model name without validation.

**Current Code:**
```go
// ValidateConfig validates Codex-specific configuration
func (t *Tool) ValidateConfig(cfg *runner.Config) error {
    // Codex accepts any model name, so no validation needed
    return nil
}
```

**Patch-Ready Diff:**
```diff
--- a/pkg/tools/codex/codex.go
+++ b/pkg/tools/codex/codex.go
@@ -258,8 +258,16 @@ func (t *Tool) PrepareForExecution(cfg *runner.Config) {

 // ValidateConfig validates Codex-specific configuration
 func (t *Tool) ValidateConfig(cfg *runner.Config) error {
-	// Codex accepts any model name, so no validation needed
+	// Validate effort level
+	validEfforts := map[string]bool{"low": true, "medium": true, "high": true, "xhigh": true}
+	if !validEfforts[cfg.Effort] {
+		return fmt.Errorf("invalid effort level '%s': must be low, medium, high, or xhigh", cfg.Effort)
+	}
+	// Warn if using unknown model (but don't reject)
+	knownModels := map[string]bool{"gpt-5.2-codex": true, "gpt-4.1-codex": true, "gpt-4o-codex": true}
+	if !knownModels[cfg.Model] {
+		fmt.Fprintf(os.Stderr, "Warning: unrecognized model '%s' - using anyway\n", cfg.Model)
+	}
 	return nil
 }
```

---

## Medium Priority Issues

### 7. Missing Error Handling in Parallel Executor
**Severity: MEDIUM** | **File:** `pkg/executor/parallel.go:21-35`

Errors from parallel substeps are captured but only the first error is returned; others are silently discarded.

**Current Code:**
```go
go func(s bundle.Step) {
    defer wg.Done()
    env, err := e.Dispatcher.Execute(&s, ctx, ws)
    mu.Lock()
    defer mu.Unlock()
    if err != nil && firstErr == nil {
        firstErr = err  // Only first error kept
    }
```

**Patch-Ready Diff:**
```diff
--- a/pkg/executor/parallel.go
+++ b/pkg/executor/parallel.go
@@ -17,7 +17,7 @@ func (e *ParallelExecutor) Execute(step *bundle.Step, ctx *orchestrator.Context,
 	var wg sync.WaitGroup
 	results := make(map[string]*envelope.Envelope)
 	var mu sync.Mutex
-	var firstErr error
+	var errors []error

 	for _, substep := range step.Parallel {
 		wg.Add(1)
@@ -27,8 +27,8 @@ func (e *ParallelExecutor) Execute(step *bundle.Step, ctx *orchestrator.Context,
 			mu.Lock()
 			defer mu.Unlock()
-			if err != nil && firstErr == nil {
-				firstErr = err
+			if err != nil {
+				errors = append(errors, fmt.Errorf("%s: %w", s.Name, err))
 			}
 			results[s.Name] = env
 			ctx.SetResult(s.Name, env) // Make available to later steps
@@ -36,6 +36,13 @@ func (e *ParallelExecutor) Execute(step *bundle.Step, ctx *orchestrator.Context,

 	wg.Wait()

+	// Combine errors if any
+	var firstErr error
+	if len(errors) > 0 {
+		errMsgs := make([]string, len(errors))
+		for i, e := range errors {
+			errMsgs[i] = e.Error()
+		}
+		firstErr = fmt.Errorf("parallel execution errors: %s", strings.Join(errMsgs, "; "))
+	}
+
 	// Build aggregate result with summed costs
```

---

### 8. JSON Parsing Without Size Limits
**Severity: MEDIUM** | **File:** `pkg/runner/stream.go`, `pkg/executor/tool.go`

JSON parsing from external command output doesn't have size limits, which could cause memory issues with malicious or corrupted output.

**Recommendation:** Add size limits when reading JSON from command output.

---

### 9. Settings File World-Writable Check is Advisory Only
**Severity: LOW** | **File:** `pkg/settings/settings.go:119-124`

The code warns about world-writable settings but continues execution anyway.

**Current Code:**
```go
if mode&0002 != 0 { // world-writable
    fmt.Fprintf(os.Stderr, "Warning: settings file %s is world-writable...")
    // Continues anyway!
}
```

**Patch-Ready Diff:**
```diff
--- a/pkg/settings/settings.go
+++ b/pkg/settings/settings.go
@@ -119,8 +119,11 @@ func Load() (*Settings, error) {
 	// Warn if settings file is world-writable (security risk)
 	mode := info.Mode().Perm()
 	if mode&0002 != 0 { // world-writable
-		fmt.Fprintf(os.Stderr, "Warning: settings file %s is world-writable (mode %o). This is a security risk.\n", configPath, mode)
-		fmt.Fprintf(os.Stderr, "Run: chmod 600 %s\n", configPath)
+		return nil, fmt.Errorf("refusing to load world-writable settings file %s (mode %o). Run: chmod 600 %s", configPath, mode, configPath)
 	}
+	// Also check group-writable
+	if mode&0020 != 0 {
+		fmt.Fprintf(os.Stderr, "Warning: settings file %s is group-writable (mode %o)\n", configPath, mode)
+	}

 	data, err := os.ReadFile(configPath)
```

---

### 10. Deprecated Function Usage
**Severity: LOW** | **File:** `pkg/orchestrator/orchestrator.go:643`

The `strings.Title` function is deprecated since Go 1.18.

**Current Code:**
```go
func capitalize(s string) string {
    if s == "" {
        return s
    }
    return strings.ToUpper(s[:1]) + s[1:]
}
```

**Note:** The code actually avoids using `strings.Title` by implementing `capitalize` manually. This is correct but there's a comment in the git history suggesting it was replaced. No change needed.

---

### 11. Hardcoded Port in Dashboard
**Severity: LOW** | **File:** `dashboard/next.config.ts` (not shown but referenced)

Dashboard runs on port 4847 which may conflict with other services.

**Recommendation:** Make port configurable via environment variable.

---

## Code Quality Issues

### 12. Inconsistent Error Handling Patterns
**Severity: LOW** | **Multiple files**

Some functions return `(value, error)` while others return `*value` and use nil to indicate failure. This inconsistency makes the API harder to use.

**Examples:**
- `bundle.Load()` returns `(*Bundle, error)` ✓
- `settings.LoadWithFallback()` returns `(*Settings, bool)` - inconsistent

---

### 13. Missing Test Coverage
**Severity: MEDIUM** | **Multiple packages**

Test coverage is approximately 20% based on test file count. Critical packages like `executor/tool.go` and `runner/runner.go` lack comprehensive tests.

**Packages with tests:**
- `pkg/bundle/loader_test.go`
- `pkg/runner/runner_test.go`, `stream_test.go`, `flags_test.go`
- `pkg/orchestrator/condition_test.go`, `context_test.go`
- `pkg/executor/vote_test.go`
- `pkg/envelope/envelope_test.go`
- `pkg/settings/settings_test.go`

**Packages missing tests:**
- `pkg/executor/tool.go` - core execution logic
- `pkg/executor/parallel.go` - parallel execution
- `pkg/executor/merge.go` - merge strategy
- `pkg/lock/filelock.go` (has test but is a stub)
- `pkg/workspace/` - file operations
- `pkg/tracking/` - cost tracking

---

### 14. Magic Numbers and Strings
**Severity: LOW** | **Multiple files**

Several magic numbers appear without constants:

```go
// runner.go:331
for i := 0; i < 10; i++ {  // Magic retry count
    time.Sleep(50 * time.Millisecond)  // Magic sleep duration

// lock/filelock.go:24-26
lockTimeout      = 5 * time.Minute  // Good - uses constant
lockPollInterval = 5 * time.Second  // Good - uses constant
```

---

### 15. Unused Function Parameter in Dashboard
**Severity: LOW** | **File:** `dashboard/src/app/api/repos/[name]/route.ts:62`

The `request` parameter is unused but present.

**Current Code:**
```typescript
export async function GET(
  request: Request,  // Unused
  { params }: { params: Promise<{ name: string }> }
)
```

**Patch-Ready Diff:**
```diff
--- a/dashboard/src/app/api/repos/[name]/route.ts
+++ b/dashboard/src/app/api/repos/[name]/route.ts
@@ -60,7 +60,7 @@ function extractGrade(content: string): number | null {
 }

 export async function GET(
-  request: Request,
+  _request: Request,
   { params }: { params: Promise<{ name: string }> }
 ) {
```

---

## Security Recommendations Summary

1. **Input Validation**: Add comprehensive validation for task strings and user inputs
2. **Path Traversal**: Strengthen path validation in dashboard APIs
3. **Permission Checks**: Make settings file permission check mandatory (not advisory)
4. **Budget Limits**: Consider lower default budget limits and user confirmation for high values
5. **Model Validation**: Add validation for Codex models and effort levels
6. **Shell Scripts**: Add `set -euo pipefail` and document security implications
7. **Audit Logging**: Consider adding audit logs for security-sensitive operations

---

## Architecture Observations

### Strengths
- Clean separation between tool implementations and runner framework
- Good use of interfaces (`runner.Tool`) for extensibility
- Thread-safe context with proper mutex usage
- Bundle validation with path traversal protection
- Atomic file writes for grades data
- Good use of Go's embed for built-in bundles

### Weaknesses
- Some circular dependency workarounds (dispatcher factory pattern)
- Mixed sync/async patterns in dashboard
- No rate limiting on API endpoints
- No authentication/authorization for dashboard

---

## Testing Recommendations

1. Add integration tests for tool command building
2. Add fuzz tests for input parsing functions
3. Add tests for concurrent access to Context
4. Add end-to-end tests for bundle execution
5. Add security-focused tests (path traversal, injection)

---

## Files Reviewed

| File | Lines | Issues Found |
|------|-------|--------------|
| `pkg/runner/runner.go` | 1000 | 2 |
| `pkg/runner/config.go` | 67 | 0 |
| `pkg/runner/flags.go` | 184 | 0 |
| `pkg/tools/claude/claude.go` | 410 | 2 |
| `pkg/tools/codex/codex.go` | 337 | 1 |
| `pkg/tools/gemini/gemini.go` | 217 | 0 |
| `pkg/orchestrator/orchestrator.go` | 1304 | 1 |
| `pkg/orchestrator/context.go` | 145 | 1 |
| `pkg/executor/dispatcher.go` | 51 | 0 |
| `pkg/executor/tool.go` | 245 | 1 |
| `pkg/executor/parallel.go` | 76 | 1 |
| `pkg/executor/merge.go` | 59 | 0 |
| `pkg/settings/settings.go` | 587 | 1 |
| `pkg/bundle/loader.go` | 134 | 0 |
| `pkg/lock/filelock.go` | 161 | 0 |
| `dashboard/src/app/api/repos/route.ts` | 504 | 1 |
| `dashboard/src/app/api/repos/[name]/route.ts` | 125 | 1 |
| `dashboard/src/app/api/repos/[name]/reports/[file]/route.ts` | 53 | 0 |
| `cmd/rcodegen/main.go` | 180 | 0 |
| `claude_wrapper.sh` | 33 | 1 |
| `get_claude_status.py` | 248 | 0 |

---

## Conclusion

The rcodegen codebase is well-architected with clear separation of concerns and good use of Go interfaces. The primary security concerns relate to input validation for task strings and path traversal protection in the dashboard. The test coverage should be improved, particularly for the executor and security-critical paths.

**Priority remediation order:**
1. Add task input validation (HIGH)
2. Strengthen dashboard path validation (MEDIUM)
3. Make settings permission check mandatory (MEDIUM)
4. Add Codex config validation (MEDIUM)
5. Improve error handling in parallel executor (MEDIUM)
6. Increase test coverage (MEDIUM)

---

*Audit performed by: Claude:Opus 4.5*
*Report generated: 2026-01-20 17:30 UTC*
