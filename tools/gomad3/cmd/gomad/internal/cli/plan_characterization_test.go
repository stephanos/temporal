package cli

// Characterization of the grammar gomad plan shares with gomad explore: the
// plan request it builds, the explore validation it inherits in the same
// first-error order, the explore-only and plan-only flags, and the routing of
// its errors and results. Dependencies are built only through
// characterization_support_test.go.

import (
	"bytes"
	"context"
	"errors"
	"io"
	"reflect"
	"runtime"
	"strings"
	"testing"
	"time"

	"go.temporal.io/server/tools/gomad3/record"
	"go.temporal.io/server/tools/gomad3/runner"
	"go.temporal.io/server/tools/gomad3/target"
)

// TestCharacterizePlanRequestDefaults pins the whole campaign a default plan
// freezes: explore's defaults with the failure policy fixed to all.
func TestCharacterizePlanRequestDefaults(t *testing.T) {
	installation := newFakeInstallation()
	var observed []runner.CampaignPlanSpec
	plan := func(_ context.Context, spec runner.CampaignPlanSpec) (runner.CampaignPlanResult, error) {
		if spec.Campaign.Progress == nil {
			t.Fatal("plan request has no progress reporter")
		}
		spec.Campaign.Progress = nil
		observed = append(observed, spec)
		return runner.CampaignPlanResult{Path: "/plan"}, nil
	}
	dependencies := installation.exploreDependencies(nil, plan)
	if got := runCommand(func(stdout, stderr *bytes.Buffer) int {
		return runPlanWith([]string{"--output=relative/plan.json", "go-run", "./cmd"}, stdout, stderr, dependencies)
	}); got.status != 0 || got.stderr != "" {
		t.Fatalf("default plan = %s", got)
	}
	want := runner.CampaignPlanSpec{
		Output: "relative/plan.json",
		Campaign: runner.CampaignSpec{
			Strategy: runner.StrategySeed, Seeds: "1", Parallel: min(runtime.NumCPU(), 8), ExecutionTimeout: 30 * time.Second, OverallTimeout: 10 * time.Minute, TerminateGrace: 2 * time.Second,
			OnFailure: runner.PolicyAll, FailureBudget: 1, OutputLimit: 8 << 20, WorldTransitionLimit: 64 << 20, ClockTick: record.ClockTickStrict, IOTranscriptLimit: 64 << 20,
			Artifacts: ".gomad/artifacts", SupervisorCommand: []string{"/bin/gomad", "__supervisor"}, CoordinatorCommand: []string{"/bin/gomad", "__coordinator"}, RunnerBuild: "sha256:runner",
			Coverage: runner.CoverageNone, KeepSuccesses: runner.KeepSuccessesNone, ProgressInterval: 5 * time.Second,
			Target: target.Spec{Kind: target.KindGoRun, Source: "./cmd", WorkingDir: "/workspace", ToolchainRoot: "/toolchain", CapabilityMode: target.CapabilityModeClosure},
		},
	}
	if len(observed) != 1 || !reflect.DeepEqual(observed[0], want) {
		t.Fatalf("default plan request =\n%#v\nwant\n%#v", observed, want)
	}
	if !reflect.DeepEqual(*installation.requested, []string{""}) {
		t.Fatalf("requested toolchain roots = %q", *installation.requested)
	}

	// The CLI forwards every explore option to the plan operation and leaves
	// the portable-plan restrictions (seed strategy, on-failure=all) to it.
	observed = nil
	arguments := []string{
		"--strategy=choice-exploration", "--seeds=5", "--max-executions=2", "--max-choice-depth=3", "--max-exploration-bytes=1MiB", "--choice-start-ordinal=4",
		"--guide-regression=false", "--output=/plans/choice.json", "--json", "exec", "--provenance", "/target.provenance.json", "--", "/target", "--flag",
	}
	if got := runCommand(func(stdout, stderr *bytes.Buffer) int {
		return runPlanWith(arguments, stdout, stderr, dependencies)
	}); got.status != 0 || got.stderr != "" {
		t.Fatalf("choice-exploration plan = %s", got)
	}
	if len(observed) != 1 {
		t.Fatalf("choice-exploration plan requests = %d", len(observed))
	}
	campaign := observed[0].Campaign
	if observed[0].Output != "/plans/choice.json" || campaign.Strategy != runner.StrategyChoiceExploration || campaign.OnFailure != runner.PolicyAll || campaign.Seeds != "5" ||
		campaign.MaxExecutions != 2 || campaign.MaxChoiceDepth != 3 || campaign.MaxExplorationBytes != 1<<20 || campaign.ChoiceStartOrdinal != 4 || campaign.ChoiceTraceLimit != 8<<20 ||
		!reflect.DeepEqual(campaign.Target, target.Spec{Kind: target.KindExec, Source: "/target", Provenance: "/target.provenance.json", Args: []string{"--flag"}, WorkingDir: "/workspace", ToolchainRoot: "/toolchain", CapabilityMode: target.CapabilityModeClosure}) {
		t.Fatalf("choice-exploration plan request = %#v", observed[0])
	}
}

// TestCharacterizePlanInputRejection pins that plan reports explore's input
// errors with the same messages and in the same first-error order, before it
// resolves the installation and before it requires --output.
func TestCharacterizePlanInputRejection(t *testing.T) {
	t.Setenv("GOMAD3_TOOLCHAIN_DIR", "")
	invalid := func(message string) commandResult {
		return commandResult{status: 2, stderr: "gomad: invalid_input: " + message + "\n"}
	}
	for _, test := range []struct {
		name      string
		arguments []string
		want      commandResult
	}{
		{"unknown strategy", []string{"--strategy=random", "go-run", "./cmd"}, invalid(`unknown exploration strategy "random"`)},
		{"unknown strategy precedes count", []string{"--strategy=random", "--count=0", "go-run", "./cmd"}, invalid(`unknown exploration strategy "random"`)},
		{"capability mode precedes strategy", []string{"--capability-mode=open", "--strategy=random", "go-run", "./cmd"}, invalid(`unknown capability mode "open"`)},
		{"zero count", []string{"--count=0", "go-run", "./cmd"}, invalid("--count must be greater than zero")},
		{"seed strategy explicit zero bound", []string{"--max-executions=0", "go-run", "./cmd"}, invalid("exploration bounds require --strategy=choice-exploration")},
		{"choice exploration seed range", []string{"--strategy=choice-exploration", "--seeds=1-2", "go-run", "./cmd"}, invalid("--strategy=choice-exploration requires exactly one base seed")},
		{"malformed seeds", []string{"--strategy=choice-exploration", "--seeds=x", "go-run", "./cmd"}, invalid(`invalid seed selection term "x": non-decimal seed`)},
		{"guide regression without guide", []string{"--guide-regression", "go-run", "./cmd"}, invalid("--guide-regression requires --guide")},
		{"guide without corpus", []string{"--guide", "go-run", "./cmd"}, invalid("--guide requires --corpus DIR")},
		{"guide with explicit empty coverage", []string{"--guide", "--corpus=/corpus", "--coverage=", "go-run", "./cmd"}, invalid("--guide requires semantic or choice coverage")},
		{"explicit empty coverage", []string{"--coverage=", "go-run", "./cmd"}, invalid(`unknown coverage mode ""`)},
		{"unknown coverage", []string{"--coverage=all", "--require-probe=stdlib.os.openfile", "go-run", "./cmd"}, invalid(`unknown coverage mode "all"`)},
		{"probe without semantic coverage", []string{"--coverage=choice", "--require-probe=stdlib.os.openfile", "go-run", "./cmd"}, invalid("--require-probe requires --coverage=semantic")},
		{"unknown probe", []string{"--coverage=semantic", "--require-probe=unknown.probe", "go-run", "./cmd"}, invalid(`unknown required semantic probe "unknown.probe"`)},
		{"diagnostics with exploration", []string{"--strategy=choice-exploration", "--max-executions=1", "--max-choice-depth=1", "--max-exploration-bytes=1MiB", "--diagnostics", "go-run", "./cmd"}, invalid("--diagnostics requires the seed strategy; forced-prefix exploration is unsupported")},
		{"choice bytes below minimum", []string{"--choices", "--choice-bytes=1", "go-run", "./cmd"}, invalid("--choice-bytes must be between 160 bytes and 64MiB")},
		{"choice coverage without choices", []string{"--coverage=semantic+choice", "go-run", "./cmd"}, invalid("--coverage=semantic+choice requires --choices")},
		{"choice coverage precedes target", []string{"--coverage=choice"}, invalid("--coverage=choice requires --choices")},
		{"missing target", nil, invalid("target kind is required")},
		{"target arguments without separator", []string{"go-run", "./cmd", "extra"}, invalid("go-run target arguments require -- separator")},
		{"relative working directory", []string{"--working-dir=module", "go-run", "./cmd"}, invalid(`working directory "module" must be an absolute, clean path`)},
	} {
		t.Run(test.name, func(t *testing.T) {
			arguments := append([]string{"plan", "--toolchain-root=relative", "--output=/plan"}, test.arguments...)
			if got := runGomad(arguments...); got != test.want {
				t.Fatalf("gomad %s:\n%s\nwant:\n%s", strings.Join(arguments, " "), got, test.want)
			}
		})
	}
}

// TestCharacterizePlanFlagErrors pins flag-parse failures of plan in text and
// JSON, and that an explicit --on-failure follows plan's fixed policy only
// when it parses.
func TestCharacterizePlanFlagErrors(t *testing.T) {
	t.Setenv("GOMAD3_TOOLCHAIN_DIR", "")
	unknown := runGomad("plan", "--unknown", "go-run", "./cmd")
	if unknown.status != 2 || unknown.stdout != "" || !strings.HasPrefix(unknown.stderr, "gomad: invalid_input: flag provided but not defined: -unknown\nUsage of gomad explore:\n") {
		t.Fatalf("plan unknown flag = %s", unknown)
	}
	if got, want := runGomad("plan", "--json", "--parallel=many", "go-run", "./cmd"), (commandResult{status: 2, stdout: `{"schema":"gomad3.explore-event/v3","type":"error","classification":"invalid_input","message":"invalid value \"many\" for flag -parallel: parse error"}` + "\n"}); got != want {
		t.Fatalf("plan JSON flag error = %s, want %s", got, want)
	}
	if got, want := runGomad("plan", "--json", "--toolchain-root=relative", "--output=/plan", "--count=0", "go-run", "./cmd"), (commandResult{status: 2, stdout: `{"schema":"gomad3.explore-event/v3","type":"error","classification":"invalid_input","message":"--count must be greater than zero"}` + "\n"}); got != want {
		t.Fatalf("plan JSON input error = %s, want %s", got, want)
	}
}

// TestCharacterizePlanOperationErrors pins how plan classifies the errors of
// its operation.
func TestCharacterizePlanOperationErrors(t *testing.T) {
	for _, test := range []struct {
		name string
		err  error
		json bool
		want commandResult
	}{
		{"portable restriction", errors.New("portable campaign plans require a seed campaign with on-failure=all"), false, commandResult{status: 2, stderr: "gomad: invalid_input: portable campaign plans require a seed campaign with on-failure=all\n"}},
		{"runner failure", &runner.HostError{Reason: "plan", Err: errors.New("write")}, false, commandResult{status: 3, stderr: "gomad: runner_failure: " + (&runner.HostError{Reason: "plan", Err: errors.New("write")}).Error() + "\n"}},
		{"cancelled JSON", &runner.HostError{Reason: "cancelled"}, true, commandResult{status: 3, stdout: `{"schema":"gomad3.explore-event/v3","type":"error","classification":"cancelled","message":"gomad3 Runner/host failure: cancelled"}` + "\n"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			dependencies := newFakeInstallation().exploreDependencies(nil, func(context.Context, runner.CampaignPlanSpec) (runner.CampaignPlanResult, error) {
				return runner.CampaignPlanResult{}, test.err
			})
			arguments := []string{"--output=/plan", "go-run", "./cmd"}
			if test.json {
				arguments = append([]string{"--json"}, arguments...)
			}
			if got := runCommand(func(stdout, stderr *bytes.Buffer) int { return runPlanWith(arguments, stdout, stderr, dependencies) }); got != test.want {
				t.Fatalf("plan:\n%s\nwant:\n%s", got, test.want)
			}
		})
	}
	// Plan reports a missing --output after installation and working-directory
	// resolution, as invalid input in JSON too.
	if got, want := runCommand(func(stdout, stderr *bytes.Buffer) int {
		return runPlanWith([]string{"--json", "go-run", "./cmd"}, stdout, stderr, newFakeInstallation().exploreDependencies(nil, nil))
	}), (commandResult{status: 2, stdout: `{"schema":"gomad3.explore-event/v3","type":"error","classification":"invalid_input","message":"gomad plan requires --output FILE"}` + "\n"}); got != want {
		t.Fatalf("plan JSON without output = %s, want %s", got, want)
	}
	if got, want := runCommand(func(stdout, stderr *bytes.Buffer) int {
		return runPlanWith([]string{"--working-dir=module", "go-run", "./cmd"}, stdout, stderr, newFakeInstallation().exploreDependencies(nil, nil))
	}), (commandResult{status: 2, stderr: "gomad: invalid_input: working directory \"module\" must be an absolute, clean path\n"}); got != want {
		t.Fatalf("plan relative working directory without output = %s, want %s", got, want)
	}
}

// TestCharacterizeExploreErrorWriteRouting pins where explore and plan report
// a failure to write a JSON error event: most input errors also print the
// write failure on stderr, while the diagnostics-strategy rejection does not.
func TestCharacterizeExploreErrorWriteRouting(t *testing.T) {
	dependencies := newFakeInstallation().exploreDependencies(nil, nil)
	for _, test := range []struct {
		name      string
		command   func(stdout, stderr io.Writer) int
		wantError bool
	}{
		{"explore count", func(stdout, stderr io.Writer) int {
			return runExploreWith([]string{"--json", "--count=0", "go-run", "./cmd"}, stdout, stderr, dependencies)
		}, true},
		{"plan count", func(stdout, stderr io.Writer) int {
			return runPlanWith([]string{"--json", "--count=0", "go-run", "./cmd"}, stdout, stderr, dependencies)
		}, true},
		{"explore diagnostics", func(stdout, stderr io.Writer) int {
			return runExploreWith([]string{"--json", "--strategy=choice-exploration", "--max-executions=1", "--max-choice-depth=1", "--max-exploration-bytes=1MiB", "--diagnostics", "go-run", "./cmd"}, stdout, stderr, dependencies)
		}, false},
		{"plan missing output", func(stdout, stderr io.Writer) int {
			return runPlanWith([]string{"--json", "go-run", "./cmd"}, stdout, stderr, dependencies)
		}, true},
		{"explore output", func(stdout, stderr io.Writer) int {
			return runExploreWith([]string{"--json", "--output=/plan", "go-run", "./cmd"}, stdout, stderr, dependencies)
		}, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			var stderr bytes.Buffer
			if status := test.command(failingWriter{}, &stderr); status != 3 || (stderr.Len() != 0) != test.wantError {
				t.Fatalf("status = %d, stderr = %q, want status 3 and stderr written = %t", status, stderr.String(), test.wantError)
			}
		})
	}
}

// TestCharacterizeHiddenPlanFlag pins the hidden --__plan route: explore
// accepts it and then creates a plan instead of exploring.
func TestCharacterizeHiddenPlanFlag(t *testing.T) {
	t.Setenv("GOMAD3_TOOLCHAIN_DIR", "")
	if got, want := runGomad("explore", "--toolchain-root=relative", "--__plan", "go-run", "./cmd"), (commandResult{status: 3, stderr: "gomad: runner_failure: resolve Gomad installation: CLI --toolchain-root toolchain root must be an absolute non-root clean path: \"relative\"\n"}); got != want {
		t.Fatalf("explore --__plan = %s, want %s", got, want)
	}
	var planned []runner.CampaignPlanSpec
	dependencies := newFakeInstallation().exploreDependencies(nil, func(_ context.Context, spec runner.CampaignPlanSpec) (runner.CampaignPlanResult, error) {
		planned = append(planned, spec)
		return runner.CampaignPlanResult{Path: "/plan"}, nil
	})
	if got := runCommand(func(stdout, stderr *bytes.Buffer) int {
		return runExploreWith([]string{"--__plan", "--output=/plan", "go-run", "./cmd"}, stdout, stderr, dependencies)
	}); got.status != 0 || len(planned) != 1 || planned[0].Campaign.OnFailure != runner.PolicyFirst {
		t.Fatalf("explore --__plan = %s, requests %#v", got, planned)
	}
	if got := runCommand(func(stdout, stderr *bytes.Buffer) int {
		return runPlanWith([]string{"--__plan", "--output=/plan", "go-run", "./cmd"}, stdout, stderr, dependencies)
	}); got.status != 0 || len(planned) != 2 {
		t.Fatalf("plan --__plan = %s", got)
	}
}
