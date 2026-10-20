package sweep

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"go.temporal.io/server/tools/gomad/internal/backends/wasm/backend"
	"go.temporal.io/server/tools/gomad/internal/backends/wasm/installation"
	"go.temporal.io/server/tools/gomad/internal/backends/wasm/wasi"
	captured "go.temporal.io/server/tools/gomad/internal/evidence/artifact/capture"
)

// fixture is a fake Temporal checkout plus executables and a fake executor
// that answers go list, go test -list, go env, git and gomad.
type fixture struct {
	options Options
	mu      sync.Mutex
	// listExtra adds names to `go test -list` output to force a mismatch.
	listExtra []string
	// outcomes maps test name to the explore behavior for it.
	outcomes map[string]func(attempt int, artifacts string) (Completed, map[string]ArtifactFacts)
	attempts map[string]int
	installs []string
	explores []string
	warmups  []string
	facts    map[string]ArtifactFacts
}

const checkoutRel = "temporal"

func newFixture(t *testing.T) *fixture {
	t.Helper()
	root := t.TempDir()
	checkout := filepath.Join(root, checkoutRel)
	write := func(path, content string, mode os.FileMode) {
		t.Helper()
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), mode); err != nil {
			t.Fatal(err)
		}
	}
	write(filepath.Join(checkout, "tests", "testcore", "core.go"), "package testcore\n", 0o600)
	write(filepath.Join(checkout, "schema", "sqlite", "schema.sql"), "CREATE TABLE x (id INTEGER);\n", 0o600)
	write(filepath.Join(checkout, "tests", "alpha", "alpha_test.go"), `package alpha
import "testing"
func TestMain(m *testing.M) {}
func TestAlpha(t *testing.T) {}
func TestBeta(t *testing.T) {}
func Testlower(t *testing.T) {}
func helper(t *testing.T) {}
type S struct{}
func (S) HelperMethod(t *testing.T) {}
func BenchmarkX(b *testing.B) {}
`, 0o600)
	write(filepath.Join(checkout, "tests", "alpha", "unix_test.go"), "package alpha_test\nimport \"testing\"\nfunc TestUnixOnly(t *testing.T) {}\n", 0o600)
	write(filepath.Join(checkout, "tests", "gomadfunctional", "frontend_test.go"), "package gomadfunctional\nimport \"testing\"\nfunc TestFrontendSystemInfo(t *testing.T) {}\n", 0o600)
	for _, name := range []string{"gomad", "go", "gomad-wasmhost"} {
		write(filepath.Join(root, "bin", name), "#!/bin/sh\nexit 0\n# "+name+"\n", 0o700)
	}
	if err := os.MkdirAll(filepath.Join(root, "runtimesource"), 0o700); err != nil {
		t.Fatal(err)
	}
	return &fixture{
		options: Options{
			Checkout: checkout, Gomad: filepath.Join(root, "bin", "gomad"), Compiler: filepath.Join(root, "bin", "go"),
			Helper: filepath.Join(root, "bin", "gomad-wasmhost"), RuntimeRoot: filepath.Join(root, "runtimesource"),
			ConfigSource: ConfigThrowaway, Packages: []string{"./tests/..."}, PerTestTimeout: 10 * time.Minute,
			PrepareTimeout: 30 * time.Minute, OverallBudget: 24 * time.Hour, RetainBytes: 1 << 20,
			Out: filepath.Join(root, "out"), Work: filepath.Join(root, "work"), Artifacts: filepath.Join(root, "retained"),
			Environ: []string{"HOME=" + root, "PATH=/usr/bin"},
		},
		outcomes: map[string]func(int, string) (Completed, map[string]ArtifactFacts){},
		attempts: map[string]int{}, facts: map[string]ArtifactFacts{},
	}
}

func (f *fixture) listJSON(guest bool) string {
	type listed struct {
		ImportPath, Dir           string
		TestGoFiles, XTestGoFiles []string
	}
	alpha := listed{ImportPath: "go.temporal.io/server/tests/alpha", Dir: filepath.Join(f.options.Checkout, "tests", "alpha"), TestGoFiles: []string{"alpha_test.go"}, XTestGoFiles: []string{"unix_test.go"}}
	if guest {
		alpha.XTestGoFiles = nil // unix_test.go is excluded under GOOS=wasip1
	}
	var out []string
	for _, value := range []listed{
		alpha,
		{ImportPath: "go.temporal.io/server/tests/gomadfunctional", Dir: filepath.Join(f.options.Checkout, "tests", "gomadfunctional"), TestGoFiles: []string{"frontend_test.go"}},
		{ImportPath: "go.temporal.io/server/tests/testcore", Dir: filepath.Join(f.options.Checkout, "tests", "testcore")},
	} {
		encoded, _ := json.Marshal(value)
		out = append(out, string(encoded))
	}
	return strings.Join(out, "\n")
}

func flagValue(args []string, name string) string {
	for index, arg := range args {
		if arg == name && index+1 < len(args) {
			return args[index+1]
		}
		if value, ok := strings.CutPrefix(arg, name+"="); ok {
			return value
		}
	}
	return ""
}

func (f *fixture) Exec(_ context.Context, command Command) Completed {
	f.mu.Lock()
	defer f.mu.Unlock()
	args := command.Args
	ok := func(stdout string) Completed { return Completed{Stdout: []byte(stdout), Exit: 0, Elapsed: time.Second} }
	switch {
	case command.Path == f.options.Compiler && args[0] == "list":
		return ok(f.listJSON(slices.Contains(command.Env, "GOOS=wasip1")))
	case command.Path == f.options.Compiler && args[0] == "test":
		pattern := args[len(args)-1]
		var names []string
		switch pattern {
		case "./tests/alpha":
			names = []string{"TestAlpha", "TestBeta", "TestUnixOnly"}
		case "./tests/gomadfunctional":
			names = []string{"TestFrontendSystemInfo"}
		}
		names = append(names, f.listExtra...)
		return ok(strings.Join(names, "\n") + "\nok  \t" + pattern + "\t0.010s\n")
	case command.Path == f.options.Compiler && args[0] == "env":
		return ok("go1.27.1\n")
	case command.Path == "git" && slices.Contains(args, "rev-parse"):
		return ok("e50c6476fb0000000000000000000000000000aa\n")
	case command.Path == "git":
		return ok("")
	case command.Path == f.options.Gomad && args[0] == "install":
		f.installs = append(f.installs, strings.Join(args, " "))
		config, err := installation.ReadConfig(flagValue(args, "--config"))
		if err != nil {
			return Completed{Exit: 2, Stderr: []byte(err.Error())}
		}
		manifest := installation.Manifest{Schema: installation.Schema, RuntimeImplementationSHA256: "sha256:runtime", Options: backend.Options{
			HelperSHA256: mustDigest(f.options.Helper), MemoryBytes: 4 << 30, Fuel: 1 << 50, ModelSHA256: "sha256:model", Engine: wasi.EngineIdentity{Name: "wasmtime", Version: "1", Configuration: "c", HostOS: "linux", HostArch: "aarch64"}, Config: config,
		}}
		encoded, _ := json.Marshal(manifest)
		if err := os.WriteFile(flagValue(args, "--output"), encoded, 0o600); err != nil {
			return Completed{Exit: 3, Stderr: []byte(err.Error())}
		}
		return ok("gomad: installed\n")
	case command.Path == f.options.Gomad && slices.Contains(args, "explore"):
		run := flagValue(args, "-test.run")
		if run == "^$" {
			f.warmups = append(f.warmups, strings.Join(args, " "))
			return Completed{Exit: 0, Elapsed: time.Second, Stdout: exploreOutput(Event{Type: "result", Classification: "success"})}
		}
		f.explores = append(f.explores, strings.Join(args, " "))
		name := strings.TrimSuffix(strings.TrimPrefix(run, "^"), "$")
		f.attempts[name]++
		behavior := f.outcomes[name]
		if behavior == nil {
			behavior = passing(name)
		}
		artifacts := flagValue(args, "--artifacts")
		completed, facts := behavior(f.attempts[name], artifacts)
		for path, value := range facts {
			if err := os.MkdirAll(path, 0o700); err != nil {
				return Completed{Exit: -1, Err: err}
			}
			if err := os.WriteFile(filepath.Join(path, "module.wasm"), make([]byte, 1000), 0o600); err != nil {
				return Completed{Exit: -1, Err: err}
			}
			f.facts[path] = value
		}
		return completed
	}
	return Completed{Exit: -1, Err: fmt.Errorf("unexpected command %s %v", command.Path, args)}
}

func mustDigest(path string) string {
	digest, err := fileSHA256(path)
	if err != nil {
		panic(err)
	}
	return digest
}

func (f *fixture) readArtifact(path string) ArtifactFacts {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.facts[path]
}

func exploreOutput(values ...Event) []byte {
	var lines []string
	for _, value := range values {
		value.Schema = "gomad.explore-event/v3"
		encoded, _ := json.Marshal(value)
		lines = append(lines, string(encoded))
	}
	return []byte(strings.Join(lines, "\n") + "\n")
}

func passing(name string) func(int, string) (Completed, map[string]ArtifactFacts) {
	return func(_ int, artifacts string) (Completed, map[string]ArtifactFacts) {
		path := filepath.Join(artifacts, "v1", "success")
		return Completed{Exit: 0, Elapsed: time.Second, Stdout: exploreOutput(Event{Type: "result", Classification: "success"}, Event{Type: "artifact", Classification: "success", Path: path})},
			map[string]ArtifactFacts{path: {Stdout: []byte("--- PASS: " + name + " (1.00s)\nPASS\n"), ModuleSHA256: "sha256:module"}}
	}
}

func failing(message string) func(int, string) (Completed, map[string]ArtifactFacts) {
	return func(_ int, artifacts string) (Completed, map[string]ArtifactFacts) {
		path := filepath.Join(artifacts, "v1", "failure")
		return Completed{Exit: 1, Elapsed: time.Second, Stdout: exploreOutput(Event{Type: "result", Classification: "target_failure", Failures: 1}, Event{Type: "artifact", Classification: "target_failure", Path: path})},
			map[string]ArtifactFacts{path: {Kind: "target_failure", Reason: "panic_or_runtime_fatal", Stderr: []byte("panic: " + message + "\n")}}
	}
}

func (f *fixture) sweeper() *Sweeper {
	return &Sweeper{Options: f.options, Executor: f, ReadArtifact: f.readArtifact, Command: []string{"wasmsweep", "test"}, HarnessSHA256: "sha256:harness-under-test"}
}

func TestOptionsRefuseInheritedVariablesAndUnboundedRuns(t *testing.T) {
	t.Parallel()
	valid := newFixture(t).options
	if err := valid.Validate(); err != nil {
		t.Fatalf("valid options refused: %v", err)
	}
	for _, mutation := range []struct {
		name   string
		change func(*Options)
		want   string
	}{
		{"gomad-variable", func(o *Options) { o.Environ = append(o.Environ, "GOMAD_WASM_INSTALLATION=/x") }, "GOMAD_WASM_INSTALLATION"},
		{"gomad-empty-variable", func(o *Options) { o.Environ = append(o.Environ, "GOMAD_TEMPORAL_CONFIG=") }, "GOMAD_TEMPORAL_CONFIG"},
		{"zero-per-test", func(o *Options) { o.PerTestTimeout = 0 }, "--per-test-timeout must be a positive bound"},
		{"zero-prepare", func(o *Options) { o.PrepareTimeout = 0 }, "--prepare-timeout must be a positive bound"},
		{"zero-overall", func(o *Options) { o.OverallBudget = 0 }, "--overall-budget must be a positive bound"},
		{"negative-overall", func(o *Options) { o.OverallBudget = -time.Hour }, "--overall-budget must be a positive bound"},
		{"huge-per-test", func(o *Options) { o.PerTestTimeout = MaximumPerTestTimeout + time.Second }, "exceeds"},
		{"huge-overall", func(o *Options) { o.OverallBudget = MaximumOverallBudget + time.Second }, "exceeds"},
		{"budget-fits-nothing", func(o *Options) { o.OverallBudget = o.PerTestTimeout }, "cannot fit one test"},
		{"relative-checkout", func(o *Options) { o.Checkout = "temporal" }, "absolute clean path"},
		{"missing-gomad", func(o *Options) { o.Gomad = "" }, "--gomad is required"},
		{"work-in-checkout", func(o *Options) { o.Work = filepath.Join(o.Checkout, "work") }, "--work must be outside --checkout"},
		{"artifacts-in-checkout", func(o *Options) { o.Artifacts = filepath.Join(o.Checkout, "a") }, "--artifacts must be outside --checkout"},
		{"config-source", func(o *Options) { o.ConfigSource = "" }, "--config-source"},
		{"no-packages", func(o *Options) { o.Packages = nil }, "--packages is required"},
		{"package-outside-tests", func(o *Options) { o.Packages = []string{"./service/..."} }, "not a clean pattern under ./tests"},
		{"package-escapes-tests", func(o *Options) { o.Packages = []string{"./tests/../service"} }, "not a clean pattern under ./tests"},
		{"work-overlaps-out", func(o *Options) { o.Work = filepath.Join(o.Out, "work") }, "--work must not overlap"},
		{"artifacts-in-work", func(o *Options) { o.Artifacts = filepath.Join(o.Work, "a") }, "--work must not overlap"},
		{"negative-max-tests", func(o *Options) { o.MaxTests = -1 }, "--max-tests"},
	} {
		t.Run(mutation.name, func(t *testing.T) {
			t.Parallel()
			options := valid
			options.Environ = slices.Clone(valid.Environ)
			mutation.change(&options)
			err := options.Validate()
			if !errors.Is(err, ErrRefused) || !strings.Contains(err.Error(), mutation.want) {
				t.Fatalf("got %v, want refusal containing %q", err, mutation.want)
			}
		})
	}
}

func TestEnumerationParsesTopLevelTestsAndReconciles(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	packages, err := Enumerate(t.Context(), f, f.options, GuestTags(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(packages) != 2 || packages[0].Pattern != "./tests/alpha" || packages[1].Pattern != "./tests/gomadfunctional" {
		t.Fatalf("packages %+v", packages)
	}
	alpha := packages[0]
	if !slices.Equal(alpha.Tests, []string{"TestAlpha", "TestBeta", "TestUnixOnly"}) || alpha.ListCount != 3 || alpha.ParsedCount != 3 {
		t.Fatalf("alpha %+v: TestMain, Testlower, methods and benchmarks must be excluded", alpha)
	}
	if alpha.GuestExcluded["TestUnixOnly"] != "unix_test.go" || len(alpha.GuestExcluded) != 1 {
		t.Fatalf("guest-excluded %v", alpha.GuestExcluded)
	}
}

func TestEnumerationMismatchFails(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	f.listExtra = []string{"TestGeneratedElsewhere"}
	_, err := Enumerate(t.Context(), f, f.options, GuestTags(), nil)
	if !errors.Is(err, ErrEnumerationMismatch) || !strings.Contains(err.Error(), "TestGeneratedElsewhere") {
		t.Fatalf("got %v, want enumeration mismatch naming the extra test", err)
	}
	if _, err := f.sweeper().Run(t.Context()); !errors.Is(err, ErrEnumerationMismatch) {
		t.Fatalf("sweep ran despite a mismatch: %v", err)
	}
	if len(f.explores) != 0 {
		t.Fatalf("explored %d tests after a mismatch", len(f.explores))
	}
}

func TestParseTestListDropsSummary(t *testing.T) {
	t.Parallel()
	got := parseTestList([]byte("TestB\nTestA\nok  \tgo.temporal.io/server/tests\t0.5s\n\n"))
	if !slices.Equal(got, []string{"TestA", "TestB"}) {
		t.Fatalf("got %v", got)
	}
}

func TestSweepClassifiesRetainsResumesAndVerifies(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	f.outcomes["TestBeta"] = failing("listen tcp 127.0.0.1:4000: address in use")
	f.outcomes["TestFrontendSystemInfo"] = func(attempt int, artifacts string) (Completed, map[string]ArtifactFacts) {
		if attempt == 1 {
			return Completed{Exit: 3, Stdout: exploreOutput(Event{Type: "error", Classification: "runner_failure", Message: "gomad Runner/host failure: target_supervision: backend infrastructure: helper exited"})}, nil
		}
		return passing("TestFrontendSystemInfo")(attempt, artifacts)
	}
	findings, err := f.sweeper().Run(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if len(findings) != 0 {
		t.Fatalf("verify findings: %v", findings)
	}
	sidecar := readSidecar(t, f.options.Out)
	rows := map[string]TestEntry{}
	for _, row := range sidecar.Rows {
		rows[row.Name] = row
	}
	if rows["TestAlpha"].Class != ClassPass || rows["TestAlpha"].ArtifactRetained {
		t.Fatalf("TestAlpha %+v: pass without retention", rows["TestAlpha"])
	}
	beta := rows["TestBeta"]
	if beta.Class != ClassGuestFailure || !strings.Contains(beta.Reason, "address in use") || !beta.ArtifactRetained {
		t.Fatalf("TestBeta %+v", beta)
	}
	if _, err := os.Stat(beta.ArtifactPath); err != nil || !strings.HasPrefix(beta.ArtifactPath, f.options.Artifacts) {
		t.Fatalf("retained exemplar %q: %v", beta.ArtifactPath, err)
	}
	if rows["TestUnixOnly"].Class != ClassBuildFailure || !strings.Contains(rows["TestUnixOnly"].Reason, "guest-file-set") {
		t.Fatalf("TestUnixOnly %+v", rows["TestUnixOnly"])
	}
	frontend := rows["TestFrontendSystemInfo"]
	if frontend.Class != ClassPass || frontend.Attempts != 2 || !strings.Contains(frontend.FirstAttemptReason, "helper exited") {
		t.Fatalf("infrastructure retry: %+v", frontend)
	}
	if sidecar.Candidate.ConfigSource != ConfigThrowaway || sidecar.Candidate.GitSHA == "" || sidecar.Candidate.ModelSHA256 != "sha256:model" {
		t.Fatalf("candidate %+v", sidecar.Candidate)
	}
	if sidecar.ByClass[ClassPass] != 2 || sidecar.ByClass[ClassGuestFailure] != 1 || sidecar.ByClass[ClassBuildFailure] != 1 {
		t.Fatalf("aggregates %v", sidecar.ByClass)
	}
	for _, explore := range f.explores {
		for _, want := range []string{"--seeds=0", "--parallel=1", "--choices", "--execution-timeout=10m0s", "--overall-timeout=40m0s", "--output-limit=256MiB", "--build-tag=gomad_wasm_slim", "--env=TEMPORAL_ROOT=/workspace", "--json", "-test.timeout=72h"} {
			if !strings.Contains(explore, want) {
				t.Fatalf("explore %q lacks %q", explore, want)
			}
		}
	}
	// One warm-up per package fills the engine cache under the prepare
	// allowance before the first recorded row.
	if len(f.warmups) != 2 || !strings.Contains(f.warmups[0], "--execution-timeout=30m0s") {
		t.Fatalf("warm-ups %v", f.warmups)
	}
	if sidecar.Candidate.HarnessSHA256 == "" || sidecar.Candidate.GuestConfigSHA256 == "" || sidecar.Candidate.MemoryBytes != 4<<30 || len(sidecar.Candidate.ExploreFlags) == 0 {
		t.Fatalf("candidate lacks harness identity: %+v", sidecar.Candidate)
	}
	if rows["TestAlpha"].ConfigSource != ConfigThrowaway || rows["TestAlpha"].ModuleSHA256 != "sha256:module" {
		t.Fatalf("row labels %+v", rows["TestAlpha"])
	}
	// Each package's scratch, caches and unretained run roots are removed.
	if entries, err := os.ReadDir(filepath.Join(f.options.Work, "packages")); err != nil || len(entries) != 0 {
		t.Fatalf("work not cleaned: %v %v", entries, err)
	}
	// A rerun on the same candidate runs nothing and keeps one row per test.
	explored := len(f.explores)
	findings, err = f.sweeper().Run(t.Context())
	if err != nil || len(findings) != 0 {
		t.Fatalf("resume: %v %v", err, findings)
	}
	if len(f.explores) != explored {
		t.Fatalf("resume re-ran %d tests", len(f.explores)-explored)
	}
	if again := readSidecar(t, f.options.Out); len(again.Rows) != 4 || len(again.Hosts) != 1 {
		t.Fatalf("resumed sidecar rows=%d hosts=%v", len(again.Rows), again.Hosts)
	}
	// A different candidate (here: another seed) runs everything again.
	f.options.Seed = 7
	if _, err := f.sweeper().Run(t.Context()); err != nil {
		t.Fatal(err)
	}
	if len(f.explores) != explored+3 {
		t.Fatalf("new candidate explored %d, want 3", len(f.explores)-explored)
	}
}

func TestSweepRecordsBudgetCutoffAsNotRunAndResumesIt(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	f.options.PerTestTimeout, f.options.PrepareTimeout, f.options.OverallBudget = time.Minute, time.Minute, 5*time.Minute
	sweeper := f.sweeper()
	now := time.Unix(1_000_000, 0)
	sweeper.Now = func() time.Time { return now }
	f.outcomes["TestAlpha"] = func(attempt int, artifacts string) (Completed, map[string]ArtifactFacts) {
		now = now.Add(4 * time.Minute) // the first run consumes most of the budget
		return passing("TestAlpha")(attempt, artifacts)
	}
	if findings, err := sweeper.Run(t.Context()); err != nil || len(findings) != 0 {
		t.Fatalf("%v %v", err, findings)
	}
	sidecar := readSidecar(t, f.options.Out)
	notRun := 0
	for _, row := range sidecar.Rows {
		if row.Class == ClassNotRun {
			notRun++
			if !strings.Contains(row.Reason, "budget") {
				t.Fatalf("not-run without budget reason: %+v", row)
			}
		}
		if row.Class == ClassTimeout {
			t.Fatalf("budget cutoff recorded as a failure: %+v", row)
		}
	}
	if notRun != 2 { // TestBeta and TestFrontendSystemInfo; TestUnixOnly needs no run
		t.Fatalf("not-run %d, want 2: %+v", notRun, sidecar.Rows)
	}
	resumed := f.sweeper()
	if _, err := resumed.Run(t.Context()); err != nil {
		t.Fatal(err)
	}
	if final := readSidecar(t, f.options.Out); final.ByClass[ClassNotRun] != 0 || final.ByClass[ClassPass] != 3 {
		t.Fatalf("resume did not run the not-run tests: %v", final.ByClass)
	}
}

func TestMaxTestsCapsEachPackageInSortedOrder(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	f.options.MaxTests = 1
	if _, err := f.sweeper().Run(t.Context()); err != nil {
		t.Fatal(err)
	}
	if len(f.explores) != 2 || !strings.Contains(f.explores[0], "^TestAlpha$") || !strings.Contains(f.explores[1], "^TestFrontendSystemInfo$") {
		t.Fatalf("explores %v", f.explores)
	}
	sidecar := readSidecar(t, f.options.Out)
	if sidecar.ByClass[ClassNotRun] != 2 {
		t.Fatalf("capped tests must be not-run rows: %v", sidecar.ByClass)
	}
	for _, row := range sidecar.Rows {
		if row.ConfigSource != ConfigThrowaway {
			t.Fatalf("row lacks configuration source: %+v", row)
		}
	}
}

func TestTemporalInstallHookUsesFn220Flags(t *testing.T) {
	t.Parallel()
	options := newFixture(t).options
	options.ConfigSource = ConfigTemporalInstall
	args := strings.Join(installArgs(options, "./tests", dirsFor(options.Work, "./tests")), " ")
	if !strings.Contains(args, "--temporal-checkout "+options.Checkout) || !strings.Contains(args, "--temporal-package ./tests") || strings.Contains(args, "--config") || strings.Contains(args, "--profile") {
		t.Fatalf("install args %s", args)
	}
	options.ConfigSource = ConfigThrowaway
	args = strings.Join(installArgs(options, "./tests", dirsFor(options.Work, "./tests")), " ")
	if !strings.Contains(args, "--config ") || !strings.Contains(args, "--memory-bytes 4294967296") || strings.Contains(args, "--temporal-") {
		t.Fatalf("install args %s", args)
	}
}

func TestThrowawayConfigIsAValidCooperativeGuest(t *testing.T) {
	t.Parallel()
	for _, pattern := range []string{"./tests", "./tests/gomadfunctional"} {
		config := ThrowawayConfig(pattern, captured.CapturedInputs{})
		if _, err := wasi.NewEnvironment(config); err != nil {
			t.Fatalf("%s: %v", pattern, err)
		}
		if config.WorkingDirectory != GuestDirectory(pattern) || config.NetworkV2 != nil || config.Profile != wasi.CooperativeProfile {
			t.Fatalf("%s: %+v", pattern, config)
		}
	}
}

func readSidecar(t *testing.T, out string) Sidecar {
	t.Helper()
	data, err := os.ReadFile(SidecarPath(out))
	if err != nil {
		t.Fatal(err)
	}
	var sidecar Sidecar
	if err := json.Unmarshal(data, &sidecar); err != nil {
		t.Fatal(err)
	}
	return sidecar
}

func TestInterruptedSweepRecordsNoRowForTheRunningTest(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	ctx, cancel := context.WithCancel(t.Context())
	f.outcomes["TestAlpha"] = func(_ int, _ string) (Completed, map[string]ArtifactFacts) {
		cancel()
		return Completed{Exit: -1, Err: errors.New("signal: killed")}, nil
	}
	if _, err := f.sweeper().Run(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("got %v, want cancellation", err)
	}
	rows, err := ReadJournal(JournalPath(f.options.Out))
	if err != nil {
		t.Fatal(err)
	}
	for _, row := range rows {
		if row.Kind == "test" {
			t.Fatalf("interrupted run recorded %+v", row.Test)
		}
	}
	if f.attempts["TestAlpha"] != 1 {
		t.Fatalf("interrupted test retried %d times", f.attempts["TestAlpha"])
	}
}
