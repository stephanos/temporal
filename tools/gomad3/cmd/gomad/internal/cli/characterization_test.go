package cli

// Characterization of the gomad command grammar, defaults, output routing and
// exit statuses. These tests pin observable CLI behavior before application
// construction is refactored, and must keep passing unchanged afterwards.
// Dependencies are built only through characterization_support_test.go.

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"go.temporal.io/server/tools/gomad3/artifact"
	"go.temporal.io/server/tools/gomad3/deterministicio"
	"go.temporal.io/server/tools/gomad3/qualification"
	"go.temporal.io/server/tools/gomad3/record"
	"go.temporal.io/server/tools/gomad3/runner"
	"go.temporal.io/server/tools/gomad3/target"
)

type commandResult struct {
	status int
	stdout string
	stderr string
}

func (result commandResult) String() string {
	return fmt.Sprintf("status=%d\n--- stdout\n%s--- stderr\n%s", result.status, result.stdout, result.stderr)
}

func runCommand(command func(stdout, stderr *bytes.Buffer) int) commandResult {
	var stdout, stderr bytes.Buffer
	status := command(&stdout, &stderr)
	return commandResult{status: status, stdout: stdout.String(), stderr: stderr.String()}
}

func runGomad(arguments ...string) commandResult {
	return runCommand(func(stdout, stderr *bytes.Buffer) int { return Run(arguments, stdout, stderr) })
}

func TestCharacterizeHelpAndUsage(t *testing.T) {
	t.Setenv("GOMAD3_TOOLCHAIN_DIR", "")
	for _, command := range []string{"", "plan", "execute-shard", "merge", "explore", "qualify", "qualify-set", "merge-set", "compare-support", "analyze", "resume", "recover", "replay", "minimize", "doctor", "inspect"} {
		name := command
		arguments := []string{command, "-h"}
		if command == "" {
			name, arguments = "usage", nil
		}
		t.Run(name, func(t *testing.T) {
			golden, err := os.ReadFile(filepath.Join("testdata", "characterization", "help-"+name+".txt"))
			if err != nil {
				t.Fatal(err)
			}
			want := strings.ReplaceAll(string(golden), "{{parallel}}", strconv.Itoa(min(runtime.NumCPU(), 8)))
			if got := runGomad(arguments...).String(); got != want {
				t.Fatalf("gomad %s:\n%s\nwant:\n%s", strings.Join(arguments, " "), got, want)
			}
		})
	}
}

func TestCharacterizeUnknownCommandsAndPrivateModes(t *testing.T) {
	if got, want := runGomad("unknown"), (commandResult{status: 2, stderr: "unknown gomad command \"unknown\"\n" + usage}); got != want {
		t.Fatalf("unknown command = %s, want %s", got, want)
	}
	if got, want := runGomad("__unknown"), (commandResult{status: 2, stderr: "unknown gomad command \"__unknown\"\n" + usage}); got != want {
		t.Fatalf("unknown private mode = %s, want %s", got, want)
	}
	// A private child mode reads its request from the process's standard
	// input, not from the command's writers, and reports a failure on stderr.
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	original := os.Stdin
	os.Stdin = reader
	defer func() {
		os.Stdin = original
		reader.Close()
	}()
	if got, want := runGomad("__coordinator"), (commandResult{status: 3, stderr: "decode coordinator request: EOF\n"}); got != want {
		t.Fatalf("coordinator with empty request = %s, want %s", got, want)
	}
}

// TestCharacterizeInputRejection pins malformed input, explicit zero values and
// flags irrelevant to the selected mode. Every case also names an unusable
// --toolchain-root, so it pins that input validation precedes installation
// resolution.
func TestCharacterizeInputRejection(t *testing.T) {
	t.Setenv("GOMAD3_TOOLCHAIN_DIR", "")
	invalid := func(message string) commandResult {
		return commandResult{status: 2, stderr: "gomad: invalid_input: " + message + "\n"}
	}
	plain := func(message string) commandResult {
		return commandResult{status: 2, stderr: message + "\n"}
	}
	simulationBounds := []string{"--strategy=simulation-exploration", "--max-executions=1", "--max-forced-decisions=1", "--max-exploration-bytes=1MiB", "--max-exploration-result-bytes=1MiB", "--max-runtime-decisions=1", "--max-scenario-decisions=1", "--max-network-decisions=1", "--max-storage-decisions=1", "--max-fault-decisions=1", "--max-crash-decisions=1"}
	withBounds := func(extra ...string) []string {
		return append(append(append([]string{"explore"}, simulationBounds...), extra...), "go-run", "./cmd")
	}
	for _, test := range []struct {
		name      string
		arguments []string
		want      commandResult
	}{
		{"explore zero count", []string{"explore", "--count=0", "go-run", "./cmd"}, invalid("--count must be greater than zero")},
		{"explore count and seeds", []string{"explore", "--count=2", "--seeds=1", "go-run", "./cmd"}, invalid("--count and --seeds are mutually exclusive")},
		{"explore unknown strategy", []string{"explore", "--strategy=random", "go-run", "./cmd"}, invalid(`unknown exploration strategy "random"`)},
		{"seed strategy explicit zero choice bound", []string{"explore", "--max-executions=0", "go-run", "./cmd"}, invalid("exploration bounds require --strategy=choice-exploration")},
		{"seed strategy explicit zero simulation bound", []string{"explore", "--max-crash-decisions=0", "go-run", "./cmd"}, invalid("simulation exploration bounds require --strategy=simulation-exploration")},
		{"seed strategy explicit zero start ordinal", []string{"explore", "--choice-start-ordinal=0", "go-run", "./cmd"}, invalid("--choice-start-ordinal requires --strategy=choice-exploration")},
		{"choice exploration explicit zero executions", []string{"explore", "--strategy=choice-exploration", "--max-executions=0", "--max-choice-depth=1", "--max-exploration-bytes=1MiB", "go-run", "./cmd"}, invalid("--strategy=choice-exploration requires an explicit positive --max-executions")},
		{"choice exploration missing depth", []string{"explore", "--strategy=choice-exploration", "--max-executions=1", "--max-exploration-bytes=1MiB", "go-run", "./cmd"}, invalid("--strategy=choice-exploration requires an explicit positive --max-choice-depth")},
		{"choice exploration count", []string{"explore", "--strategy=choice-exploration", "--count=1", "go-run", "./cmd"}, invalid("--strategy=choice-exploration does not accept --count")},
		{"choice exploration seed range", []string{"explore", "--strategy=choice-exploration", "--seeds=1-2", "go-run", "./cmd"}, invalid("--strategy=choice-exploration requires exactly one base seed")},
		{"choice exploration guide", []string{"explore", "--strategy=choice-exploration", "--guide", "go-run", "./cmd"}, invalid("--strategy=choice-exploration does not support --guide")},
		{"choice exploration simulation bound", []string{"explore", "--strategy=choice-exploration", "--max-forced-decisions=1", "go-run", "./cmd"}, invalid("simulation exploration bounds require --strategy=simulation-exploration")},
		{"simulation exploration explicit zero bound", withBounds("--max-crash-decisions=0"), invalid("--strategy=simulation-exploration requires an explicit positive --max-crash-decisions")},
		{"simulation exploration choice depth", withBounds("--max-choice-depth=1"), invalid("--strategy=simulation-exploration does not accept --max-choice-depth")},
		{"simulation exploration start ordinal", withBounds("--choice-start-ordinal=0"), invalid("--strategy=simulation-exploration does not accept --choice-start-ordinal")},
		{"simulation exploration count", withBounds("--count=1"), invalid("--strategy=simulation-exploration does not accept --count")},
		{"simulation exploration diagnostics", withBounds("--diagnostics"), invalid("--diagnostics requires the seed strategy; forced-prefix exploration is unsupported")},
		{"choice bytes without choices", []string{"explore", "--choice-bytes=1MiB", "go-run", "./cmd"}, invalid("--choice-bytes requires --choices")},
		{"choice bytes above maximum", []string{"explore", "--choices", "--choice-bytes=65MiB", "go-run", "./cmd"}, invalid(fmt.Sprintf("--choice-bytes must be between %d bytes and 64MiB", runner.MinimumChoiceTraceBytes))},
		{"choice coverage without choices", []string{"explore", "--coverage=choice", "go-run", "./cmd"}, invalid("--coverage=choice requires --choices")},
		{"unknown coverage", []string{"explore", "--coverage=all", "go-run", "./cmd"}, invalid(`unknown coverage mode "all"`)},
		{"probe without semantic coverage", []string{"explore", "--require-probe=stdlib.os.openfile", "go-run", "./cmd"}, invalid("--require-probe requires --coverage=semantic")},
		{"corpus without guide", []string{"explore", "--corpus=/corpus", "go-run", "./cmd"}, invalid("--corpus requires --guide")},
		{"guide without corpus", []string{"explore", "--guide", "go-run", "./cmd"}, invalid("--guide requires --corpus DIR")},
		{"guide regression without guide", []string{"explore", "--guide-regression", "go-run", "./cmd"}, invalid("--guide-regression requires --guide")},
		{"unknown capability mode", []string{"explore", "--capability-mode=open", "go-run", "./cmd"}, invalid(`unknown capability mode "open"`)},
		{"explore output", []string{"explore", "--output=/plan", "go-run", "./cmd"}, invalid("--output is only valid with gomad plan")},
		{"missing target", []string{"explore"}, invalid("target kind is required")},
		{"unknown target kind", []string{"explore", "go-build", "./cmd"}, invalid(`unknown target kind "go-build"`)},
		{"exec without provenance", []string{"explore", "exec", "--", "./target"}, invalid("exec requires --provenance FILE -- BINARY [ARG ...]")},
		{"go-run without package", []string{"explore", "go-run"}, invalid("go-run requires one package")},
		{"go-test arguments without separator", []string{"explore", "go-test", "./pkg", "-test.run=X"}, invalid("go-test target arguments require -- separator")},
		{"relative working directory", []string{"explore", "--working-dir=module", "go-run", "./cmd"}, invalid(`working directory "module" must be an absolute, clean path`)},
		{"explore JSON input error", []string{"explore", "--json", "--count=0", "go-run", "./cmd"}, commandResult{status: 2, stdout: `{"schema":"gomad3.explore-event/v3","type":"error","classification":"invalid_input","message":"--count must be greater than zero"}` + "\n"}},
		{"plan inherits explore validation", []string{"plan", "--output=/plan", "--count=0", "go-run", "./cmd"}, invalid("--count must be greater than zero")},
		{"qualify repeat below two", []string{"qualify", "--repeat=1", "go-run", "./cmd"}, invalid("--repeat must be between 2 and 32")},
		{"qualify repeat above maximum", []string{"qualify", "--repeat=33", "go-run", "./cmd"}, invalid("--repeat must be between 2 and 32")},
		{"qualify successful replay without bounds", []string{"qualify", "--replay-successes", "go-run", "./cmd"}, invalid("--replay-successes requires explicit --success-limit and --success-bytes bounds")},
		{"qualify success bounds without replay", []string{"qualify", "--success-limit=1", "go-run", "./cmd"}, invalid("--success-limit and --success-bytes require --replay-successes")},
		{"qualify choice bytes without choices", []string{"qualify", "--choice-bytes=1MiB", "go-run", "./cmd"}, invalid("--choice-bytes requires --choices")},
		{"qualify missing target", []string{"qualify"}, invalid("target kind is required")},
		{"analyze format", []string{"analyze", "--format=xml", "go-run", "./cmd"}, plain(`invalid analysis format "xml"`)},
		{"analyze timeout above maximum", []string{"analyze", "--timeout=31m", "go-run", "./cmd"}, plain("analysis timeout must be non-negative and no greater than 30m0s")},
		{"analyze negative timeout", []string{"analyze", "--timeout=-1s", "go-run", "./cmd"}, plain("analysis timeout must be non-negative and no greater than 30m0s")},
		{"analyze exec target", []string{"analyze", "exec", "--provenance", "/p", "--", "./target"}, plain("gomad analyze requires a go-run or go-test target")},
		{"analyze capability mode", []string{"analyze", "--capability-mode=open", "go-run", "./cmd"}, plain(`unknown capability mode "open"`)},
		{"minimize explicit zero budget", []string{"minimize", "--attempt-budget=0", "/artifact"}, commandResult{status: 2, stderr: usage}},
		{"minimize missing artifact", []string{"minimize"}, commandResult{status: 2, stderr: usage}},
		{"replay missing artifact", []string{"replay"}, commandResult{status: 2, stderr: usage}},
		{"execute-shard missing plan", []string{"execute-shard", "--shard=0/1"}, commandResult{status: 2, stderr: usage}},
		{"execute-shard malformed shard", []string{"execute-shard", "--shard=1", "/plan"}, plain(`invalid shard "1": want zero-based INDEX/COUNT`)},
		{"execute-shard index outside count", []string{"execute-shard", "--shard=1/1", "/plan"}, plain("campaign shard 1/1 is invalid")},
		{"resume missing campaign", []string{"resume"}, invalid("resume requires one interrupted campaign path")},
		{"doctor positional argument", []string{"doctor", "extra"}, commandResult{status: 2, stderr: usage}},
		{"merge missing output", []string{"merge", "/plan", "/shard"}, commandResult{status: 2, stderr: usage}},
		{"merge missing shard", []string{"merge", "--output=/merged", "/plan"}, commandResult{status: 2, stderr: usage}},
		{"recover missing campaign", []string{"recover"}, commandResult{status: 2, stderr: usage}},
		{"inspect missing artifact", []string{"inspect"}, commandResult{status: 2, stderr: usage}},
		{"qualify-set missing manifest", []string{"qualify-set", "--working-dir=/repo"}, plain("qualify-set requires --manifest, --working-dir, and --format=text|json")},
		{"qualify-set format", []string{"qualify-set", "--manifest=/m", "--working-dir=/repo", "--format=xml"}, plain("qualify-set requires --manifest, --working-dir, and --format=text|json")},
		{"qualify-set positional argument", []string{"qualify-set", "--manifest=/m", "--working-dir=/repo", "extra"}, commandResult{status: 2}},
		{"qualify-set malformed shard", []string{"qualify-set", "--manifest=/m", "--working-dir=/repo", "--shard=0"}, plain(`invalid shard "0": want zero-based INDEX/COUNT`)},
		{"merge-set missing shard reports", []string{"merge-set", "--manifest=/m"}, plain("merge-set requires --manifest, --format=text|json, and at least one shard report")},
		{"compare-support missing candidate", []string{"compare-support", "--baseline=/b"}, plain("compare-support requires --baseline, --candidate, and --format=text|json")},
	} {
		t.Run(test.name, func(t *testing.T) {
			arguments := test.arguments
			switch arguments[0] {
			case "explore", "plan", "qualify", "analyze", "minimize", "replay", "execute-shard", "resume", "doctor":
				arguments = append([]string{arguments[0], "--toolchain-root=relative"}, arguments[1:]...)
			}
			if got := runGomad(arguments...); got != test.want {
				t.Fatalf("gomad %s:\n%s\nwant:\n%s", strings.Join(arguments, " "), got, test.want)
			}
		})
	}
}

func TestCharacterizeByteSizeAndFlagValueRejection(t *testing.T) {
	for _, test := range []struct {
		arguments []string
		want      commandResult
	}{
		{[]string{"explore", "--json", "--output-limit=0", "go-run", "./cmd"}, commandResult{status: 2, stdout: `{"schema":"gomad3.explore-event/v3","type":"error","classification":"invalid_input","message":"invalid value \"0\" for flag -output-limit: invalid byte size \"0\""}` + "\n"}},
		{[]string{"explore", "--json", "--parallel=many", "go-run", "./cmd"}, commandResult{status: 2, stdout: `{"schema":"gomad3.explore-event/v3","type":"error","classification":"invalid_input","message":"invalid value \"many\" for flag -parallel: parse error"}` + "\n"}},
		{[]string{"qualify", "--json", "--seed=-1", "go-run", "./cmd"}, commandResult{status: 2, stdout: `{"schema":"gomad3.qualify-event/v1","type":"error","classification":"invalid_input","message":"invalid value \"-1\" for flag -seed: parse error"}` + "\n"}},
		{[]string{"resume", "--json", "--unknown"}, commandResult{status: 2, stdout: `{"schema":"gomad3.explore-event/v3","type":"error","classification":"invalid_input","message":"flag provided but not defined: -unknown"}` + "\n"}},
	} {
		if got := runGomad(test.arguments...); got != test.want {
			t.Fatalf("gomad %s:\n%s\nwant:\n%s", strings.Join(test.arguments, " "), got, test.want)
		}
	}
}

// TestCharacterizeInstallationResolution pins how each command reports an
// installation it cannot resolve, from the flag and from the environment.
func TestCharacterizeInstallationResolution(t *testing.T) {
	const reason = `CLI --toolchain-root toolchain root must be an absolute non-root clean path: "relative"`
	t.Setenv("GOMAD3_TOOLCHAIN_DIR", "")
	for _, test := range []struct {
		arguments []string
		want      commandResult
	}{
		{[]string{"explore", "--toolchain-root=relative", "go-run", "./cmd"}, commandResult{status: 3, stderr: "gomad: runner_failure: resolve Gomad installation: " + reason + "\n"}},
		{[]string{"explore", "--json", "--toolchain-root=relative", "go-run", "./cmd"}, commandResult{status: 3, stdout: `{"schema":"gomad3.explore-event/v3","type":"error","classification":"runner_failure","message":"resolve Gomad installation: CLI --toolchain-root toolchain root must be an absolute non-root clean path: \"relative\""}` + "\n"}},
		// Plan resolves the installation before it requires --output.
		{[]string{"plan", "--toolchain-root=relative", "go-run", "./cmd"}, commandResult{status: 3, stderr: "gomad: runner_failure: resolve Gomad installation: " + reason + "\n"}},
		{[]string{"qualify", "--toolchain-root=relative", "go-run", "./cmd"}, commandResult{status: 3, stderr: "gomad: runner_failure: resolve Gomad installation: " + reason + "\n"}},
		{[]string{"resume", "--toolchain-root=relative", "/campaign"}, commandResult{status: 3, stderr: "gomad: runner_failure: resolve Gomad installation: " + reason + "\n"}},
		{[]string{"replay", "--toolchain-root=relative", "/artifact"}, commandResult{status: 3, stderr: "resolve Gomad installation: " + reason + "\n"}},
		{[]string{"minimize", "--toolchain-root=relative", "/artifact"}, commandResult{status: 3, stderr: "resolve Gomad installation: " + reason + "\n"}},
		{[]string{"execute-shard", "--shard=0/1", "--toolchain-root=relative", "/plan"}, commandResult{status: 3, stderr: "resolve Gomad installation: " + reason + "\n"}},
		{[]string{"analyze", "--toolchain-root=relative", "go-run", "./cmd"}, commandResult{status: 3, stderr: "resolve Gomad toolchain: resolve Gomad installation: " + reason + "\n"}},
		// Doctor reports an unresolvable installation as invalid input.
		{[]string{"doctor", "--toolchain-root=relative"}, commandResult{status: 2, stderr: "resolve Gomad installation: " + reason + "\n"}},
	} {
		if got := runGomad(test.arguments...); got != test.want {
			t.Fatalf("gomad %s:\n%s\nwant:\n%s", strings.Join(test.arguments, " "), got, test.want)
		}
	}

	t.Setenv("GOMAD3_TOOLCHAIN_DIR", "environment")
	const environmentReason = `GOMAD3_TOOLCHAIN_DIR toolchain root must be an absolute non-root clean path: "environment"`
	if got, want := runGomad("replay", "/artifact"), (commandResult{status: 3, stderr: "resolve Gomad installation: " + environmentReason + "\n"}); got != want {
		t.Fatalf("environment installation = %s, want %s", got, want)
	}
	if got, want := runGomad("explore", "go-run", "./cmd"), (commandResult{status: 3, stderr: "gomad: runner_failure: resolve Gomad installation: " + environmentReason + "\n"}); got != want {
		t.Fatalf("environment installation = %s, want %s", got, want)
	}
	// An explicit --toolchain-root takes precedence over the environment.
	root := filepath.Join(t.TempDir(), "toolchain")
	artifacts := filepath.Join(t.TempDir(), "artifacts")
	doctor := runGomad("doctor", "--toolchain-root="+root, "--artifacts="+artifacts)
	if doctor.status != 1 || doctor.stderr != "" || !strings.HasSuffix(doctor.stdout, "installation: source=cli toolchain="+root+"\nrepair: install the Gomad toolchain at "+root+"\n") || !strings.Contains(doctor.stdout, "runner     ok    sha256:") {
		t.Fatalf("doctor with explicit root = %s", doctor)
	}
	doctorJSON := runGomad("doctor", "--json", "--toolchain-root="+root, "--artifacts="+artifacts)
	for _, want := range []string{`"schema":"gomad3.doctor/v3"`, `"available":false`, `"installation_source":"cli"`, `"toolchain_root":"` + root + `"`, `"artifact_directory":"` + artifacts + `"`, `"runner_build":"sha256:`} {
		if doctorJSON.status != 1 || doctorJSON.stderr != "" || !strings.Contains(doctorJSON.stdout, want) {
			t.Fatalf("doctor JSON with explicit root = %s, missing %s", doctorJSON, want)
		}
	}
}

func TestCharacterizeExploreRequestDefaultsAndWiring(t *testing.T) {
	installation := newFakeInstallation()
	var observed []runner.CampaignSpec
	explore := func(_ context.Context, spec runner.CampaignSpec) (runner.CampaignResult, error) {
		if spec.Progress == nil {
			t.Fatal("explore request has no progress reporter")
		}
		spec.Progress = nil
		observed = append(observed, spec)
		return runner.CampaignResult{CampaignPath: "/artifacts/v1/campaign", SelectionCount: 1, Attempted: 1, Succeeded: 1, StopReason: runner.StopSeedsExhausted}, nil
	}
	dependencies := installation.exploreDependencies(explore, nil)
	if got := runCommand(func(stdout, stderr *bytes.Buffer) int {
		return runExploreWith([]string{"go-run", "./cmd"}, stdout, stderr, dependencies)
	}); got.status != 0 {
		t.Fatalf("default explore = %s", got)
	}
	want := runner.CampaignSpec{
		Strategy: runner.StrategySeed, Seeds: "1", Parallel: min(runtime.NumCPU(), 8), ExecutionTimeout: 30 * time.Second, OverallTimeout: 10 * time.Minute, TerminateGrace: 2 * time.Second,
		OnFailure: runner.PolicyFirst, FailureBudget: 1, OutputLimit: 8 << 20, WorldTransitionLimit: 64 << 20, ClockTick: record.ClockTickStrict, IOTranscriptLimit: 64 << 20,
		Artifacts: ".gomad/artifacts", SupervisorCommand: []string{"/bin/gomad", "__supervisor"}, CoordinatorCommand: []string{"/bin/gomad", "__coordinator"}, RunnerBuild: "sha256:runner",
		Coverage: runner.CoverageNone, KeepSuccesses: runner.KeepSuccessesNone, ProgressInterval: 5 * time.Second,
		Target: target.Spec{Kind: target.KindGoRun, Source: "./cmd", WorkingDir: "/workspace", ToolchainRoot: "/toolchain", CapabilityMode: target.CapabilityModeClosure},
	}
	if len(observed) != 1 || !reflect.DeepEqual(observed[0], want) {
		t.Fatalf("default explore request =\n%#v\nwant\n%#v", observed, want)
	}
	if !reflect.DeepEqual(*installation.requested, []string{""}) {
		t.Fatalf("requested toolchain roots = %q", *installation.requested)
	}

	observed = nil
	module := t.TempDir()
	if err := os.WriteFile(filepath.Join(module, "go.mod"), []byte("module example.com/module\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	arguments := []string{
		"--toolchain-root=/bundle/toolchain", "--env", "A=1", "--env=B=two words", "--build-tag", "first", "--build-tag=second", "--io-ro-mount=/host=/target",
		"--coverage=semantic", "--require-probe=stdlib.os.openfile", "--choices", "--choice-bytes=2MiB", "--keep-successes=novel", "--success-limit=3", "--success-bytes=4MiB",
		"--clock-tick=forward", "--io-transcript-bytes=128MiB", "--working-dir=" + module, "--capability-mode=linked", "--artifacts=/artifacts", "--on-failure=budget", "--failure-budget=2", "--count=4",
		"go-test", "./pkg", "--", "-test.run=^TestName$", "literal;$value", "--", "--json",
	}
	if got := runCommand(func(stdout, stderr *bytes.Buffer) int {
		return runExploreWith(arguments, stdout, stderr, dependencies)
	}); got.status != 0 {
		t.Fatalf("explore = %s", got)
	}
	want = runner.CampaignSpec{
		Strategy: runner.StrategySeed, Seeds: "0-3", Parallel: min(runtime.NumCPU(), 8), ExecutionTimeout: 30 * time.Second, OverallTimeout: 10 * time.Minute, TerminateGrace: 2 * time.Second,
		OnFailure: runner.PolicyBudget, FailureBudget: 2, OutputLimit: 8 << 20, WorldTransitionLimit: 64 << 20, ChoiceTraceLimit: 2 << 20, ClockTick: record.ClockTickForward, IOTranscriptLimit: 128 << 20,
		Artifacts: "/artifacts", Environment: []string{"A=1", "B=two words"}, IOROMounts: []string{"/host=/target"},
		SupervisorCommand: []string{"/bin/gomad", "__supervisor"}, CoordinatorCommand: []string{"/bin/gomad", "__coordinator"}, RunnerBuild: "sha256:runner",
		Coverage: runner.CoverageSemantic, RequiredSemanticProbes: []string{"stdlib.os.openfile"}, KeepSuccesses: runner.KeepSuccessesNovel, SuccessArtifactLimit: 3, SuccessBytesLimit: 4 << 20,
		ProgressInterval: 5 * time.Second,
		Target: target.Spec{
			Kind: target.KindGoTest, Source: "./pkg", Args: []string{"-test.run=^TestName$", "literal;$value", "--", "--json"}, BuildTags: []string{"first", "second"},
			WorkingDir: module, ToolchainRoot: "/toolchain", CapabilityMode: target.CapabilityModeLinked,
		},
	}
	if len(observed) != 1 || !reflect.DeepEqual(observed[0], want) {
		t.Fatalf("explore request =\n%#v\nwant\n%#v", observed, want)
	}
	if !reflect.DeepEqual(*installation.requested, []string{"", "/bundle/toolchain"}) {
		t.Fatalf("requested toolchain roots = %q", *installation.requested)
	}

	observed = nil
	execArguments := []string{"--strategy=simulation-exploration", "--seeds=9", "--max-executions=2", "--max-forced-decisions=3", "--max-exploration-bytes=1MiB", "--max-exploration-result-bytes=2MiB", "--max-runtime-decisions=4", "--max-scenario-decisions=5", "--max-network-decisions=6", "--max-storage-decisions=7", "--max-fault-decisions=8", "--max-crash-decisions=9", "exec", "--provenance", "/target.provenance.json", "--", "/target", "argument"}
	if got := runCommand(func(stdout, stderr *bytes.Buffer) int {
		return runExploreWith(execArguments, stdout, stderr, dependencies)
	}); got.status != 0 {
		t.Fatalf("simulation explore = %s", got)
	}
	if len(observed) != 1 {
		t.Fatalf("simulation explore requests = %d", len(observed))
	}
	request := observed[0]
	if request.Strategy != runner.StrategySimulationExploration || request.Seeds != "9" || request.ChoiceTraceLimit != 8<<20 || request.MaxExecutions != 2 || request.MaxForcedDecisions != 3 || request.MaxExplorationBytes != 1<<20 || request.MaxExplorationResultBytes != 2<<20 ||
		request.SimulationDimensionLimits != (runner.SimulationDimensionLimits{Runtime: 4, Scenario: 5, Network: 6, Storage: 7, Fault: 8, Crash: 9}) ||
		!reflect.DeepEqual(request.Target, target.Spec{Kind: target.KindExec, Source: "/target", Provenance: "/target.provenance.json", Args: []string{"argument"}, WorkingDir: "/workspace", ToolchainRoot: "/toolchain", CapabilityMode: target.CapabilityModeClosure}) {
		t.Fatalf("simulation explore request = %#v", request)
	}
}

func TestCharacterizeExploreOutputAndStatus(t *testing.T) {
	success := runner.CampaignResult{CampaignPath: "/artifacts/v1/campaign", SelectionCount: 2, Attempted: 2, Succeeded: 2, StopReason: runner.StopSeedsExhausted}
	failure := runner.CampaignResult{CampaignPath: "/artifacts/v1/campaign", SelectionCount: 2, Attempted: 2, Succeeded: 1, Failures: 1, DistinctFailures: 1, StopReason: runner.StopFirstFailure, Artifacts: []string{"/artifacts/v1/failure one"}}
	for _, test := range []struct {
		name      string
		arguments []string
		result    runner.CampaignResult
		err       error
		want      commandResult
	}{
		{"text success", nil, success, nil, commandResult{status: 0, stdout: "gomad: classification=success attempted=2 succeeded=2 failures=0 watchdogs=0 replay-divergences=0 distinct=0 retained-successes=0 retained-success-bytes=0 stop=seeds_exhausted artifact=/artifacts/v1/campaign\n"}},
		{"JSON success", []string{"--json"}, success, nil, commandResult{status: 0, stdout: `{"schema":"gomad3.explore-event/v3","type":"result","classification":"success","campaign_path":"/artifacts/v1/campaign","selected":2,"attempted":2,"succeeded":2,"stop_reason":"seeds_exhausted"}` + "\n"}},
		{"text failure", nil, failure, nil, commandResult{status: 1, stdout: "gomad: classification=target_failure attempted=2 succeeded=1 failures=1 watchdogs=0 replay-divergences=0 distinct=1 retained-successes=0 retained-success-bytes=0 stop=first_failure artifact=/artifacts/v1/campaign\ngomad: retained failure: /artifacts/v1/failure one\ngomad: replay: gomad replay '/artifacts/v1/failure one'\n"}},
		{"JSON failure", []string{"--json"}, failure, nil, commandResult{status: 1, stdout: `{"schema":"gomad3.explore-event/v3","type":"result","classification":"target_failure","campaign_path":"/artifacts/v1/campaign","selected":2,"attempted":2,"succeeded":1,"failures":1,"novelty":1,"stop_reason":"first_failure"}` + "\n" + `{"schema":"gomad3.explore-event/v3","type":"artifact","classification":"target_failure","path":"/artifacts/v1/failure one","replay_command":"gomad replay '/artifacts/v1/failure one'"}` + "\n"}},
		{"invalid input error", nil, runner.CampaignResult{}, errors.New("bad request"), commandResult{status: 2, stderr: "gomad: invalid_input: bad request\n"}},
		{"runner failure", nil, runner.CampaignResult{}, &runner.HostError{Reason: "prepare", Err: errors.New("disk")}, commandResult{status: 3, stderr: "gomad: runner_failure: " + (&runner.HostError{Reason: "prepare", Err: errors.New("disk")}).Error() + "\n"}},
		{"cancelled", []string{"--json"}, runner.CampaignResult{}, &runner.HostError{Reason: "cancelled"}, commandResult{status: 3, stdout: `{"schema":"gomad3.explore-event/v3","type":"error","classification":"cancelled","message":"gomad3 Runner/host failure: cancelled"}` + "\n"}},
		{"semantic coverage failure", nil, runner.CampaignResult{}, &deterministicio.MissingSemanticProbesError{Probes: []string{"probe"}}, commandResult{status: 1, stderr: "gomad: semantic_coverage_failure: required semantic probes were not observed: probe\n"}},
		{"unsupported target", nil, runner.CampaignResult{}, &target.UnsupportedCapabilityError{ImportPath: "example.com/p", Capability: "network"}, commandResult{status: 2, stderr: "gomad: unsupported_target: " + (&target.UnsupportedCapabilityError{ImportPath: "example.com/p", Capability: "network"}).Error() + "\n"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			dependencies := newFakeInstallation().exploreDependencies(func(context.Context, runner.CampaignSpec) (runner.CampaignResult, error) {
				return test.result, test.err
			}, nil)
			got := runCommand(func(stdout, stderr *bytes.Buffer) int {
				return runExploreWith(append(append([]string(nil), test.arguments...), "go-run", "./cmd"), stdout, stderr, dependencies)
			})
			if got != test.want {
				t.Fatalf("explore:\n%s\nwant:\n%s", got, test.want)
			}
		})
	}
	installationFailure := newFakeInstallation().failing(errFakeInstallation)
	if got, want := runCommand(func(stdout, stderr *bytes.Buffer) int {
		return runExploreWith([]string{"go-run", "./cmd"}, stdout, stderr, installationFailure.exploreDependencies(nil, nil))
	}), (commandResult{status: 3, stderr: "gomad: runner_failure: " + errFakeInstallation.Error() + "\n"}); got != want {
		t.Fatalf("explore installation failure = %s, want %s", got, want)
	}
}

func TestCharacterizePlanRequestAndOutput(t *testing.T) {
	var observed []runner.CampaignPlanSpec
	planned := runner.CampaignPlanResult{Path: "/plans/plan.json", BundlePath: "/plans/bundle", SHA256: "sha256:plan", SelectionCount: 4, TargetSHA256: "sha256:target"}
	plan := func(_ context.Context, spec runner.CampaignPlanSpec) (runner.CampaignPlanResult, error) {
		spec.Campaign.Progress = nil
		observed = append(observed, spec)
		return planned, nil
	}
	dependencies := newFakeInstallation().exploreDependencies(nil, plan)
	if got, want := runCommand(func(stdout, stderr *bytes.Buffer) int {
		return runPlanWith([]string{"--output=/plans/plan.json", "--count=4", "--env=A=1", "--build-tag=tag", "go-run", "./cmd", "--", "argument"}, stdout, stderr, dependencies)
	}), (commandResult{status: 0, stdout: "gomad plan: path=/plans/plan.json bundle=/plans/bundle sha256=sha256:plan selected=4 target=sha256:target\n"}); got != want {
		t.Fatalf("plan = %s, want %s", got, want)
	}
	if got, want := runCommand(func(stdout, stderr *bytes.Buffer) int {
		return runPlanWith([]string{"--json", "--output=/plans/plan.json", "--on-failure=first", "go-run", "./cmd"}, stdout, stderr, dependencies)
	}), (commandResult{status: 0, stdout: `{"path":"/plans/plan.json","bundle_path":"/plans/bundle","sha256":"sha256:plan","selection_count":4,"target_sha256":"sha256:target"}` + "\n"}); got != want {
		t.Fatalf("plan JSON = %s, want %s", got, want)
	}
	if len(observed) != 2 {
		t.Fatalf("plan requests = %d", len(observed))
	}
	first := observed[0]
	if first.Output != "/plans/plan.json" || first.Campaign.OnFailure != runner.PolicyAll || first.Campaign.Seeds != "0-3" || !reflect.DeepEqual(first.Campaign.Environment, []string{"A=1"}) ||
		!reflect.DeepEqual(first.Campaign.Target.BuildTags, []string{"tag"}) || !reflect.DeepEqual(first.Campaign.Target.Args, []string{"argument"}) ||
		!reflect.DeepEqual(first.Campaign.SupervisorCommand, []string{"/bin/gomad", "__supervisor"}) || !reflect.DeepEqual(first.Campaign.CoordinatorCommand, []string{"/bin/gomad", "__coordinator"}) || first.Campaign.RunnerBuild != "sha256:runner" || first.Campaign.Target.ToolchainRoot != "/toolchain" {
		t.Fatalf("plan request = %#v", first)
	}
	// An explicit --on-failure follows plan's implicit --on-failure=all.
	if observed[1].Campaign.OnFailure != runner.PolicyFirst {
		t.Fatalf("plan failure policy = %q", observed[1].Campaign.OnFailure)
	}
	// Plan requires --output only after the installation is resolved.
	if got, want := runCommand(func(stdout, stderr *bytes.Buffer) int {
		return runPlanWith([]string{"go-run", "./cmd"}, stdout, stderr, dependencies)
	}), (commandResult{status: 2, stderr: "gomad: invalid_input: gomad plan requires --output FILE\n"}); got != want {
		t.Fatalf("plan without output = %s, want %s", got, want)
	}
	failing := newFakeInstallation().exploreDependencies(nil, func(context.Context, runner.CampaignPlanSpec) (runner.CampaignPlanResult, error) {
		return runner.CampaignPlanResult{}, &runner.HostError{Reason: "plan", Err: errors.New("write")}
	})
	if got := runCommand(func(stdout, stderr *bytes.Buffer) int {
		return runPlanWith([]string{"--json", "--output=/plan", "go-run", "./cmd"}, stdout, stderr, failing)
	}); got.status != 3 || got.stderr != "" || !strings.Contains(got.stdout, `"classification":"runner_failure"`) {
		t.Fatalf("plan failure = %s", got)
	}
}

func TestCharacterizeReplayRequestOutputAndStatus(t *testing.T) {
	installation := newFakeInstallation()
	var observed []runner.ReplaySpec
	result := func(value runner.ReplayResult, err error) replayDependencies {
		return installation.replayDependencies(func(_ context.Context, spec runner.ReplaySpec) (runner.ReplayResult, error) {
			observed = append(observed, spec)
			return value, err
		})
	}
	succeeded := runner.ReplayResult{Artifact: artifact.Artifact{Path: "/artifact", Manifest: record.ExecutionRecord{Outcome: record.Outcome{Domain: "success"}}}, Match: true, ChoiceReplayStatus: runner.ChoiceReplayExact}
	for _, test := range []struct {
		name      string
		arguments []string
		result    runner.ReplayResult
		err       error
		want      commandResult
	}{
		{"verify only", []string{"--verify-only", "/artifact"}, succeeded, nil, commandResult{status: 0, stdout: "gomad: verified /artifact\n"}},
		{"reproduced success", []string{"/artifact"}, succeeded, nil, commandResult{status: 0, stdout: "gomad: reproduced=true diagnostic=false result=success choice-replay=exact\n"}},
		{"divergence", []string{"/artifact"}, runner.ReplayResult{Divergence: "stdout", ChoiceReplayStatus: runner.ChoiceReplayExact}, nil, commandResult{status: 1, stdout: "gomad: reproduced=false divergence=stdout choice-replay=exact\n"}},
		{"preflight", []string{"/artifact"}, runner.ReplayResult{}, &runner.ReplayPreflightError{Err: errors.New("toolchain changed")}, commandResult{status: 2, stderr: "incompatible replay artifact: toolchain changed\n"}},
		{"replay failure", []string{"/artifact"}, runner.ReplayResult{}, errors.New("supervisor failed"), commandResult{status: 3, stderr: "supervisor failed\n"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := runCommand(func(stdout, stderr *bytes.Buffer) int {
				return runReplayWith(test.arguments, stdout, stderr, result(test.result, test.err))
			}); got != test.want {
				t.Fatalf("replay:\n%s\nwant:\n%s", got, test.want)
			}
		})
	}
	observed = nil
	if got := runCommand(func(stdout, stderr *bytes.Buffer) int {
		return runReplayWith([]string{"--toolchain-root=/bundle", "--observed=/observed", "--verify-only", "/artifact"}, stdout, stderr, result(succeeded, nil))
	}); got.status != 0 {
		t.Fatalf("replay = %s", got)
	}
	want := runner.ReplaySpec{ArtifactPath: "/artifact", VerifyOnly: true, ToolchainRoot: "/toolchain", ObservedDir: "/observed", SupervisorCommand: []string{"/bin/gomad", "__supervisor"}}
	if len(observed) != 1 || !reflect.DeepEqual(observed[0], want) || (*installation.requested)[len(*installation.requested)-1] != "/bundle" {
		t.Fatalf("replay request = %#v requested=%q", observed, *installation.requested)
	}
	if got, want := runCommand(func(stdout, stderr *bytes.Buffer) int {
		return runReplayWith([]string{"/artifact"}, stdout, stderr, newFakeInstallation().failing(errFakeInstallation).replayDependencies(nil))
	}), (commandResult{status: 3, stderr: errFakeInstallation.Error() + "\n"}); got != want {
		t.Fatalf("replay installation failure = %s, want %s", got, want)
	}
}

func TestCharacterizeExecuteShardRequestOutputAndStatus(t *testing.T) {
	var observed []runner.CampaignShardSpec
	dependencies := func(result runner.CampaignResult, err error) campaignShardDependencies {
		return newFakeInstallation().campaignShardDependencies(func(_ context.Context, spec runner.CampaignShardSpec) (runner.CampaignResult, error) {
			if spec.Progress == nil {
				t.Fatal("shard request has no progress reporter")
			}
			spec.Progress = nil
			observed = append(observed, spec)
			return result, err
		})
	}
	success := runner.CampaignResult{CampaignPath: "/artifacts/v1/shard", SelectionCount: 1, Attempted: 1, Succeeded: 1, StopReason: runner.StopSeedsExhausted}
	if got, want := runCommand(func(stdout, stderr *bytes.Buffer) int {
		return runCampaignShardWith([]string{"--json", "--artifacts=/artifacts", "--toolchain-root=/bundle", "--shard=1/2", "/plan"}, stdout, stderr, dependencies(success, nil))
	}), (commandResult{status: 0, stdout: `{"schema":"gomad3.explore-event/v3","type":"result","classification":"success","campaign_path":"/artifacts/v1/shard","selected":1,"attempted":1,"succeeded":1,"stop_reason":"seeds_exhausted"}` + "\n"}); got != want {
		t.Fatalf("execute-shard = %s, want %s", got, want)
	}
	want := runner.CampaignShardSpec{PlanPath: "/plan", Shard: runner.CampaignShard{Index: 1, Count: 2}, Artifacts: "/artifacts", ToolchainRoot: "/toolchain", RunnerBuild: "sha256:runner", SupervisorCommand: []string{"/bin/gomad", "__supervisor"}, ProgressInterval: 5 * time.Second}
	if len(observed) != 1 || !reflect.DeepEqual(observed[0], want) {
		t.Fatalf("execute-shard request = %#v", observed)
	}
	failed := runner.CampaignResult{CampaignPath: "/artifacts/v1/shard", Attempted: 1, Failures: 1}
	if got := runCommand(func(stdout, stderr *bytes.Buffer) int {
		return runCampaignShardWith([]string{"--shard=0/1", "/plan"}, stdout, stderr, dependencies(failed, nil))
	}); got.status != 1 || !strings.HasPrefix(got.stdout, "gomad: classification=target_failure ") {
		t.Fatalf("failed shard = %s", got)
	}
	if got, want := runCommand(func(stdout, stderr *bytes.Buffer) int {
		return runCampaignShardWith([]string{"--shard=0/1", "/plan"}, stdout, stderr, dependencies(runner.CampaignResult{}, errors.New("plan changed")))
	}), (commandResult{status: 2, stderr: "gomad: invalid_input: plan changed\n"}); got != want {
		t.Fatalf("invalid shard plan = %s, want %s", got, want)
	}
	if got, want := runCommand(func(stdout, stderr *bytes.Buffer) int {
		return runCampaignShardWith([]string{"--shard=0/1", "/plan"}, stdout, stderr, newFakeInstallation().failing(errFakeInstallation).campaignShardDependencies(nil))
	}), (commandResult{status: 3, stderr: errFakeInstallation.Error() + "\n"}); got != want {
		t.Fatalf("shard installation failure = %s, want %s", got, want)
	}
}

// TestCharacterizeInstallationWiring pins how resume, minimize, qualify and
// analyze pass the resolved installation and child-mode commands to their
// operations, and how each reports an unresolvable installation.
func TestCharacterizeInstallationWiring(t *testing.T) {
	installation := newFakeInstallation()
	var resumed runner.ResumeSpec
	resume := installation.resumeDependencies(func(_ context.Context, spec runner.ResumeSpec) (runner.CampaignResult, error) {
		spec.Progress = nil
		resumed = spec
		return runner.CampaignResult{CampaignPath: "/campaign"}, nil
	})
	if got := runCommand(func(stdout, stderr *bytes.Buffer) int {
		return runResumeWith([]string{"--toolchain-root=/bundle", "--guide-regression=false", "/campaign"}, stdout, stderr, resume)
	}); got.status != 0 {
		t.Fatalf("resume = %s", got)
	}
	regression := false
	if want := (runner.ResumeSpec{CampaignPath: "/campaign", ToolchainRoot: "/toolchain", RunnerBuild: "sha256:runner", SupervisorCommand: []string{"/bin/gomad", "__supervisor"}, CoordinatorCommand: []string{"/bin/gomad", "__coordinator"}, ProgressInterval: 5 * time.Second, GuideRegression: &regression}); !reflect.DeepEqual(resumed, want) {
		t.Fatalf("resume request = %#v, want %#v", resumed, want)
	}

	var minimized runner.MinimizeSpec
	minimize := installation.minimizeDependencies(func(_ context.Context, spec runner.MinimizeSpec) (runner.MinimizeResult, error) {
		minimized = spec
		return runner.MinimizeResult{Artifact: artifact.Artifact{Path: "/minimized"}, Attempts: 1, AttemptBudget: 64, StopReason: "minimal"}, nil
	})
	if got, want := runCommand(func(stdout, stderr *bytes.Buffer) int {
		return runMinimizeWith([]string{"--toolchain-root=/bundle", "--artifacts=/artifacts", "/artifact"}, stdout, stderr, minimize)
	}), (commandResult{status: 0, stdout: "gomad minimize: changed=false attempts=1/64 accepted=0 stop=minimal artifact=/minimized\n"}); got != want {
		t.Fatalf("minimize = %s, want %s", got, want)
	}
	if want := (runner.MinimizeSpec{ArtifactPath: "/artifact", OutputRoot: "/artifacts/minimized", AttemptBudget: 64, ToolchainRoot: "/toolchain", SupervisorCommand: []string{"/bin/gomad", "__supervisor"}}); !reflect.DeepEqual(minimized, want) {
		t.Fatalf("minimize request = %#v, want %#v", minimized, want)
	}

	var qualified []runner.CampaignSpec
	var replayed []runner.ReplaySpec
	qualify := installation.qualifyDependencies(func(_ context.Context, spec runner.CampaignSpec) (runner.CampaignResult, error) {
		spec.Progress = nil
		qualified = append(qualified, spec)
		evidence := characterizationEvidence()
		return runner.CampaignResult{CampaignPath: fmt.Sprintf("/campaign-%d", len(qualified)), SelectionCount: 1, Attempted: 1, Failures: 1, Artifacts: []string{fmt.Sprintf("/failure-%d", len(qualified))}, ExecutionEvidence: &evidence}, nil
	}, func(_ context.Context, spec runner.ReplaySpec) (runner.ReplayResult, error) {
		replayed = append(replayed, spec)
		return runner.ReplayResult{Match: true, ChoiceReplayStatus: runner.ChoiceReplayExact}, nil
	}, func(string, qualification.QualificationReport) (string, error) { return "/report.json", nil })
	if got := runCommand(func(stdout, stderr *bytes.Buffer) int {
		return runQualifyWith([]string{"--toolchain-root=/bundle", "--env=A=1", "--build-tag=tag", "go-test", "./pkg", "--", "-test.run=X"}, stdout, stderr, qualify)
	}); got.status != 1 || !strings.Contains(got.stdout, "gomad: qualification qualified=false") {
		t.Fatalf("qualify = %s", got)
	}
	if len(qualified) != 2 || len(replayed) != 2 {
		t.Fatalf("qualify requests = %d replays = %d", len(qualified), len(replayed))
	}
	for _, spec := range qualified {
		if !reflect.DeepEqual(spec.SupervisorCommand, []string{"/bin/gomad", "__supervisor"}) || !reflect.DeepEqual(spec.CoordinatorCommand, []string{"/bin/gomad", "__coordinator"}) || spec.RunnerBuild != "sha256:runner" ||
			spec.Target.ToolchainRoot != "/toolchain" || !reflect.DeepEqual(spec.Environment, []string{"A=1"}) || !reflect.DeepEqual(spec.Target.BuildTags, []string{"tag"}) || !reflect.DeepEqual(spec.Target.Args, []string{"-test.run=X"}) {
			t.Fatalf("qualify request = %#v", spec)
		}
	}
	for _, spec := range replayed {
		if spec.ToolchainRoot != "/toolchain" || !reflect.DeepEqual(spec.SupervisorCommand, []string{"/bin/gomad", "__supervisor"}) {
			t.Fatalf("qualify replay request = %#v", spec)
		}
	}
	if got := *installation.requested; !reflect.DeepEqual(got, []string{"/bundle", "/bundle", "/bundle"}) {
		t.Fatalf("requested toolchain roots = %q", got)
	}

	unavailable := newFakeInstallation().failing(errFakeInstallation)
	for _, test := range []struct {
		name    string
		command func(stdout, stderr *bytes.Buffer) int
		want    commandResult
	}{
		{"resume", func(stdout, stderr *bytes.Buffer) int {
			return runResumeWith([]string{"/campaign"}, stdout, stderr, unavailable.resumeDependencies(nil))
		}, commandResult{status: 3, stderr: "gomad: runner_failure: " + errFakeInstallation.Error() + "\n"}},
		{"minimize", func(stdout, stderr *bytes.Buffer) int {
			return runMinimizeWith([]string{"/artifact"}, stdout, stderr, unavailable.minimizeDependencies(nil))
		}, commandResult{status: 3, stderr: errFakeInstallation.Error() + "\n"}},
		{"qualify", func(stdout, stderr *bytes.Buffer) int {
			return runQualifyWith([]string{"--json", "go-run", "./cmd"}, stdout, stderr, unavailable.qualifyDependencies(nil, nil, nil))
		}, commandResult{status: 3, stdout: `{"schema":"gomad3.qualify-event/v1","type":"error","classification":"runner_failure","message":"` + errFakeInstallation.Error() + `"}` + "\n"}},
		{"analyze", func(stdout, stderr *bytes.Buffer) int {
			return runAnalyzeWith([]string{"go-run", "./cmd"}, stdout, stderr, analyzeDependencies{toolchain: unavailable.analyzeToolchain, workingDirectory: func() (string, error) { return "/workspace", nil }})
		}, commandResult{status: 3, stderr: "resolve Gomad toolchain: " + errFakeInstallation.Error() + "\n"}},
	} {
		if got := runCommand(test.command); got != test.want {
			t.Fatalf("%s installation failure:\n%s\nwant:\n%s", test.name, got, test.want)
		}
	}
}

func characterizationEvidence() runner.ExecutionEvidence {
	return runner.ExecutionEvidence{
		Schema: runner.ExecutionEvidenceSchema, Seed: record.Uint64String(1), RunnerBuild: "sha256:runner",
		Toolchain:   record.Toolchain{GoVersion: "go1.27.1", BuildKey: "build", TargetGOOS: "darwin", TargetGOARCH: "arm64"},
		Target:      record.Target{Kind: "go-test", Source: "./pkg", SHA256: "sha256:target", Size: 1, Argv: []string{"gomad3-target"}},
		IOProfile:   deterministicio.Contract{Name: "deterministic", ImplementationSHA256: "sha256:io", InventorySHA256: "sha256:inventory"},
		Environment: []record.Environment{{Name: "GOMADSEED", Value: "1"}, {Name: "TZ", Value: "UTC"}},
		Outcome:     runner.OutcomeEvidence{Domain: "target_failure", Reason: "exit_status", Termination: "exit"}, GroupGone: true,
		Stdout: record.Stream{FullSHA256: "sha256:stdout"}, Stderr: record.Stream{FullSHA256: "sha256:stderr"},
		IOTranscriptSHA256: "sha256:transcript", IOTranscriptRecords: 1, IOTranscriptComplete: true,
		SemanticCoverage: deterministicio.SemanticCoverage{Schema: deterministicio.SemanticCoverageSchema, Digest: "sha256:coverage"},
	}
}

// TestCharacterizeOutputWriterFailures pins the status each command returns
// when the writer that carries its result or its error fails.
func TestCharacterizeOutputWriterFailures(t *testing.T) {
	t.Setenv("GOMAD3_TOOLCHAIN_DIR", "")
	installation := newFakeInstallation()
	success := runner.CampaignResult{CampaignPath: "/campaign", Attempted: 1, Succeeded: 1}
	explore := installation.exploreDependencies(func(context.Context, runner.CampaignSpec) (runner.CampaignResult, error) { return success, nil }, func(context.Context, runner.CampaignPlanSpec) (runner.CampaignPlanResult, error) {
		return runner.CampaignPlanResult{Path: "/plan"}, nil
	})
	replay := installation.replayDependencies(func(context.Context, runner.ReplaySpec) (runner.ReplayResult, error) {
		return runner.ReplayResult{Artifact: artifact.Artifact{Path: "/artifact"}, Match: true}, nil
	})
	shard := installation.campaignShardDependencies(func(context.Context, runner.CampaignShardSpec) (runner.CampaignResult, error) { return success, nil })
	resume := installation.resumeDependencies(func(context.Context, runner.ResumeSpec) (runner.CampaignResult, error) { return success, nil })
	minimize := installation.minimizeDependencies(func(context.Context, runner.MinimizeSpec) (runner.MinimizeResult, error) {
		return runner.MinimizeResult{}, nil
	})
	for _, test := range []struct {
		name    string
		command func(stdout, stderr io.Writer) int
		status  int
	}{
		{"explore text result", func(stdout, stderr io.Writer) int {
			return runExploreWith([]string{"go-run", "./cmd"}, stdout, stderr, explore)
		}, 3},
		{"explore JSON result", func(stdout, stderr io.Writer) int {
			return runExploreWith([]string{"--json", "go-run", "./cmd"}, stdout, stderr, explore)
		}, 3},
		{"explore text input error", func(stdout, stderr io.Writer) int {
			return runExploreWith([]string{"--count=0", "go-run", "./cmd"}, stdout, stderr, explore)
		}, 3},
		{"explore JSON input error", func(stdout, stderr io.Writer) int {
			return runExploreWith([]string{"--json", "--count=0", "go-run", "./cmd"}, stdout, stderr, explore)
		}, 3},
		{"plan text result", func(stdout, stderr io.Writer) int {
			return runPlanWith([]string{"--output=/plan", "go-run", "./cmd"}, stdout, stderr, explore)
		}, 3},
		{"plan JSON result", func(stdout, stderr io.Writer) int {
			return runPlanWith([]string{"--json", "--output=/plan", "go-run", "./cmd"}, stdout, stderr, explore)
		}, 3},
		{"replay result", func(stdout, stderr io.Writer) int {
			return runReplayWith([]string{"/artifact"}, stdout, stderr, replay)
		}, 3},
		// Verification-only replay does not check its output write.
		{"replay verification", func(stdout, stderr io.Writer) int {
			return runReplayWith([]string{"--verify-only", "/artifact"}, stdout, stderr, replay)
		}, 0},
		{"execute-shard result", func(stdout, stderr io.Writer) int {
			return runCampaignShardWith([]string{"--shard=0/1", "/plan"}, stdout, stderr, shard)
		}, 3},
		{"resume result", func(stdout, stderr io.Writer) int {
			return runResumeWith([]string{"/campaign"}, stdout, stderr, resume)
		}, 3},
		{"minimize text result", func(stdout, stderr io.Writer) int {
			return runMinimizeWith([]string{"/artifact"}, stdout, stderr, minimize)
		}, 3},
		{"minimize JSON result", func(stdout, stderr io.Writer) int {
			return runMinimizeWith([]string{"--json", "/artifact"}, stdout, stderr, minimize)
		}, 3},
		{"minimize usage", func(stdout, stderr io.Writer) int { return runMinimizeWith(nil, stdout, stderr, minimize) }, 3},
		{"recover usage", func(stdout, stderr io.Writer) int { return Run([]string{"recover"}, stdout, stderr) }, 3},
		{"inspect usage", func(stdout, stderr io.Writer) int { return Run([]string{"inspect"}, stdout, stderr) }, 3},
		{"analyze input error", func(stdout, stderr io.Writer) int {
			return Run([]string{"analyze", "--format=xml", "go-run", "./cmd"}, stdout, stderr)
		}, 3},
		{"qualify-set input error", func(stdout, stderr io.Writer) int { return Run([]string{"qualify-set"}, stdout, stderr) }, 3},
		{"doctor installation error", func(stdout, stderr io.Writer) int {
			return Run([]string{"doctor", "--toolchain-root=relative"}, stdout, stderr)
		}, 3},
		{"replay installation error", func(stdout, stderr io.Writer) int {
			return runReplayWith([]string{"/artifact"}, stdout, stderr, installation.failing(errFakeInstallation).replayDependencies(nil))
		}, 3},
	} {
		t.Run(test.name, func(t *testing.T) {
			if status := test.command(failingWriter{}, failingWriter{}); status != test.status {
				t.Fatalf("status = %d, want %d", status, test.status)
			}
		})
	}
}
