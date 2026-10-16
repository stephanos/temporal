package cli

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
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
	qualificationworkload "go.temporal.io/server/tools/gomad/internal/backends/wasm/qualification"
	"go.temporal.io/server/tools/gomad/internal/evidence/artifact"
	"go.temporal.io/server/tools/gomad/internal/evidence/qualification"
	"go.temporal.io/server/tools/gomad/internal/evidence/record"
	"go.temporal.io/server/tools/gomad/internal/runner"
	campaignoptions "go.temporal.io/server/tools/gomad/internal/runner/options"
)

// Characterization of the gomad command grammar, defaults, output routing and
// exit statuses. These tests pin observable CLI behavior before application
// construction is refactored, and must keep passing unchanged afterwards.
// Dependencies are built only through the fake installation in this file.

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

// TestCharacterizeInputRejection pins malformed input, explicit zero values and
// flags irrelevant to the selected mode. Every case also names an unusable
// --toolchain-root, so it pins that input validation precedes installation
// resolution.
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

// TestCharacterizeInstallationResolution pins how each command reports an
// installation it cannot resolve, from the flag and from the environment.
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
		{"JSON success", []string{"--json"}, success, nil, commandResult{status: 0, stdout: `{"schema":"gomad.explore-event/v3","type":"result","classification":"success","campaign_path":"/artifacts/v1/campaign","selected":2,"attempted":2,"succeeded":2,"stop_reason":"seeds_exhausted"}` + "\n"}},
		{"text failure", nil, failure, nil, commandResult{status: 1, stdout: "gomad: classification=target_failure attempted=2 succeeded=1 failures=1 watchdogs=0 replay-divergences=0 distinct=1 retained-successes=0 retained-success-bytes=0 stop=first_failure artifact=/artifacts/v1/campaign\ngomad: retained failure: /artifacts/v1/failure one\ngomad: replay: gomad replay '/artifacts/v1/failure one'\n"}},
		{"JSON failure", []string{"--json"}, failure, nil, commandResult{status: 1, stdout: `{"schema":"gomad.explore-event/v3","type":"result","classification":"target_failure","campaign_path":"/artifacts/v1/campaign","selected":2,"attempted":2,"succeeded":1,"failures":1,"novelty":1,"stop_reason":"first_failure"}` + "\n" + `{"schema":"gomad.explore-event/v3","type":"artifact","classification":"target_failure","path":"/artifacts/v1/failure one","replay_command":"gomad replay '/artifacts/v1/failure one'"}` + "\n"}},
		{"invalid input error", nil, runner.CampaignResult{}, errors.New("bad request"), commandResult{status: 2, stderr: "gomad: invalid_input: bad request\n"}},
		{"runner failure", nil, runner.CampaignResult{}, &runner.HostError{Reason: "prepare", Err: errors.New("disk")}, commandResult{status: 3, stderr: "gomad: runner_failure: " + (&runner.HostError{Reason: "prepare", Err: errors.New("disk")}).Error() + "\n"}},
		{"cancelled", []string{"--json"}, runner.CampaignResult{}, &runner.HostError{Reason: "cancelled"}, commandResult{status: 3, stdout: `{"schema":"gomad.explore-event/v3","type":"error","classification":"cancelled","message":"gomad Runner/host failure: cancelled"}` + "\n"}},
		{"semantic coverage failure", nil, runner.CampaignResult{}, &record.MissingSemanticProbesError{Probes: []string{"probe"}}, commandResult{status: 1, stderr: "gomad: semantic_coverage_failure: required semantic probes were not observed: probe\n"}},
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
		first.Campaign.RunnerBuild != "sha256:runner" || first.Campaign.Target.ToolchainRoot != "/toolchain" {
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
	want := runner.ReplaySpec{ArtifactPath: "/artifact", VerifyOnly: true, ToolchainRoot: "/toolchain", ObservedDir: "/observed"}
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
	}), (commandResult{status: 0, stdout: `{"schema":"gomad.explore-event/v3","type":"result","classification":"success","campaign_path":"/artifacts/v1/shard","selected":1,"attempted":1,"succeeded":1,"stop_reason":"seeds_exhausted"}` + "\n"}); got != want {
		t.Fatalf("execute-shard = %s, want %s", got, want)
	}
	want := runner.CampaignShardSpec{PlanPath: "/plan", Shard: runner.CampaignShard{Index: 1, Count: 2}, Artifacts: "/artifacts", ToolchainRoot: "/toolchain", RunnerBuild: "sha256:runner", ProgressInterval: 5 * time.Second}
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
	if want := (runner.ResumeSpec{CampaignPath: "/campaign", ToolchainRoot: "/toolchain", RunnerBuild: "sha256:runner", ProgressInterval: 5 * time.Second, GuideRegression: &regression}); !reflect.DeepEqual(resumed, want) {
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
	if want := (runner.MinimizeSpec{ArtifactPath: "/artifact", OutputRoot: "/artifacts/minimized", AttemptBudget: 64, ToolchainRoot: "/toolchain"}); !reflect.DeepEqual(minimized, want) {
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
		if spec.RunnerBuild != "sha256:runner" ||
			spec.Target.ToolchainRoot != "/toolchain" || !reflect.DeepEqual(spec.Environment, []string{"A=1"}) || !reflect.DeepEqual(spec.Target.BuildTags, []string{"tag"}) || !reflect.DeepEqual(spec.Target.Args, []string{"-test.run=X"}) {
			t.Fatalf("qualify request = %#v", spec)
		}
	}
	for _, spec := range replayed {
		if spec.ToolchainRoot != "/toolchain" {
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
		}, commandResult{status: 3, stdout: `{"schema":"gomad.qualify-event/v1","type":"error","classification":"runner_failure","message":"` + errFakeInstallation.Error() + `"}` + "\n"}},
		{"analyze", func(stdout, stderr *bytes.Buffer) int {
			return runAnalyzeWith([]string{"go-run", "./cmd"}, stdout, stderr, analyzeDependencies{toolchain: unavailable.analyzeToolchain, workingDirectory: func() (string, error) { return "/workspace", nil }})
		}, commandResult{status: 3, stderr: "resolve Gomad toolchain: " + errFakeInstallation.Error() + "\n"}},
	} {
		if got := runCommand(test.command); got != test.want {
			t.Fatalf("%s installation failure:\n%s\nwant:\n%s", test.name, got, test.want)
		}
	}
}

func characterizationEvidence() record.ExecutionEvidence {
	return record.ExecutionEvidence{
		Schema: record.ExecutionEvidenceSchema, Seed: record.Uint64String(1), RunnerBuild: "sha256:runner",
		Toolchain:   record.Toolchain{GoVersion: "go1.27.1", BuildKey: "build", TargetGOOS: "darwin", TargetGOARCH: "arm64"},
		Target:      record.Target{Kind: "go-test", Source: "./pkg", SHA256: "sha256:target", Size: 1, Argv: []string{"gomad-target"}},
		IOProfile:   record.IOContract{Name: "deterministic", ImplementationSHA256: "sha256:io", InventorySHA256: "sha256:inventory"},
		Environment: []record.Environment{{Name: "GOMADSEED", Value: "1"}, {Name: "TZ", Value: "UTC"}},
		Outcome:     record.OutcomeEvidence{Domain: "target_failure", Reason: "exit_status", Termination: "exit"}, GroupGone: true,
		Stdout: record.Stream{FullSHA256: "sha256:stdout"}, Stderr: record.Stream{FullSHA256: "sha256:stderr"},
		IOTranscriptSHA256: "sha256:transcript", IOTranscriptRecords: 1, IOTranscriptComplete: true,
		SemanticCoverage: record.SemanticCoverage{Schema: record.SemanticCoverageSchema, Digest: "sha256:coverage"},
	}
}

// TestCharacterizeOutputWriterFailures pins the status each command returns
// when the writer that carries its result or its error fails.
func TestCharacterizeOutputWriterFailures(t *testing.T) {
	t.Setenv("GOMAD_TOOLCHAIN_DIR", "")
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
		{"replay verification", func(stdout, stderr io.Writer) int {
			return runReplayWith([]string{"--verify-only", "/artifact"}, stdout, stderr, replay)
		}, 3},
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

// fakeInstallation is the resolved installation a characterization test hands
// to a command through its dependency seam. It records every explicit
// --toolchain-root the command asked to resolve, so tests can pin both what a
// command requests and how it wires the resolved result into its operation.
// Only this file knows the seam's shape; the characterization tests in
// characterization_test.go build their dependencies through it.
type fakeInstallation struct {
	toolchainRoot string
	executable    string
	runnerBuild   string
	err           error
	requested     *[]string
}

func newFakeInstallation() fakeInstallation {
	return fakeInstallation{toolchainRoot: "/toolchain", executable: "/bin/gomad", runnerBuild: "sha256:runner", requested: new([]string)}
}

func (fake fakeInstallation) failing(err error) fakeInstallation {
	fake.err = err
	return fake
}

func (fake fakeInstallation) install(explicitToolchainRoot string) (installation, error) {
	*fake.requested = append(*fake.requested, explicitToolchainRoot)
	if fake.err != nil {
		return installation{}, fake.err
	}
	return installation{toolchainRoot: fake.toolchainRoot, executable: fake.executable, runnerBuild: fake.runnerBuild}, nil
}

func (fake fakeInstallation) exploreDependencies(explore func(context.Context, runner.CampaignSpec) (runner.CampaignResult, error), plan func(context.Context, runner.CampaignPlanSpec) (runner.CampaignPlanResult, error)) exploreDependencies {
	return exploreDependencies{install: fake.install, workingDirectory: func() (string, error) { return "/workspace", nil }, explore: explore, plan: plan}
}

func (fake fakeInstallation) replayDependencies(replay func(context.Context, runner.ReplaySpec) (runner.ReplayResult, error)) replayDependencies {
	return replayDependencies{install: fake.install, replay: replay}
}

func (fake fakeInstallation) campaignShardDependencies(run func(context.Context, runner.CampaignShardSpec) (runner.CampaignResult, error)) campaignShardDependencies {
	return campaignShardDependencies{install: fake.install, run: run}
}

func (fake fakeInstallation) resumeDependencies(run func(context.Context, runner.ResumeSpec) (runner.CampaignResult, error)) resumeDependencies {
	return resumeDependencies{install: fake.install, run: run}
}

func (fake fakeInstallation) minimizeDependencies(minimize func(context.Context, runner.MinimizeSpec) (runner.MinimizeResult, error)) minimizeDependencies {
	return minimizeDependencies{install: fake.install, minimize: minimize}
}

func (fake fakeInstallation) qualifyDependencies(run func(context.Context, runner.CampaignSpec) (runner.CampaignResult, error), replay func(context.Context, runner.ReplaySpec) (runner.ReplayResult, error), write func(string, qualification.QualificationReport) (string, error)) qualifyDependencies {
	return qualifyDependencies{install: fake.install, workingDirectory: func() (string, error) { return "/workspace", nil }, run: run, replay: replay, write: write}
}

// analyzeToolchain is the toolchain-root seam gomad analyze resolves through.
func (fake fakeInstallation) analyzeToolchain(explicitToolchainRoot string) (string, error) {
	resolved, err := fake.install(explicitToolchainRoot)
	return resolved.toolchainRoot, err
}

var errFakeInstallation = errors.New("resolve Gomad installation: fake installation is unavailable")

func TestPublicCommandGrammarRejectsMissingOperandsBeforeInstallation(t *testing.T) {
	t.Setenv("GOMAD_TOOLCHAIN_DIR", "relative-invalid-root")
	for _, command := range []string{
		"plan", "execute-shard", "merge", "explore", "qualify", "qualify-set",
		"merge-set", "compare-support", "analyze", "resume", "recover", "replay",
		"minimize", "doctor", "inspect",
	} {
		t.Run(command, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			arguments := []string{command}
			if command == "doctor" {
				arguments = append(arguments, "unexpected")
			}
			if status := Run(arguments, &stdout, &stderr); status != 2 {
				t.Fatalf("Run(%q) status=%d stdout=%q stderr=%q", arguments, status, stdout.String(), stderr.String())
			}
			if strings.Contains(stdout.String()+stderr.String(), "relative-invalid-root") {
				t.Fatalf("Run(%q) resolved installation before rejecting syntax: stdout=%q stderr=%q", arguments, stdout.String(), stderr.String())
			}
		})
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

func TestPublicCommandWriterFailuresPreserveStatus(t *testing.T) {
	for _, arguments := range [][]string{
		{"explore", "--json", "--count=0", "go-run", "./pkg"},
		{"analyze", "--format=invalid", "go-run", "./pkg"},
		{"inspect", "/missing/artifact"},
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

func TestQualifyGrammarPreservesEnvironmentTagsAndTargetArguments(t *testing.T) {
	for _, jsonOutput := range []bool{false, true} {
		var observed qualificationworkload.WorkloadSpec
		dependencies := qualifyDependencies{
			install: func(string) (installation, error) {
				return installation{toolchainRoot: "/toolchain", executable: "/bin/gomad", runnerBuild: "sha256:runner"}, nil
			},
			workingDirectory: func() (string, error) { return "/workspace", nil },
			workload: func(_ context.Context, spec qualificationworkload.WorkloadSpec) (qualificationworkload.WorkloadResult, error) {
				observed = spec
				return qualificationworkload.WorkloadResult{
					Report:     qualification.QualificationReport{Qualified: true, Deterministic: true, TargetSuccess: true},
					ReportPath: "/report.json",
				}, nil
			},
		}
		arguments := []string{"--env=FOO=bar", "--build-tag=gomad_fixture", "go-test", "./pkg", "--", "-test.run=Test Name", "literal;$value"}
		if jsonOutput {
			arguments = append([]string{"--json"}, arguments...)
		}
		var stdout, stderr bytes.Buffer
		status := runQualifyWith(arguments, &stdout, &stderr, dependencies)
		if status != 0 || stderr.Len() != 0 {
			t.Fatalf("json=%t status=%d stdout=%q stderr=%q", jsonOutput, status, stdout.String(), stderr.String())
		}
		got := observed.Campaign
		if len(got.Environment) != 1 || got.Environment[0] != "FOO=bar" || len(got.Target.BuildTags) != 1 || got.Target.BuildTags[0] != "gomad_fixture" || len(got.Target.Args) != 2 || got.Target.Args[0] != "-test.run=Test Name" || got.Target.Args[1] != "literal;$value" {
			t.Fatalf("json=%t campaign=%#v", jsonOutput, got)
		}
		want := "gomad: qualification qualified=true"
		if jsonOutput {
			want = `"classification":"qualified"`
		}
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("json=%t output=%q, want %q", jsonOutput, stdout.String(), want)
		}
	}
}

// Characterization of the grammar gomad plan shares with gomad explore: the
// plan request it builds, the explore validation it inherits in the same
// first-error order, the explore-only and plan-only flags, and the routing of
// its errors and results. Dependencies are built only through the fake
// installation in this file.

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
			Artifacts: ".gomad/artifacts", RunnerBuild: "sha256:runner",
			Coverage: campaignoptions.CoverageNone, KeepSuccesses: runner.KeepSuccessesNone, ProgressInterval: 5 * time.Second,
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

// TestCharacterizePlanFlagErrors pins flag-parse failures of plan in text and
// JSON, and that an explicit --on-failure follows plan's fixed policy only
// when it parses.
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
		{"cancelled JSON", &runner.HostError{Reason: "cancelled"}, true, commandResult{status: 3, stdout: `{"schema":"gomad.explore-event/v3","type":"error","classification":"cancelled","message":"gomad Runner/host failure: cancelled"}` + "\n"}},
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
	}), (commandResult{status: 2, stdout: `{"schema":"gomad.explore-event/v3","type":"error","classification":"invalid_input","message":"gomad plan requires --output FILE"}` + "\n"}); got != want {
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

// TestCharacterizeHiddenPlanFlag pins that the hidden --__plan route is gone
// (fn-109 R6): explore and plan reject --__plan as an unknown flag, before
// resolving the installation, and never reach the plan operation.
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

// preMoveWASMArtifact is a stock-profile success retained before the backend
// move and before artifacts declared a storage lineage. The native fn-113
// pre-move artifact is no longer read: historical artifacts are not reopened
// by the new build.
const preMoveWASMArtifact = "../../../../internal/backends/wasm/backend/testdata/premove-stock-artifact"

// stageRetainedArtifact copies a checked-in artifact with the private modes
// that OpenArtifact requires and version control does not keep.
func stageRetainedArtifact(t *testing.T, source string) string {
	t.Helper()
	destination := filepath.Join(t.TempDir(), "artifact")
	err := filepath.WalkDir(source, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		relative, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		target := filepath.Join(destination, relative)
		if entry.IsDir() {
			return os.Mkdir(target, 0o700)
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		mode := os.FileMode(0o600)
		if info.Mode()&0o100 != 0 {
			mode = 0o700
		}
		return os.WriteFile(target, data, mode)
	})
	if err != nil {
		t.Fatal(err)
	}
	return destination
}

const preLineageRefusal = `unsupported artifact storage format "pre-lineage": the artifact has no storage.json; this Gomad reads only "gomad.artifact-storage/v1" artifacts and does not migrate others`

// The pre-move artifact is a negative control: inspect refuses it with the
// named unsupported storage format as invalid input (status 2), in human and
// JSON form, without printing a report.
func TestInspectPreMoveArtifactIsRefusedAsUnsupportedStorageFormat(t *testing.T) {
	path := stageRetainedArtifact(t, preMoveWASMArtifact)
	for _, arguments := range [][]string{{"inspect", path}, {"inspect", "--json", path}} {
		t.Run(strings.Join(arguments[:len(arguments)-1], " "), func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			status := Run(arguments, &stdout, &stderr)
			want := "inspect " + path + ": " + preLineageRefusal + "\n"
			if status != 2 || stdout.Len() != 0 || stderr.String() != want {
				t.Fatalf("status=%d stdout=%q stderr=%q; want status 2 and %q", status, stdout.String(), stderr.String(), want)
			}
		})
	}
}

// Replay preflight refuses the pre-move artifact with the same named format
// as an incompatible replay artifact (status 2) before any toolchain is
// resolved.
func TestReplayPreMoveArtifactIsRefusedAsUnsupportedStorageFormat(t *testing.T) {
	var stdout, stderr bytes.Buffer
	status := Run([]string{"replay", "--verify-only", stageRetainedArtifact(t, preMoveWASMArtifact)}, &stdout, &stderr)
	want := "incompatible replay artifact: " + preLineageRefusal + "\n"
	if status != 2 || stdout.Len() != 0 || stderr.String() != want {
		t.Fatalf("status=%d stdout=%q stderr=%q; want status 2 and %q", status, stdout.String(), stderr.String(), want)
	}
}
