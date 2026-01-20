package runner

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"
)

// MigrateGrades scans existing report files and creates/updates .grades.json
func MigrateGrades(baseDir string) error {
	// Determine the report directory
	reportDir := filepath.Join(baseDir, "_rcodegen")
	if _, err := os.Stat(reportDir); os.IsNotExist(err) {
		return fmt.Errorf("no _rcodegen directory found in %s", baseDir)
	}

	// Find all .md files in the report directory
	pattern := filepath.Join(reportDir, "*.md")
	files, err := filepath.Glob(pattern)
	if err != nil {
		return fmt.Errorf("failed to scan report directory: %w", err)
	}

	if len(files) == 0 {
		fmt.Printf("%sNo report files found in %s%s\n", Yellow, reportDir, Reset)
		return nil
	}

	// Load existing grades (if any)
	grades, err := LoadGrades(reportDir)
	if err != nil {
		return fmt.Errorf("failed to load existing grades: %w", err)
	}

	// Build a set of existing report files to avoid duplicates
	existingFiles := make(map[string]bool)
	for _, entry := range grades.Grades {
		existingFiles[entry.ReportFile] = true
	}

	// Process each report file
	var added, skipped, failed int
	for _, filePath := range files {
		filename := filepath.Base(filePath)

		// Skip if already in grades
		if existingFiles[filename] {
			skipped++
			continue
		}

		// Parse filename to extract tool, task, and date
		tool, _, task, date, err := ParseReportFilename(filename)
		if err != nil {
			fmt.Printf("  %s⚠%s Skipping %s (invalid filename format)\n", Yellow, Reset, filename)
			failed++
			continue
		}

		// Extract grade from report
		grade, err := ExtractGradeFromReport(filePath)
		if err != nil {
			fmt.Printf("  %s⚠%s Skipping %s (no grade found)\n", Yellow, Reset, filename)
			failed++
			continue
		}

		// Add to grades (use RFC3339 for consistency with AppendGrade)
		grades.Grades = append(grades.Grades, GradeEntry{
			Date:       date.UTC().Format(time.RFC3339),
			Tool:       tool,
			Task:       task,
			Grade:      grade,
			ReportFile: filename,
		})
		added++
		fmt.Printf("  %s✓%s %s: %s %s = %.0f\n", Green, Reset, filename, tool, task, grade)
	}

	// Sort grades by date (oldest first) - parse dates for proper comparison
	sort.Slice(grades.Grades, func(i, j int) bool {
		ti, _ := time.Parse(time.RFC3339, grades.Grades[i].Date)
		tj, _ := time.Parse(time.RFC3339, grades.Grades[j].Date)
		return ti.Before(tj)
	})

	// Save grades
	if err := SaveGrades(reportDir, grades); err != nil {
		return fmt.Errorf("failed to save grades: %w", err)
	}

	// Print summary
	fmt.Println()
	fmt.Printf("%s%sMigration complete%s\n", Bold, Green, Reset)
	fmt.Printf("  Added:   %s%d%s\n", Green, added, Reset)
	fmt.Printf("  Skipped: %s%d%s (already in .grades.json)\n", Dim, skipped, Reset)
	if failed > 0 {
		fmt.Printf("  Failed:  %s%d%s (invalid filename or no grade)\n", Yellow, failed, Reset)
	}
	fmt.Printf("  Total:   %d entries in .grades.json\n", len(grades.Grades))

	return nil
}

// MigrateGradesAll scans all subdirectories for _rcodegen folders and migrates grades
func MigrateGradesAll(baseDir string) error {
	// Get list of directories
	entries, err := os.ReadDir(baseDir)
	if err != nil {
		return fmt.Errorf("failed to read directory: %w", err)
	}

	var migrated, skipped int
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		repoDir := filepath.Join(baseDir, entry.Name())
		reportDir := filepath.Join(repoDir, "_rcodegen")

		// Check if _rcodegen exists
		if _, err := os.Stat(reportDir); os.IsNotExist(err) {
			continue
		}

		// Check if there are any .md files
		files, _ := filepath.Glob(filepath.Join(reportDir, "*.md"))
		if len(files) == 0 {
			continue
		}

		fmt.Printf("\n%s%s━━━ %s ━━━%s\n", Bold, Cyan, entry.Name(), Reset)
		if err := MigrateGrades(repoDir); err != nil {
			fmt.Printf("  %s✗%s Error: %v\n", Yellow, Reset, err)
			skipped++
		} else {
			migrated++
		}
	}

	fmt.Printf("\n%s%s════════════════════════════════════════%s\n", Bold, Cyan, Reset)
	fmt.Printf("%s%s  Migration Summary%s\n", Bold, Cyan, Reset)
	fmt.Printf("%s%s════════════════════════════════════════%s\n", Bold, Cyan, Reset)
	fmt.Printf("  Repos migrated: %s%d%s\n", Green, migrated, Reset)
	if skipped > 0 {
		fmt.Printf("  Repos with errors: %s%d%s\n", Yellow, skipped, Reset)
	}

	return nil
}
