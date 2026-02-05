Date Created: 2026-01-28 12:50:00
TOTAL_SCORE: 85/100

# rcodegen Code Audit & Refactoring Report

## Executive Summary

The `rcodegen` project is a sophisticated multi-tool orchestration platform designed to automate software engineering tasks using various AI models (Claude, Gemini, Codex). The system comprises a core Go-based CLI, a Node.js scheduler, and a Next.js dashboard.

Overall, the codebase is high-quality, demonstrating strong architectural patterns in the Go core, particularly in how it abstracts tool execution and manages state. The dashboard is modern and well-structured. The scheduler, while functional, is the least robust component.

## Detailed Analysis

### 1. Architecture & Design (Score: 25/30)

**Strengths:**
*   **Modular Go Core:** The separation between `pkg/orchestrator`, `pkg/runner`, and `pkg/bundle` is clean. The `StepExecutor` interface is a key abstraction that allows for easy testing and swapping of implementations.
*   **Bundle System:** The JSON-based bundle definition (`pkg/bundle/bundle.go`) is a powerful design choice, allowing users to define complex workflows without code changes.
*   **Envelope Pattern:** The use of an `Envelope` struct for passing results between steps provides a consistent contract for data flow.

**Weaknesses:**
*   **Coupling in Orchestrator:** The `Orchestrator.Run` method (in `pkg/orchestrator/orchestrator.go`) is monolithic (over 200 lines). It mixes orchestration logic, file I/O, display updates, and reporting. This violates the Single Responsibility Principle.
*   **Hardcoded Paths:** Several paths (e.g., `~/.rcodegen`) are hardcoded in multiple places (`pkg/orchestrator/orchestrator.go`, `scheduler/index.js`). A central configuration manager is needed.

### 2. Code Quality (Score: 25/30)

**Strengths:**
*   **Idiomatic Go:** The Go code generally follows standard conventions. Error handling is prevalent and explicit.
*   **Modern Frontend:** The Dashboard uses Next.js 13+ App Router, TypeScript, and Tailwind CSS. The component structure (e.g., `repos-columns.tsx`) is logical and reusable.
*   **Type Safety:** Both the Go core and the Dashboard (TS) benefit from strong typing, reducing runtime errors.

**Weaknesses:**
*   **Scheduler Fragility:** `scheduler/index.js` is written in plain JavaScript without type safety. It relies on a local JSON file for state, which could become corrupted. It lacks robust error recovery beyond process respawning.
*   **Global State:** The `runner` package uses package-level variables like `noTrackStatus`, which makes concurrent testing difficult.

### 3. Maintainability (Score: 20/20)

**Strengths:**
*   **Configuration-Driven:** The extensive use of JSON for configuration (bundles, settings, schedules) makes the system highly adaptable.
*   **Clean Dependencies:** `go.mod` and `package.json` files show a restrained set of dependencies, minimizing bloat and security risks.

### 4. Testing (Score: 10/20)

**Strengths:**
*   **Unit Tests:** There are `_test.go` files for core packages (`orchestrator`, `runner`), indicating a commitment to testing critical logic.

**Weaknesses:**
*   **Frontend/Scheduler Gaps:** No tests were observed for the Dashboard components or the Scheduler logic.
*   **Integration Tests:** While there are unit tests, end-to-end integration tests that simulate a full run with mocked AI providers seem absent or limited.

### 5. Documentation (Score: 5/10)

**Strengths:**
*   **READMEs:** High-level documentation exists.
*   **Help Commands:** The CLI provides detailed help output (`runner.printUsage`).

**Weaknesses:**
*   **Inline Documentation:** Complex logic in `orchestrator.go` and `runner.go` often lacks explanatory comments explaining the *why*, not just the *what*.

## Refactoring Recommendations

### Priority 1: Refactor Orchestrator.Run
Break down the huge `Run` method in `pkg/orchestrator/orchestrator.go` into smaller, testable private methods:
*   `prepareWorkspace()`
*   `executeSteps()`
*   `generateReports()`
*   `handleLiveDisplay()`

### Priority 2: Harden the Scheduler
*   Rewrite `scheduler/index.js` in TypeScript to share types with the Dashboard.
*   Move schedule state from a flat JSON file to a more robust local database (e.g., SQLite) or simply improve the JSON handling with atomic writes to prevent corruption.

### Priority 3: Centralize Configuration
Create a `pkg/config` or use the existing `pkg/settings` to centrally manage all paths and constants. Remove hardcoded strings like `.rcodegen` and `~/.rcodegen` from individual files.

### Priority 4: Frontend Tests
Add basic component tests for the Dashboard using React Testing Library, focusing on data parsing and display logic in `repos-columns.tsx`.

## Conclusion

`rcodegen` is a well-built tool with a solid architectural foundation. The core complexity is managed well through the Bundle system. The primary areas for improvement are decoupling the orchestration logic and professionalizing the scheduler component.
