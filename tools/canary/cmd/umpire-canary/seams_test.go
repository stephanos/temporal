//go:build !canary_harness

package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"go.temporal.io/server/tools/canary/controller"
)

// The untagged build with no credential in its environment refuses as authority-unavailable before
// it reads the target or writes a record, and says so on stdout and stderr.
func TestTheUntaggedBuildNeedsACredential(t *testing.T) {
	recovery := filepath.Join(t.TempDir(), "recovery.json")
	var stdout, stderr bytes.Buffer
	code := Main([]string{"run", "--output", t.TempDir(), "--recovery", recovery}, &stdout, &stderr, noEnvironment, seams())
	require.Equal(t, controller.ExitFailed, code)
	summary := summaryOf(t, &stdout)
	require.Equal(t, controller.StatusAuthorityUnavailable, summary.Status)
	require.Empty(t, summary.Iterations)
	require.Contains(t, stderr.String(), controller.StatusAuthorityUnavailable)
	_, err := os.Stat(recovery)
	require.ErrorIs(t, err, os.ErrNotExist)
}
