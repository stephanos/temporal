package verification

import (
	"context"
	"fmt"
	"slices"
	"testing"

	"github.com/stretchr/testify/require"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/testpilot/internal/execution"
	"go.temporal.io/server/common/testing/testpilot/internal/ir"
	"go.temporal.io/server/common/testing/testpilot/internal/testsupport"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/dynamicpb"
	"google.golang.org/protobuf/types/known/anypb"
)

func fixture(t *testing.T, responseBytes ...int64) (*testpilotspb.Contract, *ir.Catalog, execution.ProgramView, *testpilotspb.ContractLimits) {
	t.Helper()
	catalog, err := ir.NewCatalog(&descriptorpb.FileDescriptorSet{File: []*descriptorpb.FileDescriptorProto{{Name: proto.String("contract.proto"), Package: proto.String("example"), Syntax: proto.String("proto3"), MessageType: []*descriptorpb.DescriptorProto{{Name: proto.String("Empty"), Field: []*descriptorpb.FieldDescriptorProto{{Name: proto.String("items"), Number: proto.Int32(1), Label: descriptorpb.FieldDescriptorProto_LABEL_REPEATED.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum()}}}}}}})
	require.NoError(t, err)
	limits := testsupport.ProgramLimits()
	if len(responseBytes) > 0 {
		limits.MaxResponseBytes = responseBytes[0]
		limits.MaxInstructionResponseBytes = min(limits.MaxInstructionResponseBytes, responseBytes[0])
	}
	source := &testpilotspb.Case{Version: &testpilotspb.FormatVersion{Major: 1}, CaseId: "case", Contract: &testpilotspb.Contract{ContractId: "contract"}, Program: &testpilotspb.Program{ProgramId: "program", Observations: []*testpilotspb.Observation{{ObservationId: "id", Type: scalar(testpilotspb.SCALAR_KIND_INT64)}, {ObservationId: "text", Type: scalar(testpilotspb.SCALAR_KIND_TEXT)}, {ObservationId: "flag", Type: scalar(testpilotspb.SCALAR_KIND_BOOLEAN)}, {ObservationId: "message", Type: messageType("example.Empty")}}, Entrypoints: []*testpilotspb.Entrypoint{{EntrypointId: "controller", Activation: &testpilotspb.Entrypoint_Controller{Controller: &testpilotspb.ControllerActivation{}}}}, Cleanup: &testpilotspb.Cleanup{EntrypointId: "cleanup"}}}
	prepared, err := execution.Prepare(source, catalog, execution.Profile{Identity: "host", CatalogIdentity: catalog.Identity(), Limits: proto.CloneOf(limits)})
	require.NoError(t, err)
	ceiling := &testpilotspb.ContractLimits{MaxRules: 16, MaxStates: 32, MaxTransitions: 64, MaxExpressionDepth: 16, MaxWorkPerEvent: 100000, MaxTotalWork: 1000000000, MaxCaptures: 8, MaxCaptureBytes: 65536}
	contract := &testpilotspb.Contract{ContractId: "contract", Rules: []*testpilotspb.ContractRule{{RuleId: "rule", Kind: testpilotspb.CONTRACT_RULE_KIND_SAFETY, InitialStateId: "start", States: []*testpilotspb.ContractState{{StateId: "start", Status: testpilotspb.CONTRACT_STATE_STATUS_PENDING}, {StateId: "good", Status: testpilotspb.CONTRACT_STATE_STATUS_SATISFIED}, {StateId: "bad", Status: testpilotspb.CONTRACT_STATE_STATUS_VIOLATED}}, Transitions: []*testpilotspb.ContractTransition{transition("first", "start", "good", boolean(true))}}}}
	return contract, catalog, prepared.View(), ceiling
}
func scalar(kind testpilotspb.ScalarKind) *testpilotspb.ValueType {
	return &testpilotspb.ValueType{Shape: &testpilotspb.ValueType_Singular{Singular: &testpilotspb.SingularType{Type: &testpilotspb.SingularType_Scalar{Scalar: &testpilotspb.ScalarType{Kind: kind}}}}}
}
func messageType(name string) *testpilotspb.ValueType {
	return &testpilotspb.ValueType{Shape: &testpilotspb.ValueType_Singular{Singular: &testpilotspb.SingularType{Type: &testpilotspb.SingularType_Message{Message: &testpilotspb.NamedType{ProtobufType: name}}}}}
}
func boolean(value bool) *testpilotspb.Expression {
	return &testpilotspb.Expression{Expression: &testpilotspb.Expression_Literal{Literal: &testpilotspb.Value{Value: &testpilotspb.Value_BoolValue{BoolValue: value}}}}
}
func observation(id string) *testpilotspb.Expression {
	return &testpilotspb.Expression{Expression: &testpilotspb.Expression_Reference{Reference: &testpilotspb.Reference{Reference: &testpilotspb.Reference_ObservationId{ObservationId: id}}}}
}
func capture(id string) *testpilotspb.Expression {
	return &testpilotspb.Expression{Expression: &testpilotspb.Expression_Reference{Reference: &testpilotspb.Reference{Reference: &testpilotspb.Reference_CaptureId{CaptureId: id}}}}
}
func present(value *testpilotspb.Expression) *testpilotspb.Expression {
	return &testpilotspb.Expression{Expression: &testpilotspb.Expression_Present{Present: &testpilotspb.PresentExpression{Operand: value}}}
}
func all(values ...*testpilotspb.Expression) *testpilotspb.Expression {
	return &testpilotspb.Expression{Expression: &testpilotspb.Expression_All{All: &testpilotspb.AllExpression{Operands: values}}}
}
func not(value *testpilotspb.Expression) *testpilotspb.Expression {
	return &testpilotspb.Expression{Expression: &testpilotspb.Expression_Not{Not: &testpilotspb.NotExpression{Operand: value}}}
}
func equal(left, right *testpilotspb.Expression) *testpilotspb.Expression {
	return &testpilotspb.Expression{Expression: &testpilotspb.Expression_Compare{Compare: &testpilotspb.CompareExpression{Operator: testpilotspb.COMPARISON_OPERATOR_EQUAL, Left: left, Right: right}}}
}
func transition(id, from, to string, predicate *testpilotspb.Expression) *testpilotspb.ContractTransition {
	return &testpilotspb.ContractTransition{TransitionId: id, SourceStateId: from, TargetStateId: to, Predicate: predicate, EventFilter: &testpilotspb.RunEventFilter{Kinds: []testpilotspb.RunEventKind{testpilotspb.RUN_EVENT_KIND_INSTRUCTION_COMPLETED}}, SupportKind: testpilotspb.CONTRACT_SUPPORT_KIND_MATCHING_EVENT}
}
func addCapture(rule *testpilotspb.ContractRule) {
	rule.Captures = []*testpilotspb.ContractCapture{{CaptureId: "saved", Type: &testpilotspb.SingularType{Type: &testpilotspb.SingularType_Scalar{Scalar: &testpilotspb.ScalarType{Kind: testpilotspb.SCALAR_KIND_INT64}}}}}
}

// addFlag declares a boolean capture the save transition assigns from the flag Observation.
func addFlag(rule *testpilotspb.ContractRule) {
	rule.Captures = append(rule.Captures, &testpilotspb.ContractCapture{CaptureId: "flag", Type: &testpilotspb.SingularType{Type: &testpilotspb.SingularType_Scalar{Scalar: &testpilotspb.ScalarType{Kind: testpilotspb.SCALAR_KIND_BOOLEAN}}}})
	save := rule.Transitions[0]
	save.Predicate = all(save.Predicate, present(observation("flag")))
	save.CaptureAssignments = append(save.CaptureAssignments, &testpilotspb.ContractCaptureAssignment{CaptureId: "flag", ObservationId: "flag"})
}
func assign(tr *testpilotspb.ContractTransition) {
	tr.CaptureAssignments = []*testpilotspb.ContractCaptureAssignment{{CaptureId: "saved", ObservationId: "id"}}
}

func TestPrepareMachinesAndOrder(t *testing.T) {
	for _, live := range []bool{false, true} {
		t.Run(map[bool]string{false: "safety", true: "liveness"}[live], func(t *testing.T) {
			c, catalog, view, policy := fixture(t)
			r := c.Rules[0]
			r.Transitions = append(r.Transitions, transition("second", "start", "bad", boolean(true)))
			if live {
				r.Kind = testpilotspb.CONTRACT_RULE_KIND_BOUNDED_LIVENESS
				r.Deadline = &testpilotspb.Deadline{ViolationStateId: "bad", Bound: &testpilotspb.Deadline_ElapsedMilliseconds{ElapsedMilliseconds: 1000}}
			}
			prepared, err := Prepare(c, catalog, view, policy, nil)
			require.NoError(t, err)
			require.Equal(t, []int{0, 1}, prepared.rules[0].outgoing[0][testpilotspb.RUN_EVENT_KIND_INSTRUCTION_COMPLETED])
		})
	}
}
func TestPrepareCapturePaths(t *testing.T) {
	for _, test := range []struct {
		name      string
		mutate    func(*testpilotspb.ContractRule)
		wantError bool
	}{
		{"correlation", func(r *testpilotspb.ContractRule) {}, false},
		{"missing observation guard", func(r *testpilotspb.ContractRule) { r.Transitions[0].Predicate = boolean(true) }, true},
		// A comparison with a capture its path may not have assigned is false rather than rejected, so
		// definite assignment is observed through a bare boolean capture.
		{"pretransition comparison", func(r *testpilotspb.ContractRule) {
			r.Transitions[0].Predicate = all(present(observation("id")), equal(capture("saved"), observation("id")))
		}, false},
		{"pretransition read", func(r *testpilotspb.ContractRule) {
			addFlag(r)
			r.Transitions[0].Predicate = all(present(observation("id")), present(observation("flag")), capture("flag"))
		}, true},
		{"mismatched capture", func(r *testpilotspb.ContractRule) {
			r.Captures[0].Type.GetScalar().Kind = testpilotspb.SCALAR_KIND_TEXT
		}, true},
		{"support required", func(r *testpilotspb.ContractRule) {
			r.Transitions[0].SupportKind = testpilotspb.CONTRACT_SUPPORT_KIND_NONE
		}, true},
		{"unsafe cycle", func(r *testpilotspb.ContractRule) { r.Transitions[0].TargetStateId = "start" }, true},
		{"safe cycle", func(r *testpilotspb.ContractRule) {
			r.Transitions[0].TargetStateId = "start"
			r.Transitions[0].Predicate = all(not(present(capture("saved"))), present(observation("id")))
		}, false},
		{"branch merge comparison", func(r *testpilotspb.ContractRule) {
			r.Transitions = append(r.Transitions, transition("branch", "start", "middle", boolean(true)))
		}, false},
		{"branch merge", func(r *testpilotspb.ContractRule) {
			addFlag(r)
			r.Transitions[1].Predicate = all(present(observation("id")), capture("flag"))
			r.Transitions = append(r.Transitions, transition("branch", "start", "middle", boolean(true)))
		}, true},
		{"guarded flag", func(r *testpilotspb.ContractRule) {
			addFlag(r)
			r.Transitions[1].Predicate = all(present(observation("id")), capture("flag"))
		}, false},
		{"guarded merge", func(r *testpilotspb.ContractRule) {
			r.Transitions = append(r.Transitions, transition("branch", "start", "middle", boolean(true)))
			r.Transitions[1].Predicate = all(present(capture("saved")), r.Transitions[1].Predicate)
		}, false},
		{"repeated assignment", func(r *testpilotspb.ContractRule) { assign(r.Transitions[1]) }, true},
		{"duplicate atomic assignment", func(r *testpilotspb.ContractRule) {
			r.Transitions[0].CaptureAssignments = append(r.Transitions[0].CaptureAssignments, proto.CloneOf(r.Transitions[0].CaptureAssignments[0]))
		}, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			c, catalog, view, policy := fixture(t)
			r := c.Rules[0]
			addCapture(r)
			r.States = append(r.States, &testpilotspb.ContractState{StateId: "middle", Status: testpilotspb.CONTRACT_STATE_STATUS_PENDING})
			r.Transitions = []*testpilotspb.ContractTransition{transition("save", "start", "middle", present(observation("id"))), transition("compare", "middle", "good", all(present(observation("id")), equal(observation("id"), capture("saved"))))}
			assign(r.Transitions[0])
			test.mutate(r)
			_, err := Prepare(c, catalog, view, policy, nil)
			if test.wantError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestPrepareRejectsMalformedContracts(t *testing.T) {
	for name, mutate := range map[string]func(*testpilotspb.Contract){
		"duplicate rule": func(c *testpilotspb.Contract) { c.Rules = append(c.Rules, proto.CloneOf(c.Rules[0])) },
		"missing states": func(c *testpilotspb.Contract) { c.Rules[0].States = nil },
		"duplicate state": func(c *testpilotspb.Contract) {
			c.Rules[0].States = append(c.Rules[0].States, proto.CloneOf(c.Rules[0].States[0]))
		},
		"duplicate transition": func(c *testpilotspb.Contract) {
			c.Rules[0].Transitions = append(c.Rules[0].Transitions, proto.CloneOf(c.Rules[0].Transitions[0]))
		},
		"missing transitions":  func(c *testpilotspb.Contract) { c.Rules[0].Transitions = nil },
		"missing initial":      func(c *testpilotspb.Contract) { c.Rules[0].InitialStateId = "missing" },
		"terminal initial":     func(c *testpilotspb.Contract) { c.Rules[0].InitialStateId = "bad" },
		"missing target":       func(c *testpilotspb.Contract) { c.Rules[0].Transitions[0].TargetStateId = "missing" },
		"terminal source":      func(c *testpilotspb.Contract) { c.Rules[0].Transitions[0].SourceStateId = "good" },
		"unspecified terminal": func(c *testpilotspb.Contract) { c.Rules[0].States[0].Status = 0 },
		"unknown kind":         func(c *testpilotspb.Contract) { c.Rules[0].Kind = 99 },
		"missing deadline":     func(c *testpilotspb.Contract) { c.Rules[0].Kind = testpilotspb.CONTRACT_RULE_KIND_BOUNDED_LIVENESS },
		"wrong expiry target": func(c *testpilotspb.Contract) {
			c.Rules[0].Kind = testpilotspb.CONTRACT_RULE_KIND_BOUNDED_LIVENESS
			c.Rules[0].Deadline = &testpilotspb.Deadline{ViolationStateId: "good", Bound: &testpilotspb.Deadline_ElapsedMilliseconds{ElapsedMilliseconds: 1}}
		},
		"negative deadline": func(c *testpilotspb.Contract) {
			c.Rules[0].Kind = testpilotspb.CONTRACT_RULE_KIND_BOUNDED_LIVENESS
			c.Rules[0].Deadline = &testpilotspb.Deadline{ViolationStateId: "bad", Bound: &testpilotspb.Deadline_ElapsedMilliseconds{ElapsedMilliseconds: -1}}
		},
		"safety deadline": func(c *testpilotspb.Contract) {
			c.Rules[0].Deadline = &testpilotspb.Deadline{ViolationStateId: "bad", Bound: &testpilotspb.Deadline_ElapsedMilliseconds{ElapsedMilliseconds: 1}}
		},
		"unknown observation": func(c *testpilotspb.Contract) { c.Rules[0].Transitions[0].Predicate = present(observation("missing")) },
		"Run ID intrinsic forbidden": func(c *testpilotspb.Contract) {
			c.Rules[0].Transitions[0].Predicate = present(&testpilotspb.Expression{Expression: &testpilotspb.Expression_Reference{Reference: &testpilotspb.Reference{Reference: &testpilotspb.Reference_RunEvent{RunEvent: &testpilotspb.RunEventReference{Selection: &testpilotspb.RunEventReference_Field{Field: testpilotspb.RUN_EVENT_FIELD_RUN_ID}}}}}})
		},
		"nil predicate":        func(c *testpilotspb.Contract) { c.Rules[0].Transitions[0].Predicate = nil },
		"nonboolean predicate": func(c *testpilotspb.Contract) { c.Rules[0].Transitions[0].Predicate = observation("text") },
		"unknown expression": func(c *testpilotspb.Contract) {
			c.Rules[0].Transitions[0].Predicate.ProtoReflect().SetUnknown([]byte{0x80, 0x06, 1})
		},
		"unknown event": func(c *testpilotspb.Contract) {
			c.Rules[0].Transitions[0].EventFilter.Kinds = []testpilotspb.RunEventKind{99}
		},
		"duplicate event": func(c *testpilotspb.Contract) {
			tr := c.Rules[0].Transitions[0]
			tr.EventFilter.Kinds = append(tr.EventFilter.Kinds, tr.EventFilter.Kinds[0])
		},
		"nil state":   func(c *testpilotspb.Contract) { c.Rules[0].States[0] = nil },
		"nil capture": func(c *testpilotspb.Contract) { c.Rules[0].Captures = []*testpilotspb.ContractCapture{nil} },
	} {
		t.Run(name, func(t *testing.T) {
			c, catalog, view, policy := fixture(t)
			mutate(c)
			_, err := Prepare(c, catalog, view, policy, nil)
			require.Error(t, err)
		})
	}
}

// A Contract predicate shares the one expression language, so each Program, correlated and
// evidence-lift reference rejects at preparation at the predicate's located path.
func TestPrepareLocatesAReferenceOutsideTheContractContext(t *testing.T) {
	for name, value := range map[string]*testpilotspb.Reference{
		"slot_id": {Reference: &testpilotspb.Reference_SlotId{SlotId: "slot"}},
		"outcome": {Reference: &testpilotspb.Reference_Outcome{Outcome: &testpilotspb.InstructionOutcomeReference{
			Instruction: &testpilotspb.InstructionReference{EntrypointId: "controller", InstructionId: "call"}, Field: testpilotspb.INSTRUCTION_OUTCOME_FIELD_STATUS,
		}}},
		"run":                    {Reference: &testpilotspb.Reference_Run{Run: &testpilotspb.RunReference{}}},
		"environment_binding_id": {Reference: &testpilotspb.Reference_EnvironmentBindingId{EnvironmentBindingId: "namespace"}},
		"evidence_field_id":      {Reference: &testpilotspb.Reference_EvidenceFieldId{EvidenceFieldId: "field"}},
		"correlated_capture":     {Reference: &testpilotspb.Reference_CorrelatedCapture{CorrelatedCapture: &testpilotspb.CorrelatedCaptureReference{CaptureId: "capture"}}},
		"model_value":            {Reference: &testpilotspb.Reference_ModelValue{ModelValue: &testpilotspb.ModelValue{DefinitionId: "definition", Value: "value"}}},
		"correlated_step":        {Reference: &testpilotspb.Reference_CorrelatedStep{CorrelatedStep: &testpilotspb.CorrelatedStepReference{Field: testpilotspb.CORRELATED_STEP_FIELD_ACTION, DefinitionId: "definition"}}},
		"projected_value":        {Reference: &testpilotspb.Reference_ProjectedValue{ProjectedValue: &testpilotspb.ProjectedValueReference{}}},
	} {
		t.Run(name, func(t *testing.T) {
			c, catalog, view, policy := fixture(t)
			c.Rules[0].Transitions[0].Predicate = all(boolean(true), present(&testpilotspb.Expression{Expression: &testpilotspb.Expression_Reference{Reference: value}}))
			_, err := Prepare(c, catalog, view, policy, nil)
			var diagnostic *ir.Error
			require.ErrorAs(t, err, &diagnostic)
			require.Equal(t, &ir.Error{
				Category: ir.Unknown,
				Path:     "contract.rules[rule].transitions[first].predicate.all[1].present.reference." + name,
				Detail:   "reference is not admitted in this expression context",
			}, diagnostic)
		})
	}
}

// A capture holds a scalar, enum or message value; any other singular type rejects at preparation
// located at the capture.
func TestPrepareLocatesCaptureTypesOutsideScalarEnumOrMessage(t *testing.T) {
	for name, typ := range map[string]*testpilotspb.SingularType{
		"any":   {Type: &testpilotspb.SingularType_Any{Any: &testpilotspb.AnyType{}}},
		"unset": nil,
	} {
		t.Run(name, func(t *testing.T) {
			c, catalog, view, policy := fixture(t)
			c.Rules[0].Captures = []*testpilotspb.ContractCapture{{CaptureId: "saved", Type: typ}}
			_, err := Prepare(c, catalog, view, policy, nil)
			var admissionErr *ir.Error
			require.ErrorAs(t, err, &admissionErr)
			require.Equal(t, &ir.Error{Category: ir.Malformed, Path: "contract.rules[rule].captures[saved].type", Detail: "capture requires a scalar, enum or message type"}, admissionErr)
		})
	}
}

// TestPrepareLocatesDeadlineBounds pins the deadline admission: a liveness deadline sets one positive
// bound and names a violated state, and each rejection is located at the rule's deadline.
func TestPrepareLocatesDeadlineBounds(t *testing.T) {
	events := func(n int64) *testpilotspb.Deadline_RuleEvents {
		return &testpilotspb.Deadline_RuleEvents{RuleEvents: n}
	}
	elapsed := func(n int64) *testpilotspb.Deadline_ElapsedMilliseconds {
		return &testpilotspb.Deadline_ElapsedMilliseconds{ElapsedMilliseconds: n}
	}
	for _, tc := range []struct {
		name     string
		kind     testpilotspb.ContractRuleKind
		deadline *testpilotspb.Deadline
		// want is nil when the deadline is admitted.
		want *ir.Error
	}{
		{"elapsed", testpilotspb.CONTRACT_RULE_KIND_BOUNDED_LIVENESS, &testpilotspb.Deadline{ViolationStateId: "bad", Bound: elapsed(1000)}, nil},
		{"events", testpilotspb.CONTRACT_RULE_KIND_BOUNDED_LIVENESS, &testpilotspb.Deadline{ViolationStateId: "bad", Bound: events(3)}, nil},
		{"no bound", testpilotspb.CONTRACT_RULE_KIND_BOUNDED_LIVENESS, &testpilotspb.Deadline{ViolationStateId: "bad"},
			&ir.Error{Category: ir.Malformed, Path: "contract.rules[rule].deadline", Detail: "liveness deadline requires a bound"}},
		{"no deadline", testpilotspb.CONTRACT_RULE_KIND_BOUNDED_LIVENESS, nil,
			&ir.Error{Category: ir.Malformed, Path: "contract.rules[rule].deadline", Detail: "liveness deadline requires a bound"}},
		{"zero events", testpilotspb.CONTRACT_RULE_KIND_BOUNDED_LIVENESS, &testpilotspb.Deadline{ViolationStateId: "bad", Bound: events(0)},
			&ir.Error{Category: ir.Malformed, Path: "contract.rules[rule].deadline.rule_events", Detail: "liveness deadline bound must be positive"}},
		{"negative elapsed", testpilotspb.CONTRACT_RULE_KIND_BOUNDED_LIVENESS, &testpilotspb.Deadline{ViolationStateId: "bad", Bound: elapsed(-1)},
			&ir.Error{Category: ir.Malformed, Path: "contract.rules[rule].deadline.elapsed_milliseconds", Detail: "liveness deadline bound must be positive"}},
		{"no violation state", testpilotspb.CONTRACT_RULE_KIND_BOUNDED_LIVENESS, &testpilotspb.Deadline{Bound: events(3)},
			&ir.Error{Category: ir.Malformed, Path: "contract", Detail: "liveness requires a violated deadline target"}},
		{"safety", testpilotspb.CONTRACT_RULE_KIND_SAFETY, &testpilotspb.Deadline{ViolationStateId: "bad", Bound: events(3)},
			&ir.Error{Category: ir.Malformed, Path: "contract", Detail: "safety rule cannot declare a liveness deadline"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c, catalog, view, policy := fixture(t)
			c.Rules[0].Kind = tc.kind
			c.Rules[0].Deadline = tc.deadline
			_, err := Prepare(c, catalog, view, policy, nil)
			if tc.want == nil {
				require.NoError(t, err)
				return
			}
			var admissionErr *ir.Error
			require.ErrorAs(t, err, &admissionErr)
			require.Equal(t, tc.want, admissionErr)
		})
	}
}

func TestPrepareBoundsAndImmutableIndexes(t *testing.T) {
	c, catalog, view, policy := fixture(t)
	prepared, err := Prepare(c, catalog, view, policy, nil)
	require.NoError(t, err)
	snapshot := proto.CloneOf(c)
	c.Rules[0].Transitions[0].TransitionId = "mutated"
	policy.MaxStates = 1
	observations := prepared.program.Observations()
	observations[0].ID = "mutated"
	require.True(t, proto.Equal(snapshot, prepared.source))
	require.Equal(t, "id", prepared.program.Observations()[0].ID)
	for name, mutate := range map[string]func(*testpilotspb.Contract, *testpilotspb.ContractLimits){
		"state count": func(_ *testpilotspb.Contract, l *testpilotspb.ContractLimits) { l.MaxStates = 2 },
		"capture bytes": func(c *testpilotspb.Contract, l *testpilotspb.ContractLimits) {
			addCapture(c.Rules[0])
			l.MaxCaptureBytes = 1
		},
		"depth": func(c *testpilotspb.Contract, l *testpilotspb.ContractLimits) {
			l.MaxExpressionDepth = 1
			c.Rules[0].Transitions[0].Predicate = not(not(boolean(true)))
		},
		"event work": func(_ *testpilotspb.Contract, l *testpilotspb.ContractLimits) { l.MaxWorkPerEvent = 1 },
		"total work": func(_ *testpilotspb.Contract, l *testpilotspb.ContractLimits) { l.MaxTotalWork = 1 },
		"overflow":   func(_ *testpilotspb.Contract, l *testpilotspb.ContractLimits) { l.MaxTotalWork = 1<<63 - 1 },
	} {
		t.Run(name, func(t *testing.T) {
			c, catalog, view, policy := fixture(t)
			mutate(c, policy)
			_, err := Prepare(c, catalog, view, policy, nil)
			require.Error(t, err)
		})
	}
	var total int64 = 1<<63 - 2
	require.Error(t, add(&total, 2, 1<<63-1))
	require.EqualValues(t, 1<<63-2, total)
}
func TestOrderedPresenceAndContradictions(t *testing.T) {
	for _, test := range []struct {
		name       string
		predicates []*testpilotspb.Expression
		targets    []string
		wantError  bool
	}{
		{"preceding false presence", []*testpilotspb.Expression{not(present(observation("id"))), equal(observation("id"), observation("id"))}, []string{"bad", "good"}, false},
		{"contradictory observation presence", []*testpilotspb.Expression{all(present(observation("id")), not(present(observation("id"))))}, []string{"start"}, false},
		{"contradictory capture presence", []*testpilotspb.Expression{all(present(capture("saved")), not(present(capture("saved"))), present(observation("id")))}, []string{"start"}, false},
		{"unknown match retains following", []*testpilotspb.Expression{all(boolean(true), present(observation("text"))), present(observation("id"))}, []string{"good", "start"}, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			c, catalog, view, policy := fixture(t)
			r := c.Rules[0]
			addCapture(r)
			r.Transitions = nil
			for i, p := range test.predicates {
				tr := transition(string(rune('a'+i)), "start", test.targets[i], p)
				if tr.TargetStateId == "start" {
					assign(tr)
				}
				r.Transitions = append(r.Transitions, tr)
			}
			_, err := Prepare(c, catalog, view, policy, nil)
			if test.wantError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestCaptureAlternativesAndUnreachableAssignment(t *testing.T) {
	c, catalog, view, policy := fixture(t)
	r := c.Rules[0]
	addCapture(r)
	r.Transitions = []*testpilotspb.ContractTransition{transition("left", "start", "good", all(present(observation("id")), present(observation("text")))), transition("right", "start", "good", present(observation("id")))}
	assign(r.Transitions[0])
	assign(r.Transitions[1])
	_, err := Prepare(c, catalog, view, policy, nil)
	require.NoError(t, err)
	r.Transitions = append([]*testpilotspb.ContractTransition{transition("always", "start", "good", boolean(true))}, transition("unreachable", "start", "start", present(observation("id"))))
	assign(r.Transitions[1])
	_, err = Prepare(c, catalog, view, policy, nil)
	require.NoError(t, err)
}
func TestAdmissionExplorationCeiling(t *testing.T) {
	c, catalog, view, policy := fixture(t)
	r := c.Rules[0]
	for i := 0; i < 8; i++ {
		id := string(rune('a' + i))
		r.Captures = append(r.Captures, &testpilotspb.ContractCapture{CaptureId: id, Type: &testpilotspb.SingularType{Type: &testpilotspb.SingularType_Scalar{Scalar: &testpilotspb.ScalarType{Kind: testpilotspb.SCALAR_KIND_INT64}}}})
		tr := transition(id, "start", "start", all(not(present(capture(id))), present(observation("id"))))
		tr.CaptureAssignments = []*testpilotspb.ContractCaptureAssignment{{CaptureId: id, ObservationId: "id"}}
		r.Transitions = append(r.Transitions, tr)
	}
	r.Transitions = r.Transitions[1:]
	// Each event can skip earlier candidates; safe capture subsets multiply reachable configurations.
	for _, tr := range r.Transitions {
		tr.Predicate = all(tr.Predicate, equal(observation("id"), observation("id")))
	}
	_, err := Prepare(c, catalog, view, policy, nil)
	require.Error(t, err)
	var admissionErr *ir.Error
	require.ErrorAs(t, err, &admissionErr)
	require.Equal(t, ir.LimitExceeded, admissionErr.Category)
}

func TestCaptureCostsUseValueTypes(t *testing.T) {
	for _, small := range []bool{false, true} {
		t.Run(map[bool]string{false: "fixed-width-large-response", true: "insufficient-work"}[small], func(t *testing.T) {
			c, catalog, view, policy := fixture(t, 16<<20)
			r := c.Rules[0]
			addCapture(r)
			r.States = append(r.States, &testpilotspb.ContractState{StateId: "middle", Status: testpilotspb.CONTRACT_STATE_STATUS_PENDING})
			r.Transitions = []*testpilotspb.ContractTransition{transition("save", "start", "middle", present(observation("id"))), transition("compare", "middle", "good", all(present(observation("id")), equal(observation("id"), capture("saved"))))}
			assign(r.Transitions[0])
			policy.MaxWorkPerEvent = 128
			policy.MaxCaptureBytes = 40
			if small {
				policy.MaxWorkPerEvent = 16
			}
			_, err := Prepare(c, catalog, view, policy, nil)
			if small {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
	c, catalog, view, policy := fixture(t, 16<<20)
	c.Rules[0].Transitions[0].Predicate = all(present(observation("text")), equal(observation("text"), observation("text")))
	_, err := Prepare(c, catalog, view, policy, nil)
	require.Error(t, err)
}
func TestAuthoredDepthAndSeparateAdmissionWork(t *testing.T) {
	for _, depth := range []int64{2, 64} {
		t.Run(fmt.Sprint(depth), func(t *testing.T) {
			c, catalog, view, policy := fixture(t)
			policy.MaxExpressionDepth = depth
			r := c.Rules[0]
			addCapture(r)
			predicate := present(observation("id"))
			for i := int64(2); i < depth; i++ {
				predicate = not(predicate)
			}
			r.Transitions[0].Predicate = predicate
			assign(r.Transitions[0])
			_, err := Prepare(c, catalog, view, policy, nil)
			require.NoError(t, err)
			r.Transitions[0].Predicate = not(predicate)
			_, err = Prepare(c, catalog, view, policy, nil)
			require.Error(t, err)
		})
	}
	c, catalog, view, policy := fixture(t)
	policy.MaxWorkPerEvent = 4
	prepared, err := Prepare(c, catalog, view, policy, nil)
	require.NoError(t, err)
	require.EqualValues(t, 4, prepared.workPerEvent)
}

func TestPreparedProjectionUsesProgramFanout(t *testing.T) {
	c, catalog, view, policy := fixture(t)
	source := &testpilotspb.Expression{Expression: &testpilotspb.Expression_Literal{Literal: &testpilotspb.Value{Value: &testpilotspb.Value_MessageValue{MessageValue: &anypb.Any{TypeUrl: "type.googleapis.com/example.Empty"}}}}}
	c.Rules[0].Transitions[0].Predicate = present(&testpilotspb.Expression{Expression: &testpilotspb.Expression_Path{Path: &testpilotspb.PathExpression{Operand: source, Path: "items[*]"}}})
	prepared, err := Prepare(c, catalog, view, policy, nil)
	require.NoError(t, err)
	path := prepared.rules[0].transitions[0].Children()[0].Path()
	empty, err := catalog.BindType(messageType("example.Empty"))
	require.NoError(t, err)
	items := empty.Message().Fields().ByName("items")
	for _, count := range []int{127, 128, 129} {
		source := dynamicpb.NewMessage(empty.Message())
		list := source.Mutable(items).List()
		for range count {
			list.Append(protoreflect.ValueOfString("item"))
		}
		_, _, err := path.Read(context.Background(), source, ir.DefaultLimits())
		if count > 128 {
			require.ErrorContains(t, err, "fan-out ceiling exceeded")
		} else {
			require.NoError(t, err)
		}
	}
}

// TestPrepareRejectsAPlainContractBeyondItsOwnPerEventCeiling keeps the Driver's raised per-event
// ceiling visible: it admits a correlated capability's reservation, and a Contract under a Profile
// that declares a modest per-event value is still held to exactly that value.
func TestPrepareRejectsAPlainContractBeyondItsOwnPerEventCeiling(t *testing.T) {
	source, catalog, program, ceiling := fixture(t)
	_, err := Prepare(source, catalog, program, ceiling, nil)
	require.NoError(t, err)
	ceiling.MaxWorkPerEvent = 1
	_, err = Prepare(source, catalog, program, ceiling, nil)
	require.Error(t, err)
}

func instanceValue(id string) *testpilotspb.Expression {
	return &testpilotspb.Expression{Expression: &testpilotspb.Expression_Reference{Reference: &testpilotspb.Reference{Reference: &testpilotspb.Reference_InstanceValueId{InstanceValueId: id}}}}
}
func text(value string) *testpilotspb.Value {
	return &testpilotspb.Value{Value: &testpilotspb.Value_TextValue{TextValue: value}}
}
func singular(typ *testpilotspb.ValueType) *testpilotspb.SingularType { return typ.GetSingular() }
func assignment(id string, value *testpilotspb.Value) *testpilotspb.ContractInstanceAssignment {
	return &testpilotspb.ContractInstanceAssignment{InstanceValueId: id, Value: value}
}

// readsInstanceValue compares the text Observation against the instance value op.
func readsInstanceValue() *testpilotspb.Expression {
	return all(present(observation("text")), equal(observation("text"), instanceValue("op")))
}

// instanced declares the text instance value op on rule and one instance per text, assigning it.
func instanced(rule *testpilotspb.ContractRule, texts ...string) {
	rule.InstanceValues = []*testpilotspb.ContractInstanceValue{{InstanceValueId: "op", Type: singular(scalar(testpilotspb.SCALAR_KIND_TEXT))}}
	rule.Instances = nil
	for i, value := range texts {
		rule.Instances = append(rule.Instances, &testpilotspb.ContractRuleInstance{RuleId: fmt.Sprintf("%s-%d", rule.RuleId, i+1), Assignments: []*testpilotspb.ContractInstanceAssignment{assignment("op", text(value))}})
	}
}

// expand writes each Rule instance as a plain Rule under the instance's rule ID, with its instance
// values inlined as literals.
func expand(source *testpilotspb.Contract) *testpilotspb.Contract {
	expanded := proto.CloneOf(source)
	expanded.Rules = nil
	for _, rule := range source.Rules {
		if len(rule.Instances) == 0 {
			expanded.Rules = append(expanded.Rules, proto.CloneOf(rule))
			continue
		}
		expanded.Rules = slices.AppendSeq(expanded.Rules, ir.ExpandRule(rule))
	}
	return expanded
}

// A Rule with instances binds once and keeps its instances in declaration order with their values;
// a single instance and a declared value no predicate reads are admitted like their expansions.
func TestPrepareAdmitsRuleInstances(t *testing.T) {
	for name, tc := range map[string]struct {
		mutate func(*testpilotspb.ContractRule)
		ids    []string
	}{
		"two instances": {func(r *testpilotspb.ContractRule) {
			instanced(r, "a", "b")
			r.Transitions[0].Predicate = readsInstanceValue()
		}, []string{"rule-1", "rule-2"}},
		"one instance": {func(r *testpilotspb.ContractRule) {
			instanced(r, "a")
			r.Transitions[0].Predicate = readsInstanceValue()
		}, []string{"rule-1"}},
		"unread value": {func(r *testpilotspb.ContractRule) { instanced(r, "a", "b") }, []string{"rule-1", "rule-2"}},
	} {
		t.Run(name, func(t *testing.T) {
			c, catalog, view, policy := fixture(t)
			tc.mutate(c.Rules[0])
			prepared, err := Prepare(c, catalog, view, policy, nil)
			require.NoError(t, err)
			_, err = Prepare(expand(c), catalog, view, policy, nil)
			require.NoError(t, err)
			var ids []string
			for i, instance := range prepared.rules[0].instances {
				ids = append(ids, instance.ruleID)
				require.True(t, proto.Equal(c.Rules[0].Instances[i].Assignments[0].Value, instance.values["op"]))
			}
			require.Equal(t, tc.ids, ids)
		})
	}
}

// Every malformed instance declaration, instance or instance value read rejects at preparation,
// located by rule, instance and instance value ID.
func TestPrepareLocatesInstanceErrors(t *testing.T) {
	const rule = "contract.rules[rule]"
	enumeration := &testpilotspb.SingularType{Type: &testpilotspb.SingularType_Enumeration{Enumeration: &testpilotspb.NamedType{ProtobufType: string(testpilotspb.RunEventKind(0).Descriptor().FullName())}}}
	second := func(r *testpilotspb.ContractRule) {
		r.InstanceValues = append(r.InstanceValues, &testpilotspb.ContractInstanceValue{InstanceValueId: "n", Type: singular(scalar(testpilotspb.SCALAR_KIND_INT64))})
		for _, instance := range r.Instances {
			instance.Assignments = append(instance.Assignments, assignment("n", &testpilotspb.Value{Value: &testpilotspb.Value_SignedIntegerValue{SignedIntegerValue: "1"}}))
		}
	}
	mismatch := func(category ir.ErrorCategory, path, detail string) *ir.Error {
		return &ir.Error{Category: category, Path: path, Detail: detail}
	}
	typeRequired := "instance value requires a text, integer or enum type"
	duplicateRule := "invalid or duplicate rule identity"
	for name, tc := range map[string]struct {
		mutate func(*testpilotspb.Contract)
		want   *ir.Error
	}{
		"values without instances": {func(c *testpilotspb.Contract) { c.Rules[0].Instances = nil },
			mismatch(ir.Malformed, rule+".instances", "declared instance values require Rule instances")},
		"instances without values": {func(c *testpilotspb.Contract) { c.Rules[0].InstanceValues = nil },
			mismatch(ir.Malformed, rule+".instance_values", "Rule instances require declared instance values")},
		"empty value ID": {func(c *testpilotspb.Contract) { c.Rules[0].InstanceValues[0].InstanceValueId = "" },
			mismatch(ir.Malformed, rule+".instance_values[]", "invalid or duplicate instance value identity")},
		"invalid value ID": {func(c *testpilotspb.Contract) { c.Rules[0].InstanceValues[0].InstanceValueId = "o p" },
			mismatch(ir.Malformed, rule+".instance_values[o p]", "invalid or duplicate instance value identity")},
		"duplicate value ID": {func(c *testpilotspb.Contract) {
			c.Rules[0].InstanceValues = append(c.Rules[0].InstanceValues, proto.CloneOf(c.Rules[0].InstanceValues[0]))
		}, mismatch(ir.Malformed, rule+".instance_values[op]", "invalid or duplicate instance value identity")},
		"unset type": {func(c *testpilotspb.Contract) { c.Rules[0].InstanceValues[0].Type = nil },
			mismatch(ir.Malformed, rule+".instance_values[op].type", typeRequired)},
		"boolean type": {func(c *testpilotspb.Contract) {
			c.Rules[0].InstanceValues[0].Type = singular(scalar(testpilotspb.SCALAR_KIND_BOOLEAN))
		}, mismatch(ir.Malformed, rule+".instance_values[op].type", "an instance value cannot be boolean")},
		"floating type": {func(c *testpilotspb.Contract) {
			c.Rules[0].InstanceValues[0].Type = singular(scalar(testpilotspb.SCALAR_KIND_DOUBLE))
		}, mismatch(ir.Malformed, rule+".instance_values[op].type", typeRequired)},
		"message type": {func(c *testpilotspb.Contract) {
			c.Rules[0].InstanceValues[0].Type = singular(messageType("example.Empty"))
		},
			mismatch(ir.Malformed, rule+".instance_values[op].type", typeRequired)},
		"declared type not expected": {func(c *testpilotspb.Contract) {
			c.Rules[0].Transitions[0].Predicate = all(present(observation("id")), equal(observation("id"), instanceValue("op")))
		}, mismatch(ir.TypeMismatch, rule+".transitions[first].predicate.all[1].compare.right.reference.instance_value_id", "instance value is used where its declared type is not the expected type")},
		"omitted assignment": {func(c *testpilotspb.Contract) { c.Rules[0].Instances[1].Assignments = nil },
			mismatch(ir.Malformed, rule+".instances[rule-2].assignments[op]", "instance omits a declared instance value")},
		"repeated assignment": {func(c *testpilotspb.Contract) {
			c.Rules[0].Instances[0].Assignments = append(c.Rules[0].Instances[0].Assignments, assignment("op", text("again")))
		}, mismatch(ir.Malformed, rule+".instances[rule-1].assignments[op]", "instance value is assigned more than once")},
		"reordered assignments": {func(c *testpilotspb.Contract) {
			second(c.Rules[0])
			assignments := c.Rules[0].Instances[0].Assignments
			assignments[0], assignments[1] = assignments[1], assignments[0]
		}, mismatch(ir.Malformed, rule+".instances[rule-1].assignments[n]", "assignments follow the instance values' declaration order")},
		"undeclared assignment": {func(c *testpilotspb.Contract) { c.Rules[0].Instances[0].Assignments[0].InstanceValueId = "missing" },
			mismatch(ir.Unknown, rule+".instances[rule-1].assignments[missing]", "assignment names an undeclared instance value")},
		"unset assignment value": {func(c *testpilotspb.Contract) { c.Rules[0].Instances[0].Assignments[0].Value = nil },
			mismatch(ir.Malformed, rule+".instances[rule-1].assignments[op]", "assignment value is required")},
		"wrong assignment type": {func(c *testpilotspb.Contract) {
			c.Rules[0].Instances[0].Assignments[0].Value = &testpilotspb.Value{Value: &testpilotspb.Value_SignedIntegerValue{SignedIntegerValue: "1"}}
		}, mismatch(ir.TypeMismatch, rule+".instances[rule-1].assignments[op]", "literal does not match its declared type")},
		"undefined enum value": {func(c *testpilotspb.Contract) {
			c.Rules[0].Transitions[0].Predicate = boolean(true)
			c.Rules[0].InstanceValues[0].Type = enumeration
			c.Rules[0].Instances[0].Assignments[0].Value = ir.EnumValue(testpilotspb.RunEventKind(0).Descriptor(), protoreflect.EnumNumber(testpilotspb.RUN_EVENT_KIND_RUN_OPENED))
			c.Rules[0].Instances[1].Assignments[0].Value = &testpilotspb.Value{Value: &testpilotspb.Value_EnumValue{EnumValue: &testpilotspb.EnumValue{Name: "RUN_EVENT_KIND_NONE"}}}
		}, mismatch(ir.Unknown, rule+".instances[rule-2].assignments[op]", fmt.Sprintf("enum %s declares no value %q", testpilotspb.RunEventKind(0).Descriptor().FullName(), "RUN_EVENT_KIND_NONE"))},
		"empty instance ID": {func(c *testpilotspb.Contract) { c.Rules[0].Instances[0].RuleId = "" },
			mismatch(ir.Malformed, rule+".instances[]", duplicateRule)},
		"invalid instance ID": {func(c *testpilotspb.Contract) { c.Rules[0].Instances[0].RuleId = "rule 1" },
			mismatch(ir.Malformed, rule+".instances[rule 1]", duplicateRule)},
		"instance ID of its Rule": {func(c *testpilotspb.Contract) { c.Rules[0].Instances[1].RuleId = "rule" },
			mismatch(ir.Malformed, rule+".instances[rule]", duplicateRule)},
		"instance ID of another instance": {func(c *testpilotspb.Contract) { c.Rules[0].Instances[1].RuleId = "rule-1" },
			mismatch(ir.Malformed, rule+".instances[rule-1]", duplicateRule)},
		"instance ID of an earlier Rule": {func(c *testpilotspb.Contract) {
			earlier := proto.CloneOf(c.Rules[0])
			earlier.RuleId, earlier.InstanceValues, earlier.Instances = "earlier", nil, nil
			earlier.Transitions[0].Predicate = boolean(true)
			c.Rules = append([]*testpilotspb.ContractRule{earlier}, c.Rules...)
			c.Rules[1].Instances[0].RuleId = "earlier"
		}, mismatch(ir.Malformed, rule+".instances[earlier]", duplicateRule)},
		"Rule ID of an earlier instance": {func(c *testpilotspb.Contract) {
			later := proto.CloneOf(c.Rules[0])
			later.RuleId, later.InstanceValues, later.Instances = "rule-2", nil, nil
			later.Transitions[0].Predicate = boolean(true)
			c.Rules = append(c.Rules, later)
		}, mismatch(ir.Malformed, rule+".instances[rule-2]", duplicateRule)},
		"empty read": {func(c *testpilotspb.Contract) {
			c.Rules[0].Transitions[0].Predicate.GetAll().Operands[1].GetCompare().Right = instanceValue("")
		}, mismatch(ir.Malformed, rule+".transitions[first].predicate.all[1].compare.right.reference.instance_value_id", "instance value reference names no instance value")},
		"undeclared read": {func(c *testpilotspb.Contract) {
			c.Rules[0].Transitions[0].Predicate.GetAll().Operands[1].GetCompare().Right = instanceValue("missing")
		}, mismatch(ir.Unknown, rule+".transitions[first].predicate.all[1].compare.right.reference.instance_value_id", "instance value is not declared by the rule")},
		"read in a plain Rule": {func(c *testpilotspb.Contract) {
			c.Rules[0].InstanceValues, c.Rules[0].Instances = nil, nil
		}, mismatch(ir.Unknown, rule+".transitions[first].predicate.all[1].compare.right.reference.instance_value_id", "instance value is not declared by the rule")},
	} {
		t.Run(name, func(t *testing.T) {
			c, catalog, view, policy := fixture(t)
			instanced(c.Rules[0], "a", "b")
			c.Rules[0].Transitions[0].Predicate = readsInstanceValue()
			tc.mutate(c)
			_, err := Prepare(c, catalog, view, policy, nil)
			var diagnostic *ir.Error
			require.ErrorAs(t, err, &diagnostic)
			require.Equal(t, tc.want, diagnostic)
		})
	}
}

// Rule instance IDs share one namespace with correlated rule IDs.
func TestPrepareRejectsARuleInstanceNamedLikeACorrelatedRule(t *testing.T) {
	c, catalog, view, ceiling, correlated := correlatedFixture(t, 1)
	plain, _, _, _ := fixture(t)
	rule := plain.Rules[0]
	instanced(rule, "a")
	c.Rules = []*testpilotspb.ContractRule{rule}
	_, err := Prepare(c, catalog, view, ceiling, correlated)
	require.NoError(t, err)
	rule.Instances[0].RuleId = c.Correlated.Rules[0].RuleId
	_, err = Prepare(c, catalog, view, ceiling, correlated)
	var diagnostic *ir.Error
	require.ErrorAs(t, err, &diagnostic)
	require.Equal(t, &ir.Error{Category: ir.Malformed, Path: fmt.Sprintf("contract.rules[%s].instances[%s]", rule.RuleId, rule.Instances[0].RuleId), Detail: "invalid or duplicate rule identity"}, diagnostic)
}

// requireSameAdmission prepares source and its expansion and requires the same outcome: both
// admitted, or both rejected with the same diagnostic. It reports whether they were admitted.
func requireSameAdmission(t *testing.T, source *testpilotspb.Contract, catalog *ir.Catalog, view execution.ProgramView, ceiling *testpilotspb.ContractLimits) bool {
	t.Helper()
	_, instancedErr := Prepare(source, catalog, view, ceiling, nil)
	_, expandedErr := Prepare(expand(source), catalog, view, ceiling, nil)
	if expandedErr == nil {
		require.NoError(t, instancedErr)
		return true
	}
	var want, got *ir.Error
	require.ErrorAs(t, expandedErr, &want)
	require.ErrorAs(t, instancedErr, &got)
	require.Equal(t, want, got)
	return false
}

// Every Contract ceiling is charged per Rule instance, with each instance value read charged its
// inlined literal: for each ceiling, the instanced Contract and its expansion are rejected below
// the expansion's least admitted value, with the same diagnostic, and both admitted at it.
func TestPrepareChargesCeilingsPerRuleInstance(t *testing.T) {
	source, catalog, view, _ := fixture(t)
	generous := &testpilotspb.ContractLimits{MaxRules: 10000, MaxStates: 10000, MaxTransitions: 10000, MaxExpressionDepth: 16, MaxWorkPerEvent: 100000000, MaxTotalWork: 1000000000000, MaxCaptures: 10000, MaxCaptureBytes: 16 << 20}
	plain := source.Rules[0]
	plain.RuleId = "plain"
	rule := proto.CloneOf(plain)
	rule.RuleId = "rule"
	addCapture(rule)
	rule.States = append(rule.States, &testpilotspb.ContractState{StateId: "middle", Status: testpilotspb.CONTRACT_STATE_STATUS_PENDING})
	rule.Transitions = []*testpilotspb.ContractTransition{
		transition("save", "start", "middle", all(present(observation("id")), present(observation("text")), equal(observation("text"), instanceValue("op")))),
		transition("compare", "middle", "good", all(present(observation("id")), equal(observation("id"), capture("saved")))),
		transition("reject", "middle", "bad", all(present(observation("text")), not(equal(instanceValue("op"), observation("text"))))),
	}
	assign(rule.Transitions[0])
	instanced(rule, "a", "a much longer instance value than the first", "c")
	source.Rules = append(source.Rules, rule)
	fields := generous.ProtoReflect().Descriptor().Fields()
	for i := 0; i < fields.Len(); i++ {
		field := fields.Get(i)
		t.Run(string(field.Name()), func(t *testing.T) {
			with := func(value int64) *testpilotspb.ContractLimits {
				limits := proto.CloneOf(generous)
				limits.ProtoReflect().Set(field, protoreflect.ValueOfInt64(value))
				return limits
			}
			// Admission is monotone in each ceiling, so the expansion's least admitted value is found by bisection.
			low, high := int64(0), generous.ProtoReflect().Get(field).Int()
			_, err := Prepare(expand(source), catalog, view, with(high), nil)
			require.NoError(t, err)
			for high-low > 1 {
				middle := low + (high-low)/2
				if _, err := Prepare(expand(source), catalog, view, with(middle), nil); err == nil {
					high = middle
				} else {
					low = middle
				}
			}
			require.True(t, requireSameAdmission(t, source, catalog, view, with(high)))
			if low > 0 {
				require.False(t, requireSameAdmission(t, source, catalog, view, with(low)))
			}
		})
	}
	// Binding work has no Profile ceiling, so it is reached by growing the instance count of a Rule
	// whose transitions on one event kind rebind every earlier predicate.
	t.Run("binding work", func(t *testing.T) {
		source, catalog, view, _ := fixture(t)
		rule := source.Rules[0]
		rule.Transitions = nil
		for i := 0; i < 24; i++ {
			rule.Transitions = append(rule.Transitions, transition(fmt.Sprint("t", i), "start", "good", readsInstanceValue()))
		}
		admitted, rejected := false, false
		for count := 1; count <= 40; count++ {
			texts := make([]string, count)
			for i := range texts {
				texts[i] = fmt.Sprint("value", i)
			}
			instanced(rule, texts...)
			if requireSameAdmission(t, source, catalog, view, generous) {
				admitted = true
			} else {
				rejected = true
			}
		}
		require.True(t, admitted && rejected, "the instance count sweep crosses the binding work ceiling")
	})
}
