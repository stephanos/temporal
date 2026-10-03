package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"path/filepath"

	"go.temporal.io/server/tools/gomad3/upgrade"
)

func runPinImpact(arguments []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("gomadtool pin-impact", flag.ContinueOnError)
	flags.SetOutput(stderr)
	root := flags.String("root", ".", "Gomad v3 module root")
	candidate := flags.String("candidate", "", "candidate go.mod, with adjacent go.sum")
	baseline := flags.String("baseline", "", "baseline go.mod, default repository root module")
	format := flags.String("format", "human", "human or json")
	if err := flags.Parse(arguments); err != nil {
		return 2
	}
	if flags.NArg() != 0 || (*format != "human" && *format != "json") {
		fmt.Fprintln(stderr, "invalid pin-impact arguments")
		return 2
	}
	if *baseline == "" {
		*baseline = filepath.Join(*root, "../..", "go.mod")
	}
	if *candidate == "" {
		*candidate = *baseline
	}
	report, err := upgrade.ReadPinImpact(*root, *candidate, *baseline)
	if err != nil {
		fmt.Fprintln(stderr, err)
		var invalid *upgrade.InvalidPinImpactInput
		if errors.As(err, &invalid) {
			return 2
		}
		return 3
	}
	if *format == "json" {
		contents, err := report.CanonicalJSON()
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 3
		}
		if _, err = fmt.Fprintln(stdout, string(contents)); err != nil {
			fmt.Fprintln(stderr, err)
			return 3
		}
	} else {
		for _, pin := range report.Pins {
			if _, err := fmt.Fprintf(stdout, "%s %s: %s (%s)\n", pin.Class, pin.ID, pin.Status, pin.Reason); err != nil {
				fmt.Fprintln(stderr, err)
				return 3
			}
		}
	}
	if report.Invalidated() {
		return 1
	}
	return 0
}
