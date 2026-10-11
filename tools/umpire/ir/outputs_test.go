package ir

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIRPathsDiscoversManagedFeatureTrees(t *testing.T) {
	root := t.TempDir()
	write := func(name string) string {
		path := filepath.Join(root, filepath.FromSlash(name))
		require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
		require.NoError(t, os.WriteFile(path, []byte("{}"), 0o644))
		return path
	}
	first := write("first/generated/ir.json")
	write("first/generated/ir.lint.json")
	write("first/generated/manifest.json")
	write("first/generated/witness.case.json")
	write("first/source.json")
	sibling := write("second/generated/ir.json")
	paths, err := IRPaths(root)
	require.NoError(t, err)
	require.Equal(t, []string{first, sibling}, paths)
	paths, err = IRPaths(filepath.Dir(first))
	require.NoError(t, err)
	require.Equal(t, []string{first}, paths)
	added := write("third/generated/ir.json")
	paths, err = IRPaths(root)
	require.NoError(t, err)
	require.Equal(t, []string{first, sibling, added}, paths)
}

func TestFeaturePathsPreserveLogicalNames(t *testing.T) {
	root := t.TempDir()
	main := filepath.Join(root, FeatureDirectory, "example_form", "generated", "ir.json")
	legacy := filepath.Join(filepath.Dir(main), "legacy", "example-form-race.json")
	for _, path := range []string{main, legacy} {
		require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
		require.NoError(t, os.WriteFile(path, []byte("{}"), 0o644))
	}
	require.Equal(t, main, ModelPath(root, "example-form.json"))
	require.Equal(t, legacy, ModelPath(root, "example-form-race.json"))
	require.Equal(t, "example-form.json", ModelName(main))
	require.Equal(t, "example-form-race.json", ModelName(legacy))
	require.Equal(t, filepath.Join(filepath.Dir(main), "finish.case.json"), CasePath(root, "example-form.json", "finish"))
	require.Equal(t, filepath.Join(filepath.Dir(legacy), "example-form-race-finish-case.json"), CasePath(root, "example-form-race.json", "finish"))
	paths, err := IRPaths(filepath.Dir(filepath.Dir(main)))
	require.NoError(t, err)
	require.Equal(t, []string{main, legacy}, paths)
	paths, err = IRPaths(filepath.Dir(main))
	require.NoError(t, err)
	require.Equal(t, []string{main}, paths)
}

func TestNestedFeatureFolderScopeRetainsLogicalIdentity(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, FeatureDirectory, "example", "form", "generated", "ir.json")
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte("{}"), 0o644))
	require.Equal(t, path, ModelPath(root, "example-form.json"))
	require.Equal(t, "example-form.json", ModelName(path))
	scope := FeatureRoot(filepath.Dir(path))
	paths, err := IRPaths(scope)
	require.NoError(t, err)
	require.Equal(t, []string{path}, paths)
}
