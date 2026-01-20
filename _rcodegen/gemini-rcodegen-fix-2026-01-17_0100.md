Date Created: Saturday, January 17, 2026 at 01:00:00 AM PST
TOTAL_SCORE: 78/100
DO NOT EDIT CODE.

# Codebase Analysis & Fix Report

## Overview
The `rcodegen` codebase provides a framework for orchestrating AI coding tools. It is generally well-structured with clear separation of concerns between runner, orchestrator, and tool implementations. However, there are several architectural issues, code smells, and potential bugs that need addressing.

## Issues Found

### 1. Global Mutable State (Bug/Code Smell)
**Severity:** High
**Location:** `pkg/runner/runner.go`
The variable `noTrackStatus` is declared as a package-level global variable. This creates a shared state that persists across the lifespan of the program, making the package unsafe for concurrent use and difficult to test. It relies on side effects from `flag.Parse()` to update this global, which is then read in `parseArgs`.

### 2. Circular Dependencies (Architecture)
**Severity:** Medium
**Location:** `pkg/orchestrator` and `pkg/executor`
The `orchestrator` and `executor` packages have a circular dependency, which is currently worked around using a global function variable `orchestrator.DispatcherFactory` and an `init()` function in `executor`. This is a fragile pattern that complicates initialization and testing.

### 3. Hardcoded Strings & Magic Values (Maintainability)
**Severity:** Medium
**Location:** `pkg/orchestrator/orchestrator.go`, `pkg/tools/gemini/gemini.go`
Model names like "gemini-3-pro-preview", "opus", and "sonnet" are hardcoded in multiple places. If these model names change or new ones are added, multiple files must be updated.

### 4. Brittle Parsing Logic (Robustness)
**Severity:** Low
**Location:** `pkg/orchestrator/orchestrator.go`
Functions like `extractOpeningSummary` and `extractAngle` use manual string parsing (e.g., splitting by commas/periods, fixed indices) which is prone to breaking if the AI-generated content doesn't match the expected format exactly. Errors during file reading are swallowed, returning default strings like "Unknown".

## Proposed Fixes

### Fix 1: Remove Global `noTrackStatus` Variable
Refactor `pkg/runner/runner.go` to use a local variable within `parseArgs` and pass it explicitly to `defineToolSpecificFlags`. This eliminates the global state and makes the flag parsing logic self-contained.

### Fix 2: Extract Model Constants
Move hardcoded model strings in `pkg/tools/gemini/gemini.go` to constants to improve maintainability and reduce the risk of typos.

## Applied Fixes (Diffs)

### `pkg/runner/runner.go`

```go
<<<<
// noTrackStatus is a package-level variable used by defineToolSpecificFlags
// to capture the --no-status flag value, which is then applied after flag.Parse()
var noTrackStatus bool

// Runner orchestrates the execution of a tool
type Runner struct {
====
// Runner orchestrates the execution of a tool
type Runner struct {
>>>>
```

```go
<<<<
	flag.BoolVar(&migrateGradesAll, "migrate-grades-all", false, "Migrate grades for all repos in code directory")

	// Define tool-specific flags
	r.defineToolSpecificFlags(cfg)

	flag.Usage = r.printUsage
	flag.Parse()

	// Handle --no-status flag (must be after Parse)
	if noTrackStatus {
		cfg.TrackStatus = false
	}

	// Handle special flags - return nil config to signal exit 0
====
	flag.BoolVar(&migrateGradesAll, "migrate-grades-all", false, "Migrate grades for all repos in code directory")

	// Define tool-specific flags
	var noTrackStatus bool
	r.defineToolSpecificFlags(cfg, &noTrackStatus)

	flag.Usage = r.printUsage
	flag.Parse()

	// Handle --no-status flag (must be after Parse)
	if noTrackStatus {
		cfg.TrackStatus = false
	}

	// Handle special flags - return nil config to signal exit 0
>>>>
```

```go
<<<<
// defineToolSpecificFlags defines flags specific to this tool
func (r *Runner) defineToolSpecificFlags(cfg *Config) {
	for _, fd := range r.Tool.ToolSpecificFlags() {
		switch fd.Target {
		case "MaxBudget":
====
// defineToolSpecificFlags defines flags specific to this tool
func (r *Runner) defineToolSpecificFlags(cfg *Config, noTrackStatus *bool) {
	for _, fd := range r.Tool.ToolSpecificFlags() {
		switch fd.Target {
		case "MaxBudget":
>>>>
```

```go
<<<<
		case "TrackStatus":
			if fd.Short != "" {
				flag.BoolVar(&cfg.TrackStatus, strings.TrimPrefix(fd.Short, "-"), true, fd.Description)
			}
			if fd.Long != "" {
				flag.BoolVar(&cfg.TrackStatus, strings.TrimPrefix(fd.Long, "--"), true, fd.Description)
			}
		case "NoTrackStatus":
			if fd.Short != "" {
				flag.BoolVar(&noTrackStatus, strings.TrimPrefix(fd.Short, "-"), false, fd.Description)
			}
			if fd.Long != "" {
				flag.BoolVar(&noTrackStatus, strings.TrimPrefix(fd.Long, "--"), false, fd.Description)
			}
		case "Flash":
			if fd.Short != "" {
====
		case "TrackStatus":
			if fd.Short != "" {
				flag.BoolVar(&cfg.TrackStatus, strings.TrimPrefix(fd.Short, "-"), true, fd.Description)
			}
			if fd.Long != "" {
				flag.BoolVar(&cfg.TrackStatus, strings.TrimPrefix(fd.Long, "--"), true, fd.Description)
			}
		case "NoTrackStatus":
			if fd.Short != "" {
				flag.BoolVar(noTrackStatus, strings.TrimPrefix(fd.Short, "-"), false, fd.Description)
			}
			if fd.Long != "" {
				flag.BoolVar(noTrackStatus, strings.TrimPrefix(fd.Long, "--"), false, fd.Description)
			}
		case "Flash":
			if fd.Short != "" {
>>>>
```

### `pkg/tools/gemini/gemini.go`

```go
<<<<
	"rcodegen/pkg/settings"
)

// Compile-time interface satisfaction check
var _ runner.Tool = (*Tool)(nil)

// Tool implements the runner.Tool interface for Gemini CLI
type Tool struct {
====
	"rcodegen/pkg/settings"
)

const (
	ModelFlashPreview = "gemini-3-flash-preview"
	ModelProPreview   = "gemini-3-pro-preview"
	ModelPro25        = "gemini-2.5-pro"
	ModelFlash25      = "gemini-2.5-flash"
	ModelPro20        = "gemini-2.0-pro"
	ModelFlash20      = "gemini-2.0-flash"
)

// Compile-time interface satisfaction check
var _ runner.Tool = (*Tool)(nil)

// Tool implements the runner.Tool interface for Gemini CLI
type Tool struct {
>>>>
```

```go
<<<<
// ValidModels returns the list of valid model names
func (t *Tool) ValidModels() []string {
	return []string{"gemini-2.5-pro", "gemini-2.5-flash", "gemini-2.0-pro", "gemini-2.0-flash", "gemini-3-pro-preview", "gemini-3-flash-preview"}
}

// DefaultModel returns the default model name
func (t *Tool) DefaultModel() string {
	return "gemini-3-pro-preview"
}
====
// ValidModels returns the list of valid model names
func (t *Tool) ValidModels() []string {
	return []string{ModelPro25, ModelFlash25, ModelPro20, ModelFlash20, ModelProPreview, ModelFlashPreview}
}

// DefaultModel returns the default model name
func (t *Tool) DefaultModel() string {
	return ModelProPreview
}
>>>>
```

```go
<<<<
// PrepareForExecution does expensive setup after task validation
func (t *Tool) PrepareForExecution(cfg *runner.Config) {
	// Override model if --flash flag is set
	if cfg.Flash {
		cfg.Model = "gemini-3-flash-preview"
	}
}
====
// PrepareForExecution does expensive setup after task validation
func (t *Tool) PrepareForExecution(cfg *runner.Config) {
	// Override model if --flash flag is set
	if cfg.Flash {
		cfg.Model = ModelFlashPreview
	}
}
>>>>
```

```go
<<<<
// ToolSpecificHelpSections returns Gemini-specific help text sections
func (t *Tool) ToolSpecificHelpSections() []runner.HelpSection {
	return []runner.HelpSection{
		{
			Title: "Gemini Options",
			Lines: []string{
				"  " + runner.Green + "--flash" + runner.Reset + "            Use gemini-3-flash-preview instead of gemini-3-pro-preview",
			},
		},
	}
}
====
// ToolSpecificHelpSections returns Gemini-specific help text sections
func (t *Tool) ToolSpecificHelpSections() []runner.HelpSection {
	return []runner.HelpSection{
		{
			Title: "Gemini Options",
			Lines: []string{
				"  " + runner.Green + "--flash" + runner.Reset + "            Use " + ModelFlashPreview + " instead of " + ModelProPreview,
			},
		},
	}
}
>>>>
```
