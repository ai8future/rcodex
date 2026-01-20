Date Created: Monday, January 19, 2026 12:00:00 PM

# rcodegen System Study

## 1. Introduction

`rcodegen` is a sophisticated, monorepo-based automation framework designed to orchestrate and unify the usage of multiple AI coding assistants: **Claude Code**, **OpenAI Codex**, and **Google Gemini**.

The system moves beyond simple CLI wrappers to provide a standardized layer for:
- **Task Automation:** Defining reusable prompts ("shortcuts") for common engineering tasks (Audit, Test, Fix, Refactor).
- **Report Management:** Standardizing output formats, file naming, and version control (deleting old reports).
- **Workflow Orchestration:** Chaining multiple AI steps into complex "bundles" (e.g., generating articles with comparative analysis, or multi-stage build processes).
- **Unattended Execution:** Enabling "human-out-of-the-loop" operations via flags like `--yolo` and `--dangerously-skip-permissions`.

## 2. System Architecture

The project is structured as a Go-based core with peripheral integrations.

### 2.1 Core Components (`pkg/`)

*   **`pkg/runner` (The Framework):**
    This is the shared engine used by all individual tool binaries (`rclaude`, `rcodex`, `rgemini`). It handles:
    *   **Configuration:** Loading `~/.rcodegen/settings.json`.
    *   **Context:** resolving codebases (mapping `-c project` to `~/code/project`).
    *   **Execution:** running the underlying CLI tools.
    *   **Reporting:** Managing `_rcodegen` directories, persisting grades to `.grades.json`, and cleaning up old files.
    *   **Logging:** generating `.runlog` files with token usage and cost data.

*   **`pkg/tools` (The Adapters):**
    Implements the `runner.Tool` interface for each backend.
    *   **Gemini:** Wraps the `gemini` CLI. Notable for using `--yolo` (auto-approve) and `--output-format stream-json`.
    *   **Claude/Codex:** wrappers for their respective CLIs (implied existence).

*   **`pkg/orchestrator` (The Brain):**
    A higher-level system invoked by the `rcodegen` binary. It executes "bundles" (JSON workflows).
    *   **Capabilities:** Conditional execution (`if/then/else`), parallel steps, and extensive context management.
    *   **Specialization:** It contains specific logic for "Article" generation (`generateRunReport`), creating comparative tables between models (e.g., Codex vs. Gemini) analyzing tone, angle, and word count.

### 2.2 Entry Points (`cmd/`)

*   **`cmd/rgemini`, `cmd/rclaude`, `cmd/rcodex`:**
    Lightweight entry points that instantiate the specific `Tool` and pass it to `runner.NewRunner()`. This ensures all tools behave identically regarding flags (`-c`, `-d`, `-t`, `-D`, `-R`).

*   **`cmd/rcodegen`:**
    The orchestrator binary. Instead of running a single prompt, it runs a "Bundle" (workflow).

### 2.3 Peripheral Services

*   **Dashboard (`dashboard/`):**
    A Next.js application (React, Tailwind) designed to visualize the artifacts produced by the tools. It likely consumes the `.grades.json` and report files to show project health over time.
*   **Scheduler (`scheduler/`):**
    A Node.js daemon using `cron` to execute `rcodex` tasks on a schedule. This reinforces the "unattended" design philosophy—letting AI work on codebases periodically without human intervention.

## 3. Workflows & Interactions

### 3.1 The Standard Run
Command: `rgemini -c myproject audit`

1.  **Resolution:** `runner` looks up `myproject` in `code_dir` (from settings).
2.  **Configuration:** Loads the `audit` task from `settings.json`.
3.  **Substitution:** Replaces `{report_dir}` with `_rcodegen` and `{report_file}` with the timestamped filename.
4.  **Execution:** Invokes `gemini -p "..." --yolo`.
5.  **Output:** Streams JSON output, parsing it to display progress and track token usage.
6.  **Persistence:** Saves the markdown report. If `grade` task, extracts score and appends to `.grades.json`.

### 3.2 The "Suite"
Command: `rgemini -c myproject suite`

This meta-task sequentially runs 5 standard analyses:
1.  `audit` (Security/Quality)
2.  `test` (Unit Tests)
3.  `fix` (Bug Fixes)
4.  `refactor` (Clean Code)
5.  `quick` (Combined summary)

This allows a user to "fully process" a repo with one command.

### 3.3 The Orchestrated Bundle
Command: `rcodegen build-article "Topic"`

1.  **Workspace:** Creates a job in `~/.rcodegen/workspace`.
2.  **Steps:** Executes defined steps (Research -> Draft -> Edit).
3.  **Parallelism:** Can run "Draft with Codex" and "Draft with Gemini" simultaneously.
4.  **Comparison:** The orchestrator specifically parses the resulting markdown files to generate a "Run Report" table comparing the two models' outputs (Cost, Tone, Data points).

## 4. Observations & Motivations

*   **Standardization over Customization:** The system enforces a specific way of working (reports in `_rcodegen`, specific naming conventions). This indicates a desire to treat AI output as structured data that can be tracked and graded over time.
*   **"YOLO" Mode:** The explicit use of `--yolo` (Gemini), `--dangerously-skip-permissions` (Claude), and bypass flags for Codex suggests the author trusts the AI agents significantly or works in sandboxed environments where speed is prioritized over granular approval.
*   **Cost Sensitivity:** There is extensive logic for tracking token usage and calculating USD costs, breaking it down by model and step. This suggests high-volume usage where costs matter.
*   **Comparative Analysis:** The `orchestrator` code reveals a strong interest in benchmarking models against each other (Codex vs. Gemini), analyzing not just code but prose quality (tone, angle).

## 5. Gemini Integration Details

The `rgemini` tool is currently a thin wrapper:
*   **Model:** Defaults to `gemini-3-pro-preview`.
*   **Capabilities:** Supports standard report generation.
*   **Missing:** Unlike Codex/Claude, it currently lacks status/credit tracking (`SupportsStatusTracking() returns false`).
*   **Flags:** Has a specific `--flash` flag to quickly switch to the cheaper `gemini-3-flash-preview` model.

## 6. Conclusion

`rcodegen` is a mature, production-grade harness for AI engineering. It transforms ephemeral chat sessions into persistent, trackable, and automated work units. It is designed for a power user who manages multiple codebases and wants to leverage multiple AI models for continuous code improvement and content generation.
