Date Created: Saturday, January 17, 2026 at 00:20:00 AM EST
TOTAL_SCORE: 85/100

# Refactoring Report: rcodegen

## Executive Summary

The `rcodegen` codebase is a robust, well-structured Go application that effectively manages multi-tool AI workflows. It follows standard Go project layout conventions (`cmd/`, `pkg/`) and successfully integrates with a Node.js scheduler and Next.js dashboard. The code is functional, feature-rich, and provides a polished CLI experience.

However, as the project has grown, the core orchestration logic has accumulated technical debt. The `Orchestrator` and `Runner` components have become "God Classes," handling concerns ranging from execution logic to UI rendering and report generation. There is significant duplication in text processing logic and tight coupling between the core engine and specific bundle types (e.g., "article").

## Detailed Analysis

### 1. The "God Class" Problem
*   **`pkg/orchestrator/orchestrator.go` (900+ lines)**: This file is doing too much. It handles:
    *   Tool initialization.
    *   Step execution flow.
    *   Conditional logic evaluation.
    *   Cost tracking.
    *   *Specific* report generation for "article" bundles (Markdown table building).
    *   Output file scanning.
    *   Text extraction from generated files.
*   **`pkg/runner/runner.go` (600+ lines)**: Similarly, this handles:
    *   CLI flag parsing.
    *   Configuration loading.
    *   Command execution.
    *   Output formatting (JSON vs Stream vs Text).
    *   Lock management.
    *   Summary printing.

### 2. Tight Coupling & Open/Closed Principle Violations
The `Orchestrator` contains hardcoded logic that makes extending the system difficult without modifying core files:
*   **Bundle Types**: The `Run` method explicitly checks `if strings.HasPrefix(b.Name, "article")` to trigger specific reporting logic. Adding a "video" or "code-review" bundle type would require adding more `if` statements here.
*   **Tool/Model Specifics**: Hardcoded strings like `"opus"` and `"gemini-3-flash-preview"` and tool-specific overrides (`opusOnly`, `flashOnly` flags) are baked into the orchestrator rather than being properties of the `Tool` interface or configuration.

### 3. Duplication in Text Processing
There is a significant amount of copy-pasted or slightly modified logic for extracting metadata from Markdown files in `orchestrator.go`:
*   `extractOpeningSummary`, `extractAngle`, `extractDataPoint`, `extractTone`, etc., all share similar file reading and string manipulation patterns.
*   Directory scanning logic (`findArticleFilesInDir`, `scanOutputFiles`) is partially duplicated.

### 4. Mixed Concerns (Logic vs. Presentation)
*   ANSI color codes and formatting logic are scattered throughout `runner.go` and `orchestrator.go`. If the CLI styling needs to change (e.g., to support a theme system), many files would need editing.
*   HTML/Markdown generation logic is embedded directly in Go functions using string builders (`sb.WriteString`), which is error-prone and hard to read compared to using templates.

## Recommendations

### Short Term (High Impact)
1.  **Extract Report Generators**: Move `generateRunReport`, `generateFinalReportJSON`, and their helper functions (like `extract*`) into a separate `pkg/reports/generator` package. This will immediately shrink `orchestrator.go` by ~400 lines.
2.  **Consolidate Text Extraction**: Create a generic `TextExtractor` or utility functions in `pkg/textutils` to handle the repetitive file reading and string parsing logic.

### Medium Term
3.  **Refactor Bundle Handlers**: Introduce a `BundleHandler` interface. The `Orchestrator` should delegate to a handler based on the bundle type (e.g., `ArticleHandler`, `BuildHandler`). This removes the `if "article"` checks from the core loop.
4.  **Enhance Tool Interface**: Move model-specific logic (like "opus-only" overrides) into the `Tool` implementations. The orchestrator should pass constraints to the tool, and the tool should decide which model to use.

### Long Term
5.  **Use Templates**: Replace manual string building for Markdown/Reports with Go's `text/template` package. This separates the report design from the data gathering logic.
6.  **Split Runner**: Break `Runner` into `ConfigManager`, `TaskExecutor`, and `OutputFormatter` to separate concerns.

## Score Breakdown
*   **Code Organization**: 20/25 (Good directory structure, but large files)
*   **Abstraction/DRY**: 15/25 (Significant duplication and coupling)
*   **Readability**: 20/25 (Clear variable names, distinct flows, but dense)
*   **Maintainability**: 20/25 (Hard to add new bundle types without editing core)
*   **Polish**: 10/10 (Excellent CLI output and features)

**TOTAL: 85/100**
