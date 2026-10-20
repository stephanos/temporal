package qualification

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"go.temporal.io/server/tools/gomad/internal/backends/target"
	"go.temporal.io/server/tools/gomad/internal/backends/wasm/backend"
	"go.temporal.io/server/tools/gomad/internal/backends/wasm/wasi"
	captured "go.temporal.io/server/tools/gomad/internal/evidence/artifact/capture"
	"go.temporal.io/server/tools/gomad/internal/evidence/record"
	hostexec "go.temporal.io/server/tools/gomad/internal/host/exec"
	"go.temporal.io/server/tools/gomad/internal/runner"
)

// oneboxCell is one unchanged Temporal test selection run in the one-box guest.
// Every cell shares the guest configuration below except its package and
// working directory; the canary always replaces the selection with "[".
type oneboxCell struct {
	id, source, workingDirectory, run, producer string
	command, unsupported                        []string
}

// This is the existing unchanged workflow-type smoke, not the full Temporal
// functional suite. The harness-invalid-regexp canary is not a storage oracle.
func TestActualOneboxQualification(t *testing.T) {
	start := hostexec.CurrentProcessResourceUsage()
	qualificationIntent(t)
	runOneboxCell(t, start, oneboxCell{id: "unchanged-workflow-type-smoke", source: "./tests", workingDirectory: "/workspace/tests", run: "^TestWorkflowTypeEncodingSuite$/^TestPlainASCII$/^Succeeds$", producer: "onebox-producer.json", command: []string{"wasm-qualify", "onebox", os.Getenv("GOMAD_WASM_QUALIFICATION_CELL")}, unsupported: []string{"controlled unchanged Temporal profile not qualified", "checkpoint", "uncontrolled host WASI", "native without current retained proof", "full functional suite not established by smoke"}})
}

// TestActualOneboxFrontendSystemInfoQualification is a second one-box family
// cell for the per-test support list. It is not part of ValidateCorpus's
// mandatory set, so it refuses the mandatory corpus intent and must publish
// into its own GOMAD_WASM_QUALIFICATION_ROOT.
func TestActualOneboxFrontendSystemInfoQualification(t *testing.T) {
	start := hostexec.CurrentProcessResourceUsage()
	qualificationIntent(t)
	if os.Getenv("GOMAD_WASM_QUALIFY_WORKLOADS") != "" {
		t.Fatal("frontend-system-info is not a mandatory corpus cell; run it without GOMAD_WASM_QUALIFY_WORKLOADS in its own qualification root")
	}
	runOneboxCell(t, start, oneboxCell{id: "frontend-system-info", source: "./tests/gomadfunctional", workingDirectory: "/workspace/tests/gomadfunctional", run: "^TestFrontendSystemInfo$", producer: "onebox-frontend-system-info-producer.json", command: []string{"wasm-qualify", "onebox", "frontend-system-info", os.Getenv("GOMAD_WASM_QUALIFICATION_CELL")}, unsupported: []string{"controlled unchanged Temporal profile not qualified", "checkpoint", "uncontrolled host WASI", "native without current retained proof", "full functional suite not established by one test"}})
}

func runOneboxCell(t *testing.T, start *hostexec.ResourceUsage, cell oneboxCell) {
	t.Helper()
	root := os.Getenv("GOMAD_WASM_QUALIFICATION_ROOT")
	if !filepath.IsAbs(root) {
		t.Fatal("onebox evidence root must persist and be absolute")
	}
	checkout, err := filepath.Abs("../../../../../..")
	if err != nil {
		t.Fatal(err)
	}
	compiler := os.Getenv("GOMAD_STOCK_GO")
	compilerBytes, err := os.ReadFile(compiler)
	if err != nil {
		t.Fatal(err)
	}
	helper, err := qualificationHelperPath()
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
	config, err := oneboxGuestConfig(checkout, cell.workingDirectory)
	if err != nil {
		t.Fatal(err)
	}
	cache := filepath.Join(root, "onebox-build-cache")
	compileCache := filepath.Join(root, "onebox-engine-cache")
	if err := os.MkdirAll(compileCache, 0700); err != nil {
		t.Fatal(err)
	}
	preparationRoot, err := os.MkdirTemp(root, "onebox-prepared-")
	if err != nil {
		t.Fatal(err)
	}
	options := backend.Options{CompilerPath: compiler, CompilerSHA256: string(record.HashBytes(compilerBytes)), GoVersion: "go1.27.1", HelperPath: helper, HelperSHA256: string(record.HashBytes(helperBytes)), ModelSHA256: wasi.ImplementationSHA256(), CacheRoot: cache, CompileCacheRoot: compileCache, RuntimeRoot: runtimeRoot, Engine: wasi.PinnedEngine(), MemoryBytes: 4 << 30, Fuel: 1 << 50, Config: config}
	args := []string{"-test.v", "-test.count=1", "-test.timeout=72h", "-test.run=" + cell.run, "-persistenceType=sql", "-persistenceDriver=sqlite"}
	canary := os.Getenv("GOMAD_WASM_QUALIFICATION_CELL") == "canary"
	if canary {
		args[3] = "-test.run=["
	}
	buildTags, err := qualificationGuestBuildTags(os.Getenv("GOMAD_WASM_QUALIFICATION_GUEST_PROFILE"))
	if err != nil {
		t.Fatal(err)
	}
	source := target.Spec{Backend: backend.Name, Kind: target.KindGoTest, Source: cell.source, WorkingDir: checkout, PreparationRoot: preparationRoot, BuildTags: buildTags, Args: args}
	manifestPath := os.Getenv("GOMAD_WASM_INTEGRATED_SOURCE_MANIFEST")
	integrated, err := record.ParseSHA256(os.Getenv("GOMAD_WASM_INTEGRATED_SOURCE_SHA256"))
	if err != nil {
		t.Fatal(err)
	}
	frozen, err := Freeze(t.Context(), Spec{OperatorStart: start, Manifest: Manifest{Cell: os.Getenv("GOMAD_WASM_QUALIFICATION_CELL"), Provisional: manifestPath == "", IntegratedSourceManifestPath: manifestPath, IntegratedSourceSHA256: integrated, ID: cell.id, Workload: "onebox", Seed: 7, Repeat: 2, Unsupported: cell.unsupported}, Options: options, Campaign: runner.CampaignSpec{Target: source, Environment: []string{"TEMPORAL_ROOT=/workspace", "TMPDIR=/tmp", "TEMPORAL_TEST_LOG_LEVEL=warn"}, ExecutionTimeout: 15 * time.Minute, OverallTimeout: 30 * time.Minute, TerminateGrace: time.Second, OutputLimit: 256 << 20, WorldTransitionLimit: 32 << 20, ChoiceTraceLimit: 64 << 20, OnFailure: runner.PolicyAll, FailureBudget: 1, Artifacts: root, SuccessBytesLimit: 2 << 30, RunnerBuild: "fn156-qualification"}, ArtifactRoot: root, Command: cell.command})
	if err != nil {
		t.Fatal(err)
	}
	prepared, err := PreparedTarget(frozen)
	if err != nil {
		t.Fatal(err)
	}
	// Detached genuine preparation and actual commands are shared with the
	// conductor/helper for the same-module gate; no second compiler is run here.
	producerPath, err := writePreparationProducer(root, cell.producer, frozen, prepared)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("genuine prepared module=%s sha=%s preparation=%s producer=%s", prepared.Path, prepared.SHA256, time.Duration(frozen.coldPreparationWallNanos), producerPath)
	if os.Getenv("GOMAD_WASM_QUALIFICATION_PREPARE_ONLY") == "1" {
		if os.Getenv("GOMAD_WASM_QUALIFY_WORKLOADS") != "" {
			t.Fatal("mandatory corpus cannot pass a prepare-only cell")
		}
		return
	}
	result, err := Run(t.Context(), frozen)
	if err != nil {
		t.Fatal(err)
	}
	path, _, err := WriteResultObserved(root, result)
	if err != nil {
		t.Fatal(err)
	}
	if !result.Qualification.Deterministic || result.Qualification.TargetSuccess == canary || result.Cost == nil || result.Cost.ReplayableBugs != uint64(boolInt(canary)) || result.Budget == nil || !result.Budget.WithinBudget {
		t.Fatalf("onebox qualification did not pass; retained=%s", path)
	}
	t.Logf("canary=%t measured-result=%s cpu=%d operator-rss=%d", canary, path, result.Cost.CPUNanos, result.Cost.OperatorPeakRSSBytes)
}

// This is bounded raw preparation evidence, not an adoption API: subsequent
// execution consumes the same private Frozen Spec, never this JSON file.
func writePreparationProducer(root, name string, frozen Spec, prepared target.Prepared) (string, error) {
	if filepath.Base(name) != name || !filepath.IsAbs(root) {
		return "", errors.New("private preparation producer path required")
	}
	producer, err := json.MarshalIndent(struct {
		Prepared             target.Prepared
		Options              backend.Options
		CompilerCommands     []backend.CompilerCommandMetrics
		PreparationWallNanos uint64
		OperatorBefore       *hostexec.ResourceUsage
		OperatorAfter        *hostexec.ResourceUsage
		Provisional          bool
	}{prepared, frozen.Options, frozen.initialCompilerCommands, frozen.coldPreparationWallNanos, frozen.operatorPreparationBefore, frozen.operatorPreparationAfter, frozen.Manifest.Provisional}, "", "  ")
	if err != nil {
		return "", err
	}
	if len(producer) >= 32<<20 {
		return "", errors.New("preparation producer evidence exceeds 32MiB")
	}
	path := filepath.Join(root, name)
	return path, os.WriteFile(path, append(producer, '\n'), 0600)
}

// oneboxGuestConfig is the one-box guest configuration; only the working
// directory differs between one-box family cells.
func oneboxGuestConfig(root, workingDirectory string) (wasi.Config, error) {
	return wasi.TemporalConfig(root, workingDirectory, wasi.TemporalQualification)
}

// The smoke cell's host configuration identity must not move when the
// frontend cell shares its construction.
func TestOneboxSmokeConfigurationUnchangedByFamilyCells(t *testing.T) {
	root := t.TempDir()
	schema := filepath.Join(root, "schema", "sqlite")
	if err := os.MkdirAll(schema, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(schema, "schema.sql"), []byte("CREATE TABLE example(id INTEGER);"), 0600); err != nil {
		t.Fatal(err)
	}
	inputs, err := captured.CaptureReadOnlyMountInputs([]captured.Mapping{{Source: schema, Target: "/workspace/schema/sqlite"}}, captured.DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	original := wasi.Config{Profile: wasi.CooperativeProfile, WorkingDirectory: "/workspace/tests", Environment: []string{"PWD=/workspace/tests", "TEMPORAL_ROOT=/workspace", "TMPDIR=/tmp", "TZ=UTC", "TEMPORAL_TEST_LOG_LEVEL=warn"}, WritableDirectories: []string{"/workspace", "/tmp", "/workspace/tests"}, CapturedInputs: inputs, EntropyKey: sha256.Sum256([]byte("gomad-wasm-prototype/v1")), Clock: wasi.ClockPolicy{EpochNanos: 946684800000000000}, Limits: wasi.Limits{OutputBytes: 256 << 20, TranscriptBytes: 1 << 30, Calls: 1 << 28, Descriptors: 4096, Files: 100000, FilesystemBytes: 512 << 20, PendingEvents: 4096}}
	want, err := json.Marshal(original)
	if err != nil {
		t.Fatal(err)
	}
	config, err := oneboxGuestConfig(root, "/workspace/tests")
	if err != nil {
		t.Fatal(err)
	}
	got, err := json.Marshal(config)
	if err != nil || string(got) != string(want) {
		t.Fatalf("one-box smoke configuration bytes changed: %v", err)
	}
	frontend, err := oneboxGuestConfig(root, "/workspace/tests/gomadfunctional")
	if err != nil {
		t.Fatal(err)
	}
	if frontend.WorkingDirectory != "/workspace/tests/gomadfunctional" || frontend.Environment[0] != "PWD=/workspace/tests/gomadfunctional" || frontend.WritableDirectories[2] != frontend.WorkingDirectory {
		t.Fatalf("frontend cell configuration = %+v", frontend)
	}
}
