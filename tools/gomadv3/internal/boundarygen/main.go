package main

import (
	"errors"
	"flag"
	"fmt"
	"os"

	"go.temporal.io/server/tools/gomadv3/internal/boundary"
)

func main() {
	check := flag.Bool("check", false, "check generated files without changing them")
	discover := flag.Bool("discover", false, "list source-discovered host-capability entry points")
	qualify := flag.Bool("qualify", false, "type-check manifest signatures against this Go toolchain")
	refresh := flag.Bool("refresh", false, "refresh source fingerprints in the manifest")
	checkCompilerTests := flag.Bool("check-compiler-tests", false, "validate the compiler conformance-test manifest")
	compilerTestOverlay := flag.String("compiler-test-overlay", "", "emit a compiler conformance-test overlay")
	goroot := flag.String("goroot", "", "installed production GOROOT for the compiler test overlay")
	root := flag.String("root", ".", "Gomad v3 module root")
	flag.Parse()
	var err error
	if *compilerTestOverlay != "" {
		if *goroot == "" {
			err = errors.New("-goroot is required with -compiler-test-overlay")
		} else {
			err = boundary.GenerateCompilerTestOverlay(*root, *goroot, *compilerTestOverlay)
		}
	} else if *checkCompilerTests {
		err = boundary.CheckCompilerTests(*root)
	} else if *discover {
		var candidates []string
		candidates, err = boundary.DiscoverCandidates()
		for _, candidate := range candidates {
			fmt.Println(candidate)
		}
	} else if *refresh {
		err = boundary.RefreshFingerprints(*root)
	} else if *qualify {
		err = boundary.Qualify(*root)
		if err == nil {
			err = boundary.CheckCandidateCoverage(*root)
		}
	} else {
		err = boundary.Generate(*root, *check)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
