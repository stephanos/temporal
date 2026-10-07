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

func TestPackRootLoadErrorsArePathFree(t *testing.T) {
	spec := authoringSpec(t, "")
	for _, test := range []struct {
		name     string
		explicit bool
		label    string
	}{
		{name: "explicit", explicit: true, label: "$PacksDirectory"},
		{name: "environment", label: "$GOMAD3_COMPATIBILITY_PACKS"},
	} {
		t.Run(test.name, func(t *testing.T) {
			roots := []string{t.TempDir(), t.TempDir()}
			var canonical, human []string
			for _, root := range roots {
				directory := filepath.Join(root, "packs")
				if err := os.WriteFile(directory, []byte("regular file, not a pack directory"), 0o600); err != nil {
					t.Fatal(err)
				}
				selected := spec
				if test.explicit {
					selected.PacksDirectory = directory
					t.Setenv(compatibility.ExternalPacksEnvironment, filepath.Join(t.TempDir(), "unused-environment-packs"))
				} else {
					t.Setenv(compatibility.ExternalPacksEnvironment, directory)
				}
				report, err := pinimpact.Evaluate(t.Context(), selected)
				if err != nil {
					t.Fatal(err)
				}
				requirePins(t, report, map[string]pinimpact.Status{"pack-rule compatibility packs": pinimpact.StatusUnknown})
				if !report.Invalidated {
					t.Fatal("unloadable packs did not invalidate the candidate")
				}
				encoded, err := pinimpact.Encode(report)
				if err != nil {
					t.Fatal(err)
				}
				var rendered strings.Builder
				if err := pinimpact.Render(&rendered, report); err != nil {
					t.Fatal(err)
				}
				canonical = append(canonical, string(encoded))
				human = append(human, rendered.String())
			}
			for name, outputs := range map[string][]string{"canonical": canonical, "human": human} {
				if outputs[0] != outputs[1] {
					t.Errorf("%s reports depend on the pack root:\n%s\n%s", name, outputs[0], outputs[1])
				}
				for _, output := range outputs {
					for _, root := range roots {
						if strings.Contains(output, root) {
							t.Errorf("%s report contains host root %s", name, root)
						}
					}
					if !strings.Contains(output, test.label) {
						t.Errorf("%s report lacks logical pack root %s", name, test.label)
					}
				}
			}
		})
	}
}

func TestMissingExplicitPackRootSelectsNoPacks(t *testing.T) {
	spec := authoringSpec(t, filepath.Join(t.TempDir(), "missing-packs"))
	t.Setenv(compatibility.ExternalPacksEnvironment, filepath.Join(t.TempDir(), "unused-environment-packs"))
	report, err := pinimpact.Evaluate(t.Context(), spec)
	if err != nil {
		t.Fatal(err)
	}
	if report.Invalidated || len(report.Pins) != 0 {
		t.Fatalf("missing explicit root report = %+v, want no invalidated pins", report)
	}
	for _, summary := range report.Classes {
		if summary.Class == pinimpact.ClassPackRule && summary.Total != 0 {
			t.Fatalf("missing explicit root pack summary = %+v, want no packs", summary)
		}
	}
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
