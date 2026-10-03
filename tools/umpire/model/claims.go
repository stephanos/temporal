package model

import (
	"errors"
	"slices"
	"strings"

	umpirespb "go.temporal.io/server/api/umpire/v1"
	umpire "go.temporal.io/server/tools/umpire/model/internal/checker"
)

// binding is one interpretation of a Model as the reader's private checker reads it: a table per machine and
// composition whose results carry their steps as values, and the Model's claims declared over those
// tables' keys. A claim is declared once per table, since two declarations of one name on a table
// would share a Definition ID.
type binding struct {
	model    *umpirespb.Model
	scope    Scope
	in       *Interpreter
	machines map[string]*Machine
	// failed is why a machine has no table, which stays with that machine and what depends on it.
	// unevaluated is why Build has no refinement of a machine to compare a checked one with.
	failed      map[string]error
	unevaluated map[string]error
	actions     map[string]*umpirespb.Action
	catalogs    map[string]map[string]Value
	subjects    map[string]*subject
	refined     map[string]*refined
	properties  map[claim]*PropertyDecl
	scenarios   map[scheduled]*ScenarioDecl
	// realizing is whether a machine's table also carries what a producer of Cases reads beside its
	// rows: each state's fields and the machine's Abstraction Claims.
	realizing bool
}

// scheduled keys a Scenario by the table it runs on: a Query that reads through a refinement the
// machine's holes leave unknown runs its Scenario on the machine's rows alone.
type scheduled struct {
	table *Table
	name  string
}

// bind interprets a Model's machines within the scope's ceilings, each on its own.
func bind(m *umpirespb.Model, scope Scope) *binding {
	in := NewInterpreter(m)
	in.ceilings = scope.Ceilings
	built := in.interpret(m)
	b := &binding{model: m, scope: scope, in: in, machines: built.machines, failed: built.failed, unevaluated: built.unrefined,
		actions: map[string]*umpirespb.Action{}, catalogs: map[string]map[string]Value{}, subjects: map[string]*subject{},
		refined: map[string]*refined{}, properties: map[claim]*PropertyDecl{}, scenarios: map[scheduled]*ScenarioDecl{}}
	for _, a := range m.GetActions() {
		b.actions[a.GetId()] = a
	}
	return b
}

func (b *binding) checking() *binding {
	checked := *b
	checked.catalogs = map[string]map[string]Value{}
	checked.subjects = map[string]*subject{}
	checked.refined = map[string]*refined{}
	checked.properties = map[claim]*PropertyDecl{}
	checked.scenarios = map[scheduled]*ScenarioDecl{}
	checked.realizing = false
	return &checked
}

// subject is a machine or a composition as a claim reads it: its table, and the values its keys and
// results stand for.
type subject struct {
	name      string
	family    string
	at        *umpirespb.Position
	stateType string
	// machine is the machine, and nil for a composition and for a machine that has no table.
	machine *Machine
	// monitored is whether the machine names monitors.
	monitored bool
	table     *Table
	// err is why the subject has no table.
	err error
	// monitors watch every Query over the subject, and watchErr is why they cannot.
	monitors []*Monitor
	watched  []*watched
	watchErr error
	// unsupported is why no Query over the subject is checked, or empty.
	unsupported string
	// state is the value of a state key, step the step record of a result, and key the key of a
	// state value.
	state func(key string) (Value, error)
	step  func(res Result) (Value, error)
	key   func(state Value) (string, error)
}

// subject is the machine or composition of this name, bound once.
func (b *binding) subject(name string) *subject {
	if s, ok := b.subjects[name]; ok {
		return s
	}
	s := &subject{name: name, err: &Error{Message: "no machine or composition " + name}}
	for _, decl := range b.model.GetMachines() {
		if mm := b.machines[name]; decl.GetName() == name && mm != nil {
			s = b.machineSubject(mm)
		} else if decl.GetName() == name {
			s = &subject{name: name, family: decl.GetFamily(), at: decl.GetPosition(), stateType: decl.GetStateType(),
				monitored: len(decl.GetMonitors()) > 0, err: b.failed[name]}
		}
	}
	for _, c := range b.model.GetCompositions() {
		if c.GetName() == name {
			s = b.compositionSubject(c)
		}
	}
	b.subjects[name] = s
	return s
}

func (b *binding) machineSubject(mm *Machine) *subject {
	decl := mm.Decl
	s := &subject{name: decl.GetName(), family: decl.GetFamily(), at: decl.GetPosition(), stateType: decl.GetStateType(),
		machine: mm, monitored: len(decl.GetMonitors()) > 0}
	s.table, s.err = b.claimed(mm)
	s.state = func(key string) (Value, error) {
		v, ok := mm.State(key)
		if !ok {
			return Value{}, errorAt(s.at, "%s has no state %s", s.name, key)
		}
		return v, nil
	}
	s.key = func(state Value) (string, error) { return state.Key(), nil }
	s.step = func(res Result) (Value, error) {
		if step, ok := res.Step.(Value); ok {
			return step, nil
		}
		return b.keyedStep(s, res)
	}
	for _, mo := range mm.Monitors {
		watching, err := b.watch(s, mo)
		if err != nil {
			s.watchErr = err
			break
		}
		s.watched, s.monitors = append(s.watched, watching), append(s.monitors, watching.monitor())
	}
	return s
}

// view is a machine's table as a check reads it: the keys and rows Build gave it, each result with
// its step record, the machine's assumptions, the field that carries the state it refines, and, with
// holes, its hole rows as unknown pairs. An empty list of steps stays an absent pair, and a hole row
// is never one.
func (b *binding) view(mm *Machine, holes bool) *Table {
	return umpire.NewTable(b.spec(mm, holes))
}

// claimed is the table a machine's claims are declared on: its view with its hole rows. For a
// producer of Cases it also carries each state's fields and the machine's Abstraction Claims, which
// are read off the IR here and nowhere else; a Model they cannot be read from has no such table.
func (b *binding) claimed(mm *Machine) (*Table, error) {
	spec := b.spec(mm, true)
	if b.realizing {
		var err error
		if spec.FieldValues, err = b.fieldValues(mm); err != nil {
			return nil, err
		}
		if spec.Claims, err = b.abstractionClaims(mm); err != nil {
			return nil, err
		}
	}
	t := umpire.NewTable(spec)
	return t, t.Err()
}

func (b *binding) spec(mm *Machine, holes bool) umpire.TableSpec {
	t := mm.Table
	spec := umpire.TableSpec{Machine: t.Machine, Owner: t.Owner, Family: t.Family, States: t.States, Actions: t.Actions,
		Outcomes: t.Outcomes, Facts: t.Facts, Starts: t.Starts, Ends: t.Ends, StateFields: t.StateFields, Entity: t.Entity,
		Evidence: t.Evidence, RefinedField: mm.Decl.GetRefines().GetProduct()}
	for i, row := range t.Rows {
		row.Results = slices.Clone(row.Results)
		for j := range row.Results {
			row.Results[j].Step = mm.Transitions[i].Steps[j]
		}
		spec.Rows = append(spec.Rows, row)
	}
	for _, a := range mm.Assumptions {
		spec.Assumptions = append(spec.Assumptions, b.assumption(a))
	}
	if holes {
		for _, h := range mm.Holes {
			spec.Unknown = append(spec.Unknown, UnknownPair{Row: h.Row, Source: h.Source, Action: h.Class, Cause: h.Hole})
		}
	}
	return spec
}

func (b *binding) typeNamed(name string) *umpirespb.Type {
	for _, t := range b.model.GetTypes() {
		if t.GetName() == name {
			return t
		}
	}
	return nil
}

// fieldValues is each state's fields as atoms, in the record's field order, followed by the refined
// machine's state for a refining machine: what a Contract compares a state's fields by. A state type
// that is no record has no fields.
func (b *binding) fieldValues(mm *Machine) (map[string][]Atom, error) {
	out, t, decl := map[string][]Atom{}, mm.Table, mm.Decl
	record := b.typeNamed(decl.GetStateType()).GetRecord()
	if record == nil {
		return out, nil
	}
	for _, key := range t.States {
		state, _ := mm.State(key)
		if len(state.Fields) != len(record.GetFields()) {
			return nil, errorAt(decl.GetPosition(), "%s: state %s has %d fields, and %s declares %d", decl.GetName(), key,
				len(state.Fields), decl.GetStateType(), len(record.GetFields()))
		}
		var atoms []Atom
		for i, f := range record.GetFields() {
			atoms = append(atoms, Atom{ID: t.Family.ID("state-field", t.OwnerName(), f.GetName()), Value: state.Fields[i].Key()})
		}
		if refines := decl.GetRefines(); refines != nil {
			refined, err := b.in.Call(refines.GetMap(), []Value{state}, decl.GetPosition())
			if err != nil {
				return nil, err
			}
			atoms = append(atoms, Atom{ID: t.Family.ID("state-field", t.OwnerName(), refines.GetProduct()), Value: refined.Key()})
		}
		out[key] = atoms
	}
	return out, nil
}

// abstractionClaims is the machine's Abstraction Claims: one per example of an action it binds, in the
// order the actions' classes first appear and then in declaration order.
func (b *binding) abstractionClaims(mm *Machine) ([]Claim, error) {
	t := mm.Table
	var out []Claim
	seen := map[string]bool{}
	for _, class := range mm.Classes {
		action := class.Action
		if seen[action.GetId()] {
			continue
		}
		seen[action.GetId()] = true
		for _, ex := range action.GetExamples() {
			// Admission rejects an example anywhere but on a one-input action; a Model that was not admitted
			// is refused here rather than read past its inputs.
			if len(action.GetInputs()) != 1 || ex.GetValue() == nil {
				return nil, errorAt(action.GetPosition(), "%s gives an example that is of no class of a one-input action", action.GetId())
			}
			v := b.in.literal(ex.GetValue())
			out = append(out, Claim{Member: t.Family.ID("action", t.OwnerName(), action.GetName()+"-"+v.Key()),
				Action: string(t.Family) + ".action." + action.GetName(), Field: action.GetInputs()[0].GetName(),
				ClassName: b.spelled(v), Example: ex.GetExample()})
		}
	}
	return out, nil
}

// spelled spells a class the way an `examples:` line does: a case with fields by its name and named
// fields, `handlerError (retryable := true)`, and anything else by its key.
func (b *binding) spelled(v Value) string {
	if v.Kind != EnumValue || len(v.Fields) == 0 {
		return v.Key()
	}
	var fields []string
	for _, c := range b.typeNamed(v.Type).GetEnum().GetCases() {
		if c.GetName() != v.Case || len(c.GetFields()) != len(v.Fields) {
			continue
		}
		for i, f := range c.GetFields() {
			fields = append(fields, f.GetName()+" := "+v.Fields[i].Key())
		}
	}
	return v.Case + " (" + strings.Join(fields, ", ") + ")"
}

// Realizer is an admitted Model bound once, as Check binds it, for a producer of Cases. A find Query
// it gives is the Query Check answers: the same Property read on the same step records, the same
// Scenario, Limits and watching monitors, over the machine's table with its hole rows as unknown
// pairs. That table also carries each state's fields and the machine's Abstraction Claims. It gives
// only what the Model declares, by the key Check gives a Query's receipt, so nothing it binds is
// without one.
type Realizer struct {
	b *binding
	// prints holds each table's Behavior Fingerprint, as a checker's do.
	prints map[*Table]string
}

// NewRealizer admits a Model and binds it within a scope. A Model Validate rejects is not bound.
func NewRealizer(m *umpirespb.Model, scope Scope) (*Realizer, error) {
	if err := Validate(m); err != nil {
		return nil, err
	}
	b := bind(m, scope)
	b.realizing = true
	return &Realizer{b: b, prints: map[*Table]string{}}, nil
}

// TargetFingerprint is the Behavior Fingerprint of a table the Realizer binds, computed once for each:
// a bound table is not changed after it is bound, and every Case lowered from it names its fingerprint.
func (r *Realizer) TargetFingerprint(t *Table) string {
	if _, ok := r.prints[t]; !ok {
		r.prints[t] = t.TargetFingerprint()
	}
	return r.prints[t]
}

// Declared is one Query of the bound Model with the Property and the Scenario it names.
type Declared struct {
	Query    *umpirespb.Query
	Property *umpirespb.Property
	Scenario *umpirespb.Scenario
}

// Declared is the Query the Model declares under a key: the family and the machine or composition its
// Scenario runs on, and its name, as Check keys its receipt. A key the Model declares no Query under
// names nothing.
func (r *Realizer) Declared(key ClaimKey) (*Declared, error) {
	for _, q := range r.b.model.GetQueries() {
		if q.GetName() != key.Name {
			continue
		}
		owner := q.GetScenario().GetMachine()
		if family := r.b.subject(owner).family; family != key.Family || owner != key.Owner {
			return nil, errorAt(q.GetPosition(), "query %s runs on %s of %s, not on %s of %s", key.Name, owner, family, key.Owner, key.Family)
		}
		return &Declared{Query: q, Property: r.b.declaredProperty(q.GetProperty()), Scenario: r.b.declaredScenario(q.GetScenario())}, nil
	}
	return nil, &Error{Position: r.b.model.GetSource(), Message: "no Query " + key.Name}
}

// Find is the Query the Model declares under a key, as the generic search answers it, or why this
// reader does not answer it.
func (r *Realizer) Find(key ClaimKey) (*Query, error) {
	declared, err := r.Declared(key)
	if err != nil {
		return nil, err
	}
	bound, err := r.b.query(declared.Query)
	if err != nil {
		return nil, err
	}
	return bound.q, nil
}

// Realizations is the realizations the Model declares.
func (r *Realizer) Realizations() []*umpirespb.Realization { return r.b.model.GetRealizations() }

// ClassKey is the key of one class of an action, as the Model's tables key it.
func (r *Realizer) ClassKey(c *umpirespb.ActionClass) string { return r.b.classKey(c) }

// Machine is an interpreted machine by name, or nil for one that could not be interpreted.
func (r *Realizer) Machine(name string) *Machine { return r.b.machines[name] }

// Bound is one Query of the bound Model as a reader of recorded steps takes it: the table Check reads
// the Query on, where its Scenario starts, and its Property and the machine's monitors as functions of
// that table's steps. They are the functions Check declares to the generic search, so a step read
// through them is read as the search reads it. A Bound is not changed after it is given, and its
// functions are not safe to call from two goroutines at once.
type Bound struct {
	// Table is the machine's rows, each result with its step record, and its hole rows as unknown
	// pairs.
	Table    *Table
	Start    string
	Property BoundProperty
	Monitors []BoundMonitor
}

// BoundProperty is a Query's Property over its table's keys. About reports whether the Property is
// about the steps of an action class, and Holds reads it on one step from the state the step leaves.
type BoundProperty struct {
	Name  string
	About func(action string) bool
	Holds func(before string, step Result) (bool, error)
}

// BoundMonitor is one monitor of a Query's machine over its table's keys: Next turns its state, the
// state before a step and the step into its state after the step, and Violated says whether a state
// violates it where its verdict is read: at the end of a path with AtEnds, and otherwise after every
// step Read accepts.
type BoundMonitor struct {
	Name, Initial string
	Next          func(state, before string, step Result) (string, error)
	Violated      func(state string) (bool, error)
	AtEnds        bool
	Read          func(step Result) (bool, error)
}

// Bound is the Query the Model declares under a key, bound for a reader of recorded steps. It is given
// only for a Query Check answers on one machine's own table: a Query of a composition and one read
// through a refinement are refused, as is one Check does not answer.
func (r *Realizer) Bound(key ClaimKey) (*Bound, error) {
	declared, err := r.Declared(key)
	if err != nil {
		return nil, err
	}
	q := declared.Query
	on := r.b.subject(q.GetScenario().GetMachine())
	if on.machine == nil || q.GetThrough() || declared.Property == nil || declared.Property.GetMachine() != on.name {
		return nil, errorAt(q.GetPosition(), "query %s is not read on the steps of one machine: it runs on %s and reads a Property of %s",
			key.Name, on.name, q.GetProperty().GetMachine())
	}
	bound, err := r.b.query(q)
	if err != nil {
		return nil, err
	}
	reading, err := r.b.propertyReads(on, declared.Property)
	if err != nil {
		return nil, err
	}
	out := &Bound{Table: bound.table, Start: bound.q.Scenario.Start, Property: boundProperty(declared.Property, reading)}
	for _, w := range on.watched {
		monitor := BoundMonitor{Name: w.name, Initial: w.initial, Next: w.next, Violated: w.violated, AtEnds: w.atEnds, Read: w.after}
		if monitor.Read == nil {
			monitor.Read = func(Result) (bool, error) { return !w.atEnds, nil }
		}
		out.Monitors = append(out.Monitors, monitor)
	}
	return out, nil
}

// boundProperty is a Property's reading over its table's keys, as a reader of recorded steps takes it.
func boundProperty(p *umpirespb.Property, reading *reads) BoundProperty {
	out := BoundProperty{Name: p.GetName(), About: func(action string) bool { return reading.about == nil || reading.about(action) },
		Holds: reading.across}
	if reading.across == nil {
		out.Holds = func(_ string, step Result) (bool, error) { return reading.same(step) }
	}
	return out
}

// assumption is an assumption as a table carries it: its name, fair for the actions it names.
func (b *binding) assumption(a *umpirespb.Assumption) Assumption {
	out := Assumption{Name: a.GetName()}
	for _, id := range a.GetFair() {
		out.Fair = append(out.Fair, b.actions[id].GetName())
	}
	return out
}

// catalog is a finite type's members by key.
func (b *binding) catalog(t *umpirespb.TypeRef) (map[string]Value, error) {
	name := spell(t)
	if members, ok := b.catalogs[name]; ok {
		return members, nil
	}
	listed, err := b.in.Members(t)
	if err != nil {
		return nil, err
	}
	members := make(map[string]Value, len(listed))
	for _, v := range listed {
		members[v.Key()] = v
	}
	b.catalogs[name] = members
	return members, nil
}

// keyedStep is a machine's step record from a result's keys alone. A Query that reads a Property
// through a refinement gives the Property the refined machine's keys, a state by the map and an
// outcome and facts by name, with no step record of that machine behind them.
func (b *binding) keyedStep(s *subject, res Result) (Value, error) {
	decl := s.machine.Decl
	state, err := s.state(res.State)
	if err != nil {
		return Value{}, err
	}
	outcomes, err := b.catalog(named(decl.GetOutcomeType()))
	if err != nil {
		return Value{}, err
	}
	outcome, ok := outcomes[res.Outcome]
	if !ok {
		return Value{}, errorAt(s.at, "%s has no outcome %s", s.name, res.Outcome)
	}
	facts := Value{Kind: ListValue}
	for _, key := range res.Facts {
		catalog, err := b.catalog(named(decl.GetFactType()))
		if err != nil {
			return Value{}, err
		}
		fact, ok := catalog[key]
		if !ok {
			return Value{}, errorAt(s.at, "%s has no fact %s", s.name, key)
		}
		facts.Items = append(facts.Items, fact)
	}
	return Value{Kind: RecordValue, Type: StepType,
		Fields: []Value{outcome, state, facts, {Kind: TextValue, Text: res.Because}}}, nil
}

// decide calls a function of a claim that answers yes or no. A value that is no Boolean is an error
// of the Model at the claim, never false: a claim that cannot be read neither holds nor fails.
func (b *binding) decide(function string, args []Value, at *umpirespb.Position, owner, relation, read string) (bool, error) {
	v, err := b.in.Call(function, args, at)
	if err != nil {
		return false, err
	}
	if v.Kind != BoolValue {
		return false, errorAt(at, "%s: %s is %s %s %s, not a Boolean", owner, function, v.Key(), relation, read)
	}
	return v.Bool, nil
}

// Unknown reports whether an error of a bound function leaves what it read unknown, neither held nor
// failed: the function reached a hole. Any other error is the declaration's.
// It says which errors of a claim's function leave a step unknown rather than fail the search.
func Unknown(err error) bool {
	var hole *Hole
	return errors.As(err, &hole)
}

// unsupportedError is a declaration this reader does not check, and why.
type unsupportedError struct{ why string }

func (e *unsupportedError) Error() string { return e.why }

// watched is one monitor of a machine as its functions read steps over the table's keys: what the
// checker's declaration and a reader of recorded steps both use.
type watched struct {
	name, initial string
	next          func(state, before string, res Result) (string, error)
	violated      func(state string) (bool, error)
	// atEnds says the verdict is read at the end of a path. Otherwise it is read after every step, or,
	// with after, after the steps after accepts.
	atEnds bool
	after  func(res Result) (bool, error)
}

// watch binds one of a machine's monitors over its table's keys. Its state after a step must be one of
// its state type's members.
func (b *binding) watch(s *subject, mo *umpirespb.Monitor) (*watched, error) {
	at, name := mo.GetPosition(), "monitor "+mo.GetName()
	states, err := b.catalog(mo.GetState())
	if err != nil {
		return nil, err
	}
	if mo.GetInitial() == nil {
		return nil, errorAt(at, "%s: it names no initial state", name)
	}
	initial, err := b.in.Eval(mo.GetInitial())
	if err != nil {
		return nil, err
	}
	if known, ok := states[initial.Key()]; !ok || !known.equal(initial) {
		return nil, errorAt(at, "%s: its initial state %s is outside its states", name, initial.Key())
	}
	w := &watched{name: mo.GetName(), initial: initial.Key()}
	w.next = func(state, before string, res Result) (string, error) {
		source, err := s.state(before)
		if err != nil {
			return "", err
		}
		step, err := s.step(res)
		if err != nil {
			return "", err
		}
		v, err := b.in.Call(mo.GetNext(), []Value{states[state], source, step}, at)
		if err != nil {
			return "", err
		}
		if known, ok := states[v.Key()]; !ok || !known.equal(v) {
			return "", errorAt(at, "%s: %s is %s after the step into %s, which is outside its states", name, mo.GetNext(), v.Key(), res.State)
		}
		return v.Key(), nil
	}
	w.violated = func(state string) (bool, error) {
		return b.decide(mo.GetViolated(), []Value{states[state]}, at, name, "at", state)
	}
	switch e := mo.GetEvaluate().(type) {
	case *umpirespb.Monitor_EveryStep:
	case *umpirespb.Monitor_AtEnds:
		w.atEnds = true
	case *umpirespb.Monitor_After:
		w.after = func(res Result) (bool, error) {
			step, err := s.step(res)
			if err != nil {
				return false, err
			}
			return b.decide(e.After, []Value{step}, at, name, "for the step into", res.State)
		}
	default:
		return nil, errorAt(at, "%s has no evaluation point", name)
	}
	return w, nil
}

// monitor declares a bound monitor to the checker.
func (w *watched) monitor() *Monitor {
	read := umpire.EveryStep()
	switch {
	case w.atEnds:
		read = umpire.AtEnds()
	case w.after != nil:
		read = umpire.AfterKey(w.after)
	default:
	}
	return umpire.KeyMonitor(w.name, w.initial, w.next, w.violated, read)
}

// reads is a Property as its function reads steps over its table's keys: what the checker's
// declaration and a reader of recorded steps both use. about is the classes a same-step Property is
// about, nil for every class; across is set for a transition Property, and same for any other.
type reads struct {
	about  func(action string) bool
	label  string
	same   func(res Result) (bool, error)
	across func(before string, res Result) (bool, error)
}

// propertyReads binds a Property's function over the keys of the table it is declared on.
func (b *binding) propertyReads(s *subject, p *umpirespb.Property) (*reads, error) {
	owner, at := p.GetMachine()+"."+p.GetName(), p.GetPosition()
	if p.GetTransition() && p.GetWhen() != nil {
		return nil, &unsupportedError{owner + " is a transition Property about some steps only, and a transition Property is about every step"}
	}
	if p.GetTransition() {
		return &reads{across: func(before string, res Result) (bool, error) {
			source, err := s.state(before)
			if err != nil {
				return false, err
			}
			step, err := s.step(res)
			if err != nil {
				return false, err
			}
			return b.decide(p.GetHolds(), []Value{source, step}, at, owner, "for the step into", res.State)
		}}, nil
	}
	r := &reads{same: func(res Result) (bool, error) {
		step, err := s.step(res)
		if err != nil {
			return false, err
		}
		return b.decide(p.GetHolds(), []Value{step}, at, owner, "for the step into", res.State)
	}}
	r.about, r.label = b.when(p)
	return r, nil
}

// property declares a Property on its machine's or composition's table, once.
func (b *binding) property(p *umpirespb.Property) (*PropertyDecl, error) {
	key := claim{p.GetMachine(), p.GetName()}
	if decl, ok := b.properties[key]; ok {
		return decl, nil
	}
	s := b.subject(p.GetMachine())
	if s.err != nil {
		return nil, s.err
	}
	r, err := b.propertyReads(s, p)
	if err != nil {
		return nil, err
	}
	var decl *PropertyDecl
	if r.across != nil {
		decl = umpire.KeyTransitionProperty(s.table, p.GetName(), r.across)
	} else {
		decl = umpire.KeyProperty(s.table, p.GetName(), r.about, r.label, r.same)
	}
	b.properties[key] = decl
	return decl, nil
}

// when is the steps a same-step Property is about, and how a diagnostic names them. It reads the class
// keys of the table the Property is declared on, so a composition's Property is about composed
// classes: admission has checked that it names some (validator.selectors).
func (b *binding) when(p *umpirespb.Property) (func(action string) bool, string) {
	switch w := p.GetWhen().(type) {
	case *umpirespb.Property_WhenClass:
		key := b.classKey(w.WhenClass)
		return func(action string) bool { return action == key }, key
	case *umpirespb.Property_WhenAction:
		// A class key is its action's name and then its inputs, joined by "-". Reading the name off the
		// key, as umpire's WhenAction does, also finds the action's classes on a machine that refines
		// the Property's, which the Property is read on through the refinement.
		return func(action string) bool { return actionOf(action) == w.WhenAction }, w.WhenAction
	default:
		return nil, ""
	}
}

// actionOf is the action a class key is of: the key before its inputs.
func actionOf(key string) string {
	name, _, _ := strings.Cut(key, "-")
	return name
}

// classKey is the key of one class of an action: its name, and the key of each input.
func (b *binding) classKey(c *umpirespb.ActionClass) string { return classKey(b.in, b.actions, c) }

func classKey(in *Interpreter, actions map[string]*umpirespb.Action, c *umpirespb.ActionClass) string {
	parts := []string{actions[c.GetAction()].GetName()}
	for _, x := range c.GetInputs() {
		parts = append(parts, in.literal(x).Key())
	}
	return strings.Join(parts, "-")
}

// scenario declares a Scenario of a subject on the table it runs over, once.
func (b *binding) scenario(sc *umpirespb.Scenario, on *subject, table *Table) (*ScenarioDecl, error) {
	key := scheduled{table, sc.GetName()}
	if decl, ok := b.scenarios[key]; ok {
		return decl, nil
	}
	owner := sc.GetMachine() + "." + sc.GetName()
	if sc.GetStart() == nil {
		return nil, errorAt(sc.GetPosition(), "%s names no start", owner)
	}
	state, err := b.in.Eval(sc.GetStart())
	if err != nil {
		return nil, err
	}
	start, err := on.key(state)
	if err != nil {
		return nil, err
	}
	var decl *ScenarioDecl
	if sc.GetFree() {
		decl = umpire.KeyFreeScenario(table, sc.GetName(), start)
	} else {
		actions := slices.Clone(sc.GetKeys())
		for _, c := range sc.GetActions() {
			actions = append(actions, b.classKey(c))
		}
		decl = umpire.KeyScenario(table, sc.GetName(), start, actions...)
	}
	b.scenarios[key] = decl
	return decl, nil
}

// refined is a machine's declared refinement as RefineTables checks it over the check tables.
type refined struct {
	// ref is the refinement a Query reads through, over source, and nil when the rows do not refine.
	ref    *umpire.Refinement
	source *Table
	// err is why the refinement is not established: rejected, or left unknown by a reachable hole.
	err error
	// incomplete is set when only reachable holes leave the refinement unknown: every row refines, so
	// ref reads the machine's rows, over a table without its holes.
	incomplete *RefinementError
	// unread is the holes that left what the refinement names visible unknown.
	unread []*Hole
}

// unrefined is a refinement the generic check does not accept, with the refining machine it is of,
// whose table, holes and assumptions a receipt reads.
type unrefined struct {
	source   *subject
	rejected *RefinementError
	// unread is the holes that left what the refinement names visible unknown for some fact or
	// outcome, which the rejection does not rest on.
	unread []*Hole
}

func (e *unrefined) Error() string { return e.rejected.Error() }

func (e *unrefined) Unwrap() error { return e.rejected }

// refinement checks a refining machine's declared refinement, once.
func (b *binding) refinement(s *subject) *refined {
	if r, ok := b.refined[s.name]; ok {
		return r
	}
	r := &refined{source: s.table}
	b.refined[s.name] = r
	product := b.subject(s.machine.Decl.GetRefines().GetProduct())
	if product.err != nil {
		r.err = product.err
		return r
	}
	spec, failed := b.reading(s, product)
	r.ref, r.unread, r.err = b.refines(s, product, spec, failed)
	var incomplete *RefinementError
	if errors.As(r.err, &incomplete) && incomplete.Kind == umpire.RefinementIncomplete {
		rows := b.view(s.machine, false)
		if ref, err := umpire.RefineTables(rows, product.table, spec); err == nil {
			r.ref, r.source, r.incomplete = ref, rows, incomplete
		}
	}
	return r
}

// unseen is a refinement every row of which refines with a fact or an outcome of unknown visibility
// read as unseen: not shown to hold, by the holes that left the visibility unknown.
type unseen struct {
	source *subject
	holes  []*Hole
}

func (e *unseen) Error() string { return errors.Join(e.Unwrap()...).Error() }

func (e *unseen) Unwrap() []error { return holeErrors(e.holes) }

// refines runs the generic refinement check of one machine's table by another's. It returns the
// refinement, the holes that left a visible fact or outcome unknown, and why the refinement is not
// established.
//
// The check reads a fact or an outcome whose visibility is a hole as unseen. What the refined machine
// sees only narrows the steps that carry a row and the rows that are stutters, so a rejection found
// that way holds whatever the hole hides, and stands. A refinement accepted that way is not shown to
// hold: it is returned for a Query to read through, with the hole as why it is unknown. A visible
// function that fails any other way is an error of the Model, whatever the rows say.
func (b *binding) refines(s, product *subject, spec umpire.RefinementSpec, unread func() ([]*Hole, error)) (*umpire.Refinement, []*Hole, error) {
	ref, err := umpire.RefineTables(s.table, product.table, spec)
	holes, malformed := unread()
	var rejected *RefinementError
	switch {
	case malformed != nil:
		return nil, nil, malformed
	case errors.As(err, &rejected):
		return nil, holes, &unrefined{source: s, rejected: rejected, unread: holes}
	case err != nil && len(holes) > 0 && Unknown(err):
		// The check stopped at another hole, as one in the map: the holes read before it are reported
		// with it.
		return nil, holes, errors.Join(append(holeErrors(holes), err)...)
	case err != nil:
		return nil, holes, err
	case len(holes) > 0:
		return ref, holes, &unseen{source: s, holes: holes}
	default:
		return ref, nil, nil
	}
}

// reading is how a refining machine's table reads as the table of the machine it refines: its map,
// and what the refinement names visible. A RefinementSpec's visible predicates return no error, so
// what the Model's functions fail with across the whole check is kept: every hole they reach, and
// the first error that is none, which is an error of the Model whatever was read before it.
func (b *binding) reading(s, product *subject) (umpire.RefinementSpec, func() ([]*Hole, error)) {
	r, at := s.machine.Decl.GetRefines(), s.at
	var unread unknowns
	var malformed error
	sees := func(function, typ string) func(string) bool {
		if function == "" {
			return nil
		}
		return func(key string) bool {
			members, err := b.catalog(named(typ))
			seen := false
			if err == nil {
				seen, err = b.decide(function, []Value{members[key]}, at, s.name, "for", key)
			}
			if err = unread.note(err); malformed == nil {
				malformed = err
			}
			return seen
		}
	}
	spec := umpire.RefinementSpec{
		SeesFact:    sees(r.GetVisible(), s.machine.Decl.GetFactType()),
		SeesOutcome: sees(r.GetVisibleOutcomes(), s.machine.Decl.GetOutcomeType()),
		MapState: func(key string) (string, error) {
			state, err := s.state(key)
			if err != nil {
				return "", err
			}
			v, err := b.in.Call(r.GetMap(), []Value{state}, at)
			if err != nil {
				return "", err
			}
			if known, err := product.state(v.Key()); err != nil || !known.equal(v) {
				return "", errorAt(at, "%s: %s reads %s as %s, which is no state of %s", s.name, r.GetMap(), key, v.Key(), product.name)
			}
			return v.Key(), nil
		},
	}
	return spec, func() ([]*Hole, error) { return unread.holes, malformed }
}

// boundQuery is a Query of the IR as the generic search answers it.
type boundQuery struct {
	q     *Query
	table *Table
	// through is the refinement the Query reads its Property through, or nil.
	through *refined
}

// query declares a Query over its Scenario's table, watched by every monitor of the machine.
func (b *binding) query(q *umpirespb.Query) (*boundQuery, error) {
	on := b.subject(q.GetScenario().GetMachine())
	find := q.GetForm() == umpirespb.Query_FORM_FIND
	switch {
	case on.unsupported != "":
		return nil, &unsupportedError{on.unsupported}
	case q.GetThrough() && find:
		return nil, &unsupportedError{"a find through a refinement is not supported: only a verify reads a Property through one"}
	case on.err != nil:
		return nil, on.err
	case on.watchErr != nil:
		return nil, on.watchErr
	default:
	}
	out := &boundQuery{table: on.table}
	if q.GetThrough() {
		if on.machine == nil || on.machine.Decl.GetRefines() == nil {
			return nil, errorAt(q.GetPosition(), "query %s reads through a refinement, and %s declares none", q.GetName(), on.name)
		}
		out.through = b.refinement(on)
		if out.through.ref == nil {
			return nil, out.through.err
		}
		out.table = out.through.source
	}
	p, err := b.property(b.declaredProperty(q.GetProperty()))
	if err != nil {
		return nil, err
	}
	s, err := b.scenario(b.declaredScenario(q.GetScenario()), on, out.table)
	if err != nil {
		return nil, err
	}
	limits := b.limits(q)
	switch {
	case out.through != nil:
		out.q = umpire.KeyVerifyRefined(q.GetName(), p, s, out.through.ref, limits)
	case find:
		out.q = umpire.KeyFind(q.GetName(), p, s, limits)
	default:
		out.q = umpire.KeyVerify(q.GetName(), p, s, limits)
	}
	out.q.Unknown = Unknown
	out.q.Watch(on.monitors...)
	return out, nil
}

// limits is the Limits a Query runs within: its own, with the scope's search limit where that is less.
func (b *binding) limits(q *umpirespb.Query) Limits {
	l := q.GetLimits()
	limits := Limits{Name: l.GetName(), Steps: int(l.GetSteps()), Actions: int(l.GetActions()), Search: int(l.GetSearch())}
	if b.scope.QuerySearch > 0 {
		limits.Search = min(limits.Search, b.scope.QuerySearch)
	}
	return limits
}

func (b *binding) declaredProperty(ref *umpirespb.ClaimRef) *umpirespb.Property {
	for _, p := range b.model.GetProperties() {
		if p.GetMachine() == ref.GetMachine() && p.GetName() == ref.GetName() {
			return p
		}
	}
	return nil
}

func (b *binding) declaredScenario(ref *umpirespb.ClaimRef) *umpirespb.Scenario {
	for _, s := range b.model.GetScenarios() {
		if s.GetMachine() == ref.GetMachine() && s.GetName() == ref.GetName() {
			return s
		}
	}
	return nil
}

// progress declares a progress claim of a machine over its table's state keys.
func (b *binding) progress(p *umpirespb.Progress) (*umpire.Progress, *subject, error) {
	s := b.subject(p.GetMachine())
	if s.err != nil {
		return nil, s, s.err
	}
	owner, at := p.GetMachine()+"."+p.GetName(), p.GetPosition()
	accepts := func(function string) func(string) (bool, error) {
		return func(key string) (bool, error) {
			state, err := s.state(key)
			if err != nil {
				return false, err
			}
			return b.decide(function, []Value{state}, at, owner, "at", key)
		}
	}
	var assumptions []Assumption
	for _, id := range p.GetAssumptions() {
		for _, a := range b.model.GetAssumptions() {
			if a.GetId() == id {
				assumptions = append(assumptions, b.assumption(a))
			}
		}
	}
	claim := umpire.KeyProgressFunc(p.GetName(), accepts(p.GetFrom()), accepts(p.GetTo()), int(p.GetWithin()), assumptions...)
	claim.Unknown = Unknown
	return claim, s, nil
}
