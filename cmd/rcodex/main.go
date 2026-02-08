package main

import (
	"rcodegen/pkg/runner"
	"rcodegen/pkg/tools/codex"

	chassis "github.com/ai8future/chassis-go"
)

func main() {
	chassis.RequireMajor(4)
	tool := codex.New()
	r := runner.NewRunner(tool)
	r.RunAndExit()
}
