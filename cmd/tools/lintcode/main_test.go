package main

import (
	"encoding/json"
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
