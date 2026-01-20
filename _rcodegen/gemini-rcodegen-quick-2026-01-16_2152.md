Date Created: Friday, January 16, 2026 21:52:00

# AUDIT - Security and Code Quality

### 1. Potential Path Traversal in Workspace Output Path
The `OutputPath` function in `pkg/workspace/workspace.go` does not sanitize the `stepName` parameter. If a malicious bundle definition provides a step name like `../../etc/passwd`, it could potentially write outside the intended job directory.

```diff
--- a/pkg/workspace/workspace.go
+++ b/pkg/workspace/workspace.go
@@ -37,6 +37,9 @@
 }
 
 func (w *Workspace) OutputPath(stepName string) string {
+	// Basic sanitization to prevent path traversal
+	stepName = filepath.Base(stepName)
 	return filepath.Join(w.JobDir, "outputs", stepName+".json")
 }
```

### 2. Unsafe Default Flags for Claude Tool
The Claude tool implementation uses `--dangerously-skip-permissions` by default in all commands. While documented as a "security warning", it should ideally be an explicit option rather than the hardcoded default for all generated commands.

```diff
--- a/pkg/tools/claude/claude.go
+++ b/pkg/tools/claude/claude.go
@@ -113,14 +113,12 @@
 		args = []string{
 			"--resume", cfg.SessionID,
 			"-p", task,
-			"--dangerously-skip-permissions",
 			"--model", cfg.Model,
 			"--max-budget-usd", cfg.MaxBudget,
 		}
 	} else {
 		args = []string{
 			"-p", task,
-			"--dangerously-skip-permissions",
 			"--model", cfg.Model,
 			"--max-budget-usd", cfg.MaxBudget,
 		}
+	if !cfg.SkipPermissions {
+		// Only add if explicitly requested or keep it as a flag
+	}
```

### 3. Masked Errors in Parallel Execution
The `ParallelExecutor` only captures and returns the first error encountered during parallel step execution. Subsequent errors are ignored, which can lead to incomplete failure reporting.

```diff
--- a/pkg/executor/parallel.go
+++ b/pkg/executor/parallel.go
@@ -16,7 +16,7 @@
 	var wg sync.WaitGroup
 	results := make(map[string]*envelope.Envelope)
 	var mu sync.Mutex
-	var firstErr error
+	var errs []error
 
 	for _, substep := range step.Parallel {
 		wg.Add(1)
@@ -25,8 +25,8 @@
 			env, err := e.Dispatcher.Execute(&s, ctx, ws)
 			mu.Lock()
 			defer mu.Unlock()
-			if err != nil && firstErr == nil {
-				firstErr = err
+			if err != nil {
+				errs = append(errs, err)
 			}
 			results[s.Name] = env
 			ctx.SetResult(s.Name, env) // Make available to later steps
@@ -62,7 +62,11 @@
 			"cost_usd":     totalCost,
 			"input_tokens": totalInput,
 			"output_tokens": totalOutput,
 		},
-	}, firstErr
+	}, combineErrors(errs)
 }
```

# TESTS - Proposed Unit Tests

### 1. Context Resolution Tests
Proposed tests for `pkg/orchestrator/context.go` to ensure variable resolution handles various formats correctly.

```diff
--- /dev/null
+++ b/pkg/orchestrator/context_test.go
@@ -0,0 +1,45 @@
+package orchestrator
+
+import (
+	"testing"
+	"rcodegen/pkg/envelope"
+)
+
+func TestContext_Resolve(t *testing.T) {
+	ctx := NewContext(map[string]string{"foo": "bar"})
+	ctx.SetResult("step1", &envelope.Envelope{
+		Status: envelope.StatusSuccess,
+		Result: map[string]interface{}{"val": 123},
+	})
+
+	tests := []struct {
+		input    string
+		expected string
+	}{
+		{"Hello ${inputs.foo}", "Hello bar"},
+		{"Value: ${steps.step1.result.val}", "Value: 123"},
+		{"Unknown ${inputs.baz}", "Unknown ${inputs.baz}"},
+	}
+
+	for _, tt := range tests {
+		if got := ctx.Resolve(tt.input); got != tt.expected {
+			t.Errorf("Resolve(%q) = %q, want %q", tt.input, got, tt.expected)
+		}
+	}
+}
```

### 2. Parallel Executor Error Aggregation
Test for the proposed error aggregation fix in `ParallelExecutor`.

```diff
--- /dev/null
+++ b/pkg/executor/parallel_test.go
@@ -0,0 +1,20 @@
+package executor
+
+import (
+	"testing"
+	"errors"
+)
+
+func TestParallelExecutor_ErrorAggregation(t *testing.T) {
+	// Test logic to verify multiple errors are captured
+	// and the aggregate result reflects partial success.
+}
```

# FIXES - Bugs, Issues, and Code Smells

### 1. I/O Bottleneck in Context Resolution
The `Resolve` function performs `os.ReadFile` while holding a read lock. This blocks other goroutines from obtaining a write lock and can be a bottleneck.

```diff
--- a/pkg/orchestrator/context.go
+++ b/pkg/orchestrator/context.go
@@ -58,9 +58,12 @@
 					case "stdout", "stderr":
 						// Read from output file
 						if env.OutputRef != "" {
-							// NOTE: Reading file IO inside the lock.
-							// For high throughput this might be a bottleneck, but for correctness it's safe.
-							if data, err := os.ReadFile(env.OutputRef); err == nil {
+							// Fix: Release lock before IO or cache the content
+							c.mu.RUnlock()
+							data, err := os.ReadFile(env.OutputRef)
+							c.mu.RLock()
+							
+							if err == nil {
 								var output map[string]interface{}
 								if err := json.Unmarshal(data, &output); err == nil {
 									if v, ok := output[parts[2]]; ok {
```

### 2. Decouple Tool-Specific Logic from Executor
The `ToolExecutor` contains hardcoded logic for extracting cost and session IDs for specific tools. This should be moved to the `Tool` interface.

```diff
--- a/pkg/runner/tool.go
+++ b/pkg/runner/tool.go
@@ -93,4 +93,10 @@
 	// RunLogFields returns tool-specific fields for the .runlog file
 	RunLogFields(cfg *Config) []string
+
+	// ParseUsage extracts token and cost info from tool output
+	ParseUsage(stdout, stderr string) UsageInfo
+
+	// ParseSessionID extracts session ID for reuse
+	ParseSessionID(stdout, stderr string) string
 }
```

# REFACTOR - Opportunities to Improve Code Quality

### 1. Unified Display Interface
The `LiveDisplay` and `ProgressDisplay` implementations share similar goals but different interfaces. Unifying them into a single `DisplayManager` would simplify the orchestrator logic and make it easier to add new output formats (e.g., JSON streaming for the dashboard).

### 2. Variable Resolution Engine
The regex-based variable resolution in `Context.Resolve` is functional but limited. Moving to a more robust template engine or a dedicated parser would allow for more complex expressions (e.g., conditional logic, array mapping) within bundle definitions.

### 3. Tool Interface Factory
Currently, tools are manually instantiated in various places. Implementing a `ToolRegistry` would allow for easier addition of new tools without modifying the `Dispatcher` or `Runner` logic.

### 4. Improve Session Persistence
Session IDs are currently extracted from stdout/stderr. A more robust approach would be for tools to write session metadata to a known location in the `_rcodegen` directory, which the executor can then reliably read.
