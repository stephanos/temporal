package main

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"go.temporal.io/server/tools/gomadv3/internal/doctor"
	gomadinspect "go.temporal.io/server/tools/gomadv3/internal/inspect"
	"go.temporal.io/server/tools/gomadv3/internal/ioprofile"
	"go.temporal.io/server/tools/gomadv3/internal/process"
	"go.temporal.io/server/tools/gomadv3/internal/replay"
	"go.temporal.io/server/tools/gomadv3/internal/runner"
	"go.temporal.io/server/tools/gomadv3/internal/target"
)

const usage = `usage:
  gomad explore [flags] exec --provenance FILE -- BINARY [ARG ...]
  gomad explore [flags] go-run PACKAGE -- [ARG ...]
  gomad explore [flags] go-test PACKAGE -- [TEST_BINARY_ARG ...]
  gomad qualify [flags] exec --provenance FILE -- BINARY [ARG ...]
  gomad qualify [flags] go-run PACKAGE -- [ARG ...]
  gomad qualify [flags] go-test PACKAGE -- [TEST_BINARY_ARG ...]
  gomad resume [--json] INTERRUPTED_BATCH
  gomad replay [--verify-only] ARTIFACT_DIR
  gomad doctor [--artifacts DIR] [--json]
  gomad inspect [--json] ARTIFACT_OR_BATCH
`

type byteSize uint64

func (size *byteSize) String() string {
	return strconv.FormatUint(uint64(*size), 10)
}

func (size *byteSize) Set(input string) error {
	multiplier := uint64(1)
	number := input
	for suffix, value := range map[string]uint64{"KiB": 1 << 10, "MiB": 1 << 20, "GiB": 1 << 30} {
		if strings.HasSuffix(input, suffix) {
			multiplier = value
			number = strings.TrimSuffix(input, suffix)
			break
		}
	}
	if number == "" || len(number) > 1 && number[0] == '0' {
		return fmt.Errorf("invalid byte size %q", input)
	}
	value, err := strconv.ParseUint(number, 10, 64)
	if err != nil || value == 0 || value > ^uint64(0)/multiplier {
		return fmt.Errorf("invalid byte size %q", input)
	}
	if multiplier == 1 && input != number {
		return fmt.Errorf("invalid byte size %q", input)
	}
	*size = byteSize(value * multiplier)
	return nil
}

type stringList []string

func (values *stringList) String() string {
	return strings.Join(*values, ",")
}

func (values *stringList) Set(value string) error {
	*values = append(*values, value)
	return nil
}

type targetInput struct {
	kind       target.Kind
	source     string
	provenance string
	arguments  []string
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(arguments []string, stdout, stderr io.Writer) int {
	if len(arguments) == 0 {
		fmt.Fprint(stderr, usage)
		return 2
	}
	switch arguments[0] {
	case "__coordinator":
		if err := runner.CoordinatorMain(os.Stdin, os.Stdout); err != nil {
			fmt.Fprintln(stderr, err)
			return 3
		}
		return 0
	case "__target_bootstrap":
		if err := process.BootstrapMain(); err != nil {
			fmt.Fprintln(stderr, err)
			return 3
		}
		return 0
	case "__supervisor":
		if err := process.SupervisorMain(); err != nil {
			fmt.Fprintln(stderr, err)
			return 3
		}
		return 0
	case "explore":
		return runExplore(arguments[1:], stdout, stderr)
	case "qualify":
		return runQualify(arguments[1:], stdout, stderr)
	case "resume":
		return runResume(arguments[1:], stdout, stderr)
	case "replay":
		return runReplay(arguments[1:], stdout, stderr)
	case "doctor":
		executable, err := os.Executable()
		if err != nil {
			fmt.Fprintf(stderr, "resolve gomad executable: %v\n", err)
			return 3
		}
		return runDoctor(arguments[1:], stdout, stderr, executable)
	case "inspect":
		return runInspect(arguments[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "unknown gomad command %q\n%s", arguments[0], usage)
		return 2
	}
}

func runInspect(arguments []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("gomad inspect", flag.ContinueOnError)
	flags.SetOutput(stderr)
	jsonOutput := flags.Bool("json", false, "emit stable JSON")
	if err := flags.Parse(arguments); err != nil {
		return 2
	}
	if flags.NArg() != 1 {
		fmt.Fprint(stderr, usage)
		return 2
	}
	report, err := gomadinspect.Open(flags.Arg(0))
	if err != nil {
		fmt.Fprintf(stderr, "inspect %s: %v\n", flags.Arg(0), err)
		return 2
	}
	if *jsonOutput {
		encoded, marshalErr := json.Marshal(report)
		if marshalErr != nil {
			fmt.Fprintf(stderr, "encode inspection report: %v\n", marshalErr)
			return 3
		}
		fmt.Fprintf(stdout, "%s\n", encoded)
		return 0
	}
	printInspection(stdout, report)
	return 0
}

func printInspection(output io.Writer, report gomadinspect.Report) {
	fmt.Fprintf(output, "gomad inspect: kind=%s path=%s\n", report.Kind, report.Path)
	if inspected := report.Artifact; inspected != nil {
		fmt.Fprintf(output, "identity: record=%s batch=%s ordinal=%d seed=%d toolchain=%s runner=%s\n", inspected.RecordHash, inspected.BatchID, inspected.SelectionOrdinal, inspected.Seed, inspected.Toolchain.BuildKey, inspected.Runner.RunnerBuild)
		fmt.Fprintf(output, "target: kind=%s source=%s sha256=%s size=%d argv=%q tags=%q\n", inspected.Target.Kind, inspected.Target.Source, inspected.Target.SHA256, inspected.Target.Size, inspected.Target.Argv, inspected.Target.BuildTags)
		fmt.Fprintf(output, "outcome: domain=%s reason=%s termination=%s signature=%s replay-match=%s\n", inspected.Outcome.Domain, inspected.Outcome.Reason, inspected.Outcome.Termination, inspected.Outcome.FailureSignature, optionalBool(inspected.Outcome.ReplayMatch))
		if inspected.FirstDivergence != "" {
			fmt.Fprintf(output, "first-divergence: %s\n", inspected.FirstDivergence)
		}
		if transcript := inspected.Transcript; transcript != nil {
			fmt.Fprintf(output, "transcript: records=%d bytes=%d sha256=%s\n", transcript.Records, transcript.Bytes, transcript.SHA256)
		} else {
			fmt.Fprintln(output, "transcript: none")
		}
		if mounts := inspected.CapturedMounts; mounts != nil {
			fmt.Fprintf(output, "captured-mounts: mappings=%q entries=%d missing=%d bytes=%d\n", mounts.Mappings, mounts.Entries, mounts.NotExist, mounts.TotalBytes)
		} else {
			fmt.Fprintln(output, "captured-mounts: none")
		}
		fmt.Fprintf(output, "stdout: bytes=%d retained=%d discarded=%d truncated=%t sha256=%s\n", inspected.Stdout.TotalBytes, inspected.Stdout.RetainedBytes, inspected.Stdout.DiscardedBytes, inspected.Stdout.Truncated, inspected.Stdout.FullSHA256)
		fmt.Fprintf(output, "stderr: bytes=%d retained=%d discarded=%d truncated=%t sha256=%s\n", inspected.Stderr.TotalBytes, inspected.Stderr.RetainedBytes, inspected.Stderr.DiscardedBytes, inspected.Stderr.Truncated, inspected.Stderr.FullSHA256)
		fmt.Fprintf(output, "replay: %s\n", inspected.ReplayCommand)
		return
	}
	batch := report.Batch
	fmt.Fprintf(output, "batch: id=%s selection=%s selected=%d attempted=%d succeeded=%d failures=%d watchdogs=%d cancelled=%d distinct=%d retained-successes=%d retained-success-bytes=%d stop=%s runs=%s\n", batch.RunID, batch.Selection, batch.SelectionCount, batch.Attempted, batch.Succeeded, batch.Failures, batch.Watchdogs, batch.Cancelled, batch.DistinctFailures, batch.RetainedSuccesses, batch.RetainedSuccessBytes, batch.StopReason, batch.RunsSHA256)
	for _, run := range batch.Runs {
		transcript := "none"
		if run.TranscriptSHA256 != nil && run.TranscriptRecords != nil {
			transcript = fmt.Sprintf("%s/%d", *run.TranscriptSHA256, *run.TranscriptRecords)
		}
		fmt.Fprintf(output, "run: ordinal=%d seed=%d domain=%s reason=%s termination=%s elapsed=%dns transcript=%s\n", run.SelectionOrdinal, run.Seed, run.Domain, run.Reason, run.Termination, run.ElapsedNanos, transcript)
	}
	for _, failure := range batch.FailureArtifacts {
		fmt.Fprintf(output, "failure: signature=%s path=%s\nreplay: %s\n", failure.Signature, failure.Path, failure.ReplayCommand)
	}
	for _, success := range batch.SuccessArtifacts {
		fmt.Fprintf(output, "success: bytes=%d novel=%q path=%s\nreplay: %s\n", success.StoredBytes, success.NovelProbes, success.Path, success.ReplayCommand)
	}
}

func optionalBool(value *bool) string {
	if value == nil {
		return "not-recorded"
	}
	return strconv.FormatBool(*value)
}

func runDoctor(arguments []string, stdout, stderr io.Writer, executable string) int {
	flags := flag.NewFlagSet("gomad doctor", flag.ContinueOnError)
	flags.SetOutput(stderr)
	artifacts := flags.String("artifacts", ".gomad/artifacts", "artifact root to verify")
	jsonOutput := flags.Bool("json", false, "emit stable JSON")
	if err := flags.Parse(arguments); err != nil {
		return 2
	}
	if flags.NArg() != 0 {
		fmt.Fprint(stderr, usage)
		return 2
	}
	executable, err := filepath.Abs(executable)
	if err != nil {
		fmt.Fprintf(stderr, "resolve gomad executable path: %v\n", err)
		return 3
	}
	artifactRoot, err := filepath.Abs(*artifacts)
	if err != nil {
		fmt.Fprintf(stderr, "resolve artifact directory: %v\n", err)
		return 2
	}
	root := filepath.Dir(filepath.Dir(executable))
	report := doctor.Check(doctor.Config{
		Root: root, RunnerPath: executable, ArtifactRoot: artifactRoot, HostOS: runtime.GOOS, HostArch: runtime.GOARCH,
	})
	if *jsonOutput {
		encoded, marshalErr := json.Marshal(report)
		if marshalErr != nil {
			fmt.Fprintf(stderr, "encode doctor report: %v\n", marshalErr)
			return 3
		}
		fmt.Fprintf(stdout, "%s\n", encoded)
	} else {
		fmt.Fprintf(stdout, "gomad doctor: available=%t host=%s go=%s toolchain=%s runner=%s boundary=%s\n", report.Available, report.Host, report.GoVersion, report.ToolchainBuild, report.RunnerBuild, report.BoundaryManifestVersion)
		for _, check := range report.Checks {
			fmt.Fprintf(stdout, "%-10s %-5s %s\n", check.Name, check.Status, check.Detail)
		}
		fmt.Fprintf(stdout, "build: %s\n", report.BuildCommand)
	}
	if !report.Available {
		return 1
	}
	return 0
}

func runExplore(arguments []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("gomad explore", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	seeds := flags.String("seeds", "1", "seed set or inclusive ranges")
	count := flags.Uint64("count", 0, "explore seeds 0 through N-1")
	parallel := flags.Int("parallel", min(runtime.NumCPU(), 8), "maximum active targets")
	runTimeout := flags.Duration("run-timeout", 30*time.Second, "per-seed host deadline")
	overallTimeout := flags.Duration("overall-timeout", 10*time.Minute, "complete exploration host deadline")
	terminateGrace := flags.Duration("terminate-grace", 2*time.Second, "termination grace inside deadlines")
	onFailure := flags.String("on-failure", string(runner.PolicyFirst), "first, budget, or all")
	failureBudget := flags.Uint64("failure-budget", 1, "distinct failure signature threshold")
	artifacts := flags.String("artifacts", ".gomad/artifacts", "artifact root")
	jsonOutput := flags.Bool("json", false, "emit stable JSON events")
	coverage := flags.String("coverage", string(runner.CoverageNone), "none or semantic")
	keepSuccesses := flags.String("keep-successes", string(runner.KeepSuccessesNone), "none, novel, or all")
	successLimit := flags.Uint64("success-limit", 0, "maximum retained successful runs")
	outputLimit := byteSize(8 << 20)
	worldLimit := byteSize(64 << 20)
	successBytes := byteSize(0)
	flags.Var(&outputLimit, "output-limit", "retained bytes per output stream")
	flags.Var(&worldLimit, "world-transition-limit", "World transition capacity")
	flags.Var(&successBytes, "success-bytes", "total retained successful-run bytes")
	var environment stringList
	var buildTags stringList
	var ioROMounts stringList
	var requiredSemanticProbes stringList
	flags.Var(&environment, "env", "target NAME=VALUE")
	flags.Var(&buildTags, "build-tag", "validated Go build tag")
	flags.Var(&ioROMounts, "io-ro-mount", "read-only HOST_DIRECTORY=TARGET_DIRECTORY mapping")
	flags.Var(&requiredSemanticProbes, "require-probe", "required semantic probe (requires --coverage=semantic)")
	if err := flags.Parse(arguments); err != nil {
		reporter := newExploreReporter(*jsonOutput, stdout, stderr)
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
	reporter := newExploreReporter(*jsonOutput, stdout, stderr)
	var seedsSet, countSet bool
	flags.Visit(func(visited *flag.Flag) {
		switch visited.Name {
		case "seeds":
			seedsSet = true
		case "count":
			countSet = true
		}
	})
	resolvedSeeds, err := resolveExploreSeeds(*seeds, *count, seedsSet, countSet)
	if err != nil {
		if writeErr := reporter.Error("invalid_input", err); writeErr != nil {
			fmt.Fprintln(stderr, writeErr)
			return 3
		}
		return 2
	}
	coverageMode, err := resolveExploreCoverage(*coverage, requiredSemanticProbes)
	if err != nil {
		if writeErr := reporter.Error("invalid_input", err); writeErr != nil {
			fmt.Fprintln(stderr, writeErr)
			return 3
		}
		return 2
	}
	parsedTarget, err := parseTarget(flags.Args())
	if err != nil {
		if writeErr := reporter.Error("invalid_input", err); writeErr != nil {
			fmt.Fprintln(stderr, writeErr)
			return 3
		}
		return 2
	}
	workingDirectory, err := os.Getwd()
	if err != nil {
		if writeErr := reporter.Error("runner_failure", fmt.Errorf("resolve working directory: %w", err)); writeErr != nil {
			fmt.Fprintln(stderr, writeErr)
		}
		return 3
	}
	toolchain, executable, runnerBuild, err := localIdentity()
	if err != nil {
		if writeErr := reporter.Error("runner_failure", err); writeErr != nil {
			fmt.Fprintln(stderr, writeErr)
		}
		return 3
	}
	config := runner.Config{
		Seeds: resolvedSeeds, Parallel: *parallel, RunTimeout: *runTimeout, OverallTimeout: *overallTimeout, TerminateGrace: *terminateGrace,
		OnFailure: runner.FailurePolicy(*onFailure), FailureBudget: *failureBudget, OutputLimit: uint64(outputLimit), WorldTransitionLimit: uint64(worldLimit),
		Artifacts: *artifacts, Environment: environment, IOROMounts: ioROMounts, SupervisorCommand: []string{executable, "__supervisor"}, CoordinatorCommand: []string{executable, "__coordinator"}, RunnerBuild: runnerBuild,
		Coverage: coverageMode, RequiredSemanticProbes: requiredSemanticProbes,
		KeepSuccesses: runner.KeepSuccesses(*keepSuccesses), SuccessArtifactLimit: *successLimit, SuccessBytesLimit: uint64(successBytes),
		Progress: reporter.Progress, ProgressInterval: 5 * time.Second,
		Target: target.Spec{
			Kind: parsedTarget.kind, Source: parsedTarget.source, Provenance: parsedTarget.provenance, Args: parsedTarget.arguments,
			BuildTags: buildTags, WorkingDir: workingDirectory, ToolchainRoot: toolchain,
		},
	}
	summary, err := runner.Run(context.Background(), config)
	if err != nil {
		classification := classifyExploreError(err)
		if writeErr := reporter.Error(classification, err); writeErr != nil {
			fmt.Fprintln(stderr, writeErr)
			return 3
		}
		if classification == "runner_failure" {
			return 3
		}
		if classification == "semantic_coverage_failure" {
			return 1
		}
		return 2
	}
	if err := reporter.Result(summary); err != nil {
		fmt.Fprintln(stderr, err)
		return 3
	}
	if summary.Failures != 0 {
		return 1
	}
	return 0
}

func resolveExploreSeeds(seeds string, count uint64, seedsSet, countSet bool) (string, error) {
	if seedsSet && countSet {
		return "", fmt.Errorf("--count and --seeds are mutually exclusive")
	}
	if !countSet {
		return seeds, nil
	}
	if count == 0 {
		return "", fmt.Errorf("--count must be greater than zero")
	}
	if count == 1 {
		return "0", nil
	}
	return "0-" + strconv.FormatUint(count-1, 10), nil
}

func resolveExploreCoverage(value string, required []string) (runner.CoverageMode, error) {
	mode := runner.CoverageMode(value)
	switch mode {
	case runner.CoverageNone:
		if len(required) != 0 {
			return "", fmt.Errorf("--require-probe requires --coverage=semantic")
		}
	case runner.CoverageSemantic:
		if _, err := ioprofile.MissingRequiredSemanticProbes(ioprofile.SemanticCoverage{}, required); err != nil {
			return "", err
		}
	default:
		return "", fmt.Errorf("unknown coverage mode %q", value)
	}
	return mode, nil
}

func runReplay(arguments []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("gomad replay", flag.ContinueOnError)
	flags.SetOutput(stderr)
	verifyOnly := flags.Bool("verify-only", false, "validate without executing the target")
	if err := flags.Parse(arguments); err != nil {
		return 2
	}
	if flags.NArg() != 1 {
		fmt.Fprint(stderr, usage)
		return 2
	}
	toolchain, executable, _, err := localIdentity()
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 3
	}
	result, err := replay.Replay(context.Background(), replay.Config{
		ArtifactPath: flags.Arg(0), VerifyOnly: *verifyOnly, ToolchainRoot: toolchain, SupervisorCommand: []string{executable, "__supervisor"},
	})
	if err != nil {
		var preflightError *replay.PreflightError
		if errors.As(err, &preflightError) {
			fmt.Fprintln(stderr, err)
			return 2
		}
		fmt.Fprintln(stderr, err)
		return 3
	}
	if *verifyOnly {
		fmt.Fprintf(stdout, "gomad: verified %s\n", result.Artifact.Path)
		return 0
	}
	status, err := reportReplayResult(stdout, result)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 3
	}
	return status
}

func reportReplayResult(output io.Writer, result replay.Result) (int, error) {
	if !result.Match {
		_, err := fmt.Fprintf(output, "gomad: reproduced=false divergence=%s\n", result.Divergence)
		return 1, err
	}
	if result.Diagnostic {
		_, err := fmt.Fprintln(output, "gomad: reproduced=true diagnostic=true result=watchdog_observation")
		return 1, err
	}
	if result.Artifact.Manifest.Outcome.Domain == "success" {
		_, err := fmt.Fprintln(output, "gomad: reproduced=true diagnostic=false result=success")
		return 0, err
	}
	_, err := fmt.Fprintln(output, "gomad: reproduced=true diagnostic=false result=target_failure")
	return 1, err
}

func parseTarget(arguments []string) (targetInput, error) {
	if len(arguments) == 0 {
		return targetInput{}, fmt.Errorf("target kind is required")
	}
	switch arguments[0] {
	case string(target.KindExec):
		if len(arguments) < 5 || arguments[1] != "--provenance" || arguments[2] == "" || arguments[3] != "--" || arguments[4] == "" {
			return targetInput{}, fmt.Errorf("exec requires --provenance FILE -- BINARY [ARG ...]")
		}
		return targetInput{kind: target.KindExec, source: arguments[4], provenance: arguments[2], arguments: append([]string(nil), arguments[5:]...)}, nil
	case string(target.KindGoRun), string(target.KindGoTest):
		if len(arguments) < 2 || arguments[1] == "" {
			return targetInput{}, fmt.Errorf("%s requires one package", arguments[0])
		}
		remaining := arguments[2:]
		if len(remaining) > 0 {
			if remaining[0] != "--" {
				return targetInput{}, fmt.Errorf("%s target arguments require -- separator", arguments[0])
			}
			remaining = remaining[1:]
		}
		return targetInput{kind: target.Kind(arguments[0]), source: arguments[1], arguments: append([]string(nil), remaining...)}, nil
	default:
		return targetInput{}, fmt.Errorf("unknown target kind %q", arguments[0])
	}
}

func localIdentity() (toolchainRoot, executable, runnerBuild string, err error) {
	executable, err = os.Executable()
	if err != nil {
		return "", "", "", fmt.Errorf("resolve gomad executable: %w", err)
	}
	executable, err = filepath.Abs(executable)
	if err != nil {
		return "", "", "", fmt.Errorf("resolve gomad executable path: %w", err)
	}
	toolchainRoot = filepath.Join(filepath.Dir(filepath.Dir(executable)), ".toolchain")
	bytes, err := os.ReadFile(executable)
	if err != nil {
		return "", "", "", fmt.Errorf("hash gomad executable: %w", err)
	}
	digest := sha256.Sum256(bytes)
	return toolchainRoot, executable, fmt.Sprintf("sha256:%x", digest), nil
}
