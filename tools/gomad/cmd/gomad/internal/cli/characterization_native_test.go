//go:build gomad_native

package cli

import (
	"bytes"
	"context"
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

	backend "go.temporal.io/server/tools/gomad/internal/backends"
	"go.temporal.io/server/tools/gomad/internal/backends/target"
	"go.temporal.io/server/tools/gomad/internal/evidence/record"
	"go.temporal.io/server/tools/gomad/internal/runner"
	campaignoptions "go.temporal.io/server/tools/gomad/internal/runner/options"
)

func TestCharacterizeNativeAnalyzeInstallationFailure(t *testing.T) {
	unavailable := newFakeInstallation().failing(errFakeInstallation)
	got := runCommand(func(stdout, stderr *bytes.Buffer) int {
		return runAnalyzeWith([]string{"go-run", "./cmd"}, stdout, stderr, analyzeDependencies{toolchain: unavailable.analyzeToolchain, workingDirectory: func() (string, error) { return "/workspace", nil }})
	})
	want := commandResult{status: 3, stderr: "resolve Gomad toolchain: " + errFakeInstallation.Error() + "\n"}
	if got != want {
		t.Fatalf("analyze installation failure: %v; want %v", got, want)
	}
}

func TestCharacterizeHelpAndUsage(t *testing.T) {
	t.Setenv("GOMAD_TOOLCHAIN_DIR", "")
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
		if err := reader.Close(); err != nil {
			t.Errorf("close coordinator request reader: %v", err)
		}
	}()
	if got, want := runGomad("__coordinator"), (commandResult{status: 3, stderr: "decode coordinator request: EOF\n"}); got != want {
		t.Fatalf("coordinator with empty request = %s, want %s", got, want)
	}
}

func TestCharacterizeInputRejection(t *testing.T) {
	t.Setenv("GOMAD_TOOLCHAIN_DIR", "")
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
		{"choice bytes above maximum", []string{"explore", "--choices", "--choice-bytes=65MiB", "go-run", "./cmd"}, invalid(fmt.Sprintf("--choice-bytes must be between %d bytes and 64MiB", backend.MinimumChoiceTraceBytes))},
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
		{"explore JSON input error", []string{"explore", "--json", "--count=0", "go-run", "./cmd"}, commandResult{status: 2, stdout: `{"schema":"gomad.explore-event/v3","type":"error","classification":"invalid_input","message":"--count must be greater than zero"}` + "\n"}},
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
		{[]string{"explore", "--json", "--output-limit=0", "go-run", "./cmd"}, commandResult{status: 2, stdout: `{"schema":"gomad.explore-event/v3","type":"error","classification":"invalid_input","message":"invalid value \"0\" for flag -output-limit: invalid byte size \"0\""}` + "\n"}},
		{[]string{"explore", "--json", "--choices", "--choice-bytes=0", "--coverage=choice"}, commandResult{status: 2, stdout: `{"schema":"gomad.explore-event/v3","type":"error","classification":"invalid_input","message":"invalid value \"0\" for flag -choice-bytes: invalid byte size \"0\""}` + "\n"}},
		{[]string{"plan", "--json", "--choice-bytes=0", "--coverage=choice"}, commandResult{status: 2, stdout: `{"schema":"gomad.explore-event/v3","type":"error","classification":"invalid_input","message":"invalid value \"0\" for flag -choice-bytes: invalid byte size \"0\""}` + "\n"}},
		{[]string{"explore", "--json", "--parallel=many", "go-run", "./cmd"}, commandResult{status: 2, stdout: `{"schema":"gomad.explore-event/v3","type":"error","classification":"invalid_input","message":"invalid value \"many\" for flag -parallel: parse error"}` + "\n"}},
		{[]string{"qualify", "--json", "--seed=-1", "go-run", "./cmd"}, commandResult{status: 2, stdout: `{"schema":"gomad.qualify-event/v1","type":"error","classification":"invalid_input","message":"invalid value \"-1\" for flag -seed: parse error"}` + "\n"}},
		{[]string{"resume", "--json", "--unknown"}, commandResult{status: 2, stdout: `{"schema":"gomad.explore-event/v3","type":"error","classification":"invalid_input","message":"flag provided but not defined: -unknown"}` + "\n"}},
	} {
		if got := runGomad(test.arguments...); got != test.want {
			t.Fatalf("gomad %s:\n%s\nwant:\n%s", strings.Join(test.arguments, " "), got, test.want)
		}
	}
}

func TestCharacterizeInstallationResolution(t *testing.T) {
	const reason = `CLI --toolchain-root toolchain root must be an absolute non-root clean path: "relative"`
	t.Setenv("GOMAD_TOOLCHAIN_DIR", "")
	for _, test := range []struct {
		arguments []string
		want      commandResult
	}{
		{[]string{"explore", "--toolchain-root=relative", "go-run", "./cmd"}, commandResult{status: 3, stderr: "gomad: runner_failure: resolve Gomad installation: " + reason + "\n"}},
		{[]string{"explore", "--json", "--toolchain-root=relative", "go-run", "./cmd"}, commandResult{status: 3, stdout: `{"schema":"gomad.explore-event/v3","type":"error","classification":"runner_failure","message":"resolve Gomad installation: CLI --toolchain-root toolchain root must be an absolute non-root clean path: \"relative\""}` + "\n"}},
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

	t.Setenv("GOMAD_TOOLCHAIN_DIR", "environment")
	const environmentReason = `GOMAD_TOOLCHAIN_DIR toolchain root must be an absolute non-root clean path: "environment"`
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
	for _, want := range []string{`"schema":"gomad.doctor/v3"`, `"available":false`, `"installation_source":"cli"`, `"toolchain_root":"` + root + `"`, `"artifact_directory":"` + artifacts + `"`, `"runner_build":"sha256:`} {
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
		Artifacts: ".gomad/artifacts", RunnerBuild: "sha256:runner",
		Coverage: campaignoptions.CoverageNone, KeepSuccesses: runner.KeepSuccessesNone, ProgressInterval: 5 * time.Second,
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
		RunnerBuild: "sha256:runner",
		Coverage:    campaignoptions.CoverageSemantic, RequiredSemanticProbes: []string{"stdlib.os.openfile"}, KeepSuccesses: runner.KeepSuccessesNovel, SuccessArtifactLimit: 3, SuccessBytesLimit: 4 << 20,
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

func TestPublicCommandExplicitZeroAndIrrelevantFlagClassification(t *testing.T) {
	for _, test := range []struct {
		name      string
		arguments []string
		want      string
	}{
		{name: "explore count zero", arguments: []string{"explore", "--count=0", "go-run", "./pkg"}, want: "--count must be greater than zero"},
		{name: "explore irrelevant choice bytes", arguments: []string{"explore", "--choice-bytes=8MiB", "go-run", "./pkg"}, want: "--choice-bytes requires --choices"},
		{name: "plan missing output", arguments: []string{"plan", "go-run", "./pkg"}, want: "gomad plan requires --output"},
		{name: "qualify repeat zero", arguments: []string{"qualify", "--repeat=0", "go-run", "./pkg"}, want: "--repeat must be between"},
		{name: "analyze negative timeout", arguments: []string{"analyze", "--timeout=-1s", "go-run", "./pkg"}, want: "analysis timeout must be non-negative"},
	} {
		t.Run(test.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			if status := Run(test.arguments, &stdout, &stderr); status != 2 || !strings.Contains(stdout.String()+stderr.String(), test.want) {
				t.Fatalf("Run(%q) status=%d stdout=%q stderr=%q; want %q", test.arguments, status, stdout.String(), stderr.String(), test.want)
			}
		})
	}
}

func TestPlanAndExploreRejectHiddenPlanRoute(t *testing.T) {
	for _, arguments := range [][]string{
		{"explore", "--__plan", "go-run", "./pkg"},
		{"plan", "--__plan", "--output=plan.json", "go-run", "./pkg"},
	} {
		var stdout, stderr bytes.Buffer
		status := Run(arguments, &stdout, &stderr)
		if status != 2 || stdout.Len() != 0 || !strings.Contains(stderr.String(), "flag provided but not defined: -__plan") {
			t.Fatalf("Run(%q) status=%d stdout=%q stderr=%q", arguments, status, stdout.String(), stderr.String())
		}
	}
}

func TestPlanValidationOrderAndJSONRouting(t *testing.T) {
	for _, test := range []struct {
		arguments []string
		want      string
	}{
		{[]string{"plan", "--json", "--output=x", "--count=0", "go-run", "./pkg"}, "--count must be greater than zero"},
		{[]string{"plan", "--json", "--output=x", "--choice-bytes=8MiB", "go-run", "./pkg"}, "--choice-bytes requires --choices"},
		{[]string{"plan", "--json", "--output=x", "--strategy=choice-exploration", "--count=0", "go-run", "./pkg"}, "--strategy=choice-exploration does not accept --count"},
		{[]string{"plan", "--json", "--output=x", "--coverage=choice", "go-run", "./pkg"}, "--coverage=choice requires --choices"},
		{[]string{"plan", "--json", "--output=x", "--coverage=unknown", "--choice-bytes=1", "go-run", "./pkg"}, "unknown coverage mode"},
		{[]string{"plan", "--json", "--output=x", "--coverage=semantic", "--require-probe=unknown.probe", "--choice-bytes=1", "go-run", "./pkg"}, "unknown.probe"},
		{[]string{"plan", "--json", "--output=x", "--choices", "--choice-bytes=1", "invalid-target"}, "--choice-bytes must be between"},
		{[]string{"plan", "--json", "--output=x", "--strategy=", "--max-executions=0", "invalid-target"}, "exploration bounds require --strategy=choice-exploration"},
	} {
		var stdout, stderr bytes.Buffer
		status := Run(test.arguments, &stdout, &stderr)
		if status != 2 || stderr.Len() != 0 || !strings.Contains(stdout.String(), `"classification":"invalid_input"`) || !strings.Contains(stdout.String(), test.want) {
			t.Fatalf("Run(%q) status=%d stdout=%q stderr=%q; want %q", test.arguments, status, stdout.String(), stderr.String(), test.want)
		}
	}
}

func TestCharacterizePlanInputRejection(t *testing.T) {
	t.Setenv("GOMAD_TOOLCHAIN_DIR", "")
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

func TestCharacterizePlanFlagErrors(t *testing.T) {
	t.Setenv("GOMAD_TOOLCHAIN_DIR", "")
	unknown := runGomad("plan", "--unknown", "go-run", "./cmd")
	if unknown.status != 2 || unknown.stdout != "" || !strings.HasPrefix(unknown.stderr, "gomad: invalid_input: flag provided but not defined: -unknown\nUsage of gomad explore:\n") {
		t.Fatalf("plan unknown flag = %s", unknown)
	}
	if got, want := runGomad("plan", "--json", "--parallel=many", "go-run", "./cmd"), (commandResult{status: 2, stdout: `{"schema":"gomad.explore-event/v3","type":"error","classification":"invalid_input","message":"invalid value \"many\" for flag -parallel: parse error"}` + "\n"}); got != want {
		t.Fatalf("plan JSON flag error = %s, want %s", got, want)
	}
	if got, want := runGomad("plan", "--json", "--toolchain-root=relative", "--output=/plan", "--count=0", "go-run", "./cmd"), (commandResult{status: 2, stdout: `{"schema":"gomad.explore-event/v3","type":"error","classification":"invalid_input","message":"--count must be greater than zero"}` + "\n"}); got != want {
		t.Fatalf("plan JSON input error = %s, want %s", got, want)
	}
}

func TestCharacterizeHiddenPlanFlag(t *testing.T) {
	t.Setenv("GOMAD_TOOLCHAIN_DIR", "")
	for _, command := range []string{"explore", "plan"} {
		got := runGomad(command, "--toolchain-root=relative", "--__plan", "go-run", "./cmd")
		if got.status != 2 || got.stdout != "" || !strings.HasPrefix(got.stderr, "gomad: invalid_input: flag provided but not defined: -__plan\nUsage of gomad explore:\n") || strings.Contains(got.stderr, "-__plan\n    \t") {
			t.Fatalf("%s --__plan = %s", command, got)
		}
	}
	if got, want := runGomad("explore", "--json", "--toolchain-root=relative", "--__plan", "go-run", "./cmd"), (commandResult{status: 2, stdout: `{"schema":"gomad.explore-event/v3","type":"error","classification":"invalid_input","message":"flag provided but not defined: -__plan"}` + "\n"}); got != want {
		t.Fatalf("explore --json --__plan = %s, want %s", got, want)
	}
	dependencies := newFakeInstallation().exploreDependencies(nil, func(context.Context, runner.CampaignPlanSpec) (runner.CampaignPlanResult, error) {
		t.Fatal("--__plan reached the plan operation")
		return runner.CampaignPlanResult{}, nil
	})
	for _, command := range []func(stdout, stderr io.Writer) int{
		func(stdout, stderr io.Writer) int {
			return runExploreWith([]string{"--__plan", "--output=/plan", "go-run", "./cmd"}, stdout, stderr, dependencies)
		},
		func(stdout, stderr io.Writer) int {
			return runPlanWith([]string{"--__plan", "--output=/plan", "go-run", "./cmd"}, stdout, stderr, dependencies)
		},
	} {
		if got := runCommand(func(stdout, stderr *bytes.Buffer) int { return command(stdout, stderr) }); got.status != 2 || !strings.HasPrefix(got.stderr, "gomad: invalid_input: flag provided but not defined: -__plan\n") {
			t.Fatalf("--__plan = %s", got)
		}
	}
}

func TestNativeCommandWriterFailuresPreserveStatus(t *testing.T) {
	for _, arguments := range [][]string{
		{"explore", "--json", "--count=0", "go-run", "./pkg"},
		{"analyze", "--format=invalid", "go-run", "./pkg"},
	} {
		var stdout, stderr bytes.Buffer
		output := failingWriter{}
		if arguments[0] == "inspect" || arguments[0] == "analyze" {
			if status := Run(arguments, &stdout, output); status != 3 {
				t.Fatalf("Run(%q) status=%d stdout=%q", arguments, status, stdout.String())
			}
		} else if status := Run(arguments, output, &stderr); status != 3 {
			t.Fatalf("Run(%q) status=%d stderr=%q", arguments, status, stderr.String())
		}
	}
}
