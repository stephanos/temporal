package main

import (
	"flag"
	"fmt"
	"io"
	"os"

	"go.temporal.io/server/tools/gomadv3/internal/buildkey"
	"go.temporal.io/server/tools/gomadv3/internal/testtier"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(arguments []string, stdout, stderr io.Writer) int {
	if len(arguments) == 0 {
		fmt.Fprintln(stderr, "usage: hosttool build-key|test-mode [flags]")
		return 2
	}
	switch arguments[0] {
	case "build-key":
		return runBuildKey(arguments[1:], stdout, stderr)
	case "test-mode":
		return runTestMode(arguments[1:], stdout, stderr)
	default:
		fmt.Fprintln(stderr, "usage: hosttool build-key|test-mode [flags]")
		return 2
	}
}

func runBuildKey(arguments []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("hosttool build-key", flag.ContinueOnError)
	flags.SetOutput(stderr)
	var input buildkey.Input
	var patch, overlay string
	flags.StringVar(&input.GoVersion, "go-version", "", "Go release")
	flags.StringVar(&input.ArchiveSHA256, "archive-sha256", "", "source archive digest")
	flags.StringVar(&patch, "patch", "", "patch path")
	flags.StringVar(&overlay, "overlay", "", "overlay root")
	flags.StringVar(&input.HostOS, "host-os", "", "host operating system")
	flags.StringVar(&input.HostArch, "host-arch", "", "host architecture")
	flags.StringVar(&input.BootstrapVersion, "bootstrap-version", "", "bootstrap Go version")
	flags.StringVar(&input.RecipeVersion, "recipe-version", "", "build recipe version")
	flags.StringVar(&input.BuildPath, "build-path", "", "sterile build PATH")
	flags.StringVar(&input.BashPath, "bash-path", "", "build bash path")
	flags.StringVar(&input.BashVersion, "bash-version", "", "build bash version")
	if err := flags.Parse(arguments); err != nil || flags.NArg() != 0 || patch == "" || overlay == "" {
		return 2
	}
	var err error
	input.PatchSHA256, err = buildkey.FileDigest(patch)
	if err != nil {
		fmt.Fprintf(stderr, "hash patch: %v\n", err)
		return 1
	}
	input.OverlaySHA256, err = buildkey.TreeDigest(overlay)
	if err != nil {
		fmt.Fprintf(stderr, "hash overlay: %v\n", err)
		return 1
	}
	key, err := buildkey.Compute(input)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	fmt.Fprintln(stdout, key)
	return 0
}

func runTestMode(arguments []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("hosttool test-mode", flag.ContinueOnError)
	flags.SetOutput(stderr)
	modeName := flags.String("mode", "", "test mode")
	output := flags.String("output", "", "tiers or success")
	if err := flags.Parse(arguments); err != nil || flags.NArg() != 0 {
		return 2
	}
	mode, err := testtier.Resolve(*modeName)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	switch *output {
	case "tiers":
		for _, tier := range mode.Tiers {
			if _, err := fmt.Fprintln(stdout, tier); err != nil {
				fmt.Fprintln(stderr, err)
				return 1
			}
		}
	case "success":
		if _, err := fmt.Fprintln(stdout, mode.Success); err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
	default:
		fmt.Fprintln(stderr, "test-mode output must be tiers or success")
		return 2
	}
	return 0
}
