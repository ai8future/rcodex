Date Created: 2026-01-16 23:55:32 +0100
TOTAL_SCORE: 82/100

# Refactor Review (rcodegen)

## Findings (ordered by severity)
### High
- None observed in this quick pass.

### Medium
- Orchestrator mixes input validation, workspace setup, UI display, step execution, and report generation in a single long flow, which makes changes risky and tests hard; consider splitting into subpackages (e.g., execution, reporting, UI). `pkg/orchestrator/orchestrator.go`
- Report-generation logic for article bundles is embedded in orchestrator with many helper functions; this tightly couples orchestrator to a specific bundle type and duplicates formatting logic. Move to a dedicated report/rendering package or use templates. `pkg/orchestrator/orchestrator.go`
- Tool implementations repeat a large amount of boilerplate (Name, BinaryName, ReportDir, ReportPrefix, defaults, status option help); this invites drift and makes adding a new tool costly. Consider a shared base struct + declarative config, or code-gen for common pieces. `pkg/tools/claude/claude.go` `pkg/tools/codex/codex.go` `pkg/tools/gemini/gemini.go`
- Flag parsing has multiple sources of truth (flag definitions, reorder list, help output). This can drift as options grow; consider consolidating into a single metadata table to render usage and drive parsing/reordering. `pkg/runner/runner.go` `pkg/runner/flags.go`
- Token/cost extraction is implemented separately from stream parsing and uses hard-coded pricing/heuristics. Consolidate into a single usage parser per tool and drive pricing from config/constants to reduce divergence. `pkg/executor/tool.go` `pkg/runner/stream.go`
- Context.Resolve performs file IO while holding a read lock; this can slow down parallel steps and complicate future concurrency. Consider copying needed values under lock and performing IO outside. `pkg/orchestrator/context.go`
- Multiple packages redefine ANSI color codes and formatting conventions, which can drift and makes styling changes expensive. Introduce a small shared module for terminal styling. `pkg/runner/config.go` `pkg/orchestrator/progress.go` `pkg/reports/manager.go` `pkg/lock/filelock.go`

### Low
- Report summary for edits marks both entries as Gemini; likely a copy/paste label bug. `pkg/orchestrator/orchestrator.go`
- Several helper flows drop write errors (e.g., report generation) which can mask failures in unattended runs; log or bubble errors for clarity. `pkg/orchestrator/orchestrator.go`
- Global noTrackStatus flag state is package-scoped, which can leak across tests or reused Runner instances. Prefer making it a field on Config or Runner. `pkg/runner/runner.go`

## Testing gaps
- No focused tests for orchestrator execution paths, step conditions, or report generation; adding table-driven tests around Context.Resolve, EvaluateCondition, and report rendering would reduce regression risk. `pkg/orchestrator`
- Usage parsing in executor is heuristic and untested; small fixtures per tool would protect against CLI output changes. `pkg/executor/tool.go`

## Score rationale
- Strengths: clear package boundaries, strong use of interfaces (runner.Tool), security-minded defaults, and some unit tests.
- Deductions: large multi-responsibility functions, duplicated tool boilerplate, and scattered formatting/state logic that will be hard to extend.

## Scope notes
- Quick scan of core Go packages only; no deep dive into wrapper scripts or _rcodegen artifacts per instructions.
