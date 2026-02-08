package main

import (
	"rcodegen/pkg/runner"
	"rcodegen/pkg/tools/gemini"

	chassis "github.com/ai8future/chassis-go/v5"
)

func main() {
	chassis.RequireMajor(5)
	tool := gemini.New()
	r := runner.NewRunner(tool)
	r.RunAndExit()
}
