package execution

import (
	"fmt"
	"math"
	"testing"

	"github.com/stretchr/testify/require"
	testpilotpb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/testpilot/internal/ir"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
)

func fixture(t *testing.T) (*testpilotpb.Case, *ir.Catalog, Policy) {
	t.Helper()
	catalog, err := ir.NewCatalog(&descriptorpb.FileDescriptorSet{File: []*descriptorpb.FileDescriptorProto{{Name: proto.String("admission.proto"), Package: proto.String("example"), Syntax: proto.String("proto3"), MessageType: []*descriptorpb.DescriptorProto{{Name: proto.String("Payload"), Field: []*descriptorpb.FieldDescriptorProto{{Name: proto.String("text"), Number: proto.Int32(1), Type: descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(), Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum()}, {Name: proto.String("items"), Number: proto.Int32(2), Type: descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(), Label: descriptorpb.FieldDescriptorProto_LABEL_REPEATED.Enum()}}}}, Service: []*descriptorpb.ServiceDescriptorProto{{Name: proto.String("Service"), Method: []*descriptorpb.MethodDescriptorProto{{Name: proto.String("Call"), InputType: proto.String(".example.Payload"), OutputType: proto.String(".example.Payload")}, {Name: proto.String("Stream"), InputType: proto.String(".example.Payload"), OutputType: proto.String(".example.Payload"), ServerStreaming: proto.Bool(true)}}}}}}})
	require.NoError(t, err)
	limits := &testpilotpb.ProgramLimits{MaxEntrypoints: 8, MaxNodes: 32, MaxEdges: 64, MaxActivations: 64, MaxAttempts: 32, MaxRunEvents: 256, MaxExpressionDepth: 16, MaxPathFanout: 128, MaxRequestBytes: 4096, MaxResponseBytes: 4096, MaxTotalDurationMilliseconds: 30000, MaxCleanupDurationMilliseconds: 5000}
	policy := Policy{Identity: "host", CatalogIdentity: catalog.Identity(), Roles: []RolePolicy{{ID: "endpoint", Kind: testpilotpb.ROLE_KIND_ENDPOINT, Methods: []string{"/example.Service/Call"}, ReservationCarriers: []ReservationCarrierPolicy{{Method: "/example.Service/Call", Shapes: []ReservationCarrierShape{{Context: testpilotpb.ENTRYPOINT_KIND_WORKFLOW, MaximumCount: 32}, {Context: testpilotpb.ENTRYPOINT_KIND_NEXUS_HANDLER, MaximumCount: 32}}}}}, {ID: "worker", Kind: testpilotpb.ROLE_KIND_WORKER}, {ID: "queue", Kind: testpilotpb.ROLE_KIND_TASK_QUEUE}}, Capabilities: []Opcode{InvokeRPC, AwaitSlot, CompleteNexusOperation, StartNexusOperation, Await, Finish, RespondNexus}, Limits: proto.CloneOf(limits)}
	source := &testpilotpb.Case{Version: &testpilotpb.FormatVersion{Major: 1}, CaseId: "case", Program: &testpilotpb.Program{ProgramId: "program", Roles: []*testpilotpb.RoleDefinition{{RoleId: "endpoint", Kind: testpilotpb.ROLE_KIND_ENDPOINT}}, Entrypoints: []*testpilotpb.EntrypointDefinition{{EntrypointId: "controller", Activation: &testpilotpb.EntrypointDefinition_Controller{Controller: &testpilotpb.ControllerActivation{}}, Instructions: []*testpilotpb.InstructionDefinition{rpcNode("call")}}}, Cleanup: &testpilotpb.CleanupDefinition{EntrypointId: "cleanup"}, Limits: limits}, Contract: &testpilotpb.Contract{ContractId: "contract"}}
	return source, catalog, policy
}
func scalar(kind testpilotpb.ScalarKind) *testpilotpb.ValueType {
	return &testpilotpb.ValueType{Shape: &testpilotpb.ValueType_Singular{Singular: &testpilotpb.SingularType{Type: &testpilotpb.SingularType_Scalar{Scalar: &testpilotpb.ScalarType{Kind: kind}}}}}
}

func valueSlot(id string, typ *testpilotpb.ValueType) *testpilotpb.SlotDefinition {
	return &testpilotpb.SlotDefinition{SlotId: id, Content: &testpilotpb.SlotDefinition_Value{Value: typ}}
}

func capabilitySlot(id string) *testpilotpb.SlotDefinition {
	return &testpilotpb.SlotDefinition{SlotId: id, Content: &testpilotpb.SlotDefinition_OpaqueCapability{OpaqueCapability: &testpilotpb.OpaqueCapabilityType{}}}
}
func statusSchema() *testpilotpb.InstructionOutcomeDefinition {
	return &testpilotpb.InstructionOutcomeDefinition{Fields: []*testpilotpb.OutcomeFieldDefinition{{Field: testpilotpb.INSTRUCTION_OUTCOME_FIELD_STATUS, Type: &testpilotpb.ValueType{Shape: &testpilotpb.ValueType_Singular{Singular: &testpilotpb.SingularType{Type: &testpilotpb.SingularType_Enumeration{Enumeration: &testpilotpb.NamedType{ProtobufType: "temporal.server.api.testpilot.v1.InstructionOutcomeStatus"}}}}}}}}
}
func rpcNode(id string) *testpilotpb.InstructionDefinition {
	return &testpilotpb.InstructionDefinition{InstructionId: id, Instruction: &testpilotpb.Instruction{Instruction: &testpilotpb.Instruction_InvokeRpc{InvokeRpc: &testpilotpb.InvokeRPC{EndpointRoleId: "endpoint", Method: "/example.Service/Call"}}}, Outcome: statusSchema(), Limits: &testpilotpb.InstructionLimits{TimeoutMilliseconds: 1000, MaxAttempts: 1, MaxEmittedEvents: 8, MaxResponseBytes: 4096}}
}
func field(name string) *testpilotpb.FieldPath {
	return &testpilotpb.FieldPath{Segments: []*testpilotpb.FieldPathSegment{{Field: name}}}
}
func slot(id string) *testpilotpb.ProgramExpression {
	return &testpilotpb.ProgramExpression{Expression: &testpilotpb.ProgramExpression_Slot{Slot: &testpilotpb.SlotRef{SlotId: id}}}
}
func present(value *testpilotpb.ProgramExpression) *testpilotpb.ProgramExpression {
	return &testpilotpb.ProgramExpression{Expression: &testpilotpb.ProgramExpression_Present{Present: &testpilotpb.ProgramPresentExpression{Operand: value}}}
}
func succeeded(entry, node string) *testpilotpb.ProgramExpression {
	return &testpilotpb.ProgramExpression{Expression: &testpilotpb.ProgramExpression_Equals{Equals: &testpilotpb.ProgramEqualsExpression{Left: &testpilotpb.ProgramExpression{Expression: &testpilotpb.ProgramExpression_Outcome{Outcome: &testpilotpb.InstructionOutcomeRef{Instruction: &testpilotpb.InstructionRef{EntrypointId: entry, InstructionId: node}, Field: testpilotpb.INSTRUCTION_OUTCOME_FIELD_STATUS}}}, Right: &testpilotpb.ProgramExpression{Expression: &testpilotpb.ProgramExpression_Literal{Literal: &testpilotpb.Value{Value: &testpilotpb.Value_EnumValue{EnumValue: &testpilotpb.EnumValue{Number: 1}}}}}}}}
}
func runIDExpression() *testpilotpb.ProgramExpression {
	return &testpilotpb.ProgramExpression{Expression: &testpilotpb.ProgramExpression_Run{Run: &testpilotpb.RunRef{}}}
}
func addWorker(source *testpilotpb.Case) {
	source.Program.Roles = append(source.Program.Roles, &testpilotpb.RoleDefinition{RoleId: "worker", Kind: testpilotpb.ROLE_KIND_WORKER}, &testpilotpb.RoleDefinition{RoleId: "queue", Kind: testpilotpb.ROLE_KIND_TASK_QUEUE})
	source.Program.Entrypoints = append(source.Program.Entrypoints, &testpilotpb.EntrypointDefinition{EntrypointId: "workflow", Activation: &testpilotpb.EntrypointDefinition_Workflow{Workflow: &testpilotpb.WorkflowActivation{WorkflowType: "flow", WorkerRoleId: "worker", TaskQueueRoleId: "queue"}}})
}
func TestPrepareRejectsStructuralAndPolicyErrors(t *testing.T) {
	for name, mutate := range map[string]func(*testpilotpb.Case, *Policy){
		"version":          func(c *testpilotpb.Case, _ *Policy) { c.Version.Major = 2 },
		"minor":            func(c *testpilotpb.Case, _ *Policy) { c.Version.Minor = 1 },
		"case id":          func(c *testpilotpb.Case, _ *Policy) { c.CaseId = " bad" },
		"missing contract": func(c *testpilotpb.Case, _ *Policy) { c.Contract = nil },
		"unknown field":    func(c *testpilotpb.Case, _ *Policy) { c.Program.ProtoReflect().SetUnknown([]byte{0x80, 0x06, 1}) },
		"duplicate entry": func(c *testpilotpb.Case, _ *Policy) {
			c.Program.Entrypoints = append(c.Program.Entrypoints, proto.CloneOf(c.Program.Entrypoints[0]))
		},
		"duplicate node": func(c *testpilotpb.Case, _ *Policy) {
			g := c.Program.Entrypoints[0]
			g.Instructions = append(g.Instructions, proto.CloneOf(g.Instructions[0]))
		},
		"cycle": func(c *testpilotpb.Case, _ *Policy) {
			c.Program.Entrypoints[0].Instructions[0].Dependencies = []*testpilotpb.InstructionRef{{EntrypointId: "controller", InstructionId: "call"}}
		},
		"cross entry": func(c *testpilotpb.Case, _ *Policy) {
			c.Program.Entrypoints[0].Instructions[0].Dependencies = []*testpilotpb.InstructionRef{{EntrypointId: "cleanup", InstructionId: "call"}}
		},
		"missing dependency": func(c *testpilotpb.Case, _ *Policy) {
			c.Program.Entrypoints[0].Instructions[0].Dependencies = []*testpilotpb.InstructionRef{{EntrypointId: "controller", InstructionId: "missing"}}
		},
		"binding": func(c *testpilotpb.Case, _ *Policy) { c.Program.Entrypoints[0].Activation = nil },
		"role":    func(c *testpilotpb.Case, _ *Policy) { c.Program.Roles[0].Kind = testpilotpb.ROLE_KIND_WORKER },
		"method": func(c *testpilotpb.Case, _ *Policy) {
			c.Program.Entrypoints[0].Instructions[0].Instruction.GetInvokeRpc().Method = "/example.Service/Missing"
		},
		"authorization":    func(_ *testpilotpb.Case, p *Policy) { p.Roles[0].Methods = nil },
		"capability":       func(_ *testpilotpb.Case, p *Policy) { p.Capabilities = nil },
		"catalog identity": func(_ *testpilotpb.Case, p *Policy) { p.CatalogIdentity = "other" },
		"node timeout": func(c *testpilotpb.Case, _ *Policy) {
			c.Program.Entrypoints[0].Instructions[0].Limits.TimeoutMilliseconds = 0
		},
		"attempt bound": func(c *testpilotpb.Case, _ *Policy) { c.Program.Entrypoints[0].Instructions[0].Limits.MaxAttempts = 33 },
		"response bound": func(c *testpilotpb.Case, _ *Policy) {
			c.Program.Entrypoints[0].Instructions[0].Limits.MaxResponseBytes = 4097
		},
		"event bound": func(c *testpilotpb.Case, _ *Policy) {
			c.Program.Entrypoints[0].Instructions[0].Limits.MaxEmittedEvents = 257
		},
		"rpc raw outcome": func(c *testpilotpb.Case, _ *Policy) {
			c.Program.Entrypoints[0].Instructions[0].Outcome.Fields = append(c.Program.Entrypoints[0].Instructions[0].Outcome.Fields, &testpilotpb.OutcomeFieldDefinition{Field: testpilotpb.INSTRUCTION_OUTCOME_FIELD_VALUE, Type: scalar(testpilotpb.SCALAR_KIND_TEXT)})
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

func TestRunIDIntrinsicIsOnlyAvailableToProgramInputs(t *testing.T) {
	c, catalog, policy := fixture(t)
	node := c.Program.Entrypoints[0].Instructions[0]
	node.Instruction.GetInvokeRpc().RequestAssignments = []*testpilotpb.RequestAssignment{{
		Target: field("text"), Value: runIDExpression(),
	}}
	prepared, err := Prepare(c, catalog, policy)
	require.NoError(t, err)
	for _, runID := range []string{"run-one", "run-two"} {
		store, err := newValueStore(prepared, runID)
		require.NoError(t, err)
		values, err := store.activate("controller", "activation")
		require.NoError(t, err)
		request, enabled, _, err := values.request(t.Context(), Coordinate{
			RunID: runID, EntrypointID: "controller", ActivationID: "activation",
			InstructionID: "call", Attempt: 1,
		}, prepared.graphs[0].runtimeWork)
		require.NoError(t, err)
		require.True(t, enabled)
		field := request.ProtoReflect().Descriptor().Fields().ByName("text")
		require.Equal(t, runID, request.ProtoReflect().Get(field).String())
	}

	c, catalog, policy = fixture(t)
	c.Program.Entrypoints[0].Instructions[0].Guard = present(runIDExpression())
	_, err = Prepare(c, catalog, policy)
	require.Error(t, err)

	c, catalog, policy = fixture(t)
	addWorker(c)
	c.Program.Entrypoints[1].Instructions = []*testpilotpb.InstructionDefinition{{
		InstructionId: "finish",
		Instruction: &testpilotpb.Instruction{Instruction: &testpilotpb.Instruction_Finish{Finish: &testpilotpb.Finish{
			Result: runIDExpression(),
		}}},
		Outcome: statusSchema(),
		Limits:  &testpilotpb.InstructionLimits{TimeoutMilliseconds: 1000, MaxAttempts: 1, MaxEmittedEvents: 1, MaxResponseBytes: 4096},
	}}
	_, err = Prepare(c, catalog, policy)
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
			node := c.Program.Entrypoints[0].Instructions[0]
			node.Limits.MaxAttempts = test.local
			c.Program.Limits.MaxAttempts = test.global
			c.Program.Limits.MaxActivations = test.ceiling
			node.ActivationReservations = []*testpilotpb.ActivationReservationDefinition{{EntrypointId: "workflow", Count: test.count}}
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
	c.Program.Slots = []*testpilotpb.SlotDefinition{valueSlot("result", scalar(testpilotpb.SCALAR_KIND_TEXT))}
	c.Program.Observations = []*testpilotpb.ObservationDefinition{{ObservationId: "text", Type: scalar(testpilotpb.SCALAR_KIND_TEXT)}}
	producer := c.Program.Entrypoints[0].Instructions[0]
	producer.Instruction.GetInvokeRpc().ResponseProjections = []*testpilotpb.ResponseProjection{{Source: field("text"), Kind: testpilotpb.PROJECTION_KIND_ONE, Targets: []*testpilotpb.ProjectionTarget{{Target: &testpilotpb.ProjectionTarget_SlotId{SlotId: "result"}}, {Target: &testpilotpb.ProjectionTarget_ObservationId{ObservationId: "text"}}}}}
	consumer := rpcNode("consume")
	consumer.Dependencies = []*testpilotpb.InstructionRef{{EntrypointId: "controller", InstructionId: "call"}}
	consumer.Guard = succeeded("controller", "call")
	consumer.Instruction.GetInvokeRpc().RequestAssignments = []*testpilotpb.RequestAssignment{{Target: field("text"), Value: slot("result")}}
	c.Program.Entrypoints[0].Instructions = append(c.Program.Entrypoints[0].Instructions, consumer)
	prepared, err := Prepare(c, catalog, p)
	require.NoError(t, err)
	for name, mutate := range map[string]func(*testpilotpb.Case){"unguarded": func(s *testpilotpb.Case) { s.Program.Entrypoints[0].Instructions[1].Guard = nil }, "missing dependency": func(s *testpilotpb.Case) { s.Program.Entrypoints[0].Instructions[1].Dependencies = nil }, "second writer": func(s *testpilotpb.Case) {
		s.Program.Entrypoints[0].Instructions[1].Instruction.GetInvokeRpc().ResponseProjections = proto.CloneOf(producer.Instruction).GetInvokeRpc().ResponseProjections
	}, "assignment overlap": func(s *testpilotpb.Case) {
		rpc := s.Program.Entrypoints[0].Instructions[1].Instruction.GetInvokeRpc()
		rpc.RequestAssignments = append(rpc.RequestAssignments, proto.CloneOf(rpc.RequestAssignments[0]))
	}, "crossed cardinality": func(s *testpilotpb.Case) {
		s.Program.Entrypoints[0].Instructions[0].Instruction.GetInvokeRpc().ResponseProjections[0].Kind = testpilotpb.PROJECTION_KIND_EMIT_EACH
	}, "undeclared outcome": func(s *testpilotpb.Case) { s.Program.Entrypoints[0].Instructions[0].Outcome.Fields = nil }} {
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
	for name, mutate := range map[string]func(*testpilotpb.Case){
		"missing target": func(c *testpilotpb.Case) {
			c.Program.Entrypoints[0].Instructions[0].ActivationReservations[0].EntrypointId = "missing"
		},
		"controller target": func(c *testpilotpb.Case) {
			c.Program.Entrypoints[0].Instructions[0].ActivationReservations[0].EntrypointId = "controller"
		},
		"wrong binding": func(c *testpilotpb.Case) {
			c.Program.Entrypoints[1].Activation = &testpilotpb.EntrypointDefinition_Controller{Controller: &testpilotpb.ControllerActivation{}}
		},
		"activity target": func(c *testpilotpb.Case) {
			g := c.Program.Entrypoints[1]
			g.Activation = &testpilotpb.EntrypointDefinition_Activity{Activity: &testpilotpb.ActivityActivation{ActivityType: "activity", WorkerRoleId: "worker", TaskQueueRoleId: "queue"}}
		},
		"duplicate target": func(c *testpilotpb.Case) {
			n := c.Program.Entrypoints[0].Instructions[0]
			n.ActivationReservations = append(n.ActivationReservations, proto.CloneOf(n.ActivationReservations[0]))
		},
		"worker reservation": func(c *testpilotpb.Case) {
			n := c.Program.Entrypoints[0].Instructions[0]
			worker := proto.CloneOf(n)
			worker.Instruction = &testpilotpb.Instruction{Instruction: &testpilotpb.Instruction_Finish{Finish: &testpilotpb.Finish{Result: textLiteral("done")}}}
			c.Program.Entrypoints[1].Instructions = []*testpilotpb.InstructionDefinition{worker}
		},
		"cleanup reservation": func(c *testpilotpb.Case) {
			c.Program.Cleanup.Instructions = []*testpilotpb.InstructionDefinition{proto.CloneOf(c.Program.Entrypoints[0].Instructions[0])}
		},
		"sum overflow": func(c *testpilotpb.Case) {
			other := proto.CloneOf(c.Program.Entrypoints[1])
			other.EntrypointId = "other"
			c.Program.Entrypoints = append(c.Program.Entrypoints, other)
			n := c.Program.Entrypoints[0].Instructions[0]
			n.ActivationReservations = []*testpilotpb.ActivationReservationDefinition{{EntrypointId: "workflow", Count: math.MaxInt64}, {EntrypointId: "other", Count: math.MaxInt64}}
		},
	} {
		t.Run(name, func(t *testing.T) {
			c, catalog, p := fixture(t)
			addWorker(c)
			c.Program.Entrypoints[0].Instructions[0].ActivationReservations = []*testpilotpb.ActivationReservationDefinition{{EntrypointId: "workflow", Count: 1}}
			mutate(c)
			_, err := Prepare(c, catalog, p)
			require.Error(t, err)
		})
	}
	c, catalog, p := fixture(t)
	addWorker(c)
	first := c.Program.Entrypoints[0].Instructions[0]
	first.Limits.MaxAttempts = 2
	first.ActivationReservations = []*testpilotpb.ActivationReservationDefinition{{EntrypointId: "workflow", Count: 5}}
	second := proto.CloneOf(c.Program.Entrypoints[0])
	second.EntrypointId = "second"
	second.Instructions[0].Limits.MaxAttempts = 5
	second.Instructions[0].ActivationReservations[0].Count = 3
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
	second.Instructions[0].ActivationReservations = nil
	c.Program.Limits.MaxActivations = 2
	prepared, err = Prepare(c, catalog, p)
	require.NoError(t, err)
	require.Equal(t, int64(2), prepared.View().MaximumActivations())
}
func textLiteral(value string) *testpilotpb.ProgramExpression {
	return &testpilotpb.ProgramExpression{Expression: &testpilotpb.ProgramExpression_Literal{Literal: &testpilotpb.Value{Value: &testpilotpb.Value_Text{Text: value}}}}
}

func TestInstructionContextMatrix(t *testing.T) {
	for _, context := range []testpilotpb.EntrypointKind{testpilotpb.ENTRYPOINT_KIND_CONTROLLER, testpilotpb.ENTRYPOINT_KIND_WORKFLOW, testpilotpb.ENTRYPOINT_KIND_ACTIVITY, testpilotpb.ENTRYPOINT_KIND_NEXUS_HANDLER} {
		for _, test := range []struct {
			name        string
			instruction *testpilotpb.Instruction
			expected    testpilotpb.EntrypointKind
		}{
			{"rpc", rpcNode("call").Instruction, testpilotpb.ENTRYPOINT_KIND_CONTROLLER},
			{"await slot", &testpilotpb.Instruction{Instruction: &testpilotpb.Instruction_AwaitSlot{AwaitSlot: &testpilotpb.AwaitSlot{SlotId: "value"}}}, testpilotpb.ENTRYPOINT_KIND_CONTROLLER},
			{"complete", &testpilotpb.Instruction{Instruction: &testpilotpb.Instruction_CompleteNexusOperation{CompleteNexusOperation: &testpilotpb.CompleteNexusOperation{CapabilitySlotId: "capability", Result: textLiteral("done")}}}, testpilotpb.ENTRYPOINT_KIND_CONTROLLER},
			{"start", &testpilotpb.Instruction{Instruction: &testpilotpb.Instruction_StartNexusOperation{StartNexusOperation: &testpilotpb.StartNexusOperation{EndpointRoleId: "endpoint", Service: "service", Operation: "operation", Input: textLiteral("input")}}}, testpilotpb.ENTRYPOINT_KIND_WORKFLOW},
			{"await", &testpilotpb.Instruction{Instruction: &testpilotpb.Instruction_AwaitOutcome{AwaitOutcome: &testpilotpb.AwaitInstruction{Instruction: &testpilotpb.InstructionRef{EntrypointId: "workflow", InstructionId: "prior"}}}}, testpilotpb.ENTRYPOINT_KIND_WORKFLOW},
			{"finish", &testpilotpb.Instruction{Instruction: &testpilotpb.Instruction_Finish{Finish: &testpilotpb.Finish{Result: textLiteral("done")}}}, testpilotpb.ENTRYPOINT_KIND_WORKFLOW},
			{"respond", &testpilotpb.Instruction{Instruction: &testpilotpb.Instruction_RespondNexus{RespondNexus: &testpilotpb.RespondNexus{Kind: testpilotpb.NEXUS_RESPONSE_KIND_SYNCHRONOUS, Result: textLiteral("done")}}}, testpilotpb.ENTRYPOINT_KIND_NEXUS_HANDLER},
		} {
			t.Run(context.String()+"/"+test.name, func(t *testing.T) {
				if context == test.expected {
					return
				}
				c, catalog, p := fixture(t)
				g := c.Program.Entrypoints[0]
				g.Instructions[0].Instruction = test.instruction
				switch context {
				case testpilotpb.ENTRYPOINT_KIND_WORKFLOW:
					g.Activation = &testpilotpb.EntrypointDefinition_Workflow{Workflow: &testpilotpb.WorkflowActivation{WorkflowType: "flow", WorkerRoleId: "worker", TaskQueueRoleId: "queue"}}
				case testpilotpb.ENTRYPOINT_KIND_ACTIVITY:
					g.Activation = &testpilotpb.EntrypointDefinition_Activity{Activity: &testpilotpb.ActivityActivation{ActivityType: "activity", WorkerRoleId: "worker", TaskQueueRoleId: "queue"}}
				case testpilotpb.ENTRYPOINT_KIND_NEXUS_HANDLER:
					g.Activation = &testpilotpb.EntrypointDefinition_NexusHandler{NexusHandler: &testpilotpb.NexusHandlerActivation{Service: "service", Operation: "operation", WorkerRoleId: "worker", TaskQueueRoleId: "queue"}}
				default:
				}
				c.Program.Roles = append(c.Program.Roles, &testpilotpb.RoleDefinition{RoleId: "worker", Kind: testpilotpb.ROLE_KIND_WORKER}, &testpilotpb.RoleDefinition{RoleId: "queue", Kind: testpilotpb.ROLE_KIND_TASK_QUEUE})
				_, err := Prepare(c, catalog, p)
				require.Error(t, err)
			})
		}
	}
}

func capabilityFixture(t *testing.T) (*testpilotpb.Case, *ir.Catalog, Policy) {
	t.Helper()
	c, catalog, p := fixture(t)
	addWorker(c)
	c.Program.Slots = []*testpilotpb.SlotDefinition{capabilitySlot("capability")}
	wait := rpcNode("ready")
	wait.Instruction = &testpilotpb.Instruction{Instruction: &testpilotpb.Instruction_AwaitSlot{AwaitSlot: &testpilotpb.AwaitSlot{SlotId: "capability"}}}
	complete := rpcNode("complete")
	complete.Dependencies = []*testpilotpb.InstructionRef{{EntrypointId: "controller", InstructionId: "ready"}}
	complete.Guard = succeeded("controller", "ready")
	complete.Instruction = &testpilotpb.Instruction{Instruction: &testpilotpb.Instruction_CompleteNexusOperation{CompleteNexusOperation: &testpilotpb.CompleteNexusOperation{CapabilitySlotId: "capability", Result: textLiteral("done")}}}
	c.Program.Entrypoints[0].Instructions = append(c.Program.Entrypoints[0].Instructions, wait, complete)
	handler := rpcNode("respond")
	handler.Instruction = &testpilotpb.Instruction{Instruction: &testpilotpb.Instruction_RespondNexus{RespondNexus: &testpilotpb.RespondNexus{Kind: testpilotpb.NEXUS_RESPONSE_KIND_ASYNCHRONOUS, CapabilitySlotId: "capability", Result: textLiteral("accepted")}}}
	c.Program.Entrypoints = append(c.Program.Entrypoints, &testpilotpb.EntrypointDefinition{EntrypointId: "handler", Activation: &testpilotpb.EntrypointDefinition_NexusHandler{NexusHandler: &testpilotpb.NexusHandlerActivation{Service: "service", Operation: "operation", WorkerRoleId: "worker", TaskQueueRoleId: "queue"}}, Instructions: []*testpilotpb.InstructionDefinition{handler}})
	start := rpcNode("start")
	start.Instruction = &testpilotpb.Instruction{Instruction: &testpilotpb.Instruction_StartNexusOperation{StartNexusOperation: &testpilotpb.StartNexusOperation{EndpointRoleId: "endpoint", Service: "service", Operation: "operation", Input: textLiteral("input")}}}
	await := rpcNode("await")
	await.Dependencies = []*testpilotpb.InstructionRef{{EntrypointId: "workflow", InstructionId: "start"}}
	await.Instruction = &testpilotpb.Instruction{Instruction: &testpilotpb.Instruction_AwaitOutcome{AwaitOutcome: &testpilotpb.AwaitInstruction{Instruction: &testpilotpb.InstructionRef{EntrypointId: "workflow", InstructionId: "start"}}}}
	await.Outcome.Fields = append(await.Outcome.Fields, &testpilotpb.OutcomeFieldDefinition{Field: testpilotpb.INSTRUCTION_OUTCOME_FIELD_VALUE, Type: scalar(testpilotpb.SCALAR_KIND_TEXT)})
	finish := rpcNode("finish")
	finish.Dependencies = []*testpilotpb.InstructionRef{{EntrypointId: "workflow", InstructionId: "await"}}
	finish.Guard = succeeded("workflow", "await")
	finish.Instruction = &testpilotpb.Instruction{Instruction: &testpilotpb.Instruction_Finish{Finish: &testpilotpb.Finish{Result: &testpilotpb.ProgramExpression{Expression: &testpilotpb.ProgramExpression_Outcome{Outcome: &testpilotpb.InstructionOutcomeRef{Instruction: &testpilotpb.InstructionRef{EntrypointId: "workflow", InstructionId: "await"}, Field: testpilotpb.INSTRUCTION_OUTCOME_FIELD_VALUE}}}}}}
	c.Program.Entrypoints[1].Instructions = []*testpilotpb.InstructionDefinition{start, await, finish}
	c.Program.Entrypoints[0].Instructions[0].ActivationReservations = []*testpilotpb.ActivationReservationDefinition{{EntrypointId: "workflow", Count: 1}, {EntrypointId: "handler", Count: 1}}
	return c, catalog, p
}
func TestOpaqueReadinessAndSDKPreparedPlans(t *testing.T) {
	c, catalog, p := capabilityFixture(t)
	prepared, err := Prepare(c, catalog, p)
	require.NoError(t, err)
	for name, mutate := range map[string]func(*testpilotpb.Case){
		"inspect capability": func(s *testpilotpb.Case) {
			s.Program.Entrypoints[0].Instructions[2].Guard = present(slot("capability"))
		},
		"consume without readiness": func(s *testpilotpb.Case) { s.Program.Entrypoints[0].Instructions[2].Guard = nil },
		"missing capability writer": func(s *testpilotpb.Case) {
			s.Program.Entrypoints = s.Program.Entrypoints[:2]
			s.Program.Entrypoints[0].Instructions[0].ActivationReservations = s.Program.Entrypoints[0].Instructions[0].ActivationReservations[:1]
		},
		"capability projection": func(s *testpilotpb.Case) {
			s.Program.Entrypoints[0].Instructions[0].Instruction.GetInvokeRpc().ResponseProjections = []*testpilotpb.ResponseProjection{{Source: field("text"), Kind: testpilotpb.PROJECTION_KIND_ONE, Targets: []*testpilotpb.ProjectionTarget{{Target: &testpilotpb.ProjectionTarget_SlotId{SlotId: "capability"}}}}}
		},
		"SDK value without success": func(s *testpilotpb.Case) { s.Program.Entrypoints[1].Instructions[2].Guard = nil },
		"worker RPC": func(s *testpilotpb.Case) {
			s.Program.Entrypoints[1].Instructions[0].Instruction = rpcNode("call").Instruction
		},
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
	instructions[0].Input().Literal().Value = &testpilotpb.Value_Text{Text: "changed"}
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
	second.Dependencies = []*testpilotpb.InstructionRef{{EntrypointId: "cleanup", InstructionId: "release"}}
	second.Guard = succeeded("cleanup", "release")
	c.Program.Cleanup.Instructions = []*testpilotpb.InstructionDefinition{first, second}
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
	c.Program.Slots = []*testpilotpb.SlotDefinition{valueSlot("value", scalar(testpilotpb.SCALAR_KIND_TEXT))}
	c.Program.Entrypoints[0].Instructions[0].Instruction.GetInvokeRpc().ResponseProjections = []*testpilotpb.ResponseProjection{{Source: field("text"), Kind: testpilotpb.PROJECTION_KIND_ONE, Targets: []*testpilotpb.ProjectionTarget{{Target: &testpilotpb.ProjectionTarget_SlotId{SlotId: "value"}}}}}
	other := proto.CloneOf(c.Program.Entrypoints[0])
	other.EntrypointId = "other"
	other.Instructions[0].Instruction.GetInvokeRpc().ResponseProjections = nil
	other.Instructions[0].Guard = present(slot("value"))
	other.Instructions[0].Instruction.GetInvokeRpc().RequestAssignments = []*testpilotpb.RequestAssignment{{Target: field("text"), Value: slot("value")}}
	c.Program.Entrypoints = append(c.Program.Entrypoints, other)
	prepared, err := Prepare(c, catalog, p)
	require.NoError(t, err)
	c.Program.Cleanup.Instructions = []*testpilotpb.InstructionDefinition{proto.CloneOf(other.Instructions[0])}
	_, err = Prepare(c, catalog, p)
	require.NoError(t, err)
	addWorker(c)
	worker := c.Program.Entrypoints[len(c.Program.Entrypoints)-1]
	finish := rpcNode("finish")
	finish.Guard = present(slot("value"))
	finish.Instruction = &testpilotpb.Instruction{Instruction: &testpilotpb.Instruction_Finish{Finish: &testpilotpb.Finish{Result: slot("value")}}}
	worker.Instructions = []*testpilotpb.InstructionDefinition{finish}
	_, err = Prepare(c, catalog, p)
	require.Error(t, err)
	for i := 0; i < 8; i++ {
		t.Run(fmt.Sprint(i), func(t *testing.T) {
			t.Parallel()
			for j := 0; j < 10; j++ {
				plans := prepared.Entrypoints()
				plans[0].Instructions()[0].Projections()[0].Sinks[0].Target = &testpilotpb.ProjectionTarget_SlotId{SlotId: "changed"}
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
	expression := &testpilotpb.ProgramExpression{}
	expression.Expression = &testpilotpb.ProgramExpression_Negation{Negation: &testpilotpb.ProgramNotExpression{Operand: expression}}
	c.Program.Entrypoints[0].Instructions[0].Guard = expression
	_, err := Prepare(c, catalog, p)
	require.Error(t, err)
	c, catalog, p = fixture(t)
	c.Program.Entrypoints[0].Instructions[0].Instruction.Instruction = (*testpilotpb.Instruction_InvokeRpc)(nil)
	_, err = Prepare(c, catalog, p)
	require.Error(t, err)
}

func TestStructuralCountsAndProjectionFanout(t *testing.T) {
	for name, mutate := range map[string]func(*testpilotpb.Case){
		"entrypoint count": func(c *testpilotpb.Case) { addWorker(c); c.Program.Limits.MaxEntrypoints = 1 },
		"node count": func(c *testpilotpb.Case) {
			c.Program.Entrypoints[0].Instructions = append(c.Program.Entrypoints[0].Instructions, rpcNode("other"))
			c.Program.Limits.MaxNodes = 1
		},
		"edge count": func(c *testpilotpb.Case) {
			last := rpcNode("last")
			last.Dependencies = []*testpilotpb.InstructionRef{{EntrypointId: "controller", InstructionId: "call"}, {EntrypointId: "controller", InstructionId: "other"}}
			c.Program.Entrypoints[0].Instructions = append(c.Program.Entrypoints[0].Instructions, rpcNode("other"), last)
			c.Program.Limits.MaxEdges = 1
		},
		"controller activation count": func(c *testpilotpb.Case) {
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
	c.Program.Observations = []*testpilotpb.ObservationDefinition{{ObservationId: "item", Type: scalar(testpilotpb.SCALAR_KIND_TEXT)}}
	n := c.Program.Entrypoints[0].Instructions[0]
	n.Limits.MaxEmittedEvents = 128
	n.Instruction.GetInvokeRpc().ResponseProjections = []*testpilotpb.ResponseProjection{{Source: field("items"), Kind: testpilotpb.PROJECTION_KIND_EMIT_EACH, Targets: []*testpilotpb.ProjectionTarget{{Target: &testpilotpb.ProjectionTarget_ObservationId{ObservationId: "item"}}}}}
	_, err := Prepare(c, catalog, p)
	require.NoError(t, err)
	n.Limits.MaxEmittedEvents = 127
	_, err = Prepare(c, catalog, p)
	require.Error(t, err)
	n.Limits.MaxEmittedEvents = 256
	n.Instruction.GetInvokeRpc().ResponseProjections = append(n.Instruction.GetInvokeRpc().ResponseProjections, proto.CloneOf(n.Instruction.GetInvokeRpc().ResponseProjections[0]))
	_, err = Prepare(c, catalog, p)
	require.Error(t, err)
}

func TestWholeRequestAssignments(t *testing.T) {
	c, catalog, p := fixture(t)
	typ := &testpilotpb.ValueType{Shape: &testpilotpb.ValueType_Singular{Singular: &testpilotpb.SingularType{Type: &testpilotpb.SingularType_Message{Message: &testpilotpb.NamedType{ProtobufType: "example.Payload"}}}}}
	c.Program.Slots = []*testpilotpb.SlotDefinition{valueSlot("request", typ)}
	producer := c.Program.Entrypoints[0].Instructions[0]
	producer.Instruction.GetInvokeRpc().ResponseProjections = []*testpilotpb.ResponseProjection{{Source: &testpilotpb.FieldPath{}, Kind: testpilotpb.PROJECTION_KIND_ONE, Targets: []*testpilotpb.ProjectionTarget{{Target: &testpilotpb.ProjectionTarget_SlotId{SlotId: "request"}}}}}
	consumer := rpcNode("copy")
	consumer.Dependencies = []*testpilotpb.InstructionRef{{EntrypointId: "controller", InstructionId: "call"}}
	consumer.Guard = succeeded("controller", "call")
	consumer.Instruction.GetInvokeRpc().RequestAssignments = []*testpilotpb.RequestAssignment{{Target: &testpilotpb.FieldPath{}, Value: slot("request")}}
	c.Program.Entrypoints[0].Instructions = append(c.Program.Entrypoints[0].Instructions, consumer)
	_, err := Prepare(c, catalog, p)
	require.NoError(t, err)
	consumer.Instruction.GetInvokeRpc().RequestAssignments = append(consumer.Instruction.GetInvokeRpc().RequestAssignments, &testpilotpb.RequestAssignment{Target: field("text"), Value: textLiteral("conflict")})
	_, err = Prepare(c, catalog, p)
	require.Error(t, err)
}

func TestAwaitRequiresNexusStart(t *testing.T) {
	for _, target := range []string{"start", "await", "finish"} {
		t.Run(target, func(t *testing.T) {
			c, catalog, p := capabilityFixture(t)
			g := c.Program.Entrypoints[1]
			n := rpcNode("second_await")
			n.Dependencies = []*testpilotpb.InstructionRef{{EntrypointId: "workflow", InstructionId: target}}
			n.Instruction = &testpilotpb.Instruction{Instruction: &testpilotpb.Instruction_AwaitOutcome{AwaitOutcome: &testpilotpb.AwaitInstruction{Instruction: proto.CloneOf(n.Dependencies[0])}}}
			g.Instructions = append([]*testpilotpb.InstructionDefinition{n}, g.Instructions...)
			_, err := Prepare(c, catalog, p)
			if target == "start" {
				require.NoError(t, err)
			} else {
				require.ErrorContains(t, err, "StartNexusOperation")
			}
		})
	}
}

func TestEndpointMethodPolicyRejectsDuplicates(t *testing.T) {
	c, catalog, p := fixture(t)
	p.Roles[0].Methods = append(p.Roles[0].Methods, p.Roles[0].Methods[0])
	_, err := Prepare(c, catalog, p)
	require.ErrorContains(t, err, "duplicate method")
}
