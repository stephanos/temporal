package testpilot

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.temporal.io/server/common/testing/testpilot"
	"go.temporal.io/server/common/testing/testpilot/temporal"
)

const (
	typedNexusArtifactNamespace = "typed-nexus-namespace"
	typedNexusArtifactTaskQueue = "typed-nexus-task-queue"
	typedNexusArtifactEndpoint  = "typed-nexus-endpoint"
	typedNexusEvidenceID        = "scoped-evidence"
	typedNexusProjectionID      = "temporal.nexus3.typed-nexus.projection"
	typedNexusClauseID          = "temporal.nexus3.typed-nexus.clause.bounded-completion"
)

// TestTypedNexusCaseAdmitsItsScopedCapability prepares the unchanged two-operation Case bytes
// offline. The Case declares a ScopedEvidence Observation its own history read lifts into, so
// admitting the Program's evidence lift and the Contract's scoped capability against each other is
// what this asserts; the live test then runs the clause the capability carries.
func TestTypedNexusCaseAdmitsItsScopedCapability(t *testing.T) {
	source := loadLeanCase(t, "typed-nexus")
	catalog, err := temporal.NewWorkflowServiceCatalog()
	require.NoError(t, err)

	capability := source.GetContract().GetScoped()
	require.NotNil(t, capability)
	require.Equal(t, typedNexusEvidenceID, capability.GetEvidenceObservationId())
	require.Equal(t, typedNexusProjectionID, capability.GetProjectionId())
	require.Len(t, capability.GetClauses(), 1)
	require.Equal(t, typedNexusClauseID, capability.GetClauses()[0].GetClauseId())

	prepared, err := testpilot.Prepare(source, TypedNexusProfile(catalog, source, TypedNexusEnvironment{
		Namespace: typedNexusArtifactNamespace, TaskQueue: typedNexusArtifactTaskQueue,
		NexusEndpoint: typedNexusArtifactEndpoint,
	}))
	require.NoError(t, err)
	require.Equal(t, source.GetCaseId(), prepared.Snapshot().GetCaseId())
}
