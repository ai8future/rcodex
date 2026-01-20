Date Created: 2026-01-16 23:25:38 +0100
TOTAL_SCORE: 78/100

# rcodegen Audit Report

## Scope
- Go orchestration/runner/executor/tooling, Node scheduler, and Next.js dashboard API routes.
- Focused on security, correctness, and operational safety; no tests executed.

## Score Rationale
- Solid modular structure and clear security warnings for risky flags, but the dashboard APIs are effectively open and have path traversal vectors.
- A few filesystem safety gaps and a concurrency issue reduce reliability under parallel bundles.

## Findings
### Critical
1) Path traversal in report and repo APIs allows arbitrary file reads.
   - Evidence: `dashboard/src/app/api/repos/[name]/reports/[file]/route.ts:12`, `dashboard/src/app/api/repos/[name]/route.ts:56`.
   - Impact: A crafted `name` or `file` can escape `_rcodegen` and `CODE_DIR` to read unintended files.
   - Fix: Validate segments and enforce resolved path containment (see Diff A).

2) Dashboard APIs allow schedule creation/updates without auth.
   - Evidence: `dashboard/src/app/api/schedules/route.ts:42`, `dashboard/src/app/api/schedules/[id]/route.ts:33`.
   - Impact: If the dashboard is exposed, an attacker can schedule unattended runs that execute with approvals/sandbox bypassed.
   - Fix: Add a token gate for all dashboard routes (see Diff A).

### High
3) Bundle step names can escape workspace directories when used in output/log filenames.
   - Evidence: `pkg/executor/tool.go:68`, `pkg/workspace/workspace.go:47`.
   - Impact: A malicious bundle in `~/.rcodegen/bundles` can write files outside the workspace.
   - Fix: Sanitize step names and tighten perms (see Diff B).

### Medium
4) Data race on `Context.StepResults` in vote execution.
   - Evidence: `pkg/executor/vote.go:16`.
   - Impact: Parallel steps can cause inconsistent votes or fail under the race detector.
   - Fix: Add a locked accessor on `Context` and use it (see Diff C).

5) Workspace/log directories are created with 0755.
   - Evidence: `pkg/workspace/workspace.go:39`, `pkg/executor/tool.go:69`.
   - Impact: Outputs and logs can be read by other users on shared machines.
   - Fix: Use 0700 for workspace/log directories (see Diff B).

### Low
6) Report summary labels Codex edit output as Gemini.
   - Evidence: `pkg/orchestrator/orchestrator.go:479`.
   - Impact: Misleading summary in article run reports.
   - Fix: Correct label (see Diff D).

## Patch-Ready Diffs
### Diff A: Add optional dashboard auth and block path traversal
```diff
diff --git a/dashboard/src/app/api/_auth.ts b/dashboard/src/app/api/_auth.ts
new file mode 100644
--- /dev/null
+++ b/dashboard/src/app/api/_auth.ts
@@
+import { NextResponse } from 'next/server'
+
+export function requireDashboardAuth(request: Request) {
+  const token = process.env.RCODEGEN_DASHBOARD_TOKEN
+  if (!token) {
+    return null
+  }
+  const authHeader = request.headers.get('authorization') || ''
+  const bearer = authHeader.startsWith('Bearer ') ? authHeader.slice(7) : ''
+  const apiKey = request.headers.get('x-rcodegen-token') || ''
+  if (bearer === token || apiKey === token) {
+    return null
+  }
+  return NextResponse.json({ error: 'Unauthorized' }, { status: 401 })
+}

diff --git a/dashboard/src/app/api/repos/route.ts b/dashboard/src/app/api/repos/route.ts
--- a/dashboard/src/app/api/repos/route.ts
+++ b/dashboard/src/app/api/repos/route.ts
@@
-import { NextResponse } from 'next/server'
+import { NextResponse } from 'next/server'
+import { requireDashboardAuth } from '../_auth'
@@
-const CODE_DIR = path.join(os.homedir(), 'Desktop/_code')
+const CODE_DIR = path.resolve(os.homedir(), 'Desktop/_code')
@@
-export async function GET() {
+export async function GET(request: Request) {
+  const auth = requireDashboardAuth(request)
+  if (auth) return auth
   try {
     if (!fs.existsSync(CODE_DIR)) {
       return NextResponse.json({ error: 'Code directory not found' }, { status: 404 })
     }
@@
 }

diff --git a/dashboard/src/app/api/repos/[name]/route.ts b/dashboard/src/app/api/repos/[name]/route.ts
--- a/dashboard/src/app/api/repos/[name]/route.ts
+++ b/dashboard/src/app/api/repos/[name]/route.ts
@@
-import { NextResponse } from 'next/server'
+import { NextResponse } from 'next/server'
+import { requireDashboardAuth } from '../../_auth'
@@
-const CODE_DIR = path.join(os.homedir(), 'Desktop/_code')
+const CODE_DIR = path.resolve(os.homedir(), 'Desktop/_code')
+const SAFE_SEGMENT = /^[a-zA-Z0-9._-]+$/
+
+function isSafeSegment(value: string): boolean {
+  return SAFE_SEGMENT.test(value)
+}
@@
 export async function GET(
   request: Request,
   { params }: { params: Promise<{ name: string }> }
 ) {
-  const { name } = await params
-  const repoPath = path.join(CODE_DIR, name)
+  const auth = requireDashboardAuth(request)
+  if (auth) return auth
+
+  const { name } = await params
+  if (!isSafeSegment(name)) {
+    return NextResponse.json({ error: 'Invalid repo name' }, { status: 400 })
+  }
+  const repoPath = path.resolve(CODE_DIR, name)
+  if (!repoPath.startsWith(CODE_DIR + path.sep)) {
+    return NextResponse.json({ error: 'Invalid repo path' }, { status: 400 })
+  }
   const rcodgenDir = path.join(repoPath, '_rcodegen')
@@
 }

diff --git a/dashboard/src/app/api/repos/[name]/reports/[file]/route.ts b/dashboard/src/app/api/repos/[name]/reports/[file]/route.ts
--- a/dashboard/src/app/api/repos/[name]/reports/[file]/route.ts
+++ b/dashboard/src/app/api/repos/[name]/reports/[file]/route.ts
@@
-import { NextResponse } from 'next/server'
+import { NextResponse } from 'next/server'
+import { requireDashboardAuth } from '../../../_auth'
@@
-const CODE_DIR = path.join(os.homedir(), 'Desktop/_code')
+const CODE_DIR = path.resolve(os.homedir(), 'Desktop/_code')
+const SAFE_SEGMENT = /^[a-zA-Z0-9._-]+$/
+const SAFE_REPORT = /^[a-zA-Z0-9._-]+\.md$/
+
+function isSafeSegment(value: string): boolean {
+  return SAFE_SEGMENT.test(value)
+}
@@
 export async function GET(
   request: Request,
   { params }: { params: Promise<{ name: string; file: string }> }
 ) {
-  const { name, file } = await params
-  const filePath = path.join(CODE_DIR, name, '_rcodegen', file)
+  const auth = requireDashboardAuth(request)
+  if (auth) return auth
+
+  const { name, file } = await params
+  if (!isSafeSegment(name) || !SAFE_REPORT.test(file)) {
+    return NextResponse.json({ error: 'Invalid report path' }, { status: 400 })
+  }
+  const repoPath = path.resolve(CODE_DIR, name)
+  if (!repoPath.startsWith(CODE_DIR + path.sep)) {
+    return NextResponse.json({ error: 'Invalid repo path' }, { status: 400 })
+  }
+  const reportDir = path.resolve(repoPath, '_rcodegen')
+  const filePath = path.resolve(reportDir, file)
+  if (!filePath.startsWith(reportDir + path.sep)) {
+    return NextResponse.json({ error: 'Invalid report path' }, { status: 400 })
+  }
@@
 }

diff --git a/dashboard/src/app/api/schedules/route.ts b/dashboard/src/app/api/schedules/route.ts
--- a/dashboard/src/app/api/schedules/route.ts
+++ b/dashboard/src/app/api/schedules/route.ts
@@
-import { NextResponse } from 'next/server'
+import { NextResponse } from 'next/server'
+import { requireDashboardAuth } from '../_auth'
@@
-export async function GET() {
+export async function GET(request: Request) {
+  const auth = requireDashboardAuth(request)
+  if (auth) return auth
   try {
     const data = readSchedules()
     return NextResponse.json(data.schedules)
   } catch (error) {
@@
 export async function POST(request: Request) {
+  const auth = requireDashboardAuth(request)
+  if (auth) return auth
   try {
     const body = await request.json()
     const { repo, task, cron } = body
@@
 }

diff --git a/dashboard/src/app/api/schedules/[id]/route.ts b/dashboard/src/app/api/schedules/[id]/route.ts
--- a/dashboard/src/app/api/schedules/[id]/route.ts
+++ b/dashboard/src/app/api/schedules/[id]/route.ts
@@
-import { NextResponse } from 'next/server'
+import { NextResponse } from 'next/server'
+import { requireDashboardAuth } from '../../_auth'
@@
 export async function PATCH(
   request: Request,
   { params }: { params: Promise<{ id: string }> }
 ) {
+  const auth = requireDashboardAuth(request)
+  if (auth) return auth
   try {
     const { id } = await params
     const body = await request.json()
@@
 export async function DELETE(
   request: Request,
   { params }: { params: Promise<{ id: string }> }
 ) {
+  const auth = requireDashboardAuth(request)
+  if (auth) return auth
   try {
     const { id } = await params
     const data = readSchedules()
@@
 }

diff --git a/dashboard/src/app/api/daemon/status/route.ts b/dashboard/src/app/api/daemon/status/route.ts
--- a/dashboard/src/app/api/daemon/status/route.ts
+++ b/dashboard/src/app/api/daemon/status/route.ts
@@
-import { NextResponse } from 'next/server'
+import { NextResponse } from 'next/server'
+import { requireDashboardAuth } from '../../_auth'
@@
-export async function GET() {
+export async function GET(request: Request) {
+  const auth = requireDashboardAuth(request)
+  if (auth) return auth
   try {
     if (!fs.existsSync(STATUS_FILE)) {
       return NextResponse.json({
         running: false,
@@
 }
```

### Diff B: Sanitize step names and tighten workspace permissions
```diff
diff --git a/pkg/workspace/workspace.go b/pkg/workspace/workspace.go
--- a/pkg/workspace/workspace.go
+++ b/pkg/workspace/workspace.go
@@
 import (
 	"crypto/rand"
 	"encoding/hex"
 	"encoding/json"
 	"fmt"
 	"os"
 	"path/filepath"
+	"strings"
 	"time"
 )
@@
 	for _, dir := range dirs {
-		if err := os.MkdirAll(dir, 0755); err != nil {
+		if err := os.MkdirAll(dir, 0700); err != nil {
 			return nil, err
 		}
 	}
@@
 func (w *Workspace) OutputPath(stepName string) string {
-	return filepath.Join(w.JobDir, "outputs", stepName+".json")
+	safeName := SafeStepName(stepName)
+	return filepath.Join(w.JobDir, "outputs", safeName+".json")
 }
+
+const maxStepNameLen = 80
+
+// SafeStepName returns a filesystem-safe step name for output/log files.
+func SafeStepName(name string) string {
+	name = strings.TrimSpace(name)
+	if name == "" {
+		return "step"
+	}
+	name = filepath.Base(name)
+	var b strings.Builder
+	b.Grow(len(name))
+	for _, r := range name {
+		switch {
+		case r >= 'a' && r <= 'z',
+			r >= 'A' && r <= 'Z',
+			r >= '0' && r <= '9',
+			r == '-', r == '_', r == '.':
+			b.WriteRune(r)
+		default:
+			b.WriteByte('_')
+		}
+	}
+	safe := strings.Trim(b.String(), "._-")
+	if safe == "" {
+		return "step"
+	}
+	if len(safe) > maxStepNameLen {
+		safe = safe[:maxStepNameLen]
+	}
+	return safe
+}

diff --git a/pkg/executor/tool.go b/pkg/executor/tool.go
--- a/pkg/executor/tool.go
+++ b/pkg/executor/tool.go
@@
 	// Create log file for real-time output
 	logDir := filepath.Join(ws.JobDir, "logs")
-	os.MkdirAll(logDir, 0755)
-	logPath := filepath.Join(logDir, step.Name+".log")
+	os.MkdirAll(logDir, 0700)
+	safeName := workspace.SafeStepName(step.Name)
+	logPath := filepath.Join(logDir, safeName+".log")
 	logFile, logErr := os.Create(logPath)
```

### Diff C: Avoid data race in vote executor
```diff
diff --git a/pkg/orchestrator/context.go b/pkg/orchestrator/context.go
--- a/pkg/orchestrator/context.go
+++ b/pkg/orchestrator/context.go
@@
 func (c *Context) SetResult(name string, env *envelope.Envelope) {
 	c.mu.Lock()
 	defer c.mu.Unlock()
 	c.StepResults[name] = env
 }
+
+// GetResult returns a step result with a read lock for safe concurrent access.
+func (c *Context) GetResult(name string) (*envelope.Envelope, bool) {
+	c.mu.RLock()
+	defer c.mu.RUnlock()
+	env, ok := c.StepResults[name]
+	return env, ok
+}

diff --git a/pkg/executor/vote.go b/pkg/executor/vote.go
--- a/pkg/executor/vote.go
+++ b/pkg/executor/vote.go
@@
 	for _, inputRef := range step.Vote.Inputs {
 		// Extract step name from ${steps.name.output_ref}
 		// For now, just count successful steps
 		stepName := extractStepName(inputRef)
-		if env, ok := ctx.StepResults[stepName]; ok {
+		if env, ok := ctx.GetResult(stepName); ok && env != nil {
 			if env.Status == envelope.StatusSuccess {
 				votes["success"]++
 			} else {
```

### Diff D: Fix report label
```diff
diff --git a/pkg/orchestrator/orchestrator.go b/pkg/orchestrator/orchestrator.go
--- a/pkg/orchestrator/orchestrator.go
+++ b/pkg/orchestrator/orchestrator.go
@@
-				expanded = append(expanded, ExpandedStep{stepNum, "Edit", "Gemini", "✓", codexEditCost, codexOut})
+				expanded = append(expanded, ExpandedStep{stepNum, "Edit", "Codex", "✓", codexEditCost, codexOut})
```

## Additional Observations
- The CLI wrappers intentionally run with `--dangerously-*` flags (see `pkg/tools/codex/codex.go:286`, `pkg/tools/claude/claude.go:104`, `pkg/tools/gemini/gemini.go:76`). This is reasonable for unattended automation but magnifies the impact of any exposed API surface. Consider an explicit safe-mode toggle for dashboard-driven runs.

## Testing
- Not run (audit-only).
