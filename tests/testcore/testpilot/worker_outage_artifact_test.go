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

// TestWorkerOutageCaseDeclaresAnEventCountDeadline prepares the unchanged outage Case bytes offline.
// The bound the liveness rule carries is an event count and nothing else: an elapsed-time bound
// would let a slow host decide the outage window, which is what this Case exists to avoid. Whether
// that counter agrees online and offline is owned by the one helper both paths reach it through
// (`TestEvaluatorEventCountDeadline`); this pins that the shipped Case is the shape it counts for,
// and that the rule is the one the Producer derives from the two faults the Program injects.
func TestWorkerOutageCaseDeclaresAnEventCountDeadline(t *testing.T) {
	source := loadLeanCase(t, WorkerOutageFixture)
	catalog, err := temporal.NewWorkflowServiceCatalog()
	require.NoError(t, err)

	var liveness *testpilotspb.ContractRule
	for _, rule := range source.GetContract().GetRules() {
		if rule.GetKind() == testpilotspb.CONTRACT_RULE_KIND_BOUNDED_LIVENESS {
			liveness = rule
		}
	}
	require.NotNil(t, liveness)
	require.Equal(t, WorkerOutageRuleID, liveness.GetRuleId())
	require.Positive(t, liveness.GetDeadline().GetRuleEvents())
	require.Zero(t, liveness.GetDeadline().GetElapsedMilliseconds())
	require.Equal(t, "expired", liveness.GetDeadline().GetViolationStateId())
	require.Len(t, source.GetContract().GetRules(), 1)
	require.NotNil(t, source.GetContract().GetCorrelated())

	// The faults the rule orders are the two the Program injects, stop then resume, on the Case's
	// own task-queue role.
	var faults []*testpilotspb.InjectFault
	for _, entrypoint := range source.GetProgram().GetEntrypoints() {
		for _, node := range entrypoint.GetInstructions() {
			if fault := node.GetInstruction().GetInjectFault(); fault != nil {
				faults = append(faults, fault)
			}
		}
	}
	require.Len(t, faults, 2)
	require.Equal(t, WorkerOutageTaskQueueRole, faults[0].GetRoleId())
	require.Equal(t, testpilotspb.FAULT_KIND_WORKER_STOP, faults[0].GetKind())
	require.Equal(t, WorkerOutageTaskQueueRole, faults[1].GetRoleId())
	require.Equal(t, testpilotspb.FAULT_KIND_WORKER_RESUME, faults[1].GetKind())

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
