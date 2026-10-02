package export

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"

	umpirespb "go.temporal.io/server/api/umpire/v1"
	umpiremodel "go.temporal.io/server/tools/umpire/model"
)

const pBackend = "p"

// maxRejectedRuns is how many rejected traces are also run one by one against the monitor's own
// assertion. Every trace is compared in the one agreement run whatever this is.
const maxRejectedRuns = 16

// PExport is one monitor of one machine written as a P program: the monitor as a P spec machine,
// translated from the IR's functions, and a driver that announces bounded event traces of the machine
// to it.
//
// The traces are every path of the machine from its starts within the depth, each ended by its first
// violation, by a state with no step, or by the depth. The driver is one machine that only announces
// events, so each test case has one schedule and one run of P's checker covers it whole.
//
// It is an event monitor and nothing more: the machine's transitions are not exported to P, and no
// refinement between P modules is claimed.
type PExport struct {
	// Model names the slice the monitor is exported from.
	Model   string
	Project string
	Text    string
	Machine string
	Monitor string
	Depth   int
	// Traces counts the event traces, Accepted the ones Go's monitor holds on, and Rejected the ones it
	// is violated on.
	Traces, Accepted, Rejected int
	// Tests are the test cases, with what Go's monitor says each must do.
	Tests []PTest
}

// PTest is one test case of an export. Violated is the step at which the monitor's assertion must
// fail, 0 where the run must find no bug, and Control marks the case that must fail because its
// expectation is wrong on purpose.
type PTest struct {
	Name     string
	Violated int
	Control  bool
}

// PResult is what P's checker reported of one test case.
type PResult struct {
	Ran     bool
	Bugs    int
	Message string
}

// ptype is a type of the IR as P is given it.
type ptype struct {
	p    string
	ref  *umpirespb.TypeRef
	step bool
}

type pwriter struct {
	s         *Slice
	mm        *umpiremodel.Machine
	decls     strings.Builder
	typeNames map[string]string
	functions map[string]pfunction
	step      ptype
}

type pfunction struct {
	name   string
	result ptype
}

func (w *pwriter) unsupported(at *umpirespb.Position, format string, args ...any) error {
	e := &UnsupportedError{Backend: pBackend, Construct: fmt.Sprintf(format, args...)}
	if at != nil {
		e.Position = at.GetFile() + ":" + strconv.Itoa(int(at.GetLine()))
	}
	return e
}

// typeRef is a type as P holds it: a Boolean, an integer, an enum none of whose cases has fields, a
// record of such, or a list of such. P has no enum whose cases carry values, so a type with one is
// not exported.
func (w *pwriter) typeRef(t *umpirespb.TypeRef) (ptype, error) {
	switch r := t.GetRef().(type) {
	case *umpirespb.TypeRef_Bool:
		return ptype{p: "bool", ref: t}, nil
	case *umpirespb.TypeRef_IntRange, *umpirespb.TypeRef_Int:
		return ptype{p: "int", ref: t}, nil
	case *umpirespb.TypeRef_List:
		item, err := w.typeRef(r.List)
		return ptype{p: "seq[" + item.p + "]", ref: t}, err
	case *umpirespb.TypeRef_Named:
		if r.Named == umpiremodel.StepType {
			return w.step, nil
		}
		if name, ok := w.typeNames[r.Named]; ok {
			return ptype{p: name, ref: t}, nil
		}
		decl, ok := w.s.types[r.Named]
		if !ok {
			return ptype{}, w.unsupported(nil, "the type %s", r.Named)
		}
		name := "t" + plain(r.Named)
		if decl.GetRecord() != nil {
			var fields []string
			for _, f := range decl.GetRecord().GetFields() {
				ft, err := w.typeRef(f.GetType())
				if err != nil {
					return ptype{}, err
				}
				fields = append(fields, "f_"+plain(f.GetName())+": "+ft.p)
			}
			fmt.Fprintf(&w.decls, "type %s = (%s);\n", name, strings.Join(fields, ", "))
		} else {
			var cases []string
			for _, c := range decl.GetEnum().GetCases() {
				if len(c.GetFields()) > 0 {
					return ptype{}, w.unsupported(decl.GetPosition(), "the type %s, whose case %s carries values: P's enums carry none", r.Named, c.GetName())
				}
				cases = append(cases, element(r.Named, c.GetName()))
			}
			fmt.Fprintf(&w.decls, "enum %s { %s }\n", name, strings.Join(cases, ", "))
		}
		w.typeNames[r.Named] = name
		return ptype{p: name, ref: t}, nil
	default:
		return ptype{}, w.unsupported(nil, "a channel's contents")
	}
}

// element is the name of an enum's case in P, where every enum's elements share one namespace.
func element(typ, name string) string { return "e" + plain(typ) + "_" + plain(name) }

// tuple writes a P named tuple, which takes a trailing comma when it has one field.
func tuple(fields []string) string {
	if len(fields) == 1 {
		return "(" + fields[0] + ",)"
	}
	return "(" + strings.Join(fields, ", ") + ")"
}

// value writes a value of a type as a P expression.
func (w *pwriter) value(v umpiremodel.Value, t ptype) (string, error) {
	if t.step {
		fields := []string{"f_outcome = ", "f_state = "}
		outcome, err := w.typeRef(named(w.mm.Decl.GetOutcomeType()))
		if err != nil {
			return "", err
		}
		state, err := w.typeRef(named(w.mm.Decl.GetStateType()))
		if err != nil {
			return "", err
		}
		for i, ft := range []ptype{outcome, state} {
			written, err := w.value(v.Fields[i], ft)
			if err != nil {
				return "", err
			}
			fields[i] += written
		}
		return tuple(fields), nil
	}
	switch v.Kind {
	case umpiremodel.BoolValue:
		return strconv.FormatBool(v.Bool), nil
	case umpiremodel.IntValue:
		return strconv.FormatInt(v.Int, 10), nil
	case umpiremodel.EnumValue:
		if len(v.Fields) > 0 {
			return "", w.unsupported(nil, "a value of the case %s, which carries values", v.Case)
		}
		return element(t.ref.GetNamed(), v.Case), nil
	case umpiremodel.RecordValue:
		decl := w.s.types[t.ref.GetNamed()].GetRecord().GetFields()
		fields := make([]string, len(decl))
		for i, f := range decl {
			ft, err := w.typeRef(f.GetType())
			if err != nil {
				return "", err
			}
			written, err := w.value(v.Fields[i], ft)
			if err != nil {
				return "", err
			}
			fields[i] = "f_" + plain(f.GetName()) + " = " + written
		}
		return tuple(fields), nil
	case umpiremodel.ListValue:
		// P writes no sequence in place: the driver would have to build one, and no exported monitor
		// reads a step's facts yet.
		return "", w.unsupported(nil, "a list as a value of an event")
	default:
		return "", w.unsupported(nil, "the value %s", v.Key())
	}
}

// pscope is the names a function's body may read, with their types.
type pscope map[string]pbound

type pbound struct {
	name string
	typ  ptype
}

func (sc pscope) with(name string, b pbound) pscope {
	out := pscope{}
	for k, v := range sc {
		out[k] = v
	}
	out[name] = b
	return out
}

// pbody is a P function being written: its local variables, which P declares before its statements.
type pbody struct {
	locals []string
}

func (b *pbody) local(name string, t ptype) string {
	written := fmt.Sprintf("v%d_%s", len(b.locals), plain(name))
	b.locals = append(b.locals, fmt.Sprintf("  var %s: %s;\n", written, t.p))
	return written
}

// function writes a function of the IR as a P function, after the functions it calls.
func (w *pwriter) function(name string, at *umpirespb.Position) (pfunction, error) {
	if f, ok := w.functions[name]; ok {
		return f, nil
	}
	i := slices.IndexFunc(w.s.Model.GetFunctions(), func(f *umpirespb.Function) bool { return f.GetName() == name })
	if i < 0 {
		return pfunction{}, w.unsupported(at, "a call of %s, which the Model does not declare", name)
	}
	f := w.s.Model.GetFunctions()[i]
	if f.GetRequires() != nil {
		return pfunction{}, w.unsupported(f.GetPosition(), "the precondition of %s", name)
	}
	sc, params := pscope{}, make([]string, len(f.GetParams()))
	for k, p := range f.GetParams() {
		t, err := w.typeRef(p.GetType())
		if err != nil {
			return pfunction{}, err
		}
		written := fmt.Sprintf("p%d_%s", k, plain(p.GetName()))
		sc[p.GetName()] = pbound{written, t}
		params[k] = written + ": " + t.p
	}
	var body pbody
	statements, result, err := w.returns(f.GetBody(), sc, &body, "  ")
	if err != nil {
		return pfunction{}, err
	}
	out := pfunction{name: fmt.Sprintf("f%d_%s", i, plain(name)), result: result}
	fmt.Fprintf(&w.decls, "fun %s(%s): %s {\n%s%s}\n", out.name, strings.Join(params, ", "), result.p, strings.Join(body.locals, ""), statements)
	w.functions[name] = out
	return out, nil
}

// returns writes an expression as the statements that return its value. P has no conditional
// expression, so a conditional and a match are written as branches that each return, which is where
// the lifted functions have them; one inside another expression is not translated.
func (w *pwriter) returns(x *umpirespb.Expr, sc pscope, body *pbody, indent string) (string, ptype, error) {
	switch k := x.GetKind().(type) {
	case *umpirespb.Expr_If:
		cond, _, err := w.expr(k.If.GetCondition(), sc)
		if err != nil {
			return "", ptype{}, err
		}
		then, t, err := w.returns(k.If.GetThen(), sc, body, indent+"  ")
		if err != nil {
			return "", ptype{}, err
		}
		otherwise, _, err := w.returns(k.If.GetElse(), sc, body, indent+"  ")
		if err != nil {
			return "", ptype{}, err
		}
		return fmt.Sprintf("%sif (%s) {\n%s%s} else {\n%s%s}\n", indent, cond, then, indent, otherwise, indent), t, nil
	case *umpirespb.Expr_Let:
		value, t, err := w.expr(k.Let.GetValue(), sc)
		if err != nil {
			return "", ptype{}, err
		}
		local := body.local(k.Let.GetName(), t)
		rest, result, err := w.returns(k.Let.GetBody(), sc.with(k.Let.GetName(), pbound{local, t}), body, indent)
		return fmt.Sprintf("%s%s = %s;\n%s", indent, local, value, rest), result, err
	case *umpirespb.Expr_Match:
		return w.match(k.Match, sc, body, indent)
	default:
		value, t, err := w.expr(x, sc)
		return fmt.Sprintf("%sreturn %s;\n", indent, value), t, err
	}
}

// match writes a match as one branch per case, in order, each of which returns.
func (w *pwriter) match(m *umpirespb.Match, sc pscope, body *pbody, indent string) (string, ptype, error) {
	scrutinee, t, err := w.expr(m.GetScrutinee(), sc)
	if err != nil {
		return "", ptype{}, err
	}
	local := body.local("matched", t)
	var out strings.Builder
	fmt.Fprintf(&out, "%s%s = %s;\n", indent, local, scrutinee)
	var result ptype
	for _, c := range m.GetCases() {
		cond, bound, err := w.pattern(c.GetPattern(), pbound{local, t}, sc)
		if err != nil {
			return "", ptype{}, err
		}
		if c.GetGuard() != nil {
			guard, _, err := w.expr(c.GetGuard(), bound)
			if err != nil {
				return "", ptype{}, err
			}
			cond = fmt.Sprintf("(%s && %s)", cond, guard)
		}
		branch, bt, err := w.returns(c.GetBody(), bound, body, indent+"  ")
		if err != nil {
			return "", ptype{}, err
		}
		result = bt
		fmt.Fprintf(&out, "%sif (%s) {\n%s%s}\n", indent, cond, branch, indent)
	}
	// A value no case matches is an undeclared hole: the monitor has no state for it, and says so.
	fmt.Fprintf(&out, "%sassert false, \"a value no case matches: an undeclared hole\";\n", indent)
	return out.String(), result, nil
}

func (w *pwriter) pattern(p *umpirespb.Pattern, value pbound, sc pscope) (string, pscope, error) {
	switch k := p.GetKind().(type) {
	case *umpirespb.Pattern_Wildcard:
		return "true", sc, nil
	case *umpirespb.Pattern_Bind:
		return w.pattern(k.Bind.GetPattern(), value, sc.with(k.Bind.GetName(), value))
	case *umpirespb.Pattern_Literal:
		literal, err := w.value(w.s.literal(k.Literal), value.typ)
		return fmt.Sprintf("(%s == %s)", value.name, literal), sc, err
	case *umpirespb.Pattern_Case:
		if len(k.Case.GetFields()) > 0 {
			return "", nil, w.unsupported(nil, "a pattern of the case %s with fields", k.Case.GetCase())
		}
		return fmt.Sprintf("(%s == %s)", value.name, element(k.Case.GetType(), k.Case.GetCase())), sc, nil
	case *umpirespb.Pattern_Alternatives:
		var conds []string
		for _, alt := range k.Alternatives.GetPatterns() {
			cond, bound, err := w.pattern(alt, value, sc)
			if err != nil {
				return "", nil, err
			}
			if len(bound) != len(sc) {
				return "", nil, w.unsupported(nil, "an alternative of a pattern that binds a name")
			}
			conds = append(conds, cond)
		}
		return "(" + strings.Join(conds, " || ") + ")", sc, nil
	default:
		return "", nil, w.unsupported(nil, "a pattern of no known kind")
	}
}

// literal is a literal of the IR as a value.
func (s *Slice) literal(v *umpirespb.Value) umpiremodel.Value {
	out, err := s.in.Eval(&umpirespb.Expr{Kind: &umpirespb.Expr_Literal{Literal: v}})
	if err != nil {
		return umpiremodel.Value{}
	}
	return out
}

var boolType = ptype{p: "bool", ref: &umpirespb.TypeRef{Ref: &umpirespb.TypeRef_Bool{Bool: &umpirespb.Empty{}}}}

// expr writes an expression as a P expression, with its type.
func (w *pwriter) expr(x *umpirespb.Expr, sc pscope) (string, ptype, error) {
	at := x.GetPosition()
	switch k := x.GetKind().(type) {
	case *umpirespb.Expr_Literal:
		return w.literal(k.Literal, at)
	case *umpirespb.Expr_Var:
		if b, ok := sc[k.Var]; ok {
			return b.name, b.typ, nil
		}
		return "", ptype{}, w.unsupported(at, "the name %s, which nothing in scope binds", k.Var)
	case *umpirespb.Expr_Field:
		base, t, err := w.expr(k.Field.GetBase(), sc)
		if err != nil {
			return "", ptype{}, err
		}
		ft, err := w.field(t, k.Field.GetField(), at)
		return base + ".f_" + plain(k.Field.GetField()), ft, err
	case *umpirespb.Expr_Call:
		f, err := w.function(k.Call.GetFunction(), at)
		if err != nil {
			return "", ptype{}, err
		}
		args := make([]string, len(k.Call.GetArgs()))
		for i, a := range k.Call.GetArgs() {
			if args[i], _, err = w.expr(a, sc); err != nil {
				return "", ptype{}, err
			}
		}
		return f.name + "(" + strings.Join(args, ", ") + ")", f.result, nil
	case *umpirespb.Expr_Unary:
		operand, t, err := w.expr(k.Unary.GetOperand(), sc)
		if err != nil {
			return "", ptype{}, err
		}
		if k.Unary.GetOp() == umpirespb.Unary_OP_NOT {
			return "!(" + operand + ")", t, nil
		}
		return "(-" + operand + ")", t, nil
	case *umpirespb.Expr_Binary:
		return w.binary(k.Binary, sc, at)
	case *umpirespb.Expr_If, *umpirespb.Expr_Match, *umpirespb.Expr_Let:
		return "", ptype{}, w.unsupported(at, "a conditional, a match or a binding inside an expression: P has statements for them, and no expression")
	case *umpirespb.Expr_Hole:
		return "", ptype{}, w.unsupported(at, "a hole: the monitor has no state for unknown behavior")
	default:
		return "", ptype{}, w.unsupported(at, "an expression P is not given: a construction, a copy, a list, an anonymous function or a channel operation")
	}
}

// literal writes a literal of the IR, with its type.
func (w *pwriter) literal(l *umpirespb.Value, at *umpirespb.Position) (string, ptype, error) {
	v := w.s.literal(l)
	var t ptype
	var err error
	switch v.Kind {
	case umpiremodel.BoolValue:
		t = boolType
	case umpiremodel.IntValue:
		t, err = w.typeRef(&umpirespb.TypeRef{Ref: &umpirespb.TypeRef_Int{Int: &umpirespb.Empty{}}})
	case umpiremodel.EnumValue, umpiremodel.RecordValue:
		t, err = w.typeRef(named(v.Type))
	default:
		return "", ptype{}, w.unsupported(at, "the literal %s", v.Key())
	}
	if err != nil {
		return "", ptype{}, err
	}
	written, err := w.value(v, t)
	return written, t, err
}

func (w *pwriter) binary(b *umpirespb.Binary, sc pscope, at *umpirespb.Position) (string, ptype, error) {
	l, lt, err := w.expr(b.GetLeft(), sc)
	if err != nil {
		return "", ptype{}, err
	}
	r, _, err := w.expr(b.GetRight(), sc)
	if err != nil {
		return "", ptype{}, err
	}
	infix := map[umpirespb.Binary_Op]string{
		umpirespb.Binary_OP_EQ: "==", umpirespb.Binary_OP_NE: "!=", umpirespb.Binary_OP_AND: "&&", umpirespb.Binary_OP_OR: "||",
		umpirespb.Binary_OP_LT: "<", umpirespb.Binary_OP_LE: "<=", umpirespb.Binary_OP_GT: ">", umpirespb.Binary_OP_GE: ">=",
	}
	if op, ok := infix[b.GetOp()]; ok {
		return fmt.Sprintf("(%s %s %s)", l, op, r), boolType, nil
	}
	switch b.GetOp() {
	case umpirespb.Binary_OP_ADD:
		return fmt.Sprintf("(%s + %s)", l, r), lt, nil
	case umpirespb.Binary_OP_SUB:
		return fmt.Sprintf("(%s - %s)", l, r), lt, nil
	case umpirespb.Binary_OP_CONTAINS:
		return fmt.Sprintf("(%s in %s)", l, r), boolType, nil
	default:
		return "", ptype{}, w.unsupported(at, "the binary operator %s", b.GetOp())
	}
}

// field is the type of a field of a record or of the step record.
func (w *pwriter) field(t ptype, name string, at *umpirespb.Position) (ptype, error) {
	if t.step {
		switch name {
		case "outcome":
			return w.typeRef(named(w.mm.Decl.GetOutcomeType()))
		case "state":
			return w.typeRef(named(w.mm.Decl.GetStateType()))
		default:
			return ptype{}, w.unsupported(at, "a step's %s, which the event P is given does not carry", name)
		}
	}
	if decl := w.s.types[t.ref.GetNamed()].GetRecord(); decl != nil {
		for _, f := range decl.GetFields() {
			if f.GetName() == name {
				return w.typeRef(f.GetType())
			}
		}
	}
	return ptype{}, w.unsupported(at, "the field %s of a value that is no record", name)
}

// ptrace is one event trace: the steps of a path of the machine, and the step at which Go's monitor
// is first read and violated, or 0.
type ptrace struct {
	steps    []pstep
	violated int
}

type pstep struct {
	class  string
	before umpiremodel.Value
	after  umpiremodel.Value
}

// walker walks a machine's paths from its starts with its monitors, collecting event traces.
type walker struct {
	s       *Slice
	w       *watching
	from    map[string][]umpiremodel.Transition
	monitor int
	depth   int
	out     []ptrace
}

// traces is every path of the machine from its starts within a depth, each ended by its first
// violation of the monitor at index k, by a state with no step, or by the depth.
func (s *Slice) traces(mm *umpiremodel.Machine, k, depth int) ([]ptrace, error) {
	w, initial, err := s.watching(mm)
	if err != nil {
		return nil, err
	}
	walk := &walker{s: s, w: w, from: map[string][]umpiremodel.Transition{}, monitor: k, depth: depth}
	for _, tr := range mm.Transitions {
		walk.from[tr.Source.Key()] = append(walk.from[tr.Source.Key()], tr)
	}
	for _, start := range mm.Table.Starts {
		if err := walk.walk(start, initial, nil); err != nil {
			return nil, err
		}
	}
	return walk.out, nil
}

func (k *walker) walk(state string, mu []string, path []pstep) error {
	if len(path) == k.depth || len(k.from[state]) == 0 {
		if len(path) > 0 {
			k.out = append(k.out, ptrace{steps: slices.Clone(path)})
		}
		return nil
	}
	for _, tr := range k.from[state] {
		for _, step := range tr.Steps {
			if err := k.take(tr, step, mu, path); err != nil {
				return err
			}
		}
	}
	return nil
}

// take takes one result of a row: the trace ends there where the monitor is read and violated, and
// goes on from its state otherwise.
func (k *walker) take(tr umpiremodel.Transition, step umpiremodel.Value, mu []string, path []pstep) error {
	target := step.Fields[1].Key()
	taken, err := k.s.step(k.w, mu, tr.Source, step, target)
	if err != nil {
		return err
	}
	next := append(slices.Clone(path), pstep{tr.Class.Key, tr.Source, step})
	if taken.Read[k.monitor] && taken.Viol[k.monitor] {
		k.out = append(k.out, ptrace{steps: next, violated: len(next)})
		return nil
	}
	return k.walk(target, taken.Mu, next)
}

// pmonitor is one monitor as the P program's two spec machines read it: the P type of its state, its
// initial state, and the P expressions that advance it, read its verdict and say whether a state
// violates it.
type pmonitor struct {
	name     string
	state    ptype
	mu       ptype
	initial  string
	next     string
	read     string
	violated string
}

// monitor translates a monitor's declarations and functions.
func (w *pwriter) monitor(mo *umpirespb.Monitor) (pmonitor, error) {
	out := pmonitor{name: mo.GetName(), read: "true"}
	var err error
	if out.state, err = w.typeRef(named(w.mm.Decl.GetStateType())); err != nil {
		return out, err
	}
	outcome, err := w.typeRef(named(w.mm.Decl.GetOutcomeType()))
	if err != nil {
		return out, err
	}
	// The event carries a step's outcome and state. Its facts are left out: a monitor that reads them is
	// not exported.
	w.step = ptype{p: "tStep", step: true}
	fmt.Fprintf(&w.decls, "type tStep = (f_outcome: %s, f_state: %s);\n", outcome.p, out.state.p)
	if out.mu, err = w.typeRef(mo.GetState()); err != nil {
		return out, err
	}
	first, err := w.s.in.Eval(mo.GetInitial())
	if err != nil {
		return out, err
	}
	if out.initial, err = w.value(first, out.mu); err != nil {
		return out, err
	}
	next, err := w.function(mo.GetNext(), mo.GetPosition())
	if err != nil {
		return out, err
	}
	violated, err := w.function(mo.GetViolated(), mo.GetPosition())
	if err != nil {
		return out, err
	}
	out.next, out.violated = next.name, violated.name
	switch e := mo.GetEvaluate().(type) {
	case *umpirespb.Monitor_EveryStep:
	case *umpirespb.Monitor_After:
		after, err := w.function(e.After, mo.GetPosition())
		if err != nil {
			return out, err
		}
		out.read = after.name + "(e.after)"
	default:
		return out, w.unsupported(mo.GetPosition(), "the monitor %s, which is read at the machine's ends: the ends are not exported to P", mo.GetName())
	}
	return out, nil
}

// specs writes the two spec machines and the events they observe.
func (m pmonitor) specs() string {
	advance := fmt.Sprintf("      mu = %s(mu, e.before, e.after);\n      steps = steps + 1;\n", m.next)
	return fmt.Sprintf("event eStep: (before: %s, after: tStep);\nevent eReset;\nevent eTraceEnd: int;\n\n", m.state.p) +
		fmt.Sprintf("// The monitor: it is violated where its verdict is read and its state violates it.\n"+
			"spec AuthoredMonitor observes eStep, eReset {\n  var mu: %s;\n  var steps: int;\n  start state Watching {\n"+
			"    entry { mu = %s; steps = 0; }\n    on eReset do { mu = %s; steps = 0; }\n"+
			"    on eStep do (e: (before: %s, after: tStep)) {\n%s"+
			"      if (%s) { assert !%s(mu), format(\"the monitor %s is violated at step {0}\", steps); }\n    }\n  }\n}\n\n",
			m.mu.p, m.initial, m.initial, m.state.p, advance, m.read, m.violated, m.name) +
		fmt.Sprintf("// The same monitor, noting where it is first violated: at a trace's end that must be where Go's is.\n"+
			"spec Agreement observes eStep, eReset, eTraceEnd {\n  var mu: %s;\n  var steps: int;\n  var violatedAt: int;\n  start state Watching {\n"+
			"    entry { mu = %s; steps = 0; violatedAt = 0; }\n    on eReset do { mu = %s; steps = 0; violatedAt = 0; }\n"+
			"    on eStep do (e: (before: %s, after: tStep)) {\n%s"+
			"      if (violatedAt == 0 && %s && %s(mu)) { violatedAt = steps; }\n    }\n"+
			"    on eTraceEnd do (expected: int) {\n"+
			"      assert violatedAt == expected, format(\"P's monitor is first violated at step {0}, and Go's at step {1}, 0 being never\", violatedAt, expected);\n    }\n  }\n}\n\n",
			m.mu.p, m.initial, m.initial, m.state.p, advance, m.read, m.violated)
}

// announce writes one trace as a P function that announces its steps, and then where Go's monitor is
// first violated on it.
func (w *pwriter) announce(name string, tr ptrace, state ptype) (string, error) {
	var events strings.Builder
	for _, st := range tr.steps {
		before, err := w.value(st.before, state)
		if err != nil {
			return "", err
		}
		after, err := w.value(st.after, w.step)
		if err != nil {
			return "", err
		}
		fmt.Fprintf(&events, "  announce eStep, (before = %s, after = %s);\n", before, after)
	}
	return fmt.Sprintf("fun %s(expected: int) {\n  announce eReset;\n%s  announce eTraceEnd, expected;\n}\n", name, events.String()), nil
}

// PMonitor exports one monitor of one machine to P, with every event trace of the machine within a
// depth and what Go's monitor says of each. A monitor that reads what P is not given is an
// UnsupportedError.
func (s *Slice) PMonitor(machine, monitor string, depth int) (*PExport, error) {
	mm := s.machines[machine]
	if mm == nil {
		return nil, fmt.Errorf("the Model has no machine %s", machine)
	}
	k := slices.IndexFunc(mm.Monitors, func(mo *umpirespb.Monitor) bool { return mo.GetName() == monitor })
	if k < 0 {
		return nil, fmt.Errorf("%s names no monitor %s", machine, monitor)
	}
	w := &pwriter{s: s, mm: mm, typeNames: map[string]string{}, functions: map[string]pfunction{}}
	written, err := w.monitor(mm.Monitors[k])
	if err != nil {
		return nil, err
	}
	traces, err := s.traces(mm, k, depth)
	if err != nil {
		return nil, err
	}
	x := &PExport{Model: s.Name, Project: "UmpireMonitor", Machine: machine, Monitor: monitor, Depth: depth, Traces: len(traces)}
	var drivers strings.Builder
	driver := func(machine, test, spec string, calls []string) {
		fmt.Fprintf(&drivers, "machine %s { start state Init { entry {\n  %s\n} } }\ntest %s [main=%s]: assert %s in { %s };\n",
			machine, strings.Join(calls, "\n  "), test, machine, spec, machine)
	}
	var all, accepted []string
	for i, tr := range traces {
		name := fmt.Sprintf("trace%04d", i)
		announced, err := w.announce(name, tr, written.state)
		if err != nil {
			return nil, err
		}
		drivers.WriteString(announced)
		call := fmt.Sprintf("%s(%d);", name, tr.violated)
		all = append(all, call)
		if tr.violated == 0 {
			accepted = append(accepted, call)
			continue
		}
		x.Rejected++
		if x.Rejected <= maxRejectedRuns {
			test := fmt.Sprintf("tcRejected%04d", i)
			driver(fmt.Sprintf("Rejected%04d", i), test, "AuthoredMonitor", []string{call})
			x.Tests = append(x.Tests, PTest{Name: test, Violated: tr.violated})
		}
	}
	x.Accepted = len(accepted)
	driver("EveryTrace", "tcAgreement", "Agreement", all)
	driver("AcceptedTraces", "tcAccepted", "AuthoredMonitor", accepted)
	x.Tests = append(x.Tests, PTest{Name: "tcAgreement"}, PTest{Name: "tcAccepted"})
	if len(accepted) > 0 {
		// A trace Go's monitor holds on, announced as violated at its first step: the agreement must fail
		// on it, or it compares nothing.
		driver("WrongExpectation", "tcControlWrongExpectation", "Agreement", []string{strings.Replace(accepted[0], "(0);", "(1);", 1)})
		x.Tests = append(x.Tests, PTest{Name: "tcControlWrongExpectation", Control: true})
	}
	// The types the traces' values need were declared as the traces were written, so the declarations
	// are complete only now.
	x.Text = fmt.Sprintf("// Written by model/backends from %s. Do not edit.\n// The monitor %s of the machine %s, and its event traces within %d steps.\n\n%s%s%s",
		s.Model.GetSource(), monitor, machine, depth, w.decls.String(), written.specs(), drivers.String())
	return x, nil
}

// Agreement compares what P's checker reported of each test case with what Go's monitor says it must
// do. It gives the monitor's agreement, what the checker covered, and that no refinement is claimed.
func (x *PExport) Agreement(results map[string]PResult) []Receipt {
	subject := x.Machine + "." + x.Monitor
	r := Receipt{Backend: pBackend, Model: x.Model, Claim: MonitorAgreement, Subject: subject, Traces: x.Traces, Accepted: x.Accepted, Rejected: x.Rejected}
	var d differences
	ran, alone := 0, 0
	for _, test := range x.Tests {
		got, ok := results[test.Name]
		switch {
		case !ok || !got.Ran:
			d.add("%s: P's checker did not run it", test.Name)
			continue
		case test.Control && got.Bugs == 0:
			d.add("%s: a trace Go's monitor holds on was announced as violated, and P's agreement did not fail", test.Name)
		case test.Control:
		case test.Violated == 0 && got.Bugs > 0:
			d.add("%s: P reports %q, and Go's monitor says no trace of it fails", test.Name, got.Message)
		case test.Violated > 0 && got.Bugs == 0:
			d.add("%s: P finds no violation, and Go's monitor is violated at step %d", test.Name, test.Violated)
		case test.Violated > 0 && !strings.Contains(got.Message, fmt.Sprintf("the monitor %s is violated at step %d", x.Monitor, test.Violated)):
			d.add("%s: P reports %q, and Go's monitor is violated at step %d", test.Name, got.Message, test.Violated)
		default:
		}
		ran++
		if test.Violated > 0 {
			alone++
		}
	}
	r.Explanation = fmt.Sprintf("every one of %d event traces of %s within %d steps: Go's monitor holds on %d and is first violated on %d, each at the step P's is",
		x.Traces, x.Machine, x.Depth, x.Accepted, x.Rejected)
	covered := Receipt{Backend: pBackend, Model: x.Model, Claim: CheckerCoverage, Subject: subject, Kind: Covered, Traces: x.Traces,
		Explanation: fmt.Sprintf("P's checker ran %d test cases, one schedule each: a driver that only announces events has one schedule. "+
			"All %d traces ran against the monitor as it notes its first violation, the %d accepted ones against its assertion in one case, and %d rejected ones against its assertion one by one. "+
			"The traces are the paths of %s within %d steps and no further; the machine itself is not exported to P",
			ran, x.Traces, x.Accepted, alone, x.Machine, x.Depth)}
	refinement := Receipt{Backend: pBackend, Model: x.Model, Claim: ModuleRefinement, Subject: subject, Kind: Unsupported,
		Explanation: "no P module is exported and no refinement between P modules is checked: the export is one event monitor and the traces it reads"}
	return []Receipt{d.conclude(r), covered, refinement}
}

var (
	bugs      = regexp.MustCompile(`Found (\d+) bugs?\.`)
	assertion = regexp.MustCompile(`<ErrorLog> (.*)`)
)

// RunP compiles an export with P and runs each of its test cases through P's checker, one schedule
// each, and gives the receipts of the comparison with Go's monitor.
func RunP(ctx context.Context, t Tool, x *PExport, dir string) ([]Receipt, error) {
	source := filepath.Join(dir, x.Project+".p")
	if err := os.WriteFile(source, []byte(x.Text), 0o644); err != nil {
		return nil, err
	}
	if _, err := t.run(ctx, dir, "compile", "-pf", source, "-pn", x.Project, "-o", filepath.Join(dir, "out")); err != nil {
		return nil, err
	}
	dll := filepath.Join(dir, "out", "PChecker", "net8.0", x.Project+".dll")
	if _, err := os.Stat(dll); err != nil {
		return nil, fmt.Errorf("p compile wrote no %s: %w", dll, err)
	}
	results := map[string]PResult{}
	for _, test := range x.Tests {
		reports := filepath.Join(dir, "check", test.Name)
		// The checker exits with an error where it finds a bug, which is a result here.
		out, err := t.run(ctx, dir, "check", dll, "-tc", test.Name, "-s", "1", "--seed", "1", "-o", reports)
		var exit interface{ ExitCode() int }
		if err != nil && !errors.As(err, &exit) {
			return nil, err
		}
		// A test case's name selects every case it is a prefix of, and one that selects none runs none.
		found := bugs.FindSubmatch(out)
		if found == nil || strings.Count(string(out), "Test case :: ") != 1 || !strings.Contains(string(out), "Test case :: "+test.Name+"\n") {
			results[test.Name] = PResult{}
			continue
		}
		n, _ := strconv.Atoi(string(found[1]))
		res := PResult{Ran: true, Bugs: n}
		if n > 0 {
			logs, _ := filepath.Glob(filepath.Join(reports, "BugFinding", "*_0_0.txt"))
			for _, log := range logs {
				text, err := os.ReadFile(log)
				if err != nil {
					return nil, err
				}
				if m := assertion.FindSubmatch(text); m != nil {
					res.Message = string(m[1])
				}
			}
		}
		results[test.Name] = res
	}
	return x.Agreement(results), nil
}
