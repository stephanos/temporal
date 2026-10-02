package checker

// Property is a pass/fail rule over the steps of a machine whose state type is S. A same-step
// Property names the action it is about under When and holds of the step that action produces; a
// transition Property holds of the state before a step and the step after it. The state type
// parameter is what lets the compiler reject a Query that pairs a Property with a Scenario over
// another machine's states.
type Property[S any] struct{ *PropertyDecl }

// PropertyBuilder declares a Property on a machine whose steps have type Step[S, O, F].
type PropertyBuilder[S, O, F any] struct {
	p *PropertyDecl
}

// Property starts a Property declaration on m.
func (m *Machine[S, O, F]) Property(name string) *PropertyBuilder[S, O, F] {
	m.names.declare("property", name)
	return &PropertyBuilder[S, O, F]{&PropertyDecl{Name: name, Machine: m}}
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

// Starts names the state the Scenario starts in.
func (b *ScenarioBuilder[S]) Starts(s S) *ScenarioBuilder[S] {
	b.s.Start = b.key(s)
	return b
}

// Actions pins the schedule to exactly these classes, in order.
func (b *ScenarioBuilder[S]) Actions(classes ...Class) *Scenario[S] {
	for _, c := range classes {
		b.s.Actions = append(b.s.Actions, c.Key())
	}
	return &Scenario[S]{b.s}
}

// Free admits any action at every step, within the Query's step limit.
func (b *ScenarioBuilder[S]) Free() *Scenario[S] {
	b.s.free = true
	return &Scenario[S]{b.s}
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
