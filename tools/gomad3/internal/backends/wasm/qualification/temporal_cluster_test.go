package qualification

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	runnerbackend "go.temporal.io/server/tools/gomad3/internal/backends"
	"go.temporal.io/server/tools/gomad3/internal/backends/target"
	"go.temporal.io/server/tools/gomad3/internal/backends/wasm/backend"
	simulation "go.temporal.io/server/tools/gomad3/internal/backends/wasm/cluster"
	"go.temporal.io/server/tools/gomad3/internal/backends/wasm/wasi"
	"go.temporal.io/server/tools/gomad3/internal/evidence/record"
	hostexec "go.temporal.io/server/tools/gomad3/internal/host/exec"
	hostfs "go.temporal.io/server/tools/gomad3/internal/host/fs"
	"go.temporal.io/server/tools/gomad3/internal/model/choice"
	gomad3sim "go.temporal.io/server/tools/gomad3/internal/model/cluster"
	netmodel "go.temporal.io/server/tools/gomad3/internal/model/network"
	"go.temporal.io/server/tools/gomad3/internal/model/world"
	"go.temporal.io/server/tools/gomad3/internal/runner"
)

func TestTemporalQualificationSealsStrictBidirectionalTopology(t *testing.T) {
	configuration := wasi.BrokerConfig{
		Nodes: []wasi.BrokerNode{{ID: "active", Incarnation: 1}, {ID: "standby", Incarnation: 1}}, Seed: 7,
		WorldProfile: world.CompressedSnapshotSchema, Limits: world.CompressedSnapshotProfile().Limits,
		Network:   &wasi.NetworkModelV2{Profile: wasi.NetworkStreamProfile, Limits: netmodel.Limits{Listeners: 128, Connections: 4096, Bytes: 128 << 20, ConnectionBytes: 4 << 20, StallNanos: 10_000_000_000, Transitions: 1 << 20}},
		Addresses: map[string]string{"active": "10.0.0.1", "standby": "10.0.0.2"}, StrictTopology: true,
		Faults: []wasi.BrokerNetworkFault{{Name: "partition", Action: "partition", From: "active", To: "standby", AtNanos: 1}, {Name: "heal", Action: "heal", From: "active", To: "standby", AtNanos: 30_000_000_000}},
	}
	if _, err := wasi.NewBroker(configuration); !errors.Is(err, netmodel.ErrUnsupported) {
		t.Fatalf("absent strict link did not retain native rejection: %v", err)
	}
	request := wasi.Request{Config: wasi.Config{Profile: wasi.CooperativeProfile, WorkingDirectory: "/workspace", Environment: []string{"PWD=/workspace", "TZ=UTC"}, WritableDirectories: []string{"/workspace", "/tmp"}, Limits: wasi.Limits{OutputBytes: 1024, TranscriptBytes: 1 << 20, Calls: 1000, Descriptors: 32, Files: 100, FilesystemBytes: 1 << 20, PendingEvents: 100}, NetworkV2: configuration.Network}, Runtime: &wasi.RuntimeControl{Seed: 7, Choice: &runnerbackend.ChoiceRequest{Mode: choice.ModeRecord, Limit: 1 << 20}}}
	identity, err := simulation.BootConfiguration(request)
	if err != nil {
		t.Fatal(err)
	}
	provider := &simulation.Provider{Broker: configuration, Boots: map[gomad3sim.BootID]wasi.Request{"active": request, "standby": request}}
	cluster := gomad3sim.Spec{Backend: gomad3sim.BackendWASM, Seed: 7, Limits: gomad3sim.DefaultLimits(), Nodes: []gomad3sim.NodeSpec{{ID: "active", Boot: "active", Address: "10.0.0.1", Config: identity}, {ID: "standby", Boot: "standby", Address: "10.0.0.2", Config: identity}}}
	if _, err := provider.Open(t.Context(), cluster); !errors.Is(err, netmodel.ErrUnsupported) {
		t.Fatalf("emitted absent topology did not fail Provider admission before helpers: %v", err)
	}
	links := temporalQualificationLinks()
	cluster.Links = links
	execution, err := provider.Open(t.Context(), cluster)
	if err != nil {
		t.Fatalf("emitted root topology failed Provider admission before helpers: %v", err)
	}
	if len(links) != 2 || links[0].From != "active" || links[0].To != "standby" || links[1].From != "standby" || links[1].To != "active" || !links[0].Enabled || !links[1].Enabled {
		t.Fatal("Temporal root lost explicit bidirectional connected topology")
	}
	defer func() {
		if err := execution.Shutdown(t.Context()); err != nil {
			t.Error(err)
		}
	}()
	// The startup partition has already exercised the actual native model.
	// Reconnect both directed routes, then partition and heal each explicitly.
	for _, link := range links {
		for _, action := range []string{"heal", "partition", "heal"} {
			if err := execution.Link(action, link.From, link.To, 0, false); err != nil {
				t.Fatalf("%s %s->%s rejected by native model: %v", action, link.From, link.To, err)
			}
		}
	}
}

func temporalQualificationLinks() []gomad3sim.LinkSpec {
	return []gomad3sim.LinkSpec{{From: "active", To: "standby", Enabled: true}, {From: "standby", To: "active", Enabled: true}}
}

func TestTemporalQualificationBuildCacheRemainsExplicitAndPrivate(t *testing.T) {
	root := t.TempDir()
	t.Setenv("GOMAD_WASM_TEMPORAL_BUILD_CACHE", "")
	if path, err := temporalQualificationBuildCache(root); err != nil || path != filepath.Join(root, "temporal-cluster-build-cache") {
		t.Fatalf("default cache changed: %s %v", path, err)
	}
	shared := filepath.Join(t.TempDir(), "verified-build-cache")
	t.Setenv("GOMAD_WASM_TEMPORAL_BUILD_CACHE", shared)
	if path, err := temporalQualificationBuildCache(root); err != nil || path != shared {
		t.Fatalf("explicit shared verified cache rejected: %s %v", path, err)
	}
	for _, invalid := range []string{"relative/cache", string(filepath.Separator), shared + "/../redirected"} {
		t.Setenv("GOMAD_WASM_TEMPORAL_BUILD_CACHE", invalid)
		if _, err := temporalQualificationBuildCache(root); err == nil {
			t.Fatalf("unsafe compiler cache accepted: %s", invalid)
		}
	}
}

func temporalQualificationBuildCache(root string) (string, error) {
	cache := os.Getenv("GOMAD_WASM_TEMPORAL_BUILD_CACHE")
	if cache == "" {
		cache = filepath.Join(root, "temporal-cluster-build-cache")
	}
	if !filepath.IsAbs(cache) || filepath.Clean(cache) != cache || cache == string(filepath.Separator) {
		return "", errors.New("explicit absolute non-root verified compiler cache required")
	}
	return cache, nil
}

func TestTemporalQualificationBindsPeerHistoryWithoutBorrowingOldExecution(t *testing.T) {
	for _, pair := range []struct{ role, peer string }{{"active", "10.0.0.2:7234"}, {"standby", "10.0.0.1:7234"}} {
		request := wasi.Request{Config: wasi.Config{Environment: []string{"TZ=UTC", "GOMAD_TEMPORAL_REMOTE_HISTORY=stale"}}}
		configured, err := temporalQualificationRequest(request, pair.role)
		if err != nil {
			t.Fatal(err)
		}
		if configured.Config.Environment[1] != "GOMAD_TEMPORAL_REMOTE_HISTORY="+pair.peer || request.Config.Environment[1] != "GOMAD_TEMPORAL_REMOTE_HISTORY=stale" {
			t.Fatal("peer HistoryService readiness binding absent or aliases input")
		}
	}
	if _, err := temporalQualificationRequest(wasi.Request{}, "foreign"); err == nil {
		t.Fatal("unknown Temporal role accepted")
	}
}

func temporalQualificationRequest(request wasi.Request, role string) (wasi.Request, error) {
	peer := ""
	switch role {
	case "active":
		peer = "10.0.0.2:7234"
	case "standby":
		peer = "10.0.0.1:7234"
	default:
		return wasi.Request{}, fmt.Errorf("unknown Temporal qualification role %q", role)
	}
	request.Config.Environment = append([]string{}, request.Config.Environment...)
	found := false
	for index, entry := range request.Config.Environment {
		if strings.HasPrefix(entry, "GOMAD_TEMPORAL_REMOTE_HISTORY=") {
			request.Config.Environment[index] = "GOMAD_TEMPORAL_REMOTE_HISTORY=" + peer
			found = true
		}
	}
	if !found {
		request.Config.Environment = append(request.Config.Environment, "GOMAD_TEMPORAL_REMOTE_HISTORY="+peer)
	}
	return request, nil
}

// The existing producer supplies detached configuration and staged unchanged
// sources only. Its historical module, choice identity and costs are NOT adopted.
func TestActualTemporalClusterQualification(t *testing.T) {
	start := hostexec.CurrentProcessResourceUsage()
	qualificationIntent(t)
	observer, err := temporalHostMemoryObserver(os.Getenv("GOMAD_WASM_TEMPORAL_MEMORY_STAGES"), func(line string) { t.Log(line) })
	if err != nil {
		t.Fatal(err)
	}
	if observer != nil {
		if os.Getenv("GOMAD_WASM_INTEGRATED_SOURCE_MANIFEST") != "" {
			t.Fatal("host memory diagnostics are provisional nonacceptance, not final corpus evidence")
		}
		t.Log("explicit host memory diagnostic: sampled heap and lifetime self RSS; no forced GC; all timing nonacceptance")
	}
	inputs := os.Getenv("GOMAD_WASM_TEMPORAL_QUALIFICATION_INPUTS")
	root := os.Getenv("GOMAD_WASM_QUALIFICATION_ROOT")
	if !filepath.IsAbs(inputs) || !filepath.IsAbs(root) {
		t.Fatal("actual Temporal cluster requires absolute detached configuration and evidence roots")
	}
	checkout, err := filepath.Abs("../../../../../..")
	if err != nil {
		t.Fatal(err)
	}
	staged := os.Getenv("GOMAD_WASM_TEMPORAL_PREPARATION_SOURCE")
	if staged == "" {
		t.Fatal("explicit existing staged Temporal source required")
	}
	for _, file := range []struct{ name, source string }{{"recovery_test.go", "temporal_storage"}, {"replication_test.go", "temporal_broker"}} {
		actual, err := hostfs.ReadBounded(filepath.Join(checkout, staged, file.name), 1<<20)
		if err != nil {
			t.Fatal(err)
		}
		expected, err := hostfs.ReadBounded(filepath.Join(checkout, "tools/gomad3/internal/backends/wasm/testdata", file.source, file.name), 1<<20)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(actual, expected) {
			t.Fatal("staged Temporal source differs from unchanged fixture")
		}
	}
	requests := map[string]wasi.Request{}
	for _, role := range []string{"active", "standby"} {
		data, err := hostfs.ReadBounded(filepath.Join(inputs, role+"-cluster-commit.request.json"), 16<<20)
		if err != nil {
			t.Fatal(err)
		}
		var request wasi.Request
		if err := strictDecode(data, &request); err != nil {
			t.Fatal(err)
		}
		if request.Broker != nil || request.StorageImage != nil || request.Config.Profile != wasi.CooperativeProfile || request.Config.Storage == nil {
			t.Fatal("Temporal boot input must be fresh cooperative durable configuration")
		}
		request, err = temporalQualificationRequest(request, role)
		if err != nil {
			t.Fatal(err)
		}
		requests[role] = request
	}
	brokerBytes, err := hostfs.ReadBounded(filepath.Join(inputs, "two-cluster-broker-config.json"), 16<<20)
	if err != nil {
		t.Fatal(err)
	}
	var broker wasi.BrokerConfig
	if err := strictDecode(brokerBytes, &broker); err != nil {
		t.Fatal(err)
	}
	broker.Nodes = nil
	broker.Addresses = nil
	broker.Diagnostics = false
	broker.WorldProfile = world.CompressedSnapshotSchema
	broker.Limits = world.CompressedSnapshotProfile().Limits
	compiler := os.Getenv("GOMAD3_STOCK_GO")
	compilerBytes, err := hostfs.ReadBounded(compiler, 512<<20)
	if err != nil {
		t.Fatal(err)
	}
	helper, err := filepath.Abs("../wasmhost/target/release/gomad3-wasmhost")
	if err != nil {
		t.Fatal(err)
	}
	helperBytes, err := hostfs.ReadBounded(helper, 512<<20)
	if err != nil {
		t.Fatal(err)
	}
	runtimeRoot, err := filepath.Abs("../../../model/runtimesource")
	if err != nil {
		t.Fatal(err)
	}
	compileCache := filepath.Join(root, "temporal-cluster-engine-cache")
	if err := os.MkdirAll(compileCache, 0700); err != nil {
		t.Fatal(err)
	}
	preparationRoot, err := os.MkdirTemp(root, "temporal-cluster-prepared-")
	if err != nil {
		t.Fatal(err)
	}
	initial := requests["active"]
	buildCache, err := temporalQualificationBuildCache(root)
	if err != nil {
		t.Fatal(err)
	}
	options := backend.Options{CompilerPath: compiler, CompilerSHA256: string(record.HashBytes(compilerBytes)), GoVersion: "go1.27.1", HelperPath: helper, HelperSHA256: string(record.HashBytes(helperBytes)), ModelSHA256: wasi.ImplementationSHA256(), CacheRoot: buildCache, CompileCacheRoot: compileCache, RuntimeRoot: runtimeRoot, Engine: wasi.PinnedEngine(), MemoryBytes: initial.MemoryBytes, Fuel: initial.Fuel, Config: initial.Config}
	integrated, err := record.ParseSHA256(os.Getenv("GOMAD_WASM_INTEGRATED_SOURCE_SHA256"))
	if err != nil {
		t.Fatal(err)
	}
	manifestPath := os.Getenv("GOMAD_WASM_INTEGRATED_SOURCE_MANIFEST")
	frozen, err := Freeze(t.Context(), Spec{OperatorStart: start, Manifest: Manifest{Cell: os.Getenv("GOMAD_WASM_QUALIFICATION_CELL"), Provisional: manifestPath == "", IntegratedSourceManifestPath: manifestPath, IntegratedSourceSHA256: integrated, ID: "independent-temporal-replication-durable-restart", Workload: "multi-instance", Seed: 7, Repeat: 2, Unsupported: []string{"controlled Temporal profile not qualified", "checkpoint", "uncontrolled host WASI", "native without current retained proof"}}, Options: options, Campaign: runner.CampaignSpec{Target: target.Spec{Backend: backend.Name, Kind: target.KindGoTest, Source: staged, WorkingDir: checkout, PreparationRoot: preparationRoot, BuildTags: []string{"test_dep", "disable_grpc_modules", "sqlite3_dotlk"}}, RunnerBuild: "fn156-qualification"}, ArtifactRoot: root, Command: []string{"wasm-qualify", "multi-instance", os.Getenv("GOMAD_WASM_QUALIFICATION_CELL")}})
	if err != nil {
		t.Fatal(err)
	}
	frozen.observeHostMemoryStage = observer
	prepared, err := PreparedTarget(frozen)
	if err != nil {
		t.Fatal(err)
	}
	producerPath, err := writePreparationProducer(root, "temporal-cluster-producer.json", frozen, prepared)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("genuine prepared module=%s sha=%s preparation=%s producer=%s", prepared.Path, prepared.SHA256, time.Duration(frozen.coldPreparationWallNanos), producerPath)
	module, err := hostfs.ReadBounded(prepared.Path, 512<<20)
	if err != nil {
		t.Fatal(err)
	}
	imports, pages, err := backend.InspectModule(module)
	if err != nil {
		t.Fatal(err)
	}
	implementation, err := choice.ImplementationIdentity(prepared.BuildKey)
	if err != nil {
		t.Fatal(err)
	}
	targetHash, err := record.SHA256(prepared.SHA256).Bytes()
	if err != nil {
		t.Fatal(err)
	}
	provider := &simulation.Provider{Boots: map[gomad3sim.BootID]wasi.Request{}, Broker: broker}
	cluster := gomad3sim.Spec{Schema: gomad3sim.SpecSchema, Backend: gomad3sim.BackendWASM, Fidelity: gomad3sim.FidelityHardIsolation, Seed: 7, Limits: gomad3sim.DefaultLimits(), Links: temporalQualificationLinks()}
	for index, role := range []string{"active", "standby"} {
		request := requests[role]
		request.ModulePath, request.ModuleSHA256 = prepared.Path, strings.TrimPrefix(prepared.SHA256, "sha256:")
		request.HelperPath, request.HelperSHA256 = helper, strings.TrimPrefix(options.HelperSHA256, "sha256:")
		request.Engine, request.Imports, request.InitialMemoryPages = options.Engine, imports, pages
		request.CompileCache, request.Timeout, request.StdoutMirror = compileCache, 15*time.Minute, nil
		request.Runtime = &wasi.RuntimeControl{Seed: 7, Choice: &runnerbackend.ChoiceRequest{Mode: choice.ModeRecord, Limit: 64 << 20, ExecutionIdentity: choice.ExecutionIdentity{TargetSHA256: targetHash, ToolchainBuildKey: prepared.BuildKey, GOOS: "wasip1", GOARCH: "wasm", ImplementationSHA256: implementation}}}
		configuration, err := simulation.BootConfiguration(request)
		if err != nil {
			t.Fatal(err)
		}
		provider.Boots[gomad3sim.BootID(role+"-commit")] = request
		node := gomad3sim.NodeSpec{ID: gomad3sim.NodeID(role), Boot: gomad3sim.BootID(role + "-commit"), Address: []string{"10.0.0.1", "10.0.0.2"}[index], Config: configuration}
		for _, mount := range request.Config.Storage.Volumes {
			cluster.Volumes = append(cluster.Volumes, gomad3sim.VolumeSpec{ID: gomad3sim.VolumeID(mount.ID), CapacityBytes: mount.CapacityBytes})
			node.Volumes = append(node.Volumes, gomad3sim.VolumeMount{Volume: gomad3sim.VolumeID(mount.ID), Path: mount.Path})
		}
		cluster.Nodes = append(cluster.Nodes, node)
		request.Config.Environment = append([]string{}, request.Config.Environment...)
		phaseFound := false
		for i, entry := range request.Config.Environment {
			if entry == "GOMAD_TEMPORAL_PHASE=commit" {
				request.Config.Environment[i] = "GOMAD_TEMPORAL_PHASE=reopen"
				phaseFound = true
			}
		}
		if !phaseFound {
			t.Fatal("Temporal phase input is not sealed")
		}
		provider.Boots[gomad3sim.BootID(role+"-reopen")] = request
		provider.IncarnationBoots = append(provider.IncarnationBoots, simulation.IncarnationBoot{Handle: gomad3sim.NodeHandle{Node: gomad3sim.NodeID(role), Incarnation: 2}, Boot: gomad3sim.BootID(role + "-reopen")})
	}
	active, standby := gomad3sim.NodeHandle{Node: "active", Incarnation: 1}, gomad3sim.NodeHandle{Node: "standby", Incarnation: 1}
	activeReopen, standbyReopen := gomad3sim.NodeHandle{Node: "active", Incarnation: 2}, gomad3sim.NodeHandle{Node: "standby", Incarnation: 2}
	provider.OutputCrashGroups = []simulation.OutputCrashGroup{{Handles: []gomad3sim.NodeHandle{active, standby}, Marker: []byte("GOMAD_TEMPORAL_OBSERVATION ")}}
	expectedEvents := []string{"1", "2", "3", "4", "5"}
	canary := os.Getenv("GOMAD_WASM_QUALIFICATION_CELL") == "canary"
	if canary {
		expectedEvents = append(expectedEvents, "unexpected-canary-event")
	}
	provider.Oracles = &simulation.OraclePlan{Schema: simulation.HistoryObservationSchema, Prefix: "GOMAD_TEMPORAL_OBSERVATION ", MaximumBytes: 1 << 20, Histories: []simulation.HistoryComparison{{Name: "active-durable-recovery", Expected: active, Actual: activeReopen, ExpectedEvents: expectedEvents}, {Name: "standby-durable-recovery", Expected: standby, Actual: standbyReopen, ExpectedEvents: expectedEvents}}, Convergence: []simulation.HistoryConvergence{{Name: "independent-replication-convergence", Handles: []gomad3sim.NodeHandle{activeReopen, standbyReopen}}}}
	scenario := func(ctx context.Context, c gomad3sim.Cluster) error {
		for _, role := range []gomad3sim.NodeID{"active", "standby"} {
			if _, err := c.Start(ctx, role); err != nil {
				return err
			}
		}
		for _, handle := range []gomad3sim.NodeHandle{active, standby} {
			if _, err := c.Wait(ctx, handle); err != nil {
				return err
			}
		}
		for _, role := range []gomad3sim.NodeID{"active", "standby"} {
			if _, err := c.Restart(ctx, role); err != nil {
				return err
			}
		}
		for _, handle := range []gomad3sim.NodeHandle{activeReopen, standbyReopen} {
			if _, err := c.Wait(ctx, handle); err != nil {
				return err
			}
		}
		return provider.RecordOracles(ctx, c)
	}
	result, admission, err := temporalRunAdmittedCluster(t, t.Context(), frozen, cluster, scenario, provider, runtime.GOOS, "/proc/meminfo", os.Getenv("GOMAD_WASM_TEMPORAL_MIN_AVAILABLE_BYTES"))
	if err != nil && result.Manifest.Schema == "" {
		// Genuine preparation is already retained. A safety rejection dispatches
		// nothing and must not invent a runtime Result or terminal receipt.
		t.Fatalf("helper dispatch refused after genuine preparation=%s available-bytes=%d minimum-bytes=%d cause=%v", producerPath, admission.AvailableBytes, admission.MinimumBytes, err)
	}
	if err != nil {
		// Preserve the existing returned Result without creating another report,
		// copying World/traces or asserting unavailable replay/qualification.
		result.OperatorExecutionAfter = hostexec.CurrentProcessResourceUsage()
		encoded, encodeErr := json.MarshalIndent(result, "", "  ")
		failedPath := filepath.Join(root, "temporal-cluster-failed-result.json")
		if encodeErr == nil && len(encoded) < 32<<20 {
			file, writeErr := os.OpenFile(failedPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
			if writeErr == nil {
				_, writeErr = file.Write(append(encoded, '\n'))
				closeErr := file.Close()
				t.Logf("raw nonaccepted returned Result=%s write-error=%v close-error=%v", failedPath, writeErr, closeErr)
			} else {
				t.Logf("raw nonaccepted returned Result publication failed: %v", writeErr)
			}
		} else {
			t.Logf("raw returned Result encoding failed or exceeded 32MiB: %v", encodeErr)
		}
		message := err.Error()
		if len(message) > 4096 {
			message = message[:4096]
		}
		t.Fatalf("cluster Go error (nonaccepted); retained preparation=%s failure-report=%s actual-self=%+v cause=%s", producerPath, result.QualificationPath, result.OperatorExecutionAfter, message)
	}
	path, _, err := writeResultObserved(root, result, observer)
	if err != nil {
		t.Fatal(err)
	}
	if !result.Qualification.Deterministic || result.Qualification.TargetSuccess == canary || result.Cost == nil || result.Budget == nil || !result.Budget.WithinBudget {
		t.Fatalf("Temporal cluster gate failed; retained=%s", path)
	}
	t.Logf("actual Temporal canary=%t module=%s sha=%s result=%s cpu=%d", canary, prepared.Path, prepared.SHA256, path, result.Cost.CPUNanos)
}
