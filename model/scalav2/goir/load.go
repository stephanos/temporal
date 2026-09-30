package goir

import (
	"errors"
	"fmt"
	"os"
	"slices"
	"strings"

	modelirspb "go.temporal.io/server/api/modelir/v1"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
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
	return errors.Join(v.errs...)
}

// newValidator indexes the Model's declarations by the keys the IR names them by, and reports two
// declarations under one key.
func newValidator(m *modelirspb.Model) *validator {
	v := &validator{declared: map[string]bool{}, types: map[string]*modelirspb.Type{}, functions: map[string]*modelirspb.Function{},
		actions: map[string]*modelirspb.Action{}, channels: map[string]*modelirspb.Channel{},
		monitors: map[string]*modelirspb.Monitor{}, assumptions: map[string]bool{}, holes: map[string]bool{},
		machines: map[string]*modelirspb.Machine{}, compositions: map[string]*modelirspb.Composition{},
		properties: map[claim]bool{}, scenarios: map[claim]bool{}, calls: map[string][]string{}}
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
	v.arity(at, p.GetMachine()+"."+p.GetName(), p.GetHolds(), n)
	if c := p.GetWhenClass(); c != nil {
		v.actionClass(c, at)
	}
}

func (v *validator) scenario(s *modelirspb.Scenario) {
	at := s.GetPosition()
	v.claimed(at, s.GetMachine())
	if s.GetStart() != nil {
		v.expr(s.GetStart(), map[string]bool{})
	}
	for _, c := range s.GetActions() {
		v.actionClass(c, at)
	}
}

func (v *validator) actionClass(c *modelirspb.ActionClass, at *modelirspb.Position) {
	v.actionRefs(at, []string{c.GetAction()})
	for _, x := range c.GetInputs() {
		v.value(x, at)
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
