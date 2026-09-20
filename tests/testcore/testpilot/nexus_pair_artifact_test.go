package testpilot

import (
	"testing"

	"github.com/stretchr/testify/require"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/testpilot"
	"go.temporal.io/server/common/testing/testpilot/temporal"
)

const (
	nexusPairArtifactNamespace        = "nexus-pair-namespace"
	nexusPairArtifactTaskQueue        = "nexus-pair-task-queue"
	nexusPairArtifactHandlerTaskQueue = "nexus-pair-handler-task-queue"
	nexusPairArtifactEndpoint         = "nexus-pair-endpoint"
	nexusPairRelationDefinitionID     = "temporal.nexus.pair.property.completionReferencesSchedule"
)

// TestNexusPairCaseCarriesOneCaptureRulePerInstance prepares the unchanged pair Case bytes offline.
// The Model's one field relation, over two instances of the operation, lowers to one capture rule
// per instance: each retains the scheduled event that records its own operation and matches the
// completion's reference against it; the two operations share no handler, slot or instruction; and
// the correlated capability the Query's Property lowered into sits beside them.
func TestNexusPairCaseCarriesOneCaptureRulePerInstance(t *testing.T) {
	source := loadLeanCase(t, NexusPairFixture)
	catalog, err := temporal.NewWorkflowServiceCatalog()
	require.NoError(t, err)

	rules := source.GetContract().GetRules()
	require.Len(t, rules, 2)
	for index, ruleID := range []string{NexusPairFirstRuleID, NexusPairSecondRuleID} {
		rule := rules[index]
		require.Equal(t, ruleID, rule.GetRuleId())
		require.Equal(t, nexusPairRelationDefinitionID+"."+ruleID, localNameDefinition(t, source, ruleID))
		require.Equal(t, testpilotspb.CONTRACT_RULE_KIND_SAFETY, rule.GetKind())
		require.Len(t, rule.GetCaptures(), 1)
		require.Len(t, rule.GetTransitions(), 2)
		require.Equal(t, "capture-nexusOperationScheduled-"+ruleID, rule.GetTransitions()[0].GetTransitionId())
		require.Equal(t, "match-nexusOperationCompleted-"+ruleID, rule.GetTransitions()[1].GetTransitionId())
	}
	require.NotNil(t, source.GetContract().GetCorrelated())

	var handlers []string
	for _, entrypoint := range source.GetProgram().GetEntrypoints() {
		if handler := entrypoint.GetNexusHandler(); handler != nil {
			handlers = append(handlers, handler.GetOperation())
		}
	}
	require.Equal(t, []string{NexusPairFirstOperation, NexusPairSecondOperation}, handlers)
	require.Len(t, source.GetProgram().GetSlots(), 2)

	prepared, err := testpilot.Prepare(source, NexusPairProfile(catalog, NexusCallerEnvironment{
		Namespace: nexusPairArtifactNamespace, TaskQueue: nexusPairArtifactTaskQueue,
		HandlerTaskQueue: nexusPairArtifactHandlerTaskQueue, NexusEndpoint: nexusPairArtifactEndpoint,
	}))
	require.NoError(t, err)
	require.Equal(t, source.GetCaseId(), prepared.Snapshot().GetCaseId())
}
