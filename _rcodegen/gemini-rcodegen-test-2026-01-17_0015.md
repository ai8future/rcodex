Date Created: 2026-01-17 00:15:00
TOTAL_SCORE: 65/100

# Unit Test Proposal and Coverage Analysis

## 1. Codebase Analysis

The `rcodegen` project has a solid foundation of unit tests in core areas like `pkg/bundle`, `pkg/orchestrator`, and `pkg/lock`. However, several utility packages and newer modules lack test coverage entirely.

### Untested Areas Identified:
- **`pkg/colors`**: No tests. While simple, ensuring constants remain stable is good practice.
- **`pkg/reports`**: `manager.go` contains significant logic for filesystem operations and string parsing (review detection) which is currently untested and prone to regression.
- **`pkg/tracking`**: Credit tracking logic (parsing JSON, handling errors) is untested.
- **`pkg/executor`**: While `vote.go` is tested, `dispatcher.go` and others lack specific unit tests, relying presumably on integration validation.

### Current Grade: 65/100
- **Core Logic:** 85/100 (Well covered)
- **Utilities:** 40/100 (Sparse coverage)
- **Integration:** Unknown (Assumed manual)

## 2. Proposed Test Plan

I propose adding three new test files to immediately boost coverage in the utility packages without requiring major refactoring.

1.  **`pkg/colors/colors_test.go`**: Verify ANSI codes are correct and non-empty.
2.  **`pkg/reports/manager_test.go`**: Comprehensive tests for report management:
    -   `FindNewestReport`: Verify correct sorting of files by modification time.
    -   `IsReportReviewed`: Verify detection of "Date Modified:" string.
    -   `ShouldSkipTask`: Verify logic for skipping tasks based on review status.
    -   `DeleteOldReports`: Verify cleanup logic preserves the newest file.
3.  **`pkg/tracking/tracking_test.go`**: Test helper functions and parsing logic:
    -   `FormatCredit`: Verify nil handling and formatting.
    -   `IsITerm2Error`: Verify error string classification.

## 3. Patch-Ready Diffs

### A. Package `colors`

```diff
diff --git a/pkg/colors/colors_test.go b/pkg/colors/colors_test.go
new file mode 100644
index 0000000..e69de29
--- /dev/null
+++ b/pkg/colors/colors_test.go
@@ -0,0 +1,28 @@
+package colors
+
+import (
+	"strings"
+	"testing"
+)
+
+func TestColors(t *testing.T) {
+	tests := []struct {
+		name  string
+		value string
+	}{
+		{"Reset", Reset},
+		{"Red", Red},
+		{"Green", Green},
+		{"Blue", Blue},
+	}
+
+	for _, tc := range tests {
+		t.Run(tc.name, func(t *testing.T) {
+			if tc.value == "" {
+				t.Errorf("color constant %s is empty", tc.name)
+			}
+			if !strings.HasPrefix(tc.value, "\033[") {
+				t.Errorf("color constant %s does not start with ANSI escape sequence", tc.name)
+			}
+		})
+	}
+}
```

### B. Package `reports`

```diff
diff --git a/pkg/reports/manager_test.go b/pkg/reports/manager_test.go
new file mode 100644
index 0000000..e69de29
--- /dev/null
+++ b/pkg/reports/manager_test.go
@@ -0,0 +1,130 @@
+package reports
+
+import (
+	"os"
+	"path/filepath"
+	"testing"
+	"time"
+)
+
+func TestFindNewestReport(t *testing.T) {
+	dir := t.TempDir()
+
+	// Create 3 files with different mod times
+	files := []string{
+		filepath.Join(dir, "old.md"),
+		filepath.Join(dir, "mid.md"),
+		filepath.Join(dir, "new.md"),
+	}
+
+	now := time.Now()
+
+	for i, f := range files {
+		if err := os.WriteFile(f, []byte("content"), 0644); err != nil {
+			t.Fatalf("failed to create file: %v", err)
+		}
+		// Set mod times apart
+		modTime := now.Add(time.Duration(i) * time.Hour)
+		if err := os.Chtimes(f, modTime, modTime); err != nil {
+			t.Fatalf("failed to chtimes: %v", err)
+		}
+	}
+
+	newest := FindNewestReport(files)
+	if newest != files[2] {
+		t.Errorf("expected newest file to be %s, got %s", files[2], newest)
+	}
+}
+
+func TestIsReportReviewed(t *testing.T) {
+	dir := t.TempDir()
+
+	tests := []struct {
+		name     string
+		content  string
+		reviewed bool
+	}{
+		{"reviewed", "Title\nDate Modified: 2026-01-01\nBody", true},
+		{"unreviewed", "Title\nTo Do\nBody", false},
+		{"reviewed_late", "1\n2\n3\n4\n5\n6\n7\n8\n9\n10\nDate Modified: 2026", false}, // > 10 lines
+	}
+
+	for _, tc := range tests {
+		t.Run(tc.name, func(t *testing.T) {
+			f := filepath.Join(dir, tc.name+".md")
+			if err := os.WriteFile(f, []byte(tc.content), 0644); err != nil {
+				t.Fatalf("failed to write file: %v", err)
+			}
+
+			if got := IsReportReviewed(f); got != tc.reviewed {
+				t.Errorf("IsReportReviewed() = %v, want %v", got, tc.reviewed)
+			}
+		})
+	}
+}
+
+func TestShouldSkipTask(t *testing.T) {
+	dir := t.TempDir()
+	shortcut := "test"
+	pattern := "report"
+
+	// Case 1: No report -> Run (False)
+	if skip := ShouldSkipTask(dir, shortcut, pattern, true); skip {
+		t.Error("expected ShouldSkipTask to be false (no report), got true")
+	}
+
+	// Case 2: Unreviewed report -> Skip (True)
+	f1 := filepath.Join(dir, pattern+"-1.md")
+	os.WriteFile(f1, []byte("Unreviewed content"), 0644)
+	if skip := ShouldSkipTask(dir, shortcut, pattern, true); !skip {
+		t.Error("expected ShouldSkipTask to be true (unreviewed), got false")
+	}
+
+	// Case 3: Reviewed report -> Run (False)
+	f2 := filepath.Join(dir, pattern+"-2.md")
+	os.WriteFile(f2, []byte("Date Modified: today\n"), 0644)
+	// Ensure f2 is newer
+	future := time.Now().Add(time.Hour)
+	os.Chtimes(f2, future, future)
+
+	if skip := ShouldSkipTask(dir, shortcut, pattern, true); skip {
+		t.Error("expected ShouldSkipTask to be false (reviewed), got true")
+	}
+}
```

### C. Package `tracking`

```diff
diff --git a/pkg/tracking/tracking_test.go b/pkg/tracking/tracking_test.go
new file mode 100644
index 0000000..e69de29
--- /dev/null
+++ b/pkg/tracking/tracking_test.go
@@ -0,0 +1,41 @@
+package tracking
+
+import (
+	"testing"
+)
+
+func TestFormatCredit(t *testing.T) {
+	val := 100
+	if got := FormatCredit(&val); got != "100" {
+		t.Errorf("FormatCredit(100) = %q, want %q", got, "100")
+	}
+
+	if got := FormatCredit(nil); got != "N/A" {
+		t.Errorf("FormatCredit(nil) = %q, want %q", got, "N/A")
+	}
+}
+
+func TestClaudeStatus_IsITerm2Error(t *testing.T) {
+	tests := []struct {
+		errStr string
+		want   bool
+	}{
+		{"not_iterm2", true},
+		{"no_iterm2_package", true},
+		{"other error", false},
+		{"", false},
+	}
+
+	for _, tc := range tests {
+		status := &ClaudeStatus{Error: tc.errStr}
+		if got := status.IsITerm2Error(); got != tc.want {
+			t.Errorf("IsITerm2Error(%q) = %v, want %v", tc.errStr, got, tc.want)
+		}
+	}
+}
+
+func TestFindPython(t *testing.T) {
+	// Smoke test to ensure it returns something non-empty
+	if py := FindPython(); py == "" {
+		t.Error("FindPython returned empty string")
+	}
+}
```