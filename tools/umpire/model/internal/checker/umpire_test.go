package checker_test

import (
	"slices"
	"testing"

	"github.com/stretchr/testify/require"
	umpire "go.temporal.io/server/tools/umpire/model/internal/checker"
)

// A toy door: closed, open or locked, with a knob that turns by one of two hands.
//
// Its table is given by its keys, as the model reader gives one: a state is its phase and whether it
// is oiled, an action class is the action and its hand (the left one, or the right one weakly or
// strongly), the classes are sorted by key and the rows run states-major. Only a strong right hand
// turns a closed door open, recording that it opened and that it creaked unless oiled; a push shuts
// an open door, and a lock locks a closed one.

// rowOf is the row of a state and an action class, keyed as the table keys it.
func rowOf(source, action string, results ...umpire.Result) umpire.Row {
	return umpire.Row{Key: source + "-" + action, Source: source, Action: action, Results: results}
}

// resultOf is one result of a row: its outcome, the state it reaches and the facts it records.
func resultOf(outcome, state string, facts ...string) umpire.Result {
	return umpire.Result{Outcome: outcome, State: state, Facts: append([]string{}, facts...)}
}

// tableFrom builds a table from a spec, each extension changing the spec first.
func tableFrom(spec umpire.TableSpec, extend ...func(*umpire.TableSpec)) *umpire.Table {
	for _, e := range extend {
		e(&spec)
	}
	return umpire.NewTable(spec)
}

func doorSpec(name string) umpire.TableSpec {
	return umpire.TableSpec{Machine: name, Family: "test.door",
		States:      []string{"closed-false", "closed-true", "open-false", "open-true", "locked-false", "locked-true"},
		Actions:     []string{"lock", "push", "turn-left", "turn-right-false", "turn-right-true"},
		Outcomes:    []string{"ok"},
		Facts:       []string{"opened", "creaked-false", "creaked-true", "unlocked"},
		Starts:      []string{"closed-false"},
		Ends:        []string{"locked-false", "locked-true"},
		StateFields: []string{"phase", "oiled"},
		Evidence:    [][2]string{{"opened", "opened"}, {"creaked", "creaked"}},
		Rows: []umpire.Row{
			rowOf("closed-false", "lock", resultOf("ok", "locked-false")),
			rowOf("closed-false", "turn-right-true", resultOf("ok", "open-false", "opened", "creaked-true")),
			rowOf("closed-true", "lock", resultOf("ok", "locked-true")),
			rowOf("closed-true", "turn-right-true", resultOf("ok", "open-true", "opened", "creaked-false")),
			rowOf("open-false", "push", resultOf("ok", "closed-false")),
			rowOf("open-true", "push", resultOf("ok", "closed-true")),
		}}
}

// doorTable is the door's table under a name.
func doorTable(name string, extend ...func(*umpire.TableSpec)) *umpire.Table {
	return tableFrom(doorSpec(name), extend...)
}

// walk is the witness a table's rows spell from a start, each row taken by its first result.
func walk(tb *umpire.Table, start string, rows ...string) *umpire.Trace {
	w := &umpire.Trace{Initial: tb.StateAtom(start)}
	for _, key := range rows {
		for _, r := range tb.Rows {
			if r.Key != key {
				continue
			}
			res := r.Results[0]
			facts := []umpire.Atom{}
			for _, f := range res.Facts {
				facts = append(facts, tb.FactAtom(f))
			}
			w.Steps = append(w.Steps, umpire.TraceStep{Action: tb.ActionAtom(r.Action), Outcome: tb.OutcomeAtom(res.Outcome),
				State: tb.StateAtom(res.State), Facts: facts})
		}
	}
	return w
}

// A table keeps the order its spec gives its catalogs and rows, and computes what it reaches from its
// starts by sweeping the rows in that order.
func TestTableOrdersActionsByKeyAndRowsStatesMajor(t *testing.T) {
	tb := doorTable("door")
	require.NoError(t, tb.Err())
	require.Equal(t, []string{"lock", "push", "turn-left", "turn-right-false", "turn-right-true"}, tb.Actions)
	var rows []string
	for _, r := range tb.Rows {
		rows = append(rows, r.Key)
	}
	require.Equal(t, []string{"closed-false-lock", "closed-false-turn-right-true", "closed-true-lock",
		"closed-true-turn-right-true", "open-false-push", "open-true-push"}, rows)
	require.Equal(t, []string{"closed-false", "locked-false", "open-false"}, tb.Reachable)
	require.Equal(t, []string{"locked-false", "locked-true"}, tb.Ends)
	require.Equal(t, []string{"opened", "creaked-false", "creaked-true", "unlocked"}, tb.Facts)
	require.Equal(t, "test.door.state.door.closed-false", tb.IDs().States[0])
}

// A two-phase abstraction of the door: shut or open.

// abstractTable is the abstract door: a turn opens it, recording that it opened when opensFact is
// set, and a push shuts it.
func abstractTable(opensFact bool, extend ...func(*umpire.TableSpec)) *umpire.Table {
	var facts []string
	if opensFact {
		facts = []string{"opened"}
	}
	return tableFrom(umpire.TableSpec{Machine: "abstract", Family: "test.door", States: []string{"shut", "ajar"},
		Actions: []string{"push", "turn"}, Outcomes: []string{"ok"}, Facts: []string{"opened"}, Starts: []string{"shut"},
		StateFields: []string{"phase"},
		Rows: []umpire.Row{
			rowOf("shut", "turn", resultOf("ok", "ajar", facts...)),
			rowOf("ajar", "push", resultOf("ok", "shut")),
		}}, extend...)
}

// refiningAbstract gives the door the state field that reads it as the abstract door.
func refiningAbstract(s *umpire.TableSpec) {
	s.StateFields = append(append([]string{}, s.StateFields...), "abstract")
	s.RefinedField = "abstract"
}

// readsAbstract reads the door as the abstract one.
var readsAbstract = umpire.RefinementSpec{MapState: abstractKeyOf}

func TestRefinementMatchesStepsAndStutters(t *testing.T) {
	m := doorTable("concrete", refiningAbstract)
	require.NoError(t, m.Err())
	ref, err := umpire.RefineTables(m, abstractTable(true), readsAbstract)
	require.NoError(t, err)
	byKey := map[string]*string{}
	for _, r := range ref.Rows {
		byKey[r.Key] = r.Product
	}
	require.Equal(t, "turn", *byKey["closed-false-turn-right-true"], "a turn is the abstract turn, by name")
	require.Equal(t, "push", *byKey["open-false-push"])
	require.Nil(t, byKey["closed-false-lock"], "locking stays shut: a stutter")
	require.Contains(t, m.StateFields, "abstract")
	require.EqualError(t, doorTable("concrete", func(s *umpire.TableSpec) { s.RefinedField = "abstract" }).Err(),
		"concrete: the refined field abstract is not a state field")
}

func TestRefinementRejectsARowWithNoProductStep(t *testing.T) {
	abstract := abstractTable(true, func(s *umpire.TableSpec) { s.Rows = nil })
	_, err := umpire.RefineTables(doorTable("concrete"), abstract, readsAbstract)
	require.ErrorContains(t, err, "the row 'closed-false-turn-right-true' steps from 'closed-false' to 'open-false', "+
		"which read as 'shut' and 'ajar' in abstract; abstract has no step")
}

// silentDoor turns open by any hand, and records nothing.
func silentDoor() *umpire.Table {
	return tableFrom(umpire.TableSpec{Machine: "silent", Family: "test.door", States: doorSpec("").States,
		Actions: []string{"turn-left", "turn-right-false", "turn-right-true"}, Outcomes: []string{"ok"},
		Facts: doorSpec("").Facts, Starts: []string{"closed-false"}, StateFields: []string{"phase", "oiled"},
		Rows: []umpire.Row{
			rowOf("closed-false", "turn-left", resultOf("ok", "open-false")),
			rowOf("closed-false", "turn-right-false", resultOf("ok", "open-false")),
			rowOf("closed-false", "turn-right-true", resultOf("ok", "open-false")),
			rowOf("closed-true", "turn-left", resultOf("ok", "open-true")),
			rowOf("closed-true", "turn-right-false", resultOf("ok", "open-true")),
			rowOf("closed-true", "turn-right-true", resultOf("ok", "open-true")),
		}})
}

func TestRefinementRejectsAProductFactTheRowDoesNotRecord(t *testing.T) {
	// The abstract turn records opened; a concrete turn that records nothing is not that step.
	_, err := umpire.RefineTables(silentDoor(), abstractTable(true), readsAbstract)
	require.ErrorContains(t, err, "neither a step of abstract nor a stutter")
}

// opensFirst starts open and only pushes.
func opensFirst() *umpire.Table {
	return tableFrom(umpire.TableSpec{Machine: "opensFirst", Family: "test.door", States: doorSpec("").States,
		Actions: []string{"push"}, Outcomes: []string{"ok"}, Facts: doorSpec("").Facts, Starts: []string{"open-false"},
		StateFields: []string{"phase", "oiled"},
		Rows: []umpire.Row{
			rowOf("open-false", "push", resultOf("ok", "closed-false")),
			rowOf("open-true", "push", resultOf("ok", "closed-true")),
		}})
}

func TestRefinementRejectsAStartTheProductDoesNotHave(t *testing.T) {
	_, err := umpire.RefineTables(opensFirst(), abstractTable(true), readsAbstract)
	require.EqualError(t, err, "opensFirst refines abstract: opensFirst starts at 'open-false', which reads as "+
		"'ajar', and abstract does not start there")
}

// Queries on the door.

var (
	two  = umpire.Limits{Name: "two", Steps: 2, Actions: 2, Search: 64}
	tiny = umpire.Limits{Name: "tiny", Steps: 2, Actions: 2, Search: 1}
)

// opensLoudlyOn is the claim that a strong right turn opens the door.
func opensLoudlyOn(tb *umpire.Table) *umpire.PropertyDecl {
	return umpire.KeyProperty(tb, "opensLoudly", on("turn-right-true"), "turn-right-true",
		func(s umpire.Result) (bool, error) { return phaseOf(s.State) == "open", nil })
}

// staysShutOn is the claim that no step opens the door.
func staysShutOn(tb *umpire.Table) *umpire.PropertyDecl {
	return umpire.KeyTransitionProperty(tb, "staysShut",
		func(_ string, s umpire.Result) (bool, error) { return phaseOf(s.State) != "open", nil })
}

func TestFindReturnsTheShortestWitness(t *testing.T) {
	m := doorTable("door")
	path := umpire.KeyScenario(m, "turnThenPush", "closed-false", "turn-right-true", "push")
	a, err := umpire.KeyFind("q", opensLoudlyOn(m), path, two).Answer()
	require.NoError(t, err)
	require.Equal(t, umpire.Found, a.Outcome)
	require.Equal(t, []string{"closed-false-turn-right-true", "open-false-push"}, a.Rows)
	require.Equal(t, "test.door.action.door.turn-right-true", a.Witness.Steps[0].Action.ID)
	require.Equal(t, []umpire.Atom{{ID: "test.door.fact.door.opened", Value: "opened"},
		{ID: "test.door.fact.door.creaked-true", Value: "creaked-true"}}, a.Witness.Steps[0].Facts)
}

func TestFindReportsNotFoundAndAWrongPathIsRejected(t *testing.T) {
	m := doorTable("door")
	never := umpire.KeyProperty(m, "never", on("push"), "push",
		func(s umpire.Result) (bool, error) { return phaseOf(s.State) == "open", nil })
	path := umpire.KeyScenario(m, "turnThenPush", "closed-false", "turn-right-true", "push")
	a, err := umpire.KeyFind("q", never, path, two).Answer()
	require.NoError(t, err)
	require.Equal(t, umpire.NotFound, a.Outcome)

	weak := umpire.KeyScenario(m, "weakTurn", "closed-false", "turn-left")
	a, err = umpire.KeyFind("q", never, weak, two).Answer()
	require.NoError(t, err)
	require.Equal(t, umpire.NotFound, a.Outcome, "a pinned action with no row admits no trace")

	long := umpire.KeyScenario(m, "long", "closed-false", "turn-right-true", "push", "lock")
	_, err = umpire.KeyFind("q", never, long, two).Answer()
	require.EqualError(t, err, "query q: long pins 3 actions and the limits two allow 2")
}

func TestVerifyFindsACounterexampleOrVerifies(t *testing.T) {
	m := doorTable("door")
	free := umpire.KeyFreeScenario(m, "anything", "closed-false")
	a, err := umpire.KeyVerify("q", staysShutOn(m), free, two).Answer()
	require.NoError(t, err)
	require.Equal(t, umpire.CounterexampleFound, a.Outcome)
	require.Equal(t, []string{"closed-false-turn-right-true"}, a.Rows)

	lockedIsFinal := umpire.KeyTransitionProperty(m, "lockedIsFinal", func(before string, after umpire.Result) (bool, error) {
		return phaseOf(before) != "locked" || phaseOf(after.State) == "locked", nil
	})
	a, err = umpire.KeyVerify("q", lockedIsFinal, free, two).Answer()
	require.NoError(t, err)
	require.Equal(t, umpire.VerifiedWithinLimits, a.Outcome)
}

func TestSearchStopsAtItsLimit(t *testing.T) {
	m := doorTable("door")
	p := umpire.KeyTransitionProperty(m, "p", always)
	a, err := umpire.KeyVerify("q", p, umpire.KeyFreeScenario(m, "anything", "closed-false"), tiny).Answer()
	require.NoError(t, err)
	require.Equal(t, umpire.LimitReached, a.Outcome)
}

func TestAFindCannotRealizeATransitionClaim(t *testing.T) {
	m := doorTable("door")
	p := umpire.KeyTransitionProperty(m, "p", always)
	_, err := umpire.KeyFind("q", p, umpire.KeyScenario(m, "s", "closed-false", "lock"), two).Answer()
	require.EqualError(t, err, "query q: find names p, a transition claim; a find realizes a same-step claim")
}

// A composition of the door with a keyholder who may lose the key.

// keyholderTable is a keyholder under a name: holding the key, it may use it, which keeps it, or lose
// it.
func keyholderTable(name string, extend ...func(*umpire.TableSpec)) *umpire.Table {
	return tableFrom(umpire.TableSpec{Machine: name, Family: "test.door", States: []string{"holding", "lost"},
		Actions: []string{"loseKey", "useKey"}, Outcomes: []string{"ok"}, Starts: []string{"holding"},
		StateFields: []string{"phase"},
		Rows: []umpire.Row{
			rowOf("holding", "loseKey", resultOf("ok", "lost")),
			rowOf("holding", "useKey", resultOf("ok", "holding")),
		}}, extend...)
}

// opaqueKeyTable is the keyholder as an opaque provider: one whose checks rely on its being opaque.
func opaqueKeyTable(extend ...func(*umpire.TableSpec)) *umpire.Table {
	return keyholderTable("opaqueKey", append([]func(*umpire.TableSpec){assumes(keyIsOpaque)}, extend...)...)
}

// assumes adds assumptions to a table's spec.
func assumes(assumptions ...umpire.Assumption) func(*umpire.TableSpec) {
	return func(s *umpire.TableSpec) { s.Assumptions = append(slices.Clone(s.Assumptions), assumptions...) }
}

func TestCompositionSynchronizesAndKeysByMember(t *testing.T) {
	tb, err := umpire.ComposeTables(houseOf(doorTable("door"), umpire.ComposeMember{Table: keyholderTable("keyholder")}))
	require.NoError(t, err)
	require.Equal(t, []string{"door_push", "door_turn-left", "door_turn-right-false", "door_turn-right-true",
		"key_loseKey", "lock"}, tb.Actions)
	require.Contains(t, tb.States, "closed-false_holding")
	require.Contains(t, tb.States, "locked-false_holding")
	for _, r := range tb.Rows {
		if r.Action == "lock" {
			require.NotContains(t, r.Source, "_lost", "a lost key cannot lock the door")
		}
	}
	require.Equal(t, "test.door.target.compose-house", tb.IDs().Target)
	require.Equal(t, []string{"door_phase", "door_oiled", "key"}, tb.StateFields)
}
