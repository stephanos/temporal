package wasi

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"maps"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"runtime/pprof"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	runnerbackend "go.temporal.io/server/tools/gomad/internal/backends"
	"go.temporal.io/server/tools/gomad/internal/backends/target"
	"go.temporal.io/server/tools/gomad/internal/backends/wasm/internal/sqlitebusy"
	"go.temporal.io/server/tools/gomad/internal/backends/wasm/internal/stockgo"
	"go.temporal.io/server/tools/gomad/internal/backends/wasm/toolchain"
	captured "go.temporal.io/server/tools/gomad/internal/evidence/artifact/capture"
	hostexec "go.temporal.io/server/tools/gomad/internal/host/exec"
	hostfs "go.temporal.io/server/tools/gomad/internal/host/fs"
	"go.temporal.io/server/tools/gomad/internal/model/choice"
	"go.temporal.io/server/tools/gomad/internal/model/clock"
	gomad3sim "go.temporal.io/server/tools/gomad/internal/model/cluster"
	netmodel "go.temporal.io/server/tools/gomad/internal/model/network"
	"go.temporal.io/server/tools/gomad/internal/model/volume"
	"go.temporal.io/server/tools/gomad/internal/model/world"
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
	goroot := stockgo.Root(t)
	prototypeOverlay.once.Do(func() {
		native, err := filepath.Abs("../../../model/runtimesource")
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
		warm.Config.NetworkV2 = nil
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

// prototypeTemporalConfig mirrors the admitted one-box guest configuration.
func prototypeTemporalConfig(t *testing.T, root, cwd string, args ...string) Config {
	t.Helper()
	captured, err := captured.CaptureReadOnlyMountInputs([]captured.Mapping{{Source: filepath.Join(root, "schema", "sqlite"), Target: "/workspace/schema/sqlite"}}, captured.DefaultLimits())
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

var prototypeFaults = []NetworkFaultV2{
	{Name: "workflow-reset", AtNanos: uint64(40 * time.Second), Action: "reset"},
	{Name: "workflow-partition", AtNanos: uint64(100 * time.Second), Action: "partition"},
	{Name: "workflow-heal", AtNanos: uint64(110 * time.Second), Action: "heal"},
}

type prototypeResetEvidence struct {
	Connections, StreamErrors, Reconnections uint64
}

func prototypeNetworkResetWitness(data []byte, fault NetworkFaultV2) (prototypeResetEvidence, error) {
	var transcript struct {
		Operations []struct {
			At      uint64           `json:"at"`
			Fault   *NetworkFaultV2  `json:"fault"`
			Request netmodel.Request `json:"request"`
			Result  netmodel.Result  `json:"result"`
		} `json:"operations"`
		Native netmodel.Record `json:"native"`
	}
	if err := json.Unmarshal(data, &transcript); err != nil {
		return prototypeResetEvidence{}, err
	}
	var evidence prototypeResetEvidence
	for _, transition := range transcript.Native.Transitions {
		if transition.Kind == "crash" && transition.Outcome == "reset" {
			evidence.Connections += transition.Count
		}
	}
	named := false
	handles := map[uint64]bool{}
	pending := map[uint64]uint64{}
	for _, operation := range transcript.Operations {
		if operation.Fault != nil && *operation.Fault == fault {
			named = true
		}
		if operation.Request.Op == "dial" && operation.Result.State == "completed" && operation.Result.Error == "" && operation.Result.Handle != 0 {
			if operation.At < fault.AtNanos {
				handles[operation.Result.Handle] = true
			} else {
				evidence.Reconnections++
			}
		}
		if operation.Request.Op == "read" || operation.Request.Op == "write" {
			pending[operation.Request.RequestID] = operation.Request.Handle
		}
		handle := operation.Request.Handle
		if operation.Request.Op == "poll" {
			handle = pending[operation.Request.RequestID]
		}
		if operation.At >= fault.AtNanos && handles[handle] && (operation.Request.Op == "read" || operation.Request.Op == "write" || operation.Request.Op == "poll") && (operation.Result.Error == "use of closed network connection" || operation.Result.Error == "connection reset by peer") {
			evidence.StreamErrors++
		}
	}
	if !named || evidence.Connections == 0 || evidence.StreamErrors == 0 || evidence.Reconnections == 0 {
		return evidence, fmt.Errorf("named reset lacks connection reset and recovery: named=%v evidence=%+v", named, evidence)
	}
	return evidence, nil
}

func TestPrototypeNetworkResetWitnessRequiresAffectedConnection(t *testing.T) {
	fault := NetworkFaultV2{Name: "workflow-reset", Action: "reset", AtNanos: 10}
	t.Run("no connections", func(t *testing.T) {
		e := streamEnvironment(t, streamConfig())
		e.networkV2.config.Faults = []NetworkFaultV2{fault}
		if err := e.networkV2Advance(fault.AtNanos); err != nil {
			t.Fatal(err)
		}
		data := e.NetworkV2Transcript()
		if !bytes.Contains(data, []byte(`"name":"workflow-reset"`)) {
			t.Fatal("negative control did not record named reset")
		}
		if evidence, err := prototypeNetworkResetWitness(data, fault); err == nil || evidence.Connections != 0 {
			t.Fatalf("reset without connections passed witness: %+v, %v", evidence, err)
		}
	})
	t.Run("reset and reconnect", func(t *testing.T) {
		e := streamEnvironment(t, streamConfig())
		e.networkV2.config.Faults = []NetworkFaultV2{fault}
		listener := streamCall(t, e, streamFrame(1, 1, 0, 0, 0, 0, nil))
		port := binary.BigEndian.Uint32(listener[48:])
		client := streamCall(t, e, streamFrame(2, 2, 0, 0, port, 0, nil))
		streamCall(t, e, streamFrame(3, 3, binary.BigEndian.Uint64(listener[16:]), 0, 0, 0, nil))
		e.now = fault.AtNanos
		streamCall(t, e, streamFrame(5, 4, binary.BigEndian.Uint64(client[16:]), 0, 0, 1, []byte{1}))
		streamCall(t, e, streamFrame(2, 5, 0, 0, port, 0, nil))
		evidence, err := prototypeNetworkResetWitness(e.NetworkV2Transcript(), fault)
		if err != nil || evidence.Connections != 1 || evidence.StreamErrors != 1 || evidence.Reconnections != 1 {
			t.Fatalf("reset and reconnect witness: %+v, %v", evidence, err)
		}
	})
}

// TestPrototypeTemporal runs the fn-151.6/.7/.8 one-box gates: a workflow with
// timers, retries and network faults, and an unchanged functional suite.
func TestPrototypeTemporal(t *testing.T) {
	t.Parallel()
	root, err := filepath.Abs("../../../../../..")
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
		if err := temporalStageFixture(root, filepath.Join("tools", "gomad", "internal", "backends", "wasm", "testdata", "prototype_temporal"), root, staged); err != nil {
			t.Fatal(err)
		}
		module, seconds := prototypeBuild(t, root, "test", "./"+staged, filepath.Join(modules, "prototype-temporal.wasm"), tags...)
		config := prototypeTemporalConfig(t, root, "/workspace/prototype", append([]string{"-test.run=^TestPrototypeWorkflow$"}, persistence...)...)
		config.NetworkV2 = &NetworkModelV2{Profile: NetworkStreamProfile, Limits: netmodel.Limits{Listeners: 128, Connections: 4096, Bytes: 128 << 20, ConnectionBytes: 4 << 20, StallNanos: 10_000_000_000, Transitions: 1 << 20}, Faults: prototypeFaults}
		request := prototypeRequest(t, module, config, 0, 30*time.Minute)
		baselines := prototypeDeterminism(t, report, "temporal", "workflow", request, seeds, instances, 0)
		resetEvidence := map[string]prototypeResetEvidence{}
		for seed, result := range baselines {
			if !bytes.Contains(result.Stdout, []byte("--- PASS: TestPrototypeWorkflow")) || !bytes.Contains(result.Stdout, []byte("workflow completed [step 0 attempt 2 step 1 attempt 2]")) {
				t.Fatalf("seed %d: workflow did not pass with both activity retries", seed)
			}
			witness, err := prototypeNetworkResetWitness(result.NetworkTranscript, prototypeFaults[0])
			if err != nil {
				t.Fatalf("seed %d: %v", seed, err)
			}
			resetEvidence[strconv.FormatUint(seed, 10)] = witness
		}
		result := baselines[seeds[0]]
		partitions := bytes.Count(result.NetworkTranscript, []byte(`"name":"workflow-partition"`))
		info, _ := os.Stat(module)
		report.guest("prototype-temporal", map[string]any{"module_sha256": request.ModuleSHA256, "module_bytes": info.Size(), "build_seconds": seconds, "model": config.NetworkV2, "reset_evidence": resetEvidence, "activity_retries": 2, "partitions": partitions, "distinct_schedules": distinctChoiceTraces(baselines)})
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

func cooperativeRequest(t *testing.T, fixture string) Request {
	t.Helper()
	if os.Getenv("GOMAD3_STOCK_GO") == "" {
		t.Fatal("qualified guest gate requires explicit GOMAD3_STOCK_GO")
	}
	root := stockgo.Root(t)
	native, err := filepath.Abs("../../../model/runtimesource")
	if err != nil {
		t.Fatal(err)
	}
	overlay, identity, err := toolchain.BuildOverlay(root, native, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	module := filepath.Join(t.TempDir(), "runtime.wasm")
	command := exec.Command(filepath.Join(root, "bin/go"), "build", "-p=1", "-trimpath", "-buildvcs=false", "-overlay", overlay, "-o", module, fixture)
	command.Env = append(os.Environ(), "GOROOT="+root, "GOOS=wasip1", "GOARCH=wasm", "GOEXPERIMENT=nogreenteagc", "GOWORK=off", "GOENV=off", "GOFLAGS=", "GOTOOLCHAIN=local", "CGO_ENABLED=0", "GOMAXPROCS=1")
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("build actual cooperative guest: %v\n%s", err, output)
	}
	return cooperativeModuleRequest(t, module, identity)
}

func TestCooperativeGoGuestForcesExactReplay(t *testing.T) {
	request := cooperativeRequest(t, "../toolchain/testdata/choices")
	observed, err := Run(context.Background(), request)
	if err != nil || observed.Termination != "exit" || observed.ExitCode == nil || *observed.ExitCode != 0 || !observed.Reaped {
		t.Fatalf("actual guest: %s %s err=%v stderr=%s", observed.Termination, observed.Message, err, observed.Stderr)
	}
	if observed.ChoiceTrace.Summary.Runnable == 0 || observed.ChoiceTrace.Summary.SelectPoll == 0 || len(observed.DiagnosticTrace.Records) != len(observed.ChoiceTrace.Records) || !bytes.Contains(observed.Stdout, []byte("timer")) {
		t.Fatalf("actual hook coverage: summary=%+v diagnostics=%d stdout=%s", observed.ChoiceTrace.Summary, len(observed.DiagnosticTrace.Records), observed.Stdout)
	}
	tape, err := choice.ProjectReplayPlan(observed.ChoiceTrace, request.Runtime.Choice.ExecutionIdentity)
	if err != nil {
		t.Fatal(err)
	}
	request.Runtime.Choice.Mode, request.Runtime.Choice.Tape = choice.ModeReplay, &tape
	request.Runtime.Diagnostics = false
	replayed, err := Run(context.Background(), request)
	if err != nil || replayed.Termination != "exit" || !replayed.Reaped || !bytes.Equal(observed.ChoiceTrace.Bytes, replayed.ChoiceTrace.Bytes) || !bytes.Equal(observed.Stdout, replayed.Stdout) || !bytes.Equal(observed.Transcript, replayed.Transcript) {
		t.Fatalf("actual exact replay: %s %s err=%v stderr=%s trace=%x/%x evidence=%x/%x", replayed.Termination, replayed.Message, err, replayed.Stderr, observed.ChoiceTrace.SHA256, replayed.ChoiceTrace.SHA256, sha256.Sum256(observed.Transcript), sha256.Sum256(replayed.Transcript))
	}
	t.Logf("actual cooperative guest record/exact replay: module=%s helper=%s runtime=%s choices=%+v diagnostics=%d", request.ModuleSHA256, request.HelperSHA256, request.Runtime.Choice.ExecutionIdentity.ToolchainBuildKey, observed.ChoiceTrace.Summary, len(observed.DiagnosticTrace.Records))
	for _, mutation := range []struct {
		name   string
		reason choice.DivergenceReason
	}{{"missing", choice.DivergenceTapeExhausted}, {"extra", choice.DivergenceTapeUnconsumed}, {"kind", choice.DivergenceKind}, {"site", choice.DivergenceSite}, {"enabled", choice.DivergenceAlternativeSet}} {
		t.Run(mutation.name, func(t *testing.T) {
			plan := tape
			if mutation.name == "missing" {
				plan, err = tape.Prefix(0)
			} else {
				records := slices.Clone(observed.ChoiceTrace.Records)
				target := slices.IndexFunc(records, func(record choice.Record) bool { return record.Kind == choice.KindRunnable })
				if target < 0 {
					t.Fatal("fixture has no runnable record to alter")
				}
				switch mutation.name {
				case "extra":
					extra := records[target]
					extra.Ordinal = uint64(len(records))
					records = append(records, extra)
				case "kind":
					records[target].Kind = choice.KindSelectPoll
				case "site":
					records[target].Flags &^= choice.FlagSiteMissing
					records[target].SiteOffset++
				case "enabled":
					records[target].AlternativeSetDigest = sha256.Sum256([]byte("inapplicable alternatives"))
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
			request.Runtime.Choice.Tape = &plan
			result, runErr := Run(t.Context(), request)
			if runErr != nil || result.Termination != "divergence" || result.ChoiceDivergence == nil || result.ChoiceDivergence.Reason != mutation.reason || !result.Reaped {
				t.Fatalf("actual tape %s: %s %s divergence=%+v err=%v", mutation.name, result.Termination, result.Message, result.ChoiceDivergence, runErr)
			}
		})
	}
}

func TestCooperativeRuntimeProbes(t *testing.T) {
	request := cooperativeRequest(t, "../testdata/runtime_probes")
	request.Config.Environment = append(request.Config.Environment, "EXTRA_ENTROPY=no_")
	observed, err := Run(context.Background(), request)
	if err != nil || observed.Termination != "exit" || observed.ExitCode == nil || (*observed.ExitCode != 0 && *observed.ExitCode != 17) || !observed.Reaped {
		t.Fatalf("actual runtime probes: %s %s err=%v stdout=%s stderr=%s", observed.Termination, observed.Message, err, observed.Stdout, observed.Stderr)
	}
	for _, witness := range []string{"entropy ", "map ", "runq ", "timer callbacks ", "timer parking", "note negative park wake", "collector automatic finalizer cleanup 1 1", "canary "} {
		if !bytes.Contains(observed.Stdout, []byte(witness)) {
			t.Fatalf("missing runtime witness %q: %s", witness, observed.Stdout)
		}
	}
	t.Logf("runtime probes stdout=%s summary=%+v diagnostics=%d", observed.Stdout, observed.ChoiceTrace.Summary, len(observed.DiagnosticTrace.Records))
	base := observed
	t.Run("diagnostic-divergence", func(t *testing.T) {
		records := slices.Clone(base.DiagnosticTrace.Records)
		for index := range records {
			records[index].RuntimeRandDraws++
		}
		altered, err := choice.BuildDiagnosticTrace(records, base.DiagnosticTrace.Capacity)
		if err != nil {
			t.Fatal(err)
		}
		difference, err := choice.DiffDiagnostics(base.DiagnosticTrace.Bytes, altered.Bytes)
		if err != nil || difference == nil || difference.Ordinal != 0 || !slices.Contains(difference.Fields, "runtime_rand_draws") {
			t.Fatalf("actual guest diagnostic mutation undetected: %+v %v", difference, err)
		}
	})
	t.Run("frozen-clock-parking", func(t *testing.T) {
		frozen := request
		frozen.Config.Clock.ReadStepNanos = 0
		result, err := Run(t.Context(), frozen)
		if err != nil || result.Termination != "exit" || !result.Reaped || !bytes.Contains(result.Stdout, []byte("timer parking")) || !bytes.Contains(result.Stdout, []byte("note negative park wake")) || !bytes.Contains(result.Stdout, []byte("collector automatic finalizer cleanup 1 1")) {
			t.Fatalf("frozen-clock idle handshake: %s %s err=%v stdout=%s stderr=%s", result.Termination, result.Message, err, result.Stdout, result.Stderr)
		}
		t.Logf("read-step=0 timer deadlines and negative note parking progressed: %s", result.Stdout)
	})
	for _, perturbation := range []string{"application-draws", "application-key", "runtime-seed"} {
		t.Run(perturbation, func(t *testing.T) {
			changed := request
			changed.Config.Environment = append([]string(nil), request.Config.Environment...)
			control := *request.Runtime
			changed.Runtime = &control
			switch perturbation {
			case "application-draws":
				changed.Config.Environment[len(changed.Config.Environment)-1] = "EXTRA_ENTROPY=yes"
			case "application-key":
				changed.Config.EntropyKey[31] = 99
			case "runtime-seed":
				changed.Runtime.Seed++
			}
			result, err := Run(context.Background(), changed)
			if err != nil || result.Termination != "exit" || !result.Reaped {
				t.Fatalf("entropy perturbation: %s %s %v %s", result.Termination, result.Message, err, result.Stderr)
			}
			first, rest, ok := bytes.Cut(result.Stdout, []byte{'\n'})
			baseFirst, baseRest, baseOK := bytes.Cut(base.Stdout, []byte{'\n'})
			if !ok || !baseOK {
				t.Fatal("missing entropy observation")
			}
			if perturbation == "runtime-seed" {
				if !bytes.Equal(first, baseFirst) || bytes.Equal(rest, baseRest) {
					t.Fatal("runtime seed changed application entropy or failed to vary runtime behavior")
				}
			} else {
				if perturbation == "application-key" && bytes.Equal(first, baseFirst) {
					t.Fatal("application entropy key had no effect")
				}
				if !bytes.Equal(rest, baseRest) || !bytes.Equal(result.ChoiceTrace.Bytes, base.ChoiceTrace.Bytes) || len(result.DiagnosticTrace.Records) != len(base.DiagnosticTrace.Records) {
					t.Fatal("application entropy perturbed runtime behavior")
				}
				for i, record := range result.DiagnosticTrace.Records {
					previous := base.DiagnosticTrace.Records[i]
					if record.RunqDraws != previous.RunqDraws || record.SchedulerDraws != previous.SchedulerDraws || record.SelectDraws != previous.SelectDraws || record.RuntimeRandDraws != previous.RuntimeRandDraws || record.RuntimeCheapRandDraws != previous.RuntimeCheapRandDraws || record.TimerDraws != previous.TimerDraws {
						t.Fatalf("application entropy perturbed seeded draw counters at %d", i)
					}
				}
			}
		})
	}
	tape, err := choice.ProjectReplayPlan(base.ChoiceTrace, request.Runtime.Choice.ExecutionIdentity)
	if err != nil {
		t.Fatal(err)
	}
	last := uint64(len(tape.Decisions) - 1)
	if tape.Decisions[last].Kind != choice.KindSelectPoll || tape.Decisions[last].Alternatives != 2 {
		t.Fatal("canary is not the final two-way branch")
	}
	prefix, err := choice.BuildRankPrefix(tape, last, 1-tape.Decisions[last].Selected)
	if err != nil {
		t.Fatal(err)
	}
	request.Runtime.Choice.Mode, request.Runtime.Choice.Tape = choice.ModePrefix, &prefix
	alternate, err := Run(context.Background(), request)
	if err != nil || alternate.Termination != "exit" || alternate.ExitCode == nil || *alternate.ExitCode == *base.ExitCode || !alternate.Reaped {
		t.Fatalf("alternate forced canary: %s %s %v stdout=%s stderr=%s", alternate.Termination, alternate.Message, err, alternate.Stdout, alternate.Stderr)
	}
	alternateTape, err := choice.ProjectReplayPlan(alternate.ChoiceTrace, request.Runtime.Choice.ExecutionIdentity)
	if err != nil {
		t.Fatal(err)
	}
	request.Runtime.Choice.Mode, request.Runtime.Choice.Tape = choice.ModeReplay, &alternateTape
	request.Runtime.Diagnostics = false
	exact, err := Run(context.Background(), request)
	if err != nil || exact.Termination != "exit" || exact.ExitCode == nil || *exact.ExitCode != *alternate.ExitCode || !bytes.Equal(exact.Stdout, alternate.Stdout) || !bytes.Equal(exact.ChoiceTrace.Bytes, alternate.ChoiceTrace.Bytes) || !bytes.Equal(exact.Transcript, alternate.Transcript) {
		t.Fatalf("alternate exact replay: %s %s %v stdout=%s stderr=%s", exact.Termination, exact.Message, err, exact.Stdout, exact.Stderr)
	}
	t.Logf("forced alternate canary exit=%d then full-tape exact replay; application/runtime entropy independence checked", *alternate.ExitCode)
}

func TestCooperativePreflightRejectsIncompleteRuntimeABI(t *testing.T) {
	request := fixtureRequest(t, `(module (memory (export "memory") 1) (func (export "_start")))`, nil)
	request.Config.Profile = CooperativeProfile
	identity := choice.ExecutionIdentity{TargetSHA256: sha256.Sum256([]byte("guest")), ToolchainBuildKey: strings.Repeat("a", 64), GOOS: "wasip1", GOARCH: "wasm", ImplementationSHA256: sha256.Sum256([]byte("runtime"))}
	request.Runtime = &RuntimeControl{Choice: &runnerbackend.ChoiceRequest{Mode: choice.ModeRecord, ExecutionIdentity: identity, Limit: 8192}}
	if _, err := Run(context.Background(), request); err == nil {
		t.Fatal("cooperative profile authorized guest without runtime ABI")
	}
}

type runtimeQualification struct {
	SeedIndex, Seed, Instances, ExactReplays                               uint64
	Module, Helper, Runtime, Model                                         string
	SourceSHA256, RuntimeImplementationSHA256, CompilerSHA256, GoVersion   string
	HostGOOS, HostGOARCH, Collector, Profile                               string
	Engine                                                                 EngineIdentity
	Clock                                                                  ClockPolicy
	Limits                                                                 Limits
	ChoiceSHA256, DiagnosticSHA256, StdoutSHA256, StderrSHA256, WASISHA256 string
	TerminalSHA256                                                         string
	ExitCode                                                               uint32
	Summary                                                                choice.Summary
	LastDiagnostic                                                         choice.DiagnosticRecord
}

// Owner gate budget (2026-10-12): fresh instances recorded per qualification seed.
const qualificationInstances = 25

func TestCooperativeRuntimeQualification(t *testing.T) {
	indexValue := os.Getenv("GOMAD_WASM_QUALIFICATION_INDEX")
	if indexValue == "" {
		t.Skip("run qualify-runtime for the mandatory scheduling qualification gate")
	}
	index, err := strconv.ParseUint(indexValue, 10, 64)
	if err != nil || index >= 32 {
		t.Fatal("qualification index must be 0..31")
	}
	root := os.Getenv("GOMAD_WASM_QUALIFICATION_ROOT")
	if !filepath.IsAbs(root) {
		t.Fatal("qualification requires an absolute retained evidence root")
	}
	request := cooperativeRequest(t, "../testdata/runtime_probes")
	sourceSHA256 := qualificationSourceSHA256(t)
	request.Runtime.Seed = index * 1000003
	request.Config.Environment = append(request.Config.Environment, "EXTRA_ENTROPY=no_")
	var baseline Result
	for instance := range qualificationInstances {
		observed, err := matchedPrepareAdmitted(runtime.GOOS, "/proc/meminfo", os.Getenv("GOMAD_WASM_MATCHED_COLD_MEMORY_GUARD"), func() (Result, error) {
			return Run(t.Context(), request)
		})
		if err != nil || observed.Termination != "exit" || observed.ExitCode == nil || (*observed.ExitCode != 0 && *observed.ExitCode != 17) || !observed.Reaped || observed.Cancelled || observed.WatchdogTimeout {
			t.Fatalf("seed %d fresh instance %d: %s %s err=%v stderr=%s", request.Runtime.Seed, instance, observed.Termination, observed.Message, err, observed.Stderr)
		}
		for _, witness := range []string{"entropy ", "map ", "runq ", "timer callbacks ", "timer parking", "note negative park wake", "collector automatic finalizer cleanup 1 1", "canary "} {
			if !bytes.Contains(observed.Stdout, []byte(witness)) {
				t.Fatalf("seed %d instance %d missing %q", request.Runtime.Seed, instance, witness)
			}
		}
		if observed.ChoiceTrace.Summary.Runnable == 0 || observed.ChoiceTrace.Summary.SelectPoll == 0 || observed.ChoiceTrace.Summary.SelectResult == 0 || observed.ChoiceTrace.Summary.Terminal != choice.TerminalComplete || len(observed.DiagnosticTrace.Records) != len(observed.ChoiceTrace.Records) {
			t.Fatalf("seed %d instance %d incomplete hook witnesses: %+v", request.Runtime.Seed, instance, observed.ChoiceTrace.Summary)
		}
		last := observed.DiagnosticTrace.Records[len(observed.DiagnosticTrace.Records)-1]
		if last.Allocations == 0 || last.GCCycle == 0 || last.RunqDraws == 0 || last.SelectDraws == 0 || last.RuntimeRandDraws == 0 || last.TimerDraws == 0 {
			t.Fatalf("seed %d instance %d missing collector/seeded-stream evidence: %+v", request.Runtime.Seed, instance, last)
		}
		if instance == 0 {
			baseline = observed
		} else if *observed.ExitCode != *baseline.ExitCode || !bytes.Equal(observed.Stdout, baseline.Stdout) || !bytes.Equal(observed.Stderr, baseline.Stderr) || !bytes.Equal(observed.Transcript, baseline.Transcript) || !bytes.Equal(observed.ChoiceTrace.Bytes, baseline.ChoiceTrace.Bytes) || !bytes.Equal(observed.DiagnosticTrace.Bytes, baseline.DiagnosticTrace.Bytes) || observed.ChoiceTrace.Summary != baseline.ChoiceTrace.Summary || observed.runtimeTerminal != baseline.runtimeTerminal {
			t.Fatalf("seed %d fresh instance %d diverged: stdout=%x/%x wasi=%x/%x choices=%x/%x diagnostics=%x/%x", request.Runtime.Seed, instance, sha256.Sum256(observed.Stdout), sha256.Sum256(baseline.Stdout), sha256.Sum256(observed.Transcript), sha256.Sum256(baseline.Transcript), observed.ChoiceTrace.SHA256, baseline.ChoiceTrace.SHA256, observed.DiagnosticTrace.SHA256, baseline.DiagnosticTrace.SHA256)
		}
		if difference, err := choice.DiffDiagnostics(baseline.DiagnosticTrace.Bytes, observed.DiagnosticTrace.Bytes); err != nil || difference != nil {
			t.Fatalf("seed %d instance %d diagnostic divergence: %+v %v", request.Runtime.Seed, instance, difference, err)
		}
	}
	tape, err := choice.ProjectReplayPlan(baseline.ChoiceTrace, request.Runtime.Choice.ExecutionIdentity)
	if err != nil {
		t.Fatal(err)
	}
	request.Runtime.Choice.Mode, request.Runtime.Choice.Tape = choice.ModeReplay, &tape
	request.Runtime.Diagnostics = false
	exact, err := matchedPrepareAdmitted(runtime.GOOS, "/proc/meminfo", os.Getenv("GOMAD_WASM_MATCHED_COLD_MEMORY_GUARD"), func() (Result, error) {
		return Run(t.Context(), request)
	})
	if err != nil || exact.Termination != "exit" || exact.ExitCode == nil || *exact.ExitCode != *baseline.ExitCode || !exact.Reaped || !bytes.Equal(exact.Stdout, baseline.Stdout) || !bytes.Equal(exact.Stderr, baseline.Stderr) || !bytes.Equal(exact.Transcript, baseline.Transcript) || !bytes.Equal(exact.ChoiceTrace.Bytes, baseline.ChoiceTrace.Bytes) {
		t.Fatalf("seed %d forced full-tape replay: %s %s err=%v stderr=%s", request.Runtime.Seed, exact.Termination, exact.Message, err, exact.Stderr)
	}
	report := runtimeQualification{SeedIndex: index, Seed: request.Runtime.Seed, Instances: qualificationInstances, ExactReplays: 1, Module: request.ModuleSHA256, Helper: request.HelperSHA256, Runtime: request.Runtime.Choice.ExecutionIdentity.ToolchainBuildKey, Model: ImplementationSHA256(), ChoiceSHA256: fmt.Sprintf("%x", baseline.ChoiceTrace.SHA256), DiagnosticSHA256: fmt.Sprintf("%x", baseline.DiagnosticTrace.SHA256), StdoutSHA256: fmt.Sprintf("%x", sha256.Sum256(baseline.Stdout)), StderrSHA256: fmt.Sprintf("%x", sha256.Sum256(baseline.Stderr)), WASISHA256: fmt.Sprintf("%x", sha256.Sum256(baseline.Transcript)), ExitCode: *baseline.ExitCode, Summary: baseline.ChoiceTrace.Summary, LastDiagnostic: baseline.DiagnosticTrace.Records[len(baseline.DiagnosticTrace.Records)-1]}
	version, err := exec.Command(os.Getenv("GOMAD3_STOCK_GO"), "version").Output()
	if err != nil || !strings.Contains(string(version), "go1.27.1 ") {
		t.Fatalf("qualification compiler version: %q %v", version, err)
	}
	report.SourceSHA256 = sourceSHA256
	report.RuntimeImplementationSHA256 = toolchain.ImplementationSHA256()
	report.CompilerSHA256 = fileHash(t, os.Getenv("GOMAD3_STOCK_GO"))
	report.GoVersion = strings.TrimSpace(string(version))
	report.HostGOOS, report.HostGOARCH = runtime.GOOS, runtime.GOARCH
	report.Engine, report.Clock, report.Limits = request.Engine, request.Config.Clock, request.Config.Limits
	report.Collector, report.Profile = "nogreenteagc", request.Config.Profile
	report.TerminalSHA256 = fmt.Sprintf("%x", sha256.Sum256(baseline.runtimeTerminal[:]))
	data, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	directory := filepath.Join(root, fmt.Sprintf("seed-%02d", index))
	if err := os.MkdirAll(directory, 0700); err != nil {
		t.Fatal(err)
	}
	for name, payload := range map[string][]byte{"report.json": data, "stdout": baseline.Stdout, "stderr": baseline.Stderr, "choices.bin": baseline.ChoiceTrace.Bytes, "diagnostics.bin": baseline.DiagnosticTrace.Bytes, "wasi.bin": baseline.Transcript, "finish.bin": baseline.runtimeTerminal[:]} {
		if err := os.WriteFile(filepath.Join(directory, name), payload, 0600); err != nil {
			t.Fatal(err)
		}
	}
	t.Logf("qualification %s", data)
}

func TestCooperativeRuntimeQualificationAggregate(t *testing.T) {
	root := os.Getenv("GOMAD_WASM_QUALIFICATION_ROOT")
	if root == "" {
		t.Skip("run qualify-runtime for the retained aggregate gate")
	}
	if !filepath.IsAbs(root) {
		t.Fatal("qualification requires an absolute retained evidence root")
	}
	sourceSHA256 := qualificationSourceSHA256(t)
	choices, outputs, outcomes := map[string]bool{}, map[string]bool{}, map[uint32]bool{}
	domains := map[string]map[string]bool{"map ": {}, "runq ": {}, "timer callbacks ": {}, "canary ": {}}
	var first runtimeQualification
	for index := range 32 {
		data, err := os.ReadFile(filepath.Join(root, fmt.Sprintf("seed-%02d/report.json", index)))
		if err != nil {
			t.Fatal(err)
		}
		var report runtimeQualification
		if err := json.Unmarshal(data, &report); err != nil {
			t.Fatal(err)
		}
		if report.SeedIndex != uint64(index) || report.Seed != uint64(index)*1000003 || report.Instances != qualificationInstances || report.ExactReplays != 1 || report.Model != ImplementationSHA256() || report.SourceSHA256 != sourceSHA256 || report.RuntimeImplementationSHA256 != toolchain.ImplementationSHA256() || report.Helper != fileHash(t, helperPath(t)) || report.CompilerSHA256 != fileHash(t, os.Getenv("GOMAD3_STOCK_GO")) || report.Engine != PinnedEngine() || report.HostGOOS != runtime.GOOS || report.HostGOARCH != runtime.GOARCH || report.Profile != CooperativeProfile || report.Collector != "nogreenteagc" || !strings.Contains(report.GoVersion, "go1.27.1 ") {
			t.Fatalf("missing or stale qualification shard %d", index)
		}
		if index == 0 {
			first = report
		} else if report.Module != first.Module || report.Helper != first.Helper || report.Runtime != first.Runtime || report.GoVersion != first.GoVersion || report.Clock != first.Clock || report.Limits != first.Limits {
			t.Fatal("qualification shards are not one frozen candidate")
		}
		choices[report.ChoiceSHA256], outputs[report.StdoutSHA256], outcomes[report.ExitCode] = true, true, true
		payloads := map[string][]byte{}
		for name, expected := range map[string]string{"stdout": report.StdoutSHA256, "stderr": report.StderrSHA256, "choices.bin": report.ChoiceSHA256, "diagnostics.bin": report.DiagnosticSHA256, "wasi.bin": report.WASISHA256, "finish.bin": report.TerminalSHA256} {
			payload, err := os.ReadFile(filepath.Join(root, fmt.Sprintf("seed-%02d", index), name))
			if err != nil || fmt.Sprintf("%x", sha256.Sum256(payload)) != expected {
				t.Fatalf("qualification shard %d payload %s changed: %v", index, name, err)
			}
			payloads[name] = payload
		}
		if err := validateQualificationPayloads(report, payloads); err != nil {
			t.Fatalf("qualification shard %d payload malformed: %v", index, err)
		}
		for prefix, observations := range domains {
			found := false
			for _, line := range strings.Split(string(payloads["stdout"]), "\n") {
				if strings.HasPrefix(line, prefix) {
					observations[line] = true
					found = true
				}
			}
			if !found {
				t.Fatalf("qualification shard %d lacks %s", index, prefix)
			}
		}
	}
	if len(choices) < 2 || len(outputs) < 2 || !outcomes[0] || !outcomes[17] {
		t.Fatal("representative seeds did not reach distinct choices, outputs and canary outcomes")
	}
	for name, observations := range domains {
		if len(observations) < 2 {
			t.Fatalf("representative seeds did not vary the %s domain", name)
		}
	}
	t.Logf("32 seeds x %d fresh instances: %d complete record observations and 32 fresh forced exact replays; choices=%d outputs=%d outcomes=%d module=%s helper=%s runtime=%s model=%s", qualificationInstances, 32*qualificationInstances, len(choices), len(outputs), len(outcomes), first.Module, first.Helper, first.Runtime, first.Model)
}

func qualificationSourceSHA256(t *testing.T) string {
	t.Helper()
	manifest := os.Getenv("GOMAD_WASM_QUALIFICATION_MANIFEST")
	if !filepath.IsAbs(manifest) {
		t.Fatal("qualification requires an absolute frozen source manifest")
	}
	data, err := os.ReadFile(manifest)
	if err != nil || len(data) == 0 {
		t.Fatalf("read frozen source manifest: %v", err)
	}
	for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
		digest, path, found := strings.Cut(line, "  ")
		if !found || !filepath.IsAbs(path) || digest != fileHash(t, path) {
			t.Fatalf("qualification source changed: %q", line)
		}
	}
	return fmt.Sprintf("%x", sha256.Sum256(data))
}

func TestQualificationPayloadPreservesTerminalPeak(t *testing.T) {
	a, b := sha256.Sum256([]byte("a")), sha256.Sum256([]byte("b"))
	decision, err := choice.CanonicalDecision(0, choice.KindRunnable, 20, false, [][32]byte{a, b}, a, 0)
	if err != nil {
		t.Fatal(err)
	}
	trace, err := choice.BuildTrace([]choice.Record{decision.Record()}, choice.TerminalComplete)
	if err != nil {
		t.Fatal(err)
	}
	diagnostic, err := choice.BuildDiagnosticTrace([]choice.DiagnosticRecord{{Ordinal: 0}}, 8192)
	if err != nil {
		t.Fatal(err)
	}
	trace.Summary.PeakGoroutines = 5
	finish := make([]byte, 16)
	binary.BigEndian.PutUint32(finish, 1)
	binary.BigEndian.PutUint32(finish[4:], 5)
	report := runtimeQualification{Summary: trace.Summary, LastDiagnostic: diagnostic.Records[0]}
	payloads := map[string][]byte{"choices.bin": trace.Bytes, "diagnostics.bin": diagnostic.Bytes, "finish.bin": finish}
	if err := validateQualificationPayloads(report, payloads); err != nil {
		t.Fatalf("valid terminal peak rejected: %v", err)
	}
	binary.BigEndian.PutUint32(finish[4:], 4)
	if err := validateQualificationPayloads(report, payloads); err == nil {
		t.Fatal("altered retained terminal peak accepted")
	}
}

func validateQualificationPayloads(report runtimeQualification, payloads map[string][]byte) error {
	trace, err := choice.DecodeStoredTrace(choice.Profile, payloads["choices.bin"], choice.TerminalMetadata{State: report.Summary.Terminal, Limit: 1 << 20, Records: report.Summary.Records, SHA256: sha256.Sum256(payloads["choices.bin"])})
	if err != nil {
		return err
	}
	finish := payloads["finish.bin"]
	if len(finish) != 16 || binary.BigEndian.Uint32(finish) != 1 || binary.BigEndian.Uint32(finish[4:]) == 0 {
		return fmt.Errorf("terminal frame malformed")
	}
	trace.Summary.PeakGoroutines = binary.BigEndian.Uint32(finish[4:])
	if err != nil || len(trace.Records) == 0 || trace.Summary != report.Summary || trace.Summary.Terminal != choice.TerminalComplete {
		return fmt.Errorf("choices malformed: %v", err)
	}
	diagnostic, err := choice.DecodeDiagnosticTrace(payloads["diagnostics.bin"])
	if err != nil || len(diagnostic.Records) == 0 || len(diagnostic.Records) != len(trace.Records) || diagnostic.Records[len(diagnostic.Records)-1] != report.LastDiagnostic {
		return fmt.Errorf("diagnostics malformed: %v", err)
	}
	return nil
}

func TestStockGoGuestRepeatsIn100FreshInstances(t *testing.T) {
	helper := helperPath(t)
	goCommand := os.Getenv("GOMAD3_STOCK_GO")
	if goCommand == "" {
		goCommand = "go"
	}
	compiler, err := exec.Command(goCommand, "version").Output()
	if err != nil || !strings.Contains(string(compiler), "go1.27.1 ") {
		t.Fatalf("stock compiler identity: %q, %v", compiler, err)
	}
	module := filepath.Join(t.TempDir(), "environment.wasm")
	command := exec.Command(goCommand, "build", "-trimpath", "-o", module, "../testdata/environment")
	command.Env = append(os.Environ(), "GOOS=wasip1", "GOARCH=wasm", "CGO_ENABLED=0", "GOWORK=off", "GOENV=off", "GOFLAGS=", "GOEXPERIMENT=nogreenteagc", "GOTOOLCHAIN=local")
	if data, err := command.CombinedOutput(); err != nil {
		t.Fatalf("build real guest: %v\n%s", err, data)
	}
	data, err := os.ReadFile(module)
	if err != nil {
		t.Fatal(err)
	}
	imports, pages := binaryInventory(t, data)
	config := testConfig()
	config.Clock.ReadStepNanos = 1000000
	environment := newTestEnvironment(t, config)
	request := Request{HelperPath: helper, HelperSHA256: fileHash(t, helper), ModulePath: module, ModuleSHA256: fileHash(t, module), Engine: PinnedEngine(), Imports: imports, InitialMemoryPages: pages, MemoryBytes: 256 << 20, Fuel: 4000000000, Config: config, Timeout: 60 * time.Second}
	var stdout, transcript []byte
	for i := 0; i < 100; i++ {
		result, err := Run(context.Background(), request)
		if err != nil {
			t.Fatal(err)
		}
		if result.Termination != "exit" || result.ExitCode == nil || *result.ExitCode != 0 || !result.Reaped || result.EnvironmentIdentity != environment.Identity() {
			t.Fatalf("guest %d: %s: %s reaped=%v\nstdout=%s stderr=%s transcript bytes=%d hash=%x", i, result.Termination, result.Message, result.Reaped, string(result.Stdout), string(result.Stderr), len(result.Transcript), sha256.Sum256(result.Transcript))
		}
		if i == 0 {
			stdout = result.Stdout
			transcript = result.Transcript
			for _, marker := range []string{"map ", "select ", "random ", "timer ", "fakeTCP echo", "cwd /workspace env captured arg literal $arg", "file scratch"} {
				if !strings.Contains(string(stdout), marker) {
					t.Fatalf("missing guest probe %q: %s", marker, stdout)
				}
			}
		} else if !bytes.Equal(stdout, result.Stdout) || !bytes.Equal(transcript, result.Transcript) {
			t.Fatalf("fresh guest %d diverged: stdout %q != %q; transcript %x != %x", i, result.Stdout, stdout, sha256.Sum256(result.Transcript), sha256.Sum256(transcript))
		}
	}
	t.Logf("100 fresh guests, 100 output and transcript equalities: compiler=%s experiment=nogreenteagc environment=%s module=%s helper=%s stdout=%x transcript=%x imports=%d memory-pages=%d", strings.TrimSpace(string(compiler)), environment.Identity(), request.ModuleSHA256, request.HelperSHA256, sha256.Sum256(stdout), sha256.Sum256(transcript), len(imports), pages)
}

func TestBrokerCrashRecoveryFreshGuest(t *testing.T) {
	request := cooperativeRequest(t, "../cluster/testdata/service")
	request.Config.Clock.ReadStepNanos = 0
	request.Config.Storage = &StorageModel{Volumes: []volume.VolumeConfig{{ID: "a-store", Path: "/workspace", CapacityBytes: 8 << 20}}, Limits: volume.VolumeLimits{PendingOperations: 1024, Transitions: 10000}}
	run := func() ([]Result, world.Snapshot) {
		b, err := NewBroker(brokerTestConfig())
		if err != nil {
			t.Fatal(err)
		}
		if err = b.Session("a").CrashOnOutput([]byte("committed durable-a")); err != nil {
			t.Fatal(err)
		}
		peer := request
		peer.Config = request.Config
		peer.Config.Storage = nil
		peer.Config.Args = []string{"service", "park"}
		peer.Broker = b.Session("b")
		peerDone := make(chan Result, 1)
		peerErr := make(chan error, 1)
		go func() { result, err := Run(t.Context(), peer); peerDone <- result; peerErr <- err }()
		first := request
		first.Config = request.Config
		first.Config.Args = []string{"service", "durable"}
		first.Broker = b.Session("a")
		crashed, err := Run(t.Context(), first)
		if err != nil || crashed.Termination != "crash" || crashed.StorageImage == nil || !crashed.Reaped {
			t.Fatalf("crash: %s %s err=%v stdout=%s stderr=%s", crashed.Termination, crashed.Message, err, crashed.Stdout, crashed.Stderr)
		}
		session, err := b.Restart("a")
		if err != nil {
			t.Fatal(err)
		}
		restart := request
		restart.Config = request.Config
		restart.Config.Args = []string{"service", "recover"}
		restart.Broker = session
		restart.StorageImage = crashed.StorageImage
		recovered, err := Run(t.Context(), restart)
		if err != nil || recovered.Termination != "exit" || !recovered.Reaped || !bytes.Contains(recovered.Stdout, []byte("fresh heap 1\nrecovered durable-a")) {
			t.Fatalf("recover: %s %s %v stdout=%s stderr=%s", recovered.Termination, recovered.Message, err, recovered.Stdout, recovered.Stderr)
		}
		other := <-peerDone
		if err = <-peerErr; err != nil || other.Termination != "exit" || !other.Reaped {
			t.Fatalf("peer: %s %s %v", other.Termination, other.Message, err)
		}
		return []Result{crashed, recovered, other}, b.Snapshot()
	}
	left, leftOrder := run()
	right, rightOrder := run()
	for i := range left {
		if !bytes.Equal(left[i].Stdout, right[i].Stdout) || !bytes.Equal(left[i].Transcript, right[i].Transcript) || !bytes.Equal(left[i].StorageTranscript, right[i].StorageTranscript) {
			t.Fatalf("crash/recover node incarnation %d differs", i)
		}
	}
	if !reflect.DeepEqual(leftOrder, rightOrder) {
		t.Fatal("crash recovery global event order differs")
	}
}

func TestBrokerIndependentGuestStreamsReplay(t *testing.T) {
	request := cooperativeRequest(t, "../cluster/testdata/service")
	request.Config.Clock.ReadStepNanos = 0
	request.Config.NetworkV2 = &NetworkModelV2{Profile: NetworkStreamProfile, Limits: netmodel.Limits{Listeners: 128, Connections: 128, Bytes: 8 << 20, ConnectionBytes: 4 << 20, StallNanos: 10_000_000_000, Transitions: 10000}}
	request.Config.Limits.Calls = 100000
	request.Config.Limits.TranscriptBytes = 32 << 20
	request.Runtime.Choice.Limit = 8 << 20
	run := func(tapes []*choice.ReplayPlan) ([]Result, world.Snapshot) {
		b, err := NewBroker(func() BrokerConfig {
			c := brokerTestConfig()
			c.Network = request.Config.NetworkV2
			c.Faults = []BrokerNetworkFault{{Name: "partition", Action: "partition", From: "a", To: "b", AtNanos: 1, Symmetric: true}, {Name: "heal", Action: "heal", From: "a", To: "b", AtNanos: 1_000_000_001, Symmetric: true}}
			return c
		}())
		if err != nil {
			t.Fatal(err)
		}
		results := make([]Result, 2)
		errs := make([]error, 2)
		var wg sync.WaitGroup
		for index, id := range []string{"a", "b"} {
			r := request
			r.Config = request.Config
			r.Runtime = &RuntimeControl{Seed: request.Runtime.Seed, Diagnostics: true, Choice: new(*request.Runtime.Choice)}
			r.Broker = b.Session(id)
			if index == 0 {
				r.Config.Args = []string{"service", "server"}
			} else {
				r.Config.Args = []string{"service", "client", b.Address("a")}
			}
			if tapes != nil {
				r.Runtime.Choice.Mode = choice.ModeReplay
				r.Runtime.Choice.Tape = tapes[index]
			}
			wg.Add(1)
			go func() { defer wg.Done(); results[index], errs[index] = Run(context.Background(), r) }()
		}
		wg.Wait()
		for i, result := range results {
			if errs[i] != nil || result.Termination != "exit" || result.ExitCode == nil || *result.ExitCode != 0 || !result.Reaped {
				t.Fatalf("node %d: %s %s err=%v stdout=%s stderr=%s", i, result.Termination, result.Message, errs[i], result.Stdout, result.Stderr)
			}
			if !bytes.Contains(result.Stdout, []byte("fresh heap 1")) || !bytes.Contains(result.Stdout, []byte("ordered bytes ordered-byte")) {
				t.Fatalf("node %d witnesses: %s", i, result.Stdout)
			}
		}
		return results, b.Snapshot()
	}
	observed, order := run(nil)
	tapes := make([]*choice.ReplayPlan, 2)
	for i, result := range observed {
		plan, err := choice.ProjectReplayPlan(result.ChoiceTrace, request.Runtime.Choice.ExecutionIdentity)
		if err != nil {
			t.Fatal(err)
		}
		tapes[i] = &plan
	}
	replayed, replayOrder := run(tapes)
	for i := range observed {
		if !bytes.Equal(observed[i].Stdout, replayed[i].Stdout) || !bytes.Equal(observed[i].Transcript, replayed[i].Transcript) || !bytes.Equal(observed[i].NetworkTranscript, replayed[i].NetworkTranscript) {
			t.Fatalf("node %d fresh replay differs", i)
		}
	}
	if !reflect.DeepEqual(order, replayOrder) {
		left, _ := json.Marshal(order)
		right, _ := json.Marshal(replayOrder)
		t.Fatalf("global broker event order replay differs: %s / %s", left, right)
	}
}

func TestHelperModelPreservesGoOwnerEvidence(t *testing.T) {
	request := cooperativeRequest(t, "../toolchain/testdata/choices")
	requireSmallHelperParityModule(t, request)
	for _, digestOnly := range []bool{false, true} {
		request.DigestTranscript = digestOnly
		for _, frozen := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/digest-%t", map[bool]string{false: "read-step", true: "frozen"}[frozen], digestOnly), func(t *testing.T) {
				request.Config.Clock.ReadStepNanos = 1000
				if frozen {
					request.Config.Clock.ReadStepNanos = 0
				}
				remote := request
				remote.remoteModel = true
				reference, err := Run(t.Context(), remote)
				if err != nil {
					t.Fatal(err)
				}
				requireExit(t, "Go reference", reference, 0)
				local, err := Run(t.Context(), request)
				if err != nil {
					t.Fatal(err)
				}
				if local.GoFilesystemCalls == nil || len(local.GoFilesystemCalls) != 0 {
					t.Fatalf("helper namespace leaked to Go: %v", local.GoFilesystemCalls)
				}
				if diff := sameEvidence(reference, local); diff != "" || !bytes.Equal(reference.DiagnosticTrace.Bytes, local.DiagnosticTrace.Bytes) || reference.EnvironmentIdentity != local.EnvironmentIdentity || reference.Calls != local.Calls || reference.TimeAdvances != local.TimeAdvances {
					t.Fatalf("helper evidence differs in %s: %s %s diagnostics=%v calls=%d/%d advances=%d/%d", diff, local.Termination, local.Message, bytes.Equal(reference.DiagnosticTrace.Bytes, local.DiagnosticTrace.Bytes), reference.Calls, local.Calls, reference.TimeAdvances, local.TimeAdvances)
				}
				if local.Telemetry == nil || local.Telemetry.PrepareNanos == 0 || local.Telemetry.InstantiateNanos == 0 || local.Telemetry.ExecuteNanos == 0 || local.HelperResourceUsage == nil || reference.AuthorizationToTerminalNanos == 0 || local.AuthorizationToTerminalNanos == 0 {
					t.Fatalf("measured helper phase/resource telemetry missing: %+v %+v", local.Telemetry, local.HelperResourceUsage)
				}
				t.Logf("measured authorized execution Go-owner=%dns helper-owner=%dns (includes fresh instantiation and evidence flush)", reference.AuthorizationToTerminalNanos, local.AuthorizationToTerminalNanos)
				tape, err := choice.ProjectReplayPlan(reference.ChoiceTrace, request.Runtime.Choice.ExecutionIdentity)
				if err != nil {
					t.Fatal(err)
				}
				for _, mode := range []choice.Mode{choice.ModeReplay, choice.ModePrefix} {
					forced := request
					control := *request.Runtime
					choices := *control.Choice
					choices.Mode, choices.Tape = mode, &tape
					control.Choice = &choices
					forced.Runtime = &control
					replayed, err := Run(context.Background(), forced)
					if err != nil {
						t.Fatal(err)
					}
					if diff := sameEvidence(reference, replayed); diff != "" {
						t.Fatalf("forced mode %d differs in %s: %s %s", mode, diff, replayed.Termination, replayed.Message)
					}
				}
			})
		}
	}
}

func TestNamespaceOwnerFilesystemParity(t *testing.T) {
	request := cooperativeRequest(t, "./testdata/namespace_owner")
	requireSmallHelperParityModule(t, request)
	for _, digest := range []bool{false, true} {
		request.DigestTranscript = digest
		referenceRequest := request
		referenceRequest.remoteModel = true
		reference, err := Run(t.Context(), referenceRequest)
		if err != nil {
			t.Fatal(err)
		}
		requireExit(t, "Go namespace", reference, 0)
		if reference.GoFilesystemCalls["path_open"] == 0 || reference.GoFilesystemCalls["fd_pwrite"] == 0 || reference.GoFilesystemCalls["fd_readdir"] == 0 || reference.GoFilesystemCalls["path_rename"] == 0 {
			t.Fatal("filesystem parity fixture did not exercise required Go operations", reference.GoFilesystemCalls)
		}
		var mirror bytes.Buffer
		request.StdoutMirror = &mirror
		local, err := Run(t.Context(), request)
		if err != nil {
			t.Fatal(err)
		}
		requireExit(t, "helper namespace", local, 0)
		if local.GoFilesystemCalls == nil || len(local.GoFilesystemCalls) != 0 {
			t.Fatal("filesystem operation escaped helper owner", local.GoFilesystemCalls)
		}
		if difference := sameEvidence(reference, local); difference != "" || !bytes.Equal(reference.DiagnosticTrace.Bytes, local.DiagnosticTrace.Bytes) || reference.EnvironmentIdentity != local.EnvironmentIdentity || reference.Calls != local.Calls || reference.TimeAdvances != local.TimeAdvances {
			t.Fatal("filesystem owner changed evidence", difference)
		}
		if !bytes.Equal(mirror.Bytes(), local.Stdout) || !bytes.Contains(local.Stdout, []byte("namespace owner parity")) || string(local.Stderr) != "namespace stderr\n" {
			t.Fatal("namespace stdout/stderr/mirror changed")
		}
		plan, err := choice.ProjectReplayPlan(reference.ChoiceTrace, request.Runtime.Choice.ExecutionIdentity)
		if err != nil {
			t.Fatal(err)
		}
		control, choices := *request.Runtime, *request.Runtime.Choice
		choices.Mode, choices.Tape = choice.ModeReplay, &plan
		control.Choice = &choices
		forced := request
		forced.Runtime = &control
		forced.StdoutMirror = nil
		replayed, err := Run(t.Context(), forced)
		if err != nil {
			t.Fatal(err)
		}
		if difference := sameEvidence(reference, replayed); difference != "" || replayed.GoFilesystemCalls == nil || len(replayed.GoFilesystemCalls) != 0 {
			t.Fatal("forced namespace replay changed evidence", difference, replayed.GoFilesystemCalls)
		}
	}
}

// TestNamespaceOwnerSQLiteParity runs the file-backed SQLite guest without a
// storage model, so the helper owns its namespace, and requires the Go
// namespace reference to produce identical evidence.
func TestNamespaceOwnerSQLiteParity(t *testing.T) {
	base := storageSQLiteRequest(t)
	base.Config.Storage = nil
	for _, journal := range []string{"WAL", "DELETE"} {
		t.Run(journal, func(t *testing.T) {
			request := storageGuestOperation(base, "setup", journal, nil)
			reference := request
			reference.remoteModel = true
			expected, err := Run(t.Context(), reference)
			if err != nil {
				t.Fatal(err)
			}
			requireExit(t, "Go namespace SQLite", expected, 0)
			if string(expected.Stdout) != "ACK 1\n" || expected.GoFilesystemCalls["fd_pwrite"] == 0 || expected.GoFilesystemCalls["fd_pread"] == 0 {
				t.Fatalf("SQLite reference did not exercise positional I/O: %q %v", expected.Stdout, expected.GoFilesystemCalls)
			}
			local, err := Run(t.Context(), request)
			if err != nil {
				t.Fatal(err)
			}
			requireExit(t, "helper namespace SQLite", local, 0)
			if local.GoFilesystemCalls == nil || len(local.GoFilesystemCalls) != 0 {
				t.Fatal("SQLite filesystem operation escaped helper owner", local.GoFilesystemCalls)
			}
			if difference := sameEvidence(expected, local); difference != "" || expected.EnvironmentIdentity != local.EnvironmentIdentity || expected.Calls != local.Calls || expected.TimeAdvances != local.TimeAdvances {
				t.Fatalf("SQLite namespace owner changed evidence: %s calls=%d/%d advances=%d/%d", difference, expected.Calls, local.Calls, expected.TimeAdvances, local.TimeAdvances)
			}
			t.Logf("SQLite %s: calls=%d go-served=%v authorized Go-owner=%dns helper-owner=%dns", journal, local.Calls, expected.GoFilesystemCalls, expected.AuthorizationToTerminalNanos, local.AuthorizationToTerminalNanos)
		})
	}
}

func TestBrokerCrashRetainsInterruptedRuntimePrefix(t *testing.T) {
	request := cooperativeRequest(t, "../toolchain/testdata/choices")
	requireSmallHelperParityModule(t, request)
	request.Config.Clock.ReadStepNanos = 0
	config := brokerTestConfig()
	config.Nodes = config.Nodes[:1]
	broker, err := NewBroker(config)
	if err != nil {
		t.Fatal(err)
	}
	request.Broker = broker.Session("a")
	if err := request.Broker.CrashOnOutput([]byte("timer")); err != nil {
		t.Fatal(err)
	}
	crashed, err := Run(t.Context(), request)
	if err != nil || crashed.Termination != "crash" || !crashed.Reaped || !crashed.RuntimeInterrupted || len(crashed.InterruptedChoiceBytes) == 0 || len(crashed.ChoiceTrace.Bytes) != 0 {
		t.Fatalf("modeled broker crash prefix missing or claimed complete: termination=%s reaped=%v interrupted=%v bytes=%d complete=%d err=%v message=%s", crashed.Termination, crashed.Reaped, crashed.RuntimeInterrupted, len(crashed.InterruptedChoiceBytes), len(crashed.ChoiceTrace.Bytes), err, crashed.Message)
	}
	tape, err := choice.ProjectInterruptedReplayPlan(crashed.InterruptedChoiceBytes, request.Runtime.Choice.Limit, request.Runtime.Choice.ExecutionIdentity)
	if err != nil {
		t.Fatal(err)
	}
	if crashed.Telemetry == nil || !crashed.Telemetry.ResourceUsageAvailable || crashed.Telemetry.ExecuteNanos == 0 || crashed.AuthorizationToTerminalNanos == 0 || crashed.PeakMemoryBytes == 0 || crashed.FuelRemaining == request.Fuel || crashed.HelperResourceUsage == nil {
		t.Fatalf("modeled broker interruption lost actual helper costs: metrics=%+v wait=%+v execution=%d peak=%d fuel=%d", crashed.Telemetry, crashed.HelperResourceUsage, crashed.AuthorizationToTerminalNanos, crashed.PeakMemoryBytes, crashed.FuelRemaining)
	}
	replayBroker, err := NewBroker(config)
	if err != nil {
		t.Fatal(err)
	}
	request.Broker = replayBroker.Session("a")
	if err := request.Broker.CrashOnOutput([]byte("timer")); err != nil {
		t.Fatal(err)
	}
	control, choices := *request.Runtime, *request.Runtime.Choice
	choices.Mode, choices.Tape = choice.ModePrefix, &tape
	control.Choice, request.Runtime = &choices, &control
	replayed, err := Run(t.Context(), request)
	if err != nil || replayed.Termination != "crash" || !replayed.Reaped || !replayed.RuntimeInterrupted || len(replayed.ChoiceTrace.Bytes) != 0 || replayed.Message != crashed.Message || !bytes.Equal(replayed.InterruptedChoiceBytes, crashed.InterruptedChoiceBytes) || !bytes.Equal(replayed.InterruptedDiagnosticBytes, crashed.InterruptedDiagnosticBytes) || !bytes.Equal(replayed.Transcript, crashed.Transcript) || !bytes.Equal(replayed.NetworkTranscript, crashed.NetworkTranscript) || !bytes.Equal(replayed.Stdout, crashed.Stdout) || !bytes.Equal(replayed.Stderr, crashed.Stderr) || replayed.EnvironmentIdentity != crashed.EnvironmentIdentity {
		t.Fatalf("fresh forced interrupted prefix differs: termination=%s interrupted=%v err=%v message=%s", replayed.Termination, replayed.RuntimeInterrupted, err, replayed.Message)
	}
	t.Logf("actual fresh interrupted-prefix replay: choices=%d transcript=%x diagnostics=%x", len(crashed.InterruptedChoiceBytes), sha256.Sum256(crashed.Transcript), sha256.Sum256(crashed.InterruptedDiagnosticBytes))
}

func requireSmallHelperParityModule(t *testing.T, request Request) {
	t.Helper()
	info, err := os.Stat(request.ModulePath)
	if err != nil {
		t.Fatal(err)
	}
	if info.Size() > 8<<20 {
		t.Fatalf("short parity gate unexpectedly prepared large module: %d bytes", info.Size())
	}
	t.Logf("small parity module=%s bytes=%d helper=%s runtime=%s model=%s", request.ModuleSHA256, info.Size(), request.HelperSHA256, request.Runtime.Choice.ExecutionIdentity.ToolchainBuildKey, ImplementationSHA256())
}

func TestNetworkStreamsGuestRecordsAndReplays(t *testing.T) {
	root := prototypeRoot(t)
	module := prototypeFixture(t, "network_streams")
	config := prototypeFixtureConfig()
	config.Args = []string{"network_streams"}
	config.NetworkV2 = &NetworkModelV2{Profile: NetworkStreamProfile,
		Limits: netmodel.Limits{Listeners: 16, Connections: 128, Bytes: 8 << 20, ConnectionBytes: 4 << 20, StallNanos: 10_000_000_000, Transitions: 16384},
		Faults: []NetworkFaultV2{
			{Name: "stream-delay", Action: "delay", DelayNanos: uint64(5 * time.Millisecond)},
			{Name: "dial-partition", Action: "partition", AtNanos: uint64(50 * time.Millisecond)},
			{Name: "dial-heal", Action: "heal", AtNanos: uint64(90 * time.Millisecond)},
			{Name: "connection-reset", Action: "reset", AtNanos: uint64(120 * time.Millisecond), Port: 9000},
		}}
	request := prototypeRequest(t, module, config, 7, 2*time.Minute)
	request.DigestTranscript = false
	request.MemoryBytes = 256 << 20
	request.Fuel = 1 << 36
	observed, err := Run(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	requireExit(t, "network stream recording", observed, 0)
	for _, want := range []string{"chunks 196745 sha256", "half-close drained", "read and accept deadlines", "partition dial deadline and cancellation", "named reset; new connection survived"} {
		if !bytes.Contains(observed.Stdout, []byte(want)) {
			t.Fatalf("missing %q in guest stdout: %s", want, observed.Stdout)
		}
	}
	for _, want := range []string{"stream-delay", "dial-partition", "dial-heal", "connection-reset", `"kind":"write"`, `"kind":"network"`, `"op":"cancel"`, `"op":"poll"`, `"op":"close_read"`, `"op":"close_write"`} {
		if !bytes.Contains(observed.NetworkTranscript, []byte(want)) {
			t.Fatalf("missing %q in network evidence", want)
		}
	}
	tape, err := choice.ProjectReplayPlan(observed.ChoiceTrace, request.Runtime.Choice.ExecutionIdentity)
	if err != nil {
		t.Fatal(err)
	}
	request.Runtime.Choice.Mode, request.Runtime.Choice.Tape = choice.ModeReplay, &tape
	replayed, err := Run(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	requireExit(t, "network stream replay", replayed, 0)
	if mismatch := sameEvidence(observed, replayed); mismatch != "" {
		t.Fatalf("network stream replay differs: %s", mismatch)
	}
	if !bytes.Equal(observed.Transcript, replayed.Transcript) || len(observed.NetworkTranscript) == 0 {
		t.Fatal("network stream replay transcript differs or network evidence is empty")
	}
	report, err := json.MarshalIndent(struct {
		Schema, ModuleSHA256, HelperSHA256, RuntimeSHA256 string
		NetworkSHA256, WASISHA256, ChoiceSHA256           string
		ModelImplementationSHA256, ProjectionSHA256       string
		Seed                                              uint64
		ChoiceRecords                                     int
		ExactReplay                                       bool
		Model                                             *NetworkModelV2
	}{"gomad.wasm-network-streams-execution/v1", request.ModuleSHA256, request.HelperSHA256,
		request.Runtime.Choice.ExecutionIdentity.ToolchainBuildKey, digest(observed.NetworkTranscript),
		digest(observed.Transcript), digest(observed.ChoiceTrace.Bytes), ImplementationSHA256(), netmodel.ImplementationSHA256(), request.Runtime.Seed,
		len(observed.ChoiceTrace.Records), true, config.NetworkV2}, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	for name, data := range map[string][]byte{
		"network-streams.report.json":  report,
		"network-streams.stdout":       observed.Stdout,
		"network-streams.wasi.bin":     observed.Transcript,
		"network-streams.network.json": observed.NetworkTranscript,
		"network-streams.choices":      observed.ChoiceTrace.Bytes,
	} {
		if err := os.WriteFile(filepath.Join(root, name), data, 0600); err != nil {
			t.Fatal(err)
		}
	}
	t.Logf("actual stream bytes, parked readiness and exact replay: network=%s wasi=%s choices=%d", digest(observed.NetworkTranscript), digest(observed.Transcript), len(observed.ChoiceTrace.Records))
}

func storageSQLiteRequest(t *testing.T) Request {
	t.Helper()
	helper := helperPath(t)
	version, err := exec.Command(stockgo.Command(), "version").Output()
	if err != nil || !strings.Contains(string(version), "go1.27.1 ") {
		t.Fatalf("storage guest requires stock Go 1.27.1: %q %v", version, err)
	}
	root := stockgo.Root(t)
	native, err := filepath.Abs("../../../model/runtimesource")
	if err != nil {
		t.Fatal(err)
	}
	overlay, identity, err := toolchain.BuildOverlay(root, native, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	module := filepath.Join(t.TempDir(), "storage.wasm")
	fixture, err := filepath.Abs("../testdata/storage_sqlite/main.go")
	if err != nil {
		t.Fatal(err)
	}
	command := exec.Command(filepath.Join(root, "bin/go"), "build", "-trimpath", "-tags", "sqlite3_dotlk", "-overlay", overlay, "-o", module, fixture)
	command.Dir = "../../../../../.."
	command.Env = append(os.Environ(), "GOROOT="+root, "GOOS=wasip1", "GOARCH=wasm", "GOEXPERIMENT=nogreenteagc", "GOWORK=off", "GOENV=off", "GOFLAGS=-mod=readonly", "GOTOOLCHAIN=local", "CGO_ENABLED=0")
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("build file-backed SQLite guest: %v\n%s", err, output)
	}
	data, err := os.ReadFile(module)
	if err != nil {
		t.Fatal(err)
	}
	imports, pages := binaryInventory(t, data)
	config := testConfig()
	config.Profile = CooperativeProfile
	config.WritableDirectories = append(config.WritableDirectories, "/volume")
	config.Limits.Calls = 100000
	config.Limits.TranscriptBytes = 32 << 20
	config.Storage = &StorageModel{Volumes: []volume.VolumeConfig{{ID: "sqlite", Path: "/volume", CapacityBytes: 4 << 20}}, Limits: volume.VolumeLimits{PendingOperations: 10000, Transitions: 100000}}
	execution := choice.ExecutionIdentity{TargetSHA256: sha256.Sum256(data), ToolchainBuildKey: identity, GOOS: "wasip1", GOARCH: "wasm", ImplementationSHA256: sha256.Sum256([]byte(identity))}
	t.Logf("actual SQLite guest identities: module=%s helper=%s runtime=%s model=%s", fileHash(t, module), fileHash(t, helper), identity, ImplementationSHA256())
	cache := t.TempDir()
	if err := os.Chmod(cache, 0700); err != nil {
		t.Fatal(err)
	}
	return Request{CompileCache: cache, HelperPath: helper, HelperSHA256: fileHash(t, helper), ModulePath: module, ModuleSHA256: fileHash(t, module), Engine: PinnedEngine(), Imports: imports, InitialMemoryPages: pages, MemoryBytes: 256 << 20, Fuel: 4000000000, Config: config, Timeout: time.Minute, Runtime: &RuntimeControl{Seed: 7, Choice: &runnerbackend.ChoiceRequest{Mode: choice.ModeRecord, ExecutionIdentity: execution, Limit: 1 << 20}}}
}

func storageGuestOperation(request Request, operation, journal string, image *StorageImage) Request {
	request.Config.Args = []string{"storage", operation, journal}
	request.StorageImage = image
	control := *request.Runtime
	choices := *control.Choice
	control.Choice = &choices
	request.Runtime = &control
	return request
}

func runStorageGuest(t *testing.T, request Request) Result {
	t.Helper()
	result, err := Run(t.Context(), request)
	if err != nil || !result.Reaped || result.StorageImage == nil {
		t.Fatalf("storage guest: %s %s reaped=%v image=%v err=%v stdout=%s stderr=%s", result.Termination, result.Message, result.Reaped, result.StorageImage != nil, err, result.Stdout, result.Stderr)
	}
	return result
}

func requireStorageGuestExit(t *testing.T, result Result) {
	t.Helper()
	if result.Termination != "exit" || result.ExitCode == nil || *result.ExitCode != 0 {
		t.Fatalf("storage guest failed: %s %s exit=%v stdout=%s stderr=%s", result.Termination, result.Message, result.ExitCode, result.Stdout, result.Stderr)
	}
}

func replayStorageGuest(t *testing.T, request Request, recorded Result) Result {
	t.Helper()
	if recorded.Termination == "exit" || recorded.Termination == "storage_crash" {
		var tape choice.ReplayPlan
		var err error
		mode := choice.ModeReplay
		if recorded.Termination == "storage_crash" {
			if !recorded.RuntimeInterrupted || len(recorded.ChoiceTrace.Bytes) != 0 {
				t.Fatal("storage crash was not retained as interrupted runtime evidence")
			}
			tape, err = choice.ProjectInterruptedReplayPlan(recorded.InterruptedChoiceBytes, request.Runtime.Choice.Limit, request.Runtime.Choice.ExecutionIdentity)
			mode = choice.ModePrefix
		} else {
			tape, err = choice.ProjectReplayPlan(recorded.ChoiceTrace, request.Runtime.Choice.ExecutionIdentity)
		}
		if err != nil {
			t.Fatal(err)
		}
		control := *request.Runtime
		choices := *control.Choice
		choices.Mode, choices.Tape = mode, &tape
		control.Choice = &choices
		request.Runtime = &control
	}
	replayed := runStorageGuest(t, request)
	sameExit := replayed.ExitCode == nil && recorded.ExitCode == nil || replayed.ExitCode != nil && recorded.ExitCode != nil && *replayed.ExitCode == *recorded.ExitCode
	if replayed.Termination != recorded.Termination || !sameExit || replayed.Message != recorded.Message || !bytes.Equal(replayed.Stdout, recorded.Stdout) || !bytes.Equal(replayed.Stderr, recorded.Stderr) || !bytes.Equal(replayed.Transcript, recorded.Transcript) || !bytes.Equal(replayed.StorageTranscript, recorded.StorageTranscript) || !bytes.Equal(replayed.ChoiceTrace.Bytes, recorded.ChoiceTrace.Bytes) || !bytes.Equal(replayed.InterruptedChoiceBytes, recorded.InterruptedChoiceBytes) || !bytes.Equal(replayed.InterruptedDiagnosticBytes, recorded.InterruptedDiagnosticBytes) || replayed.RuntimeInterrupted != recorded.RuntimeInterrupted || replayed.StorageImage.Identity() != recorded.StorageImage.Identity() || replayed.EnvironmentIdentity != recorded.EnvironmentIdentity {
		t.Fatalf("fresh storage replay differs: %s/%s message=%q/%q stdout=%s/%s storage=%x/%x choice=%x/%x image=%s/%s", recorded.Termination, replayed.Termination, recorded.Message, replayed.Message, recorded.Stdout, replayed.Stdout, sha256.Sum256(recorded.StorageTranscript), sha256.Sum256(replayed.StorageTranscript), recorded.ChoiceTrace.SHA256, replayed.ChoiceTrace.SHA256, recorded.StorageImage.Identity(), replayed.StorageImage.Identity())
	}
	return replayed
}

func storageGuestEvents(t *testing.T, result Result) []StorageEvent {
	t.Helper()
	decoder := json.NewDecoder(bytes.NewReader(result.StorageTranscript))
	var events []StorageEvent
	for decoder.More() {
		var event StorageEvent
		if err := decoder.Decode(&event); err != nil {
			t.Fatal(err)
		}
		events = append(events, event)
	}
	return events
}

func requireStorageHistory(t *testing.T, request Request, image *StorageImage, history string) Result {
	t.Helper()
	request.StorageImage = image
	result := runStorageGuest(t, request)
	requireStorageGuestExit(t, result)
	// Exact ordered rows cover both lost and duplicate effects; integrity_check
	// independently checks the recovered database's structural invariant.
	want := "INTEGRITY ok\nHISTORY " + history + "\n"
	if string(result.Stdout) != want {
		t.Fatalf("recovered history: %q, want %q", result.Stdout, want)
	}
	replayStorageGuest(t, request, result)
	// A second incarnation must reach the same state within two fresh opens.
	request.StorageImage = result.StorageImage
	converged := runStorageGuest(t, request)
	requireStorageGuestExit(t, converged)
	if !bytes.Equal(result.Stdout, converged.Stdout) {
		t.Fatalf("recovered history failed bounded convergence: %s/%s", result.Stdout, converged.Stdout)
	}
	return result
}

func TestSQLiteStorageGuestRecovery(t *testing.T) {
	base := storageSQLiteRequest(t)
	t.Run("raw-durable-lock", func(t *testing.T) {
		request := storageGuestOperation(base, "raw-lock", "WAL", nil)
		locked := runStorageGuest(t, request)
		requireStorageGuestExit(t, locked)
		if string(locked.Stdout) != "RAW LOCK durable\n" {
			t.Fatalf("raw lock not acknowledged: %s", locked.Stdout)
		}
		replayStorageGuest(t, request, locked)
		probe := storageGuestOperation(base, "probe-lock", "WAL", locked.StorageImage)
		blocked := runStorageGuest(t, probe)
		requireStorageGuestExit(t, blocked)
		var code, extended int
		if _, err := fmt.Sscanf(string(blocked.Stdout), "LOCKED code=%d extended=%d\n", &code, &extended); err != nil || code != 5 && code != 6 || extended == 0 {
			t.Fatalf("durable lock did not block fresh SQLite with typed BUSY/LOCKED: %s %v", blocked.Stdout, err)
		}
		replayStorageGuest(t, probe, blocked)
		if !slices.ContainsFunc(blocked.StorageSnapshots[0].Persisted, func(entry volume.SnapshotEntry) bool { return entry.Path == "/db.lock" }) {
			t.Fatal("raw persisted lock was deleted during recovery")
		}
		t.Logf("raw durable lock blocks reopened SQLite: code=%d extended=%d", code, extended)
	})
	t.Run("acknowledged-WAL-crash", func(t *testing.T) {
		request := base
		model := *base.Config.Storage
		model.SQLiteLockDatabases = []string{"/volume/db"}
		model.Faults = []StorageFault{{Name: "acknowledged-crash", Resource: "/volume/crash-trigger", Operation: "open", Occurrence: 1, Action: "crash-before"}}
		request.Config.Storage = &model
		setup := runStorageGuest(t, storageGuestOperation(request, "setup-live", "WAL", nil))
		requireStorageGuestExit(t, setup)
		crashRequest := storageGuestOperation(request, "append-crash", "WAL", setup.StorageImage)
		crashRequest.StorageCrashLimits = &volume.CrashEnumerationLimits{States: 32, Operations: 8, Depth: 8, Bytes: 4 << 20, Work: 1 << 16}
		crashed := runStorageGuest(t, crashRequest)
		if crashed.Termination != "storage_crash" || string(crashed.Stdout) != "ACK 2\n" {
			t.Fatalf("acknowledged WAL was not crashed with SQLite open: %s %s %s", crashed.Termination, crashed.Message, crashed.Stdout)
		}
		replayed := replayStorageGuest(t, crashRequest, crashed)
		recoverRequest := storageGuestOperation(request, "recover", "WAL", nil)
		requireStorageHistory(t, recoverRequest, crashed.StorageImage, "1:committed,2:committed")
		requireStorageHistory(t, recoverRequest, replayed.StorageImage, "1:committed,2:committed")
		enumeration := crashed.StorageCrashEnumeration["sqlite"]
		if !enumeration.Complete || enumeration.Capacity != "" || len(enumeration.States) <= 1 || crashed.StorageSnapshots[0].PendingOperations == 0 {
			t.Fatalf("live SQLite crash envelope incomplete or trivial: %+v", enumeration)
		}
		markerStates := map[string]bool{}
		for index, state := range enumeration.States {
			t.Run(fmt.Sprintf("selection-%d", index), func(t *testing.T) {
				selectedRequest := crashRequest
				selectedRequest.StorageCrashSelection = map[string][]uint64{"sqlite": state.SelectedOperations}
				selected := runStorageGuest(t, selectedRequest)
				if selected.Termination != "storage_crash" || string(selected.Stdout) != "ACK 2\n" {
					t.Fatalf("selected acknowledged crash: %s %s %s", selected.Termination, selected.Message, selected.Stdout)
				}
				replayStorageGuest(t, selectedRequest, selected)
				recovered := requireStorageHistory(t, recoverRequest, selected.StorageImage, "1:committed,2:committed")
				predicted := slices.IndexFunc(state.Entries, func(entry volume.CrashEntry) bool { return entry.Path == "/marker" })
				actual := slices.IndexFunc(recovered.StorageSnapshots[0].Persisted, func(entry volume.SnapshotEntry) bool { return entry.Path == "/marker" })
				if predicted < 0 {
					if actual >= 0 {
						t.Fatal("unselected marker creation survived the crash")
					}
					markerStates["absent"] = true
					t.Logf("selected operations=%v: marker absent", state.SelectedOperations)
					return
				}
				data := state.Entries[predicted].Data
				if actual < 0 || recovered.StorageSnapshots[0].Persisted[actual].Size != uint64(len(data)) || recovered.StorageSnapshots[0].Persisted[actual].DataSHA256 != fmt.Sprintf("sha256:%x", sha256.Sum256(data)) {
					t.Fatal("actual recovered marker differs from the reviewed native crash state")
				}
				markerStates[string(data)] = true
				t.Logf("selected operations=%v: marker bytes=%d sha256=%s", state.SelectedOperations, len(data), recovered.StorageSnapshots[0].Persisted[actual].DataSHA256)
			})
		}
		if len(markerStates) < 3 {
			t.Fatalf("crash selections did not realize absent, empty and written marker states: %v", markerStates)
		}
		t.Logf("all %d bounded dependency-valid SQLite crash states preserved ACK2; marker states=%d pending operations=%d", len(enumeration.States), len(markerStates), crashed.StorageSnapshots[0].PendingOperations)
	})
	t.Run("live-WAL-commit", func(t *testing.T) {
		request := base
		model := *base.Config.Storage
		model.SQLiteLockDatabases = []string{"/volume/db"}
		request.Config.Storage = &model
		setupRequest := storageGuestOperation(request, "setup-live", "WAL", nil)
		setup := runStorageGuest(t, setupRequest)
		requireStorageGuestExit(t, setup)
		replayStorageGuest(t, setupRequest, setup)
		appendRequest := storageGuestOperation(request, "append-live", "WAL", setup.StorageImage)
		appended := runStorageGuest(t, appendRequest)
		requireStorageGuestExit(t, appended)
		if string(appended.Stdout) != "ACK 2\n" {
			t.Fatalf("live WAL transaction not acknowledged: %s", appended.Stdout)
		}
		replayStorageGuest(t, appendRequest, appended)
		if !slices.ContainsFunc(storageGuestEvents(t, appended), func(event StorageEvent) bool { return event.Resource == "/volume/db-wal" && event.Operation == "sync" }) {
			t.Fatal("live WAL commit did not sync its log")
		}
		if !slices.ContainsFunc(appended.StorageSnapshots[0].Persisted, func(entry volume.SnapshotEntry) bool { return entry.Path == "/db-wal" }) {
			t.Fatal("live WAL disappeared before crash image capture")
		}
		requireStorageHistory(t, storageGuestOperation(request, "recover", "WAL", nil), appended.StorageImage, "1:committed,2:committed")
	})
	for _, journal := range []string{"WAL", "DELETE"} {
		t.Run(journal, func(t *testing.T) {
			setupRequest := storageGuestOperation(base, "setup", journal, nil)
			setup := runStorageGuest(t, setupRequest)
			requireStorageGuestExit(t, setup)
			if len(setup.ChoiceTrace.Records) == 0 {
				t.Fatal("SQLite guest produced no cooperative decisions")
			}
			replayStorageGuest(t, setupRequest, setup)
			appendRequest := storageGuestOperation(base, "append", journal, setup.StorageImage)
			appended := runStorageGuest(t, appendRequest)
			requireStorageGuestExit(t, appended)
			if string(appended.Stdout) != "ACK 2\n" {
				t.Fatalf("successful commit not acknowledged: %s", appended.Stdout)
			}
			replayStorageGuest(t, appendRequest, appended)
			recoverRequest := storageGuestOperation(base, "recover", journal, nil)
			requireStorageHistory(t, recoverRequest, appended.StorageImage, "1:committed,2:committed")
			events := storageGuestEvents(t, appended)
			for _, operation := range []string{"read", "write", "sync"} {
				if !slices.ContainsFunc(events, func(event StorageEvent) bool { return event.Operation == operation }) {
					t.Fatalf("SQLite bypassed storage %s: %+v", operation, events)
				}
			}
			records, err := DecodeTranscript(appended.Transcript)
			if err != nil {
				t.Fatal(err)
			}
			for _, operation := range []string{"fd_pread", "fd_pwrite", "fd_sync"} {
				if !slices.ContainsFunc(records, func(record TranscriptRecord) bool { return record.Op == operation }) {
					t.Fatalf("SQLite did not reach WASI %s", operation)
				}
			}
			resource := "/volume/db-journal"
			if journal == "WAL" {
				resource = "/volume/db-wal"
			}
			for _, action := range []string{"short-write", "io-error", "no-space", "sync-error", "torn-write", "crash-before"} {
				t.Run(action, func(t *testing.T) {
					operation := "write"
					if action == "sync-error" || action == "crash-before" {
						operation = "sync"
					}
					index := slices.IndexFunc(events, func(event StorageEvent) bool { return event.Resource == resource && event.Operation == operation })
					if index < 0 {
						t.Fatalf("missing SQLite %s %s", resource, operation)
					}
					fault := StorageFault{Name: "sqlite-" + action, Resource: resource, Operation: operation, Occurrence: events[index].Occurrence, Action: action}
					if action == "short-write" || action == "torn-write" {
						fault.Bytes = 1
					}
					request := base
					model := *base.Config.Storage
					model.Faults = []StorageFault{fault}
					request.Config.Storage = &model
					initialized := runStorageGuest(t, storageGuestOperation(request, "setup", journal, nil))
					requireStorageGuestExit(t, initialized)
					faultRequest := storageGuestOperation(request, "append", journal, initialized.StorageImage)
					failed := runStorageGuest(t, faultRequest)
					switch action {
					case "crash-before":
						if failed.Termination != "storage_crash" {
							t.Fatalf("fault did not kill helper: %s %s", failed.Termination, failed.Message)
						}
					case "short-write":
						requireStorageGuestExit(t, failed)
						if string(failed.Stdout) != "ACK 2\n" {
							t.Fatalf("Go WriteAt did not complete the short write: %s", failed.Stdout)
						}
					default:
						requireStorageGuestExit(t, failed)
						var code, extended int
						if _, err := fmt.Sscanf(string(failed.Stdout), "REJECTED 2 code=%d extended=%d\n", &code, &extended); err != nil || extended == 0 || code != 10 && (action != "no-space" || code != 13) {
							t.Fatalf("faulty commit did not produce SQLite IOERR/FULL: %s (%v)", failed.Stdout, err)
						}
						t.Logf("actual SQLite storage fault %s: code=%d extended=%d", action, code, extended)
					}
					if !slices.ContainsFunc(storageGuestEvents(t, failed), func(event StorageEvent) bool { return event.Fault == fault.Name && event.Action == action }) {
						t.Fatal("storage fault not realized by actual SQLite guest")
					}
					faultReplay := replayStorageGuest(t, faultRequest, failed)
					recoveredRequest := storageGuestOperation(request, "recover", journal, nil)
					history := "1:committed"
					if action == "short-write" {
						history += ",2:committed"
					}
					recovered := requireStorageHistory(t, recoveredRequest, failed.StorageImage, history)
					replayedRecovery := requireStorageHistory(t, recoveredRequest, faultReplay.StorageImage, history)
					if !bytes.Equal(recovered.Stdout, replayedRecovery.Stdout) || !bytes.Equal(recovered.StorageTranscript, replayedRecovery.StorageTranscript) {
						t.Fatal("replayed fault recovered a different history")
					}
				})
			}
		})
	}
}

func TestSQLiteStorageGuestUnsyncedAcknowledgementCanary(t *testing.T) {
	request := storageSQLiteRequest(t)
	setup := runStorageGuest(t, storageGuestOperation(request, "setup", "WAL", nil))
	requireStorageGuestExit(t, setup)
	unsyncedRequest := storageGuestOperation(request, "unsynced", "WAL", setup.StorageImage)
	unsynced := runStorageGuest(t, unsyncedRequest)
	requireStorageGuestExit(t, unsynced)
	if string(unsynced.Stdout) != "ACK 2\n" {
		t.Fatalf("negative canary did not acknowledge: %s", unsynced.Stdout)
	}
	replayStorageGuest(t, unsyncedRequest, unsynced)
	recovered := requireStorageHistory(t, storageGuestOperation(request, "recover", "WAL", nil), unsynced.StorageImage, "1:committed")
	if bytes.Contains(recovered.Stdout, []byte("2:committed")) {
		t.Fatal("missing-sync durability violation escaped the history oracle")
	}
}

const temporalObservationPrefix = "GOMAD_TEMPORAL_OBSERVATION "

type temporalRecoveryObservation struct {
	ClusterID string   `json:"cluster_id"`
	History   []byte   `json:"history"`
	EventIDs  []int64  `json:"event_ids"`
	Types     []string `json:"types"`
}

func temporalRecoveryValue(t *testing.T, stdout []byte) temporalRecoveryObservation {
	t.Helper()
	var value temporalRecoveryObservation
	count := 0
	for _, line := range bytes.Split(stdout, []byte{'\n'}) {
		if !bytes.HasPrefix(line, []byte(temporalObservationPrefix)) {
			continue
		}
		if err := json.Unmarshal(line[len(temporalObservationPrefix):], &value); err != nil {
			t.Fatal(err)
		}
		count++
	}
	if count != 1 || value.ClusterID == "" || len(value.History) == 0 || len(value.EventIDs) != len(value.Types) {
		t.Fatalf("invalid detached Temporal history observation: count=%d value=%+v", count, value)
	}
	for i, id := range value.EventIDs {
		if id != int64(i+1) {
			t.Fatalf("duplicate or lost history event: %v", value.EventIDs)
		}
	}
	return value
}

func temporalHistoryOracles(t *testing.T, evidence, name string, expected, actual temporalRecoveryObservation) {
	t.Helper()
	history := func(value temporalRecoveryObservation) []gomad3sim.HistoryOperation {
		return []gomad3sim.HistoryOperation{{ID: "committed", Actor: "workflow", Kind: "workflow-history", Invocation: 1, Completion: 2, Output: value.History}}
	}
	exact, err := gomad3sim.ExactHistory(name+".history", history(expected), history(actual), 1<<20)
	if err != nil || !exact.Passed {
		t.Fatalf("detached Temporal history oracle: passed=%v err=%v", exact.Passed, err)
	}
	ids := func(value temporalRecoveryObservation) []string {
		values := make([]string, len(value.EventIDs))
		for i, id := range value.EventIDs {
			values[i] = fmt.Sprint(id)
		}
		return values
	}
	temporalJSON(t, filepath.Join(evidence, name+"-oracle-inputs.json"), struct {
		ExpectedHistory, ActualHistory []gomad3sim.HistoryOperation
		ExpectedEvents, ActualEvents   []string
		ExpectedCluster, ActualCluster string
	}{history(expected), history(actual), ids(expected), ids(actual), expected.ClusterID, actual.ClusterID})
	delivery, err := gomad3sim.NoDuplicateOrLost(name+".events", ids(expected), ids(actual), 1<<20)
	if err != nil || !delivery.Passed {
		t.Fatalf("detached Temporal duplicate/lost oracle: passed=%v err=%v", delivery.Passed, err)
	}
	if expected.ClusterID != actual.ClusterID {
		t.Fatal("recovered Temporal cluster identity changed")
	}
	temporalJSON(t, filepath.Join(evidence, name+"-oracles.json"), []gomad3sim.OracleResult{exact, delivery})
}

func temporalRecoveryRequest(t *testing.T, fixture string) (Request, string) {
	t.Helper()
	return temporalRecoveryRequestWithAdmission(t, fixture, nil)
}

func temporalCompileCachePath(evidence, override string) (string, error) {
	if override == "" {
		return filepath.Join(evidence, "compile-cache"), nil
	}
	if !filepath.IsAbs(override) || filepath.Clean(override) != override || override == string(filepath.Separator) {
		return "", fmt.Errorf("compile cache must be an explicit clean absolute directory: %q", override)
	}
	return override, nil
}

func temporalRetainRecoveryRun(t *testing.T, evidence, label string, result Result) {
	t.Helper()
	temporalWrite(t, filepath.Join(evidence, label+".stdout"), result.Stdout)
	temporalWrite(t, filepath.Join(evidence, label+".stderr"), result.Stderr)
	temporalWrite(t, filepath.Join(evidence, label+".storage.jsonl"), result.StorageTranscript)
	temporalWrite(t, filepath.Join(evidence, label+".network.json"), result.NetworkTranscript)
	choices, diagnostics := result.ChoiceTrace.Bytes, result.DiagnosticTrace.Bytes
	if result.RuntimeInterrupted {
		choices, diagnostics = result.InterruptedChoiceBytes, result.InterruptedDiagnosticBytes
	}
	temporalWrite(t, filepath.Join(evidence, label+".choices"), choices)
	temporalWrite(t, filepath.Join(evidence, label+".diagnostics"), diagnostics)
	temporalJSON(t, filepath.Join(evidence, label+".snapshots.json"), result.StorageSnapshots)
	temporalJSON(t, filepath.Join(evidence, label+".operations.json"), result.StorageOperations)
	if result.StorageCrashEnumeration != nil {
		temporalJSON(t, filepath.Join(evidence, label+".crash-states.json"), result.StorageCrashEnumeration)
	}
	if result.Telemetry != nil {
		temporalJSON(t, filepath.Join(evidence, label+".telemetry.json"), result.Telemetry)
	}
	if result.HelperResourceUsage != nil {
		temporalJSON(t, filepath.Join(evidence, label+".host-cost.json"), result.HelperResourceUsage)
	}
	temporalJSON(t, filepath.Join(evidence, label+".terminal.json"), struct {
		Termination, Message, EnvironmentIdentity string
		ExitCode                                  *uint32
		Reaped, Cancelled, WatchdogTimeout        bool
		RuntimeInterrupted                        bool
		Calls, TimeAdvances, VirtualTimeNanos     uint64
	}{result.Termination, result.Message, result.EnvironmentIdentity, result.ExitCode, result.Reaped, result.Cancelled,
		result.WatchdogTimeout, result.RuntimeInterrupted, result.Calls, result.TimeAdvances, result.VirtualTimeElapsedNanos})
}

func temporalRetainRequest(t *testing.T, evidence, label string, request Request) {
	t.Helper()
	request.Broker, request.StdoutMirror = nil, nil
	temporalJSON(t, filepath.Join(evidence, label+".request.json"), request)
}

func temporalStorageBrokerConfiguration() BrokerConfig {
	return BrokerConfig{Nodes: []BrokerNode{{ID: "temporal", Incarnation: 1}}, Seed: 7,
		Limits: world.Limits{MaxRequests: 1000000, MaxEvents: 1000000, MaxQueuedEvents: 65536, MaxTransitions: 4000000, MaxPayloadBytes: 64 << 20, MaxStringBytes: 1024}}
}

func TestTemporalStorageBrokerAdmission(t *testing.T) {
	config := temporalStorageBrokerConfiguration()
	if _, err := NewBroker(config); err != nil {
		t.Fatal(err)
	}
	if config.Limits.MaxRequests > config.Limits.MaxPayloadBytes/(2*sha256.Size) {
		t.Fatal("storage broker can exhaust payload bytes before its request bound")
	}
	config.Limits.MaxPayloadBytes++
	if _, err := NewBroker(config); err == nil {
		t.Fatal("storage broker admitted payload bytes beyond the compressed World profile")
	}
}

func TestTemporalStorageBrokerPayloadAccounting(t *testing.T) {
	for _, test := range []struct {
		name      string
		bytes     uint64
		accepted  uint64
		dimension string
	}{{"request-first", 192, 3, "requests"}, {"payload-double-accounting", 191, 2, "payload-bytes"}} {
		t.Run(test.name, func(t *testing.T) {
			config := temporalStorageBrokerConfiguration()
			config.Limits.MaxRequests, config.Limits.MaxPayloadBytes = 3, test.bytes
			broker, err := NewBroker(config)
			if err != nil {
				t.Fatal(err)
			}
			session := broker.Session("temporal")
			payload := bytes.Repeat([]byte("storage"), 128)
			for range test.accepted {
				if err := session.register("storage", payload); err != nil {
					t.Fatal(err)
				}
				if err := session.ready("storage"); err != nil {
					t.Fatal(err)
				}
			}
			before := broker.world.DiagnosticCardinality()
			if before.Requests != test.accepted || before.Events != test.accepted || before.PayloadBytes != test.accepted*64 {
				t.Fatalf("broker payload accounting = %+v", before)
			}
			err = session.register("storage", payload)
			var capacity *world.CapacityError
			if !errors.As(err, &capacity) || capacity.Dimension != test.dimension {
				t.Fatalf("broker boundary = %v, want %s capacity", err, test.dimension)
			}
			if after := broker.world.DiagnosticCardinality(); after != before {
				t.Fatalf("rejected request changed World: before=%+v after=%+v", before, after)
			}
		})
	}
}

func TestTemporalCommittedWorkflowFreshGuestCrashReopen(t *testing.T) {
	request, evidence := temporalRecoveryRequest(t, "temporal_storage")
	request.Config.Args = []string{"recovery.test", "-test.v", "-test.count=1", "-test.timeout=1h", "-test.run=^TestCommittedWorkflowCrashRecovery$", "-persistenceType=sql", "-persistenceDriver=sqlite"}
	request.Config.Environment = append(request.Config.Environment, "GOMAD_TEMPORAL_PHASE=commit")
	request.StorageCrashLimits = &volume.CrashEnumerationLimits{States: 128, Operations: 256, Depth: 256, Bytes: 256 << 20, Work: 1 << 20}
	temporalRetainRequest(t, evidence, "temporal-commit", request)
	newBroker := func() *Broker {
		b, err := NewBroker(temporalStorageBrokerConfiguration())
		if err != nil {
			t.Fatal(err)
		}
		if err := b.Session("temporal").CrashOnOutput([]byte(temporalObservationPrefix)); err != nil {
			t.Fatal(err)
		}
		return b
	}
	b := newBroker()
	request.Broker = b.Session("temporal")
	committed, err := Run(t.Context(), request)
	temporalRetainRecoveryRun(t, evidence, "temporal-committed-crash", committed)
	if err != nil || committed.Termination != "crash" || !committed.Reaped || committed.StorageImage == nil {
		t.Fatalf("abrupt committed guest crash: termination=%s message=%s err=%v stdout=%s stderr=%s", committed.Termination, committed.Message, err, committed.Stdout, committed.Stderr)
	}
	expected := temporalRecoveryValue(t, committed.Stdout)
	crashRequest := request
	if !committed.RuntimeInterrupted {
		t.Fatal("Temporal crash lost interrupted runtime evidence")
	}
	tape, err := choice.ProjectInterruptedReplayPlan(committed.InterruptedChoiceBytes, request.Runtime.Choice.Limit, request.Runtime.Choice.ExecutionIdentity)
	if err != nil {
		t.Fatal(err)
	}
	control := *request.Runtime
	choices := *control.Choice
	choices.Mode, choices.Tape = choice.ModePrefix, &tape
	control.Choice = &choices
	crashRequest.Runtime = &control
	enumeration := committed.StorageCrashEnumeration["temporal"]
	if !enumeration.Complete || len(enumeration.States) == 0 {
		t.Fatalf("Temporal supported crash-state enumeration remains incomplete: capacity=%s states=%d", enumeration.Capacity, len(enumeration.States))
	}
	for i, state := range enumeration.States {
		b := newBroker()
		crashRequest.Broker = b.Session("temporal")
		crashRequest.StorageCrashSelection = map[string][]uint64{"temporal": state.SelectedOperations}
		crashed, err := Run(t.Context(), crashRequest)
		temporalRetainRecoveryRun(t, evidence, fmt.Sprintf("temporal-state-%d-crash", i), crashed)
		if err != nil || crashed.Termination != "crash" || !crashed.Reaped || crashed.StorageImage == nil || !bytes.Equal(committed.Stdout, crashed.Stdout) || !bytes.Equal(committed.StorageTranscript, crashed.StorageTranscript) || !bytes.Equal(committed.InterruptedChoiceBytes, crashed.InterruptedChoiceBytes) {
			t.Fatalf("reviewed Temporal crash state %d did not replay its exact committed boundary: termination=%s message=%s err=%v", i, crashed.Termination, crashed.Message, err)
		}
		reopen := request
		reopen.StorageCrashLimits = nil
		reopen.StorageImage = crashed.StorageImage
		reopen.Config.Environment = append([]string(nil), request.Config.Environment...)
		reopen.Config.Environment[len(reopen.Config.Environment)-1] = "GOMAD_TEMPORAL_PHASE=reopen"
		reopen.Broker, err = b.Restart("temporal")
		if err != nil {
			t.Fatal(err)
		}
		temporalRetainRequest(t, evidence, fmt.Sprintf("temporal-state-%d-recovered", i), reopen)
		recovered, err := Run(t.Context(), reopen)
		temporalRetainRecoveryRun(t, evidence, fmt.Sprintf("temporal-state-%d-recovered", i), recovered)
		if err != nil || recovered.Termination != "exit" || recovered.ExitCode == nil || *recovered.ExitCode != 0 || !recovered.Reaped || !reflect.DeepEqual(expected, temporalRecoveryValue(t, recovered.Stdout)) {
			t.Fatalf("reviewed Temporal crash state %d lost acknowledged history/result: termination=%s message=%s err=%v", i, recovered.Termination, recovered.Message, err)
		}
		temporalHistoryOracles(t, evidence, fmt.Sprintf("temporal.recovery.state-%d", i), expected, temporalRecoveryValue(t, recovered.Stdout))
		temporalJSON(t, filepath.Join(evidence, fmt.Sprintf("temporal-state-%d-selection.json", i)), struct {
			Identity   string
			Operations []uint64
		}{state.Identity, state.SelectedOperations})
	}
	request.StdoutMirror = nil
	request.StorageCrashLimits = nil
	request.StorageImage = committed.StorageImage
	request.Config.Environment[len(request.Config.Environment)-1] = "GOMAD_TEMPORAL_PHASE=reopen"
	for incarnation := range 2 {
		request.Broker, err = b.Restart("temporal")
		if err != nil {
			t.Fatal(err)
		}
		reopened, err := Run(t.Context(), request)
		temporalRetainRecoveryRun(t, evidence, fmt.Sprintf("temporal-reopened-%d", incarnation), reopened)
		if err != nil || reopened.Termination != "exit" || reopened.ExitCode == nil || *reopened.ExitCode != 0 || !reopened.Reaped {
			t.Fatalf("fresh recovered guest: termination=%s message=%s err=%v stdout=%s stderr=%s", reopened.Termination, reopened.Message, err, reopened.Stdout, reopened.Stderr)
		}
		actual := temporalRecoveryValue(t, reopened.Stdout)
		temporalHistoryOracles(t, evidence, fmt.Sprintf("temporal.recovery.incarnation-%d", incarnation), expected, actual)
		if !reflect.DeepEqual(expected, actual) {
			t.Fatalf("detached committed history or metadata identity changed after crash: expected=%+v actual=%+v", expected, actual)
		}
	}
	temporalJSON(t, filepath.Join(evidence, "temporal-history-oracle.json"), expected)
}

func TestTemporalGuestAdmission(t *testing.T) {
	target := os.Getenv("GOMAD_WASM_TEMPORAL_TARGET")
	if target == "" {
		t.Skip("set GOMAD_WASM_TEMPORAL_TARGET to run the explicit Temporal guest gate")
	}
	clockSelection := os.Getenv("GOMAD_WASM_TEMPORAL_CLOCK")
	clock, err := temporalClockPolicy(clockSelection)
	if err != nil {
		t.Fatal(err)
	}
	root, err := filepath.Abs("../../../../../..")
	if err != nil {
		t.Fatal(err)
	}
	evidence := filepath.Join(root, ".tmp", "wasm-backend-temporal")
	adapterMode := os.Getenv("GOMAD_WASM_SQLITE_ADAPTER")
	if adapterMode != "" && adapterMode != "cooperative-v1" {
		t.Fatalf("unknown SQLite adapter %q", adapterMode)
	}
	if target == "grpc-progress-diagnostic" && adapterMode != "" {
		t.Fatal("standalone gRPC diagnostic requires the original dependency graph without SQLite adapter")
	}
	projectionControl := os.Getenv("GOMAD_WASM_SQLITE_PROJECTION_CONTROL") == "1"
	if projectionControl && (adapterMode == "" || target != "contention") {
		t.Fatal("projection-only control requires explicit contention adapter selection")
	}
	if adapterMode != "" {
		evidence = filepath.Join(evidence, "cooperative-sqlite-v1")
		if projectionControl {
			evidence = filepath.Join(evidence, "projection-control")
		}
	}
	if clockSelection != "" {
		evidence = filepath.Join(evidence, clockSelection)
	}
	if err := os.MkdirAll(evidence, 0700); err != nil {
		t.Fatal(err)
	}
	packageName, version, cwd := "", "go1.27.0", "/workspace/tests"
	selections := map[string]string{}
	tags := "disable_grpc_modules,test_dep,sqlite3_dotlk"
	switch target {
	case "sqlite":
		packageName, cwd = "./common/persistence/sql/sqlplugin/sqlite", "/workspace/common/persistence/sql/sqlplugin/sqlite"
		selections["sqlite"] = "."
	case "persistence":
		packageName, cwd = "./common/persistence/tests", "/workspace/common/persistence/tests"
		selections["cancellation"] = "^TestSQLiteTransactionContextCancellation$"
		selections["visibility"] = "^TestSQLiteVisibilitySuite$"
		selections["visibility-persistence"] = "^TestSQLiteVisibilityPersistenceSuite$"
	case "frontend":
		packageName, cwd = "./tests/gomadfunctional", "/workspace/tests/gomadfunctional"
		selections["frontend"] = "^TestFrontendSystemInfo$"
	case "namespace-diagnostic":
		packageName, cwd = "./.tmp/wasm-backend-temporal/frontend-namespace", "/workspace/namespace-diagnostic"
		selections["namespace-diagnostic"] = "^TestNamespaceRPCCause$"
	case "namespace-stack-diagnostic":
		if adapterMode != "cooperative-v1" {
			t.Fatal("namespace stack diagnostic requires the explicitly admitted cooperative SQLite adapter")
		}
		packageName, cwd = "./.tmp/wasm-backend-temporal/frontend-namespace-stack", "/workspace/namespace-diagnostic"
		selections["namespace-stack-diagnostic"] = "^TestNamespacePendingStack$"
	case "grpc-progress-diagnostic":
		packageName, cwd = "./.tmp/wasm-backend-temporal/grpc-progress", "/workspace/grpc-progress-diagnostic"
		selections["grpc-progress-diagnostic"] = "^TestGRPCUnaryProgress$"
	case "contention":
		packageName, cwd = "./.tmp/wasm-backend-temporal/sqlite-contention", "/workspace/contention"
		selections["contention"] = "^TestBusyTimeoutAllowsQueuedLockHolder$"
	case "contention-controls":
		packageName, cwd = "./.tmp/wasm-backend-temporal/sqlite-contention", "/workspace/contention"
		selections["default-timeout"] = "^TestUnreleasedLockPreservesDefaultTimeout$"
		selections["queued-cancellation"] = "^TestQueuedCancellationInterruptsBusyRetry$"
		selections["zero-timeout"] = "^TestZeroBusyTimeoutPreserved$"
		selections["custom-timeout"] = "^TestCustomBusyTimeoutPreserved$"
	case "functional":
		packageName = "./tests"
		selections["workflow"] = "^TestWorkflowTypeEncodingSuite$/^TestPlainASCII$/^Succeeds$"
		selections["workflow-activity-cleanup"] = "^TestActivityApiPause_AttributesToActivityInContextMetadata$"
		selections["activity"] = "^TestActivityTestSuite$"
		selections["child"] = "^TestChildWorkflowSuite$"
		selections["update"] = "^TestWorkflowUpdateSuite$"
		selections["timers"] = "^TestUserTimersTestSuite$"
		selections["application-failure"] = "^TestActivityClientTestSuite$/^TestActivity_AttemptsExceeded$"
	case "failure":
		root = filepath.Join(root, "tools", "gomad", "internal", "backends", "native", "conformance", "testdata")
		packageName, version, cwd = "./io_failure", "go1.27.1", "/workspace/failure"
		tags = "test_dep,gomad_fixture"
		selections["failure"] = "^TestDeterministicIOFailure$"
	default:
		t.Fatalf("unknown Temporal gate %q", target)
	}
	if strings.HasPrefix(target, "contention") || target == "namespace-diagnostic" || target == "namespace-stack-diagnostic" || target == "grpc-progress-diagnostic" {
		owner, staged := "sqlite_contention", "sqlite-contention"
		if target == "namespace-diagnostic" {
			owner, staged = "frontend_namespace", "frontend-namespace"
		}
		if target == "namespace-stack-diagnostic" {
			owner, staged = "frontend_namespace", "frontend-namespace-stack"
		}
		if target == "grpc-progress-diagnostic" {
			owner, staged = "environment", "grpc-progress"
		}
		if err := temporalStageFixture(root, filepath.Join("tools", "gomad", "internal", "backends", "wasm", "testdata", owner), root, filepath.Join(".tmp", "wasm-backend-temporal", staged)); err != nil {
			t.Fatal(err)
		}
	}
	compiler := os.Getenv("GOMAD_WASM_TEMPORAL_GO")
	if !filepath.IsAbs(compiler) {
		t.Fatal("GOMAD_WASM_TEMPORAL_GO must name the absolute stock compiler")
	}
	buildEnv := temporalBuildEnvironment(compiler)
	compilerResult := temporalCommand(t, evidence, target+"-compiler", root, buildEnv, []string{compiler, "version"})
	if !strings.Contains(string(compilerResult.Stdout.RawBytes), version+" ") {
		t.Fatalf("wrong stock compiler: %s", compilerResult.Stdout.RawBytes)
	}
	listCommand := []string{compiler, "list", "-mod=readonly", "-tags", tags, "-deps", "-test", "-json", packageName}
	var projection *temporalSQLiteProjection
	var overlayArgs []string
	if adapterMode != "" {
		discovery := temporalCommand(t, evidence, target+"-discovery", root, buildEnv, listCommand)
		projection = temporalSQLiteAdapter(t, evidence, root, discovery.Stdout.RawBytes)
		originalGraph := temporalCommand(t, evidence, target+"-original-modules", root, buildEnv, []string{compiler, "list", "-mod=readonly", "-m", "-json", "all"})
		effectiveGraph := temporalCommand(t, evidence, target+"-projected-modules", root, buildEnv, []string{compiler, "list", "-modfile", projection.ModFile, "-mod=readonly", "-m", "-json", "all"})
		if err := temporalCompareModuleGraphs(originalGraph.Stdout.RawBytes, effectiveGraph.Stdout.RawBytes, filepath.Dir(projection.Original), projection.PrivateRoot, projection.ModFile); err != nil {
			t.Fatal(err)
		}
		overlayArgs = []string{"-modfile", projection.ModFile}
		if !projectionControl {
			overlayArgs = append(overlayArgs, "-overlay", projection.Overlay)
		}
		listCommand = append([]string{compiler, "list"}, append(overlayArgs, listCommand[2:]...)...)
	}
	listed := temporalCommand(t, evidence, target+"-inputs", root, buildEnv, listCommand)
	inputs := temporalSourceInputs(t, listed.Stdout.RawBytes)
	if projection != nil {
		inputs[projection.Original] = projection.OriginalSHA256
		inputs[projection.Replacement] = projection.ReplacementSHA256
		inputs[projection.Overlay] = projection.OverlaySHA256
		inputs[projection.ModFile] = fileHash(t, projection.ModFile)
		inputs[projection.SumFile] = fileHash(t, projection.SumFile)
		for _, name := range []string{target + "-original-modules.stdout", target + "-projected-modules.stdout"} {
			inputs[filepath.Join(evidence, name)] = fileHash(t, filepath.Join(evidence, name))
		}
		inputs[filepath.Join(projection.PrivateRoot, "conn.go")] = projection.OriginalSHA256
	}
	for _, name := range []string{"go.mod", "go.sum"} {
		file := filepath.Join(root, name)
		if _, err := os.Stat(file); err == nil {
			inputs[file] = fileHash(t, file)
		} else if !os.IsNotExist(err) {
			t.Fatal(err)
		}
	}
	t.Cleanup(func() { temporalVerifyInputs(t, inputs) })
	module := filepath.Join(evidence, target+".test.wasm")
	command := []string{compiler, "test", "-mod=readonly", "-tags", tags, "-c", "-o", module, packageName}
	if projection != nil {
		command = append([]string{compiler, "test"}, append(overlayArgs, command[2:]...)...)
	}
	temporalCommand(t, evidence, target+"-build", root, buildEnv, command)
	temporalVerifyInputs(t, inputs)
	data, err := os.ReadFile(module)
	if err != nil {
		t.Fatal(err)
	}
	imports, pages := binaryInventory(t, data)
	helper := helperPath(t)
	provenance := struct {
		Compiler, CompilerSHA256, ModuleSHA256, HelperSHA256 string
		BuildCommand, BuildEnvironment                       []string
		Inputs                                               map[string]string
		Imports                                              []Import
		InitialMemoryPages                                   uint64
		Engine                                               EngineIdentity
		SQLiteAdapter                                        *temporalSQLiteProjection
		DiagnosticOnly                                       bool
		ClockSelection, ClockClassification                  string
		Clock                                                ClockPolicy
		ConfigIdentities                                     map[string]string
	}{string(compilerResult.Stdout.RawBytes), fileHash(t, compiler), fileHash(t, module), fileHash(t, helper), command, buildEnv, inputs, imports, pages, PinnedEngine(), projection, strings.HasPrefix(target, "contention") || target == "namespace-diagnostic" || target == "namespace-stack-diagnostic" || target == "grpc-progress-diagnostic", clockSelection, "exploratory; no strict-time or production clock-policy qualification", clock, map[string]string{}}
	temporalJSON(t, filepath.Join(evidence, target+"-provenance.json"), provenance)
	config := Config{
		Profile: StockProfile, WorkingDirectory: cwd,
		Environment:         []string{"PWD=" + cwd, "TEMPORAL_ROOT=/workspace", "TMPDIR=/tmp", "TZ=UTC"},
		WritableDirectories: []string{"/workspace", "/tmp", cwd},
		EntropyKey:          sha256.Sum256([]byte("gomad-wasm-temporal-admission/v1")),
		Clock:               clock,
		Limits:              Limits{OutputBytes: 8 << 20, TranscriptBytes: 128 << 20, Calls: 250000, Descriptors: 1024, Files: 10000, FilesystemBytes: 128 << 20, PendingEvents: 4096},
	}
	if target != "failure" {
		captured, err := captured.CaptureReadOnlyMountInputs([]captured.Mapping{{Source: filepath.Join(root, "schema", "sqlite"), Target: "/workspace/schema/sqlite"}}, captured.DefaultLimits())
		if err != nil {
			t.Fatal(err)
		}
		config.CapturedInputs = captured
		temporalJSON(t, filepath.Join(evidence, target+"-captured.json"), captured)
	}
	names := make([]string, 0, len(selections))
	for name := range selections {
		names = append(names, name)
	}
	slices.Sort(names)
	if target == "functional" {
		names = []string{"workflow", "workflow-activity-cleanup", "activity", "child", "update", "timers", "application-failure"}
	}
	for _, name := range names {
		t.Run(name, func(t *testing.T) {
			config.Args = []string{"temporal.test", "-test.v", "-test.count=1", "-test.timeout=20m", "-test.run=" + selections[name]}
			if target == "frontend" || target == "functional" || target == "namespace-diagnostic" || target == "namespace-stack-diagnostic" {
				config.Args = append(config.Args, "-persistenceType=sql", "-persistenceDriver=sqlite")
			}
			environment, err := NewEnvironment(config)
			if err != nil {
				t.Fatal(err)
			}
			provenance.ConfigIdentities[name] = environment.Identity()
			temporalJSON(t, filepath.Join(evidence, target+"-provenance.json"), provenance)
			request := Request{HelperPath: helper, HelperSHA256: provenance.HelperSHA256, ModulePath: module, ModuleSHA256: provenance.ModuleSHA256, Engine: provenance.Engine, Imports: imports, InitialMemoryPages: pages, MemoryBytes: 2 << 30, Fuel: 2000000000000, Config: config, Timeout: 15 * time.Minute}
			temporalJSON(t, filepath.Join(evidence, name+"-request.json"), request)
			var first Result
			repeat := 1
			if target == "failure" {
				repeat = 2
			}
			for iteration := range repeat {
				started := time.Now()
				result, err := Run(t.Context(), request)
				prefix := filepath.Join(evidence, fmt.Sprintf("%s-%d", name, iteration))
				temporalWrite(t, prefix+".stdout", result.Stdout)
				temporalWrite(t, prefix+".stderr", result.Stderr)
				temporalWrite(t, prefix+".transcript.bin", result.Transcript)
				metadata := result
				metadata.Stdout, metadata.Stderr, metadata.Transcript = nil, nil, nil
				temporalJSON(t, prefix+".json", struct {
					Result                                       Result
					ElapsedNanos                                 int64
					StdoutSHA256, StderrSHA256, TranscriptSHA256 string
					Error                                        string
				}{metadata, time.Since(started).Nanoseconds(), temporalHash(result.Stdout), temporalHash(result.Stderr), temporalHash(result.Transcript), fmt.Sprint(err)})
				if err != nil {
					t.Fatal(err)
				}
				if result.EnvironmentIdentity != provenance.ConfigIdentities[name] {
					t.Fatal("guest configuration identity differs from admitted provenance")
				}
				wantExit := uint32(0)
				if target == "failure" || projectionControl {
					wantExit = 1
				}
				if (target == "namespace-diagnostic" || target == "namespace-stack-diagnostic") && result.ExitCode != nil && *result.ExitCode <= 1 {
					wantExit = *result.ExitCode
				}
				if result.Termination != "exit" || result.ExitCode == nil || *result.ExitCode != wantExit || !result.Reaped {
					t.Fatalf("guest %s: termination=%s exit=%v reaped=%v message=%s; complete output retained in %s", name, result.Termination, result.ExitCode, result.Reaped, result.Message, prefix)
				}
				if err := temporalCheckOutput(result.Stdout, selections[name], target == "failure" || projectionControl || (target == "namespace-diagnostic" || target == "namespace-stack-diagnostic") && wantExit == 1); err != nil {
					t.Fatal(err)
				}
				if target == "failure" && !bytes.Contains(result.Stdout, []byte(`deterministic failure after reading "pending"`)) {
					t.Fatal("deliberate application failure was not reached")
				}
				if iteration > 0 && (!bytes.Equal(first.Stdout, result.Stdout) || !bytes.Equal(first.Stderr, result.Stderr) || !bytes.Equal(first.Transcript, result.Transcript)) {
					t.Fatal("fresh deliberate failure output or transcript diverged")
				}
				first = result
				t.Logf("target=%s guest=%s iteration=%d exit=%d environment=%s module=%s transcript=%s memory=%d fuel-remaining=%d", target, name, iteration, wantExit, result.EnvironmentIdentity, request.ModuleSHA256, temporalHash(result.Transcript), result.PeakMemoryBytes, result.FuelRemaining)
			}
		})
	}
}

func temporalClockPolicy(selection string) (ClockPolicy, error) {
	step := uint64(1000000)
	switch selection {
	case "":
	case "read-step-1us-v1":
		step = 1000
	default:
		return ClockPolicy{}, fmt.Errorf("unknown Temporal exploratory clock policy %q", selection)
	}
	return ClockPolicy{EpochNanos: 946684800000000000, ReadStepNanos: step}, nil
}

func temporalVerifyInputs(t *testing.T, inputs map[string]string) {
	t.Helper()
	for name, expected := range inputs {
		if actual := fileHash(t, name); actual != expected {
			t.Fatalf("source input changed during admission: %s: expected %s, got %s", name, expected, actual)
		}
	}
}

func TestTemporalClockPolicy(t *testing.T) {
	for _, test := range []struct {
		selection string
		step      uint64
	}{
		{"", 1000000},
		{"read-step-1us-v1", 1000},
		{"read-step-1ms-v1", 0},
		{"1000", 0},
		{"read-step-1us-v2", 0},
		{"read-step-1us-v1 ", 0},
		{"\xff", 0},
	} {
		t.Run(test.selection, func(t *testing.T) {
			clock, err := temporalClockPolicy(test.selection)
			if test.step == 0 {
				if err == nil || clock != (ClockPolicy{}) {
					t.Fatal("accepted an arbitrary clock policy")
				}
				return
			}
			if err != nil || clock != (ClockPolicy{EpochNanos: 946684800000000000, ReadStepNanos: test.step}) {
				t.Fatalf("clock policy = %+v, error = %v", clock, err)
			}
		})
	}
}

func TestTemporalClockConfigurationIdentity(t *testing.T) {
	var identities []string
	for _, selection := range []string{"", "read-step-1us-v1"} {
		clock, err := temporalClockPolicy(selection)
		if err != nil {
			t.Fatal(err)
		}
		config := testConfig()
		config.Clock = clock
		environment, err := NewEnvironment(config)
		if err != nil {
			t.Fatal(err)
		}
		identities = append(identities, environment.Identity())
	}
	if identities[0] == identities[1] {
		t.Fatal("clock selection was omitted from the configuration identity")
	}
}

func TestTemporalGuestOutput(t *testing.T) {
	for _, test := range []struct {
		name, selection, output string
		failure, valid          bool
	}{
		{"empty", "^TestFrontendSystemInfo$", "", false, false},
		{"zero-match", "^TestFrontendSystemInfo$", "testing: warning: no tests to run\nPASS\n", false, false},
		{"skip", "^TestFrontendSystemInfo$", "--- SKIP: TestFrontendSystemInfo (0.00s)\nPASS\n", false, false},
		{"different", "^TestFrontendSystemInfo$", "--- PASS: TestFrontendSystemInfoOther (0.00s)\nPASS\n", false, false},
		{"pass", "^TestFrontendSystemInfo$", "--- PASS: TestFrontendSystemInfo (0.00s)\nPASS\n", false, true},
		{"grpc-zero-match", "^TestGRPCUnaryProgress$", "testing: warning: no tests to run\nPASS\n", false, false},
		{"grpc-skip", "^TestGRPCUnaryProgress$", "--- SKIP: TestGRPCUnaryProgress (0.00s)\nPASS\n", false, false},
		{"grpc-pass", "^TestGRPCUnaryProgress$", "--- PASS: TestGRPCUnaryProgress (0.00s)\nPASS\n", false, true},
		{"stack-zero-match", "^TestNamespacePendingStack$", "testing: warning: no tests to run\nPASS\n", false, false},
		{"stack-skip", "^TestNamespacePendingStack$", "--- SKIP: TestNamespacePendingStack (0.00s)\nPASS\n", false, false},
		{"stack-pass", "^TestNamespacePendingStack$", "--- PASS: TestNamespacePendingStack (0.00s)\nPASS\n", false, true},
		{"all-suite-skipped", "^TestActivityTestSuite$", "--- PASS: TestActivityTestSuite (0.00s)\n    --- SKIP: TestActivityTestSuite/TestActivity (0.00s)\nPASS\n", false, false},
		{"suite", "^TestActivityTestSuite$", "--- PASS: TestActivityTestSuite (0.00s)\n    --- PASS: TestActivityTestSuite/TestActivity (0.00s)\nPASS\n", false, true},
		{"failure", "^TestDeterministicIOFailure$", "--- FAIL: TestDeterministicIOFailure (0.00s)\nFAIL\n", true, true},
		{"wrong-failure", "^TestDeterministicIOFailure$", "--- FAIL: TestOther (0.00s)\nFAIL\n", true, false},
		{"nested", "^TestWorkflowTypeEncodingSuite$/^TestPlainASCII$/^Succeeds$", "    --- PASS: TestWorkflowTypeEncodingSuite/TestPlainASCII/Succeeds (0.00s)\nPASS\n", false, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			if err := temporalCheckOutput([]byte(test.output), test.selection, test.failure); (err == nil) != test.valid {
				t.Fatalf("output validity: expected %v, got %v", test.valid, err)
			}
		})
	}
}

func temporalCheckOutput(output []byte, selection string, failure bool) error {
	names := []string{strings.NewReplacer("^", "", "$", "").Replace(selection)}
	if selection == "." {
		names = []string{"TestDriverErrors", "TestDriverTimestampCompatibility", "TestDriverMemorySchema", "TestDriverTimestampPrecision", "TestQueryConverter_GetCoalesceCloseTimeExpr", "TestQueryConverter_ConvertKeywordListComparisonExpr", "TestQueryConverter_ConvertTextComparisonExpr", "TestQueryConverter_BuildSelectStmt", "TestQueryConverter_BuildCountStmt", "TestRewriteSchemaStatements_EnforcesVarcharLengthChecks"}
	}
	status := "PASS"
	if failure {
		status = "FAIL"
	}
	for _, name := range names {
		found, child := false, false
		for line := range strings.SplitSeq(string(output), "\n") {
			line = strings.TrimSpace(line)
			found = found || strings.HasPrefix(line, "--- "+status+": "+name+" (")
			child = child || strings.HasPrefix(line, "--- "+status+": "+name+"/")
		}
		if !found || (strings.HasSuffix(name, "Suite") && !child) {
			return fmt.Errorf("guest did not execute named %s outcome for %s", status, name)
		}
	}
	return nil
}

func temporalBuildEnvironment(compiler string) []string {
	environment := []string{}
	for _, name := range []string{"HOME", "TMPDIR", "GOTMPDIR", "GOCACHE", "GOPATH", "GOMODCACHE"} {
		if value, found := os.LookupEnv(name); found {
			environment = append(environment, name+"="+value)
		}
	}
	return append(environment, "PATH="+filepath.Dir(compiler)+":"+os.Getenv("PATH"), "GOOS=wasip1", "GOARCH=wasm", "CGO_ENABLED=0", "GOWORK=off", "GOENV=off", "GOFLAGS=", "GOEXPERIMENT=nogreenteagc", "GOTOOLCHAIN=local", "GOMAXPROCS=1")
}

type temporalSQLiteProjection struct {
	Policy, Module, Version                                  string
	Original, OriginalSHA256, Replacement, ReplacementSHA256 string
	Overlay, OverlaySHA256                                   string
	PrivateRoot, ModFile, SumFile                            string
	OriginalInventory, PrivateInventory                      captured.CapturedInputsManifest
}

func temporalSQLiteAdapter(t *testing.T, evidence, root string, listed []byte) *temporalSQLiteProjection {
	t.Helper()
	decoder := json.NewDecoder(bytes.NewReader(listed))
	var projection *temporalSQLiteProjection
	for {
		var pkg struct {
			ImportPath, Dir string
			GoFiles         []string
			Module          *struct {
				Path, Version, GoMod string
				Replace              json.RawMessage
			}
		}
		if err := decoder.Decode(&pkg); err == io.EOF {
			break
		} else if err != nil {
			t.Fatal(err)
		}
		if pkg.ImportPath != "github.com/ncruces/go-sqlite3" {
			continue
		}
		if projection != nil || pkg.Module == nil || !filepath.IsAbs(pkg.Dir) || !slices.Contains(pkg.GoFiles, "conn.go") {
			t.Fatal("SQLite adapter requires one selected pinned conn.go source and its module identity")
		}
		original := filepath.Join(pkg.Dir, "conn.go")
		source, err := os.ReadFile(original)
		if err != nil {
			t.Fatal(err)
		}
		dependency := sqlitebusy.Dependency{Path: pkg.Module.Path, Version: pkg.Module.Version, Replaced: len(pkg.Module.Replace) > 0 && string(pkg.Module.Replace) != "null"}
		adapter, err := sqlitebusy.Prepare(dependency, source)
		if err != nil {
			t.Fatal(err)
		}
		cachedMod, err := os.ReadFile(pkg.Module.GoMod)
		if err != nil {
			t.Fatal(err)
		}
		moduleMod, err := os.ReadFile(filepath.Join(pkg.Dir, "go.mod"))
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(cachedMod, moduleMod) {
			t.Fatal("cached SQLite module metadata differs from captured module go.mod")
		}
		t.Cleanup(func() {
			if fileHash(t, pkg.Module.GoMod) != temporalHash(cachedMod) {
				t.Fatal("original SQLite cached module metadata changed")
			}
		})
		captured := temporalCaptureModule(t, pkg.Dir)
		privateParent := filepath.Join(evidence, "private-modules")
		if err := os.MkdirAll(privateParent, 0700); err != nil {
			t.Fatal(err)
		}
		privateRoot := filepath.Join(privateParent, strings.TrimPrefix(string(captured.Manifest.SHA256), "sha256:"))
		if _, err := os.Stat(privateRoot); os.IsNotExist(err) {
			if err := temporalMaterializeModule(privateRoot, captured); err != nil {
				t.Fatal(err)
			}
		} else if err != nil {
			t.Fatal(err)
		}
		privateCaptured := temporalCaptureModule(t, privateRoot)
		if !bytes.Equal(captured.Descriptor, privateCaptured.Descriptor) {
			t.Fatal("private SQLite module does not match original complete captured inventory")
		}
		verify := func() {
			for _, location := range []string{pkg.Dir, privateRoot} {
				if current := temporalCaptureModule(t, location); !bytes.Equal(captured.Descriptor, current.Descriptor) {
					t.Fatalf("SQLite module inventory changed during admission: %s", location)
				}
			}
		}
		t.Cleanup(verify)
		for _, name := range []string{"go.mod", "go.sum"} {
			contents, err := os.ReadFile(filepath.Join(root, name))
			if err != nil {
				t.Fatal(err)
			}
			if name == "go.mod" {
				contents = append(contents, []byte("\nreplace github.com/ncruces/go-sqlite3 v0.35.6 => "+strconv.Quote(privateRoot)+"\n")...)
			}
			output := "temporal.mod"
			if name == "go.sum" {
				output = "temporal.sum"
			}
			temporalWrite(t, filepath.Join(evidence, output), contents)
		}
		temporalJSON(t, filepath.Join(evidence, "sqlite-original-capture.json"), captured)
		if fileHash(t, filepath.Join(privateRoot, "conn.go")) != adapter.OriginalSHA256 {
			t.Fatal("private SQLite source differs from admitted source")
		}
		replacement := filepath.Join(evidence, "sqlite-conn-cooperative.go")
		temporalWrite(t, replacement, adapter.Source)
		overlay := filepath.Join(evidence, "sqlite-overlay.json")
		temporalJSON(t, overlay, struct{ Replace map[string]string }{map[string]string{filepath.Join(privateRoot, "conn.go"): replacement}})
		projection = &temporalSQLiteProjection{Policy: adapter.Policy, Module: dependency.Path, Version: dependency.Version, Original: original, OriginalSHA256: adapter.OriginalSHA256, Replacement: replacement, ReplacementSHA256: adapter.ReplacementSHA256, Overlay: overlay, OverlaySHA256: fileHash(t, overlay), PrivateRoot: privateRoot, ModFile: filepath.Join(evidence, "temporal.mod"), SumFile: filepath.Join(evidence, "temporal.sum"), OriginalInventory: captured.Manifest, PrivateInventory: privateCaptured.Manifest}
	}
	if projection == nil {
		t.Fatal("SQLite adapter selected without the pinned SQLite dependency")
	}
	return projection
}

func temporalCaptureModule(t *testing.T, source string) captured.CapturedInputs {
	t.Helper()
	captured, err := captured.CaptureReadOnlyMountInputs([]captured.Mapping{{Source: source, Target: "/module"}}, captured.DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	return captured
}

func temporalCommand(t *testing.T, evidence, name, root string, environment, command []string) hostexec.Result {
	t.Helper()
	started := time.Now()
	result, err := hostexec.Run(context.Background(), hostexec.Request{Command: command, Dir: root, Env: environment, Timeout: 15 * time.Minute, TerminateGrace: time.Second, OutputLimit: 64 << 20, CollectResourceUsage: true})
	temporalWrite(t, filepath.Join(evidence, name+".stdout"), result.Stdout.RawBytes)
	temporalWrite(t, filepath.Join(evidence, name+".stderr"), result.Stderr.RawBytes)
	temporalJSON(t, filepath.Join(evidence, name+".json"), struct {
		Command       []string
		ExitCode      int
		ElapsedNanos  int64
		ResourceUsage *hostexec.ResourceUsage
	}{command, result.ExitCode, time.Since(started).Nanoseconds(), result.ResourceUsage})
	if err != nil || result.Termination != hostexec.TerminationExit || result.ExitCode != 0 || !result.GroupGone {
		t.Fatalf("infrastructure command %v: %v, termination=%s exit=%d\n%s\n%s", command, err, result.Termination, result.ExitCode, result.Stdout.RawBytes, result.Stderr.RawBytes)
	}
	return result
}

func temporalSourceInputs(t *testing.T, data []byte) map[string]string {
	t.Helper()
	inputs := map[string]string{}
	decoder := json.NewDecoder(bytes.NewReader(data))
	for {
		var pkg struct {
			Dir                                                              string
			GoFiles, CgoFiles, SFiles, EmbedFiles, TestGoFiles, XTestGoFiles []string
		}
		if err := decoder.Decode(&pkg); err == io.EOF {
			break
		} else if err != nil {
			t.Fatal(err)
		}
		for _, names := range [][]string{pkg.GoFiles, pkg.CgoFiles, pkg.SFiles, pkg.EmbedFiles, pkg.TestGoFiles, pkg.XTestGoFiles} {
			for _, name := range names {
				if !filepath.IsAbs(name) {
					name = filepath.Join(pkg.Dir, name)
				}
				inputs[name] = fileHash(t, name)
			}
		}
	}
	return inputs
}

func temporalHash(data []byte) string {
	digest := sha256.Sum256(data)
	return hex.EncodeToString(digest[:])
}

func temporalJSON(t *testing.T, name string, value any) {
	t.Helper()
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	temporalWrite(t, name, append(data, '\n'))
}

func temporalWrite(t *testing.T, name string, data []byte) {
	t.Helper()
	if err := os.WriteFile(name, data, 0600); err != nil {
		t.Fatal(err)
	}
}

func TestTemporalPrivateModule(t *testing.T) {
	source := t.TempDir()
	if err := os.Mkdir(filepath.Join(source, "sub"), 0750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(source, "sub", "source.go"), []byte("package fixture\n"), 0440); err != nil {
		t.Fatal(err)
	}
	inputs, err := captured.CaptureReadOnlyMountInputs([]captured.Mapping{{Source: source, Target: "/module"}}, captured.DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	destination := filepath.Join(t.TempDir(), "module")
	if err := temporalMaterializeModule(destination, inputs); err != nil {
		t.Fatal(err)
	}
	copy, err := captured.CaptureReadOnlyMountInputs([]captured.Mapping{{Source: destination, Target: "/module"}}, captured.DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(inputs.Descriptor, copy.Descriptor) {
		t.Fatal("private module changed complete bytes or modes")
	}
	if err := temporalMaterializeModule(destination, inputs); err == nil {
		t.Fatal("materializer must reject an existing root")
	}
	inputs.Descriptor = append(inputs.Descriptor, '\n')
	if err := temporalMaterializeModule(filepath.Join(t.TempDir(), "module"), inputs); err == nil {
		t.Fatal("materializer accepted unvalidated snapshot")
	}
}

func TestTemporalFixtureStaging(t *testing.T) {
	source := filepath.Join("..", "testdata", "sqlite_contention")
	original, err := os.ReadFile(filepath.Join(source, "contention_test.go"))
	if err != nil {
		t.Fatal(err)
	}
	if temporalHash(original) != "a9bfaf87df31ad4eb18abfccb4920ae0911c77851f34be02ab7f2ea4683a52ae" {
		t.Fatal("original exclusive fixture bytes changed")
	}
	destinationAnchor := t.TempDir()
	destination := filepath.Join(destinationAnchor, "parent", "fixture")
	if err := temporalStageFixture(filepath.Dir(source), filepath.Base(source), destinationAnchor, filepath.Join("parent", "fixture")); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"contention_test.go", "controls_test.go"} {
		expected, err := os.ReadFile(filepath.Join(source, name))
		if err != nil {
			t.Fatal(err)
		}
		actual, err := os.ReadFile(filepath.Join(destination, name))
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(expected, actual) {
			t.Fatal("staged fixture body changed")
		}
	}
	if err := temporalStageFixture(filepath.Dir(source), filepath.Base(source), destinationAnchor, filepath.Join("parent", "fixture")); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(destination, "contention_test.go")
	if err := os.WriteFile(target, append(bytes.Clone(original), '\n'), 0600); err != nil {
		t.Fatal(err)
	}
	if err := temporalStageFixture(filepath.Dir(source), filepath.Base(source), destinationAnchor, filepath.Join("parent", "fixture")); err == nil {
		t.Fatal("staging accepted changed fixture hash")
	}
	data, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(original, data) {
		t.Fatal("staging silently replaced changed source")
	}
	if err := os.WriteFile(target, original, 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(destination, "extra_test.go"), []byte("package contention\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := temporalStageFixture(filepath.Dir(source), filepath.Base(source), destinationAnchor, filepath.Join("parent", "fixture")); err == nil {
		t.Fatal("staging accepted an extra test body")
	}
}

func TestTemporalGRPCDiagnosticFixtureStaging(t *testing.T) {
	source := filepath.Join("..", "testdata", "environment")
	destination := t.TempDir()
	if err := temporalStageFixture(filepath.Dir(source), filepath.Base(source), destination, "grpc-progress"); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"main.go", "grpc_progress_test.go"} {
		expected, err := os.ReadFile(filepath.Join(source, name))
		if err != nil {
			t.Fatal(err)
		}
		if name == "main.go" && temporalHash(expected) != "fde93d6955cafae1d1696d405eaa46987534c05cfef5dc2fe2a14483bacc98f3" {
			t.Fatal("original environment fixture bytes changed")
		}
		actual, err := os.ReadFile(filepath.Join(destination, "grpc-progress", name))
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(expected, actual) {
			t.Fatalf("staged gRPC diagnostic changed %s", name)
		}
	}
}

func TestTemporalNamespaceStackFixtureStaging(t *testing.T) {
	source := filepath.Join("..", "testdata", "frontend_namespace")
	destination := t.TempDir()
	if err := temporalStageFixture(filepath.Dir(source), filepath.Base(source), destination, "namespace-stack"); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"namespace_test.go", "namespace_stack_test.go"} {
		expected, err := os.ReadFile(filepath.Join(source, name))
		if err != nil {
			t.Fatal(err)
		}
		actual, err := os.ReadFile(filepath.Join(destination, "namespace-stack", name))
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(expected, actual) {
			t.Fatalf("staged namespace stack diagnostic changed %s", name)
		}
	}
}

func TestTemporalFixtureStagingRejectsLinks(t *testing.T) {
	for _, name := range []string{"source-root", "source-parent", "destination-root", "destination-parent"} {
		t.Run(name, func(t *testing.T) {
			anchor, outside := t.TempDir(), t.TempDir()
			source, destination := filepath.Join(anchor, "source"), filepath.Join(anchor, "destination")
			owner := filepath.Join("..", "testdata", "sqlite_contention")
			if err := temporalStageFixture(filepath.Dir(owner), filepath.Base(owner), anchor, "source"); err != nil {
				t.Fatal(err)
			}
			outsideFixture := filepath.Join(outside, "fixture")
			if err := temporalStageFixture(filepath.Dir(owner), filepath.Base(owner), outside, "fixture"); err != nil {
				t.Fatal(err)
			}
			before, err := captured.CaptureReadOnlyMountInputs([]captured.Mapping{{Source: outside, Target: "/outside"}}, captured.DefaultLimits())
			if err != nil {
				t.Fatal(err)
			}
			switch name {
			case "source-root":
				source = filepath.Join(anchor, "source-link")
				if err := os.Symlink(outsideFixture, source); err != nil {
					t.Fatal(err)
				}
			case "source-parent":
				parent := filepath.Join(anchor, "source-parent")
				if err := os.Symlink(outside, parent); err != nil {
					t.Fatal(err)
				}
				source = filepath.Join(parent, "fixture")
			case "destination-root":
				if err := os.Symlink(outsideFixture, destination); err != nil {
					t.Fatal(err)
				}
			case "destination-parent":
				parent := filepath.Join(anchor, "destination-parent")
				if err := os.Symlink(outside, parent); err != nil {
					t.Fatal(err)
				}
				destination = filepath.Join(parent, "new-fixture")
			}
			sourceRelative, err := filepath.Rel(anchor, source)
			if err != nil {
				t.Fatal(err)
			}
			destinationRelative, err := filepath.Rel(anchor, destination)
			if err != nil {
				t.Fatal(err)
			}
			stageErr := temporalStageFixture(anchor, sourceRelative, anchor, destinationRelative)
			after, err := captured.CaptureReadOnlyMountInputs([]captured.Mapping{{Source: outside, Target: "/outside"}}, captured.DefaultLimits())
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(before.Descriptor, after.Descriptor) {
				t.Error("staging changed bytes or entries outside the trusted anchor")
			}
			if stageErr == nil {
				t.Error("staging accepted a linked fixture path")
			}
		})
	}
}

func TestTemporalFixtureStagingPolicy(t *testing.T) {
	owner := filepath.Join("..", "testdata")
	for _, relative := range []string{"../escape", "/absolute", "."} {
		t.Run(relative, func(t *testing.T) {
			anchor := t.TempDir()
			if err := temporalStageFixture(owner, "sqlite_contention", anchor, relative); err == nil {
				t.Fatal("accepted invalid destination")
			}
			if err := temporalStageFixture(owner, relative, anchor, "fixture"); err == nil {
				t.Fatal("accepted invalid source")
			}
			entries, err := os.ReadDir(anchor)
			if err != nil {
				t.Fatal(err)
			}
			if len(entries) != 0 {
				t.Fatal("invalid path admission changed the destination anchor")
			}
		})
	}
	anchor := t.TempDir()
	if err := os.MkdirAll(filepath.Join(anchor, "source", "nested"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := temporalStageFixture(anchor, "source", anchor, "destination"); err == nil {
		t.Fatal("accepted a nonflat fixture owner")
	}
	if _, err := os.Lstat(filepath.Join(anchor, "destination")); !os.IsNotExist(err) {
		t.Fatal("rejected source created a destination")
	}
}

func TestTemporalModuleGraph(t *testing.T) {
	original := []byte(`{"Path":"main","Main":true,"GoMod":"/root/go.mod"}
{"Path":"github.com/ncruces/go-sqlite3","Version":"v0.35.6","Dir":"/cache/sqlite","GoMod":"/cache/sqlite/go.mod","Sum":"pinned","GoModSum":"pinned-mod"}
{"Path":"unrelated","Version":"v1.2.3","Sum":"unchanged"}`)
	effective := []byte(`{"Path":"main","Main":true,"GoMod":"/stage/temporal.mod"}
{"Path":"github.com/ncruces/go-sqlite3","Version":"v0.35.6","Dir":"/private/sqlite","GoMod":"/private/sqlite/go.mod","Replace":{"Path":"/private/sqlite","Dir":"/private/sqlite","GoMod":"/private/sqlite/go.mod"}}
{"Path":"unrelated","Version":"v1.2.3","Sum":"unchanged"}`)
	for _, test := range []struct {
		name                string
		original, effective []byte
		valid               bool
	}{
		{"owned", original, effective, true},
		{"extra-module", original, append(bytes.Clone(effective), []byte(`{"Path":"extra","Version":"v1"}`)...), false},
		{"version", original, bytes.Replace(effective, []byte("v1.2.3"), []byte("v1.2.4"), 1), false},
		{"preexisting", bytes.Replace(original, []byte(`"Sum":"pinned"`), []byte(`"Sum":"pinned","Replace":{"Path":"/user"}`), 1), effective, false},
		{"wrong-copy", original, bytes.ReplaceAll(effective, []byte("/private/sqlite"), []byte("/elsewhere")), false},
	} {
		t.Run(test.name, func(t *testing.T) {
			if err := temporalCompareModuleGraphs(test.original, test.effective, "/cache/sqlite", "/private/sqlite", "/stage/temporal.mod"); (err == nil) != test.valid {
				t.Fatalf("expected admission %v: %v", test.valid, err)
			}
		})
	}
}

func temporalModuleSnapshot(inputs captured.CapturedInputs) (captured.Snapshot, error) {
	mappings, _, snapshot, err := captured.DecodeCapturedInputs(inputs.Manifest, inputs.Descriptor, func(name string, size uint64) ([]byte, error) {
		data, found := inputs.Payloads[name]
		if !found || uint64(len(data)) != size {
			return nil, fmt.Errorf("missing private module payload %s", name)
		}
		return data, nil
	})
	if err != nil {
		return captured.Snapshot{}, err
	}
	if len(mappings) != 1 || mappings[0].Target != "/module" || len(snapshot.NotExist) != 0 {
		return captured.Snapshot{}, fmt.Errorf("private module requires one complete /module snapshot")
	}
	return snapshot, nil
}

func temporalMaterializeModule(destination string, captured captured.CapturedInputs) (retErr error) {
	snapshot, err := temporalModuleSnapshot(captured)
	if err != nil {
		return err
	}
	if err := os.Mkdir(destination, 0700); err != nil {
		return err
	}
	root, err := os.OpenRoot(destination)
	if err != nil {
		return err
	}
	defer func() { retErr = errors.Join(retErr, root.Close()) }()
	return temporalWriteModule(root, snapshot)
}

func temporalWriteModule(root *os.Root, snapshot captured.Snapshot) error {
	for _, entry := range snapshot.Entries {
		relative, err := filepath.Rel("/module", entry.Path)
		if err != nil || relative == ".." || strings.HasPrefix(relative, "../") || filepath.IsAbs(relative) {
			return fmt.Errorf("invalid private module entry %s", entry.Path)
		}
		if entry.Kind == captured.KindDirectory {
			if relative != "." {
				if err := root.Mkdir(relative, 0700); err != nil {
					return err
				}
			}
			continue
		}
		file, err := root.OpenFile(relative, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
		if err != nil {
			return err
		}
		_, writeErr := file.Write(entry.Data)
		if err := errors.Join(writeErr, file.Close()); err != nil {
			return err
		}
		if err := root.Chmod(relative, entry.Mode); err != nil {
			return err
		}
	}
	for _, entry := range slices.Backward(snapshot.Entries) {
		if entry.Kind == captured.KindDirectory {
			relative, err := filepath.Rel("/module", entry.Path)
			if err != nil {
				return err
			}
			if err := root.Chmod(relative, entry.Mode); err != nil {
				return err
			}
		}
	}
	return nil
}

func temporalStageFixture(sourceAnchor, source, destinationAnchor, destination string) (retErr error) {
	if !filepath.IsLocal(source) || !filepath.IsLocal(destination) || filepath.Clean(source) == "." || filepath.Clean(destination) == "." {
		return fmt.Errorf("fixture paths must remain beneath their trusted anchors")
	}
	limits := captured.DefaultLimits()
	limits.Files, limits.Requests, limits.DirectoryEntries = 16, 64, 16
	limits.SingleFileBytes, limits.TotalBytes = 64<<10, 128<<10
	capture := func(anchor, relative string) (_ captured.CapturedInputs, captureErr error) {
		mapping := captured.Mapping{Source: anchor, Target: "/module"}
		broker, err := captured.Prepare([]captured.Mapping{mapping}, limits)
		if err != nil {
			return captured.CapturedInputs{}, err
		}
		defer func() { captureErr = errors.Join(captureErr, broker.Close()) }()
		prefix := "/module/" + filepath.ToSlash(filepath.Clean(relative))
		entry, err := broker.Lookup(prefix)
		if err != nil {
			return captured.CapturedInputs{}, err
		}
		if entry.Kind != captured.KindDirectory {
			return captured.CapturedInputs{}, fmt.Errorf("fixture owner must be a directory")
		}
		for _, child := range entry.Children {
			if child.Kind != captured.KindFile {
				return captured.CapturedInputs{}, fmt.Errorf("fixture owner must contain only flat regular files")
			}
			if _, err := broker.Lookup(prefix + "/" + child.Name); err != nil {
				return captured.CapturedInputs{}, err
			}
		}
		snapshot := broker.Captured()
		for index := range snapshot.Entries {
			snapshot.Entries[index].Path = "/module" + strings.TrimPrefix(snapshot.Entries[index].Path, prefix)
		}
		return captured.EncodeCapturedInputs([]captured.Mapping{mapping}, limits, snapshot)
	}
	captured, err := capture(sourceAnchor, source)
	if err != nil {
		return err
	}
	snapshot, err := temporalModuleSnapshot(captured)
	if err != nil {
		return err
	}
	anchor, err := os.OpenRoot(destinationAnchor)
	if err != nil {
		return err
	}
	defer func() { retErr = errors.Join(retErr, anchor.Close()) }()
	parent := anchor
	components := strings.Split(filepath.Clean(destination), string(filepath.Separator))
	for _, component := range components[:len(components)-1] {
		info, err := parent.Lstat(component)
		if os.IsNotExist(err) {
			if err := parent.Mkdir(component, 0700); err != nil {
				return err
			}
			info, err = parent.Lstat(component)
		}
		if err != nil {
			return err
		}
		if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("fixture destination parent is not a regular directory")
		}
		next, err := parent.OpenRoot(component)
		if err != nil {
			return err
		}
		defer func() { retErr = errors.Join(retErr, next.Close()) }()
		parent = next
	}
	name := components[len(components)-1]
	info, err := parent.Lstat(name)
	created := os.IsNotExist(err)
	if created {
		if err := parent.Mkdir(name, 0700); err != nil {
			return err
		}
		info, err = parent.Lstat(name)
	}
	if err != nil {
		return err
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("fixture destination is not a regular directory")
	}
	if created {
		root, err := parent.OpenRoot(name)
		if err != nil {
			return err
		}
		writeErr := temporalWriteModule(root, snapshot)
		if err := errors.Join(writeErr, root.Close()); err != nil {
			return err
		}
	}
	staged, err := capture(destinationAnchor, destination)
	if err != nil {
		return err
	}
	if !bytes.Equal(captured.Descriptor, staged.Descriptor) {
		return fmt.Errorf("staged diagnostic fixture differs from its source owner: %s", destination)
	}
	return nil
}

func temporalCompareModuleGraphs(original, effective []byte, originalRoot, privateRoot, modfile string) error {
	decode := func(data []byte) (map[string]map[string]any, error) {
		modules := map[string]map[string]any{}
		decoder := json.NewDecoder(bytes.NewReader(data))
		for {
			var module map[string]any
			if err := decoder.Decode(&module); err == io.EOF {
				return modules, nil
			} else if err != nil {
				return nil, err
			}
			name, valid := module["Path"].(string)
			if !valid || modules[name] != nil {
				return nil, fmt.Errorf("invalid or duplicate module graph identity")
			}
			modules[name] = module
		}
	}
	before, err := decode(original)
	if err != nil {
		return err
	}
	after, err := decode(effective)
	if err != nil {
		return err
	}
	if len(before) != len(after) {
		return fmt.Errorf("owned projection changed module graph size")
	}
	for name, module := range before {
		projected := after[name]
		if projected == nil {
			return fmt.Errorf("owned projection removed module %s", name)
		}
		if module["Main"] == true {
			if projected["GoMod"] != modfile {
				return fmt.Errorf("owned projection has wrong main modfile")
			}
			projected["GoMod"] = module["GoMod"]
		}
		if name == "github.com/ncruces/go-sqlite3" {
			if module["Replace"] != nil || module["Version"] != "v0.35.6" || module["Dir"] != originalRoot || !filepath.IsAbs(fmt.Sprint(module["GoMod"])) {
				return fmt.Errorf("owned projection requires original unreplaced pinned SQLite graph")
			}
			replacement, valid := projected["Replace"].(map[string]any)
			expected := map[string]any{"Path": privateRoot, "Dir": privateRoot, "GoMod": filepath.Join(privateRoot, "go.mod")}
			if module["GoVersion"] != nil {
				expected["GoVersion"] = module["GoVersion"]
			}
			if !valid || !reflect.DeepEqual(replacement, expected) || projected["Dir"] != privateRoot || projected["GoMod"] != filepath.Join(privateRoot, "go.mod") {
				return fmt.Errorf("SQLite replacement is not the exact owned projection")
			}
			delete(projected, "Replace")
			for _, key := range []string{"Dir", "GoMod", "Sum", "GoModSum", "Time"} {
				if key == "Sum" || key == "GoModSum" || key == "Time" {
					if projected[key] != nil {
						return fmt.Errorf("unexpected projected SQLite checksum metadata")
					}
				}
				if module[key] != nil {
					projected[key] = module[key]
				} else {
					delete(projected, key)
				}
			}
		}
		if !reflect.DeepEqual(module, projected) {
			return fmt.Errorf("owned projection changed unrelated graph metadata for %s: original=%v effective=%v", name, module, projected)
		}
	}
	if before["github.com/ncruces/go-sqlite3"] == nil {
		return fmt.Errorf("owned projection missing SQLite module")
	}
	return nil
}

const networkTemporalObservationPrefix = "GOMAD_NETWORK_TEMPORAL_OBSERVATION "

type networkTemporalValue struct {
	WorkflowID string   `json:"workflow_id"`
	RunID      string   `json:"run_id"`
	Result     string   `json:"result"`
	History    []byte   `json:"history"`
	EventIDs   []int64  `json:"event_ids"`
	Types      []string `json:"types"`
	Attempts   []struct {
		StartedNanos uint64 `json:"started_nanos"`
		EndedNanos   uint64 `json:"ended_nanos"`
		Code         string `json:"code"`
	} `json:"attempts"`
	Connections []struct {
		OpenedNanos uint64 `json:"opened_nanos"`
		LocalPort   int    `json:"local_port"`
		RemotePort  int    `json:"remote_port"`
	} `json:"connections"`
}

func networkTemporalValueFrom(t *testing.T, stdout []byte) networkTemporalValue {
	t.Helper()
	var value networkTemporalValue
	count := 0
	for _, line := range bytes.Split(stdout, []byte{'\n'}) {
		if !bytes.HasPrefix(line, []byte(networkTemporalObservationPrefix)) {
			continue
		}
		if err := json.Unmarshal(line[len(networkTemporalObservationPrefix):], &value); err != nil {
			t.Fatal(err)
		}
		count++
	}
	if count != 1 || value.WorkflowID == "" || value.RunID == "" || len(value.History) == 0 || len(value.EventIDs) == 0 || len(value.EventIDs) != len(value.Types) || value.Result != "durable timer survived rpc-reset" {
		t.Fatalf("invalid independent Temporal history observation: count=%d value=%+v", count, value)
	}
	for index, id := range value.EventIDs {
		if id != int64(index+1) {
			t.Fatalf("duplicate or lost Temporal history: %v", value.EventIDs)
		}
	}
	return value
}

func networkTemporalResetWitness(t *testing.T, result Result, observation networkTemporalValue) {
	t.Helper()
	var transcript struct {
		Operations []struct {
			At      uint64           `json:"at"`
			Fault   *NetworkFaultV2  `json:"fault"`
			Request netmodel.Request `json:"request"`
			Result  netmodel.Result  `json:"result"`
		} `json:"operations"`
	}
	if err := json.Unmarshal(result.NetworkTranscript, &transcript); err != nil {
		t.Fatal(err)
	}
	const resetAt = uint64(40 * time.Second)
	// Guest elapsed durations exclude the cooperative runtime's 1 ns clock origin.
	const clockOrigin = uint64(1)
	fault := false
	ports := map[int]bool{}
	for _, connection := range observation.Connections {
		if connection.OpenedNanos+clockOrigin < resetAt {
			ports[connection.LocalPort] = true
		}
	}
	if len(ports) == 0 {
		t.Fatal("no SDK connection existed before the named reset")
	}
	handles := map[uint64]bool{}
	pending := map[uint64]uint64{}
	resetError, reconnected := false, false
	for _, operation := range transcript.Operations {
		if operation.Fault != nil && operation.Fault.Name == "rpc-reset" && operation.Fault.Action == "reset" && operation.Fault.AtNanos == resetAt {
			fault = true
		}
		if operation.Request.Op == "dial" && operation.Result.State == "completed" && operation.Result.Error == "" && operation.Result.Handle != 0 {
			if operation.At < resetAt && ports[operation.Result.Local.Port] {
				handles[operation.Result.Handle] = true
			}
			for _, connection := range observation.Connections {
				if operation.At >= resetAt && connection.OpenedNanos+clockOrigin >= resetAt && operation.Result.Local.Port == connection.LocalPort {
					reconnected = true
				}
			}
		}
		if operation.Request.Op == "read" || operation.Request.Op == "write" {
			pending[operation.Request.RequestID] = operation.Request.Handle
		}
		handle := operation.Request.Handle
		if operation.Request.Op == "poll" {
			handle = pending[operation.Request.RequestID]
		}
		if operation.At >= resetAt && handles[handle] && (operation.Request.Op == "read" || operation.Request.Op == "write" || operation.Request.Op == "poll") && (operation.Result.Error == "use of closed network connection" || operation.Result.Error == "connection reset by peer") {
			resetError = true
		}
	}
	if !fault || !resetError || !reconnected {
		t.Fatalf("named reset lacks an established SDK stream error and fresh SDK connection: fault=%v stream_error=%v reconnected=%v", fault, resetError, reconnected)
	}
	failed := false
	retried := false
	for _, attempt := range observation.Attempts {
		if attempt.StartedNanos+clockOrigin < resetAt && attempt.EndedNanos+clockOrigin >= resetAt && attempt.Code == "Unavailable" {
			failed = true
		}
		if failed && attempt.StartedNanos+clockOrigin >= resetAt && attempt.Code == "OK" {
			retried = true
		}
	}
	if !retried {
		t.Fatalf("no in-flight history RPC failed across rpc-reset and then recovered: %+v", observation.Attempts)
	}
}

func TestTemporalNetworkNamedResetRetriesAndExactReplay(t *testing.T) {
	evidence := os.Getenv("GOMAD_WASM_NETWORK_TEMPORAL_ROOT")
	if evidence == "" {
		t.Skip("set GOMAD_WASM_NETWORK_TEMPORAL_ROOT to run the actual Temporal network retry gate")
	}
	if !filepath.IsAbs(evidence) {
		t.Fatal("GOMAD_WASM_NETWORK_TEMPORAL_ROOT must be absolute")
	}
	t.Setenv("GOMAD_WASM_PROTOTYPE_ROOT", evidence)
	root, err := filepath.Abs("../../../../../..")
	if err != nil {
		t.Fatal(err)
	}
	fixtureSource, err := os.ReadFile(filepath.Join(root, "tools", "gomad", "internal", "backends", "wasm", "testdata", "network_streams", "temporal_retry_test.go"))
	if err != nil {
		t.Fatal(err)
	}
	staged := filepath.Join(".tmp", "wasm-backend-temporal", "network-streams-retry-"+digest(fixtureSource)[:12])
	if err := temporalStageFixture(root, filepath.Join("tools", "gomad", "internal", "backends", "wasm", "testdata", "network_streams"), root, staged); err != nil {
		t.Fatal(err)
	}
	module, _ := prototypeBuild(t, root, "test", "./"+filepath.ToSlash(staged), filepath.Join(prototypeRoot(t), "network-temporal-retry.wasm"), "test_dep", "disable_grpc_modules", "sqlite3_dotlk")
	config := prototypeTemporalConfig(t, root, "/workspace/network-temporal", "-test.run=^TestTemporalWorkflowRecoversNamedNetworkReset$", "-persistenceType=sql", "-persistenceDriver=sqlite")
	config.NetworkV2 = &NetworkModelV2{Profile: NetworkStreamProfile,
		Limits: netmodel.Limits{Listeners: 128, Connections: 4096, Bytes: 128 << 20, ConnectionBytes: 4 << 20, StallNanos: 10_000_000_000, Transitions: 1 << 20},
		Faults: []NetworkFaultV2{{Name: "rpc-reset", Action: "reset", AtNanos: uint64(40 * time.Second)}},
	}
	request := prototypeRequest(t, module, config, 7, 15*time.Minute)
	request.DigestTranscript = false
	observed, err := Run(t.Context(), request)
	retain := func(label string, result Result) {
		for suffix, data := range map[string][]byte{"stdout": result.Stdout, "stderr": result.Stderr, "wasi.bin": result.Transcript, "network.json": result.NetworkTranscript, "choices": result.ChoiceTrace.Bytes} {
			temporalWrite(t, filepath.Join(evidence, "network-temporal-"+label+"."+suffix), data)
		}
		temporalJSON(t, filepath.Join(evidence, "network-temporal-"+label+".terminal.json"), map[string]any{
			"termination": result.Termination, "message": result.Message, "exit_code": result.ExitCode, "reaped": result.Reaped,
			"environment_sha256": result.EnvironmentIdentity, "model_sha256": ImplementationSHA256(), "network_model_sha256": netmodel.ImplementationSHA256(),
			"module_sha256": request.ModuleSHA256, "helper_sha256": request.HelperSHA256, "runtime_sha256": request.Runtime.Choice.ExecutionIdentity.ToolchainBuildKey,
			"clock": request.Config.Clock, "network": request.Config.NetworkV2,
		})
	}
	retain("record", observed)
	if err != nil {
		t.Fatal(err)
	}
	requireExit(t, "Temporal named network reset recording", observed, 0)
	value := networkTemporalValueFrom(t, observed.Stdout)
	networkTemporalResetWitness(t, observed, value)
	tape, err := choice.ProjectReplayPlan(observed.ChoiceTrace, request.Runtime.Choice.ExecutionIdentity)
	if err != nil {
		t.Fatal(err)
	}
	request.Runtime.Choice.Mode, request.Runtime.Choice.Tape = choice.ModeReplay, &tape
	replayed, err := Run(t.Context(), request)
	retain("replay", replayed)
	if err != nil {
		t.Fatal(err)
	}
	requireExit(t, "Temporal named network reset replay", replayed, 0)
	if mismatch := sameEvidence(observed, replayed); mismatch != "" {
		t.Fatalf("Temporal named-reset replay differs: %s", mismatch)
	}
	if !bytes.Equal(observed.Transcript, replayed.Transcript) || !reflect.DeepEqual(value, networkTemporalValueFrom(t, replayed.Stdout)) {
		t.Fatal("exact replay changed I/O or independent workflow history")
	}
	temporalJSON(t, filepath.Join(evidence, "network-temporal-history-oracle.json"), value)
	temporalJSON(t, filepath.Join(evidence, "network-temporal-retry-report.json"), map[string]any{
		"schema": "gomad3.wasm-temporal-network-retry/v1", "exact_replay": true,
		"model_sha256": ImplementationSHA256(), "network_model_sha256": netmodel.ImplementationSHA256(),
		"module_sha256": request.ModuleSHA256, "helper_sha256": request.HelperSHA256, "runtime_sha256": request.Runtime.Choice.ExecutionIdentity.ToolchainBuildKey,
		"environment_sha256": observed.EnvironmentIdentity, "seed": request.Runtime.Seed, "clock": request.Config.Clock, "network": request.Config.NetworkV2,
		"wasi_sha256": digest(observed.Transcript), "choices_sha256": digest(observed.ChoiceTrace.Bytes), "network_sha256": digest(observed.NetworkTranscript),
		"history_sha256": digest(value.History), "history": value, "termination": observed.Termination, "exit_code": observed.ExitCode,
	})
	t.Logf("Temporal history RPC retried after named reset; events=%d attempts=%d connections=%d choices=%d", len(value.EventIDs), len(value.Attempts), len(value.Connections), len(observed.ChoiceTrace.Records))
}

func TestTemporalHostDiagnosticInputs(t *testing.T) {
	for _, invalid := range []string{"not-a-duration", "0s", "-1s", "16m"} {
		if _, err := temporalPairBudget(invalid); err == nil {
			t.Fatalf("accepted invalid diagnostic budget %q", invalid)
		}
	}
	for _, value := range []struct {
		input string
		want  time.Duration
	}{{"", 15 * time.Minute}, {"3m", 3 * time.Minute}} {
		actual, err := temporalPairBudget(value.input)
		if err != nil || actual != value.want {
			t.Fatalf("budget %q: got %v, %v", value.input, actual, err)
		}
	}
	root := t.TempDir()
	if actual, err := temporalCompileCachePath(root, ""); err != nil || actual != filepath.Join(root, "compile-cache") {
		t.Fatalf("default cache changed: %s, %v", actual, err)
	}
	for _, invalid := range []string{"relative/cache", string(filepath.Separator), root + "/../"} {
		if _, err := temporalCompileCachePath(root, invalid); err == nil {
			t.Fatalf("accepted unsafe cache path %q", invalid)
		}
	}
	cache := filepath.Join(root, "owned-cache")
	if actual, err := temporalCompileCachePath(root, cache); err != nil || actual != cache {
		t.Fatalf("explicit cache changed: %s, %v", actual, err)
	}
}

func TestTemporalBrokerBoundedWorldAdmission(t *testing.T) {
	config := temporalBrokerConfiguration(&NetworkModelV2{Profile: NetworkStreamProfile,
		Limits: netmodel.Limits{Listeners: 128, Connections: 4096, Bytes: 128 << 20, ConnectionBytes: 4 << 20, StallNanos: 10_000_000_000, Transitions: 1 << 20}})
	if config.WorldProfile != world.CompressedSnapshotSchema || config.Limits != world.CompressedSnapshotProfile().Limits {
		t.Fatal("Temporal producer lost its explicit bounded World profile")
	}
	if _, err := NewBroker(config); err != nil {
		t.Fatal(err)
	}
	for _, mutate := range []func(*BrokerConfig){
		func(c *BrokerConfig) { c.Limits.MaxRequests++ },
		func(c *BrokerConfig) { c.Limits.MaxEvents++ },
		func(c *BrokerConfig) { c.Limits.MaxQueuedEvents++ },
		func(c *BrokerConfig) { c.Limits.MaxTransitions++ },
		func(c *BrokerConfig) { c.Limits.MaxPayloadBytes++ },
		func(c *BrokerConfig) { c.Limits.MaxStringBytes++ },
		func(c *BrokerConfig) { c.Limits.MaxRequests = 0 },
		func(c *BrokerConfig) { c.WorldProfile = "unknown" },
	} {
		invalid := config
		mutate(&invalid)
		if _, err := NewBroker(invalid); err == nil {
			t.Fatal("Temporal producer admitted an oversized, implicit or unknown World profile")
		}
	}
}

func TestTemporalBrokerReadinessSource(t *testing.T) {
	compiler := os.Getenv("GOMAD3_STOCK_GO")
	if !filepath.IsAbs(compiler) {
		t.Skip("set absolute GOMAD3_STOCK_GO for the ordinary fixture-source readiness unit test")
	}
	root, err := filepath.Abs("../../../../../..")
	if err != nil {
		t.Fatal(err)
	}
	var body []byte
	for _, name := range []string{"temporal_storage/recovery_test.go", "temporal_broker/replication_test.go"} {
		data, err := os.ReadFile(filepath.Join(root, "tools/gomad/internal/backends/wasm/testdata", name))
		if err != nil {
			t.Fatal(err)
		}
		body = append(body, data...)
	}
	staged := filepath.Join(".tmp", "wasm-backend-temporal", "temporal_broker-"+digest(body)[:16])
	temporalStageBrokerFixture(t, root, staged)
	environment := append(os.Environ(), "GOOS="+runtime.GOOS, "GOARCH="+runtime.GOARCH, "CGO_ENABLED=0", "GOENV=off", "GOFLAGS=", "GOWORK=off", "GOTOOLCHAIN=local", "GOROOT="+filepath.Dir(filepath.Dir(compiler)))
	temporalCommand(t, t.TempDir(), "readiness-source", root, environment, []string{compiler, "test", "-mod=readonly", "-tags", "test_dep,disable_grpc_modules,sqlite3_dotlk", "-run", "^TestReplicationReadinessRequiresRemoteAcknowledgment$", "-count=1", "./" + filepath.ToSlash(staged)})
}

func temporalPairBudget(value string) (time.Duration, error) {
	if value == "" {
		return 15 * time.Minute, nil
	}
	budget, err := time.ParseDuration(value)
	if err != nil || budget <= 0 || budget > 15*time.Minute {
		return 0, fmt.Errorf("diagnostic budget must be positive and at most 15m: %q", value)
	}
	return budget, nil
}

func temporalStageBrokerFixture(t *testing.T, root, staged string) {
	t.Helper()
	destination := filepath.Join(root, staged)
	if err := os.MkdirAll(destination, 0700); err != nil {
		t.Fatal(err)
	}
	sources := map[string]string{
		"recovery_test.go":    filepath.Join(root, "tools/gomad/internal/backends/wasm/testdata/temporal_storage/recovery_test.go"),
		"replication_test.go": filepath.Join(root, "tools/gomad/internal/backends/wasm/testdata/temporal_broker/replication_test.go"),
	}
	entries, err := os.ReadDir(destination)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if sources[entry.Name()] == "" || !entry.Type().IsRegular() {
			t.Fatalf("unexpected staged broker fixture %q", entry.Name())
		}
	}
	for name, source := range sources {
		expected, err := os.ReadFile(source)
		if err != nil {
			t.Fatal(err)
		}
		path := filepath.Join(destination, name)
		actual, err := os.ReadFile(path)
		if os.IsNotExist(err) {
			if err := os.WriteFile(path, expected, 0600); err != nil {
				t.Fatal(err)
			}
		} else if err != nil || !bytes.Equal(actual, expected) {
			t.Fatalf("staged broker source body differs from %s", source)
		}
	}
}

func temporalBrokerConfiguration(network *NetworkModelV2) BrokerConfig {
	return BrokerConfig{
		Nodes: []BrokerNode{{ID: "active", Incarnation: 1}, {ID: "standby", Incarnation: 1}}, Seed: 7,
		Limits: world.CompressedSnapshotProfile().Limits, WorldProfile: world.CompressedSnapshotSchema,
		Network: network, Addresses: map[string]string{"active": "10.0.0.1", "standby": "10.0.0.2"}, Diagnostics: true,
		Faults: []BrokerNetworkFault{
			{Name: "temporal-active-to-standby-partition", Action: "partition", From: "active", To: "standby", AtNanos: 1},
			{Name: "temporal-active-to-standby-heal", Action: "heal", From: "active", To: "standby", AtNanos: uint64(30 * time.Second)},
		},
	}
}

func TestTemporalIndependentClustersPartitionHealWholeClusterCrash(t *testing.T) {
	request, evidence := temporalRecoveryRequest(t, "temporal_broker")
	temporalIndependentClustersPartitionHealWholeClusterCrash(t, request, evidence)
}

func temporalRunPair(t *testing.T, evidence, label string, requests []Request, warmFirst bool, observers ...temporalPairObserver) ([]Result, []error) {
	t.Helper()
	return temporalObservedPair(t, evidence, label, requests, observers, func() ([]Result, []error) {
		return temporalRunPairExecution(t, evidence, label, requests, warmFirst)
	})
}

func temporalReplayClusters(t *testing.T, evidence string, configuration BrokerConfig, requests []Request, crashed, recovered []Result, expected []byte, observers ...temporalPairObserver) {
	t.Helper()
	b, err := NewBroker(configuration)
	if err != nil {
		t.Fatal(err)
	}
	names := []string{"active", "standby"}
	if err := b.CrashAfterOutputs(names, []byte(temporalObservationPrefix)); err != nil {
		t.Fatal(err)
	}
	for i, name := range names {
		if !crashed[i].RuntimeInterrupted {
			t.Fatalf("%s crash lost its interrupted choice evidence", name)
		}
		tape, err := choice.ProjectInterruptedReplayPlan(crashed[i].InterruptedChoiceBytes, requests[i].Runtime.Choice.Limit, requests[i].Runtime.Choice.ExecutionIdentity)
		if err != nil {
			t.Fatal(err)
		}
		control, choices := *requests[i].Runtime, *requests[i].Runtime.Choice
		choices.Mode, choices.Tape = choice.ModePrefix, &tape
		control.Choice, requests[i].Runtime = &choices, &control
		requests[i].Broker, requests[i].StdoutMirror = b.Session(name), nil
		requests[i].Config.Environment = append([]string(nil), requests[i].Config.Environment...)
	}
	actual, errors := temporalRunPair(t, evidence, "cluster-prefix-replay", requests, true, observers...)
	for i, name := range names {
		if errors[i] != nil || actual[i].Termination != "crash" || !actual[i].Reaped || actual[i].StorageImage == nil ||
			!bytes.Equal(crashed[i].Stdout, actual[i].Stdout) || !bytes.Equal(crashed[i].StorageTranscript, actual[i].StorageTranscript) ||
			!bytes.Equal(crashed[i].NetworkTranscript, actual[i].NetworkTranscript) || !bytes.Equal(crashed[i].InterruptedChoiceBytes, actual[i].InterruptedChoiceBytes) {
			t.Fatalf("%s interrupted cluster replay changed evidence: termination=%s message=%s err=%v", name, actual[i].Termination, actual[i].Message, errors[i])
		}
		tape, err := choice.ProjectReplayPlan(recovered[i].ChoiceTrace, requests[i].Runtime.Choice.ExecutionIdentity)
		if err != nil {
			t.Fatal(err)
		}
		control, choices := *requests[i].Runtime, *requests[i].Runtime.Choice
		choices.Mode, choices.Tape = choice.ModeReplay, &tape
		control.Choice, requests[i].Runtime = &choices, &control
		requests[i].StorageImage, requests[i].StdoutMirror = actual[i].StorageImage, nil
		for j, value := range requests[i].Config.Environment {
			if value == "GOMAD_TEMPORAL_PHASE=commit" {
				requests[i].Config.Environment[j] = "GOMAD_TEMPORAL_PHASE=reopen"
			}
		}
		requests[i].Broker, err = b.Restart(name)
		if err != nil {
			t.Fatal(err)
		}
	}
	actual, errors = temporalRunPair(t, evidence, "cluster-full-replay", requests, false, observers...)
	for i, name := range names {
		if errors[i] != nil || actual[i].Termination != "exit" || actual[i].ExitCode == nil || *actual[i].ExitCode != 0 || !actual[i].Reaped ||
			!bytes.Equal(recovered[i].Stdout, actual[i].Stdout) || !bytes.Equal(recovered[i].StorageTranscript, actual[i].StorageTranscript) ||
			!bytes.Equal(recovered[i].NetworkTranscript, actual[i].NetworkTranscript) || !bytes.Equal(recovered[i].ChoiceTrace.Bytes, actual[i].ChoiceTrace.Bytes) {
			t.Fatalf("%s full cluster replay changed evidence: termination=%s message=%s err=%v", name, actual[i].Termination, actual[i].Message, errors[i])
		}
		temporalHistoryOracles(t, evidence, "temporal.replication."+name+"-replayed", temporalRecoveryValue(t, recovered[i].Stdout), temporalRecoveryValue(t, actual[i].Stdout))
	}
	encodedWorld, err := b.EncodeWorldEvidence()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(expected, encodedWorld) {
		t.Fatal("forced independent cluster replay changed shared World delivery order")
	}
	temporalWrite(t, filepath.Join(evidence, "two-cluster-replayed-world.json"), encodedWorld)
	temporalJSON(t, filepath.Join(evidence, "two-cluster-replayed-world-cardinality.json"), b.DiagnosticCardinality())
}

type temporalGuestWarm struct {
	ready chan struct{}
	once  sync.Once
	ack   sync.Once
	test  *testing.T
	node  string
}

func (w *temporalGuestWarm) Write(data []byte) (int, error) {
	if w.ready != nil {
		w.once.Do(func() { close(w.ready) })
	}
	if bytes.Contains(data, []byte(temporalObservationPrefix)) {
		w.ack.Do(func() { w.test.Logf("%s acknowledged committed Temporal history", w.node) })
	}
	return len(data), nil
}

func TestHelperBuildPublishesConfiguredTarget(t *testing.T) {
	root := t.TempDir()
	makefile, err := os.ReadFile("../Makefile")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "Makefile"), makefile, 0600); err != nil {
		t.Fatal(err)
	}
	bin := filepath.Join(root, "bin")
	target := filepath.Join(root, "cargo-target")
	if err := os.MkdirAll(bin, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(target, "release"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(bin, "cargo"), []byte("#!/bin/sh\nexit 0\n"), 0700); err != nil {
		t.Fatal(err)
	}
	compiled := []byte("current configured helper")
	if err := os.WriteFile(filepath.Join(target, "release/gomad3-wasmhost"), compiled, 0700); err != nil {
		t.Fatal(err)
	}
	command := exec.Command("make", "-C", root, "helper")
	command.Env = append(os.Environ(), "CARGO_TARGET_DIR="+target, "PATH="+bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("helper build: %v\n%s", err, output)
	}
	installed, err := os.ReadFile(filepath.Join(root, "wasmhost/target/release/gomad3-wasmhost"))
	if err != nil || !bytes.Equal(installed, compiled) {
		t.Fatalf("configured Cargo target was not published to guest execution path: %q %v", installed, err)
	}
}

func stringSeed(seed uint64) string    { return strconv.FormatUint(seed, 10) }
func hexDigest(digest [32]byte) string { return hex.EncodeToString(digest[:]) }

func TestHelperMigrationQualificationRejectsMissingBaselineCosts(t *testing.T) {
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	for _, fixture := range []struct {
		name, cost string
		reject     bool
	}{
		{"missing", "", true},
		{"zero", `,"WallNanos":0`, true},
		{"negative", `,"WallNanos":-1`, true},
		{"positive", `,"WallNanos":1`, false},
	} {
		t.Run(fixture.name, func(t *testing.T) {
			baseline, evidence := t.TempDir(), t.TempDir()
			for _, name := range []string{"scheduling", "virtual-time"} {
				data := `{}`
				if name == "scheduling" {
					data = `{"Guests":{},"Runs":[{"Guest":"choices","Variant":"strict","Seed":0,"Instance":0,"Termination":"exit"` + fixture.cost + `}]}`
				}
				if err := os.WriteFile(filepath.Join(baseline, name+"-report.json"), []byte(data), 0600); err != nil {
					t.Fatal(err)
				}
			}
			result, err := hostexec.Run(t.Context(), hostexec.Request{
				Command: []string{executable, "-test.run=^TestHelperMigrationQualification$", "-test.v", "-test.timeout=5s"},
				Dir:     t.TempDir(),
				Env: []string{
					"GOMAD_WASM_MIGRATION_BASELINE=" + baseline,
					"GOMAD_WASM_PROTOTYPE_ROOT=" + evidence,
					"GOMAXPROCS=1", "GOMEMLIMIT=512MiB",
				},
				Timeout: 6 * time.Second, TerminateGrace: 100 * time.Millisecond, OutputLimit: 16 << 10,
			})
			if err != nil || result.Termination != hostexec.TerminationExit || result.ExitCode != 1 || !result.GroupGone {
				t.Fatalf("baseline preflight subprocess not reaped normally: %+v, %v", result, err)
			}
			output := string(result.Stdout.RawBytes) + string(result.Stderr.RawBytes)
			costRejected := strings.Contains(output, "baseline wall time must be positive")
			moduleReached := strings.Contains(output, "baseline guest module identity unavailable or changed")
			if fixture.reject && (!costRejected || moduleReached) || !fixture.reject && (costRejected || !moduleReached) {
				t.Fatalf("baseline cost preflight reject=%v did not precede module operations:\n%s", fixture.reject, output)
			}
			entries, err := os.ReadDir(evidence)
			if err != nil || len(entries) != 0 {
				t.Fatalf("baseline preflight published guest evidence: %v, %v", entries, err)
			}
		})
	}
}

// This gate executes the retained baseline modules, avoiding compiler/source
// drift and excluding cold engine compilation from the paired cost comparison.
func TestHelperMigrationQualification(t *testing.T) {
	root := os.Getenv("GOMAD_WASM_MIGRATION_BASELINE")
	if root == "" {
		t.Skip("set GOMAD_WASM_MIGRATION_BASELINE to retained pre-migration prototype evidence")
	}
	if !filepath.IsAbs(root) {
		t.Fatal("migration baseline must be absolute")
	}
	baseline := map[string]prototypeRun{}
	modules := map[string]string{}
	for _, name := range []string{"scheduling", "virtual-time"} {
		data, err := os.ReadFile(filepath.Join(root, name+"-report.json"))
		if err != nil {
			t.Fatal(err)
		}
		var report prototypeReport
		if err := json.Unmarshal(data, &report); err != nil {
			t.Fatal(err)
		}
		for _, run := range report.Runs {
			if run.Instance == 0 && !strings.Contains(run.Variant, "warm") && !strings.Contains(run.Variant, "replay") && !strings.HasPrefix(run.Variant, "tamper-") && run.Termination == "exit" {
				if run.WallNanos <= 0 {
					t.Fatalf("baseline wall time must be positive: report=%s guest=%s variant=%s seed=%d", name, run.Guest, run.Variant, run.Seed)
				}
				baseline[run.Guest+"/"+run.Variant+"/"+stringSeed(run.Seed)] = run
			}
		}
		for name, guest := range report.Guests {
			if hash, ok := guest["module_sha256"].(string); ok {
				modules[name] = hash
			}
		}
	}
	report := prototypeReportFor(t, "helper-migration")
	for _, guest := range []struct{ name, variant, module, identity string }{
		{"choices", "strict", "choices.wasm", "choices"},
		{"runtime_probes", "strict", "runtime_probes.wasm", "runtime_probes"},
		{"time_probes", "strict", "time_probes.wasm", "time_probes"},
	} {
		t.Run(guest.name+"/"+guest.variant, func(t *testing.T) {
			module := filepath.Join(root, "modules", guest.module)
			if want := modules[guest.identity]; want == "" || fileHash(t, module) != want {
				t.Fatal("baseline guest module identity unavailable or changed")
			}
			config := prototypeFixtureConfig()
			if guest.name == "runtime_probes" {
				config.Environment = append(config.Environment, "EXTRA_ENTROPY=no_")
			}
			request := prototypeRequest(t, module, config, 0, 30*time.Minute)
			request.CompileCache = filepath.Join(root, "compile-cache")
			if cache := os.Getenv("GOMAD_WASM_MIGRATION_CACHE"); cache != "" {
				if !filepath.IsAbs(cache) {
					t.Fatal("migration compiled cache must be absolute")
				}
				request.CompileCache = cache
			}
			warm := request
			warm.Config.Args = append([]string(nil), warm.Config.Args...)
			for index, arg := range warm.Config.Args {
				if strings.HasPrefix(arg, "-test.run=") {
					warm.Config.Args[index] = "-test.run=^$"
				}
			}
			prepared := prototypeExecute(t, report, guest.name, guest.variant+"-prepare", 0, warm)
			requireExit(t, "cache prepare", prepared, 0, 17)
			var first Result
			var beforeNanos, afterNanos int64
			for _, seed := range []uint64{0, 1000003} {
				request.Runtime.Seed = seed
				old, ok := baseline[guest.name+"/"+guest.variant+"/"+stringSeed(seed)]
				if !ok {
					t.Fatal("missing baseline seed")
				}
				started := time.Now()
				result := prototypeExecute(t, report, guest.name, guest.variant, 0, request)
				elapsed := time.Since(started).Nanoseconds()
				requireExit(t, "migrated", result, 0, 17)
				if digest(result.Stdout) != old.StdoutSHA256 || digest(result.ChoiceTrace.Bytes) != old.ChoiceSHA256 || digest(result.NetworkTranscript) != old.NetworkSHA256 || hexDigest(result.TranscriptSHA256) != old.WASISHA256 || int64FromTerminal(result.runtimeTerminal) != int64(old.VirtualNanos) || int(result.Calls) != old.Calls || result.TimeAdvances != old.TimeAdvances {
					t.Fatalf("baseline evidence changed: seed=%d stdout=%v choice=%v network=%v wasi=%v calls=%d/%d virtual=%d/%d", seed, digest(result.Stdout) == old.StdoutSHA256, digest(result.ChoiceTrace.Bytes) == old.ChoiceSHA256, digest(result.NetworkTranscript) == old.NetworkSHA256, hexDigest(result.TranscriptSHA256) == old.WASISHA256, result.Calls, old.Calls, int64FromTerminal(result.runtimeTerminal), old.VirtualNanos)
				}
				beforeNanos += old.WallNanos
				afterNanos += elapsed
				if seed == 0 {
					first = result
				}
			}
			tape, err := choice.ProjectReplayPlan(first.ChoiceTrace, request.Runtime.Choice.ExecutionIdentity)
			if err != nil {
				t.Fatal(err)
			}
			request.Runtime.Seed = 0
			request.Runtime.Choice.Mode, request.Runtime.Choice.Tape = choice.ModeReplay, &tape
			replayed := prototypeExecute(t, report, guest.name, guest.variant+"-replay", 0, request)
			if difference := sameEvidence(first, replayed); difference != "" {
				t.Fatalf("migrated replay differs in %s: %s %s", difference, replayed.Termination, replayed.Message)
			}
			speedup := float64(beforeNanos) / float64(afterNanos)
			t.Logf("same module/seeds: baseline=%s migrated=%s speedup=%.3fx", time.Duration(beforeNanos), time.Duration(afterNanos), speedup)
			report.guest(guest.variant, map[string]any{"module_sha256": request.ModuleSHA256, "baseline_wall_nanos": beforeNanos, "migrated_wall_nanos": afterNanos, "speedup": speedup, "byte_identical": true})
		})
	}
}

// TestHelperMatchedSuiteQualification compares both execution owners against a
// single immutable compiled suite. No historical report can substitute for the
// reference execution, and compilation is excluded by explicit warm runs.
func TestHelperMatchedSuiteQualification(t *testing.T) {
	module := os.Getenv("GOMAD_WASM_MATCHED_MODULE")
	baselineHelper := os.Getenv("GOMAD_WASM_MATCHED_BASELINE_HELPER")
	if module == "" || baselineHelper == "" {
		t.Skip("set matched immutable module and original helper")
	}
	if !filepath.IsAbs(module) || !filepath.IsAbs(baselineHelper) {
		t.Fatal("matched inputs must be absolute")
	}
	// The runtime identity is the cooperative overlay that compiled the module;
	// an operator-supplied identity is only a cross-check.
	_, _, toolchainIdentity := prototypeToolchain(t)
	if declared := os.Getenv("GOMAD_WASM_MATCHED_TOOLCHAIN_ID"); declared != "" && declared != toolchainIdentity {
		t.Fatalf("declared matched toolchain identity %s differs from cooperative overlay %s", declared, toolchainIdentity)
	}
	data, err := os.ReadFile(module)
	if err != nil {
		t.Fatal(err)
	}
	imports, pages := binaryInventory(t, data)
	helper := helperPath(t)
	repo, err := filepath.Abs("../../../../../..")
	if err != nil {
		t.Fatal(err)
	}
	report := prototypeReportFor(t, "helper-matched-suite")
	cache := os.Getenv("GOMAD_WASM_MATCHED_CACHE")
	if cache == "" {
		cache = filepath.Join(prototypeRoot(t), "compile-cache")
	}
	if !filepath.IsAbs(cache) {
		t.Fatal("matched compiled cache must be absolute")
	}
	// The baseline helper's compiled-code cache is filename-keyed and
	// unverified, so it must never share the candidate's verified cache.
	baselineCache := os.Getenv("GOMAD_WASM_MATCHED_BASELINE_CACHE")
	if err := matchedBaselineCache(baselineCache, cache); err != nil {
		t.Fatal(err)
	}
	selection := os.Getenv("GOMAD_WASM_MATCHED_SUITE")
	preparations, err := os.OpenFile(filepath.Join(prototypeRoot(t), "matched-preparations.jsonl"), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := preparations.Close(); err != nil {
			t.Error(err)
		}
	})
	preparationEpoch := 0
	preparationEvent := func(t *testing.T, event string, request Request, cold bool) {
		data, err := json.Marshal(struct {
			Event        string `json:"event"`
			Epoch        int    `json:"epoch"`
			Helper       string `json:"helper"`
			HelperSHA256 string `json:"helper_sha256"`
			Cold         bool   `json:"cold"`
		}{event, preparationEpoch, request.HelperPath, request.HelperSHA256, cold})
		if err != nil {
			t.Fatal(err)
		}
		if _, err := preparations.Write(append(data, '\n')); err != nil {
			t.Fatal(err)
		}
	}
	for _, suite := range []struct{ name, cwd, run string }{
		{"workflow", "/workspace/tests", "^TestWorkflowTypeEncodingSuite$/^TestPlainASCII$/^Succeeds$"},
		{"activity", "/workspace/tests", "^TestActivityTestSuite$"},
		{"child", "/workspace/tests", "^TestChildWorkflowSuite$"},
		{"update", "/workspace/tests", "^TestWorkflowUpdateSuite$"},
		{"timers", "/workspace/tests", "^TestUserTimersTestSuite$"},
		{"frontend", "/workspace/tests/gomadfunctional", "^TestFrontendSystemInfo$"},
	} {
		if !matchedSuiteSelected(selection, suite.name) {
			continue
		}
		t.Run(suite.name, func(t *testing.T) {
			config := prototypeTemporalConfig(t, repo, suite.cwd, "-test.run="+suite.run, "-persistenceType=sql", "-persistenceDriver=sqlite")
			config.Profile, config.Clock.ReadStepNanos = CooperativeProfile, 0
			identity := choice.ExecutionIdentity{TargetSHA256: sha256.Sum256(data), ToolchainBuildKey: toolchainIdentity, GOOS: "wasip1", GOARCH: "wasm", ImplementationSHA256: sha256.Sum256([]byte(toolchainIdentity))}
			request := Request{HelperPath: helper, HelperSHA256: fileHash(t, helper), ModulePath: module, ModuleSHA256: fileHash(t, module), Engine: PinnedEngine(), Imports: imports, InitialMemoryPages: pages, MemoryBytes: 4 << 30, Fuel: 1 << 50, Config: config, Timeout: 30 * time.Minute, CompileCache: cache, DigestTranscript: true, Runtime: &RuntimeControl{Choice: &runnerbackend.ChoiceRequest{Mode: choice.ModeRecord, ExecutionIdentity: identity, Limit: choice.MaximumTraceBytes}}}
			baseline := request
			baseline.HelperPath, baseline.HelperSHA256, baseline.remoteModel, baseline.CompileCache = baselineHelper, fileHash(t, baselineHelper), true, baselineCache
			for index, warm := range []Request{baseline, request} {
				warm.Config.Args = append([]string(nil), warm.Config.Args...)
				for index, arg := range warm.Config.Args {
					if strings.HasPrefix(arg, "-test.run=") {
						warm.Config.Args[index] = "-test.run=^$"
					}
				}
				started := time.Now()
				entries, err := os.ReadDir(warm.CompileCache)
				if err != nil {
					t.Fatal(err)
				}
				preparationEvent(t, "start", warm, len(entries) == 0)
				result, err := matchedPrepareAdmitted(runtime.GOOS, "/proc/meminfo", os.Getenv("GOMAD_WASM_MATCHED_COLD_MEMORY_GUARD"), func() (Result, error) {
					return Run(context.Background(), warm)
				})
				preparationEvent(t, "end", warm, len(entries) == 0)
				preparationEpoch++
				if err != nil {
					t.Fatal(err)
				}
				requireExit(t, "cache warmup", result, 0)
				report.guest(suite.name+"-prepare-"+strconv.Itoa(index), map[string]any{"helper_sha256": warm.HelperSHA256, "module_sha256": warm.ModuleSHA256, "wall_nanos": time.Since(started).Nanoseconds(), "phase_metrics": result.Telemetry, "wait_resource_usage": result.HelperResourceUsage})
			}
			var beforeNanos, afterNanos int64
			var beforeExecutionNanos, afterExecutionNanos uint64
			var first Result
			for sample, seed := range []uint64{0, 1000003, 2000006, 3000009} {
				request.Runtime.Seed = seed
				var before, after Result
				// Alternate the order so neither owner always runs on a warmer host.
				for turn := range 2 {
					started := time.Now()
					if matchedBaselineTurn(sample, turn) {
						before = prototypeExecute(t, report, suite.name, "baseline", 0, baseline)
						beforeNanos += time.Since(started).Nanoseconds()
					} else {
						after = prototypeExecute(t, report, suite.name, "candidate", 0, request)
						afterNanos += time.Since(started).Nanoseconds()
					}
				}
				if before.AuthorizationToTerminalNanos == 0 || after.AuthorizationToTerminalNanos == 0 {
					t.Fatal("validated authorization-to-terminal measurements unavailable")
				}
				beforeExecutionNanos += before.AuthorizationToTerminalNanos
				afterExecutionNanos += after.AuthorizationToTerminalNanos
				requireExit(t, "baseline", before, 0)
				requireExit(t, "candidate", after, 0)
				if after.GoFilesystemCalls == nil || len(after.GoFilesystemCalls) != 0 {
					t.Fatal("matched candidate namespace leaked filesystem callbacks", after.GoFilesystemCalls)
				}
				witness := []byte("=== RUN   " + strings.Trim(strings.Split(suite.run, "/")[0], "^$"))
				if !bytes.Contains(before.Stdout, witness) || !bytes.Contains(after.Stdout, witness) {
					t.Fatalf("selected suite did not execute: missing %q", witness)
				}
				if difference := sameEvidence(before, after); difference != "" || before.EnvironmentIdentity != after.EnvironmentIdentity || before.Calls != after.Calls || before.TimeAdvances != after.TimeAdvances {
					t.Fatalf("same-module paired evidence differs: %s seed=%d", difference, seed)
				}
				if seed == 0 {
					first = before
				}
			}
			plan, err := choice.ProjectReplayPlan(first.ChoiceTrace, identity)
			if err != nil {
				t.Fatal(err)
			}
			request.Runtime.Seed = 0
			request.Runtime.Choice.Mode, request.Runtime.Choice.Tape = choice.ModeReplay, &plan
			replayed := prototypeExecute(t, report, suite.name, "candidate-replay", 0, request)
			if difference := sameEvidence(first, replayed); difference != "" {
				t.Fatalf("original-helper tape differs in candidate: %s", difference)
			}
			speedup := float64(beforeNanos) / float64(afterNanos)
			executionSpeedup := float64(beforeExecutionNanos) / float64(afterExecutionNanos)
			report.guest(suite.name, map[string]any{"compatibility_profile": "prototype-legacy-v1", "module_sha256": request.ModuleSHA256, "runtime_identity": identity.ToolchainBuildKey, "model_sha256": ImplementationSHA256(), "reference_owner": "current_go_model", "host_os": runtime.GOOS, "host_arch": runtime.GOARCH, "engine": request.Engine, "environment_identity": first.EnvironmentIdentity, "baseline_helper_sha256": baseline.HelperSHA256, "candidate_helper_sha256": request.HelperSHA256, "baseline_wall_nanos": beforeNanos, "candidate_wall_nanos": afterNanos, "speedup": speedup, "byte_identical": true})
			report.guest(suite.name+"-execution-boundary", map[string]any{"module_sha256": request.ModuleSHA256, "runtime_identity": identity.ToolchainBuildKey, "samples": 4, "order": "alternating-baseline-first-on-even", "boundary": "authorize-send-to-validated-terminal-includes-instantiation-and-evidence-flush", "baseline_nanos": beforeExecutionNanos, "candidate_nanos": afterExecutionNanos, "speedup": executionSpeedup, "full_wall_speedup": speedup})
			t.Logf("same module/seeds/tape: full wall baseline=%s candidate=%s speedup=%.3fx; authorized execution baseline=%s candidate=%s speedup=%.3fx", time.Duration(beforeNanos), time.Duration(afterNanos), speedup, time.Duration(beforeExecutionNanos), time.Duration(afterExecutionNanos), executionSpeedup)
		})
	}
}

// The frontend suite stays opt-in; a selection is a comma-separated list.
func matchedSuiteSelected(selection, suite string) bool {
	if selection == "" {
		return suite != "frontend"
	}
	for _, name := range strings.Split(selection, ",") {
		if strings.TrimSpace(name) == suite {
			return true
		}
	}
	return false
}

func matchedBaselineCache(baseline, candidate string) error {
	if !filepath.IsAbs(baseline) || filepath.Clean(baseline) == filepath.Clean(candidate) {
		return errors.New("GOMAD_WASM_MATCHED_BASELINE_CACHE must be absolute and distinct from the candidate cache")
	}
	return nil
}

// Even samples run the baseline first and odd samples the candidate first.
func matchedBaselineTurn(sample, turn int) bool {
	return (sample+turn)%2 == 0
}

func TestMatchedSuiteSelectionCacheAndOrder(t *testing.T) {
	for _, test := range []struct {
		selection, suite string
		want             bool
	}{{"", "workflow", true}, {"", "frontend", false}, {"frontend", "frontend", true}, {"workflow, frontend", "frontend", true}, {"workflow,update", "update", true}, {"workflow,update", "timers", false}} {
		if got := matchedSuiteSelected(test.selection, test.suite); got != test.want {
			t.Fatalf("selection %q suite %s = %t", test.selection, test.suite, got)
		}
	}
	candidate := filepath.Join(t.TempDir(), "candidate")
	for _, baseline := range []string{"", "relative", candidate, candidate + "/"} {
		if matchedBaselineCache(baseline, candidate) == nil {
			t.Fatalf("baseline cache %q admitted", baseline)
		}
	}
	if err := matchedBaselineCache(filepath.Join(t.TempDir(), "baseline"), candidate); err != nil {
		t.Fatal(err)
	}
	for sample, want := range []bool{true, false, true, false} {
		if matchedBaselineTurn(sample, 0) != want || matchedBaselineTurn(sample, 1) == want {
			t.Fatalf("sample %d order is not alternating", sample)
		}
	}
}

func matchedPrepareAdmitted(platform, path, mode string, dispatch func() (Result, error)) (Result, error) {
	if mode == "" {
		return dispatch()
	}
	if mode != "1" || platform != "linux" {
		return Result{}, errors.New("explicit matched memory admission requires mode 1 on Linux")
	}
	data, err := hostfs.ReadBounded(path, 8<<10)
	if err != nil {
		return Result{}, err
	}
	seen := false
	var available uint64
	for _, line := range strings.Split(string(data), "\n") {
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		key, _, _ := strings.Cut(fields[0], ":")
		if !strings.EqualFold(key, "MemAvailable") {
			continue
		}
		if seen || fields[0] != "MemAvailable:" || len(fields) != 3 || fields[2] != "kB" {
			return Result{}, errors.New("unique exact MemAvailable field in kB required")
		}
		seen = true
		value, err := matchedMemoryDecimal(fields[1])
		if err != nil || value > ^uint64(0)/1024 {
			return Result{}, errors.New("invalid or overflowing MemAvailable value")
		}
		available = value * 1024
	}
	if !seen || available < 8<<30 {
		return Result{}, errors.New("at least 8GiB MemAvailable required before matched preparation dispatch")
	}
	return dispatch()
}

func matchedMemoryDecimal(value string) (uint64, error) {
	if len(value) == 0 || len(value) > 20 {
		return 0, errors.New("bounded memory decimal required")
	}
	for _, character := range value {
		if character < '0' || character > '9' {
			return 0, errors.New("memory decimal digits required")
		}
	}
	return strconv.ParseUint(value, 10, 64)
}

func TestMatchedPreparationMemoryBoundaryBeforeBothOwners(t *testing.T) {
	for _, owner := range []string{"original", "candidate"} {
		t.Run(owner, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "meminfo")
			for _, test := range []struct {
				data string
				want bool
			}{
				{"MemAvailable: 8388607 kB\n", false},
				{"MemAvailable: 8388608 kB\n", true},
				{"MemAvailable: 8388609 kB\n", true},
			} {
				if err := os.WriteFile(path, []byte(test.data), 0600); err != nil {
					t.Fatal(err)
				}
				called := false
				result, err := matchedPrepareAdmitted("linux", path, "1", func() (Result, error) {
					called = true
					return Result{Termination: owner}, nil
				})
				if called != test.want || (err == nil) != test.want || (result.Termination == owner) != test.want {
					t.Fatalf("owner=%s observation=%q dispatched=%t want=%t error=%v", owner, test.data, called, test.want, err)
				}
			}
		})
	}
}

func TestMatchedPreparationMemoryRejectsMalformedBeforeDispatch(t *testing.T) {
	for _, data := range []string{
		"", "MemTotal: 100 kB\n", "MemAvailable: 8388608 kB\nMemAvailable: 8388608 kB\n",
		"memavailable: 8388608 kB\n", "MemAvailable: -1 kB\n", "MemAvailable: +1 kB\n",
		"MemAvailable: 8388608 MB\n", "MemAvailable: 8388608 kB extra\n", "MemAvailable: 8388608\n",
		"MemAvailable: 8388608 kB\nMemAvailable : 8388608 kB\n", "MemAvailable: 8388608 kB\nMemAvailable:8388608 kB\n",
		"MemAvailable: 18446744073709551616 kB\n", "MemAvailable: 18014398509481984 kB\n",
		"MemAvailable: 8388608 kB\n" + strings.Repeat(" ", 8<<10),
	} {
		path := filepath.Join(t.TempDir(), "meminfo")
		if err := os.WriteFile(path, []byte(data), 0600); err != nil {
			t.Fatal(err)
		}
		for _, owner := range []string{"original", "candidate"} {
			called := false
			_, err := matchedPrepareAdmitted("linux", path, "1", func() (Result, error) {
				called = true
				return Result{Termination: owner}, nil
			})
			if called || err == nil {
				t.Fatalf("owner=%s malformed observation dispatched: %q error=%v", owner, data, err)
			}
		}
	}
}

func TestMatchedPreparationMemoryIsExplicitAndPlatformBound(t *testing.T) {
	for _, test := range []struct {
		platform, mode string
		want           bool
	}{
		{"darwin", "", true}, {"linux", "", true},
		{"darwin", "1", false}, {"linux", "1", false},
		{"linux", "0", false}, {"linux", "true", false}, {"linux", " 1", false},
	} {
		called := false
		_, err := matchedPrepareAdmitted(test.platform, "unreadable", test.mode, func() (Result, error) {
			called = true
			return Result{}, nil
		})
		if called != test.want || (err == nil) != test.want {
			t.Fatalf("platform=%s mode=%q called=%t error=%v", test.platform, test.mode, called, err)
		}
	}
}

func TestDiagnosticHelperStderrIsSeparateFromGuestEvidence(t *testing.T) {
	request := syntheticHelper(t, "read request\n")
	script := "#!/bin/sh\nread request\nprintf '%s\\n' '" + startedJSON(t, request) + "'\nread authorization\nprintf '%s\\n' 'diagnostic helper bytes' >&2\nprintf '%s\\n' '{\"type\":\"result\",\"termination\":\"exit\",\"exit_code\":0,\"message\":\"\",\"fuel_remaining\":100000000,\"peak_memory_bytes\":65536,\"output_bytes\":0}'\n"
	if err := os.WriteFile(request.HelperPath, []byte(script), 0700); err != nil {
		t.Fatal(err)
	}
	request.HelperSHA256 = fileHash(t, request.HelperPath)
	result, err := Run(t.Context(), request)
	if err != nil {
		t.Fatal(err)
	}
	requireExit(t, "diagnostic transport", result, 0)
	if string(result.helperStderr) != "diagnostic helper bytes\n" {
		t.Fatalf("helper diagnostic capture = %q", result.helperStderr)
	}
	if len(result.Stdout) != 0 || len(result.Stderr) != 0 || len(result.Transcript) != 0 || result.Calls != 0 {
		t.Fatalf("helper diagnostics entered guest evidence: stdout=%q stderr=%q transcript=%q calls=%d", result.Stdout, result.Stderr, result.Transcript, result.Calls)
	}
}

func TestHelperLegacyWorkflowProfile(t *testing.T) {
	root := os.Getenv("GOMAD_WASM_WORKFLOW_PROFILE_ROOT")
	if root == "" {
		t.Skip("set GOMAD_WASM_WORKFLOW_PROFILE_ROOT for diagnostic-only retained workflow profiling")
	}
	module, cache, runtimeIdentity := os.Getenv("GOMAD_WASM_MATCHED_MODULE"), os.Getenv("GOMAD_WASM_MATCHED_CACHE"), os.Getenv("GOMAD_WASM_MATCHED_TOOLCHAIN_ID")
	if !filepath.IsAbs(root) || !filepath.IsAbs(module) || !filepath.IsAbs(cache) || runtimeIdentity == "" {
		t.Fatal("profile requires absolute evidence/module/existing cache paths and explicit legacy runtime identity")
	}
	if err := os.MkdirAll(root, 0700); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(module)
	if err != nil {
		t.Fatal(err)
	}
	imports, pages := binaryInventory(t, data)
	repo, err := filepath.Abs("../../../../../..")
	if err != nil {
		t.Fatal(err)
	}
	config := prototypeTemporalConfig(t, repo, "/workspace/tests", "-test.run=^TestWorkflowTypeEncodingSuite$/^TestPlainASCII$/^Succeeds$", "-persistenceType=sql", "-persistenceDriver=sqlite")
	config.Profile, config.Clock.ReadStepNanos = CooperativeProfile, 0
	helper := helperPath(t)
	identity := choice.ExecutionIdentity{TargetSHA256: sha256.Sum256(data), ToolchainBuildKey: runtimeIdentity, GOOS: "wasip1", GOARCH: "wasm", ImplementationSHA256: sha256.Sum256([]byte(runtimeIdentity))}
	request := Request{HelperPath: helper, HelperSHA256: fileHash(t, helper), ModulePath: module, ModuleSHA256: fileHash(t, module), Engine: PinnedEngine(), Imports: imports, InitialMemoryPages: pages, MemoryBytes: 4 << 30, Fuel: 1 << 50, Config: config, Timeout: 3 * time.Minute, CompileCache: cache, DigestTranscript: true, Runtime: &RuntimeControl{Choice: &runnerbackend.ChoiceRequest{Mode: choice.ModeRecord, ExecutionIdentity: identity, Limit: choice.MaximumTraceBytes}}}
	var first Result
	for _, seed := range []uint64{0, 1000003} {
		request.Runtime.Seed = seed
		profile, err := os.Create(filepath.Join(root, "seed"+stringSeed(seed)+"-parent.pprof"))
		if err != nil {
			t.Fatal(err)
		}
		if err := pprof.StartCPUProfile(profile); err != nil {
			_ = profile.Close()
			t.Fatal(err)
		}
		started := time.Now()
		result, runErr := Run(t.Context(), request)
		wall := time.Since(started).Nanoseconds()
		pprof.StopCPUProfile()
		if err := profile.Close(); err != nil {
			t.Fatal(err)
		}
		if runErr != nil {
			t.Fatal(runErr)
		}
		requireExit(t, "profile", result, 0)
		if result.Telemetry == nil || !result.Telemetry.CacheHit || result.Telemetry.CompileNanos != 0 || !bytes.Contains(result.Stdout, []byte("=== RUN   TestWorkflowTypeEncodingSuite")) {
			t.Fatal("profile did not execute the unchanged selected workflow using existing compiled code")
		}
		temporalJSON(t, filepath.Join(root, "seed"+stringSeed(seed)+"-metrics.json"), map[string]any{"diagnostic_only": true, "profile": "prototype-legacy-v1", "module_sha256": request.ModuleSHA256, "helper_sha256": request.HelperSHA256, "runtime_identity": runtimeIdentity, "model_sha256": ImplementationSHA256(), "wall_nanos": wall, "authorization_to_terminal_nanos": result.AuthorizationToTerminalNanos, "phase_metrics": result.Telemetry, "wait_resource_usage": result.HelperResourceUsage, "choices": result.ChoiceTrace.Summary})
		if seed == 0 {
			first = result
		}
	}
	// Retain the otherwise digest-only canonical bytes in a separate diagnostic
	// run, never using its timing as the unchanged-suite performance measurement.
	request.Runtime.Seed, request.DigestTranscript = 0, false
	retained, err := Run(t.Context(), request)
	if err != nil {
		t.Fatal(err)
	}
	requireExit(t, "transcript retention", retained, 0)
	if difference := sameEvidence(first, retained); difference != "" {
		t.Fatalf("retaining canonical transcript changed evidence: %s", difference)
	}
	operations := map[string]uint64{}
	records, err := DecodeTranscript(retained.Transcript)
	if err != nil {
		t.Fatal(err)
	}
	for _, record := range records {
		operations[record.Op] += record.Count
	}
	temporalJSON(t, filepath.Join(root, "canonical-operation-counts.json"), operations)
	temporalWrite(t, filepath.Join(root, "seed0-choice.bin"), first.ChoiceTrace.Bytes)
}

func TestHelperFailingSuiteDiagnosticProfile(t *testing.T) {
	root := os.Getenv("GOMAD_WASM_DIAGNOSTIC_PROFILE_ROOT")
	if root == "" {
		t.Skip("set diagnostic profile inputs after the exclusive guest window is scheduled")
	}
	module, cache := os.Getenv("GOMAD_WASM_MATCHED_MODULE"), os.Getenv("GOMAD_WASM_MATCHED_CACHE")
	referenceHelper, referenceCache := os.Getenv("GOMAD_WASM_PROFILE_REFERENCE_HELPER"), os.Getenv("GOMAD_WASM_PROFILE_REFERENCE_CACHE")
	runtimeIdentity := os.Getenv("GOMAD_WASM_MATCHED_TOOLCHAIN_ID")
	for _, name := range []string{root, module, cache, referenceHelper, referenceCache} {
		if !filepath.IsAbs(name) {
			t.Fatal("diagnostic profile requires absolute paths")
		}
	}
	if runtimeIdentity == "" || fileHash(t, referenceHelper) == fileHash(t, helperPath(t)) {
		t.Fatal("diagnostic helper must have a distinct identity and explicit legacy runtime")
	}
	if err := os.MkdirAll(root, 0700); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(module)
	if err != nil {
		t.Fatal(err)
	}
	imports, pages := binaryInventory(t, data)
	repo, err := filepath.Abs("../../../../../..")
	if err != nil {
		t.Fatal(err)
	}
	identity := choice.ExecutionIdentity{TargetSHA256: sha256.Sum256(data), ToolchainBuildKey: runtimeIdentity, GOOS: "wasip1", GOARCH: "wasm", ImplementationSHA256: sha256.Sum256([]byte(runtimeIdentity))}
	preparations, err := os.OpenFile(filepath.Join(root, "matched-preparations.jsonl"), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := preparations.Close(); err != nil {
			t.Error(err)
		}
	})
	for index, suite := range []struct{ name, run string }{
		{"workflow", "^TestWorkflowTypeEncodingSuite$/^TestPlainASCII$/^Succeeds$"},
		{"child", "^TestChildWorkflowSuite$"},
		{"timers", "^TestUserTimersTestSuite$"},
	} {
		if !t.Run(suite.name, func(t *testing.T) {
			config := prototypeTemporalConfig(t, repo, "/workspace/tests", "-test.run="+suite.run, "-persistenceType=sql", "-persistenceDriver=sqlite")
			config.Profile, config.Clock.ReadStepNanos = CooperativeProfile, 0
			helper := helperPath(t)
			request := Request{HelperPath: helper, HelperSHA256: fileHash(t, helper), ModulePath: module, ModuleSHA256: fileHash(t, module), Engine: PinnedEngine(), Imports: imports, InitialMemoryPages: pages, MemoryBytes: 4 << 30, Fuel: 1 << 50, Config: config, Timeout: 5 * time.Minute, CompileCache: cache, DigestTranscript: true, Runtime: &RuntimeControl{Choice: &runnerbackend.ChoiceRequest{Mode: choice.ModeRecord, ExecutionIdentity: identity, Limit: choice.MaximumTraceBytes}}}
			reference := request
			reference.HelperPath, reference.HelperSHA256, reference.CompileCache = referenceHelper, fileHash(t, referenceHelper), referenceCache
			if index == 0 {
				for epoch, owner := range []struct {
					name    string
					request Request
				}{{"reference", reference}, {"diagnostic", request}} {
					warm := owner.request
					warm.Config.Args = append([]string(nil), warm.Config.Args...)
					for index, arg := range warm.Config.Args {
						if strings.HasPrefix(arg, "-test.run=") {
							warm.Config.Args[index] = "-test.run=^$"
						}
					}
					entries, err := os.ReadDir(warm.CompileCache)
					if err != nil {
						t.Fatal(err)
					}
					cold := len(entries) == 0
					profilePreparationEvent(t, preparations, "start", epoch, warm, cold)
					runDiagnosticProfile(t, root, owner.name+"-prepare", warm, owner.name == "diagnostic", !cold)
					profilePreparationEvent(t, preparations, "end", epoch, warm, cold)
				}
			}
			var first Result
			for _, seed := range []uint64{0, 1000003} {
				request.Runtime.Seed = seed
				before := runDiagnosticProfile(t, root, suite.name+"-reference-seed"+stringSeed(seed), reference, false, true)
				after := runDiagnosticProfile(t, root, suite.name+"-diagnostic-seed"+stringSeed(seed), request, true, true)
				requireDiagnosticEvidence(t, before, after)
				if !bytes.Contains(before.Stdout, []byte("=== RUN   "+strings.Trim(strings.Split(suite.run, "/")[0], "^$"))) {
					t.Fatal("diagnostic selected suite did not run")
				}
				if seed == 0 {
					first = before
				}
			}
			plan, err := choice.ProjectReplayPlan(first.ChoiceTrace, identity)
			if err != nil {
				t.Fatal(err)
			}
			request.Runtime.Seed = 0
			request.Runtime.Choice.Mode, request.Runtime.Choice.Tape = choice.ModeReplay, &plan
			replayed := runDiagnosticProfile(t, root, suite.name+"-diagnostic-replay", request, true, true)
			requireDiagnosticEvidence(t, first, replayed)
		}) {
			return
		}
	}
}

func TestDiagnosticCompiledIdentityOwners(t *testing.T) {
	owners := map[string]string{"wasi": ImplementationSHA256(), "volume": volume.ImplementationSHA256(), "network": netmodel.ImplementationSHA256(), "interrupted_replay": choice.InterruptedReplayImplementationSHA256(), "runner_clock": clock.BrokerImplementationSHA256(), "world_binary_snapshot": world.ImplementationSHA256()}
	for name, identity := range owners {
		if !strings.HasPrefix(identity, "sha256:") || len(identity) != len("sha256:")+64 {
			t.Fatalf("missing compiled owner identity: %s=%q", name, identity)
		}
	}
	root := os.Getenv("GOMAD_WASM_DIAGNOSTIC_IDENTITY_ROOT")
	if root != "" {
		if !filepath.IsAbs(root) {
			t.Fatal("identity capture requires absolute root")
		}
		if err := os.MkdirAll(root, 0700); err != nil {
			t.Fatal(err)
		}
		temporalJSON(t, filepath.Join(root, "compiled-owner-identities.json"), map[string]any{"compiled_go_version": runtime.Version(), "owners": owners})
	}
	t.Logf("compiled identity owners: %v", owners)
}

func requireDiagnosticEvidence(t *testing.T, before, after Result) {
	t.Helper()
	if difference := sameEvidence(before, after); difference != "" || before.Calls != after.Calls || before.TimeAdvances != after.TimeAdvances || before.EnvironmentIdentity != after.EnvironmentIdentity || before.FuelRemaining != after.FuelRemaining {
		t.Fatalf("profiling changed canonical execution: %s", difference)
	}
}

func profilePreparationEvent(t *testing.T, channel *os.File, event string, epoch int, request Request, cold bool) {
	t.Helper()
	data, err := json.Marshal(struct {
		Event        string `json:"event"`
		Epoch        int    `json:"epoch"`
		Helper       string `json:"helper"`
		HelperSHA256 string `json:"helper_sha256"`
		Cold         bool   `json:"cold"`
	}{event, epoch, request.HelperPath, request.HelperSHA256, cold})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := channel.Write(append(data, '\n')); err != nil {
		t.Fatal(err)
	}
}

func runDiagnosticProfile(t *testing.T, root, name string, request Request, instrumented, warm bool) Result {
	t.Helper()
	profile, err := os.Create(filepath.Join(root, name+"-parent.pprof"))
	if err != nil {
		t.Fatal(err)
	}
	if err := pprof.StartCPUProfile(profile); err != nil {
		_ = profile.Close()
		t.Fatal(err)
	}
	var memoryBefore, memoryAfter runtime.MemStats
	runtime.ReadMemStats(&memoryBefore)
	started := time.Now()
	result, runErr := matchedPrepareAdmitted(runtime.GOOS, "/proc/meminfo", os.Getenv("GOMAD_WASM_MATCHED_COLD_MEMORY_GUARD"), func() (Result, error) {
		return Run(t.Context(), request)
	})
	wall := time.Since(started).Nanoseconds()
	runtime.ReadMemStats(&memoryAfter)
	pprof.StopCPUProfile()
	if err := profile.Close(); err != nil {
		t.Fatal(err)
	}
	heap, err := os.Create(filepath.Join(root, name+"-parent-heap.pprof"))
	if err != nil {
		t.Fatal(err)
	}
	if err := pprof.WriteHeapProfile(heap); err != nil {
		_ = heap.Close()
		t.Fatal(err)
	}
	if err := heap.Close(); err != nil {
		t.Fatal(err)
	}
	temporalJSON(t, filepath.Join(root, name+"-parent-memory.json"), map[string]any{"diagnostic_only": true, "before": memoryBefore, "after": memoryAfter})
	temporalWrite(t, filepath.Join(root, name+"-helper-stderr.txt"), result.helperStderr)
	temporalWrite(t, filepath.Join(root, name+"-guest-stdout.txt"), result.Stdout)
	temporalWrite(t, filepath.Join(root, name+"-guest-stderr.txt"), result.Stderr)
	var diagnostic *diagnosticReport
	var diagnosticErr error
	if instrumented {
		var decoded diagnosticReport
		decoded, diagnosticErr = decodeDiagnosticReport(result.helperStderr)
		if diagnosticErr == nil {
			diagnostic = &decoded
		}
	}
	temporalJSON(t, filepath.Join(root, name+"-metrics.json"), map[string]any{"diagnostic_only": true, "compatibility_profile": "prototype-legacy-v1", "module_sha256": request.ModuleSHA256, "helper_sha256": request.HelperSHA256, "runtime_identity": request.Runtime.Choice.ExecutionIdentity.ToolchainBuildKey, "model_sha256": ImplementationSHA256(), "instrumented": instrumented, "run_wall_nanos": wall, "authorization_to_terminal_nanos": result.AuthorizationToTerminalNanos, "phase_metrics": result.Telemetry, "wait_resource_usage": result.HelperResourceUsage, "helper_profile": diagnostic, "scope_labels": map[string]string{"run_wall": "Go Run boundary; excludes diagnostics publication and pprof finalization", "phase_cpu": "helper RUSAGE_SELF samples within named phase; no parent CPU", "wait_cpu": "entire helper process including preparation", "nested_wall": "overlaps ABI and phase spans; never sum as disjoint work", "residual_wall": "unattributed Wasmtime/trampoline wall; never guest-only CPU", "peak_rss": "cumulative process high-water, not phase-local allocation"}})
	persistDiagnosticRawResult(t, root, name, result)
	if runErr != nil {
		t.Fatal(runErr)
	}
	requireExit(t, name, result, 0)
	if diagnosticErr != nil {
		t.Fatal(diagnosticErr)
	}
	if !instrumented && len(result.helperStderr) != 0 {
		t.Fatal("release-default helper emitted diagnostic bytes")
	}
	if result.Telemetry == nil || warm && (!result.Telemetry.CacheHit || result.Telemetry.CompileNanos != 0) {
		t.Fatal("profile phase metrics or warm cache contract")
	}
	return result
}

// This diagnostic is not a replacement for the unchanged-suite performance gate.
func TestHelperRuntimeCostProfile(t *testing.T) {
	root := os.Getenv("GOMAD_WASM_PROFILE_ROOT")
	if root == "" {
		t.Skip("set GOMAD_WASM_PROFILE_ROOT for actual runtime-loop profiling")
	}
	if !filepath.IsAbs(root) {
		t.Fatal("profile evidence root must be absolute")
	}
	if err := os.MkdirAll(root, 0700); err != nil {
		t.Fatal(err)
	}
	request := cooperativeRequest(t, "./testdata/runtime_cost")
	request.Config.Clock.ReadStepNanos = 0
	request.Config.Limits.Calls = 1 << 20
	request.Config.Limits.TranscriptBytes = 64 << 20
	request.Runtime.Diagnostics = false
	request.Runtime.Choice.Limit = 64 << 20
	request.Fuel = 1 << 40
	request.Timeout = 3 * time.Minute
	module, err := os.ReadFile(request.ModulePath)
	if err != nil {
		t.Fatal(err)
	}
	temporalWrite(t, filepath.Join(root, "runtime-cost.wasm"), module)
	var reference Result
	for _, owner := range []string{"go", "helper"} {
		request.remoteModel = owner == "go"
		profile, err := os.Create(filepath.Join(root, owner+"-parent.pprof"))
		if err != nil {
			t.Fatal(err)
		}
		if err := pprof.StartCPUProfile(profile); err != nil {
			_ = profile.Close()
			t.Fatal(err)
		}
		started := time.Now()
		result, runErr := Run(t.Context(), request)
		wall := time.Since(started).Nanoseconds()
		pprof.StopCPUProfile()
		if err := profile.Close(); err != nil {
			t.Fatal(err)
		}
		if runErr != nil {
			t.Fatal(runErr)
		}
		requireExit(t, owner, result, 0)
		if !bytes.Contains(result.Stdout, []byte("runtime-cost 100000")) || result.AuthorizationToTerminalNanos == 0 || result.ChoiceTrace.Summary.SelectPoll < 100000 {
			t.Fatalf("runtime loop witness missing: stdout=%q summary=%+v boundary=%d", result.Stdout, result.ChoiceTrace.Summary, result.AuthorizationToTerminalNanos)
		}
		if owner == "go" {
			reference = result
		} else if difference := sameEvidence(reference, result); difference != "" {
			t.Fatalf("runtime-cost profiling evidence differs: %s", difference)
		}
		temporalWrite(t, filepath.Join(root, owner+"-choice.bin"), result.ChoiceTrace.Bytes)
		temporalJSON(t, filepath.Join(root, owner+"-metrics.json"), map[string]any{
			"diagnostic_only": true, "profile": "current-runtime-go-owner-reference",
			"module_sha256": request.ModuleSHA256, "helper_sha256": request.HelperSHA256,
			"runtime_identity": request.Runtime.Choice.ExecutionIdentity.ToolchainBuildKey,
			"model_sha256":     ImplementationSHA256(), "wall_nanos": wall,
			"authorization_to_terminal_nanos": result.AuthorizationToTerminalNanos,
			"phase_metrics":                   result.Telemetry, "wait_resource_usage": result.HelperResourceUsage,
			"choices": result.ChoiceTrace.Summary,
		})
	}
}

func diagnosticFixture(t *testing.T) []byte {
	t.Helper()
	phases := map[string]any{}
	for _, name := range strings.Fields("module_read module_hash engine_init cache_identity cache_read envelope_verify deserialize compile cache_write instantiate execute") {
		phases[name] = map[string]uint64{"count": 0, "wall_nanos": 0, "user_nanos": 0, "system_nanos": 0, "cpu_samples": 0, "peak_rss_bytes_high_water": 0}
	}
	phases["execute"] = map[string]uint64{"count": 1, "wall_nanos": 100, "user_nanos": 70, "system_nanos": 10, "cpu_samples": 1, "peak_rss_bytes_high_water": 1024}
	abi := map[string]any{}
	for _, name := range strings.Fields("clock scheduler select observation idle other_local remote_callback") {
		abi[name] = map[string]uint64{"count": 0, "wall_nanos": 0, "instantiate_wall_nanos": 0, "execute_wall_nanos": 0}
	}
	abi["clock"] = map[string]uint64{"count": 1, "wall_nanos": 30, "instantiate_wall_nanos": 0, "execute_wall_nanos": 30}
	nested := map[string]any{}
	for _, name := range strings.Fields("alternative_hash encoding transcript_hash flush remote_write remote_read_wait") {
		nested[name] = map[string]uint64{"count": 0, "wall_nanos": 0}
	}
	data, err := json.Marshal(map[string]any{"schema": "gomad3.wasm-profile/v1", "diagnostic_only": true, "backend": "cranelift-jit", "host_os": "linux", "host_arch": "aarch64", "phases": phases, "abi": abi, "nested": nested, "residual": map[string]uint64{"instantiate_wall_nanos": 0, "execute_wall_nanos": 70}, "abi_overlap_errors": 0})
	if err != nil {
		t.Fatal(err)
	}
	return append(append([]byte("gomad-wasm-profile-v1 "), data...), '\n')
}

func TestDiagnosticReportRejectsMalformedOrUnattributedMeasurements(t *testing.T) {
	valid := diagnosticFixture(t)
	report, err := decodeDiagnosticReport(valid)
	if err != nil || report.Phases["execute"].UserNanos != 70 || report.ABI["clock"].ExecuteWallNanos != 30 || report.Residual.ExecuteWallNanos != 70 {
		t.Fatalf("literal phase profile: report=%+v err=%v", report, err)
	}
	for _, test := range []struct {
		name string
		data []byte
	}{
		{"missing", nil},
		{"oversized", bytes.Repeat([]byte{'x'}, 16<<10+1)},
		{"duplicate-report", append(append([]byte(nil), valid...), valid...)},
		{"trailing", append(append([]byte(nil), valid...), []byte("unexpected\n")...)},
		{"null", []byte("gomad-wasm-profile-v1 null\n")},
		{"wrong-schema", bytes.Replace(valid, []byte("gomad3.wasm-profile/v1"), []byte("gomad3.wasm-profile/v2"), 1)},
		{"unknown-top", bytes.Replace(valid, []byte(`"schema":`), []byte(`"extra":0,"schema":`), 1)},
		{"nested-duplicate", bytes.Replace(valid, []byte(`"user_nanos":70`), []byte(`"user_nanos":1,"user_nanos":70`), 1)},
		{"escaped-duplicate", bytes.Replace(valid, []byte(`"user_nanos":70`), []byte(`"\u0075ser_nanos":1,"user_nanos":70`), 1)},
		{"missing-phase", bytes.Replace(valid, []byte(`"module_read":`), []byte(`"absent":`), 1)},
		{"missing-measurement", bytes.Replace(valid, []byte(`"user_nanos":70,`), nil, 1)},
		{"unknown-measurement", bytes.Replace(valid, []byte(`"user_nanos":70`), []byte(`"user_nanos":70,"invented":0`), 1)},
		{"not-diagnostic", bytes.Replace(valid, []byte(`"diagnostic_only":true`), []byte(`"diagnostic_only":false`), 1)},
		{"not-jit", bytes.Replace(valid, []byte("cranelift-jit"), []byte("interpreter"), 1)},
		{"negative", bytes.Replace(valid, []byte(`"user_nanos":70`), []byte(`"user_nanos":-1`), 1)},
		{"too-many-cpu-samples", bytes.Replace(valid, []byte(`"cpu_samples":1`), []byte(`"cpu_samples":2`), 1)},
		{"overlapping-abi", bytes.Replace(valid, []byte(`"abi_overlap_errors":0`), []byte(`"abi_overlap_errors":1`), 1)},
		{"wrong-residual", bytes.Replace(valid, []byte(`"execute_wall_nanos":70`), []byte(`"execute_wall_nanos":71`), 1)},
	} {
		t.Run(test.name, func(t *testing.T) {
			if _, err := decodeDiagnosticReport(test.data); err == nil {
				t.Fatalf("accepted %s diagnostic", test.name)
			}
		})
	}
}

type diagnosticPhase struct {
	Count                 uint64 `json:"count"`
	WallNanos             uint64 `json:"wall_nanos"`
	UserNanos             uint64 `json:"user_nanos"`
	SystemNanos           uint64 `json:"system_nanos"`
	CPUSamples            uint64 `json:"cpu_samples"`
	PeakRSSBytesHighWater uint64 `json:"peak_rss_bytes_high_water"`
}

type diagnosticABI struct {
	Count                uint64 `json:"count"`
	WallNanos            uint64 `json:"wall_nanos"`
	InstantiateWallNanos uint64 `json:"instantiate_wall_nanos"`
	ExecuteWallNanos     uint64 `json:"execute_wall_nanos"`
}

type diagnosticNested struct {
	Count     uint64 `json:"count"`
	WallNanos uint64 `json:"wall_nanos"`
}

type diagnosticResidual struct {
	InstantiateWallNanos uint64 `json:"instantiate_wall_nanos"`
	ExecuteWallNanos     uint64 `json:"execute_wall_nanos"`
}

type diagnosticReport struct {
	Schema           string                      `json:"schema"`
	DiagnosticOnly   bool                        `json:"diagnostic_only"`
	Backend          string                      `json:"backend"`
	HostOS           string                      `json:"host_os"`
	HostArch         string                      `json:"host_arch"`
	Phases           map[string]diagnosticPhase  `json:"phases"`
	ABI              map[string]diagnosticABI    `json:"abi"`
	Nested           map[string]diagnosticNested `json:"nested"`
	Residual         diagnosticResidual          `json:"residual"`
	ABIOverlapErrors uint64                      `json:"abi_overlap_errors"`
}

func decodeDiagnosticReport(data []byte) (diagnosticReport, error) {
	var report diagnosticReport
	prefix := []byte("gomad-wasm-profile-v1 ")
	if len(data) > 16<<10 || !bytes.HasPrefix(data, prefix) || !bytes.HasSuffix(data, []byte{'\n'}) || bytes.Count(data, []byte{'\n'}) != 1 {
		return report, fmt.Errorf("missing, oversized or multiple diagnostic report")
	}
	data = data[len(prefix) : len(data)-1]
	if err := validateJSON(data); err != nil {
		return report, err
	}
	fields, err := diagnosticFields(data, "schema diagnostic_only backend host_os host_arch phases abi nested residual abi_overlap_errors")
	if err != nil {
		return report, err
	}
	for _, group := range []struct{ name, keys, measurements string }{
		{"phases", "module_read module_hash engine_init cache_identity cache_read envelope_verify deserialize compile cache_write instantiate execute", "count wall_nanos user_nanos system_nanos cpu_samples peak_rss_bytes_high_water"},
		{"abi", "clock scheduler select observation idle other_local remote_callback", "count wall_nanos instantiate_wall_nanos execute_wall_nanos"},
		{"nested", "alternative_hash encoding transcript_hash flush remote_write remote_read_wait", "count wall_nanos"},
	} {
		entries, err := diagnosticFields(fields[group.name], group.keys)
		if err != nil {
			return report, err
		}
		for _, entry := range entries {
			if _, err := diagnosticFields(entry, group.measurements); err != nil {
				return report, err
			}
		}
	}
	if _, err := diagnosticFields(fields["residual"], "instantiate_wall_nanos execute_wall_nanos"); err != nil {
		return report, err
	}
	if err := json.Unmarshal(data, &report); err != nil {
		return report, err
	}
	if report.Schema != "gomad3.wasm-profile/v1" || !report.DiagnosticOnly || report.Backend != "cranelift-jit" || report.HostOS != "linux" && report.HostOS != "macos" || report.HostArch != "aarch64" && report.HostArch != "x86_64" || report.ABIOverlapErrors != 0 {
		return report, fmt.Errorf("diagnostic identity or non-overlap claim")
	}
	for _, phase := range report.Phases {
		if phase.CPUSamples > phase.Count || phase.CPUSamples == 0 && (phase.UserNanos != 0 || phase.SystemNanos != 0) {
			return report, fmt.Errorf("diagnostic CPU sample claim")
		}
	}
	instantiate, execute := report.Residual.InstantiateWallNanos, report.Residual.ExecuteWallNanos
	for _, bucket := range report.ABI {
		if bucket.InstantiateWallNanos > math.MaxUint64-bucket.ExecuteWallNanos || bucket.InstantiateWallNanos+bucket.ExecuteWallNanos > bucket.WallNanos || bucket.InstantiateWallNanos > math.MaxUint64-instantiate || bucket.ExecuteWallNanos > math.MaxUint64-execute {
			return report, fmt.Errorf("diagnostic ABI wall overflow or attribution")
		}
		instantiate += bucket.InstantiateWallNanos
		execute += bucket.ExecuteWallNanos
	}
	if instantiate != report.Phases["instantiate"].WallNanos || execute != report.Phases["execute"].WallNanos {
		return report, fmt.Errorf("diagnostic residual attribution")
	}
	return report, nil
}

func diagnosticFields(data []byte, keys string) (map[string]json.RawMessage, error) {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return nil, err
	}
	names := strings.Fields(keys)
	if len(fields) != len(names) {
		return nil, fmt.Errorf("diagnostic field inventory")
	}
	for _, name := range names {
		if len(fields[name]) == 0 || bytes.Equal(fields[name], []byte("null")) {
			return nil, fmt.Errorf("missing diagnostic field %s", name)
		}
	}
	return fields, nil
}

type canonicalOperationCounts struct {
	Calls      uint64            `json:"calls"`
	Local      uint64            `json:"local"`
	Remote     uint64            `json:"remote"`
	Operations map[string]uint64 `json:"remote_operations"`
}

func countCanonicalRemoteOperations(data []byte, maximum uint64) (canonicalOperationCounts, error) {
	counts := canonicalOperationCounts{Operations: map[string]uint64{}}
	if uint64(len(data)) > maximum {
		return counts, invalid("retained canonical byte bound")
	}
	records, err := DecodeTranscript(data)
	if err != nil {
		return counts, err
	}
	for _, record := range records {
		var fields map[string]json.RawMessage
		if err := json.Unmarshal(record.Input, &fields); err != nil || fields == nil {
			return counts, invalid("retained input object")
		}
		fd := "no-root-fd"
		if raw, ok := fields["fd"]; ok {
			var value uint32
			if bytes.Equal(raw, []byte("null")) || json.Unmarshal(raw, &value) != nil {
				return counts, invalid("retained fd type")
			}
			fd = fmt.Sprintf("fd=%d", value)
		}
		local := false
		switch record.Op {
		case "clock_time_get", "sched_yield", "random_get", "args_sizes_get", "args_get", "environ_sizes_get", "environ_get":
			local = true
		case "poll_oneoff":
			var subscriptions []struct {
				Type uint8 `json:"type"`
			}
			if err := json.Unmarshal(fields["subscriptions"], &subscriptions); err != nil {
				return counts, invalid("retained subscriptions")
			}
			local = true
			for _, subscription := range subscriptions {
				if subscription.Type != 0 {
					local = false
					break
				}
			}
		}
		counts.Calls += record.Count
		if local {
			counts.Local += record.Count
		} else {
			counts.Remote += record.Count
			counts.Operations["wasi_snapshot_preview1/"+record.Op+"/"+fd] += record.Count
		}
	}
	return counts, nil
}

func operationCountRecord(id uint64, tag byte, body string) string {
	// Independent fixture framing; payloads below are hand-derived from protocol.md.
	data := []byte{tag, 0, 0, 0}
	data = binary.LittleEndian.AppendUint64(data, id)
	data = binary.LittleEndian.AppendUint32(data, uint32(len(body)))
	return string(append(data, body...))
}

func TestCanonicalRemoteOperationCounts(t *testing.T) {
	clock := "\x00\x00\x00\x00" + strings.Repeat("\x00", 8) + "\x01" + strings.Repeat("\x00", 7) + "\x02" + strings.Repeat("\x00", 7)
	clockPoll := "\x01\x00\x00\x00" + strings.Repeat("\x00", 35)
	mixedPoll := "\x02\x00\x00\x00" + strings.Repeat("\x00", 39) + "\x01\x04\x00\x00\x00" + strings.Repeat("\x00", 4)
	data := []byte("GMWT\x01\x00\x00\x00" + operationCountRecord(1, 33, clock) + operationCountRecord(3, 2, "") + operationCountRecord(4, 32, clockPoll) + operationCountRecord(5, 32, mixedPoll) + operationCountRecord(6, 8, "\x01"+strings.Repeat("\x00", 11)) + operationCountRecord(7, 8, "\x02"+strings.Repeat("\x00", 11)) + operationCountRecord(8, 9, "\x04"+strings.Repeat("\x00", 19)) + operationCountRecord(9, 26, "\x03"+strings.Repeat("\x00", 35)))
	original := bytes.Clone(data)
	got, err := countCanonicalRemoteOperations(data, uint64(len(data)))
	if err != nil || got.Calls != 9 || got.Local != 4 || got.Remote != 5 || got.Operations["wasi_snapshot_preview1/poll_oneoff/no-root-fd"] != 1 || got.Operations["wasi_snapshot_preview1/fd_write/fd=1"] != 1 || got.Operations["wasi_snapshot_preview1/fd_write/fd=2"] != 1 || got.Operations["wasi_snapshot_preview1/fd_pwrite/fd=4"] != 1 || got.Operations["wasi_snapshot_preview1/path_open/fd=3"] != 1 {
		t.Fatalf("actual binary operation counts = %+v, %v", got, err)
	}
	if !bytes.Equal(original, data) {
		t.Fatal("counter mutated binary input")
	}
}

func TestCanonicalRemoteOperationCountsRejectsInvalid(t *testing.T) {
	valid := "GMWT\x01\x00\x00\x00" + operationCountRecord(1, 8, "\x01"+strings.Repeat("\x00", 11))
	for name, data := range map[string]string{
		"truncated":        valid[:len(valid)-1],
		"trailing":         valid + "\x00",
		"nonincreasing":    valid + valid[8:],
		"unknown tag":      "GMWT\x01\x00\x00\x00" + operationCountRecord(1, 255, ""),
		"oversize payload": "GMWT\x01\x00\x00\x00" + operationCountRecord(1, 8, "\x01\x00\x00\x00\xff\xff\xff\xff"),
		"oversize array":   "GMWT\x01\x00\x00\x00" + operationCountRecord(1, 32, "\xff\xff\xff\xff"),
		"retired JSONL":    `{"type":"call","id":1,"op":"sched_yield","input":{}}` + "\n",
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := countCanonicalRemoteOperations([]byte(data), uint64(len(data))); err == nil {
				t.Fatal("invalid retained binary trace accepted")
			}
		})
	}
	if _, err := countCanonicalRemoteOperations([]byte(valid), uint64(len(valid)-1)); err == nil {
		t.Fatal("retained byte bound ignored")
	}
}

func TestCanonicalRemoteOperationClassification(t *testing.T) {
	for _, sample := range []struct {
		tag   byte
		body  string
		local bool
	}{
		{2, "", true}, {3, "", true}, {4, "", true}, {5, "", true}, {6, "", true}, {7, strings.Repeat("\x00", 12), true},
		{32, strings.Repeat("\x00", 8), true},
		{32, "\x01\x00\x00\x00" + strings.Repeat("\x00", 8) + "\x01\x04\x00\x00\x00" + strings.Repeat("\x00", 4), false},
		{13, "\x04\x00\x00\x00", false},
	} {
		data := []byte("GMWT\x01\x00\x00\x00" + operationCountRecord(1, sample.tag, sample.body))
		counts, err := countCanonicalRemoteOperations(data, uint64(len(data)))
		if err != nil || (counts.Local == 1) != sample.local || counts.Local+counts.Remote != 1 {
			t.Fatalf("tag=%d = %+v, %v", sample.tag, counts, err)
		}
	}
}

func TestHelperCanonicalOperationRetentionDiagnostic(t *testing.T) {
	root := os.Getenv("GOMAD_WASM_OPERATION_COUNT_ROOT")
	if root == "" {
		t.Skip("counts diagnostic requires an exclusive guest slot")
	}
	control := os.Getenv("GOMAD_WASM_OPERATION_CONTROL_ROOT")
	module, cache, runtimeIdentity := os.Getenv("GOMAD_WASM_MATCHED_MODULE"), os.Getenv("GOMAD_WASM_MATCHED_CACHE"), os.Getenv("GOMAD_WASM_MATCHED_TOOLCHAIN_ID")
	for _, path := range []string{root, control, module, cache} {
		if !filepath.IsAbs(path) {
			t.Fatal("counts diagnostic requires absolute paths")
		}
	}
	if runtimeIdentity == "" {
		t.Fatal("explicit legacy runtime required")
	}
	if err := os.MkdirAll(root, 0700); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(module)
	if err != nil {
		t.Fatal(err)
	}
	imports, pages := binaryInventory(t, data)
	repo, err := filepath.Abs("../../../../../..")
	if err != nil {
		t.Fatal(err)
	}
	identity := choice.ExecutionIdentity{TargetSHA256: sha256.Sum256(data), ToolchainBuildKey: runtimeIdentity, GOOS: "wasip1", GOARCH: "wasm", ImplementationSHA256: sha256.Sum256([]byte(runtimeIdentity))}
	for _, suite := range []struct{ name, run string }{
		{"workflow", "^TestWorkflowTypeEncodingSuite$/^TestPlainASCII$/^Succeeds$"},
		{"child", "^TestChildWorkflowSuite$"}, {"timers", "^TestUserTimersTestSuite$"},
	} {
		for _, seed := range []uint64{0, 1000003} {
			name := suite.name + "-seed" + stringSeed(seed)
			if !t.Run(name, func(t *testing.T) {
				config := prototypeTemporalConfig(t, repo, "/workspace/tests", "-test.run="+suite.run, "-persistenceType=sql", "-persistenceDriver=sqlite")
				config.Profile, config.Clock.ReadStepNanos = CooperativeProfile, 0
				helper := helperPath(t)
				request := Request{HelperPath: helper, HelperSHA256: fileHash(t, helper), ModulePath: module, ModuleSHA256: fileHash(t, module), Engine: PinnedEngine(), Imports: imports, InitialMemoryPages: pages, MemoryBytes: 4 << 30, Fuel: 1 << 50, Config: config, Timeout: 5 * time.Minute, CompileCache: cache, DigestTranscript: false, Runtime: &RuntimeControl{Seed: seed, Choice: &runnerbackend.ChoiceRequest{Mode: choice.ModeRecord, ExecutionIdentity: identity, Limit: choice.MaximumTraceBytes}}}
				reference := suite.name + "-reference-seed" + stringSeed(seed)
				var historical Result
				readDiagnosticJSON(t, filepath.Join(control, reference+"-observed-result.json"), &historical)
				var metadata struct {
					Helper  string `json:"helper_sha256"`
					Module  string `json:"module_sha256"`
					Model   string `json:"model_sha256"`
					Runtime string `json:"runtime_identity"`
				}
				readDiagnosticJSON(t, filepath.Join(control, reference+"-metrics.json"), &metadata)
				if metadata.Helper != request.HelperSHA256 || metadata.Module != request.ModuleSHA256 || metadata.Model != ImplementationSHA256() || metadata.Runtime != runtimeIdentity {
					t.Fatal("current control identities mismatch")
				}
				var before, retained Result
				for _, owner := range []string{"control", "retained"} {
					current := request
					current.DigestTranscript = owner == "control"
					started := time.Now()
					result, runErr := matchedPrepareAdmitted(runtime.GOOS, "/proc/meminfo", os.Getenv("GOMAD_WASM_MATCHED_COLD_MEMORY_GUARD"), func() (Result, error) { return Run(t.Context(), current) })
					persistOperationDiagnostic(t, root, name+"-"+owner, current, result, time.Since(started).Nanoseconds())
					if runErr != nil {
						t.Fatal(runErr)
					}
					requireExit(t, name+"-"+owner, result, 0)
					if result.Telemetry == nil || !result.Telemetry.CacheHit || result.Telemetry.CompileNanos != 0 {
						t.Fatal("warm diagnostic cache miss; not accepted as warm")
					}
					if owner == "control" {
						before = result
					} else {
						retained = result
					}
				}
				requireDiagnosticEvidence(t, before, retained)
				if difference := historicalJSONPublicProjectionDifference(historical, before); difference != "" {
					t.Fatalf("historical JSON PUBLIC projection changed (no historical private-terminal proof): %s", difference)
				}
				counts, err := countCanonicalRemoteOperations(retained.Transcript, 64<<20)
				if err != nil {
					t.Fatal(err)
				}
				var diagnostic struct {
					Profile struct {
						ABI struct {
							Remote struct {
								Count uint64 `json:"count"`
							} `json:"remote_callback"`
						} `json:"abi"`
					} `json:"helper_profile"`
				}
				readDiagnosticJSON(t, filepath.Join(control, suite.name+"-diagnostic-seed"+stringSeed(seed)+"-metrics.json"), &diagnostic)
				if counts.Remote != diagnostic.Profile.ABI.Remote.Count {
					t.Fatalf("actual remote classification=%d diagnostic callback count=%d", counts.Remote, diagnostic.Profile.ABI.Remote.Count)
				}
				temporalJSON(t, filepath.Join(root, name+"-operation-counts.json"), counts)
			}) {
				return
			}
		}
	}
}

func readDiagnosticJSON(t *testing.T, path string, target any) {
	t.Helper()
	data, err := hostfs.ReadBounded(path, 64<<20)
	if err != nil {
		t.Fatal(err)
	}
	if err := validateJSON(data); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, target); err != nil {
		t.Fatal(err)
	}
}

func persistOperationDiagnostic(t *testing.T, root, name string, request Request, result Result, wall int64) {
	t.Helper()
	runtimeIdentity := ""
	if request.Runtime != nil && request.Runtime.Choice != nil {
		runtimeIdentity = request.Runtime.Choice.ExecutionIdentity.ToolchainBuildKey
	}
	temporalJSON(t, filepath.Join(root, name+"-metrics.json"), map[string]any{"diagnostic_only": true, "retention_mode": !request.DigestTranscript, "retained_limit_scope": "64MiB postexecution only; configured transport reservation unchanged", "module_sha256": request.ModuleSHA256, "helper_sha256": request.HelperSHA256, "model_sha256": ImplementationSHA256(), "runtime_identity": runtimeIdentity, "run_wall_nanos": wall, "authorization_to_terminal_nanos": result.AuthorizationToTerminalNanos, "phase_metrics": result.Telemetry, "wait_resource_usage": result.HelperResourceUsage, "termination": result.Termination, "message": result.Message, "retained_bytes": len(result.Transcript)})
	persistDiagnosticRawResult(t, root, name, result)
}

func TestOperationDiagnosticPersistsPrivateTerminalAndRawEvidence(t *testing.T) {
	root := t.TempDir()
	result := Result{Stdout: []byte("stdout"), Transcript: []byte("canonical\n"), runtimeTerminal: [16]byte{1, 2, 3, 4}}
	result.ChoiceTrace.Bytes = []byte("choice")
	persistDiagnosticRawResult(t, root, "record", result)
	for name, expected := range map[string][]byte{"finish.bin": result.runtimeTerminal[:], "canonical.bin": result.Transcript, "choice.bin": result.ChoiceTrace.Bytes} {
		actual, err := os.ReadFile(filepath.Join(root, "record-"+name))
		if err != nil || !bytes.Equal(actual, expected) {
			t.Fatalf("raw %s not preserved: %q, %v", name, actual, err)
		}
	}
	var stored Result
	readDiagnosticJSON(t, filepath.Join(root, "record-observed-result.json"), &stored)
	if !bytes.Equal(stored.Transcript, result.Transcript) || !bytes.Equal(stored.ChoiceTrace.Bytes, result.ChoiceTrace.Bytes) {
		t.Fatal("full exported result lost evidence")
	}
	if sameEvidence(result, stored) != "terminal" {
		t.Fatal("JSON unexpectedly claims private terminal preservation")
	}
	terminal, err := os.ReadFile(filepath.Join(root, "record-finish.bin"))
	if err != nil {
		t.Fatal(err)
	}
	copy(stored.runtimeTerminal[:], terminal)
	if sameEvidence(result, stored) != "" {
		t.Fatal("actual terminal sidecar failed exact roundtrip")
	}
}

func persistDiagnosticRawResult(t *testing.T, root, name string, result Result) {
	t.Helper()
	if len(result.Transcript) > 64<<20 {
		t.Fatal("postexecution retained transcript capacity exceeded; actual costs retained")
	}
	temporalWrite(t, filepath.Join(root, name+"-canonical.bin"), result.Transcript)
	temporalWrite(t, filepath.Join(root, name+"-choice.bin"), result.ChoiceTrace.Bytes)
	temporalWrite(t, filepath.Join(root, name+"-finish.bin"), result.runtimeTerminal[:])
	temporalJSON(t, filepath.Join(root, name+"-observed-result.json"), result)
}

func historicalJSONPublicProjectionDifference(historical, actual Result) string {
	actual.runtimeTerminal = [16]byte{}
	if difference := sameEvidence(historical, actual); difference != "" {
		return difference
	}
	if historical.Calls != actual.Calls || historical.TimeAdvances != actual.TimeAdvances || historical.EnvironmentIdentity != actual.EnvironmentIdentity || historical.FuelRemaining != actual.FuelRemaining || historical.VirtualTimeElapsedNanos != actual.VirtualTimeElapsedNanos {
		return "public counters or identity"
	}
	return ""
}

func TestHistoricalJSONPublicProjectionRejectsChangedEvidence(t *testing.T) {
	exit := uint32(0)
	actual := Result{Termination: "exit", ExitCode: &exit, Stdout: []byte("stdout"), Stderr: []byte("stderr"), NetworkTranscript: []byte("network"), TranscriptSHA256: [32]byte{1}, Calls: 7, TimeAdvances: 2, EnvironmentIdentity: "model", FuelRemaining: 13, VirtualTimeElapsedNanos: 17, runtimeTerminal: [16]byte{1, 2, 3}}
	actual.ChoiceTrace.Bytes = []byte("choices")
	data, err := json.Marshal(actual)
	if err != nil {
		t.Fatal(err)
	}
	var historical Result
	if err := json.Unmarshal(data, &historical); err != nil {
		t.Fatal(err)
	}
	if sameEvidence(historical, actual) != "terminal" {
		t.Fatal("historical JSON falsely preserved private terminal")
	}
	if difference := historicalJSONPublicProjectionDifference(historical, actual); difference != "" {
		t.Fatalf("equal public projection rejected: %s", difference)
	}
	for name, mutate := range map[string]func(*Result){
		"stdout": func(r *Result) { r.Stdout = []byte("changed") }, "stderr": func(r *Result) { r.Stderr = []byte("changed") },
		"wasi": func(r *Result) { r.TranscriptSHA256[0]++ }, "choice": func(r *Result) { r.ChoiceTrace.Bytes = []byte("changed") },
		"network": func(r *Result) { r.NetworkTranscript = []byte("changed") }, "termination": func(r *Result) { r.Termination = "invalid" },
		"exit": func(r *Result) { code := uint32(1); r.ExitCode = &code }, "calls": func(r *Result) { r.Calls++ },
		"advances": func(r *Result) { r.TimeAdvances++ }, "environment": func(r *Result) { r.EnvironmentIdentity = "changed" },
		"fuel": func(r *Result) { r.FuelRemaining++ }, "virtual": func(r *Result) { r.VirtualTimeElapsedNanos++ },
	} {
		t.Run(name, func(t *testing.T) {
			changed := actual
			mutate(&changed)
			if historicalJSONPublicProjectionDifference(historical, changed) == "" {
				t.Fatal("changed public evidence accepted")
			}
		})
	}
	if actual.runtimeTerminal != [16]byte{1, 2, 3} || historical.runtimeTerminal != [16]byte{} {
		t.Fatal("projection mutated original terminal evidence")
	}
}

const diagnosticHostCPUInfoLimit = 1 << 20

func diagnosticPerformanceHost(goos, goarch, cpuinfo string) bool {
	if goos != "linux" || goarch != "arm64" || len(cpuinfo) > diagnosticHostCPUInfoLimit {
		return false
	}
	var architecture, asimd bool
	for line := range strings.SplitSeq(cpuinfo, "\n") {
		key, value, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		switch strings.TrimSpace(key) {
		case "CPU architecture":
			architecture = architecture || strings.TrimSpace(value) == "8"
		case "Features":
			asimd = asimd || slices.Contains(strings.Fields(value), "asimd")
		}
	}
	return architecture && asimd
}

func diagnosticPerformanceHostEvidence(path string) (string, bool) {
	cpuinfo, err := hostfs.ReadBounded(path, diagnosticHostCPUInfoLimit)
	if err != nil {
		return "", false
	}
	return string(cpuinfo), diagnosticPerformanceHost(runtime.GOOS, runtime.GOARCH, string(cpuinfo))
}

func TestDiagnosticPerformanceHost(t *testing.T) {
	const armCPU = "processor: 0\nFeatures: fp asimd aes crc32\nCPU architecture: 8\n"
	const x86CPU = "processor: 0\nvendor_id: GenuineIntel\nflags: fpu sse sse2\n"
	for _, test := range []struct {
		name, goos, goarch, cpuinfo string
		want                        bool
	}{
		{"Linux arm64", "linux", "arm64", armCPU, true},
		{"Darwin arm64", "darwin", "arm64", armCPU, false},
		{"Linux amd64", "linux", "amd64", x86CPU, false},
		{"emulated arm64 on x86", "linux", "arm64", x86CPU, false},
		{"missing host witness", "linux", "arm64", "", false},
		{"32-bit ARM", "linux", "arm64", "Features: fp neon\nCPU architecture: 8\n", false},
		{"oversized host witness", "linux", "arm64", armCPU + strings.Repeat("\n", diagnosticHostCPUInfoLimit), false},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := diagnosticPerformanceHost(test.goos, test.goarch, test.cpuinfo); got != test.want {
				t.Fatalf("performance host accepted=%t, want %t", got, test.want)
			}
		})
	}
}

func TestDiagnosticPerformanceHostEvidenceBounds(t *testing.T) {
	const armCPU = "Features: fp asimd\nCPU architecture: 8\n"
	for _, size := range []int{diagnosticHostCPUInfoLimit, diagnosticHostCPUInfoLimit + 1} {
		path := filepath.Join(t.TempDir(), "cpuinfo")
		data := armCPU + strings.Repeat("\n", size-len(armCPU))
		if err := os.WriteFile(path, []byte(data), 0600); err != nil {
			t.Fatal(err)
		}
		cpuinfo, admitted := diagnosticPerformanceHostEvidence(path)
		if size == diagnosticHostCPUInfoLimit {
			if cpuinfo != data {
				t.Fatal("host witness at the byte bound was lost")
			}
		} else if cpuinfo != "" || admitted {
			t.Fatal("oversized host witness was retained or admitted")
		}
	}
}

func TestDiagnosticPerformanceCurrentHostAdmission(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "arm64" {
		t.Skip("current-host admission requires Linux/arm64")
	}
	cpuinfo, admitted := diagnosticPerformanceHostEvidence("/proc/cpuinfo")
	if cpuinfo == "" || !admitted {
		t.Fatal("current Linux/arm64 host CPU witness was not admitted")
	}
}

type variantEvidence struct {
	Choice, Network, Stdout, Stderr []byte
	WASI                            []TranscriptRecord
	Args, Environment               []string
	EntropyKey                      [32]byte
	Clock                           ClockPolicy
	Termination, Message            string
	ExitCode                        *uint32
	VirtualNanos                    uint64
}

func projectVariantEvidence(result Result, config Config, gcVariant string) (variantEvidence, error) {
	transcript, err := DecodeTranscript(result.Transcript)
	if err != nil {
		return variantEvidence{}, err
	}
	if gcVariant != "" {
		config.Environment = slices.DeleteFunc(slices.Clone(config.Environment), func(value string) bool {
			name, _, _ := strings.Cut(value, "=")
			return name == "GOGC" || gcVariant == "off" && name == "GOMEMLIMIT"
		})
		for index := range transcript {
			if transcript[index].Op == "environ_get" || transcript[index].Op == "environ_sizes_get" {
				transcript[index].Output = json.RawMessage(`{}`)
			}
		}
	}
	return variantEvidence{Choice: result.ChoiceTrace.Bytes, WASI: transcript, Args: slices.Clone(config.Args), Environment: slices.Clone(config.Environment), EntropyKey: config.EntropyKey, Clock: config.Clock, Network: result.NetworkTranscript, Stdout: result.Stdout, Stderr: result.Stderr, Termination: result.Termination, Message: result.Message, ExitCode: result.ExitCode, VirtualNanos: result.VirtualTimeElapsedNanos}, nil
}

func variantIdentities(request Request, result Result) map[string]string {
	engine := request.Engine.Configuration
	if request.diagnosticConfigurationRevision != "" {
		engine = request.diagnosticConfigurationRevision
	}
	identities := map[string]string{"module_sha256": request.ModuleSHA256, "helper_sha256": request.HelperSHA256, "environment_identity": result.EnvironmentIdentity, "engine_revision": engine, "model_sha256": ImplementationSHA256()}
	if request.Runtime != nil && request.Runtime.Choice != nil {
		identities["runtime_identity"] = request.Runtime.Choice.ExecutionIdentity.ToolchainBuildKey
	}
	return identities
}

type variantTimingRange struct {
	Median uint64 `json:"median"`
	Min    uint64 `json:"min"`
	Max    uint64 `json:"max"`
}

func variantTiming(values []uint64) variantTimingRange {
	if len(values) == 0 {
		return variantTimingRange{}
	}
	ordered := slices.Clone(values)
	slices.Sort(ordered)
	return variantTimingRange{Median: ordered[len(ordered)/2], Min: ordered[0], Max: ordered[len(ordered)-1]}
}

func variantRecommendation(baseline, candidate []uint64, failed, identical bool) string {
	if failed || !identical {
		return "reject"
	}
	if len(baseline) == 0 || len(candidate) == 0 {
		return "investigate"
	}
	before, after := variantTiming(baseline), variantTiming(candidate)
	if after.Median >= before.Median {
		return "reject"
	}
	if before.Median-after.Median > max(before.Max-before.Min, after.Max-after.Min) {
		return "adopt"
	}
	return "investigate"
}

func diagnosticRunRoot(base, prefix string) (string, error) {
	if err := os.MkdirAll(base, 0700); err != nil {
		return "", err
	}
	return os.MkdirTemp(base, prefix)
}

func TestDiagnosticGuestVariants(t *testing.T) {
	root := os.Getenv("GOMAD_WASM_GUEST_PROFILE_ROOT")
	if root == "" {
		t.Skip("set GOMAD_WASM_GUEST_PROFILE_ROOT for unsampled Temporal variant measurements")
	}
	helper, fuelHelper := os.Getenv("GOMAD_WASM_TEST_HELPER"), os.Getenv("GOMAD_WASM_FUEL_OFF_HELPER")
	for _, path := range []string{root, helper, fuelHelper} {
		if !filepath.IsAbs(path) {
			t.Fatal("variant root, diagnostic-profile helper and diagnostic-fuel-off helper must be absolute")
		}
	}
	if fileHash(t, helper) == fileHash(t, fuelHelper) {
		t.Fatal("fuel-off helper must have a distinct identity")
	}
	repo, err := filepath.Abs("../../../../../..")
	if err != nil {
		t.Fatal(err)
	}
	relative, err := filepath.Rel(repo, root)
	if err != nil || relative != ".." && !strings.HasPrefix(relative, ".."+string(os.PathSeparator)) {
		t.Fatal("raw variant evidence must be retained outside the repository")
	}
	baseRoot := root
	root, err = diagnosticRunRoot(baseRoot, "guest-variants-")
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("retaining variant evidence in %s", root)
	admission := newDiagnosticAdmission(t, repo, root)
	admission.addHelper(t, helper, os.Getenv("GOMAD_WASM_DIAGNOSTIC_HELPER_ADMISSION"), []string{"diagnostic-profile"})
	admission.addHelper(t, fuelHelper, os.Getenv("GOMAD_WASM_FUEL_OFF_HELPER_ADMISSION"), []string{"diagnostic-fuel-off", "diagnostic-profile"})
	t.Setenv("GOMAD_WASM_PROTOTYPE_ROOT", root)
	for _, directory := range []string{"modules", "variants"} {
		if err := os.MkdirAll(filepath.Join(root, directory), 0700); err != nil {
			t.Fatal(err)
		}
	}
	module, optimized := filepath.Join(root, "modules", "variants-baseline.wasm"), filepath.Join(root, "modules", "variants-satconv-signext.wasm")
	for _, build := range []struct{ module, gowasm string }{{module, ""}, {optimized, "satconv,signext"}} {
		admission.buildModule(t, build.module, build.gowasm)
	}
	t.Setenv("GOWASM", "")
	commit, err := exec.Command("git", "-C", repo, "rev-parse", "HEAD").Output()
	if err != nil {
		t.Fatal(err)
	}
	status, err := exec.Command("git", "-C", repo, "status", "--porcelain").Output()
	if err != nil {
		t.Fatal(err)
	}
	seeds := []uint64{0, 1000003, 2000006}
	const repetitions = 5
	variants := []struct{ name, gc, module, helper, revision, followup string }{
		{"baseline", "", module, helper, "", "none; diagnostic control"},
		{"gc200", "200", module, helper, "", "explicit GC environment and memory policy decision; requalify affected selections"},
		{"gc400", "400", module, helper, "", "explicit GC environment and memory policy decision; requalify affected selections"},
		{"gc-off", "off", module, helper, "", "bounded GOMEMLIMIT policy and GC decision; requalify affected selections"},
		{"gowasm", "", optimized, helper, "", "production preparation currently forces GOWASM=; adopting requires preparation and build identity changes plus requalification"},
		{"fuel-off", "", module, fuelHelper, PinnedEngine().Configuration + "-fuel-off", "explicit fuel/resource containment and engine identity decision; diagnostic helper is not a release configuration"},
	}
	evidence := filepath.Join(root, "variants")
	runAdmitted := func(name string, request Request, warm bool) (Result, diagnosticReport, string) {
		admission.check(t, evidence, name+"-before", request, true)
		defer admission.check(t, evidence, name+"-after", request, false)
		return runGuestVariant(t, evidence, name, request, warm)
	}
	var reports, recommendations []map[string]any
	for _, selection := range []struct{ name, run string }{
		{"workflow", "^TestWorkflowTypeEncodingSuite$/^TestPlainASCII$/^Succeeds$"},
		{"activity", "^TestActivityTestSuite$"},
	} {
		requests := make([]Request, len(variants))
		warmFailures := make([]string, len(variants))
		for index, variant := range variants {
			config := prototypeTemporalConfig(t, repo, "/workspace/tests", "-test.run="+selection.run, "-persistenceType=sql", "-persistenceDriver=sqlite")
			request := prototypeRequest(t, variant.module, config, 0, 30*time.Minute)
			request.HelperPath, request.HelperSHA256 = variant.helper, fileHash(t, variant.helper)
			request.diagnosticConfigurationRevision = variant.revision
			request.DigestTranscript = false
			if variant.gc != "" {
				request.Config.Environment = append(request.Config.Environment, "GOGC="+variant.gc)
				if variant.gc == "off" {
					request.Config.Environment = append(request.Config.Environment, "GOMEMLIMIT="+strconv.FormatUint(request.MemoryBytes/4*3, 10)+"B")
				}
			}
			requests[index] = request
			warm := request
			warm.Config.Args = slices.Clone(request.Config.Args)
			for argIndex, argument := range warm.Config.Args {
				if strings.HasPrefix(argument, "-test.run=") {
					warm.Config.Args[argIndex] = "-test.run=^$"
				}
			}
			_, _, warmFailures[index] = runAdmitted(selection.name+"-"+variant.name+"-prepare", warm, false)
		}
		decisions := make([][]string, len(variants))
		for _, seed := range seeds {
			execute, residual := make([][]uint64, len(variants)), make([][]uint64, len(variants))
			failed, identical := make([]bool, len(variants)), make([]bool, len(variants))
			for index := range identical {
				identical[index] = true
			}
			var reference variantEvidence
			var baselineResult Result
			var baselineSeen bool
			var runs []map[string]any
			for repetition := 0; repetition < repetitions; repetition++ {
				// Rotate candidates to reduce a fixed ordering bias; baseline runs first.
				for slot := range variants {
					index := 0
					if slot > 0 {
						index = 1 + (slot-1+repetition)%(len(variants)-1)
					}
					variant, request := variants[index], requests[index]
					request.Runtime.Seed = seed
					name := fmt.Sprintf("%s-%s-seed%d-rep%d", selection.name, variant.name, seed, repetition)
					result, profile, failure := runAdmitted(name, request, true)
					if warmFailures[index] != "" {
						failure = "preparation: " + warmFailures[index] + "; " + failure
					}
					run := map[string]any{"variant": variant.name, "repetition": repetition, "identities": variantIdentities(request, result), "failure": failure, "termination": result.Termination, "exit_code": result.ExitCode, "memory_bytes": request.MemoryBytes, "peak_memory_bytes": result.PeakMemoryBytes, "environment": request.Config.Environment, "profile": profile, "status": "failed"}
					if failure != "" {
						failed[index] = true
					} else {
						projected, projectionErr := projectVariantEvidence(result, request.Config, variant.gc)
						if projectionErr != nil {
							failed[index], run["failure"] = true, projectionErr.Error()
						} else {
							if index == 0 && !baselineSeen {
								reference = projected
								baselineResult = result
								baselineSeen = true
							}
							before := reference
							if variant.gc != "" {
								before, projectionErr = projectVariantEvidence(baselineResult, requests[0].Config, variant.gc)
								if projectionErr != nil {
									t.Fatal(projectionErr)
								}
							}
							matched := baselineSeen && reflect.DeepEqual(before, projected)
							identical[index] = identical[index] && matched
							if matched {
								run["status"] = "identical"
							} else if !baselineSeen {
								run["status"] = "comparison-unavailable"
							} else {
								run["status"] = "evidence-changing"
							}
							data, err := json.Marshal(projected)
							if err != nil {
								t.Fatal(err)
							}
							run["behavior_projection_sha256"] = digest(data)
							execute[index] = append(execute[index], profile.Phases["execute"].WallNanos)
							residual[index] = append(residual[index], profile.Residual.ExecuteWallNanos)
						}
					}
					runs = append(runs, run)
				}
			}
			for index, variant := range variants {
				decision := variantRecommendation(execute[0], execute[index], failed[0] || failed[index], identical[0] && identical[index])
				if index == 0 {
					decision = "control"
				}
				decisions[index] = append(decisions[index], decision)
				reports = append(reports, map[string]any{"selection": selection.name, "test_run": selection.run, "seed": seed, "variant": variant.name, "warm_repetitions_requested": repetitions, "warm_repetitions_measured": len(execute[index]), "execute_nanos": variantTiming(execute[index]), "residual_nanos": variantTiming(residual[index]), "failed": failed[index], "evidence_identical": !failed[0] && !failed[index] && identical[0] && identical[index], "recommendation": decision})
			}
			temporalJSON(t, filepath.Join(evidence, fmt.Sprintf("%s-seed%d-runs.json", selection.name, seed)), runs)
		}
		for index, variant := range variants {
			decision := "adopt"
			if slices.Contains(decisions[index], "reject") {
				decision = "reject"
			} else if slices.Contains(decisions[index], "investigate") {
				decision = "investigate"
			}
			if index == 0 {
				decision = "control"
			}
			recommendations = append(recommendations, map[string]any{"selection": selection.name, "variant": variant.name, "recommendation": decision, "followup": variant.followup})
		}
	}
	cpuinfo, requiredHostMeasured := diagnosticPerformanceHostEvidence("/proc/cpuinfo")
	temporalJSON(t, filepath.Join(root, "guest-variants-report.json"), map[string]any{
		"host_admission": admission.Receipt, "host_admission_sha256": admission.Inputs[admission.Receipt], "module_build_receipts": admission.ModuleReceipts,
		"helper_admissions": map[string]string{helper: admission.Helpers[helper].Receipt, fuelHelper: admission.Helpers[fuelHelper].Receipt},
		"diagnostic_only":   true, "source_commit": strings.TrimSpace(string(commit)), "source_dirty": len(status) != 0,
		"host_goos": runtime.GOOS, "host_goarch": runtime.GOARCH, "go_version": runtime.Version(), "seeds": seeds, "warm_repetitions": repetitions,
		"required_host": "linux/arm64", "required_host_measured": requiredHostMeasured, "developmental": !requiredHostMeasured, "host_cpuinfo": cpuinfo,
		"measurements": reports, "recommendations": recommendations,
		"build_settings": map[string]string{"GOOS": "wasip1", "GOARCH": "wasm", "GOEXPERIMENT": "nogreenteagc", "baseline_GOWASM": "", "variant_GOWASM": "satconv,signext"},
		"scope":          "unsampled diagnostic-profile timing; full transcript retention for exact behavior comparisons; residual includes guest plus Wasmtime/trampoline work, not guest-only CPU",
		"equality":       "exact choice, WASI, network, stdout/stderr, outcome and virtual time; only GC environ reply outputs masked; module/environment/helper/engine identities separate",
		"decision_rule":  "adopt only when all seeds have identical successful evidence and median execute gain strictly exceeds both baseline and candidate run-to-run ranges; adoption remains a follow-up decision",
		"rerun":          map[string]any{"working_directory": filepath.Join(repo, "tools", "gomad"), "environment": map[string]string{"GOMAD_WASM_GUEST_PROFILE_ROOT": baseRoot, "GOMAD_WASM_TEST_HELPER": helper, "GOMAD_WASM_FUEL_OFF_HELPER": fuelHelper, "GOMAD_WASM_DIAGNOSTIC_HELPER_ADMISSION": admission.Helpers[helper].Receipt, "GOMAD_WASM_FUEL_OFF_HELPER_ADMISSION": admission.Helpers[fuelHelper].Receipt, "GOMAD3_STOCK_GO": os.Getenv("GOMAD3_STOCK_GO")}, "command": "env -u GOROOT GOENV=off GOTOOLCHAIN=local GOWORK=off go test -tags test_dep -count=1 -timeout 180m -run '^TestDiagnosticGuestVariants$' ./internal/backends/wasm/wasi"},
	})
}

func runGuestVariant(t *testing.T, root, name string, request Request, warm bool) (Result, diagnosticReport, string) {
	t.Helper()
	result, runErr := matchedPrepareAdmitted(runtime.GOOS, "/proc/meminfo", os.Getenv("GOMAD_WASM_MATCHED_COLD_MEMORY_GUARD"), func() (Result, error) {
		return Run(t.Context(), request)
	})
	temporalWrite(t, filepath.Join(root, name+"-canonical.bin"), result.Transcript)
	temporalWrite(t, filepath.Join(root, name+"-choice.bin"), result.ChoiceTrace.Bytes)
	temporalWrite(t, filepath.Join(root, name+"-network.jsonl"), result.NetworkTranscript)
	temporalWrite(t, filepath.Join(root, name+"-helper-stderr.txt"), result.helperStderr)
	temporalWrite(t, filepath.Join(root, name+"-guest-stdout.txt"), result.Stdout)
	temporalWrite(t, filepath.Join(root, name+"-guest-stderr.txt"), result.Stderr)
	profile, profileErr := decodeDiagnosticReport(result.helperStderr)
	if runErr != nil {
		return result, profile, runErr.Error()
	}
	if result.Termination != "exit" || result.ExitCode == nil || *result.ExitCode != 0 {
		return result, profile, fmt.Sprintf("termination=%s exit=%v message=%s", result.Termination, result.ExitCode, result.Message)
	}
	if profileErr != nil {
		return result, profile, profileErr.Error()
	}
	if result.Telemetry == nil || warm && (!result.Telemetry.CacheHit || result.Telemetry.CompileNanos != 0) {
		return result, profile, "missing telemetry or warm cache miss"
	}
	return result, profile, ""
}

var guestProfileBuckets = strings.Fields("gc scheduler allocation test server runtime stdlib other")

type firefoxGuestProfile struct {
	Meta struct {
		Interval float64              `json:"interval"`
		Guest    guestProfileSampling `json:"gomadGuestProfile"`
	} `json:"meta"`
	Threads []firefoxGuestThread `json:"threads"`
}

type guestProfileSampling struct {
	MaxSamples     uint64 `json:"maxSamples"`
	Samples        uint64 `json:"samples"`
	Truncated      bool   `json:"truncated"`
	IntervalMicros uint64 `json:"intervalMicros"`
}

type firefoxGuestThread struct {
	Samples struct {
		Length     int     `json:"length"`
		Stack      []*int  `json:"stack"`
		Weight     []int64 `json:"weight"`
		WeightType string  `json:"weightType"`
	} `json:"samples"`
	StackTable struct {
		Length int    `json:"length"`
		Frame  []int  `json:"frame"`
		Prefix []*int `json:"prefix"`
	} `json:"stackTable"`
	FrameTable struct {
		Length int   `json:"length"`
		Func   []int `json:"func"`
	} `json:"frameTable"`
	FuncTable struct {
		Length int   `json:"length"`
		Name   []int `json:"name"`
	} `json:"funcTable"`
	StringArray []string `json:"stringArray"`
}

type guestProfileAttribution struct {
	SampleRows       uint64            `json:"sample_rows"`
	GuestWeight      uint64            `json:"guest_sample_weight"`
	EmptyStackWeight uint64            `json:"empty_stack_weight"`
	Buckets          map[string]uint64 `json:"self_sample_weight"`
}

func guestFrameBucket(name string) string {
	name = strings.TrimPrefix(name, "$")
	for _, prefix := range strings.Fields("runtime.gcBgMarkWorker runtime.gcDrain runtime.scanobject runtime.markroot runtime.gcAssistAlloc runtime.bgsweep runtime.sweepone runtime.wbBufFlush runtime.gcWriteBarrier runtime.bulkBarrier runtime.writebarrier") {
		if strings.HasPrefix(name, prefix) {
			return "gc"
		}
	}
	for _, function := range strings.Fields("runtime.schedule runtime.findRunnable runtime.gopark runtime.park_m runtime.goready runtime.mcall runtime.gogo runtime.handleEvent runtime.beforeIdle wasm_pc_f_loop runtime.wasm_pc_f_loop") {
		if name == function || strings.HasPrefix(name, function+".") || strings.HasPrefix(name, function+"(") {
			return "scheduler"
		}
	}
	for _, prefix := range strings.Fields("runtime.mallocgc runtime.makeslice runtime.growslice runtime.newobject runtime.newarray runtime.makemap runtime.mapassign") {
		if strings.HasPrefix(name, prefix) {
			return "allocation"
		}
	}
	if strings.HasPrefix(name, "go.temporal.io/server/tests.") || strings.HasPrefix(name, "go.temporal.io/server/tests/") {
		return "test"
	}
	if strings.HasPrefix(name, "go.temporal.io/server/") {
		return "server"
	}
	if strings.HasPrefix(name, "runtime.") || strings.HasPrefix(name, "runtime/") || strings.HasPrefix(name, "internal/runtime/") {
		return "runtime"
	}
	if strings.Contains(name, ".") {
		first, _, _ := strings.Cut(name, "/")
		if !strings.Contains(name, "/") {
			first, _, _ = strings.Cut(name, ".")
		}
		if first != "" && first != "main" && !strings.ContainsAny(first, ". <>[]:") {
			return "stdlib"
		}
	}
	return ""
}

func guestStackBucket(leafToRoot []string) string {
	for _, name := range leafToRoot {
		if bucket := guestFrameBucket(name); bucket != "" {
			return bucket
		}
	}
	return "other"
}

func decodeGuestAttribution(data []byte) (firefoxGuestProfile, guestProfileAttribution, error) {
	var profile firefoxGuestProfile
	attribution := guestProfileAttribution{Buckets: make(map[string]uint64, len(guestProfileBuckets))}
	for _, bucket := range guestProfileBuckets {
		attribution.Buckets[bucket] = 0
	}
	if err := json.Unmarshal(data, &profile); err != nil {
		return profile, attribution, err
	}
	if len(profile.Threads) == 0 || profile.Meta.Interval <= 0 {
		return profile, attribution, fmt.Errorf("guest profile has no threads or sampling interval")
	}
	for threadIndex, thread := range profile.Threads {
		if thread.Samples.Length != len(thread.Samples.Stack) || thread.Samples.Weight != nil && len(thread.Samples.Weight) != thread.Samples.Length || thread.Samples.WeightType != "samples" || thread.StackTable.Length != len(thread.StackTable.Frame) || thread.StackTable.Length != len(thread.StackTable.Prefix) || thread.FrameTable.Length != len(thread.FrameTable.Func) || thread.FuncTable.Length != len(thread.FuncTable.Name) {
			return profile, attribution, fmt.Errorf("thread %d: inconsistent guest profile tables or weight type", threadIndex)
		}
		for sampleIndex, stack := range thread.Samples.Stack {
			weight := int64(1)
			if thread.Samples.Weight != nil {
				weight = thread.Samples.Weight[sampleIndex]
			}
			if weight < 0 {
				return profile, attribution, fmt.Errorf("thread %d sample %d: negative sample weight", threadIndex, sampleIndex)
			}
			attribution.SampleRows++
			var frames []string
			for depth := 0; stack != nil; depth++ {
				if *stack < 0 || *stack >= thread.StackTable.Length || depth >= thread.StackTable.Length {
					return profile, attribution, fmt.Errorf("thread %d sample %d: invalid or cyclic stack", threadIndex, sampleIndex)
				}
				frame := thread.StackTable.Frame[*stack]
				if frame < 0 || frame >= thread.FrameTable.Length {
					return profile, attribution, fmt.Errorf("thread %d: invalid frame index", threadIndex)
				}
				function := thread.FrameTable.Func[frame]
				if function < 0 || function >= thread.FuncTable.Length {
					return profile, attribution, fmt.Errorf("thread %d: invalid function index", threadIndex)
				}
				name := thread.FuncTable.Name[function]
				if name < 0 || name >= len(thread.StringArray) {
					return profile, attribution, fmt.Errorf("thread %d: invalid function name index", threadIndex)
				}
				frames = append(frames, thread.StringArray[name])
				stack = thread.StackTable.Prefix[*stack]
			}
			if len(frames) == 0 {
				if math.MaxUint64-attribution.EmptyStackWeight < uint64(weight) {
					return profile, attribution, fmt.Errorf("empty stack weight overflow")
				}
				attribution.EmptyStackWeight += uint64(weight)
				continue
			}
			if math.MaxUint64-attribution.GuestWeight < uint64(weight) {
				return profile, attribution, fmt.Errorf("guest sample weight overflow")
			}
			attribution.GuestWeight += uint64(weight)
			attribution.Buckets[guestStackBucket(frames)] += uint64(weight)
		}
	}
	return profile, attribution, nil
}

func TestDiagnosticGuestProfile(t *testing.T) {
	root := os.Getenv("GOMAD_WASM_GUEST_PROFILE_ROOT")
	if root == "" {
		t.Skip("set GOMAD_WASM_GUEST_PROFILE_ROOT for Temporal guest stack profiling")
	}
	if !filepath.IsAbs(root) || !filepath.IsAbs(os.Getenv("GOMAD_WASM_TEST_HELPER")) {
		t.Fatal("guest profile root and GOMAD_WASM_TEST_HELPER must be absolute")
	}
	repo, err := filepath.Abs("../../../../../..")
	if err != nil {
		t.Fatal(err)
	}
	relative, err := filepath.Rel(repo, root)
	if err != nil || relative != ".." && !strings.HasPrefix(relative, ".."+string(os.PathSeparator)) {
		t.Fatal("raw guest profiles must be retained outside the repository")
	}
	baseRoot := root
	root, err = diagnosticRunRoot(baseRoot, "guest-profile-")
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("retaining guest profiles in %s", root)
	admission := newDiagnosticAdmission(t, repo, root)
	helper := os.Getenv("GOMAD_WASM_TEST_HELPER")
	admission.addHelper(t, helper, os.Getenv("GOMAD_WASM_GUEST_HELPER_ADMISSION"), []string{"diagnostic-profile", "guest-profile"})
	t.Setenv("GOMAD_WASM_PROTOTYPE_ROOT", root)
	for _, directory := range []string{"modules", "evidence"} {
		if err := os.MkdirAll(filepath.Join(root, directory), 0700); err != nil {
			t.Fatal(err)
		}
	}
	module := filepath.Join(root, "modules", "tests.wasm")
	admission.buildModule(t, module, "")
	evidence := filepath.Join(root, "evidence")
	runAdmitted := func(name string, request Request, warm bool) Result {
		admission.check(t, evidence, name+"-before", request, true)
		defer admission.check(t, evidence, name+"-after", request, false)
		return runDiagnosticProfile(t, evidence, name, request, true, warm)
	}
	persistence := []string{"-persistenceType=sql", "-persistenceDriver=sqlite"}
	warm := prototypeRequest(t, module, prototypeTemporalConfig(t, repo, "/workspace/tests", append([]string{"-test.run=^$"}, persistence...)...), 0, 30*time.Minute)
	warm.diagnosticConfigurationRevision = PinnedEngine().Configuration + "-guest-profile"
	runAdmitted("guest-profile-prepare", warm, false)
	commit, err := exec.Command("git", "-C", repo, "rev-parse", "HEAD").Output()
	if err != nil {
		t.Fatal(err)
	}
	status, err := exec.Command("git", "-C", repo, "status", "--porcelain").Output()
	if err != nil {
		t.Fatal(err)
	}
	var runs []map[string]any
	for _, selection := range []struct{ name, run string }{
		{"workflow", "^TestWorkflowTypeEncodingSuite$/^TestPlainASCII$/^Succeeds$"},
		{"activity", "^TestActivityTestSuite$"},
	} {
		for _, seed := range []uint64{0, 1000003, 2000006} {
			name := selection.name + "-seed" + stringSeed(seed)
			config := prototypeTemporalConfig(t, repo, "/workspace/tests", append([]string{"-test.run=" + selection.run}, persistence...)...)
			request := prototypeRequest(t, module, config, seed, 30*time.Minute)
			request.diagnosticConfigurationRevision = warm.diagnosticConfigurationRevision
			before := runAdmitted(name+"-unsampled", request, true)
			request.guestProfile = &guestProfileConfig{Path: filepath.Join(evidence, name+"-guest.firefox.json"), IntervalMicros: 1000, MaxSamples: 10000}
			after := runAdmitted(name+"-sampled", request, true)
			requireDiagnosticEvidence(t, before, after)
			if before.VirtualTimeElapsedNanos != after.VirtualTimeElapsedNanos || !bytes.Equal(before.DiagnosticTrace.Bytes, after.DiagnosticTrace.Bytes) {
				t.Fatal("profiling changed virtual time or runtime diagnostics")
			}
			data, err := os.ReadFile(request.guestProfile.Path)
			if err != nil {
				t.Fatal(err)
			}
			profile, attribution, err := decodeGuestAttribution(data)
			if err != nil {
				t.Fatal(err)
			}
			if attribution.GuestWeight == 0 || request.guestProfile.MaxSamples != profile.Meta.Guest.MaxSamples || request.guestProfile.IntervalMicros != profile.Meta.Guest.IntervalMicros || profile.Meta.Guest.Samples > profile.Meta.Guest.MaxSamples || profile.Meta.Guest.Samples != attribution.SampleRows {
				t.Fatalf("profile samples or metadata mismatch: metadata=%+v attribution=%+v", profile.Meta.Guest, attribution)
			}
			run := map[string]any{
				"selection": selection.name, "test_run": selection.run, "seed": seed,
				"helper_sha256": request.HelperSHA256, "module_sha256": request.ModuleSHA256,
				"engine_revision":  request.diagnosticConfigurationRevision,
				"runtime_identity": request.Runtime.Choice.ExecutionIdentity.ToolchainBuildKey,
				"model_sha256":     ImplementationSHA256(), "evidence_identical": true,
				"profile": request.guestProfile.Path, "profile_sha256": digest(data),
				"sampling": profile.Meta.Guest, "attribution": attribution,
			}
			temporalJSON(t, filepath.Join(evidence, name+"-attribution.json"), run)
			runs = append(runs, run)
			t.Logf("%s: weighted guest samples=%d buckets=%v truncated=%t", name, attribution.GuestWeight, attribution.Buckets, profile.Meta.Guest.Truncated)
		}
	}
	cpuinfo, requiredHostMeasured := diagnosticPerformanceHostEvidence("/proc/cpuinfo")
	temporalJSON(t, filepath.Join(root, "guest-attribution-report.json"), map[string]any{
		"host_admission": admission.Receipt, "host_admission_sha256": admission.Inputs[admission.Receipt], "module_build_receipts": admission.ModuleReceipts,
		"helper_admission": admission.Helpers[helper].Receipt,
		"diagnostic_only":  true, "source_commit": strings.TrimSpace(string(commit)), "source_dirty": len(status) != 0,
		"host_goos": runtime.GOOS, "host_goarch": runtime.GOARCH, "go_version": runtime.Version(), "runs": runs,
		"required_host": "linux/arm64", "required_host_measured": requiredHostMeasured, "developmental": !requiredHostMeasured, "host_cpuinfo": cpuinfo,
		"scope":  "weighted guest stack samples; epoch function-entry/loop bias; host-call ticks excluded; no timestamp-gap charge; sampled wall times are not baselines",
		"limits": "scheduler-frame share does not quantify all unwind/rewind work; fuel checks are distributed across ordinary guest frames",
		"rerun": map[string]any{
			"working_directory": filepath.Join(repo, "tools", "gomad"),
			"environment": map[string]string{
				"GOMAD_WASM_GUEST_PROFILE_ROOT":     baseRoot,
				"GOMAD_WASM_TEST_HELPER":            os.Getenv("GOMAD_WASM_TEST_HELPER"),
				"GOMAD_WASM_GUEST_HELPER_ADMISSION": admission.Helpers[helper].Receipt,
				"GOMAD3_STOCK_GO":                   os.Getenv("GOMAD3_STOCK_GO"),
			},
			"command": "env -u GOROOT GOENV=off GOTOOLCHAIN=local GOWORK=off go test -tags test_dep -count=1 -timeout 60m -run '^TestDiagnosticGuestProfile$' ./internal/backends/wasm/wasi",
		},
	})
}

func TestDiagnosticGuestFrameBuckets(t *testing.T) {
	for _, test := range []struct{ bucket, names string }{
		{"gc", "runtime.gcBgMarkWorker runtime.gcDrainN runtime.scanobject runtime.markrootBlock runtime.gcAssistAlloc1 runtime.bgsweep runtime.sweepone runtime.wbBufFlush1 runtime.gcWriteBarrier runtime.bulkBarrierPreWrite runtime.writebarrierptr"},
		{"scheduler", "runtime.schedule runtime.findRunnable runtime.gopark runtime.park_m runtime.goready runtime.mcall runtime.gogo runtime.handleEvent runtime.beforeIdle wasm_pc_f_loop runtime.wasm_pc_f_loop"},
		{"allocation", "runtime.mallocgcSmallScanNoHeader runtime.makeslice runtime.growslice runtime.newobject runtime.newarray runtime.makemap runtime.mapassign"},
		{"test", "go.temporal.io/server/tests.(*ActivityTestSuite).TestActivity go.temporal.io/server/tests/testcore.NewTestServer"},
		{"server", "go.temporal.io/server/service/history.(*Engine).Execute"},
		{"runtime", "runtime.memmove runtime/atomic.Store internal/runtime/maps.(*Map).get"},
		{"stdlib", "encoding/json.Marshal net/http.(*Client).Do testing.tRunner reflect.Value.Call sync.(*Mutex).Lock"},
		{"other", "github.com/stretchr/testify/require.NoError main.main type:.eq.go.temporal.io/server.Task <wasm-function[1]>"},
	} {
		for _, name := range strings.Fields(test.names) {
			t.Run(name, func(t *testing.T) {
				if got := guestStackBucket([]string{name}); got != test.bucket {
					t.Fatalf("bucket=%s, want %s", got, test.bucket)
				}
			})
		}
	}
	for _, test := range []struct {
		frames []string
		want   string
	}{
		{[]string{"runtime.mallocgc", "runtime.gcAssistAlloc", "go.temporal.io/server/tests.Test"}, "allocation"},
		{[]string{"runtime.gcDrain", "runtime.schedule"}, "gc"},
		{[]string{"runtime.schedule", "runtime.gcDrain"}, "scheduler"},
		{[]string{"runtime.schedule.func1", "go.temporal.io/server/tests.Test"}, "scheduler"},
		{[]string{"go.temporal.io/server/tests.Test", "runtime.gcDrain"}, "test"},
		{[]string{"runtime.memmove", "go.temporal.io/server/tests.Test"}, "runtime"},
		{[]string{"<wasm-function[1]>", "runtime.gcDrain"}, "gc"},
		{nil, "other"},
	} {
		if got := guestStackBucket(test.frames); got != test.want {
			t.Fatalf("frames %v: bucket=%s, want %s", test.frames, got, test.want)
		}
	}
}

func diagnosticGuestFixture() []byte {
	return []byte(`{
 "meta":{"interval":1},
 "threads":[{
  "samples":{"length":4,"stack":[1,0,2,null],"weight":[3,2,5,7],"weightType":"samples","timeDeltas":[0,100000,1,100000],"threadCPUDelta":[0,100000000,0,100000000]},
  "stackTable":{"length":3,"frame":[2,0,1],"prefix":[null,0,0]},
  "frameTable":{"length":3,"func":[1,2,0]},
  "funcTable":{"length":3,"name":[2,0,1]},
  "stringArray":["runtime.gcDrain","runtime.schedule","go.temporal.io/server/tests.Test"]
 },{
  "samples":{"length":1,"stack":[0],"weight":null,"weightType":"samples"},
  "stackTable":{"length":1,"frame":[0],"prefix":[null]},
  "frameTable":{"length":1,"func":[0]},
  "funcTable":{"length":1,"name":[0]},
  "stringArray":["encoding/json.Marshal"]
 }]
}`)
}

func TestDiagnosticGuestAttributionWeightedThreadTables(t *testing.T) {
	_, attribution, err := decodeGuestAttribution(diagnosticGuestFixture())
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]uint64{"gc": 3, "scheduler": 5, "allocation": 0, "test": 2, "server": 0, "runtime": 0, "stdlib": 1, "other": 0}
	if attribution.SampleRows != 5 || attribution.GuestWeight != 11 || attribution.EmptyStackWeight != 7 || !maps.Equal(want, attribution.Buckets) {
		t.Fatalf("weighted attribution=%+v, want rows=5 guest=11 empty=7 buckets=%v", attribution, want)
	}
}

func TestDiagnosticGuestAttributionRejectsMalformedTables(t *testing.T) {
	for _, test := range []struct{ name, before, after string }{
		{"sample length", `"length":4`, `"length":3`},
		{"weight length", `[3,2,5,7]`, `[3,2,5]`},
		{"negative weight", `[3,2,5,7]`, `[-3,2,5,7]`},
		{"weight overflow", `[3,2,5,7]`, `[9223372036854775807,9223372036854775807,5,7]`},
		{"allocation weight", `"weightType":"samples"`, `"weightType":"bytes"`},
		{"stack length", `"frame":[2,0,1]`, `"frame":[2,0]`},
		{"prefix length", `"prefix":[null,0,0]`, `"prefix":[null,0]`},
		{"stack index", `"stack":[1,0,2,null]`, `"stack":[3,0,2,null]`},
		{"negative stack", `"stack":[1,0,2,null]`, `"stack":[-1,0,2,null]`},
		{"cycle", `"prefix":[null,0,0]`, `"prefix":[1,0,0]`},
		{"frame index", `"frame":[2,0,1]`, `"frame":[2,3,1]`},
		{"function index", `"func":[1,2,0]`, `"func":[3,2,0]`},
		{"name index", `"name":[2,0,1]`, `"name":[3,0,1]`},
		{"no interval", `"interval":1`, `"interval":0`},
	} {
		t.Run(test.name, func(t *testing.T) {
			data := strings.Replace(string(diagnosticGuestFixture()), test.before, test.after, 1)
			_, _, err := decodeGuestAttribution([]byte(data))
			if err == nil {
				t.Fatal("malformed profile accepted")
			}
		})
	}
	for _, data := range []string{`{}`, `{`, `{"meta":{"interval":1},"threads":[]}`} {
		_, _, err := decodeGuestAttribution([]byte(data))
		if err == nil {
			t.Fatal("malformed profile accepted")
		}
	}
}

func TestDiagnosticRunRootPreservesEarlierInvocation(t *testing.T) {
	base := t.TempDir()
	first, err := diagnosticRunRoot(base, "guest-profile-")
	if err != nil {
		t.Fatal(err)
	}
	marker := filepath.Join(first, "seed0-guest.firefox.json")
	if err := os.WriteFile(marker, []byte("first profile"), 0600); err != nil {
		t.Fatal(err)
	}
	second, err := diagnosticRunRoot(base, "guest-profile-")
	if err != nil {
		t.Fatal(err)
	}
	if first == second || filepath.Dir(first) != base || filepath.Dir(second) != base {
		t.Fatalf("invocations share their evidence directory: first=%s second=%s", first, second)
	}
	file, err := os.OpenFile(filepath.Join(second, "seed0-guest.firefox.json"), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		t.Fatal("fresh invocation cannot create its profile", err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(marker)
	if err != nil || string(data) != "first profile" {
		t.Fatal("earlier profile changed", err)
	}
}

func TestDiagnosticVariantArtifactSuffixes(t *testing.T) {
	root := t.TempDir()
	_, _, failure := runGuestVariant(t, root, "invalid-request", Request{}, false)
	if failure == "" {
		t.Fatal("invalid request unexpectedly executed")
	}
	files, err := os.ReadDir(root)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, file := range files {
		names = append(names, file.Name())
	}
	want := []string{
		"invalid-request-canonical.bin", "invalid-request-choice.bin",
		"invalid-request-guest-stderr.txt", "invalid-request-guest-stdout.txt",
		"invalid-request-helper-stderr.txt", "invalid-request-network.jsonl",
	}
	if !reflect.DeepEqual(names, want) {
		t.Fatalf("artifact names = %v, want %v", names, want)
	}
}

func TestDiagnosticVariantEvidencePreservesDerivedEnvironmentRecords(t *testing.T) {
	config := testConfig()
	config.Environment = append(config.Environment, "GOGC=200")
	baseline, variant := newTestEnvironment(t, testConfig()), newTestEnvironment(t, config)
	for _, environment := range []*Environment{baseline, variant} {
		call(t, environment, "environ_sizes_get", `{}`)
		call(t, environment, "environ_get", `{}`)
		call(t, environment, "args_get", `{}`)
	}
	before := Result{Transcript: baseline.Transcript()}
	after := Result{Transcript: variant.Transcript()}
	unmasked, err := projectVariantEvidence(after, config, "")
	if err != nil {
		t.Fatal(err)
	}
	want, err := projectVariantEvidence(before, testConfig(), "")
	if err != nil || reflect.DeepEqual(want, unmasked) {
		t.Fatal("unmasked derived environment change was lost", err)
	}
	want, err = projectVariantEvidence(before, testConfig(), "200")
	if err != nil {
		t.Fatal(err)
	}
	got, err := projectVariantEvidence(after, config, "200")
	if err != nil || !reflect.DeepEqual(want, got) {
		t.Fatal("GC environment was not masked", err)
	}
	for _, replacement := range []struct {
		name   string
		offset int
		value  byte
	}{
		{"environment operation", 8, 3},
		{"environment errno", 10, 1},
		{"argument operation", 40, 2},
		{"call id", 44, 9},
	} {
		changed := after
		changed.Transcript = bytes.Clone(after.Transcript)
		changed.Transcript[replacement.offset] = replacement.value
		projection, err := projectVariantEvidence(changed, config, "200")
		if err != nil || reflect.DeepEqual(want, projection) {
			t.Fatalf("mask hid non-environment change: %+v", replacement)
		}
	}
}

func TestDiagnosticVariantEvidencePreservesClockRuns(t *testing.T) {
	transcript := []byte("GMWT\x01\x00\x00\x00\x21\x00\x00\x00\x01\x00\x00\x00\x00\x00\x00\x00\x1c\x00\x00\x00\x01\x00\x00\x00\x07\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x03\x00\x00\x00\x00\x00\x00\x00")
	for _, mask := range []bool{false, true} {
		gcVariant := ""
		if mask {
			gcVariant = "200"
		}
		want, err := projectVariantEvidence(Result{Transcript: transcript}, testConfig(), gcVariant)
		if err != nil {
			t.Fatal(err)
		}
		for _, change := range []struct {
			name   string
			offset int
			value  byte
		}{
			{"id", 12, 2}, {"clock id", 24, 0}, {"precision", 28, 6}, {"position", 36, 1}, {"count", 44, 4},
		} {
			t.Run(change.name+"/mask="+strconv.FormatBool(mask), func(t *testing.T) {
				data := bytes.Clone(transcript)
				data[change.offset] = change.value
				got, err := projectVariantEvidence(Result{Transcript: data}, testConfig(), gcVariant)
				if err != nil || reflect.DeepEqual(want, got) {
					t.Fatal("clock evidence change was hidden", err)
				}
			})
		}
	}
}

func TestDiagnosticVariantEvidencePreservesInputsAndFailures(t *testing.T) {
	for _, test := range []struct {
		name         string
		tag          byte
		input        string
		inputOffsets []int
	}{
		{"random_get", 7, "\x02\x00\x00\x00\x04\x00\x00\x00\x00\x00\x00\x00", []int{0, 4}},
		{"fd_write", 8, "\x01\x00\x00\x00\x02\x00\x00\x00ab", []int{0, 9}},
		{"unsupported", 34, "\x03\x00\x00\x00bad\x0b\x00\x00\x00" + `{"value":1}`, []int{2 + 4, 20}},
		{"environ_get", 6, "", nil},
	} {
		t.Run(test.name, func(t *testing.T) {
			transcript := []byte("GMWT\x01\x00\x00\x00" + operationCountRecord(1, test.tag, test.input+"\x07\x00\x00\x00refused"))
			transcript[9] = 1
			want, err := projectVariantEvidence(Result{Transcript: transcript}, testConfig(), "200")
			if err != nil {
				t.Fatal(err)
			}
			for _, offset := range []int{10, len(transcript) - 1} {
				data := bytes.Clone(transcript)
				data[offset] ^= 1
				got, err := projectVariantEvidence(Result{Transcript: data}, testConfig(), "200")
				if err != nil || reflect.DeepEqual(want, got) {
					t.Fatal("failure evidence was hidden", err)
				}
			}
			for _, offset := range test.inputOffsets {
				data := bytes.Clone(transcript)
				data[len(transcriptHeader)+transcriptRecordHeader+offset] ^= 1
				got, err := projectVariantEvidence(Result{Transcript: data}, testConfig(), "200")
				if err != nil || reflect.DeepEqual(want, got) {
					t.Fatal("operation input was hidden", err)
				}
			}
		})
	}
}

func TestDiagnosticVariantEvidencePreservesOutputs(t *testing.T) {
	transcript := []byte("GMWT\x01\x00\x00\x00" + operationCountRecord(1, 8, "\x01\x00\x00\x00\x02\x00\x00\x00ab\x02\x00\x00\x00"))
	want, err := projectVariantEvidence(Result{Transcript: transcript}, testConfig(), "200")
	if err != nil {
		t.Fatal(err)
	}
	transcript[len(transcript)-4] = 1
	got, err := projectVariantEvidence(Result{Transcript: transcript}, testConfig(), "200")
	if err != nil || reflect.DeepEqual(want, got) {
		t.Fatal("non-environment output change was hidden", err)
	}
}

func TestDiagnosticVariantEvidencePreservesUnsupportedFlag(t *testing.T) {
	before := []byte("GMWT\x01\x00\x00\x00" + operationCountRecord(1, 6, ""))
	after := []byte("GMWT\x01\x00\x00\x00" + operationCountRecord(1, 6, "\x00\x00\x00\x00"))
	after[9] = 1
	want, err := projectVariantEvidence(Result{Transcript: before}, testConfig(), "200")
	if err != nil {
		t.Fatal(err)
	}
	got, err := projectVariantEvidence(Result{Transcript: after}, testConfig(), "200")
	if err != nil || reflect.DeepEqual(want, got) {
		t.Fatal("environment unsupported flag was hidden", err)
	}
}

func TestDiagnosticVariantEvidenceRefusesMalformedBinary(t *testing.T) {
	valid := []byte("GMWT\x01\x00\x00\x00\x06\x00\x00\x00\x01\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00")
	unknown := bytes.Clone(valid)
	unknown[8] = 255
	oversized := bytes.Clone(valid)
	binary.LittleEndian.PutUint32(oversized[20:], frameLimit)
	for _, transcript := range [][]byte{
		[]byte("{\"type\":\"call\",\"id\":1,\"op\":\"environ_get\",\"input\":{}}\n{\"type\":\"reply\",\"id\":1,\"errno\":0,\"output\":{}}\n"),
		valid[:len(valid)-1], append(bytes.Clone(valid), 0), unknown, oversized,
	} {
		for _, mask := range []bool{false, true} {
			gcVariant := ""
			if mask {
				gcVariant = "200"
			}
			if _, err := projectVariantEvidence(Result{Transcript: transcript}, testConfig(), gcVariant); err == nil {
				t.Fatalf("malformed binary transcript accepted with mask=%v: %x", mask, transcript)
			}
		}
	}
}

func TestDiagnosticVariantEvidenceSeparatesIdentities(t *testing.T) {
	code := uint32(0)
	before := Result{Termination: "exit", ExitCode: &code, Stdout: []byte("out"), Stderr: []byte("err"), Transcript: []byte{}, NetworkTranscript: []byte("network"), EnvironmentIdentity: "baseline", VirtualTimeElapsedNanos: 12}
	before.ChoiceTrace.Bytes = []byte("choice")
	after := before
	after.EnvironmentIdentity, after.FuelRemaining = "variant", 99
	baseRequest := Request{ModuleSHA256: "base-module", HelperSHA256: "base-helper", Engine: PinnedEngine()}
	variantRequest := Request{ModuleSHA256: "variant-module", HelperSHA256: "variant-helper", Engine: PinnedEngine(), diagnosticConfigurationRevision: PinnedEngine().Configuration + "-fuel-off"}
	want, err := projectVariantEvidence(before, testConfig(), "")
	if err != nil {
		t.Fatal(err)
	}
	got, err := projectVariantEvidence(after, testConfig(), "")
	if err != nil || !reflect.DeepEqual(want, got) {
		t.Fatal("identity or fuel accounting entered behavior equality", err)
	}
	if reflect.DeepEqual(variantIdentities(baseRequest, before), variantIdentities(variantRequest, after)) {
		t.Fatal("variant identities were not reported separately")
	}
	for _, field := range []string{"module_sha256", "helper_sha256", "environment_identity", "engine_revision"} {
		t.Run(field, func(t *testing.T) {
			request, result := baseRequest, before
			switch field {
			case "module_sha256":
				request.ModuleSHA256 = "other-module"
			case "helper_sha256":
				request.HelperSHA256 = "other-helper"
			case "environment_identity":
				result.EnvironmentIdentity = "other-environment"
			case "engine_revision":
				request.diagnosticConfigurationRevision = PinnedEngine().Configuration + "-fuel-off"
			}
			if reflect.DeepEqual(variantIdentities(baseRequest, before), variantIdentities(request, result)) {
				t.Fatal("changed identity was omitted")
			}
			projection, err := projectVariantEvidence(result, testConfig(), "")
			if err != nil || !reflect.DeepEqual(want, projection) {
				t.Fatal("identity change altered behavior equality", err)
			}
		})
	}
	for name, mutate := range map[string]func(*Result){
		"choice":  func(r *Result) { r.ChoiceTrace.Bytes = []byte("other") },
		"wasi":    func(r *Result) { r.Transcript = []byte("other") },
		"network": func(r *Result) { r.NetworkTranscript = []byte("other") },
		"stdout":  func(r *Result) { r.Stdout = []byte("other") },
		"stderr":  func(r *Result) { r.Stderr = []byte("other") },
		"outcome": func(r *Result) { r.Termination = "trap" },
		"message": func(r *Result) { r.Message = "boundary failure" },
		"exit":    func(r *Result) { other := uint32(1); r.ExitCode = &other },
		"time":    func(r *Result) { r.VirtualTimeElapsedNanos++ },
	} {
		t.Run(name, func(t *testing.T) {
			changed := before
			mutate(&changed)
			projection, err := projectVariantEvidence(changed, testConfig(), "")
			if err == nil && reflect.DeepEqual(want, projection) {
				t.Fatal("behavior change was hidden")
			}
		})
	}
}

func TestDiagnosticVariantEvidencePreservesDerivedInputs(t *testing.T) {
	for _, gcVariant := range []string{"", "200", "off"} {
		t.Run("gc="+gcVariant, func(t *testing.T) {
			want, err := projectVariantEvidence(Result{}, testConfig(), gcVariant)
			if err != nil {
				t.Fatal(err)
			}
			for name, mutate := range map[string]func(*Config){
				"argv":        func(c *Config) { c.Args = append(c.Args, "other") },
				"environment": func(c *Config) { c.Environment = append(c.Environment, "USER_INPUT=other") },
				"entropy":     func(c *Config) { c.EntropyKey[0]++ },
				"clock step":  func(c *Config) { c.Clock.ReadStepNanos++ },
				"clock epoch": func(c *Config) { c.Clock.EpochNanos++ },
			} {
				t.Run(name, func(t *testing.T) {
					config := testConfig()
					mutate(&config)
					got, err := projectVariantEvidence(Result{}, config, gcVariant)
					if err != nil || reflect.DeepEqual(want, got) {
						t.Fatal("derived input change was hidden", err)
					}
				})
			}
		})
	}
}

func TestDiagnosticVariantEvidenceMasksOnlyDeclaredGCInputs(t *testing.T) {
	for _, gcVariant := range []string{"200", "400", "off"} {
		t.Run(gcVariant, func(t *testing.T) {
			want, err := projectVariantEvidence(Result{}, testConfig(), gcVariant)
			if err != nil {
				t.Fatal(err)
			}
			config := testConfig()
			config.Environment = append(config.Environment, "GOGC="+gcVariant)
			if gcVariant == "off" {
				config.Environment = append(config.Environment, "GOMEMLIMIT=100B")
			}
			got, err := projectVariantEvidence(Result{}, config, gcVariant)
			if err != nil || !reflect.DeepEqual(want, got) {
				t.Fatal("declared GC policy change was not masked", err)
			}
			if gcVariant != "off" {
				config.Environment = append(config.Environment, "GOMEMLIMIT=100B")
				got, err := projectVariantEvidence(Result{}, config, gcVariant)
				if err != nil || reflect.DeepEqual(want, got) {
					t.Fatal("unexpected memory policy change was hidden", err)
				}
			}
		})
	}
}

func TestDiagnosticVariantRecommendationRequiresReliableGain(t *testing.T) {
	for _, test := range []struct {
		name                string
		baseline, candidate []uint64
		failed, identical   bool
		want                string
	}{
		{"gain", []uint64{98, 99, 100, 101, 102}, []uint64{78, 79, 80, 81, 82}, false, true, "adopt"},
		{"spread", []uint64{90, 99, 100, 101, 110}, []uint64{80, 89, 90, 91, 100}, false, true, "investigate"},
		{"slower", []uint64{98, 99, 100, 101, 102}, []uint64{108, 109, 110, 111, 112}, false, true, "reject"},
		{"mismatch", []uint64{98, 99, 100, 101, 102}, []uint64{78, 79, 80, 81, 82}, false, false, "reject"},
		{"failed", []uint64{98, 99, 100, 101, 102}, nil, true, false, "reject"},
		{"missing", []uint64{100}, nil, false, true, "investigate"},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := variantRecommendation(test.baseline, test.candidate, test.failed, test.identical); got != test.want {
				t.Fatalf("recommendation=%s want=%s", got, test.want)
			}
		})
	}
}

type terminalExitHelperBuild struct {
	Crate, Helper, HelperSHA256 string
	RustcPath                   string
	Inputs                      map[string]string
	Command, Environment        []string
	Features                    []string
	RustcVersion                string
	CargoVersion                string
}

func TestTerminalExitStagesHardlinkedCargoHelper(t *testing.T) {
	root := t.TempDir()
	artifact, alias := filepath.Join(root, "cargo-helper"), filepath.Join(root, "cargo-deps-alias")
	data := []byte("#!/bin/sh\nexit 0\n")
	if err := os.WriteFile(artifact, data, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.Link(artifact, alias); err != nil {
		t.Fatal(err)
	}
	if file, _, err := hostfs.OpenPath(artifact); err == nil {
		if err := file.Close(); err != nil {
			t.Fatal(err)
		}
		t.Fatal("hardlinked compiler output unexpectedly admitted")
	}
	helper := filepath.Join(root, "admitted-helper")
	terminalExitStageHelper(t, artifact, helper, fileHash(t, artifact))
	file, info, err := hostfs.OpenPath(helper)
	if err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0755 || !bytes.Equal(mustTerminalExitRead(t, helper), data) || !bytes.Equal(mustTerminalExitRead(t, artifact), data) {
		t.Fatal("staging changed helper bytes, executable mode, or Cargo artifact")
	}
}

func terminalExitStageHelper(t *testing.T, artifact, helper, artifactSHA256 string) {
	t.Helper()
	info, err := os.Lstat(artifact)
	if err != nil || !info.Mode().IsRegular() || info.Size() <= 0 || info.Size() > 512<<20 {
		t.Fatal("Cargo artifact is not a bounded regular executable", err)
	}
	input, err := os.Open(artifact)
	if err != nil {
		t.Fatal(err)
	}
	data, readErr := io.ReadAll(io.LimitReader(input, (512<<20)+1))
	if err := errors.Join(readErr, input.Close()); err != nil {
		t.Fatal(err)
	}
	if len(data) > 512<<20 || digest(data) != artifactSHA256 {
		t.Fatal("Cargo artifact changed before staging")
	}
	staged, err := hostfs.StageContext(t.Context(), filepath.Dir(helper), ".helper-*", 0755, func(_ context.Context, writer io.Writer) error {
		_, err := io.Copy(writer, bytes.NewReader(data))
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := staged.Cleanup(); err != nil {
			t.Error(err)
		}
	})
	if fileHash(t, staged.Path()) != artifactSHA256 {
		t.Fatal("staged helper differs from the Cargo artifact")
	}
	if published, err := staged.PublishNoReplaceContext(t.Context(), helper); err != nil || !published {
		t.Fatal("publish admitted helper", err)
	}
	file, _, err := hostfs.OpenPath(helper)
	if err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	if fileHash(t, artifact) != artifactSHA256 || fileHash(t, helper) != artifactSHA256 {
		t.Fatal("Cargo artifact or admitted helper changed during publication")
	}
}

type terminalExitHelperAdmission struct {
	BuildReceipt, BuildReceiptSHA256   string
	CargoArtifact, CargoArtifactSHA256 string
	Helper, HelperSHA256               string
}

func TestStageTerminalExitReleaseHelper(t *testing.T) {
	root := os.Getenv("GOMAD_WASM_TERMINAL_EXIT_HELPER_STAGE_ROOT")
	if root == "" {
		t.Skip("set GOMAD_WASM_TERMINAL_EXIT_HELPER_STAGE_ROOT to admit a retained Cargo helper build")
	}
	receipt := os.Getenv("GOMAD_WASM_TERMINAL_EXIT_HELPER_BUILD_RECEIPT")
	if !filepath.IsAbs(root) || !filepath.IsAbs(receipt) {
		t.Fatal("absolute staging root and retained raw helper-build receipt are required")
	}
	helper, admission := terminalExitAdmitHelper(t, root, receipt)
	t.Logf("admitted helper=%s receipt=%s sha256=%s", helper, admission, fileHash(t, helper))
}

func terminalExitAdmitHelper(t *testing.T, root, receipt string) (string, string) {
	t.Helper()
	return terminalExitAdmitHelperFeatures(t, root, receipt, []string{})
}

func TestTerminalExitCompilerEnvironmentControls(t *testing.T) {
	for _, test := range []struct {
		name        string
		environment []string
		want        bool
	}{
		{"pinned", []string{"RUSTC=/pinned/rustc", "RUSTC_WRAPPER=", "RUSTC_WORKSPACE_WRAPPER="}, true},
		{"missing-compiler", []string{"RUSTC_WRAPPER=", "RUSTC_WORKSPACE_WRAPPER="}, false},
		{"wrong-compiler", []string{"RUSTC=/other/rustc", "RUSTC_WRAPPER=", "RUSTC_WORKSPACE_WRAPPER="}, false},
		{"conflicting-compiler", []string{"RUSTC=/pinned/rustc", "RUSTC=/other/rustc", "RUSTC_WRAPPER=", "RUSTC_WORKSPACE_WRAPPER="}, false},
		{"missing-wrapper-override", []string{"RUSTC=/pinned/rustc", "RUSTC_WORKSPACE_WRAPPER="}, false},
		{"compiler-wrapper", []string{"RUSTC=/pinned/rustc", "RUSTC_WRAPPER=/wrapper", "RUSTC_WORKSPACE_WRAPPER="}, false},
		{"workspace-wrapper", []string{"RUSTC=/pinned/rustc", "RUSTC_WRAPPER=", "RUSTC_WORKSPACE_WRAPPER=/wrapper"}, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			build := terminalExitHelperBuild{RustcPath: "/pinned/rustc", Environment: test.environment, Inputs: map[string]string{"/pinned/rustc": strings.Repeat("a", 64)}}
			if got := terminalExitCompilerEnvironment(build); got != test.want {
				t.Fatalf("compiler environment admitted=%v, want %v", got, test.want)
			}
		})
	}
}

func terminalExitCompilerEnvironment(build terminalExitHelperBuild) bool {
	if !filepath.IsAbs(build.RustcPath) || len(build.Inputs[build.RustcPath]) != 64 {
		return false
	}
	for _, setting := range []struct{ name, value string }{{"RUSTC", build.RustcPath}, {"RUSTC_WRAPPER", ""}, {"RUSTC_WORKSPACE_WRAPPER", ""}} {
		count := 0
		for _, entry := range build.Environment {
			name, value, _ := strings.Cut(entry, "=")
			if name == setting.name {
				if value != setting.value {
					return false
				}
				count++
			}
		}
		if count != 1 {
			return false
		}
	}
	return true
}

func TestBuildTerminalExitReleaseHelper(t *testing.T) {
	root := os.Getenv("GOMAD_WASM_TERMINAL_EXIT_HELPER_BUILD_ROOT")
	if root == "" {
		t.Skip("set GOMAD_WASM_TERMINAL_EXIT_HELPER_BUILD_ROOT to build and retain the isolated release helper")
	}
	terminalExitBuildHelper(t, root, []string{})
}

func terminalExitCargoHelper(data []byte) (string, error) {
	return terminalExitCargoHelperFeatures(data, []string{})
}

func terminalExitRustInputs(t *testing.T, crate string) map[string]string {
	t.Helper()
	inputs := map[string]string{}
	if err := filepath.WalkDir(filepath.Join(crate, "src"), func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !entry.IsDir() {
			inputs[path] = fileHash(t, path)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"Cargo.toml", "Cargo.lock", "rust-toolchain.toml", "../Makefile"} {
		path := filepath.Clean(filepath.Join(crate, name))
		inputs[path] = fileHash(t, path)
	}
	buildScript := filepath.Join(crate, "build.rs")
	if _, err := os.Stat(buildScript); err == nil {
		inputs[buildScript] = fileHash(t, buildScript)
	} else if !os.IsNotExist(err) {
		t.Fatal(err)
	}
	cargoHome := os.Getenv("CARGO_HOME")
	if cargoHome == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			t.Fatal(err)
		}
		cargoHome = filepath.Join(home, ".cargo")
	}
	for _, name := range []string{"config", "config.toml"} {
		path := filepath.Join(cargoHome, name)
		if _, err := os.Stat(path); err == nil {
			inputs[path] = fileHash(t, path)
		} else if !os.IsNotExist(err) {
			t.Fatal(err)
		}
	}
	for directory := crate; ; directory = filepath.Dir(directory) {
		for _, name := range []string{".cargo/config", ".cargo/config.toml"} {
			path := filepath.Join(directory, name)
			if _, err := os.Stat(path); err == nil {
				inputs[path] = fileHash(t, path)
			} else if !os.IsNotExist(err) {
				t.Fatal(err)
			}
		}
		if filepath.Dir(directory) == directory {
			break
		}
	}
	return inputs
}

func TestTerminalExitCargoArtifactControls(t *testing.T) {
	for _, test := range []struct {
		name, output string
		want         bool
	}{
		{"release", `{"reason":"compiler-artifact","target":{"name":"gomad3-wasmhost"},"profile":{"opt_level":"3","debug_assertions":false},"executable":"/release/helper","features":[]} {"reason":"build-finished","success":true}`, true},
		{"debug-build", `{"reason":"compiler-artifact","target":{"name":"gomad3-wasmhost"},"profile":{"opt_level":"0","debug_assertions":true},"executable":"/release/helper","features":[]} {"reason":"build-finished","success":true}`, false},
		{"diagnostic-feature", `{"reason":"compiler-artifact","target":{"name":"gomad3-wasmhost"},"profile":{"opt_level":"3","debug_assertions":false},"executable":"/release/helper","features":["diagnostic-profile"]} {"reason":"build-finished","success":true}`, false},
		{"missing-features", `{"reason":"compiler-artifact","target":{"name":"gomad3-wasmhost"},"profile":{"opt_level":"3","debug_assertions":false},"executable":"/release/helper"} {"reason":"build-finished","success":true}`, false},
		{"failed-build", `{"reason":"compiler-artifact","target":{"name":"gomad3-wasmhost"},"profile":{"opt_level":"3","debug_assertions":false},"executable":"/release/helper","features":[]} {"reason":"build-finished","success":false}`, false},
		{"missing-finish", `{"reason":"compiler-artifact","target":{"name":"gomad3-wasmhost"},"profile":{"opt_level":"3","debug_assertions":false},"executable":"/release/helper","features":[]}`, false},
		{"missing-helper", `{"reason":"build-finished","success":true}`, false},
		{"malformed", `{`, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			helper, err := terminalExitCargoHelper([]byte(test.output))
			if (err == nil) != test.want || test.want && helper != "/release/helper" {
				t.Fatalf("Cargo artifact = %q, %v", helper, err)
			}
		})
	}
}

func terminalExitBuildInputs(t *testing.T, receipt string, request Request, crate string) map[string]string {
	t.Helper()
	return terminalExitBuildInputsFeatures(t, receipt, request, crate, []string{})
}

func terminalExitHelperInputs(t *testing.T, receipt string, request Request, crate string) map[string]string {
	t.Helper()
	return terminalExitHelperInputsFeatures(t, receipt, request, crate, []string{})
}

func terminalExitModuleInputs(t *testing.T, receipt string, request Request) map[string]string {
	t.Helper()
	data, err := os.ReadFile(receipt)
	if err != nil {
		t.Fatal(err)
	}
	var producer struct {
		Spec     target.Spec
		Prepared target.Prepared
		Options  struct{ CompilerPath string }
	}
	if err := json.Unmarshal(data, &producer); err != nil {
		t.Fatal(err)
	}
	prepared := producer.Prepared
	if producer.Spec.Source != "./tests" || prepared.Source != "./tests" || prepared.Kind != target.KindGoTest || prepared.Path != request.ModulePath || prepared.SHA256 != "sha256:"+request.ModuleSHA256 || !slices.Equal(prepared.BuildTags, []string{"disable_grpc_modules", "sqlite3_dotlk", "test_dep"}) {
		t.Fatal("producer must bind actual ./tests, unchanged tags, and the measured module")
	}
	if err := prepared.ValidateBackendPayloads(); err != nil {
		t.Fatal(err)
	}
	var provenance struct {
		Schema, Profile, Source, Kind, ModuleSHA256, GoVersion, RuntimeSHA256, RuntimeImplementationSHA256, ModelSHA256, CompilerSHA256 string
		Sources                                                                                                                         []struct{ Path, SHA256 string }
	}
	if len(prepared.BackendPayloads) != 1 {
		t.Fatal("module producer has no unique retained WASM provenance")
	}
	if err := json.Unmarshal(prepared.BackendPayloads[0].Data, &provenance); err != nil {
		t.Fatal(err)
	}
	if provenance.Schema != "gomad-wasm.provenance/v1" || provenance.Profile != CooperativeProfile || provenance.Source != "./tests" || provenance.Kind != "go-test" || provenance.ModuleSHA256 != prepared.SHA256 || provenance.GoVersion != "go1.27.1" || provenance.RuntimeSHA256 != "sha256:"+request.Runtime.Choice.ExecutionIdentity.ToolchainBuildKey || provenance.RuntimeImplementationSHA256 != toolchain.ImplementationSHA256() || provenance.ModelSHA256 != ImplementationSHA256() || provenance.CompilerSHA256 != "sha256:"+fileHash(t, producer.Options.CompilerPath) || len(provenance.Sources) == 0 {
		t.Fatal("module provenance is stale or lacks current compiler/runtime/model/source bindings")
	}
	inputs := map[string]string{receipt: fileHash(t, receipt), request.ModulePath: request.ModuleSHA256}
	for _, source := range provenance.Sources {
		inputs[source.Path] = strings.TrimPrefix(source.SHA256, "sha256:")
	}
	temporalVerifyInputs(t, inputs)
	return inputs
}

func mustTerminalExitRead(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func TestTemporalTerminalToProcessExit(t *testing.T) {
	root := os.Getenv("GOMAD_WASM_TERMINAL_EXIT_ROOT")
	if root == "" {
		t.Skip("set GOMAD_WASM_TERMINAL_EXIT_ROOT for release-helper Temporal terminal-to-reap measurements")
	}
	if !filepath.IsAbs(root) {
		t.Fatal("terminal-exit evidence root must be absolute")
	}
	if !filepath.IsAbs(os.Getenv("GOMAD_WASM_TEST_HELPER")) {
		t.Fatal("GOMAD_WASM_TEST_HELPER must explicitly name the candidate release helper")
	}
	evidence, err := diagnosticRunRoot(root, "terminal-exit-")
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("GOMAD_WASM_PROTOTYPE_ROOT", evidence)
	repo, err := filepath.Abs("../../../../../..")
	if err != nil {
		t.Fatal(err)
	}
	engineSource := filepath.Join(repo, "tools/gomad/internal/backends/wasm/wasmhost/src/engine.rs")
	engineBytes, err := os.ReadFile(engineSource)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(engineBytes, []byte("std::mem::forget")) || !bytes.Contains(engineBytes, []byte("native_unwind_info(false)")) {
		t.Fatal("measure a release helper built from the no-unwind candidate without the module/store leak")
	}
	engineSHA256 := fileHash(t, engineSource)
	module := os.Getenv("GOMAD_WASM_TERMINAL_EXIT_MODULE")
	if !filepath.IsAbs(module) {
		t.Fatal("GOMAD_WASM_TERMINAL_EXIT_MODULE must name the current ./tests cooperative module")
	}
	config := prototypeTemporalConfig(t, repo, "/workspace/tests", "-test.run=^TestWorkflowTypeEncodingSuite$/^TestPlainASCII$/^Succeeds$", "-persistenceType=sql", "-persistenceDriver=sqlite")
	request := prototypeRequest(t, module, config, 0, 30*time.Minute)
	helperReceipt := os.Getenv("GOMAD_WASM_TERMINAL_EXIT_HELPER_RECEIPT")
	moduleReceipt := os.Getenv("GOMAD_WASM_TERMINAL_EXIT_MODULE_RECEIPT")
	if !filepath.IsAbs(helperReceipt) || !filepath.IsAbs(moduleReceipt) {
		t.Fatal("absolute helper-admission and ./tests producer receipts are required")
	}
	helperInputs := terminalExitHelperInputs(t, helperReceipt, request, filepath.Dir(filepath.Dir(engineSource)))
	moduleInputs := terminalExitModuleInputs(t, moduleReceipt, request)
	var admission terminalExitHelperAdmission
	if err := json.Unmarshal(mustTerminalExitRead(t, helperReceipt), &admission); err != nil {
		t.Fatal(err)
	}
	temporalWrite(t, filepath.Join(evidence, "helper-admission.json"), mustTerminalExitRead(t, helperReceipt))
	temporalWrite(t, filepath.Join(evidence, "helper-build.json"), mustTerminalExitRead(t, admission.BuildReceipt))
	temporalWrite(t, filepath.Join(evidence, "temporal-producer.json"), mustTerminalExitRead(t, moduleReceipt))
	var observations []terminalExitObservation
	failed := false
	for _, mismatch := range []bool{false, true} {
		name := "success"
		if mismatch {
			name = "post-load-authorization-mismatch"
		}
		observation := measureTerminalExit(t, request, mismatch)
		observations = append(observations, observation)
		t.Logf("%s terminal-to-reap=%dns termination=%s reaped=%v error=%s", name, observation.TerminalToReapNanos, observation.Result.Termination, observation.Host.GroupGone, observation.Error)
		temporalJSON(t, filepath.Join(evidence, name+"-observation.json"), observation)
		temporalWrite(t, filepath.Join(evidence, name+"-helper-stderr.txt"), observation.Host.Stderr.RawBytes)
		temporalVerifyInputs(t, helperInputs)
		temporalVerifyInputs(t, moduleInputs)
		terminalExitHelperInputs(t, helperReceipt, request, filepath.Dir(filepath.Dir(engineSource)))
		if !observation.within(time.Second) {
			failed = true
		}
		if mismatch {
			if !observation.AuthorizationChanged || observation.Result.Termination != "invalid" || observation.Result.Message != "execution authorization identity mismatch" || observation.Result.Started == nil || observation.Result.Telemetry == nil || observation.Result.Telemetry.InstantiateNanos != 0 || observation.Result.Telemetry.ExecuteNanos != 0 {
				failed = true
			}
		} else if observation.Result.Termination != "exit" || observation.Result.ExitCode == nil || *observation.Result.ExitCode != 0 || !strings.Contains(string(observation.Result.Stdout), "--- PASS: TestWorkflowTypeEncodingSuite") || !strings.Contains(string(observation.Result.Stdout), "TestPlainASCII/Succeeds") {
			failed = true
		}
	}
	info, err := os.Stat(module)
	if err != nil {
		t.Fatal(err)
	}
	temporalJSON(t, filepath.Join(evidence, "terminal-exit.json"), map[string]any{
		"schema": "gomad3.wasm-terminal-exit-measurement/v1", "host": PinnedEngine(), "bound_nanos": uint64(time.Second),
		"module_sha256": request.ModuleSHA256, "module_bytes": info.Size(), "helper_sha256": request.HelperSHA256,
		"engine_source_sha256":  engineSHA256,
		"helper_receipt_sha256": fileHash(t, helperReceipt), "module_receipt_sha256": fileHash(t, moduleReceipt),
		"selection": config.Args, "observations": observations,
		"boundary": "receipt of complete actual validated terminal frame through completed host supervisor wait, capture drain and group reaping; conservative upper bound on exit time",
	})
	if failed {
		t.Fatalf("Temporal terminal-to-process-exit proof failed; raw measurements retained in %s", evidence)
	}
}

// Temporary, uncommitted: diagnostic split of the smoke-workflow and
// smoke-activity selections. Delete after profiling.

func TestScratchSmokeProfile(t *testing.T) {
	root := prototypeRoot(t)
	repo, err := filepath.Abs("../../../../../..")
	if err != nil {
		t.Fatal(err)
	}
	modules := filepath.Join(root, "modules")
	if err := os.MkdirAll(modules, 0700); err != nil {
		t.Fatal(err)
	}
	module := filepath.Join(modules, "tests.wasm")
	if _, err := os.Stat(module); err != nil {
		_, seconds := prototypeBuild(t, repo, "test", "./tests", module, "disable_grpc_modules", "test_dep", "sqlite3_dotlk")
		t.Logf("built tests.wasm in %.1fs", seconds)
	}
	persistence := []string{"-persistenceType=sql", "-persistenceDriver=sqlite"}
	suites := []struct{ name, run string }{
		{"workflow", "^TestWorkflowTypeEncodingSuite$/^TestPlainASCII$/^Succeeds$"},
		{"activity", "^TestActivityTestSuite$"},
	}
	evidence := filepath.Join(root, "evidence")
	if err := os.MkdirAll(evidence, 0700); err != nil {
		t.Fatal(err)
	}
	helpers := strings.Split(os.Getenv("GOMAD_SCRATCH_HELPERS"), ",")
	for _, owner := range helpers {
		name, path, _ := strings.Cut(owner, "=")
		instrumented := strings.HasPrefix(name, "diag")
		t.Setenv("GOMAD_WASM_TEST_HELPER", path)
		warm := prototypeRequest(t, module, prototypeTemporalConfig(t, repo, "/workspace/tests", append([]string{"-test.run=^$"}, persistence...)...), 0, 30*time.Minute)
		if os.Getenv("GOMAD_SCRATCH_SKIP_PREPARE") == "" {
			runDiagnosticProfile(t, evidence, name+"-prepare", warm, instrumented, false)
		}
		for _, suite := range suites {
			if only := os.Getenv("GOMAD_SCRATCH_ONLY"); only != "" && only != suite.name {
				continue
			}
			for _, seed := range []uint64{0, 1000003} {
				config := prototypeTemporalConfig(t, repo, "/workspace/tests", append([]string{"-test.run=" + suite.run}, persistence...)...)
				request := prototypeRequest(t, module, config, seed, 30*time.Minute)
				if os.Getenv("GOMAD_SCRATCH_TRANSCRIPT") != "" {
					request.DigestTranscript = false
				}
				started := time.Now()
				result := runDiagnosticProfile(t, evidence, name+"-"+suite.name+"-seed"+stringSeed(seed), request, instrumented, true)
				if !request.DigestTranscript {
					operations := map[string]int{}
					for _, line := range bytes.Split(result.Transcript, []byte{'\n'}) {
						var frame struct{ Type, Op string }
						if json.Unmarshal(line, &frame) == nil && frame.Type == "call" {
							operations[frame.Op]++
						}
					}
					t.Logf("operations: %v", operations)
				}
				t.Logf("%s %s seed %d: run wall %s, authorized→terminal %s", name, suite.name, seed, time.Since(started), time.Duration(result.AuthorizationToTerminalNanos))
			}
		}
	}
}

type diagnosticHelperAdmission struct {
	Receipt  string
	Features []string
}

type diagnosticModuleAdmission struct {
	Schema, Source, SourceHead, CPUInfoSHA256       string
	Directory, Target, Kind                         string
	Tags                                            []string
	Module, ModuleSHA256, Compiler, CompilerVersion string
	Overlay, RuntimeKey, RuntimeSHA256, ModelSHA256 string
	GOWASM                                          string
	Command, Environment                            []string
	Inputs                                          map[string]string
}

type diagnosticAdmission struct {
	Repo, SourceHead, CPUInfo, BootID, Receipt string
	Inputs                                     map[string]string
	Helpers                                    map[string]diagnosticHelperAdmission
	Modules                                    map[string]diagnosticModuleAdmission
	ModuleReceipts                             map[string]string
}

func diagnosticGit(repo string, args ...string) (string, error) {
	result, err := hostexec.Run(context.Background(), hostexec.Request{Command: append([]string{"git", "-C", repo}, args...), Dir: repo, Timeout: 30 * time.Second, TerminateGrace: time.Second, OutputLimit: 1 << 20})
	if err != nil || result.ExitCode != 0 || result.Termination != hostexec.TerminationExit || !result.GroupGone || result.Stdout.Truncated || result.Stderr.Truncated {
		return "", fmt.Errorf("diagnostic source identity: %v, exit=%d, termination=%s", err, result.ExitCode, result.Termination)
	}
	return strings.TrimSpace(string(result.Stdout.RawBytes)), nil
}

func diagnosticCurrentHost(t *testing.T) (string, string) {
	t.Helper()
	cpu, admitted := diagnosticPerformanceHostEvidence("/proc/cpuinfo")
	if !admitted || runtime.Version() != "go1.27.1" {
		t.Fatal("diagnostic measurements require actual Linux/arm64 CPU evidence and stock Go 1.27.1")
	}
	boot, err := hostfs.ReadBounded("/proc/sys/kernel/random/boot_id", 128)
	if err != nil || len(bytes.TrimSpace(boot)) == 0 {
		t.Fatal("diagnostic measurements require bounded current boot identity", err)
	}
	return cpu, strings.TrimSpace(string(boot))
}

func newDiagnosticAdmission(t *testing.T, repo, root string) *diagnosticAdmission {
	t.Helper()
	cpu, boot := diagnosticCurrentHost(t)
	head, err := diagnosticGit(repo, "rev-parse", "HEAD")
	if err != nil || head == "" {
		t.Fatal("diagnostic source HEAD unavailable", err)
	}
	a := &diagnosticAdmission{Repo: repo, SourceHead: head, CPUInfo: cpu, BootID: boot, Inputs: map[string]string{}, Helpers: map[string]diagnosticHelperAdmission{}, Modules: map[string]diagnosticModuleAdmission{}, ModuleReceipts: map[string]string{}}
	a.checkHost(t, true)
	a.Receipt = filepath.Join(root, "host-admission.json")
	temporalJSON(t, a.Receipt, map[string]any{"schema": "gomad-diagnostic-host/v1", "source": repo, "source_head": head, "go_version": runtime.Version(), "goos": runtime.GOOS, "goarch": runtime.GOARCH, "cpuinfo": cpu, "boot_id": boot, "minimum_mem_available_bytes": uint64(8 << 30)})
	a.Inputs[a.Receipt] = fileHash(t, a.Receipt)
	return a
}

func (a *diagnosticAdmission) checkHost(t *testing.T, before bool) {
	t.Helper()
	cpu, boot := diagnosticCurrentHost(t)
	head, headErr := diagnosticGit(a.Repo, "rev-parse", "HEAD")
	status, statusErr := diagnosticGit(a.Repo, "status", "--porcelain", "--untracked-files=all")
	if cpu != a.CPUInfo || boot != a.BootID || headErr != nil || statusErr != nil || head != a.SourceHead || status != "" {
		t.Fatal("diagnostic CPU/boot/source identity changed or worktree is dirty", headErr, statusErr)
	}
	if before {
		if _, err := matchedPrepareAdmitted(runtime.GOOS, "/proc/meminfo", "1", func() (Result, error) { return Result{}, nil }); err != nil {
			t.Fatal(err)
		}
	}
}

func (a *diagnosticAdmission) addHelper(t *testing.T, helper, receipt string, features []string) {
	t.Helper()
	if !filepath.IsAbs(receipt) {
		t.Fatal("diagnostic helper requires an absolute authentic helper-admission receipt")
	}
	crate := filepath.Join(a.Repo, "tools/gomad/internal/backends/wasm/wasmhost")
	inputs := terminalExitHelperInputsFeatures(t, receipt, Request{HelperPath: helper, HelperSHA256: fileHash(t, helper)}, crate, features)
	var admission terminalExitHelperAdmission
	if err := json.Unmarshal(mustTerminalExitRead(t, receipt), &admission); err != nil {
		t.Fatal(err)
	}
	var build terminalExitHelperBuild
	if err := json.Unmarshal(mustTerminalExitRead(t, admission.BuildReceipt), &build); err != nil || !diagnosticHelperCommand(build, features) {
		t.Fatal("diagnostic helper must retain the exact pinned, single-job feature build command", err)
	}
	for _, name := range []string{"cargo-build.stdout", "cargo-build.json", "rustc-version.stdout", "cargo-version.stdout"} {
		path := filepath.Join(filepath.Dir(admission.BuildReceipt), name)
		if build.Inputs[path] != fileHash(t, path) {
			t.Fatal("diagnostic helper is missing authenticated raw compiler evidence", name)
		}
	}
	var execution struct {
		Command  []string
		ExitCode int
	}
	if err := json.Unmarshal(mustTerminalExitRead(t, filepath.Join(filepath.Dir(admission.BuildReceipt), "cargo-build.json")), &execution); err != nil || execution.ExitCode != 0 || !slices.Equal(execution.Command, build.Command) {
		t.Fatal("diagnostic helper command differs from actual retained execution", err)
	}
	if string(mustTerminalExitRead(t, filepath.Join(filepath.Dir(admission.BuildReceipt), "rustc-version.stdout"))) != build.RustcVersion || string(mustTerminalExitRead(t, filepath.Join(filepath.Dir(admission.BuildReceipt), "cargo-version.stdout"))) != build.CargoVersion {
		t.Fatal("diagnostic helper compiler versions differ from actual retained output")
	}
	a.Helpers[helper] = diagnosticHelperAdmission{Receipt: receipt, Features: slices.Clone(features)}
	maps.Copy(a.Inputs, inputs)
}

func (a *diagnosticAdmission) check(t *testing.T, root, name string, request Request, before bool) {
	t.Helper()
	a.checkHost(t, before)
	if err := diagnosticVerifyInputs(a.Inputs); err != nil {
		t.Fatal(err)
	}
	helper, ok := a.Helpers[request.HelperPath]
	if !ok {
		t.Fatal("diagnostic dispatch has no admitted helper")
	}
	terminalExitHelperInputsFeatures(t, helper.Receipt, request, filepath.Join(a.Repo, "tools/gomad/internal/backends/wasm/wasmhost"), helper.Features)
	module, ok := a.Modules[request.ModulePath]
	if !ok || !diagnosticModuleMatches(module, request) {
		t.Fatal("diagnostic dispatch has no matching actual module build")
	}
	temporalJSON(t, filepath.Join(root, name+"-admission.json"), map[string]any{
		"host_receipt": a.Receipt, "host_receipt_sha256": a.Inputs[a.Receipt],
		"helper_receipt": helper.Receipt, "helper_receipt_sha256": a.Inputs[helper.Receipt],
		"module_receipt": a.ModuleReceipts[request.ModulePath], "module_receipt_sha256": a.Inputs[a.ModuleReceipts[request.ModulePath]],
		"helper_sha256": request.HelperSHA256, "module_sha256": request.ModuleSHA256, "source_head": a.SourceHead, "before_dispatch": before,
	})
}

func diagnosticModuleMatches(module diagnosticModuleAdmission, request Request) bool {
	return request.Runtime != nil && request.Runtime.Choice != nil && module.Schema == "gomad-diagnostic-module-build/v1" && module.Module == request.ModulePath && module.ModuleSHA256 == request.ModuleSHA256 && module.RuntimeKey == request.Runtime.Choice.ExecutionIdentity.ToolchainBuildKey && module.RuntimeSHA256 == toolchain.ImplementationSHA256() && module.ModelSHA256 == ImplementationSHA256()
}

// This producer is deliberately test-only: production preparation fixes GOWASM="".
func (a *diagnosticAdmission) buildModule(t *testing.T, output, gowasm string) {
	t.Helper()
	a.buildModuleTarget(t, a.Repo, output, gowasm, "./tests", "test", []string{"disable_grpc_modules", "test_dep", "sqlite3_dotlk"})
}

func diagnosticModuleCommands(compiler, overlay, output, target, kind string, tags []string) ([]string, []string, error) {
	if kind != "test" && kind != "build" {
		return nil, nil, fmt.Errorf("diagnostic module kind %q", kind)
	}
	list := []string{compiler, "list", "-p=1", "-deps"}
	command := []string{compiler, kind}
	if kind == "test" {
		list = append(list, "-test")
		command = append(command, "-c")
	}
	list = append(list, "-json", "-overlay", overlay)
	command = append(command, "-p=1", "-trimpath")
	if kind == "build" {
		command = append(command, "-buildvcs=false")
	}
	command = append(command, "-overlay", overlay, "-o", output)
	if len(tags) != 0 {
		list = append(list, "-tags", strings.Join(tags, ","))
		command = append(command, "-tags", strings.Join(tags, ","))
	}
	return append(list, target), append(command, target), nil
}

func (a *diagnosticAdmission) buildModuleTarget(t *testing.T, directory, output, gowasm, target, kind string, tags []string) {
	t.Helper()
	if gowasm != "" && gowasm != "satconv,signext" {
		t.Fatal("unexpected diagnostic GOWASM variant")
	}
	a.checkHost(t, true)
	evidence, err := diagnosticRunRoot(filepath.Dir(output), "module-build-")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Lstat(output); !os.IsNotExist(err) {
		t.Fatal("refusing to overwrite a diagnostic module", err)
	}
	compiler, overlay, key := prototypeToolchain(t)
	compilerSHA256 := fileHash(t, compiler)
	var environment []string
	for _, entry := range os.Environ() {
		name, _, _ := strings.Cut(entry, "=")
		if slices.Contains([]string{"PATH", "HOME", "GOCACHE", "GOMODCACHE", "GOPATH", "TMPDIR", "GOTMPDIR", "GOPROXY", "GOSUMDB", "GONOPROXY", "GONOSUMDB", "GOPRIVATE"}, name) {
			environment = append(environment, entry)
		}
	}
	environment = append(environment, "GOROOT="+filepath.Dir(filepath.Dir(compiler)), "GOOS=wasip1", "GOARCH=wasm", "GOEXPERIMENT=nogreenteagc", "GOWORK=off", "GOENV=off", "GOFLAGS=-mod=readonly", "GOTOOLCHAIN=local", "CGO_ENABLED=0", "GOWASM="+gowasm, "GOMAXPROCS=1")
	version := temporalCommand(t, evidence, "go-version", directory, environment, []string{compiler, "version"})
	if version.Stdout.Truncated || version.Stderr.Truncated || !strings.HasPrefix(string(version.Stdout.RawBytes), "go version go1.27.1 ") {
		t.Fatal("diagnostic module requires pinned stock Go 1.27.1")
	}
	listCommand, command, err := diagnosticModuleCommands(compiler, overlay, output, target, kind, tags)
	if err != nil {
		t.Fatal(err)
	}
	listed := temporalCommand(t, evidence, "sources-before", directory, environment, listCommand)
	if listed.Stdout.Truncated || listed.Stderr.Truncated {
		t.Fatal("truncated diagnostic source closure")
	}
	inputs := diagnosticModuleSourceInputs(t, listed.Stdout.RawBytes, compiler, overlay, directory)
	if inputs[compiler] != compilerSHA256 {
		t.Fatal("diagnostic compiler changed before build")
	}
	built := temporalCommand(t, evidence, "module-build", directory, environment, command)
	if built.Stdout.Truncated || built.Stderr.Truncated {
		t.Fatal("truncated diagnostic compiler output")
	}
	listed = temporalCommand(t, evidence, "sources-after", directory, environment, listCommand)
	if listed.Stdout.Truncated || listed.Stderr.Truncated {
		t.Fatal("truncated diagnostic source closure")
	}
	if !maps.Equal(inputs, diagnosticModuleSourceInputs(t, listed.Stdout.RawBytes, compiler, overlay, directory)) {
		t.Fatal("diagnostic compiler, overlay or source closure changed during module build")
	}
	a.checkHost(t, false)
	for _, operation := range []string{"go-version", "sources-before", "module-build", "sources-after"} {
		for _, suffix := range []string{".json", ".stdout", ".stderr"} {
			path := filepath.Join(evidence, operation+suffix)
			inputs[path] = fileHash(t, path)
		}
	}
	module := diagnosticModuleAdmission{Schema: "gomad-diagnostic-module-build/v1", Source: a.Repo, SourceHead: a.SourceHead, CPUInfoSHA256: digest([]byte(a.CPUInfo)), Directory: directory, Target: target, Kind: kind, Tags: slices.Clone(tags), Module: output, ModuleSHA256: fileHash(t, output), Compiler: compiler, CompilerVersion: string(version.Stdout.RawBytes), Overlay: overlay, RuntimeKey: key, RuntimeSHA256: toolchain.ImplementationSHA256(), ModelSHA256: ImplementationSHA256(), GOWASM: gowasm, Command: command, Environment: environment, Inputs: inputs}
	receipt := filepath.Join(evidence, "module-admission.json")
	temporalJSON(t, receipt, module)
	a.Modules[output], a.ModuleReceipts[output] = module, receipt
	maps.Copy(a.Inputs, inputs)
	a.Inputs[output], a.Inputs[receipt] = module.ModuleSHA256, fileHash(t, receipt)
}

func diagnosticModuleSourceInputs(t *testing.T, data []byte, compiler, overlay, repo string) map[string]string {
	t.Helper()
	var replacement struct{ Replace map[string]string }
	if err := json.Unmarshal(mustTerminalExitRead(t, overlay), &replacement); err != nil || len(replacement.Replace) == 0 {
		t.Fatal("diagnostic module requires actual cooperative overlay bindings", err)
	}
	inputs := map[string]string{}
	// Mirrors the production source inventory: a listed file the overlay replaces
	// binds its replacement, and its original only when one exists on disk.
	bind := func(name string) {
		if to, replaced := replacement.Replace[name]; replaced {
			if to == "" {
				t.Fatal("cooperative overlay removed a selected build input", name)
			}
			inputs[to] = fileHash(t, to)
			if _, err := os.Stat(name); os.IsNotExist(err) {
				return
			} else if err != nil {
				t.Fatal(err)
			}
		}
		inputs[name] = fileHash(t, name)
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	for {
		var pkg struct {
			Dir                                                        string
			GoFiles, CgoFiles, SFiles, HFiles, SysoFiles, EmbedFiles   []string
			TestGoFiles, XTestGoFiles, TestEmbedFiles, XTestEmbedFiles []string
			Module                                                     *struct {
				GoMod   string
				Replace *struct{ GoMod string }
			}
		}
		if err := decoder.Decode(&pkg); err == io.EOF {
			break
		} else if err != nil {
			t.Fatal(err)
		}
		for _, names := range [][]string{pkg.GoFiles, pkg.CgoFiles, pkg.SFiles, pkg.HFiles, pkg.SysoFiles, pkg.EmbedFiles, pkg.TestGoFiles, pkg.XTestGoFiles, pkg.TestEmbedFiles, pkg.XTestEmbedFiles} {
			for _, name := range names {
				if !filepath.IsAbs(name) {
					name = filepath.Join(pkg.Dir, name)
				}
				bind(name)
			}
		}
		if pkg.Module != nil {
			gomod := pkg.Module.GoMod
			if pkg.Module.Replace != nil {
				gomod = pkg.Module.Replace.GoMod
			}
			if gomod != "" {
				inputs[gomod] = fileHash(t, gomod)
			}
		}
	}
	for from, to := range replacement.Replace {
		if to != "" {
			inputs[to] = fileHash(t, to)
		}
		if _, err := os.Stat(from); err == nil {
			inputs[from] = fileHash(t, from)
		} else if !os.IsNotExist(err) {
			t.Fatal(err)
		}
	}
	for _, path := range []string{compiler, overlay, filepath.Join(repo, "go.mod"), filepath.Join(repo, "go.sum"), filepath.Join(filepath.Dir(filepath.Dir(compiler)), "VERSION")} {
		inputs[path] = fileHash(t, path)
	}
	for _, tool := range []string{"compile", "link", "asm"} {
		path := filepath.Join(filepath.Dir(filepath.Dir(compiler)), "pkg/tool", runtime.GOOS+"_"+runtime.GOARCH, tool)
		inputs[path] = fileHash(t, path)
	}
	return inputs
}

func TestBuildDiagnosticProfileHelper(t *testing.T) {
	diagnosticBuildHelper(t, "GOMAD_WASM_DIAGNOSTIC_HELPER_BUILD_ROOT", []string{"diagnostic-profile"})
}

func TestBuildDiagnosticGuestProfileHelper(t *testing.T) {
	diagnosticBuildHelper(t, "GOMAD_WASM_GUEST_HELPER_BUILD_ROOT", []string{"diagnostic-profile", "guest-profile"})
}

func TestBuildDiagnosticFuelOffHelper(t *testing.T) {
	diagnosticBuildHelper(t, "GOMAD_WASM_FUEL_OFF_HELPER_BUILD_ROOT", []string{"diagnostic-fuel-off", "diagnostic-profile"})
}

func diagnosticBuildHelper(t *testing.T, variable string, features []string) {
	t.Helper()
	root := os.Getenv(variable)
	if root == "" {
		t.Skipf("set %s to retain a pinned, feature-exact diagnostic helper build and admission", variable)
	}
	terminalExitBuildHelper(t, root, features)
}

func diagnosticHelperCommand(build terminalExitHelperBuild, features []string) bool {
	if len(build.Command) == 0 || !filepath.IsAbs(build.Command[0]) || build.Inputs[build.Command[0]] == "" {
		return false
	}
	return len(features) != 0 && slices.Equal(build.Command, terminalExitCargoCommand(build.Command[0], build.Crate, features))
}

func diagnosticVerifyInputs(inputs map[string]string) error {
	for path, want := range inputs {
		file, err := os.Open(path)
		if err != nil {
			return err
		}
		hash := sha256.New()
		_, readErr := io.Copy(hash, file)
		closeErr := file.Close()
		if readErr != nil {
			return readErr
		}
		if closeErr != nil {
			return closeErr
		}
		if fmt.Sprintf("%x", hash.Sum(nil)) != want {
			return fmt.Errorf("diagnostic input changed: %s", path)
		}
	}
	return nil
}

func TestDiagnosticAdmissionCommand(t *testing.T) {
	features := []string{"diagnostic-profile", "guest-profile"}
	command := []string{"/pinned/cargo", "build", "--manifest-path", "/crate/Cargo.toml", "--release", "--locked", "--jobs=1", "--no-default-features", "--message-format=json", "--features", "diagnostic-profile,guest-profile"}
	build := terminalExitHelperBuild{Crate: "/crate", Command: command, Inputs: map[string]string{"/pinned/cargo": "digest"}}
	if !diagnosticHelperCommand(build, features) {
		t.Fatal("exact command rejected")
	}
	for index := range command {
		changed := build
		changed.Command = slices.Clone(command)
		changed.Command[index] = "wrong"
		if diagnosticHelperCommand(changed, features) {
			t.Fatalf("changed command element %d admitted", index)
		}
	}
}

func TestDiagnosticModuleBuildCommands(t *testing.T) {
	for _, test := range []struct {
		kind        string
		tags        []string
		list, build []string
	}{
		{"test", []string{"test_dep"}, []string{"/go", "list", "-p=1", "-deps", "-test", "-json", "-overlay", "/overlay", "-tags", "test_dep", "./fixture"}, []string{"/go", "test", "-c", "-p=1", "-trimpath", "-overlay", "/overlay", "-o", "/module", "-tags", "test_dep", "./fixture"}},
		{"build", nil, []string{"/go", "list", "-p=1", "-deps", "-json", "-overlay", "/overlay", "./fixture"}, []string{"/go", "build", "-p=1", "-trimpath", "-buildvcs=false", "-overlay", "/overlay", "-o", "/module", "./fixture"}},
	} {
		listed, built, err := diagnosticModuleCommands("/go", "/overlay", "/module", "./fixture", test.kind, test.tags)
		if err != nil || !slices.Equal(listed, test.list) || !slices.Equal(built, test.build) {
			t.Fatalf("%s compiler commands: %v %v %v", test.kind, listed, built, err)
		}
	}
	if _, _, err := diagnosticModuleCommands("/go", "/overlay", "/module", "./fixture", "run", nil); err == nil {
		t.Fatal("non-build compiler operation admitted")
	}
}

func TestDiagnosticModuleSourceBindings(t *testing.T) {
	root := t.TempDir()
	compiler := filepath.Join(root, "stock/bin/go")
	overlay := filepath.Join(root, "overlay.json")
	paths := []string{compiler, filepath.Join(root, "stock/VERSION"), filepath.Join(root, "go.mod"), filepath.Join(root, "go.sum"), filepath.Join(root, "fixture.go"), filepath.Join(root, "fixture.h"), filepath.Join(root, "fixture.syso"), filepath.Join(root, "replacement.go"), filepath.Join(root, "added-replacement.go"), filepath.Join(root, "embedded.txt")}
	for _, tool := range []string{"compile", "link", "asm"} {
		paths = append(paths, filepath.Join(root, "stock/pkg/tool", runtime.GOOS+"_"+runtime.GOARCH, tool))
	}
	for _, path := range paths {
		if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
			t.Fatal(err)
		}
		temporalWrite(t, path, []byte("original"))
	}
	// added.go exists only in the overlay, as the cooperative runtime/net glue does.
	added := filepath.Join(root, "added.go")
	temporalJSON(t, overlay, struct{ Replace map[string]string }{map[string]string{filepath.Join(root, "fixture.go"): filepath.Join(root, "replacement.go"), added: filepath.Join(root, "added-replacement.go")}})
	listed, err := json.Marshal(struct {
		Dir                                        string
		GoFiles, HFiles, SysoFiles, TestEmbedFiles []string
	}{root, []string{"fixture.go", "added.go"}, []string{"fixture.h"}, []string{"fixture.syso"}, []string{"embedded.txt"}})
	if err != nil {
		t.Fatal(err)
	}
	inputs := diagnosticModuleSourceInputs(t, listed, compiler, overlay, root)
	if _, ok := inputs[added]; ok {
		t.Fatal("overlay-only file bound as an on-disk input")
	}
	paths = append(paths, overlay)
	for _, path := range paths {
		if inputs[path] != fileHash(t, path) {
			t.Fatalf("producer omitted %s", path)
		}
		original := mustTerminalExitRead(t, path)
		temporalWrite(t, path, []byte("changed"))
		if err := diagnosticVerifyInputs(inputs); err == nil {
			t.Fatalf("changed compiler/source/tool admitted: %s", path)
		}
		temporalWrite(t, path, original)
	}
}

func TestDiagnosticRecoveryCompilerEnvironment(t *testing.T) {
	t.Setenv("GOMAXPROCS", "1")
	t.Setenv("GOTMPDIR", "/private/compiler-temp")
	environment := temporalBuildEnvironment("/pinned/bin/go")
	for _, expected := range []string{"GOMAXPROCS=1", "GOTMPDIR=/private/compiler-temp", "GOENV=off", "GOTOOLCHAIN=local", "GOWORK=off", "GOFLAGS="} {
		if !slices.Contains(environment, expected) {
			t.Fatalf("compiler environment lost %s", expected)
		}
	}
}

func TestDiagnosticAdmissionInputs(t *testing.T) {
	path := filepath.Join(t.TempDir(), "input")
	if err := os.WriteFile(path, []byte("original"), 0600); err != nil {
		t.Fatal(err)
	}
	inputs := map[string]string{path: fileHash(t, path)}
	if err := diagnosticVerifyInputs(inputs); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("changed"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := diagnosticVerifyInputs(inputs); err == nil {
		t.Fatal("changed input admitted")
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if err := diagnosticVerifyInputs(inputs); err == nil {
		t.Fatal("missing input admitted")
	}
}

func TestDiagnosticClusterPairAdmission(t *testing.T) {
	if root := os.Getenv("GOMAD_DIAGNOSTIC_PAIR_CHILD"); root != "" {
		phase, drift := os.Getenv("GOMAD_DIAGNOSTIC_PAIR_PHASE"), os.Getenv("GOMAD_DIAGNOSTIC_PAIR_DRIFT")
		input, memory := filepath.Join(root, "module"), filepath.Join(root, "meminfo")
		temporalWrite(t, input, []byte("admitted"))
		temporalWrite(t, memory, []byte("MemAvailable: 8388608 kB\n"))
		inputs := map[string]string{input: fileHash(t, input)}
		if drift == "before" {
			temporalWrite(t, input, []byte("changed"))
		}
		if drift == "resource" {
			temporalWrite(t, memory, []byte("MemAvailable: 8388607 kB\n"))
		}
		observer := temporalPairObserver{
			Admission: func(t *testing.T, evidence, label string, requests []Request, before bool) {
				if evidence != root || label != phase || len(requests) != 2 {
					t.Fatal("pair boundary lost")
				}
				if err := diagnosticVerifyInputs(inputs); err != nil {
					t.Fatal(err)
				}
				if before {
					if _, err := matchedPrepareAdmitted("linux", memory, "1", func() (Result, error) { return Result{}, nil }); err != nil {
						t.Fatal(err)
					}
				}
			},
			Observe: func(t *testing.T, evidence, label string, requests []Request, results []Result, failures []error) {
				if len(results) != 2 || len(failures) != 2 {
					t.Fatal("pair evidence lost")
				}
				temporalWrite(t, filepath.Join(root, "observed"), []byte(label))
			},
		}
		temporalObservedPair(t, root, phase, []Request{{}, {}}, []temporalPairObserver{observer}, func() ([]Result, []error) {
			temporalWrite(t, filepath.Join(root, "dispatched"), []byte(phase))
			if drift == "after" {
				temporalWrite(t, input, []byte("changed"))
			}
			return []Result{{}, {}}, []error{nil, nil}
		})
		return
	}
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	for _, phase := range []string{"cluster-crash", "cluster-reopened", "cluster-prefix-replay", "cluster-full-replay"} {
		for _, drift := range []string{"none", "before", "after", "resource"} {
			t.Run(phase+"/"+drift, func(t *testing.T) {
				root := t.TempDir()
				ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
				defer cancel()
				child := exec.CommandContext(ctx, executable, "-test.run=^TestDiagnosticClusterPairAdmission$")
				child.Env = append(os.Environ(), "GOMAD_DIAGNOSTIC_PAIR_CHILD="+root, "GOMAD_DIAGNOSTIC_PAIR_PHASE="+phase, "GOMAD_DIAGNOSTIC_PAIR_DRIFT="+drift)
				output, err := child.CombinedOutput()
				if (err == nil) != (drift == "none") {
					t.Fatalf("phase admission: %v\n%s", err, output)
				}
				if ctx.Err() != nil {
					t.Fatal("controlled pair admission timed out", ctx.Err())
				}
				if drift != "none" {
					reason := "diagnostic input changed"
					if drift == "resource" {
						reason = "at least 8GiB MemAvailable"
					}
					if !strings.Contains(string(output), reason) {
						t.Fatalf("pair failed without admission rejection: %s", output)
					}
				}
				_, dispatchErr := os.Stat(filepath.Join(root, "dispatched"))
				if (dispatchErr == nil) != (drift == "none" || drift == "after") {
					t.Fatal("unadmitted pair dispatched")
				}
				_, observedErr := os.Stat(filepath.Join(root, "observed"))
				if (observedErr == nil) != (drift == "none") {
					t.Fatal("unadmitted pair published measurements")
				}
			})
		}
	}
}

func TestDiagnosticAdmissionModuleBindings(t *testing.T) {
	request := Request{ModulePath: "/module", ModuleSHA256: "module-sha", Runtime: &RuntimeControl{Choice: &runnerbackend.ChoiceRequest{}}}
	request.Runtime.Choice.ExecutionIdentity.ToolchainBuildKey = "runtime-key"
	module := diagnosticModuleAdmission{Schema: "gomad-diagnostic-module-build/v1", Module: request.ModulePath, ModuleSHA256: request.ModuleSHA256, RuntimeKey: "runtime-key", RuntimeSHA256: toolchain.ImplementationSHA256(), ModelSHA256: ImplementationSHA256()}
	if !diagnosticModuleMatches(module, request) {
		t.Fatal("matching module rejected")
	}
	for _, name := range []string{"schema", "module", "digest", "runtime-key", "runtime-source", "model"} {
		t.Run(name, func(t *testing.T) {
			changed := module
			switch name {
			case "schema":
				changed.Schema = "gomad-wasm.provenance/v1"
			case "module":
				changed.Module = "/other"
			case "digest":
				changed.ModuleSHA256 = "stale"
			case "runtime-key":
				changed.RuntimeKey = "stale"
			case "runtime-source":
				changed.RuntimeSHA256 = "stale"
			case "model":
				changed.ModelSHA256 = "stale"
			}
			if diagnosticModuleMatches(changed, request) {
				t.Fatal("stale module admitted")
			}
		})
	}
	request.Runtime = nil
	if diagnosticModuleMatches(module, request) {
		t.Fatal("missing runtime admitted")
	}
}

func TestDiagnosticAdmissionHelperEvidence(t *testing.T) {
	if root := os.Getenv("GOMAD_DIAGNOSTIC_ADMISSION_CHILD"); root != "" {
		a := &diagnosticAdmission{Repo: root, Helpers: map[string]diagnosticHelperAdmission{}, Inputs: map[string]string{}}
		features := []string{"diagnostic-profile", "guest-profile"}
		if os.Getenv("GOMAD_DIAGNOSTIC_ADMISSION_CLUSTER") == "1" {
			features = []string{"diagnostic-profile"}
		}
		a.addHelper(t, filepath.Join(root, "admitted-helper"), filepath.Join(root, "helper-admission.json"), features)
		return
	}
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	for _, mutation := range []string{"valid", "cluster-valid", "cluster-extra-feature", "artifact", "helper", "source", "compiler", "receipt", "wrapper", "wrong-compiler", "missing-feature", "extra-feature", "raw-command"} {
		t.Run(mutation, func(t *testing.T) {
			root := t.TempDir()
			crate := filepath.Join(root, "tools/gomad/internal/backends/wasm/wasmhost")
			if err := os.MkdirAll(filepath.Join(crate, "src"), 0700); err != nil {
				t.Fatal(err)
			}
			for _, name := range []string{"src/main.rs", "Cargo.toml", "Cargo.lock", "rust-toolchain.toml", "../Makefile"} {
				temporalWrite(t, filepath.Join(crate, name), []byte(name))
			}
			rustc, cargo := filepath.Join(root, "rustc"), filepath.Join(root, "cargo")
			artifact, helper := filepath.Join(root, "artifact"), filepath.Join(root, "admitted-helper")
			for _, path := range []string{rustc, cargo, artifact, helper} {
				if err := os.WriteFile(path, []byte("fixture; never executed"), 0755); err != nil {
					t.Fatal(err)
				}
			}
			features := []string{"diagnostic-profile", "guest-profile"}
			if mutation == "cluster-valid" {
				features = []string{"diagnostic-profile"}
			}
			command := []string{cargo, "build", "--manifest-path", filepath.Join(crate, "Cargo.toml"), "--release", "--locked", "--jobs=1", "--no-default-features", "--message-format=json", "--features", strings.Join(features, ",")}
			encoded, err := json.Marshal(artifact)
			if err != nil {
				t.Fatal(err)
			}
			encodedFeatures, err := json.Marshal(features)
			if err != nil {
				t.Fatal(err)
			}
			temporalWrite(t, filepath.Join(root, "cargo-build.stdout"), []byte(`{"reason":"compiler-artifact","target":{"name":"gomad3-wasmhost"},"profile":{"opt_level":"3","debug_assertions":false},"executable":`+string(encoded)+`,"features":`+string(encodedFeatures)+`} {"reason":"build-finished","success":true}`))
			rawCommand := slices.Clone(command)
			if mutation == "raw-command" {
				rawCommand[6] = "--jobs=8"
			}
			temporalJSON(t, filepath.Join(root, "cargo-build.json"), struct {
				Command  []string
				ExitCode int
			}{rawCommand, 0})
			temporalWrite(t, filepath.Join(root, "rustc-version.stdout"), []byte("rustc 1.94.1 fixture"))
			temporalWrite(t, filepath.Join(root, "cargo-version.stdout"), []byte("cargo 1.94.1 fixture"))
			inputs := terminalExitRustInputs(t, crate)
			for _, path := range []string{rustc, cargo, filepath.Join(root, "cargo-build.stdout"), filepath.Join(root, "cargo-build.json"), filepath.Join(root, "rustc-version.stdout"), filepath.Join(root, "cargo-version.stdout")} {
				inputs[path] = fileHash(t, path)
			}
			build := terminalExitHelperBuild{Crate: crate, Helper: artifact, HelperSHA256: fileHash(t, artifact), RustcPath: rustc, Inputs: inputs, Command: command, Environment: []string{"RUSTC=" + rustc, "RUSTC_WRAPPER=", "RUSTC_WORKSPACE_WRAPPER="}, Features: features, RustcVersion: "rustc 1.94.1 fixture", CargoVersion: "cargo 1.94.1 fixture"}
			switch mutation {
			case "wrapper":
				build.Environment[1] = "RUSTC_WRAPPER=/wrong"
			case "wrong-compiler":
				build.Environment[0] = "RUSTC=/wrong"
			case "missing-feature":
				build.Features = []string{"guest-profile"}
			case "extra-feature":
				build.Features = append(build.Features, "unexpected")
			}
			receipt := filepath.Join(root, "helper-build.json")
			temporalJSON(t, receipt, build)
			temporalJSON(t, filepath.Join(root, "helper-admission.json"), terminalExitHelperAdmission{BuildReceipt: receipt, BuildReceiptSHA256: fileHash(t, receipt), CargoArtifact: artifact, CargoArtifactSHA256: fileHash(t, artifact), Helper: helper, HelperSHA256: fileHash(t, helper)})
			changed := map[string]string{"artifact": artifact, "helper": helper, "source": filepath.Join(crate, "src/main.rs"), "compiler": rustc, "receipt": receipt}[mutation]
			if changed != "" {
				temporalWrite(t, changed, []byte("tampered"))
			}
			ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
			defer cancel()
			child := exec.CommandContext(ctx, executable, "-test.run=^TestDiagnosticAdmissionHelperEvidence$")
			child.Env = append(os.Environ(), "GOMAD_DIAGNOSTIC_ADMISSION_CHILD="+root)
			if strings.HasPrefix(mutation, "cluster-") {
				child.Env = append(child.Env, "GOMAD_DIAGNOSTIC_ADMISSION_CLUSTER=1")
			}
			output, err := child.CombinedOutput()
			if (err == nil) != (mutation == "valid" || mutation == "cluster-valid") {
				t.Fatalf("admission for %s: %v\n%s", mutation, err, output)
			}
		})
	}
}

func TestDiagnosticAdmissionCargoFeatures(t *testing.T) {
	for _, features := range [][]string{{"diagnostic-profile"}, {"diagnostic-profile", "guest-profile"}, {"diagnostic-fuel-off", "diagnostic-profile"}} {
		for _, test := range []struct {
			name     string
			actual   []string
			accepted bool
		}{
			{"exact", features, true},
			{"missing", features[:len(features)-1], false},
			{"missing-first", features[1:], false},
			{"extra", append(append([]string{}, features...), "unexpected"), false},
		} {
			t.Run(features[0]+"/"+test.name, func(t *testing.T) {
				encoded, err := json.Marshal(test.actual)
				if err != nil {
					t.Fatal(err)
				}
				data := []byte(`{"reason":"compiler-artifact","target":{"name":"gomad3-wasmhost"},"profile":{"opt_level":"3","debug_assertions":false},"executable":"/release/helper","features":` + string(encoded) + `} {"reason":"build-finished","success":true}`)
				_, err = terminalExitCargoHelperFeatures(data, features)
				if (err == nil) != test.accepted {
					t.Fatalf("features %v admitted=%t", test.actual, err == nil)
				}
				if _, err := terminalExitCargoHelper(data); err == nil && len(test.actual) != 0 {
					t.Fatal("release wrapper admitted diagnostic features")
				}
			})
		}
	}
}

type diagnosticClusterInstance struct {
	Profile     *diagnosticReport `json:"profile"`
	Unavailable string            `json:"unavailable,omitempty"`
}

type diagnosticClusterPair struct {
	Instances   []diagnosticClusterInstance `json:"instances"`
	Measured    int                         `json:"measured_helpers"`
	Complete    bool                        `json:"complete"`
	MeasuredSum *diagnosticReport           `json:"measured_sum"`
}

func decodeClusterDiagnosticPair(requests []Request, results []Result) (diagnosticClusterPair, error) {
	pair := diagnosticClusterPair{Instances: make([]diagnosticClusterInstance, len(results)), Complete: true}
	if len(requests) != len(results) || len(results) == 0 {
		pair.Complete = false
		return pair, fmt.Errorf("cluster request/result cardinality")
	}
	var failures []string
	var overflow bool
	add := func(a, b uint64) uint64 {
		if b > math.MaxUint64-a {
			overflow = true
			failures = append(failures, "cluster diagnostic sum overflow")
			return 0
		}
		return a + b
	}
	for index, result := range results {
		profile, err := decodeDiagnosticReport(result.helperStderr)
		if err != nil {
			pair.Complete = false
			pair.Instances[index].Unavailable = err.Error()
			if result.Termination != "crash" || !result.Reaped || len(result.helperStderr) > 16<<10 {
				failures = append(failures, fmt.Sprintf("helper %d: %v", index, err))
			}
			continue
		}
		if pair.Measured != 0 && (profile.HostOS != pair.MeasuredSum.HostOS || profile.HostArch != pair.MeasuredSum.HostArch) {
			pair.Complete, pair.MeasuredSum = false, nil
			return pair, fmt.Errorf("cluster diagnostic host mismatch")
		}
		if pair.MeasuredSum == nil {
			pair.MeasuredSum = &diagnosticReport{Schema: "gomad3.wasm-profile/v1", DiagnosticOnly: true, Backend: "cranelift-jit", Phases: map[string]diagnosticPhase{}, ABI: map[string]diagnosticABI{}, Nested: map[string]diagnosticNested{}}
		}
		pair.Instances[index].Profile = &profile
		pair.Measured++
		pair.MeasuredSum.HostOS, pair.MeasuredSum.HostArch = profile.HostOS, profile.HostArch
		for name, bucket := range profile.Phases {
			sum := pair.MeasuredSum.Phases[name]
			sum.Count, sum.WallNanos = add(sum.Count, bucket.Count), add(sum.WallNanos, bucket.WallNanos)
			sum.UserNanos, sum.SystemNanos = add(sum.UserNanos, bucket.UserNanos), add(sum.SystemNanos, bucket.SystemNanos)
			sum.CPUSamples = add(sum.CPUSamples, bucket.CPUSamples)
			sum.PeakRSSBytesHighWater = max(sum.PeakRSSBytesHighWater, bucket.PeakRSSBytesHighWater)
			pair.MeasuredSum.Phases[name] = sum
		}
		for name, bucket := range profile.ABI {
			sum := pair.MeasuredSum.ABI[name]
			sum.Count, sum.WallNanos = add(sum.Count, bucket.Count), add(sum.WallNanos, bucket.WallNanos)
			sum.InstantiateWallNanos, sum.ExecuteWallNanos = add(sum.InstantiateWallNanos, bucket.InstantiateWallNanos), add(sum.ExecuteWallNanos, bucket.ExecuteWallNanos)
			pair.MeasuredSum.ABI[name] = sum
		}
		for name, bucket := range profile.Nested {
			sum := pair.MeasuredSum.Nested[name]
			sum.Count, sum.WallNanos = add(sum.Count, bucket.Count), add(sum.WallNanos, bucket.WallNanos)
			pair.MeasuredSum.Nested[name] = sum
		}
		pair.MeasuredSum.Residual.InstantiateWallNanos = add(pair.MeasuredSum.Residual.InstantiateWallNanos, profile.Residual.InstantiateWallNanos)
		pair.MeasuredSum.Residual.ExecuteWallNanos = add(pair.MeasuredSum.Residual.ExecuteWallNanos, profile.Residual.ExecuteWallNanos)
	}
	if len(failures) != 0 {
		pair.Complete = false
		if overflow {
			pair.MeasuredSum = nil
		}
		return pair, fmt.Errorf("%s", strings.Join(failures, "; "))
	}
	return pair, nil
}

func TestDiagnosticClusterPairProfiles(t *testing.T) {
	requests := []Request{{HelperSHA256: "helper", ModuleSHA256: "module"}, {HelperSHA256: "helper", ModuleSHA256: "module"}}
	exit := uint32(0)
	valid := bytes.Replace(diagnosticFixture(t), []byte(`"encoding":{"count":0,"wall_nanos":0}`), []byte(`"encoding":{"count":1,"wall_nanos":5}`), 1)
	for _, test := range []struct {
		name      string
		second    Result
		complete  bool
		measured  int
		wall      uint64
		wantError bool
	}{
		{"both measured", Result{Termination: "exit", ExitCode: &exit, helperStderr: valid}, true, 2, 200, false},
		{"killed helper unavailable", Result{Termination: "crash", Reaped: true}, false, 1, 100, false},
		{"partial killed report unavailable", Result{Termination: "crash", Reaped: true, helperStderr: []byte("gomad-wasm-profile-v1 {")}, false, 1, 100, false},
		{"successful helper missing profile", Result{Termination: "exit", ExitCode: &exit}, false, 1, 100, true},
		{"successful helper malformed profile", Result{Termination: "exit", ExitCode: &exit, helperStderr: []byte("invalid\n")}, false, 1, 100, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			results := []Result{{Termination: "exit", ExitCode: &exit, helperStderr: valid}, test.second}
			pair, err := decodeClusterDiagnosticPair(requests, results)
			if (err != nil) != test.wantError || pair.Complete != test.complete || pair.Measured != test.measured || pair.MeasuredSum.Phases["execute"].WallNanos != test.wall {
				t.Fatalf("pair=%+v err=%v", pair, err)
			}
			if pair.Instances[0].Profile == nil || pair.Instances[0].Profile.Phases["execute"].UserNanos != 70 || pair.Instances[0].Profile.ABI["clock"].ExecuteWallNanos != 30 || pair.Instances[0].Profile.Residual.ExecuteWallNanos != 70 {
				t.Fatal("per-helper buckets lost")
			}
			if pair.MeasuredSum.Phases["execute"].PeakRSSBytesHighWater != 1024 {
				t.Fatal("process high-water values were added as concurrent RSS")
			}
			if pair.MeasuredSum.Phases["execute"].UserNanos != uint64(test.measured)*70 || pair.MeasuredSum.Phases["execute"].SystemNanos != uint64(test.measured)*10 || pair.MeasuredSum.ABI["clock"].ExecuteWallNanos != uint64(test.measured)*30 || pair.MeasuredSum.Nested["encoding"].WallNanos != uint64(test.measured)*5 || pair.MeasuredSum.Residual.ExecuteWallNanos != uint64(test.measured)*70 {
				t.Fatal("measured phase, ABI, nested or residual buckets not summed")
			}
			if !test.complete && (pair.Instances[1].Profile != nil || pair.Instances[1].Unavailable == "") {
				t.Fatal("missing helper measurements presented as zero or available")
			}
		})
	}
}

func TestDiagnosticClusterPairWithoutMeasurements(t *testing.T) {
	pair, err := decodeClusterDiagnosticPair([]Request{{}, {}}, []Result{{Termination: "crash", Reaped: true}, {Termination: "crash", Reaped: true}})
	if err != nil || pair.Complete || pair.Measured != 0 || !reflect.DeepEqual(pair.MeasuredSum, (*diagnosticReport)(nil)) {
		t.Fatalf("unavailable measurements acquired zero-filled buckets: pair=%+v err=%v", pair, err)
	}
}

func TestDiagnosticClusterPairRejectsOverflow(t *testing.T) {
	valid := diagnosticFixture(t)
	overflow := bytes.Replace(valid, []byte(`"count":1`), []byte(`"count":18446744073709551615`), 1)
	_, err := decodeClusterDiagnosticPair([]Request{{}, {}}, []Result{{helperStderr: overflow}, {helperStderr: valid}})
	if err == nil {
		t.Fatal("overflowing summed helper counters accepted")
	}
}

// diagnosticClusterCollectorSources are the package files holding the cluster
// baseline collector; they are bound into its retained report.
var diagnosticClusterCollectorSources = []string{"integration_test.go"}

// warmPreparationDriverSource is the repository path of the warm-preparation
// driver bound into every candidate series.
const warmPreparationDriverSource = "tools/gomad/internal/backends/wasm/wasi/integration_test.go"

// Env-gated drivers hash their own source; a moved owner file would otherwise
// fail them only when the real measurement is attempted.
func TestDiagnosticDriverSourcesExist(t *testing.T) {
	repo, err := filepath.Abs("../../../../../..")
	if err != nil {
		t.Fatal(err)
	}
	paths := []string{filepath.Join(repo, warmPreparationDriverSource)}
	for _, name := range diagnosticClusterCollectorSources {
		path, err := filepath.Abs(name)
		if err != nil {
			t.Fatal(err)
		}
		paths = append(paths, path)
	}
	for _, path := range paths {
		if _, err := os.Stat(path); err != nil {
			t.Fatal("driver source binding names a missing file", err)
		}
	}
}

func TestDiagnosticClusterBaseline(t *testing.T) {
	base := os.Getenv("GOMAD_WASM_TEMPORAL_RECOVERY_ROOT")
	if base == "" {
		t.Skip("set GOMAD_WASM_TEMPORAL_RECOVERY_ROOT for the unsampled cluster baseline")
	}
	helper := os.Getenv("GOMAD_WASM_TEST_HELPER")
	if !filepath.IsAbs(base) || filepath.Clean(base) != base || base == string(filepath.Separator) || !filepath.IsAbs(helper) || !filepath.IsAbs(os.Getenv("GOMAD3_STOCK_GO")) {
		t.Fatal("baseline root, diagnostic-profile helper and stock Go compiler must be explicit absolute paths")
	}
	repo, err := filepath.Abs("../../../../../..")
	if err != nil {
		t.Fatal(err)
	}
	relative, err := filepath.Rel(repo, base)
	if err != nil || relative != ".." && !strings.HasPrefix(relative, ".."+string(os.PathSeparator)) {
		t.Fatal("raw cluster baseline evidence must be retained outside the repository")
	}
	cpuinfo, admitted := diagnosticPerformanceHostEvidence("/proc/cpuinfo")
	if !admitted {
		t.Fatal("cluster baseline requires a real Linux/arm64 CPU witness")
	}
	prerequisites := map[string]string{}
	for index := 1; index <= 12; index++ {
		name := fmt.Sprintf(".flow/tasks/fn-154-gomad-simpler-virtual-network-with.%d.json", index)
		data, err := hostfs.ReadBounded(filepath.Join(repo, name), 1<<20)
		var task struct {
			Status string `json:"status"`
		}
		if err != nil || json.Unmarshal(data, &task) != nil || task.Status != "done" {
			t.Fatalf("cluster baseline requires completed post-fn-154 prerequisite %s", name)
		}
		prerequisites[name] = digest(data)
	}
	commit, err := exec.Command("git", "-C", repo, "rev-parse", "HEAD").Output()
	if err != nil {
		t.Fatal(err)
	}
	status, err := exec.Command("git", "-C", repo, "status", "--porcelain").Output()
	if err != nil {
		t.Fatal(err)
	}
	inputs := map[string]string{}
	for _, name := range diagnosticClusterCollectorSources {
		path, err := filepath.Abs(name)
		if err != nil {
			t.Fatal(err)
		}
		inputs[path] = fileHash(t, path)
	}
	parentExecutable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	parentSHA256 := fileHash(t, parentExecutable)
	root, err := diagnosticRunRoot(base, "cluster-baseline-")
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("retaining cluster baseline in %s", root)
	admission := newDiagnosticAdmission(t, repo, root)
	admission.addHelper(t, helper, os.Getenv("GOMAD_WASM_DIAGNOSTIC_HELPER_ADMISSION"), []string{"diagnostic-profile"})
	admission.Inputs[parentExecutable] = parentSHA256
	maps.Copy(admission.Inputs, inputs)
	for name, hash := range prerequisites {
		admission.Inputs[filepath.Join(repo, name)] = hash
	}
	t.Setenv("GOMAD_WASM_TEMPORAL_RECOVERY_ROOT", root)
	t.Setenv("GOMAD_WASM_PROTOTYPE_ROOT", root)
	// A killed helper cannot prove its diagnostic feature by emitting a final report.
	probeModule := filepath.Join(root, "diagnostic-feature-probe.wasm")
	admission.buildModuleTarget(t, filepath.Join(repo, "tools/gomad"), probeModule, "", "./internal/backends/wasm/toolchain/testdata/choices", "build", nil)
	probe := cooperativeModuleRequest(t, probeModule, admission.Modules[probeModule].RuntimeKey)
	probeResult := func() Result {
		admission.check(t, root, "diagnostic-feature-probe-before", probe, true)
		defer admission.check(t, root, "diagnostic-feature-probe-after", probe, false)
		return runDiagnosticClusterFeatureProbe(t, root, probe)
	}()
	request, evidence := temporalRecoveryRequestWithAdmission(t, "temporal_broker", admission)
	var pairs []map[string]any
	observe := func(t *testing.T, evidence, label string, requests []Request, results []Result, runErrors []error) {
		pair, decodeErr := decodeClusterDiagnosticPair(requests, results)
		var instances []map[string]any
		for index, result := range results {
			name := requests[index].Broker.Node.ID + "-" + label
			if len(result.helperStderr) > 64<<10 {
				t.Fatal("cluster helper stderr capture exceeds transport bound")
			}
			temporalWrite(t, filepath.Join(evidence, name+"-helper-stderr.txt"), result.helperStderr)
			failure := ""
			if runErrors[index] != nil {
				failure = runErrors[index].Error()
			}
			instances = append(instances, map[string]any{"node": requests[index].Broker.Node.ID, "seed": requests[index].Runtime.Seed,
				"identities": variantIdentities(requests[index], result), "termination": result.Termination, "exit_code": result.ExitCode,
				"run_error": failure, "profile": pair.Instances[index], "phase_metrics": result.Telemetry,
				"wait_resource_usage": result.HelperResourceUsage, "authorization_to_terminal_nanos": result.AuthorizationToTerminalNanos})
		}
		measurement := map[string]any{"label": label, "instances": instances, "measured_sum": pair.MeasuredSum,
			"measured_helpers": pair.Measured, "expected_helpers": len(results), "complete": pair.Complete}
		if decodeErr != nil {
			measurement["diagnostic_error"] = decodeErr.Error()
			t.Error(decodeErr)
		}
		pairs = append(pairs, measurement)
		temporalJSON(t, filepath.Join(evidence, label+"-diagnostic-profile.json"), measurement)
	}
	profilePath := filepath.Join(root, "cluster-parent.pprof")
	profile, err := os.OpenFile(profilePath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		t.Fatal(err)
	}
	if err := pprof.StartCPUProfile(profile); err != nil {
		_ = profile.Close()
		t.Fatal(err)
	}
	t.Cleanup(func() {
		pprof.StopCPUProfile()
		if err := profile.Close(); err != nil {
			t.Error(err)
		}
		summary, summaryErr := exec.Command(stockgo.Command(), "tool", "pprof", "-top", profilePath).CombinedOutput()
		temporalWrite(t, filepath.Join(root, "cluster-parent-pprof-top.txt"), summary)
		if summaryErr != nil {
			t.Error(summaryErr)
		}
		temporalVerifyInputs(t, inputs)
		admission.check(t, root, "cluster-final", request, false)
		summaryFailure := ""
		if summaryErr != nil {
			summaryFailure = summaryErr.Error()
		}
		environment := map[string]string{"GOMAD_WASM_TEMPORAL_RECOVERY_ROOT": base, "GOMAD_WASM_TEST_HELPER": helper, "GOMAD_WASM_DIAGNOSTIC_HELPER_ADMISSION": admission.Helpers[helper].Receipt, "GOMAD3_STOCK_GO": os.Getenv("GOMAD3_STOCK_GO"), "PATH": os.Getenv("PATH")}
		for _, key := range []string{"GOMAD_WASM_TEMPORAL_RECOVERY_COMPILE_CACHE", "GOMAD_WASM_TEMPORAL_RECOVERY_DIAGNOSTIC_BUDGET", "GOMAD_WASM_TEMPORAL_RECOVERY_EXPECT_MODULE_SHA256", "GOMAD_WASM_TOOLCHAIN_CACHE"} {
			environment[key] = os.Getenv(key)
		}
		temporalJSON(t, filepath.Join(root, "cluster-baseline-report.json"), map[string]any{
			"host_admission": admission.Receipt, "host_admission_sha256": admission.Inputs[admission.Receipt], "helper_admission": admission.Helpers[helper].Receipt, "helper_admission_sha256": admission.Inputs[admission.Helpers[helper].Receipt], "module_build_receipts": admission.ModuleReceipts,
			"diagnostic_only": true, "source_commit": strings.TrimSpace(string(commit)), "source_dirty": len(status) != 0, "source_status": string(status), "collector_sources": inputs,
			"prerequisite_receipts": prerequisites, "host_goos": runtime.GOOS, "host_goarch": runtime.GOARCH, "go_version": runtime.Version(), "host_cpuinfo": cpuinfo,
			"required_host_measured": admitted, "seeds": []uint64{7}, "selection": "TestTemporalIndependentClustersPartitionHealWholeClusterCrash", "pairs": pairs,
			"scenario_passed": !t.Failed() && len(pairs) == 4, "diagnostic_feature_probe": variantIdentities(probe, probeResult), "identities": variantIdentities(request, Result{}),
			"module_provenance_sha256": fileHash(t, filepath.Join(root, "temporal_broker-provenance.json")),
			"parent_pprof_sha256":      fileHash(t, profilePath), "parent_pprof_summary_sha256": digest(summary), "parent_pprof_summary_error": summaryFailure, "parent_executable_sha256": parentSHA256,
			"parent_pprof_summary_command": []string{stockgo.Command(), "tool", "pprof", "-top", profilePath},
			"scope":                        "unsampled diagnostic-profile helpers; parent CPU profile includes scenario, recovery, replay, per-pair admission checks and evidence publication; module preparation, feature probe and final admission excluded",
			"bucket_scope":                 "measured_sum excludes unavailable helper profiles; nested buckets overlap ABI/phase spans; RSS high-water is maximum per-process high-water, not concurrent cluster RSS; summed helper wall spans overlap in real time",
			"rerun": map[string]any{"working_directory": filepath.Join(repo, "tools", "gomad"), "environment": environment,
				"command": "env -u GOROOT GOENV=off GOTOOLCHAIN=local GOWORK=off go test -tags test_dep -count=1 -timeout 90m -run '^TestDiagnosticClusterBaseline$' ./internal/backends/wasm/wasi"},
		})
	})
	temporalIndependentClustersPartitionHealWholeClusterCrash(t, request, evidence, temporalPairObserver{
		Admission: func(t *testing.T, root, label string, requests []Request, before bool) {
			boundary := "after"
			if before {
				boundary = "before"
			}
			for index, request := range requests {
				admission.check(t, root, fmt.Sprintf("%s-%d-%s", label, index, boundary), request, before)
			}
		},
		Observe: observe,
	})
}

func runDiagnosticClusterFeatureProbe(t *testing.T, root string, request Request) Result {
	t.Helper()
	result, err := Run(t.Context(), request)
	temporalWrite(t, filepath.Join(root, "diagnostic-feature-probe-helper-stderr.txt"), result.helperStderr)
	if err != nil {
		t.Fatal(err)
	}
	requireExit(t, "diagnostic feature probe", result, 0)
	profile, err := decodeDiagnosticReport(result.helperStderr)
	if err != nil {
		t.Fatal(err)
	}
	temporalJSON(t, filepath.Join(root, "diagnostic-feature-probe-profile.json"), profile)
	return result
}

func cooperativeModuleRequest(t *testing.T, module, identity string) Request {
	t.Helper()
	data, err := os.ReadFile(module)
	if err != nil {
		t.Fatal(err)
	}
	imports, pages := binaryInventory(t, data)
	helper := helperPath(t)
	config := testConfig()
	config.Profile = CooperativeProfile
	config.Clock.ReadStepNanos = 1000
	execution := choice.ExecutionIdentity{TargetSHA256: sha256.Sum256(data), ToolchainBuildKey: identity, GOOS: "wasip1", GOARCH: "wasm", ImplementationSHA256: sha256.Sum256([]byte(identity))}
	cache := t.TempDir()
	if err := os.Chmod(cache, 0700); err != nil {
		t.Fatal(err)
	}
	return Request{HelperPath: helper, HelperSHA256: fileHash(t, helper), ModulePath: module, ModuleSHA256: fileHash(t, module), Engine: PinnedEngine(), Imports: imports, InitialMemoryPages: pages, MemoryBytes: 256 << 20, Fuel: 4000000000, Config: config, Timeout: time.Minute, CompileCache: cache, Runtime: &RuntimeControl{Seed: 7, Diagnostics: true, Choice: &runnerbackend.ChoiceRequest{Mode: choice.ModeRecord, ExecutionIdentity: execution, Limit: 1 << 20}}}
}

func temporalRecoveryRequestWithAdmission(t *testing.T, fixture string, admission *diagnosticAdmission) (Request, string) {
	t.Helper()
	evidence := os.Getenv("GOMAD_WASM_TEMPORAL_RECOVERY_ROOT")
	if evidence == "" {
		t.Skip("set GOMAD_WASM_TEMPORAL_RECOVERY_ROOT for the actual Temporal recovery gate")
	}
	if !filepath.IsAbs(evidence) {
		t.Fatal("GOMAD_WASM_TEMPORAL_RECOVERY_ROOT must be absolute")
	}
	started, before := time.Now(), hostexec.CurrentProcessResourceUsage()
	t.Setenv("GOMAD_WASM_PROTOTYPE_ROOT", evidence)
	root, err := filepath.Abs("../../../../../..")
	if err != nil {
		t.Fatal(err)
	}
	fixtureBody, err := os.ReadFile(filepath.Join(root, "tools/gomad/internal/backends/wasm/testdata/temporal_storage/recovery_test.go"))
	if err != nil {
		t.Fatal(err)
	}
	if fixture == "temporal_broker" {
		brokerBody, err := os.ReadFile(filepath.Join(root, "tools/gomad/internal/backends/wasm/testdata/temporal_broker/replication_test.go"))
		if err != nil {
			t.Fatal(err)
		}
		fixtureBody = append(fixtureBody, brokerBody...)
	}
	staged := filepath.Join(".tmp", "wasm-backend-temporal", fixture+"-"+digest(fixtureBody)[:16])
	if fixture == "temporal_broker" {
		temporalStageBrokerFixture(t, root, staged)
	} else {
		if err := temporalStageFixture(root, filepath.Join("tools", "gomad", "internal", "backends", "wasm", "testdata", fixture), root, staged); err != nil {
			t.Fatal(err)
		}
	}
	compiler, overlay, runtimeIdentity := prototypeToolchain(t)
	listed := temporalCommand(t, evidence, fixture+"-inputs", root, temporalBuildEnvironment(compiler), []string{
		compiler, "list", "-p=1", "-mod=readonly", "-tags", "test_dep,disable_grpc_modules,sqlite3_dotlk", "-deps", "-test", "-json", "./" + filepath.ToSlash(staged),
	})
	inputs := temporalSourceInputs(t, listed.Stdout.RawBytes)
	inputs[overlay] = fileHash(t, overlay)
	overlayBytes, err := os.ReadFile(overlay)
	if err != nil {
		t.Fatal(err)
	}
	var overlayMap struct{ Replace map[string]string }
	if err := json.Unmarshal(overlayBytes, &overlayMap); err != nil {
		t.Fatal(err)
	}
	for _, replacement := range overlayMap.Replace {
		if replacement != "" {
			inputs[replacement] = fileHash(t, replacement)
		}
	}
	for _, name := range []string{"go.mod", "go.sum", "tools/gomad/internal/backends/wasm/testdata/temporal_storage/recovery_test.go"} {
		inputs[filepath.Join(root, name)] = fileHash(t, filepath.Join(root, name))
	}
	if fixture == "temporal_broker" {
		file := filepath.Join(root, "tools/gomad/internal/backends/wasm/testdata/temporal_broker/replication_test.go")
		inputs[file] = fileHash(t, file)
	}
	t.Cleanup(func() { temporalVerifyInputs(t, inputs) })
	module := filepath.Join(evidence, fixture+".wasm")
	if admission == nil {
		temporalCommand(t, evidence, fixture+"-compile", root,
			append(temporalBuildEnvironment(compiler), "GOROOT="+filepath.Dir(filepath.Dir(compiler))),
			[]string{compiler, "test", "-c", "-p=1", "-mod=readonly", "-trimpath", "-overlay", overlay, "-o", module,
				"-tags", "test_dep,disable_grpc_modules,sqlite3_dotlk", "./" + filepath.ToSlash(staged)})
	} else {
		admission.buildModuleTarget(t, root, module, "", "./"+filepath.ToSlash(staged), "test", []string{"test_dep", "disable_grpc_modules", "sqlite3_dotlk"})
	}
	temporalVerifyInputs(t, inputs)
	if expected := os.Getenv("GOMAD_WASM_TEMPORAL_RECOVERY_EXPECT_MODULE_SHA256"); expected != "" && fileHash(t, module) != expected {
		t.Fatalf("prepared Temporal guest differs from retained module: expected %s, actual %s", expected, fileHash(t, module))
	}
	temporalJSON(t, filepath.Join(evidence, fixture+"-provenance.json"), struct {
		Inputs                                      map[string]string
		ModuleSHA256, HelperSHA256, RuntimeIdentity string
		Engine                                      EngineIdentity
	}{inputs, fileHash(t, module), fileHash(t, helperPath(t)), runtimeIdentity, PinnedEngine()})
	captured, err := captured.CaptureReadOnlyMountInputs([]captured.Mapping{{Source: filepath.Join(root, "schema", "sqlite"), Target: "/workspace/schema/sqlite"}}, captured.DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	cfg := Config{
		WorkingDirectory:    "/workspace",
		Environment:         []string{"PWD=/workspace", "TEMPORAL_ROOT=/workspace", "TMPDIR=/tmp", "TZ=UTC", "GOMAD_TEMPORAL_DATABASE=/volume/temporal.db", "GOMAD_TEMPORAL_SCHEMA=/workspace/schema/sqlite/v3"},
		WritableDirectories: []string{"/workspace", "/tmp", "/volume"}, CapturedInputs: captured,
		EntropyKey: sha256.Sum256([]byte("gomad-wasm-temporal-recovery/v1")),
		Clock:      ClockPolicy{EpochNanos: 1700000000000000000, ReadStepNanos: 0},
		Limits:     Limits{OutputBytes: 8 << 20, TranscriptBytes: 128 << 20, Calls: 4000000, Descriptors: 4096, Files: 10000, FilesystemBytes: 512 << 20, PendingEvents: 4096},
		Storage:    &StorageModel{Volumes: []volume.VolumeConfig{{ID: "temporal", Path: "/volume", CapacityBytes: 256 << 20}}, Limits: volume.VolumeLimits{PendingOperations: 100000, Transitions: 10000000}, SQLiteLockDatabases: []string{"/volume/temporal.db"}},
	}
	request := prototypeRequest(t, module, cfg, 7, 15*time.Minute)
	request.CompileCache, err = temporalCompileCachePath(evidence, os.Getenv("GOMAD_WASM_TEMPORAL_RECOVERY_COMPILE_CACHE"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(request.CompileCache, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(request.CompileCache, 0700); err != nil {
		t.Fatal(err)
	}
	temporalJSON(t, filepath.Join(evidence, fixture+"-preparation-operator.json"), struct {
		ElapsedNanos  int64
		Before, After *hostexec.ResourceUsage
	}{time.Since(started).Nanoseconds(), before, hostexec.CurrentProcessResourceUsage()})
	return request, evidence
}

type temporalPairObserver struct {
	Admission func(*testing.T, string, string, []Request, bool)
	Observe   func(*testing.T, string, string, []Request, []Result, []error)
}

func temporalIndependentClustersPartitionHealWholeClusterCrash(t *testing.T, request Request, evidence string, observers ...temporalPairObserver) {
	t.Helper()
	request.Config.Args = []string{"replication.test", "-test.v", "-test.count=1", "-test.timeout=1h", "-test.run=^TestIndependentClusterReplicationRecovery$", "-persistenceType=sql", "-persistenceDriver=sqlite"}
	network := &NetworkModelV2{Profile: NetworkStreamProfile, Limits: netmodel.Limits{Listeners: 128, Connections: 4096, Bytes: 128 << 20, ConnectionBytes: 4 << 20, StallNanos: 10_000_000_000, Transitions: 1 << 20}}
	request.Config.NetworkV2 = network
	brokerConfig := temporalBrokerConfiguration(network)
	temporalJSON(t, filepath.Join(evidence, "two-cluster-broker-config.json"), brokerConfig)
	b, err := NewBroker(brokerConfig)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		temporalWrite(t, filepath.Join(evidence, "two-cluster-broker-idle.jsonl"), b.Diagnostics())
		snapshot, err := b.NetworkSnapshot()
		if err != nil {
			t.Error(err)
			return
		}
		temporalJSON(t, filepath.Join(evidence, "two-cluster-network.json"), snapshot)
	})
	progressContext, stopProgress := context.WithCancel(t.Context())
	var progress sync.WaitGroup
	progress.Go(func() {
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-progressContext.Done():
				return
			case <-ticker.C:
				// Host-only diagnostic observation never advances modeled time.
				if err := os.WriteFile(filepath.Join(evidence, "two-cluster-broker-live.jsonl"), b.Diagnostics(), 0600); err != nil {
					t.Error(err)
				}
			}
		}
	})
	t.Cleanup(func() { stopProgress(); progress.Wait() })
	names := []string{"active", "standby"}
	if err := b.CrashAfterOutputs(names, []byte(temporalObservationPrefix)); err != nil {
		t.Fatal(err)
	}
	requests := make([]Request, 2)
	commitRequests := make([]Request, 2)
	for i, name := range names {
		requests[i] = request
		storage := *request.Config.Storage
		storage.Volumes = append([]volume.VolumeConfig(nil), storage.Volumes...)
		storage.Volumes[0].ID = name
		requests[i].Config.Storage = &storage
		requests[i].Config.EntropyKey = sha256.Sum256([]byte("gomad-wasm-temporal-replication/v1/" + name))
		requests[i].Broker = b.Session(name)
		requests[i].Config.Environment = append(append([]string(nil), request.Config.Environment...), "GOMAD_TEMPORAL_CLUSTER="+name, "GOMAD_TEMPORAL_PHASE=commit", "GOMAD_TEMPORAL_REMOTE_FRONTEND="+b.Address(names[1-i])+":7233", "GOMAD_TEMPORAL_REMOTE_HTTP="+b.Address(names[1-i])+":7243", "GOMAD_TEMPORAL_REMOTE_HISTORY="+b.Address(names[1-i])+":7234")
		temporalRetainRequest(t, evidence, name+"-cluster-commit", requests[i])
		commitRequests[i] = requests[i]
		commitRequests[i].Config.Environment = append([]string(nil), requests[i].Config.Environment...)
	}
	results, errors := temporalRunPair(t, evidence, "cluster-crash", requests, true, observers...)
	crashed := append([]Result(nil), results...)
	var expected []byte
	identities := make([]string, 2)
	original := make([]temporalRecoveryObservation, 2)
	converged := map[string][]byte{}
	for i, name := range names {
		if errors[i] != nil || results[i].Termination != "crash" || !results[i].Reaped || results[i].StorageImage == nil {
			t.Fatalf("independent %s crash: termination=%s message=%s reaped=%v image=%v err=%v stdout=%s stderr=%s", name, results[i].Termination, results[i].Message, results[i].Reaped, results[i].StorageImage != nil, errors[i], results[i].Stdout, results[i].Stderr)
		}
		value := temporalRecoveryValue(t, results[i].Stdout)
		original[i] = value
		converged[name] = value.History
		identities[i] = value.ClusterID
		if i == 0 {
			expected = value.History
		} else if !bytes.Equal(expected, value.History) {
			t.Fatal("independently observed histories did not converge after directional heal")
		}
		requests[i].StorageImage = results[i].StorageImage
		requests[i].StdoutMirror = nil
		requests[i].Config.Environment[len(request.Config.Environment)+1] = "GOMAD_TEMPORAL_PHASE=reopen"
		requests[i].Broker, err = b.Restart(name)
		if err != nil {
			t.Fatal(err)
		}
		temporalRetainRequest(t, evidence, name+"-cluster-reopen", requests[i])
	}
	if identities[0] == identities[1] {
		t.Fatal("independent Temporal clusters have the same identity")
	}
	convergence, err := gomad3sim.EventualConvergence("temporal.replication.healed", converged, 1<<20)
	if err != nil || !convergence.Passed {
		t.Fatalf("detached Temporal convergence oracle: passed=%v err=%v", convergence.Passed, err)
	}
	temporalJSON(t, filepath.Join(evidence, "temporal-replication-healed-oracle.json"), convergence)
	temporalJSON(t, filepath.Join(evidence, "temporal-replication-healed-oracle-inputs.json"), converged)
	results, errors = temporalRunPair(t, evidence, "cluster-reopened", requests, false, observers...)
	for i, name := range names {
		if errors[i] != nil || results[i].Termination != "exit" || results[i].ExitCode == nil || *results[i].ExitCode != 0 || !results[i].Reaped {
			t.Fatalf("independent %s reopen: termination=%s message=%s err=%v stdout=%s stderr=%s", name, results[i].Termination, results[i].Message, errors[i], results[i].Stdout, results[i].Stderr)
		}
		value := temporalRecoveryValue(t, results[i].Stdout)
		temporalHistoryOracles(t, evidence, "temporal.replication."+name+"-recovered", original[i], value)
		if !reflect.DeepEqual(expected, value.History) || value.ClusterID != identities[i] {
			t.Fatalf("committed %s history changed after whole-cluster crash", name)
		}
	}
	encodedWorld, err := b.EncodeWorldEvidence()
	if err != nil {
		t.Fatal(err)
	}
	temporalWrite(t, filepath.Join(evidence, "two-cluster-world.json"), encodedWorld)
	temporalJSON(t, filepath.Join(evidence, "two-cluster-world-cardinality.json"), b.DiagnosticCardinality())
	temporalReplayClusters(t, evidence, brokerConfig, commitRequests, crashed, results, encodedWorld, observers...)
	fmt.Printf("independent Temporal clusters recovered after directional partition/heal\n")
}

func temporalObservedPair(t *testing.T, evidence, label string, requests []Request, observers []temporalPairObserver, dispatch func() ([]Result, []error)) ([]Result, []error) {
	t.Helper()
	for _, observer := range observers {
		if observer.Admission != nil {
			observer.Admission(t, evidence, label, requests, true)
		}
	}
	results, failures := func() ([]Result, []error) {
		defer func() {
			for _, observer := range observers {
				if observer.Admission != nil {
					observer.Admission(t, evidence, label, requests, false)
				}
			}
		}()
		return dispatch()
	}()
	for _, observer := range observers {
		if observer.Observe != nil {
			observer.Observe(t, evidence, label, requests, results, failures)
		}
	}
	return results, failures
}

func temporalRunPairExecution(t *testing.T, evidence, label string, requests []Request, warmFirst bool) ([]Result, []error) {
	t.Helper()
	budget, err := temporalPairBudget(os.Getenv("GOMAD_WASM_TEMPORAL_RECOVERY_DIAGNOSTIC_BUDGET"))
	if err != nil {
		t.Fatal(err)
	}
	started := time.Now()
	ctx, cancel := context.WithTimeout(t.Context(), budget)
	defer cancel()
	results, errors := make([]Result, len(requests)), make([]error, len(requests))
	warmed, firstFinished := make(chan struct{}), make(chan struct{})
	var group sync.WaitGroup
	for i := range requests {
		observer := &temporalGuestWarm{test: t, node: requests[i].Broker.Node.ID}
		if i == 0 && warmFirst {
			observer.ready = warmed
		}
		requests[i].StdoutMirror = observer
		if i == 1 && warmFirst {
			select {
			case <-warmed:
			case <-firstFinished:
			case <-ctx.Done():
				// The bounded host budget is not an application terminal.
				cancel()
			}
		}
		group.Go(func() {
			results[i], errors[i] = Run(ctx, requests[i])
			name := requests[i].Broker.Node.ID
			temporalRetainRecoveryRun(t, evidence, name+"-"+label, results[i])
			t.Logf("%s %s terminated: %s %s", name, label, results[i].Termination, results[i].Message)
			if i == 0 {
				close(firstFinished)
			}
			if errors[i] != nil || results[i].Termination != "exit" && results[i].Termination != "crash" ||
				results[i].Termination == "exit" && (results[i].ExitCode == nil || *results[i].ExitCode != 0) {
				cancel()
			}
		})
	}
	group.Wait()
	temporalJSON(t, filepath.Join(evidence, label+"-host-budget.json"), struct {
		BudgetNanos, ElapsedNanos int64
		Expired                   bool
	}{int64(budget), time.Since(started).Nanoseconds(), ctx.Err() == context.DeadlineExceeded})
	return results, errors
}

func terminalExitAdmitHelperFeatures(t *testing.T, root, receipt string, features []string) (string, string) {
	t.Helper()
	var build terminalExitHelperBuild
	if err := json.Unmarshal(mustTerminalExitRead(t, receipt), &build); err != nil {
		t.Fatal(err)
	}
	crate, err := filepath.Abs("../wasmhost")
	if err != nil {
		t.Fatal(err)
	}
	request := Request{HelperPath: build.Helper, HelperSHA256: build.HelperSHA256}
	inputs := terminalExitBuildInputsFeatures(t, receipt, request, crate, features)
	evidence, err := diagnosticRunRoot(root, "admitted-helper-")
	if err != nil {
		t.Fatal(err)
	}
	helper := filepath.Join(evidence, "gomad3-wasmhost")
	terminalExitStageHelper(t, build.Helper, helper, build.HelperSHA256)
	temporalVerifyInputs(t, inputs)
	terminalExitBuildInputsFeatures(t, receipt, request, crate, features)
	admission := filepath.Join(evidence, "helper-admission.json")
	temporalJSON(t, admission, terminalExitHelperAdmission{BuildReceipt: receipt, BuildReceiptSHA256: fileHash(t, receipt), CargoArtifact: build.Helper, CargoArtifactSHA256: build.HelperSHA256, Helper: helper, HelperSHA256: fileHash(t, helper)})
	terminalExitHelperInputsFeatures(t, admission, Request{HelperPath: helper, HelperSHA256: fileHash(t, helper)}, crate, features)
	return helper, admission
}

func terminalExitBuildHelper(t *testing.T, root string, features []string) {
	t.Helper()
	if !filepath.IsAbs(root) {
		t.Fatal("helper build evidence root must be absolute")
	}
	evidence, err := diagnosticRunRoot(root, "release-helper-")
	if err != nil {
		t.Fatal(err)
	}
	crate, err := filepath.Abs("../wasmhost")
	if err != nil {
		t.Fatal(err)
	}
	inputs := terminalExitRustInputs(t, crate)
	engine, err := os.ReadFile(filepath.Join(crate, "src/engine.rs"))
	if err != nil || bytes.Contains(engine, []byte("std::mem::forget")) || !bytes.Contains(engine, []byte("native_unwind_info(false)")) {
		t.Fatal("release helper must use the no-unwind source candidate without the leak", err)
	}
	var environment []string
	for _, entry := range os.Environ() {
		name, _, _ := strings.Cut(entry, "=")
		if slices.Contains([]string{"PATH", "HOME", "CARGO_HOME", "RUSTUP_HOME", "TMPDIR"}, name) {
			environment = append(environment, entry)
		}
	}
	environment = append(environment, "CARGO_TARGET_DIR="+filepath.Join(evidence, "target"))
	rustcPath := strings.TrimSpace(string(temporalCommand(t, evidence, "rustc-path", crate, environment, []string{"rustup", "which", "--toolchain", "1.94.1", "rustc"}).Stdout.RawBytes))
	cargoPath := strings.TrimSpace(string(temporalCommand(t, evidence, "cargo-path", crate, environment, []string{"rustup", "which", "--toolchain", "1.94.1", "cargo"}).Stdout.RawBytes))
	inputs[rustcPath], inputs[cargoPath] = fileHash(t, rustcPath), fileHash(t, cargoPath)
	rustcVersion := temporalCommand(t, evidence, "rustc-version", crate, environment, []string{rustcPath, "-vV"})
	cargoVersion := temporalCommand(t, evidence, "cargo-version", crate, environment, []string{cargoPath, "--version"})
	if !strings.HasPrefix(string(rustcVersion.Stdout.RawBytes), "rustc 1.94.1 ") || !strings.HasPrefix(string(cargoVersion.Stdout.RawBytes), "cargo 1.94.1 ") {
		t.Fatal("release helper requires pinned Rust/Cargo 1.94.1")
	}
	environment = append(environment, "RUSTC="+rustcPath, "RUSTC_WRAPPER=", "RUSTC_WORKSPACE_WRAPPER=")
	command := terminalExitCargoCommand(cargoPath, crate, features)
	build := temporalCommand(t, evidence, "cargo-build", crate, environment, command)
	helper, err := terminalExitCargoHelperFeatures(build.Stdout.RawBytes, features)
	if err != nil {
		t.Fatal(err)
	}
	if helper != filepath.Join(evidence, "target/release/gomad3-wasmhost") {
		t.Fatal("Cargo did not produce the fresh isolated release executable")
	}
	current := terminalExitRustInputs(t, crate)
	current[rustcPath], current[cargoPath] = fileHash(t, rustcPath), fileHash(t, cargoPath)
	if !maps.Equal(inputs, current) {
		t.Fatal("Rust semantic inputs changed during build")
	}
	for _, name := range []string{"cargo-build.stdout", "cargo-build.json", "rustc-version.stdout", "cargo-version.stdout"} {
		path := filepath.Join(evidence, name)
		inputs[path] = fileHash(t, path)
	}
	receipt := filepath.Join(evidence, "helper-build.json")
	temporalJSON(t, receipt, terminalExitHelperBuild{Crate: crate, Helper: helper, HelperSHA256: fileHash(t, helper), RustcPath: rustcPath, Inputs: inputs, Command: command, Environment: environment, Features: features, RustcVersion: string(rustcVersion.Stdout.RawBytes), CargoVersion: string(cargoVersion.Stdout.RawBytes)})
	admitted, admission := terminalExitAdmitHelperFeatures(t, evidence, receipt, features)
	t.Logf("release Cargo artifact=%s build=%s; admitted helper=%s receipt=%s sha256=%s", helper, receipt, admitted, admission, fileHash(t, admitted))
}

// terminalExitCargoCommand is the one pinned, single-job helper build command;
// diagnostic admission checks retained receipts against the same construction.
func terminalExitCargoCommand(cargo, crate string, features []string) []string {
	command := []string{cargo, "build", "--manifest-path", filepath.Join(crate, "Cargo.toml"), "--release", "--locked", "--jobs=1", "--no-default-features", "--message-format=json"}
	if len(features) != 0 {
		command = append(command, "--features", strings.Join(features, ","))
	}
	return command
}

func terminalExitCargoHelperFeatures(data []byte, features []string) (string, error) {
	decoder := json.NewDecoder(bytes.NewReader(data))
	helper := ""
	finished := false
	for {
		var frame struct {
			Reason     string
			Executable string
			Features   *[]string
			Success    bool
			Target     struct{ Name string }
			Profile    struct {
				OptLevel        string `json:"opt_level"`
				DebugAssertions bool   `json:"debug_assertions"`
			}
		}
		if err := decoder.Decode(&frame); err == io.EOF {
			break
		} else if err != nil {
			return "", err
		}
		if frame.Reason == "compiler-artifact" && frame.Target.Name == "gomad3-wasmhost" && frame.Executable != "" {
			if helper != "" || frame.Features == nil || !slices.Equal(*frame.Features, features) || frame.Profile.OptLevel != "3" || frame.Profile.DebugAssertions {
				return "", fmt.Errorf("duplicate Cargo helper artifact or unexpected release features")
			}
			helper = frame.Executable
		}
		if frame.Reason == "build-finished" {
			finished = frame.Success
		}
	}
	if helper == "" || !finished {
		return "", fmt.Errorf("successful release helper artifact absent from Cargo output")
	}
	return helper, nil
}

func terminalExitBuildInputsFeatures(t *testing.T, receipt string, request Request, crate string, features []string) map[string]string {
	t.Helper()
	data, err := os.ReadFile(receipt)
	if err != nil {
		t.Fatal(err)
	}
	var build terminalExitHelperBuild
	if err := json.Unmarshal(data, &build); err != nil {
		t.Fatal(err)
	}
	if build.Crate != crate || build.Helper != request.HelperPath || build.HelperSHA256 != request.HelperSHA256 || build.Features == nil || !slices.Equal(build.Features, features) || !slices.Contains(build.Command, "--release") || !slices.Contains(build.Command, "--locked") || !slices.Contains(build.Command, "--no-default-features") || !strings.HasPrefix(build.RustcVersion, "rustc 1.94.1 ") || !strings.HasPrefix(build.CargoVersion, "cargo 1.94.1 ") {
		t.Fatal("helper receipt does not bind the pinned release build and measured executable")
	}
	if !terminalExitCompilerEnvironment(build) {
		t.Fatal("Cargo must use the retained pinned rustc executable with compiler wrappers disabled")
	}
	current := terminalExitRustInputs(t, crate)
	for path := range build.Inputs {
		if _, found := current[path]; !found {
			current[path] = fileHash(t, path)
		}
	}
	if !maps.Equal(current, build.Inputs) {
		t.Fatal("helper Rust semantic sources, Cargo inputs or toolchain changed after build")
	}
	cargoOutput, err := os.ReadFile(filepath.Join(filepath.Dir(receipt), "cargo-build.stdout"))
	if err != nil {
		t.Fatal(err)
	}
	if helper, err := terminalExitCargoHelperFeatures(cargoOutput, features); err != nil || helper != request.HelperPath {
		t.Fatal("Cargo artifact does not bind the measured release helper", err)
	}
	inputs := maps.Clone(build.Inputs)
	inputs[receipt], inputs[request.HelperPath] = fileHash(t, receipt), request.HelperSHA256
	temporalVerifyInputs(t, inputs)
	return inputs
}

func terminalExitHelperInputsFeatures(t *testing.T, receipt string, request Request, crate string, features []string) map[string]string {
	t.Helper()
	var admission terminalExitHelperAdmission
	if err := json.Unmarshal(mustTerminalExitRead(t, receipt), &admission); err != nil {
		t.Fatal(err)
	}
	if !filepath.IsAbs(admission.BuildReceipt) || !filepath.IsAbs(admission.CargoArtifact) || admission.CargoArtifact == admission.Helper || admission.BuildReceiptSHA256 != fileHash(t, admission.BuildReceipt) || admission.Helper != request.HelperPath || admission.HelperSHA256 != request.HelperSHA256 || admission.CargoArtifactSHA256 != admission.HelperSHA256 {
		t.Fatal("admission must bind the immutable raw build receipt, Cargo artifact and measured helper")
	}
	inputs := terminalExitBuildInputsFeatures(t, admission.BuildReceipt, Request{HelperPath: admission.CargoArtifact, HelperSHA256: admission.CargoArtifactSHA256}, crate, features)
	file, _, err := hostfs.OpenPath(request.HelperPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	inputs[receipt], inputs[request.HelperPath] = fileHash(t, receipt), request.HelperSHA256
	temporalVerifyInputs(t, inputs)
	return inputs
}

const warmPreparationSchema = "gomad3.wasm-warm-preparation/v1"

type warmPreparationRun struct {
	Telemetry                                   Telemetry                  `json:"telemetry"`
	Phases                                      map[string]diagnosticPhase `json:"phases"`
	WallNanos, TerminalToReapNanos              uint64
	EvidenceSHA256                              string
	Exited, Reaped                              bool
	ObservationPath, FinishPath, DiagnosticPath string
	RawObservationPath                          string
}

// The measurement projection is shared with the test-only historical collector;
// production host receipts remain unchanged and are retained separately.
type warmPreparationObservation struct {
	TerminalReceivedAt, SupervisorReturnedAt          time.Time
	TerminalToReapNanos                               uint64
	TerminalFrame                                     []byte
	Result                                            Result
	Error                                             string
	HostExited, GroupGone, Cancelled, WatchdogTimeout bool
	HostExitCode                                      int
}

func warmPreparationObserve(observation terminalExitObservation) warmPreparationObservation {
	return warmPreparationObservation{TerminalReceivedAt: observation.TerminalReceivedAt, SupervisorReturnedAt: observation.SupervisorReturnedAt, TerminalToReapNanos: observation.TerminalToReapNanos, TerminalFrame: observation.TerminalFrame, Result: observation.Result, Error: observation.Error, HostExited: observation.Host.Termination == "exit", GroupGone: observation.Host.GroupGone, Cancelled: observation.Host.Cancelled, WatchdogTimeout: observation.Host.WatchdogTimeout, HostExitCode: observation.Host.ExitCode}
}

func TestWarmPreparationObservationPrivateFinishControls(t *testing.T) {
	code := uint32(0)
	observation := terminalExitObservation{TerminalReceivedAt: time.Unix(1, 0), SupervisorReturnedAt: time.Unix(1, 1), TerminalToReapNanos: 1, Result: Result{Termination: "exit", ExitCode: &code, runtimeTerminal: [16]byte{1}}}
	observation.Host.Termination, observation.Host.GroupGone = "exit", true
	projection := warmPreparationObserve(observation)
	if !projection.HostExited || !projection.GroupGone || projection.TerminalToReapNanos != 1 || projection.SupervisorReturnedAt != observation.SupervisorReturnedAt {
		t.Fatal("actual terminal/host measurements lost")
	}
	data, err := json.Marshal(projection)
	if err != nil {
		t.Fatal(err)
	}
	var imported warmPreparationObservation
	if err := json.Unmarshal(data, &imported); err != nil {
		t.Fatal(err)
	}
	if warmPreparationEvidence(imported.Result) == warmPreparationEvidence(observation.Result) {
		t.Fatal("JSON unexpectedly preserved private finish; missing sidecar must remain visible")
	}
	imported.Result.runtimeTerminal = observation.Result.runtimeTerminal
	if warmPreparationEvidence(imported.Result) != warmPreparationEvidence(observation.Result) {
		t.Fatal("finish sidecar did not restore exact behavior")
	}
}

type warmPreparationSeries struct {
	Schema, Role, SourceCommit, SourceRoot, DriverSHA256     string
	HostBootID, HostCPUInfo, ConfigSHA256                    string
	Engine                                                   EngineIdentity
	ModuleSHA256, HelperSHA256, RuntimeIdentity, ModelSHA256 string
	Seed                                                     uint64
	ArtifactBytes, EnvelopeBytes                             uint64
	Inputs                                                   map[string]string
	HelperReceipt, ModuleReceipt, DriverPath, ArtifactPath   string
	CollectionPath                                           string
	Runs                                                     []warmPreparationRun
	Warmup                                                   *warmPreparationRun
	ComparisonSHA256                                         string `json:"-"`
}

func warmPreparationValidate(series warmPreparationSeries, candidate bool) error {
	role := "baseline"
	if candidate {
		role = "candidate"
	}
	if series.Schema != warmPreparationSchema || series.Role != role || len(series.Runs) < 5 || !warmPreparationHex(series.SourceCommit, 40) || !warmPreparationHex(series.DriverSHA256, 64) || series.SourceRoot == "" || len(series.Inputs) == 0 || !warmPreparationHex(series.ModuleSHA256, 64) || !warmPreparationHex(series.HelperSHA256, 64) || !warmPreparationHex(series.ConfigSHA256, 64) || series.RuntimeIdentity == "" || series.ModelSHA256 == "" || series.ArtifactBytes == 0 || series.ArtifactBytes >= series.EnvelopeBytes {
		return fmt.Errorf("missing source, build, module, artifact or five-run bindings")
	}
	if series.HostBootID == "" || series.Engine.Name != "wasmtime" || series.Engine.Version != "47.0.3" || series.Engine.Configuration == "" || !diagnosticPerformanceHost(series.Engine.HostOS, "arm64", series.HostCPUInfo) || series.Engine.HostArch != "aarch64" {
		return fmt.Errorf("real Linux/arm64 host binding absent")
	}
	var prepare []uint64
	for i, run := range series.Runs {
		if !run.Telemetry.CacheHit || run.Telemetry.CompileNanos != 0 || run.Telemetry.PrepareNanos == 0 || run.WallNanos == 0 || !run.Exited || !run.Reaped || run.TerminalToReapNanos == 0 || candidate && run.TerminalToReapNanos > uint64(time.Second) || !warmPreparationHex(run.EvidenceSHA256, 64) || run.EvidenceSHA256 != series.Runs[0].EvidenceSHA256 {
			return fmt.Errorf("run %d is not a warm successful evidence-identical reaped run", i)
		}
		for _, phase := range strings.Fields("cache_read envelope_verify deserialize cache_identity") {
			if p, ok := run.Phases[phase]; !ok || p.Count == 0 || p.WallNanos > run.Telemetry.PrepareNanos {
				return fmt.Errorf("run %d phase %s absent or inconsistent", i, phase)
			}
		}
		prepare = append(prepare, run.Telemetry.PrepareNanos)
	}
	if candidate && variantTiming(prepare).Median > uint64(time.Second) {
		return fmt.Errorf("warm preparation median exceeds one second")
	}
	return nil
}

func warmPreparationCompare(before, after warmPreparationSeries) error {
	if err := warmPreparationValidate(before, false); err != nil {
		return fmt.Errorf("baseline: %w", err)
	}
	if err := warmPreparationValidate(after, true); err != nil {
		return fmt.Errorf("candidate: %w", err)
	}
	if before.HostBootID != after.HostBootID || before.HostCPUInfo != after.HostCPUInfo || before.ModuleSHA256 != after.ModuleSHA256 || before.Seed != after.Seed || before.ConfigSHA256 != after.ConfigSHA256 || before.RuntimeIdentity != after.RuntimeIdentity {
		return fmt.Errorf("before/after must share the real host, module, seed, modeled configuration and runtime")
	}
	if !warmPreparationHex(before.ComparisonSHA256, 64) || !warmPreparationHex(after.ComparisonSHA256, 64) || before.ComparisonSHA256 != after.ComparisonSHA256 {
		return fmt.Errorf("choice/WASI/network/stdout/stderr/terminal evidence changed")
	}
	if before.Inputs[before.ModuleReceipt] == "" || before.Inputs[before.ModuleReceipt] != after.Inputs[after.ModuleReceipt] {
		return fmt.Errorf("historical origin must bind the authentic candidate-owner-validated current module producer")
	}
	return nil
}

func warmPreparationHex(value string, size int) bool {
	_, err := hex.DecodeString(value)
	return len(value) == size && err == nil
}

type warmPreparationCollection struct {
	Schema                                                string `json:"schema"`
	Accepted                                              bool   `json:"accepted"`
	CollectionValid                                       bool   `json:"collection_valid"`
	Failure                                               string `json:"failure"`
	UntimedRunError, BindingSHA256, UntimedEvidenceSHA256 string
	ResultPath, CanonicalPath, ChoicePath, FinishPath     string
	SummaryNanos                                          map[string]variantTimingRange
}

type warmPreparationRawObservation struct {
	TerminalReceivedAt, SupervisorReturnedAt time.Time
	TerminalToReapNanos                      uint64
	TerminalFrame                            []byte
	Result                                   Result
	Error                                    string
	Host                                     struct {
		Termination                           string
		ExitCode                              int
		GroupGone, Cancelled, WatchdogTimeout bool
	}
}

func warmPreparationCollectionBinding(series warmPreparationSeries) string {
	series.Inputs, series.CollectionPath = nil, ""
	data, _ := json.Marshal(series)
	return digest(data)
}

func warmPreparationProducerCheck(series warmPreparationSeries, currentReceipt string, currentInputs map[string]string) error {
	if series.Inputs[series.ModuleReceipt] == "" || series.Inputs[series.ModuleReceipt] != currentInputs[currentReceipt] {
		return fmt.Errorf("baseline module producer is not the authentic current owner-validated receipt")
	}
	for path, sha := range currentInputs {
		if path != currentReceipt && series.Inputs[path] != sha {
			return fmt.Errorf("baseline lacks authentic current producer source closure at %s", path)
		}
	}
	return nil
}

func TestWarmPreparationCurrentProducerClosureControls(t *testing.T) {
	for _, test := range []struct {
		name   string
		mutate func(map[string]string)
		want   bool
	}{
		{"same-owner-validated-producer", func(map[string]string) {}, true},
		{"missing-current-source", func(inputs map[string]string) { delete(inputs, "/current-source.go") }, false},
		{"changed-current-source", func(inputs map[string]string) { inputs["/current-source.go"] = strings.Repeat("f", 64) }, false},
		{"changed-current-producer", func(inputs map[string]string) { inputs["/module-receipt.json"] = strings.Repeat("f", 64) }, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			series := warmPreparationFixture()
			series.Inputs["/current-source.go"] = strings.Repeat("b", 64)
			current := maps.Clone(series.Inputs)
			test.mutate(series.Inputs)
			if err := warmPreparationProducerCheck(series, series.ModuleReceipt, current); (err == nil) != test.want {
				t.Fatalf("current producer closure=%v want %v", err, test.want)
			}
		})
	}
}

func warmPreparationCollectionCheck(series warmPreparationSeries, read func(string) ([]byte, error)) error {
	bound := func(path string) ([]byte, error) {
		if !filepath.IsAbs(path) || !warmPreparationHex(series.Inputs[path], 64) {
			return nil, fmt.Errorf("missing bound collection input %s", path)
		}
		data, err := read(path)
		if err != nil {
			return nil, err
		}
		if digest(data) != series.Inputs[path] {
			return nil, fmt.Errorf("changed collection input %s", path)
		}
		return data, nil
	}
	decode := func(path string, value any) error {
		data, err := bound(path)
		if err != nil {
			return err
		}
		return json.Unmarshal(data, value)
	}
	var collection warmPreparationCollection
	if err := decode(series.CollectionPath, &collection); err != nil {
		return err
	}
	if collection.Schema != "gomad3.wasm-warm-preparation-collection/v1" || collection.Accepted || !collection.CollectionValid || collection.Failure != "" || collection.UntimedRunError != "" || collection.BindingSHA256 != warmPreparationCollectionBinding(series) || !reflect.DeepEqual(collection.SummaryNanos, warmPreparationSummary(series)) || len(series.Runs) < 5 {
		return fmt.Errorf("failed or changed source/sample-bound collection")
	}
	var result Result
	if err := decode(collection.ResultPath, &result); err != nil {
		return err
	}
	finish, err := bound(collection.FinishPath)
	if err != nil {
		return err
	}
	if len(finish) != len(result.runtimeTerminal) {
		return fmt.Errorf("incomplete original Run finish")
	}
	copy(result.runtimeTerminal[:], finish)
	canonical, err := bound(collection.CanonicalPath)
	if err != nil {
		return err
	}
	choices, err := bound(collection.ChoicePath)
	if err != nil {
		return err
	}
	if !bytes.Equal(result.Transcript, canonical) || sha256.Sum256(canonical) != result.TranscriptSHA256 || !bytes.Equal(result.ChoiceTrace.Bytes, choices) || !result.Reaped || result.Cancelled || result.WatchdogTimeout || result.Termination != "exit" || result.ExitCode == nil || *result.ExitCode != 0 || result.Started == nil || result.Started.Engine != series.Engine || result.Started.ModuleSHA256 != series.ModuleSHA256 || warmPreparationEvidence(result) != collection.UntimedEvidenceSHA256 || collection.UntimedEvidenceSHA256 != series.Runs[0].EvidenceSHA256 {
		return fmt.Errorf("original Run retained bytes or behavior do not support this collection")
	}
	if series.Warmup == nil {
		return fmt.Errorf("missing actual warm-up observation")
	}
	rawRuns := append([]warmPreparationRun{*series.Warmup}, series.Runs...)
	for _, run := range rawRuns {
		var raw warmPreparationRawObservation
		if err := decode(run.RawObservationPath, &raw); err != nil {
			return err
		}
		var projected warmPreparationObservation
		if err := decode(run.ObservationPath, &projected); err != nil {
			return err
		}
		finish, err := bound(run.FinishPath)
		if err != nil {
			return err
		}
		if len(finish) != len(raw.Result.runtimeTerminal) {
			return fmt.Errorf("incomplete timed finish")
		}
		copy(raw.Result.runtimeTerminal[:], finish)
		copy(projected.Result.runtimeTerminal[:], finish)
		if raw.Error != "" || raw.Error != projected.Error || raw.TerminalReceivedAt != projected.TerminalReceivedAt || raw.SupervisorReturnedAt != projected.SupervisorReturnedAt || raw.TerminalToReapNanos != projected.TerminalToReapNanos || !bytes.Equal(raw.TerminalFrame, projected.TerminalFrame) || raw.Host.Termination != "exit" || !projected.HostExited || raw.Host.ExitCode != 0 || raw.Host.ExitCode != projected.HostExitCode || !raw.Host.GroupGone || raw.Host.GroupGone != projected.GroupGone || raw.Host.Cancelled || raw.Host.Cancelled != projected.Cancelled || raw.Host.WatchdogTimeout || raw.Host.WatchdogTimeout != projected.WatchdogTimeout || !reflect.DeepEqual(raw.Result, projected.Result) || warmPreparationEvidence(raw.Result) != run.EvidenceSHA256 || run.EvidenceSHA256 != collection.UntimedEvidenceSHA256 {
			return fmt.Errorf("original host observation differs from measured projection")
		}
	}
	return nil
}

func warmPreparationPublishCollection(t *testing.T, evidence, name string, series *warmPreparationSeries, retained Result, runError, failed string) string {
	t.Helper()
	collection := warmPreparationCollection{Schema: "gomad3.wasm-warm-preparation-collection/v1", CollectionValid: failed == "", Failure: failed, UntimedRunError: runError, BindingSHA256: warmPreparationCollectionBinding(*series), UntimedEvidenceSHA256: warmPreparationEvidence(retained), ResultPath: filepath.Join(evidence, "untimed-evidence-observed-result.json"), CanonicalPath: filepath.Join(evidence, "untimed-evidence-canonical.bin"), ChoicePath: filepath.Join(evidence, "untimed-evidence-choice.bin"), FinishPath: filepath.Join(evidence, "untimed-evidence-finish.bin"), SummaryNanos: warmPreparationSummary(*series)}
	for _, path := range []string{collection.ResultPath, collection.CanonicalPath, collection.ChoicePath, collection.FinishPath} {
		series.Inputs[path] = fileHash(t, path)
	}
	series.CollectionPath = filepath.Join(evidence, name)
	temporalJSON(t, series.CollectionPath, collection)
	series.Inputs[series.CollectionPath] = fileHash(t, series.CollectionPath)
	if failed == "" {
		if err := warmPreparationCollectionCheck(*series, os.ReadFile); err != nil {
			failed = err.Error()
			collection.CollectionValid, collection.Failure = false, failed
			temporalJSON(t, series.CollectionPath, collection)
			series.Inputs[series.CollectionPath] = fileHash(t, series.CollectionPath)
		}
	}
	temporalJSON(t, filepath.Join(evidence, "series.json"), *series)
	return failed
}

func TestWarmPreparationCollectionControls(t *testing.T) {
	for _, test := range []struct {
		name   string
		mutate func(*warmPreparationSeries, *warmPreparationCollection, map[string][]byte)
		want   bool
	}{
		{"successful-original-run", func(*warmPreparationSeries, *warmPreparationCollection, map[string][]byte) {}, true},
		{"collection-failed-five-good-samples", func(_ *warmPreparationSeries, c *warmPreparationCollection, _ map[string][]byte) {
			c.CollectionValid = false
		}, false},
		{"reported-failure", func(_ *warmPreparationSeries, c *warmPreparationCollection, _ map[string][]byte) {
			c.Failure = "warmup changed"
		}, false},
		{"original-run-error", func(_ *warmPreparationSeries, c *warmPreparationCollection, _ map[string][]byte) {
			c.UntimedRunError = "original Run failed"
		}, false},
		{"changed-source-binding", func(s *warmPreparationSeries, _ *warmPreparationCollection, _ map[string][]byte) {
			s.SourceCommit = strings.Repeat("b", 40)
		}, false},
		{"changed-sample-binding", func(s *warmPreparationSeries, _ *warmPreparationCollection, _ map[string][]byte) {
			s.Runs[1].WallNanos++
		}, false},
		{"dropped-warmup", func(s *warmPreparationSeries, _ *warmPreparationCollection, _ map[string][]byte) { s.Warmup = nil }, false},
		{"dropped-marker", func(s *warmPreparationSeries, _ *warmPreparationCollection, files map[string][]byte) {
			delete(files, s.CollectionPath)
		}, false},
		{"unbound-marker", func(s *warmPreparationSeries, _ *warmPreparationCollection, _ map[string][]byte) {
			delete(s.Inputs, s.CollectionPath)
		}, false},
		{"dropped-canonical", func(_ *warmPreparationSeries, c *warmPreparationCollection, files map[string][]byte) {
			delete(files, c.CanonicalPath)
		}, false},
		{"tampered-canonical", func(_ *warmPreparationSeries, c *warmPreparationCollection, files map[string][]byte) {
			files[c.CanonicalPath] = []byte("tampered")
		}, false},
		{"rehashed-wrong-canonical", func(s *warmPreparationSeries, c *warmPreparationCollection, files map[string][]byte) {
			files[c.CanonicalPath] = []byte("tampered")
			s.Inputs[c.CanonicalPath] = digest(files[c.CanonicalPath])
		}, false},
		{"rehashed-wrong-finish", func(s *warmPreparationSeries, c *warmPreparationCollection, files map[string][]byte) {
			files[c.FinishPath][0]++
			s.Inputs[c.FinishPath] = digest(files[c.FinishPath])
		}, false},
		{"dropped-original-observation", func(s *warmPreparationSeries, _ *warmPreparationCollection, files map[string][]byte) {
			delete(files, s.Runs[0].RawObservationPath)
		}, false},
		{"tampered-original-observation", func(s *warmPreparationSeries, _ *warmPreparationCollection, files map[string][]byte) {
			files[s.Runs[0].RawObservationPath] = []byte("{}")
		}, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			series := warmPreparationFixture()
			series.Role, series.CollectionPath = "baseline", "/collection.json"
			code := uint32(0)
			result := Result{Termination: "exit", ExitCode: &code, Reaped: true, Started: &Started{Engine: series.Engine, ModuleSHA256: series.ModuleSHA256}, Transcript: []byte("canonical"), TranscriptSHA256: sha256.Sum256([]byte("canonical")), runtimeTerminal: [16]byte{1}}
			result.ChoiceTrace.Bytes = []byte("choices")
			files := map[string][]byte{}
			put := func(path string, data []byte) { files[path] = data; series.Inputs[path] = digest(data) }
			encode := func(value any) []byte {
				data, err := json.Marshal(value)
				if err != nil {
					t.Fatal(err)
				}
				return data
			}
			for i := range series.Runs {
				run := &series.Runs[i]
				run.EvidenceSHA256 = warmPreparationEvidence(result)
				run.RawObservationPath, run.ObservationPath = fmt.Sprintf("/run-%d-raw.json", i), fmt.Sprintf("/run-%d-projected.json", i)
				run.FinishPath = fmt.Sprintf("/run-%d-finish", i)
				observation := terminalExitObservation{TerminalReceivedAt: time.Unix(1, 0), SupervisorReturnedAt: time.Unix(2, 0), TerminalToReapNanos: uint64(time.Second), Result: result}
				observation.Host.Termination, observation.Host.GroupGone = "exit", true
				put(run.RawObservationPath, encode(observation))
				put(run.ObservationPath, encode(warmPreparationObserve(observation)))
				put(run.FinishPath, bytes.Clone(result.runtimeTerminal[:]))
			}
			warmup := series.Runs[0]
			series.Warmup = &warmup
			collection := warmPreparationCollection{Schema: "gomad3.wasm-warm-preparation-collection/v1", CollectionValid: true, BindingSHA256: warmPreparationCollectionBinding(series), UntimedEvidenceSHA256: warmPreparationEvidence(result), ResultPath: "/untimed-result.json", CanonicalPath: "/untimed-canonical", ChoicePath: "/untimed-choice", FinishPath: "/untimed-finish", SummaryNanos: warmPreparationSummary(series)}
			put(collection.ResultPath, encode(result))
			put(collection.CanonicalPath, result.Transcript)
			put(collection.ChoicePath, result.ChoiceTrace.Bytes)
			put(collection.FinishPath, bytes.Clone(result.runtimeTerminal[:]))
			test.mutate(&series, &collection, files)
			marker := encode(collection)
			if test.name != "dropped-marker" {
				files[series.CollectionPath] = marker
			}
			if test.name != "unbound-marker" {
				series.Inputs[series.CollectionPath] = digest(marker)
			}
			err := warmPreparationCollectionCheck(series, func(path string) ([]byte, error) {
				data, ok := files[path]
				if !ok {
					return nil, os.ErrNotExist
				}
				return data, nil
			})
			if (err == nil) != test.want {
				t.Fatalf("collection admission=%v want %v", err, test.want)
			}
		})
	}
}

func warmPreparationFixture() warmPreparationSeries {
	series := warmPreparationSeries{Schema: warmPreparationSchema, Role: "candidate", SourceCommit: strings.Repeat("a", 40), SourceRoot: "/source", DriverSHA256: strings.Repeat("d", 64), HostBootID: "same-boot", HostCPUInfo: "Features: fp asimd\nCPU architecture: 8\n", ConfigSHA256: strings.Repeat("c", 64), Engine: PinnedEngine(), ModuleSHA256: strings.Repeat("a", 64), HelperSHA256: strings.Repeat("b", 64), RuntimeIdentity: "runtime", ModelSHA256: "model", ArtifactBytes: 100, EnvelopeBytes: 200, Inputs: map[string]string{"/source/driver.go": strings.Repeat("d", 64)}}
	series.ModuleReceipt = "/module-receipt.json"
	series.ComparisonSHA256 = strings.Repeat("e", 64)
	series.Inputs[series.ModuleReceipt] = strings.Repeat("a", 64)
	for range 5 {
		series.Runs = append(series.Runs, warmPreparationRun{Telemetry: Telemetry{CacheHit: true, PrepareNanos: uint64(time.Second)}, Phases: map[string]diagnosticPhase{"cache_read": {Count: 1, WallNanos: 2}, "envelope_verify": {Count: 1, WallNanos: 3}, "deserialize": {Count: 1, WallNanos: 4}, "cache_identity": {Count: 1, WallNanos: 1}}, WallNanos: 10, TerminalToReapNanos: uint64(time.Second), EvidenceSHA256: strings.Repeat("e", 64), Exited: true, Reaped: true})
	}
	return series
}

func TestWarmPreparationAdmissionControls(t *testing.T) {
	for _, test := range []struct {
		name   string
		mutate func(*warmPreparationSeries)
		want   bool
	}{
		{"five-warm-at-bound", func(*warmPreparationSeries) {}, true},
		{"four-runs", func(s *warmPreparationSeries) { s.Runs = s.Runs[:4] }, false},
		{"cold-run", func(s *warmPreparationSeries) { s.Runs[2].Telemetry.CacheHit = false }, false},
		{"compiled-run", func(s *warmPreparationSeries) { s.Runs[2].Telemetry.CompileNanos = 1 }, false},
		{"prepare-over-bound", func(s *warmPreparationSeries) {
			for i := range s.Runs {
				s.Runs[i].Telemetry.PrepareNanos++
			}
		}, false},
		{"one-slow-prepare-median-still-bound", func(s *warmPreparationSeries) { s.Runs[0].Telemetry.PrepareNanos++ }, true},
		{"slow-terminal", func(s *warmPreparationSeries) { s.Runs[2].TerminalToReapNanos++ }, false},
		{"missing-terminal", func(s *warmPreparationSeries) { s.Runs[2].TerminalToReapNanos = 0 }, false},
		{"unreaped", func(s *warmPreparationSeries) { s.Runs[2].Reaped = false }, false},
		{"failed-exit", func(s *warmPreparationSeries) { s.Runs[2].Exited = false }, false},
		{"missing-phase", func(s *warmPreparationSeries) { delete(s.Runs[2].Phases, "deserialize") }, false},
		{"missing-phase-sample", func(s *warmPreparationSeries) { s.Runs[2].Phases["deserialize"] = diagnosticPhase{} }, false},
		{"unstable-evidence", func(s *warmPreparationSeries) { s.Runs[2].EvidenceSHA256 = strings.Repeat("f", 64) }, false},
		{"emulated-host", func(s *warmPreparationSeries) { s.HostCPUInfo = "vendor_id: GenuineIntel\n" }, false},
		{"missing-host-binding", func(s *warmPreparationSeries) { s.HostBootID = "" }, false},
		{"missing-engine-binding", func(s *warmPreparationSeries) { s.Engine.Configuration = "" }, false},
		{"missing-source-binding", func(s *warmPreparationSeries) { s.SourceCommit = "" }, false},
		{"missing-receipts", func(s *warmPreparationSeries) { s.Inputs = nil }, false},
		{"wrong-artifact-size", func(s *warmPreparationSeries) { s.ArtifactBytes = s.EnvelopeBytes }, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			series := warmPreparationFixture()
			test.mutate(&series)
			if err := warmPreparationValidate(series, true); (err == nil) != test.want {
				t.Fatalf("admission=%v, want %v", err, test.want)
			}
		})
	}
}

func TestWarmPreparationComparisonControls(t *testing.T) {
	for _, test := range []struct {
		name   string
		mutate func(*warmPreparationSeries)
		want   bool
	}{
		{"separate-engine-helper-model-bindings", func(s *warmPreparationSeries) {
			s.Engine.Configuration += "-historical"
			s.HelperSHA256 = strings.Repeat("f", 64)
			s.ModelSHA256 = "historical-model"
			s.DriverSHA256 = strings.Repeat("f", 64)
			s.SourceCommit = strings.Repeat("b", 40)
		}, true},
		{"different-host-boot", func(s *warmPreparationSeries) { s.HostBootID = "other-boot" }, false},
		{"different-module", func(s *warmPreparationSeries) { s.ModuleSHA256 = strings.Repeat("f", 64) }, false},
		{"different-seed", func(s *warmPreparationSeries) { s.Seed++ }, false},
		{"different-config", func(s *warmPreparationSeries) { s.ConfigSHA256 = strings.Repeat("f", 64) }, false},
		{"different-runtime", func(s *warmPreparationSeries) { s.RuntimeIdentity = "other-runtime" }, false},
		{"different-current-producer", func(s *warmPreparationSeries) { s.Inputs[s.ModuleReceipt] = strings.Repeat("f", 64) }, false},
		{"different-evidence", func(s *warmPreparationSeries) {
			s.ComparisonSHA256 = strings.Repeat("f", 64)
			for i := range s.Runs {
				s.Runs[i].EvidenceSHA256 = strings.Repeat("f", 64)
			}
		}, false},
		{"different-raw-encoding-same-validated-behavior", func(s *warmPreparationSeries) {
			for i := range s.Runs {
				s.Runs[i].EvidenceSHA256 = strings.Repeat("f", 64)
			}
		}, true},
		{"missing-validated-comparison", func(s *warmPreparationSeries) { s.ComparisonSHA256 = "" }, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			before, after := warmPreparationFixture(), warmPreparationFixture()
			before.Role = "baseline"
			test.mutate(&before)
			if err := warmPreparationCompare(before, after); (err == nil) != test.want {
				t.Fatalf("comparison=%v, want %v", err, test.want)
			}
		})
	}
}

func warmPreparationEvidence(result Result) string {
	data, _ := json.Marshal(struct {
		Stdout, Stderr, Choices, Network []byte
		WASI                             [32]byte
		Terminal                         [16]byte
		Termination                      string
		ExitCode                         *uint32
	}{result.Stdout, result.Stderr, result.ChoiceTrace.Bytes, result.NetworkTranscript, result.TranscriptSHA256, result.runtimeTerminal, result.Termination, result.ExitCode})
	return digest(data)
}

func TestWarmPreparationEvidenceControls(t *testing.T) {
	code := uint32(0)
	baseline := Result{Stdout: []byte("stdout"), Stderr: []byte("stderr"), NetworkTranscript: []byte("network"), TranscriptSHA256: [32]byte{1}, runtimeTerminal: [16]byte{1}, Termination: "exit", ExitCode: &code}
	baseline.ChoiceTrace.Bytes = []byte("choices")
	for _, test := range []struct {
		field  string
		mutate func(*Result)
		same   bool
	}{
		{"stdout", func(r *Result) { r.Stdout = []byte("other") }, false},
		{"stderr", func(r *Result) { r.Stderr = []byte("other") }, false},
		{"network", func(r *Result) { r.NetworkTranscript = []byte("other") }, false},
		{"wasi", func(r *Result) { r.TranscriptSHA256[0]++ }, false},
		{"terminal", func(r *Result) { r.runtimeTerminal[0]++ }, false},
		{"choices", func(r *Result) { r.ChoiceTrace.Bytes = []byte("other") }, false},
		{"termination", func(r *Result) { r.Termination = "other" }, false},
		{"exit-code", func(r *Result) { other := uint32(1); r.ExitCode = &other }, false},
		{"environment-identity", func(r *Result) { r.EnvironmentIdentity = "other" }, true},
		{"started-identity", func(r *Result) { r.Started = &Started{} }, true},
	} {
		t.Run(test.field, func(t *testing.T) {
			candidate := baseline
			test.mutate(&candidate)
			if (warmPreparationEvidence(candidate) == warmPreparationEvidence(baseline)) != test.same {
				t.Fatal("behavior/identity separation lost", test.field)
			}
		})
	}
}

func TestWarmPreparationDiagnosticCargoFeatureControls(t *testing.T) {
	data := []byte(`{"reason":"compiler-artifact","target":{"name":"gomad3-wasmhost"},"profile":{"opt_level":"3","debug_assertions":false},"executable":"/release/helper","features":["diagnostic-profile"]} {"reason":"build-finished","success":true}`)
	if _, err := terminalExitCargoHelperFeatures(data, []string{"diagnostic-profile"}); err != nil {
		t.Fatal(err)
	}
	if _, err := terminalExitCargoHelper(data); err == nil {
		t.Fatal("release-only gate weakened")
	}
	for _, features := range []string{`[]`, `["guest-profile"]`, `["diagnostic-profile","fuel-off"]`} {
		if _, err := terminalExitCargoHelperFeatures(bytes.ReplaceAll(data, []byte(`["diagnostic-profile"]`), []byte(features)), []string{"diagnostic-profile"}); err == nil {
			t.Fatal("wrong feature admitted", features)
		}
	}
}

func TestBuildWarmPreparationDiagnosticHelper(t *testing.T) {
	root := os.Getenv("GOMAD_WASM_WARM_HELPER_BUILD_ROOT")
	if root == "" {
		t.Skip("set GOMAD_WASM_WARM_HELPER_BUILD_ROOT to build a retained diagnostic-profile helper")
	}
	terminalExitBuildHelper(t, root, []string{"diagnostic-profile"})
}

func TestWarmPreparationPrimeCache(t *testing.T) {
	root := os.Getenv("GOMAD_WASM_WARM_PRIME_ROOT")
	if root == "" {
		t.Skip("set GOMAD_WASM_WARM_PRIME_ROOT for explicit serial cache priming, never an acceptance sample")
	}
	for _, path := range []string{root, os.Getenv("GOMAD_WASM_TEST_HELPER"), os.Getenv("GOMAD_WASM_WARM_MODULE"), os.Getenv("GOMAD_WASM_WARM_CACHE"), os.Getenv("GOMAD_WASM_WARM_HELPER_RECEIPT"), os.Getenv("GOMAD_WASM_WARM_MODULE_RECEIPT")} {
		if !filepath.IsAbs(path) {
			t.Fatal("cache priming requires the same explicit bound inputs as measurement")
		}
	}
	if _, host := diagnosticPerformanceHostEvidence("/proc/cpuinfo"); !host {
		t.Fatal("real Linux/arm64 host required")
	}
	evidence, err := diagnosticRunRoot(root, "prime-only-")
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("GOMAD_WASM_PROTOTYPE_ROOT", evidence)
	repo, err := filepath.Abs("../../../../../..")
	if err != nil {
		t.Fatal(err)
	}
	config := prototypeTemporalConfig(t, repo, "/workspace/tests", "-test.run=^TestWorkflowTypeEncodingSuite$/^TestPlainASCII$/^Succeeds$", "-persistenceType=sql", "-persistenceDriver=sqlite")
	request := prototypeRequest(t, os.Getenv("GOMAD_WASM_WARM_MODULE"), config, 0, 30*time.Minute)
	request.CompileCache = os.Getenv("GOMAD_WASM_WARM_CACHE")
	crate, err := filepath.Abs("../wasmhost")
	if err != nil {
		t.Fatal(err)
	}
	inputs := terminalExitHelperInputsFeatures(t, os.Getenv("GOMAD_WASM_WARM_HELPER_RECEIPT"), request, crate, []string{"diagnostic-profile"})
	maps.Copy(inputs, terminalExitModuleInputs(t, os.Getenv("GOMAD_WASM_WARM_MODULE_RECEIPT"), request))
	result, runErr := matchedPrepareAdmitted("linux", "/proc/meminfo", "1", func() (Result, error) { return Run(t.Context(), request) })
	persistDiagnosticRawResult(t, evidence, "prime-only", result)
	temporalWrite(t, filepath.Join(evidence, "prime-only-helper-stderr.txt"), result.helperStderr)
	temporalJSON(t, filepath.Join(evidence, "prime-only.json"), map[string]any{"accepted": false, "measurement_sample": false, "purpose": "explicit compiled-cache priming outside all warm measurement windows", "inputs": inputs, "telemetry": result.Telemetry})
	if runErr != nil {
		t.Fatal(runErr)
	}
	requireExit(t, "prime-only", result, 0)
	if _, err := decodeDiagnosticReport(result.helperStderr); err != nil {
		t.Fatal(err)
	}
	temporalVerifyInputs(t, inputs)
	terminalExitHelperInputsFeatures(t, os.Getenv("GOMAD_WASM_WARM_HELPER_RECEIPT"), request, crate, []string{"diagnostic-profile"})
	terminalExitModuleInputs(t, os.Getenv("GOMAD_WASM_WARM_MODULE_RECEIPT"), request)
}

func warmPreparationSummary(series warmPreparationSeries) map[string]variantTimingRange {
	values := map[string][]uint64{}
	for _, run := range series.Runs {
		values["prepare"] = append(values["prepare"], run.Telemetry.PrepareNanos)
		values["whole_run"] = append(values["whole_run"], run.WallNanos)
		values["terminal_to_reap"] = append(values["terminal_to_reap"], run.TerminalToReapNanos)
		for _, phase := range strings.Fields("cache_read envelope_verify deserialize cache_identity") {
			values[phase] = append(values[phase], run.Phases[phase].WallNanos)
		}
		values["cache_read_plus_deserialize"] = append(values["cache_read_plus_deserialize"], run.Phases["cache_read"].WallNanos+run.Phases["deserialize"].WallNanos)
	}
	summary := map[string]variantTimingRange{}
	for name, samples := range values {
		summary[name] = variantTiming(samples)
	}
	return summary
}

func TestWarmPreparationSummaryControls(t *testing.T) {
	series := warmPreparationFixture()
	for i, value := range []uint64{7, 2, 5, 9, 4} {
		series.Runs[i].Telemetry.PrepareNanos = value
	}
	if got := warmPreparationSummary(series)["prepare"]; got != (variantTimingRange{Median: 5, Min: 2, Max: 9}) {
		t.Fatal("median/range lost", got)
	}
	baseline := warmPreparationFixture()
	baseline.Role = "baseline"
	for i := range baseline.Runs {
		baseline.Runs[i].Telemetry.PrepareNanos++
		baseline.Runs[i].TerminalToReapNanos++
	}
	if err := warmPreparationValidate(baseline, false); err != nil {
		t.Fatal("historical timings incorrectly subject to candidate bounds", err)
	}
}

func warmPreparationRead(t *testing.T, path, evidence, currentProducer string, currentRequest Request) warmPreparationSeries {
	t.Helper()
	var series warmPreparationSeries
	decoder := json.NewDecoder(bytes.NewReader(mustTerminalExitRead(t, path)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&series); err != nil {
		t.Fatal(err)
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		t.Fatal("trailing warm-preparation receipt", err)
	}
	temporalVerifyInputs(t, series.Inputs)
	if err := warmPreparationCollectionCheck(series, os.ReadFile); err != nil {
		t.Fatal("historical collection did not close original Run equivalence", err)
	}
	currentInputs := terminalExitModuleInputs(t, currentProducer, currentRequest)
	if err := warmPreparationProducerCheck(series, currentProducer, currentInputs); err != nil {
		t.Fatal(err)
	}
	commit := strings.TrimSpace(string(temporalCommand(t, evidence, "baseline-commit-check", series.SourceRoot, nil, []string{"git", "rev-parse", "HEAD"}).Stdout.RawBytes))
	if commit != series.SourceCommit {
		t.Fatal("historical source commit changed")
	}
	temporalCommand(t, evidence, "baseline-ancestor-check", series.SourceRoot, nil, []string{"git", "merge-base", "--is-ancestor", "15119dbbeb03f5a57fa79d57be0ab2d9ba4038c3", series.SourceCommit})
	changes := temporalCommand(t, evidence, "baseline-production-check", series.SourceRoot, nil, []string{"git", "diff", "--name-only", "15119dbbeb03f5a57fa79d57be0ab2d9ba4038c3", series.SourceCommit, "--", "."})
	for _, path := range strings.Fields(string(changes.Stdout.RawBytes)) {
		if path != "tools/gomad3/backends/wasm/wasi/warm_preparation_test.go" && path != "tools/gomad3/backends/wasm/wasi/warm_preparation_inputs_test.go" && path != "tools/gomad3/backends/wasm/wasi/warm_preparation_terminal_test.go" {
			t.Fatal("historical production or original driver changed", path)
		}
	}
	if series.Inputs[series.DriverPath] != series.DriverSHA256 || !filepath.IsAbs(series.SourceRoot) || !strings.HasPrefix(series.DriverPath, series.SourceRoot+string(filepath.Separator)) || series.Inputs[series.HelperReceipt] == "" || series.Inputs[series.ModuleReceipt] == "" || series.Inputs[series.ArtifactPath] == "" {
		t.Fatal("historical receipt lacks actual driver/build/module/cache bindings")
	}
	var admission terminalExitHelperAdmission
	if err := json.Unmarshal(mustTerminalExitRead(t, series.HelperReceipt), &admission); err != nil {
		t.Fatal(err)
	}
	var build terminalExitHelperBuild
	if err := json.Unmarshal(mustTerminalExitRead(t, admission.BuildReceipt), &build); err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(build.Crate, series.SourceRoot+string(filepath.Separator)) || admission.HelperSHA256 != series.HelperSHA256 {
		t.Fatal("historical helper does not belong to its retained source/build")
	}
	terminalExitHelperInputsFeatures(t, series.HelperReceipt, Request{HelperPath: admission.Helper, HelperSHA256: admission.HelperSHA256}, build.Crate, []string{"diagnostic-profile"})
	for _, run := range series.Runs {
		for _, rawPath := range []string{run.ObservationPath, run.FinishPath, run.DiagnosticPath} {
			if series.Inputs[rawPath] == "" {
				t.Fatal("historical run lacks bound raw observation/finish/diagnostic evidence")
			}
		}
		var observation warmPreparationObservation
		if err := json.Unmarshal(mustTerminalExitRead(t, run.ObservationPath), &observation); err != nil {
			t.Fatal(err)
		}
		finish := mustTerminalExitRead(t, run.FinishPath)
		if len(finish) != len(observation.Result.runtimeTerminal) {
			t.Fatal("missing historical private terminal sidecar")
		}
		copy(observation.Result.runtimeTerminal[:], finish)
		profile, err := decodeDiagnosticReport(mustTerminalExitRead(t, run.DiagnosticPath))
		if err != nil || !reflect.DeepEqual(profile.Phases, run.Phases) || observation.Result.Telemetry == nil || *observation.Result.Telemetry != run.Telemetry || warmPreparationEvidence(observation.Result) != run.EvidenceSHA256 || observation.TerminalToReapNanos != run.TerminalToReapNanos || observation.Error != "" || !observation.HostExited || observation.HostExitCode != 0 || !observation.GroupGone || observation.Cancelled || observation.WatchdogTimeout || observation.TerminalReceivedAt.IsZero() || !observation.SupervisorReturnedAt.After(observation.TerminalReceivedAt) || uint64(observation.SupervisorReturnedAt.Sub(observation.TerminalReceivedAt).Nanoseconds()) != run.TerminalToReapNanos || !observation.Result.Reaped || observation.Result.Termination != "exit" || observation.Result.ExitCode == nil || *observation.Result.ExitCode != 0 || observation.Result.Started == nil || observation.Result.Started.Engine != series.Engine || observation.Result.Started.ModuleSHA256 != series.ModuleSHA256 {
			t.Fatal("historical raw observation does not support the declared run", err)
		}
	}
	configBytes, err := json.Marshal(currentRequest.Config)
	if err != nil || series.ConfigSHA256 != digest(configBytes) {
		t.Fatal("historical comparison requires the bound modeled configuration", err)
	}
	var collection warmPreparationCollection
	if err := json.Unmarshal(mustTerminalExitRead(t, series.CollectionPath), &collection); err != nil {
		t.Fatal(err)
	}
	var retained Result
	if err := json.Unmarshal(mustTerminalExitRead(t, collection.ResultPath), &retained); err != nil {
		t.Fatal(err)
	}
	copy(retained.runtimeTerminal[:], mustTerminalExitRead(t, collection.FinishPath))
	series.ComparisonSHA256, err = warmPreparationBehavior(retained, currentRequest.Config, true)
	if err != nil {
		t.Fatal("source-bound historical transcript comparison", err)
	}
	return series
}

func warmPreparationArtifact(t *testing.T, request Request) (string, uint64, uint64) {
	t.Helper()
	entries, err := filepath.Glob(filepath.Join(request.CompileCache, "*.gomad-cache-v3"))
	if err != nil {
		t.Fatal(err)
	}
	selected := ""
	var artifactBytes, envelopeBytes uint64
	for _, entry := range entries {
		file, info, err := hostfs.OpenPath(entry)
		if err != nil {
			t.Fatal(err)
		}
		var header [24]byte
		_, err = io.ReadFull(file, header[:])
		manifestBytes := binary.LittleEndian.Uint32(header[8:12])
		artifact := binary.LittleEndian.Uint64(header[12:20])
		if err != nil || string(header[:8]) != "GOMADC03" || manifestBytes > 16<<10 || artifact == 0 || artifact > 1<<30 || uint64(info.Size()) != 24+uint64(manifestBytes)+artifact {
			_ = file.Close()
			t.Fatal("invalid current cache envelope", entry, err)
		}
		manifest := make([]byte, manifestBytes)
		_, err = io.ReadFull(file, manifest)
		closeErr := file.Close()
		if err != nil || closeErr != nil {
			t.Fatal("read cache manifest", err, closeErr)
		}
		var identity struct {
			Version       uint64 `json:"version"`
			ModuleSHA256  string `json:"module_sha256"`
			HelperSHA256  string `json:"helper_sha256"`
			Configuration string `json:"configuration"`
			HostOS        string `json:"host_os"`
			HostArch      string `json:"host_arch"`
		}
		if err := json.Unmarshal(manifest, &identity); err != nil {
			t.Fatal(err)
		}
		if identity.ModuleSHA256 != request.ModuleSHA256 || identity.HelperSHA256 != request.HelperSHA256 {
			continue
		}
		if selected != "" || identity.Version != 3 || identity.Configuration != request.Engine.Configuration || identity.HostOS != request.Engine.HostOS || identity.HostArch != request.Engine.HostArch || filepath.Base(entry) != digest(manifest)+".gomad-cache-v3" {
			t.Fatal("ambiguous or incompatible measured cache identity")
		}
		selected, artifactBytes, envelopeBytes = entry, artifact, uint64(info.Size())
	}
	if selected == "" {
		t.Fatal("prewarmed matching cache artifact required; this driver never cold-compiles intentionally")
	}
	return selected, artifactBytes, envelopeBytes
}

func TestWarmPreparationArtifactSizeControls(t *testing.T) {
	request := Request{CompileCache: t.TempDir(), ModuleSHA256: strings.Repeat("a", 64), HelperSHA256: strings.Repeat("b", 64), Engine: PinnedEngine()}
	var expected string
	var expectedEnvelope uint64
	for _, module := range []string{request.ModuleSHA256, strings.Repeat("c", 64)} {
		manifest, err := json.Marshal(map[string]any{"version": 3, "module_sha256": module, "helper_sha256": request.HelperSHA256, "configuration": request.Engine.Configuration, "host_os": request.Engine.HostOS, "host_arch": request.Engine.HostArch})
		if err != nil {
			t.Fatal(err)
		}
		header := make([]byte, 24)
		copy(header, "GOMADC03")
		binary.LittleEndian.PutUint32(header[8:12], uint32(len(manifest)))
		binary.LittleEndian.PutUint64(header[12:20], 100)
		entry := filepath.Join(request.CompileCache, digest(manifest)+".gomad-cache-v3")
		data := append(append(header, manifest...), make([]byte, 100)...)
		if err := os.WriteFile(entry, data, 0600); err != nil {
			t.Fatal(err)
		}
		if module == request.ModuleSHA256 {
			expected, expectedEnvelope = entry, uint64(len(data))
		}
	}
	entry, artifact, envelope := warmPreparationArtifact(t, request)
	if entry != expected || artifact != 100 || envelope != expectedEnvelope {
		t.Fatal("artifact/envelope bytes or matching module identity lost", entry, artifact, envelope)
	}
}

// warmPreparationFailure keeps every failure reason so retained receipts never
// replace an earlier cause with a later one.
func warmPreparationFailure(failed, reason string) string {
	switch {
	case failed == "":
		return reason
	case strings.Contains(failed, reason):
		return failed
	default:
		return failed + "; " + reason
	}
}

func TestWarmPreparationFailureKeepsEveryReason(t *testing.T) {
	failed := warmPreparationFailure("", "host bindings changed during collection")
	failed = warmPreparationFailure(failed, "baseline-material-gap")
	failed = warmPreparationFailure(failed, "baseline-material-gap")
	if failed != "host bindings changed during collection; baseline-material-gap" {
		t.Fatal(failed)
	}
}

func TestTemporalWarmPreparationAcceptance(t *testing.T) {
	root := os.Getenv("GOMAD_WASM_WARM_PREPARATION_ROOT")
	if root == "" {
		t.Skip("set GOMAD_WASM_WARM_PREPARATION_ROOT for serial real-host five-run acceptance collection")
	}
	paths := map[string]string{"root": root, "helper": os.Getenv("GOMAD_WASM_TEST_HELPER"), "module": os.Getenv("GOMAD_WASM_WARM_MODULE"), "cache": os.Getenv("GOMAD_WASM_WARM_CACHE"), "helper_receipt": os.Getenv("GOMAD_WASM_WARM_HELPER_RECEIPT"), "module_receipt": os.Getenv("GOMAD_WASM_WARM_MODULE_RECEIPT")}
	for name, path := range paths {
		if !filepath.IsAbs(path) {
			t.Fatal("explicit absolute warm-preparation input required", name)
		}
	}
	cpuinfo, realHost := diagnosticPerformanceHostEvidence("/proc/cpuinfo")
	if !realHost {
		t.Fatal("real Linux/arm64 performance host required")
	}
	boot, err := hostfs.ReadBounded("/proc/sys/kernel/random/boot_id", 128)
	if err != nil || strings.TrimSpace(string(boot)) == "" {
		t.Fatal("host boot identity required", err)
	}
	evidence, err := diagnosticRunRoot(root, "warm-preparation-")
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("GOMAD_WASM_PROTOTYPE_ROOT", evidence)
	repo, err := filepath.Abs("../../../../../..")
	if err != nil {
		t.Fatal(err)
	}
	config := prototypeTemporalConfig(t, repo, "/workspace/tests", "-test.run=^TestWorkflowTypeEncodingSuite$/^TestPlainASCII$/^Succeeds$", "-persistenceType=sql", "-persistenceDriver=sqlite")
	request := prototypeRequest(t, paths["module"], config, 0, 30*time.Minute)
	request.CompileCache = paths["cache"]
	// Digest-only matches the unchanged timing boundary; raw evidence is retained
	// once in an untimed final run and checked against every measured run.
	request.DigestTranscript = true
	crate, err := filepath.Abs("../wasmhost")
	if err != nil {
		t.Fatal(err)
	}
	inputs := terminalExitHelperInputsFeatures(t, paths["helper_receipt"], request, crate, []string{"diagnostic-profile"})
	maps.Copy(inputs, terminalExitModuleInputs(t, paths["module_receipt"], request))
	commit := strings.TrimSpace(string(temporalCommand(t, evidence, "source-commit", repo, nil, []string{"git", "rev-parse", "HEAD"}).Stdout.RawBytes))
	status := temporalCommand(t, evidence, "source-status", repo, nil, []string{"git", "status", "--porcelain", "--", "tools/gomad"})
	if len(status.Stdout.RawBytes) != 0 {
		t.Fatal("freeze and commit the measured tool source first")
	}
	tracked := temporalCommand(t, evidence, "source-inventory", repo, nil, []string{"git", "ls-files", "-z", "tools/gomad"})
	for _, name := range bytes.Split(tracked.Stdout.RawBytes, []byte{0}) {
		if len(name) != 0 {
			path := filepath.Join(repo, string(name))
			inputs[path] = fileHash(t, path)
		}
	}
	driver := filepath.Join(repo, warmPreparationDriverSource)
	artifact, artifactBytes, envelopeBytes := warmPreparationArtifact(t, request)
	inputs[artifact] = fileHash(t, artifact) // outside all timing windows; warms its pages deliberately.
	configBytes, err := json.Marshal(request.Config)
	if err != nil {
		t.Fatal(err)
	}
	series := warmPreparationSeries{Schema: warmPreparationSchema, Role: "candidate", SourceCommit: commit, SourceRoot: repo, DriverSHA256: fileHash(t, driver), HostBootID: strings.TrimSpace(string(boot)), HostCPUInfo: cpuinfo, ConfigSHA256: digest(configBytes), Engine: request.Engine, ModuleSHA256: request.ModuleSHA256, HelperSHA256: request.HelperSHA256, RuntimeIdentity: request.Runtime.Choice.ExecutionIdentity.ToolchainBuildKey, ModelSHA256: ImplementationSHA256(), Seed: request.Runtime.Seed, ArtifactBytes: artifactBytes, EnvelopeBytes: envelopeBytes, Inputs: inputs, HelperReceipt: paths["helper_receipt"], ModuleReceipt: paths["module_receipt"], DriverPath: driver, ArtifactPath: artifact}
	failed := ""
	var reference Result
	for i := range 6 {
		var observation terminalExitObservation
		started := time.Now()
		_, admissionErr := matchedPrepareAdmitted("linux", "/proc/meminfo", "1", func() (Result, error) {
			observation = measureTerminalExitContext(t, t.Context(), request, false)
			return observation.Result, nil
		})
		wall := uint64(time.Since(started).Nanoseconds())
		if admissionErr != nil {
			t.Fatal(admissionErr)
		}
		name := fmt.Sprintf("run-%d", i)
		temporalJSON(t, filepath.Join(evidence, name+"-observation.json"), observation)
		temporalWrite(t, filepath.Join(evidence, name+"-helper-stderr.txt"), observation.Host.Stderr.RawBytes)
		if observation.Error != "" || observation.Result.Telemetry == nil {
			t.Fatal("helper observation/telemetry failed", observation.Error)
		}
		if !observation.Result.Telemetry.CacheHit || observation.Result.Telemetry.CompileNanos != 0 {
			t.Fatal("every run, including the unmeasured warm-up, must hit the existing cache without compilation")
		}
		profile, err := decodeDiagnosticReport(observation.Host.Stderr.RawBytes)
		if err != nil || profile.HostOS != "linux" || profile.HostArch != "aarch64" {
			t.Fatal("diagnostic phase/host evidence required", err)
		}
		if i == 0 {
			reference = observation.Result
		} else if difference := sameEvidence(reference, observation.Result); difference != "" {
			failed = warmPreparationFailure(failed, "within-series evidence changed: "+difference)
		}
		{
			finishPath := filepath.Join(evidence, name+"-finish.bin")
			temporalWrite(t, finishPath, observation.Result.runtimeTerminal[:])
			observationPath, diagnosticPath := filepath.Join(evidence, name+"-warm-observation.json"), filepath.Join(evidence, name+"-helper-stderr.txt")
			temporalJSON(t, observationPath, warmPreparationObserve(observation))
			rawObservationPath := filepath.Join(evidence, name+"-observation.json")
			for _, path := range []string{observationPath, rawObservationPath, diagnosticPath, finishPath} {
				inputs[path] = fileHash(t, path)
			}
			run := warmPreparationRun{Telemetry: *observation.Result.Telemetry, Phases: profile.Phases, WallNanos: wall, TerminalToReapNanos: observation.TerminalToReapNanos, EvidenceSHA256: warmPreparationEvidence(observation.Result), Exited: observation.within(time.Second) && observation.Result.Termination == "exit" && observation.Result.ExitCode != nil && *observation.Result.ExitCode == 0, Reaped: observation.Result.Reaped, ObservationPath: observationPath, RawObservationPath: rawObservationPath, FinishPath: finishPath, DiagnosticPath: diagnosticPath}
			if i == 0 {
				series.Warmup = &run
			} else {
				series.Runs = append(series.Runs, run)
			}
		}
		if !strings.Contains(string(observation.Result.Stdout), "--- PASS: TestWorkflowTypeEncodingSuite") || !strings.Contains(string(observation.Result.Stdout), "TestPlainASCII/Succeeds") {
			failed = warmPreparationFailure(failed, "unchanged selected workflow did not pass")
		}
	}
	request.DigestTranscript = false
	var retained terminalExitObservation
	_, retainedAdmissionErr := matchedPrepareAdmitted("linux", "/proc/meminfo", "1", func() (Result, error) {
		retained = measureTerminalExitContext(t, t.Context(), request, false)
		return retained.Result, nil
	})
	if retainedAdmissionErr != nil {
		t.Fatal("untimed retention admission refused before execution", retainedAdmissionErr)
	}
	persistDiagnosticRawResult(t, evidence, "untimed-evidence", retained.Result)
	temporalJSON(t, filepath.Join(evidence, "untimed-evidence-observation.json"), retained)
	if retained.Error != "" || sameEvidence(reference, retained.Result) != "" {
		failed = warmPreparationFailure(failed, "untimed canonical retention changed behavior")
	}
	temporalVerifyInputs(t, inputs)
	terminalExitHelperInputsFeatures(t, paths["helper_receipt"], request, crate, []string{"diagnostic-profile"})
	terminalExitModuleInputs(t, paths["module_receipt"], request)
	afterCommit := strings.TrimSpace(string(temporalCommand(t, evidence, "source-commit-after", repo, nil, []string{"git", "rev-parse", "HEAD"}).Stdout.RawBytes))
	if afterCommit != commit {
		failed = warmPreparationFailure(failed, "source commit changed during collection")
	}
	afterBoot, err := hostfs.ReadBounded("/proc/sys/kernel/random/boot_id", 128)
	afterCPU, afterHost := diagnosticPerformanceHostEvidence("/proc/cpuinfo")
	if err != nil || !bytes.Equal(boot, afterBoot) || !afterHost || cpuinfo != afterCPU {
		failed = warmPreparationFailure(failed, "host bindings changed during collection")
	}
	if err := warmPreparationValidate(series, true); err != nil {
		failed = warmPreparationFailure(failed, err.Error())
	}
	series.ComparisonSHA256, err = warmPreparationBehavior(retained.Result, request.Config, false)
	if err != nil {
		failed = warmPreparationFailure(failed, "candidate transcript comparison: "+err.Error())
	}
	failed = warmPreparationPublishCollection(t, evidence, "candidate-collection.json", &series, retained.Result, retained.Error, failed)
	baselinePath := os.Getenv("GOMAD_WASM_WARM_BASELINE_RECEIPT")
	var baseline *warmPreparationSeries
	var baselineSummary map[string]variantTimingRange
	baselineComparison := ""
	if baselinePath == "" {
		failed = warmPreparationFailure(failed, "baseline-material-gap: collect the historical source with its original matching engine/driver; this candidate never overrides engine identity")
	} else {
		if !filepath.IsAbs(baselinePath) {
			t.Fatal("baseline receipt must be absolute")
		}
		before := warmPreparationRead(t, baselinePath, evidence, paths["module_receipt"], request)
		baseline = &before
		baselineSummary = warmPreparationSummary(before)
		baselineComparison = before.ComparisonSHA256
		if err := warmPreparationCompare(before, series); err != nil {
			failed = warmPreparationFailure(failed, err.Error())
		}
		inputs[baselinePath] = fileHash(t, baselinePath)
	}
	summary := warmPreparationSummary(series)
	temporalJSON(t, filepath.Join(evidence, "acceptance.json"), map[string]any{"schema": "gomad3.wasm-warm-preparation-acceptance/v1", "accepted": failed == "", "failure": failed, "candidate": series, "baseline": baseline, "candidate_summary_nanos": summary, "baseline_summary_nanos": baselineSummary, "candidate_comparison_sha256": series.ComparisonSHA256, "baseline_comparison_sha256": baselineComparison, "comparison_policy": "source-bound historical paired JSONL validated through the typed GMWT owner; original raw digests remain separate; no format fallback or modeled-input masking", "artifact_bytes": artifactBytes, "envelope_bytes": envelopeBytes, "parked_mapping_owner_decision_required": summary["cache_read_plus_deserialize"].Median > uint64(time.Second), "boundary": "one unmeasured warm-up, five fresh helpers with same workflow and seed on prewarmed page cache; no artifact hashing in timed windows; full source/build/module/cache bindings verified before and after; terminal receipt through supervisor wait/capture/group reaping", "baseline_policy": "historical matching driver and PinnedEngine only; helper/engine/model/source identities separate, modeled inputs and exact behavior must remain equal"})
	if failed != "" {
		t.Fatalf("warm-preparation acceptance not met: %s; retained %s", failed, evidence)
	}
}

func warmPreparationLegacyTranscript(data []byte, config Config) ([]byte, error) {
	environment, err := NewEnvironment(config)
	if err != nil {
		return nil, err
	}
	if len(data) == 0 {
		return nil, nil
	}
	if uint64(len(data)) > environment.config.Limits.TranscriptBytes || data[len(data)-1] != '\n' {
		return nil, fmt.Errorf("historical transcript bounds or framing")
	}
	rows := bytes.Split(data[:len(data)-1], []byte{'\n'})
	if len(rows)%2 != 0 {
		return nil, fmt.Errorf("unpaired historical transcript")
	}
	var previous uint64
	for i := 0; i < len(rows); i += 2 {
		if len(rows[i]) > frameLimit || len(rows[i+1]) > frameLimit {
			return nil, fmt.Errorf("historical frame bound")
		}
		var call Call
		if err := decodeInput(rows[i], &call, "type", "id", "op", "input"); err != nil {
			return nil, err
		}
		frame, err := newValidatedFrame(rows[i+1])
		if err != nil {
			return nil, err
		}
		keys := []string{"type", "id", "errno", "output"}
		for _, optional := range []string{"unsupported", "message"} {
			if _, ok := frame.fields[optional]; ok {
				keys = append(keys, optional)
			}
		}
		var reply Reply
		if err := frame.decode(&reply, keys...); err != nil {
			return nil, err
		}
		if call.Type != "call" || reply.Type != "reply" || call.ID == 0 || call.ID <= previous || call.ID != reply.ID || strings.HasPrefix(call.Op, "runtime_") || !reply.Unsupported && reply.Message != "" {
			return nil, fmt.Errorf("historical call/reply identity or error evidence")
		}
		previous = call.ID
		tag, input, err := transcriptInput(call, environment.entropyOffset)
		if err != nil {
			return nil, err
		}
		if reply.Unsupported || reply.Errno != 0 {
			if err := decodeInput(reply.Output, &struct{}{}); err != nil {
				return nil, err
			}
		} else if tag >= 3 && tag <= 7 {
			output, errno, message, err := environment.dispatch(call, nil)
			if err != nil {
				return nil, err
			}
			if errno != 0 || message != "" {
				return nil, fmt.Errorf("historical derived output owner refused")
			}
			keys := []string{"values"}
			switch tag {
			case 3, 5:
				keys = []string{"count", "bytes"}
			case 7:
				keys = []string{"data_base64"}
			}
			decoded := reflect.New(reflect.TypeOf(output))
			if err := decodeInput(reply.Output, decoded.Interface(), keys...); err != nil {
				return nil, err
			}
			if !reflect.DeepEqual(output, decoded.Elem().Interface()) {
				return nil, fmt.Errorf("historical derived output differs from captured inputs")
			}
		}
		if tag == 1 && reply.Errno == 0 && !reply.Unsupported {
			var clock struct {
				ID        uint32 `json:"clock_id"`
				Precision uint64 `json:"precision"`
			}
			var output struct {
				Timestamp uint64 `json:"timestamp"`
			}
			if err := decodeInput(call.Input, &clock, "clock_id", "precision"); err != nil {
				return nil, err
			}
			if err := decodeInput(reply.Output, &output, "timestamp"); err != nil {
				return nil, err
			}
			if clock.ID > 1 || clock.ID == 0 && output.Timestamp < config.Clock.EpochNanos {
				return nil, fmt.Errorf("historical clock domain")
			}
			now := output.Timestamp
			if clock.ID == 0 {
				now -= config.Clock.EpochNanos
			}
			if now > math.MaxInt64 {
				return nil, fmt.Errorf("historical clock overflow")
			}
			environment.appendClockRead(call.ID, clock.ID, clock.Precision, now)
			continue
		}
		record, err := encodeTranscriptRecord(tag, call.ID, input, reply)
		if err != nil {
			return nil, err
		}
		environment.appendTranscript(record)
	}
	converted := environment.Transcript()
	if _, err := DecodeTranscript(converted); err != nil {
		return nil, err
	}
	return converted, nil
}

func warmPreparationBehavior(result Result, config Config, historical bool) (string, error) {
	if historical {
		transcript, err := warmPreparationLegacyTranscript(result.Transcript, config)
		if err != nil {
			return "", err
		}
		result.Transcript = transcript
	}
	evidence, err := projectVariantEvidence(result, config, "")
	if err != nil {
		return "", err
	}
	data, err := json.Marshal(struct {
		Evidence variantEvidence
		Finish   [16]byte
	}{evidence, result.runtimeTerminal})
	if err != nil {
		return "", err
	}
	return digest(data), nil
}

func TestWarmPreparationHistoricalClockCountsAndPositions(t *testing.T) {
	config := testConfig()
	config.Profile, config.Clock.ReadStepNanos = CooperativeProfile, 0
	legacy := []byte("{\"type\":\"call\",\"id\":1,\"op\":\"clock_time_get\",\"input\":{\"clock_id\":1,\"precision\":7}}\n{\"type\":\"reply\",\"id\":1,\"errno\":0,\"output\":{\"timestamp\":1}}\n{\"type\":\"call\",\"id\":2,\"op\":\"clock_time_get\",\"input\":{\"clock_id\":1,\"precision\":7}}\n{\"type\":\"reply\",\"id\":2,\"errno\":0,\"output\":{\"timestamp\":1}}\n")
	converted, err := warmPreparationLegacyTranscript(legacy, config)
	if err != nil {
		t.Fatal(err)
	}
	records, err := DecodeTranscript(converted)
	if err != nil || len(records) != 1 || records[0].ID != 1 || records[0].Count != 2 || records[0].Now != 1 || string(records[0].Input) != `{"clock_id":1,"precision":7}` {
		t.Fatal("clock run lost original positions, IDs or counts", err, records)
	}
	for _, data := range [][]byte{
		legacy[:bytes.Index(legacy, []byte(`{"type":"call","id":2`))],
		bytes.ReplaceAll(legacy, []byte(`"timestamp":1`), []byte(`"timestamp":2`)),
		bytes.ReplaceAll(legacy, []byte(`"clock_id":1`), []byte(`"clock_id":2`)),
		bytes.TrimSuffix(legacy, []byte{'\n'}),
	} {
		changed, err := warmPreparationLegacyTranscript(data, config)
		if err == nil && bytes.Equal(converted, changed) {
			t.Fatal("clock count/position/domain or framing change was hidden")
		}
	}
}

func TestWarmPreparationCrossEncodingBehaviorControls(t *testing.T) {
	config := testConfig()
	config.Profile, config.Clock.ReadStepNanos = CooperativeProfile, 0
	environment := newTestEnvironment(t, config)
	var legacy []byte
	for i, operation := range []struct{ op, input string }{
		{"args_sizes_get", `{}`}, {"args_get", `{}`}, {"environ_sizes_get", `{}`}, {"environ_get", `{}`},
		{"random_get", `{"length":2}`}, {"random_get", `{"length":3}`},
		{"clock_time_get", `{"clock_id":1,"precision":7}`}, {"clock_time_get", `{"clock_id":1,"precision":7}`},
		{"clock_time_get", `{"clock_id":0,"precision":7}`},
		{"fd_write", `{"fd":1,"data_base64":"YWI="}`},
		{"fd_read", `{"fd":99,"length":1}`}, {"unknown", `{"value":1}`},
	} {
		request := Call{Type: "call", ID: uint64(i + 1), Op: operation.op, Input: json.RawMessage(operation.input)}
		reply, err := environment.Handle(request)
		if err != nil {
			t.Fatal(err)
		}
		requestData, _ := json.Marshal(request)
		replyData, _ := json.Marshal(reply)
		legacy = append(append(append(append(legacy, requestData...), '\n'), replyData...), '\n')
	}
	code := uint32(0)
	current := Result{Transcript: environment.Transcript(), Stdout: []byte("ab"), Termination: "exit", ExitCode: &code, runtimeTerminal: [16]byte{1}}
	historical := current
	historical.Transcript = legacy
	want, err := warmPreparationBehavior(current, config, false)
	if err != nil {
		t.Fatal(err)
	}
	got, err := warmPreparationBehavior(historical, config, true)
	if err != nil || got != want {
		t.Fatal("same real owner behavior changed solely because of transcript encoding", err)
	}
	for _, test := range []struct{ name, old, replacement string }{
		{"clock-position", `"timestamp":1`, `"timestamp":2`},
		{"clock-precision", `"precision":7`, `"precision":8`},
		{"operation-id", `"call","id":12`, `"call","id":13`},
		{"operation-input", `"value":1`, `"value":2`},
		{"error", `"errno":8`, `"errno":9`},
		{"output", `"written":2`, `"written":1`},
		{"random-reply", `"data_base64":`, `"extra":1,"data_base64":`},
		{"unsupported-message", `operation outside stock WASI profile: unknown`, `operation outside stock WASI profile: other`},
	} {
		t.Run(test.name, func(t *testing.T) {
			changed := historical
			changed.Transcript = bytes.Replace(legacy, []byte(test.old), []byte(test.replacement), 1)
			if bytes.Equal(changed.Transcript, legacy) {
				t.Fatal("control did not mutate its real input")
			}
			got, err := warmPreparationBehavior(changed, config, true)
			if err == nil && got == want {
				t.Fatal("cross-encoding projection hid changed behavior")
			}
		})
	}
	for _, field := range []string{"stdout", "stderr", "network", "choices", "finish", "virtual-time", "message", "termination", "exit"} {
		t.Run(field, func(t *testing.T) {
			changed := historical
			switch field {
			case "stdout":
				changed.Stdout = []byte("other")
			case "stderr":
				changed.Stderr = []byte("other")
			case "network":
				changed.NetworkTranscript = []byte("other")
			case "choices":
				changed.ChoiceTrace.Bytes = []byte("other")
			case "finish":
				changed.runtimeTerminal[0]++
			case "virtual-time":
				changed.VirtualTimeElapsedNanos++
			case "message":
				changed.Message = "other"
			case "termination":
				changed.Termination = "other"
			case "exit":
				code := uint32(1)
				changed.ExitCode = &code
			}
			got, err := warmPreparationBehavior(changed, config, true)
			if err != nil || got == want {
				t.Fatal("non-WASI evidence lost", err)
			}
		})
	}
}

func TestWarmPreparationHistoricalTranscriptOwnerProjection(t *testing.T) {
	config := testConfig()
	config.Args = []string{"guest"}
	environment := newTestEnvironment(t, config)
	legacy := []byte("{\"type\":\"call\",\"id\":1,\"op\":\"args_get\",\"input\":{}}\n{\"type\":\"reply\",\"id\":1,\"errno\":0,\"output\":{\"values\":[\"Z3Vlc3Q=\"]}}\n")
	call(t, environment, "args_get", `{}`)
	converted, err := warmPreparationLegacyTranscript(legacy, config)
	if err != nil || !bytes.Equal(converted, environment.Transcript()) {
		t.Fatal("historical derived arguments did not match the typed owner", err)
	}
	if _, err := DecodeTranscript(legacy); err == nil {
		t.Fatal("production decoder must not accept historical JSONL")
	}
}

func TestWarmPreparationCollectionRetainsBinaryArtifactName(t *testing.T) {
	root := t.TempDir()
	series := warmPreparationFixture()
	result := Result{Transcript: bytes.Clone(transcriptHeader)}
	persistDiagnosticRawResult(t, root, "untimed-evidence", result)
	if failed := warmPreparationPublishCollection(t, root, "candidate-collection.json", &series, result, "", "unqualified control"); failed != "unqualified control" {
		t.Fatal(failed)
	}
	var collection warmPreparationCollection
	data, err := os.ReadFile(series.CollectionPath)
	if err != nil || json.Unmarshal(data, &collection) != nil {
		t.Fatal("missing collection marker", err)
	}
	if collection.CanonicalPath != filepath.Join(root, "untimed-evidence-canonical.bin") || series.Inputs[collection.CanonicalPath] != digest(result.Transcript) {
		t.Fatal("collection did not bind the binary owner's actual artifact")
	}
}

func TestWarmPreparationHistoricalTranscriptRefusesLostEvidence(t *testing.T) {
	config := testConfig()
	config.Args = []string{"guest"}
	legacy := []byte("{\"type\":\"call\",\"id\":1,\"op\":\"args_get\",\"input\":{}}\n{\"type\":\"reply\",\"id\":1,\"errno\":0,\"output\":{\"values\":[\"Z3Vlc3Q=\"]}}\n")
	for _, test := range []struct{ name, old, replacement string }{
		{"derived-argument", "Z3Vlc3Q=", "b3RoZXI="},
		{"reply-id", `"reply","id":1`, `"reply","id":2`},
		{"missing-reply", string(legacy[bytes.IndexByte(legacy, '\n')+1:]), ""},
		{"duplicate-field", `"op":"args_get"`, `"op":"args_get","op":"args_get"`},
		{"unknown-envelope-field", `"errno":0`, `"errno":0,"extra":true`},
		{"hidden-message", `"errno":0`, `"errno":0,"message":"changed"`},
		{"noncanonical-runtime", "args_get", "runtime_choice"},
	} {
		t.Run(test.name, func(t *testing.T) {
			changed := bytes.Replace(legacy, []byte(test.old), []byte(test.replacement), 1)
			if _, err := warmPreparationLegacyTranscript(changed, config); err == nil {
				t.Fatal("historical evidence change was discarded")
			}
		})
	}
	for _, replacement := range []Config{func() Config { c := config; c.Args = []string{"other"}; return c }(), func() Config { c := config; c.Environment = []string{"OTHER=value"}; return c }()} {
		op := "args_get"
		if replacement.Environment[0] == "OTHER=value" {
			op = "environ_get"
		}
		environment := newTestEnvironment(t, config)
		request := Call{Type: "call", ID: 1, Op: op, Input: json.RawMessage(`{}`)}
		reply, err := environment.Handle(request)
		if err != nil {
			t.Fatal(err)
		}
		requestData, _ := json.Marshal(request)
		replyData, _ := json.Marshal(reply)
		data := append(append(append(requestData, '\n'), replyData...), '\n')
		if _, err := warmPreparationLegacyTranscript(data, replacement); err == nil {
			t.Fatal("derived captured configuration change was discarded", op)
		}
	}
}
