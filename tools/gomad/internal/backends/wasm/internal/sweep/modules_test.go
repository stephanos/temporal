package sweep

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestModuleSelectionsHonorPackageFilters(t *testing.T) {
	t.Parallel()
	checkout := t.TempDir()
	for _, path := range []string{"tests/mixedbrain", "tests/mixedbrain/deeper", "tests/vendor/ignored", "tests/.hidden"} {
		dir := filepath.Join(checkout, path)
		require.NoError(t, os.MkdirAll(dir, 0o700))
		require.NoError(t, os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/test\n"), 0o600))
	}
	for _, tc := range []struct {
		name     string
		patterns []string
		want     []moduleSelection
	}{
		{"root exact", []string{"./tests"}, []moduleSelection{{Directory: checkout, Patterns: []string{"./tests"}}}},
		{"root recursive", []string{"./tests/..."}, []moduleSelection{
			{Directory: checkout, Patterns: []string{"./tests/..."}},
			{Directory: filepath.Join(checkout, "tests/mixedbrain"), Patterns: []string{"./..."}},
			{Directory: filepath.Join(checkout, "tests/mixedbrain/deeper"), Patterns: []string{"./..."}},
		}},
		{"nested exact", []string{"./tests/mixedbrain"}, []moduleSelection{{Directory: filepath.Join(checkout, "tests/mixedbrain"), Patterns: []string{"./."}}}},
		{"nested recursive", []string{"./tests/mixedbrain/..."}, []moduleSelection{
			{Directory: filepath.Join(checkout, "tests/mixedbrain"), Patterns: []string{"./..."}},
			{Directory: filepath.Join(checkout, "tests/mixedbrain/deeper"), Patterns: []string{"./..."}},
		}},
		{"mixed filters", []string{"./tests/mixedbrain/sub", "./tests", "./tests/mixedbrain/sub"}, []moduleSelection{
			{Directory: checkout, Patterns: []string{"./tests"}},
			{Directory: filepath.Join(checkout, "tests/mixedbrain"), Patterns: []string{"./sub"}},
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			selections, err := discoverModules(checkout, tc.patterns)
			require.NoError(t, err)
			require.Equal(t, tc.want, selections)
		})
	}
}

type moduleExecutor struct {
	fixture   *fixture
	directory string
	commands  []Command
}

func (executor *moduleExecutor) Exec(ctx context.Context, command Command) Completed {
	executor.commands = append(executor.commands, command)
	if command.Dir != executor.directory {
		return executor.fixture.Exec(ctx, command)
	}
	switch command.Args[0] {
	case "list":
		encoded, err := json.Marshal(listedPackage{ImportPath: "example.com/nested", Dir: executor.directory, TestGoFiles: []string{"nested_test.go"}})
		return Completed{Stdout: encoded, Err: err}
	case "test":
		return Completed{Stdout: []byte("TestNested\nok example.com/nested\n")}
	}
	return executor.fixture.Exec(ctx, command)
}

func TestEnumerationIncludesNestedModules(t *testing.T) {
	t.Parallel()
	f, executor := newModuleFixture(t)
	dir := executor.directory
	packages, err := Enumerate(t.Context(), executor, f.options, GuestTags(), nil)
	require.NoError(t, err)
	require.Len(t, packages, 3)
	index := slices.IndexFunc(packages, func(pkg Package) bool { return pkg.Pattern == "./tests/mixedbrain" })
	require.NotEqual(t, -1, index)
	pkg := packages[index]
	require.Equal(t, dir, pkg.ModuleDirectory)
	require.Equal(t, "./.", pkg.Target)
	require.Equal(t, []string{"TestNested"}, pkg.Tests)
	require.Equal(t, 1, pkg.ListCount)
	var nestedCommands []Command
	for _, command := range executor.commands {
		if command.Dir == dir {
			nestedCommands = append(nestedCommands, command)
		}
	}
	require.Len(t, nestedCommands, 3)
	require.Equal(t, "./...", nestedCommands[0].Args[len(nestedCommands[0].Args)-1])
	require.Contains(t, nestedCommands[1].Env, "GOOS=wasip1")
	require.Equal(t, "./.", nestedCommands[2].Args[len(nestedCommands[2].Args)-1])

	executor.commands = nil
	options := f.options
	options.Packages = []string{"./tests/mixedbrain"}
	packages, err = Enumerate(t.Context(), executor, options, GuestTags(), nil)
	require.NoError(t, err)
	require.Len(t, packages, 1)
	for _, command := range executor.commands {
		require.Equal(t, dir, command.Dir)
	}
}

func newModuleFixture(t *testing.T) (*fixture, *moduleExecutor) {
	t.Helper()
	f := newFixture(t)
	dir := filepath.Join(f.options.Checkout, "tests/mixedbrain")
	require.NoError(t, os.MkdirAll(dir, 0o700))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/nested\n"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "nested_test.go"), []byte("package nested\nimport \"testing\"\nfunc TestNested(t *testing.T) {}\n"), 0o600))
	return f, &moduleExecutor{fixture: f, directory: dir}
}

func TestSweepRunsNestedModuleTarget(t *testing.T) {
	t.Parallel()
	f, executor := newModuleFixture(t)
	f.options.Packages = []string{"./tests/mixedbrain"}
	sweeper := f.sweeper()
	sweeper.Executor = executor
	findings, err := sweeper.Run(t.Context())
	require.NoError(t, err)
	require.Empty(t, findings)
	var explorations []Command
	for _, command := range executor.commands {
		if command.Path == f.options.Gomad && slices.Contains(command.Args, "explore") {
			explorations = append(explorations, command)
		}
	}
	require.Len(t, explorations, 2)
	for _, command := range explorations {
		require.Equal(t, executor.directory, flagValue(command.Args, "--working-dir"))
		index := slices.Index(command.Args, "go-test")
		require.Positive(t, index)
		require.Equal(t, "./.", command.Args[index+1])
	}
	encoded, err := os.ReadFile(SidecarPath(f.options.Out))
	require.NoError(t, err)
	var sidecar Sidecar
	require.NoError(t, json.Unmarshal(encoded, &sidecar))
	require.Len(t, sidecar.Rows, 1)
	require.Equal(t, "./tests/mixedbrain", sidecar.Rows[0].Package)
}
