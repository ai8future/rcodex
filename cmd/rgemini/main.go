package main

import (
	"rcodegen/pkg/runner"
	"rcodegen/pkg/tools/gemini"

	chassis "github.com/ai8future/chassis-go"
)

func main() {
	chassis.RequireMajor(4)
	tool := gemini.New()
	r := runner.NewRunner(tool)
	r.RunAndExit()
}
