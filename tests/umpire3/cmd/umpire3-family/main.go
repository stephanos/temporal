package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"go.temporal.io/server/tests/umpire3/familycheck"
	"go.temporal.io/server/tests/umpire3/protocol"
)

func main() {
	family := flag.String("family", "", "catalog model-family identifier")
	repositoryRoot := flag.String("repository-root", ".", "Temporal repository root")
	flag.Parse()
	if *family == "" {
		fmt.Fprintln(os.Stderr, "Umpire3 model family is required")
		os.Exit(2)
	}
	graph, err := protocol.DefaultFamilyDependencyGraph()
	if err == nil {
		var plan familycheck.Plan
		plan, err = familycheck.PlanFor(graph, protocol.TargetID(*family), *repositoryRoot)
		if err == nil {
			err = familycheck.Run(context.Background(), plan, familycheck.ExecRunner{})
		}
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
