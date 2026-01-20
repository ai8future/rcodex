Date Created: Sunday, January 18, 2026 at 10:05 PM
TOTAL_SCORE: 78/100

# rcodegen Code Quality & Refactor Report

## Executive Summary
The `rcodegen` project demonstrates a solid modular structure with a clear separation of concerns between the core logic (`pkg/`), CLI entry points (`cmd/`), and the web dashboard (`dashboard/`). The use of a shared `runner` package for different AI tool binaries is a strong architectural choice. However, as the project has grown, certain components—specifically the orchestrator and report generation logic—have become monolithic. The frontend code, while functional, suffers from type duplication and slight component bloating.

## Score Breakdown
- **Architecture & Structure (25/30):** Good package separation, but `orchestrator` is becoming a "god object".
- **Code Quality & Clarity (22/25):** Go code is generally idiomatic. Frontend code is clean but needs refactoring.
- **Maintainability (18/25):** Report generation logic is tightly coupled to the orchestrator, making it hard to add new report types without modifying core files.
- **Duplication (13/20):** CLI entry points and frontend interfaces show unnecessary duplication.

**Total: 78/100**

## Detailed Findings

### Backend (Go)

#### 1. Orchestrator Complexity & Coupling
The `pkg/orchestrator/orchestrator.go` file is over 900 lines long and handles mixed responsibilities:
- **Core Orchestration:** Running steps, dependencies, conditionals.
- **Report Generation:** `generateRunReport` and `generateFinalReportJSON` contain heavy business logic for formatting Markdown and JSON output.
- **Domain Specifics:** Explicit handling of "article" bundle types (checking string prefixes) couples the generic orchestrator to specific use cases.

**Impact:** Hard to test core orchestration logic in isolation; adding new report formats requires modifying the orchestrator.

#### 2. Report Generation Logic
The logic for generating reports (both Markdown and JSON) is embedded directly within the `orchestrator` package. Functions like `scanOutputFiles`, `extractGradeFromReport`, and `generateRunReport` are substantial and essentially form a separate domain of "Reporting".

**Impact:** Bloats the orchestrator and violates the Single Responsibility Principle.

#### 3. CLI Entry Point Duplication
The `cmd/` directory contains multiple binaries (`rclaude`, `rcodegen`, etc.). While `pkg/runner` abstracts much of the logic, the `main.go` files likely share identical boilerplate setup. For example, `cmd/rclaude/main.go` is very clean, but checking other potential binaries suggests they might just be copy-pastes with a different tool constructor.

### Frontend (Next.js)

#### 1. Type Definitions Duplication
The interfaces `RepoSummary`, `TaskGrades`, `TaskGradeInfo`, `GradeHistoryPoint`, and `TaskHistory` are defined in **both**:
- `dashboard/src/app/page.tsx`
- `dashboard/src/components/repos-columns.tsx`

**Impact:** High risk of drift between definitions. Changing the API response requires updating multiple files.

#### 2. Component Extraction (`MiniSparkline`)
The `MiniSparkline` component is a complex visualization (SVG generation, collision detection) embedded entirely within `dashboard/src/components/repos-columns.tsx`. It accounts for a significant portion of the file's complexity.

**Impact:** Reduces readability of the column definitions and makes the sparkline component harder to reuse or test.

#### 3. Default Metadata
`dashboard/src/app/layout.tsx` retains the default "Create Next App" title and description.

## Recommendations

### Short Term (High Impact, Low Effort)

1.  **Centralize Frontend Types:**
    -   Create `dashboard/src/types/index.ts`.
    -   Move `RepoSummary`, `TaskGrades`, etc., to this file.
    -   Import them in `page.tsx` and `repos-columns.tsx`.

2.  **Extract `MiniSparkline`:**
    -   Move the `MiniSparkline` component to `dashboard/src/components/ui/sparkline.tsx` (or similar).

3.  **Update Dashboard Metadata:**
    -   Update `title` and `description` in `dashboard/src/app/layout.tsx` to reflect the application's purpose.

### Long Term (Strategic Refactoring)

1.  **Extract Reporting Package:**
    -   Create a new package `pkg/reporting`.
    -   Move `generateRunReport`, `generateFinalReportJSON`, `scanOutputFiles`, and related helper functions out of `orchestrator`.
    -   Define an interface for `ReportGenerator` if multiple formats are expected.

2.  **Decouple "Article" Logic:**
    -   The special handling for "article" bundles in `orchestrator.go` should be generalized. Consider using a `BundleType` field in the bundle definition or a plugin/hook system so the orchestrator doesn't need hardcoded string checks.

3.  **Refactor `pkg/runner` Tasks:**
    -   `ReportTypes` in `pkg/runner/tasks.go` is a global variable. Consider moving this to `settings` or making it configurable so the "suite" task isn't hardcoded to a specific list of report types.
