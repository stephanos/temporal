package model

import (
	"errors"
	"slices"

	umpirespb "go.temporal.io/server/api/umpire/v1"
)

// Decision is one branch the evaluator took while it evaluated a step function: which way an `if`
// went, or which case of a `match` it took, at the position of the `if` or the `match`.
type Decision struct {
	Position string
	// Expr is the decided expression, an If or a Match.
	Expr *umpirespb.Expr
	// Then is the way an `if` went. Case is the index of the case a `match` took, or -1 when no case
	// matched and the value is a hole.
	Then bool
	Case int
	// Wildcard is whether the case taken matches every value: a default arm.
	Wildcard bool
	// State is whether deciding read the state the step function was given. An `if` whose condition
	// names no field of the state decided on something else: an input, or nothing at all.
	State bool
	// Calls is the functions deciding called, in call order: the named predicates the step function
	// evaluated to decide.
	Calls []string
	// Nested is whether the decision was taken while another was being decided, inside its condition,
	// scrutinee or guard, rather than on the way to the value.
	Nested bool
}

// Match is whether a `match` decided, rather than an `if`.
func (d Decision) Match() bool { return d.Expr.GetMatch() != nil }

// Why is how one state and class of a machine evaluate: the steps, or the hole the value is, and the
// branch decisions taken on the way, in the order the evaluator took them. A channel's delivery and
// loss take no decision of a step function: their rows are derived (model/SEMANTICS.md, Channels).
type Why struct {
	Steps     []Value
	Hole      *Hole
	Decisions []Decision
}

// Last is the last decision taken on the way to the value, which is the one that decided it, or false
// when the value took none.
func (w *Why) Last() (Decision, bool) {
	for i := len(w.Decisions) - 1; i >= 0; i-- {
		if !w.Decisions[i].Nested {
			return w.Decisions[i], true
		}
	}
	return Decision{}, false
}

// Why evaluates a machine's step function at one state and class again, recording the decisions the
// evaluation takes. It computes nothing the table does not: the steps are the row's, or none for a
// disabled pair.
func (in *Interpreter) Why(m *Machine, state, class string) (*Why, error) {
	s, ok := m.State(state)
	if !ok {
		return nil, ErrorAt(m.Decl.GetPosition(), "%s has no state %s", m.Decl.GetName(), state)
	}
	i := slices.IndexFunc(m.Classes, func(c Class) bool { return c.Key == class })
	if i < 0 {
		return nil, ErrorAt(m.Decl.GetPosition(), "%s has no class %s", m.Decl.GetName(), class)
	}
	c := m.Classes[i]
	if c.Action.GetDelivers() != "" || c.Action.GetLoses() != "" {
		steps, err := in.steps(m.Decl, s, c)
		return explained(steps, nil, err)
	}
	traced := *in
	traced.trace = &tracer{}
	tainted := make([]bool, 1+len(c.Inputs))
	tainted[0] = true
	v, err := traced.call(c.step, append([]Value{s}, c.Inputs...), tainted, c.at)
	var steps []Value
	if err == nil {
		steps, err = in.stepList(m.Decl, c, v)
	}
	return explained(steps, traced.trace.decisions, err)
}

func explained(steps []Value, decisions []Decision, err error) (*Why, error) {
	w := &Why{Steps: steps, Decisions: decisions}
	var hole *Hole
	if errors.As(err, &hole) {
		w.Hole, w.Steps = hole, nil
		return w, nil
	}
	if err != nil {
		return nil, err
	}
	return w, nil
}

// tracer records what a traced evaluation decides: its decisions, the reads of a value computed from
// the state, the functions called, and how many decisions are being taken around the evaluation.
type tracer struct {
	decisions []Decision
	reads     int
	calls     []string
	depth     int
}

// pending is what a decision being taken started from.
type pending struct {
	reads int
	calls int
}

func (in *Interpreter) reads() int {
	if in.trace == nil {
		return 0
	}
	return in.trace.reads
}

// deciding starts a decision: what follows until decided is its condition, scrutinee or guard.
func (in *Interpreter) deciding() pending {
	if in.trace == nil {
		return pending{}
	}
	in.trace.depth++
	return pending{reads: in.trace.reads, calls: len(in.trace.calls)}
}

func (in *Interpreter) decided(p pending, x *umpirespb.Expr, then bool, taken int, wildcard bool) {
	if in.trace == nil {
		return
	}
	in.trace.depth--
	in.trace.decisions = append(in.trace.decisions, Decision{Position: Where(x.GetPosition()), Expr: x, Then: then,
		Case: taken, Wildcard: wildcard, State: in.trace.reads > p.reads, Calls: slices.Clone(in.trace.calls[p.calls:]),
		Nested: in.trace.depth > 0})
}

// tracedCall evaluates a call's arguments one by one, so that each parameter knows whether its value
// was computed from the state.
func (in *Interpreter) tracedCall(x *umpirespb.Expr, c *umpirespb.Call, e *env) (Value, error) {
	args := make([]Value, len(c.GetArgs()))
	state := make([]bool, len(c.GetArgs()))
	for i, a := range c.GetArgs() {
		reads := in.trace.reads
		v, err := in.eval(a, e)
		if err != nil {
			return Value{}, err
		}
		args[i], state[i] = v, in.trace.reads > reads
	}
	return in.call(c.GetFunction(), args, state, x.GetPosition())
}

// matchesAnything is whether a pattern matches every value: a wildcard, or a name bound to one.
func matchesAnything(p *umpirespb.Pattern) bool {
	switch k := p.GetKind().(type) {
	case *umpirespb.Pattern_Wildcard:
		return true
	case *umpirespb.Pattern_Bind:
		return matchesAnything(k.Bind.GetPattern())
	default:
		return false
	}
}

// Reads calls a function and says which of its arguments the evaluation read. A claim whose
// evaluation at a step does not read the step says nothing about it there: `!paused(before) ||
// after.state.phase != started` reads its step only where the state before is paused.
func (in *Interpreter) Reads(function string, args []Value, at *umpirespb.Position) (Value, []bool, error) {
	read := make([]bool, len(args))
	if len(args) == 0 {
		v, err := in.Call(function, args, at)
		return v, read, err
	}
	var out Value
	for i := range args {
		traced := *in
		traced.trace = &tracer{}
		tainted := make([]bool, len(args))
		tainted[i] = true
		v, err := traced.call(function, args, tainted, at)
		if err != nil {
			return Value{}, nil, err
		}
		out, read[i] = v, traced.trace.reads > 0
	}
	return out, read, nil
}
