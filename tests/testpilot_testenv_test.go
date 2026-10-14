//go:build test_dep && integration

package tests

import (
	"testing"

	"go.temporal.io/server/tests/testcore"
)

// newTestpilotTestEnvironment is one dedicated cluster over in-memory persistence, with whatever
// dynamic configuration the caller sets at construction: a switch value's settings apply cluster
// wide, so a Case provisioned into its own namespace runs under them.
func newTestpilotTestEnvironment(t *testing.T, options ...testcore.TestOption) *testcore.TestEnv {
	t.Helper()
	return testcore.NewEnv(t, append([]testcore.TestOption{testcore.WithInMemorySQLitePersistence()}, options...)...)
}
