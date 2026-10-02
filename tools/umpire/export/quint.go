package export

import (
	"fmt"
	"slices"
	"strconv"
	"strings"

	umpirespb "go.temporal.io/server/api/umpire/v1"
	umpiremodel "go.temporal.io/server/tools/umpire/model"
)

const quintBackend = "quint"

// QuintExport is a slice written as one Quint module. The module translates the IR's types and
// functions, declaration for declaration, and for each machine computes, with Quint's own evaluator,
// the states its starts reach and every result of every class from each of them, and the same for the
// product with its monitors. Its one state variable holds that dump, which `quint run` writes as a
// trace.
//
// Nothing in the module is a table Go computed. The one number Go gives it is how many rounds of
// successors to take, and the dump says whether the states it reached are closed under a step, so a
// bound that was too small shows as a disagreement.
type QuintExport struct {
	Module string
	Text   string
	// Machines names the exported machines, in the order of the dump's fields.
	Machines []string
	// Compositions names the exported compositions, in the order of the dump's fields after the
	// machines'.
	Compositions []string
	// Unsupported lists what the slice declares and the module leaves out.
	Unsupported []Receipt

	// from is the slice the module was written from, whose types read the dump back.
	from *Slice
	// tags reads a variant of the module back as the IR case it stands for.
	tags map[string]variant
	// bindings is, for each exported machine, the action of each of its class variants.
	bindings [][]*umpirespb.Action
	// composed is how each exported composition's part of a dump is read back.
	composed []*composedExport
	// definitions is the module's text before its state variable, and stateful what a check module
	// declares in its place for each machine that names monitors.
	definitions string
	stateful    map[string]string
}

type variant struct {
	typ  string
	enum *umpirespb.Case
}

// quint writes one Model as a Quint module.
type quint struct {
	s         *Slice
	x         *QuintExport
	out       strings.Builder
	typeNames map[string]string
	functions map[string]string
	writing   map[string]bool
	members   map[string]string
	accessors map[string]bool
	holes     map[string]string
	fresh     int
	// mu is the type of the monitors' states of the machine being written.
	mu string
	// only names the one machine a check module is written for, or is empty for the whole export.
	only string
}

// Quint exports the slice's machines to Quint. A construct the exporter does not translate is an
// UnsupportedError, and nothing is exported around it: a hole is never written as a disabled action.
func (s *Slice) Quint() (*QuintExport, error) { return s.quint("") }

// quint writes the export, or, for one machine, only what a check module of it needs: the machine's
// own definitions, without the dump. A model checker is given no more than it explores.
func (s *Slice) quint(only string) (*QuintExport, error) {
	q := &quint{s: s, only: only, x: &QuintExport{Module: "umpire_slice", from: s, stateful: map[string]string{}, tags: map[string]variant{}}, typeNames: map[string]string{},
		functions: map[string]string{}, writing: map[string]bool{}, members: map[string]string{}, accessors: map[string]bool{},
		holes: map[string]string{}}
	m := s.Model
	if channels := m.GetChannels(); len(channels) > 0 {
		return nil, q.unsupported(channels[0].GetPosition(), "the channel %s: channels and their derived deliveries are not exported", channels[0].GetName())
	}
	for _, h := range m.GetHoles() {
		q.holes[h.GetId()] = h.GetName()
	}
	for i, t := range m.GetTypes() {
		q.typeNames[t.GetName()] = fmt.Sprintf("T%d_%s", i, plain(t.GetName()))
	}
	for _, t := range m.GetTypes() {
		if err := q.typeDecl(t); err != nil {
			return nil, err
		}
	}
	var fields, views []string
	for i, decl := range m.GetMachines() {
		mm := s.machines[decl.GetName()]
		if only != "" && decl.GetName() != only {
			q.x.Machines, q.x.bindings = append(q.x.Machines, decl.GetName()), append(q.x.bindings, nil)
			continue
		}
		if len(mm.Holes) > 0 {
			h := mm.Holes[0]
			return nil, q.unsupported(decl.GetPosition(), "%s at the row '%s' of %s: a hole row is neither a result nor a disabled pair, and the module has no value for it",
				describeHole(m, h.Hole.ID), h.Row, decl.GetName())
		}
		typ, err := q.machine(i, mm)
		if err != nil {
			return nil, err
		}
		q.x.Machines = append(q.x.Machines, decl.GetName())
		fields = append(fields, fmt.Sprintf("m%d: %s", i, typ))
		views = append(views, fmt.Sprintf("m%d: m%d_view", i, i))
		if r := decl.GetRefines(); r != nil {
			q.x.Unsupported = append(q.x.Unsupported, Receipt{Backend: quintBackend, Claim: ModuleRefinement, Subject: decl.GetName(), Kind: Unsupported,
				Explanation: fmt.Sprintf("the refinement of %s by %s is not exported: Quint is given each machine's transitions and monitors, and no refinement is claimed of it",
					r.GetProduct(), decl.GetName())})
		}
	}
	if only == "" {
		composedFields, composedViews, err := q.compositions()
		if err != nil {
			return nil, err
		}
		fields, views = append(fields, composedFields...), append(views, composedViews...)
	}
	for _, p := range m.GetProgress() {
		q.x.Unsupported = append(q.x.Unsupported, Receipt{Backend: quintBackend, Claim: ProgressAgreement, Subject: p.GetMachine() + "." + p.GetName(), Kind: Unsupported,
			Explanation: fmt.Sprintf("the progress claim %s of %s is not exported: fairness, deadlock and deadline witnesses are goir's alone", p.GetName(), p.GetMachine())})
	}
	if n := len(m.GetQueries()); n > 0 {
		q.x.Unsupported = append(q.x.Unsupported, Receipt{Backend: quintBackend, Claim: QueryAgreement, Subject: fmt.Sprintf("%d Queries", n), Kind: Unsupported,
			Explanation: "the Model's Queries are not exported: Quint is given the Properties they ask, read on every step, and the monitors that watch them, and no Scenario or Limits"})
	}
	q.x.definitions = q.out.String()
	q.x.Text = fmt.Sprintf("// Written by model/backends from %s. Do not edit.\nmodule %s {\n%s\n  var out: {%s}\n  action init = out' = {%s}\n  action step = out' = out\n}\n",
		m.GetSource(), q.x.Module, q.x.definitions, strings.Join(fields, ", "), strings.Join(views, ", "))
	return q.x, nil
}

func (q *quint) unsupported(at *umpirespb.Position, format string, args ...any) error {
	e := &UnsupportedError{Backend: quintBackend, Construct: fmt.Sprintf(format, args...)}
	if at != nil {
		e.Position = at.GetFile() + ":" + strconv.Itoa(int(at.GetLine()))
	}
	return e
}

// plain spells the last segment of a qualified name with letters, digits and underscores alone.
func plain(name string) string {
	name = name[strings.LastIndexAny(name, ".$")+1:]
	var b strings.Builder
	for _, r := range name {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' {
			b.WriteRune(r)
		} else {
			b.WriteByte('_')
		}
	}
	return b.String()
}

func (q *quint) typeDecl(t *umpirespb.Type) error {
	name := q.typeNames[t.GetName()]
	switch {
	case t.GetEnum() != nil:
		var variants []string
		for _, c := range t.GetEnum().GetCases() {
			tag := name + "_" + plain(c.GetName())
			q.x.tags[tag] = variant{typ: t.GetName(), enum: c}
			if len(c.GetFields()) == 0 {
				variants = append(variants, tag)
				continue
			}
			fields, err := q.fields(c.GetFields())
			if err != nil {
				return err
			}
			variants = append(variants, tag+"("+fields+")")
		}
		fmt.Fprintf(&q.out, "  type %s =\n    | %s\n", name, strings.Join(variants, "\n    | "))
	case t.GetRecord() != nil:
		fields, err := q.fields(t.GetRecord().GetFields())
		if err != nil {
			return err
		}
		fmt.Fprintf(&q.out, "  type %s = %s\n", name, fields)
	default:
		return q.unsupported(t.GetPosition(), "the type %s, which is neither an enum nor a record", t.GetName())
	}
	return nil
}

func (q *quint) fields(fields []*umpirespb.Field) (string, error) {
	parts := make([]string, len(fields))
	for i, f := range fields {
		t, err := q.typeRef(f.GetType())
		if err != nil {
			return "", err
		}
		parts[i] = "f_" + plain(f.GetName()) + ": " + t
	}
	return "{" + strings.Join(parts, ", ") + "}", nil
}

func (q *quint) typeRef(t *umpirespb.TypeRef) (string, error) {
	switch r := t.GetRef().(type) {
	case *umpirespb.TypeRef_Named:
		if name, ok := q.typeNames[r.Named]; ok {
			return name, nil
		}
		return "", q.unsupported(nil, "the type %s in a state, an input or a monitor", r.Named)
	case *umpirespb.TypeRef_Bool:
		return "bool", nil
	case *umpirespb.TypeRef_IntRange, *umpirespb.TypeRef_Int:
		return "int", nil
	case *umpirespb.TypeRef_List:
		item, err := q.typeRef(r.List)
		return "List[" + item + "]", err
	default:
		return "", q.unsupported(nil, "a channel's contents as a type")
	}
}

// membersOf is the set of a finite type's members, which the module computes itself: the classes of
// an action are every assignment of its inputs.
func (q *quint) membersOf(t *umpirespb.TypeRef) (string, error) {
	switch r := t.GetRef().(type) {
	case *umpirespb.TypeRef_Bool:
		return "Set(false, true)", nil
	case *umpirespb.TypeRef_IntRange:
		return fmt.Sprintf("%d.to(%d)", r.IntRange.GetLow(), r.IntRange.GetHigh()), nil
	case *umpirespb.TypeRef_Named:
		if name, ok := q.members[r.Named]; ok {
			return name, nil
		}
		decl, ok := q.s.types[r.Named]
		if !ok {
			return "", q.unsupported(nil, "an input of the type %s", r.Named)
		}
		name := "members_" + q.typeNames[r.Named]
		q.members[r.Named] = name
		var expr string
		if decl.GetRecord() != nil {
			product, err := q.product(decl.GetRecord().GetFields(), "")
			if err != nil {
				return "", err
			}
			expr = product
		} else {
			var parts []string
			for _, c := range decl.GetEnum().GetCases() {
				tag := q.typeNames[r.Named] + "_" + plain(c.GetName())
				if len(c.GetFields()) == 0 {
					parts = append(parts, "Set("+tag+")")
					continue
				}
				product, err := q.product(c.GetFields(), tag)
				if err != nil {
					return "", err
				}
				parts = append(parts, product)
			}
			expr = parts[0]
			for _, p := range parts[1:] {
				expr += ".union(" + p + ")"
			}
		}
		fmt.Fprintf(&q.out, "  pure val %s: Set[%s] = %s\n", name, q.typeNames[r.Named], expr)
		return name, nil
	default:
		return "", q.unsupported(nil, "an input with no finite catalog")
	}
}

// product is every assignment of some fields as a set of records, each wrapped in a variant when one
// is named.
func (q *quint) product(fields []*umpirespb.Field, tag string) (string, error) {
	sets, record := make([]string, len(fields)), make([]string, len(fields))
	for i, f := range fields {
		set, err := q.membersOf(f.GetType())
		if err != nil {
			return "", err
		}
		sets[i] = set
		record[i] = fmt.Sprintf("f_%s: t._%d", plain(f.GetName()), i+1)
	}
	if len(fields) == 1 {
		record[0] = "f_" + plain(fields[0].GetName()) + ": t"
		return fmt.Sprintf("%s.map(t => %s({%s}))", sets[0], tag, record[0]), nil
	}
	return fmt.Sprintf("tuples(%s).map(t => %s({%s}))", strings.Join(sets, ", "), tag, strings.Join(record, ", ")), nil
}

// scope is the names an expression may read, each as the Quint expression that stands for it.
type scope map[string]string

func (sc scope) with(name, expr string) scope {
	out := scope{}
	for k, v := range sc {
		out[k] = v
	}
	out[name] = expr
	return out
}

// function writes a function of the IR, after the functions it calls, and gives its Quint name.
func (q *quint) function(name string, at *umpirespb.Position) (string, error) {
	if written, ok := q.functions[name]; ok {
		return written, nil
	}
	i := slices.IndexFunc(q.s.Model.GetFunctions(), func(f *umpirespb.Function) bool { return f.GetName() == name })
	if i < 0 {
		return "", q.unsupported(at, "a call of %s, which the Model does not declare", name)
	}
	f := q.s.Model.GetFunctions()[i]
	if q.writing[name] {
		return "", q.unsupported(f.GetPosition(), "the function %s, which calls itself", name)
	}
	q.writing[name] = true
	sc, params := scope{}, make([]string, len(f.GetParams()))
	for k, p := range f.GetParams() {
		params[k] = fmt.Sprintf("p%d_%s", k, plain(p.GetName()))
		sc[p.GetName()] = params[k]
	}
	body, err := q.expr(f.GetBody(), sc)
	if err != nil {
		return "", err
	}
	if f.GetRequires() != nil {
		requires, err := q.expr(f.GetRequires(), sc)
		if err != nil {
			return "", err
		}
		// A call outside the precondition is an error of the Model: the evaluator stops on it.
		body = fmt.Sprintf("if (%s) %s else List().head()", requires, body)
	}
	written := fmt.Sprintf("f%d_%s", i, plain(name))
	if len(params) == 0 {
		fmt.Fprintf(&q.out, "  pure val %s = %s\n", written, body)
	} else {
		fmt.Fprintf(&q.out, "  pure def %s(%s) = %s\n", written, strings.Join(params, ", "), body)
	}
	q.functions[name] = written
	return written, nil
}

func (q *quint) exprs(xs []*umpirespb.Expr, sc scope) ([]string, error) {
	out := make([]string, len(xs))
	for i, x := range xs {
		var err error
		if out[i], err = q.expr(x, sc); err != nil {
			return nil, err
		}
	}
	return out, nil
}

func (q *quint) expr(x *umpirespb.Expr, sc scope) (string, error) {
	at := x.GetPosition()
	switch k := x.GetKind().(type) {
	case *umpirespb.Expr_Literal:
		return q.literal(k.Literal, at)
	case *umpirespb.Expr_Var:
		if written, ok := sc[k.Var]; ok {
			return written, nil
		}
		return "", q.unsupported(at, "the name %s, which nothing in scope binds", k.Var)
	case *umpirespb.Expr_Field:
		base, err := q.expr(k.Field.GetBase(), sc)
		return base + ".f_" + plain(k.Field.GetField()), err
	case *umpirespb.Expr_Call:
		f, err := q.function(k.Call.GetFunction(), at)
		if err != nil {
			return "", err
		}
		args, err := q.exprs(k.Call.GetArgs(), sc)
		if err != nil || len(args) == 0 {
			return f, err
		}
		return f + "(" + strings.Join(args, ", ") + ")", nil
	case *umpirespb.Expr_Construct:
		args, err := q.exprs(k.Construct.GetArgs(), sc)
		if err != nil {
			return "", err
		}
		return q.construct(k.Construct.GetType(), k.Construct.GetCase(), args, at)
	case *umpirespb.Expr_Copy:
		return q.copied(k.Copy, sc)
	case *umpirespb.Expr_Unary:
		return q.unary(k.Unary, sc, at)
	case *umpirespb.Expr_Binary:
		return q.binary(k.Binary, sc, at)
	case *umpirespb.Expr_If:
		parts, err := q.exprs([]*umpirespb.Expr{k.If.GetCondition(), k.If.GetThen(), k.If.GetElse()}, sc)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("(if (%s) %s else %s)", parts[0], parts[1], parts[2]), nil
	case *umpirespb.Expr_Match:
		return q.match(k.Match, sc)
	case *umpirespb.Expr_Let:
		value, err := q.expr(k.Let.GetValue(), sc)
		if err != nil {
			return "", err
		}
		name := q.local(k.Let.GetName())
		body, err := q.expr(k.Let.GetBody(), sc.with(k.Let.GetName(), name))
		return fmt.Sprintf("{ pure val %s = %s\n      %s }", name, value, body), err
	case *umpirespb.Expr_List:
		items, err := q.exprs(k.List.GetItems(), sc)
		return "[" + strings.Join(items, ", ") + "]", err
	case *umpirespb.Expr_Lambda:
		return "", q.unsupported(at, "an anonymous function as a value")
	case *umpirespb.Expr_Hole:
		return "", q.unsupported(at, "the hole %s: the module has no value for unknown behavior, and it is not written as a disabled action", q.holes[k.Hole])
	case *umpirespb.Expr_Inbox:
		return "", q.unsupported(at, "the channel operation %s", k.Inbox.GetOp())
	default:
		return "", q.unsupported(at, "an expression of no known kind")
	}
}

// copied writes a record with some of its fields replaced.
func (q *quint) copied(c *umpirespb.Copy, sc scope) (string, error) {
	base, err := q.expr(c.GetBase(), sc)
	if err != nil {
		return "", err
	}
	parts := []string{"..." + base}
	for _, u := range c.GetUpdates() {
		v, err := q.expr(u.GetValue(), sc)
		if err != nil {
			return "", err
		}
		parts = append(parts, "f_"+plain(u.GetName())+": "+v)
	}
	return "{" + strings.Join(parts, ", ") + "}", nil
}

func (q *quint) unary(u *umpirespb.Unary, sc scope, at *umpirespb.Position) (string, error) {
	operand, err := q.expr(u.GetOperand(), sc)
	if err != nil {
		return "", err
	}
	switch u.GetOp() {
	case umpirespb.Unary_OP_NOT:
		return "not(" + operand + ")", nil
	case umpirespb.Unary_OP_NEG:
		return "(-" + operand + ")", nil
	default:
		return "", q.unsupported(at, "the unary operator %s", u.GetOp())
	}
}

func (q *quint) local(name string) string {
	q.fresh++
	return fmt.Sprintf("v%d_%s", q.fresh, plain(name))
}

func (q *quint) binary(b *umpirespb.Binary, sc scope, at *umpirespb.Position) (string, error) {
	sides, err := q.exprs([]*umpirespb.Expr{b.GetLeft(), b.GetRight()}, sc)
	if err != nil {
		return "", err
	}
	l, r := sides[0], sides[1]
	infix := map[umpirespb.Binary_Op]string{
		umpirespb.Binary_OP_EQ: "==", umpirespb.Binary_OP_NE: "!=", umpirespb.Binary_OP_AND: "and", umpirespb.Binary_OP_OR: "or",
		umpirespb.Binary_OP_LT: "<", umpirespb.Binary_OP_LE: "<=", umpirespb.Binary_OP_GT: ">", umpirespb.Binary_OP_GE: ">=",
		umpirespb.Binary_OP_ADD: "+", umpirespb.Binary_OP_SUB: "-",
	}
	if op, ok := infix[b.GetOp()]; ok {
		return fmt.Sprintf("(%s %s %s)", l, op, r), nil
	}
	switch b.GetOp() {
	case umpirespb.Binary_OP_CONCAT:
		return fmt.Sprintf("%s.concat(%s)", l, r), nil
	case umpirespb.Binary_OP_CONTAINS:
		name := q.local("item")
		return fmt.Sprintf("(%s.select(%s => %s == %s).length() > 0)", r, name, name, l), nil
	default:
		return "", q.unsupported(at, "the binary operator %s", b.GetOp())
	}
}

func (q *quint) literal(v *umpirespb.Value, at *umpirespb.Position) (string, error) {
	values := func(vs []*umpirespb.Value) ([]string, error) {
		out := make([]string, len(vs))
		for i, f := range vs {
			var err error
			if out[i], err = q.literal(f, at); err != nil {
				return nil, err
			}
		}
		return out, nil
	}
	switch k := v.GetKind().(type) {
	case *umpirespb.Value_Bool:
		return strconv.FormatBool(k.Bool), nil
	case *umpirespb.Value_Int:
		return strconv.FormatInt(k.Int, 10), nil
	case *umpirespb.Value_Text:
		return strconv.Quote(k.Text), nil
	case *umpirespb.Value_Enum:
		fields, err := values(k.Enum.GetFields())
		if err != nil {
			return "", err
		}
		return q.construct(k.Enum.GetType(), k.Enum.GetCase(), fields, at)
	case *umpirespb.Value_Record:
		fields, err := values(k.Record.GetFields())
		if err != nil {
			return "", err
		}
		return q.construct(k.Record.GetType(), "", fields, at)
	case *umpirespb.Value_List:
		items, err := values(k.List.GetItems())
		return "[" + strings.Join(items, ", ") + "]", err
	default:
		return "", q.unsupported(at, "a literal of no known kind")
	}
}

var stepFields = []string{"outcome", "state", "facts", "because"}

// construct writes a record, the step record, or an enum case from its arguments in field order.
func (q *quint) construct(typ, name string, args []string, at *umpirespb.Position) (string, error) {
	record := func(names []string) (string, error) {
		if len(names) != len(args) {
			return "", q.unsupported(at, "%s built from %d values, and it has %d fields", typ, len(args), len(names))
		}
		parts := make([]string, len(args))
		for i, a := range args {
			parts[i] = "f_" + plain(names[i]) + ": " + a
		}
		return "{" + strings.Join(parts, ", ") + "}", nil
	}
	if typ == umpiremodel.StepType {
		return record(stepFields)
	}
	decl, ok := q.s.types[typ]
	if !ok {
		return "", q.unsupported(at, "a value of the type %s", typ)
	}
	if decl.GetRecord() != nil {
		return record(fieldNames(decl.GetRecord().GetFields()))
	}
	for _, c := range decl.GetEnum().GetCases() {
		if c.GetName() != name {
			continue
		}
		tag := q.typeNames[typ] + "_" + plain(name)
		if len(c.GetFields()) == 0 && len(args) == 0 {
			return tag, nil
		}
		fields, err := record(fieldNames(c.GetFields()))
		return tag + "(" + fields + ")", err
	}
	return "", q.unsupported(at, "the case %s, which %s does not declare", name, typ)
}

func fieldNames(fields []*umpirespb.Field) []string {
	out := make([]string, len(fields))
	for i, f := range fields {
		out[i] = f.GetName()
	}
	return out
}

// match writes a match as a chain of conditions, one per case in order. Quint's own `match` reads one
// variant deep and takes no literal, guard or alternative, so a pattern is written as the condition
// it tests, and a name it binds as the expression that reads the value out. A value no case matches
// is an undeclared hole: the chain ends in an expression the evaluator stops on, so that it is an
// error of the run and never a result.
func (q *quint) match(m *umpirespb.Match, sc scope) (string, error) {
	scrutinee, err := q.expr(m.GetScrutinee(), sc)
	if err != nil {
		return "", err
	}
	name := q.local("matched")
	chain := "List().head()"
	for i := len(m.GetCases()) - 1; i >= 0; i-- {
		c := m.GetCases()[i]
		cond, bound, err := q.pattern(c.GetPattern(), name, sc)
		if err != nil {
			return "", err
		}
		if c.GetGuard() != nil {
			guard, err := q.expr(c.GetGuard(), bound)
			if err != nil {
				return "", err
			}
			cond = fmt.Sprintf("(%s and %s)", cond, guard)
		}
		body, err := q.expr(c.GetBody(), bound)
		if err != nil {
			return "", err
		}
		chain = fmt.Sprintf("(if (%s) %s\n      else %s)", cond, body, chain)
	}
	return fmt.Sprintf("{ pure val %s = %s\n      %s }", name, scrutinee, chain), nil
}

// pattern is the condition under which a value matches a pattern, and the scope its body reads.
func (q *quint) pattern(p *umpirespb.Pattern, value string, sc scope) (string, scope, error) {
	switch k := p.GetKind().(type) {
	case *umpirespb.Pattern_Wildcard:
		return "true", sc, nil
	case *umpirespb.Pattern_Bind:
		return q.pattern(k.Bind.GetPattern(), value, sc.with(k.Bind.GetName(), value))
	case *umpirespb.Pattern_Literal:
		literal, err := q.literal(k.Literal, nil)
		return fmt.Sprintf("(%s == %s)", value, literal), sc, err
	case *umpirespb.Pattern_Case:
		decl := q.s.types[k.Case.GetType()]
		i := slices.IndexFunc(decl.GetEnum().GetCases(), func(c *umpirespb.Case) bool { return c.GetName() == k.Case.GetCase() })
		if i < 0 {
			return "", nil, q.unsupported(nil, "a pattern of the case %s, which %s does not declare", k.Case.GetCase(), k.Case.GetType())
		}
		c := decl.GetEnum().GetCases()[i]
		tag := q.typeNames[k.Case.GetType()] + "_" + plain(c.GetName())
		if len(c.GetFields()) == 0 {
			return fmt.Sprintf("(%s == %s)", value, tag), sc, nil
		}
		if len(k.Case.GetFields()) != len(c.GetFields()) {
			return "", nil, q.unsupported(nil, "a pattern of %s with %d fields, and the case has %d", tag, len(k.Case.GetFields()), len(c.GetFields()))
		}
		q.accessor(tag, c)
		conds := []string{fmt.Sprintf("is_%s(%s)", tag, value)}
		for j, sub := range k.Case.GetFields() {
			cond, bound, err := q.pattern(sub, fmt.Sprintf("get_%s_%s(%s)", tag, plain(c.GetFields()[j].GetName()), value), sc)
			if err != nil {
				return "", nil, err
			}
			sc = bound
			if cond != "true" {
				conds = append(conds, cond)
			}
		}
		return "(" + strings.Join(conds, " and ") + ")", sc, nil
	case *umpirespb.Pattern_Alternatives:
		var conds []string
		for _, alt := range k.Alternatives.GetPatterns() {
			cond, bound, err := q.pattern(alt, value, sc)
			if err != nil {
				return "", nil, err
			}
			if len(bound) != len(sc) {
				return "", nil, q.unsupported(nil, "an alternative of a pattern that binds a name")
			}
			conds = append(conds, cond)
		}
		return "(" + strings.Join(conds, " or ") + ")", sc, nil
	default:
		return "", nil, q.unsupported(nil, "a pattern of no known kind")
	}
}

// accessor writes, once, the test of a variant with fields and the reader of each of its fields.
// A reader is called only under its test.
func (q *quint) accessor(tag string, c *umpirespb.Case) {
	if q.accessors[tag] {
		return
	}
	q.accessors[tag] = true
	fmt.Fprintf(&q.out, "  pure def is_%s(v) = match v { | %s(_) => true | _ => false }\n", tag, tag)
	for _, f := range c.GetFields() {
		name := plain(f.GetName())
		fmt.Fprintf(&q.out, "  pure def get_%s_%s(v) = match v { | %s(r) => r.f_%s | _ => List().head() }\n", tag, name, tag, name)
	}
}

// binding is one step binding of a machine as the module writes its classes: the variant of the
// machine's class type, the set of the action's classes, and the arm of the step that takes one.
type binding struct {
	variant string
	classes string
	arm     string
	action  *umpirespb.Action
}

// binding writes step binding j of machine i. The classes of an action are every assignment of its
// inputs, which the module computes from the inputs' types.
func (q *quint) binding(i, j int, b *umpirespb.StepBinding) (binding, error) {
	a, ok := q.s.actions[b.GetAction()]
	if !ok {
		return binding{}, q.unsupported(b.GetPosition(), "a step of the action %s, which the Model does not declare", b.GetAction())
	}
	if a.GetDelivers() != "" || a.GetLoses() != "" {
		return binding{}, q.unsupported(b.GetPosition(), "the action %s, whose rows a channel derives", a.GetName())
	}
	f, err := q.function(b.GetFunction(), b.GetPosition())
	if err != nil {
		return binding{}, err
	}
	tag := fmt.Sprintf("K%d_%d", i, j)
	if len(a.GetInputs()) == 0 {
		return binding{variant: tag, classes: "Set(" + tag + ")", arm: fmt.Sprintf("| %s => %s(s)", tag, f), action: a}, nil
	}
	n := len(a.GetInputs())
	inputs, sets, record, args := make([]string, n), make([]string, n), make([]string, n), []string{"s"}
	for k, p := range a.GetInputs() {
		t, err := q.typeRef(p.GetType())
		if err != nil {
			return binding{}, err
		}
		if sets[k], err = q.membersOf(p.GetType()); err != nil {
			return binding{}, err
		}
		inputs[k] = fmt.Sprintf("i%d: %s", k, t)
		record[k] = fmt.Sprintf("i%d: t._%d", k, k+1)
		args = append(args, fmt.Sprintf("a.i%d", k))
	}
	classes := fmt.Sprintf("tuples(%s).map(t => %s({%s}))", strings.Join(sets, ", "), tag, strings.Join(record, ", "))
	if n == 1 {
		classes = fmt.Sprintf("%s.map(t => %s({i0: t}))", sets[0], tag)
	}
	return binding{variant: tag + "({" + strings.Join(inputs, ", ") + "})", classes: classes,
		arm: fmt.Sprintf("| %s(a) => %s(%s)", tag, f, strings.Join(args, ", ")), action: a}, nil
}

// ends writes the body of a machine's or a composition's `ends` over the state `s`: a function of one
// state written in place, or false for a declaration with none.
func (q *quint) ends(e *umpirespb.Expr, owner string) (string, error) {
	if e == nil {
		return "false", nil
	}
	l := e.GetLambda()
	if l == nil || len(l.GetParams()) != 1 {
		return "", q.unsupported(e.GetPosition(), "the ends of %s, which is no function of one state written in place", owner)
	}
	return q.expr(l.GetBody(), scope{l.GetParams()[0].GetName(): "s"})
}

// stepType is the Quint types of a machine's state and of its step record.
func (q *quint) stepType(decl *umpirespb.Machine) (state, step string, err error) {
	if state, err = q.typeRef(named(decl.GetStateType())); err != nil {
		return "", "", err
	}
	outcome, err := q.typeRef(named(decl.GetOutcomeType()))
	if err != nil {
		return "", "", err
	}
	// A machine that names no fact type records none: its lists of facts are empty, of any type.
	fact := "str"
	if decl.GetFactType() != "" {
		if fact, err = q.typeRef(named(decl.GetFactType())); err != nil {
			return "", "", err
		}
	}
	return state, fmt.Sprintf("{f_outcome: %s, f_state: %s, f_facts: List[%s], f_because: str}", outcome, state, fact), nil
}

// machine writes one machine: its classes, its step, its starts and ends, the states its starts reach
// and its rows from them, the same for the product with its monitors, and what its Properties say of
// every step. It gives the type of the machine's part of the dump.
func (q *quint) machine(i int, mm *umpiremodel.Machine) (string, error) {
	decl := mm.Decl
	state, step, err := q.stepType(decl)
	if err != nil {
		return "", err
	}
	var variants, classes, arms []string
	var actions []*umpirespb.Action
	for j, b := range decl.GetSteps() {
		bound, err := q.binding(i, j, b)
		if err != nil {
			return "", err
		}
		variants, classes, arms = append(variants, bound.variant), append(classes, bound.classes), append(arms, bound.arm)
		actions = append(actions, bound.action)
	}
	q.x.bindings = append(q.x.bindings, actions)
	written, err := q.exprs(decl.GetStarts(), scope{})
	if err != nil {
		return "", err
	}
	starts := strings.Join(written, ", ")
	ends, err := q.ends(decl.GetEnds(), decl.GetName())
	if err != nil {
		return "", err
	}
	m := fmt.Sprintf("m%d", i)
	w := &q.out
	fmt.Fprintf(w, "\n  // The machine %s.\n", decl.GetName())
	fmt.Fprintf(w, "  type K%d =\n    | %s\n", i, strings.Join(variants, "\n    | "))
	fmt.Fprintf(w, "  pure val %s_classes: Set[K%d] = %s\n", m, i, strings.Join(classes, ".union(")+strings.Repeat(")", len(classes)-1))
	fmt.Fprintf(w, "  pure def %s_step(s, c) = match c {\n    %s\n  }\n", m, strings.Join(arms, "\n    "))
	fmt.Fprintf(w, "  pure val %s_starts = [%s]\n", m, starts)
	fmt.Fprintf(w, "  pure def %s_ends(s) = %s\n", m, ends)
	if q.only != "" {
		// A check module has the machine's definitions and no dump of it.
		if len(mm.Monitors) == 0 {
			return "", nil
		}
		if _, err := q.monitors(i, mm, state, starts); err != nil {
			return "", err
		}
		q.stateful(i, mm, state, step, starts)
		return "", nil
	}
	fmt.Fprintf(w, "  pure def %s_succ(s) = %s_classes.fold(Set(), (acc, c) => %s_step(s, c).foldl(acc, (a, r) => a.union(Set(r.f_state))))\n", m, m, m)
	q.reach(m+"_bfs", "Set("+starts+")", m+"_succ", depth(mm.Table.Starts, func(s string) []string {
		var out []string
		for _, row := range mm.Table.RowsFrom(s) {
			for _, r := range row.Results {
				out = append(out, r.State)
			}
		}
		return out
	})+1)
	typ := fmt.Sprintf("starts: List[%s], reach: Set[%s], closed: bool, ends: Set[%s], classes: Set[K%d], rows: Set[{src: %s, by: Set[{cls: K%d, steps: List[%s]}]}]",
		state, state, state, i, state, i, step)
	view := fmt.Sprintf("starts: %s_starts, reach: %s_bfs.seen, closed: %s_bfs.frontier == Set(), ends: %s_bfs.seen.filter(s => %s_ends(s)), classes: %s_classes,\n"+
		"    rows: %s_bfs.seen.map(s => {src: s, by: %s_classes.map(c => {cls: c, steps: %s_step(s, c)})})", m, m, m, m, m, m, m, m, m)
	if len(mm.Monitors) > 0 {
		productType, err := q.monitors(i, mm, state, starts)
		if err != nil {
			return "", err
		}
		q.stateful(i, mm, state, step, starts)
		typ += ", product: " + productType
		view += fmt.Sprintf(",\n    product: %s_product", m)
	}
	if properties := q.s.properties(decl.GetName()); len(properties) > 0 {
		claimsType, err := q.claims(m, fmt.Sprintf("K%d", i), properties, state, func(p *umpirespb.Property) (string, error) { return q.about(i, mm, p) })
		if err != nil {
			return "", err
		}
		typ += ", claims: " + claimsType
		view += fmt.Sprintf(",\n    claims: %s_bfs.seen.map(s => {src: s, by: %s_classes.map(c => {cls: c, steps: %s_step(s, c).foldl([], (l, r) => l.append(%s_claims(s, c, r)))})})", m, m, m, m)
	}
	fmt.Fprintf(w, "  pure val %s_view = {%s}\n", m, view)
	return "{" + typ + "}", nil
}

func named(name string) *umpirespb.TypeRef {
	return &umpirespb.TypeRef{Ref: &umpirespb.TypeRef_Named{Named: name}}
}

// reach writes the states a set of starts reaches by a successor function: as many rounds of
// successors as Go's own table is deep, and one more. The dump says whether that was the end.
func (q *quint) reach(name, starts, succ string, rounds int) {
	q.fresh++
	fmt.Fprintf(&q.out, "  pure val %s = 1.to(%d).fold({seen: %s, frontier: %s}, (acc, n) => {\n"+
		"    pure val fresh%d = acc.frontier.map(s => %s(s)).flatten().exclude(acc.seen)\n"+
		"    {seen: acc.seen.union(fresh%d), frontier: fresh%d}\n  })\n", name, rounds, starts, starts, q.fresh, succ, q.fresh, q.fresh)
}

// monitors writes the product of a machine and the monitors it names: the monitors' states advance
// with every step by each monitor's `next`, and each step says, for each monitor, whether its verdict
// is read there and whether its state violates it. It gives the type of the product's dump.
func (q *quint) monitors(i int, mm *umpiremodel.Machine, state, starts string) (string, error) {
	m := fmt.Sprintf("m%d", i)
	var initial, next, read, viol, mu, flags []string
	for k, mo := range mm.Monitors {
		t, err := q.typeRef(mo.GetState())
		if err != nil {
			return "", err
		}
		first, err := q.expr(mo.GetInitial(), scope{})
		if err != nil {
			return "", err
		}
		advance, err := q.function(mo.GetNext(), mo.GetPosition())
		if err != nil {
			return "", err
		}
		violated, err := q.function(mo.GetViolated(), mo.GetPosition())
		if err != nil {
			return "", err
		}
		at := "true"
		switch e := mo.GetEvaluate().(type) {
		case *umpirespb.Monitor_EveryStep:
		case *umpirespb.Monitor_AtEnds:
			at = m + "_ends(r.f_state)"
		case *umpirespb.Monitor_After:
			after, err := q.function(e.After, mo.GetPosition())
			if err != nil {
				return "", err
			}
			at = after + "(r)"
		default:
			return "", q.unsupported(mo.GetPosition(), "the monitor %s, which names no evaluation point", mo.GetName())
		}
		field := fmt.Sprintf("m%d", k)
		initial = append(initial, field+": "+first)
		next = append(next, fmt.Sprintf("%s: %s(mu.%s, s, r)", field, advance, field))
		read = append(read, field+": "+at)
		viol = append(viol, fmt.Sprintf("%s: %s(mu.%s)", field, violated, field))
		mu = append(mu, field+": "+t)
		flags = append(flags, field+": bool")
	}
	w := &q.out
	fmt.Fprintf(w, "  pure val %s_mu = {%s}\n", m, strings.Join(initial, ", "))
	fmt.Fprintf(w, "  pure def %s_mnext(mu, s, r) = {%s}\n", m, strings.Join(next, ", "))
	fmt.Fprintf(w, "  pure def %s_mread(r) = {%s}\n", m, strings.Join(read, ", "))
	fmt.Fprintf(w, "  pure def %s_mviol(mu) = {%s}\n", m, strings.Join(viol, ", "))
	if q.only != "" {
		q.mu = strings.Join(mu, ", ")
		return "", nil
	}
	fmt.Fprintf(w, "  pure val %s_pstarts = %s_starts.foldl([], (l, s) => l.append({s: s, mu: %s_mu}))\n", m, m, m)
	fmt.Fprintf(w, "  pure def %s_psucc(p) = %s_classes.fold(Set(), (acc, c) => %s_step(p.s, c).foldl(acc, (a, r) => a.union(Set({s: r.f_state, mu: %s_mnext(p.mu, p.s, r)}))))\n", m, m, m, m)
	p, err := q.s.product(mm)
	if err != nil {
		return "", err
	}
	var keys []string
	for _, ps := range p.Starts {
		keys = append(keys, ps.key())
	}
	rounds := depth(keys, func(key string) []string { return p.successors[key] }) + 1
	q.reach(m+"_pbfs", fmt.Sprintf("Set(%s).map(s => {s: s, mu: %s_mu})", starts, m), m+"_psucc", rounds)
	fmt.Fprintf(w, "  pure val %s_product = {starts: %s_pstarts, closed: %s_pbfs.frontier == Set(),\n"+
		"    edges: %s_pbfs.seen.map(p => {src: p, by: %s_classes.map(c => {cls: c, steps: %s_step(p.s, c).foldl([], (l, r) =>\n"+
		"      l.append({mu: %s_mnext(p.mu, p.s, r), read: %s_mread(r), viol: %s_mviol(%s_mnext(p.mu, p.s, r))}))})})}\n",
		m, m, m, m, m, m, m, m, m, m)
	q.mu = strings.Join(mu, ", ")
	pstate := fmt.Sprintf("{s: %s, mu: {%s}}", state, q.mu)
	return fmt.Sprintf("{starts: List[%s], closed: bool, edges: Set[{src: %s, by: Set[{cls: K%d, steps: List[{mu: {%s}, read: {%s}, viol: {%s}}]}]}]}",
		pstate, pstate, i, strings.Join(mu, ", "), strings.Join(flags, ", "), strings.Join(flags, ", ")), nil
}

// claims writes what each Property of a machine says of one step: whether it is about the step, by
// the step's class, and whether it holds of it, which is read only where it is about it. It gives
// the type of the claims' dump.
func (q *quint) claims(prefix, class string, properties []*umpirespb.Property, state string, about func(*umpirespb.Property) (string, error)) (string, error) {
	var reads, fields []string
	for n, p := range properties {
		holds, err := q.function(p.GetHolds(), p.GetPosition())
		if err != nil {
			return "", err
		}
		when, err := about(p)
		if err != nil {
			return "", err
		}
		call := holds + "(r)"
		if p.GetTransition() {
			call = holds + "(s, r)"
		}
		name := q.local("about")
		reads = append(reads, fmt.Sprintf("p%d: { pure val %s = %s\n      {about: %s, holds: if (%s) %s else true} }", n, name, when, name, name, call))
		fields = append(fields, fmt.Sprintf("p%d: {about: bool, holds: bool}", n))
	}
	fmt.Fprintf(&q.out, "  pure def %s_claims(s, c, r) = {\n    %s\n  }\n", prefix, strings.Join(reads, ",\n    "))
	return fmt.Sprintf("Set[{src: %s, by: Set[{cls: %s, steps: List[{%s}]}]}]", state, class, strings.Join(fields, ", ")), nil
}

// about writes whether a Property is about the step of the class `c`: every step, the steps of one
// class, by the action's id and its inputs, or the steps of every class of one action, by its name.
func (q *quint) about(i int, mm *umpiremodel.Machine, p *umpirespb.Property) (string, error) {
	steps := mm.Decl.GetSteps()
	variant := func(bound func(*umpirespb.Action) bool) (string, *umpirespb.Action, error) {
		j := slices.IndexFunc(steps, func(b *umpirespb.StepBinding) bool { return bound(q.s.actions[b.GetAction()]) })
		if j < 0 {
			return "", nil, q.unsupported(p.GetPosition(), "the Property %s, about an action %s does not bind", p.GetName(), mm.Decl.GetName())
		}
		return fmt.Sprintf("K%d_%d", i, j), q.s.actions[steps[j].GetAction()], nil
	}
	switch w := p.GetWhen().(type) {
	case *umpirespb.Property_WhenClass:
		tag, _, err := variant(func(a *umpirespb.Action) bool { return a.GetId() == w.WhenClass.GetAction() })
		if err != nil || len(w.WhenClass.GetInputs()) == 0 {
			return fmt.Sprintf("(c == %s)", tag), err
		}
		record := make([]string, len(w.WhenClass.GetInputs()))
		for k, input := range w.WhenClass.GetInputs() {
			written, err := q.literal(input, p.GetPosition())
			if err != nil {
				return "", err
			}
			record[k] = fmt.Sprintf("i%d: %s", k, written)
		}
		return fmt.Sprintf("(c == %s({%s}))", tag, strings.Join(record, ", ")), nil
	case *umpirespb.Property_WhenAction:
		tag, action, err := variant(func(a *umpirespb.Action) bool { return a.GetName() == w.WhenAction })
		if err != nil || len(action.GetInputs()) == 0 {
			return fmt.Sprintf("(c == %s)", tag), err
		}
		return fmt.Sprintf("match c { | %s(_) => true | _ => false }", tag), nil
	default:
		return "true", nil
	}
}

// stateful writes what a check module declares for a machine in place of the dump: the machine as a
// transition system a model checker explores. Its state is the machine's state, its monitors'
// states, whether each monitor was read and violated on the last step, and the steps taken so far, so
// that a counterexample carries its own path. A step takes any class and any result of its row.
func (q *quint) stateful(i int, mm *umpiremodel.Machine, state, step, starts string) {
	m := fmt.Sprintf("m%d", i)
	var flags, unset, bad, invariants []string
	for k := range mm.Monitors {
		flags = append(flags, fmt.Sprintf("m%d: bool", k))
		unset = append(unset, fmt.Sprintf("m%d: false", k))
		bad = append(bad, fmt.Sprintf("m%d: rd.m%d and vl.m%d", k, k, k))
		invariants = append(invariants, fmt.Sprintf("  val inv_m%d = not(bad.m%d)\n", k, k))
	}
	q.x.stateful[mm.Decl.GetName()] = fmt.Sprintf("\n  var st: %s\n  var mu: {%s}\n  var bad: {%s}\n  var hist: List[{cls: K%d, step: %s}]\n"+
		"  action init = {\n    nondet s0 = oneOf(Set(%s))\n    all { st' = s0, mu' = %s_mu, bad' = {%s}, hist' = [] }\n  }\n"+
		"  action step = {\n    nondet c = oneOf(%s_classes)\n    val rs = %s_step(st, c)\n    nondet n = oneOf(rs.indices())\n    val r = rs[n]\n"+
		"    val after = %s_mnext(mu, st, r)\n    val rd = %s_mread(r)\n    val vl = %s_mviol(after)\n"+
		"    all { st' = r.f_state, mu' = after, bad' = {%s}, hist' = hist.append({cls: c, step: r}) }\n  }\n%s",
		state, q.mu, strings.Join(flags, ", "), i, step, starts, m, strings.Join(unset, ", "), m, m, m, m, m, strings.Join(bad, ", "), strings.Join(invariants, ""))
}

// QuintCheck is one machine of an export as a module a model checker explores, with one invariant
// for each monitor the machine names: that the monitor was not read and violated on the last step.
type QuintCheck struct {
	Module  string
	Text    string
	Machine string
	// Monitors names the machine's monitors: the invariant of the one at index k is `inv_m<k>`.
	Monitors []string
	// Steps is how many steps reach every state of Go's product of the machine and its monitors.
	Steps int

	from  *QuintExport
	index int
}

// Check is the module a model checker explores one of the export's machines by. A machine that names
// no monitor has nothing to check.
func (x *QuintExport) Check(machine string) (*QuintCheck, error) {
	i := slices.Index(x.Machines, machine)
	if i < 0 || x.stateful[machine] == "" {
		return nil, fmt.Errorf("the export has no machine %s with monitors", machine)
	}
	mm := x.from.machines[machine]
	p, err := x.from.product(mm)
	if err != nil {
		return nil, err
	}
	alone, err := x.from.quint(machine)
	if err != nil {
		return nil, err
	}
	c := &QuintCheck{Module: "umpire_check", Machine: machine, Monitors: p.Monitors, from: x, index: i, Steps: p.depth + 1}
	c.Text = fmt.Sprintf("// Written by model/backends from %s. Do not edit.\nmodule %s {\n%s%s}\n",
		x.from.Model.GetSource(), c.Module, alone.definitions, alone.stateful[machine])
	return c, nil
}
