package workspace

import (
	"os"
	"path/filepath"
	"regexp"
	"testing"
)

func TestGenerateJobID_Format(t *testing.T) {
	jobID := GenerateJobID()

	// Format: YYYYMMDD-HHMMSS-{8 hex chars}
	pattern := regexp.MustCompile(`^\d{8}-\d{6}-[a-f0-9]{8}$`)
	if !pattern.MatchString(jobID) {
		t.Errorf("job ID %q does not match expected format YYYYMMDD-HHMMSS-{8 hex}", jobID)
	}
}

func TestGenerateJobID_Unique(t *testing.T) {
	ids := make(map[string]bool)
	for i := 0; i < 100; i++ {
		id := GenerateJobID()
		if ids[id] {
			t.Errorf("duplicate job ID generated: %s", id)
		}
		ids[id] = true
	}
}

func TestNew_CreatesDirectories(t *testing.T) {
	// Use a temp directory
	tmpDir := t.TempDir()

	ws, err := New(tmpDir)
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}

	// Check that job directory was created
	if _, err := os.Stat(ws.JobDir); os.IsNotExist(err) {
		t.Errorf("job directory not created: %s", ws.JobDir)
	}

	// Check subdirectories
	for _, subdir := range []string{"outputs", "errors", "logs"} {
		path := filepath.Join(ws.JobDir, subdir)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("subdirectory not created: %s", path)
		}
	}
}

func TestWorkspace_OutputPath(t *testing.T) {
	ws := &Workspace{
		JobDir: "/tmp/test-job",
	}

	path := ws.OutputPath("step1")
	expected := "/tmp/test-job/outputs/step1.json"
	if path != expected {
		t.Errorf("OutputPath() = %q, want %q", path, expected)
	}
}

func TestWorkspace_WriteOutput(t *testing.T) {
	tmpDir := t.TempDir()
	ws, err := New(tmpDir)
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}

	data := map[string]interface{}{
		"key":   "value",
		"count": 42,
	}

	path, err := ws.WriteOutput("test-step", data)
	if err != nil {
		t.Fatalf("WriteOutput() error: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Errorf("output file not created: %s", path)
	}

	// Verify content
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("could not read output file: %v", err)
	}

	// Should contain our data
	if !regexp.MustCompile(`"key":\s*"value"`).Match(content) {
		t.Errorf("output file missing expected content: %s", content)
	}
}
