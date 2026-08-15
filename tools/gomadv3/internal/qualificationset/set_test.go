package qualificationset

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"go.temporal.io/server/tools/gomadv3/internal/capabilityanalysis"
	"go.temporal.io/server/tools/gomadv3/internal/choicewire"
	"go.temporal.io/server/tools/gomadv3/internal/compatibility"
	"go.temporal.io/server/tools/gomadv3/internal/ioprofile"
	"go.temporal.io/server/tools/gomadv3/internal/qualify"
	"go.temporal.io/server/tools/gomadv3/internal/record"
	"go.temporal.io/server/tools/gomadv3/internal/target"
)

func TestRunPublishesCheckedExpectedBoundaryEvidence(t *testing.T) {
	root := t.TempDir()
	manifestPath := writeManifest(t, root, "unsupported_target")
	output := filepath.Join(root, "set-report.json")
	report, err := Run(context.Background(), Config{
		ManifestPath: manifestPath, GomadPath: filepath.Join(root, "gomad"), WorkingDir: root,
		ArtifactRoot: filepath.Join(root, "artifacts"), OutputPath: output, Execute: expectedBoundaryExecutor(t),
	})
	if err != nil {
		t.Fatal(err)
	}
	if !report.ExpectationsMet || report.Completed != 1 || report.Supported != 0 || report.Unsupported != 1 || report.Failed != 0 || report.InfrastructureErrors != 0 || len(report.Suites) != 1 || !report.Suites[0].ExpectationMet || report.Suites[0].Classification != "unsupported_target" || report.Suites[0].Analysis == nil || len(report.Suites[0].Blockers) != 1 {
		t.Fatalf("qualification set report = %#v", report)
	}
	contents, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	var public map[string]any
	if err := json.Unmarshal(contents, &public); err != nil {
		t.Fatal(err)
	}
	if public["schema"] != "gomadv3.qualification-set-report/v4" || public["expectations_met"] != true || public["supported"] != float64(0) || public["unsupported"] != float64(1) || public["failed"] != float64(0) || public["infrastructure_errors"] != float64(0) {
		t.Fatalf("public qualification set report = %#v", public)
	}
	for _, forbidden := range []string{"manifest", "report_path", "command"} {
		if _, found := public[forbidden]; found {
			t.Fatalf("public qualification set report retains %s: %#v", forbidden, public)
		}
	}
	opened, err := OpenReport(output)
	if err != nil {
		t.Fatal(err)
	}
	if opened.ManifestSHA256 == "" || opened.Module.GoModSHA256 == "" || opened.Suites[0].Analysis == nil {
		t.Fatalf("opened qualification set report = %#v", opened)
	}
	if err := os.Chmod(output, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := OpenReport(output); err == nil {
		t.Fatal("OpenReport() accepted a non-private report")
	}
}

func TestOpenReportNormalizesPreviousExactChoiceDimensions(t *testing.T) {
	root := t.TempDir()
	report, err := Run(context.Background(), Config{
		ManifestPath: writeManifest(t, root, "unsupported_target"), GomadPath: filepath.Join(root, "gomad"), WorkingDir: root,
		ArtifactRoot: filepath.Join(root, "artifacts"), OutputPath: filepath.Join(root, "current.json"), Execute: expectedBoundaryExecutor(t),
	})
	if err != nil {
		t.Fatal(err)
	}
	report.Schema = PreviousReportSchema
	encoded, err := record.CanonicalJSON(report)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, "previous.json")
	if err := os.WriteFile(path, append(encoded, '\n'), 0o600); err != nil {
		t.Fatal(err)
	}
	opened, err := OpenReport(path)
	if err != nil {
		t.Fatal(err)
	}
	if opened.Schema != ReportSchema || opened.Suites[0].Choice.ExactReplayAvailable {
		t.Fatalf("normalized previous report = %#v", opened)
	}
}

func TestRunRetainsAllEvidenceWhenAnExpectationChanges(t *testing.T) {
	root := t.TempDir()
	output := filepath.Join(root, "set-report.json")
	report, err := Run(context.Background(), Config{
		ManifestPath: writeManifest(t, root, "qualified"), GomadPath: filepath.Join(root, "gomad"), WorkingDir: root,
		ArtifactRoot: filepath.Join(root, "artifacts"), OutputPath: output, Execute: expectedBoundaryExecutor(t),
	})
	var mismatch *ExpectationError
	if !errors.As(err, &mismatch) || report.ExpectationsMet || report.Completed != 1 || len(report.Suites) != 1 || report.Suites[0].ExpectationMet {
		t.Fatalf("qualification set report = %#v, error = %v", report, err)
	}
	if _, err := OpenReport(output); err != nil {
		t.Fatal(err)
	}
}

func TestRunReportRetainsRequestedSeeds(t *testing.T) {
	root := t.TempDir()
	output := filepath.Join(root, "set-report.json")
	_, err := Run(context.Background(), Config{
		ManifestPath: writeV2Manifest(t, root, []uint64{7, 11}, "unsupported_target"),
		GomadPath:    filepath.Join(root, "gomad"), WorkingDir: root,
		ArtifactRoot: filepath.Join(root, "artifacts"), OutputPath: output, Execute: expectedBoundaryExecutor(t),
	})
	if err != nil {
		t.Fatal(err)
	}
	contents, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(contents, &decoded); err != nil {
		t.Fatal(err)
	}
	seeds, ok := decoded["seeds"].([]any)
	if !ok || len(seeds) != 2 || seeds[0] != "7" || seeds[1] != "11" {
		t.Fatalf("report seeds = %#v", decoded["seeds"])
	}
}

func TestOpenReportRejectsCountersThatDisagreeWithWorkloads(t *testing.T) {
	root := t.TempDir()
	output := filepath.Join(root, "set-report.json")
	_, err := Run(context.Background(), Config{
		ManifestPath: writeV2Manifest(t, root, []uint64{7}, "unsupported_target"),
		GomadPath:    filepath.Join(root, "gomad"), WorkingDir: root,
		ArtifactRoot: filepath.Join(root, "artifacts"), OutputPath: output, Execute: expectedBoundaryExecutor(t),
	})
	if err != nil {
		t.Fatal(err)
	}
	contents, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	var report SetReport
	if err := record.DecodeCanonicalJSON(bytes.TrimSuffix(contents, []byte{'\n'}), &report); err != nil {
		t.Fatal(err)
	}
	report.Unsupported = 0
	report.Supported = 1
	tampered, err := record.CanonicalJSON(report)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(output, append(tampered, '\n'), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := OpenReport(output); err == nil {
		t.Fatal("OpenReport() accepted counters that disagree with workload evidence")
	}
}

func TestOpenReportRejectsHardLinkedEvidence(t *testing.T) {
	root := t.TempDir()
	output := filepath.Join(root, "set-report.json")
	_, err := Run(context.Background(), Config{
		ManifestPath: writeV2Manifest(t, root, []uint64{7}, "unsupported_target"),
		GomadPath:    filepath.Join(root, "gomad"), WorkingDir: root,
		ArtifactRoot: filepath.Join(root, "artifacts"), OutputPath: output, Execute: expectedBoundaryExecutor(t),
	})
	if err != nil {
		t.Fatal(err)
	}
	linked := filepath.Join(root, "linked-report.json")
	if err := os.Link(output, linked); err != nil {
		t.Fatal(err)
	}
	if _, err := OpenReport(linked); err == nil {
		t.Fatal("OpenReport() accepted hard-linked evidence")
	}
}

func TestRunCountsRetainedRunnerFailureAsInfrastructure(t *testing.T) {
	root := t.TempDir()
	output := filepath.Join(root, "set-report.json")
	report, err := Run(context.Background(), Config{
		ManifestPath: writeManifest(t, root, "qualified"), GomadPath: filepath.Join(root, "gomad"), WorkingDir: root,
		ArtifactRoot: filepath.Join(root, "artifacts"), OutputPath: output, Execute: failureExecutor(t, "runner_failure"),
	})
	var mismatch *ExpectationError
	if !errors.As(err, &mismatch) || report.Completed != 1 || report.Supported != 0 || report.Unsupported != 0 || report.Failed != 0 || report.InfrastructureErrors != 1 {
		t.Fatalf("qualification set report = %#v, error = %v", report, err)
	}
	if _, err := OpenReport(output); err != nil {
		t.Fatal(err)
	}
}

func TestRunPreservesRequestedSeedWhenEvidenceProjectionFails(t *testing.T) {
	root := t.TempDir()
	report, err := Run(context.Background(), Config{
		ManifestPath: writeV2Manifest(t, root, []uint64{7}, "qualified"),
		GomadPath:    filepath.Join(root, "gomad"), WorkingDir: root,
		ArtifactRoot: filepath.Join(root, "artifacts"), OutputPath: filepath.Join(root, "set-report.json"),
		Execute: failureExecutor(t, "runner_failure"),
	})
	var mismatch *ExpectationError
	if !errors.As(err, &mismatch) {
		t.Fatalf("Run() error = %v", err)
	}
	if len(report.Suites[0].Seeds) != 1 || report.Suites[0].Seeds[0].Seed != 7 || report.Suites[0].Seeds[0].Classification != "runner_failure" {
		t.Fatalf("seed evidence = %#v", report.Suites[0].Seeds)
	}
}

func TestLoadManifestRejectsDuplicateSuiteNames(t *testing.T) {
	root := t.TempDir()
	path := writeManifest(t, root, "unsupported_target")
	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var manifest map[string]any
	if err := json.Unmarshal(contents, &manifest); err != nil {
		t.Fatal(err)
	}
	suites := manifest["suites"].([]any)
	manifest["suites"] = append(suites, suites[0])
	contents, err = json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, contents, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadManifest(path); err == nil {
		t.Fatal("LoadManifest() accepted duplicate suite names")
	}
}

func TestLoadManifestNormalizesLegacyManifest(t *testing.T) {
	manifest, err := LoadManifest(writeManifest(t, t.TempDir(), "qualified"))
	if err != nil {
		t.Fatal(err)
	}
	if manifest.Schema != ManifestSchema || len(manifest.Seeds) != 1 || manifest.Seeds[0] != 7 || !manifest.legacy || manifest.Suites[0].ID != "fixture-case" || manifest.Suites[0].Tier != 1 {
		t.Fatalf("normalized legacy manifest = %#v", manifest)
	}
}

func TestLoadManifestV2RetainsPortableWorkloadContract(t *testing.T) {
	root := t.TempDir()
	manifest := map[string]any{
		"schema": ManifestSchema, "name": "test-set", "description": "portable qualification set", "module": "example.com/target",
		"seeds": []uint64{7, 11}, "repeat": 2,
		"run_timeout": "30s", "overall_timeout": "2m", "terminate_grace": "2s",
		"output_bytes": 1024, "world_transition_bytes": 2048,
		"suites": []any{map[string]any{
			"id": "fixture-case", "name": "Fixture case", "tier": 1, "invariant": "the fixture remains deterministic",
			"package": "./pkg", "test": "TestScenario", "choice_bytes": 4096, "replay_successes": true,
			"success_artifact_limit": 1, "success_bytes_limit": 1048576,
			"expectation": map[string]any{"classification": "qualified"},
		}},
	}
	contents, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, "manifest-v2.json")
	if err := os.WriteFile(path, contents, 0o600); err != nil {
		t.Fatal(err)
	}
	loaded, err := LoadManifest(path)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.legacy || loaded.Module != "example.com/target" || len(loaded.Seeds) != 2 || loaded.Suites[0].ID != "fixture-case" || loaded.Suites[0].ChoiceBytes != 4096 || !loaded.Suites[0].ReplaySuccesses {
		t.Fatalf("loaded manifest = %#v", loaded)
	}
}

func TestIdentifyModuleRequiresExactSafeGoMod(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/target\n\ngo 1.26.4\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	identity, err := identifyModule(root, "example.com/target")
	if err != nil {
		t.Fatal(err)
	}
	if identity.Path != "example.com/target" || identity.GoModSHA256 == "" {
		t.Fatalf("module identity = %#v", identity)
	}
	if _, err := identifyModule(root, "example.com/other"); err == nil {
		t.Fatal("identifyModule() accepted a different expected module")
	}

	symlinkRoot := t.TempDir()
	if err := os.Symlink(filepath.Join(root, "go.mod"), filepath.Join(symlinkRoot, "go.mod")); err != nil {
		t.Fatal(err)
	}
	if _, err := identifyModule(symlinkRoot, "example.com/target"); err == nil {
		t.Fatal("identifyModule() accepted a symbolic go.mod")
	}

	linkedWorkingDirectory := filepath.Join(t.TempDir(), "module")
	if err := os.Symlink(root, linkedWorkingDirectory); err != nil {
		t.Fatal(err)
	}
	if _, err := identifyModule(linkedWorkingDirectory, "example.com/target"); err == nil {
		t.Fatal("identifyModule() accepted a symbolic working directory")
	}
}

func TestOpenReportNormalizesLegacyPortableDimensions(t *testing.T) {
	manifest := legacyManifest{
		Schema: LegacyManifestSchema, Name: "test-set", Seed: 7, Repeat: 2,
		RunTimeout: "30s", OverallTimeout: "2m", TerminateGrace: "2s", OutputBytes: 1024, WorldTransitionBytes: 2048,
		Suites: []legacySuite{{Name: "fixture-case", Package: "./pkg", Test: "TestScenario", Expectation: Expectation{
			Classification: "unsupported_target", ImportPath: "golang.org/x/net/internal/socket", Capability: "uses go:linkname in sys_unix.go",
		}}},
	}
	manifestBytes, err := record.CanonicalJSON(manifest)
	if err != nil {
		t.Fatal(err)
	}
	command := []string{"gomad", "qualify", "--json", "go-test", "./pkg"}
	qualification, err := qualify.BuildFailure(command, 7, 2, nil, qualify.Failure{
		Classification: "unsupported_target", Message: "unsupported", Iteration: 1,
		ImportPath: "golang.org/x/net/internal/socket", Capability: "uses go:linkname in sys_unix.go",
	})
	if err != nil {
		t.Fatal(err)
	}
	qualificationBytes, err := record.CanonicalJSON(qualification)
	if err != nil {
		t.Fatal(err)
	}
	legacy := legacySetReport{
		Schema: LegacyReportSchema, Name: "test-set", ExpectationsMet: true, Selected: 1, Completed: 1, Unsupported: 1,
		ManifestSHA256: record.HashBytes(manifestBytes), Manifest: manifest,
		Suites: []legacySuiteReport{{
			Name: "fixture-case", Expected: manifest.Suites[0].Expectation, ExpectationMet: true,
			Classification: "unsupported_target", ExitCode: 2, Command: command,
			StdoutSHA256: record.HashBytes([]byte("stdout")), StderrSHA256: record.HashBytes(nil),
			ReportPath: "/private/artifacts/report.json", ReportSHA256: record.HashBytes(append(append([]byte(nil), qualificationBytes...), '\n')),
			Report: qualificationBytes,
		}},
	}
	encoded, err := record.CanonicalJSON(legacy)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "legacy.json")
	if err := os.WriteFile(path, append(encoded, '\n'), 0o600); err != nil {
		t.Fatal(err)
	}
	opened, err := OpenReport(path)
	if err != nil {
		t.Fatal(err)
	}
	if opened.Schema != ReportSchema || opened.Dimensions.PortableV3 || opened.Dimensions.Analysis || opened.Suites[0].AnalysisError != "dimension_unavailable" {
		t.Fatalf("normalized legacy report = %#v", opened)
	}
	normalized, err := record.CanonicalJSON(opened)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(normalized), "/private/") || strings.Contains(string(normalized), `"command"`) {
		t.Fatalf("normalized report contains operational paths: %s", normalized)
	}
}

func TestRunAnalyzesEveryWorkloadBeforeTargetExecution(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/target\n\ngo 1.26.4\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	legacy := map[string]any{
		"schema": LegacyManifestSchema, "name": "test-set", "seed": 7, "repeat": 2,
		"run_timeout": "30s", "overall_timeout": "2m", "terminate_grace": "2s", "output_bytes": 1024, "world_transition_bytes": 2048,
		"suites": []any{
			map[string]any{"name": "a-unsupported", "package": "./a", "test": "TestScenario", "expectation": map[string]any{"classification": "unsupported_target", "import_path": "golang.org/x/net/internal/socket", "capability": "uses go:linkname in sys_unix.go"}},
			map[string]any{"name": "b-supported", "package": "./b", "test": "TestScenario", "expectation": map[string]any{"classification": "qualified"}},
		},
	}
	contents, err := json.Marshal(legacy)
	if err != nil {
		t.Fatal(err)
	}
	manifestPath := filepath.Join(root, "manifest.json")
	if err := os.WriteFile(manifestPath, contents, 0o600); err != nil {
		t.Fatal(err)
	}
	calls := []string{}
	executor := func(_ context.Context, command Command) CommandResult {
		phase := command.Args[0]
		calls = append(calls, phase)
		if phase == "analyze" {
			classification := capabilityanalysis.ClassificationSupported
			if slices.Contains(command.Args, "./a") {
				classification = capabilityanalysis.ClassificationUnsupported
			}
			return encodedAnalysisResult(t, classification)
		}
		return failureExecutor(t, "runner_failure")(context.Background(), command)
	}
	report, err := Run(context.Background(), Config{
		ManifestPath: manifestPath, GomadPath: filepath.Join(root, "gomad"), WorkingDir: root,
		ArtifactRoot: filepath.Join(root, "artifacts"), OutputPath: filepath.Join(root, "report.json"), Execute: executor,
	})
	var mismatch *ExpectationError
	if !errors.As(err, &mismatch) || !slices.Equal(calls, []string{"analyze", "analyze", "qualify"}) || report.AnalysisCompleted != 2 || report.Unsupported != 1 || report.InfrastructureErrors != 1 {
		t.Fatalf("calls=%v report=%#v error=%v", calls, report, err)
	}
}

func TestRunClassifiesEveryWorkloadWhenAnalysisFails(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/target\n\ngo 1.26.4\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	manifest := map[string]any{
		"schema": ManifestSchema, "name": "test-set", "description": "analysis failure fixture", "module": "example.com/target",
		"seeds": []uint64{7}, "repeat": 2, "run_timeout": "30s", "overall_timeout": "2m", "terminate_grace": "2s",
		"output_bytes": 1024, "world_transition_bytes": 2048,
		"suites": []any{
			map[string]any{"id": "a-supported", "name": "A", "tier": 1, "invariant": "A remains classified", "package": "./a", "test": "TestScenario", "choice_bytes": 1024, "replay_successes": true, "success_artifact_limit": 2, "success_bytes_limit": 4096, "expectation": map[string]any{"classification": "qualified"}},
			map[string]any{"id": "b-broken", "name": "B", "tier": 1, "invariant": "B reports analysis failure", "package": "./b", "test": "TestScenario", "choice_bytes": 1024, "replay_successes": true, "success_artifact_limit": 2, "success_bytes_limit": 4096, "expectation": map[string]any{"classification": "qualified"}},
			map[string]any{"id": "c-supported", "name": "C", "tier": 1, "invariant": "C remains classified", "package": "./c", "test": "TestScenario", "choice_bytes": 1024, "replay_successes": true, "success_artifact_limit": 2, "success_bytes_limit": 4096, "expectation": map[string]any{"classification": "qualified"}},
		},
	}
	contents, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	manifestPath := filepath.Join(root, "manifest.json")
	if err := os.WriteFile(manifestPath, contents, 0o600); err != nil {
		t.Fatal(err)
	}
	executor := func(_ context.Context, command Command) CommandResult {
		if slices.Contains(command.Args, "./b") {
			return CommandResult{ExitCode: 3, Err: errors.New("analysis failed")}
		}
		return encodedAnalysisResult(t, capabilityanalysis.ClassificationSupported)
	}
	report, err := Run(context.Background(), Config{
		ManifestPath: manifestPath, GomadPath: filepath.Join(root, "gomad"), WorkingDir: root,
		ArtifactRoot: filepath.Join(root, "artifacts"), OutputPath: filepath.Join(root, "report.json"), Execute: executor,
	})
	var mismatch *ExpectationError
	if !errors.As(err, &mismatch) {
		t.Fatalf("Run() error = %v", err)
	}
	if report.InfrastructureErrors != 3 || report.Completed != 0 {
		t.Fatalf("report counts = completed %d, infrastructure %d", report.Completed, report.InfrastructureErrors)
	}
	for index, workload := range report.Suites {
		if workload.Classification != "runner_failure" {
			t.Fatalf("workload %d classification = %q", index, workload.Classification)
		}
	}
}

func TestAggregateChoiceCoverageRejectsIdentityMismatch(t *testing.T) {
	left := ChoiceCoverage{
		Available: true, Profile: choicewire.Profile, ImplementationSHA256: record.HashBytes([]byte("left")),
		Limit: 1024, Features: []choicewire.Feature{},
	}
	right := left
	right.ImplementationSHA256 = record.HashBytes([]byte("right"))
	if _, err := aggregateChoiceCoverage([]SeedReport{{Choice: left}, {Choice: right}}); err == nil {
		t.Fatal("aggregateChoiceCoverage() accepted inconsistent choice identities")
	}
}

func TestRunLeavesPrivateCheckpointWhenInterrupted(t *testing.T) {
	root := t.TempDir()
	manifestPath := writeManifest(t, root, "qualified")
	output := filepath.Join(root, "report.json")
	ctx, cancel := context.WithCancel(context.Background())
	executor := func(_ context.Context, command Command) CommandResult {
		if command.Args[0] == "analyze" {
			cancel()
			return encodedAnalysisResult(t, capabilityanalysis.ClassificationSupported)
		}
		t.Fatal("unexpected target execution after cancellation")
		return CommandResult{}
	}
	_, err := Run(ctx, Config{
		ManifestPath: manifestPath, GomadPath: filepath.Join(root, "gomad"), WorkingDir: root,
		ArtifactRoot: filepath.Join(root, "artifacts"), OutputPath: output, Execute: executor,
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Run() error = %v", err)
	}
	info, statErr := os.Stat(output + ".partial")
	if statErr != nil {
		t.Fatal(statErr)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("checkpoint mode = %o", info.Mode().Perm())
	}
}

func writeManifest(t *testing.T, root, classification string) string {
	t.Helper()
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/target\n\ngo 1.26.4\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	expectation := map[string]any{"classification": classification}
	if classification == "unsupported_target" {
		expectation["import_path"] = "golang.org/x/net/internal/socket"
		expectation["capability"] = "uses go:linkname in sys_unix.go"
	}
	manifest := map[string]any{
		"schema": "gomadv3.qualification-set/v1", "name": "test-set", "seed": 7, "repeat": 2,
		"run_timeout": "30s", "overall_timeout": "2m", "terminate_grace": "2s",
		"output_bytes": 1024, "world_transition_bytes": 2048,
		"suites": []any{map[string]any{
			"name": "fixture-case", "package": "./pkg", "test": "TestScenario",
			"expectation": expectation,
		}},
	}
	contents, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, "manifest.json")
	if err := os.WriteFile(path, contents, 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func writeV2Manifest(t *testing.T, root string, seeds []uint64, classification string) string {
	t.Helper()
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/target\n\ngo 1.26.4\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	expectation := map[string]any{"classification": classification}
	if classification == "unsupported_target" {
		expectation["import_path"] = "golang.org/x/net/internal/socket"
		expectation["capability"] = "uses go:linkname in sys_unix.go"
	}
	manifest := map[string]any{
		"schema": ManifestSchema, "name": "test-set", "description": "portable fixture", "module": "example.com/target",
		"seeds": seeds, "repeat": 2, "run_timeout": "30s", "overall_timeout": "2m", "terminate_grace": "2s",
		"output_bytes": 1024, "world_transition_bytes": 2048,
		"suites": []any{map[string]any{
			"id": "fixture-case", "name": "Fixture case", "tier": 1, "invariant": "the fixture remains deterministic",
			"package": "./pkg", "test": "TestScenario", "choice_bytes": 4096, "replay_successes": true,
			"success_artifact_limit": 2, "success_bytes_limit": 1048576, "expectation": expectation,
		}},
	}
	contents, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, "manifest-v2.json")
	if err := os.WriteFile(path, contents, 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func expectedBoundaryExecutor(t *testing.T) ExecuteFunc {
	t.Helper()
	return func(_ context.Context, command Command) CommandResult {
		if command.Args[0] == "analyze" {
			return encodedAnalysisResult(t, capabilityanalysis.ClassificationUnsupported)
		}
		logicalCommand := append([]string{"gomad"}, command.Args...)
		report, err := qualify.BuildFailure(logicalCommand, 7, 2, nil, qualify.Failure{
			Classification: "unsupported_target", Message: "unsupported boundary", Iteration: 1,
			ImportPath: "golang.org/x/net/internal/socket", Capability: "uses go:linkname in sys_unix.go",
		})
		if err != nil {
			t.Fatal(err)
		}
		path, err := qualify.Write(command.ArtifactRoot, report)
		if err != nil {
			t.Fatal(err)
		}
		var event bytes.Buffer
		if err := qualify.WriteResultEvent(&event, report, path); err != nil {
			t.Fatal(err)
		}
		return CommandResult{ExitCode: 2, Stdout: event.Bytes()}
	}
}

func failureExecutor(t *testing.T, classification string) ExecuteFunc {
	t.Helper()
	return func(_ context.Context, command Command) CommandResult {
		if command.Args[0] == "analyze" {
			return encodedAnalysisResult(t, capabilityanalysis.ClassificationSupported)
		}
		logicalCommand := append([]string{"gomad"}, command.Args...)
		report, err := qualify.BuildFailure(logicalCommand, 7, 2, nil, qualify.Failure{
			Classification: classification, Message: "runner failed", Iteration: 1,
		})
		if err != nil {
			t.Fatal(err)
		}
		path, err := qualify.Write(command.ArtifactRoot, report)
		if err != nil {
			t.Fatal(err)
		}
		var event bytes.Buffer
		if err := qualify.WriteResultEvent(&event, report, path); err != nil {
			t.Fatal(err)
		}
		return CommandResult{ExitCode: qualify.ExitStatus(classification), Stdout: event.Bytes()}
	}
}

func encodedAnalysisResult(t *testing.T, classification capabilityanalysis.Classification) CommandResult {
	t.Helper()
	boundaryVersion, boundaryDigest := ioprofile.BoundaryManifestIdentity()
	report := capabilityanalysis.Report{
		Schema: capabilityanalysis.Schema, Classification: classification,
		Target: capabilityanalysis.Target{Kind: target.KindGoTest, Source: "pkg", Arguments: []string{"-test.run=^TestScenario$"}, BuildTags: []string{}},
		Toolchain: capabilityanalysis.Toolchain{
			GoVersion: "go1.26.4", BuildKey: strings.Repeat("a", 64), TargetGOOS: "darwin", TargetGOARCH: "arm64",
			BoundaryManifestVersion: boundaryVersion, BoundaryManifestSHA256: boundaryDigest,
		},
		Closure: capabilityanalysis.Closure{
			SHA256: record.HashBytes([]byte("closure")), PackageCount: 1,
			Roots: []target.CapabilityPackageReference{{ImportPath: "example.com/target/pkg", Name: "pkg"}},
		},
		IOProfile: ioprofile.Default().Identity(), Packs: []compatibility.PackEvidence{}, Requirements: []ioprofile.Requirement{}, Blockers: []capabilityanalysis.Blocker{},
	}
	status := 0
	if classification == capabilityanalysis.ClassificationUnsupported {
		status = 1
		report.Blockers = []capabilityanalysis.Blocker{{
			CapabilityFinding: target.CapabilityFinding{
				Kind: target.FindingForbiddenImport, Package: target.CapabilityPackageReference{ImportPath: "golang.org/x/net/internal/socket", Name: "socket"},
				Capability: "uses go:linkname in sys_unix.go", Directives: []string{}, PolicyDisposition: compatibility.DispositionDenied,
				Remediation: compatibility.RemediationRemainUnsupported,
			},
			DependencyPath: []target.CapabilityPackageReference{{ImportPath: "example.com/target/pkg", Name: "pkg"}, {ImportPath: "golang.org/x/net/internal/socket", Name: "socket"}},
		}}
	}
	encoded, err := record.CanonicalJSON(report)
	if err != nil {
		t.Fatal(err)
	}
	return CommandResult{ExitCode: status, Stdout: append(encoded, '\n')}
}
