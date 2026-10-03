package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"go.temporal.io/server/tools/umpire/lower"
)

const repository = "../../../.."

// modelTree is the checked-in complete tree, which the lowering package's own test holds equal to
// what the checked IR lowers to.
func modelTree(t *testing.T) map[string][]byte {
	t.Helper()
	paths, err := filepath.Glob(filepath.Join(repository, kinds["model"].directory, "*.json"))
	require.NoError(t, err)
	files := map[string][]byte{}
	for _, path := range paths {
		files[filepath.Base(path)], err = os.ReadFile(path)
		require.NoError(t, err)
	}
	return files
}

// Every pinned tree is checked in as the complete tree's selection, so a functional fixture and the
// canary's pin of one Query are the same bytes as the model tree's Case.
func TestEveryPinnedTreeIsCheckedInAsTheModelTreesSelection(t *testing.T) {
	complete := modelTree(t)
	pinned := 0
	for name, selected := range kinds {
		if len(selected.pinned) == 0 {
			continue
		}
		pinned++
		t.Run(name, func(t *testing.T) {
			require.NoError(t, sync(repository, complete, selected, false))
			encoded, err := os.ReadFile(filepath.Join(repository, selected.directory, "manifest.json"))
			require.NoError(t, err)
			manifest, err := lower.DecodeManifest(encoded)
			require.NoError(t, err)
			require.Len(t, manifest.Queries, len(selected.pinned))
			for _, entry := range manifest.Queries {
				checkedIn, err := os.ReadFile(filepath.Join(repository, selected.directory, entry.File))
				require.NoError(t, err)
				require.Equal(t, complete[entry.File], checkedIn, entry.File)
			}
		})
	}
	require.Equal(t, 2, pinned)

	shared := "nexus-caller-syncCompletion-case.json"
	functional, err := os.ReadFile(filepath.Join(repository, kinds["functional"].directory, shared))
	require.NoError(t, err)
	canary, err := os.ReadFile(filepath.Join(repository, kinds["canary"].directory, shared))
	require.NoError(t, err)
	require.Equal(t, functional, canary, "the canary pins the functional fixture's bytes")
}

// A check of a tree that differs from the selection fails and writes nothing; an update publishes
// the whole tree, the check that follows finds no difference, and a selection that is refused
// leaves the published tree as it was.
func TestAPinnedTreeIsPublishedWholeAndThenChecksClean(t *testing.T) {
	complete := modelTree(t)
	root := t.TempDir()
	selected := kinds["canary"]
	parent := filepath.Dir(filepath.Join(root, selected.directory))
	require.NoError(t, os.MkdirAll(parent, 0o755))

	require.Error(t, sync(root, complete, selected, false), "an absent tree is stale")
	entries, err := os.ReadDir(parent)
	require.NoError(t, err)
	require.Empty(t, entries, "a failed check leaves nothing behind")

	require.NoError(t, sync(root, complete, selected, true))
	require.NoError(t, sync(root, complete, selected, false))
	manifest := filepath.Join(root, selected.directory, "manifest.json")
	published, err := os.ReadFile(manifest)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(manifest, append(published, '\n'), 0o644))
	require.ErrorContains(t, sync(root, complete, selected, false), "manifest.json is stale")
	require.NoError(t, sync(root, complete, selected, true))
	republished, err := os.ReadFile(manifest)
	require.NoError(t, err)
	require.Equal(t, published, republished)

	unknown := selected
	unknown.pinned = []lower.Selected{{Model: "nexus-caller.json", Query: "absent"}}
	require.ErrorContains(t, sync(root, complete, unknown, true), "no Model declares the selected Query")
	require.NoError(t, sync(root, complete, selected, false), "a refused selection publishes nothing")
}
