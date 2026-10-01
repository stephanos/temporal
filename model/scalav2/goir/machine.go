package goir

import (
	"cmp"
	"errors"
	"slices"
	"strings"

	modelirspb "go.temporal.io/server/api/modelir/v1"
	"go.temporal.io/server/model/go/umpire"
)

// Members lists every value of a finite type in catalog order: an enum's cases in declaration order,
// each case with fields once per assignment of them; a record as the product of its fields; with the
// last field varying fastest in both, which is the order the Lean `Finite` derivation produces.
// A channel's contents are listed as its Channels section says. A catalog larger than the Members
// ceiling is refused from its count, before any of it is listed.
func (in *Interpreter) Members(t *modelirspb.TypeRef) ([]Value, error) {
	n, err := in.size(t)
	if err != nil {
		return nil, err
	}
	if err := in.within("members", in.ceilings.Members, n); err != nil {
		return nil, err
	}
	switch r := t.GetRef().(type) {
	case *modelirspb.TypeRef_Bool:
		return []Value{{Kind: BoolValue, Bool: false}, {Kind: BoolValue, Bool: true}}, nil
	case *modelirspb.TypeRef_IntRange:
		var out []Value
		for i := r.IntRange.GetLow(); i <= r.IntRange.GetHigh(); i++ {
			out = append(out, Value{Kind: IntValue, Int: i})
			// Past the high end of a range ending at math.MaxInt64, i++ would wrap around.
			if i == r.IntRange.GetHigh() {
				break
			}
		}
		return out, nil
	case *modelirspb.TypeRef_Named:
		out, err := in.declaredMembers(r.Named)
		if err != nil {
			return nil, err
		}
		return out, distinctKeys(in.types[r.Named].GetPosition(), r.Named, out)
	case *modelirspb.TypeRef_Channel:
		c, err := in.channel(r.Channel, nil)
		if err != nil {
			return nil, err
		}
		out, err := in.contents(c)
		if err != nil {
			return nil, err
		}
		return out, distinctKeys(c.GetPosition(), "channel "+r.Channel, out)
	default:
		return nil, &Error{Message: "a finite type is a named type, the Booleans, an integer range, or a channel's contents"}
	}
}

// distinctKeys refuses a catalog two of whose values share a key: tables, rows and Definition IDs name
// a value by its key alone, so the two would be one.
func distinctKeys(at *modelirspb.Position, catalog string, values []Value) error {
	seen := make(map[string]Value, len(values))
	for _, v := range values {
		if earlier, ok := seen[v.Key()]; ok {
			return errorAt(at, "the catalog of %s holds %s and %s, which share the key %q", catalog, earlier.spelled(), v.spelled(), v.Key())
		}
		seen[v.Key()] = v
	}
	return nil
}

// declaredMembers lists a declared type's catalog.
func (in *Interpreter) declaredMembers(name string) ([]Value, error) {
	decl, ok := in.types[name]
	if !ok {
		return nil, &Error{Message: "no type " + name}
	}
	switch s := decl.GetShape().(type) {
	case *modelirspb.Type_Enum:
		var out []Value
		for _, c := range s.Enum.GetCases() {
			assignments, err := in.product(c.GetFields())
			if err != nil {
				return nil, err
			}
			for _, fields := range assignments {
				out = append(out, Value{Kind: EnumValue, Type: name, Case: c.GetName(), Fields: fields})
			}
		}
		return out, nil
	case *modelirspb.Type_Record:
		assignments, err := in.product(s.Record.GetFields())
		if err != nil {
			return nil, err
		}
		out := make([]Value, len(assignments))
		for i, fields := range assignments {
			out[i] = Value{Kind: RecordValue, Type: name, Fields: fields}
		}
		return out, nil
	default:
		return nil, errorAt(decl.GetPosition(), "type %s has no shape", name)
	}
}

// product lists every assignment of fields, the last varying fastest, once their count is within the
// Members ceiling.
func (in *Interpreter) product(fields []*modelirspb.Field) ([][]Value, error) {
	n, err := in.sizeOfProduct(fields)
	if err != nil {
		return nil, err
	}
	if err := in.within("members", in.ceilings.Members, n); err != nil {
		return nil, err
	}
	out := [][]Value{{}}
	for _, f := range fields {
		members, err := in.Members(f.GetType())
		if err != nil {
			return nil, err
		}
		next := make([][]Value, 0, len(out)*len(members))
		for _, prefix := range out {
			for _, m := range members {
				next = append(next, append(slices.Clone(prefix), m))
			}
		}
		out = next
	}
	return out, nil
}

func named(name string) *modelirspb.TypeRef {
	return &modelirspb.TypeRef{Ref: &modelirspb.TypeRef_Named{Named: name}}
}

// Class is what the checks that read a machine's classes are given:
// class is one class of an action: the action and one assignment of its inputs.
type Class struct {
	Key    string
	Action *modelirspb.Action
	Inputs []Value
	step   string
	at     *modelirspb.Position
}

// Transition is one row of a machine's table as values: the state it leaves, its class, and its step
// records, in the row's result order.
type Transition struct {
	Row    string
	Source Value
	Class  Class
	Steps  []Value
}

// HoleRow is a state and class whose value is a hole: neither a row of the table nor a disabled pair.
// The table has no row for it, so its reachability and stuck state read the pair as if disabled, and
// a refinement covers the table's rows alone: a check reads the hole rows beside them.
type HoleRow struct {
	Row    string
	Source string
	Class  string
	Hole   *Hole
}

// Work counts what interpreting a machine took.
type Work struct {
	States      int
	Classes     int
	Evaluations int
}

// Machine is one machine of the IR, interpreted: its table and, for a refining machine, its
// refinement rows.
// It also holds the same rows as values, its hole rows, the monitors and assumptions it names, and,
// for a refining machine, why the refinement does not hold. Nothing here applies a monitor or an
// assumption: a check that reads the table alone answers a question about the rows, not about the
// machine's declarations.
type Machine struct {
	Decl        *modelirspb.Machine
	Table       *umpire.Table
	states      map[string]Value
	Refinement  []umpire.RefinementRow
	Rejected    error
	Classes     []Class
	Transitions []Transition
	Holes       []HoleRow
	Monitors    []*modelirspb.Monitor
	Assumptions []*modelirspb.Assumption
	Work        Work
}

// State is the value of a state key.
func (m *Machine) State(key string) (Value, bool) {
	v, ok := m.states[key]
	return v, ok
}

// Disabled is whether a state and class of the machine are a disabled pair: an empty list of steps,
// neither a row nor a hole row.
func (m *Machine) Disabled(state, class string) bool {
	if _, ok := m.states[state]; !ok || !slices.ContainsFunc(m.Classes, func(c Class) bool { return c.Key == class }) {
		return false
	}
	key := state + "-" + class
	return !slices.ContainsFunc(m.Table.RowsFrom(state), func(r umpire.Row) bool { return r.Key == key }) &&
		!slices.ContainsFunc(m.Holes, func(h HoleRow) bool { return h.Row == key })
}

// ReachableHoles is the hole rows whose state the table reaches.
func (m *Machine) ReachableHoles() []HoleRow {
	var out []HoleRow
	for _, h := range m.Holes {
		if slices.Contains(m.Table.Reachable, h.Source) {
			out = append(out, h)
		}
	}
	return out
}

// Build interprets every machine of a Model. A refining machine's refinement is checked against the
// machine it names, which must be in the Model.
// It interprets them within DefaultCeilings, and a refinement that does not hold is the machine's
// Rejected, not an error of the Model.
func Build(m *modelirspb.Model) (map[string]*Machine, error) {
	return NewInterpreter(m).build(m)
}

func (in *Interpreter) build(m *modelirspb.Model) (map[string]*Machine, error) {
	out := in.interpret(m)
	for _, failures := range []map[string]error{out.failed, out.unrefined} {
		for _, decl := range m.GetMachines() {
			if err := failures[decl.GetName()]; err != nil {
				return nil, err
			}
		}
	}
	return out.machines, nil
}

// interpretation is a Model's machines interpreted one by one, so that what one machine's
// declarations leave unread does not take the others with it: failed is why a machine has no table,
// and unrefined why a refining machine that has one has no refinement, neither held nor rejected.
type interpretation struct {
	machines  map[string]*Machine
	failed    map[string]error
	unrefined map[string]error
}

func (in *Interpreter) interpret(m *modelirspb.Model) interpretation {
	actions := map[string]*modelirspb.Action{}
	for _, a := range m.GetActions() {
		actions[a.GetId()] = a
	}
	out := interpretation{machines: map[string]*Machine{}, failed: map[string]error{}, unrefined: map[string]error{}}
	for _, decl := range m.GetMachines() {
		mm, err := in.machine(decl, actions)
		var limit *LimitError
		if errors.As(err, &limit) && limit.Machine == "" {
			limit.Machine = decl.GetName()
		}
		if err == nil {
			err = declarations(m, mm)
		}
		if err != nil {
			out.failed[decl.GetName()] = err
			continue
		}
		out.machines[decl.GetName()] = mm
	}
	for _, decl := range m.GetMachines() {
		mm, r := out.machines[decl.GetName()], decl.GetRefines()
		if mm == nil || r == nil {
			continue
		}
		product, ok := out.machines[r.GetProduct()]
		switch {
		case ok:
			out.unrefined[decl.GetName()] = in.refinement(mm, product)
		case out.failed[r.GetProduct()] != nil:
			out.unrefined[decl.GetName()] = out.failed[r.GetProduct()]
		default:
			out.unrefined[decl.GetName()] = errorAt(decl.GetPosition(), "%s refines %s, which the Model does not declare", decl.GetName(), r.GetProduct())
		}
	}
	return out
}

// declarations resolves the monitors and assumptions a machine names.
func declarations(m *modelirspb.Model, mm *Machine) error {
	for _, id := range mm.Decl.GetMonitors() {
		i := slices.IndexFunc(m.GetMonitors(), func(d *modelirspb.Monitor) bool { return d.GetId() == id })
		if i < 0 {
			return errorAt(mm.Decl.GetPosition(), "%s: no monitor %s", mm.Decl.GetName(), id)
		}
		mm.Monitors = append(mm.Monitors, m.GetMonitors()[i])
	}
	for _, id := range mm.Decl.GetAssumes() {
		i := slices.IndexFunc(m.GetAssumptions(), func(d *modelirspb.Assumption) bool { return d.GetId() == id })
		if i < 0 {
			return errorAt(mm.Decl.GetPosition(), "%s: no assumption %s", mm.Decl.GetName(), id)
		}
		mm.Assumptions = append(mm.Assumptions, m.GetAssumptions()[i])
	}
	return nil
}

func (in *Interpreter) machine(decl *modelirspb.Machine, actions map[string]*modelirspb.Action) (*Machine, error) {
	if err := in.preflight(decl, actions); err != nil {
		return nil, err
	}
	states, err := in.Members(named(decl.GetStateType()))
	if err != nil {
		return nil, err
	}
	outcomes, err := in.Members(named(decl.GetOutcomeType()))
	if err != nil {
		return nil, err
	}
	var facts []Value
	if decl.GetFactType() != "" {
		if facts, err = in.Members(named(decl.GetFactType())); err != nil {
			return nil, err
		}
	}
	classes, err := in.classes(decl, actions)
	if err != nil {
		return nil, err
	}
	spec := umpire.TableSpec{Machine: decl.GetName(), Owner: decl.GetName(), Family: umpire.Family(decl.GetFamily()),
		Entity: decl.GetEntity()}
	mm := &Machine{Decl: decl, states: map[string]Value{}, Classes: classes,
		Work: Work{States: len(states), Classes: len(classes), Evaluations: len(states) * len(classes)}}
	for _, s := range states {
		spec.States = append(spec.States, s.Key())
		mm.states[s.Key()] = s
	}
	for _, c := range classes {
		spec.Actions = append(spec.Actions, c.Key)
	}
	for _, o := range outcomes {
		spec.Outcomes = append(spec.Outcomes, o.Key())
	}
	for _, f := range facts {
		spec.Facts = append(spec.Facts, f.Key())
	}
	if err = in.rows(mm, states, &spec); err != nil {
		return nil, err
	}
	// The starts, the ends and the evidence are read one after another past any hole, so that a hole
	// in one hides neither a hole nor an error of the Model in the next.
	var unread unknowns
	if spec.Starts, spec.Ends, err = in.startsAndEnds(decl, states, mm.states, &unread); err != nil {
		return nil, err
	}
	names, _ := in.fieldNames(decl.GetStateType(), "")
	spec.StateFields = append(spec.StateFields, names...)
	if r := decl.GetRefines(); r != nil {
		// A refining machine carries the refined state it reads as in a field named after the refined
		// machine (`Umpire.Command.refinedProperty`).
		spec.StateFields = append(spec.StateFields, r.GetProduct())
	}
	if spec.Evidence, err = in.evidence(decl, facts, &unread); err != nil {
		return nil, err
	}
	if err := unread.err(); err != nil {
		return nil, err
	}
	mm.Table = umpire.NewTable(spec)
	return mm, nil
}

// preflight counts a machine's states, its classes of every bound action together, and their pairs,
// and refuses work past a ceiling before any of it is listed.
func (in *Interpreter) preflight(decl *modelirspb.Machine, actions map[string]*modelirspb.Action) error {
	states, err := in.size(named(decl.GetStateType()))
	if err != nil {
		return err
	}
	if err := in.within("members", in.ceilings.Members, states); err != nil {
		return err
	}
	classes, err := in.classCount(decl, actions)
	if err != nil {
		return err
	}
	return in.within("evaluations", in.ceilings.Evaluations, states.times(classes))
}

// classCount counts a machine's classes of every bound action together, refusing them past the
// Members ceiling, so no caller lists them first.
func (in *Interpreter) classCount(decl *modelirspb.Machine, actions map[string]*modelirspb.Action) (count, error) {
	var classes count
	for _, b := range decl.GetSteps() {
		a, ok := actions[b.GetAction()]
		if !ok {
			return count{}, errorAt(b.GetPosition(), "no action %s", b.GetAction())
		}
		n, err := in.sizeOfProduct(inputFields(a))
		if err != nil {
			return count{}, err
		}
		classes = classes.plus(n)
	}
	return classes, in.within("classes", in.ceilings.Members, classes)
}

func inputFields(a *modelirspb.Action) []*modelirspb.Field {
	fields := make([]*modelirspb.Field, len(a.GetInputs()))
	for i, p := range a.GetInputs() {
		fields[i] = &modelirspb.Field{Name: p.GetName(), Type: p.GetType()}
	}
	return fields
}

// classes lists every class of every bound action, sorted by key, and rejects a class two steps bind.
func (in *Interpreter) classes(decl *modelirspb.Machine, actions map[string]*modelirspb.Action) ([]Class, error) {
	if _, err := in.classCount(decl, actions); err != nil {
		return nil, err
	}
	var out []Class
	for _, b := range decl.GetSteps() {
		a, ok := actions[b.GetAction()]
		if !ok {
			return nil, errorAt(b.GetPosition(), "no action %s", b.GetAction())
		}
		assignments, err := in.product(inputFields(a))
		if err != nil {
			return nil, err
		}
		for _, inputs := range assignments {
			parts := []string{a.GetName()}
			for _, v := range inputs {
				parts = append(parts, v.Key())
			}
			out = append(out, Class{Key: strings.Join(parts, "-"), Action: a, Inputs: inputs, step: b.GetFunction(), at: b.GetPosition()})
		}
	}
	slices.SortStableFunc(out, func(x, y Class) int { return strings.Compare(x.Key, y.Key) })
	for i := 1; i < len(out); i++ {
		x, y := out[i-1], out[i]
		switch {
		case x.Key != y.Key:
		case x.Action.GetId() == y.Action.GetId() && slices.EqualFunc(x.Inputs, y.Inputs, Value.Equal):
			return nil, errorAt(y.at, "%s: two steps bind the action class %q", decl.GetName(), y.Key)
		default:
			return nil, errorAt(y.at, "%s: the classes %s and %s share the key %q", decl.GetName(), x.spelled(), y.spelled(), y.Key)
		}
	}
	return out, nil
}

func (c Class) spelled() string {
	if len(c.Inputs) == 0 {
		return c.Action.GetName()
	}
	return c.Action.GetName() + Value{Kind: RecordValue, Fields: c.Inputs}.spelled()
}

// rowKeys refuses a machine two of whose state and class pairs share a row key, as a state and a
// class whose keys hold a "-" can: rows, hole rows and disabled pairs are found by it.
func rowKeys(decl *modelirspb.Machine, states []Value, classes []Class) error {
	seen := make(map[string][2]string, len(states)*len(classes))
	for _, s := range states {
		for _, c := range classes {
			key := s.Key() + "-" + c.Key
			if earlier, ok := seen[key]; ok {
				return errorAt(decl.GetPosition(), "%s: the state %s with the class %s, and the state %s with the class %s, share the row key %q",
					decl.GetName(), earlier[0], earlier[1], s.Key(), c.Key, key)
			}
			seen[key] = [2]string{s.Key(), c.Key}
		}
	}
	return nil
}

// rows evaluates every step function once per state and class, states-major, keeping the enabled
// pairs as rows and rejecting a result outside the state domain.
// It also keeps each row's transition, and the pairs whose value is a hole as hole rows; a channel's
// delivery and loss are evaluated as transfers.
func (in *Interpreter) rows(mm *Machine, states []Value, spec *umpire.TableSpec) error {
	decl := mm.Decl
	if err := rowKeys(decl, states, mm.Classes); err != nil {
		return err
	}
	for _, s := range states {
		for _, c := range mm.Classes {
			key := s.Key() + "-" + c.Key
			steps, err := in.steps(decl, s, c)
			var hole *Hole
			if errors.As(err, &hole) {
				mm.Holes = append(mm.Holes, HoleRow{Row: key, Source: s.Key(), Class: c.Key, Hole: hole})
				continue
			}
			if err != nil {
				return err
			}
			if len(steps) == 0 {
				continue
			}
			row := umpire.Row{Key: key, Source: s.Key(), Action: c.Key}
			for _, step := range steps {
				res, err := in.result(mm, c, key, step)
				if err != nil {
					return err
				}
				row.Results = append(row.Results, res)
			}
			spec.Rows = append(spec.Rows, row)
			mm.Transitions = append(mm.Transitions, Transition{Row: key, Source: s, Class: c, Steps: steps})
		}
	}
	return nil
}

// result keys one step record of a row, rejecting a state outside the domain.
// It also rejects an outcome or a fact not of the machine's types.
func (in *Interpreter) result(m *Machine, c Class, row string, step Value) (umpire.Result, error) {
	decl := m.Decl
	if outcome := step.Fields[0]; !in.conforms(outcome, named(decl.GetOutcomeType())) {
		return umpire.Result{}, errorAt(c.at, "%s: row %s has outcome %s, which is no %s", decl.GetName(), row, outcome.Key(), decl.GetOutcomeType())
	}
	for _, f := range step.Fields[2].Items {
		if decl.GetFactType() == "" || !in.conforms(f, named(decl.GetFactType())) {
			return umpire.Result{}, errorAt(c.at, "%s: row %s records %s, which is no %s", decl.GetName(), row, f.Key(), cmp.Or(decl.GetFactType(), "fact of it"))
		}
	}
	next := step.Fields[1].Key()
	if v, ok := m.states[next]; !ok || !v.Equal(step.Fields[1]) {
		return umpire.Result{}, errorAt(c.at, "%s: row %s lands in %s, which is outside the state domain", m.Decl.GetName(), row, next)
	}
	res := umpire.Result{Outcome: step.Fields[0].Key(), State: next, Facts: []string{}, Because: step.Fields[3].Text}
	for _, f := range step.Fields[2].Items {
		res.Facts = append(res.Facts, f.Key())
	}
	return res, nil
}

func (in *Interpreter) steps(decl *modelirspb.Machine, s Value, c Class) ([]Value, error) {
	if c.Action.GetDelivers() != "" || c.Action.GetLoses() != "" {
		return in.transfer(decl, s, c)
	}
	results, err := in.Call(c.step, append([]Value{s}, c.Inputs...), c.at)
	if err != nil {
		return nil, err
	}
	return in.stepList(decl, c, results)
}

// stepList is the steps a bound function returned: a list of step records of four fields, each with a
// state of the machine's state type, a list of facts and an explanation; whether the state is in the
// domain is result's to say. Anything else is an error of
// the Model, never an empty list.
func (in *Interpreter) stepList(decl *modelirspb.Machine, c Class, v Value) ([]Value, error) {
	if v.Kind != ListValue {
		return nil, errorAt(c.at, "%s returns %s, not a list of steps", c.step, v.Key())
	}
	for _, step := range v.Items {
		switch {
		case step.Kind != RecordValue || step.Type != StepType:
			return nil, errorAt(c.at, "%s returns %s, not a list of steps", c.step, v.Key())
		case len(step.Fields) != len(stepFields):
			return nil, errorAt(c.at, "%s returns a step of %d fields, not %d", c.step, len(step.Fields), len(stepFields))
		case step.Fields[1].Type != decl.GetStateType():
			return nil, errorAt(c.at, "%s returns a step to %s, which is no %s", c.step, step.Fields[1].Key(), decl.GetStateType())
		case step.Fields[2].Kind != ListValue || step.Fields[3].Kind != TextValue:
			return nil, errorAt(c.at, "%s returns a step whose facts are no list or whose explanation is no string", c.step)
		default:
		}
	}
	return v.Items, nil
}

// unknowns collects the holes a declaration reaches as it is read at one value after another, each
// once. The reading goes on past a hole, so that an error of the Model at a later value is not lost
// behind it, and every hole it reaches is reported.
type unknowns struct{ holes []*Hole }

// note keeps a hole, for which it is nil, and is any other error itself.
func (u *unknowns) note(err error) error {
	var hole *Hole
	if !errors.As(err, &hole) {
		return err
	}
	if !slices.ContainsFunc(u.holes, func(h *Hole) bool { return *h == *hole }) {
		u.holes = append(u.holes, hole)
	}
	return nil
}

// err is the holes reached, as why the declaration is not read, or nil when it reached none.
func (u *unknowns) err() error {
	errs := holeErrors(u.holes)
	if len(errs) == 1 {
		return errs[0]
	}
	return errors.Join(errs...)
}

func holeErrors(holes []*Hole) []error {
	errs := make([]error, len(holes))
	for i, h := range holes {
		errs[i] = h
	}
	return errs
}

// startsAndEnds reads a machine's starts and the states it may end in. A hole either reaches is noted
// in unread and read past, and is no error here: the caller has no table for a machine with one.
func (in *Interpreter) startsAndEnds(decl *modelirspb.Machine, states []Value, domain map[string]Value, unread *unknowns) (
	starts, ends []string, err error) {
	for _, x := range decl.GetStarts() {
		v, err := in.Eval(x)
		if err != nil {
			if err = unread.note(err); err != nil {
				return nil, nil, err
			}
			continue
		}
		if _, ok := domain[v.Key()]; !ok {
			return nil, nil, errorAt(x.GetPosition(), "start %s is outside the state domain", v.Key())
		}
		starts = append(starts, v.Key())
	}
	if len(starts) == 0 && len(unread.holes) == 0 {
		return nil, nil, errorAt(decl.GetPosition(), "%s declares no start", decl.GetName())
	}
	if decl.GetEnds() == nil {
		return starts, nil, nil
	}
	end, evalErr := in.Eval(decl.GetEnds())
	if evalErr != nil {
		return starts, nil, unread.note(evalErr)
	}
	for _, s := range states {
		v, err := in.Apply(end, []Value{s})
		if err != nil {
			if err = unread.note(err); err != nil {
				return nil, nil, err
			}
			continue
		}
		if v.Kind != BoolValue {
			return nil, nil, errorAt(decl.GetEnds().GetPosition(), "%s: ends is %s at %s, not a Boolean", decl.GetName(), v.Key(), s.Key())
		}
		if v.Bool {
			ends = append(ends, s.Key())
		}
	}
	return starts, ends, nil
}

// evidence is one line per fact constructor, in catalog order: the constructor, and the recorded
// event or observation the evidence function names for it.
// It notes a hole in unread and reads past it, as startsAndEnds does.
func (in *Interpreter) evidence(decl *modelirspb.Machine, facts []Value, unread *unknowns) ([][2]string, error) {
	if decl.GetEvidence() == "" {
		return nil, nil
	}
	var out [][2]string
	for _, f := range facts {
		v, err := in.Call(decl.GetEvidence(), []Value{f}, decl.GetPosition())
		if err != nil {
			if err = unread.note(err); err != nil {
				return nil, err
			}
			continue
		}
		if !slices.ContainsFunc(out, func(l [2]string) bool { return l[0] == f.Case }) {
			out = append(out, [2]string{f.Case, v.Text})
		}
	}
	return out, nil
}

// refinement checks the declared refinement under the rule `Umpire.Command.deriveRefinement`
// applies, as model/go's `Refinement` does: every outcome reads as a product outcome of the same name,
// every start as a product start, and every row result is carried by a product row from the mapped
// source that reaches the mapped target with the same outcome and whose facts all appear among the
// result's facts, preferring the product action of the row's own name, or else the mapped states are
// equal and the result is a stutter.
// A refinement that names what the product sees narrows both, as Machines 6 says. It sets the
// machine's refinement rows, or why the refinement does not hold as its Rejected, and returns only an
// error evaluating the Model's functions.
func (in *Interpreter) refinement(mm, product *Machine) error {
	src, dst := mm.Table, product.Table
	where := mm.Decl.GetName() + " refines " + dst.Machine
	mapKey := func(state string) (string, error) {
		v, err := in.Call(mm.Decl.GetRefines().GetMap(), []Value{mm.states[state]}, mm.Decl.GetPosition())
		return v.Key(), err
	}
	if err := in.readsAs(mm, dst, where, mapKey); err != nil || mm.Rejected != nil {
		return err
	}
	var rows []umpire.RefinementRow
	for i, row := range src.Rows {
		from, err := mapKey(row.Source)
		if err != nil {
			return err
		}
		for j, res := range row.Results {
			to, err := mapKey(res.State)
			if err != nil {
				return err
			}
			seen, seenOutcome, err := in.seen(mm, dst, mm.Transitions[i].Steps[j])
			if err != nil {
				return err
			}
			if carrier, ok := carrierOf(dst, row, res, from, to, seen); ok {
				rows = append(rows, umpire.RefinementRow{Key: row.Key, Product: &carrier})
				continue
			}
			if mm.Rejected = noStutter(mm.Decl.GetPosition(), where, dst.Machine, row, res, from, to, seen, seenOutcome); mm.Rejected != nil {
				return nil
			}
			rows = append(rows, umpire.RefinementRow{Key: row.Key})
		}
	}
	mm.Refinement = rows
	return nil
}

// readsAs checks that every outcome of the refining machine is a product outcome of the same name,
// and that every start reads as a product start.
func (in *Interpreter) readsAs(mm *Machine, dst *umpire.Table, where string, mapKey func(string) (string, error)) error {
	src := mm.Table
	for _, o := range src.Outcomes {
		if !slices.Contains(dst.Outcomes, o) {
			mm.Rejected = errorAt(mm.Decl.GetPosition(), "%s: '%s' is an outcome of %s and no outcome of %s has that name",
				where, o, src.Machine, dst.Machine)
			return nil
		}
	}
	for _, s := range src.Starts {
		mapped, err := mapKey(s)
		if err != nil {
			return err
		}
		if !slices.Contains(dst.Starts, mapped) {
			mm.Rejected = errorAt(mm.Decl.GetPosition(), "%s: %s starts at '%s', which reads as '%s', and %s does not start there",
				where, src.Machine, s, mapped, dst.Machine)
			return nil
		}
	}
	return nil
}

// seen is what the product sees of a step: the product keys of the facts it records that the
// refinement's `visible` accepts, and whether `visible_outcomes` accepts its outcome.
func (in *Interpreter) seen(mm *Machine, dst *umpire.Table, step Value) ([]string, bool, error) {
	r := mm.Decl.GetRefines()
	sees := func(function string, v Value) (bool, error) {
		if function == "" {
			return false, nil
		}
		b, err := in.Call(function, []Value{v}, mm.Decl.GetPosition())
		if err == nil && b.Kind != BoolValue {
			return false, errorAt(mm.Decl.GetPosition(), "%s: %s is %s for %s, not a Boolean", mm.Decl.GetName(), function, b.Key(), v.Key())
		}
		return b.Bool, err
	}
	var facts []string
	for _, f := range step.Fields[2].Items {
		visible, err := sees(r.GetVisible(), f)
		if err != nil {
			return nil, false, err
		}
		if visible {
			k, ok := sameNamedKey(dst.Facts, f.Key())
			if !ok {
				k = f.Key()
			}
			facts = append(facts, k)
		}
	}
	outcome, err := sees(r.GetVisibleOutcomes(), step.Fields[0])
	return facts, outcome, err
}

// noStutter is why a result no product step carries is not a stutter either, or nil when it is one.
func noStutter(at *modelirspb.Position, where, product string, row umpire.Row, res umpire.Result, from, to string,
	seen []string, seenOutcome bool) error {
	switch {
	case from != to:
		return errorAt(at, "%s: the row '%s' steps from '%s' to '%s', which read as '%s' and '%s' in %s, "+
			"and %s has no step between them with outcome '%s', so the row is neither a step of %s nor a stutter",
			where, row.Key, row.Source, res.State, from, to, product, product, res.Outcome, product)
	case len(seen) > 0:
		return errorAt(at, "%s: the row '%s' steps from '%s' to '%s', which both read as '%s' in %s, "+
			"and records %v, which %s sees, so the row is no stutter, and no step of %s carries it",
			where, row.Key, row.Source, res.State, from, product, seen, product, product)
	case seenOutcome:
		return errorAt(at, "%s: the row '%s' steps from '%s' to '%s', which both read as '%s' in %s, "+
			"and its outcome '%s' is one %s sees, so the row is no stutter, and no step of %s carries it",
			where, row.Key, row.Source, res.State, from, product, res.Outcome, product, product)
	default:
		return nil
	}
}

func carrierOf(dst *umpire.Table, row umpire.Row, res umpire.Result, from, to string, seen []string) (string, bool) {
	var facts []string
	for _, f := range res.Facts {
		if k, ok := sameNamedKey(dst.Facts, f); ok {
			facts = append(facts, k)
		}
	}
	var carriers []string
	for _, c := range dst.RowsFrom(from) {
		if slices.ContainsFunc(c.Results, func(cr umpire.Result) bool {
			return cr.State == to && cr.Outcome == res.Outcome && allIn(cr.Facts, facts) && allIn(seen, cr.Facts)
		}) {
			carriers = append(carriers, c.Action)
		}
	}
	if preferred, ok := sameNamedKey(dst.Actions, row.Action); ok && slices.Contains(carriers, preferred) {
		return preferred, true
	}
	if len(carriers) > 0 {
		return carriers[0], true
	}
	return "", false
}

// sameNamedKey is the product key a key names by default: the same key, or the constructor it
// applies (`Umpire.Command.sameNamedKey`).
func sameNamedKey(product []string, key string) (string, bool) {
	if slices.Contains(product, key) {
		return key, true
	}
	constructor, _, _ := strings.Cut(key, "-")
	if slices.Contains(product, constructor) {
		return constructor, true
	}
	return "", false
}

func allIn(xs, ys []string) bool {
	for _, x := range xs {
		if !slices.Contains(ys, x) {
			return false
		}
	}
	return true
}
