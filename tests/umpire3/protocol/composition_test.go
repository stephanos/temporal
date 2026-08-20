package protocol

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDefaultCompositionConnectsNexusAndUpdateToSharedDelivery(t *testing.T) {
	composition, err := DefaultComposition()
	require.NoError(t, err)
	require.Equal(t, ResultClassMetadataValidated, composition.ResultClass)
	require.Equal(t, TrustBadgeKernel, composition.TrustBadge)
	require.Len(t, composition.Targets, 15)

	provider, ok := composition.Module("Temporal.System.TaskDelivery")
	require.True(t, ok)
	require.Len(t, provider.Provides, 1)
	for _, identifier := range []ModuleID{ModuleIDTemporalSystemNexusTasks, ModuleIDTemporalSystemUpdateTasks} {
		consumer, ok := composition.Module(identifier)
		require.True(t, ok)
		require.Equal(t, provider.Identifier, consumer.Requires[0].ProviderModule)
		require.Equal(t, provider.Provides[0].StatementHash, consumer.Requires[0].StatementHash)
	}
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
		"Temporal.Product.NexusClosure",
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
		"Temporal.Product.NexusLifecycle",
		"Temporal.Product.NexusClosure",
		"Temporal.System.NexusClosure",
		"Temporal.Refinement.NexusClosure",
	}, target.Modules)
}

func TestCompositionOwnsNexusEvidenceThroughRelationalModules(t *testing.T) {
	composition, err := DefaultComposition()
	require.NoError(t, err)
	for _, identifier := range []ModuleID{
		"Temporal.Product.NexusTimeout",
		"Temporal.System.NexusTimeout",
		"Temporal.Refinement.NexusTimeout",
		"Temporal.Product.NexusActivityLink",
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
			"Temporal.Product.NexusActivityLink",
			"Temporal.System.NexusActivityLink",
			"Temporal.Refinement.NexusActivityLink",
		},
		TargetIDIntegrationNexusTimeout: {
			"Temporal.Product.NexusTimeout",
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
		"Temporal.Product.CallbackReference",
		"Temporal.System.CallbackReference",
		"Temporal.Refinement.CallbackReference",
		"Temporal.Product.CallbackResponse",
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
			"Temporal.Product.CallbackReference",
			"Temporal.System.CallbackReference",
			"Temporal.Refinement.CallbackReference",
		},
		TargetIDIntegrationCallbackWorkflow: {
			"Temporal.Product.CallbackResponse",
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
		"Temporal.Product.WorkflowLineage",
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
			require.Subset(t, target.Modules, []ModuleID{
				"Temporal.Product.WorkflowLineage",
				"Temporal.System.WorkflowLineage",
				"Temporal.Refinement.WorkflowLineage",
			})
			return
		}
	}
	require.FailNow(t, "foundation routing target is missing")
}

func TestCompositionOwnsWorkflowOwnershipThroughRelationalModules(t *testing.T) {
	composition, err := DefaultComposition()
	require.NoError(t, err)
	for _, identifier := range []ModuleID{
		"Temporal.Product.WorkflowOwnership",
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
				"Temporal.Product.WorkflowOwnership",
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
	composition.Modules[2].Requires[0].ProviderModule = "missing"
	require.ErrorContains(t, composition.Validate(), "missing provider")

	composition, err = DefaultComposition()
	require.NoError(t, err)
	composition.Modules[2].Rank = 0
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

func TestCompositionRejectsDroppedInterferenceAction(t *testing.T) {
	composition, err := DefaultComposition()
	require.NoError(t, err)
	composition.Targets[0].RetainedActions = composition.Targets[0].RetainedActions[:len(composition.Targets[0].RetainedActions)-1]
	require.ErrorContains(t, composition.Validate(), "interference")
}
