package main

import (
	"context"
	"crypto/sha256"
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

	"go.temporal.io/server/tools/gomadv3/internal/process"
	"go.temporal.io/server/tools/gomadv3/internal/replay"
	"go.temporal.io/server/tools/gomadv3/internal/runner"
	"go.temporal.io/server/tools/gomadv3/internal/target"
)

const usage = `usage:
  gomad explore [flags] exec --provenance FILE -- BINARY [ARG ...]
  gomad explore [flags] go-run PACKAGE -- [ARG ...]
  gomad explore [flags] go-test PACKAGE -- [TEST_BINARY_ARG ...]
  gomad replay [--verify-only] ARTIFACT_DIR
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
	case "replay":
		return runReplay(arguments[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "unknown gomad command %q\n%s", arguments[0], usage)
		return 2
	}
}

func runExplore(arguments []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("gomad explore", flag.ContinueOnError)
	flags.SetOutput(stderr)
	seeds := flags.String("seeds", "1", "seed set or inclusive ranges")
	parallel := flags.Int("parallel", min(runtime.NumCPU(), 8), "maximum active targets")
	runTimeout := flags.Duration("run-timeout", 30*time.Second, "per-seed host deadline")
	overallTimeout := flags.Duration("overall-timeout", 10*time.Minute, "complete exploration host deadline")
	terminateGrace := flags.Duration("terminate-grace", 2*time.Second, "termination grace inside deadlines")
	onFailure := flags.String("on-failure", string(runner.PolicyFirst), "first, budget, or all")
	failureBudget := flags.Uint64("failure-budget", 1, "distinct failure signature threshold")
	artifacts := flags.String("artifacts", ".gomad/artifacts", "artifact root")
	ioProfile := flags.String("io-profile", "", "deterministic application I/O profile")
	outputLimit := byteSize(8 << 20)
	worldLimit := byteSize(64 << 20)
	flags.Var(&outputLimit, "output-limit", "retained bytes per output stream")
	flags.Var(&worldLimit, "world-transition-limit", "World transition capacity")
	var environment stringList
	var buildTags stringList
	flags.Var(&environment, "env", "target NAME=VALUE")
	flags.Var(&buildTags, "build-tag", "validated Go build tag")
	if err := flags.Parse(arguments); err != nil {
		return 2
	}
	parsedTarget, err := parseTarget(flags.Args())
	if err != nil {
		fmt.Fprintf(stderr, "%v\n%s", err, usage)
		return 2
	}
	workingDirectory, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(stderr, "resolve working directory: %v\n", err)
		return 3
	}
	toolchain, executable, runnerBuild, err := localIdentity()
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 3
	}
	config := runner.Config{
		Seeds: *seeds, Parallel: *parallel, RunTimeout: *runTimeout, OverallTimeout: *overallTimeout, TerminateGrace: *terminateGrace,
		OnFailure: runner.FailurePolicy(*onFailure), FailureBudget: *failureBudget, OutputLimit: uint64(outputLimit), WorldTransitionLimit: uint64(worldLimit),
		Artifacts: *artifacts, Environment: environment, IOProfile: *ioProfile, SupervisorCommand: []string{executable, "__supervisor"}, CoordinatorCommand: []string{executable, "__coordinator"}, RunnerBuild: runnerBuild,
		Target: target.Spec{
			Kind: parsedTarget.kind, Source: parsedTarget.source, Provenance: parsedTarget.provenance, Args: parsedTarget.arguments,
			BuildTags: buildTags, WorkingDir: workingDirectory, ToolchainRoot: toolchain,
		},
	}
	summary, err := runner.Run(context.Background(), config)
	if err != nil {
		var hostError *runner.HostError
		if errors.As(err, &hostError) {
			fmt.Fprintln(stderr, err)
			return 3
		}
		fmt.Fprintln(stderr, err)
		return 2
	}
	fmt.Fprintf(stdout, "gomad: attempted=%d succeeded=%d failures=%d distinct=%d stop=%s artifact=%s\n", summary.Attempted, summary.Succeeded, summary.Failures, summary.DistinctFailures, summary.StopReason, summary.BatchPath)
	if summary.Failures != 0 {
		return 1
	}
	return 0
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
	if !result.Match {
		fmt.Fprintf(stdout, "gomad: replay diverged at %s\n", result.Divergence)
		return 1
	}
	if result.Diagnostic {
		fmt.Fprintln(stdout, "gomad: diagnostic watchdog observation recurred")
	} else {
		fmt.Fprintln(stdout, "gomad: target failure reproduced exactly")
	}
	return 1
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
