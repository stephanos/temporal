package model

import (
	"cmp"
	"errors"
	"slices"
	"strings"

	umpirespb "go.temporal.io/server/api/umpire/v1"
	umpire "go.temporal.io/server/tools/umpire/model/internal/checker"
)

// Members lists every value of a finite type in catalog order: an enum's cases in declaration order,
// each case with fields once per assignment of them; a record as the product of its fields; with the
// last field varying fastest in both, which is the order the Lean `Finite` derivation produces.
// A channel's contents are listed as its Channels section says. A catalog larger than the Members
// ceiling is refused from its count, before any of it is listed.
func (in *Interpreter) Members(t *umpirespb.TypeRef) ([]Value, error) {
	n, err := in.Size(t)
	if err != nil {
		return nil, err
	}
	if err := in.Within("members", in.ceilings.Members, n); err != nil {
		return nil, err
	}
	switch r := t.GetRef().(type) {
	case *umpirespb.TypeRef_Bool:
		return []Value{{Kind: BoolValue, Bool: false}, {Kind: BoolValue, Bool: true}}, nil
	case *umpirespb.TypeRef_IntRange:
		var out []Value
		for i := r.IntRange.GetLow(); i <= r.IntRange.GetHigh(); i++ {
			out = append(out, Value{Kind: IntValue, Int: i})
			// Past the high end of a range ending at math.MaxInt64, i++ would wrap around.
			if i == r.IntRange.GetHigh() {
				break
			}
		}
		return out, nil
	case *umpirespb.TypeRef_Named:
		out, err := in.declaredMembers(r.Named)
		if err != nil {
			return nil, err
		}
		return out, distinctKeys(in.types[r.Named].GetPosition(), r.Named, out)
	case *umpirespb.TypeRef_Channel:
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
func distinctKeys(at *umpirespb.Position, catalog string, values []Value) error {
	seen := make(map[string]Value, len(values))
	for _, v := range values {
		if earlier, ok := seen[v.Key()]; ok {
			return ErrorAt(at, "the catalog of %s holds %s and %s, which share the key %q", catalog, earlier.spelled(), v.spelled(), v.Key())
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
	case *umpirespb.Type_Enum:
		var out []Value
		for _, c := range s.Enum.GetCases() {
			assignments, err := in.Product(c.GetFields())
			if err != nil {
				return nil, err
			}
			for _, fields := range assignments {
				out = append(out, Value{Kind: EnumValue, Type: name, Case: c.GetName(), Fields: fields})
			}
		}
		return out, nil
	case *umpirespb.Type_Record:
		assignments, err := in.Product(s.Record.GetFields())
		if err != nil {
			return nil, err
		}
		out := make([]Value, len(assignments))
		for i, fields := range assignments {
			out[i] = Value{Kind: RecordValue, Type: name, Fields: fields}
		}
		return out, nil
	default:
		return nil, ErrorAt(decl.GetPosition(), "type %s has no shape", name)
	}
}

// product lists every assignment of fields, the last varying fastest, once their count is within the
// Members ceiling.
func (in *Interpreter) Product(fields []*umpirespb.Field) ([][]Value, error) {
	n, err := in.SizeOfProduct(fields)
	if err != nil {
		return nil, err
	}
	if err := in.Within("members", in.ceilings.Members, n); err != nil {
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

func Named(name string) *umpirespb.TypeRef {
	return &umpirespb.TypeRef{Ref: &umpirespb.TypeRef_Named{Named: name}}
}

// Class is what the checks that read a machine's classes are given:
// class is one class of an action: the action and one assignment of its inputs.
type Class struct {
	Key    string
	Action *umpirespb.Action
	Inputs []Value
	step   string
	at     *umpirespb.Position
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

// Machine is one machine of the IR, interpreted: its table, the same rows as values, its hole rows,
// and the monitors and assumptions it names. Nothing here applies a monitor or an assumption, or
// checks a refinement: a check that reads the table alone answers a question about the rows, not
// about the machine's declarations. Check reads a refining machine's refinement (RefineTables).
type Machine struct {
	Decl        *umpirespb.Machine
	Table       *Table
	states      map[string]Value
	Classes     []Class
	Transitions []Transition
	Holes       []HoleRow
	Monitors    []*umpirespb.Monitor
	Assumptions []*umpirespb.Assumption
	Work        Work
}

// State is the value of a state key.
func (m *Machine) State(key string) (Value, bool) {
	v, ok := m.states[key]
	return v, ok
}

// reachableHoles is the hole rows whose state the table reaches.
func (m *Machine) ReachableHoles() []HoleRow {
	var out []HoleRow
	for _, h := range m.Holes {
		if slices.Contains(m.Table.Reachable, h.Source) {
			out = append(out, h)
		}
	}
	return out
}

// Build interprets every machine of a Model, within defaultCeilings.
func Build(m *umpirespb.Model) (map[string]*Machine, error) {
	return NewInterpreter(m).Build(m)
}

func (in *Interpreter) Build(m *umpirespb.Model) (map[string]*Machine, error) {
	out := in.Interpret(m)
	for _, decl := range m.GetMachines() {
		if err := out.Failed[decl.GetName()]; err != nil {
			return nil, err
		}
	}
	return out.Machines, nil
}

// interpretation is a Model's machines interpreted one by one, so that what one machine's
// declarations leave unread does not take the others with it: failed is why a machine has no table.
type Interpretation struct {
	Machines map[string]*Machine
	Failed   map[string]error
}

func (in *Interpreter) Interpret(m *umpirespb.Model) Interpretation {
	actions := map[string]*umpirespb.Action{}
	for _, a := range m.GetActions() {
		actions[a.GetId()] = a
	}
	out := Interpretation{Machines: map[string]*Machine{}, Failed: map[string]error{}}
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
			out.Failed[decl.GetName()] = err
			continue
		}
		out.Machines[decl.GetName()] = mm
	}
	return out
}

// declarations resolves the monitors and assumptions a machine names.
func declarations(m *umpirespb.Model, mm *Machine) error {
	for _, id := range mm.Decl.GetMonitors() {
		i := slices.IndexFunc(m.GetMonitors(), func(d *umpirespb.Monitor) bool { return d.GetId() == id })
		if i < 0 {
			return ErrorAt(mm.Decl.GetPosition(), "%s: no monitor %s", mm.Decl.GetName(), id)
		}
		mm.Monitors = append(mm.Monitors, m.GetMonitors()[i])
	}
	for _, id := range mm.Decl.GetAssumes() {
		i := slices.IndexFunc(m.GetAssumptions(), func(d *umpirespb.Assumption) bool { return d.GetId() == id })
		if i < 0 {
			return ErrorAt(mm.Decl.GetPosition(), "%s: no assumption %s", mm.Decl.GetName(), id)
		}
		mm.Assumptions = append(mm.Assumptions, m.GetAssumptions()[i])
	}
	return nil
}

func (in *Interpreter) machine(decl *umpirespb.Machine, actions map[string]*umpirespb.Action) (*Machine, error) {
	if err := in.Preflight(decl, actions); err != nil {
		return nil, err
	}
	states, err := in.Members(Named(decl.GetStateType()))
	if err != nil {
		return nil, err
	}
	outcomes, err := in.Members(Named(decl.GetOutcomeType()))
	if err != nil {
		return nil, err
	}
	var facts []Value
	if decl.GetFactType() != "" {
		if facts, err = in.Members(Named(decl.GetFactType())); err != nil {
			return nil, err
		}
	}
	classes, err := in.Classes(decl, actions)
	if err != nil {
		return nil, err
	}
	spec := umpire.TableSpec{Machine: decl.GetName(), Owner: decl.GetName(), Family: Family(decl.GetFamily()),
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
	var unread Unknowns
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
	if err := unread.Err(); err != nil {
		return nil, err
	}
	mm.Table = umpire.NewTable(spec)
	return mm, nil
}

// preflight counts a machine's states, its classes of every bound action together, and their pairs,
// and refuses work past a ceiling before any of it is listed.
func (in *Interpreter) Preflight(decl *umpirespb.Machine, actions map[string]*umpirespb.Action) error {
	states, err := in.Size(Named(decl.GetStateType()))
	if err != nil {
		return err
	}
	if err := in.Within("members", in.ceilings.Members, states); err != nil {
		return err
	}
	classes, err := in.ClassCount(decl, actions)
	if err != nil {
		return err
	}
	return in.Within("evaluations", in.ceilings.Evaluations, states.Times(classes))
}

// classCount counts a machine's classes of every bound action together, refusing them past the
// Members ceiling, so no caller lists them first.
func (in *Interpreter) ClassCount(decl *umpirespb.Machine, actions map[string]*umpirespb.Action) (Count, error) {
	classes, err := in.BoundClasses(decl, actions)
	if err != nil {
		return Count{}, err
	}
	return classes, in.Within("classes", in.ceilings.Members, classes)
}

// boundClasses counts a machine's classes of every bound action together, past any ceiling.
func (in *Interpreter) BoundClasses(decl *umpirespb.Machine, actions map[string]*umpirespb.Action) (Count, error) {
	var classes Count
	for _, b := range decl.GetSteps() {
		a, ok := actions[b.GetAction()]
		if !ok {
			return Count{}, ErrorAt(b.GetPosition(), "no action %s", b.GetAction())
		}
		n, err := in.SizeOfProduct(InputFields(a))
		if err != nil {
			return Count{}, err
		}
		classes = classes.Plus(n)
	}
	return classes, nil
}

func InputFields(a *umpirespb.Action) []*umpirespb.Field {
	fields := make([]*umpirespb.Field, len(a.GetInputs()))
	for i, p := range a.GetInputs() {
		fields[i] = &umpirespb.Field{Name: p.GetName(), Type: p.GetType()}
	}
	return fields
}

// classes lists every class of every bound action, sorted by key, and rejects a class two steps bind.
func (in *Interpreter) Classes(decl *umpirespb.Machine, actions map[string]*umpirespb.Action) ([]Class, error) {
	if _, err := in.ClassCount(decl, actions); err != nil {
		return nil, err
	}
	var out []Class
	for _, b := range decl.GetSteps() {
		a, ok := actions[b.GetAction()]
		if !ok {
			return nil, ErrorAt(b.GetPosition(), "no action %s", b.GetAction())
		}
		assignments, err := in.Product(InputFields(a))
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
			return nil, ErrorAt(y.at, "%s: two steps bind the action class %q", decl.GetName(), y.Key)
		default:
			return nil, ErrorAt(y.at, "%s: the classes %s and %s share the key %q", decl.GetName(), x.spelled(), y.spelled(), y.Key)
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
func RowKeys(decl *umpirespb.Machine, states []Value, classes []Class) error {
	seen := make(map[string][2]string, len(states)*len(classes))
	for _, s := range states {
		for _, c := range classes {
			key := s.Key() + "-" + c.Key
			if earlier, ok := seen[key]; ok {
				return ErrorAt(decl.GetPosition(), "%s: the state %s with the class %s, and the state %s with the class %s, share the row key %q",
					decl.GetName(), earlier[0], earlier[1], s.Key(), c.Key, key)
			}
			seen[key] = [2]string{s.Key(), c.Key}
		}
	}
	return nil
}

// rows evaluates every step function once per state and class, states-major, keeping the enabled
// pairs as rows and rejecting a result outside the state domain and two results of one row with the
// same name (model/SEMANTICS.md, Named choices).
// It also keeps each row's transition, and the pairs whose value is a hole as hole rows; a channel's
// delivery and loss are evaluated as transfers.
func (in *Interpreter) rows(mm *Machine, states []Value, spec *umpire.TableSpec) error {
	decl := mm.Decl
	if err := RowKeys(decl, states, mm.Classes); err != nil {
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
			row := Row{Key: key, Source: s.Key(), Action: c.Key}
			named := map[string]bool{}
			for _, step := range steps {
				res, err := in.result(mm, c, key, step)
				if err != nil {
					return err
				}
				if res.Choice != "" && named[res.Choice] {
					return ErrorAt(c.at, "%s: row %s has two results named %s", decl.GetName(), key, res.Choice)
				}
				named[res.Choice] = true
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
func (in *Interpreter) result(m *Machine, c Class, row string, step Value) (Result, error) {
	decl := m.Decl
	if outcome := step.Fields[0]; !in.Conforms(outcome, Named(decl.GetOutcomeType())) {
		return Result{}, ErrorAt(c.at, "%s: row %s has outcome %s, which is no %s", decl.GetName(), row, outcome.Key(), decl.GetOutcomeType())
	}
	for _, f := range step.Fields[2].Items {
		if decl.GetFactType() == "" || !in.Conforms(f, Named(decl.GetFactType())) {
			return Result{}, ErrorAt(c.at, "%s: row %s records %s, which is no %s", decl.GetName(), row, f.Key(), cmp.Or(decl.GetFactType(), "fact of it"))
		}
	}
	next := step.Fields[1].Key()
	if v, ok := m.states[next]; !ok || !v.Equal(step.Fields[1]) {
		return Result{}, ErrorAt(c.at, "%s: row %s lands in %s, which is outside the state domain", m.Decl.GetName(), row, next)
	}
	res := Result{Outcome: step.Fields[0].Key(), State: next, Facts: []string{}, Because: step.Fields[3].Text, Choice: step.Choice}
	for _, f := range step.Fields[2].Items {
		res.Facts = append(res.Facts, f.Key())
	}
	return res, nil
}

func (in *Interpreter) steps(decl *umpirespb.Machine, s Value, c Class) ([]Value, error) {
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
func (in *Interpreter) stepList(decl *umpirespb.Machine, c Class, v Value) ([]Value, error) {
	if v.Kind != ListValue {
		return nil, ErrorAt(c.at, "%s returns %s, not a list of steps", c.step, v.Key())
	}
	for _, step := range v.Items {
		switch {
		case step.Kind != RecordValue || step.Type != StepType:
			return nil, ErrorAt(c.at, "%s returns %s, not a list of steps", c.step, v.Key())
		case len(step.Fields) != len(StepFields):
			return nil, ErrorAt(c.at, "%s returns a step of %d fields, not %d", c.step, len(step.Fields), len(StepFields))
		case step.Fields[1].Type != decl.GetStateType():
			return nil, ErrorAt(c.at, "%s returns a step to %s, which is no %s", c.step, step.Fields[1].Key(), decl.GetStateType())
		case step.Fields[2].Kind != ListValue || step.Fields[3].Kind != TextValue:
			return nil, ErrorAt(c.at, "%s returns a step whose facts are no list or whose explanation is no string", c.step)
		default:
		}
	}
	return v.Items, nil
}

// unknowns collects the holes a declaration reaches as it is read at one value after another, each
// once. The reading goes on past a hole, so that an error of the Model at a later value is not lost
// behind it, and every hole it reaches is reported.
type Unknowns struct{ Holes []*Hole }

// note keeps a hole, for which it is nil, and is any other error itself.
func (u *Unknowns) Note(err error) error {
	var hole *Hole
	if !errors.As(err, &hole) {
		return err
	}
	if !slices.ContainsFunc(u.Holes, func(h *Hole) bool { return *h == *hole }) {
		u.Holes = append(u.Holes, hole)
	}
	return nil
}

// err is the holes reached, as why the declaration is not read, or nil when it reached none.
func (u *Unknowns) Err() error {
	errs := HoleErrors(u.Holes)
	if len(errs) == 1 {
		return errs[0]
	}
	return errors.Join(errs...)
}

func HoleErrors(holes []*Hole) []error {
	errs := make([]error, len(holes))
	for i, h := range holes {
		errs[i] = h
	}
	return errs
}

// startsAndEnds reads a machine's starts and the states it may end in. A hole either reaches is noted
// in unread and read past, and is no error here: the caller has no table for a machine with one.
func (in *Interpreter) startsAndEnds(decl *umpirespb.Machine, states []Value, domain map[string]Value, unread *Unknowns) (
	starts, ends []string, err error) {
	for _, x := range decl.GetStarts() {
		v, err := in.Eval(x)
		if err != nil {
			if err = unread.Note(err); err != nil {
				return nil, nil, err
			}
			continue
		}
		if _, ok := domain[v.Key()]; !ok {
			return nil, nil, ErrorAt(x.GetPosition(), "start %s is outside the state domain", v.Key())
		}
		starts = append(starts, v.Key())
	}
	if len(starts) == 0 && len(unread.Holes) == 0 {
		return nil, nil, ErrorAt(decl.GetPosition(), "%s declares no start", decl.GetName())
	}
	if decl.GetEnds() == nil {
		return starts, nil, nil
	}
	end, evalErr := in.Eval(decl.GetEnds())
	if evalErr != nil {
		return starts, nil, unread.Note(evalErr)
	}
	for _, s := range states {
		v, err := in.Apply(end, []Value{s})
		if err != nil {
			if err = unread.Note(err); err != nil {
				return nil, nil, err
			}
			continue
		}
		if v.Kind != BoolValue {
			return nil, nil, ErrorAt(decl.GetEnds().GetPosition(), "%s: ends is %s at %s, not a Boolean", decl.GetName(), v.Key(), s.Key())
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
func (in *Interpreter) evidence(decl *umpirespb.Machine, facts []Value, unread *Unknowns) ([][2]string, error) {
	if decl.GetEvidence() == "" {
		return nil, nil
	}
	var out [][2]string
	for _, f := range facts {
		v, err := in.Call(decl.GetEvidence(), []Value{f}, decl.GetPosition())
		if err != nil {
			if err = unread.Note(err); err != nil {
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
