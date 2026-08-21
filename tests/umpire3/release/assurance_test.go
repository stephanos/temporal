package release

import (
	"bytes"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"go.temporal.io/server/tests/umpire3/migration"
	"go.temporal.io/server/tests/umpire3/protocol"
)

func TestBindReplacesAuthoredAssuranceAndRejectsResealedRelabel(t *testing.T) {
	manifest, experiments := releaseFixture(t)
	for index := range manifest.Assurance.Goals {
		manifest.Assurance.Goals[index].Omissions = nil
	}
	manifest.Assurance, _ = protocol.SealReleaseAssurance(manifest.Assurance)
	ledger, ledgerBytes, err := migration.DefaultLedger()
	require.NoError(t, err)

	bound, err := Bind(manifest, experiments, ledger, ledgerBytes)
	require.NoError(t, err)
	require.False(t, bound.Assurance.Complete())
	require.NotEmpty(t, findGoal(t, bound, "portable-profiles").Omissions)
	require.NoError(t, ValidateAgainstCurrent(bound))

	for index := range bound.Assurance.Goals {
		bound.Assurance.Goals[index].Omissions = nil
	}
	bound.Assurance, err = protocol.SealReleaseAssurance(bound.Assurance)
	require.NoError(t, err)
	require.ErrorContains(t, ValidateAgainstCurrent(bound), "assurance graph")
}

func TestBindUsesTypedArtifactEvidence(t *testing.T) {
	manifest, experiments := releaseFixture(t)
	ledger, ledgerBytes, err := migration.DefaultLedger()
	require.NoError(t, err)
	bound, err := Bind(manifest, experiments, ledger, ledgerBytes)
	require.NoError(t, err)

	nodes := make(map[string]protocol.ReleaseEvidenceNode, len(bound.Assurance.Nodes))
	for _, node := range bound.Assurance.Nodes {
		nodes[node.Identifier] = node
	}
	require.Equal(t, protocol.ResultClassCompositionProved, nodes["composition"].ResultClass)
	require.Equal(t, protocol.TrustBadgeKernelWithDeclaredAxioms, nodes["composition"].TrustBadge)
	require.Equal(t, protocol.ResultClassImplementationConforming, nodes["migration"].ResultClass)
	require.Equal(t, protocol.TrustBadgeTestedInstance, nodes["migration"].TrustBadge)
	require.Equal(t, bound.Migration.LedgerHash, nodes["migration"].Digest)
	require.Equal(t, bound.CheckerCoverageHash, nodes["checker-coverage"].Digest)
	require.Equal(t, protocol.ResultClassImplementationConforming, nodes["clock-skew-audit"].ResultClass)
	require.Equal(t, protocol.TrustBadgeTestedInstance, nodes["clock-skew-audit"].TrustBadge)
	require.Equal(t, protocol.ResultClassTraceWitness, nodes["approved-mutation-audit"].ResultClass)
	require.Equal(t, protocol.TrustBadgeTestedInstance, nodes["approved-mutation-audit"].TrustBadge)
	require.Equal(t, protocol.ResultClassMetadataValidated, nodes["developer-ux"].ResultClass)
	require.Equal(t, protocol.TrustBadgeTestedInstance, nodes["developer-ux"].TrustBadge)
	require.Equal(t, protocol.ResultClassMetadataValidated, nodes["documentation"].ResultClass)
	require.Equal(t, protocol.TrustBadgeTestedInstance, nodes["documentation"].TrustBadge)
	require.Equal(t, protocol.ResultClassMetadataValidated, nodes["native-scale-benchmark"].ResultClass)
	require.Equal(t, protocol.TrustBadgeTestedInstance, nodes["native-scale-benchmark"].TrustBadge)
	require.Equal(t, protocol.ResultClassImplementationConforming, nodes["resilience-audit"].ResultClass)
	require.Equal(t, protocol.TrustBadgeTestedInstance, nodes["resilience-audit"].TrustBadge)
	require.Equal(t, protocol.ResultClassMetadataValidated, nodes["semantic-mutation-portfolio"].ResultClass)
	require.Equal(t, protocol.TrustBadgeTestedInstance, nodes["semantic-mutation-portfolio"].TrustBadge)
	require.Contains(t, findGoal(t, bound, "developer-authoring").Requires, "documentation")
	require.Contains(t, findGoal(t, bound, "guided-exploration").Requires, "native-scale-benchmark")
	require.Contains(t, findGoal(t, bound, "unknown-bug-exploration").Requires,
		"semantic-mutation-portfolio")
	require.Contains(t, findGoal(t, bound, "portable-profiles").Requires, "resilience-audit")
	require.Empty(t, findGoal(t, bound, "unknown-bug-exploration").Omissions)
	require.Empty(t, findGoal(t, bound, "coverage-guided-fuzzing").Omissions)
	require.Empty(t, findGoal(t, bound, "developer-authoring").Omissions)
	require.Empty(t, findGoal(t, bound, "clock-skew-safety").Omissions)
}

func TestBindIncludesWholeExperimentDigests(t *testing.T) {
	manifest, experiments := releaseFixture(t)
	ledger, ledgerBytes, err := migration.DefaultLedger()
	require.NoError(t, err)
	bound, err := Bind(manifest, experiments, ledger, ledgerBytes)
	require.NoError(t, err)

	for _, experiment := range experiments {
		digest, digestErr := experiment.Digest()
		require.NoError(t, digestErr)
		binding := bound.Experiments[experiment.ExperimentID]
		require.Equal(t, experiment.Model.SemanticHash, binding.SemanticHash)
		require.Equal(t, digest, binding.Digest)
	}
	bound.Experiments[experiments[0].ExperimentID] = protocol.ReleaseExperiment{
		SemanticHash: experiments[0].Model.SemanticHash,
		Digest:       "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
	}
	require.ErrorContains(t, ValidateAgainstCurrent(bound), "mutation gate source digest")
}

func releaseFixture(t *testing.T) (protocol.ReleaseManifest, []protocol.Experiment) {
	t.Helper()
	encoded, err := os.ReadFile("../testdata/umpire3-1.2.json")
	require.NoError(t, err)
	manifest, err := protocol.DecodeReleaseManifest(encoded)
	require.NoError(t, err)
	experiments := make([]protocol.Experiment, 0, 2)
	for _, path := range []string{"../testdata/nexus-cancellation.json", "../testdata/update-lifecycle.json"} {
		encoded, readErr := os.ReadFile(path)
		require.NoError(t, readErr)
		experiment, decodeErr := protocol.DecodeExperiment(bytes.NewReader(encoded), protocol.DefaultDecodeLimit)
		require.NoError(t, decodeErr)
		experiments = append(experiments, experiment)
	}
	return manifest, experiments
}

func findGoal(t *testing.T, manifest protocol.ReleaseManifest, identifier string) protocol.ReleaseEvidenceGoal {
	t.Helper()
	for _, candidate := range manifest.Assurance.Goals {
		if candidate.Identifier == identifier {
			return candidate
		}
	}
	require.FailNow(t, "release assurance goal not found", identifier)
	return protocol.ReleaseEvidenceGoal{}
}
