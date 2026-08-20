package main

import (
	"flag"
	"fmt"
	"os"

	"go.temporal.io/server/tests/umpire3/protocol"
)

func main() {
	leanVersion := flag.String("lean-version", "", "Lean toolchain version")
	flag.Parse()
	if *leanVersion == "" {
		fmt.Fprintln(os.Stderr, "-lean-version is required")
		os.Exit(2)
	}
	if err := protocol.WriteManifest(os.Stdout, protocol.NewEmptyManifest(*leanVersion)); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
