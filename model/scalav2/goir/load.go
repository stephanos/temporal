package goir

import (
	"errors"
	"fmt"
	"os"
	"regexp"
	"slices"
	"strings"

	modelirspb "go.temporal.io/server/api/modelir/v1"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// Load reads a Model in ProtoJSON, rejects fields the schema does not have, and validates it.
func Load(path string) (*modelirspb.Model, error) {
	encoded, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	m := &modelirspb.Model{}
	if err := (protojson.UnmarshalOptions{}).Unmarshal(encoded, m); err != nil {
		return nil, &Error{Position: path, Message: err.Error()}
	}
	if err := Validate(m); err != nil {
		return nil, err
	}
	return m, nil
}

// Validate checks that every name the Model uses is declared, with the arity it is used at, and that
// every variable is bound where it is read. It reports every problem, each at the position the front
// end gave the node, rather than the first.
// It also rejects the rest of model/scalav2/SEMANTICS.md's Admission list: an unknown version or a
// construct of no known kind, crossed types, duplicate keys, catalogs that are not finite, bounds,
// misused channels, readings without a refinement, and recursion.
// A realization is checked as a whole of its own: what it names it declares, once, and its commands
// depend on each other without a cycle.
func Validate(m *modelirspb.Model) error {
	v := newValidator(m)
	if m.GetVersion() != 0 {
		v.errs = append(v.errs, &Error{Position: m.GetSource(),
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
		v.realization(r)
	}
	v.catalogs(m)
	if len(v.errs) == 0 {
		composed := v.composedClasses(m)
		v.schedules(m, composed)
		v.selectors(m, composed)
		v.identities(m)
	}
	return errors.Join(v.errs...)
}

// newValidator indexes the Model's declarations by the keys the IR names them by, and reports two
// declarations under one key.
func newValidator(m *modelirspb.Model) *validator {
	v := &validator{declared: map[string]bool{}, types: map[string]*modelirspb.Type{}, functions: map[string]*modelirspb.Function{},
		actions: map[string]*modelirspb.Action{}, channels: map[string]*modelirspb.Channel{},
		monitors: map[string]*modelirspb.Monitor{}, assumptions: map[string]bool{}, holes: map[string]bool{},
		machines: map[string]*modelirspb.Machine{}, compositions: map[string]*modelirspb.Composition{},
		properties: map[claim]bool{}, scenarios: map[claim]bool{}, calls: map[string][]string{},
		in: NewInterpreter(m), returning: map[string]bool{}}
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

func (v *validator) declareChecks(m *modelirspb.Model) {
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

func (v *validator) function(f *modelirspb.Function) {
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
func (v *validator) recursion(f *modelirspb.Function) {
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

func (v *validator) cycle(f *modelirspb.Function, last string, from map[string]string) {
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

func (v *validator) action(a *modelirspb.Action) {
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
			v.report(a.GetPosition(), "%s %s %s, so it takes one input of %s", a.GetId(), use.verb, c.GetId(), spell(c.GetMessage()))
		default:
		}
	}
	if c, ok := v.channels[a.GetLoses()]; ok && !c.GetLossy() {
		v.report(a.GetPosition(), "%s loses channel %s, which is not lossy", a.GetId(), c.GetId())
	}
	if a.GetDelivers() != "" && a.GetLoses() != "" {
		v.report(a.GetPosition(), "%s %s; an action does one", a.GetId(), bothRoles(a))
	}
}

// examples checks an action's Abstraction Claims: an example is of one class of a one-input action,
// so its value is a member of that input's type.
func (v *validator) examples(a *modelirspb.Action) {
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
			if x := v.in.literal(ex.GetValue()); !v.in.conforms(x, a.GetInputs()[0].GetType()) {
				v.report(a.GetPosition(), "%s gives an example of %s, which is no %s", a.GetId(), x.Key(), spell(a.GetInputs()[0].GetType()))
			}
		}
	}
}

func (v *validator) actionRefs(at *modelirspb.Position, ids []string) {
	for _, id := range ids {
		if _, ok := v.actions[id]; !ok {
			v.report(at, "no action %s", id)
		}
	}
}

func (v *validator) assumptionRefs(at *modelirspb.Position, ids []string) {
	for _, id := range ids {
		if !v.assumptions[id] {
			v.report(at, "no assumption %s", id)
		}
	}
}

type validator struct {
	// declared holds each kind of declaration's keys, as once spells them.
	declared     map[string]bool
	types        map[string]*modelirspb.Type
	functions    map[string]*modelirspb.Function
	actions      map[string]*modelirspb.Action
	channels     map[string]*modelirspb.Channel
	monitors     map[string]*modelirspb.Monitor
	assumptions  map[string]bool
	holes        map[string]bool
	machines     map[string]*modelirspb.Machine
	compositions map[string]*modelirspb.Composition
	properties   map[claim]bool
	scenarios    map[claim]bool
	// calls holds, for each function, the functions its body and precondition call, in the order they
	// are first called; caller is the function whose expressions are being checked.
	calls  map[string][]string
	caller string
	errs   []error
	// in evaluates what admission reads as values: selectors' inputs and Scenarios' starts.
	in *Interpreter
	// returning holds the step functions whose results have been checked.
	returning map[string]bool
}

// once reports a declaration whose key an earlier one of its kind took, and is whether it did. The
// key's words alternate the kind and its parts: "Properties named", name, "on", machine.
func (v *validator) once(at *modelirspb.Position, words ...string) bool {
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
func accepts(param, arg *modelirspb.TypeRef) bool {
	return proto.Equal(param, arg) || (param.GetInt() != nil && arg.GetIntRange() != nil)
}

// spell writes a type reference as a diagnostic names it.
func spell(t *modelirspb.TypeRef) string {
	switch r := t.GetRef().(type) {
	case *modelirspb.TypeRef_Named:
		return r.Named
	case *modelirspb.TypeRef_Bool:
		return "Boolean"
	case *modelirspb.TypeRef_IntRange:
		return fmt.Sprintf("%d..%d", r.IntRange.GetLow(), r.IntRange.GetHigh())
	case *modelirspb.TypeRef_List:
		return "List[" + spell(r.List) + "]"
	case *modelirspb.TypeRef_Int:
		return "Int"
	case *modelirspb.TypeRef_Channel:
		return "channel " + r.Channel
	default:
		return "no type"
	}
}

// known is whether n is one of an enum's declared values other than its unspecified zero.
func known(names map[int32]string, n int32) bool {
	_, ok := names[n]
	return ok && n != 0
}

func (v *validator) report(at *modelirspb.Position, format string, args ...any) {
	v.errs = append(v.errs, errorAt(at, format, args...))
}

func (v *validator) typeDecl(t *modelirspb.Type) {
	switch s := t.GetShape().(type) {
	case *modelirspb.Type_Enum:
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
	case *modelirspb.Type_Record:
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

func (v *validator) fieldNames(at *modelirspb.Position, fields []*modelirspb.Field, owner string) {
	names := map[string]bool{}
	for _, f := range fields {
		if names[f.GetName()] {
			v.report(at, "%s has two fields named %s", owner, f.GetName())
		}
		names[f.GetName()] = true
	}
}

// stateField is a field of a finite type: a named type, the Booleans, or an integer range.
func (v *validator) stateField(t *modelirspb.TypeRef, at *modelirspb.Position) {
	switch t.GetRef().(type) {
	case *modelirspb.TypeRef_Int, *modelirspb.TypeRef_List:
		v.report(at, "a field of a finite type needs a finite type, not an unbounded integer or a list")
	default:
		v.typeRef(t, at)
	}
}

func (v *validator) typeRef(t *modelirspb.TypeRef, at *modelirspb.Position) {
	switch r := t.GetRef().(type) {
	case *modelirspb.TypeRef_Named:
		if _, ok := v.types[r.Named]; !ok && r.Named != StepType {
			v.report(at, "no type %s", r.Named)
		}
	case *modelirspb.TypeRef_List:
		v.typeRef(r.List, at)
	case *modelirspb.TypeRef_IntRange:
		if r.IntRange.GetLow() > r.IntRange.GetHigh() {
			v.report(at, "the range %d..%d is empty", r.IntRange.GetLow(), r.IntRange.GetHigh())
		}
	case *modelirspb.TypeRef_Channel:
		if _, ok := v.channels[r.Channel]; !ok {
			v.report(at, "no channel %s", r.Channel)
		}
	case nil:
		v.report(at, "a type reference names no type")
	default:
	}
}

func (v *validator) fields(typeName, caseName string) (int, bool) {
	if typeName == StepType {
		return len(stepFields), true
	}
	t, ok := v.types[typeName]
	if !ok {
		return 0, false
	}
	switch s := t.GetShape().(type) {
	case *modelirspb.Type_Record:
		return len(s.Record.GetFields()), caseName == ""
	case *modelirspb.Type_Enum:
		for _, c := range s.Enum.GetCases() {
			if c.GetName() == caseName {
				return len(c.GetFields()), true
			}
		}
	default:
	}
	return 0, false
}

func (v *validator) expr(x *modelirspb.Expr, scope map[string]bool) {
	at := x.GetPosition()
	with := func(name string) map[string]bool {
		inner := map[string]bool{name: true}
		for k := range scope {
			inner[k] = true
		}
		return inner
	}
	switch k := x.GetKind().(type) {
	case *modelirspb.Expr_Literal:
		v.value(k.Literal, at)
	case *modelirspb.Expr_Var:
		if !scope[k.Var] {
			v.report(at, "%s is not bound here", k.Var)
		}
	case *modelirspb.Expr_Field:
		v.expr(k.Field.GetBase(), scope)
	case *modelirspb.Expr_Call:
		v.call(k.Call, scope, at)
	case *modelirspb.Expr_Construct:
		v.construct(k.Construct, scope, at)
	case *modelirspb.Expr_Copy:
		v.expr(k.Copy.GetBase(), scope)
		for _, u := range k.Copy.GetUpdates() {
			v.expr(u.GetValue(), scope)
		}
	case *modelirspb.Expr_Unary:
		v.operator(at, "unary operator", modelirspb.Unary_Op_name, int32(k.Unary.GetOp()))
		v.expr(k.Unary.GetOperand(), scope)
	case *modelirspb.Expr_Binary:
		v.operator(at, "binary operator", modelirspb.Binary_Op_name, int32(k.Binary.GetOp()))
		v.expr(k.Binary.GetLeft(), scope)
		v.expr(k.Binary.GetRight(), scope)
	case *modelirspb.Expr_If:
		v.expr(k.If.GetCondition(), scope)
		v.expr(k.If.GetThen(), scope)
		v.expr(k.If.GetElse(), scope)
	case *modelirspb.Expr_Match:
		v.match(k.Match, scope, at)
	case *modelirspb.Expr_Let:
		v.expr(k.Let.GetValue(), scope)
		v.expr(k.Let.GetBody(), with(k.Let.GetName()))
	case *modelirspb.Expr_List:
		for _, item := range k.List.GetItems() {
			v.expr(item, scope)
		}
	case *modelirspb.Expr_Lambda:
		inner := with("")
		for _, p := range k.Lambda.GetParams() {
			inner[p.GetName()] = true
		}
		v.expr(k.Lambda.GetBody(), inner)
	case *modelirspb.Expr_Hole:
		if !v.holes[k.Hole] {
			v.report(at, "no hole %s", k.Hole)
		}
	case *modelirspb.Expr_Inbox:
		v.inbox(k.Inbox, scope, at)
	default:
		v.report(at, "an expression of no known kind")
	}
}

// operator reports an operator that is none of its enum's declared values.
func (v *validator) operator(at *modelirspb.Position, kind string, names map[int32]string, op int32) {
	if !known(names, op) {
		v.report(at, "no %s %d", kind, op)
	}
}

func (v *validator) inbox(x *modelirspb.Inbox, scope map[string]bool, at *modelirspb.Position) {
	v.operator(at, "inbox operation", modelirspb.Inbox_Op_name, int32(x.GetOp()))
	if _, ok := v.channels[x.GetChannel()]; !ok {
		v.report(at, "no channel %s", x.GetChannel())
	}
	v.expr(x.GetContents(), scope)
	if x.GetOp() == modelirspb.Inbox_OP_SEND {
		v.expr(x.GetMessage(), scope)
	}
}

func (v *validator) call(c *modelirspb.Call, scope map[string]bool, at *modelirspb.Position) {
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

func (v *validator) construct(c *modelirspb.Construct, scope map[string]bool, at *modelirspb.Position) {
	n, ok := v.fields(c.GetType(), c.GetCase())
	switch {
	case !ok:
		v.report(at, "no type %s %s", c.GetType(), c.GetCase())
	case n != len(c.GetArgs()):
		v.report(at, "%s %s has %d fields, not %d", c.GetType(), c.GetCase(), n, len(c.GetArgs()))
	default:
	}
	for _, a := range c.GetArgs() {
		v.expr(a, scope)
	}
}

// match checks each case under the names its pattern binds.
func (v *validator) match(m *modelirspb.Match, scope map[string]bool, at *modelirspb.Position) {
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

func (v *validator) value(x *modelirspb.Value, at *modelirspb.Position) {
	switch k := x.GetKind().(type) {
	case *modelirspb.Value_Enum:
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
	case *modelirspb.Value_Record:
		for _, f := range k.Record.GetFields() {
			v.value(f, at)
		}
	case *modelirspb.Value_List:
		for _, item := range k.List.GetItems() {
			v.value(item, at)
		}
	case nil:
		v.report(at, "a value of no known kind")
	default:
	}
}

func (v *validator) pattern(p *modelirspb.Pattern, scope map[string]bool, at *modelirspb.Position) {
	switch k := p.GetKind().(type) {
	case *modelirspb.Pattern_Bind:
		scope[k.Bind.GetName()] = true
		v.pattern(k.Bind.GetPattern(), scope, at)
	case *modelirspb.Pattern_Literal:
		v.value(k.Literal, at)
	case *modelirspb.Pattern_Case:
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
	case *modelirspb.Pattern_Alternatives:
		for _, alt := range k.Alternatives.GetPatterns() {
			v.pattern(alt, scope, at)
		}
	case *modelirspb.Pattern_Wildcard:
	default:
		v.report(at, "a pattern of no known kind")
	}
}

func (v *validator) machine(m *modelirspb.Machine) {
	at := m.GetPosition()
	for _, t := range []string{m.GetStateType(), m.GetOutcomeType(), m.GetFactType()} {
		if t != "" {
			v.typeRef(named(t), at)
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
func (v *validator) holds(m *modelirspb.Machine) {
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

func (v *validator) step(m *modelirspb.Machine, b *modelirspb.StepBinding) {
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
		args := []*modelirspb.TypeRef{named(m.GetStateType())}
		for _, in := range a.GetInputs() {
			args = append(args, in.GetType())
		}
		for i, p := range f.GetParams() {
			if !accepts(p.GetType(), args[i]) {
				v.report(b.GetPosition(), "%s steps %s, so its parameter %s takes %s, not %s",
					f.GetName(), a.GetName(), p.GetName(), spell(args[i]), spell(p.GetType()))
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
func (v *validator) returnsSteps(step string, x *modelirspb.Expr, called map[string]bool) {
	what := ""
	switch k := x.GetKind().(type) {
	case *modelirspb.Expr_Literal:
		what = spellSteps(k.Literal)
	case *modelirspb.Expr_Construct:
		if k.Construct.GetType() != StepType {
			what = "a " + k.Construct.GetType()
		}
	case *modelirspb.Expr_List:
		for _, item := range k.List.GetItems() {
			if c := item.GetConstruct(); c != nil && c.GetType() != StepType {
				what = "a list holding a " + c.GetType()
			} else if l := item.GetLiteral(); l != nil && l.GetRecord().GetType() != StepType {
				what = "a list holding " + spellValue(l)
			}
		}
	case *modelirspb.Expr_If:
		v.returnsSteps(step, k.If.GetThen(), called)
		v.returnsSteps(step, k.If.GetElse(), called)
	case *modelirspb.Expr_Match:
		for _, c := range k.Match.GetCases() {
			v.returnsSteps(step, c.GetBody(), called)
		}
	case *modelirspb.Expr_Let:
		v.returnsSteps(step, k.Let.GetBody(), called)
	case *modelirspb.Expr_Binary:
		what = v.binaryReturns(step, k.Binary, called)
	case *modelirspb.Expr_Call:
		if f, ok := v.functions[k.Call.GetFunction()]; ok && !called[f.GetName()] {
			called[f.GetName()] = true
			v.returnsSteps(step, f.GetBody(), called)
		}
	case *modelirspb.Expr_Unary, *modelirspb.Expr_Inbox, *modelirspb.Expr_Copy, *modelirspb.Expr_Lambda:
		what = "no list"
	default:
	}
	if what != "" {
		v.report(x.GetPosition(), "%s returns %s, not a list of steps", step, what)
	}
}

func (v *validator) binaryReturns(step string, b *modelirspb.Binary, called map[string]bool) string {
	switch b.GetOp() {
	case modelirspb.Binary_OP_CONCAT:
		v.returnsSteps(step, b.GetLeft(), called)
		v.returnsSteps(step, b.GetRight(), called)
		return ""
	case modelirspb.Binary_OP_ADD, modelirspb.Binary_OP_SUB:
		return "an integer"
	default:
		return "a Boolean"
	}
}

// spellSteps is how a literal a step function returns is no list of steps, or empty when it is one.
func spellSteps(x *modelirspb.Value) string {
	l := x.GetList()
	if l == nil {
		return spellValue(x)
	}
	for _, item := range l.GetItems() {
		if item.GetRecord().GetType() != StepType {
			return "a list holding " + spellValue(item)
		}
	}
	return ""
}

// spellValue writes the type of a literal as a diagnostic names it.
func spellValue(x *modelirspb.Value) string {
	switch k := x.GetKind().(type) {
	case *modelirspb.Value_Bool:
		return "a Boolean"
	case *modelirspb.Value_Int:
		return "an integer"
	case *modelirspb.Value_Text:
		return "a string"
	case *modelirspb.Value_Enum:
		return "a " + k.Enum.GetType()
	case *modelirspb.Value_Record:
		return "a " + k.Record.GetType()
	case *modelirspb.Value_List:
		return "a list"
	default:
		return "no value"
	}
}

// watched checks that the monitors a machine names read its state.
func (v *validator) watched(m *modelirspb.Machine) {
	for _, id := range m.GetMonitors() {
		mo, ok := v.monitors[id]
		if !ok {
			v.report(m.GetPosition(), "no monitor %s", id)
			continue
		}
		if next := v.functions[mo.GetNext()]; len(next.GetParams()) == 3 && !accepts(next.GetParams()[1].GetType(), named(m.GetStateType())) {
			v.report(m.GetPosition(), "%s names monitor %s, whose next takes %s, not the state %s",
				m.GetName(), mo.GetName(), spell(next.GetParams()[1].GetType()), m.GetStateType())
		}
	}
}

var arities = []string{"no", "one", "two", "three"}

// arity checks that owner names a function of n arguments, and reports it at the owner's position.
func (v *validator) arity(at *modelirspb.Position, owner, name string, n int) *modelirspb.Function {
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

func (v *validator) channel(c *modelirspb.Channel) {
	at := c.GetPosition()
	if !known(modelirspb.Channel_Order_name, int32(c.GetOrder())) {
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
func (v *validator) catalog(at *modelirspb.Position, owner, role string, t *modelirspb.TypeRef) {
	switch t.GetRef().(type) {
	case *modelirspb.TypeRef_Int, *modelirspb.TypeRef_List, *modelirspb.TypeRef_Channel:
		v.report(at, "%s needs a %s of a finite type, not %s", owner, role, spell(t))
	default:
		v.typeRef(t, at)
	}
}

func (v *validator) monitor(mo *modelirspb.Monitor) {
	at := mo.GetPosition()
	switch e := mo.GetEvaluate().(type) {
	case nil:
		v.report(at, "monitor %s has no evaluation point", mo.GetName())
	case *modelirspb.Monitor_After:
		v.arity(at, mo.GetName(), e.After, 1)
	default:
	}
	for _, fn := range []struct {
		role, name string
		arity      int
	}{{"next", mo.GetNext(), 3}, {"violated", mo.GetViolated(), 1}} {
		if f := v.arity(at, mo.GetName(), fn.name, fn.arity); f != nil && !accepts(f.GetParams()[0].GetType(), mo.GetState()) {
			v.report(at, "monitor %s's %s takes %s, not its state %s", mo.GetName(), fn.role, spell(f.GetParams()[0].GetType()), spell(mo.GetState()))
		}
	}
	v.catalog(at, "monitor "+mo.GetName(), "state", mo.GetState())
	if mo.GetInitial() != nil {
		v.expr(mo.GetInitial(), map[string]bool{})
	}
}

func (v *validator) composition(c *modelirspb.Composition) {
	at := c.GetPosition()
	v.typeRef(named(c.GetStateType()), at)
	fields := map[string]*modelirspb.TypeRef{}
	for _, f := range v.types[c.GetStateType()].GetRecord().GetFields() {
		fields[f.GetName()] = f.GetType()
	}
	members := map[string]*modelirspb.Machine{}
	for _, mb := range c.GetMembers() {
		mm, ok := v.machines[mb.GetMachine()]
		field, isField := fields[mb.GetField()]
		switch {
		case !ok:
			v.report(at, "no machine %s", mb.GetMachine())
		case !isField:
			v.report(at, "member %s of %s is no field of %s", mb.GetField(), c.GetName(), c.GetStateType())
		case !proto.Equal(field, named(mm.GetStateType())):
			v.report(at, "member %s of %s holds %s, not the state %s of %s", mb.GetField(), c.GetName(), spell(field),
				mm.GetStateType(), mm.GetName())
		default:
		}
		members[mb.GetField()] = mm
		v.replaces(c, mb, mm)
	}
	for _, s := range c.GetSyncs() {
		for _, move := range []*modelirspb.SyncMove{s.GetFirst(), s.GetSecond()} {
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

func (v *validator) replaces(c *modelirspb.Composition, mb *modelirspb.Member, mm *modelirspb.Machine) {
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
func (v *validator) binds(m *modelirspb.Machine, action string) bool {
	for _, b := range m.GetSteps() {
		if v.actions[b.GetAction()].GetName() == action {
			return true
		}
	}
	return false
}

func (v *validator) claimed(at *modelirspb.Position, machine string) {
	if v.machines[machine] == nil && v.compositions[machine] == nil {
		v.report(at, "no machine or composition %s", machine)
	}
}

func (v *validator) property(p *modelirspb.Property) {
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

func (v *validator) scenario(s *modelirspb.Scenario) {
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
func (v *validator) actionClass(owner string, mm *modelirspb.Machine, c *modelirspb.ActionClass, at *modelirspb.Position) {
	v.actionRefs(at, []string{c.GetAction()})
	for _, x := range c.GetInputs() {
		v.value(x, at)
	}
	a, ok := v.actions[c.GetAction()]
	switch {
	case !ok:
	case !slices.ContainsFunc(mm.GetSteps(), func(b *modelirspb.StepBinding) bool { return b.GetAction() == a.GetId() }):
		v.report(at, "%s: %s binds no action %s", owner, mm.GetName(), a.GetId())
	case len(c.GetInputs()) != len(a.GetInputs()):
		v.report(at, "%s: %s takes %d inputs, not %d", owner, a.GetId(), len(a.GetInputs()), len(c.GetInputs()))
	default:
		for i, p := range a.GetInputs() {
			if x := v.in.literal(c.GetInputs()[i]); !v.in.conforms(x, p.GetType()) {
				v.report(at, "%s: %s takes a %s for %s, not %s", owner, a.GetId(), spell(p.GetType()), p.GetName(), x.Key())
			}
		}
	}
}

// schedules evaluates every Scenario's start, which must be a state of its machine or composition,
// and checks a composition's keys against its classes. It runs once the rest of the Model admits, so
// what it evaluates is well formed.
func (v *validator) schedules(m *modelirspb.Model, composed map[string]classKeys) {
	for _, s := range m.GetScenarios() {
		at, owner := s.GetPosition(), s.GetMachine()+"."+s.GetName()
		state := v.machines[s.GetMachine()].GetStateType()
		c := v.compositions[s.GetMachine()]
		if c != nil {
			state = c.GetStateType()
		}
		if s.GetStart() != nil {
			start, err := v.in.Eval(s.GetStart())
			switch {
			case err != nil:
				v.report(at, "%s: its start: %v", owner, err)
			case !v.in.conforms(start, named(state)):
				v.report(at, "%s starts at %s, which is no %s", owner, start.Key(), state)
			default:
			}
		}
		if c == nil || len(s.GetKeys()) == 0 {
			continue
		}
		keys, ok := v.readable(at, owner, composed[c.GetName()])
		if !ok {
			continue
		}
		for _, k := range s.GetKeys() {
			if _, ok := keys[k]; !ok {
				v.report(at, "%s: %s has no class %s", owner, c.GetName(), k)
			}
		}
	}
}

// readable is a composition's class keys for a claim that names some of them, and whether they are
// known. Keys past a ceiling were not listed, which is the claim's error at its own position; two
// classes that share a key were reported where the composition is declared.
func (v *validator) readable(at *modelirspb.Position, owner string, keys classKeys) (map[string]string, bool) {
	var limit *LimitError
	if errors.As(keys.err, &limit) {
		v.errs = append(v.errs, fmt.Errorf("%s: %s: its classes: %w", where(at), owner, keys.err))
	}
	return keys.owners, keys.err == nil
}

// selectors checks the steps a composition's Property is about against the composition's classes:
// one class by its key, and an action by the classes its name begins, a sync's name or
// `<field>_<action>` for a member's own. A member's action a sync takes has no class of its own, so
// neither names it. It runs once the rest of the Model admits, as schedules does.
func (v *validator) selectors(m *modelirspb.Model, composed map[string]classKeys) {
	for _, p := range m.GetProperties() {
		c := v.compositions[p.GetMachine()]
		if c == nil || p.GetWhen() == nil {
			continue
		}
		at, owner := p.GetPosition(), p.GetMachine()+"."+p.GetName()
		keys, ok := v.readable(at, owner, composed[c.GetName()])
		if !ok {
			continue
		}
		switch w := p.GetWhen().(type) {
		case *modelirspb.Property_WhenClass:
			if key := classKey(v.in, v.actions, w.WhenClass); !hasKey(keys, key) {
				v.report(at, "%s: %s has no class %s", owner, c.GetName(), key)
			}
		case *modelirspb.Property_WhenAction:
			if !hasAction(keys, w.WhenAction) {
				v.report(at, "%s: %s has no class of the action %s", owner, c.GetName(), w.WhenAction)
			}
		default:
		}
	}
}

func hasKey(keys map[string]string, key string) bool {
	_, ok := keys[key]
	return ok
}

// hasAction is whether some class of these keys is of the action of this name.
func hasAction(keys map[string]string, action string) bool {
	for key := range keys {
		if actionOf(key) == action {
			return true
		}
	}
	return false
}

// identities lists each machine's state, outcome and fact catalogs and its classes, and refuses two
// values, two classes, or two state and class pairs that share a key. A catalog past the default
// ceilings is left to Build, which refuses it within its own.
func (v *validator) identities(m *modelirspb.Model) {
	report := func(err error) {
		var limit *LimitError
		if err != nil && !errors.As(err, &limit) {
			v.errs = append(v.errs, err)
		}
	}
	for _, mm := range m.GetMachines() {
		// The same counts Build refuses a machine by bound what is listed here.
		if err := v.in.preflight(mm, v.actions); err != nil {
			report(err)
			continue
		}
		var states []Value
		for i, t := range []string{mm.GetStateType(), mm.GetOutcomeType(), mm.GetFactType()} {
			if t == "" {
				continue
			}
			values, err := v.in.Members(named(t))
			report(err)
			if i == 0 {
				states = values
			}
		}
		classes, err := v.in.classes(mm, v.actions)
		report(err)
		if states != nil && err == nil {
			report(rowKeys(mm, states, classes))
		}
	}
}

// classKeys is a composition's class keys, each with the class it names, or why they are not known.
type classKeys struct {
	owners map[string]string
	err    error
}

// composedClasses keys every composition's classes, and reports a composition two of whose classes
// share a key; one past the default ceilings is left to the Scenarios that read its keys.
func (v *validator) composedClasses(m *modelirspb.Model) map[string]classKeys {
	out := map[string]classKeys{}
	for _, c := range m.GetCompositions() {
		owners, err := v.composedKeys(c)
		var limit *LimitError
		if err != nil && !errors.As(err, &limit) {
			v.errs = append(v.errs, err)
		}
		if limit != nil {
			limit.Machine = c.GetName()
		}
		out[c.GetName()] = classKeys{owners: owners, err: err}
	}
	return out
}

// composedKeys is every class key of a composition: `<field>_<class>` for a member's own class, and a
// sync's name followed by the inputs of each of its classes. A member's action a sync names steps
// only with its pair, so it has no class of its own.
// A key two classes spell alike, as `_` in a field or an action name and a sync named like a
// member's class allow, is refused where the composition is declared.
func (v *validator) composedKeys(c *modelirspb.Composition) (map[string]string, error) {
	if err := v.composedCount(c); err != nil {
		return nil, err
	}
	keys := &owned{composition: c, owners: map[string]string{}}
	synced := syncedActions(c)
	members := map[string]*modelirspb.Machine{}
	for _, mb := range c.GetMembers() {
		members[mb.GetField()] = v.machines[mb.GetMachine()]
		if err := v.memberKeys(keys, mb, members[mb.GetField()], synced); err != nil {
			return nil, err
		}
	}
	for _, s := range c.GetSyncs() {
		if err := v.syncKeys(keys, s, members); err != nil {
			return nil, err
		}
	}
	return keys.owners, nil
}

// syncedActions is the member actions a composition's syncs name, each as its member's field and the
// action's name.
func syncedActions(c *modelirspb.Composition) map[[2]string]bool {
	synced := map[[2]string]bool{}
	for _, s := range c.GetSyncs() {
		for _, move := range []*modelirspb.SyncMove{s.GetFirst(), s.GetSecond()} {
			synced[[2]string{move.GetMember(), move.GetAction()}] = true
		}
	}
	return synced
}

// composedCount counts a composition's class keys, its members' classes no sync names and each
// sync's pairs of them, refusing them past the Members ceiling before any is made.
func (v *validator) composedCount(c *modelirspb.Composition) error {
	var n count
	synced := syncedActions(c)
	members := map[string]*modelirspb.Machine{}
	for _, mb := range c.GetMembers() {
		members[mb.GetField()] = v.machines[mb.GetMachine()]
		for _, b := range members[mb.GetField()].GetSteps() {
			a, ok := v.actions[b.GetAction()]
			if !ok {
				return errorAt(b.GetPosition(), "no action %s", b.GetAction())
			}
			if synced[[2]string{mb.GetField(), a.GetName()}] {
				continue
			}
			k, err := v.in.sizeOfProduct(inputFields(a))
			if err != nil {
				return err
			}
			n = n.plus(k)
		}
	}
	for _, s := range c.GetSyncs() {
		first, err := v.actionCount(members[s.GetFirst().GetMember()], s.GetFirst().GetAction())
		if err != nil {
			return err
		}
		second, err := v.actionCount(members[s.GetSecond().GetMember()], s.GetSecond().GetAction())
		if err != nil {
			return err
		}
		n = n.plus(first.times(second))
	}
	return v.in.within("classes", v.in.ceilings.Members, n)
}

// actionCount counts the classes of the action of this name a member binds.
func (v *validator) actionCount(mm *modelirspb.Machine, action string) (count, error) {
	for _, b := range mm.GetSteps() {
		if a := v.actions[b.GetAction()]; a.GetName() == action {
			return v.in.sizeOfProduct(inputFields(a))
		}
	}
	return count{}, nil
}

// owned is a composition's class keys as they are made, each with the class it names.
type owned struct {
	composition *modelirspb.Composition
	owners      map[string]string
}

func (o *owned) own(key, owner string) error {
	if earlier, ok := o.owners[key]; ok && earlier != owner {
		c := o.composition
		return errorAt(c.GetPosition(), "composition %s: %s and %s share the key %q", c.GetName(), earlier, owner, key)
	}
	o.owners[key] = owner
	return nil
}

func (v *validator) memberKeys(keys *owned, mb *modelirspb.Member, mm *modelirspb.Machine, synced map[[2]string]bool) error {
	for _, b := range mm.GetSteps() {
		if synced[[2]string{mb.GetField(), v.actions[b.GetAction()].GetName()}] {
			continue
		}
		classes, err := v.inputKeys(v.actions[b.GetAction()])
		if err != nil {
			return err
		}
		for _, k := range classes {
			class := strings.Join(append([]string{v.actions[b.GetAction()].GetName()}, k...), "-")
			if err := keys.own(mb.GetField()+"_"+class, "member "+mb.GetField()+"'s class "+class); err != nil {
				return err
			}
		}
	}
	return nil
}

func (v *validator) syncKeys(keys *owned, s *modelirspb.Sync, members map[string]*modelirspb.Machine) error {
	first, err := v.syncInputs(members[s.GetFirst().GetMember()], s.GetFirst().GetAction())
	if err != nil {
		return err
	}
	second, err := v.syncInputs(members[s.GetSecond().GetMember()], s.GetSecond().GetAction())
	if err != nil {
		return err
	}
	for _, x := range first {
		for _, y := range second {
			owner := "sync " + s.GetName() + " of " + s.GetFirst().GetMember() + "." + s.GetFirst().GetAction() + " and " +
				s.GetSecond().GetMember() + "." + s.GetSecond().GetAction()
			if len(x)+len(y) > 0 {
				owner += "(" + strings.Join(x, ", ") + "; " + strings.Join(y, ", ") + ")"
			}
			if err := keys.own(strings.Join(append(append([]string{s.GetName()}, x...), y...), "-"), owner); err != nil {
				return err
			}
		}
	}
	return nil
}

// syncInputs is the input keys of every class of the action of this name a member binds.
func (v *validator) syncInputs(mm *modelirspb.Machine, action string) ([][]string, error) {
	for _, b := range mm.GetSteps() {
		if a := v.actions[b.GetAction()]; a.GetName() == action {
			return v.inputKeys(a)
		}
	}
	return nil, nil
}

// inputKeys is, for every class of an action, the keys of its inputs.
func (v *validator) inputKeys(a *modelirspb.Action) ([][]string, error) {
	assignments, err := v.in.product(inputFields(a))
	if err != nil {
		return nil, err
	}
	out := make([][]string, len(assignments))
	for i, inputs := range assignments {
		for _, x := range inputs {
			out[i] = append(out[i], x.Key())
		}
	}
	return out, nil
}

// catalogs reports a type or a channel whose catalog contains itself, through the fields of types and
// the messages of channels, naming the shortest chain of others it does so through.
func (v *validator) catalogs(m *modelirspb.Model) {
	holds := map[string][]string{}
	var refs func(from string, t *modelirspb.TypeRef)
	refs = func(from string, t *modelirspb.TypeRef) {
		switch r := t.GetRef().(type) {
		case *modelirspb.TypeRef_Named:
			holds[from] = append(holds[from], "type "+r.Named)
		case *modelirspb.TypeRef_Channel:
			holds[from] = append(holds[from], "channel "+r.Channel)
		case *modelirspb.TypeRef_List:
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

func (v *validator) containsItself(at *modelirspb.Position, name string, holds map[string][]string) {
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

func (v *validator) progress(p *modelirspb.Progress) {
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

func (v *validator) query(q *modelirspb.Query) {
	if !known(modelirspb.Query_Form_name, int32(q.GetForm())) {
		v.report(q.GetPosition(), "query %s has no known form", q.GetName())
	}
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

// realizing is one realization being admitted: what it declares, by id, and what its commands bind,
// read and perform.
type realizing struct {
	v *validator
	r *modelirspb.Realization
	// label is what scopes the realization's declarations, and owner what a diagnostic calls it: its
	// name, its id where it has no name, or where it was written where it has neither.
	label        string
	owner        string
	roles        map[string]*modelirspb.Role
	learned      map[string]*modelirspb.Learned
	observations map[string]bool
	evidence     map[string]*modelirspb.Evidence
	controls     map[string]bool
	// bound and performed name the command that binds a learned value and that performs a class;
	// read holds the learned values some command reads.
	bound     map[string]string
	performed map[string]string
	read      map[string]bool
	// closed names the command whose read closes an exhaustive kind of evidence.
	closed map[string]string
}

var (
	roleKinds    = map[modelirspb.Role_Kind]string{modelirspb.Role_KIND_ENDPOINT: "an endpoint", modelirspb.Role_KIND_WORKER: "a worker", modelirspb.Role_KIND_TASK_QUEUE: "a task queue", modelirspb.Role_KIND_PARTICIPANT: "a participant"}
	learnedKinds = map[modelirspb.Learned_Kind]string{modelirspb.Learned_KIND_TEXT: "a text", modelirspb.Learned_KIND_HANDLE: "a handle"}
	fieldRoles   = map[modelirspb.EvidenceField_Role]string{modelirspb.EvidenceField_ROLE_OPERATION: "the operation", modelirspb.EvidenceField_ROLE_ATTEMPT: "the attempt", modelirspb.EvidenceField_ROLE_DELIVERY: "the delivery"}
)

// realization checks that a realization names what it declares, declares each thing once, binds each
// learned value and performs each class once, crosses no kind and no correlation, and orders its
// commands without a cycle. It is named by its id and by its name, and one that lacks either, or
// names no machine of the Model, is still read whole: only its classes, which are read against the
// machine, are left unchecked.
func (v *validator) realization(r *modelirspb.Realization) {
	at := r.GetPosition()
	a := &realizing{v: v, r: r, label: r.GetName(), owner: "realization " + r.GetName(), roles: map[string]*modelirspb.Role{},
		learned: map[string]*modelirspb.Learned{}, observations: map[string]bool{}, evidence: map[string]*modelirspb.Evidence{},
		controls: map[string]bool{}, bound: map[string]string{}, performed: map[string]string{}, read: map[string]bool{},
		closed: map[string]string{}}
	switch {
	case r.GetName() != "":
		v.once(at, "realizations named", r.GetName())
	case r.GetId() != "":
		v.report(at, "a realization has no name")
		a.label, a.owner = r.GetId(), "realization "+r.GetId()
	default:
		v.report(at, "a realization has no name")
		a.label, a.owner = fmt.Sprintf("at %s", where(at)), "a realization"
	}
	if r.GetId() == "" {
		v.report(at, "%s has no id", a.owner)
	} else {
		v.once(at, "realizations with id", r.GetId())
	}
	mm, ok := v.machines[r.GetMachine()]
	if !ok {
		a.report(at, "no machine %s", r.GetMachine())
	}
	if r.GetProducer() == "" {
		a.report(at, "it names no producer")
	}
	a.declarations()
	a.correlation()
	for _, s := range r.GetScripts() {
		a.script(mm, s)
	}
	for _, l := range r.GetLearned() {
		switch _, bound := a.bound[l.GetId()]; {
		case l.GetId() == "" || bound:
		case a.read[l.GetId()]:
			a.report(l.GetPosition(), "learned value %s is read and no command binds it", l.GetId())
		default:
			a.report(l.GetPosition(), "learned value %s is neither bound nor read", l.GetId())
		}
	}
	for _, e := range r.GetEvidence() {
		if _, closed := a.closed[e.GetId()]; e.GetExhaustive() && !closed {
			a.report(e.GetPosition(), "evidence %s is exhaustive and no command closes it", e.GetId())
		}
		a.recordedBy(e)
		a.attemptOf(e)
	}
	a.confirms(mm)
	a.held(mm)
}

// attemptOf checks evidence that is the Run's record of an attempt: it names an attempt, counted from
// one, of a script an activity activates, whose activation is a delivery a path can take, and it is
// what a worker reports of an activation, which is how a Run records an attempt. What a worker reports
// of an activation is such a record whatever the realization says of it, so it names its attempt: a
// reader places it by that attempt, and has nothing else to place it by.
func (a *realizing) attemptOf(e *modelirspb.Evidence) {
	of := e.GetRunEvent().GetAttempt()
	if of == nil {
		if e.GetRunEvent().GetKind() == modelirspb.RunEventSource_KIND_DIAGNOSTIC {
			a.report(e.GetPosition(), "evidence %s is what a worker reports of an activation and is declared the record of no attempt: "+
				"a Run records it once the attempt is answered, and the realization says which attempt that is", e.GetId())
		}
		return
	}
	at := of.GetPosition()
	if at.GetFile() == "" {
		at = e.GetPosition()
	}
	if e.GetRunEvent().GetKind() != modelirspb.RunEventSource_KIND_DIAGNOSTIC {
		a.report(at, "evidence %s is the record of an attempt and of no diagnostic: a Run records an attempt as what a worker reports of an activation", e.GetId())
	}
	if of.GetScript() == "" {
		a.report(at, "evidence %s is the record of an attempt of no script", e.GetId())
		return
	}
	for _, s := range a.r.GetScripts() {
		if s.GetId() != of.GetScript() {
			continue
		}
		switch {
		case s.GetActivity() == nil:
			a.report(at, "evidence %s is the record of an attempt of script %s, which no activity activates", e.GetId(), s.GetId())
		case len(s.GetActivity().GetStarts()) == 0:
			a.report(at, "evidence %s is the record of an attempt of script %s, which starts with no delivery: no step of a path is an attempt of it", e.GetId(), s.GetId())
		case of.GetNumber() < 1:
			a.report(at, "evidence %s is the record of attempt %d of script %s; the attempts of an activity are counted from one", e.GetId(), of.GetNumber(), s.GetId())
		default:
		}
		return
	}
	a.report(at, "evidence %s is the record of an attempt of script %s, which the realization does not declare", e.GetId(), of.GetScript())
}

// confirms checks the steps of a path that kinds of evidence name as the ones they confirm: each is a
// step of a class the machine binds, counted from one, which its kind names once and no other kind
// names; and no kind that names steps is exhaustive, or records a fact an exhaustive kind records.
func (a *realizing) confirms(mm *modelirspb.Machine) {
	// An exhaustive kind reports every occurrence of the fact it records, so it is the one kind of
	// that fact: read with a second kind beside it, its silence would say nothing of the steps the
	// second confirms.
	exhaustive := map[string]string{}
	for _, e := range a.r.GetEvidence() {
		if e.GetExhaustive() && e.GetRecords() != "" {
			exhaustive[e.GetRecords()] = e.GetId()
		}
	}
	named := map[string]string{}
	for _, e := range a.r.GetEvidence() {
		switch reporting, reported := exhaustive[e.GetRecords()]; {
		case len(e.GetConfirms()) == 0:
		case e.GetExhaustive():
			a.report(e.GetPosition(), "evidence %s is exhaustive and names the steps it confirms; an exhaustive kind reports every step that records its fact", e.GetId())
		case reported:
			a.report(e.GetPosition(), "evidence %s records %s, every occurrence of which the exhaustive %s reports", e.GetId(), e.GetRecords(), reporting)
		default:
		}
		own := map[string]bool{}
		for _, taking := range e.GetConfirms() {
			at := taking.GetPosition()
			if at.GetFile() == "" {
				at = e.GetPosition()
			}
			if taking.GetStep() == nil {
				a.report(at, "evidence %s confirms a step of no class", e.GetId())
				continue
			}
			if mm == nil {
				continue
			}
			before := len(a.v.errs)
			a.v.actionClass(a.owner+": evidence "+e.GetId(), mm, taking.GetStep(), at)
			if len(a.v.errs) != before {
				continue
			}
			step := fmt.Sprintf("step %d of class %s", taking.GetOccurrence(), classKey(a.v.in, a.v.actions, taking.GetStep()))
			switch other, taken := named[step]; {
			case taking.GetOccurrence() < 1:
				a.report(at, "evidence %s confirms %s; the steps of a class on a path are counted from one", e.GetId(), step)
			case own[step]:
				a.report(at, "evidence %s confirms %s twice", e.GetId(), step)
			case taken:
				a.report(at, "evidence %s and %s both confirm %s; one kind confirms a step", other, e.GetId(), step)
			default:
				named[step], own[step] = e.GetId(), true
			}
		}
	}
}

// recordedBy checks that a Run Event's evidence names a command of a script the realization declares.
func (a *realizing) recordedBy(e *modelirspb.Evidence) {
	source := e.GetRunEvent()
	if source == nil {
		return
	}
	for _, s := range a.r.GetScripts() {
		if s.GetId() != source.GetScript() || s.GetId() == "" {
			continue
		}
		for _, item := range s.GetItems() {
			if item.GetCommand() != nil && item.GetCommand().GetId() == source.GetCommand() {
				return
			}
			for _, p := range item.GetPerforms() {
				if p.GetCommand() != nil && p.GetCommand().GetId() == source.GetCommand() {
					return
				}
			}
		}
		a.report(e.GetPosition(), "evidence %s: no command %s of script %s", e.GetId(), source.GetCommand(), s.GetId())
		return
	}
	a.report(e.GetPosition(), "evidence %s: no script %s", e.GetId(), source.GetScript())
}

func (a *realizing) report(at *modelirspb.Position, format string, args ...any) {
	if at.GetFile() == "" {
		at = a.r.GetPosition()
	}
	a.v.report(at, a.owner+": "+format, args...)
}

// declared reports a declaration with no id, and one whose id an earlier one of its kind took. It is
// whether the declaration can be named.
func (a *realizing) declared(at *modelirspb.Position, kind, plural, id string) bool {
	if id == "" {
		a.report(at, "%s has no id", kind)
		return false
	}
	if at.GetFile() == "" {
		at = a.r.GetPosition()
	}
	return !a.v.once(at, plural+" with id", id, "of realization", a.label)
}

func (a *realizing) declarations() {
	for _, role := range a.r.GetRoles() {
		if a.declared(role.GetPosition(), "a role", "roles", role.GetId()) {
			a.roles[role.GetId()] = role
		}
		if !known(modelirspb.Role_Kind_name, int32(role.GetKind())) {
			a.report(role.GetPosition(), "role %s is of no known kind", role.GetId())
		}
	}
	for _, l := range a.r.GetLearned() {
		if a.declared(l.GetPosition(), "a learned value", "learned values", l.GetId()) {
			a.learned[l.GetId()] = l
		}
		if !known(modelirspb.Learned_Kind_name, int32(l.GetKind())) {
			a.report(l.GetPosition(), "learned value %s is of no known kind", l.GetId())
		}
	}
	for _, o := range a.r.GetObservations() {
		if a.declared(o.GetPosition(), "an observation", "observations", o.GetId()) {
			a.observations[o.GetId()] = true
		}
		if o.GetMessage() == "" {
			a.report(o.GetPosition(), "observation %s names no message", o.GetId())
		}
	}
	for _, e := range a.r.GetEvidence() {
		a.evidenceKind(e)
	}
	for _, c := range a.r.GetControls() {
		if a.declared(c.GetPosition(), "a control", "controls", c.GetId()) {
			a.controls[c.GetId()] = true
		}
		switch k := c.GetKind().(type) {
		case *modelirspb.Control_HoldDelivery:
			if _, ok := a.v.channels[k.HoldDelivery]; !ok {
				a.report(c.GetPosition(), "control %s: no channel %s", c.GetId(), k.HoldDelivery)
			}
		case *modelirspb.Control_HoldDispatched:
			// A run holds a dispatch through the deliveries of a queue, so the control names one.
			switch {
			case k.HoldDispatched.GetStep() == nil:
				a.report(c.GetPosition(), "control %s holds what a step of no class dispatches", c.GetId())
			case c.GetRole() == "":
				a.report(c.GetPosition(), "control %s holds deliveries and names no task-queue role", c.GetId())
			default:
				a.role(c.GetPosition(), "control "+c.GetId(), c.GetRole(), modelirspb.Role_KIND_TASK_QUEUE)
			}
		default:
			a.report(c.GetPosition(), "control %s is of no known kind", c.GetId())
		}
	}
}

// held checks that a control that holds what a step dispatches names a class the machine binds.
func (a *realizing) held(mm *modelirspb.Machine) {
	if mm == nil {
		return
	}
	for _, c := range a.r.GetControls() {
		if step := c.GetHoldDispatched().GetStep(); step != nil {
			at := c.GetPosition()
			if at.GetFile() == "" {
				at = a.r.GetPosition()
			}
			a.v.actionClass(a.owner+": control "+c.GetId(), mm, step, at)
		}
	}
}

func (a *realizing) evidenceKind(e *modelirspb.Evidence) {
	at, id := e.GetPosition(), e.GetId()
	if a.declared(at, "a kind of evidence", "kinds of evidence", id) {
		a.evidence[id] = e
	}
	switch {
	case e.GetRecords() == "":
		a.report(at, "evidence %s names no recorded kind", id)
	case len(e.GetConfirms()) == 0:
		// One kind confirms the step of a path that records a fact. Kinds that name the steps they
		// confirm are told apart by those steps, and may record one fact beside it.
		a.v.once(at, "kinds of evidence recording", e.GetRecords(), "of realization", a.label)
	default:
	}
	if e.GetSource() == "" {
		a.report(at, "evidence %s names no source", id)
	}
	switch {
	case e.GetRunEvent() != nil && e.GetOperation() != "":
		a.report(at, "evidence %s names a field that keys its operation, and a Run Event's key is its source's", id)
	case e.GetRunEvent() == nil && e.GetOperation() == "":
		a.report(at, "evidence %s names no field that keys its operation", id)
	default:
	}
	switch from := e.GetFrom().(type) {
	case *modelirspb.Evidence_History:
		if from.History == "" {
			a.report(at, "evidence %s names no history event", id)
		}
	case *modelirspb.Evidence_Read:
		if from.Read.GetMethod() == "" || from.Read.GetPath() == "" {
			a.report(at, "evidence %s reads no method or no path", id)
		}
	case *modelirspb.Evidence_Single:
		if from.Single.GetMethod() == "" || from.Single.GetPath() == "" {
			a.report(at, "evidence %s reads no method or no path", id)
		}
	case *modelirspb.Evidence_RunEvent:
		a.runEvent(e, from.RunEvent)
	default:
		a.report(at, "evidence %s is recorded nowhere", id)
	}
	if !known(modelirspb.Evidence_Commitment_name, int32(e.GetCommitment())) {
		a.report(at, "evidence %s is of no known commitment", id)
	}
	a.evidenceFields(e)
}

// runEvent checks the Run's own record as a source of evidence: it is of a known kind, its key is the
// run's id or a path of the event's payload, and its guard reads the payload alone. The command it
// names is checked once the scripts are read.
func (a *realizing) runEvent(e *modelirspb.Evidence, source *modelirspb.RunEventSource) {
	at, id := e.GetPosition(), e.GetId()
	if !known(modelirspb.RunEventSource_Kind_name, int32(source.GetKind())) {
		a.report(at, "evidence %s is a Run Event of no known kind", id)
	}
	key := source.GetKey()
	if key.GetRun() == nil && (key.GetPath() == nil || key.GetPath().GetOf().GetProjected() == nil || key.GetPath().GetPath() == "") {
		a.report(at, "evidence %s: a Run Event's key is the run's id or a path of its payload", id)
	}
	if source.GetGuard() != nil {
		if err := GuardProblem(source.GetGuard(), nil); err != nil {
			a.report(at, "evidence %s: its guard %s", id, err)
		}
	}
}

// Shape is the type of the value an operand computes.
type Shape string

const (
	// AnyShape is a value its reader does not type: a path read where no descriptor is.
	AnyShape       Shape = ""
	ConditionShape Shape = "a condition"
	NumberShape    Shape = "a number"
	TextShape      Shape = "a text"
	EnumShape      Shape = "an enum value"
	MessageShape   Shape = "a message"
	SeveralShape   Shape = "several values"
	OtherShape     Shape = "a value that is no text, flag, integer or enum value"
)

// Typed is what an operand computes: its shape and, where a descriptor was read, the enum an enum
// value is of and the message a message is. An enum value written out has its name and no enum.
type Typed struct {
	Shape   Shape
	Enum    protoreflect.EnumDescriptor
	Name    string
	Message protoreflect.MessageDescriptor
}

// Mistype is what is wrong with the types of an operand, as the rest of a sentence about it.
type Mistype struct{ Says string }

func (m *Mistype) Error() string { return m.Says }

func mistype(format string, args ...any) error { return &Mistype{Says: fmt.Sprintf(format, args...)} }

// Paths types the value at a path of a value that is a message, by its descriptor where one was read,
// or of a value of any shape.
type Paths func(of Typed, path string) (Typed, error)

// TypeOf is the one reading of the types of an operand, which admission, the lowering to a Case and
// the evaluation of a guard all make, so that what is well typed for one is well typed for all. An
// order is of numbers, a negation and a conjunction of conditions, a comparison of two values of one
// type that is no message, and a path of a message. projected is the message the projected value is,
// or nil where no descriptor is read, and paths types a path; with no paths every path is of any
// shape. An operand of no known kind is of any shape: what a command and a guard may be made of is
// checked where each is admitted.
func TypeOf(o *modelirspb.Operand, projected protoreflect.MessageDescriptor, paths Paths) (Typed, error) {
	of := func(operands ...*modelirspb.Operand) ([]Typed, error) {
		out := make([]Typed, len(operands))
		for i, operand := range operands {
			var err error
			if out[i], err = TypeOf(operand, projected, paths); err != nil {
				return nil, err
			}
		}
		return out, nil
	}
	condition := Typed{Shape: ConditionShape}
	switch k := o.GetKind().(type) {
	case *modelirspb.Operand_Literal:
		return writtenType(k.Literal), nil
	case *modelirspb.Operand_Environment, *modelirspb.Operand_Run, *modelirspb.Operand_LearnedValue:
		return Typed{Shape: TextShape}, nil
	case *modelirspb.Operand_Projected:
		return Typed{Shape: MessageShape, Message: projected}, nil
	case *modelirspb.Operand_Path:
		from, err := of(k.Path.GetOf())
		switch {
		case err != nil:
			return Typed{}, err
		case from[0].Shape != AnyShape && from[0].Shape != MessageShape:
			return Typed{}, mistype("reads %s of %s, which is no message", k.Path.GetPath(), from[0].Shape)
		case paths == nil:
			return Typed{}, nil
		default:
			return paths(from[0], k.Path.GetPath())
		}
	case *modelirspb.Operand_Present:
		_, err := of(k.Present.GetOf())
		return condition, err
	case *modelirspb.Operand_Equal:
		sides, err := of(k.Equal.GetLeft(), k.Equal.GetRight())
		if err != nil {
			return Typed{}, err
		}
		return condition, compared(sides[0], sides[1])
	case *modelirspb.Operand_Greater:
		sides, err := of(k.Greater.GetLeft(), k.Greater.GetRight())
		return condition, every(sides, err, NumberShape, "orders", "numbers are ordered")
	case *modelirspb.Operand_Not:
		operands, err := of(k.Not.GetOf())
		return condition, every(operands, err, ConditionShape, "negates", "a condition is negated")
	case *modelirspb.Operand_All:
		operands, err := of(k.All.GetOperands()...)
		return condition, every(operands, err, ConditionShape, "joins", "conditions are joined")
	default:
		return Typed{}, nil
	}
}

// every is what is wrong with the first operand that is of neither one shape nor any, after what was
// wrong with typing them.
func every(operands []Typed, err error, want Shape, verb, only string) error {
	if err != nil {
		return err
	}
	for _, operand := range operands {
		if operand.Shape != AnyShape && operand.Shape != want {
			return mistype("%s %s, and only %s", verb, operand.Shape, only)
		}
	}
	return nil
}

// compared is what is wrong with comparing two values, or nil: each is one flag, number, text or
// enum value, they are of one type, and two enum values are of one enum, a written one being a name
// the other's enum has.
func compared(left, right Typed) error {
	for _, side := range []Typed{left, right} {
		if side.Shape == MessageShape || side.Shape == SeveralShape || side.Shape == OtherShape {
			return mistype("compares %s", side.Shape)
		}
	}
	switch {
	case left.Shape == AnyShape || right.Shape == AnyShape:
		return nil
	case left.Shape != right.Shape:
		return mistype("compares %s with %s", left.Shape, right.Shape)
	case left.Shape != EnumShape:
		return nil
	case left.Enum == nil && right.Enum == nil:
		return mistype("compares two enum values it writes out")
	case left.Enum != nil && right.Enum != nil && left.Enum.FullName() != right.Enum.FullName():
		return mistype("compares a value of %s with one of %s", left.Enum.FullName(), right.Enum.FullName())
	case left.Enum == nil && right.Enum.Values().ByName(protoreflect.Name(left.Name)) == nil:
		return mistype("compares a value of %s with %s, which it does not have", right.Enum.FullName(), left.Name)
	case right.Enum == nil && left.Enum.Values().ByName(protoreflect.Name(right.Name)) == nil:
		return mistype("compares a value of %s with %s, which it does not have", left.Enum.FullName(), right.Name)
	default:
		return nil
	}
}

// writtenType is the type of a value an operand writes out.
func writtenType(value *modelirspb.ProtoValue) Typed {
	switch v := value.GetKind().(type) {
	case *modelirspb.ProtoValue_Text, *modelirspb.ProtoValue_Named:
		return Typed{Shape: TextShape}
	case *modelirspb.ProtoValue_Flag:
		return Typed{Shape: ConditionShape}
	case *modelirspb.ProtoValue_Number:
		return Typed{Shape: NumberShape}
	case *modelirspb.ProtoValue_EnumName:
		return Typed{Shape: EnumShape, Name: v.EnumName}
	default:
		return Typed{Shape: OtherShape}
	}
}

var payloadSegment = regexp.MustCompile(`^([a-z0-9_]+)(<([a-z0-9_]+)>)?$`)

// PayloadFields is the fields a path of a Run Event's payload names, in order: plain fields, and
// `oneof<member>` for one member of a oneof, each one value. With no descriptor it checks how the
// path is written and names no field.
func PayloadFields(of protoreflect.MessageDescriptor, path string) ([]protoreflect.FieldDescriptor, error) {
	if path == "" {
		return nil, mistype("reads an empty path")
	}
	segments := strings.Split(path, ".")
	parts := make([][]string, len(segments))
	for i, segment := range segments {
		if parts[i] = payloadSegment.FindStringSubmatch(segment); parts[i] == nil {
			return nil, mistype("reads %q of the path %s, and a guard reads a field or oneof<member>", segment, path)
		}
	}
	if of == nil {
		return nil, nil
	}
	fields := make([]protoreflect.FieldDescriptor, len(segments))
	for i, segment := range segments {
		name, member := protoreflect.Name(parts[i][1]), protoreflect.Name(parts[i][3])
		field := of.Fields().ByName(name)
		if member != "" {
			oneof := of.Oneofs().ByName(name)
			if oneof == nil || oneof.Fields().ByName(member) == nil {
				return nil, mistype("reads %s, and %s has no such member", segment, of.FullName())
			}
			field = oneof.Fields().ByName(member)
		}
		switch {
		case field == nil:
			return nil, mistype("reads %s, and %s has no field %s", path, of.FullName(), name)
		case field.IsList() || field.IsMap():
			return nil, mistype("reads %s, and %s holds several values", path, field.FullName())
		case i < len(segments)-1 && field.Message() == nil:
			return nil, mistype("reads %s, and %s is no message", path, field.FullName())
		default:
		}
		fields[i], of = field, field.Message()
	}
	return fields, nil
}

// PayloadPath types a path of a Run Event's payload, as a guard reads one: a message, or one flag,
// text, enum value or signed integer.
func PayloadPath(of Typed, path string) (Typed, error) {
	fields, err := PayloadFields(of.Message, path)
	if err != nil || fields == nil {
		return Typed{}, err
	}
	switch field := fields[len(fields)-1]; field.Kind() {
	case protoreflect.MessageKind, protoreflect.GroupKind:
		return Typed{Shape: MessageShape, Message: field.Message()}, nil
	case protoreflect.BoolKind:
		return Typed{Shape: ConditionShape}, nil
	case protoreflect.StringKind:
		return Typed{Shape: TextShape}, nil
	case protoreflect.EnumKind:
		return Typed{Shape: EnumShape, Enum: field.Enum()}, nil
	case protoreflect.Int32Kind, protoreflect.Sint32Kind, protoreflect.Sfixed32Kind, protoreflect.Int64Kind, protoreflect.Sint64Kind, protoreflect.Sfixed64Kind:
		return Typed{Shape: NumberShape}, nil
	default:
		return Typed{}, mistype("reads %s, which is of kind %s", field.FullName(), field.Kind())
	}
}

// GuardProblem is what is wrong with the guard of a Run Event source, or nil. It is the one check of
// a guard: admission makes it with no descriptor, the lowering to a Case with the descriptor of the
// payload, and the evaluation of the guard on a recorded Run with the descriptor of the payload it
// evaluates. A guard is made of the payload and of the flags, numbers, texts and enum values it writes
// out, is well typed, and is a condition.
func GuardProblem(guard *modelirspb.Operand, payload protoreflect.MessageDescriptor) error {
	if problem := overThePayload(guard); problem != "" {
		return &Mistype{Says: problem}
	}
	computes, err := TypeOf(guard, payload, PayloadPath)
	if err != nil {
		return err
	}
	if computes.Shape != AnyShape && computes.Shape != ConditionShape {
		return mistype("is %s, and a guard is a condition", computes.Shape)
	}
	return nil
}

// payloadAlone ends what is wrong with a guard that reads something a recorded Run does not hold.
const payloadAlone = "; a Run Event's guard reads the event's payload alone"

// writtenInAGuard is what is wrong with a value a guard writes out, or empty. A guard is evaluated on
// a recorded Run, which holds no name a Case binds.
func writtenInAGuard(value *modelirspb.ProtoValue) string {
	switch value.GetKind().(type) {
	case *modelirspb.ProtoValue_Text, *modelirspb.ProtoValue_Flag, *modelirspb.ProtoValue_Number, *modelirspb.ProtoValue_EnumName:
		return ""
	case *modelirspb.ProtoValue_Named:
		return "writes out a name a Case binds" + payloadAlone
	default:
		return "writes out a value that is no text, flag, number or enum value"
	}
}

// overThePayload is what is wrong with what a Run Event's guard is made of, or empty: it reads the
// event's payload and what is written out, and nothing of the run.
func overThePayload(o *modelirspb.Operand) string {
	const alone = payloadAlone
	switch k := o.GetKind().(type) {
	case *modelirspb.Operand_Literal:
		return writtenInAGuard(k.Literal)
	case *modelirspb.Operand_Projected:
		return ""
	case *modelirspb.Operand_Run:
		return "reads the run's id" + alone
	case *modelirspb.Operand_Environment:
		return "reads the environment binding " + k.Environment + alone
	case *modelirspb.Operand_LearnedValue:
		return "reads the learned value " + k.LearnedValue + alone
	case *modelirspb.Operand_Path:
		return overThePayload(k.Path.GetOf())
	case *modelirspb.Operand_Present:
		return overThePayload(k.Present.GetOf())
	case *modelirspb.Operand_Equal:
		if left := overThePayload(k.Equal.GetLeft()); left != "" {
			return left
		}
		return overThePayload(k.Equal.GetRight())
	case *modelirspb.Operand_Greater:
		if left := overThePayload(k.Greater.GetLeft()); left != "" {
			return left
		}
		return overThePayload(k.Greater.GetRight())
	case *modelirspb.Operand_Not:
		return overThePayload(k.Not.GetOf())
	case *modelirspb.Operand_All:
		if len(k.All.GetOperands()) == 0 {
			return "is a conjunction of no operand"
		}
		for _, operand := range k.All.GetOperands() {
			if problem := overThePayload(operand); problem != "" {
				return problem
			}
		}
		return ""
	default:
		return "has an operand of no known kind"
	}
}

// evidenceFields checks the fields a kind of evidence carries: each is named once and read from a
// path, and an identity is named by one field of the kind, which the evidence retains.
func (a *realizing) evidenceFields(e *modelirspb.Evidence) {
	declared, named := map[string]bool{}, map[modelirspb.EvidenceField_Role]string{}
	for _, f := range e.GetFields() {
		at := f.GetPosition()
		if at.GetFile() == "" {
			at = e.GetPosition()
		}
		switch {
		case f.GetId() == "":
			a.report(at, "evidence %s has a field with no id", e.GetId())
			continue
		case declared[f.GetId()]:
			a.report(at, "evidence %s declares field %s twice", e.GetId(), f.GetId())
			continue
		default:
			declared[f.GetId()] = true
		}
		if f.GetPath() == "" {
			a.report(at, "evidence %s: field %s names no path", e.GetId(), f.GetId())
		}
		role := f.GetRole()
		if role == modelirspb.EvidenceField_ROLE_UNSPECIFIED {
			continue
		}
		spelled, isKnown := fieldRoles[role]
		switch other, taken := named[role]; {
		case !isKnown:
			a.report(at, "evidence %s: field %s names an identity of no known role", e.GetId(), f.GetId())
		case f.GetRedacted():
			a.report(at, "evidence %s: field %s names %s and is redacted; an identity is read from a field the evidence retains", e.GetId(), f.GetId(), spelled)
		case taken:
			a.report(at, "evidence %s: fields %s and %s both name %s; one field of a kind names an identity", e.GetId(), other, f.GetId(), spelled)
		default:
			named[role] = f.GetId()
		}
	}
}

// correlation checks that evidence is keyed by two different fields and carried by one declared
// observation, within a window that keeps something.
func (a *realizing) correlation() {
	c := a.r.GetCorrelation()
	if c == nil {
		a.v.report(a.r.GetPosition(), "%s declares no correlation", a.owner)
		return
	}
	at := c.GetPosition()
	if c.GetProjection() == "" || c.GetRun() == "" || c.GetOperation() == "" {
		a.report(at, "the correlation names no projection, no run field or no operation field")
	} else if c.GetRun() == c.GetOperation() {
		a.report(at, "the correlation keys its runs and its operations by one field, %s", c.GetRun())
	}
	if !a.observations[c.GetObservation()] {
		a.report(at, "the correlation: no observation %s", c.GetObservation())
	}
	for _, bound := range []struct {
		name string
		n    int64
	}{{"events", c.GetEvents()}, {"buffered", c.GetBuffered()}, {"keys", c.GetKeys()}, {"support", c.GetSupport()},
		{"work", c.GetWork()}, {"event size", c.GetEventSize()}} {
		if bound.n < 1 {
			a.report(at, "the correlation keeps %d %s; a window keeps at least one", bound.n, bound.name)
		}
	}
}

// commandOf is one command of a script with what a diagnostic calls it.
type commandOf struct {
	c    *modelirspb.Command
	name string
	at   *modelirspb.Position
	// always says every Case carries the command: it is no performance, and is under no condition.
	always bool
	// script is the id of the script the command is of.
	script string
}

func (a *realizing) script(mm *modelirspb.Machine, s *modelirspb.Script) {
	at := s.GetPosition()
	if !a.declared(at, "a script", "scripts", s.GetId()) && s.GetId() == "" {
		return
	}
	a.activation(mm, s)
	a.items(mm, s)
	fixed, all := map[string]bool{}, map[string]*modelirspb.Command{}
	var commands []commandOf
	note := func(c *modelirspb.Command, performs, always bool) {
		pos := c.GetPosition()
		if pos.GetFile() == "" {
			pos = at
		}
		if c.GetId() == "" {
			a.report(pos, "a command of script %s has no id", s.GetId())
			return
		}
		switch {
		case !performs:
			a.v.once(pos, "commands with id", c.GetId(), "of script", s.GetId(), "of realization", a.label)
			fixed[c.GetId()] = true
		case fixed[c.GetId()]:
			a.report(pos, "two commands with id %s of script %s", c.GetId(), s.GetId())
		default:
		}
		all[c.GetId()] = c
		commands = append(commands, commandOf{c, fmt.Sprintf("%s of script %s", c.GetId(), s.GetId()), pos, always, s.GetId()})
	}
	for _, item := range s.GetItems() {
		if item.GetCommand() != nil && len(item.GetPerforms()) == 0 {
			note(item.GetCommand(), false, len(item.GetWhen()) == 0)
		}
	}
	for _, item := range s.GetItems() {
		for _, p := range item.GetPerforms() {
			if p.GetCommand() == nil {
				a.report(p.GetPosition(), "script %s performs a class with no command", s.GetId())
				continue
			}
			note(p.GetCommand(), true, false)
			a.performs(mm, s, p)
		}
	}
	for _, c := range commands {
		a.command(s, c, all)
	}
	a.cycles(s, all)
}

// items checks that each item of a script is a command, with the classes it is carried for, or the
// place steps are performed, and not both.
func (a *realizing) items(mm *modelirspb.Machine, s *modelirspb.Script) {
	for _, item := range s.GetItems() {
		pos := item.GetPosition()
		if pos.GetFile() == "" {
			pos = s.GetPosition()
		}
		switch {
		case item.GetCommand() == nil && len(item.GetPerforms()) == 0:
			a.report(pos, "script %s has an item that is neither a command nor the place steps are performed", s.GetId())
		case item.GetCommand() != nil && len(item.GetPerforms()) > 0:
			a.report(pos, "script %s has an item that is both a command and the place steps are performed", s.GetId())
		case item.GetCommand() != nil && mm != nil:
			for _, c := range item.GetWhen() {
				a.v.actionClass(a.owner+": script "+s.GetId(), mm, c, pos)
			}
		case item.GetCommand() != nil:
		case len(item.GetWhen()) > 0:
			a.report(pos, "script %s performs steps under a condition; the path decides which are performed", s.GetId())
		default:
		}
	}
}

// performs checks the class a performance binds, and that no other performance of the realization
// binds it.
func (a *realizing) performs(mm *modelirspb.Machine, s *modelirspb.Script, p *modelirspb.Performance) {
	a.performing(mm, "script "+s.GetId(), p.GetStep(), p.GetPosition(), fmt.Sprintf("%s of script %s", p.GetCommand().GetId(), s.GetId()))
}

// performing records what performs one class of the machine, a command or the activation of a script,
// and reports a class the machine does not bind and one something else performs already.
func (a *realizing) performing(mm *modelirspb.Machine, where string, class *modelirspb.ActionClass, at *modelirspb.Position, by string) {
	if mm == nil {
		return
	}
	before := len(a.v.errs)
	a.v.actionClass(a.owner+": "+where, mm, class, at)
	if len(a.v.errs) != before {
		return
	}
	key := classKey(a.v.in, a.v.actions, class)
	if other, ok := a.performed[key]; ok {
		a.report(at, "class %s is performed by %s and by %s; a class is performed once", key, other, by)
		return
	}
	a.performed[key] = by
}

// role checks that a role is declared and of the kind its use needs.
func (a *realizing) role(at *modelirspb.Position, where, id string, kind modelirspb.Role_Kind) {
	switch role, ok := a.roles[id]; {
	case !ok:
		a.report(at, "%s: no role %s", where, id)
	case role.GetKind() != kind && known(modelirspb.Role_Kind_name, int32(role.GetKind())):
		a.report(at, "%s: role %s is %s, not %s", where, id, roleKinds[role.GetKind()], roleKinds[kind])
	default:
	}
}

func (a *realizing) activation(mm *modelirspb.Machine, s *modelirspb.Script) {
	at, where := s.GetPosition(), "script "+s.GetId()
	worker := func(name *modelirspb.Name, workerRole, queue string) {
		if name != nil && name.GetPrefix() == "" && name.GetSuffix() == "" && !name.GetFixture() {
			a.report(at, "%s is activated by a type with no name", where)
		}
		a.role(at, where, workerRole, modelirspb.Role_KIND_WORKER)
		a.role(at, where, queue, modelirspb.Role_KIND_TASK_QUEUE)
	}
	switch act := s.GetActivation().(type) {
	case *modelirspb.Script_Controller:
	case *modelirspb.Script_Workflow:
		worker(act.Workflow.GetWorkflowType(), act.Workflow.GetWorker(), act.Workflow.GetTaskQueue())
	case *modelirspb.Script_Activity:
		worker(act.Activity.GetActivityType(), act.Activity.GetWorker(), act.Activity.GetTaskQueue())
		for _, class := range act.Activity.GetStarts() {
			a.performing(mm, where, class, at, "the activation of "+where)
		}
	case *modelirspb.Script_NexusHandler:
		if act.NexusHandler.GetService() == "" || act.NexusHandler.GetOperation() == "" {
			a.report(at, "%s answers no service or no operation", where)
		}
		worker(nil, act.NexusHandler.GetWorker(), act.NexusHandler.GetTaskQueue())
	default:
		a.report(at, "%s names no activation", where)
	}
}

// binds records the one command that binds a learned value of the kind the binding gives it.
func (a *realizing) binds(c commandOf, id string, kind modelirspb.Learned_Kind) {
	if !a.reads(c, id, kind) {
		return
	}
	if other, ok := a.bound[id]; ok {
		a.report(c.at, "learned value %s is bound by %s and by %s; a learned value is bound once", id, other, c.name)
		return
	}
	a.bound[id] = c.name
}

// reads checks that a learned value is declared and of the kind its reader needs, any kind when kind
// is unspecified, and is whether it is declared.
func (a *realizing) reads(c commandOf, id string, kind modelirspb.Learned_Kind) bool {
	l, ok := a.learned[id]
	switch {
	case !ok:
		a.report(c.at, "command %s: no learned value %s", c.name, id)
		return false
	case kind != modelirspb.Learned_KIND_UNSPECIFIED && l.GetKind() != kind && known(modelirspb.Learned_Kind_name, int32(l.GetKind())):
		a.report(c.at, "command %s: learned value %s is %s, not %s", c.name, id, learnedKinds[l.GetKind()], learnedKinds[kind])
	default:
	}
	return true
}

func (a *realizing) command(s *modelirspb.Script, c commandOf, all map[string]*modelirspb.Command) {
	named := func(id string) {
		if _, ok := all[id]; !ok {
			a.report(c.at, "command %s: no command %s of script %s", c.name, id, s.GetId())
		}
	}
	for _, id := range c.c.GetAfter().GetCommands() {
		named(id)
	}
	if c.c.GetTimeoutMs() < 0 {
		a.report(c.at, "command %s has a deadline of %d milliseconds", c.name, c.c.GetTimeoutMs())
	}
	for _, id := range c.c.GetCloses() {
		a.closes(c, id)
	}
	switch in := c.c.GetInstruction().(type) {
	case *modelirspb.Command_Rpc:
		a.rpc(c, in.Rpc)
	case *modelirspb.Command_Poll:
		a.poll(c, in.Poll)
	case *modelirspb.Command_AwaitLearned:
		if a.reads(c, in.AwaitLearned, modelirspb.Learned_KIND_UNSPECIFIED) {
			a.read[in.AwaitLearned] = true
		}
	case *modelirspb.Command_AwaitCommand:
		named(in.AwaitCommand)
	case *modelirspb.Command_Finish:
		a.typed(c, in.Finish.GetResult(), false)
	case *modelirspb.Command_Fault:
		a.role(c.at, "command "+c.name, in.Fault.GetRole(), modelirspb.Role_KIND_TASK_QUEUE)
		if !known(modelirspb.Fault_Kind_name, int32(in.Fault.GetKind())) {
			a.report(c.at, "command %s is a fault of no known kind", c.name)
		}
	case *modelirspb.Command_WorkflowCommand:
		a.message(c, in.WorkflowCommand.GetCommand())
	case *modelirspb.Command_NexusReply:
		a.message(c, in.NexusReply.GetReply())
		if in.NexusReply.GetBinds() != "" {
			a.binds(c, in.NexusReply.GetBinds(), modelirspb.Learned_KIND_HANDLE)
		}
	case *modelirspb.Command_NexusCompletion:
		if a.reads(c, in.NexusCompletion.GetHandle(), modelirspb.Learned_KIND_HANDLE) {
			a.read[in.NexusCompletion.GetHandle()] = true
		}
		a.message(c, in.NexusCompletion.GetResult())
	case *modelirspb.Command_Hold:
		a.control(c, in.Hold)
	case *modelirspb.Command_Release:
		a.control(c, in.Release)
	case *modelirspb.Command_AttemptFailure:
		if s.GetActivity() == nil {
			a.report(c.at, "command %s fails an attempt, and script %s is no activity's", c.name, s.GetId())
		}
		a.message(c, in.AttemptFailure.GetFailure())
	case *modelirspb.Command_AttemptCanceled:
		if s.GetActivity() == nil {
			a.report(c.at, "command %s cancels an attempt, and script %s is no activity's", c.name, s.GetId())
		}
	default:
		a.report(c.at, "command %s names no instruction", c.name)
	}
}

// closes checks that a command's read is the one closing read of an exhaustive kind of evidence: it
// reads the kind, and every Case carries it.
func (a *realizing) closes(c commandOf, id string) {
	e, declared := a.evidence[id]
	switch other, closed := a.closed[id]; {
	case !declared:
		a.report(c.at, "command %s closes evidence %s, which is not declared", c.name, id)
		return
	case !e.GetExhaustive():
		a.report(c.at, "command %s closes evidence %s, which is not exhaustive", c.name, id)
		return
	case closed:
		a.report(c.at, "evidence %s is closed by %s and by %s; an exhaustive kind has one closing read", id, other, c.name)
		return
	default:
		a.closed[id] = c.name
	}
	reads := c.c.GetPoll().GetEvidence() == id
	if e.GetHistory() != "" {
		reads = false
		for _, read := range c.c.GetRpc().GetReads() {
			for _, target := range read.GetTargets() {
				reads = reads || target.GetLift() != ""
			}
		}
	}
	// The Run's own record of a command is read by that command: what the command records is all
	// there is of the kind. A Case that does not carry the command has no record of it, and nothing
	// is inferred from a source that was never read, so such a command need not be in every Case.
	if record := e.GetRunEvent(); record != nil && record.GetScript() == c.script && record.GetCommand() == c.c.GetId() {
		return
	}
	if !reads {
		a.report(c.at, "command %s closes evidence %s and does not read it: a history kind is closed by the read that lifts it, the Run's own record of a command by that command, and any other by a poll of it", c.name, id)
	}
	if !c.always {
		a.report(c.at, "command %s closes evidence %s and is not a command every Case carries", c.name, id)
	}
}

func (a *realizing) control(c commandOf, id string) {
	if !a.controls[id] {
		a.report(c.at, "command %s: no control %s", c.name, id)
	}
}

func (a *realizing) assignments(c commandOf, assign []*modelirspb.Assignment, polls bool) {
	targets := map[string]bool{}
	for _, as := range assign {
		if as.GetTarget() == "" {
			a.report(c.at, "command %s assigns a value to no field", c.name)
		} else if targets[as.GetTarget()] {
			a.report(c.at, "command %s assigns %s twice", c.name, as.GetTarget())
		}
		targets[as.GetTarget()] = true
		a.typed(c, as.GetValue(), polls)
	}
}

// typed checks a value a command computes, and the types of what it computes it from, and is its
// shape.
func (a *realizing) typed(c commandOf, o *modelirspb.Operand, polls bool) Shape {
	a.operand(c, o, polls)
	computes, err := TypeOf(o, nil, nil)
	if err != nil {
		a.report(c.at, "command %s: %s", c.name, err)
	}
	return computes.Shape
}

func (a *realizing) rpc(c commandOf, rpc *modelirspb.Rpc) {
	a.role(c.at, "command "+c.name, rpc.GetRole(), modelirspb.Role_KIND_ENDPOINT)
	if rpc.GetMethod() == "" {
		a.report(c.at, "command %s calls no method", c.name)
	}
	a.assignments(c, rpc.GetAssign(), false)
	held := a.r.GetCorrelation().GetObservation()
	for _, read := range rpc.GetReads() {
		if !known(modelirspb.ResponseRead_Cardinality_name, int32(read.GetCardinality())) {
			a.report(c.at, "command %s reads %s at no known cardinality", c.name, read.GetPath())
		}
		for _, target := range read.GetTargets() {
			switch tg := target.GetTarget().(type) {
			case *modelirspb.Target_Observe:
				switch {
				case !a.observations[tg.Observe]:
					a.report(c.at, "command %s: no observation %s", c.name, tg.Observe)
				case tg.Observe == held:
					a.report(c.at, "command %s observes a value into %s, which holds the correlation's evidence", c.name, held)
				default:
				}
			case *modelirspb.Target_Bind:
				if read.GetCardinality() == modelirspb.ResponseRead_CARDINALITY_EACH {
					a.report(c.at, "command %s binds %s from each element of %s; a learned value is one value", c.name, tg.Bind, read.GetPath())
				}
				a.binds(c, tg.Bind, modelirspb.Learned_KIND_TEXT)
			case *modelirspb.Target_Lift:
				if tg.Lift != held {
					a.report(c.at, "command %s lifts evidence into %s, and the correlation reads %s", c.name, tg.Lift, held)
				}
			default:
				a.report(c.at, "command %s reads %s into nothing", c.name, read.GetPath())
			}
		}
	}
}

func (a *realizing) poll(c commandOf, poll *modelirspb.Poll) {
	a.role(c.at, "command "+c.name, poll.GetRole(), modelirspb.Role_KIND_ENDPOINT)
	switch e, ok := a.evidence[poll.GetEvidence()]; {
	case !ok:
		a.report(c.at, "command %s: no evidence %s", c.name, poll.GetEvidence())
	case e.GetRunEvent() != nil:
		a.report(c.at, "command %s: evidence %s is a Run Event, which a poll does not read", c.name, poll.GetEvidence())
	case e.GetRead() == nil && e.GetSingle() == nil && e.GetFrom() != nil:
		a.report(c.at, "command %s: evidence %s is a history event, which a poll does not read", c.name, poll.GetEvidence())
	default:
	}
	a.assignments(c, poll.GetAssign(), false)
	if computes := a.typed(c, poll.GetUntil(), true); computes != AnyShape && computes != ConditionShape {
		a.report(c.at, "command %s polls until %s, and a poll's condition is a condition", c.name, computes)
	}
	if poll.GetIntervalMs() < 1 {
		a.report(c.at, "command %s polls every %d milliseconds", c.name, poll.GetIntervalMs())
	}
}

// operand checks a value a command computes. Only a poll's condition reads the value the poll is
// looking at.
func (a *realizing) operand(c commandOf, o *modelirspb.Operand, polls bool) {
	switch k := o.GetKind().(type) {
	case *modelirspb.Operand_Literal:
		switch k.Literal.GetKind().(type) {
		case *modelirspb.ProtoValue_Text, *modelirspb.ProtoValue_Flag, *modelirspb.ProtoValue_Number,
			*modelirspb.ProtoValue_EnumName, *modelirspb.ProtoValue_Named:
		default:
			a.report(c.at, "command %s: a literal operand is a text, a flag, a number, an enum value or a name", c.name)
		}
	case *modelirspb.Operand_Environment:
		if k.Environment == "" {
			a.report(c.at, "command %s reads an environment binding with no id", c.name)
		}
	case *modelirspb.Operand_Run:
	case *modelirspb.Operand_LearnedValue:
		if a.reads(c, k.LearnedValue, modelirspb.Learned_KIND_TEXT) {
			a.read[k.LearnedValue] = true
		}
		if c.c.GetRegardless() {
			a.report(c.at, "command %s runs whatever became of the commands before it, and reads learned value %s, which is read only once it is bound",
				c.name, k.LearnedValue)
		}
	case *modelirspb.Operand_Projected:
		if !polls {
			a.report(c.at, "command %s reads the value a poll is looking at, and is no poll's condition", c.name)
		}
	case *modelirspb.Operand_Path:
		a.operand(c, k.Path.GetOf(), polls)
	case *modelirspb.Operand_Present:
		a.operand(c, k.Present.GetOf(), polls)
	case *modelirspb.Operand_Equal:
		a.operand(c, k.Equal.GetLeft(), polls)
		a.operand(c, k.Equal.GetRight(), polls)
	case *modelirspb.Operand_All:
		if len(k.All.GetOperands()) == 0 {
			a.report(c.at, "command %s: a conjunction of no operand", c.name)
		}
		for _, operand := range k.All.GetOperands() {
			a.operand(c, operand, polls)
		}
	case *modelirspb.Operand_Greater:
		a.operand(c, k.Greater.GetLeft(), polls)
		a.operand(c, k.Greater.GetRight(), polls)
	case *modelirspb.Operand_Not:
		a.operand(c, k.Not.GetOf(), polls)
	default:
		a.report(c.at, "command %s: an operand of no known kind", c.name)
	}
}

// message checks a protobuf message written out: it names its type, sets no field twice, and every
// value it sets is of a known kind.
func (a *realizing) message(c commandOf, m *modelirspb.Proto) {
	if m.GetMessage() == "" {
		a.report(c.at, "command %s: a message with no name", c.name)
		return
	}
	set := map[string]bool{}
	for _, f := range m.GetFields() {
		if set[f.GetName()] {
			a.report(c.at, "command %s: %s sets %s twice", c.name, m.GetMessage(), f.GetName())
		}
		set[f.GetName()] = true
		a.protoValue(c, m.GetMessage()+"."+f.GetName(), f.GetValue())
	}
}

func (a *realizing) protoValue(c commandOf, field string, value *modelirspb.ProtoValue) {
	switch k := value.GetKind().(type) {
	case *modelirspb.ProtoValue_Text, *modelirspb.ProtoValue_Flag, *modelirspb.ProtoValue_Number,
		*modelirspb.ProtoValue_EnumName, *modelirspb.ProtoValue_Utf8, *modelirspb.ProtoValue_Named:
	case *modelirspb.ProtoValue_Message:
		a.message(c, k.Message)
	case *modelirspb.ProtoValue_Mapping:
		keys := map[string]bool{}
		for _, e := range k.Mapping.GetEntries() {
			if keys[e.GetKey()] {
				a.report(c.at, "command %s: %s has the key %s twice", c.name, field, e.GetKey())
			}
			keys[e.GetKey()] = true
			a.protoValue(c, field, e.GetValue())
		}
	case *modelirspb.ProtoValue_RoleId:
		if _, ok := a.roles[k.RoleId]; !ok {
			a.report(c.at, "command %s: no role %s", c.name, k.RoleId)
		}
	default:
		a.report(c.at, "command %s: %s is set to a value of no known kind", c.name, field)
	}
}

// cycles reports each command of a script that runs after itself. A command with no `after` runs
// after the item before it, which is earlier in the script and so closes no cycle of its own.
func (a *realizing) cycles(s *modelirspb.Script, all map[string]*modelirspb.Command) {
	reported := map[string]bool{}
	for _, item := range s.GetItems() {
		ids := []string{item.GetCommand().GetId()}
		for _, p := range item.GetPerforms() {
			ids = append(ids, p.GetCommand().GetId())
		}
		for _, start := range ids {
			through, cyclic := runsAfterItself(all, start)
			if all[start] == nil || reported[start] || !cyclic {
				continue
			}
			reported[start] = true
			for _, id := range through {
				reported[id] = true
			}
			at := all[start].GetPosition()
			if at.GetFile() == "" {
				at = s.GetPosition()
			}
			if len(through) == 0 {
				a.report(at, "script %s: command %s runs after itself", s.GetId(), start)
			} else {
				a.report(at, "script %s: command %s runs after itself, through %s", s.GetId(), start, strings.Join(through, ", "))
			}
		}
	}
}

// runsAfterItself is whether following the commands a command runs after leads back to it, and the
// commands the way back goes through.
func runsAfterItself(all map[string]*modelirspb.Command, start string) ([]string, bool) {
	var path []string
	seen := map[string]bool{}
	var visit func(id string) bool
	visit = func(id string) bool {
		for _, next := range all[id].GetAfter().GetCommands() {
			if next == start {
				return true
			}
			if all[next] == nil || seen[next] {
				continue
			}
			seen[next] = true
			path = append(path, next)
			if visit(next) {
				return true
			}
			path = path[:len(path)-1]
		}
		return false
	}
	return path, visit(start)
}
