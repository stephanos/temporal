package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"time"

	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/testpilot"
	"go.temporal.io/server/tools/umpire/binding"
	"go.temporal.io/server/tools/umpire/internal/cli"
	"go.temporal.io/server/tools/umpire/replay"
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
	CasePath string
	// Deployment is what the Case binds to; its handler task queue, when empty, derives
	// `<task-queue>-handler` for a Case that binds one.
	Deployment binding.Deployment
	// RecordPath, when named, receives the closed Run with the identity it was prepared under,
	// the recorded Run a replay reads; empty records nothing.
	RecordPath string
	Timeout    time.Duration
}

// session is one bound Case: how to run it once, and how to release everything the binding opened.
type session struct {
	// run executes the prepared Case against the Driver the binding opened.
	run func(ctx context.Context) (*testpilotspb.Run, *testpilotspb.Verdict, error)
	// identity is the Profile identity the Case was prepared under, recorded beside its Run.
	identity testpilot.DriverIdentity
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
		cli.WriteLine(stderr, "%s", err)
		return exitFailed
	}

	ctx, cancel := cli.Interruptible(context.Background(), configuration.Timeout)
	defer cancel()

	bound, err := open(ctx, configuration, source)
	if err != nil {
		cli.WriteLine(stderr, "%s", describeFailure(err))
		return exitFailed
	}
	defer releaseSession(bound, stderr)

	run, verdict, err := bound.run(ctx)
	if err != nil {
		cli.WriteLine(stderr, "run Case %q: %v", source.GetCaseId(), err)
		return exitFailed
	}

	report(stdout, run, verdict)
	if configuration.RecordPath != "" {
		// The Run is reported whatever happens to its record; a record that could not be written
		// is the command's failure, said after the report.
		if err := replay.WriteRecordedRun(configuration.RecordPath, bound.identity, run); err != nil {
			cli.WriteLine(stderr, "%s", err)
			return exitFailed
		}
	}
	return exitCode(verdict)
}

func parseConfig(arguments []string, stderr io.Writer) (config, error) {
	var configuration config
	flags := flag.NewFlagSet("umpire-run", flag.ContinueOnError)
	flags.SetOutput(stderr)
	flags.StringVar(&configuration.CasePath, "case", "", "path to a checked-in Case fixture")
	binding.RegisterFlags(flags, &configuration.Deployment, "the Case")
	flags.DurationVar(&configuration.Timeout, "timeout", defaultTimeout, "bound on the whole Run")
	flags.StringVar(&configuration.RecordPath, "record", "", "write the closed Run with the identity it was prepared under to this file, which must not exist")
	if err := flags.Parse(arguments); err != nil {
		return config{}, err
	}
	if flags.NArg() != 0 {
		cli.WriteLine(stderr, "umpire-run accepts no positional arguments")
		return config{}, errors.New("unexpected positional arguments")
	}
	if configuration.CasePath == "" {
		cli.WriteLine(stderr, "--case is required")
		return config{}, errors.New("missing --case")
	}
	if missing := binding.Missing(configuration.Deployment); len(missing) > 0 {
		cli.WriteLine(stderr, "%s is required", missing[0])
		return config{}, fmt.Errorf("missing %s", missing[0])
	}
	if configuration.Timeout <= 0 {
		cli.WriteLine(stderr, "--timeout must be positive")
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
		for _, failure := range cli.Flatten(err) {
			cli.WriteLine(stderr, "%s", failure)
		}
	}
}

// flatten reports one line per leaked resource rather than one joined blob.
func report(stdout io.Writer, run *testpilotspb.Run, verdict *testpilotspb.Verdict) {
	cli.WriteLine(stdout, "run %s", run.GetDisposition())
	cli.WriteLine(stdout, "cleanup %s", run.GetCleanup().GetStatus())
	cli.WriteLine(stdout, "verdict %s", verdict.GetStatus())
	for _, rule := range verdict.GetRules() {
		cli.WriteLine(stdout, "rule %s %s %s", rule.GetRuleId(), rule.GetStatus(), rule.GetTerminalStateId())
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
	deployment := configuration.Deployment
	campaign, err := binding.Open(ctx, deployment, binding.HandlerQueueFor(deployment, source.GetProgram()))
	if err != nil {
		return nil, err
	}
	bound, err := campaign.Bind(ctx, "umpire-run."+deployment.Namespace, source)
	if err != nil {
		// A binding that failed rolls back on a context the failure cannot have cancelled.
		return nil, errors.Join(err, campaign.Close(context.WithoutCancel(ctx)))
	}
	return &session{
		run:      bound.Run,
		identity: bound.Identity(),
		release: func(ctx context.Context) error {
			return errors.Join(bound.Release(ctx), campaign.Close(ctx))
		},
	}, nil
}
