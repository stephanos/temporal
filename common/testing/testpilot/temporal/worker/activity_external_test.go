package worker

import (
	"context"
	"testing"
	"time"

	celpb "cel.dev/expr"
	"github.com/stretchr/testify/require"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/activity"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/testpilot"
	cel "go.temporal.io/server/common/testing/testpilot/cel"
	"go.temporal.io/server/common/testing/testpilot/internal/testsupport"
	"go.temporal.io/server/common/testing/testpilot/internal/testsupport/facadetest"
	"go.temporal.io/server/common/testing/testpilot/temporal/internal/delivery"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/emptypb"
)

func externallyHeldActivity(program *testpilotspb.Program) {
	standaloneActivity(program)
	ref := func(id string) *testpilotspb.InstructionReference {
		return &testpilotspb.InstructionReference{EntrypointId: "controller", InstructionId: id}
	}
	slot := func(id string) *testpilotspb.Expression {
		return cel.Ref(&testpilotspb.Reference{Reference: &testpilotspb.Reference_SlotId{SlotId: id}})
	}
	success := func(id string) *testpilotspb.Expression {
		status := cel.Ref(&testpilotspb.Reference{Reference: &testpilotspb.Reference_Outcome{Outcome: &testpilotspb.InstructionOutcomeReference{Instruction: ref(id), Field: testpilotspb.INSTRUCTION_OUTCOME_FIELD_STATUS}}})
		return cel.All(cel.Present(status), cel.Compare("_==_", status, cel.Literal(cel.Enum(testpilotspb.INSTRUCTION_OUTCOME_STATUS_SUCCEEDED))))
	}
	guard := func(previous string) *testpilotspb.Expression {
		return cel.All([]*testpilotspb.Expression{success("start-activity"), success(previous)}...)
	}
	messageType := func(name string) *testpilotspb.ValueType {
		return &testpilotspb.ValueType{Shape: &testpilotspb.ValueType_Singular{Singular: &testpilotspb.SingularType{Type: &testpilotspb.SingularType_Message{Message: &testpilotspb.NamedType{ProtobufType: name}}}}}
	}
	program.Slots = []*testpilotspb.Slot{{SlotId: "activity-run", Content: &testpilotspb.Slot_Value{Value: &testpilotspb.ValueType{Shape: &testpilotspb.ValueType_Singular{Singular: &testpilotspb.SingularType{Type: &testpilotspb.SingularType_Scalar{Scalar: &testpilotspb.ScalarType{Kind: testpilotspb.SCALAR_KIND_TEXT}}}}}}}, {SlotId: "pending", Content: &testpilotspb.Slot_Value{Value: messageType("temporal.server.api.testpilot.v1.ActivityAttempt")}}}
	start := program.Entrypoints[0].Instructions[0]
	start.Instruction.GetInvokeRpc().ResponseReads = []*testpilotspb.ResponseRead{{Path: "run_id", Targets: []*testpilotspb.ReadTarget{{Target: &testpilotspb.ReadTarget_SlotId{SlotId: "activity-run"}}}}}
	assignments := func() []*testpilotspb.RequestAssignment {
		var result []*testpilotspb.RequestAssignment
		for _, a := range start.Instruction.GetInvokeRpc().RequestAssignments {
			if a.Target == "namespace" || a.Target == "activity_id" {
				result = append(result, proto.CloneOf(a))
			}
		}
		return append(result, &testpilotspb.RequestAssignment{Target: "run_id", Value: slot("activity-run")})
	}
	truth := cel.Literal(&celpb.Value{Kind: &celpb.Value_BoolValue{BoolValue: true}})
	await := &testpilotspb.InstructionNode{InstructionId: "published", Guard: success("start-activity"), Limits: facadetest.Bounds(), Instruction: &testpilotspb.Instruction{Instruction: &testpilotspb.Instruction_AwaitSlot{AwaitSlot: &testpilotspb.AwaitSlot{SlotId: "pending"}}}}
	read := func(id, previous string) *testpilotspb.InstructionNode {
		return &testpilotspb.InstructionNode{InstructionId: id, Guard: guard(previous), Limits: facadetest.Bounds(), Instruction: &testpilotspb.Instruction{Instruction: &testpilotspb.Instruction_ReadEvidence{ReadEvidence: &testpilotspb.ReadEvidence{EvidenceId: id, EndpointRoleId: "endpoint", RequestAssignments: assignments(), Until: proto.CloneOf(truth)}}}}
	}
	held, settled := read("held", "published"), read("settled", "answer")
	answer := &testpilotspb.InstructionNode{InstructionId: "answer", Guard: guard("held"), Limits: facadetest.Bounds(), Instruction: &testpilotspb.Instruction{Instruction: &testpilotspb.Instruction_InvokeRpc{InvokeRpc: &testpilotspb.InvokeRpc{EndpointRoleId: "endpoint", Method: "/temporal.api.workflowservice.v1.WorkflowService/RespondActivityTaskFailedById", RequestAssignments: assignments()}}}}
	program.Entrypoints[0].Instructions = []*testpilotspb.InstructionNode{start, await, held, answer, settled}
	program.Entrypoints[1].Instructions = []*testpilotspb.InstructionNode{{InstructionId: "pending", Limits: facadetest.Bounds(), Instruction: &testpilotspb.Instruction{Instruction: &testpilotspb.Instruction_ActivityAttemptWithholding{ActivityAttemptWithholding: &testpilotspb.ActivityAttemptWithholding{Mode: testpilotspb.ACTIVITY_WITHHOLDING_MODE_SDK_PENDING, ExternalSettlement: ref("answer")}}}}}
	program.ActivityExternalSettlements = []*testpilotspb.ActivityExternalSettlement{{Carrier: ref("start-activity"), ActivityEntrypointId: "activity", PendingSlotId: "pending", Held: ref("held"), Answer: ref("answer"), Settlement: ref("settled"), Cleanup: &testpilotspb.InstructionReference{EntrypointId: "cleanup", InstructionId: "terminate"}}}
	program.Cleanup = &testpilotspb.Cleanup{EntrypointId: "cleanup", Instructions: []*testpilotspb.InstructionNode{{InstructionId: "terminate", Guard: cel.Present(slot("activity-run")), Limits: facadetest.Bounds(), Instruction: &testpilotspb.Instruction{Instruction: &testpilotspb.Instruction_InvokeRpc{InvokeRpc: &testpilotspb.InvokeRpc{EndpointRoleId: "endpoint", Method: "/temporal.api.workflowservice.v1.WorkflowService/TerminateActivityExecution", RequestAssignments: assignments()}}}}}}
	program.Observations = []*testpilotspb.Observation{{ObservationId: "external-evidence", Type: messageType("temporal.server.api.testpilot.v1.CorrelatedEvidence")}}
	program.Cleanup.Instructions[0].Guard = cel.All([]*testpilotspb.Expression{success("start-activity"), cel.Not(success("settled"))}...)
	program.Evidence = nil
	for _, id := range []string{"held", "settled"} {
		program.Evidence = append(program.Evidence, &testpilotspb.EvidenceDeclaration{EvidenceId: id, EvidenceSource: id, Operation: cel.Path(cel.Ref(&testpilotspb.Reference{Reference: &testpilotspb.Reference_ProjectedValue{ProjectedValue: &emptypb.Empty{}}}), "activity_id"), Source: &testpilotspb.EvidenceDeclaration_Read{Read: &testpilotspb.ReadSource{Method: "/temporal.api.workflowservice.v1.WorkflowService/DescribeActivityExecution", Path: "info"}}, Scope: testsupport.LiteralExpressions([]*testpilotspb.NamedValue{{FieldId: "run", Value: &celpb.Value{Kind: &celpb.Value_StringValue{StringValue: "external"}}}}), Kind: id})
	}
}

func externalActivityProfile(t *testing.T, profile *testpilot.ProfileSpec) {
	t.Helper()
	catalog, err := testpilot.NewCatalog(testsupport.DescriptorClosure(workflowservice.File_temporal_api_workflowservice_v1_service_proto, testpilotspb.File_temporal_server_api_testpilot_v1_run_proto))
	require.NoError(t, err)
	profile.Catalog = catalog
	profile.Opcodes = append(profile.Opcodes, testpilot.AwaitSlot, testpilot.ReadEvidence)
	profile.Roles[0].Methods = append(profile.Roles[0].Methods, "/temporal.api.workflowservice.v1.WorkflowService/DescribeActivityExecution", "/temporal.api.workflowservice.v1.WorkflowService/RespondActivityTaskFailedById", "/temporal.api.workflowservice.v1.WorkflowService/TerminateActivityExecution")
}

func TestTransportExternalPendingNeedsNoHeartbeatTimer(t *testing.T) {
	server, session, task := transportSession(t, externallyHeldActivity, func(profile *testpilot.ProfileSpec) { externalActivityProfile(t, profile) })
	bounded, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()
	firstTask := task("token-external", 1)
	require.Nil(t, firstTask.HeartbeatTimeout)
	server.tasks <- firstTask
	result, err := session.reservations["reservation-1"].Wait(bounded)
	require.NoError(t, err)
	requireOutcome(t, answered("activity-run", 1, tokenDigest("token-external"), testpilotspb.ACTIVITY_ATTEMPT_RESPONSE_PENDING), result)
	require.NoError(t, session.reservations["reservation-1"].Drain(bounded))
	require.Empty(t, server.completed)
	require.Empty(t, server.failed)
	require.Empty(t, server.canceled)
	require.Empty(t, server.heartbeats)
	replay := delivery.ActivityDelivery{Header: firstTask.Header, Namespace: "namespace", ActivityID: "activity-id", ActivityType: "activity-type", TaskQueue: "task-queue", ActivityRunID: "activity-run", Attempt: 1, DeliveryID: tokenDigest("token-external")}
	got, replayErr := activityWorker(session.host, session.definition).activateActivity(bounded, replay, func(context.Context) (interface{}, error) {
		t.Fatal("duplicate pending delivery executed worker")
		return nil, nil
	})
	require.Nil(t, got)
	require.Same(t, activity.ErrResultPending, replayErr)
	require.NoError(t, session.Close(bounded))
}
