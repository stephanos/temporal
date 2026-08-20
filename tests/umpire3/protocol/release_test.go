package protocol

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestReleaseManifestMatchesExportedExperiments(t *testing.T) {
	var release struct {
		Release       string            `json:"release"`
		FormatVersion string            `json:"formatVersion"`
		Experiments   map[string]string `json:"experiments"`
		Profiles      []string          `json:"profiles"`
	}
	encoded, err := os.ReadFile("../testdata/umpire3-1.0.json")
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(encoded, &release))
	require.Equal(t, "umpire3/1.0", release.Release)
	require.Equal(t, FormatVersion, release.FormatVersion)
	require.ElementsMatch(t, []string{"controlled-local", "ci"}, release.Profiles)

	for _, path := range []string{"../testdata/nexus-cancellation.json", "../testdata/update-lifecycle.json"} {
		encoded, err := os.ReadFile(path)
		require.NoError(t, err)
		var experiment Experiment
		require.NoError(t, json.Unmarshal(encoded, &experiment))
		require.Equal(t, experiment.Model.SemanticHash, release.Experiments[experiment.ExperimentID])
	}
}
