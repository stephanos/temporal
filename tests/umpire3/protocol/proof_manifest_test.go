package protocol

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNexusProofManifestTracksSemanticsAndAssumptions(t *testing.T) {
	manifestBytes, err := os.ReadFile("../testdata/nexus-proof-manifest.json")
	require.NoError(t, err)
	var manifest ProofManifest
	require.NoError(t, json.Unmarshal(manifestBytes, &manifest))
	require.NoError(t, manifest.Validate())

	experimentBytes, err := os.ReadFile("../testdata/nexus-cancellation.json")
	require.NoError(t, err)
	var experiment Experiment
	require.NoError(t, json.Unmarshal(experimentBytes, &experiment))
	require.Equal(t, experiment.Model.SemanticHash, manifest.SemanticHash)
	require.Equal(t, experiment.Scope.Assumptions[0].StatementHash, manifest.Assumptions[0].StatementHash)

	before, err := manifest.Digest()
	require.NoError(t, err)
	manifest.StatementHash = "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	after, err := manifest.Digest()
	require.NoError(t, err)
	require.NotEqual(t, before, after)
}

func TestCompositionManifestsConsumeSameTaskDeliveryGuarantee(t *testing.T) {
	var nexus ProofManifest
	encoded, err := os.ReadFile("../testdata/nexus-proof-manifest.json")
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(encoded, &nexus))

	var update ProofManifest
	encoded, err = os.ReadFile("../testdata/update-proof-manifest.json")
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(encoded, &update))
	require.NoError(t, update.Validate())

	providerHash := update.Assumptions[0].StatementHash
	require.Equal(t, "task-delivery.current-completion-only", update.Assumptions[0].Identifier)
	require.Equal(t, update.Assumptions[0], nexus.Assumptions[2])
	changedProviderHash := "sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"
	require.NotEqual(t, providerHash, changedProviderHash)
}
