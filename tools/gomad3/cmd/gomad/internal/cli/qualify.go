package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"
	"time"

	"go.temporal.io/server/tools/gomad3/qualification"
	qualificationworkload "go.temporal.io/server/tools/gomad3/qualification/workload"
	"go.temporal.io/server/tools/gomad3/runner"
	"go.temporal.io/server/tools/gomad3/target"
)

const maximumQualificationRepeats = 32

type qualifyDependencies struct {
	identity         func(string) (string, string, string, error)
	workingDirectory func() (string, error)
	workload         func(context.Context, qualificationworkload.Spec) (qualificationworkload.Result, error)
	run              func(context.Context, runner.CampaignSpec) (runner.CampaignResult, error)
	replay           func(context.Context, runner.ReplaySpec) (runner.ReplayResult, error)
	write            func(string, qualification.QualificationReport) (string, error)
}

type qualifyReporter struct {
	json   bool
	stdout io.Writer
	stderr io.Writer
}

func runQualify(arguments []string, stdout, stderr io.Writer) int {
	return runQualifyWith(arguments, stdout, stderr, qualifyDependencies{
		identity: localIdentity, workingDirectory: os.Getwd, workload: qualificationworkload.Run,
	})
}

func runQualifyWith(arguments []string, stdout, stderr io.Writer, dependencies qualifyDependencies) int {
	flags := flag.NewFlagSet("gomad qualify", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	seed := flags.Uint64("seed", 1, "schedule seed")
	repeat := flags.Uint64("repeat", 2, "fresh same-seed repetitions")
	runTimeout := flags.Duration("execution-timeout", 30*time.Second, "per-execution host deadline")
	overallTimeout := flags.Duration("overall-timeout", 10*time.Minute, "complete qualification host deadline")
	terminateGrace := flags.Duration("terminate-grace", 2*time.Second, "termination grace inside deadlines")
	artifacts := flags.String("artifacts", ".gomad/artifacts", "artifact and qualification report root")
	toolchainRoot := flags.String("toolchain-root", "", "absolute pinned toolchain root")
	capabilityMode := flags.String("capability-mode", string(target.CapabilityModeClosure), "closure, linked, or guarded capability assessment")
	jsonOutput := flags.Bool("json", false, "emit stable JSON events")
	choices := flags.Bool("choices", false, "record bounded runtime choices")
	replaySuccesses := flags.Bool("replay-successes", false, "retain and replay every successful repetition")
	successLimit := flags.Uint64("success-limit", 0, "maximum retained successful executions per repetition")
	outputLimit := byteSize(8 << 20)
	worldLimit := byteSize(64 << 20)
	successBytes := byteSize(0)
	choiceLimit := byteSize(8 << 20)
	flags.Var(&outputLimit, "output-limit", "retained bytes per output stream")
	flags.Var(&worldLimit, "world-transition-limit", "World transition capacity")
	flags.Var(&successBytes, "success-bytes", "retained successful-execution bytes per repetition")
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
	resolvedCapabilityMode, err := parseCapabilityMode(*capabilityMode)
	if err != nil {
		return reportQualifyInputError(reporter, stderr, err)
	}
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
	config := runner.CampaignSpec{
		Seeds: strconv.FormatUint(*seed, 10), Parallel: 1, ExecutionTimeout: *runTimeout, OverallTimeout: *overallTimeout, TerminateGrace: *terminateGrace,
		OnFailure: runner.PolicyAll, FailureBudget: 1, OutputLimit: uint64(outputLimit), WorldTransitionLimit: uint64(worldLimit),
		ChoiceTraceLimit: resolvedChoiceLimit,
		Artifacts:        *artifacts, Environment: environment, IOROMounts: ioROMounts,
		SupervisorCommand: []string{executable, "__supervisor"}, CoordinatorCommand: []string{executable, "__coordinator"}, RunnerBuild: runnerBuild,
		Coverage: coverage, RequiredSemanticProbes: requiredSemanticProbes, CollectExecutionEvidence: true,
		Target: target.Spec{
			Kind: parsedTarget.kind, Source: parsedTarget.source, Provenance: parsedTarget.provenance, Args: parsedTarget.arguments,
			BuildTags: buildTags, WorkingDir: workingDirectory, ToolchainRoot: toolchain, CapabilityMode: resolvedCapabilityMode,
		},
	}
	if *replaySuccesses {
		config.KeepSuccesses = runner.KeepSuccessesAll
		config.SuccessArtifactLimit = *successLimit
		config.SuccessBytesLimit = uint64(successBytes)
	}
	ctx, cancel := context.WithTimeout(context.Background(), *overallTimeout)
	defer cancel()
	runWorkload := dependencies.workload
	if runWorkload == nil {
		runWorkload = qualificationworkload.Run
	}
	result, err := runWorkload(ctx, qualificationworkload.Spec{
		Command: command, Seed: *seed, Repeat: *repeat, ArtifactRoot: *artifacts, Campaign: config,
		Replay:          runner.ReplaySpec{ToolchainRoot: toolchain, SupervisorCommand: []string{executable, "__supervisor"}},
		ReplaySuccesses: *replaySuccesses,
		Progress: func(event qualificationworkload.Progress) error {
			return reporter.Progress(event.Iteration, event.Repeat)
		},
		Explore: dependencies.run, ReplayArtifact: dependencies.replay, Write: dependencies.write,
	})
	if err != nil {
		return reportQualifyUnretainedError(reporter, stderr, "runner_failure", err, 3)
	}
	if result.ChoiceTrace != nil && !*jsonOutput {
		if _, err := fmt.Fprintf(stderr, "gomad:%s\n", formatChoiceTrace(result.ChoiceTrace)); err != nil {
			return 3
		}
	}
	if err := reporter.Result(result.Report, result.ReportPath); err != nil {
		fmt.Fprintln(stderr, err)
		return 3
	}
	return qualification.ExitStatus(qualification.ClassifyQualification(result.Report))
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
		return qualification.WriteProgressEvent(reporter.stdout, iteration, repeat)
	}
	_, err := fmt.Fprintf(reporter.stderr, "gomad: qualification iteration=%d/%d\n", iteration, repeat)
	return err
}

func (reporter *qualifyReporter) Error(classification string, err error) error {
	if reporter.json {
		return qualification.WriteErrorEvent(reporter.stdout, classification, err)
	}
	_, writeErr := fmt.Fprintf(reporter.stderr, "gomad: %s: %v\n", classification, err)
	return writeErr
}

func (reporter *qualifyReporter) Result(report qualification.QualificationReport, path string) error {
	if reporter.json {
		return qualification.WriteResultEvent(reporter.stdout, report, path)
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
		for _, run := range report.Executions {
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
