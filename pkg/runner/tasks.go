// Package runner provides the core execution framework for rcodegen tools.
package runner

// Task type constants for the standard rcodegen report types.
// These should be used instead of magic strings throughout the codebase.
const (
	TaskAudit    = "audit"
	TaskTest     = "test"
	TaskFix      = "fix"
	TaskRefactor = "refactor"
	TaskQuick    = "quick"
	TaskSuite    = "suite" // Special: runs all report types
)

// ReportTypes contains the ordered list of all standard report types.
// Used when running suite mode or iterating over report types.
var ReportTypes = []string{TaskAudit, TaskTest, TaskFix, TaskRefactor, TaskQuick}
