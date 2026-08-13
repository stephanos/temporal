package main

import (
	"context"
	"encoding/json"
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

const qualifyEventSchema = "gomadv3.qualify-event/v1"

const maximumQualificationRepeats = 32

type qualifyDependencies struct {
	identity         func(string) (string, string, string, error)
	workingDirectory func() (string, error)
	run              func(context.Context, runner.Config) (runner.Summary, error)
	replay           func(context.Context, replay.Config) (replay.Result, error)
	write            func(string, qualify.Report) (string, error)
}

type qualifyEvent struct {
	Schema         string          `json:"schema"`
	Type           string          `json:"type"`
	Classification string          `json:"classification,omitempty"`
	Message        string          `json:"message,omitempty"`
	Iteration      uint64          `json:"iteration,omitempty"`
	Repeat         uint64          `json:"repeat,omitempty"`
	ReportPath     string          `json:"report_path,omitempty"`
	Report         *qualify.Report `json:"report,omitempty"`
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
	outputLimit := byteSize(8 << 20)
	worldLimit := byteSize(64 << 20)
	flags.Var(&outputLimit, "output-limit", "retained bytes per output stream")
	flags.Var(&worldLimit, "world-transition-limit", "World transition capacity")
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
	if *repeat < 2 || *repeat > maximumQualificationRepeats {
		return reportQualifyInputError(reporter, stderr, fmt.Errorf("--repeat must be between 2 and %d", maximumQualificationRepeats))
	}
	if _, err := resolveExploreCoverage(string(runner.CoverageSemantic), requiredSemanticProbes); err != nil {
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
		Artifacts: *artifacts, Environment: environment, IOROMounts: ioROMounts,
		SupervisorCommand: []string{executable, "__supervisor"}, CoordinatorCommand: []string{executable, "__coordinator"}, RunnerBuild: runnerBuild,
		Coverage: runner.CoverageSemantic, RequiredSemanticProbes: requiredSemanticProbes, CollectRunEvidence: true,
		Target: target.Spec{
			Kind: parsedTarget.kind, Source: parsedTarget.source, Provenance: parsedTarget.provenance, Args: parsedTarget.arguments,
			BuildTags: buildTags, WorkingDir: workingDirectory, ToolchainRoot: toolchain,
		},
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
		runs = append(runs, run)
	}

	var replayEvidence *qualify.Replay
	for _, run := range runs {
		if run.Evidence.Outcome.Domain == "success" {
			continue
		}
		replayed, replayErr := dependencies.replay(ctx, replay.Config{
			ArtifactPath: run.ArtifactPath, ToolchainRoot: toolchain, SupervisorCommand: []string{executable, "__supervisor"},
		})
		replayEvidence = &qualify.Replay{ArtifactPath: run.ArtifactPath, Attempted: true}
		if replayErr != nil {
			replayEvidence.Divergence = replayErr.Error()
		} else {
			replayEvidence.Match = replayed.Match
			replayEvidence.Diagnostic = replayed.Diagnostic
			replayEvidence.Divergence = replayed.Divergence
		}
		break
	}
	report, err := qualify.Build(qualify.Input{Command: command, Runs: runs, Replay: replayEvidence})
	if err != nil {
		return reportQualifyUnretainedError(reporter, stderr, "runner_failure", err, 3)
	}
	reportPath, err := dependencies.write(*artifacts, report)
	if err != nil {
		return reportQualifyUnretainedError(reporter, stderr, "runner_failure", err, 3)
	}
	classification := classifyQualification(report)
	if err := reporter.Result(report, reportPath, classification); err != nil {
		fmt.Fprintln(stderr, err)
		return 3
	}
	if report.Qualified {
		return 0
	}
	return 1
}

func qualificationRun(summary runner.Summary) qualify.Run {
	run := qualify.Run{BatchPath: summary.BatchPath, Evidence: *summary.RunEvidence}
	if len(summary.Artifacts) != 0 {
		run.ArtifactPath = summary.Artifacts[0]
	}
	return run
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
	if err := reporter.Result(report, path, failure.Classification); err != nil {
		fmt.Fprintln(stderr, err)
		return 3
	}
	switch failure.Classification {
	case "unsupported_target", "invalid_input":
		return 2
	case "semantic_coverage_failure":
		return 1
	default:
		return 3
	}
}

func classifyQualification(report qualify.Report) string {
	if !report.Deterministic {
		return "nondeterministic"
	}
	if report.Replay != nil && !report.Replay.Match {
		return "replay_divergence"
	}
	if !report.TargetSuccess {
		return "target_failure"
	}
	return "qualified"
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
		return reporter.writeEvent(qualifyEvent{Schema: qualifyEventSchema, Type: "progress", Iteration: iteration, Repeat: repeat})
	}
	_, err := fmt.Fprintf(reporter.stderr, "gomad: qualification iteration=%d/%d\n", iteration, repeat)
	return err
}

func (reporter *qualifyReporter) Error(classification string, err error) error {
	if reporter.json {
		return reporter.writeEvent(qualifyEvent{Schema: qualifyEventSchema, Type: "error", Classification: classification, Message: err.Error()})
	}
	_, writeErr := fmt.Fprintf(reporter.stderr, "gomad: %s: %v\n", classification, err)
	return writeErr
}

func (reporter *qualifyReporter) Result(report qualify.Report, path, classification string) error {
	if reporter.json {
		return reporter.writeEvent(qualifyEvent{Schema: qualifyEventSchema, Type: "result", Classification: classification, ReportPath: path, Report: &report})
	}
	_, err := fmt.Fprintf(reporter.stdout, "gomad: qualification qualified=%t deterministic=%t target-success=%t seed=%d repeat=%d report=%s\n", report.Qualified, report.Deterministic, report.TargetSuccess, report.Seed, report.Repeat, path)
	if err == nil && report.FirstDivergence != "" {
		_, err = fmt.Fprintf(reporter.stdout, "gomad: first-divergence=%s\n", report.FirstDivergence)
	}
	if err == nil && report.Replay != nil {
		_, err = fmt.Fprintf(reporter.stdout, "gomad: replay artifact=%s attempted=%t match=%t diagnostic=%t divergence=%s\n", report.Replay.ArtifactPath, report.Replay.Attempted, report.Replay.Match, report.Replay.Diagnostic, report.Replay.Divergence)
	}
	if err == nil && report.Failure != nil {
		_, err = fmt.Fprintf(reporter.stdout, "gomad: first-boundary classification=%s iteration=%d import=%s capability=%s message=%s\n", report.Failure.Classification, report.Failure.Iteration, report.Failure.ImportPath, report.Failure.Capability, report.Failure.Message)
	}
	return err
}

func (reporter *qualifyReporter) writeEvent(event qualifyEvent) error {
	encoded, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("encode qualification event: %w", err)
	}
	_, err = fmt.Fprintf(reporter.stdout, "%s\n", encoded)
	return err
}
