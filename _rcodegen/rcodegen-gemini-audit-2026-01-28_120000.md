Date Created: Wednesday, January 28, 2026 12:00:00 PM
TOTAL_SCORE: 72/100

# rcodegen Code Audit Report

## Executive Summary
`rcodegen` is a well-structured CLI tool for orchestrating AI coding agents. It features a clean, interface-based design for tool integration (Claude, Codex, Gemini) and a flexible JSON-based "bundle" system for defining workflows.

The codebase is generally readable and idiomatic Go. However, a critical security vulnerability exists regarding file path handling in bundle steps, and the orchestrator component exhibits signs of tight coupling with specific use cases.

## Grading Breakdown (72/100)

*   **Functionality (20/20):** The tool effectively manages multi-step workflows, tool execution, and output handling as described.
*   **Architecture (15/20):** The `runner.Tool` interface and `orchestrator` pattern are solid. Separation of concerns is mostly good, though the orchestrator knows too much about "article" bundles.
*   **Code Quality (15/20):** Clean Go code, but lacks comprehensive error handling in some areas and relies on hardcoded paths.
*   **Security (10/20):** **CRITICAL:** Path traversal vulnerability in step names. `--yolo` mode in Gemini tool is risky (though documented).
*   **Testing (5/10):** Unit tests exist for some packages (`workspace`, `loader`), but comprehensive integration tests and security tests are missing.
*   **Documentation (7/10):** Good CLI help and structural documentation, but code comments could be more descriptive in complex areas.

## Critical Security Findings

### 1. Path Traversal via Step Names (High Severity)
**Location:** `pkg/workspace/workspace.go` and `pkg/executor/tool.go`
**Description:** The application uses `step.Name` directly to construct file paths for logs and outputs:
```go
// pkg/workspace/workspace.go
func (w *Workspace) OutputPath(stepName string) string {
    return filepath.Join(w.JobDir, "outputs", stepName+".json")
}
```
**Impact:** A malicious bundle could define a step named `../../../../etc/passwd` (or similar), causing `rcodegen` to overwrite sensitive files outside the workspace directory when running that bundle.
**Recommendation:** Validate `step.Name` during bundle loading to ensure it contains only safe characters (alphanumeric, hyphens, underscores).

### 2. Unrestricted Command Execution (Medium Severity)
**Location:** `pkg/tools/gemini/gemini.go` (and others)
**Description:** The `gemini` tool runs with `--yolo`, effectively bypassing user confirmation for tool actions. While this is likely intended for automation, it poses a risk if the LLM is hallucinating or malicious prompts are injected.
**Recommendation:** Consider adding a "safe mode" or requiring explicit user opt-in flag for `--yolo` behavior at the top-level CLI.

## Code Quality Issues

### 1. Hardcoded Paths
**Location:** `pkg/bundle/loader.go`, `pkg/workspace/workspace.go`
**Description:** Paths like `.rcodegen/bundles` are hardcoded.
**Recommendation:** Move these to a constant or configuration setup that respects XDG Base Directory specification or similar.

### 2. Orchestrator Coupling
**Location:** `pkg/orchestrator/orchestrator.go`
**Description:** The `Run` method contains specific logic for "article" bundles (e.g., `if strings.HasPrefix(b.Name, "article")`). This makes the orchestrator brittle and hard to extend for other bundle types.
**Recommendation:** Refactor this into a "BundleProcessor" interface or use bundle metadata/tags to drive this behavior.

## Patch-Ready Diffs

### Fix: Validate Step Names to Prevent Path Traversal

This patch adds validation to `pkg/bundle/loader.go` to ensure step names are safe before loading a bundle.

```go
diff --git a/pkg/bundle/loader.go b/pkg/bundle/loader.go
index 1234567..890abcd 100644
--- a/pkg/bundle/loader.go
+++ b/pkg/bundle/loader.go
@@ -35,6 +35,16 @@ func validateBundleName(name string) error {
 	return nil
 }
 
+// validateStepName checks if a step name is safe to use in file paths
+func validateStepName(name string) error {
+	if name == "" {
+		return fmt.Errorf("empty step name")
+	}
+	if !validBundleNamePattern.MatchString(name) {
+		return fmt.Errorf("invalid step name '%s': must contain only alphanumeric, hyphens, underscores", name)
+	}
+	return nil
+}
+
 func Load(name string) (*Bundle, error) {
 	// Validate bundle name to prevent path traversal
 	if err := validateBundleName(name); err != nil {
@@ -51,6 +61,9 @@ func Load(name string) (*Bundle, error) {
 		if err := json.Unmarshal(data, &b); err != nil {
 			return nil, fmt.Errorf("invalid bundle %s: %w", name, err)
 		}
+		if err := validateBundleSteps(&b); err != nil {
+			return nil, err
+		}
 		b.SourcePath = userPath
 		return &b, nil
 	}
@@ -64,11 +77,28 @@ func Load(name string) (*Bundle, error) {
 	if err := json.Unmarshal(data, &b); err != nil {
 		return nil, fmt.Errorf("invalid builtin bundle %s: %w", name, err)
 	}
+	if err := validateBundleSteps(&b); err != nil {
+		return nil, err
+	}
 	// For builtin bundles, find the source path relative to the executable
 	b.SourcePath = findBuiltinBundlePath(name)
 	return &b, nil
 }
 
+func validateBundleSteps(b *Bundle) error {
+	for _, step := range b.Steps {
+		if err := validateStepName(step.Name); err != nil {
+			return err
+		}
+		// Also validate parallel steps if any
+		for _, pStep := range step.Parallel {
+			if err := validateStepName(pStep.Name); err != nil {
+				return err
+			}
+		}
+	}
+	return nil
+}
+
 // findBuiltinBundlePath attempts to locate the source file for a builtin bundle
 // This is useful for copying the bundle to output directories
 func findBuiltinBundlePath(name string) string {
```
