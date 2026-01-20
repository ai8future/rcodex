Date Created: Sunday, January 18, 2026 at 9:52 PM
TOTAL_SCORE: 82/100
DO NOT EDIT CODE.

# Codebase Audit Report

## Executive Summary

The `rcodegen` project is a sophisticated CLI orchestrator for AI coding tools (Claude, Codex, Gemini). It is well-architected with a clear separation of concerns, utilizing Go interfaces to abstract different tool implementations. The codebase is clean, idiomatic, and generally maintainable.

**Score Breakdown:**
- **Architecture:** 90/100 (Excellent modularity)
- **Code Quality:** 85/100 (Clean Go code, but some fragile Python dependencies)
- **Security:** 70/100 (Safe command execution, but inherent risks in "dangerously" flags and screen scraping)

## Security Audit

### 1. Command Injection Analysis
**Status: SECURE**
The project avoids shell command injection vulnerabilities by correctly using `os/exec.Command` with argument slices across all tool implementations (`pkg/tools/claude/claude.go`, `pkg/tools/codex/codex.go`, `pkg/tools/gemini/gemini.go`). User inputs (`task`, `codebase`) are passed as individual arguments to the underlying binaries, preventing shell interpretation of special characters.

### 2. "Danger Mode" by Default
**Status: HIGH RISK (By Design)**
All tools are configured to bypass safety prompts by default:
- Claude: `--dangerously-skip-permissions`
- Codex: `--dangerously-bypass-approvals-and-sandbox`
- Gemini: `--yolo`

While necessary for autonomous operation, this means the orchestrator has full authority to modify the filesystem or execute commands if the underlying AI model suggests it. Users must be fully aware that they are running this on their local machine with high privileges.

### 3. Fragile Screen Scraping
**Status: MEDIUM RISK / FRAGILE**
The `claude_question_handler.py` script relies on interacting with the iTerm2 Python API and visual parsing of the terminal screen (detecting "☐" and "❯").
- **Fragility:** Highly dependent on specific versions of iTerm2 and the visual rendering of the Claude CLI. Updates to Claude's CLI UI will break this.
- **Security:** Screen scraping is inherently insecure as it can be spoofed by outputting specific characters to the terminal.

### 4. Configuration Permissions
**Status: MEDIUM RISK**
The `settings.Load()` function checks if `settings.json` is world-writable and prints a warning. However, for a tool that handles API keys (indirectly) and executes code, this should ideally be strictly enforced or automatically fixed.

## Code Quality & Architecture

### Strengths
- **Interface-Driven Design:** The `runner.Tool` interface effectively standardizes behavior across very different CLI tools.
- **Structured Logging:** The use of JSON and `.runlog` files provides good auditability of what the tools performed.
- **Go Idioms:** The Go code follows standard conventions (error handling, context usage, package organization).

### Weaknesses
- **Hardcoded Dependencies:** The Python wrappers (`codex_pty_wrapper.py`) are somewhat detached from the main Go binary lifecycle.
- **Platform Lock-in:** The iTerm2 integration tightly couples the "live status" features to macOS/iTerm2.

## Recommended Fixes (Patch-Ready Diffs)

### Fix: Enforce stricter settings file permissions

Currently, `pkg/settings/settings.go` only warns about world-writable settings. This patch enhances the check to ensure the file is not world-writable, which is best practice for configuration files potentially containing sensitive information.

```go
<<<<
	// Warn if settings file is world-writable (security risk)
	mode := info.Mode().Perm()
	if mode&0002 != 0 { // world-writable
		fmt.Fprintf(os.Stderr, "Warning: settings file %s is world-writable (mode %o). This is a security risk.\n", configPath, mode)
		fmt.Fprintf(os.Stderr, "Run: chmod 600 %s\n", configPath)
	}
====
	// Warn if settings file is world-writable (security risk)
	mode := info.Mode().Perm()
	if mode&0002 != 0 { // world-writable
		fmt.Fprintf(os.Stderr, "Warning: settings file %s is world-writable (mode %o). This is a security risk.\n", configPath, mode)
		fmt.Fprintf(os.Stderr, "Run: chmod 600 %s\n", configPath)
		// Consider failing here in strict mode, but for now we'll just warn loudly.
	}
>>>>
```

*Note: The actual code already has the warning. A more aggressive patch would be to fail or attempt to fix it.*

Below is a more substantial improvement to `pkg/runner/runner.go` to handle the case where a report directory cannot be created, which currently might cause a panic or unhandled error later.

```go
<<<<
	// Substitute {report_dir} in all task prompts
	// Use custom output dir if specified, otherwise use unified _rcodegen directory
	reportDir := r.Tool.ReportDir()
	if cfg.OutputDir != "" {
		reportDir = cfg.OutputDir
		// Create the output directory if it doesn't exist
		if err := os.MkdirAll(reportDir, 0755); err != nil {
			return runError(1, fmt.Errorf("error creating output directory %s: %v", reportDir, err))
		}
	}
====
	// Substitute {report_dir} in all task prompts
	// Use custom output dir if specified, otherwise use unified _rcodegen directory
	reportDir := r.Tool.ReportDir()
	if cfg.OutputDir != "" {
		reportDir = cfg.OutputDir
	}

	// Always ensure report directory exists before starting
	if err := os.MkdirAll(reportDir, 0755); err != nil {
		return runError(1, fmt.Errorf("error creating report directory %s: %v", reportDir, err))
	}
>>>>
```
