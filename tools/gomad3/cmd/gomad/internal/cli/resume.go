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
	identity func(string) (string, string, string, error)
	commands func(string) privateCommands
	run      func(context.Context, runner.ResumeSpec) (runner.CampaignResult, error)
}

func runResume(arguments []string, stdout, stderr io.Writer) int {
	return runResumeWithApplication(arguments, stdout, stderr, newApplication())
}

func runResumeWithApplication(arguments []string, stdout, stderr io.Writer, app *application) int {
	return runResumeWith(arguments, stdout, stderr, resumeDependencies{identity: app.identity, commands: app.commands, run: runner.Resume})
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
		if err := reporter.Error("invalid_input", errors.New("resume requires one interrupted campaign path")); err != nil {
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
	var regressionOverride *bool
	flags.Visit(func(value *flag.Flag) {
		if value.Name == "guide-regression" {
			regressionOverride = guideRegression
		}
	})
	commandsFor := dependencies.commands
	if commandsFor == nil {
		commandsFor = privateCommandsFor
	}
	commands := commandsFor(executable)
	summary, err := dependencies.run(context.Background(), runner.ResumeSpec{
		CampaignPath: flags.Arg(0), GuideRegression: regressionOverride, RunnerBuild: runnerBuild, ToolchainRoot: toolchain,
		SupervisorCommand: commands.supervisor, CoordinatorCommand: commands.coordinator,
		Progress: reporter.Progress, ProgressInterval: 5 * time.Second,
	})
	if err != nil {
		classification := classifyResumeError(err)
		if writeErr := reporter.Error(classification, err); writeErr != nil {
			fmt.Fprintln(stderr, writeErr)
			return 3
		}
		return exploreErrorStatus(classification)
	}
	if err := reporter.Result(summary); err != nil {
		fmt.Fprintln(stderr, err)
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
