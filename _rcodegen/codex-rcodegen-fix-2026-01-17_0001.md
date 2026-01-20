Date Created: 2026-01-17 00:01:16 +0100
Date Updated: 2026-01-17
TOTAL_SCORE: 87/100

## Items Fixed (2026-01-17)
- ~~Vote step name parsing / race condition~~ - FIXED: Added Context.GetResult() in commit 415ba77
- ~~Run report "edits" substep labels Codex as Gemini~~ - FIXED in commit 415ba77
- ~~MergeExecutor ignores input read failures~~ - FIXED in commit 30a0f39

Scope
- Quick scan of Go sources in cmd/ and pkg/ (orchestrator, executor, runner, bundle, workspace, tools).
- Focused on correctness, concurrency safety, and report accuracy.
- Tests not run (no code changes applied).

Findings (ordered by severity)
1) Medium - Vote step name parsing drops ${steps.name} to empty string, so vote inputs can be skipped.
   - File: pkg/executor/vote.go
   - Impact: votes can be under-counted when inputs are references without a trailing dot.
   - Fix: parse until '.' or '}' and fall back to the trimmed step name.

2) Medium - Vote executor reads Context.StepResults without a lock, risking data races if vote runs alongside parallel step writes.
   - Files: pkg/executor/vote.go, pkg/orchestrator/context.go
   - Impact: potential race in parallel executions and hard-to-reproduce vote inconsistencies.
   - Fix: add Context.GetResult accessor with read lock and use it in VoteExecutor.

3) Low - Run report "edits" substep labels Codex edit as Gemini.
   - File: pkg/orchestrator/orchestrator.go
   - Impact: report is misleading for parallel edit steps.
   - Fix: label Codex row correctly.

Patch-ready diffs (not applied; per DO NOT EDIT CODE)

```diff
diff --git a/pkg/orchestrator/context.go b/pkg/orchestrator/context.go
index 0e7bca7..4fbb84f 100644
--- a/pkg/orchestrator/context.go
+++ b/pkg/orchestrator/context.go
@@ -105,6 +105,14 @@ func (c *Context) SetResult(name string, env *envelope.Envelope) {
 	c.mu.Lock()
 	defer c.mu.Unlock()
 	c.StepResults[name] = env
 }
+
+// GetResult returns a stored step result safely.
+func (c *Context) GetResult(name string) (*envelope.Envelope, bool) {
+	c.mu.RLock()
+	defer c.mu.RUnlock()
+	env, ok := c.StepResults[name]
+	return env, ok
+}
```

```diff
diff --git a/pkg/executor/vote.go b/pkg/executor/vote.go
index 2f1a1c1..f6ad66b 100644
--- a/pkg/executor/vote.go
+++ b/pkg/executor/vote.go
@@ -1,6 +1,8 @@
 package executor
 
 import (
+	"strings"
+
 	"rcodegen/pkg/bundle"
 	"rcodegen/pkg/envelope"
 	"rcodegen/pkg/orchestrator"
@@ -18,7 +20,7 @@ func (e *VoteExecutor) Execute(step *bundle.Step, ctx *orchestrator.Context, ws *workspace.Workspace) (*envelope.Envelope, error) {
 	for _, inputRef := range step.Vote.Inputs {
 		// Extract step name from ${steps.name.output_ref}
 		// For now, just count successful steps
 		stepName := extractStepName(inputRef)
-		if env, ok := ctx.StepResults[stepName]; ok {
+		if env, ok := ctx.GetResult(stepName); ok && env != nil {
 			if env.Status == envelope.StatusSuccess {
 				votes["success"]++
 			} else {
@@ -63,16 +65,18 @@ func (e *VoteExecutor) Execute(step *bundle.Step, ctx *orchestrator.Context, ws *workspace.Workspace) (*envelope.Envelope, error) {
 }
 
 func extractStepName(ref string) string {
 	// ${steps.name.output_ref} -> name
-	if len(ref) > 9 && ref[:8] == "${steps." {
-		end := 8
-		for i := 8; i < len(ref); i++ {
-			if ref[i] == '.' {
-				return ref[8:i]
-			}
-		}
-		return ref[8:end]
+	const prefix = "${steps."
+	if strings.HasPrefix(ref, prefix) {
+		trimmed := strings.TrimSuffix(ref[len(prefix):], "}")
+		if idx := strings.Index(trimmed, "."); idx != -1 {
+			return trimmed[:idx]
+		}
+		if trimmed != "" {
+			return trimmed
+		}
 	}
 	return ref
 }
```

```diff
diff --git a/pkg/orchestrator/orchestrator.go b/pkg/orchestrator/orchestrator.go
index 9f0e8d3..21d2a14 100644
--- a/pkg/orchestrator/orchestrator.go
+++ b/pkg/orchestrator/orchestrator.go
@@ -478,7 +478,7 @@ func generateRunReport(path, jobID, bundleName string, duration time.Duration, totalCost float64, stats []StepStats, ctx *Context, outputDir string) {
 			}
 			codexEditCost := getSubstepCost(ctx, "edit-codex")
 			geminiEditCost := getSubstepCost(ctx, "edit-gemini")
-			expanded = append(expanded, ExpandedStep{stepNum, "Edit", "Gemini", "✓", codexEditCost, codexOut})
+			expanded = append(expanded, ExpandedStep{stepNum, "Edit", "Codex", "✓", codexEditCost, codexOut})
 			expanded = append(expanded, ExpandedStep{stepNum, "Edit", "Gemini", "✓", geminiEditCost, geminiOut})
 		} else {
 			// Generic parallel
 			expanded = append(expanded, ExpandedStep{stepNum, s.Name, "parallel", "✓", s.Cost, "-"})
 		}
```

Notes / suggestions
- MergeExecutor ignores input read failures; consider returning a failure or surfacing missing inputs to avoid silent empty merges. (pkg/executor/merge.go)
- A few os.WriteFile calls ignore errors (e.g., report/bundle copy); consider logging to stderr for visibility.

Status
- No code edits applied (per DO NOT EDIT CODE). Diffs above are ready to apply.
