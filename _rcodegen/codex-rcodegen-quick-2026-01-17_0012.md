Date Created: 2026-01-17 00:12:47 +0100
Date Updated: 2026-01-17
TOTAL_SCORE: 82/100

## Items Fixed (2026-01-17)
- ~~Path traversal risk in dashboard report APIs~~ - FIXED: Added resolveWithin() validation in commits 39e239d, e8fe7c4
- ~~Report summary labels Codex edit row as Gemini~~ - FIXED in commit 415ba77
- ~~GenerateJobID ignores rand.Read errors~~ - FIXED in commit 30a0f39

## AUDIT - Security and code quality issues with PATCH-READY DIFFS
- Path traversal risk in dashboard report APIs: `name` and `file` are joined directly into paths, allowing `../` to escape `CODE_DIR` and read arbitrary files. Add a safe resolver that enforces a base directory boundary and return 400 on invalid paths.

```diff
diff --git a/dashboard/src/app/api/repos/[name]/reports/[file]/route.ts b/dashboard/src/app/api/repos/[name]/reports/[file]/route.ts
--- a/dashboard/src/app/api/repos/[name]/reports/[file]/route.ts
+++ b/dashboard/src/app/api/repos/[name]/reports/[file]/route.ts
@@ -4,6 +4,17 @@ import fs from 'fs'
 import path from 'path'
 import os from 'os'
 
 const CODE_DIR = path.join(os.homedir(), 'Desktop/_code')
 
+function resolveWithin(baseDir: string, targetPath: string): string | null {
+  const resolvedBase = path.resolve(baseDir)
+  const resolvedTarget = path.resolve(targetPath)
+
+  if (!resolvedTarget.startsWith(resolvedBase + path.sep)) {
+    return null
+  }
+
+  return resolvedTarget
+}
+
 export async function GET(
   request: Request,
   { params }: { params: Promise<{ name: string; file: string }> }
 ) {
   const { name, file } = await params
-  const filePath = path.join(CODE_DIR, name, '_rcodegen', file)
+  const repoPath = path.join(CODE_DIR, name)
+  const safeRepoPath = resolveWithin(CODE_DIR, repoPath)
+  if (!safeRepoPath) {
+    return NextResponse.json({ error: 'Invalid repo path' }, { status: 400 })
+  }
+
+  const reportDir = path.join(safeRepoPath, '_rcodegen')
+  const candidatePath = path.join(reportDir, file)
+  const filePath = resolveWithin(reportDir, candidatePath)
+  if (!filePath) {
+    return NextResponse.json({ error: 'Invalid report path' }, { status: 400 })
+  }
 
   if (!fs.existsSync(filePath)) {
     return NextResponse.json({ error: 'Report not found' }, { status: 404 })
   }
```

```diff
diff --git a/dashboard/src/app/api/repos/[name]/route.ts b/dashboard/src/app/api/repos/[name]/route.ts
--- a/dashboard/src/app/api/repos/[name]/route.ts
+++ b/dashboard/src/app/api/repos/[name]/route.ts
@@ -4,6 +4,17 @@ import fs from 'fs'
 import path from 'path'
 import os from 'os'
 
 const CODE_DIR = path.join(os.homedir(), 'Desktop/_code')
 
+function resolveWithin(baseDir: string, targetPath: string): string | null {
+  const resolvedBase = path.resolve(baseDir)
+  const resolvedTarget = path.resolve(targetPath)
+
+  if (!resolvedTarget.startsWith(resolvedBase + path.sep)) {
+    return null
+  }
+
+  return resolvedTarget
+}
+
 interface ReportDetail {
   filename: string
   tool: string
@@ -54,8 +65,12 @@ export async function GET(
 ) {
   const { name } = await params
   const repoPath = path.join(CODE_DIR, name)
-  const rcodgenDir = path.join(repoPath, '_rcodegen')
+  const safeRepoPath = resolveWithin(CODE_DIR, repoPath)
+  if (!safeRepoPath) {
+    return NextResponse.json({ error: 'Invalid repo path' }, { status: 400 })
+  }
+  const rcodgenDir = path.join(safeRepoPath, '_rcodegen')
 
   if (!fs.existsSync(rcodgenDir)) {
     return NextResponse.json({ error: 'Repo not found' }, { status: 404 })
@@ -100,7 +115,7 @@ export async function GET(
 
     return NextResponse.json({
       name,
-      path: repoPath,
+      path: safeRepoPath,
       reports,
       grouped
     })
```

## TESTS - Proposed unit tests for untested code with PATCH-READY DIFFS
- Add coverage for condition evaluation (AND/OR, contains, input resolution) and runner flag parsing (duplicate detection, var extraction).

```diff
diff --git a/pkg/orchestrator/condition_test.go b/pkg/orchestrator/condition_test.go
new file mode 100644
--- /dev/null
+++ b/pkg/orchestrator/condition_test.go
@@ -0,0 +1,44 @@
+package orchestrator
+
+import "testing"
+
+func TestEvaluateCondition(t *testing.T) {
+  ctx := NewContext(map[string]string{
+    "name":  "alice",
+    "count": "3",
+  })
+
+  tests := []struct {
+    name string
+    expr string
+    want bool
+  }{
+    {"boolean true", "true", true},
+    {"boolean false", "false", false},
+    {"numeric compare", "3 >= 2", true},
+    {"contains", "'alpha' contains 'ph'", true},
+    {"and", "2 > 1 AND 3 < 5", true},
+    {"or", "2 > 3 OR 1 == 1", true},
+    {"inputs resolve", "${inputs.name} == alice", true},
+    {"inputs numeric", "${inputs.count} >= 3", true},
+  }
+
+  for _, tt := range tests {
+    t.Run(tt.name, func(t *testing.T) {
+      if got := EvaluateCondition(tt.expr, ctx); got != tt.want {
+        t.Errorf("EvaluateCondition(%q) = %v, want %v", tt.expr, got, tt.want)
+      }
+    })
+  }
+}
```

```diff
diff --git a/pkg/runner/flags_test.go b/pkg/runner/flags_test.go
new file mode 100644
--- /dev/null
+++ b/pkg/runner/flags_test.go
@@ -0,0 +1,39 @@
+package runner
+
+import (
+  "reflect"
+  "testing"
+)
+
+func TestCheckDuplicateFlags(t *testing.T) {
+  if err := CheckDuplicateFlags([]string{"--model", "gpt-4.1", "-m", "gpt-4.1"}, CommonFlagGroups()); err != nil {
+    t.Fatalf("expected no error, got %v", err)
+  }
+
+  if err := CheckDuplicateFlags([]string{"--model", "gpt-4.1", "-m", "gpt-5.2"}, CommonFlagGroups()); err == nil {
+    t.Fatalf("expected conflict error")
+  }
+}
+
+func TestParseVarFlags(t *testing.T) {
+  args := []string{"-x", "foo=bar", "--model", "gpt-4.1", "-x=baz=qux", "task"}
+  cleaned, vars := ParseVarFlags(args)
+
+  expectedArgs := []string{"--model", "gpt-4.1", "task"}
+  if !reflect.DeepEqual(cleaned, expectedArgs) {
+    t.Fatalf("cleaned args = %v, want %v", cleaned, expectedArgs)
+  }
+  if vars["foo"] != "bar" || vars["baz"] != "qux" {
+    t.Fatalf("vars = %v, want foo=bar and baz=qux", vars)
+  }
+}
```

## FIXES - Bugs, issues, and code smells with fixes and PATCH-READY DIFFS
- Report summary labels the Codex edit row as Gemini, which misattributes costs in article reports. Fix the tool label.

```diff
diff --git a/pkg/orchestrator/orchestrator.go b/pkg/orchestrator/orchestrator.go
--- a/pkg/orchestrator/orchestrator.go
+++ b/pkg/orchestrator/orchestrator.go
@@ -476,7 +476,7 @@ func generateRunReport(path, jobID, bundleName string, duration time.Duration, t
 				}
 				codexEditCost := getSubstepCost(ctx, "edit-codex")
 				geminiEditCost := getSubstepCost(ctx, "edit-gemini")
-				expanded = append(expanded, ExpandedStep{stepNum, "Edit", "Gemini", "✓", codexEditCost, codexOut})
+				expanded = append(expanded, ExpandedStep{stepNum, "Edit", "Codex", "✓", codexEditCost, codexOut})
 				expanded = append(expanded, ExpandedStep{stepNum, "Edit", "Gemini", "✓", geminiEditCost, geminiOut})
 			} else {
 				// Generic parallel
```

- `GenerateJobID` ignores `rand.Read` errors, which can silently reduce entropy or return all-zero IDs. Add an explicit fallback when entropy cannot be read.

```diff
diff --git a/pkg/workspace/workspace.go b/pkg/workspace/workspace.go
--- a/pkg/workspace/workspace.go
+++ b/pkg/workspace/workspace.go
@@ -22,8 +22,12 @@ type Workspace struct {
 func GenerateJobID() string {
 	now := time.Now()
 	b := make([]byte, 4)
-	rand.Read(b)
+	if _, err := rand.Read(b); err != nil {
+		fallback := fmt.Sprintf("%08x", now.UnixNano())
+		return fmt.Sprintf("%s-%s", now.Format("20060102-150405"), fallback[len(fallback)-8:])
+	}
 	return fmt.Sprintf("%s-%s", now.Format("20060102-150405"), hex.EncodeToString(b))
 }
```

## REFACTOR - Opportunities to improve code quality (no diffs needed)
- Centralize path safety logic for dashboard API routes (shared helper to avoid duplication and keep future endpoints consistent).
- Split `generateRunReport` into smaller helpers (data collection vs table rendering) to reduce branching and improve testability.
- Avoid file I/O under the `Context` read lock in `Resolve` by preloading outputs or caching parsed results.
- Unify grade extraction logic between Go and dashboard (single spec or shared patterns) to reduce drift.
