package main

import (
	"flag"
	"fmt"
	"io"

	"go.temporal.io/server/tools/gomad3/toolchain/version"
)

func runVersionGenerate(arguments []string, stderr io.Writer) int {
	flags := flag.NewFlagSet("gomadtool version-generate", flag.ContinueOnError)
	flags.SetOutput(stderr)
	check := flags.Bool("check", false, "check generated files without changing them")
	root := flags.String("root", ".", "Gomad v3 module root")
	if err := flags.Parse(arguments); err != nil {
		return 2
	}
	if err := version.Generate(*root, *check); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	return 0
}
