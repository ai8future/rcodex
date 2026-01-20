Date Created: Friday, January 16, 2026 23:45:00
Date Updated: 2026-01-17
TOTAL_SCORE: 88/100

## Items Fixed (2026-01-17)
- ~~Incorrect Operator Precedence in Condition Evaluation~~ - FIXED in commit d950e95
- ~~Deprecated strings.Title~~ - FIXED: Added capitalizeWord() helper in commit d950e95

# (1) AUDIT - Security and Code Quality Issues

### Incorrect Operator Precedence in Condition Evaluation
The condition evaluator in `pkg/orchestrator/condition.go` incorrectly evaluates `AND` before `OR`. In most expression languages, `AND` has higher precedence than `OR`, meaning it should be evaluated later (deeper in the recursion) or processed specifically to ensure `A OR B AND C` is treated as `A OR (B AND C)`. The current implementation treats it as `(A OR B) AND C`.

```diff
--- a/pkg/orchestrator/condition.go
+++ b/pkg/orchestrator/condition.go
@@ -17,11 +17,11 @@
 func evaluate(expr string) bool {
 	expr = strings.TrimSpace(expr)
 
-	// Handle AND/OR
-	if idx := strings.Index(expr, " AND "); idx != -1 {
-		return evaluate(expr[:idx]) && evaluate(expr[idx+5:])
-	}
+	// Handle OR first (lower precedence)
 	if idx := strings.Index(expr, " OR "); idx != -1 {
 		return evaluate(expr[:idx]) || evaluate(expr[idx+4:])
 	}
+	// Handle AND (higher precedence)
+	if idx := strings.Index(expr, " AND "); idx != -1 {
+		return evaluate(expr[:idx]) && evaluate(expr[idx+5:])
+	}
 
 	// Handle comparisons
```

### Performance: File IO inside RWMutex
In `pkg/orchestrator/context.go`, the `Resolve` method performs `os.ReadFile` while holding a read lock (`c.mu.RLock()`). While functionally correct, this can cause significant performance bottlenecks if multiple goroutines are resolving variables simultaneously, as the read lock will be held for the duration of the IO operation.

```diff
--- a/pkg/orchestrator/context.go
+++ b/pkg/orchestrator/context.go
@@ -62,13 +62,17 @@
 					case "stdout", "stderr":
 						// Read from output file
 						if env.OutputRef != "" {
-							// NOTE: Reading file IO inside the lock.
-							// For high throughput this might be a bottleneck, but for correctness it's safe.
-							if data, err := os.ReadFile(env.OutputRef); err == nil {
+							// RELEASE LOCK BEFORE IO
+							c.mu.RUnlock()
+							data, err := os.ReadFile(env.OutputRef)
+							c.mu.RLock()
+							if err == nil {
 								var output map[string]interface{}
 								if err := json.Unmarshal(data, &output); err == nil {
 									if v, ok := output[parts[2]]; ok {
 										content := fmt.Sprintf("%v", v)
 										// For Claude/Codex streaming JSON output, extract the result
 										return extractStreamingResult(content)
 									}
```

# (2) TESTS - Proposed Unit Tests

### Comprehensive Condition Evaluation Tests
The current `condition.go` lacks unit tests for complex logical expressions and edge cases.

```diff
--- /dev/null
+++ b/pkg/orchestrator/condition_test.go
@@ -0,0 +1,45 @@
+package orchestrator
+
+import "testing"
+
+func TestEvaluateCondition(t *testing.T) {
+	ctx := NewContext(map[string]string{"foo": "bar"})
+	
+	tests := []struct {
+		name      string
+		condition string
+		expected  bool
+	}{
+		{"Empty", "", true},
+		{"Simple True", "true", true},
+		{"Simple False", "false", false},
+		{"Equals", "'bar' == 'bar'", true},
+		{"Not Equals", "'bar' != 'baz'", true},
+		{"AND True", "true AND true", true},
+		{"AND False", "true AND false", false},
+		{"OR True", "true OR false", true},
+		{"Precedence OR-AND", "true OR false AND false", true}, // Should be true OR (false AND false)
+		{"Precedence AND-OR", "false AND false OR true", true}, // Should be (false AND false) OR true
+		{"Contains", "'hello world' contains 'hello'", true},
+		{"Numeric GT", "10 > 5", true},
+		{"Numeric LTE", "5 <= 5", true},
+	}
+
+	for _, tt := range tests {
+		t.Run(tt.name, func(t *testing.T) {
+			if got := evaluate(tt.condition); got != tt.expected {
+				t.Errorf("evaluate() = %v, want %v", got, tt.expected)
+			}
+		})
+	}
+}
```

# (3) FIXES - Bugs, Issues, and Code Smells

### Deprecated `strings.Title`
`strings.Title` is deprecated since Go 1.18. It should be replaced with a custom capitalization function or `cases.Title`.

```diff
--- a/pkg/orchestrator/live_display.go
+++ b/pkg/orchestrator/live_display.go
@@ -310,8 +310,13 @@
 		iconColor = colorDim
 		statusInfo = fmt.Sprintf(" %s(skipped)%s", colorDim, colorReset)
 	}
+	
+	capitalize := func(s string) string {
+		if s == "" { return "" }
+		return strings.ToUpper(s[:1]) + s[1:]
+	}
 
-	toolClr := toolColor(step.Tool)
-	toolName := strings.Title(step.Tool)
+	toolClr := toolColor(step.Tool)
+	toolName := capitalize(step.Tool)
 
 	// Show tool/model (e.g., "Claude/Sonnet" or just "Claude")
 	toolDisplay := toolName
-	if step.Model != "" {
-		modelName := strings.Title(step.Model)
+	if step.Model != "" {
+		modelName := capitalize(step.Model)
```

### Incomplete Error Handling in `runner.go`
The `Run` method in `runner.go` ignores some errors that could lead to unexpected behavior, especially when creating directories.

```diff
--- a/pkg/runner/runner.go
+++ b/pkg/runner/runner.go
@@ -99,6 +99,9 @@
 		if err := os.MkdirAll(reportDir, 0755); err != nil {
 			return runError(1, fmt.Errorf("error creating output directory %s: %v", reportDir, err))
 		}
+	} else {
+		// Ensure default report directory exists
+		_ = os.MkdirAll(reportDir, 0755)
 	}
 	// Substitute {report_dir} in the task being executed (don't mutate shared TaskConfig)
 	cfg.Task = strings.ReplaceAll(cfg.Task, "{report_dir}", reportDir)
```

# (4) REFACTOR - Opportunities to Improve Code Quality

1.  **Robust Condition Parser**: Replace the recursive `strings.Index` based evaluator in `condition.go` with a proper Pratt parser or Shunting-yard algorithm to support parentheses and proper operator precedence naturally.
2.  **Context Caching**: Implement a cache for file contents in `Context.Resolve` to avoid repeated `os.ReadFile` calls during a single task run.
3.  **Tool Interface Expansion**: Add a `Capabilities()` method to the `Tool` interface to allow the orchestrator to dynamically discover supported features (like status tracking or streaming) rather than checking interfaces or using hardcoded switches.
4.  **Logging Abstraction**: Create a dedicated `Logger` interface instead of relying on `fmt.Printf` and direct file writes throughout the codebase. This would make testing easier and allow for different log targets (e.g., syslog, cloud logging).
5.  **Environment Variable Injection**: Allow passing environment variables to the tool processes through the `Config` and `BuildCommand` flow, which is currently missing.
