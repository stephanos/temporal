package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"go.temporal.io/server/tools/canary/authority"
	"go.temporal.io/server/tools/canary/controller"
	"go.temporal.io/server/tools/umpire/publish"
)

// statusUsage is a command line the command refuses before reading anything.
const statusUsage = "usage"

// modelRoot is the model package, which never receives the canary's output.
const modelRoot = "model"

const usage = "usage: umpire-canary run|reconcile --output <dir> --recovery <file>"

// The two closed modes.
const (
	modeRun       = "run"
	modeReconcile = "reconcile"
)

// options are the only things a caller names: the mode, where the artifact goes and where the
// job's recovery record lives.
type options struct {
	Mode     string
	Output   string
	Recovery string
}

// usageSummary is the one document a refused command line writes.
type usageSummary struct {
	Status string `json:"status"`
	Detail string `json:"detail"`
}

// Main is the whole command: parse the closed mode and its two flags, run the mode, and write its
// one document on stdout. It returns the exit code.
func Main(arguments []string, stdout, stderr io.Writer, lookup authority.Lookup, seams controller.Seams) int {
	parsed, err := parse(arguments, stderr)
	if err != nil {
		return report(stdout, stderr, "umpire-canary", statusUsage, usageSummary{Status: statusUsage, Detail: err.Error()}, controller.ExitFailed)
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if parsed.Mode == modeReconcile {
		// The report goes to stdout only: the workflow keeps it beside the artifact.
		reconciled, code := controller.Reconcile(ctx, controller.Reconciliation{
			Seams: seams, Lookup: lookup, Recovery: parsed.Recovery, Progress: stderr, Now: time.Now(),
		})
		return report(stdout, stderr, "umpire-canary reconcile", reconciled.Status, reconciled, code)
	}
	summary, code := controller.Invoke(ctx, controller.Invocation{
		Seams: seams, Lookup: lookup, Output: parsed.Output, Recovery: parsed.Recovery,
		Progress: stderr, Started: time.Now(),
	})
	return report(stdout, stderr, "umpire-canary run", summary.Status, summary, code)
}

func parse(arguments []string, stderr io.Writer) (options, error) {
	if len(arguments) == 0 || (arguments[0] != modeRun && arguments[0] != modeReconcile) {
		_, _ = fmt.Fprintln(stderr, usage)
		return options{}, errors.New("the mode is run or reconcile")
	}
	parsed := options{Mode: arguments[0]}
	flags := flag.NewFlagSet("umpire-canary "+parsed.Mode, flag.ContinueOnError)
	flags.SetOutput(stderr)
	flags.StringVar(&parsed.Output, "output", "", "an existing directory outside the model for receipts and provenance")
	flags.StringVar(&parsed.Recovery, "recovery", "", "the job's recovery record: run creates it, reconcile reads it")
	if err := flags.Parse(arguments[1:]); err != nil {
		return options{}, err
	}
	if flags.NArg() != 0 {
		return options{}, fmt.Errorf("umpire-canary %s accepts no positional arguments", parsed.Mode)
	}
	if parsed.Output == "" || parsed.Recovery == "" {
		return options{}, errors.New(usage)
	}
	output, err := publish.Resolve(parsed.Output)
	if err != nil {
		return options{}, fmt.Errorf("--output: %w", err)
	}
	if info, err := os.Stat(output); err != nil || !info.IsDir() {
		return options{}, errors.New("--output: the directory does not exist")
	}
	model, err := publish.Resolve(modelRoot)
	if err != nil {
		return options{}, fmt.Errorf("the model root: %w", err)
	}
	recovery, err := publish.Resolve(parsed.Recovery)
	if err != nil {
		return options{}, fmt.Errorf("--recovery: %w", err)
	}
	switch {
	case publish.Within(model, output):
		return options{}, errors.New("--output is under the model, which never receives the canary's output")
	case publish.Within(output, recovery):
		return options{}, errors.New("--recovery is under --output, which is uploaded; the record lives only for its job")
	}
	if info, err := os.Stat(filepath.Dir(recovery)); err != nil || !info.IsDir() {
		return options{}, errors.New("--recovery: its directory does not exist")
	}
	parsed.Output, parsed.Recovery = output, recovery
	return parsed, nil
}

// report writes the document on stdout and one line on stderr, and returns the code. A document
// stdout cannot take goes to stderr, and the exit is 3.
func report(stdout, stderr io.Writer, command, status string, document any, code int) int {
	encoded, err := json.Marshal(document)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "%s: encode the summary: %s\n", command, err)
		return controller.ExitFailed
	}
	if _, err := stdout.Write(append(encoded, '\n')); err != nil {
		_, _ = fmt.Fprintf(stderr, "%s\n", encoded)
		return controller.ExitFailed
	}
	_, _ = fmt.Fprintf(stderr, "%s: %s\n", command, status)
	return code
}
