package runner

import (
	"fmt"
	"testing"
)

func TestRunError(t *testing.T) {
	result := runError(1, fmt.Errorf("test error"))

	if result.ExitCode != 1 {
		t.Errorf("expected exit code 1, got %d", result.ExitCode)
	}
	if result.Error == nil {
		t.Error("expected error to be set")
	}
	if result.Error.Error() != "test error" {
		t.Errorf("expected error message 'test error', got %q", result.Error.Error())
	}
}

func TestRunResult_SuccessResult(t *testing.T) {
	result := &RunResult{
		ExitCode:     0,
		TokenUsage:   &TokenUsage{InputTokens: 100, OutputTokens: 50},
		TotalCostUSD: 0.0015,
	}

	if result.ExitCode != 0 {
		t.Errorf("expected exit code 0, got %d", result.ExitCode)
	}
	if result.Error != nil {
		t.Errorf("expected no error, got %v", result.Error)
	}
	if result.TokenUsage == nil {
		t.Error("expected TokenUsage to be set")
	}
	if result.TokenUsage.InputTokens != 100 {
		t.Errorf("expected input tokens 100, got %d", result.TokenUsage.InputTokens)
	}
	if result.TotalCostUSD != 0.0015 {
		t.Errorf("expected cost 0.0015, got %f", result.TotalCostUSD)
	}
}

func TestRunError_DifferentCodes(t *testing.T) {
	tests := []struct {
		code int
		msg  string
	}{
		{0, "success with error message"},
		{1, "general error"},
		{2, "usage error"},
		{127, "command not found"},
	}

	for _, tc := range tests {
		result := runError(tc.code, fmt.Errorf("%s", tc.msg))
		if result.ExitCode != tc.code {
			t.Errorf("runError(%d, %q): expected exit code %d, got %d",
				tc.code, tc.msg, tc.code, result.ExitCode)
		}
		if result.Error.Error() != tc.msg {
			t.Errorf("runError(%d, %q): expected error message %q, got %q",
				tc.code, tc.msg, tc.msg, result.Error.Error())
		}
	}
}
