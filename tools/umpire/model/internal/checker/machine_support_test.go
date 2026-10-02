package checker

// The *_support_test.go files hold the typed declaration layer: tables built from Go step functions
// and claims over typed steps. No reader builds a table this way, so it is no part of the package;
// the tests of the search, refinement, composition, monitor and progress checks build their
// fixtures with it.

import (
	"reflect"
	"slices"
	"sync"
)

// Step is one result of an action: the outcome, the next state, and the facts it records. Because
// is an optional explanation a generated table view shows beside the row; it is not part of the
// Model's behavior and no fingerprint reads it.
type Step[S, O, F any] struct {
	Outcome O
	State   S
	Facts   []F
	Because string
}

type binding[S, O, F any] struct {
	decl *ActionDecl
	step func(S, Class) []Step[S, O, F]
}

// Machine is a transition relation over a finite state type S with outcomes O and facts F: one
// step function per action, the states it starts in, the states it may end in, and the evidence
// that confirms each fact. It is the Go form of the Lean `machine` command.
type Machine[S, O, F any] struct {
	family       Family
	name         string
	entity       *Entity
	starts       []S
	ends         func(S) bool
	bindings     []binding[S, O, F]
	evidence     [][2]string
	unobservable map[string]bool
	refinement   *refinementDecl[S]
	names        claimNames

	visibleFact    func(F) bool
	visibleOutcome func(O) bool
	coverStarts    bool
	assumptions    []Assumption

	once  sync.Once
	table *Table
	err   error
}

// NewMachine starts a machine declaration under a family.
func NewMachine[S, O, F any](family Family, name string) *Machine[S, O, F] {
	return &Machine[S, O, F]{family: family, name: name, unobservable: map[string]bool{}}
}

// Name is the machine's declared name.
func (m *Machine[S, O, F]) Name() string { return m.name }

// Starts lists the states the machine starts in.
func (m *Machine[S, O, F]) Starts(states ...S) *Machine[S, O, F] { m.starts = states; return m }

// Ends says which states the machine may end in.
func (m *Machine[S, O, F]) Ends(end func(S) bool) *Machine[S, O, F] { m.ends = end; return m }

// Evidence names the recorded event or observation that confirms a fact, by the fact's
// constructor name, as a Lean `evidence:` line does.
func (m *Machine[S, O, F]) Evidence(fact, recorded string) *Machine[S, O, F] {
	m.evidence = append(m.evidence, [2]string{fact, recorded})
	return m
}

// Step0 binds the step function of an action with no input, or of a timer.
func (m *Machine[S, O, F]) Step0(a *Action0, f func(S) []Step[S, O, F]) *Machine[S, O, F] {
	m.bindings = append(m.bindings, binding[S, O, F]{a.ActionDecl, func(s S, _ Class) []Step[S, O, F] {
		return f(s)
	}})
	return m
}

// Step1 binds the step function of a one-input action. The generic method infers A from both
// arguments, so a step written for another action's input is a compile error.
func (m *Machine[S, O, F]) Step1[A any](a *Action1[A], f func(S, A) []Step[S, O, F]) *Machine[S, O, F] {
	m.bindings = append(m.bindings, binding[S, O, F]{a.ActionDecl, func(s S, c Class) []Step[S, O, F] {
		return f(s, c.values[0].Interface().(A))
	}})
	return m
}

// Restrict derives a machine that keeps the rows of the named actions and drops the rest: the Go
// form of Lean's `from: <machine> restrict: [...]`. It keeps the source's state type, starts and
// ends, owns its own name and Definition IDs, and does not inherit a refinement.
func (m *Machine[S, O, F]) Restrict(family Family, name string, keep ...*ActionDecl) *Machine[S, O, F] {
	d := NewMachine[S, O, F](family, name)
	d.entity, d.starts, d.ends = m.entity, m.starts, m.ends
	d.evidence = append(d.evidence, m.evidence...)
	d.assumptions = append(d.assumptions, m.assumptions...)
	for _, b := range m.bindings {
		if slices.Contains(keep, b.decl) {
			d.bindings = append(d.bindings, b)
		}
	}
	return d
}

// Table enumerates the machine: every state in catalog order, every action class sorted by key,
// and one row per enabled pair, states-major, as `Umpire.Command.Finite.enumerate` does. It is
// computed once.
func (m *Machine[S, O, F]) Table() (*Table, error) {
	m.once.Do(func() { m.table, m.err = m.build() })
	return m.table, m.err
}

func (m *Machine[S, O, F]) build() (*Table, error) {
	if m.refinement == nil && (m.visibleFact != nil || m.visibleOutcome != nil || m.coverStarts) {
		return nil, errorf(m.name, "the machine names what a refined machine sees, and refines none")
	}
	t := &Table{Machine: m.name, Family: m.family, stateValue: map[string]any{}}
	states, err := m.catalogs(t)
	if err != nil {
		return nil, err
	}
	bound, err := m.bind(t)
	if err != nil {
		return nil, err
	}
	if err := m.enumerate(t, states, bound); err != nil {
		return nil, err
	}
	if err := m.startsAndEnds(t, states); err != nil {
		return nil, err
	}
	if m.entity != nil {
		t.Entity = m.entity.Name
	}
	t.alter = m.alterer(t)
	t.Evidence = append([][2]string{}, m.evidence...)
	t.Assumptions = slices.Clone(m.assumptions)
	t.fieldValues = m.fieldValues(t, states)
	classes := make([]Class, len(bound))
	for i, b := range bound {
		classes[i] = b.class
	}
	t.keyClaims = claimsOf(t, classes)
	t.finish()
	return t, nil
}

// fieldValues is each state's fields as atoms, in the structure's field order, followed by the
// refined machine's state for a refining machine: what a Contract compares a state's fields by
// (`DeclaredModel.stateFieldValues`).
func (m *Machine[S, O, F]) fieldValues(t *Table, states []S) map[string][]Atom {
	out := map[string][]Atom{}
	st := reflect.TypeFor[S]()
	if st.Kind() != reflect.Struct {
		return out
	}
	for _, s := range states {
		v := reflect.ValueOf(s)
		var atoms []Atom
		for i := range st.NumField() {
			f := st.Field(i)
			if !f.IsExported() {
				continue
			}
			atoms = append(atoms, Atom{ID: t.Family.ID("state-field", t.owner(), fieldName(f)),
				Value: keyOf(v.Field(i), f.Type)})
		}
		if m.refinement != nil {
			atoms = append(atoms, Atom{ID: t.Family.ID("state-field", t.owner(), t.refinedField),
				Value: m.refinement.mapKey(s)})
		}
		out[KeyOf(s)] = atoms
	}
	return out
}

// alterer rebuilds a typed step with its state, outcome or one fact changed, for the predicate
// lowering to ask the Property about.
func (m *Machine[S, O, F]) alterer(t *Table) alterer {
	outcomes, _ := DomainOf[O]()
	outcomeOf := map[string]O{}
	for _, o := range outcomes {
		outcomeOf[KeyOf(o)] = o
	}
	rebuild := func(r Result, f func(*Step[S, O, F])) Result {
		step, ok := r.Step.(Step[S, O, F])
		if !ok {
			return r
		}
		step.Facts = slices.Clone(step.Facts)
		f(&step)
		out := Result{Outcome: KeyOf(step.Outcome), State: KeyOf(step.State), Step: step, Facts: []string{}}
		for _, fact := range step.Facts {
			out.Facts = append(out.Facts, KeyOf(fact))
		}
		return out
	}
	return alterer{
		state: func(r Result, key string) Result {
			return rebuild(r, func(s *Step[S, O, F]) {
				if state, ok := t.stateValue[key].(S); ok {
					s.State = state
				}
			})
		},
		outcome: func(r Result, key string) Result {
			return rebuild(r, func(s *Step[S, O, F]) { s.Outcome = outcomeOf[key] })
		},
		without: func(r Result, key string) Result {
			return rebuild(r, func(s *Step[S, O, F]) {
				s.Facts = slices.DeleteFunc(s.Facts, func(f F) bool { return KeyOf(f) == key })
			})
		},
	}
}

// catalogs fills the state, outcome and fact catalogs and the state fields, and returns the typed
// states in catalog order.
func (m *Machine[S, O, F]) catalogs(t *Table) ([]S, error) {
	states, err := DomainOf[S]()
	if err != nil {
		return nil, errorf(m.name, "state type: %v", err)
	}
	outcomes, err := DomainOf[O]()
	if err != nil {
		return nil, errorf(m.name, "outcome type: %v", err)
	}
	facts, err := DomainOf[F]()
	if err != nil {
		return nil, errorf(m.name, "fact type: %v", err)
	}
	for _, s := range states {
		k := KeyOf(s)
		t.States = append(t.States, k)
		t.stateValue[k] = s
	}
	for _, o := range outcomes {
		t.Outcomes = append(t.Outcomes, KeyOf(o))
	}
	t.Facts = []string{}
	for _, f := range facts {
		t.Facts = append(t.Facts, KeyOf(f))
	}
	if st := reflect.TypeFor[S](); st.Kind() == reflect.Struct {
		for i := range st.NumField() {
			if f := st.Field(i); f.IsExported() {
				t.StateFields = append(t.StateFields, fieldName(f))
			}
		}
	}
	if m.refinement != nil {
		// A refining machine carries the refined state it reads as in a field named after the
		// refined machine (`Umpire.Command.refinedProperty`).
		t.refinedField = m.refinement.product.Name()
		t.StateFields = append(t.StateFields, t.refinedField)
	}
	return states, nil
}

type boundClass[S, O, F any] struct {
	class Class
	step  func(S, Class) []Step[S, O, F]
}

// bind lists every class of every bound action, sorted by key, and rejects a class two steps bind.
func (m *Machine[S, O, F]) bind(t *Table) ([]boundClass[S, O, F], error) {
	var bound []boundClass[S, O, F]
	for _, b := range m.bindings {
		classes, err := b.decl.classes()
		if err != nil {
			return nil, errorf(m.name, "%v", err)
		}
		for _, c := range classes {
			bound = append(bound, boundClass[S, O, F]{c, b.step})
		}
	}
	slices.SortFunc(bound, func(a, b boundClass[S, O, F]) int {
		return compareStrings(a.class.Key(), b.class.Key())
	})
	for i, b := range bound {
		k := b.class.Key()
		if i > 0 && bound[i-1].class.Key() == k {
			return nil, errorf(m.name, "two steps bind the action class %q", k)
		}
		t.Actions = append(t.Actions, k)
	}
	return bound, nil
}

// enumerate evaluates every step function once per state and class, states-major, keeping the
// enabled pairs as rows and rejecting a result outside the state domain.
func (m *Machine[S, O, F]) enumerate(t *Table, states []S, bound []boundClass[S, O, F]) error {
	for _, s := range states {
		sk := KeyOf(s)
		for _, b := range bound {
			steps := b.step(s, b.class)
			if len(steps) == 0 {
				continue
			}
			row := Row{Key: rowKey(sk, b.class.Key()), Source: sk, Action: b.class.Key()}
			for _, step := range steps {
				res, err := m.result(t, row.Key, step)
				if err != nil {
					return err
				}
				row.Results = append(row.Results, res)
			}
			t.Rows = append(t.Rows, row)
		}
	}
	return nil
}

func (m *Machine[S, O, F]) result(t *Table, row string, step Step[S, O, F]) (Result, error) {
	nk := KeyOf(step.State)
	if _, ok := t.stateValue[nk]; !ok {
		return Result{}, errorf(m.name, "row %s lands in %s, which is outside the state domain", row, nk)
	}
	res := Result{Outcome: KeyOf(step.Outcome), State: nk, Facts: []string{}, Step: step, Because: step.Because}
	for _, f := range step.Facts {
		res.Facts = append(res.Facts, KeyOf(f))
	}
	return res, nil
}

func (m *Machine[S, O, F]) startsAndEnds(t *Table, states []S) error {
	for _, s := range m.starts {
		k := KeyOf(s)
		if _, ok := t.stateValue[k]; !ok {
			return errorf(m.name, "start %s is outside the state domain", k)
		}
		t.Starts = append(t.Starts, k)
	}
	if len(t.Starts) == 0 {
		return errorf(m.name, "the machine declares no start")
	}
	for _, s := range states {
		if m.ends != nil && m.ends(s) {
			t.Ends = append(t.Ends, KeyOf(s))
		}
	}
	return nil
}
