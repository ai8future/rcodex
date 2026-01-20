# Codebase Fix Report - rcodegen

Date Created: Friday, January 16, 2026, 21:47

## Overview
This report identifies several bugs, logical issues, and code smells found in the `rcodegen` codebase during an audit on 2026-01-16. The most critical issue involves broken conditional step logic in the `Orchestrator`, which prevents `else` branches from ever executing. Other issues include incorrect operator precedence in condition evaluation, potential performance bottlenecks in terminal display, and inconsistent error handling for system calls.

## Identified Issues

### 1. Broken Conditional Logic (Critical)
**Location:** `pkg/orchestrator/orchestrator.go`
**Issue:** The `Orchestrator` uses a top-level `if step.If != ""` guard that skips the entire step if the condition is false. This prevents the subsequent `if step.Then != nil` block from ever reaching its `else` branch, as the step is already skipped when the condition is false.
**Impact:** `else` branches in task bundles are unreachable and never execute.

### 2. Incorrect Operator Precedence (Medium)
**Location:** `pkg/orchestrator/condition.go`
**Issue:** The `evaluate` function processes `AND` before `OR`. In boolean logic, `AND` usually has higher precedence, but the current implementation splits on the first operator found. Since it checks for `AND` first, an expression like `A OR B AND C` is evaluated as `(A OR B) AND C` instead of `A OR (B AND C)`.
**Impact:** Complex conditional expressions may yield incorrect results.

### 3. Performance Bottleneck in Live Display (Medium)
**Location:** `pkg/orchestrator/live_display.go`
**Issue:** `readLastMeaningfulLine` reads the entire log file from disk every 100ms for each running step. As log files grow (especially during long-running agent tasks), this will cause significant I/O overhead and CPU usage.
**Impact:** Terminal lag and high resource consumption during long tasks.

### 4. Unhandled Errors in System Calls (Low)
**Locations:** Multiple files (`pkg/workspace/workspace.go`, `pkg/orchestrator/orchestrator.go`)
**Issue:** Errors from `rand.Read`, `os.Getwd`, and `os.WriteFile` are frequently ignored or only result in a warning printed to stderr without stopping the execution or returning the error.
**Impact:** Potential silent failures or undefined behavior when system resources are constrained.

## Proposed Fixes (Patch-Ready Diffs)

### Fix 1: Correct Conditional Logic in `Orchestrator`

```diff
--- pkg/orchestrator/orchestrator.go
+++ pkg/orchestrator/orchestrator.go
@@ -155,14 +155,14 @@
 		display.SetStepRunning(i)
 		// Set model immediately so it shows while running
 		display.SetStepModel(i, o.getStepModel(step.Tool, step.Model))
 
-		// Check condition
-		if step.If != "" && !EvaluateCondition(step.If, ctx) {
-			display.SetStepSkipped(i)
-			ctx.SetResult(step.Name, &envelope.Envelope{Status: envelope.StatusSkipped})
-			continue
-		}
-
 		// Handle conditional step
 		if step.Then != nil {
 			if EvaluateCondition(step.If, ctx) {
@@ -179,6 +179,16 @@
 				}
 			}
 			continue
+		}
+
+		// Check condition for normal step
+		if step.If != "" && !EvaluateCondition(step.If, ctx) {
+			display.SetStepSkipped(i)
+			ctx.SetResult(step.Name, &envelope.Envelope{Status: envelope.StatusSkipped})
+			continue
 		}
 
 		// Apply model overrides
```

### Fix 2: Correct Operator Precedence in `EvaluateCondition`

```diff
--- pkg/orchestrator/condition.go
+++ pkg/orchestrator/condition.go
@@ -19,11 +19,13 @@
 func evaluate(expr string) bool {
 	expr = strings.TrimSpace(expr)
 
-	// Handle AND/OR
-	if idx := strings.Index(expr, " AND "); idx != -1 {
-		return evaluate(expr[:idx]) && evaluate(expr[idx+5:])
-	}
+	// Handle OR first (lowest precedence)
 	if idx := strings.Index(expr, " OR "); idx != -1 {
 		return evaluate(expr[:idx]) || evaluate(expr[idx+4:])
 	}
+
+	// Handle AND (higher precedence)
+	if idx := strings.Index(expr, " AND "); idx != -1 {
+		return evaluate(expr[:idx]) && evaluate(expr[idx+5:])
+	}
 
 	// Handle comparisons
```

### Fix 3: Handle `rand.Read` error in `Workspace`

```diff
--- pkg/workspace/workspace.go
+++ pkg/workspace/workspace.go
@@ -20,7 +20,9 @@
 func GenerateJobID() string {
 	now := time.Now()
 	b := make([]byte, 4)
-	rand.Read(b)
+	if _, err := rand.Read(b); err != nil {
+		return fmt.Sprintf("%s-%08x", now.Format("20060102-150405"), now.UnixNano())
+	}
 	return fmt.Sprintf("%s-%s", now.Format("20060102-150405"), hex.EncodeToString(b))
 }
```

### Fix 4: Optimize `LiveDisplay` Log Reading

```diff
--- pkg/orchestrator/live_display.go
+++ pkg/orchestrator/live_display.go
@@ -131,6 +131,14 @@
 	}
 	defer f.Close()
 
+	// If file is large, only read the last 4KB
+	info, err := f.Stat()
+	if err == nil && info.Size() > 4096 {
+		f.Seek(-4096, io.SeekEnd)
+	}
+
 	var lastLine string
 	scanner := bufio.NewScanner(f)
 	for scanner.Scan() {
```

## Summary of Changes
1.  **Orchestrator**: Relocated the condition guard to only apply to non-container steps, allowing `If/Then/Else` steps to correctly evaluate their branches.
2.  **Condition Parser**: Swapped the evaluation order of `AND` and `OR` to ensure `AND` has higher precedence (standard boolean logic).
3.  **Workspace**: Added error handling for random number generation, with a timestamp-based fallback to ensure Job ID uniqueness even if entropy is unavailable.
4.  **LiveDisplay**: Added a seek optimization to avoid reading massive log files repeatedly, limiting the scan to the tail of the file where recent status updates are located.
5.  **General**: Recommended adding error checks for `os.Getwd()` and `os.WriteFile` in `orchestrator.go` to improve system robustness.
