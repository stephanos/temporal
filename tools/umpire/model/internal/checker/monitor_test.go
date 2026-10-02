package checker_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	umpire "go.temporal.io/server/tools/umpire/model/internal/checker"
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

func TestAMonitorWithoutANameIsRejected(t *testing.T) {
	unnamed := countOpenings(umpire.EveryStep())
	unnamed.Name = ""
	_, err := anything(newDoor("door")).Watch(unnamed).Answer()
	require.EqualError(t, err, "query q: a monitor has no name")
}

// A machine whose two first steps reach one state by different outcomes.

type fork string

func (fork) Values() []fork { return []fork{"s0", "s1", "s2"} }

type forkOutcome string

func (forkOutcome) Values() []forkOutcome { return []forkOutcome{"l", "r", "ok"} }

type forkStep = umpire.Step[fork, forkOutcome, keyFact]

var (
	goLeft  = umpire.NewAction0("left", "person")
	goRight = umpire.NewAction0("right", "person")
	goOn    = umpire.NewAction0("go", "person")
)

func forked() *umpire.Machine[fork, forkOutcome, keyFact] {
	first := func(o forkOutcome) func(fork) []forkStep {
		return func(s fork) []forkStep {
			if s != "s0" {
				return nil
			}
			return []forkStep{{Outcome: o, State: "s1"}}
		}
	}
	return umpire.NewMachine[fork, forkOutcome, keyFact]("test.fork", "fork").
		Starts("s0").
		Ends(func(s fork) bool { return s == "s2" }).
		Step0(goLeft, first("l")).Step0(goRight, first("r")).
		Step0(goOn, func(s fork) []forkStep {
			if s != "s1" {
				return nil
			}
			return []forkStep{{Outcome: "ok", State: "s2"}}
		})
}

// spells is a Monitor whose state is set, from its initial state, by the first step's outcome.
func spells(name string, byOutcome map[string]string, violated string) *umpire.Monitor {
	return &umpire.Monitor{Name: name, Initial: "i",
		Next: func(key string, _ any, step umpire.Result) (string, error) {
			if next, ok := byOutcome[step.Outcome]; ok && key == "i" {
				return next, nil
			}
			return key, nil
		},
		Violated: func(key string) bool { return key == violated },
		At:       umpire.After(func(r umpire.Result) bool { return r.State == "s2" }),
	}
}

func TestMonitorStatesThatSpellAlikeStayApart(t *testing.T) {
	m := forked()
	always := m.Property("always").HoldsAcross(func(fork, forkStep) bool { return true })
	q := m.Scenario("anything").Starts("s0").Free().Verify("q", always, four).Watch(
		spells("first", map[string]string{"l": "x/false/false\x00y", "r": "x"}, ""),
		spells("second", map[string]string{"l": "z", "r": "y/false/false\x00z"}, "y/false/false\x00z"))
	a, err := q.Answer()
	require.NoError(t, err)
	require.Equal(t, umpire.CounterexampleFound, a.Outcome, "the right history violates the second monitor")
	require.Equal(t, []string{"s0-right", "s1-go"}, a.Rows)
	require.Equal(t, "second", a.Monitor)
	require.NoError(t, q.Replay(a))
}

func TestACounterexampleFoundBeforeTheLimitStands(t *testing.T) {
	cramped := umpire.Limits{Name: "cramped", Steps: 4, Actions: 4, Search: 6}
	m := newDoor("door")
	always := m.Property("always").HoldsAcross(func(door, doorStep) bool { return true })
	q := m.Scenario("anything").Starts(door{Phase: closed}).Free().Verify("q", always, cramped).
		Watch(countOpenings(umpire.EveryStep()))
	a, err := q.Answer()
	require.NoError(t, err)
	require.Equal(t, umpire.CounterexampleFound, a.Outcome, "the monitor failed at the sixth state, within the limit")
	require.Equal(t, "openedTwice", a.Monitor)
	require.Equal(t, []string{"closed-false-turn-right-true", "open-false-push", "closed-false-turn-right-true"}, a.Rows)
	require.Equal(t, cramped.Search, a.Explored, "the search stops at its limit, the seventh state unvisited")
	require.NoError(t, q.Replay(a))
}

func TestAViolationOnlyBeyondTheLimitIsLimitReached(t *testing.T) {
	short := umpire.Limits{Name: "short", Steps: 4, Actions: 4, Search: 5}
	m := newDoor("door")
	always := m.Property("always").HoldsAcross(func(door, doorStep) bool { return true })
	a, err := m.Scenario("anything").Starts(door{Phase: closed}).Free().Verify("q", always, short).
		Watch(countOpenings(umpire.EveryStep())).Answer()
	require.NoError(t, err)
	require.Equal(t, umpire.LimitReached, a.Outcome, "the monitor fails only at the sixth state, past the limit")
	require.Empty(t, a.Monitor)
	require.Nil(t, a.Witness)
	require.Equal(t, short.Search, a.Explored)
}

func TestTheLimitBoundsEveryStateAPropertySearchVisits(t *testing.T) {
	m := newDoor("door")
	staysShut := m.Property("staysShut").HoldsAcross(func(_ door, after doorStep) bool { return after.State.Phase != open })
	free := m.Scenario("anything").Starts(door{Phase: closed}).Free()
	within := umpire.Limits{Name: "within", Steps: 2, Actions: 2, Search: 3}
	a, err := free.Verify("q", staysShut, within).Answer()
	require.NoError(t, err)
	require.Equal(t, umpire.CounterexampleFound, a.Outcome, "the door opens at the third state")
	require.Equal(t, []string{"closed-false-turn-right-true"}, a.Rows)

	beyond := umpire.Limits{Name: "beyond", Steps: 2, Actions: 2, Search: 2}
	a, err = free.Verify("q", staysShut, beyond).Answer()
	require.NoError(t, err)
	require.Equal(t, umpire.LimitReached, a.Outcome)
	require.Equal(t, beyond.Search, a.Explored)

	opensLoudly := m.Property("opensLoudly").When(turn.With(right{Strong: true})).
		Holds(func(s doorStep) bool { return s.State.Phase == open })
	path := m.Scenario("turnThenPush").Starts(door{Phase: closed}).Actions(turn.With(right{Strong: true}), push.With())
	a, err = path.Find("q", opensLoudly, umpire.Limits{Name: "three", Steps: 2, Actions: 2, Search: 3}).Answer()
	require.NoError(t, err)
	require.Equal(t, umpire.Found, a.Outcome, "the pinned trace completes at the third state")
	a, err = path.Find("q", opensLoudly, beyond).Answer()
	require.NoError(t, err)
	require.Equal(t, umpire.LimitReached, a.Outcome, "a trace completing past the limit is not searched")
	require.Equal(t, beyond.Search, a.Explored)
}

func TestAReplayChecksEveryDefinitionID(t *testing.T) {
	m := newDoor("door")
	staysShut := m.Property("staysShut").HoldsAcross(func(_ door, after doorStep) bool { return after.State.Phase != open })
	q := m.Scenario("anything").Starts(door{Phase: closed}).Free().Verify("shut", staysShut, four)
	a, err := q.Answer()
	require.NoError(t, err)
	tb := tableOf(t, m)
	require.NoError(t, tb.Replay(a.Witness))
	require.Len(t, a.Witness.Steps[0].Facts, 2)

	foreign := func(atom *umpire.Atom) { atom.ID = "test.other" + atom.ID[len("test.door"):] }
	for name, rebind := range map[string]func(w *umpire.Trace){
		"initial": func(w *umpire.Trace) { foreign(&w.Initial) },
		"action":  func(w *umpire.Trace) { foreign(&w.Steps[0].Action) },
		"outcome": func(w *umpire.Trace) { foreign(&w.Steps[0].Outcome) },
		"state":   func(w *umpire.Trace) { foreign(&w.Steps[0].State) },
		"fact":    func(w *umpire.Trace) { foreign(&w.Steps[0].Facts[1]) },
		"kind":    func(w *umpire.Trace) { w.Steps[0].State.ID = "test.door.fact.door." + w.Steps[0].State.Value },
	} {
		w := &umpire.Trace{Initial: a.Witness.Initial}
		for _, s := range a.Witness.Steps {
			s.Facts = append([]umpire.Atom{}, s.Facts...)
			w.Steps = append(w.Steps, s)
		}
		rebind(w)
		require.ErrorContains(t, tb.Replay(w), "is not the Definition ID", name)
		forged := a
		forged.Witness = w
		require.ErrorContains(t, q.Replay(forged), "is not the Definition ID", name)
	}
}
