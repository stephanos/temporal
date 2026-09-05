package worker

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.temporal.io/api/workflowservice/v1"
	umpirespb "go.temporal.io/server/api/umpire/v1"
	"go.temporal.io/server/tools/umpire/internal/execution"
	"go.temporal.io/server/tools/umpire/internal/ir"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
)

func preparedRuntimeFixture(t *testing.T, responseKind umpirespb.NexusResponseKind, modify ...func(*umpirespb.Program)) *execution.PreparedProgram {
	t.Helper()
	file := workflowservice.File_temporal_api_workflowservice_v1_service_proto
	catalog, err := ir.NewCatalog(descriptorClosure(file))
	require.NoError(t, err)
	limits := &umpirespb.ProgramLimits{MaxEntrypoints: 8, MaxNodes: 32, MaxEdges: 64, MaxActivations: 16, MaxAttempts: 16, MaxRunEvents: 16, MaxExpressionDepth: 16, MaxPathFanout: 32, MaxRequestBytes: 64 << 10, MaxResponseBytes: 64 << 10, MaxTotalDurationMilliseconds: 30000, MaxCleanupDurationMilliseconds: 5000}
	method := "/temporal.api.workflowservice.v1.WorkflowService/StartWorkflowExecution"
	policy := execution.Policy{
		Identity: "profile", CatalogIdentity: catalog.Identity(),
		Roles: []execution.RolePolicy{
			{ID: "endpoint", Kind: umpirespb.SYMBOLIC_ROLE_KIND_ENDPOINT, Methods: []string{method}, ReservationCarriers: []execution.ReservationCarrierPolicy{{Method: method, Shapes: []execution.ReservationCarrierShape{{Context: umpirespb.ENTRYPOINT_CONTEXT_WORKFLOW, MaximumCount: 8}, {Context: umpirespb.ENTRYPOINT_CONTEXT_NEXUS_HANDLER, MaximumCount: 8}}}}},
			{ID: "worker", Kind: umpirespb.SYMBOLIC_ROLE_KIND_WORKER},
			{ID: "queue", Kind: umpirespb.SYMBOLIC_ROLE_KIND_TASK_QUEUE},
		},
		Capabilities: []execution.Opcode{execution.InvokeRPC, execution.StartNexusOperation, execution.Await, execution.Finish, execution.RespondNexus}, Limits: proto.CloneOf(limits),
	}
	status := runtimeStatusSchema()
	controller := &umpirespb.InstructionNode{
		InstructionId: "call", Instruction: &umpirespb.Instruction{Instruction: &umpirespb.Instruction_InvokeRpc{InvokeRpc: &umpirespb.InvokeRPC{EndpointRoleId: "endpoint", Method: method}}},
		Outcome: proto.CloneOf(status), Bounds: runtimeBounds(), ActivationReservations: []*umpirespb.ActivationReservation{{EntrypointId: "workflow", Count: 1}, {EntrypointId: "handler", Count: 1}},
	}
	start := &umpirespb.InstructionNode{
		InstructionId: "start", Instruction: &umpirespb.Instruction{Instruction: &umpirespb.Instruction_StartNexusOperation{StartNexusOperation: &umpirespb.StartNexusOperation{EndpointRoleId: "endpoint", Service: "service", Operation: "operation", Input: runtimeText("request")}}},
		Outcome: proto.CloneOf(status), Bounds: runtimeBounds(),
	}
	await := &umpirespb.InstructionNode{
		InstructionId: "await", Dependencies: []*umpirespb.InstructionReference{{EntrypointId: "workflow", InstructionId: "start"}},
		Instruction: &umpirespb.Instruction{Instruction: &umpirespb.Instruction_AwaitOutcome{AwaitOutcome: &umpirespb.Await{Instruction: &umpirespb.InstructionReference{EntrypointId: "workflow", InstructionId: "start"}}}},
		Outcome:     runtimeValueOutcomeSchema(), Bounds: runtimeBounds(),
	}
	finish := &umpirespb.InstructionNode{
		InstructionId: "finish", Dependencies: []*umpirespb.InstructionReference{{EntrypointId: "workflow", InstructionId: "await"}},
		Guard:       runtimeSucceeded("workflow", "await"),
		Instruction: &umpirespb.Instruction{Instruction: &umpirespb.Instruction_Finish{Finish: &umpirespb.Finish{Result: &umpirespb.ValueExpression{Expression: &umpirespb.ValueExpression_Outcome{Outcome: &umpirespb.InstructionOutcomeReference{Instruction: &umpirespb.InstructionReference{EntrypointId: "workflow", InstructionId: "await"}, Field: umpirespb.INSTRUCTION_OUTCOME_FIELD_VALUE}}}}}},
		Outcome:     proto.CloneOf(status), Bounds: runtimeBounds(),
	}
	respond := &umpirespb.InstructionNode{
		InstructionId: "respond", Instruction: &umpirespb.Instruction{Instruction: &umpirespb.Instruction_RespondNexus{RespondNexus: &umpirespb.RespondNexus{Kind: responseKind, Result: runtimeText("accepted")}}},
		Outcome: proto.CloneOf(status), Bounds: runtimeBounds(),
	}
	program := &umpirespb.Program{
		ProgramId: "program", Roles: []*umpirespb.ProgramRole{{RoleId: "endpoint", Kind: umpirespb.SYMBOLIC_ROLE_KIND_ENDPOINT}, {RoleId: "worker", Kind: umpirespb.SYMBOLIC_ROLE_KIND_WORKER}, {RoleId: "queue", Kind: umpirespb.SYMBOLIC_ROLE_KIND_TASK_QUEUE}},
		Entrypoints: []*umpirespb.Entrypoint{
			{EntrypointId: "controller", Context: umpirespb.ENTRYPOINT_CONTEXT_CONTROLLER, Activation: &umpirespb.ActivationBinding{Binding: &umpirespb.ActivationBinding_Controller{Controller: &umpirespb.ControllerActivation{}}}, Nodes: []*umpirespb.InstructionNode{controller}},
			{EntrypointId: "workflow", Context: umpirespb.ENTRYPOINT_CONTEXT_WORKFLOW, Activation: &umpirespb.ActivationBinding{Binding: &umpirespb.ActivationBinding_Workflow{Workflow: &umpirespb.WorkflowActivation{WorkflowType: "workflow-type", WorkerRoleId: "worker", TaskQueueRoleId: "queue"}}}, Nodes: []*umpirespb.InstructionNode{start, await, finish}},
			{EntrypointId: "handler", Context: umpirespb.ENTRYPOINT_CONTEXT_NEXUS_HANDLER, Activation: &umpirespb.ActivationBinding{Binding: &umpirespb.ActivationBinding_NexusHandler{NexusHandler: &umpirespb.NexusHandlerActivation{Service: "service", Operation: "operation", WorkerRoleId: "worker", TaskQueueRoleId: "queue"}}}, Nodes: []*umpirespb.InstructionNode{respond}},
		},
		Cleanup: &umpirespb.CleanupGraph{EntrypointId: "cleanup", Context: umpirespb.ENTRYPOINT_CONTEXT_CONTROLLER}, Limits: limits,
	}
	if responseKind == umpirespb.NEXUS_RESPONSE_KIND_ASYNCHRONOUS {
		program.Slots = []*umpirespb.SlotSchema{{SlotId: "capability", Kind: umpirespb.SLOT_KIND_OPAQUE_CAPABILITY, Type: &umpirespb.ValueType{Shape: &umpirespb.ValueType_Singular{Singular: &umpirespb.SingularType{Type: &umpirespb.SingularType_OpaqueCapability{OpaqueCapability: &umpirespb.OpaqueCapabilityType{}}}}}}}
		respond.Instruction.GetRespondNexus().CapabilitySlotId = "capability"
	}
	for _, apply := range modify {
		apply(program)
	}
	prepared, err := execution.Prepare(&umpirespb.Case{Version: &umpirespb.FormatVersion{Major: 1}, CaseId: "case", Program: program, Contract: &umpirespb.Contract{ContractId: "contract"}}, catalog, policy)
	require.NoError(t, err)
	return prepared
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

func runtimeBounds() *umpirespb.InstructionBounds {
	return &umpirespb.InstructionBounds{TimeoutMilliseconds: 1000, MaxAttempts: 1, MaxEmittedEvents: 4, MaxResponseBytes: 64 << 10}
}

func runtimeText(value string) *umpirespb.ValueExpression {
	return &umpirespb.ValueExpression{Expression: &umpirespb.ValueExpression_Literal{Literal: &umpirespb.Value{Value: &umpirespb.Value_Text{Text: value}}}}
}

func runtimeSucceeded(entrypoint, instruction string) *umpirespb.ValueExpression {
	return &umpirespb.ValueExpression{Expression: &umpirespb.ValueExpression_Equals{Equals: &umpirespb.EqualsExpression{
		Left:  &umpirespb.ValueExpression{Expression: &umpirespb.ValueExpression_Outcome{Outcome: &umpirespb.InstructionOutcomeReference{Instruction: &umpirespb.InstructionReference{EntrypointId: entrypoint, InstructionId: instruction}, Field: umpirespb.INSTRUCTION_OUTCOME_FIELD_STATUS}}},
		Right: &umpirespb.ValueExpression{Expression: &umpirespb.ValueExpression_Literal{Literal: &umpirespb.Value{Value: &umpirespb.Value_EnumValue{EnumValue: &umpirespb.EnumValue{Number: int32(umpirespb.INSTRUCTION_OUTCOME_STATUS_SUCCEEDED)}}}}},
	}}}
}

func runtimeStatusSchema() *umpirespb.InstructionOutcomeSchema {
	return &umpirespb.InstructionOutcomeSchema{Fields: []*umpirespb.OutcomeFieldSchema{{Field: umpirespb.INSTRUCTION_OUTCOME_FIELD_STATUS, Type: runtimeStatusType()}}}
}

func runtimeValueOutcomeSchema() *umpirespb.InstructionOutcomeSchema {
	return &umpirespb.InstructionOutcomeSchema{Fields: []*umpirespb.OutcomeFieldSchema{{Field: umpirespb.INSTRUCTION_OUTCOME_FIELD_STATUS, Type: runtimeStatusType()}, {Field: umpirespb.INSTRUCTION_OUTCOME_FIELD_VALUE, Type: runtimeTextType()}}}
}

func runtimeStatusType() *umpirespb.ValueType {
	return &umpirespb.ValueType{Shape: &umpirespb.ValueType_Singular{Singular: &umpirespb.SingularType{Type: &umpirespb.SingularType_Enumeration{Enumeration: &umpirespb.NamedType{ProtobufType: "temporal.server.api.umpire.v1.InstructionOutcomeStatus"}}}}}
}

func runtimeTextType() *umpirespb.ValueType {
	return &umpirespb.ValueType{Shape: &umpirespb.ValueType_Singular{Singular: &umpirespb.SingularType{Type: &umpirespb.SingularType_Scalar{Scalar: &umpirespb.ScalarType{Kind: umpirespb.SCALAR_KIND_TEXT}}}}}
}
