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

func TestLintRuntimeHostRegistration(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name, entries, wantError string
		symlinkAncestor          bool
		wantPackages             map[string]bool
	}{
		{"absent directories", `"toolchain/runtime/testdata/vfdpointer", "toolchain/runtime/testdata/vfdnative"`, "", false, map[string]bool{"toolchain/runtime/testdata/vfdpointer": true, "toolchain/runtime/testdata/vfdnative": true}},
		{"missing leaf under symlink", `"toolchain/runtime/testdata/vfdpointer", "toolchain/runtime/testdata/vfdnative"`, "uncovered source symlink tools/gomad3/toolchain/runtime/testdata/vfdpointer", true, map[string]bool{}},
		{"partial registration and first error", `"toolchain/runtime/testdata/vfdpointer", "toolchain/runtime/testdata/.invalid", "toolchain/runtime/testdata/_later"`, `invalid Gomad host source package "toolchain/runtime/testdata/.invalid"`, false, map[string]bool{"toolchain/runtime/testdata/vfdpointer": true}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			repo := &lintRepo{t: t, root: t.TempDir()}
			repo.write("tools/gomad3/internal/gomadtool/architecture/architecture.go", "package architecture\nvar sourceExclusions = []string{\"testdata\"}\nvar expectedModules = map[string]bool{\"testdata/go.mod\":true}\nvar hostSourcePackages = []string{"+tc.entries+"}\n")
			repo.write("tools/gomad3/toolchain/version/version.json", `{"overlay_allowlist":["src/os/gomad.go"]}`)
			if tc.symlinkAncestor {
				require.NoError(t, os.MkdirAll(filepath.Join(repo.root, "tools/gomad3/toolchain/runtime"), 0o700))
				require.NoError(t, os.Symlink(t.TempDir(), filepath.Join(repo.root, "tools/gomad3/toolchain/runtime/testdata")))
			}
			policy, err := loadOwnership(repo.root)
			if tc.wantError == "" {
				require.NoError(t, err)
				require.Equal(t, map[string]bool{"src/os/gomad.go": true}, policy.overlays)
			} else {
				require.EqualError(t, err, tc.wantError)
				require.Empty(t, policy.overlays)
			}
			require.Equal(t, tc.wantPackages, policy.hostPackages)
			require.Equal(t, []string{"testdata"}, policy.fixtures)
			require.Equal(t, map[string]bool{"tools/gomad3/testdata/go.mod": true}, policy.modules)
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

func TestLintRuntimeHostPackages(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct{ target, failure string }{
		{"lint-code-fast", ""},
		{"lint-code-gomad3", ""},
		{"lint-code-fast", "golangci"},
		{"lint-code-gomad3", "vet"},
	} {
		t.Run(tc.target+tc.failure, func(t *testing.T) {
			t.Parallel()
			repo := newLintRepo(t)
			for _, directory := range []string{"tools/gomad3/toolchain/runtime/testdata/vfdpointer", "tools/gomad3/toolchain/runtime/testdata/vfdnative"} {
				repo.write(directory+"/source.go", "package runner\n")
				repo.write(directory+"/source_test.go", "package runner\nimport \"testing\"\nfunc TestRunner(t *testing.T) {}\n")
			}
			repo.env = append(repo.env, "LINT_TEST_FAIL="+tc.failure, "LINT_TEST_FAIL_PACKAGE=./toolchain/runtime/testdata/vfdpointer")
			output, err := repo.runMake(tc.target)
			if tc.failure == "" {
				require.NoError(t, err, output)
			} else {
				require.Error(t, err, output)
			}
			packages := []string{".", "./internal/gomadtool/architecture", "./runner", "./toolchain", "./toolchain/runtime/testdata/vfdnative", "./toolchain/runtime/testdata/vfdpointer"}
			want := []lintCall{{"golangci", "tools/gomad3", repo.lintArgs(packages...)}}
			if tc.failure != "golangci" {
				want = append(want, lintCall{"vet", "tools/gomad3", repo.vetArgs(packages...)})
			}
			require.Equal(t, want, repo.calls())
		})
	}
}

func TestLintRuntimeHostPackageImportFailure(t *testing.T) {
	t.Parallel()
	repo := newLintRepo(t)
	repo.write("tools/gomad3/toolchain/runtime/testdata/vfdpointer/source.go", "package runner\nimport _ \"example.invalid/fixture/missing\"\n")
	output, err := repo.runMake("lint-code-gomad3")
	require.Error(t, err)
	require.Contains(t, output, "example.invalid/fixture/missing")
	require.Empty(t, repo.calls())
}

func TestLintRejectsRuntimeHostSourceSymlinks(t *testing.T) {
	t.Parallel()
	repo := newLintRepo(t)
	sourcePath := filepath.Join(t.TempDir(), "source.go")
	require.NoError(t, os.WriteFile(sourcePath, []byte("package runner\n"), 0o600))
	path := "tools/gomad3/toolchain/runtime/testdata/vfdpointer/source.go"
	require.NoError(t, os.MkdirAll(filepath.Dir(filepath.Join(repo.root, path)), 0o700))
	require.NoError(t, os.Symlink(sourcePath, filepath.Join(repo.root, path)))
	_, err := repo.runMake("lint-code-gomad3")
	require.Error(t, err)
	require.Empty(t, repo.calls())
}

func TestLintRejectsRuntimeHostDirectorySymlinks(t *testing.T) {
	t.Parallel()
	for _, target := range []string{"lint-code-fast", "lint-code-gomad3"} {
		for _, tc := range []struct{ path, source string }{
			{"tools/gomad3/toolchain/runtime/testdata/vfdpointer", "source.go"},
			{"tools/gomad3/toolchain/runtime/testdata", "vfdpointer/source.go"},
		} {
			t.Run(target+"/"+tc.path, func(t *testing.T) {
				t.Parallel()
				repo := newLintRepo(t)
				linked := t.TempDir()
				sourcePath := filepath.Join(linked, tc.source)
				require.NoError(t, os.MkdirAll(filepath.Dir(sourcePath), 0o700))
				require.NoError(t, os.WriteFile(sourcePath, []byte("package runner\n"), 0o600))
				require.NoError(t, os.MkdirAll(filepath.Dir(filepath.Join(repo.root, tc.path)), 0o700))
				require.NoError(t, os.Symlink(linked, filepath.Join(repo.root, tc.path)))
				repo.git("add", tc.path)
				output, err := repo.runMake(target)
				require.Error(t, err, output)
				require.Contains(t, output, "source symlink")
				require.Empty(t, repo.calls())
			})
		}
	}
}

func TestLintRejectsInvalidRuntimeHostPackageList(t *testing.T) {
	t.Parallel()
	for _, declaration := range []string{
		"",
		"var hostSourcePackages = paths()\n",
		"var hostSourcePackages = []string{}\n",
		"var hostSourcePackages = []string{\"toolchain/runtime/testdata/vfdpointer\", \"toolchain/runtime/testdata/vfdpointer\"}\n",
		"var hostSourcePackages = []string{\"../outside\"}\n",
		"var hostSourcePackages = []string{\"toolchain/runtime/testdata/...\"}\n",
		"var hostSourcePackages = []string{\"toolchain/runtime/testdata/vfdpointer/nested\"}\n",
		"var hostSourcePackages = []string{\"toolchain/runtime/testdata/.hidden\"}\n",
	} {
		t.Run(declaration, func(t *testing.T) {
			t.Parallel()
			repo := newLintRepo(t)
			repo.write("tools/gomad3/internal/gomadtool/architecture/architecture.go", "package architecture\nvar sourceExclusions = []string{\"testdata\"}\nvar expectedModules = map[string]bool{\"testdata/go.mod\":true}\n"+declaration)
			_, err := loadOwnership(repo.root)
			require.Error(t, err)
			require.Empty(t, repo.calls())
		})
	}
}

func TestFastLintRoutesModuleOwners(t *testing.T) {
	t.Parallel()
	repo := newLintRepo(t)
	for _, path := range []string{"ordinary/source.go", "tools/gomad3/runner/source.go", "tests/mixedbrain/source.go", ".github/actions/build-docker-images/scripts/source.go", "tools/gomad3sim/source.go", "tools/gomad3integration/source.go"} {
		repo.write(path, "package sample\nvar Changed = 1\n")
	}
	repo.make("lint-code-fast")
	integrationLintArgs := repo.lintArgs("./tools/gomad3integration")
	integrationLintArgs[3] += ",gomad3_integration"
	integrationVetArgs := repo.vetArgs("./tools/gomad3integration")
	integrationVetArgs[2] += ",gomad3_integration"
	require.Equal(t, []lintCall{
		{"golangci", ".", repo.lintArgs("./.github/actions/build-docker-images/scripts", "./ordinary", "./tools/gomad3sim")},
		{"vet", ".", repo.vetArgs("./.github/actions/build-docker-images/scripts", "./ordinary", "./tools/gomad3sim")},
		{"golangci", ".", integrationLintArgs},
		{"vet", ".", integrationVetArgs},
		{"golangci", "tools/gomad3", repo.lintArgs(".", "./internal/gomadtool/architecture", "./runner", "./toolchain")},
		{"vet", "tools/gomad3", repo.vetArgs(".", "./internal/gomadtool/architecture", "./runner", "./toolchain")},
		{"golangci", "tests/mixedbrain", repo.lintArgs(".")},
		{"vet", "tests/mixedbrain", repo.vetArgs(".")},
	}, repo.calls())
}

func TestWASMLintModuleRoutesHostSourcesAndRetainsClosedInventory(t *testing.T) {
	t.Parallel()
	for _, target := range []string{"lint-code-fast", "lint-code-gomad-wasm"} {
		t.Run(target, func(t *testing.T) {
			t.Parallel()
			repo := newLintRepo(t)
			repo.write("tools/gomad_wasm/go.mod", "module example.invalid/wasm\n\ngo 1.27.0\n")
			repo.write("tools/gomad_wasm/wasi/source.go", "package wasi\n")
			repo.write("tools/gomad_wasm/testdata/environment/main.go", "package main\n")
			repo.make(target)
			require.Equal(t, []lintCall{
				{"golangci", "tools/gomad_wasm", repo.lintArgs("./wasi")},
				{"vet", "tools/gomad_wasm", repo.vetArgs("./wasi")},
			}, repo.calls())
		})
	}
	for _, path := range []string{"tools/gomad_wasm_extra/go.mod", "tools/gomad_wasm/testdata/unknown/main.go", "tools/gomad_wasm/testdata/environment/nested/go.mod"} {
		t.Run(path, func(t *testing.T) {
			t.Parallel()
			repo := newLintRepo(t)
			repo.write("tools/gomad_wasm/go.mod", "module example.invalid/wasm\n\ngo 1.27.0\n")
			repo.write("tools/gomad_wasm/wasi/source.go", "package wasi\n")
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

func TestWASMLintRoutesTemporalDiagnosticFixtures(t *testing.T) {
	t.Parallel()
	for _, owner := range []string{"frontend_namespace", "sqlite_contention"} {
		for _, target := range []string{"lint-code-fast", "lint-code-gomad-wasm"} {
			t.Run(owner+"/"+target, func(t *testing.T) {
				t.Parallel()
				repo := newLintRepo(t)
				repo.write("tools/gomad_wasm/go.mod", "module example.invalid/wasm\n\ngo 1.27.0\n")
				repo.write("tools/gomad_wasm/wasi/source.go", "package wasi\n")
				repo.write("tools/gomad_wasm/testdata/"+owner+"/fixture_test.go", "package fixture\n")
				repo.make(target)
				require.Equal(t, []lintCall{
					{"golangci", "tools/gomad_wasm", repo.lintArgs("./wasi")},
					{"vet", "tools/gomad_wasm", repo.vetArgs("./wasi")},
				}, repo.calls())
			})
		}
		t.Run(owner+"/nested-module", func(t *testing.T) {
			t.Parallel()
			repo := newLintRepo(t)
			repo.write("tools/gomad_wasm/go.mod", "module example.invalid/wasm\n\ngo 1.27.0\n")
			path := "tools/gomad_wasm/testdata/" + owner + "/nested/go.mod"
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
	for _, fixture := range []string{"toolchain/testdata/choices", "testdata/runtime_probes"} {
		t.Run(fixture, func(t *testing.T) {
			t.Parallel()
			repo := newLintRepo(t)
			repo.write("tools/gomad_wasm/go.mod", "module example.invalid/wasm\n\ngo 1.27.0\n")
			repo.write("tools/gomad_wasm/wasi/source.go", "package wasi\n")
			repo.write("tools/gomad_wasm/"+fixture+"/main.go", "package main\n")
			repo.make("lint-code-fast")
			require.Equal(t, []lintCall{
				{"golangci", "tools/gomad_wasm", repo.lintArgs("./wasi")},
				{"vet", "tools/gomad_wasm", repo.vetArgs("./wasi")},
			}, repo.calls())
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
		{"golangci", ".", repo.lintArgs("./newpkg", "./ordinary", "./renamenew", "./staged")},
		{"vet", ".", repo.vetArgs("./newpkg", "./ordinary", "./renamenew", "./staged")},
	}, repo.calls())
}

func TestFastLintRootIntegrationTagsAndFailure(t *testing.T) {
	t.Parallel()
	for _, failure := range []string{"", "golangci", "vet"} {
		t.Run(failure, func(t *testing.T) {
			t.Parallel()
			repo := newLintRepo(t)
			repo.write("tools/gomad3integration/source.go", "//go:build gomad3_integration\n\npackage sample\n")
			repo.write("ordinary/source.go", "package sample\nvar Changed = 1\n")
			repo.env = append(repo.env, "LINT_TEST_FAIL="+failure, "LINT_TEST_FAIL_PACKAGE=./tools/gomad3integration")
			output, err := repo.runMake("lint-code-fast")
			if failure == "" {
				require.NoError(t, err, output)
			} else {
				require.Error(t, err, output)
			}
			lintArgs := repo.lintArgs("./tools/gomad3integration")
			lintArgs[3] += ",gomad3_integration"
			vetArgs := repo.vetArgs("./tools/gomad3integration")
			vetArgs[2] += ",gomad3_integration"
			want := []lintCall{
				{"golangci", ".", repo.lintArgs("./ordinary")},
				{"vet", ".", repo.vetArgs("./ordinary")},
				{"golangci", ".", lintArgs},
			}
			if failure != "golangci" {
				want = append(want, lintCall{"vet", ".", vetArgs})
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
		"tools/gomad3sim/testdata/simulation_exploration/source.go",
		"tools/gomad3integration/testdata/tagged/source.go",
		"tools/gomad3/internal/gomadtool/conformance/testdata/negative.go",
		"tools/gomad3/cmd/gomad/testdata/source.go",
		"tools/gomad3/deterministicio/testdata/source.go",
		"tools/gomad3/internal/compatibilitypack/testdata/source.go",
		"tools/gomad3/testdata/source.go",
		"tools/gomad3/qualification/corpus/source.go",
		"tools/gomad3/toolchain/runtime/overlay/src/os/gomad.go",
	} {
		repo.write(path, "package fixture\n")
	}
	output := repo.make("lint-code-fast")
	require.Empty(t, repo.calls())
	for _, owner := range []string{"retained evidence", "simulation fixture", "integration fixture", "Gomad qualification fixture", "runtime overlay"} {
		require.Contains(t, output, owner)
	}
}

func TestLintRejectsUnclassifiedSource(t *testing.T) {
	t.Parallel()
	for _, path := range []string{
		"extra/go.mod", ".hidden/go.mod", "_hidden/go.mod",
		"tools/gomad3/internal/gomadtool/conformance/testdata/extra/go.mod",
		"tools/gomad3sim/testdata/simulation_exploration/extra/go.mod",
		"tools/gomad3integration/testdata/tagged/extra/go.mod",
		"tools/gomad3/toolchain/runtime/overlayextra/source.go",
		"tools/gomad3/toolchain/runtime/overlay/src/os/uninventoried.go",
		"tools/gomad3/record/testdata/source.go",
		"tools/gomad3/runner/.hidden/source.go",
		"tools/gomad3/runner/_hidden/source.go",
		"tools/gomad3/toolchain/runtime/testdata/unknown/source.go",
		"tools/gomad3/toolchain/runtime/testdata/vfdpointerextra/source.go",
		"tools/gomad3/toolchain/runtime/testdata/vfdpointer/nested/source.go",
		"tools/gomad3/toolchain/runtime/testdata/vfdpointer/.hidden/source.go",
		"tools/gomad3/toolchain/runtime/testdata/vfdpointer/_hidden/source.go",
		"tools/gomad3/toolchain/runtime/testdata/vfdpointer/go.mod",
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
		{"lint-code-gomad3", "tools/gomad3", "", []string{".", "./internal/gomadtool/architecture", "./runner", "./toolchain"}},
		{"lint-code-mixedbrain", "tests/mixedbrain", "", []string{"."}},
		{"lint-code-fast", "tools/gomad3", "golangci", []string{".", "./internal/gomadtool/architecture", "./runner", "./toolchain"}},
		{"lint-code-fast", "tools/gomad3", "vet", []string{".", "./internal/gomadtool/architecture", "./runner", "./toolchain"}},
	} {
		t.Run(tc.target+tc.dir+tc.fail, func(t *testing.T) {
			t.Parallel()
			repo := newLintRepo(t)
			repo.write("tools/gomad3/runner/source.go", "package sample\nvar Changed = 1\n")
			repo.env = append(repo.env, "LINT_TEST_FAIL="+tc.fail)
			output, err := repo.runMake(tc.target)
			if tc.fail == "" {
				require.NoError(t, err, output)
			} else {
				require.Error(t, err, output)
			}
			want := []lintCall{{"golangci", tc.dir, repo.lintArgs(tc.packages...)}}
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
	repo.make("-C", "tools/gomad3", "lint-code")
	require.Equal(t, []lintCall{
		{"golangci", "tools/gomad3", repo.lintArgs(".", "./internal/gomadtool/architecture", "./runner", "./toolchain")},
		{"vet", "tools/gomad3", repo.vetArgs(".", "./internal/gomadtool/architecture", "./runner", "./toolchain")},
	}, repo.calls())
}

func TestGomadDefaultRemainsGenerate(t *testing.T) {
	t.Parallel()
	repo := newLintRepo(t)
	repo.write("default-goal.mk", "inspect-default:\n\t@printf '%s\\n' '$(.DEFAULT_GOAL)'\n")
	output := repo.make("--no-print-directory", "-C", "tools/gomad3", "-f", "Makefile", "-f", "../../default-goal.mk", "inspect-default")
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
	for _, path := range []string{"tools/gomad3/go.mod", "tests/mixedbrain/go.mod"} {
		t.Run(path, func(t *testing.T) {
			t.Parallel()
			repo := newLintRepo(t)
			require.NoError(t, os.Remove(filepath.Join(repo.root, path)))
			output, err := repo.runMake("lint-code-gomad3")
			require.Error(t, err)
			require.Contains(t, output, path)
			require.Empty(t, repo.calls())
		})
	}
}

func TestRootFixturePrefixNearMissRemainsLive(t *testing.T) {
	t.Parallel()
	repo := newLintRepo(t)
	repo.write("tools/gomad3integration/testdata/taggedextra/source.go", "package sample\n")
	repo.write("tools/gomad3sim/testdata/simulation_exploration_extra/source.go", "package sample\n")
	repo.make("lint-code-fast")
	require.Equal(t, []lintCall{
		{"golangci", ".", repo.lintArgs("./tools/gomad3integration/testdata/taggedextra", "./tools/gomad3sim/testdata/simulation_exploration_extra")},
		{"vet", ".", repo.vetArgs("./tools/gomad3integration/testdata/taggedextra", "./tools/gomad3sim/testdata/simulation_exploration_extra")},
	}, repo.calls())
}

func TestLintRejectsInvalidOwnership(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct{ path, contents string }{
		{"tools/gomad3/toolchain/version/version.json", `{"overlay_allowlist":[]}`},
		{"tools/gomad3/toolchain/version/version.json", `{"overlay_allowlist":["../outside.go"]}`},
		{"tools/gomad3/internal/gomadtool/architecture/architecture.go", "package architecture\nvar sourceExclusions = paths()\nvar expectedModules = map[string]bool{\"testdata/go.mod\":true}\n"},
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
		{"gomad3.yml", "host-tools-linux", "Lint Gomad host module", "tools/gomad3", []string{".", "./internal/gomadtool/architecture", "./runner", "./toolchain"}},
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
				{"golangci", tc.module, repo.lintArgs(tc.packages...)},
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
	data, err := json.Marshal(lintCall{tool, filepath.ToSlash(rel), os.Args[separator+1:]})
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
	for _, path := range []string{"Makefile", "tools/gomad3/Makefile", "tools/gomad3/version_generated.mk", "tools/gomad3/internal/gomadtool/architecture/architecture.go"} {
		data, err := os.ReadFile(filepath.Join(root, path))
		require.NoError(t, err)
		if filepath.Base(path) == "architecture.go" {
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
	for _, path := range []string{"go.mod", "tools/gomad3/go.mod", "tests/mixedbrain/go.mod", "tools/gomad3/internal/gomadtool/conformance/testdata/go.mod", "tools/gomad3/qualification/corpus/go.mod"} {
		repo.write(path, "module example.invalid/fixture\n\ngo 1.27.0\n")
	}
	repo.write("go.mod", "module example.invalid/root\n\ngo 1.27.0\n")
	for _, dir := range []string{"proto/internal", "chasm/lib", "tools/gomad3/toolchain/runtime/overlay", "tools/gomad3/deterministicio/boundary", "tools/gomad3/internal/gomadtool/generation/boundary", "tools/gomad3/internal/compatibilitypack/requests", "tools/gomad3/internal/compatibilitypack/authoring"} {
		require.NoError(t, os.MkdirAll(filepath.Join(repo.root, dir), 0o700))
	}
	for _, path := range []string{"ordinary/source.go", "staged/staged.go", "deleted/only.go", "renameold/only.go", "tools/gomad3/runner/source.go", "tools/gomad3/toolchain/source.go", "tests/mixedbrain/source.go", ".github/actions/build-docker-images/scripts/source.go", "tools/gomad3sim/source.go", "tools/gomad3integration/source.go"} {
		repo.write(path, "package sample\n")
	}
	repo.write("tools/gomad3/root_test.go", "package sample\n")
	repo.write("tools/gomad3/toolchain/version/version.json", `{"overlay_allowlist":["src/os/gomad.go"]}`)
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

func (r *lintRepo) lintArgs(packages ...string) []string {
	return append([]string{"run", "--verbose", "--build-tags", "disable_grpc_modules,,test_dep,", "--timeout", "10m", "--fix=false", "--new-from-rev=" + r.base, "--config=" + filepath.Join(r.root, ".github/.golangci.yml")}, packages...)
}

func (r *lintRepo) vetArgs(packages ...string) []string {
	return append([]string{"vet", "-tags", "disable_grpc_modules,,test_dep,", "-vettool=" + filepath.Join(r.root, "bin/errortype"), "-style-check=false"}, packages...)
}
