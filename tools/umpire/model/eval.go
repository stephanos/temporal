// Package model loads the Umpire IR, validates it, and interprets it: every table, identity and
// fingerprint of a Model is computed here from the IR's declarations, with no code of the front end
// that wrote them. The evaluation rules are the ones model/SEMANTICS.md states.
package model

import (
	"fmt"
	"slices"
	"strconv"
	"strings"

	umpirespb "go.temporal.io/server/api/umpire/v1"
)

// Kind is what a Value is.
type Kind int

const (
	BoolValue Kind = iota
	IntValue
	TextValue
	EnumValue
	RecordValue
	ListValue
	LambdaValue
)

// Value is one value of the IR: a Boolean, an integer, a string, an enum case with its fields, a
// record with its fields, a list, or an anonymous function.
type Value struct {
	Kind   Kind
	Bool   bool
	Int    int64
	Text   string
	Type   string
	Case   string
	Fields []Value
	Items  []Value
	lambda *umpirespb.Lambda
	env    *env
}

// Key spells a value the way Umpire keys it: a case by its name, followed by its fields; a record by
// its fields; all joined by "-".
func (v Value) Key() string {
	switch v.Kind {
	case BoolValue:
		return strconv.FormatBool(v.Bool)
	case IntValue:
		return strconv.FormatInt(v.Int, 10)
	case TextValue:
		return v.Text
	case EnumValue:
		parts := []string{v.Case}
		for _, f := range v.Fields {
			parts = append(parts, f.Key())
		}
		return strings.Join(parts, "-")
	case RecordValue:
		parts := make([]string, len(v.Fields))
		for i, f := range v.Fields {
			parts[i] = f.Key()
		}
		return strings.Join(parts, "-")
	case ListValue:
		parts := make([]string, len(v.Items))
		for i, f := range v.Items {
			parts[i] = f.Key()
		}
		return "[" + strings.Join(parts, ",") + "]"
	default:
		return "<lambda>"
	}
}

// spelled writes a value's structure, which its key may not tell apart: a record as its fields in
// parentheses, a case with fields as the case applied to them, and a list in brackets.
func (v Value) spelled() string {
	spell := func(vs []Value) string {
		parts := make([]string, len(vs))
		for i, x := range vs {
			parts[i] = x.spelled()
		}
		return strings.Join(parts, ", ")
	}
	switch {
	case v.Kind == EnumValue && len(v.Fields) > 0:
		return v.Case + "(" + spell(v.Fields) + ")"
	case v.Kind == RecordValue:
		return "(" + spell(v.Fields) + ")"
	case v.Kind == ListValue:
		return "[" + spell(v.Items) + "]"
	default:
		return v.Key()
	}
}

// equal is structural equality.
func (v Value) equal(o Value) bool {
	if v.Kind != o.Kind || v.Bool != o.Bool || v.Int != o.Int || v.Text != o.Text || v.Type != o.Type ||
		v.Case != o.Case || len(v.Fields) != len(o.Fields) || len(v.Items) != len(o.Items) {
		return false
	}
	for i := range v.Fields {
		if !v.Fields[i].equal(o.Fields[i]) {
			return false
		}
	}
	for i := range v.Items {
		if !v.Items[i].equal(o.Items[i]) {
			return false
		}
	}
	return true
}

// StepType is the framework's step record, `{outcome, state, facts, because}`, which step functions
// return lists of.
const StepType = "umpire.Step"

var stepFields = []string{"outcome", "state", "facts", "because"}

// deliveryType is the framework's delivery record, `{message, redeliveries}`: one message a channel
// holds, and how many more times than once it has been delivered.
const deliveryType = "umpire.Delivery"

var deliveryFields = []string{"message", "redeliveries"}

// Error is an evaluation or validation failure at the position of the IR node it concerns, which is
// where the front end says the author wrote it.
type Error struct {
	Position string
	Message  string
}

func (e *Error) Error() string {
	if e.Position == "" {
		return e.Message
	}
	return e.Position + ": " + e.Message
}

// Hole is ⊥, a value of the Model rather than a failure of it: an evaluation that reached the declared
// hole ID, or, with no ID, a value no case of a match matches, an undeclared hole. It propagates the
// way an error does, and a row whose value it is is a hole row.
type Hole struct {
	ID       string
	Position string
	Message  string
}

func (h *Hole) Error() string {
	if h.Position == "" {
		return h.Message
	}
	return h.Position + ": " + h.Message
}

func errorAt(p *umpirespb.Position, format string, args ...any) error {
	return &Error{Position: where(p), Message: fmt.Sprintf(format, args...)}
}

func where(p *umpirespb.Position) string {
	if p.GetFile() == "" {
		return ""
	}
	return fmt.Sprintf("%s:%d", p.GetFile(), p.GetLine())
}

type env struct {
	name   string
	value  Value
	parent *env
}

func (e *env) bind(name string, v Value) *env { return &env{name: name, value: v, parent: e} }

func (e *env) lookup(name string) (Value, bool) {
	for ; e != nil; e = e.parent {
		if e.name == name {
			return e.value, true
		}
	}
	return Value{}, false
}

// Interpreter evaluates a Model's expressions.
// It also lists catalogs, within its ceilings.
type Interpreter struct {
	model     *umpirespb.Model
	types     map[string]*umpirespb.Type
	functions map[string]*umpirespb.Function
	channels  map[string]*umpirespb.Channel
	holes     map[string]*umpirespb.Hole
	ceilings  Ceilings
	// sizing holds the types and channels whose catalogs are being counted.
	sizing map[string]bool
}

// NewInterpreter indexes a Model's declarations.
// It interprets them within defaultCeilings.
func NewInterpreter(m *umpirespb.Model) *Interpreter {
	in := &Interpreter{model: m, types: map[string]*umpirespb.Type{}, functions: map[string]*umpirespb.Function{},
		channels: map[string]*umpirespb.Channel{}, holes: map[string]*umpirespb.Hole{}, ceilings: defaultCeilings,
		sizing: map[string]bool{}}
	for _, t := range m.GetTypes() {
		in.types[t.GetName()] = t
	}
	for _, f := range m.GetFunctions() {
		in.functions[f.GetName()] = f
	}
	for _, c := range m.GetChannels() {
		in.channels[c.GetId()] = c
	}
	for _, h := range m.GetHoles() {
		in.holes[h.GetId()] = h
	}
	return in
}

// Call applies a function to arguments, checking its precondition first: a call outside it is a
// Model error, not a value.
func (in *Interpreter) Call(name string, args []Value, at *umpirespb.Position) (Value, error) {
	f, ok := in.functions[name]
	if !ok {
		return Value{}, errorAt(at, "no function %s", name)
	}
	if len(args) != len(f.GetParams()) {
		return Value{}, errorAt(at, "%s takes %d arguments, got %d", name, len(f.GetParams()), len(args))
	}
	var e *env
	for i, p := range f.GetParams() {
		e = e.bind(p.GetName(), args[i])
	}
	if f.GetRequires() != nil {
		ok, err := in.eval(f.GetRequires(), e)
		if err != nil {
			return Value{}, err
		}
		if !ok.Bool {
			return Value{}, errorAt(f.GetPosition(), "%s is called outside its precondition", name)
		}
	}
	return in.eval(f.GetBody(), e)
}

// apply applies an anonymous function value.
func (in *Interpreter) apply(l Value, args []Value) (Value, error) {
	if l.Kind != LambdaValue {
		return Value{}, &Error{Message: "not a function"}
	}
	e := l.env
	for i, p := range l.lambda.GetParams() {
		e = e.bind(p.GetName(), args[i])
	}
	return in.eval(l.lambda.GetBody(), e)
}

// Eval evaluates a closed expression.
func (in *Interpreter) Eval(x *umpirespb.Expr) (Value, error) { return in.eval(x, nil) }

func (in *Interpreter) eval(x *umpirespb.Expr, e *env) (Value, error) {
	switch k := x.GetKind().(type) {
	case *umpirespb.Expr_Literal:
		return in.literal(k.Literal), nil
	case *umpirespb.Expr_Var:
		v, ok := e.lookup(k.Var)
		if !ok {
			return Value{}, errorAt(x.GetPosition(), "unbound name %s", k.Var)
		}
		return v, nil
	case *umpirespb.Expr_Field:
		return in.field(x, k.Field, e)
	case *umpirespb.Expr_Call:
		args, err := in.evalAll(k.Call.GetArgs(), e)
		if err != nil {
			return Value{}, err
		}
		return in.Call(k.Call.GetFunction(), args, x.GetPosition())
	case *umpirespb.Expr_Construct:
		return in.construct(x, k.Construct, e)
	case *umpirespb.Expr_Copy:
		return in.copy(x, k.Copy, e)
	case *umpirespb.Expr_Unary:
		return in.unary(x, k.Unary, e)
	case *umpirespb.Expr_Binary:
		return in.binary(x, k.Binary, e)
	case *umpirespb.Expr_If:
		c, err := in.eval(k.If.GetCondition(), e)
		if err != nil {
			return Value{}, err
		}
		if c.Bool {
			return in.eval(k.If.GetThen(), e)
		}
		return in.eval(k.If.GetElse(), e)
	case *umpirespb.Expr_Match:
		return in.match(x, k.Match, e)
	case *umpirespb.Expr_Let:
		v, err := in.eval(k.Let.GetValue(), e)
		if err != nil {
			return Value{}, err
		}
		return in.eval(k.Let.GetBody(), e.bind(k.Let.GetName(), v))
	case *umpirespb.Expr_List:
		items, err := in.evalAll(k.List.GetItems(), e)
		if err != nil {
			return Value{}, err
		}
		return Value{Kind: ListValue, Items: items}, nil
	case *umpirespb.Expr_Lambda:
		return Value{Kind: LambdaValue, lambda: k.Lambda, env: e}, nil
	case *umpirespb.Expr_Hole:
		name := k.Hole
		if h, ok := in.holes[k.Hole]; ok {
			name = h.GetName()
		}
		return Value{}, &Hole{ID: k.Hole, Position: where(x.GetPosition()), Message: "reaches the hole " + name}
	case *umpirespb.Expr_Inbox:
		return in.inbox(x, k.Inbox, e)
	default:
		return Value{}, errorAt(x.GetPosition(), "unknown expression %T", k)
	}
}

func (in *Interpreter) evalAll(xs []*umpirespb.Expr, e *env) ([]Value, error) {
	out := make([]Value, len(xs))
	for i, x := range xs {
		v, err := in.eval(x, e)
		if err != nil {
			return nil, err
		}
		out[i] = v
	}
	return out, nil
}

func (in *Interpreter) literal(v *umpirespb.Value) Value {
	switch k := v.GetKind().(type) {
	case *umpirespb.Value_Bool:
		return Value{Kind: BoolValue, Bool: k.Bool}
	case *umpirespb.Value_Int:
		return Value{Kind: IntValue, Int: k.Int}
	case *umpirespb.Value_Text:
		return Value{Kind: TextValue, Text: k.Text}
	case *umpirespb.Value_Enum:
		out := Value{Kind: EnumValue, Type: k.Enum.GetType(), Case: k.Enum.GetCase()}
		for _, f := range k.Enum.GetFields() {
			out.Fields = append(out.Fields, in.literal(f))
		}
		return out
	case *umpirespb.Value_Record:
		out := Value{Kind: RecordValue, Type: k.Record.GetType()}
		for _, f := range k.Record.GetFields() {
			out.Fields = append(out.Fields, in.literal(f))
		}
		return out
	case *umpirespb.Value_List:
		out := Value{Kind: ListValue}
		for _, f := range k.List.GetItems() {
			out.Items = append(out.Items, in.literal(f))
		}
		return out
	default:
		return Value{}
	}
}

// conforms is whether a value is of a type: of the kind the type admits, within a range's bounds, of a
// declared type's own case with fields of theirs, and for a channel a list of its deliveries. It does
// not ask whether a channel's contents are in its catalog: the state domain does.
func (in *Interpreter) conforms(v Value, t *umpirespb.TypeRef) bool {
	all := func(vs []Value, t *umpirespb.TypeRef) bool {
		for _, x := range vs {
			if !in.conforms(x, t) {
				return false
			}
		}
		return true
	}
	switch r := t.GetRef().(type) {
	case *umpirespb.TypeRef_Bool:
		return v.Kind == BoolValue
	case *umpirespb.TypeRef_Int:
		return v.Kind == IntValue
	case *umpirespb.TypeRef_IntRange:
		return v.Kind == IntValue && r.IntRange.GetLow() <= v.Int && v.Int <= r.IntRange.GetHigh()
	case *umpirespb.TypeRef_List:
		return v.Kind == ListValue && all(v.Items, r.List)
	case *umpirespb.TypeRef_Channel:
		c, ok := in.channels[r.Channel]
		if !ok || v.Kind != ListValue {
			return false
		}
		redeliveries := &umpirespb.TypeRef{Ref: &umpirespb.TypeRef_IntRange{IntRange: &umpirespb.IntRange{High: int64(c.GetDuplicates())}}}
		for _, d := range v.Items {
			if d.Kind != RecordValue || d.Type != deliveryType || len(d.Fields) != len(deliveryFields) ||
				!in.conforms(d.Fields[0], c.GetMessage()) || !in.conforms(d.Fields[1], redeliveries) {
				return false
			}
		}
		return true
	case *umpirespb.TypeRef_Named:
		return in.conformsToDeclared(v, r.Named)
	default:
		return false
	}
}

func (in *Interpreter) conformsToDeclared(v Value, name string) bool {
	decl, ok := in.types[name]
	if !ok || v.Type != name {
		return false
	}
	fields := decl.GetRecord().GetFields()
	switch decl.GetShape().(type) {
	case *umpirespb.Type_Record:
		if v.Kind != RecordValue {
			return false
		}
	case *umpirespb.Type_Enum:
		i := slices.IndexFunc(decl.GetEnum().GetCases(), func(c *umpirespb.Case) bool { return c.GetName() == v.Case })
		if v.Kind != EnumValue || i < 0 {
			return false
		}
		fields = decl.GetEnum().GetCases()[i].GetFields()
	default:
		return false
	}
	if len(v.Fields) != len(fields) {
		return false
	}
	for i, f := range fields {
		if !in.conforms(v.Fields[i], f.GetType()) {
			return false
		}
	}
	return true
}

// fieldNames is the field names of a record, or of one case of an enum.
func (in *Interpreter) fieldNames(typeName, caseName string) ([]string, bool) {
	switch typeName {
	case StepType:
		return stepFields, true
	case deliveryType:
		return deliveryFields, true
	default:
	}
	t, ok := in.types[typeName]
	if !ok {
		return nil, false
	}
	var fields []*umpirespb.Field
	switch s := t.GetShape().(type) {
	case *umpirespb.Type_Record:
		fields = s.Record.GetFields()
	case *umpirespb.Type_Enum:
		found := false
		for _, c := range s.Enum.GetCases() {
			if c.GetName() == caseName {
				fields, found = c.GetFields(), true
			}
		}
		if !found {
			return nil, false
		}
	default:
	}
	names := make([]string, len(fields))
	for i, f := range fields {
		names[i] = f.GetName()
	}
	return names, true
}

func (in *Interpreter) field(x *umpirespb.Expr, f *umpirespb.FieldAccess, e *env) (Value, error) {
	base, err := in.eval(f.GetBase(), e)
	if err != nil {
		return Value{}, err
	}
	names, _ := in.fieldNames(base.Type, base.Case)
	for i, n := range names {
		if n == f.GetField() {
			return base.Fields[i], nil
		}
	}
	return Value{}, errorAt(x.GetPosition(), "%s has no field %s", base.Type, f.GetField())
}

func (in *Interpreter) construct(x *umpirespb.Expr, c *umpirespb.Construct, e *env) (Value, error) {
	args, err := in.evalAll(c.GetArgs(), e)
	if err != nil {
		return Value{}, err
	}
	names, ok := in.fieldNames(c.GetType(), c.GetCase())
	if !ok {
		return Value{}, errorAt(x.GetPosition(), "no type %s %s", c.GetType(), c.GetCase())
	}
	if len(args) != len(names) {
		return Value{}, errorAt(x.GetPosition(), "%s %s takes %d fields, got %d", c.GetType(), c.GetCase(), len(names), len(args))
	}
	if c.GetCase() != "" {
		return Value{Kind: EnumValue, Type: c.GetType(), Case: c.GetCase(), Fields: args}, nil
	}
	return Value{Kind: RecordValue, Type: c.GetType(), Fields: args}, nil
}

func (in *Interpreter) copy(x *umpirespb.Expr, c *umpirespb.Copy, e *env) (Value, error) {
	base, err := in.eval(c.GetBase(), e)
	if err != nil {
		return Value{}, err
	}
	names, _ := in.fieldNames(base.Type, base.Case)
	out := base
	out.Fields = append([]Value{}, base.Fields...)
	for _, u := range c.GetUpdates() {
		v, err := in.eval(u.GetValue(), e)
		if err != nil {
			return Value{}, err
		}
		i := indexOf(names, u.GetName())
		if i < 0 {
			return Value{}, errorAt(x.GetPosition(), "%s has no field %s", base.Type, u.GetName())
		}
		out.Fields[i] = v
	}
	return out, nil
}

func (in *Interpreter) unary(x *umpirespb.Expr, u *umpirespb.Unary, e *env) (Value, error) {
	v, err := in.eval(u.GetOperand(), e)
	if err != nil {
		return Value{}, err
	}
	switch u.GetOp() {
	case umpirespb.Unary_OP_NOT:
		return Value{Kind: BoolValue, Bool: !v.Bool}, nil
	case umpirespb.Unary_OP_NEG:
		return Value{Kind: IntValue, Int: -v.Int}, nil
	default:
		return Value{}, errorAt(x.GetPosition(), "unknown unary operator %v", u.GetOp())
	}
}

func (in *Interpreter) binary(x *umpirespb.Expr, b *umpirespb.Binary, e *env) (Value, error) {
	l, err := in.eval(b.GetLeft(), e)
	if err != nil {
		return Value{}, err
	}
	// And and Or do not evaluate the right operand when the left decides, as Scala does not.
	switch b.GetOp() {
	case umpirespb.Binary_OP_AND:
		if !l.Bool {
			return l, nil
		}
		return in.eval(b.GetRight(), e)
	case umpirespb.Binary_OP_OR:
		if l.Bool {
			return l, nil
		}
		return in.eval(b.GetRight(), e)
	default:
	}
	r, err := in.eval(b.GetRight(), e)
	if err != nil {
		return Value{}, err
	}
	boolean := func(v bool) (Value, error) { return Value{Kind: BoolValue, Bool: v}, nil }
	switch b.GetOp() {
	case umpirespb.Binary_OP_EQ:
		return boolean(l.equal(r))
	case umpirespb.Binary_OP_NE:
		return boolean(!l.equal(r))
	case umpirespb.Binary_OP_LT:
		return boolean(l.Int < r.Int)
	case umpirespb.Binary_OP_LE:
		return boolean(l.Int <= r.Int)
	case umpirespb.Binary_OP_GT:
		return boolean(l.Int > r.Int)
	case umpirespb.Binary_OP_GE:
		return boolean(l.Int >= r.Int)
	case umpirespb.Binary_OP_ADD:
		return Value{Kind: IntValue, Int: l.Int + r.Int}, nil
	case umpirespb.Binary_OP_SUB:
		return Value{Kind: IntValue, Int: l.Int - r.Int}, nil
	case umpirespb.Binary_OP_CONCAT:
		return Value{Kind: ListValue, Items: append(append([]Value{}, l.Items...), r.Items...)}, nil
	case umpirespb.Binary_OP_CONTAINS:
		for _, item := range r.Items {
			if item.equal(l) {
				return boolean(true)
			}
		}
		return boolean(false)
	default:
		return Value{}, errorAt(x.GetPosition(), "unknown binary operator %v", b.GetOp())
	}
}

// match takes the first case whose pattern matches and whose guard holds. A value no case matches is
// a hole in the Model.
// That hole is an undeclared one: a Hole with no ID.
func (in *Interpreter) match(x *umpirespb.Expr, m *umpirespb.Match, e *env) (Value, error) {
	v, err := in.eval(m.GetScrutinee(), e)
	if err != nil {
		return Value{}, err
	}
	for _, c := range m.GetCases() {
		bound, ok := in.bindPattern(c.GetPattern(), v, e)
		if !ok {
			continue
		}
		if c.GetGuard() != nil {
			g, err := in.eval(c.GetGuard(), bound)
			if err != nil {
				return Value{}, err
			}
			if !g.Bool {
				continue
			}
		}
		return in.eval(c.GetBody(), bound)
	}
	return Value{}, &Hole{Position: where(x.GetPosition()), Message: "no case matches " + v.Key()}
}

func (in *Interpreter) bindPattern(p *umpirespb.Pattern, v Value, e *env) (*env, bool) {
	switch k := p.GetKind().(type) {
	case *umpirespb.Pattern_Wildcard:
		return e, true
	case *umpirespb.Pattern_Bind:
		inner, ok := in.bindPattern(k.Bind.GetPattern(), v, e)
		if !ok {
			return nil, false
		}
		return inner.bind(k.Bind.GetName(), v), true
	case *umpirespb.Pattern_Literal:
		return e, in.literal(k.Literal).equal(v)
	case *umpirespb.Pattern_Case:
		if v.Kind != EnumValue || v.Type != k.Case.GetType() || v.Case != k.Case.GetCase() ||
			len(v.Fields) != len(k.Case.GetFields()) {
			return nil, false
		}
		for i, fp := range k.Case.GetFields() {
			var ok bool
			if e, ok = in.bindPattern(fp, v.Fields[i], e); !ok {
				return nil, false
			}
		}
		return e, true
	case *umpirespb.Pattern_Alternatives:
		for _, alt := range k.Alternatives.GetPatterns() {
			if bound, ok := in.bindPattern(alt, v, e); ok {
				return bound, true
			}
		}
		return nil, false
	default:
		return nil, false
	}
}

func indexOf(xs []string, x string) int {
	for i, y := range xs {
		if y == x {
			return i
		}
	}
	return -1
}
