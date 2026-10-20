package main

import (
	"flag"
	"fmt"
	"io"

	"go.temporal.io/server/tools/gomad/internal/backends/native/generation/protocolnative"
)

func runProtocolGenerate(arguments []string, stderr io.Writer) int {
	flags := flag.NewFlagSet("gomadtool protocol-generate", flag.ContinueOnError)
	flags.SetOutput(stderr)
	check := flags.Bool("check", false, "check generated files without changing them")
	root := flags.String("root", ".", "Gomad module root")
	if err := flags.Parse(arguments); err != nil {
		return 2
	}
	stale, err := protocolnative.Generate(*root, *check)
	if err != nil {
		_, _ = fmt.Fprintln(stderr, err)
		return 1
	}
	for _, path := range stale {
		_, _ = fmt.Fprintf(stderr, "generated native protocol output is stale: %s\n", path)
	}
	if len(stale) != 0 {
		return 1
	}
	return 0
}
