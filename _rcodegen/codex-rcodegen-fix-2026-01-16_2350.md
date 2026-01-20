Date Created: 2026-01-16 23:50:04 +0100
TOTAL_SCORE: 84/100

# Rcodegen Quick Bug/Smell Review

## Scope
- Timeboxed scan of core Go orchestration/execution paths and report generation.
- Focus on correctness, concurrency, and reporting accuracy.

## Score Rationale
- Solid structure and error envelopes overall.
- A few correctness issues in conditional flow and reporting.
- Minor concurrency risk from direct map access and ignored write errors.

## Findings And Fixes (Proposed)
1) Conditional steps with Then/Else skip the Else branch entirely
- Location: `pkg/orchestrator/orchestrator.go:207-229`
- Impact: when `step.Then` is set and the condition is false, the step is marked skipped before the `Else` block can run.
- Fix: handle `Then/Else` before the generic skip check and evaluate the condition once.

2) Vote aggregation reads shared state without a lock and ignores output write errors
- Location: `pkg/executor/vote.go:16-52`, `pkg/orchestrator/context.go:105-109`
- Impact: potential data race if a vote substep runs inside a parallel block; silent failure if the vote output can’t be written.
- Fix: add a `Context.GetResult` helper with an `RLock`, use it in vote execution, and propagate write errors.

3) Report labels misattribute Codex edits as Gemini
- Location: `pkg/orchestrator/orchestrator.go:465-480`
- Impact: run reports show the wrong tool for Codex edit costs/outputs.
- Fix: label the Codex edit row as `Codex`.

## Patch-Ready Diffs

```diff
diff --git a/pkg/orchestrator/orchestrator.go b/pkg/orchestrator/orchestrator.go
--- a/pkg/orchestrator/orchestrator.go
+++ b/pkg/orchestrator/orchestrator.go
@@ -205,26 +205,37 @@ func (o *Orchestrator) Run(b *bundle.Bundle, inputs map[string]string) (*envelope
-		// Check condition
-		if step.If != "" && !EvaluateCondition(step.If, ctx) {
-			display.SetStepSkipped(i)
-			ctx.SetResult(step.Name, &envelope.Envelope{Status: envelope.StatusSkipped})
-			continue
-		}
-
-		// Handle conditional step
-		if step.Then != nil {
-			if EvaluateCondition(step.If, ctx) {
-				env, err := o.dispatcher.Execute(step.Then, ctx, ws)
-				ctx.SetResult(step.Name, env)
-				if err != nil {
-					return env, err
-				}
-			} else if step.Else != nil {
-				env, err := o.dispatcher.Execute(step.Else, ctx, ws)
-				ctx.SetResult(step.Name, env)
-				if err != nil {
-					return env, err
-				}
-			}
-			continue
-		}
+		// Handle conditional step (then/else)
+		if step.Then != nil {
+			conditionMet := true
+			if step.If != "" {
+				conditionMet = EvaluateCondition(step.If, ctx)
+			}
+			if conditionMet {
+				env, err := o.dispatcher.Execute(step.Then, ctx, ws)
+				ctx.SetResult(step.Name, env)
+				if err != nil {
+					return env, err
+				}
+			} else if step.Else != nil {
+				env, err := o.dispatcher.Execute(step.Else, ctx, ws)
+				ctx.SetResult(step.Name, env)
+				if err != nil {
+					return env, err
+				}
+			} else {
+				display.SetStepSkipped(i)
+				ctx.SetResult(step.Name, &envelope.Envelope{Status: envelope.StatusSkipped})
+			}
+			continue
+		}
+
+		// Check condition
+		if step.If != "" && !EvaluateCondition(step.If, ctx) {
+			display.SetStepSkipped(i)
+			ctx.SetResult(step.Name, &envelope.Envelope{Status: envelope.StatusSkipped})
+			continue
+		}
@@ -476,7 +487,7 @@ func generateRunReport(path, jobID, bundleName string, duration time.Duration, to
-				expanded = append(expanded, ExpandedStep{stepNum, "Edit", "Gemini", "✓", codexEditCost, codexOut})
+				expanded = append(expanded, ExpandedStep{stepNum, "Edit", "Codex", "✓", codexEditCost, codexOut})
 				expanded = append(expanded, ExpandedStep{stepNum, "Edit", "Gemini", "✓", geminiEditCost, geminiOut})
```

```diff
diff --git a/pkg/orchestrator/context.go b/pkg/orchestrator/context.go
--- a/pkg/orchestrator/context.go
+++ b/pkg/orchestrator/context.go
@@ -105,6 +105,13 @@ func (c *Context) SetResult(name string, env *envelope.Envelope) {
 	c.mu.Lock()
 	defer c.mu.Unlock()
 	c.StepResults[name] = env
 }
+
+func (c *Context) GetResult(name string) (*envelope.Envelope, bool) {
+	c.mu.RLock()
+	defer c.mu.RUnlock()
+	env, ok := c.StepResults[name]
+	return env, ok
+}
```

```diff
diff --git a/pkg/executor/vote.go b/pkg/executor/vote.go
--- a/pkg/executor/vote.go
+++ b/pkg/executor/vote.go
@@ -17,7 +17,7 @@ func (e *VoteExecutor) Execute(step *bundle.Step, ctx *orchestrator.Context, ws *
 		// Extract step name from ${steps.name.output_ref}
 		// For now, just count successful steps
 		stepName := extractStepName(inputRef)
-		if env, ok := ctx.StepResults[stepName]; ok {
+		if env, ok := ctx.GetResult(stepName); ok {
 			if env.Status == envelope.StatusSuccess {
 				votes["success"]++
 			} else {
@@ -46,7 +46,11 @@ func (e *VoteExecutor) Execute(step *bundle.Step, ctx *orchestrator.Context, ws *
-	outputPath, _ := ws.WriteOutput(step.Name, map[string]interface{}{
+	outputPath, err := ws.WriteOutput(step.Name, map[string]interface{}{
 		"votes":    votes,
 		"decision": decision,
 	})
+	if err != nil {
+		return envelope.New().Failure("WRITE_ERROR", err.Error()).Build(), err
+	}
```

## Notes
- Per instruction, no code was modified in the repository; the diffs above are patch-ready proposals.

## Suggested Verification
- `go test ./...`
- Run a bundle with a conditional `Then/Else` step to verify else execution.
