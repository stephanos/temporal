package veil

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	protocolchecker "go.temporal.io/server/tools/umpire3/protocol/checker"
)

func TestPinnedVeilToolchainAndDependencyClosure(t *testing.T) {
	toolchain, err := os.ReadFile("../../model/lean-toolchain")
	require.NoError(t, err)
	require.Equal(t, "leanprover/lean4:v4.28.0\n", string(toolchain))

	manifestJSON, err := os.ReadFile("../../model/lake-manifest.json")
	require.NoError(t, err)
	var manifest struct {
		Version  string `json:"version"`
		Packages []struct {
			Name string `json:"name"`
			Rev  string `json:"rev"`
		} `json:"packages"`
	}
	require.NoError(t, json.Unmarshal(manifestJSON, &manifest))
	require.Equal(t, "1.1.0", manifest.Version)
	revisions := make(map[string]string, len(manifest.Packages))
	for _, dependency := range manifest.Packages {
		require.NotEmpty(t, dependency.Name)
		require.Len(t, dependency.Rev, 40)
		require.NotContains(t, revisions, dependency.Name)
		revisions[dependency.Name] = dependency.Rev
	}
	require.Equal(t, protocolchecker.VeilBackendRevision, revisions["veil"])
}
