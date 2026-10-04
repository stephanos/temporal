package pinimpact_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	compatibility "go.temporal.io/server/tools/gomad3/internal/compatibilitypack"
	"go.temporal.io/server/tools/gomad3/upgrade/pinimpact"
)

func authoringSpec(t *testing.T, directory string) pinimpact.Spec {
	t.Helper()
	files := moduleFiles(t, baselineRequirements(t), "")
	return pinimpact.Spec{Root: gomadRoot(t), Baseline: files, Candidate: files, Resolver: goModResolver{}, PacksDirectory: directory}
}

func TestExplicitPackRootOverridesEnvironmentAndRetainsReports(t *testing.T) {
	directory := filepath.Join(gomadRoot(t), "internal", "compatibilitypack", "packs")
	spec := authoringSpec(t, directory)
	baseline, err := pinimpact.Evaluate(context.Background(), spec)
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv(compatibility.ExternalPacksEnvironment, filepath.Join(t.TempDir(), "absent-environment-packs"))
	actual, err := pinimpact.Evaluate(context.Background(), spec)
	if err != nil {
		t.Fatal(err)
	}
	wantBytes, err := json.Marshal(baseline)
	if err != nil {
		t.Fatal(err)
	}
	actualBytes, err := json.Marshal(actual)
	if err != nil {
		t.Fatal(err)
	}
	if string(actualBytes) != string(wantBytes) {
		t.Fatalf("explicit root followed environment:\n%s\nwant\n%s", actualBytes, wantBytes)
	}
	defaultSpec := spec
	defaultSpec.PacksDirectory = ""
	defaultReport, err := pinimpact.Evaluate(context.Background(), defaultSpec)
	if err != nil {
		t.Fatal(err)
	}
	defaultBytes, err := json.Marshal(defaultReport)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(defaultBytes), "load compatibility packs:") {
		t.Fatalf("default did not retain pack-error report: %s", defaultBytes)
	}
}

func TestPackLoadRemainsAfterModuleValidationAndErrorsStayInReport(t *testing.T) {
	directory := t.TempDir()
	if err := os.WriteFile(filepath.Join(directory, "bad.json"), []byte("not-json"), 0600); err != nil {
		t.Fatal(err)
	}
	spec := authoringSpec(t, directory)
	report, err := pinimpact.Evaluate(context.Background(), spec)
	if err != nil {
		t.Fatalf("pack error escaped report: %v", err)
	}
	encoded, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(encoded), "load compatibility packs:") {
		t.Fatalf("missing pack report: %s", encoded)
	}
	spec.Baseline.GoMod = []byte("not a module")
	if _, err := pinimpact.Evaluate(context.Background(), spec); err == nil || !pinimpact.IsInputError(err) || !strings.Contains(err.Error(), "baseline") {
		t.Fatalf("baseline error precedence: %v", err)
	}
	spec = authoringSpec(t, filepath.Join(t.TempDir(), "missing-authoring-packs"))
	spec.Candidate.GoMod = []byte("not a module")
	if _, err := pinimpact.Evaluate(context.Background(), spec); err == nil || !pinimpact.IsInputError(err) || !strings.Contains(err.Error(), "candidate") {
		t.Fatalf("candidate error precedence: %v", err)
	}
}
