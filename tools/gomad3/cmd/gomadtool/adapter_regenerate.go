package main

import (
	"errors"
	"flag"
	"fmt"
	"io"

	"go.temporal.io/server/tools/gomad3/upgrade"
)

func runAdapterRegenerate(arguments []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("gomadtool adapter-regenerate", flag.ContinueOnError)
	flags.SetOutput(stderr)
	root := flags.String("root", ".", "Gomad v3 module root")
	module := flags.String("module", "", "adapted module path")
	version := flags.String("version", "", "new exact module version")
	approval := flags.String("approve", "", "approval digest printed by the dry run")
	if err := flags.Parse(arguments); err != nil {
		return 2
	}
	if flags.NArg() != 0 || *module == "" || *version == "" {
		if _, err := fmt.Fprintln(stderr, errors.New("adapter-regenerate requires --module and --version")); err != nil {
			return 1
		}
		return 2
	}
	status, err := upgrade.RegenerateAdapter(*root, *module, *version, *approval, stdout)
	if err != nil {
		if _, writeErr := fmt.Fprintln(stderr, err); writeErr != nil {
			return 1
		}
	}
	return status
}
