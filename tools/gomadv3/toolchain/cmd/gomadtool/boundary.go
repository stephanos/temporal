package main

import (
	"errors"
	"flag"
	"fmt"
	"io"

	"go.temporal.io/server/tools/gomadv3/toolchain/internal/generate"
)

func runBoundaryGenerate(arguments []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("gomadtool boundary-generate", flag.ContinueOnError)
	flags.SetOutput(stderr)
	check := flags.Bool("check", false, "check generated files without changing them")
	discover := flags.Bool("discover", false, "list source-discovered host-capability entry points")
	qualify := flags.Bool("qualify", false, "type-check manifest signatures against this Go toolchain")
	refresh := flags.Bool("refresh", false, "refresh source fingerprints in the manifest")
	checkCompilerTests := flags.Bool("check-compiler-tests", false, "validate the compiler conformance-test manifest")
	compilerTestOverlay := flags.String("compiler-test-overlay", "", "emit a compiler conformance-test overlay")
	goroot := flags.String("goroot", "", "installed production GOROOT for the compiler test overlay")
	root := flags.String("root", ".", "Gomad v3 module root")
	if err := flags.Parse(arguments); err != nil {
		return 2
	}
	var err error
	if *compilerTestOverlay != "" {
		if *goroot == "" {
			err = errors.New("-goroot is required with -compiler-test-overlay")
		} else {
			err = generate.GenerateCompilerTestOverlay(*root, *goroot, *compilerTestOverlay)
		}
	} else if *checkCompilerTests {
		err = generate.CheckCompilerTests(*root)
	} else if *discover {
		var candidates []string
		candidates, err = generate.DiscoverCandidates()
		for _, candidate := range candidates {
			fmt.Fprintln(stdout, candidate)
		}
	} else if *refresh {
		err = generate.RefreshFingerprints(*root)
	} else if *qualify {
		err = generate.Qualify(*root)
		if err == nil {
			err = generate.CheckCandidateCoverage(*root)
		}
	} else {
		err = generate.Generate(*root, *check)
	}
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	return 0
}
