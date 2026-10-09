package check

// Every Property of the standalone activity Model, evaluated on every row of its machine. A Query over
// one of the Scenarios' paths reads a Property on the few steps of that path; a predicate that differs
// on any other row would answer every such Query alike. Here each row is a Scenario of its own, one
// step from the row's state by the row's class, and each Property is verified over it through Check.
// The expected answers are pinned as counts and failing rows, and a counterexample must be the row's
// own step, so an edited predicate cannot alter its own expectation.
//
// What one answer says of a row: whether the Property is about its step (the claim fired), and
// whether it holds there (verified, or a counterexample that is the step). A product machine's
// Properties are also read on every row of the machine that refines it, through the refinement, and
// the composition's Property on every row of the composition, by the row's composed class key.

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"hash"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
	umpirespb "go.temporal.io/server/api/umpire/v1"
	umpire "go.temporal.io/server/tools/umpire/internal/engine"
	"go.temporal.io/server/tools/umpire/interp"
	"go.temporal.io/server/tools/umpire/ir"
	"google.golang.org/protobuf/proto"
)

// oneStep bounds a row's Scenario: its one step, and room for every result of the row.
var oneStep = Limits{Name: "oneStep", Steps: 1, Actions: 1, Search: 64}

// rowSide is a Property's answer on one row.
type rowSide struct {
	Outcome            Outcome
	Explored, Expanded int
	Rows               []string
	// About is whether the Property is about the row's step.
	About   bool
	Witness *Trace
	Results []interp.Result
}

func rowKeyOf(property, machine, row string) string {
	return fmt.Sprintf("%s on %s at %s", property, machine, row)
}

// rowStep is the one-step witness of a row and one of its results.
func rowStep(t *testing.T, table *interp.Table, row interp.Row, result interp.Result) *Trace {
	t.Helper()
	atom := func(kind, key string) interp.Atom {
		return interp.Atom{ID: table.Family.ID(kind, table.OwnerName(), key), Value: key}
	}
	step := TraceStep{Action: atom("action", row.Action), Outcome: atom("outcome", result.Outcome),
		State: atom("state", result.State)}
	for _, fact := range result.Facts {
		step.Facts = append(step.Facts, atom("fact", fact))
	}
	return &Trace{Initial: atom("state", row.Source), Steps: []TraceStep{step}}
}

func counterexampleResults(t *testing.T, b *binding, q *umpirespb.Query, r Receipt) []interp.Result {
	t.Helper()
	require.Len(t, r.Rows, 1, q.GetName())
	require.NotNil(t, r.Witness, q.GetName())
	require.Len(t, r.Witness.Steps, 1, q.GetName())
	on := b.subject(q.GetScenario().GetMachine())
	require.NoError(t, on.err)
	p := b.declaredProperty(q.GetProperty())
	require.NotNil(t, p)
	owner := b.subject(p.GetMachine())
	reading, err := b.propertyReads(owner, p)
	require.NoError(t, err)
	property := boundProperty(p, reading)
	var matching []interp.Result
	for _, row := range on.table.RowsFrom(r.Witness.Initial.Value) {
		if row.Key != r.Rows[0] {
			continue
		}
		require.True(t, property.About(row.Action), q.GetName())
		for _, result := range row.Results {
			if !sameTrace(rowStep(t, on.table, row, result), r.Witness) {
				continue
			}
			before, after := row.Source, result
			if q.GetThrough() {
				refinement := b.refinement(on)
				require.NoError(t, refinement.err)
				before, err = refinement.ref.MapState(before)
				require.NoError(t, err)
				after.State, err = refinement.ref.MapState(result.State)
				require.NoError(t, err)
				after.Step, after.Facts = nil, []string{}
				for _, fact := range result.Facts {
					name := fact
					if !slices.Contains(owner.table.Facts, name) {
						name, _, _ = strings.Cut(name, "-")
					}
					if slices.Contains(owner.table.Facts, name) {
						after.Facts = append(after.Facts, name)
					}
				}
			}
			holds, err := property.Holds(before, after)
			require.NoError(t, err, q.GetName())
			require.False(t, holds, "%s: its counterexample satisfies its predicate", q.GetName())
			result.Step = nil
			matching = append(matching, result)
		}
	}
	require.NotEmpty(t, matching, "%s: the counterexample is no result of its row", q.GetName())
	return matching
}

// protoValue is a value as the IR writes one, for a Scenario that starts in it or takes it as input.
func protoValue(v interp.Value) *umpirespb.Value {
	fields := func(vs []interp.Value) []*umpirespb.Value {
		var out []*umpirespb.Value
		for _, f := range vs {
			out = append(out, protoValue(f))
		}
		return out
	}
	switch v.Kind {
	case interp.BoolValue:
		return &umpirespb.Value{Kind: &umpirespb.Value_Bool{Bool: v.Bool}}
	case interp.IntValue:
		return &umpirespb.Value{Kind: &umpirespb.Value_Int{Int: v.Int}}
	case interp.TextValue:
		return &umpirespb.Value{Kind: &umpirespb.Value_Text{Text: v.Text}}
	case interp.EnumValue:
		return &umpirespb.Value{Kind: &umpirespb.Value_Enum{Enum: &umpirespb.EnumValue{Type: v.Type, Case: v.Case, Fields: fields(v.Fields)}}}
	case interp.RecordValue:
		return &umpirespb.Value{Kind: &umpirespb.Value_Record{Record: &umpirespb.RecordValue{Type: v.Type, Fields: fields(v.Fields)}}}
	case interp.ListValue:
		return &umpirespb.Value{Kind: &umpirespb.Value_List{List: &umpirespb.ListValue{Items: fields(v.Items)}}}
	case interp.LambdaValue:
		panic("a function is no state and no input")
	default:
		panic("no such kind of value")
	}
}

// irPropertyRowsBatch asks selected rows in one synthetic Model, independently of the bounded walk.
// Each selected row has a one-step Scenario and a verify of every declared Property of its owner
// and, through refinement, every Property of the machine it refines.
func irPropertyRowsBatch(t *testing.T, base *umpirespb.Model, selected map[string][]string) map[string]rowSide {
	t.Helper()
	m := proto.Clone(base).(*umpirespb.Model)
	m.Queries = nil
	properties := slices.Clone(m.GetProperties())
	asked := map[string]*umpirespb.Query{}
	// ask adds a row's Scenario of the machine or composition `name`, and the verifies over it.
	ask := func(name, refined, row string, scenario *umpirespb.Scenario) {
		declared := false
		for _, p := range properties {
			own := p.GetMachine() == name
			if !own && p.GetMachine() != refined {
				continue
			}
			declared = true
			key := rowKeyOf(p.GetName(), name, row)
			q := &umpirespb.Query{Name: key, Position: scenario.GetPosition(), Form: umpirespb.Query_FORM_VERIFY,
				Property: &umpirespb.ClaimRef{Machine: p.GetMachine(), Name: p.GetName()},
				Scenario: &umpirespb.ClaimRef{Machine: name, Name: scenario.GetName()}, Through: !own,
				Limits: &umpirespb.Limits{Name: oneStep.Name, Steps: int32(oneStep.Steps), Actions: int32(oneStep.Actions),
					Search: int32(oneStep.Search)}}
			asked["query "+name+" "+key] = q
			m.Queries = append(m.Queries, q)
		}
		if declared {
			m.Scenarios = append(m.Scenarios, scenario)
		}
	}
	from := func(name string, at *umpirespb.Position, row interp.Row, state interp.Value) *umpirespb.Scenario {
		return &umpirespb.Scenario{Machine: name, Name: "row." + row.Key, Position: at,
			Start: &umpirespb.Expr{Position: at, Kind: &umpirespb.Expr_Literal{Literal: protoValue(state)}}}
	}
	for name, mm := range built(t, m) {
		if _, ok := selected[name]; !ok {
			continue
		}
		classes := map[string]interp.Class{}
		for _, c := range mm.Classes {
			classes[c.Key] = c
		}
		for _, row := range mm.Table.Rows {
			if !slices.Contains(selected[name], row.Key) {
				continue
			}
			state, ok := mm.State(row.Source)
			require.True(t, ok, row.Key)
			class := classes[row.Action]
			step := &umpirespb.ActionClass{Action: class.Action.GetId()}
			for _, in := range class.Inputs {
				step.Inputs = append(step.Inputs, protoValue(in))
			}
			scenario := from(name, mm.Decl.GetPosition(), row, state)
			scenario.Actions = []*umpirespb.ActionClass{step}
			ask(name, mm.Decl.GetRefines().GetProduct(), row.Key, scenario)
		}
	}
	composed := bind(base, DefaultScope)
	for _, c := range m.GetCompositions() {
		if _, ok := selected[c.GetName()]; !ok {
			continue
		}
		s := composed.subject(c.GetName())
		require.NoError(t, s.err)
		for _, row := range s.table.Rows {
			if !slices.Contains(selected[c.GetName()], row.Key) {
				continue
			}
			state, err := s.state(row.Source)
			require.NoError(t, err, row.Key)
			scenario := from(c.GetName(), c.GetPosition(), row, state)
			scenario.Keys = []string{row.Action}
			ask(c.GetName(), "", row.Key, scenario)
		}
	}
	out := map[string]rowSide{}
	for _, r := range Check(m, DefaultScope).Receipts {
		q, ok := asked[receiptKey(r)]
		if !ok {
			continue
		}
		side := rowSide{About: r.Exercised, Witness: r.Witness, Explored: r.Explored, Expanded: r.Expanded, Rows: r.Rows}
		switch r.Kind {
		case Verified:
			side.Outcome = umpire.VerifiedWithinLimits
		case Counterexample:
			side.Outcome = umpire.CounterexampleFound
			side.Results = counterexampleResults(t, composed, q, r)
		default:
			// Any other kind is no answer about the row, and equals no other answer.
			side.Outcome = Outcome(r.Kind)
		}
		out[q.GetName()] = side
	}
	require.Len(t, out, len(asked))
	return out
}

type propertyRowOwner struct {
	name       string
	on         *subject
	rows       []interp.Row
	properties []*umpirespb.Property
	classes    map[string]interp.Class
}

type propertyRowWalk struct {
	model   *umpirespb.Model
	checker *checker
	owner   propertyRowOwner
}

func propertyRowMachines(m *umpirespb.Model, owner string) ([]*umpirespb.Machine, error) {
	machines := map[string]*umpirespb.Machine{}
	compositions := map[string]*umpirespb.Composition{}
	for _, machine := range m.GetMachines() {
		machines[machine.GetName()] = machine
	}
	for _, composition := range m.GetCompositions() {
		compositions[composition.GetName()] = composition
	}
	seen, needed := map[string]bool{}, map[string]bool{}
	var visit func(string) error
	visit = func(name string) error {
		if name == "" || seen[name] {
			return nil
		}
		seen[name] = true
		if machine := machines[name]; machine != nil {
			needed[name] = true
			return visit(machine.GetRefines().GetProduct())
		}
		if composition := compositions[name]; composition != nil {
			for _, member := range composition.GetMembers() {
				if err := visit(member.GetMachine()); err != nil {
					return err
				}
				if err := visit(member.GetReplaces()); err != nil {
					return err
				}
			}
			return nil
		}
		return fmt.Errorf("row owner %s requires undeclared subject %s", owner, name)
	}
	if err := visit(owner); err != nil {
		return nil, err
	}
	var out []*umpirespb.Machine
	for _, machine := range m.GetMachines() {
		if needed[machine.GetName()] {
			out = append(out, machine)
		}
	}
	return out, nil
}

// Only row tables are scoped: the interpreter and binding keep the admitted Model's full metadata.
func bindPropertyRowOwner(t *testing.T, m *umpirespb.Model, owner string) *binding {
	t.Helper()
	machines, err := propertyRowMachines(m, owner)
	require.NoError(t, err)
	in := interp.NewInterpreterWithin(m, DefaultScope.Ceilings)
	built := in.Interpret(&umpirespb.Model{Machines: machines, Actions: m.GetActions(),
		Monitors: m.GetMonitors(), Assumptions: m.GetAssumptions()})
	require.Empty(t, built.Failed, owner)
	require.Len(t, built.Machines, len(machines), owner)
	b := &binding{model: m, scope: DefaultScope, in: in, machines: built.Machines, failed: built.Failed,
		actions: map[string]*umpirespb.Action{}, catalogs: map[string]map[string]interp.Value{}, subjects: map[string]*subject{},
		refined: map[string]*refined{}, properties: map[claim]*PropertyDecl{}, scenarios: map[scheduled]*ScenarioDecl{}}
	for _, action := range m.GetActions() {
		b.actions[action.GetId()] = action
	}
	return b
}

func newPropertyRowWalk(t *testing.T, base *umpirespb.Model, name string) *propertyRowWalk {
	t.Helper()
	m := proto.Clone(base).(*umpirespb.Model)
	m.Scenarios, m.Queries = nil, nil
	b := bindPropertyRowOwner(t, m, name)
	w := &propertyRowWalk{model: m, checker: newCheckerWithBinding(b, m)}
	// again must replay against a second scoped interpretation, never fall back to full bind.
	w.checker.fresh = bindPropertyRowOwner(t, m, name)
	on := b.subject(name)
	require.NoError(t, on.err)
	o := propertyRowOwner{name: name, on: on, classes: map[string]interp.Class{}}
	refined := ""
	if on.machine != nil {
		refined = on.machine.Decl.GetRefines().GetProduct()
		for _, class := range on.machine.Classes {
			o.classes[class.Key] = class
		}
	}
	for _, p := range m.GetProperties() {
		if p.GetMachine() == name || p.GetMachine() == refined {
			o.properties = append(o.properties, p)
		}
	}
	o.rows = slices.Clone(on.table.Rows)
	slices.SortFunc(o.rows, func(a, b interp.Row) int { return strings.Compare(a.Key, b.Key) })
	w.owner = o
	return w
}

type propertyRowInventory struct {
	Rows, Properties int
	Machine          bool
	Refined          string
}

func propertyRowOwnerNames(m *umpirespb.Model) []string {
	var names []string
	for _, machine := range m.GetMachines() {
		names = append(names, machine.GetName())
	}
	for _, composition := range m.GetCompositions() {
		names = append(names, composition.GetName())
	}
	slices.Sort(names)
	return names
}

func (w *propertyRowWalk) prepare(t *testing.T, owner propertyRowOwner, row interp.Row) *umpirespb.Scenario {
	t.Helper()
	state, err := owner.on.state(row.Source)
	require.NoError(t, err, row.Key)
	scenario := &umpirespb.Scenario{Machine: owner.name, Name: "row." + row.Key, Position: owner.on.at,
		Start: &umpirespb.Expr{Position: owner.on.at, Kind: &umpirespb.Expr_Literal{Literal: protoValue(state)}}}
	if owner.on.machine == nil {
		scenario.Keys = []string{row.Action}
	} else {
		class, ok := owner.classes[row.Action]
		require.True(t, ok, row.Key)
		step := &umpirespb.ActionClass{Action: class.Action.GetId()}
		for _, in := range class.Inputs {
			step.Inputs = append(step.Inputs, protoValue(in))
		}
		scenario.Actions = []*umpirespb.ActionClass{step}
	}
	w.model.Scenarios = []*umpirespb.Scenario{scenario}
	w.checker.first.scenarios = map[scheduled]*ScenarioDecl{}
	if w.checker.fresh != nil {
		w.checker.fresh.scenarios = map[scheduled]*ScenarioDecl{}
	}
	return scenario
}

func (w *propertyRowWalk) answer(t *testing.T, owner propertyRowOwner, p *umpirespb.Property, scenario *umpirespb.Scenario, row interp.Row) rowSide {
	t.Helper()
	q := &umpirespb.Query{Name: rowKeyOf(p.GetName(), owner.name, row.Key), Position: scenario.GetPosition(),
		Form: umpirespb.Query_FORM_VERIFY, Property: &umpirespb.ClaimRef{Machine: p.GetMachine(), Name: p.GetName()},
		Scenario: &umpirespb.ClaimRef{Machine: owner.name, Name: scenario.GetName()}, Through: p.GetMachine() != owner.name,
		Limits: &umpirespb.Limits{Name: oneStep.Name, Steps: int32(oneStep.Steps), Actions: int32(oneStep.Actions), Search: int32(oneStep.Search)}}
	r := w.checker.query(q)
	side := rowSide{About: r.Exercised, Witness: r.Witness, Explored: r.Explored, Expanded: r.Expanded, Rows: r.Rows}
	switch r.Kind {
	case Verified:
		side.Outcome = umpire.VerifiedWithinLimits
	case Counterexample:
		require.True(t, r.Exercised, q.GetName())
		side.Outcome = umpire.CounterexampleFound
		side.Results = counterexampleResults(t, w.checker.first, q, r)
	default:
		require.FailNow(t, "a row Query gave no property answer", "%s: %s: %v", q.GetName(), r.Kind, r.Cause)
	}
	return side
}

// walkPropertyRows retains one Scenario per binding and one Query per answer. Every declared
// Property reads every result of every selected row through the normal checker, including replay.
// A nil selection asks the complete universe; selections are only for the independent batch oracle.
func walkPropertyRows(t *testing.T, models []*umpirespb.Model, selected map[string][]string,
	visit func(string, *interp.Table, interp.Row, []rowSide)) map[string]propertyRowInventory {
	t.Helper()
	require.NotEmpty(t, models)
	names := propertyRowOwnerNames(models[0])
	for _, m := range models {
		require.NoError(t, ir.Validate(m))
		require.Equal(t, names, propertyRowOwnerNames(m))
	}
	inventory := map[string]propertyRowInventory{}
	for _, name := range names {
		if selected != nil {
			if _, ok := selected[name]; !ok {
				continue
			}
		}
		inventory[name] = walkPropertyRowOwner(t, models, name, selected[name], selected != nil, visit)
	}
	return inventory
}

// A phase owns every table and Scenario-name registry it uses; none survives into the next owner.
func walkPropertyRowOwner(t *testing.T, models []*umpirespb.Model, name string, selected []string, selecting bool,
	visit func(string, *interp.Table, interp.Row, []rowSide)) propertyRowInventory {
	t.Helper()
	var walks []*propertyRowWalk
	for _, m := range models {
		walks = append(walks, newPropertyRowWalk(t, m, name))
	}
	owner := walks[0].owner
	for _, w := range walks[1:] {
		require.Equal(t, owner.name, w.owner.name)
		require.Len(t, w.owner.rows, len(owner.rows), owner.name)
		require.Len(t, w.owner.properties, len(owner.properties), owner.name)
	}
	for j, row := range owner.rows {
		if selecting && !slices.Contains(selected, row.Key) {
			continue
		}
		if len(owner.properties) == 0 {
			continue
		}
		scenarios := make([]*umpirespb.Scenario, len(walks))
		for n, w := range walks {
			other := w.owner.rows[j]
			require.Equal(t, row.Key, other.Key, owner.name)
			scenarios[n] = w.prepare(t, w.owner, other)
		}
		for k, p := range owner.properties {
			key := rowKeyOf(p.GetName(), owner.name, row.Key)
			sides := make([]rowSide, len(walks))
			for n, w := range walks {
				other := w.owner.properties[k]
				if p.GetName() != other.GetName() {
					require.Equal(t, p.GetName(), other.GetName(), key)
				}
				if p.GetMachine() != other.GetMachine() {
					require.Equal(t, p.GetMachine(), other.GetMachine(), key)
				}
				sides[n] = w.answer(t, w.owner, other, scenarios[n], w.owner.rows[j])
			}
			visit(key, owner.on.table, row, sides)
		}
	}
	inventory := propertyRowInventory{Rows: len(owner.rows), Properties: len(owner.properties), Machine: owner.on.machine != nil}
	if inventory.Machine {
		inventory.Refined = owner.on.machine.Decl.GetRefines().GetProduct()
	}
	return inventory
}

func irPropertyRows(t *testing.T, base *umpirespb.Model, selected map[string][]string) map[string]rowSide {
	t.Helper()
	out := map[string]rowSide{}
	walkPropertyRows(t, []*umpirespb.Model{base}, selected, func(key string, _ *interp.Table, _ interp.Row, sides []rowSide) {
		require.NotContains(t, out, key)
		out[key] = sides[0]
	})
	return out
}

// rowDisagreements is every row answer of got that is not want's, in key order.
func rowDisagreements(want, got map[string]rowSide) []string {
	var out []string
	for key, w := range want {
		g, ok := got[key]
		switch {
		case !ok:
			out = append(out, key+": no answer")
		default:
			if difference := rowDifference(key, w, g); difference != "" {
				out = append(out, difference)
			}
		}
	}
	for key := range got {
		if _, ok := want[key]; !ok {
			out = append(out, key+": no answer before")
		}
	}
	slices.Sort(out)
	return out
}

func rowDifference(key string, want, got rowSide) string {
	switch {
	case want.Outcome != got.Outcome || want.About != got.About:
		return fmt.Sprintf("%s: %s (about the step: %t), now %s (about the step: %t)", key, want.Outcome, want.About,
			got.Outcome, got.About)
	case want.Explored != got.Explored || want.Expanded != got.Expanded || !slices.Equal(want.Rows, got.Rows):
		return key + ": the search or witness rows differ"
	case !sameTrace(want.Witness, got.Witness):
		return key + ": the witnesses differ"
	case !slices.EqualFunc(want.Results, got.Results, func(a, b interp.Result) bool {
		return a.Outcome == b.Outcome && a.State == b.State && slices.Equal(a.Facts, b.Facts) && a.Because == b.Because && a.Choice == b.Choice
	}):
		return key + ": the counterexample results differ"
	default:
		return ""
	}
}

func propertyRowDisagreements(t *testing.T, want, got *umpirespb.Model) []string {
	t.Helper()
	var out []string
	walkPropertyRows(t, []*umpirespb.Model{want, got}, nil, func(key string, _ *interp.Table, _ interp.Row, sides []rowSide) {
		if difference := rowDifference(key, sides[0], sides[1]); difference != "" {
			out = append(out, difference)
		}
	})
	slices.Sort(out)
	return out
}

func sameTrace(a, b *Trace) bool {
	if a == nil || b == nil {
		return a == b
	}
	return a.Initial == b.Initial && slices.EqualFunc(a.Steps, b.Steps, func(x, y TraceStep) bool {
		return x.Action == y.Action && x.Outcome == y.Outcome && x.State == y.State && slices.Equal(x.Facts, y.Facts)
	})
}

// propertyTally is how a Property answers the rows of one machine or composition: on how many the
// step is not what it is about, on how many it holds and on how many it fails.
type propertyTally struct{ NotAbout, Holds, Fails int }

type propertyRowsSummary struct {
	Rows    int
	Tally   map[string]propertyTally
	Failing map[string]int
	Digests map[string]string
	Owners  map[string]propertyRowInventory
}

func summarizePropertyRows(t *testing.T, m *umpirespb.Model) propertyRowsSummary {
	return summarizeSelectedPropertyRows(t, m, nil)
}

func summarizeSelectedPropertyRows(t *testing.T, m *umpirespb.Model, selected map[string][]string) propertyRowsSummary {
	t.Helper()
	out := propertyRowsSummary{Tally: map[string]propertyTally{}, Failing: map[string]int{}, Digests: map[string]string{}}
	hashes := map[string]hash.Hash{}
	out.Owners = walkPropertyRows(t, []*umpirespb.Model{m}, selected, func(key string, table *interp.Table, row interp.Row, sides []rowSide) {
		side := sides[0]
		claim, at, _ := strings.Cut(key, " at ")
		h, ok := hashes[claim]
		if !ok {
			h = sha256.New()
			hashes[claim] = h
		} else {
			_, err := h.Write([]byte("\n"))
			require.NoError(t, err)
		}
		_, err := fmt.Fprintf(h, "%s: %t %s", at, side.About, side.Outcome)
		require.NoError(t, err)
		counts := out.Tally[claim]
		switch {
		case !side.About:
			counts.NotAbout++
		case side.Outcome == umpire.VerifiedWithinLimits:
			counts.Holds++
		default:
			counts.Fails++
			require.Equal(t, umpire.CounterexampleFound, side.Outcome, key)
			require.True(t, slices.ContainsFunc(row.Results, func(result interp.Result) bool {
				return sameTrace(rowStep(t, table, row, result), side.Witness)
			}), "%s: the counterexample is not a result of the row", key)
			require.NotEmpty(t, side.Results, key)
			phase, _, _ := strings.Cut(row.Source, "-")
			out.Failing[claim+": "+phase+" "+row.Action]++
		}
		out.Tally[claim] = counts
		out.Rows++
	})
	for claim, h := range hashes {
		out.Digests[claim] = hex.EncodeToString(h.Sum(nil)[:8])
	}
	return out
}

// Every Property of the Model is asked on every row of its machine, of the machine that refines it and
// of the composition, and says of each row what is pinned here: how many steps it is about and holds
// on, and on which rows it fails, by the phase of the row's state and the row's class. A row a
// Property fails on is its own counterexample: the witness is the row's one step.
func TestActivityPropertiesOnEveryRow(t *testing.T) {
	m := activityModel(t)
	all := summarizePropertyRows(t, m)
	require.Len(t, m.GetMachines(), 20)
	require.Len(t, m.GetProperties(), 65)
	owners := map[string]int{}
	for _, p := range m.GetProperties() {
		owners[p.GetMachine()]++
	}
	require.Equal(t, map[string]int{
		"activityProduct": 3, "activitySystem": 29, "byIDCancellation": 2,
		"completion": 1, "retryFailures": 3, "cancellation": 2,
		"pausing": 1, "dispatchEligibility": 3, "timeouts": 2,
		"byIDCompletion": 1, "byIDFailure": 1, "deferredReset": 1,
		"heartbeatCompletion": 1, "heartbeatExhaustion": 1, "heartbeatRetry": 1,
		"resetKeepingPause": 1, "resetSettlement": 9, "standaloneActivity": 1, "timeoutRetry": 2,
	}, owners)
	expected := 0
	machines := 0
	require.Len(t, all.Owners, len(m.GetMachines())+len(m.GetCompositions()))
	for name, inventory := range all.Owners {
		require.Equal(t, owners[name]+owners[inventory.Refined], inventory.Properties, name)
		expected += inventory.Properties * inventory.Rows
		if inventory.Machine {
			machines++
		}
	}
	require.Equal(t, 20, machines)
	require.Equal(t, propertyRowInventory{Rows: 2, Machine: true}, all.Owners["activityWorker"])
	require.Equal(t, propertyRowInventory{Rows: 3, Machine: true}, all.Owners["polling"])
	require.Equal(t, 9980920, expected)
	require.Equal(t, expected, all.Rows)
	tally, failing := all.Tally, all.Failing
	require.Equal(t, map[string]propertyTally{
		"activityProduct.closedIsRejectedUniformly on activityProduct":                   {0, 210, 0},
		"activityProduct.closedIsRejectedUniformly on activitySystem":                    {0, 119016, 2160},
		"activityProduct.closedIsRejectedUniformly on completion":                        {0, 119016, 2160},
		"activityProduct.closedIsRejectedUniformly on retryFailures":                     {0, 119016, 2160},
		"activityProduct.closedIsRejectedUniformly on cancellation":                      {0, 119016, 2160},
		"activityProduct.closedIsRejectedUniformly on pausing":                           {0, 119016, 2160},
		"activityProduct.closedIsRejectedUniformly on dispatchEligibility":               {0, 119016, 2160},
		"activityProduct.closedIsRejectedUniformly on timeouts":                          {0, 119016, 2160},
		"activityProduct.pausedIsNotDispatched on activityProduct":                       {0, 210, 0},
		"activityProduct.pausedIsNotDispatched on activitySystem":                        {0, 121176, 0},
		"activityProduct.pausedIsNotDispatched on completion":                            {0, 121176, 0},
		"activityProduct.pausedIsNotDispatched on retryFailures":                         {0, 121176, 0},
		"activityProduct.pausedIsNotDispatched on cancellation":                          {0, 121176, 0},
		"activityProduct.pausedIsNotDispatched on pausing":                               {0, 121176, 0},
		"activityProduct.pausedIsNotDispatched on dispatchEligibility":                   {0, 121176, 0},
		"activityProduct.pausedIsNotDispatched on timeouts":                              {0, 121176, 0},
		"activityProduct.terminalStatesAreFinal on activityProduct":                      {0, 210, 0},
		"activityProduct.terminalStatesAreFinal on activitySystem":                       {0, 121176, 0},
		"activityProduct.terminalStatesAreFinal on completion":                           {0, 121176, 0},
		"activityProduct.terminalStatesAreFinal on retryFailures":                        {0, 121176, 0},
		"activityProduct.terminalStatesAreFinal on cancellation":                         {0, 121176, 0},
		"activityProduct.terminalStatesAreFinal on pausing":                              {0, 121176, 0},
		"activityProduct.terminalStatesAreFinal on dispatchEligibility":                  {0, 121176, 0},
		"activityProduct.terminalStatesAreFinal on timeouts":                             {0, 121176, 0},
		"activitySystem.cancelIsRequested on byIDCancellation":                           {115992, 1728, 3456},
		"activitySystem.fatalFailure.attemptCountIsWithinPolicy on activitySystem":       {0, 112976, 8200},
		"activitySystem.fatalFailure.failureCancels on activitySystem":                   {119016, 2160, 0},
		"activitySystem.fatalFailure.failureEndsFailed on activitySystem":                {119016, 2160, 0},
		"activitySystem.fatalFailure.failurePauses on activitySystem":                    {119016, 2160, 0},
		"activitySystem.fatalFailure.failureReturnsToWaiting on activitySystem":          {119016, 2160, 0},
		"activitySystem.heartbeatDeadline.deadlinePauses on activitySystem":              {120096, 1080, 0},
		"activitySystem.heartbeatDeadline.deadlineReturnsToWaiting on activitySystem":    {120096, 1080, 0},
		"activitySystem.heartbeatDeadline.deadlineTimesOut on activitySystem":            {120096, 1080, 0},
		"activitySystem.heartbeatDeadline.firesInWindow on activitySystem":               {120096, 1080, 0},
		"activitySystem.retryableFailure.attemptCountIsWithinPolicy on activitySystem":   {0, 112976, 8200},
		"activitySystem.retryableFailure.failureCancels on activitySystem":               {119016, 2160, 0},
		"activitySystem.retryableFailure.failureEndsFailed on activitySystem":            {119016, 2160, 0},
		"activitySystem.retryableFailure.failurePauses on activitySystem":                {119016, 2160, 0},
		"activitySystem.retryableFailure.failureReturnsToWaiting on activitySystem":      {119016, 2160, 0},
		"activitySystem.scheduleToCloseDeadline.deadlineTimesOut on activitySystem":      {120168, 1008, 0},
		"activitySystem.scheduleToCloseDeadline.firesInWindow on activitySystem":         {120168, 1008, 0},
		"activitySystem.scheduleToStartDeadline.deadlineTimesOut on activitySystem":      {121104, 72, 0},
		"activitySystem.scheduleToStartDeadline.firesInWindow on activitySystem":         {121104, 72, 0},
		"activitySystem.startToCloseDeadline.deadlinePauses on activitySystem":           {120096, 1080, 0},
		"activitySystem.startToCloseDeadline.deadlineReturnsToWaiting on activitySystem": {120096, 1080, 0},
		"activitySystem.startToCloseDeadline.deadlineTimesOut on activitySystem":         {120096, 1080, 0},
		"activitySystem.startToCloseDeadline.firesInWindow on activitySystem":            {120096, 1080, 0},
		"activitySystem.terminateSettles on activitySystem":                              {115992, 3024, 2160},
		"canceledByID on byIDCancellation":                                               {115560, 432, 5184},
		"canceledByWorker on cancellation":                                               {119016, 432, 1728},
		"cancelIsNotUndone on activitySystem":                                            {0, 121176, 0},
		"cancellationReplacesReset on resetSettlement":                                   {115992, 1728, 3456},
		"cancelRequestedWhileStarted on cancellation":                                    {115992, 1728, 3456},
		"completedByID on byIDCompletion":                                                {115560, 3024, 2592},
		"completes on completion":                                                        {119016, 2160, 0},
		"completes on pausing":                                                           {119016, 2160, 0},
		"completes on dispatchEligibility":                                               {119016, 2160, 0},
		"completesAfterTimeout on timeoutRetry":                                          {119016, 5, 2155},
		"completionWins on resetSettlement":                                              {119016, 720, 1440},
		"controlPrecedence on activitySystem":                                            {0, 121176, 0},
		"dispatchRequiresReady on dispatchEligibility":                                   {0, 121176, 0},
		"failsAfterTimeout on timeoutRetry":                                              {119016, 2, 2158},
		"fatalFailureByID on byIDFailure":                                                {115560, 1296, 4320},
		"heartbeatCompletes on heartbeatCompletion":                                      {119016, 2160, 0},
		"heartbeatExhausts on heartbeatExhaustion":                                       {120096, 1080, 0},
		"heartbeatRetryCompletes on heartbeatRetry":                                      {119016, 2160, 0},
		"nonRetryableFails on retryFailures":                                             {119016, 1296, 864},
		"pauseLeavesResetPending on resetSettlement":                                     {115992, 432, 4752},
		"resetAttemptCompletes on deferredReset":                                         {119016, 2160, 0},
		"resetKeepsPause on resetSettlement":                                             {119016, 9, 2151},
		"resetKeepsPaused on activitySystem":                                             {115560, 5616, 0},
		"resetKeptPaused on resetKeepingPause":                                           {115560, 6, 5610},
		"resetOnExhaustion on resetSettlement":                                           {119016, 9, 2151},
		"resetOnFatal on resetSettlement":                                                {119016, 9, 2151},
		"resetOnTimeout on resetSettlement":                                              {120096, 9, 1071},
		"resetResumes on activitySystem":                                                 {115560, 5616, 0},
		"resetSettles on activitySystem":                                                 {0, 120600, 576},
		"resetStaysPending on resetSettlement":                                           {115560, 1296, 4320},
		"retryCompletes on retryFailures":                                                {119016, 5, 2155},
		"retryExhausts on retryFailures":                                                 {119016, 5, 2155},
		"scheduleToCloseStaysTerminal on resetSettlement":                                {120168, 1008, 0},
		"scheduleToStartFires on timeouts":                                               {121104, 72, 0},
		"scheduleToStartRequiresDispatch on dispatchEligibility":                         {0, 121176, 0},
		"startedByPollingWorker on standaloneActivity":                                   {43762, 96, 0},
		"startToCloseFires on timeouts":                                                  {120096, 360, 720},
		"terminated on activitySystem":                                                   {115992, 3024, 2160},
	}, tally)
	require.Equal(t, map[string]int{
		"activityProduct.closedIsRejectedUniformly on activitySystem: canceled stop":                                                432,
		"activityProduct.closedIsRejectedUniformly on completion: canceled stop":                                                    432,
		"activityProduct.closedIsRejectedUniformly on retryFailures: canceled stop":                                                 432,
		"activityProduct.closedIsRejectedUniformly on cancellation: canceled stop":                                                  432,
		"activityProduct.closedIsRejectedUniformly on pausing: canceled stop":                                                       432,
		"activityProduct.closedIsRejectedUniformly on dispatchEligibility: canceled stop":                                           432,
		"activityProduct.closedIsRejectedUniformly on timeouts: canceled stop":                                                      432,
		"activityProduct.closedIsRejectedUniformly on activitySystem: completed stop":                                               432,
		"activityProduct.closedIsRejectedUniformly on completion: completed stop":                                                   432,
		"activityProduct.closedIsRejectedUniformly on retryFailures: completed stop":                                                432,
		"activityProduct.closedIsRejectedUniformly on cancellation: completed stop":                                                 432,
		"activityProduct.closedIsRejectedUniformly on pausing: completed stop":                                                      432,
		"activityProduct.closedIsRejectedUniformly on dispatchEligibility: completed stop":                                          432,
		"activityProduct.closedIsRejectedUniformly on timeouts: completed stop":                                                     432,
		"activityProduct.closedIsRejectedUniformly on activitySystem: failed stop":                                                  432,
		"activityProduct.closedIsRejectedUniformly on completion: failed stop":                                                      432,
		"activityProduct.closedIsRejectedUniformly on retryFailures: failed stop":                                                   432,
		"activityProduct.closedIsRejectedUniformly on cancellation: failed stop":                                                    432,
		"activityProduct.closedIsRejectedUniformly on pausing: failed stop":                                                         432,
		"activityProduct.closedIsRejectedUniformly on dispatchEligibility: failed stop":                                             432,
		"activityProduct.closedIsRejectedUniformly on timeouts: failed stop":                                                        432,
		"activityProduct.closedIsRejectedUniformly on activitySystem: terminated stop":                                              432,
		"activityProduct.closedIsRejectedUniformly on completion: terminated stop":                                                  432,
		"activityProduct.closedIsRejectedUniformly on retryFailures: terminated stop":                                               432,
		"activityProduct.closedIsRejectedUniformly on cancellation: terminated stop":                                                432,
		"activityProduct.closedIsRejectedUniformly on pausing: terminated stop":                                                     432,
		"activityProduct.closedIsRejectedUniformly on dispatchEligibility: terminated stop":                                         432,
		"activityProduct.closedIsRejectedUniformly on timeouts: terminated stop":                                                    432,
		"activityProduct.closedIsRejectedUniformly on activitySystem: timedOut stop":                                                432,
		"activityProduct.closedIsRejectedUniformly on completion: timedOut stop":                                                    432,
		"activityProduct.closedIsRejectedUniformly on retryFailures: timedOut stop":                                                 432,
		"activityProduct.closedIsRejectedUniformly on cancellation: timedOut stop":                                                  432,
		"activityProduct.closedIsRejectedUniformly on pausing: timedOut stop":                                                       432,
		"activityProduct.closedIsRejectedUniformly on dispatchEligibility: timedOut stop":                                           432,
		"activityProduct.closedIsRejectedUniformly on timeouts: timedOut stop":                                                      432,
		"activitySystem.cancelIsRequested on byIDCancellation: canceled requestCancel":                                              432,
		"activitySystem.cancelIsRequested on byIDCancellation: cancelRequested requestCancel":                                       432,
		"activitySystem.cancelIsRequested on byIDCancellation: completed requestCancel":                                             432,
		"activitySystem.cancelIsRequested on byIDCancellation: failed requestCancel":                                                432,
		"activitySystem.cancelIsRequested on byIDCancellation: paused requestCancel":                                                432,
		"activitySystem.cancelIsRequested on byIDCancellation: scheduled requestCancel":                                             432,
		"activitySystem.cancelIsRequested on byIDCancellation: terminated requestCancel":                                            432,
		"activitySystem.cancelIsRequested on byIDCancellation: timedOut requestCancel":                                              432,
		"activitySystem.fatalFailure.attemptCountIsWithinPolicy on activitySystem: canceled pause":                                  48,
		"activitySystem.fatalFailure.attemptCountIsWithinPolicy on activitySystem: canceled recordHeartbeat":                        48,
		"activitySystem.fatalFailure.attemptCountIsWithinPolicy on activitySystem: canceled requestCancel":                          48,
		"activitySystem.fatalFailure.attemptCountIsWithinPolicy on activitySystem: canceled reset-keepPaused":                       48,
		"activitySystem.fatalFailure.attemptCountIsWithinPolicy on activitySystem: canceled reset-resume":                           48,
		"activitySystem.fatalFailure.attemptCountIsWithinPolicy on activitySystem: canceled respondCanceledByID":                    48,
		"activitySystem.fatalFailure.attemptCountIsWithinPolicy on activitySystem: canceled respondCompletedByID":                   48,
		"activitySystem.fatalFailure.attemptCountIsWithinPolicy on activitySystem: canceled respondFailedByID-fatal":                48,
		"activitySystem.fatalFailure.attemptCountIsWithinPolicy on activitySystem: canceled respondFailedByID-retryable":            48,
		"activitySystem.fatalFailure.attemptCountIsWithinPolicy on activitySystem: canceled stop":                                   48,
		"activitySystem.fatalFailure.attemptCountIsWithinPolicy on activitySystem: canceled terminate":                              48,
		"activitySystem.fatalFailure.attemptCountIsWithinPolicy on activitySystem: canceled unpause":                                48,
		"activitySystem.fatalFailure.attemptCountIsWithinPolicy on activitySystem: cancelRequested backoff":                         16,
		"activitySystem.fatalFailure.attemptCountIsWithinPolicy on activitySystem: cancelRequested heartbeat":                       24,
		"activitySystem.fatalFailure.attemptCountIsWithinPolicy on activitySystem: cancelRequested pause":                           48,
		"activitySystem.fatalFailure.attemptCountIsWithinPolicy on activitySystem: cancelRequested recordHeartbeat":                 48,
		"activitySystem.fatalFailure.attemptCountIsWithinPolicy on activitySystem: cancelRequested requestCancel":                   48,
		"activitySystem.fatalFailure.attemptCountIsWithinPolicy on activitySystem: cancelRequested reset-keepPaused":                48,
		"activitySystem.fatalFailure.attemptCountIsWithinPolicy on activitySystem: cancelRequested reset-resume":                    48,
		"activitySystem.fatalFailure.attemptCountIsWithinPolicy on activitySystem: cancelRequested respondCanceled":                 48,
		"activitySystem.fatalFailure.attemptCountIsWithinPolicy on activitySystem: cancelRequested respondCanceledByID":             48,
		"activitySystem.fatalFailure.attemptCountIsWithinPolicy on activitySystem: cancelRequested respondCompleted":                48,
		"activitySystem.fatalFailure.attemptCountIsWithinPolicy on activitySystem: cancelRequested respondCompletedByID":            48,
		"activitySystem.fatalFailure.attemptCountIsWithinPolicy on activitySystem: cancelRequested respondFailed-fatal":             48,
		"activitySystem.fatalFailure.attemptCountIsWithinPolicy on activitySystem: cancelRequested respondFailed-retryable":         48,
		"activitySystem.fatalFailure.attemptCountIsWithinPolicy on activitySystem: cancelRequested respondFailedByID-fatal":         48,
		"activitySystem.fatalFailure.attemptCountIsWithinPolicy on activitySystem: cancelRequested respondFailedByID-retryable":     48,
		"activitySystem.fatalFailure.attemptCountIsWithinPolicy on activitySystem: cancelRequested scheduleToClose":                 16,
		"activitySystem.fatalFailure.attemptCountIsWithinPolicy on activitySystem: cancelRequested startDelay":                      16,
		"activitySystem.fatalFailure.attemptCountIsWithinPolicy on activitySystem: cancelRequested startToClose":                    24,
		"activitySystem.fatalFailure.attemptCountIsWithinPolicy on activitySystem: cancelRequested stop":                            48,
		"activitySystem.fatalFailure.attemptCountIsWithinPolicy on activitySystem: cancelRequested terminate":                       48,
		"activitySystem.fatalFailure.attemptCountIsWithinPolicy on activitySystem: cancelRequested unpause":                         48,
		"activitySystem.fatalFailure.attemptCountIsWithinPolicy on activitySystem: completed pause":                                 48,
		"activitySystem.fatalFailure.attemptCountIsWithinPolicy on activitySystem: completed recordHeartbeat":                       48,
		"activitySystem.fatalFailure.attemptCountIsWithinPolicy on activitySystem: completed requestCancel":                         48,
		"activitySystem.fatalFailure.attemptCountIsWithinPolicy on activitySystem: completed reset-keepPaused":                      48,
		"activitySystem.fatalFailure.attemptCountIsWithinPolicy on activitySystem: completed reset-resume":                          48,
		"activitySystem.fatalFailure.attemptCountIsWithinPolicy on activitySystem: completed respondCanceledByID":                   48,
		"activitySystem.fatalFailure.attemptCountIsWithinPolicy on activitySystem: completed respondCompletedByID":                  48,
		"activitySystem.fatalFailure.attemptCountIsWithinPolicy on activitySystem: completed respondFailedByID-fatal":               48,
		"activitySystem.fatalFailure.attemptCountIsWithinPolicy on activitySystem: completed respondFailedByID-retryable":           48,
		"activitySystem.fatalFailure.attemptCountIsWithinPolicy on activitySystem: completed stop":                                  48,
		"activitySystem.fatalFailure.attemptCountIsWithinPolicy on activitySystem: completed terminate":                             48,
		"activitySystem.fatalFailure.attemptCountIsWithinPolicy on activitySystem: completed unpause":                               48,
		"activitySystem.fatalFailure.attemptCountIsWithinPolicy on activitySystem: failed pause":                                    48,
		"activitySystem.fatalFailure.attemptCountIsWithinPolicy on activitySystem: failed recordHeartbeat":                          48,
		"activitySystem.fatalFailure.attemptCountIsWithinPolicy on activitySystem: failed requestCancel":                            48,
		"activitySystem.fatalFailure.attemptCountIsWithinPolicy on activitySystem: failed reset-keepPaused":                         48,
		"activitySystem.fatalFailure.attemptCountIsWithinPolicy on activitySystem: failed reset-resume":                             48,
		"activitySystem.fatalFailure.attemptCountIsWithinPolicy on activitySystem: failed respondCanceledByID":                      48,
		"activitySystem.fatalFailure.attemptCountIsWithinPolicy on activitySystem: failed respondCompletedByID":                     48,
		"activitySystem.fatalFailure.attemptCountIsWithinPolicy on activitySystem: failed respondFailedByID-fatal":                  48,
		"activitySystem.fatalFailure.attemptCountIsWithinPolicy on activitySystem: failed respondFailedByID-retryable":              48,
		"activitySystem.fatalFailure.attemptCountIsWithinPolicy on activitySystem: failed stop":                                     48,
		"activitySystem.fatalFailure.attemptCountIsWithinPolicy on activitySystem: failed terminate":                                48,
		"activitySystem.fatalFailure.attemptCountIsWithinPolicy on activitySystem: failed unpause":                                  48,
		"activitySystem.fatalFailure.attemptCountIsWithinPolicy on activitySystem: paused backoff":                                  16,
		"activitySystem.fatalFailure.attemptCountIsWithinPolicy on activitySystem: paused pause":                                    48,
		"activitySystem.fatalFailure.attemptCountIsWithinPolicy on activitySystem: paused recordHeartbeat":                          48,
		"activitySystem.fatalFailure.attemptCountIsWithinPolicy on activitySystem: paused requestCancel":                            48,
		"activitySystem.fatalFailure.attemptCountIsWithinPolicy on activitySystem: paused respondCanceledByID":                      48,
		"activitySystem.fatalFailure.attemptCountIsWithinPolicy on activitySystem: paused respondCompletedByID":                     48,
		"activitySystem.fatalFailure.attemptCountIsWithinPolicy on activitySystem: paused respondFailedByID-fatal":                  48,
		"activitySystem.fatalFailure.attemptCountIsWithinPolicy on activitySystem: paused respondFailedByID-retryable":              48,
		"activitySystem.fatalFailure.attemptCountIsWithinPolicy on activitySystem: paused scheduleToClose":                          16,
		"activitySystem.fatalFailure.attemptCountIsWithinPolicy on activitySystem: paused startDelay":                               16,
		"activitySystem.fatalFailure.attemptCountIsWithinPolicy on activitySystem: paused stop":                                     48,
		"activitySystem.fatalFailure.attemptCountIsWithinPolicy on activitySystem: paused terminate":                                48,
		"activitySystem.fatalFailure.attemptCountIsWithinPolicy on activitySystem: paused unpause":                                  48,
		"activitySystem.fatalFailure.attemptCountIsWithinPolicy on activitySystem: pauseRequested backoff":                          16,
		"activitySystem.fatalFailure.attemptCountIsWithinPolicy on activitySystem: pauseRequested heartbeat":                        24,
		"activitySystem.fatalFailure.attemptCountIsWithinPolicy on activitySystem: pauseRequested pause":                            48,
		"activitySystem.fatalFailure.attemptCountIsWithinPolicy on activitySystem: pauseRequested recordHeartbeat":                  48,
		"activitySystem.fatalFailure.attemptCountIsWithinPolicy on activitySystem: pauseRequested requestCancel":                    48,
		"activitySystem.fatalFailure.attemptCountIsWithinPolicy on activitySystem: pauseRequested reset-keepPaused":                 48,
		"activitySystem.fatalFailure.attemptCountIsWithinPolicy on activitySystem: pauseRequested reset-resume":                     48,
		"activitySystem.fatalFailure.attemptCountIsWithinPolicy on activitySystem: pauseRequested respondCanceled":                  48,
		"activitySystem.fatalFailure.attemptCountIsWithinPolicy on activitySystem: pauseRequested respondCanceledByID":              48,
		"activitySystem.fatalFailure.attemptCountIsWithinPolicy on activitySystem: pauseRequested respondCompleted":                 48,
		"activitySystem.fatalFailure.attemptCountIsWithinPolicy on activitySystem: pauseRequested respondCompletedByID":             48,
		"activitySystem.fatalFailure.attemptCountIsWithinPolicy on activitySystem: pauseRequested respondFailed-fatal":              48,
		"activitySystem.fatalFailure.attemptCountIsWithinPolicy on activitySystem: pauseRequested respondFailed-retryable":          48,
		"activitySystem.fatalFailure.attemptCountIsWithinPolicy on activitySystem: pauseRequested respondFailedByID-fatal":          48,
		"activitySystem.fatalFailure.attemptCountIsWithinPolicy on activitySystem: pauseRequested respondFailedByID-retryable":      48,
		"activitySystem.fatalFailure.attemptCountIsWithinPolicy on activitySystem: pauseRequested scheduleToClose":                  16,
		"activitySystem.fatalFailure.attemptCountIsWithinPolicy on activitySystem: pauseRequested startDelay":                       16,
		"activitySystem.fatalFailure.attemptCountIsWithinPolicy on activitySystem: pauseRequested startToClose":                     24,
		"activitySystem.fatalFailure.attemptCountIsWithinPolicy on activitySystem: pauseRequested stop":                             48,
		"activitySystem.fatalFailure.attemptCountIsWithinPolicy on activitySystem: pauseRequested terminate":                        48,
		"activitySystem.fatalFailure.attemptCountIsWithinPolicy on activitySystem: pauseRequested unpause":                          48,
		"activitySystem.fatalFailure.attemptCountIsWithinPolicy on activitySystem: resetKeepingPause backoff":                       16,
		"activitySystem.fatalFailure.attemptCountIsWithinPolicy on activitySystem: resetKeepingPause pause":                         48,
		"activitySystem.fatalFailure.attemptCountIsWithinPolicy on activitySystem: resetKeepingPause recordHeartbeat":               48,
		"activitySystem.fatalFailure.attemptCountIsWithinPolicy on activitySystem: resetKeepingPause requestCancel":                 48,
		"activitySystem.fatalFailure.attemptCountIsWithinPolicy on activitySystem: resetKeepingPause reset-keepPaused":              48,
		"activitySystem.fatalFailure.attemptCountIsWithinPolicy on activitySystem: resetKeepingPause reset-resume":                  48,
		"activitySystem.fatalFailure.attemptCountIsWithinPolicy on activitySystem: resetKeepingPause respondCanceled":               48,
		"activitySystem.fatalFailure.attemptCountIsWithinPolicy on activitySystem: resetKeepingPause respondCanceledByID":           48,
		"activitySystem.fatalFailure.attemptCountIsWithinPolicy on activitySystem: resetKeepingPause respondCompleted":              48,
		"activitySystem.fatalFailure.attemptCountIsWithinPolicy on activitySystem: resetKeepingPause respondCompletedByID":          48,
		"activitySystem.fatalFailure.attemptCountIsWithinPolicy on activitySystem: resetKeepingPause scheduleToClose":               16,
		"activitySystem.fatalFailure.attemptCountIsWithinPolicy on activitySystem: resetKeepingPause startDelay":                    16,
		"activitySystem.fatalFailure.attemptCountIsWithinPolicy on activitySystem: resetKeepingPause stop":                          48,
		"activitySystem.fatalFailure.attemptCountIsWithinPolicy on activitySystem: resetKeepingPause terminate":                     48,
		"activitySystem.fatalFailure.attemptCountIsWithinPolicy on activitySystem: resetKeepingPause unpause":                       48,
		"activitySystem.fatalFailure.attemptCountIsWithinPolicy on activitySystem: resetRequested backoff":                          16,
		"activitySystem.fatalFailure.attemptCountIsWithinPolicy on activitySystem: resetRequested pause":                            48,
		"activitySystem.fatalFailure.attemptCountIsWithinPolicy on activitySystem: resetRequested recordHeartbeat":                  48,
		"activitySystem.fatalFailure.attemptCountIsWithinPolicy on activitySystem: resetRequested requestCancel":                    48,
		"activitySystem.fatalFailure.attemptCountIsWithinPolicy on activitySystem: resetRequested reset-keepPaused":                 48,
		"activitySystem.fatalFailure.attemptCountIsWithinPolicy on activitySystem: resetRequested reset-resume":                     48,
		"activitySystem.fatalFailure.attemptCountIsWithinPolicy on activitySystem: resetRequested respondCanceled":                  48,
		"activitySystem.fatalFailure.attemptCountIsWithinPolicy on activitySystem: resetRequested respondCanceledByID":              48,
		"activitySystem.fatalFailure.attemptCountIsWithinPolicy on activitySystem: resetRequested respondCompleted":                 48,
		"activitySystem.fatalFailure.attemptCountIsWithinPolicy on activitySystem: resetRequested respondCompletedByID":             48,
		"activitySystem.fatalFailure.attemptCountIsWithinPolicy on activitySystem: resetRequested scheduleToClose":                  16,
		"activitySystem.fatalFailure.attemptCountIsWithinPolicy on activitySystem: resetRequested startDelay":                       16,
		"activitySystem.fatalFailure.attemptCountIsWithinPolicy on activitySystem: resetRequested stop":                             48,
		"activitySystem.fatalFailure.attemptCountIsWithinPolicy on activitySystem: resetRequested terminate":                        48,
		"activitySystem.fatalFailure.attemptCountIsWithinPolicy on activitySystem: resetRequested unpause":                          48,
		"activitySystem.fatalFailure.attemptCountIsWithinPolicy on activitySystem: scheduled backoff":                               16,
		"activitySystem.fatalFailure.attemptCountIsWithinPolicy on activitySystem: scheduled pause":                                 48,
		"activitySystem.fatalFailure.attemptCountIsWithinPolicy on activitySystem: scheduled poll":                                  32,
		"activitySystem.fatalFailure.attemptCountIsWithinPolicy on activitySystem: scheduled recordHeartbeat":                       48,
		"activitySystem.fatalFailure.attemptCountIsWithinPolicy on activitySystem: scheduled requestCancel":                         48,
		"activitySystem.fatalFailure.attemptCountIsWithinPolicy on activitySystem: scheduled respondCanceledByID":                   48,
		"activitySystem.fatalFailure.attemptCountIsWithinPolicy on activitySystem: scheduled respondCompletedByID":                  48,
		"activitySystem.fatalFailure.attemptCountIsWithinPolicy on activitySystem: scheduled respondFailedByID-fatal":               48,
		"activitySystem.fatalFailure.attemptCountIsWithinPolicy on activitySystem: scheduled respondFailedByID-retryable":           48,
		"activitySystem.fatalFailure.attemptCountIsWithinPolicy on activitySystem: scheduled scheduleToClose":                       16,
		"activitySystem.fatalFailure.attemptCountIsWithinPolicy on activitySystem: scheduled scheduleToStart":                       8,
		"activitySystem.fatalFailure.attemptCountIsWithinPolicy on activitySystem: scheduled startDelay":                            16,
		"activitySystem.fatalFailure.attemptCountIsWithinPolicy on activitySystem: scheduled stop":                                  48,
		"activitySystem.fatalFailure.attemptCountIsWithinPolicy on activitySystem: scheduled terminate":                             48,
		"activitySystem.fatalFailure.attemptCountIsWithinPolicy on activitySystem: scheduled unpause":                               48,
		"activitySystem.fatalFailure.attemptCountIsWithinPolicy on activitySystem: started backoff":                                 16,
		"activitySystem.fatalFailure.attemptCountIsWithinPolicy on activitySystem: started heartbeat":                               24,
		"activitySystem.fatalFailure.attemptCountIsWithinPolicy on activitySystem: started pause":                                   48,
		"activitySystem.fatalFailure.attemptCountIsWithinPolicy on activitySystem: started recordHeartbeat":                         48,
		"activitySystem.fatalFailure.attemptCountIsWithinPolicy on activitySystem: started requestCancel":                           48,
		"activitySystem.fatalFailure.attemptCountIsWithinPolicy on activitySystem: started reset-keepPaused":                        48,
		"activitySystem.fatalFailure.attemptCountIsWithinPolicy on activitySystem: started reset-resume":                            48,
		"activitySystem.fatalFailure.attemptCountIsWithinPolicy on activitySystem: started respondCanceled":                         48,
		"activitySystem.fatalFailure.attemptCountIsWithinPolicy on activitySystem: started respondCanceledByID":                     48,
		"activitySystem.fatalFailure.attemptCountIsWithinPolicy on activitySystem: started respondCompleted":                        48,
		"activitySystem.fatalFailure.attemptCountIsWithinPolicy on activitySystem: started respondCompletedByID":                    48,
		"activitySystem.fatalFailure.attemptCountIsWithinPolicy on activitySystem: started respondFailed-fatal":                     48,
		"activitySystem.fatalFailure.attemptCountIsWithinPolicy on activitySystem: started respondFailed-retryable":                 48,
		"activitySystem.fatalFailure.attemptCountIsWithinPolicy on activitySystem: started respondFailedByID-fatal":                 48,
		"activitySystem.fatalFailure.attemptCountIsWithinPolicy on activitySystem: started respondFailedByID-retryable":             48,
		"activitySystem.fatalFailure.attemptCountIsWithinPolicy on activitySystem: started scheduleToClose":                         16,
		"activitySystem.fatalFailure.attemptCountIsWithinPolicy on activitySystem: started startDelay":                              16,
		"activitySystem.fatalFailure.attemptCountIsWithinPolicy on activitySystem: started startToClose":                            24,
		"activitySystem.fatalFailure.attemptCountIsWithinPolicy on activitySystem: started stop":                                    48,
		"activitySystem.fatalFailure.attemptCountIsWithinPolicy on activitySystem: started terminate":                               48,
		"activitySystem.fatalFailure.attemptCountIsWithinPolicy on activitySystem: started unpause":                                 48,
		"activitySystem.fatalFailure.attemptCountIsWithinPolicy on activitySystem: terminated pause":                                48,
		"activitySystem.fatalFailure.attemptCountIsWithinPolicy on activitySystem: terminated recordHeartbeat":                      48,
		"activitySystem.fatalFailure.attemptCountIsWithinPolicy on activitySystem: terminated requestCancel":                        48,
		"activitySystem.fatalFailure.attemptCountIsWithinPolicy on activitySystem: terminated reset-keepPaused":                     48,
		"activitySystem.fatalFailure.attemptCountIsWithinPolicy on activitySystem: terminated reset-resume":                         48,
		"activitySystem.fatalFailure.attemptCountIsWithinPolicy on activitySystem: terminated respondCanceledByID":                  48,
		"activitySystem.fatalFailure.attemptCountIsWithinPolicy on activitySystem: terminated respondCompletedByID":                 48,
		"activitySystem.fatalFailure.attemptCountIsWithinPolicy on activitySystem: terminated respondFailedByID-fatal":              48,
		"activitySystem.fatalFailure.attemptCountIsWithinPolicy on activitySystem: terminated respondFailedByID-retryable":          48,
		"activitySystem.fatalFailure.attemptCountIsWithinPolicy on activitySystem: terminated stop":                                 48,
		"activitySystem.fatalFailure.attemptCountIsWithinPolicy on activitySystem: terminated terminate":                            48,
		"activitySystem.fatalFailure.attemptCountIsWithinPolicy on activitySystem: terminated unpause":                              48,
		"activitySystem.fatalFailure.attemptCountIsWithinPolicy on activitySystem: timedOut pause":                                  48,
		"activitySystem.fatalFailure.attemptCountIsWithinPolicy on activitySystem: timedOut recordHeartbeat":                        48,
		"activitySystem.fatalFailure.attemptCountIsWithinPolicy on activitySystem: timedOut requestCancel":                          48,
		"activitySystem.fatalFailure.attemptCountIsWithinPolicy on activitySystem: timedOut reset-keepPaused":                       48,
		"activitySystem.fatalFailure.attemptCountIsWithinPolicy on activitySystem: timedOut reset-resume":                           48,
		"activitySystem.fatalFailure.attemptCountIsWithinPolicy on activitySystem: timedOut respondCanceledByID":                    48,
		"activitySystem.fatalFailure.attemptCountIsWithinPolicy on activitySystem: timedOut respondCompletedByID":                   48,
		"activitySystem.fatalFailure.attemptCountIsWithinPolicy on activitySystem: timedOut respondFailedByID-fatal":                48,
		"activitySystem.fatalFailure.attemptCountIsWithinPolicy on activitySystem: timedOut respondFailedByID-retryable":            48,
		"activitySystem.fatalFailure.attemptCountIsWithinPolicy on activitySystem: timedOut stop":                                   48,
		"activitySystem.fatalFailure.attemptCountIsWithinPolicy on activitySystem: timedOut terminate":                              48,
		"activitySystem.fatalFailure.attemptCountIsWithinPolicy on activitySystem: timedOut unpause":                                48,
		"activitySystem.fatalFailure.attemptCountIsWithinPolicy on activitySystem: unstarted recordHeartbeat":                       48,
		"activitySystem.fatalFailure.attemptCountIsWithinPolicy on activitySystem: unstarted reset-keepPaused":                      48,
		"activitySystem.fatalFailure.attemptCountIsWithinPolicy on activitySystem: unstarted reset-resume":                          48,
		"activitySystem.fatalFailure.attemptCountIsWithinPolicy on activitySystem: unstarted respondCanceledByID":                   48,
		"activitySystem.fatalFailure.attemptCountIsWithinPolicy on activitySystem: unstarted respondCompletedByID":                  48,
		"activitySystem.fatalFailure.attemptCountIsWithinPolicy on activitySystem: unstarted respondFailedByID-fatal":               48,
		"activitySystem.fatalFailure.attemptCountIsWithinPolicy on activitySystem: unstarted respondFailedByID-retryable":           48,
		"activitySystem.fatalFailure.attemptCountIsWithinPolicy on activitySystem: unstarted stop":                                  48,
		"activitySystem.retryableFailure.attemptCountIsWithinPolicy on activitySystem: canceled pause":                              48,
		"activitySystem.retryableFailure.attemptCountIsWithinPolicy on activitySystem: canceled recordHeartbeat":                    48,
		"activitySystem.retryableFailure.attemptCountIsWithinPolicy on activitySystem: canceled requestCancel":                      48,
		"activitySystem.retryableFailure.attemptCountIsWithinPolicy on activitySystem: canceled reset-keepPaused":                   48,
		"activitySystem.retryableFailure.attemptCountIsWithinPolicy on activitySystem: canceled reset-resume":                       48,
		"activitySystem.retryableFailure.attemptCountIsWithinPolicy on activitySystem: canceled respondCanceledByID":                48,
		"activitySystem.retryableFailure.attemptCountIsWithinPolicy on activitySystem: canceled respondCompletedByID":               48,
		"activitySystem.retryableFailure.attemptCountIsWithinPolicy on activitySystem: canceled respondFailedByID-fatal":            48,
		"activitySystem.retryableFailure.attemptCountIsWithinPolicy on activitySystem: canceled respondFailedByID-retryable":        48,
		"activitySystem.retryableFailure.attemptCountIsWithinPolicy on activitySystem: canceled stop":                               48,
		"activitySystem.retryableFailure.attemptCountIsWithinPolicy on activitySystem: canceled terminate":                          48,
		"activitySystem.retryableFailure.attemptCountIsWithinPolicy on activitySystem: canceled unpause":                            48,
		"activitySystem.retryableFailure.attemptCountIsWithinPolicy on activitySystem: cancelRequested backoff":                     16,
		"activitySystem.retryableFailure.attemptCountIsWithinPolicy on activitySystem: cancelRequested heartbeat":                   24,
		"activitySystem.retryableFailure.attemptCountIsWithinPolicy on activitySystem: cancelRequested pause":                       48,
		"activitySystem.retryableFailure.attemptCountIsWithinPolicy on activitySystem: cancelRequested recordHeartbeat":             48,
		"activitySystem.retryableFailure.attemptCountIsWithinPolicy on activitySystem: cancelRequested requestCancel":               48,
		"activitySystem.retryableFailure.attemptCountIsWithinPolicy on activitySystem: cancelRequested reset-keepPaused":            48,
		"activitySystem.retryableFailure.attemptCountIsWithinPolicy on activitySystem: cancelRequested reset-resume":                48,
		"activitySystem.retryableFailure.attemptCountIsWithinPolicy on activitySystem: cancelRequested respondCanceled":             48,
		"activitySystem.retryableFailure.attemptCountIsWithinPolicy on activitySystem: cancelRequested respondCanceledByID":         48,
		"activitySystem.retryableFailure.attemptCountIsWithinPolicy on activitySystem: cancelRequested respondCompleted":            48,
		"activitySystem.retryableFailure.attemptCountIsWithinPolicy on activitySystem: cancelRequested respondCompletedByID":        48,
		"activitySystem.retryableFailure.attemptCountIsWithinPolicy on activitySystem: cancelRequested respondFailed-fatal":         48,
		"activitySystem.retryableFailure.attemptCountIsWithinPolicy on activitySystem: cancelRequested respondFailed-retryable":     48,
		"activitySystem.retryableFailure.attemptCountIsWithinPolicy on activitySystem: cancelRequested respondFailedByID-fatal":     48,
		"activitySystem.retryableFailure.attemptCountIsWithinPolicy on activitySystem: cancelRequested respondFailedByID-retryable": 48,
		"activitySystem.retryableFailure.attemptCountIsWithinPolicy on activitySystem: cancelRequested scheduleToClose":             16,
		"activitySystem.retryableFailure.attemptCountIsWithinPolicy on activitySystem: cancelRequested startDelay":                  16,
		"activitySystem.retryableFailure.attemptCountIsWithinPolicy on activitySystem: cancelRequested startToClose":                24,
		"activitySystem.retryableFailure.attemptCountIsWithinPolicy on activitySystem: cancelRequested stop":                        48,
		"activitySystem.retryableFailure.attemptCountIsWithinPolicy on activitySystem: cancelRequested terminate":                   48,
		"activitySystem.retryableFailure.attemptCountIsWithinPolicy on activitySystem: cancelRequested unpause":                     48,
		"activitySystem.retryableFailure.attemptCountIsWithinPolicy on activitySystem: completed pause":                             48,
		"activitySystem.retryableFailure.attemptCountIsWithinPolicy on activitySystem: completed recordHeartbeat":                   48,
		"activitySystem.retryableFailure.attemptCountIsWithinPolicy on activitySystem: completed requestCancel":                     48,
		"activitySystem.retryableFailure.attemptCountIsWithinPolicy on activitySystem: completed reset-keepPaused":                  48,
		"activitySystem.retryableFailure.attemptCountIsWithinPolicy on activitySystem: completed reset-resume":                      48,
		"activitySystem.retryableFailure.attemptCountIsWithinPolicy on activitySystem: completed respondCanceledByID":               48,
		"activitySystem.retryableFailure.attemptCountIsWithinPolicy on activitySystem: completed respondCompletedByID":              48,
		"activitySystem.retryableFailure.attemptCountIsWithinPolicy on activitySystem: completed respondFailedByID-fatal":           48,
		"activitySystem.retryableFailure.attemptCountIsWithinPolicy on activitySystem: completed respondFailedByID-retryable":       48,
		"activitySystem.retryableFailure.attemptCountIsWithinPolicy on activitySystem: completed stop":                              48,
		"activitySystem.retryableFailure.attemptCountIsWithinPolicy on activitySystem: completed terminate":                         48,
		"activitySystem.retryableFailure.attemptCountIsWithinPolicy on activitySystem: completed unpause":                           48,
		"activitySystem.retryableFailure.attemptCountIsWithinPolicy on activitySystem: failed pause":                                48,
		"activitySystem.retryableFailure.attemptCountIsWithinPolicy on activitySystem: failed recordHeartbeat":                      48,
		"activitySystem.retryableFailure.attemptCountIsWithinPolicy on activitySystem: failed requestCancel":                        48,
		"activitySystem.retryableFailure.attemptCountIsWithinPolicy on activitySystem: failed reset-keepPaused":                     48,
		"activitySystem.retryableFailure.attemptCountIsWithinPolicy on activitySystem: failed reset-resume":                         48,
		"activitySystem.retryableFailure.attemptCountIsWithinPolicy on activitySystem: failed respondCanceledByID":                  48,
		"activitySystem.retryableFailure.attemptCountIsWithinPolicy on activitySystem: failed respondCompletedByID":                 48,
		"activitySystem.retryableFailure.attemptCountIsWithinPolicy on activitySystem: failed respondFailedByID-fatal":              48,
		"activitySystem.retryableFailure.attemptCountIsWithinPolicy on activitySystem: failed respondFailedByID-retryable":          48,
		"activitySystem.retryableFailure.attemptCountIsWithinPolicy on activitySystem: failed stop":                                 48,
		"activitySystem.retryableFailure.attemptCountIsWithinPolicy on activitySystem: failed terminate":                            48,
		"activitySystem.retryableFailure.attemptCountIsWithinPolicy on activitySystem: failed unpause":                              48,
		"activitySystem.retryableFailure.attemptCountIsWithinPolicy on activitySystem: paused backoff":                              16,
		"activitySystem.retryableFailure.attemptCountIsWithinPolicy on activitySystem: paused pause":                                48,
		"activitySystem.retryableFailure.attemptCountIsWithinPolicy on activitySystem: paused recordHeartbeat":                      48,
		"activitySystem.retryableFailure.attemptCountIsWithinPolicy on activitySystem: paused requestCancel":                        48,
		"activitySystem.retryableFailure.attemptCountIsWithinPolicy on activitySystem: paused respondCanceledByID":                  48,
		"activitySystem.retryableFailure.attemptCountIsWithinPolicy on activitySystem: paused respondCompletedByID":                 48,
		"activitySystem.retryableFailure.attemptCountIsWithinPolicy on activitySystem: paused respondFailedByID-fatal":              48,
		"activitySystem.retryableFailure.attemptCountIsWithinPolicy on activitySystem: paused respondFailedByID-retryable":          48,
		"activitySystem.retryableFailure.attemptCountIsWithinPolicy on activitySystem: paused scheduleToClose":                      16,
		"activitySystem.retryableFailure.attemptCountIsWithinPolicy on activitySystem: paused startDelay":                           16,
		"activitySystem.retryableFailure.attemptCountIsWithinPolicy on activitySystem: paused stop":                                 48,
		"activitySystem.retryableFailure.attemptCountIsWithinPolicy on activitySystem: paused terminate":                            48,
		"activitySystem.retryableFailure.attemptCountIsWithinPolicy on activitySystem: paused unpause":                              48,
		"activitySystem.retryableFailure.attemptCountIsWithinPolicy on activitySystem: pauseRequested backoff":                      16,
		"activitySystem.retryableFailure.attemptCountIsWithinPolicy on activitySystem: pauseRequested heartbeat":                    24,
		"activitySystem.retryableFailure.attemptCountIsWithinPolicy on activitySystem: pauseRequested pause":                        48,
		"activitySystem.retryableFailure.attemptCountIsWithinPolicy on activitySystem: pauseRequested recordHeartbeat":              48,
		"activitySystem.retryableFailure.attemptCountIsWithinPolicy on activitySystem: pauseRequested requestCancel":                48,
		"activitySystem.retryableFailure.attemptCountIsWithinPolicy on activitySystem: pauseRequested reset-keepPaused":             48,
		"activitySystem.retryableFailure.attemptCountIsWithinPolicy on activitySystem: pauseRequested reset-resume":                 48,
		"activitySystem.retryableFailure.attemptCountIsWithinPolicy on activitySystem: pauseRequested respondCanceled":              48,
		"activitySystem.retryableFailure.attemptCountIsWithinPolicy on activitySystem: pauseRequested respondCanceledByID":          48,
		"activitySystem.retryableFailure.attemptCountIsWithinPolicy on activitySystem: pauseRequested respondCompleted":             48,
		"activitySystem.retryableFailure.attemptCountIsWithinPolicy on activitySystem: pauseRequested respondCompletedByID":         48,
		"activitySystem.retryableFailure.attemptCountIsWithinPolicy on activitySystem: pauseRequested respondFailed-fatal":          48,
		"activitySystem.retryableFailure.attemptCountIsWithinPolicy on activitySystem: pauseRequested respondFailed-retryable":      48,
		"activitySystem.retryableFailure.attemptCountIsWithinPolicy on activitySystem: pauseRequested respondFailedByID-fatal":      48,
		"activitySystem.retryableFailure.attemptCountIsWithinPolicy on activitySystem: pauseRequested respondFailedByID-retryable":  48,
		"activitySystem.retryableFailure.attemptCountIsWithinPolicy on activitySystem: pauseRequested scheduleToClose":              16,
		"activitySystem.retryableFailure.attemptCountIsWithinPolicy on activitySystem: pauseRequested startDelay":                   16,
		"activitySystem.retryableFailure.attemptCountIsWithinPolicy on activitySystem: pauseRequested startToClose":                 24,
		"activitySystem.retryableFailure.attemptCountIsWithinPolicy on activitySystem: pauseRequested stop":                         48,
		"activitySystem.retryableFailure.attemptCountIsWithinPolicy on activitySystem: pauseRequested terminate":                    48,
		"activitySystem.retryableFailure.attemptCountIsWithinPolicy on activitySystem: pauseRequested unpause":                      48,
		"activitySystem.retryableFailure.attemptCountIsWithinPolicy on activitySystem: resetKeepingPause backoff":                   16,
		"activitySystem.retryableFailure.attemptCountIsWithinPolicy on activitySystem: resetKeepingPause pause":                     48,
		"activitySystem.retryableFailure.attemptCountIsWithinPolicy on activitySystem: resetKeepingPause recordHeartbeat":           48,
		"activitySystem.retryableFailure.attemptCountIsWithinPolicy on activitySystem: resetKeepingPause requestCancel":             48,
		"activitySystem.retryableFailure.attemptCountIsWithinPolicy on activitySystem: resetKeepingPause reset-keepPaused":          48,
		"activitySystem.retryableFailure.attemptCountIsWithinPolicy on activitySystem: resetKeepingPause reset-resume":              48,
		"activitySystem.retryableFailure.attemptCountIsWithinPolicy on activitySystem: resetKeepingPause respondCanceled":           48,
		"activitySystem.retryableFailure.attemptCountIsWithinPolicy on activitySystem: resetKeepingPause respondCanceledByID":       48,
		"activitySystem.retryableFailure.attemptCountIsWithinPolicy on activitySystem: resetKeepingPause respondCompleted":          48,
		"activitySystem.retryableFailure.attemptCountIsWithinPolicy on activitySystem: resetKeepingPause respondCompletedByID":      48,
		"activitySystem.retryableFailure.attemptCountIsWithinPolicy on activitySystem: resetKeepingPause scheduleToClose":           16,
		"activitySystem.retryableFailure.attemptCountIsWithinPolicy on activitySystem: resetKeepingPause startDelay":                16,
		"activitySystem.retryableFailure.attemptCountIsWithinPolicy on activitySystem: resetKeepingPause stop":                      48,
		"activitySystem.retryableFailure.attemptCountIsWithinPolicy on activitySystem: resetKeepingPause terminate":                 48,
		"activitySystem.retryableFailure.attemptCountIsWithinPolicy on activitySystem: resetKeepingPause unpause":                   48,
		"activitySystem.retryableFailure.attemptCountIsWithinPolicy on activitySystem: resetRequested backoff":                      16,
		"activitySystem.retryableFailure.attemptCountIsWithinPolicy on activitySystem: resetRequested pause":                        48,
		"activitySystem.retryableFailure.attemptCountIsWithinPolicy on activitySystem: resetRequested recordHeartbeat":              48,
		"activitySystem.retryableFailure.attemptCountIsWithinPolicy on activitySystem: resetRequested requestCancel":                48,
		"activitySystem.retryableFailure.attemptCountIsWithinPolicy on activitySystem: resetRequested reset-keepPaused":             48,
		"activitySystem.retryableFailure.attemptCountIsWithinPolicy on activitySystem: resetRequested reset-resume":                 48,
		"activitySystem.retryableFailure.attemptCountIsWithinPolicy on activitySystem: resetRequested respondCanceled":              48,
		"activitySystem.retryableFailure.attemptCountIsWithinPolicy on activitySystem: resetRequested respondCanceledByID":          48,
		"activitySystem.retryableFailure.attemptCountIsWithinPolicy on activitySystem: resetRequested respondCompleted":             48,
		"activitySystem.retryableFailure.attemptCountIsWithinPolicy on activitySystem: resetRequested respondCompletedByID":         48,
		"activitySystem.retryableFailure.attemptCountIsWithinPolicy on activitySystem: resetRequested scheduleToClose":              16,
		"activitySystem.retryableFailure.attemptCountIsWithinPolicy on activitySystem: resetRequested startDelay":                   16,
		"activitySystem.retryableFailure.attemptCountIsWithinPolicy on activitySystem: resetRequested stop":                         48,
		"activitySystem.retryableFailure.attemptCountIsWithinPolicy on activitySystem: resetRequested terminate":                    48,
		"activitySystem.retryableFailure.attemptCountIsWithinPolicy on activitySystem: resetRequested unpause":                      48,
		"activitySystem.retryableFailure.attemptCountIsWithinPolicy on activitySystem: scheduled backoff":                           16,
		"activitySystem.retryableFailure.attemptCountIsWithinPolicy on activitySystem: scheduled pause":                             48,
		"activitySystem.retryableFailure.attemptCountIsWithinPolicy on activitySystem: scheduled poll":                              32,
		"activitySystem.retryableFailure.attemptCountIsWithinPolicy on activitySystem: scheduled recordHeartbeat":                   48,
		"activitySystem.retryableFailure.attemptCountIsWithinPolicy on activitySystem: scheduled requestCancel":                     48,
		"activitySystem.retryableFailure.attemptCountIsWithinPolicy on activitySystem: scheduled respondCanceledByID":               48,
		"activitySystem.retryableFailure.attemptCountIsWithinPolicy on activitySystem: scheduled respondCompletedByID":              48,
		"activitySystem.retryableFailure.attemptCountIsWithinPolicy on activitySystem: scheduled respondFailedByID-fatal":           48,
		"activitySystem.retryableFailure.attemptCountIsWithinPolicy on activitySystem: scheduled respondFailedByID-retryable":       48,
		"activitySystem.retryableFailure.attemptCountIsWithinPolicy on activitySystem: scheduled scheduleToClose":                   16,
		"activitySystem.retryableFailure.attemptCountIsWithinPolicy on activitySystem: scheduled scheduleToStart":                   8,
		"activitySystem.retryableFailure.attemptCountIsWithinPolicy on activitySystem: scheduled startDelay":                        16,
		"activitySystem.retryableFailure.attemptCountIsWithinPolicy on activitySystem: scheduled stop":                              48,
		"activitySystem.retryableFailure.attemptCountIsWithinPolicy on activitySystem: scheduled terminate":                         48,
		"activitySystem.retryableFailure.attemptCountIsWithinPolicy on activitySystem: scheduled unpause":                           48,
		"activitySystem.retryableFailure.attemptCountIsWithinPolicy on activitySystem: started backoff":                             16,
		"activitySystem.retryableFailure.attemptCountIsWithinPolicy on activitySystem: started heartbeat":                           24,
		"activitySystem.retryableFailure.attemptCountIsWithinPolicy on activitySystem: started pause":                               48,
		"activitySystem.retryableFailure.attemptCountIsWithinPolicy on activitySystem: started recordHeartbeat":                     48,
		"activitySystem.retryableFailure.attemptCountIsWithinPolicy on activitySystem: started requestCancel":                       48,
		"activitySystem.retryableFailure.attemptCountIsWithinPolicy on activitySystem: started reset-keepPaused":                    48,
		"activitySystem.retryableFailure.attemptCountIsWithinPolicy on activitySystem: started reset-resume":                        48,
		"activitySystem.retryableFailure.attemptCountIsWithinPolicy on activitySystem: started respondCanceled":                     48,
		"activitySystem.retryableFailure.attemptCountIsWithinPolicy on activitySystem: started respondCanceledByID":                 48,
		"activitySystem.retryableFailure.attemptCountIsWithinPolicy on activitySystem: started respondCompleted":                    48,
		"activitySystem.retryableFailure.attemptCountIsWithinPolicy on activitySystem: started respondCompletedByID":                48,
		"activitySystem.retryableFailure.attemptCountIsWithinPolicy on activitySystem: started respondFailed-fatal":                 48,
		"activitySystem.retryableFailure.attemptCountIsWithinPolicy on activitySystem: started respondFailed-retryable":             48,
		"activitySystem.retryableFailure.attemptCountIsWithinPolicy on activitySystem: started respondFailedByID-fatal":             48,
		"activitySystem.retryableFailure.attemptCountIsWithinPolicy on activitySystem: started respondFailedByID-retryable":         48,
		"activitySystem.retryableFailure.attemptCountIsWithinPolicy on activitySystem: started scheduleToClose":                     16,
		"activitySystem.retryableFailure.attemptCountIsWithinPolicy on activitySystem: started startDelay":                          16,
		"activitySystem.retryableFailure.attemptCountIsWithinPolicy on activitySystem: started startToClose":                        24,
		"activitySystem.retryableFailure.attemptCountIsWithinPolicy on activitySystem: started stop":                                48,
		"activitySystem.retryableFailure.attemptCountIsWithinPolicy on activitySystem: started terminate":                           48,
		"activitySystem.retryableFailure.attemptCountIsWithinPolicy on activitySystem: started unpause":                             48,
		"activitySystem.retryableFailure.attemptCountIsWithinPolicy on activitySystem: terminated pause":                            48,
		"activitySystem.retryableFailure.attemptCountIsWithinPolicy on activitySystem: terminated recordHeartbeat":                  48,
		"activitySystem.retryableFailure.attemptCountIsWithinPolicy on activitySystem: terminated requestCancel":                    48,
		"activitySystem.retryableFailure.attemptCountIsWithinPolicy on activitySystem: terminated reset-keepPaused":                 48,
		"activitySystem.retryableFailure.attemptCountIsWithinPolicy on activitySystem: terminated reset-resume":                     48,
		"activitySystem.retryableFailure.attemptCountIsWithinPolicy on activitySystem: terminated respondCanceledByID":              48,
		"activitySystem.retryableFailure.attemptCountIsWithinPolicy on activitySystem: terminated respondCompletedByID":             48,
		"activitySystem.retryableFailure.attemptCountIsWithinPolicy on activitySystem: terminated respondFailedByID-fatal":          48,
		"activitySystem.retryableFailure.attemptCountIsWithinPolicy on activitySystem: terminated respondFailedByID-retryable":      48,
		"activitySystem.retryableFailure.attemptCountIsWithinPolicy on activitySystem: terminated stop":                             48,
		"activitySystem.retryableFailure.attemptCountIsWithinPolicy on activitySystem: terminated terminate":                        48,
		"activitySystem.retryableFailure.attemptCountIsWithinPolicy on activitySystem: terminated unpause":                          48,
		"activitySystem.retryableFailure.attemptCountIsWithinPolicy on activitySystem: timedOut pause":                              48,
		"activitySystem.retryableFailure.attemptCountIsWithinPolicy on activitySystem: timedOut recordHeartbeat":                    48,
		"activitySystem.retryableFailure.attemptCountIsWithinPolicy on activitySystem: timedOut requestCancel":                      48,
		"activitySystem.retryableFailure.attemptCountIsWithinPolicy on activitySystem: timedOut reset-keepPaused":                   48,
		"activitySystem.retryableFailure.attemptCountIsWithinPolicy on activitySystem: timedOut reset-resume":                       48,
		"activitySystem.retryableFailure.attemptCountIsWithinPolicy on activitySystem: timedOut respondCanceledByID":                48,
		"activitySystem.retryableFailure.attemptCountIsWithinPolicy on activitySystem: timedOut respondCompletedByID":               48,
		"activitySystem.retryableFailure.attemptCountIsWithinPolicy on activitySystem: timedOut respondFailedByID-fatal":            48,
		"activitySystem.retryableFailure.attemptCountIsWithinPolicy on activitySystem: timedOut respondFailedByID-retryable":        48,
		"activitySystem.retryableFailure.attemptCountIsWithinPolicy on activitySystem: timedOut stop":                               48,
		"activitySystem.retryableFailure.attemptCountIsWithinPolicy on activitySystem: timedOut terminate":                          48,
		"activitySystem.retryableFailure.attemptCountIsWithinPolicy on activitySystem: timedOut unpause":                            48,
		"activitySystem.retryableFailure.attemptCountIsWithinPolicy on activitySystem: unstarted recordHeartbeat":                   48,
		"activitySystem.retryableFailure.attemptCountIsWithinPolicy on activitySystem: unstarted reset-keepPaused":                  48,
		"activitySystem.retryableFailure.attemptCountIsWithinPolicy on activitySystem: unstarted reset-resume":                      48,
		"activitySystem.retryableFailure.attemptCountIsWithinPolicy on activitySystem: unstarted respondCanceledByID":               48,
		"activitySystem.retryableFailure.attemptCountIsWithinPolicy on activitySystem: unstarted respondCompletedByID":              48,
		"activitySystem.retryableFailure.attemptCountIsWithinPolicy on activitySystem: unstarted respondFailedByID-fatal":           48,
		"activitySystem.retryableFailure.attemptCountIsWithinPolicy on activitySystem: unstarted respondFailedByID-retryable":       48,
		"activitySystem.retryableFailure.attemptCountIsWithinPolicy on activitySystem: unstarted stop":                              48,
		"activitySystem.terminateSettles on activitySystem: canceled terminate":                                                     432,
		"activitySystem.terminateSettles on activitySystem: completed terminate":                                                    432,
		"activitySystem.terminateSettles on activitySystem: failed terminate":                                                       432,
		"activitySystem.terminateSettles on activitySystem: terminated terminate":                                                   432,
		"activitySystem.terminateSettles on activitySystem: timedOut terminate":                                                     432,
		"canceledByID on byIDCancellation: canceled respondCanceledByID":                                                            432,
		"canceledByID on byIDCancellation: completed respondCanceledByID":                                                           432,
		"canceledByID on byIDCancellation: failed respondCanceledByID":                                                              432,
		"canceledByID on byIDCancellation: paused respondCanceledByID":                                                              432,
		"canceledByID on byIDCancellation: pauseRequested respondCanceledByID":                                                      432,
		"canceledByID on byIDCancellation: resetKeepingPause respondCanceledByID":                                                   432,
		"canceledByID on byIDCancellation: resetRequested respondCanceledByID":                                                      432,
		"canceledByID on byIDCancellation: scheduled respondCanceledByID":                                                           432,
		"canceledByID on byIDCancellation: started respondCanceledByID":                                                             432,
		"canceledByID on byIDCancellation: terminated respondCanceledByID":                                                          432,
		"canceledByID on byIDCancellation: timedOut respondCanceledByID":                                                            432,
		"canceledByID on byIDCancellation: unstarted respondCanceledByID":                                                           432,
		"canceledByWorker on cancellation: pauseRequested respondCanceled":                                                          432,
		"canceledByWorker on cancellation: resetKeepingPause respondCanceled":                                                       432,
		"canceledByWorker on cancellation: resetRequested respondCanceled":                                                          432,
		"canceledByWorker on cancellation: started respondCanceled":                                                                 432,
		"cancellationReplacesReset on resetSettlement: canceled requestCancel":                                                      432,
		"cancellationReplacesReset on resetSettlement: cancelRequested requestCancel":                                               432,
		"cancellationReplacesReset on resetSettlement: completed requestCancel":                                                     432,
		"cancellationReplacesReset on resetSettlement: failed requestCancel":                                                        432,
		"cancellationReplacesReset on resetSettlement: paused requestCancel":                                                        432,
		"cancellationReplacesReset on resetSettlement: scheduled requestCancel":                                                     432,
		"cancellationReplacesReset on resetSettlement: terminated requestCancel":                                                    432,
		"cancellationReplacesReset on resetSettlement: timedOut requestCancel":                                                      432,
		"cancelRequestedWhileStarted on cancellation: canceled requestCancel":                                                       432,
		"cancelRequestedWhileStarted on cancellation: cancelRequested requestCancel":                                                432,
		"cancelRequestedWhileStarted on cancellation: completed requestCancel":                                                      432,
		"cancelRequestedWhileStarted on cancellation: failed requestCancel":                                                         432,
		"cancelRequestedWhileStarted on cancellation: paused requestCancel":                                                         432,
		"cancelRequestedWhileStarted on cancellation: scheduled requestCancel":                                                      432,
		"cancelRequestedWhileStarted on cancellation: terminated requestCancel":                                                     432,
		"cancelRequestedWhileStarted on cancellation: timedOut requestCancel":                                                       432,
		"completedByID on byIDCompletion: canceled respondCompletedByID":                                                            432,
		"completedByID on byIDCompletion: completed respondCompletedByID":                                                           432,
		"completedByID on byIDCompletion: failed respondCompletedByID":                                                              432,
		"completedByID on byIDCompletion: terminated respondCompletedByID":                                                          432,
		"completedByID on byIDCompletion: timedOut respondCompletedByID":                                                            432,
		"completedByID on byIDCompletion: unstarted respondCompletedByID":                                                           432,
		"completesAfterTimeout on timeoutRetry: cancelRequested respondCompleted":                                                   431,
		"completesAfterTimeout on timeoutRetry: pauseRequested respondCompleted":                                                    431,
		"completesAfterTimeout on timeoutRetry: resetKeepingPause respondCompleted":                                                 431,
		"completesAfterTimeout on timeoutRetry: resetRequested respondCompleted":                                                    431,
		"completesAfterTimeout on timeoutRetry: started respondCompleted":                                                           431,
		"completionWins on resetSettlement: cancelRequested respondCompleted":                                                       288,
		"completionWins on resetSettlement: pauseRequested respondCompleted":                                                        288,
		"completionWins on resetSettlement: resetKeepingPause respondCompleted":                                                     288,
		"completionWins on resetSettlement: resetRequested respondCompleted":                                                        288,
		"completionWins on resetSettlement: started respondCompleted":                                                               288,
		"failsAfterTimeout on timeoutRetry: cancelRequested respondFailed-retryable":                                                432,
		"failsAfterTimeout on timeoutRetry: pauseRequested respondFailed-retryable":                                                 431,
		"failsAfterTimeout on timeoutRetry: resetKeepingPause respondFailed-retryable":                                              432,
		"failsAfterTimeout on timeoutRetry: resetRequested respondFailed-retryable":                                                 432,
		"failsAfterTimeout on timeoutRetry: started respondFailed-retryable":                                                        431,
		"fatalFailureByID on byIDFailure: canceled respondFailedByID-fatal":                                                         432,
		"fatalFailureByID on byIDFailure: completed respondFailedByID-fatal":                                                        432,
		"fatalFailureByID on byIDFailure: failed respondFailedByID-fatal":                                                           432,
		"fatalFailureByID on byIDFailure: paused respondFailedByID-fatal":                                                           432,
		"fatalFailureByID on byIDFailure: resetKeepingPause respondFailedByID-fatal":                                                432,
		"fatalFailureByID on byIDFailure: resetRequested respondFailedByID-fatal":                                                   432,
		"fatalFailureByID on byIDFailure: scheduled respondFailedByID-fatal":                                                        432,
		"fatalFailureByID on byIDFailure: terminated respondFailedByID-fatal":                                                       432,
		"fatalFailureByID on byIDFailure: timedOut respondFailedByID-fatal":                                                         432,
		"fatalFailureByID on byIDFailure: unstarted respondFailedByID-fatal":                                                        432,
		"nonRetryableFails on retryFailures: resetKeepingPause respondFailed-fatal":                                                 432,
		"nonRetryableFails on retryFailures: resetRequested respondFailed-fatal":                                                    432,
		"pauseLeavesResetPending on resetSettlement: canceled pause":                                                                432,
		"pauseLeavesResetPending on resetSettlement: cancelRequested pause":                                                         432,
		"pauseLeavesResetPending on resetSettlement: completed pause":                                                               432,
		"pauseLeavesResetPending on resetSettlement: failed pause":                                                                  432,
		"pauseLeavesResetPending on resetSettlement: paused pause":                                                                  432,
		"pauseLeavesResetPending on resetSettlement: pauseRequested pause":                                                          432,
		"pauseLeavesResetPending on resetSettlement: resetKeepingPause pause":                                                       432,
		"pauseLeavesResetPending on resetSettlement: scheduled pause":                                                               432,
		"pauseLeavesResetPending on resetSettlement: started pause":                                                                 432,
		"pauseLeavesResetPending on resetSettlement: terminated pause":                                                              432,
		"pauseLeavesResetPending on resetSettlement: timedOut pause":                                                                432,
		"resetKeepsPause on resetSettlement: cancelRequested respondFailed-retryable":                                               432,
		"resetKeepsPause on resetSettlement: pauseRequested respondFailed-retryable":                                                432,
		"resetKeepsPause on resetSettlement: resetKeepingPause respondFailed-retryable":                                             423,
		"resetKeepsPause on resetSettlement: resetRequested respondFailed-retryable":                                                432,
		"resetKeepsPause on resetSettlement: started respondFailed-retryable":                                                       432,
		"resetKeptPaused on resetKeepingPause: canceled reset-keepPaused":                                                           432,
		"resetKeptPaused on resetKeepingPause: cancelRequested reset-keepPaused":                                                    432,
		"resetKeptPaused on resetKeepingPause: completed reset-keepPaused":                                                          432,
		"resetKeptPaused on resetKeepingPause: failed reset-keepPaused":                                                             432,
		"resetKeptPaused on resetKeepingPause: paused reset-keepPaused":                                                             426,
		"resetKeptPaused on resetKeepingPause: pauseRequested reset-keepPaused":                                                     432,
		"resetKeptPaused on resetKeepingPause: resetKeepingPause reset-keepPaused":                                                  432,
		"resetKeptPaused on resetKeepingPause: resetRequested reset-keepPaused":                                                     432,
		"resetKeptPaused on resetKeepingPause: scheduled reset-keepPaused":                                                          432,
		"resetKeptPaused on resetKeepingPause: started reset-keepPaused":                                                            432,
		"resetKeptPaused on resetKeepingPause: terminated reset-keepPaused":                                                         432,
		"resetKeptPaused on resetKeepingPause: timedOut reset-keepPaused":                                                           432,
		"resetKeptPaused on resetKeepingPause: unstarted reset-keepPaused":                                                          432,
		"resetOnExhaustion on resetSettlement: cancelRequested respondFailed-retryable":                                             432,
		"resetOnExhaustion on resetSettlement: pauseRequested respondFailed-retryable":                                              432,
		"resetOnExhaustion on resetSettlement: resetKeepingPause respondFailed-retryable":                                           432,
		"resetOnExhaustion on resetSettlement: resetRequested respondFailed-retryable":                                              423,
		"resetOnExhaustion on resetSettlement: started respondFailed-retryable":                                                     432,
		"resetOnFatal on resetSettlement: cancelRequested respondFailed-fatal":                                                      432,
		"resetOnFatal on resetSettlement: pauseRequested respondFailed-fatal":                                                       432,
		"resetOnFatal on resetSettlement: resetKeepingPause respondFailed-fatal":                                                    432,
		"resetOnFatal on resetSettlement: resetRequested respondFailed-fatal":                                                       423,
		"resetOnFatal on resetSettlement: started respondFailed-fatal":                                                              432,
		"resetOnTimeout on resetSettlement: cancelRequested startToClose":                                                           216,
		"resetOnTimeout on resetSettlement: pauseRequested startToClose":                                                            216,
		"resetOnTimeout on resetSettlement: resetKeepingPause startToClose":                                                         216,
		"resetOnTimeout on resetSettlement: resetRequested startToClose":                                                            207,
		"resetOnTimeout on resetSettlement: started startToClose":                                                                   216,
		"resetSettles on activitySystem: resetKeepingPause backoff":                                                                 144,
		"resetSettles on activitySystem: resetKeepingPause startDelay":                                                              144,
		"resetSettles on activitySystem: resetRequested backoff":                                                                    144,
		"resetSettles on activitySystem: resetRequested startDelay":                                                                 144,
		"resetStaysPending on resetSettlement: canceled reset-resume":                                                               432,
		"resetStaysPending on resetSettlement: cancelRequested reset-resume":                                                        432,
		"resetStaysPending on resetSettlement: completed reset-resume":                                                              432,
		"resetStaysPending on resetSettlement: failed reset-resume":                                                                 432,
		"resetStaysPending on resetSettlement: paused reset-resume":                                                                 432,
		"resetStaysPending on resetSettlement: resetKeepingPause reset-resume":                                                      432,
		"resetStaysPending on resetSettlement: scheduled reset-resume":                                                              432,
		"resetStaysPending on resetSettlement: terminated reset-resume":                                                             432,
		"resetStaysPending on resetSettlement: timedOut reset-resume":                                                               432,
		"resetStaysPending on resetSettlement: unstarted reset-resume":                                                              432,
		"retryCompletes on retryFailures: cancelRequested respondCompleted":                                                         431,
		"retryCompletes on retryFailures: pauseRequested respondCompleted":                                                          431,
		"retryCompletes on retryFailures: resetKeepingPause respondCompleted":                                                       431,
		"retryCompletes on retryFailures: resetRequested respondCompleted":                                                          431,
		"retryCompletes on retryFailures: started respondCompleted":                                                                 431,
		"retryExhausts on retryFailures: cancelRequested respondFailed-retryable":                                                   432,
		"retryExhausts on retryFailures: pauseRequested respondFailed-retryable":                                                    431,
		"retryExhausts on retryFailures: resetKeepingPause respondFailed-retryable":                                                 432,
		"retryExhausts on retryFailures: resetRequested respondFailed-retryable":                                                    432,
		"retryExhausts on retryFailures: started respondFailed-retryable":                                                           428,
		"startToCloseFires on timeouts: pauseRequested startToClose":                                                                144,
		"startToCloseFires on timeouts: resetKeepingPause startToClose":                                                             216,
		"startToCloseFires on timeouts: resetRequested startToClose":                                                                216,
		"startToCloseFires on timeouts: started startToClose":                                                                       144,
		"terminated on activitySystem: canceled terminate":                                                                          432,
		"terminated on activitySystem: completed terminate":                                                                         432,
		"terminated on activitySystem: failed terminate":                                                                            432,
		"terminated on activitySystem: terminated terminate":                                                                        432,
		"terminated on activitySystem: timedOut terminate":                                                                          432,
	}, failing)

	// Which rows each Property is about and holds on, as a digest of its sorted row answers, so two
	// Properties with the same counts cannot trade rows. Properties that answer every row alike share
	// a digest: the product's three capability Properties on the product and the two that hold everywhere on the
	// protocol, and the protocol's two Properties about a cancel request and its two about a terminate.
	digests := all.Digests
	require.Equal(t, map[string]string{
		"activityProduct.closedIsRejectedUniformly on activityProduct":                   "47346d65177c8c2a",
		"activityProduct.closedIsRejectedUniformly on activitySystem":                    "84d0f4eaaabd50b4",
		"activityProduct.closedIsRejectedUniformly on completion":                        "84d0f4eaaabd50b4",
		"activityProduct.closedIsRejectedUniformly on retryFailures":                     "84d0f4eaaabd50b4",
		"activityProduct.closedIsRejectedUniformly on cancellation":                      "84d0f4eaaabd50b4",
		"activityProduct.closedIsRejectedUniformly on pausing":                           "84d0f4eaaabd50b4",
		"activityProduct.closedIsRejectedUniformly on dispatchEligibility":               "84d0f4eaaabd50b4",
		"activityProduct.closedIsRejectedUniformly on timeouts":                          "84d0f4eaaabd50b4",
		"activityProduct.pausedIsNotDispatched on activityProduct":                       "47346d65177c8c2a",
		"activityProduct.pausedIsNotDispatched on activitySystem":                        "dbfcf53fbe87cf17",
		"activityProduct.pausedIsNotDispatched on completion":                            "dbfcf53fbe87cf17",
		"activityProduct.pausedIsNotDispatched on retryFailures":                         "dbfcf53fbe87cf17",
		"activityProduct.pausedIsNotDispatched on cancellation":                          "dbfcf53fbe87cf17",
		"activityProduct.pausedIsNotDispatched on pausing":                               "dbfcf53fbe87cf17",
		"activityProduct.pausedIsNotDispatched on dispatchEligibility":                   "dbfcf53fbe87cf17",
		"activityProduct.pausedIsNotDispatched on timeouts":                              "dbfcf53fbe87cf17",
		"activityProduct.terminalStatesAreFinal on activityProduct":                      "47346d65177c8c2a",
		"activityProduct.terminalStatesAreFinal on activitySystem":                       "dbfcf53fbe87cf17",
		"activityProduct.terminalStatesAreFinal on completion":                           "dbfcf53fbe87cf17",
		"activityProduct.terminalStatesAreFinal on retryFailures":                        "dbfcf53fbe87cf17",
		"activityProduct.terminalStatesAreFinal on cancellation":                         "dbfcf53fbe87cf17",
		"activityProduct.terminalStatesAreFinal on pausing":                              "dbfcf53fbe87cf17",
		"activityProduct.terminalStatesAreFinal on dispatchEligibility":                  "dbfcf53fbe87cf17",
		"activityProduct.terminalStatesAreFinal on timeouts":                             "dbfcf53fbe87cf17",
		"activitySystem.cancelIsRequested on byIDCancellation":                           "cf4eafb4fde9d03f",
		"activitySystem.fatalFailure.attemptCountIsWithinPolicy on activitySystem":       "bc543de2c8f7694f",
		"activitySystem.fatalFailure.failureCancels on activitySystem":                   "7bf22af91e10c41d",
		"activitySystem.fatalFailure.failureEndsFailed on activitySystem":                "7bf22af91e10c41d",
		"activitySystem.fatalFailure.failurePauses on activitySystem":                    "7bf22af91e10c41d",
		"activitySystem.fatalFailure.failureReturnsToWaiting on activitySystem":          "7bf22af91e10c41d",
		"activitySystem.heartbeatDeadline.deadlinePauses on activitySystem":              "59dbb8228ce4a4fb",
		"activitySystem.heartbeatDeadline.deadlineReturnsToWaiting on activitySystem":    "59dbb8228ce4a4fb",
		"activitySystem.heartbeatDeadline.deadlineTimesOut on activitySystem":            "59dbb8228ce4a4fb",
		"activitySystem.heartbeatDeadline.firesInWindow on activitySystem":               "59dbb8228ce4a4fb",
		"activitySystem.retryableFailure.attemptCountIsWithinPolicy on activitySystem":   "bc543de2c8f7694f",
		"activitySystem.retryableFailure.failureCancels on activitySystem":               "5a7409303b83aaf3",
		"activitySystem.retryableFailure.failureEndsFailed on activitySystem":            "5a7409303b83aaf3",
		"activitySystem.retryableFailure.failurePauses on activitySystem":                "5a7409303b83aaf3",
		"activitySystem.retryableFailure.failureReturnsToWaiting on activitySystem":      "5a7409303b83aaf3",
		"activitySystem.scheduleToCloseDeadline.deadlineTimesOut on activitySystem":      "e168db08b6c2c8f6",
		"activitySystem.scheduleToCloseDeadline.firesInWindow on activitySystem":         "e168db08b6c2c8f6",
		"activitySystem.scheduleToStartDeadline.deadlineTimesOut on activitySystem":      "3856a9defbf72137",
		"activitySystem.scheduleToStartDeadline.firesInWindow on activitySystem":         "3856a9defbf72137",
		"activitySystem.startToCloseDeadline.deadlinePauses on activitySystem":           "ec24b7035116159d",
		"activitySystem.startToCloseDeadline.deadlineReturnsToWaiting on activitySystem": "ec24b7035116159d",
		"activitySystem.startToCloseDeadline.deadlineTimesOut on activitySystem":         "ec24b7035116159d",
		"activitySystem.startToCloseDeadline.firesInWindow on activitySystem":            "ec24b7035116159d",
		"activitySystem.terminateSettles on activitySystem":                              "eec113aa6c88afb4",
		"canceledByID on byIDCancellation":                                               "756281535c53b38c",
		"canceledByWorker on cancellation":                                               "83304886f813968b",
		"cancelIsNotUndone on activitySystem":                                            "dbfcf53fbe87cf17",
		"cancellationReplacesReset on resetSettlement":                                   "cf4eafb4fde9d03f",
		"cancelRequestedWhileStarted on cancellation":                                    "cf4eafb4fde9d03f",
		"completedByID on byIDCompletion":                                                "6ed1bec07f1dde2f",
		"completes on completion":                                                        "ee05c495bbebdf6a",
		"completes on pausing":                                                           "ee05c495bbebdf6a",
		"completes on dispatchEligibility":                                               "ee05c495bbebdf6a",
		"completesAfterTimeout on timeoutRetry":                                          "df97983f12abe645",
		"completionWins on resetSettlement":                                              "f838835538d0e0a8",
		"controlPrecedence on activitySystem":                                            "dbfcf53fbe87cf17",
		"dispatchRequiresReady on dispatchEligibility":                                   "dbfcf53fbe87cf17",
		"failsAfterTimeout on timeoutRetry":                                              "ba257a34bdcd887b",
		"fatalFailureByID on byIDFailure":                                                "a0e6d6d8280ba1c1",
		"heartbeatCompletes on heartbeatCompletion":                                      "ee05c495bbebdf6a",
		"heartbeatExhausts on heartbeatExhaustion":                                       "59dbb8228ce4a4fb",
		"heartbeatRetryCompletes on heartbeatRetry":                                      "ee05c495bbebdf6a",
		"nonRetryableFails on retryFailures":                                             "e5dd352cbf39a861",
		"pauseLeavesResetPending on resetSettlement":                                     "df4cafd0bf553a4c",
		"resetAttemptCompletes on deferredReset":                                         "ee05c495bbebdf6a",
		"resetKeepsPause on resetSettlement":                                             "62d087882d4a7d7c",
		"resetKeepsPaused on activitySystem":                                             "a6b68614e9a3c8d0",
		"resetKeptPaused on resetKeepingPause":                                           "e8299e6e15546a2e",
		"resetOnExhaustion on resetSettlement":                                           "80716524c7538b24",
		"resetOnFatal on resetSettlement":                                                "ba634d2130edd51a",
		"resetOnTimeout on resetSettlement":                                              "afa2a2f54c7c780e",
		"resetResumes on activitySystem":                                                 "20c9f64fcff6b835",
		"resetSettles on activitySystem":                                                 "7243f1fbe52662c0",
		"resetStaysPending on resetSettlement":                                           "d8d711c0e67c2bd5",
		"retryCompletes on retryFailures":                                                "70e263b17f5cec29",
		"retryExhausts on retryFailures":                                                 "b76daef36d54d61a",
		"scheduleToCloseStaysTerminal on resetSettlement":                                "e168db08b6c2c8f6",
		"scheduleToStartFires on timeouts":                                               "3856a9defbf72137",
		"scheduleToStartRequiresDispatch on dispatchEligibility":                         "dbfcf53fbe87cf17",
		"startedByPollingWorker on standaloneActivity":                                   "09ac55136c06e20a",
		"startToCloseFires on timeouts":                                                  "f5c95d1fed5590bf",
		"terminated on activitySystem":                                                   "eec113aa6c88afb4",
	}, digests)
}

// pathAnswer is what a Query says: its verdict, how much its search explored and expanded, whether
// its claim was read on some step, and each step of its witness: the class, the answer, the state it
// reaches and the facts it records.
type pathAnswer struct {
	Kind               ReceiptKind
	Explored, Expanded int
	Exercised          bool
	Witness            []string
}

func witnessStep(class, outcome, state string, facts []string) string {
	return fmt.Sprintf("%s: %s, %s %v", class, outcome, state, facts)
}

func pathAnswerOf(r Receipt) pathAnswer {
	answer := pathAnswer{Kind: r.Kind, Explored: r.Explored, Expanded: r.Expanded, Exercised: r.Exercised}
	if r.Witness != nil {
		for _, step := range r.Witness.Steps {
			var facts []string
			for _, fact := range step.Facts {
				facts = append(facts, fact.Value)
			}
			answer.Witness = append(answer.Witness, witnessStep(step.Action.Value, step.Outcome.Value, step.State.Value, facts))
		}
	}
	return answer
}

// Every Query of the Model answers as pinned: a found one by the path its Scenario places, a verified
// one over its whole search, and each reads its claim on some step.
func TestActivityQueriesAnswerAlongTheirPaths(t *testing.T) {
	got := map[string]pathAnswer{}
	generated := 0
	for _, r := range activityChecked(t).Receipts {
		if r.Subject == QuerySubject {
			if r.Kind == Found {
				require.Len(t, r.Rows, len(r.Witness.Steps), "%s: a row for each step of the witness", receiptKey(r))
			}
			require.NotContains(t, got, r.Key.Name)
			got[r.Key.Name] = pathAnswerOf(r)
			if generatedQuery(activityModel(t), r) {
				generated++
			}
		}
	}
	require.Len(t, got, 65)
	require.Equal(t, 26, generated)
	require.False(t, generatedQuery(activityModel(t), Receipt{Key: ClaimKey{Owner: "byIDCancellation", Name: "activitySystem.cancelIsRequested"}}))
	require.Equal(t, map[string]pathAnswer{
		"activityProduct.closedIsRejectedUniformly": {Kind: Verified, Explored: 10, Expanded: 10, Exercised: true},
		"activityProduct.pausedIsNotDispatched":     {Kind: Verified, Explored: 10, Expanded: 10, Exercised: true},
		"activityProduct.terminalStatesAreFinal":    {Kind: Verified, Explored: 10, Expanded: 10, Exercised: true},
		"activitySystem.cancelIsRequested": {Kind: Found, Explored: 5, Expanded: 4, Exercised: true, Witness: []string{
			"start-unset-unset-unset-unset-unset-unlimited: accepted, scheduled-now-0-unset-unset-unset-unset-unlimited [statusScheduled]",
			"poll: accepted, started-now-1-unset-unset-unset-unset-unlimited [statusStarted attemptCount]",
			"requestCancel: accepted, cancelRequested-now-1-unset-unset-unset-unset-unlimited [statusCancelRequested]",
			"respondCanceledByID: accepted, canceled-now-1-unset-unset-unset-unset-unlimited [statusCanceled]",
		}},
		"activitySystem.fatalFailure.attemptCountIsWithinPolicy":       {Kind: Verified, Explored: 1671, Expanded: 1654, Exercised: true},
		"activitySystem.fatalFailure.failureCancels":                   {Kind: Verified, Explored: 2726, Expanded: 2643, Exercised: true},
		"activitySystem.fatalFailure.failureEndsFailed":                {Kind: Verified, Explored: 2726, Expanded: 2643, Exercised: true},
		"activitySystem.fatalFailure.failurePauses":                    {Kind: Verified, Explored: 2726, Expanded: 2643, Exercised: true},
		"activitySystem.fatalFailure.failureReturnsToWaiting":          {Kind: Verified, Explored: 2726, Expanded: 2643, Exercised: true},
		"activitySystem.heartbeatDeadline.deadlinePauses":              {Kind: Verified, Explored: 2381, Expanded: 2355, Exercised: true},
		"activitySystem.heartbeatDeadline.deadlineReturnsToWaiting":    {Kind: Verified, Explored: 2381, Expanded: 2355, Exercised: true},
		"activitySystem.heartbeatDeadline.deadlineTimesOut":            {Kind: Verified, Explored: 2381, Expanded: 2355, Exercised: true},
		"activitySystem.heartbeatDeadline.firesInWindow":               {Kind: Verified, Explored: 2381, Expanded: 2355, Exercised: true},
		"activitySystem.retryableFailure.attemptCountIsWithinPolicy":   {Kind: Verified, Explored: 1671, Expanded: 1654, Exercised: true},
		"activitySystem.retryableFailure.failureCancels":               {Kind: Verified, Explored: 3099, Expanded: 3065, Exercised: true},
		"activitySystem.retryableFailure.failureEndsFailed":            {Kind: Verified, Explored: 3099, Expanded: 3065, Exercised: true},
		"activitySystem.retryableFailure.failurePauses":                {Kind: Verified, Explored: 3099, Expanded: 3065, Exercised: true},
		"activitySystem.retryableFailure.failureReturnsToWaiting":      {Kind: Verified, Explored: 3099, Expanded: 3065, Exercised: true},
		"activitySystem.scheduleToCloseDeadline.deadlineTimesOut":      {Kind: Verified, Explored: 1715, Expanded: 1697, Exercised: true},
		"activitySystem.scheduleToCloseDeadline.firesInWindow":         {Kind: Verified, Explored: 1715, Expanded: 1697, Exercised: true},
		"activitySystem.scheduleToStartDeadline.deadlineTimesOut":      {Kind: Verified, Explored: 1703, Expanded: 1679, Exercised: true},
		"activitySystem.scheduleToStartDeadline.firesInWindow":         {Kind: Verified, Explored: 1703, Expanded: 1679, Exercised: true},
		"activitySystem.startToCloseDeadline.deadlinePauses":           {Kind: Verified, Explored: 2381, Expanded: 2355, Exercised: true},
		"activitySystem.startToCloseDeadline.deadlineReturnsToWaiting": {Kind: Verified, Explored: 2381, Expanded: 2355, Exercised: true},
		"activitySystem.startToCloseDeadline.deadlineTimesOut":         {Kind: Verified, Explored: 2381, Expanded: 2355, Exercised: true},
		"activitySystem.startToCloseDeadline.firesInWindow":            {Kind: Verified, Explored: 2381, Expanded: 2355, Exercised: true},
		"activitySystem.terminateSettles": {Kind: Found, Explored: 4, Expanded: 3, Exercised: true, Witness: []string{
			"start-unset-unset-unset-unset-unset-unlimited: accepted, scheduled-now-0-unset-unset-unset-unset-unlimited [statusScheduled]",
			"stop: accepted, scheduled-now-0-unset-unset-unset-unset-unlimited []",
			"terminate: accepted, terminated-now-0-unset-unset-unset-unset-unlimited [statusTerminated]",
		}},
		"cancel": {Kind: Found, Explored: 5, Expanded: 4, Exercised: true, Witness: []string{
			"start-unset-unset-unset-unset-unset-unlimited: accepted, scheduled-now-0-unset-unset-unset-unset-unlimited [statusScheduled]",
			"poll: accepted, started-now-1-unset-unset-unset-unset-unlimited [statusStarted attemptCount]",
			"requestCancel: accepted, cancelRequested-now-1-unset-unset-unset-unset-unlimited [statusCancelRequested]",
			"respondCanceled: accepted, canceled-now-1-unset-unset-unset-unset-unlimited [statusCanceled]",
		}},
		"cancellationKeepsPrecedence": {Kind: Verified, Explored: 1671, Expanded: 1654, Exercised: true},
		"cancelRequest": {Kind: Found, Explored: 5, Expanded: 4, Exercised: true, Witness: []string{
			"start-unset-unset-unset-unset-unset-unlimited: accepted, scheduled-now-0-unset-unset-unset-unset-unlimited [statusScheduled]",
			"poll: accepted, started-now-1-unset-unset-unset-unset-unlimited [statusStarted attemptCount]",
			"requestCancel: accepted, cancelRequested-now-1-unset-unset-unset-unset-unlimited [statusCancelRequested]",
			"respondCanceled: accepted, canceled-now-1-unset-unset-unset-unset-unlimited [statusCanceled]",
		}},
		"completion": {Kind: Found, Explored: 4, Expanded: 3, Exercised: true, Witness: []string{
			"start-unset-unset-unset-unset-unset-unlimited: accepted, scheduled-now-0-unset-unset-unset-unset-unlimited [statusScheduled]",
			"poll: accepted, started-now-1-unset-unset-unset-unset-unlimited [statusStarted attemptCount]",
			"respondCompleted: accepted, completed-now-1-unset-unset-unset-unset-unlimited [statusCompleted]",
		}},
		"controlsKeepPrecedence": {Kind: Verified, Explored: 1671, Expanded: 1654, Exercised: true},
		"deferredResetCompletes": {Kind: Found, Explored: 7, Expanded: 6, Exercised: true, Witness: []string{
			"start-unset-unset-unset-expires-unset-one: accepted, scheduled-now-0-unset-unset-unset-expires-one [statusScheduled]",
			"poll: accepted, started-now-1-unset-unset-unset-expires-one [statusStarted attemptCount]",
			"reset-resume: accepted, resetRequested-now-1-unset-unset-unset-expires-one [statusStarted]",
			"heartbeat: accepted, scheduled-now-0-unset-unset-unset-expires-one [statusScheduled attemptCount heartbeatTimedOut]",
			"poll: accepted, started-now-1-unset-unset-unset-expires-one [statusStarted attemptCount]",
			"respondCompleted: accepted, completed-now-1-unset-unset-unset-expires-one [statusCompleted]",
		}},
		"delayedAttemptsAreNotDispatched": {Kind: Verified, Explored: 1671, Expanded: 1654, Exercised: true},
		"directResetKeepsPaused":          {Kind: Verified, Explored: 3243, Expanded: 3100, Exercised: true},
		"directResetResumes":              {Kind: Verified, Explored: 3323, Expanded: 3148, Exercised: true},
		"heartbeatThenCompletes": {Kind: Found, Explored: 5, Expanded: 4, Exercised: true, Witness: []string{
			"start-unset-unset-unset-expires-unset-unlimited: accepted, scheduled-now-0-unset-unset-unset-expires-unlimited [statusScheduled]",
			"poll: accepted, started-now-1-unset-unset-unset-expires-unlimited [statusStarted attemptCount]",
			"recordHeartbeat: accepted, started-now-1-unset-unset-unset-expires-unlimited [heartbeatReceived]",
			"respondCompleted: accepted, completed-now-1-unset-unset-unset-expires-unlimited [statusCompleted]",
		}},
		"heartbeatTimeoutExhausts": {Kind: Found, Explored: 5, Expanded: 4, Exercised: true, Witness: []string{
			"start-unset-unset-unset-expires-unset-one: accepted, scheduled-now-0-unset-unset-unset-expires-one [statusScheduled]",
			"poll: accepted, started-now-1-unset-unset-unset-expires-one [statusStarted attemptCount]",
			"recordHeartbeat: accepted, started-now-1-unset-unset-unset-expires-one [heartbeatReceived]",
			"heartbeat: accepted, timedOut-now-1-unset-unset-unset-expires-one [statusTimedOut-heartbeat heartbeatTimedOut]",
		}},
		"heartbeatTimeoutRetriesThenCompletes": {Kind: Found, Explored: 8, Expanded: 7, Exercised: true, Witness: []string{
			"start-unset-unset-unset-expires-unset-two: accepted, scheduled-now-0-unset-unset-unset-expires-two [statusScheduled]",
			"poll: accepted, started-now-1-unset-unset-unset-expires-two [statusStarted attemptCount]",
			"recordHeartbeat: accepted, started-now-1-unset-unset-unset-expires-two [heartbeatReceived]",
			"heartbeat: accepted, scheduled-backoff-1-unset-unset-unset-expires-two [statusScheduled attemptCount heartbeatTimedOut]",
			"backoff: accepted, scheduled-now-1-unset-unset-unset-expires-two []",
			"poll: accepted, started-now-2-unset-unset-unset-expires-two [statusStarted attemptCount]",
			"respondCompleted: accepted, completed-now-2-unset-unset-unset-expires-two [statusCompleted]",
		}},
		"heldCanceledByID": {Kind: Found, Explored: 5, Expanded: 4, Exercised: true, Witness: []string{
			"start-unset-unset-unset-unset-unset-unlimited: accepted, scheduled-now-0-unset-unset-unset-unset-unlimited [statusScheduled]",
			"poll: accepted, started-now-1-unset-unset-unset-unset-unlimited [statusStarted attemptCount]",
			"requestCancel: accepted, cancelRequested-now-1-unset-unset-unset-unset-unlimited [statusCancelRequested]",
			"respondCanceledByID: accepted, canceled-now-1-unset-unset-unset-unset-unlimited [statusCanceled]",
		}},
		"heldFailedByID": {Kind: Found, Explored: 4, Expanded: 3, Exercised: true, Witness: []string{
			"start-unset-unset-unset-unset-unset-unlimited: accepted, scheduled-now-0-unset-unset-unset-unset-unlimited [statusScheduled]",
			"poll: accepted, started-now-1-unset-unset-unset-unset-unlimited [statusStarted attemptCount]",
			"respondFailedByID-fatal: accepted, failed-now-1-unset-unset-unset-unset-unlimited [statusFailed]",
		}},
		"keepPausedReset": {Kind: Found, Explored: 4, Expanded: 3, Exercised: true, Witness: []string{
			"start-unset-unset-unset-unset-unset-unlimited: accepted, scheduled-now-0-unset-unset-unset-unset-unlimited [statusScheduled]",
			"pause: accepted, paused-now-0-unset-unset-unset-unset-unlimited [statusPaused]",
			"reset-keepPaused: accepted, paused-now-0-unset-unset-unset-unset-unlimited [statusPaused]",
		}},
		"nonRetryableFailure": {Kind: Found, Explored: 4, Expanded: 3, Exercised: true, Witness: []string{
			"start-unset-unset-unset-unset-unset-unlimited: accepted, scheduled-now-0-unset-unset-unset-unset-unlimited [statusScheduled]",
			"poll: accepted, started-now-1-unset-unset-unset-unset-unlimited [statusStarted attemptCount]",
			"respondFailed-fatal: accepted, failed-now-1-unset-unset-unset-unset-unlimited [statusFailed]",
		}},
		"pauseResume": {Kind: Found, Explored: 6, Expanded: 5, Exercised: true, Witness: []string{
			"start-unset-unset-unset-unset-unset-unlimited: accepted, scheduled-now-0-unset-unset-unset-unset-unlimited [statusScheduled]",
			"pause: accepted, paused-now-0-unset-unset-unset-unset-unlimited [statusPaused]",
			"unpause: accepted, scheduled-now-0-unset-unset-unset-unset-unlimited [statusScheduled]",
			"poll: accepted, started-now-1-unset-unset-unset-unset-unlimited [statusStarted attemptCount]",
			"respondCompleted: accepted, completed-now-1-unset-unset-unset-unset-unlimited [statusCompleted]",
		}},
		"resetCancellation": {Kind: Found, Explored: 5, Expanded: 4, Exercised: true, Witness: []string{
			"start-unset-unset-unset-unset-unset-unlimited: accepted, scheduled-now-0-unset-unset-unset-unset-unlimited [statusScheduled]",
			"poll: accepted, started-now-1-unset-unset-unset-unset-unlimited [statusStarted attemptCount]",
			"reset-resume: accepted, resetRequested-now-1-unset-unset-unset-unset-unlimited [statusStarted]",
			"requestCancel: accepted, cancelRequested-now-1-unset-unset-unset-unset-unlimited [statusCancelRequested]",
		}},
		"resetCompletion": {Kind: Found, Explored: 5, Expanded: 4, Exercised: true, Witness: []string{
			"start-unset-unset-unset-unset-unset-unlimited: accepted, scheduled-now-0-unset-unset-unset-unset-unlimited [statusScheduled]",
			"poll: accepted, started-now-1-unset-unset-unset-unset-unlimited [statusStarted attemptCount]",
			"reset-resume: accepted, resetRequested-now-1-unset-unset-unset-unset-unlimited [statusStarted]",
			"respondCompleted: accepted, completed-now-1-unset-unset-unset-unset-unlimited [statusCompleted]",
		}},
		"resetExhaustion": {Kind: Found, Explored: 5, Expanded: 4, Exercised: true, Witness: []string{
			"start-unset-unset-unset-unset-unset-one: accepted, scheduled-now-0-unset-unset-unset-unset-one [statusScheduled]",
			"poll: accepted, started-now-1-unset-unset-unset-unset-one [statusStarted attemptCount]",
			"reset-resume: accepted, resetRequested-now-1-unset-unset-unset-unset-one [statusStarted]",
			"respondFailed-retryable: accepted, scheduled-now-0-unset-unset-unset-unset-one [statusScheduled attemptCount]",
		}},
		"resetFatality": {Kind: Found, Explored: 5, Expanded: 4, Exercised: true, Witness: []string{
			"start-unset-unset-unset-unset-unset-unlimited: accepted, scheduled-now-0-unset-unset-unset-unset-unlimited [statusScheduled]",
			"poll: accepted, started-now-1-unset-unset-unset-unset-unlimited [statusStarted attemptCount]",
			"reset-resume: accepted, resetRequested-now-1-unset-unset-unset-unset-unlimited [statusStarted]",
			"respondFailed-fatal: accepted, scheduled-now-0-unset-unset-unset-unset-unlimited [statusScheduled attemptCount]",
		}},
		"resetKeptPause": {Kind: Found, Explored: 6, Expanded: 5, Exercised: true, Witness: []string{
			"start-unset-unset-unset-unset-unset-unlimited: accepted, scheduled-now-0-unset-unset-unset-unset-unlimited [statusScheduled]",
			"poll: accepted, started-now-1-unset-unset-unset-unset-unlimited [statusStarted attemptCount]",
			"pause: accepted, pauseRequested-now-1-unset-unset-unset-unset-unlimited [statusPaused]",
			"reset-keepPaused: accepted, resetKeepingPause-now-1-unset-unset-unset-unset-unlimited [statusPaused]",
			"respondFailed-retryable: accepted, paused-now-0-unset-unset-unset-unset-unlimited [statusPaused attemptCount]",
		}},
		"resetOutranksPause": {Kind: Found, Explored: 5, Expanded: 4, Exercised: true, Witness: []string{
			"start-unset-unset-unset-unset-unset-unlimited: accepted, scheduled-now-0-unset-unset-unset-unset-unlimited [statusScheduled]",
			"poll: accepted, started-now-1-unset-unset-unset-unset-unlimited [statusStarted attemptCount]",
			"reset-resume: accepted, resetRequested-now-1-unset-unset-unset-unset-unlimited [statusStarted]",
			"pause: rejected-failedPrecondition, resetRequested-now-1-unset-unset-unset-unset-unlimited []",
		}},
		"resetRepeated": {Kind: Found, Explored: 5, Expanded: 4, Exercised: true, Witness: []string{
			"start-unset-unset-unset-unset-unset-unlimited: accepted, scheduled-now-0-unset-unset-unset-unset-unlimited [statusScheduled]",
			"poll: accepted, started-now-1-unset-unset-unset-unset-unlimited [statusStarted attemptCount]",
			"reset-resume: accepted, resetRequested-now-1-unset-unset-unset-unset-unlimited [statusStarted]",
			"reset-resume: rejected-failedPrecondition, resetRequested-now-1-unset-unset-unset-unset-unlimited []",
		}},
		"resetScheduleToClose": {Kind: Found, Explored: 5, Expanded: 4, Exercised: true, Witness: []string{
			"start-expires-unset-unset-unset-unset-unlimited: accepted, scheduled-now-0-expires-unset-unset-unset-unlimited [statusScheduled]",
			"poll: accepted, started-now-1-expires-unset-unset-unset-unlimited [statusStarted attemptCount]",
			"reset-resume: accepted, resetRequested-now-1-expires-unset-unset-unset-unlimited [statusStarted]",
			"scheduleToClose: accepted, timedOut-now-1-expires-unset-unset-unset-unlimited [statusTimedOut-scheduleToClose]",
		}},
		"resetSettlement": {Kind: Verified, Explored: 1671, Expanded: 1654, Exercised: true},
		"resetTimeout": {Kind: Found, Explored: 5, Expanded: 4, Exercised: true, Witness: []string{
			"start-unset-unset-expires-unset-unset-one: accepted, scheduled-now-0-unset-unset-expires-unset-one [statusScheduled]",
			"poll: accepted, started-now-1-unset-unset-expires-unset-one [statusStarted attemptCount]",
			"reset-resume: accepted, resetRequested-now-1-unset-unset-expires-unset-one [statusStarted]",
			"startToClose: accepted, scheduled-now-0-unset-unset-expires-unset-one [statusScheduled attemptCount]",
		}},
		"retry": {Kind: Found, Explored: 7, Expanded: 6, Exercised: true, Witness: []string{
			"start-unset-unset-unset-unset-unset-unlimited: accepted, scheduled-now-0-unset-unset-unset-unset-unlimited [statusScheduled]",
			"poll: accepted, started-now-1-unset-unset-unset-unset-unlimited [statusStarted attemptCount]",
			"respondFailed-retryable: accepted, scheduled-backoff-1-unset-unset-unset-unset-unlimited [statusScheduled attemptCount]",
			"backoff: accepted, scheduled-now-1-unset-unset-unset-unset-unlimited []",
			"poll: accepted, started-now-2-unset-unset-unset-unset-unlimited [statusStarted attemptCount]",
			"respondCompleted: accepted, completed-now-2-unset-unset-unset-unset-unlimited [statusCompleted]",
		}},
		"retryAfterTimeout": {Kind: Found, Explored: 7, Expanded: 6, Exercised: true, Witness: []string{
			"start-unset-unset-expires-unset-unset-two: accepted, scheduled-now-0-unset-unset-expires-unset-two [statusScheduled]",
			"poll: accepted, started-now-1-unset-unset-expires-unset-two [statusStarted attemptCount]",
			"startToClose: accepted, scheduled-backoff-1-unset-unset-expires-unset-two [statusScheduled attemptCount]",
			"backoff: accepted, scheduled-now-1-unset-unset-expires-unset-two []",
			"poll: accepted, started-now-2-unset-unset-expires-unset-two [statusStarted attemptCount]",
			"respondCompleted: accepted, completed-now-2-unset-unset-expires-unset-two [statusCompleted]",
		}},
		"retryExhaustion": {Kind: Found, Explored: 7, Expanded: 6, Exercised: true, Witness: []string{
			"start-unset-unset-expires-unset-unset-two: accepted, scheduled-now-0-unset-unset-expires-unset-two [statusScheduled]",
			"poll: accepted, started-now-1-unset-unset-expires-unset-two [statusStarted attemptCount]",
			"startToClose: accepted, scheduled-backoff-1-unset-unset-expires-unset-two [statusScheduled attemptCount]",
			"backoff: accepted, scheduled-now-1-unset-unset-expires-unset-two []",
			"poll: accepted, started-now-2-unset-unset-expires-unset-two [statusStarted attemptCount]",
			"respondFailed-retryable: accepted, failed-now-2-unset-unset-expires-unset-two [statusFailed]",
		}},
		"retryExhaustionByFailures": {Kind: Verified, Explored: 7, Expanded: 6, Exercised: true},
		"scheduledCompletedByID": {Kind: Found, Explored: 3, Expanded: 2, Exercised: true, Witness: []string{
			"start-unset-unset-unset-unset-unset-unlimited: accepted, scheduled-now-0-unset-unset-unset-unset-unlimited [statusScheduled]",
			"respondCompletedByID: accepted, completed-now-0-unset-unset-unset-unset-unlimited [statusCompleted]",
		}},
		"scheduleToStartTimeout": {Kind: Found, Explored: 4, Expanded: 3, Exercised: true, Witness: []string{
			"start-unset-expires-unset-unset-unset-unlimited: accepted, scheduled-now-0-unset-expires-unset-unset-unlimited [statusScheduled]",
			"stop: accepted, scheduled-now-0-unset-expires-unset-unset-unlimited []",
			"scheduleToStart: accepted, timedOut-now-0-unset-expires-unset-unset-unlimited [statusTimedOut-scheduleToStart]",
		}},
		"scheduleToStartWaitsForDispatch": {Kind: Verified, Explored: 1671, Expanded: 1654, Exercised: true},
		"startDelayedCompletion": {Kind: Found, Explored: 5, Expanded: 4, Exercised: true, Witness: []string{
			"start-unset-unset-unset-unset-expires-unlimited: accepted, scheduled-startDelay-0-unset-unset-unset-unset-unlimited [statusScheduled]",
			"startDelay: accepted, scheduled-now-0-unset-unset-unset-unset-unlimited []",
			"poll: accepted, started-now-1-unset-unset-unset-unset-unlimited [statusStarted attemptCount]",
			"respondCompleted: accepted, completed-now-1-unset-unset-unset-unset-unlimited [statusCompleted]",
		}},
		"startToCloseTimeout": {Kind: Found, Explored: 4, Expanded: 3, Exercised: true, Witness: []string{
			"start-unset-unset-expires-unset-unset-one: accepted, scheduled-now-0-unset-unset-expires-unset-one [statusScheduled]",
			"poll: accepted, started-now-1-unset-unset-expires-unset-one [statusStarted attemptCount]",
			"startToClose: accepted, timedOut-now-1-unset-unset-expires-unset-one [statusTimedOut-startToClose]",
		}},
		"stoppedWorkerStartsNothing": {Kind: Verified, Explored: 7, Expanded: 6, Exercised: true},
		"terminate": {Kind: Found, Explored: 4, Expanded: 3, Exercised: true, Witness: []string{
			"start-unset-unset-unset-unset-unset-unlimited: accepted, scheduled-now-0-unset-unset-unset-unset-unlimited [statusScheduled]",
			"stop: accepted, scheduled-now-0-unset-unset-unset-unset-unlimited []",
			"terminate: accepted, terminated-now-0-unset-unset-unset-unset-unlimited [statusTerminated]",
		}},
	}, got)
}

func generatedQuery(m *umpirespb.Model, r Receipt) bool {
	for _, q := range m.GetQueries() {
		if q.GetScenario().GetMachine() == r.Key.Owner && q.GetName() == r.Key.Name {
			p := admProperty(m, q.GetProperty().GetMachine(), q.GetProperty().GetName())
			return p != nil && p.GetOrigin() != nil
		}
	}
	return false
}

// activityReport is the Model as lifted, checked once: what a mutant's answers are compared with.
var activityReport = sync.OnceValues(func() (*Report, error) {
	m, err := activityBaseline()
	if err != nil {
		return nil, err
	}
	return Check(m, DefaultScope), nil
})

func activityChecked(t *testing.T) *Report {
	t.Helper()
	r, err := activityReport()
	require.NoError(t, err)
	return r
}

// pathDisagreements is every path Query whose answer through m differs from its answer through the
// Model as lifted, as a list.
func pathDisagreements(t *testing.T, m *umpirespb.Model) []string {
	t.Helper()
	report := checked(t, m)
	var out []string
	for _, expected := range activityChecked(t).Receipts {
		if expected.Subject != QuerySubject || generatedQuery(activityModel(t), expected) {
			continue
		}
		got := receiptOf(t, report, receiptKey(expected))
		if got.Kind != expected.Kind || got.Explored != expected.Explored || got.Expanded != expected.Expanded ||
			got.Exercised != expected.Exercised || !slices.Equal(got.Rows, expected.Rows) || !sameTrace(got.Witness, expected.Witness) {
			out = append(out, expected.Key.Name)
		}
	}
	return out
}

// generatedDisagreements is every generated Query that answers m with another verdict than the Model as
// lifted, as a list.
func generatedDisagreements(t *testing.T, m *umpirespb.Model) []string {
	t.Helper()
	report := checked(t, m)
	var out []string
	for _, expected := range activityChecked(t).Receipts {
		if expected.Subject != QuerySubject || !generatedQuery(activityModel(t), expected) {
			continue
		}
		if receiptOf(t, report, receiptKey(expected)).Kind != expected.Kind {
			out = append(out, expected.Key.Name)
		}
	}
	slices.Sort(out)
	return out
}

func activityFunction(t *testing.T, m *umpirespb.Model, name string) *umpirespb.Function {
	t.Helper()
	f := functionNamed(m, name)
	require.NotNil(t, f, name)
	return f
}

// narrowed is a predicate that also asks `extra`.
func narrowed(f *umpirespb.Function, extra *umpirespb.Expr) {
	f.Body = &umpirespb.Expr{Position: f.GetBody().GetPosition(),
		Kind: &umpirespb.Expr_Binary{Binary: &umpirespb.Binary{Op: umpirespb.Binary_OP_AND, Left: f.GetBody(), Right: extra}}}
}

// stateField reads a field of the state of the step a predicate's parameter names.
func stateField(f *umpirespb.Function, param int, name string) *umpirespb.Expr {
	return field(field(expr(f.GetParams()[param].GetName()), "state"), name)
}

// A predicate that differs from the lifted one away from the Scenarios' paths answers every path
// Query as the lifted one does, and is told apart by the rows: one mutant for each way a Property is
// declared, a same-step predicate, the class a Property is about, a transition predicate read on its
// own machine and through the refinement, and the composition's Property, by its predicate and by the
// composed action it is about. A generated Query searches its machine freely, so it is no path Query
// and may catch a mutant the paths miss; `generated` names those that do.
type activityPropertyMutant struct {
	mutate    func(t *testing.T, m *umpirespb.Model)
	rows      []string
	paths     []string
	generated []string
}

func activityPropertyMutants() map[string]activityPropertyMutant {
	return map[string]activityPropertyMutant{
		// The cancel request of the Scenario's path is taken at attempt 1.
		"cancelRequestedWhileStarted also wants the first attempt": {
			mutate: func(t *testing.T, m *umpirespb.Model) {
				f := activityFunction(t, m, "cancellation.property.cancelRequestedWhileStarted")
				narrowed(f, binary(umpirespb.Binary_OP_EQ, stateField(f, 0, "attempts"), expr(admIntValue(1))))
			},
			rows: []string{"cancelRequestedWhileStarted on cancellation at started-now-2-unset-unset-unset-unset-unlimited-requestCancel"},
		},
		// The one worker answer on the path to a canceled activity is the canceled answer.
		"canceledByWorker is about completion": {
			mutate: func(_ *testing.T, m *umpirespb.Model) {
				admProperty(m, "cancellation", "canceledByWorker").When = &umpirespb.Property_WhenAction{WhenAction: "respondCompleted"}
			},
			rows:  []string{"canceledByWorker on cancellation at started-now-1-unset-unset-unset-unset-unlimited-respondCompleted"},
			paths: []string{"cancel"},
		},
		// No path that reads this Property terminates the activity. The free search of the product
		// that verifies it does: it terminates a scheduled activity.
		"pausedIsNotDispatched also forbids a terminate": {
			mutate: func(t *testing.T, m *umpirespb.Model) {
				f := activityFunction(t, m, "activityProduct.property.activityProduct.pausedIsNotDispatched")
				narrowed(f, binary(umpirespb.Binary_OP_NE, stateField(f, 1, "phase"),
					expr(admEnum("temporal.features.activity.standalone.product.Phase", "terminated"))))
			},
			rows: []string{
				"activityProduct.pausedIsNotDispatched on activityProduct at scheduled-terminate",
				"activityProduct.pausedIsNotDispatched on activitySystem at scheduled-now-0-unset-unset-unset-unset-unlimited-terminate",
				"activityProduct.pausedIsNotDispatched on completion at scheduled-now-0-unset-unset-unset-unset-unlimited-terminate",
				"activityProduct.pausedIsNotDispatched on retryFailures at scheduled-now-0-unset-unset-unset-unset-unlimited-terminate",
				"activityProduct.pausedIsNotDispatched on cancellation at scheduled-now-0-unset-unset-unset-unset-unlimited-terminate",
				"activityProduct.pausedIsNotDispatched on pausing at scheduled-now-0-unset-unset-unset-unset-unlimited-terminate",
				"activityProduct.pausedIsNotDispatched on dispatchEligibility at scheduled-now-0-unset-unset-unset-unset-unlimited-terminate",
				"activityProduct.pausedIsNotDispatched on timeouts at scheduled-now-0-unset-unset-unset-unset-unlimited-terminate",
			},
			generated: []string{"activityProduct.pausedIsNotDispatched"},
		},
		// The one attempt start on the cross-entity path is the first attempt's.
		"startedByPollingWorker also wants the first attempt": {
			mutate: func(t *testing.T, m *umpirespb.Model) {
				f := activityFunction(t, m, "standaloneActivity.property.startedByPollingWorker")
				narrowed(f, binary(umpirespb.Binary_OP_EQ, field(stateField(f, 0, "activity"), "attempts"), expr(admIntValue(1))))
			},
			rows: []string{"startedByPollingWorker on standaloneActivity at scheduled-now-1-unset-unset-unset-unset-unlimited_polling-poll"},
		},
		// The backoff on the cross-entity path is taken while the worker polls, so a claim about the
		// activity's own backoff holds there too, and is read on a step of the path.
		"startedByPollingWorker is about the backoff": {
			mutate: func(_ *testing.T, m *umpirespb.Model) {
				admProperty(m, "standaloneActivity", "startedByPollingWorker").When = &umpirespb.Property_WhenAction{WhenAction: "activity_backoff"}
			},
			rows: []string{
				"startedByPollingWorker on standaloneActivity at scheduled-now-0-unset-unset-unset-unset-unlimited_polling-poll",
				"startedByPollingWorker on standaloneActivity at scheduled-backoff-1-unset-unset-unset-unset-unlimited_stopped-activity_backoff",
			},
		},
	}
}

func TestActivityPropertyRowsCatchWhatThePathsMiss(t *testing.T) {
	for name, mutant := range activityPropertyMutants() {
		t.Run(name, func(t *testing.T) {
			m := proto.Clone(activityModel(t)).(*umpirespb.Model)
			mutant.mutate(t, m)
			require.NoError(t, ir.Validate(m))
			require.Equal(t, mutant.paths, pathDisagreements(t, m))
			require.Equal(t, mutant.generated, generatedDisagreements(t, m))
			differing := propertyRowDisagreements(t, activityModel(t), m)
			require.NotEmpty(t, differing)
			for _, row := range mutant.rows {
				require.True(t, slices.ContainsFunc(differing, func(d string) bool { return strings.HasPrefix(d, row+":") }),
					"%s is not among the %d rows that differ", row, len(differing))
			}
		})
	}
}

func TestActivityBoundedRowsMatchBatchCheck(t *testing.T) {
	activityBoundedRowsMatchBatchCheck(t, activityModel(t))
}

func activityBoundedRowsMatchBatchCheck(t *testing.T, baseline *umpirespb.Model) {
	t.Helper()
	require.NoError(t, ir.Validate(baseline))
	for owner, names := range map[string][]string{
		"activityProduct":     {"activityProduct"},
		"activitySystem":      {"activityProduct", "activitySystem"},
		"completion":          {"activityProduct", "completion"},
		"retryFailures":       {"activityProduct", "retryFailures"},
		"cancellation":        {"activityProduct", "cancellation"},
		"pausing":             {"activityProduct", "pausing"},
		"dispatchEligibility": {"activityProduct", "dispatchEligibility"},
		"timeouts":            {"activityProduct", "timeouts"},
		"byIDCancellation":    {"byIDCancellation"},
		"standaloneActivity":  {"activityProduct", "activitySystem", "activityWorker"},
	} {
		t.Run("owner closure/"+owner, func(t *testing.T) {
			w := newPropertyRowWalk(t, baseline, owner)
			declared := proto.Clone(baseline).(*umpirespb.Model)
			declared.Scenarios, declared.Queries = nil, nil
			for _, b := range []*binding{w.checker.first, w.checker.fresh} {
				var got []string
				for name := range b.machines {
					got = append(got, name)
				}
				slices.Sort(got)
				require.Equal(t, names, got)
				require.NotContains(t, b.machines, "heartbeatRetry")
				require.NotContains(t, b.machines, "resetSettlement")
				require.True(t, proto.Equal(declared, b.model))
				require.Same(t, b.model, b.in.Model())
				require.Equal(t, DefaultScope.Ceilings, b.in.Ceilings())
			}
			require.NotSame(t, w.checker.first.in, w.checker.fresh.in)
			for name, machine := range w.checker.first.machines {
				require.NotSame(t, machine.Table, w.checker.fresh.machines[name].Table)
			}
		})
	}
	for _, dependency := range []string{"activityProduct", "activityWorker"} {
		bad := proto.Clone(baseline).(*umpirespb.Model)
		bad.Machines = slices.DeleteFunc(bad.Machines, func(m *umpirespb.Machine) bool { return m.GetName() == dependency })
		_, err := propertyRowMachines(bad, "standaloneActivity")
		require.ErrorContains(t, err, "undeclared subject "+dependency)
	}
	_, err := propertyRowMachines(baseline, "missing")
	require.ErrorContains(t, err, "undeclared subject missing")
	replacing := proto.Clone(baseline).(*umpirespb.Model)
	replacing.Compositions[0].Members[1].Replaces = "polling"
	closure, err := propertyRowMachines(replacing, "standaloneActivity")
	require.NoError(t, err)
	var replacementNames []string
	for _, machine := range closure {
		replacementNames = append(replacementNames, machine.GetName())
	}
	slices.Sort(replacementNames)
	require.Equal(t, []string{"activityProduct", "activitySystem", "activityWorker", "polling"}, replacementNames)
	replacing.Compositions[0].Members[1].Replaces = "missingReplacement"
	_, err = propertyRowMachines(replacing, "standaloneActivity")
	require.ErrorContains(t, err, "undeclared subject missingReplacement")
	selected := map[string][]string{
		"activityProduct": {"started-heartbeat", "started-respondFailed-retryable", "scheduled-terminate"},
		"activitySystem": {
			"started-now-2-unset-unset-unset-unset-unlimited-requestCancel",
			"started-now-1-unset-unset-unset-unset-unlimited-respondCanceled",
			"started-now-1-unset-unset-unset-unset-unlimited-respondCompleted",
			"scheduled-now-0-unset-unset-unset-unset-unlimited-terminate",
		},
		"cancellation": {
			"started-now-2-unset-unset-unset-unset-unlimited-requestCancel",
			"started-now-1-unset-unset-unset-unset-unlimited-respondCanceled",
			"started-now-1-unset-unset-unset-unset-unlimited-respondCompleted",
			"scheduled-now-0-unset-unset-unset-unset-unlimited-terminate",
		},
		"completion":          {"scheduled-now-0-unset-unset-unset-unset-unlimited-terminate"},
		"retryFailures":       {"scheduled-now-0-unset-unset-unset-unset-unlimited-terminate"},
		"pausing":             {"scheduled-now-0-unset-unset-unset-unset-unlimited-terminate"},
		"dispatchEligibility": {"scheduled-now-0-unset-unset-unset-unset-unlimited-terminate"},
		"timeouts":            {"scheduled-now-0-unset-unset-unset-unset-unlimited-terminate"},
		"standaloneActivity": {
			"scheduled-now-1-unset-unset-unset-unset-unlimited_polling-poll",
			"scheduled-now-0-unset-unset-unset-unset-unlimited_polling-poll",
			"scheduled-backoff-1-unset-unset-unset-unset-unlimited_stopped-activity_backoff",
		},
	}
	pristine := proto.Clone(baseline).(*umpirespb.Model)
	want := irPropertyRowsBatch(t, baseline, selected)
	require.Len(t, want, 3*3+4*32+4*5+4+6+4+6+5+3)
	require.Equal(t, want, irPropertyRows(t, baseline, selected))
	batchSummary := propertyRowsSummary{Rows: len(want), Tally: map[string]propertyTally{}, Failing: map[string]int{}, Digests: map[string]string{},
		Owners: map[string]propertyRowInventory{
			"activityProduct":     {Rows: 210, Properties: 3, Machine: true},
			"activitySystem":      {Rows: 121176, Properties: 32, Machine: true, Refined: "activityProduct"},
			"completion":          {Rows: 121176, Properties: 4, Machine: true, Refined: "activityProduct"},
			"retryFailures":       {Rows: 121176, Properties: 6, Machine: true, Refined: "activityProduct"},
			"cancellation":        {Rows: 121176, Properties: 5, Machine: true, Refined: "activityProduct"},
			"pausing":             {Rows: 121176, Properties: 4, Machine: true, Refined: "activityProduct"},
			"dispatchEligibility": {Rows: 121176, Properties: 6, Machine: true, Refined: "activityProduct"},
			"timeouts":            {Rows: 121176, Properties: 5, Machine: true, Refined: "activityProduct"},
			"standaloneActivity":  {Rows: 43858, Properties: 1},
		}}
	batchLines := map[string][]string{}
	for key, side := range want {
		claim, at, _ := strings.Cut(key, " at ")
		batchLines[claim] = append(batchLines[claim], fmt.Sprintf("%s: %t %s", at, side.About, side.Outcome))
		counts := batchSummary.Tally[claim]
		switch {
		case !side.About:
			counts.NotAbout++
		case side.Outcome == umpire.VerifiedWithinLimits:
			counts.Holds++
		default:
			counts.Fails++
			phase, _, _ := strings.Cut(side.Witness.Initial.Value, "-")
			batchSummary.Failing[claim+": "+phase+" "+side.Witness.Steps[0].Action.Value]++
		}
		batchSummary.Tally[claim] = counts
	}
	for claim, lines := range batchLines {
		slices.Sort(lines)
		sum := sha256.Sum256([]byte(strings.Join(lines, "\n")))
		batchSummary.Digests[claim] = hex.EncodeToString(sum[:8])
	}
	require.Equal(t, batchSummary, summarizeSelectedPropertyRows(t, baseline, selected))
	require.False(t, want[rowKeyOf("canceledByWorker", "cancellation", "started-now-1-unset-unset-unset-unset-unlimited-respondCompleted")].About)
	require.True(t, want[rowKeyOf("canceledByWorker", "cancellation", "started-now-1-unset-unset-unset-unset-unlimited-respondCanceled")].About)
	mutants := activityPropertyMutants()
	mutants["the third heartbeat choice alone fails"] = activityPropertyMutant{mutate: func(t *testing.T, m *umpirespb.Model) {
		f := activityFunction(t, m, "activityProduct.property.activityProduct.pausedIsNotDispatched")
		narrowed(f, binary(umpirespb.Binary_OP_NE, stateField(f, 1, "phase"),
			expr(admEnum("temporal.features.activity.standalone.product.Phase", "timedOut"))))
	}}
	mutants["three choices are unselected by the property"] = activityPropertyMutant{mutate: func(_ *testing.T, m *umpirespb.Model) {
		admProperty(m, "activityProduct", "activityProduct.pausedIsNotDispatched").When = &umpirespb.Property_WhenAction{WhenAction: "poll"}
	}}
	for name, mutant := range mutants {
		t.Run(name, func(t *testing.T) {
			m := proto.Clone(baseline).(*umpirespb.Model)
			mutant.mutate(t, m)
			pristineMutant := proto.Clone(m).(*umpirespb.Model)
			batch := irPropertyRowsBatch(t, m, selected)
			bounded := irPropertyRows(t, m, selected)
			require.Equal(t, batch, bounded)
			require.NotEmpty(t, rowDisagreements(want, batch))
			require.True(t, proto.Equal(pristineMutant, m))
			if name == "the third heartbeat choice alone fails" {
				key := rowKeyOf("activityProduct.pausedIsNotDispatched", "activityProduct", "started-heartbeat")
				require.Equal(t, umpire.CounterexampleFound, bounded[key].Outcome)
				require.Len(t, bounded[key].Results, 1)
				require.Equal(t, "heartbeatExhausted", bounded[key].Results[0].Choice)
				rows := built(t, m)["activityProduct"]
				require.Len(t, row(t, rows, "started-heartbeat").Results, 3)
				require.Equal(t, "heartbeatExhausted", row(t, rows, "started-heartbeat").Results[2].Choice)
			}
			if name == "three choices are unselected by the property" {
				for _, row := range []string{"started-heartbeat", "started-respondFailed-retryable"} {
					side := bounded[rowKeyOf("activityProduct.pausedIsNotDispatched", "activityProduct", row)]
					require.Equal(t, umpire.VerifiedWithinLimits, side.Outcome)
					require.False(t, side.About)
					require.Nil(t, side.Witness)
					require.Empty(t, side.Results)
				}
			}
		})
	}
	// The synthetic Scenarios are private; neither reader mutates the baseline or another mutant.
	require.True(t, proto.Equal(pristine, baseline))
	require.Equal(t, want, irPropertyRowsBatch(t, baseline, selected))
}

// The sync poll is named as the activity's action it takes, so that action's one class is
// keyed as the composed step is. The cross-entity Property declared about that class answers every
// row and every path as the one declared about the action does.
func TestActivityCrossEntityClaimByClassAnswersAlike(t *testing.T) {
	m := proto.Clone(activityModel(t)).(*umpirespb.Model)
	var poll string
	for _, a := range m.GetActions() {
		if a.GetName() == "poll" {
			poll = a.GetId()
		}
	}
	require.NotEmpty(t, poll)
	admProperty(m, "standaloneActivity", "startedByPollingWorker").When = &umpirespb.Property_WhenClass{
		WhenClass: &umpirespb.ActionClass{Action: poll}}
	require.NoError(t, ir.Validate(m))
	require.Empty(t, pathDisagreements(t, m))
	require.Empty(t, generatedDisagreements(t, m))
	require.Empty(t, propertyRowDisagreements(t, activityModel(t), m))
}
