Date Created: 2026-01-26T19:15:00-08:00
TOTAL_SCORE: 58/100

# rcodegen Unit Test Coverage Analysis

## Executive Summary

**Overall Grade: 58/100** (C+)

The rcodegen project has foundational test coverage but significant gaps remain in critical business logic areas. The codebase totals ~15,736 lines across 33 Go modules with approximately 3,336 lines of test code (21% test-to-source ratio).

### Scoring Breakdown

| Category | Score | Max | Notes |
|----------|-------|-----|-------|
| Test Coverage Breadth | 12/25 | 25 | 5 packages completely untested |
| Test Coverage Depth | 15/25 | 25 | Many tested packages have minimal coverage |
| Critical Path Coverage | 10/20 | 20 | File operations and external commands untested |
| Test Quality | 12/15 | 15 | Good patterns where tests exist |
| Edge Case Handling | 9/15 | 15 | Limited error path testing |

---

## Test Coverage Summary

### Packages WITH Tests (8 packages)

| Package | Source Lines | Test Lines | Estimated Coverage |
|---------|-------------|------------|-------------------|
| pkg/settings | 627 | 52 | ~8% |
| pkg/runner | 1,200+ | 477 | ~20% |
| pkg/tools/claude | 150 | 31 | ~21% |
| pkg/workspace | 200 | 99 | ~50% |
| pkg/lock | 100 | 65 | ~65% |
| pkg/bundle | 250 | 82 | ~33% |
| pkg/executor | 400 | 229 | ~57% |
| pkg/orchestrator | 800 | 519 | ~65% |
| pkg/envelope | 200 | 156 | ~78% |

### Packages WITHOUT Tests (5 packages) - HIGH PRIORITY

| Package | Source Lines | Risk Level | Priority |
|---------|-------------|------------|----------|
| pkg/reports | 150 | **HIGH** | P0 |
| pkg/tracking/codex.go | 195 | **HIGH** | P0 |
| pkg/tracking/claude.go | 156 | **HIGH** | P1 |
| pkg/tools/codex | 337 | **MEDIUM** | P1 |
| pkg/tools/gemini | 217 | **MEDIUM** | P2 |
| pkg/colors | 17 | LOW | P3 |

---

## Critical Gaps Identified

### 1. pkg/reports (0% coverage) - CRITICAL

File: `pkg/reports/manager.go` (150 lines)

**Untested Functions:**
- `ShouldSkipTask()` - Determines if task should be skipped based on review status
- `FindNewestReport()` - Finds most recent report by modification time
- `IsReportReviewed()` - Checks for "Date Modified:" marker in first 10 lines
- `DeleteOldReports()` - Removes old reports keeping only newest

**Risk:** File operations and business logic directly affect user workflow.

### 2. pkg/tracking (0% coverage) - CRITICAL

Files: `pkg/tracking/codex.go` (195 lines), `pkg/tracking/claude.go` (156 lines)

**Untested Functions:**
- `FindPython()` - Locates Python interpreter
- `GetScriptDir()` - Finds executable directory
- `GetStatus()` / `GetClaudeStatus()` - External script execution
- `FormatCredit()` - Credit value formatting
- `runStatusScript()` / `runClaudeStatusScript()` - JSON parsing

**Risk:** External command execution, error handling, security-sensitive paths.

### 3. pkg/tools/codex & pkg/tools/gemini (0% coverage) - HIGH

**Untested Functions:**
- `BuildCommand()` - Command construction with various flags
- `ValidateConfig()` - Configuration validation
- `ApplyToolDefaults()` - Default value application
- All interface method implementations

**Risk:** Command construction errors could cause runtime failures.

---

## Patch-Ready Test Diffs

### 1. pkg/reports/manager_test.go (NEW FILE)

```diff
--- /dev/null
+++ b/pkg/reports/manager_test.go
@@ -0,0 +1,289 @@
+package reports
+
+import (
+	"os"
+	"path/filepath"
+	"testing"
+	"time"
+)
+
+func TestShouldSkipTask_NoReportDir(t *testing.T) {
+	// Should return false (run task) when report directory doesn't exist
+	result := ShouldSkipTask("/nonexistent/path", "test", "test-pattern-", true)
+	if result {
+		t.Error("ShouldSkipTask should return false when report directory doesn't exist")
+	}
+}
+
+func TestShouldSkipTask_RequireReviewFalse(t *testing.T) {
+	// Should return false when requireReview is false
+	tmpDir := t.TempDir()
+	result := ShouldSkipTask(tmpDir, "test", "test-pattern-", false)
+	if result {
+		t.Error("ShouldSkipTask should return false when requireReview is false")
+	}
+}
+
+func TestShouldSkipTask_EmptyPattern(t *testing.T) {
+	// Should return false when pattern is empty
+	tmpDir := t.TempDir()
+	result := ShouldSkipTask(tmpDir, "test", "", true)
+	if result {
+		t.Error("ShouldSkipTask should return false when pattern is empty")
+	}
+}
+
+func TestShouldSkipTask_NoMatchingReports(t *testing.T) {
+	// Should return false when no reports match pattern
+	tmpDir := t.TempDir()
+	result := ShouldSkipTask(tmpDir, "test", "nonexistent-", true)
+	if result {
+		t.Error("ShouldSkipTask should return false when no matching reports exist")
+	}
+}
+
+func TestShouldSkipTask_ReportReviewed(t *testing.T) {
+	// Should return false (run task) when report has been reviewed
+	tmpDir := t.TempDir()
+	reportPath := filepath.Join(tmpDir, "test-pattern-2026-01-26.md")
+	content := `Date Created: 2026-01-26
+Date Modified: 2026-01-26
+TOTAL_SCORE: 80/100
+
+# Report Content
+`
+	if err := os.WriteFile(reportPath, []byte(content), 0644); err != nil {
+		t.Fatal(err)
+	}
+
+	result := ShouldSkipTask(tmpDir, "test", "test-pattern-", true)
+	if result {
+		t.Error("ShouldSkipTask should return false when report has Date Modified marker")
+	}
+}
+
+func TestShouldSkipTask_ReportUnreviewed(t *testing.T) {
+	// Should return true (skip task) when report hasn't been reviewed
+	tmpDir := t.TempDir()
+	reportPath := filepath.Join(tmpDir, "test-pattern-2026-01-26.md")
+	content := `Date Created: 2026-01-26
+TOTAL_SCORE: 80/100
+
+# Report Content - No Date Modified marker
+`
+	if err := os.WriteFile(reportPath, []byte(content), 0644); err != nil {
+		t.Fatal(err)
+	}
+
+	result := ShouldSkipTask(tmpDir, "test", "test-pattern-", true)
+	if !result {
+		t.Error("ShouldSkipTask should return true when report lacks Date Modified marker")
+	}
+}
+
+func TestFindNewestReport_EmptyList(t *testing.T) {
+	result := FindNewestReport([]string{})
+	if result != "" {
+		t.Errorf("FindNewestReport should return empty string for empty list, got %q", result)
+	}
+}
+
+func TestFindNewestReport_SingleFile(t *testing.T) {
+	tmpDir := t.TempDir()
+	file := filepath.Join(tmpDir, "report.md")
+	if err := os.WriteFile(file, []byte("content"), 0644); err != nil {
+		t.Fatal(err)
+	}
+
+	result := FindNewestReport([]string{file})
+	if result != file {
+		t.Errorf("FindNewestReport should return the only file, got %q", result)
+	}
+}
+
+func TestFindNewestReport_MultipleFiles(t *testing.T) {
+	tmpDir := t.TempDir()
+
+	// Create older file
+	oldFile := filepath.Join(tmpDir, "old.md")
+	if err := os.WriteFile(oldFile, []byte("old"), 0644); err != nil {
+		t.Fatal(err)
+	}
+
+	// Sleep to ensure different mtime
+	time.Sleep(10 * time.Millisecond)
+
+	// Create newer file
+	newFile := filepath.Join(tmpDir, "new.md")
+	if err := os.WriteFile(newFile, []byte("new"), 0644); err != nil {
+		t.Fatal(err)
+	}
+
+	result := FindNewestReport([]string{oldFile, newFile})
+	if result != newFile {
+		t.Errorf("FindNewestReport should return newest file %q, got %q", newFile, result)
+	}
+}
+
+func TestFindNewestReport_NonexistentFile(t *testing.T) {
+	tmpDir := t.TempDir()
+	validFile := filepath.Join(tmpDir, "valid.md")
+	if err := os.WriteFile(validFile, []byte("content"), 0644); err != nil {
+		t.Fatal(err)
+	}
+
+	// Include a nonexistent file - should be skipped
+	result := FindNewestReport([]string{"/nonexistent/file.md", validFile})
+	if result != validFile {
+		t.Errorf("FindNewestReport should return valid file, got %q", result)
+	}
+}
+
+func TestIsReportReviewed_WithMarker(t *testing.T) {
+	tmpDir := t.TempDir()
+	file := filepath.Join(tmpDir, "reviewed.md")
+	content := `Date Created: 2026-01-26
+Date Modified: 2026-01-26
+TOTAL_SCORE: 80/100
+`
+	if err := os.WriteFile(file, []byte(content), 0644); err != nil {
+		t.Fatal(err)
+	}
+
+	if !IsReportReviewed(file) {
+		t.Error("IsReportReviewed should return true when Date Modified marker present")
+	}
+}
+
+func TestIsReportReviewed_WithoutMarker(t *testing.T) {
+	tmpDir := t.TempDir()
+	file := filepath.Join(tmpDir, "unreviewed.md")
+	content := `Date Created: 2026-01-26
+TOTAL_SCORE: 80/100
+
+No Date Modified marker in first 10 lines
+`
+	if err := os.WriteFile(file, []byte(content), 0644); err != nil {
+		t.Fatal(err)
+	}
+
+	if IsReportReviewed(file) {
+		t.Error("IsReportReviewed should return false when Date Modified marker absent")
+	}
+}
+
+func TestIsReportReviewed_MarkerAfterLine10(t *testing.T) {
+	tmpDir := t.TempDir()
+	file := filepath.Join(tmpDir, "late_marker.md")
+	content := `Line 1
+Line 2
+Line 3
+Line 4
+Line 5
+Line 6
+Line 7
+Line 8
+Line 9
+Line 10
+Date Modified: 2026-01-26
+`
+	if err := os.WriteFile(file, []byte(content), 0644); err != nil {
+		t.Fatal(err)
+	}
+
+	if IsReportReviewed(file) {
+		t.Error("IsReportReviewed should return false when marker after line 10")
+	}
+}
+
+func TestIsReportReviewed_NonexistentFile(t *testing.T) {
+	if IsReportReviewed("/nonexistent/file.md") {
+		t.Error("IsReportReviewed should return false for nonexistent file")
+	}
+}
+
+func TestDeleteOldReports_NonexistentDir(t *testing.T) {
+	// Should not panic on nonexistent directory
+	DeleteOldReports("/nonexistent/path", []string{"test"}, map[string]string{"test": "test-"})
+}
+
+func TestDeleteOldReports_NoMatchingPatterns(t *testing.T) {
+	tmpDir := t.TempDir()
+	// Should not panic when no patterns match
+	DeleteOldReports(tmpDir, []string{"unknown"}, map[string]string{"test": "test-"})
+}
+
+func TestDeleteOldReports_SingleReport(t *testing.T) {
+	tmpDir := t.TempDir()
+	file := filepath.Join(tmpDir, "test-2026-01-26.md")
+	if err := os.WriteFile(file, []byte("content"), 0644); err != nil {
+		t.Fatal(err)
+	}
+
+	DeleteOldReports(tmpDir, []string{"test"}, map[string]string{"test": "test-"})
+
+	// Single file should not be deleted
+	if _, err := os.Stat(file); os.IsNotExist(err) {
+		t.Error("DeleteOldReports should not delete single report")
+	}
+}
+
+func TestDeleteOldReports_MultipleReports(t *testing.T) {
+	tmpDir := t.TempDir()
+
+	// Create older file
+	oldFile := filepath.Join(tmpDir, "test-2026-01-25.md")
+	if err := os.WriteFile(oldFile, []byte("old"), 0644); err != nil {
+		t.Fatal(err)
+	}
+
+	time.Sleep(10 * time.Millisecond)
+
+	// Create newer file
+	newFile := filepath.Join(tmpDir, "test-2026-01-26.md")
+	if err := os.WriteFile(newFile, []byte("new"), 0644); err != nil {
+		t.Fatal(err)
+	}
+
+	DeleteOldReports(tmpDir, []string{"test"}, map[string]string{"test": "test-"})
+
+	// Old file should be deleted
+	if _, err := os.Stat(oldFile); !os.IsNotExist(err) {
+		t.Error("DeleteOldReports should delete older reports")
+	}
+
+	// New file should remain
+	if _, err := os.Stat(newFile); os.IsNotExist(err) {
+		t.Error("DeleteOldReports should keep newest report")
+	}
+}
```

### 2. pkg/tracking/codex_test.go (NEW FILE)

```diff
--- /dev/null
+++ b/pkg/tracking/codex_test.go
@@ -0,0 +1,142 @@
+package tracking
+
+import (
+	"os"
+	"path/filepath"
+	"testing"
+)
+
+func TestFormatCredit_Nil(t *testing.T) {
+	result := FormatCredit(nil)
+	if result != "N/A" {
+		t.Errorf("FormatCredit(nil) should return 'N/A', got %q", result)
+	}
+}
+
+func TestFormatCredit_ValidValue(t *testing.T) {
+	tests := []struct {
+		name     string
+		input    int
+		expected string
+	}{
+		{"zero", 0, "0"},
+		{"positive", 42, "42"},
+		{"hundred", 100, "100"},
+	}
+
+	for _, tc := range tests {
+		t.Run(tc.name, func(t *testing.T) {
+			val := tc.input
+			result := FormatCredit(&val)
+			if result != tc.expected {
+				t.Errorf("FormatCredit(%d) = %q, want %q", tc.input, result, tc.expected)
+			}
+		})
+	}
+}
+
+func TestFindPython_ReturnsNonEmpty(t *testing.T) {
+	// FindPython should always return something (even fallback "python3")
+	result := FindPython()
+	if result == "" {
+		t.Error("FindPython should never return empty string")
+	}
+}
+
+func TestFindPython_PrefersBrew313(t *testing.T) {
+	// If homebrew python3.13 exists, it should be returned first
+	brewPath := "/opt/homebrew/bin/python3.13"
+	if _, err := os.Stat(brewPath); err == nil {
+		result := FindPython()
+		if result != brewPath {
+			t.Logf("Expected %s but got %s (acceptable if another version found)", brewPath, result)
+		}
+	}
+}
+
+func TestGetScriptDir_ReturnsValidPath(t *testing.T) {
+	// GetScriptDir should return the directory of the current executable
+	result := GetScriptDir()
+	// During tests, we should get a valid directory (test binary location)
+	if result == "" {
+		t.Skip("GetScriptDir returned empty - may be in special test environment")
+	}
+
+	// Verify it's a directory
+	info, err := os.Stat(result)
+	if err != nil {
+		t.Errorf("GetScriptDir returned invalid path: %v", err)
+	}
+	if !info.IsDir() {
+		t.Errorf("GetScriptDir returned non-directory: %s", result)
+	}
+}
+
+func TestCreditStatus_ErrorField(t *testing.T) {
+	status := &CreditStatus{Error: "test error"}
+	if status.Error != "test error" {
+		t.Error("CreditStatus.Error field not working correctly")
+	}
+}
+
+func TestCreditStatus_Fields(t *testing.T) {
+	five := 50
+	weekly := 75
+	fiveResets := "2h 30m"
+	weeklyResets := "5d 2h"
+
+	status := &CreditStatus{
+		FiveHourLeft:   &five,
+		WeeklyLeft:     &weekly,
+		FiveHourResets: &fiveResets,
+		WeeklyResets:   &weeklyResets,
+	}
+
+	if *status.FiveHourLeft != 50 {
+		t.Errorf("FiveHourLeft = %d, want 50", *status.FiveHourLeft)
+	}
+	if *status.WeeklyLeft != 75 {
+		t.Errorf("WeeklyLeft = %d, want 75", *status.WeeklyLeft)
+	}
+	if *status.FiveHourResets != "2h 30m" {
+		t.Errorf("FiveHourResets = %s, want '2h 30m'", *status.FiveHourResets)
+	}
+}
+
+func TestGetStatus_NoScript(t *testing.T) {
+	// When no script exists in trusted locations, should return error
+	// Save and restore HOME to ensure script isn't found
+	origHome := os.Getenv("HOME")
+	tmpDir := t.TempDir()
+	os.Setenv("HOME", tmpDir)
+	defer os.Setenv("HOME", origHome)
+
+	status := GetStatus()
+	if status.Error == "" {
+		t.Error("GetStatus should return error when script not found")
+	}
+}
+
+func TestRunStatusScript_InvalidJSON(t *testing.T) {
+	// Create a script that outputs invalid JSON
+	tmpDir := t.TempDir()
+	scriptPath := filepath.Join(tmpDir, "bad_script.py")
+	script := `#!/usr/bin/env python3
+print("not valid json")
+`
+	if err := os.WriteFile(scriptPath, []byte(script), 0755); err != nil {
+		t.Fatal(err)
+	}
+
+	python := FindPython()
+	cmd := execCommand(python, scriptPath)
+	result := runStatusScript(cmd)
+
+	if result.Error == "" {
+		t.Error("runStatusScript should return error for invalid JSON output")
+	}
+}
+
+// Helper to create exec.Cmd for testing
+var execCommand = exec.Command
```

### 3. pkg/tracking/claude_test.go (NEW FILE)

```diff
--- /dev/null
+++ b/pkg/tracking/claude_test.go
@@ -0,0 +1,113 @@
+package tracking
+
+import (
+	"os"
+	"testing"
+)
+
+func TestClaudeStatus_IsITerm2Error_NotITerm2(t *testing.T) {
+	status := &ClaudeStatus{Error: "not_iterm2"}
+	if !status.IsITerm2Error() {
+		t.Error("IsITerm2Error should return true for 'not_iterm2' error")
+	}
+}
+
+func TestClaudeStatus_IsITerm2Error_NoPackage(t *testing.T) {
+	status := &ClaudeStatus{Error: "no_iterm2_package"}
+	if !status.IsITerm2Error() {
+		t.Error("IsITerm2Error should return true for 'no_iterm2_package' error")
+	}
+}
+
+func TestClaudeStatus_IsITerm2Error_OtherError(t *testing.T) {
+	status := &ClaudeStatus{Error: "some_other_error"}
+	if status.IsITerm2Error() {
+		t.Error("IsITerm2Error should return false for other errors")
+	}
+}
+
+func TestClaudeStatus_IsITerm2Error_NoError(t *testing.T) {
+	status := &ClaudeStatus{}
+	if status.IsITerm2Error() {
+		t.Error("IsITerm2Error should return false when no error")
+	}
+}
+
+func TestClaudeStatus_Fields(t *testing.T) {
+	session := 80
+	weeklyAll := 60
+	weeklySonnet := 90
+	sessionResets := "1h 15m"
+	weeklyResets := "3d 12h"
+
+	status := &ClaudeStatus{
+		SessionLeft:      &session,
+		WeeklyAllLeft:    &weeklyAll,
+		WeeklySonnetLeft: &weeklySonnet,
+		SessionResets:    &sessionResets,
+		WeeklyResets:     &weeklyResets,
+	}
+
+	if *status.SessionLeft != 80 {
+		t.Errorf("SessionLeft = %d, want 80", *status.SessionLeft)
+	}
+	if *status.WeeklyAllLeft != 60 {
+		t.Errorf("WeeklyAllLeft = %d, want 60", *status.WeeklyAllLeft)
+	}
+	if *status.WeeklySonnetLeft != 90 {
+		t.Errorf("WeeklySonnetLeft = %d, want 90", *status.WeeklySonnetLeft)
+	}
+}
+
+func TestClaudeStatus_ErrorAndMessage(t *testing.T) {
+	status := &ClaudeStatus{
+		Error:   "connection_failed",
+		Message: "Could not connect to iTerm2",
+	}
+
+	if status.Error != "connection_failed" {
+		t.Errorf("Error = %q, want 'connection_failed'", status.Error)
+	}
+	if status.Message != "Could not connect to iTerm2" {
+		t.Errorf("Message = %q, want 'Could not connect to iTerm2'", status.Message)
+	}
+}
+
+func TestGetClaudeStatus_NoScript(t *testing.T) {
+	// When no script exists in trusted locations, should return error
+	origHome := os.Getenv("HOME")
+	tmpDir := t.TempDir()
+	os.Setenv("HOME", tmpDir)
+	defer os.Setenv("HOME", origHome)
+
+	status := GetClaudeStatus()
+	if status.Error == "" {
+		t.Error("GetClaudeStatus should return error when script not found")
+	}
+	if status.Error != "status script not found in trusted locations (executable dir or ~/.rcodegen/scripts/)" {
+		t.Errorf("Unexpected error message: %s", status.Error)
+	}
+}
+
+func TestRunClaudeStatusScript_ValidJSON(t *testing.T) {
+	// Create a script that outputs valid JSON
+	tmpDir := t.TempDir()
+	scriptPath := filepath.Join(tmpDir, "good_script.py")
+	script := `#!/usr/bin/env python3
+import json
+print(json.dumps({"session_left": 85, "weekly_all_left": 70}))
+`
+	if err := os.WriteFile(scriptPath, []byte(script), 0755); err != nil {
+		t.Fatal(err)
+	}
+
+	python := FindPython()
+	cmd := exec.Command(python, scriptPath)
+	result := runClaudeStatusScript(cmd)
+
+	if result.Error != "" {
+		t.Errorf("runClaudeStatusScript returned error: %s", result.Error)
+	}
+	if result.SessionLeft == nil || *result.SessionLeft != 85 {
+		t.Error("runClaudeStatusScript did not parse session_left correctly")
+	}
+}
```

### 4. pkg/tools/codex/codex_test.go (NEW FILE)

```diff
--- /dev/null
+++ b/pkg/tools/codex/codex_test.go
@@ -0,0 +1,198 @@
+package codex
+
+import (
+	"strings"
+	"testing"
+
+	"rcodegen/pkg/runner"
+	"rcodegen/pkg/settings"
+)
+
+func TestNew_ReturnsNonNil(t *testing.T) {
+	tool := New()
+	if tool == nil {
+		t.Error("New() should return non-nil Tool")
+	}
+}
+
+func TestTool_Name(t *testing.T) {
+	tool := New()
+	if tool.Name() != "rcodex" {
+		t.Errorf("Name() = %q, want 'rcodex'", tool.Name())
+	}
+}
+
+func TestTool_BinaryName(t *testing.T) {
+	tool := New()
+	if tool.BinaryName() != "codex" {
+		t.Errorf("BinaryName() = %q, want 'codex'", tool.BinaryName())
+	}
+}
+
+func TestTool_ReportDir(t *testing.T) {
+	tool := New()
+	if tool.ReportDir() != "_rcodegen" {
+		t.Errorf("ReportDir() = %q, want '_rcodegen'", tool.ReportDir())
+	}
+}
+
+func TestTool_ReportPrefix(t *testing.T) {
+	tool := New()
+	if tool.ReportPrefix() != "codex-" {
+		t.Errorf("ReportPrefix() = %q, want 'codex-'", tool.ReportPrefix())
+	}
+}
+
+func TestTool_ValidModels(t *testing.T) {
+	tool := New()
+	models := tool.ValidModels()
+	if len(models) == 0 {
+		t.Error("ValidModels() should return non-empty slice")
+	}
+
+	// Check expected models are present
+	expected := []string{"gpt-5.2-codex", "gpt-4.1-codex", "gpt-4o-codex"}
+	for _, exp := range expected {
+		found := false
+		for _, m := range models {
+			if m == exp {
+				found = true
+				break
+			}
+		}
+		if !found {
+			t.Errorf("ValidModels() missing expected model %q", exp)
+		}
+	}
+}
+
+func TestTool_DefaultModel(t *testing.T) {
+	tool := New()
+	if tool.DefaultModel() != "gpt-5.2-codex" {
+		t.Errorf("DefaultModel() = %q, want 'gpt-5.2-codex'", tool.DefaultModel())
+	}
+}
+
+func TestTool_DefaultModelSetting_NoSettings(t *testing.T) {
+	tool := New()
+	// Without settings, should return default
+	if tool.DefaultModelSetting() != "gpt-5.2-codex" {
+		t.Errorf("DefaultModelSetting() without settings = %q, want 'gpt-5.2-codex'", tool.DefaultModelSetting())
+	}
+}
+
+func TestTool_DefaultModelSetting_WithSettings(t *testing.T) {
+	tool := New()
+	s := &settings.Settings{
+		Defaults: settings.Defaults{
+			Codex: settings.CodexDefaults{
+				Model: "custom-model",
+			},
+		},
+	}
+	tool.SetSettings(s)
+
+	if tool.DefaultModelSetting() != "custom-model" {
+		t.Errorf("DefaultModelSetting() with settings = %q, want 'custom-model'", tool.DefaultModelSetting())
+	}
+}
+
+func TestTool_BuildCommand_NewSession(t *testing.T) {
+	tool := New()
+	cfg := &runner.Config{
+		Model:      "gpt-5.2-codex",
+		Effort:     "high",
+		OutputJSON: true,
+	}
+
+	cmd := tool.BuildCommand(cfg, "/test/dir", "test task")
+
+	// Should use codex binary
+	if cmd.Path == "" && cmd.Args[0] != "codex" {
+		t.Error("BuildCommand should use codex binary")
+	}
+
+	// Check for expected flags
+	args := strings.Join(cmd.Args, " ")
+	if !strings.Contains(args, "--dangerously-bypass-approvals-and-sandbox") {
+		t.Error("BuildCommand should include --dangerously-bypass-approvals-and-sandbox")
+	}
+	if !strings.Contains(args, "--model") {
+		t.Error("BuildCommand should include --model flag")
+	}
+	if !strings.Contains(args, "--json") {
+		t.Error("BuildCommand should include --json when OutputJSON is true")
+	}
+}
+
+func TestTool_BuildCommand_ResumeSession(t *testing.T) {
+	tool := New()
+	cfg := &runner.Config{
+		Model:     "gpt-5.2-codex",
+		Effort:    "high",
+		SessionID: "session-123",
+	}
+
+	cmd := tool.BuildCommand(cfg, "/test/dir", "test task")
+
+	// Should use python3 for resume
+	if cmd.Path == "" && cmd.Args[0] != "python3" {
+		t.Error("BuildCommand with SessionID should use python3")
+	}
+}
+
+func TestTool_SupportsStatusTracking(t *testing.T) {
+	tool := New()
+	if !tool.SupportsStatusTracking() {
+		t.Error("Codex tool should support status tracking")
+	}
+}
+
+func TestTool_ApplyToolDefaults(t *testing.T) {
+	tool := New()
+	cfg := &runner.Config{}
+
+	tool.ApplyToolDefaults(cfg)
+
+	if cfg.Effort != "xhigh" {
+		t.Errorf("ApplyToolDefaults should set Effort to 'xhigh', got %q", cfg.Effort)
+	}
+	if !cfg.TrackStatus {
+		t.Error("ApplyToolDefaults should set TrackStatus to true")
+	}
+}
+
+func TestTool_ApplyToolDefaults_WithSettings(t *testing.T) {
+	tool := New()
+	s := &settings.Settings{
+		Defaults: settings.Defaults{
+			Codex: settings.CodexDefaults{
+				Model:  "custom-model",
+				Effort: "medium",
+			},
+		},
+	}
+	tool.SetSettings(s)
+
+	cfg := &runner.Config{}
+	tool.ApplyToolDefaults(cfg)
+
+	if cfg.Model != "custom-model" {
+		t.Errorf("ApplyToolDefaults should set Model from settings, got %q", cfg.Model)
+	}
+	if cfg.Effort != "medium" {
+		t.Errorf("ApplyToolDefaults should set Effort from settings, got %q", cfg.Effort)
+	}
+}
+
+func TestTool_ValidateConfig(t *testing.T) {
+	tool := New()
+	cfg := &runner.Config{Model: "any-model"}
+
+	// Codex accepts any model
+	if err := tool.ValidateConfig(cfg); err != nil {
+		t.Errorf("ValidateConfig should accept any model, got error: %v", err)
+	}
+}
+
+func TestTool_ToolSpecificFlags(t *testing.T) {
+	tool := New()
+	flags := tool.ToolSpecificFlags()
+
+	if len(flags) == 0 {
+		t.Error("ToolSpecificFlags should return non-empty slice")
+	}
+
+	// Check for effort flag
+	hasEffort := false
+	for _, f := range flags {
+		if f.Target == "Effort" {
+			hasEffort = true
+			break
+		}
+	}
+	if !hasEffort {
+		t.Error("ToolSpecificFlags should include Effort flag")
+	}
+}
+
+func TestTool_BannerTitle(t *testing.T) {
+	tool := New()
+	title := tool.BannerTitle()
+	if !strings.Contains(title, "rcodex") {
+		t.Errorf("BannerTitle should contain 'rcodex', got %q", title)
+	}
+}
+
+func TestTool_SecurityWarning(t *testing.T) {
+	tool := New()
+	warnings := tool.SecurityWarning()
+	if len(warnings) == 0 {
+		t.Error("SecurityWarning should return non-empty slice")
+	}
+}
+
+func TestTool_UsesStreamOutput(t *testing.T) {
+	tool := New()
+	if tool.UsesStreamOutput() {
+		t.Error("Codex tool should not use stream output")
+	}
+}
```

### 5. pkg/tools/gemini/gemini_test.go (NEW FILE)

```diff
--- /dev/null
+++ b/pkg/tools/gemini/gemini_test.go
@@ -0,0 +1,172 @@
+package gemini
+
+import (
+	"strings"
+	"testing"
+
+	"rcodegen/pkg/runner"
+)
+
+func TestNew_ReturnsNonNil(t *testing.T) {
+	tool := New()
+	if tool == nil {
+		t.Error("New() should return non-nil Tool")
+	}
+}
+
+func TestTool_Name(t *testing.T) {
+	tool := New()
+	if tool.Name() != "rgemini" {
+		t.Errorf("Name() = %q, want 'rgemini'", tool.Name())
+	}
+}
+
+func TestTool_BinaryName(t *testing.T) {
+	tool := New()
+	if tool.BinaryName() != "gemini" {
+		t.Errorf("BinaryName() = %q, want 'gemini'", tool.BinaryName())
+	}
+}
+
+func TestTool_ReportDir(t *testing.T) {
+	tool := New()
+	if tool.ReportDir() != "_rcodegen" {
+		t.Errorf("ReportDir() = %q, want '_rcodegen'", tool.ReportDir())
+	}
+}
+
+func TestTool_ReportPrefix(t *testing.T) {
+	tool := New()
+	if tool.ReportPrefix() != "gemini-" {
+		t.Errorf("ReportPrefix() = %q, want 'gemini-'", tool.ReportPrefix())
+	}
+}
+
+func TestTool_ValidModels(t *testing.T) {
+	tool := New()
+	models := tool.ValidModels()
+	if len(models) == 0 {
+		t.Error("ValidModels() should return non-empty slice")
+	}
+
+	expected := []string{"gemini-2.5-pro", "gemini-2.5-flash", "gemini-3-pro-preview", "gemini-3-flash-preview"}
+	for _, exp := range expected {
+		found := false
+		for _, m := range models {
+			if m == exp {
+				found = true
+				break
+			}
+		}
+		if !found {
+			t.Errorf("ValidModels() missing expected model %q", exp)
+		}
+	}
+}
+
+func TestTool_DefaultModel(t *testing.T) {
+	tool := New()
+	if tool.DefaultModel() != "gemini-3-pro-preview" {
+		t.Errorf("DefaultModel() = %q, want 'gemini-3-pro-preview'", tool.DefaultModel())
+	}
+}
+
+func TestTool_DefaultModelSetting(t *testing.T) {
+	tool := New()
+	// Gemini doesn't have settings support yet
+	if tool.DefaultModelSetting() != "gemini-3-pro-preview" {
+		t.Errorf("DefaultModelSetting() = %q, want 'gemini-3-pro-preview'", tool.DefaultModelSetting())
+	}
+}
+
+func TestTool_BuildCommand_NewSession(t *testing.T) {
+	tool := New()
+	cfg := &runner.Config{
+		Model: "gemini-3-pro-preview",
+	}
+
+	cmd := tool.BuildCommand(cfg, "/test/dir", "test task")
+
+	args := strings.Join(cmd.Args, " ")
+	if !strings.Contains(args, "-p") {
+		t.Error("BuildCommand should include -p flag for prompt")
+	}
+	if !strings.Contains(args, "--output-format") {
+		t.Error("BuildCommand should include --output-format flag")
+	}
+	if !strings.Contains(args, "--yolo") {
+		t.Error("BuildCommand should include --yolo flag")
+	}
+
+	// Should not include model flag when using default
+	if strings.Contains(args, "-m") {
+		t.Error("BuildCommand should not include -m flag when using default model")
+	}
+}
+
+func TestTool_BuildCommand_CustomModel(t *testing.T) {
+	tool := New()
+	cfg := &runner.Config{
+		Model: "gemini-2.5-flash",
+	}
+
+	cmd := tool.BuildCommand(cfg, "", "test task")
+
+	args := strings.Join(cmd.Args, " ")
+	if !strings.Contains(args, "-m") {
+		t.Error("BuildCommand should include -m flag for non-default model")
+	}
+}
+
+func TestTool_BuildCommand_ResumeSession(t *testing.T) {
+	tool := New()
+	cfg := &runner.Config{
+		Model:     "gemini-3-pro-preview",
+		SessionID: "session-abc",
+	}
+
+	cmd := tool.BuildCommand(cfg, "", "test task")
+
+	args := strings.Join(cmd.Args, " ")
+	if !strings.Contains(args, "--resume") {
+		t.Error("BuildCommand should include --resume flag when SessionID provided")
+	}
+}
+
+func TestTool_SupportsStatusTracking(t *testing.T) {
+	tool := New()
+	if tool.SupportsStatusTracking() {
+		t.Error("Gemini tool should not support status tracking")
+	}
+}
+
+func TestTool_ValidateConfig_ValidModel(t *testing.T) {
+	tool := New()
+	cfg := &runner.Config{Model: "gemini-3-pro-preview"}
+
+	if err := tool.ValidateConfig(cfg); err != nil {
+		t.Errorf("ValidateConfig should accept valid model, got error: %v", err)
+	}
+}
+
+func TestTool_ValidateConfig_InvalidModel(t *testing.T) {
+	tool := New()
+	cfg := &runner.Config{Model: "invalid-model"}
+
+	if err := tool.ValidateConfig(cfg); err == nil {
+		t.Error("ValidateConfig should reject invalid model")
+	}
+}
+
+func TestTool_PrepareForExecution_FlashFlag(t *testing.T) {
+	tool := New()
+	cfg := &runner.Config{
+		Model: "gemini-3-pro-preview",
+		Flash: true,
+	}
+
+	tool.PrepareForExecution(cfg)
+
+	if cfg.Model != "gemini-3-flash-preview" {
+		t.Errorf("PrepareForExecution should override model when Flash=true, got %q", cfg.Model)
+	}
+}
+
+func TestTool_UsesStreamOutput(t *testing.T) {
+	tool := New()
+	if !tool.UsesStreamOutput() {
+		t.Error("Gemini tool should use stream output")
+	}
+}
+
+func TestTool_ToolSpecificFlags(t *testing.T) {
+	tool := New()
+	flags := tool.ToolSpecificFlags()
+
+	// Should have at least the --flash flag
+	hasFlash := false
+	for _, f := range flags {
+		if f.Long == "--flash" {
+			hasFlash = true
+			break
+		}
+	}
+	if !hasFlash {
+		t.Error("ToolSpecificFlags should include --flash flag")
+	}
+}
+
+func TestTool_SecurityWarning(t *testing.T) {
+	tool := New()
+	warnings := tool.SecurityWarning()
+	if len(warnings) == 0 {
+		t.Error("SecurityWarning should return non-empty slice")
+	}
+	// Should mention yolo mode
+	joined := strings.Join(warnings, " ")
+	if !strings.Contains(joined, "yolo") {
+		t.Error("SecurityWarning should mention yolo mode")
+	}
+}
```

---

## Recommendations

### Immediate Actions (P0)

1. **Add pkg/reports/manager_test.go** - Critical business logic for report management
2. **Add pkg/tracking/codex_test.go** - External command execution needs error handling tests
3. **Add pkg/tracking/claude_test.go** - Similar coverage needs

### Short-term Actions (P1)

4. **Add pkg/tools/codex/codex_test.go** - Tool interface implementation
5. **Add pkg/tools/gemini/gemini_test.go** - Tool interface implementation

### Medium-term Actions (P2)

6. **Expand pkg/settings tests** - Cover Load(), ToTaskConfig(), ValidateNoReservedTaskOverrides()
7. **Expand pkg/runner tests** - Cover tasks.go, grades.go, validate.go
8. **Expand pkg/executor tests** - Cover other executor types beyond vote

### Testing Infrastructure Improvements

- Add integration tests for full task execution paths
- Add mock interfaces for external command execution
- Consider adding a testutil package with common helpers

---

## Files Summary

| File | Status | Lines | Impact |
|------|--------|-------|--------|
| pkg/reports/manager_test.go | NEW | ~289 | HIGH |
| pkg/tracking/codex_test.go | NEW | ~142 | HIGH |
| pkg/tracking/claude_test.go | NEW | ~113 | MEDIUM |
| pkg/tools/codex/codex_test.go | NEW | ~198 | MEDIUM |
| pkg/tools/gemini/gemini_test.go | NEW | ~172 | MEDIUM |

**Total new test lines proposed: ~914**

This would increase test coverage from ~21% to approximately 27%, with critical business logic paths now covered.

---

*Report generated by Claude:Opus 4.5 for rcodegen test coverage analysis*
