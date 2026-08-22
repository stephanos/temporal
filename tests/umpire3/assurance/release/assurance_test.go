package release

import (
	"bytes"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"go.temporal.io/server/tests/umpire3/assurance/migration"
	protocolcatalog "go.temporal.io/server/tests/umpire3/protocol/catalog"
	protocolexperiment "go.temporal.io/server/tests/umpire3/protocol/experiment"
	protocolrelease "go.temporal.io/server/tests/umpire3/protocol/release"
)

func TestBindReplacesAuthoredAssuranceAndRejectsResealedRelabel(t *testing.T) {
	manifest, experiments := releaseFixture(t)
	for index := range manifest.Assurance.Goals {
		manifest.Assurance.Goals[index].Omissions = nil
	}
	manifest.Assurance, _ = protocolrelease.SealReleaseAssurance(manifest.Assurance)
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
	bound.Assurance, err = protocolrelease.SealReleaseAssurance(bound.Assurance)
	require.NoError(t, err)
	require.ErrorContains(t, ValidateAgainstCurrent(bound), "assurance graph")
}

func TestBindArtifactBindingsReplacesStaleBindings(t *testing.T) {
	encoded, err := os.ReadFile("testdata/generated/umpire3-1.3.json")
	require.NoError(t, err)
	manifest, err := protocolrelease.DecodeReleaseManifest(encoded)
	require.NoError(t, err)
	manifest.CatalogHash = "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	manifest.Experiments = map[string]protocolrelease.ReleaseExperiment{
		"stale": {
			SemanticHash: "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			Digest:       "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
		},
	}
	manifest.ProofManifests = []protocolrelease.ReleaseProofManifest{{
		Identifier: "stale",
		Digest:     "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
	}}

	experiments := make([]protocolexperiment.Experiment, 0, 2)
	for _, path := range []string{"../../testdata/generated/nexus-cancellation.json", "../../testdata/generated/update-lifecycle.json"} {
		experimentBytes, readErr := os.ReadFile(path)
		require.NoError(t, readErr)
		experiment, decodeErr := protocolexperiment.DecodeExperiment(bytes.NewReader(experimentBytes), protocolexperiment.DefaultDecodeLimit)
		require.NoError(t, decodeErr)
		experiments = append(experiments, experiment)
	}

	bound, err := BindArtifactBindings(manifest, experiments)
	require.NoError(t, err)
	require.NoError(t, ValidateArtifactBindingsAgainstCurrent(bound))
	for _, experiment := range experiments {
		digest, digestErr := experiment.Digest()
		require.NoError(t, digestErr)
		require.Equal(t, protocolrelease.ReleaseExperiment{
			SemanticHash: experiment.Model.SemanticHash,
			Digest:       digest,
		}, bound.Experiments[experiment.ExperimentID])
	}
	require.Len(t, bound.ProofManifests, 4)
}

func TestBindUsesTypedArtifactEvidence(t *testing.T) {
	manifest, experiments := releaseFixture(t)
	ledger, ledgerBytes, err := migration.DefaultLedger()
	require.NoError(t, err)
	bound, err := Bind(manifest, experiments, ledger, ledgerBytes)
	require.NoError(t, err)

	nodes := make(map[string]protocolrelease.ReleaseEvidenceNode, len(bound.Assurance.Nodes))
	for _, node := range bound.Assurance.Nodes {
		nodes[node.Identifier] = node
	}
	require.Equal(t, protocolcatalog.ResultClassCompositionProved, nodes["composition"].ResultClass)
	require.Equal(t, protocolcatalog.TrustBadgeKernelWithDeclaredAxioms, nodes["composition"].TrustBadge)
	require.Equal(t, protocolcatalog.ResultClassImplementationConforming, nodes["migration"].ResultClass)
	require.Equal(t, protocolcatalog.TrustBadgeTestedInstance, nodes["migration"].TrustBadge)
	require.Equal(t, bound.Migration.LedgerHash, nodes["migration"].Digest)
	require.Equal(t, bound.CheckerCoverageHash, nodes["checker-coverage"].Digest)
	require.Equal(t, protocolcatalog.ResultClassImplementationConforming, nodes["clock-skew-audit"].ResultClass)
	require.Equal(t, protocolcatalog.TrustBadgeTestedInstance, nodes["clock-skew-audit"].TrustBadge)
	require.Equal(t, protocolcatalog.ResultClassTraceWitness, nodes["approved-mutation-audit"].ResultClass)
	require.Equal(t, protocolcatalog.TrustBadgeTestedInstance, nodes["approved-mutation-audit"].TrustBadge)
	require.Equal(t, protocolcatalog.ResultClassMetadataValidated, nodes["developer-ux"].ResultClass)
	require.Equal(t, protocolcatalog.TrustBadgeTestedInstance, nodes["developer-ux"].TrustBadge)
	require.Equal(t, protocolcatalog.ResultClassMetadataValidated, nodes["documentation"].ResultClass)
	require.Equal(t, protocolcatalog.TrustBadgeTestedInstance, nodes["documentation"].TrustBadge)
	require.Equal(t, protocolcatalog.ResultClassMetadataValidated, nodes["native-scale-benchmark"].ResultClass)
	require.Equal(t, protocolcatalog.TrustBadgeTestedInstance, nodes["native-scale-benchmark"].TrustBadge)
	require.Equal(t, protocolcatalog.ResultClassImplementationConforming, nodes["resilience-audit"].ResultClass)
	require.Equal(t, protocolcatalog.TrustBadgeTestedInstance, nodes["resilience-audit"].TrustBadge)
	require.Equal(t, protocolcatalog.ResultClassMetadataValidated, nodes["semantic-mutation-portfolio"].ResultClass)
	require.Equal(t, protocolcatalog.TrustBadgeTestedInstance, nodes["semantic-mutation-portfolio"].TrustBadge)
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
	bound.Experiments[experiments[0].ExperimentID] = protocolrelease.ReleaseExperiment{
		SemanticHash: experiments[0].Model.SemanticHash,
		Digest:       "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
	}
	require.ErrorContains(t, ValidateAgainstCurrent(bound), "mutation gate source digest")
}

func releaseFixture(t *testing.T) (protocolrelease.ReleaseManifest, []protocolexperiment.Experiment) {
	t.Helper()
	encoded, err := os.ReadFile("testdata/generated/umpire3-1.3.json")
	require.NoError(t, err)
	manifest, err := protocolrelease.DecodeReleaseManifest(encoded)
	require.NoError(t, err)
	experiments := make([]protocolexperiment.Experiment, 0, 2)
	for _, path := range []string{"../../testdata/generated/nexus-cancellation.json", "../../testdata/generated/update-lifecycle.json"} {
		encoded, readErr := os.ReadFile(path)
		require.NoError(t, readErr)
		experiment, decodeErr := protocolexperiment.DecodeExperiment(bytes.NewReader(encoded), protocolexperiment.DefaultDecodeLimit)
		require.NoError(t, decodeErr)
		experiments = append(experiments, experiment)
	}
	return manifest, experiments
}

func findGoal(t *testing.T, manifest protocolrelease.ReleaseManifest, identifier string) protocolrelease.ReleaseEvidenceGoal {
	t.Helper()
	for _, candidate := range manifest.Assurance.Goals {
		if candidate.Identifier == identifier {
			return candidate
		}
	}
	require.FailNow(t, "release assurance goal not found", identifier)
	return protocolrelease.ReleaseEvidenceGoal{}
}
