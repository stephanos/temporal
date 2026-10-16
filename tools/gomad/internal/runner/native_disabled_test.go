//go:build !gomad_native

package runner

import (
	"testing"

	testbackend "go.temporal.io/server/tools/gomad/internal/backends/testfixture"
)

const nativeTestsEnabled = false

func replayNativeExecutionProvider(t *testing.T) *testbackend.Provider {
	t.Helper()
	return testbackend.New(testbackend.Config{})
}
