package execution

import (
	"testing"

	"github.com/stretchr/testify/require"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/testpilot/contract"
	"go.temporal.io/server/common/testing/testpilot/internal/ir"
)

// canceling is the instruction that answers its attempt as canceled.
func canceling(id string) *testpilotspb.InstructionNode {
	return activityNode(id, &testpilotspb.Instruction{Instruction: &testpilotspb.Instruction_ActivityAttemptCancellation{ActivityAttemptCancellation: &testpilotspb.ActivityAttemptCancellation{}}})
}

// An attempt is answered as canceled only through the instruction that says so, which only an
// activity entrypoint runs and a Profile authorizes as its own Opcode. Like every attempt it is one
// reserved activation of the activity's carrier.
func TestPrepareAdmitsACanceledAnswerOnlyOfAnActivityAttempt(t *testing.T) {
	unsupported := func(path string) *ir.Error {
		return &ir.Error{Category: ir.Unsupported, Path: path, Detail: "unsupported instruction context or Driver capability"}
	}
	for name, test := range map[string]struct {
		mutate func(*testpilotspb.Case, *Profile)
		want   *ir.Error
	}{
		"an activity attempt": {},
		"an answer that carries no message": {
			mutate: func(source *testpilotspb.Case, _ *Profile) {
				source.Program.Entrypoints[1].Instructions[0].Instruction = &testpilotspb.Instruction{Instruction: &testpilotspb.Instruction_ActivityAttemptCancellation{}}
			},
			want: &ir.Error{Category: ir.Malformed, Path: "$.program.entrypoints.instructions.instruction.activity_attempt_cancellation", Detail: "nil message"},
		},
		"a Profile without the capability": {
			mutate: func(_ *testpilotspb.Case, policy *Profile) { policy.Opcodes = policy.Opcodes[:len(policy.Opcodes)-1] },
			want:   &ir.Error{Category: ir.Unsupported, Path: "activity.run-attempt", Detail: "instruction activity_attempt_cancellation the Profile does not authorize"},
		},
		"in a workflow": {
			mutate: func(source *testpilotspb.Case, policy *Profile) {
				policy.Roles[0].ReservationCarriers[0].Shapes[0].Kind = contract.WorkflowEntrypoint
				source.Program.Entrypoints[1].Activation = &testpilotspb.Entrypoint_Workflow{Workflow: &testpilotspb.WorkflowActivation{WorkflowType: "flow", WorkerRoleId: "worker", TaskQueueRoleId: "queue"}}
			},
			want: unsupported("activity.run-attempt"),
		},
		"in a Nexus handler": {
			mutate: func(source *testpilotspb.Case, _ *Profile) {
				source.Program.Entrypoints[1].Activation = &testpilotspb.Entrypoint_NexusHandler{NexusHandler: &testpilotspb.NexusHandlerActivation{Service: "service", Operation: "operation", WorkerRoleId: "worker", TaskQueueRoleId: "queue"}}
			},
			want: unsupported("activity.run-attempt"),
		},
		"in a controller": {
			mutate: func(source *testpilotspb.Case, _ *Profile) {
				node := canceling("cancel")
				node.After = runsAfter("controller")
				source.Program.Entrypoints[0].Instructions = append(source.Program.Entrypoints[0].Instructions, node)
			},
			want: unsupported("controller.cancel"),
		},
	} {
		t.Run(name, func(t *testing.T) {
			source, catalog, policy := activityFixture(t)
			policy.Opcodes = append(policy.Opcodes, contract.ActivityAttemptCancellation)
			source.Program.Entrypoints[1].Instructions[0] = canceling("run-attempt")
			if test.mutate != nil {
				test.mutate(source, &policy)
			}
			prepared, err := Prepare(source, catalog, policy)
			if test.want != nil {
				var rejected *ir.Error
				require.ErrorAs(t, err, &rejected)
				require.Equal(t, test.want, rejected)
				return
			}
			require.NoError(t, err)
			require.Equal(t, contract.ActivityAttemptCancellation, prepared.Entrypoints()[1].Instructions()[0].Opcode())
			plan, carried := prepared.ReservationCarrier("controller", "call")
			require.True(t, carried)
			require.Equal(t, []contract.ReservationTopology{{EntrypointID: "activity", Kind: contract.ActivityEntrypoint, Count: 1}}, plan.Reservations)
		})
	}
}
