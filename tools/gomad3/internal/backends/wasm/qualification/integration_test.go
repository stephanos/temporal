package qualification

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"go.temporal.io/server/tools/gomad3/internal/backends/target"
	"go.temporal.io/server/tools/gomad3/internal/backends/wasm/backend"
	"go.temporal.io/server/tools/gomad3/internal/backends/wasm/wasi"
	"go.temporal.io/server/tools/gomad3/internal/evidence/record"
	"go.temporal.io/server/tools/gomad3/internal/host/exec"
	"go.temporal.io/server/tools/gomad3/internal/runner"
)

func TestActualMatchedFixtureQualification(t *testing.T) {
	start := hostexec.CurrentProcessResourceUsage()
	qualificationIntent(t)
	integrated, err := record.ParseSHA256(os.Getenv("GOMAD_WASM_INTEGRATED_SOURCE_SHA256"))
	if err != nil {
		t.Fatal("explicit conductor current integrated source inventory digest required")
	}
	compiler := os.Getenv("GOMAD3_STOCK_GO")
	if compiler == "" {
		t.Fatal("explicit stock compiler required")
	}
	helper, err := filepath.Abs("../wasmhost/target/release/gomad3-wasmhost")
	if err != nil {
		t.Fatal(err)
	}
	helperBytes, err := os.ReadFile(helper)
	if err != nil {
		t.Fatal(err)
	}
	helper = filepath.Join(t.TempDir(), "gomad3-wasmhost")
	if err := os.WriteFile(helper, helperBytes, 0700); err != nil {
		t.Fatal(err)
	}
	compilerBytes, err := os.ReadFile(compiler)
	if err != nil {
		t.Fatal(err)
	}
	fixture, err := filepath.Abs("testdata/fixture")
	if err != nil {
		t.Fatal(err)
	}
	runtimeRoot, err := filepath.Abs("../../../model/runtimesource")
	if err != nil {
		t.Fatal(err)
	}
	root := os.Getenv("GOMAD_WASM_QUALIFICATION_ROOT")
	if root == "" {
		root = t.TempDir()
	}
	var previous Manifest
	for _, profile := range []string{"controlled", "cooperative"} {
		if selected := os.Getenv("GOMAD_WASM_QUALIFICATION_PROFILE"); selected != "" && selected != profile {
			continue
		}
		if os.Getenv("GOMAD_WASM_INTEGRATED_SOURCE_MANIFEST") != "" && os.Getenv("GOMAD_WASM_QUALIFICATION_PROFILE") == "" {
			t.Fatal("final matched fixture requires one profile per fresh operator process")
		}
		t.Run(profile, func(t *testing.T) {
			config := wasi.Config{Profile: wasi.StockProfile, Environment: []string{"PWD=/workspace"}, WorkingDirectory: "/workspace", WritableDirectories: []string{"/workspace", "/tmp"}, Clock: wasi.ClockPolicy{EpochNanos: 946684800000000000, ReadStepNanos: 1000}, Limits: wasi.Limits{OutputBytes: 1 << 20, TranscriptBytes: 8 << 20, Calls: 10000, Descriptors: 64, Files: 1000, FilesystemBytes: 8 << 20, PendingEvents: 4096}}
			if profile == "cooperative" {
				config.Profile = wasi.CooperativeProfile
				config.Clock.ReadStepNanos = 0
			}
			options := backend.Options{CompilerPath: compiler, CompilerSHA256: string(record.HashBytes(compilerBytes)), GoVersion: "go1.27.1", HelperPath: helper, HelperSHA256: string(record.HashBytes(helperBytes)), ModelSHA256: wasi.ImplementationSHA256(), CacheRoot: t.TempDir(), CompileCacheRoot: t.TempDir(), RuntimeRoot: runtimeRoot, Engine: wasi.PinnedEngine(), MemoryBytes: 256 << 20, Fuel: 4_000_000_000, Config: config}
			if err := os.Chmod(options.CompileCacheRoot, 0700); err != nil {
				t.Fatal(err)
			}
			spec := target.Spec{Backend: backend.Name, Kind: target.KindGoRun, Source: ".", WorkingDir: fixture, PreparationRoot: t.TempDir(), BuildTags: []string{"test_dep"}}
			frozen, err := Freeze(t.Context(), Spec{OperatorStart: start, Manifest: Manifest{Cell: os.Getenv("GOMAD_WASM_QUALIFICATION_CELL"), Provisional: os.Getenv("GOMAD_WASM_INTEGRATED_SOURCE_MANIFEST") == "", IntegratedSourceManifestPath: os.Getenv("GOMAD_WASM_INTEGRATED_SOURCE_MANIFEST"), ID: "ready-select-timer", Workload: "fixture", IntegratedSourceSHA256: integrated, Seed: 7, Repeat: 2, Unsupported: []string{"weak-memory and data races", "uncontrolled stock host WASI", "checkpoint", "cross-host exact portability"}}, Options: options, Campaign: runner.CampaignSpec{Target: spec}, ArtifactRoot: root, Command: []string{"wasm-qualify", profile}})
			if err != nil {
				t.Fatal(err)
			}
			manifest := frozen.Manifest
			if previous.Profile != "" {
				if err := Matched(previous, manifest); err != nil {
					t.Fatal(err)
				}
			}
			previous = manifest
			for _, canary := range []bool{false, true} {
				if !qualificationCell(t, canary) {
					continue
				}
				if canary {
					spec.Args = []string{"canary"}
				}
				campaign := runner.CampaignSpec{Target: spec, ExecutionTimeout: time.Minute, OverallTimeout: 3 * time.Minute, TerminateGrace: 100 * time.Millisecond, OnFailure: runner.PolicyAll, FailureBudget: 1, OutputLimit: 1 << 20, WorldTransitionLimit: 1 << 20, Artifacts: root, RunnerBuild: "fn156-qualification"}
				frozen.Campaign = campaign
				observed, err := Run(t.Context(), frozen)
				if err != nil {
					t.Fatal(err)
				}
				if !observed.Qualification.Deterministic || observed.Qualification.TargetSuccess == canary || len(observed.Qualification.Executions) != 2 || len(observed.Samples) != 4 {
					t.Fatalf("actual repeated qualification: %+v", observed)
				}
				for _, run := range observed.Qualification.Executions {
					if run.Replay == nil || !run.Replay.Match {
						t.Fatalf("missing actual replay: %+v", run)
					}
				}
				wantHits := uint64(1)
				if canary && os.Getenv("GOMAD_WASM_QUALIFICATION_CELL") == "" {
					wantHits = 2
				}
				if observed.Cost == nil || observed.CostFailure != "" || observed.Cost.CPUNanos == 0 || observed.Cost.CompileCacheHits != wantHits || observed.Cost.ReplayableBugs != uint64(boolInt(canary)) {
					t.Fatalf("measured gate not satisfied: %+v", observed)
				}
				path, _, err := WriteResultObserved(root, observed)
				if err != nil {
					t.Fatal(err)
				}
				// Full source-less report validation runs in the separately measured
				// corpus reader, not after this producer's terminal self snapshot.
				t.Logf("profile=%s canary=%t initial-preparation-wall=%s retained=%s measured-result=%s cpu-ns=%d rss=%d engine-compile-ns=%d", profile, canary, time.Duration(observed.InitialPreparationWallNanos), observed.QualificationPath, path, observed.Cost.CPUNanos, observed.Cost.HostPeakRSSBytes, observed.Cost.EngineCompileNanos)
			}
		})
	}
}

func qualificationIntent(t *testing.T) {
	t.Helper()
	digest, manifest, intent := os.Getenv("GOMAD_WASM_INTEGRATED_SOURCE_SHA256"), os.Getenv("GOMAD_WASM_INTEGRATED_SOURCE_MANIFEST"), os.Getenv("GOMAD_WASM_QUALIFY_WORKLOADS")
	if digest == "" && manifest == "" && intent == "" {
		t.Skip("set GOMAD_WASM_QUALIFY_WORKLOADS and conductor integrated-source manifest/digest for mandatory actual Phase 9 corpus gate")
	}
	if _, err := record.ParseSHA256(digest); err != nil {
		t.Fatal("qualification intent requires valid integrated-source digest")
	}
	if intent != "" && !filepath.IsAbs(manifest) {
		t.Fatal("mandatory workload qualification requires absolute frozen integrated-source manifest")
	}
	if manifest != "" && os.Getenv("GOMAD_WASM_QUALIFICATION_CELL") == "" {
		t.Fatal("final actual cell requires explicit success/canary selection in a fresh operator process")
	}
}

func qualificationCell(t *testing.T, canary bool) bool {
	t.Helper()
	switch selected := os.Getenv("GOMAD_WASM_QUALIFICATION_CELL"); selected {
	case "":
		return true
	case "success":
		return !canary
	case "canary":
		return canary
	default:
		t.Fatal("qualification cell must be success or canary")
		return false
	}
}

func boolInt(value bool) int {
	if value {
		return 1
	}
	return 0
}
