package umpire_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.temporal.io/server/model/go/umpire"
)

// opening counts the door's openings, up to two.
type opening int

func (opening) Values() []opening { return []opening{0, 1, 2} }

func countOpenings(at umpire.Evaluation) *umpire.Monitor {
	return umpire.NewMonitor("openedTwice", opening(0),
		func(n opening, before door, after doorStep) opening {
			if before.Phase != open && after.State.Phase == open {
				return min(n+1, 2)
			}
			return n
		},
		func(n opening) bool { return n == 2 }, at)
}

var four = umpire.Limits{Name: "four", Steps: 4, Actions: 4, Search: 64}

func anything(m *umpire.Machine[door, doorOutcome, doorFact]) *umpire.Query {
	always := m.Property("always").HoldsAcross(func(door, doorStep) bool { return true })
	return m.Scenario("anything").Starts(door{Phase: closed}).Free().Verify("q", always, four)
}

func TestAMonitorKeepsDistinctHistoriesApart(t *testing.T) {
	plain, err := anything(newDoor("door")).Answer()
	require.NoError(t, err)
	require.Equal(t, umpire.VerifiedWithinLimits, plain.Outcome)
	require.Equal(t, 4, plain.Explored, "the start, and closed, open and locked after a step, whatever the path")

	q := anything(newDoor("door")).Watch(countOpenings(umpire.EveryStep()))
	a, err := q.Answer()
	require.NoError(t, err)
	require.Equal(t, umpire.CounterexampleFound, a.Outcome,
		"closing and reopening reaches an explored state again with a different count")
	require.Equal(t, "openedTwice", a.Monitor)
	require.Equal(t, []string{"closed-false-turn-right-true", "open-false-push", "closed-false-turn-right-true"}, a.Rows)
	require.Equal(t, []umpire.MonitorVerdict{{Name: "openedTwice", State: "2", Verdict: umpire.MonitorViolated}}, a.Monitors)
	require.Greater(t, a.Explored, plain.Explored)
	require.NoError(t, q.Replay(a))
}

func TestAMonitorNeverDisablesAStep(t *testing.T) {
	m := newDoor("door")
	opensTwice := m.Property("opensTwice").When(turn.With(right{Strong: true})).
		Holds(func(s doorStep) bool { return s.State.Phase == open })
	path := m.Scenario("twice").Starts(door{Phase: closed}).
		Actions(turn.With(right{Strong: true}), push.With(), turn.With(right{Strong: true}))
	plain, err := path.Find("q", opensTwice, four).Answer()
	require.NoError(t, err)

	q := path.Find("q", opensTwice, four).Watch(countOpenings(umpire.EveryStep()))
	a, err := q.Answer()
	require.NoError(t, err)
	require.Equal(t, umpire.Found, a.Outcome, "the violated monitor takes nothing away from the find")
	require.Equal(t, plain.Rows, a.Rows)
	require.Equal(t, plain.Witness, a.Witness)
	require.Equal(t, []umpire.MonitorVerdict{{Name: "openedTwice", State: "2", Verdict: umpire.MonitorViolated}}, a.Monitors)
	require.NoError(t, q.Replay(a))
}

func TestAMonitorIsReadOnlyAtItsEvaluationPoint(t *testing.T) {
	neverOpened := func(at umpire.Evaluation) *umpire.Monitor {
		return umpire.NewMonitor("lockedUnopened", opening(0),
			func(n opening, before door, after doorStep) opening {
				if after.State.Phase == open {
					return 1
				}
				return n
			},
			func(n opening) bool { return n == 0 }, at)
	}

	a, err := anything(newDoor("door")).Watch(neverOpened(umpire.EveryStep())).Answer()
	require.NoError(t, err)
	require.Equal(t, umpire.CounterexampleFound, a.Outcome)
	require.Equal(t, []string{"closed-false-lock"}, a.Rows)

	a, err = anything(newDoor("door")).Watch(neverOpened(umpire.AtEnds())).Answer()
	require.NoError(t, err)
	require.Equal(t, umpire.CounterexampleFound, a.Outcome, "locked is an end, where the monitor is read")
	require.Equal(t, []string{"closed-false-lock"}, a.Rows)

	afterOpening := umpire.After(func(r umpire.Result) bool { return len(r.Facts) > 0 })
	a, err = anything(newDoor("door")).Watch(neverOpened(afterOpening)).Answer()
	require.NoError(t, err)
	require.Equal(t, umpire.VerifiedWithinLimits, a.Outcome, "every step recording a fact opens the door")
	require.Equal(t, []umpire.MonitorVerdict{{Name: "lockedUnopened", Verdict: umpire.MonitorHeld}}, a.Monitors)

	m := newDoor("door")
	locks := m.Property("locks").When(lock.With()).Holds(func(s doorStep) bool { return s.State.Phase == locked })
	a, err = m.Scenario("lockAtOnce").Starts(door{Phase: closed}).Actions(lock.With()).
		Find("q", locks, four).Watch(neverOpened(afterOpening)).Answer()
	require.NoError(t, err)
	require.Equal(t, umpire.Found, a.Outcome)
	require.Equal(t, []umpire.MonitorVerdict{{Name: "lockedUnopened", State: "0", Verdict: umpire.MonitorUnread}}, a.Monitors)
}

func TestAReplayRejectsAWitnessTheModelDoesNotTake(t *testing.T) {
	q := anything(newDoor("door")).Watch(countOpenings(umpire.EveryStep()))
	a, err := q.Answer()
	require.NoError(t, err)

	forged := a
	forged.Witness = &umpire.Trace{Initial: a.Witness.Initial, Steps: a.Witness.Steps[:2]}
	require.ErrorContains(t, q.Replay(forged), "query q: the witness ends in a state where no claim fails")

	jumped := a
	jumped.Witness = &umpire.Trace{Initial: a.Witness.Initial, Steps: append([]umpire.TraceStep{}, a.Witness.Steps...)}
	jumped.Witness.Steps[1].State = umpire.Atom{ID: "test.door.state.door.locked-false", Value: "locked-false"}
	require.ErrorContains(t, q.Replay(jumped), "step 2 takes push from 'open-false' to 'locked-false', which is no result of that row")

	m := newDoor("door")
	staysShut := m.Property("staysShut").HoldsAcross(func(_ door, after doorStep) bool { return after.State.Phase != open })
	shut := m.Scenario("anything").Starts(door{Phase: closed}).Free().Verify("shut", staysShut, four)
	a, err = shut.Answer()
	require.NoError(t, err)
	require.Equal(t, umpire.CounterexampleFound, a.Outcome)
	require.NoError(t, shut.Replay(a))
	a.Witness = &umpire.Trace{Initial: a.Witness.Initial}
	require.EqualError(t, shut.Replay(a), "query shut: the witness ends in a state where no claim fails")
}

func TestAQueryRejectsTwoMonitorsOfOneName(t *testing.T) {
	q := anything(newDoor("door")).Watch(countOpenings(umpire.EveryStep()), countOpenings(umpire.AtEnds()))
	_, err := q.Answer()
	require.EqualError(t, err, "query q: two monitors are named openedTwice")
}

func TestAMonitorOfAnInfiniteStateTypeIsRejected(t *testing.T) {
	endless := umpire.NewMonitor("endless", 0,
		func(n int, _ door, _ doorStep) int { return n + 1 }, func(int) bool { return false }, umpire.EveryStep())
	_, err := anything(newDoor("door")).Watch(endless).Answer()
	require.ErrorContains(t, err, "monitor endless: state type: int is not finite")
}
