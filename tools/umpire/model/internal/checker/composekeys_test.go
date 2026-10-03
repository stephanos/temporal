package checker_test

import (
	"crypto/sha256"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	umpire "go.temporal.io/server/tools/umpire/model/internal/checker"
)

// composition is what a composed table says, without the steps its results carry.
type composition struct {
	States, Actions, Outcomes, Facts, StateFields, Starts, Ends, Reachable []string
	Rows                                                                   []umpire.Row
	Assumptions                                                            []umpire.Assumption
	IDs                                                                    umpire.IDs
	Fingerprint                                                            string
}

func compositionOf(tb *umpire.Table) composition {
	c := composition{States: tb.States, Actions: tb.Actions, Outcomes: tb.Outcomes, Facts: tb.Facts,
		StateFields: tb.StateFields, Starts: tb.Starts, Ends: tb.Ends, Reachable: tb.Reachable,
		Assumptions: tb.Assumptions, IDs: tb.IDs(), Fingerprint: tb.TargetFingerprint()}
	for _, r := range tb.Rows {
		row := umpire.Row{Key: r.Key, Source: r.Source, Action: r.Action}
		for _, res := range r.Results {
			row.Results = append(row.Results, umpire.Result{Outcome: res.Outcome, State: res.State, Facts: res.Facts})
		}
		c.Rows = append(c.Rows, row)
	}
	return c
}

// pinned is what the typed composition of the same members said: the typed compositions are retired,
// and ComposeTables matched each of them whole before they were. The Behavior Fingerprint sorts what
// it reads, so Order holds the order of the states, actions, row keys and reachable states, which
// decides a search's order and its explored counts.
type pinned struct {
	Fingerprint, Order          string
	Starts, Ends, StateFields   []string
	Assumptions                 []umpire.Assumption
	States, Rows, ReachableSize int
}

func pinOf(tb *umpire.Table) pinned {
	c := compositionOf(tb)
	rows := make([]string, len(c.Rows))
	for i, r := range c.Rows {
		rows[i] = r.Key
	}
	order := sha256.Sum256(fmt.Appendf(nil, "%q\n%q\n%q\n%q\n", c.States, c.Actions, rows, c.Reachable))
	return pinned{Fingerprint: c.Fingerprint, Order: fmt.Sprintf("sha256:%x", order), Starts: c.Starts,
		Ends: c.Ends, StateFields: c.StateFields, Assumptions: c.Assumptions, States: len(c.States),
		Rows: len(c.Rows), ReachableSize: len(c.Reachable)}
}

var (
	roomy    = umpire.ComposeCeiling{States: 1 << 12, Evaluations: 1 << 16, Results: 1 << 16}
	lockSync = umpire.ComposeSync{Name: "lock", FirstMember: "door", FirstAction: "lock",
		SecondMember: "key", SecondAction: "useKey"}
)

func keyOfWornKey(state string) (string, error) { return phaseOf(state), nil }

// houseOf composes a door and a key, as the houses of compose_test.go are composed.
func houseOf(doorTable *umpire.Table, key umpire.ComposeMember) umpire.ComposeSpec {
	key.Field = "key"
	return umpire.ComposeSpec{Family: "test.door", Name: "house", Ceiling: roomy,
		Members: []umpire.ComposeMember{{Field: "door", Table: doorTable}, key}, Syncs: []umpire.ComposeSync{lockSync}}
}

// replacing is a detailed key standing in for the opaque one it refines.
func replacing(detailed, opaque *umpire.Table) umpire.ComposeMember {
	return umpire.ComposeMember{Table: detailed, Replaces: opaque, Refinement: umpire.RefinementSpec{MapState: keyOfWornKey}}
}

func TestComposeTablesMatchesPinnedComposition(t *testing.T) {
	fairDoor := umpire.Assumption{Name: "doorIsFair", Fair: []string{"push", "lock"}}
	plainHouse := pinned{Fingerprint: "sha256:54d1eff41b8b7d65efc9fd6cf04bb29ece24fcd29a5da624cd9d5dea12bbfcee",
		Order:  "sha256:4fe8dadb0037cfdd85f6cf51991eac2ebb06890d01d2123839d3177b239add72",
		Starts: []string{"closed-false_holding"}, StateFields: []string{"door_phase", "door_oiled", "key"},
		States: 6, Rows: 8, ReachableSize: 6}
	wornHouse := pinned{Fingerprint: "sha256:a78ef2f2aac99cba5bbee591e8785f633e5e29dd4ff9591a8740921a412d9e02",
		Order:  "sha256:bd63853073becd08ccfec73970922edebff88d2918a58fae9fffb97cc3436198",
		Starts: []string{"closed-false_holding-false"}, StateFields: []string{"door_phase", "door_oiled", "key_phase", "key_worn"},
		States: 6, Rows: 8, ReachableSize: 6}
	with := func(p pinned, change func(*pinned)) pinned {
		change(&p)
		return p
	}
	for _, c := range []struct {
		name  string
		build func() umpire.ComposeSpec
		want  pinned
	}{
		{"a door of two starts", func() umpire.ComposeSpec {
			either := doorTable("door", startsAt("closed-false", "closed-true"))
			return houseOf(either, umpire.ComposeMember{Table: keyholderTable("keyholder")})
		}, pinned{Fingerprint: "sha256:8cf225282415088a8552902223a13c26bec4145a709112fb51941ea334969d62",
			Order:  "sha256:17ebc1fbbf9024a5bfa77ffa52aa6033a964d668ef33ea0505bcfe11735430f6",
			Starts: []string{"closed-false_holding", "closed-true_holding"}, StateFields: plainHouse.StateFields,
			States: 12, Rows: 16, ReachableSize: 12}},
		{"no synchronized step", func() umpire.ComposeSpec {
			spec := houseOf(doorTable("door"), umpire.ComposeMember{Table: keyholderTable("keyholder")})
			spec.Syncs = nil
			return spec
		}, pinned{Fingerprint: "sha256:dd8889c307eca424fc05e9c04a050123a9839586a8050c69ae0a9a0c6d3c5ce1",
			Order:  "sha256:a2e1f770942bee95814d2e8d4b3d3b779a3f0d229470757109309f0077d4647c",
			Starts: plainHouse.Starts, StateFields: plainHouse.StateFields, States: 6, Rows: 12, ReachableSize: 6}},
		{"the states it may end in", func() umpire.ComposeSpec {
			spec := houseOf(doorTable("door"), umpire.ComposeMember{Table: keyholderTable("keyholder")})
			spec.Ends = func(_ string, parts []string) (bool, error) {
				return phaseOf(parts[0]) == "locked" && parts[1] == "holding", nil
			}
			return spec
		}, with(plainHouse, func(p *pinned) { p.Ends = []string{"locked-false_holding"} })},
		{"an opaque key", func() umpire.ComposeSpec {
			return houseOf(doorTable("door", assumes(doorIsOiled)), umpire.ComposeMember{Table: opaqueKeyTable()})
		}, with(plainHouse, func(p *pinned) { p.Assumptions = []umpire.Assumption{doorIsOiled, keyIsOpaque} })},
		{"a refining key in the opaque one's place", func() umpire.ComposeSpec {
			detailed := detailedKeyTable("detailedKey", false, refining("opaqueKey"))
			return houseOf(doorTable("door", assumes(doorIsOiled)), replacing(detailed, opaqueKeyTable()))
		}, with(wornHouse, func(p *pinned) { p.Assumptions = []umpire.Assumption{doorIsOiled} })},
		{"a replacing key that states the opaque one's assumption", func() umpire.ComposeSpec {
			inherits := detailedKeyTable("detailedKey", false, assumes(keyIsOpaque), refining("opaqueKey"))
			return houseOf(doorTable("door", assumes(doorIsOiled)), replacing(inherits, opaqueKeyTable()))
		}, with(wornHouse, func(p *pinned) { p.Assumptions = []umpire.Assumption{doorIsOiled, keyIsOpaque} })},
		{"a door that states the replaced key's assumption", func() umpire.ComposeSpec {
			inherits := detailedKeyTable("detailedKey", false, assumes(keyIsOpaque), refining("opaqueKey"))
			return houseOf(doorTable("door", assumes(doorIsOiled, keyIsOpaque)), replacing(inherits, opaqueKeyTable()))
		}, with(wornHouse, func(p *pinned) { p.Assumptions = []umpire.Assumption{doorIsOiled, keyIsOpaque} })},
		{"a member's fair classes", func() umpire.ComposeSpec {
			return houseOf(doorTable("door", assumes(fairDoor)), umpire.ComposeMember{Table: keyholderTable("keyholder")})
		}, with(plainHouse, func(p *pinned) {
			p.Assumptions = []umpire.Assumption{{Name: "doorIsFair", Fair: []string{"door_push", "lock"}}}
		})},
	} {
		t.Run(c.name, func(t *testing.T) {
			got, err := umpire.ComposeTables(c.build())
			require.NoError(t, err)
			require.Equal(t, c.want, pinOf(got))
			require.Empty(t, got.Unknown)
		})
	}
}

func TestComposeTablesKeepsEveryAssumptionOfAReplacingMember(t *testing.T) {
	inherits := detailedKeyTable("detailedKey", false, assumes(keyIsOpaque), refining("opaqueKey"))
	tb, err := umpire.ComposeTables(houseOf(doorTable("door", assumes(doorIsOiled)), replacing(inherits, opaqueKeyTable())))
	require.NoError(t, err)
	require.Equal(t, []umpire.Assumption{doorIsOiled, keyIsOpaque}, tb.Assumptions,
		"the replacing member declares it itself, whatever the table it replaces names")
	require.Equal(t, []string{"door_phase", "door_oiled", "key_phase", "key_worn"}, tb.StateFields,
		"the field that reads the key as the opaque one is no field of the composition")

	unnamed := replacing(detailedKeyTable("detailedKey", false, assumes(keyIsOpaque), func(s *umpire.TableSpec) {
		s.StateFields = append(s.StateFields, "opaqueKey")
	}), opaqueKeyTable())
	tb, err = umpire.ComposeTables(houseOf(doorTable("door"), unnamed))
	require.NoError(t, err)
	require.Contains(t, tb.StateFields, "key_opaqueKey", "a table that does not name its refined field keeps it")

	tb, err = umpire.ComposeTables(houseOf(doorTable("door", assumes(doorIsOiled, keyIsOpaque)),
		replacing(inherits, opaqueKeyTable())))
	require.NoError(t, err)
	require.Equal(t, []umpire.Assumption{doorIsOiled, keyIsOpaque}, tb.Assumptions,
		"the door relies on the opaque key's assumption on its own")
}

func TestComposeTablesStartsInEveryMemberStart(t *testing.T) {
	either := doorTable("door", startsAt("closed-false", "closed-true"))
	anyKey := opaqueKeyTable(startsAt("holding", "lost"))
	tb, err := umpire.ComposeTables(houseOf(either, umpire.ComposeMember{Table: anyKey}))
	require.NoError(t, err)
	require.Equal(t, []string{"closed-false_holding", "closed-false_lost", "closed-true_holding", "closed-true_lost"},
		tb.Starts, "the product of the members' starts, the last member fastest")
	require.Equal(t, pinned{Fingerprint: "sha256:938611d9b591c741e5cf64447c55ff040ce8d0c4873831ed8d522fbb88ce25bf",
		Order:       "sha256:d98b0a9e423cbf3dee017c8779059b27606e6bc9e53974679e976d1ac238c9ba",
		Starts:      []string{"closed-false_holding", "closed-false_lost", "closed-true_holding", "closed-true_lost"},
		StateFields: []string{"door_phase", "door_oiled", "key"},
		Assumptions: []umpire.Assumption{keyIsOpaque}, States: 12, Rows: 16, ReachableSize: 12}, pinOf(tb))
	require.Contains(t, tb.Reachable, "open-true_lost", "an oiled door opens from its own start")
}

func TestComposeTablesDeclarationsAreChecked(t *testing.T) {
	doors, keys := doorTable("door"), keyholderTable("keyholder")
	plain := func() umpire.ComposeSpec { return houseOf(doors, umpire.ComposeMember{Table: keys}) }
	identity := umpire.RefinementSpec{MapState: func(s string) (string, error) { return s, nil }}
	ghost := umpire.Assumption{Name: "ghostIsFair", Fair: []string{"ghost"}}
	malformed := keyCopy(keys)
	malformed.Unknown = []umpire.UnknownPair{{Row: "nowhere-useKey", Source: "nowhere", Action: "useKey"}}
	startless := keyCopy(keys)
	startless.Starts = nil
	for want, change := range map[string]func(*umpire.ComposeSpec){
		"compose-house: sync fly names door.fly, and door has no action fly": func(s *umpire.ComposeSpec) {
			s.Syncs = []umpire.ComposeSync{{Name: "fly", FirstMember: "door", FirstAction: "fly", SecondMember: "key", SecondAction: "useKey"}}
		},
		"compose-house: sync lock names a member the composition does not have": func(s *umpire.ComposeSpec) {
			s.Members = s.Members[:1]
		},
		"compose-house: the assumption ghostIsFair of door makes ghost fair, which is no action of door": func(s *umpire.ComposeSpec) {
			s.Members[0].Table = doorTable("door", assumes(ghost))
		},
		"compose-house: the ceiling allows 0 states, 1 evaluations and 1 results, and a composition is bounded by one above 0 of each": func(s *umpire.ComposeSpec) {
			s.Ceiling = umpire.ComposeCeiling{Evaluations: 1, Results: 1}
		},
		"compose-house: the ceiling allows 1 states, -1 evaluations and 1 results, and a composition is bounded by one above 0 of each": func(s *umpire.ComposeSpec) {
			s.Ceiling = umpire.ComposeCeiling{States: 1, Evaluations: -1, Results: 1}
		},
		"compose-house: the ceiling allows 1 states, 1 evaluations and 0 results, and a composition is bounded by one above 0 of each": func(s *umpire.ComposeSpec) {
			s.Ceiling = umpire.ComposeCeiling{States: 1, Evaluations: 1}
		},
		"compose-house: a member has no field":       func(s *umpire.ComposeSpec) { s.Members[1].Field = "" },
		"compose-house: two members are at door":     func(s *umpire.ComposeSpec) { s.Members[1].Field = "door" },
		"compose-house: the member key has no table": func(s *umpire.ComposeSpec) { s.Members[1].Table = nil },
		"keyholder: the table has no start":          func(s *umpire.ComposeSpec) { s.Members[1].Table = umpire.NewTable(startless) },
		"keyholder: the unknown pair 'nowhere-useKey' is at 'nowhere', which is not a state": func(s *umpire.ComposeSpec) {
			s.Members[1].Table = umpire.NewTable(malformed)
		},
		"compose-house: the member key names a refinement, and replaces nothing": func(s *umpire.ComposeSpec) {
			s.Members[1].Refinement = identity
		},
		"compose-house: the member key replaces keyholder, and names no map that reads keyholder as it": func(s *umpire.ComposeSpec) {
			s.Members[1].Replaces = keys
		},
	} {
		spec := plain()
		change(&spec)
		tb, err := umpire.ComposeTables(spec)
		require.EqualError(t, err, want)
		require.ErrorAs(t, err, new(*umpire.Error), want)
		require.Nil(t, tb)
	}
}

// toggles composes n switches, each off or on: 2^n states under n actions.
func toggles(n int, ceiling umpire.ComposeCeiling, starts ...string) umpire.ComposeSpec {
	spec := umpire.ComposeSpec{Family: "test.toggle", Name: "toggles", Ceiling: ceiling}
	for i := range n {
		spec.Members = append(spec.Members, umpire.ComposeMember{Field: fmt.Sprintf("m%02d", i),
			Table: keyTable("toggle", starts, [3]string{"off", "flip", "on"}, [3]string{"on", "flip", "off"})})
	}
	return spec
}

func limitError(t *testing.T, tb *umpire.Table, err error) *umpire.ComposeLimitError {
	t.Helper()
	require.Nil(t, tb, "a composition past its ceiling builds no table")
	var le *umpire.ComposeLimitError
	require.ErrorAs(t, err, &le)
	return le
}

func TestComposeTablesCeilingBeforeAllocation(t *testing.T) {
	one, err := umpire.ComposeTables(toggles(1, umpire.ComposeCeiling{States: 2, Evaluations: 2, Results: 2}, "off"))
	require.NoError(t, err)
	require.Len(t, one.States, 2)

	tenfold, err := umpire.ComposeTables(toggles(10, umpire.ComposeCeiling{States: 1024, Evaluations: 10240, Results: 10240}, "off"))
	require.NoError(t, err, "ten times the members fit a ceiling of exactly their states and evaluations")
	require.Len(t, tenfold.States, 1024)
	require.Len(t, tenfold.Rows, 10240)
	require.Len(t, tenfold.Reachable, 1024)

	tb, err := umpire.ComposeTables(toggles(10, umpire.ComposeCeiling{States: 1023, Evaluations: 10240, Results: 10240}, "off"))
	require.Equal(t, &umpire.ComposeLimitError{Composition: "toggles", Resource: "states", Ceiling: 1023, Needed: 1024},
		limitError(t, tb, err))
	require.EqualError(t, err, "compose-toggles: the ceiling allows 1023 states, and the composition needs at least 1024")

	tb, err = umpire.ComposeTables(toggles(10, umpire.ComposeCeiling{States: 1024, Evaluations: 10239, Results: 10240}, "off"))
	require.Equal(t, &umpire.ComposeLimitError{Composition: "toggles", Resource: "evaluations", Ceiling: 10239, Needed: 10240},
		limitError(t, tb, err))

	tb, err = umpire.ComposeTables(toggles(10, umpire.ComposeCeiling{States: 1024, Evaluations: 10240, Results: 10239}, "off"))
	require.Equal(t, &umpire.ComposeLimitError{Composition: "toggles", Resource: "results", Ceiling: 10239, Needed: 10240},
		limitError(t, tb, err))

	tb, err = umpire.ComposeTables(toggles(40, umpire.ComposeCeiling{States: 1000, Evaluations: 1 << 20, Results: 1 << 20}, "off", "on"))
	require.Equal(t, &umpire.ComposeLimitError{Composition: "toggles", Resource: "states", Ceiling: 1000, Needed: 1024},
		limitError(t, tb, err), "the product of the starts passes the ceiling at the tenth member, and is never listed")

	tb, err = umpire.ComposeTables(toggles(70, umpire.ComposeCeiling{States: 1 << 62, Evaluations: 1 << 62, Results: 1 << 62}, "off", "on"))
	le := limitError(t, tb, err)
	require.True(t, le.Overflow, "a product of starts past what a count holds is refused, not wrapped around")
	require.Equal(t, "states", le.Resource)
	require.ErrorContains(t, err, "the composition needs more than")
}

// A left member with a hole after its first step, and a right member it meets.

func holedPair() umpire.ComposeSpec {
	left := holed(keyTable("left", []string{"l0"}, [3]string{"l0", "go", "l1"}, [3]string{"l0", "meet", "l0"}),
		[2]string{"l1", "go"}, [2]string{"l1", "meet"})
	right := keyTable("right", []string{"r0"}, [3]string{"r0", "hop", "r1"}, [3]string{"r0", "meet", "r0"})
	return umpire.ComposeSpec{Family: "test.pair", Name: "pair", Ceiling: roomy,
		Members: []umpire.ComposeMember{{Field: "left", Table: left}, {Field: "right", Table: right}},
		Syncs:   []umpire.ComposeSync{{Name: "meet", FirstMember: "left", FirstAction: "meet", SecondMember: "right", SecondAction: "meet"}}}
}

func TestComposeTablesPropagatesUnknownPairs(t *testing.T) {
	tb, err := umpire.ComposeTables(holedPair())
	require.NoError(t, err)
	require.NoError(t, tb.Err())
	require.Equal(t, []string{"l0_r0", "l0_r1", "l1_r0", "l1_r1"}, tb.States, "no state lies behind an unknown pair")

	var pairs []umpire.UnknownPair
	var causes []string
	for _, u := range tb.Unknown {
		require.ErrorAs(t, u.Cause, new(*hole))
		causes = append(causes, u.Cause.Error())
		u.Cause = nil
		pairs = append(pairs, u)
	}
	require.Equal(t, []umpire.UnknownPair{
		{Row: "l1_r0-left_go", Source: "l1_r0", Action: "left_go"},
		{Row: "l1_r0-meet", Source: "l1_r0", Action: "meet"},
		{Row: "l1_r1-left_go", Source: "l1_r1", Action: "left_go"},
	}, pairs, "meeting at l1_r1 needs the right member's move, which is disabled, so it is disabled")
	require.Equal(t, []string{
		"compose-pair: the pair 'l1-go' of the member left is unknown: a hole at l1-go",
		"compose-pair: the pair 'l1-meet' of the member left is unknown: a hole at l1-meet",
		"compose-pair: the pair 'l1-go' of the member left is unknown: a hole at l1-go",
	}, causes)
	var rowKeys []string
	for _, r := range tb.Rows {
		rowKeys = append(rowKeys, r.Key)
	}
	require.Equal(t, []string{"l0_r0-left_go", "l0_r0-meet", "l0_r0-right_hop", "l0_r1-left_go", "l1_r0-right_hop"}, rowKeys,
		"a step whose other move is disabled is disabled, and an unknown pair is no row")

	holds := umpire.KeyTransitionProperty(tb, "holds", always)
	near := umpire.Limits{Name: "near", Steps: 1, Actions: 4, Search: 64}
	a, err := umpire.KeyVerify("q", holds, umpire.KeyFreeScenario(tb, "near", "l0_r0"), near).Answer()
	require.NoError(t, err)
	require.Equal(t, umpire.VerifiedWithinLimits, a.Outcome)
	require.False(t, a.Incomplete(), "one step from the start reads no state with an unknown pair")

	a, err = umpire.KeyVerify("q", holds, umpire.KeyFreeScenario(tb, "far", "l0_r0"), four).Answer()
	require.NoError(t, err)
	require.Equal(t, umpire.VerifiedWithinLimits, a.Outcome)
	require.True(t, a.Incomplete())
	var explored []string
	for _, u := range a.Unknown {
		explored = append(explored, u.Row)
		require.NoError(t, tb.Replay(u.Prefix))
	}
	require.Equal(t, []string{"l1_r0-left_go", "l1_r0-meet", "l1_r1-left_go"}, explored)
}

func TestComposeTablesEndsErrorKeepsItsType(t *testing.T) {
	spec := houseOf(doorTable("door"), umpire.ComposeMember{Table: keyholderTable("keyholder")})
	spec.Ends = func(key string, _ []string) (bool, error) { return false, &hole{key} }
	tb, err := umpire.ComposeTables(spec)
	require.EqualError(t, err, "compose-house: a hole at closed-false_holding")
	var h *hole
	require.ErrorAs(t, err, &h)
	require.Equal(t, &hole{"closed-false_holding"}, h)
	require.Nil(t, tb)
}

func TestAComposedStepNamesItsMemberMoves(t *testing.T) {
	tb, err := umpire.ComposeTables(houseOf(doorTable("door"), umpire.ComposeMember{Table: keyholderTable("keyholder")}))
	require.NoError(t, err)
	parts, ok := tb.Parts("closed-false_holding")
	require.True(t, ok)
	require.Equal(t, []string{"closed-false", "holding"}, parts)
	_, ok = tb.Parts("closed-false")
	require.False(t, ok)

	steps := map[string]any{}
	for _, r := range tb.RowsFrom("closed-false_holding") {
		steps[r.Action] = r.Results[0].Step
	}
	require.Equal(t, map[string]any{
		"door_turn-right-true": umpire.ComposedStep{Parts: []string{"open-false", "holding"},
			Moves: []umpire.MemberMove{{Member: 0, Row: "closed-false-turn-right-true"}}},
		"key_loseKey": umpire.ComposedStep{Parts: []string{"closed-false", "lost"},
			Moves: []umpire.MemberMove{{Member: 1, Row: "holding-loseKey"}}},
		"lock": umpire.ComposedStep{Parts: []string{"locked-false", "holding"},
			Moves: []umpire.MemberMove{{Member: 0, Row: "closed-false-lock"}, {Member: 1, Row: "holding-useKey"}}},
	}, steps)

	locksWithTheKey := umpire.KeyProperty(tb, "locksWithTheKey", func(a string) bool { return a == "lock" }, "lock",
		func(s umpire.Result) (bool, error) {
			step, isComposed := s.Step.(umpire.ComposedStep)
			return isComposed && len(step.Moves) == 2 && step.Parts[1] == "holding", nil
		})
	q := umpire.KeyVerify("q", locksWithTheKey, umpire.KeyFreeScenario(tb, "anything", "closed-false_holding"), four)
	a, err := q.Answer()
	require.NoError(t, err)
	require.Equal(t, umpire.VerifiedWithinLimits, a.Outcome)
	require.True(t, a.Exercised)
}
