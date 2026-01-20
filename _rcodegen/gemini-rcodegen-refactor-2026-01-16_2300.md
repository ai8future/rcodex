# Code Audit & Refactoring Report: rcodegen

Date Created: Friday, January 16, 2026 11:00 PM
Project: rcodegen
Auditor: Gemini CLI Agent

## 1. Introduction
This report provides a comprehensive audit of the `rcodegen` codebase, identifying opportunities for improving code quality, reducing duplication, and enhancing maintainability. The focus is on architectural improvements and structural refactoring rather than specific bug fixes.

## 2. High-Level Summary
The `rcodegen` project is well-structured as a multi-tool orchestrator, successfully abstracting AI tool execution through a unified interface. However, as the project has grown to support multiple tools (Claude, Codex, Gemini) and complex bundle workflows, several areas of technical debt and duplication have emerged. 

Key themes for refactoring include:
- **Interface Bloat:** The `Tool` interface is excessively large, leading to repetitive boilerplate in implementations.
- **Leaky Abstractions:** Tool-specific logic for parsing output (costs, session IDs) is currently residing in the execution engine instead of the tool implementations.
- **Monolithic Orchestrator:** The core orchestration logic is tightly coupled with specific bundle types (e.g., "article" bundles) and report generation formats.
- **Duplicated Utilities:** Infrastructure code for Python discovery, terminal styling, and status tracking is duplicated across multiple packages.

## 3. Key Areas for Improvement

### 3.1. Tool Interface & Base Implementation
The `runner.Tool` interface in `pkg/runner/tool.go` defines 23 methods. Many implementations (`claude.go`, `gemini.go`) return static constants or simple fields from the configuration.

**Observation:** 
- `ReportDir()` always returns `"_rcodegen"`.
- `UsesStreamOutput()` often returns `true`.
- Many "status" methods are no-ops for tools that don't support tracking.

**Recommendation:** 
Introduce a `BaseTool` struct that provides default implementations for common methods. Specific tools can then embed `BaseTool` and override only what is necessary, significantly reducing boilerplate.

### 3.2. Encapsulation of Tool Output Parsing
Currently, `pkg/executor/tool.go` contains `extractCostInfo` and `extractSessionID` functions with `switch` statements covering all supported tools.

**Observation:** 
This violates the Open/Closed Principle. Adding a new tool requires modifying the executor's core logic. The knowledge of how a tool outputs its metadata (JSON stream vs. regex on stderr) should belong to the tool implementation itself.

**Recommendation:** 
Move parsing logic into the `Tool` interface. Add methods like `ParseMetadata(stdout, stderr string) *Metadata` to allow each tool to handle its own output format.

### 3.3. Orchestrator Decomposition
`pkg/orchestrator/orchestrator.go` has grown into a monolithic coordinator handling workspace management, execution loops, condition evaluation, and multiple reporting formats (Markdown, JSON).

**Observation:** 
- Hardcoded logic for `article` bundle reporting.
- Hardcoded logic for `project_name` build reporting.
- Direct dependencies on specific tool implementations to handle model overrides (`--opus-only`, `--flash`).

**Recommendation:** 
- **Reporter Plugins:** Move Markdown and JSON report generation into a `Reporter` interface or dedicated `reports` package.
- **Lifecycle Hooks:** Allow tools or bundles to register hooks for specific events (e.g., `OnStepComplete`, `OnBundleComplete`).
- **Generalized Model Overrides:** Instead of checking for "claude" or "gemini" by name, use tool capabilities or model families.

### 3.4. Consolidation of the `tracking` Package
The `pkg/tracking` package contains `claude.go` and `codex.go` which share nearly identical code for discovering Python interpreters and executing iTerm2-based status scripts.

**Observation:** 
- `FindPython()` and `GetScriptDir()` are duplicated.
- ANSI color constants are redefined in multiple files.
- `runStatusScript` logic is duplicated.

**Recommendation:** 
Consolidate shared utilities into a `pkg/tracking/util.go` or `pkg/internal/util` package. Use a generic `ScriptRunner` to handle the execution and JSON unmarshaling of Python status scripts.

### 3.5. Dependency Management (Circular Dependencies)
The project uses a global `DispatcherFactory` variable in the `orchestrator` package, initialized by the `executor` package via `init()`, to break a circular dependency.

**Observation:** 
While effective, this pattern makes the code harder to trace and can lead to initialization order issues in tests.

**Recommendation:** 
Consider reorganizing packages to create a clear hierarchy. For example, moving the `StepExecutor` interface into a more neutral `pkg/types` or `pkg/workflow` package that both `orchestrator` and `executor` can depend on without depending on each other.

### 3.6. Redundant Cost & Token Aggregation
Logic for summing `cost_usd`, `input_tokens`, and `output_tokens` is repeated in `ParallelExecutor` and `Orchestrator`.

**Recommendation:** 
Create a `UsageStats` struct with an `Add(UsageStats)` method to centralize aggregation logic.

## 4. Specific Refactoring Tasks (Summary)

1.  **Refactor `runner.Tool`:** Add `BaseTool` and move output parsing into tool-specific files.
2.  **Modularize Reporting:** Extract `generateRunReport` and `generateFinalReportJSON` from `orchestrator.go`.
3.  **Unify Tracking Utilities:** Remove code duplication in `pkg/tracking`.
4.  **Refactor Orchestrator Flags:** Generalize model overrides (e.g., `Tool.SetModelOverride(category string)`).
5.  **Strengthen Error Handling:** In `ToolExecutor`, ensure logs are flushed and errors are wrapped with more context.

## 5. Conclusion
`rcodegen` has a strong foundation but is currently carrying duplication that will hinder the addition of future tools and workflow types. By focusing on better encapsulation in the `Tool` interface and decomposing the `Orchestrator`, the codebase will become more resilient, easier to test, and significantly more maintainable.
