package main

import (
	"flag"
	"fmt"
	"io"

	"go.temporal.io/server/tools/gomad3/internal/gomadtool/generation/protocol"
)

func runProtocolGenerate(arguments []string, stderr io.Writer) int {
	flags := flag.NewFlagSet("gomadtool protocol-generate", flag.ContinueOnError)
	flags.SetOutput(stderr)
	check := flags.Bool("check", false, "check generated files without changing them")
	root := flags.String("root", ".", "Gomad v3 module root")
	if err := flags.Parse(arguments); err != nil {
		return 2
	}
	if err := protocol.GenerateProtocols(*root, *check); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	return 0
}
