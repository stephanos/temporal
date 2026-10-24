//go:build test_dep && integration

package tests

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"go.temporal.io/server/chasm/lib/activity"
	"go.temporal.io/server/common/dynamicconfig"
	"go.temporal.io/server/tests/testcore"
)

// newTestpilotTestEnvironment is one dedicated cluster over in-memory persistence, with whatever
// dynamic configuration the caller sets at construction: a switch value's settings apply cluster
// wide, so everything provisioned into its own namespace runs under them.
func newTestpilotTestEnvironment(t *testing.T, options ...testcore.TestOption) *testcore.TestEnv {
	t.Helper()
	return testcore.NewEnv(t, append([]testcore.TestOption{testcore.WithInMemorySQLitePersistence()}, options...)...)
}

// activityEnvironment is a cluster with standalone activities and their operator commands enabled.
func activityEnvironment(t *testing.T) *testcore.TestEnv {
	t.Helper()
	return newTestpilotTestEnvironment(t,
		testcore.WithDynamicConfig(dynamicconfig.EnableChasm, true),
		testcore.WithDynamicConfig(activity.Enabled, true),
		testcore.WithDynamicConfig(activity.EnableStandaloneActivityOperatorCommands, true))
}

// buildCommand builds one command of this module into the test's temporary directory.
func buildCommand(t *testing.T, pkg string) string {
	t.Helper()
	binary := filepath.Join(t.TempDir(), filepath.Base(pkg))
	build := exec.Command("go", "build", "-o", binary, "go.temporal.io/server/"+pkg)
	build.Env = os.Environ()
	output, err := build.CombinedOutput()
	require.NoError(t, err, "build %s: %s", pkg, output)
	return binary
}
