package execution

import (
	"context"
	"testing"
	"time"

	celpb "cel.dev/expr"
	"github.com/stretchr/testify/require"
	activitypb "go.temporal.io/api/activity/v1"
	enumspb "go.temporal.io/api/enums/v1"
	"go.temporal.io/api/workflowservice/v1"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	cel "go.temporal.io/server/common/testing/testpilot/cel"
	"go.temporal.io/server/common/testing/testpilot/contract"
	"go.temporal.io/server/common/testing/testpilot/internal/ir"
	"go.temporal.io/server/common/testing/testpilot/internal/testsupport"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// resetFixture is a deferred reset: the controller starts the activity, awaits the held attempt's
// published pending record, reads it held, resets it and reads its terminal settlement. The activity
// script's first group withholds its answer on the timer basis that applies the reset; its second
// group is the server's first attempt again, and completes.
func resetFixture(t *testing.T) (*testpilotspb.Case, *ir.Catalog, Profile) {
	t.Helper()
	source, catalog, policy := externalFixture(t, false)
	policy.Roles[0].Methods = append(policy.Roles[0].Methods, activityService+"ResetActivityExecution")
	policy.Roles[0].ReservationCarriers[0].Shapes[0].MaximumCount = 2
	controller := source.Program.Entrypoints[0]
	held, reset, settled := controller.Instructions[2], controller.Instructions[3], controller.Instructions[4]
	reset.InstructionId = "reset"
	reset.Instruction.GetInvokeRpc().Method = activityService + "ResetActivityExecution"
	settled.Guard = externalSuccess("reset")
	settled.Instruction.GetReadEvidence().Until = projectedEnum("status", enumspb.ACTIVITY_EXECUTION_STATUS_COMPLETED)
	controller.Instructions = []*testpilotspb.InstructionNode{controller.Instructions[0], controller.Instructions[1], held, reset, settled}
	source.Program.Entrypoints[1].Instructions = []*testpilotspb.InstructionNode{
		activityNode("pending", &testpilotspb.Instruction{Instruction: &testpilotspb.Instruction_ActivityAttemptWithholding{ActivityAttemptWithholding: &testpilotspb.ActivityAttemptWithholding{Mode: testpilotspb.ACTIVITY_WITHHOLDING_MODE_SDK_PENDING}}}),
		activityNode("complete", &testpilotspb.Instruction{Instruction: &testpilotspb.Instruction_Finish{Finish: &testpilotspb.Finish{Result: textLiteral("done")}}}),
	}
	source.Program.ActivityExternalSettlements = nil
	source.Program.ActivityResetSettlements = []*testpilotspb.ActivityResetSettlement{{
		Carrier: externalRef("call"), ActivityEntrypointId: "activity", ReservationOrdinal: 0, PendingSlotId: "pending-attempt",
		Held: externalRef("held"), ResetRequest: externalRef("reset"), FreshReservationOrdinal: 1, Settlement: externalRef("settled"),
		Cleanup: &testpilotspb.InstructionReference{EntrypointId: "cleanup", InstructionId: "terminate"},
	}}
	return source, catalog, policy
}

func TestActivityResetSettlementAdmitsTheDeclaredRestart(t *testing.T) {
	source, catalog, policy := resetFixture(t)
	prepared, err := Prepare(source, catalog, policy)
	require.NoError(t, err)
	require.True(t, proto.Equal(source.Program, prepared.Snapshot()))
	plan, carried := prepared.ReservationCarrier("controller", "call")
	require.True(t, carried)
	require.Equal(t, []contract.ReservationTopology{{EntrypointID: "activity", Kind: contract.ActivityEntrypoint, Count: 2, Restart: 1}}, plan.Reservations)
}

func TestActivityResetSettlementRefusesInvalidDeclarations(t *testing.T) {
	for _, test := range []struct {
		name   string
		mutate func(*testpilotspb.Case)
	}{
		{"undeclared reset", func(c *testpilotspb.Case) { c.Program.ActivityResetSettlements = nil }},
		{"duplicate numbering", func(c *testpilotspb.Case) {
			c.Program.ActivityResetSettlements = append(c.Program.ActivityResetSettlements, proto.CloneOf(c.Program.ActivityResetSettlements[0]))
		}},
		{"fresh group is not the next", func(c *testpilotspb.Case) { c.Program.ActivityResetSettlements[0].FreshReservationOrdinal = 0 }},
		{"fresh group past the script", func(c *testpilotspb.Case) {
			c.Program.ActivityResetSettlements[0].ReservationOrdinal, c.Program.ActivityResetSettlements[0].FreshReservationOrdinal = 1, 2
		}},
		{"held group answers", func(c *testpilotspb.Case) {
			c.Program.Entrypoints[1].Instructions[0], c.Program.Entrypoints[1].Instructions[1] = c.Program.Entrypoints[1].Instructions[1], c.Program.Entrypoints[1].Instructions[0]
		}},
		{"pending names an external answer", func(c *testpilotspb.Case) {
			c.Program.Entrypoints[1].Instructions[0].Instruction.GetActivityAttemptWithholding().ExternalSettlement = externalRef("reset")
		}},
		{"missing held read", func(c *testpilotspb.Case) { c.Program.ActivityResetSettlements[0].Held = nil }},
		{"reset is no reset", func(c *testpilotspb.Case) {
			c.Program.ActivityResetSettlements[0].ResetRequest = externalRef("settled")
		}},
		{"reset before held", func(c *testpilotspb.Case) {
			c.Program.Entrypoints[0].Instructions[3].Guard = externalSuccess("published")
		}},
		{"held before publication", func(c *testpilotspb.Case) {
			c.Program.Entrypoints[0].Instructions[2].Guard = succeeded("controller", "call")
		}},
		{"settlement before reset", func(c *testpilotspb.Case) { c.Program.Entrypoints[0].Instructions[4].Guard = externalSuccess("held") }},
		{"wrong publication type", func(c *testpilotspb.Case) {
			c.Program.Slots[1].Content = &testpilotspb.Slot_Value{Value: scalarSchema(testpilotspb.SCALAR_KIND_TEXT)}
		}},
		{"cleanup always false", func(c *testpilotspb.Case) {
			c.Program.Cleanup.Instructions[0].Guard = cel.Literal(&celpb.Value{Kind: &celpb.Value_BoolValue{BoolValue: false}})
		}},
		{"crossed namespace", func(c *testpilotspb.Case) {
			c.Program.Entrypoints[0].Instructions[3].Instruction.GetInvokeRpc().RequestAssignments[0].Value = textLiteral("namespace-id-not-name")
		}},
		{"Testpilot run is not the activity run", func(c *testpilotspb.Case) {
			c.Program.Entrypoints[0].Instructions[3].Instruction.GetInvokeRpc().RequestAssignments[2].Value = cel.Ref(&testpilotspb.Reference{Reference: &testpilotspb.Reference_Run{Run: &emptypb.Empty{}}})
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			c, catalog, policy := resetFixture(t)
			test.mutate(c)
			_, err := Prepare(c, catalog, policy)
			require.Error(t, err)
		})
	}
}

// The reservation a reset restarts at is the attempt numbered first again, and so is judged: a
// rewound SDK attempt is admitted there and nowhere else, and no reservation is relabeled.
func TestJudgeReservationNumbersADeclaredRestart(t *testing.T) {
	numbering := &testpilotspb.AttemptNumbering{First: 1, OneRun: true}
	outcome := func(attempt int32) *testpilotspb.InstructionOutcome {
		return &testpilotspb.InstructionOutcome{Status: testpilotspb.INSTRUCTION_OUTCOME_STATUS_SUCCEEDED, ActivityAttempt: &testpilotspb.ActivityAttempt{ActivityRunId: "activity-run", SdkAttempt: attempt, DeliveryId: "delivery", Response: testpilotspb.ACTIVITY_ATTEMPT_RESPONSE_OFFERED_COMPLETED}}
	}
	for _, test := range []struct {
		name             string
		restart, ordinal int64
		attempt          int32
		want             reservationVerdict
	}{
		{"fresh first attempt at the restart", 1, 1, 1, reservationRecorded},
		{"unrewound attempt at the restart", 1, 1, 2, reservationRejected},
		{"held attempt before the restart", 1, 0, 1, reservationRecorded},
		{"attempt after the restart", 1, 2, 2, reservationRecorded},
		{"rewind without a declared restart", 0, 1, 1, reservationRejected},
		{"ordinary retry without a restart", 0, 1, 2, reservationRecorded},
	} {
		t.Run(test.name, func(t *testing.T) {
			require.Equal(t, test.want, judgeReservation(contract.ActivityEntrypoint, numbering, test.restart, false, test.ordinal, "activity-run", outcome(test.attempt)))
		})
	}
}

func resetRuntime(t *testing.T) (*scheduler, *activityResetSettlement, *activationValues, *testsupport.Session) {
	t.Helper()
	source, catalog, policy := resetFixture(t)
	p, err := Prepare(source, catalog, policy)
	require.NoError(t, err)
	session := &testsupport.Session{OnInvokeRPC: func(context.Context, contract.Coordinate, string, protoreflect.MethodDescriptor, proto.Message) (contract.EffectHandle, error) {
		return testsupport.Completed(contract.EffectResult{}), nil
	}}
	s, err := newScheduler(p, "testpilot-run", source.CaseId, session, &schedulerMonitor{}, time.Now)
	require.NoError(t, err)
	_, err = s.recorder.publish(t.Context(), []*testpilotspb.RunEvent{{Kind: testpilotspb.RUN_EVENT_KIND_RUN_OPENED, SourceId: "opened"}}, nil)
	require.NoError(t, err)
	values, err := s.values.activate("controller", "controller.0")
	require.NoError(t, err)
	b := p.resets[0]
	s.values.slots[b.runSlot] = &celpb.Value{Kind: &celpb.Value_StringValue{StringValue: "actual-activity-run"}}
	s.values.externalRequests = map[*node]proto.Message{b.carrier: &workflowservice.StartActivityExecutionRequest{Namespace: "namespace", ActivityId: "activity-id"}}
	pending, err := anypb.New(&testpilotspb.ActivityAttempt{ActivityRunId: "actual-activity-run", NamespaceName: "namespace", ActivityId: "activity-id", DeliveryId: "held-delivery", SdkAttempt: 1, Response: testpilotspb.ACTIVITY_ATTEMPT_RESPONSE_PENDING})
	require.NoError(t, err)
	s.values.slots[b.source.PendingSlotId] = &celpb.Value{Kind: &celpb.Value_ObjectValue{ObjectValue: pending}}
	s.values.externalSucceeded = map[*node]bool{b.held: true}
	return s, b, values, session
}

// No reset reaches the server before the held attempt's pending record was published and a held
// Describe of the same run succeeded, and none crosses the learned execution.
func TestActivityResetRequestRefusesBeforeEffect(t *testing.T) {
	for _, test := range []struct {
		name   string
		mutate func(*valueStore, *activityResetSettlement, *workflowservice.ResetActivityExecutionRequest) proto.Message
	}{
		{"valid", func(_ *valueStore, _ *activityResetSettlement, r *workflowservice.ResetActivityExecutionRequest) proto.Message {
			return r
		}},
		{"no pending publication", func(v *valueStore, b *activityResetSettlement, r *workflowservice.ResetActivityExecutionRequest) proto.Message {
			delete(v.slots, b.source.PendingSlotId)
			return r
		}},
		{"no held receipt", func(v *valueStore, b *activityResetSettlement, r *workflowservice.ResetActivityExecutionRequest) proto.Message {
			delete(v.externalSucceeded, b.held)
			return r
		}},
		{"Testpilot run is not activity run", func(_ *valueStore, _ *activityResetSettlement, r *workflowservice.ResetActivityExecutionRequest) proto.Message {
			r.RunId = "testpilot-run"
			return r
		}},
		{"foreign activity", func(_ *valueStore, _ *activityResetSettlement, r *workflowservice.ResetActivityExecutionRequest) proto.Message {
			r.ActivityId = "foreign"
			return r
		}},
		{"workflow ID forbidden", func(_ *valueStore, _ *activityResetSettlement, r *workflowservice.ResetActivityExecutionRequest) proto.Message {
			r.WorkflowId = "workflow"
			return r
		}},
		{"published pending of another run", func(v *valueStore, b *activityResetSettlement, r *workflowservice.ResetActivityExecutionRequest) proto.Message {
			foreign, _ := anypb.New(&testpilotspb.ActivityAttempt{ActivityRunId: "foreign-run", NamespaceName: "namespace", ActivityId: "activity-id", DeliveryId: "held-delivery", SdkAttempt: 1, Response: testpilotspb.ACTIVITY_ATTEMPT_RESPONSE_PENDING})
			v.slots[b.source.PendingSlotId] = &celpb.Value{Kind: &celpb.Value_ObjectValue{ObjectValue: foreign}}
			return r
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			s, b, values, session := resetRuntime(t)
			r := &workflowservice.ResetActivityExecutionRequest{Namespace: "namespace", ActivityId: "activity-id", RunId: "actual-activity-run"}
			request := test.mutate(s.values, b, r)
			task := scheduledNode{activation: &scheduledActivation{values: values}, index: values.graph.index[b.reset.source.InstructionId]}
			_, _, _, err := s.admitDispatch(t.Context(), task, request, false)
			if test.name == "valid" {
				require.NoError(t, err)
				require.Equal(t, 1, session.Calls("InvokeRPC"))
				return
			}
			require.Error(t, err)
			require.Zero(t, session.Calls("InvokeRPC"))
		})
	}
}

// The held reservation's pending record becomes the publication only as the recorder appends it,
// and only the held reservation publishes: the fresh first attempt's record never does.
func TestActivityResetPublicationIsTheHeldReservationsRecordOnly(t *testing.T) {
	s, b, values, _ := resetRuntime(t)
	delete(s.values.slots, b.source.PendingSlotId)
	attempt := &testpilotspb.ActivityAttempt{ActivityRunId: "actual-activity-run", SdkAttempt: 1, DeliveryId: "held-delivery", Response: testpilotspb.ACTIVITY_ATTEMPT_RESPONSE_PENDING, NamespaceName: "namespace", ActivityId: "activity-id"}
	identity := func(ordinal int64) contract.ReservationIdentity {
		return contract.ReservationIdentity{Origin: contract.Coordinate{RunID: "testpilot-run", EntrypointID: "controller", InstructionID: "call", ActivationID: "controller.0", Attempt: 1}, EntrypointID: "activity", Ordinal: ordinal, ID: "reservation"}
	}
	slot, _, err := s.values.resetPublication(identity(1), attempt)
	require.NoError(t, err)
	require.Empty(t, slot, "the fresh group publishes nothing")
	_, _, err = s.values.resetPublication(identity(0), &testpilotspb.ActivityAttempt{ActivityRunId: "actual-activity-run", SdkAttempt: 1, DeliveryId: "held-delivery", Response: testpilotspb.ACTIVITY_ATTEMPT_RESPONSE_OFFERED_COMPLETED, NamespaceName: "namespace", ActivityId: "activity-id"})
	require.Error(t, err, "only a pending record publishes")
	completion := schedulerCompletion{reservation: &scheduledReservation{identity: identity(0), source: "controller.0.call.r0.i0", cause: "opened", values: values}, result: contract.EffectResult{Outcome: &testpilotspb.InstructionOutcome{Status: testpilotspb.INSTRUCTION_OUTCOME_STATUS_SUCCEEDED, ActivityAttempt: attempt}}}
	_, err = s.publishCompletion(t.Context(), completion)
	require.NoError(t, err)
	published := &testpilotspb.ActivityAttempt{}
	require.NoError(t, anypb.UnmarshalTo(s.values.slots[b.source.PendingSlotId].GetObjectValue(), published, proto.UnmarshalOptions{}))
	require.True(t, proto.Equal(attempt, published))
	_, commit, err := s.values.resetPublication(identity(0), attempt)
	require.NoError(t, err)
	require.Error(t, commit(), "a second publication of the held record is refused")
}

func TestActivityResetReceiptsProveHeldAndTerminalStates(t *testing.T) {
	for _, test := range []struct {
		name   string
		held   bool
		mutate func(*workflowservice.DescribeActivityExecutionResponse)
		valid  bool
	}{
		{"held", true, func(*workflowservice.DescribeActivityExecutionResponse) {}, true},
		{"held but paused", true, func(r *workflowservice.DescribeActivityExecutionResponse) {
			r.Info.RunState = enumspb.PENDING_ACTIVITY_STATE_PAUSE_REQUESTED
		}, false},
		{"held of another run", true, func(r *workflowservice.DescribeActivityExecutionResponse) { r.Info.RunId = "testpilot-run" }, false},
		{"terminal", false, func(*workflowservice.DescribeActivityExecutionResponse) {}, true},
		{"terminal without close", false, func(r *workflowservice.DescribeActivityExecutionResponse) { r.Info.CloseTime = nil }, false},
		{"still running", false, func(r *workflowservice.DescribeActivityExecutionResponse) {
			r.Info.Status = enumspb.ACTIVITY_EXECUTION_STATUS_RUNNING
		}, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			s, b, values, _ := resetRuntime(t)
			read := b.settlement
			response := &workflowservice.DescribeActivityExecutionResponse{RunId: "actual-activity-run", Info: &activitypb.ActivityExecutionInfo{RunId: "actual-activity-run", ActivityId: "activity-id", Status: enumspb.ACTIVITY_EXECUTION_STATUS_COMPLETED, CloseTime: timestamppb.Now()}}
			if test.held {
				read = b.held
				response.Info.Status, response.Info.RunState, response.Info.CloseTime = enumspb.ACTIVITY_EXECUTION_STATUS_RUNNING, enumspb.PENDING_ACTIVITY_STATE_STARTED, nil
			}
			test.mutate(response)
			s.values.externalRequests[read] = &workflowservice.DescribeActivityExecutionRequest{Namespace: "namespace", ActivityId: "activity-id", RunId: "actual-activity-run"}
			task := scheduledNode{activation: &scheduledActivation{values: values}, index: values.graph.index[read.source.InstructionId]}
			err := s.values.checkResetReceipt(task.activation.values.graph.nodes[task.index], contract.EffectResult{Outcome: &testpilotspb.InstructionOutcome{Status: testpilotspb.INSTRUCTION_OUTCOME_STATUS_SUCCEEDED}, Response: response})
			if test.valid {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
		})
	}
}
