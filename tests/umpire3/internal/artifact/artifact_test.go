package artifact

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPublishAtomicallyReplacesProtectedArtifact(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "nested", "result.json")
	require.NoError(t, Publish(path, []byte("first")))
	require.NoError(t, Publish(path, []byte("second")))
	encoded, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Equal(t, []byte("second"), encoded)
	info, err := os.Stat(path)
	require.NoError(t, err)
	require.Equal(t, os.FileMode(0o600), info.Mode().Perm())
	matches, err := filepath.Glob(filepath.Join(filepath.Dir(path), ".umpire3-artifact-*"))
	require.NoError(t, err)
	require.Empty(t, matches)
}

func TestRemoveDurablyDeletesArtifact(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "result.json")
	require.NoError(t, Publish(path, []byte("result")))
	require.NoError(t, Remove(path))
	_, err := os.Stat(path)
	require.ErrorIs(t, err, os.ErrNotExist)
	require.NoError(t, Remove(path))
}
