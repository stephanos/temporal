package cli

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func entries(t *testing.T, directory string) []string {
	t.Helper()
	listed, err := os.ReadDir(directory)
	require.NoError(t, err)
	var names []string
	for _, entry := range listed {
		names = append(names, entry.Name())
	}
	return names
}

// Publishing creates the name once with the bytes, 0644, and leaves no temporary file; the same
// bytes again are already published; other bytes are a conflict that changes nothing.
func TestPublishIsExclusiveAndIdempotent(t *testing.T) {
	root := t.TempDir()
	publication, err := Publish(t.Context(), root, "abc.json", []byte("receipt\n"))
	require.NoError(t, err)
	resolved, err := Resolve(root)
	require.NoError(t, err)
	require.Equal(t, Publication{Status: StatusPublished, Path: filepath.Join(resolved, "abc.json")}, publication)
	info, err := os.Stat(publication.Path)
	require.NoError(t, err)
	require.Equal(t, os.FileMode(0o644), info.Mode().Perm())
	require.Equal(t, []string{"abc.json"}, entries(t, root))

	again, err := Publish(t.Context(), root, "abc.json", []byte("receipt\n"))
	require.NoError(t, err)
	require.Equal(t, StatusAlreadyPublished, again.Status)

	for name, other := range map[string]string{"other bytes": "tampered\n", "a longer file": "receipt\nand more\n", "a prefix": "receipt"} {
		t.Run(name, func(t *testing.T) {
			_, err := Publish(t.Context(), root, "abc.json", []byte(other))
			var conflict *ConflictError
			require.ErrorAs(t, err, &conflict)
			require.Contains(t, conflict.Detail, "other bytes")
			stored, err := os.ReadFile(publication.Path)
			require.NoError(t, err)
			require.Equal(t, "receipt\n", string(stored), "a conflict is never overwritten")
		})
	}
	require.Equal(t, []string{"abc.json"}, entries(t, root), "no temporary file is left behind")
}

// A name that is not a regular file is a conflict and is left as it was.
func TestPublishRefusesANameThatIsNotARegularFile(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "elsewhere.json")
	require.NoError(t, os.WriteFile(target, []byte("receipt\n"), 0o644))
	require.NoError(t, os.Symlink(target, filepath.Join(root, "linked.json")))
	require.NoError(t, os.Mkdir(filepath.Join(root, "directory.json"), 0o755))
	for name, detail := range map[string]string{"linked.json": "not a regular file", "directory.json": "not a regular file"} {
		_, err := Publish(t.Context(), root, name, []byte("receipt\n"))
		var conflict *ConflictError
		require.ErrorAs(t, err, &conflict, name)
		require.Contains(t, conflict.Detail, detail)
	}
	destination, err := os.Readlink(filepath.Join(root, "linked.json"))
	require.NoError(t, err)
	require.Equal(t, target, destination, "the symlink is left as it was, even though its target holds the bytes")
}

// The root must exist and be a directory, the name must be bare, and a cancelled context publishes
// nothing and leaves nothing.
func TestPublishRefusesBeforeWriting(t *testing.T) {
	root := t.TempDir()
	for _, name := range []string{"", ".", "..", "a/b.json", "../b.json"} {
		_, err := Publish(t.Context(), root, name, []byte("x"))
		require.ErrorContains(t, err, "not a bare file name", name)
	}
	_, err := Publish(t.Context(), filepath.Join(root, "missing"), "a.json", []byte("x"))
	require.Error(t, err)
	file := filepath.Join(root, "file")
	require.NoError(t, os.WriteFile(file, nil, 0o644))
	_, err = Publish(t.Context(), file, "a.json", []byte("x"))
	require.ErrorContains(t, err, "not a directory")

	cancelled, cancel := context.WithCancel(t.Context())
	cancel()
	_, err = Publish(cancelled, root, "a.json", []byte("x"))
	require.ErrorContains(t, err, "interrupted before publishing")
	require.Equal(t, []string{"file"}, entries(t, root), "nothing is published and no temporary file is left")

	// Cancelling after the link changes nothing: the bytes stand.
	running, stop := context.WithCancel(t.Context())
	publication, err := Publish(running, root, "a.json", []byte("x"))
	require.NoError(t, err)
	stop()
	stored, err := os.ReadFile(publication.Path)
	require.NoError(t, err)
	require.Equal(t, "x", string(stored))
}

// A root reached through a symlink is published into where it really is.
func TestPublishResolvesTheRoot(t *testing.T) {
	actual := t.TempDir()
	link := filepath.Join(t.TempDir(), "root")
	require.NoError(t, os.Symlink(actual, link))
	publication, err := Publish(t.Context(), link, "a.json", []byte("x"))
	require.NoError(t, err)
	resolved, err := Resolve(actual)
	require.NoError(t, err)
	require.Equal(t, filepath.Join(resolved, "a.json"), publication.Path)
}
