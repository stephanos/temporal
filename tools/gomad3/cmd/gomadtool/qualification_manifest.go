package main

import (
	"flag"
	"fmt"
	"io"

	"go.temporal.io/server/tools/gomad3/qualification/set/manifestgen"
)

func runQualificationManifestGenerate(arguments []string, stderr io.Writer) int {
	flags := flag.NewFlagSet("gomadtool qualification-manifest-generate", flag.ContinueOnError)
	flags.SetOutput(stderr)
	check := flags.Bool("check", false, "check the generated manifest without changing it")
	root := flags.String("root", ".", "root the spec, output, and enumerated package resolve against")
	spec := flags.String("spec", "", "generator spec: defaults, per-test overrides, and exclusions")
	output := flags.String("output", "", "generated qualification-set manifest")
	if err := flags.Parse(arguments); err != nil || flags.NArg() != 0 {
		return 2
	}
	if *spec == "" || *output == "" {
		if _, writeErr := fmt.Fprintln(stderr, "qualification-manifest-generate requires --spec and --output"); writeErr != nil {
			return 2
		}
		return 2
	}
	if err := manifestgen.Run(manifestgen.Config{Root: *root, Spec: *spec, Output: *output, Check: *check}); err != nil {
		if _, writeErr := fmt.Fprintln(stderr, err); writeErr != nil {
			return 1
		}
		return 1
	}
	return 0
}
