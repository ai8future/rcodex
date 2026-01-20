package runner

import (
	"reflect"
	"strings"
	"testing"
)

func TestCheckDuplicateFlags_NoConflict(t *testing.T) {
	args := []string{"-m", "claude", "-c", "do something"}
	groups := CommonFlagGroups()

	err := CheckDuplicateFlags(args, groups)
	if err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
}

func TestCheckDuplicateFlags_SameValueOK(t *testing.T) {
	// Same value specified twice should not be an error
	args := []string{"-m", "claude", "--model", "claude"}
	groups := CommonFlagGroups()

	err := CheckDuplicateFlags(args, groups)
	if err != nil {
		t.Errorf("expected no error for same values, got: %v", err)
	}
}

func TestCheckDuplicateFlags_Conflict(t *testing.T) {
	args := []string{"-m", "claude", "--model", "gpt-4"}
	groups := CommonFlagGroups()

	err := CheckDuplicateFlags(args, groups)
	if err == nil {
		t.Error("expected error for conflicting values")
	}
	if !strings.Contains(err.Error(), "conflicting") {
		t.Errorf("expected error to mention 'conflicting', got: %v", err)
	}
}

func TestCheckDuplicateFlags_EqualsFormat(t *testing.T) {
	args := []string{"-m=claude", "--model=gpt-4"}
	groups := CommonFlagGroups()

	err := CheckDuplicateFlags(args, groups)
	if err == nil {
		t.Error("expected error for conflicting values with = format")
	}
}

func TestCheckDuplicateFlags_MixedFormat(t *testing.T) {
	// -m value vs --model=value
	args := []string{"-m", "claude", "--model=gpt-4"}
	groups := CommonFlagGroups()

	err := CheckDuplicateFlags(args, groups)
	if err == nil {
		t.Error("expected error for mixed format conflicting values")
	}
}

func TestCheckDuplicateFlags_BooleanFlags(t *testing.T) {
	// Duplicate boolean flags are fine (both mean "true")
	args := []string{"-j", "--json"}
	groups := CommonFlagGroups()

	err := CheckDuplicateFlags(args, groups)
	if err != nil {
		t.Errorf("expected no error for duplicate boolean flags, got: %v", err)
	}
}

func TestParseVarFlags_Basic(t *testing.T) {
	args := []string{"-x", "foo=bar", "task", "-x", "baz=qux"}
	cleaned, vars := ParseVarFlags(args)

	expectedCleaned := []string{"task"}
	expectedVars := map[string]string{"foo": "bar", "baz": "qux"}

	if !reflect.DeepEqual(cleaned, expectedCleaned) {
		t.Errorf("cleaned args = %v, want %v", cleaned, expectedCleaned)
	}
	if !reflect.DeepEqual(vars, expectedVars) {
		t.Errorf("vars = %v, want %v", vars, expectedVars)
	}
}

func TestParseVarFlags_EqualsFormat(t *testing.T) {
	args := []string{"-x=key=value", "task"}
	cleaned, vars := ParseVarFlags(args)

	if vars["key"] != "value" {
		t.Errorf("vars[key] = %q, want 'value'", vars["key"])
	}
	if len(cleaned) != 1 || cleaned[0] != "task" {
		t.Errorf("cleaned = %v, want [task]", cleaned)
	}
}

func TestParseVarFlags_NoVars(t *testing.T) {
	args := []string{"task", "-m", "claude"}
	cleaned, vars := ParseVarFlags(args)

	if len(vars) != 0 {
		t.Errorf("expected no vars, got %v", vars)
	}
	if !reflect.DeepEqual(cleaned, args) {
		t.Errorf("cleaned = %v, want %v", cleaned, args)
	}
}

func TestParseVarFlags_InvalidFormat(t *testing.T) {
	// -x without = in value should be skipped
	args := []string{"-x", "noequals", "task"}
	cleaned, vars := ParseVarFlags(args)

	if len(vars) != 0 {
		t.Errorf("expected no vars for invalid format, got %v", vars)
	}
	if len(cleaned) != 1 {
		t.Errorf("cleaned = %v, want [task]", cleaned)
	}
}

func TestParseVarFlags_ValueWithEquals(t *testing.T) {
	// Values can contain = signs
	args := []string{"-x", "config=a=b=c", "task"}
	cleaned, vars := ParseVarFlags(args)

	if vars["config"] != "a=b=c" {
		t.Errorf("vars[config] = %q, want 'a=b=c'", vars["config"])
	}
	if len(cleaned) != 1 {
		t.Errorf("cleaned = %v, want [task]", cleaned)
	}
}

func TestCommonFlagGroups(t *testing.T) {
	groups := CommonFlagGroups()

	// Should have common flags
	found := false
	for _, g := range groups {
		for _, name := range g.Names {
			if name == "-m" || name == "--model" {
				found = true
				if !g.TakesArg {
					t.Error("-m/--model should take an argument")
				}
			}
		}
	}
	if !found {
		t.Error("expected -m/--model in CommonFlagGroups")
	}

	// Check boolean flags exist
	foundJSON := false
	for _, g := range groups {
		for _, name := range g.Names {
			if name == "-j" || name == "--json" {
				foundJSON = true
				if g.TakesArg {
					t.Error("-j/--json should not take an argument")
				}
			}
		}
	}
	if !foundJSON {
		t.Error("expected -j/--json in CommonFlagGroups")
	}
}

func TestReorderArgsForFlagParsing(t *testing.T) {
	groups := CommonFlagGroups()

	tests := []struct {
		name     string
		args     []string
		expected []string
	}{
		{
			name:     "flags already first",
			args:     []string{"-m", "claude", "task"},
			expected: []string{"-m", "claude", "task"},
		},
		{
			name:     "positional before flag",
			args:     []string{"task", "-m", "claude"},
			expected: []string{"-m", "claude", "task"},
		},
		{
			name:     "boolean flag after task",
			args:     []string{"task", "-j"},
			expected: []string{"-j", "task"},
		},
		{
			name:     "equals format",
			args:     []string{"task", "--model=claude"},
			expected: []string{"--model=claude", "task"},
		},
		{
			name:     "empty args",
			args:     []string{},
			expected: nil, // Function returns nil for empty input
		},
		{
			name:     "only flags",
			args:     []string{"-m", "claude", "-j"},
			expected: []string{"-m", "claude", "-j"},
		},
		{
			name:     "only positional",
			args:     []string{"task1", "task2"},
			expected: []string{"task1", "task2"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := reorderArgsForFlagParsing(tc.args, groups)
			if !reflect.DeepEqual(result, tc.expected) {
				t.Errorf("reorderArgsForFlagParsing(%v) = %v, want %v", tc.args, result, tc.expected)
			}
		})
	}
}

func TestReorderArgsForFlagParsing_UnknownFlags(t *testing.T) {
	groups := CommonFlagGroups()

	// Unknown flags should still be treated as flags
	args := []string{"task", "--unknown-flag"}
	result := reorderArgsForFlagParsing(args, groups)

	// Unknown flags should be moved before positional args
	if result[0] != "--unknown-flag" {
		t.Errorf("expected unknown flag first, got %v", result)
	}
}
