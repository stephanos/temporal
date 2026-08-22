package checker

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNexusProofManifestTracksSemanticsAndAssumptions(t *testing.T) {
	manifestBytes := defaultNexusProofManifestJSON
	var raw map[string]any
	require.NoError(t, json.Unmarshal(manifestBytes, &raw))
	require.Equal(t, "refinement-proved", raw["resultClass"])
	require.Equal(t, "kernel-with-declared-axioms", raw["trustBadge"])
	require.Equal(t, []any{"propext"}, raw["axioms"])
	require.NotEmpty(t, raw["statement"])
	require.NotEmpty(t, raw["sourceDependencies"])
	require.NotEmpty(t, raw["sourceDigest"])
	require.NotEmpty(t, raw["dependencyDigest"])

	var manifest ProofManifest
	require.NoError(t, json.Unmarshal(manifestBytes, &manifest))
	require.NoError(t, manifest.Validate())

	experimentBytes, err := os.ReadFile("../../testdata/generated/nexus-cancellation.json")
	require.NoError(t, err)
	var experiment Experiment
	require.NoError(t, json.Unmarshal(experimentBytes, &experiment))
	require.Equal(t, experiment.Model.SemanticHash, manifest.SemanticHash)
	require.Equal(t, experiment.Scope.Assumptions[0].StatementHash, manifest.Assumptions[0].StatementHash)

	before, err := manifest.Digest()
	require.NoError(t, err)
	manifest.Identifier += "-changed"
	after, err := manifest.Digest()
	require.NoError(t, err)
	require.NotEqual(t, before, after)
}

func TestDefaultProofManifestsIncludeMutationRejectionEvidence(t *testing.T) {
	manifests, err := DefaultProofManifests()
	require.NoError(t, err)

	byIdentifier := make(map[string]ProofManifest, len(manifests))
	for _, manifest := range manifests {
		byIdentifier[manifest.Identifier] = manifest
	}
	require.Equal(t,
		"Umpire3.Temporal.Refinement.NexusCancellationFencing.mutationBreaksDeclaredSimulation",
		byIdentifier["nexus-cancellation-mutation-rejection-v1"].Theorem)
	require.Equal(t, ResultClassRefinementProved,
		byIdentifier["nexus-cancellation-mutation-rejection-v1"].ResultClass)
	require.Equal(t,
		"Umpire3.Temporal.Mutations.NexusCancellationFencing.exactMutationWitness",
		byIdentifier["nexus-cancellation-exact-witness-v1"].Theorem)
	require.Equal(t, ResultClassTraceWitness,
		byIdentifier["nexus-cancellation-exact-witness-v1"].ResultClass)
}

func TestProofManifestRejectsNonProofClaimsAndTampering(t *testing.T) {
	manifestBytes := defaultNexusProofManifestJSON
	var manifest ProofManifest
	require.NoError(t, json.Unmarshal(manifestBytes, &manifest))
	require.NoError(t, manifest.Validate())

	for _, resultClass := range []ResultClass{
		ResultClassSampledNoCounterexample,
		ResultClassBoundedSafe,
		ResultClassFiniteExhaustive,
		ResultClassExternalNoCounterexample,
		ResultClassImplementationConforming,
		ResultClassMetadataValidated,
		ResultClassUnknown,
	} {
		t.Run(string(resultClass), func(t *testing.T) {
			changed := manifest
			changed.ResultClass = resultClass
			require.Error(t, changed.Validate())
		})
	}

	t.Run("statement", func(t *testing.T) {
		changed := manifest
		changed.Statement += " altered"
		require.Error(t, changed.Validate())
	})

	t.Run("trust badge", func(t *testing.T) {
		changed := manifest
		changed.TrustBadge = TrustBadgeKernel
		require.Error(t, changed.Validate())
	})

	for _, forbiddenAxiom := range []string{"sorryAx", "Lean.ofReduceBool"} {
		t.Run(forbiddenAxiom, func(t *testing.T) {
			changed := manifest
			changed.Axioms = []string{forbiddenAxiom}
			require.ErrorContains(t, changed.Validate(), "forbidden axiom")
		})
	}

	t.Run("source graph", func(t *testing.T) {
		changed := manifest
		changed.SourceDependencies = append([]SourceDependency(nil), manifest.SourceDependencies...)
		changed.SourceDependencies[0].Digest = "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
		require.Error(t, changed.Validate())
	})
}

func TestProofManifestRejectsLegacySelfAttestation(t *testing.T) {
	_, err := DecodeProofManifest(strings.NewReader(`{
		"formatVersion":"umpire3/v2",
		"identifier":"legacy",
		"theorem":"Some.Unresolved.Theorem",
		"statementHash":"sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		"semanticHash":"sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
		"leanVersion":"4.33.0",
		"assumptions":[]
	}`), DefaultDecodeLimit)
	require.Error(t, err)
}

func TestCompositionManifestsConsumeSameTaskDeliveryGuarantee(t *testing.T) {
	var nexus ProofManifest
	encoded := defaultNexusProofManifestJSON
	require.NoError(t, json.Unmarshal(encoded, &nexus))

	var update ProofManifest
	encoded = defaultUpdateProofManifestJSON
	require.NoError(t, json.Unmarshal(encoded, &update))
	require.NoError(t, update.Validate())

	providerHash := update.Assumptions[0].StatementHash
	require.Equal(t, "task-delivery.current-completion-only", update.Assumptions[0].Identifier)
	require.Equal(t, update.Assumptions[0], nexus.Assumptions[2])
	changedProviderHash := "sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"
	require.NotEqual(t, providerHash, changedProviderHash)
}
