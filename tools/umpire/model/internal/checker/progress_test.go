package checker_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	umpire "go.temporal.io/server/tools/umpire/model/internal/checker"
)

// keyTable builds a key-only table from its starts and its edges, each a source, an action and a
// target, one row per source and action.
func keyTable(name string, starts []string, edges ...[3]string) *umpire.Table {
	spec := umpire.TableSpec{Machine: name, Family: "test.progress", Outcomes: []string{"ok"}, Starts: starts}
	seen := map[string]bool{}
	add := func(s string) {
		if !seen[s] {
			seen[s] = true
			spec.States = append(spec.States, s)
		}
	}
	actions := map[string]bool{}
	for _, s := range starts {
		add(s)
	}
	for _, e := range edges {
		add(e[0])
		add(e[2])
		if !actions[e[1]] {
			actions[e[1]] = true
			spec.Actions = append(spec.Actions, e[1])
		}
		res := umpire.Result{Outcome: "ok", State: e[2], Facts: []string{}}
		if n := len(spec.Rows); n > 0 && spec.Rows[n-1].Source == e[0] && spec.Rows[n-1].Action == e[1] {
			spec.Rows[n-1].Results = append(spec.Rows[n-1].Results, res)
			continue
		}
		spec.Rows = append(spec.Rows, umpire.Row{Key: e[0] + "-" + e[1], Source: e[0], Action: e[1],
			Results: []umpire.Result{res}})
	}
	return umpire.NewTable(spec)
}

func is(key string) func(string) bool { return func(s string) bool { return s == key } }

var wide = umpire.Limits{Name: "wide", Steps: 8, Search: 256}

func requireVerdict(t *testing.T, want umpire.Outcome, v umpire.ProgressVerdict) {
	t.Helper()
	require.Equal(t, want, v.Outcome, v.Explanation)
}

func TestProgressFindsADeadlock(t *testing.T) {
	tb := keyTable("stalls", []string{"waiting"},
		[3]string{"waiting", "fail", "stuck"}, [3]string{"waiting", "finish", "done"})
	p := umpire.KeyProgress("finishes", is("waiting"), is("done"), 3)
	a, err := umpire.CheckProgress(tb, p, wide)
	require.NoError(t, err)
	requireVerdict(t, umpire.CounterexampleFound, a.Deadlock)
	require.Equal(t, []string{"fail"}, actionsOf(a.Deadlock.Witness))
	require.NoError(t, p.Replay(tb, umpire.DeadlockKind, a.Deadlock))
	requireVerdict(t, umpire.VerifiedWithinLimits, a.Cycle)
	requireVerdict(t, umpire.VerifiedWithinLimits, a.Deadline)
	require.Equal(t, 1, a.From)

	cut := a.Deadlock
	cut.Witness = &umpire.Trace{Initial: cut.Witness.Initial}
	require.ErrorContains(t, p.Replay(tb, umpire.DeadlockKind, cut), "finishes: the witness ends at 'waiting', which has a step")
}

func TestProgressTellsAFairCycleFromAnUnfairOne(t *testing.T) {
	tb := keyTable("polls", []string{"waiting"},
		[3]string{"waiting", "poll", "waiting"}, [3]string{"waiting", "finish", "done"})

	unfair := umpire.KeyProgress("finishes", is("waiting"), is("done"), 2)
	a, err := umpire.CheckProgress(tb, unfair, wide)
	require.NoError(t, err)
	requireVerdict(t, umpire.CounterexampleFound, a.Cycle)
	require.Equal(t, []string{"poll"}, actionsOf(a.Cycle.Witness))
	require.Equal(t, 0, a.Cycle.Loop)
	require.NoError(t, unfair.Replay(tb, umpire.CycleKind, a.Cycle))
	require.Empty(t, a.Assumptions)

	finishes := umpire.Assumption{Name: "finishIsFair", Fair: []string{"finish"}}
	fair := umpire.KeyProgress("finishes", is("waiting"), is("done"), 2, finishes)
	a, err = umpire.CheckProgress(tb, fair, wide)
	require.NoError(t, err)
	requireVerdict(t, umpire.VerifiedWithinLimits, a.Cycle)
	requireVerdict(t, umpire.VerifiedWithinLimits, a.Deadlock)
	require.Equal(t, []string{"finishIsFair"}, a.Assumptions)
	requireVerdict(t, umpire.CounterexampleFound, a.Deadline)
	require.Equal(t, []string{"poll", "poll"}, actionsOf(a.Deadline.Witness), "fairness bounds no finite path")
	require.NoError(t, fair.Replay(tb, umpire.DeadlineKind, a.Deadline))

	unfairCycle := a.Deadline
	unfairCycle.Loop = 1
	require.ErrorContains(t, fair.Replay(tb, umpire.CycleKind, unfairCycle),
		"finishes: finish stays enabled on the cycle and is never taken, which finishIsFair forbids")
}

func TestWeakFairnessAdmitsACycleThatDisablesTheFairClass(t *testing.T) {
	tb := keyTable("bounces", []string{"a"},
		[3]string{"a", "finish", "done"}, [3]string{"a", "step", "b"}, [3]string{"b", "back", "a"})
	finishes := umpire.Assumption{Name: "finishIsFair", Fair: []string{"finish"}}
	p := umpire.KeyProgress("finishes", is("a"), is("done"), 4, finishes)
	a, err := umpire.CheckProgress(tb, p, wide)
	require.NoError(t, err)
	requireVerdict(t, umpire.CounterexampleFound, a.Cycle)
	require.Equal(t, []string{"step", "back"}, actionsOf(a.Cycle.Witness), "finish is disabled at b")
	require.NoError(t, p.Replay(tb, umpire.CycleKind, a.Cycle))
}

func TestAStepBoundLeavesProgressUnresolved(t *testing.T) {
	tb := keyTable("chain", []string{"s0"},
		[3]string{"s0", "next", "s1"}, [3]string{"s1", "next", "s2"}, [3]string{"s2", "next", "s3"})
	p := umpire.KeyProgress("reachesS9", func(string) bool { return true }, is("s9"), 8)
	a, err := umpire.CheckProgress(tb, p, umpire.Limits{Name: "short", Steps: 1, Search: 256})
	require.NoError(t, err)
	requireVerdict(t, umpire.Unresolved, a.Deadlock)
	requireVerdict(t, umpire.Unresolved, a.Cycle)
	requireVerdict(t, umpire.Unresolved, a.Deadline)
	require.Contains(t, a.Deadlock.Explanation, "a finite prefix that ends open is not a counterexample")

	a, err = umpire.CheckProgress(tb, p, wide)
	require.NoError(t, err)
	requireVerdict(t, umpire.CounterexampleFound, a.Deadlock)
	require.Equal(t, []string{"next", "next", "next"}, actionsOf(a.Deadlock.Witness))
	requireVerdict(t, umpire.VerifiedWithinLimits, a.Cycle)
	requireVerdict(t, umpire.VerifiedWithinLimits, a.Deadline)
}

func TestAWorkCeilingIsLimitReachedAndKeepsAnEstablishedViolation(t *testing.T) {
	tb := keyTable("fans", []string{"root"},
		[3]string{"root", "fail", "stuck"}, [3]string{"root", "go", "x1"}, [3]string{"x1", "go", "x2"},
		[3]string{"x2", "go", "x3"}, [3]string{"x3", "go", "x1"})
	p := umpire.KeyProgress("never", is("root"), is("done"), 2)
	a, err := umpire.CheckProgress(tb, p, umpire.Limits{Name: "tight", Steps: 8, Search: 3})
	require.NoError(t, err)
	requireVerdict(t, umpire.CounterexampleFound, a.Deadlock)
	require.NoError(t, p.Replay(tb, umpire.DeadlockKind, a.Deadlock))
	requireVerdict(t, umpire.LimitReached, a.Cycle)
	require.Contains(t, a.Cycle.Explanation, "the limits tight allow 3")
	requireVerdict(t, umpire.LimitReached, a.Deadline)
	require.Equal(t, 3, a.Explored, "the check stops at its ceiling")
}

// endless is Within steps no finite path takes: a check must not spend work or memory in it.
const endless = 1 << 40

func TestAHugeDeadlineOnACycleIsMissedWithinTheCeiling(t *testing.T) {
	tb := keyTable("polls", []string{"waiting"},
		[3]string{"waiting", "poll", "waiting"}, [3]string{"waiting", "finish", "done"})
	finishes := umpire.Assumption{Name: "finishIsFair", Fair: []string{"finish"}}
	p := umpire.KeyProgress("finishes", is("waiting"), is("done"), endless, finishes)
	small := umpire.Limits{Name: "small", Steps: 8, Search: 8}
	a, err := umpire.CheckProgress(tb, p, small)
	require.NoError(t, err)
	requireVerdict(t, umpire.CounterexampleFound, a.Deadline)
	require.Equal(t, []string{"poll"}, actionsOf(a.Deadline.Witness), "the path that misses it is a lasso")
	require.Equal(t, 0, a.Deadline.Loop)
	require.NoError(t, p.Replay(tb, umpire.DeadlineKind, a.Deadline))
	requireVerdict(t, umpire.VerifiedWithinLimits, a.Cycle)
	require.LessOrEqual(t, a.Explored, small.Search)

	short := a.Deadline
	short.Loop = -1
	require.ErrorContains(t, p.Replay(tb, umpire.DeadlineKind, short), "the witness takes 1 steps after 'waiting', fewer than")
}

func TestAHugeDeadlineOnAFiniteGraphIsNotMissed(t *testing.T) {
	tb := keyTable("chain", []string{"s0"},
		[3]string{"s0", "next", "s1"}, [3]string{"s1", "next", "s2"}, [3]string{"s2", "done", "s3"},
		[3]string{"s3", "idle", "s3"})
	p := umpire.KeyProgress("reachesS3", func(string) bool { return true }, is("s3"), endless)
	enough := umpire.Limits{Name: "enough", Steps: 8, Search: 16}
	a, err := umpire.CheckProgress(tb, p, enough)
	require.NoError(t, err)
	requireVerdict(t, umpire.VerifiedWithinLimits, a.Deadline)
	requireVerdict(t, umpire.VerifiedWithinLimits, a.Cycle)
	requireVerdict(t, umpire.VerifiedWithinLimits, a.Deadlock)
	require.LessOrEqual(t, a.Explored, enough.Search)

	explored := umpire.Limits{Name: "explored", Steps: 8, Search: 4}
	a, err = umpire.CheckProgress(tb, p, explored)
	require.NoError(t, err)
	requireVerdict(t, umpire.VerifiedWithinLimits, a.Deadlock)
	requireVerdict(t, umpire.LimitReached, a.Cycle)
	requireVerdict(t, umpire.LimitReached, a.Deadline)
	require.Contains(t, a.Deadline.Explanation, "the limits explored allow 4")
	require.LessOrEqual(t, a.Explored, explored.Search)
}

func TestADeadlineWitnessLongerThanTheCeilingIsALasso(t *testing.T) {
	tb := keyTable("polls", []string{"waiting"}, [3]string{"waiting", "poll", "waiting"})
	p := umpire.KeyProgress("finishes", is("waiting"), is("done"), 6)
	a, err := umpire.CheckProgress(tb, p, umpire.Limits{Name: "roomy", Steps: 8, Search: 16})
	require.NoError(t, err)
	require.Len(t, a.Deadline.Witness.Steps, 6, "a witness the ceiling allows takes every step")
	require.NoError(t, p.Replay(tb, umpire.DeadlineKind, a.Deadline))

	a, err = umpire.CheckProgress(tb, p, umpire.Limits{Name: "tight", Steps: 8, Search: 6})
	require.NoError(t, err)
	requireVerdict(t, umpire.CounterexampleFound, a.Deadline)
	require.Equal(t, []string{"poll"}, actionsOf(a.Deadline.Witness))
	require.Equal(t, 0, a.Deadline.Loop)
	require.NoError(t, p.Replay(tb, umpire.DeadlineKind, a.Deadline))
	require.Equal(t, 6, a.Explored)
}

func TestAFairCycleWhoseTourExceedsTheCeilingIsLimitReached(t *testing.T) {
	tb := keyTable("ring", []string{"r0"},
		[3]string{"r0", "go", "r1"}, [3]string{"r1", "go", "r2"}, [3]string{"r2", "go", "r3"},
		[3]string{"r3", "go", "r0"})
	p := umpire.KeyProgress("never", is("r0"), is("done"), 2)
	a, err := umpire.CheckProgress(tb, p, wide)
	require.NoError(t, err)
	requireVerdict(t, umpire.CounterexampleFound, a.Cycle)
	require.NoError(t, p.Replay(tb, umpire.CycleKind, a.Cycle))

	a, err = umpire.CheckProgress(tb, p, umpire.Limits{Name: "tight", Steps: 8, Search: 10})
	require.NoError(t, err)
	requireVerdict(t, umpire.LimitReached, a.Cycle)
	require.LessOrEqual(t, a.Explored, 10)
}

func TestProgressOnATypedMachine(t *testing.T) {
	m := newDoor("door")
	tb := tableOf(t, m)
	shuts := umpire.NewProgress("shuts", func(d door) bool { return d.Phase == open },
		func(d door) bool { return d.Phase == closed }, 1)
	a, err := umpire.CheckProgress(tb, shuts, wide)
	require.NoError(t, err)
	requireVerdict(t, umpire.VerifiedWithinLimits, a.Deadlock)
	requireVerdict(t, umpire.VerifiedWithinLimits, a.Cycle)
	requireVerdict(t, umpire.VerifiedWithinLimits, a.Deadline)

	lockIsFair := umpire.Assumption{Name: "lockIsFair", Fair: []string{"lock"}}
	locks := umpire.NewProgress("locks", func(d door) bool { return d.Phase == closed },
		func(d door) bool { return d.Phase == locked }, 1, lockIsFair)
	a, err = umpire.CheckProgress(tb, locks, wide)
	require.NoError(t, err)
	requireVerdict(t, umpire.CounterexampleFound, a.Cycle)
	require.Equal(t, []string{"turn-right-true", "push"}, actionsOf(a.Cycle.Witness),
		"lock is disabled while the door is open, so a fair path may keep turning and pushing")
	require.NoError(t, locks.Replay(tb, umpire.CycleKind, a.Cycle))
	requireVerdict(t, umpire.CounterexampleFound, a.Deadline)
	require.NoError(t, locks.Replay(tb, umpire.DeadlineKind, a.Deadline))
}

func TestProgressDeclarationsAreChecked(t *testing.T) {
	tb := keyTable("polls", []string{"waiting"}, [3]string{"waiting", "poll", "waiting"})
	_, err := umpire.CheckProgress(tb, umpire.KeyProgress("instant", is("waiting"), is("done"), 0), wide)
	require.EqualError(t, err, "progress instant: within 0 steps is fewer than one")

	ghost := umpire.Assumption{Name: "ghostIsFair", Fair: []string{"ghost"}}
	_, err = umpire.CheckProgress(tb, umpire.KeyProgress("haunted", is("waiting"), is("done"), 1, ghost), wide)
	require.EqualError(t, err, "progress haunted: the assumption ghostIsFair makes ghost fair, which is no action of polls")

	typed := umpire.NewProgress("typed", func(d door) bool { return true }, func(d door) bool { return false }, 1)
	_, err = umpire.CheckProgress(tb, typed, wide)
	require.ErrorContains(t, err, "progress typed: the state waiting of polls is not a checker_test.door")
}

func TestAMachinesAssumptionsJoinEveryProgressCheck(t *testing.T) {
	pushIsFair := umpire.Assumption{Name: "pushIsFair", Fair: []string{"push"}}
	m := newDoor("door").Assumes(pushIsFair)
	tb := tableOf(t, m)
	require.Equal(t, []umpire.Assumption{pushIsFair}, tb.Assumptions)
	shuts := umpire.NewProgress("shuts", func(d door) bool { return d.Phase == open },
		func(d door) bool { return d.Phase == closed }, 1)
	a, err := umpire.CheckProgress(tb, shuts, wide)
	require.NoError(t, err)
	require.Equal(t, []string{"pushIsFair"}, a.Assumptions)

	restricted := m.Restrict("test.door", "pushOnly", push.ActionDecl)
	require.Equal(t, []umpire.Assumption{pushIsFair}, tableOf(t, restricted).Assumptions)
}

func TestSameNamedAssumptionsJoinTheirFairness(t *testing.T) {
	spec := umpire.TableSpec{Machine: "polls", Family: "test.progress", Outcomes: []string{"ok"},
		States: []string{"waiting", "done"}, Actions: []string{"finish", "poll"}, Starts: []string{"waiting"},
		Assumptions: []umpire.Assumption{{Name: "finishIsFair"}},
		Rows: []umpire.Row{
			{Key: "waiting-finish", Source: "waiting", Action: "finish",
				Results: []umpire.Result{{Outcome: "ok", State: "done", Facts: []string{}}}},
			{Key: "waiting-poll", Source: "waiting", Action: "poll",
				Results: []umpire.Result{{Outcome: "ok", State: "waiting", Facts: []string{}}}},
		}}
	tb := umpire.NewTable(spec)
	p := umpire.KeyProgress("finishes", is("waiting"), is("done"), 2,
		umpire.Assumption{Name: "finishIsFair", Fair: []string{"finish"}})
	a, err := umpire.CheckProgress(tb, p, wide)
	require.NoError(t, err)
	require.Equal(t, []string{"finishIsFair"}, a.Assumptions)
	requireVerdict(t, umpire.VerifiedWithinLimits, a.Cycle)
	require.Equal(t, []umpire.Assumption{{Name: "finishIsFair"}}, tb.Assumptions, "the table's own declaration is unchanged")
}
