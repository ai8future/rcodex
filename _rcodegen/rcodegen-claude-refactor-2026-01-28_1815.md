Date Created: 2026-01-28 18:15:32
TOTAL_SCORE: 82/100

# rcodegen Code Quality & Refactoring Report

## Executive Summary

**rcodegen** is a well-architected Go project providing unified automation for AI coding assistants (Claude, Codex, Gemini). The codebase demonstrates solid software engineering practices with clean separation of concerns, thoughtful abstractions, and comprehensive functionality. However, there are opportunities to reduce duplication, improve consistency, and enhance maintainability.

---

## Grade Breakdown

| Category | Score | Weight | Weighted |
|----------|-------|--------|----------|
| Architecture & Design | 88/100 | 25% | 22.0 |
| Code Duplication | 72/100 | 20% | 14.4 |
| Consistency | 78/100 | 15% | 11.7 |
| Testability | 75/100 | 15% | 11.25 |
| Maintainability | 85/100 | 15% | 12.75 |
| Documentation | 80/100 | 10% | 8.0 |
| **TOTAL** | | | **82/100** |

---

## Detailed Analysis

### 1. Architecture & Design (88/100)

**Strengths:**
- **Clean plugin architecture**: The `runner.Tool` interface (pkg/runner/tool.go:8-90) is well-designed with 20 methods covering all tool-specific behaviors
- **Good separation of concerns**: Distinct packages for runner, tools, orchestrator, executor, settings, etc.
- **Strategy pattern**: Different executors (tool, parallel, merge, vote) in pkg/executor/
- **Builder pattern**: Clean envelope construction (pkg/envelope/envelope.go:35-82)
- **Factory pattern**: DispatcherFactory breaks circular dependency elegantly (pkg/orchestrator/orchestrator.go:56)

**Areas for Improvement:**
- The orchestrator.go file is 1304 lines - consider splitting report generation into a separate package
- Some business logic in orchestrator (article-specific handling) could be more generic

### 2. Code Duplication (72/100)

**Issue 1: ANSI Color Constants Duplicated Across Packages**

The same color constants are defined in multiple places:

```go
// pkg/colors/colors.go (canonical source)
const (
    Reset   = "\033[0m"
    Bold    = "\033[1m"
    ...
)

// pkg/runner/config.go - re-exports from colors
const (
    Bold    = colors.Bold
    ...
)

// pkg/tracking/codex.go - ALSO re-exports from colors
const (
    Bold   = colors.Bold
    ...
)

// pkg/lock/filelock.go - DUPLICATES (doesn't use pkg/colors!)
const (
    Dim   = "\033[2m"
    Green = "\033[32m"
    ...
)

// pkg/reports/manager.go - ALSO DUPLICATES
const (
    Dim    = "\033[2m"
    Yellow = "\033[33m"
    Reset  = "\033[0m"
)

// pkg/settings/settings.go - ALSO DUPLICATES
const (
    bold    = "\033[1m"
    dim     = "\033[2m"
    ...
)

// pkg/orchestrator/orchestrator.go (implied by usage of colorDim, colorReset, etc.)
```

**Recommendation**: All packages should import `rcodegen/pkg/colors` directly instead of re-exporting or duplicating.

**Issue 2: Similar Script-Finding Logic**

`pkg/tracking/codex.go` and `pkg/tracking/claude.go` have nearly identical script-finding patterns:

```go
// In both files:
func GetStatus() *StatusType {
    var statusScript string
    scriptDir := GetScriptDir()
    if scriptDir != "" {
        statusScript = filepath.Join(scriptDir, "get_XXX_status.py")
        if _, err := os.Stat(statusScript); err == nil {
            cmd := exec.Command(FindPython(), statusScript)
            return runStatusScript(cmd)
        }
    }
    home, err := os.UserHomeDir()
    if err == nil {
        statusScript = filepath.Join(home, ".rcodegen", "scripts", "get_XXX_status.py")
        // ... same pattern
    }
}
```

**Recommendation**: Extract common `FindScript(scriptName string) (string, error)` function.

**Issue 3: Tool Implementation Repetition**

The three tool implementations (claude.go, codex.go, gemini.go) share significant boilerplate. Each has:
- Identical `ReportDir()` returning `"_rcodegen"`
- Very similar `SetSettings()` methods
- Similar validation patterns

**Recommendation**: Consider a base struct with embedded functionality.

### 3. Consistency Issues (78/100)

**Issue 1: Inconsistent Function Naming**

```go
// pkg/reports/manager.go
func FindNewestReport(files []string) string  // Takes slice

// pkg/runner/grades.go
func FindNewestReport(reportDir, toolName, taskShortcut string) (string, error)  // Takes separate params
```

Same function name, different signatures - confusing.

**Issue 2: Mixed Error Handling Styles**

```go
// pkg/tracking/codex.go - exits on error
func ShowStatusOnly() {
    if status.Error != "" {
        os.Exit(1)  // Hard exit
    }
}

// pkg/tracking/claude.go - just returns
func ShowClaudeStatusOnly() {
    if status.Error != "" {
        return  // Graceful return
    }
}
```

**Issue 3: Inconsistent Status Struct Naming**

```go
type CreditStatus struct   // For Codex (codex.go)
type ClaudeStatus struct    // For Claude (claude.go)
```

While descriptive, a unified naming like `ToolStatus` with a discriminator field could improve consistency.

### 4. Testability (75/100)

**Strengths:**
- 12 test files covering critical components
- Good coverage of edge cases in flags_test.go, stream_test.go
- Uses table-driven tests appropriately

**Areas for Improvement:**
- No integration tests for the full tool pipeline
- Tool implementations (claude.go, codex.go, gemini.go) would benefit from interface-based mocking
- The orchestrator has no unit tests (only context_test.go and condition_test.go)
- No tests for report generation in orchestrator.go

**Test Files Found:**
1. pkg/runner/stream_test.go
2. pkg/runner/runner_test.go
3. pkg/runner/flags_test.go
4. pkg/workspace/workspace_test.go
5. pkg/lock/filelock_test.go
6. pkg/bundle/loader_test.go
7. pkg/settings/settings_test.go
8. pkg/tools/claude/claude_test.go
9. pkg/orchestrator/context_test.go
10. pkg/orchestrator/condition_test.go
11. pkg/executor/vote_test.go
12. pkg/envelope/envelope_test.go

### 5. Maintainability (85/100)

**Strengths:**
- Clear package structure with logical boundaries
- Good use of interfaces for extensibility
- Comprehensive flag system with short/long aliases
- Settings system with interactive setup wizard

**Areas for Improvement:**
- pkg/orchestrator/orchestrator.go at 1304 lines is too large
  - Lines 433-630: `generateRunReport()` and helpers (198 lines)
  - Lines 799-916: Article-specific extraction functions
  - Lines 1016-1124: `generateFinalReportJSON()` and helpers

**Recommendation**: Split into:
  - `pkg/orchestrator/orchestrator.go` - core orchestration
  - `pkg/orchestrator/reports.go` - report generation
  - `pkg/orchestrator/articles.go` - article-specific logic

**Issue: Magic Numbers**

```go
// pkg/reports/manager.go:23
const reviewScanLines = 10  // Good - named constant

// pkg/runner/runner.go:346-348
for i := 0; i < 10; i++ {  // Bad - magic number
    time.Sleep(50 * time.Millisecond)
}
```

### 6. Documentation (80/100)

**Strengths:**
- Most packages have good package-level doc comments
- Interface methods are well-documented in tool.go
- README.md and CHANGELOG.md present

**Areas for Improvement:**
- Some exported functions lack documentation:
  - `pkg/orchestrator/orchestrator.go:getStepModel()` - no doc
  - Many helper functions in orchestrator.go
- No godoc examples for key APIs

---

## Specific Refactoring Opportunities

### Priority 1: Consolidate Color Constants

**Current State**: 6+ files define ANSI color constants
**Target State**: Single source of truth in `pkg/colors/colors.go`

**Files to Update**:
- pkg/lock/filelock.go - import pkg/colors
- pkg/reports/manager.go - import pkg/colors
- pkg/settings/settings.go - import pkg/colors (rename to match public names)
- pkg/tracking/codex.go - already imports, remove re-exports
- pkg/runner/config.go - remove re-exports, update imports in other runner files

### Priority 2: Extract Report Generation from Orchestrator

**Current**: 1304 lines in orchestrator.go with mixed responsibilities
**Target**:
- orchestrator.go: ~500 lines (core orchestration)
- reports.go: ~400 lines (generateRunReport, generateFinalReportJSON)
- articles.go: ~200 lines (article-specific helpers)

### Priority 3: Unify Script Finding Logic

Create shared function:

```go
// pkg/tracking/scripts.go
func FindScript(scriptName string) (string, error) {
    // Check executable directory
    if dir := GetScriptDir(); dir != "" {
        path := filepath.Join(dir, scriptName)
        if _, err := os.Stat(path); err == nil {
            return path, nil
        }
    }
    // Check user scripts directory
    if home, err := os.UserHomeDir(); err == nil {
        path := filepath.Join(home, ".rcodegen", "scripts", scriptName)
        if _, err := os.Stat(path); err == nil {
            return path, nil
        }
    }
    return "", fmt.Errorf("script %s not found in trusted locations", scriptName)
}
```

### Priority 4: Rename Conflicting Functions

```go
// pkg/reports/manager.go
func FindNewestReport(files []string) string
// Rename to:
func FindNewestFromPaths(files []string) string

// Keep pkg/runner/grades.go as is since it's the primary API
func FindNewestReport(reportDir, toolName, taskShortcut string) (string, error)
```

### Priority 5: Add Missing Tests

Critical untested code:
1. `pkg/orchestrator/orchestrator.go:Run()` - the main orchestration loop
2. `pkg/tools/codex/codex.go` - no codex_test.go exists
3. `pkg/tools/gemini/gemini.go` - no gemini_test.go exists
4. Report generation functions in orchestrator.go

---

## Code Smells Identified

| Issue | Location | Severity |
|-------|----------|----------|
| Long function | orchestrator.go:Run() - 307 lines | Medium |
| Duplicate constants | 6 files with ANSI colors | Medium |
| Magic numbers | runner.go:346-348 (retry loop) | Low |
| Same function name, different signatures | FindNewestReport() | Medium |
| Inconsistent error handling | ShowStatusOnly() vs ShowClaudeStatusOnly() | Low |
| Large file | orchestrator.go - 1304 lines | Medium |
| Missing tests | codex.go, gemini.go, orchestrator.go:Run() | Medium |

---

## Positive Patterns Worth Preserving

1. **Tool Interface Design** (pkg/runner/tool.go): Comprehensive interface with clear method contracts
2. **Builder Pattern** (pkg/envelope/envelope.go): Clean, fluent API for envelope construction
3. **Settings System**: Interactive wizard with sensible defaults and validation
4. **Flag System**: Comprehensive short/long aliases with conflict detection
5. **Stream Parser**: Well-structured JSON stream processing with clear event handling
6. **Lock System**: Secure file-based locking with timeout and polling

---

## Summary

The rcodegen codebase is well-designed with good architectural decisions. The main opportunities for improvement are:

1. **Consolidate duplicated color constants** - Quick win, high impact
2. **Split orchestrator.go** - Medium effort, high maintainability improvement
3. **Unify tracking script logic** - Small effort, reduces duplication
4. **Add missing tests** - Medium effort, improves confidence
5. **Fix naming inconsistencies** - Small effort, reduces confusion

The codebase earns **82/100** - a solid B+ reflecting good engineering practices with room for polish.

---

*Report generated by Claude:Opus 4.5 for rcodegen v1.9.4*
