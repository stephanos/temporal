package main

import (
	"bytes"
	"encoding/json"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

type lintCall struct {
	Tool string
	Dir  string
	Args []string
}

func TestLintOwnershipUsesRetainedClassifications(t *testing.T) {
	t.Parallel()
	repo := &lintRepo{t: t, root: t.TempDir()}
	repo.write("tools/gomad/internal/archcheck/inventory.go", "package archcheck\nvar sourceExclusions = []string{\"internal/backends/wasm/testdata\"}\nvar expectedModules = map[string]bool{\"internal/backends/wasm/testdata/fixture/go.mod\":true}\n")
	policy, err := loadOwnership(repo.root)
	require.NoError(t, err)
	require.Equal(t, ownership{
		fixtures: []string{"internal/backends/wasm/testdata"},
		modules:  map[string]bool{"tools/gomad/internal/backends/wasm/testdata/fixture/go.mod": true},
	}, policy)
}

func TestLintRejectsInvalidClassifications(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct{ name, source, wantError string }{
		{"missing exclusions", "package archcheck\nvar expectedModules = map[string]bool{\"testdata/go.mod\":true}\n", "sourceExclusions is absent"},
		{"missing modules", "package archcheck\nvar sourceExclusions = []string{\"testdata\"}\n", "expectedModules is absent"},
		{"nonliteral exclusions", "package archcheck\nvar sourceExclusions = paths()\nvar expectedModules = map[string]bool{\"testdata/go.mod\":true}\n", "sourceExclusions must remain a literal"},
		{"inactive module", "package archcheck\nvar sourceExclusions = []string{\"testdata\"}\nvar expectedModules = map[string]bool{\"testdata/go.mod\":false}\n", "expectedModules has an inactive classification"},
		{"invalid exclusion", "package archcheck\nvar sourceExclusions = []string{\"../outside\"}\nvar expectedModules = map[string]bool{\"testdata/go.mod\":true}\n", "sourceExclusions has an invalid path"},
		{"duplicate module", "package archcheck\nvar sourceExclusions = []string{\"testdata\"}\nvar expectedModules = map[string]bool{\"testdata/go.mod\":true,\"testdata/go.mod\":true}\n", "is classified twice"},
		{"empty exclusions", "package archcheck\nvar sourceExclusions = []string{}\nvar expectedModules = map[string]bool{\"testdata/go.mod\":true}\n", "need literal sourceExclusions and expectedModules"},
		{"empty modules", "package archcheck\nvar sourceExclusions = []string{\"testdata\"}\nvar expectedModules = map[string]bool{}\n", "need literal sourceExclusions and expectedModules"},
		{"unparsable", "package archcheck\nvar (\n", "read Gomad source classifications"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			repo := &lintRepo{t: t, root: t.TempDir()}
			repo.write("tools/gomad/internal/archcheck/inventory.go", tc.source)
			_, err := loadOwnership(repo.root)
			require.ErrorContains(t, err, tc.wantError)
		})
	}
}

func TestCoveredPackagesMetadata(t *testing.T) {
	for _, tc := range []struct {
		name, metadata, wantError string
		paths, wantPackages       []string
	}{
		{"empty EOF", "", "uncovered host source ordinary/source.go: absent from Go package metadata", []string{"ordinary/source.go"}, nil},
		{"whitespace EOF", " \n\t", "uncovered host source ordinary/source.go: absent from Go package metadata", []string{"ordinary/source.go"}, nil},
		{"malformed trailing JSON", `{"Dir":"$ROOT/ordinary","GoFiles":["source.go"]} !`, "syntax", []string{"ordinary/source.go"}, nil},
		{"truncated trailing JSON", `{"Dir":"$ROOT/ordinary","GoFiles":["source.go"]} {`, "unexpected EOF", []string{"ordinary/source.go"}, nil},
		{"package error before dependencies", `{"Dir":"package-dir","Error":{"Err":"package problem"},"DepsErrors":[{"Err":"first dependency"},{"Err":"second dependency"}]}`, "go package package-dir: package problem", []string{"ordinary/source.go"}, nil},
		{"first dependency error", `{"Dir":"package-dir","DepsErrors":[{"Err":"first dependency"},{"Err":"second dependency"}]}`, "go package package-dir: first dependency", []string{"ordinary/source.go"}, nil},
		{"all source categories and multiple packages", `{"Dir":"$ROOT/ordinary","GoFiles":["source.go"],"CgoFiles":["cgo.go"],"IgnoredGoFiles":["ignored.go"],"TestGoFiles":["source_test.go"],"XTestGoFiles":["external_test.go"]} {"Dir":"$ROOT/aaa","GoFiles":["alpha.go"]}`, "", []string{"ordinary/external_test.go", "ordinary/ignored.go", "ordinary/cgo.go", "aaa/alpha.go", "ordinary/source_test.go", "ordinary/source.go"}, []string{"./aaa", "./ordinary"}},
		{"uncovered source after valid metadata", `{"Dir":"$ROOT/ordinary","GoFiles":["source.go"]}`, "uncovered host source ordinary/missing.go: absent from Go package metadata", []string{"ordinary/source.go", "ordinary/missing.go"}, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			repo := &lintRepo{t: t, root: t.TempDir()}
			var sources []source
			for _, path := range tc.paths {
				repo.write(path, "package sample\n")
				sources = append(sources, source{path: path, module: "."})
			}
			repo.write("bin/go", "#!/bin/sh\nprintf '%s\\n' \"$PWD\" \"$@\" > \"$LINT_TEST_ARGS\"\nprintf '%s' \"$LINT_TEST_METADATA\"\n")
			require.NoError(t, os.Chmod(filepath.Join(repo.root, "bin/go"), 0o700))
			t.Setenv("PATH", filepath.Join(repo.root, "bin")+string(os.PathListSeparator)+os.Getenv("PATH"))
			t.Setenv("LINT_TEST_ARGS", filepath.Join(repo.root, "args"))
			t.Setenv("LINT_TEST_METADATA", strings.ReplaceAll(tc.metadata, "$ROOT", repo.root))
			packages, err := coveredPackages(t.Context(), repo.root, ".", "test_dep", sources)
			switch tc.wantError {
			case "":
				require.NoError(t, err)
			case "syntax":
				var syntaxError *json.SyntaxError
				require.ErrorAs(t, err, &syntaxError)
			case "unexpected EOF":
				require.ErrorIs(t, err, io.ErrUnexpectedEOF)
			default:
				require.EqualError(t, err, tc.wantError)
			}
			require.Equal(t, tc.wantPackages, packages)
			args, err := os.ReadFile(filepath.Join(repo.root, "args"))
			require.NoError(t, err)
			wantArgs := repo.root + "\nlist\n-e\n-mod=readonly\n-json\n-tags\ntest_dep\n"
			if tc.wantPackages != nil {
				wantArgs += "./aaa\n./ordinary\n"
			} else {
				wantArgs += "./ordinary\n"
			}
			require.Equal(t, wantArgs, string(args))
		})
	}
}

func TestLintHostPackages(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct{ target, failure string }{
		{"lint-code-fast", ""},
		{"lint-code-gomad", ""},
		{"lint-code-fast", "golangci"},
		{"lint-code-gomad", "vet"},
	} {
		t.Run(tc.target+tc.failure, func(t *testing.T) {
			t.Parallel()
			repo := newLintRepo(t)
			for _, directory := range []string{"tools/gomad/internal/hostcustom/scheduler", "tools/gomad/internal/hostcustom/transport"} {
				repo.write(directory+"/source.go", "package runner\n")
				repo.write(directory+"/source_test.go", "package runner\nimport \"testing\"\nfunc TestRunner(t *testing.T) {}\n")
			}
			repo.env = append(repo.env, "LINT_TEST_FAIL="+tc.failure, "LINT_TEST_FAIL_PACKAGE=./internal/hostcustom/scheduler")
			output, err := repo.runMake(tc.target)
			if tc.failure == "" {
				require.NoError(t, err, output)
			} else {
				require.Error(t, err, output)
			}
			packages := []string{"./internal/archcheck", "./internal/hostcustom/scheduler", "./internal/hostcustom/transport", "./internal/model/cluster", "./internal/runner"}
			want := []lintCall{{"golangci", "tools/gomad", repo.lintArgs("tools/gomad", packages...)}}
			if tc.failure != "golangci" {
				want = append(want, lintCall{"vet", "tools/gomad", repo.vetArgs(packages...)})
			}
			require.Equal(t, want, repo.calls())
		})
	}
}

func TestLintHostPackageImportFailure(t *testing.T) {
	t.Parallel()
	repo := newLintRepo(t)
	repo.write("tools/gomad/internal/hostcustom/scheduler/source.go", "package runner\nimport _ \"example.invalid/fixture/missing\"\n")
	output, err := repo.runMake("lint-code-gomad")
	require.Error(t, err)
	require.Contains(t, output, "example.invalid/fixture/missing")
	require.Empty(t, repo.calls())
}

func TestLintRejectsHostSourceSymlinks(t *testing.T) {
	t.Parallel()
	repo := newLintRepo(t)
	sourcePath := filepath.Join(t.TempDir(), "source.go")
	require.NoError(t, os.WriteFile(sourcePath, []byte("package runner\n"), 0o600))
	path := "tools/gomad/internal/hostcustom/scheduler/source.go"
	require.NoError(t, os.MkdirAll(filepath.Dir(filepath.Join(repo.root, path)), 0o700))
	require.NoError(t, os.Symlink(sourcePath, filepath.Join(repo.root, path)))
	_, err := repo.runMake("lint-code-gomad")
	require.Error(t, err)
	require.Empty(t, repo.calls())
}

func TestLintRejectsHostDirectorySymlinks(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct{ path, source string }{
		{"tools/gomad/internal/hostcustom/scheduler", "source.go"},
		{"tools/gomad/internal/hostcustom", "scheduler/source.go"},
	} {
		t.Run(tc.path, func(t *testing.T) {
			t.Parallel()
			repo := &lintRepo{t: t, root: t.TempDir()}
			linked := t.TempDir()
			sourcePath := filepath.Join(linked, tc.source)
			require.NoError(t, os.MkdirAll(filepath.Dir(sourcePath), 0o700))
			require.NoError(t, os.WriteFile(sourcePath, []byte("package runner\n"), 0o600))
			require.NoError(t, os.MkdirAll(filepath.Dir(filepath.Join(repo.root, tc.path)), 0o700))
			require.NoError(t, os.Symlink(linked, filepath.Join(repo.root, tc.path)))
			path := filepath.ToSlash(filepath.Join(tc.path, tc.source))
			_, err := coveredPackages(t.Context(), repo.root, gomadModule, "test_dep", []source{{path: path, module: gomadModule}})
			require.EqualError(t, err, "uncovered source symlink "+path)
		})
	}
}

func TestFastLintRoutesModuleOwners(t *testing.T) {
	t.Parallel()
	repo := newLintRepo(t)
	for _, path := range []string{"ordinary/source.go", "tools/gomad/internal/runner/source.go", "tests/mixedbrain/source.go", ".github/actions/build-docker-images/scripts/source.go", "tools/gomad/internal/model/cluster/source.go"} {
		repo.write(path, "package sample\nvar Changed = 1\n")
	}
	repo.make("lint-code-fast")
	require.Equal(t, []lintCall{
		{"golangci", ".", repo.lintArgs(".", "./.github/actions/build-docker-images/scripts", "./ordinary")},
		{"vet", ".", repo.vetArgs("./.github/actions/build-docker-images/scripts", "./ordinary")},
		{"golangci", "tools/gomad", repo.lintArgs("tools/gomad", "./internal/archcheck", "./internal/model/cluster", "./internal/runner")},
		{"vet", "tools/gomad", repo.vetArgs("./internal/archcheck", "./internal/model/cluster", "./internal/runner")},
		{"golangci", "tests/mixedbrain", repo.lintArgs("tests/mixedbrain", ".")},
		{"vet", "tests/mixedbrain", repo.vetArgs(".")},
	}, repo.calls())
}

func TestWASMLintModuleRoutesHostSourcesAndRetainsClosedInventory(t *testing.T) {
	t.Parallel()
	for _, target := range []string{"lint-code-fast", "lint-code-gomad-wasm"} {
		t.Run(target, func(t *testing.T) {
			t.Parallel()
			repo := newLintRepo(t)
			repo.write("tools/gomad/internal/model/wasi/source.go", "package wasi\n")
			repo.write("tools/gomad/internal/backends/wasm/testdata/environment/main.go", "package main\n")
			repo.make(target)
			require.Equal(t, []lintCall{
				{"golangci", "tools/gomad", repo.lintArgs("tools/gomad", "./internal/archcheck", "./internal/model/cluster", "./internal/model/wasi", "./internal/runner")},
				{"vet", "tools/gomad", repo.vetArgs("./internal/archcheck", "./internal/model/cluster", "./internal/model/wasi", "./internal/runner")},
			}, repo.calls())
		})
	}
	for _, path := range []string{"tools/gomad/internal/backends/wasm/go.mod", "tools/gomad/internal/backends/wasm_extra/go.mod", "tools/gomad/internal/backends/wasm/testdata/unknown/main.go", "tools/gomad/internal/model/wasi/testdata/runtime_cost_extra/main.go", "tools/gomad/internal/backends/wasm/command/unknown/testdata/main.go", "tools/gomad/internal/backends/wasm/testdata/environment/nested/go.mod"} {
		t.Run(path, func(t *testing.T) {
			t.Parallel()
			repo := newLintRepo(t)
			repo.write("tools/gomad/internal/model/wasi/source.go", "package wasi\n")
			contents := "package main\n"
			if strings.HasSuffix(path, "go.mod") {
				contents = "module example.invalid/unowned\n\ngo 1.27.0\n"
			}
			repo.write(path, contents)
			output, err := repo.runMake("lint-code-fast")
			require.Error(t, err)
			require.Contains(t, output, path)
			require.Empty(t, repo.calls())
		})
	}
}

func TestWASMLintHostSourceBoundaries(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		relative, disposition string
		uncovered             bool
	}{
		{"source.go", "", false},
		{"wasi/source.go", "", false},
		{"wasi/testdata_extra/source.go", "", false},
		{".hidden/source.go", "", true},
		{"_hidden/source.go", "", true},
		{"wasi/.hidden/source.go", "", true},
		{"wasi/_hidden/source.go", "", true},
		{"wasi/testdata/source.go", "", true},
		{"testdata/environment_extra/source.go", "", true},
		{"testdata/unsupported_clock_extra/source.go", "", true},
		{"wasi/testdata/runtime_cost_extra/source.go", "", true},
		{"testdata/environment/.hidden/source.go", "WASM environment fixture", false},
		{"testdata/unsupported_clock/main.go", "WASM determinism prototype fixture", false},
		{"wasi/testdata/runtime_cost/_hidden/source.go", "WASM productionization fixture", false},
	} {
		t.Run(tc.relative, func(t *testing.T) {
			t.Parallel()
			path := "tools/gomad/internal/backends/wasm/" + tc.relative
			entry, err := (ownership{}).classify(path, []string{"tools/gomad", "."})
			if tc.uncovered {
				require.EqualError(t, err, "uncovered WASM host source "+path)
				require.Equal(t, source{path: path}, entry)
				return
			}
			require.NoError(t, err)
			want := source{path: path, disposition: tc.disposition}
			if tc.disposition == "" {
				want.module = "tools/gomad"
			}
			require.Equal(t, want, entry)
		})
	}
}

func TestLintRelocatedWASIFixtureBoundaries(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		relative, disposition string
		uncovered             bool
	}{
		{"source.go", "", false},
		{"testdata_extra/source.go", "", false},
		{".hidden/source.go", "", true},
		{"_hidden/source.go", "", true},
		{"testdata/source.go", "", true},
		{"testdata/runtime_cost_extra/source.go", "", true},
		{"testdata/namespace_owner_extra/source.go", "", true},
		{"testdata/runtime_cost/main.go", "WASM productionization fixture", false},
		{"testdata/runtime_cost/_hidden/source.go", "WASM productionization fixture", false},
		{"testdata/namespace_owner/main.go", "WASM productionization fixture", false},
	} {
		t.Run(tc.relative, func(t *testing.T) {
			t.Parallel()
			path := gomadModule + "/internal/model/wasi/" + tc.relative
			policy := ownership{fixtures: []string{"internal/model/wasi/testdata"}}
			entry, err := policy.classify(path, []string{gomadModule, "."})
			if tc.uncovered {
				require.EqualError(t, err, "uncovered Gomad host source "+path)
				require.Equal(t, source{path: path}, entry)
				return
			}
			require.NoError(t, err)
			want := source{path: path, disposition: tc.disposition}
			if tc.disposition == "" {
				want.module = gomadModule
			}
			require.Equal(t, want, entry)
		})
	}
}

func TestWASMLintRoutesSoftAssertionFixtureToItsServerOwner(t *testing.T) {
	t.Parallel()
	for _, path := range []string{"tests/gomadfunctional/testdata/softassert/softassert_test.go", "tests/gomadfunctional/testdata/softassert_extra/softassert_test.go"} {
		entry, err := (ownership{}).classify(path, []string{"tools/gomad", "."})
		require.NoError(t, err)
		if strings.Contains(path, "softassert_extra") {
			require.Equal(t, source{path: path, module: "."}, entry)
		} else {
			require.Equal(t, source{path: path, disposition: "WASM soft assertion fixture"}, entry)
		}
	}
}

func TestLintRoutesGomadDebugFixtures(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		path        string
		disposition string
	}{
		{"common/gomaddebug/testdata/determinism/main.go", "Gomad debug qualification fixture"},
		{"common/gomaddebug/testdata/determinism/nested/main.go", "Gomad debug qualification fixture"},
		{"common/gomaddebug/testdata/fixture/main.go", "Gomad debug qualification fixture"},
		{"common/gomaddebug/testdata/fixture/nested/main.go", "Gomad debug qualification fixture"},
		{"common/gomaddebug/testdata/determinism_extra/main.go", ""},
		{"common/gomaddebug/testdata/fixture_extra/main.go", ""},
		{"common/gomaddebug/testdata/unclassified/main.go", ""},
		{"common/gomaddebug/debug.go", ""},
	} {
		t.Run(tc.path, func(t *testing.T) {
			t.Parallel()
			entry, err := (ownership{}).classify(tc.path, []string{"tools/gomad", "."})
			require.NoError(t, err)
			want := source{path: tc.path, disposition: tc.disposition}
			if tc.disposition == "" {
				want.module = "."
			}
			require.Equal(t, want, entry)
		})
	}
}

func TestWASMLintRoutesTemporalDiagnosticFixtures(t *testing.T) {
	t.Parallel()
	for _, owner := range []string{"frontend_namespace", "sqlite_contention"} {
		for _, target := range []string{"lint-code-fast", "lint-code-gomad-wasm"} {
			t.Run(owner+"/"+target, func(t *testing.T) {
				t.Parallel()
				repo := newLintRepo(t)
				repo.write("tools/gomad/internal/model/wasi/source.go", "package wasi\n")
				repo.write("tools/gomad/internal/backends/wasm/testdata/"+owner+"/fixture_test.go", "package fixture\n")
				repo.make(target)
				require.Equal(t, []lintCall{
					{"golangci", "tools/gomad", repo.lintArgs("tools/gomad", "./internal/archcheck", "./internal/model/cluster", "./internal/model/wasi", "./internal/runner")},
					{"vet", "tools/gomad", repo.vetArgs("./internal/archcheck", "./internal/model/cluster", "./internal/model/wasi", "./internal/runner")},
				}, repo.calls())
			})
		}
		t.Run(owner+"/nested-module", func(t *testing.T) {
			t.Parallel()
			repo := newLintRepo(t)
			path := "tools/gomad/internal/backends/wasm/testdata/" + owner + "/nested/go.mod"
			repo.write(path, "module example.invalid/unowned\n\ngo 1.27.0\n")
			output, err := repo.runMake("lint-code-fast")
			require.Error(t, err)
			require.Contains(t, output, path)
			require.Empty(t, repo.calls())
		})
	}
}

func TestWASMLintRoutesRuntimeQualificationFixtures(t *testing.T) {
	t.Parallel()
	for _, fixture := range []string{"toolchain/testdata/choices", "toolchain/testdata/choice_divergence", "testdata/runtime_probes", "testdata/time_probes", "testdata/unsupported_clock", "testdata/network_probes", "testdata/prototype_temporal"} {
		t.Run(fixture, func(t *testing.T) {
			t.Parallel()
			repo := newLintRepo(t)
			repo.write("tools/gomad/internal/model/wasi/source.go", "package wasi\n")
			repo.write("tools/gomad/internal/backends/wasm/"+fixture+"/main.go", "package main\n")
			repo.make("lint-code-fast")
			require.Equal(t, []lintCall{
				{"golangci", "tools/gomad", repo.lintArgs("tools/gomad", "./internal/archcheck", "./internal/model/cluster", "./internal/model/wasi", "./internal/runner")},
				{"vet", "tools/gomad", repo.vetArgs("./internal/archcheck", "./internal/model/cluster", "./internal/model/wasi", "./internal/runner")},
			}, repo.calls())
		})
	}
}

func TestWASMLintRoutesProductionizationFixtures(t *testing.T) {
	t.Parallel()
	for _, fixture := range []string{"testdata/storage_sqlite", "testdata/network_streams", "testdata/temporal_storage", "testdata/temporal_broker", "wasi/testdata/runtime_cost", "cluster/testdata/service", "command/testdata"} {
		t.Run(fixture, func(t *testing.T) {
			t.Parallel()
			repo := newLintRepo(t)
			repo.write("tools/gomad/internal/model/wasi/source.go", "package wasi\n")
			repo.write("tools/gomad/internal/backends/wasm/"+fixture+"/main.go", "package main\n")
			repo.make("lint-code-fast")
			require.Equal(t, []lintCall{
				{"golangci", "tools/gomad", repo.lintArgs("tools/gomad", "./internal/archcheck", "./internal/model/cluster", "./internal/model/wasi", "./internal/runner")},
				{"vet", "tools/gomad", repo.vetArgs("./internal/archcheck", "./internal/model/cluster", "./internal/model/wasi", "./internal/runner")},
			}, repo.calls())
		})
		t.Run(fixture+"/nested-module", func(t *testing.T) {
			t.Parallel()
			repo := newLintRepo(t)
			fixtureModule := "tools/gomad/internal/backends/wasm/" + fixture + "/nested/go.mod"
			repo.write(fixtureModule, "module example.invalid/unowned\n\ngo 1.27.0\n")
			output, err := repo.runMake("lint-code-fast")
			require.Error(t, err)
			require.Contains(t, output, fixtureModule)
			require.Empty(t, repo.calls())
		})
	}
}

func TestWASMLintAdmitsOnlyDeclaredQualificationFixtureModule(t *testing.T) {
	t.Parallel()
	for _, extra := range []string{"", "nested/go.mod", "../unknown/go.mod"} {
		t.Run(extra, func(t *testing.T) {
			t.Parallel()
			repo := newLintRepo(t)
			repo.write("tools/gomad/internal/model/wasi/source.go", "package wasi\n")
			fixture := "tools/gomad/internal/backends/wasm/qualification/testdata/fixture/"
			repo.write(fixture+"go.mod", "module example.invalid/qualification\n\ngo 1.27.0\n")
			repo.write(fixture+"main.go", "package main\n")
			if extra != "" {
				repo.write(fixture+extra, "module example.invalid/unowned\n\ngo 1.27.0\n")
			}
			output, err := repo.runMake("lint-code-fast")
			if extra != "" {
				require.Error(t, err, output)
				require.Contains(t, output, "unclassified module")
				require.Empty(t, repo.calls())
				return
			}
			require.NoError(t, err, output)
			require.Equal(t, []lintCall{
				{"golangci", "tools/gomad", repo.lintArgs("tools/gomad", "./internal/archcheck", "./internal/model/cluster", "./internal/model/wasi", "./internal/runner")},
				{"vet", "tools/gomad", repo.vetArgs("./internal/archcheck", "./internal/model/cluster", "./internal/model/wasi", "./internal/runner")},
			}, repo.calls())
		})
	}
}

func TestWASMLintAdmitsDeclaredFixtureModules(t *testing.T) {
	t.Parallel()
	for _, test := range []struct{ fixture, source, extra string }{
		{"io_failure_module/", "io_failure/io_failure_test.go", ""},
		{"io_failure_module/", "io_failure/io_failure_test.go", "nested/go.mod"},
		{"ringpop/", "main.go", ""},
		{"ringpop/", "main.go", "nested/go.mod"},
	} {
		fixture, extra := "tools/gomad/internal/backends/wasm/testdata/"+test.fixture, test.extra
		t.Run(test.fixture+extra, func(t *testing.T) {
			t.Parallel()
			repo := newLintRepo(t)
			repo.write("tools/gomad/internal/model/wasi/source.go", "package wasi\n")
			repo.write(fixture+"go.mod", "module example.invalid/fixture\n\ngo 1.27.0\n")
			repo.write(fixture+test.source, "package fixture\n")
			if extra != "" {
				repo.write(fixture+extra, "module example.invalid/unowned\n\ngo 1.27.0\n")
			}
			for _, target := range []string{"lint-code-fast", "lint-code-gomad-wasm"} {
				output, err := repo.runMake(target)
				if extra != "" {
					require.Error(t, err, output)
					require.Contains(t, output, "unclassified module "+fixture+extra)
					require.Empty(t, repo.calls())
					continue
				}
				require.NoError(t, err, output)
				require.NotContains(t, output, "uncovered")
			}
			if extra != "" {
				return
			}
			packages := []string{"./internal/archcheck", "./internal/model/cluster", "./internal/model/wasi", "./internal/runner"}
			calls := repo.calls()
			require.Len(t, calls, 4)
			for _, call := range calls {
				require.Equal(t, "tools/gomad", call.Dir)
				require.NotContains(t, strings.Join(call.Args, " "), "testdata/"+test.fixture)
			}
			require.Equal(t, []lintCall{{"golangci", "tools/gomad", repo.lintArgs("tools/gomad", packages...)}, {"vet", "tools/gomad", repo.vetArgs(packages...)}}, calls[:2])
		})
	}
}

func TestFastLintKeepsGitChangeSemantics(t *testing.T) {
	t.Parallel()
	repo := newLintRepo(t)
	repo.write("ordinary/source.go", "package sample\nvar Tracked = 1\n")
	repo.write("newpkg/new.go", "package sample\n")
	require.NoError(t, os.Remove(filepath.Join(repo.root, "deleted/only.go")))
	repo.git("mv", "renameold/only.go", "renamenew/moved.go")
	repo.write("staged/staged.go", "package sample\nvar Staged = 1\n")
	repo.git("add", "staged/staged.go")
	repo.git("commit", "-m", "changed after comparison")
	repo.make("lint-code-fast")
	require.Equal(t, []lintCall{
		{"golangci", ".", repo.lintArgs(".", "./newpkg", "./ordinary", "./renamenew", "./staged")},
		{"vet", ".", repo.vetArgs("./newpkg", "./ordinary", "./renamenew", "./staged")},
	}, repo.calls())
}

func TestFastLintRoutesHistoricalDeletions(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name, deleted, moved, module, failure string
		packages                              []string
	}{
		{"retired fixture", "tools/gomad-legacy/deterministicio/testdata/cactusstatsd/cactusstatsd_test.go", "", "", "", nil},
		{"folded module", "tools/gomad_wasm/wasi/clock.go", "tools/gomad/internal/model/wasi/clock.go", "tools/gomad", "", []string{"./internal/archcheck", "./internal/model/cluster", "./internal/model/wasi", "./internal/runner"}},
		{"surviving package", "ordinary/deleted.go", "", ".", "", []string{"./ordinary"}},
		{"surviving package lint failure", "ordinary/deleted.go", "", ".", "golangci", []string{"./ordinary"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			repo := newLintRepo(t)
			repo.write(tc.deleted, "package sample\n")
			if tc.moved != "" {
				repo.write("tools/gomad_wasm/go.mod", "module example.invalid/oldwasm\n\ngo 1.27.0\n")
			}
			repo.git("add", ".")
			repo.git("commit", "-qm", "historical source layout")
			repo.base = strings.TrimSpace(repo.git("rev-parse", "HEAD"))
			require.NoError(t, os.Remove(filepath.Join(repo.root, tc.deleted)))
			if tc.moved != "" {
				require.NoError(t, os.Remove(filepath.Join(repo.root, "tools/gomad_wasm/go.mod")))
				repo.write(tc.moved, "package sample\n")
			}
			repo.env = append(repo.env, "LINT_TEST_FAIL="+tc.failure)
			output, err := repo.runMake("lint-code-fast")
			if tc.failure == "" {
				require.NoError(t, err, output)
			} else {
				require.Error(t, err, output)
			}
			var want []lintCall
			if tc.module != "" {
				want = append(want, lintCall{"golangci", tc.module, repo.lintArgs(tc.module, tc.packages...)})
				if tc.failure != "golangci" {
					want = append(want, lintCall{"vet", tc.module, repo.vetArgs(tc.packages...)})
				}
			}
			require.Equal(t, want, repo.calls())
		})
	}
}

func TestFastLintReportsOwnedExclusions(t *testing.T) {
	t.Parallel()
	repo := newLintRepo(t)
	for _, path := range []string{
		".flow/frozen/old.go",
		"tools/gomad/internal/model/cluster/testdata/simulation_exploration/source.go",
		"tools/gomad/cmd/gomad/testdata/source.go",
		"tools/gomad/internal/evidence/qualification/corpus/source.go",
	} {
		repo.write(path, "package fixture\n")
	}
	output := repo.make("lint-code-fast")
	require.Empty(t, repo.calls())
	for _, owner := range []string{"retained evidence", "simulation fixture", "Gomad qualification fixture"} {
		require.Contains(t, output, owner)
	}
}

func TestLintRejectsUnclassifiedSource(t *testing.T) {
	t.Parallel()
	for _, path := range []string{
		"extra/go.mod", ".hidden/go.mod", "_hidden/go.mod",
		"tools/gomad/internal/model/cluster/testdata/simulation_exploration/extra/go.mod",
		"tools/gomad/internal/evidence/record/testdata/source.go",
		"tools/gomad/internal/runner/testdata/source.go",
		"tools/gomad/internal/runner/.hidden/source.go",
		"tools/gomad/internal/runner/_hidden/source.go",
		"ordinary/_ignored.go",
	} {
		t.Run(path, func(t *testing.T) {
			t.Parallel()
			repo := newLintRepo(t)
			contents := "package sample\n"
			if strings.HasSuffix(path, "go.mod") {
				contents = "module example.invalid/unclassified\n\ngo 1.27.0\n"
				repo.write(filepath.ToSlash(filepath.Join(filepath.Dir(path), "source.go")), "package sample\n")
			}
			repo.write(path, contents)
			output, err := repo.runMake("lint-code-fast")
			require.Error(t, err)
			require.Contains(t, output, path)
			require.Empty(t, repo.calls())
		})
	}
}

func TestLintRejectsBadComparisonBeforeDispatch(t *testing.T) {
	t.Parallel()
	repo := newLintRepo(t)
	repo.write("ordinary/source.go", "package sample\nvar Changed = 1\n")
	output, err := repo.runMake("lint-code-fast", "GOLANGCI_LINT_BASE_REV=missing-revision")
	require.Error(t, err)
	require.Contains(t, output, "missing-revision")
	require.Empty(t, repo.calls())
}

func TestScopedLintAndNestedFailure(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		target, dir, fail string
		packages          []string
	}{
		{"lint-code-gomad", "tools/gomad", "", []string{"./internal/archcheck", "./internal/model/cluster", "./internal/runner"}},
		{"lint-code-mixedbrain", "tests/mixedbrain", "", []string{"."}},
		{"lint-code-fast", "tools/gomad", "golangci", []string{"./internal/archcheck", "./internal/model/cluster", "./internal/runner"}},
		{"lint-code-fast", "tools/gomad", "vet", []string{"./internal/archcheck", "./internal/model/cluster", "./internal/runner"}},
	} {
		t.Run(tc.target+tc.dir+tc.fail, func(t *testing.T) {
			t.Parallel()
			repo := newLintRepo(t)
			repo.write("tools/gomad/internal/runner/source.go", "package sample\nvar Changed = 1\n")
			repo.env = append(repo.env, "LINT_TEST_FAIL="+tc.fail)
			output, err := repo.runMake(tc.target)
			if tc.fail == "" {
				require.NoError(t, err, output)
			} else {
				require.Error(t, err, output)
			}
			want := []lintCall{{"golangci", tc.dir, repo.lintArgs(tc.dir, tc.packages...)}}
			if tc.fail != "golangci" {
				want = append(want, lintCall{"vet", tc.dir, repo.vetArgs(tc.packages...)})
			}
			require.Equal(t, want, repo.calls())
		})
	}
}

func TestGomadLocalLintTarget(t *testing.T) {
	t.Parallel()
	repo := newLintRepo(t)
	repo.make("-C", "tools/gomad", "lint-code")
	require.Equal(t, []lintCall{
		{"golangci", "tools/gomad", repo.lintArgs("tools/gomad", "./internal/archcheck", "./internal/model/cluster", "./internal/runner")},
		{"vet", "tools/gomad", repo.vetArgs("./internal/archcheck", "./internal/model/cluster", "./internal/runner")},
	}, repo.calls())
}

func TestGomadDefaultRemainsGenerate(t *testing.T) {
	t.Parallel()
	repo := newLintRepo(t)
	repo.write("default-goal.mk", "inspect-default:\n\t@printf '%s\\n' '$(.DEFAULT_GOAL)'\n")
	output := repo.make("--no-print-directory", "-C", "tools/gomad", "-f", "Makefile", "-f", "../../default-goal.mk", "inspect-default")
	require.Equal(t, "generate\n", output)
}

func TestScopedLintRejectsEmptyModule(t *testing.T) {
	t.Parallel()
	repo := newLintRepo(t)
	require.NoError(t, os.Remove(filepath.Join(repo.root, "tests/mixedbrain/source.go")))
	output, err := repo.runMake("lint-code-mixedbrain")
	require.Error(t, err)
	require.Contains(t, output, "no ordinary host source")
	require.Empty(t, repo.calls())
}

func TestLintRejectsModuleSymlink(t *testing.T) {
	t.Parallel()
	repo := newLintRepo(t)
	path := filepath.Join(repo.root, "tests/mixedbrain/go.mod")
	require.NoError(t, os.Remove(path))
	target := filepath.Join(t.TempDir(), "go.mod")
	require.NoError(t, os.WriteFile(target, []byte("module example.invalid/outside\n\ngo 1.27.0\n"), 0o600))
	require.NoError(t, os.Symlink(target, path))
	output, err := repo.runMake("lint-code-mixedbrain")
	require.Error(t, err)
	require.Contains(t, output, "module symlink")
	require.Empty(t, repo.calls())
}

func TestLintRejectsMissingLiveModule(t *testing.T) {
	t.Parallel()
	for _, path := range []string{"tools/gomad/go.mod", "tests/mixedbrain/go.mod"} {
		t.Run(path, func(t *testing.T) {
			t.Parallel()
			repo := newLintRepo(t)
			require.NoError(t, os.Remove(filepath.Join(repo.root, path)))
			output, err := repo.runMake("lint-code-gomad")
			require.Error(t, err)
			require.Contains(t, output, path)
			require.Empty(t, repo.calls())
		})
	}
}

func TestClusterFixturePrefixNearMissIsRejected(t *testing.T) {
	t.Parallel()
	repo := newLintRepo(t)
	repo.write("tools/gomad/internal/model/cluster/testdata/simulation_exploration_extra/source.go", "package sample\n")
	output, err := repo.runMake("lint-code-fast")
	require.Error(t, err)
	require.Contains(t, output, "uncovered Gomad host source tools/gomad/internal/model/cluster/testdata/simulation_exploration_extra/source.go")
	require.Empty(t, repo.calls())
}

func TestLintRejectsInvalidOwnership(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct{ path, contents string }{
		{"tools/gomad/internal/archcheck/inventory.go", "package archcheck\nvar sourceExclusions = paths()\nvar expectedModules = map[string]bool{\"testdata/go.mod\":true}\n"},
	} {
		t.Run(tc.contents, func(t *testing.T) {
			t.Parallel()
			repo := newLintRepo(t)
			repo.write(tc.path, tc.contents)
			_, err := repo.runMake("lint-code-fast")
			require.Error(t, err)
			require.Empty(t, repo.calls())
		})
	}
}

func TestCINestedLintSteps(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		workflow, job, step, module string
		packages                    []string
	}{
		{"gomad.yml", "reference-tools-linux", "Lint Gomad host module", "tools/gomad", []string{"./internal/archcheck", "./internal/model/cluster", "./internal/runner"}},
		{"linters.yml", "golangci", "lint mixedbrain module", "tests/mixedbrain", []string{"."}},
	} {
		t.Run(tc.workflow, func(t *testing.T) {
			t.Parallel()
			repo := newLintRepo(t)
			path := ".github/workflows/" + tc.workflow
			data, err := os.ReadFile(filepath.Join("../../..", path))
			require.NoError(t, err)
			if revision := os.Getenv("LINT_TEST_BASE_REV"); revision != "" {
				data, err = exec.CommandContext(t.Context(), "git", "show", revision+":"+path).Output()
				require.NoError(t, err)
			}
			var workflow struct {
				Jobs map[string]struct {
					Steps []struct{ Name, Run, WorkingDirectory string }
				}
			}
			require.NoError(t, yaml.Unmarshal(data, &workflow))
			var run string
			for _, step := range workflow.Jobs[tc.job].Steps {
				if step.Name == tc.step {
					run = step.Run
				}
			}
			require.NotEmpty(t, run, "nested lint step is not enforced")
			args := strings.Fields(run)
			require.Equal(t, "make", args[0])
			repo.git("commit", "--allow-empty", "-qm", "CI head")
			repo.base = "HEAD~"
			repo.make(args[1:]...)
			require.Equal(t, []lintCall{
				{"golangci", tc.module, repo.lintArgs(tc.module, tc.packages...)},
				{"vet", tc.module, repo.vetArgs(tc.packages...)},
			}, repo.calls())
		})
	}
}

func TestLintToolProcess(t *testing.T) {
	tool := os.Getenv("LINT_TEST_TOOL")
	if tool == "" {
		return
	}
	dir, err := os.Getwd()
	require.NoError(t, err)
	rel, err := filepath.Rel(os.Getenv("LINT_TEST_ROOT"), dir)
	require.NoError(t, err)
	separator := slices.Index(os.Args, "--")
	require.NotEqual(t, -1, separator)
	args := slices.Clone(os.Args[separator+1:])
	for i, arg := range args {
		if path, ok := strings.CutPrefix(arg, "--new-from-patch="); ok {
			_, err := os.ReadFile(path)
			require.NoError(t, err)
			args[i] = "--new-from-patch=" + filepath.ToSlash(rel)
		}
	}
	data, err := json.Marshal(lintCall{tool, filepath.ToSlash(rel), args})
	require.NoError(t, err)
	file, err := os.OpenFile(os.Getenv("LINT_TEST_LOG"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	require.NoError(t, err)
	_, err = file.Write(append(data, '\n'))
	require.NoError(t, err)
	require.NoError(t, file.Close())
	failurePackage := os.Getenv("LINT_TEST_FAIL_PACKAGE")
	if os.Getenv("LINT_TEST_FAIL") == tool && (failurePackage == "" || slices.Contains(os.Args[separator+1:], failurePackage)) {
		t.Fatal("controlled lint tool failure")
	}
}

type lintRepo struct {
	t    *testing.T
	root string
	base string
	env  []string
}

func newLintRepo(t *testing.T) *lintRepo {
	t.Helper()
	root, err := filepath.Abs("../../..")
	require.NoError(t, err)
	repo := &lintRepo{t: t, root: t.TempDir()}
	for _, path := range []string{"Makefile", "tools/gomad/Makefile", "tools/gomad/internal/archcheck/inventory.go"} {
		data, err := os.ReadFile(filepath.Join(root, path))
		require.NoError(t, err)
		if filepath.Dir(path) == "tools/gomad/internal/archcheck" {
			fileSet := token.NewFileSet()
			file, err := parser.ParseFile(fileSet, path, data, 0)
			require.NoError(t, err)
			file.Decls = slices.DeleteFunc(file.Decls, func(declaration ast.Decl) bool {
				group, ok := declaration.(*ast.GenDecl)
				return !ok || group.Tok != token.VAR
			})
			var fixture bytes.Buffer
			require.NoError(t, printer.Fprint(&fixture, fileSet, file))
			data = fixture.Bytes()
		}
		if revision := os.Getenv("LINT_TEST_BASE_REV"); revision != "" && filepath.Base(path) == "Makefile" {
			command := exec.CommandContext(t.Context(), "git", "show", revision+":"+path)
			command.Dir = root
			data, err = command.Output()
			require.NoError(t, err)
		}
		repo.write(path, string(data))
	}
	files, err := filepath.Glob(filepath.Join(root, "cmd/tools/lintcode/*.go"))
	require.NoError(t, err)
	for _, path := range files {
		if strings.HasSuffix(path, "_test.go") {
			continue
		}
		data, err := os.ReadFile(path)
		require.NoError(t, err)
		repo.write("cmd/tools/lintcode/"+filepath.Base(path), string(data))
	}
	for _, path := range []string{"go.mod", "tools/gomad/go.mod", "tests/mixedbrain/go.mod", "tools/gomad/internal/evidence/qualification/corpus/go.mod"} {
		repo.write(path, "module example.invalid/fixture\n\ngo 1.27.0\n")
	}
	repo.write("go.mod", "module example.invalid/root\n\ngo 1.27.0\n")
	for _, dir := range []string{"proto/internal", "chasm/lib"} {
		require.NoError(t, os.MkdirAll(filepath.Join(repo.root, dir), 0o700))
	}
	for _, path := range []string{"ordinary/source.go", "staged/staged.go", "deleted/only.go", "renameold/only.go", "tools/gomad/internal/runner/source.go", "tests/mixedbrain/source.go", ".github/actions/build-docker-images/scripts/source.go", "tools/gomad/internal/model/cluster/source.go"} {
		repo.write(path, "package sample\n")
	}
	repo.write(".github/.golangci.yml", "version: \"2\"\n")
	require.NoError(t, os.MkdirAll(filepath.Join(repo.root, "renamenew"), 0o700))
	goPath, err := exec.LookPath("go")
	require.NoError(t, err)
	binary, err := os.Executable()
	require.NoError(t, err)
	toolScript := "#!/bin/sh\nexec '" + binary + "' -test.run=^TestLintToolProcess$ -- \"$@\"\n"
	repo.write("bin/golangci", "#!/bin/sh\nexport LINT_TEST_TOOL=golangci\n"+strings.TrimPrefix(toolScript, "#!/bin/sh\n"))
	repo.write("bin/go", "#!/bin/sh\nif [ \"$1\" = vet ]; then\n export LINT_TEST_TOOL=vet\n "+strings.TrimPrefix(toolScript, "#!/bin/sh\n")+"fi\nif [ \"$1\" = run ] && [ \"$2\" = ./cmd/tools/lintcode ]; then\n '"+goPath+"' build -o '"+filepath.Join(repo.root, "bin/selector")+"' ./cmd/tools/lintcode || exit $?\n shift 2\n exec '"+filepath.Join(repo.root, "bin/selector")+"' \"$@\"\nfi\nexec '"+goPath+"' \"$@\"\n")
	repo.write("bin/errortype", "#!/bin/sh\nexit 0\n")
	for _, path := range []string{"bin/go", "bin/golangci", "bin/errortype"} {
		require.NoError(t, os.Chmod(filepath.Join(repo.root, path), 0o700))
	}
	repo.env = append(os.Environ(), "PATH="+filepath.Join(repo.root, "bin")+string(os.PathListSeparator)+os.Getenv("PATH"), "LINT_TEST_LOG="+filepath.Join(repo.root, "calls.jsonl"), "LINT_TEST_ROOT="+repo.root)
	repo.git("init", "-q")
	repo.git("config", "user.email", "lint-test@example.invalid")
	repo.git("config", "user.name", "Lint test")
	repo.git("add", ".")
	repo.git("commit", "-qm", "baseline")
	repo.base = strings.TrimSpace(repo.git("rev-parse", "HEAD"))
	return repo
}

func (r *lintRepo) write(path, data string) {
	r.t.Helper()
	path = filepath.Join(r.root, path)
	require.NoError(r.t, os.MkdirAll(filepath.Dir(path), 0o700))
	require.NoError(r.t, os.WriteFile(path, []byte(data), 0o600))
}

func (r *lintRepo) git(args ...string) string {
	r.t.Helper()
	command := exec.CommandContext(r.t.Context(), "git", args...)
	command.Dir = r.root
	output, err := command.CombinedOutput()
	require.NoError(r.t, err, string(output))
	return string(output)
}

func (r *lintRepo) runMake(args ...string) (string, error) {
	r.t.Helper()
	args = append([]string{"GO_API_VER=v0.0.0", "GOLANGCI_LINT_BASE_REV=" + r.base, "GOLANGCI_LINT_FIX=false", "GOLANGCI_LINT=" + filepath.Join(r.root, "bin/golangci"), "ERRORTYPE=" + filepath.Join(r.root, "bin/errortype"), "LOCALBIN=" + filepath.Join(r.root, "bin")}, args...)
	command := exec.CommandContext(r.t.Context(), "make", args...)
	command.Dir, command.Env = r.root, r.env
	output, err := command.CombinedOutput()
	return string(output), err
}

func (r *lintRepo) make(args ...string) string {
	r.t.Helper()
	output, err := r.runMake(args...)
	require.NoError(r.t, err, output)
	return output
}

func (r *lintRepo) calls() []lintCall {
	r.t.Helper()
	data, err := os.ReadFile(filepath.Join(r.root, "calls.jsonl"))
	if os.IsNotExist(err) {
		return nil
	}
	require.NoError(r.t, err)
	var calls []lintCall
	for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
		var call lintCall
		require.NoError(r.t, json.Unmarshal([]byte(line), &call))
		calls = append(calls, call)
	}
	return calls
}

func (r *lintRepo) lintArgs(module string, packages ...string) []string {
	diff := "--new-from-rev=" + r.base
	if module != "." {
		diff = "--new-from-patch=" + module
	}
	return append([]string{"run", "--verbose", "--build-tags", "disable_grpc_modules,,test_dep,", "--timeout", "10m", "--fix=false", diff, "--config=" + filepath.Join(r.root, ".github/.golangci.yml")}, packages...)
}

func (r *lintRepo) vetArgs(packages ...string) []string {
	return append([]string{"vet", "-tags", "disable_grpc_modules,,test_dep,", "-vettool=" + filepath.Join(r.root, "bin/errortype"), "-style-check=false"}, packages...)
}
