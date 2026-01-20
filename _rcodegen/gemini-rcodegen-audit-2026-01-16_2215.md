# rcodegen Codebase Audit Report
Date Created: Friday, January 16, 2026 22:15:00
TOTAL_SCORE: 78/100

## Executive Summary
The `rcodegen` project is a robust and well-architected Go-based framework for orchestrating LLM-driven tasks across multiple codebases. It features a modular design, consistent concurrency control, and a thoughtful CLI experience. However, a critical path traversal vulnerability was found in the Next.js dashboard API, along with several consistency and security-related improvements that should be addressed.

## 1. Security Audit

### 1.1 Path Traversal in Dashboard API (CRITICAL)
The Next.js API routes responsible for fetching reports do not sanitize the `name` and `file` parameters. This allows an attacker to read arbitrary files on the system that the user running the dashboard has access to.

**Vulnerable Code:** `dashboard/src/app/api/repos/[name]/reports/[file]/route.ts`
```typescript
const { name, file } = await params
const filePath = path.join(CODE_DIR, name, '_rcodegen', file)
```

**Exploit Example:**
Setting `name` to `..` and `file` to `../../.ssh/id_rsa` resolves to `/Users/user/.ssh/id_rsa`.

**Fix:**
Sanitize inputs using `path.basename()` or validate that the resulting path is within the intended directory.

### 1.2 Unauthenticated Dashboard API (HIGH)
The Next.js dashboard API provides full read access to all codebases and reports without any authentication. While this may be acceptable for local-only use, it poses a significant risk if the dashboard is ever exposed to a network.

**Recommendation:**
Implement a simple authentication mechanism or restrict the dashboard to bind only to `localhost`.

### 1.3 Risky Claude Code Flags (MEDIUM)
The `rclaude` tool uses the `--dangerously-skip-permissions` flag by default for automation.
```go
args = []string{
    "-p", task,
    "--dangerously-skip-permissions",
    // ...
}
```
While necessary for unattended operation, this bypasses all safety prompts. If a malicious task or a compromised codebase is audited, the LLM could execute harmful commands with full user permissions.

**Recommendation:**
Ensure users are prominently warned of this risk (currently implemented as a banner, which is good).

---

## 2. Code Quality & Consistency

### 2.1 Hardcoded `CODE_DIR` in Dashboard (MEDIUM)
The dashboard hardcodes the code directory to `~/Desktop/_code`, whereas the Go backend uses a configurable `code_dir` in `settings.json`. This will lead to broken functionality if the user changes their configuration.

**Fix:**
The dashboard API should read `~/.rcodegen/settings.json` to determine the correct `code_dir`.

### 2.2 Scheduler Tool Support (LOW)
The `scheduler/index.js` is currently limited to running `rcodex`. It should be extended to support `rclaude` and `rgemini`.

---

## 3. Patch-Ready Diffs

### Patch 1: Fix Path Traversal in Dashboard API
This patch sanitizes the `name` and `file` parameters to prevent path traversal attacks.

```diff
--- a/dashboard/src/app/api/repos/[name]/reports/[file]/route.ts
+++ b/dashboard/src/app/api/repos/[name]/reports/[file]/route.ts
@@ -10,7 +10,12 @@
   { params }: { params: Promise<{ name: string; file: string }> }
 ) {
   const { name, file } = await params
-  const filePath = path.join(CODE_DIR, name, '_rcodegen', file)
+
+  // Sanitize inputs to prevent path traversal
+  const safeName = path.basename(name)
+  const safeFile = path.basename(file)
+
+  const filePath = path.join(CODE_DIR, safeName, '_rcodegen', safeFile)
 
   if (!fs.existsSync(filePath)) {
     return NextResponse.json({ error: 'Report not found' }, { status: 404 })
```

```diff
--- a/dashboard/src/app/api/repos/[name]/route.ts
+++ b/dashboard/src/app/api/repos/[name]/route.ts
@@ -43,7 +43,10 @@
   { params }: { params: Promise<{ name: string }> }
 ) {
   const { name } = await params
-  const repoPath = path.join(CODE_DIR, name)
+
+  // Sanitize input to prevent path traversal
+  const safeName = path.basename(name)
+  const repoPath = path.join(CODE_DIR, safeName)
   const rcodgenDir = path.join(repoPath, '_rcodegen')
 
   if (!fs.existsSync(rcodgenDir)) {
```

### Patch 2: Read `CODE_DIR` from settings.json in Dashboard
This patch ensures the dashboard uses the same code directory as the Go tools.

```diff
--- a/dashboard/src/app/api/repos/route.ts
+++ b/dashboard/src/app/api/repos/route.ts
@@ -3,7 +3,21 @@
 import path from 'path'
 import os from 'os'
 
-const CODE_DIR = path.join(os.homedir(), 'Desktop/_code')
+function getCodeDir(): string {
+  const settingsPath = path.join(os.homedir(), '.rcodegen', 'settings.json')
+  if (fs.existsSync(settingsPath)) {
+    try {
+      const settings = JSON.parse(fs.readFileSync(settingsPath, 'utf-8'))
+      if (settings.code_dir) {
+        return settings.code_dir.replace(/^~/, os.homedir())
+      }
+    } catch (e) {
+      console.error('Error reading settings.json:', e)
+    }
+  }
+  return path.join(os.homedir(), 'Desktop/_code') // Fallback
+}
+
+const CODE_DIR = getCodeDir()
 
 const PRIMARY_TASKS = ['audit', 'test', 'fix', 'refactor'] as const
```
*(Note: Similar change should be applied to all dashboard API routes using `CODE_DIR`)*

---

## 4. Final Verdict
The project demonstrates high technical proficiency. Addressing the path traversal vulnerability and unifying configuration should be the immediate priorities. The use of LLM automation with dangerous flags is a core feature, but it must be managed with extreme care.

**Audit Status: PASS (with critical security findings)**
