package measure

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	errutil "errors"
	"fmt"
	"io"
	"io/fs"
	"maps"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"regexp"
	"runtime"
	"runtime/pprof"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	runnerbackend "go.temporal.io/server/tools/gomad/internal/backends"
	"go.temporal.io/server/tools/gomad/internal/backends/target"
	"go.temporal.io/server/tools/gomad/internal/backends/wasm/internal/measurement"
	"go.temporal.io/server/tools/gomad/internal/backends/wasm/internal/stockgo"
	"go.temporal.io/server/tools/gomad/internal/backends/wasm/internal/temporalinput"
	"go.temporal.io/server/tools/gomad/internal/backends/wasm/toolchain"
	wasi "go.temporal.io/server/tools/gomad/internal/backends/wasm/wasi"
	captured "go.temporal.io/server/tools/gomad/internal/evidence/artifact/capture"
	hostexec "go.temporal.io/server/tools/gomad/internal/host/exec"
	hostfs "go.temporal.io/server/tools/gomad/internal/host/fs"
	"go.temporal.io/server/tools/gomad/internal/model/choice"
	"go.temporal.io/server/tools/gomad/internal/model/clock"
	gomadsim "go.temporal.io/server/tools/gomad/internal/model/cluster"
	netmodel "go.temporal.io/server/tools/gomad/internal/model/network"
	"go.temporal.io/server/tools/gomad/internal/model/volume"
	"go.temporal.io/server/tools/gomad/internal/model/world"
)

func choiceReplayRequest(original *runnerbackend.ChoiceRequest, mode choice.Mode, source []byte, plan choice.ReplayPlan) runnerbackend.ChoiceRequest {
	return runnerbackend.ChoiceRequest{Mode: mode, ExecutionIdentity: original.ExecutionIdentity, Limit: original.Limit,
		Trace: source, TraceSHA256: sha256.Sum256(source), TapeSHA256: plan.SHA256, Decisions: uint64(len(plan.Decisions))}
}

func namespaceReplayRequest(path string, original *runnerbackend.ChoiceRequest, plan choice.ReplayPlan) (runnerbackend.ChoiceRequest, error) {
	source, err := hostfs.ReadBounded(path+".trace", original.Limit)
	if err != nil {
		return runnerbackend.ChoiceRequest{}, err
	}
	if sha256.Sum256(source) != plan.SourceTraceSHA256 {
		return runnerbackend.ChoiceRequest{}, errutil.New("namespace replay source differs from retained tape")
	}
	return choiceReplayRequest(original, choice.ModeReplay, source, plan), nil
}

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
	Telemetry                         *wasi.Telemetry
	HelperResourceUsage               *hostexec.ResourceUsage
}
type prototypeReport struct {
	mu     sync.Mutex
	path   string
	Guests map[string]map[string]any
	Runs   []prototypeRun
}

func prototypeRoot(runCtx context.Context) (measureValue0 string, measureErr error) {
	root := input(runCtx, "GOMAD_WASM_PROTOTYPE_ROOT")
	if root == "" {
		measureErr = fmt.Errorf("%w: %v", ErrRefused, errutil.New("set GOMAD_WASM_PROTOTYPE_ROOT to run the determinism prototype gates"))
		return
	}
	if !filepath.IsAbs(root) {
		measureErr = errutil.New("GOMAD_WASM_PROTOTYPE_ROOT must be absolute")
		return
	}
	err := os.MkdirAll(root, 0700)
	if err != nil {
		measureErr = err
		return
	}
	return root, measureErr
}
func prototypeReportFor(runCtx context.Context, name string) (measureValue0 *prototypeReport, measureErr error) {
	prototypeRoot2, prototypeRootErr := prototypeRoot(runCtx)
	if prototypeRootErr != nil {
		measureErr = prototypeRootErr
		return
	}
	path := filepath.Join(prototypeRoot2, name+"-report.json")
	value, _ := scopeFor(runCtx).reports.LoadOrStore(path, &prototypeReport{path: path, Guests: map[string]map[string]any{}})
	return value.(*prototypeReport), measureErr
}
func (r *prototypeReport) guest(name string, fields map[string]any) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.Guests[name] = fields
	return r.flush()
}
func (r *prototypeReport) add(run prototypeRun) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.Runs = append(r.Runs, run)
	return r.flush()
}
func (r *prototypeReport) flush() error {
	data, err := json.MarshalIndent(r, "", "  ")
	if err == nil {
		err = os.WriteFile(r.path, data, 0600)
	}
	return err
}
func prototypeToolchain(runCtx context.Context) (compiler, overlay, identity string, measureErr error) {
	compiler = input(runCtx, "GOMAD_STOCK_GO")
	if !filepath.IsAbs(compiler) {
		measureErr = errutil.New("prototype gates require an absolute GOMAD_STOCK_GO (Go 1.27.1)")
		return
	}
	root, prototypeRootErr := prototypeRoot(runCtx)
	if prototypeRootErr != nil {
		measureErr = prototypeRootErr
		return
	}
	goroot, err := stockgo.SharedWithCache(input(runCtx, "GOMAD_STOCK_GO"), input(runCtx, "GOMAD_WASM_TOOLCHAIN_CACHE"))
	if err != nil {
		return "", "", "", err
	}
	scopeFor(runCtx).overlay.once.Do(func() {
		native, err := absolute(runCtx, "../../../model/runtimesource")
		if err != nil {
			scopeFor(runCtx).overlay.err = err
			return
		}
		scopeFor(runCtx).overlay.path, scopeFor(runCtx).overlay.identity, scopeFor(runCtx).overlay.err = toolchain.BuildOverlay(goroot, native, filepath.Join(root, "overlay"))
	})
	if scopeFor(runCtx).overlay.err != nil {
		measureErr = scopeFor(runCtx).overlay.err
		return
	}
	return filepath.Join(goroot, "bin", "go"), scopeFor(runCtx).overlay.path, scopeFor(runCtx).overlay.identity, measureErr
}
func prototypeBuild(runCtx context.Context, dir, command, target, output string, tags ...string) (module string, seconds float64, measureErr error) {
	prototypeToolchain2, prototypeToolchain22, prototypeToolchain3, prototypeToolchainErr := prototypeToolchain(runCtx)
	if prototypeToolchainErr != nil {
		measureErr = prototypeToolchainErr
		return
	}
	compiler, overlay, _ := prototypeToolchain2, prototypeToolchain22, prototypeToolchain3
	args := []string{command}
	if command == "test" {
		args = append(args, "-c")
	}
	args = append(args, "-trimpath", "-overlay", overlay, "-o", output)
	if len(tags) > 0 {
		args = append(args, "-tags", strings.Join(tags, ","))
	}
	args = append(args, target)
	build := exec.CommandContext(runCtx, compiler, args...)
	build.Dir = dir
	build.Env = append(environ(runCtx), "GOROOT="+filepath.Dir(filepath.Dir(compiler)), "GOOS=wasip1", "GOARCH=wasm", "GOEXPERIMENT=nogreenteagc", "GOWORK=off", "GOENV=off", "GOFLAGS=-mod=readonly", "GOTOOLCHAIN=local", "CGO_ENABLED=0")
	started := time.Now()
	out, err := build.CombinedOutput()
	if err != nil {
		measureErr = fmt.Errorf("build cooperative guest %s: %v\n%s", target, err, out)
		return
	}
	return output, time.Since(started).Seconds(), measureErr
}
func prototypeRequest(runCtx context.Context, module string, config wasi.Config, seed uint64, timeout time.Duration) (measureValue0 wasi.Request, measureErr error) {
	prototypeToolchain2, prototypeToolchain22, prototypeToolchain3, prototypeToolchainErr := prototypeToolchain(runCtx)
	if prototypeToolchainErr != nil {
		measureErr = prototypeToolchainErr
		return
	}
	_, _, identity := prototypeToolchain2, prototypeToolchain22, prototypeToolchain3
	data, err := os.ReadFile(module)
	if err != nil {
		measureErr = err
		return
	}
	imports, pages, binaryInventoryErr := binaryInventory(runCtx, data)
	if binaryInventoryErr != nil {
		measureErr = binaryInventoryErr
		return
	}
	helper, helperErr := helperPath(runCtx)
	if helperErr != nil {
		measureErr = helperErr
		return
	}
	config.Profile = wasi.CooperativeProfile
	config.Clock.ReadStepNanos = 0
	execution := choice.ExecutionIdentity{TargetSHA256: sha256.Sum256(data), ToolchainBuildKey: identity, GOOS: "wasip1", GOARCH: "wasm", ImplementationSHA256: sha256.Sum256([]byte(identity))}
	prototypeRoot2, prototypeRootErr := prototypeRoot(runCtx)
	if prototypeRootErr != nil {
		measureErr = prototypeRootErr
		return
	}
	helperSHA256, helperSHA256Err := fileHash(runCtx, helper)
	if helperSHA256Err != nil {
		measureErr = helperSHA256Err
		return
	}
	moduleSHA256, moduleSHA256Err := fileHash(runCtx, module)
	if moduleSHA256Err != nil {
		measureErr = moduleSHA256Err
		return
	}
	return wasi.Request{CompileCache: filepath.Join(prototypeRoot2, "compile-cache"), DigestTranscript: true, HelperPath: helper, HelperSHA256: helperSHA256, ModulePath: module, ModuleSHA256: moduleSHA256, Engine: wasi.PinnedEngine(), Imports: imports, InitialMemoryPages: pages, MemoryBytes: 4 << 30, Fuel: 1 << 50, Config: config, Timeout: timeout, Runtime: &wasi.RuntimeControl{Seed: seed, Choice: &runnerbackend.ChoiceRequest{Mode: choice.ModeRecord, ExecutionIdentity: execution, Limit: choice.MaximumTraceBytes}}}, measureErr
}
func digest(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}
func prototypeLimit(runCtx context.Context, name string, fallback int) int {
	if value, err := strconv.Atoi(input(runCtx, name)); err == nil && value > 0 {
		return value
	}
	return fallback
}
func prototypeRunOne(runCtx context.Context, report *prototypeReport, guest, variant string, instance int, request wasi.Request) (measureValue0 wasi.Result, measureErr error) {
	pool := scopeFor(runCtx).light
	info, err := os.Stat(request.ModulePath)
	if err == nil && info.Size() > 64<<20 {
		pool = scopeFor(runCtx).heavy
	}
	select {
	case pool <- struct{}{}:
	case <-runCtx.Done():
		return wasi.Result{}, runCtx.Err()
	}
	defer func() {
		<-pool
	}()
	name := fmt.Sprintf("%s-%s-seed%d-%d", guest, variant, request.Runtime.Seed, instance)
	prefix := input(runCtx, "GOMAD_WASM_PROTOTYPE_MIRROR")
	if prefix != "" {
		mirror, err := os.Create(prefix + "-" + name + ".stdout")
		if err != nil {
			return wasi.Result{}, err
		}
		defer func() {
			closeResource(runCtx, mirror)
		}()
		request.StdoutMirror = mirror
	}
	started := time.Now()
	result, err := wasi.Run(runCtx, request)
	wall := time.Since(started)
	if err != nil {
		return wasi.Result{}, fmt.Errorf("%s: %w", name, err)
	}
	run := prototypeRun{Guest: guest, Variant: variant, Seed: request.Runtime.Seed, Instance: instance, Mode: fmt.Sprint(request.Runtime.Choice.Mode), Termination: result.Termination, Message: result.Message, Calls: int(result.Calls), TimeAdvances: result.TimeAdvances, ExitCode: result.ExitCode, WallNanos: wall.Nanoseconds(), PeakMemoryBytes: result.PeakMemoryBytes, FuelUsed: request.Fuel - result.FuelRemaining, ChoiceRecords: int(result.ChoiceTrace.Summary.Records), ChoiceSHA256: hex.EncodeToString(result.ChoiceTrace.SHA256[:]), StdoutSHA256: digest(result.Stdout), WASISHA256: hex.EncodeToString(result.TranscriptSHA256[:]), NetworkSHA256: digest(result.NetworkTranscript), StdoutBytes: len(result.Stdout), NetworkEvents: bytes.Count(result.NetworkTranscript, []byte{'\n'})}
	run.AuthorizationToTerminalNanos, run.Telemetry, run.HelperResourceUsage = result.AuthorizationToTerminalNanos, result.Telemetry, result.HelperResourceUsage
	if result.Termination == "exit" {
		run.VirtualNanos = uint64(int64FromTerminal(result.RuntimeTerminal()))
	}
	if err := report.add(run); err != nil {
		measureErr = errutil.Join(measureErr, err)
		return
	}
	if instance == 0 || result.Termination != "exit" || result.ExitCode == nil || *result.ExitCode != 0 {
		prefix := filepath.Join(filepath.Dir(report.path), name)
		for suffix, data := range map[string][]byte{".stdout": result.Stdout, ".stderr": result.Stderr, ".network.jsonl": result.NetworkTranscript} {
			err := os.WriteFile(prefix+suffix, data, 0600)
			if err != nil {
				return wasi.Result{}, err
			}
		}
	}
	return result, measureErr
}
func prototypeExecute(runCtx context.Context, report *prototypeReport, guest, variant string, instance int, request wasi.Request) (measureValue0 wasi.Result, measureErr error) {
	result, err := prototypeRunOne(runCtx, report, guest, variant, instance, request)
	if err != nil {
		measureErr = err
		return
	}
	return result, measureErr
}
func int64FromTerminal(terminal [16]byte) int64 {
	var value int64
	for _, b := range terminal[8:] {
		value = value<<8 | int64(b)
	}
	return value
}
func sameEvidence(a, b wasi.Result) string {
	for _, check := range []struct {
		name string
		x, y []byte
	}{{"stdout", a.Stdout, b.Stdout}, {"stderr", a.Stderr, b.Stderr}, {"wasi", a.TranscriptSHA256[:], b.TranscriptSHA256[:]}, {"choices", a.ChoiceTrace.Bytes, b.ChoiceTrace.Bytes}, {"network", a.NetworkTranscript, b.NetworkTranscript}, {"terminal", runtimeTerminalBytes(a)[:], runtimeTerminalBytes(b)[:]}} {
		if !bytes.Equal(check.x, check.y) {
			return check.name
		}
	}
	if a.Termination != b.Termination || (a.ExitCode == nil) != (b.ExitCode == nil) || a.ExitCode != nil && *a.ExitCode != *b.ExitCode {
		return "termination"
	}
	return ""
}
func requireExit(runCtx context.Context, label string, result wasi.Result, codes ...uint32) (measureErr error) {
	if result.Termination != "exit" || result.ExitCode == nil || !slices.Contains(codes, *result.ExitCode) || !result.Reaped {
		code := "none"
		if result.ExitCode != nil {
			code = fmt.Sprint(*result.ExitCode)
		}
		tail := result.Stdout
		if len(tail) > 4000 {
			tail = tail[len(tail)-4000:]
		}
		measureErr = fmt.Errorf("%s: termination=%s exit=%s message=%s\nstdout tail:\n%s\nstderr:\n%s", label, result.Termination, code, result.Message, tail, result.Stderr)
		return
	}
	return
}
func distinctChoiceTraces(baselines map[uint64]wasi.Result) int {
	seen := map[[32]byte]bool{}
	for _, result := range baselines {
		seen[result.ChoiceTrace.SHA256] = true
	}
	return len(seen)
}
func prototypeFixtureConfig() wasi.Config {
	config := standaloneConfig()
	config.Limits.Calls = 1000000
	config.Limits.TranscriptBytes = 64 << 20
	return config
}
func prototypeTemporalConfig(runCtx context.Context, root, cwd string, args ...string) (measureValue0 wasi.Config, measureErr error) {
	inputs, err := temporalinput.CaptureSchema(root)
	if err != nil {
		measureErr = err
		return
	}
	return wasi.TemporalConfig(inputs, cwd, wasi.TemporalPrototype, args...)
}
func cooperativeRequest(runCtx context.Context, fixture string) (measureValue0 wasi.Request, measureErr error) {
	if input(runCtx, "GOMAD_STOCK_GO") == "" {
		measureErr = errutil.New("qualified guest gate requires explicit GOMAD_STOCK_GO")
		return
	}
	root, err := stockgo.SharedWithCache(input(runCtx, "GOMAD_STOCK_GO"), input(runCtx, "GOMAD_WASM_TOOLCHAIN_CACHE"))
	if err != nil {
		return wasi.Request{}, err
	}
	native, err := absolute(runCtx, "../../../model/runtimesource")
	if err != nil {
		measureErr = err
		return
	}
	temporaryDirectory2, temporaryDirectoryErr := temporaryDirectory(runCtx)
	if temporaryDirectoryErr != nil {
		measureErr = temporaryDirectoryErr
		return
	}
	overlay, identity, err := toolchain.BuildOverlay(root, native, temporaryDirectory2)
	if err != nil {
		measureErr = err
		return
	}
	temporaryDirectory3, temporaryDirectoryErr2 := temporaryDirectory(runCtx)
	if temporaryDirectoryErr2 != nil {
		measureErr = temporaryDirectoryErr2
		return
	}
	module := filepath.Join(temporaryDirectory3, "runtime.wasm")
	command := exec.CommandContext(runCtx, filepath.Join(root, "bin/go"), "build", "-p=1", "-trimpath", "-buildvcs=false", "-overlay", overlay, "-o", module, fixture)
	command.Dir = filepath.Join(scopeFor(runCtx).options.Repo, "tools/gomad/internal/backends/wasm/wasi")
	command.Env = append(environ(runCtx), "GOROOT="+root, "GOOS=wasip1", "GOARCH=wasm", "GOEXPERIMENT=nogreenteagc", "GOWORK=off", "GOENV=off", "GOFLAGS=", "GOTOOLCHAIN=local", "CGO_ENABLED=0", "GOMAXPROCS=1")
	output, err := command.CombinedOutput()
	if err != nil {
		measureErr = fmt.Errorf("build actual cooperative guest: %v\n%s", err, output)
		return
	}
	cooperativeModuleRequest2, cooperativeModuleRequestErr := cooperativeModuleRequest(runCtx, module, identity)
	if cooperativeModuleRequestErr != nil {
		measureErr = cooperativeModuleRequestErr
		return
	}
	return cooperativeModuleRequest2, measureErr
}
func requireSplitServedOnlyOutput(runCtx context.Context, label string, result wasi.Result) (measureErr error) {
	writes := uint64(0)
	for op, count := range result.GoFilesystemCalls {
		if op != "fd_write" {
			measureErr = fmt.Errorf("%s served %s in Go: %v", label, op, result.GoFilesystemCalls)
			return
		}
		writes += count
	}
	records, err := wasi.DecodeTranscript(result.Transcript)
	if err != nil {
		measureErr = err
		return
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
	digestOnly := len(result.Transcript) == 0 && result.TranscriptSHA256 != [32]byte{}
	if result.GoFilesystemCalls == nil || !digestOnly && writes != output {
		measureErr = fmt.Errorf("%s Go fs-served %v, want only its %d fd 1/2 writes", label, result.GoFilesystemCalls, output)
		return
	}
	return
}

const temporalObservationPrefix = "GOMAD_TEMPORAL_OBSERVATION "

type temporalRecoveryObservation struct {
	ClusterID string   `json:"cluster_id"`
	History   []byte   `json:"history"`
	EventIDs  []int64  `json:"event_ids"`
	Types     []string `json:"types"`
}

func temporalRecoveryValue(runCtx context.Context, stdout []byte) (measureValue0 temporalRecoveryObservation, measureErr error) {
	var value temporalRecoveryObservation
	count := 0
	for _, line := range bytes.Split(stdout, []byte{'\n'}) {
		if !bytes.HasPrefix(line, []byte(temporalObservationPrefix)) {
			continue
		}
		err := json.Unmarshal(line[len(temporalObservationPrefix):], &value)
		if err != nil {
			measureErr = err
			return
		}
		count++
	}
	if count != 1 || value.ClusterID == "" || len(value.History) == 0 || len(value.EventIDs) != len(value.Types) {
		measureErr = fmt.Errorf("invalid detached Temporal history observation: count=%d value=%+v", count, value)
		return
	}
	for i, id := range value.EventIDs {
		if id != int64(i+1) {
			measureErr = fmt.Errorf("duplicate or lost history event: %v", value.EventIDs)
			return
		}
	}
	return value, measureErr
}
func temporalHistoryOracles(runCtx context.Context, evidence, name string, expected, actual temporalRecoveryObservation) (measureErr error) {
	history := func(value temporalRecoveryObservation) []gomadsim.HistoryOperation {
		return []gomadsim.HistoryOperation{{ID: "committed", Actor: "workflow", Kind: "workflow-history", Invocation: 1, Completion: 2, Output: value.History}}
	}
	exact, err := gomadsim.ExactHistory(name+".history", history(expected), history(actual), 1<<20)
	if err != nil || !exact.Passed {
		measureErr = fmt.Errorf("detached Temporal history oracle: passed=%v err=%v", exact.Passed, err)
		return
	}
	ids := func(value temporalRecoveryObservation) []string {
		values := make([]string, len(value.EventIDs))
		for i, id := range value.EventIDs {
			values[i] = fmt.Sprint(id)
		}
		return values
	}
	writeReportErr := temporalJSON(runCtx, filepath.Join(evidence, name+"-oracle-inputs.json"), struct {
		ExpectedHistory, ActualHistory []gomadsim.HistoryOperation
		ExpectedEvents, ActualEvents   []string
		ExpectedCluster, ActualCluster string
	}{history(expected), history(actual), ids(expected), ids(actual), expected.ClusterID, actual.ClusterID})
	if writeReportErr != nil {
		measureErr = writeReportErr
		return
	}
	delivery, err := gomadsim.NoDuplicateOrLost(name+".events", ids(expected), ids(actual), 1<<20)
	if err != nil || !delivery.Passed {
		measureErr = fmt.Errorf("detached Temporal duplicate/lost oracle: passed=%v err=%v", delivery.Passed, err)
		return
	}
	if expected.ClusterID != actual.ClusterID {
		measureErr = errutil.New("recovered Temporal cluster identity changed")
		return
	}
	writeReportErr2 := temporalJSON(runCtx, filepath.Join(evidence, name+"-oracles.json"), []gomadsim.OracleResult{exact, delivery})
	if writeReportErr2 != nil {
		measureErr = writeReportErr2
		return
	}
	return
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
func temporalRetainRecoveryRun(runCtx context.Context, evidence, label string, result wasi.Result) (measureErr error) {
	if result.StorageImage != nil {
		logf(runCtx, "storage recovery image %s: bytes=%d", label, result.StorageImage.Bytes())
	}
	if result.Termination != "" {
		requireSplitServedOnlyOutputErr := requireSplitServedOnlyOutput(runCtx, label, result)
		if requireSplitServedOnlyOutputErr != nil {
			measureErr = requireSplitServedOnlyOutputErr
			return
		}
	}
	retainRecoveryEvidenceErr := temporalRetainRecoveryEvidence(runCtx, evidence, label, result)
	if retainRecoveryEvidenceErr != nil {
		measureErr = retainRecoveryEvidenceErr
		return
	}
	return
}
func temporalRetainClusterRun(runCtx context.Context, evidence, label string, result wasi.Result, guestCallbacks uint64) (measureErr error) {
	if result.StorageImage != nil {
		logf(runCtx, "storage recovery image %s: bytes=%d", label, result.StorageImage.Bytes())
	}
	err := clusterServedOnlyByHelper(result, guestCallbacks)
	if err != nil {
		measureErr = fmt.Errorf("%s %v", label, err)
		return
	}
	retainRecoveryEvidenceErr := temporalRetainRecoveryEvidence(runCtx, evidence, label, result)
	if retainRecoveryEvidenceErr != nil {
		measureErr = retainRecoveryEvidenceErr
		return
	}
	return
}
func clusterServedOnlyByHelper(result wasi.Result, guestCallbacks uint64) error {
	if result.GoFilesystemCalls != nil {
		return fmt.Errorf("in-process cluster incarnation has a Go filesystem: %v", result.GoFilesystemCalls)
	}
	if guestCallbacks != 0 {
		return fmt.Errorf("in-process cluster relayed %d guest callbacks to Go", guestCallbacks)
	}
	return nil
}
func temporalRetainRecoveryEvidence(runCtx context.Context, evidence, label string, result wasi.Result) (measureErr error) {
	writeEvidenceErr := temporalWrite(runCtx, filepath.Join(evidence, label+".stdout"), result.Stdout)
	if writeEvidenceErr != nil {
		measureErr = writeEvidenceErr
		return
	}
	writeEvidenceErr2 := temporalWrite(runCtx, filepath.Join(evidence, label+".stderr"), result.Stderr)
	if writeEvidenceErr2 != nil {
		measureErr = writeEvidenceErr2
		return
	}
	writeEvidenceErr3 := temporalWrite(runCtx, filepath.Join(evidence, label+".storage.jsonl"), result.StorageTranscript)
	if writeEvidenceErr3 != nil {
		measureErr = writeEvidenceErr3
		return
	}
	writeEvidenceErr4 := temporalWrite(runCtx, filepath.Join(evidence, label+".network.json"), result.NetworkTranscript)
	if writeEvidenceErr4 != nil {
		measureErr = writeEvidenceErr4
		return
	}
	choices, diagnostics := result.ChoiceTrace.Bytes, result.DiagnosticTrace.Bytes
	if result.RuntimeInterrupted {
		choices, diagnostics = result.InterruptedChoiceBytes, result.InterruptedDiagnosticBytes
	}
	writeEvidenceErr5 := temporalWrite(runCtx, filepath.Join(evidence, label+".choices"), choices)
	if writeEvidenceErr5 != nil {
		measureErr = writeEvidenceErr5
		return
	}
	writeEvidenceErr6 := temporalWrite(runCtx, filepath.Join(evidence, label+".diagnostics"), diagnostics)
	if writeEvidenceErr6 != nil {
		measureErr = writeEvidenceErr6
		return
	}
	writeReportErr := temporalJSON(runCtx, filepath.Join(evidence, label+".snapshots.json"), result.StorageSnapshots)
	if writeReportErr != nil {
		measureErr = writeReportErr
		return
	}
	writeReportErr2 := temporalJSON(runCtx, filepath.Join(evidence, label+".operations.json"), result.StorageOperations)
	if writeReportErr2 != nil {
		measureErr = writeReportErr2
		return
	}
	if result.StorageCrashEnumeration != nil {
		writeReportErr3 := temporalJSON(runCtx, filepath.Join(evidence, label+".crash-states.json"), result.StorageCrashEnumeration)
		if writeReportErr3 != nil {
			measureErr = writeReportErr3
			return
		}
	}
	if result.Telemetry != nil {
		writeReportErr4 := temporalJSON(runCtx, filepath.Join(evidence, label+".telemetry.json"), result.Telemetry)
		if writeReportErr4 != nil {
			measureErr = writeReportErr4
			return
		}
	}
	if result.HelperResourceUsage != nil {
		writeReportErr5 := temporalJSON(runCtx, filepath.Join(evidence, label+".host-cost.json"), result.HelperResourceUsage)
		if writeReportErr5 != nil {
			measureErr = writeReportErr5
			return
		}
	}
	writeReportErr6 := temporalJSON(runCtx, filepath.Join(evidence, label+".terminal.json"), struct {
		Termination, Message, EnvironmentIdentity string
		ExitCode                                  *uint32
		Reaped, Cancelled, WatchdogTimeout        bool
		RuntimeInterrupted                        bool
		Calls, TimeAdvances, VirtualTimeNanos     uint64
	}{result.Termination, result.Message, result.EnvironmentIdentity, result.ExitCode, result.Reaped, result.Cancelled, result.WatchdogTimeout, result.RuntimeInterrupted, result.Calls, result.TimeAdvances, result.VirtualTimeElapsedNanos})
	if writeReportErr6 != nil {
		measureErr = writeReportErr6
		return
	}
	return
}
func temporalRetainRequest(runCtx context.Context, evidence, label string, request wasi.Request) (measureErr error) {
	request.Broker, request.StdoutMirror = nil, nil
	writeReportErr := temporalJSON(runCtx, filepath.Join(evidence, label+".request.json"), request)
	if writeReportErr != nil {
		measureErr = writeReportErr
		return
	}
	return
}
func temporalVerifyInputs(runCtx context.Context, inputs map[string]string) (measureErr error) {
	for name, expected := range inputs {
		actual, nameSHA256Err := fileHash(runCtx, name)
		if nameSHA256Err != nil {
			measureErr = nameSHA256Err
			return
		}
		if actual != expected {
			measureErr = fmt.Errorf("source input changed during admission: %s: expected %s, got %s", name, expected, actual)
			return
		}
	}
	return
}
func temporalBuildEnvironment(runCtx context.Context, compiler string) (measureValue0 []string, measureErr error) {
	environment := []string{}
	for _, name := range []string{"HOME", "TMPDIR", "GOTMPDIR", "GOCACHE", "GOPATH", "GOMODCACHE"} {
		value, found := lookupInput(runCtx, name)
		if found {
			environment = append(environment, name+"="+value)
		}
	}
	return append(environment, "PATH="+filepath.Dir(compiler)+":"+input(runCtx, "PATH"), "GOOS=wasip1", "GOARCH=wasm", "CGO_ENABLED=0", "GOWORK=off", "GOENV=off", "GOFLAGS=", "GOEXPERIMENT=nogreenteagc", "GOTOOLCHAIN=local", "GOMAXPROCS=1"), measureErr
}
func temporalCommand(runCtx context.Context, evidence, name, root string, environment, command []string) (measureValue0 hostexec.Result, measureErr error) {
	started := time.Now()
	result, err := hostexec.Run(runCtx, hostexec.Request{Command: command, Dir: root, Env: environment, Timeout: 15 * time.Minute, TerminateGrace: time.Second, OutputLimit: 64 << 20, CollectResourceUsage: true})
	writeEvidenceErr := temporalWrite(runCtx, filepath.Join(evidence, name+".stdout"), result.Stdout.RawBytes)
	if writeEvidenceErr != nil {
		measureErr = writeEvidenceErr
		return
	}
	writeEvidenceErr2 := temporalWrite(runCtx, filepath.Join(evidence, name+".stderr"), result.Stderr.RawBytes)
	if writeEvidenceErr2 != nil {
		measureErr = writeEvidenceErr2
		return
	}
	writeReportErr := temporalJSON(runCtx, filepath.Join(evidence, name+".json"), struct {
		Command       []string
		ExitCode      int
		ElapsedNanos  int64
		ResourceUsage *hostexec.ResourceUsage
	}{command, result.ExitCode, time.Since(started).Nanoseconds(), result.ResourceUsage})
	if writeReportErr != nil {
		measureErr = writeReportErr
		return
	}
	if err != nil || result.Termination != hostexec.TerminationExit || result.ExitCode != 0 || !result.GroupGone {
		measureErr = fmt.Errorf("infrastructure command %v: %v, termination=%s exit=%d\n%s\n%s", command, err, result.Termination, result.ExitCode, result.Stdout.RawBytes, result.Stderr.RawBytes)
		return
	}
	return result, measureErr
}
func temporalSourceInputs(runCtx context.Context, data []byte) (measureValue0 map[string]string, measureErr error) {
	inputs := map[string]string{}
	decoder := json.NewDecoder(bytes.NewReader(data))
	for {
		var pkg struct {
			Dir                                                              string
			GoFiles, CgoFiles, SFiles, EmbedFiles, TestGoFiles, XTestGoFiles []string
		}
		err := decoder.Decode(&pkg)
		if err == io.EOF {
			break
		} else if err != nil {
			measureErr = err
			return
		}
		for _, names := range [][]string{pkg.GoFiles, pkg.CgoFiles, pkg.SFiles, pkg.EmbedFiles, pkg.TestGoFiles, pkg.XTestGoFiles} {
			for _, name := range names {
				if !filepath.IsAbs(name) {
					name = filepath.Join(pkg.Dir, name)
				}
				nameSHA256, nameSHA256Err := fileHash(runCtx, name)
				if nameSHA256Err != nil {
					measureErr = nameSHA256Err
					return
				}
				inputs[name] = nameSHA256
			}
		}
	}
	return inputs, measureErr
}
func temporalJSON(runCtx context.Context, name string, value any) (measureErr error) {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		measureErr = err
		return
	}
	writeEvidenceErr := temporalWrite(runCtx, name, append(data, '\n'))
	if writeEvidenceErr != nil {
		measureErr = writeEvidenceErr
		return
	}
	return
}
func temporalWrite(runCtx context.Context, name string, data []byte) (measureErr error) {
	err := os.WriteFile(name, data, 0600)
	if err != nil {
		measureErr = err
		return
	}
	return
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
				err := root.Mkdir(relative, 0700)
				if err != nil {
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
		err = errutil.Join(writeErr, file.Close())
		if err != nil {
			return err
		}
		err = root.Chmod(relative, entry.Mode)
		if err != nil {
			return err
		}
	}
	for _, entry := range slices.Backward(snapshot.Entries) {
		if entry.Kind == captured.KindDirectory {
			relative, err := filepath.Rel("/module", entry.Path)
			if err != nil {
				return err
			}
			err = root.Chmod(relative, entry.Mode)
			if err != nil {
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
		defer func() {
			captureErr = errutil.Join(captureErr, broker.Close())
		}()
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
			_, err := broker.Lookup(prefix + "/" + child.Name)
			if err != nil {
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
	defer func() {
		retErr = errutil.Join(retErr, anchor.Close())
	}()
	parent := anchor
	components := strings.Split(filepath.Clean(destination), string(filepath.Separator))
	for _, component := range components[:len(components)-1] {
		info, err := parent.Lstat(component)
		if os.IsNotExist(err) {
			err = parent.Mkdir(component, 0700)
			if err != nil {
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
		defer func() {
			retErr = errutil.Join(retErr, next.Close())
		}()
		parent = next
	}
	name := components[len(components)-1]
	info, err := parent.Lstat(name)
	created := os.IsNotExist(err)
	if created {
		err = parent.Mkdir(name, 0700)
		if err != nil {
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
		err = errutil.Join(writeErr, root.Close())
		if err != nil {
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
func temporalStageBrokerFixture(runCtx context.Context, root, staged string) (measureErr error) {
	destination := filepath.Join(root, staged)
	err := os.MkdirAll(destination, 0700)
	if err != nil {
		measureErr = err
		return
	}
	sources := map[string]string{"recovery_test.go": filepath.Join(root, "tools/gomad/internal/backends/wasm/testdata/temporal_storage/recovery_test.go"), "replication_test.go": filepath.Join(root, "tools/gomad/internal/backends/wasm/testdata/temporal_broker/replication_test.go")}
	entries, err := os.ReadDir(destination)
	if err != nil {
		measureErr = err
		return
	}
	for _, entry := range entries {
		if sources[entry.Name()] == "" || !entry.Type().IsRegular() {
			measureErr = fmt.Errorf("unexpected staged broker fixture %q", entry.Name())
			return
		}
	}
	for name, source := range sources {
		expected, err := os.ReadFile(source)
		if err != nil {
			measureErr = err
			return
		}
		path := filepath.Join(destination, name)
		actual, err := os.ReadFile(path)
		if os.IsNotExist(err) {
			err := os.WriteFile(path, expected, 0600)
			if err != nil {
				measureErr = err
				return
			}
		} else if err != nil || !bytes.Equal(actual, expected) {
			measureErr = fmt.Errorf("staged broker source body differs from %s", source)
			return
		}
	}
	return
}
func temporalBrokerConfiguration(network *wasi.NetworkModelV2) wasi.BrokerConfig {
	return wasi.BrokerConfig{Nodes: []wasi.BrokerNode{{ID: "active", Incarnation: 1}, {ID: "standby", Incarnation: 1}}, Seed: 7, Limits: world.CompressedSnapshotProfile().Limits, WorldProfile: world.CompressedSnapshotSchema, Network: network, Addresses: map[string]string{"active": "10.0.0.1", "standby": "10.0.0.2"}, Diagnostics: true, Faults: []wasi.BrokerNetworkFault{{Name: "temporal-active-to-standby-partition", Action: "partition", From: "active", To: "standby", AtNanos: 1}, {Name: "temporal-active-to-standby-heal", Action: "heal", From: "active", To: "standby", AtNanos: uint64(30 * time.Second)}}}
}

type temporalClusterPair struct {
	session *wasi.ClusterSession
	names   []string
}

func temporalReopenRequests(requests []wasi.Request) []wasi.Request {
	reopened := append([]wasi.Request(nil), requests...)
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
func temporalOpenClusterPair(runCtx context.Context, configuration wasi.BrokerConfig, requests, reopened []wasi.Request) (measureValue0 *temporalClusterPair, measureErr error) {
	pair := &temporalClusterPair{}
	cluster := wasi.ClusterRequest{Broker: configuration, Boots: map[string]wasi.Request{}, OutputCrashGroup: &wasi.ClusterCrashGroup{Marker: []byte(temporalObservationPrefix)}}
	for i, request := range requests {
		name := configuration.Nodes[i].ID
		pair.names = append(pair.names, name)
		request.Broker, request.StorageImage = nil, nil
		request.ReturnStorageImage = true
		request.StdoutMirror = &temporalGuestWarm{ctx: runCtx, node: name}
		cluster.Instances = append(cluster.Instances, wasi.ClusterInstance{ID: name, Request: request})
		cluster.OutputCrashGroup.Nodes = append(cluster.OutputCrashGroup.Nodes, name)
		recovery := reopened[i]
		recovery.Broker, recovery.StorageImage = nil, nil
		recovery.ReturnStorageImage = true
		recovery.StdoutMirror = &temporalGuestWarm{ctx: runCtx, node: name}
		cluster.Boots[name+"-reopen"] = recovery
	}
	var err error
	pair.session, err = wasi.OpenCluster(runCtx, cluster)
	if err != nil {
		measureErr = errutil.Join(measureErr, err)
		return
	}
	addCleanup(runCtx, func(runCtx context.Context) (measureErr error) {
		ctx, cancel := context.WithTimeout(context.WithoutCancel(runCtx), 5*time.Second)
		defer cancel()
		err := pair.session.Shutdown(ctx)
		if err != nil {
			logf(runCtx, "cluster cleanup: %v", err)
		}
		return
	})
	return pair, measureErr
}
func (pair *temporalClusterPair) run(runCtx context.Context, evidence, label string, requests []wasi.Request, incarnation uint64, observers ...temporalPairObserver) (measureValue0 []wasi.Result, measureValue1 []error, measureErr error) {
	observedPair, observedPair2, observedPairErr := temporalObservedPair(runCtx, evidence, label, requests, observers, func() ([]wasi.Result, []error) {
		callbackValue0, callbackValue1, callbackValue2 := (func(runCtx context.Context) (measureValue0 []wasi.Result, measureValue1 []error, measureErr error) {
			budget, err := temporalPairBudget(input(runCtx, "GOMAD_WASM_TEMPORAL_RECOVERY_DIAGNOSTIC_BUDGET"))
			if err != nil {
				measureErr = errutil.Join(measureErr, err)
				return
			}
			ctx, cancel := context.WithTimeout(runCtx, budget)
			defer cancel()
			started := time.Now()
			results, failures := make([]wasi.Result, len(requests)), make([]error, len(requests))
			for i, name := range pair.names {
				boot := name
				if incarnation > 1 {
					boot += "-reopen"
				}
				err := pair.session.Start(ctx, name, incarnation, boot, requests[i].Runtime)
				if err != nil {
					failures[i] = err
					return results, failures, measureErr
				}
			}
			for i, name := range pair.names {
				results[i], failures[i] = pair.session.Wait(ctx, name, incarnation)
				retainClusterRunErr := temporalRetainClusterRun(runCtx, evidence, name+"-"+label, results[i], pair.session.FrameCounts().GuestCallbacks)
				if retainClusterRunErr != nil {
					measureErr = errutil.Join(measureErr, retainClusterRunErr)
					return
				}
				logf(runCtx, "%s %s terminated in shared Rust helper: %s %s", name, label, results[i].Termination, results[i].Message)
			}
			writeReportErr := temporalJSON(runCtx, filepath.Join(evidence, label+"-host-budget.json"), struct {
				BudgetNanos, ElapsedNanos        int64
				Expired                          bool
				NetworkOwner                     string
				NetworkOwnerImplementationSHA256 string
			}{int64(budget), time.Since(started).Nanoseconds(), ctx.Err() == context.DeadlineExceeded, "rust-cluster", requests[0].HelperSHA256})
			if writeReportErr != nil {
				measureErr = errutil.Join(measureErr, writeReportErr)
				return
			}
			return results, failures, measureErr
		})(runCtx)
		recordFailure(runCtx, callbackValue2)
		return callbackValue0, callbackValue1
	})
	if observedPairErr != nil {
		measureErr = errutil.Join(measureErr, observedPairErr)
		return
	}
	return observedPair, observedPair2, measureErr
}
func (pair *temporalClusterPair) finish(runCtx context.Context, evidence, label string, observers ...temporalPairObserver) (measureValue0 wasi.ClusterResult, measureErr error) {
	result, err := pair.session.Finish(runCtx)
	if err != nil {
		measureErr = err
		return
	}
	writeReportErr := temporalJSON(runCtx, filepath.Join(evidence, label+"-cluster-process.json"), pair.session.CompletedResult())
	if writeReportErr != nil {
		measureErr = writeReportErr
		return
	}
	for _, observer := range observers {
		if observer.Cluster != nil {
			if err := observer.Cluster(runCtx, label, pair.session.CompletedResult()); err != nil {
				measureErr = err
				return
			}
		}
	}
	if result.Frames.GuestCallbacks != 0 {
		measureErr = fmt.Errorf("cluster made %d guest callbacks to Go", result.Frames.GuestCallbacks)
		return
	}
	return result, measureErr
}
func temporalRunPair(runCtx context.Context, evidence, label string, requests []wasi.Request, warmFirst bool, observers ...temporalPairObserver) (measureValue0 []wasi.Result, measureValue1 []error, measureErr error) {
	observedPair, observedPair2, observedPairErr := temporalObservedPair(runCtx, evidence, label, requests, observers, func() ([]wasi.Result, []error) {
		callbackValue0, callbackValue1, callbackValue2 := (func(runCtx context.Context) (measureValue0 []wasi.Result, measureValue1 []error, measureErr error) {
			runPairExecution, runPairExecution2, runPairExecutionErr := temporalRunPairExecution(runCtx, evidence, label, requests, warmFirst)
			if runPairExecutionErr != nil {
				measureErr = runPairExecutionErr
				return
			}
			return runPairExecution, runPairExecution2, measureErr
		})(runCtx)
		recordFailure(runCtx, callbackValue2)
		return callbackValue0, callbackValue1
	})
	if observedPairErr != nil {
		measureErr = observedPairErr
		return
	}
	return observedPair, observedPair2, measureErr
}
func temporalReplayClusters(runCtx context.Context, evidence string, configuration wasi.BrokerConfig, requests []wasi.Request, crashed, recovered []wasi.Result, expected []byte, expectedNetwork netmodel.Record, observers ...temporalPairObserver) (measureErr error) {
	if requests[0].DiagnosticConfigurationRevision() == "" && requests[0].GuestProfile() == nil {
		replayClustersInProcessErr := temporalReplayClustersInProcess(runCtx, evidence, configuration, requests, crashed, recovered, expected, expectedNetwork, observers...)
		if replayClustersInProcessErr != nil {
			measureErr = replayClustersInProcessErr
			return
		}
		return
	}
	b, err := wasi.NewBroker(configuration)
	if err != nil {
		measureErr = err
		return
	}
	names := []string{"active", "standby"}
	err = b.CrashAfterOutputs(names, []byte(temporalObservationPrefix))
	if err != nil {
		measureErr = err
		return
	}
	for i, name := range names {
		if !crashed[i].RuntimeInterrupted {
			measureErr = fmt.Errorf("%s crash lost its interrupted choice evidence", name)
			return
		}
		tape, err := choice.ProjectInterruptedReplayPlan(crashed[i].InterruptedChoiceBytes, requests[i].Runtime.Choice.Limit, requests[i].Runtime.Choice.ExecutionIdentity)
		if err != nil {
			measureErr = err
			return
		}
		control, choices := *requests[i].Runtime, *requests[i].Runtime.Choice
		choices = choiceReplayRequest(&choices, choice.ModePrefix, crashed[i].InterruptedChoiceBytes, tape)
		control.Choice, requests[i].Runtime = &choices, &control
		requests[i].Broker, requests[i].StdoutMirror = b.Session(name), nil
		requests[i].Config.Environment = append([]string(nil), requests[i].Config.Environment...)
	}
	actual, errors, runPairErr := temporalRunPair(runCtx, evidence, "cluster-prefix-replay", requests, true, observers...)
	if runPairErr != nil {
		measureErr = runPairErr
		return
	}
	for i, name := range names {
		if errors[i] != nil || actual[i].Termination != "crash" || !actual[i].Reaped || actual[i].StorageImage == nil || !bytes.Equal(crashed[i].Stdout, actual[i].Stdout) || !bytes.Equal(crashed[i].StorageTranscript, actual[i].StorageTranscript) || !bytes.Equal(crashed[i].NetworkTranscript, actual[i].NetworkTranscript) || !bytes.Equal(crashed[i].InterruptedChoiceBytes, actual[i].InterruptedChoiceBytes) {
			measureErr = fmt.Errorf("%s interrupted cluster replay changed evidence: termination=%s message=%s err=%v", name, actual[i].Termination, actual[i].Message, errors[i])
			return
		}
		tape, err := choice.ProjectReplayPlan(recovered[i].ChoiceTrace, requests[i].Runtime.Choice.ExecutionIdentity)
		if err != nil {
			measureErr = err
			return
		}
		control, choices := *requests[i].Runtime, *requests[i].Runtime.Choice
		choices = choiceReplayRequest(&choices, choice.ModeReplay, recovered[i].ChoiceTrace.Bytes, tape)
		control.Choice, requests[i].Runtime = &choices, &control
		requests[i].StorageImage, requests[i].StdoutMirror = actual[i].StorageImage, nil
		for j, value := range requests[i].Config.Environment {
			if value == "GOMAD_TEMPORAL_PHASE=commit" {
				requests[i].Config.Environment[j] = "GOMAD_TEMPORAL_PHASE=reopen"
			}
		}
		requests[i].Broker, err = b.Restart(name)
		if err != nil {
			measureErr = err
			return
		}
	}
	runPair, runPair2, runPairErr2 := temporalRunPair(runCtx, evidence, "cluster-full-replay", requests, false, observers...)
	if runPairErr2 != nil {
		measureErr = runPairErr2
		return
	}
	actual, errors = runPair, runPair2
	for i, name := range names {
		if errors[i] != nil || actual[i].Termination != "exit" || actual[i].ExitCode == nil || *actual[i].ExitCode != 0 || !actual[i].Reaped || !bytes.Equal(recovered[i].Stdout, actual[i].Stdout) || !bytes.Equal(recovered[i].StorageTranscript, actual[i].StorageTranscript) || !bytes.Equal(recovered[i].NetworkTranscript, actual[i].NetworkTranscript) || !bytes.Equal(recovered[i].ChoiceTrace.Bytes, actual[i].ChoiceTrace.Bytes) {
			measureErr = fmt.Errorf("%s full cluster replay changed evidence: termination=%s message=%s err=%v", name, actual[i].Termination, actual[i].Message, errors[i])
			return
		}
		recoveryValue, recoveryValueErr := temporalRecoveryValue(runCtx, recovered[i].Stdout)
		if recoveryValueErr != nil {
			measureErr = recoveryValueErr
			return
		}
		recoveryValue2, recoveryValueErr2 := temporalRecoveryValue(runCtx, actual[i].Stdout)
		if recoveryValueErr2 != nil {
			measureErr = recoveryValueErr2
			return
		}
		historyOraclesErr := temporalHistoryOracles(runCtx, evidence, "temporal.replication."+name+"-replayed", recoveryValue, recoveryValue2)
		if historyOraclesErr != nil {
			measureErr = historyOraclesErr
			return
		}
	}
	encodedWorld, err := b.EncodeWorldEvidence()
	if err != nil {
		measureErr = err
		return
	}
	if !bytes.Equal(expected, encodedWorld) {
		measureErr = errutil.New("forced independent cluster replay changed shared World delivery order")
		return
	}
	writeEvidenceErr := temporalWrite(runCtx, filepath.Join(evidence, "two-cluster-replayed-world.json"), encodedWorld)
	if writeEvidenceErr != nil {
		measureErr = writeEvidenceErr
		return
	}
	writeReportErr := temporalJSON(runCtx, filepath.Join(evidence, "two-cluster-replayed-world-cardinality.json"), b.DiagnosticCardinality())
	if writeReportErr != nil {
		measureErr = writeReportErr
		return
	}
	return
}
func temporalReplayClustersInProcess(runCtx context.Context, evidence string, configuration wasi.BrokerConfig, requests []wasi.Request, crashed, recovered []wasi.Result, expected []byte, expectedNetwork netmodel.Record, observers ...temporalPairObserver) (measureErr error) {
	initial := append([]wasi.Request(nil), requests...)
	reopened := temporalReopenRequests(requests)
	for i := range initial {
		if !crashed[i].RuntimeInterrupted {
			measureErr = errutil.New("cluster replay lacks interrupted choice evidence")
			return
		}
		prefix, err := choice.ProjectInterruptedReplayPlan(crashed[i].InterruptedChoiceBytes, initial[i].Runtime.Choice.Limit, initial[i].Runtime.Choice.ExecutionIdentity)
		if err != nil {
			measureErr = err
			return
		}
		control, choices := *initial[i].Runtime, *initial[i].Runtime.Choice
		choices = choiceReplayRequest(&choices, choice.ModePrefix, crashed[i].InterruptedChoiceBytes, prefix)
		control.Choice, initial[i].Runtime = &choices, &control
		tape, err := choice.ProjectReplayPlan(recovered[i].ChoiceTrace, reopened[i].Runtime.Choice.ExecutionIdentity)
		if err != nil {
			measureErr = err
			return
		}
		recoveryControl, recoveryChoices := *reopened[i].Runtime, *reopened[i].Runtime.Choice
		recoveryChoices = choiceReplayRequest(&recoveryChoices, choice.ModeReplay, recovered[i].ChoiceTrace.Bytes, tape)
		recoveryControl.Choice, reopened[i].Runtime = &recoveryChoices, &recoveryControl
	}
	pair, openClusterPairErr := temporalOpenClusterPair(runCtx, configuration, initial, reopened)
	if openClusterPairErr != nil {
		measureErr = openClusterPairErr
		return
	}
	actual, failures, runErr := pair.run(runCtx, evidence, "cluster-prefix-replay", initial, 1, observers...)
	if runErr != nil {
		measureErr = runErr
		return
	}
	for i, name := range pair.names {
		if failures[i] != nil || actual[i].Termination != "crash" || !actual[i].Reaped || actual[i].StorageImage == nil {
			measureErr = fmt.Errorf("%s interrupted cluster replay: %s %s %v", name, actual[i].Termination, actual[i].Message, failures[i])
			return
		}
		diff := sameEvidence(crashed[i], actual[i])
		if diff != "" {
			measureErr = fmt.Errorf("%s interrupted cluster replay differs in %s", name, diff)
			return
		}
		diff = sameStorageEvidence(crashed[i], actual[i])
		if diff != "" {
			measureErr = fmt.Errorf("%s interrupted cluster replay storage differs in %s", name, diff)
			return
		}
		reopened[i].StorageImage = actual[i].StorageImage
		reopened[i].Broker = &wasi.BrokerSession{Node: wasi.BrokerNode{ID: name, Incarnation: 2}}
	}
	run2, run22, runErr2 := pair.run(runCtx, evidence, "cluster-full-replay", reopened, 2, observers...)
	if runErr2 != nil {
		measureErr = runErr2
		return
	}
	actual, failures = run2, run22
	for i, name := range pair.names {
		if failures[i] != nil {
			measureErr = failures[i]
			return
		}
		requireExitErr := requireExit(runCtx, "replayed "+name, actual[i], 0)
		if requireExitErr != nil {
			measureErr = requireExitErr
			return
		}
		diff := sameEvidence(recovered[i], actual[i])
		if diff != "" {
			measureErr = fmt.Errorf("%s full cluster replay differs in %s", name, diff)
			return
		}
		diff = sameStorageEvidence(recovered[i], actual[i])
		if diff != "" {
			measureErr = fmt.Errorf("%s full cluster replay storage differs in %s", name, diff)
			return
		}
		recoveryValue, recoveryValueErr := temporalRecoveryValue(runCtx, recovered[i].Stdout)
		if recoveryValueErr != nil {
			measureErr = recoveryValueErr
			return
		}
		recoveryValue2, recoveryValueErr2 := temporalRecoveryValue(runCtx, actual[i].Stdout)
		if recoveryValueErr2 != nil {
			measureErr = recoveryValueErr2
			return
		}
		historyOraclesErr := temporalHistoryOracles(runCtx, evidence, "temporal.replication."+name+"-replayed", recoveryValue, recoveryValue2)
		if historyOraclesErr != nil {
			measureErr = historyOraclesErr
			return
		}
	}
	finished, finishErr := pair.finish(runCtx, evidence, "two-cluster-replayed", observers...)
	if finishErr != nil {
		measureErr = finishErr
		return
	}
	if !bytes.Equal(expected, finished.World) {
		measureErr = errutil.New("forced independent cluster replay changed shared World delivery order")
		return
	}
	wantNetwork, err := json.Marshal(expectedNetwork)
	if err != nil {
		measureErr = err
		return
	}
	actualNetwork, err := json.Marshal(finished.Network)
	if err != nil {
		measureErr = err
		return
	}
	if !bytes.Equal(wantNetwork, actualNetwork) {
		measureErr = errutil.New("forced independent cluster replay changed shared network Record")
		return
	}
	writeEvidenceErr := temporalWrite(runCtx, filepath.Join(evidence, "two-cluster-replayed-world.json"), finished.World)
	if writeEvidenceErr != nil {
		measureErr = writeEvidenceErr
		return
	}
	summary, err := world.DecodeCompressedSnapshot(finished.World, world.CompressedSnapshotProfile())
	if err != nil {
		measureErr = err
		return
	}
	writeReportErr := temporalJSON(runCtx, filepath.Join(evidence, "two-cluster-replayed-world-cardinality.json"), summary)
	if writeReportErr != nil {
		measureErr = writeReportErr
		return
	}
	writeReportErr2 := temporalJSON(runCtx, filepath.Join(evidence, "two-cluster-replayed-network-record.json"), finished.Network)
	if writeReportErr2 != nil {
		measureErr = writeReportErr2
		return
	}
	return
}

type temporalGuestWarm struct {
	ready chan struct{}
	once  sync.Once
	ack   sync.Once
	ctx   context.Context
	node  string
}

func (w *temporalGuestWarm) Write(data []byte) (int, error) {
	if w.ready != nil {
		w.once.Do(func() {
			close(w.ready)
		})
	}
	if bytes.Contains(data, []byte(temporalObservationPrefix)) {
		w.ack.Do(func() {
			logf(w.ctx, "%s acknowledged committed Temporal history", w.node)
		})
	}
	return len(data), nil
}
func stringSeed(seed uint64) string {
	return strconv.FormatUint(seed, 10)
}
func hexDigest(digest [32]byte) string {
	return hex.EncodeToString(digest[:])
}
func migrationBaselinePreflight(root string) (map[string]prototypeRun, map[string]string, error) {
	baseline := map[string]prototypeRun{}
	modules := map[string]string{}
	for _, name := range []string{"scheduling", "virtual-time"} {
		data, err := os.ReadFile(filepath.Join(root, name+"-report.json"))
		if err != nil {
			return nil, nil, err
		}
		var report prototypeReport
		err = json.Unmarshal(data, &report)
		if err != nil {
			return nil, nil, err
		}
		for _, run := range report.Runs {
			if run.Instance == 0 && !strings.Contains(run.Variant, "warm") && !strings.Contains(run.Variant, "replay") && !strings.HasPrefix(run.Variant, "tamper-") && run.Termination == "exit" {
				if run.WallNanos <= 0 {
					return nil, nil, fmt.Errorf("baseline wall time must be positive: report=%s guest=%s variant=%s seed=%d", name, run.Guest, run.Variant, run.Seed)
				}
				baseline[run.Guest+"/"+run.Variant+"/"+stringSeed(run.Seed)] = run
			}
		}
		for name, guest := range report.Guests {
			hash, ok := guest["module_sha256"].(string)
			if ok {
				modules[name] = hash
			}
		}
	}
	return baseline, modules, nil
}
func migrationBaselineModule(module, want string) error {
	data, err := os.ReadFile(module)
	if want == "" || err != nil || digest(data) != want {
		return errutil.New("baseline guest module identity unavailable or changed")
	}
	return nil
}
func measureMigration(runCtx context.Context) (measureErr error) {
	root := input(runCtx, "GOMAD_WASM_MIGRATION_BASELINE")
	if root == "" {
		measureErr = fmt.Errorf("%w: %v", ErrRefused, errutil.New("set GOMAD_WASM_MIGRATION_BASELINE to retained pre-migration prototype evidence"))
		return
	}
	if !filepath.IsAbs(root) {
		measureErr = errutil.New("migration baseline must be absolute")
		return
	}
	baseline, modules, err := migrationBaselinePreflight(root)
	if err != nil {
		measureErr = err
		return
	}
	report, prototypeReportForErr := prototypeReportFor(runCtx, "helper-migration")
	if prototypeReportForErr != nil {
		measureErr = prototypeReportForErr
		return
	}
	for _, guest := range []struct{ name, variant, module, identity string }{{"choices", "strict", "choices.wasm", "choices"}, {"runtime_probes", "strict", "runtime_probes.wasm", "runtime_probes"}, {"time_probes", "strict", "time_probes.wasm", "time_probes"}} {
		runStep2, runStepErr := runStep(runCtx, guest.name+"/"+guest.variant, func(runCtx context.Context) (measureErr error) {
			module := filepath.Join(root, "modules", guest.module)
			err := migrationBaselineModule(module, modules[guest.identity])
			if err != nil {
				measureErr = err
				return
			}
			config := prototypeFixtureConfig()
			if guest.name == "runtime_probes" {
				config.Environment = append(config.Environment, "EXTRA_ENTROPY=no_")
			}
			request, requestErr := prototypeRequest(runCtx, module, config, 0, 30*time.Minute)
			if requestErr != nil {
				measureErr = requestErr
				return
			}
			request.CompileCache = filepath.Join(root, "compile-cache")
			cache := input(runCtx, "GOMAD_WASM_MIGRATION_CACHE")
			if cache != "" {
				if !filepath.IsAbs(cache) {
					measureErr = errutil.New("migration compiled cache must be absolute")
					return
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
			prepared, prototypeExecuteErr := prototypeExecute(runCtx, report, guest.name, guest.variant+"-prepare", 0, warm)
			if prototypeExecuteErr != nil {
				measureErr = prototypeExecuteErr
				return
			}
			requireExitErr := requireExit(runCtx, "cache prepare", prepared, 0, 17)
			if requireExitErr != nil {
				measureErr = requireExitErr
				return
			}
			var first wasi.Result
			var beforeNanos, afterNanos int64
			for _, seed := range []uint64{0, 1000003} {
				request.Runtime.Seed = seed
				old, ok := baseline[guest.name+"/"+guest.variant+"/"+stringSeed(seed)]
				if !ok {
					measureErr = errutil.New("missing baseline seed")
					return
				}
				started := time.Now()
				result, prototypeExecuteErr2 := prototypeExecute(runCtx, report, guest.name, guest.variant, 0, request)
				if prototypeExecuteErr2 != nil {
					measureErr = prototypeExecuteErr2
					return
				}
				elapsed := time.Since(started).Nanoseconds()
				requireExitErr2 := requireExit(runCtx, "migrated", result, 0, 17)
				if requireExitErr2 != nil {
					measureErr = requireExitErr2
					return
				}
				if digest(result.Stdout) != old.StdoutSHA256 || digest(result.ChoiceTrace.Bytes) != old.ChoiceSHA256 || digest(result.NetworkTranscript) != old.NetworkSHA256 || hexDigest(result.TranscriptSHA256) != old.WASISHA256 || int64FromTerminal(result.RuntimeTerminal()) != int64(old.VirtualNanos) || int(result.Calls) != old.Calls || result.TimeAdvances != old.TimeAdvances {
					measureErr = fmt.Errorf("baseline evidence changed: seed=%d stdout=%v choice=%v network=%v wasi=%v calls=%d/%d virtual=%d/%d", seed, digest(result.Stdout) == old.StdoutSHA256, digest(result.ChoiceTrace.Bytes) == old.ChoiceSHA256, digest(result.NetworkTranscript) == old.NetworkSHA256, hexDigest(result.TranscriptSHA256) == old.WASISHA256, result.Calls, old.Calls, int64FromTerminal(result.RuntimeTerminal()), old.VirtualNanos)
					return
				}
				beforeNanos += old.WallNanos
				afterNanos += elapsed
				if seed == 0 {
					first = result
				}
			}
			tape, err := choice.ProjectReplayPlan(first.ChoiceTrace, request.Runtime.Choice.ExecutionIdentity)
			if err != nil {
				measureErr = err
				return
			}
			request.Runtime.Seed = 0
			*request.Runtime.Choice = choiceReplayRequest(request.Runtime.Choice, choice.ModeReplay, first.ChoiceTrace.Bytes, tape)
			replayed, prototypeExecuteErr3 := prototypeExecute(runCtx, report, guest.name, guest.variant+"-replay", 0, request)
			if prototypeExecuteErr3 != nil {
				measureErr = prototypeExecuteErr3
				return
			}
			difference := sameEvidence(first, replayed)
			if difference != "" {
				measureErr = fmt.Errorf("migrated replay differs in %s: %s %s", difference, replayed.Termination, replayed.Message)
				return
			}
			speedup := float64(beforeNanos) / float64(afterNanos)
			logf(runCtx, "same module/seeds: baseline=%s migrated=%s speedup=%.3fx", time.Duration(beforeNanos), time.Duration(afterNanos), speedup)
			if err := report.guest(guest.variant, map[string]any{"module_sha256": request.ModuleSHA256, "baseline_wall_nanos": beforeNanos, "migrated_wall_nanos": afterNanos, "speedup": speedup, "byte_identical": true}); err != nil {
				measureErr = err
				return
			}
			return
		})
		if runStepErr != nil {
			measureErr = runStepErr
			return
		}
		_ = runStep2
	}
	return
}
func measureMatchedSuite(runCtx context.Context) (measureErr error) {
	module := input(runCtx, "GOMAD_WASM_MATCHED_MODULE")
	baselineHelper := input(runCtx, "GOMAD_WASM_MATCHED_BASELINE_HELPER")
	if module == "" || baselineHelper == "" {
		measureErr = errutil.Join(measureErr, fmt.Errorf("%w: %v", ErrRefused, errutil.New("set matched immutable module and original helper")))
		return
	}
	if !filepath.IsAbs(module) || !filepath.IsAbs(baselineHelper) {
		measureErr = errutil.Join(measureErr, errutil.New("matched inputs must be absolute"))
		return
	}
	prototypeToolchain2, prototypeToolchain22, prototypeToolchain3, prototypeToolchainErr := prototypeToolchain(runCtx)
	if prototypeToolchainErr != nil {
		measureErr = errutil.Join(measureErr, prototypeToolchainErr)
		return
	}
	_, _, toolchainIdentity := prototypeToolchain2, prototypeToolchain22, prototypeToolchain3
	declared := input(runCtx, "GOMAD_WASM_MATCHED_TOOLCHAIN_ID")
	if declared != "" && declared != toolchainIdentity {
		measureErr = errutil.Join(measureErr, fmt.Errorf("declared matched toolchain identity %s differs from cooperative overlay %s", declared, toolchainIdentity))
		return
	}
	data, err := os.ReadFile(module)
	if err != nil {
		measureErr = errutil.Join(measureErr, err)
		return
	}
	imports, pages, binaryInventoryErr := binaryInventory(runCtx, data)
	if binaryInventoryErr != nil {
		measureErr = errutil.Join(measureErr, binaryInventoryErr)
		return
	}
	helper, helperErr := helperPath(runCtx)
	if helperErr != nil {
		measureErr = errutil.Join(measureErr, helperErr)
		return
	}
	repo, err := absolute(runCtx, "../../../../../..")
	if err != nil {
		measureErr = errutil.Join(measureErr, err)
		return
	}
	report, prototypeReportForErr := prototypeReportFor(runCtx, "helper-matched-suite")
	if prototypeReportForErr != nil {
		measureErr = errutil.Join(measureErr, prototypeReportForErr)
		return
	}
	cache := input(runCtx, "GOMAD_WASM_MATCHED_CACHE")
	if cache == "" {
		prototypeRoot2, prototypeRootErr := prototypeRoot(runCtx)
		if prototypeRootErr != nil {
			measureErr = errutil.Join(measureErr, prototypeRootErr)
			return
		}
		cache = filepath.Join(prototypeRoot2, "compile-cache")
	}
	if !filepath.IsAbs(cache) {
		measureErr = errutil.Join(measureErr, errutil.New("matched compiled cache must be absolute"))
		return
	}
	baselineCache := input(runCtx, "GOMAD_WASM_MATCHED_BASELINE_CACHE")
	err = matchedBaselineCache(baselineCache, cache)
	if err != nil {
		measureErr = errutil.Join(measureErr, err)
		return
	}
	selection := input(runCtx, "GOMAD_WASM_MATCHED_SUITE")
	prototypeRoot3, prototypeRootErr2 := prototypeRoot(runCtx)
	if prototypeRootErr2 != nil {
		measureErr = errutil.Join(measureErr, prototypeRootErr2)
		return
	}
	preparations, err := os.OpenFile(filepath.Join(prototypeRoot3, "matched-preparations.jsonl"), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		measureErr = errutil.Join(measureErr, err)
		return
	}
	addCleanup(runCtx, func(runCtx context.Context) (measureErr error) {
		err := preparations.Close()
		if err != nil {
			measureErr = errutil.Join(measureErr, err)
		}
		return
	})
	preparationEpoch := 0
	preparationEvent := func(runCtx context.Context, event string, request wasi.Request, cold bool) (measureErr error) {
		data, err := json.Marshal(struct {
			Event        string `json:"event"`
			Epoch        int    `json:"epoch"`
			Helper       string `json:"helper"`
			HelperSHA256 string `json:"helper_sha256"`
			Cold         bool   `json:"cold"`
		}{event, preparationEpoch, request.HelperPath, request.HelperSHA256, cold})
		if err != nil {
			measureErr = errutil.Join(measureErr, err)
			return
		}
		_, err = preparations.Write(append(data, '\n'))
		if err != nil {
			measureErr = errutil.Join(measureErr, err)
			return
		}
		return
	}
	for _, suite := range []struct{ name, cwd, run string }{{"workflow", "/workspace/tests", "^TestWorkflowTypeEncodingSuite$/^TestPlainASCII$/^Succeeds$"}, {"activity", "/workspace/tests", "^TestActivityTestSuite$"}, {"child", "/workspace/tests", "^TestChildWorkflowSuite$"}, {"update", "/workspace/tests", "^TestWorkflowUpdateSuite$"}, {"timers", "/workspace/tests", "^TestUserTimersTestSuite$"}, {"frontend", "/workspace/tests/gomadfunctional", "^TestFrontendSystemInfo$"}} {
		if !matchedSuiteSelected(selection, suite.name) {
			continue
		}
		runStep2, runStepErr := runStep(runCtx, suite.name, func(runCtx context.Context) (measureErr error) {
			config, prototypeTemporalConfigErr := prototypeTemporalConfig(runCtx, repo, suite.cwd, "-test.run="+suite.run, "-persistenceType=sql", "-persistenceDriver=sqlite")
			if prototypeTemporalConfigErr != nil {
				measureErr = errutil.Join(measureErr, prototypeTemporalConfigErr)
				return
			}
			config.Profile, config.Clock.ReadStepNanos = wasi.CooperativeProfile, 0
			identity := choice.ExecutionIdentity{TargetSHA256: sha256.Sum256(data), ToolchainBuildKey: toolchainIdentity, GOOS: "wasip1", GOARCH: "wasm", ImplementationSHA256: sha256.Sum256([]byte(toolchainIdentity))}
			helperSHA256, helperSHA256Err := fileHash(runCtx, helper)
			if helperSHA256Err != nil {
				measureErr = errutil.Join(measureErr, helperSHA256Err)
				return
			}
			moduleSHA256, moduleSHA256Err := fileHash(runCtx, module)
			if moduleSHA256Err != nil {
				measureErr = errutil.Join(measureErr, moduleSHA256Err)
				return
			}
			request := wasi.Request{HelperPath: helper, HelperSHA256: helperSHA256, ModulePath: module, ModuleSHA256: moduleSHA256, Engine: wasi.PinnedEngine(), Imports: imports, InitialMemoryPages: pages, MemoryBytes: 4 << 30, Fuel: 1 << 50, Config: config, Timeout: 30 * time.Minute, CompileCache: cache, DigestTranscript: true, Runtime: &wasi.RuntimeControl{Choice: &runnerbackend.ChoiceRequest{Mode: choice.ModeRecord, ExecutionIdentity: identity, Limit: choice.MaximumTraceBytes}}}
			baseline := request
			baselineHelperSHA256, baselineHelperSHA256Err := fileHash(runCtx, baselineHelper)
			if baselineHelperSHA256Err != nil {
				measureErr = errutil.Join(measureErr, baselineHelperSHA256Err)
				return
			}
			baseline.HelperPath, baseline.HelperSHA256, baseline.CompileCache = baselineHelper, baselineHelperSHA256, baselineCache
			baseline.SetRemoteModel(true)
			for index, warm := range []wasi.Request{baseline, request} {
				warm.Config.Args = append([]string(nil), warm.Config.Args...)
				for index, arg := range warm.Config.Args {
					if strings.HasPrefix(arg, "-test.run=") {
						warm.Config.Args[index] = "-test.run=^$"
					}
				}
				started := time.Now()
				entries, err := os.ReadDir(warm.CompileCache)
				if err != nil {
					measureErr = errutil.Join(measureErr, err)
					return
				}
				preparationEventErr := preparationEvent(runCtx, "start", warm, len(entries) == 0)
				if preparationEventErr != nil {
					measureErr = errutil.Join(measureErr, preparationEventErr)
					return
				}
				result, err := matchedCurrentPrepareAdmitted(runCtx, input(runCtx, "GOMAD_WASM_MATCHED_COLD_MEMORY_GUARD"), func() (wasi.Result, error) {
					return wasi.Run(runCtx, warm)
				})
				preparationEventErr2 := preparationEvent(runCtx, "end", warm, len(entries) == 0)
				if preparationEventErr2 != nil {
					measureErr = errutil.Join(measureErr, preparationEventErr2)
					return
				}
				preparationEpoch++
				if err != nil {
					measureErr = errutil.Join(measureErr, err)
					return
				}
				requireExitErr := requireExit(runCtx, "cache warmup", result, 0)
				if requireExitErr != nil {
					measureErr = errutil.Join(measureErr, requireExitErr)
					return
				}
				if err := report.guest(suite.name+"-prepare-"+strconv.Itoa(index), map[string]any{"helper_sha256": warm.HelperSHA256, "module_sha256": warm.ModuleSHA256, "wall_nanos": time.Since(started).Nanoseconds(), "phase_metrics": result.Telemetry, "wait_resource_usage": result.HelperResourceUsage}); err != nil {
					measureErr = errutil.Join(measureErr, err)
					return
				}
			}
			var beforeNanos, afterNanos int64
			var beforeExecutionNanos, afterExecutionNanos uint64
			var first wasi.Result
			for sample, seed := range []uint64{0, 1000003, 2000006, 3000009} {
				request.Runtime.Seed = seed
				var before, after wasi.Result
				for turn := range 2 {
					started := time.Now()
					if matchedBaselineTurn(sample, turn) {
						prototypeExecute2, prototypeExecuteErr := prototypeExecute(runCtx, report, suite.name, "baseline", 0, baseline)
						if prototypeExecuteErr != nil {
							measureErr = errutil.Join(measureErr, prototypeExecuteErr)
							return
						}
						before = prototypeExecute2
						beforeNanos += time.Since(started).Nanoseconds()
					} else {
						prototypeExecute3, prototypeExecuteErr2 := prototypeExecute(runCtx, report, suite.name, "candidate", 0, request)
						if prototypeExecuteErr2 != nil {
							measureErr = errutil.Join(measureErr, prototypeExecuteErr2)
							return
						}
						after = prototypeExecute3
						afterNanos += time.Since(started).Nanoseconds()
					}
				}
				if before.AuthorizationToTerminalNanos == 0 || after.AuthorizationToTerminalNanos == 0 {
					measureErr = errutil.Join(measureErr, errutil.New("validated authorization-to-terminal measurements unavailable"))
					return
				}
				beforeExecutionNanos += before.AuthorizationToTerminalNanos
				afterExecutionNanos += after.AuthorizationToTerminalNanos
				requireExitErr2 := requireExit(runCtx, "baseline", before, 0)
				if requireExitErr2 != nil {
					measureErr = errutil.Join(measureErr, requireExitErr2)
					return
				}
				requireExitErr3 := requireExit(runCtx, "candidate", after, 0)
				if requireExitErr3 != nil {
					measureErr = errutil.Join(measureErr, requireExitErr3)
					return
				}
				if after.GoFilesystemCalls == nil || len(after.GoFilesystemCalls) != 0 {
					measureErr = errutil.Join(measureErr, errutil.New(fmt.Sprint("matched candidate namespace leaked filesystem callbacks", after.GoFilesystemCalls)))
					return
				}
				witness := []byte("=== RUN   " + strings.Trim(strings.Split(suite.run, "/")[0], "^$"))
				if !bytes.Contains(before.Stdout, witness) || !bytes.Contains(after.Stdout, witness) {
					measureErr = errutil.Join(measureErr, fmt.Errorf("selected suite did not execute: missing %q", witness))
					return
				}
				difference := sameEvidence(before, after)
				if difference != "" || before.EnvironmentIdentity != after.EnvironmentIdentity || before.Calls != after.Calls || before.TimeAdvances != after.TimeAdvances {
					measureErr = errutil.Join(measureErr, fmt.Errorf("same-module paired evidence differs: %s seed=%d", difference, seed))
					return
				}
				if seed == 0 {
					first = before
				}
			}
			plan, err := choice.ProjectReplayPlan(first.ChoiceTrace, identity)
			if err != nil {
				measureErr = errutil.Join(measureErr, err)
				return
			}
			request.Runtime.Seed = 0
			*request.Runtime.Choice = choiceReplayRequest(request.Runtime.Choice, choice.ModeReplay, first.ChoiceTrace.Bytes, plan)
			replayed, prototypeExecuteErr3 := prototypeExecute(runCtx, report, suite.name, "candidate-replay", 0, request)
			if prototypeExecuteErr3 != nil {
				measureErr = errutil.Join(measureErr, prototypeExecuteErr3)
				return
			}
			difference := sameEvidence(first, replayed)
			if difference != "" {
				measureErr = errutil.Join(measureErr, fmt.Errorf("original-helper tape differs in candidate: %s", difference))
				return
			}
			speedup := float64(beforeNanos) / float64(afterNanos)
			executionSpeedup := float64(beforeExecutionNanos) / float64(afterExecutionNanos)
			if err := report.guest(suite.name, map[string]any{"compatibility_profile": "prototype-legacy-v1", "module_sha256": request.ModuleSHA256, "runtime_identity": identity.ToolchainBuildKey, "model_sha256": wasi.ImplementationSHA256(), "reference_owner": "current_go_model", "host_os": runtime.GOOS, "host_arch": runtime.GOARCH, "engine": request.Engine, "environment_identity": first.EnvironmentIdentity, "baseline_helper_sha256": baseline.HelperSHA256, "candidate_helper_sha256": request.HelperSHA256, "baseline_wall_nanos": beforeNanos, "candidate_wall_nanos": afterNanos, "speedup": speedup, "byte_identical": true}); err != nil {
				measureErr = errutil.Join(measureErr, err)
				return
			}
			if err := report.guest(suite.name+"-execution-boundary", map[string]any{"module_sha256": request.ModuleSHA256, "runtime_identity": identity.ToolchainBuildKey, "samples": 4, "order": "alternating-baseline-first-on-even", "boundary": "authorize-send-to-validated-terminal-includes-instantiation-and-evidence-flush", "baseline_nanos": beforeExecutionNanos, "candidate_nanos": afterExecutionNanos, "speedup": executionSpeedup, "full_wall_speedup": speedup}); err != nil {
				measureErr = errutil.Join(measureErr, err)
				return
			}
			logf(runCtx, "same module/seeds/tape: full wall baseline=%s candidate=%s speedup=%.3fx; authorized execution baseline=%s candidate=%s speedup=%.3fx", time.Duration(beforeNanos), time.Duration(afterNanos), speedup, time.Duration(beforeExecutionNanos), time.Duration(afterExecutionNanos), executionSpeedup)
			return
		})
		if runStepErr != nil {
			measureErr = errutil.Join(measureErr, runStepErr)
			return
		}
		_ = runStep2
	}
	return
}
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
		return errutil.New("GOMAD_WASM_MATCHED_BASELINE_CACHE must be absolute and distinct from the candidate cache")
	}
	return nil
}
func matchedBaselineTurn(sample, turn int) bool {
	return (sample+turn)%2 == 0
}
func matchedPrepareAdmitted(platform, path, mode string, dispatch func() (wasi.Result, error)) (wasi.Result, error) {
	if mode == "" {
		return dispatch()
	}
	if mode != "1" || platform != "linux" {
		return wasi.Result{}, errutil.New("explicit matched memory admission requires mode 1 on Linux")
	}
	data, err := hostfs.ReadBounded(path, 8<<10)
	if err != nil {
		return wasi.Result{}, err
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
			return wasi.Result{}, errutil.New("unique exact MemAvailable field in kB required")
		}
		seen = true
		value, err := matchedMemoryDecimal(fields[1])
		if err != nil || value > ^uint64(0)/1024 {
			return wasi.Result{}, errutil.New("invalid or overflowing MemAvailable value")
		}
		available = value * 1024
	}
	if !seen || available < 8<<30 {
		return wasi.Result{}, errutil.New("at least 8GiB MemAvailable required before matched preparation dispatch")
	}
	return dispatch()
}
func matchedMemoryDecimal(value string) (uint64, error) {
	if len(value) == 0 || len(value) > 20 {
		return 0, errutil.New("bounded memory decimal required")
	}
	for _, character := range value {
		if character < '0' || character > '9' {
			return 0, errutil.New("memory decimal digits required")
		}
	}
	return strconv.ParseUint(value, 10, 64)
}
func measureLegacyWorkflowProfile(runCtx context.Context) (measureErr error) {
	root := input(runCtx, "GOMAD_WASM_WORKFLOW_PROFILE_ROOT")
	if root == "" {
		measureErr = fmt.Errorf("%w: %v", ErrRefused, errutil.New("set GOMAD_WASM_WORKFLOW_PROFILE_ROOT for diagnostic-only retained workflow profiling"))
		return
	}
	module, cache, runtimeIdentity := input(runCtx, "GOMAD_WASM_MATCHED_MODULE"), input(runCtx, "GOMAD_WASM_MATCHED_CACHE"), input(runCtx, "GOMAD_WASM_MATCHED_TOOLCHAIN_ID")
	if !filepath.IsAbs(root) || !filepath.IsAbs(module) || !filepath.IsAbs(cache) || runtimeIdentity == "" {
		measureErr = errutil.New("profile requires absolute evidence/module/existing cache paths and explicit legacy runtime identity")
		return
	}
	err := os.MkdirAll(root, 0700)
	if err != nil {
		measureErr = err
		return
	}
	data, err := os.ReadFile(module)
	if err != nil {
		measureErr = err
		return
	}
	imports, pages, binaryInventoryErr := binaryInventory(runCtx, data)
	if binaryInventoryErr != nil {
		measureErr = binaryInventoryErr
		return
	}
	repo, err := absolute(runCtx, "../../../../../..")
	if err != nil {
		measureErr = err
		return
	}
	config, prototypeTemporalConfigErr := prototypeTemporalConfig(runCtx, repo, "/workspace/tests", "-test.run=^TestWorkflowTypeEncodingSuite$/^TestPlainASCII$/^Succeeds$", "-persistenceType=sql", "-persistenceDriver=sqlite")
	if prototypeTemporalConfigErr != nil {
		measureErr = prototypeTemporalConfigErr
		return
	}
	config.Profile, config.Clock.ReadStepNanos = wasi.CooperativeProfile, 0
	helper, helperErr := helperPath(runCtx)
	if helperErr != nil {
		measureErr = helperErr
		return
	}
	identity := choice.ExecutionIdentity{TargetSHA256: sha256.Sum256(data), ToolchainBuildKey: runtimeIdentity, GOOS: "wasip1", GOARCH: "wasm", ImplementationSHA256: sha256.Sum256([]byte(runtimeIdentity))}
	helperSHA256, helperSHA256Err := fileHash(runCtx, helper)
	if helperSHA256Err != nil {
		measureErr = helperSHA256Err
		return
	}
	moduleSHA256, moduleSHA256Err := fileHash(runCtx, module)
	if moduleSHA256Err != nil {
		measureErr = moduleSHA256Err
		return
	}
	request := wasi.Request{HelperPath: helper, HelperSHA256: helperSHA256, ModulePath: module, ModuleSHA256: moduleSHA256, Engine: wasi.PinnedEngine(), Imports: imports, InitialMemoryPages: pages, MemoryBytes: 4 << 30, Fuel: 1 << 50, Config: config, Timeout: 3 * time.Minute, CompileCache: cache, DigestTranscript: true, Runtime: &wasi.RuntimeControl{Choice: &runnerbackend.ChoiceRequest{Mode: choice.ModeRecord, ExecutionIdentity: identity, Limit: choice.MaximumTraceBytes}}}
	var first wasi.Result
	for _, seed := range []uint64{0, 1000003} {
		request.Runtime.Seed = seed
		profile, err := os.Create(filepath.Join(root, "seed"+stringSeed(seed)+"-parent.pprof"))
		if err != nil {
			measureErr = err
			return
		}
		err = pprof.StartCPUProfile(profile)
		if err != nil {
			closeResource(runCtx, profile)
			measureErr = err
			return
		}
		started := time.Now()
		result, runErr := wasi.Run(runCtx, request)
		wall := time.Since(started).Nanoseconds()
		pprof.StopCPUProfile()
		err = profile.Close()
		if err != nil {
			measureErr = err
			return
		}
		if runErr != nil {
			measureErr = errutil.Join(runErr)
			return
		}
		requireExitErr := requireExit(runCtx, "profile", result, 0)
		if requireExitErr != nil {
			measureErr = requireExitErr
			return
		}
		if result.Telemetry == nil || !result.Telemetry.CacheHit || result.Telemetry.CompileNanos != 0 || !bytes.Contains(result.Stdout, []byte("=== RUN   TestWorkflowTypeEncodingSuite")) {
			measureErr = errutil.New("profile did not execute the unchanged selected workflow using existing compiled code")
			return
		}
		writeReportErr := temporalJSON(runCtx, filepath.Join(root, "seed"+stringSeed(seed)+"-metrics.json"), map[string]any{"diagnostic_only": true, "profile": "prototype-legacy-v1", "module_sha256": request.ModuleSHA256, "helper_sha256": request.HelperSHA256, "runtime_identity": runtimeIdentity, "model_sha256": wasi.ImplementationSHA256(), "wall_nanos": wall, "authorization_to_terminal_nanos": result.AuthorizationToTerminalNanos, "phase_metrics": result.Telemetry, "wait_resource_usage": result.HelperResourceUsage, "choices": result.ChoiceTrace.Summary})
		if writeReportErr != nil {
			measureErr = writeReportErr
			return
		}
		if seed == 0 {
			first = result
		}
	}
	request.Runtime.Seed, request.DigestTranscript = 0, false
	retained, err := wasi.Run(runCtx, request)
	if err != nil {
		measureErr = err
		return
	}
	requireExitErr2 := requireExit(runCtx, "transcript retention", retained, 0)
	if requireExitErr2 != nil {
		measureErr = requireExitErr2
		return
	}
	difference := sameEvidence(first, retained)
	if difference != "" {
		measureErr = fmt.Errorf("retaining canonical transcript changed evidence: %s", difference)
		return
	}
	operations := map[string]uint64{}
	records, err := wasi.DecodeTranscript(retained.Transcript)
	if err != nil {
		measureErr = err
		return
	}
	for _, record := range records {
		operations[record.Op] += record.Count
	}
	writeReportErr2 := temporalJSON(runCtx, filepath.Join(root, "canonical-operation-counts.json"), operations)
	if writeReportErr2 != nil {
		measureErr = writeReportErr2
		return
	}
	writeEvidenceErr := temporalWrite(runCtx, filepath.Join(root, "seed0-choice.bin"), first.ChoiceTrace.Bytes)
	if writeEvidenceErr != nil {
		measureErr = writeEvidenceErr
		return
	}
	return
}
func measureFailingSuiteProfile(runCtx context.Context) (measureErr error) {
	root := input(runCtx, "GOMAD_WASM_DIAGNOSTIC_PROFILE_ROOT")
	if root == "" {
		measureErr = errutil.Join(measureErr, fmt.Errorf("%w: %v", ErrRefused, errutil.New("set diagnostic profile inputs after the exclusive guest window is scheduled")))
		return
	}
	module, cache := input(runCtx, "GOMAD_WASM_MATCHED_MODULE"), input(runCtx, "GOMAD_WASM_MATCHED_CACHE")
	referenceHelper, referenceCache := input(runCtx, "GOMAD_WASM_PROFILE_REFERENCE_HELPER"), input(runCtx, "GOMAD_WASM_PROFILE_REFERENCE_CACHE")
	runtimeIdentity := input(runCtx, "GOMAD_WASM_MATCHED_TOOLCHAIN_ID")
	for _, name := range []string{root, module, cache, referenceHelper, referenceCache} {
		if !filepath.IsAbs(name) {
			measureErr = errutil.Join(measureErr, errutil.New("diagnostic profile requires absolute paths"))
			return
		}
	}
	referenceHelperSHA256, referenceHelperSHA256Err := fileHash(runCtx, referenceHelper)
	if referenceHelperSHA256Err != nil {
		measureErr = errutil.Join(measureErr, referenceHelperSHA256Err)
		return
	}
	helper2, helperErr := helperPath(runCtx)
	if helperErr != nil {
		measureErr = errutil.Join(measureErr, helperErr)
		return
	}
	helper2SHA256, helper2SHA256Err := fileHash(runCtx, helper2)
	if helper2SHA256Err != nil {
		measureErr = errutil.Join(measureErr, helper2SHA256Err)
		return
	}
	if runtimeIdentity == "" || referenceHelperSHA256 == helper2SHA256 {
		measureErr = errutil.Join(measureErr, errutil.New("diagnostic helper must have a distinct identity and explicit legacy runtime"))
		return
	}
	err := os.MkdirAll(root, 0700)
	if err != nil {
		measureErr = errutil.Join(measureErr, err)
		return
	}
	data, err := os.ReadFile(module)
	if err != nil {
		measureErr = errutil.Join(measureErr, err)
		return
	}
	imports, pages, binaryInventoryErr := binaryInventory(runCtx, data)
	if binaryInventoryErr != nil {
		measureErr = errutil.Join(measureErr, binaryInventoryErr)
		return
	}
	repo, err := absolute(runCtx, "../../../../../..")
	if err != nil {
		measureErr = errutil.Join(measureErr, err)
		return
	}
	identity := choice.ExecutionIdentity{TargetSHA256: sha256.Sum256(data), ToolchainBuildKey: runtimeIdentity, GOOS: "wasip1", GOARCH: "wasm", ImplementationSHA256: sha256.Sum256([]byte(runtimeIdentity))}
	preparations, err := os.OpenFile(filepath.Join(root, "matched-preparations.jsonl"), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		measureErr = errutil.Join(measureErr, err)
		return
	}
	addCleanup(runCtx, func(runCtx context.Context) (measureErr error) {
		err := preparations.Close()
		if err != nil {
			measureErr = errutil.Join(measureErr, err)
		}
		return
	})
	for index, suite := range []struct{ name, run string }{{"workflow", "^TestWorkflowTypeEncodingSuite$/^TestPlainASCII$/^Succeeds$"}, {"child", "^TestChildWorkflowSuite$"}, {"timers", "^TestUserTimersTestSuite$"}} {
		runStep2, runStepErr := runStep(runCtx, suite.name, func(runCtx context.Context) (measureErr error) {
			config, prototypeTemporalConfigErr := prototypeTemporalConfig(runCtx, repo, "/workspace/tests", "-test.run="+suite.run, "-persistenceType=sql", "-persistenceDriver=sqlite")
			if prototypeTemporalConfigErr != nil {
				measureErr = errutil.Join(measureErr, prototypeTemporalConfigErr)
				return
			}
			config.Profile, config.Clock.ReadStepNanos = wasi.CooperativeProfile, 0
			helper, helperErr2 := helperPath(runCtx)
			if helperErr2 != nil {
				measureErr = errutil.Join(measureErr, helperErr2)
				return
			}
			helperSHA256, helperSHA256Err := fileHash(runCtx, helper)
			if helperSHA256Err != nil {
				measureErr = errutil.Join(measureErr, helperSHA256Err)
				return
			}
			moduleSHA256, moduleSHA256Err := fileHash(runCtx, module)
			if moduleSHA256Err != nil {
				measureErr = errutil.Join(measureErr, moduleSHA256Err)
				return
			}
			request := wasi.Request{HelperPath: helper, HelperSHA256: helperSHA256, ModulePath: module, ModuleSHA256: moduleSHA256, Engine: wasi.PinnedEngine(), Imports: imports, InitialMemoryPages: pages, MemoryBytes: 4 << 30, Fuel: 1 << 50, Config: config, Timeout: 5 * time.Minute, CompileCache: cache, DigestTranscript: true, Runtime: &wasi.RuntimeControl{Choice: &runnerbackend.ChoiceRequest{Mode: choice.ModeRecord, ExecutionIdentity: identity, Limit: choice.MaximumTraceBytes}}}
			reference := request
			referenceHelperSHA2562, referenceHelperSHA256Err2 := fileHash(runCtx, referenceHelper)
			if referenceHelperSHA256Err2 != nil {
				measureErr = errutil.Join(measureErr, referenceHelperSHA256Err2)
				return
			}
			reference.HelperPath, reference.HelperSHA256, reference.CompileCache = referenceHelper, referenceHelperSHA2562, referenceCache
			if index == 0 {
				for epoch, owner := range []struct {
					name    string
					request wasi.Request
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
						measureErr = errutil.Join(measureErr, err)
						return
					}
					cold := len(entries) == 0
					profilePreparationEventErr := profilePreparationEvent(runCtx, preparations, "start", epoch, warm, cold)
					if profilePreparationEventErr != nil {
						measureErr = errutil.Join(measureErr, profilePreparationEventErr)
						return
					}
					runDiagnosticProfile2, runDiagnosticProfileErr := runDiagnosticProfile(runCtx, root, owner.name+"-prepare", warm, owner.name == "diagnostic", !cold)
					if runDiagnosticProfileErr != nil {
						measureErr = errutil.Join(measureErr, runDiagnosticProfileErr)
						return
					}
					_ = runDiagnosticProfile2
					profilePreparationEventErr2 := profilePreparationEvent(runCtx, preparations, "end", epoch, warm, cold)
					if profilePreparationEventErr2 != nil {
						measureErr = errutil.Join(measureErr, profilePreparationEventErr2)
						return
					}
				}
			}
			var first wasi.Result
			for _, seed := range []uint64{0, 1000003} {
				request.Runtime.Seed = seed
				before, runDiagnosticProfileErr2 := runDiagnosticProfile(runCtx, root, suite.name+"-reference-seed"+stringSeed(seed), reference, false, true)
				if runDiagnosticProfileErr2 != nil {
					measureErr = errutil.Join(measureErr, runDiagnosticProfileErr2)
					return
				}
				after, runDiagnosticProfileErr3 := runDiagnosticProfile(runCtx, root, suite.name+"-diagnostic-seed"+stringSeed(seed), request, true, true)
				if runDiagnosticProfileErr3 != nil {
					measureErr = errutil.Join(measureErr, runDiagnosticProfileErr3)
					return
				}
				requireDiagnosticEvidenceErr := requireDiagnosticEvidence(runCtx, before, after)
				if requireDiagnosticEvidenceErr != nil {
					measureErr = errutil.Join(measureErr, requireDiagnosticEvidenceErr)
					return
				}
				if !bytes.Contains(before.Stdout, []byte("=== RUN   "+strings.Trim(strings.Split(suite.run, "/")[0], "^$"))) {
					measureErr = errutil.Join(measureErr, errutil.New("diagnostic selected suite did not run"))
					return
				}
				if seed == 0 {
					first = before
				}
			}
			plan, err := choice.ProjectReplayPlan(first.ChoiceTrace, identity)
			if err != nil {
				measureErr = errutil.Join(measureErr, err)
				return
			}
			request.Runtime.Seed = 0
			*request.Runtime.Choice = choiceReplayRequest(request.Runtime.Choice, choice.ModeReplay, first.ChoiceTrace.Bytes, plan)
			replayed, runDiagnosticProfileErr4 := runDiagnosticProfile(runCtx, root, suite.name+"-diagnostic-replay", request, true, true)
			if runDiagnosticProfileErr4 != nil {
				measureErr = errutil.Join(measureErr, runDiagnosticProfileErr4)
				return
			}
			requireDiagnosticEvidenceErr2 := requireDiagnosticEvidence(runCtx, first, replayed)
			if requireDiagnosticEvidenceErr2 != nil {
				measureErr = errutil.Join(measureErr, requireDiagnosticEvidenceErr2)
				return
			}
			return
		})
		if runStepErr != nil {
			measureErr = errutil.Join(measureErr, runStepErr)
			return
		}
		if !runStep2 {
			return
		}
	}
	return
}
func compiledOwnerIdentities() (map[string]string, error) {
	owners := map[string]string{"wasi": wasi.ImplementationSHA256(), "volume": volume.ImplementationSHA256(), "network": netmodel.ImplementationSHA256(), "interrupted_replay": choice.InterruptedReplayImplementationSHA256(), "runner_clock": clock.BrokerImplementationSHA256(), "world_binary_snapshot": world.ImplementationSHA256()}
	for name, identity := range owners {
		if !strings.HasPrefix(identity, "sha256:") || len(identity) != len("sha256:")+64 {
			return nil, fmt.Errorf("missing compiled owner identity: %s=%q", name, identity)
		}
	}
	return owners, nil
}
func measureCompiledIdentityOwners(runCtx context.Context) (measureErr error) {
	owners, err := compiledOwnerIdentities()
	if err != nil {
		measureErr = err
		return
	}
	root := input(runCtx, "GOMAD_WASM_DIAGNOSTIC_IDENTITY_ROOT")
	if root == "" {
		measureErr = fmt.Errorf("%w: %v", ErrRefused, errutil.New("set the identity capture root"))
		return
	}
	if !filepath.IsAbs(root) {
		measureErr = errutil.New("identity capture requires absolute root")
		return
	}
	err = os.MkdirAll(root, 0700)
	if err != nil {
		measureErr = err
		return
	}
	writeReportErr := temporalJSON(runCtx, filepath.Join(root, "compiled-owner-identities.json"), map[string]any{"compiled_go_version": runtime.Version(), "owners": owners})
	if writeReportErr != nil {
		measureErr = writeReportErr
		return
	}
	logf(runCtx, "compiled identity owners: %v", owners)
	return
}
func requireDiagnosticEvidence(runCtx context.Context, before, after wasi.Result) (measureErr error) {
	difference := sameEvidence(before, after)
	if difference != "" || before.Calls != after.Calls || before.TimeAdvances != after.TimeAdvances || before.EnvironmentIdentity != after.EnvironmentIdentity || before.FuelRemaining != after.FuelRemaining {
		measureErr = fmt.Errorf("profiling changed canonical execution: %s", difference)
		return
	}
	return
}
func profilePreparationEvent(runCtx context.Context, channel *os.File, event string, epoch int, request wasi.Request, cold bool) (measureErr error) {
	data, err := json.Marshal(struct {
		Event        string `json:"event"`
		Epoch        int    `json:"epoch"`
		Helper       string `json:"helper"`
		HelperSHA256 string `json:"helper_sha256"`
		Cold         bool   `json:"cold"`
	}{event, epoch, request.HelperPath, request.HelperSHA256, cold})
	if err != nil {
		measureErr = err
		return
	}
	_, err = channel.Write(append(data, '\n'))
	if err != nil {
		measureErr = err
		return
	}
	return
}
func runDiagnosticProfile(runCtx context.Context, root, name string, request wasi.Request, instrumented, warm bool) (measureValue0 wasi.Result, measureErr error) {
	profile, err := os.Create(filepath.Join(root, name+"-parent.pprof"))
	if err != nil {
		measureErr = err
		return
	}
	err = pprof.StartCPUProfile(profile)
	if err != nil {
		closeResource(runCtx, profile)
		measureErr = err
		return
	}
	var memoryBefore, memoryAfter runtime.MemStats
	runtime.ReadMemStats(&memoryBefore)
	started := time.Now()
	result, runErr := matchedCurrentPrepareAdmitted(runCtx, input(runCtx, "GOMAD_WASM_MATCHED_COLD_MEMORY_GUARD"), func() (wasi.Result, error) {
		callbackValue0, callbackValue1, callbackValue2 := (func(runCtx context.Context) (measureValue0 wasi.Result, measureValue1 error, measureErr error) {
			returnedValue0, returnedValue1 := wasi.Run(runCtx, request)
			return returnedValue0, returnedValue1, measureErr
		})(runCtx)
		return callbackValue0, errutil.Join(callbackValue1, callbackValue2)
	})
	wall := time.Since(started).Nanoseconds()
	runtime.ReadMemStats(&memoryAfter)
	pprof.StopCPUProfile()
	err = profile.Close()
	if err != nil {
		measureErr = err
		return
	}
	heap, err := os.Create(filepath.Join(root, name+"-parent-heap.pprof"))
	if err != nil {
		measureErr = err
		return
	}
	err = pprof.WriteHeapProfile(heap)
	if err != nil {
		closeResource(runCtx, heap)
		measureErr = err
		return
	}
	err = heap.Close()
	if err != nil {
		measureErr = err
		return
	}
	writeReportErr := temporalJSON(runCtx, filepath.Join(root, name+"-parent-memory.json"), map[string]any{"diagnostic_only": true, "before": memoryBefore, "after": memoryAfter})
	if writeReportErr != nil {
		measureErr = writeReportErr
		return
	}
	writeEvidenceErr := temporalWrite(runCtx, filepath.Join(root, name+"-helper-stderr.txt"), result.HelperStderr())
	if writeEvidenceErr != nil {
		measureErr = writeEvidenceErr
		return
	}
	writeEvidenceErr2 := temporalWrite(runCtx, filepath.Join(root, name+"-guest-stdout.txt"), result.Stdout)
	if writeEvidenceErr2 != nil {
		measureErr = writeEvidenceErr2
		return
	}
	writeEvidenceErr3 := temporalWrite(runCtx, filepath.Join(root, name+"-guest-stderr.txt"), result.Stderr)
	if writeEvidenceErr3 != nil {
		measureErr = writeEvidenceErr3
		return
	}
	var diagnostic *diagnosticReport
	var diagnosticErr error
	if instrumented {
		var decoded diagnosticReport
		decoded, diagnosticErr = decodeDiagnosticReport(result.HelperStderr())
		if diagnosticErr == nil {
			diagnostic = &decoded
		}
	}
	writeReportErr2 := temporalJSON(runCtx, filepath.Join(root, name+"-metrics.json"), map[string]any{"diagnostic_only": true, "compatibility_profile": "prototype-legacy-v1", "module_sha256": request.ModuleSHA256, "helper_sha256": request.HelperSHA256, "runtime_identity": request.Runtime.Choice.ExecutionIdentity.ToolchainBuildKey, "model_sha256": wasi.ImplementationSHA256(), "instrumented": instrumented, "run_wall_nanos": wall, "authorization_to_terminal_nanos": result.AuthorizationToTerminalNanos, "phase_metrics": result.Telemetry, "wait_resource_usage": result.HelperResourceUsage, "helper_profile": diagnostic, "scope_labels": map[string]string{"run_wall": "Go Run boundary; excludes diagnostics publication and pprof finalization", "phase_cpu": "helper RUSAGE_SELF samples within named phase; no parent CPU", "wait_cpu": "entire helper process including preparation", "nested_wall": "overlaps ABI and phase spans; never sum as disjoint work", "residual_wall": "unattributed Wasmtime/trampoline wall; never guest-only CPU", "peak_rss": "cumulative process high-water, not phase-local allocation"}})
	if writeReportErr2 != nil {
		measureErr = writeReportErr2
		return
	}
	persistDiagnosticRawResultErr := persistDiagnosticRawResult(runCtx, root, name, result)
	if persistDiagnosticRawResultErr != nil {
		measureErr = persistDiagnosticRawResultErr
		return
	}
	if runErr != nil {
		measureErr = errutil.Join(runErr)
		return
	}
	requireExitErr := requireExit(runCtx, name, result, 0)
	if requireExitErr != nil {
		measureErr = requireExitErr
		return
	}
	if diagnosticErr != nil {
		measureErr = errutil.Join(diagnosticErr)
		return
	}
	if !instrumented && len(result.HelperStderr()) != 0 {
		measureErr = errutil.New("release-default helper emitted diagnostic bytes")
		return
	}
	if result.Telemetry == nil || warm && (!result.Telemetry.CacheHit || result.Telemetry.CompileNanos != 0) {
		measureErr = errutil.New("profile phase metrics or warm cache contract")
		return
	}
	return result, measureErr
}
func measureRuntimeCostProfile(runCtx context.Context) (measureErr error) {
	root := input(runCtx, "GOMAD_WASM_PROFILE_ROOT")
	if root == "" {
		measureErr = fmt.Errorf("%w: %v", ErrRefused, errutil.New("set GOMAD_WASM_PROFILE_ROOT for actual runtime-loop profiling"))
		return
	}
	if !filepath.IsAbs(root) {
		measureErr = errutil.New("profile evidence root must be absolute")
		return
	}
	err := os.MkdirAll(root, 0700)
	if err != nil {
		measureErr = err
		return
	}
	request, cooperativeRequestErr := cooperativeRequest(runCtx, "./testdata/runtime_cost")
	if cooperativeRequestErr != nil {
		measureErr = cooperativeRequestErr
		return
	}
	request.Config.Clock.ReadStepNanos = 0
	request.Config.Limits.Calls = 1 << 20
	request.Config.Limits.TranscriptBytes = 64 << 20
	request.Runtime.Diagnostics = false
	request.Runtime.Choice.Limit = 64 << 20
	request.Fuel = 1 << 40
	request.Timeout = 3 * time.Minute
	module, err := os.ReadFile(request.ModulePath)
	if err != nil {
		measureErr = err
		return
	}
	writeEvidenceErr := temporalWrite(runCtx, filepath.Join(root, "runtime-cost.wasm"), module)
	if writeEvidenceErr != nil {
		measureErr = writeEvidenceErr
		return
	}
	var reference wasi.Result
	for _, owner := range []string{"go", "helper"} {
		request.SetRemoteModel(owner == "go")
		profile, err := os.Create(filepath.Join(root, owner+"-parent.pprof"))
		if err != nil {
			measureErr = err
			return
		}
		err = pprof.StartCPUProfile(profile)
		if err != nil {
			closeResource(runCtx, profile)
			measureErr = err
			return
		}
		started := time.Now()
		result, runErr := wasi.Run(runCtx, request)
		wall := time.Since(started).Nanoseconds()
		pprof.StopCPUProfile()
		err = profile.Close()
		if err != nil {
			measureErr = err
			return
		}
		if runErr != nil {
			measureErr = errutil.Join(runErr)
			return
		}
		requireExitErr := requireExit(runCtx, owner, result, 0)
		if requireExitErr != nil {
			measureErr = requireExitErr
			return
		}
		if !bytes.Contains(result.Stdout, []byte("runtime-cost 100000")) || result.AuthorizationToTerminalNanos == 0 || result.ChoiceTrace.Summary.SelectPoll < 100000 {
			measureErr = fmt.Errorf("runtime loop witness missing: stdout=%q summary=%+v boundary=%d", result.Stdout, result.ChoiceTrace.Summary, result.AuthorizationToTerminalNanos)
			return
		}
		if owner == "go" {
			reference = result
		} else {
			difference := sameEvidence(reference, result)
			if difference != "" {
				measureErr = fmt.Errorf("runtime-cost profiling evidence differs: %s", difference)
				return
			}
		}
		writeEvidenceErr2 := temporalWrite(runCtx, filepath.Join(root, owner+"-choice.bin"), result.ChoiceTrace.Bytes)
		if writeEvidenceErr2 != nil {
			measureErr = writeEvidenceErr2
			return
		}
		writeReportErr := temporalJSON(runCtx, filepath.Join(root, owner+"-metrics.json"), map[string]any{"diagnostic_only": true, "profile": "current-runtime-go-owner-reference", "module_sha256": request.ModuleSHA256, "helper_sha256": request.HelperSHA256, "runtime_identity": request.Runtime.Choice.ExecutionIdentity.ToolchainBuildKey, "model_sha256": wasi.ImplementationSHA256(), "wall_nanos": wall, "authorization_to_terminal_nanos": result.AuthorizationToTerminalNanos, "phase_metrics": result.Telemetry, "wait_resource_usage": result.HelperResourceUsage, "choices": result.ChoiceTrace.Summary})
		if writeReportErr != nil {
			measureErr = writeReportErr
			return
		}
	}
	return
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
	// Schema v2 (fn-185): exact per-op and per-owner attribution.
	ABIExcluded         *diagnosticABI              `json:"abi_excluded,omitempty"`
	ABIOps              map[string]diagnosticOp     `json:"abi_ops,omitempty"`
	Owners              map[string]diagnosticSeries `json:"owners,omitempty"`
	Reconciliation      *diagnosticReconciliation   `json:"reconciliation,omitempty"`
	UnattributedImports uint64                      `json:"unattributed_imports"`
	UnframedImports     uint64                      `json:"unframed_imports"`
	SpanErrors          uint64                      `json:"span_errors"`
	Threads             []diagnosticThread          `json:"threads,omitempty"`
	ThreadsOmitted      uint64                      `json:"threads_omitted"`
}

// diagnosticThread is one cluster store thread's own reconciliation row.
type diagnosticThread struct {
	Slot                    uint64 `json:"slot"`
	Incarnation             uint64 `json:"incarnation"`
	ExecuteWallNanos        uint64 `json:"execute_wall_nanos"`
	AttributedHostWallNanos uint64 `json:"attributed_host_wall_nanos"`
	GuestResidualWallNanos  uint64 `json:"guest_residual_wall_nanos"`
	OuterHostWallNanos      uint64 `json:"outer_host_wall_nanos"`
	DifferenceNanos         uint64 `json:"difference_nanos"`
	ToleranceNanos          uint64 `json:"tolerance_nanos"`
}

// diagnosticSeries is one v2 `owners` entry; `abi_ops` entries add served_by.
type diagnosticSeries struct {
	Count                     uint64 `json:"count"`
	WallNanos                 uint64 `json:"wall_nanos"`
	ExclusiveWallNanos        uint64 `json:"exclusive_wall_nanos"`
	InstantiateWallNanos      uint64 `json:"instantiate_wall_nanos"`
	ExecuteWallNanos          uint64 `json:"execute_wall_nanos"`
	ExecuteExclusiveWallNanos uint64 `json:"execute_exclusive_wall_nanos"`
	MinNanos                  uint64 `json:"min_nanos"`
	MedianNanos               uint64 `json:"median_nanos"`
	P99Nanos                  uint64 `json:"p99_nanos"`
	MaxNanos                  uint64 `json:"max_nanos"`
}

type diagnosticOp struct {
	diagnosticSeries
	ServedBy map[string]uint64 `json:"served_by"`
}

type diagnosticReconciliation struct {
	ExecuteWallNanos             uint64 `json:"execute_wall_nanos"`
	AttributedHostWallNanos      uint64 `json:"attributed_host_wall_nanos"`
	GuestResidualWallNanos       uint64 `json:"guest_residual_wall_nanos"`
	OuterHostWallNanos           uint64 `json:"outer_host_wall_nanos"`
	DifferenceNanos              uint64 `json:"difference_nanos"`
	ToleranceNanos               uint64 `json:"tolerance_nanos"`
	DistributionRelativeErrorPPM uint64 `json:"distribution_relative_error_ppm"`
}

const (
	diagnosticReportLimit      = 48 << 10
	diagnosticSeriesFields     = "count wall_nanos exclusive_wall_nanos instantiate_wall_nanos execute_wall_nanos execute_exclusive_wall_nanos min_nanos median_nanos p99_nanos max_nanos"
	diagnosticV1Fields         = "schema diagnostic_only backend host_os host_arch phases abi nested residual abi_overlap_errors"
	diagnosticV2Fields         = diagnosticV1Fields + " abi_excluded abi_ops owners reconciliation threads threads_omitted unattributed_imports unframed_imports span_errors"
	diagnosticThreadFields     = "slot incarnation execute_wall_nanos attributed_host_wall_nanos guest_residual_wall_nanos outer_host_wall_nanos difference_nanos tolerance_nanos"
	diagnosticReconcileFields  = "execute_wall_nanos attributed_host_wall_nanos guest_residual_wall_nanos outer_host_wall_nanos difference_nanos tolerance_nanos distribution_relative_error_ppm"
	diagnosticToleranceFloor   = 50_000
	diagnosticToleranceDivisor = 200
)

// diagnosticOwnerNames is the helper's closed owner set (fn-185).
// instrumentation is the profiler's own per-import cost and handoff_wait a
// cluster instance blocked for the baton: neither is host work.
var diagnosticOwnerNames = strings.Fields("clock scheduler scheduler_handoff namespace volume network world storage_transport cache transcript evidence encoding remote inputs inspection lifecycle instrumentation handoff_wait")

// diagnosticOpNames lists every helper import as `module.op`, independently
// of the helper's operation table.
var diagnosticOpNames = func() map[string]bool {
	names := map[string]bool{}
	for _, name := range strings.Fields("sched_yield proc_exit fd_close fd_sync args_get args_sizes_get environ_get environ_sizes_get random_get fd_filestat_get fd_fdstat_get fd_prestat_get fd_fdstat_set_flags sock_shutdown clock_time_get fd_filestat_set_size fd_write fd_read poll_oneoff path_create_directory path_remove_directory path_unlink_file fd_prestat_dir_name sock_accept fd_pread fd_pwrite fd_readdir fd_seek path_filestat_get path_symlink path_readlink path_rename path_open") {
		names["wasi_snapshot_preview1."+name] = true
	}
	for _, name := range strings.Fields("gomad_wasm_v1.config gomad_wasm_v1.decision gomad_wasm_v1.observation gomad_wasm_v1.finish gomad_wasm_v1.idle gomad_wasm_v2.net_stream gomad_debug_v1.exchange") {
		names[name] = true
	}
	return names
}()

func decodeDiagnosticReport(data []byte) (diagnosticReport, error) {
	var report diagnosticReport
	prefix := []byte("gomad-wasm-profile-v1 ")
	if len(data) > diagnosticReportLimit || !bytes.HasPrefix(data, prefix) || !bytes.HasSuffix(data, []byte{'\n'}) || bytes.Count(data, []byte{'\n'}) != 1 {
		return report, fmt.Errorf("missing, oversized or multiple diagnostic report")
	}
	data = data[len(prefix) : len(data)-1]
	if err := wasi.ValidateMeasurementJSON(data); err != nil {
		return report, err
	}
	var schema struct {
		Schema string `json:"schema"`
	}
	if err := json.Unmarshal(data, &schema); err != nil {
		return report, err
	}
	v2 := schema.Schema == "gomad.wasm-profile/v2"
	keys := diagnosticV1Fields
	if v2 {
		keys = diagnosticV2Fields
	}
	fields, err := diagnosticFields(data, keys)
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
	if v2 {
		if err := decodeDiagnosticAttributionFields(fields); err != nil {
			return report, err
		}
	}
	if err := json.Unmarshal(data, &report); err != nil {
		return report, err
	}
	if v2 {
		if err := checkDiagnosticAttribution(report); err != nil {
			return report, err
		}
	}
	if report.Schema != "gomad.wasm-profile/v1" && !v2 || !report.DiagnosticOnly || report.Backend != "cranelift-jit" || report.HostOS != "linux" && report.HostOS != "macos" || report.HostArch != "aarch64" && report.HostArch != "x86_64" || report.ABIOverlapErrors != 0 {
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

// decodeDiagnosticAttributionFields requires the v2 inventory exactly:
// known op and owner keys with their fixed measurements.
func decodeDiagnosticAttributionFields(fields map[string]json.RawMessage) error {
	if _, err := diagnosticFields(fields["abi_excluded"], "count wall_nanos instantiate_wall_nanos execute_wall_nanos"); err != nil {
		return err
	}
	if _, err := diagnosticFields(fields["reconciliation"], diagnosticReconcileFields); err != nil {
		return err
	}
	var threads []json.RawMessage
	if err := json.Unmarshal(fields["threads"], &threads); err != nil || threads == nil {
		return fmt.Errorf("diagnostic threads: %v", err)
	}
	for _, thread := range threads {
		if _, err := diagnosticFields(thread, diagnosticThreadFields); err != nil {
			return err
		}
	}
	owners := map[string]bool{}
	for _, name := range diagnosticOwnerNames {
		owners[name] = true
	}
	for _, group := range []struct {
		name, measurements string
		known              map[string]bool
	}{{"abi_ops", diagnosticSeriesFields + " served_by", diagnosticOpNames}, {"owners", diagnosticSeriesFields, owners}} {
		var entries map[string]json.RawMessage
		if err := json.Unmarshal(fields[group.name], &entries); err != nil || entries == nil {
			return fmt.Errorf("diagnostic %s: %v", group.name, err)
		}
		for key, entry := range entries {
			if !group.known[key] {
				return fmt.Errorf("diagnostic %s: unknown key %q", group.name, key)
			}
			measured, err := diagnosticFields(entry, group.measurements)
			if err != nil {
				return err
			}
			if group.name != "abi_ops" {
				continue
			}
			var served map[string]json.RawMessage
			if err := json.Unmarshal(measured["served_by"], &served); err != nil || served == nil {
				return fmt.Errorf("diagnostic op %s served_by: %v", key, err)
			}
			for owner := range served {
				if !owners[owner] {
					return fmt.Errorf("diagnostic op %s: unknown owner %q", key, owner)
				}
			}
		}
	}
	return nil
}

// checkDiagnosticAttribution checks the structure of a decoded v2 report:
// per-series bounds and the cross-checks against the v1 buckets. Timing
// tolerance and zero error counters are requireAttributedProfile's.
func checkDiagnosticAttribution(report diagnosticReport) error {
	series := func(kind, name string, entry diagnosticSeries) error {
		if entry.Count == 0 || entry.ExclusiveWallNanos > entry.WallNanos || entry.ExecuteExclusiveWallNanos > entry.ExecuteWallNanos || entry.ExecuteWallNanos > entry.WallNanos || entry.InstantiateWallNanos > entry.WallNanos-entry.ExecuteWallNanos || entry.MinNanos > entry.MedianNanos || entry.MedianNanos > entry.P99Nanos || entry.P99Nanos > entry.MaxNanos {
			return fmt.Errorf("diagnostic %s %s: inconsistent series %+v", kind, name, entry)
		}
		return nil
	}
	var calls, wall, instantiate, execute uint64
	for name, op := range report.ABIOps {
		if err := series("op", name, op.diagnosticSeries); err != nil {
			return err
		}
		var served uint64
		for _, count := range op.ServedBy {
			if count == 0 || count > math.MaxUint64-served {
				return fmt.Errorf("diagnostic op %s: served_by count", name)
			}
			served += count
		}
		if served > op.Count {
			return fmt.Errorf("diagnostic op %s: served_by exceeds calls", name)
		}
		calls, wall, instantiate, execute = calls+op.Count, wall+op.WallNanos, instantiate+op.InstantiateWallNanos, execute+op.ExecuteWallNanos
	}
	for name, owner := range report.Owners {
		if err := series("owner", name, owner); err != nil {
			return err
		}
	}
	legacy := *report.ABIExcluded
	for _, bucket := range report.ABI {
		legacy.Count, legacy.WallNanos, legacy.InstantiateWallNanos, legacy.ExecuteWallNanos = legacy.Count+bucket.Count, legacy.WallNanos+bucket.WallNanos, legacy.InstantiateWallNanos+bucket.InstantiateWallNanos, legacy.ExecuteWallNanos+bucket.ExecuteWallNanos
	}
	if legacy != (diagnosticABI{Count: calls, WallNanos: wall, InstantiateWallNanos: instantiate, ExecuteWallNanos: execute}) {
		return fmt.Errorf("diagnostic abi_ops %d calls/%dns differ from abi buckets %d calls/%dns", calls, wall, legacy.Count, legacy.WallNanos)
	}
	if report.Reconciliation.ExecuteWallNanos != report.Phases["execute"].WallNanos || report.Reconciliation.DistributionRelativeErrorPPM == 0 {
		return fmt.Errorf("diagnostic reconciliation execute wall")
	}
	return nil
}

// requireAttributedProfile is the fn-185 collector gate: every import is
// framed and attributed, owners partition the op wall exactly, and attributed
// host wall plus guest residual reconcile to execute wall within
// max(50 µs, 0.5 % of execute wall), recomputed here.
func requireAttributedProfile(report diagnosticReport) error {
	if report.Schema != "gomad.wasm-profile/v2" || report.Reconciliation == nil {
		return fmt.Errorf("diagnostic profile is not attributed (schema %q)", report.Schema)
	}
	if report.UnattributedImports != 0 || report.UnframedImports != 0 || report.SpanErrors != 0 || report.ABIOverlapErrors != 0 || report.ThreadsOmitted != 0 {
		return fmt.Errorf("diagnostic attribution errors: unattributed=%d unframed=%d span=%d overlap=%d threads_omitted=%d", report.UnattributedImports, report.UnframedImports, report.SpanErrors, report.ABIOverlapErrors, report.ThreadsOmitted)
	}
	for name, op := range report.ABIOps {
		var served uint64
		for owner, count := range op.ServedBy {
			if report.Owners[owner].Count == 0 {
				return fmt.Errorf("diagnostic op %s: serving owner %s is missing", name, owner)
			}
			served += count
		}
		if served != op.Count {
			return fmt.Errorf("diagnostic op %s: %d of %d calls have an owner", name, served, op.Count)
		}
	}
	row := report.Reconciliation
	var attributed, opExecute uint64
	for _, owner := range report.Owners {
		attributed += owner.ExecuteExclusiveWallNanos
	}
	for _, op := range report.ABIOps {
		opExecute += op.ExecuteWallNanos
	}
	// Op frames partition into every owner but instrumentation, which is
	// the boundary glue outside them.
	if attributed != row.AttributedHostWallNanos || attributed-report.Owners["instrumentation"].ExecuteExclusiveWallNanos != opExecute {
		return fmt.Errorf("diagnostic owner wall %dns, reconciliation %dns, op wall %dns", attributed, row.AttributedHostWallNanos, opExecute)
	}
	if err := diagnosticReconciles("", row.ExecuteWallNanos, attributed, row.GuestResidualWallNanos, row.OuterHostWallNanos, row.DifferenceNanos, row.ToleranceNanos); err != nil {
		return err
	}
	// A merged cluster profile also closes per store thread.
	if len(report.Threads) > 64 {
		return fmt.Errorf("diagnostic thread row capacity")
	}
	identities := make(map[[2]uint64]bool, len(report.Threads))
	var totals [4]uint64
	for _, thread := range report.Threads {
		identity := [2]uint64{thread.Slot, thread.Incarnation}
		if identities[identity] {
			return fmt.Errorf("diagnostic duplicate thread row: slot=%d incarnation=%d", thread.Slot, thread.Incarnation)
		}
		identities[identity] = true
		label := fmt.Sprintf("thread slot=%d incarnation=%d ", thread.Slot, thread.Incarnation)
		if err := diagnosticReconciles(label, thread.ExecuteWallNanos, thread.AttributedHostWallNanos, thread.GuestResidualWallNanos, thread.OuterHostWallNanos, thread.DifferenceNanos, thread.ToleranceNanos); err != nil {
			return err
		}
		for index, value := range [4]uint64{thread.ExecuteWallNanos, thread.AttributedHostWallNanos, thread.GuestResidualWallNanos, thread.OuterHostWallNanos} {
			if value > math.MaxUint64-totals[index] {
				return fmt.Errorf("diagnostic thread totals overflow")
			}
			totals[index] += value
		}
	}
	if len(report.Threads) != 0 && totals != [4]uint64{row.ExecuteWallNanos, row.AttributedHostWallNanos, row.GuestResidualWallNanos, row.OuterHostWallNanos} {
		return fmt.Errorf("diagnostic thread totals differ from aggregate reconciliation")
	}
	return nil
}

// diagnosticReconciles recomputes one reconciliation row: guest gaps and
// boundary wall telescope to execute, and execute == guest + Σ owners within
// max(50 µs, 0.5 % of execute wall).
func diagnosticReconciles(label string, execute, attributed, guest, outer, reportedDifference, reportedTolerance uint64) error {
	if guest > math.MaxUint64-outer || guest+outer != execute {
		return fmt.Errorf("diagnostic %sguest residual %dns plus outer host %dns differ from execute %dns", label, guest, outer, execute)
	}
	accounted := attributed + guest
	difference := max(accounted, execute) - min(accounted, execute)
	tolerance := max(uint64(diagnosticToleranceFloor), execute/diagnosticToleranceDivisor)
	if difference != reportedDifference || tolerance != reportedTolerance {
		return fmt.Errorf("diagnostic %sreconciliation row: difference %d/%d tolerance %d/%d", label, reportedDifference, difference, reportedTolerance, tolerance)
	}
	if difference > tolerance {
		return fmt.Errorf("diagnostic %sreconciliation miss: difference=%dns tolerance=%dns", label, difference, tolerance)
	}
	return nil
}

func diagnosticFields(data []byte, keys string) (map[string]json.RawMessage, error) {
	var fields map[string]json.RawMessage
	err := json.Unmarshal(data, &fields)
	if err != nil {
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
		return counts, invalidMeasurement("retained canonical byte bound")
	}
	records, err := wasi.DecodeTranscript(data)
	if err != nil {
		return counts, err
	}
	for _, record := range records {
		var fields map[string]json.RawMessage
		err := json.Unmarshal(record.Input, &fields)
		if err != nil || fields == nil {
			return counts, invalidMeasurement("retained input object")
		}
		fd := "no-root-fd"
		raw, ok := fields["fd"]
		if ok {
			var value uint32
			if bytes.Equal(raw, []byte("null")) || json.Unmarshal(raw, &value) != nil {
				return counts, invalidMeasurement("retained fd type")
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
			err := json.Unmarshal(fields["subscriptions"], &subscriptions)
			if err != nil {
				return counts, invalidMeasurement("retained subscriptions")
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
func measureOperationRetention(runCtx context.Context) (measureErr error) {
	root := input(runCtx, "GOMAD_WASM_OPERATION_COUNT_ROOT")
	if root == "" {
		measureErr = fmt.Errorf("%w: %v", ErrRefused, errutil.New("counts diagnostic requires an exclusive guest slot"))
		return
	}
	control := input(runCtx, "GOMAD_WASM_OPERATION_CONTROL_ROOT")
	module, cache, runtimeIdentity := input(runCtx, "GOMAD_WASM_MATCHED_MODULE"), input(runCtx, "GOMAD_WASM_MATCHED_CACHE"), input(runCtx, "GOMAD_WASM_MATCHED_TOOLCHAIN_ID")
	for _, path := range []string{root, control, module, cache} {
		if !filepath.IsAbs(path) {
			measureErr = errutil.New("counts diagnostic requires absolute paths")
			return
		}
	}
	if runtimeIdentity == "" {
		measureErr = errutil.New("explicit legacy runtime required")
		return
	}
	err := os.MkdirAll(root, 0700)
	if err != nil {
		measureErr = err
		return
	}
	data, err := os.ReadFile(module)
	if err != nil {
		measureErr = err
		return
	}
	imports, pages, binaryInventoryErr := binaryInventory(runCtx, data)
	if binaryInventoryErr != nil {
		measureErr = binaryInventoryErr
		return
	}
	repo, err := absolute(runCtx, "../../../../../..")
	if err != nil {
		measureErr = err
		return
	}
	identity := choice.ExecutionIdentity{TargetSHA256: sha256.Sum256(data), ToolchainBuildKey: runtimeIdentity, GOOS: "wasip1", GOARCH: "wasm", ImplementationSHA256: sha256.Sum256([]byte(runtimeIdentity))}
	for _, suite := range []struct{ name, run string }{{"workflow", "^TestWorkflowTypeEncodingSuite$/^TestPlainASCII$/^Succeeds$"}, {"child", "^TestChildWorkflowSuite$"}, {"timers", "^TestUserTimersTestSuite$"}} {
		for _, seed := range []uint64{0, 1000003} {
			name := suite.name + "-seed" + stringSeed(seed)
			runStep2, runStepErr := runStep(runCtx, name, func(runCtx context.Context) (measureErr error) {
				config, prototypeTemporalConfigErr := prototypeTemporalConfig(runCtx, repo, "/workspace/tests", "-test.run="+suite.run, "-persistenceType=sql", "-persistenceDriver=sqlite")
				if prototypeTemporalConfigErr != nil {
					measureErr = prototypeTemporalConfigErr
					return
				}
				config.Profile, config.Clock.ReadStepNanos = wasi.CooperativeProfile, 0
				helper, helperErr := helperPath(runCtx)
				if helperErr != nil {
					measureErr = helperErr
					return
				}
				helperSHA256, helperSHA256Err := fileHash(runCtx, helper)
				if helperSHA256Err != nil {
					measureErr = helperSHA256Err
					return
				}
				moduleSHA256, moduleSHA256Err := fileHash(runCtx, module)
				if moduleSHA256Err != nil {
					measureErr = moduleSHA256Err
					return
				}
				request := wasi.Request{HelperPath: helper, HelperSHA256: helperSHA256, ModulePath: module, ModuleSHA256: moduleSHA256, Engine: wasi.PinnedEngine(), Imports: imports, InitialMemoryPages: pages, MemoryBytes: 4 << 30, Fuel: 1 << 50, Config: config, Timeout: 5 * time.Minute, CompileCache: cache, DigestTranscript: false, Runtime: &wasi.RuntimeControl{Seed: seed, Choice: &runnerbackend.ChoiceRequest{Mode: choice.ModeRecord, ExecutionIdentity: identity, Limit: choice.MaximumTraceBytes}}}
				reference := suite.name + "-reference-seed" + stringSeed(seed)
				var historical wasi.Result
				diagnosticJSONErr := readDiagnosticJSON(runCtx, filepath.Join(control, reference+"-observed-result.json"), &historical)
				if diagnosticJSONErr != nil {
					measureErr = diagnosticJSONErr
					return
				}
				var metadata struct {
					Helper  string `json:"helper_sha256"`
					Module  string `json:"module_sha256"`
					Model   string `json:"model_sha256"`
					Runtime string `json:"runtime_identity"`
				}
				diagnosticJSONErr2 := readDiagnosticJSON(runCtx, filepath.Join(control, reference+"-metrics.json"), &metadata)
				if diagnosticJSONErr2 != nil {
					measureErr = diagnosticJSONErr2
					return
				}
				if metadata.Helper != request.HelperSHA256 || metadata.Module != request.ModuleSHA256 || metadata.Model != wasi.ImplementationSHA256() || metadata.Runtime != runtimeIdentity {
					measureErr = errutil.New("current control identities mismatch")
					return
				}
				var before, retained wasi.Result
				for _, owner := range []string{"control", "retained"} {
					current := request
					current.DigestTranscript = owner == "control"
					started := time.Now()
					result, runErr := matchedCurrentPrepareAdmitted(runCtx, input(runCtx, "GOMAD_WASM_MATCHED_COLD_MEMORY_GUARD"), func() (wasi.Result, error) {
						callbackValue0, callbackValue1, callbackValue2 := (func(runCtx context.Context) (measureValue0 wasi.Result, measureValue1 error, measureErr error) {
							returnedValue0, returnedValue1 := wasi.Run(runCtx, current)
							return returnedValue0, returnedValue1, measureErr
						})(runCtx)
						return callbackValue0, errutil.Join(callbackValue1, callbackValue2)
					})
					persistOperationDiagnosticErr := persistOperationDiagnostic(runCtx, root, name+"-"+owner, current, result, time.Since(started).Nanoseconds())
					if persistOperationDiagnosticErr != nil {
						measureErr = persistOperationDiagnosticErr
						return
					}
					if runErr != nil {
						measureErr = errutil.Join(runErr)
						return
					}
					requireExitErr := requireExit(runCtx, name+"-"+owner, result, 0)
					if requireExitErr != nil {
						measureErr = requireExitErr
						return
					}
					if result.Telemetry == nil || !result.Telemetry.CacheHit || result.Telemetry.CompileNanos != 0 {
						measureErr = errutil.New("warm diagnostic cache miss; not accepted as warm")
						return
					}
					if owner == "control" {
						before = result
					} else {
						retained = result
					}
				}
				requireDiagnosticEvidenceErr := requireDiagnosticEvidence(runCtx, before, retained)
				if requireDiagnosticEvidenceErr != nil {
					measureErr = requireDiagnosticEvidenceErr
					return
				}
				difference := historicalJSONPublicProjectionDifference(historical, before)
				if difference != "" {
					measureErr = fmt.Errorf("historical JSON PUBLIC projection changed (no historical private-terminal proof): %s", difference)
					return
				}
				counts, err := countCanonicalRemoteOperations(retained.Transcript, 64<<20)
				if err != nil {
					measureErr = err
					return
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
				diagnosticJSONErr3 := readDiagnosticJSON(runCtx, filepath.Join(control, suite.name+"-diagnostic-seed"+stringSeed(seed)+"-metrics.json"), &diagnostic)
				if diagnosticJSONErr3 != nil {
					measureErr = diagnosticJSONErr3
					return
				}
				if counts.Remote != diagnostic.Profile.ABI.Remote.Count {
					measureErr = fmt.Errorf("actual remote classification=%d diagnostic callback count=%d", counts.Remote, diagnostic.Profile.ABI.Remote.Count)
					return
				}
				writeReportErr := temporalJSON(runCtx, filepath.Join(root, name+"-operation-counts.json"), counts)
				if writeReportErr != nil {
					measureErr = writeReportErr
					return
				}
				return
			})
			if runStepErr != nil {
				measureErr = runStepErr
				return
			}
			if !runStep2 {
				return
			}
		}
	}
	return
}
func readDiagnosticJSON(runCtx context.Context, path string, target any) (measureErr error) {
	data, err := hostfs.ReadBounded(path, 64<<20)
	if err != nil {
		measureErr = err
		return
	}
	err = wasi.ValidateMeasurementJSON(data)
	if err != nil {
		measureErr = err
		return
	}
	err = json.Unmarshal(data, target)
	if err != nil {
		measureErr = err
		return
	}
	return
}
func persistOperationDiagnostic(runCtx context.Context, root, name string, request wasi.Request, result wasi.Result, wall int64) (measureErr error) {
	runtimeIdentity := ""
	if request.Runtime != nil && request.Runtime.Choice != nil {
		runtimeIdentity = request.Runtime.Choice.ExecutionIdentity.ToolchainBuildKey
	}
	writeReportErr := temporalJSON(runCtx, filepath.Join(root, name+"-metrics.json"), map[string]any{"diagnostic_only": true, "retention_mode": !request.DigestTranscript, "retained_limit_scope": "64MiB postexecution only; configured transport reservation unchanged", "module_sha256": request.ModuleSHA256, "helper_sha256": request.HelperSHA256, "model_sha256": wasi.ImplementationSHA256(), "runtime_identity": runtimeIdentity, "run_wall_nanos": wall, "authorization_to_terminal_nanos": result.AuthorizationToTerminalNanos, "phase_metrics": result.Telemetry, "wait_resource_usage": result.HelperResourceUsage, "termination": result.Termination, "message": result.Message, "retained_bytes": len(result.Transcript)})
	if writeReportErr != nil {
		measureErr = writeReportErr
		return
	}
	persistDiagnosticRawResultErr := persistDiagnosticRawResult(runCtx, root, name, result)
	if persistDiagnosticRawResultErr != nil {
		measureErr = persistDiagnosticRawResultErr
		return
	}
	return
}
func persistDiagnosticRawResult(runCtx context.Context, root, name string, result wasi.Result) (measureErr error) {
	if len(result.Transcript) > 64<<20 {
		measureErr = errutil.New("postexecution retained transcript capacity exceeded; actual costs retained")
		return
	}
	writeEvidenceErr := temporalWrite(runCtx, filepath.Join(root, name+"-canonical.bin"), result.Transcript)
	if writeEvidenceErr != nil {
		measureErr = writeEvidenceErr
		return
	}
	writeEvidenceErr2 := temporalWrite(runCtx, filepath.Join(root, name+"-choice.bin"), result.ChoiceTrace.Bytes)
	if writeEvidenceErr2 != nil {
		measureErr = writeEvidenceErr2
		return
	}
	writeEvidenceErr3 := temporalWrite(runCtx, filepath.Join(root, name+"-finish.bin"), runtimeTerminalBytes(result)[:])
	if writeEvidenceErr3 != nil {
		measureErr = writeEvidenceErr3
		return
	}
	writeReportErr := temporalJSON(runCtx, filepath.Join(root, name+"-observed-result.json"), result)
	if writeReportErr != nil {
		measureErr = writeReportErr
		return
	}
	return
}
func historicalJSONPublicProjectionDifference(historical, actual wasi.Result) string {
	actual.SetRuntimeTerminal([16]byte{})
	difference := sameEvidence(historical, actual)
	if difference != "" {
		return difference
	}
	if historical.Calls != actual.Calls || historical.TimeAdvances != actual.TimeAdvances || historical.EnvironmentIdentity != actual.EnvironmentIdentity || historical.FuelRemaining != actual.FuelRemaining || historical.VirtualTimeElapsedNanos != actual.VirtualTimeElapsedNanos {
		return "public counters or identity"
	}
	return ""
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
func currentDiagnosticPerformanceHostEvidence(ctx context.Context) (string, bool) {
	_, cpu, err := warmPreparationHostEvidence(ctx)
	return cpu, err == nil
}
func matchedCurrentPrepareAdmitted(ctx context.Context, mode string, dispatch func() (wasi.Result, error)) (wasi.Result, error) {
	if runtime.GOOS == "linux" {
		return matchedPrepareAdmitted(runtime.GOOS, "/proc/meminfo", mode, dispatch)
	}
	if mode == "" {
		return dispatch()
	}
	if mode != "1" {
		return wasi.Result{}, errutil.New("cold-memory guard must be explicitly enabled with 1")
	}
	return warmPreparationAdmitted(ctx, dispatch)
}

type variantEvidence struct {
	Choice, Network, Stdout, Stderr []byte
	WASI                            []wasi.TranscriptRecord
	Args, Environment               []string
	EntropyKey                      [32]byte
	Clock                           wasi.ClockPolicy
	Termination, Message            string
	ExitCode                        *uint32
	VirtualNanos                    uint64
}

func projectVariantEvidence(result wasi.Result, config wasi.Config, gcVariant string) (variantEvidence, error) {
	transcript, err := wasi.DecodeTranscript(result.Transcript)
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
func variantIdentities(request wasi.Request, result wasi.Result) map[string]string {
	engine := request.Engine.Configuration
	if request.DiagnosticConfigurationRevision() != "" {
		engine = request.DiagnosticConfigurationRevision()
	}
	identities := map[string]string{"module_sha256": request.ModuleSHA256, "helper_sha256": request.HelperSHA256, "environment_identity": result.EnvironmentIdentity, "engine_revision": engine, "model_sha256": wasi.ImplementationSHA256()}
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
	err := os.MkdirAll(base, 0700)
	if err != nil {
		return "", err
	}
	return os.MkdirTemp(base, prefix)
}
func measureGuestVariants(runCtx context.Context) (measureErr error) {
	root := input(runCtx, "GOMAD_WASM_GUEST_PROFILE_ROOT")
	if root == "" {
		measureErr = errutil.Join(measureErr, fmt.Errorf("%w: %v", ErrRefused, errutil.New("set GOMAD_WASM_GUEST_PROFILE_ROOT for unsampled Temporal variant measurements")))
		return
	}
	helper, fuelHelper := input(runCtx, "GOMAD_WASM_TEST_HELPER"), input(runCtx, "GOMAD_WASM_FUEL_OFF_HELPER")
	for _, path := range []string{root, helper, fuelHelper} {
		if !filepath.IsAbs(path) {
			measureErr = errutil.Join(measureErr, errutil.New("variant root, diagnostic-profile helper and diagnostic-fuel-off helper must be absolute"))
			return
		}
	}
	helperSHA256, helperSHA256Err := fileHash(runCtx, helper)
	if helperSHA256Err != nil {
		measureErr = errutil.Join(measureErr, helperSHA256Err)
		return
	}
	fuelHelperSHA256, fuelHelperSHA256Err := fileHash(runCtx, fuelHelper)
	if fuelHelperSHA256Err != nil {
		measureErr = errutil.Join(measureErr, fuelHelperSHA256Err)
		return
	}
	if helperSHA256 == fuelHelperSHA256 {
		measureErr = errutil.Join(measureErr, errutil.New("fuel-off helper must have a distinct identity"))
		return
	}
	repo, err := absolute(runCtx, "../../../../../..")
	if err != nil {
		measureErr = errutil.Join(measureErr, err)
		return
	}
	relative, err := filepath.Rel(repo, root)
	if err != nil || relative != ".." && !strings.HasPrefix(relative, ".."+string(os.PathSeparator)) {
		measureErr = errutil.Join(measureErr, errutil.New("raw variant evidence must be retained outside the repository"))
		return
	}
	baseRoot := root
	root, err = diagnosticRunRoot(baseRoot, "guest-variants-")
	if err != nil {
		measureErr = errutil.Join(measureErr, err)
		return
	}
	logf(runCtx, "retaining variant evidence in %s", root)
	admission, diagnosticAdmissionErr := newDiagnosticAdmission(runCtx, repo, root)
	if diagnosticAdmissionErr != nil {
		measureErr = errutil.Join(measureErr, diagnosticAdmissionErr)
		return
	}
	addHelperErr := admission.addHelper(runCtx, helper, input(runCtx, "GOMAD_WASM_DIAGNOSTIC_HELPER_ADMISSION"), []string{"diagnostic-profile"})
	if addHelperErr != nil {
		measureErr = errutil.Join(measureErr, addHelperErr)
		return
	}
	addHelperErr2 := admission.addHelper(runCtx, fuelHelper, input(runCtx, "GOMAD_WASM_FUEL_OFF_HELPER_ADMISSION"), []string{"diagnostic-fuel-off", "diagnostic-profile"})
	if addHelperErr2 != nil {
		measureErr = errutil.Join(measureErr, addHelperErr2)
		return
	}
	setInput(runCtx, "GOMAD_WASM_PROTOTYPE_ROOT", root)
	for _, directory := range []string{"modules", "variants"} {
		err := os.MkdirAll(filepath.Join(root, directory), 0700)
		if err != nil {
			measureErr = errutil.Join(measureErr, err)
			return
		}
	}
	module, optimized := filepath.Join(root, "modules", "variants-baseline.wasm"), filepath.Join(root, "modules", "variants-satconv-signext.wasm")
	for _, build := range []struct{ module, gowasm string }{{module, ""}, {optimized, "satconv,signext"}} {
		moduleErr := admission.buildModule(runCtx, build.module, build.gowasm)
		if moduleErr != nil {
			measureErr = errutil.Join(measureErr, moduleErr)
			return
		}
	}
	setInput(runCtx, "GOWASM", "")
	commit, err := exec.CommandContext(runCtx, "git", "-C", repo, "rev-parse", "HEAD").Output()
	if err != nil {
		measureErr = errutil.Join(measureErr, err)
		return
	}
	status, err := exec.CommandContext(runCtx, "git", "-C", repo, "status", "--porcelain").Output()
	if err != nil {
		measureErr = errutil.Join(measureErr, err)
		return
	}
	seeds := []uint64{0, 1000003, 2000006}
	const repetitions = 5
	variants := []struct{ name, gc, module, helper, revision, followup string }{{"baseline", "", module, helper, "", "none; diagnostic control"}, {"gc200", "200", module, helper, "", "explicit GC environment and memory policy decision; requalify affected selections"}, {"gc400", "400", module, helper, "", "explicit GC environment and memory policy decision; requalify affected selections"}, {"gc-off", "off", module, helper, "", "bounded GOMEMLIMIT policy and GC decision; requalify affected selections"}, {"gowasm", "", optimized, helper, "", "production preparation currently forces GOWASM=; adopting requires preparation and build identity changes plus requalification"}, {"fuel-off", "", module, fuelHelper, wasi.PinnedEngine().Configuration + "-fuel-off", "explicit fuel/resource containment and engine identity decision; diagnostic helper is not a release configuration"}}
	evidence := filepath.Join(root, "variants")
	runAdmitted := func(runCtx context.Context, name string, request wasi.Request, warm bool) (measureValue0 wasi.Result, measureValue1 diagnosticReport, measureValue2 string, measureErr error) {
		checkErr := admission.check(runCtx, evidence, name+"-before", request, true)
		if checkErr != nil {
			measureErr = errutil.Join(measureErr, checkErr)
			return
		}
		defer func() {
			measureErr = errutil.Join(measureErr, admission.check(runCtx, evidence, name+"-after", request, false))
		}()
		runGuestVariant2, runGuestVariant22, runGuestVariant3, runGuestVariantErr := runGuestVariant(runCtx, evidence, name, request, warm)
		if runGuestVariantErr != nil {
			measureErr = errutil.Join(measureErr, runGuestVariantErr)
			return
		}
		return runGuestVariant2, runGuestVariant22, runGuestVariant3, measureErr
	}
	var reports, recommendations []map[string]any
	for _, selection := range []struct{ name, run string }{{"workflow", "^TestWorkflowTypeEncodingSuite$/^TestPlainASCII$/^Succeeds$"}, {"activity", "^TestActivityTestSuite$"}} {
		requests := make([]wasi.Request, len(variants))
		warmFailures := make([]string, len(variants))
		for index, variant := range variants {
			config, prototypeTemporalConfigErr := prototypeTemporalConfig(runCtx, repo, "/workspace/tests", "-test.run="+selection.run, "-persistenceType=sql", "-persistenceDriver=sqlite")
			if prototypeTemporalConfigErr != nil {
				measureErr = errutil.Join(measureErr, prototypeTemporalConfigErr)
				return
			}
			request, requestErr := prototypeRequest(runCtx, variant.module, config, 0, 30*time.Minute)
			if requestErr != nil {
				measureErr = errutil.Join(measureErr, requestErr)
				return
			}
			helperSHA2562, helperSHA256Err2 := fileHash(runCtx, variant.helper)
			if helperSHA256Err2 != nil {
				measureErr = errutil.Join(measureErr, helperSHA256Err2)
				return
			}
			request.HelperPath, request.HelperSHA256 = variant.helper, helperSHA2562
			request.SetDiagnosticConfigurationRevision(variant.revision)
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
			runAdmitted2, runAdmitted22, runAdmitted3, runAdmittedErr := runAdmitted(runCtx, selection.name+"-"+variant.name+"-prepare", warm, false)
			if runAdmittedErr != nil {
				measureErr = errutil.Join(measureErr, runAdmittedErr)
				return
			}
			_, _, warmFailures[index] = runAdmitted2, runAdmitted22, runAdmitted3
		}
		decisions := make([][]string, len(variants))
		for _, seed := range seeds {
			execute, residual := make([][]uint64, len(variants)), make([][]uint64, len(variants))
			failed, identical := make([]bool, len(variants)), make([]bool, len(variants))
			for index := range identical {
				identical[index] = true
			}
			var reference variantEvidence
			var baselineResult wasi.Result
			var baselineSeen bool
			var runs []map[string]any
			repetition := 0
			for ; repetition < repetitions; repetition++ {
				for slot := range variants {
					index := 0
					if slot > 0 {
						index = 1 + (slot-1+repetition)%(len(variants)-1)
					}
					variant, request := variants[index], requests[index]
					request.Runtime.Seed = seed
					name := fmt.Sprintf("%s-%s-seed%d-rep%d", selection.name, variant.name, seed, repetition)
					result, profile, failure, runAdmittedErr2 := runAdmitted(runCtx, name, request, true)
					if runAdmittedErr2 != nil {
						measureErr = errutil.Join(measureErr, runAdmittedErr2)
						return
					}
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
									measureErr = errutil.Join(measureErr, errutil.Join(projectionErr))
									return
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
								measureErr = errutil.Join(measureErr, err)
								return
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
			writeReportErr := temporalJSON(runCtx, filepath.Join(evidence, fmt.Sprintf("%s-seed%d-runs.json", selection.name, seed)), runs)
			if writeReportErr != nil {
				measureErr = errutil.Join(measureErr, writeReportErr)
				return
			}
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
	cpuinfo, requiredHostMeasured := currentDiagnosticPerformanceHostEvidence(runCtx)
	writeReportErr2 := temporalJSON(runCtx, filepath.Join(root, "guest-variants-report.json"), map[string]any{"host_admission": admission.Receipt, "host_admission_sha256": admission.Inputs[admission.Receipt], "module_build_receipts": admission.ModuleReceipts, "helper_admissions": map[string]string{helper: admission.Helpers[helper].Receipt, fuelHelper: admission.Helpers[fuelHelper].Receipt}, "diagnostic_only": true, "source_commit": strings.TrimSpace(string(commit)), "source_dirty": len(status) != 0, "host_goos": runtime.GOOS, "host_goarch": runtime.GOARCH, "go_version": runtime.Version(), "seeds": seeds, "warm_repetitions": repetitions, "required_host": runtime.GOOS + "/" + runtime.GOARCH, "required_host_measured": requiredHostMeasured, "developmental": !requiredHostMeasured, "host_cpuinfo": cpuinfo, "measurements": reports, "recommendations": recommendations, "build_settings": map[string]string{"GOOS": "wasip1", "GOARCH": "wasm", "GOEXPERIMENT": "nogreenteagc", "baseline_GOWASM": "", "variant_GOWASM": "satconv,signext"}, "scope": "unsampled diagnostic-profile timing; full transcript retention for exact behavior comparisons; residual includes guest plus Wasmtime/trampoline work, not guest-only CPU", "equality": "exact choice, WASI, network, stdout/stderr, outcome and virtual time; only GC environ reply outputs masked; module/environment/helper/engine identities separate", "decision_rule": "adopt only when all seeds have identical successful evidence and median execute gain strictly exceeds both baseline and candidate run-to-run ranges; adoption remains a follow-up decision", "rerun": rerunRecord(runCtx, "go", repo, "guest-variants", map[string]string{"GOMAD_WASM_GUEST_PROFILE_ROOT": baseRoot, "GOMAD_WASM_TEST_HELPER": helper, "GOMAD_WASM_FUEL_OFF_HELPER": fuelHelper, "GOMAD_WASM_DIAGNOSTIC_HELPER_ADMISSION": admission.Helpers[helper].Receipt, "GOMAD_WASM_FUEL_OFF_HELPER_ADMISSION": admission.Helpers[fuelHelper].Receipt, "GOMAD_STOCK_GO": input(runCtx, "GOMAD_STOCK_GO")}, true)})
	if writeReportErr2 != nil {
		measureErr = errutil.Join(measureErr, writeReportErr2)
		return
	}
	return
}
func runGuestVariant(runCtx context.Context, root, name string, request wasi.Request, warm bool) (measureValue0 wasi.Result, measureValue1 diagnosticReport, measureValue2 string, measureErr error) {
	result, runErr := matchedCurrentPrepareAdmitted(runCtx, input(runCtx, "GOMAD_WASM_MATCHED_COLD_MEMORY_GUARD"), func() (wasi.Result, error) {
		callbackValue0, callbackValue1, callbackValue2 := (func(runCtx context.Context) (measureValue0 wasi.Result, measureValue1 error, measureErr error) {
			returnedValue0, returnedValue1 := wasi.Run(runCtx, request)
			return returnedValue0, returnedValue1, measureErr
		})(runCtx)
		return callbackValue0, errutil.Join(callbackValue1, callbackValue2)
	})
	writeEvidenceErr := temporalWrite(runCtx, filepath.Join(root, name+"-canonical.bin"), result.Transcript)
	if writeEvidenceErr != nil {
		measureErr = writeEvidenceErr
		return
	}
	writeEvidenceErr2 := temporalWrite(runCtx, filepath.Join(root, name+"-choice.bin"), result.ChoiceTrace.Bytes)
	if writeEvidenceErr2 != nil {
		measureErr = writeEvidenceErr2
		return
	}
	writeEvidenceErr3 := temporalWrite(runCtx, filepath.Join(root, name+"-network.jsonl"), result.NetworkTranscript)
	if writeEvidenceErr3 != nil {
		measureErr = writeEvidenceErr3
		return
	}
	writeEvidenceErr4 := temporalWrite(runCtx, filepath.Join(root, name+"-helper-stderr.txt"), result.HelperStderr())
	if writeEvidenceErr4 != nil {
		measureErr = writeEvidenceErr4
		return
	}
	writeEvidenceErr5 := temporalWrite(runCtx, filepath.Join(root, name+"-guest-stdout.txt"), result.Stdout)
	if writeEvidenceErr5 != nil {
		measureErr = writeEvidenceErr5
		return
	}
	writeEvidenceErr6 := temporalWrite(runCtx, filepath.Join(root, name+"-guest-stderr.txt"), result.Stderr)
	if writeEvidenceErr6 != nil {
		measureErr = writeEvidenceErr6
		return
	}
	profile, profileErr := decodeDiagnosticReport(result.HelperStderr())
	if runErr != nil {
		return result, profile, runErr.Error(), measureErr
	}
	if result.Termination != "exit" || result.ExitCode == nil || *result.ExitCode != 0 {
		return result, profile, fmt.Sprintf("termination=%s exit=%v message=%s", result.Termination, result.ExitCode, result.Message), measureErr
	}
	if profileErr != nil {
		return result, profile, profileErr.Error(), measureErr
	}
	if result.Telemetry == nil || warm && (!result.Telemetry.CacheHit || result.Telemetry.CompileNanos != 0) {
		return result, profile, "missing telemetry or warm cache miss", measureErr
	}
	return result, profile, "", measureErr
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

var guestMangledModulePath = regexp.MustCompile(`^[a-z0-9-]+(\.[a-z0-9-]+)+[/_]`)
var guestStdlibTopLevel = map[string]bool{}

func guestThirdPartyFrame(name string) bool {
	label, _, _ := strings.Cut(name, ".")
	return guestMangledModulePath.MatchString(name) && !guestStdlibTopLevel[label]
}
func guestFrameBucket(name string) string {
	name = strings.TrimPrefix(name, "$")
	for _, prefix := range strings.Fields("runtime.gcBgMarkWorker runtime.gcDrain runtime.scanobject runtime.markroot runtime.gcAssistAlloc runtime.bgsweep runtime.sweepone runtime.wbBufFlush runtime.gcWriteBarrier runtime.bulkBarrier runtime.writebarrier runtime.greyobject runtime.findobject runtime.trydefertospanscan runtime.scanblock runtime.scanstack runtime.scanframeworker") {
		if strings.HasPrefix(strings.ToLower(name), strings.ToLower(prefix)) {
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
	for _, separator := range []string{"/", "_"} {
		if strings.HasPrefix(name, "go.temporal.io"+separator+"server"+separator+"tests.") || strings.HasPrefix(name, "go.temporal.io"+separator+"server"+separator+"tests"+separator) {
			return "test"
		}
		if strings.HasPrefix(name, "go.temporal.io"+separator+"server"+separator) {
			return "server"
		}
	}
	if strings.HasPrefix(name, "runtime.") || strings.HasPrefix(name, "runtime/") || strings.HasPrefix(name, "internal/runtime/") || strings.HasPrefix(name, "internal_runtime_") {
		return "runtime"
	}
	if guestThirdPartyFrame(name) {
		return ""
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
		bucket := guestFrameBucket(name)
		if bucket != "" {
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
	err := json.Unmarshal(data, &profile)
	if err != nil {
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
			depth := 0
			for ; stack != nil; depth++ {
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
func measureGuestProfile(runCtx context.Context) (measureErr error) {
	root := input(runCtx, "GOMAD_WASM_GUEST_PROFILE_ROOT")
	if root == "" {
		measureErr = errutil.Join(measureErr, fmt.Errorf("%w: %v", ErrRefused, errutil.New("set GOMAD_WASM_GUEST_PROFILE_ROOT for Temporal guest stack profiling")))
		return
	}
	if !filepath.IsAbs(root) || !filepath.IsAbs(input(runCtx, "GOMAD_WASM_TEST_HELPER")) {
		measureErr = errutil.Join(measureErr, errutil.New("guest profile root and GOMAD_WASM_TEST_HELPER must be absolute"))
		return
	}
	repo, err := absolute(runCtx, "../../../../../..")
	if err != nil {
		measureErr = errutil.Join(measureErr, err)
		return
	}
	relative, err := filepath.Rel(repo, root)
	if err != nil || relative != ".." && !strings.HasPrefix(relative, ".."+string(os.PathSeparator)) {
		measureErr = errutil.Join(measureErr, errutil.New("raw guest profiles must be retained outside the repository"))
		return
	}
	baseRoot := root
	root, err = diagnosticRunRoot(baseRoot, "guest-profile-")
	if err != nil {
		measureErr = errutil.Join(measureErr, err)
		return
	}
	logf(runCtx, "retaining guest profiles in %s", root)
	admission, diagnosticAdmissionErr := newDiagnosticAdmission(runCtx, repo, root)
	if diagnosticAdmissionErr != nil {
		measureErr = errutil.Join(measureErr, diagnosticAdmissionErr)
		return
	}
	helper := input(runCtx, "GOMAD_WASM_TEST_HELPER")
	addHelperErr := admission.addHelper(runCtx, helper, input(runCtx, "GOMAD_WASM_GUEST_HELPER_ADMISSION"), []string{"diagnostic-profile", "guest-profile"})
	if addHelperErr != nil {
		measureErr = errutil.Join(measureErr, addHelperErr)
		return
	}
	setInput(runCtx, "GOMAD_WASM_PROTOTYPE_ROOT", root)
	for _, directory := range []string{"modules", "evidence"} {
		err := os.MkdirAll(filepath.Join(root, directory), 0700)
		if err != nil {
			measureErr = errutil.Join(measureErr, err)
			return
		}
	}
	module := filepath.Join(root, "modules", "tests.wasm")
	moduleErr := admission.buildModule(runCtx, module, "")
	if moduleErr != nil {
		measureErr = errutil.Join(measureErr, moduleErr)
		return
	}
	evidence := filepath.Join(root, "evidence")
	runAdmitted := func(runCtx context.Context, name string, request wasi.Request, warm bool) (measureValue0 wasi.Result, measureErr error) {
		checkErr := admission.check(runCtx, evidence, name+"-before", request, true)
		if checkErr != nil {
			measureErr = errutil.Join(measureErr, checkErr)
			return
		}
		defer func() {
			measureErr = errutil.Join(measureErr, admission.check(runCtx, evidence, name+"-after", request, false))
		}()
		runDiagnosticProfile2, runDiagnosticProfileErr := runDiagnosticProfile(runCtx, evidence, name, request, true, warm)
		if runDiagnosticProfileErr != nil {
			measureErr = errutil.Join(measureErr, runDiagnosticProfileErr)
			return
		}
		return runDiagnosticProfile2, measureErr
	}
	persistence := []string{"-persistenceType=sql", "-persistenceDriver=sqlite"}
	prototypeTemporalConfig2, prototypeTemporalConfigErr := prototypeTemporalConfig(runCtx, repo, "/workspace/tests", append([]string{"-test.run=^$"}, persistence...)...)
	if prototypeTemporalConfigErr != nil {
		measureErr = errutil.Join(measureErr, prototypeTemporalConfigErr)
		return
	}
	warm, requestErr := prototypeRequest(runCtx, module, prototypeTemporalConfig2, 0, 30*time.Minute)
	if requestErr != nil {
		measureErr = errutil.Join(measureErr, requestErr)
		return
	}
	warm.SetDiagnosticConfigurationRevision(wasi.PinnedEngine().Configuration + "-guest-profile")
	runAdmitted2, runAdmittedErr := runAdmitted(runCtx, "guest-profile-prepare", warm, false)
	if runAdmittedErr != nil {
		measureErr = errutil.Join(measureErr, runAdmittedErr)
		return
	}
	_ = runAdmitted2
	commit, err := exec.CommandContext(runCtx, "git", "-C", repo, "rev-parse", "HEAD").Output()
	if err != nil {
		measureErr = errutil.Join(measureErr, err)
		return
	}
	status, err := exec.CommandContext(runCtx, "git", "-C", repo, "status", "--porcelain").Output()
	if err != nil {
		measureErr = errutil.Join(measureErr, err)
		return
	}
	var runs []map[string]any
	for _, selection := range []struct{ name, run string }{{"workflow", "^TestWorkflowTypeEncodingSuite$/^TestPlainASCII$/^Succeeds$"}, {"activity", "^TestActivityTestSuite$"}} {
		for _, seed := range []uint64{0, 1000003, 2000006} {
			name := selection.name + "-seed" + stringSeed(seed)
			config, prototypeTemporalConfigErr2 := prototypeTemporalConfig(runCtx, repo, "/workspace/tests", append([]string{"-test.run=" + selection.run}, persistence...)...)
			if prototypeTemporalConfigErr2 != nil {
				measureErr = errutil.Join(measureErr, prototypeTemporalConfigErr2)
				return
			}
			request, requestErr2 := prototypeRequest(runCtx, module, config, seed, 30*time.Minute)
			if requestErr2 != nil {
				measureErr = errutil.Join(measureErr, requestErr2)
				return
			}
			request.SetDiagnosticConfigurationRevision(warm.DiagnosticConfigurationRevision())
			before, runAdmittedErr2 := runAdmitted(runCtx, name+"-unsampled", request, true)
			if runAdmittedErr2 != nil {
				measureErr = errutil.Join(measureErr, runAdmittedErr2)
				return
			}
			request.SetGuestProfile(&wasi.GuestProfileConfig{Path: filepath.Join(evidence, name+"-guest.firefox.json"), IntervalMicros: 1000, MaxSamples: 10000})
			after, runAdmittedErr3 := runAdmitted(runCtx, name+"-sampled", request, true)
			if runAdmittedErr3 != nil {
				measureErr = errutil.Join(measureErr, runAdmittedErr3)
				return
			}
			requireDiagnosticEvidenceErr := requireDiagnosticEvidence(runCtx, before, after)
			if requireDiagnosticEvidenceErr != nil {
				measureErr = errutil.Join(measureErr, requireDiagnosticEvidenceErr)
				return
			}
			if before.VirtualTimeElapsedNanos != after.VirtualTimeElapsedNanos || !bytes.Equal(before.DiagnosticTrace.Bytes, after.DiagnosticTrace.Bytes) {
				measureErr = errutil.Join(measureErr, errutil.New("profiling changed virtual time or runtime diagnostics"))
				return
			}
			data, err := os.ReadFile(request.GuestProfile().Path)
			if err != nil {
				measureErr = errutil.Join(measureErr, err)
				return
			}
			profile, attribution, err := decodeGuestAttribution(data)
			if err != nil {
				measureErr = errutil.Join(measureErr, err)
				return
			}
			if attribution.GuestWeight == 0 || request.GuestProfile().MaxSamples != profile.Meta.Guest.MaxSamples || request.GuestProfile().IntervalMicros != profile.Meta.Guest.IntervalMicros || profile.Meta.Guest.Samples > profile.Meta.Guest.MaxSamples || profile.Meta.Guest.Samples != attribution.SampleRows {
				measureErr = errutil.Join(measureErr, fmt.Errorf("profile samples or metadata mismatch: metadata=%+v attribution=%+v", profile.Meta.Guest, attribution))
				return
			}
			run := map[string]any{"selection": selection.name, "test_run": selection.run, "seed": seed, "helper_sha256": request.HelperSHA256, "module_sha256": request.ModuleSHA256, "engine_revision": request.DiagnosticConfigurationRevision(), "runtime_identity": request.Runtime.Choice.ExecutionIdentity.ToolchainBuildKey, "model_sha256": wasi.ImplementationSHA256(), "evidence_identical": true, "profile": request.GuestProfile().Path, "profile_sha256": digest(data), "sampling": profile.Meta.Guest, "attribution": attribution}
			writeReportErr := temporalJSON(runCtx, filepath.Join(evidence, name+"-attribution.json"), run)
			if writeReportErr != nil {
				measureErr = errutil.Join(measureErr, writeReportErr)
				return
			}
			runs = append(runs, run)
			logf(runCtx, "%s: weighted guest samples=%d buckets=%v truncated=%t", name, attribution.GuestWeight, attribution.Buckets, profile.Meta.Guest.Truncated)
		}
	}
	cpuinfo, requiredHostMeasured := currentDiagnosticPerformanceHostEvidence(runCtx)
	writeReportErr2 := temporalJSON(runCtx, filepath.Join(root, "guest-attribution-report.json"), map[string]any{"host_admission": admission.Receipt, "host_admission_sha256": admission.Inputs[admission.Receipt], "module_build_receipts": admission.ModuleReceipts, "helper_admission": admission.Helpers[helper].Receipt, "diagnostic_only": true, "source_commit": strings.TrimSpace(string(commit)), "source_dirty": len(status) != 0, "host_goos": runtime.GOOS, "host_goarch": runtime.GOARCH, "go_version": runtime.Version(), "runs": runs, "required_host": runtime.GOOS + "/" + runtime.GOARCH, "required_host_measured": requiredHostMeasured, "developmental": !requiredHostMeasured, "host_cpuinfo": cpuinfo, "scope": "weighted guest stack samples; epoch function-entry/loop bias; host-call ticks excluded; no timestamp-gap charge; sampled wall times are not baselines", "limits": "scheduler-frame share does not quantify all unwind/rewind work; fuel checks are distributed across ordinary guest frames", "rerun": rerunRecord(runCtx, "go", repo, "guest-profile", map[string]string{"GOMAD_WASM_GUEST_PROFILE_ROOT": baseRoot, "GOMAD_WASM_TEST_HELPER": input(runCtx, "GOMAD_WASM_TEST_HELPER"), "GOMAD_WASM_GUEST_HELPER_ADMISSION": admission.Helpers[helper].Receipt, "GOMAD_STOCK_GO": input(runCtx, "GOMAD_STOCK_GO")}, true)})
	if writeReportErr2 != nil {
		measureErr = errutil.Join(measureErr, writeReportErr2)
		return
	}
	return
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

func terminalExitStageHelper(runCtx context.Context, artifact, helper, artifactSHA256 string) (measureErr error) {
	info, err := os.Lstat(artifact)
	if err != nil || !info.Mode().IsRegular() || info.Size() <= 0 || info.Size() > 512<<20 {
		measureErr = errutil.Join(measureErr, errutil.New(fmt.Sprint("Cargo artifact is not a bounded regular executable", err)))
		return
	}
	input, err := os.Open(artifact)
	if err != nil {
		measureErr = errutil.Join(measureErr, err)
		return
	}
	data, readErr := io.ReadAll(io.LimitReader(input, (512<<20)+1))
	err = errutil.Join(readErr, input.Close())
	if err != nil {
		measureErr = errutil.Join(measureErr, err)
		return
	}
	if len(data) > 512<<20 || digest(data) != artifactSHA256 {
		measureErr = errutil.Join(measureErr, errutil.New("cargo artifact changed before staging"))
		return
	}
	staged, err := hostfs.StageContext(runCtx, filepath.Dir(helper), ".helper-*", 0755, func(_ context.Context, writer io.Writer) error {
		_, err := io.Copy(writer, bytes.NewReader(data))
		return err
	})
	if err != nil {
		measureErr = errutil.Join(measureErr, err)
		return
	}
	addCleanup(runCtx, func(runCtx context.Context) (measureErr error) {
		err := staged.Cleanup()
		if err != nil {
			measureErr = errutil.Join(measureErr, err)
		}
		return
	})
	digest2, digestErr := fileHash(runCtx, staged.Path())
	if digestErr != nil {
		measureErr = errutil.Join(measureErr, digestErr)
		return
	}
	if digest2 != artifactSHA256 {
		measureErr = errutil.Join(measureErr, errutil.New("staged helper differs from the Cargo artifact"))
		return
	}
	published, err := staged.PublishNoReplaceContext(runCtx, helper)
	if err != nil || !published {
		measureErr = errutil.Join(measureErr, errutil.New(fmt.Sprint("publish admitted helper", err)))
		return
	}
	file, _, err := hostfs.OpenPath(helper)
	if err != nil {
		measureErr = errutil.Join(measureErr, err)
		return
	}
	err = file.Close()
	if err != nil {
		measureErr = errutil.Join(measureErr, err)
		return
	}
	artifactSHA2562, artifactSHA256Err := fileHash(runCtx, artifact)
	if artifactSHA256Err != nil {
		measureErr = errutil.Join(measureErr, artifactSHA256Err)
		return
	}
	helperSHA256, helperSHA256Err := fileHash(runCtx, helper)
	if helperSHA256Err != nil {
		measureErr = errutil.Join(measureErr, helperSHA256Err)
		return
	}
	if artifactSHA2562 != artifactSHA256 || helperSHA256 != artifactSHA256 {
		measureErr = errutil.Join(measureErr, errutil.New("cargo artifact or admitted helper changed during publication"))
		return
	}
	return
}

type terminalExitHelperAdmission struct {
	BuildReceipt, BuildReceiptSHA256   string
	CargoArtifact, CargoArtifactSHA256 string
	Helper, HelperSHA256               string
}

func measureTerminalExitHelperStage(runCtx context.Context) (measureErr error) {
	root := input(runCtx, "GOMAD_WASM_TERMINAL_EXIT_HELPER_STAGE_ROOT")
	if root == "" {
		measureErr = fmt.Errorf("%w: %v", ErrRefused, errutil.New("set GOMAD_WASM_TERMINAL_EXIT_HELPER_STAGE_ROOT to admit a retained Cargo helper build"))
		return
	}
	receipt := input(runCtx, "GOMAD_WASM_TERMINAL_EXIT_HELPER_BUILD_RECEIPT")
	if !filepath.IsAbs(root) || !filepath.IsAbs(receipt) {
		measureErr = errutil.New("absolute staging root and retained raw helper-build receipt are required")
		return
	}
	helper, admission, helperReceiptErr := terminalExitAdmitHelper(runCtx, root, receipt)
	if helperReceiptErr != nil {
		measureErr = helperReceiptErr
		return
	}
	helperSHA256, helperSHA256Err := fileHash(runCtx, helper)
	if helperSHA256Err != nil {
		measureErr = helperSHA256Err
		return
	}
	logf(runCtx, "admitted helper=%s receipt=%s sha256=%s", helper, admission, helperSHA256)
	return
}
func terminalExitAdmitHelper(runCtx context.Context, root, receipt string) (measureValue0 string, measureValue1 string, measureErr error) {
	terminalExitAdmitHelperFeatures2, terminalExitAdmitHelperFeatures22, terminalExitAdmitHelperFeaturesErr := terminalExitAdmitHelperFeatures(runCtx, root, receipt, []string{})
	if terminalExitAdmitHelperFeaturesErr != nil {
		measureErr = terminalExitAdmitHelperFeaturesErr
		return
	}
	return terminalExitAdmitHelperFeatures2, terminalExitAdmitHelperFeatures22, measureErr
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
func measureTerminalExitHelperBuild(runCtx context.Context) (measureErr error) {
	root := input(runCtx, "GOMAD_WASM_TERMINAL_EXIT_HELPER_BUILD_ROOT")
	if root == "" {
		measureErr = fmt.Errorf("%w: %v", ErrRefused, errutil.New("set GOMAD_WASM_TERMINAL_EXIT_HELPER_BUILD_ROOT to build and retain the isolated release helper"))
		return
	}
	terminalExitBuildHelperErr := terminalExitBuildHelper(runCtx, root, []string{})
	if terminalExitBuildHelperErr != nil {
		measureErr = terminalExitBuildHelperErr
		return
	}
	return
}
func terminalExitRustInputs(runCtx context.Context, crate string) (measureValue0 map[string]string, measureErr error) {
	inputs := map[string]string{}
	err := filepath.WalkDir(filepath.Join(crate, "src"), func(path string, entry fs.DirEntry, err error) error {
		callbackValue0, callbackValue1 := (func(runCtx context.Context, path string, entry fs.DirEntry, err error) (measureValue0 error, measureErr error) {
			if err != nil {
				return err, measureErr
			}
			if !entry.IsDir() {
				pathSHA256, pathSHA256Err := fileHash(runCtx, path)
				if pathSHA256Err != nil {
					measureErr = pathSHA256Err
					return
				}
				inputs[path] = pathSHA256
			}
			return nil, measureErr
		})(runCtx, path, entry, err)
		return errutil.Join(callbackValue0, callbackValue1)
	})
	if err != nil {
		measureErr = err
		return
	}
	for _, name := range []string{"Cargo.toml", "Cargo.lock", "rust-toolchain.toml", "../Makefile"} {
		path := filepath.Clean(filepath.Join(crate, name))
		pathSHA2562, pathSHA256Err2 := fileHash(runCtx, path)
		if pathSHA256Err2 != nil {
			measureErr = pathSHA256Err2
			return
		}
		inputs[path] = pathSHA2562
	}
	buildScript := filepath.Join(crate, "build.rs")
	_, err = os.Stat(buildScript)
	if err == nil {
		buildScriptSHA256, buildScriptSHA256Err := fileHash(runCtx, buildScript)
		if buildScriptSHA256Err != nil {
			measureErr = buildScriptSHA256Err
			return
		}
		inputs[buildScript] = buildScriptSHA256
	} else if !os.IsNotExist(err) {
		measureErr = err
		return
	}
	cargoHome := input(runCtx, "CARGO_HOME")
	if cargoHome == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			measureErr = err
			return
		}
		cargoHome = filepath.Join(home, ".cargo")
	}
	for _, name := range []string{"config", "config.toml"} {
		path := filepath.Join(cargoHome, name)
		_, err := os.Stat(path)
		if err == nil {
			pathSHA2563, pathSHA256Err3 := fileHash(runCtx, path)
			if pathSHA256Err3 != nil {
				measureErr = pathSHA256Err3
				return
			}
			inputs[path] = pathSHA2563
		} else if !os.IsNotExist(err) {
			measureErr = err
			return
		}
	}
	directory := crate
	for ; ; directory = filepath.Dir(directory) {
		for _, name := range []string{".cargo/config", ".cargo/config.toml"} {
			path := filepath.Join(directory, name)
			_, err := os.Stat(path)
			if err == nil {
				pathSHA2564, pathSHA256Err4 := fileHash(runCtx, path)
				if pathSHA256Err4 != nil {
					measureErr = pathSHA256Err4
					return
				}
				inputs[path] = pathSHA2564
			} else if !os.IsNotExist(err) {
				measureErr = err
				return
			}
		}
		if filepath.Dir(directory) == directory {
			break
		}
	}
	return inputs, measureErr
}
func terminalExitHelperInputs(runCtx context.Context, receipt string, request wasi.Request, crate string) (measureValue0 map[string]string, measureErr error) {
	terminalExitHelperInputsFeatures2, terminalExitHelperInputsFeaturesErr := terminalExitHelperInputsFeatures(runCtx, receipt, request, crate, []string{})
	if terminalExitHelperInputsFeaturesErr != nil {
		measureErr = terminalExitHelperInputsFeaturesErr
		return
	}
	return terminalExitHelperInputsFeatures2, measureErr
}
func terminalExitModuleInputs(runCtx context.Context, receipt string, request wasi.Request) (measureValue0 map[string]string, measureErr error) {
	data, err := os.ReadFile(receipt)
	if err != nil {
		measureErr = err
		return
	}
	var producer struct {
		Spec     target.Spec
		Prepared target.Prepared
		Options  struct{ CompilerPath string }
	}
	err = json.Unmarshal(data, &producer)
	if err != nil {
		measureErr = err
		return
	}
	prepared := producer.Prepared
	if producer.Spec.Source != "./tests" || prepared.Source != "./tests" || prepared.Kind != target.KindGoTest || prepared.Path != request.ModulePath || prepared.SHA256 != "sha256:"+request.ModuleSHA256 || !slices.Equal(prepared.BuildTags, []string{"disable_grpc_modules", "sqlite3_dotlk", "test_dep"}) {
		measureErr = errutil.New("producer must bind actual ./tests, unchanged tags, and the measured module")
		return
	}
	err = prepared.ValidateBackendPayloads()
	if err != nil {
		measureErr = err
		return
	}
	var provenance struct {
		Schema, Profile, Source, Kind, ModuleSHA256, GoVersion, RuntimeSHA256, RuntimeImplementationSHA256, ModelSHA256, CompilerSHA256 string
		Sources                                                                                                                         []struct{ Path, SHA256 string }
	}
	if len(prepared.BackendPayloads) != 1 {
		measureErr = errutil.New("module producer has no unique retained WASM provenance")
		return
	}
	err = json.Unmarshal(prepared.BackendPayloads[0].Data, &provenance)
	if err != nil {
		measureErr = err
		return
	}
	compilerPathSHA256, compilerPathSHA256Err := fileHash(runCtx, producer.Options.CompilerPath)
	if compilerPathSHA256Err != nil {
		measureErr = compilerPathSHA256Err
		return
	}
	if provenance.Schema != "gomad-wasm.provenance/v1" || provenance.Profile != wasi.CooperativeProfile || provenance.Source != "./tests" || provenance.Kind != "go-test" || provenance.ModuleSHA256 != prepared.SHA256 || provenance.GoVersion != "go1.27.1" || provenance.RuntimeSHA256 != "sha256:"+request.Runtime.Choice.ExecutionIdentity.ToolchainBuildKey || provenance.RuntimeImplementationSHA256 != toolchain.ImplementationSHA256() || provenance.ModelSHA256 != wasi.ImplementationSHA256() || provenance.CompilerSHA256 != "sha256:"+compilerPathSHA256 || len(provenance.Sources) == 0 {
		measureErr = errutil.New("module provenance is stale or lacks current compiler/runtime/model/source bindings")
		return
	}
	receiptSHA256, receiptSHA256Err := fileHash(runCtx, receipt)
	if receiptSHA256Err != nil {
		measureErr = receiptSHA256Err
		return
	}
	inputs := map[string]string{receipt: receiptSHA256, request.ModulePath: request.ModuleSHA256}
	for _, source := range provenance.Sources {
		inputs[source.Path] = strings.TrimPrefix(source.SHA256, "sha256:")
	}
	verifyInputsErr := temporalVerifyInputs(runCtx, inputs)
	if verifyInputsErr != nil {
		measureErr = verifyInputsErr
		return
	}
	return inputs, measureErr
}
func mustTerminalExitRead(runCtx context.Context, path string) (measureValue0 []byte, measureErr error) {
	data, err := os.ReadFile(path)
	if err != nil {
		measureErr = err
		return
	}
	return data, measureErr
}
func measureTerminalToProcessExit(runCtx context.Context) (measureErr error) {
	root := input(runCtx, "GOMAD_WASM_TERMINAL_EXIT_ROOT")
	if root == "" {
		measureErr = fmt.Errorf("%w: %v", ErrRefused, errutil.New("set GOMAD_WASM_TERMINAL_EXIT_ROOT for release-helper Temporal terminal-to-reap measurements"))
		return
	}
	if !filepath.IsAbs(root) {
		measureErr = errutil.New("terminal-exit evidence root must be absolute")
		return
	}
	if !filepath.IsAbs(input(runCtx, "GOMAD_WASM_TEST_HELPER")) {
		measureErr = errutil.New("GOMAD_WASM_TEST_HELPER must explicitly name the candidate release helper")
		return
	}
	evidence, err := diagnosticRunRoot(root, "terminal-exit-")
	if err != nil {
		measureErr = err
		return
	}
	setInput(runCtx, "GOMAD_WASM_PROTOTYPE_ROOT", evidence)
	repo, err := absolute(runCtx, "../../../../../..")
	if err != nil {
		measureErr = err
		return
	}
	engineSource := filepath.Join(repo, "tools/gomad/internal/backends/wasm/wasmhost/src/engine.rs")
	engineBytes, err := os.ReadFile(engineSource)
	if err != nil {
		measureErr = err
		return
	}
	if bytes.Contains(engineBytes, []byte("std::mem::forget")) || !bytes.Contains(engineBytes, []byte("native_unwind_info(false)")) {
		measureErr = errutil.New("measure a release helper built from the no-unwind candidate without the module/store leak")
		return
	}
	engineSHA256, engineSourceSHA256Err := fileHash(runCtx, engineSource)
	if engineSourceSHA256Err != nil {
		measureErr = engineSourceSHA256Err
		return
	}
	module := input(runCtx, "GOMAD_WASM_TERMINAL_EXIT_MODULE")
	if !filepath.IsAbs(module) {
		measureErr = errutil.New("GOMAD_WASM_TERMINAL_EXIT_MODULE must name the current ./tests cooperative module")
		return
	}
	config, prototypeTemporalConfigErr := prototypeTemporalConfig(runCtx, repo, "/workspace/tests", "-test.run=^TestWorkflowTypeEncodingSuite$/^TestPlainASCII$/^Succeeds$", "-persistenceType=sql", "-persistenceDriver=sqlite")
	if prototypeTemporalConfigErr != nil {
		measureErr = prototypeTemporalConfigErr
		return
	}
	request, requestErr := prototypeRequest(runCtx, module, config, 0, 30*time.Minute)
	if requestErr != nil {
		measureErr = requestErr
		return
	}
	helperReceipt := input(runCtx, "GOMAD_WASM_TERMINAL_EXIT_HELPER_RECEIPT")
	moduleReceipt := input(runCtx, "GOMAD_WASM_TERMINAL_EXIT_MODULE_RECEIPT")
	if !filepath.IsAbs(helperReceipt) || !filepath.IsAbs(moduleReceipt) {
		measureErr = errutil.New("absolute helper-admission and ./tests producer receipts are required")
		return
	}
	helperInputs, terminalExitHelperInputsErr := terminalExitHelperInputs(runCtx, helperReceipt, request, filepath.Dir(filepath.Dir(engineSource)))
	if terminalExitHelperInputsErr != nil {
		measureErr = terminalExitHelperInputsErr
		return
	}
	moduleInputs, terminalExitModuleInputsErr := terminalExitModuleInputs(runCtx, moduleReceipt, request)
	if terminalExitModuleInputsErr != nil {
		measureErr = terminalExitModuleInputsErr
		return
	}
	var admission terminalExitHelperAdmission
	mustTerminalExitRead2, mustTerminalExitReadErr := mustTerminalExitRead(runCtx, helperReceipt)
	if mustTerminalExitReadErr != nil {
		measureErr = mustTerminalExitReadErr
		return
	}
	err = json.Unmarshal(mustTerminalExitRead2, &admission)
	if err != nil {
		measureErr = err
		return
	}
	mustTerminalExitRead3, mustTerminalExitReadErr2 := mustTerminalExitRead(runCtx, helperReceipt)
	if mustTerminalExitReadErr2 != nil {
		measureErr = mustTerminalExitReadErr2
		return
	}
	writeEvidenceErr := temporalWrite(runCtx, filepath.Join(evidence, "helper-admission.json"), mustTerminalExitRead3)
	if writeEvidenceErr != nil {
		measureErr = writeEvidenceErr
		return
	}
	mustTerminalExitRead4, mustTerminalExitReadErr3 := mustTerminalExitRead(runCtx, admission.BuildReceipt)
	if mustTerminalExitReadErr3 != nil {
		measureErr = mustTerminalExitReadErr3
		return
	}
	writeEvidenceErr2 := temporalWrite(runCtx, filepath.Join(evidence, "helper-build.json"), mustTerminalExitRead4)
	if writeEvidenceErr2 != nil {
		measureErr = writeEvidenceErr2
		return
	}
	mustTerminalExitRead5, mustTerminalExitReadErr4 := mustTerminalExitRead(runCtx, moduleReceipt)
	if mustTerminalExitReadErr4 != nil {
		measureErr = mustTerminalExitReadErr4
		return
	}
	writeEvidenceErr3 := temporalWrite(runCtx, filepath.Join(evidence, "temporal-producer.json"), mustTerminalExitRead5)
	if writeEvidenceErr3 != nil {
		measureErr = writeEvidenceErr3
		return
	}
	return terminalExitCollect(runCtx, evidence, request, wasi.MeasureTerminalExit, func() error {
		if err := temporalVerifyInputs(runCtx, helperInputs); err != nil {
			return err
		}
		if err := temporalVerifyInputs(runCtx, moduleInputs); err != nil {
			return err
		}
		_, err := terminalExitHelperInputs(runCtx, helperReceipt, request, filepath.Dir(filepath.Dir(engineSource)))
		return err
	}, func(observations []wasi.TerminalExitObservation) (measureErr error) {
		info, err := os.Stat(module)
		if err != nil {
			measureErr = err
			return
		}
		helperReceiptSHA256, helperReceiptSHA256Err := fileHash(runCtx, helperReceipt)
		if helperReceiptSHA256Err != nil {
			measureErr = helperReceiptSHA256Err
			return
		}
		moduleReceiptSHA256, moduleReceiptSHA256Err := fileHash(runCtx, moduleReceipt)
		if moduleReceiptSHA256Err != nil {
			measureErr = moduleReceiptSHA256Err
			return
		}
		writeReportErr2 := temporalJSON(runCtx, filepath.Join(evidence, "terminal-exit.json"), map[string]any{"schema": "gomad.wasm-terminal-exit-measurement/v1", "host": wasi.PinnedEngine(), "bound_nanos": uint64(time.Second), "module_sha256": request.ModuleSHA256, "module_bytes": info.Size(), "helper_sha256": request.HelperSHA256, "engine_source_sha256": engineSHA256, "helper_receipt_sha256": helperReceiptSHA256, "module_receipt_sha256": moduleReceiptSHA256, "selection": config.Args, "observations": observations, "boundary": "receipt of complete actual validated terminal frame through completed host supervisor wait, capture drain and group reaping; conservative upper bound on exit time"})
		if writeReportErr2 != nil {
			measureErr = writeReportErr2
			return
		}
		return
	})
}

func terminalExitCollect(runCtx context.Context, evidence string, request wasi.Request,
	execute func(context.Context, wasi.Request, bool) (wasi.TerminalExitObservation, error),
	verify func() error, report func([]wasi.TerminalExitObservation) error) (measureErr error) {
	var observations []wasi.TerminalExitObservation
	failed := false
	for _, mismatch := range []bool{false, true} {
		name := "success"
		if mismatch {
			name = "post-load-authorization-mismatch"
		}
		observation, measureTerminalExitErr := execute(runCtx, request, mismatch)
		if measureTerminalExitErr != nil {
			measureErr = errutil.Join(measureErr, measureTerminalExitErr)
			failed = true
		}
		observations = append(observations, observation)
		logf(runCtx, "%s terminal-to-reap=%dns termination=%s reaped=%v error=%s", name, observation.TerminalToReapNanos, observation.Result.Termination, observation.Host.GroupGone, observation.Error)
		writeReportErr := temporalJSON(runCtx, filepath.Join(evidence, name+"-observation.json"), observation)
		if writeReportErr != nil {
			measureErr = errutil.Join(measureErr, writeReportErr)
			return
		}
		writeEvidenceErr4 := temporalWrite(runCtx, filepath.Join(evidence, name+"-helper-stderr.txt"), observation.Host.Stderr.RawBytes)
		if writeEvidenceErr4 != nil {
			measureErr = errutil.Join(measureErr, writeEvidenceErr4)
			return
		}
		if err := verify(); err != nil {
			return errutil.Join(measureErr, err)
		}
		if !observation.Within(time.Second) {
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
	if err := report(observations); err != nil {
		return errutil.Join(measureErr, err)
	}
	if failed {
		measureErr = errutil.Join(measureErr, fmt.Errorf("temporal terminal-to-process-exit proof failed; raw measurements retained in %s", evidence))
		return
	}
	return
}
func measureScratchSmokeProfile(runCtx context.Context) (measureErr error) {
	root, prototypeRootErr := prototypeRoot(runCtx)
	if prototypeRootErr != nil {
		measureErr = prototypeRootErr
		return
	}
	repo, err := absolute(runCtx, "../../../../../..")
	if err != nil {
		measureErr = err
		return
	}
	modules := filepath.Join(root, "modules")
	err = os.MkdirAll(modules, 0700)
	if err != nil {
		measureErr = err
		return
	}
	module := filepath.Join(modules, "tests.wasm")
	_, err = os.Stat(module)
	if err != nil {
		prototypeBuild2, prototypeBuild22, prototypeBuildErr := prototypeBuild(runCtx, repo, "test", "./tests", module, "disable_grpc_modules", "test_dep", "sqlite3_dotlk")
		if prototypeBuildErr != nil {
			measureErr = prototypeBuildErr
			return
		}
		_, seconds := prototypeBuild2, prototypeBuild22
		logf(runCtx, "built tests.wasm in %.1fs", seconds)
	}
	persistence := []string{"-persistenceType=sql", "-persistenceDriver=sqlite"}
	suites := []struct{ name, run string }{{"workflow", "^TestWorkflowTypeEncodingSuite$/^TestPlainASCII$/^Succeeds$"}, {"activity", "^TestActivityTestSuite$"}}
	evidence := filepath.Join(root, "evidence")
	err = os.MkdirAll(evidence, 0700)
	if err != nil {
		measureErr = err
		return
	}
	helpers := strings.Split(input(runCtx, "GOMAD_SCRATCH_HELPERS"), ",")
	for _, owner := range helpers {
		name, path, _ := strings.Cut(owner, "=")
		instrumented := strings.HasPrefix(name, "diag")
		setInput(runCtx, "GOMAD_WASM_TEST_HELPER", path)
		prototypeTemporalConfig2, prototypeTemporalConfigErr := prototypeTemporalConfig(runCtx, repo, "/workspace/tests", append([]string{"-test.run=^$"}, persistence...)...)
		if prototypeTemporalConfigErr != nil {
			measureErr = prototypeTemporalConfigErr
			return
		}
		warm, requestErr := prototypeRequest(runCtx, module, prototypeTemporalConfig2, 0, 30*time.Minute)
		if requestErr != nil {
			measureErr = requestErr
			return
		}
		if input(runCtx, "GOMAD_SCRATCH_SKIP_PREPARE") == "" {
			runDiagnosticProfile2, runDiagnosticProfileErr := runDiagnosticProfile(runCtx, evidence, name+"-prepare", warm, instrumented, false)
			if runDiagnosticProfileErr != nil {
				measureErr = runDiagnosticProfileErr
				return
			}
			_ = runDiagnosticProfile2
		}
		for _, suite := range suites {
			only := input(runCtx, "GOMAD_SCRATCH_ONLY")
			if only != "" && only != suite.name {
				continue
			}
			for _, seed := range []uint64{0, 1000003} {
				config, prototypeTemporalConfigErr2 := prototypeTemporalConfig(runCtx, repo, "/workspace/tests", append([]string{"-test.run=" + suite.run}, persistence...)...)
				if prototypeTemporalConfigErr2 != nil {
					measureErr = prototypeTemporalConfigErr2
					return
				}
				request, requestErr2 := prototypeRequest(runCtx, module, config, seed, 30*time.Minute)
				if requestErr2 != nil {
					measureErr = requestErr2
					return
				}
				if input(runCtx, "GOMAD_SCRATCH_TRANSCRIPT") != "" {
					request.DigestTranscript = false
				}
				started := time.Now()
				result, runDiagnosticProfileErr2 := runDiagnosticProfile(runCtx, evidence, name+"-"+suite.name+"-seed"+stringSeed(seed), request, instrumented, true)
				if runDiagnosticProfileErr2 != nil {
					measureErr = runDiagnosticProfileErr2
					return
				}
				if !request.DigestTranscript {
					operations := map[string]int{}
					for _, line := range bytes.Split(result.Transcript, []byte{'\n'}) {
						var frame struct{ Type, Op string }
						if json.Unmarshal(line, &frame) == nil && frame.Type == "call" {
							operations[frame.Op]++
						}
					}
					logf(runCtx, "operations: %v", operations)
				}
				logf(runCtx, "%s %s seed %d: run wall %s, authorized→terminal %s", name, suite.name, seed, time.Since(started), time.Duration(result.AuthorizationToTerminalNanos))
			}
		}
	}
	return
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

func diagnosticGit(runCtx context.Context, repo string, args ...string) (string, error) {
	result, err := hostexec.Run(runCtx, hostexec.Request{Command: append([]string{"git", "-C", repo}, args...), Dir: repo, Timeout: 30 * time.Second, TerminateGrace: time.Second, OutputLimit: 1 << 20})
	if err != nil || result.ExitCode != 0 || result.Termination != hostexec.TerminationExit || !result.GroupGone || result.Stdout.Truncated || result.Stderr.Truncated {
		return "", fmt.Errorf("diagnostic source identity: %v, exit=%d, termination=%s", err, result.ExitCode, result.Termination)
	}
	return strings.TrimSpace(string(result.Stdout.RawBytes)), nil
}
func diagnosticCurrentHost(runCtx context.Context) (measureValue0 string, measureValue1 string, measureErr error) {
	boot, cpu, err := warmPreparationHostEvidence(runCtx)
	if err != nil || runtime.Version() != "go1.27.1" {
		measureErr = errutil.New(fmt.Sprint("diagnostic measurements require current host evidence and stock Go 1.27.1", err))
		return
	}
	return cpu, boot, measureErr
}
func newDiagnosticAdmission(runCtx context.Context, repo, root string) (measureValue0 *diagnosticAdmission, measureErr error) {
	cpu, boot, diagnosticCurrentHostErr := diagnosticCurrentHost(runCtx)
	if diagnosticCurrentHostErr != nil {
		measureErr = diagnosticCurrentHostErr
		return
	}
	head, err := diagnosticGit(runCtx, repo, "rev-parse", "HEAD")
	if err != nil || head == "" {
		measureErr = errutil.New(fmt.Sprint("diagnostic source HEAD unavailable", err))
		return
	}
	a := &diagnosticAdmission{Repo: repo, SourceHead: head, CPUInfo: cpu, BootID: boot, Inputs: map[string]string{}, Helpers: map[string]diagnosticHelperAdmission{}, Modules: map[string]diagnosticModuleAdmission{}, ModuleReceipts: map[string]string{}}
	checkHostErr := a.checkHost(runCtx, true)
	if checkHostErr != nil {
		measureErr = checkHostErr
		return
	}
	a.Receipt = filepath.Join(root, "host-admission.json")
	writeReportErr := temporalJSON(runCtx, a.Receipt, map[string]any{"schema": "gomad-diagnostic-host/v1", "source": repo, "source_head": head, "go_version": runtime.Version(), "goos": runtime.GOOS, "goarch": runtime.GOARCH, "cpuinfo": cpu, "boot_id": boot, "minimum_mem_available_bytes": uint64(8 << 30)})
	if writeReportErr != nil {
		measureErr = writeReportErr
		return
	}
	receiptSHA256, receiptSHA256Err := fileHash(runCtx, a.Receipt)
	if receiptSHA256Err != nil {
		measureErr = receiptSHA256Err
		return
	}
	a.Inputs[a.Receipt] = receiptSHA256
	return a, measureErr
}
func (a *diagnosticAdmission) checkHost(runCtx context.Context, before bool) (measureErr error) {
	cpu, boot, diagnosticCurrentHostErr := diagnosticCurrentHost(runCtx)
	if diagnosticCurrentHostErr != nil {
		measureErr = diagnosticCurrentHostErr
		return
	}
	head, headErr := diagnosticGit(runCtx, a.Repo, "rev-parse", "HEAD")
	status, statusErr := diagnosticGit(runCtx, a.Repo, "status", "--porcelain", "--untracked-files=all")
	if cpu != a.CPUInfo || boot != a.BootID || headErr != nil || statusErr != nil || head != a.SourceHead || status != "" {
		measureErr = errutil.New(fmt.Sprint("diagnostic CPU/boot/source identity changed or worktree is dirty", headErr, statusErr))
		return
	}
	if before {
		_, err := warmPreparationAdmitted(runCtx, func() (wasi.Result, error) {
			return wasi.Result{}, nil
		})
		if err != nil {
			measureErr = err
			return
		}
	}
	return
}
func (a *diagnosticAdmission) addHelper(runCtx context.Context, helper, receipt string, features []string) (measureErr error) {
	if !filepath.IsAbs(receipt) {
		measureErr = errutil.New("diagnostic helper requires an absolute authentic helper-admission receipt")
		return
	}
	crate := filepath.Join(a.Repo, "tools/gomad/internal/backends/wasm/wasmhost")
	helperSHA256, helperSHA256Err := fileHash(runCtx, helper)
	if helperSHA256Err != nil {
		measureErr = helperSHA256Err
		return
	}
	inputs, terminalExitHelperInputsFeaturesErr := terminalExitHelperInputsFeatures(runCtx, receipt, wasi.Request{HelperPath: helper, HelperSHA256: helperSHA256}, crate, features)
	if terminalExitHelperInputsFeaturesErr != nil {
		measureErr = terminalExitHelperInputsFeaturesErr
		return
	}
	var admission terminalExitHelperAdmission
	mustTerminalExitRead2, mustTerminalExitReadErr := mustTerminalExitRead(runCtx, receipt)
	if mustTerminalExitReadErr != nil {
		measureErr = mustTerminalExitReadErr
		return
	}
	err := json.Unmarshal(mustTerminalExitRead2, &admission)
	if err != nil {
		measureErr = err
		return
	}
	var build terminalExitHelperBuild
	mustTerminalExitRead3, mustTerminalExitReadErr2 := mustTerminalExitRead(runCtx, admission.BuildReceipt)
	if mustTerminalExitReadErr2 != nil {
		measureErr = mustTerminalExitReadErr2
		return
	}
	err = json.Unmarshal(mustTerminalExitRead3, &build)
	if err != nil || !diagnosticHelperCommand(build, features) {
		measureErr = errutil.New(fmt.Sprint("diagnostic helper must retain the exact pinned, single-job feature build command", err))
		return
	}
	for _, name := range []string{"cargo-build.stdout", "cargo-build.json", "rustc-version.stdout", "cargo-version.stdout"} {
		path := filepath.Join(filepath.Dir(admission.BuildReceipt), name)
		pathSHA256, pathSHA256Err := fileHash(runCtx, path)
		if pathSHA256Err != nil {
			measureErr = pathSHA256Err
			return
		}
		if build.Inputs[path] != pathSHA256 {
			measureErr = errutil.New(fmt.Sprint("diagnostic helper is missing authenticated raw compiler evidence", name))
			return
		}
	}
	var execution struct {
		Command  []string
		ExitCode int
	}
	mustTerminalExitRead4, mustTerminalExitReadErr3 := mustTerminalExitRead(runCtx, filepath.Join(filepath.Dir(admission.BuildReceipt), "cargo-build.json"))
	if mustTerminalExitReadErr3 != nil {
		measureErr = mustTerminalExitReadErr3
		return
	}
	err = json.Unmarshal(mustTerminalExitRead4, &execution)
	if err != nil || execution.ExitCode != 0 || !slices.Equal(execution.Command, build.Command) {
		measureErr = errutil.New(fmt.Sprint("diagnostic helper command differs from actual retained execution", err))
		return
	}
	mustTerminalExitRead5, mustTerminalExitReadErr4 := mustTerminalExitRead(runCtx, filepath.Join(filepath.Dir(admission.BuildReceipt), "rustc-version.stdout"))
	if mustTerminalExitReadErr4 != nil {
		measureErr = mustTerminalExitReadErr4
		return
	}
	mustTerminalExitRead6, mustTerminalExitReadErr5 := mustTerminalExitRead(runCtx, filepath.Join(filepath.Dir(admission.BuildReceipt), "cargo-version.stdout"))
	if mustTerminalExitReadErr5 != nil {
		measureErr = mustTerminalExitReadErr5
		return
	}
	if string(mustTerminalExitRead5) != build.RustcVersion || string(mustTerminalExitRead6) != build.CargoVersion {
		measureErr = errutil.New("diagnostic helper compiler versions differ from actual retained output")
		return
	}
	a.Helpers[helper] = diagnosticHelperAdmission{Receipt: receipt, Features: slices.Clone(features)}
	maps.Copy(a.Inputs, inputs)
	return
}
func (a *diagnosticAdmission) check(runCtx context.Context, root, name string, request wasi.Request, before bool) (measureErr error) {
	checkHostErr := a.checkHost(runCtx, before)
	if checkHostErr != nil {
		measureErr = checkHostErr
		return
	}
	err := diagnosticVerifyInputs(a.Inputs)
	if err != nil {
		measureErr = err
		return
	}
	helper, ok := a.Helpers[request.HelperPath]
	if !ok {
		measureErr = errutil.New("diagnostic dispatch has no admitted helper")
		return
	}
	terminalExitHelperInputsFeatures2, terminalExitHelperInputsFeaturesErr := terminalExitHelperInputsFeatures(runCtx, helper.Receipt, request, filepath.Join(a.Repo, "tools/gomad/internal/backends/wasm/wasmhost"), helper.Features)
	if terminalExitHelperInputsFeaturesErr != nil {
		measureErr = terminalExitHelperInputsFeaturesErr
		return
	}
	_ = terminalExitHelperInputsFeatures2
	module, ok := a.Modules[request.ModulePath]
	if !ok || !diagnosticModuleMatches(module, request) {
		measureErr = errutil.New("diagnostic dispatch has no matching actual module build")
		return
	}
	writeReportErr := temporalJSON(runCtx, filepath.Join(root, name+"-admission.json"), map[string]any{"host_receipt": a.Receipt, "host_receipt_sha256": a.Inputs[a.Receipt], "helper_receipt": helper.Receipt, "helper_receipt_sha256": a.Inputs[helper.Receipt], "module_receipt": a.ModuleReceipts[request.ModulePath], "module_receipt_sha256": a.Inputs[a.ModuleReceipts[request.ModulePath]], "helper_sha256": request.HelperSHA256, "module_sha256": request.ModuleSHA256, "source_head": a.SourceHead, "before_dispatch": before})
	if writeReportErr != nil {
		measureErr = writeReportErr
		return
	}
	return
}
func diagnosticModuleMatches(module diagnosticModuleAdmission, request wasi.Request) bool {
	return request.Runtime != nil && request.Runtime.Choice != nil && module.Schema == "gomad-diagnostic-module-build/v1" && module.Module == request.ModulePath && module.ModuleSHA256 == request.ModuleSHA256 && module.RuntimeKey == request.Runtime.Choice.ExecutionIdentity.ToolchainBuildKey && module.RuntimeSHA256 == toolchain.ImplementationSHA256() && module.ModelSHA256 == wasi.ImplementationSHA256()
}
func (a *diagnosticAdmission) buildModule(runCtx context.Context, output, gowasm string) (measureErr error) {
	moduleTargetErr := a.buildModuleTarget(runCtx, a.Repo, output, gowasm, "./tests", "test", []string{"disable_grpc_modules", "test_dep", "sqlite3_dotlk"})
	if moduleTargetErr != nil {
		measureErr = moduleTargetErr
		return
	}
	return
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
func (a *diagnosticAdmission) buildModuleTarget(runCtx context.Context, directory, output, gowasm, target, kind string, tags []string) (measureErr error) {
	if gowasm != "" && gowasm != "satconv,signext" {
		measureErr = errutil.New("unexpected diagnostic GOWASM variant")
		return
	}
	checkHostErr := a.checkHost(runCtx, true)
	if checkHostErr != nil {
		measureErr = checkHostErr
		return
	}
	evidence, err := diagnosticRunRoot(filepath.Dir(output), "module-build-")
	if err != nil {
		measureErr = err
		return
	}
	_, err = os.Lstat(output)
	if !os.IsNotExist(err) {
		measureErr = errutil.New(fmt.Sprint("refusing to overwrite a diagnostic module", err))
		return
	}
	compiler, overlay, key, prototypeToolchainErr := prototypeToolchain(runCtx)
	if prototypeToolchainErr != nil {
		measureErr = prototypeToolchainErr
		return
	}
	compilerSHA256, compilerSHA256Err := fileHash(runCtx, compiler)
	if compilerSHA256Err != nil {
		measureErr = compilerSHA256Err
		return
	}
	var environment []string
	for _, entry := range environ(runCtx) {
		name, _, _ := strings.Cut(entry, "=")
		if slices.Contains([]string{"PATH", "HOME", "GOCACHE", "GOMODCACHE", "GOPATH", "TMPDIR", "GOTMPDIR", "GOPROXY", "GOSUMDB", "GONOPROXY", "GONOSUMDB", "GOPRIVATE"}, name) {
			environment = append(environment, entry)
		}
	}
	environment = append(environment, "GOROOT="+filepath.Dir(filepath.Dir(compiler)), "GOOS=wasip1", "GOARCH=wasm", "GOEXPERIMENT=nogreenteagc", "GOWORK=off", "GOENV=off", "GOFLAGS=-mod=readonly", "GOTOOLCHAIN=local", "CGO_ENABLED=0", "GOWASM="+gowasm, "GOMAXPROCS=1")
	version, commandErr := temporalCommand(runCtx, evidence, "go-version", directory, environment, []string{compiler, "version"})
	if commandErr != nil {
		measureErr = commandErr
		return
	}
	if version.Stdout.Truncated || version.Stderr.Truncated || !strings.HasPrefix(string(version.Stdout.RawBytes), "go version go1.27.1 ") {
		measureErr = errutil.New("diagnostic module requires pinned stock Go 1.27.1")
		return
	}
	listCommand, command, err := diagnosticModuleCommands(compiler, overlay, output, target, kind, tags)
	if err != nil {
		measureErr = err
		return
	}
	listed, commandErr2 := temporalCommand(runCtx, evidence, "sources-before", directory, environment, listCommand)
	if commandErr2 != nil {
		measureErr = commandErr2
		return
	}
	if listed.Stdout.Truncated || listed.Stderr.Truncated {
		measureErr = errutil.New("truncated diagnostic source closure")
		return
	}
	inputs, diagnosticModuleSourceInputsErr := diagnosticModuleSourceInputs(runCtx, listed.Stdout.RawBytes, compiler, overlay, directory)
	if diagnosticModuleSourceInputsErr != nil {
		measureErr = diagnosticModuleSourceInputsErr
		return
	}
	if inputs[compiler] != compilerSHA256 {
		measureErr = errutil.New("diagnostic compiler changed before build")
		return
	}
	built, commandErr3 := temporalCommand(runCtx, evidence, "module-build", directory, environment, command)
	if commandErr3 != nil {
		measureErr = commandErr3
		return
	}
	if built.Stdout.Truncated || built.Stderr.Truncated {
		measureErr = errutil.New("truncated diagnostic compiler output")
		return
	}
	command2, commandErr4 := temporalCommand(runCtx, evidence, "sources-after", directory, environment, listCommand)
	if commandErr4 != nil {
		measureErr = commandErr4
		return
	}
	listed = command2
	if listed.Stdout.Truncated || listed.Stderr.Truncated {
		measureErr = errutil.New("truncated diagnostic source closure")
		return
	}
	diagnosticModuleSourceInputs2, diagnosticModuleSourceInputsErr2 := diagnosticModuleSourceInputs(runCtx, listed.Stdout.RawBytes, compiler, overlay, directory)
	if diagnosticModuleSourceInputsErr2 != nil {
		measureErr = diagnosticModuleSourceInputsErr2
		return
	}
	if !maps.Equal(inputs, diagnosticModuleSourceInputs2) {
		measureErr = errutil.New("diagnostic compiler, overlay or source closure changed during module build")
		return
	}
	checkHostErr2 := a.checkHost(runCtx, false)
	if checkHostErr2 != nil {
		measureErr = checkHostErr2
		return
	}
	for _, operation := range []string{"go-version", "sources-before", "module-build", "sources-after"} {
		for _, suffix := range []string{".json", ".stdout", ".stderr"} {
			path := filepath.Join(evidence, operation+suffix)
			pathSHA256, pathSHA256Err := fileHash(runCtx, path)
			if pathSHA256Err != nil {
				measureErr = pathSHA256Err
				return
			}
			inputs[path] = pathSHA256
		}
	}
	outputSHA256, outputSHA256Err := fileHash(runCtx, output)
	if outputSHA256Err != nil {
		measureErr = outputSHA256Err
		return
	}
	module := diagnosticModuleAdmission{Schema: "gomad-diagnostic-module-build/v1", Source: a.Repo, SourceHead: a.SourceHead, CPUInfoSHA256: digest([]byte(a.CPUInfo)), Directory: directory, Target: target, Kind: kind, Tags: slices.Clone(tags), Module: output, ModuleSHA256: outputSHA256, Compiler: compiler, CompilerVersion: string(version.Stdout.RawBytes), Overlay: overlay, RuntimeKey: key, RuntimeSHA256: toolchain.ImplementationSHA256(), ModelSHA256: wasi.ImplementationSHA256(), GOWASM: gowasm, Command: command, Environment: environment, Inputs: inputs}
	receipt := filepath.Join(evidence, "module-admission.json")
	writeReportErr := temporalJSON(runCtx, receipt, module)
	if writeReportErr != nil {
		measureErr = writeReportErr
		return
	}
	a.Modules[output], a.ModuleReceipts[output] = module, receipt
	maps.Copy(a.Inputs, inputs)
	receiptSHA256, receiptSHA256Err := fileHash(runCtx, receipt)
	if receiptSHA256Err != nil {
		measureErr = receiptSHA256Err
		return
	}
	a.Inputs[output], a.Inputs[receipt] = module.ModuleSHA256, receiptSHA256
	return
}
func diagnosticModuleSourceInputs(runCtx context.Context, data []byte, compiler, overlay, repo string) (measureValue0 map[string]string, measureErr error) {
	var replacement struct{ Replace map[string]string }
	mustTerminalExitRead2, mustTerminalExitReadErr := mustTerminalExitRead(runCtx, overlay)
	if mustTerminalExitReadErr != nil {
		measureErr = mustTerminalExitReadErr
		return
	}
	err := json.Unmarshal(mustTerminalExitRead2, &replacement)
	if err != nil || len(replacement.Replace) == 0 {
		measureErr = errutil.New(fmt.Sprint("diagnostic module requires actual cooperative overlay bindings", err))
		return
	}
	inputs := map[string]string{}
	bind := func(runCtx context.Context, name string) (measureErr error) {
		to, replaced := replacement.Replace[name]
		if replaced {
			if to == "" {
				measureErr = errutil.New(fmt.Sprint("cooperative overlay removed a selected build input", name))
				return
			}
			toSHA256, toSHA256Err := fileHash(runCtx, to)
			if toSHA256Err != nil {
				measureErr = toSHA256Err
				return
			}
			inputs[to] = toSHA256
			_, err := os.Stat(name)
			if os.IsNotExist(err) {
				return
			} else if err != nil {
				measureErr = err
				return
			}
		}
		nameSHA256, nameSHA256Err := fileHash(runCtx, name)
		if nameSHA256Err != nil {
			measureErr = nameSHA256Err
			return
		}
		inputs[name] = nameSHA256
		return
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
		err := decoder.Decode(&pkg)
		if err == io.EOF {
			break
		} else if err != nil {
			measureErr = err
			return
		}
		for _, names := range [][]string{pkg.GoFiles, pkg.CgoFiles, pkg.SFiles, pkg.HFiles, pkg.SysoFiles, pkg.EmbedFiles, pkg.TestGoFiles, pkg.XTestGoFiles, pkg.TestEmbedFiles, pkg.XTestEmbedFiles} {
			for _, name := range names {
				if !filepath.IsAbs(name) {
					name = filepath.Join(pkg.Dir, name)
				}
				bindErr := bind(runCtx, name)
				if bindErr != nil {
					measureErr = bindErr
					return
				}
			}
		}
		if pkg.Module != nil {
			gomod := pkg.Module.GoMod
			if pkg.Module.Replace != nil {
				gomod = pkg.Module.Replace.GoMod
			}
			if gomod != "" {
				gomodSHA256, gomodSHA256Err := fileHash(runCtx, gomod)
				if gomodSHA256Err != nil {
					measureErr = gomodSHA256Err
					return
				}
				inputs[gomod] = gomodSHA256
			}
		}
	}
	for from, to := range replacement.Replace {
		if to != "" {
			toSHA2562, toSHA256Err2 := fileHash(runCtx, to)
			if toSHA256Err2 != nil {
				measureErr = toSHA256Err2
				return
			}
			inputs[to] = toSHA2562
		}
		_, err := os.Stat(from)
		if err == nil {
			fromSHA256, fromSHA256Err := fileHash(runCtx, from)
			if fromSHA256Err != nil {
				measureErr = fromSHA256Err
				return
			}
			inputs[from] = fromSHA256
		} else if !os.IsNotExist(err) {
			measureErr = err
			return
		}
	}
	for _, path := range []string{compiler, overlay, filepath.Join(repo, "go.mod"), filepath.Join(repo, "go.sum"), filepath.Join(filepath.Dir(filepath.Dir(compiler)), "VERSION")} {
		pathSHA256, pathSHA256Err := fileHash(runCtx, path)
		if pathSHA256Err != nil {
			measureErr = pathSHA256Err
			return
		}
		inputs[path] = pathSHA256
	}
	for _, tool := range []string{"compile", "link", "asm"} {
		path := filepath.Join(filepath.Dir(filepath.Dir(compiler)), "pkg/tool", runtime.GOOS+"_"+runtime.GOARCH, tool)
		pathSHA2562, pathSHA256Err2 := fileHash(runCtx, path)
		if pathSHA256Err2 != nil {
			measureErr = pathSHA256Err2
			return
		}
		inputs[path] = pathSHA2562
	}
	return inputs, measureErr
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

func decodeClusterDiagnosticPair(requests []wasi.Request, results []wasi.Result) (diagnosticClusterPair, error) {
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
		profile, err := decodeDiagnosticReport(result.HelperStderr())
		if err != nil {
			pair.Complete = false
			pair.Instances[index].Unavailable = err.Error()
			if result.Termination != "crash" || !result.Reaped || len(result.HelperStderr()) > 16<<10 {
				failures = append(failures, fmt.Sprintf("helper %d: %v", index, err))
			}
			continue
		}
		if pair.Measured != 0 && (profile.HostOS != pair.MeasuredSum.HostOS || profile.HostArch != pair.MeasuredSum.HostArch) {
			pair.Complete, pair.MeasuredSum = false, nil
			return pair, fmt.Errorf("cluster diagnostic host mismatch")
		}
		if pair.MeasuredSum == nil {
			pair.MeasuredSum = &diagnosticReport{Schema: "gomad.wasm-profile/v1", DiagnosticOnly: true, Backend: "cranelift-jit", Phases: map[string]diagnosticPhase{}, ABI: map[string]diagnosticABI{}, Nested: map[string]diagnosticNested{}}
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

var diagnosticClusterCollectorSources = []string{"../cmd/wasmmeasure/main.go", "../internal/measure/run.go", "../internal/measure/producers.go", "../internal/measure/cluster_candidate.go", "../internal/measure/support_process.go", "../internal/measure/support_environment.go", "../internal/measure/terminal_bytes.go", "../internal/measure/profile_stdlib.go", "../internal/measurement/measurement.go", "measurement.go", "process.go"}

const warmPreparationDriverSource = "tools/gomad/internal/backends/wasm/internal/measure/producers.go"

func measureDiagnosticClusterBaseline(runCtx context.Context) (measureErr error) {
	base := input(runCtx, "GOMAD_WASM_TEMPORAL_RECOVERY_ROOT")
	if base == "" {
		measureErr = errutil.Join(measureErr, fmt.Errorf("%w: %v", ErrRefused, errutil.New("set GOMAD_WASM_TEMPORAL_RECOVERY_ROOT for the unsampled cluster baseline")))
		return
	}
	helper := input(runCtx, "GOMAD_WASM_TEST_HELPER")
	if !filepath.IsAbs(base) || filepath.Clean(base) != base || base == string(filepath.Separator) || !filepath.IsAbs(helper) || !filepath.IsAbs(input(runCtx, "GOMAD_STOCK_GO")) {
		measureErr = errutil.Join(measureErr, errutil.New("baseline root, diagnostic-profile helper and stock Go compiler must be explicit absolute paths"))
		return
	}
	repo, err := absolute(runCtx, "../../../../../..")
	if err != nil {
		measureErr = errutil.Join(measureErr, err)
		return
	}
	relative, err := filepath.Rel(repo, base)
	if err != nil || relative != ".." && !strings.HasPrefix(relative, ".."+string(os.PathSeparator)) {
		measureErr = errutil.Join(measureErr, errutil.New("raw cluster baseline evidence must be retained outside the repository"))
		return
	}
	cpuinfo, admitted := currentDiagnosticPerformanceHostEvidence(runCtx)
	if !admitted {
		measureErr = errutil.Join(measureErr, errutil.New("cluster baseline requires a current host CPU witness"))
		return
	}
	prerequisites := map[string]string{}
	index := 1
	for ; index <= 12; index++ {
		name := fmt.Sprintf(".flow/tasks/fn-154-gomad-simpler-virtual-network-with.%d.json", index)
		data, err := hostfs.ReadBounded(filepath.Join(repo, name), 1<<20)
		var task struct {
			Status string `json:"status"`
		}
		if err != nil || json.Unmarshal(data, &task) != nil || task.Status != "done" {
			measureErr = errutil.Join(measureErr, fmt.Errorf("cluster baseline requires completed post-fn-154 prerequisite %s", name))
			return
		}
		prerequisites[name] = digest(data)
	}
	commit, err := exec.CommandContext(runCtx, "git", "-C", repo, "rev-parse", "HEAD").Output()
	if err != nil {
		measureErr = errutil.Join(measureErr, err)
		return
	}
	status, err := exec.CommandContext(runCtx, "git", "-C", repo, "status", "--porcelain").Output()
	if err != nil {
		measureErr = errutil.Join(measureErr, err)
		return
	}
	inputs := map[string]string{}
	for _, name := range diagnosticClusterCollectorSources {
		path, err := absolute(runCtx, name)
		if err != nil {
			measureErr = errutil.Join(measureErr, err)
			return
		}
		pathSHA256, pathSHA256Err := fileHash(runCtx, path)
		if pathSHA256Err != nil {
			measureErr = errutil.Join(measureErr, pathSHA256Err)
			return
		}
		inputs[path] = pathSHA256
	}
	parentExecutable, err := os.Executable()
	if err != nil {
		measureErr = errutil.Join(measureErr, err)
		return
	}
	parentSHA256, parentExecutableSHA256Err := fileHash(runCtx, parentExecutable)
	if parentExecutableSHA256Err != nil {
		measureErr = errutil.Join(measureErr, parentExecutableSHA256Err)
		return
	}
	root, err := diagnosticRunRoot(base, "cluster-baseline-")
	if err != nil {
		measureErr = errutil.Join(measureErr, err)
		return
	}
	logf(runCtx, "retaining cluster baseline in %s", root)
	admission, diagnosticAdmissionErr := newDiagnosticAdmission(runCtx, repo, root)
	if diagnosticAdmissionErr != nil {
		measureErr = errutil.Join(measureErr, diagnosticAdmissionErr)
		return
	}
	addHelperErr := admission.addHelper(runCtx, helper, input(runCtx, "GOMAD_WASM_DIAGNOSTIC_HELPER_ADMISSION"), []string{"diagnostic-profile"})
	if addHelperErr != nil {
		measureErr = errutil.Join(measureErr, addHelperErr)
		return
	}
	admission.Inputs[parentExecutable] = parentSHA256
	maps.Copy(admission.Inputs, inputs)
	for name, hash := range prerequisites {
		admission.Inputs[filepath.Join(repo, name)] = hash
	}
	setInput(runCtx, "GOMAD_WASM_TEMPORAL_RECOVERY_ROOT", root)
	setInput(runCtx, "GOMAD_WASM_PROTOTYPE_ROOT", root)
	probeModule := filepath.Join(root, "diagnostic-feature-probe.wasm")
	moduleTargetErr := admission.buildModuleTarget(runCtx, filepath.Join(repo, "tools/gomad"), probeModule, "", "./internal/backends/wasm/toolchain/testdata/choices", "build", nil)
	if moduleTargetErr != nil {
		measureErr = errutil.Join(measureErr, moduleTargetErr)
		return
	}
	probe, cooperativeModuleRequestErr := cooperativeModuleRequest(runCtx, probeModule, admission.Modules[probeModule].RuntimeKey)
	if cooperativeModuleRequestErr != nil {
		measureErr = errutil.Join(measureErr, cooperativeModuleRequestErr)
		return
	}
	probeResult, probeError := func(runCtx context.Context) (measureValue0 wasi.Result, measureErr error) {
		checkErr := admission.check(runCtx, root, "diagnostic-feature-probe-before", probe, true)
		if checkErr != nil {
			measureErr = errutil.Join(measureErr, checkErr)
			return
		}
		defer func() {
			measureErr = errutil.Join(measureErr, admission.check(runCtx, root, "diagnostic-feature-probe-after", probe, false))
		}()
		runDiagnosticClusterFeatureProbe2, runDiagnosticClusterFeatureProbeErr := runDiagnosticClusterFeatureProbe(runCtx, root, probe)
		if runDiagnosticClusterFeatureProbeErr != nil {
			measureErr = errutil.Join(measureErr, runDiagnosticClusterFeatureProbeErr)
			return
		}
		return runDiagnosticClusterFeatureProbe2, measureErr
	}(runCtx)
	if probeError != nil {
		return probeError
	}
	request, evidence, recoveryRequestWithAdmissionErr := temporalRecoveryRequestWithAdmission(runCtx, "temporal_broker", admission)
	if recoveryRequestWithAdmissionErr != nil {
		measureErr = errutil.Join(measureErr, recoveryRequestWithAdmissionErr)
		return
	}
	var pairs []map[string]any
	observe := func(runCtx context.Context, evidence, label string, requests []wasi.Request, results []wasi.Result, runErrors []error) (measureErr error) {
		pair, decodeErr := decodeClusterDiagnosticPair(requests, results)
		var instances []map[string]any
		for index, result := range results {
			name := requests[index].Broker.Node.ID + "-" + label
			if len(result.HelperStderr()) > 64<<10 {
				measureErr = errutil.Join(measureErr, errutil.New("cluster helper stderr capture exceeds transport bound"))
				return
			}
			writeEvidenceErr := temporalWrite(runCtx, filepath.Join(evidence, name+"-helper-stderr.txt"), result.HelperStderr())
			if writeEvidenceErr != nil {
				measureErr = errutil.Join(measureErr, writeEvidenceErr)
				return
			}
			failure := ""
			if runErrors[index] != nil {
				failure = runErrors[index].Error()
			}
			instances = append(instances, map[string]any{"node": requests[index].Broker.Node.ID, "seed": requests[index].Runtime.Seed, "identities": variantIdentities(requests[index], result), "termination": result.Termination, "exit_code": result.ExitCode, "run_error": failure, "profile": pair.Instances[index], "phase_metrics": result.Telemetry, "wait_resource_usage": result.HelperResourceUsage, "authorization_to_terminal_nanos": result.AuthorizationToTerminalNanos})
		}
		pairMeasurement := map[string]any{"label": label, "instances": instances, "measured_sum": pair.MeasuredSum, "measured_helpers": pair.Measured, "expected_helpers": len(results), "complete": pair.Complete}
		if decodeErr != nil {
			pairMeasurement["diagnostic_error"] = decodeErr.Error()
			measureErr = errutil.Join(measureErr, errutil.Join(decodeErr))
		}
		pairs = append(pairs, pairMeasurement)
		writeReportErr := temporalJSON(runCtx, filepath.Join(evidence, label+"-diagnostic-profile.json"), pairMeasurement)
		if writeReportErr != nil {
			measureErr = errutil.Join(measureErr, writeReportErr)
			return
		}
		return
	}
	profilePath := filepath.Join(root, "cluster-parent.pprof")
	profile, err := os.OpenFile(profilePath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		measureErr = errutil.Join(measureErr, err)
		return
	}
	err = pprof.StartCPUProfile(profile)
	if err != nil {
		closeResource(runCtx, profile)
		measureErr = errutil.Join(measureErr, err)
		return
	}
	addCleanup(runCtx, func(runCtx context.Context) (measureErr error) {
		pprof.StopCPUProfile()
		err := profile.Close()
		if err != nil {
			measureErr = errutil.Join(measureErr, err)
		}
		summary, summaryErr := exec.CommandContext(runCtx, input(runCtx, "GOMAD_STOCK_GO"), "tool", "pprof", "-top", profilePath).CombinedOutput()
		writeEvidenceErr2 := temporalWrite(runCtx, filepath.Join(root, "cluster-parent-pprof-top.txt"), summary)
		if writeEvidenceErr2 != nil {
			measureErr = errutil.Join(measureErr, writeEvidenceErr2)
			return
		}
		if summaryErr != nil {
			measureErr = errutil.Join(measureErr, errutil.Join(summaryErr))
		}
		verifyInputsErr := temporalVerifyInputs(runCtx, inputs)
		if verifyInputsErr != nil {
			measureErr = errutil.Join(measureErr, verifyInputsErr)
			return
		}
		checkErr2 := admission.check(runCtx, root, "cluster-final", request, false)
		if checkErr2 != nil {
			measureErr = errutil.Join(measureErr, checkErr2)
			return
		}
		summaryFailure := ""
		if summaryErr != nil {
			summaryFailure = summaryErr.Error()
		}
		environment := map[string]string{"GOMAD_WASM_TEMPORAL_RECOVERY_ROOT": base, "GOMAD_WASM_TEST_HELPER": helper, "GOMAD_WASM_DIAGNOSTIC_HELPER_ADMISSION": admission.Helpers[helper].Receipt, "GOMAD_STOCK_GO": input(runCtx, "GOMAD_STOCK_GO"), "PATH": input(runCtx, "PATH")}
		for _, key := range []string{"GOMAD_WASM_TEMPORAL_RECOVERY_COMPILE_CACHE", "GOMAD_WASM_TEMPORAL_RECOVERY_DIAGNOSTIC_BUDGET", "GOMAD_WASM_TEMPORAL_RECOVERY_EXPECT_MODULE_SHA256", "GOMAD_WASM_TOOLCHAIN_CACHE"} {
			environment[key] = input(runCtx, key)
		}
		temporalBrokerProvenanceSHA256, temporalBrokerProvenanceSHA256Err := fileHash(runCtx, filepath.Join(root, "temporal_broker-provenance.json"))
		if temporalBrokerProvenanceSHA256Err != nil {
			measureErr = errutil.Join(measureErr, temporalBrokerProvenanceSHA256Err)
			return
		}
		profilePathSHA256, profilePathSHA256Err := fileHash(runCtx, profilePath)
		if profilePathSHA256Err != nil {
			measureErr = errutil.Join(measureErr, profilePathSHA256Err)
			return
		}
		writeReportErr2 := temporalJSON(runCtx, filepath.Join(root, "cluster-baseline-report.json"), map[string]any{"host_admission": admission.Receipt, "host_admission_sha256": admission.Inputs[admission.Receipt], "helper_admission": admission.Helpers[helper].Receipt, "helper_admission_sha256": admission.Inputs[admission.Helpers[helper].Receipt], "module_build_receipts": admission.ModuleReceipts, "diagnostic_only": true, "source_commit": strings.TrimSpace(string(commit)), "source_dirty": len(status) != 0, "source_status": string(status), "collector_sources": inputs, "prerequisite_receipts": prerequisites, "host_goos": runtime.GOOS, "host_goarch": runtime.GOARCH, "go_version": runtime.Version(), "host_cpuinfo": cpuinfo, "required_host_measured": admitted, "seeds": []uint64{7}, "selection": "TestTemporalIndependentClustersPartitionHealWholeClusterCrash", "pairs": pairs, "scenario_passed": measureErr == nil && !failed(runCtx) && len(pairs) == 4, "diagnostic_feature_probe": variantIdentities(probe, probeResult), "identities": variantIdentities(request, wasi.Result{}), "module_provenance_sha256": temporalBrokerProvenanceSHA256, "parent_pprof_sha256": profilePathSHA256, "parent_pprof_summary_sha256": digest(summary), "parent_pprof_summary_error": summaryFailure, "parent_executable_sha256": parentSHA256, "parent_pprof_summary_command": []string{input(runCtx, "GOMAD_STOCK_GO"), "tool", "pprof", "-top", profilePath}, "scope": "unsampled diagnostic-profile helpers; parent CPU profile includes scenario, recovery, replay, per-pair admission checks and evidence publication; module preparation, feature probe and final admission excluded", "bucket_scope": "measured_sum excludes unavailable helper profiles; nested buckets overlap ABI/phase spans; RSS high-water is maximum per-process high-water, not concurrent cluster RSS; summed helper wall spans overlap in real time", "rerun": rerunRecord(runCtx, "go", repo, "diagnostic-cluster-baseline", environment, true)})
		if writeReportErr2 != nil {
			measureErr = errutil.Join(measureErr, writeReportErr2)
			return
		}
		return
	})
	independentClustersPartitionHealWholeClusterCrashErr := temporalIndependentClustersPartitionHealWholeClusterCrash(runCtx, request, evidence, temporalPairObserver{Admission: func(runCtx context.Context, root, label string, requests []wasi.Request, before bool) (measureErr error) {
		boundary := "after"
		if before {
			boundary = "before"
		}
		for index, request := range requests {
			checkErr3 := admission.check(runCtx, root, fmt.Sprintf("%s-%d-%s", label, index, boundary), request, before)
			if checkErr3 != nil {
				measureErr = errutil.Join(measureErr, checkErr3)
				return
			}
		}
		return
	}, Observe: observe})
	if independentClustersPartitionHealWholeClusterCrashErr != nil {
		measureErr = errutil.Join(measureErr, independentClustersPartitionHealWholeClusterCrashErr)
		return
	}
	return
}
func runDiagnosticClusterFeatureProbe(runCtx context.Context, root string, request wasi.Request) (measureValue0 wasi.Result, measureErr error) {
	result, err := wasi.Run(runCtx, request)
	writeEvidenceErr := temporalWrite(runCtx, filepath.Join(root, "diagnostic-feature-probe-helper-stderr.txt"), result.HelperStderr())
	if writeEvidenceErr != nil {
		measureErr = writeEvidenceErr
		return
	}
	if err != nil {
		measureErr = err
		return
	}
	requireExitErr := requireExit(runCtx, "diagnostic feature probe", result, 0)
	if requireExitErr != nil {
		measureErr = requireExitErr
		return
	}
	profile, err := decodeDiagnosticReport(result.HelperStderr())
	if err != nil {
		measureErr = err
		return
	}
	writeReportErr := temporalJSON(runCtx, filepath.Join(root, "diagnostic-feature-probe-profile.json"), profile)
	if writeReportErr != nil {
		measureErr = writeReportErr
		return
	}
	return result, measureErr
}
func cooperativeModuleRequest(runCtx context.Context, module, identity string) (measureValue0 wasi.Request, measureErr error) {
	data, err := os.ReadFile(module)
	if err != nil {
		measureErr = err
		return
	}
	imports, pages, binaryInventoryErr := binaryInventory(runCtx, data)
	if binaryInventoryErr != nil {
		measureErr = binaryInventoryErr
		return
	}
	helper, helperErr := helperPath(runCtx)
	if helperErr != nil {
		measureErr = helperErr
		return
	}
	config := standaloneConfig()
	config.Profile = wasi.CooperativeProfile
	config.Clock.ReadStepNanos = 1000
	execution := choice.ExecutionIdentity{TargetSHA256: sha256.Sum256(data), ToolchainBuildKey: identity, GOOS: "wasip1", GOARCH: "wasm", ImplementationSHA256: sha256.Sum256([]byte(identity))}
	cache, temporaryDirectoryErr := temporaryDirectory(runCtx)
	if temporaryDirectoryErr != nil {
		measureErr = temporaryDirectoryErr
		return
	}
	err = os.Chmod(cache, 0700)
	if err != nil {
		measureErr = err
		return
	}
	helperSHA256, helperSHA256Err := fileHash(runCtx, helper)
	if helperSHA256Err != nil {
		measureErr = helperSHA256Err
		return
	}
	moduleSHA256, moduleSHA256Err := fileHash(runCtx, module)
	if moduleSHA256Err != nil {
		measureErr = moduleSHA256Err
		return
	}
	return wasi.Request{HelperPath: helper, HelperSHA256: helperSHA256, ModulePath: module, ModuleSHA256: moduleSHA256, Engine: wasi.PinnedEngine(), Imports: imports, InitialMemoryPages: pages, MemoryBytes: 256 << 20, Fuel: 4000000000, Config: config, Timeout: time.Minute, CompileCache: cache, Runtime: &wasi.RuntimeControl{Seed: 7, Diagnostics: true, Choice: &runnerbackend.ChoiceRequest{Mode: choice.ModeRecord, ExecutionIdentity: execution, Limit: 1 << 20}}}, measureErr
}
func temporalRecoveryRequestWithAdmission(runCtx context.Context, fixture string, admission *diagnosticAdmission) (measureValue0 wasi.Request, measureValue1 string, measureErr error) {
	evidence := input(runCtx, "GOMAD_WASM_TEMPORAL_RECOVERY_ROOT")
	if evidence == "" {
		measureErr = fmt.Errorf("%w: %v", ErrRefused, errutil.New("set GOMAD_WASM_TEMPORAL_RECOVERY_ROOT for the actual Temporal recovery gate"))
		return
	}
	if !filepath.IsAbs(evidence) {
		measureErr = errutil.New("GOMAD_WASM_TEMPORAL_RECOVERY_ROOT must be absolute")
		return
	}
	started, before := time.Now(), hostexec.CurrentProcessResourceUsage()
	setInput(runCtx, "GOMAD_WASM_PROTOTYPE_ROOT", evidence)
	root, err := absolute(runCtx, "../../../../../..")
	if err != nil {
		measureErr = err
		return
	}
	fixtureBody, err := os.ReadFile(filepath.Join(root, "tools/gomad/internal/backends/wasm/testdata/temporal_storage/recovery_test.go"))
	if err != nil {
		measureErr = err
		return
	}
	if fixture == "temporal_broker" {
		brokerBody, err := os.ReadFile(filepath.Join(root, "tools/gomad/internal/backends/wasm/testdata/temporal_broker/replication_test.go"))
		if err != nil {
			measureErr = err
			return
		}
		fixtureBody = append(fixtureBody, brokerBody...)
	}
	staged := filepath.Join(".tmp", "wasm-backend-temporal", fixture+"-"+digest(fixtureBody)[:16])
	if fixture == "temporal_broker" {
		stageBrokerFixtureErr := temporalStageBrokerFixture(runCtx, root, staged)
		if stageBrokerFixtureErr != nil {
			measureErr = stageBrokerFixtureErr
			return
		}
	} else {
		err := temporalStageFixture(root, filepath.Join("tools", "gomad", "internal", "backends", "wasm", "testdata", fixture), root, staged)
		if err != nil {
			measureErr = err
			return
		}
	}
	compiler, overlay, runtimeIdentity, prototypeToolchainErr := prototypeToolchain(runCtx)
	if prototypeToolchainErr != nil {
		measureErr = prototypeToolchainErr
		return
	}
	buildEnvironment, buildEnvironmentErr := temporalBuildEnvironment(runCtx, compiler)
	if buildEnvironmentErr != nil {
		measureErr = buildEnvironmentErr
		return
	}
	listed, commandErr := temporalCommand(runCtx, evidence, fixture+"-inputs", root, buildEnvironment, []string{compiler, "list", "-p=1", "-mod=readonly", "-tags", "test_dep,disable_grpc_modules,sqlite3_dotlk", "-deps", "-test", "-json", "./" + filepath.ToSlash(staged)})
	if commandErr != nil {
		measureErr = commandErr
		return
	}
	inputs, sourceInputsErr := temporalSourceInputs(runCtx, listed.Stdout.RawBytes)
	if sourceInputsErr != nil {
		measureErr = sourceInputsErr
		return
	}
	overlaySHA256, overlaySHA256Err := fileHash(runCtx, overlay)
	if overlaySHA256Err != nil {
		measureErr = overlaySHA256Err
		return
	}
	inputs[overlay] = overlaySHA256
	overlayBytes, err := os.ReadFile(overlay)
	if err != nil {
		measureErr = err
		return
	}
	var overlayMap struct{ Replace map[string]string }
	err = json.Unmarshal(overlayBytes, &overlayMap)
	if err != nil {
		measureErr = err
		return
	}
	for _, replacement := range overlayMap.Replace {
		if replacement != "" {
			replacementSHA256, replacementSHA256Err := fileHash(runCtx, replacement)
			if replacementSHA256Err != nil {
				measureErr = replacementSHA256Err
				return
			}
			inputs[replacement] = replacementSHA256
		}
	}
	for _, name := range []string{"go.mod", "go.sum", "tools/gomad/internal/backends/wasm/testdata/temporal_storage/recovery_test.go"} {
		nameSHA256, nameSHA256Err := fileHash(runCtx, filepath.Join(root, name))
		if nameSHA256Err != nil {
			measureErr = nameSHA256Err
			return
		}
		inputs[filepath.Join(root, name)] = nameSHA256
	}
	if fixture == "temporal_broker" {
		file := filepath.Join(root, "tools/gomad/internal/backends/wasm/testdata/temporal_broker/replication_test.go")
		fileSHA256, fileSHA256Err := fileHash(runCtx, file)
		if fileSHA256Err != nil {
			measureErr = fileSHA256Err
			return
		}
		inputs[file] = fileSHA256
	}
	addCleanup(runCtx, func(runCtx context.Context) (measureErr error) {
		verifyInputsErr := temporalVerifyInputs(runCtx, inputs)
		if verifyInputsErr != nil {
			measureErr = verifyInputsErr
			return
		}
		return
	})
	module := filepath.Join(evidence, fixture+".wasm")
	if admission == nil {
		buildEnvironment2, buildEnvironmentErr2 := temporalBuildEnvironment(runCtx, compiler)
		if buildEnvironmentErr2 != nil {
			measureErr = buildEnvironmentErr2
			return
		}
		command, commandErr2 := temporalCommand(runCtx, evidence, fixture+"-compile", root, append(buildEnvironment2, "GOROOT="+filepath.Dir(filepath.Dir(compiler))), []string{compiler, "test", "-c", "-p=1", "-mod=readonly", "-trimpath", "-overlay", overlay, "-o", module, "-tags", "test_dep,disable_grpc_modules,sqlite3_dotlk", "./" + filepath.ToSlash(staged)})
		if commandErr2 != nil {
			measureErr = commandErr2
			return
		}
		_ = command
	} else {
		moduleTargetErr := admission.buildModuleTarget(runCtx, root, module, "", "./"+filepath.ToSlash(staged), "test", []string{"test_dep", "disable_grpc_modules", "sqlite3_dotlk"})
		if moduleTargetErr != nil {
			measureErr = moduleTargetErr
			return
		}
	}
	verifyInputsErr2 := temporalVerifyInputs(runCtx, inputs)
	if verifyInputsErr2 != nil {
		measureErr = verifyInputsErr2
		return
	}
	expected := input(runCtx, "GOMAD_WASM_TEMPORAL_RECOVERY_EXPECT_MODULE_SHA256")
	moduleSHA256, moduleSHA256Err := fileHash(runCtx, module)
	if moduleSHA256Err != nil {
		measureErr = moduleSHA256Err
		return
	}
	if expected != "" && moduleSHA256 != expected {
		moduleSHA2562, moduleSHA256Err2 := fileHash(runCtx, module)
		if moduleSHA256Err2 != nil {
			measureErr = moduleSHA256Err2
			return
		}
		measureErr = fmt.Errorf("prepared Temporal guest differs from retained module: expected %s, actual %s", expected, moduleSHA2562)
		return
	}
	moduleSHA2563, moduleSHA256Err3 := fileHash(runCtx, module)
	if moduleSHA256Err3 != nil {
		measureErr = moduleSHA256Err3
		return
	}
	helper, helperErr := helperPath(runCtx)
	if helperErr != nil {
		measureErr = helperErr
		return
	}
	helperSHA256, helperSHA256Err := fileHash(runCtx, helper)
	if helperSHA256Err != nil {
		measureErr = helperSHA256Err
		return
	}
	writeReportErr := temporalJSON(runCtx, filepath.Join(evidence, fixture+"-provenance.json"), struct {
		Inputs                                      map[string]string
		ModuleSHA256, HelperSHA256, RuntimeIdentity string
		Engine                                      wasi.EngineIdentity
	}{inputs, moduleSHA2563, helperSHA256, runtimeIdentity, wasi.PinnedEngine()})
	if writeReportErr != nil {
		measureErr = writeReportErr
		return
	}
	cfg, err := temporalRecoveryConfig(root)
	if err != nil {
		measureErr = err
		return
	}
	request, requestErr := prototypeRequest(runCtx, module, cfg, 7, 15*time.Minute)
	if requestErr != nil {
		measureErr = requestErr
		return
	}
	request.CompileCache, err = temporalCompileCachePath(evidence, input(runCtx, "GOMAD_WASM_TEMPORAL_RECOVERY_COMPILE_CACHE"))
	if err != nil {
		measureErr = err
		return
	}
	err = os.MkdirAll(request.CompileCache, 0700)
	if err != nil {
		measureErr = err
		return
	}
	err = os.Chmod(request.CompileCache, 0700)
	if err != nil {
		measureErr = err
		return
	}
	writeReportErr2 := temporalJSON(runCtx, filepath.Join(evidence, fixture+"-preparation-operator.json"), struct {
		ElapsedNanos  int64
		Before, After *hostexec.ResourceUsage
	}{time.Since(started).Nanoseconds(), before, hostexec.CurrentProcessResourceUsage()})
	if writeReportErr2 != nil {
		measureErr = writeReportErr2
		return
	}
	return request, evidence, measureErr
}

type temporalPairObserver struct {
	Admission func(context.Context, string, string, []wasi.Request, bool) error
	Observe   func(context.Context, string, string, []wasi.Request, []wasi.Result, []error) error
	Cluster   func(context.Context, string, *wasi.ClusterResult) error
}

func temporalIndependentClustersPartitionHealWholeClusterCrash(runCtx context.Context, request wasi.Request, evidence string, observers ...temporalPairObserver) (measureErr error) {
	inProcess := request.DiagnosticConfigurationRevision() == "" && request.GuestProfile() == nil
	request.Config.Args = []string{"replication.test", "-test.v", "-test.count=1", "-test.timeout=1h", "-test.run=^TestIndependentClusterReplicationRecovery$", "-persistenceType=sql", "-persistenceDriver=sqlite"}
	network := &wasi.NetworkModelV2{Profile: wasi.NetworkStreamProfile, Limits: netmodel.Limits{Listeners: 128, Connections: 4096, Bytes: 128 << 20, ConnectionBytes: 4 << 20, StallNanos: 10_000_000_000, Transitions: 1 << 20}}
	request.Config.NetworkV2 = network
	brokerConfig := temporalBrokerConfiguration(network)
	brokerConfig.Seed = request.Runtime.Seed
	writeReportErr := temporalJSON(runCtx, filepath.Join(evidence, "two-cluster-broker-config.json"), brokerConfig)
	if writeReportErr != nil {
		measureErr = errutil.Join(measureErr, writeReportErr)
		return
	}
	b, err := wasi.NewBroker(brokerConfig)
	if err != nil {
		measureErr = errutil.Join(measureErr, err)
		return
	}
	addCleanup(runCtx, func(runCtx context.Context) (measureErr error) {
		if inProcess {
			return
		}
		writeEvidenceErr := temporalWrite(runCtx, filepath.Join(evidence, "two-cluster-broker-idle.jsonl"), b.Diagnostics())
		if writeEvidenceErr != nil {
			measureErr = errutil.Join(measureErr, writeEvidenceErr)
			return
		}
		snapshot, err := b.NetworkSnapshot()
		if err != nil {
			measureErr = errutil.Join(measureErr, err)
			return
		}
		writeReportErr2 := temporalJSON(runCtx, filepath.Join(evidence, "two-cluster-network.json"), snapshot)
		if writeReportErr2 != nil {
			measureErr = errutil.Join(measureErr, writeReportErr2)
			return
		}
		return
	})
	progressContext, stopProgress := context.WithCancel(runCtx)
	var progress sync.WaitGroup
	progress.Go(func() {
		callbackValue0 := (func(runCtx context.Context) (measureErr error) {
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
					err := os.WriteFile(filepath.Join(evidence, "two-cluster-broker-live.jsonl"), b.Diagnostics(), 0600)
					if err != nil {
						measureErr = errutil.Join(measureErr, err)
					}
				}
			}
		})(runCtx)
		recordFailure(runCtx, callbackValue0)
	})
	addCleanup(runCtx, func(context.Context) error {
		stopProgress()
		progress.Wait()
		return nil
	})
	names := []string{"active", "standby"}
	err = b.CrashAfterOutputs(names, []byte(temporalObservationPrefix))
	if err != nil {
		measureErr = errutil.Join(measureErr, err)
		return
	}
	requests := make([]wasi.Request, 2)
	commitRequests := make([]wasi.Request, 2)
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
		retainRequestErr := temporalRetainRequest(runCtx, evidence, name+"-cluster-commit", requests[i])
		if retainRequestErr != nil {
			measureErr = errutil.Join(measureErr, retainRequestErr)
			return
		}
		commitRequests[i] = requests[i]
		commitRequests[i].Config.Environment = append([]string(nil), requests[i].Config.Environment...)
	}
	var pair *temporalClusterPair
	if inProcess {
		openClusterPair, openClusterPairErr := temporalOpenClusterPair(runCtx, brokerConfig, requests, temporalReopenRequests(requests))
		if openClusterPairErr != nil {
			measureErr = errutil.Join(measureErr, openClusterPairErr)
			return
		}
		pair = openClusterPair
	}
	runPair := func(runCtx context.Context, label string, requests []wasi.Request, incarnation uint64, warm bool) (measureValue0 []wasi.Result, measureValue1 []error, measureErr error) {
		if pair != nil {
			run2, run22, runErr := pair.run(runCtx, evidence, label, requests, incarnation, observers...)
			if runErr != nil {
				measureErr = errutil.Join(measureErr, runErr)
				return
			}
			return run2, run22, measureErr
		}
		runPair2, runPair22, runPairErr := temporalRunPair(runCtx, evidence, label, requests, warm, observers...)
		if runPairErr != nil {
			measureErr = errutil.Join(measureErr, runPairErr)
			return
		}
		return runPair2, runPair22, measureErr
	}
	results, errors, runPairErr2 := runPair(runCtx, "cluster-crash", requests, 1, true)
	if runPairErr2 != nil {
		measureErr = errutil.Join(measureErr, runPairErr2)
		return
	}
	crashed := append([]wasi.Result(nil), results...)
	var expected []byte
	identities := make([]string, 2)
	original := make([]temporalRecoveryObservation, 2)
	converged := map[string][]byte{}
	for i, name := range names {
		if errors[i] != nil || results[i].Termination != "crash" || !results[i].Reaped || results[i].StorageImage == nil {
			measureErr = errutil.Join(measureErr, fmt.Errorf("independent %s crash: termination=%s message=%s reaped=%v image=%v err=%v stdout=%s stderr=%s", name, results[i].Termination, results[i].Message, results[i].Reaped, results[i].StorageImage != nil, errors[i], results[i].Stdout, results[i].Stderr))
			return
		}
		value, recoveryValueErr := temporalRecoveryValue(runCtx, results[i].Stdout)
		if recoveryValueErr != nil {
			measureErr = errutil.Join(measureErr, recoveryValueErr)
			return
		}
		original[i] = value
		converged[name] = value.History
		identities[i] = value.ClusterID
		if i == 0 {
			expected = value.History
		} else if !bytes.Equal(expected, value.History) {
			measureErr = errutil.Join(measureErr, errutil.New("independently observed histories did not converge after directional heal"))
			return
		}
		requests[i].StorageImage = results[i].StorageImage
		requests[i].StdoutMirror = nil
		requests[i].Config.Environment[len(request.Config.Environment)+1] = "GOMAD_TEMPORAL_PHASE=reopen"
		if inProcess {
			requests[i].Broker = &wasi.BrokerSession{Node: wasi.BrokerNode{ID: name, Incarnation: 2}}
		} else {
			requests[i].Broker, err = b.Restart(name)
			if err != nil {
				measureErr = errutil.Join(measureErr, err)
				return
			}
		}
		retainRequestErr2 := temporalRetainRequest(runCtx, evidence, name+"-cluster-reopen", requests[i])
		if retainRequestErr2 != nil {
			measureErr = errutil.Join(measureErr, retainRequestErr2)
			return
		}
	}
	if identities[0] == identities[1] {
		measureErr = errutil.Join(measureErr, errutil.New("independent Temporal clusters have the same identity"))
		return
	}
	convergence, err := gomadsim.EventualConvergence("temporal.replication.healed", converged, 1<<20)
	if err != nil || !convergence.Passed {
		measureErr = errutil.Join(measureErr, fmt.Errorf("detached Temporal convergence oracle: passed=%v err=%v", convergence.Passed, err))
		return
	}
	writeReportErr3 := temporalJSON(runCtx, filepath.Join(evidence, "temporal-replication-healed-oracle.json"), convergence)
	if writeReportErr3 != nil {
		measureErr = errutil.Join(measureErr, writeReportErr3)
		return
	}
	writeReportErr4 := temporalJSON(runCtx, filepath.Join(evidence, "temporal-replication-healed-oracle-inputs.json"), converged)
	if writeReportErr4 != nil {
		measureErr = errutil.Join(measureErr, writeReportErr4)
		return
	}
	runPair3, runPair23, runPairErr3 := runPair(runCtx, "cluster-reopened", requests, 2, false)
	if runPairErr3 != nil {
		measureErr = errutil.Join(measureErr, runPairErr3)
		return
	}
	results, errors = runPair3, runPair23
	for i, name := range names {
		if errors[i] != nil || results[i].Termination != "exit" || results[i].ExitCode == nil || *results[i].ExitCode != 0 || !results[i].Reaped {
			measureErr = errutil.Join(measureErr, fmt.Errorf("independent %s reopen: termination=%s message=%s err=%v stdout=%s stderr=%s", name, results[i].Termination, results[i].Message, errors[i], results[i].Stdout, results[i].Stderr))
			return
		}
		value, recoveryValueErr2 := temporalRecoveryValue(runCtx, results[i].Stdout)
		if recoveryValueErr2 != nil {
			measureErr = errutil.Join(measureErr, recoveryValueErr2)
			return
		}
		historyOraclesErr := temporalHistoryOracles(runCtx, evidence, "temporal.replication."+name+"-recovered", original[i], value)
		if historyOraclesErr != nil {
			measureErr = errutil.Join(measureErr, historyOraclesErr)
			return
		}
		if !reflect.DeepEqual(expected, value.History) || value.ClusterID != identities[i] {
			measureErr = errutil.Join(measureErr, fmt.Errorf("committed %s history changed after whole-cluster crash", name))
			return
		}
	}
	var encodedWorld []byte
	var finalNetwork netmodel.Record
	networkOwner := "go-broker-reference"
	networkOwnerImplementation := netmodel.ImplementationSHA256()
	if inProcess {
		finished, finishErr := pair.finish(runCtx, evidence, "two-cluster", observers...)
		if finishErr != nil {
			measureErr = errutil.Join(measureErr, finishErr)
			return
		}
		encodedWorld = finished.World
		finalNetwork = finished.Network
		networkOwner = "rust-cluster"
		networkOwnerImplementation = request.HelperSHA256
		writeReportErr5 := temporalJSON(runCtx, filepath.Join(evidence, "two-cluster-network.json"), finished.Network.Snapshot)
		if writeReportErr5 != nil {
			measureErr = errutil.Join(measureErr, writeReportErr5)
			return
		}
		writeReportErr6 := temporalJSON(runCtx, filepath.Join(evidence, "two-cluster-network-record.json"), finished.Network)
		if writeReportErr6 != nil {
			measureErr = errutil.Join(measureErr, writeReportErr6)
			return
		}
	} else {
		encodedWorld, err = b.EncodeWorldEvidence()
		if err != nil {
			measureErr = errutil.Join(measureErr, err)
			return
		}
		finalNetwork, err = b.NetworkRecord()
		if err != nil {
			measureErr = errutil.Join(measureErr, err)
			return
		}
	}
	writeEvidenceErr2 := temporalWrite(runCtx, filepath.Join(evidence, "two-cluster-world.json"), encodedWorld)
	if writeEvidenceErr2 != nil {
		measureErr = errutil.Join(measureErr, writeEvidenceErr2)
		return
	}
	if inProcess {
		summary, err := world.DecodeCompressedSnapshot(encodedWorld, world.CompressedSnapshotProfile())
		if err != nil {
			measureErr = errutil.Join(measureErr, err)
			return
		}
		writeReportErr7 := temporalJSON(runCtx, filepath.Join(evidence, "two-cluster-world-cardinality.json"), summary)
		if writeReportErr7 != nil {
			measureErr = errutil.Join(measureErr, writeReportErr7)
			return
		}
	} else {
		writeReportErr8 := temporalJSON(runCtx, filepath.Join(evidence, "two-cluster-world-cardinality.json"), b.DiagnosticCardinality())
		if writeReportErr8 != nil {
			measureErr = errutil.Join(measureErr, writeReportErr8)
			return
		}
	}
	writeReportErr9 := temporalJSON(runCtx, filepath.Join(evidence, "two-cluster-owner.json"), map[string]any{"network_owner": networkOwner, "network_owner_implementation_sha256": networkOwnerImplementation, "helper_sha256": request.HelperSHA256, "seed": request.Runtime.Seed})
	if writeReportErr9 != nil {
		measureErr = errutil.Join(measureErr, writeReportErr9)
		return
	}
	replayClustersErr := temporalReplayClusters(runCtx, evidence, brokerConfig, commitRequests, crashed, results, encodedWorld, finalNetwork, observers...)
	if replayClustersErr != nil {
		measureErr = errutil.Join(measureErr, replayClustersErr)
		return
	}
	logf(runCtx, "independent Temporal clusters recovered after directional partition/heal\n")
	return
}
func temporalObservedPair(runCtx context.Context, evidence, label string, requests []wasi.Request, observers []temporalPairObserver, dispatch func() ([]wasi.Result, []error)) (results []wasi.Result, failures []error, measureErr error) {
	for _, observer := range observers {
		if observer.Admission != nil {
			if err := observer.Admission(runCtx, evidence, label, requests, true); err != nil {
				return nil, nil, err
			}
		}
	}
	results, failures = dispatch()
	for _, observer := range observers {
		if observer.Admission != nil {
			measureErr = errutil.Join(measureErr, observer.Admission(runCtx, evidence, label, requests, false))
		}
	}
	if measureErr != nil {
		return results, failures, measureErr
	}
	for _, observer := range observers {
		if observer.Observe != nil {
			if err := observer.Observe(runCtx, evidence, label, requests, results, failures); err != nil {
				return results, failures, err
			}
		}
	}
	return
}
func temporalRunPairExecution(runCtx context.Context, evidence, label string, requests []wasi.Request, warmFirst bool) (measureValue0 []wasi.Result, measureValue1 []error, measureErr error) {
	return temporalRunPairWith(runCtx, evidence, label, requests, warmFirst, wasi.Run)
}

func temporalRunPairWith(runCtx context.Context, evidence, label string, requests []wasi.Request, warmFirst bool, execute func(context.Context, wasi.Request) (wasi.Result, error)) (measureValue0 []wasi.Result, measureValue1 []error, measureErr error) {
	budget, err := temporalPairBudget(input(runCtx, "GOMAD_WASM_TEMPORAL_RECOVERY_DIAGNOSTIC_BUDGET"))
	if err != nil {
		measureErr = errutil.Join(measureErr, err)
		return
	}
	started := time.Now()
	ctx, cancel := context.WithTimeout(runCtx, budget)
	defer cancel()
	results, errors := make([]wasi.Result, len(requests)), make([]error, len(requests))
	warmed, firstFinished := make(chan struct{}), make(chan struct{})
	var group sync.WaitGroup
	for i := range requests {
		observer := &temporalGuestWarm{ctx: runCtx, node: requests[i].Broker.Node.ID}
		if i == 0 && warmFirst {
			observer.ready = warmed
		}
		requests[i].StdoutMirror = observer
		if i == 1 && warmFirst {
			select {
			case <-warmed:
			case <-firstFinished:
			case <-ctx.Done():
				cancel()
			}
		}
		group.Go(func() {
			callbackValue0 := (func(runCtx context.Context) (measureErr error) {
				results[i], errors[i] = execute(ctx, requests[i])
				name := requests[i].Broker.Node.ID
				retainRecoveryRunErr := temporalRetainRecoveryRun(runCtx, evidence, name+"-"+label, results[i])
				if retainRecoveryRunErr != nil {
					measureErr = errutil.Join(measureErr, retainRecoveryRunErr)
					return
				}
				logf(runCtx, "%s %s terminated: %s %s", name, label, results[i].Termination, results[i].Message)
				if i == 0 {
					close(firstFinished)
				}
				if errors[i] != nil || results[i].Termination != "exit" && results[i].Termination != "crash" || results[i].Termination == "exit" && (results[i].ExitCode == nil || *results[i].ExitCode != 0) {
					cancel()
				}
				return
			})(runCtx)
			recordFailure(runCtx, callbackValue0)
		})
	}
	group.Wait()
	writeReportErr := temporalJSON(runCtx, filepath.Join(evidence, label+"-host-budget.json"), struct {
		BudgetNanos, ElapsedNanos int64
		Expired                   bool
	}{int64(budget), time.Since(started).Nanoseconds(), ctx.Err() == context.DeadlineExceeded})
	if writeReportErr != nil {
		measureErr = errutil.Join(measureErr, writeReportErr)
		return
	}
	return results, errors, measureErr
}
func terminalExitAdmitHelperFeatures(runCtx context.Context, root, receipt string, features []string) (measureValue0 string, measureValue1 string, measureErr error) {
	var build terminalExitHelperBuild
	mustTerminalExitRead2, mustTerminalExitReadErr := mustTerminalExitRead(runCtx, receipt)
	if mustTerminalExitReadErr != nil {
		measureErr = mustTerminalExitReadErr
		return
	}
	err := json.Unmarshal(mustTerminalExitRead2, &build)
	if err != nil {
		measureErr = err
		return
	}
	crate, err := absolute(runCtx, "../wasmhost")
	if err != nil {
		measureErr = err
		return
	}
	request := wasi.Request{HelperPath: build.Helper, HelperSHA256: build.HelperSHA256}
	inputs, terminalExitBuildInputsFeaturesErr := terminalExitBuildInputsFeatures(runCtx, receipt, request, crate, features)
	if terminalExitBuildInputsFeaturesErr != nil {
		measureErr = terminalExitBuildInputsFeaturesErr
		return
	}
	evidence, err := diagnosticRunRoot(root, "admitted-helper-")
	if err != nil {
		measureErr = err
		return
	}
	helper := filepath.Join(evidence, "gomad-wasmhost")
	terminalExitStageHelperErr := terminalExitStageHelper(runCtx, build.Helper, helper, build.HelperSHA256)
	if terminalExitStageHelperErr != nil {
		measureErr = terminalExitStageHelperErr
		return
	}
	verifyInputsErr := temporalVerifyInputs(runCtx, inputs)
	if verifyInputsErr != nil {
		measureErr = verifyInputsErr
		return
	}
	terminalExitBuildInputsFeatures2, terminalExitBuildInputsFeaturesErr2 := terminalExitBuildInputsFeatures(runCtx, receipt, request, crate, features)
	if terminalExitBuildInputsFeaturesErr2 != nil {
		measureErr = terminalExitBuildInputsFeaturesErr2
		return
	}
	_ = terminalExitBuildInputsFeatures2
	admission := filepath.Join(evidence, "helper-admission.json")
	receiptSHA256, receiptSHA256Err := fileHash(runCtx, receipt)
	if receiptSHA256Err != nil {
		measureErr = receiptSHA256Err
		return
	}
	helperSHA256, helperSHA256Err := fileHash(runCtx, helper)
	if helperSHA256Err != nil {
		measureErr = helperSHA256Err
		return
	}
	writeReportErr := temporalJSON(runCtx, admission, terminalExitHelperAdmission{BuildReceipt: receipt, BuildReceiptSHA256: receiptSHA256, CargoArtifact: build.Helper, CargoArtifactSHA256: build.HelperSHA256, Helper: helper, HelperSHA256: helperSHA256})
	if writeReportErr != nil {
		measureErr = writeReportErr
		return
	}
	helperSHA2562, helperSHA256Err2 := fileHash(runCtx, helper)
	if helperSHA256Err2 != nil {
		measureErr = helperSHA256Err2
		return
	}
	terminalExitHelperInputsFeatures2, terminalExitHelperInputsFeaturesErr := terminalExitHelperInputsFeatures(runCtx, admission, wasi.Request{HelperPath: helper, HelperSHA256: helperSHA2562}, crate, features)
	if terminalExitHelperInputsFeaturesErr != nil {
		measureErr = terminalExitHelperInputsFeaturesErr
		return
	}
	_ = terminalExitHelperInputsFeatures2
	return helper, admission, measureErr
}
func terminalExitBuildHelper(runCtx context.Context, root string, features []string) (measureErr error) {
	if !filepath.IsAbs(root) {
		measureErr = errutil.New("helper build evidence root must be absolute")
		return
	}
	evidence, err := diagnosticRunRoot(root, "release-helper-")
	if err != nil {
		measureErr = err
		return
	}
	crate, err := absolute(runCtx, "../wasmhost")
	if err != nil {
		measureErr = err
		return
	}
	inputs, terminalExitRustInputsErr := terminalExitRustInputs(runCtx, crate)
	if terminalExitRustInputsErr != nil {
		measureErr = terminalExitRustInputsErr
		return
	}
	engine, err := os.ReadFile(filepath.Join(crate, "src/engine.rs"))
	if err != nil || bytes.Contains(engine, []byte("std::mem::forget")) || !bytes.Contains(engine, []byte("native_unwind_info(false)")) {
		measureErr = errutil.New(fmt.Sprint("release helper must use the no-unwind source candidate without the leak", err))
		return
	}
	var environment []string
	for _, entry := range environ(runCtx) {
		name, _, _ := strings.Cut(entry, "=")
		if slices.Contains([]string{"PATH", "HOME", "CARGO_HOME", "RUSTUP_HOME", "TMPDIR"}, name) {
			environment = append(environment, entry)
		}
	}
	environment = append(environment, "CARGO_TARGET_DIR="+filepath.Join(evidence, "target"))
	command2, commandErr := temporalCommand(runCtx, evidence, "rustc-path", crate, environment, []string{"rustup", "which", "--toolchain", "1.99.1", "rustc"})
	if commandErr != nil {
		measureErr = commandErr
		return
	}
	rustcPath := strings.TrimSpace(string(command2.Stdout.RawBytes))
	command3, commandErr2 := temporalCommand(runCtx, evidence, "cargo-path", crate, environment, []string{"rustup", "which", "--toolchain", "1.99.1", "cargo"})
	if commandErr2 != nil {
		measureErr = commandErr2
		return
	}
	cargoPath := strings.TrimSpace(string(command3.Stdout.RawBytes))
	rustcPathSHA256, rustcPathSHA256Err := fileHash(runCtx, rustcPath)
	if rustcPathSHA256Err != nil {
		measureErr = rustcPathSHA256Err
		return
	}
	cargoPathSHA256, cargoPathSHA256Err := fileHash(runCtx, cargoPath)
	if cargoPathSHA256Err != nil {
		measureErr = cargoPathSHA256Err
		return
	}
	inputs[rustcPath], inputs[cargoPath] = rustcPathSHA256, cargoPathSHA256
	rustcVersion, commandErr3 := temporalCommand(runCtx, evidence, "rustc-version", crate, environment, []string{rustcPath, "-vV"})
	if commandErr3 != nil {
		measureErr = commandErr3
		return
	}
	cargoVersion, commandErr4 := temporalCommand(runCtx, evidence, "cargo-version", crate, environment, []string{cargoPath, "--version"})
	if commandErr4 != nil {
		measureErr = commandErr4
		return
	}
	if !strings.HasPrefix(string(rustcVersion.Stdout.RawBytes), "rustc 1.99.1 ") || !strings.HasPrefix(string(cargoVersion.Stdout.RawBytes), "cargo 1.99.1 ") {
		measureErr = errutil.New("release helper requires pinned Rust/Cargo 1.99.1")
		return
	}
	environment = append(environment, "RUSTC="+rustcPath, "RUSTC_WRAPPER=", "RUSTC_WORKSPACE_WRAPPER=")
	command := terminalExitCargoCommand(cargoPath, crate, features)
	build, commandErr5 := temporalCommand(runCtx, evidence, "cargo-build", crate, environment, command)
	if commandErr5 != nil {
		measureErr = commandErr5
		return
	}
	helper, err := terminalExitCargoHelperFeatures(build.Stdout.RawBytes, features)
	if err != nil {
		measureErr = err
		return
	}
	if helper != filepath.Join(evidence, "target/release/gomad-wasmhost") {
		measureErr = errutil.New("cargo did not produce the fresh isolated release executable")
		return
	}
	current, terminalExitRustInputsErr2 := terminalExitRustInputs(runCtx, crate)
	if terminalExitRustInputsErr2 != nil {
		measureErr = terminalExitRustInputsErr2
		return
	}
	rustcPathSHA2562, rustcPathSHA256Err2 := fileHash(runCtx, rustcPath)
	if rustcPathSHA256Err2 != nil {
		measureErr = rustcPathSHA256Err2
		return
	}
	cargoPathSHA2562, cargoPathSHA256Err2 := fileHash(runCtx, cargoPath)
	if cargoPathSHA256Err2 != nil {
		measureErr = cargoPathSHA256Err2
		return
	}
	current[rustcPath], current[cargoPath] = rustcPathSHA2562, cargoPathSHA2562
	if !maps.Equal(inputs, current) {
		measureErr = errutil.New("rust semantic inputs changed during build")
		return
	}
	for _, name := range []string{"cargo-build.stdout", "cargo-build.json", "rustc-version.stdout", "cargo-version.stdout"} {
		path := filepath.Join(evidence, name)
		pathSHA256, pathSHA256Err := fileHash(runCtx, path)
		if pathSHA256Err != nil {
			measureErr = pathSHA256Err
			return
		}
		inputs[path] = pathSHA256
	}
	receipt := filepath.Join(evidence, "helper-build.json")
	helperSHA256, helperSHA256Err := fileHash(runCtx, helper)
	if helperSHA256Err != nil {
		measureErr = helperSHA256Err
		return
	}
	writeReportErr := temporalJSON(runCtx, receipt, terminalExitHelperBuild{Crate: crate, Helper: helper, HelperSHA256: helperSHA256, RustcPath: rustcPath, Inputs: inputs, Command: command, Environment: environment, Features: features, RustcVersion: string(rustcVersion.Stdout.RawBytes), CargoVersion: string(cargoVersion.Stdout.RawBytes)})
	if writeReportErr != nil {
		measureErr = writeReportErr
		return
	}
	admitted, admission, terminalExitAdmitHelperFeaturesErr := terminalExitAdmitHelperFeatures(runCtx, evidence, receipt, features)
	if terminalExitAdmitHelperFeaturesErr != nil {
		measureErr = terminalExitAdmitHelperFeaturesErr
		return
	}
	admittedSHA256, admittedSHA256Err := fileHash(runCtx, admitted)
	if admittedSHA256Err != nil {
		measureErr = admittedSHA256Err
		return
	}
	logf(runCtx, "release Cargo artifact=%s build=%s; admitted helper=%s receipt=%s sha256=%s", helper, receipt, admitted, admission, admittedSHA256)
	return
}
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
		err := decoder.Decode(&frame)
		if err == io.EOF {
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
func terminalExitBuildInputsFeatures(runCtx context.Context, receipt string, request wasi.Request, crate string, features []string) (measureValue0 map[string]string, measureErr error) {
	data, err := os.ReadFile(receipt)
	if err != nil {
		measureErr = err
		return
	}
	var build terminalExitHelperBuild
	err = json.Unmarshal(data, &build)
	if err != nil {
		measureErr = err
		return
	}
	if build.Crate != crate || build.Helper != request.HelperPath || build.HelperSHA256 != request.HelperSHA256 || build.Features == nil || !slices.Equal(build.Features, features) || !slices.Contains(build.Command, "--release") || !slices.Contains(build.Command, "--locked") || !slices.Contains(build.Command, "--no-default-features") || !strings.HasPrefix(build.RustcVersion, "rustc 1.99.1 ") || !strings.HasPrefix(build.CargoVersion, "cargo 1.99.1 ") {
		measureErr = errutil.New("helper receipt does not bind the pinned release build and measured executable")
		return
	}
	if !terminalExitCompilerEnvironment(build) {
		measureErr = errutil.New("cargo must use the retained pinned rustc executable with compiler wrappers disabled")
		return
	}
	current, terminalExitRustInputsErr := terminalExitRustInputs(runCtx, crate)
	if terminalExitRustInputsErr != nil {
		measureErr = terminalExitRustInputsErr
		return
	}
	for path := range build.Inputs {
		_, found := current[path]
		if !found {
			pathSHA256, pathSHA256Err := fileHash(runCtx, path)
			if pathSHA256Err != nil {
				measureErr = pathSHA256Err
				return
			}
			current[path] = pathSHA256
		}
	}
	if !maps.Equal(current, build.Inputs) {
		measureErr = errutil.New("helper Rust semantic sources, Cargo inputs or toolchain changed after build")
		return
	}
	cargoOutput, err := os.ReadFile(filepath.Join(filepath.Dir(receipt), "cargo-build.stdout"))
	if err != nil {
		measureErr = err
		return
	}
	helper, err := terminalExitCargoHelperFeatures(cargoOutput, features)
	if err != nil || helper != request.HelperPath {
		measureErr = errutil.New(fmt.Sprint("Cargo artifact does not bind the measured release helper", err))
		return
	}
	inputs := maps.Clone(build.Inputs)
	receiptSHA256, receiptSHA256Err := fileHash(runCtx, receipt)
	if receiptSHA256Err != nil {
		measureErr = receiptSHA256Err
		return
	}
	inputs[receipt], inputs[request.HelperPath] = receiptSHA256, request.HelperSHA256
	verifyInputsErr := temporalVerifyInputs(runCtx, inputs)
	if verifyInputsErr != nil {
		measureErr = verifyInputsErr
		return
	}
	return inputs, measureErr
}
func terminalExitHelperInputsFeatures(runCtx context.Context, receipt string, request wasi.Request, crate string, features []string) (measureValue0 map[string]string, measureErr error) {
	var admission terminalExitHelperAdmission
	mustTerminalExitRead2, mustTerminalExitReadErr := mustTerminalExitRead(runCtx, receipt)
	if mustTerminalExitReadErr != nil {
		measureErr = mustTerminalExitReadErr
		return
	}
	err := json.Unmarshal(mustTerminalExitRead2, &admission)
	if err != nil {
		measureErr = err
		return
	}
	if !filepath.IsAbs(admission.BuildReceipt) || !filepath.IsAbs(admission.CargoArtifact) || admission.CargoArtifact == admission.Helper || admission.Helper != request.HelperPath || admission.HelperSHA256 != request.HelperSHA256 || admission.CargoArtifactSHA256 != admission.HelperSHA256 {
		return nil, fmt.Errorf("admission must bind the immutable raw build receipt, Cargo artifact and measured helper")
	}
	buildReceiptSHA256, buildReceiptSHA256Err := fileHash(runCtx, admission.BuildReceipt)
	if buildReceiptSHA256Err != nil {
		measureErr = buildReceiptSHA256Err
		return
	}
	if admission.BuildReceiptSHA256 != buildReceiptSHA256 {
		measureErr = errutil.New("admission must bind the immutable raw build receipt, Cargo artifact and measured helper")
		return
	}
	inputs, terminalExitBuildInputsFeaturesErr := terminalExitBuildInputsFeatures(runCtx, admission.BuildReceipt, wasi.Request{HelperPath: admission.CargoArtifact, HelperSHA256: admission.CargoArtifactSHA256}, crate, features)
	if terminalExitBuildInputsFeaturesErr != nil {
		measureErr = terminalExitBuildInputsFeaturesErr
		return
	}
	file, _, err := hostfs.OpenPath(request.HelperPath)
	if err != nil {
		measureErr = err
		return
	}
	err = file.Close()
	if err != nil {
		measureErr = err
		return
	}
	receiptSHA256, receiptSHA256Err := fileHash(runCtx, receipt)
	if receiptSHA256Err != nil {
		measureErr = receiptSHA256Err
		return
	}
	inputs[receipt], inputs[request.HelperPath] = receiptSHA256, request.HelperSHA256
	verifyInputsErr := temporalVerifyInputs(runCtx, inputs)
	if verifyInputsErr != nil {
		measureErr = verifyInputsErr
		return
	}
	return inputs, measureErr
}

const warmPreparationSchema = "gomad.wasm-warm-preparation/v1"

type warmPreparationRun struct {
	Telemetry                                   wasi.Telemetry             `json:"telemetry"`
	Phases                                      map[string]diagnosticPhase `json:"phases"`
	WallNanos, TerminalToReapNanos              uint64
	EvidenceSHA256                              string
	Exited, Reaped                              bool
	ObservationPath, FinishPath, DiagnosticPath string
	RawObservationPath                          string
}
type warmPreparationObservation struct {
	TerminalReceivedAt, SupervisorReturnedAt          time.Time
	TerminalToReapNanos                               uint64
	TerminalFrame                                     []byte
	Result                                            wasi.Result
	Error                                             string
	HostExited, GroupGone, Cancelled, WatchdogTimeout bool
	HostExitCode                                      int
}

func warmPreparationObserve(observation wasi.TerminalExitObservation) warmPreparationObservation {
	return warmPreparationObservation{TerminalReceivedAt: observation.TerminalReceivedAt, SupervisorReturnedAt: observation.SupervisorReturnedAt, TerminalToReapNanos: observation.TerminalToReapNanos, TerminalFrame: observation.TerminalFrame, Result: observation.Result, Error: observation.Error, HostExited: observation.Host.Termination == "exit", GroupGone: observation.Host.GroupGone, Cancelled: observation.Host.Cancelled, WatchdogTimeout: observation.Host.WatchdogTimeout, HostExitCode: observation.Host.ExitCode}
}
func warmPreparationReapConsistent(received, returned time.Time, recorded uint64) bool {
	wall := returned.Sub(received)
	if wall <= 0 || recorded == 0 || recorded > math.MaxInt64 {
		return false
	}
	difference := wall - time.Duration(recorded)
	if difference < 0 {
		difference = -difference
	}
	return difference <= max(10*time.Microsecond, time.Duration(recorded)/1000)
}

type warmPreparationSeries struct {
	Schema, Role, SourceCommit, SourceRoot, DriverSHA256     string
	HostBootID, HostCPUInfo, ConfigSHA256                    string
	Engine                                                   wasi.EngineIdentity
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
	if series.HostBootID == "" || series.Engine.Name != "wasmtime" || series.Engine.Version != "47.0.3" || series.Engine.Configuration == "" || !warmPreparationHostCheck(series.Engine, series.HostCPUInfo) {
		return fmt.Errorf("real host evidence or engine host binding absent")
	}
	var prepare []uint64
	for i, run := range series.Runs {
		if !run.Telemetry.CacheHit || run.Telemetry.CompileNanos != 0 || run.Telemetry.PrepareNanos == 0 || run.WallNanos == 0 || !run.Exited || !run.Reaped || run.TerminalToReapNanos == 0 || candidate && run.TerminalToReapNanos > uint64(time.Second) || !warmPreparationHex(run.EvidenceSHA256, 64) || run.EvidenceSHA256 != series.Runs[0].EvidenceSHA256 {
			return fmt.Errorf("run %d is not a warm successful evidence-identical reaped run", i)
		}
		for _, phase := range strings.Fields("cache_read envelope_verify deserialize cache_identity") {
			p, ok := run.Phases[phase]
			if !ok || p.Count == 0 || p.WallNanos > run.Telemetry.PrepareNanos {
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
	err := warmPreparationValidate(before, false)
	if err != nil {
		return fmt.Errorf("baseline: %w", err)
	}
	err = warmPreparationValidate(after, true)
	if err != nil {
		return fmt.Errorf("candidate: %w", err)
	}
	if before.Engine.HostOS != after.Engine.HostOS || before.Engine.HostArch != after.Engine.HostArch || before.HostBootID != after.HostBootID || before.HostCPUInfo != after.HostCPUInfo || before.ModuleSHA256 != after.ModuleSHA256 || before.Seed != after.Seed || before.ConfigSHA256 != after.ConfigSHA256 || before.RuntimeIdentity != after.RuntimeIdentity {
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
	Result                                   wasi.Result
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
	err := decode(series.CollectionPath, &collection)
	if err != nil {
		return err
	}
	if collection.Schema != "gomad.wasm-warm-preparation-collection/v1" || collection.Accepted || !collection.CollectionValid || collection.Failure != "" || collection.UntimedRunError != "" || collection.BindingSHA256 != warmPreparationCollectionBinding(series) || !reflect.DeepEqual(collection.SummaryNanos, warmPreparationSummary(series)) || len(series.Runs) < 5 {
		return fmt.Errorf("failed or changed source/sample-bound collection")
	}
	var result wasi.Result
	err = decode(collection.ResultPath, &result)
	if err != nil {
		return err
	}
	finish, err := bound(collection.FinishPath)
	if err != nil {
		return err
	}
	if len(finish) != len(result.RuntimeTerminal()) {
		return fmt.Errorf("incomplete original Run finish")
	}
	setRuntimeTerminalBytes(&result, finish)
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
		err := decode(run.RawObservationPath, &raw)
		if err != nil {
			return err
		}
		var projected warmPreparationObservation
		err = decode(run.ObservationPath, &projected)
		if err != nil {
			return err
		}
		finish, err := bound(run.FinishPath)
		if err != nil {
			return err
		}
		if len(finish) != len(raw.Result.RuntimeTerminal()) {
			return fmt.Errorf("incomplete timed finish")
		}
		setRuntimeTerminalBytes(&raw.Result, finish)
		setRuntimeTerminalBytes(&projected.Result, finish)
		if raw.Error != "" || raw.Error != projected.Error || raw.TerminalReceivedAt != projected.TerminalReceivedAt || raw.SupervisorReturnedAt != projected.SupervisorReturnedAt || raw.TerminalToReapNanos != projected.TerminalToReapNanos || !bytes.Equal(raw.TerminalFrame, projected.TerminalFrame) || raw.Host.Termination != "exit" || !projected.HostExited || raw.Host.ExitCode != 0 || raw.Host.ExitCode != projected.HostExitCode || !raw.Host.GroupGone || raw.Host.GroupGone != projected.GroupGone || raw.Host.Cancelled || raw.Host.Cancelled != projected.Cancelled || raw.Host.WatchdogTimeout || raw.Host.WatchdogTimeout != projected.WatchdogTimeout || !reflect.DeepEqual(raw.Result, projected.Result) || warmPreparationEvidence(raw.Result) != run.EvidenceSHA256 || run.EvidenceSHA256 != collection.UntimedEvidenceSHA256 {
			return fmt.Errorf("original host observation differs from measured projection")
		}
	}
	return nil
}
func warmPreparationPublishCollection(runCtx context.Context, evidence, name string, series *warmPreparationSeries, retained wasi.Result, runError, failed string) (measureValue0 string, measureErr error) {
	collection := warmPreparationCollection{Schema: "gomad.wasm-warm-preparation-collection/v1", CollectionValid: failed == "", Failure: failed, UntimedRunError: runError, BindingSHA256: warmPreparationCollectionBinding(*series), UntimedEvidenceSHA256: warmPreparationEvidence(retained), ResultPath: filepath.Join(evidence, "untimed-evidence-observed-result.json"), CanonicalPath: filepath.Join(evidence, "untimed-evidence-canonical.bin"), ChoicePath: filepath.Join(evidence, "untimed-evidence-choice.bin"), FinishPath: filepath.Join(evidence, "untimed-evidence-finish.bin"), SummaryNanos: warmPreparationSummary(*series)}
	for _, path := range []string{collection.ResultPath, collection.CanonicalPath, collection.ChoicePath, collection.FinishPath} {
		pathSHA256, pathSHA256Err := fileHash(runCtx, path)
		if pathSHA256Err != nil {
			measureErr = pathSHA256Err
			return
		}
		series.Inputs[path] = pathSHA256
	}
	series.CollectionPath = filepath.Join(evidence, name)
	writeReportErr := temporalJSON(runCtx, series.CollectionPath, collection)
	if writeReportErr != nil {
		measureErr = writeReportErr
		return
	}
	collectionPathSHA256, collectionPathSHA256Err := fileHash(runCtx, series.CollectionPath)
	if collectionPathSHA256Err != nil {
		measureErr = collectionPathSHA256Err
		return
	}
	series.Inputs[series.CollectionPath] = collectionPathSHA256
	if failed == "" {
		err := warmPreparationCollectionCheck(*series, os.ReadFile)
		if err != nil {
			failed = err.Error()
			collection.CollectionValid, collection.Failure = false, failed
			writeReportErr2 := temporalJSON(runCtx, series.CollectionPath, collection)
			if writeReportErr2 != nil {
				measureErr = writeReportErr2
				return
			}
			collectionPathSHA2562, collectionPathSHA256Err2 := fileHash(runCtx, series.CollectionPath)
			if collectionPathSHA256Err2 != nil {
				measureErr = collectionPathSHA256Err2
				return
			}
			series.Inputs[series.CollectionPath] = collectionPathSHA2562
		}
	}
	writeReportErr3 := temporalJSON(runCtx, filepath.Join(evidence, "series.json"), *series)
	if writeReportErr3 != nil {
		measureErr = writeReportErr3
		return
	}
	return failed, measureErr
}
func warmPreparationEvidence(result wasi.Result) string {
	data, _ := json.Marshal(struct {
		Stdout, Stderr, Choices, Network []byte
		WASI                             [32]byte
		Terminal                         [16]byte
		Termination                      string
		ExitCode                         *uint32
	}{result.Stdout, result.Stderr, result.ChoiceTrace.Bytes, result.NetworkTranscript, result.TranscriptSHA256, result.RuntimeTerminal(), result.Termination, result.ExitCode})
	return digest(data)
}
func measureWarmPreparationHelperBuild(runCtx context.Context) (measureErr error) {
	root := input(runCtx, "GOMAD_WASM_WARM_HELPER_BUILD_ROOT")
	if root == "" {
		measureErr = fmt.Errorf("%w: %v", ErrRefused, errutil.New("set GOMAD_WASM_WARM_HELPER_BUILD_ROOT to build a retained diagnostic-profile helper"))
		return
	}
	terminalExitBuildHelperErr := terminalExitBuildHelper(runCtx, root, []string{"diagnostic-profile"})
	if terminalExitBuildHelperErr != nil {
		measureErr = terminalExitBuildHelperErr
		return
	}
	return
}
func measureWarmPreparationPrime(runCtx context.Context) (measureErr error) {
	root := input(runCtx, "GOMAD_WASM_WARM_PRIME_ROOT")
	if root == "" {
		measureErr = fmt.Errorf("%w: %v", ErrRefused, errutil.New("set GOMAD_WASM_WARM_PRIME_ROOT for explicit serial cache priming, never an acceptance sample"))
		return
	}
	for _, path := range []string{root, input(runCtx, "GOMAD_WASM_TEST_HELPER"), input(runCtx, "GOMAD_WASM_WARM_MODULE"), input(runCtx, "GOMAD_WASM_WARM_CACHE"), input(runCtx, "GOMAD_WASM_WARM_HELPER_RECEIPT"), input(runCtx, "GOMAD_WASM_WARM_MODULE_RECEIPT")} {
		if !filepath.IsAbs(path) {
			measureErr = errutil.New("cache priming requires the same explicit bound inputs as measurement")
			return
		}
	}
	_, _, err := warmPreparationHostEvidence(runCtx)
	if err != nil {
		measureErr = errutil.New(fmt.Sprint("real host evidence required", err))
		return
	}
	evidence, err := diagnosticRunRoot(root, "prime-only-")
	if err != nil {
		measureErr = err
		return
	}
	setInput(runCtx, "GOMAD_WASM_PROTOTYPE_ROOT", evidence)
	repo, err := absolute(runCtx, "../../../../../..")
	if err != nil {
		measureErr = err
		return
	}
	config, prototypeTemporalConfigErr := prototypeTemporalConfig(runCtx, repo, "/workspace/tests", "-test.run=^TestWorkflowTypeEncodingSuite$/^TestPlainASCII$/^Succeeds$", "-persistenceType=sql", "-persistenceDriver=sqlite")
	if prototypeTemporalConfigErr != nil {
		measureErr = prototypeTemporalConfigErr
		return
	}
	request, requestErr := prototypeRequest(runCtx, input(runCtx, "GOMAD_WASM_WARM_MODULE"), config, 0, 30*time.Minute)
	if requestErr != nil {
		measureErr = requestErr
		return
	}
	request.CompileCache = input(runCtx, "GOMAD_WASM_WARM_CACHE")
	crate, err := absolute(runCtx, "../wasmhost")
	if err != nil {
		measureErr = err
		return
	}
	inputs, terminalExitHelperInputsFeaturesErr := terminalExitHelperInputsFeatures(runCtx, input(runCtx, "GOMAD_WASM_WARM_HELPER_RECEIPT"), request, crate, []string{"diagnostic-profile"})
	if terminalExitHelperInputsFeaturesErr != nil {
		measureErr = terminalExitHelperInputsFeaturesErr
		return
	}
	terminalExitModuleInputs2, terminalExitModuleInputsErr := terminalExitModuleInputs(runCtx, input(runCtx, "GOMAD_WASM_WARM_MODULE_RECEIPT"), request)
	if terminalExitModuleInputsErr != nil {
		measureErr = terminalExitModuleInputsErr
		return
	}
	maps.Copy(inputs, terminalExitModuleInputs2)
	result, runErr := warmPreparationAdmitted(runCtx, func() (wasi.Result, error) {
		callbackValue0, callbackValue1, callbackValue2 := (func(runCtx context.Context) (measureValue0 wasi.Result, measureValue1 error, measureErr error) {
			returnedValue0, returnedValue1 := wasi.Run(runCtx, request)
			return returnedValue0, returnedValue1, measureErr
		})(runCtx)
		return callbackValue0, errutil.Join(callbackValue1, callbackValue2)
	})
	persistDiagnosticRawResultErr := persistDiagnosticRawResult(runCtx, evidence, "prime-only", result)
	if persistDiagnosticRawResultErr != nil {
		measureErr = persistDiagnosticRawResultErr
		return
	}
	writeEvidenceErr := temporalWrite(runCtx, filepath.Join(evidence, "prime-only-helper-stderr.txt"), result.HelperStderr())
	if writeEvidenceErr != nil {
		measureErr = writeEvidenceErr
		return
	}
	writeReportErr := temporalJSON(runCtx, filepath.Join(evidence, "prime-only.json"), map[string]any{"accepted": false, "measurement_sample": false, "purpose": "explicit compiled-cache priming outside all warm measurement windows", "inputs": inputs, "telemetry": result.Telemetry})
	if writeReportErr != nil {
		measureErr = writeReportErr
		return
	}
	if runErr != nil {
		measureErr = errutil.Join(runErr)
		return
	}
	requireExitErr := requireExit(runCtx, "prime-only", result, 0)
	if requireExitErr != nil {
		measureErr = requireExitErr
		return
	}
	_, err = decodeDiagnosticReport(result.HelperStderr())
	if err != nil {
		measureErr = err
		return
	}
	verifyInputsErr := temporalVerifyInputs(runCtx, inputs)
	if verifyInputsErr != nil {
		measureErr = verifyInputsErr
		return
	}
	terminalExitHelperInputsFeatures2, terminalExitHelperInputsFeaturesErr2 := terminalExitHelperInputsFeatures(runCtx, input(runCtx, "GOMAD_WASM_WARM_HELPER_RECEIPT"), request, crate, []string{"diagnostic-profile"})
	if terminalExitHelperInputsFeaturesErr2 != nil {
		measureErr = terminalExitHelperInputsFeaturesErr2
		return
	}
	_ = terminalExitHelperInputsFeatures2
	terminalExitModuleInputs3, terminalExitModuleInputsErr2 := terminalExitModuleInputs(runCtx, input(runCtx, "GOMAD_WASM_WARM_MODULE_RECEIPT"), request)
	if terminalExitModuleInputsErr2 != nil {
		measureErr = terminalExitModuleInputsErr2
		return
	}
	_ = terminalExitModuleInputs3
	return
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
func warmPreparationRead(runCtx context.Context, path, evidence, currentProducer string, currentRequest wasi.Request) (measureValue0 warmPreparationSeries, measureErr error) {
	var series warmPreparationSeries
	mustTerminalExitRead2, mustTerminalExitReadErr := mustTerminalExitRead(runCtx, path)
	if mustTerminalExitReadErr != nil {
		measureErr = mustTerminalExitReadErr
		return
	}
	decoder := json.NewDecoder(bytes.NewReader(mustTerminalExitRead2))
	decoder.DisallowUnknownFields()
	err := decoder.Decode(&series)
	if err != nil {
		measureErr = err
		return
	}
	err = decoder.Decode(new(any))
	if err != io.EOF {
		measureErr = errutil.New(fmt.Sprint("trailing warm-preparation receipt", err))
		return
	}
	verifyInputsErr := temporalVerifyInputs(runCtx, series.Inputs)
	if verifyInputsErr != nil {
		measureErr = verifyInputsErr
		return
	}
	err = warmPreparationCollectionCheck(series, os.ReadFile)
	if err != nil {
		measureErr = errutil.New(fmt.Sprint("historical collection did not close original Run equivalence", err))
		return
	}
	currentInputs, terminalExitModuleInputsErr := terminalExitModuleInputs(runCtx, currentProducer, currentRequest)
	if terminalExitModuleInputsErr != nil {
		measureErr = terminalExitModuleInputsErr
		return
	}
	err = warmPreparationProducerCheck(series, currentProducer, currentInputs)
	if err != nil {
		measureErr = err
		return
	}
	command, commandErr := temporalCommand(runCtx, evidence, "baseline-commit-check", series.SourceRoot, nil, []string{"git", "rev-parse", "HEAD"})
	if commandErr != nil {
		measureErr = commandErr
		return
	}
	commit := strings.TrimSpace(string(command.Stdout.RawBytes))
	if commit != series.SourceCommit {
		measureErr = errutil.New("historical source commit changed")
		return
	}
	command2, commandErr2 := temporalCommand(runCtx, evidence, "baseline-ancestor-check", series.SourceRoot, nil, []string{"git", "merge-base", "--is-ancestor", "15119dbbeb03f5a57fa79d57be0ab2d9ba4038c3", series.SourceCommit})
	if commandErr2 != nil {
		measureErr = commandErr2
		return
	}
	_ = command2
	changes, commandErr3 := temporalCommand(runCtx, evidence, "baseline-production-check", series.SourceRoot, nil, []string{"git", "diff", "--name-only", "15119dbbeb03f5a57fa79d57be0ab2d9ba4038c3", series.SourceCommit, "--", "."})
	if commandErr3 != nil {
		measureErr = commandErr3
		return
	}
	for _, path := range strings.Fields(string(changes.Stdout.RawBytes)) {
		if path != "tools/gomad/backends/wasm/wasi/warm_preparation_test.go" && path != "tools/gomad/backends/wasm/wasi/warm_preparation_inputs_test.go" && path != "tools/gomad/backends/wasm/wasi/warm_preparation_terminal_test.go" {
			measureErr = errutil.New(fmt.Sprint("historical production or original driver changed", path))
			return
		}
	}
	if series.Inputs[series.DriverPath] != series.DriverSHA256 || !filepath.IsAbs(series.SourceRoot) || !strings.HasPrefix(series.DriverPath, series.SourceRoot+string(filepath.Separator)) || series.Inputs[series.HelperReceipt] == "" || series.Inputs[series.ModuleReceipt] == "" || series.Inputs[series.ArtifactPath] == "" {
		measureErr = errutil.New("historical receipt lacks actual driver/build/module/cache bindings")
		return
	}
	var admission terminalExitHelperAdmission
	mustTerminalExitRead3, mustTerminalExitReadErr2 := mustTerminalExitRead(runCtx, series.HelperReceipt)
	if mustTerminalExitReadErr2 != nil {
		measureErr = mustTerminalExitReadErr2
		return
	}
	err = json.Unmarshal(mustTerminalExitRead3, &admission)
	if err != nil {
		measureErr = err
		return
	}
	var build terminalExitHelperBuild
	mustTerminalExitRead4, mustTerminalExitReadErr3 := mustTerminalExitRead(runCtx, admission.BuildReceipt)
	if mustTerminalExitReadErr3 != nil {
		measureErr = mustTerminalExitReadErr3
		return
	}
	err = json.Unmarshal(mustTerminalExitRead4, &build)
	if err != nil {
		measureErr = err
		return
	}
	if !strings.HasPrefix(build.Crate, series.SourceRoot+string(filepath.Separator)) || admission.HelperSHA256 != series.HelperSHA256 {
		measureErr = errutil.New("historical helper does not belong to its retained source/build")
		return
	}
	terminalExitHelperInputsFeatures2, terminalExitHelperInputsFeaturesErr := terminalExitHelperInputsFeatures(runCtx, series.HelperReceipt, wasi.Request{HelperPath: admission.Helper, HelperSHA256: admission.HelperSHA256}, build.Crate, []string{"diagnostic-profile"})
	if terminalExitHelperInputsFeaturesErr != nil {
		measureErr = terminalExitHelperInputsFeaturesErr
		return
	}
	_ = terminalExitHelperInputsFeatures2
	for _, run := range series.Runs {
		for _, rawPath := range []string{run.ObservationPath, run.FinishPath, run.DiagnosticPath} {
			if series.Inputs[rawPath] == "" {
				measureErr = errutil.New("historical run lacks bound raw observation/finish/diagnostic evidence")
				return
			}
		}
		var observation warmPreparationObservation
		mustTerminalExitRead5, mustTerminalExitReadErr4 := mustTerminalExitRead(runCtx, run.ObservationPath)
		if mustTerminalExitReadErr4 != nil {
			measureErr = mustTerminalExitReadErr4
			return
		}
		err := json.Unmarshal(mustTerminalExitRead5, &observation)
		if err != nil {
			measureErr = err
			return
		}
		finish, mustTerminalExitReadErr5 := mustTerminalExitRead(runCtx, run.FinishPath)
		if mustTerminalExitReadErr5 != nil {
			measureErr = mustTerminalExitReadErr5
			return
		}
		if len(finish) != len(observation.Result.RuntimeTerminal()) {
			measureErr = errutil.New("missing historical private terminal sidecar")
			return
		}
		setRuntimeTerminalBytes(&observation.Result, finish)
		mustTerminalExitRead6, mustTerminalExitReadErr6 := mustTerminalExitRead(runCtx, run.DiagnosticPath)
		if mustTerminalExitReadErr6 != nil {
			measureErr = mustTerminalExitReadErr6
			return
		}
		profile, err := decodeDiagnosticReport(mustTerminalExitRead6)
		if err != nil || !reflect.DeepEqual(profile.Phases, run.Phases) || observation.Result.Telemetry == nil || *observation.Result.Telemetry != run.Telemetry || warmPreparationEvidence(observation.Result) != run.EvidenceSHA256 || observation.TerminalToReapNanos != run.TerminalToReapNanos || observation.Error != "" || !observation.HostExited || observation.HostExitCode != 0 || !observation.GroupGone || observation.Cancelled || observation.WatchdogTimeout || observation.TerminalReceivedAt.IsZero() || !observation.SupervisorReturnedAt.After(observation.TerminalReceivedAt) || !warmPreparationReapConsistent(observation.TerminalReceivedAt, observation.SupervisorReturnedAt, run.TerminalToReapNanos) || !observation.Result.Reaped || observation.Result.Termination != "exit" || observation.Result.ExitCode == nil || *observation.Result.ExitCode != 0 || observation.Result.Started == nil || observation.Result.Started.Engine != series.Engine || observation.Result.Started.ModuleSHA256 != series.ModuleSHA256 {
			measureErr = errutil.New(fmt.Sprint("historical raw observation does not support the declared run", err))
			return
		}
	}
	configBytes, err := json.Marshal(currentRequest.Config)
	if err != nil || series.ConfigSHA256 != digest(configBytes) {
		measureErr = errutil.New(fmt.Sprint("historical comparison requires the bound modeled configuration", err))
		return
	}
	var collection warmPreparationCollection
	mustTerminalExitRead7, mustTerminalExitReadErr7 := mustTerminalExitRead(runCtx, series.CollectionPath)
	if mustTerminalExitReadErr7 != nil {
		measureErr = mustTerminalExitReadErr7
		return
	}
	err = json.Unmarshal(mustTerminalExitRead7, &collection)
	if err != nil {
		measureErr = err
		return
	}
	var retained wasi.Result
	mustTerminalExitRead8, mustTerminalExitReadErr8 := mustTerminalExitRead(runCtx, collection.ResultPath)
	if mustTerminalExitReadErr8 != nil {
		measureErr = mustTerminalExitReadErr8
		return
	}
	err = json.Unmarshal(mustTerminalExitRead8, &retained)
	if err != nil {
		measureErr = err
		return
	}
	mustTerminalExitRead9, mustTerminalExitReadErr9 := mustTerminalExitRead(runCtx, collection.FinishPath)
	if mustTerminalExitReadErr9 != nil {
		measureErr = mustTerminalExitReadErr9
		return
	}
	setRuntimeTerminalBytes(&retained, mustTerminalExitRead9)
	series.ComparisonSHA256, err = warmPreparationBehavior(retained, currentRequest.Config, true)
	if err != nil {
		measureErr = errutil.New(fmt.Sprint("source-bound historical transcript comparison", err))
		return
	}
	return series, measureErr
}
func warmPreparationArtifact(runCtx context.Context, request wasi.Request) (measureValue0 string, measureValue1 uint64, measureValue2 uint64, measureErr error) {
	entries, err := filepath.Glob(filepath.Join(request.CompileCache, "*.gomad-cache-v3"))
	if err != nil {
		measureErr = err
		return
	}
	selected := ""
	var artifactBytes, envelopeBytes uint64
	for _, entry := range entries {
		file, info, err := hostfs.OpenPath(entry)
		if err != nil {
			measureErr = err
			return
		}
		var header [24]byte
		_, err = io.ReadFull(file, header[:])
		manifestBytes := binary.LittleEndian.Uint32(header[8:12])
		artifact := binary.LittleEndian.Uint64(header[12:20])
		if err != nil || string(header[:8]) != "GOMADC03" || manifestBytes > 16<<10 || artifact == 0 || artifact > 1<<30 || uint64(info.Size()) != 24+uint64(manifestBytes)+artifact {
			closeResource(runCtx, file)
			measureErr = errutil.New(fmt.Sprint("invalid current cache envelope", entry, err))
			return
		}
		manifest := make([]byte, manifestBytes)
		_, err = io.ReadFull(file, manifest)
		closeErr := file.Close()
		if err != nil || closeErr != nil {
			measureErr = errutil.New(fmt.Sprint("read cache manifest", err, closeErr))
			return
		}
		var identity struct {
			Version       uint64 `json:"version"`
			ModuleSHA256  string `json:"module_sha256"`
			HelperSHA256  string `json:"helper_sha256"`
			Configuration string `json:"configuration"`
			HostOS        string `json:"host_os"`
			HostArch      string `json:"host_arch"`
		}
		err = json.Unmarshal(manifest, &identity)
		if err != nil {
			measureErr = err
			return
		}
		if identity.ModuleSHA256 != request.ModuleSHA256 || identity.HelperSHA256 != request.HelperSHA256 {
			continue
		}
		if selected != "" || identity.Version != 3 || identity.Configuration != request.Engine.Configuration || identity.HostOS != request.Engine.HostOS || identity.HostArch != request.Engine.HostArch || filepath.Base(entry) != digest(manifest)+".gomad-cache-v3" {
			measureErr = errutil.New("ambiguous or incompatible measured cache identity")
			return
		}
		selected, artifactBytes, envelopeBytes = entry, artifact, uint64(info.Size())
	}
	if selected == "" {
		measureErr = errutil.New("prewarmed matching cache artifact required; this driver never cold-compiles intentionally")
		return
	}
	return selected, artifactBytes, envelopeBytes, measureErr
}
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
func measureWarmPreparation(runCtx context.Context) (measureErr error) {
	root := input(runCtx, "GOMAD_WASM_WARM_PREPARATION_ROOT")
	if root == "" {
		measureErr = fmt.Errorf("%w: %v", ErrRefused, errutil.New("set GOMAD_WASM_WARM_PREPARATION_ROOT for serial real-host five-run acceptance collection"))
		return
	}
	paths := map[string]string{"root": root, "helper": input(runCtx, "GOMAD_WASM_TEST_HELPER"), "module": input(runCtx, "GOMAD_WASM_WARM_MODULE"), "cache": input(runCtx, "GOMAD_WASM_WARM_CACHE"), "helper_receipt": input(runCtx, "GOMAD_WASM_WARM_HELPER_RECEIPT"), "module_receipt": input(runCtx, "GOMAD_WASM_WARM_MODULE_RECEIPT")}
	for name, path := range paths {
		if !filepath.IsAbs(path) {
			measureErr = errutil.New(fmt.Sprint("explicit absolute warm-preparation input required", name))
			return
		}
	}
	boot, cpuinfo, err := warmPreparationHostEvidence(runCtx)
	if err != nil {
		measureErr = errutil.New(fmt.Sprint("real host evidence required", err))
		return
	}
	evidence, err := diagnosticRunRoot(root, "warm-preparation-")
	if err != nil {
		measureErr = err
		return
	}
	setInput(runCtx, "GOMAD_WASM_PROTOTYPE_ROOT", evidence)
	repo, err := absolute(runCtx, "../../../../../..")
	if err != nil {
		measureErr = err
		return
	}
	config, prototypeTemporalConfigErr := prototypeTemporalConfig(runCtx, repo, "/workspace/tests", "-test.run=^TestWorkflowTypeEncodingSuite$/^TestPlainASCII$/^Succeeds$", "-persistenceType=sql", "-persistenceDriver=sqlite")
	if prototypeTemporalConfigErr != nil {
		measureErr = prototypeTemporalConfigErr
		return
	}
	request, requestErr := prototypeRequest(runCtx, paths["module"], config, 0, 30*time.Minute)
	if requestErr != nil {
		measureErr = requestErr
		return
	}
	request.CompileCache = paths["cache"]
	request.DigestTranscript = true
	crate, err := absolute(runCtx, "../wasmhost")
	if err != nil {
		measureErr = err
		return
	}
	inputs, terminalExitHelperInputsFeaturesErr := terminalExitHelperInputsFeatures(runCtx, paths["helper_receipt"], request, crate, []string{"diagnostic-profile"})
	if terminalExitHelperInputsFeaturesErr != nil {
		measureErr = terminalExitHelperInputsFeaturesErr
		return
	}
	terminalExitModuleInputs2, terminalExitModuleInputsErr := terminalExitModuleInputs(runCtx, paths["module_receipt"], request)
	if terminalExitModuleInputsErr != nil {
		measureErr = terminalExitModuleInputsErr
		return
	}
	maps.Copy(inputs, terminalExitModuleInputs2)
	command, commandErr := temporalCommand(runCtx, evidence, "source-commit", repo, nil, []string{"git", "rev-parse", "HEAD"})
	if commandErr != nil {
		measureErr = commandErr
		return
	}
	commit := strings.TrimSpace(string(command.Stdout.RawBytes))
	status, commandErr2 := temporalCommand(runCtx, evidence, "source-status", repo, nil, []string{"git", "status", "--porcelain", "--", "tools/gomad"})
	if commandErr2 != nil {
		measureErr = commandErr2
		return
	}
	if len(status.Stdout.RawBytes) != 0 {
		measureErr = errutil.New("freeze and commit the measured tool source first")
		return
	}
	tracked, commandErr3 := temporalCommand(runCtx, evidence, "source-inventory", repo, nil, []string{"git", "ls-files", "-z", "tools/gomad"})
	if commandErr3 != nil {
		measureErr = commandErr3
		return
	}
	for _, name := range bytes.Split(tracked.Stdout.RawBytes, []byte{0}) {
		if len(name) != 0 {
			path := filepath.Join(repo, string(name))
			pathSHA256, pathSHA256Err := fileHash(runCtx, path)
			if pathSHA256Err != nil {
				measureErr = pathSHA256Err
				return
			}
			inputs[path] = pathSHA256
		}
	}
	driver := filepath.Join(repo, warmPreparationDriverSource)
	artifact, artifactBytes, envelopeBytes, warmPreparationArtifactErr := warmPreparationArtifact(runCtx, request)
	if warmPreparationArtifactErr != nil {
		measureErr = warmPreparationArtifactErr
		return
	}
	artifactSHA256, artifactSHA256Err := fileHash(runCtx, artifact)
	if artifactSHA256Err != nil {
		measureErr = artifactSHA256Err
		return
	}
	inputs[artifact] = artifactSHA256
	configBytes, err := json.Marshal(request.Config)
	if err != nil {
		measureErr = err
		return
	}
	driverSHA256, driverSHA256Err := fileHash(runCtx, driver)
	if driverSHA256Err != nil {
		measureErr = driverSHA256Err
		return
	}
	series := warmPreparationSeries{Schema: warmPreparationSchema, Role: "candidate", SourceCommit: commit, SourceRoot: repo, DriverSHA256: driverSHA256, HostBootID: boot, HostCPUInfo: cpuinfo, ConfigSHA256: digest(configBytes), Engine: request.Engine, ModuleSHA256: request.ModuleSHA256, HelperSHA256: request.HelperSHA256, RuntimeIdentity: request.Runtime.Choice.ExecutionIdentity.ToolchainBuildKey, ModelSHA256: wasi.ImplementationSHA256(), Seed: request.Runtime.Seed, ArtifactBytes: artifactBytes, EnvelopeBytes: envelopeBytes, Inputs: inputs, HelperReceipt: paths["helper_receipt"], ModuleReceipt: paths["module_receipt"], DriverPath: driver, ArtifactPath: artifact}
	failed := ""
	var reference wasi.Result
	for i := range 6 {
		var observation wasi.TerminalExitObservation
		started := time.Now()
		_, admissionErr := warmPreparationAdmitted(runCtx, func() (wasi.Result, error) {
			callbackValue0, callbackValue1, callbackValue2 := (func(runCtx context.Context) (measureValue0 wasi.Result, measureValue1 error, measureErr error) {
				observation = terminalExitObservationFor(runCtx, request, false)
				return observation.Result, nil, measureErr
			})(runCtx)
			return callbackValue0, errutil.Join(callbackValue1, callbackValue2)
		})
		wall := uint64(time.Since(started).Nanoseconds())
		if admissionErr != nil {
			measureErr = errutil.Join(admissionErr)
			return
		}
		name := fmt.Sprintf("run-%d", i)
		writeReportErr := temporalJSON(runCtx, filepath.Join(evidence, name+"-observation.json"), observation)
		if writeReportErr != nil {
			measureErr = writeReportErr
			return
		}
		writeEvidenceErr := temporalWrite(runCtx, filepath.Join(evidence, name+"-helper-stderr.txt"), observation.Host.Stderr.RawBytes)
		if writeEvidenceErr != nil {
			measureErr = writeEvidenceErr
			return
		}
		if observation.Error != "" || observation.Result.Telemetry == nil {
			measureErr = errutil.New(fmt.Sprint("helper observation/telemetry failed", observation.Error))
			return
		}
		if !observation.Result.Telemetry.CacheHit || observation.Result.Telemetry.CompileNanos != 0 {
			measureErr = errutil.New("every run, including the unmeasured warm-up, must hit the existing cache without compilation")
			return
		}
		profile, err := decodeDiagnosticReport(observation.Host.Stderr.RawBytes)
		if err != nil || profile.HostOS != request.Engine.HostOS || profile.HostArch != request.Engine.HostArch {
			measureErr = errutil.New(fmt.Sprint("diagnostic phase/host evidence required", err))
			return
		}
		if i == 0 {
			reference = observation.Result
		} else {
			difference := sameEvidence(reference, observation.Result)
			if difference != "" {
				failed = warmPreparationFailure(failed, "within-series evidence changed: "+difference)
			}
		}
		{
			finishPath := filepath.Join(evidence, name+"-finish.bin")
			writeEvidenceErr2 := temporalWrite(runCtx, finishPath, runtimeTerminalBytes(observation.Result)[:])
			if writeEvidenceErr2 != nil {
				measureErr = writeEvidenceErr2
				return
			}
			observationPath, diagnosticPath := filepath.Join(evidence, name+"-warm-observation.json"), filepath.Join(evidence, name+"-helper-stderr.txt")
			writeReportErr2 := temporalJSON(runCtx, observationPath, warmPreparationObserve(observation))
			if writeReportErr2 != nil {
				measureErr = writeReportErr2
				return
			}
			rawObservationPath := filepath.Join(evidence, name+"-observation.json")
			for _, path := range []string{observationPath, rawObservationPath, diagnosticPath, finishPath} {
				pathSHA2562, pathSHA256Err2 := fileHash(runCtx, path)
				if pathSHA256Err2 != nil {
					measureErr = pathSHA256Err2
					return
				}
				inputs[path] = pathSHA2562
			}
			run := warmPreparationRun{Telemetry: *observation.Result.Telemetry, Phases: profile.Phases, WallNanos: wall, TerminalToReapNanos: observation.TerminalToReapNanos, EvidenceSHA256: warmPreparationEvidence(observation.Result), Exited: observation.Within(time.Second) && observation.Result.Termination == "exit" && observation.Result.ExitCode != nil && *observation.Result.ExitCode == 0, Reaped: observation.Result.Reaped, ObservationPath: observationPath, RawObservationPath: rawObservationPath, FinishPath: finishPath, DiagnosticPath: diagnosticPath}
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
	var retained wasi.TerminalExitObservation
	_, retainedAdmissionErr := warmPreparationAdmitted(runCtx, func() (wasi.Result, error) {
		callbackValue0, callbackValue1, callbackValue2 := (func(runCtx context.Context) (measureValue0 wasi.Result, measureValue1 error, measureErr error) {
			retained = terminalExitObservationFor(runCtx, request, false)
			return retained.Result, nil, measureErr
		})(runCtx)
		return callbackValue0, errutil.Join(callbackValue1, callbackValue2)
	})
	if retainedAdmissionErr != nil {
		measureErr = errutil.New(fmt.Sprint("untimed retention admission refused before execution", retainedAdmissionErr))
		return
	}
	persistDiagnosticRawResultErr := persistDiagnosticRawResult(runCtx, evidence, "untimed-evidence", retained.Result)
	if persistDiagnosticRawResultErr != nil {
		measureErr = persistDiagnosticRawResultErr
		return
	}
	writeReportErr3 := temporalJSON(runCtx, filepath.Join(evidence, "untimed-evidence-observation.json"), retained)
	if writeReportErr3 != nil {
		measureErr = writeReportErr3
		return
	}
	if retained.Error != "" || sameEvidence(reference, retained.Result) != "" {
		failed = warmPreparationFailure(failed, "untimed canonical retention changed behavior")
	}
	verifyInputsErr := temporalVerifyInputs(runCtx, inputs)
	if verifyInputsErr != nil {
		measureErr = verifyInputsErr
		return
	}
	terminalExitHelperInputsFeatures2, terminalExitHelperInputsFeaturesErr2 := terminalExitHelperInputsFeatures(runCtx, paths["helper_receipt"], request, crate, []string{"diagnostic-profile"})
	if terminalExitHelperInputsFeaturesErr2 != nil {
		measureErr = terminalExitHelperInputsFeaturesErr2
		return
	}
	_ = terminalExitHelperInputsFeatures2
	terminalExitModuleInputs3, terminalExitModuleInputsErr2 := terminalExitModuleInputs(runCtx, paths["module_receipt"], request)
	if terminalExitModuleInputsErr2 != nil {
		measureErr = terminalExitModuleInputsErr2
		return
	}
	_ = terminalExitModuleInputs3
	command2, commandErr4 := temporalCommand(runCtx, evidence, "source-commit-after", repo, nil, []string{"git", "rev-parse", "HEAD"})
	if commandErr4 != nil {
		measureErr = commandErr4
		return
	}
	afterCommit := strings.TrimSpace(string(command2.Stdout.RawBytes))
	if afterCommit != commit {
		failed = warmPreparationFailure(failed, "source commit changed during collection")
	}
	afterBoot, afterCPU, err := warmPreparationHostEvidence(runCtx)
	if err != nil || boot != afterBoot || cpuinfo != afterCPU {
		failed = warmPreparationFailure(failed, "host bindings changed during collection")
	}
	err = warmPreparationValidate(series, true)
	if err != nil {
		failed = warmPreparationFailure(failed, err.Error())
	}
	series.ComparisonSHA256, err = warmPreparationBehavior(retained.Result, request.Config, false)
	if err != nil {
		failed = warmPreparationFailure(failed, "candidate transcript comparison: "+err.Error())
	}
	warmPreparationPublishCollection2, warmPreparationPublishCollectionErr := warmPreparationPublishCollection(runCtx, evidence, "candidate-collection.json", &series, retained.Result, retained.Error, failed)
	if warmPreparationPublishCollectionErr != nil {
		measureErr = warmPreparationPublishCollectionErr
		return
	}
	failed = warmPreparationPublishCollection2
	baselinePath := input(runCtx, "GOMAD_WASM_WARM_BASELINE_RECEIPT")
	var baseline *warmPreparationSeries
	var baselineSummary map[string]variantTimingRange
	baselineComparison := ""
	if baselinePath == "" {
		failed = warmPreparationFailure(failed, "baseline-material-gap: collect the historical source with its original matching engine/driver; this candidate never overrides engine identity")
	} else {
		if !filepath.IsAbs(baselinePath) {
			measureErr = errutil.New("baseline receipt must be absolute")
			return
		}
		before, warmPreparationReadErr := warmPreparationRead(runCtx, baselinePath, evidence, paths["module_receipt"], request)
		if warmPreparationReadErr != nil {
			measureErr = warmPreparationReadErr
			return
		}
		baseline = &before
		baselineSummary = warmPreparationSummary(before)
		baselineComparison = before.ComparisonSHA256
		err := warmPreparationCompare(before, series)
		if err != nil {
			failed = warmPreparationFailure(failed, err.Error())
		}
		baselinePathSHA256, baselinePathSHA256Err := fileHash(runCtx, baselinePath)
		if baselinePathSHA256Err != nil {
			measureErr = baselinePathSHA256Err
			return
		}
		inputs[baselinePath] = baselinePathSHA256
	}
	summary := warmPreparationSummary(series)
	writeReportErr4 := temporalJSON(runCtx, filepath.Join(evidence, "acceptance.json"), map[string]any{"schema": "gomad.wasm-warm-preparation-acceptance/v1", "accepted": failed == "", "failure": failed, "candidate": series, "baseline": baseline, "candidate_summary_nanos": summary, "baseline_summary_nanos": baselineSummary, "candidate_comparison_sha256": series.ComparisonSHA256, "baseline_comparison_sha256": baselineComparison, "comparison_policy": "source-bound historical paired JSONL validated through the typed GMWT owner; original raw digests remain separate; no format fallback or modeled-input masking", "artifact_bytes": artifactBytes, "envelope_bytes": envelopeBytes, "parked_mapping_owner_decision_required": summary["cache_read_plus_deserialize"].Median > uint64(time.Second), "boundary": "one unmeasured warm-up, five fresh helpers with same workflow and seed on prewarmed page cache; no artifact hashing in timed windows; full source/build/module/cache bindings verified before and after; terminal receipt through supervisor wait/capture/group reaping", "baseline_policy": "historical matching driver and PinnedEngine only; helper/engine/model/source identities separate, modeled inputs and exact behavior must remain equal"})
	if writeReportErr4 != nil {
		measureErr = writeReportErr4
		return
	}
	if failed != "" {
		measureErr = fmt.Errorf("warm-preparation acceptance not met: %s; retained %s", failed, evidence)
		return
	}
	return
}
func warmPreparationLegacyTranscript(data []byte, config wasi.Config) ([]byte, error) {
	return wasi.MeasurementLegacyTranscript(data, config)
}
func warmPreparationBehavior(result wasi.Result, config wasi.Config, historical bool) (string, error) {
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
	}{evidence, result.RuntimeTerminal()})
	if err != nil {
		return "", err
	}
	return digest(data), nil
}
func measureHostCallProfile(runCtx context.Context) (measureErr error) {
	root := input(runCtx, "GOMAD_WASM_HOST_CALL_PROFILE_ROOT")
	if root == "" {
		measureErr = fmt.Errorf("%w: %v", ErrRefused, errutil.New("set GOMAD_WASM_HOST_CALL_PROFILE_ROOT and a diagnostic GOMAD_WASM_TEST_HELPER"))
		return
	}
	if !filepath.IsAbs(root) || !filepath.IsAbs(input(runCtx, "GOMAD_WASM_TEST_HELPER")) {
		measureErr = errutil.New("profile root and GOMAD_WASM_TEST_HELPER must be absolute")
		return
	}
	repetitions := 3
	value := input(runCtx, "GOMAD_WASM_HOST_CALL_PROFILE_REPETITIONS")
	if value != "" {
		parsed, err := strconv.Atoi(value)
		if err != nil || parsed < 1 || parsed > 20 {
			measureErr = errutil.New("repetitions must be 1..20")
			return
		}
		repetitions = parsed
	}
	repo, err := absolute(runCtx, "../../../../../..")
	if err != nil {
		measureErr = err
		return
	}
	setInput(runCtx, "GOMAD_WASM_PROTOTYPE_ROOT", root)
	evidence := filepath.Join(root, "evidence")
	err = os.MkdirAll(evidence, 0700)
	if err != nil {
		measureErr = err
		return
	}
	module := input(runCtx, "GOMAD_WASM_HOST_CALL_PROFILE_MODULE")
	if module == "" {
		module = filepath.Join(root, "tests.wasm")
		prototypeBuild2, prototypeBuild22, prototypeBuildErr := prototypeBuild(runCtx, repo, "test", "./tests", module, "disable_grpc_modules", "test_dep", "sqlite3_dotlk")
		if prototypeBuildErr != nil {
			measureErr = prototypeBuildErr
			return
		}
		_, seconds := prototypeBuild2, prototypeBuild22
		logf(runCtx, "built Temporal guest in %.1fs", seconds)
	} else if !filepath.IsAbs(module) {
		measureErr = errutil.New("GOMAD_WASM_HOST_CALL_PROFILE_MODULE must be absolute")
		return
	}
	persistence := []string{"-persistenceType=sql", "-persistenceDriver=sqlite"}
	prototypeTemporalConfig2, prototypeTemporalConfigErr := prototypeTemporalConfig(runCtx, repo, "/workspace/tests", append([]string{"-test.run=^$"}, persistence...)...)
	if prototypeTemporalConfigErr != nil {
		measureErr = prototypeTemporalConfigErr
		return
	}
	warm, requestErr := prototypeRequest(runCtx, module, prototypeTemporalConfig2, 0, 30*time.Minute)
	if requestErr != nil {
		measureErr = requestErr
		return
	}
	runDiagnosticProfile2, runDiagnosticProfileErr := runDiagnosticProfile(runCtx, evidence, "prepare", warm, true, false)
	if runDiagnosticProfileErr != nil {
		measureErr = runDiagnosticProfileErr
		return
	}
	_ = runDiagnosticProfile2
	commit, err := exec.CommandContext(runCtx, "git", "-C", repo, "rev-parse", "HEAD").Output()
	if err != nil {
		measureErr = err
		return
	}
	var rows []map[string]any
	for _, seed := range []uint64{0, 1000003, 2000006} {
		var first wasi.Result
		for repetition := range repetitions {
			config, prototypeTemporalConfigErr2 := prototypeTemporalConfig(runCtx, repo, "/workspace/tests", append([]string{"-test.run=^TestActivityTestSuite$"}, persistence...)...)
			if prototypeTemporalConfigErr2 != nil {
				measureErr = prototypeTemporalConfigErr2
				return
			}
			request, requestErr2 := prototypeRequest(runCtx, module, config, seed, 30*time.Minute)
			if requestErr2 != nil {
				measureErr = requestErr2
				return
			}
			name := "activity-seed" + stringSeed(seed) + "-rep" + strconv.Itoa(repetition)
			result, runDiagnosticProfileErr2 := runDiagnosticProfile(runCtx, evidence, name, request, true, true)
			if runDiagnosticProfileErr2 != nil {
				measureErr = runDiagnosticProfileErr2
				return
			}
			report, err := decodeDiagnosticReport(result.HelperStderr())
			if err != nil {
				measureErr = err
				return
			}
			if repetition == 0 {
				first = result
			} else {
				requireDiagnosticEvidenceErr := requireDiagnosticEvidence(runCtx, first, result)
				if requireDiagnosticEvidenceErr != nil {
					measureErr = requireDiagnosticEvidenceErr
					return
				}
			}
			encoding, hash := report.Nested["encoding"], report.Nested["transcript_hash"]
			rows = append(rows, map[string]any{"seed": seed, "repetition": repetition, "calls": result.Calls, "encoding_count": encoding.Count, "encoding_wall_nanos": encoding.WallNanos, "transcript_hash_count": hash.Count, "transcript_hash_wall_nanos": hash.WallNanos, "encoding_plus_hash_wall_nanos": encoding.WallNanos + hash.WallNanos, "authorization_to_terminal_nanos": result.AuthorizationToTerminalNanos, "transcript_sha256": fmt.Sprintf("%x", result.TranscriptSHA256)})
			logf(runCtx, "%s: encoding=%d/%dns transcript_hash=%d/%dns calls=%d", name, encoding.Count, encoding.WallNanos, hash.Count, hash.WallNanos, result.Calls)
		}
	}
	cpuinfo, requiredHost := currentDiagnosticPerformanceHostEvidence(runCtx)
	writeReportErr := temporalJSON(runCtx, filepath.Join(root, "host-call-profile.json"), map[string]any{"diagnostic_only": true, "selection": "^TestActivityTestSuite$", "source_commit": strings.TrimSpace(string(commit)), "helper_sha256": warm.HelperSHA256, "module_sha256": warm.ModuleSHA256, "model_sha256": wasi.ImplementationSHA256(), "host_goos": runtime.GOOS, "host_goarch": runtime.GOARCH, "required_host_measured": requiredHost, "host_cpuinfo": cpuinfo, "repetitions": repetitions, "rows": rows})
	if writeReportErr != nil {
		measureErr = writeReportErr
		return
	}
	return
}

const namespaceMeasurementBase = "3858b64690cda466f5ee7611b701471852f87ae4"

type namespaceMeasurementSample struct {
	Role                                string
	Seed                                uint64
	Repetition                          int
	AuthorizedNanos                     uint64
	EvidenceSHA256, EnvironmentIdentity string
	Calls, TimeAdvances                 uint64
	GoFilesystemCalls                   map[string]uint64
	CounterPresent                      bool
}

func namespaceMeasurementCompare(before, after []namespaceMeasurementSample) error {
	seeds := []uint64{0, 1000003, 2000006, 3000009}
	if len(before) != len(seeds)*5 || len(after) != len(before) {
		return fmt.Errorf("four matched seeds with five fresh repetitions required")
	}
	for seedIndex, seed := range seeds {
		var baseline, candidate []uint64
		for repetition := range 5 {
			index := seedIndex*5 + repetition
			b, a := before[index], after[index]
			if b.Role != "baseline" || a.Role != "candidate" || b.Seed != seed || a.Seed != seed || b.Repetition != repetition || a.Repetition != repetition || b.AuthorizedNanos == 0 || a.AuthorizedNanos == 0 || !a.CounterPresent || len(a.GoFilesystemCalls) != 0 {
				return fmt.Errorf("seed %d repetition %d lacks matched timings or empty actual candidate counter", seed, repetition)
			}
			if !warmPreparationHex(b.EvidenceSHA256, 64) || a.EvidenceSHA256 != b.EvidenceSHA256 || b.EnvironmentIdentity == "" || a.EnvironmentIdentity != b.EnvironmentIdentity || a.Calls != b.Calls || a.TimeAdvances != b.TimeAdvances || b.EvidenceSHA256 != before[seedIndex*5].EvidenceSHA256 {
				return fmt.Errorf("seed %d repetition %d changed choice/WASI/network/output/terminal or model counts", seed, repetition)
			}
			baseline, candidate = append(baseline, b.AuthorizedNanos), append(candidate, a.AuthorizedNanos)
		}
		b, a := variantTiming(baseline).Median, variantTiming(candidate).Median
		if a > b/4*3+b%4*3/4 {
			return fmt.Errorf("seed %d authorized median %d exceeds 0.75 baseline %d", seed, a, b)
		}
	}
	return nil
}
func namespaceMeasurementCounter(data []byte, candidate bool) (map[string]uint64, error) {
	var fields map[string]json.RawMessage
	err := json.Unmarshal(data, &fields)
	if err != nil {
		return nil, err
	}
	raw, present := fields["GoFilesystemCalls"]
	if !candidate && !present {
		return nil, nil
	}
	if candidate && (!present || bytes.Equal(bytes.TrimSpace(raw), []byte("null"))) {
		return nil, fmt.Errorf("actual Result filesystem counter absent")
	}
	var counts map[string]uint64
	err = json.Unmarshal(raw, &counts)
	if err != nil {
		return nil, err
	}
	if candidate && len(counts) != 0 {
		return nil, fmt.Errorf("candidate Go namespace served operations: %v", counts)
	}
	return counts, nil
}

type namespaceMeasurementCommon struct {
	Schema, CurrentSourceRoot, CurrentSourceCommit, ModuleReceipt string
	Request                                                       wasi.Request
	ProducerInputs                                                map[string]string
}
type namespaceMeasurementReceipt struct {
	Schema, Role, SourceRoot, SourceCommit, ModelSHA256                                                  string
	HostBootID, HostCPUInfo, CommonPath, HelperReceipt, ArtifactPath                                     string
	ResultPath, RequestPath, FinishPath, TranscriptPath, ChoicePath, StdoutPath, StderrPath, NetworkPath string
	Seed                                                                                                 uint64
	Repetition                                                                                           int
	Replay                                                                                               bool
	WallNanos                                                                                            uint64
	RunError, Failure                                                                                    string
	CollectionValid                                                                                      bool
	Inputs                                                                                               map[string]string
}

func namespaceMeasurementReadJSON(runCtx context.Context, path string, value any) (measureErr error) {
	mustTerminalExitRead2, mustTerminalExitReadErr := mustTerminalExitRead(runCtx, path)
	if mustTerminalExitReadErr != nil {
		measureErr = mustTerminalExitReadErr
		return
	}
	decoder := json.NewDecoder(bytes.NewReader(mustTerminalExitRead2))
	decoder.DisallowUnknownFields()
	err := decoder.Decode(value)
	if err != nil {
		measureErr = errutil.New(fmt.Sprint(path, err))
		return
	}
	err = decoder.Decode(new(any))
	if err != io.EOF {
		measureErr = errutil.New(fmt.Sprint("trailing measurement receipt", path, err))
		return
	}
	return
}
func namespaceMeasurementRequest(runCtx context.Context, common namespaceMeasurementCommon) (measureValue0 wasi.Request, measureErr error) {
	data, err := json.Marshal(common.Request)
	if err != nil {
		measureErr = err
		return
	}
	var request wasi.Request
	err = json.Unmarshal(data, &request)
	if err != nil {
		measureErr = err
		return
	}
	return request, measureErr
}
func namespaceMeasurementGit(runCtx context.Context, evidence, repo, name string, args ...string) (measureValue0 []byte, measureErr error) {
	command, commandErr := temporalCommand(runCtx, evidence, name, repo, nil, append([]string{"git"}, args...))
	if commandErr != nil {
		measureErr = commandErr
		return
	}
	return command.Stdout.RawBytes, measureErr
}
func namespaceMeasurementSource(runCtx context.Context, evidence, repo, role string) (measureValue0 string, measureValue1 map[string]string, measureErr error) {
	namespaceMeasurementGit2, namespaceMeasurementGitErr := namespaceMeasurementGit(runCtx, evidence, repo, "source-commit", "rev-parse", "HEAD")
	if namespaceMeasurementGitErr != nil {
		measureErr = namespaceMeasurementGitErr
		return
	}
	commit := strings.TrimSpace(string(namespaceMeasurementGit2))
	if !warmPreparationHex(commit, 40) {
		measureErr = errutil.New("source commit missing")
		return
	}
	status, namespaceMeasurementGitErr2 := namespaceMeasurementGit(runCtx, evidence, repo, "source-status", "status", "--porcelain", "--", "tools/gomad")
	if namespaceMeasurementGitErr2 != nil {
		measureErr = namespaceMeasurementGitErr2
		return
	}
	if len(status) != 0 {
		measureErr = errutil.New("freeze and commit collector source before measurement")
		return
	}
	if role == "baseline" {
		namespaceMeasurementGit3, namespaceMeasurementGitErr3 := namespaceMeasurementGit(runCtx, evidence, repo, "baseline-ancestor", "merge-base", "--is-ancestor", namespaceMeasurementBase, commit)
		if namespaceMeasurementGitErr3 != nil {
			measureErr = namespaceMeasurementGitErr3
			return
		}
		_ = namespaceMeasurementGit3
		changed, namespaceMeasurementGitErr4 := namespaceMeasurementGit(runCtx, evidence, repo, "baseline-production", "diff", "--name-only", namespaceMeasurementBase, commit, "--", ".")
		if namespaceMeasurementGitErr4 != nil {
			measureErr = namespaceMeasurementGitErr4
			return
		}
		for _, path := range strings.Fields(string(changed)) {
			if path != "tools/gomad/internal/backends/wasm/wasi/namespace_measurement_test.go" && path != "tools/gomad/internal/backends/wasm/wasi/namespace_measurement_controls_test.go" {
				measureErr = errutil.New(fmt.Sprint("frozen fn174 production/original driver changed", path))
				return
			}
		}
	}
	inputs := map[string]string{}
	tracked, namespaceMeasurementGitErr5 := namespaceMeasurementGit(runCtx, evidence, repo, "source-inventory", "ls-files", "-z", "tools/gomad")
	if namespaceMeasurementGitErr5 != nil {
		measureErr = namespaceMeasurementGitErr5
		return
	}
	for _, name := range bytes.Split(tracked, []byte{0}) {
		if len(name) > 0 {
			path := filepath.Join(repo, string(name))
			pathSHA256, pathSHA256Err := fileHash(runCtx, path)
			if pathSHA256Err != nil {
				measureErr = pathSHA256Err
				return
			}
			inputs[path] = pathSHA256
		}
	}
	return commit, inputs, measureErr
}
func namespaceMeasurementResultCheck(result wasi.Result, raw []byte, request wasi.Request, candidate bool, runError string) error {
	if runError != "" || result.Termination != "exit" || result.ExitCode == nil || *result.ExitCode != 0 || !result.Reaped || result.Cancelled || result.WatchdogTimeout || result.Started == nil || result.Started.Engine != request.Engine || result.Started.ModuleSHA256 != request.ModuleSHA256 || result.AuthorizationToTerminalNanos == 0 || result.Telemetry == nil || !result.Telemetry.CacheHit || result.Telemetry.CompileNanos != 0 {
		return fmt.Errorf("original Run lacks successful warm source-matched authorization/terminal evidence: %s", runError)
	}
	if !bytes.Contains(result.Stdout, []byte("=== RUN   TestActivityTestSuite")) || !bytes.Contains(result.Stdout, []byte("--- PASS: TestActivityTestSuite")) {
		return fmt.Errorf("unchanged activity selection did not pass")
	}
	_, err := namespaceMeasurementCounter(raw, candidate)
	if err != nil {
		return err
	}
	if sha256.Sum256(result.Transcript) != result.TranscriptSHA256 {
		return fmt.Errorf("raw WASI bytes differ from actual Run digest")
	}
	_, err = wasi.DecodeTranscript(result.Transcript)
	if err != nil {
		return err
	}
	return nil
}
func namespaceMeasurementLoad(runCtx context.Context, path string, common namespaceMeasurementCommon, expected wasi.Request) (measureValue0 namespaceMeasurementReceipt, measureValue1 wasi.Result, measureValue2 namespaceMeasurementSample, measureErr error) {
	var receipt namespaceMeasurementReceipt
	namespaceMeasurementReadJSONErr := namespaceMeasurementReadJSON(runCtx, path, &receipt)
	if namespaceMeasurementReadJSONErr != nil {
		measureErr = namespaceMeasurementReadJSONErr
		return
	}
	verifyInputsErr := temporalVerifyInputs(runCtx, receipt.Inputs)
	if verifyInputsErr != nil {
		measureErr = verifyInputsErr
		return
	}
	if receipt.Schema != "gomad.wasm-namespace-measurement-sample/v1" || receipt.Role != "baseline" && receipt.Role != "candidate" || !receipt.CollectionValid || receipt.RunError != "" || receipt.Failure != "" || receipt.ModelSHA256 == "" || receipt.HostBootID == "" || !warmPreparationHostCheck(wasi.PinnedEngine(), receipt.HostCPUInfo) {
		measureErr = errutil.New("failed or incomplete original namespace measurement collection")
		return
	}
	for _, bound := range []string{receipt.CommonPath, receipt.ResultPath, receipt.RequestPath, receipt.FinishPath, receipt.TranscriptPath, receipt.ChoicePath, receipt.StdoutPath, receipt.StderrPath, receipt.NetworkPath, receipt.HelperReceipt, receipt.ArtifactPath} {
		if !filepath.IsAbs(bound) || !warmPreparationHex(receipt.Inputs[bound], 64) {
			measureErr = errutil.New(fmt.Sprint("missing raw/source/build binding", bound))
			return
		}
	}
	var retainedCommon namespaceMeasurementCommon
	namespaceMeasurementReadJSONErr2 := namespaceMeasurementReadJSON(runCtx, receipt.CommonPath, &retainedCommon)
	if namespaceMeasurementReadJSONErr2 != nil {
		measureErr = namespaceMeasurementReadJSONErr2
		return
	}
	if !reflect.DeepEqual(common, retainedCommon) {
		measureErr = errutil.New("collection substituted common module/config/producer")
		return
	}
	err := warmPreparationProducerCheck(warmPreparationSeries{Inputs: receipt.Inputs, ModuleReceipt: common.ModuleReceipt}, common.ModuleReceipt, common.ProducerInputs)
	if err != nil {
		measureErr = err
		return
	}
	var request wasi.Request
	namespaceMeasurementReadJSONErr3 := namespaceMeasurementReadJSON(runCtx, receipt.RequestPath, &request)
	if namespaceMeasurementReadJSONErr3 != nil {
		measureErr = namespaceMeasurementReadJSONErr3
		return
	}
	if !reflect.DeepEqual(request, expected) {
		measureErr = errutil.New("actual source request differs from paired specification")
		return
	}
	var result wasi.Result
	raw, mustTerminalExitReadErr := mustTerminalExitRead(runCtx, receipt.ResultPath)
	if mustTerminalExitReadErr != nil {
		measureErr = mustTerminalExitReadErr
		return
	}
	err = json.Unmarshal(raw, &result)
	if err != nil {
		measureErr = err
		return
	}
	finish, mustTerminalExitReadErr2 := mustTerminalExitRead(runCtx, receipt.FinishPath)
	if mustTerminalExitReadErr2 != nil {
		measureErr = mustTerminalExitReadErr2
		return
	}
	if len(finish) != len(result.RuntimeTerminal()) {
		measureErr = errutil.New("private finish sidecar absent")
		return
	}
	setRuntimeTerminalBytes(&result, finish)
	for _, stream := range []struct {
		path string
		data []byte
	}{{receipt.TranscriptPath, result.Transcript}, {receipt.ChoicePath, result.ChoiceTrace.Bytes}, {receipt.StdoutPath, result.Stdout}, {receipt.StderrPath, result.Stderr}, {receipt.NetworkPath, result.NetworkTranscript}} {
		mustTerminalExitRead2, mustTerminalExitReadErr3 := mustTerminalExitRead(runCtx, stream.path)
		if mustTerminalExitReadErr3 != nil {
			measureErr = mustTerminalExitReadErr3
			return
		}
		if !bytes.Equal(mustTerminalExitRead2, stream.data) {
			measureErr = errutil.New(fmt.Sprint("raw original Run stream changed", stream.path))
			return
		}
	}
	err = namespaceMeasurementResultCheck(result, raw, request, receipt.Role == "candidate", receipt.RunError)
	if err != nil {
		measureErr = err
		return
	}
	evidence, err := warmPreparationBehavior(result, request.Config, false)
	if err != nil {
		measureErr = err
		return
	}
	counts, err := namespaceMeasurementCounter(raw, receipt.Role == "candidate")
	if err != nil {
		measureErr = err
		return
	}
	sample := namespaceMeasurementSample{Role: receipt.Role, Seed: receipt.Seed, Repetition: receipt.Repetition, AuthorizedNanos: result.AuthorizationToTerminalNanos, EvidenceSHA256: evidence, EnvironmentIdentity: result.EnvironmentIdentity, Calls: result.Calls, TimeAdvances: result.TimeAdvances, GoFilesystemCalls: counts, CounterPresent: receipt.Role == "candidate"}
	return receipt, result, sample, measureErr
}
func measureNamespaceSample(runCtx context.Context) (measureErr error) {
	root := input(runCtx, "GOMAD_WASM_NAMESPACE_SAMPLE_ROOT")
	if root == "" {
		measureErr = fmt.Errorf("%w: %v", ErrRefused, errutil.New("explicit source-matched sample collection only"))
		return
	}
	commonPath, helperReceipt, cache := input(runCtx, "GOMAD_WASM_NAMESPACE_COMMON"), input(runCtx, "GOMAD_WASM_NAMESPACE_HELPER_RECEIPT"), input(runCtx, "GOMAD_WASM_NAMESPACE_CACHE")
	for _, path := range []string{root, commonPath, helperReceipt, cache} {
		if !filepath.IsAbs(path) {
			measureErr = errutil.New("absolute sample/config/build/cache bindings required")
			return
		}
	}
	role := input(runCtx, "GOMAD_WASM_NAMESPACE_ROLE")
	if role != "baseline" && role != "candidate" {
		measureErr = errutil.New("explicit baseline/candidate source role required")
		return
	}
	seed, err := strconv.ParseUint(input(runCtx, "GOMAD_WASM_NAMESPACE_SEED"), 10, 64)
	if err != nil {
		measureErr = err
		return
	}
	repetition, err := strconv.Atoi(input(runCtx, "GOMAD_WASM_NAMESPACE_REPETITION"))
	if err != nil || repetition < -1 || repetition > 4 {
		measureErr = errutil.New("warmup (-1) or measured repetition (0..4) required")
		return
	}
	boot, cpu, err := warmPreparationHostEvidence(runCtx)
	if err != nil {
		measureErr = errutil.New(fmt.Sprint("current host evidence required", err))
		return
	}
	evidence, err := diagnosticRunRoot(root, "namespace-sample-")
	if err != nil {
		measureErr = err
		return
	}
	repo, err := absolute(runCtx, "../../../../../..")
	if err != nil {
		measureErr = err
		return
	}
	commit, inputs, namespaceMeasurementSourceErr := namespaceMeasurementSource(runCtx, evidence, repo, role)
	if namespaceMeasurementSourceErr != nil {
		measureErr = namespaceMeasurementSourceErr
		return
	}
	var common namespaceMeasurementCommon
	namespaceMeasurementReadJSONErr := namespaceMeasurementReadJSON(runCtx, commonPath, &common)
	if namespaceMeasurementReadJSONErr != nil {
		measureErr = namespaceMeasurementReadJSONErr
		return
	}
	if common.Schema != "gomad.wasm-namespace-measurement-common/v1" || common.Request.Config.Profile != wasi.CooperativeProfile || common.Request.Config.Storage != nil || common.Request.Config.NetworkV2 != nil || common.Request.Runtime == nil || common.Request.Runtime.Choice == nil || common.Request.DigestTranscript || common.Request.Broker != nil {
		measureErr = errutil.New("original standalone cooperative full-evidence request required")
		return
	}
	request, namespaceMeasurementRequestErr := namespaceMeasurementRequest(runCtx, common)
	if namespaceMeasurementRequestErr != nil {
		measureErr = namespaceMeasurementRequestErr
		return
	}
	request.Engine, request.CompileCache, request.Runtime.Seed = wasi.PinnedEngine(), cache, seed
	var admission terminalExitHelperAdmission
	namespaceMeasurementReadJSONErr2 := namespaceMeasurementReadJSON(runCtx, helperReceipt, &admission)
	if namespaceMeasurementReadJSONErr2 != nil {
		measureErr = namespaceMeasurementReadJSONErr2
		return
	}
	request.HelperPath, request.HelperSHA256 = admission.Helper, admission.HelperSHA256
	crate := filepath.Join(repo, "tools/gomad/internal/backends/wasm/wasmhost")
	terminalExitHelperInputs2, terminalExitHelperInputsErr := terminalExitHelperInputs(runCtx, helperReceipt, request, crate)
	if terminalExitHelperInputsErr != nil {
		measureErr = terminalExitHelperInputsErr
		return
	}
	maps.Copy(inputs, terminalExitHelperInputs2)
	if role == "candidate" {
		if repo != common.CurrentSourceRoot || commit != common.CurrentSourceCommit {
			measureErr = errutil.New("candidate source changed from common producer validation")
			return
		}
		terminalExitModuleInputs2, terminalExitModuleInputsErr := terminalExitModuleInputs(runCtx, common.ModuleReceipt, request)
		if terminalExitModuleInputsErr != nil {
			measureErr = terminalExitModuleInputsErr
			return
		}
		if !maps.Equal(terminalExitModuleInputs2, common.ProducerInputs) {
			measureErr = errutil.New("current producer closure changed")
			return
		}
	}
	verifyInputsErr := temporalVerifyInputs(runCtx, common.ProducerInputs)
	if verifyInputsErr != nil {
		measureErr = verifyInputsErr
		return
	}
	maps.Copy(inputs, common.ProducerInputs)
	commonPathSHA256, commonPathSHA256Err := fileHash(runCtx, commonPath)
	if commonPathSHA256Err != nil {
		measureErr = commonPathSHA256Err
		return
	}
	inputs[commonPath] = commonPathSHA256
	replayPath := input(runCtx, "GOMAD_WASM_NAMESPACE_REPLAY_TAPE")
	if replayPath != "" {
		if !filepath.IsAbs(replayPath) {
			measureErr = errutil.New("replay tape must be absolute")
			return
		}
		var tape choice.ReplayPlan
		namespaceMeasurementReadJSONErr3 := namespaceMeasurementReadJSON(runCtx, replayPath, &tape)
		if namespaceMeasurementReadJSONErr3 != nil {
			measureErr = namespaceMeasurementReadJSONErr3
			return
		}
		replay, err := namespaceReplayRequest(replayPath, request.Runtime.Choice, tape)
		if err != nil {
			return err
		}
		request.Runtime.Choice = &replay
		replayPathSHA256, replayPathSHA256Err := fileHash(runCtx, replayPath)
		if replayPathSHA256Err != nil {
			measureErr = replayPathSHA256Err
			return
		}
		inputs[replayPath] = replayPathSHA256
		inputs[replayPath+".trace"] = digest(replay.Trace)
	}
	warmPreparationArtifact2, warmPreparationArtifact22, warmPreparationArtifact3, warmPreparationArtifactErr := warmPreparationArtifact(runCtx, request)
	if warmPreparationArtifactErr != nil {
		measureErr = warmPreparationArtifactErr
		return
	}
	artifact, _, _ := warmPreparationArtifact2, warmPreparationArtifact22, warmPreparationArtifact3
	artifactSHA256, artifactSHA256Err := fileHash(runCtx, artifact)
	if artifactSHA256Err != nil {
		measureErr = artifactSHA256Err
		return
	}
	inputs[artifact] = artifactSHA256
	receipt := namespaceMeasurementReceipt{Schema: "gomad.wasm-namespace-measurement-sample/v1", Role: role, SourceRoot: repo, SourceCommit: commit, ModelSHA256: wasi.ImplementationSHA256(), HostBootID: strings.TrimSpace(string(boot)), HostCPUInfo: cpu, CommonPath: commonPath, HelperReceipt: helperReceipt, ArtifactPath: artifact, Seed: seed, Repetition: repetition, Replay: request.Runtime.Choice.Mode == choice.ModeReplay, Inputs: inputs}
	receipt.RequestPath = filepath.Join(evidence, "request.json")
	writeReportErr := temporalJSON(runCtx, receipt.RequestPath, request)
	if writeReportErr != nil {
		measureErr = writeReportErr
		return
	}
	requestPathSHA256, requestPathSHA256Err := fileHash(runCtx, receipt.RequestPath)
	if requestPathSHA256Err != nil {
		measureErr = requestPathSHA256Err
		return
	}
	inputs[receipt.RequestPath] = requestPathSHA256
	started := time.Now()
	result, runErr := warmPreparationAdmitted(runCtx, func() (wasi.Result, error) {
		callbackValue0, callbackValue1, callbackValue2 := (func(runCtx context.Context) (measureValue0 wasi.Result, measureValue1 error, measureErr error) {
			returnedValue0, returnedValue1 := wasi.Run(runCtx, request)
			return returnedValue0, returnedValue1, measureErr
		})(runCtx)
		return callbackValue0, errutil.Join(callbackValue1, callbackValue2)
	})
	receipt.WallNanos = uint64(time.Since(started).Nanoseconds())
	if runErr != nil {
		receipt.RunError = runErr.Error()
	}
	receipt.ResultPath, receipt.FinishPath = filepath.Join(evidence, "result.json"), filepath.Join(evidence, "finish.bin")
	writeReportErr2 := temporalJSON(runCtx, receipt.ResultPath, result)
	if writeReportErr2 != nil {
		measureErr = writeReportErr2
		return
	}
	writeEvidenceErr := temporalWrite(runCtx, receipt.FinishPath, runtimeTerminalBytes(result)[:])
	if writeEvidenceErr != nil {
		measureErr = writeEvidenceErr
		return
	}
	receipt.TranscriptPath, receipt.ChoicePath = filepath.Join(evidence, "wasi.bin"), filepath.Join(evidence, "choice.bin")
	receipt.StdoutPath, receipt.StderrPath, receipt.NetworkPath = filepath.Join(evidence, "stdout.bin"), filepath.Join(evidence, "stderr.bin"), filepath.Join(evidence, "network.bin")
	for _, stream := range []struct {
		path string
		data []byte
	}{{receipt.TranscriptPath, result.Transcript}, {receipt.ChoicePath, result.ChoiceTrace.Bytes}, {receipt.StdoutPath, result.Stdout}, {receipt.StderrPath, result.Stderr}, {receipt.NetworkPath, result.NetworkTranscript}} {
		writeEvidenceErr2 := temporalWrite(runCtx, stream.path, stream.data)
		if writeEvidenceErr2 != nil {
			measureErr = writeEvidenceErr2
			return
		}
	}
	for _, path := range []string{receipt.ResultPath, receipt.FinishPath, receipt.TranscriptPath, receipt.ChoicePath, receipt.StdoutPath, receipt.StderrPath, receipt.NetworkPath} {
		pathSHA256, pathSHA256Err := fileHash(runCtx, path)
		if pathSHA256Err != nil {
			measureErr = pathSHA256Err
			return
		}
		inputs[path] = pathSHA256
	}
	mustTerminalExitRead2, mustTerminalExitReadErr := mustTerminalExitRead(runCtx, receipt.ResultPath)
	if mustTerminalExitReadErr != nil {
		measureErr = mustTerminalExitReadErr
		return
	}
	err = namespaceMeasurementResultCheck(result, mustTerminalExitRead2, request, role == "candidate", receipt.RunError)
	if err != nil {
		receipt.Failure = err.Error()
	}
	afterBoot, afterCPU, err := warmPreparationHostEvidence(runCtx)
	if err != nil || boot != afterBoot || cpu != afterCPU {
		receipt.Failure = "host changed during collection"
	}
	namespaceMeasurementSource2, namespaceMeasurementSource22, namespaceMeasurementSourceErr2 := namespaceMeasurementSource(runCtx, evidence, repo, role)
	if namespaceMeasurementSourceErr2 != nil {
		measureErr = namespaceMeasurementSourceErr2
		return
	}
	afterCommit, _ := namespaceMeasurementSource2, namespaceMeasurementSource22
	if afterCommit != commit {
		receipt.Failure = "source changed during collection"
	}
	verifyInputsErr2 := temporalVerifyInputs(runCtx, inputs)
	if verifyInputsErr2 != nil {
		measureErr = verifyInputsErr2
		return
	}
	terminalExitHelperInputs3, terminalExitHelperInputsErr2 := terminalExitHelperInputs(runCtx, helperReceipt, request, crate)
	if terminalExitHelperInputsErr2 != nil {
		measureErr = terminalExitHelperInputsErr2
		return
	}
	_ = terminalExitHelperInputs3
	if role == "candidate" {
		terminalExitModuleInputs3, terminalExitModuleInputsErr2 := terminalExitModuleInputs(runCtx, common.ModuleReceipt, request)
		if terminalExitModuleInputsErr2 != nil {
			measureErr = terminalExitModuleInputsErr2
			return
		}
		_ = terminalExitModuleInputs3
	}
	receipt.CollectionValid = receipt.Failure == "" && receipt.RunError == ""
	receiptPath := filepath.Join(evidence, "sample.json")
	writeReportErr3 := temporalJSON(runCtx, receiptPath, receipt)
	if writeReportErr3 != nil {
		measureErr = writeReportErr3
		return
	}
	output := input(runCtx, "GOMAD_WASM_NAMESPACE_SAMPLE_INDEX")
	if !filepath.IsAbs(output) {
		measureErr = errutil.New("absolute controller-owned sample index required")
		return
	}
	receiptPathSHA256, receiptPathSHA256Err := fileHash(runCtx, receiptPath)
	if receiptPathSHA256Err != nil {
		measureErr = receiptPathSHA256Err
		return
	}
	writeReportErr4 := temporalJSON(runCtx, output, map[string]string{"receipt": receiptPath, "sha256": receiptPathSHA256})
	if writeReportErr4 != nil {
		measureErr = writeReportErr4
		return
	}
	if !receipt.CollectionValid {
		measureErr = fmt.Errorf("namespace collection refused: %s %s; retained %s", receipt.RunError, receipt.Failure, evidence)
		return
	}
	return
}
func measureNamespaceActivity(runCtx context.Context) (measureErr error) {
	root := input(runCtx, "GOMAD_WASM_NAMESPACE_MEASUREMENT_ROOT")
	if root == "" {
		measureErr = fmt.Errorf("%w: %v", ErrRefused, errutil.New("explicit paired activity measurement only"))
		return
	}
	paths := map[string]string{"root": root, "baseline_source": input(runCtx, "GOMAD_WASM_NAMESPACE_BASELINE_SOURCE"), "baseline_helper_receipt": input(runCtx, "GOMAD_WASM_NAMESPACE_BASELINE_HELPER_RECEIPT"), "candidate_helper_receipt": input(runCtx, "GOMAD_WASM_NAMESPACE_HELPER_RECEIPT"), "baseline_cache": input(runCtx, "GOMAD_WASM_NAMESPACE_BASELINE_CACHE"), "candidate_cache": input(runCtx, "GOMAD_WASM_NAMESPACE_CACHE"), "module": input(runCtx, "GOMAD_WASM_NAMESPACE_MODULE"), "module_receipt": input(runCtx, "GOMAD_WASM_NAMESPACE_MODULE_RECEIPT"), "stock_go": input(runCtx, "GOMAD_WASM_NAMESPACE_STOCK_GO")}
	for name, path := range paths {
		if !filepath.IsAbs(path) {
			measureErr = errutil.New(fmt.Sprint("explicit absolute paired collection input required", name))
			return
		}
	}
	_, host := currentDiagnosticPerformanceHostEvidence(runCtx)
	if !host {
		measureErr = errutil.New("current performance host required")
		return
	}
	err := matchedBaselineCache(paths["baseline_cache"], paths["candidate_cache"])
	if err != nil {
		measureErr = err
		return
	}
	evidence, err := diagnosticRunRoot(root, "namespace-paired-")
	if err != nil {
		measureErr = err
		return
	}
	version, commandErr := temporalCommand(runCtx, evidence, "stock-go-version", evidence, append(environ(runCtx), "GOROOT=", "GOTOOLCHAIN=local"), []string{paths["stock_go"], "version"})
	if commandErr != nil {
		measureErr = commandErr
		return
	}
	if !strings.HasPrefix(string(version.Stdout.RawBytes), "go version go1.27.1 "+runtime.GOOS+"/"+runtime.GOARCH) {
		measureErr = errutil.New("source drivers require stock Go 1.27.1 on the measured host")
		return
	}
	setInput(runCtx, "GOMAD_WASM_PROTOTYPE_ROOT", evidence)
	setInput(runCtx, "GOMAD_STOCK_GO", paths["stock_go"])
	var candidateAdmission terminalExitHelperAdmission
	namespaceMeasurementReadJSONErr := namespaceMeasurementReadJSON(runCtx, paths["candidate_helper_receipt"], &candidateAdmission)
	if namespaceMeasurementReadJSONErr != nil {
		measureErr = namespaceMeasurementReadJSONErr
		return
	}
	setInput(runCtx, "GOMAD_WASM_TEST_HELPER", candidateAdmission.Helper)
	repo, err := absolute(runCtx, "../../../../../..")
	if err != nil {
		measureErr = err
		return
	}
	if repo == paths["baseline_source"] {
		measureErr = errutil.New("separate immutable baseline source and driver required")
		return
	}
	config, prototypeTemporalConfigErr := prototypeTemporalConfig(runCtx, repo, "/workspace/tests", "-test.run=^TestActivityTestSuite$", "-persistenceType=sql", "-persistenceDriver=sqlite")
	if prototypeTemporalConfigErr != nil {
		measureErr = prototypeTemporalConfigErr
		return
	}
	request, requestErr := prototypeRequest(runCtx, paths["module"], config, 0, 30*time.Minute)
	if requestErr != nil {
		measureErr = requestErr
		return
	}
	request.DigestTranscript, request.CompileCache = false, paths["candidate_cache"]
	commit, sourceInputs, namespaceMeasurementSourceErr := namespaceMeasurementSource(runCtx, evidence, repo, "candidate")
	if namespaceMeasurementSourceErr != nil {
		measureErr = namespaceMeasurementSourceErr
		return
	}
	namespaceMeasurementSource2, namespaceMeasurementSource22, namespaceMeasurementSourceErr2 := namespaceMeasurementSource(runCtx, evidence, paths["baseline_source"], "baseline")
	if namespaceMeasurementSourceErr2 != nil {
		measureErr = namespaceMeasurementSourceErr2
		return
	}
	_, _ = namespaceMeasurementSource2, namespaceMeasurementSource22
	terminalExitModuleInputs2, terminalExitModuleInputsErr := terminalExitModuleInputs(runCtx, paths["module_receipt"], request)
	if terminalExitModuleInputsErr != nil {
		measureErr = terminalExitModuleInputsErr
		return
	}
	common := namespaceMeasurementCommon{Schema: "gomad.wasm-namespace-measurement-common/v1", CurrentSourceRoot: repo, CurrentSourceCommit: commit, ModuleReceipt: paths["module_receipt"], Request: request, ProducerInputs: terminalExitModuleInputs2}
	commonPath := filepath.Join(evidence, "common.json")
	writeReportErr := temporalJSON(runCtx, commonPath, common)
	if writeReportErr != nil {
		measureErr = writeReportErr
		return
	}
	inputs := maps.Clone(sourceInputs)
	maps.Copy(inputs, common.ProducerInputs)
	commonPathSHA256, commonPathSHA256Err := fileHash(runCtx, commonPath)
	if commonPathSHA256Err != nil {
		measureErr = commonPathSHA256Err
		return
	}
	digest, digestErr := fileHash(runCtx, paths["stock_go"])
	if digestErr != nil {
		measureErr = digestErr
		return
	}
	inputs[commonPath], inputs[paths["stock_go"]] = commonPathSHA256, digest
	var baseline, candidate []namespaceMeasurementSample
	var baselineResults []wasi.Result
	var rawReceipts []string
	var prior [2]namespaceMeasurementReceipt
	var order []map[string]any
	collect := func(runCtx context.Context, role string, seed uint64, repetition int, replayPath string) (measureValue0 namespaceMeasurementSample, measureValue1 wasi.Result, measureErr error) {
		index := 0
		owner := repo
		if role == "candidate" {
			index = 1
		} else {
			owner = paths["baseline_source"]
		}
		tag := fmt.Sprintf("seed%d-%s-rep%d", seed, role, repetition)
		if replayPath != "" {
			tag += "-replay"
		}
		sampleRoot := filepath.Join(evidence, tag)
		err := os.Mkdir(sampleRoot, 0700)
		if err != nil {
			measureErr = err
			return
		}
		indexPath := filepath.Join(sampleRoot, "index.json")
		environment := append(environ(runCtx), "GOROOT=", "GOTOOLCHAIN=local", "GOFLAGS=", "GOMAXPROCS=2", "GOENV=off", "GOWORK=off", "GOOS=", "GOARCH=", "GOEXPERIMENT=", "GOWASM=", "GOMAD_WASM_NAMESPACE_MEASUREMENT_ROOT=", "GOMAD_WASM_NAMESPACE_SAMPLE_ROOT="+sampleRoot, "GOMAD_WASM_NAMESPACE_COMMON="+commonPath, "GOMAD_WASM_NAMESPACE_ROLE="+role, "GOMAD_WASM_NAMESPACE_HELPER_RECEIPT="+paths[role+"_helper_receipt"], "GOMAD_WASM_NAMESPACE_CACHE="+paths[role+"_cache"], "GOMAD_WASM_NAMESPACE_SEED="+strconv.FormatUint(seed, 10), "GOMAD_WASM_NAMESPACE_REPETITION="+strconv.Itoa(repetition), "GOMAD_WASM_NAMESPACE_REPLAY_TAPE="+replayPath, "GOMAD_WASM_NAMESPACE_SAMPLE_INDEX="+indexPath)
		command := []string{paths["stock_go"], "test", "-p", "1", "-tags=test_dep", "-count=1", "-timeout=35m", "-run=^TestNamespaceMeasurementSample$", "./internal/backends/wasm/wasi"}
		_, err = os.Stat(filepath.Join(owner, "tools/gomad/internal/backends/wasm/cmd/wasmmeasure"))
		if err == nil {
			accepted := map[string]bool{}
			for _, input := range measurement.Inputs("namespace-sample") {
				accepted[input.Variable] = true
			}
			inputs := map[string]string{}
			var kept []string
			for _, entry := range environment {
				key, value, _ := strings.Cut(entry, "=")
				if !strings.HasPrefix(key, "GOMAD_") {
					kept = append(kept, entry)
				} else if accepted[key] {
					inputs[key] = value
				}
			}
			environment = kept
			wasmmeasure := measurement.Command(paths["stock_go"], owner, "namespace-sample", inputs)
			command = append(append([]string{wasmmeasure[0], "run", "-p", "1"}, wasmmeasure[2:]...), "--go", paths["stock_go"])
		}
		command2, commandErr2 := temporalCommand(runCtx, evidence, tag, filepath.Join(owner, "tools/gomad"), environment, command)
		if commandErr2 != nil {
			measureErr = commandErr2
			return
		}
		_ = command2
		var sampleIndex map[string]string
		namespaceMeasurementReadJSONErr2 := namespaceMeasurementReadJSON(runCtx, indexPath, &sampleIndex)
		if namespaceMeasurementReadJSONErr2 != nil {
			measureErr = namespaceMeasurementReadJSONErr2
			return
		}
		path := sampleIndex["receipt"]
		if !filepath.IsAbs(path) {
			return namespaceMeasurementSample{}, wasi.Result{}, fmt.Errorf("source collector index changed")
		}
		pathSHA256, pathSHA256Err := fileHash(runCtx, path)
		if pathSHA256Err != nil {
			measureErr = pathSHA256Err
			return
		}
		if sampleIndex["sha256"] != pathSHA256 {
			measureErr = errutil.New("source collector index changed")
			return
		}
		var admission terminalExitHelperAdmission
		namespaceMeasurementReadJSONErr3 := namespaceMeasurementReadJSON(runCtx, paths[role+"_helper_receipt"], &admission)
		if namespaceMeasurementReadJSONErr3 != nil {
			measureErr = namespaceMeasurementReadJSONErr3
			return
		}
		expected, namespaceMeasurementRequestErr := namespaceMeasurementRequest(runCtx, common)
		if namespaceMeasurementRequestErr != nil {
			measureErr = namespaceMeasurementRequestErr
			return
		}
		expected.HelperPath, expected.HelperSHA256, expected.Engine, expected.CompileCache = admission.Helper, admission.HelperSHA256, wasi.PinnedEngine(), paths[role+"_cache"]
		expected.Runtime.Seed = seed
		if replayPath != "" {
			var tape choice.ReplayPlan
			namespaceMeasurementReadJSONErr4 := namespaceMeasurementReadJSON(runCtx, replayPath, &tape)
			if namespaceMeasurementReadJSONErr4 != nil {
				measureErr = namespaceMeasurementReadJSONErr4
				return
			}
			replay, err := namespaceReplayRequest(replayPath, expected.Runtime.Choice, tape)
			if err != nil {
				return namespaceMeasurementSample{}, wasi.Result{}, err
			}
			expected.Runtime.Choice = &replay
		}
		bound, result, sample, namespaceMeasurementLoadErr := namespaceMeasurementLoad(runCtx, path, common, expected)
		if namespaceMeasurementLoadErr != nil {
			measureErr = namespaceMeasurementLoadErr
			return
		}
		if bound.Role != role || bound.SourceRoot != owner || bound.Seed != seed || bound.Repetition != repetition || bound.Replay != (replayPath != "") {
			measureErr = errutil.New("source collector substituted role/source/seed/repetition/replay")
			return
		}
		if index == 1 && bound.ModelSHA256 != wasi.ImplementationSHA256() {
			measureErr = errutil.New("current Run model binding changed")
			return
		}
		if prior[index].SourceCommit != "" && (bound.SourceCommit != prior[index].SourceCommit || bound.ModelSHA256 != prior[index].ModelSHA256 || bound.HostBootID != prior[index].HostBootID || bound.HostCPUInfo != prior[index].HostCPUInfo) {
			measureErr = errutil.New("source/model/host changed within series")
			return
		}
		other := prior[1-index]
		if other.SourceCommit != "" && (bound.HostBootID != other.HostBootID || bound.HostCPUInfo != other.HostCPUInfo) {
			measureErr = errutil.New("baseline/candidate did not share real host")
			return
		}
		terminalExitHelperInputs2, terminalExitHelperInputsErr := terminalExitHelperInputs(runCtx, bound.HelperReceipt, expected, filepath.Join(owner, "tools/gomad/internal/backends/wasm/wasmhost"))
		if terminalExitHelperInputsErr != nil {
			measureErr = terminalExitHelperInputsErr
			return
		}
		_ = terminalExitHelperInputs2
		prior[index] = bound
		pathSHA2562, pathSHA256Err2 := fileHash(runCtx, path)
		if pathSHA256Err2 != nil {
			measureErr = pathSHA256Err2
			return
		}
		indexPathSHA256, indexPathSHA256Err := fileHash(runCtx, indexPath)
		if indexPathSHA256Err != nil {
			measureErr = indexPathSHA256Err
			return
		}
		inputs[path], inputs[indexPath] = pathSHA2562, indexPathSHA256
		maps.Copy(inputs, bound.Inputs)
		rawReceipts = append(rawReceipts, path)
		order = append(order, map[string]any{"role": role, "seed": seed, "repetition": repetition, "replay": replayPath != "", "receipt": path, "authorized_nanos": sample.AuthorizedNanos, "wall_nanos": bound.WallNanos})
		writeReportErr2 := temporalJSON(runCtx, filepath.Join(evidence, "collection-order.json"), order)
		if writeReportErr2 != nil {
			measureErr = writeReportErr2
			return
		}
		return sample, result, measureErr
	}
	var medians []map[string]any
	for seedIndex, seed := range []uint64{0, 1000003, 2000006, 3000009} {
		collect2, collect22, collectErr := collect(runCtx, "baseline", seed, -1, "")
		if collectErr != nil {
			measureErr = collectErr
			return
		}
		_, warmBefore := collect2, collect22
		collect3, collect23, collectErr2 := collect(runCtx, "candidate", seed, -1, "")
		if collectErr2 != nil {
			measureErr = collectErr2
			return
		}
		_, warmAfter := collect3, collect23
		difference := sameEvidence(warmBefore, warmAfter)
		if difference != "" {
			measureErr = errutil.New(fmt.Sprint("source-matched warmup evidence changed", difference))
			return
		}
		for repetition := range 5 {
			var samples [2]namespaceMeasurementSample
			var results [2]wasi.Result
			for turn := range 2 {
				index := turn
				if (seedIndex*5+repetition)%2 != 0 {
					index = 1 - turn
				}
				role := "baseline"
				if index == 1 {
					role = "candidate"
				}
				collect4, collect24, collectErr3 := collect(runCtx, role, seed, repetition, "")
				if collectErr3 != nil {
					measureErr = collectErr3
					return
				}
				samples[index], results[index] = collect4, collect24
			}
			difference := sameEvidence(results[0], results[1])
			if difference != "" {
				measureErr = errutil.New(fmt.Sprint("same-module paired evidence changed", seed, repetition, difference))
				return
			}
			difference = sameEvidence(warmBefore, results[0])
			if difference != "" {
				measureErr = errutil.New(fmt.Sprint("baseline warmup/sample changed", difference))
				return
			}
			difference = sameEvidence(warmAfter, results[1])
			if difference != "" {
				measureErr = errutil.New(fmt.Sprint("candidate warmup/sample changed", difference))
				return
			}
			baseline, candidate = append(baseline, samples[0]), append(candidate, samples[1])
			baselineResults = append(baselineResults, results[0])
		}
		first := baselineResults[seedIndex*5]
		tape, err := choice.ProjectReplayPlan(first.ChoiceTrace, request.Runtime.Choice.ExecutionIdentity)
		if err != nil {
			measureErr = err
			return
		}
		tapePath := filepath.Join(evidence, fmt.Sprintf("seed%d-replay-plan.json", seed))
		writeReportErr3 := temporalJSON(runCtx, tapePath, tape)
		if writeReportErr3 != nil {
			measureErr = writeReportErr3
			return
		}
		if err := temporalWrite(runCtx, tapePath+".trace", first.ChoiceTrace.Bytes); err != nil {
			return err
		}
		inputs[tapePath+".trace"] = fmt.Sprintf("sha256:%x", sha256.Sum256(first.ChoiceTrace.Bytes))
		tapePathSHA256, tapePathSHA256Err := fileHash(runCtx, tapePath)
		if tapePathSHA256Err != nil {
			measureErr = tapePathSHA256Err
			return
		}
		inputs[tapePath] = tapePathSHA256
		for _, role := range []string{"baseline", "candidate"} {
			collect5, collect25, collectErr4 := collect(runCtx, role, seed, 0, tapePath)
			if collectErr4 != nil {
				measureErr = collectErr4
				return
			}
			_, replayed := collect5, collect25
			difference := sameEvidence(first, replayed)
			if difference != "" || replayed.EnvironmentIdentity != first.EnvironmentIdentity || replayed.Calls != first.Calls || replayed.TimeAdvances != first.TimeAdvances {
				measureErr = errutil.New(fmt.Sprint("fresh exact baseline-tape replay changed", seed, role, difference))
				return
			}
		}
		var b, a []uint64
		i := seedIndex * 5
		for ; i < (seedIndex+1)*5; i++ {
			b, a = append(b, baseline[i].AuthorizedNanos), append(a, candidate[i].AuthorizedNanos)
		}
		medians = append(medians, map[string]any{"seed": seed, "baseline_authorized_nanos": variantTiming(b), "candidate_authorized_nanos": variantTiming(a)})
	}
	failure := ""
	err = namespaceMeasurementCompare(baseline, candidate)
	if err != nil {
		failure = err.Error()
	}
	if prior[0].ModelSHA256 == prior[1].ModelSHA256 {
		failure = "namespace image source must rotate model identity; historical model was substituted"
	}
	verifyInputsErr := temporalVerifyInputs(runCtx, inputs)
	if verifyInputsErr != nil {
		measureErr = verifyInputsErr
		return
	}
	terminalExitModuleInputs3, terminalExitModuleInputsErr2 := terminalExitModuleInputs(runCtx, common.ModuleReceipt, request)
	if terminalExitModuleInputsErr2 != nil {
		measureErr = terminalExitModuleInputsErr2
		return
	}
	_ = terminalExitModuleInputs3
	for _, source := range []struct{ root, role, commit string }{{repo, "candidate", commit}, {paths["baseline_source"], "baseline", prior[0].SourceCommit}} {
		namespaceMeasurementSource3, namespaceMeasurementSource23, namespaceMeasurementSourceErr3 := namespaceMeasurementSource(runCtx, evidence, source.root, source.role)
		if namespaceMeasurementSourceErr3 != nil {
			measureErr = namespaceMeasurementSourceErr3
			return
		}
		frozen, _ := namespaceMeasurementSource3, namespaceMeasurementSource23
		if frozen != source.commit {
			failure = "source changed after paired collection"
		}
	}
	writeReportErr4 := temporalJSON(runCtx, filepath.Join(evidence, "measurement.json"), map[string]any{"schema": "gomad.wasm-namespace-activity-measurement/v1", "accepted": failure == "", "failure": failure, "baseline_source": prior[0], "candidate_source": prior[1], "baseline": baseline, "candidate": candidate, "per_seed_nanos": medians, "raw_receipts": rawReceipts, "collection_order": order, "inputs": inputs, "boundary": "five fresh helpers per matched seed on existing warm caches; actual Run authorization through validated terminal; full raw evidence and per-seed full-tape replay; alternating paired source drivers; no cold-compilation measurement", "baseline_base": namespaceMeasurementBase})
	if writeReportErr4 != nil {
		measureErr = writeReportErr4
		return
	}
	if failure != "" {
		measureErr = fmt.Errorf("namespace activity requirement not met: %s; retained %s", failure, evidence)
		return
	}
	return
}
func sameStorageEvidence(reference, local wasi.Result) string {
	diff := sameEvidence(reference, local)
	if diff != "" {
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

var diagnosticHelperFeatures = [][]string{{"diagnostic-profile"}, {"diagnostic-profile", "guest-profile"}, {"diagnostic-fuel-off", "diagnostic-profile"}}

func measureDiagnosticHelperBuild(runCtx context.Context) (measureErr error) {
	root := input(runCtx, "GOMAD_WASM_DIAGNOSTIC_HELPER_BUILD_ROOT")
	if root == "" {
		measureErr = fmt.Errorf("%w: %v", ErrRefused, errutil.New("set the diagnostic helper build root to retain a pinned, feature-exact diagnostic helper build and admission"))
		return
	}
	features := strings.Split(input(runCtx, "GOMAD_WASM_DIAGNOSTIC_HELPER_FEATURES"), ",")
	slices.Sort(features)
	if !slices.ContainsFunc(diagnosticHelperFeatures, func(admitted []string) bool {
		return slices.Equal(admitted, features)
	}) {
		measureErr = fmt.Errorf("diagnostic helper features %v are not an admitted build %v", features, diagnosticHelperFeatures)
		return
	}
	terminalExitBuildHelperErr := terminalExitBuildHelper(runCtx, root, features)
	if terminalExitBuildHelperErr != nil {
		measureErr = terminalExitBuildHelperErr
		return
	}
	return
}
func warmPreparationDarwinAdmitted(stats string, dispatch func() (wasi.Result, error)) (wasi.Result, error) {
	header, rest, ok := strings.Cut(stats, "\n")
	pageSizeText := strings.TrimSuffix(strings.TrimPrefix(header, "Mach Virtual Memory Statistics: (page size of "), " bytes)")
	pageSize, err := matchedMemoryDecimal(pageSizeText)
	if len(stats) > diagnosticHostCPUInfoLimit || !ok || !strings.HasPrefix(header, "Mach Virtual Memory Statistics: (page size of ") || !strings.HasSuffix(header, " bytes)") || err != nil || pageSize == 0 {
		return wasi.Result{}, errutil.New("vm_stat page size required")
	}
	seen := map[string]bool{}
	var pages uint64
	for line := range strings.SplitSeq(rest, "\n") {
		key, value, ok := strings.Cut(line, ":")
		if key != "Pages free" && key != "Pages inactive" {
			continue
		}
		count, err := matchedMemoryDecimal(strings.TrimSuffix(strings.TrimSpace(value), "."))
		if !ok || seen[key] || err != nil || count > math.MaxUint64-pages {
			return wasi.Result{}, errutil.New("unique nonoverflowing vm_stat available page counts required")
		}
		seen[key] = true
		pages += count
	}
	if len(seen) != 2 || pages > math.MaxUint64/pageSize || pages*pageSize < 8<<30 {
		return wasi.Result{}, errutil.New("at least 8GiB free plus inactive memory required before matched preparation dispatch")
	}
	return dispatch()
}
func warmPreparationHostCommand(ctx context.Context, command ...string) (string, error) {
	result, err := hostexec.Run(ctx, hostexec.Request{Command: command, Dir: ".", Timeout: 10 * time.Second, OutputLimit: diagnosticHostCPUInfoLimit})
	if err != nil {
		return "", fmt.Errorf("host evidence command %v: %w", command, err)
	}
	if result.Termination != hostexec.TerminationExit || result.ExitCode != 0 || result.Cancelled || result.WatchdogTimeout || !result.GroupGone || result.Stdout.TotalBytes > diagnosticHostCPUInfoLimit {
		return "", fmt.Errorf("host evidence command %v: termination=%s exit=%d cancelled=%t watchdog=%t reaped=%t output_bytes=%d", command, result.Termination, result.ExitCode, result.Cancelled, result.WatchdogTimeout, result.GroupGone, result.Stdout.TotalBytes)
	}
	return strings.TrimSpace(string(result.Stdout.RawBytes)), nil
}
func warmPreparationAdmitted(ctx context.Context, dispatch func() (wasi.Result, error)) (wasi.Result, error) {
	if runtime.GOOS == "darwin" {
		stats, err := warmPreparationHostCommand(ctx, "vm_stat")
		if err != nil {
			return wasi.Result{}, err
		}
		return warmPreparationDarwinAdmitted(stats, dispatch)
	}
	return matchedPrepareAdmitted(runtime.GOOS, "/proc/meminfo", "1", dispatch)
}
func warmPreparationHostCheck(engine wasi.EngineIdentity, cpuinfo string) bool {
	if len(cpuinfo) == 0 || len(cpuinfo) > diagnosticHostCPUInfoLimit {
		return false
	}
	if engine.HostOS == "linux" && engine.HostArch == "aarch64" && diagnosticPerformanceHost("linux", "arm64", cpuinfo) {
		return true
	}
	var witness warmPreparationHostWitness
	if json.Unmarshal([]byte(cpuinfo), &witness) != nil || witness.KernelVersion == "" || witness.CPUInfo == "" || (witness.Translation != "" && witness.Translation != "0") || warmPreparationHostArch(witness.Machine) != engine.HostArch || engine.HostArch == "" {
		return false
	}
	os := strings.ToLower(witness.KernelOS)
	if os == "darwin" {
		os = "macos"
	}
	if os == "" || os != engine.HostOS {
		return false
	}
	switch os {
	case "linux":
		switch engine.HostArch {
		case "aarch64":
			return diagnosticPerformanceHost("linux", "arm64", witness.CPUInfo)
		case "x86_64":
			return strings.Contains(witness.CPUInfo, "vendor_id:") || strings.Contains(witness.CPUInfo, "vendor_id\t:")
		}
	case "macos":
		switch engine.HostArch {
		case "aarch64":
			return strings.Contains(witness.CPUInfo, "hw.optional.arm64: 1\n")
		case "x86_64":
			return strings.Contains(witness.CPUInfo, "hw.optional.arm64: 0\n")
		}
	}
	return warmPreparationHostArch(strings.TrimSpace(witness.CPUInfo)) == engine.HostArch
}
func warmPreparationHostEvidence(ctx context.Context) (boot, cpuinfo string, err error) {
	var witness warmPreparationHostWitness
	for _, field := range []struct {
		flag  string
		value *string
	}{{"-s", &witness.KernelOS}, {"-m", &witness.Machine}, {"-v", &witness.KernelVersion}} {
		*field.value, err = warmPreparationHostCommand(ctx, "uname", field.flag)
		if err != nil {
			return "", "", err
		}
	}
	switch runtime.GOOS {
	case "linux":
		var data []byte
		data, err = hostfs.ReadBounded("/proc/sys/kernel/random/boot_id", 128)
		boot = strings.TrimSpace(string(data))
		if err == nil {
			data, err = hostfs.ReadBounded("/proc/cpuinfo", diagnosticHostCPUInfoLimit)
			witness.CPUInfo = string(data)
		}
	case "darwin":
		boot, err = warmPreparationHostCommand(ctx, "sysctl", "-n", "kern.boottime")
		if err == nil {
			var arm, brand string
			arm, err = warmPreparationHostCommand(ctx, "sysctl", "-n", "hw.optional.arm64")
			if err == nil {
				brand, err = warmPreparationHostCommand(ctx, "sysctl", "-n", "machdep.cpu.brand_string")
			}
			witness.CPUInfo = "hw.optional.arm64: " + arm + "\nmachdep.cpu.brand_string: " + brand
			if err == nil {
				witness.Translation, err = warmPreparationHostCommand(ctx, "sysctl", "-in", "sysctl.proc_translated")
			}
		}
	default:
		boot, err = warmPreparationHostCommand(ctx, "sysctl", "-n", "kern.boottime")
		if err == nil {
			witness.CPUInfo, err = warmPreparationHostCommand(ctx, "uname", "-p")
		}
	}
	if err != nil {
		return "", "", err
	}
	if runtime.GOOS == "linux" && runtime.GOARCH == "arm64" && diagnosticPerformanceHost("linux", "arm64", witness.CPUInfo) {
		if boot == "" {
			return "", "", errutil.New("host boot identity absent")
		}
		return boot, witness.CPUInfo, nil
	}
	data, err := json.Marshal(witness)
	cpuinfo = string(data)
	if err != nil || boot == "" || !warmPreparationHostCheck(wasi.PinnedEngine(), cpuinfo) {
		return "", "", fmt.Errorf("host evidence does not bind the current engine host")
	}
	return boot, cpuinfo, nil
}

type warmPreparationHostWitness struct{ KernelOS, Machine, KernelVersion, CPUInfo, Translation string }

func warmPreparationHostArch(arch string) string {
	switch arch {
	case "arm64":
		return "aarch64"
	case "amd64":
		return "x86_64"
	default:
		return arch
	}
}
