package qualification

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	runnerbackend "go.temporal.io/server/tools/gomad/internal/backends"
	"go.temporal.io/server/tools/gomad/internal/backends/target"
	"go.temporal.io/server/tools/gomad/internal/backends/wasm/backend"
	simulation "go.temporal.io/server/tools/gomad/internal/backends/wasm/cluster"
	"go.temporal.io/server/tools/gomad/internal/backends/wasm/installation"
	"go.temporal.io/server/tools/gomad/internal/backends/wasm/wasi"
	"go.temporal.io/server/tools/gomad/internal/evidence/artifact"
	"go.temporal.io/server/tools/gomad/internal/evidence/record"
	hostexec "go.temporal.io/server/tools/gomad/internal/host/exec"
	"go.temporal.io/server/tools/gomad/internal/model/choice"
	gomad3sim "go.temporal.io/server/tools/gomad/internal/model/cluster"
	"go.temporal.io/server/tools/gomad/internal/model/volume"
	"go.temporal.io/server/tools/gomad/internal/runner"
)

func TestClusterQualificationRejectsUnboundExecutable(t *testing.T) {
	_, err := RunCluster(context.Background(), Spec{}, gomad3sim.Spec{}, func(context.Context, gomad3sim.Cluster) error { return nil }, &simulation.Provider{})
	if err == nil {
		t.Fatal("unbound cluster executable accepted")
	}
}

func TestClusterOutcomeNeverTurnsCapacityIntoApplicationFailure(t *testing.T) {
	for _, termination := range []string{"capacity", "unsupported", "invalid", "infrastructure", "divergence", "timeout", "cancelled", ""} {
		_, err := clusterOutcome(gomad3sim.Result{Outcome: gomad3sim.OutcomeCompleted}, []simulation.IncarnationSummary{{Termination: termination, Reaped: true}})
		if err == nil {
			t.Errorf("accepted %q as a retained application outcome", termination)
		}
	}
	zero, nonzero := uint32(0), uint32(17)
	for _, tc := range []struct {
		termination string
		code        *uint32
		domain      string
	}{{"exit", &zero, "success"}, {"exit", &nonzero, "target"}, {"trap", nil, "target"}, {"crash", nil, "success"}, {"storage_crash", nil, "success"}, {"stop", nil, "success"}} {
		outcome, err := clusterOutcome(gomad3sim.Result{Outcome: gomad3sim.OutcomeCompleted}, []simulation.IncarnationSummary{{Termination: tc.termination, ExitCode: tc.code, Reaped: true}})
		if err != nil || outcome.Domain != tc.domain {
			t.Errorf("%q: %+v, %v", tc.termination, outcome, err)
		}
	}
}

func TestIncarnationSamplesUseDetachedCountersAndActualOSFallback(t *testing.T) {
	incarnations := []simulation.IncarnationSummary{{Handle: gomad3sim.NodeHandle{Node: "a", Incarnation: 2}, WallElapsedNanos: 41, FuelRemaining: 7, PeakMemoryBytes: 11, VirtualTimeElapsedNanos: 13, AuthorizationToTerminalNanos: 17, WASITraceBytes: 2, ChoiceTraceBytes: 3, DiagnosticTraceBytes: 5, StorageTraceBytes: 7, NetworkTraceBytes: 11, Telemetry: &wasi.Telemetry{PrepareNanos: 19, CompileNanos: 23, CacheLoadNanos: 29, CacheWriteNanos: 31, CacheHit: true, InstantiateNanos: 37, ExecuteNanos: 43, ResourceUsageAvailable: true, ProcessUserNanos: 47, ProcessSystemNanos: 53, PeakRSSBytes: 59}, HelperResourceUsage: &hostexec.ResourceUsage{UserNanos: 61, SystemNanos: 67, PeakRSSBytes: 71}}}
	got := incarnationSamples(incarnations, "replay", 2, "", 100)[0]
	if got.Incarnation != "a/2" || got.TraceBytes != 28 || got.FuelUsed != 93 || got.CPUNanos != 128 || got.HostPeakRSSBytes != 59 || got.GuestPeakMemoryBytes != 11 || got.VirtualTimeNanos != 13 || got.HelperWallNanos != 41 || got.AuthorizationToTerminalNanos != 17 || got.EnginePrepareNanos != 19 || got.EngineCompileNanos != 23 || got.CacheLoadNanos != 29 || got.CacheWriteNanos != 31 || !got.CompileCacheHit || got.InstantiationNanos != 37 || got.ExecutionWallNanos != 43 {
		t.Fatalf("detached counters changed real sample accounting: %+v", got)
	}
	// Helper image high-water (59) wins over Linux wait4 maxrss (71), which
	// folds in the operator's peak at execve; wait4 stays the fallback.
	incarnations[0].Telemetry = nil
	got = incarnationSamples(incarnations, "execute", 1, "", 100)[0]
	if got.CPUNanos != 128 || got.HostPeakRSSBytes != 71 || got.FuelUsed != 0 || got.EngineCompileNanos != 0 || got.ExecutionWallNanos != 0 {
		t.Fatalf("interrupted helper fabricated unavailable phase metrics: %+v", got)
	}
	if _, err := clusterOutcome(gomad3sim.Result{Outcome: gomad3sim.OutcomeCompleted}, []simulation.IncarnationSummary{{Termination: "exit", Reaped: false}}); err == nil {
		t.Fatal("unreaped summary accepted")
	}
}

func TestClusterSamplesSeparateSharedProcessFromEveryIncarnation(t *testing.T) {
	incarnations := []simulation.IncarnationSummary{}
	for _, handle := range []gomad3sim.NodeHandle{{Node: "a", Incarnation: 1}, {Node: "a", Incarnation: 2}, {Node: "b", Incarnation: 1}} {
		incarnations = append(incarnations, simulation.IncarnationSummary{Handle: handle, MemoryLimitBytes: 64, PeakMemoryBytes: 32, FuelRemaining: 7, AuthorizationToTerminalNanos: 11, Telemetry: &wasi.Telemetry{ExecuteNanos: 5, InstantiateNanos: 3}})
	}
	shared := &wasi.ClusterResult{WallElapsedNanos: 101, AuthorizationToTerminalNanos: 79, Metrics: &wasi.Telemetry{CompileNanos: 17, CacheHit: true, PeakRSSBytes: 7 << 30, ResourceUsageAvailable: true, ProcessUserNanos: 13, ProcessSystemNanos: 19}, HelperResourceUsage: &hostexec.ResourceUsage{UserNanos: 23, SystemNanos: 29, PeakRSSBytes: 8 << 30}}
	samples := clusterSamples(incarnations, shared, "execute", 2, record.HashBytes(nil), 100)
	if len(samples) != 4 {
		t.Fatalf("lost an incarnation or shared sample: %+v", samples)
	}
	for _, sample := range samples[:3] {
		if sample.ClusterHelper || sample.GuestMemoryLimitBytes != 64 || sample.GuestPeakMemoryBytes != 32 || sample.CPUNanos != 0 || sample.HostPeakRSSBytes != 0 || sample.ExecutionWallNanos != 5 {
			t.Fatalf("store sample duplicates process accounting: %+v", sample)
		}
	}
	helper := samples[3]
	if !helper.ClusterHelper || !slices.Equal(helper.ResidentInstances, []string{"a", "b"}) || helper.HostPeakRSSBytes != 7<<30 || helper.CPUNanos != 52 || helper.HelperWallNanos != 101 || helper.AuthorizationToTerminalNanos != 79 || !helper.CompileCacheHit || helper.GuestPeakMemoryBytes != 0 {
		t.Fatalf("shared helper sample: %+v", helper)
	}
	cost, err := Summarize(samples)
	if err != nil || cost.Executions != 3 || cost.CPUNanos != 52 || cost.EngineCompileNanos != 17 || cost.CompileCacheHits != 1 {
		t.Fatalf("cluster costs: %+v %v", cost, err)
	}
	var repeated []Sample
	for iteration := uint64(1); iteration <= 2; iteration++ {
		for _, phase := range []string{"execute", "replay"} {
			repeated = append(repeated, clusterSamples(incarnations, shared, phase, iteration, record.HashBytes(nil), 100)...)
		}
	}
	cost, err = Summarize(repeated)
	if err != nil || cost.Executions != 6 || cost.Replays != 6 || cost.CPUNanos != 208 || cost.EngineCompileNanos != 34 || cost.CompileCacheHits != 2 {
		t.Fatalf("separate repeated recording/replay sessions rejected or miscounted: %+v %v", cost, err)
	}
}

func clusterQualificationServiceTarget(directory, preparationRoot string) target.Spec {
	return target.Spec{Backend: backend.Name, Kind: target.KindGoRun, Source: "./internal/backends/wasm/cluster/testdata/service", WorkingDir: directory, PreparationRoot: preparationRoot}
}

func TestClusterQualificationServicePreparation(t *testing.T) {
	compiler := os.Getenv("GOMAD3_STOCK_GO")
	if compiler == "" {
		t.Skip("explicit stock compiler is required for source preparation")
	}
	helper, err := filepath.Abs("../wasmhost/target/release/gomad3-wasmhost")
	if err != nil {
		t.Fatal(err)
	}
	digest := func(path string) string {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		return string(record.HashBytes(data))
	}
	provider, err := backend.New(backend.Options{
		CompilerPath: compiler, CompilerSHA256: digest(compiler), GoVersion: "go1.27.1",
		HelperPath: helper, HelperSHA256: digest(helper), ModelSHA256: wasi.ImplementationSHA256(),
		CacheRoot: t.TempDir(), Engine: wasi.PinnedEngine(), MemoryBytes: 256 << 20, Fuel: 4_000_000_000,
		Config: installation.DefaultConfig(wasi.StockProfile),
	})
	if err != nil {
		t.Fatal(err)
	}
	directory, err := filepath.Abs("../../../..")
	if err != nil {
		t.Fatal(err)
	}
	spec := clusterQualificationServiceTarget(directory, t.TempDir())
	prepared, err := provider.Prepare(t.Context(), spec)
	if err != nil {
		t.Fatal(err)
	}
	module, err := os.ReadFile(prepared.Path)
	if err != nil {
		t.Fatal(err)
	}
	imports, pages, err := backend.InspectModule(module)
	if err != nil || len(imports) == 0 || pages == 0 {
		t.Fatalf("prepared service is not a runnable WASI module: imports=%d pages=%d err=%v", len(imports), pages, err)
	}
	if err := provider.ValidatePrepared(spec, prepared, nil); err != nil {
		t.Fatal(err)
	}
}

func TestActualClusterQualificationAndDetachedReplay(t *testing.T) {
	start := hostexec.CurrentProcessResourceUsage()
	qualificationIntent(t)
	compiler := os.Getenv("GOMAD3_STOCK_GO")
	integrated, err := record.ParseSHA256(os.Getenv("GOMAD_WASM_INTEGRATED_SOURCE_SHA256"))
	if err != nil {
		t.Fatal(err)
	}
	helper, err := filepath.Abs("../wasmhost/target/release/gomad3-wasmhost")
	if err != nil {
		t.Fatal(err)
	}
	helperBytes, err := os.ReadFile(helper)
	if err != nil {
		t.Fatal(err)
	}
	helper = filepath.Join(t.TempDir(), "helper")
	if err = os.WriteFile(helper, helperBytes, 0700); err != nil {
		t.Fatal(err)
	}
	compilerBytes, err := os.ReadFile(compiler)
	if err != nil {
		t.Fatal(err)
	}
	runtimeRoot, err := filepath.Abs("../../../model/runtimesource")
	if err != nil {
		t.Fatal(err)
	}
	directory, err := filepath.Abs("../../../..")
	if err != nil {
		t.Fatal(err)
	}
	config := wasi.Config{Profile: wasi.CooperativeProfile, Environment: []string{"PWD=/workspace", "TZ=UTC"}, WorkingDirectory: "/workspace", WritableDirectories: []string{"/workspace", "/tmp"}, Clock: wasi.ClockPolicy{EpochNanos: 946684800000000000}, Limits: wasi.Limits{OutputBytes: 1 << 20, TranscriptBytes: 8 << 20, Calls: 10000, Descriptors: 64, Files: 1000, FilesystemBytes: 8 << 20, PendingEvents: 4096}}
	options := backend.Options{CompilerPath: compiler, CompilerSHA256: string(record.HashBytes(compilerBytes)), GoVersion: "go1.27.1", HelperPath: helper, HelperSHA256: string(record.HashBytes(helperBytes)), ModelSHA256: wasi.ImplementationSHA256(), CacheRoot: t.TempDir(), CompileCacheRoot: t.TempDir(), RuntimeRoot: runtimeRoot, Engine: wasi.PinnedEngine(), MemoryBytes: 256 << 20, Fuel: 4_000_000_000, Config: config}
	if err = os.Chmod(options.CompileCacheRoot, 0700); err != nil {
		t.Fatal(err)
	}
	root := os.Getenv("GOMAD_WASM_QUALIFICATION_ROOT")
	if root == "" {
		root = t.TempDir()
	}
	frozen, err := Freeze(t.Context(), Spec{OperatorStart: start, Manifest: Manifest{Cell: os.Getenv("GOMAD_WASM_QUALIFICATION_CELL"), Provisional: os.Getenv("GOMAD_WASM_INTEGRATED_SOURCE_MANIFEST") == "", IntegratedSourceManifestPath: os.Getenv("GOMAD_WASM_INTEGRATED_SOURCE_MANIFEST"), ID: "independent-guest-lifecycle", Workload: "fixture", IntegratedSourceSHA256: integrated, Seed: 7, Repeat: 2}, Options: options, Campaign: runner.CampaignSpec{Target: clusterQualificationServiceTarget(directory, t.TempDir()), RunnerBuild: "fn156-qualification"}, ArtifactRoot: root, Command: []string{"wasm-qualify", "independent-guest-lifecycle"}})
	if err != nil {
		t.Fatal(err)
	}
	module, err := os.ReadFile(frozen.preparedTarget.Path)
	if err != nil {
		t.Fatal(err)
	}
	imports, pages, err := backend.InspectModule(module)
	if err != nil {
		t.Fatal(err)
	}
	implementation, err := choice.ImplementationIdentity(frozen.preparedTarget.BuildKey)
	if err != nil {
		t.Fatal(err)
	}
	targetHash, err := record.SHA256(frozen.preparedTarget.SHA256).Bytes()
	if err != nil {
		t.Fatal(err)
	}
	request := wasi.Request{HelperPath: helper, HelperSHA256: strings.TrimPrefix(options.HelperSHA256, "sha256:"), ModulePath: frozen.preparedTarget.Path, ModuleSHA256: strings.TrimPrefix(frozen.preparedTarget.SHA256, "sha256:"), Engine: options.Engine, Imports: imports, InitialMemoryPages: pages, MemoryBytes: options.MemoryBytes, Fuel: options.Fuel, Config: config, Timeout: time.Minute, CompileCache: options.CompileCacheRoot, Runtime: &wasi.RuntimeControl{Seed: 7, Choice: &runnerbackend.ChoiceRequest{Mode: choice.ModeRecord, Limit: 1 << 20, ExecutionIdentity: choice.ExecutionIdentity{TargetSHA256: targetHash, ToolchainBuildKey: frozen.preparedTarget.BuildKey, GOOS: "wasip1", GOARCH: "wasm", ImplementationSHA256: implementation}}}}
	for _, canary := range []bool{false, true} {
		if !qualificationCell(t, canary) {
			continue
		}
		request.Config.Args = []string{"service", "park"}
		if canary {
			request.Config.Args = []string{"service", "canary"}
		}
		identity, err := simulation.BootConfiguration(request)
		if err != nil {
			t.Fatal(err)
		}
		clusterSpec := gomad3sim.Spec{Schema: gomad3sim.SpecSchema, Backend: gomad3sim.BackendWASM, Fidelity: gomad3sim.FidelitySimulationModel, Seed: 7, Limits: gomad3sim.DefaultLimits(), Nodes: []gomad3sim.NodeSpec{{ID: "a", Boot: "service", Address: "10.0.0.1", Config: identity}, {ID: "b", Boot: "service", Address: "10.0.0.2", Config: identity}}}
		provider := &simulation.Provider{Boots: map[gomad3sim.BootID]wasi.Request{"service": request}}
		scenario := func(ctx context.Context, c gomad3sim.Cluster) error {
			for round := 0; round < 2; round++ {
				var a, b gomad3sim.NodeHandle
				var err error
				if round == 0 {
					a, err = c.Start(ctx, "a")
				} else {
					a, err = c.Restart(ctx, "a")
				}
				if err != nil {
					return err
				}
				if round == 0 {
					b, err = c.Start(ctx, "b")
				} else {
					b, err = c.Restart(ctx, "b")
				}
				if err != nil {
					return err
				}
				if _, err = c.Wait(ctx, a); err != nil {
					return err
				}
				if _, err = c.Wait(ctx, b); err != nil {
					return err
				}
			}
			return nil
		}
		result, err := RunCluster(t.Context(), frozen, clusterSpec, scenario, provider)
		if err != nil {
			t.Fatal(err)
		}
		if !result.Qualification.Deterministic || result.Qualification.TargetSuccess == canary || result.Cost == nil || result.Cost.ReplayableBugs != uint64(boolInt(canary)) || result.Budget == nil || !result.Budget.WithinBudget {
			t.Fatalf("actual cluster qualification: %+v", result)
		}
		path, err := WriteResult(root, result)
		if err != nil {
			t.Fatal(err)
		}
		if _, err = OpenResult(path); err != nil {
			t.Fatal(err)
		}
		t.Logf("canary=%t result=%s guest-executions=%d replays=%d cpu=%d budget=%+v", canary, path, result.Cost.Executions, result.Cost.Replays, result.Cost.CPUNanos, result.Budget)
	}
}

// publishRetainedClusterArtifact publishes a minimal Artifact whose evidence
// names schema. Its payload is never a valid retained bundle.
func publishRetainedClusterArtifact(t *testing.T, schema string) string {
	t.Helper()
	module := []byte("\x00asm\x01\x00\x00\x00")
	modulePath := filepath.Join(t.TempDir(), "target")
	if err := os.WriteFile(modulePath, module, 0o700); err != nil {
		t.Fatal(err)
	}
	evidence := []byte(`{"schema":"` + schema + `"}`)
	provenance := []byte(`{}`)
	metadata := &record.BackendMetadata{Name: backend.Name, ReplayMode: record.ReplayExact, Provenance: record.BackendPayload{Schema: "provenance", File: "backend/provenance.json", SHA256: record.HashBytes(provenance), Bytes: record.Uint64String(len(provenance))}, Evidence: &record.BackendPayload{Schema: schema, File: "backend/simulation.json", SHA256: record.HashBytes(evidence), Bytes: record.Uint64String(len(evidence))}}
	world, worldPayloads := record.NoneWorld()
	exitCode := record.Uint64String(1)
	manifest := record.ExecutionRecord{
		SchemaVersion: record.SchemaVersion, ArtifactKind: record.ArtifactTargetFailure, CreatedAt: "2026-10-12T12:00:00Z", CampaignID: "retained", Seed: 7, ReplayMode: record.ReplayExact,
		Runner:    record.Runner{RecordContract: record.RecordContract, RunnerBuild: "test", HostOS: "linux", HostArch: "amd64"},
		Toolchain: record.Toolchain{GoVersion: "go1.26.4", BuildKey: strings.Repeat("a", 64), TargetGOOS: "wasip1", TargetGOARCH: "wasm"},
		Target: record.Target{Kind: "go-run", Source: ".", SHA256: record.HashBytes(module), Size: record.Uint64String(len(module)), Argv: []string{"gomad3-target"}, BuildTags: []string{},
			Adapters: []record.TargetAdapter{}, Compatibility: []record.CompatibilityPack{}, BuildInfo: record.BuildInfo{GoVersion: "go1.26.4", Path: "example.com/target"}, Backend: metadata},
		IOProfile:   record.BackendIOProfile(*metadata),
		Environment: []record.Environment{{Name: "GOMAD3_IO_PROFILE", Value: backend.Name}, {Name: "GOMADSEED", Value: "7"}, {Name: "TZ", Value: "UTC"}},
		Limits:      record.Limits{ExecutionTimeoutNanos: 1, OverallTimeoutNanos: 2, OutputBytes: 64, WorldTransitionBytes: 64},
		World:       world,
		Outcome:     record.Outcome{Domain: "target", Reason: "nonzero_exit", Termination: "exit", ExitCode: &exitCode},
		Streams:     record.Streams{Stdout: record.Stream{FullSHA256: record.HashBytes(nil)}, Stderr: record.Stream{FullSHA256: record.HashBytes(nil)}},
		Host:        record.Host{StartedAt: "2026-10-12T12:00:00Z", FinishedAt: "2026-10-12T12:00:01Z", ElapsedNanos: 1},
	}
	published, err := artifact.PublishArtifact(artifact.Store{Root: t.TempDir()}, artifact.ArtifactInput{Manifest: manifest, TargetPath: modulePath, World: worldPayloads, BackendPayloads: []target.BackendPayload{{Reference: metadata.Provenance, Data: provenance}, {Reference: *metadata.Evidence, Data: evidence}}})
	if err != nil {
		t.Fatal(err)
	}
	return published.Path
}

func TestRetainedV1ArtifactIsRefusedBeforeItsPayloadIsRead(t *testing.T) {
	path := publishRetainedClusterArtifact(t, "gomad3.wasm-simulation-retained/v1")
	var stages []string
	_, _, err := replayClusterArtifactObserved(t.Context(), path, t.TempDir(), simulation.InstalledHelper{}, func(stage string) { stages = append(stages, stage) })
	if !errors.Is(err, simulation.ErrRetainedSchemaUnsupported) || slices.Contains(stages, "replay.read.before") {
		t.Fatalf("v1 cluster replay: stages %v, err %v", stages, err)
	}
	if err := validateSQLiteArtifactSelection(path, volume.CrashState{}, wasi.StorageCrashContext{}); !errors.Is(err, simulation.ErrRetainedSchemaUnsupported) {
		t.Fatalf("v1 selected-state Artifact: %v", err)
	}
}

func TestRetainedTargetModuleIsBoundedBeforeCopy(t *testing.T) {
	listed := record.Target{File: "target", SHA256: record.HashBytes(nil), Size: 512<<20 + 1}
	// A nil Artifact proves the bound is checked from the listed size alone.
	if _, err := retainedTargetModule(nil, listed); err == nil {
		t.Fatal("oversized retained target module admitted for copy")
	}
	listed.Size = 512 << 20
	module, err := retainedTargetModule(nil, listed)
	if err != nil || module.SHA256 != string(listed.SHA256) || module.Copy == nil {
		t.Fatalf("bounded retained target module: %+v %v", module, err)
	}
}
