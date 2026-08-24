package api

import (
	"context"
	"flag"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

var rewriteFixture = flag.Bool("rewrite", false, "rewrite generator golden files")

func TestBasicFixture(t *testing.T) {
	outputRoot := t.TempDir()
	arguments := []string{
		"generate",
		"--descriptor", "fixture=testdata/basic/input.pb",
		"--source", "Public=public/",
		"--source", "Internal=internal/",
		"--default-source", "External",
		"--lean-root", "Fixture",
		"--output-root", outputRoot,
	}
	require.NoError(t, Run(context.Background(), arguments, io.Discard))

	actual := readTree(t, outputRoot)
	expectedRoot := filepath.Join("testdata", "basic", "expected")
	if *rewriteFixture {
		require.NoError(t, os.RemoveAll(expectedRoot))
		for path, encoded := range actual {
			target := filepath.Join(expectedRoot, filepath.FromSlash(path))
			require.NoError(t, os.MkdirAll(filepath.Dir(target), 0o755))
			require.NoError(t, os.WriteFile(target, encoded, 0o644))
		}
		return
	}
	require.Equal(t, readTree(t, expectedRoot), actual)
}

func readTree(t *testing.T, root string) map[string][]byte {
	t.Helper()
	result := make(map[string][]byte)
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		encoded, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		result[filepath.ToSlash(relative)] = encoded
		return nil
	})
	require.NoError(t, err)
	return result
}
