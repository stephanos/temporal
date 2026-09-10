package execution

import (
	"fmt"
	"math"
	"testing"

	"github.com/stretchr/testify/require"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/testpilot/internal/ir"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
)

func fixture(t *testing.T) (*testpilotspb.Case, *ir.Catalog, Profile) {
	t.Helper()
	catalog, err := ir.NewCatalog(&descriptorpb.FileDescriptorSet{File: []*descriptorpb.FileDescriptorProto{{Name: proto.String("admission.proto"), Package: proto.String("example"), Syntax: proto.String("proto3"), MessageType: []*descriptorpb.DescriptorProto{{Name: proto.String("Payload"), Field: []*descriptorpb.FieldDescriptorProto{{Name: proto.String("text"), Number: proto.Int32(1), Type: descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(), Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum()}, {Name: proto.String("items"), Number: proto.Int32(2), Type: descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(), Label: descriptorpb.FieldDescriptorProto_LABEL_REPEATED.Enum()}}}}, Service: []*descriptorpb.ServiceDescriptorProto{{Name: proto.String("Service"), Method: []*descriptorpb.MethodDescriptorProto{{Name: proto.String("Call"), InputType: proto.String(".example.Payload"), OutputType: proto.String(".example.Payload")}, {Name: proto.String("Stream"), InputType: proto.String(".example.Payload"), OutputType: proto.String(".example.Payload"), ServerStreaming: proto.Bool(true)}}}}}}})
	require.NoError(t, err)
	limits := &testpilotspb.ProgramLimits{MaxEntrypoints: 8, MaxNodes: 32, MaxEdges: 64, MaxActivations: 64, MaxAttempts: 32, MaxRunEvents: 256, MaxExpressionDepth: 16, MaxPathFanout: 128, MaxRequestBytes: 4096, MaxResponseBytes: 4096, MaxTotalDurationMilliseconds: 30000, MaxCleanupDurationMilliseconds: 5000}
	policy := Profile{Identity: "host", CatalogIdentity: catalog.Identity(), Roles: []RolePolicy{{ID: "endpoint", Kind: testpilotspb.ROLE_KIND_ENDPOINT, Methods: []string{"/example.Service/Call"}, ReservationCarriers: []ReservationCarrierPolicy{{Method: "/example.Service/Call", Shapes: []ReservationCarrierShape{{Kind: testpilotspb.ENTRYPOINT_KIND_WORKFLOW, MaximumCount: 32}, {Kind: testpilotspb.ENTRYPOINT_KIND_NEXUS_HANDLER, MaximumCount: 32}}}}}, {ID: "worker", Kind: testpilotspb.ROLE_KIND_WORKER}, {ID: "queue", Kind: testpilotspb.ROLE_KIND_TASK_QUEUE}}, Opcodes: []Opcode{InvokeRPC, AwaitSlot, CompleteNexusOperation, StartNexusOperation, Await, Finish, RespondNexus}, Limits: proto.CloneOf(limits)}
	source := &testpilotspb.Case{Version: &testpilotspb.FormatVersion{Major: 1}, CaseId: "case", Program: &testpilotspb.Program{ProgramId: "program", Roles: []*testpilotspb.RoleDefinition{{RoleId: "endpoint", Kind: testpilotspb.ROLE_KIND_ENDPOINT}}, Entrypoints: []*testpilotspb.EntrypointDefinition{{EntrypointId: "controller", Activation: &testpilotspb.EntrypointDefinition_Controller{Controller: &testpilotspb.ControllerActivation{}}, Instructions: []*testpilotspb.InstructionDefinition{rpcNode("call")}}}, Cleanup: &testpilotspb.CleanupDefinition{EntrypointId: "cleanup"}, Limits: limits}, Contract: &testpilotspb.Contract{ContractId: "contract"}}
	return source, catalog, policy
}
func scalar(kind testpilotspb.ScalarKind) *testpilotspb.ValueType {
	return &testpilotspb.ValueType{Shape: &testpilotspb.ValueType_Singular{Singular: &testpilotspb.SingularType{Type: &testpilotspb.SingularType_Scalar{Scalar: &testpilotspb.ScalarType{Kind: kind}}}}}
}

func valueSlot(id string, typ *testpilotspb.ValueType) *testpilotspb.SlotDefinition {
	return &testpilotspb.SlotDefinition{SlotId: id, Content: &testpilotspb.SlotDefinition_Value{Value: typ}}
}

func capabilitySlot(id string) *testpilotspb.SlotDefinition {
	return &testpilotspb.SlotDefinition{SlotId: id, Content: &testpilotspb.SlotDefinition_OpaqueCapability{OpaqueCapability: &testpilotspb.OpaqueCapabilityType{}}}
}
func statusSchema() *testpilotspb.InstructionOutcomeDefinition {
	return &testpilotspb.InstructionOutcomeDefinition{Fields: []*testpilotspb.OutcomeFieldDefinition{{Field: testpilotspb.INSTRUCTION_OUTCOME_FIELD_STATUS, Type: &testpilotspb.ValueType{Shape: &testpilotspb.ValueType_Singular{Singular: &testpilotspb.SingularType{Type: &testpilotspb.SingularType_Enumeration{Enumeration: &testpilotspb.NamedType{ProtobufType: "temporal.server.api.testpilot.v1.InstructionOutcomeStatus"}}}}}}}}
}
func rpcNode(id string) *testpilotspb.InstructionDefinition {
	return &testpilotspb.InstructionDefinition{InstructionId: id, Instruction: &testpilotspb.Instruction{Instruction: &testpilotspb.Instruction_InvokeRpc{InvokeRpc: &testpilotspb.InvokeRPC{EndpointRoleId: "endpoint", Method: "/example.Service/Call"}}}, Outcome: statusSchema(), Limits: &testpilotspb.InstructionLimits{TimeoutMilliseconds: 1000, MaxAttempts: 1, MaxEmittedEvents: 8, MaxResponseBytes: 4096}}
}
func field(name string) *testpilotspb.FieldPath {
	return &testpilotspb.FieldPath{Segments: []*testpilotspb.FieldPathSegment{{Field: name}}}
}
func slot(id string) *testpilotspb.ProgramExpression {
	return &testpilotspb.ProgramExpression{Expression: &testpilotspb.ProgramExpression_Slot{Slot: &testpilotspb.SlotRef{SlotId: id}}}
}
func present(value *testpilotspb.ProgramExpression) *testpilotspb.ProgramExpression {
	return &testpilotspb.ProgramExpression{Expression: &testpilotspb.ProgramExpression_Present{Present: &testpilotspb.ProgramPresentExpression{Operand: value}}}
}
func succeeded(entry, node string) *testpilotspb.ProgramExpression {
	return &testpilotspb.ProgramExpression{Expression: &testpilotspb.ProgramExpression_Equals{Equals: &testpilotspb.ProgramEqualsExpression{Left: &testpilotspb.ProgramExpression{Expression: &testpilotspb.ProgramExpression_Outcome{Outcome: &testpilotspb.InstructionOutcomeRef{Instruction: &testpilotspb.InstructionRef{EntrypointId: entry, InstructionId: node}, Field: testpilotspb.INSTRUCTION_OUTCOME_FIELD_STATUS}}}, Right: &testpilotspb.ProgramExpression{Expression: &testpilotspb.ProgramExpression_Literal{Literal: &testpilotspb.Value{Value: &testpilotspb.Value_EnumValue{EnumValue: &testpilotspb.EnumValue{Number: 1}}}}}}}}
}
func runIDExpression() *testpilotspb.ProgramExpression {
	return &testpilotspb.ProgramExpression{Expression: &testpilotspb.ProgramExpression_Run{Run: &testpilotspb.RunRef{}}}
}
func addWorker(source *testpilotspb.Case, policy *Profile) {
	source.Program.Environment = append(source.Program.Environment, &testpilotspb.EnvironmentDefinition{BindingId: "namespace"}, &testpilotspb.EnvironmentDefinition{BindingId: "queue"})
	policy.EnvironmentBindings = append(policy.EnvironmentBindings, EnvironmentBinding{ID: "namespace", Value: "namespace"}, EnvironmentBinding{ID: "queue", Value: "queue"})
	source.Program.Roles = append(source.Program.Roles, &testpilotspb.RoleDefinition{RoleId: "worker", Kind: testpilotspb.ROLE_KIND_WORKER, NamespaceBindingId: "namespace"}, &testpilotspb.RoleDefinition{RoleId: "queue", Kind: testpilotspb.ROLE_KIND_TASK_QUEUE, NamespaceBindingId: "namespace", ResourceBindingId: "queue"})
	source.Program.Entrypoints = append(source.Program.Entrypoints, &testpilotspb.EntrypointDefinition{EntrypointId: "workflow", Activation: &testpilotspb.EntrypointDefinition_Workflow{Workflow: &testpilotspb.WorkflowActivation{WorkflowType: "flow", WorkerRoleId: "worker", TaskQueueRoleId: "queue"}}})
}
func TestPrepareRejectsStructuralAndPolicyErrors(t *testing.T) {
	for name, mutate := range map[string]func(*testpilotspb.Case, *Profile){
		"version":          func(c *testpilotspb.Case, _ *Profile) { c.Version.Major = 2 },
		"case id":          func(c *testpilotspb.Case, _ *Profile) { c.CaseId = " bad" },
		"missing contract": func(c *testpilotspb.Case, _ *Profile) { c.Contract = nil },
		"unknown field":    func(c *testpilotspb.Case, _ *Profile) { c.Program.ProtoReflect().SetUnknown([]byte{0x80, 0x06, 1}) },
		"duplicate entry": func(c *testpilotspb.Case, _ *Profile) {
			c.Program.Entrypoints = append(c.Program.Entrypoints, proto.CloneOf(c.Program.Entrypoints[0]))
		},
		"duplicate node": func(c *testpilotspb.Case, _ *Profile) {
			g := c.Program.Entrypoints[0]
			g.Instructions = append(g.Instructions, proto.CloneOf(g.Instructions[0]))
		},
		"cycle": func(c *testpilotspb.Case, _ *Profile) {
			c.Program.Entrypoints[0].Instructions[0].Dependencies = []*testpilotspb.InstructionRef{{EntrypointId: "controller", InstructionId: "call"}}
		},
		"cross entry": func(c *testpilotspb.Case, _ *Profile) {
			c.Program.Entrypoints[0].Instructions[0].Dependencies = []*testpilotspb.InstructionRef{{EntrypointId: "cleanup", InstructionId: "call"}}
		},
		"missing dependency": func(c *testpilotspb.Case, _ *Profile) {
			c.Program.Entrypoints[0].Instructions[0].Dependencies = []*testpilotspb.InstructionRef{{EntrypointId: "controller", InstructionId: "missing"}}
		},
		"binding": func(c *testpilotspb.Case, _ *Profile) { c.Program.Entrypoints[0].Activation = nil },
		"role":    func(c *testpilotspb.Case, _ *Profile) { c.Program.Roles[0].Kind = testpilotspb.ROLE_KIND_WORKER },
		"method": func(c *testpilotspb.Case, _ *Profile) {
			c.Program.Entrypoints[0].Instructions[0].Instruction.GetInvokeRpc().Method = "/example.Service/Missing"
		},
		"authorization":    func(_ *testpilotspb.Case, p *Profile) { p.Roles[0].Methods = nil },
		"capability":       func(_ *testpilotspb.Case, p *Profile) { p.Opcodes = nil },
		"catalog identity": func(_ *testpilotspb.Case, p *Profile) { p.CatalogIdentity = "other" },
		"node timeout": func(c *testpilotspb.Case, _ *Profile) {
			c.Program.Entrypoints[0].Instructions[0].Limits.TimeoutMilliseconds = 0
		},
		"attempt bound": func(c *testpilotspb.Case, _ *Profile) {
			c.Program.Entrypoints[0].Instructions[0].Limits.MaxAttempts = 33
		},
		"response bound": func(c *testpilotspb.Case, _ *Profile) {
			c.Program.Entrypoints[0].Instructions[0].Limits.MaxResponseBytes = 4097
		},
		"event bound": func(c *testpilotspb.Case, _ *Profile) {
			c.Program.Entrypoints[0].Instructions[0].Limits.MaxEmittedEvents = 257
		},
		"rpc raw outcome": func(c *testpilotspb.Case, _ *Profile) {
			c.Program.Entrypoints[0].Instructions[0].Outcome.Fields = append(c.Program.Entrypoints[0].Instructions[0].Outcome.Fields, &testpilotspb.OutcomeFieldDefinition{Field: testpilotspb.INSTRUCTION_OUTCOME_FIELD_VALUE, Type: scalar(testpilotspb.SCALAR_KIND_TEXT)})
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
	node.Instruction.GetInvokeRpc().RequestAssignments = []*testpilotspb.RequestAssignment{{
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
	addWorker(c, &policy)
	c.Program.Entrypoints[1].Instructions = []*testpilotspb.InstructionDefinition{{
		InstructionId: "finish",
		Instruction: &testpilotspb.Instruction{Instruction: &testpilotspb.Instruction_Finish{Finish: &testpilotspb.Finish{
			Result: runIDExpression(),
		}}},
		Outcome: statusSchema(),
		Limits:  &testpilotspb.InstructionLimits{TimeoutMilliseconds: 1000, MaxAttempts: 1, MaxEmittedEvents: 1, MaxResponseBytes: 4096},
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
			addWorker(c, &p)
			node := c.Program.Entrypoints[0].Instructions[0]
			node.Limits.MaxAttempts = test.local
			c.Program.Limits.MaxAttempts = test.global
			c.Program.Limits.MaxActivations = test.ceiling
			node.ActivationReservations = []*testpilotspb.ActivationReservationDefinition{{EntrypointId: "workflow", Count: test.count}}
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
	c.Program.Slots = []*testpilotspb.SlotDefinition{valueSlot("result", scalar(testpilotspb.SCALAR_KIND_TEXT))}
	c.Program.Observations = []*testpilotspb.ObservationDefinition{{ObservationId: "text", Type: scalar(testpilotspb.SCALAR_KIND_TEXT)}}
	producer := c.Program.Entrypoints[0].Instructions[0]
	producer.Instruction.GetInvokeRpc().ResponseProjections = []*testpilotspb.ResponseProjection{{Source: field("text"), Kind: testpilotspb.PROJECTION_KIND_ONE, Targets: []*testpilotspb.ProjectionTarget{{Target: &testpilotspb.ProjectionTarget_SlotId{SlotId: "result"}}, {Target: &testpilotspb.ProjectionTarget_ObservationId{ObservationId: "text"}}}}}
	consumer := rpcNode("consume")
	consumer.Dependencies = []*testpilotspb.InstructionRef{{EntrypointId: "controller", InstructionId: "call"}}
	consumer.Guard = succeeded("controller", "call")
	consumer.Instruction.GetInvokeRpc().RequestAssignments = []*testpilotspb.RequestAssignment{{Target: field("text"), Value: slot("result")}}
	c.Program.Entrypoints[0].Instructions = append(c.Program.Entrypoints[0].Instructions, consumer)
	prepared, err := Prepare(c, catalog, p)
	require.NoError(t, err)
	for name, mutate := range map[string]func(*testpilotspb.Case){"unguarded": func(s *testpilotspb.Case) { s.Program.Entrypoints[0].Instructions[1].Guard = nil }, "missing dependency": func(s *testpilotspb.Case) { s.Program.Entrypoints[0].Instructions[1].Dependencies = nil }, "second writer": func(s *testpilotspb.Case) {
		s.Program.Entrypoints[0].Instructions[1].Instruction.GetInvokeRpc().ResponseProjections = proto.CloneOf(producer.Instruction).GetInvokeRpc().ResponseProjections
	}, "assignment overlap": func(s *testpilotspb.Case) {
		rpc := s.Program.Entrypoints[0].Instructions[1].Instruction.GetInvokeRpc()
		rpc.RequestAssignments = append(rpc.RequestAssignments, proto.CloneOf(rpc.RequestAssignments[0]))
	}, "crossed cardinality": func(s *testpilotspb.Case) {
		s.Program.Entrypoints[0].Instructions[0].Instruction.GetInvokeRpc().ResponseProjections[0].Kind = testpilotspb.PROJECTION_KIND_EMIT_EACH
	}, "undeclared outcome": func(s *testpilotspb.Case) { s.Program.Entrypoints[0].Instructions[0].Outcome.Fields = nil }} {
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
	for name, mutate := range map[string]func(*testpilotspb.Case){
		"missing target": func(c *testpilotspb.Case) {
			c.Program.Entrypoints[0].Instructions[0].ActivationReservations[0].EntrypointId = "missing"
		},
		"controller target": func(c *testpilotspb.Case) {
			c.Program.Entrypoints[0].Instructions[0].ActivationReservations[0].EntrypointId = "controller"
		},
		"wrong binding": func(c *testpilotspb.Case) {
			c.Program.Entrypoints[1].Activation = &testpilotspb.EntrypointDefinition_Controller{Controller: &testpilotspb.ControllerActivation{}}
		},
		"activity target": func(c *testpilotspb.Case) {
			g := c.Program.Entrypoints[1]
			g.Activation = &testpilotspb.EntrypointDefinition_Activity{Activity: &testpilotspb.ActivityActivation{ActivityType: "activity", WorkerRoleId: "worker", TaskQueueRoleId: "queue"}}
		},
		"duplicate target": func(c *testpilotspb.Case) {
			n := c.Program.Entrypoints[0].Instructions[0]
			n.ActivationReservations = append(n.ActivationReservations, proto.CloneOf(n.ActivationReservations[0]))
		},
		"worker reservation": func(c *testpilotspb.Case) {
			n := c.Program.Entrypoints[0].Instructions[0]
			worker := proto.CloneOf(n)
			worker.Instruction = &testpilotspb.Instruction{Instruction: &testpilotspb.Instruction_Finish{Finish: &testpilotspb.Finish{Result: textLiteral("done")}}}
			c.Program.Entrypoints[1].Instructions = []*testpilotspb.InstructionDefinition{worker}
		},
		"cleanup reservation": func(c *testpilotspb.Case) {
			c.Program.Cleanup.Instructions = []*testpilotspb.InstructionDefinition{proto.CloneOf(c.Program.Entrypoints[0].Instructions[0])}
		},
		"sum overflow": func(c *testpilotspb.Case) {
			other := proto.CloneOf(c.Program.Entrypoints[1])
			other.EntrypointId = "other"
			c.Program.Entrypoints = append(c.Program.Entrypoints, other)
			n := c.Program.Entrypoints[0].Instructions[0]
			n.ActivationReservations = []*testpilotspb.ActivationReservationDefinition{{EntrypointId: "workflow", Count: math.MaxInt64}, {EntrypointId: "other", Count: math.MaxInt64}}
		},
	} {
		t.Run(name, func(t *testing.T) {
			c, catalog, p := fixture(t)
			addWorker(c, &p)
			c.Program.Entrypoints[0].Instructions[0].ActivationReservations = []*testpilotspb.ActivationReservationDefinition{{EntrypointId: "workflow", Count: 1}}
			mutate(c)
			_, err := Prepare(c, catalog, p)
			require.Error(t, err)
		})
	}
	c, catalog, p := fixture(t)
	addWorker(c, &p)
	first := c.Program.Entrypoints[0].Instructions[0]
	first.Limits.MaxAttempts = 2
	first.ActivationReservations = []*testpilotspb.ActivationReservationDefinition{{EntrypointId: "workflow", Count: 5}}
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
func textLiteral(value string) *testpilotspb.ProgramExpression {
	return &testpilotspb.ProgramExpression{Expression: &testpilotspb.ProgramExpression_Literal{Literal: &testpilotspb.Value{Value: &testpilotspb.Value_Text{Text: value}}}}
}

func environment(id string) *testpilotspb.ProgramExpression {
	return &testpilotspb.ProgramExpression{Expression: &testpilotspb.ProgramExpression_Environment{Environment: &testpilotspb.EnvironmentRef{BindingId: id}}}
}

func TestPrepareResolvesClosedEnvironmentGraph(t *testing.T) {
	c, catalog, policy := fixture(t)
	c.Program.Environment = []*testpilotspb.EnvironmentDefinition{{BindingId: "namespace"}, {BindingId: "queue"}}
	c.Program.Roles = append(c.Program.Roles,
		&testpilotspb.RoleDefinition{RoleId: "worker", Kind: testpilotspb.ROLE_KIND_WORKER, NamespaceBindingId: "namespace"},
		&testpilotspb.RoleDefinition{RoleId: "queue", Kind: testpilotspb.ROLE_KIND_TASK_QUEUE, NamespaceBindingId: "namespace", ResourceBindingId: "queue"},
	)
	c.Program.Roles[0].ResourceBindingId = "queue"
	policy.EnvironmentBindings = []EnvironmentBinding{{ID: "namespace", Value: "namespace-a"}, {ID: "queue", Value: "queue-a"}, {ID: "unused", Value: "allowed"}}
	policy.EnvironmentFingerprint = "fingerprint"
	c.Program.Entrypoints[0].Instructions[0].Instruction.GetInvokeRpc().RequestAssignments = []*testpilotspb.RequestAssignment{{Target: field("text"), Value: environment("namespace")}}

	prepared, err := Prepare(c, catalog, policy)
	require.NoError(t, err)
	require.Equal(t, "namespace", prepared.graphs[0].nodes[0].assignments[0].environmentBindingID)
	require.Equal(t, "namespace-a", prepared.graphs[0].nodes[0].assignments[0].value.Literal().GetText())
	require.Equal(t, resolvedRole{ID: "endpoint", Kind: testpilotspb.ROLE_KIND_ENDPOINT, ResourceBindingID: "queue", Resource: "queue-a"}, prepared.roles["endpoint"])
	require.Equal(t, resolvedRole{ID: "queue", Kind: testpilotspb.ROLE_KIND_TASK_QUEUE, NamespaceBindingID: "namespace", Namespace: "namespace-a", ResourceBindingID: "queue", Resource: "queue-a"}, prepared.roles["queue"])
	require.NotEmpty(t, prepared.environmentFingerprint)
	store, err := newValueStore(prepared, "run")
	require.NoError(t, err)
	values, err := store.activate("controller", "activation")
	require.NoError(t, err)
	request, dispatched, _, err := values.request(t.Context(), Coordinate{RunID: "run", EntrypointID: "controller", ActivationID: "activation", InstructionID: "call", Attempt: 1}, prepared.graphs[0].runtimeWork)
	require.NoError(t, err)
	require.True(t, dispatched)
	textField := request.ProtoReflect().Descriptor().Fields().ByName("text")
	require.Equal(t, "namespace-a", request.ProtoReflect().Get(textField).String())

	c.Program.Environment[0].BindingId = "changed"
	c.Program.Roles[2].ResourceBindingId = "changed"
	policy.EnvironmentBindings[0].Value = "changed"
	require.Equal(t, "namespace", prepared.Snapshot().Environment[0].BindingId)
	require.Equal(t, "namespace", prepared.Snapshot().Entrypoints[0].Instructions[0].Instruction.GetInvokeRpc().RequestAssignments[0].Value.GetEnvironment().BindingId)
	require.Equal(t, "namespace-a", prepared.graphs[0].nodes[0].assignments[0].value.Literal().GetText())
	require.Equal(t, "queue-a", prepared.roles["queue"].Resource)
	require.Equal(t, "fingerprint", prepared.environmentFingerprint)
}

func TestPrepareEnvironmentVersionAndClosure(t *testing.T) {
	for name, mutate := range map[string]func(*testpilotspb.Case, *Profile){
		"unsupported 1.1": func(c *testpilotspb.Case, _ *Profile) { c.Version.Minor = 1 },
		"duplicate definition": func(c *testpilotspb.Case, p *Profile) {
			configureEnvironmentCase(c, p)
			c.Program.Environment = append(c.Program.Environment, &testpilotspb.EnvironmentDefinition{BindingId: "binding"})
		},
		"unused definition": func(c *testpilotspb.Case, p *Profile) {
			configureEnvironmentCase(c, p)
			c.Program.Environment = append(c.Program.Environment, &testpilotspb.EnvironmentDefinition{BindingId: "unused"})
			p.EnvironmentBindings = append(p.EnvironmentBindings, EnvironmentBinding{ID: "unused", Value: "value"})
		},
		"undeclared reference": func(c *testpilotspb.Case, p *Profile) {
			configureEnvironmentCase(c, p)
			c.Program.Entrypoints[0].Instructions[0].Instruction.GetInvokeRpc().RequestAssignments[0].Value = environment("missing")
		},
		"missing profile value": func(c *testpilotspb.Case, p *Profile) {
			configureEnvironmentCase(c, p)
			p.EnvironmentBindings = nil
		},
		"nested reference": func(c *testpilotspb.Case, p *Profile) {
			configureEnvironmentCase(c, p)
			c.Program.Entrypoints[0].Instructions[0].Guard = &testpilotspb.ProgramExpression{Expression: &testpilotspb.ProgramExpression_Equals{Equals: &testpilotspb.ProgramEqualsExpression{Left: environment("binding"), Right: textLiteral("value")}}}
		},
		"non-text destination": func(c *testpilotspb.Case, p *Profile) {
			configureEnvironmentCase(c, p)
			c.Program.Entrypoints[0].Instructions[0].Instruction.GetInvokeRpc().RequestAssignments[0].Target = field("items")
		},
		"resolved byte overflow": func(c *testpilotspb.Case, p *Profile) {
			configureEnvironmentCase(c, p)
			c.Program.Limits.MaxRequestBytes = 8
		},
		"incompatible endpoint namespace": func(c *testpilotspb.Case, p *Profile) {
			configureEnvironmentCase(c, p)
			c.Program.Roles[0].NamespaceBindingId = "binding"
		},
		"1.1 worker without namespace": func(c *testpilotspb.Case, p *Profile) {
			configureEnvironmentCase(c, p)
			c.Program.Roles = append(c.Program.Roles, &testpilotspb.RoleDefinition{RoleId: "worker", Kind: testpilotspb.ROLE_KIND_WORKER})
		},
		"1.1 worker with resource": func(c *testpilotspb.Case, p *Profile) {
			configureEnvironmentCase(c, p)
			c.Program.Roles = append(c.Program.Roles, &testpilotspb.RoleDefinition{RoleId: "worker", Kind: testpilotspb.ROLE_KIND_WORKER, NamespaceBindingId: "binding", ResourceBindingId: "binding"})
		},
		"1.1 task queue without namespace": func(c *testpilotspb.Case, p *Profile) {
			configureEnvironmentCase(c, p)
			c.Program.Roles = append(c.Program.Roles, &testpilotspb.RoleDefinition{RoleId: "queue", Kind: testpilotspb.ROLE_KIND_TASK_QUEUE, ResourceBindingId: "binding"})
		},
		"1.1 task queue without resource": func(c *testpilotspb.Case, p *Profile) {
			configureEnvironmentCase(c, p)
			c.Program.Roles = append(c.Program.Roles, &testpilotspb.RoleDefinition{RoleId: "queue", Kind: testpilotspb.ROLE_KIND_TASK_QUEUE, NamespaceBindingId: "binding"})
		},
		"1.1 participant with resource": func(c *testpilotspb.Case, p *Profile) {
			configureEnvironmentCase(c, p)
			p.Roles = append(p.Roles, RolePolicy{ID: "participant", Kind: testpilotspb.ROLE_KIND_PARTICIPANT})
			c.Program.Roles = append(c.Program.Roles, &testpilotspb.RoleDefinition{RoleId: "participant", Kind: testpilotspb.ROLE_KIND_PARTICIPANT, ResourceBindingId: "binding"})
		},
	} {
		t.Run(name, func(t *testing.T) {
			c, catalog, policy := fixture(t)
			mutate(c, &policy)
			_, err := Prepare(c, catalog, policy)
			require.Error(t, err)
		})
	}
}

func TestPrepareRejectsMalformedEnvironmentPolicy(t *testing.T) {
	for name, bindings := range map[string][]EnvironmentBinding{
		"invalid id":    {{ID: "bad id", Value: "value"}},
		"invalid value": {{ID: "id", Value: string([]byte{0xff})}},
		"empty value":   {{ID: "id"}},
		"duplicate":     {{ID: "id", Value: "one"}, {ID: "id", Value: "two"}},
	} {
		t.Run(name, func(t *testing.T) {
			c, catalog, policy := fixture(t)
			policy.EnvironmentBindings = bindings
			_, err := Prepare(c, catalog, policy)
			require.Error(t, err)
		})
	}

	c, catalog, policy := fixture(t)
	policy.EnvironmentBindings = make([]EnvironmentBinding, 10001)
	_, err := Prepare(c, catalog, policy)
	require.Error(t, err)

	c, catalog, policy = fixture(t)
	policy.Limits.MaxRequestBytes = 8
	c.Program.Limits.MaxRequestBytes = 8
	policy.EnvironmentBindings = []EnvironmentBinding{{ID: "id", Value: "1234567"}}
	_, err = Prepare(c, catalog, policy)
	require.Error(t, err)
}

func configureEnvironmentCase(c *testpilotspb.Case, policy *Profile) {
	c.Program.Environment = []*testpilotspb.EnvironmentDefinition{{BindingId: "binding"}}
	policy.EnvironmentBindings = []EnvironmentBinding{{ID: "binding", Value: "value"}}
	c.Program.Entrypoints[0].Instructions[0].Instruction.GetInvokeRpc().RequestAssignments = []*testpilotspb.RequestAssignment{{Target: field("text"), Value: environment("binding")}}
}

func TestConcurrentEnvironmentPreparationsResolveIndependently(t *testing.T) {
	c, catalog, base := fixture(t)
	configureEnvironmentCase(c, &base)
	for i := 0; i < 8; i++ {
		i := i
		t.Run(fmt.Sprint(i), func(t *testing.T) {
			t.Parallel()
			policy := base
			value := fmt.Sprintf("environment-%d", i)
			policy.EnvironmentBindings = []EnvironmentBinding{{ID: "binding", Value: value}}
			prepared, err := Prepare(c, catalog, policy)
			require.NoError(t, err)
			store, err := newValueStore(prepared, "run")
			require.NoError(t, err)
			values, err := store.activate("controller", "activation")
			require.NoError(t, err)
			request, dispatched, _, err := values.request(t.Context(), Coordinate{RunID: "run", EntrypointID: "controller", ActivationID: "activation", InstructionID: "call", Attempt: 1}, prepared.graphs[0].runtimeWork)
			require.NoError(t, err)
			require.True(t, dispatched)
			textField := request.ProtoReflect().Descriptor().Fields().ByName("text")
			require.Equal(t, value, request.ProtoReflect().Get(textField).String())
		})
	}
}

func TestConcurrentEnvironmentRequestsUsePreparedSnapshot(t *testing.T) {
	c, catalog, policy := fixture(t)
	configureEnvironmentCase(c, &policy)
	prepared, err := Prepare(c, catalog, policy)
	require.NoError(t, err)
	c.Program.Environment[0].BindingId = "changed"
	policy.EnvironmentBindings[0].Value = "changed"

	for i := 0; i < 8; i++ {
		i := i
		t.Run(fmt.Sprint(i), func(t *testing.T) {
			t.Parallel()
			store, err := newValueStore(prepared, fmt.Sprintf("run-%d", i))
			require.NoError(t, err)
			values, err := store.activate("controller", "activation")
			require.NoError(t, err)
			request, dispatched, _, err := values.request(t.Context(), Coordinate{RunID: fmt.Sprintf("run-%d", i), EntrypointID: "controller", ActivationID: "activation", InstructionID: "call", Attempt: 1}, prepared.graphs[0].runtimeWork)
			require.NoError(t, err)
			require.True(t, dispatched)
			textField := request.ProtoReflect().Descriptor().Fields().ByName("text")
			require.Equal(t, "value", request.ProtoReflect().Get(textField).String())
		})
	}
}

func TestInstructionContextMatrix(t *testing.T) {
	for _, context := range []testpilotspb.EntrypointKind{testpilotspb.ENTRYPOINT_KIND_CONTROLLER, testpilotspb.ENTRYPOINT_KIND_WORKFLOW, testpilotspb.ENTRYPOINT_KIND_ACTIVITY, testpilotspb.ENTRYPOINT_KIND_NEXUS_HANDLER} {
		for _, test := range []struct {
			name        string
			instruction *testpilotspb.Instruction
			expected    testpilotspb.EntrypointKind
		}{
			{"rpc", rpcNode("call").Instruction, testpilotspb.ENTRYPOINT_KIND_CONTROLLER},
			{"await slot", &testpilotspb.Instruction{Instruction: &testpilotspb.Instruction_AwaitSlot{AwaitSlot: &testpilotspb.AwaitSlot{SlotId: "value"}}}, testpilotspb.ENTRYPOINT_KIND_CONTROLLER},
			{"complete", &testpilotspb.Instruction{Instruction: &testpilotspb.Instruction_CompleteNexusOperation{CompleteNexusOperation: &testpilotspb.CompleteNexusOperation{CapabilitySlotId: "capability", Result: textLiteral("done")}}}, testpilotspb.ENTRYPOINT_KIND_CONTROLLER},
			{"start", &testpilotspb.Instruction{Instruction: &testpilotspb.Instruction_StartNexusOperation{StartNexusOperation: &testpilotspb.StartNexusOperation{EndpointRoleId: "endpoint", Service: "service", Operation: "operation", Input: textLiteral("input")}}}, testpilotspb.ENTRYPOINT_KIND_WORKFLOW},
			{"await", &testpilotspb.Instruction{Instruction: &testpilotspb.Instruction_AwaitInstruction{AwaitInstruction: &testpilotspb.AwaitInstruction{Instruction: &testpilotspb.InstructionRef{EntrypointId: "workflow", InstructionId: "prior"}}}}, testpilotspb.ENTRYPOINT_KIND_WORKFLOW},
			{"finish", &testpilotspb.Instruction{Instruction: &testpilotspb.Instruction_Finish{Finish: &testpilotspb.Finish{Result: textLiteral("done")}}}, testpilotspb.ENTRYPOINT_KIND_WORKFLOW},
			{"respond", &testpilotspb.Instruction{Instruction: &testpilotspb.Instruction_RespondNexus{RespondNexus: &testpilotspb.RespondNexus{Kind: testpilotspb.NEXUS_RESPONSE_KIND_SYNCHRONOUS, Result: textLiteral("done")}}}, testpilotspb.ENTRYPOINT_KIND_NEXUS_HANDLER},
		} {
			t.Run(context.String()+"/"+test.name, func(t *testing.T) {
				if context == test.expected {
					return
				}
				c, catalog, p := fixture(t)
				g := c.Program.Entrypoints[0]
				g.Instructions[0].Instruction = test.instruction
				switch context {
				case testpilotspb.ENTRYPOINT_KIND_WORKFLOW:
					g.Activation = &testpilotspb.EntrypointDefinition_Workflow{Workflow: &testpilotspb.WorkflowActivation{WorkflowType: "flow", WorkerRoleId: "worker", TaskQueueRoleId: "queue"}}
				case testpilotspb.ENTRYPOINT_KIND_ACTIVITY:
					g.Activation = &testpilotspb.EntrypointDefinition_Activity{Activity: &testpilotspb.ActivityActivation{ActivityType: "activity", WorkerRoleId: "worker", TaskQueueRoleId: "queue"}}
				case testpilotspb.ENTRYPOINT_KIND_NEXUS_HANDLER:
					g.Activation = &testpilotspb.EntrypointDefinition_NexusHandler{NexusHandler: &testpilotspb.NexusHandlerActivation{Service: "service", Operation: "operation", WorkerRoleId: "worker", TaskQueueRoleId: "queue"}}
				default:
				}
				c.Program.Roles = append(c.Program.Roles, &testpilotspb.RoleDefinition{RoleId: "worker", Kind: testpilotspb.ROLE_KIND_WORKER}, &testpilotspb.RoleDefinition{RoleId: "queue", Kind: testpilotspb.ROLE_KIND_TASK_QUEUE})
				_, err := Prepare(c, catalog, p)
				require.Error(t, err)
			})
		}
	}
}

func capabilityFixture(t *testing.T) (*testpilotspb.Case, *ir.Catalog, Profile) {
	t.Helper()
	c, catalog, p := fixture(t)
	addWorker(c, &p)
	c.Program.Slots = []*testpilotspb.SlotDefinition{capabilitySlot("capability")}
	wait := rpcNode("ready")
	wait.Instruction = &testpilotspb.Instruction{Instruction: &testpilotspb.Instruction_AwaitSlot{AwaitSlot: &testpilotspb.AwaitSlot{SlotId: "capability"}}}
	complete := rpcNode("complete")
	complete.Dependencies = []*testpilotspb.InstructionRef{{EntrypointId: "controller", InstructionId: "ready"}}
	complete.Guard = succeeded("controller", "ready")
	complete.Instruction = &testpilotspb.Instruction{Instruction: &testpilotspb.Instruction_CompleteNexusOperation{CompleteNexusOperation: &testpilotspb.CompleteNexusOperation{CapabilitySlotId: "capability", Result: textLiteral("done")}}}
	c.Program.Entrypoints[0].Instructions = append(c.Program.Entrypoints[0].Instructions, wait, complete)
	handler := rpcNode("respond")
	handler.Instruction = &testpilotspb.Instruction{Instruction: &testpilotspb.Instruction_RespondNexus{RespondNexus: &testpilotspb.RespondNexus{Kind: testpilotspb.NEXUS_RESPONSE_KIND_ASYNCHRONOUS, CapabilitySlotId: "capability", Result: textLiteral("accepted")}}}
	c.Program.Entrypoints = append(c.Program.Entrypoints, &testpilotspb.EntrypointDefinition{EntrypointId: "handler", Activation: &testpilotspb.EntrypointDefinition_NexusHandler{NexusHandler: &testpilotspb.NexusHandlerActivation{Service: "service", Operation: "operation", WorkerRoleId: "worker", TaskQueueRoleId: "queue"}}, Instructions: []*testpilotspb.InstructionDefinition{handler}})
	start := rpcNode("start")
	start.Instruction = &testpilotspb.Instruction{Instruction: &testpilotspb.Instruction_StartNexusOperation{StartNexusOperation: &testpilotspb.StartNexusOperation{EndpointRoleId: "endpoint", Service: "service", Operation: "operation", Input: textLiteral("input")}}}
	await := rpcNode("await")
	await.Dependencies = []*testpilotspb.InstructionRef{{EntrypointId: "workflow", InstructionId: "start"}}
	await.Instruction = &testpilotspb.Instruction{Instruction: &testpilotspb.Instruction_AwaitInstruction{AwaitInstruction: &testpilotspb.AwaitInstruction{Instruction: &testpilotspb.InstructionRef{EntrypointId: "workflow", InstructionId: "start"}}}}
	await.Outcome.Fields = append(await.Outcome.Fields, &testpilotspb.OutcomeFieldDefinition{Field: testpilotspb.INSTRUCTION_OUTCOME_FIELD_VALUE, Type: scalar(testpilotspb.SCALAR_KIND_TEXT)})
	finish := rpcNode("finish")
	finish.Dependencies = []*testpilotspb.InstructionRef{{EntrypointId: "workflow", InstructionId: "await"}}
	finish.Guard = succeeded("workflow", "await")
	finish.Instruction = &testpilotspb.Instruction{Instruction: &testpilotspb.Instruction_Finish{Finish: &testpilotspb.Finish{Result: &testpilotspb.ProgramExpression{Expression: &testpilotspb.ProgramExpression_Outcome{Outcome: &testpilotspb.InstructionOutcomeRef{Instruction: &testpilotspb.InstructionRef{EntrypointId: "workflow", InstructionId: "await"}, Field: testpilotspb.INSTRUCTION_OUTCOME_FIELD_VALUE}}}}}}
	c.Program.Entrypoints[1].Instructions = []*testpilotspb.InstructionDefinition{start, await, finish}
	c.Program.Entrypoints[0].Instructions[0].ActivationReservations = []*testpilotspb.ActivationReservationDefinition{{EntrypointId: "workflow", Count: 1}, {EntrypointId: "handler", Count: 1}}
	return c, catalog, p
}
func TestOpaqueReadinessAndSDKPreparedPlans(t *testing.T) {
	c, catalog, p := capabilityFixture(t)
	prepared, err := Prepare(c, catalog, p)
	require.NoError(t, err)
	for name, mutate := range map[string]func(*testpilotspb.Case){
		"inspect capability": func(s *testpilotspb.Case) {
			s.Program.Entrypoints[0].Instructions[2].Guard = present(slot("capability"))
		},
		"consume without readiness": func(s *testpilotspb.Case) { s.Program.Entrypoints[0].Instructions[2].Guard = nil },
		"missing capability writer": func(s *testpilotspb.Case) {
			s.Program.Entrypoints = s.Program.Entrypoints[:2]
			s.Program.Entrypoints[0].Instructions[0].ActivationReservations = s.Program.Entrypoints[0].Instructions[0].ActivationReservations[:1]
		},
		"capability projection": func(s *testpilotspb.Case) {
			s.Program.Entrypoints[0].Instructions[0].Instruction.GetInvokeRpc().ResponseProjections = []*testpilotspb.ResponseProjection{{Source: field("text"), Kind: testpilotspb.PROJECTION_KIND_ONE, Targets: []*testpilotspb.ProjectionTarget{{Target: &testpilotspb.ProjectionTarget_SlotId{SlotId: "capability"}}}}}
		},
		"SDK value without success": func(s *testpilotspb.Case) { s.Program.Entrypoints[1].Instructions[2].Guard = nil },
		"worker RPC": func(s *testpilotspb.Case) {
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
	instructions[0].Input().Literal().Value = &testpilotspb.Value_Text{Text: "changed"}
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
	second.Dependencies = []*testpilotspb.InstructionRef{{EntrypointId: "cleanup", InstructionId: "release"}}
	second.Guard = succeeded("cleanup", "release")
	c.Program.Cleanup.Instructions = []*testpilotspb.InstructionDefinition{first, second}
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
	c.Program.Slots = []*testpilotspb.SlotDefinition{valueSlot("value", scalar(testpilotspb.SCALAR_KIND_TEXT))}
	c.Program.Entrypoints[0].Instructions[0].Instruction.GetInvokeRpc().ResponseProjections = []*testpilotspb.ResponseProjection{{Source: field("text"), Kind: testpilotspb.PROJECTION_KIND_ONE, Targets: []*testpilotspb.ProjectionTarget{{Target: &testpilotspb.ProjectionTarget_SlotId{SlotId: "value"}}}}}
	other := proto.CloneOf(c.Program.Entrypoints[0])
	other.EntrypointId = "other"
	other.Instructions[0].Instruction.GetInvokeRpc().ResponseProjections = nil
	other.Instructions[0].Guard = present(slot("value"))
	other.Instructions[0].Instruction.GetInvokeRpc().RequestAssignments = []*testpilotspb.RequestAssignment{{Target: field("text"), Value: slot("value")}}
	c.Program.Entrypoints = append(c.Program.Entrypoints, other)
	prepared, err := Prepare(c, catalog, p)
	require.NoError(t, err)
	c.Program.Cleanup.Instructions = []*testpilotspb.InstructionDefinition{proto.CloneOf(other.Instructions[0])}
	_, err = Prepare(c, catalog, p)
	require.NoError(t, err)
	addWorker(c, &p)
	worker := c.Program.Entrypoints[len(c.Program.Entrypoints)-1]
	finish := rpcNode("finish")
	finish.Guard = present(slot("value"))
	finish.Instruction = &testpilotspb.Instruction{Instruction: &testpilotspb.Instruction_Finish{Finish: &testpilotspb.Finish{Result: slot("value")}}}
	worker.Instructions = []*testpilotspb.InstructionDefinition{finish}
	_, err = Prepare(c, catalog, p)
	require.Error(t, err)
	for i := 0; i < 8; i++ {
		t.Run(fmt.Sprint(i), func(t *testing.T) {
			t.Parallel()
			for j := 0; j < 10; j++ {
				plans := prepared.Entrypoints()
				plans[0].Instructions()[0].Projections()[0].Sinks[0].Target = &testpilotspb.ProjectionTarget_SlotId{SlotId: "changed"}
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
	expression := &testpilotspb.ProgramExpression{}
	expression.Expression = &testpilotspb.ProgramExpression_Negation{Negation: &testpilotspb.ProgramNotExpression{Operand: expression}}
	c.Program.Entrypoints[0].Instructions[0].Guard = expression
	_, err := Prepare(c, catalog, p)
	require.Error(t, err)
	c, catalog, p = fixture(t)
	c.Program.Entrypoints[0].Instructions[0].Instruction.Instruction = (*testpilotspb.Instruction_InvokeRpc)(nil)
	_, err = Prepare(c, catalog, p)
	require.Error(t, err)
}

func TestStructuralCountsAndProjectionFanout(t *testing.T) {
	for name, mutate := range map[string]func(*testpilotspb.Case, *Profile){
		"entrypoint count": func(c *testpilotspb.Case, p *Profile) { addWorker(c, p); c.Program.Limits.MaxEntrypoints = 1 },
		"node count": func(c *testpilotspb.Case, _ *Profile) {
			c.Program.Entrypoints[0].Instructions = append(c.Program.Entrypoints[0].Instructions, rpcNode("other"))
			c.Program.Limits.MaxNodes = 1
		},
		"edge count": func(c *testpilotspb.Case, _ *Profile) {
			last := rpcNode("last")
			last.Dependencies = []*testpilotspb.InstructionRef{{EntrypointId: "controller", InstructionId: "call"}, {EntrypointId: "controller", InstructionId: "other"}}
			c.Program.Entrypoints[0].Instructions = append(c.Program.Entrypoints[0].Instructions, rpcNode("other"), last)
			c.Program.Limits.MaxEdges = 1
		},
		"controller activation count": func(c *testpilotspb.Case, _ *Profile) {
			other := proto.CloneOf(c.Program.Entrypoints[0])
			other.EntrypointId = "other"
			c.Program.Entrypoints = append(c.Program.Entrypoints, other)
			c.Program.Limits.MaxActivations = 1
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
	c.Program.Observations = []*testpilotspb.ObservationDefinition{{ObservationId: "item", Type: scalar(testpilotspb.SCALAR_KIND_TEXT)}}
	n := c.Program.Entrypoints[0].Instructions[0]
	n.Limits.MaxEmittedEvents = 128
	n.Instruction.GetInvokeRpc().ResponseProjections = []*testpilotspb.ResponseProjection{{Source: field("items"), Kind: testpilotspb.PROJECTION_KIND_EMIT_EACH, Targets: []*testpilotspb.ProjectionTarget{{Target: &testpilotspb.ProjectionTarget_ObservationId{ObservationId: "item"}}}}}
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
	typ := &testpilotspb.ValueType{Shape: &testpilotspb.ValueType_Singular{Singular: &testpilotspb.SingularType{Type: &testpilotspb.SingularType_Message{Message: &testpilotspb.NamedType{ProtobufType: "example.Payload"}}}}}
	c.Program.Slots = []*testpilotspb.SlotDefinition{valueSlot("request", typ)}
	producer := c.Program.Entrypoints[0].Instructions[0]
	producer.Instruction.GetInvokeRpc().ResponseProjections = []*testpilotspb.ResponseProjection{{Source: &testpilotspb.FieldPath{}, Kind: testpilotspb.PROJECTION_KIND_ONE, Targets: []*testpilotspb.ProjectionTarget{{Target: &testpilotspb.ProjectionTarget_SlotId{SlotId: "request"}}}}}
	consumer := rpcNode("copy")
	consumer.Dependencies = []*testpilotspb.InstructionRef{{EntrypointId: "controller", InstructionId: "call"}}
	consumer.Guard = succeeded("controller", "call")
	consumer.Instruction.GetInvokeRpc().RequestAssignments = []*testpilotspb.RequestAssignment{{Target: &testpilotspb.FieldPath{}, Value: slot("request")}}
	c.Program.Entrypoints[0].Instructions = append(c.Program.Entrypoints[0].Instructions, consumer)
	_, err := Prepare(c, catalog, p)
	require.NoError(t, err)
	consumer.Instruction.GetInvokeRpc().RequestAssignments = append(consumer.Instruction.GetInvokeRpc().RequestAssignments, &testpilotspb.RequestAssignment{Target: field("text"), Value: textLiteral("conflict")})
	_, err = Prepare(c, catalog, p)
	require.Error(t, err)
}

func TestAwaitRequiresNexusStart(t *testing.T) {
	for _, target := range []string{"start", "await", "finish"} {
		t.Run(target, func(t *testing.T) {
			c, catalog, p := capabilityFixture(t)
			g := c.Program.Entrypoints[1]
			n := rpcNode("second_await")
			n.Dependencies = []*testpilotspb.InstructionRef{{EntrypointId: "workflow", InstructionId: target}}
			n.Instruction = &testpilotspb.Instruction{Instruction: &testpilotspb.Instruction_AwaitInstruction{AwaitInstruction: &testpilotspb.AwaitInstruction{Instruction: proto.CloneOf(n.Dependencies[0])}}}
			g.Instructions = append([]*testpilotspb.InstructionDefinition{n}, g.Instructions...)
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
