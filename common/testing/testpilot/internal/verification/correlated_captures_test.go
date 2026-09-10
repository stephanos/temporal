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

func scopedText(text string) *testpilotspb.Value {
	return &testpilotspb.Value{Value: &testpilotspb.Value_Text{Text: text}}
}
func scopedFieldOperand(id string) *testpilotspb.CorrelatedOperand {
	return &testpilotspb.CorrelatedOperand{Operand: &testpilotspb.CorrelatedOperand_FieldId{FieldId: id}}
}
func scopedCaptureOperand(id string, ordinal int64) *testpilotspb.CorrelatedOperand {
	return &testpilotspb.CorrelatedOperand{Operand: &testpilotspb.CorrelatedOperand_Capture{Capture: &testpilotspb.CorrelatedCaptureRef{CaptureId: id, Ordinal: ordinal}}}
}
func scopedLiteralOperand(text string) *testpilotspb.CorrelatedOperand {
	return &testpilotspb.CorrelatedOperand{Operand: &testpilotspb.CorrelatedOperand_Literal{Literal: scopedText(text)}}
}
func correlatedComparison(operator testpilotspb.CorrelatedComparisonOperator, left, right *testpilotspb.CorrelatedOperand) *testpilotspb.CorrelatedCorrelation {
	return &testpilotspb.CorrelatedCorrelation{Condition: &testpilotspb.CorrelatedCorrelation_Comparison{Comparison: &testpilotspb.CorrelatedComparison{Operator: operator, Left: left, Right: right}}}
}
func scopedTriggered() *testpilotspb.CorrelatedCorrelation {
	return &testpilotspb.CorrelatedCorrelation{Condition: &testpilotspb.CorrelatedCorrelation_Predicate{Predicate: &testpilotspb.CorrelatedPredicate{Field: testpilotspb.CORRELATED_PREDICATE_FIELD_ACTION, DefinitionId: "request", Constraint: &testpilotspb.CorrelatedPredicate_Present{Present: true}}}}
}
func scopedAny(operands ...*testpilotspb.CorrelatedCorrelation) *testpilotspb.CorrelatedCorrelation {
	return &testpilotspb.CorrelatedCorrelation{Condition: &testpilotspb.CorrelatedCorrelation_Any{Any: &testpilotspb.CorrelatedCorrelationGroup{Operands: operands}}}
}
func scopedAll(operands ...*testpilotspb.CorrelatedCorrelation) *testpilotspb.CorrelatedCorrelation {
	return &testpilotspb.CorrelatedCorrelation{Condition: &testpilotspb.CorrelatedCorrelation_All{All: &testpilotspb.CorrelatedCorrelationGroup{Operands: operands}}}
}

// correlatedCorrelation is the qualifying shape: the request step creates the occurrence its own
// correlation would read, so the trigger disjunct decides it before the capture operand is reached.
func correlatedCorrelation(ordinal int64) *testpilotspb.CorrelatedCorrelation {
	return scopedAny(scopedTriggered(), correlatedComparison(testpilotspb.CORRELATED_COMPARISON_OPERATOR_EQUAL, scopedFieldOperand(repliedField), scopedCaptureOperand("seen", ordinal)))
}

func scopedCaptureFixture(t *testing.T, bound, lifetime int64, correlation *testpilotspb.CorrelatedCorrelation) (*testpilotspb.Contract, *ir.Catalog, execution.ProgramView, *testpilotspb.ContractLimits) {
	t.Helper()
	c, catalog, view, ceiling := correlatedFixture(t, bound)
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
	clause := c.Correlated.Clauses[0]
	if lifetime > 0 {
		clause.Captures = []*testpilotspb.CorrelatedCaptureDeclaration{{CaptureId: "seen", FieldId: capturedField, Lifetime: lifetime}}
	}
	clause.Correlation = correlation
	c.Correlated.Limits.MaxCaptures = 8
	c.Correlated.Limits.MaxCorrelationDepth = 4
	return c, catalog, view, ceiling
}

func scopedFieldEvidence(ordinal int64, kind, operation, value string) *testpilotspb.CorrelatedEvidence {
	e := correlatedEvidence(ordinal, kind, operation)
	switch kind {
	case "request", "both":
		e.Fields = []*testpilotspb.CorrelatedEvidenceField{{FieldId: capturedField, Value: scopedText(value)}}
	case "reply":
		e.Fields = []*testpilotspb.CorrelatedEvidenceField{{FieldId: repliedField, Value: scopedText(value)}}
	default:
		// A kind whose rule declares no field supplies none.
	}
	return e
}

type scopedStep struct {
	kind, operation, value string
}

// observeCorrelated replays the steps live and reports the clause verdict, or the first rejection.
func observeCorrelated(t *testing.T, c *testpilotspb.Contract, catalog *ir.Catalog, view execution.ProgramView, ceiling *testpilotspb.ContractLimits, steps []scopedStep) (testpilotspb.RuleVerdictStatus, error) {
	t.Helper()
	p, err := Prepare(c, catalog, view, ceiling)
	if err != nil {
		return testpilotspb.RULE_VERDICT_STATUS_UNSPECIFIED, err
	}
	e, err := p.newEvaluator(context.Background(), view)
	require.NoError(t, err)
	_, err = e.Observe(context.Background(), event(1, 0, testpilotspb.RUN_EVENT_KIND_RUN_OPENED))
	require.NoError(t, err)
	for i, step := range steps {
		evidence := scopedFieldEvidence(int64(i), step.kind, step.operation, step.value)
		if _, err = e.Observe(context.Background(), scopedEvent(t, int64(i+2), evidence)); err != nil {
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
		correlation *testpilotspb.CorrelatedCorrelation
		steps       []scopedStep
		want        testpilotspb.RuleVerdictStatus
		reject      string
	}{
		{
			name: "correlated-reply", bound: 1, lifetime: 2, correlation: correlatedCorrelation(0),
			steps: []scopedStep{{"request", "a", "1"}, {"reply", "a", "1"}},
			want:  testpilotspb.RULE_VERDICT_STATUS_SATISFIED,
		},
		{
			name: "uncorrelated-reply", bound: 1, lifetime: 2, correlation: correlatedCorrelation(0),
			steps:  []scopedStep{{"request", "a", "1"}, {"reply", "a", "2"}},
			reject: "correlation rejected this operation's step",
		},
		{
			name: "no-correlation-admits-any-reply", bound: 1, lifetime: 2,
			steps: []scopedStep{{"request", "a", "1"}, {"reply", "a", "2"}},
			want:  testpilotspb.RULE_VERDICT_STATUS_SATISFIED,
		},
		{
			name: "future-occurrence", bound: 1, lifetime: 2, correlation: correlatedCorrelation(1),
			steps:  []scopedStep{{"request", "a", "1"}, {"reply", "a", "1"}},
			reject: "missing retained capture occurrence",
		},
		{
			name: "foreign-operation", bound: 1, lifetime: 2, correlation: correlatedCorrelation(0),
			steps:  []scopedStep{{"request", "a", "1"}, {"reply", "b", "1"}},
			reject: "missing retained capture occurrence",
		},
		{
			name: "occurrence-zero-is-immutable", bound: 2, lifetime: 2, correlation: correlatedCorrelation(0),
			steps: []scopedStep{{"request", "a", "1"}, {"request", "a", "2"}, {"reply", "a", "1"}},
			want:  testpilotspb.RULE_VERDICT_STATUS_SATISFIED,
		},
		{
			name: "latest-match-is-not-occurrence-zero", bound: 2, lifetime: 2, correlation: correlatedCorrelation(0),
			steps:  []scopedStep{{"request", "a", "1"}, {"request", "a", "2"}, {"reply", "a", "2"}},
			reject: "correlation rejected this operation's step",
		},
		{
			name: "second-occurrence", bound: 2, lifetime: 2, correlation: correlatedCorrelation(1),
			steps: []scopedStep{{"request", "a", "1"}, {"request", "a", "2"}, {"reply", "a", "2"}},
			want:  testpilotspb.RULE_VERDICT_STATUS_SATISFIED,
		},
		{
			name: "missing-field-operand", bound: 1, lifetime: 2, correlation: correlatedCorrelation(0),
			steps:  []scopedStep{{"request", "a", "1"}, {"tick", "a", ""}},
			reject: "missing correlation field operand",
		},
		{
			name:     "conjunction-stops-at-its-first-false-operand",
			bound:    1,
			lifetime: 2,
			correlation: scopedAny(scopedTriggered(), scopedAll(
				correlatedComparison(testpilotspb.CORRELATED_COMPARISON_OPERATOR_EQUAL, scopedFieldOperand(repliedField), scopedLiteralOperand("9")),
				correlatedComparison(testpilotspb.CORRELATED_COMPARISON_OPERATOR_EQUAL, scopedFieldOperand(repliedField), scopedCaptureOperand("seen", 1)))),
			steps:  []scopedStep{{"request", "a", "1"}, {"reply", "a", "1"}},
			reject: "correlation rejected this operation's step",
		},
		{
			name:     "nested-conjunction-admits",
			bound:    1,
			lifetime: 2,
			correlation: scopedAny(scopedTriggered(), scopedAll(
				correlatedComparison(testpilotspb.CORRELATED_COMPARISON_OPERATOR_EQUAL, scopedFieldOperand(repliedField), scopedLiteralOperand("1")),
				correlatedComparison(testpilotspb.CORRELATED_COMPARISON_OPERATOR_NOT_EQUAL, scopedFieldOperand(repliedField), scopedLiteralOperand("2")))),
			steps: []scopedStep{{"request", "a", "1"}, {"reply", "a", "1"}},
			want:  testpilotspb.RULE_VERDICT_STATUS_SATISFIED,
		},
		{
			name: "retained-occurrences-never-move-a-countdown", bound: 0, lifetime: 2, correlation: correlatedCorrelation(0),
			steps: []scopedStep{{"request", "a", "1"}, {"reply", "a", "1"}},
			want:  testpilotspb.RULE_VERDICT_STATUS_VIOLATED,
		},
		{
			name: "lifetime-exhausted", bound: 2, lifetime: 1, correlation: correlatedCorrelation(0),
			steps:  []scopedStep{{"request", "a", "1"}, {"request", "a", "2"}},
			reject: "capture lifetime exhausted",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c, catalog, view, ceiling := scopedCaptureFixture(t, tc.bound, tc.lifetime, tc.correlation)
			status, err := observeCorrelated(t, c, catalog, view, ceiling, tc.steps)
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
	c, catalog, view, ceiling := scopedCaptureFixture(t, 1, 2, correlatedCorrelation(0))
	p, err := Prepare(c, catalog, view, ceiling)
	require.NoError(t, err)
	e, err := p.newEvaluator(context.Background(), view)
	require.NoError(t, err)
	_, err = e.Observe(context.Background(), event(1, 0, testpilotspb.RUN_EVENT_KIND_RUN_OPENED))
	require.NoError(t, err)
	_, err = e.Observe(context.Background(), scopedEvent(t, 2, scopedFieldEvidence(0, "request", "a", "1")))
	require.NoError(t, err)
	before := proto.CloneOf(e.result)
	require.EqualValues(t, 1, e.scoped.capturedValues)
	_, err = e.Observe(context.Background(), scopedEvent(t, 3, scopedFieldEvidence(1, "reply", "a", "2")))
	require.ErrorContains(t, err, "correlation rejected this operation's step")
	// A rejected append publishes no state: the retained occurrence, the semantic transition count
	// and the clause verdict are exactly the ones the admitted prefix established.
	require.EqualValues(t, 1, e.scoped.capturedValues)
	require.EqualValues(t, 1, e.scoped.transitions)
	require.True(t, proto.Equal(before.Rules[0], e.result.Rules[0]))
	occurrences := e.scoped.operations["a"].captures
	require.Len(t, occurrences, 1)
	require.Equal(t, retainedCapture{capture: "seen", ordinal: 0, value: occurrences[0].value}, occurrences[0])
	require.True(t, proto.Equal(scopedText("1"), occurrences[0].value))
}

func TestCorrelatedCaptureCeilingRejects(t *testing.T) {
	c, catalog, view, ceiling := scopedCaptureFixture(t, 2, 4, correlatedCorrelation(0))
	c.Correlated.Limits.MaxCaptures = 1
	_, err := observeCorrelated(t, c, catalog, view, ceiling, []scopedStep{{"request", "a", "1"}, {"request", "a", "2"}})
	require.ErrorContains(t, err, "ceiling")
}

func TestCorrelatedCapturePrepareRejectsUnsupportedDeclarations(t *testing.T) {
	for name, tc := range map[string]struct {
		mutate func(*testpilotspb.CorrelatedContract)
		reason string
	}{
		"unretained-capture-field": {
			mutate: func(s *testpilotspb.CorrelatedContract) { s.Clauses[0].Captures[0].FieldId = "absent" },
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
			mutate: func(s *testpilotspb.CorrelatedContract) { s.Clauses[0].Captures[0].Lifetime = 0 },
			reason: "invalid capture declaration",
		},
		"repeated-capture-id": {
			mutate: func(s *testpilotspb.CorrelatedContract) {
				s.Clauses[0].Captures = append(s.Clauses[0].Captures, proto.CloneOf(s.Clauses[0].Captures[0]))
			},
			reason: "invalid capture declaration",
		},
		"ordinal-beyond-lifetime": {
			mutate: func(s *testpilotspb.CorrelatedContract) { s.Clauses[0].Correlation = correlatedCorrelation(2) },
			reason: "unbound capture reference",
		},
		"unknown-capture": {
			mutate: func(s *testpilotspb.CorrelatedContract) {
				s.Clauses[0].Correlation = correlatedComparison(testpilotspb.CORRELATED_COMPARISON_OPERATOR_EQUAL, scopedFieldOperand(repliedField), scopedCaptureOperand("other", 0))
			},
			reason: "unbound capture reference",
		},
		"unretained-operand": {
			mutate: func(s *testpilotspb.CorrelatedContract) {
				s.Clauses[0].Correlation = correlatedComparison(testpilotspb.CORRELATED_COMPARISON_OPERATOR_EQUAL, scopedFieldOperand("absent"), scopedCaptureOperand("seen", 0))
			},
			reason: "unretained correlation field operand",
		},
		"empty-group": {
			mutate: func(s *testpilotspb.CorrelatedContract) { s.Clauses[0].Correlation = scopedAny() },
			reason: "empty correlation group",
		},
		"unsupported-operator": {
			mutate: func(s *testpilotspb.CorrelatedContract) {
				s.Clauses[0].Correlation = correlatedComparison(testpilotspb.CORRELATED_COMPARISON_OPERATOR_UNSPECIFIED, scopedLiteralOperand("1"), scopedCaptureOperand("seen", 0))
			},
			reason: "unsupported comparison operator",
		},
		"unsupported-literal": {
			mutate: func(s *testpilotspb.CorrelatedContract) {
				s.Clauses[0].Correlation = correlatedComparison(testpilotspb.CORRELATED_COMPARISON_OPERATOR_EQUAL, &testpilotspb.CorrelatedOperand{Operand: &testpilotspb.CorrelatedOperand_Literal{Literal: &testpilotspb.Value{Value: &testpilotspb.Value_Natural{Natural: "01"}}}}, scopedCaptureOperand("seen", 0))
			},
			reason: "unsupported correlation literal",
		},
		"repeated-capture-id-across-clauses": {
			mutate: func(s *testpilotspb.CorrelatedContract) {
				second := proto.CloneOf(s.Clauses[0])
				second.ClauseId = "second"
				s.Clauses = append(s.Clauses, second)
			},
			reason: "invalid capture declaration",
		},
		"capture-of-another-clause": {
			mutate: func(s *testpilotspb.CorrelatedContract) {
				second := proto.CloneOf(s.Clauses[0])
				second.ClauseId = "second"
				second.Captures = nil
				s.Clauses = append(s.Clauses, second)
			},
			reason: "unbound capture reference",
		},
		"mismatched-operand-kinds": {
			mutate: func(s *testpilotspb.CorrelatedContract) {
				s.Clauses[0].Correlation = correlatedComparison(testpilotspb.CORRELATED_COMPARISON_OPERATOR_EQUAL, scopedFieldOperand(repliedField), &testpilotspb.CorrelatedOperand{Operand: &testpilotspb.CorrelatedOperand_Literal{Literal: &testpilotspb.Value{Value: &testpilotspb.Value_Natural{Natural: "1"}}}})
			},
			reason: "incompatible correlation operand types",
		},
		"mismatched-capture-kind": {
			mutate: func(s *testpilotspb.CorrelatedContract) {
				for _, rule := range s.ProjectionRules {
					if rule.Kind == "reply" {
						rule.Fields[0].Type = &testpilotspb.ScalarType{Kind: testpilotspb.SCALAR_KIND_NATURAL}
					}
				}
			},
			reason: "incompatible correlation operand types",
		},
		"ambiguous-retained-field-kind": {
			mutate: func(s *testpilotspb.CorrelatedContract) {
				for _, rule := range s.ProjectionRules {
					if rule.Kind == "reply" {
						rule.Fields = append(rule.Fields, &testpilotspb.CorrelatedFieldPolicy{FieldId: capturedField, Type: &testpilotspb.ScalarType{Kind: testpilotspb.SCALAR_KIND_NATURAL}, Disposition: testpilotspb.CORRELATED_FIELD_DISPOSITION_RETAIN})
					}
				}
			},
			reason: "invalid capture declaration",
		},
		"depth-exhausted": {
			mutate: func(s *testpilotspb.CorrelatedContract) { s.Limits.MaxCorrelationDepth = 1 },
			reason: "correlation depth exhausted",
		},
		"missing-capture-ceiling": {
			mutate: func(s *testpilotspb.CorrelatedContract) { s.Limits.MaxCaptures = 0 },
			reason: "scoped capture limits must be positive",
		},
		"missing-depth-ceiling": {
			mutate: func(s *testpilotspb.CorrelatedContract) { s.Limits.MaxCorrelationDepth = 0 },
			reason: "scoped capture limits must be positive",
		},
	} {
		t.Run(name, func(t *testing.T) {
			c, catalog, view, ceiling := scopedCaptureFixture(t, 1, 2, correlatedCorrelation(0))
			tc.mutate(c.Correlated)
			_, err := Prepare(c, catalog, view, ceiling)
			require.ErrorContains(t, err, tc.reason)
		})
	}
}

// A capability that declares neither captures nor a correlation keeps its exact prior meaning and
// leaves both new ceilings unset.
func TestCorrelatedCapabilityWithoutCapturesIsUnchanged(t *testing.T) {
	c, catalog, view, ceiling := correlatedFixture(t, 1)
	require.Zero(t, c.Correlated.Limits.MaxCaptures)
	require.Zero(t, c.Correlated.Limits.MaxCorrelationDepth)
	require.True(t, slices.ContainsFunc(c.Correlated.Clauses, func(clause *testpilotspb.CorrelatedRule) bool {
		return len(clause.Captures) == 0 && clause.Correlation == nil
	}))
	p, err := Prepare(c, catalog, view, ceiling)
	require.NoError(t, err)
	e, err := p.newEvaluator(context.Background(), view)
	require.NoError(t, err)
	_, err = e.Observe(context.Background(), event(1, 0, testpilotspb.RUN_EVENT_KIND_RUN_OPENED))
	require.NoError(t, err)
	for i, kind := range []string{"request", "reply"} {
		_, err = e.Observe(context.Background(), scopedEvent(t, int64(i+2), correlatedEvidence(int64(i), kind, "a")))
		require.NoError(t, err)
	}
	require.Equal(t, testpilotspb.RULE_VERDICT_STATUS_SATISFIED, e.result.Rules[0].Status)
	require.Zero(t, e.scoped.capturedValues)
}

// Live admission and offline replay agree on a capture-bearing capability at every chunk boundary.
func TestCorrelatedCaptureLiveAndOfflineAgree(t *testing.T) {
	for _, tc := range []struct {
		name  string
		steps []scopedStep
		want  testpilotspb.RuleVerdictStatus
	}{
		{"one-operation", []scopedStep{{"request", "a", "1"}, {"reply", "a", "1"}}, testpilotspb.RULE_VERDICT_STATUS_SATISFIED},
		// The second operation correlates its reply against its own retained occurrence, and the
		// first operation's window is still open, so the clause stays unresolved.
		{"interleaved-operations", []scopedStep{{"request", "a", "1"}, {"request", "b", "2"}, {"reply", "b", "2"}}, testpilotspb.RULE_VERDICT_STATUS_INCONCLUSIVE},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c, catalog, view, ceiling := scopedCaptureFixture(t, 2, 2, correlatedCorrelation(0))
			prepared, err := Prepare(c, catalog, view, ceiling)
			require.NoError(t, err)
			for split := 0; split <= len(tc.steps); split++ {
				monitor, err := prepared.New(context.Background(), view)
				require.NoError(t, err)
				run := &testpilotspb.Run{RunId: "one", CaseId: "correlated.case", ProgramId: "scoped.program", Status: testpilotspb.RUN_STATUS_COMPLETED, Events: []*testpilotspb.RunEvent{event(1, 0, testpilotspb.RUN_EVENT_KIND_RUN_OPENED)}}
				_, err = monitor.Observe(context.Background(), run.Events[0])
				require.NoError(t, err)
				for _, chunk := range [][]scopedStep{tc.steps[:split], tc.steps[split:]} {
					for _, step := range chunk {
						i := slices.Index(tc.steps, step)
						observation := scopedEvent(t, int64(len(run.Events)+1), scopedFieldEvidence(int64(i), step.kind, step.operation, step.value))
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
				offline, err := prepared.Evaluate(context.Background(), run)
				require.NoError(t, err)
				require.True(t, proto.Equal(live, offline))
				require.Equal(t, tc.want, live.Rules[0].Status)
			}
		})
	}
}
