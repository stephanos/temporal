package testpilot

import (
	"testing"

	"github.com/stretchr/testify/require"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/testpilot"
	"go.temporal.io/server/common/testing/testpilot/temporal"
)

const (
	typedNexusArtifactNamespace = "typed-nexus-namespace"
	typedNexusArtifactTaskQueue = "typed-nexus-task-queue"
	typedNexusArtifactEndpoint  = "typed-nexus-endpoint"
	typedNexusEvidenceID        = "correlated-evidence"
	// The Case names the projection and the clause by Case-local names; its provenance maps them to
	// their Definition IDs.
	typedNexusProjectionID           = "projection"
	typedNexusProjectionDefinitionID = "temporal.nexus.success.typed-nexus.projection"
	typedNexusClauseID               = "bounded-completion"
	typedNexusClauseDefinitionID     = "temporal.nexus.success.typed-nexus.clause.bounded-completion"
)

// localNameDefinition returns the Definition ID the Case's provenance maps a Case-local name to; a
// name no row maps is its own Definition ID.
func localNameDefinition(t testing.TB, source *testpilotspb.Case, localName string) string {
	t.Helper()
	for _, row := range source.GetProvenance().GetLocalNames() {
		if row.GetLocalName() == localName {
			return row.GetDefinitionId()
		}
	}
	return localName
}

// TestTypedNexusCaseAdmitsItsCorrelatedCapability prepares the unchanged two-operation Case bytes
// offline. The Case declares a CorrelatedEvidence Observation its own history read lifts into, so
// admitting the Program's evidence lift and the Contract's correlated capability against each other is
// what this asserts; the live test then runs the clause the opcode carries.
func TestTypedNexusCaseAdmitsItsCorrelatedCapability(t *testing.T) {
	source := loadLeanCase(t, "typed-nexus")
	catalog, err := temporal.NewWorkflowServiceCatalog()
	require.NoError(t, err)

	opcode := source.GetContract().GetCorrelated()
	require.NotNil(t, opcode)
	require.Equal(t, typedNexusEvidenceID, opcode.GetEvidenceObservationId())
	require.Equal(t, typedNexusProjectionID, opcode.GetProjectionId())
	require.Equal(t, typedNexusProjectionDefinitionID, localNameDefinition(t, source, opcode.GetProjectionId()))
	require.Len(t, opcode.GetRules(), 1)
	require.Equal(t, typedNexusClauseID, opcode.GetRules()[0].GetRuleId())
	require.Equal(t, typedNexusClauseDefinitionID, localNameDefinition(t, source, opcode.GetRules()[0].GetRuleId()))

	prepared, err := testpilot.Prepare(source, TypedNexusProfile(catalog, TypedNexusEnvironment{
		Namespace: typedNexusArtifactNamespace, TaskQueue: typedNexusArtifactTaskQueue,
		NexusEndpoint: typedNexusArtifactEndpoint,
	}))
	require.NoError(t, err)
	require.Equal(t, source.GetCaseId(), prepared.Snapshot().GetCaseId())
}
