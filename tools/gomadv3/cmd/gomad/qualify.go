package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"
	"time"

	"go.temporal.io/server/tools/gomadv3/internal/qualify"
	"go.temporal.io/server/tools/gomadv3/internal/record"
	"go.temporal.io/server/tools/gomadv3/internal/replay"
	"go.temporal.io/server/tools/gomadv3/internal/runner"
	"go.temporal.io/server/tools/gomadv3/internal/target"
)

const maximumQualificationRepeats = 32

type qualifyDependencies struct {
	identity         func(string) (string, string, string, error)
	workingDirectory func() (string, error)
	run              func(context.Context, runner.Config) (runner.Summary, error)
	replay           func(context.Context, replay.Config) (replay.Result, error)
	write            func(string, qualify.Report) (string, error)
}

type qualifyReporter struct {
	json   bool
	stdout io.Writer
	stderr io.Writer
}

func runQualify(arguments []string, stdout, stderr io.Writer) int {
	return runQualifyWith(arguments, stdout, stderr, qualifyDependencies{
		identity: localIdentity, workingDirectory: os.Getwd, run: runner.Run, replay: replay.Replay, write: qualify.Write,
	})
}

func runQualifyWith(arguments []string, stdout, stderr io.Writer, dependencies qualifyDependencies) int {
	flags := flag.NewFlagSet("gomad qualify", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	seed := flags.Uint64("seed", 1, "schedule seed")
	repeat := flags.Uint64("repeat", 2, "fresh same-seed repetitions")
	runTimeout := flags.Duration("run-timeout", 30*time.Second, "per-run host deadline")
	overallTimeout := flags.Duration("overall-timeout", 10*time.Minute, "complete qualification host deadline")
	terminateGrace := flags.Duration("terminate-grace", 2*time.Second, "termination grace inside deadlines")
	artifacts := flags.String("artifacts", ".gomad/artifacts", "artifact and qualification report root")
	toolchainRoot := flags.String("toolchain-root", "", "absolute pinned toolchain root")
	jsonOutput := flags.Bool("json", false, "emit stable JSON events")
	choices := flags.Bool("choices", false, "record bounded runtime choices")
	replaySuccesses := flags.Bool("replay-successes", false, "retain and replay every successful repetition")
	successLimit := flags.Uint64("success-limit", 0, "maximum retained successful runs per repetition")
	outputLimit := byteSize(8 << 20)
	worldLimit := byteSize(64 << 20)
	successBytes := byteSize(0)
	choiceLimit := byteSize(8 << 20)
	flags.Var(&outputLimit, "output-limit", "retained bytes per output stream")
	flags.Var(&worldLimit, "world-transition-limit", "World transition capacity")
	flags.Var(&successBytes, "success-bytes", "retained successful-run bytes per repetition")
	flags.Var(&choiceLimit, "choice-bytes", "runtime choice trace capacity")
	var environment stringList
	var buildTags stringList
	var ioROMounts stringList
	var requiredSemanticProbes stringList
	flags.Var(&environment, "env", "target NAME=VALUE")
	flags.Var(&buildTags, "build-tag", "validated Go build tag")
	flags.Var(&ioROMounts, "io-ro-mount", "read-only HOST_DIRECTORY=TARGET_DIRECTORY mapping")
	flags.Var(&requiredSemanticProbes, "require-probe", "required semantic probe")
	if err := flags.Parse(arguments); err != nil {
		reporter := newQualifyReporter(*jsonOutput, stdout, stderr)
		if writeErr := reporter.Error("invalid_input", err); writeErr != nil {
			fmt.Fprintln(stderr, writeErr)
			return 3
		}
		if !*jsonOutput {
			flags.SetOutput(stderr)
			flags.Usage()
		}
		return 2
	}
	reporter := newQualifyReporter(*jsonOutput, stdout, stderr)
	choiceLimitSet := false
	flags.Visit(func(visited *flag.Flag) {
		if visited.Name == "choice-bytes" {
			choiceLimitSet = true
		}
	})
	resolvedChoiceLimit, err := resolveChoiceTrace(*choices, choiceLimit, choiceLimitSet)
	if err != nil {
		return reportQualifyInputError(reporter, stderr, err)
	}
	if *repeat < 2 || *repeat > maximumQualificationRepeats {
		return reportQualifyInputError(reporter, stderr, fmt.Errorf("--repeat must be between 2 and %d", maximumQualificationRepeats))
	}
	if *replaySuccesses && (*successLimit == 0 || successBytes == 0) {
		return reportQualifyInputError(reporter, stderr, errors.New("--replay-successes requires explicit --success-limit and --success-bytes bounds"))
	}
	if !*replaySuccesses && (*successLimit != 0 || successBytes != 0) {
		return reportQualifyInputError(reporter, stderr, errors.New("--success-limit and --success-bytes require --replay-successes"))
	}
	coverage := runner.CoverageSemantic
	if *choices {
		coverage = runner.CoverageSemanticChoice
	}
	if _, err := resolveExploreCoverage(string(coverage), requiredSemanticProbes); err != nil {
		return reportQualifyInputError(reporter, stderr, err)
	}
	parsedTarget, err := parseTarget(flags.Args())
	if err != nil {
		return reportQualifyInputError(reporter, stderr, err)
	}
	workingDirectory, err := dependencies.workingDirectory()
	if err != nil {
		return reportQualifyUnretainedError(reporter, stderr, "runner_failure", fmt.Errorf("resolve working directory: %w", err), 3)
	}
	toolchain, executable, runnerBuild, err := dependencies.identity(*toolchainRoot)
	if err != nil {
		return reportQualifyUnretainedError(reporter, stderr, "runner_failure", err, 3)
	}
	command := append([]string{"gomad", "qualify"}, arguments...)
	config := runner.Config{
		Seeds: strconv.FormatUint(*seed, 10), Parallel: 1, RunTimeout: *runTimeout, OverallTimeout: *overallTimeout, TerminateGrace: *terminateGrace,
		OnFailure: runner.PolicyAll, FailureBudget: 1, OutputLimit: uint64(outputLimit), WorldTransitionLimit: uint64(worldLimit),
		ChoiceTraceLimit: resolvedChoiceLimit,
		Artifacts:        *artifacts, Environment: environment, IOROMounts: ioROMounts,
		SupervisorCommand: []string{executable, "__supervisor"}, CoordinatorCommand: []string{executable, "__coordinator"}, RunnerBuild: runnerBuild,
		Coverage: coverage, RequiredSemanticProbes: requiredSemanticProbes, CollectRunEvidence: true,
		Target: target.Spec{
			Kind: parsedTarget.kind, Source: parsedTarget.source, Provenance: parsedTarget.provenance, Args: parsedTarget.arguments,
			BuildTags: buildTags, WorkingDir: workingDirectory, ToolchainRoot: toolchain,
		},
	}
	if *replaySuccesses {
		config.KeepSuccesses = runner.KeepSuccessesAll
		config.SuccessArtifactLimit = *successLimit
		config.SuccessBytesLimit = uint64(successBytes)
	}
	ctx, cancel := context.WithTimeout(context.Background(), *overallTimeout)
	defer cancel()
	runs := make([]qualify.Run, 0, *repeat)
	for iteration := uint64(1); iteration <= *repeat; iteration++ {
		if err := reporter.Progress(iteration, *repeat); err != nil {
			fmt.Fprintln(stderr, err)
			return 3
		}
		summary, runErr := dependencies.run(ctx, config)
		if runErr != nil {
			if summary.ChoiceTrace != nil && !*jsonOutput {
				fmt.Fprintf(stderr, "gomad:%s\n", formatChoiceTrace(summary.ChoiceTrace))
			}
			if summary.RunEvidence != nil {
				runs = append(runs, qualificationRun(summary))
			}
			failure := qualificationFailure(runErr, iteration)
			return retainQualificationFailure(reporter, stderr, dependencies.write, *artifacts, command, *seed, *repeat, runs, failure)
		}
		if summary.RunEvidence == nil {
			failure := qualify.Failure{Classification: "runner_failure", Message: "Runner omitted bounded qualification evidence", Iteration: record.Uint64String(iteration)}
			return retainQualificationFailure(reporter, stderr, dependencies.write, *artifacts, command, *seed, *repeat, runs, failure)
		}
		run := qualificationRun(summary)
		if summary.RunEvidence.Outcome.Domain != "success" && run.ArtifactPath == "" {
			failure := qualify.Failure{Classification: "runner_failure", Message: "Runner omitted the retained failure artifact", Iteration: record.Uint64String(iteration)}
			return retainQualificationFailure(reporter, stderr, dependencies.write, *artifacts, command, *seed, *repeat, runs, failure)
		}
		if *replaySuccesses && summary.RunEvidence.Outcome.Domain == "success" && (summary.RetainedSuccesses != 1 || len(summary.SuccessArtifacts) != 1 || run.ArtifactPath == "") {
			failure := qualify.Failure{Classification: "runner_failure", Message: "Runner did not retain exactly one successful replay artifact", Iteration: record.Uint64String(iteration)}
			return retainQualificationFailure(reporter, stderr, dependencies.write, *artifacts, command, *seed, *repeat, runs, failure)
		}
		runs = append(runs, run)
	}

	for index := range runs {
		if runs[index].ArtifactPath == "" || runs[index].Evidence.Outcome.Domain == "success" && !*replaySuccesses {
			continue
		}
		replayed, replayErr := dependencies.replay(ctx, replay.Config{
			ArtifactPath: runs[index].ArtifactPath, ToolchainRoot: toolchain, SupervisorCommand: []string{executable, "__supervisor"},
		})
		runs[index].Replay = &qualify.Replay{ArtifactPath: runs[index].ArtifactPath, Attempted: true}
		if replayErr != nil {
			runs[index].Replay.Divergence = replayErr.Error()
			failure := qualificationReplayFailure(replayErr, uint64(index)+1)
			return retainQualificationFailure(reporter, stderr, dependencies.write, *artifacts, command, *seed, *repeat, runs, failure)
		} else {
			runs[index].Replay.Match = replayed.Match
			runs[index].Replay.Diagnostic = replayed.Diagnostic
			runs[index].Replay.Divergence = replayed.Divergence
			runs[index].Replay.ChoiceReplayStatus = replayed.ChoiceReplayStatus
		}
	}
	report, err := qualify.Build(qualify.Input{Command: command, Runs: runs})
	if err != nil {
		return reportQualifyUnretainedError(reporter, stderr, "runner_failure", err, 3)
	}
	reportPath, err := dependencies.write(*artifacts, report)
	if err != nil {
		return reportQualifyUnretainedError(reporter, stderr, "runner_failure", err, 3)
	}
	if err := reporter.Result(report, reportPath); err != nil {
		fmt.Fprintln(stderr, err)
		return 3
	}
	return qualify.ExitStatus(qualify.Classify(report))
}

func qualificationRun(summary runner.Summary) qualify.Run {
	run := qualify.Run{BatchPath: summary.BatchPath, Evidence: *summary.RunEvidence}
	if summary.RunEvidence.Outcome.Domain == "success" && len(summary.SuccessArtifacts) != 0 {
		run.ArtifactPath = summary.SuccessArtifacts[0]
	} else if len(summary.Artifacts) != 0 {
		run.ArtifactPath = summary.Artifacts[0]
	}
	return run
}

func qualificationReplayFailure(err error, iteration uint64) qualify.Failure {
	classification := "runner_failure"
	if errors.Is(err, context.Canceled) {
		classification = "cancelled"
	} else if errors.Is(err, context.DeadlineExceeded) {
		classification = "overall_timeout"
	}
	return qualify.Failure{Classification: classification, Message: err.Error(), Iteration: record.Uint64String(iteration)}
}

func qualificationFailure(err error, iteration uint64) qualify.Failure {
	failure := qualify.Failure{Classification: classifyExploreError(err), Message: err.Error(), Iteration: record.Uint64String(iteration)}
	var unsupported *target.UnsupportedCapabilityError
	if errors.As(err, &unsupported) {
		failure.ImportPath = unsupported.ImportPath
		failure.Capability = unsupported.Capability
	}
	return failure
}

func retainQualificationFailure(
	reporter *qualifyReporter,
	stderr io.Writer,
	write func(string, qualify.Report) (string, error),
	artifactRoot string,
	command []string,
	seed uint64,
	repeat uint64,
	runs []qualify.Run,
	failure qualify.Failure,
) int {
	report, err := qualify.BuildFailure(command, seed, repeat, runs, failure)
	if err != nil {
		return reportQualifyUnretainedError(reporter, stderr, "runner_failure", err, 3)
	}
	path, err := write(artifactRoot, report)
	if err != nil {
		return reportQualifyUnretainedError(reporter, stderr, "runner_failure", err, 3)
	}
	if err := reporter.Result(report, path); err != nil {
		fmt.Fprintln(stderr, err)
		return 3
	}
	return qualify.ExitStatus(failure.Classification)
}

func reportQualifyInputError(reporter *qualifyReporter, stderr io.Writer, err error) int {
	return reportQualifyUnretainedError(reporter, stderr, "invalid_input", err, 2)
}

func reportQualifyUnretainedError(reporter *qualifyReporter, stderr io.Writer, classification string, err error, status int) int {
	if writeErr := reporter.Error(classification, err); writeErr != nil {
		fmt.Fprintln(stderr, writeErr)
		return 3
	}
	return status
}

func newQualifyReporter(jsonOutput bool, stdout, stderr io.Writer) *qualifyReporter {
	return &qualifyReporter{json: jsonOutput, stdout: stdout, stderr: stderr}
}

func (reporter *qualifyReporter) Progress(iteration, repeat uint64) error {
	if reporter.json {
		return qualify.WriteProgressEvent(reporter.stdout, iteration, repeat)
	}
	_, err := fmt.Fprintf(reporter.stderr, "gomad: qualification iteration=%d/%d\n", iteration, repeat)
	return err
}

func (reporter *qualifyReporter) Error(classification string, err error) error {
	if reporter.json {
		return qualify.WriteErrorEvent(reporter.stdout, classification, err)
	}
	_, writeErr := fmt.Fprintf(reporter.stderr, "gomad: %s: %v\n", classification, err)
	return writeErr
}

func (reporter *qualifyReporter) Result(report qualify.Report, path string) error {
	if reporter.json {
		return qualify.WriteResultEvent(reporter.stdout, report, path)
	}
	_, err := fmt.Fprintf(reporter.stdout, "gomad: qualification qualified=%t deterministic=%t target-success=%t seed=%d repeat=%d report=%s\n", report.Qualified, report.Deterministic, report.TargetSuccess, report.Seed, report.Repeat, path)
	if err == nil && report.Evidence != nil && report.Evidence.Choices != nil {
		choices := report.Evidence.Choices
		_, err = fmt.Fprintf(reporter.stdout, "gomad: choices profile=%s records=%d decisions=%d branching=%d runnable=%d select-poll=%d select-result=%d sha256=%s tape-sha256=%s terminal=%s\n", choices.Profile, choices.Records, choices.Decisions, choices.BranchingRecords, choices.Runnable, choices.SelectPoll, choices.SelectResult, choices.SHA256, choices.TapeSHA256, choices.TerminalState)
	}
	if err == nil && report.FirstDivergence != "" {
		_, err = fmt.Fprintf(reporter.stdout, "gomad: first-divergence=%s\n", report.FirstDivergence)
	}
	if err == nil {
		for _, run := range report.Runs {
			if run.Replay == nil {
				continue
			}
			if _, err = fmt.Fprintf(reporter.stdout, "gomad: replay artifact=%s attempted=%t match=%t diagnostic=%t choice-replay=%s divergence=%s\n", run.Replay.ArtifactPath, run.Replay.Attempted, run.Replay.Match, run.Replay.Diagnostic, run.Replay.ChoiceReplayStatus, run.Replay.Divergence); err != nil {
				break
			}
		}
	}
	if err == nil && report.Failure != nil {
		_, err = fmt.Fprintf(reporter.stdout, "gomad: first-boundary classification=%s iteration=%d import=%s capability=%s message=%s\n", report.Failure.Classification, report.Failure.Iteration, report.Failure.ImportPath, report.Failure.Capability, report.Failure.Message)
	}
	return err
}
