Date Created: 2026-01-17 00:07:16 +0100
TOTAL_SCORE: 82/100

# rcodegen Refactor Review

## Scope
Quick scan of the Go core (runner/orchestrator/executor/settings/bundle/reports/workspace/tracking) plus dashboard API routes to identify code quality, duplication, and maintainability opportunities. No code changes requested or performed.

## Overall Assessment
Strengths: Clear package separation, CLI UX is polished, and core flows are readable. There is still a fair amount of duplicated parsing/formatting logic and a few stringly-typed/configuration patterns that make changes riskier than they need to be.

## Highest-Impact Opportunities
1) Centralize stream parsing and tool-use display mapping.
- Current duplication across `pkg/runner/stream.go`, `pkg/orchestrator/live_display.go`, and `pkg/executor/tool.go` (tool-use name/icon mapping, stream-json parsing, token usage extraction, session ID parsing).
- Risk: small format changes in one tool require edits in multiple places; inconsistent output formatting across modes.
- Suggestion: create a small `pkg/stream` or `pkg/output` package with a shared decoder + formatter used by both runner and orchestrator.

2) Consolidate output/banner/summary rendering and ANSI colors.
- `pkg/runner/output.go` and `pkg/runner/runner.go` both render summaries; color constants are separately defined in `pkg/runner/config.go`, `pkg/orchestrator/progress.go`, and `pkg/tracking/codex.go`.
- Risk: inconsistent styling and duplicated logic for changes like new fields or formatting rules.
- Suggestion: one shared terminal-style module (colors + helpers) and one summary renderer.

3) Thread-safety for context results in vote path.
- `pkg/executor/vote.go` reads `ctx.StepResults` without locks while `pkg/executor/parallel.go` writes in goroutines via `ctx.SetResult`.
- Risk: data race if vote steps ever run after or alongside parallel steps.
- Suggestion: add `Context.GetResult(name)` and use it everywhere; avoid direct map access.

4) Bundle-specific behavior is embedded in orchestrator core.
- `pkg/orchestrator/orchestrator.go` contains special handling for article and build bundles with hard-coded step names and output conventions.
- Risk: orchestrator grows as bundle types increase; harder to reason about bundle behavior in isolation.
- Suggestion: move these into bundle-specific handlers or introduce a per-bundle post-processing hook.

5) Duplicate grade parsing and report filename parsing across Go and dashboard.
- `pkg/runner/grades.go` and `dashboard/src/app/api/repos/route.ts` both parse grades/report filenames with subtly different regexes.
- Risk: mismatched parsing logic causes missing grades or incorrect grouping when names change.
- Suggestion: define a single parsing spec (shared tests or generated source) and align both implementations. The Go regex currently only allows `[a-z]+` for tasks, which could fail for custom tasks with hyphens or digits.

## Additional Maintainability Notes
- `pkg/runner/runner.go` uses stringly-typed `FlagDef.Target` and a package-level `noTrackStatus` flag, which increases the chance of mistakes when adding flags. Consider typed flag registration or a small interface to bind flags directly to config fields.
- `pkg/orchestrator/live_display.go` re-scans full log files every 100ms to find the last meaningful line. This is O(n) per tick and can become expensive for long logs; a tailing approach with file offsets would be simpler and more efficient.
- `pkg/bundle/loader.go` validates bundle names but does not validate step schemas (missing tool/task, duplicate step names, invalid merge/vote configs). A small validation pass during load would prevent runtime surprises.

## Quick Wins (Low Effort)
- Add `Context.GetResult` and use it in `pkg/executor/vote.go` and any other direct map access.
- Reuse one summary renderer and remove `printDetailedSummary` duplication.
- Extract the tool-use display mapping into a shared helper used by both `stream.go` and `live_display.go`.

## Summary
The codebase is generally well-structured, but a few high-leverage refactors would reduce duplication, eliminate a concurrency footgun, and make future feature work (new tools/bundles) safer and faster.
