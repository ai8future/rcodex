Date Created: 2026-01-16 23:46:00
TOTAL_SCORE: 78/100

# rcodegen Refactoring Report

## Executive Summary

The rcodegen codebase is a well-structured Go monorepo providing unified automation for AI code assistants (Claude, Codex, Gemini). The architecture demonstrates strong design principles with a clean separation of concerns, consistent interface patterns, and thoughtful dependency management. However, there are opportunities to reduce code duplication, consolidate related functionality, and improve maintainability.

---

## Scoring Breakdown

| Category | Score | Weight | Weighted |
|----------|-------|--------|----------|
| Architecture & Design | 85/100 | 25% | 21.25 |
| Code Duplication | 65/100 | 20% | 13.00 |
| Maintainability | 80/100 | 20% | 16.00 |
| Error Handling | 75/100 | 15% | 11.25 |
| Testing Coverage | 60/100 | 10% | 6.00 |
| Documentation | 85/100 | 10% | 8.50 |
| **TOTAL** | | | **76/100** |

Adjusted to **78/100** for good use of interfaces and dependency injection patterns.

---

## Detailed Analysis

### 1. Code Duplication Issues (Major)

#### 1.1 ANSI Color Constants Defined Multiple Times

**Problem:** Color constants are defined in at least 3 separate locations:

- `pkg/runner/config.go` lines 8-17
- `pkg/tracking/codex.go` lines 12-19
- `pkg/orchestrator/orchestrator.go` (using local `colorXxx` variables)

**Impact:** Medium - Maintenance burden, risk of inconsistency

**Recommendation:** Create a single `pkg/colors/colors.go` package:
```go
package colors

const (
    Bold    = "\033[1m"
    Dim     = "\033[2m"
    Reset   = "\033[0m"
    // ... etc
)
```

All packages should import from this single source.

---

#### 1.2 Tool Implementations Share 60%+ Boilerplate

**Problem:** The three tool implementations (`claude.go`, `codex.go`, `gemini.go`) share significant structural similarities:

- `Name()`, `BinaryName()`, `ReportDir()`, `ReportPrefix()` - trivial one-liners
- `SetSettings()` - identical implementation
- `DefaultModelSetting()` - same pattern, different field access
- `ApplyToolDefaults()` - similar structure
- Status tracking methods - parallel implementations

**Files affected:**
- `pkg/tools/claude/claude.go` (408 lines)
- `pkg/tools/codex/codex.go` (337 lines)
- `pkg/tools/gemini/gemini.go` (229 lines)

**Impact:** Medium - Adding a new tool requires copying significant boilerplate

**Recommendation:** Create a `BaseTool` struct with embedded composition:
```go
type BaseTool struct {
    name         string
    binaryName   string
    reportPrefix string
    settings     *settings.Settings
}

func (b *BaseTool) ReportDir() string { return "_rcodegen" }
// ... other shared methods
```

Tools would embed `BaseTool` and only override specific behavior.

---

#### 1.3 Status Tracking Scripts Follow Same Pattern

**Problem:** `GetClaudeStatus()` and `GetStatus()` in the tracking package are nearly identical:
- Same script discovery logic (executable dir, then ~/.rcodegen/scripts/)
- Same error handling patterns
- Same JSON parsing approach

**Files:** `pkg/tracking/claude.go:29-60`, `pkg/tracking/codex.go:87-118`

**Impact:** Low-Medium - Code is functional but could be DRY-er

**Recommendation:** Create a generic `runStatusScript[T any]()` function:
```go
func getStatusWithScript[T any](scriptName string) *T {
    scriptPath := findScript(scriptName) // shared discovery logic
    if scriptPath == "" {
        return nil
    }
    return runScript[T](scriptPath)
}
```

---

#### 1.4 PrintStatusSummary Nearly Identical Across Tools

**Problem:** `PrintStatusSummary()` in both Claude and Codex tools follows the same pattern:
- Type assertion check
- Extract before/after values
- Calculate differences
- Format and print with colors

**Impact:** Low - The differences are subtle but extractable

**Recommendation:** Create a generic status display helper that accepts a `StatusDiff` struct.

---

### 2. Architecture Concerns

#### 2.1 Orchestrator is Overly Long (1300 lines)

**Problem:** `pkg/orchestrator/orchestrator.go` is 1300 lines and handles:
- Bundle execution coordination
- Step statistics tracking
- Run report generation (article bundles)
- Final report JSON generation (build bundles)
- Text extraction utilities (title, angle, tone, data)
- File scanning and categorization

**Impact:** Medium - Difficult to navigate, test, and maintain

**Recommendation:** Split into focused files:
- `orchestrator.go` - Core execution (~300 lines)
- `article_report.go` - Article-specific reporting (~400 lines)
- `build_report.go` - Build-specific JSON reporting (~300 lines)
- `text_analysis.go` - Title/angle/tone extraction (~200 lines)

---

#### 2.2 Large Interface (23 Methods)

**Problem:** The `runner.Tool` interface has 23 methods, some of which have default behaviors that all tools share.

**File:** `pkg/runner/tool.go`

**Impact:** Medium - New tool implementations require implementing many methods

**Recommendation:** Use interface segregation:
```go
type Tool interface {
    CoreTool    // Name, BinaryName, BuildCommand (required)
    Reporter    // ReportDir, ReportPrefix
    Validator   // ValidModels, DefaultModel, ValidateConfig
}

type StatusTracker interface {
    SupportsStatusTracking() bool
    CaptureStatusBefore() interface{}
    CaptureStatusAfter() interface{}
    PrintStatusSummary(before, after interface{})
}
```

---

#### 2.3 Package-Level Variable for Circular Dependency

**Problem:** `orchestrator.DispatcherFactory` is a package-level variable set via `init()` in the executor package to break a circular dependency.

**Files:**
- `pkg/orchestrator/orchestrator.go:56`
- `pkg/executor/dispatcher.go:11-15`

**Impact:** Low - Works but is implicit and could be confusing

**Recommendation:** Consider using a constructor injection pattern:
```go
func NewOrchestrator(settings *Settings, dispatcherFactory func(tools) StepExecutor) *Orchestrator
```

---

### 3. Maintainability Concerns

#### 3.1 Magic Strings for Task Names

**Problem:** Report types hardcoded as strings in multiple places:
```go
reportTypes := []string{"audit", "test", "fix", "refactor", "quick"}
```

**Files:** `runner.go:383`, `runner.go:254`

**Impact:** Low - Risk of typos, no compile-time safety

**Recommendation:** Define task type constants:
```go
const (
    TaskAudit    = "audit"
    TaskTest     = "test"
    TaskFix      = "fix"
    TaskRefactor = "refactor"
    TaskQuick    = "quick"
)

var ReportTypes = []string{TaskAudit, TaskTest, TaskFix, TaskRefactor, TaskQuick}
```

---

#### 3.2 Model Validation Duplicated

**Problem:** Model validation logic duplicated between tools:
- `claude.go:306-309` - hardcoded map
- `gemini.go:157-168` - hardcoded map

Both also have `ValidModels()` returning the same list.

**Impact:** Low - Validation logic should derive from `ValidModels()`

**Recommendation:** Add a generic validation helper:
```go
func ValidateModel(model string, tool Tool) error {
    for _, valid := range tool.ValidModels() {
        if model == valid {
            return nil
        }
    }
    return fmt.Errorf("invalid model '%s'. Valid: %v", model, tool.ValidModels())
}
```

---

#### 3.3 Settings Defaults Scattered

**Problem:** Default values for settings defined in multiple places:
- `GetDefaultSettings()` in settings.go
- `LoadWithFallback()` fills in missing defaults
- Each tool's `ApplyToolDefaults()` also has defaults

**Impact:** Medium - Confusing which defaults take precedence

**Recommendation:** Consolidate all defaults into a single source of truth, possibly in the settings package.

---

### 4. Error Handling

#### 4.1 Silent Failures in Orchestrator

**Problem:** Some file operations silently ignore errors:
```go
os.WriteFile(path, []byte(sb.String()), 0644) // line 625 - no error check
```

**Impact:** Low - User may not know report generation failed

**Recommendation:** At minimum log warnings for failed file operations.

---

#### 4.2 Inconsistent Error Returns

**Problem:** Some functions return `nil, nil` for "exit cleanly" conditions:
```go
// parseArgs returns (nil, nil) when help or tasks were shown
```

**Impact:** Low - Semantic meaning of nil config + nil error must be documented

**Recommendation:** Consider using a custom type or sentinel error for "clean exit" scenarios.

---

### 5. Testing Coverage

**Problem:** Limited test files found (4 `_test.go` files for 35 source files).

**Impact:** Medium-High - Refactoring is risky without test coverage

**Recommendation:** Prioritize tests for:
1. `pkg/runner/flags.go` - flag parsing logic
2. `pkg/settings/settings.go` - configuration loading
3. `pkg/bundle/loader.go` - bundle parsing
4. `pkg/executor/dispatcher.go` - step routing

---

### 6. Minor Issues

#### 6.1 Unused Function
`getArticleNames()` in orchestrator.go appears unused.

#### 6.2 Hardcoded Python Path Priority
The `FindPython()` function prioritizes `/opt/homebrew/bin/python3.13` - this may not be portable.

#### 6.3 Version Discovery
`getVersion()` searches relative paths for VERSION file - fragile approach.

---

## Recommended Refactoring Priority

### High Priority
1. **Extract shared color constants** - Quick win, reduces maintenance burden
2. **Split orchestrator.go** - Improves navigability significantly
3. **Add core unit tests** - Enables safer refactoring

### Medium Priority
4. **Create BaseTool abstraction** - Reduces tool implementation boilerplate
5. **Consolidate status tracking** - DRY improvement
6. **Interface segregation** - Reduces implementation burden

### Low Priority
7. **Task name constants** - Minor type safety improvement
8. **Generic model validation** - Minor DRY improvement
9. **Dependency injection for DispatcherFactory** - Architectural cleanliness

---

## Conclusion

The rcodegen codebase exhibits solid architectural foundations with good separation of concerns and interface-driven design. The main opportunities for improvement are:

1. **DRY violations** - Color constants and tool boilerplate
2. **Large file sizes** - orchestrator.go should be split
3. **Test coverage** - Current coverage is insufficient for confident refactoring

The score of **78/100** reflects a codebase that is functional and well-organized but has room for improvement in reducing duplication and improving testability.

---

*Report generated by Claude Code automation*
