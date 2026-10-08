package worker

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/testpilot"
	"go.temporal.io/server/common/testing/testpilot/internal/testsupport/facadetest"
)

func durationAssignment(field string, seconds int64) *testpilotspb.RequestAssignment {
	return &testpilotspb.RequestAssignment{Target: field + ".seconds", Value: &testpilotspb.Expression{Expression: &testpilotspb.Expression_Literal{Literal: &testpilotspb.Value{Value: &testpilotspb.Value_SignedIntegerValue{SignedIntegerValue: strconv.FormatInt(seconds, 10)}}}}}
}

func TestActivityWithholdingRequiresItsRequestTimeoutBasis(t *testing.T) {
	for _, test := range []struct {
		name     string
		mode     testpilotspb.ActivityWithholdingMode
		field    string
		seconds  int64
		admitted bool
	}{
		{"context with start-to-close", testpilotspb.ACTIVITY_WITHHOLDING_MODE_CONTEXT, "start_to_close_timeout", 1, true},
		{"context with schedule-to-close", testpilotspb.ACTIVITY_WITHHOLDING_MODE_CONTEXT, "schedule_to_close_timeout", 1, true},
		{"heartbeat alone is no context deadline", testpilotspb.ACTIVITY_WITHHOLDING_MODE_CONTEXT, "heartbeat_timeout", 1, false},
		{"unarmed context", testpilotspb.ACTIVITY_WITHHOLDING_MODE_CONTEXT, "start_to_close_timeout", 0, false},
		{"pending with heartbeat", testpilotspb.ACTIVITY_WITHHOLDING_MODE_SDK_PENDING, "heartbeat_timeout", 1, true},
		{"pending without heartbeat", testpilotspb.ACTIVITY_WITHHOLDING_MODE_SDK_PENDING, "start_to_close_timeout", 1, false},
		{"unarmed pending", testpilotspb.ACTIVITY_WITHHOLDING_MODE_SDK_PENDING, "heartbeat_timeout", 0, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			prepared := preparedActivityFixture(t, func(program *testpilotspb.Program) {
				standaloneActivity(program)
				start := program.Entrypoints[0].Instructions[0].Instruction.GetInvokeRpc()
				start.RequestAssignments = append(start.RequestAssignments, durationAssignment(test.field, test.seconds))
				program.Entrypoints[1].Instructions = []*testpilotspb.InstructionNode{{InstructionId: "withheld", Limits: facadetest.Bounds(), Instruction: &testpilotspb.Instruction{Instruction: &testpilotspb.Instruction_ActivityAttemptWithholding{ActivityAttemptWithholding: &testpilotspb.ActivityAttemptWithholding{Mode: test.mode}}}}}
			}, func(profile *testpilot.ProfileSpec) {
				profile.Opcodes = append(profile.Opcodes, testpilot.ActivityAttemptWithholding)
			})
			host := symbolicRuntimeDriver(t, prepared.Limits())
			err := host.Validate(t.Context(), prepared)
			if test.admitted {
				require.NoError(t, err)
			} else {
				require.ErrorIs(t, err, ErrInvalid)
			}
		})
	}
}
