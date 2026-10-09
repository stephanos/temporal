package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"strconv"
	"strings"

	"go.temporal.io/server/tools/gomad3/qualification/soak"
)

// runSoak runs the determinism soak. It exits 0 for a pass or a divergence on
// an informational platform, 1 for a divergence or target failure, 2 for
// invalid input, and 3 when an overflow or infrastructure failure left the
// soak unable to conclude.
func runSoak(arguments []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("gomadtool soak", flag.ContinueOnError)
	flags.SetOutput(stderr)
	spec := soak.Spec{}
	flags.StringVar(&spec.ManifestPath, "manifest", "", "determinism soak manifest")
	flags.StringVar(&spec.GomadPath, "gomad", "", "gomad executable")
	flags.StringVar(&spec.WorkRoot, "work", "", "directory for each batch's Campaigns, pruned after the batch")
	flags.StringVar(&spec.LedgerDir, "ledger", "", "cumulative ledger directory restored from the previous run")
	flags.StringVar(&spec.OutputDir, "output", "", "report, summary, batch reports, and divergence evidence directory")
	flags.StringVar(&spec.RunID, "run-id", "", "identity of this run in the ledger")
	workload := flags.String("workload", "", "comma-separated workloads to run (default: the whole selection)")
	seeds := flags.String("seed", "", "comma-separated seeds to run (default: the manifest seeds)")
	flags.Uint64Var(&spec.Batches, "batches", 0, "batches per workload and seed (default: the manifest)")
	flags.DurationVar(&spec.Budget, "budget", 0, "run budget (default: the manifest)")
	flags.BoolVar(&spec.KeepBatches, "keep-batches", false, "keep each batch's Campaigns")
	if err := flags.Parse(arguments); err != nil {
		return 2
	}
	if flags.NArg() != 0 {
		if _, err := fmt.Fprintln(stderr, "usage: gomadtool soak --manifest=FILE --gomad=FILE --work=DIR --ledger=DIR --output=DIR [flags]"); err != nil {
			return 2
		}
		return 2
	}
	if *workload != "" {
		spec.Workloads = strings.Split(*workload, ",")
	}
	if *seeds != "" {
		for _, value := range strings.Split(*seeds, ",") {
			seed, err := strconv.ParseUint(value, 10, 64)
			if err != nil {
				if _, err := fmt.Fprintf(stderr, "invalid --seed %q\n", value); err != nil {
					return 2
				}
				return 2
			}
			spec.Seeds = append(spec.Seeds, seed)
		}
	}
	report, err := soak.Run(context.Background(), spec)
	if err != nil {
		if _, writeErr := fmt.Fprintln(stderr, err); writeErr != nil {
			if report.Schema == "" {
				return 2
			}
			return 3
		}
		if report.Schema == "" {
			return 2
		}
		return 3
	}
	if _, err := io.WriteString(stdout, soak.Summary(report)); err != nil {
		if _, writeErr := fmt.Fprintln(stderr, err); writeErr != nil {
			return 3
		}
		return 3
	}
	return soak.ExitStatus(report)
}
