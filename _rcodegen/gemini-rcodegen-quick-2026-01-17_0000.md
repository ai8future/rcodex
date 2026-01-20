Date Created: 2026-01-17 00:00:00
TOTAL_SCORE: 85/100

# 1. AUDIT

## Security
*   **Permissions Bypass**: `pkg/tools/claude/claude.go` runs `claude` with `--dangerously-skip-permissions`. While necessary for automation, it exposes the system to potentially malicious tool use if the LLM is compromised.
*   **Unauthenticated API**: `claude_question_handler.py` interacts with iTerm2's Python API. While local, it accepts commands without validation, which could be abused if a malicious script runs on the machine.

## Code Quality
*   **Error Swallowing**: `pkg/runner/grades.go` in `LoadGrades` returns an empty grade file if `os.ReadFile` fails (unless it's a `NotExists` error, but the check is imperfect). It should bubble up errors.
*   **Hardcoded Models**: `pkg/tools/claude/claude.go` contains a hardcoded list of models (`sonnet`, `opus`, `haiku`) which is outdated (missing `sonnet-3-5`).
*   **Brittle Parsing**: `pkg/executor/vote.go` uses manual string indexing to parse `${steps...}` references, which is error-prone.

# 2. TESTS

## pkg/runner/stream.go
The `extractToolInfo` and `shortenPath` functions contain complex logic with many edge cases but lack unit tests.

```go
package runner

import (
	"encoding/json"
	"os"
	"testing"
)

func TestExtractToolInfo(t *testing.T) {
	tests := []struct {
		name     string
		toolName string
		input    map[string]interface{}
		want     string
	}{
		{
			name:     "Bash command short",
			toolName: "Bash",
			input:    map[string]interface{}{"command": "ls -la"},
			want:     "ls -la",
		},
		{
			name:     "Bash command long",
			toolName: "Bash",
			input:    map[string]interface{}{"command": "this is a very long command that should be truncated because it exceeds the limit"},
			want:     "this is a very long command that should be truncated beca...",
		},
		{
			name:     "Read file",
			toolName: "Read",
			input:    map[string]interface{}{"file_path": "/tmp/test.txt"},
			want:     "/tmp/test.txt",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractToolInfo(tt.toolName, tt.input)
			if got != tt.want {
				t.Errorf("extractToolInfo() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestShortenPath(t *testing.T) {
	home, _ := os.UserHomeDir()
	tests := []struct {
		name string
		path string
		want string
	}{
		{"Absolute path", "/tmp/file", "/tmp/file"},
		{"Home path", home + "/project", "~/project"},
		{"Root path", "/", "/"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := shortenPath(tt.path)
			if got != tt.want {
				t.Errorf("shortenPath() = %v, want %v", got, tt.want)
			}
		})
	}
}
```

# 3. FIXES

## Fix brittle step name extraction
The current implementation fails for inputs like `${steps.name}` (returns empty string) and uses fragile indexing.

`pkg/executor/vote.go`:
```go
func extractStepName(ref string) string {
	// ${steps.name.output_ref} -> name
	if len(ref) > 9 && ref[:8] == "${steps." {
		end := 8
		for i := 8; i < len(ref); i++ {
			if ref[i] == '.' {
				return ref[8:i]
			}
		}
		return ref[8:end]
	}
	return ref
}
====
func extractStepName(ref string) string {
	// ${steps.name.output_ref} -> name
	if strings.HasPrefix(ref, "${steps.") && strings.HasSuffix(ref, "}") {
		content := ref[8 : len(ref)-1] // Remove ${steps. and }
		parts := strings.Split(content, ".")
		if len(parts) > 0 {
			return parts[0]
		}
	}
	return ref
}
>>>>

## Update Claude models
Add support for the newer `sonnet-3-5` model.

`pkg/tools/claude/claude.go`:
```go
<<<<
// ValidModels returns the list of valid model names
func (t *Tool) ValidModels() []string {
	return []string{"sonnet", "opus", "haiku"}
}
====
// ValidModels returns the list of valid model names
func (t *Tool) ValidModels() []string {
	return []string{"sonnet", "sonnet-3-5", "opus", "haiku"}
}
>>>>

## Fix silent runlog failure
If `MkdirAll` fails, the tool currently prints a warning and returns, but this might mask issues in CI/CD environments. It should probably log to stderr more aggressively or handle the error better. For now, let's at least ensure the warning is visible and we don't proceed to try writing the file.

`pkg/runner/runner.go`:
```go
<<<<
	// Create directory if it doesn't exist
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "%sWarning:%s Could not create runlog directory: %v\n", Yellow, Reset, err)
		return
	}
====
	// Create directory if it doesn't exist
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "%sError:%s Could not create runlog directory: %v\n", Red, Reset, err)
		return
	}
>>>>

# 4. REFACTOR

## Split `pkg/runner/runner.go`
The `Runner` struct and its methods (`Run`, `parseArgs`, `executeCommand`, logging, summary) comprise a "God class" of ~500 lines.
*   **Action**: Move argument parsing to a new `pkg/runner/args.go`.
*   **Action**: Move summary printing to `pkg/runner/summary.go`.
*   **Action**: Move execution logic (`runSingleTask`, `executeCommand`) to `pkg/runner/exec.go`.

## Use `pkg/lock` in `pkg/runner/grades.go`
`grades.go` uses a process-local `sync.Mutex` (`gradesFileMutex`). This protects against concurrent goroutines but not against concurrent CLI processes (e.g., running `rcodegen` in two terminals).
*   **Action**: Replace `sync.Mutex` with `lock.FileLock` from `rcodegen/pkg/lock` to ensure inter-process safety when updating `.grades.json`.

```