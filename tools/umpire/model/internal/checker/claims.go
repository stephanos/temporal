package checker

import (
	"fmt"
	"strings"
)

// Property is a pass/fail rule over the steps of a machine whose state type is S. A same-step
// Property names the action it is about under When and holds of the step that action produces; a
// transition Property holds of the state before a step and the step after it. The state type
// parameter is what lets the compiler reject a Query that pairs a Property with a Scenario over
// another machine's states.
type Property[S any] struct{ *PropertyDecl }

// PropertyDecl is a Property's untyped part, which Queries and the search read.
type PropertyDecl struct {
	Name      string
	Machine   Model
	when      func(action string) bool
	whenLabel string
	holds     func(step any) bool
	holds2    func(before, after any) bool
	// keyHolds and keyHolds2 are the claim of a Property declared over a table's keys.
	keyHolds  func(step Result) (bool, error)
	keyHolds2 func(before string, step Result) (bool, error)
}

// IsTransition reports whether the Property relates a step to the state before it.
func (p *PropertyDecl) IsTransition() bool { return p.holds2 != nil || p.keyHolds2 != nil }

// Triggers reports whether a same-step Property is about the step of this action class.
func (p *PropertyDecl) Triggers(action string) bool { return p.when == nil || p.when(action) }

// PropertyBuilder declares a Property on a machine whose steps have type Step[S, O, F].
type PropertyBuilder[S, O, F any] struct {
	p *PropertyDecl
}

// Property starts a Property declaration on m.
func (m *Machine[S, O, F]) Property(name string) *PropertyBuilder[S, O, F] {
	m.names.declare("property", name)
	return &PropertyBuilder[S, O, F]{&PropertyDecl{Name: name, Machine: m}}
}

// Property starts a Property declaration on a composition. A composed step's outcome and facts
// are their composed keys.
func (c *Composition[S]) Property(name string) *PropertyBuilder[S, string, string] {
	c.names.declare("property", name)
	return &PropertyBuilder[S, string, string]{&PropertyDecl{Name: name, Machine: c}}
}

// When restricts the Property to the step of one action class.
func (b *PropertyBuilder[S, O, F]) When(class Class) *PropertyBuilder[S, O, F] {
	key := class.Key()
	b.p.when = func(action string) bool { return action == key }
	b.p.whenLabel = key
	return b
}

// WhenAction restricts the Property to the steps of every class of an action, including a
// composition's synchronized step of that name.
func (b *PropertyBuilder[S, O, F]) WhenAction(name string) *PropertyBuilder[S, O, F] {
	b.p.when = func(action string) bool { return actionName(action) == name }
	b.p.whenLabel = name
	return b
}

// Holds finishes a same-step Property.
func (b *PropertyBuilder[S, O, F]) Holds(f func(Step[S, O, F]) bool) *Property[S] {
	b.p.holds = func(step any) bool { return f(step.(Step[S, O, F])) }
	return &Property[S]{b.p}
}

// HoldsAcross finishes a transition Property: f reads the state before a step and the step after.
func (b *PropertyBuilder[S, O, F]) HoldsAcross(f func(before S, after Step[S, O, F]) bool) *Property[S] {
	b.p.holds2 = func(before, after any) bool { return f(before.(S), after.(Step[S, O, F])) }
	return &Property[S]{b.p}
}

// Scenario is a named action schedule over a machine whose state type is S, from one start: the
// path a Query runs. A pinned Scenario lists its actions exactly; a free one lists none and admits
// any action at every step.
type Scenario[S any] struct{ *ScenarioDecl }

// ScenarioDecl is a Scenario's untyped part.
type ScenarioDecl struct {
	Name    string
	Machine Model
	Start   string
	Actions []string
	// Classes keeps the declared classes, for a realization that places them.
	Classes []Class
	free    bool
}

// ScenarioBuilder declares a Scenario.
type ScenarioBuilder[S any] struct {
	s   *ScenarioDecl
	key func(S) string
}

// Scenario starts a Scenario declaration on m.
func (m *Machine[S, O, F]) Scenario(name string) *ScenarioBuilder[S] {
	m.names.declare("scenario", name)
	return &ScenarioBuilder[S]{&ScenarioDecl{Name: name, Machine: m}, KeyOf[S]}
}

// Scenario starts a Scenario declaration on a composition.
func (c *Composition[S]) Scenario(name string) *ScenarioBuilder[S] {
	c.names.declare("scenario", name)
	return &ScenarioBuilder[S]{&ScenarioDecl{Name: name, Machine: c}, c.stateKey}
}

// Starts names the state the Scenario starts in.
func (b *ScenarioBuilder[S]) Starts(s S) *ScenarioBuilder[S] {
	b.s.Start = b.key(s)
	return b
}

// Actions pins the schedule to exactly these classes, in order.
func (b *ScenarioBuilder[S]) Actions(classes ...Class) *Scenario[S] {
	for _, c := range classes {
		b.s.Actions = append(b.s.Actions, c.Key())
		b.s.Classes = append(b.s.Classes, c)
	}
	return &Scenario[S]{b.s}
}

// ActionKeys pins the schedule to these action keys, for a composition whose keys name members.
func (b *ScenarioBuilder[S]) ActionKeys(keys ...string) *Scenario[S] {
	b.s.Actions = append(b.s.Actions, keys...)
	return &Scenario[S]{b.s}
}

// Free admits any action at every step, within the Query's step limit.
func (b *ScenarioBuilder[S]) Free() *Scenario[S] {
	b.s.free = true
	return &Scenario[S]{b.s}
}

// Limits bounds a Query: steps is the depth bound, actions the schedule length, and search the
// number of product states the search may visit before it reports limit-reached.
type Limits struct {
	Name    string
	Steps   int
	Actions int
	Search  int
}

// QueryForm is what a Query asks.
type QueryForm int

const (
	// FindForm asks for one trace of the Scenario on which the Property holds.
	FindForm QueryForm = iota
	// VerifyForm asks whether the Property holds on every trace of the Scenario.
	VerifyForm
)

// Query is a bounded question about a machine: a Property, a Scenario, and Limits.
type Query struct {
	Name       string
	Form       QueryForm
	Property   *PropertyDecl
	Scenario   *ScenarioDecl
	Limits     Limits
	refinement func() (*Refinement, error)
	monitors   []*Monitor
	// Unknown says which errors of a Property's or a Monitor's functions leave a step unknown rather
	// than fail the search: the step is reported, the search does not continue through it, and an
	// answer with no witness is incomplete. Nil lets every such error fail the search.
	Unknown func(error) bool
}

// Find asks for a trace of the Scenario on which p holds. p and the Scenario share the state type
// S, so a Property of another machine does not compile here.
func (s *Scenario[S]) Find(name string, p *Property[S], limits Limits) *Query {
	return &Query{Name: name, Form: FindForm, Property: p.PropertyDecl, Scenario: s.ScenarioDecl, Limits: limits}
}

// Verify asks whether p holds on every trace of the Scenario.
func (s *Scenario[S]) Verify(name string, p *Property[S], limits Limits) *Query {
	return &Query{Name: name, Form: VerifyForm, Property: p.PropertyDecl, Scenario: s.ScenarioDecl, Limits: limits}
}

// Via is a typed handle on a machine with states S refining a machine with states PS.
type Via[S, PS any] struct {
	refinement func() (*Refinement, error)
	product    Model
}

// Via returns the handle a refined Query reads through. The generic method fixes PS to the product
// machine's state type; that the machine declares this refinement is checked when the Query runs.
func (m *Machine[S, O, F]) Via[PS, PO, PF any](product *Machine[PS, PO, PF]) Via[S, PS] {
	return Via[S, PS]{refinement: m.Refinement, product: product}
}

// VerifyRefined asks whether a Property declared on the refined machine holds on every trace of
// this Scenario over the refining machine, read through the refinement. The Property's state type
// must be the refined machine's and the Scenario's the refining machine's, which the compiler
// checks through the Via handle.
func (s *Scenario[S]) VerifyRefined[PS any](name string, p *Property[PS], via Via[S, PS], limits Limits) *Query {
	return &Query{Name: name, Form: VerifyForm, Property: p.PropertyDecl, Scenario: s.ScenarioDecl, Limits: limits,
		refinement: via.refinement}
}

// Outcome is a Query's answer, spelled as Lean spells `PlanningOutcome.name`.
type Outcome string

const (
	Found                Outcome = "found"
	NotFound             Outcome = "not-found"
	VerifiedWithinLimits Outcome = "verified-within-limits"
	CounterexampleFound  Outcome = "counterexample-found"
	LimitReached         Outcome = "limit-reached"
)

// Atom is a value on a trace with the Definition ID it belongs to, as Lean's `ModelValue`.
type Atom struct {
	ID    string `json:"id"`
	Value string `json:"value"`
}

// TraceStep is one step of a witness.
type TraceStep struct {
	Action  Atom   `json:"action"`
	Outcome Atom   `json:"outcome"`
	State   Atom   `json:"state"`
	Facts   []Atom `json:"facts"`
}

// Trace is a witness: the start and the steps taken.
type Trace struct {
	Initial Atom        `json:"initial"`
	Steps   []TraceStep `json:"steps"`
}

// Answer is a Query's outcome, its witness for a found Query or its counterexample, and how much
// the search explored.
type Answer struct {
	Outcome     Outcome
	Witness     *Trace
	Explored    int
	Explanation string
	// Rows are the row keys the witness takes, in order.
	Rows []string
	// Exercised reports that the Property's clause fired on some explored step, so a verified
	// answer was earned rather than vacuous.
	Exercised bool
	// Monitor names the Monitor whose violation is the counterexample, or "".
	Monitor string
	// Monitors are the watching Monitors' verdicts, in the order the Query names them.
	Monitors []MonitorVerdict
	// Unknown lists the unknown pairs and steps the search explored, in the order it met them. A pair
	// past the step bound, one the Scenario does not schedule, and one at a state the search never
	// expanded are not explored.
	Unknown []UnknownReach
	// Expanded is how many product states the search read the successors of.
	Expanded int
}

func (q *Query) decl() string { return "query " + q.Name }

// check rejects a Query whose parts do not fit together.
func (q *Query) check() error {
	if q.Property == nil || q.Scenario == nil {
		return errorf(q.decl(), "a Query names a Property and a Scenario")
	}
	if err := q.checkKeyClaims(); err != nil {
		return err
	}
	if q.Form == FindForm && q.Property.IsTransition() {
		return errorf(q.decl(), "find names %s, a transition claim; a find realizes a same-step claim",
			q.Property.Name)
	}
	if q.refinement == nil && q.Property.Machine != q.Scenario.Machine {
		return errorf(q.decl(), "%s is declared on %s, but %s runs on %s, which does not refine it",
			q.Property.Name, q.Property.Machine.Name(), q.Scenario.Name, q.Scenario.Machine.Name())
	}
	if len(q.Scenario.Actions) > q.Limits.Actions {
		return errorf(q.decl(), "%s pins %d actions and the limits %s allow %d",
			q.Scenario.Name, len(q.Scenario.Actions), q.Limits.Name, q.Limits.Actions)
	}
	return q.checkMonitors()
}

func describeState(key string) string { return "{" + strings.ReplaceAll(key, "-", ", ") + "}" }

func (a Answer) String() string {
	return fmt.Sprintf("%s after %d product states%s", a.Outcome, a.Explored,
		map[bool]string{true: ": " + a.Explanation, false: ""}[a.Explanation != ""])
}
