package set

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

	"go.temporal.io/server/tools/gomadv3/choice"
	"go.temporal.io/server/tools/gomadv3/deterministicio"
	"go.temporal.io/server/tools/gomadv3/internal/canonicaljson"
	"go.temporal.io/server/tools/gomadv3/qualification"
	capabilityanalysis "go.temporal.io/server/tools/gomadv3/qualification/analysis"
	"go.temporal.io/server/tools/gomadv3/record"
	"go.temporal.io/server/tools/gomadv3/target"
)

func TestRunPublishesCheckedExpectedBoundaryEvidence(t *testing.T) {
	root := t.TempDir()
	manifestPath := writeManifest(t, root, "unsupported_target")
	output := filepath.Join(root, "set-report.json")
	report, err := Run(context.Background(), Spec{
		ManifestPath: manifestPath, GomadPath: filepath.Join(root, "gomad"), WorkingDir: root,
		ArtifactRoot: filepath.Join(root, "artifacts"), OutputPath: output, Execute: expectedBoundaryExecutor(t),
	})
	if err != nil {
		t.Fatal(err)
	}
	if !report.ExpectationsMet || report.Completed != 1 || report.Supported != 0 || report.Unsupported != 1 || report.Failed != 0 || report.InfrastructureErrors != 0 || len(report.Workloads) != 1 || !report.Workloads[0].ExpectationMet || report.Workloads[0].Classification != "unsupported_target" || report.Workloads[0].Analysis == nil || len(report.Workloads[0].Blockers) != 1 {
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
	if public["schema"] != ReportSchema || public["expectations_met"] != true || public["supported"] != float64(0) || public["unsupported"] != float64(1) || public["failed"] != float64(0) || public["infrastructure_errors"] != float64(0) {
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
	if opened.ManifestSHA256 == "" || opened.Module.GoModSHA256 == "" || opened.Workloads[0].Analysis == nil {
		t.Fatalf("opened qualification set report = %#v", opened)
	}
	if err := os.Chmod(output, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := OpenReport(output); err == nil {
		t.Fatal("OpenReport() accepted a non-private report")
	}
}

func TestRunRetainsAllEvidenceWhenAnExpectationChanges(t *testing.T) {
	root := t.TempDir()
	output := filepath.Join(root, "set-report.json")
	report, err := Run(context.Background(), Spec{
		ManifestPath: writeManifest(t, root, "qualified"), GomadPath: filepath.Join(root, "gomad"), WorkingDir: root,
		ArtifactRoot: filepath.Join(root, "artifacts"), OutputPath: output, Execute: expectedBoundaryExecutor(t),
	})
	var mismatch *ExpectationError
	if !errors.As(err, &mismatch) || report.ExpectationsMet || report.Completed != 1 || len(report.Workloads) != 1 || report.Workloads[0].ExpectationMet {
		t.Fatalf("qualification set report = %#v, error = %v", report, err)
	}
	if _, err := OpenReport(output); err != nil {
		t.Fatal(err)
	}
}

func TestRunReportRetainsRequestedSeeds(t *testing.T) {
	root := t.TempDir()
	output := filepath.Join(root, "set-report.json")
	_, err := Run(context.Background(), Spec{
		ManifestPath: writeManifestWithSeeds(t, root, []uint64{7, 11}, "unsupported_target"),
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
	_, err := Run(context.Background(), Spec{
		ManifestPath: writeManifestWithSeeds(t, root, []uint64{7}, "unsupported_target"),
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
	var report Report
	if err := canonicaljson.DecodeCanonicalJSON(bytes.TrimSuffix(contents, []byte{'\n'}), &report); err != nil {
		t.Fatal(err)
	}
	report.Unsupported = 0
	report.Supported = 1
	tampered, err := canonicaljson.CanonicalJSON(report)
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

func TestOpenReportRejectsInvalidModuleDigest(t *testing.T) {
	root := t.TempDir()
	output := filepath.Join(root, "set-report.json")
	_, err := Run(context.Background(), Spec{
		ManifestPath: writeManifestWithSeeds(t, root, []uint64{7}, "unsupported_target"),
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
	var report Report
	if err := canonicaljson.DecodeCanonicalJSON(bytes.TrimSuffix(contents, []byte{'\n'}), &report); err != nil {
		t.Fatal(err)
	}
	report.Module.GoModSHA256 = "invalid"
	tampered, err := canonicaljson.CanonicalJSON(report)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(output, append(tampered, '\n'), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := OpenReport(output); err == nil {
		t.Fatal("OpenReport() accepted an invalid module digest")
	}
}

func TestOpenReportRejectsHardLinkedEvidence(t *testing.T) {
	root := t.TempDir()
	output := filepath.Join(root, "set-report.json")
	_, err := Run(context.Background(), Spec{
		ManifestPath: writeManifestWithSeeds(t, root, []uint64{7}, "unsupported_target"),
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
	report, err := Run(context.Background(), Spec{
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
	report, err := Run(context.Background(), Spec{
		ManifestPath: writeManifestWithSeeds(t, root, []uint64{7}, "qualified"),
		GomadPath:    filepath.Join(root, "gomad"), WorkingDir: root,
		ArtifactRoot: filepath.Join(root, "artifacts"), OutputPath: filepath.Join(root, "set-report.json"),
		Execute: failureExecutor(t, "runner_failure"),
	})
	var mismatch *ExpectationError
	if !errors.As(err, &mismatch) {
		t.Fatalf("Run() error = %v", err)
	}
	if len(report.Workloads[0].Seeds) != 1 || report.Workloads[0].Seeds[0].Seed != 7 || report.Workloads[0].Seeds[0].Classification != "runner_failure" {
		t.Fatalf("seed evidence = %#v", report.Workloads[0].Seeds)
	}
}

func TestLoadManifestRejectsDuplicateWorkloadNames(t *testing.T) {
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
	workloads := manifest["workloads"].([]any)
	manifest["workloads"] = append(workloads, workloads[0])
	contents, err = json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, contents, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadManifest(path); err == nil {
		t.Fatal("LoadManifest() accepted duplicate workload names")
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

func TestRunAnalyzesEveryWorkloadBeforeTargetExecution(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/target\n\ngo 1.26.4\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	manifest := map[string]any{
		"schema": ManifestSchema, "name": "test-set", "description": "analysis ordering fixture", "module": "example.com/target", "seeds": []uint64{7}, "repeat": 2,
		"execution_timeout": "30s", "overall_timeout": "2m", "terminate_grace": "2s", "output_bytes": 1024, "world_transition_bytes": 2048,
		"workloads": []any{
			map[string]any{"id": "a-unsupported", "name": "A", "tier": 1, "invariant": "A remains unsupported", "package": "./a", "test": "TestScenario", "capability_mode": "closure", "choice_bytes": 1024, "replay_successes": true, "success_artifact_limit": 2, "success_bytes_limit": 4096, "expectation": map[string]any{"classification": "unsupported_target", "import_path": "golang.org/x/net/internal/socket", "capability": "uses go:linkname in sys_unix.go"}},
			map[string]any{"id": "b-supported", "name": "B", "tier": 1, "invariant": "B remains qualified", "package": "./b", "test": "TestScenario", "capability_mode": "closure", "choice_bytes": 1024, "replay_successes": true, "success_artifact_limit": 2, "success_bytes_limit": 4096, "expectation": map[string]any{"classification": "qualified"}},
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
	report, err := Run(context.Background(), Spec{
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
		"seeds": []uint64{7}, "repeat": 2, "execution_timeout": "30s", "overall_timeout": "2m", "terminate_grace": "2s",
		"output_bytes": 1024, "world_transition_bytes": 2048,
		"workloads": []any{
			map[string]any{"id": "a-supported", "name": "A", "tier": 1, "invariant": "A remains classified", "package": "./a", "test": "TestScenario", "capability_mode": "closure", "choice_bytes": 1024, "replay_successes": true, "success_artifact_limit": 2, "success_bytes_limit": 4096, "expectation": map[string]any{"classification": "qualified"}},
			map[string]any{"id": "b-broken", "name": "B", "tier": 1, "invariant": "B reports analysis failure", "package": "./b", "test": "TestScenario", "capability_mode": "closure", "choice_bytes": 1024, "replay_successes": true, "success_artifact_limit": 2, "success_bytes_limit": 4096, "expectation": map[string]any{"classification": "qualified"}},
			map[string]any{"id": "c-supported", "name": "C", "tier": 1, "invariant": "C remains classified", "package": "./c", "test": "TestScenario", "capability_mode": "closure", "choice_bytes": 1024, "replay_successes": true, "success_artifact_limit": 2, "success_bytes_limit": 4096, "expectation": map[string]any{"classification": "qualified"}},
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
	report, err := Run(context.Background(), Spec{
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
	for index, workload := range report.Workloads {
		if workload.Classification != "runner_failure" {
			t.Fatalf("workload %d classification = %q", index, workload.Classification)
		}
	}
}

func TestAggregateChoiceCoverageRejectsIdentityMismatch(t *testing.T) {
	left := ChoiceCoverage{
		Available: true, Profile: choice.Profile, ImplementationSHA256: record.HashBytes([]byte("left")),
		Limit: 1024, Features: []choice.Feature{},
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
	_, err := Run(ctx, Spec{
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
	return writeManifestWithSeeds(t, root, []uint64{7}, classification)
}

func writeManifestWithSeeds(t *testing.T, root string, seeds []uint64, classification string) string {
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
		"seeds": seeds, "repeat": 2, "execution_timeout": "30s", "overall_timeout": "2m", "terminate_grace": "2s",
		"output_bytes": 1024, "world_transition_bytes": 2048,
		"workloads": []any{map[string]any{
			"id": "fixture-case", "name": "Fixture case", "tier": 1, "invariant": "the fixture remains deterministic",
			"package": "./pkg", "test": "TestScenario", "capability_mode": "closure", "choice_bytes": 4096, "replay_successes": true,
			"success_artifact_limit": 2, "success_bytes_limit": 1048576, "expectation": expectation,
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

func expectedBoundaryExecutor(t *testing.T) ExecuteFunc {
	t.Helper()
	return func(_ context.Context, command Command) CommandResult {
		if command.Args[0] == "analyze" {
			return encodedAnalysisResult(t, capabilityanalysis.ClassificationUnsupported)
		}
		logicalCommand := append([]string{"gomad"}, command.Args...)
		report, err := qualification.BuildQualificationFailure(logicalCommand, 7, 2, nil, qualification.QualificationFailure{
			Classification: "unsupported_target", Message: "unsupported boundary", Iteration: 1,
			ImportPath: "golang.org/x/net/internal/socket", Capability: "uses go:linkname in sys_unix.go",
		})
		if err != nil {
			t.Fatal(err)
		}
		path, err := qualification.WriteQualificationReport(command.ArtifactRoot, report)
		if err != nil {
			t.Fatal(err)
		}
		var event bytes.Buffer
		if err := qualification.WriteResultEvent(&event, report, path); err != nil {
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
		report, err := qualification.BuildQualificationFailure(logicalCommand, 7, 2, nil, qualification.QualificationFailure{
			Classification: classification, Message: "runner failed", Iteration: 1,
		})
		if err != nil {
			t.Fatal(err)
		}
		path, err := qualification.WriteQualificationReport(command.ArtifactRoot, report)
		if err != nil {
			t.Fatal(err)
		}
		var event bytes.Buffer
		if err := qualification.WriteResultEvent(&event, report, path); err != nil {
			t.Fatal(err)
		}
		return CommandResult{ExitCode: qualification.ExitStatus(classification), Stdout: event.Bytes()}
	}
}

func encodedAnalysisResult(t *testing.T, classification capabilityanalysis.Classification) CommandResult {
	t.Helper()
	boundaryVersion, boundaryDigest := deterministicio.BoundaryManifestIdentity()
	report := capabilityanalysis.Report{
		Schema: capabilityanalysis.AnalysisSchema, Classification: classification,
		Target: capabilityanalysis.Target{Kind: target.KindGoTest, Source: "pkg", Arguments: []string{"-test.run=^TestScenario$"}, BuildTags: []string{}, CapabilityMode: target.CapabilityModeClosure},
		Toolchain: capabilityanalysis.Toolchain{
			GoVersion: "go1.26.4", BuildKey: strings.Repeat("a", 64), TargetGOOS: "darwin", TargetGOARCH: "arm64",
			BoundaryManifestVersion: boundaryVersion, BoundaryManifestSHA256: record.SHA256(boundaryDigest),
		},
		Closure: capabilityanalysis.Closure{
			SHA256: record.HashBytes([]byte("closure")), PackageCount: 1,
			Roots: []target.CapabilityPackageReference{{ImportPath: "example.com/target/pkg", Name: "pkg"}},
		},
		IOProfile: deterministicio.Default().Identity(), Packs: []target.CompatibilityPackEvidence{}, Requirements: []deterministicio.Requirement{}, Blockers: []capabilityanalysis.Blocker{}, GuardedBlockers: []capabilityanalysis.Blocker{}, EliminatedBlockers: []capabilityanalysis.Blocker{},
	}
	status := 0
	if classification == capabilityanalysis.ClassificationUnsupported {
		status = 1
		report.Blockers = []capabilityanalysis.Blocker{{
			CapabilityFinding: target.CapabilityFinding{
				Kind: target.FindingForbiddenImport, Package: target.CapabilityPackageReference{ImportPath: "golang.org/x/net/internal/socket", Name: "socket"},
				Capability: "uses go:linkname in sys_unix.go", Directives: []string{}, PolicyDisposition: target.DispositionDenied,
				Remediation: target.RemediationRemainUnsupported,
			},
			DependencyPath: []target.CapabilityPackageReference{{ImportPath: "example.com/target/pkg", Name: "pkg"}, {ImportPath: "golang.org/x/net/internal/socket", Name: "socket"}},
		}}
	}
	encoded, err := canonicaljson.CanonicalJSON(report)
	if err != nil {
		t.Fatal(err)
	}
	return CommandResult{ExitCode: status, Stdout: append(encoded, '\n')}
}
