package catalog

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDefaultCompositionConnectsConsumersToProvedSharedDelivery(t *testing.T) {
	composition, err := DefaultComposition()
	require.NoError(t, err)
	require.Equal(t, ResultClassCompositionProved, composition.ResultClass)
	require.Equal(t, TrustBadgeKernelWithDeclaredAxioms, composition.TrustBadge)
	require.NotEmpty(t, composition.Proof.Declaration)
	require.NotEmpty(t, composition.Proof.Type)
	require.NotContains(t, composition.Proof.Axioms, "sorryAx")
	require.Equal(t, composition.SemanticHash, composition.SourceDigest)
	require.True(t, validHash(composition.DependencyDigest))
	require.True(t, validHash(composition.ArtifactDigest))
	require.Len(t, composition.Targets, 16)

	provider, ok := composition.Module(ModuleIDTemporalMechanismsTaskDelivery)
	require.True(t, ok)
	require.Len(t, provider.Provides, 1)
	require.NotEmpty(t, provider.Provides[0].Theorem)
	require.NotEmpty(t, provider.Provides[0].Statement)
	require.NotContains(t, provider.Provides[0].Axioms, "sorryAx")
	for _, identifier := range []ModuleID{
		ModuleIDTemporalSystemUpdateLifecycle,
		ModuleIDTemporalSystemWorkflowOwnership,
	} {
		consumer, ok := composition.Module(identifier)
		require.True(t, ok)
		require.Equal(t, provider.Identifier, consumer.Requires[0].ProviderModule)
		require.Equal(t, provider.Provides[0].StatementHash, consumer.Requires[0].StatementHash)
		require.Equal(t, provider.Provides[0].Theorem, consumer.Requires[0].Theorem)
	}
}

func TestCompositionRejectsMissingOrForgedSemanticProof(t *testing.T) {
	composition, err := DefaultComposition()
	require.NoError(t, err)
	composition.Proof.Declaration = ""
	require.ErrorContains(t, composition.Validate(), "composition proof")

	composition, err = DefaultComposition()
	require.NoError(t, err)
	composition.Proof.Type = "True"
	require.ErrorContains(t, composition.Validate(), "type hash")

	composition, err = DefaultComposition()
	require.NoError(t, err)
	composition.ArtifactDigest = "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	require.ErrorContains(t, composition.Validate(), "artifact digest")
}

func TestCompositionHasNoMissingMetadata(t *testing.T) {
	composition, err := DefaultComposition()
	require.NoError(t, err)
	require.Empty(t, composition.MissingMetadata())
}

func TestCompositionOwnsNexusClosureThroughRelationalProductSystemAndRefinement(t *testing.T) {
	composition, err := DefaultComposition()
	require.NoError(t, err)
	for _, identifier := range []ModuleID{
		"Temporal.Feature.NexusClosure",
		"Temporal.System.NexusClosure",
		"Temporal.Refinement.NexusClosure",
	} {
		module, ok := composition.Module(identifier)
		require.True(t, ok, identifier)
		require.NotEmpty(t, module.Obligations, identifier)
		for _, obligation := range module.Obligations {
			require.Equal(t, MetadataPresent, obligation.Status, identifier)
		}
	}
	var target TargetProjection
	for _, candidate := range composition.Targets {
		if candidate.Identifier == TargetIDFeatureNexus {
			target = candidate
			break
		}
	}
	require.Equal(t, []ModuleID{
		"Temporal.Feature.NexusLifecycle",
		"Temporal.Feature.NexusClosure",
		"Temporal.System.NexusClosure",
		"Temporal.Refinement.NexusClosure",
	}, target.Modules)
}

func TestCompositionOwnsNexusEvidenceThroughRelationalModules(t *testing.T) {
	composition, err := DefaultComposition()
	require.NoError(t, err)
	for _, identifier := range []ModuleID{
		"Temporal.Feature.NexusTimeout",
		"Temporal.System.NexusTimeout",
		"Temporal.Refinement.NexusTimeout",
		"Temporal.Feature.NexusActivityLink",
		"Temporal.System.NexusActivityLink",
		"Temporal.Refinement.NexusActivityLink",
	} {
		module, ok := composition.Module(identifier)
		require.True(t, ok, identifier)
		require.NotEmpty(t, module.Obligations, identifier)
		for _, obligation := range module.Obligations {
			require.Equal(t, MetadataPresent, obligation.Status, identifier)
		}
	}
	expectedTargets := map[TargetID][]ModuleID{
		TargetIDIntegrationNexusActivity: {
			"Temporal.Feature.NexusActivityLink",
			"Temporal.System.NexusActivityLink",
			"Temporal.Refinement.NexusActivityLink",
		},
		TargetIDIntegrationNexusTimeout: {
			"Temporal.Feature.NexusTimeout",
			"Temporal.System.NexusTimeout",
			"Temporal.Refinement.NexusTimeout",
		},
	}
	for identifier, expected := range expectedTargets {
		for _, target := range composition.Targets {
			if target.Identifier == identifier {
				require.Equal(t, expected, target.Modules)
				delete(expectedTargets, identifier)
				break
			}
		}
	}
	require.Empty(t, expectedTargets)
}

func TestCompositionOwnsCallbackEvidenceThroughRelationalModules(t *testing.T) {
	composition, err := DefaultComposition()
	require.NoError(t, err)
	for _, identifier := range []ModuleID{
		"Temporal.Feature.CallbackReference",
		"Temporal.System.CallbackReference",
		"Temporal.Refinement.CallbackReference",
		"Temporal.Feature.CallbackResponse",
		"Temporal.System.CallbackResponse",
		"Temporal.Refinement.CallbackResponse",
	} {
		module, ok := composition.Module(identifier)
		require.True(t, ok, identifier)
		require.NotEmpty(t, module.Obligations, identifier)
		for _, obligation := range module.Obligations {
			require.Equal(t, MetadataPresent, obligation.Status, identifier)
		}
	}
	expectedTargets := map[TargetID][]ModuleID{
		TargetIDIntegrationCallbackNexus: {
			"Temporal.Feature.CallbackReference",
			"Temporal.System.CallbackReference",
			"Temporal.Refinement.CallbackReference",
		},
		TargetIDIntegrationCallbackWorkflow: {
			"Temporal.Feature.CallbackResponse",
			"Temporal.System.CallbackResponse",
			"Temporal.Refinement.CallbackResponse",
		},
	}
	for identifier, expected := range expectedTargets {
		for _, target := range composition.Targets {
			if target.Identifier == identifier {
				require.Equal(t, expected, target.Modules)
				delete(expectedTargets, identifier)
				break
			}
		}
	}
	require.Empty(t, expectedTargets)
}

func TestCompositionOwnsWorkflowLineageThroughRelationalModules(t *testing.T) {
	composition, err := DefaultComposition()
	require.NoError(t, err)
	for _, identifier := range []ModuleID{
		"Temporal.Feature.WorkflowLineage",
		"Temporal.System.WorkflowLineage",
		"Temporal.Refinement.WorkflowLineage",
	} {
		module, ok := composition.Module(identifier)
		require.True(t, ok, identifier)
		require.NotEmpty(t, module.Obligations, identifier)
		for _, obligation := range module.Obligations {
			require.Equal(t, MetadataPresent, obligation.Status, identifier)
		}
	}
	for _, target := range composition.Targets {
		if target.Identifier == TargetIDFoundationRoutingIsolation {
			require.Equal(t, []ModuleID{
				"Temporal.Feature.RoutingIsolation",
				"Temporal.System.RoutingIsolation",
				"Temporal.Refinement.RoutingIsolation",
			}, target.Modules)
			return
		}
	}
	require.FailNow(t, "foundation routing target is missing")
}

func TestCompositionOwnsWorkflowOwnershipThroughRelationalModules(t *testing.T) {
	composition, err := DefaultComposition()
	require.NoError(t, err)
	for _, identifier := range []ModuleID{
		"Temporal.Feature.WorkflowOwnership",
		"Temporal.System.WorkflowOwnership",
		"Temporal.Refinement.WorkflowOwnership",
	} {
		module, ok := composition.Module(identifier)
		require.True(t, ok, identifier)
		require.NotEmpty(t, module.Obligations, identifier)
		for _, obligation := range module.Obligations {
			require.Equal(t, MetadataPresent, obligation.Status, identifier)
		}
	}
	for _, target := range composition.Targets {
		if target.Identifier == TargetIDFoundationOwnershipFencing {
			require.Equal(t, []ModuleID{
				"Temporal.Feature.WorkflowOwnership",
				"Temporal.Mechanisms.TaskDelivery",
				"Temporal.System.WorkflowOwnership",
				"Temporal.Refinement.WorkflowOwnership",
			}, target.Modules)
			return
		}
	}
	require.FailNow(t, "foundation ownership target is missing")
}

func TestCompositionRejectsMissingProviderCycleConflictAndVacuity(t *testing.T) {
	composition, err := DefaultComposition()
	require.NoError(t, err)
	consumer := compositionModuleIndex(t, composition, ModuleIDTemporalSystemUpdateLifecycle)
	composition.Modules[consumer].Requires[0].ProviderModule = "missing"
	require.ErrorContains(t, composition.Validate(), "missing provider")

	composition, err = DefaultComposition()
	require.NoError(t, err)
	consumer = compositionModuleIndex(t, composition, ModuleIDTemporalSystemUpdateLifecycle)
	composition.Modules[consumer].Rank = 0
	require.ErrorContains(t, composition.Validate(), "dependency cycle")

	composition, err = DefaultComposition()
	require.NoError(t, err)
	composition.Modules[1].Owns = append(composition.Modules[1].Owns, composition.Modules[2].Owns[0])
	require.ErrorContains(t, composition.Validate(), "conflicting owner")

	composition, err = DefaultComposition()
	require.NoError(t, err)
	composition.Targets[0].RetainedActions = nil
	require.ErrorContains(t, composition.Validate(), "vacuous")
}

func compositionModuleIndex(t *testing.T, composition Composition, identifier ModuleID) int {
	t.Helper()
	for index, module := range composition.Modules {
		if module.Identifier == identifier {
			return index
		}
	}
	require.FailNow(t, "composition module is missing", identifier)
	return -1
}

func TestCompositionRejectsDroppedInterferenceAction(t *testing.T) {
	composition, err := DefaultComposition()
	require.NoError(t, err)
	composition.Targets[0].RetainedActions = composition.Targets[0].RetainedActions[:len(composition.Targets[0].RetainedActions)-1]
	require.ErrorContains(t, composition.Validate(), "interference")
}

func TestCompositionRejectsTargetMissingCatalogModule(t *testing.T) {
	composition, err := DefaultComposition()
	require.NoError(t, err)
	for index := range composition.Targets {
		if composition.Targets[index].Identifier != TargetIDFoundationBacklogAck {
			continue
		}
		composition.Targets[index].Modules = composition.Targets[index].Modules[:2]
		composition.ArtifactDigest = "derived"
		composition.deriveArtifactDigest()
		require.ErrorContains(t, composition.Validate(), "catalog module")
		return
	}
	require.FailNow(t, "foundation backlog acknowledgement target is missing")
}
