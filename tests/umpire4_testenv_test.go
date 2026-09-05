//go:build test_dep && integration

package tests

import (
	"testing"

	"go.temporal.io/server/tests/testcore"
)

func newUmpireTestEnvironment(t *testing.T) *testcore.TestEnv {
	t.Helper()
	return testcore.NewEnv(t, testcore.WithInMemorySQLitePersistence())
}
