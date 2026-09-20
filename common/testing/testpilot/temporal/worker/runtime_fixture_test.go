package worker

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	commandpb "go.temporal.io/api/command/v1"
	commonpb "go.temporal.io/api/common/v1"
	enumspb "go.temporal.io/api/enums/v1"
	nexuspb "go.temporal.io/api/nexus/v1"
	"go.temporal.io/api/workflowservice/v1"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/testpilot"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/known/anypb"
)

func preparedRuntimeFixture(t *testing.T, responseKind replyKind, modify ...func(*testpilotspb.Program)) testpilot.PreparedProgram {
	return preparedRuntimeFixtureWithProfile(t, responseKind, nil, modify...)
}

func preparedRuntimeFixtureForNamespace(t *testing.T, namespace string, responseKind replyKind, modify ...func(*testpilotspb.Program)) testpilot.PreparedProgram {
	return preparedRuntimeFixtureWithProfile(t, responseKind, func(profile *testpilot.ProfileSpec) {
		for index := range profile.EnvironmentBindings {
			if profile.EnvironmentBindings[index].ID == "namespace" {
				profile.EnvironmentBindings[index].Value = namespace
			}
		}
	}, modify...)
}

func preparedRuntimeFixtureWithProfile(t *testing.T, responseKind replyKind, modifyProfile func(*testpilot.ProfileSpec), modify ...func(*testpilotspb.Program)) testpilot.PreparedProgram {
	t.Helper()
	return capturePreparedProgram(t, preparedRuntimeCase(t, responseKind, modifyProfile, modify...))
}

func preparedRuntimeCase(t *testing.T, responseKind replyKind, modifyProfile func(*testpilot.ProfileSpec), modify ...func(*testpilotspb.Program)) *testpilot.PreparedCase {
	t.Helper()
	file := workflowservice.File_temporal_api_workflowservice_v1_service_proto
	catalog, err := testpilot.NewCatalog(descriptorClosure(file))
	require.NoError(t, err)
	limits := &testpilotspb.ProgramLimits{MaxEntrypoints: 8, MaxNodes: 32, MaxEdges: 64, MaxActivations: 16, MaxAttempts: 16, MaxRunEvents: 16, MaxExpressionDepth: 16, MaxPathFanout: 32, MaxRequestBytes: 64 << 10, MaxResponseBytes: 64 << 10, MaxTotalDurationMilliseconds: 30000, MaxCleanupDurationMilliseconds: 5000, MaxInstructionEmittedEvents: 4, MaxInstructionResponseBytes: 64 << 10}
	method := "/temporal.api.workflowservice.v1.WorkflowService/StartWorkflowExecution"
	contractLimits := &testpilotspb.ContractLimits{MaxRules: 8, MaxStates: 16, MaxTransitions: 16, MaxExpressionDepth: 16, MaxWorkPerEvent: 100000, MaxTotalWork: 1000000000, MaxCaptures: 8, MaxCaptureBytes: 65536}
	profile := testpilot.ProfileSpec{
		Identity: "profile", Catalog: catalog,
		Roles: []testpilot.RolePolicy{
			{ID: "endpoint", Kind: testpilotspb.ROLE_KIND_ENDPOINT, Methods: []string{method}, ReservationCarriers: []testpilot.ReservationCarrierPolicy{{Method: method, Shapes: []testpilot.ReservationCarrierShape{{Kind: testpilot.WorkflowEntrypoint, MaximumCount: 8}, {Kind: testpilot.NexusHandlerEntrypoint, MaximumCount: 8}}}}},
			{ID: "worker", Kind: testpilotspb.ROLE_KIND_WORKER},
			{ID: "queue", Kind: testpilotspb.ROLE_KIND_TASK_QUEUE},
			{ID: "nexus-endpoint", Kind: testpilotspb.ROLE_KIND_ENDPOINT},
		},
		Opcodes:      []testpilot.Opcode{testpilot.InvokeRPC, testpilot.WorkflowCommand, testpilot.Await, testpilot.Finish, testpilot.NexusHandlerReply},
		CommandTypes: CommandTypes(),
		EnvironmentBindings: []testpilot.EnvironmentBinding{
			{ID: "namespace", Value: "namespace"}, {ID: "task-queue", Value: "task-queue"}, {ID: "nexus-endpoint", Value: "endpoint"},
		},
		ProgramLimits: proto.CloneOf(limits), ContractLimits: contractLimits,
	}
	controller := &testpilotspb.InstructionNode{
		InstructionId: "call", Instruction: &testpilotspb.Instruction{Instruction: &testpilotspb.Instruction_InvokeRpc{InvokeRpc: &testpilotspb.InvokeRpc{EndpointRoleId: "endpoint", Method: method, RequestAssignments: []*testpilotspb.RequestAssignment{
			{Target: runtimeField("namespace"), Value: runtimeEnvironment("namespace")},
			{Target: runtimeField("task_queue", "name"), Value: runtimeEnvironment("task-queue")},
		}}}},
		Limits: runtimeBounds(),
	}
	start := &testpilotspb.InstructionNode{
		InstructionId: "start", Instruction: &testpilotspb.Instruction{Instruction: &testpilotspb.Instruction_WorkflowCommand{WorkflowCommand: &testpilotspb.WorkflowCommand{Command: &commandpb.Command{
			CommandType: enumspb.COMMAND_TYPE_SCHEDULE_NEXUS_OPERATION,
			Attributes: &commandpb.Command_ScheduleNexusOperationCommandAttributes{ScheduleNexusOperationCommandAttributes: &commandpb.ScheduleNexusOperationCommandAttributes{
				Endpoint: "nexus-endpoint", Service: "service", Operation: "operation", Input: runtimePayload("request"),
			}},
		}}}},
		Limits: runtimeBounds(),
	}
	await := &testpilotspb.InstructionNode{
		InstructionId: "await", Guard: &testpilotspb.Expression{Expression: &testpilotspb.Expression_Literal{Literal: &testpilotspb.Value{Value: &testpilotspb.Value_BoolValue{BoolValue: true}}}},
		Instruction: &testpilotspb.Instruction{Instruction: &testpilotspb.Instruction_AwaitInstruction{AwaitInstruction: &testpilotspb.AwaitInstruction{Instruction: &testpilotspb.InstructionReference{EntrypointId: "workflow", InstructionId: "start"}}}},
		Limits:      runtimeBounds(),
	}
	finish := &testpilotspb.InstructionNode{
		InstructionId: "finish",
		Guard:         runtimeSucceeded("workflow", "await"),
		Instruction:   &testpilotspb.Instruction{Instruction: &testpilotspb.Instruction_Finish{Finish: &testpilotspb.Finish{Result: &testpilotspb.Expression{Expression: &testpilotspb.Expression_Reference{Reference: &testpilotspb.Reference{Reference: &testpilotspb.Reference_Outcome{Outcome: &testpilotspb.InstructionOutcomeReference{Instruction: &testpilotspb.InstructionReference{EntrypointId: "workflow", InstructionId: "await"}, Field: testpilotspb.INSTRUCTION_OUTCOME_FIELD_VALUE}}}}}}}},
		Limits:        runtimeBounds(),
	}
	reply := &testpilotspb.NexusHandlerReply{Reply: &testpilotspb.NexusHandlerReply_Response{Response: &nexuspb.StartOperationResponse{
		Variant: &nexuspb.StartOperationResponse_SyncSuccess{SyncSuccess: &nexuspb.StartOperationResponse_Sync{Payload: runtimePayload("accepted")}},
	}}}
	if responseKind == replyAsynchronous {
		reply = &testpilotspb.NexusHandlerReply{HandleSlotId: "capability", Reply: &testpilotspb.NexusHandlerReply_Response{Response: &nexuspb.StartOperationResponse{
			Variant: &nexuspb.StartOperationResponse_AsyncSuccess{AsyncSuccess: &nexuspb.StartOperationResponse_Async{}},
		}}}
	}
	respond := &testpilotspb.InstructionNode{
		InstructionId: "respond", Instruction: &testpilotspb.Instruction{Instruction: &testpilotspb.Instruction_NexusHandlerReply{NexusHandlerReply: reply}},
		Limits: runtimeBounds(),
	}
	program := &testpilotspb.Program{
		ProgramId: "program",
		Roles: []*testpilotspb.Role{
			{RoleId: "endpoint", Kind: testpilotspb.ROLE_KIND_ENDPOINT},
			{RoleId: "worker", Kind: testpilotspb.ROLE_KIND_WORKER, NamespaceBindingId: "namespace"},
			{RoleId: "queue", Kind: testpilotspb.ROLE_KIND_TASK_QUEUE, NamespaceBindingId: "namespace", ResourceBindingId: "task-queue"},
			{RoleId: "nexus-endpoint", Kind: testpilotspb.ROLE_KIND_ENDPOINT, ResourceBindingId: "nexus-endpoint"},
		},
		Entrypoints: []*testpilotspb.Entrypoint{
			{EntrypointId: "controller", Activation: &testpilotspb.Entrypoint_Controller{Controller: &testpilotspb.ControllerActivation{}}, Instructions: []*testpilotspb.InstructionNode{controller}},
			{EntrypointId: "workflow", Activation: &testpilotspb.Entrypoint_Workflow{Workflow: &testpilotspb.WorkflowActivation{WorkflowType: "workflow-type", WorkerRoleId: "worker", TaskQueueRoleId: "queue"}}, Instructions: []*testpilotspb.InstructionNode{start, await, finish}},
			{EntrypointId: "handler", Activation: &testpilotspb.Entrypoint_NexusHandler{NexusHandler: &testpilotspb.NexusHandlerActivation{Service: "service", Operation: "operation", WorkerRoleId: "worker", TaskQueueRoleId: "queue"}}, Instructions: []*testpilotspb.InstructionNode{respond}},
		},
		Cleanup: &testpilotspb.Cleanup{EntrypointId: "cleanup"},
	}
	if responseKind == replyAsynchronous {
		program.Slots = []*testpilotspb.Slot{{SlotId: "capability", Content: &testpilotspb.Slot_OpaqueHandle{OpaqueHandle: &testpilotspb.OpaqueHandleType{}}}}
	}
	for _, apply := range modify {
		apply(program)
	}
	if modifyProfile != nil {
		modifyProfile(&profile)
	}
	contract := &testpilotspb.Contract{
		ContractId: "contract",
		Rules: []*testpilotspb.ContractRule{{
			RuleId:         "complete",
			Kind:           testpilotspb.CONTRACT_RULE_KIND_SAFETY,
			InitialStateId: "open",
			States: []*testpilotspb.ContractState{
				{StateId: "open", Status: testpilotspb.CONTRACT_STATE_STATUS_PENDING},
				{StateId: "closed", Status: testpilotspb.CONTRACT_STATE_STATUS_SATISFIED},
			},
			Transitions: []*testpilotspb.ContractTransition{{
				TransitionId:  "close",
				SourceStateId: "open",
				TargetStateId: "closed",
				EventFilter:   &testpilotspb.RunEventFilter{Kinds: []testpilotspb.RunEventKind{testpilotspb.RUN_EVENT_KIND_RUN_CLOSED}},
				Predicate:     &testpilotspb.Expression{Expression: &testpilotspb.Expression_Literal{Literal: &testpilotspb.Value{Value: &testpilotspb.Value_BoolValue{BoolValue: true}}}},
				SupportKind:   testpilotspb.CONTRACT_SUPPORT_KIND_MATCHING_EVENT,
			}},
		}},
	}
	prepared, err := testpilot.Prepare(&testpilotspb.Case{Version: &testpilotspb.FormatVersion{Major: 1}, CaseId: "case", Program: program, Contract: contract}, profile)
	require.NoError(t, err)
	return prepared
}

func capturePreparedProgram(t *testing.T, prepared *testpilot.PreparedCase) testpilot.PreparedProgram {
	t.Helper()
	driver := &programCaptureDriver{identity: prepared.Identity()}
	_, _, err := prepared.Run(t.Context(), driver)
	require.ErrorIs(t, err, errProgramCaptured)
	return driver.program
}

var errProgramCaptured = errors.New("prepared Program captured")

type programCaptureDriver struct {
	identity testpilot.DriverIdentity
	program  testpilot.PreparedProgram
}

func (d *programCaptureDriver) Identity(context.Context) (testpilot.DriverIdentity, error) {
	return d.identity, nil
}

func (d *programCaptureDriver) Validate(context.Context, testpilot.PreparedProgram) error { return nil }

func (d *programCaptureDriver) Open(_ context.Context, _ string, program testpilot.PreparedProgram) (testpilot.Session, error) {
	d.program = program
	return nil, errProgramCaptured
}

func descriptorClosure(root protoreflect.FileDescriptor) *descriptorpb.FileDescriptorSet {
	seen := make(map[string]struct{})
	result := &descriptorpb.FileDescriptorSet{}
	var add func(protoreflect.FileDescriptor)
	add = func(file protoreflect.FileDescriptor) {
		if _, exists := seen[file.Path()]; exists {
			return
		}
		seen[file.Path()] = struct{}{}
		imports := file.Imports()
		for i := 0; i < imports.Len(); i++ {
			add(imports.Get(i))
		}
		result.File = append(result.File, protodesc.ToFileDescriptorProto(file))
	}
	add(root)
	return result
}

func runtimeBounds() *testpilotspb.InstructionLimits {
	return &testpilotspb.InstructionLimits{Timeout: &testpilotspb.InstructionLimits_TimeoutMilliseconds{TimeoutMilliseconds: 1000}, Attempts: &testpilotspb.InstructionLimits_MaxAttempts{MaxAttempts: 1}}
}

// carriedValue is what an awaited Nexus operation answers with: the handler's payload, carried
// whole rather than converted, so the Await's VALUE is that message.
func carriedValue(t *testing.T, value string) *testpilotspb.Value {
	t.Helper()
	carried, err := anypb.New(runtimePayload(value))
	require.NoError(t, err)
	return &testpilotspb.Value{Value: &testpilotspb.Value_MessageValue{MessageValue: carried}}
}

// carriedText reads the text back out of the payload carriedValue wrapped.
func carriedText(t *testing.T, value *testpilotspb.Value) string {
	t.Helper()
	var payload commonpb.Payload
	require.NoError(t, value.GetMessageValue().UnmarshalTo(&payload))
	text, err := strconv.Unquote(string(payload.GetData()))
	require.NoError(t, err)
	return text
}

// runtimePayload is the carried payload a typed instruction of the fixture holds, encoded as the
// SDK's default data converter encodes a string.
func runtimePayload(value string) *commonpb.Payload {
	return &commonpb.Payload{Metadata: map[string][]byte{"encoding": []byte("json/plain")}, Data: []byte(strconv.Quote(value))}
}

func runtimeText(value string) *testpilotspb.Expression {
	return &testpilotspb.Expression{Expression: &testpilotspb.Expression_Literal{Literal: &testpilotspb.Value{Value: &testpilotspb.Value_TextValue{TextValue: value}}}}
}

func runtimeEnvironment(id string) *testpilotspb.Expression {
	return &testpilotspb.Expression{Expression: &testpilotspb.Expression_Reference{Reference: &testpilotspb.Reference{Reference: &testpilotspb.Reference_EnvironmentBindingId{EnvironmentBindingId: id}}}}
}

func runtimeField(fields ...string) string {
	return strings.Join(fields, ".")
}

func runtimeSucceeded(entrypoint, instruction string) *testpilotspb.Expression {
	return &testpilotspb.Expression{Expression: &testpilotspb.Expression_Compare{Compare: &testpilotspb.CompareExpression{Operator: testpilotspb.COMPARISON_OPERATOR_EQUAL,
		Left:  &testpilotspb.Expression{Expression: &testpilotspb.Expression_Reference{Reference: &testpilotspb.Reference{Reference: &testpilotspb.Reference_Outcome{Outcome: &testpilotspb.InstructionOutcomeReference{Instruction: &testpilotspb.InstructionReference{EntrypointId: entrypoint, InstructionId: instruction}, Field: testpilotspb.INSTRUCTION_OUTCOME_FIELD_STATUS}}}}},
		Right: &testpilotspb.Expression{Expression: &testpilotspb.Expression_Literal{Literal: &testpilotspb.Value{Value: &testpilotspb.Value_EnumValue{EnumValue: &testpilotspb.EnumValue{Name: "INSTRUCTION_OUTCOME_STATUS_SUCCEEDED"}}}}},
	}}}
}

func runtimeStatusType() *testpilotspb.ValueType {
	return &testpilotspb.ValueType{Shape: &testpilotspb.ValueType_Singular{Singular: &testpilotspb.SingularType{Type: &testpilotspb.SingularType_Enumeration{Enumeration: &testpilotspb.NamedType{ProtobufType: "temporal.server.api.testpilot.v1.InstructionOutcomeStatus"}}}}}
}

func runtimeTextType() *testpilotspb.ValueType {
	return &testpilotspb.ValueType{Shape: &testpilotspb.ValueType_Singular{Singular: &testpilotspb.SingularType{Type: &testpilotspb.SingularType_Scalar{Scalar: &testpilotspb.ScalarType{Kind: testpilotspb.SCALAR_KIND_TEXT}}}}}
}
