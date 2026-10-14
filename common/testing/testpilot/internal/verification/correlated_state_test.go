package verification

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"google.golang.org/protobuf/proto"
)

// withStateFields supplies the complete state that transitions and results reference.
func withStateFields(c *testpilotspb.Contract, fields ...*testpilotspb.ModelValue) {
	s := c.Correlated
	s.States[0].Fields = fields
}

// correlatedRun admits kinds for operation "a" in order and returns the evaluator and the first
// admission error, with the index of the event that returned it.
func correlatedRun(t *testing.T, c *testpilotspb.Contract, kinds ...string) (*Evaluator, int, error) {
	t.Helper()
	_, catalog, view, ceiling, correlated := correlatedFixture(t, 1)
	p, err := Prepare(c, catalog, view, ceiling, correlated)
	require.NoError(t, err)
	e, err := p.newEvaluator(context.Background(), view)
	require.NoError(t, err)
	_, err = e.Observe(context.Background(), &testpilotspb.RunEvent{Sequence: 1, Kind: testpilotspb.RUN_EVENT_KIND_RUN_OPENED})
	require.NoError(t, err)
	for i, kind := range kinds {
		if _, err := e.Observe(context.Background(), correlatedEvent(t, int64(i+2), correlatedEvidence(int64(i), kind, "a"))); err != nil {
			return e, i, err
		}
	}
	return e, -1, nil
}

func TestCorrelatedMonitorMatchesStatesOnAtomAndFields(t *testing.T) {
	ready := &testpilotspb.ModelValue{DefinitionId: "phase", Value: "ready"}
	other := &testpilotspb.ModelValue{DefinitionId: "phase", Value: "other"}

	t.Run("prior-fields-disagree-with-reached-fields", func(t *testing.T) {
		c, _, _, _, _ := correlatedFixture(t, 1)
		withStateFields(c, ready)
		c.Correlated.States = append(c.Correlated.States, &testpilotspb.CorrelatedState{StateId: "other", Atom: proto.CloneOf(c.Correlated.States[0].Atom), Fields: []*testpilotspb.ModelValue{other}})
		// "request" reaches the ready fields; "tick" is declared only from the other fields, so its
		// atom matches the reached state but its prior does not.
		for _, tr := range c.Correlated.Transitions {
			if tr.ResultId == "tick" {
				tr.PriorStateId = "other"
			}
		}
		e, at, err := correlatedRun(t, c, "request", "tick")
		require.ErrorContains(t, err, "unauthorized operation transition")
		require.Equal(t, 1, at)
		require.EqualValues(t, 1, e.correlated.transitions)
		require.Equal(t, testpilotspb.RULE_VERDICT_STATUS_INCONCLUSIVE, e.result.Rules[0].Status)
	})

	t.Run("initial-fields-disagree-with-first-prior", func(t *testing.T) {
		c, _, _, _, _ := correlatedFixture(t, 1)
		withStateFields(c, ready)
		c.Correlated.States = append(c.Correlated.States, &testpilotspb.CorrelatedState{StateId: "other", Atom: proto.CloneOf(c.Correlated.States[0].Atom), Fields: []*testpilotspb.ModelValue{other}})
		c.Correlated.InitialStateId = "other"
		e, at, err := correlatedRun(t, c, "request")
		require.ErrorContains(t, err, "unauthorized operation transition")
		require.Equal(t, 0, at)
		require.Zero(t, e.correlated.transitions)
	})

	t.Run("consistent-fields-keep-outcome-and-work", func(t *testing.T) {
		kinds := []string{"request", "reply"}
		plain, _, _, _, _ := correlatedFixture(t, 1)
		want, at, err := correlatedRun(t, plain, kinds...)
		require.NoError(t, err, "event %d", at)

		c, _, _, _, _ := correlatedFixture(t, 1)
		withStateFields(c, ready)
		c.Correlated.States = append(c.Correlated.States, &testpilotspb.CorrelatedState{StateId: "other", Atom: proto.CloneOf(c.Correlated.States[0].Atom), Fields: []*testpilotspb.ModelValue{other}})
		// A row declared from other fields shares the request action, so only the field match keeps
		// it out of the candidate count that obligation work charges.
		for _, tr := range c.Correlated.Transitions {
			if tr.ResultId == "request" {
				unreachable := proto.CloneOf(tr)
				unreachable.PriorStateId = "other"
				c.Correlated.Transitions = append(c.Correlated.Transitions, unreachable)
				break
			}
		}
		got, at, err := correlatedRun(t, c, kinds...)
		require.NoError(t, err, "event %d", at)
		require.Equal(t, testpilotspb.RULE_VERDICT_STATUS_SATISFIED, got.result.Rules[0].Status)
		require.Equal(t, want.result.Rules[0].Status, got.result.Rules[0].Status)
		require.Equal(t,
			[]int64{want.correlated.transitions, want.correlated.obligations, want.correlated.obligationWork},
			[]int64{got.correlated.transitions, got.correlated.obligations, got.correlated.obligationWork})
	})
}
