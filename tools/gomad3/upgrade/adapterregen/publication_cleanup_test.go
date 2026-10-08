package adapterregen

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	"go.temporal.io/server/tools/gomad3/deterministicio"
	"go.temporal.io/server/tools/gomad3/internal/hostfs"
)

func TestRunScratchCleanupBeforePublication(t *testing.T) {
	sentinel := errors.New("staging stopped")
	for _, test := range []struct {
		name    string
		primary error
	}{
		{name: "primary", primary: sentinel},
		{name: "input", primary: &InputError{Err: sentinel}},
		{name: "blocked", primary: &BlockedError{Err: sentinel}},
	} {
		for _, fault := range []bool{false, true} {
			name := test.name + "/control"
			if fault {
				name = test.name + "/permission"
			}
			t.Run(name, func(t *testing.T) {
				fixture, review, temporary := newCleanupFixture(t, fault)
				before := fixture.snapshot()
				spec := fixture.spec(goodVersion, review.Regeneration.ApprovalSHA256)
				spec.afterStage = func() error {
					if fault {
						denyScratchRemoval(t, temporary)
					}
					return test.primary
				}
				result, err := Run(t.Context(), spec)
				checkScratchRemoval(t, temporary, fault)
				requireUnchanged(t, before, fixture.snapshot())
				checkCleanupLock(t, fixture.root)
				if !reflect.DeepEqual(result, Result{}) {
					t.Fatalf("failed staging returned %+v", result)
				}
				if !fault {
					if err != test.primary {
						t.Fatalf("primary error identity changed: %v", err)
					}
					return
				}
				if !errors.Is(err, sentinel) || !errors.Is(err, fs.ErrPermission) {
					t.Fatalf("primary and cleanup causes were not retained: %v", err)
				}
				checkPrimaryFirst(t, err, test.primary)
				checkCleanupCauses(t, err, temporary)
				switch test.primary.(type) {
				case *InputError:
					var input *InputError
					if !errors.As(err, &input) || input != test.primary {
						t.Fatalf("input classification changed: %v", err)
					}
				case *BlockedError:
					var blocked *BlockedError
					if !errors.As(err, &blocked) || blocked != test.primary {
						t.Fatalf("blocked classification changed: %v", err)
					}
				}
			})
		}
	}
}

func TestRunScratchCleanupStageOnly(t *testing.T) {
	for _, fault := range []bool{false, true} {
		name := "control"
		if fault {
			name = "permission"
		}
		t.Run(name, func(t *testing.T) {
			fixture, review, temporary := newCleanupFixture(t, fault)
			before := fixture.snapshot()
			spec := fixture.spec(goodVersion, review.Regeneration.ApprovalSHA256)
			spec.StageOnly = true
			spec.afterStage = func() error {
				if fault {
					denyScratchRemoval(t, temporary)
				}
				return nil
			}
			result, err := Run(t.Context(), spec)
			checkScratchRemoval(t, temporary, fault)
			requireUnchanged(t, before, fixture.snapshot())
			checkCleanupLock(t, fixture.root)
			checkCleanupResult(t, result, review, false)
			if len(result.Warnings) != 0 || len(result.Residual) != 0 {
				t.Fatalf("stage-only returned publication data: %+v", result)
			}
			if !fault {
				if err != nil {
					t.Fatal(err)
				}
				return
			}
			checkCleanupCauses(t, err, temporary)
			joined, ok := err.(interface{ Unwrap() []error })
			if !ok || len(joined.Unwrap()) != 2 {
				t.Fatalf("stage and download failures were not the direct causes: %v", err)
			}
			if _, ok := joined.Unwrap()[0].(*os.PathError); !ok {
				t.Fatalf("sole stage cleanup failure was wrapped: %v", joined.Unwrap()[0])
			}
		})
	}
}

func TestRunScratchCleanupMissingDownloadRoot(t *testing.T) {
	fixture, review, temporary := newCleanupFixture(t, false)
	before := fixture.snapshot()
	spec := fixture.spec(goodVersion, review.Regeneration.ApprovalSHA256)
	spec.StageOnly = true
	var downloadRoot string
	spec.afterStage = func() error {
		entries, err := os.ReadDir(temporary)
		if err != nil {
			return err
		}
		for _, entry := range entries {
			if strings.HasPrefix(entry.Name(), "gomad3-adapter-regenerate-") {
				downloadRoot = filepath.Join(temporary, entry.Name())
				return os.RemoveAll(downloadRoot)
			}
		}
		return errors.New("download scratch root is missing before staging completes")
	}
	result, err := Run(t.Context(), spec)
	checkScratchRemoval(t, temporary, false)
	requireUnchanged(t, before, fixture.snapshot())
	checkCleanupLock(t, fixture.root)
	checkCleanupResult(t, result, review, false)
	pathError, ok := err.(*os.PathError)
	if !ok || !errors.Is(err, fs.ErrNotExist) || pathError.Path != downloadRoot {
		t.Fatalf("sole traversal error was lost or wrapped: %v", err)
	}
}

func TestRunScratchCleanupAfterPublication(t *testing.T) {
	for _, scanFailure := range []bool{false, true} {
		for _, fault := range []bool{false, true} {
			name := "residual/control"
			if scanFailure {
				name = "scan-warning/control"
			}
			if fault {
				name = strings.TrimSuffix(name, "control") + "permission"
			}
			t.Run(name, func(t *testing.T) {
				fixture, review, temporary := newCleanupFixture(t, fault)
				spec := fixture.spec(goodVersion, review.Regeneration.ApprovalSHA256)
				spec.residualScan = func(string, deterministicio.AdapterRegeneration) ([]string, error) {
					if fault {
						denyScratchRemoval(t, temporary)
					}
					if scanFailure {
						return nil, errors.New("scan failed")
					}
					return []string{"remaining.go:7"}, nil
				}
				result, err := Run(t.Context(), spec)
				checkScratchRemoval(t, temporary, fault)
				checkCleanupLock(t, fixture.root)
				if err != nil {
					t.Fatalf("complete publication returned an error: %v", err)
				}
				checkCleanupResult(t, result, review, true)
				if !strings.Contains(fixture.read("deterministicio/sentry_adapter.go"), `"`+goodVersion+`"`) ||
					!strings.Contains(fixture.read("deterministicio/testdata/sentry/go.mod"), sentryModule+" "+goodVersion) ||
					fixture.read(generatedFile) == "initial\n" {
					t.Fatal("published checkout does not hold the approved adapter, fixture and generated output")
				}
				warningIndex := 0
				if scanFailure {
					warningIndex++
					if result.Residual != nil || len(result.Warnings) == 0 || !strings.Contains(result.Warnings[0], "scan failed") {
						t.Fatalf("scan warning was lost: %+v", result)
					}
				} else if !slices.Equal(result.Residual, []string{"remaining.go:7"}) {
					t.Fatalf("residual references changed: %v", result.Residual)
				}
				if fault {
					if len(result.Warnings) != warningIndex+2 ||
						!strings.Contains(result.Warnings[warningIndex], "gomad3-adapter-stage-") ||
						!strings.Contains(result.Warnings[warningIndex+1], "gomad3-adapter-regenerate-") ||
						!strings.Contains(result.Warnings[warningIndex], "permission denied") ||
						!strings.Contains(result.Warnings[warningIndex+1], "permission denied") {
						t.Fatalf("cleanup warning order changed: %v", result.Warnings)
					}
				} else if len(result.Warnings) != warningIndex {
					t.Fatalf("successful cleanup added warnings: %v", result.Warnings)
				}
			})
		}
	}
}

func newCleanupFixture(t *testing.T, fault bool) (*fixture, Result, string) {
	t.Helper()
	fixture := newFixture(t)
	review := fixture.dryRun(goodVersion)
	temporary := t.TempDir()
	t.Cleanup(func() {
		if err := os.Chmod(temporary, 0o700); err != nil {
			t.Fatal(err)
		}
		entries, err := os.ReadDir(temporary)
		if err != nil {
			t.Fatal(err)
		}
		for _, entry := range entries {
			if err := os.RemoveAll(filepath.Join(temporary, entry.Name())); err != nil {
				t.Fatal(err)
			}
		}
	})
	if fault {
		probe := filepath.Join(temporary, "permission-probe")
		if err := os.Mkdir(probe, 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.Chmod(temporary, 0o500); err != nil {
			t.Fatal(err)
		}
		err := os.Remove(probe)
		if restoreErr := os.Chmod(temporary, 0o700); restoreErr != nil {
			t.Fatal(restoreErr)
		}
		if !errors.Is(err, fs.ErrPermission) {
			t.Skipf("scratch unlink permission faults are unsupported (including privileged users): %v", err)
		}
		if err := os.Remove(probe); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("TMPDIR", temporary)
	fixture.env = append(fixture.env, "TMPDIR="+temporary)
	return fixture, review, temporary
}

func denyScratchRemoval(t *testing.T, temporary string) {
	t.Helper()
	entries, err := os.ReadDir(temporary)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 {
		t.Fatalf("scratch trees before cleanup = %v", entries)
	}
	if err := os.Chmod(temporary, 0o500); err != nil {
		t.Fatal(err)
	}
}

func checkScratchRemoval(t *testing.T, temporary string, fault bool) {
	t.Helper()
	entries, err := os.ReadDir(temporary)
	if err != nil {
		t.Fatal(err)
	}
	if !fault {
		if len(entries) != 0 {
			t.Fatalf("successful cleanup retained %v", entries)
		}
		return
	}
	if len(entries) != 2 || !strings.HasPrefix(entries[0].Name(), "gomad3-adapter-regenerate-") || !strings.HasPrefix(entries[1].Name(), "gomad3-adapter-stage-") {
		t.Fatalf("cleanup did not retain both named scratch roots: %v", entries)
	}
	for _, entry := range entries {
		path := filepath.Join(temporary, entry.Name())
		children, err := os.ReadDir(path)
		if err != nil || len(children) != 0 {
			t.Fatalf("scratch traversal did not finish for %s: children=%v error=%v", path, children, err)
		}
		if err := os.Remove(path); !errors.Is(err, fs.ErrPermission) {
			t.Fatalf("scratch root removal did not fail with permission denied: %v", err)
		}
		t.Logf("observed real scratch unlink EACCES: %s", entry.Name())
	}
}

func checkCleanupCauses(t *testing.T, err error, temporary string) {
	t.Helper()
	var pathError *os.PathError
	if !errors.Is(err, fs.ErrPermission) || !errors.As(err, &pathError) {
		t.Fatalf("cleanup permission and PathError causes were lost: %v", err)
	}
	var paths []string
	var visit func(error)
	visit = func(cause error) {
		if pathError, ok := cause.(*os.PathError); ok && errors.Is(pathError, fs.ErrPermission) {
			paths = append(paths, pathError.Path)
			return
		}
		if joined, ok := cause.(interface{ Unwrap() []error }); ok {
			for _, child := range joined.Unwrap() {
				visit(child)
			}
		} else if wrapped, ok := cause.(interface{ Unwrap() error }); ok {
			visit(wrapped.Unwrap())
		}
	}
	visit(err)
	if len(paths) != 2 || filepath.Dir(paths[0]) != temporary || filepath.Dir(paths[1]) != temporary ||
		!strings.HasPrefix(filepath.Base(paths[0]), "gomad3-adapter-stage-") || !strings.HasPrefix(filepath.Base(paths[1]), "gomad3-adapter-regenerate-") {
		t.Fatalf("cleanup causes are not stage then download: %v", paths)
	}
}

func checkPrimaryFirst(t *testing.T, err, primary error) {
	t.Helper()
	for err != primary {
		joined, ok := err.(interface{ Unwrap() []error })
		if !ok || len(joined.Unwrap()) == 0 {
			t.Fatalf("primary error is not first: %v", err)
		}
		err = joined.Unwrap()[0]
	}
}

func checkCleanupLock(t *testing.T, root string) {
	t.Helper()
	lock, err := hostfs.Try(filepath.Join(root, filepath.FromSlash(stateDirectory), "lock"))
	if err != nil {
		t.Fatalf("publication lock was not released: %v", err)
	}
	if err := lock.Release(); err != nil {
		t.Fatal(err)
	}
}

func checkCleanupResult(t *testing.T, result, review Result, applied bool) {
	t.Helper()
	if result.Applied != applied || !reflect.DeepEqual(result.Regeneration, review.Regeneration) ||
		!reflect.DeepEqual(result.Diffs, review.Diffs) || !reflect.DeepEqual(result.StalePacks, review.StalePacks) {
		t.Fatalf("regeneration result changed: %+v", result)
	}
	want := []string{
		"deterministicio/adapter_pin_decisions_test.go",
		"deterministicio/adapter_registry_test.go",
		"deterministicio/sentry_adapter.go",
		"deterministicio/testdata/pebble/go.mod",
		"deterministicio/testdata/pebble/go.sum",
		"deterministicio/testdata/sentry/go.mod",
		"deterministicio/testdata/sentry/go.sum",
		generatedFile,
		"toolchain/version/version.json",
	}
	paths := make([]string, len(result.Staged))
	for index, file := range result.Staged {
		paths[index] = file.Path
		if file.Change != "changed" {
			t.Fatalf("staged file change = %+v", file)
		}
	}
	if !slices.Equal(paths, want) || (applied && !slices.Equal(result.Published, want)) || (!applied && result.Published != nil) {
		t.Fatalf("staged or published files changed: staged=%v published=%v", paths, result.Published)
	}
}
