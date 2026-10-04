package lint

import (
	"fmt"
	"io"
	"maps"
	"slices"
	"strings"
	"text/tabwriter"

	umpirespb "go.temporal.io/server/api/umpire/v1"
	"go.temporal.io/server/tools/umpire/model"
)

// Modality is what one state and class of a machine is, as .plans/MODALITIES.md reads the table: a
// row is permission with its results fixed, a disabled pair of a system action is prohibition, and a
// disabled pair the author did not decide, or of a party action, which can always be sent, is the
// Model being silent.
type Modality string

const (
	May     Modality = "MAY"
	MustNot Modality = "MUST NOT"
	Silent  Modality = "?"
	// HoleRow is a pair whose value is a declared or undeclared hole: neither enabled nor disabled.
	HoleRow Modality = "hole"
)

// Cell is one reachable state and class of a machine: its modality, the results of a MAY, the guard
// of the decision that disabled it and the position of that decision, the named predicates the step
// function evaluated to decide, the hole kinds it is part of, and the claims that pin it.
type Cell struct {
	State      string
	Class      string
	Modality   Modality
	Results    []model.Result
	Guard      string
	Position   string
	Predicates []string
	Holes      []Kind
	Pinned     []string
}

// Rule is the cells of one class that read alike: one line of the per-operation table, labelled by
// the values of the state field the table groups by. Text is a MAY's results, or the guard of the
// decision that disabled the rule's pairs.
type Rule struct {
	Class string
	// Label is the field values of the rule's states, a value the rule holds only some states of with
	// how many.
	Label      string
	States     []string
	Modality   Modality
	Text       string
	Position   string
	Predicates []string
	Holes      []Kind
	Pinned     []string
}

// Table is a machine's per-operation modality table: every reachable cell, and the rules they group
// into, class by class. Field is the state field rules are labelled by: the state record's first
// enum-typed field, or none.
type Table struct {
	Machine string
	Field   string
	Cells   []Cell
	Rules   []Rule
}

// holes reads every machine's modality table, and from it the hole kinds H1-H5 and their tallies.
func (m *Model) holes() ([]*Table, []Tally, error) {
	views := map[string]*view{}
	var tables []*Table
	var tallies []Tally
	for _, name := range slices.Sorted(maps.Keys(m.Machines)) {
		v, err := m.view(name, views)
		if err != nil {
			return nil, nil, err
		}
		t, err := v.table()
		if err != nil {
			return nil, nil, fmt.Errorf("%s: %w", name, err)
		}
		tables = append(tables, t)
		tallies = append(tallies, v.tallies(t)...)
	}
	witness, err := m.witnessOnly()
	if err != nil {
		return nil, nil, err
	}
	return tables, append(tallies, witness...), nil
}

// view is one machine as the modality views read it.
type view struct {
	m     *Model
	name  string
	mm    *model.Machine
	field int
	// fieldName is the field rules group by, or the state type's own name where the state is an enum.
	fieldName string
	reachable []string
	reach     map[string]bool
	ends      map[string]bool
	rows      map[string]model.Row
	steps     map[string][]model.Value
	// naming is the same-step Properties that name each class, and reaching the progress claims a
	// class's rows reach the target of.
	naming     map[string][]string
	reaching   map[string][]string
	transition []*umpirespb.Property
	monitors   []*umpirespb.Monitor
	// carriers is, for each row, the product class carrying each of its results, nil for a stutter.
	carriers map[string][]*string
	product  *view
	// stepPins caches the claims that read a step, by the state before and the step's key.
	stepPins map[string][]string
	at       map[string]*umpirespb.Position
}

func (m *Model) view(name string, views map[string]*view) (*view, error) {
	if v, ok := views[name]; ok {
		return v, nil
	}
	mm := m.Machines[name]
	v := &view{m: m, name: name, mm: mm, field: -1, reach: map[string]bool{}, ends: map[string]bool{}, rows: map[string]model.Row{},
		steps: map[string][]model.Value{}, naming: map[string][]string{}, reaching: map[string][]string{}, carriers: map[string][]*string{},
		stepPins: map[string][]string{}, at: map[string]*umpirespb.Position{}}
	views[name] = v
	v.stateField()
	for _, s := range mm.Table.Reachable {
		v.reach[s] = true
	}
	for _, s := range mm.Table.States {
		if v.reach[s] {
			v.reachable = append(v.reachable, s)
		}
	}
	for _, s := range mm.Table.Ends {
		v.ends[s] = true
	}
	for _, r := range mm.Table.Rows {
		v.rows[r.Key] = r
	}
	for _, t := range mm.Transitions {
		v.steps[t.Row] = t.Steps
	}
	for _, b := range mm.Decl.GetSteps() {
		v.at[m.actions[b.GetAction()].GetName()] = b.GetPosition()
	}
	if err := v.claims(); err != nil {
		return nil, err
	}
	if r := mm.Decl.GetRefines(); r != nil && mm.Rejected == nil && m.Machines[r.GetProduct()] != nil {
		product, err := m.view(r.GetProduct(), views)
		if err != nil {
			return nil, err
		}
		v.product = product
		for _, rr := range mm.Refinement {
			v.carriers[rr.Key] = append(v.carriers[rr.Key], rr.Product)
		}
	}
	return v, nil
}

// stateField finds the field rules group by: the state record's first field of an enum type, or the
// state itself where its type is an enum.
func (v *view) stateField() {
	decl := v.m.declared(v.mm.Decl.GetStateType())
	if decl.GetEnum() != nil {
		v.fieldName = short(decl.GetName())
		return
	}
	for i, f := range decl.GetRecord().GetFields() {
		if v.m.declared(f.GetType().GetNamed()).GetEnum() != nil {
			v.field, v.fieldName = i, f.GetName()
			return
		}
	}
}

// phase is the value of the grouping field at a state, or "" where the machine has none.
func (v *view) phase(state string) string {
	s, _ := v.mm.State(state)
	switch {
	case v.field >= 0:
		return s.Fields[v.field].Key()
	case v.fieldName != "":
		return state
	default:
		return ""
	}
}

// claims indexes the claims that pin the machine's cells: the same-step Properties naming each class,
// of the machine or of a composition it is a member of, its transition Properties and monitors, and
// the progress claims each class reaches the target of.
func (v *view) claims() error {
	classes := map[string][]string{}
	for _, c := range v.mm.Classes {
		classes[c.Action.GetName()] = append(classes[c.Action.GetName()], c.Key)
	}
	for _, p := range v.m.IR.GetProperties() {
		if p.GetTransition() {
			if p.GetMachine() == v.name {
				v.transition = append(v.transition, p)
			}
			continue
		}
		named, err := v.named(p, classes)
		if err != nil {
			return err
		}
		for _, c := range named {
			v.naming[c] = append(v.naming[c], p.GetName())
		}
	}
	v.monitors = v.mm.Monitors
	for _, p := range v.m.IR.GetProgress() {
		if p.GetMachine() != v.name {
			continue
		}
		if err := v.reaches(p); err != nil {
			return err
		}
	}
	return nil
}

// reaches notes a progress claim against each class a reachable row of which steps into its target
// from outside it.
func (v *view) reaches(p *umpirespb.Progress) error {
	to := map[string]bool{}
	for _, s := range v.reachable {
		value, _ := v.mm.State(s)
		b, err := v.m.In.Call(p.GetTo(), []model.Value{value}, p.GetPosition())
		if err != nil && !model.Unknown(err) {
			return err
		}
		to[s] = err == nil && b.Bool
	}
	for _, r := range v.mm.Table.Rows {
		if !v.reach[r.Source] || to[r.Source] || slices.Contains(v.reaching[r.Action], p.GetName()) {
			continue
		}
		if slices.ContainsFunc(r.Results, func(res model.Result) bool { return to[res.State] }) {
			v.reaching[r.Action] = append(v.reaching[r.Action], p.GetName())
		}
	}
	return nil
}

// named is the classes of the machine a same-step Property names: of its own, by class, by action or
// every class; of a composition the machine is a member of, the classes of the member actions its
// class or action is, a sync taking one action of each member.
func (v *view) named(p *umpirespb.Property, classes map[string][]string) ([]string, error) {
	if p.GetMachine() == v.name {
		switch {
		case p.GetWhenClass() != nil:
			key, err := v.m.classKey(p.GetWhenClass())
			if err != nil {
				return nil, err
			}
			return []string{key}, nil
		case p.GetWhenAction() != "":
			return classes[p.GetWhenAction()], nil
		default:
			var all []string
			for _, c := range v.mm.Classes {
				all = append(all, c.Key)
			}
			return all, nil
		}
	}
	var out []string
	for _, c := range v.m.IR.GetCompositions() {
		if c.GetName() != p.GetMachine() {
			continue
		}
		action := p.GetWhenAction()
		if p.GetWhenClass() != nil {
			action = v.m.actions[p.GetWhenClass().GetAction()].GetName()
		}
		for _, member := range c.GetMembers() {
			if member.GetMachine() == v.name {
				out = append(out, v.memberClasses(c, member, action, classes)...)
			}
		}
	}
	slices.Sort(out)
	return slices.Compact(out), nil
}

// memberClasses is the classes of a composition's member that a composed action names: every class
// for no action, the member's own action `<field>_<action>`, or its move of a sync.
func (v *view) memberClasses(c *umpirespb.Composition, member *umpirespb.Member, action string, classes map[string][]string) []string {
	var out []string
	if action == "" {
		for _, keys := range classes {
			out = append(out, keys...)
		}
		return out
	}
	if own, ok := strings.CutPrefix(action, member.GetField()+"_"); ok {
		out = append(out, classes[own]...)
	}
	for _, s := range c.GetSyncs() {
		for _, move := range []*umpirespb.SyncMove{s.GetFirst(), s.GetSecond()} {
			if s.GetName() == action && move.GetMember() == member.GetField() {
				out = append(out, classes[move.GetAction()]...)
			}
		}
	}
	return out
}

// classKey keys an IR action class as the reader keys its classes: the action's name, then each
// input's key.
func (m *Model) classKey(c *umpirespb.ActionClass) (string, error) {
	parts := []string{m.actions[c.GetAction()].GetName()}
	for _, in := range c.GetInputs() {
		v, err := m.In.Eval(&umpirespb.Expr{Kind: &umpirespb.Expr_Literal{Literal: in}})
		if err != nil {
			return "", err
		}
		parts = append(parts, v.Key())
	}
	return strings.Join(parts, "-"), nil
}

// readers is the transition Properties and monitors of the machine whose evaluation at the step from
// before reads the step: the claims that constrain where a step from there may go.
func (v *view) readers(before string, step model.Value) ([]string, error) {
	key := before + "|" + step.Key()
	if pins, ok := v.stepPins[key]; ok {
		return pins, nil
	}
	state, _ := v.mm.State(before)
	var pins []string
	for _, p := range v.transition {
		_, read, err := v.m.In.Reads(p.GetHolds(), []model.Value{state, step}, p.GetPosition())
		if err != nil && !model.Unknown(err) {
			return nil, err
		}
		if err == nil && read[1] {
			pins = append(pins, p.GetName())
		}
	}
	for _, mo := range v.monitors {
		read, err := v.monitorReads(mo, state, step)
		if err != nil {
			return nil, err
		}
		if read {
			pins = append(pins, mo.GetName())
		}
	}
	v.stepPins[key] = pins
	return pins, nil
}

// monitorReads is whether a monitor's next state reads the step from some monitor state.
func (v *view) monitorReads(mo *umpirespb.Monitor, state, step model.Value) (bool, error) {
	states, err := v.m.In.Members(mo.GetState())
	if err != nil {
		return false, err
	}
	for _, ms := range states {
		_, read, err := v.m.In.Reads(mo.GetNext(), []model.Value{ms, state, step}, mo.GetPosition())
		if err != nil && !model.Unknown(err) {
			return false, err
		}
		if err == nil && read[2] {
			return true, nil
		}
	}
	return false, nil
}

// enabledPins is the claims that pin a row: the same-step Properties naming its class, the progress
// claims its class reaches, the transition claims that read its results, and through a refinement
// the product's pins of each carrying row.
func (v *view) enabledPins(state, class string) ([]string, error) {
	key := state + "-" + class
	pins := append(slices.Clone(v.naming[class]), v.reaching[class]...)
	for _, step := range v.steps[key] {
		read, err := v.readers(state, step)
		if err != nil {
			return nil, err
		}
		pins = append(pins, read...)
	}
	if v.product != nil {
		from, err := v.mapped(state)
		if err != nil {
			return nil, err
		}
		for _, carrier := range v.carriers[key] {
			if carrier == nil {
				continue
			}
			carried, err := v.product.enabledPins(from, *carrier)
			if err != nil {
				return nil, err
			}
			pins = append(pins, carried...)
		}
	}
	slices.Sort(pins)
	return slices.Compact(pins), nil
}

// disabledPins is the transition claims that pin a disabled pair at a state: those that read a step
// from there, of the machine and through a refinement of its product. A step that keeps the state is
// asked, since the pair has no step of its own.
func (v *view) disabledPins(state string) ([]string, error) {
	s, _ := v.mm.State(state)
	outcomes, err := v.m.In.Members(&umpirespb.TypeRef{Ref: &umpirespb.TypeRef_Named{Named: v.mm.Decl.GetOutcomeType()}})
	if err != nil {
		return nil, err
	}
	stay := model.Value{Kind: model.RecordValue, Type: model.StepType,
		Fields: []model.Value{outcomes[0], s, {Kind: model.ListValue}, {Kind: model.TextValue}}}
	pins, err := v.readers(state, stay)
	if err != nil {
		return nil, err
	}
	pins = slices.Clone(pins)
	if v.product != nil {
		from, err := v.mapped(state)
		if err != nil {
			return nil, err
		}
		carried, err := v.product.disabledPins(from)
		if err != nil {
			return nil, err
		}
		pins = append(pins, carried...)
	}
	slices.Sort(pins)
	return slices.Compact(pins), nil
}

// mapped is the product state a state of a refining machine reads as.
func (v *view) mapped(state string) (string, error) {
	s, _ := v.mm.State(state)
	r := v.mm.Decl.GetRefines()
	p, err := v.m.In.Call(r.GetMap(), []model.Value{s}, v.mm.Decl.GetPosition())
	return p.Key(), err
}

// table reads every reachable cell of the machine and groups them into rules.
func (v *view) table() (*Table, error) {
	t := &Table{Machine: v.name, Field: v.fieldName}
	for _, c := range v.mm.Classes {
		for _, s := range v.reachable {
			cell, err := v.cell(s, c)
			if err != nil {
				return nil, err
			}
			t.Cells = append(t.Cells, cell)
		}
	}
	t.Rules = v.rules(t.Cells)
	return t, nil
}

func (v *view) cell(state string, c model.Class) (Cell, error) {
	cell := Cell{State: state, Class: c.Key}
	why, err := v.m.In.Why(v.mm, state, c.Key)
	if err != nil {
		return Cell{}, err
	}
	last, decided := why.Last()
	for _, d := range why.Decisions {
		if d.Nested {
			continue
		}
		for _, f := range d.Calls {
			if name := short(f); !slices.Contains(cell.Predicates, name) {
				cell.Predicates = append(cell.Predicates, name)
			}
		}
	}
	if decided {
		cell.Position = last.Position
	}
	row, enabled := v.rows[state+"-"+c.Key]
	switch {
	case why.Hole != nil:
		cell.Modality, cell.Position = HoleRow, why.Hole.Position
		return cell, nil
	case enabled:
		cell.Modality, cell.Results = May, row.Results
		cell.Pinned, err = v.enabledPins(state, c.Key)
		return cell, err
	default:
	}
	cell.Modality, cell.Guard = MustNot, guard(last, decided)
	if !decided {
		cell.Position = where(v.at[c.Action.GetName()])
	}
	if decided && (last.Wildcard || !last.Match() && !last.State) {
		cell.Holes = append(cell.Holes, DisabledByDefault)
	}
	system := c.Action.GetParty() == "system" || c.Action.GetTimer() || c.Action.GetInternal()
	if !system && !v.ends[state] {
		cell.Holes = append(cell.Holes, SilentRejection)
	}
	if len(cell.Holes) > 0 {
		cell.Modality = Silent
	}
	if cell.Pinned, err = v.disabledPins(state); err != nil {
		return Cell{}, err
	}
	// An end state is where nothing more is to happen, as H2 reads it too.
	if system && v.m.options.MustNotPinned && len(cell.Pinned) == 0 && !v.ends[state] {
		cell.Holes = append(cell.Holes, MustNotPinned)
	}
	return cell, nil
}

// guard spells the decision that disabled a pair: the condition of an `if`, negated where its else
// was taken, or the scrutinee of a `match` and the pattern of the case taken. A step function that
// decided nothing is disabled everywhere.
func guard(d model.Decision, decided bool) string {
	switch {
	case !decided:
		return "always"
	case d.Match():
		m := d.Expr.GetMatch()
		return operand(m.GetScrutinee()) + " is " + spellPattern(m.GetCases()[d.Case].GetPattern())
	case d.Then:
		return spell(d.Expr.GetIf().GetCondition())
	default:
		return "!" + operand(d.Expr.GetIf().GetCondition())
	}
}

// results spells what a MAY does: each result's choice, outcome, the value of the grouping field it
// lands in, or `itself` where it keeps the state, and its facts.
func (v *view) results(state string, rs []model.Result) string {
	parts := make([]string, len(rs))
	for i, r := range rs {
		to := r.State
		if r.State == state {
			to = "itself"
		} else if p := v.phase(r.State); p != "" {
			to = p
		}
		s := r.Outcome + " -> " + to
		if len(r.Facts) > 0 {
			s += " [" + strings.Join(r.Facts, ", ") + "]"
		}
		if r.Choice != "" {
			s = r.Choice + ": " + s
		}
		parts[i] = s
	}
	return strings.Join(parts, "; ")
}

// rules groups each class's cells that read alike, in the order of their first state in the catalog.
func (v *view) rules(cells []Cell) []Rule {
	total := map[string]int{}
	for _, s := range v.reachable {
		total[v.phase(s)]++
	}
	var out []Rule
	index := map[string]int{}
	for _, c := range cells {
		text := c.Guard
		if c.Modality == May {
			text = v.results(c.State, c.Results)
		}
		key := strings.Join([]string{c.Class, string(c.Modality), text, c.Position, strings.Join(c.Predicates, ","),
			fmt.Sprint(c.Holes), strings.Join(c.Pinned, ",")}, "|")
		i, ok := index[key]
		if !ok {
			i = len(out)
			index[key] = i
			out = append(out, Rule{Class: c.Class, Modality: c.Modality, Text: text, Position: c.Position,
				Predicates: c.Predicates, Holes: c.Holes, Pinned: c.Pinned})
		}
		out[i].States = append(out[i].States, c.State)
	}
	for i := range out {
		out[i].Label = v.label(out[i].States, total)
	}
	return out
}

// label names a rule's states by the grouping field's values, in catalog order, a value the rule
// holds only some reachable states of with how many.
func (v *view) label(states []string, total map[string]int) string {
	if v.fieldName == "" {
		if len(states) == len(v.reachable) {
			return "every state"
		}
		return fmt.Sprintf("%d of %d states", len(states), len(v.reachable))
	}
	held := map[string]int{}
	var order []string
	for _, s := range states {
		p := v.phase(s)
		if held[p] == 0 {
			order = append(order, p)
		}
		held[p]++
	}
	parts := make([]string, len(order))
	for i, p := range order {
		parts[i] = p
		if held[p] < total[p] {
			parts[i] = fmt.Sprintf("%s (%d of %d)", p, held[p], total[p])
		}
	}
	return strings.Join(parts, ", ")
}

// tallies reads the hole kinds of a machine off its table. A finding is a class and the values of
// the grouping field its triggering pairs are in, never a state: H1, H2 and H5 count the classes with
// disabled pairs, H3 the classes with enabled ones.
func (v *view) tallies(t *Table) []Tally {
	kinds := []Kind{DisabledByDefault, SilentRejection}
	if v.m.options.MustNotPinned {
		kinds = append(kinds, MustNotPinned)
	}
	h := holeCount{v: v, population: map[Kind]map[string]bool{}, holes: map[Kind]map[string]*classHole{}}
	for _, c := range t.Cells {
		if c.Modality == May || c.Modality == HoleRow {
			continue
		}
		h.note(DisabledByDefault, c)
		if v.ends[c.State] {
			continue
		}
		h.note(SilentRejection, c)
		if v.m.options.MustNotPinned && !slices.Contains(c.Holes, SilentRejection) {
			h.note(MustNotPinned, c)
		}
	}
	var out []Tally
	for _, k := range kinds {
		out = append(out, h.tally(k, t.Cells))
	}
	return append(out, v.unconstrained(t.Cells))
}

// classHole is one class's pairs of a hole kind: the field values they are in, the position of the
// first, and their guards.
type classHole struct {
	phases   []string
	position string
	guards   []string
}

// holeCount counts the classes of each hole kind's population and collects its findings' pairs.
type holeCount struct {
	v          *view
	population map[Kind]map[string]bool
	holes      map[Kind]map[string]*classHole
}

var holeMessages = map[Kind]string{
	DisabledByDefault: "disabled by a default arm, which no author decided",
	SilentRejection:   "disabled where a party may send it, so the Model is silent on what it is answered",
	MustNotPinned:     "disabled, and no transition claim pins it",
}

// note counts a disabled cell in a kind's population, and among its findings when it has the hole.
func (h holeCount) note(k Kind, c Cell) {
	if h.population[k] == nil {
		h.population[k], h.holes[k] = map[string]bool{}, map[string]*classHole{}
	}
	h.population[k][c.Class] = true
	if !slices.Contains(c.Holes, k) {
		return
	}
	hole, ok := h.holes[k][c.Class]
	if !ok {
		hole = &classHole{position: c.Position}
		h.holes[k][c.Class] = hole
	}
	if p := h.v.phase(c.State); !slices.Contains(hole.phases, p) {
		hole.phases = append(hole.phases, p)
	}
	if !slices.Contains(hole.guards, c.Guard) {
		hole.guards = append(hole.guards, c.Guard)
	}
}

// tally is a kind's findings, one per class in the table's class order.
func (h holeCount) tally(k Kind, cells []Cell) Tally {
	t := Tally{Kind: k, Owner: h.v.name, Population: len(h.population[k])}
	seen := map[string]bool{}
	for _, c := range cells {
		hole, ok := h.holes[k][c.Class]
		if !ok || seen[c.Class] {
			continue
		}
		seen[c.Class] = true
		subject := c.Class
		if h.v.fieldName != "" {
			subject += " in " + strings.Join(hole.phases, ", ")
		}
		t.Findings = append(t.Findings, Finding{Kind: k, Owner: h.v.name, Subject: subject,
			Message: fmt.Sprintf("%s: %s (%s)", subject, holeMessages[k], strings.Join(hole.guards, "; ")), Position: hole.position})
	}
	return t
}

// unconstrained is H3: each class with enabled pairs none of which a claim pins.
func (v *view) unconstrained(cells []Cell) Tally {
	t := Tally{Kind: UnconstrainedResult, Owner: v.name}
	enabled := map[string][]string{}
	pinned := map[string]bool{}
	var classes []string
	for _, c := range cells {
		if c.Modality != May {
			continue
		}
		if len(enabled[c.Class]) == 0 {
			classes = append(classes, c.Class)
		}
		if p := v.phase(c.State); !slices.Contains(enabled[c.Class], p) {
			enabled[c.Class] = append(enabled[c.Class], p)
		}
		pinned[c.Class] = pinned[c.Class] || len(c.Pinned) > 0
	}
	t.Population = len(classes)
	for _, class := range classes {
		if !pinned[class] {
			t.Findings = append(t.Findings, Finding{Kind: UnconstrainedResult, Owner: v.name, Subject: class,
				Message:  fmt.Sprintf("%s: MAY in %s, and no claim constrains its results", class, strings.Join(enabled[class], ", ")),
				Position: where(v.at[v.actionOf(class)])})
		}
	}
	return t
}

// actionOf is the name of the action a class key belongs to.
func (v *view) actionOf(class string) string {
	for _, c := range v.mm.Classes {
		if c.Key == class {
			return c.Action.GetName()
		}
	}
	return ""
}

// witnessOnly is H4: a same-step Property that only find Queries over pinned Scenarios ask, a MUST on
// one path and not on the table.
func (m *Model) witnessOnly() ([]Tally, error) {
	scenarios := map[[2]string]*umpirespb.Scenario{}
	for _, s := range m.IR.GetScenarios() {
		scenarios[[2]string{s.GetMachine(), s.GetName()}] = s
	}
	asked := map[[2]string][]*umpirespb.Query{}
	for _, q := range m.IR.GetQueries() {
		key := [2]string{q.GetProperty().GetMachine(), q.GetProperty().GetName()}
		asked[key] = append(asked[key], q)
	}
	tallies := map[string]*Tally{}
	for _, p := range m.IR.GetProperties() {
		queries := asked[[2]string{p.GetMachine(), p.GetName()}]
		if p.GetTransition() || len(queries) == 0 {
			continue
		}
		t, ok := tallies[p.GetMachine()]
		if !ok {
			t = &Tally{Kind: WitnessOnly, Owner: p.GetMachine()}
			tallies[p.GetMachine()] = t
		}
		t.Population++
		witnessing := !slices.ContainsFunc(queries, func(q *umpirespb.Query) bool {
			s := scenarios[[2]string{q.GetScenario().GetMachine(), q.GetScenario().GetName()}]
			return q.GetForm() != umpirespb.Query_FORM_FIND || s.GetFree()
		})
		if witnessing {
			t.Findings = append(t.Findings, Finding{Kind: WitnessOnly, Owner: p.GetMachine(), Subject: p.GetName(),
				Message:  fmt.Sprintf("%s is asked only by find Queries over pinned Scenarios, so no Query holds the table to it", p.GetName()),
				Position: where(p.GetPosition())})
		}
	}
	var out []Tally
	for _, owner := range slices.Sorted(maps.Keys(tallies)) {
		out = append(out, *tallies[owner])
	}
	return out, nil
}

// notes is a rule's hole kinds, the claims that pin it and the predicates it was decided by.
func (r Rule) notes() string {
	var notes []string
	for _, h := range r.Holes {
		notes = append(notes, string(h))
	}
	if len(r.Pinned) > 0 {
		notes = append(notes, "pinned: "+strings.Join(r.Pinned, ", "))
	}
	if len(r.Predicates) > 0 {
		notes = append(notes, "by: "+strings.Join(r.Predicates, ", "))
	}
	return strings.Join(notes, "  ")
}

// WriteTables writes each machine's per-operation modality table: a block per machine, a line per
// rule, class by class.
func WriteTables(w io.Writer, r *Result) error {
	for _, t := range r.Tables {
		by := ""
		if t.Field != "" {
			by = " by " + t.Field
		}
		if _, err := fmt.Fprintf(w, "rules %s %s%s\n", r.File, t.Machine, by); err != nil {
			return err
		}
		tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
		class := ""
		for _, rule := range t.Rules {
			if rule.Class != class {
				class = rule.Class
				if _, err := fmt.Fprintf(tw, "  %s\n", class); err != nil {
					return err
				}
			}
			if _, err := fmt.Fprintf(tw, "    %s\t%s\t%s\t%s\t%s\n", rule.Label, rule.Modality, rule.Text, rule.Position,
				rule.notes()); err != nil {
				return err
			}
		}
		if err := tw.Flush(); err != nil {
			return err
		}
	}
	return nil
}
