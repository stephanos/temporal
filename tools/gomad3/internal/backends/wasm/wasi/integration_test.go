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

	runnerbackend "go.temporal.io/server/tools/gomad3/internal/backends"
	"go.temporal.io/server/tools/gomad3/internal/backends/wasm/internal/sqlitebusy"
	"go.temporal.io/server/tools/gomad3/internal/backends/wasm/internal/stockgo"
	"go.temporal.io/server/tools/gomad3/internal/backends/wasm/toolchain"
	captured "go.temporal.io/server/tools/gomad3/internal/evidence/artifact/capture"
	hostexec "go.temporal.io/server/tools/gomad3/internal/host/exec"
	hostfs "go.temporal.io/server/tools/gomad3/internal/host/fs"
	"go.temporal.io/server/tools/gomad3/internal/model/choice"
	"go.temporal.io/server/tools/gomad3/internal/model/clock"
	gomad3sim "go.temporal.io/server/tools/gomad3/internal/model/cluster"
	netmodel "go.temporal.io/server/tools/gomad3/internal/model/network"
	"go.temporal.io/server/tools/gomad3/internal/model/volume"
	"go.temporal.io/server/tools/gomad3/internal/model/world"
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
		if err := temporalStageFixture(root, filepath.Join("tools", "gomad3", "backends", "wasm", "testdata", "prototype_temporal"), root, staged); err != nil {
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
	command := exec.Command(filepath.Join(root, "bin/go"), "build", "-trimpath", "-buildvcs=false", "-overlay", overlay, "-o", module, fixture)
	command.Env = append(os.Environ(), "GOROOT="+root, "GOOS=wasip1", "GOARCH=wasm", "GOEXPERIMENT=nogreenteagc", "GOWORK=off", "GOENV=off", "GOFLAGS=", "GOTOOLCHAIN=local", "CGO_ENABLED=0")
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("build actual cooperative guest: %v\n%s", err, output)
	}
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
	for name, payload := range map[string][]byte{"report.json": data, "stdout": baseline.Stdout, "stderr": baseline.Stderr, "choices.bin": baseline.ChoiceTrace.Bytes, "diagnostics.bin": baseline.DiagnosticTrace.Bytes, "wasi.jsonl": baseline.Transcript, "finish.bin": baseline.runtimeTerminal[:]} {
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
		for name, expected := range map[string]string{"stdout": report.StdoutSHA256, "stderr": report.StderrSHA256, "choices.bin": report.ChoiceSHA256, "diagnostics.bin": report.DiagnosticSHA256, "wasi.jsonl": report.WASISHA256, "finish.bin": report.TerminalSHA256} {
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
	for _, frozen := range []bool{false, true} {
		t.Run(map[bool]string{false: "read-step", true: "frozen"}[frozen], func(t *testing.T) {
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
		"network-streams.wasi.jsonl":   observed.Transcript,
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
			for _, operation := range []string{"fd_pread", "fd_pwrite", "fd_sync"} {
				if !bytes.Contains(appended.Transcript, []byte(`"op":"`+operation+`"`)) {
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
	fixtureBody, err := os.ReadFile(filepath.Join(root, "tools/gomad3/internal/backends/wasm/testdata/temporal_storage/recovery_test.go"))
	if err != nil {
		t.Fatal(err)
	}
	if fixture == "temporal_broker" {
		brokerBody, err := os.ReadFile(filepath.Join(root, "tools/gomad3/internal/backends/wasm/testdata/temporal_broker/replication_test.go"))
		if err != nil {
			t.Fatal(err)
		}
		fixtureBody = append(fixtureBody, brokerBody...)
	}
	staged := filepath.Join(".tmp", "wasm-backend-temporal", fixture+"-"+digest(fixtureBody)[:16])
	if fixture == "temporal_broker" {
		temporalStageBrokerFixture(t, root, staged)
	} else {
		if err := temporalStageFixture(root, filepath.Join("tools", "gomad3", "backends", "wasm", "testdata", fixture), root, staged); err != nil {
			t.Fatal(err)
		}
	}
	compiler, overlay, runtimeIdentity := prototypeToolchain(t)
	listed := temporalCommand(t, evidence, fixture+"-inputs", root, temporalBuildEnvironment(compiler), []string{
		compiler, "list", "-mod=readonly", "-tags", "test_dep,disable_grpc_modules,sqlite3_dotlk", "-deps", "-test", "-json", "./" + filepath.ToSlash(staged),
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
	for _, name := range []string{"go.mod", "go.sum", "tools/gomad3/internal/backends/wasm/testdata/temporal_storage/recovery_test.go"} {
		inputs[filepath.Join(root, name)] = fileHash(t, filepath.Join(root, name))
	}
	if fixture == "temporal_broker" {
		file := filepath.Join(root, "tools/gomad3/internal/backends/wasm/testdata/temporal_broker/replication_test.go")
		inputs[file] = fileHash(t, file)
	}
	t.Cleanup(func() { temporalVerifyInputs(t, inputs) })
	module := filepath.Join(evidence, fixture+".wasm")
	temporalCommand(t, evidence, fixture+"-compile", root,
		append(temporalBuildEnvironment(compiler), "GOROOT="+filepath.Dir(filepath.Dir(compiler))),
		[]string{compiler, "test", "-c", "-mod=readonly", "-trimpath", "-overlay", overlay, "-o", module,
			"-tags", "test_dep,disable_grpc_modules,sqlite3_dotlk", "./" + filepath.ToSlash(staged)})
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

func TestTemporalCommittedWorkflowFreshGuestCrashReopen(t *testing.T) {
	request, evidence := temporalRecoveryRequest(t, "temporal_storage")
	request.Config.Args = []string{"recovery.test", "-test.v", "-test.count=1", "-test.timeout=1h", "-test.run=^TestCommittedWorkflowCrashRecovery$", "-persistenceType=sql", "-persistenceDriver=sqlite"}
	request.Config.Environment = append(request.Config.Environment, "GOMAD_TEMPORAL_PHASE=commit")
	request.StorageCrashLimits = &volume.CrashEnumerationLimits{States: 128, Operations: 256, Depth: 256, Bytes: 256 << 20, Work: 1 << 20}
	temporalRetainRequest(t, evidence, "temporal-commit", request)
	newBroker := func() *Broker {
		b, err := NewBroker(BrokerConfig{Nodes: []BrokerNode{{ID: "temporal", Incarnation: 1}}, Seed: 7,
			Limits: world.Limits{MaxRequests: 1000000, MaxEvents: 1000000, MaxQueuedEvents: 65536, MaxTransitions: 4000000, MaxPayloadBytes: 512 << 20, MaxStringBytes: 1024}})
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
		root = filepath.Join(root, "tools", "gomad3", "backends", "native", "conformance", "testdata")
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
		if err := temporalStageFixture(root, filepath.Join("tools", "gomad3", "backends", "wasm", "testdata", owner), root, filepath.Join(".tmp", "wasm-backend-temporal", staged)); err != nil {
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
				temporalWrite(t, prefix+".transcript.jsonl", result.Transcript)
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
	for _, name := range []string{"HOME", "TMPDIR", "GOCACHE", "GOPATH", "GOMODCACHE"} {
		if value, found := os.LookupEnv(name); found {
			environment = append(environment, name+"="+value)
		}
	}
	return append(environment, "PATH="+filepath.Dir(compiler)+":"+os.Getenv("PATH"), "GOOS=wasip1", "GOARCH=wasm", "CGO_ENABLED=0", "GOWORK=off", "GOENV=off", "GOFLAGS=", "GOEXPERIMENT=nogreenteagc", "GOTOOLCHAIN=local")
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
	fixtureSource, err := os.ReadFile(filepath.Join(root, "tools", "gomad3", "backends", "wasm", "testdata", "network_streams", "temporal_retry_test.go"))
	if err != nil {
		t.Fatal(err)
	}
	staged := filepath.Join(".tmp", "wasm-backend-temporal", "network-streams-retry-"+digest(fixtureSource)[:12])
	if err := temporalStageFixture(root, filepath.Join("tools", "gomad3", "backends", "wasm", "testdata", "network_streams"), root, staged); err != nil {
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
		for suffix, data := range map[string][]byte{"stdout": result.Stdout, "stderr": result.Stderr, "wasi.jsonl": result.Transcript, "network.json": result.NetworkTranscript, "choices": result.ChoiceTrace.Bytes} {
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
		data, err := os.ReadFile(filepath.Join(root, "tools/gomad3/internal/backends/wasm/testdata", name))
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
		"recovery_test.go":    filepath.Join(root, "tools/gomad3/internal/backends/wasm/testdata/temporal_storage/recovery_test.go"),
		"replication_test.go": filepath.Join(root, "tools/gomad3/internal/backends/wasm/testdata/temporal_broker/replication_test.go"),
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
	results, errors := temporalRunPair(t, evidence, "cluster-crash", requests, true)
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
	results, errors = temporalRunPair(t, evidence, "cluster-reopened", requests, false)
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
	temporalReplayClusters(t, evidence, brokerConfig, commitRequests, crashed, results, encodedWorld)
	fmt.Printf("independent Temporal clusters recovered after directional partition/heal\n")
}

func temporalRunPair(t *testing.T, evidence, label string, requests []Request, warmFirst bool) ([]Result, []error) {
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

func temporalReplayClusters(t *testing.T, evidence string, configuration BrokerConfig, requests []Request, crashed, recovered []Result, expected []byte) {
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
	actual, errors := temporalRunPair(t, evidence, "cluster-prefix-replay", requests, true)
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
	actual, errors = temporalRunPair(t, evidence, "cluster-full-replay", requests, false)
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
	for _, line := range bytes.Split(retained.Transcript, []byte{'\n'}) {
		if len(line) == 0 {
			continue
		}
		var frame struct {
			Type string `json:"type"`
			Op   string `json:"op"`
		}
		if err := json.Unmarshal(line, &frame); err != nil {
			t.Fatal(err)
		}
		if frame.Type == "call" {
			operations[frame.Op]++
		}
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
	if uint64(len(data)) > maximum || len(data) > 0 && data[len(data)-1] != '\n' {
		return counts, invalid("retained canonical byte bound or framing")
	}
	var last, pending uint64
	for len(data) > 0 {
		end := bytes.IndexByte(data, '\n')
		line := data[:end]
		data = data[end+1:]
		if pending == 0 {
			var call Call
			if err := decodeInput(line, &call, "type", "id", "op", "input"); err != nil {
				return counts, err
			}
			if call.Type != "call" || call.ID <= last || call.Op == "" {
				return counts, invalid("retained call sequence")
			}
			var fields map[string]json.RawMessage
			if err := json.Unmarshal(call.Input, &fields); err != nil || fields == nil {
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
			local := strings.HasPrefix(call.Op, "runtime_") && call.Op != "runtime_net_stream" || call.Op == "clock_time_get" || call.Op == "sched_yield"
			if call.Op == "poll_oneoff" {
				var subscriptions []map[string]json.RawMessage
				if raw, ok := fields["subscriptions"]; ok && len(raw) > 0 && raw[0] == '[' && json.Unmarshal(raw, &subscriptions) == nil {
					local = true
					for _, subscription := range subscriptions {
						var kind uint64
						raw, ok := subscription["type"]
						if !ok || bytes.Equal(raw, []byte("null")) || json.Unmarshal(raw, &kind) != nil || kind != 0 {
							local = false
							break
						}
					}
				}
			}
			counts.Calls++
			if local {
				counts.Local++
			} else {
				counts.Remote++
				namespace := "wasi_snapshot_preview1"
				if call.Op == "runtime_net_stream" {
					namespace = "gomad_wasm_v2"
				}
				counts.Operations[namespace+"/"+call.Op+"/"+fd]++
			}
			last, pending = call.ID, call.ID
		} else {
			if err := validateJSON(line); err != nil {
				return counts, err
			}
			var fields map[string]json.RawMessage
			if err := json.Unmarshal(line, &fields); err != nil {
				return counts, invalid("retained reply object")
			}
			keys := []string{"type", "id", "errno", "output"}
			for _, key := range []string{"unsupported", "message"} {
				if _, ok := fields[key]; ok {
					keys = append(keys, key)
				}
			}
			var reply Reply
			if err := decodeFields(line, fields, &reply, keys...); err != nil {
				return counts, err
			}
			if reply.Type != "reply" || reply.ID != pending {
				return counts, invalid("retained reply sequence")
			}
			pending = 0
		}
	}
	if pending != 0 {
		return counts, invalid("unfinished retained call")
	}
	return counts, nil
}

func operationCountPair(id int, op, input string) string {
	return fmt.Sprintf("{\"type\":\"call\",\"id\":%d,\"op\":%q,\"input\":%s}\n{\"type\":\"reply\",\"id\":%d,\"errno\":0,\"output\":{}}\n", id, op, input, id)
}

func TestCanonicalRemoteOperationCounts(t *testing.T) {
	data := []byte(operationCountPair(1, "clock_time_get", `{"clock_id":0}`) + operationCountPair(3, "sched_yield", `{}`) + operationCountPair(4, "poll_oneoff", `{"subscriptions":[{"type":0}]}`) + operationCountPair(5, "poll_oneoff", `{"subscriptions":[{"type":0},{"type":1,"fd":4}]}`) + operationCountPair(6, "fd_write", `{"fd":1}`) + operationCountPair(7, "fd_write", `{"fd":2}`) + operationCountPair(8, "fd_pwrite", `{"fd":4}`) + operationCountPair(9, "path_open", `{"fd":3}`))
	original := bytes.Clone(data)
	got, err := countCanonicalRemoteOperations(data, uint64(len(data)))
	if err != nil || got.Calls != 8 || got.Local != 3 || got.Remote != 5 || got.Operations["wasi_snapshot_preview1/poll_oneoff/no-root-fd"] != 1 || got.Operations["wasi_snapshot_preview1/fd_write/fd=1"] != 1 || got.Operations["wasi_snapshot_preview1/fd_write/fd=2"] != 1 || got.Operations["wasi_snapshot_preview1/fd_pwrite/fd=4"] != 1 || got.Operations["wasi_snapshot_preview1/path_open/fd=3"] != 1 {
		t.Fatalf("actual canonical operation counts = %+v, %v", got, err)
	}
	if !bytes.Equal(original, data) {
		t.Fatal("counter mutated canonical input")
	}
}

func TestCanonicalRemoteOperationCountsRejectsInvalid(t *testing.T) {
	valid := operationCountPair(1, "fd_write", `{"fd":1}`)
	for name, data := range map[string]string{
		"duplicate":             strings.Replace(valid, `"fd":1`, `"fd":1,"f\u0064":2`, 1),
		"nested duplicate":      strings.Replace(valid, `"fd":1`, `"fd":1,"x":{"a":0,"a":1}`, 1),
		"null fd":               strings.Replace(valid, `"fd":1`, `"fd":null`, 1),
		"fd type":               strings.Replace(valid, `"fd":1`, `"fd":"1"`, 1),
		"fd overflow":           strings.Replace(valid, `"fd":1`, `"fd":4294967296`, 1),
		"unknown call key":      strings.Replace(valid, `"op":"fd_write"`, `"op":"fd_write","extra":0`, 1),
		"wrong reply":           strings.Replace(valid, `"type":"reply","id":1`, `"type":"reply","id":2`, 1),
		"unfinished":            strings.Split(valid, "\n")[0] + "\n",
		"nonincreasing":         valid + valid,
		"wrong tag":             strings.Replace(valid, `"type":"call"`, `"type":"unknown"`, 1),
		"malformed":             "{\n",
		"missing final newline": strings.TrimSuffix(valid, "\n"),
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := countCanonicalRemoteOperations([]byte(data), uint64(len(data))); err == nil {
				t.Fatal("invalid retained canonical trace accepted")
			}
		})
	}
	if _, err := countCanonicalRemoteOperations([]byte(valid), uint64(len(valid)-1)); err == nil {
		t.Fatal("retained byte bound ignored")
	}
}

func TestCanonicalRemoteOperationClassification(t *testing.T) {
	for _, sample := range []struct {
		op, input string
		local     bool
	}{
		{"runtime_decision", `{}`, true}, {"runtime_net_stream", `{}`, false},
		{"poll_oneoff", `{"subscriptions":[]}`, true}, {"poll_oneoff", `{}`, false},
		{"poll_oneoff", `{"subscriptions":null}`, false}, {"poll_oneoff", `{"subscriptions":[{"type":null}]}`, false},
		{"poll_oneoff", `{"subscriptions":[{"type":0.0}]}`, false}, {"poll_oneoff", `{"subscriptions":[0]}`, false},
	} {
		data := []byte(operationCountPair(1, sample.op, sample.input))
		counts, err := countCanonicalRemoteOperations(data, uint64(len(data)))
		if err != nil || (counts.Local == 1) != sample.local || counts.Local+counts.Remote != 1 {
			t.Fatalf("%s/%s = %+v, %v", sample.op, sample.input, counts, err)
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
	for name, expected := range map[string][]byte{"finish.bin": result.runtimeTerminal[:], "canonical.jsonl": result.Transcript, "choice.bin": result.ChoiceTrace.Bytes} {
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
	temporalWrite(t, filepath.Join(root, name+"-canonical.jsonl"), result.Transcript)
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
