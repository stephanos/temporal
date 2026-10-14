package artifactio

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPublishAndRemove(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "artifact.bin")
	require.NoError(t, Publish(path, []byte("first")))

	contents, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Equal(t, []byte("first"), contents)

	require.NoError(t, Publish(path, []byte("second")))
	contents, err = os.ReadFile(path)
	require.NoError(t, err)
	require.Equal(t, []byte("second"), contents)

	if runtime.GOOS != "windows" {
		directoryInfo, err := os.Stat(filepath.Dir(path))
		require.NoError(t, err)
		require.Equal(t, os.FileMode(0o700), directoryInfo.Mode().Perm())

		artifactInfo, err := os.Stat(path)
		require.NoError(t, err)
		require.Equal(t, os.FileMode(0o600), artifactInfo.Mode().Perm())
	}

	require.NoError(t, Remove(path))
	_, err = os.Stat(path)
	require.ErrorIs(t, err, os.ErrNotExist)
	require.NoError(t, Remove(path))
}

func TestArtifactPathIsRequired(t *testing.T) {
	for _, path := range []string{"", "."} {
		t.Run(path, func(t *testing.T) {
			require.EqualError(t, Publish(path, nil), "artifact path is required")
			require.EqualError(t, Remove(path), "artifact path is required")
		})
	}
}

func TestPublishFailureLeavesExistingDestination(t *testing.T) {
	if runtime.GOOS == "windows" || os.Geteuid() == 0 {
		t.Skip("directory permissions do not reliably prevent writes")
	}
	directory := t.TempDir()
	destination := filepath.Join(directory, "artifact.bin")
	require.NoError(t, os.WriteFile(destination, []byte("original"), 0o600))
	require.NoError(t, os.Chmod(directory, 0o500))
	t.Cleanup(func() {
		require.NoError(t, os.Chmod(directory, 0o700))
	})

	err := Publish(destination, []byte("replacement"))
	require.ErrorContains(t, err, "create temporary artifact")
	require.NoError(t, os.Chmod(directory, 0o700))

	contents, err := os.ReadFile(destination)
	require.NoError(t, err)
	require.Equal(t, []byte("original"), contents)
}
