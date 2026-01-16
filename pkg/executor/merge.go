package executor

import (
	"fmt"
	"os"
	"strings"

	"rcodegen/pkg/bundle"
	"rcodegen/pkg/envelope"
	"rcodegen/pkg/orchestrator"
	"rcodegen/pkg/workspace"
)

type MergeExecutor struct {
	ToolExecutor *ToolExecutor
}

func (e *MergeExecutor) Execute(step *bundle.Step, ctx *orchestrator.Context, ws *workspace.Workspace) (*envelope.Envelope, error) {
	// Collect inputs
	var contents []string
	var failedInputs []string
	for _, inputRef := range step.Merge.Inputs {
		path := ctx.Resolve(inputRef)
		data, err := os.ReadFile(path)
		if err != nil {
			failedInputs = append(failedInputs, fmt.Sprintf("%s: %v", inputRef, err))
			continue
		}
		contents = append(contents, string(data))
	}

	var merged string
	switch step.Merge.Strategy {
	case "concat":
		merged = strings.Join(contents, "\n\n---\n\n")
	case "union", "dedupe":
		// For now, just concat - could add deduplication later
		merged = strings.Join(contents, "\n\n")
	default:
		merged = strings.Join(contents, "\n\n")
	}

	// Write merged output
	outputPath, err := ws.WriteOutput(step.Name, map[string]interface{}{
		"merged":      merged,
		"input_count": len(contents),
	})
	if err != nil {
		return envelope.New().Failure("WRITE_ERROR", err.Error()).Build(), err
	}

	return envelope.New().
		Success().
		WithOutputRef(outputPath).
		WithResult("input_count", len(contents)).
		WithResult("failed_inputs", failedInputs).
		Build(), nil
}
