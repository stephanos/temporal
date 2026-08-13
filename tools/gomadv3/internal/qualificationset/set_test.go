package qualificationset

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"go.temporal.io/server/tools/gomadv3/internal/qualify"
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
	if !report.ExpectationsMet || report.Completed != 1 || report.Supported != 0 || report.Unsupported != 1 || report.Failed != 0 || report.InfrastructureErrors != 0 || len(report.Suites) != 1 || !report.Suites[0].ExpectationMet || report.Suites[0].Classification != "unsupported_target" || report.Suites[0].Report.Failure == nil {
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
	if public["schema"] != "gomadv3.qualification-set-report/v2" || public["expectations_met"] != true || public["supported"] != float64(0) || public["unsupported"] != float64(1) || public["failed"] != float64(0) || public["infrastructure_errors"] != float64(0) {
		t.Fatalf("public qualification set report = %#v", public)
	}
	if _, found := public["qualified"]; found {
		t.Fatalf("public qualification set report retains ambiguous qualified field: %#v", public)
	}
	opened, err := OpenReport(output)
	if err != nil {
		t.Fatal(err)
	}
	if opened.ManifestSHA256 == "" || opened.Suites[0].ReportSHA256 == "" || len(opened.Suites[0].Command) == 0 {
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

func writeManifest(t *testing.T, root, classification string) string {
	t.Helper()
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

func expectedBoundaryExecutor(t *testing.T) ExecuteFunc {
	t.Helper()
	return func(_ context.Context, command Command) CommandResult {
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
