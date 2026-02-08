package main

import (
	"rcodegen/pkg/runner"
	"rcodegen/pkg/tools/claude"

	chassis "github.com/ai8future/chassis-go"
)

func main() {
	chassis.RequireMajor(4)
	tool := claude.New()
	r := runner.NewRunner(tool)
	r.RunAndExit()
}
