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
	typedNexusEvidenceID        = "correlated-evidence"
	typedNexusProjectionID      = "temporal.nexus3.typed-nexus.projection"
	typedNexusClauseID          = "temporal.nexus3.typed-nexus.clause.bounded-completion"
)

// TestTypedNexusCaseAdmitsItsCorrelatedCapability prepares the unchanged two-operation Case bytes
// offline. The Case declares a CorrelatedEvidence Observation its own history read lifts into, so
// admitting the Program's evidence lift and the Contract's scoped opcode against each other is
// what this asserts; the live test then runs the clause the opcode carries.
func TestTypedNexusCaseAdmitsItsCorrelatedCapability(t *testing.T) {
	source := loadLeanCase(t, "typed-nexus")
	catalog, err := temporal.NewWorkflowServiceCatalog()
	require.NoError(t, err)

	opcode := source.GetContract().GetCorrelated()
	require.NotNil(t, opcode)
	require.Equal(t, typedNexusEvidenceID, opcode.GetEvidenceObservationId())
	require.Equal(t, typedNexusProjectionID, opcode.GetProjectionId())
	require.Len(t, opcode.GetClauses(), 1)
	require.Equal(t, typedNexusClauseID, opcode.GetClauses()[0].GetClauseId())

	prepared, err := testpilot.Prepare(source, TypedNexusProfile(catalog, source, TypedNexusEnvironment{
		Namespace: typedNexusArtifactNamespace, TaskQueue: typedNexusArtifactTaskQueue,
		NexusEndpoint: typedNexusArtifactEndpoint,
	}))
	require.NoError(t, err)
	require.Equal(t, source.GetCaseId(), prepared.Snapshot().GetCaseId())
}
