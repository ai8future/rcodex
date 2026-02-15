# Design: --list _suite Fallback Scanning & --dir-all Flag

## Problem

Repos have been moved into `*_suite/` folders (e.g. `serp_svc` now lives at `~/Desktop/_code/serp_suite/serp_svc`). The `--list` flag only checks direct children of the base directory, so it can no longer find these repos.

Additionally, there's no way to say "run against all repos in this directory" without manually listing them.

## Change 1: --list with _suite fallback

**Current behavior:** `--list name` resolves to `baseDir/name` or errors.

**New behavior:**
1. Check `baseDir/name` — if exists, use it (top-level priority)
2. If not found, scan `baseDir` for `*_suite/` subdirectories
3. Check each `*_suite/name` — use first match
4. If still not found, error as before

The `_suite` directory listing is cached across names so it's only scanned once.

## Change 2: --dir-all flag

**Flag:** `--dir-all <path>` (comma-separated paths supported)

**Behavior:**
1. Parse comma-separated paths
2. For each path, discover git repos one level deep (reuse `discoverDirectories(path, 1)`)
3. Collect all found repos as work dirs
4. Mutually exclusive with `--list`, `--recursive`, and `-d`/`-c`

**Example:**
```bash
rclaude --dir-all ~/Desktop/_code/serp_suite suite
rclaude --dir-all ~/Desktop/_code/serp_suite,~/Desktop/_code/ai_suite audit
```

## Files Changed

- `pkg/runner/config.go` — Add `DirAll string` field
- `pkg/runner/runner.go` — Flag definition, mutual exclusivity, --dir-all handling, --list fallback logic
