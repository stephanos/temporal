package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"time"

	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/testpilot"
	"go.temporal.io/server/tools/umpire/binding"
)

// Exit codes. 3 is deliberately separate from 2 so a caller can tell an unreachable server or a
// Case that could not be prepared from a Run that really was inconclusive.
const (
	exitSatisfied    = 0
	exitViolated     = 1
	exitInconclusive = 2
	exitFailed       = 3
)

const defaultTimeout = 5 * time.Minute

// config is what the caller names. Nothing here ever enters the Case: addresses and credentials
// stay outside the bytes, and no flag widens a declared Limit.
type config struct {
	CasePath      string
	GRPCAddress   string
	HTTPAddress   string
	Namespace     string
	TaskQueue     string
	NexusEndpoint string
	// HandlerTaskQueue is the queue the Case's Nexus handler polls when the Case binds one apart
	// from the caller's; empty derives `<task-queue>-handler` for a Case that binds one.
	HandlerTaskQueue string
	Create           bool
	Timeout          time.Duration
}

// session is one bound Case: how to run it once, and how to release everything the binding opened.
type session struct {
	// run executes the prepared Case against the Driver the binding opened.
	run func(ctx context.Context) (*testpilotspb.Run, *testpilotspb.Verdict, error)
	// release is best-effort teardown, run whatever the Run did. It names every resource it could
	// not remove.
	release func(ctx context.Context) error
}

// opener binds one Case. The real one dials the server; a test supplies its own.
type opener func(ctx context.Context, configuration config, source *testpilotspb.Case) (*session, error)

// Run is the whole command. It never panics on a caller mistake: a rejected flag, an unreadable
// fixture, a Case whose Profile cannot be derived, and an unreachable server all exit 3 with one
// line on stderr naming the cause.
func Run(arguments []string, stdout, stderr io.Writer, open opener) int {
	configuration, err := parseConfig(arguments, stderr)
	if err != nil {
		return exitFailed
	}

	source, err := readCase(configuration.CasePath)
	if err != nil {
		writeLine(stderr, "%s", err)
		return exitFailed
	}

	ctx, cancel := interruptible(context.Background(), configuration.Timeout)
	defer cancel()

	bound, err := open(ctx, configuration, source)
	if err != nil {
		writeLine(stderr, "%s", describeFailure(err))
		return exitFailed
	}
	defer releaseSession(bound, stderr)

	run, verdict, err := bound.run(ctx)
	if err != nil {
		writeLine(stderr, "run Case %q: %v", source.GetCaseId(), err)
		return exitFailed
	}

	report(stdout, run, verdict)
	return exitCode(verdict)
}

// interruptible bounds the Run by the caller's timeout and cancels it on SIGINT. Teardown runs on
// its own context afterwards, so an interrupted Run still removes what it created.
func interruptible(parent context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	notified, stopNotify := signal.NotifyContext(parent, os.Interrupt)
	ctx, cancelTimeout := context.WithTimeout(notified, timeout)
	return ctx, func() {
		cancelTimeout()
		stopNotify()
	}
}

// writeLine reports one line. A report the caller cannot receive is not a failure worth changing
// the exit code for, so the write error is deliberately dropped.
func writeLine(destination io.Writer, format string, arguments ...any) {
	_, _ = fmt.Fprintf(destination, format+"\n", arguments...)
}

func parseConfig(arguments []string, stderr io.Writer) (config, error) {
	var configuration config
	flags := flag.NewFlagSet("umpire-run", flag.ContinueOnError)
	flags.SetOutput(stderr)
	flags.StringVar(&configuration.CasePath, "case", "", "path to a checked-in Case fixture")
	flags.StringVar(&configuration.GRPCAddress, "grpc", "", "frontend gRPC address")
	flags.StringVar(&configuration.HTTPAddress, "http", "", "frontend HTTP address")
	flags.StringVar(&configuration.Namespace, "namespace", "", "namespace the Case binds to")
	flags.StringVar(&configuration.TaskQueue, "task-queue", "", "task queue the Case binds to")
	flags.StringVar(&configuration.NexusEndpoint, "nexus-endpoint", "", "Nexus endpoint the Case binds to")
	flags.StringVar(&configuration.HandlerTaskQueue, "handler-task-queue", "", "task queue the Case's Nexus handler polls when it binds one of its own (default <task-queue>-handler)")
	flags.BoolVar(&configuration.Create, "create", false, "create the named resources and delete them on exit")
	flags.DurationVar(&configuration.Timeout, "timeout", defaultTimeout, "bound on the whole Run")
	if err := flags.Parse(arguments); err != nil {
		return config{}, err
	}
	if flags.NArg() != 0 {
		writeLine(stderr, "umpire-run accepts no positional arguments")
		return config{}, errors.New("unexpected positional arguments")
	}
	for _, required := range []struct{ name, value string }{
		{"--case", configuration.CasePath},
		{"--grpc", configuration.GRPCAddress},
		{"--http", configuration.HTTPAddress},
		{"--namespace", configuration.Namespace},
		{"--task-queue", configuration.TaskQueue},
	} {
		if required.value == "" {
			writeLine(stderr, "%s is required", required.name)
			return config{}, fmt.Errorf("missing %s", required.name)
		}
	}
	if configuration.Timeout <= 0 {
		writeLine(stderr, "--timeout must be positive")
		return config{}, errors.New("non-positive timeout")
	}
	return configuration, nil
}

func readCase(path string) (*testpilotspb.Case, error) {
	encoded, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read Case fixture: %w", err)
	}
	source, err := testpilot.DecodeCaseProtoJSON(encoded)
	if err != nil {
		return nil, fmt.Errorf("decode Case fixture %q: %w", path, err)
	}
	return source, nil
}

// describeFailure names a static admission rejection by its category, because "typed fixture" and
// "unreachable server" are different problems and the exit code is the same.
func describeFailure(err error) string {
	var rejection *testpilot.PreparationError
	if errors.As(err, &rejection) {
		return fmt.Sprintf("prepare Case: %s at %s: %s", rejection.Category, rejection.Path, rejection.Detail)
	}
	return err.Error()
}

func releaseSession(bound *session, stderr io.Writer) {
	if bound == nil || bound.release == nil {
		return
	}
	// Teardown runs on its own context: an interrupted or timed-out Run still releases what it
	// created.
	if err := bound.release(context.Background()); err != nil {
		for _, failure := range flatten(err) {
			writeLine(stderr, "%s", failure)
		}
	}
}

// flatten reports one line per leaked resource rather than one joined blob.
func flatten(err error) []string {
	var joined interface{ Unwrap() []error }
	if errors.As(err, &joined) {
		var lines []string
		for _, nested := range joined.Unwrap() {
			lines = append(lines, flatten(nested)...)
		}
		return lines
	}
	return []string{err.Error()}
}

func report(stdout io.Writer, run *testpilotspb.Run, verdict *testpilotspb.Verdict) {
	writeLine(stdout, "run %s", run.GetDisposition())
	writeLine(stdout, "cleanup %s", run.GetCleanup().GetStatus())
	writeLine(stdout, "verdict %s", verdict.GetStatus())
	for _, rule := range verdict.GetRules() {
		writeLine(stdout, "rule %s %s %s", rule.GetRuleId(), rule.GetStatus(), rule.GetTerminalStateId())
	}
}

func exitCode(verdict *testpilotspb.Verdict) int {
	switch verdict.GetStatus() {
	case testpilotspb.VERDICT_STATUS_SATISFIED:
		return exitSatisfied
	case testpilotspb.VERDICT_STATUS_VIOLATED:
		return exitViolated
	default:
		return exitInconclusive
	}
}

// openSession is the real binding: the campaign-scoped part (dial the frontend, create the named
// resources when asked, build the catalog) opened for this one Case, then the candidate-scoped part
// (derive the Profile the Case implies, prepare its unchanged bytes, open one composite Driver with
// its own SDK worker). Both live in `tools/umpire/binding`, which a campaign shares; the CLI's
// behavior and exit codes are unchanged.
func openSession(ctx context.Context, configuration config, source *testpilotspb.Case) (*session, error) {
	deployment := binding.Deployment{
		GRPCAddress:      configuration.GRPCAddress,
		HTTPAddress:      configuration.HTTPAddress,
		Namespace:        configuration.Namespace,
		TaskQueue:        configuration.TaskQueue,
		NexusEndpoint:    configuration.NexusEndpoint,
		HandlerTaskQueue: configuration.HandlerTaskQueue,
		Create:           configuration.Create,
	}
	campaign, err := binding.Open(ctx, deployment, binding.HandlerQueueFor(deployment, source.GetProgram()))
	if err != nil {
		return nil, err
	}
	bound, err := campaign.Bind(ctx, "umpire-run."+configuration.Namespace, source)
	if err != nil {
		// A binding that failed rolls back on a context the failure cannot have cancelled.
		return nil, errors.Join(err, campaign.Close(context.WithoutCancel(ctx)))
	}
	return &session{
		run: bound.Run,
		release: func(ctx context.Context) error {
			return errors.Join(bound.Release(ctx), campaign.Close(ctx))
		},
	}, nil
}
