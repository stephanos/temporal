package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"time"

	"go.temporal.io/server/tools/gomadv3/internal/runner"
	"go.temporal.io/server/tools/gomadv3/internal/target"
)

type resumeDependencies struct {
	identity func(string) (string, string, string, error)
	run      func(context.Context, runner.Config) (runner.Summary, error)
}

func runResume(arguments []string, stdout, stderr io.Writer) int {
	return runResumeWith(arguments, stdout, stderr, resumeDependencies{identity: localIdentity, run: runner.Run})
}

func runResumeWith(arguments []string, stdout, stderr io.Writer, dependencies resumeDependencies) int {
	flags := flag.NewFlagSet("gomad resume", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	jsonOutput := flags.Bool("json", false, "emit stable JSON events")
	toolchainRoot := flags.String("toolchain-root", "", "absolute pinned toolchain root")
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
	if flags.NArg() != 1 || flags.Arg(0) == "" {
		if err := reporter.Error("invalid_input", errors.New("resume requires one interrupted batch path")); err != nil {
			fmt.Fprintln(stderr, err)
			return 3
		}
		return 2
	}
	toolchain, executable, runnerBuild, err := dependencies.identity(*toolchainRoot)
	if err != nil {
		if writeErr := reporter.Error("runner_failure", err); writeErr != nil {
			fmt.Fprintln(stderr, writeErr)
		}
		return 3
	}
	summary, err := dependencies.run(context.Background(), runner.Config{
		ResumeBatch: flags.Arg(0), RunnerBuild: runnerBuild, Target: target.Spec{ToolchainRoot: toolchain},
		SupervisorCommand: []string{executable, "__supervisor"}, CoordinatorCommand: []string{executable, "__coordinator"},
		Progress: reporter.Progress, ProgressInterval: 5 * time.Second,
	})
	if err != nil {
		classification := classifyResumeError(err)
		if writeErr := reporter.Error(classification, err); writeErr != nil {
			fmt.Fprintln(stderr, writeErr)
			return 3
		}
		if classification == "invalid_input" || classification == "unsupported_target" {
			return 2
		}
		if classification == "semantic_coverage_failure" {
			return 1
		}
		return 3
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

func classifyResumeError(err error) string {
	var hostError *runner.HostError
	if errors.As(err, &hostError) && hostError.Reason == "resume_setup" {
		return "invalid_input"
	}
	return classifyExploreError(err)
}
