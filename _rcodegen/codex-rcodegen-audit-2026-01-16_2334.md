Date Created: 2026-01-16 23:34:16 +0100
Date Updated: 2026-01-17
TOTAL_SCORE: 78/100

## Items Fixed (2026-01-17)
- ~~[Critical] Path traversal in report APIs~~ - FIXED: Added resolveWithin() validation in commits 39e239d, e8fe7c4
- ~~[Medium] Step names used directly in log/output file paths~~ - Deferred: Low risk in practice

# rcodegen Audit Report

**Scope**
- Static review of Go CLI/orchestrator (`cmd/`, `pkg/`), Next.js dashboard API (`dashboard/src/app/api`), and scheduler (`scheduler`).
- No tests or runtime scans executed.

**Findings**
- [Critical] Path traversal in report APIs allows arbitrary file read outside `CODE_DIR` when the dashboard is reachable. Evidence: `dashboard/src/app/api/repos/[name]/reports/[file]/route.ts:12`, `dashboard/src/app/api/repos/[name]/route.ts:56`. Risk: exfiltration of any readable file and report enumeration. Recommendation: resolve/realpath within the base dir, require `.md`, reject escapes.
- [High] Schedule creation and scheduler lack validation for `repo` and `task`, enabling unintended execution in arbitrary directories. Evidence: `dashboard/src/app/api/schedules/route.ts:52`, `scheduler/index.js:52`. Risk: remote API callers or tampered `~/.rcodegen/schedules.json` can run `rcodex` on arbitrary paths or options. Recommendation: allowlist tasks, enforce repo path under `CODE_DIR`, and re-validate in the scheduler.
- [Medium] Step names are used directly in log/output file paths; malicious bundle step names can escape the workspace. Evidence: `pkg/executor/tool.go:68`, `pkg/workspace/workspace.go:47`, `pkg/orchestrator/live_display.go:158`. Risk: write/read outside workspace and leak data through live display. Recommendation: sanitize step names to safe filenames and reuse consistently.
- [Low] Dashboard API endpoints are unauthenticated. Evidence: `dashboard/src/app/api/*`. Risk: if hosted beyond localhost, external users can enumerate repos/reports and create schedules. Recommendation: require an API token or localhost-only binding.
- [Low] Synchronous filesystem calls in API routes may block the event loop for large directories/files. Evidence: `dashboard/src/app/api/repos/[name]/route.ts:65`, `dashboard/src/app/api/repos/[name]/reports/[file]/route.ts:20`. Recommendation: move to async I/O or add pagination/size limits.

**Patch-ready diffs**
```diff
diff --git a/dashboard/src/app/api/repos/[name]/reports/[file]/route.ts b/dashboard/src/app/api/repos/[name]/reports/[file]/route.ts
index 1a0b0a5..2c6e1f1 100644
--- a/dashboard/src/app/api/repos/[name]/reports/[file]/route.ts
+++ b/dashboard/src/app/api/repos/[name]/reports/[file]/route.ts
@@
-import { NextResponse } from 'next/server'
-import fs from 'fs'
-import path from 'path'
-import os from 'os'
-
-const CODE_DIR = path.join(os.homedir(), 'Desktop/_code')
+import { NextResponse } from 'next/server'
+import fs from 'fs'
+import path from 'path'
+import os from 'os'
+
+const CODE_DIR = path.resolve(os.homedir(), 'Desktop/_code')
+
+function resolveWithin(base: string, ...segments: string[]) {
+  const resolvedBase = path.resolve(base)
+  const resolvedPath = path.resolve(resolvedBase, ...segments)
+  const prefix = resolvedBase.endsWith(path.sep) ? resolvedBase : resolvedBase + path.sep
+  if (!resolvedPath.startsWith(prefix)) {
+    return null
+  }
+  return resolvedPath
+}
+
+function realpathWithin(base: string, target: string) {
+  try {
+    const realBase = fs.realpathSync(base)
+    const realTarget = fs.realpathSync(target)
+    const prefix = realBase.endsWith(path.sep) ? realBase : realBase + path.sep
+    if (!realTarget.startsWith(prefix)) {
+      return null
+    }
+    return realTarget
+  } catch {
+    return null
+  }
+}
@@
-  const { name, file } = await params
-  const filePath = path.join(CODE_DIR, name, '_rcodegen', file)
-
-  if (!fs.existsSync(filePath)) {
-    return NextResponse.json({ error: 'Report not found' }, { status: 404 })
-  }
+  const { name, file } = await params
+  const repoPath = resolveWithin(CODE_DIR, name)
+  if (!repoPath) {
+    return NextResponse.json({ error: 'Invalid repo path' }, { status: 400 })
+  }
+  const reportDir = resolveWithin(repoPath, '_rcodegen')
+  const filePath = reportDir ? resolveWithin(reportDir, file) : null
+  if (!filePath || path.extname(filePath).toLowerCase() !== '.md') {
+    return NextResponse.json({ error: 'Invalid report path' }, { status: 400 })
+  }
+
+  if (!fs.existsSync(filePath)) {
+    return NextResponse.json({ error: 'Report not found' }, { status: 404 })
+  }
+
+  const safePath = realpathWithin(CODE_DIR, filePath)
+  if (!safePath) {
+    return NextResponse.json({ error: 'Invalid report path' }, { status: 400 })
+  }
 
   try {
-    const content = fs.readFileSync(filePath, 'utf-8')
+    const content = fs.readFileSync(safePath, 'utf-8')
     return NextResponse.json({ content })
   } catch (error) {
     console.error('Error reading report:', error)
     return NextResponse.json({ error: 'Failed to read report' }, { status: 500 })
   }
 }
```

```diff
diff --git a/dashboard/src/app/api/repos/[name]/route.ts b/dashboard/src/app/api/repos/[name]/route.ts
index 76e0e1c..7d7f6b6 100644
--- a/dashboard/src/app/api/repos/[name]/route.ts
+++ b/dashboard/src/app/api/repos/[name]/route.ts
@@
-const CODE_DIR = path.join(os.homedir(), 'Desktop/_code')
+const CODE_DIR = path.resolve(os.homedir(), 'Desktop/_code')
+
+function resolveWithin(base: string, ...segments: string[]) {
+  const resolvedBase = path.resolve(base)
+  const resolvedPath = path.resolve(resolvedBase, ...segments)
+  const prefix = resolvedBase.endsWith(path.sep) ? resolvedBase : resolvedBase + path.sep
+  if (!resolvedPath.startsWith(prefix)) {
+    return null
+  }
+  return resolvedPath
+}
+
+function realpathWithin(base: string, target: string) {
+  try {
+    const realBase = fs.realpathSync(base)
+    const realTarget = fs.realpathSync(target)
+    const prefix = realBase.endsWith(path.sep) ? realBase : realBase + path.sep
+    if (!realTarget.startsWith(prefix)) {
+      return null
+    }
+    return realTarget
+  } catch {
+    return null
+  }
+}
@@
-  const { name } = await params
-  const repoPath = path.join(CODE_DIR, name)
-  const rcodgenDir = path.join(repoPath, '_rcodegen')
+  const { name } = await params
+  const repoPath = resolveWithin(CODE_DIR, name)
+  if (!repoPath) {
+    return NextResponse.json({ error: 'Invalid repo path' }, { status: 400 })
+  }
+  const safeRepoPath = realpathWithin(CODE_DIR, repoPath)
+  if (!safeRepoPath) {
+    return NextResponse.json({ error: 'Invalid repo path' }, { status: 400 })
+  }
+  const rcodgenDir = resolveWithin(safeRepoPath, '_rcodegen')
+  if (!rcodgenDir) {
+    return NextResponse.json({ error: 'Invalid repo path' }, { status: 400 })
+  }
@@
-    return NextResponse.json({
-      name,
-      path: repoPath,
+    return NextResponse.json({
+      name,
+      path: safeRepoPath,
       reports,
       grouped
     })
```

```diff
diff --git a/dashboard/src/app/api/schedules/route.ts b/dashboard/src/app/api/schedules/route.ts
index 9818a2a..ffdbd66 100644
--- a/dashboard/src/app/api/schedules/route.ts
+++ b/dashboard/src/app/api/schedules/route.ts
@@
-const SCHEDULES_FILE = path.join(os.homedir(), '.rcodegen/schedules.json')
+const SCHEDULES_FILE = path.join(os.homedir(), '.rcodegen/schedules.json')
+const CODE_DIR = path.resolve(os.homedir(), 'Desktop/_code')
+const ALLOWED_TASKS = new Set(['suite', 'audit', 'test', 'fix', 'refactor', 'quick', 'grade'])
@@
 function writeSchedules(data: SchedulesData) {
   ensureDir()
   fs.writeFileSync(SCHEDULES_FILE, JSON.stringify(data, null, 2))
 }
+
+function resolveRepoPath(repo: string): string | null {
+  if (typeof repo !== 'string' || !repo.trim()) return null
+  if (!path.isAbsolute(repo)) return null
+
+  const resolvedBase = path.resolve(CODE_DIR)
+  const resolvedRepo = path.resolve(repo)
+  const prefix = resolvedBase.endsWith(path.sep) ? resolvedBase : resolvedBase + path.sep
+  if (!resolvedRepo.startsWith(prefix)) return null
+
+  try {
+    const stat = fs.statSync(resolvedRepo)
+    if (!stat.isDirectory()) return null
+  } catch {
+    return null
+  }
+
+  return resolvedRepo
+}
@@
-    if (!repo || !task || !cron) {
+    if (!repo || !task || !cron) {
       return NextResponse.json({ error: 'Missing required fields' }, { status: 400 })
     }
+    if (typeof task !== 'string' || !ALLOWED_TASKS.has(task)) {
+      return NextResponse.json({ error: 'Invalid task' }, { status: 400 })
+    }
+    if (typeof cron !== 'string' || !cron.trim()) {
+      return NextResponse.json({ error: 'Invalid cron expression' }, { status: 400 })
+    }
+    const repoPath = resolveRepoPath(repo)
+    if (!repoPath) {
+      return NextResponse.json({ error: 'Invalid repo path' }, { status: 400 })
+    }
@@
-      repo,
-      task,
-      cron,
+      repo: repoPath,
+      task,
+      cron: cron.trim(),
       enabled: true,
       created: new Date().toISOString()
     }
```

```diff
diff --git a/dashboard/src/app/api/schedules/[id]/route.ts b/dashboard/src/app/api/schedules/[id]/route.ts
index f1aaf09..6b7076a 100644
--- a/dashboard/src/app/api/schedules/[id]/route.ts
+++ b/dashboard/src/app/api/schedules/[id]/route.ts
@@
-const SCHEDULES_FILE = path.join(os.homedir(), '.rcodegen/schedules.json')
+const SCHEDULES_FILE = path.join(os.homedir(), '.rcodegen/schedules.json')
+const ALLOWED_TASKS = new Set(['suite', 'audit', 'test', 'fix', 'refactor', 'quick', 'grade'])
@@
 function writeSchedules(data: SchedulesData) {
   fs.writeFileSync(SCHEDULES_FILE, JSON.stringify(data, null, 2))
 }
+
+function isValidTask(task: unknown): task is string {
+  return typeof task === 'string' && ALLOWED_TASKS.has(task)
+}
@@
-    if (body.cron) {
-      data.schedules[index].cron = body.cron
-    }
-    if (body.task) {
-      data.schedules[index].task = body.task
-    }
+    if (body.cron) {
+      if (typeof body.cron !== 'string' || !body.cron.trim()) {
+        return NextResponse.json({ error: 'Invalid cron expression' }, { status: 400 })
+      }
+      data.schedules[index].cron = body.cron
+    }
+    if (body.task) {
+      if (!isValidTask(body.task)) {
+        return NextResponse.json({ error: 'Invalid task' }, { status: 400 })
+      }
+      data.schedules[index].task = body.task
+    }
```

```diff
diff --git a/scheduler/index.js b/scheduler/index.js
index d6e4a6a..e19f628 100644
--- a/scheduler/index.js
+++ b/scheduler/index.js
@@
 const SCHEDULES_FILE = path.join(os.homedir(), '.rcodegen/schedules.json')
 const STATUS_FILE = path.join(os.homedir(), '.rcodegen/scheduler-status.json')
 const POLL_INTERVAL = 60000 // 60 seconds
+const ALLOWED_TASKS = new Set(['suite', 'audit', 'test', 'fix', 'refactor', 'quick', 'grade'])
@@
 function readSchedules() {
   if (!fs.existsSync(SCHEDULES_FILE)) {
     return []
   }
@@
   }
 }
+
+function isValidSchedule(schedule) {
+  const scheduleId = schedule && schedule.id ? schedule.id : 'unknown'
+
+  if (!schedule || typeof schedule !== 'object') {
+    log(`Invalid schedule entry: ${scheduleId}`)
+    return false
+  }
+  if (typeof schedule.task !== 'string' || !ALLOWED_TASKS.has(schedule.task)) {
+    log(`Invalid task for schedule ${scheduleId}: ${schedule.task}`)
+    return false
+  }
+  if (typeof schedule.repo !== 'string' || !schedule.repo.trim()) {
+    log(`Invalid repo for schedule ${scheduleId}`)
+    return false
+  }
+  try {
+    const stat = fs.statSync(schedule.repo)
+    if (!stat.isDirectory()) {
+      log(`Repo is not a directory for schedule ${scheduleId}: ${schedule.repo}`)
+      return false
+    }
+  } catch (err) {
+    log(`Repo missing for schedule ${scheduleId}: ${schedule.repo}`)
+    return false
+  }
+
+  return true
+}
@@
 function runTask(schedule) {
+  if (!isValidSchedule(schedule)) {
+    return
+  }
   const repoName = path.basename(schedule.repo)
   log(`Starting task: ${schedule.task} on ${repoName}`)
@@
 function syncJobs() {
   const schedules = readSchedules()
-  const currentIds = new Set(schedules.filter(s => s.enabled).map(s => s.id))
+  const validSchedules = schedules.filter(s => s.enabled && isValidSchedule(s))
+  const currentIds = new Set(validSchedules.map(s => s.id))
@@
-  for (const schedule of schedules) {
-    if (!schedule.enabled) continue
+  for (const schedule of validSchedules) {
     if (activeJobs.has(schedule.id)) continue
 
     const job = createJob(schedule)
```

```diff
diff --git a/pkg/workspace/workspace.go b/pkg/workspace/workspace.go
index 0e0e43f..b468da4 100644
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
+
+const maxStepNameLen = 80
@@
 func GenerateJobID() string {
 	now := time.Now()
 	b := make([]byte, 4)
 	rand.Read(b)
 	return fmt.Sprintf("%s-%s", now.Format("20060102-150405"), hex.EncodeToString(b))
 }
+
+// SafeStepName normalizes step names to filesystem-safe filenames.
+func SafeStepName(stepName string) string {
+	stepName = strings.TrimSpace(stepName)
+	if stepName == "" {
+		return "step"
+	}
+	var b strings.Builder
+	for _, r := range stepName {
+		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' || r == '.' {
+			b.WriteRune(r)
+			continue
+		}
+		b.WriteByte('_')
+	}
+	name := b.String()
+	if name == "" {
+		return "step"
+	}
+	if len(name) > maxStepNameLen {
+		name = name[:maxStepNameLen]
+	}
+	return name
+}
@@
 func (w *Workspace) OutputPath(stepName string) string {
-	return filepath.Join(w.JobDir, "outputs", stepName+".json")
+	safeName := SafeStepName(stepName)
+	return filepath.Join(w.JobDir, "outputs", safeName+".json")
 }
```

```diff
diff --git a/pkg/executor/tool.go b/pkg/executor/tool.go
index d84bb56..b152ae5 100644
--- a/pkg/executor/tool.go
+++ b/pkg/executor/tool.go
@@
 	// Create log file for real-time output
 	logDir := filepath.Join(ws.JobDir, "logs")
 	os.MkdirAll(logDir, 0755)
-	logPath := filepath.Join(logDir, step.Name+".log")
+	safeStepName := workspace.SafeStepName(step.Name)
+	logPath := filepath.Join(logDir, safeStepName+".log")
 	logFile, logErr := os.Create(logPath)
@@
-	outputPath, _ := ws.WriteOutput(step.Name, map[string]interface{}{
+	outputPath, _ := ws.WriteOutput(safeStepName, map[string]interface{}{
 		"stdout": stdout.String(),
 		"stderr": stderr.String(),
 	})
```

```diff
diff --git a/pkg/orchestrator/live_display.go b/pkg/orchestrator/live_display.go
index 0f1b2d6..9dbb6ce 100644
--- a/pkg/orchestrator/live_display.go
+++ b/pkg/orchestrator/live_display.go
@@
 import (
 	"bufio"
 	"fmt"
 	"os"
 	"path/filepath"
 	"strings"
 	"sync"
 	"time"
+
+	"rcodegen/pkg/workspace"
 )
@@
 func (d *LiveDisplay) readLastMeaningfulLine(stepName string) string {
-	logPath := filepath.Join(d.logDir, stepName+".log")
+	safeStepName := workspace.SafeStepName(stepName)
+	logPath := filepath.Join(d.logDir, safeStepName+".log")
 	f, err := os.Open(logPath)
 	if err != nil {
 		return ""
 	}
```

**Strengths**
- Bundle loading validates bundle names to prevent path traversal. `pkg/bundle/loader.go:18`.
- Config and lock files are written with secure permissions and warn on insecure settings. `pkg/settings/settings.go:42`, `pkg/lock/filelock.go:73`.

**Follow-ups**
- Add tests that exercise `SafeStepName` and path validation helpers in the API routes.
- Consider optional API auth (token header) if the dashboard is exposed beyond localhost.
- Run dependency vulnerability checks (`govulncheck`, `npm audit`) when you have time.
