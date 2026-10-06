package ir

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	umpirespb "go.temporal.io/server/api/umpire/v1"
	"go.temporal.io/server/tools/umpire/interp"
	"go.temporal.io/server/tools/umpire/realization"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// Validate checks that every name the Model uses is declared, with the arity it is used at, and that
// every variable is bound where it is read. It reports every problem, each at the position the front
// end gave the node, rather than the first.
// It also rejects the rest of model/SEMANTICS.md's Admission list: an unknown version or a
// construct of no known kind, crossed types, duplicate keys, catalogs that are not finite, bounds,
// misused channels, readings without a refinement, recursion, and a Query total that is not the
// Query's static combination count.
// A realization is checked as a whole of its own: what it names it declares, once, and its commands
// depend on each other without a cycle.
func Validate(m *umpirespb.Model) error {
	v := newValidator(m)
	if m.GetVersion() != 0 {
		v.errs = append(v.errs, &interp.Error{Position: m.GetSource(),
			Message: fmt.Sprintf("version %d is not a version this reader knows", m.GetVersion())})
	}
	for _, t := range m.GetTypes() {
		v.typeDecl(t)
	}
	for _, f := range m.GetFunctions() {
		v.function(f)
	}
	for _, f := range m.GetFunctions() {
		v.recursion(f)
	}
	for _, a := range m.GetActions() {
		v.action(a)
	}
	for _, mm := range m.GetMachines() {
		v.machine(mm)
	}
	for _, c := range m.GetChannels() {
		v.channel(c)
	}
	for _, mo := range m.GetMonitors() {
		v.monitor(mo)
	}
	for _, a := range m.GetAssumptions() {
		v.actionRefs(a.GetPosition(), a.GetFair())
	}
	for _, c := range m.GetCompositions() {
		v.composition(c)
	}
	for _, p := range m.GetProperties() {
		v.property(p)
	}
	for _, s := range m.GetScenarios() {
		v.scenario(s)
	}
	for _, q := range m.GetQueries() {
		v.query(q)
	}
	for _, p := range m.GetProgress() {
		v.progress(p)
	}
	for _, r := range m.GetRealizations() {
		realization.Admit(admission{v}, r)
	}
	v.catalogs(m)
	if len(v.errs) == 0 {
		composed := v.composedClasses(m)
		v.schedules(m, composed)
		v.selectors(m, composed)
		v.identities(m)
		v.totals(m)
	}
	return errors.Join(v.errs...)
}

// newValidator indexes the Model's declarations by the keys the IR names them by, and reports two
// declarations under one key.
func newValidator(m *umpirespb.Model) *validator {
	v := &validator{declared: map[string]bool{}, types: map[string]*umpirespb.Type{}, functions: map[string]*umpirespb.Function{},
		actions: map[string]*umpirespb.Action{}, channels: map[string]*umpirespb.Channel{},
		monitors: map[string]*umpirespb.Monitor{}, assumptions: map[string]bool{}, holes: map[string]bool{},
		machines: map[string]*umpirespb.Machine{}, compositions: map[string]*umpirespb.Composition{},
		properties: map[claim]bool{}, scenarios: map[claim]bool{}, calls: map[string][]string{},
		in: interp.NewInterpreter(m), returning: map[string]bool{}}
	for _, t := range m.GetTypes() {
		v.once(t.GetPosition(), "types named", t.GetName())
		v.types[t.GetName()] = t
	}
	for _, f := range m.GetFunctions() {
		v.once(f.GetPosition(), "functions named", f.GetName())
		v.functions[f.GetName()] = f
	}
	for _, a := range m.GetActions() {
		v.once(a.GetPosition(), "actions with id", a.GetId())
		v.actions[a.GetId()] = a
	}
	for _, c := range m.GetChannels() {
		v.once(c.GetPosition(), "channels with id", c.GetId())
		v.channels[c.GetId()] = c
	}
	for _, mo := range m.GetMonitors() {
		if !v.once(mo.GetPosition(), "monitors with id", mo.GetId()) {
			v.once(mo.GetPosition(), "monitors named", mo.GetName())
		}
		v.monitors[mo.GetId()] = mo
	}
	for _, a := range m.GetAssumptions() {
		v.once(a.GetPosition(), "assumptions with id", a.GetId())
		v.assumptions[a.GetId()] = true
	}
	for _, h := range m.GetHoles() {
		v.once(h.GetPosition(), "holes with id", h.GetId())
		v.holes[h.GetId()] = true
	}
	v.declareChecks(m)
	return v
}

func (v *validator) declareChecks(m *umpirespb.Model) {
	for _, mm := range m.GetMachines() {
		v.once(mm.GetPosition(), "machines named", mm.GetName())
		v.machines[mm.GetName()] = mm
	}
	for _, c := range m.GetCompositions() {
		v.once(c.GetPosition(), "compositions named", c.GetName())
		if v.machines[c.GetName()] != nil {
			v.report(c.GetPosition(), "a machine and a composition are both named %s", c.GetName())
		}
		v.compositions[c.GetName()] = c
	}
	for _, p := range m.GetProperties() {
		v.once(p.GetPosition(), "Properties named", p.GetName(), "on", p.GetMachine())
		v.properties[claim{p.GetMachine(), p.GetName()}] = true
	}
	for _, s := range m.GetScenarios() {
		v.once(s.GetPosition(), "Scenarios named", s.GetName(), "on", s.GetMachine())
		v.scenarios[claim{s.GetMachine(), s.GetName()}] = true
	}
	for _, q := range m.GetQueries() {
		v.once(q.GetPosition(), "Queries named", q.GetName())
	}
	for _, p := range m.GetProgress() {
		v.once(p.GetPosition(), "progress claims named", p.GetName(), "on", p.GetMachine())
	}
}

func (v *validator) function(f *umpirespb.Function) {
	scope := map[string]bool{}
	for _, p := range f.GetParams() {
		v.typeRef(p.GetType(), f.GetPosition())
		if scope[p.GetName()] {
			v.report(f.GetPosition(), "%s has two parameters named %s", f.GetName(), p.GetName())
		}
		scope[p.GetName()] = true
	}
	v.caller = f.GetName()
	if f.GetRequires() != nil {
		v.expr(f.GetRequires(), scope)
	}
	v.expr(f.GetBody(), scope)
	v.caller = ""
}

// recursion reports a function that calls itself, naming the shortest chain of others it does so
// through.
func (v *validator) recursion(f *umpirespb.Function) {
	name := f.GetName()
	// from records, for each function reached, the one whose call reached it first.
	from := map[string]string{}
	queue := []string{name}
	for len(queue) > 0 {
		at := queue[0]
		queue = queue[1:]
		for _, callee := range v.calls[at] {
			if callee == name {
				v.cycle(f, at, from)
				return
			}
			if _, seen := from[callee]; !seen {
				from[callee] = at
				queue = append(queue, callee)
			}
		}
	}
}

func (v *validator) cycle(f *umpirespb.Function, last string, from map[string]string) {
	var through []string
	for at := last; at != f.GetName(); at = from[at] {
		through = append([]string{at}, through...)
	}
	if len(through) == 0 {
		v.report(f.GetPosition(), "%s calls itself", f.GetName())
		return
	}
	v.report(f.GetPosition(), "%s calls itself through %s", f.GetName(), strings.Join(through, ", "))
}

func (v *validator) action(a *umpirespb.Action) {
	for _, p := range a.GetInputs() {
		v.typeRef(p.GetType(), a.GetPosition())
	}
	v.examples(a)
	for _, use := range []struct{ verb, channel string }{{"delivers", a.GetDelivers()}, {"loses", a.GetLoses()}} {
		c, ok := v.channels[use.channel]
		switch {
		case use.channel == "":
		case !ok:
			v.report(a.GetPosition(), "no channel %s", use.channel)
		case len(a.GetInputs()) != 1 || !proto.Equal(a.GetInputs()[0].GetType(), c.GetMessage()):
			v.report(a.GetPosition(), "%s %s %s, so it takes one input of %s", a.GetId(), use.verb, c.GetId(), Spell(c.GetMessage()))
		default:
		}
	}
	if c, ok := v.channels[a.GetLoses()]; ok && !c.GetLossy() {
		v.report(a.GetPosition(), "%s loses channel %s, which is not lossy", a.GetId(), c.GetId())
	}
	if a.GetDelivers() != "" && a.GetLoses() != "" {
		v.report(a.GetPosition(), "%s %s; an action does one", a.GetId(), interp.BothRoles(a))
	}
}

// examples checks an action's Abstraction Claims: an example is of one class of a one-input action,
// so its value is a member of that input's type.
func (v *validator) examples(a *umpirespb.Action) {
	for _, ex := range a.GetExamples() {
		switch before := len(v.errs); {
		case len(a.GetInputs()) != 1:
			v.report(a.GetPosition(), "%s gives an example and takes %d inputs; an example is of one class of a one-input action",
				a.GetId(), len(a.GetInputs()))
		case ex.GetValue().GetKind() == nil:
			v.report(a.GetPosition(), "%s gives an example of no value", a.GetId())
		default:
			v.value(ex.GetValue(), a.GetPosition())
			if len(v.errs) != before {
				continue
			}
			if x := v.in.Literal(ex.GetValue()); !v.in.Conforms(x, a.GetInputs()[0].GetType()) {
				v.report(a.GetPosition(), "%s gives an example of %s, which is no %s", a.GetId(), x.Key(), Spell(a.GetInputs()[0].GetType()))
			}
		}
	}
}

func (v *validator) actionRefs(at *umpirespb.Position, ids []string) {
	for _, id := range ids {
		if _, ok := v.actions[id]; !ok {
			v.report(at, "no action %s", id)
		}
	}
}

func (v *validator) assumptionRefs(at *umpirespb.Position, ids []string) {
	for _, id := range ids {
		if !v.assumptions[id] {
			v.report(at, "no assumption %s", id)
		}
	}
}

type validator struct {
	// declared holds each kind of declaration's keys, as once spells them.
	declared     map[string]bool
	types        map[string]*umpirespb.Type
	functions    map[string]*umpirespb.Function
	actions      map[string]*umpirespb.Action
	channels     map[string]*umpirespb.Channel
	monitors     map[string]*umpirespb.Monitor
	assumptions  map[string]bool
	holes        map[string]bool
	machines     map[string]*umpirespb.Machine
	compositions map[string]*umpirespb.Composition
	properties   map[claim]bool
	scenarios    map[claim]bool
	// calls holds, for each function, the functions its body and precondition call, in the order they
	// are first called; caller is the function whose expressions are being checked.
	calls  map[string][]string
	caller string
	errs   []error
	// in evaluates what admission reads as values: selectors' inputs and Scenarios' starts.
	in *interp.Interpreter
	// returning holds the step functions whose results have been checked.
	returning map[string]bool
}

// admission is the validator as realization admission reads it, an Admitter. It is a type of its
// own so that the validator's methods keep their names.
type admission struct{ v *validator }

func (a admission) Once(at *umpirespb.Position, words ...string) bool { return a.v.once(at, words...) }

func (a admission) Report(at *umpirespb.Position, format string, args ...any) {
	a.v.report(at, format, args...)
}

func (a admission) Errors() int { return len(a.v.errs) }

func (a admission) ActionClass(owner string, mm *umpirespb.Machine, c *umpirespb.ActionClass, at *umpirespb.Position) {
	a.v.actionClass(owner, mm, c, at)
}

func (a admission) ClassKey(c *umpirespb.ActionClass) string {
	return interp.ClassKey(a.v.in, a.v.actions, c)
}

func (a admission) Machine(name string) (*umpirespb.Machine, bool) {
	mm, ok := a.v.machines[name]
	return mm, ok
}

func (a admission) Channel(id string) bool {
	_, ok := a.v.channels[id]
	return ok
}

// once reports a declaration whose key an earlier one of its kind took, and is whether it did. The
// key's words alternate the kind and its parts: "Properties named", name, "on", machine.
func (v *validator) once(at *umpirespb.Position, words ...string) bool {
	key := strings.Join(words, " ")
	if v.declared[key] {
		v.report(at, "two %s", key)
		return true
	}
	v.declared[key] = true
	return false
}

// claim is the key of a Property or a Scenario: the machine or composition it is declared on, and its
// name.
type claim struct{ machine, name string }

// accepts is whether a parameter of type param takes an argument of type arg: the same type, or an
// integer range where the parameter is any integer.
func accepts(param, arg *umpirespb.TypeRef) bool {
	return proto.Equal(param, arg) || (param.GetInt() != nil && arg.GetIntRange() != nil)
}

// Spell writes a type reference as a diagnostic names it.
func Spell(t *umpirespb.TypeRef) string {
	switch r := t.GetRef().(type) {
	case *umpirespb.TypeRef_Named:
		return r.Named
	case *umpirespb.TypeRef_Bool:
		return "Boolean"
	case *umpirespb.TypeRef_IntRange:
		return fmt.Sprintf("%d..%d", r.IntRange.GetLow(), r.IntRange.GetHigh())
	case *umpirespb.TypeRef_List:
		return "List[" + Spell(r.List) + "]"
	case *umpirespb.TypeRef_Int:
		return "Int"
	case *umpirespb.TypeRef_Channel:
		return "channel " + r.Channel
	default:
		return "no type"
	}
}

func (v *validator) report(at *umpirespb.Position, format string, args ...any) {
	v.errs = append(v.errs, interp.ErrorAt(at, format, args...))
}

func (v *validator) typeDecl(t *umpirespb.Type) {
	switch s := t.GetShape().(type) {
	case *umpirespb.Type_Enum:
		cases := map[string]bool{}
		for _, c := range s.Enum.GetCases() {
			if cases[c.GetName()] {
				v.report(t.GetPosition(), "%s has two cases named %s", t.GetName(), c.GetName())
			}
			cases[c.GetName()] = true
			v.fieldNames(t.GetPosition(), c.GetFields(), "case "+c.GetName()+" of "+t.GetName())
			for _, f := range c.GetFields() {
				v.stateField(f.GetType(), t.GetPosition())
			}
		}
	case *umpirespb.Type_Record:
		v.fieldNames(t.GetPosition(), s.Record.GetFields(), t.GetName())
		for _, f := range s.Record.GetFields() {
			v.stateField(f.GetType(), t.GetPosition())
		}
	default:
		v.report(t.GetPosition(), "type %s has no shape", t.GetName())
	}
	fields := map[string]int{}
	for _, c := range v.held(t.GetName()) {
		if fields[c]++; fields[c] == 2 {
			v.report(t.GetPosition(), "%s holds channel %s in two fields", t.GetName(), c)
		}
	}
}

// held is the channels the fields of a type hold, one per field that holds one: a record's fields, or
// the fields of every case of an enum.
func (v *validator) held(typeName string) []string {
	t := v.types[typeName]
	fields := t.GetRecord().GetFields()
	for _, c := range t.GetEnum().GetCases() {
		fields = append(fields, c.GetFields()...)
	}
	var channels []string
	for _, f := range fields {
		if c := f.GetType().GetChannel(); c != "" {
			channels = append(channels, c)
		}
	}
	return channels
}

func (v *validator) fieldNames(at *umpirespb.Position, fields []*umpirespb.Field, owner string) {
	names := map[string]bool{}
	for _, f := range fields {
		if names[f.GetName()] {
			v.report(at, "%s has two fields named %s", owner, f.GetName())
		}
		names[f.GetName()] = true
	}
}

// stateField is a field of a finite type: a named type, the Booleans, or an integer range.
func (v *validator) stateField(t *umpirespb.TypeRef, at *umpirespb.Position) {
	switch t.GetRef().(type) {
	case *umpirespb.TypeRef_Int, *umpirespb.TypeRef_List:
		v.report(at, "a field of a finite type needs a finite type, not an unbounded integer or a list")
	default:
		v.typeRef(t, at)
	}
}

func (v *validator) typeRef(t *umpirespb.TypeRef, at *umpirespb.Position) {
	switch r := t.GetRef().(type) {
	case *umpirespb.TypeRef_Named:
		if _, ok := v.types[r.Named]; !ok && r.Named != interp.StepType {
			v.report(at, "no type %s", r.Named)
		}
	case *umpirespb.TypeRef_List:
		v.typeRef(r.List, at)
	case *umpirespb.TypeRef_IntRange:
		if r.IntRange.GetLow() > r.IntRange.GetHigh() {
			v.report(at, "the range %d..%d is empty", r.IntRange.GetLow(), r.IntRange.GetHigh())
		}
	case *umpirespb.TypeRef_Channel:
		if _, ok := v.channels[r.Channel]; !ok {
			v.report(at, "no channel %s", r.Channel)
		}
	case nil:
		v.report(at, "a type reference names no type")
	default:
	}
}

func (v *validator) fields(typeName, caseName string) (int, bool) {
	if typeName == interp.StepType {
		return len(interp.StepFields), true
	}
	t, ok := v.types[typeName]
	if !ok {
		return 0, false
	}
	switch s := t.GetShape().(type) {
	case *umpirespb.Type_Record:
		return len(s.Record.GetFields()), caseName == ""
	case *umpirespb.Type_Enum:
		for _, c := range s.Enum.GetCases() {
			if c.GetName() == caseName {
				return len(c.GetFields()), true
			}
		}
	default:
	}
	return 0, false
}

func (v *validator) expr(x *umpirespb.Expr, scope map[string]bool) {
	at := x.GetPosition()
	with := func(name string) map[string]bool {
		inner := map[string]bool{name: true}
		for k := range scope {
			inner[k] = true
		}
		return inner
	}
	switch k := x.GetKind().(type) {
	case *umpirespb.Expr_Literal:
		v.value(k.Literal, at)
	case *umpirespb.Expr_Var:
		if !scope[k.Var] {
			v.report(at, "%s is not bound here", k.Var)
		}
	case *umpirespb.Expr_Field:
		v.expr(k.Field.GetBase(), scope)
	case *umpirespb.Expr_Call:
		v.call(k.Call, scope, at)
	case *umpirespb.Expr_Construct:
		v.construct(k.Construct, scope, at)
	case *umpirespb.Expr_Copy:
		v.expr(k.Copy.GetBase(), scope)
		for _, u := range k.Copy.GetUpdates() {
			v.expr(u.GetValue(), scope)
		}
	case *umpirespb.Expr_Unary:
		v.operator(at, "unary operator", umpirespb.Unary_Op_name, int32(k.Unary.GetOp()))
		v.expr(k.Unary.GetOperand(), scope)
	case *umpirespb.Expr_Binary:
		v.operator(at, "binary operator", umpirespb.Binary_Op_name, int32(k.Binary.GetOp()))
		v.expr(k.Binary.GetLeft(), scope)
		v.expr(k.Binary.GetRight(), scope)
	case *umpirespb.Expr_If:
		v.expr(k.If.GetCondition(), scope)
		v.expr(k.If.GetThen(), scope)
		v.expr(k.If.GetElse(), scope)
	case *umpirespb.Expr_Match:
		v.match(k.Match, scope, at)
	case *umpirespb.Expr_Let:
		v.expr(k.Let.GetValue(), scope)
		v.expr(k.Let.GetBody(), with(k.Let.GetName()))
	case *umpirespb.Expr_List:
		for _, item := range k.List.GetItems() {
			v.expr(item, scope)
		}
	case *umpirespb.Expr_Lambda:
		inner := with("")
		for _, p := range k.Lambda.GetParams() {
			inner[p.GetName()] = true
		}
		v.expr(k.Lambda.GetBody(), inner)
	case *umpirespb.Expr_Hole:
		if !v.holes[k.Hole] {
			v.report(at, "no hole %s", k.Hole)
		}
	case *umpirespb.Expr_Inbox:
		v.inbox(k.Inbox, scope, at)
	default:
		v.report(at, "an expression of no known kind")
	}
}

// operator reports an operator that is none of its enum's declared values.
func (v *validator) operator(at *umpirespb.Position, kind string, names map[int32]string, op int32) {
	if !interp.Known(names, op) {
		v.report(at, "no %s %d", kind, op)
	}
}

func (v *validator) inbox(x *umpirespb.Inbox, scope map[string]bool, at *umpirespb.Position) {
	v.operator(at, "inbox operation", umpirespb.Inbox_Op_name, int32(x.GetOp()))
	if _, ok := v.channels[x.GetChannel()]; !ok {
		v.report(at, "no channel %s", x.GetChannel())
	}
	v.expr(x.GetContents(), scope)
	if x.GetOp() == umpirespb.Inbox_OP_SEND {
		v.expr(x.GetMessage(), scope)
	}
}

func (v *validator) call(c *umpirespb.Call, scope map[string]bool, at *umpirespb.Position) {
	f, ok := v.functions[c.GetFunction()]
	switch {
	case !ok:
		v.report(at, "no function %s", c.GetFunction())
	case len(f.GetParams()) != len(c.GetArgs()):
		v.report(at, "%s takes %d arguments, not %d", f.GetName(), len(f.GetParams()), len(c.GetArgs()))
	default:
	}
	if ok && v.caller != "" && !slices.Contains(v.calls[v.caller], f.GetName()) {
		v.calls[v.caller] = append(v.calls[v.caller], f.GetName())
	}
	for _, a := range c.GetArgs() {
		v.expr(a, scope)
	}
}

func (v *validator) construct(c *umpirespb.Construct, scope map[string]bool, at *umpirespb.Position) {
	n, ok := v.fields(c.GetType(), c.GetCase())
	switch {
	case !ok:
		v.report(at, "no type %s %s", c.GetType(), c.GetCase())
	case n != len(c.GetArgs()):
		v.report(at, "%s %s has %d fields, not %d", c.GetType(), c.GetCase(), n, len(c.GetArgs()))
	default:
	}
	// A name is the alternative of a named choice a step record is, so only a step record has one.
	if c.GetChoice() != "" && c.GetType() != interp.StepType {
		v.report(at, "%s names the choice %s, which only a step record can", strings.TrimSpace(c.GetType()+" "+c.GetCase()), c.GetChoice())
	}
	for _, a := range c.GetArgs() {
		v.expr(a, scope)
	}
}

// match checks each case under the names its pattern binds.
func (v *validator) match(m *umpirespb.Match, scope map[string]bool, at *umpirespb.Position) {
	v.expr(m.GetScrutinee(), scope)
	for _, c := range m.GetCases() {
		inner := map[string]bool{}
		for k := range scope {
			inner[k] = true
		}
		v.pattern(c.GetPattern(), inner, at)
		if c.GetGuard() != nil {
			v.expr(c.GetGuard(), inner)
		}
		v.expr(c.GetBody(), inner)
	}
}

func (v *validator) value(x *umpirespb.Value, at *umpirespb.Position) {
	switch k := x.GetKind().(type) {
	case *umpirespb.Value_Enum:
		e := k.Enum
		n, ok := v.fields(e.GetType(), e.GetCase())
		switch {
		case !ok:
			v.report(at, "no case %s of %s", e.GetCase(), e.GetType())
		case n != len(e.GetFields()):
			v.report(at, "%s %s has %d fields, not %d", e.GetType(), e.GetCase(), n, len(e.GetFields()))
		default:
		}
		for _, f := range e.GetFields() {
			v.value(f, at)
		}
	case *umpirespb.Value_Record:
		for _, f := range k.Record.GetFields() {
			v.value(f, at)
		}
	case *umpirespb.Value_List:
		for _, item := range k.List.GetItems() {
			v.value(item, at)
		}
	case nil:
		v.report(at, "a value of no known kind")
	default:
	}
}

func (v *validator) pattern(p *umpirespb.Pattern, scope map[string]bool, at *umpirespb.Position) {
	switch k := p.GetKind().(type) {
	case *umpirespb.Pattern_Bind:
		scope[k.Bind.GetName()] = true
		v.pattern(k.Bind.GetPattern(), scope, at)
	case *umpirespb.Pattern_Literal:
		v.value(k.Literal, at)
	case *umpirespb.Pattern_Case:
		n, ok := v.fields(k.Case.GetType(), k.Case.GetCase())
		switch {
		case !ok:
			v.report(at, "no case %s of %s", k.Case.GetCase(), k.Case.GetType())
		case n != len(k.Case.GetFields()):
			v.report(at, "the pattern for %s %s has %d fields, not %d", k.Case.GetType(), k.Case.GetCase(), len(k.Case.GetFields()), n)
		default:
		}
		for _, f := range k.Case.GetFields() {
			v.pattern(f, scope, at)
		}
	case *umpirespb.Pattern_Alternatives:
		for _, alt := range k.Alternatives.GetPatterns() {
			v.pattern(alt, scope, at)
		}
	case *umpirespb.Pattern_Wildcard:
	default:
		v.report(at, "a pattern of no known kind")
	}
}

func (v *validator) machine(m *umpirespb.Machine) {
	at := m.GetPosition()
	for _, t := range []string{m.GetStateType(), m.GetOutcomeType(), m.GetFactType()} {
		if t != "" {
			v.typeRef(interp.Named(t), at)
		}
	}
	for _, s := range m.GetStarts() {
		v.expr(s, map[string]bool{})
	}
	if m.GetEnds() != nil {
		v.expr(m.GetEnds(), map[string]bool{})
	}
	for _, b := range m.GetSteps() {
		v.step(m, b)
	}
	for _, name := range []string{m.GetEvidence(), m.GetRefines().GetMap(), m.GetRefines().GetVisible(), m.GetRefines().GetVisibleOutcomes()} {
		if name != "" {
			v.arity(at, m.GetName(), name, 1)
		}
	}
	v.actionRefs(at, m.GetUnobservable())
	r := m.GetRefines()
	switch {
	case r.GetProduct() != "" && v.machines[r.GetProduct()] == nil:
		v.report(at, "no machine %s", r.GetProduct())
	case r.GetProduct() == "" && (r.GetVisible() != "" || r.GetVisibleOutcomes() != ""):
		v.report(at, "%s names what a refined machine sees but refines none", m.GetName())
	default:
	}
	v.watched(m)
	v.assumptionRefs(at, m.GetAssumes())
	v.holds(m)
}

// holds checks that a machine delivers and loses only the channels its state holds, and binds a loss
// of each lossy one.
func (v *validator) holds(m *umpirespb.Machine) {
	held := map[string]bool{}
	for _, c := range v.held(m.GetStateType()) {
		held[c] = true
	}
	lost := map[string]bool{}
	for _, b := range m.GetSteps() {
		a := v.actions[b.GetAction()]
		for _, use := range []struct{ kind, channel string }{{"delivery", a.GetDelivers()}, {"loss", a.GetLoses()}} {
			if use.channel != "" && !held[use.channel] {
				v.report(b.GetPosition(), "%s binds a %s of channel %s, which its state %s does not hold",
					m.GetName(), use.kind, use.channel, m.GetStateType())
			}
		}
		lost[a.GetLoses()] = true
	}
	for _, c := range v.held(m.GetStateType()) {
		if v.channels[c].GetLossy() && !lost[c] {
			v.report(m.GetPosition(), "%s holds lossy channel %s but binds no loss of it", m.GetName(), c)
			lost[c] = true
		}
	}
}

func (v *validator) step(m *umpirespb.Machine, b *umpirespb.StepBinding) {
	a, okA := v.actions[b.GetAction()]
	f, okF := v.functions[b.GetFunction()]
	switch {
	case !okA:
		v.report(b.GetPosition(), "no action %s", b.GetAction())
	case !okF:
		v.report(b.GetPosition(), "no function %s", b.GetFunction())
	case len(f.GetParams()) != 1+len(a.GetInputs()):
		v.report(b.GetPosition(), "%s steps %s, which has %d inputs, so it takes the state and %d arguments, not %d",
			f.GetName(), a.GetName(), len(a.GetInputs()), len(a.GetInputs()), len(f.GetParams())-1)
	default:
		args := []*umpirespb.TypeRef{interp.Named(m.GetStateType())}
		for _, in := range a.GetInputs() {
			args = append(args, in.GetType())
		}
		for i, p := range f.GetParams() {
			if !accepts(p.GetType(), args[i]) {
				v.report(b.GetPosition(), "%s steps %s, so its parameter %s takes %s, not %s",
					f.GetName(), a.GetName(), p.GetName(), Spell(args[i]), Spell(p.GetType()))
			}
		}
		if !v.returning[f.GetName()] {
			v.returning[f.GetName()] = true
			v.returnsSteps(f.GetName(), f.GetBody(), map[string]bool{f.GetName(): true})
		}
	}
}

// returnsSteps reports what a step function returns that is certainly no list of steps: a literal or
// a record that is none, an operator's Boolean or integer, a function, or a list holding one of those.
// What only evaluation shows, a parameter, a field or a hole, is left to it.
func (v *validator) returnsSteps(step string, x *umpirespb.Expr, called map[string]bool) {
	what := ""
	switch k := x.GetKind().(type) {
	case *umpirespb.Expr_Literal:
		what = spellSteps(k.Literal)
	case *umpirespb.Expr_Construct:
		if k.Construct.GetType() != interp.StepType {
			what = "a " + k.Construct.GetType()
		}
	case *umpirespb.Expr_List:
		for _, item := range k.List.GetItems() {
			if c := item.GetConstruct(); c != nil && c.GetType() != interp.StepType {
				what = "a list holding a " + c.GetType()
			} else if l := item.GetLiteral(); l != nil && l.GetRecord().GetType() != interp.StepType {
				what = "a list holding " + spellValue(l)
			}
		}
	case *umpirespb.Expr_If:
		v.returnsSteps(step, k.If.GetThen(), called)
		v.returnsSteps(step, k.If.GetElse(), called)
	case *umpirespb.Expr_Match:
		for _, c := range k.Match.GetCases() {
			v.returnsSteps(step, c.GetBody(), called)
		}
	case *umpirespb.Expr_Let:
		v.returnsSteps(step, k.Let.GetBody(), called)
	case *umpirespb.Expr_Binary:
		what = v.binaryReturns(step, k.Binary, called)
	case *umpirespb.Expr_Call:
		if f, ok := v.functions[k.Call.GetFunction()]; ok && !called[f.GetName()] {
			called[f.GetName()] = true
			v.returnsSteps(step, f.GetBody(), called)
		}
	case *umpirespb.Expr_Unary, *umpirespb.Expr_Inbox, *umpirespb.Expr_Copy, *umpirespb.Expr_Lambda:
		what = "no list"
	default:
	}
	if what != "" {
		v.report(x.GetPosition(), "%s returns %s, not a list of steps", step, what)
	}
}

func (v *validator) binaryReturns(step string, b *umpirespb.Binary, called map[string]bool) string {
	switch b.GetOp() {
	case umpirespb.Binary_OP_CONCAT:
		v.returnsSteps(step, b.GetLeft(), called)
		v.returnsSteps(step, b.GetRight(), called)
		return ""
	case umpirespb.Binary_OP_ADD, umpirespb.Binary_OP_SUB:
		return "an integer"
	default:
		return "a Boolean"
	}
}

// spellSteps is how a literal a step function returns is no list of steps, or empty when it is one.
func spellSteps(x *umpirespb.Value) string {
	l := x.GetList()
	if l == nil {
		return spellValue(x)
	}
	for _, item := range l.GetItems() {
		if item.GetRecord().GetType() != interp.StepType {
			return "a list holding " + spellValue(item)
		}
	}
	return ""
}

// spellValue writes the type of a literal as a diagnostic names it.
func spellValue(x *umpirespb.Value) string {
	switch k := x.GetKind().(type) {
	case *umpirespb.Value_Bool:
		return "a Boolean"
	case *umpirespb.Value_Int:
		return "an integer"
	case *umpirespb.Value_Text:
		return "a string"
	case *umpirespb.Value_Enum:
		return "a " + k.Enum.GetType()
	case *umpirespb.Value_Record:
		return "a " + k.Record.GetType()
	case *umpirespb.Value_List:
		return "a list"
	default:
		return "no value"
	}
}

// watched checks that the monitors a machine names read its state.
func (v *validator) watched(m *umpirespb.Machine) {
	for _, id := range m.GetMonitors() {
		mo, ok := v.monitors[id]
		if !ok {
			v.report(m.GetPosition(), "no monitor %s", id)
			continue
		}
		if next := v.functions[mo.GetNext()]; len(next.GetParams()) == 3 && !accepts(next.GetParams()[1].GetType(), interp.Named(m.GetStateType())) {
			v.report(m.GetPosition(), "%s names monitor %s, whose next takes %s, not the state %s",
				m.GetName(), mo.GetName(), Spell(next.GetParams()[1].GetType()), m.GetStateType())
		}
	}
}

var arities = []string{"no", "one", "two", "three"}

// arity checks that owner names a function of n arguments, and reports it at the owner's position.
func (v *validator) arity(at *umpirespb.Position, owner, name string, n int) *umpirespb.Function {
	f, ok := v.functions[name]
	if !ok || len(f.GetParams()) != n {
		if n == 1 {
			v.report(at, "%s: %s is not a function of one argument", owner, name)
		} else {
			v.report(at, "%s: %s is not a function of %s arguments", owner, name, arities[n])
		}
		return nil
	}
	return f
}

func (v *validator) channel(c *umpirespb.Channel) {
	at := c.GetPosition()
	if !interp.Known(umpirespb.Channel_Order_name, int32(c.GetOrder())) {
		v.report(at, "channel %s has no known order", c.GetId())
	}
	v.catalog(at, "channel "+c.GetId(), "message", c.GetMessage())
	if c.GetCapacity() < 1 {
		v.report(at, "channel %s has capacity %d, below 1", c.GetId(), c.GetCapacity())
	}
	if c.GetDuplicates() < 0 {
		v.report(at, "channel %s has %d duplicates, below 0", c.GetId(), c.GetDuplicates())
	}
}

// catalog checks that a monitor's state or a channel's message is of a type with a finite catalog: a
// named type, the Booleans, or an integer range.
func (v *validator) catalog(at *umpirespb.Position, owner, role string, t *umpirespb.TypeRef) {
	switch t.GetRef().(type) {
	case *umpirespb.TypeRef_Int, *umpirespb.TypeRef_List, *umpirespb.TypeRef_Channel:
		v.report(at, "%s needs a %s of a finite type, not %s", owner, role, Spell(t))
	default:
		v.typeRef(t, at)
	}
}

func (v *validator) monitor(mo *umpirespb.Monitor) {
	at := mo.GetPosition()
	switch e := mo.GetEvaluate().(type) {
	case nil:
		v.report(at, "monitor %s has no evaluation point", mo.GetName())
	case *umpirespb.Monitor_After:
		v.arity(at, mo.GetName(), e.After, 1)
	default:
	}
	for _, fn := range []struct {
		role, name string
		arity      int
	}{{"next", mo.GetNext(), 3}, {"violated", mo.GetViolated(), 1}} {
		if f := v.arity(at, mo.GetName(), fn.name, fn.arity); f != nil && !accepts(f.GetParams()[0].GetType(), mo.GetState()) {
			v.report(at, "monitor %s's %s takes %s, not its state %s", mo.GetName(), fn.role, Spell(f.GetParams()[0].GetType()), Spell(mo.GetState()))
		}
	}
	v.catalog(at, "monitor "+mo.GetName(), "state", mo.GetState())
	if mo.GetInitial() != nil {
		v.expr(mo.GetInitial(), map[string]bool{})
	}
}

func (v *validator) composition(c *umpirespb.Composition) {
	at := c.GetPosition()
	v.typeRef(interp.Named(c.GetStateType()), at)
	fields := map[string]*umpirespb.TypeRef{}
	for _, f := range v.types[c.GetStateType()].GetRecord().GetFields() {
		fields[f.GetName()] = f.GetType()
	}
	members := map[string]*umpirespb.Machine{}
	for _, mb := range c.GetMembers() {
		mm, ok := v.machines[mb.GetMachine()]
		field, isField := fields[mb.GetField()]
		switch {
		case !ok:
			v.report(at, "no machine %s", mb.GetMachine())
		case !isField:
			v.report(at, "member %s of %s is no field of %s", mb.GetField(), c.GetName(), c.GetStateType())
		case !proto.Equal(field, interp.Named(mm.GetStateType())):
			v.report(at, "member %s of %s holds %s, not the state %s of %s", mb.GetField(), c.GetName(), Spell(field),
				mm.GetStateType(), mm.GetName())
		default:
		}
		members[mb.GetField()] = mm
		v.replaces(c, mb, mm)
	}
	for _, s := range c.GetSyncs() {
		for _, move := range []*umpirespb.SyncMove{s.GetFirst(), s.GetSecond()} {
			mm, ok := members[move.GetMember()]
			switch {
			case !ok:
				v.report(at, "sync %s of %s has no member %s", s.GetName(), c.GetName(), move.GetMember())
			case mm != nil && !v.binds(mm, move.GetAction()):
				v.report(at, "sync %s of %s: %s binds no action %s", s.GetName(), c.GetName(), mm.GetName(), move.GetAction())
			default:
			}
		}
	}
	if c.GetEnds() != nil {
		v.expr(c.GetEnds(), map[string]bool{})
	}
}

func (v *validator) replaces(c *umpirespb.Composition, mb *umpirespb.Member, mm *umpirespb.Machine) {
	r := mb.GetReplaces()
	switch {
	case r == "":
	case v.machines[r] == nil:
		v.report(c.GetPosition(), "no machine %s", r)
	case mm != nil && mm.GetRefines().GetProduct() != r:
		v.report(c.GetPosition(), "member %s of %s replaces %s, which %s does not refine", mb.GetField(), c.GetName(), r, mm.GetName())
	default:
	}
}

// binds is whether a step of m binds an action of this name.
func (v *validator) binds(m *umpirespb.Machine, action string) bool {
	for _, b := range m.GetSteps() {
		if v.actions[b.GetAction()].GetName() == action {
			return true
		}
	}
	return false
}

func (v *validator) claimed(at *umpirespb.Position, machine string) {
	if v.machines[machine] == nil && v.compositions[machine] == nil {
		v.report(at, "no machine or composition %s", machine)
	}
}

func (v *validator) property(p *umpirespb.Property) {
	at := p.GetPosition()
	v.claimed(at, p.GetMachine())
	n := 1
	if p.GetTransition() {
		n = 2
	}
	owner := p.GetMachine() + "." + p.GetName()
	v.arity(at, owner, p.GetHolds(), n)
	mm := v.machines[p.GetMachine()]
	switch {
	case p.GetWhen() == nil:
	case mm == nil:
		// A composition's Property names composed classes, which selectors checks once they can be
		// listed. What a class is spelled from is checked here.
		if c := p.GetWhenClass(); c != nil {
			v.actionRefs(at, []string{c.GetAction()})
			for _, x := range c.GetInputs() {
				v.value(x, at)
			}
		}
	case p.GetWhenClass() != nil:
		v.actionClass(owner, mm, p.GetWhenClass(), at)
	case !v.binds(mm, p.GetWhenAction()):
		v.report(at, "%s: %s binds no action %s", owner, mm.GetName(), p.GetWhenAction())
	default:
	}
}

func (v *validator) scenario(s *umpirespb.Scenario) {
	at := s.GetPosition()
	v.claimed(at, s.GetMachine())
	if s.GetStart() != nil {
		v.expr(s.GetStart(), map[string]bool{})
	}
	owner := s.GetMachine() + "." + s.GetName()
	if mm := v.machines[s.GetMachine()]; mm != nil {
		for _, c := range s.GetActions() {
			v.actionClass(owner, mm, c, at)
		}
		if len(s.GetKeys()) > 0 {
			v.report(at, "%s: a Scenario of a machine schedules action classes, not keys", owner)
		}
	}
	if v.compositions[s.GetMachine()] != nil && len(s.GetActions()) > 0 {
		v.report(at, "%s: a Scenario of a composition schedules its class keys, not actions", owner)
	}
}

// actionClass checks that a selector names an action its machine binds, with one value of each
// input's type.
func (v *validator) actionClass(owner string, mm *umpirespb.Machine, c *umpirespb.ActionClass, at *umpirespb.Position) {
	v.actionRefs(at, []string{c.GetAction()})
	for _, x := range c.GetInputs() {
		v.value(x, at)
	}
	a, ok := v.actions[c.GetAction()]
	switch {
	case !ok:
	case !slices.ContainsFunc(mm.GetSteps(), func(b *umpirespb.StepBinding) bool { return b.GetAction() == a.GetId() }):
		v.report(at, "%s: %s binds no action %s", owner, mm.GetName(), a.GetId())
	case len(c.GetInputs()) != len(a.GetInputs()):
		v.report(at, "%s: %s takes %d inputs, not %d", owner, a.GetId(), len(a.GetInputs()), len(c.GetInputs()))
	default:
		for i, p := range a.GetInputs() {
			if x := v.in.Literal(c.GetInputs()[i]); !v.in.Conforms(x, p.GetType()) {
				v.report(at, "%s: %s takes a %s for %s, not %s", owner, a.GetId(), Spell(p.GetType()), p.GetName(), x.Key())
			}
		}
	}
}

// catalogs reports a type or a channel whose catalog contains itself, through the fields of types and
// the messages of channels, naming the shortest chain of others it does so through.
func (v *validator) catalogs(m *umpirespb.Model) {
	holds := map[string][]string{}
	var refs func(from string, t *umpirespb.TypeRef)
	refs = func(from string, t *umpirespb.TypeRef) {
		switch r := t.GetRef().(type) {
		case *umpirespb.TypeRef_Named:
			holds[from] = append(holds[from], "type "+r.Named)
		case *umpirespb.TypeRef_Channel:
			holds[from] = append(holds[from], "channel "+r.Channel)
		case *umpirespb.TypeRef_List:
			refs(from, r.List)
		default:
		}
	}
	for _, t := range m.GetTypes() {
		fields := t.GetRecord().GetFields()
		for _, c := range t.GetEnum().GetCases() {
			fields = append(fields, c.GetFields()...)
		}
		for _, f := range fields {
			refs("type "+t.GetName(), f.GetType())
		}
	}
	for _, c := range m.GetChannels() {
		refs("channel "+c.GetId(), c.GetMessage())
	}
	for _, t := range m.GetTypes() {
		v.containsItself(t.GetPosition(), "type "+t.GetName(), holds)
	}
	for _, c := range m.GetChannels() {
		v.containsItself(c.GetPosition(), "channel "+c.GetId(), holds)
	}
}

func (v *validator) containsItself(at *umpirespb.Position, name string, holds map[string][]string) {
	from := map[string]string{}
	queue := []string{name}
	for len(queue) > 0 {
		here := queue[0]
		queue = queue[1:]
		for _, next := range holds[here] {
			if next == name {
				var through []string
				for x := here; x != name; x = from[x] {
					through = append([]string{x}, through...)
				}
				if len(through) == 0 {
					v.report(at, "%s has no finite catalog: it contains itself", name)
				} else {
					v.report(at, "%s has no finite catalog: it contains itself through %s", name, strings.Join(through, ", "))
				}
				return
			}
			if _, seen := from[next]; !seen {
				from[next] = here
				queue = append(queue, next)
			}
		}
	}
}

func (v *validator) progress(p *umpirespb.Progress) {
	at := p.GetPosition()
	if v.machines[p.GetMachine()] == nil {
		v.report(at, "no machine %s", p.GetMachine())
	}
	v.arity(at, p.GetMachine()+"."+p.GetName(), p.GetFrom(), 1)
	v.arity(at, p.GetMachine()+"."+p.GetName(), p.GetTo(), 1)
	v.assumptionRefs(at, p.GetAssumptions())
	if p.GetWithin() < 1 {
		v.report(at, "progress claim %s of %s is within %d steps, fewer than one", p.GetName(), p.GetMachine(), p.GetWithin())
	}
}

func (v *validator) query(q *umpirespb.Query) {
	if !interp.Known(umpirespb.Query_Form_name, int32(q.GetForm())) {
		v.report(q.GetPosition(), "query %s has no known form", q.GetName())
	}
	v.expectedRun(q)
	v.exploration(q)
	p, s := q.GetProperty(), q.GetScenario()
	hasProperty, hasScenario := v.properties[claim{p.GetMachine(), p.GetName()}], v.scenarios[claim{s.GetMachine(), s.GetName()}]
	if !hasProperty {
		v.report(q.GetPosition(), "query %s: no Property %s of %s", q.GetName(), p.GetName(), p.GetMachine())
	}
	if !hasScenario {
		v.report(q.GetPosition(), "query %s: no Scenario %s of %s", q.GetName(), s.GetName(), s.GetMachine())
	}
	through := q.GetThrough() && v.machines[s.GetMachine()].GetRefines().GetProduct() == p.GetMachine()
	if hasProperty && hasScenario && p.GetMachine() != s.GetMachine() && !through {
		v.report(q.GetPosition(), "query %s pairs a Property of %s with a Scenario of %s", q.GetName(), p.GetMachine(), s.GetMachine())
	}
	l := q.GetLimits()
	for _, limit := range []struct {
		name  string
		bound int32
	}{{"steps", l.GetSteps()}, {"actions", l.GetActions()}, {"search", l.GetSearch()}} {
		if limit.bound < 0 {
			v.report(q.GetPosition(), "query %s limits %s to %d, below 0", q.GetName(), limit.name, limit.bound)
		}
	}
}

func (v *validator) expectedRun(q *umpirespb.Query) {
	expected := q.GetExpectedRun()
	if expected == nil {
		return
	}
	at := q.GetPosition()
	// What the Run is expected to end as is declared, never inferred: its Contract verdict, its
	// disposition and its cleanup.
	if expected.GetContract() != umpirespb.RunExpectation_OUTCOME_SATISFIED && expected.GetContract() != umpirespb.RunExpectation_OUTCOME_VIOLATED {
		v.report(at, "query %s expected Run has no supported Contract verdict", q.GetName())
	}
	if !interp.Known(umpirespb.RunExpectation_Disposition_name, int32(expected.GetDisposition())) {
		v.report(at, "query %s expected Run declares no known disposition", q.GetName())
	}
	if !interp.Known(umpirespb.RunExpectation_Cleanup_name, int32(expected.GetCleanup())) {
		v.report(at, "query %s expected Run declares no known cleanup", q.GetName())
	}
	if !interp.Known(umpirespb.RunExpectation_Conformance_name, int32(expected.GetConformance())) {
		v.report(at, "query %s expected Run has no known conformance", q.GetName())
	}
	// Conformance short of conformant names the judge's reason; conformant names none.
	why, conformance := expected.GetConformanceReason(), expected.GetConformance()
	named := why != umpirespb.RunExpectation_REASON_UNSPECIFIED
	short := conformance == umpirespb.RunExpectation_CONFORMANCE_NONCONFORMANT || conformance == umpirespb.RunExpectation_CONFORMANCE_INCONCLUSIVE
	if (named && !interp.Known(umpirespb.RunExpectation_Reason_name, int32(why))) || named != short {
		v.report(at, "query %s expected Run has an invalid conformance reason", q.GetName())
	}
	// An outcome short of satisfied names the judge's reason; a satisfied one names none.
	outcome := func(status umpirespb.RunExpectation_Outcome, reason umpirespb.RunExpectation_Reason) {
		named := reason != umpirespb.RunExpectation_REASON_UNSPECIFIED
		if !interp.Known(umpirespb.RunExpectation_Outcome_name, int32(status)) || (named && !interp.Known(umpirespb.RunExpectation_Reason_name, int32(reason))) ||
			(status == umpirespb.RunExpectation_OUTCOME_SATISFIED) == named {
			v.report(at, "query %s expected Run has an invalid outcome or reason", q.GetName())
		}
	}
	outcome(expected.GetProperty(), expected.GetReason())
	monitors := map[string]bool{}
	for _, id := range v.machines[q.GetScenario().GetMachine()].GetMonitors() {
		monitors[v.monitors[id].GetName()] = true
	}
	for _, monitor := range expected.GetMonitors() {
		if !monitors[monitor.GetName()] {
			v.report(at, "query %s expected Run names unknown or duplicate monitor %s", q.GetName(), monitor.GetName())
		}
		delete(monitors, monitor.GetName())
		outcome(monitor.GetOutcome(), monitor.GetReason())
	}
	if len(monitors) != 0 {
		v.report(at, "query %s expected Run omits monitored claims", q.GetName())
	}
}

// ExpectationID is the stable id of a value of one of an expected Run's enums, as the Case manifest
// and an Assessment spell it: the value's name, lower-cased, without its enum's prefix, so
// REASON_EXPLANATIONS_DISAGREE is explanations_disagree. An unspecified value has no id, and a number
// the enum does not name, as one decoded from newer bytes may be, is unknown(N).
func ExpectationID(value protoreflect.Enum) string {
	if value.Number() == 0 {
		return ""
	}
	return EnumID(value, strings.ToUpper(string(value.Descriptor().Name()))+"_")
}

// EnumID is an enum value's name, lower-cased, without prefix: how an expected Run names a value of
// the IR's enums or of Testpilot's. A number the enum does not name is unknown(N).
func EnumID(value protoreflect.Enum, prefix string) string {
	named := value.Descriptor().Values().ByNumber(value.Number())
	if named == nil {
		return fmt.Sprintf("unknown(%d)", value.Number())
	}
	return strings.ToLower(strings.TrimPrefix(string(named.Name()), prefix))
}
