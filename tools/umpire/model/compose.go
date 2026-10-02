package model

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	modelirspb "go.temporal.io/server/api/modelir/v1"
	umpire "go.temporal.io/server/tools/umpire/model/internal/checker"
)

// composedState reads a composition's states. The composed state is the composition's state record,
// one member state per field in the record's field order; its key joins the members' state keys by
// "_" in member order.
type composedState struct {
	stateType string
	members   []*subject
	// filledBy is, for each field of the record, the member that fills it.
	filledBy []int
}

func (c *composedState) value(parts []string) (Value, error) {
	state := Value{Kind: RecordValue, Type: c.stateType, Fields: make([]Value, len(c.filledBy))}
	for k, i := range c.filledBy {
		var err error
		if state.Fields[k], err = c.members[i].state(parts[i]); err != nil {
			return Value{}, err
		}
	}
	return state, nil
}

func (c *composedState) key(state Value) string {
	parts := make([]string, len(c.members))
	for k, i := range c.filledBy {
		parts[i] = state.Fields[k].Key()
	}
	return strings.Join(parts, "_")
}

// compositionSubject builds a composition with model/go's bounded composition of tables: its members
// by field, its syncs and its `ends`, within the scope's ceiling. Its starts are the product of every
// member's starts.
//
// A claim of a composition reads a composed step as model/go and the Scala front end do: its state is
// the composition's state record, and its outcome and facts are strings, the composed keys
// `<field>_<key>`.
func (b *binding) compositionSubject(c *modelirspb.Composition) *subject {
	s := &subject{name: c.GetName(), family: c.GetFamily(), at: c.GetPosition(), stateType: c.GetStateType()}
	spec := umpire.ComposeSpec{Family: Family(c.GetFamily()), Name: c.GetName(), Ceiling: b.scope.Compose}
	var state *composedState
	if state, s.err = b.composedMembers(c, s, &spec); s.err != nil {
		return s
	}
	for _, sync := range c.GetSyncs() {
		spec.Syncs = append(spec.Syncs, umpire.ComposeSync{Name: sync.GetName(),
			FirstMember: sync.GetFirst().GetMember(), FirstAction: sync.GetFirst().GetAction(),
			SecondMember: sync.GetSecond().GetMember(), SecondAction: sync.GetSecond().GetAction()})
	}
	var unread unknowns
	if spec.Ends, s.err = b.composedEnds(c, state, &unread); s.err != nil {
		return s
	}
	if s.table, s.err = umpire.ComposeTables(spec); s.err == nil {
		s.err = unread.err()
	}
	if s.err != nil {
		s.table = nil
		return s
	}
	s.state = func(key string) (Value, error) {
		parts, ok := s.table.Parts(key)
		if !ok {
			return Value{}, errorAt(s.at, "%s has no state %s", s.name, key)
		}
		return state.value(parts)
	}
	s.step = func(res Result) (Value, error) {
		step, ok := res.Step.(umpire.ComposedStep)
		if !ok {
			return Value{}, errorAt(s.at, "%s: the step into %s is no step of a composition", s.name, res.State)
		}
		reached, err := state.value(step.Parts)
		if err != nil {
			return Value{}, err
		}
		facts := Value{Kind: ListValue}
		for _, f := range res.Facts {
			facts.Items = append(facts.Items, Value{Kind: TextValue, Text: f})
		}
		return Value{Kind: RecordValue, Type: StepType,
			Fields: []Value{{Kind: TextValue, Text: res.Outcome}, reached, facts, {Kind: TextValue, Text: res.Because}}}, nil
	}
	s.key = func(v Value) (string, error) {
		if !b.in.conforms(v, named(c.GetStateType())) {
			return "", errorAt(s.at, "%s is no state of %s", v.Key(), s.name)
		}
		return state.key(v), nil
	}
	return s
}

// composedMembers resolves a composition's members into its spec, and says how its states read. It
// also notes why a Query over the composition is not supported, which does not wait on whether the
// composition builds.
func (b *binding) composedMembers(c *modelirspb.Composition, s *subject, spec *umpire.ComposeSpec) (*composedState, error) {
	fields := b.in.types[c.GetStateType()].GetRecord().GetFields()
	state := &composedState{stateType: c.GetStateType(), filledBy: slices.Repeat([]int{-1}, len(fields))}
	var unbuilt []error
	for i, mb := range c.GetMembers() {
		member := b.subject(mb.GetMachine())
		if member.monitored && s.unsupported == "" {
			s.unsupported = fmt.Sprintf("the member %s of %s is %s, which names monitors, and whether a member's monitors "+
				"watch a composition is undefined", mb.GetField(), s.name, mb.GetMachine())
		}
		k := slices.IndexFunc(fields, func(f *modelirspb.Field) bool { return f.GetName() == mb.GetField() })
		var err error
		switch {
		case member.err != nil:
			err = member.err
		case member.machine == nil:
			err = errorAt(s.at, "composition %s: its member %s is %s, which is no machine", s.name, mb.GetField(), mb.GetMachine())
		case k < 0:
			err = errorAt(s.at, "composition %s: its member %s is no field of %s", s.name, mb.GetField(), c.GetStateType())
		default:
			state.filledBy[k] = i
			err = b.replacement(spec, mb, member)
		}
		state.members = append(state.members, member)
		if err != nil {
			unbuilt = append(unbuilt, err)
		}
	}
	// Every member is read, so that one member's failure hides no other's. What the composition's
	// result is of several failures is the checker's to say, by its one rule for folding results.
	switch len(unbuilt) {
	case 0:
	case 1:
		return nil, unbuilt[0]
	default:
		return nil, &unbuiltMembers{errs: unbuilt}
	}
	if k := slices.Index(state.filledBy, -1); k >= 0 {
		return nil, errorAt(s.at, "composition %s: no member fills the field %s of %s", s.name, fields[k].GetName(), c.GetStateType())
	}
	return state, nil
}

// unbuiltMembers is why a composition is not built when several of its members could not be read
// or replaced: each member's own reason, in member order.
type unbuiltMembers struct{ errs []error }

func (e *unbuiltMembers) Error() string { return errors.Join(e.errs...).Error() }

func (e *unbuiltMembers) Unwrap() []error { return e.errs }

// replacement adds a member to the composition's spec. A member that replaces a machine must refine
// it and read every start of it, which is checked here so that a refinement that fails is reported
// with the machine it is of; the composition then relies on none of the replaced machine's
// assumptions.
func (b *binding) replacement(spec *umpire.ComposeSpec, mb *modelirspb.Member, member *subject) error {
	composed := umpire.ComposeMember{Field: mb.GetField(), Table: member.table}
	if mb.GetReplaces() != "" {
		replaced := b.subject(mb.GetReplaces())
		if replaced.err != nil {
			return replaced.err
		}
		reading, failed := b.reading(member, replaced)
		reading.CoverStarts = true
		if _, _, err := b.refines(member, replaced, reading, failed); err != nil {
			return err
		}
		composed.Replaces, composed.Refinement = replaced.table, reading
	}
	spec.Members = append(spec.Members, composed)
	return nil
}

// composedEnds is the composition's `ends` over a composed state, or nil when it declares none.
// A composed state it reaches a hole at is noted as unread and read as no end, so that the
// composition's every state is read: with a hole noted, the table that comes of it is not the
// composition's.
func (b *binding) composedEnds(c *modelirspb.Composition, state *composedState, unread *unknowns) (
	func(key string, parts []string) (bool, error), error) {
	if c.GetEnds() == nil {
		return nil, nil
	}
	end, err := b.in.Eval(c.GetEnds())
	if err != nil {
		return nil, err
	}
	return func(key string, parts []string) (bool, error) {
		composed, err := state.value(parts)
		if err != nil {
			return false, err
		}
		v, err := b.in.Apply(end, []Value{composed})
		if err != nil {
			return false, unread.note(err)
		}
		if v.Kind != BoolValue {
			return false, errorAt(c.GetEnds().GetPosition(), "%s: ends is %s at %s, not a Boolean", c.GetName(), v.Key(), key)
		}
		return v.Bool, nil
	}, nil
}

// Composed is one composition of the bound Model as Check reads it: the composed table the binding
// builds for its claims, and what that table's keys stand for. Table holds the states the members'
// starts reach, the starts, the composed classes, a row for each enabled pair and an unknown pair for
// each one a member's hole leaves unknown; a state and class with neither is a disabled pair. A
// Composed is not changed after it is given, and its functions are not safe to call from two
// goroutines at once.
type Composed struct {
	Decl  *modelirspb.Composition
	Table *Table
	// State is the composition's state record a state key stands for, one member state per field, and
	// Step the step record a claim reads of a result: that state, with the composed outcome and facts
	// as strings.
	State func(key string) (Value, error)
	Step  func(res Result) (Value, error)
	// Properties are the Properties the Model declares on the composition, in the Model's order, as
	// Check declares them to the generic search.
	Properties []BoundProperty
}

// Composition is the composition the Model declares under a name, as Check builds it within the
// Realizer's scope. A composition Check has no table for has none here, for the same reason: a
// *ComposeLimitError past the scope's ceiling, a *RefinementError for a member that does
// not refine what it replaces, and a member's own failure otherwise. A name that is no composition
// names nothing.
func (r *Realizer) Composition(name string) (*Composed, error) {
	i := slices.IndexFunc(r.b.model.GetCompositions(), func(c *modelirspb.Composition) bool { return c.GetName() == name })
	if i < 0 {
		return nil, &Error{Position: r.b.model.GetSource(), Message: "no composition " + name}
	}
	s := r.b.subject(name)
	if s.err != nil {
		return nil, s.err
	}
	out := &Composed{Decl: r.b.model.GetCompositions()[i], Table: s.table, State: s.state, Step: s.step}
	for _, p := range r.b.model.GetProperties() {
		if p.GetMachine() != name {
			continue
		}
		reading, err := r.b.propertyReads(s, p)
		if err != nil {
			return nil, err
		}
		out.Properties = append(out.Properties, boundProperty(p, reading))
	}
	return out, nil
}
