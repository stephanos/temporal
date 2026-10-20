package wasi

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"maps"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	runnerbackend "go.temporal.io/server/tools/gomad/internal/backends"
	"go.temporal.io/server/tools/gomad/internal/backends/wasm/internal/stockgo"
	"go.temporal.io/server/tools/gomad/internal/backends/wasm/toolchain"
	captured "go.temporal.io/server/tools/gomad/internal/evidence/artifact/capture"
	hostexec "go.temporal.io/server/tools/gomad/internal/host/exec"
	hostfs "go.temporal.io/server/tools/gomad/internal/host/fs"
	"go.temporal.io/server/tools/gomad/internal/model/choice"
	gomadsim "go.temporal.io/server/tools/gomad/internal/model/cluster"
	netmodel "go.temporal.io/server/tools/gomad/internal/model/network"
	"go.temporal.io/server/tools/gomad/internal/model/volume"
	"go.temporal.io/server/tools/gomad/internal/model/world"
)

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
	compiler = os.Getenv("GOMAD_STOCK_GO")
	if !filepath.IsAbs(compiler) {
		t.Fatal("prototype gates require an absolute GOMAD_STOCK_GO (Go 1.27.1)")
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
// GiBs of host memory each, fixture guests very little. The bounds are read
// on first use, after wasmmeasure or the test has set them.
var (
	prototypeLight = sync.OnceValue(func() chan struct{} {
		return make(chan struct{}, prototypeLimit("GOMAD_WASM_PROTOTYPE_PARALLEL", 4))
	})
	prototypeHeavy = sync.OnceValue(func() chan struct{} {
		return make(chan struct{}, prototypeLimit("GOMAD_WASM_PROTOTYPE_HEAVY", 4))
	})
	prototypeWarm sync.Mutex
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
	pool := prototypeLight()
	if info, err := os.Stat(request.ModulePath); err == nil && info.Size() > 64<<20 {
		pool = prototypeHeavy()
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

func distinctChoiceTraces(baselines map[uint64]Result) int {
	seen := map[[32]byte]bool{}
	for _, result := range baselines {
		seen[result.ChoiceTrace.SHA256] = true
	}
	return len(seen)
}

func prototypeFixtureConfig() Config {
	config := testConfig()
	config.Limits.Calls = 1000000
	config.Limits.TranscriptBytes = 64 << 20
	return config
}

func prototypeTemporalConfig(t *testing.T, root, cwd string, args ...string) Config {
	t.Helper()
	config, err := TemporalConfig(root, cwd, args...)
	if err != nil {
		t.Fatal(err)
	}
	return config
}

func cooperativeRequest(t *testing.T, fixture string) Request {
	t.Helper()
	if os.Getenv("GOMAD_STOCK_GO") == "" {
		t.Fatal("qualified guest gate requires explicit GOMAD_STOCK_GO")
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

// requireSplitServedOnlyOutput proves a split-owned broker storage run used the
// Rust volume: Go's filesystem served exactly the guest's fd 1/2 writes.
func requireSplitServedOnlyOutput(t *testing.T, label string, result Result) {
	t.Helper()
	writes := uint64(0)
	for op, count := range result.GoFilesystemCalls {
		if op != "fd_write" {
			t.Fatalf("%s served %s in Go: %v", label, op, result.GoFilesystemCalls)
		}
		writes += count
	}
	records, err := DecodeTranscript(result.Transcript)
	if err != nil {
		t.Fatal(err)
	}
	output := uint64(0)
	for _, record := range records {
		var input struct {
			FD uint32 `json:"fd"`
		}
		if (record.Op == "fd_write" || record.Op == "fd_pwrite") && json.Unmarshal(record.Input, &input) == nil && (input.FD == 1 || input.FD == 2) {
			output++
		}
	}
	// A digest-only run retains no transcript bytes to count; its Go
	// filesystem may still serve only fd_write.
	digestOnly := len(result.Transcript) == 0 && result.TranscriptSHA256 != [32]byte{}
	if result.GoFilesystemCalls == nil || !digestOnly && writes != output {
		t.Fatalf("%s Go fs-served %v, want only its %d fd 1/2 writes", label, result.GoFilesystemCalls, output)
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
	history := func(value temporalRecoveryObservation) []gomadsim.HistoryOperation {
		return []gomadsim.HistoryOperation{{ID: "committed", Actor: "workflow", Kind: "workflow-history", Invocation: 1, Completion: 2, Output: value.History}}
	}
	exact, err := gomadsim.ExactHistory(name+".history", history(expected), history(actual), 1<<20)
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
		ExpectedHistory, ActualHistory []gomadsim.HistoryOperation
		ExpectedEvents, ActualEvents   []string
		ExpectedCluster, ActualCluster string
	}{history(expected), history(actual), ids(expected), ids(actual), expected.ClusterID, actual.ClusterID})
	delivery, err := gomadsim.NoDuplicateOrLost(name+".events", ids(expected), ids(actual), 1<<20)
	if err != nil || !delivery.Passed {
		t.Fatalf("detached Temporal duplicate/lost oracle: passed=%v err=%v", delivery.Passed, err)
	}
	if expected.ClusterID != actual.ClusterID {
		t.Fatal("recovered Temporal cluster identity changed")
	}
	temporalJSON(t, filepath.Join(evidence, name+"-oracles.json"), []gomadsim.OracleResult{exact, delivery})
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
	if result.StorageImage != nil {
		t.Logf("storage recovery image %s: bytes=%d", label, result.StorageImage.Bytes())
	}
	if result.Termination != "" {
		// Every per-process Temporal storage gate run is a split-owned broker run.
		requireSplitServedOnlyOutput(t, label, result)
	}
	temporalRetainRecoveryEvidence(t, evidence, label, result)
}

// temporalRetainClusterRun retains an in-process cluster incarnation. The
// single helper owns every guest call: no Go filesystem is attached to the
// incarnation and the session has relayed no guest callback frame.
func temporalRetainClusterRun(t *testing.T, evidence, label string, result Result, guestCallbacks uint64) {
	t.Helper()
	if result.StorageImage != nil {
		t.Logf("storage recovery image %s: bytes=%d", label, result.StorageImage.Bytes())
	}
	if err := clusterServedOnlyByHelper(result, guestCallbacks); err != nil {
		t.Fatalf("%s %v", label, err)
	}
	temporalRetainRecoveryEvidence(t, evidence, label, result)
}

func clusterServedOnlyByHelper(result Result, guestCallbacks uint64) error {
	if result.GoFilesystemCalls != nil {
		return fmt.Errorf("in-process cluster incarnation has a Go filesystem: %v", result.GoFilesystemCalls)
	}
	if guestCallbacks != 0 {
		return fmt.Errorf("in-process cluster relayed %d guest callbacks to Go", guestCallbacks)
	}
	return nil
}

func temporalRetainRecoveryEvidence(t *testing.T, evidence, label string, result Result) {
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

func temporalVerifyInputs(t *testing.T, inputs map[string]string) {
	t.Helper()
	for name, expected := range inputs {
		if actual := fileHash(t, name); actual != expected {
			t.Fatalf("source input changed during admission: %s: expected %s, got %s", name, expected, actual)
		}
	}
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

type temporalClusterPair struct {
	session *ClusterSession
	names   []string
}

func temporalReopenRequests(requests []Request) []Request {
	reopened := append([]Request(nil), requests...)
	for i := range reopened {
		reopened[i].Config.Environment = append([]string(nil), requests[i].Config.Environment...)
		for j, value := range reopened[i].Config.Environment {
			if value == "GOMAD_TEMPORAL_PHASE=commit" {
				reopened[i].Config.Environment[j] = "GOMAD_TEMPORAL_PHASE=reopen"
			}
		}
		reopened[i].StorageImage = nil
	}
	return reopened
}

func temporalOpenClusterPair(t *testing.T, configuration BrokerConfig, requests, reopened []Request) *temporalClusterPair {
	t.Helper()
	pair := &temporalClusterPair{}
	cluster := ClusterRequest{Broker: configuration, Boots: map[string]Request{}, OutputCrashGroup: &ClusterCrashGroup{Marker: []byte(temporalObservationPrefix)}}
	for i, request := range requests {
		name := configuration.Nodes[i].ID
		pair.names = append(pair.names, name)
		request.Broker, request.StorageImage = nil, nil
		request.ReturnStorageImage = true
		request.StdoutMirror = &temporalGuestWarm{test: t, node: name}
		cluster.Instances = append(cluster.Instances, ClusterInstance{ID: name, Request: request})
		cluster.OutputCrashGroup.Nodes = append(cluster.OutputCrashGroup.Nodes, name)
		recovery := reopened[i]
		recovery.Broker, recovery.StorageImage = nil, nil
		recovery.ReturnStorageImage = true
		recovery.StdoutMirror = &temporalGuestWarm{test: t, node: name}
		cluster.Boots[name+"-reopen"] = recovery
	}
	var err error
	pair.session, err = OpenCluster(t.Context(), cluster)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := pair.session.Shutdown(ctx); err != nil {
			t.Logf("cluster cleanup: %v", err)
		}
	})
	return pair
}

func (pair *temporalClusterPair) run(t *testing.T, evidence, label string, requests []Request, incarnation uint64, observers ...temporalPairObserver) ([]Result, []error) {
	t.Helper()
	return temporalObservedPair(t, evidence, label, requests, observers, func() ([]Result, []error) {
		budget, err := temporalPairBudget(os.Getenv("GOMAD_WASM_TEMPORAL_RECOVERY_DIAGNOSTIC_BUDGET"))
		if err != nil {
			t.Fatal(err)
		}
		ctx, cancel := context.WithTimeout(t.Context(), budget)
		defer cancel()
		started := time.Now()
		results, failures := make([]Result, len(requests)), make([]error, len(requests))
		for i, name := range pair.names {
			boot := name
			if incarnation > 1 {
				boot += "-reopen"
			}
			if err := pair.session.Start(ctx, name, incarnation, boot, requests[i].Runtime); err != nil {
				failures[i] = err
				return results, failures
			}
		}
		for i, name := range pair.names {
			results[i], failures[i] = pair.session.Wait(ctx, name, incarnation)
			temporalRetainClusterRun(t, evidence, name+"-"+label, results[i], pair.session.FrameCounts().GuestCallbacks)
			t.Logf("%s %s terminated in shared Rust helper: %s %s", name, label, results[i].Termination, results[i].Message)
		}
		temporalJSON(t, filepath.Join(evidence, label+"-host-budget.json"), struct {
			BudgetNanos, ElapsedNanos        int64
			Expired                          bool
			NetworkOwner                     string
			NetworkOwnerImplementationSHA256 string
		}{int64(budget), time.Since(started).Nanoseconds(), ctx.Err() == context.DeadlineExceeded, "rust-cluster", requests[0].HelperSHA256})
		return results, failures
	})
}

func (pair *temporalClusterPair) finish(t *testing.T, evidence, label string, observers ...temporalPairObserver) ClusterResult {
	t.Helper()
	result, err := pair.session.Finish(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	temporalJSON(t, filepath.Join(evidence, label+"-cluster-process.json"), pair.session.CompletedResult())
	for _, observer := range observers {
		if observer.Cluster != nil {
			observer.Cluster(t, label, pair.session.CompletedResult())
		}
	}
	if result.Frames.GuestCallbacks != 0 {
		t.Fatalf("cluster made %d guest callbacks to Go", result.Frames.GuestCallbacks)
	}
	return result
}

func temporalRunPair(t *testing.T, evidence, label string, requests []Request, warmFirst bool, observers ...temporalPairObserver) ([]Result, []error) {
	t.Helper()
	return temporalObservedPair(t, evidence, label, requests, observers, func() ([]Result, []error) {
		return temporalRunPairExecution(t, evidence, label, requests, warmFirst)
	})
}

func temporalReplayClusters(t *testing.T, evidence string, configuration BrokerConfig, requests []Request, crashed, recovered []Result, expected []byte, expectedNetwork netmodel.Record, observers ...temporalPairObserver) {
	t.Helper()
	if requests[0].diagnosticConfigurationRevision == "" && requests[0].guestProfile == nil {
		temporalReplayClustersInProcess(t, evidence, configuration, requests, crashed, recovered, expected, expectedNetwork, observers...)
		return
	}
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

func temporalReplayClustersInProcess(t *testing.T, evidence string, configuration BrokerConfig, requests []Request, crashed, recovered []Result, expected []byte, expectedNetwork netmodel.Record, observers ...temporalPairObserver) {
	t.Helper()
	initial := append([]Request(nil), requests...)
	reopened := temporalReopenRequests(requests)
	for i := range initial {
		if !crashed[i].RuntimeInterrupted {
			t.Fatal("cluster replay lacks interrupted choice evidence")
		}
		prefix, err := choice.ProjectInterruptedReplayPlan(crashed[i].InterruptedChoiceBytes, initial[i].Runtime.Choice.Limit, initial[i].Runtime.Choice.ExecutionIdentity)
		if err != nil {
			t.Fatal(err)
		}
		control, choices := *initial[i].Runtime, *initial[i].Runtime.Choice
		choices.Mode, choices.Tape = choice.ModePrefix, &prefix
		control.Choice, initial[i].Runtime = &choices, &control
		tape, err := choice.ProjectReplayPlan(recovered[i].ChoiceTrace, reopened[i].Runtime.Choice.ExecutionIdentity)
		if err != nil {
			t.Fatal(err)
		}
		recoveryControl, recoveryChoices := *reopened[i].Runtime, *reopened[i].Runtime.Choice
		recoveryChoices.Mode, recoveryChoices.Tape = choice.ModeReplay, &tape
		recoveryControl.Choice, reopened[i].Runtime = &recoveryChoices, &recoveryControl
	}
	pair := temporalOpenClusterPair(t, configuration, initial, reopened)
	actual, failures := pair.run(t, evidence, "cluster-prefix-replay", initial, 1, observers...)
	for i, name := range pair.names {
		if failures[i] != nil || actual[i].Termination != "crash" || !actual[i].Reaped || actual[i].StorageImage == nil {
			t.Fatalf("%s interrupted cluster replay: %s %s %v", name, actual[i].Termination, actual[i].Message, failures[i])
		}
		if diff := sameEvidence(crashed[i], actual[i]); diff != "" {
			t.Fatalf("%s interrupted cluster replay differs in %s", name, diff)
		}
		if diff := sameStorageEvidence(crashed[i], actual[i]); diff != "" {
			t.Fatalf("%s interrupted cluster replay storage differs in %s", name, diff)
		}
		reopened[i].StorageImage = actual[i].StorageImage
		reopened[i].Broker = &BrokerSession{Node: BrokerNode{ID: name, Incarnation: 2}}
	}
	actual, failures = pair.run(t, evidence, "cluster-full-replay", reopened, 2, observers...)
	for i, name := range pair.names {
		if failures[i] != nil {
			t.Fatal(failures[i])
		}
		requireExit(t, "replayed "+name, actual[i], 0)
		if diff := sameEvidence(recovered[i], actual[i]); diff != "" {
			t.Fatalf("%s full cluster replay differs in %s", name, diff)
		}
		if diff := sameStorageEvidence(recovered[i], actual[i]); diff != "" {
			t.Fatalf("%s full cluster replay storage differs in %s", name, diff)
		}
		temporalHistoryOracles(t, evidence, "temporal.replication."+name+"-replayed", temporalRecoveryValue(t, recovered[i].Stdout), temporalRecoveryValue(t, actual[i].Stdout))
	}
	finished := pair.finish(t, evidence, "two-cluster-replayed", observers...)
	if !bytes.Equal(expected, finished.World) {
		t.Fatal("forced independent cluster replay changed shared World delivery order")
	}
	wantNetwork, err := json.Marshal(expectedNetwork)
	if err != nil {
		t.Fatal(err)
	}
	actualNetwork, err := json.Marshal(finished.Network)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(wantNetwork, actualNetwork) {
		t.Fatal("forced independent cluster replay changed shared network Record")
	}
	temporalWrite(t, filepath.Join(evidence, "two-cluster-replayed-world.json"), finished.World)
	summary, err := world.DecodeCompressedSnapshot(finished.World, world.CompressedSnapshotProfile())
	if err != nil {
		t.Fatal(err)
	}
	temporalJSON(t, filepath.Join(evidence, "two-cluster-replayed-world-cardinality.json"), summary)
	temporalJSON(t, filepath.Join(evidence, "two-cluster-replayed-network-record.json"), finished.Network)
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

func diagnosticRunRoot(base, prefix string) (string, error) {
	if err := os.MkdirAll(base, 0700); err != nil {
		return "", err
	}
	return os.MkdirTemp(base, prefix)
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

type terminalExitHelperAdmission struct {
	BuildReceipt, BuildReceiptSHA256   string
	CargoArtifact, CargoArtifactSHA256 string
	Helper, HelperSHA256               string
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

func mustTerminalExitRead(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return data
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
	Cluster   func(*testing.T, string, *ClusterResult)
}

func temporalIndependentClustersPartitionHealWholeClusterCrash(t *testing.T, request Request, evidence string, observers ...temporalPairObserver) {
	t.Helper()
	inProcess := request.diagnosticConfigurationRevision == "" && request.guestProfile == nil
	request.Config.Args = []string{"replication.test", "-test.v", "-test.count=1", "-test.timeout=1h", "-test.run=^TestIndependentClusterReplicationRecovery$", "-persistenceType=sql", "-persistenceDriver=sqlite"}
	network := &NetworkModelV2{Profile: NetworkStreamProfile, Limits: netmodel.Limits{Listeners: 128, Connections: 4096, Bytes: 128 << 20, ConnectionBytes: 4 << 20, StallNanos: 10_000_000_000, Transitions: 1 << 20}}
	request.Config.NetworkV2 = network
	brokerConfig := temporalBrokerConfiguration(network)
	brokerConfig.Seed = request.Runtime.Seed
	temporalJSON(t, filepath.Join(evidence, "two-cluster-broker-config.json"), brokerConfig)
	b, err := NewBroker(brokerConfig)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if inProcess {
			return
		}
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
		if inProcess {
			return
		}
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
		requests[i].ReturnStorageImage = true
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
	var pair *temporalClusterPair
	if inProcess {
		pair = temporalOpenClusterPair(t, brokerConfig, requests, temporalReopenRequests(requests))
	}
	runPair := func(label string, requests []Request, incarnation uint64, warm bool) ([]Result, []error) {
		if pair != nil {
			return pair.run(t, evidence, label, requests, incarnation, observers...)
		}
		return temporalRunPair(t, evidence, label, requests, warm, observers...)
	}
	results, errors := runPair("cluster-crash", requests, 1, true)
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
		if inProcess {
			requests[i].Broker = &BrokerSession{Node: BrokerNode{ID: name, Incarnation: 2}}
		} else {
			requests[i].Broker, err = b.Restart(name)
			if err != nil {
				t.Fatal(err)
			}
		}
		temporalRetainRequest(t, evidence, name+"-cluster-reopen", requests[i])
	}
	if identities[0] == identities[1] {
		t.Fatal("independent Temporal clusters have the same identity")
	}
	convergence, err := gomadsim.EventualConvergence("temporal.replication.healed", converged, 1<<20)
	if err != nil || !convergence.Passed {
		t.Fatalf("detached Temporal convergence oracle: passed=%v err=%v", convergence.Passed, err)
	}
	temporalJSON(t, filepath.Join(evidence, "temporal-replication-healed-oracle.json"), convergence)
	temporalJSON(t, filepath.Join(evidence, "temporal-replication-healed-oracle-inputs.json"), converged)
	results, errors = runPair("cluster-reopened", requests, 2, false)
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
	var encodedWorld []byte
	var finalNetwork netmodel.Record
	networkOwner := "go-broker-reference"
	networkOwnerImplementation := netmodel.ImplementationSHA256()
	if inProcess {
		finished := pair.finish(t, evidence, "two-cluster", observers...)
		encodedWorld = finished.World
		finalNetwork = finished.Network
		networkOwner = "rust-cluster"
		networkOwnerImplementation = request.HelperSHA256
		temporalJSON(t, filepath.Join(evidence, "two-cluster-network.json"), finished.Network.Snapshot)
		temporalJSON(t, filepath.Join(evidence, "two-cluster-network-record.json"), finished.Network)
	} else {
		encodedWorld, err = b.EncodeWorldEvidence()
		if err != nil {
			t.Fatal(err)
		}
		finalNetwork, err = b.NetworkRecord()
		if err != nil {
			t.Fatal(err)
		}
	}
	temporalWrite(t, filepath.Join(evidence, "two-cluster-world.json"), encodedWorld)
	if inProcess {
		summary, err := world.DecodeCompressedSnapshot(encodedWorld, world.CompressedSnapshotProfile())
		if err != nil {
			t.Fatal(err)
		}
		temporalJSON(t, filepath.Join(evidence, "two-cluster-world-cardinality.json"), summary)
	} else {
		temporalJSON(t, filepath.Join(evidence, "two-cluster-world-cardinality.json"), b.DiagnosticCardinality())
	}
	temporalJSON(t, filepath.Join(evidence, "two-cluster-owner.json"), map[string]any{"network_owner": networkOwner, "network_owner_implementation_sha256": networkOwnerImplementation, "helper_sha256": request.HelperSHA256, "seed": request.Runtime.Seed})
	temporalReplayClusters(t, evidence, brokerConfig, commitRequests, crashed, results, encodedWorld, finalNetwork, observers...)
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
		if frame.Reason == "compiler-artifact" && frame.Target.Name == "gomad-wasmhost" && frame.Executable != "" {
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
	if build.Crate != crate || build.Helper != request.HelperPath || build.HelperSHA256 != request.HelperSHA256 || build.Features == nil || !slices.Equal(build.Features, features) || !slices.Contains(build.Command, "--release") || !slices.Contains(build.Command, "--locked") || !slices.Contains(build.Command, "--no-default-features") || !strings.HasPrefix(build.RustcVersion, "rustc 1.99.1 ") || !strings.HasPrefix(build.CargoVersion, "cargo 1.99.1 ") {
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

// sameStorageEvidence compares the storage evidence of the Go-owner reference
// and the helper-owned volume, including the recovery image identity: both
// owners encode the same self-contained image v1 bytes.
func sameStorageEvidence(reference, local Result) string {
	if diff := sameEvidence(reference, local); diff != "" {
		return diff
	}
	switch {
	case reference.Message != local.Message:
		return "message"
	case reference.EnvironmentIdentity != local.EnvironmentIdentity:
		return "environment"
	case reference.Calls != local.Calls || reference.TimeAdvances != local.TimeAdvances:
		return "accounting"
	case !bytes.Equal(reference.StorageTranscript, local.StorageTranscript):
		return "storage transcript"
	case !reflect.DeepEqual(reference.StorageSnapshots, local.StorageSnapshots):
		return "snapshots"
	case !reflect.DeepEqual(reference.StorageOperations, local.StorageOperations):
		return "operations"
	case !reflect.DeepEqual(reference.StorageCrashEnumeration, local.StorageCrashEnumeration):
		return "enumerations"
	case !reflect.DeepEqual(reference.StorageCrashContext, local.StorageCrashContext):
		return "crash context"
	case reference.StorageImage.Identity() != local.StorageImage.Identity():
		return "image"
	case reference.RuntimeInterrupted != local.RuntimeInterrupted || !bytes.Equal(reference.InterruptedChoiceBytes, local.InterruptedChoiceBytes):
		return "interrupted choices"
	}
	return ""
}
