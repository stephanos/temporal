package wasi

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"go.temporal.io/server/tools/gomad3/captured"
	"go.temporal.io/server/tools/gomad3/choice"
	"go.temporal.io/server/tools/gomad3/hostexec"
	"go.temporal.io/server/tools/gomad3/volume"
	"go.temporal.io/server/tools/gomad3/world"
	"go.temporal.io/server/tools/gomad3sim"
)

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
	root, err := filepath.Abs("../../../../..")
	if err != nil {
		t.Fatal(err)
	}
	fixtureBody, err := os.ReadFile(filepath.Join(root, "tools/gomad3/backends/wasm/testdata/temporal_storage/recovery_test.go"))
	if err != nil {
		t.Fatal(err)
	}
	if fixture == "temporal_broker" {
		brokerBody, err := os.ReadFile(filepath.Join(root, "tools/gomad3/backends/wasm/testdata/temporal_broker/replication_test.go"))
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
	inputs := temporalSourceInputs(t, listed.Stdout)
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
	for _, name := range []string{"go.mod", "go.sum", "tools/gomad3/backends/wasm/testdata/temporal_storage/recovery_test.go"} {
		inputs[filepath.Join(root, name)] = fileHash(t, filepath.Join(root, name))
	}
	if fixture == "temporal_broker" {
		file := filepath.Join(root, "tools/gomad3/backends/wasm/testdata/temporal_broker/replication_test.go")
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
