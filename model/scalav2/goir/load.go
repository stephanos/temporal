package goir

import (
	"errors"
	"os"

	modelirspb "go.temporal.io/server/api/modelir/v1"
	"google.golang.org/protobuf/encoding/protojson"
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
func Validate(m *modelirspb.Model) error {
	v := &validator{types: map[string]*modelirspb.Type{}, functions: map[string]*modelirspb.Function{},
		actions: map[string]*modelirspb.Action{}}
	for _, t := range m.GetTypes() {
		v.types[t.GetName()] = t
	}
	for _, f := range m.GetFunctions() {
		v.functions[f.GetName()] = f
	}
	for _, a := range m.GetActions() {
		v.actions[a.GetId()] = a
	}
	for _, t := range m.GetTypes() {
		v.typeDecl(t)
	}
	for _, f := range m.GetFunctions() {
		scope := map[string]bool{}
		for _, p := range f.GetParams() {
			v.typeRef(p.GetType(), f.GetPosition())
			scope[p.GetName()] = true
		}
		if f.GetRequires() != nil {
			v.expr(f.GetRequires(), scope)
		}
		v.expr(f.GetBody(), scope)
	}
	for _, a := range m.GetActions() {
		for _, p := range a.GetInputs() {
			v.typeRef(p.GetType(), a.GetPosition())
		}
	}
	for _, mm := range m.GetMachines() {
		v.machine(mm)
	}
	return errors.Join(v.errs...)
}

type validator struct {
	types     map[string]*modelirspb.Type
	functions map[string]*modelirspb.Function
	actions   map[string]*modelirspb.Action
	errs      []error
}

func (v *validator) report(at *modelirspb.Position, format string, args ...any) {
	v.errs = append(v.errs, errorAt(at, format, args...))
}

func (v *validator) typeDecl(t *modelirspb.Type) {
	switch s := t.GetShape().(type) {
	case *modelirspb.Type_Enum:
		for _, c := range s.Enum.GetCases() {
			for _, f := range c.GetFields() {
				v.stateField(f.GetType(), t.GetPosition())
			}
		}
	case *modelirspb.Type_Record:
		for _, f := range s.Record.GetFields() {
			v.stateField(f.GetType(), t.GetPosition())
		}
	default:
		v.report(t.GetPosition(), "type %s has no shape", t.GetName())
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
		v.expr(k.Unary.GetOperand(), scope)
	case *modelirspb.Expr_Binary:
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
	default:
		v.report(at, "an expression of no known kind")
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
	if e := x.GetEnum(); e != nil {
		n, ok := v.fields(e.GetType(), e.GetCase())
		switch {
		case !ok:
			v.report(at, "no case %s of %s", e.GetCase(), e.GetType())
		case n != len(e.GetFields()):
			v.report(at, "%s %s has %d fields, not %d", e.GetType(), e.GetCase(), n, len(e.GetFields()))
		default:
		}
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
	default:
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
		}
	}
	for _, name := range []string{m.GetEvidence(), m.GetRefines().GetMap()} {
		if f, ok := v.functions[name]; name != "" && (!ok || len(f.GetParams()) != 1) {
			v.report(at, "%s: %s is not a function of one argument", m.GetName(), name)
		}
	}
	for _, a := range m.GetUnobservable() {
		if _, ok := v.actions[a]; !ok {
			v.report(at, "no action %s", a)
		}
	}
}
