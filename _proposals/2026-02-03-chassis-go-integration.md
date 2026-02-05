# Proposal: Integrate chassis-go into rcodegen

**Date:** 2026-02-03
**Author:** Claude:Opus 4.5

---

## Summary

Integrate `github.com/ai8future/chassis-go` into the rcodegen codebase to adopt structured logging, environment-based configuration, and improved error handling patterns. Since rcodegen is a **CLI tool suite** (not an HTTP/gRPC service), only a subset of chassis-go packages apply directly. The gRPC, HTTP middleware, and health check packages are not relevant here.

---

## Applicable Packages

| Package | Applicable? | Rationale |
|---------|-------------|-----------|
| `config` | **Yes** | Replace scattered `os.Getenv` and manual JSON config loading |
| `logz` | **Yes** | Replace ad-hoc colored fmt.Printf with structured logging |
| `lifecycle` | **Partial** | Useful for orchestrator's parallel step execution and signal handling |
| `testkit` | **Yes** | Improve test setup with env helpers and test loggers |
| `httpkit` | No | rcodegen doesn't serve HTTP |
| `grpckit` | No | rcodegen doesn't serve gRPC |
| `health` | No | No health endpoint needed for CLI tools |
| `call` | No | rcodegen delegates HTTP to external CLI binaries (claude, gemini, codex) |

---

## Current State Analysis

### Configuration (pkg/settings/)
- Settings loaded from `~/.rcodegen/settings.json` via manual JSON unmarshaling
- Interactive setup wizard writes JSON config on first run
- No environment variable support for overriding settings
- 627 lines of hand-rolled config loading, validation, and setup

### Logging
- All output via `fmt.Printf` / `fmt.Fprintf` with ANSI color codes
- No structured logging anywhere
- Color constants in `pkg/colors/` used throughout
- Stream parser in `pkg/runner/stream.go` does real-time colored output

### Error Handling
- Standard Go `error` returns throughout
- Exit codes 0/1
- No structured error types or wrapping patterns

### Process Lifecycle
- No signal handling in CLI tools (rclaude, rgemini, rcodex)
- Orchestrator runs parallel steps via goroutines with `sync.WaitGroup`
- No coordinated shutdown for parallel bundle execution

---

## Proposed Changes

### Phase 1: Add dependency and structured logging

**1.1 Add chassis-go dependency**
- `go get github.com/ai8future/chassis-go`
- Import top-level package and log `chassis.Version` at startup in verbose/debug mode

**1.2 Introduce logz for internal logging**
- Create a shared logger instance in each `cmd/*/main.go`
- Pass `*slog.Logger` through the runner framework to tool implementations
- This does NOT replace the user-facing colored output (that stays as-is for UX)
- The structured logger handles: debug diagnostics, error context, execution tracing

**Files to modify:**
- `cmd/rclaude/main.go` — Initialize logger, pass to runner
- `cmd/rgemini/main.go` — Same
- `cmd/rcodex/main.go` — Same
- `cmd/rcodegen/main.go` — Same
- `pkg/runner/runner.go` — Accept `*slog.Logger` in Config, use for internal logging
- `pkg/runner/config.go` — Add Logger field
- `pkg/runner/stream.go` — Use logger for parse errors instead of silent failures
- `pkg/orchestrator/orchestrator.go` — Accept and use logger
- `pkg/tools/claude/claude.go` — Use logger for debug info
- `pkg/tools/gemini/gemini.go` — Same
- `pkg/tools/codex/codex.go` — Same

**Behavioral notes:**
- Default log level: `warn` (CLI tools should be quiet by default)
- Add `-v/--verbose` flag to set log level to `debug`
- Structured logs go to stderr, user-facing output stays on stdout
- This enables `rclaude audit myproject 2>/dev/null` to get clean output while retaining diagnostics

### Phase 2: Environment-based configuration overrides

**2.1 Add env var overrides to settings**
- Define a `EnvOverrides` struct using chassis-go `config` tags
- Allow environment variables to override settings.json values
- This enables CI/CD usage without interactive setup

```go
type EnvOverrides struct {
    CodeDir    string `env:"RCODEGEN_CODE_DIR" required:"false"`
    OutputDir  string `env:"RCODEGEN_OUTPUT_DIR" required:"false"`
    Model      string `env:"RCODEGEN_MODEL" required:"false"`
    Budget     string `env:"RCODEGEN_BUDGET" required:"false"`
    Effort     string `env:"RCODEGEN_EFFORT" required:"false"`
    LogLevel   string `env:"RCODEGEN_LOG_LEVEL" default:"warn"`
}
```

**2.2 Merge order: defaults < settings.json < env vars < CLI flags**
- Settings.json remains the primary config source
- Env vars override settings.json (for CI/CD and scripting)
- CLI flags override everything (current behavior preserved)

**Files to modify:**
- `pkg/settings/settings.go` — Add env override loading after JSON load
- `pkg/settings/settings.go` — Use `config.MustLoad` with `required:"false"` for all env fields (soft override, not hard requirement)

### Phase 3: Lifecycle for orchestrator

**3.1 Replace WaitGroup with lifecycle.Run in orchestrator**
- The orchestrator currently runs parallel steps with raw goroutines
- Replace with `lifecycle.Run` for coordinated cancellation
- If one parallel step fails, cancel remaining steps immediately
- Handle SIGTERM/SIGINT during long bundle executions

**Files to modify:**
- `pkg/orchestrator/orchestrator.go` — Use `lifecycle.Run` for parallel step execution
- `cmd/rcodegen/main.go` — Wrap top-level execution in signal-aware context

**Behavioral improvement:**
- Ctrl+C during a long suite run will cleanly stop remaining tasks
- A failing parallel step won't leave other steps running wastefully

### Phase 4: Adopt testkit in tests

**4.1 Use testkit for test infrastructure**
- Replace any manual `os.Setenv`/cleanup in tests with `testkit.SetEnv`
- Use `testkit.NewLogger` for tests that need a logger
- Use `testkit.GetFreePort` if any tests need port allocation

**Files to modify:**
- All `*_test.go` files that manipulate environment variables
- New tests written for the config override system

---

## What NOT to Change

1. **User-facing colored output** — The ANSI-colored terminal output is core UX for a CLI tool. Structured logging supplements it for diagnostics; it doesn't replace it.

2. **Settings.json as primary config** — The interactive setup wizard and JSON config are good for CLI tools. Env vars add an override layer, not a replacement.

3. **Tool execution model** — rcodegen delegates to external CLI binaries. There's no HTTP client to replace with `call`, no server to add middleware to.

4. **Bundle/orchestrator JSON format** — The bundle definition format stays as-is.

---

## Migration Risk Assessment

| Change | Risk | Mitigation |
|--------|------|------------|
| Adding logz | Low | Additive change, no existing behavior modified |
| Env config overrides | Low | All env fields are optional, settings.json still works |
| Lifecycle in orchestrator | Medium | Changes parallel execution semantics; needs thorough testing of bundle runs |
| testkit adoption | Low | Test-only changes |

---

## Estimated Scope

- **New dependency:** `github.com/ai8future/chassis-go` (pulls in `golang.org/x/sync`, which is tiny)
- **Files modified:** ~15 files
- **Files created:** 0 (all changes are in existing files)
- **Breaking changes:** None (all changes are backward-compatible)

---

## Packages NOT Adopted (and why)

- **httpkit**: rcodegen doesn't serve HTTP. No request middleware needed.
- **grpckit**: No gRPC server or client in rcodegen.
- **health**: No health endpoint for CLI tools.
- **call**: rcodegen invokes external CLI binaries (claude, gemini, codex) via `exec.Command`, not HTTP calls. The resilient HTTP client pattern doesn't apply.

---

## Verdict

Chassis-go is a good fit for the **foundation layer** of rcodegen: structured logging, environment-based config overrides, and lifecycle management. The transport packages (HTTP, gRPC, health, call) don't apply since rcodegen is a CLI orchestrator, not a network service. The adoption is low-risk and additive — it improves diagnostics, CI/CD support, and shutdown behavior without changing existing user-facing behavior.
