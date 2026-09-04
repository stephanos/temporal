package verification

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	umpirespb "go.temporal.io/server/api/umpire/v1"
	"go.temporal.io/server/tools/umpire/internal/execution"
	"go.temporal.io/server/tools/umpire/internal/ir"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/known/anypb"
)

func fixture(t *testing.T, responseBytes ...int64) (*umpirespb.Contract, *ir.Catalog, execution.ProgramView, *umpirespb.ContractLimits) {
	t.Helper()
	catalog, err := ir.NewCatalog(&descriptorpb.FileDescriptorSet{File: []*descriptorpb.FileDescriptorProto{{Name: proto.String("contract.proto"), Package: proto.String("example"), Syntax: proto.String("proto3"), MessageType: []*descriptorpb.DescriptorProto{{Name: proto.String("Empty"), Field: []*descriptorpb.FieldDescriptorProto{{Name: proto.String("items"), Number: proto.Int32(1), Label: descriptorpb.FieldDescriptorProto_LABEL_REPEATED.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum()}}}}}}})
	require.NoError(t, err)
	limits := &umpirespb.ProgramLimits{MaxEntrypoints: 8, MaxNodes: 32, MaxEdges: 64, MaxActivations: 64, MaxAttempts: 32, MaxRunEvents: 256, MaxExpressionDepth: 16, MaxPathFanout: 128, MaxRequestBytes: 4096, MaxResponseBytes: 4096, MaxTotalDurationMilliseconds: 30000, MaxCleanupDurationMilliseconds: 5000}
	if len(responseBytes) > 0 {
		limits.MaxResponseBytes = responseBytes[0]
	}
	source := &umpirespb.Case{Version: &umpirespb.FormatVersion{Major: 1}, CaseId: "case", Contract: &umpirespb.Contract{ContractId: "contract"}, Program: &umpirespb.Program{ProgramId: "program", Limits: limits, Observations: []*umpirespb.ObservationSchema{{ObservationId: "id", Type: scalar(umpirespb.SCALAR_KIND_INT64)}, {ObservationId: "text", Type: scalar(umpirespb.SCALAR_KIND_TEXT)}}, Entrypoints: []*umpirespb.Entrypoint{{EntrypointId: "controller", Context: umpirespb.ENTRYPOINT_CONTEXT_CONTROLLER, Activation: &umpirespb.ActivationBinding{Binding: &umpirespb.ActivationBinding_Controller{Controller: &umpirespb.ControllerActivation{}}}}}, Cleanup: &umpirespb.CleanupGraph{EntrypointId: "cleanup", Context: umpirespb.ENTRYPOINT_CONTEXT_CONTROLLER}}}
	prepared, err := execution.Prepare(source, catalog, execution.Policy{Identity: "host", CatalogIdentity: catalog.Identity(), Limits: proto.CloneOf(limits)})
	require.NoError(t, err)
	ceiling := &umpirespb.ContractLimits{MaxRules: 16, MaxStates: 32, MaxTransitions: 64, MaxExpressionDepth: 16, MaxWorkPerEvent: 100000, MaxTotalWork: 1000000000, MaxCaptures: 8, MaxCaptureBytes: 65536}
	contract := &umpirespb.Contract{ContractId: "contract", Limits: proto.CloneOf(ceiling), Rules: []*umpirespb.ContractRule{{RuleId: "rule", Kind: umpirespb.CONTRACT_RULE_KIND_SAFETY, InitialState: "start", States: []*umpirespb.ContractState{{StateId: "start", Terminal: umpirespb.CONTRACT_TERMINAL_STATE_NONTERMINAL}, {StateId: "good", Terminal: umpirespb.CONTRACT_TERMINAL_STATE_SATISFIED}, {StateId: "bad", Terminal: umpirespb.CONTRACT_TERMINAL_STATE_VIOLATED}}, Transitions: []*umpirespb.ContractTransition{transition("first", "start", "good", boolean(true))}}}}
	return contract, catalog, prepared.View(), ceiling
}
func scalar(kind umpirespb.ScalarKind) *umpirespb.ValueType {
	return &umpirespb.ValueType{Shape: &umpirespb.ValueType_Singular{Singular: &umpirespb.SingularType{Type: &umpirespb.SingularType_Scalar{Scalar: &umpirespb.ScalarType{Kind: kind}}}}}
}
func boolean(value bool) *umpirespb.ValueExpression {
	return &umpirespb.ValueExpression{Expression: &umpirespb.ValueExpression_Literal{Literal: &umpirespb.Value{Value: &umpirespb.Value_BoolValue{BoolValue: value}}}}
}
func observation(id string) *umpirespb.ValueExpression {
	return &umpirespb.ValueExpression{Expression: &umpirespb.ValueExpression_Observation{Observation: &umpirespb.ObservationReference{ObservationId: id}}}
}
func capture(id string) *umpirespb.ValueExpression {
	return &umpirespb.ValueExpression{Expression: &umpirespb.ValueExpression_Capture{Capture: &umpirespb.CaptureReference{CaptureId: id}}}
}
func present(value *umpirespb.ValueExpression) *umpirespb.ValueExpression {
	return &umpirespb.ValueExpression{Expression: &umpirespb.ValueExpression_Present{Present: &umpirespb.PresentExpression{Operand: value}}}
}
func all(values ...*umpirespb.ValueExpression) *umpirespb.ValueExpression {
	return &umpirespb.ValueExpression{Expression: &umpirespb.ValueExpression_All{All: &umpirespb.AllExpression{Operands: values}}}
}
func not(value *umpirespb.ValueExpression) *umpirespb.ValueExpression {
	return &umpirespb.ValueExpression{Expression: &umpirespb.ValueExpression_Negation{Negation: &umpirespb.NotExpression{Operand: value}}}
}
func equal(left, right *umpirespb.ValueExpression) *umpirespb.ValueExpression {
	return &umpirespb.ValueExpression{Expression: &umpirespb.ValueExpression_Equals{Equals: &umpirespb.EqualsExpression{Left: left, Right: right}}}
}
func transition(id, from, to string, predicate *umpirespb.ValueExpression) *umpirespb.ContractTransition {
	return &umpirespb.ContractTransition{TransitionId: id, SourceState: from, TargetState: to, Predicate: predicate, EventKinds: &umpirespb.RunEventKinds{Kinds: []umpirespb.RunEventKind{umpirespb.RUN_EVENT_KIND_INSTRUCTION_COMPLETED}}, Support: umpirespb.CONTRACT_SUPPORT_MATCHING_EVENT}
}
func addCapture(rule *umpirespb.ContractRule) {
	rule.Captures = []*umpirespb.ContractCaptureSchema{{CaptureId: "saved", Type: &umpirespb.ContractCaptureType{Type: &umpirespb.ContractCaptureType_Scalar{Scalar: &umpirespb.ScalarType{Kind: umpirespb.SCALAR_KIND_INT64}}}}}
}
func assign(tr *umpirespb.ContractTransition) {
	tr.CaptureAssignments = []*umpirespb.ContractCaptureAssignment{{CaptureId: "saved", Observation: &umpirespb.ObservationReference{ObservationId: "id"}}}
}

func TestPrepareMachinesAndOrder(t *testing.T) {
	for _, live := range []bool{false, true} {
		t.Run(map[bool]string{false: "safety", true: "liveness"}[live], func(t *testing.T) {
			c, catalog, view, policy := fixture(t)
			r := c.Rules[0]
			r.Transitions = append(r.Transitions, transition("second", "start", "bad", boolean(true)))
			if live {
				r.Kind = umpirespb.CONTRACT_RULE_KIND_BOUNDED_LIVENESS
				r.Horizon = &umpirespb.ContractHorizon{ElapsedMilliseconds: 1000, ViolationStateId: "bad"}
			}
			prepared, err := Prepare(c, catalog, view, policy)
			require.NoError(t, err)
			require.Equal(t, []int{0, 1}, prepared.rules[0].outgoing[0][umpirespb.RUN_EVENT_KIND_INSTRUCTION_COMPLETED])
		})
	}
}
func TestPrepareCapturePaths(t *testing.T) {
	for _, test := range []struct {
		name      string
		mutate    func(*umpirespb.ContractRule)
		wantError bool
	}{
		{"correlation", func(r *umpirespb.ContractRule) {}, false},
		{"missing observation guard", func(r *umpirespb.ContractRule) { r.Transitions[0].Predicate = boolean(true) }, true},
		{"pretransition read", func(r *umpirespb.ContractRule) {
			r.Transitions[0].Predicate = all(present(observation("id")), equal(capture("saved"), observation("id")))
		}, true},
		{"mismatched capture", func(r *umpirespb.ContractRule) { r.Captures[0].Type.GetScalar().Kind = umpirespb.SCALAR_KIND_TEXT }, true},
		{"support required", func(r *umpirespb.ContractRule) { r.Transitions[0].Support = umpirespb.CONTRACT_SUPPORT_NONE }, true},
		{"unsafe cycle", func(r *umpirespb.ContractRule) { r.Transitions[0].TargetState = "start" }, true},
		{"safe cycle", func(r *umpirespb.ContractRule) {
			r.Transitions[0].TargetState = "start"
			r.Transitions[0].Predicate = all(not(present(capture("saved"))), present(observation("id")))
		}, false},
		{"branch merge", func(r *umpirespb.ContractRule) {
			r.Transitions = append(r.Transitions, transition("branch", "start", "middle", boolean(true)))
		}, true},
		{"guarded merge", func(r *umpirespb.ContractRule) {
			r.Transitions = append(r.Transitions, transition("branch", "start", "middle", boolean(true)))
			r.Transitions[1].Predicate = all(present(capture("saved")), r.Transitions[1].Predicate)
		}, false},
		{"repeated assignment", func(r *umpirespb.ContractRule) { assign(r.Transitions[1]) }, true},
		{"duplicate atomic assignment", func(r *umpirespb.ContractRule) {
			r.Transitions[0].CaptureAssignments = append(r.Transitions[0].CaptureAssignments, proto.CloneOf(r.Transitions[0].CaptureAssignments[0]))
		}, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			c, catalog, view, policy := fixture(t)
			r := c.Rules[0]
			addCapture(r)
			r.States = append(r.States, &umpirespb.ContractState{StateId: "middle", Terminal: umpirespb.CONTRACT_TERMINAL_STATE_NONTERMINAL})
			r.Transitions = []*umpirespb.ContractTransition{transition("save", "start", "middle", present(observation("id"))), transition("compare", "middle", "good", all(present(observation("id")), equal(observation("id"), capture("saved"))))}
			assign(r.Transitions[0])
			test.mutate(r)
			_, err := Prepare(c, catalog, view, policy)
			if test.wantError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestPrepareRejectsMalformedContracts(t *testing.T) {
	for name, mutate := range map[string]func(*umpirespb.Contract){
		"duplicate rule": func(c *umpirespb.Contract) { c.Rules = append(c.Rules, proto.CloneOf(c.Rules[0])) },
		"missing states": func(c *umpirespb.Contract) { c.Rules[0].States = nil },
		"duplicate state": func(c *umpirespb.Contract) {
			c.Rules[0].States = append(c.Rules[0].States, proto.CloneOf(c.Rules[0].States[0]))
		},
		"duplicate transition": func(c *umpirespb.Contract) {
			c.Rules[0].Transitions = append(c.Rules[0].Transitions, proto.CloneOf(c.Rules[0].Transitions[0]))
		},
		"missing transitions":  func(c *umpirespb.Contract) { c.Rules[0].Transitions = nil },
		"missing initial":      func(c *umpirespb.Contract) { c.Rules[0].InitialState = "missing" },
		"terminal initial":     func(c *umpirespb.Contract) { c.Rules[0].InitialState = "bad" },
		"missing target":       func(c *umpirespb.Contract) { c.Rules[0].Transitions[0].TargetState = "missing" },
		"terminal source":      func(c *umpirespb.Contract) { c.Rules[0].Transitions[0].SourceState = "good" },
		"unspecified terminal": func(c *umpirespb.Contract) { c.Rules[0].States[0].Terminal = 0 },
		"unknown kind":         func(c *umpirespb.Contract) { c.Rules[0].Kind = 99 },
		"missing horizon":      func(c *umpirespb.Contract) { c.Rules[0].Kind = umpirespb.CONTRACT_RULE_KIND_BOUNDED_LIVENESS },
		"wrong expiry target": func(c *umpirespb.Contract) {
			c.Rules[0].Kind = umpirespb.CONTRACT_RULE_KIND_BOUNDED_LIVENESS
			c.Rules[0].Horizon = &umpirespb.ContractHorizon{ElapsedMilliseconds: 1, ViolationStateId: "good"}
		},
		"negative horizon": func(c *umpirespb.Contract) {
			c.Rules[0].Kind = umpirespb.CONTRACT_RULE_KIND_BOUNDED_LIVENESS
			c.Rules[0].Horizon = &umpirespb.ContractHorizon{ElapsedMilliseconds: -1, ViolationStateId: "bad"}
		},
		"safety horizon": func(c *umpirespb.Contract) {
			c.Rules[0].Horizon = &umpirespb.ContractHorizon{ElapsedMilliseconds: 1, ViolationStateId: "bad"}
		},
		"unknown observation": func(c *umpirespb.Contract) { c.Rules[0].Transitions[0].Predicate = present(observation("missing")) },
		"Slot forbidden": func(c *umpirespb.Contract) {
			c.Rules[0].Transitions[0].Predicate = present(&umpirespb.ValueExpression{Expression: &umpirespb.ValueExpression_Slot{Slot: &umpirespb.SlotReference{SlotId: "private"}}})
		},
		"outcome forbidden": func(c *umpirespb.Contract) {
			c.Rules[0].Transitions[0].Predicate = present(&umpirespb.ValueExpression{Expression: &umpirespb.ValueExpression_Outcome{Outcome: &umpirespb.InstructionOutcomeReference{Instruction: &umpirespb.InstructionReference{EntrypointId: "controller", InstructionId: "call"}, Field: umpirespb.INSTRUCTION_OUTCOME_FIELD_VALUE}}})
		},
		"nil predicate":        func(c *umpirespb.Contract) { c.Rules[0].Transitions[0].Predicate = nil },
		"nonboolean predicate": func(c *umpirespb.Contract) { c.Rules[0].Transitions[0].Predicate = observation("text") },
		"unknown expression": func(c *umpirespb.Contract) {
			c.Rules[0].Transitions[0].Predicate.ProtoReflect().SetUnknown([]byte{0x80, 0x06, 1})
		},
		"unknown event": func(c *umpirespb.Contract) { c.Rules[0].Transitions[0].EventKinds.Kinds = []umpirespb.RunEventKind{99} },
		"duplicate event": func(c *umpirespb.Contract) {
			tr := c.Rules[0].Transitions[0]
			tr.EventKinds.Kinds = append(tr.EventKinds.Kinds, tr.EventKinds.Kinds[0])
		},
		"nil state":   func(c *umpirespb.Contract) { c.Rules[0].States[0] = nil },
		"nil capture": func(c *umpirespb.Contract) { c.Rules[0].Captures = []*umpirespb.ContractCaptureSchema{nil} },
	} {
		t.Run(name, func(t *testing.T) {
			c, catalog, view, policy := fixture(t)
			mutate(c)
			_, err := Prepare(c, catalog, view, policy)
			require.Error(t, err)
		})
	}
}
func TestPrepareBoundsAndImmutableIndexes(t *testing.T) {
	c, catalog, view, policy := fixture(t)
	prepared, err := Prepare(c, catalog, view, policy)
	require.NoError(t, err)
	snapshot := prepared.Snapshot()
	c.Rules[0].Transitions[0].TransitionId = "mutated"
	policy.MaxStates = 1
	prepared.Snapshot().Rules[0].States[0].StateId = "mutated"
	observations := prepared.ProgramView().Observations()
	observations[0].ID = "mutated"
	require.Equal(t, snapshot, prepared.Snapshot())
	require.Equal(t, "id", prepared.ProgramView().Observations()[0].ID)
	for name, mutate := range map[string]func(*umpirespb.Contract){
		"state count":   func(c *umpirespb.Contract) { c.Limits.MaxStates = 2 },
		"capture bytes": func(c *umpirespb.Contract) { addCapture(c.Rules[0]); c.Limits.MaxCaptureBytes = 1 },
		"depth": func(c *umpirespb.Contract) {
			c.Limits.MaxExpressionDepth = 1
			c.Rules[0].Transitions[0].Predicate = not(not(boolean(true)))
		},
		"event work": func(c *umpirespb.Contract) { c.Limits.MaxWorkPerEvent = 1 },
		"total work": func(c *umpirespb.Contract) { c.Limits.MaxTotalWork = 1 },
		"overflow":   func(c *umpirespb.Contract) { c.Limits.MaxTotalWork = 1<<63 - 1 },
	} {
		t.Run(name, func(t *testing.T) {
			c, catalog, view, policy := fixture(t)
			mutate(c)
			_, err := Prepare(c, catalog, view, policy)
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
		predicates []*umpirespb.ValueExpression
		targets    []string
		wantError  bool
	}{
		{"preceding false presence", []*umpirespb.ValueExpression{not(present(observation("id"))), equal(observation("id"), observation("id"))}, []string{"bad", "good"}, false},
		{"contradictory observation presence", []*umpirespb.ValueExpression{all(present(observation("id")), not(present(observation("id"))))}, []string{"start"}, false},
		{"contradictory capture presence", []*umpirespb.ValueExpression{all(present(capture("saved")), not(present(capture("saved"))), present(observation("id")))}, []string{"start"}, false},
		{"unknown match retains following", []*umpirespb.ValueExpression{all(boolean(true), present(observation("text"))), present(observation("id"))}, []string{"good", "start"}, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			c, catalog, view, policy := fixture(t)
			r := c.Rules[0]
			addCapture(r)
			r.Transitions = nil
			for i, p := range test.predicates {
				tr := transition(string(rune('a'+i)), "start", test.targets[i], p)
				if tr.TargetState == "start" {
					assign(tr)
				}
				r.Transitions = append(r.Transitions, tr)
			}
			_, err := Prepare(c, catalog, view, policy)
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
	r.Transitions = []*umpirespb.ContractTransition{transition("left", "start", "good", all(present(observation("id")), present(observation("text")))), transition("right", "start", "good", present(observation("id")))}
	assign(r.Transitions[0])
	assign(r.Transitions[1])
	_, err := Prepare(c, catalog, view, policy)
	require.NoError(t, err)
	r.Transitions = append([]*umpirespb.ContractTransition{transition("always", "start", "good", boolean(true))}, transition("unreachable", "start", "start", present(observation("id"))))
	assign(r.Transitions[1])
	_, err = Prepare(c, catalog, view, policy)
	require.NoError(t, err)
}
func TestAdmissionExplorationCeiling(t *testing.T) {
	c, catalog, view, policy := fixture(t)
	r := c.Rules[0]
	for i := 0; i < 8; i++ {
		id := string(rune('a' + i))
		r.Captures = append(r.Captures, &umpirespb.ContractCaptureSchema{CaptureId: id, Type: &umpirespb.ContractCaptureType{Type: &umpirespb.ContractCaptureType_Scalar{Scalar: &umpirespb.ScalarType{Kind: umpirespb.SCALAR_KIND_INT64}}}})
		tr := transition(id, "start", "start", all(not(present(capture(id))), present(observation("id"))))
		tr.CaptureAssignments = []*umpirespb.ContractCaptureAssignment{{CaptureId: id, Observation: &umpirespb.ObservationReference{ObservationId: "id"}}}
		r.Transitions = append(r.Transitions, tr)
	}
	r.Transitions = r.Transitions[1:]
	// Each event can skip earlier candidates; safe capture subsets multiply reachable configurations.
	for _, tr := range r.Transitions {
		tr.Predicate = all(tr.Predicate, equal(observation("id"), observation("id")))
	}
	_, err := Prepare(c, catalog, view, policy)
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
			r.States = append(r.States, &umpirespb.ContractState{StateId: "middle", Terminal: umpirespb.CONTRACT_TERMINAL_STATE_NONTERMINAL})
			r.Transitions = []*umpirespb.ContractTransition{transition("save", "start", "middle", present(observation("id"))), transition("compare", "middle", "good", all(present(observation("id")), equal(observation("id"), capture("saved"))))}
			assign(r.Transitions[0])
			c.Limits.MaxWorkPerEvent = 128
			c.Limits.MaxCaptureBytes = 40
			if small {
				c.Limits.MaxWorkPerEvent = 16
			}
			_, err := Prepare(c, catalog, view, policy)
			if small {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
	c, catalog, view, policy := fixture(t, 16<<20)
	c.Rules[0].Transitions[0].Predicate = all(present(observation("text")), equal(observation("text"), observation("text")))
	_, err := Prepare(c, catalog, view, policy)
	require.Error(t, err)
}
func TestAuthoredDepthAndSeparateAdmissionWork(t *testing.T) {
	for _, depth := range []int64{2, 64} {
		t.Run(fmt.Sprint(depth), func(t *testing.T) {
			c, catalog, view, policy := fixture(t)
			c.Limits.MaxExpressionDepth = depth
			policy.MaxExpressionDepth = 64
			r := c.Rules[0]
			addCapture(r)
			predicate := present(observation("id"))
			for i := int64(2); i < depth; i++ {
				predicate = not(predicate)
			}
			r.Transitions[0].Predicate = predicate
			assign(r.Transitions[0])
			_, err := Prepare(c, catalog, view, policy)
			require.NoError(t, err)
			r.Transitions[0].Predicate = not(predicate)
			_, err = Prepare(c, catalog, view, policy)
			require.Error(t, err)
		})
	}
	c, catalog, view, policy := fixture(t)
	c.Limits.MaxWorkPerEvent = 4
	prepared, err := Prepare(c, catalog, view, policy)
	require.NoError(t, err)
	require.EqualValues(t, 4, prepared.workPerEvent)
}

func TestPreparedProjectionUsesProgramFanout(t *testing.T) {
	c, catalog, view, policy := fixture(t)
	source := &umpirespb.ValueExpression{Expression: &umpirespb.ValueExpression_Literal{Literal: &umpirespb.Value{Value: &umpirespb.Value_MessageValue{MessageValue: &anypb.Any{TypeUrl: "type.googleapis.com/example.Empty"}}}}}
	c.Rules[0].Transitions[0].Predicate = present(&umpirespb.ValueExpression{Expression: &umpirespb.ValueExpression_Path{Path: &umpirespb.PathExpression{Source: source, Path: &umpirespb.FieldPath{Segments: []*umpirespb.FieldPathSegment{{Field: "items", Selector: &umpirespb.FieldPathSegment_Repeated{Repeated: &umpirespb.RepeatedWildcard{}}}}}}}})
	prepared, err := Prepare(c, catalog, view, policy)
	require.NoError(t, err)
	path := prepared.rules[0].transitions[0].Children()[0].Path()
	for _, count := range []int64{127, 128, 129} {
		_, err := path.CheckFanout(1, count)
		if count > 128 {
			require.Error(t, err)
		} else {
			require.NoError(t, err)
		}
	}
}
