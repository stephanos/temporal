package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"strings"

	"go.temporal.io/server/tools/gomad3/choice"
)

func runDiagnosticDiff(arguments []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("gomadtool diagnostic-diff", flag.ContinueOnError)
	flags.SetOutput(stderr)
	jsonOutput := flags.Bool("json", false, "emit the first divergent ordinal and fields as JSON")
	if err := flags.Parse(arguments); err != nil {
		return 2
	}
	if flags.NArg() != 2 {
		fmt.Fprintln(stderr, "usage: gomadtool diagnostic-diff [--json] EXPECTED_TRACE ACTUAL_TRACE")
		return 2
	}
	expected, err := choice.ReadDiagnosticTrace(flags.Arg(0))
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	actual, err := choice.ReadDiagnosticTrace(flags.Arg(1))
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	difference, err := choice.DiffDiagnostics(expected.Bytes, actual.Bytes)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	if *jsonOutput {
		err = json.NewEncoder(stdout).Encode(struct {
			Equal      bool                         `json:"equal"`
			Divergence *choice.DiagnosticDivergence `json:"divergence,omitempty"`
		}{Equal: difference == nil, Divergence: difference})
	} else if difference == nil {
		_, err = fmt.Fprintln(stdout, "diagnostic traces match")
	} else {
		_, err = fmt.Fprintf(stdout, "first-divergent-ordinal=%d fields=%s\n", difference.Ordinal, strings.Join(difference.Fields, ","))
	}
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 3
	}
	if difference != nil {
		return 1
	}
	return 0
}
