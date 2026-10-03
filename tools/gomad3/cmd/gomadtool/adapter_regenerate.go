package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"go.temporal.io/server/tools/gomad3/upgrade/adapterregen"
)

const adapterRegenerateUsage = "usage: gomadtool adapter-regenerate [--root=DIR] --module=PATH --version=VERSION [--approve-review=SHA256] [--go=GO] [--json]\n" +
	"       gomadtool adapter-regenerate [--root=DIR] --recover\n" +
	"       gomadtool adapter-regenerate --verify --module=PATH --module-dir=DIR [--go=GO]"

// runAdapterRegenerate exits 0 when the dry run or apply succeeds, 1 when the
// regeneration needs a person (an anchor that no longer matches once, a
// rewritten file gone upstream, a staged output that fails, a changed
// checkout), 2 for invalid input including an approval that does not match,
// and 3 for infrastructure failures.
func runAdapterRegenerate(arguments []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("gomadtool adapter-regenerate", flag.ContinueOnError)
	flags.SetOutput(stderr)
	root := flags.String("root", ".", "Gomad v3 module root")
	module := flags.String("module", "", "adapted module path")
	version := flags.String("version", "", "exact candidate module version")
	approval := flags.String("approve-review", "", "review digest a dry run printed; applies the regeneration")
	goCommand := flags.String("go", os.Getenv("GOMAD3_BOOTSTRAP_GO"), "pinned Go release command")
	jsonOutput := flags.Bool("json", false, "write the review as JSON")
	recoverOnly := flags.Bool("recover", false, "complete an interrupted publication and exit")
	verify := flags.Bool("verify", false, "check the compiled adapter against --module-dir")
	moduleDirectory := flags.String("module-dir", "", "module source directory for --verify")
	if err := flags.Parse(arguments); err != nil || flags.NArg() != 0 {
		fmt.Fprintln(stderr, adapterRegenerateUsage)
		return 2
	}
	ctx := context.Background()
	absoluteRoot, err := filepath.Abs(*root)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	if *recoverOnly {
		if err := adapterregen.Recover(absoluteRoot); err != nil {
			fmt.Fprintln(stderr, err)
			return adapterRegenerateStatus(err)
		}
		return 0
	}
	if *goCommand == "" {
		*goCommand = "go"
	}
	*goCommand, err = exec.LookPath(*goCommand)
	if err == nil {
		*goCommand, err = filepath.Abs(*goCommand)
	}
	if err != nil {
		fmt.Fprintf(stderr, "adapter regeneration requires the pinned go command; set GOMAD3_BOOTSTRAP_GO or pass --go: %v\n", err)
		return 3
	}
	if *verify {
		if *module == "" || *moduleDirectory == "" {
			fmt.Fprintln(stderr, adapterRegenerateUsage)
			return 2
		}
		if err := adapterregen.Verify(ctx, *module, *moduleDirectory, *goCommand); err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		return 0
	}
	if *module == "" || *version == "" {
		fmt.Fprintln(stderr, adapterRegenerateUsage)
		fmt.Fprintf(stderr, "regenerable adapters: %s\n", strings.Join(adapterregen.Regenerable(), ", "))
		return 2
	}
	result, err := adapterregen.Run(ctx, adapterregen.Spec{
		Root: absoluteRoot, Module: *module, Version: *version, GoCommand: *goCommand, Environment: os.Environ(), Approval: *approval,
	})
	if err != nil {
		fmt.Fprintln(stderr, err)
		return adapterRegenerateStatus(err)
	}
	if *jsonOutput {
		encoder := json.NewEncoder(stdout)
		encoder.SetIndent("", "  ")
		err = encoder.Encode(result)
	} else {
		err = adapterregen.Render(stdout, result)
	}
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 3
	}
	return 0
}

func adapterRegenerateStatus(err error) int {
	var input *adapterregen.InputError
	var blocked *adapterregen.BlockedError
	switch {
	case errors.As(err, &input):
		return 2
	case errors.As(err, &blocked):
		return 1
	default:
		return 3
	}
}
