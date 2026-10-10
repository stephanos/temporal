package facadetest

import (
	"strconv"
	"testing"
	"time"

	celpb "cel.dev/expr"
	"github.com/stretchr/testify/require"
	commandpb "go.temporal.io/api/command/v1"
	commonpb "go.temporal.io/api/common/v1"
	enumspb "go.temporal.io/api/enums/v1"
	nexuspb "go.temporal.io/api/nexus/v1"
	"go.temporal.io/api/workflowservice/v1"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/testpilot"
	"go.temporal.io/server/common/testing/testpilot/casefile"
	cel "go.temporal.io/server/common/testing/testpilot/cel"
	"go.temporal.io/server/common/testing/testpilot/internal/testsupport"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/emptypb"
)

// Reply picks the typed handler reply the runtime fixture's handler entrypoint answers with.
type Reply uint8

const (
	SyncReply Reply = iota
	AsyncReply
)

const startWorkflowMethod = "/temporal.api.workflowservice.v1.WorkflowService/StartWorkflowExecution"

// RuntimeCase prepares the runtime fixture. A controller starts a workflow whose entrypoint
// schedules a Nexus operation, awaits it and finishes with its value; a handler entrypoint answers
// the operation with reply. The Profile admits commandTypes; modifyProfile and modify then adjust
// the Profile and the Program.
func RuntimeCase(t testing.TB, reply Reply, commandTypes []enumspb.CommandType, modifyProfile func(*testpilot.ProfileSpec), modify ...func(*testpilotspb.Program)) *testpilot.PreparedCase {
	t.Helper()
	catalog, err := testpilot.NewCatalog(testsupport.DescriptorClosure(workflowservice.File_temporal_api_workflowservice_v1_service_proto))
	require.NoError(t, err)
	// The byte ceilings sit at 64 KiB so the oversized-outcome tests reject just past them.
	limits := testsupport.ProgramLimits()
	limits.MaxRequestBytes, limits.MaxResponseBytes, limits.MaxInstructionResponseBytes = 64<<10, 64<<10, 64<<10
	profile := testpilot.ProfileSpec{
		Identity: "profile", Catalog: catalog,
		Roles: []testpilot.RolePolicy{
			{ID: "endpoint", Kind: testpilotspb.ROLE_KIND_ENDPOINT, Methods: []string{startWorkflowMethod}, ReservationCarriers: []testpilot.ReservationCarrierPolicy{{Method: startWorkflowMethod, Shapes: []testpilot.ReservationCarrierShape{{Kind: testpilot.WorkflowEntrypoint, MaximumCount: 8}, {Kind: testpilot.NexusHandlerEntrypoint, MaximumCount: 8}}}}},
			{ID: "worker", Kind: testpilotspb.ROLE_KIND_WORKER},
			{ID: "queue", Kind: testpilotspb.ROLE_KIND_TASK_QUEUE},
			{ID: "nexus-endpoint", Kind: testpilotspb.ROLE_KIND_ENDPOINT},
		},
		Opcodes:      []testpilot.Opcode{testpilot.InvokeRPC, testpilot.WorkflowCommand, testpilot.Await, testpilot.Finish, testpilot.NexusHandlerReply},
		CommandTypes: commandTypes,
		EnvironmentBindings: []testpilot.EnvironmentBinding{
			{ID: "namespace", Value: "namespace"}, {ID: "task-queue", Value: "task-queue"}, {ID: "nexus-endpoint", Value: "endpoint"},
		},
		ProgramLimits:  limits,
		ContractLimits: &testpilotspb.ContractLimits{MaxRules: 8, MaxStates: 16, MaxTransitions: 16, MaxExpressionDepth: 16, MaxWorkPerEvent: 100000, MaxTotalWork: 1000000000, MaxCaptures: 8, MaxCaptureBytes: 65536},
	}
	controller := &testpilotspb.InstructionNode{
		InstructionId: "call", Instruction: &testpilotspb.Instruction{Instruction: &testpilotspb.Instruction_InvokeRpc{InvokeRpc: &testpilotspb.InvokeRpc{EndpointRoleId: "endpoint", Method: startWorkflowMethod, RequestAssignments: []*testpilotspb.RequestAssignment{
			{Target: "namespace", Value: environment("namespace")},
			{Target: "task_queue.name", Value: environment("task-queue")},
		}}}},
		Limits: Bounds(),
	}
	start := &testpilotspb.InstructionNode{
		InstructionId: "start", Instruction: &testpilotspb.Instruction{Instruction: &testpilotspb.Instruction_WorkflowCommand{WorkflowCommand: &testpilotspb.WorkflowCommand{Command: &commandpb.Command{
			CommandType: enumspb.COMMAND_TYPE_SCHEDULE_NEXUS_OPERATION,
			Attributes: &commandpb.Command_ScheduleNexusOperationCommandAttributes{ScheduleNexusOperationCommandAttributes: &commandpb.ScheduleNexusOperationCommandAttributes{
				Endpoint: "nexus-endpoint", Service: "service", Operation: "operation", Input: Payload("request"),
			}},
		}}}},
		Limits: Bounds(),
	}
	await := &testpilotspb.InstructionNode{
		InstructionId: "await", Guard: cel.Literal(&celpb.Value{Kind: &celpb.Value_BoolValue{BoolValue: true}}),
		Instruction: &testpilotspb.Instruction{Instruction: &testpilotspb.Instruction_AwaitInstruction{AwaitInstruction: &testpilotspb.AwaitInstruction{Instruction: testsupport.LocalReference(&testpilotspb.InstructionReference{EntrypointId: "workflow", InstructionId: "start"})}}},
		Limits:      Bounds(),
	}
	finish := &testpilotspb.InstructionNode{
		InstructionId: "finish",
		Guard:         succeeded("workflow", "await"),
		Instruction:   &testpilotspb.Instruction{Instruction: &testpilotspb.Instruction_Finish{Finish: &testpilotspb.Finish{Result: cel.Ref(&testpilotspb.Reference{Reference: &testpilotspb.Reference_Outcome{Outcome: &testpilotspb.InstructionOutcomeReference{Instruction: &testpilotspb.InstructionReference{EntrypointId: "workflow", InstructionId: "await"}, Field: testpilotspb.INSTRUCTION_OUTCOME_FIELD_VALUE}}})}}},
		Limits:        Bounds(),
	}
	runtimeReply := &testpilotspb.NexusHandlerReply{Reply: &testpilotspb.NexusHandlerReply_Response{Response: &nexuspb.StartOperationResponse{
		Variant: &nexuspb.StartOperationResponse_SyncSuccess{SyncSuccess: &nexuspb.StartOperationResponse_Sync{Payload: Payload("accepted")}},
	}}}
	if reply == AsyncReply {
		runtimeReply = &testpilotspb.NexusHandlerReply{HandleSlotId: "handle", Reply: &testpilotspb.NexusHandlerReply_Response{Response: &nexuspb.StartOperationResponse{
			Variant: &nexuspb.StartOperationResponse_AsyncSuccess{AsyncSuccess: &nexuspb.StartOperationResponse_Async{}},
		}}}
	}
	respond := &testpilotspb.InstructionNode{
		InstructionId: "respond", Instruction: &testpilotspb.Instruction{Instruction: &testpilotspb.Instruction_NexusHandlerReply{NexusHandlerReply: runtimeReply}},
		Limits: Bounds(),
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
			{EntrypointId: "controller", Activation: &testpilotspb.Entrypoint_Controller{Controller: &emptypb.Empty{}}, Instructions: []*testpilotspb.InstructionNode{controller}},
			{EntrypointId: "workflow", Activation: &testpilotspb.Entrypoint_Workflow{Workflow: &testpilotspb.WorkflowActivation{WorkflowType: "workflow-type", WorkerRoleId: "worker", TaskQueueRoleId: "queue"}}, Instructions: []*testpilotspb.InstructionNode{start, await, finish}},
			{EntrypointId: "handler", Activation: &testpilotspb.Entrypoint_NexusHandler{NexusHandler: &testpilotspb.NexusHandlerActivation{Service: "service", Operation: "operation", WorkerRoleId: "worker", TaskQueueRoleId: "queue"}}, Instructions: []*testpilotspb.InstructionNode{respond}},
		},
		Cleanup: &testpilotspb.Cleanup{EntrypointId: "cleanup"},
	}
	if reply == AsyncReply {
		program.Slots = []*testpilotspb.Slot{{SlotId: "handle", Content: &testpilotspb.Slot_OpaqueHandle{OpaqueHandle: &emptypb.Empty{}}}}
	}
	for _, apply := range modify {
		apply(program)
	}
	if modifyProfile != nil {
		modifyProfile(&profile)
	}
	contract := &testpilotspb.Contract{
		ContractId: "contract",
		Rules:      []*testpilotspb.ContractRule{{RuleId: "complete", InitialStateId: "open", States: []*testpilotspb.ContractState{{StateId: "open", Status: testpilotspb.CONTRACT_STATE_STATUS_PENDING}, {StateId: "closed", Status: testpilotspb.CONTRACT_STATE_STATUS_SATISFIED}}, Transitions: []*testpilotspb.ContractTransition{{TransitionId: "close", SourceStateId: "open", TargetStateId: "closed", EventFilter: &testpilotspb.RunEventFilter{Kinds: []testpilotspb.RunEventKind{testpilotspb.RUN_EVENT_KIND_RUN_CLOSED}}, Predicate: cel.Literal(&celpb.Value{Kind: &celpb.Value_BoolValue{BoolValue: true}}), SupportsEvent: proto.Bool(true)}}}},
	}
	prepared, err := testpilot.Prepare(&testpilotspb.Case{Version: &testpilotspb.FormatVersion{Major: casefile.CurrentMajor}, CaseId: "case", Program: program, Contract: contract}, profile)
	require.NoError(t, err)
	return prepared
}

// Bounds are the instruction limits of every fixture instruction.
func Bounds() *testpilotspb.InstructionLimits {
	return &testpilotspb.InstructionLimits{Timeout: durationpb.New(time.Duration(1000) * time.Millisecond), MaxAttempts: proto.Int64(1)}
}

// Payload is the carried payload a typed instruction of the fixture holds, encoded as the SDK's
// default data converter encodes a string.
func Payload(value string) *commonpb.Payload {
	return &commonpb.Payload{Metadata: map[string][]byte{"encoding": []byte("json/plain")}, Data: []byte(strconv.Quote(value))}
}

// CarriedValue is what an awaited Nexus operation answers with: the handler's payload, carried
// whole rather than converted, so the Await's VALUE is that message.
func CarriedValue(value string) *celpb.Value {
	carried, err := anypb.New(Payload(value))
	if err != nil {
		panic(err)
	}
	return &celpb.Value{Kind: &celpb.Value_ObjectValue{ObjectValue: carried}}
}

// CarriedText reads the text back out of the payload CarriedValue wrapped.
func CarriedText(t testing.TB, value *celpb.Value) string {
	t.Helper()
	var payload commonpb.Payload
	require.NoError(t, value.GetObjectValue().UnmarshalTo(&payload))
	text, err := strconv.Unquote(string(payload.GetData()))
	require.NoError(t, err)
	return text
}

// Text is a text literal.
func Text(value string) *testpilotspb.Expression {
	return cel.Literal(&celpb.Value{Kind: &celpb.Value_StringValue{StringValue: value}})
}

// environment reads the environment binding id.
func environment(id string) *testpilotspb.Expression {
	return cel.Ref(&testpilotspb.Reference{Reference: &testpilotspb.Reference_EnvironmentBindingId{EnvironmentBindingId: id}})
}

// succeeded is true once the instruction's outcome status is SUCCEEDED.
func succeeded(entrypoint, instruction string) *testpilotspb.Expression {
	return cel.Compare("_==_", cel.Ref(&testpilotspb.Reference{Reference: &testpilotspb.Reference_Outcome{Outcome: &testpilotspb.InstructionOutcomeReference{Instruction: &testpilotspb.InstructionReference{EntrypointId: entrypoint, InstructionId: instruction}, Field: testpilotspb.INSTRUCTION_OUTCOME_FIELD_STATUS}}}), cel.Literal(cel.Enum(testpilotspb.INSTRUCTION_OUTCOME_STATUS_SUCCEEDED)))
}
