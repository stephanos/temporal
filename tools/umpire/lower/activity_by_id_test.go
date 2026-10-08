package lower

import (
	"slices"
	"testing"

	"github.com/stretchr/testify/require"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	umpirespb "go.temporal.io/server/api/umpire/v1"
	"go.temporal.io/server/tools/umpire/realization"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoregistry"
)

var activityByIDCases = map[string]activityCase{
	"scheduledCompletedByID": {controller: []string{"start-activity", "respond-completed-by-id", "await-external-completed", "read-external-attempt-count"}, confirmed: map[string][]string{"statusScheduled": {plainStart}, "externalCompleted": {"respondCompletedByID"}}},
	"heldFailedByID":         {controller: []string{"start-activity", "await-failure-publication", "await-external-started", "respond-failed-by-id", "await-external-failed", "read-external-attempt-count"}, activity: []string{"failure-pending"}, confirmed: map[string][]string{"statusScheduled": {plainStart}, "externalStarted": {"poll"}, "externalFailed": {"respondFailedByID-fatal"}}},
	"heldCanceledByID":       {controller: []string{"start-activity", "await-cancellation-publication", "await-external-started", "request-external-cancellation", "await-external-cancel-requested", "respond-canceled-by-id", "await-external-canceled", "read-external-attempt-count"}, activity: []string{"cancellation-pending"}, confirmed: map[string][]string{"statusScheduled": {plainStart}, "externalStarted": {"poll"}, "externalCancelRequested": {"requestCancel"}, "externalCanceled": {"respondCanceledByID"}}},
}

func TestActivityByIDCasesKeepServiceAnswersSeparateFromWorkerPublication(t *testing.T) {
	m := loaded(t, "activity-standalone")
	p, err := NewProducer(m)
	require.NoError(t, err)
	for _, test := range []struct{ query, answer, method, script, publication, held, settlement string }{
		{"scheduledCompletedByID", "respond-completed-by-id", "RespondActivityTaskCompletedById", "", "", "", "await-external-completed"},
		{"heldFailedByID", "respond-failed-by-id", "RespondActivityTaskFailedById", "external-failure-attempts", "await-failure-publication", "await-external-started", "await-external-failed"},
		{"heldCanceledByID", "respond-canceled-by-id", "RespondActivityTaskCanceledById", "external-cancellation-attempts", "await-cancellation-publication", "await-external-cancel-requested", "await-external-canceled"},
	} {
		t.Run(test.query, func(t *testing.T) {
			result, err := p.Lower(test.query, activityIdentity(test.query))
			require.NoError(t, err)
			require.Equal(t, Lowered, result.Standing, "%v", result.Unsupported)
			require.Empty(t, result.OffPath)
			c := result.Case
			want := map[string][]string{"controller": activityByIDCases[test.query].controller}
			if test.script != "" {
				want[test.script] = activityByIDCases[test.query].activity
			}
			require.Equal(t, want, instructionIDs(c))
			answer := instruction(t, c, "controller", test.answer).GetInstruction()
			require.Equal(t, "/temporal.api.workflowservice.v1.WorkflowService/"+test.method, answer.GetInvokeRpc().GetMethod())
			require.Nil(t, answer.GetActivityAttemptFailure())
			require.Nil(t, answer.GetActivityAttemptCancellation())
			require.Nil(t, answer.GetFinish())
			require.Len(t, c.GetProgram().GetActivityExternalSettlements(), 1)
			basis := c.GetProgram().GetActivityExternalSettlements()[0]
			require.Equal(t, &testpilotspb.InstructionReference{EntrypointId: "controller", InstructionId: "start-activity"}, basis.GetCarrier())
			require.Equal(t, &testpilotspb.InstructionReference{EntrypointId: "controller", InstructionId: test.answer}, basis.GetAnswer())
			require.Equal(t, &testpilotspb.InstructionReference{EntrypointId: "controller", InstructionId: test.settlement}, basis.GetSettlement())
			require.Equal(t, int64(0), basis.GetReservationOrdinal())
			require.Equal(t, test.script, basis.GetActivityEntrypointId())
			require.Equal(t, &testpilotspb.InstructionReference{EntrypointId: "cleanup", InstructionId: "external-cleanup"}, basis.GetCleanup())
			require.Len(t, c.GetProgram().GetCleanup().GetInstructions(), 1)
			cleanup := c.GetProgram().GetCleanup().GetInstructions()[0]
			require.Equal(t, "/temporal.api.workflowservice.v1.WorkflowService/TerminateActivityExecution", cleanup.GetInstruction().GetInvokeRpc().GetMethod())
			guards := cleanup.GetGuard().GetAll().GetOperands()
			require.Len(t, guards, 2, "cleanup runs after successful Start unless terminal Describe already proves closure")
			require.Equal(t, &testpilotspb.InstructionReference{EntrypointId: "controller", InstructionId: "start-activity"}, guards[0].GetCompare().GetLeft().GetReference().GetOutcome().GetInstruction())
			require.Equal(t, testpilotspb.INSTRUCTION_OUTCOME_FIELD_STATUS, guards[0].GetCompare().GetLeft().GetReference().GetOutcome().GetField())
			require.Equal(t, "INSTRUCTION_OUTCOME_STATUS_SUCCEEDED", guards[0].GetCompare().GetRight().GetLiteral().GetEnumValue().GetName())
			require.Equal(t, &testpilotspb.InstructionReference{EntrypointId: "controller", InstructionId: test.settlement}, guards[1].GetNot().GetOperand().GetCompare().GetLeft().GetReference().GetOutcome().GetInstruction())
			require.Equal(t, testpilotspb.INSTRUCTION_OUTCOME_FIELD_STATUS, guards[1].GetNot().GetOperand().GetCompare().GetLeft().GetReference().GetOutcome().GetField())
			require.Equal(t, "INSTRUCTION_OUTCOME_STATUS_SUCCEEDED", guards[1].GetNot().GetOperand().GetCompare().GetRight().GetLiteral().GetEnumValue().GetName())
			if test.script == "" {
				require.Nil(t, basis.GetHeld())
				require.Empty(t, basis.GetPendingSlotId())
				require.False(t, slices.ContainsFunc(c.GetProgram().GetRoles(), func(r *testpilotspb.Role) bool { return r.GetKind() == testpilotspb.ROLE_KIND_WORKER }))
			} else {
				require.Equal(t, &testpilotspb.InstructionReference{EntrypointId: "controller", InstructionId: test.held}, basis.GetHeld())
				require.Equal(t, basis.GetPendingSlotId(), instruction(t, c, "controller", test.publication).GetInstruction().GetAwaitSlot().GetSlotId())
				var publicationType string
				for _, slot := range c.GetProgram().GetSlots() {
					if slot.GetSlotId() == basis.GetPendingSlotId() {
						publicationType = slot.GetValue().GetSingular().GetMessage().GetProtobufType()
					}
				}
				require.Equal(t, "temporal.server.api.testpilot.v1.ActivityAttempt", publicationType)
				pending := instruction(t, c, test.script, activityByIDCases[test.query].activity[0]).GetInstruction().GetActivityAttemptWithholding()
				require.Equal(t, testpilotspb.ACTIVITY_WITHHOLDING_MODE_SDK_PENDING, pending.GetMode())
				require.True(t, proto.Equal(basis.GetAnswer(), pending.GetExternalSettlement()))
			}
			if test.query == "heldCanceledByID" {
				require.Equal(t, &testpilotspb.InstructionReference{EntrypointId: "controller", InstructionId: "request-external-cancellation"}, basis.GetRequestCancel())
			} else {
				require.Nil(t, basis.GetRequestCancel())
			}
			preparedAsIs(t, c)
			a, _, err := p.ask(test.query)
			require.NoError(t, err)
			carriers, err := realization.Carriers(a.r, protoregistry.GlobalFiles)
			require.NoError(t, err)
			var byID []realization.Carrier
			for _, mapping := range carriers {
				for _, carrier := range mapping.Carriers {
					if carrier.Method == "/temporal.api.workflowservice.v1.WorkflowService/"+test.method {
						byID = append(byID, carrier)
					}
				}
			}
			require.Equal(t, []realization.Carrier{{Kind: "rpc", Method: "/temporal.api.workflowservice.v1.WorkflowService/" + test.method, Message: "temporal.api.workflowservice.v1." + test.method + "Request"}}, byID)
		})
	}
}

func TestActivityByIDMissingMethodVisibilityIsNotWorkerVisibility(t *testing.T) {
	m := loaded(t, "activity-standalone")
	for _, r := range m.GetRealizations() {
		r.GetBehavior().Visibility = slices.DeleteFunc(r.GetBehavior().GetVisibility(), func(v *umpirespb.Visibility) bool {
			return v.GetMethod() == "/temporal.api.workflowservice.v1.WorkflowService/RespondActivityTaskFailedById"
		})
	}
	p, err := NewProducer(m)
	require.NoError(t, err)
	_, err = p.Lower("heldFailedByID", activityIdentity("heldFailedByID"))
	require.ErrorContains(t, err, "RespondActivityTaskFailedById")
}

func TestActivityByIDPublicationRequiresOneSelectedAnswerOfTheDeclaredAttempt(t *testing.T) {
	p, err := NewProducer(loaded(t, "activity-standalone"))
	require.NoError(t, err)
	a, _, err := p.ask("heldFailedByID")
	require.NoError(t, err)
	path, problems := p.check(a, activityIdentity("heldFailedByID"))
	require.Empty(t, problems)
	require.Empty(t, path.withholdingOccurrences())
	basis := a.r.GetExternalSettlements()[0]
	script := scriptNamed(t, a.r, basis.GetActivity())
	answer := path.adapter.classKey(script.GetItems()[0].GetWhen()[0])
	selected := slices.Clone(path.keys)
	for _, test := range []struct {
		name, message string
		keys          []string
		attempt       int64
	}{
		{"absent answer", "requires exactly one occurrence of its external answer; got 0", slices.DeleteFunc(slices.Clone(selected), func(key string) bool { return key == answer }), 1},
		{"duplicate answer", "requires exactly one occurrence of its external answer; got 2", append(slices.Clone(selected), answer), 1},
		{"crossed attempt", "declares attempt 2; selected publication is attempt 1", selected, 2},
	} {
		t.Run(test.name, func(t *testing.T) {
			path.keys, basis.Attempt = test.keys, test.attempt
			gaps := path.withholdingOccurrences()
			require.Len(t, gaps, 1)
			require.ErrorContains(t, gaps[0], test.message)
		})
	}
}
