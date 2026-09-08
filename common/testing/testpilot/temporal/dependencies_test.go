package temporal

import (
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestServerTransportContainsNoNexusKnowledge(t *testing.T) {
	_, current, _, ok := runtime.Caller(0)
	require.True(t, ok)
	root := filepath.Join(filepath.Dir(current), "server")
	require.NoError(t, filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		require.NoError(t, err)
		if entry.IsDir() {
			return nil
		}
		require.NotContains(t, strings.ToLower(entry.Name()), "nexus", path)
		contents, err := os.ReadFile(path)
		require.NoError(t, err)
		require.NotContains(t, strings.ToLower(string(contents)), "nexus", path)
		return nil
	}))
}
