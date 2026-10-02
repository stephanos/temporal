package checker_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	umpire "go.temporal.io/server/tools/umpire/model/internal/checker"
)

// keysOf is a typed machine's table as a table of keys only. refined names the state field that
// carries the machine it refines, which a typed table knows of itself.
func keysOf(t *testing.T, m umpire.Model, refined string) *umpire.Table {
	t.Helper()
	spec := keyCopy(tableOf(t, m))
	spec.RefinedField = refined
	tb := umpire.NewTable(spec)
	require.NoError(t, tb.Err())
	return tb
}

// composition is what a composed table says, without the typed steps only a typed one carries.
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

var (
	roomy    = umpire.ComposeCeiling{States: 1 << 12, Evaluations: 1 << 16, Results: 1 << 16}
	lockSync = umpire.ComposeSync{Name: "lock", FirstMember: "door", FirstAction: "lock",
		SecondMember: "key", SecondAction: "useKey"}
)

func keyOfWornKey(state string) (string, error) { return phaseOf(state), nil }

// houseOf composes a door and a key as the typed houses of compose_test.go do.
func houseOf(doorTable *umpire.Table, key umpire.ComposeMember) umpire.ComposeSpec {
	key.Field = "key"
	return umpire.ComposeSpec{Family: "test.door", Name: "house", Ceiling: roomy,
		Members: []umpire.ComposeMember{{Field: "door", Table: doorTable}, key}, Syncs: []umpire.ComposeSync{lockSync}}
}

// replacing is a detailed key standing in for the opaque one it refines.
func replacing(t *testing.T, detailed, opaque umpire.Model) umpire.ComposeMember {
	return umpire.ComposeMember{Table: keysOf(t, detailed, opaque.Name()), Replaces: keysOf(t, opaque, ""),
		Refinement: umpire.RefinementSpec{MapState: keyOfWornKey}}
}

func TestComposeTablesMatchesTypedComposition(t *testing.T) {
	fairDoor := umpire.Assumption{Name: "doorIsFair", Fair: []string{"push", "lock"}}
	for _, c := range []struct {
		name  string
		build func(t *testing.T) (umpire.Model, umpire.ComposeSpec)
	}{
		{"a door of two starts", func(t *testing.T) (umpire.Model, umpire.ComposeSpec) {
			either := newDoor("door").Starts(door{Phase: closed}, door{Phase: closed, Oiled: true})
			return umpire.Compose[house]("test.door", "house").
					Member("door", either).Member("key", keyholder).Sync("lock", "door.lock", "key.useKey"),
				houseOf(keysOf(t, either, ""), umpire.ComposeMember{Table: keysOf(t, keyholder, "")})
		}},
		{"no synchronized step", func(t *testing.T) (umpire.Model, umpire.ComposeSpec) {
			spec := houseOf(keysOf(t, newDoor("door"), ""), umpire.ComposeMember{Table: keysOf(t, keyholder, "")})
			spec.Syncs = nil
			return umpire.Compose[house]("test.door", "house").Member("door", newDoor("door")).Member("key", keyholder), spec
		}},
		{"the states it may end in", func(t *testing.T) (umpire.Model, umpire.ComposeSpec) {
			spec := houseOf(keysOf(t, newDoor("door"), ""), umpire.ComposeMember{Table: keysOf(t, keyholder, "")})
			spec.Ends = func(_ string, parts []string) (bool, error) {
				return phaseOf(parts[0]) == "locked" && parts[1] == "holding", nil
			}
			return umpire.Compose[house]("test.door", "house").
				Member("door", newDoor("door")).Member("key", keyholder).Sync("lock", "door.lock", "key.useKey").
				Ends(func(h house) bool { return h.Door.Phase == locked && h.Key.Phase == holding }), spec
		}},
		{"an opaque key", func(t *testing.T) (umpire.Model, umpire.ComposeSpec) {
			oiled, opaque := newDoor("door").Assumes(doorIsOiled), opaqueKey()
			return umpire.Compose[house]("test.door", "house").
					Member("door", oiled).Member("key", opaque).Sync("lock", "door.lock", "key.useKey"),
				houseOf(keysOf(t, oiled, ""), umpire.ComposeMember{Table: keysOf(t, opaque, "")})
		}},
		{"a refining key in the opaque one's place", func(t *testing.T) (umpire.Model, umpire.ComposeSpec) {
			oiled, opaque := newDoor("door").Assumes(doorIsOiled), opaqueKey()
			detailed := detailedKey("detailedKey", false).Refines(opaque, keyOfWorn)
			return umpire.Compose[wornHouse]("test.door", "house").
					Member("door", oiled).Member("key", detailed).Replaces("key", opaque).Sync("lock", "door.lock", "key.useKey"),
				houseOf(keysOf(t, oiled, ""), replacing(t, detailed, opaque))
		}},
		{"a replacing key that states the opaque one's assumption", func(t *testing.T) (umpire.Model, umpire.ComposeSpec) {
			oiled, opaque := newDoor("door").Assumes(doorIsOiled), opaqueKey()
			inherits := detailedKey("detailedKey", false).Assumes(keyIsOpaque).Refines(opaque, keyOfWorn)
			return umpire.Compose[wornHouse]("test.door", "house").
					Member("door", oiled).Member("key", inherits).Replaces("key", opaque).Sync("lock", "door.lock", "key.useKey"),
				houseOf(keysOf(t, oiled, ""), replacing(t, inherits, opaque))
		}},
		{"a door that states the replaced key's assumption", func(t *testing.T) (umpire.Model, umpire.ComposeSpec) {
			relies, opaque := newDoor("door").Assumes(doorIsOiled, keyIsOpaque), opaqueKey()
			inherits := detailedKey("detailedKey", false).Assumes(keyIsOpaque).Refines(opaque, keyOfWorn)
			return umpire.Compose[wornHouse]("test.door", "house").
					Member("door", relies).Member("key", inherits).Replaces("key", opaque).Sync("lock", "door.lock", "key.useKey"),
				houseOf(keysOf(t, relies, ""), replacing(t, inherits, opaque))
		}},
		{"a member's fair classes", func(t *testing.T) (umpire.Model, umpire.ComposeSpec) {
			fair := newDoor("door").Assumes(fairDoor)
			return umpire.Compose[house]("test.door", "house").
					Member("door", fair).Member("key", keyholder).Sync("lock", "door.lock", "key.useKey"),
				houseOf(keysOf(t, fair, ""), umpire.ComposeMember{Table: keysOf(t, keyholder, "")})
		}},
	} {
		t.Run(c.name, func(t *testing.T) {
			typed, spec := c.build(t)
			got, err := umpire.ComposeTables(spec)
			require.NoError(t, err)
			require.Equal(t, compositionOf(tableOf(t, typed)), compositionOf(got))
			require.Empty(t, got.Unknown)
		})
	}
}

func TestComposeTablesKeepsEveryAssumptionOfAReplacingMember(t *testing.T) {
	opaque := opaqueKey()
	inherits := detailedKey("detailedKey", false).Assumes(keyIsOpaque).Refines(opaque, keyOfWorn)
	tb, err := umpire.ComposeTables(houseOf(keysOf(t, newDoor("door").Assumes(doorIsOiled), ""), replacing(t, inherits, opaque)))
	require.NoError(t, err)
	require.Equal(t, []umpire.Assumption{doorIsOiled, keyIsOpaque}, tb.Assumptions,
		"the replacing member declares it itself, whatever the table it replaces names")
	require.Equal(t, []string{"door_phase", "door_oiled", "key_phase", "key_worn"}, tb.StateFields,
		"the field that reads the key as the opaque one is no field of the composition")

	unnamed := replacing(t, inherits, opaque)
	unnamed.Table = keysOf(t, inherits, "")
	tb, err = umpire.ComposeTables(houseOf(keysOf(t, newDoor("door"), ""), unnamed))
	require.NoError(t, err)
	require.Contains(t, tb.StateFields, "key_opaqueKey", "a table that does not name its refined field keeps it")

	tb, err = umpire.ComposeTables(houseOf(keysOf(t, newDoor("door").Assumes(doorIsOiled, keyIsOpaque), ""),
		replacing(t, inherits, opaque)))
	require.NoError(t, err)
	require.Equal(t, []umpire.Assumption{doorIsOiled, keyIsOpaque}, tb.Assumptions,
		"the door relies on the opaque key's assumption on its own")
}

func TestComposeTablesStartsInEveryMemberStart(t *testing.T) {
	either := newDoor("door").Starts(door{Phase: closed}, door{Phase: closed, Oiled: true})
	anyKey := opaqueKey().Starts(keyState{holding}, keyState{lost})
	tb, err := umpire.ComposeTables(houseOf(keysOf(t, either, ""), umpire.ComposeMember{Table: keysOf(t, anyKey, "")}))
	require.NoError(t, err)
	require.Equal(t, []string{"closed-false_holding", "closed-false_lost", "closed-true_holding", "closed-true_lost"},
		tb.Starts, "the product of the members' starts, the last member fastest")
	typed := tableOf(t, umpire.Compose[house]("test.door", "house").
		Member("door", either).Member("key", anyKey).Sync("lock", "door.lock", "key.useKey"))
	require.Equal(t, compositionOf(typed), compositionOf(tb))
	require.Contains(t, tb.Reachable, "open-true_lost", "an oiled door opens from its own start")
}

func TestComposeTablesReplacementCoversEveryOpaqueStart(t *testing.T) {
	opaque := opaqueKey().Starts(keyState{holding}, keyState{lost})
	detailed := detailedKey("detailedKey", false).Refines(opaque, keyOfWorn)
	member := replacing(t, detailed, opaque)
	_, err := umpire.RefineTables(member.Table, member.Replaces, member.Refinement)
	require.NoError(t, err, "on its own, the refinement may start in fewer states")
	tb, err := umpire.ComposeTables(houseOf(keysOf(t, newDoor("door"), ""), member))
	require.EqualError(t, err, "detailedKey refines opaqueKey: opaqueKey starts at 'lost', which no start of detailedKey reads as")
	re := refinementError(t, err)
	require.Equal(t, umpire.RefinementInitial, re.Kind)
	require.NoError(t, member.Replaces.Replay(re.ProductWitness))
	require.Nil(t, tb)
}

func TestComposeTablesViolatingProviderFails(t *testing.T) {
	opaque := opaqueKey()
	finder := detailedKey("finderKey", true).Refines(opaque, keyOfWorn)
	member := replacing(t, finder, opaque)
	_, err := umpire.ComposeTables(houseOf(keysOf(t, newDoor("door"), ""), member))
	re := refinementError(t, err)
	require.Equal(t, umpire.RefinementUnmatched, re.Kind)
	require.ErrorContains(t, err, "finderKey refines opaqueKey: the row 'lost-false-findKey'")
	require.Equal(t, []string{"loseKey", "findKey"}, actionsOf(re.Witness))
	require.NoError(t, member.Table.Replay(re.Witness))
	require.Contains(t, member.Table.Starts, re.Witness.Initial.Value)
}

func TestComposeTablesRejectsCollidingKeys(t *testing.T) {
	heads := umpire.NewMachine[head, doorOutcome, keyFact]("test.join", "heads").Starts("a_b", "a")
	tails := umpire.NewMachine[tail, doorOutcome, keyFact]("test.join", "tails").Starts("c", "b_c")
	_, err := umpire.ComposeTables(umpire.ComposeSpec{Family: "test.join", Name: "joint", Ceiling: roomy,
		Members: []umpire.ComposeMember{{Field: "head", Table: keysOf(t, heads, "")}, {Field: "tail", Table: keysOf(t, tails, "")}}})
	require.EqualError(t, err, "compose-joint: the member states [a_b c] and [a b_c] are both keyed 'a_b_c', "+
		"so the composed key does not tell them apart")
}

func TestComposeTablesDeclarationsAreChecked(t *testing.T) {
	doorTable, keyTable := keysOf(t, newDoor("door"), ""), keysOf(t, keyholder, "")
	plain := func() umpire.ComposeSpec { return houseOf(doorTable, umpire.ComposeMember{Table: keyTable}) }
	identity := umpire.RefinementSpec{MapState: func(s string) (string, error) { return s, nil }}
	ghost := umpire.Assumption{Name: "ghostIsFair", Fair: []string{"ghost"}}
	malformed := keyCopy(keyTable)
	malformed.Unknown = []umpire.UnknownPair{{Row: "nowhere-useKey", Source: "nowhere", Action: "useKey"}}
	startless := keyCopy(keyTable)
	startless.Starts = nil
	for want, change := range map[string]func(*umpire.ComposeSpec){
		"compose-house: sync fly names door.fly, and door has no action fly": func(s *umpire.ComposeSpec) {
			s.Syncs = []umpire.ComposeSync{{Name: "fly", FirstMember: "door", FirstAction: "fly", SecondMember: "key", SecondAction: "useKey"}}
		},
		"compose-house: sync lock names a member the composition does not have": func(s *umpire.ComposeSpec) {
			s.Members = s.Members[:1]
		},
		"compose-house: the assumption ghostIsFair of door makes ghost fair, which is no action of door": func(s *umpire.ComposeSpec) {
			s.Members[0].Table = keysOf(t, newDoor("door").Assumes(ghost), "")
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
		"compose-house: the member key has no start": func(s *umpire.ComposeSpec) { s.Members[1].Table = umpire.NewTable(startless) },
		"keyholder: the unknown pair 'nowhere-useKey' is at 'nowhere', which is not a state": func(s *umpire.ComposeSpec) {
			s.Members[1].Table = umpire.NewTable(malformed)
		},
		"compose-house: the member key names a refinement, and replaces nothing": func(s *umpire.ComposeSpec) {
			s.Members[1].Refinement = identity
		},
		"compose-house: the member key replaces keyholder, and names no map that reads keyholder as it": func(s *umpire.ComposeSpec) {
			s.Members[1].Replaces = keyTable
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
	spec := houseOf(keysOf(t, newDoor("door"), ""), umpire.ComposeMember{Table: keysOf(t, keyholder, "")})
	spec.Ends = func(key string, _ []string) (bool, error) { return false, &hole{key} }
	tb, err := umpire.ComposeTables(spec)
	require.EqualError(t, err, "compose-house: a hole at closed-false_holding")
	var h *hole
	require.ErrorAs(t, err, &h)
	require.Equal(t, &hole{"closed-false_holding"}, h)
	require.Nil(t, tb)
}

func TestAComposedStepNamesItsMemberMoves(t *testing.T) {
	tb, err := umpire.ComposeTables(houseOf(keysOf(t, newDoor("door"), ""), umpire.ComposeMember{Table: keysOf(t, keyholder, "")}))
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
