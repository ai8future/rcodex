# Comprehensive Unit Test Proposal for rcodegen

**Date Created:** 2026-01-21 04:05:00 UTC

---

## Executive Summary

This report analyzes the rcodegen codebase and proposes comprehensive unit tests for untested or under-tested code. The codebase has 12 existing test files with ~30+ test functions, but significant portions of the code lack test coverage.

**Coverage Analysis:**
- **Well-tested:** `flags.go`, `stream.go`, `condition.go` (orchestrator)
- **Partially tested:** `runner.go`, `claude.go`, `settings.go`
- **Untested:** `grades.go`, `migrate.go`, `output.go`, `validate.go`, `codex.go`, `gemini.go`, `tracking/*.go`, `orchestrator.go`, `executor/*.go`, `bundle/loader.go`

---

## Table of Contents

1. [Package: runner - grades.go](#1-package-runner---gradesgo)
2. [Package: runner - migrate.go](#2-package-runner---migratego)
3. [Package: runner - output.go](#3-package-runner---outputgo)
4. [Package: runner - validate.go](#4-package-runner---validatego)
5. [Package: runner - config.go](#5-package-runner---configgo)
6. [Package: tools/codex](#6-package-toolscodex)
7. [Package: tools/gemini](#7-package-toolsgemini)
8. [Package: tools/claude](#8-package-toolsclaude)
9. [Package: tracking](#9-package-tracking)
10. [Package: orchestrator](#10-package-orchestrator)
11. [Package: executor](#11-package-executor)
12. [Package: bundle](#12-package-bundle)

---

## 1. Package: runner - grades.go

**File:** `pkg/runner/grades.go`
**Lines:** 275
**Current Coverage:** None

### Functions to Test

| Function | Lines | Complexity | Priority |
|----------|-------|------------|----------|
| `ExtractGradeFromReport` | 55-75 | Medium | High |
| `ParseReportFilename` | 81-118 | High | High |
| `parseFlexibleDate` | 121-139 | Medium | High |
| `LoadGrades` | 142-165 | Low | Medium |
| `SaveGrades` | 168-190 | Low | Medium |
| `AppendGrade` | 194-222 | Medium | High |
| `escapeGlobPattern` | 225-235 | Low | Medium |
| `FindNewestReport` | 238-274 | Medium | High |

### Proposed Tests

#### Test File: `pkg/runner/grades_test.go`

```diff
--- /dev/null
+++ b/pkg/runner/grades_test.go
@@ -0,0 +1,386 @@
+package runner
+
+import (
+	"os"
+	"path/filepath"
+	"testing"
+	"time"
+)
+
+func TestExtractGradeFromReport_TOTAL_SCORE(t *testing.T) {
+	// Create temp file with TOTAL_SCORE format
+	content := `# Audit Report
+
+## Summary
+The code is well-structured.
+
+TOTAL_SCORE: 85/100
+`
+	tmpFile := createTempReportFile(t, content)
+	defer os.Remove(tmpFile)
+
+	grade, err := ExtractGradeFromReport(tmpFile)
+	if err != nil {
+		t.Fatalf("unexpected error: %v", err)
+	}
+	if grade != 85 {
+		t.Errorf("expected grade 85, got %f", grade)
+	}
+}
+
+func TestExtractGradeFromReport_OverallGrade(t *testing.T) {
+	content := `# Code Review
+
+Overall Grade: 92/100
+`
+	tmpFile := createTempReportFile(t, content)
+	defer os.Remove(tmpFile)
+
+	grade, err := ExtractGradeFromReport(tmpFile)
+	if err != nil {
+		t.Fatalf("unexpected error: %v", err)
+	}
+	if grade != 92 {
+		t.Errorf("expected grade 92, got %f", grade)
+	}
+}
+
+func TestExtractGradeFromReport_GenericGrade(t *testing.T) {
+	content := `# Report
+Grade: 78/100
+`
+	tmpFile := createTempReportFile(t, content)
+	defer os.Remove(tmpFile)
+
+	grade, err := ExtractGradeFromReport(tmpFile)
+	if err != nil {
+		t.Fatalf("unexpected error: %v", err)
+	}
+	if grade != 78 {
+		t.Errorf("expected grade 78, got %f", grade)
+	}
+}
+
+func TestExtractGradeFromReport_ScorePattern(t *testing.T) {
+	content := `# Analysis
+score: 65/100
+`
+	tmpFile := createTempReportFile(t, content)
+	defer os.Remove(tmpFile)
+
+	grade, err := ExtractGradeFromReport(tmpFile)
+	if err != nil {
+		t.Fatalf("unexpected error: %v", err)
+	}
+	if grade != 65 {
+		t.Errorf("expected grade 65, got %f", grade)
+	}
+}
+
+func TestExtractGradeFromReport_NoGrade(t *testing.T) {
+	content := `# Report without grade
+This report has no score.
+`
+	tmpFile := createTempReportFile(t, content)
+	defer os.Remove(tmpFile)
+
+	_, err := ExtractGradeFromReport(tmpFile)
+	if err == nil {
+		t.Error("expected error for report without grade")
+	}
+}
+
+func TestExtractGradeFromReport_ZeroGrade(t *testing.T) {
+	content := `# Failed Report
+TOTAL_SCORE: 0/100
+`
+	tmpFile := createTempReportFile(t, content)
+	defer os.Remove(tmpFile)
+
+	grade, err := ExtractGradeFromReport(tmpFile)
+	if err != nil {
+		t.Fatalf("unexpected error: %v", err)
+	}
+	if grade != 0 {
+		t.Errorf("expected grade 0, got %f", grade)
+	}
+}
+
+func TestExtractGradeFromReport_DecimalGrade(t *testing.T) {
+	content := `TOTAL_SCORE: 87.5/100`
+	tmpFile := createTempReportFile(t, content)
+	defer os.Remove(tmpFile)
+
+	grade, err := ExtractGradeFromReport(tmpFile)
+	if err != nil {
+		t.Fatalf("unexpected error: %v", err)
+	}
+	if grade != 87.5 {
+		t.Errorf("expected grade 87.5, got %f", grade)
+	}
+}
+
+func TestExtractGradeFromReport_FileNotFound(t *testing.T) {
+	_, err := ExtractGradeFromReport("/nonexistent/path/report.md")
+	if err == nil {
+		t.Error("expected error for nonexistent file")
+	}
+}
+
+func TestParseReportFilename_NewFormat(t *testing.T) {
+	tests := []struct {
+		filename     string
+		wantTool     string
+		wantCodebase string
+		wantTask     string
+		wantYear     int
+		wantMonth    time.Month
+		wantDay      int
+	}{
+		{
+			filename:     "myproject-claude-audit-2026-01-20_2204.md",
+			wantTool:     "claude",
+			wantCodebase: "myproject",
+			wantTask:     "audit",
+			wantYear:     2026,
+			wantMonth:    time.January,
+			wantDay:      20,
+		},
+		{
+			filename:     "dispatch-gemini-test-2026-01-15_1430.md",
+			wantTool:     "gemini",
+			wantCodebase: "dispatch",
+			wantTask:     "test",
+			wantYear:     2026,
+			wantMonth:    time.January,
+			wantDay:      15,
+		},
+		{
+			filename:     "rcodegen-codex-fix-2025-12-31_2359.md",
+			wantTool:     "codex",
+			wantCodebase: "rcodegen",
+			wantTask:     "fix",
+			wantYear:     2025,
+			wantMonth:    time.December,
+			wantDay:      31,
+		},
+	}
+
+	for _, tc := range tests {
+		t.Run(tc.filename, func(t *testing.T) {
+			tool, codebase, task, date, err := ParseReportFilename(tc.filename)
+			if err != nil {
+				t.Fatalf("unexpected error: %v", err)
+			}
+			if tool != tc.wantTool {
+				t.Errorf("tool = %q, want %q", tool, tc.wantTool)
+			}
+			if codebase != tc.wantCodebase {
+				t.Errorf("codebase = %q, want %q", codebase, tc.wantCodebase)
+			}
+			if task != tc.wantTask {
+				t.Errorf("task = %q, want %q", task, tc.wantTask)
+			}
+			if date.Year() != tc.wantYear {
+				t.Errorf("year = %d, want %d", date.Year(), tc.wantYear)
+			}
+			if date.Month() != tc.wantMonth {
+				t.Errorf("month = %v, want %v", date.Month(), tc.wantMonth)
+			}
+			if date.Day() != tc.wantDay {
+				t.Errorf("day = %d, want %d", date.Day(), tc.wantDay)
+			}
+		})
+	}
+}
+
+func TestParseReportFilename_OldFormat(t *testing.T) {
+	// Old format: {tool}-{codebase}-{task}-{date}.md
+	tool, codebase, task, _, err := ParseReportFilename("claude-dispatch-audit-2026-01-16_2331.md")
+	if err != nil {
+		t.Fatalf("unexpected error: %v", err)
+	}
+	if tool != "claude" {
+		t.Errorf("tool = %q, want 'claude'", tool)
+	}
+	if codebase != "dispatch" {
+		t.Errorf("codebase = %q, want 'dispatch'", codebase)
+	}
+	if task != "audit" {
+		t.Errorf("task = %q, want 'audit'", task)
+	}
+}
+
+func TestParseReportFilename_InvalidFormat(t *testing.T) {
+	invalidFilenames := []string{
+		"invalid.md",
+		"no-date.md",
+		"one-two.md",
+		"readme.md",
+		"",
+	}
+
+	for _, fn := range invalidFilenames {
+		t.Run(fn, func(t *testing.T) {
+			_, _, _, _, err := ParseReportFilename(fn)
+			if err == nil {
+				t.Errorf("expected error for invalid filename %q", fn)
+			}
+		})
+	}
+}
+
+func TestParseFlexibleDate(t *testing.T) {
+	tests := []struct {
+		dateStr  string
+		wantYear int
+		wantErr  bool
+	}{
+		{"2026-01-20_150405", 2026, false}, // YYYY-MM-DD_HHMMSS
+		{"2026-01-20_1504", 2026, false},   // YYYY-MM-DD_HHMM (standard)
+		{"20260120-150405", 2026, false},   // YYYYMMDD-HHMMSS (compact)
+		{"20260120-1504", 2026, false},     // YYYYMMDD-HHMM
+		{"2026-01-20", 2026, false},        // YYYY-MM-DD
+		{"20260120", 2026, false},          // YYYYMMDD
+		{"invalid", 0, true},
+		{"2026/01/20", 0, true}, // Wrong separator
+	}
+
+	for _, tc := range tests {
+		t.Run(tc.dateStr, func(t *testing.T) {
+			date, err := parseFlexibleDate(tc.dateStr)
+			if tc.wantErr {
+				if err == nil {
+					t.Errorf("expected error for %q", tc.dateStr)
+				}
+				return
+			}
+			if err != nil {
+				t.Fatalf("unexpected error: %v", err)
+			}
+			if date.Year() != tc.wantYear {
+				t.Errorf("year = %d, want %d", date.Year(), tc.wantYear)
+			}
+		})
+	}
+}
+
+func TestLoadGrades_NonexistentFile(t *testing.T) {
+	tmpDir := t.TempDir()
+
+	grades, err := LoadGrades(tmpDir)
+	if err != nil {
+		t.Fatalf("unexpected error: %v", err)
+	}
+	if len(grades.Grades) != 0 {
+		t.Errorf("expected empty grades, got %d entries", len(grades.Grades))
+	}
+}
+
+func TestLoadGrades_ValidFile(t *testing.T) {
+	tmpDir := t.TempDir()
+	gradesContent := `{"grades":[{"date":"2026-01-20T10:00:00Z","tool":"claude","task":"audit","grade":85,"reportFile":"test.md"}]}`
+	gradesPath := filepath.Join(tmpDir, ".grades.json")
+	if err := os.WriteFile(gradesPath, []byte(gradesContent), 0644); err != nil {
+		t.Fatal(err)
+	}
+
+	grades, err := LoadGrades(tmpDir)
+	if err != nil {
+		t.Fatalf("unexpected error: %v", err)
+	}
+	if len(grades.Grades) != 1 {
+		t.Errorf("expected 1 grade, got %d", len(grades.Grades))
+	}
+	if grades.Grades[0].Grade != 85 {
+		t.Errorf("expected grade 85, got %f", grades.Grades[0].Grade)
+	}
+}
+
+func TestLoadGrades_InvalidJSON(t *testing.T) {
+	tmpDir := t.TempDir()
+	gradesPath := filepath.Join(tmpDir, ".grades.json")
+	if err := os.WriteFile(gradesPath, []byte("invalid json"), 0644); err != nil {
+		t.Fatal(err)
+	}
+
+	_, err := LoadGrades(tmpDir)
+	if err == nil {
+		t.Error("expected error for invalid JSON")
+	}
+}
+
+func TestSaveGrades(t *testing.T) {
+	tmpDir := t.TempDir()
+	grades := &GradesFile{
+		Grades: []GradeEntry{
+			{Date: "2026-01-20T10:00:00Z", Tool: "claude", Task: "audit", Grade: 90, ReportFile: "test.md"},
+		},
+	}
+
+	err := SaveGrades(tmpDir, grades)
+	if err != nil {
+		t.Fatalf("unexpected error: %v", err)
+	}
+
+	// Verify file was written
+	gradesPath := filepath.Join(tmpDir, ".grades.json")
+	if _, err := os.Stat(gradesPath); err != nil {
+		t.Errorf("grades file not created: %v", err)
+	}
+
+	// Verify content can be loaded back
+	loaded, err := LoadGrades(tmpDir)
+	if err != nil {
+		t.Fatalf("failed to reload grades: %v", err)
+	}
+	if len(loaded.Grades) != 1 || loaded.Grades[0].Grade != 90 {
+		t.Errorf("loaded grades don't match saved grades")
+	}
+}
+
+func TestAppendGrade(t *testing.T) {
+	tmpDir := t.TempDir()
+
+	// Append first grade
+	err := AppendGrade(tmpDir, "report1.md", "claude", "audit", 85, time.Now())
+	if err != nil {
+		t.Fatalf("unexpected error: %v", err)
+	}
+
+	// Append second grade
+	err = AppendGrade(tmpDir, "report2.md", "gemini", "test", 90, time.Now())
+	if err != nil {
+		t.Fatalf("unexpected error: %v", err)
+	}
+
+	// Verify both grades exist
+	grades, err := LoadGrades(tmpDir)
+	if err != nil {
+		t.Fatalf("failed to load grades: %v", err)
+	}
+	if len(grades.Grades) != 2 {
+		t.Errorf("expected 2 grades, got %d", len(grades.Grades))
+	}
+}
+
+func TestAppendGrade_DuplicatePrevention(t *testing.T) {
+	tmpDir := t.TempDir()
+
+	// Append same report file twice
+	err := AppendGrade(tmpDir, "report1.md", "claude", "audit", 85, time.Now())
+	if err != nil {
+		t.Fatalf("unexpected error: %v", err)
+	}
+
+	err = AppendGrade(tmpDir, "report1.md", "claude", "audit", 90, time.Now())
+	if err != nil {
+		t.Fatalf("unexpected error: %v", err)
+	}
+
+	// Should still only have one entry
+	grades, err := LoadGrades(tmpDir)
+	if err != nil {
+		t.Fatalf("failed to load grades: %v", err)
+	}
+	if len(grades.Grades) != 1 {
+		t.Errorf("expected 1 grade (duplicate prevented), got %d", len(grades.Grades))
+	}
+}
+
+func TestEscapeGlobPattern(t *testing.T) {
+	tests := []struct {
+		input    string
+		expected string
+	}{
+		{"simple", "simple"},
+		{"file*.md", "file\\*.md"},
+		{"test?.txt", "test\\?.txt"},
+		{"[test]", "\\[test\\]"},
+		{"path\\to\\file", "path\\\\to\\\\file"},
+		{"*?[]\\", "\\*\\?\\[\\]\\\\"},
+	}
+
+	for _, tc := range tests {
+		t.Run(tc.input, func(t *testing.T) {
+			result := escapeGlobPattern(tc.input)
+			if result != tc.expected {
+				t.Errorf("escapeGlobPattern(%q) = %q, want %q", tc.input, result, tc.expected)
+			}
+		})
+	}
+}
+
+func TestFindNewestReport(t *testing.T) {
+	tmpDir := t.TempDir()
+
+	// Create test report files with different modification times
+	file1 := filepath.Join(tmpDir, "project-claude-audit-2026-01-01_1000.md")
+	file2 := filepath.Join(tmpDir, "project-claude-audit-2026-01-02_1000.md")
+
+	if err := os.WriteFile(file1, []byte("old report"), 0644); err != nil {
+		t.Fatal(err)
+	}
+	time.Sleep(10 * time.Millisecond) // Ensure different mod times
+	if err := os.WriteFile(file2, []byte("new report"), 0644); err != nil {
+		t.Fatal(err)
+	}
+
+	newest, err := FindNewestReport(tmpDir, "claude", "audit")
+	if err != nil {
+		t.Fatalf("unexpected error: %v", err)
+	}
+
+	if filepath.Base(newest) != "project-claude-audit-2026-01-02_1000.md" {
+		t.Errorf("expected newest report, got %s", filepath.Base(newest))
+	}
+}
+
+func TestFindNewestReport_NoMatches(t *testing.T) {
+	tmpDir := t.TempDir()
+
+	_, err := FindNewestReport(tmpDir, "claude", "audit")
+	if err == nil {
+		t.Error("expected error when no matching reports found")
+	}
+}
+
+// Helper function to create temp report files
+func createTempReportFile(t *testing.T, content string) string {
+	t.Helper()
+	tmpFile, err := os.CreateTemp("", "report-*.md")
+	if err != nil {
+		t.Fatal(err)
+	}
+	if _, err := tmpFile.WriteString(content); err != nil {
+		t.Fatal(err)
+	}
+	tmpFile.Close()
+	return tmpFile.Name()
+}
```

---

## 2. Package: runner - migrate.go

**File:** `pkg/runner/migrate.go`
**Lines:** 154
**Current Coverage:** None

### Proposed Tests

#### Test File: `pkg/runner/migrate_test.go`

```diff
--- /dev/null
+++ b/pkg/runner/migrate_test.go
@@ -0,0 +1,120 @@
+package runner
+
+import (
+	"os"
+	"path/filepath"
+	"testing"
+)
+
+func TestMigrateGrades_NoReportDir(t *testing.T) {
+	tmpDir := t.TempDir()
+
+	err := MigrateGrades(tmpDir)
+	if err == nil {
+		t.Error("expected error when _rcodegen directory doesn't exist")
+	}
+}
+
+func TestMigrateGrades_EmptyReportDir(t *testing.T) {
+	tmpDir := t.TempDir()
+	reportDir := filepath.Join(tmpDir, "_rcodegen")
+	if err := os.MkdirAll(reportDir, 0755); err != nil {
+		t.Fatal(err)
+	}
+
+	// Should complete without error even with empty directory
+	err := MigrateGrades(tmpDir)
+	if err != nil {
+		t.Errorf("unexpected error: %v", err)
+	}
+}
+
+func TestMigrateGrades_WithReports(t *testing.T) {
+	tmpDir := t.TempDir()
+	reportDir := filepath.Join(tmpDir, "_rcodegen")
+	if err := os.MkdirAll(reportDir, 0755); err != nil {
+		t.Fatal(err)
+	}
+
+	// Create valid report with grade
+	reportContent := `# Audit Report
+TOTAL_SCORE: 85/100
+`
+	reportFile := filepath.Join(reportDir, "project-claude-audit-2026-01-20_1000.md")
+	if err := os.WriteFile(reportFile, []byte(reportContent), 0644); err != nil {
+		t.Fatal(err)
+	}
+
+	err := MigrateGrades(tmpDir)
+	if err != nil {
+		t.Errorf("unexpected error: %v", err)
+	}
+
+	// Verify .grades.json was created
+	gradesPath := filepath.Join(reportDir, ".grades.json")
+	if _, err := os.Stat(gradesPath); err != nil {
+		t.Errorf(".grades.json not created: %v", err)
+	}
+
+	// Verify grade was extracted
+	grades, err := LoadGrades(reportDir)
+	if err != nil {
+		t.Fatalf("failed to load grades: %v", err)
+	}
+	if len(grades.Grades) != 1 {
+		t.Errorf("expected 1 grade, got %d", len(grades.Grades))
+	}
+	if grades.Grades[0].Grade != 85 {
+		t.Errorf("expected grade 85, got %f", grades.Grades[0].Grade)
+	}
+}
+
+func TestMigrateGrades_SkipsExisting(t *testing.T) {
+	tmpDir := t.TempDir()
+	reportDir := filepath.Join(tmpDir, "_rcodegen")
+	if err := os.MkdirAll(reportDir, 0755); err != nil {
+		t.Fatal(err)
+	}
+
+	// Create report file
+	reportFile := filepath.Join(reportDir, "project-claude-audit-2026-01-20_1000.md")
+	if err := os.WriteFile(reportFile, []byte("TOTAL_SCORE: 85/100"), 0644); err != nil {
+		t.Fatal(err)
+	}
+
+	// Pre-populate grades
+	grades := &GradesFile{
+		Grades: []GradeEntry{
+			{Date: "2026-01-20T10:00:00Z", Tool: "claude", Task: "audit", Grade: 85, ReportFile: "project-claude-audit-2026-01-20_1000.md"},
+		},
+	}
+	if err := SaveGrades(reportDir, grades); err != nil {
+		t.Fatal(err)
+	}
+
+	// Run migration
+	err := MigrateGrades(tmpDir)
+	if err != nil {
+		t.Errorf("unexpected error: %v", err)
+	}
+
+	// Should still have exactly 1 entry (not duplicated)
+	loaded, err := LoadGrades(reportDir)
+	if err != nil {
+		t.Fatalf("failed to load grades: %v", err)
+	}
+	if len(loaded.Grades) != 1 {
+		t.Errorf("expected 1 grade (no duplicate), got %d", len(loaded.Grades))
+	}
+}
+
+func TestMigrateGrades_InvalidFilename(t *testing.T) {
+	tmpDir := t.TempDir()
+	reportDir := filepath.Join(tmpDir, "_rcodegen")
+	if err := os.MkdirAll(reportDir, 0755); err != nil {
+		t.Fatal(err)
+	}
+
+	// Create file with invalid filename format
+	if err := os.WriteFile(filepath.Join(reportDir, "readme.md"), []byte("TOTAL_SCORE: 85/100"), 0644); err != nil {
+		t.Fatal(err)
+	}
+
+	// Should complete without error (file skipped)
+	err := MigrateGrades(tmpDir)
+	if err != nil {
+		t.Errorf("unexpected error: %v", err)
+	}
+}
```

---

## 3. Package: runner - output.go

**File:** `pkg/runner/output.go`
**Lines:** 238
**Current Coverage:** None

### Proposed Tests

#### Test File: `pkg/runner/output_test.go`

```diff
--- /dev/null
+++ b/pkg/runner/output_test.go
@@ -0,0 +1,98 @@
+package runner
+
+import (
+	"testing"
+	"time"
+)
+
+func TestFormatDuration(t *testing.T) {
+	tests := []struct {
+		duration time.Duration
+		expected string
+	}{
+		{0, "0m 0s"},
+		{30 * time.Second, "0m 30s"},
+		{60 * time.Second, "1m 0s"},
+		{90 * time.Second, "1m 30s"},
+		{5 * time.Minute, "5m 0s"},
+		{5*time.Minute + 30*time.Second, "5m 30s"},
+		{65*time.Minute + 45*time.Second, "65m 45s"},
+	}
+
+	for _, tc := range tests {
+		t.Run(tc.expected, func(t *testing.T) {
+			result := FormatDuration(tc.duration)
+			if result != tc.expected {
+				t.Errorf("FormatDuration(%v) = %q, want %q", tc.duration, result, tc.expected)
+			}
+		})
+	}
+}
+
+func TestNewConfig(t *testing.T) {
+	cfg := NewConfig()
+
+	if cfg == nil {
+		t.Fatal("NewConfig() returned nil")
+	}
+	if cfg.Vars == nil {
+		t.Error("Vars map should be initialized")
+	}
+	if len(cfg.Vars) != 0 {
+		t.Error("Vars map should be empty")
+	}
+}
+
+func TestRunStats_JSON(t *testing.T) {
+	stats := RunStats{
+		Tool:         "rclaude",
+		Task:         "test task",
+		TaskShortcut: "audit",
+		Model:        "sonnet",
+		Codebases:    []string{"/path/to/project"},
+		StartTime:    "2026-01-20T10:00:00Z",
+		EndTime:      "2026-01-20T10:05:00Z",
+		DurationSecs: 300,
+		ExitCode:     0,
+		Success:      true,
+		Options: map[string]bool{
+			"lock":       true,
+			"delete_old": false,
+		},
+	}
+
+	if stats.Tool != "rclaude" {
+		t.Errorf("Tool = %q, want 'rclaude'", stats.Tool)
+	}
+	if !stats.Success {
+		t.Error("Success should be true")
+	}
+	if stats.ExitCode != 0 {
+		t.Errorf("ExitCode = %d, want 0", stats.ExitCode)
+	}
+}
+
+func TestConfig_DefaultValues(t *testing.T) {
+	cfg := NewConfig()
+
+	// Check that boolean fields default to false
+	if cfg.OutputJSON {
+		t.Error("OutputJSON should default to false")
+	}
+	if cfg.UseLock {
+		t.Error("UseLock should default to false")
+	}
+	if cfg.DeleteOld {
+		t.Error("DeleteOld should default to false")
+	}
+	if cfg.RequireReview {
+		t.Error("RequireReview should default to false")
+	}
+	if cfg.DryRun {
+		t.Error("DryRun should default to false")
+	}
+
+	// Check string fields are empty
+	if cfg.Task != "" {
+		t.Errorf("Task should be empty, got %q", cfg.Task)
+	}
+	if cfg.Model != "" {
+		t.Errorf("Model should be empty, got %q", cfg.Model)
+	}
+}
```

---

## 4. Package: runner - validate.go

**File:** `pkg/runner/validate.go`
**Lines:** 25
**Current Coverage:** None

### Proposed Tests

#### Test File: `pkg/runner/validate_test.go`

```diff
--- /dev/null
+++ b/pkg/runner/validate_test.go
@@ -0,0 +1,72 @@
+package runner
+
+import (
+	"strings"
+	"testing"
+)
+
+// mockTool implements the Tool interface for testing
+type mockTool struct {
+	validModels []string
+}
+
+func (m *mockTool) ValidModels() []string {
+	return m.validModels
+}
+
+// Implement other required Tool interface methods as no-ops
+func (m *mockTool) Name() string                                      { return "mock" }
+func (m *mockTool) BinaryName() string                                { return "mock" }
+func (m *mockTool) ReportDir() string                                 { return "_rcodegen" }
+func (m *mockTool) ReportPrefix() string                              { return "mock-" }
+func (m *mockTool) DefaultModel() string                              { return m.validModels[0] }
+func (m *mockTool) DefaultModelSetting() string                       { return m.validModels[0] }
+func (m *mockTool) BuildCommand(cfg *Config, workDir, task string) *exec.Cmd { return nil }
+func (m *mockTool) ShowStatus()                                       {}
+func (m *mockTool) SupportsStatusTracking() bool                      { return false }
+func (m *mockTool) CaptureStatusBefore() interface{}                  { return nil }
+func (m *mockTool) CaptureStatusAfter() interface{}                   { return nil }
+func (m *mockTool) PrintStatusSummary(before, after interface{})      {}
+func (m *mockTool) ToolSpecificFlags() []FlagDef                      { return nil }
+func (m *mockTool) ApplyToolDefaults(cfg *Config)                     {}
+func (m *mockTool) PrepareForExecution(cfg *Config)                   {}
+func (m *mockTool) ValidateConfig(cfg *Config) error                  { return nil }
+func (m *mockTool) BannerTitle() string                               { return "Mock" }
+func (m *mockTool) BannerSubtitle() string                            { return "Mock Tool" }
+func (m *mockTool) PrintToolSpecificBannerFields(cfg *Config)         {}
+func (m *mockTool) PrintToolSpecificSummaryFields(cfg *Config)        {}
+func (m *mockTool) SecurityWarning() []string                         { return nil }
+func (m *mockTool) ToolSpecificHelpSections() []HelpSection           { return nil }
+func (m *mockTool) StatsJSONFields(cfg *Config) map[string]interface{} { return nil }
+func (m *mockTool) UsesStreamOutput() bool                            { return false }
+func (m *mockTool) RunLogFields(cfg *Config) []string                 { return nil }
+
+func TestValidateModel_ValidModel(t *testing.T) {
+	tool := &mockTool{validModels: []string{"sonnet", "opus", "haiku"}}
+
+	err := ValidateModel(tool, "sonnet")
+	if err != nil {
+		t.Errorf("unexpected error for valid model: %v", err)
+	}
+}
+
+func TestValidateModel_InvalidModel(t *testing.T) {
+	tool := &mockTool{validModels: []string{"sonnet", "opus", "haiku"}}
+
+	err := ValidateModel(tool, "gpt-4")
+	if err == nil {
+		t.Error("expected error for invalid model")
+	}
+	if !strings.Contains(err.Error(), "invalid model") {
+		t.Errorf("error should mention 'invalid model', got: %v", err)
+	}
+}
+
+func TestIsValidModel(t *testing.T) {
+	tool := &mockTool{validModels: []string{"sonnet", "opus", "haiku"}}
+
+	if !IsValidModel(tool, "opus") {
+		t.Error("IsValidModel should return true for valid model")
+	}
+	if IsValidModel(tool, "invalid") {
+		t.Error("IsValidModel should return false for invalid model")
+	}
+}
```

---

## 5. Package: runner - config.go

**File:** `pkg/runner/config.go`
**Lines:** 66
**Current Coverage:** Partial (via other tests)

### Proposed Tests

Covered in output_test.go above with `TestNewConfig` and `TestConfig_DefaultValues`.

---

## 6. Package: tools/codex

**File:** `pkg/tools/codex/codex.go`
**Lines:** 337
**Current Coverage:** None

### Proposed Tests

#### Test File: `pkg/tools/codex/codex_test.go`

```diff
--- /dev/null
+++ b/pkg/tools/codex/codex_test.go
@@ -0,0 +1,147 @@
+package codex
+
+import (
+	"testing"
+
+	"rcodegen/pkg/runner"
+)
+
+func TestNew_ReturnsNonNil(t *testing.T) {
+	tool := New()
+	if tool == nil {
+		t.Fatal("New() returned nil")
+	}
+}
+
+func TestName(t *testing.T) {
+	tool := New()
+	if tool.Name() != "rcodex" {
+		t.Errorf("Name() = %q, want 'rcodex'", tool.Name())
+	}
+}
+
+func TestBinaryName(t *testing.T) {
+	tool := New()
+	if tool.BinaryName() != "codex" {
+		t.Errorf("BinaryName() = %q, want 'codex'", tool.BinaryName())
+	}
+}
+
+func TestReportDir(t *testing.T) {
+	tool := New()
+	if tool.ReportDir() != "_rcodegen" {
+		t.Errorf("ReportDir() = %q, want '_rcodegen'", tool.ReportDir())
+	}
+}
+
+func TestReportPrefix(t *testing.T) {
+	tool := New()
+	if tool.ReportPrefix() != "codex-" {
+		t.Errorf("ReportPrefix() = %q, want 'codex-'", tool.ReportPrefix())
+	}
+}
+
+func TestValidModels(t *testing.T) {
+	tool := New()
+	models := tool.ValidModels()
+
+	if len(models) == 0 {
+		t.Fatal("ValidModels() returned empty list")
+	}
+
+	// Check for expected models
+	expected := map[string]bool{
+		"gpt-5.2-codex": true,
+		"gpt-4.1-codex": true,
+		"gpt-4o-codex":  true,
+	}
+
+	for _, m := range models {
+		if !expected[m] {
+			t.Errorf("unexpected model %q in ValidModels()", m)
+		}
+	}
+}
+
+func TestDefaultModel(t *testing.T) {
+	tool := New()
+	if tool.DefaultModel() != "gpt-5.2-codex" {
+		t.Errorf("DefaultModel() = %q, want 'gpt-5.2-codex'", tool.DefaultModel())
+	}
+}
+
+func TestSupportsStatusTracking(t *testing.T) {
+	tool := New()
+	if !tool.SupportsStatusTracking() {
+		t.Error("SupportsStatusTracking() should return true")
+	}
+}
+
+func TestUsesStreamOutput(t *testing.T) {
+	tool := New()
+	if tool.UsesStreamOutput() {
+		t.Error("UsesStreamOutput() should return false for Codex")
+	}
+}
+
+func TestToolSpecificFlags(t *testing.T) {
+	tool := New()
+	flags := tool.ToolSpecificFlags()
+
+	if len(flags) == 0 {
+		t.Fatal("ToolSpecificFlags() returned empty list")
+	}
+
+	// Check for effort flag
+	foundEffort := false
+	for _, f := range flags {
+		if f.Target == "Effort" {
+			foundEffort = true
+			if f.Default != "xhigh" {
+				t.Errorf("Effort flag default = %q, want 'xhigh'", f.Default)
+			}
+		}
+	}
+	if !foundEffort {
+		t.Error("expected Effort flag in ToolSpecificFlags")
+	}
+}
+
+func TestApplyToolDefaults(t *testing.T) {
+	tool := New()
+	cfg := runner.NewConfig()
+
+	tool.ApplyToolDefaults(cfg)
+
+	if cfg.Effort != "xhigh" {
+		t.Errorf("Effort = %q, want 'xhigh'", cfg.Effort)
+	}
+	if !cfg.TrackStatus {
+		t.Error("TrackStatus should be true by default")
+	}
+}
+
+func TestBuildCommand_WithoutSessionID(t *testing.T) {
+	tool := New()
+	cfg := &runner.Config{
+		Model:  "gpt-5.2-codex",
+		Effort: "high",
+	}
+
+	cmd := tool.BuildCommand(cfg, "/test/dir", "test task")
+
+	if cmd == nil {
+		t.Fatal("BuildCommand returned nil")
+	}
+	if cmd.Path == "" {
+		t.Error("command path should not be empty")
+	}
+}
+
+func TestStatsJSONFields(t *testing.T) {
+	tool := New()
+	cfg := &runner.Config{Effort: "high", TrackStatus: true}
+
+	fields := tool.StatsJSONFields(cfg)
+
+	if fields["effort"] != "high" {
+		t.Errorf("effort = %v, want 'high'", fields["effort"])
+	}
+	if fields["track_status"] != true {
+		t.Error("track_status should be true")
+	}
+}
```

---

## 7. Package: tools/gemini

**File:** `pkg/tools/gemini/gemini.go`
**Lines:** 217
**Current Coverage:** None

### Proposed Tests

#### Test File: `pkg/tools/gemini/gemini_test.go`

```diff
--- /dev/null
+++ b/pkg/tools/gemini/gemini_test.go
@@ -0,0 +1,137 @@
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
+		t.Fatal("New() returned nil")
+	}
+}
+
+func TestName(t *testing.T) {
+	tool := New()
+	if tool.Name() != "rgemini" {
+		t.Errorf("Name() = %q, want 'rgemini'", tool.Name())
+	}
+}
+
+func TestBinaryName(t *testing.T) {
+	tool := New()
+	if tool.BinaryName() != "gemini" {
+		t.Errorf("BinaryName() = %q, want 'gemini'", tool.BinaryName())
+	}
+}
+
+func TestReportDir(t *testing.T) {
+	tool := New()
+	if tool.ReportDir() != "_rcodegen" {
+		t.Errorf("ReportDir() = %q, want '_rcodegen'", tool.ReportDir())
+	}
+}
+
+func TestReportPrefix(t *testing.T) {
+	tool := New()
+	if tool.ReportPrefix() != "gemini-" {
+		t.Errorf("ReportPrefix() = %q, want 'gemini-'", tool.ReportPrefix())
+	}
+}
+
+func TestValidModels(t *testing.T) {
+	tool := New()
+	models := tool.ValidModels()
+
+	if len(models) == 0 {
+		t.Fatal("ValidModels() returned empty list")
+	}
+
+	// Check for expected models
+	foundProPreview := false
+	foundFlashPreview := false
+	for _, m := range models {
+		if strings.Contains(m, "pro-preview") {
+			foundProPreview = true
+		}
+		if strings.Contains(m, "flash-preview") {
+			foundFlashPreview = true
+		}
+	}
+
+	if !foundProPreview {
+		t.Error("expected pro-preview model in ValidModels()")
+	}
+	if !foundFlashPreview {
+		t.Error("expected flash-preview model in ValidModels()")
+	}
+}
+
+func TestDefaultModel(t *testing.T) {
+	tool := New()
+	if tool.DefaultModel() != "gemini-3-pro-preview" {
+		t.Errorf("DefaultModel() = %q, want 'gemini-3-pro-preview'", tool.DefaultModel())
+	}
+}
+
+func TestSupportsStatusTracking(t *testing.T) {
+	tool := New()
+	if tool.SupportsStatusTracking() {
+		t.Error("SupportsStatusTracking() should return false for Gemini")
+	}
+}
+
+func TestUsesStreamOutput(t *testing.T) {
+	tool := New()
+	if !tool.UsesStreamOutput() {
+		t.Error("UsesStreamOutput() should return true for Gemini")
+	}
+}
+
+func TestValidateConfig_ValidModel(t *testing.T) {
+	tool := New()
+	cfg := &runner.Config{Model: "gemini-3-pro-preview"}
+
+	err := tool.ValidateConfig(cfg)
+	if err != nil {
+		t.Errorf("unexpected error for valid model: %v", err)
+	}
+}
+
+func TestValidateConfig_InvalidModel(t *testing.T) {
+	tool := New()
+	cfg := &runner.Config{Model: "invalid-model"}
+
+	err := tool.ValidateConfig(cfg)
+	if err == nil {
+		t.Error("expected error for invalid model")
+	}
+}
+
+func TestPrepareForExecution_FlashFlag(t *testing.T) {
+	tool := New()
+	cfg := &runner.Config{
+		Flash: true,
+		Model: "gemini-3-pro-preview",
+	}
+
+	tool.PrepareForExecution(cfg)
+
+	if cfg.Model != "gemini-3-flash-preview" {
+		t.Errorf("Model = %q, want 'gemini-3-flash-preview' after flash flag", cfg.Model)
+	}
+}
+
+func TestBuildCommand(t *testing.T) {
+	tool := New()
+	cfg := &runner.Config{Model: "gemini-3-pro-preview"}
+
+	cmd := tool.BuildCommand(cfg, "/test/dir", "test task")
+
+	if cmd == nil {
+		t.Fatal("BuildCommand returned nil")
+	}
+	if cmd.Dir != "/test/dir" {
+		t.Errorf("cmd.Dir = %q, want '/test/dir'", cmd.Dir)
+	}
+}
```

---

## 8. Package: tools/claude

**File:** `pkg/tools/claude/claude.go`
**Lines:** 410
**Current Coverage:** Partial (2 tests)

### Proposed Additional Tests

#### Test File: `pkg/tools/claude/claude_test.go` (append to existing)

```diff
--- a/pkg/tools/claude/claude_test.go
+++ b/pkg/tools/claude/claude_test.go
@@ -29,3 +29,112 @@ func TestNew_ReturnsNonNil(t *testing.T) {
 		t.Error("New() should return non-nil tool")
 	}
 }
+
+func TestName(t *testing.T) {
+	tool := New()
+	if tool.Name() != "rclaude" {
+		t.Errorf("Name() = %q, want 'rclaude'", tool.Name())
+	}
+}
+
+func TestBinaryName(t *testing.T) {
+	tool := New()
+	if tool.BinaryName() != "claude" {
+		t.Errorf("BinaryName() = %q, want 'claude'", tool.BinaryName())
+	}
+}
+
+func TestReportDir(t *testing.T) {
+	tool := New()
+	if tool.ReportDir() != "_rcodegen" {
+		t.Errorf("ReportDir() = %q, want '_rcodegen'", tool.ReportDir())
+	}
+}
+
+func TestReportPrefix(t *testing.T) {
+	tool := New()
+	if tool.ReportPrefix() != "claude-" {
+		t.Errorf("ReportPrefix() = %q, want 'claude-'", tool.ReportPrefix())
+	}
+}
+
+func TestValidModels(t *testing.T) {
+	tool := New()
+	models := tool.ValidModels()
+
+	expected := map[string]bool{"sonnet": true, "opus": true, "haiku": true}
+	for _, m := range models {
+		if !expected[m] {
+			t.Errorf("unexpected model %q in ValidModels()", m)
+		}
+	}
+	if len(models) != 3 {
+		t.Errorf("expected 3 models, got %d", len(models))
+	}
+}
+
+func TestDefaultModel(t *testing.T) {
+	tool := New()
+	if tool.DefaultModel() != "sonnet" {
+		t.Errorf("DefaultModel() = %q, want 'sonnet'", tool.DefaultModel())
+	}
+}
+
+func TestSupportsStatusTracking(t *testing.T) {
+	tool := New()
+	if !tool.SupportsStatusTracking() {
+		t.Error("SupportsStatusTracking() should return true")
+	}
+}
+
+func TestUsesStreamOutput(t *testing.T) {
+	tool := New()
+	if !tool.UsesStreamOutput() {
+		t.Error("UsesStreamOutput() should return true for Claude")
+	}
+}
+
+func TestApplyToolDefaults(t *testing.T) {
+	tool := New()
+	cfg := runner.NewConfig()
+
+	tool.ApplyToolDefaults(cfg)
+
+	if cfg.MaxBudget != "10.00" {
+		t.Errorf("MaxBudget = %q, want '10.00'", cfg.MaxBudget)
+	}
+}
+
+func TestValidateConfig_ValidModel(t *testing.T) {
+	tool := New()
+	cfg := &runner.Config{Model: "sonnet", MaxBudget: "10.00"}
+
+	err := tool.ValidateConfig(cfg)
+	if err != nil {
+		t.Errorf("unexpected error: %v", err)
+	}
+}
+
+func TestValidateConfig_InvalidModel(t *testing.T) {
+	tool := New()
+	cfg := &runner.Config{Model: "gpt-4", MaxBudget: "10.00"}
+
+	err := tool.ValidateConfig(cfg)
+	if err == nil {
+		t.Error("expected error for invalid model")
+	}
+}
+
+func TestValidateConfig_InvalidBudget(t *testing.T) {
+	tests := []struct {
+		budget string
+	}{
+		{"invalid"},
+		{"-10"},
+		{"0"},
+		{"1001"},
+	}
+
+	tool := New()
+	for _, tc := range tests {
+		cfg := &runner.Config{Model: "sonnet", MaxBudget: tc.budget}
+		err := tool.ValidateConfig(cfg)
+		if err == nil {
+			t.Errorf("expected error for budget %q", tc.budget)
+		}
+	}
+}
+
+func TestBuildCommand(t *testing.T) {
+	tool := New()
+	cfg := &runner.Config{Model: "sonnet", MaxBudget: "10.00"}
+
+	cmd := tool.BuildCommand(cfg, "/test/dir", "test task")
+
+	if cmd == nil {
+		t.Fatal("BuildCommand returned nil")
+	}
+	if cmd.Dir != "/test/dir" {
+		t.Errorf("cmd.Dir = %q, want '/test/dir'", cmd.Dir)
+	}
+}
```

---

## 9. Package: tracking

**File:** `pkg/tracking/codex.go` and `pkg/tracking/claude.go`
**Lines:** 195 + 156 = 351
**Current Coverage:** None

### Proposed Tests

#### Test File: `pkg/tracking/tracking_test.go`

```diff
--- /dev/null
+++ b/pkg/tracking/tracking_test.go
@@ -0,0 +1,91 @@
+package tracking
+
+import (
+	"testing"
+)
+
+func TestFormatCredit_Nil(t *testing.T) {
+	result := FormatCredit(nil)
+	if result != "N/A" {
+		t.Errorf("FormatCredit(nil) = %q, want 'N/A'", result)
+	}
+}
+
+func TestFormatCredit_Values(t *testing.T) {
+	tests := []struct {
+		value    int
+		expected string
+	}{
+		{0, "0"},
+		{50, "50"},
+		{100, "100"},
+		{-5, "-5"},
+	}
+
+	for _, tc := range tests {
+		val := tc.value
+		result := FormatCredit(&val)
+		if result != tc.expected {
+			t.Errorf("FormatCredit(%d) = %q, want %q", tc.value, result, tc.expected)
+		}
+	}
+}
+
+func TestFindPython(t *testing.T) {
+	// FindPython should return a non-empty string
+	python := FindPython()
+	if python == "" {
+		t.Error("FindPython() returned empty string")
+	}
+}
+
+func TestGetScriptDir(t *testing.T) {
+	// GetScriptDir should work (may return empty in some test environments)
+	dir := GetScriptDir()
+	// Just verify it doesn't panic
+	_ = dir
+}
+
+func TestCreditStatus_Structure(t *testing.T) {
+	// Test struct initialization
+	status := &CreditStatus{
+		Error: "test error",
+	}
+
+	if status.Error != "test error" {
+		t.Error("Error field not set correctly")
+	}
+	if status.FiveHourLeft != nil {
+		t.Error("FiveHourLeft should be nil by default")
+	}
+}
+
+func TestClaudeStatus_IsITerm2Error(t *testing.T) {
+	tests := []struct {
+		error    string
+		expected bool
+	}{
+		{"not_iterm2", true},
+		{"no_iterm2_package", true},
+		{"other_error", false},
+		{"", false},
+	}
+
+	for _, tc := range tests {
+		status := &ClaudeStatus{Error: tc.error}
+		if status.IsITerm2Error() != tc.expected {
+			t.Errorf("IsITerm2Error() for %q = %v, want %v", tc.error, status.IsITerm2Error(), tc.expected)
+		}
+	}
+}
+
+func TestClaudeStatus_Structure(t *testing.T) {
+	status := &ClaudeStatus{
+		Error:   "test error",
+		Message: "detailed message",
+	}
+
+	if status.Error != "test error" {
+		t.Error("Error field not set correctly")
+	}
+	if status.Message != "detailed message" {
+		t.Error("Message field not set correctly")
+	}
+}
```

---

## 10. Package: orchestrator

**File:** `pkg/orchestrator/orchestrator.go`
**Lines:** 1304
**Current Coverage:** Partial (context and condition tested)

### Proposed Additional Tests

#### Test File: `pkg/orchestrator/orchestrator_test.go`

```diff
--- /dev/null
+++ b/pkg/orchestrator/orchestrator_test.go
@@ -0,0 +1,143 @@
+package orchestrator
+
+import (
+	"os"
+	"path/filepath"
+	"testing"
+)
+
+func TestCapitalize(t *testing.T) {
+	tests := []struct {
+		input    string
+		expected string
+	}{
+		{"", ""},
+		{"a", "A"},
+		{"hello", "Hello"},
+		{"HELLO", "HELLO"},
+		{"hELLO", "HELLO"},
+	}
+
+	for _, tc := range tests {
+		result := capitalize(tc.input)
+		if result != tc.expected {
+			t.Errorf("capitalize(%q) = %q, want %q", tc.input, result, tc.expected)
+		}
+	}
+}
+
+func TestTruncate(t *testing.T) {
+	tests := []struct {
+		input    string
+		maxLen   int
+		expected string
+	}{
+		{"short", 10, "short"},
+		{"exactly10!", 10, "exactly10!"},
+		{"this is longer than ten", 10, "this is..."},
+		{"", 5, ""},
+		{"ab", 5, "ab"},
+	}
+
+	for _, tc := range tests {
+		result := truncate(tc.input, tc.maxLen)
+		if result != tc.expected {
+			t.Errorf("truncate(%q, %d) = %q, want %q", tc.input, tc.maxLen, result, tc.expected)
+		}
+	}
+}
+
+func TestMax(t *testing.T) {
+	tests := []struct {
+		nums     []int
+		expected int
+	}{
+		{[]int{1}, 1},
+		{[]int{1, 2, 3}, 3},
+		{[]int{3, 2, 1}, 3},
+		{[]int{5, 5, 5}, 5},
+		{[]int{-1, -2, -3}, -1},
+		{[]int{-5, 0, 5}, 5},
+	}
+
+	for _, tc := range tests {
+		result := max(tc.nums...)
+		if result != tc.expected {
+			t.Errorf("max(%v) = %d, want %d", tc.nums, result, tc.expected)
+		}
+	}
+}
+
+func TestCategorizeFile(t *testing.T) {
+	tests := []struct {
+		path     string
+		expected string
+	}{
+		{"src/main.go", "source"},
+		{"lib/utils.py", "source"},
+		{"main.go", "source"},
+		{"script.py", "source"},
+		{"app.js", "source"},
+		{"index.ts", "source"},
+		{"samples/test.txt", "sample"},
+		{"test_file.go", "sample"},
+		{"output.pdf", "output"},
+		{"final-report.md", "report"},
+		{"README.md", "docs"},
+		{"docs/guide.md", "docs"},
+		{"config.json", "config"},
+		{"random.xyz", "other"},
+	}
+
+	for _, tc := range tests {
+		result := categorizeFile(tc.path)
+		if result != tc.expected {
+			t.Errorf("categorizeFile(%q) = %q, want %q", tc.path, result, tc.expected)
+		}
+	}
+}
+
+func TestExtractTitle(t *testing.T) {
+	tmpDir := t.TempDir()
+
+	// File with title
+	withTitle := filepath.Join(tmpDir, "with_title.md")
+	os.WriteFile(withTitle, []byte("# My Title\n\nContent here"), 0644)
+
+	title := extractTitle(withTitle)
+	if title != "My Title" {
+		t.Errorf("extractTitle() = %q, want 'My Title'", title)
+	}
+
+	// File without title
+	withoutTitle := filepath.Join(tmpDir, "no_title.md")
+	os.WriteFile(withoutTitle, []byte("No heading here"), 0644)
+
+	title = extractTitle(withoutTitle)
+	if title != "no_title.md" {
+		t.Errorf("extractTitle() = %q, want filename", title)
+	}
+}
+
+func TestCountWords(t *testing.T) {
+	tmpDir := t.TempDir()
+
+	// Create test file
+	testFile := filepath.Join(tmpDir, "test.md")
+	os.WriteFile(testFile, []byte("one two three four five"), 0644)
+
+	count := countWords(testFile)
+	if count != 5 {
+		t.Errorf("countWords() = %d, want 5", count)
+	}
+
+	// Nonexistent file
+	count = countWords("/nonexistent/file.md")
+	if count != 0 {
+		t.Errorf("countWords(nonexistent) = %d, want 0", count)
+	}
+}
+
+func TestGetVersion(t *testing.T) {
+	version := getVersion()
+	// Should return either a version string or "unknown"
+	if version == "" {
+		t.Error("getVersion() returned empty string")
+	}
+}
```

#### Test File: `pkg/orchestrator/context_test.go` (append to existing)

```diff
--- a/pkg/orchestrator/context_test.go
+++ b/pkg/orchestrator/context_test.go
@@ -0,0 +1,80 @@
+// Add these tests to the existing context_test.go file
+
+func TestContext_GetToolSession(t *testing.T) {
+	ctx := NewContext(nil)
+	ctx.SetToolSession("claude", "session-123")
+
+	session := ctx.GetToolSession("claude")
+	if session != "session-123" {
+		t.Errorf("GetToolSession('claude') = %q, want 'session-123'", session)
+	}
+
+	// Non-existent tool
+	session = ctx.GetToolSession("nonexistent")
+	if session != "" {
+		t.Errorf("GetToolSession('nonexistent') = %q, want ''", session)
+	}
+}
+
+func TestContext_GetResult(t *testing.T) {
+	ctx := NewContext(nil)
+	ctx.SetResult("step1", &envelope.Envelope{Status: envelope.StatusSuccess})
+
+	env, ok := ctx.GetResult("step1")
+	if !ok {
+		t.Error("GetResult('step1') should return true")
+	}
+	if env.Status != envelope.StatusSuccess {
+		t.Errorf("status = %v, want success", env.Status)
+	}
+
+	// Non-existent step
+	_, ok = ctx.GetResult("nonexistent")
+	if ok {
+		t.Error("GetResult('nonexistent') should return false")
+	}
+}
+
+func TestExtractStreamingResult_ResultObject(t *testing.T) {
+	content := `{"type":"assistant","message":"processing"}
+{"type":"result","result":"final answer"}`
+
+	result := extractStreamingResult(content)
+	if result != "final answer" {
+		t.Errorf("extractStreamingResult() = %q, want 'final answer'", result)
+	}
+}
+
+func TestExtractStreamingResult_NoResultObject(t *testing.T) {
+	content := "plain text output"
+
+	result := extractStreamingResult(content)
+	if result != content {
+		t.Errorf("extractStreamingResult() = %q, want original content", result)
+	}
+}
+
+func TestContext_Resolve_NestedResult(t *testing.T) {
+	ctx := NewContext(nil)
+	ctx.SetResult("analyze", &envelope.Envelope{
+		Status: envelope.StatusSuccess,
+		Result: map[string]interface{}{
+			"score": 85,
+			"grade": "B+",
+		},
+	})
+
+	// Test resolving result field
+	resolved := ctx.Resolve("${steps.analyze.result.score}")
+	if resolved != "85" {
+		t.Errorf("Resolve result.score = %q, want '85'", resolved)
+	}
+
+	resolved = ctx.Resolve("${steps.analyze.result.grade}")
+	if resolved != "B+" {
+		t.Errorf("Resolve result.grade = %q, want 'B+'", resolved)
+	}
+}
```

---

## 11. Package: executor

**File:** `pkg/executor/dispatcher.go`, `pkg/executor/parallel.go`, `pkg/executor/merge.go`
**Lines:** 51 + 76 + 59 = 186
**Current Coverage:** Partial (vote_test.go exists)

### Proposed Tests

#### Test File: `pkg/executor/dispatcher_test.go`

```diff
--- /dev/null
+++ b/pkg/executor/dispatcher_test.go
@@ -0,0 +1,45 @@
+package executor
+
+import (
+	"testing"
+
+	"rcodegen/pkg/bundle"
+	"rcodegen/pkg/runner"
+)
+
+func TestNewDispatcher(t *testing.T) {
+	tools := make(map[string]runner.Tool)
+	d := NewDispatcher(tools)
+
+	if d == nil {
+		t.Fatal("NewDispatcher returned nil")
+	}
+	if d.tool == nil {
+		t.Error("ToolExecutor should be initialized")
+	}
+	if d.parallel == nil {
+		t.Error("ParallelExecutor should be initialized")
+	}
+	if d.merge == nil {
+		t.Error("MergeExecutor should be initialized")
+	}
+	if d.vote == nil {
+		t.Error("VoteExecutor should be initialized")
+	}
+}
+
+func TestDispatcher_DetermineStepType(t *testing.T) {
+	tools := make(map[string]runner.Tool)
+	d := NewDispatcher(tools)
+
+	// Test unknown step type
+	step := &bundle.Step{Name: "unknown"}
+
+	env, _ := d.Execute(step, nil, nil)
+
+	if env.Status != "failure" {
+		t.Errorf("expected failure status for unknown step type, got %v", env.Status)
+	}
+	if env.ErrorCode != "UNKNOWN_STEP" {
+		t.Errorf("expected UNKNOWN_STEP error code, got %q", env.ErrorCode)
+	}
+}
```

---

## 12. Package: bundle

**File:** `pkg/bundle/loader.go`
**Lines:** 134
**Current Coverage:** Partial (loader_test.go exists)

### Proposed Additional Tests

#### Test File: `pkg/bundle/loader_test.go` (append to existing)

```diff
--- a/pkg/bundle/loader_test.go
+++ b/pkg/bundle/loader_test.go
@@ -0,0 +1,65 @@
+// Add these tests to the existing loader_test.go
+
+func TestValidateBundleName_Valid(t *testing.T) {
+	validNames := []string{
+		"simple",
+		"test-bundle",
+		"bundle_name",
+		"Test123",
+		"a",
+		"bundle-with-many-parts",
+	}
+
+	for _, name := range validNames {
+		err := validateBundleName(name)
+		if err != nil {
+			t.Errorf("validateBundleName(%q) returned error: %v", name, err)
+		}
+	}
+}
+
+func TestValidateBundleName_Invalid(t *testing.T) {
+	invalidNames := []string{
+		"",                              // empty
+		"-starts-with-dash",            // starts with dash
+		"_starts_with_underscore",      // starts with underscore
+		"../path/traversal",            // path traversal
+		"has spaces",                   // spaces
+		"has.dots",                     // dots
+		strings.Repeat("x", 101),       // too long
+		"has/slash",                    // slash
+	}
+
+	for _, name := range invalidNames {
+		err := validateBundleName(name)
+		if err == nil {
+			t.Errorf("validateBundleName(%q) should return error", name)
+		}
+	}
+}
+
+func TestLoad_NonexistentBundle(t *testing.T) {
+	_, err := Load("definitely-nonexistent-bundle-name")
+	if err == nil {
+		t.Error("Load() should return error for nonexistent bundle")
+	}
+}
+
+func TestLoad_PathTraversal(t *testing.T) {
+	_, err := Load("../../../etc/passwd")
+	if err == nil {
+		t.Error("Load() should reject path traversal attempts")
+	}
+}
+
+func TestList(t *testing.T) {
+	names, err := List()
+	if err != nil {
+		t.Fatalf("List() returned error: %v", err)
+	}
+
+	// Should return at least empty slice (not nil)
+	if names == nil {
+		t.Error("List() returned nil instead of empty slice")
+	}
+}
```

---

## Summary

### Test Files to Create

| File | Priority | Tests | Est. Lines |
|------|----------|-------|------------|
| `pkg/runner/grades_test.go` | High | 17 | 386 |
| `pkg/runner/migrate_test.go` | Medium | 5 | 120 |
| `pkg/runner/output_test.go` | Medium | 5 | 98 |
| `pkg/runner/validate_test.go` | Medium | 3 | 72 |
| `pkg/tools/codex/codex_test.go` | High | 15 | 147 |
| `pkg/tools/gemini/gemini_test.go` | High | 13 | 137 |
| `pkg/tracking/tracking_test.go` | Medium | 6 | 91 |
| `pkg/orchestrator/orchestrator_test.go` | Medium | 7 | 143 |
| `pkg/executor/dispatcher_test.go` | Low | 2 | 45 |

### Test Files to Extend

| File | Priority | Additional Tests |
|------|----------|------------------|
| `pkg/tools/claude/claude_test.go` | High | +14 tests |
| `pkg/orchestrator/context_test.go` | Medium | +5 tests |
| `pkg/bundle/loader_test.go` | Medium | +5 tests |

### Total Estimated New Test Code

- **New test files:** ~1,239 lines
- **Extended test files:** ~200 lines
- **Total new tests:** ~77 tests
- **Estimated coverage increase:** 40-50%

---

## Running Tests

After implementing these tests, run the full test suite:

```bash
cd /Users/cliff/Desktop/_code/rcodegen
go test ./pkg/... -v
```

To generate coverage report:

```bash
go test ./pkg/... -coverprofile=coverage.out
go tool cover -html=coverage.out -o coverage.html
```

---

## Notes

1. **Mock considerations:** Some tests require mocking external dependencies (filesystem, executables). The tests above use temporary directories and minimal mocking to keep tests simple and reliable.

2. **Thread safety:** The grades package uses mutexes; tests should verify thread-safe behavior under concurrent access.

3. **Integration tests:** This report focuses on unit tests. Integration tests for the full orchestrator pipeline would require additional setup and are not included here.

4. **Error cases:** Each function's error paths should be tested, including file permission errors, network errors, and invalid inputs.

5. **Test data:** Some tests create temporary files and directories. All cleanup is handled via `t.TempDir()` or explicit `defer os.Remove()` calls.
