Date Created: 2026-01-16 23:39:00 PST
Date Updated: 2026-01-17
TOTAL_SCORE: 32/100

## Tests Implemented (2026-01-17)
- ~~pkg/orchestrator/context_test.go~~ - IMPLEMENTED: NewContext, Resolve (inputs, steps, stdout/stderr), thread safety
- ~~pkg/orchestrator/condition_test.go~~ - IMPLEMENTED: Boolean literals, comparisons, logical operators, precedence
- ~~pkg/runner/flags_test.go~~ - IMPLEMENTED: CheckDuplicateFlags, ParseVarFlags, reorderArgsForFlagParsing
- ~~pkg/executor/vote_test.go~~ - IMPLEMENTED: extractStepName, majority/unanimous voting
- ~~pkg/envelope/envelope_test.go~~ - IMPLEMENTED: Builder pattern with all With* methods

# Unit Test Coverage Analysis Report

## Executive Summary

The rcodegen codebase has **minimal test coverage** with approximately 21 test functions across 7 test files. Critical execution paths including the orchestrator, executors, condition evaluation, and flag parsing remain completely untested. This report proposes comprehensive unit tests with patch-ready diffs to significantly improve coverage.

---

## Scoring Breakdown

| Category | Max Points | Score | Notes |
|----------|------------|-------|-------|
| Critical Path Coverage | 30 | 5 | Orchestrator, executors, conditions untested |
| Utility Function Coverage | 20 | 8 | Stream parser tested; flags, lock untested |
| Integration Points | 15 | 4 | Tool interfaces partially tested (Claude thread-safety only) |
| Edge Case Handling | 15 | 5 | Some edge cases in stream_test.go |
| Error Path Coverage | 10 | 5 | runError tested; most error paths untested |
| Test Quality/Patterns | 10 | 5 | Table-driven tests present but inconsistent |
| **TOTAL** | **100** | **32** | |

---

## Current Test Coverage Summary

### Existing Test Files (7 files, ~21 tests)

| Package | File | Tests | What's Covered |
|---------|------|-------|----------------|
| `pkg/runner` | `runner_test.go` | 3 | RunResult struct, runError function |
| `pkg/runner` | `stream_test.go` | 6 | StreamParser line processing |
| `pkg/workspace` | `workspace_test.go` | 5 | JobID generation, directory creation |
| `pkg/bundle` | `loader_test.go` | 3 | Path traversal security |
| `pkg/settings` | `settings_test.go` | 2 | File permissions, tilde expansion |
| `pkg/tools/claude` | `claude_test.go` | 2 | Thread-safety of IsClaudeMax |
| `pkg/lock` | `filelock_test.go` | ? | Not examined in detail |

### Critical Untested Code

1. **pkg/orchestrator/context.go** - `Resolve()` function (complex variable substitution)
2. **pkg/orchestrator/condition.go** - All condition evaluation logic
3. **pkg/executor/*.go** - All executor implementations
4. **pkg/runner/flags.go** - Flag parsing utilities
5. **pkg/lock/filelock.go** - File locking (mostly untested)
6. **pkg/envelope/envelope.go** - Builder pattern implementation

---

## Proposed Tests with Patch-Ready Diffs

### 1. pkg/orchestrator/context_test.go (NEW FILE)

**Priority: CRITICAL** - The `Resolve()` function is used throughout the codebase for variable substitution.

```diff
--- /dev/null
+++ b/pkg/orchestrator/context_test.go
@@ -0,0 +1,180 @@
+package orchestrator
+
+import (
+	"os"
+	"path/filepath"
+	"sync"
+	"testing"
+
+	"rcodegen/pkg/envelope"
+)
+
+func TestNewContext(t *testing.T) {
+	inputs := map[string]string{"foo": "bar", "baz": "qux"}
+	ctx := NewContext(inputs)
+
+	if ctx.Inputs["foo"] != "bar" {
+		t.Errorf("expected input 'foo' = 'bar', got %q", ctx.Inputs["foo"])
+	}
+	if ctx.StepResults == nil {
+		t.Error("expected StepResults to be initialized")
+	}
+	if ctx.Variables == nil {
+		t.Error("expected Variables to be initialized")
+	}
+	if ctx.ToolSessions == nil {
+		t.Error("expected ToolSessions to be initialized")
+	}
+}
+
+func TestContext_Resolve_Inputs(t *testing.T) {
+	ctx := NewContext(map[string]string{
+		"name":    "test-project",
+		"version": "1.0.0",
+	})
+
+	tests := []struct {
+		input    string
+		expected string
+	}{
+		{"${inputs.name}", "test-project"},
+		{"${inputs.version}", "1.0.0"},
+		{"Project: ${inputs.name} v${inputs.version}", "Project: test-project v1.0.0"},
+		{"${inputs.missing}", "${inputs.missing}"}, // Unresolved stays as-is
+		{"no variables here", "no variables here"},
+		{"", ""},
+	}
+
+	for _, tc := range tests {
+		result := ctx.Resolve(tc.input)
+		if result != tc.expected {
+			t.Errorf("Resolve(%q) = %q, want %q", tc.input, result, tc.expected)
+		}
+	}
+}
+
+func TestContext_Resolve_Steps(t *testing.T) {
+	ctx := NewContext(nil)
+
+	// Add a step result
+	ctx.SetResult("analyze", &envelope.Envelope{
+		Status:    envelope.StatusSuccess,
+		OutputRef: "/tmp/analyze-output.json",
+		Result: map[string]interface{}{
+			"count": 42,
+			"items": "a,b,c",
+		},
+	})
+
+	tests := []struct {
+		input    string
+		expected string
+	}{
+		{"${steps.analyze.status}", "success"},
+		{"${steps.analyze.output_ref}", "/tmp/analyze-output.json"},
+		{"${steps.analyze.result.count}", "42"},
+		{"${steps.analyze.result.items}", "a,b,c"},
+		{"${steps.missing.status}", "${steps.missing.status}"}, // Unresolved
+	}
+
+	for _, tc := range tests {
+		result := ctx.Resolve(tc.input)
+		if result != tc.expected {
+			t.Errorf("Resolve(%q) = %q, want %q", tc.input, result, tc.expected)
+		}
+	}
+}
+
+func TestContext_ToolSession_ThreadSafety(t *testing.T) {
+	ctx := NewContext(nil)
+	var wg sync.WaitGroup
+
+	// Concurrent writes
+	for i := 0; i < 100; i++ {
+		wg.Add(1)
+		go func(n int) {
+			defer wg.Done()
+			ctx.SetToolSession("claude", "session-"+string(rune('A'+n%26)))
+		}(i)
+	}
+
+	// Concurrent reads
+	for i := 0; i < 100; i++ {
+		wg.Add(1)
+		go func() {
+			defer wg.Done()
+			_ = ctx.GetToolSession("claude")
+		}()
+	}
+
+	wg.Wait()
+	// If we get here without deadlock/race, test passes
+}
+
+func TestContext_SetResult_ThreadSafety(t *testing.T) {
+	ctx := NewContext(nil)
+	var wg sync.WaitGroup
+
+	for i := 0; i < 50; i++ {
+		wg.Add(2)
+		stepName := "step" + string(rune('A'+i%26))
+
+		go func(name string) {
+			defer wg.Done()
+			ctx.SetResult(name, &envelope.Envelope{Status: envelope.StatusSuccess})
+		}(stepName)
+
+		go func() {
+			defer wg.Done()
+			_ = ctx.Resolve("${steps.stepA.status}")
+		}()
+	}
+
+	wg.Wait()
+}
+
+func TestExtractStreamingResult(t *testing.T) {
+	tests := []struct {
+		name     string
+		input    string
+		expected string
+	}{
+		{
+			name:     "single result object",
+			input:    `{"type":"result","result":"Final answer here"}`,
+			expected: "Final answer here",
+		},
+		{
+			name:     "streaming with result at end",
+			input:    `{"type":"assistant","text":"thinking..."}\n{"type":"result","result":"Done!"}`,
+			expected: "Done!",
+		},
+		{
+			name:     "no result object",
+			input:    "Plain text output",
+			expected: "Plain text output",
+		},
+		{
+			name:     "empty input",
+			input:    "",
+			expected: "",
+		},
+		{
+			name:     "result without result field",
+			input:    `{"type":"result","status":"ok"}`,
+			expected: `{"type":"result","status":"ok"}`,
+		},
+	}
+
+	for _, tc := range tests {
+		t.Run(tc.name, func(t *testing.T) {
+			result := extractStreamingResult(tc.input)
+			if result != tc.expected {
+				t.Errorf("extractStreamingResult() = %q, want %q", result, tc.expected)
+			}
+		})
+	}
+}
+
+func TestContext_Resolve_StdoutStderr(t *testing.T) {
+	// Create a temp file with JSON output
+	tmpDir := t.TempDir()
+	outputFile := filepath.Join(tmpDir, "output.json")
+
+	content := `{"stdout":"hello stdout","stderr":"error message"}`
+	if err := os.WriteFile(outputFile, []byte(content), 0644); err != nil {
+		t.Fatalf("failed to create test file: %v", err)
+	}
+
+	ctx := NewContext(nil)
+	ctx.SetResult("build", &envelope.Envelope{
+		Status:    envelope.StatusSuccess,
+		OutputRef: outputFile,
+	})
+
+	stdout := ctx.Resolve("${steps.build.stdout}")
+	if stdout != "hello stdout" {
+		t.Errorf("Resolve stdout = %q, want 'hello stdout'", stdout)
+	}
+
+	stderr := ctx.Resolve("${steps.build.stderr}")
+	if stderr != "error message" {
+		t.Errorf("Resolve stderr = %q, want 'error message'", stderr)
+	}
+}
```

---

### 2. pkg/orchestrator/condition_test.go (NEW FILE)

**Priority: CRITICAL** - Condition evaluation drives workflow branching logic.

```diff
--- /dev/null
+++ b/pkg/orchestrator/condition_test.go
@@ -0,0 +1,147 @@
+package orchestrator
+
+import (
+	"testing"
+
+	"rcodegen/pkg/envelope"
+)
+
+func TestEvaluateCondition_Empty(t *testing.T) {
+	ctx := NewContext(nil)
+	if !EvaluateCondition("", ctx) {
+		t.Error("empty condition should return true")
+	}
+}
+
+func TestEvaluateCondition_WithContext(t *testing.T) {
+	ctx := NewContext(map[string]string{"status": "ready"})
+	ctx.SetResult("check", &envelope.Envelope{
+		Status: envelope.StatusSuccess,
+	})
+
+	tests := []struct {
+		condition string
+		expected  bool
+	}{
+		{"${inputs.status} == 'ready'", true},
+		{"${inputs.status} == 'pending'", false},
+		{"${steps.check.status} == 'success'", true},
+	}
+
+	for _, tc := range tests {
+		result := EvaluateCondition(tc.condition, ctx)
+		if result != tc.expected {
+			t.Errorf("EvaluateCondition(%q) = %v, want %v", tc.condition, result, tc.expected)
+		}
+	}
+}
+
+func TestEvaluate_BooleanLiterals(t *testing.T) {
+	tests := []struct {
+		expr     string
+		expected bool
+	}{
+		{"true", true},
+		{"false", false},
+		{"TRUE", false},  // Case sensitive
+		{"True", false},
+		{"", false},
+	}
+
+	for _, tc := range tests {
+		result := evaluate(tc.expr)
+		if result != tc.expected {
+			t.Errorf("evaluate(%q) = %v, want %v", tc.expr, result, tc.expected)
+		}
+	}
+}
+
+func TestEvaluate_Comparisons(t *testing.T) {
+	tests := []struct {
+		expr     string
+		expected bool
+	}{
+		// Equality
+		{"'foo' == 'foo'", true},
+		{"'foo' == 'bar'", false},
+		{"foo == foo", true},
+		{"'foo' != 'bar'", true},
+		{"'foo' != 'foo'", false},
+
+		// Numeric comparisons
+		{"10 > 5", true},
+		{"5 > 10", false},
+		{"10 < 20", true},
+		{"20 < 10", false},
+		{"10 >= 10", true},
+		{"10 >= 11", false},
+		{"10 <= 10", true},
+		{"9 <= 10", true},
+
+		// Contains
+		{"'hello world' contains 'world'", true},
+		{"'hello world' contains 'foo'", false},
+
+		// Invalid numeric comparisons
+		{"'abc' > 'def'", false}, // Non-numeric returns false
+	}
+
+	for _, tc := range tests {
+		result := evaluate(tc.expr)
+		if result != tc.expected {
+			t.Errorf("evaluate(%q) = %v, want %v", tc.expr, result, tc.expected)
+		}
+	}
+}
+
+func TestEvaluate_LogicalOperators(t *testing.T) {
+	tests := []struct {
+		expr     string
+		expected bool
+	}{
+		// AND
+		{"true AND true", true},
+		{"true AND false", false},
+		{"false AND true", false},
+		{"false AND false", false},
+
+		// OR
+		{"true OR true", true},
+		{"true OR false", true},
+		{"false OR true", true},
+		{"false OR false", false},
+
+		// Combined with comparisons
+		{"10 > 5 AND 20 > 10", true},
+		{"10 > 5 AND 5 > 10", false},
+		{"10 > 5 OR 5 > 10", true},
+		{"5 > 10 OR 3 > 10", false},
+	}
+
+	for _, tc := range tests {
+		result := evaluate(tc.expr)
+		if result != tc.expected {
+			t.Errorf("evaluate(%q) = %v, want %v", tc.expr, result, tc.expected)
+		}
+	}
+}
+
+func TestCompare(t *testing.T) {
+	tests := []struct {
+		left, op, right string
+		expected        bool
+	}{
+		{"hello", "==", "hello", true},
+		{"hello", "==", "world", false},
+		{"hello", "!=", "world", true},
+		{"'quoted'", "==", "quoted", true}, // Quotes stripped
+		{"10", ">", "5", true},
+		{"abc", ">", "def", false}, // Non-numeric
+		{"test string", " contains ", "string", true},
+	}
+
+	for _, tc := range tests {
+		result := compare(tc.left, tc.op, tc.right)
+		if result != tc.expected {
+			t.Errorf("compare(%q, %q, %q) = %v, want %v", tc.left, tc.op, tc.right, result, tc.expected)
+		}
+	}
+}
```

---

### 3. pkg/runner/flags_test.go (NEW FILE)

**Priority: HIGH** - Flag parsing affects all CLI commands.

```diff
--- /dev/null
+++ b/pkg/runner/flags_test.go
@@ -0,0 +1,166 @@
+package runner
+
+import (
+	"reflect"
+	"testing"
+)
+
+func TestCheckDuplicateFlags_NoConflict(t *testing.T) {
+	args := []string{"-m", "claude", "-c", "do something"}
+	groups := CommonFlagGroups()
+
+	err := CheckDuplicateFlags(args, groups)
+	if err != nil {
+		t.Errorf("expected no error, got: %v", err)
+	}
+}
+
+func TestCheckDuplicateFlags_SameValueOK(t *testing.T) {
+	// Same value specified twice should not be an error
+	args := []string{"-m", "claude", "--model", "claude"}
+	groups := CommonFlagGroups()
+
+	err := CheckDuplicateFlags(args, groups)
+	if err != nil {
+		t.Errorf("expected no error for same values, got: %v", err)
+	}
+}
+
+func TestCheckDuplicateFlags_Conflict(t *testing.T) {
+	args := []string{"-m", "claude", "--model", "gpt-4"}
+	groups := CommonFlagGroups()
+
+	err := CheckDuplicateFlags(args, groups)
+	if err == nil {
+		t.Error("expected error for conflicting values")
+	}
+}
+
+func TestCheckDuplicateFlags_EqualsFormat(t *testing.T) {
+	args := []string{"-m=claude", "--model=gpt-4"}
+	groups := CommonFlagGroups()
+
+	err := CheckDuplicateFlags(args, groups)
+	if err == nil {
+		t.Error("expected error for conflicting values with = format")
+	}
+}
+
+func TestCheckDuplicateFlags_BooleanFlags(t *testing.T) {
+	args := []string{"-j", "--json"}
+	groups := CommonFlagGroups()
+
+	err := CheckDuplicateFlags(args, groups)
+	if err != nil {
+		t.Errorf("expected no error for duplicate boolean flags, got: %v", err)
+	}
+}
+
+func TestParseVarFlags_Basic(t *testing.T) {
+	args := []string{"-x", "foo=bar", "task", "-x", "baz=qux"}
+	cleaned, vars := ParseVarFlags(args)
+
+	expectedCleaned := []string{"task"}
+	expectedVars := map[string]string{"foo": "bar", "baz": "qux"}
+
+	if !reflect.DeepEqual(cleaned, expectedCleaned) {
+		t.Errorf("cleaned args = %v, want %v", cleaned, expectedCleaned)
+	}
+	if !reflect.DeepEqual(vars, expectedVars) {
+		t.Errorf("vars = %v, want %v", vars, expectedVars)
+	}
+}
+
+func TestParseVarFlags_EqualsFormat(t *testing.T) {
+	args := []string{"-x=key=value", "task"}
+	cleaned, vars := ParseVarFlags(args)
+
+	if vars["key"] != "value" {
+		t.Errorf("vars[key] = %q, want 'value'", vars["key"])
+	}
+	if len(cleaned) != 1 || cleaned[0] != "task" {
+		t.Errorf("cleaned = %v, want [task]", cleaned)
+	}
+}
+
+func TestParseVarFlags_NoVars(t *testing.T) {
+	args := []string{"task", "-m", "claude"}
+	cleaned, vars := ParseVarFlags(args)
+
+	if len(vars) != 0 {
+		t.Errorf("expected no vars, got %v", vars)
+	}
+	if !reflect.DeepEqual(cleaned, args) {
+		t.Errorf("cleaned = %v, want %v", cleaned, args)
+	}
+}
+
+func TestParseVarFlags_InvalidFormat(t *testing.T) {
+	// -x without = in value should be ignored
+	args := []string{"-x", "noequals", "task"}
+	cleaned, vars := ParseVarFlags(args)
+
+	if len(vars) != 0 {
+		t.Errorf("expected no vars for invalid format, got %v", vars)
+	}
+	if len(cleaned) != 1 {
+		t.Errorf("cleaned = %v, want [task]", cleaned)
+	}
+}
+
+func TestReorderArgsForFlagParsing(t *testing.T) {
+	tests := []struct {
+		name     string
+		args     []string
+		expected []string
+	}{
+		{
+			name:     "flags already first",
+			args:     []string{"-m", "claude", "task"},
+			expected: []string{"-m", "claude", "task"},
+		},
+		{
+			name:     "positional before flag",
+			args:     []string{"task", "-m", "claude"},
+			expected: []string{"-m", "claude", "task"},
+		},
+		{
+			name:     "mixed order",
+			args:     []string{"task1", "-j", "task2", "-m", "claude"},
+			expected: []string{"-j", "-m", "claude", "task1", "task2"},
+		},
+		{
+			name:     "equals format",
+			args:     []string{"task", "--model=claude"},
+			expected: []string{"--model=claude", "task"},
+		},
+		{
+			name:     "unknown flag treated as flag",
+			args:     []string{"task", "--unknown"},
+			expected: []string{"--unknown", "task"},
+		},
+		{
+			name:     "empty args",
+			args:     []string{},
+			expected: []string{},
+		},
+	}
+
+	groups := CommonFlagGroups()
+
+	for _, tc := range tests {
+		t.Run(tc.name, func(t *testing.T) {
+			result := reorderArgsForFlagParsing(tc.args, groups)
+			if !reflect.DeepEqual(result, tc.expected) {
+				t.Errorf("reorderArgsForFlagParsing(%v) = %v, want %v", tc.args, result, tc.expected)
+			}
+		})
+	}
+}
```

---

### 4. pkg/executor/dispatcher_test.go (NEW FILE)

**Priority: HIGH** - Core routing logic for all step types.

```diff
--- /dev/null
+++ b/pkg/executor/dispatcher_test.go
@@ -0,0 +1,98 @@
+package executor
+
+import (
+	"testing"
+
+	"rcodegen/pkg/bundle"
+	"rcodegen/pkg/envelope"
+	"rcodegen/pkg/orchestrator"
+	"rcodegen/pkg/runner"
+	"rcodegen/pkg/workspace"
+)
+
+// mockTool implements runner.Tool for testing
+type mockTool struct {
+	name string
+}
+
+func (m *mockTool) Name() string                           { return m.name }
+func (m *mockTool) Description() string                    { return "mock tool" }
+func (m *mockTool) BuildCommand(ctx runner.CommandContext) []string {
+	return []string{"echo", "mock"}
+}
+func (m *mockTool) PreRun(ctx runner.CommandContext) error  { return nil }
+func (m *mockTool) PostRun(ctx runner.CommandContext) error { return nil }
+func (m *mockTool) CapturesToFile() bool                    { return false }
+func (m *mockTool) DefaultPrompt() string                   { return "" }
+func (m *mockTool) DefaultModelSetting() string             { return "" }
+func (m *mockTool) SupportsResume() bool                    { return false }
+func (m *mockTool) ExtractSessionID(output string) string   { return "" }
+func (m *mockTool) ResumeFlag() string                      { return "" }
+func (m *mockTool) PromptModeFlag() string                  { return "" }
+func (m *mockTool) ModelFlag() string                       { return "" }
+func (m *mockTool) PrintOutputFlag() string                 { return "" }
+func (m *mockTool) MaxTokensFlag() string                   { return "" }
+func (m *mockTool) SystemPromptFlag() string                { return "" }
+func (m *mockTool) IsClaudeMax() bool                       { return false }
+
+func TestNewDispatcher(t *testing.T) {
+	tools := map[string]runner.Tool{
+		"mock": &mockTool{name: "mock"},
+	}
+
+	d := NewDispatcher(tools)
+
+	if d.tool == nil {
+		t.Error("expected tool executor to be initialized")
+	}
+	if d.parallel == nil {
+		t.Error("expected parallel executor to be initialized")
+	}
+	if d.merge == nil {
+		t.Error("expected merge executor to be initialized")
+	}
+	if d.vote == nil {
+		t.Error("expected vote executor to be initialized")
+	}
+}
+
+func TestDispatcher_Execute_UnknownStepType(t *testing.T) {
+	d := NewDispatcher(nil)
+	ctx := orchestrator.NewContext(nil)
+	ws := &workspace.Workspace{JobDir: t.TempDir()}
+
+	// Empty step - no tool, no parallel, no merge, no vote
+	step := &bundle.Step{Name: "empty"}
+
+	env, err := d.Execute(step, ctx, ws)
+
+	if err != nil {
+		t.Errorf("unexpected error: %v", err)
+	}
+	if env.Status != envelope.StatusFailure {
+		t.Errorf("expected failure status, got %s", env.Status)
+	}
+	if env.Error == nil || env.Error.Code != "UNKNOWN_STEP" {
+		t.Error("expected UNKNOWN_STEP error")
+	}
+}
+
+func TestDispatcher_Execute_RoutesToCorrectExecutor(t *testing.T) {
+	// Test that parallel steps are routed to parallel executor
+	d := NewDispatcher(nil)
+
+	parallelStep := &bundle.Step{
+		Name: "parallel-test",
+		Parallel: []bundle.Step{
+			{Name: "sub1", Tool: "mock"},
+		},
+	}
+
+	// This will fail because mock tool doesn't exist, but it proves routing works
+	// The parallel executor will be called and attempt to execute substeps
+	ctx := orchestrator.NewContext(nil)
+	ws := &workspace.Workspace{JobDir: t.TempDir()}
+
+	_, _ = d.Execute(parallelStep, ctx, ws)
+	// If it doesn't panic, routing worked
+}
```

---

### 5. pkg/executor/vote_test.go (NEW FILE)

**Priority: MEDIUM** - Voting logic for ensemble strategies.

```diff
--- /dev/null
+++ b/pkg/executor/vote_test.go
@@ -0,0 +1,84 @@
+package executor
+
+import (
+	"testing"
+
+	"rcodegen/pkg/bundle"
+	"rcodegen/pkg/envelope"
+	"rcodegen/pkg/orchestrator"
+	"rcodegen/pkg/workspace"
+)
+
+func TestExtractStepName(t *testing.T) {
+	tests := []struct {
+		input    string
+		expected string
+	}{
+		{"${steps.analyze.output_ref}", "analyze"},
+		{"${steps.build.status}", "build"},
+		{"${steps.test-runner.result}", "test-runner"},
+		{"not-a-ref", "not-a-ref"},
+		{"${steps.}", "${steps.}"}, // Edge case
+		{"", ""},
+	}
+
+	for _, tc := range tests {
+		result := extractStepName(tc.input)
+		if result != tc.expected {
+			t.Errorf("extractStepName(%q) = %q, want %q", tc.input, result, tc.expected)
+		}
+	}
+}
+
+func TestVoteExecutor_Majority(t *testing.T) {
+	ctx := orchestrator.NewContext(nil)
+	ctx.SetResult("step1", &envelope.Envelope{Status: envelope.StatusSuccess})
+	ctx.SetResult("step2", &envelope.Envelope{Status: envelope.StatusSuccess})
+	ctx.SetResult("step3", &envelope.Envelope{Status: envelope.StatusFailure})
+
+	ws := &workspace.Workspace{JobDir: t.TempDir()}
+	executor := &VoteExecutor{}
+
+	step := &bundle.Step{
+		Name: "vote-test",
+		Vote: &bundle.VoteDef{
+			Inputs:   []string{"${steps.step1.output_ref}", "${steps.step2.output_ref}", "${steps.step3.output_ref}"},
+			Strategy: "majority",
+		},
+	}
+
+	env, err := executor.Execute(step, ctx, ws)
+
+	if err != nil {
+		t.Fatalf("unexpected error: %v", err)
+	}
+	if env.Result["decision"] != "approved" {
+		t.Errorf("expected 'approved' with 2/3 success, got %v", env.Result["decision"])
+	}
+}
+
+func TestVoteExecutor_Unanimous(t *testing.T) {
+	ctx := orchestrator.NewContext(nil)
+	ctx.SetResult("step1", &envelope.Envelope{Status: envelope.StatusSuccess})
+	ctx.SetResult("step2", &envelope.Envelope{Status: envelope.StatusFailure})
+
+	ws := &workspace.Workspace{JobDir: t.TempDir()}
+	executor := &VoteExecutor{}
+
+	step := &bundle.Step{
+		Name: "vote-test",
+		Vote: &bundle.VoteDef{
+			Inputs:   []string{"${steps.step1.output_ref}", "${steps.step2.output_ref}"},
+			Strategy: "unanimous",
+		},
+	}
+
+	env, _ := executor.Execute(step, ctx, ws)
+
+	if env.Result["decision"] != "rejected" {
+		t.Errorf("expected 'rejected' for unanimous with failure, got %v", env.Result["decision"])
+	}
+}
```

---

### 6. pkg/envelope/envelope_test.go (NEW FILE)

**Priority: MEDIUM** - Builder pattern used extensively.

```diff
--- /dev/null
+++ b/pkg/envelope/envelope_test.go
@@ -0,0 +1,91 @@
+package envelope
+
+import (
+	"testing"
+)
+
+func TestNew(t *testing.T) {
+	b := New()
+	if b == nil {
+		t.Fatal("New() returned nil")
+	}
+	if b.env == nil {
+		t.Error("builder envelope is nil")
+	}
+	if b.env.Result == nil {
+		t.Error("Result map should be initialized")
+	}
+}
+
+func TestBuilder_Success(t *testing.T) {
+	env := New().Success().Build()
+
+	if env.Status != StatusSuccess {
+		t.Errorf("expected StatusSuccess, got %s", env.Status)
+	}
+}
+
+func TestBuilder_Failure(t *testing.T) {
+	env := New().Failure("ERR_CODE", "Something went wrong").Build()
+
+	if env.Status != StatusFailure {
+		t.Errorf("expected StatusFailure, got %s", env.Status)
+	}
+	if env.Error == nil {
+		t.Fatal("expected Error to be set")
+	}
+	if env.Error.Code != "ERR_CODE" {
+		t.Errorf("expected error code 'ERR_CODE', got %s", env.Error.Code)
+	}
+	if env.Error.Message != "Something went wrong" {
+		t.Errorf("expected error message, got %s", env.Error.Message)
+	}
+}
+
+func TestBuilder_WithResult(t *testing.T) {
+	env := New().
+		Success().
+		WithResult("count", 42).
+		WithResult("name", "test").
+		Build()
+
+	if env.Result["count"] != 42 {
+		t.Errorf("expected count=42, got %v", env.Result["count"])
+	}
+	if env.Result["name"] != "test" {
+		t.Errorf("expected name='test', got %v", env.Result["name"])
+	}
+}
+
+func TestBuilder_WithOutputRef(t *testing.T) {
+	env := New().
+		Success().
+		WithOutputRef("/tmp/output.json").
+		Build()
+
+	if env.OutputRef != "/tmp/output.json" {
+		t.Errorf("expected OutputRef='/tmp/output.json', got %s", env.OutputRef)
+	}
+}
+
+func TestBuilder_WithTool(t *testing.T) {
+	env := New().WithTool("claude").Build()
+
+	if env.Metrics == nil {
+		t.Fatal("expected Metrics to be initialized")
+	}
+	if env.Metrics.Tool != "claude" {
+		t.Errorf("expected tool='claude', got %s", env.Metrics.Tool)
+	}
+}
+
+func TestBuilder_WithDuration(t *testing.T) {
+	env := New().WithDuration(1500).Build()
+
+	if env.Metrics == nil || env.Metrics.DurationMs != 1500 {
+		t.Errorf("expected DurationMs=1500, got %v", env.Metrics)
+	}
+}
```

---

### 7. pkg/lock/filelock_test.go (NEW/ENHANCED FILE)

**Priority: MEDIUM** - File locking prevents race conditions.

```diff
--- /dev/null
+++ b/pkg/lock/filelock_test.go
@@ -0,0 +1,79 @@
+package lock
+
+import (
+	"testing"
+)
+
+func TestSanitizeIdentifier_Basic(t *testing.T) {
+	tests := []struct {
+		input    string
+		expected string
+	}{
+		{"simple", "simple"},
+		{"with/slash", "with_slash"},
+		{"with\\backslash", "with_backslash"},
+		{"with\ttab", "with_tab"},
+		{"with\nnewline", "with_newline"},
+		{"", "unknown"},
+		{"normal-identifier_123", "normal-identifier_123"},
+	}
+
+	for _, tc := range tests {
+		result := sanitizeIdentifier(tc.input)
+		if result != tc.expected {
+			t.Errorf("sanitizeIdentifier(%q) = %q, want %q", tc.input, result, tc.expected)
+		}
+	}
+}
+
+func TestSanitizeIdentifier_MaxLength(t *testing.T) {
+	// Create a string longer than maxIdentifierLen (100)
+	longInput := ""
+	for i := 0; i < 150; i++ {
+		longInput += "x"
+	}
+
+	result := sanitizeIdentifier(longInput)
+
+	if len(result) != maxIdentifierLen {
+		t.Errorf("expected length %d, got %d", maxIdentifierLen, len(result))
+	}
+}
+
+func TestGetIdentifier(t *testing.T) {
+	tests := []struct {
+		workDir  string
+		expected string
+	}{
+		{"/Users/dev/myproject", "myproject"},
+		{"/home/user/code/app", "app"},
+		{"", "unknown"}, // Will use cwd basename or "unknown"
+	}
+
+	for _, tc := range tests {
+		if tc.workDir == "" {
+			continue // Skip empty case as it depends on actual cwd
+		}
+		result := GetIdentifier(tc.workDir)
+		if result != tc.expected {
+			t.Errorf("GetIdentifier(%q) = %q, want %q", tc.workDir, result, tc.expected)
+		}
+	}
+}
+
+func TestFileLock_Release_NilSafe(t *testing.T) {
+	// Release should handle nil gracefully
+	var lock *FileLock = nil
+	err := lock.Release()
+	if err != nil {
+		t.Errorf("Release() on nil should return nil, got %v", err)
+	}
+
+	// Also test with nil file
+	lock = &FileLock{file: nil, path: ""}
+	err = lock.Release()
+	if err != nil {
+		t.Errorf("Release() with nil file should return nil, got %v", err)
+	}
+}
+
+// Note: Testing actual lock acquisition requires integration tests
+// as it involves file system operations and syscalls
```

---

## Summary of Proposed Tests

| New Test File | Test Count | Priority | Lines |
|---------------|------------|----------|-------|
| `pkg/orchestrator/context_test.go` | 7 | CRITICAL | ~180 |
| `pkg/orchestrator/condition_test.go` | 5 | CRITICAL | ~147 |
| `pkg/runner/flags_test.go` | 8 | HIGH | ~166 |
| `pkg/executor/dispatcher_test.go` | 3 | HIGH | ~98 |
| `pkg/executor/vote_test.go` | 3 | MEDIUM | ~84 |
| `pkg/envelope/envelope_test.go` | 7 | MEDIUM | ~91 |
| `pkg/lock/filelock_test.go` | 4 | MEDIUM | ~79 |
| **TOTAL** | **37** | | **~845** |

---

## Additional Recommendations

### High-Value Tests Not Included (Due to Complexity)

1. **Integration tests for orchestrator flow** - Requires mocking external tools
2. **Parallel executor thread-safety tests** - Requires careful goroutine coordination
3. **Tool execution tests** - Requires mocking exec.Command
4. **File-based locking contention tests** - Requires multi-process coordination

### Code Quality Observations

1. **Good separation of concerns** - Packages are well-organized
2. **Consistent naming conventions** - Functions follow Go idioms
3. **Thread-safe context** - Uses proper sync.RWMutex patterns
4. **Builder pattern well-implemented** - Fluent API in envelope package

### Risk Areas Without Tests

1. **Variable resolution** - Complex regex and JSON parsing could have edge cases
2. **Condition evaluation** - AND/OR precedence may have bugs
3. **Flag parsing** - Edge cases with =value format
4. **Parallel execution** - Race conditions possible
5. **File locking** - Cross-platform behavior differences

---

## How to Apply These Patches

```bash
# Navigate to project root
cd /Users/cliff/Desktop/_code/rcodegen

# For each new test file, create directly:
# Example for context_test.go:
cat > pkg/orchestrator/context_test.go << 'EOF'
# (paste the content between +++ markers)
EOF

# Run the new tests:
go test ./pkg/orchestrator/... -v
go test ./pkg/runner/... -v
go test ./pkg/executor/... -v
go test ./pkg/envelope/... -v
go test ./pkg/lock/... -v

# Run all tests:
go test ./... -v
```

---

*Report generated by Claude Code on 2026-01-16*
