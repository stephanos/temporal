package execution

import (
	"slices"
	"testing"

	"github.com/stretchr/testify/require"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/testpilot/contract"
	"go.temporal.io/server/common/testing/testpilot/internal/ir"
	"google.golang.org/protobuf/proto"
)

// workflowActivityFixture is scheduledActivityFixture with the activity its workflow schedules
// declared as an activity entrypoint, whose attempts complete, fail retryably, and withhold their
// answer; the workflow's start carries them.
func workflowActivityFixture(t *testing.T) (*testpilotspb.Case, *ir.Catalog, Profile) {
	t.Helper()
	c, catalog, p := scheduledActivityFixture(t)
	p.Roles[0].ReservationCarriers[0].Shapes = []contract.ReservationCarrierShape{{Kind: contract.WorkflowEntrypoint, MaximumCount: 1}, {Kind: contract.ActivityEntrypoint, MaximumCount: 3}}
	p.Opcodes = append(p.Opcodes, contract.ActivityAttemptFailure, contract.ActivityAttemptWithholding)
	c.Program.Entrypoints = append(c.Program.Entrypoints, &testpilotspb.Entrypoint{
		EntrypointId: "activity",
		Activation:   &testpilotspb.Entrypoint_Activity{Activity: &testpilotspb.ActivityActivation{ActivityType: "activity-type", WorkerRoleId: "worker", TaskQueueRoleId: "queue", AttemptNumbering: &testpilotspb.AttemptNumbering{First: 1, OneRun: true}}},
		Instructions: []*testpilotspb.InstructionNode{
			activityNode("withhold", &testpilotspb.Instruction{Instruction: &testpilotspb.Instruction_ActivityAttemptWithholding{ActivityAttemptWithholding: &testpilotspb.ActivityAttemptWithholding{}}}),
			failing("fail", retryableFailure()),
			activityNode("complete", &testpilotspb.Instruction{Instruction: &testpilotspb.Instruction_Finish{Finish: &testpilotspb.Finish{Result: textLiteral("done")}}}),
		},
	})
	return c, catalog, p
}

// The start of a workflow that schedules an activity carries one reservation per attempt the
// activity's script declares, and routes the schedule command to the activity's first attempt.
func TestPrepareCarriesAScheduledActivityOnItsWorkflowsStart(t *testing.T) {
	c, catalog, p := workflowActivityFixture(t)
	prepared, err := Prepare(c, catalog, p)
	require.NoError(t, err)
	plan, carried := prepared.ReservationCarrier("controller", "call")
	require.True(t, carried)
	require.Equal(t, []contract.ReservationTopology{
		{EntrypointID: "workflow", Kind: contract.WorkflowEntrypoint, Count: 1},
		{EntrypointID: "activity", Kind: contract.ActivityEntrypoint, Count: 3},
	}, plan.Reservations)
	require.Equal(t, []contract.ReservationRoute{{WorkflowEntrypointID: "workflow", SourceInstructionID: "start", HandlerEntrypointID: "activity"}}, plan.Routes)
}

// A scheduled activity is reserved only by its workflow's carrier: a Profile whose workflow start
// carries no activity leaves it unreserved, which rejects before any I/O; a second schedule of it
// rejects; and an instruction the Profile does not authorize is named.
func TestPrepareRejectsScheduledActivitiesItCannotRoute(t *testing.T) {
	second := func(c *testpilotspb.Case) {
		workflow := c.Program.Entrypoints[1]
		again := proto.CloneOf(workflow.Instructions[0])
		again.InstructionId = "again"
		workflow.Instructions = append(workflow.Instructions, again)
	}
	for name, test := range map[string]struct {
		mutate func(*testpilotspb.Case, *Profile)
		want   *ir.Error
	}{
		"a workflow start that carries no activity": {
			mutate: func(_ *testpilotspb.Case, p *Profile) {
				p.Roles[0].ReservationCarriers[0].Shapes = slices.DeleteFunc(p.Roles[0].ReservationCarriers[0].Shapes, func(shape contract.ReservationCarrierShape) bool {
					return shape.Kind == contract.ActivityEntrypoint
				})
			},
			want: &ir.Error{Category: ir.Unavailable, Path: "activity", Detail: "no reservation carrier of the Profile activates the activity entrypoint"},
		},
		"an activity scheduled twice": {
			mutate: func(c *testpilotspb.Case, _ *Profile) { second(c) },
			want:   &ir.Error{Category: ir.Unsupported, Path: "workflow.again", Detail: "activity entrypoint activity is scheduled more than once"},
		},
		"a Profile without the withholding": {
			mutate: func(_ *testpilotspb.Case, p *Profile) {
				p.Opcodes = slices.DeleteFunc(p.Opcodes, func(opcode contract.Opcode) bool { return opcode == contract.ActivityAttemptWithholding })
			},
			want: &ir.Error{Category: ir.Unsupported, Path: "activity.withhold", Detail: "instruction activity_attempt_withholding the Profile does not authorize"},
		},
		"a withholding outside an activity": {
			mutate: func(c *testpilotspb.Case, _ *Profile) {
				c.Program.Entrypoints[1].Instructions[2].Instruction = &testpilotspb.Instruction{Instruction: &testpilotspb.Instruction_ActivityAttemptWithholding{ActivityAttemptWithholding: &testpilotspb.ActivityAttemptWithholding{}}}
			},
			want: &ir.Error{Category: ir.Unsupported, Path: "workflow.finish", Detail: "unsupported instruction context or Driver capability"},
		},
	} {
		t.Run(name, func(t *testing.T) {
			c, catalog, p := workflowActivityFixture(t)
			test.mutate(c, &p)
			_, err := Prepare(c, catalog, p)
			var diagnostic *ir.Error
			require.ErrorAs(t, err, &diagnostic)
			require.Equal(t, test.want, diagnostic)
		})
	}
}
