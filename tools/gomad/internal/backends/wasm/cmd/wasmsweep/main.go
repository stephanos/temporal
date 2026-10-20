// Command wasmsweep is the fn-221 functional-suite triage sweep. It runs every
// top-level test function of a Temporal checkout's ./tests packages under the
// cooperative WASM profile through a prebuilt gomad binary, one `gomad explore
// --json` campaign per function, and classifies each outcome. Inputs are
// flags only; an inherited GOMAD_* variable and an unbounded budget are
// refused. The output is an observation on one candidate, never a support or
// qualification claim.
//
//	wasmsweep --checkout /abs/temporal --gomad /abs/bin/gomad \
//	    --compiler /abs/go/bin/go --helper /abs/gomad-wasmhost \
//	    --runtime-root /abs/temporal/tools/gomad/internal/model/runtimesource \
//	    --config-source throwaway --packages ./tests/... \
//	    --per-test-timeout 10m --prepare-timeout 30m --overall-budget 24h \
//	    --retain-bytes 5GiB --out /abs/out --work /abs/work --artifacts /abs/artifacts
//	wasmsweep --verify /abs/out/sweep.json
//
// Rerunning with the same --out resumes: functions already classified on the
// same candidate identity are skipped. Exit status: 0 success, 1 failure or
// verification findings, 2 refusal.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"

	"go.temporal.io/server/tools/gomad/internal/backends/wasm/internal/sweep"
)

const (
	exitFailed  = 1
	exitRefused = 2
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	code := run(ctx, os.Args[1:], os.Environ(), os.Stdout, os.Stderr, func(ctx context.Context, sweeper *sweep.Sweeper) ([]string, error) {
		return sweeper.Run(ctx)
	})
	stop()
	os.Exit(code)
}

type runner func(context.Context, *sweep.Sweeper) ([]string, error)

func run(ctx context.Context, args, environ []string, stdout, stderr io.Writer, start runner) int {
	flags := flag.NewFlagSet("wasmsweep", flag.ContinueOnError)
	flags.SetOutput(stderr)
	verify := flags.String("verify", "", "verify a sweep sidecar JSON and exit")
	var options sweep.Options
	flags.StringVar(&options.Checkout, "checkout", "", "absolute Temporal checkout (required)")
	flags.StringVar(&options.Gomad, "gomad", "", "prebuilt gomad binary (required)")
	flags.StringVar(&options.Compiler, "compiler", "", "pinned stock Go 1.27.1 go binary (required)")
	flags.StringVar(&options.Helper, "helper", "", "pinned gomad-wasmhost helper (required)")
	flags.StringVar(&options.RuntimeRoot, "runtime-root", "", "candidate runtimesource directory (required)")
	configSource := flags.String("config-source", "", "throwaway or temporal-install (required)")
	packages := flags.String("packages", "", "comma-separated ./tests package patterns (required)")
	flags.IntVar(&options.MaxTests, "max-tests", 0, "per-package cap in sorted order; 0 runs every function")
	flags.Uint64Var(&options.Seed, "seed", 0, "the one seed each function runs with")
	flags.DurationVar(&options.PerTestTimeout, "per-test-timeout", 0, "explore --execution-timeout (required)")
	flags.DurationVar(&options.PrepareTimeout, "prepare-timeout", 0, "guest build and engine compile allowance added to --overall-timeout (required)")
	flags.DurationVar(&options.OverallBudget, "overall-budget", 0, "wall budget of this invocation (required)")
	retainBytes := flags.String("retain-bytes", "", "exemplar artifact byte budget, e.g. 5GiB or 0 (required)")
	flags.StringVar(&options.Out, "out", "", "journal and sidecar directory (required)")
	flags.StringVar(&options.Work, "work", "", "installation, cache and per-test scratch directory outside the checkout (required)")
	flags.StringVar(&options.Artifacts, "artifacts", "", "retained exemplar artifact root outside the checkout (required)")
	if err := flags.Parse(args); err != nil {
		return exitRefused
	}
	if flags.NArg() != 0 {
		_, _ = fmt.Fprintf(stderr, "wasmsweep: unexpected arguments %q\n", flags.Args())
		return exitRefused
	}
	if *verify != "" {
		set := 0
		flags.Visit(func(*flag.Flag) { set++ })
		if set != 1 {
			_, _ = fmt.Fprintln(stderr, "wasmsweep: --verify takes no other flags")
			return exitRefused
		}
		findings, err := sweep.VerifyFile(*verify)
		if err != nil {
			_, _ = fmt.Fprintf(stderr, "wasmsweep: verify: %v\n", err)
			return exitFailed
		}
		return report(stdout, *verify, findings)
	}
	retainSet := false
	flags.Visit(func(defined *flag.Flag) { retainSet = retainSet || defined.Name == "retain-bytes" })
	if !retainSet {
		_, _ = fmt.Fprintln(stderr, "wasmsweep: --retain-bytes is required")
		return exitRefused
	}
	var err error
	if options.RetainBytes, err = parseBytes(*retainBytes); err != nil {
		_, _ = fmt.Fprintf(stderr, "wasmsweep: --retain-bytes: %v\n", err)
		return exitRefused
	}
	options.ConfigSource = sweep.ConfigSource(*configSource)
	for _, pattern := range strings.Split(*packages, ",") {
		if pattern = strings.TrimSpace(pattern); pattern != "" {
			options.Packages = append(options.Packages, pattern)
		}
	}
	options.Environ = environ
	if err := options.Validate(); err != nil {
		_, _ = fmt.Fprintf(stderr, "wasmsweep: %v\n", err)
		return exitRefused
	}
	executable, err := os.Executable()
	if err == nil {
		executable, err = sweep.FileSHA256(executable)
	}
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "wasmsweep: identify the harness build: %v\n", err)
		return exitFailed
	}
	sweeper := &sweep.Sweeper{Options: options, Log: stderr, Command: append([]string{"wasmsweep"}, args...), HarnessSHA256: executable}
	findings, err := start(ctx, sweeper)
	if errors.Is(err, sweep.ErrRefused) {
		_, _ = fmt.Fprintf(stderr, "wasmsweep: %v\n", err)
		return exitRefused
	}
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "wasmsweep: %v\n", err)
		return exitFailed
	}
	return report(stdout, sweep.SidecarPath(options.Out), findings)
}

func report(stdout io.Writer, path string, findings []string) int {
	for _, finding := range findings {
		_, _ = fmt.Fprintf(stdout, "wasmsweep: finding: %s\n", finding)
	}
	if len(findings) != 0 {
		_, _ = fmt.Fprintf(stdout, "wasmsweep: %s: %d findings\n", path, len(findings))
		return exitFailed
	}
	_, _ = fmt.Fprintf(stdout, "wasmsweep: %s: verified\n", path)
	return 0
}

func parseBytes(value string) (uint64, error) {
	multiplier := uint64(1)
	for _, unit := range []struct {
		suffix string
		scale  uint64
	}{{"GiB", 1 << 30}, {"MiB", 1 << 20}, {"KiB", 1 << 10}} {
		if trimmed, ok := strings.CutSuffix(value, unit.suffix); ok {
			value, multiplier = trimmed, unit.scale
			break
		}
	}
	number, err := strconv.ParseUint(value, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("want a whole byte count with optional KiB, MiB or GiB")
	}
	if number != 0 && number > ^uint64(0)/multiplier {
		return 0, fmt.Errorf("overflows uint64")
	}
	return number * multiplier, nil
}
