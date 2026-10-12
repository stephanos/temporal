package qualification

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"go.temporal.io/server/tools/gomad3/deterministicio/volume"
	"go.temporal.io/server/tools/gomad3/hostexec"
	"go.temporal.io/server/tools/gomad3/record"
	"go.temporal.io/server/tools/gomad3/runner"
	"go.temporal.io/server/tools/gomad3/target"
	"go.temporal.io/server/tools/gomad_wasm/backend"
	"go.temporal.io/server/tools/gomad_wasm/wasi"
)

// This cell proves real file-backed SQLite preparation/execution/retained replay
// costs. Crash/reopen and missing-sync oracle cells remain separate required
// corpus gates; a nonzero operation canary is not their substitute.
func TestActualSQLiteQualification(t *testing.T) {
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
	runtimeRoot, err := filepath.Abs("../../gomad3/runtimesemantics")
	if err != nil {
		t.Fatal(err)
	}
	directory, err := filepath.Abs("../../..")
	if err != nil {
		t.Fatal(err)
	}
	config := wasi.Config{Profile: wasi.CooperativeProfile, Environment: []string{"PWD=/workspace", "TZ=UTC"}, WorkingDirectory: "/workspace", WritableDirectories: []string{"/workspace", "/tmp", "/volume"}, Clock: wasi.ClockPolicy{EpochNanos: 946684800000000000}, Limits: wasi.Limits{OutputBytes: 1 << 20, TranscriptBytes: 32 << 20, Calls: 100000, Descriptors: 128, Files: 1024, FilesystemBytes: 8 << 20, PendingEvents: 4096}, Storage: &wasi.StorageModel{Volumes: []volume.VolumeConfig{{ID: "sqlite", Path: "/volume", CapacityBytes: 4 << 20}}, Limits: volume.VolumeLimits{PendingOperations: 10000, Transitions: 100000}}}
	options := backend.Options{CompilerPath: compiler, CompilerSHA256: string(record.HashBytes(compilerBytes)), GoVersion: "go1.27.1", HelperPath: helper, HelperSHA256: string(record.HashBytes(helperBytes)), ModelSHA256: wasi.ImplementationSHA256(), CacheRoot: t.TempDir(), CompileCacheRoot: t.TempDir(), RuntimeRoot: runtimeRoot, Engine: wasi.PinnedEngine(), MemoryBytes: 256 << 20, Fuel: 4_000_000_000, Config: config}
	if err = os.Chmod(options.CompileCacheRoot, 0700); err != nil {
		t.Fatal(err)
	}
	root := os.Getenv("GOMAD_WASM_QUALIFICATION_ROOT")
	if root == "" {
		root = t.TempDir()
	}
	preparedSpec := target.Spec{Backend: backend.Name, Kind: target.KindGoRun, Source: "./tools/gomad_wasm/testdata/storage_sqlite/main.go", WorkingDir: directory, PreparationRoot: t.TempDir(), BuildTags: []string{"sqlite3_dotlk"}}
	frozen, err := Freeze(t.Context(), Spec{OperatorStart: start, Manifest: Manifest{Cell: os.Getenv("GOMAD_WASM_QUALIFICATION_CELL"), Provisional: os.Getenv("GOMAD_WASM_INTEGRATED_SOURCE_MANIFEST") == "", IntegratedSourceManifestPath: os.Getenv("GOMAD_WASM_INTEGRATED_SOURCE_MANIFEST"), ID: "sqlite-file-backed-operation", Workload: "sqlite", IntegratedSourceSHA256: integrated, Seed: 7, Repeat: 2, Unsupported: []string{"controlled SQLite profile not qualified", "checkpoint", "uncontrolled host WASI", "native without retained current proof"}}, Options: options, Campaign: runner.CampaignSpec{Target: preparedSpec}, ArtifactRoot: root, Command: []string{"wasm-qualify", "sqlite"}})
	if err != nil {
		t.Fatal(err)
	}
	for _, canary := range []bool{false, true} {
		if !qualificationCell(t, canary) {
			continue
		}
		preparedSpec.Args = []string{"setup", "WAL"}
		if canary {
			preparedSpec.Args = []string{"not-an-operation", "WAL"}
		}
		frozen.Campaign = runner.CampaignSpec{Target: preparedSpec, ExecutionTimeout: time.Minute, OverallTimeout: 3 * time.Minute, TerminateGrace: 100 * time.Millisecond, OnFailure: runner.PolicyAll, FailureBudget: 1, OutputLimit: 1 << 20, WorldTransitionLimit: 1 << 20, Artifacts: root, RunnerBuild: "fn156-qualification"}
		result, err := Run(t.Context(), frozen)
		if err != nil {
			t.Fatal(err)
		}
		if !result.Qualification.Deterministic || result.Qualification.TargetSuccess == canary || result.Cost == nil || result.Cost.ReplayableBugs != uint64(boolInt(canary)) || result.Budget == nil || !result.Budget.WithinBudget {
			t.Fatalf("actual SQLite qualification: support=%+v cost-failure=%s budget=%+v", result.Qualification, result.CostFailure, result.Budget)
		}
		for _, sample := range result.Samples {
			if sample.StorageTraceBytes == 0 {
				t.Fatal("SQLite modeled storage trace missing")
			}
		}
		path, err := WriteResult(root, result)
		if err != nil {
			t.Fatal(err)
		}
		if _, err = OpenResult(path); err != nil {
			t.Fatal(err)
		}
		t.Logf("canary=%t retained=%s cpu=%d helper-rss=%d operator-rss=%d trace=%d", canary, path, result.Cost.CPUNanos, result.Cost.HostPeakRSSBytes, result.Cost.OperatorPeakRSSBytes, result.Cost.TraceBytes)
	}
}
