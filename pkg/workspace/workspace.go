// Package workspace manages temporary working directories for isolated
// code generation tasks, with automatic cleanup on completion.
package workspace

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type Workspace struct {
	BaseDir string
	JobID   string
	JobDir  string
}

// GenerateJobID creates YYYYMMDD-HHMMSS-{4 hex bytes}
func GenerateJobID() string {
	now := time.Now()
	b := make([]byte, 4)
	if _, err := rand.Read(b); err != nil {
		// Fallback: use nanoseconds if crypto/rand fails
		return fmt.Sprintf("%s-%08x", now.Format("20060102-150405"), now.UnixNano()&0xFFFFFFFF)
	}
	return fmt.Sprintf("%s-%s", now.Format("20060102-150405"), hex.EncodeToString(b))
}

func New(baseDir string) (*Workspace, error) {
	jobID := GenerateJobID()
	jobDir := filepath.Join(baseDir, "jobs", jobID)

	dirs := []string{
		filepath.Join(jobDir, "outputs"),
		filepath.Join(jobDir, "errors"),
		filepath.Join(jobDir, "logs"),
	}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, err
		}
	}

	return &Workspace{BaseDir: baseDir, JobID: jobID, JobDir: jobDir}, nil
}

func (w *Workspace) OutputPath(stepName string) string {
	return filepath.Join(w.JobDir, "outputs", stepName+".json")
}

func (w *Workspace) WriteOutput(stepName string, data interface{}) (string, error) {
	path := w.OutputPath(stepName)
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return "", err
	}
	defer f.Close()

	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	if err := enc.Encode(data); err != nil {
		return "", err
	}
	return path, nil
}
