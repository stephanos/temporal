package execution

import (
	"context"
	"slices"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	activitypb "go.temporal.io/api/activity/v1"
	commonpb "go.temporal.io/api/common/v1"
	enumspb "go.temporal.io/api/enums/v1"
	failurepb "go.temporal.io/api/failure/v1"
	"go.temporal.io/api/workflowservice/v1"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/testpilot/contract"
	"go.temporal.io/server/common/testing/testpilot/internal/ir"
	"go.temporal.io/server/common/testing/testpilot/internal/testsupport"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const activityService = "/temporal.api.workflowservice.v1.WorkflowService/"

func externalRef(id string) *testpilotspb.InstructionReference {
	return &testpilotspb.InstructionReference{EntrypointId: "controller", InstructionId: id}
}

func externalSuccess(id string) *testpilotspb.Expression {
	return &testpilotspb.Expression{Expression: &testpilotspb.Expression_All{All: &testpilotspb.AllExpression{Operands: []*testpilotspb.Expression{succeeded("controller", "call"), succeeded("controller", id)}}}}
}

func externalAssignments() []*testpilotspb.RequestAssignment {
	return []*testpilotspb.RequestAssignment{
		{Target: "namespace", Value: textLiteral("namespace")},
		{Target: "activity_id", Value: textLiteral("activity-id")},
		{Target: "run_id", Value: slot("activity-run")},
	}
}

func projectedEnum(field, value string) *testpilotspb.Expression {
	return &testpilotspb.Expression{Expression: &testpilotspb.Expression_Compare{Compare: &testpilotspb.CompareExpression{
		Operator: testpilotspb.COMPARISON_OPERATOR_EQUAL,
		Left: &testpilotspb.Expression{Expression: &testpilotspb.Expression_Path{Path: &testpilotspb.PathExpression{
			Operand: &testpilotspb.Expression{Expression: &testpilotspb.Expression_Reference{Reference: &testpilotspb.Reference{Reference: &testpilotspb.Reference_ProjectedValue{ProjectedValue: &testpilotspb.ProjectedValueReference{}}}}}, Path: field,
		}}},
		Right: &testpilotspb.Expression{Expression: &testpilotspb.Expression_Literal{Literal: &testpilotspb.Value{Value: &testpilotspb.Value_EnumValue{EnumValue: &testpilotspb.EnumValue{Name: value}}}}},
	}}}
}

func externalFixture(t *testing.T, canceled bool) (*testpilotspb.Case, *ir.Catalog, Profile) {
	t.Helper()
	source, _, policy := activityFixture(t)
	descriptors := testsupport.DescriptorClosure(workflowservice.File_temporal_api_workflowservice_v1_service_proto, testpilotspb.File_temporal_server_api_testpilot_v1_program_proto, testpilotspb.File_temporal_server_api_testpilot_v1_run_proto)
	catalog, err := ir.NewCatalog(descriptors)
	require.NoError(t, err)
	policy.CatalogIdentity = catalog.Identity()
	for _, opcode := range []contract.Opcode{contract.AwaitSlot, contract.ReadEvidence, contract.ActivityAttemptWithholding} {
		if !slices.Contains(policy.Opcodes, opcode) {
			policy.Opcodes = append(policy.Opcodes, opcode)
		}
	}
	policy.Roles[0].Methods = []string{activityService + "StartActivityExecution", activityService + "DescribeActivityExecution", activityService + "RespondActivityTaskFailedById", activityService + "RespondActivityTaskCanceledById", activityService + "RequestCancelActivityExecution", activityService + "TerminateActivityExecution"}
	policy.Roles[0].ReservationCarriers[0].Method = activityService + "StartActivityExecution"
	source.Program.Slots = []*testpilotspb.Slot{
		{SlotId: "activity-run", Content: &testpilotspb.Slot_Value{Value: scalarSchema(testpilotspb.SCALAR_KIND_TEXT)}},
		{SlotId: "pending-attempt", Content: &testpilotspb.Slot_Value{Value: &testpilotspb.ValueType{Shape: &testpilotspb.ValueType_Singular{Singular: &testpilotspb.SingularType{Type: &testpilotspb.SingularType_Message{Message: &testpilotspb.NamedType{ProtobufType: "temporal.server.api.testpilot.v1.ActivityAttempt"}}}}}}},
	}
	start := rpcNode("call")
	start.Instruction = &testpilotspb.Instruction{Instruction: &testpilotspb.Instruction_InvokeRpc{InvokeRpc: &testpilotspb.InvokeRpc{EndpointRoleId: "endpoint", Method: activityService + "StartActivityExecution", RequestAssignments: []*testpilotspb.RequestAssignment{
		{Target: "namespace", Value: textLiteral("namespace")}, {Target: "activity_id", Value: textLiteral("activity-id")},
	}, ResponseReads: []*testpilotspb.ResponseRead{{Path: "run_id", Cardinality: testpilotspb.READ_CARDINALITY_ONE, Targets: []*testpilotspb.ReadTarget{{Target: &testpilotspb.ReadTarget_SlotId{SlotId: "activity-run"}}}}}}}}
	await := activityNode("published", &testpilotspb.Instruction{Instruction: &testpilotspb.Instruction_AwaitSlot{AwaitSlot: &testpilotspb.AwaitSlot{SlotId: "pending-attempt"}}})
	await.Guard = succeeded("controller", "call")
	runState := enumspb.PENDING_ACTIVITY_STATE_STARTED
	terminal := enumspb.ACTIVITY_EXECUTION_STATUS_FAILED
	answerMethod := "RespondActivityTaskFailedById"
	if canceled {
		runState, terminal, answerMethod = enumspb.PENDING_ACTIVITY_STATE_CANCEL_REQUESTED, enumspb.ACTIVITY_EXECUTION_STATUS_CANCELED, "RespondActivityTaskCanceledById"
	}
	held := rpcNode("held")
	held.Guard = externalSuccess("published")
	held.Instruction = &testpilotspb.Instruction{Instruction: &testpilotspb.Instruction_ReadEvidence{ReadEvidence: &testpilotspb.ReadEvidence{EvidenceId: "held", EndpointRoleId: "endpoint", RequestAssignments: externalAssignments(), Once: true, Until: &testpilotspb.Expression{Expression: &testpilotspb.Expression_All{All: &testpilotspb.AllExpression{Operands: []*testpilotspb.Expression{
		projectedEnum("status", ir.EnumName(enumspb.ACTIVITY_EXECUTION_STATUS_RUNNING)),
		projectedEnum("run_state", ir.EnumName(runState)),
	}}}}}}}
	answer := rpcNode("answer")
	answer.Guard = externalSuccess("held")
	answer.Instruction = &testpilotspb.Instruction{Instruction: &testpilotspb.Instruction_InvokeRpc{InvokeRpc: &testpilotspb.InvokeRpc{EndpointRoleId: "endpoint", Method: activityService + answerMethod, RequestAssignments: externalAssignments()}}}
	settlement := rpcNode("settled")
	settlement.Guard = externalSuccess("answer")
	settlement.Instruction = &testpilotspb.Instruction{Instruction: &testpilotspb.Instruction_ReadEvidence{ReadEvidence: &testpilotspb.ReadEvidence{EvidenceId: "settled", EndpointRoleId: "endpoint", RequestAssignments: externalAssignments(), Once: true, Until: projectedEnum("status", ir.EnumName(terminal))}}}
	controller := []*testpilotspb.InstructionNode{start, await}
	declaration := &testpilotspb.ActivityExternalSettlement{Carrier: externalRef("call"), ActivityEntrypointId: "activity", PendingSlotId: "pending-attempt", Held: externalRef("held"), Answer: externalRef("answer"), Settlement: externalRef("settled"), Cleanup: &testpilotspb.InstructionReference{EntrypointId: "cleanup", InstructionId: "terminate"}}
	if canceled {
		request := rpcNode("request-cancel")
		request.Guard = externalSuccess("published")
		request.Instruction = &testpilotspb.Instruction{Instruction: &testpilotspb.Instruction_InvokeRpc{InvokeRpc: &testpilotspb.InvokeRpc{EndpointRoleId: "endpoint", Method: activityService + "RequestCancelActivityExecution", RequestAssignments: externalAssignments()}}}
		controller = append(controller, request)
		held.Guard = externalSuccess("request-cancel")
		declaration.RequestCancel = externalRef("request-cancel")
		settlement.Instruction.GetReadEvidence().RequestAssignments = append(settlement.Instruction.GetReadEvidence().RequestAssignments, &testpilotspb.RequestAssignment{Target: "include_outcome", Value: &testpilotspb.Expression{Expression: &testpilotspb.Expression_Literal{Literal: &testpilotspb.Value{Value: &testpilotspb.Value_BoolValue{BoolValue: true}}}}})
	}
	source.Program.Entrypoints[0].Instructions = append(controller, held, answer, settlement)
	source.Program.Entrypoints[1].Instructions = []*testpilotspb.InstructionNode{activityNode("pending", &testpilotspb.Instruction{Instruction: &testpilotspb.Instruction_ActivityAttemptWithholding{ActivityAttemptWithholding: &testpilotspb.ActivityAttemptWithholding{Mode: testpilotspb.ACTIVITY_WITHHOLDING_MODE_SDK_PENDING, ExternalSettlement: externalRef("answer")}}})}
	cleanup := rpcNode("terminate")
	cleanup.Guard = &testpilotspb.Expression{Expression: &testpilotspb.Expression_All{All: &testpilotspb.AllExpression{Operands: []*testpilotspb.Expression{succeeded("controller", "call"), {Expression: &testpilotspb.Expression_Not{Not: &testpilotspb.NotExpression{Operand: succeeded("controller", "settled")}}}}}}}
	cleanup.Instruction = &testpilotspb.Instruction{Instruction: &testpilotspb.Instruction_InvokeRpc{InvokeRpc: &testpilotspb.InvokeRpc{EndpointRoleId: "endpoint", Method: activityService + "TerminateActivityExecution", RequestAssignments: externalAssignments()}}}
	source.Program.Cleanup = &testpilotspb.Cleanup{EntrypointId: "cleanup", Instructions: []*testpilotspb.InstructionNode{cleanup}}
	source.Program.ActivityExternalSettlements = []*testpilotspb.ActivityExternalSettlement{declaration}
	source.Program.Evidence = nil
	for _, id := range []string{"held", "settled"} {
		source.Program.Evidence = append(source.Program.Evidence, &testpilotspb.EvidenceDeclaration{EvidenceId: id, EvidenceSource: id, Source: &testpilotspb.EvidenceDeclaration_Read{Read: &testpilotspb.ReadSource{Method: activityService + "DescribeActivityExecution", Path: "info", Single: true}}, Operation: "activity_id", Scope: []*testpilotspb.NamedValue{{FieldId: "run", Value: &testpilotspb.Value{Value: &testpilotspb.Value_TextValue{TextValue: "scope"}}}}})
	}
	source.Program.Observations = append(source.Program.Observations, &testpilotspb.Observation{ObservationId: "evidence", Type: &testpilotspb.ValueType{Shape: &testpilotspb.ValueType_Singular{Singular: &testpilotspb.SingularType{Type: &testpilotspb.SingularType_Message{Message: &testpilotspb.NamedType{ProtobufType: "temporal.server.api.testpilot.v1.CorrelatedEvidence"}}}}}})
	return source, catalog, policy
}

func TestActivityExternalSettlementAdmitsTimerFreePendingPublication(t *testing.T) {
	for _, canceled := range []bool{false, true} {
		source, catalog, policy := externalFixture(t, canceled)
		prepared, err := Prepare(source, catalog, policy)
		require.NoError(t, err)
		require.True(t, proto.Equal(source.Program, prepared.Snapshot()))
	}
}

func TestActivityExternalSettlementRejectsIncompleteBasis(t *testing.T) {
	for _, test := range []struct {
		name   string
		mutate func(*testpilotspb.Case)
	}{
		{"missing held", func(c *testpilotspb.Case) { c.Program.ActivityExternalSettlements[0].Held = nil }},
		{"missing cleanup", func(c *testpilotspb.Case) { c.Program.ActivityExternalSettlements[0].Cleanup = nil }},
		{"cleanup always false", func(c *testpilotspb.Case) {
			c.Program.Cleanup.Instructions[0].Guard = &testpilotspb.Expression{Expression: &testpilotspb.Expression_Literal{Literal: &testpilotspb.Value{Value: &testpilotspb.Value_BoolValue{BoolValue: false}}}}
		}},
		{"wrong ordinal", func(c *testpilotspb.Case) { c.Program.ActivityExternalSettlements[0].ReservationOrdinal = 1 }},
		{"wrong carrier", func(c *testpilotspb.Case) { c.Program.ActivityExternalSettlements[0].Carrier = externalRef("answer") }},
		{"wrong publication type", func(c *testpilotspb.Case) {
			c.Program.Slots[1].Content = &testpilotspb.Slot_Value{Value: scalarSchema(testpilotspb.SCALAR_KIND_TEXT)}
		}},
		{"missing withholding basis", func(c *testpilotspb.Case) {
			c.Program.Entrypoints[1].Instructions[0].Instruction.GetActivityAttemptWithholding().ExternalSettlement = nil
		}},
		{"undeclared withholding basis", func(c *testpilotspb.Case) { c.Program.ActivityExternalSettlements = nil }},
		{"duplicate basis", func(c *testpilotspb.Case) {
			c.Program.ActivityExternalSettlements = append(c.Program.ActivityExternalSettlements, proto.CloneOf(c.Program.ActivityExternalSettlements[0]))
		}},
		{"wrong namespace", func(c *testpilotspb.Case) {
			c.Program.Entrypoints[0].Instructions[4].Instruction.GetInvokeRpc().RequestAssignments[0].Value = textLiteral("namespace-id-not-name")
		}},
		{"answer before held", func(c *testpilotspb.Case) {
			c.Program.Entrypoints[0].Instructions[4].Guard = externalSuccess("published")
		}},
		{"cancel before publication", func(c *testpilotspb.Case) {
			c.Program.Entrypoints[0].Instructions[2].Guard = succeeded("controller", "call")
		}},
		{"missing cancel", func(c *testpilotspb.Case) { c.Program.ActivityExternalSettlements[0].RequestCancel = nil }},
		{"missing outcome", func(c *testpilotspb.Case) {
			r := c.Program.Entrypoints[0].Instructions[5].Instruction.GetReadEvidence()
			r.RequestAssignments = r.RequestAssignments[:3]
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			c, catalog, policy := externalFixture(t, true)
			test.mutate(c)
			_, err := Prepare(c, catalog, policy)
			require.Error(t, err)
		})
	}
}

func TestActivityExternalPublicationFollowsRecorderAppend(t *testing.T) {
	source, catalog, policy := externalFixture(t, false)
	p, err := Prepare(source, catalog, policy)
	require.NoError(t, err)
	entered, release := make(chan struct{}), make(chan struct{})
	monitor := schedulerMonitor{observe: func(event *testpilotspb.RunEvent) Decision {
		if event.GetOutcome().GetActivityAttempt().GetResponse() == testpilotspb.ACTIVITY_ATTEMPT_RESPONSE_PENDING {
			close(entered)
			<-release
		}
		return Continue
	}}
	s, err := newScheduler(p, "testpilot-run", source.CaseId, &testsupport.Session{}, &monitor, time.Now)
	require.NoError(t, err)
	_, err = s.recorder.publish(t.Context(), []*testpilotspb.RunEvent{{Kind: testpilotspb.RUN_EVENT_KIND_RUN_OPENED, SourceId: "opened"}}, nil)
	require.NoError(t, err)
	values, err := s.values.activate("controller", "controller.0")
	require.NoError(t, err)
	s.values.externalRequests = map[*node]proto.Message{p.external[0].carrier: &workflowservice.StartActivityExecutionRequest{Namespace: "namespace", ActivityId: "activity-id"}}
	require.Nil(t, s.values.slots["activity-run"])
	identity := contract.ReservationIdentity{Origin: contract.Coordinate{RunID: "testpilot-run", EntrypointID: "controller", InstructionID: "call", ActivationID: "controller.0", Attempt: 1}, EntrypointID: "activity", Ordinal: 0, ID: "reservation"}
	completion := schedulerCompletion{reservation: &scheduledReservation{identity: identity, source: "controller.0.call.r0.i0", cause: "opened", values: values}, result: contract.EffectResult{Outcome: &testpilotspb.InstructionOutcome{Status: testpilotspb.INSTRUCTION_OUTCOME_STATUS_SUCCEEDED, ActivityAttempt: &testpilotspb.ActivityAttempt{ActivityRunId: "actual-activity-run", SdkAttempt: 1, DeliveryId: "actual-delivery", Response: testpilotspb.ACTIVITY_ATTEMPT_RESPONSE_PENDING, NamespaceName: "namespace", ActivityId: "activity-id"}}}}
	done := make(chan error, 1)
	go func() { _, err := s.publishCompletion(t.Context(), completion); done <- err }()
	<-entered
	s.values.mu.Lock()
	before := s.values.slots["pending-attempt"]
	s.values.mu.Unlock()
	require.Nil(t, before)
	waitCtx, cancel := context.WithCancel(t.Context())
	defer cancel()
	awaited := make(chan error, 1)
	go func() { awaited <- values.awaitSlot(waitCtx, "pending-attempt") }()
	select {
	case err := <-awaited:
		t.Fatalf("publication became visible during append: %v", err)
	case <-time.After(10 * time.Millisecond):
	}
	close(release)
	require.NoError(t, <-done)
	select {
	case err := <-awaited:
		require.NoError(t, err)
	case <-time.After(time.Second):
		t.Fatal("publication never woke AwaitSlot")
	}
	s.values.mu.Lock()
	published := proto.CloneOf(s.values.slots["pending-attempt"])
	s.values.mu.Unlock()
	actual := &testpilotspb.ActivityAttempt{}
	require.NoError(t, anypb.UnmarshalTo(published.GetMessageValue(), actual, proto.UnmarshalOptions{}))
	require.True(t, proto.Equal(completion.result.Outcome.ActivityAttempt, actual))
}

func externalRuntime(t *testing.T, canceled bool) (*scheduler, *activityExternalSettlement, *activationValues, *testsupport.Session) {
	t.Helper()
	source, catalog, policy := externalFixture(t, canceled)
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
	b := p.external[0]
	s.values.slots[b.runSlot] = &testpilotspb.Value{Value: &testpilotspb.Value_TextValue{TextValue: "actual-activity-run"}}
	s.values.externalRequests = map[*node]proto.Message{b.carrier: &workflowservice.StartActivityExecutionRequest{Namespace: "namespace", ActivityId: "activity-id"}}
	pending, err := anypb.New(&testpilotspb.ActivityAttempt{ActivityRunId: "actual-activity-run", NamespaceName: "namespace", ActivityId: "activity-id", DeliveryId: "actual-delivery", SdkAttempt: 1, Response: testpilotspb.ACTIVITY_ATTEMPT_RESPONSE_PENDING})
	require.NoError(t, err)
	s.values.slots[b.source.PendingSlotId] = &testpilotspb.Value{Value: &testpilotspb.Value_MessageValue{MessageValue: pending}}
	s.values.externalSucceeded = map[*node]bool{b.held: true}
	if b.cancel != nil {
		s.values.externalSucceeded[b.cancel] = true
	}
	return s, b, values, session
}

func TestActivityExternalIdentityRejectsBeforeEffect(t *testing.T) {
	for _, test := range []struct {
		name   string
		mutate func(*valueStore, *activityExternalSettlement, *workflowservice.RespondActivityTaskFailedByIdRequest) proto.Message
	}{
		{"namespace ID is not name", func(_ *valueStore, _ *activityExternalSettlement, r *workflowservice.RespondActivityTaskFailedByIdRequest) proto.Message {
			r.Namespace = "namespace-uuid"
			return r
		}},
		{"foreign activity", func(_ *valueStore, _ *activityExternalSettlement, r *workflowservice.RespondActivityTaskFailedByIdRequest) proto.Message {
			r.ActivityId = "foreign"
			return r
		}},
		{"Testpilot run is not activity run", func(_ *valueStore, _ *activityExternalSettlement, r *workflowservice.RespondActivityTaskFailedByIdRequest) proto.Message {
			r.RunId = "testpilot-run"
			return r
		}},
		{"empty run is not latest", func(_ *valueStore, _ *activityExternalSettlement, r *workflowservice.RespondActivityTaskFailedByIdRequest) proto.Message {
			r.RunId = ""
			return r
		}},
		{"workflow ID forbidden", func(_ *valueStore, _ *activityExternalSettlement, r *workflowservice.RespondActivityTaskFailedByIdRequest) proto.Message {
			r.WorkflowId = "workflow"
			return r
		}},
		{"wrong request descriptor", func(_ *valueStore, _ *activityExternalSettlement, _ *workflowservice.RespondActivityTaskFailedByIdRequest) proto.Message {
			return &workflowservice.RespondActivityTaskCanceledByIdRequest{Namespace: "namespace", ActivityId: "activity-id", RunId: "actual-activity-run"}
		}},
		{"no pending publication", func(v *valueStore, b *activityExternalSettlement, r *workflowservice.RespondActivityTaskFailedByIdRequest) proto.Message {
			delete(v.slots, b.source.PendingSlotId)
			return r
		}},
		{"no held receipt", func(v *valueStore, b *activityExternalSettlement, r *workflowservice.RespondActivityTaskFailedByIdRequest) proto.Message {
			delete(v.externalSucceeded, b.held)
			return r
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			s, b, values, session := externalRuntime(t, false)
			r := &workflowservice.RespondActivityTaskFailedByIdRequest{Namespace: "namespace", ActivityId: "activity-id", RunId: "actual-activity-run"}
			request := test.mutate(s.values, b, r)
			task := scheduledNode{activation: &scheduledActivation{values: values}, index: values.graph.index[b.answer.source.InstructionId]}
			_, _, _, err := s.admitDispatch(t.Context(), task, request, false)
			require.Error(t, err)
			require.Zero(t, session.Calls("InvokeRPC"))
		})
	}
}

func TestActivityExternalCanceledReceiptRetainsRequesterAndDetails(t *testing.T) {
	for _, test := range []struct {
		name   string
		mutate func(*workflowservice.DescribeActivityExecutionResponse)
	}{
		{"valid", func(*workflowservice.DescribeActivityExecutionResponse) {}},
		{"wrong run", func(r *workflowservice.DescribeActivityExecutionResponse) { r.Info.RunId = "testpilot-run" }},
		{"wrong activity", func(r *workflowservice.DescribeActivityExecutionResponse) { r.Info.ActivityId = "foreign" }},
		{"still running", func(r *workflowservice.DescribeActivityExecutionResponse) {
			r.Info.Status = enumspb.ACTIVITY_EXECUTION_STATUS_RUNNING
		}},
		{"still cancellation requested", func(r *workflowservice.DescribeActivityExecutionResponse) {
			r.Info.RunState = enumspb.PENDING_ACTIVITY_STATE_CANCEL_REQUESTED
		}},
		{"no positive close", func(r *workflowservice.DescribeActivityExecutionResponse) { r.Info.CloseTime = nil }},
		{"no outcome", func(r *workflowservice.DescribeActivityExecutionResponse) { r.Outcome = nil }},
		{"answer caller is not requester", func(r *workflowservice.DescribeActivityExecutionResponse) {
			r.Outcome.GetFailure().GetCanceledFailureInfo().Identity = "external-answer-controller"
		}},
		{"wrong requested details", func(r *workflowservice.DescribeActivityExecutionResponse) {
			r.Outcome.GetFailure().GetCanceledFailureInfo().Details.Payloads[0].Data = []byte("wrong")
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			s, b, values, _ := externalRuntime(t, true)
			details := &commonpb.Payloads{Payloads: []*commonpb.Payload{{Data: []byte("requested details")}}}
			s.values.externalRequests[b.cancel] = &workflowservice.RequestCancelActivityExecutionRequest{Namespace: "namespace", ActivityId: "activity-id", RunId: "actual-activity-run", Identity: "external-cancel-controller"}
			s.values.externalRequests[b.answer] = &workflowservice.RespondActivityTaskCanceledByIdRequest{Namespace: "namespace", ActivityId: "activity-id", RunId: "actual-activity-run", Identity: "external-answer-controller", Details: details}
			s.values.externalRequests[b.settlement] = &workflowservice.DescribeActivityExecutionRequest{Namespace: "namespace", ActivityId: "activity-id", RunId: "actual-activity-run", IncludeOutcome: true}
			response := &workflowservice.DescribeActivityExecutionResponse{RunId: "actual-activity-run", Info: &activitypb.ActivityExecutionInfo{RunId: "actual-activity-run", ActivityId: "activity-id", Status: enumspb.ACTIVITY_EXECUTION_STATUS_CANCELED, CloseTime: timestamppb.Now()}, Outcome: &activitypb.ActivityExecutionOutcome{Value: &activitypb.ActivityExecutionOutcome_Failure{Failure: &failurepb.Failure{FailureInfo: &failurepb.Failure_CanceledFailureInfo{CanceledFailureInfo: &failurepb.CanceledFailureInfo{Identity: "external-cancel-controller", Details: proto.CloneOf(details)}}}}}}
			test.mutate(response)
			task := scheduledNode{activation: &scheduledActivation{values: values}, index: values.graph.index[b.settlement.source.InstructionId]}
			_, err := s.recorder.publish(t.Context(), []*testpilotspb.RunEvent{{Kind: testpilotspb.RUN_EVENT_KIND_INSTRUCTION_STARTED, SourceId: s.nodeSource(task) + ".started", CausalSourceIds: []string{"opened"}, Coordinates: eventCoordinates(s.coordinate(task))}}, nil)
			require.NoError(t, err)
			_, err = s.publishCompletion(t.Context(), schedulerCompletion{node: &task, result: contract.EffectResult{Outcome: &testpilotspb.InstructionOutcome{Status: testpilotspb.INSTRUCTION_OUTCOME_STATUS_SUCCEEDED}, Response: response}})
			if test.name == "valid" {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
				for _, event := range s.recorder.run.Events {
					require.NotEqual(t, s.nodeSource(task)+".completed", event.SourceId)
				}
			}
		})
	}
}

func TestActivityExternalCarrierMustMatchAnEarlierPendingPublication(t *testing.T) {
	for _, run := range []string{"actual-activity-run", "foreign-activity-run", ""} {
		t.Run(run, func(t *testing.T) {
			s, b, values, _ := externalRuntime(t, false)
			delete(s.values.slots, b.runSlot)
			task := scheduledNode{activation: &scheduledActivation{values: values}, index: values.graph.index[b.carrier.source.InstructionId]}
			_, err := s.recorder.publish(t.Context(), []*testpilotspb.RunEvent{{Kind: testpilotspb.RUN_EVENT_KIND_INSTRUCTION_STARTED, SourceId: s.nodeSource(task) + ".started", CausalSourceIds: []string{"opened"}, Coordinates: eventCoordinates(s.coordinate(task))}}, nil)
			require.NoError(t, err)
			_, err = s.publishCompletion(t.Context(), schedulerCompletion{node: &task, result: contract.EffectResult{Outcome: &testpilotspb.InstructionOutcome{Status: testpilotspb.INSTRUCTION_OUTCOME_STATUS_SUCCEEDED}, Response: &workflowservice.StartActivityExecutionResponse{RunId: run}}})
			if run == "actual-activity-run" {
				require.NoError(t, err)
				require.Equal(t, run, s.values.slots[b.runSlot].GetTextValue())
			} else {
				require.Error(t, err)
				require.Nil(t, s.values.slots[b.runSlot])
			}
		})
	}
}

func TestActivityExternalScheduledCompletionDoesNotClaimAWorker(t *testing.T) {
	c, catalog, policy := externalFixture(t, false)
	emptyWorker := proto.CloneOf(c.Program.Entrypoints[1])
	emptyWorker.Instructions = nil
	d := c.Program.ActivityExternalSettlements[0]
	d.ActivityEntrypointId = ""
	d.PendingSlotId = ""
	d.Held = nil
	controller := c.Program.Entrypoints[0]
	answer, settled := controller.Instructions[3], controller.Instructions[4]
	answer.Instruction.GetInvokeRpc().Method = activityService + "RespondActivityTaskCompletedById"
	answer.Guard = succeeded("controller", "call")
	controller.Instructions = []*testpilotspb.InstructionNode{controller.Instructions[0], answer, settled}
	c.Program.Entrypoints = c.Program.Entrypoints[:1]
	c.Program.Slots = c.Program.Slots[:1]
	c.Program.Evidence = c.Program.Evidence[1:]
	policy.Roles[0].Methods = append(policy.Roles[0].Methods, activityService+"RespondActivityTaskCompletedById")
	p, err := Prepare(c, catalog, policy)
	require.NoError(t, err)
	_, carried := p.ReservationCarrier("controller", "call")
	require.False(t, carried)
	c.Program.Entrypoints = append(c.Program.Entrypoints, emptyWorker)
	_, err = Prepare(c, catalog, policy)
	require.ErrorContains(t, err, "controller-only")
	c.Program.Entrypoints = c.Program.Entrypoints[:1]
	c.Program.ActivityExternalSettlements = nil
	_, err = Prepare(c, catalog, policy)
	require.ErrorContains(t, err, "external settlement basis")
}

func TestActivityExternalRejectsAnImpostorRequestDescriptor(t *testing.T) {
	c, _, policy := externalFixture(t, false)
	descriptors := testsupport.DescriptorClosure(workflowservice.File_temporal_api_workflowservice_v1_service_proto, testpilotspb.File_temporal_server_api_testpilot_v1_run_proto)
	changed := false
	for _, file := range descriptors.File {
		for _, message := range file.MessageType {
			if message.GetName() != "RespondActivityTaskFailedByIdRequest" {
				continue
			}
			for _, field := range message.Field {
				if field.GetName() == "run_id" {
					field.Number = proto.Int32(99)
					changed = true
				}
			}
		}
	}
	require.True(t, changed)
	catalog, err := ir.NewCatalog(descriptors)
	require.NoError(t, err)
	policy.CatalogIdentity = catalog.Identity()
	_, err = Prepare(c, catalog, policy)
	require.Error(t, err)
}

func TestActivityExternalCleanupRemainsAvailableWithoutPublication(t *testing.T) {
	for _, settled := range []bool{false, true} {
		t.Run(map[bool]string{false: "refusal cleans activity", true: "terminal settlement skips terminate"}[settled], func(t *testing.T) {
			s, b, values, session := externalRuntime(t, false)
			delete(s.values.slots, b.source.PendingSlotId)
			status := func(n *node) {
				values.latest[n.source.InstructionId] = &valueBatch{fields: map[testpilotspb.InstructionOutcomeField]*testpilotspb.Value{testpilotspb.INSTRUCTION_OUTCOME_FIELD_STATUS: ir.EnumValue(testpilotspb.INSTRUCTION_OUTCOME_STATUS_SUCCEEDED.Descriptor(), testpilotspb.INSTRUCTION_OUTCOME_STATUS_SUCCEEDED.Number())}}
			}
			status(b.carrier)
			if settled {
				status(b.settlement)
			}
			session.OnInvokeRPC = func(ctx context.Context, _ contract.Coordinate, _ string, _ protoreflect.MethodDescriptor, request proto.Message) (contract.EffectHandle, error) {
				_, bounded := ctx.Deadline()
				require.True(t, bounded)
				require.Equal(t, "actual-activity-run", externalText(request, "run_id"))
				return testsupport.Completed(contract.EffectResult{Outcome: &testpilotspb.InstructionOutcome{Status: testpilotspb.INSTRUCTION_OUTCOME_STATUS_SUCCEEDED}, Response: &workflowservice.TerminateActivityExecutionResponse{}}), nil
			}
			bounded, cancel := context.WithTimeout(t.Context(), time.Second)
			defer cancel()
			require.NoError(t, s.executeCleanup(bounded))
			want := 1
			if settled {
				want = 0
			}
			require.Equal(t, want, session.Calls("InvokeRPC"))
			require.NoError(t, session.Close(bounded))
			require.Equal(t, 1, session.Calls("Close"))
		})
	}
}
