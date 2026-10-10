package engine_test

import (
	"slices"
	"testing"

	"github.com/stretchr/testify/require"
	umpire "go.temporal.io/server/tools/umpire/internal/engine"
)

// startsAt gives a table these starts.
func startsAt(states ...string) func(*umpire.TableSpec) {
	return func(s *umpire.TableSpec) { s.Starts = states }
}

// composed builds a composition that is expected to fit together.
func composed(t *testing.T, spec umpire.ComposeSpec) *umpire.Table {
	t.Helper()
	tb, err := umpire.ComposeTables(spec)
	require.NoError(t, err)
	return tb
}

func TestACompositionStartsInEveryAdmittedStart(t *testing.T) {
	either := doorTable("door", startsAt("closed-false", "closed-true"))
	tb := composed(t, houseOf(either, umpire.ComposeMember{Table: keyholderTable("keyholder")}))
	require.Equal(t, []string{"closed-false_holding", "closed-true_holding"}, tb.Starts)
	require.Contains(t, tb.Reachable, "open-true_holding", "an oiled door opens from its own start")

	single := composed(t, houseOf(doorTable("door"), umpire.ComposeMember{Table: keyholderTable("keyholder")}))
	require.Equal(t, []string{"closed-false_holding"}, single.Starts)
	require.NotContains(t, single.Reachable, "open-true_holding", "no step oils the door")
}

// An opaque keyholder that may use or lose the key, and a detailed one that also wears it down.

var keyIsOpaque = umpire.Assumption{Name: "keyIsOpaque"}

// detailedKeyTable is a keyholder that wears the key down by using it, and finds a lost key when
// finds is set.
func detailedKeyTable(name string, finds bool, extend ...func(*umpire.TableSpec)) *umpire.Table {
	spec := umpire.TableSpec{Machine: name, Family: "test.door",
		States:  []string{"holding-false", "holding-true", "lost-false", "lost-true"},
		Actions: []string{"loseKey", "useKey"}, Outcomes: []string{"ok"}, Starts: []string{"holding-false"},
		StateFields: []string{"phase", "worn"},
		Rows: []umpire.Row{
			rowOf("holding-false", "loseKey", resultOf("ok", "lost-false")),
			rowOf("holding-false", "useKey", resultOf("ok", "holding-true")),
			rowOf("holding-true", "loseKey", resultOf("ok", "lost-true")),
			rowOf("holding-true", "useKey", resultOf("ok", "holding-true")),
		}}
	if finds {
		spec.Actions = append([]string{"findKey"}, spec.Actions...)
		spec.Rows = append(spec.Rows,
			rowOf("lost-false", "findKey", resultOf("ok", "holding-false")),
			rowOf("lost-true", "findKey", resultOf("ok", "holding-true")))
	}
	return tableFrom(spec, extend...)
}

// refining gives a table the state field that reads it as the machine it refines.
func refining(field string) func(*umpire.TableSpec) {
	return func(s *umpire.TableSpec) {
		s.StateFields = append(slices.Clone(s.StateFields), field)
		s.RefinedField = field
	}
}

var doorIsOiled = umpire.Assumption{Name: "doorIsOiled"}

func TestAReplacingMemberStandsInForTheOpaqueProvider(t *testing.T) {
	opaque := opaqueKeyTable()
	detailed := detailedKeyTable("detailedKey", false, refining("opaqueKey"))
	tb := composed(t, houseOf(doorTable("door", assumes(doorIsOiled)), replacing(detailed, opaque)))
	require.Equal(t, []umpire.Assumption{doorIsOiled}, tb.Assumptions, "the check relies on none of the opaque key's")
	require.Contains(t, tb.Reachable, "locked-false_holding-true")

	withOpaque := composed(t, houseOf(doorTable("door", assumes(doorIsOiled)), umpire.ComposeMember{Table: opaque}))
	require.Equal(t, []umpire.Assumption{doorIsOiled, keyIsOpaque}, withOpaque.Assumptions)
}

func TestAReplacingMemberKeepsEveryAssumptionItDeclares(t *testing.T) {
	opaque := opaqueKeyTable()
	inherits := detailedKeyTable("detailedKey", false, assumes(keyIsOpaque), refining("opaqueKey"))
	tb := composed(t, houseOf(doorTable("door", assumes(doorIsOiled)), replacing(inherits, opaque)))
	require.Equal(t, []umpire.Assumption{doorIsOiled, keyIsOpaque}, tb.Assumptions,
		"the replacing member declares it itself, whatever the machine it replaces names")
	shuts := umpire.KeyProgress("shuts", doorIs("open"), doorIs("closed"), 1)
	a, err := umpire.CheckProgress(tb, shuts, wide)
	require.NoError(t, err)
	require.Equal(t, []string{"doorIsOiled", "keyIsOpaque"}, a.Assumptions, "a check of the composition names it")

	// The replaced machine names the assumption with no fair action; the member's own declaration of
	// it, fair for the key's use, is the one the composition carries.
	keyIsFair := umpire.Assumption{Name: "keyIsOpaque", Fair: []string{"useKey"}}
	fair := detailedKeyTable("detailedKey", false, assumes(keyIsFair), refining("opaqueKey"))
	tb = composed(t, houseOf(doorTable("door"), replacing(fair, opaqueKeyTable())))
	require.Equal(t, []umpire.Assumption{{Name: "keyIsOpaque", Fair: []string{"lock"}}}, tb.Assumptions)

	tb = composed(t, houseOf(doorTable("door", assumes(doorIsOiled, keyIsOpaque)), replacing(inherits, opaqueKeyTable())))
	require.Equal(t, []umpire.Assumption{doorIsOiled, keyIsOpaque}, tb.Assumptions,
		"the door relies on the opaque key's assumption on its own")
}

func TestReplacingFairnessUsesOnlyTheProvidersDeclarationOfTheSameName(t *testing.T) {
	opaque := opaqueKeyTable(assumes(umpire.Assumption{Name: "keyIsOpaque", Fair: []string{"useKey"}}))
	for _, own := range []bool{false, true} {
		name := "without provider assumption"
		if own {
			name = "with same named provider assumption"
		}
		t.Run(name, func(t *testing.T) {
			provider := detailedKeyTable("provider", false, refining("opaqueKey"))
			want := []umpire.Assumption{{Name: "keyIsOpaque", Fair: []string{"door_push"}}}
			if own {
				provider = detailedKeyTable("provider", false, refining("opaqueKey"),
					assumes(umpire.Assumption{Name: "keyIsOpaque", Fair: []string{"loseKey"}}))
				want[0].Fair = append(want[0].Fair, "key_loseKey")
			}
			door := doorTable("door", assumes(umpire.Assumption{Name: "keyIsOpaque", Fair: []string{"push"}}))
			tb := composed(t, houseOf(door, replacing(provider, opaque)))
			require.Equal(t, want, tb.Assumptions)
			require.NotContains(t, tb.Assumptions[0].Fair, "lock", "the replaced member's useKey fairness is absent")
			a, err := umpire.CheckProgress(tb, umpire.KeyProgress("shuts", doorIs("open"), doorIs("closed"), 1), wide)
			require.NoError(t, err)
			require.Equal(t, []string{"keyIsOpaque"}, a.Assumptions)
			requireVerdict(t, umpire.VerifiedWithinLimits, a.Cycle)
		})
	}
}

func TestAMembersFairnessNamesItsComposedClasses(t *testing.T) {
	fairDoor := umpire.Assumption{Name: "doorIsFair", Fair: []string{"push", "lock"}}
	tb := composed(t, houseOf(doorTable("door", assumes(fairDoor)), umpire.ComposeMember{Table: keyholderTable("keyholder")}))
	require.Equal(t, []umpire.Assumption{{Name: "doorIsFair", Fair: []string{"door_push", "lock"}}}, tb.Assumptions)

	shuts := umpire.KeyProgress("shuts", doorIs("open"), doorIs("closed"), 1)
	a, err := umpire.CheckProgress(tb, shuts, wide)
	require.NoError(t, err)
	require.Equal(t, []string{"doorIsFair"}, a.Assumptions)
	requireVerdict(t, umpire.VerifiedWithinLimits, a.Cycle)

	ghost := umpire.Assumption{Name: "ghostIsFair", Fair: []string{"ghost"}}
	haunted := houseOf(doorTable("door", assumes(ghost)), umpire.ComposeMember{Table: keyholderTable("keyholder")})
	haunted.Name, haunted.Syncs = "haunted", nil
	_, err = umpire.ComposeTables(haunted)
	require.EqualError(t, err, "compose-haunted: the assumption ghostIsFair of door makes ghost fair, which is no action of door")
}

func TestComposedSameNamedFairnessUnionsRepeatedMembersWithoutLosingClasses(t *testing.T) {
	member := umpire.NewTable(umpire.TableSpec{Machine: "counter", Family: "test.progress",
		States: []string{"waiting", "done"}, Starts: []string{"waiting"}, Actions: []string{"finish", "poll"}, Outcomes: []string{"ok"},
		Assumptions: []umpire.Assumption{{Name: "finishes", Fair: []string{"finish"}}},
		Rows:        []umpire.Row{rowOf("waiting", "finish", resultOf("ok", "done")), rowOf("waiting", "poll", resultOf("ok", "waiting"))}})
	for _, synced := range []bool{false, true} {
		name := "independent"
		if synced {
			name = "synchronized"
		}
		t.Run(name, func(t *testing.T) {
			spec := umpire.ComposeSpec{Family: "test.progress", Name: "repeated", Ceiling: roomy,
				Members: []umpire.ComposeMember{{Field: "left", Table: member}, {Field: "right", Table: member}}}
			fair := []string{"left_finish", "right_finish"}
			if synced {
				spec.Syncs = []umpire.ComposeSync{{Name: "finishBoth", FirstMember: "left", FirstAction: "finish", SecondMember: "right", SecondAction: "finish"}}
				fair = []string{"finishBoth"}
			}
			tb := composed(t, spec)
			require.Equal(t, []umpire.Assumption{{Name: "finishes", Fair: fair}}, tb.Assumptions)
			require.Equal(t, []umpire.Assumption{{Name: "finishes", Fair: []string{"finish"}}}, member.Assumptions)
			p := umpire.KeyProgress("bothFinish", is("waiting_waiting"), is("done_done"), 2)
			a, err := umpire.CheckProgress(tb, p, wide)
			require.NoError(t, err)
			require.Equal(t, []string{"finishes"}, a.Assumptions)
			requireVerdict(t, umpire.VerifiedWithinLimits, a.Cycle)
			requireVerdict(t, umpire.CounterexampleFound, a.Deadline)
			require.NoError(t, p.Replay(tb, umpire.DeadlineKind, a.Deadline))
			unfair := umpire.ProgressVerdict{Outcome: umpire.CounterexampleFound, Loop: 0,
				Witness: tb.PathTo("waiting_waiting")}
			poll := slices.IndexFunc(tb.Rows, func(row umpire.Row) bool { return row.Source == "waiting_waiting" && row.Action == "left_poll" })
			require.NotEqual(t, -1, poll)
			row := tb.Rows[poll]
			unfair.Witness.Steps = append(unfair.Witness.Steps, umpire.TraceStep{Action: tb.ActionAtom(row.Action), Outcome: tb.OutcomeAtom(row.Results[0].Outcome), State: tb.StateAtom(row.Results[0].State)})
			require.ErrorContains(t, p.Replay(tb, umpire.CycleKind, unfair), "stays enabled on the cycle and is never taken")
		})
	}
}

func TestAViolatingProviderFailsItsReplacement(t *testing.T) {
	finder := detailedKeyTable("finderKey", true, refining("opaqueKey"))
	_, err := umpire.ComposeTables(houseOf(doorTable("door"), replacing(finder, opaqueKeyTable())))
	re := refinementError(t, err)
	require.Equal(t, umpire.RefinementUnmatched, re.Kind)
	require.ErrorContains(t, err, "finderKey refines opaqueKey: the row 'lost-false-findKey'")
	require.Equal(t, []string{"loseKey", "findKey"}, actionsOf(re.Witness))
	requireReplaysFromAStart(t, finder, re.Witness)
}

func TestAReplacementAccountsForEveryOpaqueStart(t *testing.T) {
	opaque := opaqueKeyTable(startsAt("holding", "lost"))
	detailed := detailedKeyTable("detailedKey", false, refining("opaqueKey"))
	_, err := umpire.RefineTables(detailed, opaque, umpire.RefinementSpec{MapState: keyOfWornKey})
	require.NoError(t, err, "on its own, the refinement may start in fewer states")
	tb, err := umpire.ComposeTables(houseOf(doorTable("door"), replacing(detailed, opaque)))
	require.EqualError(t, err, "detailedKey refines opaqueKey: opaqueKey starts at 'lost', which no start of detailedKey reads as")
	re := refinementError(t, err)
	require.Equal(t, umpire.RefinementInitial, re.Kind)
	require.NoError(t, opaque.Replay(re.ProductWitness))
	require.Nil(t, tb)
}

func TestAReplacementIsDeclaredAgainstARefinementOfAMember(t *testing.T) {
	unread := umpire.ComposeMember{Table: detailedKeyTable("unrelated", false), Replaces: opaqueKeyTable()}
	_, err := umpire.ComposeTables(houseOf(doorTable("door"), unread))
	require.EqualError(t, err, "compose-house: the member key replaces opaqueKey, and names no map that reads unrelated as it")
}

// Member states whose keys hold the "_" a composed key joins them with.

// joinTable is a table of no action whose state keys may hold the "_" a composed key joins with.
func joinTable(name string, states, starts []string) *umpire.Table {
	return umpire.NewTable(umpire.TableSpec{Machine: name, Family: "test.join", States: states,
		Outcomes: []string{"ok"}, Starts: starts})
}

func TestComposedKeysThatCollideAreRejected(t *testing.T) {
	heads := joinTable("heads", []string{"a", "a_b"}, []string{"a_b", "a"})
	tails := joinTable("tails", []string{"c", "b_c"}, []string{"c", "b_c"})
	_, err := umpire.ComposeTables(umpire.ComposeSpec{Family: "test.join", Name: "joint", Ceiling: roomy,
		Members: []umpire.ComposeMember{{Field: "head", Table: heads}, {Field: "tail", Table: tails}}})
	require.EqualError(t, err, "compose-joint: the member states [a_b c] and [a b_c] are both keyed 'a_b_c', "+
		"so the composed key does not tell them apart")
}
