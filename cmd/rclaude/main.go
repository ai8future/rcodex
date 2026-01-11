package main

import (
	"rcodegen/pkg/runner"
	"rcodegen/pkg/tools/claude"
)

func main() {
	tool := claude.New()
	r := runner.NewRunner(tool)
	r.RunAndExit()
}
