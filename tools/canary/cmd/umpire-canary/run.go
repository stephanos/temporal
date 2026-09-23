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

const usage = "usage: umpire-canary run --output <dir> --recovery <file>"

// options are the only things a caller names: where the artifact goes and where the job's
// recovery record lives.
type options struct {
	Output   string
	Recovery string
}

// Main is the whole command: parse the closed mode and its two flags, invoke, and write the one
// summary on stdout. It returns the exit code.
func Main(arguments []string, stdout, stderr io.Writer, lookup authority.Lookup, seams controller.Seams) int {
	parsed, err := parse(arguments, stderr)
	if err != nil {
		return report(stdout, stderr, controller.Summary{Status: statusUsage, Detail: err.Error(), Iterations: []controller.SummaryIteration{}}, controller.ExitFailed)
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	summary, code := controller.Invoke(ctx, controller.Invocation{
		Seams: seams, Lookup: lookup, Output: parsed.Output, Recovery: parsed.Recovery,
		Progress: stderr, Started: time.Now(),
	})
	return report(stdout, stderr, summary, code)
}

func parse(arguments []string, stderr io.Writer) (options, error) {
	if len(arguments) == 0 || arguments[0] != "run" {
		_, _ = fmt.Fprintln(stderr, usage)
		return options{}, errors.New("the mode is run")
	}
	var parsed options
	flags := flag.NewFlagSet("umpire-canary run", flag.ContinueOnError)
	flags.SetOutput(stderr)
	flags.StringVar(&parsed.Output, "output", "", "an existing directory outside the model for receipts and provenance")
	flags.StringVar(&parsed.Recovery, "recovery", "", "the job's recovery record, created by this run")
	if err := flags.Parse(arguments[1:]); err != nil {
		return options{}, err
	}
	if flags.NArg() != 0 {
		return options{}, errors.New("umpire-canary run accepts no positional arguments")
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
	return options{Output: output, Recovery: recovery}, nil
}

// report writes the summary on stdout and one line on stderr, and returns the code. A summary
// stdout cannot take goes to stderr, and the exit is 3.
func report(stdout, stderr io.Writer, summary controller.Summary, code int) int {
	encoded, err := json.Marshal(summary)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "umpire-canary: encode the summary: %s\n", err)
		return controller.ExitFailed
	}
	if _, err := stdout.Write(append(encoded, '\n')); err != nil {
		_, _ = fmt.Fprintf(stderr, "%s\n", encoded)
		return controller.ExitFailed
	}
	_, _ = fmt.Fprintf(stderr, "umpire-canary run: %s\n", summary.Status)
	return code
}
