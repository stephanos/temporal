package goir

// Keys join a value's parts with "-", so case names that hold a "-" can spell two different values
// alike. A small generic Model, of no feature, puts that aliasing in each catalog a machine keys.

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	modelirspb "go.temporal.io/server/api/modelir/v1"
)

func at(line int32) *modelirspb.Position { return &modelirspb.Position{File: "generic", Line: line} }

func enumType(name string, line int32, cases ...string) *modelirspb.Type {
	e := &modelirspb.Enum{}
	for _, c := range cases {
		e.Cases = append(e.Cases, &modelirspb.Case{Name: c})
	}
	return &modelirspb.Type{Name: name, Position: at(line), Shape: &modelirspb.Type_Enum{Enum: e}}
}

func caseOf(typ, c string) *modelirspb.Value {
	return &modelirspb.Value{Kind: &modelirspb.Value_Enum{Enum: &modelirspb.EnumValue{Type: typ, Case: c}}}
}

// identityAction is an action of the generic Model: its name and the types of its inputs.
type identityAction struct {
	name   string
	inputs []string
}

// identityModel is a machine over the generic types: X = {a-b, a}, Y = {c, b-c}, T = (x: X, y: Y),
// S = {a, a-b}, C = {c} and O = {ok}, whose steps are all disabled.
func identityModel(state, outcome, fact string, actions ...identityAction) *modelirspb.Model {
	first := map[string]*modelirspb.Value{
		"T": {Kind: &modelirspb.Value_Record{Record: &modelirspb.RecordValue{Type: "T", Fields: []*modelirspb.Value{caseOf("X", "a-b"), caseOf("Y", "c")}}}},
		"S": caseOf("S", "a"),
		"O": caseOf("O", "ok"),
	}
	m := &modelirspb.Model{Source: "generic", Types: []*modelirspb.Type{
		enumType("X", 1, "a-b", "a"),
		enumType("Y", 2, "c", "b-c"),
		{Name: "T", Position: at(3), Shape: &modelirspb.Type_Record{Record: &modelirspb.Record{Fields: []*modelirspb.Field{
			{Name: "x", Type: named("X")}, {Name: "y", Type: named("Y")}}}}},
		enumType("S", 4, "a", "a-b"),
		enumType("O", 5, "ok"),
		enumType("C", 6, "c"),
	}}
	mm := &modelirspb.Machine{Family: "generic", Name: "m", Position: at(30), StateType: state, OutcomeType: outcome, FactType: fact,
		Starts: []*modelirspb.Expr{{Position: at(30), Kind: &modelirspb.Expr_Literal{Literal: first[state]}}}}
	for i, a := range actions {
		action := &modelirspb.Action{Id: "generic." + a.name, Name: a.name, Position: at(10 + int32(i)), Party: "generic"}
		step := &modelirspb.Function{Name: "generic.step." + a.name, Position: at(20 + int32(i)),
			Params: []*modelirspb.Param{{Name: "s", Type: named(state)}},
			Body: &modelirspb.Expr{Position: at(20 + int32(i)), Kind: &modelirspb.Expr_Literal{Literal: &modelirspb.Value{
				Kind: &modelirspb.Value_List{List: &modelirspb.ListValue{}}}}}}
		for j, typ := range a.inputs {
			name := string(rune('p' + j))
			action.Inputs = append(action.Inputs, &modelirspb.Param{Name: name, Type: named(typ)})
			step.Params = append(step.Params, &modelirspb.Param{Name: name, Type: named(typ)})
		}
		m.Actions = append(m.Actions, action)
		m.Functions = append(m.Functions, step)
		mm.Steps = append(mm.Steps, &modelirspb.StepBinding{Action: action.GetId(), Function: step.GetName(), Position: at(31 + int32(i))})
	}
	m.Machines = []*modelirspb.Machine{mm}
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
	for name, m := range map[string]*modelirspb.Model{
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
	m := &modelirspb.Model{Source: "generic", Types: []*modelirspb.Type{enumType("N", 7, "p", "q", "p-0,q")},
		Channels: []*modelirspb.Channel{{Id: "k", Name: "k", Position: at(8), Message: named("N"), Capacity: 2,
			Order: modelirspb.Channel_ORDER_FIFO}}}
	_, err := NewInterpreter(m).Members(&modelirspb.TypeRef{Ref: &modelirspb.TypeRef_Channel{Channel: "k"}})
	require.ErrorContains(t, err, `generic:8: the catalog of channel k holds [(p-0,q, 0)] and [(p, 0), (q, 0)], which share the key "[p-0,q-0]"`)
}

// A state catalog of 100³ values, past the default ceilings: admission does not refuse the Model for
// its size, and Build refuses it within its ceilings.
func TestAdmissionLeavesALargeCatalogToBuild(t *testing.T) {
	m := identityModel("S", "O", "")
	hundred := &modelirspb.TypeRef{Ref: &modelirspb.TypeRef_IntRange{IntRange: &modelirspb.IntRange{High: 99}}}
	m.Types[3] = &modelirspb.Type{Name: "S", Position: at(4), Shape: &modelirspb.Type_Record{Record: &modelirspb.Record{Fields: []*modelirspb.Field{
		{Name: "x", Type: hundred}, {Name: "y", Type: hundred}, {Name: "z", Type: hundred}}}}}
	m.Machines[0].Starts[0].Kind = &modelirspb.Expr_Literal{Literal: &modelirspb.Value{Kind: &modelirspb.Value_Record{Record: &modelirspb.RecordValue{
		Type: "S", Fields: []*modelirspb.Value{admIntValue(0), admIntValue(0), admIntValue(0)}}}}}
	require.NoError(t, Validate(m))
	_, err := Build(m)
	var limit *LimitError
	require.ErrorAs(t, err, &limit)
	require.Equal(t, LimitError{Machine: "m", Resource: "members", Ceiling: 1 << 16, Needed: 1_000_000}, *limit)
}

// compositionModel is a composition p of two machines over S: m1 binds the action b_c and m2 binds c,
// filling the fields of P named first and second, with the syncs pairing m1's b_c with m2's c, or,
// named "rev:<name>", m2's c with m1's b_c.
func compositionModel(first, second string, syncs ...string) *modelirspb.Model {
	m := &modelirspb.Model{Source: "generic", Types: []*modelirspb.Type{
		enumType("S", 1, "a", "a-b"),
		enumType("O", 2, "ok"),
		{Name: "P", Position: at(3), Shape: &modelirspb.Type_Record{Record: &modelirspb.Record{Fields: []*modelirspb.Field{
			{Name: first, Type: named("S")}, {Name: second, Type: named("S")}}}}},
	}}
	for i, action := range []string{"b_c", "c"} {
		line := int32(10 * (i + 1))
		name := "m" + string(rune('1'+i))
		m.Actions = append(m.Actions, &modelirspb.Action{Id: "generic." + action, Name: action, Position: at(line), Party: "generic"})
		m.Functions = append(m.Functions, &modelirspb.Function{Name: "generic.step." + action, Position: at(line + 1),
			Params: []*modelirspb.Param{{Name: "s", Type: named("S")}},
			Body: &modelirspb.Expr{Position: at(line + 1), Kind: &modelirspb.Expr_Literal{Literal: &modelirspb.Value{
				Kind: &modelirspb.Value_List{List: &modelirspb.ListValue{}}}}}})
		m.Machines = append(m.Machines, &modelirspb.Machine{Family: "generic", Name: name, Position: at(line + 2), StateType: "S", OutcomeType: "O",
			Starts: []*modelirspb.Expr{{Position: at(line + 2), Kind: &modelirspb.Expr_Literal{Literal: caseOf("S", "a")}}},
			Steps:  []*modelirspb.StepBinding{{Action: "generic." + action, Function: "generic.step." + action, Position: at(line + 3)}}})
	}
	p := &modelirspb.Composition{Family: "generic", Name: "p", Position: at(40), StateType: "P",
		Members: []*modelirspb.Member{{Field: first, Machine: "m1"}, {Field: second, Machine: "m2"}}}
	for _, s := range syncs {
		sync := &modelirspb.Sync{Name: s, First: &modelirspb.SyncMove{Member: first, Action: "b_c"},
			Second: &modelirspb.SyncMove{Member: second, Action: "c"}}
		// "rev:" pairs them the other way round.
		if name, ok := strings.CutPrefix(s, "rev:"); ok {
			sync.Name, sync.First, sync.Second = name, sync.GetSecond(), sync.GetFirst()
		}
		p.Syncs = append(p.Syncs, sync)
	}
	m.Compositions = []*modelirspb.Composition{p}
	return m
}

func TestAWellKeyedCompositionIsAdmitted(t *testing.T) {
	m := compositionModel("a", "d", "both")
	m.Scenarios = []*modelirspb.Scenario{{Machine: "p", Name: "each", Position: at(50), Keys: []string{"a_b_c", "d_c", "both"},
		Start: &modelirspb.Expr{Position: at(50), Kind: &modelirspb.Expr_Literal{Literal: &modelirspb.Value{Kind: &modelirspb.Value_Record{
			Record: &modelirspb.RecordValue{Type: "P", Fields: []*modelirspb.Value{caseOf("S", "a"), caseOf("S", "a")}}}}}}}}
	require.NoError(t, Validate(m))
}

// Member a's class b_c and member a_b's class c both key a_b_c; member d's class c and the sync d_c
// both key d_c. Either would be one class of p under two owners.
func TestComposedClassesSharingAKeyAreRefused(t *testing.T) {
	for name, c := range map[string]struct {
		model *modelirspb.Model
		want  string
	}{
		"two members":         {compositionModel("a", "a_b"), `generic:40: composition p: member a's class b_c and member a_b's class c share the key "a_b_c"`},
		"a member and a sync": {compositionModel("a", "d", "d_c"), `generic:40: composition p: member d's class c and sync d_c of a.b_c and d.c share the key "d_c"`},
		"two syncs of one name": {compositionModel("a", "d", "both", "rev:both"),
			`generic:40: composition p: sync both of a.b_c and d.c and sync both of d.c and a.b_c share the key "both"`},
	} {
		t.Run(name, func(t *testing.T) {
			require.ErrorContains(t, Validate(c.model), c.want)
		})
	}
}
