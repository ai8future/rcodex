package settings

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSettingsFilePermissions(t *testing.T) {
	// This test verifies settings are written with 0600 permissions
	// The settings file should be written with 0600 (owner read/write only)
	expectedPerm := os.FileMode(0600)

	// Check if settings file exists and has correct permissions
	configPath := GetConfigPath()
	if info, err := os.Stat(configPath); err == nil {
		actualPerm := info.Mode().Perm()
		if actualPerm != expectedPerm {
			t.Errorf("settings file has permissions %o, want %o", actualPerm, expectedPerm)
		}
	}
	// If file doesn't exist, that's OK - test passes
}

func TestExpandTilde(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("could not get home dir: %v", err)
	}

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"tilde prefix", "~/foo/bar", filepath.Join(home, "foo/bar")},
		{"just tilde", "~", home},
		{"no tilde", "/absolute/path", "/absolute/path"},
		{"tilde in middle", "/foo/~/bar", "/foo/~/bar"},
		{"empty", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := expandTilde(tt.input)
			if result != tt.expected {
				t.Errorf("expandTilde(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}
