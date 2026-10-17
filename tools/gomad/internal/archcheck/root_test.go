package archcheck

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSymlinkRootPreservesArchitectureChecks(t *testing.T) {
	root := fixture(t, map[string]string{
		"pure/source.go": "package pure; import \"os\"; func F() { _ = os.Getenv(\"HOME\") }",
	})
	alias := filepath.Join(t.TempDir(), "root")
	require.NoError(t, os.Symlink(root, alias))
	pure := func(path string) bool { return path == "pure/source.go" }

	t.Run("inventory", func(t *testing.T) {
		inventory, findings, err := discover(alias, goCommand(), host, false)
		require.NoError(t, err)
		require.Empty(t, findings)
		require.Equal(t, []string{"pure/source.go"}, inventory.Sources)
		require.Len(t, inventory.Packages, 1)
		require.Equal(t, "example.invalid/check/pure", inventory.Packages[0].ImportPath)
	})
	t.Run("structure", func(t *testing.T) {
		findings, err := Check(alias, goCommand(), host, Config{Module: "example.invalid/check", PureFiles: pure})
		require.NoError(t, err)
		require.Equal(t, []Finding{{Category: "pure-import", Path: "pure/source.go", Detail: "os"}}, findings)
	})
	t.Run("effects", func(t *testing.T) {
		findings, err := CheckPurity(alias, goCommand(), host, pure)
		require.NoError(t, err)
		requireCategories(t, findings, "host-effect")
	})
}

func TestSymlinkRootStillRejectsSourceSymlinks(t *testing.T) {
	root := fixture(t, map[string]string{"pure/source.go": "package pure"})
	alias := filepath.Join(t.TempDir(), "root")
	require.NoError(t, os.Symlink(root, alias))
	require.NoError(t, os.Symlink(filepath.Join(root, "pure/source.go"), filepath.Join(root, "pure/redirect.go")))
	require.NoError(t, os.Symlink(filepath.Join(root, "pure"), filepath.Join(root, "redirect")))
	require.NoError(t, os.Mkdir(filepath.Join(root, "nested"), 0700))
	require.NoError(t, os.Symlink(filepath.Join(root, "go.mod"), filepath.Join(root, "nested/go.mod")))

	_, findings, err := discover(alias, goCommand(), host, false)
	require.NoError(t, err)
	require.Equal(t, []Finding{
		{Category: "uncovered-source", Path: "nested/go.mod", Detail: "source symlink is not inventoried"},
		{Category: "uncovered-source", Path: "pure/redirect.go", Detail: "source symlink is not inventoried"},
		{Category: "uncovered-source", Path: "redirect", Detail: "source symlink is not inventoried"},
	}, findings)
}
