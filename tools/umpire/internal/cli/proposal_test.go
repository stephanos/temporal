package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// resolvedTemp is a temporary directory with its symlinks resolved, as the writer reports paths:
// on some systems the temporary root itself is reached through a symlink.
func resolvedTemp(t *testing.T) string {
	t.Helper()
	resolved, err := filepath.EvalSymlinks(t.TempDir())
	require.NoError(t, err)
	return resolved
}

// Every path is checked before any write: one that leaves the root writes nothing.
func TestWriteProposalsChecksEveryPathFirst(t *testing.T) {
	root := filepath.Join(resolvedTemp(t), "proposals")
	written, err := WriteProposals(root, []Proposal{
		{Candidate: "a", Path: "set-a.lean", Source: "a"},
		{Candidate: "b", Path: "../escaped.lean", Source: "b"},
	})
	require.ErrorContains(t, err, "outside the promotion root")
	require.Empty(t, written)
	require.NoDirExists(t, root)

	written, err = WriteProposals(root, []Proposal{{Candidate: "a", Path: "set-a.lean", Source: "a"}})
	require.NoError(t, err)
	require.Equal(t, map[string]string{"a": filepath.Join(root, "set-a.lean")}, written)
	source, err := os.ReadFile(written["a"])
	require.NoError(t, err)
	require.Equal(t, "a", string(source))

	written, err = WriteProposals("", []Proposal{{Candidate: "a", Path: "set-a.lean"}})
	require.NoError(t, err)
	require.Empty(t, written, "no root, nothing written")
}

// An existing destination is never replaced.
func TestWriteProposalsNeverReplacesAFile(t *testing.T) {
	root := resolvedTemp(t)
	require.NoError(t, os.WriteFile(filepath.Join(root, "set-a.lean"), []byte("reviewed"), 0o644))
	written, err := WriteProposals(root, []Proposal{{Candidate: "a", Path: "set-a.lean", Source: "new"}})
	require.ErrorIs(t, err, os.ErrExist)
	require.Empty(t, written)
	source, err := os.ReadFile(filepath.Join(root, "set-a.lean"))
	require.NoError(t, err)
	require.Equal(t, "reviewed", string(source))
}

// A root reached through a symlink into the model is under the model; a directory under the root
// that resolves out of it through a symlink is refused.
func TestProposalRootsResolveSymlinks(t *testing.T) {
	base := resolvedTemp(t)
	model := filepath.Join(base, "model")
	require.NoError(t, os.MkdirAll(model, 0o755))
	link := filepath.Join(base, "looks-outside")
	require.NoError(t, os.Symlink(model, link))
	_, err := OutsideModel("--promotion-root", filepath.Join(link, "proposals"), model)
	require.ErrorContains(t, err, "must not be under the model root")
	resolved, err := OutsideModel("--promotion-root", filepath.Join(base, "proposals"), model)
	require.NoError(t, err)
	require.Equal(t, filepath.Join(base, "proposals"), resolved)

	root := filepath.Join(base, "proposals")
	require.NoError(t, os.MkdirAll(root, 0o755))
	require.NoError(t, os.Symlink(model, filepath.Join(root, "into-model")))
	written, err := WriteProposals(root, []Proposal{{Candidate: "a", Path: "into-model/set-a.lean", Source: "a"}})
	require.ErrorContains(t, err, "resolves outside the promotion root")
	require.Empty(t, written)
	require.NoFileExists(t, filepath.Join(model, "set-a.lean"))
	// A nested path through the symlink creates nothing on its far side.
	_, err = WriteProposals(root, []Proposal{{Candidate: "b", Path: "into-model/sub/set-b.lean", Source: "b"}})
	require.ErrorContains(t, err, "resolves outside the promotion root")
	require.NoDirExists(t, filepath.Join(model, "sub"))
}
