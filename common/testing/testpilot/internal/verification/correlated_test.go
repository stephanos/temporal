package verification

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/testpilot/internal/execution"
	"go.temporal.io/server/common/testing/testpilot/internal/ir"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/dynamicpb"
	"google.golang.org/protobuf/types/known/anypb"
)

// The Case carries the only format version, so a correlated contract that still writes its own fails
// strict decoding naming the field.
func TestCorrelatedContractCarriesNoVersion(t *testing.T) {
	var contract testpilotspb.Contract
	require.ErrorContains(t, protojson.Unmarshal([]byte(`{"contractId":"correlated","correlated":{"version":1}}`), &contract), `unknown field "version"`)
}

func correlatedFixture(t *testing.T, bound int64) (*testpilotspb.Contract, *ir.Catalog, execution.ProgramView, *testpilotspb.ContractLimits, *testpilotspb.CorrelatedLimits) {
	t.Helper()
	files := []*descriptorpb.FileDescriptorProto{}
	seen := map[string]bool{}
	var collect func(protoreflect.FileDescriptor)
	collect = func(f protoreflect.FileDescriptor) {
		if seen[f.Path()] {
			return
		}
		seen[f.Path()] = true
		for i := 0; i < f.Imports().Len(); i++ {
			collect(f.Imports().Get(i).FileDescriptor)
		}
		files = append(files, protodesc.ToFileDescriptorProto(f))
	}
	collect(testpilotspb.File_temporal_server_api_testpilot_v1_run_proto)
	catalog, err := ir.NewCatalog(&descriptorpb.FileDescriptorSet{File: files})
	require.NoError(t, err)
	_, _, view, ceiling := fixture(t)
	limits := view.Limits()
	limits.MaxRunEvents = 32
	typ := messageType("temporal.server.api.testpilot.v1.CorrelatedEvidence")
	program := &testpilotspb.Program{ProgramId: "correlated.program", Observations: []*testpilotspb.Observation{{ObservationId: "evidence", Type: typ}}, Entrypoints: []*testpilotspb.Entrypoint{{EntrypointId: "controller", Activation: &testpilotspb.Entrypoint_Controller{Controller: &testpilotspb.ControllerActivation{}}}}, Cleanup: &testpilotspb.Cleanup{EntrypointId: "cleanup"}}
	prepared, err := execution.Prepare(&testpilotspb.Case{Version: &testpilotspb.FormatVersion{Major: 1}, CaseId: "correlated.case", Program: program, Contract: &testpilotspb.Contract{ContractId: "correlated"}}, catalog, execution.Profile{Identity: "host", CatalogIdentity: catalog.Identity(), Limits: proto.CloneOf(limits)})
	require.NoError(t, err)
	ceiling.MaxCaptures = 32
	ceiling.MaxCaptureBytes = 65536
	state := &testpilotspb.ModelValue{DefinitionId: "state", Value: "ready"}
	value := func(id, v string) *testpilotspb.ModelValue {
		return &testpilotspb.ModelValue{DefinitionId: id, Value: v}
	}
	trigger := stepPresent(testpilotspb.CORRELATED_STEP_FIELD_ACTION, "request")
	response := stepEquals(testpilotspb.CORRELATED_STEP_FIELD_OUTCOME, "outcome", "response")
	s := &testpilotspb.CorrelatedContract{ProjectionId: "projection", ProjectionFingerprint: "projection-v1", EvidenceObservationId: "evidence", ScopeFields: []string{"run"}, OperationField: "operation", Sources: []string{"source"}, InitialState: state, Rules: []*testpilotspb.CorrelatedRule{{RuleId: "response", Clock: testpilotspb.CORRELATED_CLOCK_OPERATION_TRANSITIONS, Bound: bound, Ending: testpilotspb.TRACE_ENDING_PARTIAL, Trigger: trigger, Response: response}}}
	for _, kind := range []string{"request", "both", "tick", "reply"} {
		action := kind
		if kind == "both" {
			action = "request"
		}
		outcome := "quiet"
		if kind == "both" || kind == "reply" {
			outcome = "response"
		}
		tr := &testpilotspb.CorrelatedTransition{PriorState: state, Action: value(action, kind), State: state, Outcome: value("outcome", outcome)}
		s.Transitions = append(s.Transitions, tr)
		out := proto.CloneOf(tr)
		out.PriorState = nil
		s.ProjectionRules = append(s.ProjectionRules, &testpilotspb.CorrelatedProjectionRule{Kind: kind, Meaning: testpilotspb.CORRELATED_EVIDENCE_MEANING_CONFIRMED, Outputs: []*testpilotspb.CorrelatedTransition{out}})
	}
	s.ProjectionRules = append(s.ProjectionRules, &testpilotspb.CorrelatedProjectionRule{Kind: "poll", Meaning: testpilotspb.CORRELATED_EVIDENCE_MEANING_IRRELEVANT})
	correlated := &testpilotspb.CorrelatedLimits{MaxEvents: 16, MaxBuffered: 8, MaxKeys: 8, MaxSupport: 256, MaxProjectionWork: 1000000000, MaxEventBytes: 512, MaxSemanticTransitions: 32, MaxObligations: 16, MaxObligationWork: 1000000000}
	return &testpilotspb.Contract{ContractId: "correlated", Correlated: s}, catalog, prepared.View(), ceiling, correlated
}

// stepPresent is the step condition matching a step that carries any value of definitionID at field.
func stepPresent(field testpilotspb.CorrelatedStepField, definitionID string) *testpilotspb.Expression {
	return &testpilotspb.Expression{Expression: &testpilotspb.Expression_Present{Present: &testpilotspb.PresentExpression{Operand: stepReference(field, definitionID)}}}
}

// stepEquals is the step condition matching a step that carries exactly text of definitionID at field.
func stepEquals(field testpilotspb.CorrelatedStepField, definitionID, text string) *testpilotspb.Expression {
	return correlationComparison(testpilotspb.COMPARISON_OPERATOR_EQUAL, stepReference(field, definitionID), correlatedLiteralOperand(text))
}

func stepReference(field testpilotspb.CorrelatedStepField, definitionID string) *testpilotspb.Expression {
	return correlatedReference(&testpilotspb.Reference{Reference: &testpilotspb.Reference_CorrelatedStep{CorrelatedStep: &testpilotspb.CorrelatedStepReference{Field: field, DefinitionId: definitionID}}})
}

func correlatedEvidence(ordinal int64, kind, operation string, parents ...int64) *testpilotspb.CorrelatedEvidence {
	id := func(n int64) *testpilotspb.CorrelatedIdentity {
		return &testpilotspb.CorrelatedIdentity{Scope: []*testpilotspb.NamedValue{{FieldId: "run", Value: &testpilotspb.Value{Value: &testpilotspb.Value_TextValue{TextValue: "one"}}}}, EvidenceSource: "source", Ordinal: n}
	}
	e := &testpilotspb.CorrelatedEvidence{Identity: id(ordinal), Operation: operation, Kind: kind}
	for _, p := range parents {
		e.Parents = append(e.Parents, id(p))
	}
	return e
}
func correlatedEvent(t *testing.T, seq int64, e *testpilotspb.CorrelatedEvidence) *testpilotspb.RunEvent {
	t.Helper()
	a, err := anypb.New(e)
	require.NoError(t, err)
	return &testpilotspb.RunEvent{Sequence: seq, Kind: testpilotspb.RUN_EVENT_KIND_INSTRUCTION_COMPLETED, Observations: []*testpilotspb.ObservationResult{{ObservationId: "evidence", Value: &testpilotspb.Value{Value: &testpilotspb.Value_MessageValue{MessageValue: a}}}}}
}
func TestCorrelatedDeadlinesAndCausalAdmission(t *testing.T) {
	for _, tc := range []struct {
		name  string
		bound int64
		kinds []string
		ops   []string
		want  testpilotspb.RuleVerdictStatus
	}{
		{"zero-response", 0, []string{"both"}, nil, testpilotspb.RULE_VERDICT_STATUS_SATISFIED},
		{"zero-missing", 0, []string{"request"}, nil, testpilotspb.RULE_VERDICT_STATUS_VIOLATED},
		{"deadline", 1, []string{"request", "reply"}, nil, testpilotspb.RULE_VERDICT_STATUS_SATISFIED},
		{"late", 1, []string{"request", "tick", "reply"}, nil, testpilotspb.RULE_VERDICT_STATUS_VIOLATED},
		{"prefix", 2, []string{"request", "tick"}, nil, testpilotspb.RULE_VERDICT_STATUS_INCONCLUSIVE},
		{"rearmed", 1, []string{"both", "request"}, nil, testpilotspb.RULE_VERDICT_STATUS_INCONCLUSIVE},
		{"repeated", 2, []string{"request", "request", "reply"}, nil, testpilotspb.RULE_VERDICT_STATUS_SATISFIED},
		{"interleaved", 1, []string{"request", "tick", "reply"}, []string{"a", "b", "a"}, testpilotspb.RULE_VERDICT_STATUS_SATISFIED},
		{"poll-stutter", 1, []string{"request", "poll", "reply"}, nil, testpilotspb.RULE_VERDICT_STATUS_SATISFIED},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c, catalog, view, ceiling, correlated := correlatedFixture(t, tc.bound)
			p, err := Prepare(c, catalog, view, ceiling, correlated)
			require.NoError(t, err)
			e, err := p.newEvaluator(context.Background(), view)
			require.NoError(t, err)
			_, err = e.Observe(context.Background(), &testpilotspb.RunEvent{Sequence: 1, Kind: testpilotspb.RUN_EVENT_KIND_RUN_OPENED})
			require.NoError(t, err)
			for i, kind := range tc.kinds {
				op := "a"
				if tc.ops != nil {
					op = tc.ops[i]
				}
				event := correlatedEvent(t, int64(i+2), correlatedEvidence(int64(i), kind, op))
				_, err = e.Observe(context.Background(), event)
				require.NoError(t, err)
			}
			require.Equal(t, tc.want, e.result.Rules[0].Status)
		})
	}
}

func TestCorrelatedPrepareRejectsUnsupportedCapability(t *testing.T) {
	for name, mutate := range map[string]func(*testpilotspb.Contract){
		"unknown-field": func(c *testpilotspb.Contract) { c.Correlated.ProtoReflect().SetUnknown([]byte{0xf8, 0x07, 1}) },
		"nested-unknown": func(c *testpilotspb.Contract) {
			c.Correlated.InitialState.ProtoReflect().SetUnknown([]byte{0xf8, 0x07, 1})
		},
		"clock":          func(c *testpilotspb.Contract) { c.Correlated.Rules[0].Clock = 99 },
		"endpoint":       func(c *testpilotspb.Contract) { c.Correlated.Rules[0].Ending = 99 },
		"negative-bound": func(c *testpilotspb.Contract) { c.Correlated.Rules[0].Bound = -1 },
		"unsupported-formula": func(c *testpilotspb.Contract) {
			c.Correlated.Rules[0].Trigger = correlatedAll(c.Correlated.Rules[0].Trigger)
		},
		"absent-constraint": func(c *testpilotspb.Contract) { c.Correlated.Rules[0].Response.Expression = nil },
		"nontext-equality": func(c *testpilotspb.Contract) {
			c.Correlated.Rules[0].Response.GetCompare().Right = correlatedLiteral(&testpilotspb.Value{Value: &testpilotspb.Value_BoolValue{BoolValue: true}})
		},
		"not-equal-condition": func(c *testpilotspb.Contract) {
			c.Correlated.Rules[0].Response.GetCompare().Operator = testpilotspb.COMPARISON_OPERATOR_NOT_EQUAL
		},
		"observation-type": func(c *testpilotspb.Contract) { c.Correlated.EvidenceObservationId = "missing" },
		"prior-state-output": func(c *testpilotspb.Contract) {
			c.Correlated.ProjectionRules[0].Outputs[0].PriorState = c.Correlated.InitialState
		},
		"unauthorized-result": func(c *testpilotspb.Contract) { c.Correlated.ProjectionRules[0].Outputs[0].Outcome.Value = "unmodeled" },
		"missing-submission": func(c *testpilotspb.Contract) {
			c.Correlated.ProjectionRules[0].Submission = c.Correlated.Transitions[0].Action
		},
		"duplicate-clause": func(c *testpilotspb.Contract) {
			c.Correlated.Rules = append(c.Correlated.Rules, proto.CloneOf(c.Correlated.Rules[0]))
		},
	} {
		t.Run(name, func(t *testing.T) {
			c, catalog, view, ceiling, correlated := correlatedFixture(t, 1)
			mutate(c)
			_, err := Prepare(c, catalog, view, ceiling, correlated)
			require.Error(t, err)
		})
	}
	for name, constrain := range map[string]func(*testpilotspb.ContractLimits, *testpilotspb.CorrelatedLimits){
		"zero-limit":     func(_ *testpilotspb.ContractLimits, l *testpilotspb.CorrelatedLimits) { l.MaxSupport = 0 },
		"negative-limit": func(_ *testpilotspb.ContractLimits, l *testpilotspb.CorrelatedLimits) { l.MaxProjectionWork = -1 },
		"overflow-product": func(_ *testpilotspb.ContractLimits, l *testpilotspb.CorrelatedLimits) {
			l.MaxSupport = 9223372036854775807
		},
		"program-event-limit": func(_ *testpilotspb.ContractLimits, l *testpilotspb.CorrelatedLimits) { l.MaxEvents = 10000 },
		"capture-count":       func(l *testpilotspb.ContractLimits, _ *testpilotspb.CorrelatedLimits) { l.MaxCaptures = 1 },
		"capture-bytes":       func(l *testpilotspb.ContractLimits, _ *testpilotspb.CorrelatedLimits) { l.MaxCaptureBytes = 1 },
		"missing": func(_ *testpilotspb.ContractLimits, l *testpilotspb.CorrelatedLimits) {
			*l = testpilotspb.CorrelatedLimits{}
		},
	} {
		t.Run(name, func(t *testing.T) {
			c, catalog, view, ceiling, correlated := correlatedFixture(t, 1)
			constrain(ceiling, correlated)
			_, err := Prepare(c, catalog, view, ceiling, correlated)
			require.Error(t, err)
		})
	}
	c, catalog, view, ceiling, _ := correlatedFixture(t, 1)
	_, err := Prepare(c, catalog, view, ceiling, nil)
	require.ErrorContains(t, err, "Profile correlated limits required")
}

// A correlated condition shares the one expression language, so every reference outside the
// correlated context rejects at preparation at its located path, as does a trigger or response
// reading a part of the step its rule does not test.
func TestCorrelatedPrepareLocatesConditionsOutsideTheCorrelatedContext(t *testing.T) {
	const rule = "contract.correlated.rules[response]"
	for name, value := range map[string]*testpilotspb.Reference{
		"slot_id": {Reference: &testpilotspb.Reference_SlotId{SlotId: "slot"}},
		"outcome": {Reference: &testpilotspb.Reference_Outcome{Outcome: &testpilotspb.InstructionOutcomeReference{
			Instruction: &testpilotspb.InstructionReference{EntrypointId: "controller", InstructionId: "call"}, Field: testpilotspb.INSTRUCTION_OUTCOME_FIELD_STATUS,
		}}},
		"run":                    {Reference: &testpilotspb.Reference_Run{Run: &testpilotspb.RunReference{}}},
		"environment_binding_id": {Reference: &testpilotspb.Reference_EnvironmentBindingId{EnvironmentBindingId: "namespace"}},
		"observation_id":         {Reference: &testpilotspb.Reference_ObservationId{ObservationId: "evidence"}},
		"run_event":              {Reference: &testpilotspb.Reference_RunEvent{RunEvent: &testpilotspb.RunEventReference{Selection: &testpilotspb.RunEventReference_Field{Field: testpilotspb.RUN_EVENT_FIELD_KIND}}}},
		"capture_id":             {Reference: &testpilotspb.Reference_CaptureId{CaptureId: "capture"}},
		"model_value":            {Reference: &testpilotspb.Reference_ModelValue{ModelValue: &testpilotspb.ModelValue{DefinitionId: "request", Value: "request"}}},
		"projected_value":        {Reference: &testpilotspb.Reference_ProjectedValue{ProjectedValue: &testpilotspb.ProjectedValueReference{}}},
	} {
		reference := correlatedReference(value)
		for site, mutate := range map[string]func(*testpilotspb.CorrelatedRule){
			rule + ".trigger.present": func(r *testpilotspb.CorrelatedRule) {
				r.Trigger = &testpilotspb.Expression{Expression: &testpilotspb.Expression_Present{Present: &testpilotspb.PresentExpression{Operand: reference}}}
			},
			rule + ".response.compare.left": func(r *testpilotspb.CorrelatedRule) { r.Response.GetCompare().Left = reference },
			rule + ".correlation.any[1].compare.right": func(r *testpilotspb.CorrelatedRule) {
				r.Correlation = correlatedAny(correlatedTriggered(), correlationComparison(testpilotspb.COMPARISON_OPERATOR_EQUAL, correlatedLiteralOperand("1"), reference))
			},
		} {
			t.Run(site+"/"+name, func(t *testing.T) {
				c, catalog, view, ceiling, correlated := correlatedFixture(t, 1)
				correlated.MaxCorrelationDepth = 4
				mutate(c.Correlated.Rules[0])
				_, err := Prepare(c, catalog, view, ceiling, correlated)
				var diagnostic *ir.Error
				require.ErrorAs(t, err, &diagnostic)
				require.Equal(t, &ir.Error{Category: ir.Unknown, Path: site + ".reference." + name, Detail: "reference is not admitted in this expression context"}, diagnostic)
			})
		}
	}
	for name, tc := range map[string]struct {
		mutate func(*testpilotspb.CorrelatedRule)
		want   *ir.Error
	}{
		"trigger reads the outcome": {
			mutate: func(r *testpilotspb.CorrelatedRule) {
				r.Trigger = stepPresent(testpilotspb.CORRELATED_STEP_FIELD_OUTCOME, "outcome")
			},
			want: &ir.Error{Category: ir.Unknown, Path: rule + ".trigger.present.reference.correlated_step.field", Detail: "a trigger reads only the step's action"},
		},
		"response reads the action": {
			mutate: func(r *testpilotspb.CorrelatedRule) {
				r.Response = stepEquals(testpilotspb.CORRELATED_STEP_FIELD_ACTION, "request", "request")
			},
			want: &ir.Error{Category: ir.Unknown, Path: rule + ".response.compare.left.reference.correlated_step.field", Detail: "a response reads only the step's outcome, state or facts"},
		},
	} {
		t.Run(name, func(t *testing.T) {
			c, catalog, view, ceiling, correlated := correlatedFixture(t, 1)
			tc.mutate(c.Correlated.Rules[0])
			_, err := Prepare(c, catalog, view, ceiling, correlated)
			var diagnostic *ir.Error
			require.ErrorAs(t, err, &diagnostic)
			require.Equal(t, tc.want, diagnostic)
		})
	}
}

func TestCorrelatedWireAdmissionIsAtomic(t *testing.T) {
	for name, mutate := range map[string]func(*testpilotspb.CorrelatedEvidence){
		"scope":   func(e *testpilotspb.CorrelatedEvidence) { e.Identity.Scope[0].Value = correlatedText("another") },
		"source":  func(e *testpilotspb.CorrelatedEvidence) { e.Identity.EvidenceSource = "unknown" },
		"ordinal": func(e *testpilotspb.CorrelatedEvidence) { e.Identity.Ordinal = -1 },
		"kind":    func(e *testpilotspb.CorrelatedEvidence) { e.Kind = "unknown" },
		"field": func(e *testpilotspb.CorrelatedEvidence) {
			e.Fields = []*testpilotspb.NamedValue{{FieldId: "unauthorized"}}
		},
		"cycle": func(e *testpilotspb.CorrelatedEvidence) {
			e.Parents = []*testpilotspb.CorrelatedIdentity{proto.CloneOf(e.Identity)}
		},
		"unknown-field": func(e *testpilotspb.CorrelatedEvidence) { e.ProtoReflect().SetUnknown([]byte{0xf8, 0x07, 1}) },
	} {
		t.Run(name, func(t *testing.T) {
			c, catalog, view, ceiling, correlated := correlatedFixture(t, 1)
			p, err := Prepare(c, catalog, view, ceiling, correlated)
			require.NoError(t, err)
			e, err := p.newEvaluator(context.Background(), view)
			require.NoError(t, err)
			_, err = e.Observe(context.Background(), event(1, 0, testpilotspb.RUN_EVENT_KIND_RUN_OPENED))
			require.NoError(t, err)
			_, err = e.Observe(context.Background(), correlatedEvent(t, 2, correlatedEvidence(0, "request", "a")))
			require.NoError(t, err)
			before := proto.CloneOf(e.result)
			bad := correlatedEvidence(1, "reply", "a")
			mutate(bad)
			_, err = e.Observe(context.Background(), correlatedEvent(t, 3, bad))
			require.Error(t, err)
			require.Equal(t, testpilotspb.RULE_VERDICT_STATUS_INCONCLUSIVE, e.result.Rules[0].Status)
			require.True(t, proto.Equal(before.Rules[0], e.result.Rules[0]))
			require.EqualValues(t, 1, e.correlated.transitions)
		})
	}
}

// A scope value is text: the first evidence of a Run fixes the scope, so one spelling it as any other
// value rejects before anything is compared with it.
func TestCorrelatedEvidenceScopeValuesAreText(t *testing.T) {
	c, catalog, view, ceiling, correlated := correlatedFixture(t, 1)
	p, err := Prepare(c, catalog, view, ceiling, correlated)
	require.NoError(t, err)
	e, err := p.newEvaluator(context.Background(), view)
	require.NoError(t, err)
	_, err = e.Observe(context.Background(), event(1, 0, testpilotspb.RUN_EVENT_KIND_RUN_OPENED))
	require.NoError(t, err)
	evidence := correlatedEvidence(0, "request", "a")
	evidence.Identity.Scope[0].Value = &testpilotspb.Value{Value: &testpilotspb.Value_UnsignedIntegerValue{UnsignedIntegerValue: "1"}}
	_, err = e.Observe(context.Background(), correlatedEvent(t, 2, evidence))
	require.ErrorContains(t, err, "wrong correlated bindings")
}

func TestCorrelatedCausalChunksDuplicatesAndIsolation(t *testing.T) {
	c, catalog, view, ceiling, correlated := correlatedFixture(t, 1)
	p, err := Prepare(c, catalog, view, ceiling, correlated)
	require.NoError(t, err)
	for split := 0; split <= 4; split++ {
		e, err := p.newEvaluator(context.Background(), view)
		require.NoError(t, err)
		_, err = e.Observe(context.Background(), event(1, 0, testpilotspb.RUN_EVENT_KIND_RUN_OPENED))
		require.NoError(t, err)
		observations := []*testpilotspb.RunEvent{
			correlatedEvent(t, 2, correlatedEvidence(1, "reply", "a", 0)),
			correlatedEvent(t, 3, correlatedEvidence(1, "reply", "a", 0)),
			correlatedEvent(t, 4, correlatedEvidence(0, "request", "a")),
			correlatedEvent(t, 5, correlatedEvidence(0, "request", "a")),
		}
		for _, chunk := range [][]*testpilotspb.RunEvent{observations[:split], observations[split:]} {
			for _, observation := range chunk {
				_, err = e.Observe(context.Background(), observation)
				require.NoError(t, err)
			}
		}
		require.EqualValues(t, 2, e.correlated.transitions)
		require.Equal(t, testpilotspb.RULE_VERDICT_STATUS_SATISFIED, e.result.Rules[0].Status)
		require.Equal(t, []int64{2, 4}, e.result.Rules[0].SupportingEventSequences)
	}
}

func TestCorrelatedCheckedLeanFixtures(t *testing.T) {
	var violationsWithEvidence atomic.Int64
	encoded, err := os.ReadFile("../../testdata/case-runtime-conformance/correlated.json")
	require.NoError(t, err)
	var fixtures []struct {
		Name       string            `json:"name"`
		Case       json.RawMessage   `json:"case"`
		Events     []json.RawMessage `json:"events"`
		Expected   int               `json:"expected"`
		Incomplete bool              `json:"incomplete"`
	}
	require.NoError(t, json.Unmarshal(encoded, &fixtures))
	require.Len(t, fixtures, 19)
	for _, fixture := range fixtures {
		t.Run(fixture.Name, func(t *testing.T) {
			var artifact testpilotspb.Case
			require.NoError(t, protojson.Unmarshal(fixture.Case, &artifact))
			rules := artifact.GetProvenance().GetCorrelatedRules()
			require.Len(t, rules, 1)
			// The Contract names the rule and projection by Case-local names, which provenance maps
			// back to the Definition IDs the correlated rule binding records.
			definitions := map[string]string{}
			for _, row := range artifact.GetProvenance().GetLocalNames() {
				definitions[row.GetLocalName()] = row.GetDefinitionId()
			}
			require.Equal(t, rules[0].GetRuleId(), definitions[artifact.Contract.Correlated.Rules[0].RuleId])
			require.Equal(t, "test.property", rules[0].GetPropertyId())
			require.Equal(t, rules[0].GetProjectionId(), definitions[artifact.Contract.Correlated.ProjectionId])
			require.Equal(t, artifact.Contract.Correlated.ProjectionFingerprint, rules[0].GetProjectionFingerprint())
			_, catalog, fixtureView, ceiling, correlated := correlatedFixture(t, 1)
			// The structured fixtures project six events over a nine-row table, so one event's
			// projection work is larger than the atomic fixtures' ceiling allows.
			ceiling.MaxWorkPerEvent = 1000000
			program, err := execution.Prepare(&artifact, catalog, execution.Profile{Identity: "host", CatalogIdentity: catalog.Identity(), Limits: fixtureView.Limits()})
			require.NoError(t, err)
			view := program.View()
			prepared, err := Prepare(artifact.Contract, catalog, view, ceiling, correlated)
			require.NoError(t, err)
			for split := 0; split <= len(fixture.Events); split++ {
				monitor, err := prepared.New(context.Background(), view)
				require.NoError(t, err)
				run := &testpilotspb.Run{RunId: "one", CaseId: artifact.CaseId, ProgramId: artifact.Program.ProgramId, Disposition: testpilotspb.RUN_DISPOSITION_COMPLETED, Events: []*testpilotspb.RunEvent{event(1, 0, testpilotspb.RUN_EVENT_KIND_RUN_OPENED)}}
				_, err = monitor.Observe(context.Background(), run.Events[0])
				require.NoError(t, err)
				for _, chunk := range [][]json.RawMessage{fixture.Events[:split], fixture.Events[split:]} {
					for _, encoded := range chunk {
						var evidence testpilotspb.CorrelatedEvidence
						require.NoError(t, protojson.Unmarshal(encoded, &evidence))
						observation := correlatedEvent(t, int64(len(run.Events)+1), &evidence)
						observation.ElapsedMilliseconds = 9000
						run.Events = append(run.Events, observation)
						decision, err := monitor.Observe(context.Background(), observation)
						require.NoError(t, err)
						if decision == execution.Stop {
							run.Disposition = testpilotspb.RUN_DISPOSITION_STOPPED_BY_MONITOR
						}
					}
				}
				closed := event(int64(len(run.Events)+1), 9000, testpilotspb.RUN_EVENT_KIND_RUN_CLOSED)
				closed.ExecutionIncomplete = fixture.Incomplete
				if fixture.Incomplete {
					run.Disposition = testpilotspb.RUN_DISPOSITION_INCOMPLETE
				}
				run.Events = append(run.Events, closed)
				decision, err := monitor.Observe(context.Background(), closed)
				require.NoError(t, err)
				if decision == execution.Stop && !fixture.Incomplete {
					run.Disposition = testpilotspb.RUN_DISPOSITION_STOPPED_BY_MONITOR
				}
				live, err := monitor.Close(context.Background(), run)
				require.NoError(t, err)
				offline, violations, err := prepared.Evaluate(context.Background(), run)
				require.NoError(t, err)
				require.True(t, proto.Equal(live, offline))
				want := map[int]testpilotspb.RuleVerdictStatus{0: testpilotspb.RULE_VERDICT_STATUS_INCONCLUSIVE, 2: testpilotspb.RULE_VERDICT_STATUS_SATISFIED, 3: testpilotspb.RULE_VERDICT_STATUS_VIOLATED}[fixture.Expected]
				require.Equal(t, want, live.Rules[0].Status)
				if want == testpilotspb.RULE_VERDICT_STATUS_VIOLATED {
					// The violation names the evidence whose release resolved the obligation, by
					// its kind and the event that carried it, or nothing for a violation found at
					// closure with the obligation still pending.
					require.Len(t, violations, 1)
					require.Equal(t, live.Rules[0].RuleId, violations[0].RuleID)
					require.Empty(t, violations[0].ObservationIDs)
					if violations[0].Sequence > 0 {
						var evidence testpilotspb.CorrelatedEvidence
						require.NoError(t, run.Events[violations[0].Sequence-1].Observations[0].Value.GetMessageValue().UnmarshalTo(&evidence))
						require.Equal(t, evidence.GetKind(), violations[0].Kind)
						violationsWithEvidence.Add(1)
					} else {
						require.Empty(t, violations[0].Kind)
					}
				} else {
					require.Empty(t, violations)
				}
				live.Rules[0].Status = testpilotspb.RULE_VERDICT_STATUS_UNSPECIFIED
				replayed, _, err := prepared.Evaluate(context.Background(), run)
				require.NoError(t, err)
				require.True(t, proto.Equal(offline, replayed))
			}
		})
	}
	require.Positive(t, violationsWithEvidence.Load(), "no violated fixture named its evidence")
}

func TestCorrelatedResourceBoundaries(t *testing.T) {
	for _, tc := range []struct {
		name     string
		limit    func(*testpilotspb.CorrelatedLimits, int64)
		boundary int64
	}{
		{"event-size", func(l *testpilotspb.CorrelatedLimits, n int64) { l.MaxEventBytes = n }, 19},
		{"obligation-work", func(l *testpilotspb.CorrelatedLimits, n int64) { l.MaxObligationWork = n }, 17},
		{"projection-work", func(l *testpilotspb.CorrelatedLimits, n int64) { l.MaxProjectionWork = n }, 1600},
		{"retained-support", func(l *testpilotspb.CorrelatedLimits, n int64) { l.MaxSupport = n }, 5},
	} {
		for _, delta := range []int64{-1, 0} {
			t.Run(fmt.Sprintf("%s/%d", tc.name, delta), func(t *testing.T) {
				c, catalog, view, ceiling, correlated := correlatedFixture(t, 0)
				tc.limit(correlated, tc.boundary+delta)
				p, err := Prepare(c, catalog, view, ceiling, correlated)
				require.NoError(t, err)
				e, err := p.newEvaluator(context.Background(), view)
				require.NoError(t, err)
				_, err = e.Observe(context.Background(), event(1, 0, testpilotspb.RUN_EVENT_KIND_RUN_OPENED))
				require.NoError(t, err)
				_, err = e.Observe(context.Background(), correlatedEvent(t, 2, correlatedEvidence(0, "both", "a")))
				if delta < 0 {
					require.Error(t, err)
					require.Zero(t, e.correlated.transitions)
					require.Empty(t, e.correlated.accepted)
				} else {
					require.NoError(t, err)
					require.EqualValues(t, 1, e.correlated.transitions)
				}
			})
		}
	}
}

func TestCorrelatedCombinedStaticLimits(t *testing.T) {
	for name, constrain := range map[string]func(*testpilotspb.ContractLimits){
		"rules":         func(l *testpilotspb.ContractLimits) { l.MaxRules = 1 },
		"states":        func(l *testpilotspb.ContractLimits) { l.MaxStates = 3 },
		"transitions":   func(l *testpilotspb.ContractLimits) { l.MaxTransitions = 4 },
		"captures":      func(l *testpilotspb.ContractLimits) { l.MaxCaptures = 16 },
		"capture-bytes": func(l *testpilotspb.ContractLimits) { l.MaxCaptureBytes = 10496 },
	} {
		t.Run(name, func(t *testing.T) {
			c, catalog, view, ceiling, correlated := correlatedFixture(t, 1)
			ordinary, _, _, _ := fixture(t)
			addCapture(ordinary.Rules[0])
			c.Rules = ordinary.Rules
			correlated.MaxSemanticTransitions = 4
			constrain(ceiling)
			_, err := Prepare(c, catalog, view, ceiling, correlated)
			require.Error(t, err)
		})
	}
}

func TestCorrelatedRunCeilingsAreAtomic(t *testing.T) {
	for _, tc := range []struct {
		name          string
		configure     func(*testpilotspb.CorrelatedLimits)
		first, second *testpilotspb.CorrelatedEvidence
	}{
		{"buffer", func(l *testpilotspb.CorrelatedLimits) { l.MaxBuffered = 1 }, correlatedEvidence(2, "request", "a"), correlatedEvidence(3, "request", "a")},
		{"keys", func(l *testpilotspb.CorrelatedLimits) { l.MaxKeys = 1 }, correlatedEvidence(0, "request", "a"), correlatedEvidence(1, "request", "b")},
		{"obligations", func(l *testpilotspb.CorrelatedLimits) { l.MaxObligations = 1 }, correlatedEvidence(0, "request", "a"), correlatedEvidence(1, "request", "a")},
		{"transitions", func(l *testpilotspb.CorrelatedLimits) { l.MaxSemanticTransitions = 1 }, correlatedEvidence(0, "request", "a"), correlatedEvidence(1, "tick", "a")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c, catalog, view, ceiling, correlated := correlatedFixture(t, 5)
			tc.configure(correlated)
			p, err := Prepare(c, catalog, view, ceiling, correlated)
			require.NoError(t, err)
			e, err := p.newEvaluator(context.Background(), view)
			require.NoError(t, err)
			_, err = e.Observe(context.Background(), event(1, 0, testpilotspb.RUN_EVENT_KIND_RUN_OPENED))
			require.NoError(t, err)
			_, err = e.Observe(context.Background(), correlatedEvent(t, 2, tc.first))
			require.NoError(t, err)
			before := e.correlated
			_, err = e.Observe(context.Background(), correlatedEvent(t, 3, tc.second))
			require.Error(t, err)
			require.Same(t, before, e.correlated)
			require.Equal(t, testpilotspb.RULE_VERDICT_STATUS_INCONCLUSIVE, e.result.Rules[0].Status)
		})
	}
}

func TestCorrelatedMigrationRejectsStaleReader(t *testing.T) {
	c, _, _, _, _ := correlatedFixture(t, 1)
	wire, err := protojson.Marshal(c)
	require.NoError(t, err)
	file := protodesc.ToFileDescriptorProto(testpilotspb.File_temporal_server_api_testpilot_v1_contract_proto)
	for _, message := range file.MessageType {
		if message.GetName() == "Contract" {
			message.Field = message.Field[:2]
		}
	}
	stale, err := protodesc.NewFile(file, protoregistry.GlobalFiles)
	require.NoError(t, err)
	err = protojson.Unmarshal(wire, dynamicpb.NewMessage(stale.Messages().ByName("Contract")))
	require.ErrorContains(t, err, "unknown field")
}

func TestCorrelatedViolationSurvivesEvaluatorAndCleanupFailure(t *testing.T) {
	c, catalog, view, ceiling, correlated := correlatedFixture(t, 0)
	p, err := Prepare(c, catalog, view, ceiling, correlated)
	require.NoError(t, err)
	e, err := p.newEvaluator(context.Background(), view)
	require.NoError(t, err)
	run := &testpilotspb.Run{RunId: "one", ProgramId: "correlated.program", Disposition: testpilotspb.RUN_DISPOSITION_INCOMPLETE,
		Events:  []*testpilotspb.RunEvent{event(1, 0, testpilotspb.RUN_EVENT_KIND_RUN_OPENED), correlatedEvent(t, 2, correlatedEvidence(0, "request", "a"))},
		Cleanup: &testpilotspb.CleanupOutcome{Status: testpilotspb.CLEANUP_STATUS_FAILED}}
	for _, observation := range run.Events {
		_, err = e.Observe(context.Background(), observation)
		require.NoError(t, err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	failed := event(3, 0, testpilotspb.RUN_EVENT_KIND_DIAGNOSTIC)
	_, err = e.Observe(ctx, failed)
	require.ErrorIs(t, err, context.Canceled)
	run.Events = append(run.Events, failed, event(4, 0, testpilotspb.RUN_EVENT_KIND_RUN_CLOSED))
	run.EvaluationFailure = &testpilotspb.Run_EvaluationFailureSequence{EvaluationFailureSequence: 3}
	live, err := e.Close(context.Background(), run)
	require.NoError(t, err)
	offline, _, err := p.Evaluate(context.Background(), run)
	require.NoError(t, err)
	require.True(t, proto.Equal(live, offline))
	require.Equal(t, testpilotspb.VERDICT_STATUS_VIOLATED, live.Status)
	require.Equal(t, []int64{2}, live.SupportingEventSequences)
}

func TestCorrelatedPreparedConcurrentIsolation(t *testing.T) {
	c, catalog, view, ceiling, correlated := correlatedFixture(t, 1)
	p, err := Prepare(c, catalog, view, ceiling, correlated)
	require.NoError(t, err)
	c.Correlated.Rules[0].Bound = 0
	for _, kind := range []string{"both", "request"} {
		t.Run(kind, func(t *testing.T) {
			t.Parallel()
			e, err := p.newEvaluator(context.Background(), view)
			require.NoError(t, err)
			_, err = e.Observe(context.Background(), event(1, 0, testpilotspb.RUN_EVENT_KIND_RUN_OPENED))
			require.NoError(t, err)
			_, err = e.Observe(context.Background(), correlatedEvent(t, 2, correlatedEvidence(0, kind, "a")))
			require.NoError(t, err)
			want := testpilotspb.RULE_VERDICT_STATUS_INCONCLUSIVE
			if kind == "both" {
				want = testpilotspb.RULE_VERDICT_STATUS_SATISFIED
			}
			require.Equal(t, want, e.result.Rules[0].Status)
		})
	}
}

func TestCorrelatedConfirmedSubmissionAndFieldPolicies(t *testing.T) {
	for _, valid := range []bool{false, true} {
		t.Run(fmt.Sprint(valid), func(t *testing.T) {
			c, catalog, view, ceiling, correlated := correlatedFixture(t, 1)
			request := c.Correlated.ProjectionRules[0]
			request.Submission = c.Correlated.Transitions[0].Action
			request.Fields = []*testpilotspb.CorrelatedFieldPolicy{{FieldId: "payload", Type: &testpilotspb.ScalarType{Kind: testpilotspb.SCALAR_KIND_TEXT}, Disposition: testpilotspb.CORRELATED_FIELD_DISPOSITION_RETAIN}}
			c.Correlated.ProjectionRules = append(c.Correlated.ProjectionRules, &testpilotspb.CorrelatedProjectionRule{Kind: "submit", Meaning: testpilotspb.CORRELATED_EVIDENCE_MEANING_SUBMISSION, Submission: request.Submission})
			p, err := Prepare(c, catalog, view, ceiling, correlated)
			require.NoError(t, err)
			e, err := p.newEvaluator(context.Background(), view)
			require.NoError(t, err)
			_, err = e.Observe(context.Background(), event(1, 0, testpilotspb.RUN_EVENT_KIND_RUN_OPENED))
			require.NoError(t, err)
			_, err = e.Observe(context.Background(), correlatedEvent(t, 2, correlatedEvidence(0, "submit", "a")))
			require.NoError(t, err)
			require.Zero(t, e.correlated.transitions)
			confirmed := correlatedEvidence(1, "request", "a")
			confirmed.Fields = []*testpilotspb.NamedValue{{FieldId: "payload", Value: &testpilotspb.Value{Value: &testpilotspb.Value_TextValue{TextValue: "value"}}}}
			if valid {
				confirmed.Parents = []*testpilotspb.CorrelatedIdentity{correlatedEvidence(0, "submit", "a").Identity}
			}
			_, err = e.Observe(context.Background(), correlatedEvent(t, 3, confirmed))
			if valid {
				require.NoError(t, err)
				require.EqualValues(t, 1, e.correlated.transitions)
				require.Equal(t, []int64{2, 3}, e.result.Rules[0].SupportingEventSequences)
			} else {
				require.Error(t, err)
				require.Zero(t, e.correlated.transitions)
			}
		})
	}
}

func TestCorrelatedPhysicalObservationSizeRemainsBounded(t *testing.T) {
	c, catalog, view, ceiling, correlated := correlatedFixture(t, 1)
	correlated.MaxEventBytes = view.Limits().MaxResponseBytes
	ceiling.MaxCaptureBytes = 131072
	ceiling.MaxCaptureBytes = 131072
	c.Correlated.ProjectionRules[1].Fields = []*testpilotspb.CorrelatedFieldPolicy{{FieldId: "payload", Type: &testpilotspb.ScalarType{Kind: testpilotspb.SCALAR_KIND_TEXT}, Disposition: testpilotspb.CORRELATED_FIELD_DISPOSITION_RETAIN}}
	p, err := Prepare(c, catalog, view, ceiling, correlated)
	require.NoError(t, err)
	e, err := p.newEvaluator(context.Background(), view)
	require.NoError(t, err)
	_, err = e.Observe(context.Background(), event(1, 0, testpilotspb.RUN_EVENT_KIND_RUN_OPENED))
	require.NoError(t, err)
	evidence := correlatedEvidence(0, "both", "a")
	evidence.Fields = []*testpilotspb.NamedValue{{FieldId: "payload", Value: &testpilotspb.Value{Value: &testpilotspb.Value_TextValue{TextValue: strings.Repeat("💡", 2000)}}}}
	require.Less(t, evidenceSize(&admittedCorrelatedEvidence{CorrelatedEvidence: evidence, supportingEventSequences: []int64{2}}), correlated.MaxEventBytes)
	_, err = e.Observe(context.Background(), correlatedEvent(t, 2, evidence))
	require.Error(t, err)
	require.Empty(t, e.correlated.accepted)
}

// A STATE condition reads the state and every field the machine keeps, so a Model whose states are
// structured is compared field by field rather than by taking one spelling apart. The same
// transition answers a reference to the state's own definition and to each of its fields, and
// answers a definition it carries at neither.
func TestCorrelatedStateConditionReadsMachineFields(t *testing.T) {
	stateCondition := func(definitionID, text string) *testpilotspb.Expression {
		step := &testpilotspb.Expression{Expression: &testpilotspb.Expression_Reference{
			Reference: &testpilotspb.Reference{Reference: &testpilotspb.Reference_CorrelatedStep{
				CorrelatedStep: &testpilotspb.CorrelatedStepReference{
					Field:        testpilotspb.CORRELATED_STEP_FIELD_STATE,
					DefinitionId: definitionID,
				}}}}}
		if text == "" {
			return &testpilotspb.Expression{Expression: &testpilotspb.Expression_Present{
				Present: &testpilotspb.PresentExpression{Operand: step}}}
		}
		return &testpilotspb.Expression{Expression: &testpilotspb.Expression_Compare{
			Compare: &testpilotspb.CompareExpression{
				Operator: testpilotspb.COMPARISON_OPERATOR_EQUAL,
				Left:     step,
				Right: &testpilotspb.Expression{Expression: &testpilotspb.Expression_Literal{
					Literal: &testpilotspb.Value{Value: &testpilotspb.Value_TextValue{TextValue: text}}}},
			}}}
	}
	structured := &testpilotspb.CorrelatedTransition{
		State: &testpilotspb.ModelValue{DefinitionId: "state", Value: "backingOff-1"},
		StateFields: []*testpilotspb.ModelValue{
			{DefinitionId: "phase", Value: "backingOff"},
			{DefinitionId: "attempts", Value: "1"},
		},
	}
	atomOnly := &testpilotspb.CorrelatedTransition{
		State: &testpilotspb.ModelValue{DefinitionId: "state", Value: "backingOff-1"},
	}

	require.True(t, predicate(stateCondition("state", "backingOff-1"), structured))
	require.True(t, predicate(stateCondition("phase", "backingOff"), structured))
	require.True(t, predicate(stateCondition("attempts", "1"), structured))
	require.True(t, predicate(stateCondition("attempts", ""), structured))

	// The field is read as itself, so the wrong member of the right field is false rather than a
	// substring of the state's spelling.
	require.False(t, predicate(stateCondition("phase", "scheduled"), structured))
	require.False(t, predicate(stateCondition("attempts", "0"), structured))
	require.False(t, predicate(stateCondition("cancel", ""), structured))

	// A Model whose states are atoms carries no fields and reads exactly as it did.
	require.True(t, predicate(stateCondition("state", "backingOff-1"), atomOnly))
	require.False(t, predicate(stateCondition("phase", "backingOff"), atomOnly))
}
