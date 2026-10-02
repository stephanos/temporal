package canary_test

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func repositoryRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	require.True(t, ok)
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}

func TestCanaryCIKeepsOfflineChecks(t *testing.T) {
	encoded, err := os.ReadFile(filepath.Join(repositoryRoot(t), ".github/workflows/umpire.yml"))
	require.NoError(t, err)
	var workflow struct {
		Jobs map[string]struct {
			Environment string
			Steps       []struct{ Run string }
		}
	}
	require.NoError(t, yaml.Unmarshal(encoded, &workflow))
	job, ok := workflow.Jobs["canary"]
	require.True(t, ok)
	require.Empty(t, job.Environment)
	var commands []string
	for _, step := range job.Steps {
		commands = append(commands, step.Run)
	}
	joined := strings.Join(commands, "\n")
	require.Contains(t, joined, "go test -count=1 -tags test_dep ./tools/canary/...")
	require.Contains(t, joined, "go test -count=1 -tags 'test_dep canary_harness' ./tools/canary/testharness/")
	require.Contains(t, joined, "make canary-build")
	require.Contains(t, joined, "make umpire-check-cases")
	require.NotContains(t, joined, "umpire-canary run")
	require.NotContains(t, joined, "umpire-canary reconcile")
}
