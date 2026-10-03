package preparation

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"go.temporal.io/server/tools/gomad3/deterministicio"
	"go.temporal.io/server/tools/gomad3/record"
	"go.temporal.io/server/tools/gomad3/target"
)

type testPreparer func(context.Context, target.Spec) (target.Prepared, error)

func (prepare testPreparer) Prepare(ctx context.Context, spec target.Spec) (target.Prepared, error) {
	return prepare(ctx, spec)
}

func validPrepared() target.Prepared {
	contract := deterministicio.Default().TargetContract()
	return target.Prepared{
		Kind: target.KindGoRun, Source: ".", Argv: []string{"gomad3-target"},
		GoVersion: contract.GoVersion, TargetGOOS: contract.GOOS, TargetGOARCH: contract.GOARCH,
	}
}

func TestPrepareCustomPreparerSkipsAdaptersAndValidates(t *testing.T) {
	want := validPrepared()
	want.Adapters = []record.TargetAdapter{}
	provided := want
	provided.Adapters = []record.TargetAdapter{{Module: "unselected"}}
	root := t.TempDir()
	workspace := filepath.Join(root, ".io-adapter")
	if err := os.Mkdir(workspace, 0o700); err != nil {
		t.Fatal(err)
	}
	got, err := Prepare(t.Context(), Request{
		Target:   target.Spec{Kind: target.KindGoRun, Source: ".", PreparationRoot: root},
		Preparer: testPreparer(func(context.Context, target.Spec) (target.Prepared, error) { return provided, nil }),
	})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("prepared = %#v, want %#v", got, want)
	}
	if _, err := os.Stat(workspace); err != nil {
		t.Fatalf("caller workspace was changed: %v", err)
	}
}

func TestPrepareKeepsTargetFailureStageAndIdentity(t *testing.T) {
	want := errors.New("preparer failed")
	_, err := Prepare(t.Context(), Request{
		Target:   target.Spec{Kind: target.KindGoRun, Source: ".", PreparationRoot: t.TempDir()},
		Preparer: testPreparer(func(context.Context, target.Spec) (target.Prepared, error) { return target.Prepared{}, want }),
	})
	if !errors.Is(err, want) || StageOf(err) != StageTarget || err.Error() != want.Error() {
		t.Fatalf("Prepare() error = %v, stage = %s", err, StageOf(err))
	}
}

func testModuleRoot(t *testing.T) string {
	t.Helper()
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	return root
}

func writeTestFile(t *testing.T, path, contents string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestPrepareExternalModuleWithLocalReplacement(t *testing.T) {
	module := t.TempDir()
	root := testModuleRoot(t)
	writeTestFile(t, filepath.Join(module, "go.mod"), "module example.com/external\n\ngo 1.27.1\n\nrequire go.temporal.io/server/tools/gomad3 v0.0.0\n\nreplace go.temporal.io/server/tools/gomad3 => "+root+"\n")
	writeTestFile(t, filepath.Join(module, "main.go"), "package main\n\nimport _ \"go.temporal.io/server/tools/gomad3/record\"\n\nfunc main() {}\n")
	prepared, err := Prepare(t.Context(), Request{Target: target.Spec{
		Kind: target.KindGoRun, Source: ".", WorkingDir: module,
		PreparationRoot: t.TempDir(), ToolchainRoot: filepath.Join(root, ".toolchain"),
	}})
	if err != nil {
		t.Fatal(err)
	}
	if err := prepared.Verify(); err != nil {
		t.Fatal(err)
	}
	if prepared.BuildInfo.MainModule != "example.com/external" || len(prepared.Adapters) != 0 {
		t.Fatalf("prepared = %#v", prepared)
	}
}

func TestPrepareIndependentAdapterTargetsHaveStableIdentityAndCleanWorkspaces(t *testing.T) {
	root := testModuleRoot(t)
	module := filepath.Join(root, "deterministicio", "testdata", "sprig")
	var first target.Prepared
	for index := 0; index < 2; index++ {
		preparationRoot := t.TempDir()
		prepared, err := Prepare(t.Context(), Request{Target: target.Spec{
			Kind: target.KindGoTest, Source: ".", WorkingDir: module,
			PreparationRoot: preparationRoot, ToolchainRoot: filepath.Join(root, ".toolchain"),
		}})
		if err != nil {
			t.Fatal(err)
		}
		if err := prepared.Verify(); err != nil {
			t.Fatal(err)
		}
		entries, err := os.ReadDir(preparationRoot)
		if err != nil {
			t.Fatal(err)
		}
		for _, entry := range entries {
			if strings.HasPrefix(entry.Name(), ".adapter-work-") {
				t.Fatalf("adapter workspace retained: %s", entry.Name())
			}
		}
		if len(prepared.Adapters) != 1 || prepared.Adapters[0].Module != "github.com/Masterminds/sprig/v3" {
			t.Fatalf("prepared adapters = %#v", prepared.Adapters)
		}
		prepared.Path = "<prepared-path>"
		if index == 0 {
			first = prepared
		} else if !reflect.DeepEqual(prepared, first) || !reflect.DeepEqual(prepared.RecordTarget(), first.RecordTarget()) {
			t.Fatalf("independent preparations differ: %#v %#v", first, prepared)
		}
	}
}

func TestPrepareRejectsAdapterSumsAndReplacementConflicts(t *testing.T) {
	root := testModuleRoot(t)
	fixture := filepath.Join(root, "deterministicio", "testdata", "sprig")
	moduleFile, err := os.ReadFile(filepath.Join(fixture, "go.mod"))
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name    string
		modFile string
	}{
		{name: "missing sum", modFile: string(moduleFile)},
		{name: "replacement conflict", modFile: string(moduleFile) + "\nreplace github.com/Masterminds/sprig/v3 => ./other\n"},
	} {
		t.Run(test.name, func(t *testing.T) {
			module := t.TempDir()
			writeTestFile(t, filepath.Join(module, "go.mod"), test.modFile)
			if test.name == "replacement conflict" {
				sums, err := os.ReadFile(filepath.Join(fixture, "go.sum"))
				if err != nil {
					t.Fatal(err)
				}
				writeTestFile(t, filepath.Join(module, "go.sum"), string(sums))
			}
			_, err := Prepare(t.Context(), Request{Target: target.Spec{
				Kind: target.KindGoTest, Source: ".", WorkingDir: module,
				PreparationRoot: t.TempDir(), ToolchainRoot: filepath.Join(root, ".toolchain"),
			}})
			if StageOf(err) != StageAdapters || !deterministicio.IsInvalidBuildAdapterConfiguration(err) {
				t.Fatalf("Prepare() error = %v, stage = %s", err, StageOf(err))
			}
		})
	}
}

func TestPrepareReportsCleanupFailureAndKeepsPrimaryFailure(t *testing.T) {
	root := testModuleRoot(t)
	fixture := filepath.Join(root, "deterministicio", "testdata", "sprig")
	cleanupFailure := errors.New("cleanup failed")
	for _, test := range []struct {
		name  string
		mode  target.CapabilityMode
		stage Stage
	}{
		{name: "successful target", stage: StageCleanup},
		{name: "failed target", mode: target.CapabilityMode("invalid"), stage: StageTarget},
	} {
		t.Run(test.name, func(t *testing.T) {
			root := t.TempDir()
			removed := false
			prepared, err := prepare(t.Context(), Request{Target: target.Spec{
				Kind: target.KindGoTest, Source: ".", CapabilityMode: test.mode, WorkingDir: fixture,
				PreparationRoot: root, ToolchainRoot: filepath.Join(testModuleRoot(t), ".toolchain"),
			}}, func(path string) error {
				removed = true
				if !strings.HasPrefix(filepath.Base(path), ".adapter-work-") || filepath.Dir(path) != root {
					t.Fatalf("cleanup path = %s", path)
				}
				if err := os.RemoveAll(path); err != nil {
					t.Fatal(err)
				}
				return cleanupFailure
			})
			if !removed || !errors.Is(err, cleanupFailure) || StageOf(err) != test.stage {
				t.Fatalf("Prepare() error = %v, stage = %s, removed = %t", err, StageOf(err), removed)
			}
			if test.stage == StageCleanup && prepared.Kind != target.KindGoTest {
				t.Fatalf("prepared = %#v", prepared)
			}
			if test.stage == StageTarget && !strings.Contains(err.Error(), "unsupported capability mode") {
				t.Fatalf("primary target error lost: %v", err)
			}
		})
	}
}
