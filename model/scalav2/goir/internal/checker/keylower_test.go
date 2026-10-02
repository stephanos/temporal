package checker_test

import (
	"slices"
	"testing"

	"github.com/stretchr/testify/require"
	umpire "go.temporal.io/server/model/scalav2/goir/internal/checker"
)

// lowerable is a table over keys with two actions, two outcomes and two facts:
//
//	s0-a → ok s1 [f]
//	s0-b → ok s2 [g], no s0 []
//	s1-a → ok s1 [f, g]
func lowerable(extend ...func(*umpire.TableSpec)) *umpire.Table {
	spec := umpire.TableSpec{Machine: "steps", Family: "test.lower", States: []string{"s0", "s1", "s2"},
		Actions: []string{"a", "b"}, Outcomes: []string{"ok", "no"}, Facts: []string{"f", "g"}, Starts: []string{"s0"},
		Rows: []umpire.Row{
			{Key: "s0-a", Source: "s0", Action: "a", Results: []umpire.Result{{Outcome: "ok", State: "s1", Facts: []string{"f"}}}},
			{Key: "s0-b", Source: "s0", Action: "b", Results: []umpire.Result{
				{Outcome: "ok", State: "s2", Facts: []string{"g"}}, {Outcome: "no", State: "s0", Facts: []string{}}}},
			{Key: "s1-a", Source: "s1", Action: "a", Results: []umpire.Result{{Outcome: "ok", State: "s1", Facts: []string{"f", "g"}}}},
		}}
	for _, e := range extend {
		e(&spec)
	}
	return umpire.NewTable(spec)
}

func on(action string) func(string) bool { return func(a string) bool { return a == action } }

func records(fact string) func(umpire.Result) (bool, error) {
	return func(r umpire.Result) (bool, error) { return slices.Contains(r.Facts, fact), nil }
}

// A Property over a table's keys lowers to the clauses a typed one does: a value is fixed where every
// accepted step carries it and the predicate rejects every accepted step with it changed.
func TestAKeyLevelPropertyLowersToItsClauses(t *testing.T) {
	state := umpire.Requirement{Label: "state-s2", Kind: umpire.StateRequirement, Value: "s2"}
	outcome := umpire.Requirement{Label: "outcome-ok", Kind: umpire.OutcomeRequirement, Value: "ok"}
	fact := umpire.Requirement{Label: "fact-f", Kind: umpire.FactRequirement, Value: "f"}
	for _, c := range []struct {
		name  string
		when  func(string) bool
		holds func(umpire.Result) (bool, error)
		want  []umpire.Group
	}{
		{"a fact every accepted step records", on("a"), records("f"),
			[]umpire.Group{{Trigger: "a", Requirements: []umpire.Requirement{fact}}}},
		{"one state and one outcome", on("b"), func(r umpire.Result) (bool, error) { return r.State == "s2" && r.Outcome == "ok", nil },
			[]umpire.Group{{Trigger: "b", Requirements: []umpire.Requirement{state, outcome}}}},
		{"every action it is about", nil, func(r umpire.Result) (bool, error) { return r.Outcome == "ok", nil },
			[]umpire.Group{{Trigger: "a", Requirements: []umpire.Requirement{outcome}}, {Trigger: "b", Requirements: []umpire.Requirement{outcome}}}},
	} {
		t.Run(c.name, func(t *testing.T) {
			groups, err := umpire.KeyProperty(lowerable(), "p", c.when, "", c.holds).Lower()
			require.NoError(t, err)
			require.Equal(t, c.want, groups)
		})
	}
}

// A predicate that cannot be read on a step is not lowered as if it were false there: the Property is
// not lowered, and the error is the predicate's own, whether the step is one of the table's or one
// the lowering asks about with a value changed.
func TestAKeyLevelPropertyThatCannotBeReadIsNotLowered(t *testing.T) {
	for _, c := range []struct {
		name  string
		holds func(umpire.Result) (bool, error)
		at    string
	}{
		{"on a step of the table", func(r umpire.Result) (bool, error) {
			if slices.Contains(r.Facts, "g") {
				return false, &hole{"s1-a"}
			}
			return true, nil
		}, "s1-a"},
		{"on a step with its state changed", func(r umpire.Result) (bool, error) {
			if r.State == "s0" {
				return false, &hole{"changed"}
			}
			return slices.Contains(r.Facts, "f"), nil
		}, "changed"},
		{"on a step with a fact removed", func(r umpire.Result) (bool, error) {
			if len(r.Facts) == 0 {
				return false, &hole{"factless"}
			}
			return true, nil
		}, "factless"},
	} {
		t.Run(c.name, func(t *testing.T) {
			groups, err := umpire.KeyProperty(lowerable(), "p", on("a"), "", c.holds).Lower()
			require.Nil(t, groups)
			require.EqualError(t, err, "property p: a hole at "+c.at)
			require.ErrorAs(t, err, new(*umpire.Error))
			require.True(t, isHole(err), "the hole keeps its type")
		})
	}
}

// What a typed Property is refused for, a key-level one is refused for in the same words; a
// transition claim is never lowered.
func TestAKeyLevelPropertyIsRefusedAsATypedOneIs(t *testing.T) {
	tb := lowerable()
	for _, c := range []struct {
		name string
		p    *umpire.PropertyDecl
		want string
	}{
		{"a transition claim", umpire.KeyTransitionProperty(tb, "transition", always),
			"property transition: a transition claim is searched and verified, never realized"},
		{"a predicate no step satisfies", umpire.KeyProperty(tb, "never", on("a"), "", records("h")),
			"property never: the predicate holds on no step of this machine at `a`"},
		{"a predicate that fixes nothing", umpire.KeyProperty(tb, "anything", on("a"), "", func(umpire.Result) (bool, error) { return true, nil }),
			"property anything: the predicate holds on every step of this machine at `a` and fixes no state, outcome or fact, so it claims nothing"},
		{"a predicate that is no conjunction", umpire.KeyProperty(tb, "without", on("a"), "",
			func(r umpire.Result) (bool, error) { return !slices.Contains(r.Facts, "g"), nil }),
			"property without: the predicate is not a conjunction of one state, one outcome and facts: the clauses it fixes cannot tell " +
				"the step to s1 with outcome ok and facts [f g] apart from the steps it accepts at `a`"},
	} {
		t.Run(c.name, func(t *testing.T) {
			_, err := c.p.Lower()
			require.EqualError(t, err, c.want)
			require.ErrorAs(t, err, new(*umpire.Error))
		})
	}
}

// A composed table has no reading of one member's field, which a composed claim needs, so a Property
// of it is refused rather than lowered over whole composed states.
func TestAPropertyOfAComposedTableIsNotLowered(t *testing.T) {
	tb, err := umpire.ComposeTables(houseOf(keysOf(t, newDoor("door"), ""), umpire.ComposeMember{Table: keysOf(t, opaqueKey(), "")}))
	require.NoError(t, err)
	_, err = umpire.KeyProperty(tb, "p", nil, "", func(umpire.Result) (bool, error) { return true, nil }).Lower()
	require.EqualError(t, err, "property p: a claim of a composition is searched and verified, never realized")
}

// A keyed table carries the state fields and the Abstraction Claims its spec gives it, which a typed
// table derives from its declarations.
func TestAKeyedTableCarriesItsFieldValuesAndClaims(t *testing.T) {
	fields := map[string][]umpire.Atom{"s0": {{ID: "test.lower.state-field.steps.phase", Value: "zero"}},
		"s1": {{ID: "test.lower.state-field.steps.phase", Value: "one"}}}
	claims := []umpire.Claim{{Member: "test.lower.action.steps.a", Action: "test.lower.action.a", Field: "kind",
		ClassName: "a", Example: "an example"}}
	tb := lowerable(func(s *umpire.TableSpec) { s.FieldValues, s.Claims = fields, claims })
	require.NoError(t, tb.Err())
	require.Equal(t, fields["s1"], tb.FieldValues("s1"))
	require.Empty(t, tb.FieldValues("s2"))
	require.Equal(t, claims, tb.Claims())
	require.Empty(t, lowerable().Claims())

	for _, c := range []struct {
		name   string
		extend func(*umpire.TableSpec)
		want   string
	}{
		{"fields of no state", func(s *umpire.TableSpec) { s.FieldValues = map[string][]umpire.Atom{"s9": fields["s0"]} },
			"steps: the fields of 's9' are given, and it is not a state"},
		{"a claim of no action class", func(s *umpire.TableSpec) { s.Claims = []umpire.Claim{{Member: "test.lower.action.steps.z"}} },
			"steps: the claim on test.lower.action.steps.z is of no action class"},
	} {
		t.Run(c.name, func(t *testing.T) {
			require.EqualError(t, lowerable(c.extend).Err(), c.want)
		})
	}
}
