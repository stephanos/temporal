package execution

import (
	"fmt"
	"math"
	"testing"

	"github.com/stretchr/testify/require"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/testpilot/contract"
	"go.temporal.io/server/common/testing/testpilot/internal/ir"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
)

func fixture(t *testing.T) (*testpilotspb.Case, *ir.Catalog, Profile) {
	t.Helper()
	catalog, err := ir.NewCatalog(&descriptorpb.FileDescriptorSet{File: []*descriptorpb.FileDescriptorProto{{Name: proto.String("admission.proto"), Package: proto.String("example"), Syntax: proto.String("proto3"), MessageType: []*descriptorpb.DescriptorProto{{Name: proto.String("Payload"), Field: []*descriptorpb.FieldDescriptorProto{{Name: proto.String("text"), Number: proto.Int32(1), Type: descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(), Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum()}, {Name: proto.String("items"), Number: proto.Int32(2), Type: descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(), Label: descriptorpb.FieldDescriptorProto_LABEL_REPEATED.Enum()}}}}, Service: []*descriptorpb.ServiceDescriptorProto{{Name: proto.String("Service"), Method: []*descriptorpb.MethodDescriptorProto{{Name: proto.String("Call"), InputType: proto.String(".example.Payload"), OutputType: proto.String(".example.Payload")}, {Name: proto.String("Stream"), InputType: proto.String(".example.Payload"), OutputType: proto.String(".example.Payload"), ServerStreaming: proto.Bool(true)}}}}}}})
	require.NoError(t, err)
	limits := &testpilotspb.ProgramLimits{MaxEntrypoints: 8, MaxNodes: 32, MaxEdges: 64, MaxActivations: 64, MaxAttempts: 32, MaxRunEvents: 256, MaxExpressionDepth: 16, MaxPathFanout: 128, MaxRequestBytes: 4096, MaxResponseBytes: 4096, MaxTotalDurationMilliseconds: 30000, MaxCleanupDurationMilliseconds: 5000, MaxInstructionEmittedEvents: 8, MaxInstructionResponseBytes: 4096}
	policy := Profile{Identity: "host", CatalogIdentity: catalog.Identity(), Roles: []contract.RolePolicy{{ID: "endpoint", Kind: testpilotspb.ROLE_KIND_ENDPOINT, Methods: []string{"/example.Service/Call"}, ReservationCarriers: []contract.ReservationCarrierPolicy{{Method: "/example.Service/Call", Shapes: []contract.ReservationCarrierShape{{Kind: contract.WorkflowEntrypoint, MaximumCount: 32}, {Kind: contract.NexusHandlerEntrypoint, MaximumCount: 32}}}}}, {ID: "worker", Kind: testpilotspb.ROLE_KIND_WORKER}, {ID: "queue", Kind: testpilotspb.ROLE_KIND_TASK_QUEUE}}, Opcodes: []contract.Opcode{contract.InvokeRPC, contract.AwaitSlot, contract.CompleteNexusOperation, contract.StartNexusOperation, contract.Await, contract.Finish, contract.RespondNexus}, Limits: proto.CloneOf(limits)}
	source := &testpilotspb.Case{Version: &testpilotspb.FormatVersion{Major: 1}, CaseId: "case", Program: &testpilotspb.Program{ProgramId: "program", Roles: []*testpilotspb.Role{{RoleId: "endpoint", Kind: testpilotspb.ROLE_KIND_ENDPOINT}}, Entrypoints: []*testpilotspb.Entrypoint{{EntrypointId: "controller", Activation: &testpilotspb.Entrypoint_Controller{Controller: &testpilotspb.ControllerActivation{}}, Instructions: []*testpilotspb.InstructionNode{rpcNode("call")}}}, Cleanup: &testpilotspb.Cleanup{EntrypointId: "cleanup"}}, Contract: &testpilotspb.Contract{ContractId: "contract"}}
	return source, catalog, policy
}
func scalar(kind testpilotspb.ScalarKind) *testpilotspb.ValueType {
	return &testpilotspb.ValueType{Shape: &testpilotspb.ValueType_Singular{Singular: &testpilotspb.SingularType{Type: &testpilotspb.SingularType_Scalar{Scalar: &testpilotspb.ScalarType{Kind: kind}}}}}
}

func valueSlot(id string, typ *testpilotspb.ValueType) *testpilotspb.Slot {
	return &testpilotspb.Slot{SlotId: id, Content: &testpilotspb.Slot_Value{Value: typ}}
}

func capabilitySlot(id string) *testpilotspb.Slot {
	return &testpilotspb.Slot{SlotId: id, Content: &testpilotspb.Slot_OpaqueHandle{OpaqueHandle: &testpilotspb.OpaqueHandleType{}}}
}
func statusSchema() *testpilotspb.InstructionOutcomeDefinition {
	return &testpilotspb.InstructionOutcomeDefinition{Fields: []*testpilotspb.OutcomeFieldDefinition{{Field: testpilotspb.INSTRUCTION_OUTCOME_FIELD_STATUS, Type: &testpilotspb.ValueType{Shape: &testpilotspb.ValueType_Singular{Singular: &testpilotspb.SingularType{Type: &testpilotspb.SingularType_Enumeration{Enumeration: &testpilotspb.NamedType{ProtobufType: "temporal.server.api.testpilot.v1.InstructionOutcomeStatus"}}}}}}}}
}
func rpcNode(id string) *testpilotspb.InstructionNode {
	return &testpilotspb.InstructionNode{InstructionId: id, Instruction: &testpilotspb.Instruction{Instruction: &testpilotspb.Instruction_InvokeRpc{InvokeRpc: &testpilotspb.InvokeRpc{EndpointRoleId: "endpoint", Method: "/example.Service/Call"}}}, Outcome: statusSchema(), Limits: &testpilotspb.InstructionLimits{TimeoutMilliseconds: 1000, MaxAttempts: 1}}
}
func field(name string) *testpilotspb.FieldPath {
	return &testpilotspb.FieldPath{Segments: []*testpilotspb.FieldPathSegment{{Field: name}}}
}
func slot(id string) *testpilotspb.Expression {
	return &testpilotspb.Expression{Expression: &testpilotspb.Expression_Reference{Reference: &testpilotspb.Reference{Reference: &testpilotspb.Reference_SlotId{SlotId: id}}}}
}
func present(value *testpilotspb.Expression) *testpilotspb.Expression {
	return &testpilotspb.Expression{Expression: &testpilotspb.Expression_Present{Present: &testpilotspb.PresentExpression{Operand: value}}}
}
func succeeded(entry, node string) *testpilotspb.Expression {
	return &testpilotspb.Expression{Expression: &testpilotspb.Expression_Compare{Compare: &testpilotspb.CompareExpression{Operator: testpilotspb.COMPARISON_OPERATOR_EQUAL, Left: &testpilotspb.Expression{Expression: &testpilotspb.Expression_Reference{Reference: &testpilotspb.Reference{Reference: &testpilotspb.Reference_Outcome{Outcome: &testpilotspb.InstructionOutcomeReference{Instruction: &testpilotspb.InstructionReference{EntrypointId: entry, InstructionId: node}, Field: testpilotspb.INSTRUCTION_OUTCOME_FIELD_STATUS}}}}}, Right: &testpilotspb.Expression{Expression: &testpilotspb.Expression_Literal{Literal: &testpilotspb.Value{Value: &testpilotspb.Value_EnumValue{EnumValue: &testpilotspb.EnumValue{Number: 1}}}}}}}}
}
func runIDExpression() *testpilotspb.Expression {
	return &testpilotspb.Expression{Expression: &testpilotspb.Expression_Reference{Reference: &testpilotspb.Reference{Reference: &testpilotspb.Reference_Run{Run: &testpilotspb.RunReference{}}}}}
}
func addWorker(source *testpilotspb.Case, policy *Profile) {
	source.Program.Environment = append(source.Program.Environment, &testpilotspb.EnvironmentDefinition{BindingId: "namespace"}, &testpilotspb.EnvironmentDefinition{BindingId: "queue"})
	policy.EnvironmentBindings = append(policy.EnvironmentBindings, contract.EnvironmentBinding{ID: "namespace", Value: "namespace"}, contract.EnvironmentBinding{ID: "queue", Value: "queue"})
	source.Program.Roles = append(source.Program.Roles, &testpilotspb.Role{RoleId: "worker", Kind: testpilotspb.ROLE_KIND_WORKER, NamespaceBindingId: "namespace"}, &testpilotspb.Role{RoleId: "queue", Kind: testpilotspb.ROLE_KIND_TASK_QUEUE, NamespaceBindingId: "namespace", ResourceBindingId: "queue"})
	source.Program.Entrypoints = append(source.Program.Entrypoints, &testpilotspb.Entrypoint{EntrypointId: "workflow", Activation: &testpilotspb.Entrypoint_Workflow{Workflow: &testpilotspb.WorkflowActivation{WorkflowType: "flow", WorkerRoleId: "worker", TaskQueueRoleId: "queue"}}})
}

// An instruction input or guard shares the one expression language, so each Contract reference,
// and each correlated reference, rejects at preparation at the expression's located path.
func TestPrepareLocatesAReferenceOutsideTheProgramContext(t *testing.T) {
	references := map[string]*testpilotspb.Reference{
		"observation_id":     {Reference: &testpilotspb.Reference_ObservationId{ObservationId: "observation"}},
		"run_event":          {Reference: &testpilotspb.Reference_RunEvent{RunEvent: &testpilotspb.RunEventReference{Selection: &testpilotspb.RunEventReference_Field{Field: testpilotspb.RUN_EVENT_FIELD_KIND}}}},
		"capture_id":         {Reference: &testpilotspb.Reference_CaptureId{CaptureId: "capture"}},
		"evidence_field_id":  {Reference: &testpilotspb.Reference_EvidenceFieldId{EvidenceFieldId: "field"}},
		"correlated_capture": {Reference: &testpilotspb.Reference_CorrelatedCapture{CorrelatedCapture: &testpilotspb.CorrelatedCaptureReference{CaptureId: "capture"}}},
		"model_value":        {Reference: &testpilotspb.Reference_ModelValue{ModelValue: &testpilotspb.ModelValue{DefinitionId: "definition", Value: "value"}}},
		"correlated_step":    {Reference: &testpilotspb.Reference_CorrelatedStep{CorrelatedStep: &testpilotspb.CorrelatedStepReference{Field: testpilotspb.CORRELATED_STEP_FIELD_ACTION, DefinitionId: "definition"}}},
		"projected_value":    {Reference: &testpilotspb.Reference_ProjectedValue{ProjectedValue: &testpilotspb.ProjectedValueReference{}}},
	}
	for name, value := range references {
		expression := &testpilotspb.Expression{Expression: &testpilotspb.Expression_Reference{Reference: value}}
		for site, mutate := range map[string]func(*testpilotspb.Case){
			"program.entrypoints[controller].instructions[call].guard.present": func(c *testpilotspb.Case) {
				c.Program.Entrypoints[0].Instructions[0].Guard = present(expression)
			},
			"program.entrypoints[controller].instructions[call].instruction.invoke_rpc.request_assignments[1].value": func(c *testpilotspb.Case) {
				c.Program.Entrypoints[0].Instructions[0].Instruction.GetInvokeRpc().RequestAssignments = []*testpilotspb.RequestAssignment{
					{Target: field("items"), Value: &testpilotspb.Expression{Expression: &testpilotspb.Expression_Literal{Literal: &testpilotspb.Value{Value: &testpilotspb.Value_ListValue{ListValue: &testpilotspb.ValueList{}}}}}},
					{Target: field("text"), Value: expression},
				}
			},
			"program.cleanup.instructions[undo].guard.present": func(c *testpilotspb.Case) {
				undo := rpcNode("undo")
				undo.Guard = present(expression)
				c.Program.Cleanup.Instructions = []*testpilotspb.InstructionNode{undo}
			},
		} {
			t.Run(site+"/"+name, func(t *testing.T) {
				c, catalog, p := fixture(t)
				mutate(c)
				_, err := Prepare(c, catalog, p)
				var diagnostic *ir.Error
				require.ErrorAs(t, err, &diagnostic)
				require.Equal(t, &ir.Error{Category: ir.Unknown, Path: site + ".reference." + name, Detail: "reference is not admitted in this expression context"}, diagnostic)
			})
		}
	}
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
			c.Program.Entrypoints[0].Instructions[0].Dependencies = []*testpilotspb.InstructionReference{{EntrypointId: "controller", InstructionId: "call"}}
		},
		"cross entry": func(c *testpilotspb.Case, _ *Profile) {
			c.Program.Entrypoints[0].Instructions[0].Dependencies = []*testpilotspb.InstructionReference{{EntrypointId: "cleanup", InstructionId: "call"}}
		},
		"missing dependency": func(c *testpilotspb.Case, _ *Profile) {
			c.Program.Entrypoints[0].Instructions[0].Dependencies = []*testpilotspb.InstructionReference{{EntrypointId: "controller", InstructionId: "missing"}}
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
		"instruction response ceiling": func(_ *testpilotspb.Case, p *Profile) {
			p.Limits.MaxInstructionResponseBytes = 4097
		},
		"instruction event ceiling": func(_ *testpilotspb.Case, p *Profile) {
			p.Limits.MaxInstructionEmittedEvents = 257
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
	fields := p.Limits.ProtoReflect().Descriptor().Fields()
	for i := 0; i < fields.Len(); i++ {
		f := fields.Get(i)
		t.Run(string(f.Name()), func(t *testing.T) {
			for _, v := range []int64{0, hardLimits().ProtoReflect().Get(f).Int() + 1} {
				policy := p
				policy.Limits = proto.CloneOf(p.Limits)
				policy.Limits.ProtoReflect().Set(f, protoreflect.ValueOfInt64(v))
				_, err := Prepare(c, catalog, policy)
				var diagnostic *ir.Error
				require.ErrorAs(t, err, &diagnostic)
				require.Equal(t, ir.Error{Category: ir.LimitExceeded, Path: string(f.Name()), Detail: "limit is outside the positive Driver ceiling"}, *diagnostic)
			}
		})
	}
	_, err := Prepare(nil, catalog, p)
	require.Error(t, err)
	_, err = Prepare(c, nil, p)
	require.Error(t, err)
}

// A Case keeps only the bounds that carry its behavior; each still rejects at preparation when it
// exceeds the Profile ceiling it is admitted under, and is admitted at that ceiling.
func TestPrepareRejectsCaseBoundsAboveProfileCeilings(t *testing.T) {
	for _, tc := range []struct {
		name  string
		bound func(*testpilotspb.Case) *testpilotspb.InstructionLimits
		set   func(limits *testpilotspb.InstructionLimits, ceiling *testpilotspb.ProgramLimits, over int64)
		path  string
	}{
		{"ordinary timeout", func(c *testpilotspb.Case) *testpilotspb.InstructionLimits {
			return c.Program.Entrypoints[0].Instructions[0].Limits
		}, func(limits *testpilotspb.InstructionLimits, ceiling *testpilotspb.ProgramLimits, over int64) {
			limits.TimeoutMilliseconds = ceiling.MaxTotalDurationMilliseconds + over
		}, "controller.call"},
		{"cleanup timeout", func(c *testpilotspb.Case) *testpilotspb.InstructionLimits {
			c.Program.Cleanup.Instructions = []*testpilotspb.InstructionNode{rpcNode("cleanup-call")}
			return c.Program.Cleanup.Instructions[0].Limits
		}, func(limits *testpilotspb.InstructionLimits, ceiling *testpilotspb.ProgramLimits, over int64) {
			limits.TimeoutMilliseconds = ceiling.MaxCleanupDurationMilliseconds + over
		}, "cleanup.cleanup-call"},
		{"attempts", func(c *testpilotspb.Case) *testpilotspb.InstructionLimits {
			return c.Program.Entrypoints[0].Instructions[0].Limits
		}, func(limits *testpilotspb.InstructionLimits, ceiling *testpilotspb.ProgramLimits, over int64) {
			limits.MaxAttempts = ceiling.MaxAttempts + over
		}, "controller.call"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c, catalog, p := fixture(t)
			bound := tc.bound(c)
			tc.set(bound, p.Limits, 0)
			_, err := Prepare(c, catalog, p)
			require.NoError(t, err)
			tc.set(bound, p.Limits, 1)
			_, err = Prepare(c, catalog, p)
			var diagnostic *ir.Error
			require.ErrorAs(t, err, &diagnostic)
			require.Equal(t, ir.Error{Category: ir.LimitExceeded, Path: tc.path, Detail: "instruction bounds exceed Profile ceilings"}, *diagnostic)
		})
	}
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
		request, enabled, _, err := values.request(t.Context(), contract.Coordinate{
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
	c.Program.Entrypoints[1].Instructions = []*testpilotspb.InstructionNode{{
		InstructionId: "finish",
		Instruction: &testpilotspb.Instruction{Instruction: &testpilotspb.Instruction_Finish{Finish: &testpilotspb.Finish{
			Result: runIDExpression(),
		}}},
		Outcome: statusSchema(),
		Limits:  &testpilotspb.InstructionLimits{TimeoutMilliseconds: 1000, MaxAttempts: 1},
	}}
	_, err = Prepare(c, catalog, policy)
	require.Error(t, err)
}

// capCarrierShapes splits the Profile's activation ceiling evenly across each carrier's shapes, since a
// carrier's shapes together may not exceed that ceiling.
func capCarrierShapes(p *Profile) {
	for _, role := range p.Roles {
		for _, carrier := range role.ReservationCarriers {
			for i := range carrier.Shapes {
				carrier.Shapes[i].MaximumCount = max(1, p.Limits.MaxActivations/int64(len(carrier.Shapes)))
			}
		}
	}
}

func TestReservationAdmissionBoundsLocalAndGlobalAttempts(t *testing.T) {
	for _, test := range []struct {
		name                                string
		local, global, count, ceiling, want int64
		good                                bool
	}{
		{"local cap", 2, 32, 3, 7, 7, true}, {"equal caps", 2, 2, 3, 7, 7, true}, {"local above global", 8, 2, 3, 64, 0, false}, {"ceiling", 2, 32, 3, 6, 0, false}, {"zero", 1, 32, 0, 64, 0, false}, {"negative", 1, 32, -1, 64, 0, false}, {"overflow", 2, 32, math.MaxInt64, 64, 0, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			c, catalog, p := fixture(t)
			addWorker(c, &p)
			node := c.Program.Entrypoints[0].Instructions[0]
			node.Limits.MaxAttempts = test.local
			p.Limits.MaxAttempts = test.global
			p.Limits.MaxActivations = test.ceiling
			capCarrierShapes(&p)
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
	c.Program.Slots = []*testpilotspb.Slot{valueSlot("result", scalar(testpilotspb.SCALAR_KIND_TEXT))}
	c.Program.Observations = []*testpilotspb.Observation{{ObservationId: "text", Type: scalar(testpilotspb.SCALAR_KIND_TEXT)}}
	producer := c.Program.Entrypoints[0].Instructions[0]
	producer.Instruction.GetInvokeRpc().ResponseReads = []*testpilotspb.ResponseRead{{Path: field("text"), Cardinality: testpilotspb.READ_CARDINALITY_ONE, Targets: []*testpilotspb.ReadTarget{{Target: &testpilotspb.ReadTarget_SlotId{SlotId: "result"}}, {Target: &testpilotspb.ReadTarget_ObservationId{ObservationId: "text"}}}}}
	consumer := rpcNode("consume")
	consumer.Dependencies = []*testpilotspb.InstructionReference{{EntrypointId: "controller", InstructionId: "call"}}
	consumer.Guard = succeeded("controller", "call")
	consumer.Instruction.GetInvokeRpc().RequestAssignments = []*testpilotspb.RequestAssignment{{Target: field("text"), Value: slot("result")}}
	c.Program.Entrypoints[0].Instructions = append(c.Program.Entrypoints[0].Instructions, consumer)
	prepared, err := Prepare(c, catalog, p)
	require.NoError(t, err)
	for name, mutate := range map[string]func(*testpilotspb.Case){"unguarded": func(s *testpilotspb.Case) { s.Program.Entrypoints[0].Instructions[1].Guard = nil }, "missing dependency": func(s *testpilotspb.Case) { s.Program.Entrypoints[0].Instructions[1].Dependencies = nil }, "second writer": func(s *testpilotspb.Case) {
		s.Program.Entrypoints[0].Instructions[1].Instruction.GetInvokeRpc().ResponseReads = proto.CloneOf(producer.Instruction).GetInvokeRpc().ResponseReads
	}, "assignment overlap": func(s *testpilotspb.Case) {
		rpc := s.Program.Entrypoints[0].Instructions[1].Instruction.GetInvokeRpc()
		rpc.RequestAssignments = append(rpc.RequestAssignments, proto.CloneOf(rpc.RequestAssignments[0]))
	}, "crossed cardinality": func(s *testpilotspb.Case) {
		s.Program.Entrypoints[0].Instructions[0].Instruction.GetInvokeRpc().ResponseReads[0].Cardinality = testpilotspb.READ_CARDINALITY_EMIT_EACH
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
			c.Program.Entrypoints[1].Activation = &testpilotspb.Entrypoint_Controller{Controller: &testpilotspb.ControllerActivation{}}
		},
		"activity target": func(c *testpilotspb.Case) {
			g := c.Program.Entrypoints[1]
			g.Activation = &testpilotspb.Entrypoint_Activity{Activity: &testpilotspb.ActivityActivation{ActivityType: "activity", WorkerRoleId: "worker", TaskQueueRoleId: "queue"}}
		},
		"duplicate target": func(c *testpilotspb.Case) {
			n := c.Program.Entrypoints[0].Instructions[0]
			n.ActivationReservations = append(n.ActivationReservations, proto.CloneOf(n.ActivationReservations[0]))
		},
		"worker reservation": func(c *testpilotspb.Case) {
			n := c.Program.Entrypoints[0].Instructions[0]
			worker := proto.CloneOf(n)
			worker.Instruction = &testpilotspb.Instruction{Instruction: &testpilotspb.Instruction_Finish{Finish: &testpilotspb.Finish{Result: textLiteral("done")}}}
			c.Program.Entrypoints[1].Instructions = []*testpilotspb.InstructionNode{worker}
		},
		"cleanup reservation": func(c *testpilotspb.Case) {
			c.Program.Cleanup.Instructions = []*testpilotspb.InstructionNode{proto.CloneOf(c.Program.Entrypoints[0].Instructions[0])}
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
	second.Instructions[0].Limits.MaxAttempts = 4
	second.Instructions[0].ActivationReservations[0].Count = 3
	c.Program.Entrypoints = append(c.Program.Entrypoints, second)
	p.Limits.MaxAttempts = 4
	p.Limits.MaxActivations = 17
	capCarrierShapes(&p)
	p.Limits.MaxActivations = 18
	prepared, err := Prepare(c, catalog, p)
	require.NoError(t, err)
	require.Equal(t, int64(18), prepared.View().MaximumActivations())
	p.Limits.MaxActivations = 17
	_, err = Prepare(c, catalog, p)
	require.Error(t, err)
	first.ActivationReservations = nil
	second.Instructions[0].ActivationReservations = nil
	p.Limits.MaxActivations = 2
	capCarrierShapes(&p)
	prepared, err = Prepare(c, catalog, p)
	require.NoError(t, err)
	require.Equal(t, int64(2), prepared.View().MaximumActivations())
}
func textLiteral(value string) *testpilotspb.Expression {
	return &testpilotspb.Expression{Expression: &testpilotspb.Expression_Literal{Literal: &testpilotspb.Value{Value: &testpilotspb.Value_TextValue{TextValue: value}}}}
}

func environment(id string) *testpilotspb.Expression {
	return &testpilotspb.Expression{Expression: &testpilotspb.Expression_Reference{Reference: &testpilotspb.Reference{Reference: &testpilotspb.Reference_EnvironmentBindingId{EnvironmentBindingId: id}}}}
}

func TestPrepareResolvesClosedEnvironmentGraph(t *testing.T) {
	c, catalog, policy := fixture(t)
	c.Program.Environment = []*testpilotspb.EnvironmentDefinition{{BindingId: "namespace"}, {BindingId: "queue"}}
	c.Program.Roles = append(c.Program.Roles,
		&testpilotspb.Role{RoleId: "worker", Kind: testpilotspb.ROLE_KIND_WORKER, NamespaceBindingId: "namespace"},
		&testpilotspb.Role{RoleId: "queue", Kind: testpilotspb.ROLE_KIND_TASK_QUEUE, NamespaceBindingId: "namespace", ResourceBindingId: "queue"},
	)
	c.Program.Roles[0].ResourceBindingId = "queue"
	policy.EnvironmentBindings = []contract.EnvironmentBinding{{ID: "namespace", Value: "namespace-a"}, {ID: "queue", Value: "queue-a"}, {ID: "unused", Value: "allowed"}}
	policy.EnvironmentFingerprint = "fingerprint"
	c.Program.Entrypoints[0].Instructions[0].Instruction.GetInvokeRpc().RequestAssignments = []*testpilotspb.RequestAssignment{{Target: field("text"), Value: environment("namespace")}}

	prepared, err := Prepare(c, catalog, policy)
	require.NoError(t, err)
	require.Equal(t, "namespace", prepared.graphs[0].nodes[0].assignments[0].environmentBindingID)
	require.Equal(t, "namespace-a", prepared.graphs[0].nodes[0].assignments[0].value.Literal().GetTextValue())
	require.Equal(t, resolvedRole{ID: "endpoint", Kind: testpilotspb.ROLE_KIND_ENDPOINT, ResourceBindingID: "queue", Resource: "queue-a"}, prepared.roles["endpoint"])
	require.Equal(t, resolvedRole{ID: "queue", Kind: testpilotspb.ROLE_KIND_TASK_QUEUE, NamespaceBindingID: "namespace", Namespace: "namespace-a", ResourceBindingID: "queue", Resource: "queue-a"}, prepared.roles["queue"])
	require.NotEmpty(t, prepared.environmentFingerprint)
	store, err := newValueStore(prepared, "run")
	require.NoError(t, err)
	values, err := store.activate("controller", "activation")
	require.NoError(t, err)
	request, dispatched, _, err := values.request(t.Context(), contract.Coordinate{RunID: "run", EntrypointID: "controller", ActivationID: "activation", InstructionID: "call", Attempt: 1}, prepared.graphs[0].runtimeWork)
	require.NoError(t, err)
	require.True(t, dispatched)
	textField := request.ProtoReflect().Descriptor().Fields().ByName("text")
	require.Equal(t, "namespace-a", request.ProtoReflect().Get(textField).String())

	c.Program.Environment[0].BindingId = "changed"
	c.Program.Roles[2].ResourceBindingId = "changed"
	policy.EnvironmentBindings[0].Value = "changed"
	require.Equal(t, "namespace", prepared.Snapshot().Environment[0].BindingId)
	require.Equal(t, "namespace", prepared.Snapshot().Entrypoints[0].Instructions[0].Instruction.GetInvokeRpc().RequestAssignments[0].Value.GetReference().GetEnvironmentBindingId())
	require.Equal(t, "namespace-a", prepared.graphs[0].nodes[0].assignments[0].value.Literal().GetTextValue())
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
			p.EnvironmentBindings = append(p.EnvironmentBindings, contract.EnvironmentBinding{ID: "unused", Value: "value"})
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
			c.Program.Entrypoints[0].Instructions[0].Guard = &testpilotspb.Expression{Expression: &testpilotspb.Expression_Compare{Compare: &testpilotspb.CompareExpression{Operator: testpilotspb.COMPARISON_OPERATOR_EQUAL, Left: environment("binding"), Right: textLiteral("value")}}}
		},
		"non-text destination": func(c *testpilotspb.Case, p *Profile) {
			configureEnvironmentCase(c, p)
			c.Program.Entrypoints[0].Instructions[0].Instruction.GetInvokeRpc().RequestAssignments[0].Target = field("items")
		},
		"resolved byte overflow": func(c *testpilotspb.Case, p *Profile) {
			configureEnvironmentCase(c, p)
			p.Limits.MaxRequestBytes = 8
		},
		"incompatible endpoint namespace": func(c *testpilotspb.Case, p *Profile) {
			configureEnvironmentCase(c, p)
			c.Program.Roles[0].NamespaceBindingId = "binding"
		},
		"1.1 worker without namespace": func(c *testpilotspb.Case, p *Profile) {
			configureEnvironmentCase(c, p)
			c.Program.Roles = append(c.Program.Roles, &testpilotspb.Role{RoleId: "worker", Kind: testpilotspb.ROLE_KIND_WORKER})
		},
		"1.1 worker with resource": func(c *testpilotspb.Case, p *Profile) {
			configureEnvironmentCase(c, p)
			c.Program.Roles = append(c.Program.Roles, &testpilotspb.Role{RoleId: "worker", Kind: testpilotspb.ROLE_KIND_WORKER, NamespaceBindingId: "binding", ResourceBindingId: "binding"})
		},
		"1.1 task queue without namespace": func(c *testpilotspb.Case, p *Profile) {
			configureEnvironmentCase(c, p)
			c.Program.Roles = append(c.Program.Roles, &testpilotspb.Role{RoleId: "queue", Kind: testpilotspb.ROLE_KIND_TASK_QUEUE, ResourceBindingId: "binding"})
		},
		"1.1 task queue without resource": func(c *testpilotspb.Case, p *Profile) {
			configureEnvironmentCase(c, p)
			c.Program.Roles = append(c.Program.Roles, &testpilotspb.Role{RoleId: "queue", Kind: testpilotspb.ROLE_KIND_TASK_QUEUE, NamespaceBindingId: "binding"})
		},
		"1.1 participant with resource": func(c *testpilotspb.Case, p *Profile) {
			configureEnvironmentCase(c, p)
			p.Roles = append(p.Roles, contract.RolePolicy{ID: "participant", Kind: testpilotspb.ROLE_KIND_PARTICIPANT})
			c.Program.Roles = append(c.Program.Roles, &testpilotspb.Role{RoleId: "participant", Kind: testpilotspb.ROLE_KIND_PARTICIPANT, ResourceBindingId: "binding"})
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
	for name, bindings := range map[string][]contract.EnvironmentBinding{
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
	policy.EnvironmentBindings = make([]contract.EnvironmentBinding, 10001)
	_, err := Prepare(c, catalog, policy)
	require.Error(t, err)

	c, catalog, policy = fixture(t)
	policy.Limits.MaxRequestBytes = 8
	policy.EnvironmentBindings = []contract.EnvironmentBinding{{ID: "id", Value: "1234567"}}
	_, err = Prepare(c, catalog, policy)
	require.Error(t, err)
}

func configureEnvironmentCase(c *testpilotspb.Case, policy *Profile) {
	c.Program.Environment = []*testpilotspb.EnvironmentDefinition{{BindingId: "binding"}}
	policy.EnvironmentBindings = []contract.EnvironmentBinding{{ID: "binding", Value: "value"}}
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
			policy.EnvironmentBindings = []contract.EnvironmentBinding{{ID: "binding", Value: value}}
			prepared, err := Prepare(c, catalog, policy)
			require.NoError(t, err)
			store, err := newValueStore(prepared, "run")
			require.NoError(t, err)
			values, err := store.activate("controller", "activation")
			require.NoError(t, err)
			request, dispatched, _, err := values.request(t.Context(), contract.Coordinate{RunID: "run", EntrypointID: "controller", ActivationID: "activation", InstructionID: "call", Attempt: 1}, prepared.graphs[0].runtimeWork)
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
			request, dispatched, _, err := values.request(t.Context(), contract.Coordinate{RunID: fmt.Sprintf("run-%d", i), EntrypointID: "controller", ActivationID: "activation", InstructionID: "call", Attempt: 1}, prepared.graphs[0].runtimeWork)
			require.NoError(t, err)
			require.True(t, dispatched)
			textField := request.ProtoReflect().Descriptor().Fields().ByName("text")
			require.Equal(t, "value", request.ProtoReflect().Get(textField).String())
		})
	}
}

func TestInstructionContextMatrix(t *testing.T) {
	for _, context := range []contract.EntrypointKind{contract.ControllerEntrypoint, contract.WorkflowEntrypoint, contract.ActivityEntrypoint, contract.NexusHandlerEntrypoint} {
		for _, test := range []struct {
			name        string
			instruction *testpilotspb.Instruction
			expected    contract.EntrypointKind
		}{
			{"rpc", rpcNode("call").Instruction, contract.ControllerEntrypoint},
			{"await slot", &testpilotspb.Instruction{Instruction: &testpilotspb.Instruction_AwaitSlot{AwaitSlot: &testpilotspb.AwaitSlot{SlotId: "value"}}}, contract.ControllerEntrypoint},
			{"complete", &testpilotspb.Instruction{Instruction: &testpilotspb.Instruction_CompleteNexusOperation{CompleteNexusOperation: &testpilotspb.CompleteNexusOperation{HandleSlotId: "capability", Result: textLiteral("done")}}}, contract.ControllerEntrypoint},
			{"start", &testpilotspb.Instruction{Instruction: &testpilotspb.Instruction_StartNexusOperation{StartNexusOperation: &testpilotspb.StartNexusOperation{EndpointRoleId: "endpoint", Service: "service", Operation: "operation", Input: textLiteral("input")}}}, contract.WorkflowEntrypoint},
			{"await", &testpilotspb.Instruction{Instruction: &testpilotspb.Instruction_AwaitInstruction{AwaitInstruction: &testpilotspb.AwaitInstruction{Instruction: &testpilotspb.InstructionReference{EntrypointId: "workflow", InstructionId: "prior"}}}}, contract.WorkflowEntrypoint},
			{"finish", &testpilotspb.Instruction{Instruction: &testpilotspb.Instruction_Finish{Finish: &testpilotspb.Finish{Result: textLiteral("done")}}}, contract.WorkflowEntrypoint},
			{"respond", &testpilotspb.Instruction{Instruction: &testpilotspb.Instruction_RespondNexus{RespondNexus: &testpilotspb.RespondNexus{Kind: testpilotspb.NEXUS_RESPONSE_KIND_SYNCHRONOUS, Result: textLiteral("done")}}}, contract.NexusHandlerEntrypoint},
		} {
			t.Run(fmt.Sprintf("kind %d/%s", context, test.name), func(t *testing.T) {
				if context == test.expected {
					return
				}
				c, catalog, p := fixture(t)
				g := c.Program.Entrypoints[0]
				g.Instructions[0].Instruction = test.instruction
				switch context {
				case contract.WorkflowEntrypoint:
					g.Activation = &testpilotspb.Entrypoint_Workflow{Workflow: &testpilotspb.WorkflowActivation{WorkflowType: "flow", WorkerRoleId: "worker", TaskQueueRoleId: "queue"}}
				case contract.ActivityEntrypoint:
					g.Activation = &testpilotspb.Entrypoint_Activity{Activity: &testpilotspb.ActivityActivation{ActivityType: "activity", WorkerRoleId: "worker", TaskQueueRoleId: "queue"}}
				case contract.NexusHandlerEntrypoint:
					g.Activation = &testpilotspb.Entrypoint_NexusHandler{NexusHandler: &testpilotspb.NexusHandlerActivation{Service: "service", Operation: "operation", WorkerRoleId: "worker", TaskQueueRoleId: "queue"}}
				default:
				}
				c.Program.Roles = append(c.Program.Roles, &testpilotspb.Role{RoleId: "worker", Kind: testpilotspb.ROLE_KIND_WORKER}, &testpilotspb.Role{RoleId: "queue", Kind: testpilotspb.ROLE_KIND_TASK_QUEUE})
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
	c.Program.Slots = []*testpilotspb.Slot{capabilitySlot("capability")}
	wait := rpcNode("ready")
	wait.Instruction = &testpilotspb.Instruction{Instruction: &testpilotspb.Instruction_AwaitSlot{AwaitSlot: &testpilotspb.AwaitSlot{SlotId: "capability"}}}
	complete := rpcNode("complete")
	complete.Dependencies = []*testpilotspb.InstructionReference{{EntrypointId: "controller", InstructionId: "ready"}}
	complete.Guard = succeeded("controller", "ready")
	complete.Instruction = &testpilotspb.Instruction{Instruction: &testpilotspb.Instruction_CompleteNexusOperation{CompleteNexusOperation: &testpilotspb.CompleteNexusOperation{HandleSlotId: "capability", Result: textLiteral("done")}}}
	c.Program.Entrypoints[0].Instructions = append(c.Program.Entrypoints[0].Instructions, wait, complete)
	handler := rpcNode("respond")
	handler.Instruction = &testpilotspb.Instruction{Instruction: &testpilotspb.Instruction_RespondNexus{RespondNexus: &testpilotspb.RespondNexus{Kind: testpilotspb.NEXUS_RESPONSE_KIND_ASYNCHRONOUS, HandleSlotId: "capability", Result: textLiteral("accepted")}}}
	c.Program.Entrypoints = append(c.Program.Entrypoints, &testpilotspb.Entrypoint{EntrypointId: "handler", Activation: &testpilotspb.Entrypoint_NexusHandler{NexusHandler: &testpilotspb.NexusHandlerActivation{Service: "service", Operation: "operation", WorkerRoleId: "worker", TaskQueueRoleId: "queue"}}, Instructions: []*testpilotspb.InstructionNode{handler}})
	start := rpcNode("start")
	start.Instruction = &testpilotspb.Instruction{Instruction: &testpilotspb.Instruction_StartNexusOperation{StartNexusOperation: &testpilotspb.StartNexusOperation{EndpointRoleId: "endpoint", Service: "service", Operation: "operation", Input: textLiteral("input")}}}
	await := rpcNode("await")
	await.Dependencies = []*testpilotspb.InstructionReference{{EntrypointId: "workflow", InstructionId: "start"}}
	await.Instruction = &testpilotspb.Instruction{Instruction: &testpilotspb.Instruction_AwaitInstruction{AwaitInstruction: &testpilotspb.AwaitInstruction{Instruction: &testpilotspb.InstructionReference{EntrypointId: "workflow", InstructionId: "start"}}}}
	await.Outcome.Fields = append(await.Outcome.Fields, &testpilotspb.OutcomeFieldDefinition{Field: testpilotspb.INSTRUCTION_OUTCOME_FIELD_VALUE, Type: scalar(testpilotspb.SCALAR_KIND_TEXT)})
	finish := rpcNode("finish")
	finish.Dependencies = []*testpilotspb.InstructionReference{{EntrypointId: "workflow", InstructionId: "await"}}
	finish.Guard = succeeded("workflow", "await")
	finish.Instruction = &testpilotspb.Instruction{Instruction: &testpilotspb.Instruction_Finish{Finish: &testpilotspb.Finish{Result: &testpilotspb.Expression{Expression: &testpilotspb.Expression_Reference{Reference: &testpilotspb.Reference{Reference: &testpilotspb.Reference_Outcome{Outcome: &testpilotspb.InstructionOutcomeReference{Instruction: &testpilotspb.InstructionReference{EntrypointId: "workflow", InstructionId: "await"}, Field: testpilotspb.INSTRUCTION_OUTCOME_FIELD_VALUE}}}}}}}}
	c.Program.Entrypoints[1].Instructions = []*testpilotspb.InstructionNode{start, await, finish}
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
			s.Program.Entrypoints[0].Instructions[0].Instruction.GetInvokeRpc().ResponseReads = []*testpilotspb.ResponseRead{{Path: field("text"), Cardinality: testpilotspb.READ_CARDINALITY_ONE, Targets: []*testpilotspb.ReadTarget{{Target: &testpilotspb.ReadTarget_SlotId{SlotId: "capability"}}}}}
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
	require.Equal(t, "input", instructions[0].Input().Literal().GetTextValue())
	require.Equal(t, []int{0}, instructions[1].Dependencies())
	instructions[0].Input().Literal().Value = &testpilotspb.Value_TextValue{TextValue: "changed"}
	instructions[0].Source().Instruction = nil
	worker.Activation().GetWorkflow().WorkflowType = "changed"
	worker.Order()[0] = 99
	require.Equal(t, "input", worker.Instructions()[0].Input().Literal().GetTextValue())
	require.Equal(t, "flow", worker.Activation().GetWorkflow().WorkflowType)
	require.Equal(t, []int{0, 1, 2}, worker.Order())
}

func TestOutcomeStatusesAndCleanupLocalReferences(t *testing.T) {
	c, catalog, p := fixture(t)
	first := rpcNode("release")
	second := rpcNode("confirm")
	second.Dependencies = []*testpilotspb.InstructionReference{{EntrypointId: "cleanup", InstructionId: "release"}}
	second.Guard = succeeded("cleanup", "release")
	c.Program.Cleanup.Instructions = []*testpilotspb.InstructionNode{first, second}
	for status := int32(1); status <= 5; status++ {
		second.Guard.GetCompare().Right.GetLiteral().GetEnumValue().Number = status
		_, err := Prepare(c, catalog, p)
		require.NoError(t, err)
	}
	second.Guard.GetCompare().Right.GetLiteral().GetEnumValue().Number = 99
	_, err := Prepare(c, catalog, p)
	require.Error(t, err)
	second.Guard = succeeded("controller", "call")
	_, err = Prepare(c, catalog, p)
	require.Error(t, err)
}

func TestSlotOwnersAndConcurrentPreparedViews(t *testing.T) {
	c, catalog, p := fixture(t)
	c.Program.Slots = []*testpilotspb.Slot{valueSlot("value", scalar(testpilotspb.SCALAR_KIND_TEXT))}
	c.Program.Entrypoints[0].Instructions[0].Instruction.GetInvokeRpc().ResponseReads = []*testpilotspb.ResponseRead{{Path: field("text"), Cardinality: testpilotspb.READ_CARDINALITY_ONE, Targets: []*testpilotspb.ReadTarget{{Target: &testpilotspb.ReadTarget_SlotId{SlotId: "value"}}}}}
	other := proto.CloneOf(c.Program.Entrypoints[0])
	other.EntrypointId = "other"
	other.Instructions[0].Instruction.GetInvokeRpc().ResponseReads = nil
	other.Instructions[0].Guard = present(slot("value"))
	other.Instructions[0].Instruction.GetInvokeRpc().RequestAssignments = []*testpilotspb.RequestAssignment{{Target: field("text"), Value: slot("value")}}
	c.Program.Entrypoints = append(c.Program.Entrypoints, other)
	prepared, err := Prepare(c, catalog, p)
	require.NoError(t, err)
	c.Program.Cleanup.Instructions = []*testpilotspb.InstructionNode{proto.CloneOf(other.Instructions[0])}
	_, err = Prepare(c, catalog, p)
	require.NoError(t, err)
	addWorker(c, &p)
	worker := c.Program.Entrypoints[len(c.Program.Entrypoints)-1]
	finish := rpcNode("finish")
	finish.Guard = present(slot("value"))
	finish.Instruction = &testpilotspb.Instruction{Instruction: &testpilotspb.Instruction_Finish{Finish: &testpilotspb.Finish{Result: slot("value")}}}
	worker.Instructions = []*testpilotspb.InstructionNode{finish}
	_, err = Prepare(c, catalog, p)
	require.Error(t, err)
	for i := 0; i < 8; i++ {
		t.Run(fmt.Sprint(i), func(t *testing.T) {
			t.Parallel()
			for j := 0; j < 10; j++ {
				plans := prepared.Entrypoints()
				plans[0].Instructions()[0].Projections()[0].Sinks[0].Target = &testpilotspb.ReadTarget_SlotId{SlotId: "changed"}
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
	expression := &testpilotspb.Expression{}
	expression.Expression = &testpilotspb.Expression_Not{Not: &testpilotspb.NotExpression{Operand: expression}}
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
		"entrypoint count": func(c *testpilotspb.Case, p *Profile) { addWorker(c, p); p.Limits.MaxEntrypoints = 1 },
		"node count": func(c *testpilotspb.Case, p *Profile) {
			c.Program.Entrypoints[0].Instructions = append(c.Program.Entrypoints[0].Instructions, rpcNode("other"))
			p.Limits.MaxNodes = 1
		},
		"edge count": func(c *testpilotspb.Case, p *Profile) {
			last := rpcNode("last")
			last.Dependencies = []*testpilotspb.InstructionReference{{EntrypointId: "controller", InstructionId: "call"}, {EntrypointId: "controller", InstructionId: "other"}}
			c.Program.Entrypoints[0].Instructions = append(c.Program.Entrypoints[0].Instructions, rpcNode("other"), last)
			p.Limits.MaxEdges = 1
		},
		"controller activation count": func(c *testpilotspb.Case, p *Profile) {
			other := proto.CloneOf(c.Program.Entrypoints[0])
			other.EntrypointId = "other"
			c.Program.Entrypoints = append(c.Program.Entrypoints, other)
			p.Limits.MaxActivations = 1
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
	c.Program.Observations = []*testpilotspb.Observation{{ObservationId: "item", Type: scalar(testpilotspb.SCALAR_KIND_TEXT)}}
	n := c.Program.Entrypoints[0].Instructions[0]
	p.Limits.MaxInstructionEmittedEvents = 128
	n.Instruction.GetInvokeRpc().ResponseReads = []*testpilotspb.ResponseRead{{Path: field("items"), Cardinality: testpilotspb.READ_CARDINALITY_EMIT_EACH, Targets: []*testpilotspb.ReadTarget{{Target: &testpilotspb.ReadTarget_ObservationId{ObservationId: "item"}}}}}
	_, err := Prepare(c, catalog, p)
	require.NoError(t, err)
	p.Limits.MaxInstructionEmittedEvents = 127
	_, err = Prepare(c, catalog, p)
	require.Error(t, err)
	p.Limits.MaxInstructionEmittedEvents = 256
	n.Instruction.GetInvokeRpc().ResponseReads = append(n.Instruction.GetInvokeRpc().ResponseReads, proto.CloneOf(n.Instruction.GetInvokeRpc().ResponseReads[0]))
	_, err = Prepare(c, catalog, p)
	require.Error(t, err)
}

func TestWholeRequestAssignments(t *testing.T) {
	c, catalog, p := fixture(t)
	typ := &testpilotspb.ValueType{Shape: &testpilotspb.ValueType_Singular{Singular: &testpilotspb.SingularType{Type: &testpilotspb.SingularType_Message{Message: &testpilotspb.NamedType{ProtobufType: "example.Payload"}}}}}
	c.Program.Slots = []*testpilotspb.Slot{valueSlot("request", typ)}
	producer := c.Program.Entrypoints[0].Instructions[0]
	producer.Instruction.GetInvokeRpc().ResponseReads = []*testpilotspb.ResponseRead{{Path: &testpilotspb.FieldPath{}, Cardinality: testpilotspb.READ_CARDINALITY_ONE, Targets: []*testpilotspb.ReadTarget{{Target: &testpilotspb.ReadTarget_SlotId{SlotId: "request"}}}}}
	consumer := rpcNode("copy")
	consumer.Dependencies = []*testpilotspb.InstructionReference{{EntrypointId: "controller", InstructionId: "call"}}
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
			n.Dependencies = []*testpilotspb.InstructionReference{{EntrypointId: "workflow", InstructionId: target}}
			n.Instruction = &testpilotspb.Instruction{Instruction: &testpilotspb.Instruction_AwaitInstruction{AwaitInstruction: &testpilotspb.AwaitInstruction{Instruction: proto.CloneOf(n.Dependencies[0])}}}
			g.Instructions = append([]*testpilotspb.InstructionNode{n}, g.Instructions...)
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
