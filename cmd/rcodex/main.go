package main

import (
	"rcodegen/pkg/runner"
	"rcodegen/pkg/tools/codex"

	chassis "github.com/ai8future/chassis-go/v5"
)

func main() {
	chassis.RequireMajor(5)
	tool := codex.New()
	r := runner.NewRunner(tool)
	r.RunAndExit()
}
