package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(arguments []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("planindex", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	repositoryRoot := flags.String("repository-root", ".", "repository root to validate")
	if err := flags.Parse(arguments); err != nil {
		_, _ = fmt.Fprintf(stderr, "planindex: %v\n", err)
		return 2
	}
	if flags.NArg() != 0 {
		_, _ = fmt.Fprintln(stderr, "planindex accepts no positional arguments")
		return 2
	}

	root, err := resolveRoot(*repositoryRoot)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "check Umpire plan index: %v\n", err)
		return 2
	}
	indexPath, err := resolveRepositoryFile(root, ".plans/index.json")
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "check Umpire plan index: read .plans/index.json: %v\n", err)
		return 2
	}
	encoded, err := os.ReadFile(filepath.Clean(indexPath))
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "check Umpire plan index: read .plans/index.json: %v\n", err)
		return 2
	}
	index, err := parseIndex(encoded)
	if err != nil {
		_, _ = fmt.Fprintln(stderr, err)
		return 1
	}
	findings := checkRepository(root, index)
	for _, finding := range findings {
		_, _ = fmt.Fprintln(stderr, finding)
	}
	if len(findings) != 0 {
		return 1
	}
	_, _ = fmt.Fprintln(stdout, "Umpire plan index is valid.")
	return 0
}
