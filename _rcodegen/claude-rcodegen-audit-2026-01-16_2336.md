Date Created: 2026-01-16 23:36:00 UTC
TOTAL_SCORE: 82/100

# Comprehensive Audit Report: rcodegen

## Executive Summary

rcodegen is a well-architected multi-tool orchestration framework for AI coding assistants (Claude, Codex, Gemini). The codebase demonstrates solid Go practices with minimal dependencies (stdlib only), good separation of concerns, and thoughtful security measures in many areas. However, there are several issues that should be addressed, particularly around security, error handling, and code quality.

---

## Scoring Breakdown

| Category | Score | Weight | Weighted |
|----------|-------|--------|----------|
| Architecture & Design | 90/100 | 25% | 22.5 |
| Security Practices | 72/100 | 20% | 14.4 |
| Error Handling | 78/100 | 15% | 11.7 |
| Testing | 75/100 | 15% | 11.25 |
| Idioms & Style | 88/100 | 15% | 13.2 |
| Documentation | 90/100 | 10% | 9.0 |
| **TOTAL** | | | **82.05** |

---

## Critical Issues

### 1. [SECURITY] Codex PTY Wrapper Searches CWD - Path Injection Risk

**File:** `pkg/tools/codex/codex.go:121-128`
**Severity:** HIGH

The `findWrapper()` function searches the current working directory for `codex_pty_wrapper.py`, which could be attacker-controlled. This is inconsistent with the Claude tracking script which explicitly avoids CWD.

```go
// Current code (VULNERABLE)
// 2. Check current working directory
if cwd, err := os.Getwd(); err == nil {
    path := filepath.Join(cwd, wrapperName)
    if _, err := os.Stat(path); err == nil {
        return path
    }
}
```

**Impact:** An attacker who can place a malicious `codex_pty_wrapper.py` in a project directory could execute arbitrary Python code when the user runs `rcodex` with session resume.

**Patch-Ready Diff:**
```diff
--- a/pkg/tools/codex/codex.go
+++ b/pkg/tools/codex/codex.go
@@ -108,22 +108,23 @@ func (t *Tool) BuildCommand(cfg *runner.Config, workDir, task string) *exec.Cmd
 }

 func (t *Tool) findWrapper() string {
 	const wrapperName = "codex_pty_wrapper.py"

+	// Security: Only search trusted locations, NOT the current working directory
+	// (which could be attacker-controlled)
+
 	// 1. Check same directory as executable
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
-
-	// 3. Check ~/.rcodegen/
+	// 2. Check ~/.rcodegen/scripts/ (user scripts directory)
 	home := os.Getenv("HOME")
-	path := filepath.Join(home, ".rcodegen", wrapperName)
+	path := filepath.Join(home, ".rcodegen", "scripts", wrapperName)
 	if _, err := os.Stat(path); err == nil {
 		return path
 	}

-	// Fallback (mostly for dev environment if CWD check failed)
-	return wrapperName
+	// 3. Fallback to ~/.rcodegen/ root (legacy location)
+	path = filepath.Join(home, ".rcodegen", wrapperName)
+	if _, err := os.Stat(path); err == nil {
+		return path
+	}
+
+	// Return error indicator - caller should handle missing wrapper
+	return ""
 }
```

---

### 2. [SECURITY] Dangerous CLI Flags Used Without User Consent Warning

**Files:**
- `pkg/tools/claude/claude.go:104-115`
- `pkg/tools/codex/codex.go:78-92`
- `pkg/tools/gemini/gemini.go:77-83`

**Severity:** MEDIUM

All three tools automatically add dangerous flags that bypass safety mechanisms:
- Claude: `--dangerously-skip-permissions`
- Codex: `--dangerously-bypass-approvals-and-sandbox`
- Gemini: `--yolo`

While there's a security warning in the help text, there's no runtime warning or confirmation when the tool actually runs.

**Recommendation:** Add a visible warning banner at startup (not just in help).

**Patch-Ready Diff:**
```diff
--- a/pkg/runner/output.go
+++ b/pkg/runner/output.go
@@ -42,6 +42,10 @@ func PrintStartupBanner(tool Tool, cfg *Config) {
 	fmt.Printf("  %s%sDirectory:%s    %s\n", Bold, Green, Reset, workDir)
 	tool.PrintToolSpecificBannerFields(cfg)
 	fmt.Printf("%s%s╚════════════════════════════════════════════════════════════════╝%s\n\n", Bold, Cyan, Reset)
+
+	// Security warning - always shown
+	fmt.Printf("  %s⚠ WARNING:%s Running with safety checks disabled.%s\n", Yellow, Reset, Dim)
+	fmt.Printf("  %sOnly use on trusted codebases.%s\n\n", Dim, Reset)
 }
```

---

### 3. [SECURITY] File Permission Issue - Config Created with 0755 Directory

**File:** `pkg/settings/settings.go:518`

**Severity:** LOW

The config directory is created with mode 0755, which is overly permissive. While the settings.json file is correctly created with 0600, the directory should also be more restrictive.

```go
// Current code
if err := os.MkdirAll(configDir, 0755); err != nil {
```

**Patch-Ready Diff:**
```diff
--- a/pkg/settings/settings.go
+++ b/pkg/settings/settings.go
@@ -515,7 +515,8 @@ func RunInteractiveSetup() (*Settings, bool) {
 	}

 	// Create config directory
 	configDir := GetConfigDir()
-	if err := os.MkdirAll(configDir, 0755); err != nil {
+	// Use restrictive permissions - config may contain sensitive defaults
+	if err := os.MkdirAll(configDir, 0700); err != nil {
 		fmt.Fprintf(os.Stderr, "%sError:%s creating config directory: %v\n", yellow, reset, err)
 		return nil, false
 	}
```

---

### 4. [BUG] Error Handling Ignores JSON Encoding Errors

**File:** `cmd/rcodegen/main.go:114`

**Severity:** MEDIUM

The JSON encoder error is silently ignored when outputting results:

```go
if *jsonOutput {
    json.NewEncoder(os.Stdout).Encode(env)  // Error ignored!
}
```

**Patch-Ready Diff:**
```diff
--- a/cmd/rcodegen/main.go
+++ b/cmd/rcodegen/main.go
@@ -111,7 +111,10 @@ func runBundle() {
 	env, err := orch.Run(b, inputs)

 	if *jsonOutput {
-		json.NewEncoder(os.Stdout).Encode(env)
+		if encErr := json.NewEncoder(os.Stdout).Encode(env); encErr != nil {
+			fmt.Fprintf(os.Stderr, "Error encoding JSON output: %v\n", encErr)
+			os.Exit(1)
+		}
 	}

 	if err != nil || env.Status != "success" {
```

---

### 5. [BUG] File Write Error Ignored in Report Generation

**File:** `pkg/orchestrator/orchestrator.go:356`

**Severity:** MEDIUM

The result of `os.WriteFile` for copying bundle data is ignored:

```go
if bundleData, err := os.ReadFile(b.SourcePath); err == nil {
    os.WriteFile(bundleDest, bundleData, 0644)  // Error ignored!
}
```

**Patch-Ready Diff:**
```diff
--- a/pkg/orchestrator/orchestrator.go
+++ b/pkg/orchestrator/orchestrator.go
@@ -353,7 +353,9 @@ func (o *Orchestrator) Run(b *bundle.Bundle, inputs map[string]string) (*envelop
 			// Copy bundle to output directory
 			if b.SourcePath != "" {
 				bundleDest := filepath.Join(projectDir, "bundle-used.json")
 				if bundleData, err := os.ReadFile(b.SourcePath); err == nil {
-					os.WriteFile(bundleDest, bundleData, 0644)
+					if writeErr := os.WriteFile(bundleDest, bundleData, 0644); writeErr != nil {
+						fmt.Fprintf(os.Stderr, "Warning: could not write bundle copy: %v\n", writeErr)
+					}
 				}
 			}
```

---

### 6. [BUG] Race Condition in Context File IO

**File:** `pkg/orchestrator/context.go:75-87`

**Severity:** LOW

File I/O is performed inside a read-locked section. While the comment acknowledges this, reading files under a mutex is poor practice and could cause performance issues:

```go
c.mu.RLock()
defer c.mu.RUnlock()
// ...
if data, err := os.ReadFile(env.OutputRef); err == nil {  // File IO under lock!
```

**Recommendation:** Copy the necessary data while holding the lock, then perform file I/O outside the lock.

**Patch-Ready Diff:**
```diff
--- a/pkg/orchestrator/context.go
+++ b/pkg/orchestrator/context.go
@@ -44,9 +44,20 @@ func (c *Context) SetToolSession(toolName, sessionID string) {
 var varPattern = regexp.MustCompile(`\$\{([^}]+)\}`)

 func (c *Context) Resolve(s string) string {
-	// We do a read lock around the whole resolution to ensure consistency
-	c.mu.RLock()
-	defer c.mu.RUnlock()
+	// First pass: collect data we need under lock
+	c.mu.RLock()
+	inputsCopy := make(map[string]string)
+	for k, v := range c.Inputs {
+		inputsCopy[k] = v
+	}
+	stepResultsCopy := make(map[string]*envelope.Envelope)
+	for k, v := range c.StepResults {
+		stepResultsCopy[k] = v
+	}
+	c.mu.RUnlock()
+
+	// Second pass: do resolution without holding lock (file IO is safe now)

 	return varPattern.ReplaceAllStringFunc(s, func(match string) string {
 		ref := match[2 : len(match)-1] // Strip ${ and }
@@ -54,21 +65,20 @@ func (c *Context) Resolve(s string) string {

 		switch parts[0] {
 		case "inputs":
 			if len(parts) >= 2 {
-				if v, ok := c.Inputs[parts[1]]; ok {
+				if v, ok := inputsCopy[parts[1]]; ok {
 					return v
 				}
 			}
 		case "steps":
 			if len(parts) >= 3 {
 				stepName := parts[1]
-				if env, ok := c.StepResults[stepName]; ok {
+				if env, ok := stepResultsCopy[stepName]; ok {
 					switch parts[2] {
 					case "output_ref":
 						return env.OutputRef
```

---

### 7. [CODE QUALITY] Inconsistent Error Handling Patterns

**Multiple files**

**Severity:** LOW

The codebase uses inconsistent patterns for error handling:
- Some functions return `(nil, nil)` for non-error early exits
- Some use sentinel errors
- Some ignore errors silently

**Examples:**
- `pkg/runner/runner.go:91-93`: Returns `(nil, nil)` for help displayed
- `pkg/bundle/loader.go:111`: Ignores `builtinBundles.ReadDir` error

**Recommendation:** Adopt consistent error handling patterns. Consider using sentinel errors or a Result type pattern.

---

### 8. [CODE QUALITY] Magic Numbers and Hardcoded Values

**Multiple files**

**Severity:** LOW

Several magic numbers appear throughout the code:

```go
// pkg/lock/filelock.go
lockTimeout      = 5 * time.Minute
lockPollInterval = 5 * time.Second
maxIdentifierLen = 100

// pkg/bundle/loader.go
if len(name) > 100 {  // Same magic number

// pkg/settings/settings.go
if codeDir[0] >= '1' && codeDir[0] <= '9'  // Magic character range
```

**Recommendation:** Define these as named constants in a central location or at package level with documentation.

---

### 9. [CODE QUALITY] Duplicate Code in expandTilde/expandPath

**Files:**
- `pkg/settings/settings.go:78-103` - `expandTilde()`
- `cmd/rcodegen/main.go:174-178` - `expandPath()`

**Severity:** LOW

Two different implementations of tilde expansion exist. The one in settings.go handles edge cases better.

**Patch-Ready Diff:**
```diff
--- a/cmd/rcodegen/main.go
+++ b/cmd/rcodegen/main.go
@@ -171,8 +171,17 @@ func printUsage() {
 }

 func expandPath(path string) string {
+	if path == "" {
+		return path
+	}
 	if strings.HasPrefix(path, "~/") {
-		return os.Getenv("HOME") + path[1:]
+		home, err := os.UserHomeDir()
+		if err != nil {
+			home = os.Getenv("HOME")
+		}
+		if home != "" {
+			return filepath.Join(home, path[2:])
+		}
 	}
 	return path
 }
```

---

### 10. [PERFORMANCE] Large File Reads Into Memory

**File:** `pkg/orchestrator/orchestrator.go:663-695`

**Severity:** LOW

Multiple functions read entire files into memory for analysis (extractTitle, countWords, extractAngle, etc.). For very large markdown files, this could cause memory pressure.

**Recommendation:** For word counting and similar operations, consider streaming the file or setting reasonable size limits.

---

## Positive Highlights

### Strong Security Patterns

1. **Bundle Name Validation** (`pkg/bundle/loader.go:17-32`): Excellent use of regex validation to prevent path traversal attacks.

2. **Lock Directory Permissions** (`pkg/lock/filelock.go:76`): Lock directory created with 0700 (owner-only).

3. **Settings File Permission Check** (`pkg/settings/settings.go:119-124`): Warns user about world-writable settings.

4. **Claude Tracking Script Location** (`pkg/tracking/claude.go:29-59`): Correctly avoids searching CWD for security.

5. **Command Execution** - Uses `exec.Command` with separate arguments rather than shell concatenation, preventing command injection.

### Good Architecture

1. **Interface-Based Tool Design**: The `runner.Tool` interface allows clean pluggability.

2. **Minimal Dependencies**: Only Go stdlib - reduces supply chain risk.

3. **Separation of Concerns**: Clear package boundaries (runner, orchestrator, tools, bundle).

4. **Thread-Safe Context**: Uses `sync.RWMutex` appropriately for concurrent access.

5. **Factory Pattern for Dispatcher**: Breaks circular dependency cleanly.

### Clean Code Practices

1. **Compile-Time Interface Checks**: `var _ runner.Tool = (*Tool)(nil)` in gemini.go

2. **Builder Pattern**: Used well in envelope package

3. **Consistent Naming**: Package and function names follow Go conventions

4. **Good Documentation**: Most packages have doc comments explaining purpose

---

## Testing Coverage

The codebase includes test files for:
- `pkg/runner/runner_test.go`
- `pkg/runner/stream_test.go`
- `pkg/tools/claude/claude_test.go`
- `pkg/bundle/loader_test.go`
- `pkg/lock/filelock_test.go`
- `pkg/settings/settings_test.go`
- `pkg/workspace/workspace_test.go`

**Missing Tests:**
- `pkg/orchestrator/` - No test files found
- `pkg/executor/` - No test files found
- `pkg/tracking/` - No test files found
- `cmd/rcodegen/` - No test files found

**Recommendation:** Add tests for orchestrator and executor packages, particularly for parallel execution and error handling paths.

---

## Security Recommendations Summary

| Priority | Issue | Effort |
|----------|-------|--------|
| HIGH | Remove CWD search in codex findWrapper() | Low |
| MEDIUM | Add runtime security warning banner | Low |
| MEDIUM | Handle JSON encoding errors | Low |
| LOW | Use 0700 for config directory | Low |
| LOW | Move file I/O outside mutex lock | Medium |

---

## Final Notes

This is a well-designed codebase that demonstrates good Go practices. The main concerns are:

1. **Security inconsistency** between Claude (which correctly avoids CWD) and Codex (which searches CWD)
2. **Silent error handling** in several places that could mask issues
3. **Missing test coverage** for critical orchestration logic

The architecture is clean, the code is readable, and the minimal dependency approach reduces maintenance burden and supply chain risk. With the fixes suggested above, this would be a very solid 88-90 point codebase.

---

*Audit performed by Claude Opus 4.5*
*Generated: 2026-01-16T23:36:00Z*
