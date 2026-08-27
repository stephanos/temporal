package main

import (
	"flag"
	"fmt"
	"os"

	"go.temporal.io/server/tools/umpire/internal/legacyvocabulary"
)

func main() {
	repositoryRoot := flag.String("repository-root", ".", "repository root to validate")
	flag.Parse()
	if flag.NArg() != 0 {
		fmt.Fprintln(os.Stderr, "umpire-check-legacy-vocabulary accepts no positional arguments")
		os.Exit(2)
	}

	violations, err := legacyvocabulary.Check(*repositoryRoot)
	if err != nil {
		fmt.Fprintf(os.Stderr, "check legacy Umpire vocabulary: %v\n", err)
		os.Exit(2)
	}
	for _, violation := range violations {
		fmt.Fprintf(os.Stderr, "%s:%d: retired Umpire vocabulary %s\n", violation.Path, violation.Line, violation.Token)
	}
	if len(violations) != 0 {
		os.Exit(1)
	}
}
