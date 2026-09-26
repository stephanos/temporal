package verification

import (
	"context"
	"slices"
	"testing"

	"github.com/stretchr/testify/require"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/testpilot/internal/execution"
	"go.temporal.io/server/common/testing/testpilot/internal/ir"
	"google.golang.org/protobuf/proto"
)

// Each evidence kind declares its own field, exactly as the two distinct operand paths a model
// correlation names are projected. Expected verdicts below are written out from these declarations
// rather than read back from the interpreter.
const (
	capturedField = "requested"
	repliedField  = "replied"
)

func correlatedText(text string) *testpilotspb.Value {
	return &testpilotspb.Value{Value: &testpilotspb.Value_TextValue{TextValue: text}}
}
func correlatedReference(reference *testpilotspb.Reference) *testpilotspb.Expression {
	return &testpilotspb.Expression{Expression: &testpilotspb.Expression_Reference{Reference: reference}}
}
func correlatedFieldOperand(id string) *testpilotspb.Expression {
	return correlatedReference(&testpilotspb.Reference{Reference: &testpilotspb.Reference_EvidenceFieldId{EvidenceFieldId: id}})
}
func correlatedCaptureOperand(id string, ordinal int64) *testpilotspb.Expression {
	return correlatedReference(&testpilotspb.Reference{Reference: &testpilotspb.Reference_CorrelatedCapture{CorrelatedCapture: &testpilotspb.CorrelatedCaptureReference{CaptureId: id, Ordinal: ordinal}}})
}
func correlatedLiteralOperand(text string) *testpilotspb.Expression {
	return correlatedLiteral(correlatedText(text))
}
func correlatedLiteral(value *testpilotspb.Value) *testpilotspb.Expression {
	return &testpilotspb.Expression{Expression: &testpilotspb.Expression_Literal{Literal: value}}
}
func correlationComparison(operator testpilotspb.ComparisonOperator, left, right *testpilotspb.Expression) *testpilotspb.Expression {
	return &testpilotspb.Expression{Expression: &testpilotspb.Expression_Compare{Compare: &testpilotspb.CompareExpression{Operator: operator, Left: left, Right: right}}}
}
func correlatedTriggered() *testpilotspb.Expression {
	return stepPresent(testpilotspb.CORRELATED_STEP_FIELD_ACTION, "request")
}
func correlatedAny(operands ...*testpilotspb.Expression) *testpilotspb.Expression {
	return &testpilotspb.Expression{Expression: &testpilotspb.Expression_Any{Any: &testpilotspb.AnyExpression{Operands: operands}}}
}
func correlatedAll(operands ...*testpilotspb.Expression) *testpilotspb.Expression {
	return &testpilotspb.Expression{Expression: &testpilotspb.Expression_All{All: &testpilotspb.AllExpression{Operands: operands}}}
}

// keyedCorrelation is the qualifying shape: the request step creates the occurrence its own
// correlation would read, so the trigger disjunct decides it before the capture operand is reached.
func keyedCorrelation(ordinal int64) *testpilotspb.Expression {
	return correlatedAny(correlatedTriggered(), correlationComparison(testpilotspb.COMPARISON_OPERATOR_EQUAL, correlatedFieldOperand(repliedField), correlatedCaptureOperand("seen", ordinal)))
}

func correlatedCaptureFixture(t *testing.T, bound, lifetime int64, correlation *testpilotspb.Expression) (*testpilotspb.Contract, *ir.Catalog, execution.ProgramView, *testpilotspb.ContractLimits, *testpilotspb.CorrelatedLimits) {
	t.Helper()
	c, catalog, view, ceiling, correlated := correlatedFixture(t, bound)
	policy := func(id string) []*testpilotspb.CorrelatedFieldPolicy {
		return []*testpilotspb.CorrelatedFieldPolicy{{FieldId: id, Type: &testpilotspb.ScalarType{Kind: testpilotspb.SCALAR_KIND_TEXT}, Disposition: testpilotspb.CORRELATED_FIELD_DISPOSITION_RETAIN}}
	}
	for _, rule := range c.Correlated.ProjectionRules {
		switch rule.Kind {
		case "request", "both":
			rule.Fields = policy(capturedField)
		case "reply":
			rule.Fields = policy(repliedField)
		default:
			// "tick" and "poll" declare no field, so a correlation reading one finds none.
		}
	}
	clause := c.Correlated.Rules[0]
	if lifetime > 0 {
		clause.Captures = []*testpilotspb.CorrelatedCaptureDeclaration{{CaptureId: "seen", FieldId: capturedField, Lifetime: lifetime}}
	}
	clause.Correlation = correlation
	correlated.MaxCaptures = 8
	correlated.MaxCorrelationDepth = 4
	return c, catalog, view, ceiling, correlated
}

func correlatedFieldEvidence(ordinal int64, kind, operation, value string) *testpilotspb.CorrelatedEvidence {
	e := correlatedEvidence(ordinal, kind, operation)
	switch kind {
	case "request", "both":
		e.Fields = []*testpilotspb.NamedValue{{FieldId: capturedField, Value: correlatedText(value)}}
	case "reply":
		e.Fields = []*testpilotspb.NamedValue{{FieldId: repliedField, Value: correlatedText(value)}}
	default:
		// A kind whose rule declares no field supplies none.
	}
	return e
}

type correlatedStep struct {
	kind, operation, value string
}

// observeCorrelated replays the steps live and reports the clause verdict, or the first rejection.
func observeCorrelated(t *testing.T, c *testpilotspb.Contract, catalog *ir.Catalog, view execution.ProgramView, ceiling *testpilotspb.ContractLimits, correlated *testpilotspb.CorrelatedLimits, steps []correlatedStep) (testpilotspb.RuleVerdictStatus, error) {
	t.Helper()
	p, err := Prepare(c, catalog, view, ceiling, correlated)
	if err != nil {
		return testpilotspb.RULE_VERDICT_STATUS_UNSPECIFIED, err
	}
	e, err := p.newEvaluator(context.Background(), view)
	require.NoError(t, err)
	_, err = e.Observe(context.Background(), event(1, 0, testpilotspb.RUN_EVENT_KIND_RUN_OPENED))
	require.NoError(t, err)
	for i, step := range steps {
		evidence := correlatedFieldEvidence(int64(i), step.kind, step.operation, step.value)
		if _, err = e.Observe(context.Background(), correlatedEvent(t, int64(i+2), evidence)); err != nil {
			return e.result.Rules[0].Status, err
		}
	}
	return e.result.Rules[0].Status, nil
}

func TestCorrelatedCapturesCorrelateOperationSteps(t *testing.T) {
	for _, tc := range []struct {
		name        string
		bound       int64
		lifetime    int64
		correlation *testpilotspb.Expression
		steps       []correlatedStep
		want        testpilotspb.RuleVerdictStatus
		reject      string
	}{
		{
			name: "correlated-reply", bound: 1, lifetime: 2, correlation: keyedCorrelation(0),
			steps: []correlatedStep{{"request", "a", "1"}, {"reply", "a", "1"}},
			want:  testpilotspb.RULE_VERDICT_STATUS_SATISFIED,
		},
		{
			name: "uncorrelated-reply", bound: 1, lifetime: 2, correlation: keyedCorrelation(0),
			steps:  []correlatedStep{{"request", "a", "1"}, {"reply", "a", "2"}},
			reject: "correlation rejected this operation's step",
		},
		{
			name: "no-correlation-admits-any-reply", bound: 1, lifetime: 2,
			steps: []correlatedStep{{"request", "a", "1"}, {"reply", "a", "2"}},
			want:  testpilotspb.RULE_VERDICT_STATUS_SATISFIED,
		},
		{
			name: "future-occurrence", bound: 1, lifetime: 2, correlation: keyedCorrelation(1),
			steps:  []correlatedStep{{"request", "a", "1"}, {"reply", "a", "1"}},
			reject: "correlation rejected this operation's step",
		},
		{
			name: "foreign-operation", bound: 1, lifetime: 2, correlation: keyedCorrelation(0),
			steps:  []correlatedStep{{"request", "a", "1"}, {"reply", "b", "1"}},
			reject: "correlation rejected this operation's step",
		},
		{
			name: "occurrence-zero-is-immutable", bound: 2, lifetime: 2, correlation: keyedCorrelation(0),
			steps: []correlatedStep{{"request", "a", "1"}, {"request", "a", "2"}, {"reply", "a", "1"}},
			want:  testpilotspb.RULE_VERDICT_STATUS_SATISFIED,
		},
		{
			name: "latest-match-is-not-occurrence-zero", bound: 2, lifetime: 2, correlation: keyedCorrelation(0),
			steps:  []correlatedStep{{"request", "a", "1"}, {"request", "a", "2"}, {"reply", "a", "2"}},
			reject: "correlation rejected this operation's step",
		},
		{
			name: "second-occurrence", bound: 2, lifetime: 2, correlation: keyedCorrelation(1),
			steps: []correlatedStep{{"request", "a", "1"}, {"request", "a", "2"}, {"reply", "a", "2"}},
			want:  testpilotspb.RULE_VERDICT_STATUS_SATISFIED,
		},
		{
			name: "missing-field-operand", bound: 1, lifetime: 2, correlation: keyedCorrelation(0),
			steps:  []correlatedStep{{"request", "a", "1"}, {"tick", "a", ""}},
			reject: "correlation rejected this operation's step",
		},
		{
			name: "missing-operand-is-not-unequal", bound: 1, lifetime: 2,
			correlation: correlatedAny(correlatedTriggered(),
				correlationComparison(testpilotspb.COMPARISON_OPERATOR_NOT_EQUAL, correlatedFieldOperand(repliedField), correlatedCaptureOperand("seen", 1))),
			steps:  []correlatedStep{{"request", "a", "1"}, {"reply", "a", "2"}},
			reject: "correlation rejected this operation's step",
		},
		{
			name: "missing-operand-leaves-a-disjunction-open", bound: 1, lifetime: 2,
			correlation: correlatedAny(correlatedTriggered(),
				correlationComparison(testpilotspb.COMPARISON_OPERATOR_EQUAL, correlatedFieldOperand(repliedField), correlatedCaptureOperand("seen", 1)),
				correlationComparison(testpilotspb.COMPARISON_OPERATOR_EQUAL, correlatedFieldOperand(repliedField), correlatedCaptureOperand("seen", 0))),
			steps: []correlatedStep{{"request", "a", "1"}, {"reply", "a", "1"}},
			want:  testpilotspb.RULE_VERDICT_STATUS_SATISFIED,
		},
		{
			name:     "conjunction-stops-at-its-first-false-operand",
			bound:    1,
			lifetime: 2,
			correlation: correlatedAny(correlatedTriggered(), correlatedAll(
				correlationComparison(testpilotspb.COMPARISON_OPERATOR_EQUAL, correlatedFieldOperand(repliedField), correlatedLiteralOperand("9")),
				correlationComparison(testpilotspb.COMPARISON_OPERATOR_EQUAL, correlatedFieldOperand(repliedField), correlatedCaptureOperand("seen", 1)))),
			steps:  []correlatedStep{{"request", "a", "1"}, {"reply", "a", "1"}},
			reject: "correlation rejected this operation's step",
		},
		{
			name:     "nested-conjunction-admits",
			bound:    1,
			lifetime: 2,
			correlation: correlatedAny(correlatedTriggered(), correlatedAll(
				correlationComparison(testpilotspb.COMPARISON_OPERATOR_EQUAL, correlatedFieldOperand(repliedField), correlatedLiteralOperand("1")),
				correlationComparison(testpilotspb.COMPARISON_OPERATOR_NOT_EQUAL, correlatedFieldOperand(repliedField), correlatedLiteralOperand("2")))),
			steps: []correlatedStep{{"request", "a", "1"}, {"reply", "a", "1"}},
			want:  testpilotspb.RULE_VERDICT_STATUS_SATISFIED,
		},
		{
			name: "retained-occurrences-never-move-a-countdown", bound: 0, lifetime: 2, correlation: keyedCorrelation(0),
			steps: []correlatedStep{{"request", "a", "1"}, {"reply", "a", "1"}},
			want:  testpilotspb.RULE_VERDICT_STATUS_VIOLATED,
		},
		{
			name: "lifetime-exhausted", bound: 2, lifetime: 1, correlation: keyedCorrelation(0),
			steps:  []correlatedStep{{"request", "a", "1"}, {"request", "a", "2"}},
			reject: "capture lifetime exhausted",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c, catalog, view, ceiling, correlated := correlatedCaptureFixture(t, tc.bound, tc.lifetime, tc.correlation)
			status, err := observeCorrelated(t, c, catalog, view, ceiling, correlated, tc.steps)
			if tc.reject != "" {
				require.ErrorContains(t, err, tc.reject)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.want, status)
		})
	}
}

func TestCorrelatedCaptureAdmissionIsAtomic(t *testing.T) {
	c, catalog, view, ceiling, correlated := correlatedCaptureFixture(t, 1, 2, keyedCorrelation(0))
	p, err := Prepare(c, catalog, view, ceiling, correlated)
	require.NoError(t, err)
	e, err := p.newEvaluator(context.Background(), view)
	require.NoError(t, err)
	_, err = e.Observe(context.Background(), event(1, 0, testpilotspb.RUN_EVENT_KIND_RUN_OPENED))
	require.NoError(t, err)
	_, err = e.Observe(context.Background(), correlatedEvent(t, 2, correlatedFieldEvidence(0, "request", "a", "1")))
	require.NoError(t, err)
	before := proto.CloneOf(e.result)
	require.EqualValues(t, 1, e.correlated.capturedValues)
	_, err = e.Observe(context.Background(), correlatedEvent(t, 3, correlatedFieldEvidence(1, "reply", "a", "2")))
	require.ErrorContains(t, err, "correlation rejected this operation's step")
	// A rejected append publishes no state: the retained occurrence, the semantic transition count
	// and the clause verdict are exactly the ones the admitted prefix established.
	require.EqualValues(t, 1, e.correlated.capturedValues)
	require.EqualValues(t, 1, e.correlated.transitions)
	require.True(t, proto.Equal(before.Rules[0], e.result.Rules[0]))
	occurrences := e.correlated.operations["a"].captures
	require.Len(t, occurrences, 1)
	require.Equal(t, retainedCapture{capture: "seen", ordinal: 0, value: occurrences[0].value}, occurrences[0])
	require.True(t, proto.Equal(correlatedText("1"), occurrences[0].value))
}

func TestCorrelatedCaptureCeilingRejects(t *testing.T) {
	c, catalog, view, ceiling, correlated := correlatedCaptureFixture(t, 2, 4, keyedCorrelation(0))
	correlated.MaxCaptures = 1
	_, err := observeCorrelated(t, c, catalog, view, ceiling, correlated, []correlatedStep{{"request", "a", "1"}, {"request", "a", "2"}})
	require.ErrorContains(t, err, "ceiling")
}

func TestCorrelatedCapturePrepareRejectsUnsupportedDeclarations(t *testing.T) {
	for name, tc := range map[string]struct {
		mutate func(*testpilotspb.CorrelatedContract)
		limit  func(*testpilotspb.CorrelatedLimits)
		reason string
	}{
		"unretained-capture-field": {
			mutate: func(s *testpilotspb.CorrelatedContract) { s.Rules[0].Captures[0].FieldId = "absent" },
			reason: "invalid capture declaration",
		},
		"redacted-capture-field": {
			mutate: func(s *testpilotspb.CorrelatedContract) {
				for _, rule := range s.ProjectionRules {
					for _, f := range rule.Fields {
						if f.FieldId == capturedField {
							f.Disposition = testpilotspb.CORRELATED_FIELD_DISPOSITION_REDACT
						}
					}
				}
			},
			reason: "invalid capture declaration",
		},
		"zero-lifetime": {
			mutate: func(s *testpilotspb.CorrelatedContract) { s.Rules[0].Captures[0].Lifetime = 0 },
			reason: "invalid capture declaration",
		},
		"repeated-capture-id": {
			mutate: func(s *testpilotspb.CorrelatedContract) {
				s.Rules[0].Captures = append(s.Rules[0].Captures, proto.CloneOf(s.Rules[0].Captures[0]))
			},
			reason: "invalid capture declaration",
		},
		"ordinal-beyond-lifetime": {
			mutate: func(s *testpilotspb.CorrelatedContract) { s.Rules[0].Correlation = keyedCorrelation(2) },
			reason: "unbound capture reference",
		},
		"unknown-capture": {
			mutate: func(s *testpilotspb.CorrelatedContract) {
				s.Rules[0].Correlation = correlationComparison(testpilotspb.COMPARISON_OPERATOR_EQUAL, correlatedFieldOperand(repliedField), correlatedCaptureOperand("other", 0))
			},
			reason: "unbound capture reference",
		},
		"unretained-operand": {
			mutate: func(s *testpilotspb.CorrelatedContract) {
				s.Rules[0].Correlation = correlationComparison(testpilotspb.COMPARISON_OPERATOR_EQUAL, correlatedFieldOperand("absent"), correlatedCaptureOperand("seen", 0))
			},
			reason: "unretained correlation field operand",
		},
		"empty-group": {
			mutate: func(s *testpilotspb.CorrelatedContract) { s.Rules[0].Correlation = correlatedAny() },
			reason: "empty correlation group",
		},
		"unsupported-operator": {
			mutate: func(s *testpilotspb.CorrelatedContract) {
				s.Rules[0].Correlation = correlationComparison(testpilotspb.COMPARISON_OPERATOR_UNSPECIFIED, correlatedLiteralOperand("1"), correlatedCaptureOperand("seen", 0))
			},
			reason: "unsupported comparison operator",
		},
		"unsupported-literal": {
			mutate: func(s *testpilotspb.CorrelatedContract) {
				s.Rules[0].Correlation = correlationComparison(testpilotspb.COMPARISON_OPERATOR_EQUAL, correlatedLiteral(&testpilotspb.Value{Value: &testpilotspb.Value_UnsignedIntegerValue{UnsignedIntegerValue: "01"}}), correlatedCaptureOperand("seen", 0))
			},
			reason: "unsupported correlation literal",
		},
		"oversized-unsigned-literal": {
			mutate: func(s *testpilotspb.CorrelatedContract) {
				s.Rules[0].Correlation = correlationComparison(testpilotspb.COMPARISON_OPERATOR_EQUAL, correlatedLiteral(&testpilotspb.Value{Value: &testpilotspb.Value_UnsignedIntegerValue{UnsignedIntegerValue: "18446744073709551616"}}), correlatedCaptureOperand("seen", 0))
			},
			reason: "unsupported correlation literal",
		},
		"repeated-capture-id-across-clauses": {
			mutate: func(s *testpilotspb.CorrelatedContract) {
				second := proto.CloneOf(s.Rules[0])
				second.RuleId = "second"
				s.Rules = append(s.Rules, second)
			},
			reason: "invalid capture declaration",
		},
		"capture-of-another-clause": {
			mutate: func(s *testpilotspb.CorrelatedContract) {
				second := proto.CloneOf(s.Rules[0])
				second.RuleId = "second"
				second.Captures = nil
				s.Rules = append(s.Rules, second)
			},
			reason: "unbound capture reference",
		},
		"mismatched-operand-kinds": {
			mutate: func(s *testpilotspb.CorrelatedContract) {
				s.Rules[0].Correlation = correlationComparison(testpilotspb.COMPARISON_OPERATOR_EQUAL, correlatedFieldOperand(repliedField), correlatedLiteral(&testpilotspb.Value{Value: &testpilotspb.Value_UnsignedIntegerValue{UnsignedIntegerValue: "1"}}))
			},
			reason: "incompatible correlation operand types",
		},
		"mismatched-capture-kind": {
			mutate: func(s *testpilotspb.CorrelatedContract) {
				for _, rule := range s.ProjectionRules {
					if rule.Kind == "reply" {
						rule.Fields[0].Type = &testpilotspb.ScalarType{Kind: testpilotspb.SCALAR_KIND_UINT64}
					}
				}
			},
			reason: "incompatible correlation operand types",
		},
		"ambiguous-retained-field-kind": {
			mutate: func(s *testpilotspb.CorrelatedContract) {
				for _, rule := range s.ProjectionRules {
					if rule.Kind == "reply" {
						rule.Fields = append(rule.Fields, &testpilotspb.CorrelatedFieldPolicy{FieldId: capturedField, Type: &testpilotspb.ScalarType{Kind: testpilotspb.SCALAR_KIND_UINT64}, Disposition: testpilotspb.CORRELATED_FIELD_DISPOSITION_RETAIN})
					}
				}
			},
			reason: "invalid capture declaration",
		},
		"depth-exhausted": {
			limit:  func(l *testpilotspb.CorrelatedLimits) { l.MaxCorrelationDepth = 1 },
			reason: "correlation depth exhausted",
		},
		"missing-capture-ceiling": {
			limit:  func(l *testpilotspb.CorrelatedLimits) { l.MaxCaptures = 0 },
			reason: "correlated capture limits must be positive",
		},
		"missing-depth-ceiling": {
			limit:  func(l *testpilotspb.CorrelatedLimits) { l.MaxCorrelationDepth = 0 },
			reason: "correlated capture limits must be positive",
		},
	} {
		t.Run(name, func(t *testing.T) {
			c, catalog, view, ceiling, correlated := correlatedCaptureFixture(t, 1, 2, keyedCorrelation(0))
			if tc.mutate != nil {
				tc.mutate(c.Correlated)
			}
			if tc.limit != nil {
				tc.limit(correlated)
			}
			_, err := Prepare(c, catalog, view, ceiling, correlated)
			require.ErrorContains(t, err, tc.reason)
		})
	}
}

// A capability that declares neither captures nor a correlation keeps its exact prior meaning, and is
// admitted under a Profile that leaves both capture ceilings unset.
func TestCorrelatedCapabilityWithoutCapturesIsUnchanged(t *testing.T) {
	c, catalog, view, ceiling, correlated := correlatedFixture(t, 1)
	require.Zero(t, correlated.MaxCaptures)
	require.Zero(t, correlated.MaxCorrelationDepth)
	require.True(t, slices.ContainsFunc(c.Correlated.Rules, func(clause *testpilotspb.CorrelatedRule) bool {
		return len(clause.Captures) == 0 && clause.Correlation == nil
	}))
	p, err := Prepare(c, catalog, view, ceiling, correlated)
	require.NoError(t, err)
	e, err := p.newEvaluator(context.Background(), view)
	require.NoError(t, err)
	_, err = e.Observe(context.Background(), event(1, 0, testpilotspb.RUN_EVENT_KIND_RUN_OPENED))
	require.NoError(t, err)
	for i, kind := range []string{"request", "reply"} {
		_, err = e.Observe(context.Background(), correlatedEvent(t, int64(i+2), correlatedEvidence(int64(i), kind, "a")))
		require.NoError(t, err)
	}
	require.Equal(t, testpilotspb.RULE_VERDICT_STATUS_SATISFIED, e.result.Rules[0].Status)
	require.Zero(t, e.correlated.capturedValues)
}

// Live admission and offline replay agree on a capture-bearing capability at every chunk boundary.
func TestCorrelatedCaptureLiveAndOfflineAgree(t *testing.T) {
	for _, tc := range []struct {
		name  string
		steps []correlatedStep
		want  testpilotspb.RuleVerdictStatus
	}{
		{"one-operation", []correlatedStep{{"request", "a", "1"}, {"reply", "a", "1"}}, testpilotspb.RULE_VERDICT_STATUS_SATISFIED},
		// The second operation correlates its reply against its own retained occurrence, and the
		// first operation's window is still open, so the clause stays unresolved.
		{"interleaved-operations", []correlatedStep{{"request", "a", "1"}, {"request", "b", "2"}, {"reply", "b", "2"}}, testpilotspb.RULE_VERDICT_STATUS_INCONCLUSIVE},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c, catalog, view, ceiling, correlated := correlatedCaptureFixture(t, 2, 2, keyedCorrelation(0))
			prepared, err := Prepare(c, catalog, view, ceiling, correlated)
			require.NoError(t, err)
			for split := 0; split <= len(tc.steps); split++ {
				monitor, err := prepared.New(context.Background(), view)
				require.NoError(t, err)
				run := &testpilotspb.Run{RunId: "one", CaseId: "correlated.case", ProgramId: "correlated.program", Disposition: testpilotspb.RUN_DISPOSITION_COMPLETED, Events: []*testpilotspb.RunEvent{event(1, 0, testpilotspb.RUN_EVENT_KIND_RUN_OPENED)}}
				_, err = monitor.Observe(context.Background(), run.Events[0])
				require.NoError(t, err)
				for _, chunk := range [][]correlatedStep{tc.steps[:split], tc.steps[split:]} {
					for _, step := range chunk {
						i := slices.Index(tc.steps, step)
						observation := correlatedEvent(t, int64(len(run.Events)+1), correlatedFieldEvidence(int64(i), step.kind, step.operation, step.value))
						run.Events = append(run.Events, observation)
						_, err = monitor.Observe(context.Background(), observation)
						require.NoError(t, err)
					}
				}
				closed := event(int64(len(run.Events)+1), 9000, testpilotspb.RUN_EVENT_KIND_RUN_CLOSED)
				run.Events = append(run.Events, closed)
				_, err = monitor.Observe(context.Background(), closed)
				require.NoError(t, err)
				live, err := monitor.Close(context.Background(), run)
				require.NoError(t, err)
				offline, _, err := prepared.Evaluate(context.Background(), run)
				require.NoError(t, err)
				require.True(t, proto.Equal(live, offline))
				require.Equal(t, tc.want, live.Rules[0].Status)
			}
		})
	}
}
