Date Created: 2026-01-16 23:27:00
Date Updated: 2026-01-17
TOTAL_SCORE: 84/100

## Items Fixed (2026-01-17)
- ~~Critical: Path Traversal in Dashboard API~~ - FIXED: Added resolveWithin() validation in commits 39e239d, e8fe7c4

# Project Audit: rcodegen

## Overview
`rcodegen` is a sophisticated automation framework for AI coding assistants. It provides a unified interface for Claude, Codex, and Gemini, supporting complex multi-step workflows (bundles) and automated reporting. The codebase is primarily written in Go, with supporting Python scripts and a Next.js dashboard.

## Security Analysis

### 1. Critical: Path Traversal in Dashboard API
The Next.js dashboard contains multiple API routes that are vulnerable to path traversal. The `name` and `file` parameters from the URL are used directly in `path.join` without validation, allowing an attacker to read files outside the intended directory.

**Affected Files:**
- `dashboard/src/app/api/repos/[name]/route.ts`
- `dashboard/src/app/api/repos/[name]/reports/[file]/route.ts`

### 2. Risky Defaults: Disabled Sandboxing
The tool explicitly runs AI CLIs with flags that disable security sandboxes and permission prompts (`--dangerously-skip-permissions`, `--dangerously-bypass-approvals-and-sandbox`). While this is necessary for "unattended mode," it places complete trust in the AI's actions.

### 3. Shell Command Safety
The Go codebase correctly uses `exec.Command` with argument lists, which prevents shell injection. Python scripts also use `subprocess.Popen` with argument lists correctly.

## Code Quality & Architecture

### Strengths
- **Modular Design:** The `Tool` interface in `pkg/runner` allows for easy addition of new AI providers.
- **Robust Execution:** Good use of PTY wrappers and stream-json parsing for real-time output.
- **Concurrency Control:** Effective file-based locking prevents conflicting runs.
- **Testing:** Core packages have good unit test coverage.

### Weaknesses
- **Portability:** Features like credit tracking are tightly coupled to iTerm2 on macOS.
- **Clutter:** Several `.sync-conflict` files remain in the repository.

## Recommendations
1. **Fix Path Traversal:** Implement strict validation for repository and file names in the Dashboard API.
2. **Sanitize Dashboard Inputs:** Ensure all user-provided parameters are sanitized before file system operations.
3. **Enhance Portability:** Consider more generic ways to track costs that don't depend on specific terminal emulators.
4. **Cleanup:** Remove conflict files and update `.gitignore` to prevent future inclusion.

## Patch-Ready Diffs

### Fix Path Traversal in Repo API
```diff
--- a/dashboard/src/app/api/repos/[name]/route.ts
+++ b/dashboard/src/app/api/repos/[name]/route.ts
@@ -48,7 +48,13 @@
   { params }: { params: Promise<{ name: string }> } 
 ) {
   const { name } = await params
+
+  // Sanitize name to prevent path traversal
+  if (name.includes('..') || name.includes('/') || name.includes('\\')) {
+    return NextResponse.json({ error: 'Invalid repo name' }, { status: 400 })
+  }
+
   const repoPath = path.join(CODE_DIR, name)
   const rcodgenDir = path.join(repoPath, '_rcodegen')
```

### Fix Path Traversal in Reports API
```diff
--- a/dashboard/src/app/api/repos/[name]/reports/[file]/route.ts
+++ b/dashboard/src/app/api/repos/[name]/reports/[file]/route.ts
@@ -11,6 +11,15 @@
   { params }: { params: Promise<{ name: string; file: string }> } 
 ) {
   const { name, file } = await params
+
+  // Sanitize inputs to prevent path traversal
+  if (name.includes('..') || name.includes('/') || name.includes('\\')) {
+    return NextResponse.json({ error: 'Invalid repo name' }, { status: 400 })
+  }
+  if (file.includes('..') || file.includes('/') || file.includes('\\')) {
+    return NextResponse.json({ error: 'Invalid file name' }, { status: 400 })
+  }
+
   const filePath = path.join(CODE_DIR, name, '_rcodegen', file)
```

## Category Grades
- **Architecture & Design:** 22/25
- **Security Practices:** 12/20 (Impacted by Path Traversal)
- **Error Handling:** 14/15
- **Testing:** 13/15
- **Idioms & Style:** 14/15
- **Documentation:** 9/10

**TOTAL_SCORE: 84/100**
