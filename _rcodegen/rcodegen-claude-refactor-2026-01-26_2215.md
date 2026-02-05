Date Created: 2026-01-26 22:15:00 UTC
TOTAL_SCORE: 74/100

# rcodegen Code Quality & Refactoring Report

**Reviewed by:** Claude Code (Opus 4.5)
**Project Version:** 1.9.2
**Total Production Lines:** ~8,000 Go LOC + ~500 Python LOC

---

## Executive Summary

rcodegen is a well-architected monorepo providing unified CLI wrappers for AI coding assistants (Claude Code, Codex, Gemini). The codebase demonstrates solid software engineering principles with clean plugin architecture and good separation of concerns. However, opportunities exist to reduce duplication, improve test coverage, and enhance maintainability.

| Category | Score | Weight | Weighted |
|----------|-------|--------|----------|
| Architecture & Design | 85/100 | 25% | 21.25 |
| Code Duplication | 68/100 | 20% | 13.60 |
| Test Coverage | 35/100 | 20% | 7.00 |
| Maintainability | 72/100 | 20% | 14.40 |
| Error Handling | 70/100 | 10% | 7.00 |
| Build & Tooling | 75/100 | 5% | 3.75 |
| **TOTAL** | | | **74/100** |

---

## 1. Architecture & Design (85/100)

### Strengths

**Clean Plugin Interface Pattern**
The `runner.Tool` interface (~23 methods) provides excellent abstraction for adding new AI tools:

```
pkg/runner/tool.go (interface)
    ↓
pkg/tools/claude/claude.go (409 lines)
pkg/tools/codex/codex.go (336 lines)
pkg/tools/gemini/gemini.go (216 lines)
```

**Well-Layered Architecture**
```
CLI Entry Points (cmd/*)
    ↓
Runner (orchestrates execution)
    ↓
Tool Implementations (tool-specific logic)
    ↓
External Binaries (claude, codex, gemini CLIs)
```

**Dispatcher Factory Pattern**
The orchestrator avoids circular imports via factory registration:
```go
// orchestrator.go
var DispatcherFactory func(tools map[string]runner.Tool) StepExecutor

// executor/dispatcher.go - registered via init()
orchestrator.DispatcherFactory = func(...) { ... }
```

### Concerns

- Large files: `orchestrator.go` (1,303 lines), `runner.go` (1,006 lines)
- `parseArgs()` function exceeds 350 lines
- Settings migration logic tightly coupled to config loading

---

## 2. Code Duplication (68/100)

### SIGNIFICANT: Tool Interface Implementation Boilerplate

All three tools repeat nearly identical implementations for 10+ methods:

| Method | claude.go | codex.go | gemini.go |
|--------|-----------|----------|-----------|
| `Name()` | ✓ identical pattern | ✓ identical pattern | ✓ identical pattern |
| `BinaryName()` | ✓ identical pattern | ✓ identical pattern | ✓ identical pattern |
| `ReportDir()` | Returns `_rcodegen` | Returns `_rcodegen` | Returns `_rcodegen` |
| `ReportPrefix()` | Only differs by name | Only differs by name | Only differs by name |
| `DefaultModelSetting()` | ✓ identical logic | ✓ identical logic | ✓ identical logic |
| `ValidateConfig()` | Calls `runner.ValidateModel()` | Calls `runner.ValidateModel()` | Calls `runner.ValidateModel()` |

**Estimated Duplication:** 200-300 lines across tools

**Recommendation:** Extract `BaseTool` struct with default implementations:

```go
// pkg/runner/base_tool.go
type BaseTool struct {
    name          string
    binaryName    string
    reportPrefix  string
    validModels   []string
    defaultModel  string
    settings      *settings.Settings
}

func (b *BaseTool) Name() string { return b.name }
func (b *BaseTool) ReportDir() string { return "_rcodegen" }
// ... other defaults
```

### MODERATE: Status Tracking Stubs

Gemini implements 8+ no-op methods for status tracking that it doesn't support:

```go
func (t *Tool) SupportsStatusTracking() bool { return false }
func (t *Tool) GetStatusBefore() interface{} { return nil }
func (t *Tool) GetStatusAfter(interface{}) interface{} { return nil }
func (t *Tool) FormatStatusDiff(...) string { return "" }
// ... more stubs
```

**Recommendation:** Extract `StatusTracker` interface with `NoOpStatusTracker` default.

### MODERATE: Banner/Output Formatting

Each tool duplicates banner generation logic:
- `PrintToolSpecificBannerFields()`
- `PrintToolSpecificSummaryFields()`
- `BannerTitle()` / `BannerSubtitle()`

---

## 3. Test Coverage (35/100)

### Critical Gap

| Component | LOC | Test Count | Effective Coverage |
|-----------|-----|------------|-------------------|
| orchestrator | 1,303 | 2 tests | ~0.15% |
| runner | 1,006 | 3 tests | ~0.3% |
| settings | 626 | 5 tests | ~0.8% |
| stream | 317 | 8 tests | ~2.5% |
| flags | 183 | 14 tests | ~7.6% |

**Key Issues:**
1. Orchestrator (largest file) has minimal test coverage
2. No integration tests for end-to-end workflows
3. No mocked CLI binary tests
4. Report generation workflow untested

**Recommendation:** Target 80%+ coverage with:
- Unit tests for all public functions
- Integration tests with mocked CLI binaries
- Edge case tests for stream parsing

---

## 4. Maintainability (72/100)

### Large Function Analysis

| Function | Location | LOC | Complexity |
|----------|----------|-----|------------|
| `parseArgs()` | runner.go | ~350 | Very High |
| `orchestrator.Run()` | orchestrator.go | ~250 | High |
| `runner.Run()` | runner.go | ~175 | High |
| `live_display.Render()` | live_display.go | ~200 | High |
| `Stream.ProcessLine()` | stream.go | ~100 | Moderate |

### Recommendations

**Break Up `parseArgs()`:**
```go
// Current: 350+ line monolith
func parseArgs(...) (...)

// Proposed: Split by responsibility
func parseBaseFlags(...) (...)
func parseTaskArgs(...) (...)
func parseModelArgs(...) (...)
func validateArgs(...) (...)
```

**Extract Orchestrator Phases:**
```go
// Current: Single 250+ line Run()
func (o *Orchestrator) Run() error

// Proposed: Phase separation
func (o *Orchestrator) initializeRun() error
func (o *Orchestrator) executeSteps() error
func (o *Orchestrator) finalizeRun() error
```

### Code Organization

**Well-organized packages:**
- `pkg/runner/` - Core execution framework
- `pkg/tools/` - Tool implementations (good isolation)
- `pkg/settings/` - Configuration management
- `pkg/orchestrator/` - Workflow orchestration
- `pkg/executor/` - Step execution engine

**Minor concerns:**
- Migration logic in `migrate.go` could be separate package
- Some utility functions scattered across files

---

## 5. Error Handling (70/100)

### Issues Identified

**Silent Error Swallowing:**
```go
// settings.go - errors silently converted to defaults
func LoadWithFallback() (*Settings, error) {
    s, err := Load()
    if err != nil {
        return DefaultSettings(), nil  // Error lost
    }
    return s, nil
}
```

**Generic Error Messages:**
```go
// Lacks context (file path, line number, value)
return fmt.Errorf("invalid model: %s", model)
```

**No Structured Error Types:**
```go
// Current: string-based errors
// Recommended:
type ConfigError struct {
    Field   string
    Value   string
    Reason  string
    Source  string
}
```

### Recommendations

1. Define custom error types in `pkg/errors/`
2. Add context to all error messages
3. Use error wrapping: `fmt.Errorf("loading config: %w", err)`
4. Log silently-handled errors for debugging

---

## 6. Build & Tooling (75/100)

### Makefile Analysis

**Current:**
```makefile
all: rcodex rclaude rcodegen rgemini
clean: rm -f binaries
test: go test ./pkg/...
```

**Missing:**
- No linting targets (`gofmt`, `golangci-lint`)
- No test coverage reporting
- No version injection into binaries
- No separate targets per binary

**Recommended Additions:**
```makefile
.PHONY: lint coverage vet

lint:
	golangci-lint run ./...

vet:
	go vet ./...

coverage:
	go test -coverprofile=coverage.out ./pkg/...
	go tool cover -html=coverage.out -o coverage.html

build-rclaude:
	go build -ldflags "-X main.Version=$(VERSION)" -o rclaude ./cmd/rclaude
```

---

## 7. Python Scripts Quality

### get_claude_status.py (~300 lines)

**Strengths:**
- Multiple fallback paths for robustness
- Secure script discovery (won't load from CWD)
- Comprehensive regex-based status parsing

**Issues:**
- No timeout on iTerm2 API calls (potential hang)
- Complex regex patterns lack documentation
- Heavy dependency on iTerm2 Python package

### codex_pty_wrapper.py (~140 lines)

**Purpose:** PTY session resumption for Codex

**Issues:**
- Limited error handling for PTY operations
- Could be replaced with shell script for portability

---

## 8. Concurrency Safety (Good)

**Correctly Implemented:**
- `sync.Once` for lazy status checking initialization
- `sync.WaitGroup` for parallel step execution
- File locking via `syscall.FcntlFlock` for atomic operations

**No issues found** in concurrency handling.

---

## 9. Priority Recommendations

### Priority 1: High Impact

| # | Recommendation | Effort | Impact | Lines Saved |
|---|----------------|--------|--------|-------------|
| 1 | Extract BaseTool struct | Medium | High | 200-300 |
| 2 | Add integration tests | High | High | - |
| 3 | Break up parseArgs() | Medium | Medium | - |

### Priority 2: Medium Impact

| # | Recommendation | Effort | Impact |
|---|----------------|--------|--------|
| 4 | Add structured error types | Medium | Medium |
| 5 | Extract StatusTracker interface | Low | Low |
| 6 | Add linting to Makefile | Low | Low |

### Priority 3: Code Health

| # | Recommendation | Effort | Impact |
|---|----------------|--------|--------|
| 7 | Add timeout to iTerm2 API calls | Low | Medium |
| 8 | Document complex regex patterns | Low | Low |
| 9 | Add version to settings schema | Low | Low |

---

## 10. File-by-File Summary

| File | LOC | Quality | Notes |
|------|-----|---------|-------|
| `pkg/runner/runner.go` | 1,006 | Good | Large functions need splitting |
| `pkg/orchestrator/orchestrator.go` | 1,303 | Good | Largest file, phases could separate |
| `pkg/settings/settings.go` | 626 | Good | Clean config management |
| `pkg/tools/claude/claude.go` | 409 | Good | Some boilerplate duplication |
| `pkg/tools/codex/codex.go` | 336 | Good | Some boilerplate duplication |
| `pkg/tools/gemini/gemini.go` | 216 | Fair | Many stub methods |
| `pkg/runner/stream.go` | 317 | Good | Complex but well-structured |
| `pkg/runner/flags.go` | 183 | Good | Best test coverage |
| `pkg/runner/output.go` | 237 | Good | Clean formatting logic |
| `pkg/runner/grades.go` | 274 | Good | Grade calculation well-isolated |

---

## Conclusion

rcodegen demonstrates solid software engineering with a clean plugin architecture that enables easy extension. The primary opportunities for improvement are:

1. **Reduce duplication** via base class extraction (~300 lines)
2. **Increase test coverage** from ~2% to 80%+
3. **Improve maintainability** by breaking up large functions
4. **Standardize error handling** with structured error types

The codebase is production-ready but would benefit from the refactoring investments outlined above, particularly in test coverage which is the most significant gap.

**Estimated Refactoring Effort:** 4-6 developer-weeks for Priority 1 items.
