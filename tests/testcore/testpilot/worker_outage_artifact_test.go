package testpilot

import (
	"testing"

	"github.com/stretchr/testify/require"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/testpilot"
	"go.temporal.io/server/common/testing/testpilot/temporal"
)

const (
	workerOutageArtifactNamespace = "worker-outage-namespace"
	workerOutageArtifactTaskQueue = "worker-outage-task-queue"
)

// TestWorkerOutageCaseDeclaresAnEventCountHorizon prepares the unchanged outage Case bytes offline.
// The bound the liveness rule carries is an event count and nothing else: an elapsed-time bound
// would let a slow host decide the outage window, which is what this Case exists to avoid. Whether
// that counter agrees online and offline is owned by the one helper both paths reach it through
// (`TestEvaluatorEventCountHorizon`); this pins that the shipped Case is the shape it counts for.
func TestWorkerOutageCaseDeclaresAnEventCountHorizon(t *testing.T) {
	source := loadLeanCase(t, "worker-outage")
	catalog, err := temporal.NewWorkflowServiceCatalog()
	require.NoError(t, err)

	var liveness *testpilotspb.ContractRuleDefinition
	for _, rule := range source.GetContract().GetRules() {
		if rule.GetKind() == testpilotspb.CONTRACT_RULE_KIND_BOUNDED_LIVENESS {
			liveness = rule
		}
	}
	require.NotNil(t, liveness)
	require.Equal(t, "worker-outage-order", liveness.GetRuleId())
	require.Positive(t, liveness.GetDeadline().GetRuleEvents())
	require.Zero(t, liveness.GetDeadline().GetElapsedMilliseconds())
	require.Equal(t, "expired", liveness.GetDeadline().GetViolationStateId())

	derived, err := temporal.DeriveProfile(source, catalog, temporal.Environment{
		Identity: "worker-outage-profile", Namespace: workerOutageArtifactNamespace,
		TaskQueue: workerOutageArtifactTaskQueue,
	})
	require.NoError(t, err)
	require.Contains(t, derived.Opcodes, testpilot.InjectFault)

	prepared, err := testpilot.Prepare(source, derived)
	require.NoError(t, err)
	require.Equal(t, source.GetCaseId(), prepared.Snapshot().GetCaseId())
}
