package checker_test

import (
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	umpire "go.temporal.io/server/tools/umpire/model/internal/checker"
)

// keyCopy is a typed table as the spec of a table that carries only its keys: no typed state and no
// typed step.
func keyCopy(tb *umpire.Table) umpire.TableSpec {
	spec := umpire.TableSpec{Machine: tb.Machine, Owner: tb.Owner, Family: tb.Family, States: tb.States,
		Actions: tb.Actions, Outcomes: tb.Outcomes, Facts: tb.Facts, Starts: tb.Starts, Ends: tb.Ends,
		StateFields: tb.StateFields, Entity: tb.Entity, Evidence: tb.Evidence, Assumptions: tb.Assumptions}
	for _, r := range tb.Rows {
		row := umpire.Row{Key: r.Key, Source: r.Source, Action: r.Action}
		for _, res := range r.Results {
			row.Results = append(row.Results, umpire.Result{Outcome: res.Outcome, State: res.State, Facts: res.Facts})
		}
		spec.Rows = append(spec.Rows, row)
	}
	return spec
}

// phaseOf is the first field of a state key, which a door's and a key's states spell first.
func phaseOf(key string) string {
	phase, _, _ := strings.Cut(key, "-")
	return phase
}

func always(string, umpire.Result) (bool, error) { return true, nil }

func countOpeningKeys(at umpire.Evaluation) *umpire.Monitor {
	return umpire.KeyMonitor("openedTwice", "0",
		func(n, before string, step umpire.Result) (string, error) {
			if phaseOf(before) != "open" && phaseOf(step.State) == "open" {
				return map[string]string{"0": "1", "1": "2", "2": "2"}[n], nil
			}
			return n, nil
		},
		func(n string) (bool, error) { return n == "2", nil }, at)
}

type doorMachine = *umpire.Machine[door, doorOutcome, doorFact]

func TestKeyLevelQueryMatchesTypedQuery(t *testing.T) {
	strong := turn.With(right{Strong: true})
	opensLoudly := func(m doorMachine) *umpire.Property[door] {
		return m.Property("opensLoudly").When(strong).Holds(func(s doorStep) bool { return s.State.Phase == open })
	}
	opensLoudlyKeys := func(tb *umpire.Table) *umpire.PropertyDecl {
		return umpire.KeyProperty(tb, "opensLoudly", func(a string) bool { return a == "turn-right-true" },
			"turn-right-true", func(s umpire.Result) (bool, error) { return phaseOf(s.State) == "open", nil })
	}
	turnThenPush := func(m doorMachine) *umpire.Scenario[door] {
		return m.Scenario("turnThenPush").Starts(door{Phase: closed}).Actions(strong, push.With())
	}
	turnThenPushKeys := func(tb *umpire.Table) *umpire.ScenarioDecl {
		return umpire.KeyScenario(tb, "turnThenPush", "closed-false", "turn-right-true", "push")
	}
	free := func(m doorMachine) *umpire.Scenario[door] {
		return m.Scenario("anything").Starts(door{Phase: closed}).Free()
	}
	freeKeys := func(tb *umpire.Table) *umpire.ScenarioDecl {
		return umpire.KeyFreeScenario(tb, "anything", "closed-false")
	}
	staysShut := func(m doorMachine) *umpire.Property[door] {
		return m.Property("staysShut").HoldsAcross(func(_ door, after doorStep) bool { return after.State.Phase != open })
	}
	staysShutKeys := func(tb *umpire.Table) *umpire.PropertyDecl {
		return umpire.KeyTransitionProperty(tb, "staysShut",
			func(_ string, s umpire.Result) (bool, error) { return phaseOf(s.State) != "open", nil })
	}
	anythingKeys := func(tb *umpire.Table, limits umpire.Limits) *umpire.Query {
		return umpire.KeyVerify("q", umpire.KeyTransitionProperty(tb, "always", always), freeKeys(tb), limits)
	}
	afterOpening := func(r umpire.Result) bool { return len(r.Facts) > 0 }

	for _, c := range []struct {
		name  string
		typed func(doorMachine) *umpire.Query
		keyed func(*umpire.Table) *umpire.Query
		want  umpire.Outcome
	}{
		{"a pinned find",
			func(m doorMachine) *umpire.Query { return turnThenPush(m).Find("q", opensLoudly(m), two) },
			func(tb *umpire.Table) *umpire.Query {
				return umpire.KeyFind("q", opensLoudlyKeys(tb), turnThenPushKeys(tb), two)
			}, umpire.Found},
		{"a free find",
			func(m doorMachine) *umpire.Query { return free(m).Find("q", opensLoudly(m), two) },
			func(tb *umpire.Table) *umpire.Query {
				return umpire.KeyFind("q", opensLoudlyKeys(tb), freeKeys(tb), two)
			},
			umpire.Found},
		{"a find no trace realizes",
			func(m doorMachine) *umpire.Query {
				never := m.Property("never").When(push.With()).Holds(func(s doorStep) bool { return s.State.Phase == open })
				return turnThenPush(m).Find("q", never, two)
			},
			func(tb *umpire.Table) *umpire.Query {
				never := umpire.KeyProperty(tb, "never", func(a string) bool { return a == "push" }, "push",
					func(s umpire.Result) (bool, error) { return phaseOf(s.State) == "open", nil })
				return umpire.KeyFind("q", never, turnThenPushKeys(tb), two)
			}, umpire.NotFound},
		{"a pinned same-step verify",
			func(m doorMachine) *umpire.Query { return turnThenPush(m).Verify("q", opensLoudly(m), two) },
			func(tb *umpire.Table) *umpire.Query {
				return umpire.KeyVerify("q", opensLoudlyKeys(tb), turnThenPushKeys(tb), two)
			}, umpire.VerifiedWithinLimits},
		{"a transition verify that fails",
			func(m doorMachine) *umpire.Query { return free(m).Verify("q", staysShut(m), two) },
			func(tb *umpire.Table) *umpire.Query {
				return umpire.KeyVerify("q", staysShutKeys(tb), freeKeys(tb), two)
			},
			umpire.CounterexampleFound},
		{"a transition verify that holds",
			func(m doorMachine) *umpire.Query {
				final := m.Property("lockedIsFinal").HoldsAcross(func(before door, after doorStep) bool {
					return before.Phase != locked || after.State.Phase == locked
				})
				return free(m).Verify("q", final, two)
			},
			func(tb *umpire.Table) *umpire.Query {
				final := umpire.KeyTransitionProperty(tb, "lockedIsFinal", func(before string, s umpire.Result) (bool, error) {
					return phaseOf(before) != "locked" || phaseOf(s.State) == "locked", nil
				})
				return umpire.KeyVerify("q", final, freeKeys(tb), two)
			}, umpire.VerifiedWithinLimits},
		{"a search past its limit",
			func(m doorMachine) *umpire.Query {
				p := m.Property("always").HoldsAcross(func(door, doorStep) bool { return true })
				return free(m).Verify("q", p, tiny)
			},
			func(tb *umpire.Table) *umpire.Query { return anythingKeys(tb, tiny) }, umpire.LimitReached},
		{"a verify a monitor fails",
			func(m doorMachine) *umpire.Query { return anything(m).Watch(countOpenings(umpire.EveryStep())) },
			func(tb *umpire.Table) *umpire.Query {
				return anythingKeys(tb, four).Watch(countOpeningKeys(umpire.EveryStep()))
			}, umpire.CounterexampleFound},
		{"a verify a monitor read at ends holds on",
			func(m doorMachine) *umpire.Query { return anything(m).Watch(countOpenings(umpire.AtEnds())) },
			func(tb *umpire.Table) *umpire.Query {
				return anythingKeys(tb, four).Watch(countOpeningKeys(umpire.AtEnds()))
			}, umpire.VerifiedWithinLimits},
		{"a verify a monitor read after some steps fails",
			func(m doorMachine) *umpire.Query { return anything(m).Watch(countOpenings(umpire.After(afterOpening))) },
			func(tb *umpire.Table) *umpire.Query {
				at := umpire.AfterKey(func(r umpire.Result) (bool, error) { return afterOpening(r), nil })
				return anythingKeys(tb, four).Watch(countOpeningKeys(at))
			}, umpire.CounterexampleFound},
		{"a find a violated monitor watches",
			func(m doorMachine) *umpire.Query {
				twice := m.Scenario("twice").Starts(door{Phase: closed}).Actions(strong, push.With(), strong)
				return twice.Find("q", opensLoudly(m), four).Watch(countOpenings(umpire.EveryStep()))
			},
			func(tb *umpire.Table) *umpire.Query {
				twice := umpire.KeyScenario(tb, "twice", "closed-false", "turn-right-true", "push", "turn-right-true")
				return umpire.KeyFind("q", opensLoudlyKeys(tb), twice, four).Watch(countOpeningKeys(umpire.EveryStep()))
			}, umpire.Found},
		{"a monitor failing at the search limit",
			func(m doorMachine) *umpire.Query {
				cramped := umpire.Limits{Name: "cramped", Steps: 4, Actions: 4, Search: 6}
				p := m.Property("always").HoldsAcross(func(door, doorStep) bool { return true })
				return free(m).Verify("q", p, cramped).Watch(countOpenings(umpire.EveryStep()))
			},
			func(tb *umpire.Table) *umpire.Query {
				cramped := umpire.Limits{Name: "cramped", Steps: 4, Actions: 4, Search: 6}
				return anythingKeys(tb, cramped).Watch(countOpeningKeys(umpire.EveryStep()))
			}, umpire.CounterexampleFound},
	} {
		t.Run(c.name, func(t *testing.T) {
			m := newDoor("door")
			want, err := c.typed(m).Answer()
			require.NoError(t, err)
			require.Equal(t, c.want, want.Outcome)

			tb := umpire.NewTable(keyCopy(tableOf(t, m)))
			q := c.keyed(tb)
			got, err := q.Answer()
			require.NoError(t, err)
			require.Equal(t, want, got, "outcome, explored states, rows, witness, exercise and monitor verdicts")
			require.False(t, got.Incomplete())
			if got.Witness != nil {
				require.NoError(t, q.Replay(got))
				require.NoError(t, tb.Replay(got.Witness))
			}
		})
	}
}

func TestKeyLevelMonitorsKeepHistoriesApart(t *testing.T) {
	tb := umpire.NewTable(keyCopy(tableOf(t, forked())))
	neverEnds := umpire.KeyTransitionProperty(tb, "neverEnds",
		func(_ string, s umpire.Result) (bool, error) { return s.State != "s2", nil })
	plain, err := umpire.KeyVerify("plain", neverEnds, umpire.KeyFreeScenario(tb, "plain", "s0"), four).Answer()
	require.NoError(t, err)
	require.Equal(t, umpire.CounterexampleFound, plain.Outcome)
	require.Equal(t, 3, plain.Explored, "left and right commute: both reach s1")

	firstOutcome := umpire.KeyMonitor("firstOutcome", "none",
		func(mon, _ string, step umpire.Result) (string, error) {
			if mon == "none" {
				return step.Outcome, nil
			}
			return mon, nil
		},
		func(string) (bool, error) { return false, nil }, umpire.EveryStep())
	q := umpire.KeyVerify("watched", neverEnds, umpire.KeyFreeScenario(tb, "watched", "s0"), four).Watch(firstOutcome)
	watched, err := q.Answer()
	require.NoError(t, err)
	require.Equal(t, 5, watched.Explored, "s1 and s2 once per first outcome")
	require.Equal(t, umpire.CounterexampleFound, watched.Outcome, "the monitor takes no counterexample away")
	require.Equal(t, plain.Rows, watched.Rows)
	require.Equal(t, plain.Witness, watched.Witness)
	require.Equal(t, []umpire.MonitorVerdict{{Name: "firstOutcome", State: "l", Verdict: umpire.MonitorHeld}}, watched.Monitors)
	require.NoError(t, q.Replay(watched))
}

// Through a refinement: a Property of the abstract door read on the concrete one.

func abstractKeyOf(state string) (string, error) {
	if phaseOf(state) == "open" {
		return "ajar", nil
	}
	return "shut", nil
}

func TestKeyLevelThroughQueryReadsByMapAndName(t *testing.T) {
	for _, c := range []struct {
		name  string
		typed func(*umpire.Machine[abstractDoor, doorOutcome, abstractFact]) *umpire.Property[abstractDoor]
		keyed func(*umpire.Table) *umpire.PropertyDecl
		want  umpire.Outcome
	}{
		{"a same-step claim reads the state by its map and the facts by name",
			func(a *umpire.Machine[abstractDoor, doorOutcome, abstractFact]) *umpire.Property[abstractDoor] {
				return a.Property("turnOpens").WhenAction("turn").Holds(func(s abstractStep) bool {
					return s.State.Phase == ajar && s.Outcome == ok && slices.Equal(s.Facts, []abstractFact{opens})
				})
			},
			func(tb *umpire.Table) *umpire.PropertyDecl {
				return umpire.KeyProperty(tb, "turnOpens", func(a string) bool { return phaseOf(a) == "turn" }, "turn",
					func(s umpire.Result) (bool, error) {
						return s.State == "ajar" && s.Outcome == "ok" && slices.Equal(s.Facts, []string{"opened"}), nil
					})
			}, umpire.VerifiedWithinLimits},
		{"a transition claim reads both states by their map",
			func(a *umpire.Machine[abstractDoor, doorOutcome, abstractFact]) *umpire.Property[abstractDoor] {
				return a.Property("staysShut").HoldsAcross(func(before abstractDoor, after abstractStep) bool {
					return before.Phase == shut && after.State.Phase == shut
				})
			},
			func(tb *umpire.Table) *umpire.PropertyDecl {
				return umpire.KeyTransitionProperty(tb, "staysShut", func(before string, s umpire.Result) (bool, error) {
					return before == "shut" && s.State == "shut", nil
				})
			}, umpire.CounterexampleFound},
	} {
		t.Run(c.name, func(t *testing.T) {
			abstract := abstractMachine(true)
			concrete := newDoor("concrete").Refines(abstract, abstractOf)
			typed := concrete.Scenario("anything").Starts(door{Phase: closed}).Free().
				VerifyRefined("q", c.typed(abstract), concrete.Via(abstract), four)
			want, err := typed.Answer()
			require.NoError(t, err)
			require.Equal(t, c.want, want.Outcome)

			product := umpire.NewTable(keyCopy(tableOf(t, abstract)))
			detail := umpire.NewTable(keyCopy(tableOf(t, concrete)))
			ref, err := umpire.RefineTables(detail, product, umpire.RefinementSpec{MapState: abstractKeyOf})
			require.NoError(t, err)
			q := umpire.KeyVerifyRefined("q", c.keyed(product), umpire.KeyFreeScenario(detail, "anything", "closed-false"), ref, four)
			got, err := q.Answer()
			require.NoError(t, err)
			require.Equal(t, want, got)
			if got.Witness != nil {
				require.NoError(t, q.Replay(got))
				require.NoError(t, detail.Replay(got.Witness))
			}
		})
	}
}

func TestAThroughQueryReadsKeysThroughAKeyRefinementOnly(t *testing.T) {
	abstract := abstractMachine(true)
	concrete := newDoor("concrete").Refines(abstract, abstractOf)
	product := umpire.NewTable(keyCopy(tableOf(t, abstract)))
	detail := umpire.NewTable(keyCopy(tableOf(t, concrete)))
	ref, err := umpire.RefineTables(detail, product, umpire.RefinementSpec{MapState: abstractKeyOf})
	require.NoError(t, err)
	anywhere := umpire.KeyFreeScenario(detail, "anything", "closed-false")

	typedClaim := abstract.Property("typed").HoldsAcross(func(abstractDoor, abstractStep) bool { return true })
	_, err = umpire.KeyVerifyRefined("q", typedClaim.PropertyDecl, anywhere, ref, four).Answer()
	require.EqualError(t, err, "query q: typed reads typed states, and the refinement of abstract by concrete reads keys")

	typedRef, err := concrete.Refinement()
	require.NoError(t, err)
	keyClaim := umpire.KeyTransitionProperty(product, "keyed", always)
	_, err = umpire.KeyVerifyRefined("q", keyClaim, anywhere, typedRef, four).Answer()
	require.EqualError(t, err, "query q: keyed reads keys, and the refinement of abstract by concrete reads typed states")

	other := umpire.NewTable(keyCopy(tableOf(t, abstract)))
	elsewhere := umpire.KeyTransitionProperty(other, "elsewhere", always)
	_, err = umpire.KeyVerifyRefined("q", elsewhere, anywhere, ref, four).Answer()
	require.EqualError(t, err, "query q: the refinement of abstract by concrete is not the one checked of the table "+
		"anything runs on by the table elsewhere is declared on")

	_, err = umpire.KeyVerifyRefined("q", keyClaim, anywhere, nil, four).Answer()
	require.EqualError(t, err, "query q: a refined Query names the refinement it reads through")
}

// A chain with one fork, and the pairs of it that are unknown.

type hole struct{ at string }

func (h *hole) Error() string { return "a hole at " + h.at }

func isHole(err error) bool {
	var h *hole
	return errors.As(err, &h)
}

// holed is the table of keyTable with some of its pairs unknown, each a source and an action.
func holed(tb *umpire.Table, pairs ...[2]string) *umpire.Table {
	spec := keyCopy(tb)
	for _, p := range pairs {
		if !slices.Contains(spec.Actions, p[1]) {
			spec.Actions = append(spec.Actions, p[1])
		}
		spec.Unknown = append(spec.Unknown, umpire.UnknownPair{Row: p[0] + "-" + p[1], Source: p[0], Action: p[1],
			Cause: &hole{p[0] + "-" + p[1]}})
	}
	return umpire.NewTable(spec)
}

func pathOf(tb *umpire.Table, start string, steps ...[2]string) *umpire.Trace {
	w := &umpire.Trace{Initial: tb.StateAtom(start)}
	for _, s := range steps {
		w.Steps = append(w.Steps, umpire.TraceStep{Action: tb.ActionAtom(s[0]), Outcome: tb.OutcomeAtom("ok"),
			State: tb.StateAtom(s[1]), Facts: []umpire.Atom{}})
	}
	return w
}

func forkedChain(pairs ...[2]string) *umpire.Table {
	return holed(keyTable("chain", []string{"s0"},
		[3]string{"s0", "go", "s1"}, [3]string{"s0", "slip", "bad"}, [3]string{"s1", "stay", "s1"}), pairs...)
}

func neverBad(tb *umpire.Table) *umpire.PropertyDecl {
	return umpire.KeyTransitionProperty(tb, "neverBad",
		func(_ string, s umpire.Result) (bool, error) { return s.State != "bad", nil })
}

func TestAnExploredUnknownPairMakesAVerifyIncomplete(t *testing.T) {
	tb := forkedChain([2]string{"s1", "leap"})
	holds := umpire.KeyTransitionProperty(tb, "holds", always)
	q := umpire.KeyVerify("q", holds, umpire.KeyFreeScenario(tb, "anything", "s0"), four)
	a, err := q.Answer()
	require.NoError(t, err)
	require.Equal(t, umpire.VerifiedWithinLimits, a.Outcome, "the outcome keeps its spelling")
	require.True(t, a.Incomplete(), "what lies behind the hole is not verified")
	require.Equal(t, []umpire.UnknownReach{{Kind: umpire.UnknownRow, Row: "s1-leap", Source: "s1", Action: "leap",
		Depth: 1, Prefix: pathOf(tb, "s0", [2]string{"go", "s1"}), Cause: &hole{"s1-leap"}}}, a.Unknown)
	require.Equal(t, 3, a.Expanded)
	require.NoError(t, tb.Replay(a.Unknown[0].Prefix))
	require.Equal(t, []string{"s0", "s1", "bad"}, tb.Reachable, "reachability reads rows only")
}

func TestUnexploredUnknownPairsChangeNothing(t *testing.T) {
	tb := forkedChain([2]string{"s1", "leap"})
	holds := func(name string) *umpire.PropertyDecl { return umpire.KeyTransitionProperty(tb, name, always) }
	reachesS1 := umpire.KeyProperty(tb, "reachesS1", nil, "",
		func(s umpire.Result) (bool, error) { return s.State == "s1", nil })
	one := umpire.Limits{Name: "one", Steps: 1, Actions: 4, Search: 64}
	for _, c := range []struct {
		name string
		q    *umpire.Query
		want umpire.Outcome
	}{
		{"a pair at the step bound",
			umpire.KeyVerify("q", holds("bounded"), umpire.KeyFreeScenario(tb, "bounded", "s0"), one),
			umpire.VerifiedWithinLimits},
		{"a pair the pinned Scenario does not schedule",
			umpire.KeyVerify("q", holds("pinned"), umpire.KeyScenario(tb, "pinned", "s0", "go", "stay"), four),
			umpire.VerifiedWithinLimits},
		{"a pair behind a found witness",
			umpire.KeyFind("q", reachesS1, umpire.KeyFreeScenario(tb, "found", "s0"), four), umpire.Found},
	} {
		t.Run(c.name, func(t *testing.T) {
			a, err := c.q.Answer()
			require.NoError(t, err)
			require.Equal(t, c.want, a.Outcome)
			require.Empty(t, a.Unknown)
			require.False(t, a.Incomplete())
		})
	}
}

func TestAnUnrelatedUnknownDoesNotDowngradeACounterexample(t *testing.T) {
	tb := forkedChain([2]string{"s1", "leap"})
	q := umpire.KeyVerify("q", neverBad(tb), umpire.KeyFreeScenario(tb, "anything", "s0"), four)
	a, err := q.Answer()
	require.NoError(t, err)
	require.Equal(t, umpire.CounterexampleFound, a.Outcome)
	require.Equal(t, []string{"s0-slip"}, a.Rows)
	require.False(t, a.Incomplete(), "a violation stands whatever is unknown")
	require.Equal(t, []umpire.UnknownReach{{Kind: umpire.UnknownRow, Row: "s1-leap", Source: "s1", Action: "leap",
		Depth: 1, Prefix: pathOf(tb, "s0", [2]string{"go", "s1"}), Cause: &hole{"s1-leap"}}}, a.Unknown)
	require.NoError(t, q.Replay(a))
	require.NoError(t, tb.Replay(a.Witness))
}

func TestADisabledPairIsNeverUnknown(t *testing.T) {
	tb := forkedChain()
	require.NoError(t, tb.Err())
	require.Empty(t, tb.UnknownFrom("s1"), "s1 takes neither go nor slip, and neither is unknown")
	a, err := umpire.KeyVerify("q", umpire.KeyTransitionProperty(tb, "holds", always),
		umpire.KeyFreeScenario(tb, "anything", "s0"), four).Answer()
	require.NoError(t, err)
	require.Equal(t, umpire.VerifiedWithinLimits, a.Outcome)
	require.Empty(t, a.Unknown)
	require.False(t, a.Incomplete())
}

func TestAClassifiedClaimErrorIsAnUnknownEdge(t *testing.T) {
	tb := forkedChain()
	unreadAtS1 := func(name string) *umpire.PropertyDecl {
		return umpire.KeyTransitionProperty(tb, name, func(_ string, s umpire.Result) (bool, error) {
			if s.State == "s1" {
				return false, &hole{"the claim"}
			}
			return true, nil
		})
	}
	q := umpire.KeyVerify("q", unreadAtS1("classified"), umpire.KeyFreeScenario(tb, "classified", "s0"), four)
	q.Unknown = isHole
	a, err := q.Answer()
	require.NoError(t, err)
	require.Equal(t, umpire.VerifiedWithinLimits, a.Outcome)
	require.True(t, a.Incomplete())
	require.Equal(t, 2, a.Explored, "the search does not continue into s1")
	require.Len(t, a.Unknown, 1)
	reach := a.Unknown[0]
	require.ErrorAs(t, reach.Cause, new(*hole))
	reach.Cause = nil
	require.Equal(t, umpire.UnknownReach{Kind: umpire.UnknownClaim, Row: "s0-go", Source: "s0", Action: "go",
		Prefix: pathOf(tb, "s0")}, reach)
	require.NoError(t, tb.Replay(reach.Prefix))

	unclassified := umpire.KeyVerify("q", unreadAtS1("unclassified"), umpire.KeyFreeScenario(tb, "unclassified", "s0"), four)
	_, err = unclassified.Answer()
	require.EqualError(t, err, "query q: a hole at the claim")
	require.ErrorAs(t, err, new(*hole))

	fails := umpire.KeyVerify("q", neverBad(tb), umpire.KeyFreeScenario(tb, "fails", "s0"), four).
		Watch(umpire.KeyMonitor("unread", "m", func(mon, _ string, s umpire.Result) (string, error) {
			if s.State == "s1" {
				return "", &hole{"the monitor"}
			}
			return mon, nil
		}, func(string) (bool, error) { return false, nil }, umpire.EveryStep()))
	fails.Unknown = isHole
	a, err = fails.Answer()
	require.NoError(t, err)
	require.Equal(t, umpire.CounterexampleFound, a.Outcome, "the claim fails on the other branch")
	require.Equal(t, []string{"s0-slip"}, a.Rows)
	require.Len(t, a.Unknown, 1)
	require.NoError(t, fails.Replay(a))
}

func TestCallbackErrorsKeepTheirType(t *testing.T) {
	tb := forkedChain()
	failing := func(at string) error { return &hole{at} }
	free := func(name string) *umpire.ScenarioDecl { return umpire.KeyFreeScenario(tb, name, "s0") }
	watching := func(name string, m *umpire.Monitor) *umpire.Query {
		return umpire.KeyVerify("q", umpire.KeyTransitionProperty(tb, name, always), free(name), four).Watch(m)
	}
	stays := func(mon, _ string, _ umpire.Result) (string, error) { return mon, nil }
	unviolated := func(string) (bool, error) { return false, nil }
	for at, q := range map[string]*umpire.Query{
		"holds": umpire.KeyVerify("q", umpire.KeyProperty(tb, "holds", nil, "",
			func(umpire.Result) (bool, error) { return false, failing("holds") }), free("holds"), four),
		"holds across": umpire.KeyVerify("q", umpire.KeyTransitionProperty(tb, "across",
			func(string, umpire.Result) (bool, error) { return false, failing("holds across") }), free("across"), four),
		"next": watching("next", umpire.KeyMonitor("m", "m",
			func(string, string, umpire.Result) (string, error) { return "", failing("next") }, unviolated, umpire.EveryStep())),
		"violated": watching("violated", umpire.KeyMonitor("m", "m", stays,
			func(string) (bool, error) { return false, failing("violated") }, umpire.EveryStep())),
		"after": watching("after", umpire.KeyMonitor("m", "m", stays, unviolated,
			umpire.AfterKey(func(umpire.Result) (bool, error) { return false, failing("after") }))),
	} {
		_, err := q.Answer()
		require.EqualError(t, err, "query q: a hole at "+at)
		var h *hole
		require.ErrorAs(t, err, &h, at)
		require.Equal(t, &hole{at}, h)
	}

	yes := func(string) (bool, error) { return true, nil }
	for at, p := range map[string]*umpire.Progress{
		"from": umpire.KeyProgressFunc("p", func(string) (bool, error) { return false, failing("from") }, yes, 1),
		"to":   umpire.KeyProgressFunc("p", yes, func(string) (bool, error) { return false, failing("to") }, 1),
	} {
		_, err := umpire.CheckProgress(tb, p, wide)
		var h *hole
		require.ErrorAs(t, err, &h, at)
		require.Equal(t, &hole{at}, h)
	}
}

func TestProgressReadsUnknownPairs(t *testing.T) {
	finishes := umpire.KeyProgress("finishes", is("waiting"), is("done"), 3)

	stalls := keyTable("stalls", []string{"waiting"},
		[3]string{"waiting", "fail", "stuck"}, [3]string{"waiting", "finish", "done"})
	a, err := umpire.CheckProgress(stalls, finishes, wide)
	require.NoError(t, err)
	requireVerdict(t, umpire.CounterexampleFound, a.Deadlock)
	require.False(t, a.Incomplete())

	retries := holed(stalls, [2]string{"stuck", "retry"})
	a, err = umpire.CheckProgress(retries, finishes, wide)
	require.NoError(t, err)
	requireVerdict(t, umpire.VerifiedWithinLimits, a.Deadlock)
	require.True(t, a.Incomplete(), "a state whose only pair is unknown is no deadlock, and none is ruled out")
	require.Equal(t, []umpire.UnknownReach{{Kind: umpire.UnknownRow, Row: "stuck-retry", Source: "stuck", Action: "retry",
		Depth: 1, Prefix: pathOf(retries, "waiting", [2]string{"fail", "stuck"}), Cause: &hole{"stuck-retry"}}}, a.Unknown)
	require.NoError(t, retries.Replay(a.Unknown[0].Prefix))
	forged := umpire.ProgressVerdict{Outcome: umpire.CounterexampleFound, Witness: a.Unknown[0].Prefix, Loop: -1}
	require.EqualError(t, finishes.Replay(retries, umpire.DeadlockKind, forged),
		"progress finishes: the witness ends at 'stuck', where a pair is unknown, so it may have a step")

	polls := holed(keyTable("polls", []string{"waiting"},
		[3]string{"waiting", "poll", "waiting"}, [3]string{"waiting", "finish", "done"}), [2]string{"done", "reopen"})
	finishIsFair := umpire.Assumption{Name: "finishIsFair", Fair: []string{"finish"}}
	soon := umpire.KeyProgress("finishes", is("waiting"), is("done"), 2, finishIsFair)
	a, err = umpire.CheckProgress(polls, soon, wide)
	require.NoError(t, err)
	requireVerdict(t, umpire.CounterexampleFound, a.Deadline)
	require.NoError(t, soon.Replay(polls, umpire.DeadlineKind, a.Deadline), "a deadline is missed on rows alone")
	require.Len(t, a.Unknown, 1)

	bounces := keyTable("bounces", []string{"a"},
		[3]string{"a", "finish", "done"}, [3]string{"a", "step", "b"}, [3]string{"b", "back", "a"})
	eventually := umpire.KeyProgress("finishes", is("a"), is("done"), 4, finishIsFair)
	a, err = umpire.CheckProgress(bounces, eventually, wide)
	require.NoError(t, err)
	requireVerdict(t, umpire.CounterexampleFound, a.Cycle)
	unfair := a.Cycle

	maybe := holed(bounces, [2]string{"b", "finish"})
	a, err = umpire.CheckProgress(maybe, eventually, wide)
	require.NoError(t, err)
	requireVerdict(t, umpire.VerifiedWithinLimits, a.Cycle)
	require.True(t, a.Incomplete(), "finish may be enabled at b, so the cycle that never takes it may be unfair")
	require.EqualError(t, eventually.Replay(maybe, umpire.CycleKind, unfair), "progress finishes: finish may stay enabled "+
		"on the cycle, where a pair of it is unknown, and is never taken, which finishIsFair forbids")
}

func TestMalformedUnknownPairsAreRejected(t *testing.T) {
	base := keyCopy(keyTable("chain", []string{"s0"}, [3]string{"s0", "go", "s1"}))
	pair := func(source, action string) umpire.UnknownPair {
		return umpire.UnknownPair{Row: source + "-" + action, Source: source, Action: action}
	}
	for _, c := range []struct {
		want    string
		unknown []umpire.UnknownPair
	}{
		{"chain: the pair 's0-go' is unknown and is a row; a pair is one or the other", []umpire.UnknownPair{pair("s0", "go")}},
		{"chain: the unknown pair 'other' is at 's0' and takes go, so its key is 's0-go'",
			[]umpire.UnknownPair{{Row: "other", Source: "s0", Action: "go"}}},
		{"chain: the pair 's1-go' is unknown twice", []umpire.UnknownPair{pair("s1", "go"), pair("s1", "go")}},
		{"chain: the unknown pair 's9-go' is at 's9', which is not a state", []umpire.UnknownPair{pair("s9", "go")}},
		{"chain: the unknown pair 's1-fly' takes fly, which is not an action class", []umpire.UnknownPair{pair("s1", "fly")}},
	} {
		spec := base
		spec.Unknown = c.unknown
		tb := umpire.NewTable(spec)
		require.EqualError(t, tb.Err(), c.want)
		require.ErrorAs(t, tb.Err(), new(*umpire.Error))

		_, err := tb.Model().Table()
		require.Equal(t, tb.Err(), err)
		_, err = umpire.KeyVerify("q", umpire.KeyTransitionProperty(tb, "holds", always),
			umpire.KeyFreeScenario(tb, "anything", "s0"), four).Answer()
		require.Equal(t, tb.Err(), err)
		_, err = umpire.CheckProgress(tb, umpire.KeyProgress("p", is("s0"), is("s1"), 1), wide)
		require.Equal(t, tb.Err(), err)
		_, err = umpire.RefineTables(tb, tb, umpire.RefinementSpec{MapState: func(s string) (string, error) { return s, nil }})
		require.Equal(t, tb.Err(), err)
	}
	require.NoError(t, umpire.NewTable(base).Err())
}

func TestKeyLevelClaimNamesAreScopedPerTable(t *testing.T) {
	first, second := forkedChain(), forkedChain()
	query := func(tb *umpire.Table) *umpire.Query {
		return umpire.KeyVerify("q", umpire.KeyTransitionProperty(tb, "holds", always),
			umpire.KeyFreeScenario(tb, "anything", "s0"), four)
	}
	one, other := query(first), query(second)
	for _, q := range []*umpire.Query{one, other} {
		a, err := q.Answer()
		require.NoError(t, err, "one name on two tables is two claims")
		require.Equal(t, umpire.VerifiedWithinLimits, a.Outcome)
	}
	require.Equal(t, "test.progress.property.holds", one.Property.PropertyID(first))
	require.Equal(t, "test.progress.behavior.anything", one.Scenario.ScenarioID(first))

	again := umpire.KeyTransitionProperty(first, "holds", always)
	for _, q := range []*umpire.Query{one, umpire.KeyVerify("q", again, umpire.KeyFreeScenario(first, "elsewhere", "s0"), four)} {
		_, err := q.Answer()
		require.EqualError(t, err, "machine chain: property holds is declared twice, and both declarations would share "+
			"one Definition ID; rename one")
	}
	_, err := umpire.KeyVerify("q", umpire.KeyTransitionProperty(first, "fresh", always),
		umpire.KeyFreeScenario(first, "elsewhere", "s0"), four).Answer()
	require.EqualError(t, err, "machine chain: scenario elsewhere is declared twice, and both declarations would share "+
		"one Definition ID; rename one")
	require.ErrorContains(t, umpire.Check(first.Model()), "property holds is declared twice")

	_, err = umpire.KeyVerify("q", umpire.KeyTransitionProperty(first, "mine", always),
		umpire.KeyFreeScenario(second, "theirs", "s0"), four).Answer()
	require.EqualError(t, err, "query q: mine is declared on chain, but theirs runs on chain, which does not refine it")

	_, err = umpire.KeyVerify("q", umpire.KeyTransitionProperty(first, "nothing", nil),
		umpire.KeyFreeScenario(first, "nowhere", "s0"), four).Answer()
	require.EqualError(t, err, "query q: nothing names no function that says whether it holds")
}
