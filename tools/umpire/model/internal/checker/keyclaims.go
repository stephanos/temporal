package checker

// Claims over a table's keys, for a table computed outside this package (NewTable, ComposeTables):
// such a table has no typed states or steps, so its Properties, Scenarios and Monitors read keys.
// They build the same declarations the typed constructors build, which one search answers.

// tableModel is a table as the Model its key-level claims name.
type tableModel struct {
	table *Table
	names claimNames
}

func (m *tableModel) Name() string { return m.table.Machine }

func (m *tableModel) Table() (*Table, error) {
	if m.table.err != nil {
		return nil, m.table.err
	}
	return m.table, nil
}

func (m *tableModel) claimNames() *claimNames { return &m.names }

// KeyProperty declares a same-step Property over a table's keys: holds reads the step an action
// class admitted by when produces, and a nil when admits every class. whenLabel names what when
// admits in diagnostics. An error from holds is the search's to report: it is not a failed claim.
func KeyProperty(t *Table, name string, when func(action string) bool, whenLabel string,
	holds func(step Result) (bool, error)) *PropertyDecl {
	t.model.names.declare("property", name)
	return &PropertyDecl{Name: name, Machine: t.model, when: when, whenLabel: whenLabel, keyHolds: holds}
}

// KeyTransitionProperty declares a transition Property over a table's keys: holds reads the key of
// the state before a step and the step after it.
func KeyTransitionProperty(t *Table, name string, holds func(before string, step Result) (bool, error)) *PropertyDecl {
	t.model.names.declare("property", name)
	return &PropertyDecl{Name: name, Machine: t.model, keyHolds2: holds}
}

// KeyScenario declares a Scenario pinned to these action class keys, in order, from a start key.
func KeyScenario(t *Table, name, start string, actions ...string) *ScenarioDecl {
	t.model.names.declare("scenario", name)
	return &ScenarioDecl{Name: name, Machine: t.model, Start: start, Actions: actions}
}

// KeyFreeScenario declares a Scenario that admits any action at every step from a start key.
func KeyFreeScenario(t *Table, name, start string) *ScenarioDecl {
	t.model.names.declare("scenario", name)
	return &ScenarioDecl{Name: name, Machine: t.model, Start: start, free: true}
}

// KeyFind asks for a trace of the Scenario on which p holds. Nothing ties a key-level Property to
// its Scenario at compile time, so the Query is checked to name one table when it runs.
func KeyFind(name string, p *PropertyDecl, s *ScenarioDecl, limits Limits) *Query {
	return &Query{Name: name, Form: FindForm, Property: p, Scenario: s, Limits: limits}
}

// KeyVerify asks whether p holds on every trace of the Scenario.
func KeyVerify(name string, p *PropertyDecl, s *ScenarioDecl, limits Limits) *Query {
	return &Query{Name: name, Form: VerifyForm, Property: p, Scenario: s, Limits: limits}
}

// KeyVerifyRefined asks whether a Property declared on the refined table holds on every trace of
// the Scenario over the refining table, read through ref: a state by its map, and an outcome and
// facts by name. ref is the refinement RefineTables checked of the Scenario's table by the
// Property's.
func KeyVerifyRefined(name string, p *PropertyDecl, s *ScenarioDecl, ref *Refinement, limits Limits) *Query {
	return &Query{Name: name, Form: VerifyForm, Property: p, Scenario: s, Limits: limits,
		refinement: func() (*Refinement, error) {
			if ref == nil {
				return nil, errorf("query "+name, "a refined Query names the refinement it reads through")
			}
			return ref, nil
		}}
}

// keyLevel reports whether the Property reads keys rather than typed steps.
func (p *PropertyDecl) keyLevel() bool { return p.keyHolds != nil || p.keyHolds2 != nil }

// checkKeyClaims rejects a claim over a table's keys that names no function, or whose name its table
// declares twice.
func (q *Query) checkKeyClaims() error {
	if m, ok := q.Property.Machine.(*tableModel); ok {
		if !q.Property.keyLevel() {
			return errorf(q.decl(), "%s names no function that says whether it holds", q.Property.Name)
		}
		if err := m.names.duplicate(m.Name(), "property", q.Property.Name); err != nil {
			return err
		}
	}
	if m, ok := q.Scenario.Machine.(*tableModel); ok {
		if err := m.names.duplicate(m.Name(), "scenario", q.Scenario.Name); err != nil {
			return err
		}
	}
	return nil
}

// UnknownKind is what a search could not read.
type UnknownKind string

const (
	// UnknownRow is an unknown pair of the table the search explored.
	UnknownRow UnknownKind = "row"
	// UnknownClaim is a step on which a Property or a Monitor could not be read.
	UnknownClaim UnknownKind = "claim"
)

// UnknownReach is one unknown a check explored: the pair or the row, the state it is at, how many
// steps the check took to reach that state, and the shortest path it found there, which the table
// replays.
type UnknownReach struct {
	Kind   UnknownKind
	Row    string
	Source string
	Action string
	Depth  int
	Prefix *Trace
	Cause  error
}

// Incomplete reports that the answer found no witness and explored an unknown, so what it says is
// absent may lie behind one. A found witness or counterexample stands whatever was unknown.
func (a Answer) Incomplete() bool {
	return (a.Outcome == VerifiedWithinLimits || a.Outcome == NotFound) && len(a.Unknown) > 0
}

// observeKeys advances a key-level Property's monitor over one step.
func (s *searcher) observeKeys(n node, row Row, res Result) (monitor, error) {
	p, mon := s.q.Property, n.mon
	if !p.IsTransition() && !p.Triggers(row.Action) {
		return mon, nil
	}
	held, err := s.holdsOnKeys(n.state, res)
	if err != nil {
		return mon, wrapError(s.q.decl(), err)
	}
	mon.fired, mon.held = true, mon.held && held
	s.exercised = true
	return mon, nil
}

// holdsOnKeys asks a key-level Property about one step: the step itself, or the refined table's
// through the refinement.
func (s *searcher) holdsOnKeys(before string, res Result) (bool, error) {
	p, step := s.q.Property, res
	if s.ref != nil {
		var err error
		if step, err = s.ref.keyStep(res); err != nil {
			return false, err
		}
		if p.IsTransition() {
			if before, err = s.ref.MapState(before); err != nil {
				return false, err
			}
		}
	}
	if p.IsTransition() {
		return p.keyHolds2(before, step)
	}
	return p.keyHolds(step)
}

// keyStep reads a refining result as the refined table's step over keys: its state through the map,
// its outcome by name, and the facts the refined table names.
func (r *Refinement) keyStep(res Result) (Result, error) {
	state, err := r.MapState(res.State)
	if err != nil {
		return Result{}, err
	}
	return Result{Outcome: res.Outcome, State: state, Facts: append([]string{}, mapFacts(res.Facts, r.product.Facts)...),
		Because: res.Because}, nil
}

// noteUnknownRows records the unknown pairs at a node the Scenario schedules as its next step.
func (s *searcher) noteUnknownRows(i int) {
	n := s.nodes[i]
	for _, u := range s.t.UnknownFrom(n.state) {
		if s.admits(n, u.Action) {
			s.noteUnknown(UnknownRow, i, u.Row, u.Action, u.Cause)
		}
	}
}

// noteUnknown records an unknown at a node, once per kind and row: the search is breadth-first, so
// the first node it is met at is the nearest.
func (s *searcher) noteUnknown(kind UnknownKind, i int, row, action string, cause error) {
	key := [2]string{string(kind), row}
	if s.noted[key] {
		return
	}
	if s.noted == nil {
		s.noted = map[[2]string]bool{}
	}
	s.noted[key] = true
	n := s.nodes[i]
	s.unknown = append(s.unknown, UnknownReach{Kind: kind, Row: row, Source: n.state, Action: action,
		Depth: n.pos, Prefix: s.witness(i), Cause: cause})
}

// KeyMonitor declares a Monitor over a table's keys: next turns its state, the key of the state
// before a step and the step's result into its state after the step, and violated says whether a
// state violates it where at reads it. An error from either is the search's to report.
func KeyMonitor(name, initial string, next func(monitor, before string, step Result) (string, error),
	violated func(monitor string) (bool, error), at Evaluation) *Monitor {
	return &Monitor{Name: name, Initial: initial, At: at, keyNext: next, keyViolated: violated}
}

// AfterKey reads a Monitor's verdict after each step whose result f accepts, where f may fail.
func AfterKey(f func(Result) (bool, error)) Evaluation {
	return Evaluation{kind: afterSteps, afterKey: f}
}
