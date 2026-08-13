package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"

	"go.temporal.io/server/tools/gomadv3/internal/qualificationset"
)

type runSetFunc func(context.Context, qualificationset.Config) (qualificationset.SetReport, error)
type loadManifestFunc func(string) (qualificationset.Manifest, error)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr, qualificationset.Run, qualificationset.LoadManifest))
}

func run(arguments []string, stdout, stderr io.Writer, runSet runSetFunc, loadManifest loadManifestFunc) int {
	flags := flag.NewFlagSet("qualificationgen", flag.ContinueOnError)
	flags.SetOutput(stderr)
	manifest := flags.String("manifest", "", "qualification set manifest")
	gomad := flags.String("gomad", ".bin/gomad", "gomad executable")
	workingDirectory := flags.String("working-dir", "", "target repository working directory")
	artifacts := flags.String("artifacts", ".toolchain/qualification", "qualification artifact root")
	output := flags.String("output", ".toolchain/qualification-set.json", "aggregate qualification report")
	check := flags.Bool("check", false, "validate the qualification set manifest without executing it")
	if err := flags.Parse(arguments); err != nil || flags.NArg() != 0 {
		return 2
	}
	if *manifest == "" || *workingDirectory == "" {
		if _, err := fmt.Fprintln(stderr, "qualification set requires explicit -manifest and -working-dir paths"); err != nil {
			return 1
		}
		return 2
	}
	if *check {
		loaded, err := loadManifest(*manifest)
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		fmt.Fprintf(stdout, "gomadv3 qualification set manifest: name=%s suites=%d\n", loaded.Name, len(loaded.Suites))
		return 0
	}
	report, err := runSet(context.Background(), qualificationset.Config{
		ManifestPath: *manifest, GomadPath: *gomad, WorkingDir: *workingDirectory, ArtifactRoot: *artifacts, OutputPath: *output,
	})
	if err != nil {
		fmt.Fprintln(stderr, err)
	}
	fmt.Fprintf(stdout, "gomadv3 qualification set: name=%s qualified=%t completed=%d/%d report=%s\n", report.Name, report.Qualified, report.Completed, report.Selected, *output)
	if err != nil {
		return 1
	}
	return 0
}
