package vocabulary_test

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
)

// checkoutRoot locates the repository both tests in this package run against. They read
// the real tree rather than a fixture, so the path has to come from this file's own
// location and not from a working directory the test runner chose.
func checkoutRoot(t *testing.T) string {
	t.Helper()

	_, currentFile, _, ok := runtime.Caller(0)
	require.True(t, ok)
	return filepath.Clean(filepath.Join(filepath.Dir(currentFile), "..", "..", ".."))
}
