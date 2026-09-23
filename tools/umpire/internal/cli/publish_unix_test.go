//go:build unix

package cli

import (
	"os"
	"path/filepath"
	"syscall"
	"testing"

	"github.com/stretchr/testify/require"
)

// A FIFO under the final name is a conflict, found without opening it, so publication never
// blocks on a reader that is not there.
func TestPublishRefusesAFIFOWithoutBlocking(t *testing.T) {
	root := t.TempDir()
	fifo := filepath.Join(root, "a.json")
	require.NoError(t, syscall.Mkfifo(fifo, 0o644))
	_, err := Publish(t.Context(), root, "a.json", []byte("x"))
	var conflict *ConflictError
	require.ErrorAs(t, err, &conflict)
	require.Contains(t, conflict.Detail, "not a regular file")
	info, err := os.Lstat(fifo)
	require.NoError(t, err)
	require.Equal(t, os.ModeNamedPipe, info.Mode().Type())
}

// Opening the existing name never follows a symlink nor blocks on a FIFO, whatever was inspected
// before it.
func TestOpenExistingNeitherFollowsNorBlocks(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "target")
	require.NoError(t, os.WriteFile(target, []byte("x"), 0o644))
	link := filepath.Join(root, "link")
	require.NoError(t, os.Symlink(target, link))
	_, err := openExisting(link)
	require.Error(t, err, "a symlink is not followed")
	fifo := filepath.Join(root, "fifo")
	require.NoError(t, syscall.Mkfifo(fifo, 0o644))
	file, err := openExisting(fifo)
	require.NoError(t, err, "a FIFO opens without blocking")
	info, err := file.Stat()
	require.NoError(t, err)
	require.False(t, info.Mode().IsRegular())
	require.NoError(t, file.Close())
}
