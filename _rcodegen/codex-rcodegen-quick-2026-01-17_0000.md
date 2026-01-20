Date Created: 2026-01-17 00:00:57 +0100
Date Updated: 2026-01-17
TOTAL_SCORE: 84/100

## Items Fixed (2026-01-17)
- ~~Orchestrator run report mislabels Codex edits as Gemini~~ - FIXED in commit 415ba77
- ~~VoteExecutor reads shared step results without locking~~ - FIXED: Added Context.GetResult() in commit 415ba77
- ~~MergeExecutor silently drops missing inputs~~ - FIXED in commit 30a0f39

Quick score rationale: solid structure and CLI ergonomics, but several correctness/robustness gaps in executor/orchestrator and thin tests around cost/session parsing.

## AUDIT
- Sensitive run artifacts (workspace outputs + logs) are created with permissive permissions (0755 dirs / 0666 files), which can leak prompts/output on multi-user systems. Tighten directory/file modes.

```diff
diff --git a/pkg/workspace/workspace.go b/pkg/workspace/workspace.go
index 4a4d8f6..b3510a6 100644
--- a/pkg/workspace/workspace.go
+++ b/pkg/workspace/workspace.go
@@
-	for _, dir := range dirs {
-		if err := os.MkdirAll(dir, 0755); err != nil {
+	for _, dir := range dirs {
+		if err := os.MkdirAll(dir, 0700); err != nil {
 			return nil, err
 		}
 	}
@@
-	f, err := os.Create(path)
+	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
 	if err != nil {
 		return "", err
 	}
*** End Patch
```

```diff
diff --git a/pkg/executor/tool.go b/pkg/executor/tool.go
index 61b3fa4..b3d1c0b 100644
--- a/pkg/executor/tool.go
+++ b/pkg/executor/tool.go
@@
-	logDir := filepath.Join(ws.JobDir, "logs")
-	os.MkdirAll(logDir, 0755)
-	logPath := filepath.Join(logDir, step.Name+".log")
-	logFile, logErr := os.Create(logPath)
+	logDir := filepath.Join(ws.JobDir, "logs")
+	logPath := filepath.Join(logDir, step.Name+".log")
+	var logFile *os.File
+	logErr := os.MkdirAll(logDir, 0700)
+	if logErr == nil {
+		logFile, logErr = os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
+	}
*** End Patch
```

- Settings file permission check only flags world-writable; group-writable configs are also risky (config poisoning). Warn on either.

```diff
diff --git a/pkg/settings/settings.go b/pkg/settings/settings.go
index c9c7a74..6e4a9a2 100644
--- a/pkg/settings/settings.go
+++ b/pkg/settings/settings.go
@@
-	// Warn if settings file is world-writable (security risk)
+	// Warn if settings file is group/world-writable (security risk)
 	mode := info.Mode().Perm()
-	if mode&0002 != 0 { // world-writable
-		fmt.Fprintf(os.Stderr, "Warning: settings file %s is world-writable (mode %o). This is a security risk.\n", configPath, mode)
+	if mode&0022 != 0 { // group or world-writable
+		fmt.Fprintf(os.Stderr, "Warning: settings file %s is writable by group/others (mode %o). This is a security risk.\n", configPath, mode)
 		fmt.Fprintf(os.Stderr, "Run: chmod 600 %s\n", configPath)
 	}
*** End Patch
```

## TESTS
- Executor parsing utilities have no coverage; add focused unit tests for cost/token parsing and session ID extraction to prevent silent regressions.

```diff
diff --git a/pkg/executor/tool_test.go b/pkg/executor/tool_test.go
new file mode 100644
index 0000000..a5b6a3c
--- /dev/null
+++ b/pkg/executor/tool_test.go
@@
+package executor
+
+import (
+	"math"
+	"testing"
+)
+
+func assertFloatClose(t *testing.T, got, want float64) {
+	t.Helper()
+	if math.Abs(got-want) > 1e-9 {
+		t.Fatalf("expected %f, got %f", want, got)
+	}
+}
+
+func TestExtractCostInfoClaude(t *testing.T) {
+	stdout := `{"type":"assistant"}
+{"type":"result","total_cost_usd":0.0123,"usage":{"input_tokens":100,"output_tokens":50,"cache_read_input_tokens":10,"cache_creation_input_tokens":5}}`
+	usage := extractCostInfo("claude", stdout, "")
+
+	if usage.InputTokens != 100 {
+		t.Fatalf("expected input tokens 100, got %d", usage.InputTokens)
+	}
+	if usage.OutputTokens != 50 {
+		t.Fatalf("expected output tokens 50, got %d", usage.OutputTokens)
+	}
+	if usage.CacheReadTokens != 10 {
+		t.Fatalf("expected cache read tokens 10, got %d", usage.CacheReadTokens)
+	}
+	if usage.CacheWriteTokens != 5 {
+		t.Fatalf("expected cache write tokens 5, got %d", usage.CacheWriteTokens)
+	}
+	assertFloatClose(t, usage.CostUSD, 0.0123)
+}
+
+func TestExtractCostInfoCodex(t *testing.T) {
+	stderr := "tokens used\n7,476\n"
+	usage := extractCostInfo("codex", "", stderr)
+
+	if usage.InputTokens != 5233 {
+		t.Fatalf("expected input tokens 5233, got %d", usage.InputTokens)
+	}
+	if usage.OutputTokens != 2242 {
+		t.Fatalf("expected output tokens 2242, got %d", usage.OutputTokens)
+	}
+	assertFloatClose(t, usage.CostUSD, 0.11959)
+}
+
+func TestExtractCostInfoGemini(t *testing.T) {
+	stdout := `{"type":"result","stats":{"input_tokens":100,"output_tokens":200,"cached":10}}`
+	usage := extractCostInfo("gemini", stdout, "")
+
+	if usage.InputTokens != 100 {
+		t.Fatalf("expected input tokens 100, got %d", usage.InputTokens)
+	}
+	if usage.OutputTokens != 200 {
+		t.Fatalf("expected output tokens 200, got %d", usage.OutputTokens)
+	}
+	if usage.CacheReadTokens != 10 {
+		t.Fatalf("expected cache read tokens 10, got %d", usage.CacheReadTokens)
+	}
+	assertFloatClose(t, usage.CostUSD, 0.00035)
+}
+
+func TestExtractSessionIDClaudeGemini(t *testing.T) {
+	stdout := `{"type":"system","session_id":"abc-123"}`
+	if got := extractSessionID("claude", stdout, ""); got != "abc-123" {
+		t.Fatalf("expected Claude session id abc-123, got %q", got)
+	}
+	if got := extractSessionID("gemini", stdout, ""); got != "abc-123" {
+		t.Fatalf("expected Gemini session id abc-123, got %q", got)
+	}
+}
+
+func TestExtractSessionIDCodex(t *testing.T) {
+	stderr := "session id: 123e4567-e89b-12d3-a456-426614174000"
+	if got := extractSessionID("codex", "", stderr); got != "123e4567-e89b-12d3-a456-426614174000" {
+		t.Fatalf("expected Codex session id, got %q", got)
+	}
+}
*** End Patch
```

## FIXES
- Orchestrator run report mislabels Codex edits as Gemini in the expanded summary table, which skews cost attribution.

```diff
diff --git a/pkg/orchestrator/orchestrator.go b/pkg/orchestrator/orchestrator.go
index 3e5fb14..b1ef1bf 100644
--- a/pkg/orchestrator/orchestrator.go
+++ b/pkg/orchestrator/orchestrator.go
@@
-				codexEditCost := getSubstepCost(ctx, "edit-codex")
-				geminiEditCost := getSubstepCost(ctx, "edit-gemini")
-				expanded = append(expanded, ExpandedStep{stepNum, "Edit", "Gemini", "✓", codexEditCost, codexOut})
-				expanded = append(expanded, ExpandedStep{stepNum, "Edit", "Gemini", "✓", geminiEditCost, geminiOut})
+				codexEditCost := getSubstepCost(ctx, "edit-codex")
+				geminiEditCost := getSubstepCost(ctx, "edit-gemini")
+				expanded = append(expanded, ExpandedStep{stepNum, "Edit", "Codex", "✓", codexEditCost, codexOut})
+				expanded = append(expanded, ExpandedStep{stepNum, "Edit", "Gemini", "✓", geminiEditCost, geminiOut})
*** End Patch
```

- VoteExecutor reads shared step results without locking (data race in parallel runs) and ignores write errors from workspace outputs.

```diff
diff --git a/pkg/orchestrator/context.go b/pkg/orchestrator/context.go
index 82f3184..c0f8f83 100644
--- a/pkg/orchestrator/context.go
+++ b/pkg/orchestrator/context.go
@@
 func (c *Context) SetResult(name string, env *envelope.Envelope) {
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
*** End Patch
```

```diff
diff --git a/pkg/executor/vote.go b/pkg/executor/vote.go
index 2f55185..ed2ab1c 100644
--- a/pkg/executor/vote.go
+++ b/pkg/executor/vote.go
@@
-		stepName := extractStepName(inputRef)
-		if env, ok := ctx.StepResults[stepName]; ok {
+		stepName := extractStepName(inputRef)
+		if env, ok := ctx.GetResult(stepName); ok {
 			if env.Status == envelope.StatusSuccess {
 				votes["success"]++
 			} else {
 				votes["failure"]++
 			}
 		}
 	}
@@
-	outputPath, _ := ws.WriteOutput(step.Name, map[string]interface{}{
+	outputPath, err := ws.WriteOutput(step.Name, map[string]interface{}{
 		"votes":    votes,
 		"decision": decision,
 	})
+	if err != nil {
+		return envelope.New().Failure("WRITE_ERROR", err.Error()).Build(), err
+	}
*** End Patch
```

- MergeExecutor silently drops missing inputs, producing incomplete merges without surfacing an error.

```diff
diff --git a/pkg/executor/merge.go b/pkg/executor/merge.go
index 89f59c6..b52d2fd 100644
--- a/pkg/executor/merge.go
+++ b/pkg/executor/merge.go
@@
-import (
-	"os"
-	"strings"
+import (
+	"fmt"
+	"os"
+	"strings"
@@
-	var contents []string
+	var contents []string
+	var missing []string
 	for _, inputRef := range step.Merge.Inputs {
 		path := ctx.Resolve(inputRef)
 		data, err := os.ReadFile(path)
 		if err != nil {
-			continue
+			missing = append(missing, path)
+			continue
 		}
 		contents = append(contents, string(data))
 	}
+	if len(missing) > 0 {
+		err := fmt.Errorf("missing inputs: %s", strings.Join(missing, ", "))
+		return envelope.New().Failure("READ_ERROR", err.Error()).Build(), err
+	}
*** End Patch
```

## REFACTOR
- Consolidate duplicate summary rendering between `pkg/runner/output.go` and `pkg/runner/runner.go` to avoid drift.
- Centralize artifact-writing helpers (permissions + error handling) so executor/orchestrator can reuse a single safe API.
- Replace ad-hoc parsing like `extractStepName` with a small shared parser or `strings.SplitN` to simplify and reduce edge cases.
- Consider typed result structs for envelope results to avoid repeated `map[string]interface{}` assertions throughout orchestrator/executor.
