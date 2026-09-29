package set

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"testing"

	"go.temporal.io/server/tools/gomad3/choice"
	"go.temporal.io/server/tools/gomad3/deterministicio"
	"go.temporal.io/server/tools/gomad3/internal/canonicaljson"
	"go.temporal.io/server/tools/gomad3/qualification"
	capabilityanalysis "go.temporal.io/server/tools/gomad3/qualification/analysis"
	"go.temporal.io/server/tools/gomad3/record"
	"go.temporal.io/server/tools/gomad3/target"
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

func TestRunKeepsNondeterministicSeedClassification(t *testing.T) {
	root := t.TempDir()
	report, err := Run(context.Background(), Spec{
		ManifestPath: writeManifestWithSeeds(t, root, []uint64{7}, "nondeterministic"),
		GomadPath:    filepath.Join(root, "gomad"), WorkingDir: root,
		ArtifactRoot: filepath.Join(root, "artifacts"), OutputPath: filepath.Join(root, "set-report.json"),
		Execute: failureExecutor(t, "nondeterministic"),
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	workload := report.Workloads[0]
	if !report.ExpectationsMet || report.Failed != 1 || report.InfrastructureErrors != 0 || workload.Classification != "nondeterministic" || !workload.ExpectationMet {
		t.Fatalf("report = %#v", report)
	}
	if len(workload.Seeds) != 1 || workload.Seeds[0].Seed != 7 || workload.Seeds[0].Classification != "nondeterministic" || workload.Seeds[0].ReplayMatch || workload.Seeds[0].ChoiceReplayExact {
		t.Fatalf("seed evidence = %#v", workload.Seeds)
	}
}

func TestProjectSeedReportDoesNotClaimExactReplayForDivergedReplay(t *testing.T) {
	report := qualification.QualificationReport{Seed: 7, EvidenceDigest: record.HashBytes([]byte("evidence")), Executions: []qualification.QualificationExecutionReport{
		{Replay: &qualification.QualificationReplay{Attempted: true, Divergence: "stderr.full_sha256", ChoiceReplayStatus: qualification.ChoiceReplayExact}},
		{Replay: &qualification.QualificationReplay{Attempted: true, Match: true, ChoiceReplayStatus: qualification.ChoiceReplayExact}},
	}}
	seed, err := projectSeedReport(report, "replay_divergence", 7, Workload{ChoiceBytes: 1, ReplaySuccesses: true}, capabilityanalysis.Report{})
	if err != nil {
		t.Fatal(err)
	}
	if !seed.Replayed || seed.ReplayMatch || seed.ChoiceReplayExact || seed.ReplayDivergence != "stderr.full_sha256" {
		t.Fatalf("seed evidence = %#v", seed)
	}
}

func TestProjectSeedReportDoesNotClaimExactReplayWithoutOneChoiceTape(t *testing.T) {
	report := qualification.QualificationReport{Seed: 7, EvidenceDigest: record.HashBytes([]byte("evidence")), Executions: []qualification.QualificationExecutionReport{
		{Replay: &qualification.QualificationReplay{Attempted: true, Match: true, ChoiceReplayStatus: qualification.ChoiceReplayExact}},
		{Replay: &qualification.QualificationReplay{Attempted: true, Match: true, ChoiceReplayStatus: qualification.ChoiceReplayExact}},
	}}
	seed, err := projectSeedReport(report, "nondeterministic", 7, Workload{ChoiceBytes: 1, ReplaySuccesses: true}, capabilityanalysis.Report{})
	if err != nil {
		t.Fatal(err)
	}
	if !seed.Replayed || !seed.ReplayMatch || seed.ChoiceReplayExact {
		t.Fatalf("seed evidence = %#v", seed)
	}
}

func TestRunMeetsUnrepeatableExpectationForEitherOutcome(t *testing.T) {
	for _, classification := range []string{"nondeterministic", "replay_divergence"} {
		t.Run(classification, func(t *testing.T) {
			root := t.TempDir()
			report, err := Run(context.Background(), Spec{
				ManifestPath: writeManifestWithSeeds(t, root, []uint64{7}, "unrepeatable"),
				GomadPath:    filepath.Join(root, "gomad"), WorkingDir: root,
				ArtifactRoot: filepath.Join(root, "artifacts"), OutputPath: filepath.Join(root, "set-report.json"),
				Execute: failureExecutor(t, classification),
			})
			if err != nil {
				t.Fatalf("Run() error = %v", err)
			}
			workload := report.Workloads[0]
			if !report.ExpectationsMet || report.Failed != 1 || workload.Classification != classification || !workload.ExpectationMet {
				t.Fatalf("report = %#v", report)
			}
		})
	}
}

func TestUnrepeatableExpectationRejectsQualifiedSeeds(t *testing.T) {
	expected := WorkloadExpectation{Classification: "unrepeatable"}
	qualified := WorkloadReport{Classification: "qualified", Seeds: []SeedReport{{Seed: 7, Classification: "qualified"}}}
	if matchesSupportedExpectation(expected, qualified) {
		t.Fatal("unrepeatable expectation accepted a qualified workload")
	}
	mixed := WorkloadReport{Classification: "nondeterministic", Seeds: []SeedReport{{Seed: 7, Classification: "replay_divergence"}, {Seed: 11, Classification: "nondeterministic"}}}
	if !matchesSupportedExpectation(expected, mixed) {
		t.Fatal("unrepeatable expectation rejected mixed unrepeatable seeds")
	}
}

func TestIntermittentExpectationAcceptsQualifiedAndUnrepeatableSeeds(t *testing.T) {
	expected := WorkloadExpectation{Classification: "intermittent"}
	for name, workload := range map[string]WorkloadReport{
		"qualified": {Classification: "qualified", Seeds: []SeedReport{{Seed: 7, Classification: "qualified"}, {Seed: 11, Classification: "qualified"}}},
		"mixed":     {Classification: "nondeterministic", Seeds: []SeedReport{{Seed: 7, Classification: "qualified"}, {Seed: 11, Classification: "nondeterministic"}}},
		"diverged":  {Classification: "replay_divergence", Seeds: []SeedReport{{Seed: 7, Classification: "replay_divergence"}}},
	} {
		if !matchesSupportedExpectation(expected, workload) {
			t.Fatalf("intermittent expectation rejected %s workload", name)
		}
	}
	failed := WorkloadReport{Classification: "target_failure", Seeds: []SeedReport{{Seed: 7, Classification: "target_failure"}}}
	if matchesSupportedExpectation(expected, failed) {
		t.Fatal("intermittent expectation accepted a target failure")
	}
}

func TestRunMeetsIntermittentExpectationForQualifiedAndUnrepeatableOutcomes(t *testing.T) {
	for _, classification := range []string{"nondeterministic", "replay_divergence"} {
		t.Run(classification, func(t *testing.T) {
			root := t.TempDir()
			report, err := Run(context.Background(), Spec{
				ManifestPath: writeManifestWithSeeds(t, root, []uint64{7}, "intermittent"),
				GomadPath:    filepath.Join(root, "gomad"), WorkingDir: root,
				ArtifactRoot: filepath.Join(root, "artifacts"), OutputPath: filepath.Join(root, "set-report.json"),
				Execute: failureExecutor(t, classification),
			})
			if err != nil {
				t.Fatalf("Run() error = %v", err)
			}
			workload := report.Workloads[0]
			if !report.ExpectationsMet || report.Failed != 1 || workload.Classification != classification || !workload.ExpectationMet {
				t.Fatalf("report = %#v", report)
			}
		})
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
		t.Fatal("LoadManifest() accepted duplicate workload names")
	}
}

func TestLoadManifestRejectsUnknownRequiredProbes(t *testing.T) {
	root := t.TempDir()
	path := writeManifest(t, root, "qualified")
	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var manifest map[string]any
	if err := json.Unmarshal(contents, &manifest); err != nil {
		t.Fatal(err)
	}
	suite := manifest["suites"].([]any)[0].(map[string]any)
	for _, test := range []struct {
		probes  []string
		wantErr string
	}{
		{probes: []string{"stdlib.os.getwd", "stdlib.os.openfile"}},
		{probes: []string{"stdlib.os.getwdx", "stdlib.os.openfile"}, wantErr: `unknown required semantic probe "stdlib.os.getwdx"`},
	} {
		suite["required_probes"] = test.probes
		contents, err = json.Marshal(manifest)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, contents, 0o600); err != nil {
			t.Fatal(err)
		}
		_, err := LoadManifest(path)
		if test.wantErr == "" && err != nil || test.wantErr != "" && (err == nil || !strings.Contains(err.Error(), test.wantErr)) {
			t.Fatalf("LoadManifest(%v) error = %v, want %q", test.probes, err, test.wantErr)
		}
	}
}

func TestLoadManifestRequiresFindingOnFailureExpectations(t *testing.T) {
	root := t.TempDir()
	path := writeManifest(t, root, "qualified")
	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var manifest map[string]any
	if err := json.Unmarshal(contents, &manifest); err != nil {
		t.Fatal(err)
	}
	suite := manifest["suites"].([]any)[0].(map[string]any)
	boundary := map[string]any{"classification": "unsupported_target", "import_path": "example.com/host", "capability": "foreign:assembly:host.s"}
	for name, test := range map[string]struct {
		expectation map[string]any
		wantErr     string
	}{
		"intermittent with finding":    {expectation: map[string]any{"classification": "intermittent", "finding": "GOMAD_MILESTONES.md#f6-a-package-level-functional-slice"}},
		"intermittent without finding": {expectation: map[string]any{"classification": "intermittent"}, wantErr: "intermittent expectation requires a finding identity"},
		"target failure blank finding": {expectation: map[string]any{"classification": "target_failure", "finding": " "}, wantErr: "target_failure expectation requires a finding identity"},
		"qualified with finding":       {expectation: map[string]any{"classification": "qualified", "finding": "F6"}, wantErr: "qualified expectation cannot include an unsupported boundary or finding"},
		"unsupported with finding":     {expectation: map[string]any{"classification": "unsupported_target", "import_path": "example.com/host", "capability": "foreign:assembly:host.s", "finding": "F6"}, wantErr: "unsupported expectation names its boundary, not a finding"},
		"unsupported without finding":  {expectation: boundary},
	} {
		t.Run(name, func(t *testing.T) {
			suite["expectation"] = test.expectation
			contents, err := json.Marshal(manifest)
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(path, contents, 0o600); err != nil {
				t.Fatal(err)
			}
			_, err = LoadManifest(path)
			if test.wantErr == "" && err != nil || test.wantErr != "" && (err == nil || !strings.Contains(err.Error(), test.wantErr)) {
				t.Fatalf("LoadManifest(%v) error = %v, want %q", test.expectation, err, test.wantErr)
			}
		})
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
		"run_timeout": "30s", "overall_timeout": "2m", "terminate_grace": "2s", "output_bytes": 1024, "world_transition_bytes": 2048,
		"suites": []any{
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

func TestAnalysisCommandBoundsAnalysisByWorkloadBudget(t *testing.T) {
	manifest := Manifest{RunTimeout: "2m", OverallTimeout: "5m", TerminateGrace: "1s"}
	for _, test := range []struct {
		name     string
		override string
		want     string
	}{
		{name: "manifest", want: "--timeout=5m0s"},
		{name: "workload override", override: "20m", want: "--timeout=20m0s"},
		{name: "analyzer maximum", override: "45m", want: "--timeout=30m0s"},
	} {
		t.Run(test.name, func(t *testing.T) {
			command := analysisCommand(Spec{}, manifest, Workload{CapabilityMode: target.CapabilityModeGuarded, Package: "./p", Test: "TestP", OverallTimeout: test.override})
			if !slices.Contains(command.Args, test.want) {
				t.Fatalf("analysis arguments = %v, want %s", command.Args, test.want)
			}
		})
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
	} else if classification != "qualified" {
		expectation["finding"] = "fixture-finding"
	}
	manifest := map[string]any{
		"schema": ManifestSchema, "name": "test-set", "description": "portable fixture", "module": "example.com/target",
		"seeds": seeds, "repeat": 2, "run_timeout": "30s", "overall_timeout": "2m", "terminate_grace": "2s",
		"output_bytes": 1024, "world_transition_bytes": 2048,
		"suites": []any{map[string]any{
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

func TestLoadManifestResolvesPlatformExpectationsForTheHost(t *testing.T) {
	root := t.TempDir()
	path := writeManifest(t, root, "qualified")
	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var manifest map[string]any
	if err := json.Unmarshal(contents, &manifest); err != nil {
		t.Fatal(err)
	}
	host := runtime.GOOS + "/" + runtime.GOARCH
	suite := manifest["suites"].([]any)[0].(map[string]any)
	suite["platform_expectations"] = map[string]any{
		host:         map[string]any{"classification": "unsupported_target", "import_path": "example.com/host", "capability": "foreign:assembly:host.s"},
		"plan9/mips": map[string]any{"classification": "target_failure", "finding": "fixture-finding"},
	}
	write := func() {
		contents, err = json.Marshal(manifest)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, contents, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	write()
	loaded, err := LoadManifest(path)
	if err != nil {
		t.Fatal(err)
	}
	workload := loaded.Suites[0]
	if got := workload.expectationFor(host); got.Classification != "unsupported_target" || got.ImportPath != "example.com/host" {
		t.Fatalf("host expectation = %#v", got)
	}
	if got := workload.expectationFor("darwin/arm64"); got != workload.Expectation && runtime.GOOS+"/"+runtime.GOARCH != "darwin/arm64" {
		t.Fatalf("other platform expectation = %#v, want the default", got)
	}
	suite["platform_expectations"] = map[string]any{"linux": map[string]any{"classification": "qualified"}}
	write()
	if _, err := LoadManifest(path); err == nil || !strings.Contains(err.Error(), "platform expectation key is invalid") {
		t.Fatalf("LoadManifest() error = %v", err)
	}
	suite["platform_expectations"] = map[string]any{host: map[string]any{"classification": "unsupported_target"}}
	write()
	if _, err := LoadManifest(path); err == nil || !strings.Contains(err.Error(), "requires exact import and capability") {
		t.Fatalf("LoadManifest() error = %v", err)
	}
}

func TestShardSelectsOrdinalModuloWorkloadsOfTheManifest(t *testing.T) {
	root := t.TempDir()
	manifest, err := LoadManifest(writeManifestWithSuiteIDs(t, root, "unsupported_target", "case-a", "case-b", "case-c"))
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name      string
		shard     Shard
		want      []string
		wantError string
	}{
		{name: "whole manifest", shard: Shard{}, want: []string{"case-a", "case-b", "case-c"}},
		{name: "single shard", shard: Shard{Index: 0, Count: 1}, want: []string{"case-a", "case-b", "case-c"}},
		{name: "first of two", shard: Shard{Index: 0, Count: 2}, want: []string{"case-a", "case-c"}},
		{name: "second of two", shard: Shard{Index: 1, Count: 2}, want: []string{"case-b"}},
		{name: "index past count", shard: Shard{Index: 3, Count: 3}, wantError: "want zero-based INDEX/COUNT"},
		{name: "index without count", shard: Shard{Index: 1}, wantError: "want zero-based INDEX/COUNT"},
		{name: "count past manifest", shard: Shard{Index: 0, Count: 4}, wantError: "exceeds the manifest's 3 workloads"},
	} {
		t.Run(test.name, func(t *testing.T) {
			selected, err := test.shard.Select(manifest)
			if test.wantError != "" {
				if err == nil || !strings.Contains(err.Error(), test.wantError) {
					t.Fatalf("Select() error = %v, want %q", err, test.wantError)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			got := make([]string, len(selected))
			for index, workload := range selected {
				got[index] = workload.ID
			}
			if !slices.Equal(got, test.want) {
				t.Fatalf("Select() = %v, want %v", got, test.want)
			}
		})
	}
}

func TestRunShardReportsOnlyItsWorkloadsUnderTheWholeManifestDigest(t *testing.T) {
	root := t.TempDir()
	manifestPath := writeManifestWithSuiteIDs(t, root, "unsupported_target", "case-a", "case-b", "case-c")
	whole, err := Run(context.Background(), Spec{
		ManifestPath: manifestPath, GomadPath: filepath.Join(root, "gomad"), WorkingDir: root,
		ArtifactRoot: filepath.Join(root, "artifacts"), OutputPath: filepath.Join(root, "whole.json"), Execute: expectedBoundaryExecutor(t),
	})
	if err != nil {
		t.Fatal(err)
	}
	shard, err := Run(context.Background(), Spec{
		ManifestPath: manifestPath, GomadPath: filepath.Join(root, "gomad"), WorkingDir: root, Shard: Shard{Index: 1, Count: 2},
		ArtifactRoot: filepath.Join(root, "artifacts"), OutputPath: filepath.Join(root, "shard.json"), Execute: expectedBoundaryExecutor(t),
	})
	if err != nil {
		t.Fatal(err)
	}
	if shard.Selected != 1 || len(shard.Workloads) != 1 || shard.Workloads[0].ID != "case-b" || shard.Unsupported != 1 || !shard.ExpectationsMet || shard.ManifestSHA256 != whole.ManifestSHA256 {
		t.Fatalf("shard report = %#v", shard)
	}
	if _, err := OpenReport(filepath.Join(root, "shard.json")); err != nil {
		t.Fatal(err)
	}
	_, err = Run(context.Background(), Spec{
		ManifestPath: manifestPath, GomadPath: filepath.Join(root, "gomad"), WorkingDir: root, Shard: Shard{Index: 0, Count: 4},
		ArtifactRoot: filepath.Join(root, "artifacts"), OutputPath: filepath.Join(root, "empty.json"), Execute: expectedBoundaryExecutor(t),
	})
	if err == nil || !strings.Contains(err.Error(), "exceeds the manifest's 3 workloads") {
		t.Fatalf("Run() with an oversized shard count error = %v", err)
	}
}

func TestMergeCombinesShardReportsIntoTheWholeManifestReport(t *testing.T) {
	root := t.TempDir()
	manifestPath := writeManifestWithSuiteIDs(t, root, "unsupported_target", "case-a", "case-b", "case-c")
	whole, err := Run(context.Background(), Spec{
		ManifestPath: manifestPath, GomadPath: filepath.Join(root, "gomad"), WorkingDir: root,
		ArtifactRoot: filepath.Join(root, "artifacts"), OutputPath: filepath.Join(root, "whole.json"), Execute: expectedBoundaryExecutor(t),
	})
	if err != nil {
		t.Fatal(err)
	}
	shards := runShards(t, root, root, manifestPath, 2, expectedBoundaryExecutor(t), false)
	output := filepath.Join(root, "merged", "report.json")
	merged, err := Merge(context.Background(), MergeSpec{ManifestPath: manifestPath, ShardReports: []string{shards[1], shards[0]}, OutputPath: output})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(merged, whole) {
		t.Fatalf("merged report = %#v, want the whole run's %#v", merged, whole)
	}
	opened, err := OpenReport(output)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(opened, whole) {
		t.Fatalf("opened merged report = %#v, want the whole run's %#v", opened, whole)
	}
}

func TestMergeRejectsShardsThatDoNotCoverTheManifestExactlyOnce(t *testing.T) {
	root := t.TempDir()
	manifestPath := writeManifestWithSuiteIDs(t, root, "unsupported_target", "case-a", "case-b", "case-c")
	shards := runShards(t, root, root, manifestPath, 2, expectedBoundaryExecutor(t), false)
	prunedRoot := filepath.Join(root, "pruned")
	if err := os.MkdirAll(prunedRoot, 0o700); err != nil {
		t.Fatal(err)
	}
	pruned := runShards(t, root, prunedRoot, manifestPath, 2, expectedBoundaryExecutor(t), true)
	otherRoot := filepath.Join(root, "other")
	if err := os.MkdirAll(otherRoot, 0o700); err != nil {
		t.Fatal(err)
	}
	other := runShards(t, otherRoot, otherRoot, writeManifestWithSuiteIDs(t, otherRoot, "unsupported_target", "case-a", "case-b", "case-c", "case-d"), 2, expectedBoundaryExecutor(t), false)
	for _, test := range []struct {
		name      string
		reports   []string
		wantError string
	}{
		{name: "missing shard", reports: []string{shards[0]}, wantError: "omit 1 manifest workloads: case-b"},
		{name: "repeated shard", reports: []string{shards[0], shards[1], shards[0]}, wantError: "repeats workload case-a"},
		{name: "another manifest", reports: []string{shards[0], other[1]}, wantError: "produced from another manifest"},
		{name: "another run configuration", reports: []string{shards[0], pruned[1]}, wantError: "another run configuration"},
		{name: "missing report", reports: []string{shards[0], filepath.Join(root, "absent.json")}, wantError: "absent.json"},
	} {
		t.Run(test.name, func(t *testing.T) {
			output := filepath.Join(root, test.name+".json")
			_, err := Merge(context.Background(), MergeSpec{ManifestPath: manifestPath, ShardReports: test.reports, OutputPath: output})
			if err == nil || !IsInvalidReport(err) || !strings.Contains(err.Error(), test.wantError) {
				t.Fatalf("Merge() error = %v, want invalid input containing %q", err, test.wantError)
			}
			if _, statErr := os.Stat(output); !errors.Is(statErr, os.ErrNotExist) {
				t.Fatalf("Merge() wrote %s despite invalid input: %v", output, statErr)
			}
		})
	}
}

func TestMergeRetainsShardExpectationMismatches(t *testing.T) {
	root := t.TempDir()
	manifestPath := writeManifestWithSuiteIDs(t, root, "qualified", "case-a", "case-b", "case-c")
	shards := runShards(t, root, root, manifestPath, 3, expectedBoundaryExecutor(t), false)
	output := filepath.Join(root, "merged.json")
	merged, err := Merge(context.Background(), MergeSpec{ManifestPath: manifestPath, ShardReports: shards, OutputPath: output})
	var mismatch *ExpectationError
	if !errors.As(err, &mismatch) || !slices.Equal(mismatch.Workloads, []string{"case-a", "case-b", "case-c"}) || merged.ExpectationsMet || merged.Completed != 3 || merged.Unsupported != 3 {
		t.Fatalf("Merge() = %#v, error = %v", merged, err)
	}
	if _, err := OpenReport(output); err != nil {
		t.Fatal(err)
	}
}

// runShards runs every shard of the manifest against the target module in
// workingDir and returns the shard reports, written under outputDir, in shard
// order; a shard that fails its expectations still publishes.
func runShards(t *testing.T, workingDir, outputDir, manifestPath string, count uint64, execute ExecuteFunc, prune bool) []string {
	t.Helper()
	reports := make([]string, count)
	for index := range count {
		reports[index] = filepath.Join(outputDir, "shard-"+strconv.FormatUint(index, 10)+".json")
		_, err := Run(context.Background(), Spec{
			ManifestPath: manifestPath, GomadPath: filepath.Join(workingDir, "gomad"), WorkingDir: workingDir, Shard: Shard{Index: index, Count: count},
			ArtifactRoot: filepath.Join(outputDir, "artifacts"), OutputPath: reports[index], PruneQualifiedArtifacts: prune, Execute: execute,
		})
		var mismatch *ExpectationError
		if err != nil && !errors.As(err, &mismatch) {
			t.Fatal(err)
		}
	}
	return reports
}

func writeManifestWithSuiteIDs(t *testing.T, root, classification string, ids ...string) string {
	t.Helper()
	path := writeManifest(t, root, classification)
	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var manifest map[string]any
	if err := json.Unmarshal(contents, &manifest); err != nil {
		t.Fatal(err)
	}
	template := manifest["suites"].([]any)[0].(map[string]any)
	suites := make([]any, len(ids))
	for index, id := range ids {
		suite := make(map[string]any, len(template))
		for key, value := range template {
			suite[key] = value
		}
		suite["id"] = id
		suite["name"] = "Fixture " + id
		suites[index] = suite
	}
	manifest["suites"] = suites
	contents, err = json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, contents, 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestTestArgumentsAnchorEverySkippedSubtestElement(t *testing.T) {
	for _, test := range []struct {
		name string
		skip []string
		want []string
	}{
		{name: "no skips", want: []string{"-test.run=^TestScenario$"}},
		{name: "nested and sibling subtests", skip: []string{"Group/Case.1", "Other"}, want: []string{"-test.run=^TestScenario$", "-test.skip=^TestScenario$/^Group$/^Case\\.1$|^TestScenario$/^Other$"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := testArguments(Workload{Test: "TestScenario", Skip: test.skip})
			if !slices.Equal(got, test.want) {
				t.Fatalf("testArguments() = %q, want %q", got, test.want)
			}
		})
	}
}

func TestRunPassesSkippedSubtestsToAnalysisAndQualification(t *testing.T) {
	root := t.TempDir()
	manifestPath := writeManifest(t, root, "unsupported_target")
	contents, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	var manifest map[string]any
	if err := json.Unmarshal(contents, &manifest); err != nil {
		t.Fatal(err)
	}
	suite := manifest["suites"].([]any)[0].(map[string]any)
	suite["skip"] = []string{"Flaky"}
	contents, err = json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(manifestPath, contents, 0o600); err != nil {
		t.Fatal(err)
	}
	var observed [][]string
	executor := func(ctx context.Context, command Command) CommandResult {
		observed = append(observed, command.Args)
		return expectedBoundaryExecutor(t)(ctx, command)
	}
	if _, err := Run(context.Background(), Spec{
		ManifestPath: manifestPath, GomadPath: filepath.Join(root, "gomad"), WorkingDir: root,
		ArtifactRoot: filepath.Join(root, "artifacts"), OutputPath: filepath.Join(root, "report.json"), Execute: executor,
	}); err != nil {
		t.Fatal(err)
	}
	if len(observed) != 1 || !slices.Contains(observed[0], "-test.skip=^TestScenario$/^Flaky$") {
		t.Fatalf("analysis arguments = %q, want a -test.skip", observed)
	}
	suite["skip"] = []string{"bad name"}
	contents, err = json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(manifestPath, contents, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadManifest(manifestPath); err == nil || !strings.Contains(err.Error(), "skips an invalid subtest name") {
		t.Fatalf("LoadManifest() error = %v", err)
	}
}
