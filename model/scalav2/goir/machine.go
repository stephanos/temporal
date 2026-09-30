package goir

import (
	"slices"
	"strings"

	umpirespb "go.temporal.io/server/api/umpire/v1"
	"go.temporal.io/server/model/go/umpire"
)

// Members lists every value of a finite type in catalog order: an enum's cases in declaration order,
// each case with fields once per assignment of them; a record as the product of its fields; with the
// last field varying fastest in both, which is the order the Lean `Finite` derivation produces.
func (in *Interpreter) Members(t *umpirespb.TypeRef) ([]Value, error) {
	switch r := t.GetRef().(type) {
	case *umpirespb.TypeRef_Bool:
		return []Value{{Kind: BoolValue, Bool: false}, {Kind: BoolValue, Bool: true}}, nil
	case *umpirespb.TypeRef_IntRange:
		var out []Value
		for i := r.IntRange.GetLow(); i <= r.IntRange.GetHigh(); i++ {
			out = append(out, Value{Kind: IntValue, Int: i})
		}
		return out, nil
	case *umpirespb.TypeRef_Named:
		decl, ok := in.types[r.Named]
		if !ok {
			return nil, &Error{Message: "no type " + r.Named}
		}
		switch s := decl.GetShape().(type) {
		case *umpirespb.Type_Enum:
			var out []Value
			for _, c := range s.Enum.GetCases() {
				assignments, err := in.product(c.GetFields())
				if err != nil {
					return nil, err
				}
				for _, fields := range assignments {
					out = append(out, Value{Kind: EnumValue, Type: r.Named, Case: c.GetName(), Fields: fields})
				}
			}
			return out, nil
		case *umpirespb.Type_Record:
			assignments, err := in.product(s.Record.GetFields())
			if err != nil {
				return nil, err
			}
			out := make([]Value, len(assignments))
			for i, fields := range assignments {
				out[i] = Value{Kind: RecordValue, Type: r.Named, Fields: fields}
			}
			return out, nil
		default:
			return nil, errorAt(decl.GetPosition(), "type %s has no shape", r.Named)
		}
	default:
		return nil, &Error{Message: "a finite type is a named type, the Booleans, or an integer range"}
	}
}

func (in *Interpreter) product(fields []*umpirespb.Field) ([][]Value, error) {
	out := [][]Value{{}}
	for _, f := range fields {
		members, err := in.Members(f.GetType())
		if err != nil {
			return nil, err
		}
		next := make([][]Value, 0, len(out)*len(members))
		for _, prefix := range out {
			for _, m := range members {
				next = append(next, append(slices.Clone(prefix), m))
			}
		}
		out = next
	}
	return out, nil
}

func named(name string) *umpirespb.TypeRef {
	return &umpirespb.TypeRef{Ref: &umpirespb.TypeRef_Named{Named: name}}
}

// class is one class of an action: the action and one assignment of its inputs.
type class struct {
	key    string
	inputs []Value
	step   string
	at     *umpirespb.Position
}

// Machine is one machine of the IR, interpreted: its table and, for a refining machine, its
// refinement rows.
type Machine struct {
	Decl       *umpirespb.Machine
	Table      *umpire.Table
	states     map[string]Value
	Refinement []umpire.RefinementRow
}

// Build interprets every machine of a Model. A refining machine's refinement is checked against the
// machine it names, which must be in the Model.
func Build(m *umpirespb.Model) (map[string]*Machine, error) {
	in := NewInterpreter(m)
	actions := map[string]*umpirespb.Action{}
	for _, a := range m.GetActions() {
		actions[a.GetId()] = a
	}
	out := map[string]*Machine{}
	for _, decl := range m.GetMachines() {
		mm, err := in.machine(decl, actions)
		if err != nil {
			return nil, err
		}
		out[decl.GetName()] = mm
	}
	for _, mm := range out {
		if r := mm.Decl.GetRefines(); r != nil {
			product, ok := out[r.GetProduct()]
			if !ok {
				return nil, errorAt(mm.Decl.GetPosition(), "%s refines %s, which the Model does not declare", mm.Decl.GetName(), r.GetProduct())
			}
			rows, err := in.refinement(mm, product)
			if err != nil {
				return nil, err
			}
			mm.Refinement = rows
		}
	}
	return out, nil
}

func (in *Interpreter) machine(decl *umpirespb.Machine, actions map[string]*umpirespb.Action) (*Machine, error) {
	states, err := in.Members(named(decl.GetStateType()))
	if err != nil {
		return nil, err
	}
	outcomes, err := in.Members(named(decl.GetOutcomeType()))
	if err != nil {
		return nil, err
	}
	var facts []Value
	if decl.GetFactType() != "" {
		if facts, err = in.Members(named(decl.GetFactType())); err != nil {
			return nil, err
		}
	}
	classes, err := in.classes(decl, actions)
	if err != nil {
		return nil, err
	}
	spec := umpire.TableSpec{Machine: decl.GetName(), Owner: decl.GetName(), Family: umpire.Family(decl.GetFamily()),
		Entity: decl.GetEntity()}
	mm := &Machine{Decl: decl, states: map[string]Value{}}
	for _, s := range states {
		spec.States = append(spec.States, s.Key())
		mm.states[s.Key()] = s
	}
	for _, c := range classes {
		spec.Actions = append(spec.Actions, c.key)
	}
	for _, o := range outcomes {
		spec.Outcomes = append(spec.Outcomes, o.Key())
	}
	for _, f := range facts {
		spec.Facts = append(spec.Facts, f.Key())
	}
	if spec.Rows, err = in.rows(decl, states, classes, mm.states); err != nil {
		return nil, err
	}
	if spec.Starts, spec.Ends, err = in.startsAndEnds(decl, states, mm.states); err != nil {
		return nil, err
	}
	names, _ := in.fieldNames(decl.GetStateType(), "")
	spec.StateFields = append(spec.StateFields, names...)
	if r := decl.GetRefines(); r != nil {
		// A refining machine carries the refined state it reads as in a field named after the refined
		// machine (`Umpire.Command.refinedProperty`).
		spec.StateFields = append(spec.StateFields, r.GetProduct())
	}
	if spec.Evidence, err = in.evidence(decl, facts); err != nil {
		return nil, err
	}
	mm.Table = umpire.NewTable(spec)
	return mm, nil
}

// classes lists every class of every bound action, sorted by key, and rejects a class two steps bind.
func (in *Interpreter) classes(decl *umpirespb.Machine, actions map[string]*umpirespb.Action) ([]class, error) {
	var out []class
	for _, b := range decl.GetSteps() {
		a, ok := actions[b.GetAction()]
		if !ok {
			return nil, errorAt(b.GetPosition(), "no action %s", b.GetAction())
		}
		fields := make([]*umpirespb.Field, len(a.GetInputs()))
		for i, p := range a.GetInputs() {
			fields[i] = &umpirespb.Field{Name: p.GetName(), Type: p.GetType()}
		}
		assignments, err := in.product(fields)
		if err != nil {
			return nil, err
		}
		for _, inputs := range assignments {
			parts := []string{a.GetName()}
			for _, v := range inputs {
				parts = append(parts, v.Key())
			}
			out = append(out, class{key: strings.Join(parts, "-"), inputs: inputs, step: b.GetFunction(), at: b.GetPosition()})
		}
	}
	slices.SortStableFunc(out, func(x, y class) int { return strings.Compare(x.key, y.key) })
	for i := 1; i < len(out); i++ {
		if out[i].key == out[i-1].key {
			return nil, errorAt(out[i].at, "%s: two steps bind the action class %q", decl.GetName(), out[i].key)
		}
	}
	return out, nil
}

// rows evaluates every step function once per state and class, states-major, keeping the enabled
// pairs as rows and rejecting a result outside the state domain.
func (in *Interpreter) rows(decl *umpirespb.Machine, states []Value, classes []class, domain map[string]Value) ([]umpire.Row, error) {
	var rows []umpire.Row
	for _, s := range states {
		for _, c := range classes {
			results, err := in.Call(c.step, append([]Value{s}, c.inputs...), c.at)
			if err != nil {
				return nil, err
			}
			if len(results.Items) == 0 {
				continue
			}
			row := umpire.Row{Key: s.Key() + "-" + c.key, Source: s.Key(), Action: c.key}
			for _, step := range results.Items {
				if step.Type != StepType {
					return nil, errorAt(c.at, "%s returns %s, not a list of steps", c.step, step.Type)
				}
				next := step.Fields[1].Key()
				if _, ok := domain[next]; !ok {
					return nil, errorAt(c.at, "%s: row %s lands in %s, which is outside the state domain", decl.GetName(), row.Key, next)
				}
				res := umpire.Result{Outcome: step.Fields[0].Key(), State: next, Facts: []string{}, Because: step.Fields[3].Text}
				for _, f := range step.Fields[2].Items {
					res.Facts = append(res.Facts, f.Key())
				}
				row.Results = append(row.Results, res)
			}
			rows = append(rows, row)
		}
	}
	return rows, nil
}

func (in *Interpreter) startsAndEnds(decl *umpirespb.Machine, states []Value, domain map[string]Value) (starts, ends []string, err error) {
	for _, x := range decl.GetStarts() {
		v, err := in.Eval(x)
		if err != nil {
			return nil, nil, err
		}
		if _, ok := domain[v.Key()]; !ok {
			return nil, nil, errorAt(x.GetPosition(), "start %s is outside the state domain", v.Key())
		}
		starts = append(starts, v.Key())
	}
	if len(starts) == 0 {
		return nil, nil, errorAt(decl.GetPosition(), "%s declares no start", decl.GetName())
	}
	if decl.GetEnds() == nil {
		return starts, nil, nil
	}
	end, evalErr := in.Eval(decl.GetEnds())
	if evalErr != nil {
		return nil, nil, evalErr
	}
	for _, s := range states {
		v, err := in.Apply(end, []Value{s})
		if err != nil {
			return nil, nil, err
		}
		if v.Bool {
			ends = append(ends, s.Key())
		}
	}
	return starts, ends, nil
}

// evidence is one line per fact constructor, in catalog order: the constructor, and the recorded
// event or observation the evidence function names for it.
func (in *Interpreter) evidence(decl *umpirespb.Machine, facts []Value) ([][2]string, error) {
	if decl.GetEvidence() == "" {
		return nil, nil
	}
	var out [][2]string
	for _, f := range facts {
		v, err := in.Call(decl.GetEvidence(), []Value{f}, decl.GetPosition())
		if err != nil {
			return nil, err
		}
		if !slices.ContainsFunc(out, func(l [2]string) bool { return l[0] == f.Case }) {
			out = append(out, [2]string{f.Case, v.Text})
		}
	}
	return out, nil
}

// refinement checks the declared refinement under the rule `Umpire.Command.deriveRefinement`
// applies, as model/go's `Refinement` does: every outcome reads as a product outcome of the same name,
// every start as a product start, and every row result is carried by a product row from the mapped
// source that reaches the mapped target with the same outcome and whose facts all appear among the
// result's facts, preferring the product action of the row's own name, or else the mapped states are
// equal and the result is a stutter.
func (in *Interpreter) refinement(mm, product *Machine) ([]umpire.RefinementRow, error) {
	src, dst := mm.Table, product.Table
	where := mm.Decl.GetName() + " refines " + dst.Machine
	mapKey := func(state string) (string, error) {
		v, err := in.Call(mm.Decl.GetRefines().GetMap(), []Value{mm.states[state]}, mm.Decl.GetPosition())
		return v.Key(), err
	}
	for _, o := range src.Outcomes {
		if !slices.Contains(dst.Outcomes, o) {
			return nil, errorAt(mm.Decl.GetPosition(), "%s: '%s' is an outcome of %s and no outcome of %s has that name",
				where, o, src.Machine, dst.Machine)
		}
	}
	for _, s := range src.Starts {
		mapped, err := mapKey(s)
		if err != nil {
			return nil, err
		}
		if !slices.Contains(dst.Starts, mapped) {
			return nil, errorAt(mm.Decl.GetPosition(), "%s: %s starts at '%s', which reads as '%s', and %s does not start there",
				where, src.Machine, s, mapped, dst.Machine)
		}
	}
	var rows []umpire.RefinementRow
	for _, row := range src.Rows {
		from, err := mapKey(row.Source)
		if err != nil {
			return nil, err
		}
		for _, res := range row.Results {
			to, err := mapKey(res.State)
			if err != nil {
				return nil, err
			}
			if carrier, ok := carrierOf(dst, row, res, from, to); ok {
				rows = append(rows, umpire.RefinementRow{Key: row.Key, Product: &carrier})
				continue
			}
			if from != to {
				return nil, errorAt(mm.Decl.GetPosition(), "%s: the row '%s' steps from '%s' to '%s', which read as '%s' and '%s' in %s, "+
					"and %s has no step between them with outcome '%s', so the row is neither a step of %s nor a stutter",
					where, row.Key, row.Source, res.State, from, to, dst.Machine, dst.Machine, res.Outcome, dst.Machine)
			}
			rows = append(rows, umpire.RefinementRow{Key: row.Key})
		}
	}
	return rows, nil
}

func carrierOf(dst *umpire.Table, row umpire.Row, res umpire.Result, from, to string) (string, bool) {
	var facts []string
	for _, f := range res.Facts {
		if k, ok := sameNamedKey(dst.Facts, f); ok {
			facts = append(facts, k)
		}
	}
	var carriers []string
	for _, c := range dst.RowsFrom(from) {
		if slices.ContainsFunc(c.Results, func(cr umpire.Result) bool {
			return cr.State == to && cr.Outcome == res.Outcome && allIn(cr.Facts, facts)
		}) {
			carriers = append(carriers, c.Action)
		}
	}
	if preferred, ok := sameNamedKey(dst.Actions, row.Action); ok && slices.Contains(carriers, preferred) {
		return preferred, true
	}
	if len(carriers) > 0 {
		return carriers[0], true
	}
	return "", false
}

// sameNamedKey is the product key a key names by default: the same key, or the constructor it
// applies (`Umpire.Command.sameNamedKey`).
func sameNamedKey(product []string, key string) (string, bool) {
	if slices.Contains(product, key) {
		return key, true
	}
	constructor, _, _ := strings.Cut(key, "-")
	if slices.Contains(product, constructor) {
		return constructor, true
	}
	return "", false
}

func allIn(xs, ys []string) bool {
	for _, x := range xs {
		if !slices.Contains(ys, x) {
			return false
		}
	}
	return true
}
