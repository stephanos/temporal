package checker_test

import (
	"errors"
	"slices"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"
	umpire "go.temporal.io/server/tools/umpire/model/internal/checker"
)

// A detail that begins before it finishes, read as a product that only finishes.

func finishing() *umpire.Table {
	return keyTable("product", []string{"idle"}, [3]string{"idle", "finish", "done"})
}

// detailing is the detail with some of its pairs unknown. Its state orphan is one no start reaches.
func detailing(pairs ...[2]string) *umpire.Table {
	return holed(keyTable("detail", []string{"idle"}, [3]string{"idle", "begin", "working"},
		[3]string{"working", "finish", "done"}, [3]string{"orphan", "finish", "done"}), pairs...)
}

func idleWhileWorking(state string) (string, error) {
	if state == "working" || state == "orphan" {
		return "idle", nil
	}
	return state, nil
}

func TestRefineTablesStopsAtAReachableUnknownPair(t *testing.T) {
	product := finishing()
	reads := umpire.RefinementSpec{MapState: idleWhileWorking}
	whole, err := umpire.RefineTables(detailing(), product, reads)
	require.NoError(t, err)

	for _, c := range []struct {
		name    string
		pair    [2]string
		message string
		prefix  [][2]string
	}{
		{"at a start", [2]string{"idle", "skip"},
			"detail refines product: detail reaches 'idle', where the pair 'idle-skip' is unknown, so no step it " +
				"takes there is shown to be a step of product or a stutter: a hole at idle-skip", nil},
		{"after a step", [2]string{"working", "skip"},
			"detail refines product: detail reaches 'working', where the pair 'working-skip' is unknown, so no step it " +
				"takes there is shown to be a step of product or a stutter: a hole at working-skip",
			[][2]string{{"begin", "working"}}},
	} {
		t.Run(c.name, func(t *testing.T) {
			detail := detailing(c.pair)
			ref, err := umpire.RefineTables(detail, product, reads)
			require.Nil(t, ref, "a hole a start reaches is not read as a disabled pair")
			require.EqualError(t, err, c.message)
			var h *hole
			require.ErrorAs(t, err, &h)
			require.Equal(t, &hole{c.pair[0] + "-" + c.pair[1]}, h)
			re := refinementError(t, err)
			require.Equal(t, umpire.RefinementIncomplete, re.Kind)
			require.Equal(t, pathOf(detail, "idle", c.prefix...), re.Witness)
			require.NoError(t, detail.Replay(re.Witness))
		})
	}

	ref, err := umpire.RefineTables(detailing([2]string{"orphan", "skip"}), product, reads)
	require.NoError(t, err, "a hole no start reaches changes nothing")
	require.Equal(t, whole.Rows, ref.Rows)

	strays := holed(keyTable("detail", []string{"idle"}, [3]string{"idle", "begin", "working"},
		[3]string{"working", "finish", "done"}, [3]string{"done", "reopen", "idle"}), [2]string{"idle", "skip"})
	_, err = umpire.RefineTables(strays, product, reads)
	require.Equal(t, umpire.RefinementUnmatched, refinementError(t, err).Kind, "a step that refines nothing stands beside a hole")
}

func TestComposeTablesReplacementStopsAtAReachableUnknownPair(t *testing.T) {
	replacing := func(detail *umpire.Table) umpire.ComposeSpec {
		return umpire.ComposeSpec{Family: "test.progress", Name: "work", Ceiling: roomy,
			Members: []umpire.ComposeMember{{Field: "work", Table: detail, Replaces: finishing(),
				Refinement: umpire.RefinementSpec{MapState: idleWhileWorking}}}}
	}
	holedDetail := detailing([2]string{"working", "skip"})
	tb, err := umpire.ComposeTables(replacing(holedDetail))
	require.Nil(t, tb)
	re := refinementError(t, err)
	require.Equal(t, umpire.RefinementIncomplete, re.Kind)
	require.ErrorAs(t, err, new(*hole))
	require.NoError(t, holedDetail.Replay(re.Witness))
	_, direct := umpire.RefineTables(holedDetail, finishing(), umpire.RefinementSpec{MapState: idleWhileWorking, CoverStarts: true})
	require.EqualError(t, err, direct.Error(), "the composition reports the refinement's own error")

	tb, err = umpire.ComposeTables(replacing(detailing([2]string{"orphan", "skip"})))
	require.NoError(t, err, "a hole no start reaches changes nothing")
	require.Equal(t, []string{"done", "idle", "working"}, tb.States)
	require.Empty(t, tb.Unknown)
}

// A Monitor that cannot be read on the step into bad.

func unreadAtBad(name, at string) *umpire.Monitor {
	fail := func(state string) error {
		if state == "bad" {
			return &hole{at}
		}
		return nil
	}
	next := func(_, _ string, s umpire.Result) (string, error) { return s.State, nil }
	violated := func(string) (bool, error) { return false, nil }
	evaluation := umpire.EveryStep()
	switch at {
	case "next":
		next = func(_, _ string, s umpire.Result) (string, error) { return s.State, fail(s.State) }
	case "violated":
		violated = func(mon string) (bool, error) { return false, fail(mon) }
	default:
		evaluation = umpire.AfterKey(func(s umpire.Result) (bool, error) { return true, fail(s.State) })
	}
	return umpire.KeyMonitor(name, "m", next, violated, evaluation)
}

func TestAnUnknownMonitorDoesNotEraseAViolationOnItsStep(t *testing.T) {
	tb := keyTable("chain", []string{"s0"},
		[3]string{"s0", "go", "s1"}, [3]string{"s0", "slip", "bad"}, [3]string{"bad", "on", "far"})
	entersBad := umpire.KeyMonitor("entersBad", "m",
		func(_, _ string, s umpire.Result) (string, error) { return s.State, nil },
		func(mon string) (bool, error) { return mon == "bad", nil }, umpire.EveryStep())
	bad := neverBad(tb)
	holds := umpire.KeyTransitionProperty(tb, "holds", always)
	for _, at := range []string{"next", "violated", "after"} {
		unread := umpire.MonitorVerdict{Name: "unread", State: "m", Verdict: umpire.MonitorUnknown}
		for _, c := range []struct {
			name        string
			q           *umpire.Query
			explanation string
			monitor     string
			verdicts    []umpire.MonitorVerdict
		}{
			{"the Property fails",
				umpire.KeyVerify("q", bad, umpire.KeyFreeScenario(tb, "claim "+at, "s0"), four).Watch(unreadAtBad("unread", at)),
				"neverBad fails at {bad}", "", []umpire.MonitorVerdict{unread}},
			{"an earlier Monitor is violated",
				umpire.KeyVerify("q", holds, umpire.KeyFreeScenario(tb, "monitor "+at, "s0"), four).
					Watch(entersBad, unreadAtBad("unread", at)),
				"the monitor entersBad is violated at {bad}", "entersBad",
				[]umpire.MonitorVerdict{{Name: "entersBad", State: "bad", Verdict: umpire.MonitorViolated}, unread}},
		} {
			t.Run(at+": "+c.name, func(t *testing.T) {
				c.q.Unknown = isHole
				a, err := c.q.Answer()
				require.NoError(t, err)
				require.Len(t, a.Unknown, 1)
				require.ErrorAs(t, a.Unknown[0].Cause, new(*hole))
				a.Unknown[0].Cause = nil
				require.Equal(t, umpire.Answer{Outcome: umpire.CounterexampleFound, Witness: pathOf(tb, "s0", [2]string{"slip", "bad"}),
					Explored: 3, Explanation: c.explanation, Rows: []string{"s0-slip"}, Exercised: true, Monitor: c.monitor,
					Monitors: c.verdicts, Expanded: 2,
					Unknown: []umpire.UnknownReach{{Kind: umpire.UnknownClaim, Row: "s0-slip", Source: "s0", Action: "slip",
						Prefix: pathOf(tb, "s0")}}}, a, "the step is a counterexample, is unknown, and is not searched past")
				require.False(t, a.Incomplete())
				require.NoError(t, c.q.Replay(a))
			})
		}
	}

	reachesBad := umpire.KeyProperty(tb, "reachesBad", nil, "",
		func(s umpire.Result) (bool, error) { return s.State == "bad", nil })
	find := umpire.KeyFind("q", reachesBad, umpire.KeyFreeScenario(tb, "find", "s0"), four).Watch(unreadAtBad("unread", "next"))
	find.Unknown = isHole
	a, err := find.Answer()
	require.NoError(t, err)
	require.Equal(t, umpire.Found, a.Outcome, "a Monitor takes nothing away from a find, read or not")
	require.Equal(t, []string{"s0-slip"}, a.Rows)
	require.NoError(t, find.Replay(a))
}

func TestAnUnknownPairIsKeyedAsItsRow(t *testing.T) {
	spec := keyCopy(keyTable("chain", []string{"s0"}, [3]string{"s0", "go", "s1"}))
	spec.Unknown = []umpire.UnknownPair{{Row: "bogus", Source: "s1", Action: "go"}}
	tb := umpire.NewTable(spec)
	require.EqualError(t, tb.Err(), "chain: the unknown pair 'bogus' is at 's1' and takes go, so its key is 's1-go'")
	require.ErrorAs(t, tb.Err(), new(*umpire.Error))

	spec.Unknown = []umpire.UnknownPair{{Row: "s1-go", Source: "s1", Action: "go"}}
	require.NoError(t, umpire.NewTable(spec).Err())
}

// wideTable is a one-state table with n classes of the action meet, and one row of every class with
// results results each, all back to its state.
func wideTable(name string, classes, results int) *umpire.Table {
	spec := umpire.TableSpec{Machine: name, Family: "test.wide", States: []string{"s"}, Outcomes: []string{"ok"},
		Starts: []string{"s"}}
	for i := range classes {
		class := "meet-" + strconv.Itoa(i)
		row := umpire.Row{Key: "s-" + class, Source: "s", Action: class}
		for range results {
			row.Results = append(row.Results, umpire.Result{Outcome: "ok", State: "s", Facts: []string{}})
		}
		spec.Actions = append(spec.Actions, class)
		spec.Rows = append(spec.Rows, row)
	}
	return umpire.NewTable(spec)
}

func TestComposeTablesRefusesAProductBeforeBuildingIt(t *testing.T) {
	meeting := func(classes, results int, ceiling umpire.ComposeCeiling) umpire.ComposeSpec {
		return umpire.ComposeSpec{Family: "test.wide", Name: "wide", Ceiling: ceiling,
			Members: []umpire.ComposeMember{{Field: "left", Table: wideTable("left", classes, results)},
				{Field: "right", Table: wideTable("right", classes, results)}},
			Syncs: []umpire.ComposeSync{{Name: "meet", FirstMember: "left", FirstAction: "meet",
				SecondMember: "right", SecondAction: "meet"}}}
	}
	for _, c := range []struct {
		name string
		spec umpire.ComposeSpec
		want umpire.ComposeLimitError
	}{
		{"the classes of a synchronized step",
			meeting(300, 1, umpire.ComposeCeiling{States: 1, Evaluations: 1, Results: 1}),
			umpire.ComposeLimitError{Composition: "wide", Resource: "evaluations", Ceiling: 1, Needed: 90000}},
		{"the results of a synchronized step",
			meeting(1, 300, umpire.ComposeCeiling{States: 1, Evaluations: 1, Results: 100}),
			umpire.ComposeLimitError{Composition: "wide", Resource: "results", Ceiling: 100, Needed: 90000}},
	} {
		t.Run(c.name, func(t *testing.T) {
			var tb *umpire.Table
			var err error
			allocations := testing.AllocsPerRun(1, func() { tb, err = umpire.ComposeTables(c.spec) })
			require.Equal(t, &c.want, limitError(t, tb, err))
			require.Less(t, allocations, 1000.0, "a product of 90000 is refused by its size, before any of it is built")
		})
	}

	fits, err := umpire.ComposeTables(meeting(3, 4, umpire.ComposeCeiling{States: 1, Evaluations: 9, Results: 144}))
	require.NoError(t, err, "nine synchronized classes of sixteen results each fit a ceiling of exactly that")
	require.Len(t, fits.Rows, 9)
	tb, err := umpire.ComposeTables(meeting(3, 4, umpire.ComposeCeiling{States: 1, Evaluations: 9, Results: 143}))
	require.Equal(t, &umpire.ComposeLimitError{Composition: "wide", Resource: "results", Ceiling: 143, Needed: 144},
		limitError(t, tb, err))
}

// meetingRight is a one-state table whose one class of meet is, at its state, a disabled pair, an
// unknown pair, or a row with no result.
func meetingRight(pair string) *umpire.Table {
	spec := umpire.TableSpec{Machine: "right", Family: "test.wide", States: []string{"s"}, Actions: []string{"meet-0"},
		Outcomes: []string{"ok"}, Starts: []string{"s"}}
	switch pair {
	case "unknown":
		spec.Unknown = []umpire.UnknownPair{{Row: "s-meet-0", Source: "s", Action: "meet-0", Cause: &hole{"s-meet-0"}}}
	case "resultless":
		spec.Rows = []umpire.Row{{Key: "s-meet-0", Source: "s", Action: "meet-0"}}
	default:
	}
	return umpire.NewTable(spec)
}

func TestComposeTablesExpandsNoResultOfAStepThatDoesNotHappen(t *testing.T) {
	wide := wideTable("left", 1, 5000)
	for _, c := range []struct {
		name    string
		members [2]*umpire.Table
		unknown []string
	}{
		{"the second move is disabled", [2]*umpire.Table{wide, meetingRight("disabled")}, nil},
		{"the second move is unknown", [2]*umpire.Table{wide, meetingRight("unknown")}, []string{"s_s-meet-0-0"}},
		{"the second move has no result", [2]*umpire.Table{wide, meetingRight("resultless")}, nil},
		{"the first move is disabled", [2]*umpire.Table{meetingRight("disabled"), wide}, nil},
		{"the first move is unknown", [2]*umpire.Table{meetingRight("unknown"), wide}, []string{"s_s-meet-0-0"}},
	} {
		t.Run(c.name, func(t *testing.T) {
			spec := umpire.ComposeSpec{Family: "test.wide", Name: "wide",
				Ceiling: umpire.ComposeCeiling{States: 1, Evaluations: 1, Results: 1},
				Members: []umpire.ComposeMember{{Field: "left", Table: c.members[0]}, {Field: "right", Table: c.members[1]}},
				Syncs: []umpire.ComposeSync{{Name: "meet", FirstMember: "left", FirstAction: "meet",
					SecondMember: "right", SecondAction: "meet"}}}
			var tb *umpire.Table
			var err error
			allocations := testing.AllocsPerRun(1, func() { tb, err = umpire.ComposeTables(spec) })
			require.NoError(t, err, "a step that does not happen produces no result, so none counts")
			require.Less(t, allocations, 1000.0, "none of the other move's 5000 results is expanded")
			require.Empty(t, tb.Rows)
			var unknown []string
			for _, u := range tb.Unknown {
				unknown = append(unknown, u.Row)
			}
			require.Equal(t, c.unknown, unknown)
		})
	}
}

// One rule reads every pair: an absent row and a row with no result are disabled, an unknown pair is
// unknown, a step that needs several moves is disabled if one is and unknown if one is and none is
// disabled.

func TestAResultlessMoveDisablesAStepWhoseOtherMoveIsUnknown(t *testing.T) {
	for _, c := range []struct {
		name    string
		members [2]*umpire.Table
	}{
		{"the result-less move first", [2]*umpire.Table{meetingRight("resultless"), meetingRight("unknown")}},
		{"the unknown move first", [2]*umpire.Table{meetingRight("unknown"), meetingRight("resultless")}},
	} {
		t.Run(c.name, func(t *testing.T) {
			tb, err := umpire.ComposeTables(umpire.ComposeSpec{Family: "test.wide", Name: "wide", Ceiling: roomy,
				Members: []umpire.ComposeMember{{Field: "left", Table: c.members[0]}, {Field: "right", Table: c.members[1]}},
				Syncs: []umpire.ComposeSync{{Name: "meet", FirstMember: "left", FirstAction: "meet",
					SecondMember: "right", SecondAction: "meet"}}})
			require.NoError(t, err)
			require.Empty(t, tb.Unknown, "a step one move of which cannot happen is disabled, whatever the other is")
			require.Empty(t, tb.Rows)

			a, err := umpire.KeyVerify("q", umpire.KeyTransitionProperty(tb, "holds", always),
				umpire.KeyFreeScenario(tb, "anything", "s_s"), four).Answer()
			require.NoError(t, err)
			require.Equal(t, umpire.VerifiedWithinLimits, a.Outcome)
			require.False(t, a.Incomplete(), "no step that can happen is unknown")
		})
	}
}

// resultless is the table of keyTable with a row of no result added at some pairs.
func resultless(tb *umpire.Table, pairs ...[2]string) *umpire.Table {
	spec := keyCopy(tb)
	for _, p := range pairs {
		if !slices.Contains(spec.Actions, p[1]) {
			spec.Actions = append(spec.Actions, p[1])
		}
		spec.Rows = append(spec.Rows, umpire.Row{Key: p[0] + "-" + p[1], Source: p[0], Action: p[1]})
	}
	return umpire.NewTable(spec)
}

func TestARowWithNoResultIsADisabledPair(t *testing.T) {
	stalls := resultless(keyTable("stalls", []string{"waiting"},
		[3]string{"waiting", "fail", "stuck"}, [3]string{"waiting", "finish", "done"}), [2]string{"stuck", "retry"})
	t.Run("a state whose only row has no result is stuck", func(t *testing.T) {
		require.Equal(t, "stuck", stalls.Stuck)
	})
	t.Run("and is a deadlock", func(t *testing.T) {
		finishes := umpire.KeyProgress("finishes", is("waiting"), is("done"), 3)
		a, err := umpire.CheckProgress(stalls, finishes, wide)
		require.NoError(t, err)
		requireVerdict(t, umpire.CounterexampleFound, a.Deadlock)
		require.Equal(t, []string{"fail"}, actionsOf(a.Deadlock.Witness))
		require.NoError(t, finishes.Replay(stalls, umpire.DeadlockKind, a.Deadlock))
		require.False(t, a.Incomplete())
	})
	t.Run("a fair class is disabled where its row has no result", func(t *testing.T) {
		bounces := resultless(keyTable("bounces", []string{"a"},
			[3]string{"a", "finish", "done"}, [3]string{"a", "step", "b"}, [3]string{"b", "back", "a"}), [2]string{"b", "finish"})
		finishIsFair := umpire.Assumption{Name: "finishIsFair", Fair: []string{"finish"}}
		eventually := umpire.KeyProgress("finishes", is("a"), is("done"), 4, finishIsFair)
		a, err := umpire.CheckProgress(bounces, eventually, wide)
		require.NoError(t, err)
		requireVerdict(t, umpire.CounterexampleFound, a.Cycle)
		require.Equal(t, []string{"step", "back"}, actionsOf(a.Cycle.Witness))
		require.NoError(t, eventually.Replay(bounces, umpire.CycleKind, a.Cycle))
	})
	t.Run("a search takes no step by it and a witness cannot", func(t *testing.T) {
		chain := resultless(forkedChain(), [2]string{"s1", "leap"})
		q := umpire.KeyVerify("q", umpire.KeyTransitionProperty(chain, "holds", always),
			umpire.KeyFreeScenario(chain, "anything", "s0"), four)
		answer, err := q.Answer()
		require.NoError(t, err)
		require.Equal(t, umpire.VerifiedWithinLimits, answer.Outcome)
		require.Empty(t, answer.Unknown)
		leaps := pathOf(chain, "s0", [2]string{"go", "s1"}, [2]string{"leap", "s1"})
		require.EqualError(t, chain.Replay(leaps), "chain: step 2 takes leap, which is not enabled at 's1'")
	})
}

// unreadAt is a progress predicate that cannot be read at one state, and fails outright at another.
func unreadAt(unread, failing string, f func(string) bool) func(string) (bool, error) {
	return func(s string) (bool, error) {
		switch s {
		case unread:
			return false, &hole{s}
		case failing:
			return false, errors.New("the claim was read past a state it could not be read at")
		default:
			return f(s), nil
		}
	}
}

// A progress claim that cannot be read at the state murky, one step off a path that violates it. The
// state is unknown evidence, as a step a Property cannot be read on is: the violation stands, the
// state is listed, and the check goes no further through it, so beyond is never read.
func TestAnUnreadProgressStateDoesNotEraseAViolation(t *testing.T) {
	aside := [][3]string{{"waiting", "look", "murky"}, {"murky", "go", "beyond"}, {"beyond", "finish", "done"}}
	for _, c := range []struct {
		name    string
		edges   [][3]string
		kind    umpire.ProgressKind
		witness [][2]string
		loop    int
	}{
		{"a deadlock", [][3]string{{"waiting", "fail", "stuck"}}, umpire.DeadlockKind, [][2]string{{"fail", "stuck"}}, -1},
		{"a missed deadline", [][3]string{{"waiting", "poll", "later"}, {"later", "poll", "latest"}, {"latest", "finish", "done"}},
			umpire.DeadlineKind, [][2]string{{"poll", "later"}, {"poll", "latest"}}, -1},
		{"a fair cycle", [][3]string{{"waiting", "poll", "waiting"}}, umpire.CycleKind, [][2]string{{"poll", "waiting"}}, 0},
	} {
		for _, unread := range []string{"from", "to"} {
			t.Run(c.name+" beside an unread "+unread, func(t *testing.T) {
				tb := keyTable("murky", []string{"waiting"}, append(slices.Clone(c.edges), aside...)...)
				from, to := unreadAt("", "", is("waiting")), unreadAt("", "", is("done"))
				if unread == "from" {
					from = unreadAt("murky", "beyond", is("waiting"))
				} else {
					to = unreadAt("murky", "beyond", is("done"))
				}
				claim := umpire.KeyProgressFunc("finishes", from, to, 2)

				_, err := umpire.CheckProgress(tb, claim, wide)
				var h *hole
				require.ErrorAs(t, err, &h, "an error no classifier accepts fails the check, with its type")
				require.Equal(t, &hole{"murky"}, h)
				claim.Unknown = func(error) bool { return false }
				_, err = umpire.CheckProgress(tb, claim, wide)
				require.ErrorAs(t, err, &h, "and so does one the classifier rejects")

				claim.Unknown = isHole
				a, err := umpire.CheckProgress(tb, claim, wide)
				require.NoError(t, err)
				verdict := map[umpire.ProgressKind]umpire.ProgressVerdict{umpire.DeadlockKind: a.Deadlock,
					umpire.CycleKind: a.Cycle, umpire.DeadlineKind: a.Deadline}[c.kind]
				requireVerdict(t, umpire.CounterexampleFound, verdict)
				require.Equal(t, pathOf(tb, "waiting", c.witness...), verdict.Witness)
				require.Equal(t, c.loop, verdict.Loop)
				require.NoError(t, claim.Replay(tb, c.kind, verdict))
				require.Equal(t, []umpire.UnknownReach{{Kind: umpire.UnknownClaim, Source: "murky", Depth: 1,
					Prefix: pathOf(tb, "waiting", [2]string{"look", "murky"}), Cause: &hole{"murky"}}}, a.Unknown)
				require.True(t, a.Incomplete(), "the kinds of violation it rules out may lie behind the unread state")
			})
		}
	}
}

func TestAnUnreadProgressStateLeavesAVerifiedClaimIncomplete(t *testing.T) {
	tb := keyTable("murky", []string{"waiting"}, [3]string{"waiting", "finish", "done"}, [3]string{"waiting", "go", "mid"},
		[3]string{"mid", "finish", "done"}, [3]string{"mid", "look", "murky"}, [3]string{"murky", "finish", "done"})
	// Two steps is the deadline: the one step from waiting to mid is within it, and the step on to the
	// state the claim cannot be read at counts as no step of a path that misses it.
	claim := umpire.KeyProgressFunc("finishes", unreadAt("", "", is("waiting")), unreadAt("murky", "", is("done")), 2)
	claim.Unknown = isHole

	a, err := umpire.CheckProgress(tb, claim, wide)
	require.NoError(t, err)
	for _, v := range []umpire.ProgressVerdict{a.Deadlock, a.Cycle, a.Deadline} {
		requireVerdict(t, umpire.VerifiedWithinLimits, v)
	}
	require.Equal(t, []umpire.UnknownReach{{Kind: umpire.UnknownClaim, Source: "murky", Depth: 2,
		Prefix: pathOf(tb, "waiting", [2]string{"go", "mid"}, [2]string{"look", "murky"}), Cause: &hole{"murky"}}}, a.Unknown)
	require.True(t, a.Incomplete())

	// One step explores waiting and mid: the state the claim cannot be read at is past the bound.
	a, err = umpire.CheckProgress(tb, claim, umpire.Limits{Name: "one step", Steps: 1, Search: 256})
	require.NoError(t, err)
	require.Empty(t, a.Unknown, "a state the check never reaches is never read")
	require.False(t, a.Incomplete())

	// A state the claim cannot be read at, with no step: whether it is where the claim leads is not
	// known, so it is no deadlock, by whichever of the two steps that reach it.
	end := keyTable("murky", []string{"waiting"}, [3]string{"waiting", "finish", "done"}, [3]string{"waiting", "look", "murky"},
		[3]string{"waiting", "peek", "murky"})
	a, err = umpire.CheckProgress(end, claim, wide)
	require.NoError(t, err)
	requireVerdict(t, umpire.VerifiedWithinLimits, a.Deadlock)
	require.Len(t, a.Unknown, 1)
	require.True(t, a.Incomplete())
}
