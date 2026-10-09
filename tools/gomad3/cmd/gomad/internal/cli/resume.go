package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"time"

	"go.temporal.io/server/tools/gomad3/runner"
)

type resumeDependencies struct {
	install func(string) (installation, error)
	run     func(context.Context, runner.ResumeSpec) (runner.CampaignResult, error)
}

func (app application) runResume(arguments []string, stdout, stderr io.Writer) int {
	return runResumeWith(arguments, stdout, stderr, resumeDependencies{install: app.install, run: runner.Resume})
}

func runResumeWith(arguments []string, stdout, stderr io.Writer, dependencies resumeDependencies) int {
	flags := flag.NewFlagSet("gomad resume", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	guideRegression := flags.Bool("guide-regression", false, "require the recorded guidance regression mode")
	jsonOutput := flags.Bool("json", false, "emit stable JSON events")
	toolchainRoot := flags.String("toolchain-root", "", "absolute pinned toolchain root")
	if err := flags.Parse(arguments); err != nil {
		reporter := newExploreReporter(*jsonOutput, stdout, stderr)
		if writeErr := reporter.Error("invalid_input", err); writeErr != nil {
			if _, printErr := fmt.Fprintln(stderr, writeErr); printErr != nil {
				return 3
			}
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
		if err := reporter.Error("invalid_input", errors.New("resume requires one interrupted campaign path")); err != nil {
			if _, printErr := fmt.Fprintln(stderr, err); printErr != nil {
				return 3
			}
			return 3
		}
		return 2
	}
	installed, err := dependencies.install(*toolchainRoot)
	if err != nil {
		if writeErr := reporter.Error("runner_failure", err); writeErr != nil {
			if _, printErr := fmt.Fprintln(stderr, writeErr); printErr != nil {
				return 3
			}
		}
		return 3
	}
	var regressionOverride *bool
	flags.Visit(func(value *flag.Flag) {
		if value.Name == "guide-regression" {
			regressionOverride = guideRegression
		}
	})
	summary, err := dependencies.run(context.Background(), runner.ResumeSpec{
		CampaignPath: flags.Arg(0), GuideRegression: regressionOverride, RunnerBuild: installed.runnerBuild, ToolchainRoot: installed.toolchainRoot,
		SupervisorCommand: installed.supervisorCommand(), CoordinatorCommand: installed.coordinatorCommand(),
		Progress: reporter.Progress, ProgressInterval: 5 * time.Second,
	})
	if err != nil {
		classification := classifyResumeError(err)
		if writeErr := reporter.Error(classification, err); writeErr != nil {
			if _, printErr := fmt.Fprintln(stderr, writeErr); printErr != nil {
				return 3
			}
			return 3
		}
		return exploreErrorStatus(classification)
	}
	if err := reporter.Result(summary); err != nil {
		if _, printErr := fmt.Fprintln(stderr, err); printErr != nil {
			return 3
		}
		return 3
	}
	return exploreSummaryStatus(summary)
}

func classifyResumeError(err error) string {
	var hostError *runner.HostError
	if errors.As(err, &hostError) && hostError.Reason == "resume_setup" {
		return "invalid_input"
	}
	return classifyExploreError(err)
}
