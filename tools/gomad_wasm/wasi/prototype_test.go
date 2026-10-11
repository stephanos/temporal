package wasi

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"go.temporal.io/server/tools/gomad3/choice"
	"go.temporal.io/server/tools/gomad3/deterministicio/readonlymount"
	"go.temporal.io/server/tools/gomad3/hostexec"
	runnerbackend "go.temporal.io/server/tools/gomad3/runner/backend"
	"go.temporal.io/server/tools/gomad_wasm/toolchain"
)

// The determinism prototype gates (fn-151 tasks 5-8) run only when
// GOMAD_WASM_PROTOTYPE_ROOT names an absolute evidence directory. Every guest
// uses the cooperative profile with a frozen clock (zero read-step), so time
// moves only when the guest is idle.

type prototypeRun struct {
	Guest, Variant                    string
	Seed                              uint64
	Instance                          int
	Mode                              string
	Termination, Message              string
	ExitCode                          *uint32
	WallNanos                         int64
	PeakMemoryBytes, FuelUsed         uint64
	ChoiceRecords                     int
	ChoiceSHA256, StdoutSHA256        string
	WASISHA256, NetworkSHA256         string
	VirtualNanos                      uint64
	Calls, NetworkEvents, StdoutBytes int
	TimeAdvances                      uint64
	AuthorizationToTerminalNanos      uint64
	Telemetry                         *Telemetry
	HelperResourceUsage               *hostexec.ResourceUsage
}

type prototypeReport struct {
	mu     sync.Mutex
	path   string
	Guests map[string]map[string]any
	Runs   []prototypeRun
}

var prototypeReports sync.Map

func prototypeRoot(t *testing.T) string {
	t.Helper()
	root := os.Getenv("GOMAD_WASM_PROTOTYPE_ROOT")
	if root == "" {
		t.Skip("set GOMAD_WASM_PROTOTYPE_ROOT to run the determinism prototype gates")
	}
	if !filepath.IsAbs(root) {
		t.Fatal("GOMAD_WASM_PROTOTYPE_ROOT must be absolute")
	}
	if err := os.MkdirAll(root, 0700); err != nil {
		t.Fatal(err)
	}
	return root
}

func prototypeReportFor(t *testing.T, name string) *prototypeReport {
	path := filepath.Join(prototypeRoot(t), name+"-report.json")
	value, _ := prototypeReports.LoadOrStore(path, &prototypeReport{path: path, Guests: map[string]map[string]any{}})
	return value.(*prototypeReport)
}

func (r *prototypeReport) guest(name string, fields map[string]any) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.Guests[name] = fields
	r.flush()
}

func (r *prototypeReport) add(run prototypeRun) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.Runs = append(r.Runs, run)
	r.flush()
}

func (r *prototypeReport) flush() {
	data, err := json.MarshalIndent(r, "", "  ")
	if err == nil {
		err = os.WriteFile(r.path, data, 0600)
	}
	if err != nil {
		panic(err)
	}
}

var prototypeOverlay struct {
	once           sync.Once
	path, identity string
	err            error
}

// prototypeToolchain derives the cooperative runtime overlay once per process.
func prototypeToolchain(t *testing.T) (compiler, overlay, identity string) {
	t.Helper()
	compiler = os.Getenv("GOMAD3_STOCK_GO")
	if !filepath.IsAbs(compiler) {
		t.Fatal("prototype gates require an absolute GOMAD3_STOCK_GO (Go 1.27.1)")
	}
	root := prototypeRoot(t)
	// Overlays cannot replace files beneath GOMODCACHE, so builds use a private GOROOT copy.
	goroot := filepath.Join(root, "goroot")
	prototypeOverlay.once.Do(func() {
		if _, err := os.Stat(filepath.Join(goroot, "bin", "go")); err != nil {
			output, err := exec.Command(compiler, "env", "GOROOT").Output()
			if err != nil {
				prototypeOverlay.err = err
				return
			}
			staging := goroot + ".staging"
			if err := os.RemoveAll(staging); err != nil {
				prototypeOverlay.err = err
				return
			}
			if err := os.CopyFS(staging, os.DirFS(strings.TrimSpace(string(output)))); err != nil {
				prototypeOverlay.err = err
				return
			}
			if err := os.Rename(staging, goroot); err != nil {
				prototypeOverlay.err = err
				return
			}
		}
		native, err := filepath.Abs("../../gomad3/toolchain/runtime/overlay/src/runtime")
		if err != nil {
			prototypeOverlay.err = err
			return
		}
		prototypeOverlay.path, prototypeOverlay.identity, prototypeOverlay.err = toolchain.BuildOverlay(goroot, native, filepath.Join(root, "overlay"))
	})
	if prototypeOverlay.err != nil {
		t.Fatal(prototypeOverlay.err)
	}
	return filepath.Join(goroot, "bin", "go"), prototypeOverlay.path, prototypeOverlay.identity
}

// prototypeBuild compiles a guest with the cooperative overlay; command is
// "build" or "test" (the latter produces a test binary with -c).
func prototypeBuild(t *testing.T, dir, command, target, output string, tags ...string) (module string, seconds float64) {
	t.Helper()
	compiler, overlay, _ := prototypeToolchain(t)
	args := []string{command}
	if command == "test" {
		args = append(args, "-c")
	}
	args = append(args, "-trimpath", "-overlay", overlay, "-o", output)
	if len(tags) > 0 {
		args = append(args, "-tags", strings.Join(tags, ","))
	}
	args = append(args, target)
	build := exec.Command(compiler, args...)
	build.Dir = dir
	build.Env = append(os.Environ(), "GOROOT="+filepath.Dir(filepath.Dir(compiler)), "GOOS=wasip1", "GOARCH=wasm", "GOEXPERIMENT=nogreenteagc", "GOWORK=off", "GOENV=off", "GOFLAGS=-mod=readonly", "GOTOOLCHAIN=local", "CGO_ENABLED=0")
	started := time.Now()
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build cooperative guest %s: %v\n%s", target, err, out)
	}
	return output, time.Since(started).Seconds()
}

func prototypeRequest(t *testing.T, module string, config Config, seed uint64, timeout time.Duration) Request {
	t.Helper()
	_, _, identity := prototypeToolchain(t)
	data, err := os.ReadFile(module)
	if err != nil {
		t.Fatal(err)
	}
	imports, pages := binaryInventory(t, data)
	helper := helperPath(t)
	config.Profile = CooperativeProfile
	config.Clock.ReadStepNanos = 0
	execution := choice.ExecutionIdentity{TargetSHA256: sha256.Sum256(data), ToolchainBuildKey: identity, GOOS: "wasip1", GOARCH: "wasm", ImplementationSHA256: sha256.Sum256([]byte(identity))}
	return Request{CompileCache: filepath.Join(prototypeRoot(t), "compile-cache"), DigestTranscript: true, HelperPath: helper, HelperSHA256: fileHash(t, helper), ModulePath: module, ModuleSHA256: fileHash(t, module), Engine: PinnedEngine(), Imports: imports, InitialMemoryPages: pages, MemoryBytes: 4 << 30, Fuel: 1 << 50, Config: config, Timeout: timeout, Runtime: &RuntimeControl{Seed: seed, Choice: &runnerbackend.ChoiceRequest{Mode: choice.ModeRecord, ExecutionIdentity: execution, Limit: choice.MaximumTraceBytes}}}
}

func digest(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

// Guests run concurrently in bounded pools: whole-program Temporal guests need
// GiBs of host memory each, fixture guests very little.
var (
	prototypeLight = make(chan struct{}, prototypeLimit("GOMAD_WASM_PROTOTYPE_PARALLEL", 4))
	prototypeHeavy = make(chan struct{}, prototypeLimit("GOMAD_WASM_PROTOTYPE_HEAVY", 4))
	prototypeWarm  sync.Mutex
)

func prototypeLimit(name string, fallback int) int {
	if value, err := strconv.Atoi(os.Getenv(name)); err == nil && value > 0 {
		return value
	}
	return fallback
}

// prototypeRunOne runs one fresh guest and records its evidence summary. It is
// safe to call from any goroutine.
func prototypeRunOne(report *prototypeReport, guest, variant string, instance int, request Request) (Result, error) {
	pool := prototypeLight
	if info, err := os.Stat(request.ModulePath); err == nil && info.Size() > 64<<20 {
		pool = prototypeHeavy
	}
	pool <- struct{}{}
	defer func() { <-pool }()
	name := fmt.Sprintf("%s-%s-seed%d-%d", guest, variant, request.Runtime.Seed, instance)
	if prefix := os.Getenv("GOMAD_WASM_PROTOTYPE_MIRROR"); prefix != "" {
		mirror, err := os.Create(prefix + "-" + name + ".stdout")
		if err != nil {
			return Result{}, err
		}
		defer func() { _ = mirror.Close() }()
		request.StdoutMirror = mirror
	}
	started := time.Now()
	result, err := Run(context.Background(), request)
	wall := time.Since(started)
	if err != nil {
		return Result{}, fmt.Errorf("%s: %w", name, err)
	}
	run := prototypeRun{Guest: guest, Variant: variant, Seed: request.Runtime.Seed, Instance: instance, Mode: fmt.Sprint(request.Runtime.Choice.Mode), Termination: result.Termination, Message: result.Message, Calls: int(result.Calls), TimeAdvances: result.TimeAdvances, ExitCode: result.ExitCode, WallNanos: wall.Nanoseconds(), PeakMemoryBytes: result.PeakMemoryBytes, FuelUsed: request.Fuel - result.FuelRemaining, ChoiceRecords: len(result.ChoiceTrace.Records), ChoiceSHA256: hex.EncodeToString(result.ChoiceTrace.SHA256[:]), StdoutSHA256: digest(result.Stdout), WASISHA256: hex.EncodeToString(result.TranscriptSHA256[:]), NetworkSHA256: digest(result.NetworkTranscript), StdoutBytes: len(result.Stdout), NetworkEvents: bytes.Count(result.NetworkTranscript, []byte{'\n'})}
	run.AuthorizationToTerminalNanos, run.Telemetry, run.HelperResourceUsage = result.AuthorizationToTerminalNanos, result.Telemetry, result.HelperResourceUsage
	if result.Termination == "exit" {
		run.VirtualNanos = uint64(int64FromTerminal(result.runtimeTerminal))
	}
	report.add(run)
	if instance == 0 || result.Termination != "exit" || result.ExitCode == nil || *result.ExitCode != 0 {
		prefix := filepath.Join(filepath.Dir(report.path), name)
		for suffix, data := range map[string][]byte{".stdout": result.Stdout, ".stderr": result.Stderr, ".network.jsonl": result.NetworkTranscript} {
			if err := os.WriteFile(prefix+suffix, data, 0600); err != nil {
				return Result{}, err
			}
		}
	}
	return result, nil
}

func prototypeExecute(t *testing.T, report *prototypeReport, guest, variant string, instance int, request Request) Result {
	t.Helper()
	result, err := prototypeRunOne(report, guest, variant, instance, request)
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func int64FromTerminal(terminal [16]byte) int64 {
	var value int64
	for _, b := range terminal[8:] {
		value = value<<8 | int64(b)
	}
	return value
}

// sameEvidence reports the first evidence stream that differs between runs.
func sameEvidence(a, b Result) string {
	for _, check := range []struct {
		name string
		x, y []byte
	}{{"stdout", a.Stdout, b.Stdout}, {"stderr", a.Stderr, b.Stderr}, {"wasi", a.TranscriptSHA256[:], b.TranscriptSHA256[:]}, {"choices", a.ChoiceTrace.Bytes, b.ChoiceTrace.Bytes}, {"network", a.NetworkTranscript, b.NetworkTranscript}, {"terminal", a.runtimeTerminal[:], b.runtimeTerminal[:]}} {
		if !bytes.Equal(check.x, check.y) {
			return check.name
		}
	}
	if a.Termination != b.Termination || (a.ExitCode == nil) != (b.ExitCode == nil) || a.ExitCode != nil && *a.ExitCode != *b.ExitCode {
		return "termination"
	}
	return ""
}

func requireExit(t *testing.T, label string, result Result, codes ...uint32) {
	t.Helper()
	if result.Termination != "exit" || result.ExitCode == nil || !slices.Contains(codes, *result.ExitCode) || !result.Reaped {
		code := "none"
		if result.ExitCode != nil {
			code = fmt.Sprint(*result.ExitCode)
		}
		tail := result.Stdout
		if len(tail) > 4000 {
			tail = tail[len(tail)-4000:]
		}
		t.Fatalf("%s: termination=%s exit=%s message=%s\nstdout tail:\n%s\nstderr:\n%s", label, result.Termination, code, result.Message, tail, result.Stderr)
	}
}

// prototypeDeterminism runs seeds x instances fresh guests, requires identical
// evidence per seed, and replays the first seed's tape exactly in another fresh
// guest. Recorded guests run concurrently after a large module's compiled code
// is cached. It returns the per-seed baselines.
func prototypeDeterminism(t *testing.T, report *prototypeReport, guest, variant string, request Request, seeds []uint64, instances int, codes ...uint32) map[uint64]Result {
	t.Helper()
	withSeed := func(seed uint64, tape *choice.ReplayPlan) Request {
		next := request
		runtime := *request.Runtime
		choiceRequest := *runtime.Choice
		if tape != nil {
			choiceRequest.Mode, choiceRequest.Tape = choice.ModeReplay, tape
		}
		runtime.Choice, runtime.Seed = &choiceRequest, seed
		next.Runtime = &runtime
		return next
	}
	// Compiling a whole-program module takes several GiB, so cold first runs of
	// large modules never overlap.
	if info, err := os.Stat(request.ModulePath); err == nil && info.Size() > 64<<20 {
		// A run that selects no tests exits right after startup and fills the cache.
		warm := withSeed(seeds[0], nil)
		warm.Config.Args = []string{"temporal.test", "-test.run=^$"}
		warm.Config.Network = nil
		prototypeWarm.Lock()
		result, err := prototypeRunOne(report, guest, variant+"-warm", 0, warm)
		prototypeWarm.Unlock()
		if err != nil || result.Termination == "infrastructure" {
			t.Fatalf("warm compile cache: %v %s %s", err, result.Termination, result.Message)
		}
	}
	type job struct {
		seed     uint64
		instance int
		result   Result
		err      error
	}
	var jobs []*job
	for _, seed := range seeds {
		for instance := range instances {
			jobs = append(jobs, &job{seed: seed, instance: instance})
		}
	}
	var wg sync.WaitGroup
	for _, j := range jobs {
		wg.Go(func() {
			j.result, j.err = prototypeRunOne(report, guest, variant, j.instance, withSeed(j.seed, nil))
		})
	}
	wg.Wait()
	baselines := map[uint64]Result{}
	for _, j := range jobs {
		if j.err != nil {
			t.Fatal(j.err)
		}
		requireExit(t, fmt.Sprintf("%s/%s seed %d instance %d", guest, variant, j.seed, j.instance), j.result, codes...)
		if j.instance == 0 {
			baselines[j.seed] = j.result
		}
	}
	for _, j := range jobs {
		if j.instance > 0 {
			if stream := sameEvidence(baselines[j.seed], j.result); stream != "" {
				t.Fatalf("%s/%s seed %d fresh instance %d diverged in %s", guest, variant, j.seed, j.instance, stream)
			}
		}
	}
	// The replay is another fresh guest forced through the first seed's tape.
	first := baselines[seeds[0]]
	tape, err := choice.ProjectReplayPlan(first.ChoiceTrace, request.Runtime.Choice.ExecutionIdentity)
	if err != nil {
		t.Fatal(err)
	}
	replayed := prototypeExecute(t, report, guest, variant+"-replay", 0, withSeed(seeds[0], &tape))
	if stream := sameEvidence(first, replayed); stream != "" {
		t.Fatalf("%s/%s exact replay diverged in %s: %s %s", guest, variant, stream, replayed.Termination, replayed.Message)
	}
	return baselines
}

func distinctChoiceTraces(baselines map[uint64]Result) int {
	seen := map[[32]byte]bool{}
	for _, result := range baselines {
		seen[result.ChoiceTrace.SHA256] = true
	}
	return len(seen)
}

func prototypeFixture(t *testing.T, name string) string {
	t.Helper()
	module := filepath.Join(prototypeRoot(t), "modules", name+".wasm")
	if err := os.MkdirAll(filepath.Dir(module), 0700); err != nil {
		t.Fatal(err)
	}
	dir, err := filepath.Abs(filepath.Join("..", "testdata", name))
	if err != nil {
		t.Fatal(err)
	}
	if name == "choices" {
		dir, err = filepath.Abs(filepath.Join("..", "toolchain", "testdata", name))
		if err != nil {
			t.Fatal(err)
		}
	}
	module, _ = prototypeBuild(t, dir, "build", ".", module)
	return module
}

func prototypeFixtureConfig() Config {
	config := testConfig()
	config.Limits.Calls = 1000000
	config.Limits.TranscriptBytes = 64 << 20
	return config
}

// TestPrototypeSchedulingDeterminism is the fn-151.5 gate: identical evidence
// for 4 seeds x 10 fresh instances, schedule diversity across seeds, exact
// replay, and rejection of tampered tapes.
func TestPrototypeSchedulingDeterminism(t *testing.T) {
	t.Parallel()
	report := prototypeReportFor(t, "scheduling")
	seeds := []uint64{0, 1000003, 2000006, 3000009}
	for _, fixture := range []string{"choices", "runtime_probes"} {
		t.Run(fixture, func(t *testing.T) {
			module := prototypeFixture(t, fixture)
			config := prototypeFixtureConfig()
			if fixture == "runtime_probes" {
				config.Environment = append(config.Environment, "EXTRA_ENTROPY=no_")
			}
			request := prototypeRequest(t, module, config, 0, 2*time.Minute)
			baselines := prototypeDeterminism(t, report, fixture, "strict", request, seeds, 10, 0, 17)
			distinct := distinctChoiceTraces(baselines)
			report.guest(fixture, map[string]any{"module_sha256": request.ModuleSHA256, "distinct_schedules": distinct, "seeds": seeds})
			// The small choices fixture checks tape rejection; runtime_probes must show schedule diversity.
			if fixture != "choices" {
				if distinct < 2 {
					t.Fatalf("%s: %d distinct schedules across %d seeds", fixture, distinct, len(seeds))
				}
				return
			}
			tape, err := choice.ProjectReplayPlan(baselines[0].ChoiceTrace, request.Runtime.Choice.ExecutionIdentity)
			if err != nil {
				t.Fatal(err)
			}
			for _, mutation := range []struct {
				name   string
				reason choice.DivergenceReason
			}{{"missing", choice.DivergenceTapeExhausted}, {"extra", choice.DivergenceTapeUnconsumed}, {"altered", choice.DivergenceAlternativeSet}} {
				plan := tape
				if mutation.name == "missing" {
					plan, err = tape.Prefix(0)
				} else {
					records := slices.Clone(baselines[0].ChoiceTrace.Records)
					target := slices.IndexFunc(records, func(record choice.Record) bool { return record.Kind == choice.KindRunnable })
					if mutation.name == "extra" {
						extra := records[target]
						extra.Ordinal = uint64(len(records))
						records = append(records, extra)
					} else {
						records[target].AlternativeSetDigest = sha256.Sum256([]byte("tampered"))
					}
					altered, buildErr := choice.BuildTrace(records, choice.TerminalComplete)
					if buildErr != nil {
						t.Fatal(buildErr)
					}
					plan, err = choice.ProjectReplayPlan(altered, request.Runtime.Choice.ExecutionIdentity)
				}
				if err != nil {
					t.Fatal(err)
				}
				tampered := request
				runtime := *request.Runtime
				choiceRequest := *runtime.Choice
				choiceRequest.Mode, choiceRequest.Tape = choice.ModeReplay, &plan
				runtime.Choice = &choiceRequest
				tampered.Runtime = &runtime
				result := prototypeExecute(t, report, fixture, "tamper-"+mutation.name, 0, tampered)
				if result.Termination != "divergence" || result.ChoiceDivergence == nil || result.ChoiceDivergence.Reason != mutation.reason {
					t.Fatalf("tampered tape %s: %s %s %+v", mutation.name, result.Termination, result.Message, result.ChoiceDivergence)
				}
			}
		})
	}
}

// TestPrototypeVirtualTime is the fn-151.6 fixture gate: hours of virtual
// time pass with exact deadlines while wall time stays small.
func TestPrototypeVirtualTime(t *testing.T) {
	t.Parallel()
	report := prototypeReportFor(t, "virtual-time")
	module := prototypeFixture(t, "time_probes")
	request := prototypeRequest(t, module, prototypeFixtureConfig(), 0, 2*time.Minute)
	baselines := prototypeDeterminism(t, report, "time_probes", "strict", request, []uint64{0, 1000003, 2000006}, 5, 0)
	for _, result := range baselines {
		for _, line := range []string{"runnable work keeps time frozen", "sleep 2h woke at 2h0m0s", "timer tie fired at 2h30m0s", "36 ticks took 6h0m0s", "cancellation after 5s"} {
			if !bytes.Contains(result.Stdout, []byte(line)) {
				t.Fatalf("missing %q in:\n%s", line, result.Stdout)
			}
		}
	}
	report.guest("time_probes", map[string]any{"module_sha256": request.ModuleSHA256})
}

// TestPrototypeNetwork is the fn-151.7 fixture gate: the simulated network
// delays, resets and partitions a TCP exchange reproducibly.
func TestPrototypeNetwork(t *testing.T) {
	t.Parallel()
	report := prototypeReportFor(t, "network")
	module := prototypeFixture(t, "network_probes")
	config := prototypeFixtureConfig()
	config.Args = []string{"network_probes", "expect-faults"}
	config.Network = &NetworkModel{Seed: 11, MaxDelayNanos: uint64(5 * time.Millisecond), Faults: []NetworkFault{
		{AtNanos: uint64(3 * time.Second), Action: "reset", Port: 9000},
		{AtNanos: uint64(6 * time.Second), Action: "partition", Port: 0},
		{AtNanos: uint64(8500 * time.Millisecond), Action: "heal", Port: 0},
	}}
	request := prototypeRequest(t, module, config, 0, 2*time.Minute)
	baselines := prototypeDeterminism(t, report, "network_probes", "faults", request, []uint64{0, 1000003, 2000006}, 5, 0)
	for seed, result := range baselines {
		for _, want := range []string{"reset", "dial refused"} {
			if !bytes.Contains(result.Stdout, []byte(want)) {
				t.Fatalf("seed %d: missing %q in:\n%s", seed, want, result.Stdout)
			}
		}
	}
	// A different network seed changes delays and therefore the virtual timeline.
	reseeded := request
	model := *config.Network
	model.Seed = 12
	reseeded.Config.Network = &model
	other := prototypeExecute(t, report, "network_probes", "reseeded", 0, reseeded)
	requireExit(t, "reseeded network", other, 0)
	if bytes.Equal(other.NetworkTranscript, baselines[0].NetworkTranscript) {
		t.Fatal("network seed did not change model decisions")
	}
	// Without a model the guest never consults the network import.
	plain := request
	plain.Config.Network = nil
	plain.Config.Args = []string{"network_probes"}
	clean := prototypeExecute(t, report, "network_probes", "no-model", 0, plain)
	requireExit(t, "network without model", clean, 0)
	if len(clean.NetworkTranscript) != 0 || !bytes.Contains(clean.Stdout, []byte("echoed 48 failures 0")) {
		t.Fatalf("model-free run: network=%d stdout=%s", len(clean.NetworkTranscript), clean.Stdout)
	}
	report.guest("network_probes", map[string]any{"module_sha256": request.ModuleSHA256, "model": config.Network})
}

// prototypeTemporalConfig mirrors the admitted one-box guest configuration.
func prototypeTemporalConfig(t *testing.T, root, cwd string, args ...string) Config {
	t.Helper()
	captured, err := readonlymount.CaptureReadOnlyMountInputs([]readonlymount.Mapping{{Source: filepath.Join(root, "schema", "sqlite"), Target: "/workspace/schema/sqlite"}}, readonlymount.DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	return Config{
		Profile: CooperativeProfile, WorkingDirectory: cwd,
		Args:                append([]string{"temporal.test", "-test.v", "-test.count=1", "-test.timeout=72h"}, args...),
		Environment:         []string{"PWD=" + cwd, "TEMPORAL_ROOT=/workspace", "TMPDIR=/tmp", "TZ=UTC", "TEMPORAL_TEST_LOG_LEVEL=warn"},
		WritableDirectories: []string{"/workspace", "/tmp", cwd},
		CapturedInputs:      captured,
		EntropyKey:          sha256.Sum256([]byte("gomad-wasm-prototype/v1")),
		Clock:               ClockPolicy{EpochNanos: 946684800000000000},
		Limits:              Limits{OutputBytes: 256 << 20, TranscriptBytes: 1 << 30, Calls: maxCooperativeCalls, Descriptors: 4096, Files: 100000, FilesystemBytes: 512 << 20, PendingEvents: 4096},
	}
}

// prototypeFaults resets every connection during the workflow's first sleep
// and partitions everything for ten seconds during its second sleep.
var prototypeFaults = []NetworkFault{
	{AtNanos: uint64(40 * time.Second), Action: "reset"},
	{AtNanos: uint64(100 * time.Second), Action: "partition"},
	{AtNanos: uint64(110 * time.Second), Action: "heal"},
}

// TestPrototypeTemporal runs the fn-151.6/.7/.8 one-box gates: a workflow with
// timers, retries and network faults, and an unchanged functional suite.
func TestPrototypeTemporal(t *testing.T) {
	t.Parallel()
	root, err := filepath.Abs("../../..")
	if err != nil {
		t.Fatal(err)
	}
	instances := prototypeLimit("GOMAD_WASM_PROTOTYPE_INSTANCES", 1)
	only := os.Getenv("GOMAD_WASM_PROTOTYPE_ONLY")
	selected := func(name string) bool { return only == "" || strings.Contains(","+only+",", ","+name+",") }
	tags := []string{"disable_grpc_modules", "test_dep", "sqlite3_dotlk"}
	persistence := []string{"-persistenceType=sql", "-persistenceDriver=sqlite"}
	modules := filepath.Join(prototypeRoot(t), "modules")
	if err := os.MkdirAll(modules, 0700); err != nil {
		t.Fatal(err)
	}
	seeds := []uint64{0, 1000003}

	t.Run("workflow", func(t *testing.T) {
		if !selected("workflow") {
			t.Skip("not selected")
		}
		t.Parallel()
		report := prototypeReportFor(t, "temporal-workflow")
		staged := filepath.Join(".tmp", "wasm-backend-temporal", "prototype-temporal")
		if err := os.RemoveAll(filepath.Join(root, staged)); err != nil {
			t.Fatal(err)
		}
		if err := temporalStageFixture(root, filepath.Join("tools", "gomad_wasm", "testdata", "prototype_temporal"), root, staged); err != nil {
			t.Fatal(err)
		}
		module, seconds := prototypeBuild(t, root, "test", "./"+staged, filepath.Join(modules, "prototype-temporal.wasm"), tags...)
		config := prototypeTemporalConfig(t, root, "/workspace/prototype", append([]string{"-test.run=^TestPrototypeWorkflow$"}, persistence...)...)
		config.Network = &NetworkModel{Seed: 21, MaxDelayNanos: uint64(2 * time.Millisecond), Faults: prototypeFaults}
		request := prototypeRequest(t, module, config, 0, 30*time.Minute)
		baselines := prototypeDeterminism(t, report, "temporal", "workflow", request, seeds, instances, 0)
		for seed, result := range baselines {
			if !bytes.Contains(result.Stdout, []byte("--- PASS: TestPrototypeWorkflow")) {
				t.Fatalf("seed %d: workflow did not pass", seed)
			}
		}
		result := baselines[seeds[0]]
		resets := bytes.Count(result.NetworkTranscript, []byte(`"action":"reset"`))
		refusals := bytes.Count(result.NetworkTranscript, []byte(`"action":"refuse"`))
		info, _ := os.Stat(module)
		report.guest("prototype-temporal", map[string]any{"module_sha256": request.ModuleSHA256, "module_bytes": info.Size(), "build_seconds": seconds, "model": config.Network, "resets": resets, "refusals": refusals, "distinct_schedules": distinctChoiceTraces(baselines)})
		if resets == 0 {
			t.Fatal("network faults never reset a connection")
		}
		if distinctChoiceTraces(baselines) < 2 {
			t.Fatal("seeds produced identical schedules")
		}
	})

	// The default unchanged suite is the small gomadfunctional package. The
	// ./tests suites (smoke-workflow, smoke-activity, smoke-child, smoke-update,
	// smoke-timers) compile a 338 MB module that needs about 6 GiB of free host
	// memory once, so they run only when GOMAD_WASM_PROTOTYPE_ONLY selects them.
	type suite struct{ pkg, cwd, run string }
	smoke := map[string]suite{
		"frontend": {"./tests/gomadfunctional", "/workspace/tests/gomadfunctional", "^TestFrontendSystemInfo$"},
		"workflow": {"./tests", "/workspace/tests", "^TestWorkflowTypeEncodingSuite$/^TestPlainASCII$/^Succeeds$"},
		"activity": {"./tests", "/workspace/tests", "^TestActivityTestSuite$"},
		"child":    {"./tests", "/workspace/tests", "^TestChildWorkflowSuite$"},
		"update":   {"./tests", "/workspace/tests", "^TestWorkflowUpdateSuite$"},
		"timers":   {"./tests", "/workspace/tests", "^TestUserTimersTestSuite$"},
	}
	var suites []string
	for name := range smoke {
		if only == "" && name == "frontend" || only != "" && (selected("smoke") || selected("smoke-"+name)) {
			suites = append(suites, name)
		}
	}
	slices.Sort(suites)
	if len(suites) == 0 {
		return
	}
	t.Run("smoke", func(t *testing.T) {
		t.Parallel()
		report := prototypeReportFor(t, "temporal-smoke")
		for _, name := range suites {
			t.Run(name, func(t *testing.T) {
				selected := smoke[name]
				module, seconds := prototypeBuild(t, root, "test", selected.pkg, filepath.Join(modules, strings.ReplaceAll(strings.TrimPrefix(selected.pkg, "./"), "/", "-")+".wasm"), tags...)
				info, _ := os.Stat(module)
				report.guest(name, map[string]any{"package": selected.pkg, "module_sha256": fileHash(t, module), "module_bytes": info.Size(), "build_seconds": seconds})
				config := prototypeTemporalConfig(t, root, selected.cwd, append([]string{"-test.run=" + selected.run}, persistence...)...)
				request := prototypeRequest(t, module, config, 0, 30*time.Minute)
				baselines := prototypeDeterminism(t, report, "tests", "smoke-"+name, request, seeds, instances, 0)
				for seed, result := range baselines {
					if !bytes.Contains(result.Stdout, []byte("\nPASS\n")) {
						t.Fatalf("seed %d: suite %s did not pass", seed, name)
					}
				}
				if distinctChoiceTraces(baselines) < 2 {
					t.Fatalf("suite %s seeds produced identical schedules", name)
				}
			})
		}
	})
}
