package activation

import (
	"testing"

	"github.com/stretchr/testify/require"
	enumspb "go.temporal.io/api/enums/v1"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/testpilot"
	"go.temporal.io/server/common/testing/testpilot/internal/testsupport/facadetest"
	"google.golang.org/protobuf/proto"
)

const startActivityMethod = "/temporal.api.workflowservice.v1.WorkflowService/StartActivityExecution"

// activityPlan is the activity entrypoint of a Program whose controller starts one standalone
// activity: the script an activity realization lowers to, one Finish that completes the attempt.
func activityPlan(t *testing.T) testpilot.EntrypointPlan {
	t.Helper()
	finish := &testpilotspb.InstructionNode{
		InstructionId: "run-attempt",
		Instruction:   &testpilotspb.Instruction{Instruction: &testpilotspb.Instruction_Finish{Finish: &testpilotspb.Finish{Result: facadetest.Text("done")}}},
		Limits:        facadetest.Bounds(),
	}
	prepared := facadetest.RuntimeCase(t, facadetest.SyncReply, []enumspb.CommandType{enumspb.COMMAND_TYPE_SCHEDULE_NEXUS_OPERATION}, func(profile *testpilot.ProfileSpec) {
		profile.Roles[0].Methods = append(profile.Roles[0].Methods, startActivityMethod)
		profile.Roles[0].ReservationCarriers = append(profile.Roles[0].ReservationCarriers, testpilot.ReservationCarrierPolicy{Method: startActivityMethod, Shapes: []testpilot.ReservationCarrierShape{{Kind: testpilot.ActivityEntrypoint, MaximumCount: 1}}})
	}, func(program *testpilotspb.Program) {
		program.Entrypoints[0].Instructions[0].GetInstruction().GetInvokeRpc().Method = startActivityMethod
		program.Entrypoints = append(program.Entrypoints[:1], &testpilotspb.Entrypoint{
			EntrypointId: "activity",
			Activation:   &testpilotspb.Entrypoint_Activity{Activity: &testpilotspb.ActivityActivation{ActivityType: "activity-type", WorkerRoleId: "worker", TaskQueueRoleId: "queue"}},
			Instructions: []*testpilotspb.InstructionNode{finish},
		})
	})
	plan, ok := findEntrypoint(facadetest.Capture(t, prepared), "activity")
	require.True(t, ok)
	return plan
}

// An activity attempt interprets its script through a fresh State as a workflow or Nexus-handler
// activation does: the Finish evaluates to the attempt's result and admits once.
func TestActivityEntrypointActivates(t *testing.T) {
	plan := activityPlan(t)
	state, err := New(plan)
	require.NoError(t, err)
	require.Equal(t, plan.RuntimeWorkLimit(), state.remaining)

	result, enabled, err := state.Evaluate(t.Context(), 0)
	require.NoError(t, err)
	require.True(t, enabled)
	require.True(t, proto.Equal(&testpilotspb.Value{Value: &testpilotspb.Value_TextValue{TextValue: "done"}}, result))
	require.NoError(t, state.Admit(t.Context(), 0, &testpilotspb.InstructionOutcome{Status: testpilotspb.INSTRUCTION_OUTCOME_STATUS_SUCCEEDED}))
	require.Error(t, state.Admit(t.Context(), 0, &testpilotspb.InstructionOutcome{Status: testpilotspb.INSTRUCTION_OUTCOME_STATUS_SUCCEEDED}))

	// Two attempts of one prepared plan share nothing.
	other, err := New(plan)
	require.NoError(t, err)
	_, enabled, err = other.Evaluate(t.Context(), 0)
	require.NoError(t, err)
	require.True(t, enabled)
}
