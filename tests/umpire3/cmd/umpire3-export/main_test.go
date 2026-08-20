package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestHashSourcesIncludesPathsAndContents(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(root, "a"), []byte("one"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(root, "b"), []byte("two"), 0o600))

	first, err := hashSources(root, []string{"a", "b"})
	require.NoError(t, err)
	second, err := hashSources(root, []string{"b", "a"})
	require.NoError(t, err)
	require.NotEqual(t, first, second)

	require.NoError(t, os.WriteFile(filepath.Join(root, "b"), []byte("changed"), 0o600))
	changed, err := hashSources(root, []string{"a", "b"})
	require.NoError(t, err)
	require.NotEqual(t, first, changed)
}
