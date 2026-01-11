package main

import (
	"rcodegen/pkg/runner"
	"rcodegen/pkg/tools/codex"
)

func main() {
	tool := codex.New()
	r := runner.NewRunner(tool)
	r.RunAndExit()
}
