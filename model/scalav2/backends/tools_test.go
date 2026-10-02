package backends

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// needs gives a backend's tool, or skips the test where it is not installed. Under
// UMPIRE_BACKENDS=require, as run.sh sets it, a missing tool fails the test instead: an agreement
// that was not run is not reported as one.
func needs(t *testing.T, find func() (Tool, bool)) Tool {
	t.Helper()
	found, ok := find()
	switch {
	case ok:
	case os.Getenv("UMPIRE_BACKENDS") == "require":
		t.Fatalf("%s is not installed, and UMPIRE_BACKENDS=require", found.Name)
	default:
		t.Skipf("%s is not installed: its agreement is not run here; model/scalav2/backends/run.sh runs it", found.Name)
	}
	return found
}

// workDir is where a test's tool reads its export and writes its reports: a temporary directory, or,
// under UMPIRE_BACKENDS_OUT, a fresh directory of that one named after the test, which is kept.
func workDir(t *testing.T) string {
	t.Helper()
	out := os.Getenv("UMPIRE_BACKENDS_OUT")
	if out == "" {
		return t.TempDir()
	}
	dir, err := os.MkdirTemp(out, strings.NewReplacer("/", "-", " ", "_", "'", "").Replace(t.Name())+"-*")
	require.NoError(t, err)
	return dir
}

// report logs a receipt on a line of its own, which run.sh collects.
func report(t *testing.T, r Receipt) {
	t.Helper()
	t.Logf("RECEIPT %s", r)
}
