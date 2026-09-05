package umpire

import (
	"testing"

	"github.com/stretchr/testify/require"
	umpirespb "go.temporal.io/server/api/umpire/v1"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"
)

func publicCarrierFixture(t *testing.T) (*umpirespb.Case, ProfileSpec) {
	t.Helper()
	source, profile := preparationFixture(t)
	catalog, err := NewCatalog(&descriptorpb.FileDescriptorSet{File: []*descriptorpb.FileDescriptorProto{{Name: proto.String("carrier.proto"), Package: proto.String("example"), Syntax: proto.String("proto3"), MessageType: []*descriptorpb.DescriptorProto{{Name: proto.String("Payload")}}, Service: []*descriptorpb.ServiceDescriptorProto{{Name: proto.String("Service"), Method: []*descriptorpb.MethodDescriptorProto{{Name: proto.String("Call"), InputType: proto.String(".example.Payload"), OutputType: proto.String(".example.Payload")}}}}}}})
	require.NoError(t, err)
	profile.Catalog = catalog
	profile.Roles = []RolePolicy{
		{ID: "endpoint", Kind: umpirespb.SYMBOLIC_ROLE_KIND_ENDPOINT, Methods: []string{"/example.Service/Call"}, ReservationCarriers: []ReservationCarrierPolicy{{Method: "/example.Service/Call", Shapes: []ReservationCarrierShape{{Context: umpirespb.ENTRYPOINT_CONTEXT_WORKFLOW, MaximumCount: 1}, {Context: umpirespb.ENTRYPOINT_CONTEXT_NEXUS_HANDLER, MaximumCount: 1}}}}},
		{ID: "worker", Kind: umpirespb.SYMBOLIC_ROLE_KIND_WORKER},
		{ID: "queue", Kind: umpirespb.SYMBOLIC_ROLE_KIND_TASK_QUEUE},
	}
	profile.Capabilities = []Capability{InvokeRPC, StartNexusOperation, RespondNexus}
	source.Program.Roles = []*umpirespb.ProgramRole{{RoleId: "endpoint", Kind: umpirespb.SYMBOLIC_ROLE_KIND_ENDPOINT}, {RoleId: "worker", Kind: umpirespb.SYMBOLIC_ROLE_KIND_WORKER}, {RoleId: "queue", Kind: umpirespb.SYMBOLIC_ROLE_KIND_TASK_QUEUE}}
	status := &umpirespb.InstructionOutcomeSchema{Fields: []*umpirespb.OutcomeFieldSchema{{Field: umpirespb.INSTRUCTION_OUTCOME_FIELD_STATUS, Type: &umpirespb.ValueType{Shape: &umpirespb.ValueType_Singular{Singular: &umpirespb.SingularType{Type: &umpirespb.SingularType_Enumeration{Enumeration: &umpirespb.NamedType{ProtobufType: "temporal.server.api.umpire.v1.InstructionOutcomeStatus"}}}}}}}}
	bounds := &umpirespb.InstructionBounds{TimeoutMilliseconds: 1000, MaxAttempts: 1, MaxEmittedEvents: 1, MaxResponseBytes: 4096}
	call := &umpirespb.InstructionNode{InstructionId: "call", Instruction: &umpirespb.Instruction{Instruction: &umpirespb.Instruction_InvokeRpc{InvokeRpc: &umpirespb.InvokeRPC{EndpointRoleId: "endpoint", Method: "/example.Service/Call"}}}, Outcome: proto.CloneOf(status), Bounds: proto.CloneOf(bounds), ActivationReservations: []*umpirespb.ActivationReservation{{EntrypointId: "workflow", Count: 1}, {EntrypointId: "handler", Count: 1}}}
	source.Program.Entrypoints[0].Nodes = []*umpirespb.InstructionNode{call}
	value := &umpirespb.ValueExpression{Expression: &umpirespb.ValueExpression_Literal{Literal: &umpirespb.Value{Value: &umpirespb.Value_Text{Text: "value"}}}}
	start := &umpirespb.InstructionNode{InstructionId: "start", Instruction: &umpirespb.Instruction{Instruction: &umpirespb.Instruction_StartNexusOperation{StartNexusOperation: &umpirespb.StartNexusOperation{EndpointRoleId: "endpoint", Service: "service", Operation: "operation", Input: value}}}, Outcome: proto.CloneOf(status), Bounds: proto.CloneOf(bounds)}
	respond := &umpirespb.InstructionNode{InstructionId: "respond", Instruction: &umpirespb.Instruction{Instruction: &umpirespb.Instruction_RespondNexus{RespondNexus: &umpirespb.RespondNexus{Kind: umpirespb.NEXUS_RESPONSE_KIND_SYNCHRONOUS, Result: value}}}, Outcome: proto.CloneOf(status), Bounds: proto.CloneOf(bounds)}
	source.Program.Entrypoints = append(source.Program.Entrypoints,
		&umpirespb.Entrypoint{EntrypointId: "workflow", Context: umpirespb.ENTRYPOINT_CONTEXT_WORKFLOW, Activation: &umpirespb.ActivationBinding{Binding: &umpirespb.ActivationBinding_Workflow{Workflow: &umpirespb.WorkflowActivation{WorkflowType: "flow", WorkerRoleId: "worker", TaskQueueRoleId: "queue"}}}, Nodes: []*umpirespb.InstructionNode{start}},
		&umpirespb.Entrypoint{EntrypointId: "handler", Context: umpirespb.ENTRYPOINT_CONTEXT_NEXUS_HANDLER, Activation: &umpirespb.ActivationBinding{Binding: &umpirespb.ActivationBinding_NexusHandler{NexusHandler: &umpirespb.NexusHandlerActivation{Service: "service", Operation: "operation", WorkerRoleId: "worker", TaskQueueRoleId: "queue"}}}, Nodes: []*umpirespb.InstructionNode{respond}},
	)
	return source, profile
}

func TestPrepareCaseFreezesAndExposesReservationCarrierTopology(t *testing.T) {
	source, profile := publicCarrierFixture(t)
	snapshot := profile.Snapshot()
	snapshot.Roles[0].ReservationCarriers[0].Shapes[0].MaximumCount = 0
	require.Equal(t, int64(1), profile.Roles[0].ReservationCarriers[0].Shapes[0].MaximumCount)

	prepared, err := PrepareCase(source, profile)
	require.NoError(t, err)
	program := PreparedProgram{program: prepared.program}
	want := ReservationCarrierPlan{
		EndpointRoleID: "endpoint",
		Method:         "/example.Service/Call",
		Reservations: []ReservationTopology{
			{EntrypointID: "workflow", Context: umpirespb.ENTRYPOINT_CONTEXT_WORKFLOW, Count: 1},
			{EntrypointID: "handler", Context: umpirespb.ENTRYPOINT_CONTEXT_NEXUS_HANDLER, Count: 1},
		},
		Routes: []ReservationRoute{{WorkflowEntrypointID: "workflow", SourceInstructionID: "start", HandlerEntrypointID: "handler"}},
	}
	plan, ok := program.ReservationCarrier("controller", "call")
	require.True(t, ok)
	require.Equal(t, want, plan)

	profile.Roles[0].ReservationCarriers[0].Method = "/example.Service/Changed"
	profile.Roles[0].ReservationCarriers[0].Shapes[0].MaximumCount = 0
	source.Program.Entrypoints[1].Nodes[0].Instruction.GetStartNexusOperation().Operation = "changed"
	plan.Reservations[0].Count = 99
	plan.Routes[0].HandlerOrdinal = 99
	again, ok := program.ReservationCarrier("controller", "call")
	require.True(t, ok)
	require.Equal(t, want, again)
	require.Equal(t, HostIdentity{Profile: "host", Catalog: profile.Catalog.Identity()}, prepared.Identity())
}
