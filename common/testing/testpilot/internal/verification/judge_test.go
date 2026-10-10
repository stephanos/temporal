package verification

import (
	"context"
	"strconv"
	"testing"

	celpb "cel.dev/expr"
	"github.com/stretchr/testify/require"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	cel "go.temporal.io/server/common/testing/testpilot/cel"
	"go.temporal.io/server/common/testing/testpilot/internal/execution"
	"go.temporal.io/server/common/testing/testpilot/internal/ir"
	"google.golang.org/protobuf/proto"
)

// The judge's generic rules, as the Testpilot README's "How a Run is judged" states them. Verdict
// aggregation is TestConclude in internal/execution, and disposition precedence is
// TestRunDispositionPrecedence there.

func integer(value string) *testpilotspb.Expression {
	number, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		panic(err)
	}
	return cel.Literal(&celpb.Value{Kind: &celpb.Value_Int64Value{Int64Value: number}})
}

// A rule nothing in the Run resolved is inconclusive on a completed Run, never satisfied: a plain
// rule still pending at closure, and a correlated rule that admitted no evidence, though a Model
// trace reads an empty obligation list as vacuously satisfied.
func TestSilenceIsInconclusive(t *testing.T) {
	closedRun := func(programID string) *testpilotspb.Run {
		return &testpilotspb.Run{RunId: "run", ProgramId: programID, Disposition: testpilotspb.RUN_DISPOSITION_COMPLETED, Events: []*testpilotspb.RunEvent{
			event(1, 0, testpilotspb.RUN_EVENT_KIND_RUN_OPENED),
			event(2, 10, testpilotspb.RUN_EVENT_KIND_RUN_CLOSED),
		}}
	}
	t.Run("plain rule", func(t *testing.T) {
		c, cat, view, limits := fixture(t)
		p, err := Prepare(c, cat, view, limits, nil)
		require.NoError(t, err)
		verdict, _, err := p.Evaluate(context.Background(), closedRun("program"))
		require.NoError(t, err)
		require.Equal(t, testpilotspb.RULE_VERDICT_STATUS_INCONCLUSIVE, verdict.Rules[0].Status)
		require.Equal(t, testpilotspb.VERDICT_STATUS_INCONCLUSIVE, verdict.Status)
	})
	t.Run("correlated rule", func(t *testing.T) {
		c, catalog, view, ceiling, correlated := correlatedFixture(t, 1)
		p, err := Prepare(c, catalog, view, ceiling, correlated)
		require.NoError(t, err)
		verdict, _, err := p.Evaluate(context.Background(), closedRun("correlated.program"))
		require.NoError(t, err)
		require.Equal(t, testpilotspb.RULE_VERDICT_STATUS_INCONCLUSIVE, verdict.Rules[0].Status)
		require.Equal(t, testpilotspb.VERDICT_STATUS_INCONCLUSIVE, verdict.Status)
	})
}

// The first event that violates a rule freezes the evaluation: the Monitor stops, and no later
// event violates, satisfies or supports anything, so the Verdict is the proved bad prefix.
func TestEvaluationFreezesAtTheFirstViolation(t *testing.T) {
	c, cat, view, limits := fixture(t)
	first := c.Rules[0]
	first.Transitions = []*testpilotspb.ContractTransition{transition("one", "start", "bad", equal(observation("id"), integer("1")))}
	later := proto.CloneOf(first)
	later.RuleId = "later"
	later.Transitions = []*testpilotspb.ContractTransition{transition("two", "start", "bad", equal(observation("id"), integer("2")))}
	satisfiable := proto.CloneOf(first)
	satisfiable.RuleId = "satisfiable"
	satisfiable.Transitions = []*testpilotspb.ContractTransition{transition("two", "start", "good", equal(observation("id"), integer("2")))}
	c.Rules = append(c.Rules, later, satisfiable)
	p, err := Prepare(c, cat, view, limits, nil)
	require.NoError(t, err)
	run := &testpilotspb.Run{RunId: "run", ProgramId: "program", Disposition: testpilotspb.RUN_DISPOSITION_STOPPED_BY_MONITOR, Events: []*testpilotspb.RunEvent{
		event(1, 0, testpilotspb.RUN_EVENT_KIND_RUN_OPENED),
		observed(2, 10, 1),
		observed(3, 20, 2),
		event(4, 30, testpilotspb.RUN_EVENT_KIND_RUN_CLOSED),
	}}
	monitor, err := execution.NewMonitor(context.Background(), p, view)
	require.NoError(t, err)
	var decisions []execution.Decision
	for _, e := range run.Events {
		decision, err := monitor.Observe(context.Background(), e)
		require.NoError(t, err)
		decisions = append(decisions, decision)
	}
	require.Equal(t, []execution.Decision{execution.Continue, execution.Stop, execution.Stop, execution.Stop}, decisions)
	live, err := monitor.Close(context.Background(), run)
	require.NoError(t, err)
	verdict, violations, err := p.Evaluate(context.Background(), run)
	require.NoError(t, err)
	require.True(t, proto.Equal(live, verdict))
	require.Equal(t, testpilotspb.VERDICT_STATUS_VIOLATED, verdict.Status)
	require.Equal(t, []testpilotspb.RuleVerdictStatus{testpilotspb.RULE_VERDICT_STATUS_VIOLATED, testpilotspb.RULE_VERDICT_STATUS_INCONCLUSIVE, testpilotspb.RULE_VERDICT_STATUS_INCONCLUSIVE},
		[]testpilotspb.RuleVerdictStatus{verdict.Rules[0].Status, verdict.Rules[1].Status, verdict.Rules[2].Status})
	require.Equal(t, []int64{2}, verdict.SupportingEventSequences)
	require.Equal(t, []Violation{{RuleID: "rule", Sequence: 2, ObservationIDs: []string{"id"}}}, violations)
}

// Correlated evidence is one piece per identity: the same evidence recorded again is the piece
// already admitted, adding no transition and no support, and different evidence under an admitted
// identity is malformed.
func TestCorrelatedEvidenceIsDeduplicatedByIdentity(t *testing.T) {
	c, catalog, view, ceiling, correlated := correlatedFixture(t, 1)
	p, err := Prepare(c, catalog, view, ceiling, correlated)
	require.NoError(t, err)
	opened := func(t *testing.T) *Evaluator {
		e, err := p.newEvaluator(context.Background(), view)
		require.NoError(t, err)
		_, err = e.Observe(context.Background(), event(1, 0, testpilotspb.RUN_EVENT_KIND_RUN_OPENED))
		require.NoError(t, err)
		_, err = e.Observe(context.Background(), correlatedEvent(t, 2, correlatedEvidence(0, "request", "a")))
		require.NoError(t, err)
		return e
	}
	t.Run("the same evidence again", func(t *testing.T) {
		e := opened(t)
		_, err := e.Observe(context.Background(), correlatedEvent(t, 3, correlatedEvidence(0, "request", "a")))
		require.NoError(t, err)
		require.Len(t, e.correlated.accepted, 1)
		require.EqualValues(t, 1, e.correlated.transitions)
		require.Equal(t, []int64{2}, e.correlated.accepted[0].supportingEventSequences)
	})
	t.Run("other evidence under the same identity", func(t *testing.T) {
		e := opened(t)
		_, err := e.Observe(context.Background(), correlatedEvent(t, 3, correlatedEvidence(0, "reply", "a")))
		require.Equal(t, &ir.Error{Category: ir.Malformed, Path: "contract", Detail: "conflicting source identity"}, err)
	})
}
