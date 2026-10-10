package execution

import (
	"fmt"
	"strings"
	"testing"
	"time"

	celpb "cel.dev/expr"
	"github.com/stretchr/testify/require"
	commandpb "go.temporal.io/api/command/v1"
	enumspb "go.temporal.io/api/enums/v1"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/testpilot/casefile"
	cel "go.temporal.io/server/common/testing/testpilot/cel"
	"go.temporal.io/server/common/testing/testpilot/contract"
	pbduration "go.temporal.io/server/common/testing/testpilot/duration"
	"go.temporal.io/server/common/testing/testpilot/internal/ir"
	"go.temporal.io/server/common/testing/testpilot/internal/testsupport"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/emptypb"
)

func fixture(t *testing.T) (*testpilotspb.Case, *ir.Catalog, Profile) {
	t.Helper()
	catalog, err := ir.NewCatalog(&descriptorpb.FileDescriptorSet{File: []*descriptorpb.FileDescriptorProto{{Name: proto.String("admission.proto"), Package: proto.String("example"), Syntax: proto.String("proto3"), MessageType: []*descriptorpb.DescriptorProto{{Name: proto.String("Payload"), Field: []*descriptorpb.FieldDescriptorProto{{Name: proto.String("text"), Number: proto.Int32(1), Type: descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(), Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum()}, {Name: proto.String("items"), Number: proto.Int32(2), Type: descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(), Label: descriptorpb.FieldDescriptorProto_LABEL_REPEATED.Enum()}}}}, Service: []*descriptorpb.ServiceDescriptorProto{{Name: proto.String("Service"), Method: []*descriptorpb.MethodDescriptorProto{{Name: proto.String("Call"), InputType: proto.String(".example.Payload"), OutputType: proto.String(".example.Payload")}, {Name: proto.String("Stream"), InputType: proto.String(".example.Payload"), OutputType: proto.String(".example.Payload"), ServerStreaming: proto.Bool(true)}}}}}}})
	require.NoError(t, err)
	limits := testsupport.ProgramLimits()
	policy := Profile{Identity: "host", CatalogIdentity: catalog.Identity(), Roles: []contract.RolePolicy{{ID: "endpoint", Kind: testpilotspb.ROLE_KIND_ENDPOINT, Methods: []string{"/example.Service/Call"}, ReservationCarriers: []contract.ReservationCarrierPolicy{{Method: "/example.Service/Call", Shapes: []contract.ReservationCarrierShape{{Kind: contract.WorkflowEntrypoint, MaximumCount: 32}, {Kind: contract.NexusHandlerEntrypoint, MaximumCount: 32}}}}}, {ID: "worker", Kind: testpilotspb.ROLE_KIND_WORKER}, {ID: "queue", Kind: testpilotspb.ROLE_KIND_TASK_QUEUE}}, Opcodes: []contract.Opcode{contract.InvokeRPC, contract.AwaitSlot, contract.Await, contract.Finish, contract.WorkflowCommand, contract.NexusHandlerReply, contract.NexusOperationCompletion}, CommandTypes: []enumspb.CommandType{enumspb.COMMAND_TYPE_SCHEDULE_NEXUS_OPERATION}, Limits: proto.CloneOf(limits)}
	source := &testpilotspb.Case{Version: &testpilotspb.FormatVersion{Major: casefile.CurrentMajor}, CaseId: "case", Program: &testpilotspb.Program{ProgramId: "program", Roles: []*testpilotspb.Role{{RoleId: "endpoint", Kind: testpilotspb.ROLE_KIND_ENDPOINT}}, Entrypoints: []*testpilotspb.Entrypoint{{EntrypointId: "controller", Activation: &testpilotspb.Entrypoint_Controller{Controller: &emptypb.Empty{}}, Instructions: []*testpilotspb.InstructionNode{rpcNode("call")}}}, Cleanup: &testpilotspb.Cleanup{EntrypointId: "cleanup"}}, Contract: &testpilotspb.Contract{ContractId: "contract"}}
	return source, catalog, policy
}
func scalar(kind testpilotspb.ScalarKind) *testpilotspb.ValueType {
	return &testpilotspb.ValueType{Shape: &testpilotspb.ValueType_Singular{Singular: &testpilotspb.SingularType{Type: &testpilotspb.SingularType_Scalar{Scalar: &testpilotspb.ScalarType{Kind: kind}}}}}
}

func valueSlot(id string, typ *testpilotspb.ValueType) *testpilotspb.Slot {
	return &testpilotspb.Slot{SlotId: id, Content: &testpilotspb.Slot_Value{Value: typ}}
}

func handleSlot(id string) *testpilotspb.Slot {
	return &testpilotspb.Slot{SlotId: id, Content: &testpilotspb.Slot_OpaqueHandle{OpaqueHandle: &emptypb.Empty{}}}
}
func rpcNode(id string) *testpilotspb.InstructionNode {
	return &testpilotspb.InstructionNode{InstructionId: id, Instruction: &testpilotspb.Instruction{Instruction: &testpilotspb.Instruction_InvokeRpc{InvokeRpc: &testpilotspb.InvokeRpc{EndpointRoleId: "endpoint", Method: "/example.Service/Call"}}}, Limits: &testpilotspb.InstructionLimits{Timeout: durationpb.New(time.Duration(1000) * time.Millisecond), MaxAttempts: proto.Int64(1)}}
}
func slot(id string) *testpilotspb.Expression {
	return cel.Ref(&testpilotspb.Reference{Reference: &testpilotspb.Reference_SlotId{SlotId: id}})
}
func present(value *testpilotspb.Expression) *testpilotspb.Expression {
	return cel.Present(value)
}
func succeeded(entry, node string) *testpilotspb.Expression {
	status := cel.Ref(&testpilotspb.Reference{Reference: &testpilotspb.Reference_Outcome{Outcome: &testpilotspb.InstructionOutcomeReference{Instruction: &testpilotspb.InstructionReference{EntrypointId: entry, InstructionId: node}, Field: testpilotspb.INSTRUCTION_OUTCOME_FIELD_STATUS}}})
	return cel.All(cel.Present(status), cel.Compare("_==_", status, cel.Literal(cel.Enum(testpilotspb.INSTRUCTION_OUTCOME_STATUS_SUCCEEDED))))
}

// runsAfter names the instructions of entrypointID a node runs after; with no ids it makes the node
// a root wherever it is declared.
func runsAfter(entrypointID string, instructionIDs ...string) *testpilotspb.After {
	after := &testpilotspb.After{Instructions: testsupport.LocalReferences([]*testpilotspb.InstructionReference{})}
	for _, id := range instructionIDs {
		after.Instructions = append(after.Instructions, &testpilotspb.LocalInstructionReference{InstructionId: id})
	}
	return after
}

func alwaysRuns() *testpilotspb.Expression {
	return cel.Literal(&celpb.Value{Kind: &celpb.Value_BoolValue{BoolValue: true}})
}
func runIDExpression() *testpilotspb.Expression {
	return cel.Ref(&testpilotspb.Reference{Reference: &testpilotspb.Reference_Run{Run: &emptypb.Empty{}}})
}
func addWorker(source *testpilotspb.Case, policy *Profile) {
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
		"correlated_step":    {Reference: &testpilotspb.Reference_CorrelatedStep{CorrelatedStep: &testpilotspb.CorrelatedStepReference{Field: testpilotspb.CORRELATED_STEP_FIELD_ACTION, DefinitionId: "definition"}}},
		"projected_value":    {Reference: &testpilotspb.Reference_ProjectedValue{ProjectedValue: &emptypb.Empty{}}},
		"instance_value_id":  {Reference: &testpilotspb.Reference_InstanceValueId{InstanceValueId: "value"}},
	}
	for name, value := range references {
		expression := cel.Ref(value)
		for site, mutate := range map[string]func(*testpilotspb.Case){
			"program.entrypoints[controller].instructions[call].guard.present": func(c *testpilotspb.Case) {
				c.Program.Entrypoints[0].Instructions[0].Guard = present(expression)
			},
			"program.entrypoints[controller].instructions[call].instruction.invoke_rpc.request_assignments[1].value": func(c *testpilotspb.Case) {
				c.Program.Entrypoints[0].Instructions[0].Instruction.GetInvokeRpc().RequestAssignments = []*testpilotspb.RequestAssignment{
					{Target: "items", Value: cel.Literal(&celpb.Value{Kind: &celpb.Value_ListValue{ListValue: &celpb.ListValue{}}})},
					{Target: "text", Value: expression},
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
				require.Equal(t, &ir.Error{Category: ir.Unknown, Path: strings.TrimSuffix(site, ".present") + ".bindings[0].reference." + name, Detail: "reference is not admitted in this expression context"}, diagnostic)
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
		"binding": func(c *testpilotspb.Case, _ *Profile) { c.Program.Entrypoints[0].Activation = nil },
		"role":    func(c *testpilotspb.Case, _ *Profile) { c.Program.Roles[0].Kind = testpilotspb.ROLE_KIND_WORKER },
		"method": func(c *testpilotspb.Case, _ *Profile) {
			c.Program.Entrypoints[0].Instructions[0].Instruction.GetInvokeRpc().Method = "/example.Service/Missing"
		},
		"authorization":    func(_ *testpilotspb.Case, p *Profile) { p.Roles[0].Methods = nil },
		"capability":       func(_ *testpilotspb.Case, p *Profile) { p.Opcodes = nil },
		"catalog identity": func(_ *testpilotspb.Case, p *Profile) { p.CatalogIdentity = "other" },
		"node timeout": func(c *testpilotspb.Case, _ *Profile) {
			c.Program.Entrypoints[0].Instructions[0].Limits.Timeout = pbduration.FromMilliseconds(0)
		},
		"attempt bound": func(c *testpilotspb.Case, _ *Profile) {
			c.Program.Entrypoints[0].Instructions[0].Limits.MaxAttempts = proto.Int64(33)
		},
		"instruction response ceiling": func(_ *testpilotspb.Case, p *Profile) {
			p.Limits.MaxInstructionResponseBytes = 4097
		},
		"instruction event ceiling": func(_ *testpilotspb.Case, p *Profile) {
			p.Limits.MaxInstructionEmittedEvents = 257
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
		if f.Kind() == protoreflect.MessageKind {
			continue
		}
		t.Run(string(f.Name()), func(t *testing.T) {
			for _, v := range []int64{0, ProgramCeiling().ProtoReflect().Get(f).Int() + 1} {
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
			limits.Timeout = pbduration.FromMilliseconds(ceiling.MaxDuration.AsDuration().Milliseconds() + over)
		}, "controller.call"},
		{"cleanup timeout", func(c *testpilotspb.Case) *testpilotspb.InstructionLimits {
			c.Program.Cleanup.Instructions = []*testpilotspb.InstructionNode{rpcNode("cleanup-call")}
			return c.Program.Cleanup.Instructions[0].Limits
		}, func(limits *testpilotspb.InstructionLimits, ceiling *testpilotspb.ProgramLimits, over int64) {
			limits.Timeout = pbduration.FromMilliseconds(ceiling.CleanupDuration.AsDuration().Milliseconds() + over)
		}, "cleanup.cleanup-call"},
		{"attempts", func(c *testpilotspb.Case) *testpilotspb.InstructionLimits {
			return c.Program.Entrypoints[0].Instructions[0].Limits
		}, func(limits *testpilotspb.InstructionLimits, ceiling *testpilotspb.ProgramLimits, over int64) {
			limits.MaxAttempts = proto.Int64(ceiling.MaxAttempts + over)
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
		Target: "text", Value: runIDExpression(),
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
		Limits: &testpilotspb.InstructionLimits{Timeout: durationpb.New(time.Duration(1000) * time.Millisecond), MaxAttempts: proto.Int64(1)},
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

// addWorkflows adds count further workflow entrypoints beside addWorker's, each of which the fixture's
// carrier reserves one activation of.
func addWorkflows(source *testpilotspb.Case, count int) {
	for i := range count {
		workflow := proto.CloneOf(source.Program.Entrypoints[1])
		workflow.EntrypointId = fmt.Sprintf("workflow_%d", i)
		source.Program.Entrypoints = append(source.Program.Entrypoints, workflow)
	}
}

// Each Case admits exactly at the activation ceiling its attempt-scaled reservations reach, and is
// rejected one below it.
func TestReservationAdmissionBoundsLocalAndGlobalAttempts(t *testing.T) {
	for _, test := range []struct {
		name                              string
		local, global, workflows, ceiling int64
		good                              bool
	}{
		{"local cap", 2, 32, 3, 7, true}, {"equal caps", 2, 2, 3, 7, true}, {"local above global", 8, 2, 3, 64, false}, {"ceiling", 2, 32, 3, 6, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			prepare := func(ceiling int64) error {
				c, catalog, p := fixture(t)
				addWorker(c, &p)
				addWorkflows(c, int(test.workflows-1))
				node := c.Program.Entrypoints[0].Instructions[0]
				node.Limits.MaxAttempts = proto.Int64(test.local)
				p.Limits.MaxAttempts = test.global
				p.Limits.MaxEntrypoints = test.workflows + 1
				p.Limits.MaxActivations = ceiling
				capCarrierShapes(&p)
				_, err := Prepare(c, catalog, p)
				return err
			}
			if test.good {
				require.NoError(t, prepare(test.ceiling))
				require.ErrorContains(t, prepare(test.ceiling-1), "attempt-scaled reservations exceed ceiling")
			} else {
				require.Error(t, prepare(test.ceiling))
			}
		})
	}
}

func TestPrepareSlotDataflowAndImmutableViews(t *testing.T) {
	c, catalog, p := fixture(t)
	c.Program.Slots = []*testpilotspb.Slot{valueSlot("result", scalar(testpilotspb.SCALAR_KIND_TEXT))}
	c.Program.Observations = []*testpilotspb.Observation{{ObservationId: "text", Type: scalar(testpilotspb.SCALAR_KIND_TEXT)}}
	producer := c.Program.Entrypoints[0].Instructions[0]
	producer.Instruction.GetInvokeRpc().ResponseReads = []*testpilotspb.ResponseRead{{Path: "text", Targets: []*testpilotspb.ReadTarget{{Target: &testpilotspb.ReadTarget_SlotId{SlotId: "result"}}, {Target: &testpilotspb.ReadTarget_ObservationId{ObservationId: "text"}}}}}
	consumer := rpcNode("consume")
	consumer.Guard = succeeded("controller", "call")
	consumer.Instruction.GetInvokeRpc().RequestAssignments = []*testpilotspb.RequestAssignment{{Target: "text", Value: slot("result")}}
	c.Program.Entrypoints[0].Instructions = append(c.Program.Entrypoints[0].Instructions, consumer)
	prepared, err := Prepare(c, catalog, p)
	require.NoError(t, err)
	for name, mutate := range map[string]func(*testpilotspb.Case){"runs regardless": func(s *testpilotspb.Case) { s.Program.Entrypoints[0].Instructions[1].Guard = alwaysRuns() }, "no dependency": func(s *testpilotspb.Case) { s.Program.Entrypoints[0].Instructions[1].After = runsAfter("controller") }, "second writer": func(s *testpilotspb.Case) {
		s.Program.Entrypoints[0].Instructions[1].Instruction.GetInvokeRpc().ResponseReads = proto.CloneOf(producer.Instruction).GetInvokeRpc().ResponseReads
	}, "assignment overlap": func(s *testpilotspb.Case) {
		rpc := s.Program.Entrypoints[0].Instructions[1].Instruction.GetInvokeRpc()
		rpc.RequestAssignments = append(rpc.RequestAssignments, proto.CloneOf(rpc.RequestAssignments[0]))
	}, "crossed cardinality": func(s *testpilotspb.Case) {
		s.Program.Entrypoints[0].Instructions[0].Instruction.GetInvokeRpc().ResponseReads[0].Path = "items[*]"
	}} {
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

// A carrier reserves one activation of every workflow and Nexus-handler entrypoint its shapes admit,
// in declaration order; an activity, a controller or a cleanup call reserves nothing.
func TestPrepareDerivesReservationsFromTheProfileCarriers(t *testing.T) {
	c, catalog, p := handleFixture(t)
	c.Program.Entrypoints = append(c.Program.Entrypoints, &testpilotspb.Entrypoint{EntrypointId: "activity", Activation: &testpilotspb.Entrypoint_Activity{Activity: &testpilotspb.ActivityActivation{ActivityType: "activity", WorkerRoleId: "worker", TaskQueueRoleId: "queue", AttemptNumbering: &testpilotspb.AttemptNumbering{First: 1, OneRun: true}}}})
	c.Program.Cleanup.Instructions = []*testpilotspb.InstructionNode{rpcNode("cleanup-call")}
	prepared, err := Prepare(c, catalog, p)
	require.NoError(t, err)
	want := []contract.ReservationTopology{{EntrypointID: "workflow", Kind: contract.WorkflowEntrypoint, Count: 1}, {EntrypointID: "handler", Kind: contract.NexusHandlerEntrypoint, Count: 1}}
	require.Equal(t, want, prepared.Entrypoints()[0].Instructions()[0].Reservations())
	require.Empty(t, prepared.Entrypoints()[0].Instructions()[1].Reservations())
	cleanup, ok := prepared.Cleanup()
	require.True(t, ok)
	require.Empty(t, cleanup.Instructions()[0].Reservations())

	// A carrier whose shapes admit only workflows leaves the handler to no one.
	p.Roles[0].ReservationCarriers[0].Shapes = p.Roles[0].ReservationCarriers[0].Shapes[:1]
	_, err = Prepare(c, catalog, p)
	require.Error(t, err)
}

// Two instructions that could both carry one entrypoint's reservation reject, naming both.
func TestPrepareRejectsAnAmbiguousReservationCarrier(t *testing.T) {
	for name, mutate := range map[string]func(*testpilotspb.Case){
		"same entrypoint": func(c *testpilotspb.Case) {
			second := rpcNode("call_second")
			c.Program.Entrypoints[0].Instructions = append(c.Program.Entrypoints[0].Instructions, second)
		},
		"another controller": func(c *testpilotspb.Case) {
			other := proto.CloneOf(c.Program.Entrypoints[0])
			other.EntrypointId = "other"
			other.Instructions[0].InstructionId = "call_second"
			c.Program.Entrypoints = append(c.Program.Entrypoints, other)
		},
	} {
		t.Run(name, func(t *testing.T) {
			c, catalog, p := fixture(t)
			addWorker(c, &p)
			mutate(c)
			_, err := Prepare(c, catalog, p)
			var diagnostic *ir.Error
			require.ErrorAs(t, err, &diagnostic)
			require.Equal(t, ir.Unsupported, diagnostic.Category)
			require.Equal(t, "instructions call and call_second both carry the reservation of entrypoint workflow", diagnostic.Detail)
		})
	}
}

func textLiteral(value string) *testpilotspb.Expression {
	return cel.Literal(&celpb.Value{Kind: &celpb.Value_StringValue{StringValue: value}})
}

func environment(id string) *testpilotspb.Expression {
	return cel.Ref(&testpilotspb.Reference{Reference: &testpilotspb.Reference_EnvironmentBindingId{EnvironmentBindingId: id}})
}

func TestPrepareResolvesClosedEnvironmentGraph(t *testing.T) {
	c, catalog, policy := fixture(t)
	c.Program.Roles = append(c.Program.Roles,
		&testpilotspb.Role{RoleId: "worker", Kind: testpilotspb.ROLE_KIND_WORKER, NamespaceBindingId: "namespace"},
		&testpilotspb.Role{RoleId: "queue", Kind: testpilotspb.ROLE_KIND_TASK_QUEUE, NamespaceBindingId: "namespace", ResourceBindingId: "queue"},
	)
	c.Program.Roles[0].ResourceBindingId = "queue"
	policy.EnvironmentBindings = []contract.EnvironmentBinding{{ID: "namespace", Value: "namespace-a"}, {ID: "queue", Value: "queue-a"}, {ID: "unused", Value: "allowed"}}
	c.Program.Entrypoints[0].Instructions[0].Instruction.GetInvokeRpc().RequestAssignments = []*testpilotspb.RequestAssignment{{Target: "text", Value: environment("namespace")}}

	prepared, err := Prepare(c, catalog, policy)
	require.NoError(t, err)
	require.Equal(t, "namespace-a", prepared.graphs[0].nodes[0].assignments[0].value.Literal().GetStringValue())
	require.Equal(t, contract.PreparedRole{ID: "endpoint", Kind: testpilotspb.ROLE_KIND_ENDPOINT, ResourceBindingID: "queue", Resource: "queue-a"}, prepared.roles["endpoint"])
	require.Equal(t, contract.PreparedRole{ID: "queue", Kind: testpilotspb.ROLE_KIND_TASK_QUEUE, NamespaceBindingID: "namespace", Namespace: "namespace-a", ResourceBindingID: "queue", Resource: "queue-a"}, prepared.roles["queue"])
	store, err := newValueStore(prepared, "run")
	require.NoError(t, err)
	values, err := store.activate("controller", "activation")
	require.NoError(t, err)
	request, dispatched, _, err := values.request(t.Context(), contract.Coordinate{RunID: "run", EntrypointID: "controller", ActivationID: "activation", InstructionID: "call", Attempt: 1}, prepared.graphs[0].runtimeWork)
	require.NoError(t, err)
	require.True(t, dispatched)
	textField := request.ProtoReflect().Descriptor().Fields().ByName("text")
	require.Equal(t, "namespace-a", request.ProtoReflect().Get(textField).String())

	c.Program.Roles[1].NamespaceBindingId = "changed"
	c.Program.Roles[2].ResourceBindingId = "changed"
	policy.EnvironmentBindings[0].Value = "changed"
	require.Equal(t, "namespace", prepared.Snapshot().Roles[1].NamespaceBindingId)
	require.Equal(t, "namespace", expressionReference(prepared.Snapshot().Entrypoints[0].Instructions[0].Instruction.GetInvokeRpc().RequestAssignments[0].Value).GetEnvironmentBindingId())
	require.Equal(t, "namespace-a", prepared.graphs[0].nodes[0].assignments[0].value.Literal().GetStringValue())
	require.Equal(t, "queue-a", prepared.roles["queue"].Resource)
}

// A Program's binding graph is every binding its roles and expressions reference, each once, roles
// first; preparation resolves exactly that set, and a referenced binding the Profile lacks rejects.
func TestPrepareDerivesTheEnvironmentBindingGraph(t *testing.T) {
	c, catalog, policy := fixture(t)
	addWorker(c, &policy)
	c.Program.Entrypoints[0].Instructions[0].Instruction.GetInvokeRpc().RequestAssignments = []*testpilotspb.RequestAssignment{{Target: "text", Value: environment("request")}}
	cleanup := rpcNode("cleanup-call")
	cleanup.Instruction.GetInvokeRpc().RequestAssignments = []*testpilotspb.RequestAssignment{{Target: "text", Value: environment("namespace")}}
	c.Program.Cleanup.Instructions = []*testpilotspb.InstructionNode{cleanup}
	require.Equal(t, []string{"namespace", "queue", "request"}, EnvironmentBindingIDs(c.Program))

	_, err := Prepare(c, catalog, policy)
	var diagnostic *ir.Error
	require.ErrorAs(t, err, &diagnostic)
	require.Equal(t, ir.Error{Category: ir.Unknown, Path: "environment", Detail: `environment binding "request" is not supplied by the Profile`}, *diagnostic)
	policy.EnvironmentBindings = append(policy.EnvironmentBindings, contract.EnvironmentBinding{ID: "request", Value: "request-value"})
	prepared, err := Prepare(c, catalog, policy)
	require.NoError(t, err)
	require.Equal(t, "request-value", prepared.graphs[0].nodes[0].assignments[0].value.Literal().GetStringValue())
}

// An instruction limit the Case omits takes the Profile's default, one it writes overrides it, and an
// omitted limit the Profile has no default for rejects.
func TestInstructionLimitsTakeTheProfileDefaults(t *testing.T) {
	c, catalog, p := fixture(t)
	c.Program.Entrypoints[0].Instructions[0].Limits = nil
	_, err := Prepare(c, catalog, p)
	var diagnostic *ir.Error
	require.ErrorAs(t, err, &diagnostic)
	require.Equal(t, ir.Error{Category: ir.Malformed, Path: "controller.call", Detail: "instruction writes no limit that neither the Program nor the Profile has a default for"}, *diagnostic)

	p.InstructionDefaults = contract.InstructionDefaults{TimeoutMilliseconds: 2000, MaxAttempts: 3}
	prepared, err := Prepare(c, catalog, p)
	require.NoError(t, err)
	plan := prepared.Entrypoints()[0].Instructions()[0]
	require.Equal(t, []int64{2000, 3}, []int64{plan.TimeoutMilliseconds(), plan.MaxAttempts()})

	c.Program.Entrypoints[0].Instructions[0].Limits = &testpilotspb.InstructionLimits{MaxAttempts: proto.Int64(1)}
	prepared, err = Prepare(c, catalog, p)
	require.NoError(t, err)
	plan = prepared.Entrypoints()[0].Instructions()[0]
	require.Equal(t, []int64{2000, 1}, []int64{plan.TimeoutMilliseconds(), plan.MaxAttempts()})

	for _, test := range []struct {
		name     string
		defaults contract.InstructionDefaults
		want     ir.Error
	}{
		{"negative", contract.InstructionDefaults{TimeoutMilliseconds: -1}, ir.Error{Category: ir.Malformed, Path: "policy.instruction_defaults", Detail: "negative instruction default"}},
		{"timeout above the ceiling", contract.InstructionDefaults{TimeoutMilliseconds: p.Limits.MaxDuration.AsDuration().Milliseconds() + 1}, ir.Error{Category: ir.LimitExceeded, Path: "policy.instruction_defaults", Detail: "instruction default exceeds the Profile ceiling"}},
		{"attempts above the ceiling", contract.InstructionDefaults{MaxAttempts: p.Limits.MaxAttempts + 1}, ir.Error{Category: ir.LimitExceeded, Path: "policy.instruction_defaults", Detail: "instruction default exceeds the Profile ceiling"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			policy := p
			policy.InstructionDefaults = test.defaults
			_, err := Prepare(c, catalog, policy)
			var diagnostic *ir.Error
			require.ErrorAs(t, err, &diagnostic)
			require.Equal(t, test.want, *diagnostic)
		})
	}
}

// A Program that declares instruction defaults gives an instruction that writes no limit its own:
// each default it declares takes the place of the Profile's, one it leaves out is still the
// Profile's, and a declared default is positive and within the Profile's ceiling.
func TestInstructionLimitsTakeTheProgramDefaults(t *testing.T) {
	c, catalog, p := fixture(t)
	c.Program.Entrypoints[0].Instructions[0].Limits = nil
	timeout := func(ms int64) *durationpb.Duration {
		return pbduration.FromMilliseconds(ms)
	}
	c.Program.InstructionDefaults = &testpilotspb.InstructionLimits{Timeout: timeout(4000), MaxAttempts: proto.Int64(1)}
	prepared, err := Prepare(c, catalog, p)
	require.NoError(t, err)
	plan := prepared.Entrypoints()[0].Instructions()[0]
	require.Equal(t, []int64{4000, 1}, []int64{plan.TimeoutMilliseconds(), plan.MaxAttempts()})

	p.InstructionDefaults = contract.InstructionDefaults{TimeoutMilliseconds: 2000, MaxAttempts: 3}
	c.Program.InstructionDefaults = &testpilotspb.InstructionLimits{Timeout: timeout(4000)}
	prepared, err = Prepare(c, catalog, p)
	require.NoError(t, err)
	plan = prepared.Entrypoints()[0].Instructions()[0]
	require.Equal(t, []int64{4000, 3}, []int64{plan.TimeoutMilliseconds(), plan.MaxAttempts()})

	for _, test := range []struct {
		name     string
		declared *testpilotspb.InstructionLimits
		want     ir.Error
	}{
		{"zero", &testpilotspb.InstructionLimits{Timeout: timeout(0)},
			ir.Error{Category: ir.Malformed, Path: "program.instruction_defaults", Detail: "a declared instruction default is positive"}},
		{"above the ceiling", &testpilotspb.InstructionLimits{Timeout: timeout(p.Limits.MaxDuration.AsDuration().Milliseconds() + 1)},
			ir.Error{Category: ir.LimitExceeded, Path: "program.instruction_defaults", Detail: "a declared instruction default exceeds the Profile ceiling"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			c.Program.InstructionDefaults = test.declared
			_, err := Prepare(c, catalog, p)
			var diagnostic *ir.Error
			require.ErrorAs(t, err, &diagnostic)
			require.Equal(t, test.want, *diagnostic)
		})
	}
}

func TestPrepareEnvironmentVersionAndClosure(t *testing.T) {
	for name, mutate := range map[string]func(*testpilotspb.Case, *Profile){
		"unsupported 1.1": func(c *testpilotspb.Case, _ *Profile) { c.Version.Minor = 1 },
		"reference the Profile lacks": func(c *testpilotspb.Case, p *Profile) {
			configureEnvironmentCase(c, p)
			c.Program.Entrypoints[0].Instructions[0].Instruction.GetInvokeRpc().RequestAssignments[0].Value = environment("missing")
		},
		"missing profile value": func(c *testpilotspb.Case, p *Profile) {
			configureEnvironmentCase(c, p)
			p.EnvironmentBindings = nil
		},
		"nested reference": func(c *testpilotspb.Case, p *Profile) {
			configureEnvironmentCase(c, p)
			c.Program.Entrypoints[0].Instructions[0].Guard = cel.Compare("_==_", environment("binding"), textLiteral("value"))
		},
		"non-text destination": func(c *testpilotspb.Case, p *Profile) {
			configureEnvironmentCase(c, p)
			c.Program.Entrypoints[0].Instructions[0].Instruction.GetInvokeRpc().RequestAssignments[0].Target = "items"
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

func configureEnvironmentCase(c *testpilotspb.Case, policy *Profile) {
	policy.EnvironmentBindings = []contract.EnvironmentBinding{{ID: "binding", Value: "value"}}
	c.Program.Entrypoints[0].Instructions[0].Instruction.GetInvokeRpc().RequestAssignments = []*testpilotspb.RequestAssignment{{Target: "text", Value: environment("binding")}}
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
	c.Program.Entrypoints[0].Instructions[0].Instruction.GetInvokeRpc().RequestAssignments[0].Value = environment("changed")
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
			{"complete", completionNode("complete", payloadCompletion("handle")).Instruction, contract.ControllerEntrypoint},
			{"start", scheduleNode("start").Instruction, contract.WorkflowEntrypoint},
			{"await", &testpilotspb.Instruction{Instruction: &testpilotspb.Instruction_AwaitInstruction{AwaitInstruction: &testpilotspb.AwaitInstruction{Instruction: testsupport.LocalReference(&testpilotspb.InstructionReference{EntrypointId: "workflow", InstructionId: "prior"})}}}, contract.WorkflowEntrypoint},
			{"finish", &testpilotspb.Instruction{Instruction: &testpilotspb.Instruction_Finish{Finish: &testpilotspb.Finish{Result: textLiteral("done")}}}, contract.WorkflowEntrypoint},
			{"respond", replyNode("respond", syncReply()).Instruction, contract.NexusHandlerEntrypoint},
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
					g.Activation = &testpilotspb.Entrypoint_Activity{Activity: &testpilotspb.ActivityActivation{ActivityType: "activity", WorkerRoleId: "worker", TaskQueueRoleId: "queue", AttemptNumbering: &testpilotspb.AttemptNumbering{First: 1, OneRun: true}}}
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

func handleFixture(t *testing.T) (*testpilotspb.Case, *ir.Catalog, Profile) {
	t.Helper()
	c, catalog, p := fixture(t)
	addWorker(c, &p)
	c.Program.Slots = []*testpilotspb.Slot{handleSlot("handle")}
	wait := rpcNode("ready")
	wait.Instruction = &testpilotspb.Instruction{Instruction: &testpilotspb.Instruction_AwaitSlot{AwaitSlot: &testpilotspb.AwaitSlot{SlotId: "handle"}}}
	wait.After = runsAfter("controller")
	complete := completionNode("complete", payloadCompletion("handle"))
	complete.Guard = succeeded("controller", "ready")
	c.Program.Entrypoints[0].Instructions = append(c.Program.Entrypoints[0].Instructions, wait, complete)
	handler := replyNode("respond", asyncReply("handle"))
	c.Program.Entrypoints = append(c.Program.Entrypoints, &testpilotspb.Entrypoint{EntrypointId: "handler", Activation: &testpilotspb.Entrypoint_NexusHandler{NexusHandler: &testpilotspb.NexusHandlerActivation{Service: "service", Operation: "operation", WorkerRoleId: "worker", TaskQueueRoleId: "queue"}}, Instructions: []*testpilotspb.InstructionNode{handler}})
	start := scheduleNode("start")
	await := rpcNode("await")
	await.Guard = alwaysRuns()
	await.Instruction = &testpilotspb.Instruction{Instruction: &testpilotspb.Instruction_AwaitInstruction{AwaitInstruction: &testpilotspb.AwaitInstruction{Instruction: testsupport.LocalReference(&testpilotspb.InstructionReference{EntrypointId: "workflow", InstructionId: "start"})}}}
	finish := rpcNode("finish")
	finish.Guard = succeeded("workflow", "await")
	finish.Instruction = &testpilotspb.Instruction{Instruction: &testpilotspb.Instruction_Finish{Finish: &testpilotspb.Finish{Result: cel.Ref(&testpilotspb.Reference{Reference: &testpilotspb.Reference_Outcome{Outcome: &testpilotspb.InstructionOutcomeReference{Instruction: &testpilotspb.InstructionReference{EntrypointId: "workflow", InstructionId: "await"}, Field: testpilotspb.INSTRUCTION_OUTCOME_FIELD_VALUE}}})}}}
	c.Program.Entrypoints[1].Instructions = []*testpilotspb.InstructionNode{start, await, finish}
	return c, catalog, p
}
func TestOpaqueReadinessAndSDKPreparedPlans(t *testing.T) {
	c, catalog, p := handleFixture(t)
	prepared, err := Prepare(c, catalog, p)
	require.NoError(t, err)
	for name, mutate := range map[string]func(*testpilotspb.Case){
		"inspect handle": func(s *testpilotspb.Case) {
			s.Program.Entrypoints[0].Instructions[2].Guard = present(slot("handle"))
		},
		"consume without readiness": func(s *testpilotspb.Case) { s.Program.Entrypoints[0].Instructions[2].Guard = alwaysRuns() },
		"missing handle writer": func(s *testpilotspb.Case) {
			s.Program.Entrypoints = s.Program.Entrypoints[:2]
		},
		"handle response read": func(s *testpilotspb.Case) {
			s.Program.Entrypoints[0].Instructions[0].Instruction.GetInvokeRpc().ResponseReads = []*testpilotspb.ResponseRead{{Path: "text", Targets: []*testpilotspb.ReadTarget{{Target: &testpilotspb.ReadTarget_SlotId{SlotId: "handle"}}}}}
		},
		"SDK value without success": func(s *testpilotspb.Case) { s.Program.Entrypoints[1].Instructions[2].Guard = alwaysRuns() },
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
	scheduled := func(plan InstructionPlan) *commandpb.ScheduleNexusOperationCommandAttributes {
		return plan.Source().GetInstruction().GetWorkflowCommand().GetCommand().GetScheduleNexusOperationCommandAttributes()
	}
	require.Equal(t, "operation", scheduled(instructions[0]).GetOperation())
	require.Equal(t, []int{0}, instructions[1].node.dependencies)
	scheduled(instructions[0]).Operation = "changed"
	instructions[0].Source().Instruction = nil
	worker.Activation().GetWorkflow().WorkflowType = "changed"
	worker.Order()[0] = 99
	require.Equal(t, "operation", scheduled(worker.Instructions()[0]).GetOperation())
	require.Equal(t, "flow", worker.Activation().GetWorkflow().WorkflowType)
	require.Equal(t, []int{0, 1, 2}, worker.Order())
}

func TestOutcomeStatusesAndCleanupLocalReferences(t *testing.T) {
	c, catalog, p := fixture(t)
	first := rpcNode("release")
	second := rpcNode("confirm")
	second.Guard = succeeded("cleanup", "release")
	c.Program.Cleanup.Instructions = []*testpilotspb.InstructionNode{first, second}
	for status := int32(1); status <= 5; status++ {
		second.Guard = cel.Compare("_==_", cel.Ref(&testpilotspb.Reference{Reference: &testpilotspb.Reference_Outcome{Outcome: &testpilotspb.InstructionOutcomeReference{Instruction: &testpilotspb.InstructionReference{EntrypointId: "cleanup", InstructionId: "release"}, Field: testpilotspb.INSTRUCTION_OUTCOME_FIELD_STATUS}}}), cel.Literal(cel.Enum(testpilotspb.InstructionOutcomeStatus(status))))
		_, err := Prepare(c, catalog, p)
		require.NoError(t, err)
	}
	second.Guard = cel.Compare("_==_", cel.Ref(&testpilotspb.Reference{Reference: &testpilotspb.Reference_Outcome{Outcome: &testpilotspb.InstructionOutcomeReference{Instruction: &testpilotspb.InstructionReference{EntrypointId: "cleanup", InstructionId: "release"}, Field: testpilotspb.INSTRUCTION_OUTCOME_FIELD_STATUS}}}), cel.Literal(cel.Enum(testpilotspb.InstructionOutcomeStatus(99))))
	_, err := Prepare(c, catalog, p)
	require.Error(t, err)
	second.Guard = succeeded("controller", "call")
	_, err = Prepare(c, catalog, p)
	require.Error(t, err)
}

func TestSlotOwnersAndConcurrentPreparedViews(t *testing.T) {
	c, catalog, p := fixture(t)
	c.Program.Slots = []*testpilotspb.Slot{valueSlot("value", scalar(testpilotspb.SCALAR_KIND_TEXT))}
	c.Program.Entrypoints[0].Instructions[0].Instruction.GetInvokeRpc().ResponseReads = []*testpilotspb.ResponseRead{{Path: "text", Targets: []*testpilotspb.ReadTarget{{Target: &testpilotspb.ReadTarget_SlotId{SlotId: "value"}}}}}
	other := proto.CloneOf(c.Program.Entrypoints[0])
	other.EntrypointId = "other"
	other.Instructions[0].Instruction.GetInvokeRpc().ResponseReads = nil
	other.Instructions[0].Guard = present(slot("value"))
	other.Instructions[0].Instruction.GetInvokeRpc().RequestAssignments = []*testpilotspb.RequestAssignment{{Target: "text", Value: slot("value")}}
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
				plans[0].Instructions()[0].Source().GetInstruction().GetInvokeRpc().ResponseReads[0].Targets[0].Target = &testpilotspb.ReadTarget_SlotId{SlotId: "changed"}
				plans[0].Order()[0] = 99
				prepared.Snapshot().ProgramId = "changed"
				require.Equal(t, "value", prepared.Entrypoints()[0].Instructions()[0].Source().GetInstruction().GetInvokeRpc().ResponseReads[0].Targets[0].GetSlotId())
				require.Equal(t, "program", prepared.View().ProgramID())
			}
		})
	}
}

func TestPrepareBoundsSurfaceBeforeCloning(t *testing.T) {
	c, catalog, p := fixture(t)
	root := &celpb.Expr{Id: 1}
	root.ExprKind = &celpb.Expr_CallExpr{CallExpr: &celpb.Expr_Call{Function: "!_", Args: []*celpb.Expr{root}}}
	expression := &testpilotspb.Expression{Cel: &celpb.ParsedExpr{Expr: root}}
	c.Program.Entrypoints[0].Instructions[0].Guard = expression
	_, err := Prepare(c, catalog, p)
	require.Error(t, err)
	c, catalog, p = fixture(t)
	c.Program.Entrypoints[0].Instructions[0].Instruction.Instruction = (*testpilotspb.Instruction_InvokeRpc)(nil)
	_, err = Prepare(c, catalog, p)
	require.Error(t, err)
}

// A Case is charged its surface as its Contract's Rule instances expand, so a Case whose instanced
// Rule fits but whose expansion does not is rejected before the Contract is prepared.
func TestPrepareBoundsTheCaseSurfaceAsExpanded(t *testing.T) {
	c, catalog, p := fixture(t)
	large := strings.Repeat("x", 6<<20)
	read := cel.Ref(&testpilotspb.Reference{Reference: &testpilotspb.Reference_InstanceValueId{InstanceValueId: "op"}})
	instance := func(ruleID, value string) *testpilotspb.ContractRuleInstance {
		return &testpilotspb.ContractRuleInstance{RuleId: ruleID, Assignments: []*testpilotspb.ContractInstanceAssignment{{InstanceValueId: "op", Value: expressionLiteral(textLiteral(value))}}}
	}
	c.Contract.Rules = []*testpilotspb.ContractRule{{
		RuleId:         "rule",
		Transitions:    []*testpilotspb.ContractTransition{{TransitionId: "t", Predicate: cel.All([]*testpilotspb.Expression{read, proto.CloneOf(read)}...)}},
		InstanceValues: []*testpilotspb.ContractInstanceValue{{InstanceValueId: "op", Type: scalar(testpilotspb.SCALAR_KIND_TEXT).GetSingular()}},
		Instances:      []*testpilotspb.ContractRuleInstance{instance("rule-1", large), instance("rule-2", large+"y")},
	}}
	require.NoError(t, ir.CheckSurface(c, ir.DefaultLimits()), "the Case as written fits")
	_, err := Prepare(c, catalog, p)
	var diagnostic *ir.Error
	require.ErrorAs(t, err, &diagnostic)
	require.Equal(t, ir.LimitExceeded, diagnostic.Category)
	require.True(t, strings.HasPrefix(diagnostic.Path, "$.contract.rules"), diagnostic.Path)
}

func TestStructuralCountsAndPathFanout(t *testing.T) {
	for name, mutate := range map[string]func(*testpilotspb.Case, *Profile){
		"entrypoint count": func(c *testpilotspb.Case, p *Profile) { addWorker(c, p); p.Limits.MaxEntrypoints = 1 },
		"node count": func(c *testpilotspb.Case, p *Profile) {
			c.Program.Entrypoints[0].Instructions = append(c.Program.Entrypoints[0].Instructions, rpcNode("other"))
			p.Limits.MaxNodes = 1
		},
		"edge count": func(c *testpilotspb.Case, p *Profile) {
			last := rpcNode("last")
			last.After = runsAfter("controller", "call", "other")
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
	n.Instruction.GetInvokeRpc().ResponseReads = []*testpilotspb.ResponseRead{{Path: "items[*]", Targets: []*testpilotspb.ReadTarget{{Target: &testpilotspb.ReadTarget_ObservationId{ObservationId: "item"}}}}}
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
	producer.Instruction.GetInvokeRpc().ResponseReads = []*testpilotspb.ResponseRead{{Targets: []*testpilotspb.ReadTarget{{Target: &testpilotspb.ReadTarget_SlotId{SlotId: "request"}}}}}
	consumer := rpcNode("copy")
	consumer.Guard = succeeded("controller", "call")
	consumer.Instruction.GetInvokeRpc().RequestAssignments = []*testpilotspb.RequestAssignment{{Value: slot("request")}}
	c.Program.Entrypoints[0].Instructions = append(c.Program.Entrypoints[0].Instructions, consumer)
	_, err := Prepare(c, catalog, p)
	require.NoError(t, err)
	consumer.Instruction.GetInvokeRpc().RequestAssignments = append(consumer.Instruction.GetInvokeRpc().RequestAssignments, &testpilotspb.RequestAssignment{Target: "text", Value: textLiteral("conflict")})
	_, err = Prepare(c, catalog, p)
	require.Error(t, err)
}

func TestAwaitRequiresAScheduleCommand(t *testing.T) {
	for _, target := range []string{"start", "await", "finish"} {
		t.Run(target, func(t *testing.T) {
			c, catalog, p := handleFixture(t)
			g := c.Program.Entrypoints[1]
			n := rpcNode("second_await")
			n.After = runsAfter("workflow", target)
			n.Instruction = &testpilotspb.Instruction{Instruction: &testpilotspb.Instruction_AwaitInstruction{AwaitInstruction: &testpilotspb.AwaitInstruction{Instruction: proto.CloneOf(n.After.Instructions[0])}}}
			g.Instructions[0].After = runsAfter("workflow")
			g.Instructions = append([]*testpilotspb.InstructionNode{n}, g.Instructions...)
			_, err := Prepare(c, catalog, p)
			if target == "start" {
				require.NoError(t, err)
			} else {
				require.ErrorContains(t, err, "Nexus or activity schedule command")
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
