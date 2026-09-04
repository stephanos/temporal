package execution

import (
	"fmt"
	"math"
	"testing"

	"github.com/stretchr/testify/require"
	umpirespb "go.temporal.io/server/api/umpire/v1"
	"go.temporal.io/server/tools/umpire/internal/ir"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
)

func fixture(t *testing.T) (*umpirespb.Case, *ir.Catalog, Policy) {
	t.Helper()
	catalog, err := ir.NewCatalog(&descriptorpb.FileDescriptorSet{File: []*descriptorpb.FileDescriptorProto{{Name: proto.String("admission.proto"), Package: proto.String("example"), Syntax: proto.String("proto3"), MessageType: []*descriptorpb.DescriptorProto{{Name: proto.String("Payload"), Field: []*descriptorpb.FieldDescriptorProto{{Name: proto.String("text"), Number: proto.Int32(1), Type: descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(), Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum()}, {Name: proto.String("items"), Number: proto.Int32(2), Type: descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(), Label: descriptorpb.FieldDescriptorProto_LABEL_REPEATED.Enum()}}}}, Service: []*descriptorpb.ServiceDescriptorProto{{Name: proto.String("Service"), Method: []*descriptorpb.MethodDescriptorProto{{Name: proto.String("Call"), InputType: proto.String(".example.Payload"), OutputType: proto.String(".example.Payload")}}}}}}})
	require.NoError(t, err)
	limits := &umpirespb.ProgramLimits{MaxEntrypoints: 8, MaxNodes: 32, MaxEdges: 64, MaxActivations: 64, MaxAttempts: 32, MaxRunEvents: 256, MaxExpressionDepth: 16, MaxPathFanout: 128, MaxRequestBytes: 4096, MaxResponseBytes: 4096, MaxTotalDurationMilliseconds: 30000, MaxCleanupDurationMilliseconds: 5000}
	policy := Policy{Identity: "host", CatalogIdentity: catalog.Identity(), Roles: []RolePolicy{{ID: "endpoint", Kind: umpirespb.SYMBOLIC_ROLE_KIND_ENDPOINT, Methods: []string{"/example.Service/Call"}}, {ID: "worker", Kind: umpirespb.SYMBOLIC_ROLE_KIND_WORKER}, {ID: "queue", Kind: umpirespb.SYMBOLIC_ROLE_KIND_TASK_QUEUE}}, Capabilities: []Opcode{InvokeRPC, AwaitSlot, CompleteNexusOperation, StartNexusOperation, Await, Finish, RespondNexus}, Limits: proto.CloneOf(limits)}
	source := &umpirespb.Case{Version: &umpirespb.FormatVersion{Major: 1}, CaseId: "case", Program: &umpirespb.Program{ProgramId: "program", Roles: []*umpirespb.ProgramRole{{RoleId: "endpoint", Kind: umpirespb.SYMBOLIC_ROLE_KIND_ENDPOINT}}, Entrypoints: []*umpirespb.Entrypoint{{EntrypointId: "controller", Context: umpirespb.ENTRYPOINT_CONTEXT_CONTROLLER, Activation: &umpirespb.ActivationBinding{Binding: &umpirespb.ActivationBinding_Controller{Controller: &umpirespb.ControllerActivation{}}}, Nodes: []*umpirespb.InstructionNode{rpcNode("call")}}}, Cleanup: &umpirespb.CleanupGraph{EntrypointId: "cleanup", Context: umpirespb.ENTRYPOINT_CONTEXT_CONTROLLER}, Limits: limits}, Contract: &umpirespb.Contract{ContractId: "contract"}}
	return source, catalog, policy
}
func scalar(kind umpirespb.ScalarKind) *umpirespb.ValueType {
	return &umpirespb.ValueType{Shape: &umpirespb.ValueType_Singular{Singular: &umpirespb.SingularType{Type: &umpirespb.SingularType_Scalar{Scalar: &umpirespb.ScalarType{Kind: kind}}}}}
}
func statusSchema() *umpirespb.InstructionOutcomeSchema {
	return &umpirespb.InstructionOutcomeSchema{Fields: []*umpirespb.OutcomeFieldSchema{{Field: umpirespb.INSTRUCTION_OUTCOME_FIELD_STATUS, Type: &umpirespb.ValueType{Shape: &umpirespb.ValueType_Singular{Singular: &umpirespb.SingularType{Type: &umpirespb.SingularType_Enumeration{Enumeration: &umpirespb.NamedType{ProtobufType: "temporal.server.api.umpire.v1.InstructionOutcomeStatus"}}}}}}}}
}
func rpcNode(id string) *umpirespb.InstructionNode {
	return &umpirespb.InstructionNode{InstructionId: id, Instruction: &umpirespb.Instruction{Instruction: &umpirespb.Instruction_InvokeRpc{InvokeRpc: &umpirespb.InvokeRPC{EndpointRoleId: "endpoint", Method: "/example.Service/Call"}}}, Outcome: statusSchema(), Bounds: &umpirespb.InstructionBounds{TimeoutMilliseconds: 1000, MaxAttempts: 1, MaxEmittedEvents: 8, MaxResponseBytes: 4096}}
}
func field(name string) *umpirespb.FieldPath {
	return &umpirespb.FieldPath{Segments: []*umpirespb.FieldPathSegment{{Field: name}}}
}
func slot(id string) *umpirespb.ValueExpression {
	return &umpirespb.ValueExpression{Expression: &umpirespb.ValueExpression_Slot{Slot: &umpirespb.SlotReference{SlotId: id}}}
}
func present(value *umpirespb.ValueExpression) *umpirespb.ValueExpression {
	return &umpirespb.ValueExpression{Expression: &umpirespb.ValueExpression_Present{Present: &umpirespb.PresentExpression{Operand: value}}}
}
func succeeded(entry, node string) *umpirespb.ValueExpression {
	return &umpirespb.ValueExpression{Expression: &umpirespb.ValueExpression_Equals{Equals: &umpirespb.EqualsExpression{Left: &umpirespb.ValueExpression{Expression: &umpirespb.ValueExpression_Outcome{Outcome: &umpirespb.InstructionOutcomeReference{Instruction: &umpirespb.InstructionReference{EntrypointId: entry, InstructionId: node}, Field: umpirespb.INSTRUCTION_OUTCOME_FIELD_STATUS}}}, Right: &umpirespb.ValueExpression{Expression: &umpirespb.ValueExpression_Literal{Literal: &umpirespb.Value{Value: &umpirespb.Value_EnumValue{EnumValue: &umpirespb.EnumValue{Number: 1}}}}}}}}
}
func addWorker(source *umpirespb.Case) {
	source.Program.Roles = append(source.Program.Roles, &umpirespb.ProgramRole{RoleId: "worker", Kind: umpirespb.SYMBOLIC_ROLE_KIND_WORKER}, &umpirespb.ProgramRole{RoleId: "queue", Kind: umpirespb.SYMBOLIC_ROLE_KIND_TASK_QUEUE})
	source.Program.Entrypoints = append(source.Program.Entrypoints, &umpirespb.Entrypoint{EntrypointId: "workflow", Context: umpirespb.ENTRYPOINT_CONTEXT_WORKFLOW, Activation: &umpirespb.ActivationBinding{Binding: &umpirespb.ActivationBinding_Workflow{Workflow: &umpirespb.WorkflowActivation{WorkflowType: "flow", WorkerRoleId: "worker", TaskQueueRoleId: "queue"}}}})
}
func TestPrepareRejectsStructuralAndPolicyErrors(t *testing.T) {
	for name, mutate := range map[string]func(*umpirespb.Case, *Policy){
		"version":          func(c *umpirespb.Case, _ *Policy) { c.Version.Major = 2 },
		"minor":            func(c *umpirespb.Case, _ *Policy) { c.Version.Minor = 1 },
		"case id":          func(c *umpirespb.Case, _ *Policy) { c.CaseId = " bad" },
		"missing contract": func(c *umpirespb.Case, _ *Policy) { c.Contract = nil },
		"unknown field":    func(c *umpirespb.Case, _ *Policy) { c.Program.ProtoReflect().SetUnknown([]byte{0x80, 0x06, 1}) },
		"duplicate entry": func(c *umpirespb.Case, _ *Policy) {
			c.Program.Entrypoints = append(c.Program.Entrypoints, proto.CloneOf(c.Program.Entrypoints[0]))
		},
		"duplicate node": func(c *umpirespb.Case, _ *Policy) {
			g := c.Program.Entrypoints[0]
			g.Nodes = append(g.Nodes, proto.CloneOf(g.Nodes[0]))
		},
		"cycle": func(c *umpirespb.Case, _ *Policy) {
			c.Program.Entrypoints[0].Nodes[0].Dependencies = []*umpirespb.InstructionReference{{EntrypointId: "controller", InstructionId: "call"}}
		},
		"cross entry": func(c *umpirespb.Case, _ *Policy) {
			c.Program.Entrypoints[0].Nodes[0].Dependencies = []*umpirespb.InstructionReference{{EntrypointId: "cleanup", InstructionId: "call"}}
		},
		"missing dependency": func(c *umpirespb.Case, _ *Policy) {
			c.Program.Entrypoints[0].Nodes[0].Dependencies = []*umpirespb.InstructionReference{{EntrypointId: "controller", InstructionId: "missing"}}
		},
		"context": func(c *umpirespb.Case, _ *Policy) {
			c.Program.Entrypoints[0].Context = umpirespb.ENTRYPOINT_CONTEXT_WORKFLOW
		},
		"binding": func(c *umpirespb.Case, _ *Policy) { c.Program.Entrypoints[0].Activation = nil },
		"role":    func(c *umpirespb.Case, _ *Policy) { c.Program.Roles[0].Kind = umpirespb.SYMBOLIC_ROLE_KIND_WORKER },
		"method": func(c *umpirespb.Case, _ *Policy) {
			c.Program.Entrypoints[0].Nodes[0].Instruction.GetInvokeRpc().Method = "/example.Service/Missing"
		},
		"authorization":    func(_ *umpirespb.Case, p *Policy) { p.Roles[0].Methods = nil },
		"capability":       func(_ *umpirespb.Case, p *Policy) { p.Capabilities = nil },
		"catalog identity": func(_ *umpirespb.Case, p *Policy) { p.CatalogIdentity = "other" },
		"node timeout":     func(c *umpirespb.Case, _ *Policy) { c.Program.Entrypoints[0].Nodes[0].Bounds.TimeoutMilliseconds = 0 },
		"attempt bound":    func(c *umpirespb.Case, _ *Policy) { c.Program.Entrypoints[0].Nodes[0].Bounds.MaxAttempts = 33 },
		"response bound":   func(c *umpirespb.Case, _ *Policy) { c.Program.Entrypoints[0].Nodes[0].Bounds.MaxResponseBytes = 4097 },
		"event bound":      func(c *umpirespb.Case, _ *Policy) { c.Program.Entrypoints[0].Nodes[0].Bounds.MaxEmittedEvents = 257 },
		"rpc raw outcome": func(c *umpirespb.Case, _ *Policy) {
			c.Program.Entrypoints[0].Nodes[0].Outcome.Fields = append(c.Program.Entrypoints[0].Nodes[0].Outcome.Fields, &umpirespb.OutcomeFieldSchema{Field: umpirespb.INSTRUCTION_OUTCOME_FIELD_VALUE, Type: scalar(umpirespb.SCALAR_KIND_TEXT)})
		},
	} {
		t.Run(name, func(t *testing.T) {
			c, catalog, p := fixture(t)
			mutate(c, &p)
			_, err := Prepare(c, catalog, p)
			require.Error(t, err)
		})
	}
	c, catalog, p := fixture(t)
	fields := c.Program.Limits.ProtoReflect().Descriptor().Fields()
	for i := 0; i < fields.Len(); i++ {
		f := fields.Get(i)
		t.Run(string(f.Name()), func(t *testing.T) {
			for _, v := range []int64{0, p.Limits.ProtoReflect().Get(f).Int() + 1} {
				source := proto.CloneOf(c)
				source.Program.Limits.ProtoReflect().Set(f, protoreflect.ValueOfInt64(v))
				_, err := Prepare(source, catalog, p)
				require.Error(t, err)
			}
		})
	}
	_, err := Prepare(nil, catalog, p)
	require.Error(t, err)
	_, err = Prepare(c, nil, p)
	require.Error(t, err)
}

func TestReservationAdmissionBoundsLocalAndGlobalAttempts(t *testing.T) {
	for _, test := range []struct {
		name                                string
		local, global, count, ceiling, want int64
		good                                bool
	}{
		{"local cap", 2, 32, 3, 7, 7, true}, {"global cap", 8, 2, 3, 7, 7, true}, {"ceiling", 2, 32, 3, 6, 0, false}, {"zero", 1, 32, 0, 64, 0, false}, {"negative", 1, 32, -1, 64, 0, false}, {"overflow", 2, 32, math.MaxInt64, 64, 0, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			c, catalog, p := fixture(t)
			addWorker(c)
			node := c.Program.Entrypoints[0].Nodes[0]
			node.Bounds.MaxAttempts = test.local
			c.Program.Limits.MaxAttempts = test.global
			c.Program.Limits.MaxActivations = test.ceiling
			node.ActivationReservations = []*umpirespb.ActivationReservation{{EntrypointId: "workflow", Count: test.count}}
			prepared, err := Prepare(c, catalog, p)
			if test.good {
				require.NoError(t, err)
				require.Equal(t, test.want, prepared.View().MaximumActivations())
			} else {
				require.Error(t, err)
			}
		})
	}
}

func TestPrepareSlotDataflowAndImmutableViews(t *testing.T) {
	c, catalog, p := fixture(t)
	c.Program.Slots = []*umpirespb.SlotSchema{{SlotId: "result", Kind: umpirespb.SLOT_KIND_VALUE, Type: scalar(umpirespb.SCALAR_KIND_TEXT)}}
	c.Program.Observations = []*umpirespb.ObservationSchema{{ObservationId: "text", Type: scalar(umpirespb.SCALAR_KIND_TEXT)}}
	producer := c.Program.Entrypoints[0].Nodes[0]
	producer.Instruction.GetInvokeRpc().ResponseProjections = []*umpirespb.ResponseProjection{{Source: field("text"), Cardinality: umpirespb.PROJECTION_CARDINALITY_ONE, Sinks: []*umpirespb.ProjectionSink{{Sink: &umpirespb.ProjectionSink_SlotId{SlotId: "result"}}, {Sink: &umpirespb.ProjectionSink_ObservationId{ObservationId: "text"}}}}}
	consumer := rpcNode("consume")
	consumer.Dependencies = []*umpirespb.InstructionReference{{EntrypointId: "controller", InstructionId: "call"}}
	consumer.Guard = succeeded("controller", "call")
	consumer.Instruction.GetInvokeRpc().RequestAssignments = []*umpirespb.RequestAssignment{{Target: field("text"), Value: slot("result")}}
	c.Program.Entrypoints[0].Nodes = append(c.Program.Entrypoints[0].Nodes, consumer)
	prepared, err := Prepare(c, catalog, p)
	require.NoError(t, err)
	for name, mutate := range map[string]func(*umpirespb.Case){"unguarded": func(s *umpirespb.Case) { s.Program.Entrypoints[0].Nodes[1].Guard = nil }, "missing dependency": func(s *umpirespb.Case) { s.Program.Entrypoints[0].Nodes[1].Dependencies = nil }, "second writer": func(s *umpirespb.Case) {
		s.Program.Entrypoints[0].Nodes[1].Instruction.GetInvokeRpc().ResponseProjections = proto.CloneOf(producer.Instruction).GetInvokeRpc().ResponseProjections
	}, "assignment overlap": func(s *umpirespb.Case) {
		rpc := s.Program.Entrypoints[0].Nodes[1].Instruction.GetInvokeRpc()
		rpc.RequestAssignments = append(rpc.RequestAssignments, proto.CloneOf(rpc.RequestAssignments[0]))
	}, "crossed cardinality": func(s *umpirespb.Case) {
		s.Program.Entrypoints[0].Nodes[0].Instruction.GetInvokeRpc().ResponseProjections[0].Cardinality = umpirespb.PROJECTION_CARDINALITY_EMIT_EACH
	}, "undeclared outcome": func(s *umpirespb.Case) { s.Program.Entrypoints[0].Nodes[0].Outcome.Fields = nil }} {
		t.Run(name, func(t *testing.T) {
			source := proto.CloneOf(c)
			mutate(source)
			_, err := Prepare(source, catalog, p)
			require.Error(t, err)
		})
	}
	consumer.Guard = present(slot("result"))
	_, err = Prepare(c, catalog, p)
	require.NoError(t, err)
	c.Program.ProgramId = "changed"
	p.Roles[0].Methods[0] = "changed"
	p.Limits.MaxNodes = 1
	snapshot := prepared.Snapshot()
	snapshot.ProgramId = "changed again"
	view := prepared.View()
	view.Limits().MaxNodes = 1
	observations := view.Observations()
	observations[0].ID = "changed"
	require.Equal(t, "program", prepared.Snapshot().ProgramId)
	require.Equal(t, int64(32), view.Limits().MaxNodes)
	require.Equal(t, "text", view.Observations()[0].ID)
}

func TestReservationTargetsAndExactCombinedBound(t *testing.T) {
	for name, mutate := range map[string]func(*umpirespb.Case){
		"missing target": func(c *umpirespb.Case) {
			c.Program.Entrypoints[0].Nodes[0].ActivationReservations[0].EntrypointId = "missing"
		},
		"controller target": func(c *umpirespb.Case) {
			c.Program.Entrypoints[0].Nodes[0].ActivationReservations[0].EntrypointId = "controller"
		},
		"wrong binding": func(c *umpirespb.Case) {
			c.Program.Entrypoints[1].Activation = &umpirespb.ActivationBinding{Binding: &umpirespb.ActivationBinding_Controller{Controller: &umpirespb.ControllerActivation{}}}
		},
		"activity target": func(c *umpirespb.Case) {
			g := c.Program.Entrypoints[1]
			g.Context = umpirespb.ENTRYPOINT_CONTEXT_ACTIVITY
			g.Activation = &umpirespb.ActivationBinding{Binding: &umpirespb.ActivationBinding_Activity{Activity: &umpirespb.ActivityActivation{ActivityType: "activity", WorkerRoleId: "worker", TaskQueueRoleId: "queue"}}}
		},
		"duplicate target": func(c *umpirespb.Case) {
			n := c.Program.Entrypoints[0].Nodes[0]
			n.ActivationReservations = append(n.ActivationReservations, proto.CloneOf(n.ActivationReservations[0]))
		},
		"worker reservation": func(c *umpirespb.Case) {
			n := c.Program.Entrypoints[0].Nodes[0]
			worker := proto.CloneOf(n)
			worker.Instruction = &umpirespb.Instruction{Instruction: &umpirespb.Instruction_Finish{Finish: &umpirespb.Finish{Result: textLiteral("done")}}}
			c.Program.Entrypoints[1].Nodes = []*umpirespb.InstructionNode{worker}
		},
		"cleanup reservation": func(c *umpirespb.Case) {
			c.Program.Cleanup.Nodes = []*umpirespb.InstructionNode{proto.CloneOf(c.Program.Entrypoints[0].Nodes[0])}
		},
		"sum overflow": func(c *umpirespb.Case) {
			other := proto.CloneOf(c.Program.Entrypoints[1])
			other.EntrypointId = "other"
			c.Program.Entrypoints = append(c.Program.Entrypoints, other)
			n := c.Program.Entrypoints[0].Nodes[0]
			n.ActivationReservations = []*umpirespb.ActivationReservation{{EntrypointId: "workflow", Count: math.MaxInt64}, {EntrypointId: "other", Count: math.MaxInt64}}
		},
	} {
		t.Run(name, func(t *testing.T) {
			c, catalog, p := fixture(t)
			addWorker(c)
			c.Program.Entrypoints[0].Nodes[0].ActivationReservations = []*umpirespb.ActivationReservation{{EntrypointId: "workflow", Count: 1}}
			mutate(c)
			_, err := Prepare(c, catalog, p)
			require.Error(t, err)
		})
	}
	c, catalog, p := fixture(t)
	addWorker(c)
	first := c.Program.Entrypoints[0].Nodes[0]
	first.Bounds.MaxAttempts = 2
	first.ActivationReservations = []*umpirespb.ActivationReservation{{EntrypointId: "workflow", Count: 5}}
	second := proto.CloneOf(c.Program.Entrypoints[0])
	second.EntrypointId = "second"
	second.Nodes[0].Bounds.MaxAttempts = 5
	second.Nodes[0].ActivationReservations[0].Count = 3
	c.Program.Entrypoints = append(c.Program.Entrypoints, second)
	c.Program.Limits.MaxAttempts = 4
	c.Program.Limits.MaxActivations = 18
	prepared, err := Prepare(c, catalog, p)
	require.NoError(t, err)
	require.Equal(t, int64(18), prepared.View().MaximumActivations())
	c.Program.Limits.MaxActivations = 17
	_, err = Prepare(c, catalog, p)
	require.Error(t, err)
	first.ActivationReservations = nil
	second.Nodes[0].ActivationReservations = nil
	c.Program.Limits.MaxActivations = 2
	prepared, err = Prepare(c, catalog, p)
	require.NoError(t, err)
	require.Equal(t, int64(2), prepared.View().MaximumActivations())
}
func textLiteral(value string) *umpirespb.ValueExpression {
	return &umpirespb.ValueExpression{Expression: &umpirespb.ValueExpression_Literal{Literal: &umpirespb.Value{Value: &umpirespb.Value_Text{Text: value}}}}
}

func TestInstructionContextMatrix(t *testing.T) {
	for _, context := range []umpirespb.EntrypointContext{umpirespb.ENTRYPOINT_CONTEXT_CONTROLLER, umpirespb.ENTRYPOINT_CONTEXT_WORKFLOW, umpirespb.ENTRYPOINT_CONTEXT_ACTIVITY, umpirespb.ENTRYPOINT_CONTEXT_NEXUS_HANDLER} {
		for _, test := range []struct {
			name        string
			instruction *umpirespb.Instruction
			expected    umpirespb.EntrypointContext
		}{
			{"rpc", rpcNode("call").Instruction, umpirespb.ENTRYPOINT_CONTEXT_CONTROLLER},
			{"await slot", &umpirespb.Instruction{Instruction: &umpirespb.Instruction_AwaitSlot{AwaitSlot: &umpirespb.AwaitSlot{SlotId: "value"}}}, umpirespb.ENTRYPOINT_CONTEXT_CONTROLLER},
			{"complete", &umpirespb.Instruction{Instruction: &umpirespb.Instruction_CompleteNexusOperation{CompleteNexusOperation: &umpirespb.CompleteNexusOperation{CapabilitySlotId: "capability", Result: textLiteral("done")}}}, umpirespb.ENTRYPOINT_CONTEXT_CONTROLLER},
			{"start", &umpirespb.Instruction{Instruction: &umpirespb.Instruction_StartNexusOperation{StartNexusOperation: &umpirespb.StartNexusOperation{EndpointRoleId: "endpoint", Service: "service", Operation: "operation", Input: textLiteral("input")}}}, umpirespb.ENTRYPOINT_CONTEXT_WORKFLOW},
			{"await", &umpirespb.Instruction{Instruction: &umpirespb.Instruction_AwaitOutcome{AwaitOutcome: &umpirespb.Await{Instruction: &umpirespb.InstructionReference{EntrypointId: "workflow", InstructionId: "prior"}}}}, umpirespb.ENTRYPOINT_CONTEXT_WORKFLOW},
			{"finish", &umpirespb.Instruction{Instruction: &umpirespb.Instruction_Finish{Finish: &umpirespb.Finish{Result: textLiteral("done")}}}, umpirespb.ENTRYPOINT_CONTEXT_WORKFLOW},
			{"respond", &umpirespb.Instruction{Instruction: &umpirespb.Instruction_RespondNexus{RespondNexus: &umpirespb.RespondNexus{Kind: umpirespb.NEXUS_RESPONSE_KIND_SYNCHRONOUS, Result: textLiteral("done")}}}, umpirespb.ENTRYPOINT_CONTEXT_NEXUS_HANDLER},
		} {
			t.Run(context.String()+"/"+test.name, func(t *testing.T) {
				if context == test.expected {
					return
				}
				c, catalog, p := fixture(t)
				g := c.Program.Entrypoints[0]
				g.Context = context
				g.Nodes[0].Instruction = test.instruction
				switch context {
				case umpirespb.ENTRYPOINT_CONTEXT_WORKFLOW:
					g.Activation = &umpirespb.ActivationBinding{Binding: &umpirespb.ActivationBinding_Workflow{Workflow: &umpirespb.WorkflowActivation{WorkflowType: "flow", WorkerRoleId: "worker", TaskQueueRoleId: "queue"}}}
				case umpirespb.ENTRYPOINT_CONTEXT_ACTIVITY:
					g.Activation = &umpirespb.ActivationBinding{Binding: &umpirespb.ActivationBinding_Activity{Activity: &umpirespb.ActivityActivation{ActivityType: "activity", WorkerRoleId: "worker", TaskQueueRoleId: "queue"}}}
				case umpirespb.ENTRYPOINT_CONTEXT_NEXUS_HANDLER:
					g.Activation = &umpirespb.ActivationBinding{Binding: &umpirespb.ActivationBinding_NexusHandler{NexusHandler: &umpirespb.NexusHandlerActivation{Service: "service", Operation: "operation", WorkerRoleId: "worker", TaskQueueRoleId: "queue"}}}
				default:
				}
				c.Program.Roles = append(c.Program.Roles, &umpirespb.ProgramRole{RoleId: "worker", Kind: umpirespb.SYMBOLIC_ROLE_KIND_WORKER}, &umpirespb.ProgramRole{RoleId: "queue", Kind: umpirespb.SYMBOLIC_ROLE_KIND_TASK_QUEUE})
				_, err := Prepare(c, catalog, p)
				require.Error(t, err)
			})
		}
	}
}

func capabilityFixture(t *testing.T) (*umpirespb.Case, *ir.Catalog, Policy) {
	t.Helper()
	c, catalog, p := fixture(t)
	addWorker(c)
	c.Program.Slots = []*umpirespb.SlotSchema{{SlotId: "capability", Kind: umpirespb.SLOT_KIND_OPAQUE_CAPABILITY, Type: &umpirespb.ValueType{Shape: &umpirespb.ValueType_Singular{Singular: &umpirespb.SingularType{Type: &umpirespb.SingularType_OpaqueCapability{OpaqueCapability: &umpirespb.OpaqueCapabilityType{}}}}}}}
	wait := rpcNode("ready")
	wait.Instruction = &umpirespb.Instruction{Instruction: &umpirespb.Instruction_AwaitSlot{AwaitSlot: &umpirespb.AwaitSlot{SlotId: "capability"}}}
	complete := rpcNode("complete")
	complete.Dependencies = []*umpirespb.InstructionReference{{EntrypointId: "controller", InstructionId: "ready"}}
	complete.Guard = succeeded("controller", "ready")
	complete.Instruction = &umpirespb.Instruction{Instruction: &umpirespb.Instruction_CompleteNexusOperation{CompleteNexusOperation: &umpirespb.CompleteNexusOperation{CapabilitySlotId: "capability", Result: textLiteral("done")}}}
	c.Program.Entrypoints[0].Nodes = append(c.Program.Entrypoints[0].Nodes, wait, complete)
	handler := rpcNode("respond")
	handler.Instruction = &umpirespb.Instruction{Instruction: &umpirespb.Instruction_RespondNexus{RespondNexus: &umpirespb.RespondNexus{Kind: umpirespb.NEXUS_RESPONSE_KIND_ASYNCHRONOUS, CapabilitySlotId: "capability", Result: textLiteral("accepted")}}}
	c.Program.Entrypoints = append(c.Program.Entrypoints, &umpirespb.Entrypoint{EntrypointId: "handler", Context: umpirespb.ENTRYPOINT_CONTEXT_NEXUS_HANDLER, Activation: &umpirespb.ActivationBinding{Binding: &umpirespb.ActivationBinding_NexusHandler{NexusHandler: &umpirespb.NexusHandlerActivation{Service: "service", Operation: "operation", WorkerRoleId: "worker", TaskQueueRoleId: "queue"}}}, Nodes: []*umpirespb.InstructionNode{handler}})
	start := rpcNode("start")
	start.Instruction = &umpirespb.Instruction{Instruction: &umpirespb.Instruction_StartNexusOperation{StartNexusOperation: &umpirespb.StartNexusOperation{EndpointRoleId: "endpoint", Service: "service", Operation: "operation", Input: textLiteral("input")}}}
	await := rpcNode("await")
	await.Dependencies = []*umpirespb.InstructionReference{{EntrypointId: "workflow", InstructionId: "start"}}
	await.Instruction = &umpirespb.Instruction{Instruction: &umpirespb.Instruction_AwaitOutcome{AwaitOutcome: &umpirespb.Await{Instruction: &umpirespb.InstructionReference{EntrypointId: "workflow", InstructionId: "start"}}}}
	await.Outcome.Fields = append(await.Outcome.Fields, &umpirespb.OutcomeFieldSchema{Field: umpirespb.INSTRUCTION_OUTCOME_FIELD_VALUE, Type: scalar(umpirespb.SCALAR_KIND_TEXT)})
	finish := rpcNode("finish")
	finish.Dependencies = []*umpirespb.InstructionReference{{EntrypointId: "workflow", InstructionId: "await"}}
	finish.Guard = succeeded("workflow", "await")
	finish.Instruction = &umpirespb.Instruction{Instruction: &umpirespb.Instruction_Finish{Finish: &umpirespb.Finish{Result: &umpirespb.ValueExpression{Expression: &umpirespb.ValueExpression_Outcome{Outcome: &umpirespb.InstructionOutcomeReference{Instruction: &umpirespb.InstructionReference{EntrypointId: "workflow", InstructionId: "await"}, Field: umpirespb.INSTRUCTION_OUTCOME_FIELD_VALUE}}}}}}
	c.Program.Entrypoints[1].Nodes = []*umpirespb.InstructionNode{start, await, finish}
	c.Program.Entrypoints[0].Nodes[0].ActivationReservations = []*umpirespb.ActivationReservation{{EntrypointId: "workflow", Count: 1}, {EntrypointId: "handler", Count: 1}}
	return c, catalog, p
}
func TestOpaqueReadinessAndSDKPreparedPlans(t *testing.T) {
	c, catalog, p := capabilityFixture(t)
	prepared, err := Prepare(c, catalog, p)
	require.NoError(t, err)
	for name, mutate := range map[string]func(*umpirespb.Case){
		"inspect capability":        func(s *umpirespb.Case) { s.Program.Entrypoints[0].Nodes[2].Guard = present(slot("capability")) },
		"consume without readiness": func(s *umpirespb.Case) { s.Program.Entrypoints[0].Nodes[2].Guard = nil },
		"missing capability writer": func(s *umpirespb.Case) {
			s.Program.Entrypoints = s.Program.Entrypoints[:2]
			s.Program.Entrypoints[0].Nodes[0].ActivationReservations = s.Program.Entrypoints[0].Nodes[0].ActivationReservations[:1]
		},
		"capability projection": func(s *umpirespb.Case) {
			s.Program.Entrypoints[0].Nodes[0].Instruction.GetInvokeRpc().ResponseProjections = []*umpirespb.ResponseProjection{{Source: field("text"), Cardinality: umpirespb.PROJECTION_CARDINALITY_ONE, Sinks: []*umpirespb.ProjectionSink{{Sink: &umpirespb.ProjectionSink_SlotId{SlotId: "capability"}}}}}
		},
		"SDK value without success": func(s *umpirespb.Case) { s.Program.Entrypoints[1].Nodes[2].Guard = nil },
		"worker RPC":                func(s *umpirespb.Case) { s.Program.Entrypoints[1].Nodes[0].Instruction = rpcNode("call").Instruction },
	} {
		t.Run(name, func(t *testing.T) {
			source := proto.CloneOf(c)
			mutate(source)
			_, err := Prepare(source, catalog, p)
			require.Error(t, err)
		})
	}
	worker := prepared.Entrypoints()[1]
	require.Equal(t, []int{0, 1, 2}, worker.Order())
	instructions := worker.Instructions()
	require.Equal(t, "input", instructions[0].Input().Literal().GetText())
	require.Equal(t, []int{0}, instructions[1].Dependencies())
	instructions[0].Input().Literal().Value = &umpirespb.Value_Text{Text: "changed"}
	instructions[0].Source().Instruction = nil
	worker.Activation().GetWorkflow().WorkflowType = "changed"
	worker.Order()[0] = 99
	require.Equal(t, "input", worker.Instructions()[0].Input().Literal().GetText())
	require.Equal(t, "flow", worker.Activation().GetWorkflow().WorkflowType)
	require.Equal(t, []int{0, 1, 2}, worker.Order())
}

func TestOutcomeStatusesAndCleanupLocalReferences(t *testing.T) {
	c, catalog, p := fixture(t)
	first := rpcNode("release")
	second := rpcNode("confirm")
	second.Dependencies = []*umpirespb.InstructionReference{{EntrypointId: "cleanup", InstructionId: "release"}}
	second.Guard = succeeded("cleanup", "release")
	c.Program.Cleanup.Nodes = []*umpirespb.InstructionNode{first, second}
	for status := int32(1); status <= 5; status++ {
		second.Guard.GetEquals().Right.GetLiteral().GetEnumValue().Number = status
		_, err := Prepare(c, catalog, p)
		require.NoError(t, err)
	}
	second.Guard.GetEquals().Right.GetLiteral().GetEnumValue().Number = 99
	_, err := Prepare(c, catalog, p)
	require.Error(t, err)
	second.Guard = succeeded("controller", "call")
	_, err = Prepare(c, catalog, p)
	require.Error(t, err)
}

func TestSlotOwnersAndConcurrentPreparedViews(t *testing.T) {
	c, catalog, p := fixture(t)
	c.Program.Slots = []*umpirespb.SlotSchema{{SlotId: "value", Kind: umpirespb.SLOT_KIND_VALUE, Type: scalar(umpirespb.SCALAR_KIND_TEXT)}}
	c.Program.Entrypoints[0].Nodes[0].Instruction.GetInvokeRpc().ResponseProjections = []*umpirespb.ResponseProjection{{Source: field("text"), Cardinality: umpirespb.PROJECTION_CARDINALITY_ONE, Sinks: []*umpirespb.ProjectionSink{{Sink: &umpirespb.ProjectionSink_SlotId{SlotId: "value"}}}}}
	other := proto.CloneOf(c.Program.Entrypoints[0])
	other.EntrypointId = "other"
	other.Nodes[0].Instruction.GetInvokeRpc().ResponseProjections = nil
	other.Nodes[0].Guard = present(slot("value"))
	other.Nodes[0].Instruction.GetInvokeRpc().RequestAssignments = []*umpirespb.RequestAssignment{{Target: field("text"), Value: slot("value")}}
	c.Program.Entrypoints = append(c.Program.Entrypoints, other)
	prepared, err := Prepare(c, catalog, p)
	require.NoError(t, err)
	c.Program.Cleanup.Nodes = []*umpirespb.InstructionNode{proto.CloneOf(other.Nodes[0])}
	_, err = Prepare(c, catalog, p)
	require.NoError(t, err)
	addWorker(c)
	worker := c.Program.Entrypoints[len(c.Program.Entrypoints)-1]
	finish := rpcNode("finish")
	finish.Guard = present(slot("value"))
	finish.Instruction = &umpirespb.Instruction{Instruction: &umpirespb.Instruction_Finish{Finish: &umpirespb.Finish{Result: slot("value")}}}
	worker.Nodes = []*umpirespb.InstructionNode{finish}
	_, err = Prepare(c, catalog, p)
	require.Error(t, err)
	for i := 0; i < 8; i++ {
		t.Run(fmt.Sprint(i), func(t *testing.T) {
			t.Parallel()
			for j := 0; j < 10; j++ {
				plans := prepared.Entrypoints()
				plans[0].Instructions()[0].Projections()[0].Sinks[0].Sink = &umpirespb.ProjectionSink_SlotId{SlotId: "changed"}
				plans[0].Order()[0] = 99
				prepared.Snapshot().ProgramId = "changed"
				require.Equal(t, "value", prepared.Entrypoints()[0].Instructions()[0].Projections()[0].Sinks[0].GetSlotId())
				require.Equal(t, "program", prepared.View().ProgramID())
			}
		})
	}
}

func TestPrepareBoundsSurfaceBeforeCloning(t *testing.T) {
	c, catalog, p := fixture(t)
	expression := &umpirespb.ValueExpression{}
	expression.Expression = &umpirespb.ValueExpression_Negation{Negation: &umpirespb.NotExpression{Operand: expression}}
	c.Program.Entrypoints[0].Nodes[0].Guard = expression
	_, err := Prepare(c, catalog, p)
	require.Error(t, err)
	c, catalog, p = fixture(t)
	c.Program.Entrypoints[0].Nodes[0].Instruction.Instruction = (*umpirespb.Instruction_InvokeRpc)(nil)
	_, err = Prepare(c, catalog, p)
	require.Error(t, err)
}

func TestStructuralCountsAndProjectionFanout(t *testing.T) {
	for name, mutate := range map[string]func(*umpirespb.Case){
		"entrypoint count": func(c *umpirespb.Case) { addWorker(c); c.Program.Limits.MaxEntrypoints = 1 },
		"node count": func(c *umpirespb.Case) {
			c.Program.Entrypoints[0].Nodes = append(c.Program.Entrypoints[0].Nodes, rpcNode("other"))
			c.Program.Limits.MaxNodes = 1
		},
		"edge count": func(c *umpirespb.Case) {
			last := rpcNode("last")
			last.Dependencies = []*umpirespb.InstructionReference{{EntrypointId: "controller", InstructionId: "call"}, {EntrypointId: "controller", InstructionId: "other"}}
			c.Program.Entrypoints[0].Nodes = append(c.Program.Entrypoints[0].Nodes, rpcNode("other"), last)
			c.Program.Limits.MaxEdges = 1
		},
		"controller activation count": func(c *umpirespb.Case) {
			other := proto.CloneOf(c.Program.Entrypoints[0])
			other.EntrypointId = "other"
			c.Program.Entrypoints = append(c.Program.Entrypoints, other)
			c.Program.Limits.MaxActivations = 1
		},
	} {
		t.Run(name, func(t *testing.T) {
			c, catalog, p := fixture(t)
			mutate(c)
			_, err := Prepare(c, catalog, p)
			require.Error(t, err)
		})
	}
	c, catalog, p := fixture(t)
	c.Program.Observations = []*umpirespb.ObservationSchema{{ObservationId: "item", Type: scalar(umpirespb.SCALAR_KIND_TEXT)}}
	n := c.Program.Entrypoints[0].Nodes[0]
	n.Bounds.MaxEmittedEvents = 128
	n.Instruction.GetInvokeRpc().ResponseProjections = []*umpirespb.ResponseProjection{{Source: field("items"), Cardinality: umpirespb.PROJECTION_CARDINALITY_EMIT_EACH, Sinks: []*umpirespb.ProjectionSink{{Sink: &umpirespb.ProjectionSink_ObservationId{ObservationId: "item"}}}}}
	_, err := Prepare(c, catalog, p)
	require.NoError(t, err)
	n.Bounds.MaxEmittedEvents = 127
	_, err = Prepare(c, catalog, p)
	require.Error(t, err)
	n.Bounds.MaxEmittedEvents = 256
	n.Instruction.GetInvokeRpc().ResponseProjections = append(n.Instruction.GetInvokeRpc().ResponseProjections, proto.CloneOf(n.Instruction.GetInvokeRpc().ResponseProjections[0]))
	_, err = Prepare(c, catalog, p)
	require.Error(t, err)
}

func TestWholeRequestAssignments(t *testing.T) {
	c, catalog, p := fixture(t)
	typ := &umpirespb.ValueType{Shape: &umpirespb.ValueType_Singular{Singular: &umpirespb.SingularType{Type: &umpirespb.SingularType_Message{Message: &umpirespb.NamedType{ProtobufType: "example.Payload"}}}}}
	c.Program.Slots = []*umpirespb.SlotSchema{{SlotId: "request", Kind: umpirespb.SLOT_KIND_VALUE, Type: typ}}
	producer := c.Program.Entrypoints[0].Nodes[0]
	producer.Instruction.GetInvokeRpc().ResponseProjections = []*umpirespb.ResponseProjection{{Source: &umpirespb.FieldPath{}, Cardinality: umpirespb.PROJECTION_CARDINALITY_ONE, Sinks: []*umpirespb.ProjectionSink{{Sink: &umpirespb.ProjectionSink_SlotId{SlotId: "request"}}}}}
	consumer := rpcNode("copy")
	consumer.Dependencies = []*umpirespb.InstructionReference{{EntrypointId: "controller", InstructionId: "call"}}
	consumer.Guard = succeeded("controller", "call")
	consumer.Instruction.GetInvokeRpc().RequestAssignments = []*umpirespb.RequestAssignment{{Target: &umpirespb.FieldPath{}, Value: slot("request")}}
	c.Program.Entrypoints[0].Nodes = append(c.Program.Entrypoints[0].Nodes, consumer)
	_, err := Prepare(c, catalog, p)
	require.NoError(t, err)
	consumer.Instruction.GetInvokeRpc().RequestAssignments = append(consumer.Instruction.GetInvokeRpc().RequestAssignments, &umpirespb.RequestAssignment{Target: field("text"), Value: textLiteral("conflict")})
	_, err = Prepare(c, catalog, p)
	require.Error(t, err)
}
