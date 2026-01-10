package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	"rcodegen/pkg/bundle"
	"rcodegen/pkg/orchestrator"
	"rcodegen/pkg/settings"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "bundle", "run":
		runBundle()
	case "list":
		listBundles()
	case "help", "-h", "--help":
		printUsage()
	default:
		// Treat as bundle name shortcut
		os.Args = append([]string{os.Args[0], "bundle"}, os.Args[1:]...)
		runBundle()
	}
}

func runBundle() {
	fs := flag.NewFlagSet("bundle", flag.ExitOnError)
	codebase := fs.String("c", "", "Codebase path")
	jsonOutput := fs.Bool("j", false, "Output JSON")

	fs.Parse(os.Args[2:])

	bundleName := fs.Arg(0)
	if bundleName == "" {
		fmt.Fprintln(os.Stderr, "Error: bundle name required")
		os.Exit(1)
	}

	// Parse remaining args as inputs (key=value)
	inputs := make(map[string]string)
	if *codebase != "" {
		inputs["codebase"] = expandPath(*codebase)
	}

	for _, arg := range fs.Args()[1:] {
		if idx := strings.Index(arg, "="); idx != -1 {
			inputs[arg[:idx]] = arg[idx+1:]
		}
	}

	// Load settings
	s, _ := settings.LoadWithFallback()

	// Load bundle
	b, err := bundle.Load(bundleName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Run
	orch := orchestrator.New(s)
	env, err := orch.Run(b, inputs)

	if *jsonOutput {
		json.NewEncoder(os.Stdout).Encode(env)
	}

	if err != nil || env.Status != "success" {
		os.Exit(1)
	}
}

func listBundles() {
	names, _ := bundle.List()
	fmt.Println("Available bundles:")
	for _, name := range names {
		b, err := bundle.Load(name)
		if err == nil {
			fmt.Printf("  %s - %s\n", name, b.Description)
		} else {
			fmt.Printf("  %s\n", name)
		}
	}
}

func printUsage() {
	fmt.Println(`rcodegen - Multi-tool orchestrator

Usage:
  rcodegen <bundle> -c <codebase> [inputs...]
  rcodegen bundle <name> -c <codebase> [key=value...]
  rcodegen list

Examples:
  rcodegen compete -c myproject task="audit for security issues"
  rcodegen security-review -c myproject
  rcodegen list`)
}

func expandPath(path string) string {
	if strings.HasPrefix(path, "~/") {
		return os.Getenv("HOME") + path[1:]
	}
	return path
}
