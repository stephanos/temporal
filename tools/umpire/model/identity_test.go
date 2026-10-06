package model

// Keys join a value's parts with "-", so case names that hold a "-" can spell two different values
// alike. A small generic Model, of no feature, puts that aliasing in each catalog a machine keys.

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	umpirespb "go.temporal.io/server/api/umpire/v1"
)

func at(line int32) *umpirespb.Position { return &umpirespb.Position{File: "generic", Line: line} }

func enumType(name string, line int32, cases ...string) *umpirespb.Type {
	e := &umpirespb.Enum{}
	for _, c := range cases {
		e.Cases = append(e.Cases, &umpirespb.Case{Name: c})
	}
	return &umpirespb.Type{Name: name, Position: at(line), Shape: &umpirespb.Type_Enum{Enum: e}}
}

func caseOf(typ, c string) *umpirespb.Value {
	return &umpirespb.Value{Kind: &umpirespb.Value_Enum{Enum: &umpirespb.EnumValue{Type: typ, Case: c}}}
}

// identityAction is an action of the generic Model: its name and the types of its inputs.
type identityAction struct {
	name   string
	inputs []string
}

// identityModel is a machine over the generic types: X = {a-b, a}, Y = {c, b-c}, T = (x: X, y: Y),
// S = {a, a-b}, C = {c} and O = {ok}, whose steps are all disabled.
func identityModel(state, outcome, fact string, actions ...identityAction) *umpirespb.Model {
	first := map[string]*umpirespb.Value{
		"T": {Kind: &umpirespb.Value_Record{Record: &umpirespb.RecordValue{Type: "T", Fields: []*umpirespb.Value{caseOf("X", "a-b"), caseOf("Y", "c")}}}},
		"S": caseOf("S", "a"),
		"O": caseOf("O", "ok"),
	}
	m := &umpirespb.Model{Source: "generic", Types: []*umpirespb.Type{
		enumType("X", 1, "a-b", "a"),
		enumType("Y", 2, "c", "b-c"),
		{Name: "T", Position: at(3), Shape: &umpirespb.Type_Record{Record: &umpirespb.Record{Fields: []*umpirespb.Field{
			{Name: "x", Type: named("X")}, {Name: "y", Type: named("Y")}}}}},
		enumType("S", 4, "a", "a-b"),
		enumType("O", 5, "ok"),
		enumType("C", 6, "c"),
	}}
	mm := &umpirespb.Machine{Family: "generic", Name: "m", Position: at(30), StateType: state, OutcomeType: outcome, FactType: fact,
		Starts: []*umpirespb.Expr{{Position: at(30), Kind: &umpirespb.Expr_Literal{Literal: first[state]}}}}
	for i, a := range actions {
		action := &umpirespb.Action{Id: "generic." + a.name, Name: a.name, Position: at(10 + int32(i)), Actor: "generic"}
		step := &umpirespb.Function{Name: "generic.step." + a.name, Position: at(20 + int32(i)),
			Params: []*umpirespb.Param{{Name: "s", Type: named(state)}},
			Body: &umpirespb.Expr{Position: at(20 + int32(i)), Kind: &umpirespb.Expr_Literal{Literal: &umpirespb.Value{
				Kind: &umpirespb.Value_List{List: &umpirespb.ListValue{}}}}}}
		for j, typ := range a.inputs {
			name := string(rune('p' + j))
			action.Inputs = append(action.Inputs, &umpirespb.Param{Name: name, Type: named(typ)})
			step.Params = append(step.Params, &umpirespb.Param{Name: name, Type: named(typ)})
		}
		m.Actions = append(m.Actions, action)
		m.Functions = append(m.Functions, step)
		mm.Steps = append(mm.Steps, &umpirespb.StepBinding{Action: action.GetId(), Function: step.GetName(), Position: at(31 + int32(i))})
	}
	m.Machines = []*umpirespb.Machine{mm}
	return m
}

func TestTheGenericModelIsAdmittedWithoutAliasing(t *testing.T) {
	m := identityModel("S", "O", "", identityAction{name: "go", inputs: []string{"X"}})
	require.NoError(t, Validate(m))
	built, err := Build(m)
	require.NoError(t, err)
	require.Equal(t, []string{"a", "a-b"}, built["m"].Table.States)
	require.Equal(t, []string{"go-a", "go-a-b"}, built["m"].Table.Actions)
}

// Two values of one catalog that share a key are refused, at admission and when interpreted, where
// the catalog's type is declared: (a-b, c) and (a, b-c) both key a-b-c.
func TestValuesSharingAKeyAreRefused(t *testing.T) {
	aliased := `generic:3: the catalog of T holds (a-b, c) and (a, b-c), which share the key "a-b-c"`
	for name, m := range map[string]*umpirespb.Model{
		"states":   identityModel("T", "O", ""),
		"outcomes": identityModel("S", "T", ""),
		"facts":    identityModel("S", "O", "T"),
	} {
		t.Run(name, func(t *testing.T) {
			require.ErrorContains(t, Validate(m), aliased)
			_, err := Build(m)
			require.ErrorContains(t, err, aliased)
		})
	}
}

// go(x: X, y: Y) has the classes go(a-b, c) and go(a, b-c), both keyed go-a-b-c.
func TestClassesSharingAKeyAreRefused(t *testing.T) {
	m := identityModel("S", "O", "", identityAction{name: "go", inputs: []string{"X", "Y"}})
	aliased := `generic:31: m: the classes go(a-b, c) and go(a, b-c) share the key "go-a-b-c"`
	require.ErrorContains(t, Validate(m), aliased)
	_, err := Build(m)
	require.ErrorContains(t, err, aliased)
}

// State a with class b-c, and state a-b with class c, both key the row a-b-c.
func TestRowsSharingAKeyAreRefused(t *testing.T) {
	m := identityModel("S", "O", "", identityAction{name: "b", inputs: []string{"C"}}, identityAction{name: "c"})
	aliased := `generic:30: m: the state a with the class b-c, and the state a-b with the class c, share the row key "a-b-c"`
	require.ErrorContains(t, Validate(m), aliased)
	_, err := Build(m)
	require.ErrorContains(t, err, aliased)
}

// Messages p, q and "p-0,q": the one delivery of "p-0,q" and the two of p and q both key [p-0,q-0].
func TestChannelContentsSharingAKeyAreRefused(t *testing.T) {
	m := &umpirespb.Model{Source: "generic", Types: []*umpirespb.Type{enumType("N", 7, "p", "q", "p-0,q")},
		Channels: []*umpirespb.Channel{{Id: "k", Name: "k", Position: at(8), Message: named("N"), Capacity: 2,
			Order: umpirespb.Channel_ORDER_FIFO}}}
	_, err := NewInterpreter(m).Members(&umpirespb.TypeRef{Ref: &umpirespb.TypeRef_Channel{Channel: "k"}})
	require.ErrorContains(t, err, `generic:8: the catalog of channel k holds [(p-0,q, 0)] and [(p, 0), (q, 0)], which share the key "[p-0,q-0]"`)
}

// A state catalog of 100³ values, past the default ceilings: admission does not refuse the Model for
// its size, and Build refuses it within its ceilings.
func TestAdmissionLeavesALargeCatalogToBuild(t *testing.T) {
	m := identityModel("S", "O", "")
	hundred := &umpirespb.TypeRef{Ref: &umpirespb.TypeRef_IntRange{IntRange: &umpirespb.IntRange{High: 99}}}
	m.Types[3] = &umpirespb.Type{Name: "S", Position: at(4), Shape: &umpirespb.Type_Record{Record: &umpirespb.Record{Fields: []*umpirespb.Field{
		{Name: "x", Type: hundred}, {Name: "y", Type: hundred}, {Name: "z", Type: hundred}}}}}
	m.Machines[0].Starts[0].Kind = &umpirespb.Expr_Literal{Literal: &umpirespb.Value{Kind: &umpirespb.Value_Record{Record: &umpirespb.RecordValue{
		Type: "S", Fields: []*umpirespb.Value{admIntValue(0), admIntValue(0), admIntValue(0)}}}}}
	require.NoError(t, Validate(m))
	_, err := Build(m)
	var limit *LimitError
	require.ErrorAs(t, err, &limit)
	require.Equal(t, LimitError{Machine: "m", Resource: "members", Ceiling: 1 << 16, Needed: 1_000_000}, *limit)
}

// compositionModel is a composition p of two machines over S: m1 binds the action b_c and m2 binds c,
// filling the fields of P named first and second, with the syncs pairing m1's b_c with m2's c, or,
// named "rev:<name>", m2's c with m1's b_c.
func compositionModel(first, second string, syncs ...string) *umpirespb.Model {
	m := &umpirespb.Model{Source: "generic", Types: []*umpirespb.Type{
		enumType("S", 1, "a", "a-b"),
		enumType("O", 2, "ok"),
		{Name: "P", Position: at(3), Shape: &umpirespb.Type_Record{Record: &umpirespb.Record{Fields: []*umpirespb.Field{
			{Name: first, Type: named("S")}, {Name: second, Type: named("S")}}}}},
	}}
	for i, action := range []string{"b_c", "c"} {
		line := int32(10 * (i + 1))
		name := "m" + string(rune('1'+i))
		m.Actions = append(m.Actions, &umpirespb.Action{Id: "generic." + action, Name: action, Position: at(line), Actor: "generic"})
		m.Functions = append(m.Functions, &umpirespb.Function{Name: "generic.step." + action, Position: at(line + 1),
			Params: []*umpirespb.Param{{Name: "s", Type: named("S")}},
			Body: &umpirespb.Expr{Position: at(line + 1), Kind: &umpirespb.Expr_Literal{Literal: &umpirespb.Value{
				Kind: &umpirespb.Value_List{List: &umpirespb.ListValue{}}}}}})
		m.Machines = append(m.Machines, &umpirespb.Machine{Family: "generic", Name: name, Position: at(line + 2), StateType: "S", OutcomeType: "O",
			Starts: []*umpirespb.Expr{{Position: at(line + 2), Kind: &umpirespb.Expr_Literal{Literal: caseOf("S", "a")}}},
			Steps:  []*umpirespb.StepBinding{{Action: "generic." + action, Function: "generic.step." + action, Position: at(line + 3)}}})
	}
	p := &umpirespb.Composition{Family: "generic", Name: "p", Position: at(40), StateType: "P",
		Members: []*umpirespb.Member{{Field: first, Machine: "m1"}, {Field: second, Machine: "m2"}}}
	for _, s := range syncs {
		sync := &umpirespb.Sync{Name: s, First: &umpirespb.SyncMove{Member: first, Action: "b_c"},
			Second: &umpirespb.SyncMove{Member: second, Action: "c"}}
		// "rev:" pairs them the other way round.
		if name, ok := strings.CutPrefix(s, "rev:"); ok {
			sync.Name, sync.First, sync.Second = name, sync.GetSecond(), sync.GetFirst()
		}
		p.Syncs = append(p.Syncs, sync)
	}
	m.Compositions = []*umpirespb.Composition{p}
	return m
}

func TestAWellKeyedCompositionIsAdmitted(t *testing.T) {
	for name, c := range map[string]struct {
		model *umpirespb.Model
		keys  []string
	}{
		"no sync": {compositionModel("a", "d"), []string{"a_b_c", "d_c"}},
		"a sync":  {compositionModel("a", "d", "both"), []string{"both"}},
		// Member d's action c steps only as the sync, so the sync alone keys d_c.
		"a sync named like the class it takes": {compositionModel("a", "d", "d_c"), []string{"d_c"}},
	} {
		t.Run(name, func(t *testing.T) {
			c.model.Scenarios = []*umpirespb.Scenario{compositionScenario(c.keys...)}
			require.NoError(t, Validate(c.model))
		})
	}
}

// A member's action a sync names is no class of the composition, so a Scenario cannot schedule it.
func TestAScenarioCannotScheduleASyncedMemberAction(t *testing.T) {
	for _, key := range []string{"a_b_c", "d_c"} {
		t.Run(key, func(t *testing.T) {
			m := compositionModel("a", "d", "both")
			m.Scenarios = []*umpirespb.Scenario{compositionScenario(key)}
			require.ErrorContains(t, Validate(m), "p has no class "+key)
		})
	}
}

// compositionScenario is a Scenario of compositionModel's p scheduling these class keys.
func compositionScenario(keys ...string) *umpirespb.Scenario {
	return &umpirespb.Scenario{Machine: "p", Name: "each", Position: at(50), Keys: keys,
		Start: &umpirespb.Expr{Position: at(50), Kind: &umpirespb.Expr_Literal{Literal: &umpirespb.Value{Kind: &umpirespb.Value_Record{
			Record: &umpirespb.RecordValue{Type: "P", Fields: []*umpirespb.Value{caseOf("S", "a"), caseOf("S", "a")}}}}}}}
}

// alsoBinds gives compositionModel's m2 a second action, which no sync of the fixture names.
func alsoBinds(m *umpirespb.Model, action string) *umpirespb.Model {
	m.Actions = append(m.Actions, &umpirespb.Action{Id: "generic." + action, Name: action, Position: at(30), Actor: "generic"})
	m.Functions = append(m.Functions, &umpirespb.Function{Name: "generic.step." + action, Position: at(31),
		Params: []*umpirespb.Param{{Name: "s", Type: named("S")}},
		Body: &umpirespb.Expr{Position: at(31), Kind: &umpirespb.Expr_Literal{Literal: &umpirespb.Value{
			Kind: &umpirespb.Value_List{List: &umpirespb.ListValue{}}}}}})
	m.Machines[1].Steps = append(m.Machines[1].Steps,
		&umpirespb.StepBinding{Action: "generic." + action, Function: "generic.step." + action, Position: at(33)})
	return m
}

// Member a's class b_c and member a_b's class c both key a_b_c; member d's class x, which no sync
// takes, and the sync d_x both key d_x. Either would be one class of p under two owners.
func TestComposedClassesSharingAKeyAreRefused(t *testing.T) {
	for name, c := range map[string]struct {
		model *umpirespb.Model
		want  string
	}{
		"two members":         {compositionModel("a", "a_b"), `generic:40: composition p: member a's class b_c and member a_b's class c share the key "a_b_c"`},
		"a member and a sync": {alsoBinds(compositionModel("a", "d", "d_x"), "x"), `generic:40: composition p: member d's class x and sync d_x of a.b_c and d.c share the key "d_x"`},
		"two syncs of one name": {compositionModel("a", "d", "both", "rev:both"),
			`generic:40: composition p: sync both of a.b_c and d.c and sync both of d.c and a.b_c share the key "both"`},
	} {
		t.Run(name, func(t *testing.T) {
			require.ErrorContains(t, Validate(c.model), c.want)
		})
	}
}
