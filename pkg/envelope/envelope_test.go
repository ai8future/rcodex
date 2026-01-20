package envelope

import (
	"testing"
)

func TestNew(t *testing.T) {
	b := New()
	if b == nil {
		t.Fatal("New() returned nil")
	}
	if b.env == nil {
		t.Error("builder envelope is nil")
	}
	if b.env.Result == nil {
		t.Error("Result map should be initialized")
	}
}

func TestBuilder_Success(t *testing.T) {
	env := New().Success().Build()

	if env.Status != StatusSuccess {
		t.Errorf("expected StatusSuccess, got %s", env.Status)
	}
}

func TestBuilder_Failure(t *testing.T) {
	env := New().Failure("ERR_CODE", "Something went wrong").Build()

	if env.Status != StatusFailure {
		t.Errorf("expected StatusFailure, got %s", env.Status)
	}
	if env.Error == nil {
		t.Fatal("expected Error to be set")
	}
	if env.Error.Code != "ERR_CODE" {
		t.Errorf("expected error code 'ERR_CODE', got %s", env.Error.Code)
	}
	if env.Error.Message != "Something went wrong" {
		t.Errorf("expected error message, got %s", env.Error.Message)
	}
}

func TestBuilder_WithResult(t *testing.T) {
	env := New().
		Success().
		WithResult("count", 42).
		WithResult("name", "test").
		WithResult("active", true).
		Build()

	if env.Result["count"] != 42 {
		t.Errorf("expected count=42, got %v", env.Result["count"])
	}
	if env.Result["name"] != "test" {
		t.Errorf("expected name='test', got %v", env.Result["name"])
	}
	if env.Result["active"] != true {
		t.Errorf("expected active=true, got %v", env.Result["active"])
	}
}

func TestBuilder_WithOutputRef(t *testing.T) {
	env := New().
		Success().
		WithOutputRef("/tmp/output.json").
		Build()

	if env.OutputRef != "/tmp/output.json" {
		t.Errorf("expected OutputRef='/tmp/output.json', got %s", env.OutputRef)
	}
}

func TestBuilder_WithTool(t *testing.T) {
	env := New().WithTool("claude").Build()

	if env.Metrics == nil {
		t.Fatal("expected Metrics to be initialized")
	}
	if env.Metrics.Tool != "claude" {
		t.Errorf("expected tool='claude', got %s", env.Metrics.Tool)
	}
}

func TestBuilder_WithDuration(t *testing.T) {
	env := New().WithDuration(1500).Build()

	if env.Metrics == nil {
		t.Fatal("expected Metrics to be initialized")
	}
	if env.Metrics.DurationMs != 1500 {
		t.Errorf("expected DurationMs=1500, got %d", env.Metrics.DurationMs)
	}
}

func TestBuilder_Chaining(t *testing.T) {
	// Test full fluent builder pattern
	env := New().
		WithTool("gemini").
		WithDuration(2000).
		WithOutputRef("/output/path.json").
		WithResult("tokens", 100).
		Success().
		Build()

	if env.Status != StatusSuccess {
		t.Errorf("status: got %s, want success", env.Status)
	}
	if env.Metrics.Tool != "gemini" {
		t.Errorf("tool: got %s, want gemini", env.Metrics.Tool)
	}
	if env.Metrics.DurationMs != 2000 {
		t.Errorf("duration: got %d, want 2000", env.Metrics.DurationMs)
	}
	if env.OutputRef != "/output/path.json" {
		t.Errorf("output_ref: got %s, want /output/path.json", env.OutputRef)
	}
	if env.Result["tokens"] != 100 {
		t.Errorf("result[tokens]: got %v, want 100", env.Result["tokens"])
	}
}

func TestBuilder_MultipleWithTool(t *testing.T) {
	// Calling WithTool or WithDuration multiple times should use same Metrics
	env := New().
		WithTool("claude").
		WithDuration(100).
		WithTool("gemini").
		WithDuration(200).
		Build()

	if env.Metrics.Tool != "gemini" {
		t.Errorf("expected last tool 'gemini', got %s", env.Metrics.Tool)
	}
	if env.Metrics.DurationMs != 200 {
		t.Errorf("expected last duration 200, got %d", env.Metrics.DurationMs)
	}
}

func TestStatusConstants(t *testing.T) {
	// Verify status constants have expected values
	if StatusSuccess != "success" {
		t.Errorf("StatusSuccess = %q, want 'success'", StatusSuccess)
	}
	if StatusFailure != "failure" {
		t.Errorf("StatusFailure = %q, want 'failure'", StatusFailure)
	}
	if StatusPartial != "partial" {
		t.Errorf("StatusPartial = %q, want 'partial'", StatusPartial)
	}
	if StatusSkipped != "skipped" {
		t.Errorf("StatusSkipped = %q, want 'skipped'", StatusSkipped)
	}
}
