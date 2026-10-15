package qualification

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	runnerbackend "go.temporal.io/server/tools/gomad/internal/backends"
	"go.temporal.io/server/tools/gomad/internal/backends/target"
	"go.temporal.io/server/tools/gomad/internal/backends/wasm/backend"
	simulation "go.temporal.io/server/tools/gomad/internal/backends/wasm/cluster"
	"go.temporal.io/server/tools/gomad/internal/backends/wasm/wasi"
	"go.temporal.io/server/tools/gomad/internal/evidence/record"
	hostexec "go.temporal.io/server/tools/gomad/internal/host/exec"
	"go.temporal.io/server/tools/gomad/internal/model/choice"
	gomad3sim "go.temporal.io/server/tools/gomad/internal/model/cluster"
	"go.temporal.io/server/tools/gomad/internal/model/volume"
	"go.temporal.io/server/tools/gomad/internal/runner"
)

// The canary really omits SQLite's sync. Expected history remains [1,2] and is
// re-evaluated against new recovered output by the native oracle owner.
func TestActualSQLiteDurableQualification(t *testing.T) {
	start := hostexec.CurrentProcessResourceUsage()
	actualSQLiteDurableQualification(t, start, false)
}

func TestActualSQLiteSelectedCrashQualification(t *testing.T) {
	start := hostexec.CurrentProcessResourceUsage()
	if os.Getenv("GOMAD_WASM_SQLITE_SELECTION_INDEX") == "" && os.Getenv("GOMAD_WASM_SQLITE_NATIVE_CATALOGUE_RESULT") == "" {
		t.Skip("explicit retained native catalogue and selected crash-state index required")
	}
	if os.Getenv("GOMAD_WASM_INTEGRATED_SOURCE_SHA256") == "" {
		t.Fatal("selected-state intent requires current integrated source binding")
	}
	actualSQLiteDurableQualification(t, start, true)
}

func actualSQLiteDurableQualification(t *testing.T, start *hostexec.ResourceUsage, selected bool) {
	t.Helper()
	qualificationIntent(t)
	root := os.Getenv("GOMAD_WASM_QUALIFICATION_ROOT")
	if !filepath.IsAbs(root) {
		t.Fatal("durable SQLite evidence root must persist and be absolute")
	}
	checkout, err := filepath.Abs("../../../../../..")
	if err != nil {
		t.Fatal(err)
	}
	compiler := os.Getenv("GOMAD3_STOCK_GO")
	compilerBytes, err := os.ReadFile(compiler)
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
	runtimeRoot, err := filepath.Abs("../../../model/runtimesource")
	if err != nil {
		t.Fatal(err)
	}
	compileCache := filepath.Join(root, "sqlite-durable-engine-cache")
	if err := os.MkdirAll(compileCache, 0700); err != nil {
		t.Fatal(err)
	}
	preparationRoot, err := os.MkdirTemp(root, "sqlite-durable-prepared-")
	if err != nil {
		t.Fatal(err)
	}
	model := &wasi.StorageModel{Volumes: []volume.VolumeConfig{{ID: "sqlite", Path: "/volume", CapacityBytes: 4 << 20}}, Limits: volume.VolumeLimits{PendingOperations: 10000, Transitions: 100000}, SQLiteLockDatabases: []string{"/volume/db"}, Faults: []wasi.StorageFault{{Name: "acknowledged-crash", Resource: "/volume/crash-trigger", Operation: "open", Occurrence: 1, Action: "crash-before"}}}
	config := wasi.Config{Profile: wasi.CooperativeProfile, Environment: []string{"PWD=/workspace", "TZ=UTC"}, WorkingDirectory: "/workspace", WritableDirectories: []string{"/workspace", "/tmp", "/volume"}, Clock: wasi.ClockPolicy{EpochNanos: 946684800000000000}, Limits: wasi.Limits{OutputBytes: 1 << 20, TranscriptBytes: 32 << 20, Calls: 100000, Descriptors: 128, Files: 1024, FilesystemBytes: 8 << 20, PendingEvents: 4096}, Storage: model}
	options := backend.Options{CompilerPath: compiler, CompilerSHA256: string(record.HashBytes(compilerBytes)), GoVersion: "go1.27.1", HelperPath: helper, HelperSHA256: string(record.HashBytes(helperBytes)), ModelSHA256: wasi.ImplementationSHA256(), CacheRoot: filepath.Join(root, "sqlite-durable-build-cache"), CompileCacheRoot: compileCache, RuntimeRoot: runtimeRoot, Engine: wasi.PinnedEngine(), MemoryBytes: 256 << 20, Fuel: 4_000_000_000, Config: config}
	integrated, err := record.ParseSHA256(os.Getenv("GOMAD_WASM_INTEGRATED_SOURCE_SHA256"))
	if err != nil {
		t.Fatal(err)
	}
	manifestPath := os.Getenv("GOMAD_WASM_INTEGRATED_SOURCE_MANIFEST")
	id := "sqlite-durable-recovery"
	var state *volume.CrashState
	var crashContext wasi.StorageCrashContext
	var primary Result
	var sourceHash record.SHA256
	if selected {
		if os.Getenv("GOMAD_WASM_QUALIFICATION_CELL") != "success" {
			t.Fatal("selected-state cells require explicit success selection")
		}
		index, parseErr := strconv.Atoi(os.Getenv("GOMAD_WASM_SQLITE_SELECTION_INDEX"))
		path := os.Getenv("GOMAD_WASM_SQLITE_NATIVE_CATALOGUE_RESULT")
		if parseErr != nil || index < 0 || index >= 32 || !filepath.IsAbs(path) {
			t.Fatal("bounded selected-state index and absolute retained native catalogue required")
		}
		primary, err = OpenResult(path)
		if err != nil {
			t.Fatal(err)
		}
		sourceHash, err = publicationResultIdentity(path)
		if err != nil {
			t.Fatal(err)
		}
		receipt, err := OpenPublicationReceipt(path)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := ValidatePublicationReceipt(path, primary, receipt); err != nil {
			t.Fatal(err)
		}
		enumeration, err := MatchedNativeCrashEnumeration(primary, gomad3sim.NodeHandle{Node: "sqlite", Incarnation: 2}, "sqlite")
		if err != nil || !primary.Qualification.Qualified || len(enumeration.States) != 4 || index >= len(enumeration.States) {
			t.Fatalf("actual reviewed four-state source catalogue unavailable: %v", err)
		}
		state = &enumeration.States[index]
		crashContext, err = MatchedNativeCrashContext(primary, gomad3sim.NodeHandle{Node: "sqlite", Incarnation: 2}, "sqlite")
		if err != nil {
			t.Fatal(err)
		}
		id = "sqlite-durable-selection-" + strings.TrimPrefix(state.Identity, "sha256:")
	}
	frozen, err := Freeze(t.Context(), Spec{OperatorStart: start, Manifest: Manifest{Cell: os.Getenv("GOMAD_WASM_QUALIFICATION_CELL"), Provisional: manifestPath == "", IntegratedSourceManifestPath: manifestPath, IntegratedSourceSHA256: integrated, ID: id, Workload: "sqlite", Seed: 7, Repeat: 2, Unsupported: []string{"controlled SQLite profile not qualified", "checkpoint", "uncontrolled host WASI", "native without current retained proof"}}, Options: options, Campaign: runner.CampaignSpec{Target: target.Spec{Backend: backend.Name, Kind: target.KindGoRun, Source: "./tools/gomad/internal/backends/wasm/testdata/storage_sqlite/main.go", WorkingDir: checkout, PreparationRoot: preparationRoot, BuildTags: []string{"sqlite3_dotlk"}}, RunnerBuild: "fn156-qualification"}, ArtifactRoot: root, Command: []string{"wasm-qualify", id, os.Getenv("GOMAD_WASM_QUALIFICATION_CELL")}})
	if err != nil {
		t.Fatal(err)
	}
	if selected {
		if err := Matched(primary.Manifest, frozen.Manifest); err != nil {
			t.Fatal(err)
		}
		if primary.Manifest.HelperSHA256 != frozen.Manifest.HelperSHA256 || primary.Manifest.CompilerSHA256 != frozen.Manifest.CompilerSHA256 || primary.Manifest.ModelSHA256 != frozen.Manifest.ModelSHA256 || primary.Manifest.RuntimeSHA256 != frozen.Manifest.RuntimeSHA256 || primary.Manifest.EngineSHA256 != frozen.Manifest.EngineSHA256 {
			t.Fatal("selected-state source catalogue belongs to another candidate/toolchain")
		}
	}
	prepared, err := PreparedTarget(frozen)
	if err != nil {
		t.Fatal(err)
	}
	module, err := os.ReadFile(prepared.Path)
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
	request := wasi.Request{HelperPath: helper, HelperSHA256: strings.TrimPrefix(options.HelperSHA256, "sha256:"), ModulePath: prepared.Path, ModuleSHA256: strings.TrimPrefix(prepared.SHA256, "sha256:"), Engine: options.Engine, Imports: imports, InitialMemoryPages: pages, MemoryBytes: options.MemoryBytes, Fuel: options.Fuel, Config: config, Timeout: time.Minute, CompileCache: compileCache, StorageCrashLimits: &volume.CrashEnumerationLimits{States: 32, Operations: 8, Depth: 8, Bytes: 4 << 20, Work: 1 << 16}, Runtime: &wasi.RuntimeControl{Seed: 7, Choice: &runnerbackend.ChoiceRequest{Mode: choice.ModeRecord, Limit: 8 << 20, ExecutionIdentity: choice.ExecutionIdentity{TargetSHA256: targetHash, ToolchainBuildKey: prepared.BuildKey, GOOS: "wasip1", GOARCH: "wasm", ImplementationSHA256: implementation}}}}
	canary := os.Getenv("GOMAD_WASM_QUALIFICATION_CELL") == "canary"
	provider := &simulation.Provider{Boots: map[gomad3sim.BootID]wasi.Request{}}
	for _, phase := range []string{"setup-live", "append-crash", "recover"} {
		boot := request
		operation := phase
		if canary && phase == "append-crash" {
			operation = "unsynced"
		}
		boot.Config.Args = []string{"sqlite", operation, "WAL"}
		if selected && phase == "append-crash" {
			boot.StorageCrashSelection = map[string][]uint64{state.Volume: slices.Clone(state.SelectedOperations)}
			boot.StorageCrashContext = map[string]wasi.StorageCrashContext{state.Volume: crashContext.Clone()}
		}
		provider.Boots[gomad3sim.BootID(phase)] = boot
	}
	ack, recovered := gomad3sim.NodeHandle{Node: "sqlite", Incarnation: 2}, gomad3sim.NodeHandle{Node: "sqlite", Incarnation: 3}
	provider.IncarnationBoots = []simulation.IncarnationBoot{{Handle: ack, Boot: "append-crash"}, {Handle: recovered, Boot: "recover"}}
	provider.Oracles = &simulation.OraclePlan{Schema: simulation.SQLiteObservationSchema, Prefix: "HISTORY ", MaximumBytes: 1 << 20, SQLite: []simulation.SQLiteComparison{{Name: "acknowledged-durable-recovery", Acknowledged: ack, Recovered: recovered, ExpectedEvents: []string{"1", "2"}, ExpectedValue: "committed", ExpectedIntegrity: "ok"}}}
	identity, err := simulation.BootConfiguration(provider.Boots["setup-live"])
	if err != nil {
		t.Fatal(err)
	}
	cluster := gomad3sim.Spec{Schema: gomad3sim.SpecSchema, Backend: gomad3sim.BackendWASM, Fidelity: gomad3sim.FidelityHardIsolation, Seed: 7, Limits: gomad3sim.DefaultLimits(), Volumes: []gomad3sim.VolumeSpec{{ID: "sqlite", CapacityBytes: 4 << 20}}, Nodes: []gomad3sim.NodeSpec{{ID: "sqlite", Boot: "setup-live", Address: "10.0.0.1", Config: identity, Volumes: []gomad3sim.VolumeMount{{Volume: "sqlite", Path: "/volume"}}}}}
	scenario := func(ctx context.Context, c gomad3sim.Cluster) error {
		for phase := 0; phase < 3; phase++ {
			var handle gomad3sim.NodeHandle
			var err error
			if phase == 0 {
				handle, err = c.Start(ctx, "sqlite")
			} else {
				handle, err = c.Restart(ctx, "sqlite")
			}
			if err != nil {
				return err
			}
			if phase == 1 && !canary {
				if err := c.Crash(ctx, handle); err != nil {
					return err
				}
				continue
			}
			if _, err := c.Wait(ctx, handle); err != nil {
				return err
			}
		}
		return provider.RecordOracles(ctx, c)
	}
	result, err := RunCluster(t.Context(), frozen, cluster, scenario, provider)
	if err != nil {
		t.Fatal(err)
	}
	if selected {
		result.NativeCrashSelection = state
		result.NativeCrashSourceResultSHA256 = sourceHash
	}
	// All potentially large clones and retained artifact reads precede the
	// publication self snapshot. Preserve their failure for a bounded assertion
	// after publishing the raw Result, rather than losing failed evidence.
	var observationErr error
	stateCount := 0
	completed := provider.CompletedResults()
	for _, incarnation := range completed {
		observationErr = errors.Join(observationErr, splitServedOnlyOutput(incarnation.Result))
	}
	if len(completed) != 3 {
		observationErr = errors.Join(observationErr, errors.New("expected three actual incarnations"))
	} else if !canary {
		crashed := completed[1].Result
		enumeration := crashed.StorageCrashEnumeration["sqlite"]
		stateCount = len(enumeration.States)
		if crashed.Termination != "storage_crash" || string(crashed.Stdout) != "ACK 2\n" || !enumeration.Complete || stateCount <= 1 {
			observationErr = errors.New("actual acknowledged crash/enumeration missing")
		}
		_, catalogueErr := MatchedNativeCrashEnumeration(result, ack, "sqlite")
		observationErr = errors.Join(observationErr, catalogueErr)
		if selected {
			for _, execution := range result.Qualification.Executions {
				observationErr = errors.Join(observationErr, validateSQLiteArtifactSelection(execution.ArtifactPath, *state, crashContext))
			}
		}
	}
	// Preserve the first actual failed qualification and its raw costs before
	// asserting acceptance; infrastructure/input failures are not app canaries.
	path, receipt, err := WriteResultObserved(root, result)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("retained durable SQLite measured-result=%s", path)
	t.Logf("terminal publication self receipt=%+v", receipt)
	if !result.Qualification.Deterministic || result.Qualification.TargetSuccess == canary || result.Cost == nil || result.Cost.ReplayableBugs != uint64(boolInt(canary)) || result.Budget == nil || !result.Budget.WithinBudget {
		t.Fatalf("durable SQLite gate failed; retained=%s", path)
	}
	if observationErr != nil {
		t.Fatalf("durable SQLite observation gate failed: %v; retained=%s", observationErr, path)
	}
	if !canary {
		t.Logf("bounded native crash enumeration states=%d; default state measured, supplementary selections remain separately required", stateCount)
	}
	t.Logf("actual SQLite durable canary=%t module=%s result=%s cpu=%d operator-rss=%d", canary, prepared.SHA256, path, result.Cost.CPUNanos, result.Cost.OperatorPeakRSSBytes)
}

// splitServedOnlyOutput proves a broker storage incarnation used the helper's
// Rust volume (fn-176): Go's filesystem served exactly its fd 1/2 writes.
func splitServedOnlyOutput(result wasi.Result) error {
	writes := uint64(0)
	for op, count := range result.GoFilesystemCalls {
		if op != "fd_write" {
			return fmt.Errorf("broker storage incarnation served %s in Go: %v", op, result.GoFilesystemCalls)
		}
		writes += count
	}
	records, err := wasi.DecodeTranscript(result.Transcript)
	if err != nil {
		return err
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
		return fmt.Errorf("broker storage incarnation Go fs-served %v, want only its %d fd 1/2 writes", result.GoFilesystemCalls, output)
	}
	return nil
}
