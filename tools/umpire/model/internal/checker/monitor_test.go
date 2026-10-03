package checker_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	umpire "go.temporal.io/server/tools/umpire/model/internal/checker"
)

// countOpenings counts the door's openings, up to two: its states are "0", "1" and "2".
func countOpenings(at umpire.Evaluation) *umpire.Monitor { return countOpeningKeys(at) }

var four = umpire.Limits{Name: "four", Steps: 4, Actions: 4, Search: 64}

func anything(m *umpire.Table) *umpire.Query {
	holds := umpire.KeyTransitionProperty(m, "always", always)
	return umpire.KeyVerify("q", holds, umpire.KeyFreeScenario(m, "anything", "closed-false"), four)
}

func TestAMonitorKeepsDistinctHistoriesApart(t *testing.T) {
	plain, err := anything(doorTable("door")).Answer()
	require.NoError(t, err)
	require.Equal(t, umpire.VerifiedWithinLimits, plain.Outcome)
	require.Equal(t, 4, plain.Explored, "the start, and closed, open and locked after a step, whatever the path")

	q := anything(doorTable("door")).Watch(countOpenings(umpire.EveryStep()))
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
	m := doorTable("door")
	opensTwice := umpire.KeyProperty(m, "opensTwice", on("turn-right-true"), "turn-right-true",
		func(s umpire.Result) (bool, error) { return phaseOf(s.State) == "open", nil })
	path := umpire.KeyScenario(m, "twice", "closed-false", "turn-right-true", "push", "turn-right-true")
	plain, err := umpire.KeyFind("q", opensTwice, path, four).Answer()
	require.NoError(t, err)

	q := umpire.KeyFind("q", opensTwice, path, four).Watch(countOpenings(umpire.EveryStep()))
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
		return umpire.KeyMonitor("lockedUnopened", "0",
			func(n, _ string, after umpire.Result) (string, error) {
				if phaseOf(after.State) == "open" {
					return "1", nil
				}
				return n, nil
			},
			func(n string) (bool, error) { return n == "0", nil }, at)
	}

	a, err := anything(doorTable("door")).Watch(neverOpened(umpire.EveryStep())).Answer()
	require.NoError(t, err)
	require.Equal(t, umpire.CounterexampleFound, a.Outcome)
	require.Equal(t, []string{"closed-false-lock"}, a.Rows)

	a, err = anything(doorTable("door")).Watch(neverOpened(umpire.AtEnds())).Answer()
	require.NoError(t, err)
	require.Equal(t, umpire.CounterexampleFound, a.Outcome, "locked is an end, where the monitor is read")
	require.Equal(t, []string{"closed-false-lock"}, a.Rows)

	afterOpening := umpire.AfterKey(func(r umpire.Result) (bool, error) { return len(r.Facts) > 0, nil })
	a, err = anything(doorTable("door")).Watch(neverOpened(afterOpening)).Answer()
	require.NoError(t, err)
	require.Equal(t, umpire.VerifiedWithinLimits, a.Outcome, "every step recording a fact opens the door")
	require.Equal(t, []umpire.MonitorVerdict{{Name: "lockedUnopened", Verdict: umpire.MonitorHeld}}, a.Monitors)

	m := doorTable("door")
	locks := umpire.KeyProperty(m, "locks", on("lock"), "lock",
		func(s umpire.Result) (bool, error) { return phaseOf(s.State) == "locked", nil })
	a, err = umpire.KeyFind("q", locks, umpire.KeyScenario(m, "lockAtOnce", "closed-false", "lock"), four).
		Watch(neverOpened(afterOpening)).Answer()
	require.NoError(t, err)
	require.Equal(t, umpire.Found, a.Outcome)
	require.Equal(t, []umpire.MonitorVerdict{{Name: "lockedUnopened", State: "0", Verdict: umpire.MonitorUnread}}, a.Monitors)
}

func TestAReplayRejectsAWitnessTheModelDoesNotTake(t *testing.T) {
	q := anything(doorTable("door")).Watch(countOpenings(umpire.EveryStep()))
	a, err := q.Answer()
	require.NoError(t, err)

	forged := a
	forged.Witness = &umpire.Trace{Initial: a.Witness.Initial, Steps: a.Witness.Steps[:2]}
	require.ErrorContains(t, q.Replay(forged), "query q: the witness ends in a state where no claim fails")

	jumped := a
	jumped.Witness = &umpire.Trace{Initial: a.Witness.Initial, Steps: append([]umpire.TraceStep{}, a.Witness.Steps...)}
	jumped.Witness.Steps[1].State = umpire.Atom{ID: "test.door.state.door.locked-false", Value: "locked-false"}
	require.ErrorContains(t, q.Replay(jumped), "step 2 takes push from 'open-false' to 'locked-false', which is no result of that row")

	m := doorTable("door")
	shut := umpire.KeyVerify("shut", staysShutOn(m), umpire.KeyFreeScenario(m, "anything", "closed-false"), four)
	a, err = shut.Answer()
	require.NoError(t, err)
	require.Equal(t, umpire.CounterexampleFound, a.Outcome)
	require.NoError(t, shut.Replay(a))
	a.Witness = &umpire.Trace{Initial: a.Witness.Initial}
	require.EqualError(t, shut.Replay(a), "query shut: the witness ends in a state where no claim fails")
}

func TestAQueryRejectsTwoMonitorsOfOneName(t *testing.T) {
	q := anything(doorTable("door")).Watch(countOpenings(umpire.EveryStep()), countOpenings(umpire.AtEnds()))
	_, err := q.Answer()
	require.EqualError(t, err, "query q: two monitors are named openedTwice")
}

// A key-level Monitor's state is a key, with no type to enumerate: what is checked of its declaration
// is that it names the functions that move and read that key.
func TestAMonitorWithoutItsNextFunctionIsRejected(t *testing.T) {
	endless := umpire.KeyMonitor("endless", "0", nil, func(string) (bool, error) { return false, nil }, umpire.EveryStep())
	_, err := anything(doorTable("door")).Watch(endless).Answer()
	require.EqualError(t, err, "query q: monitor endless: a Monitor names its next and violated functions")
}

func TestAMonitorWithoutANameIsRejected(t *testing.T) {
	unnamed := countOpenings(umpire.EveryStep())
	unnamed.Name = ""
	_, err := anything(doorTable("door")).Watch(unnamed).Answer()
	require.EqualError(t, err, "query q: a monitor has no name")
}

// A machine whose two first steps reach one state by different outcomes: forkedTable of
// keyclaims_test.go.

// spells is a Monitor whose state is set, from its initial state, by the first step's outcome.
func spells(name string, byOutcome map[string]string, violated string) *umpire.Monitor {
	return umpire.KeyMonitor(name, "i",
		func(key, _ string, step umpire.Result) (string, error) {
			if next, ok := byOutcome[step.Outcome]; ok && key == "i" {
				return next, nil
			}
			return key, nil
		},
		func(key string) (bool, error) { return key == violated, nil },
		umpire.AfterKey(func(r umpire.Result) (bool, error) { return r.State == "s2", nil }))
}

func TestMonitorStatesThatSpellAlikeStayApart(t *testing.T) {
	m := forkedTable()
	holds := umpire.KeyTransitionProperty(m, "always", always)
	q := umpire.KeyVerify("q", holds, umpire.KeyFreeScenario(m, "anything", "s0"), four).Watch(
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
	m := doorTable("door")
	holds := umpire.KeyTransitionProperty(m, "always", always)
	q := umpire.KeyVerify("q", holds, umpire.KeyFreeScenario(m, "anything", "closed-false"), cramped).
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
	m := doorTable("door")
	holds := umpire.KeyTransitionProperty(m, "always", always)
	a, err := umpire.KeyVerify("q", holds, umpire.KeyFreeScenario(m, "anything", "closed-false"), short).
		Watch(countOpenings(umpire.EveryStep())).Answer()
	require.NoError(t, err)
	require.Equal(t, umpire.LimitReached, a.Outcome, "the monitor fails only at the sixth state, past the limit")
	require.Empty(t, a.Monitor)
	require.Nil(t, a.Witness)
	require.Equal(t, short.Search, a.Explored)
}

func TestTheLimitBoundsEveryStateAPropertySearchVisits(t *testing.T) {
	m := doorTable("door")
	staysShut := staysShutOn(m)
	free := umpire.KeyFreeScenario(m, "anything", "closed-false")
	within := umpire.Limits{Name: "within", Steps: 2, Actions: 2, Search: 3}
	a, err := umpire.KeyVerify("q", staysShut, free, within).Answer()
	require.NoError(t, err)
	require.Equal(t, umpire.CounterexampleFound, a.Outcome, "the door opens at the third state")
	require.Equal(t, []string{"closed-false-turn-right-true"}, a.Rows)

	beyond := umpire.Limits{Name: "beyond", Steps: 2, Actions: 2, Search: 2}
	a, err = umpire.KeyVerify("q", staysShut, free, beyond).Answer()
	require.NoError(t, err)
	require.Equal(t, umpire.LimitReached, a.Outcome)
	require.Equal(t, beyond.Search, a.Explored)

	opensLoudly := opensLoudlyOn(m)
	path := umpire.KeyScenario(m, "turnThenPush", "closed-false", "turn-right-true", "push")
	a, err = umpire.KeyFind("q", opensLoudly, path, umpire.Limits{Name: "three", Steps: 2, Actions: 2, Search: 3}).Answer()
	require.NoError(t, err)
	require.Equal(t, umpire.Found, a.Outcome, "the pinned trace completes at the third state")
	a, err = umpire.KeyFind("q", opensLoudly, path, beyond).Answer()
	require.NoError(t, err)
	require.Equal(t, umpire.LimitReached, a.Outcome, "a trace completing past the limit is not searched")
	require.Equal(t, beyond.Search, a.Explored)
}

func TestAReplayChecksEveryDefinitionID(t *testing.T) {
	tb := doorTable("door")
	q := umpire.KeyVerify("shut", staysShutOn(tb), umpire.KeyFreeScenario(tb, "anything", "closed-false"), four)
	a, err := q.Answer()
	require.NoError(t, err)
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
