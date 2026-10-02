package export

import (
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"

	umpirespb "go.temporal.io/server/api/umpire/v1"
	umpiremodel "go.temporal.io/server/tools/umpire/model"
)

// reader reads one machine's part of an export's output back as keys, by the IR's types. The output
// is ITF: a record is an object, a list an array, a set `{"#set": […]}`, an integer
// `{"#bigint": "n"}`, and a variant `{"tag": …, "value": …}`.
type reader struct {
	x     *QuintExport
	index int
	mm    *umpiremodel.Machine
	// composed is set for a composition's part, whose states, classes and results are read as the
	// composed table keys them.
	composed *composedExport
}

func (x *QuintExport) reader(i int) reader {
	return reader{x: x, index: i, mm: x.from.machines[x.Machines[i]]}
}

// record reads an ITF record.
func record(raw any, what string) (map[string]any, error) {
	fields, ok := raw.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("%s is no record", what)
	}
	return fields, nil
}

func boolean(raw any, what string) (bool, error) {
	b, ok := raw.(bool)
	if !ok {
		return false, fmt.Errorf("%s is no Boolean", what)
	}
	return b, nil
}

// text reads an ITF string, and anything else as none.
func text(raw any) string {
	s, ok := raw.(string)
	if !ok {
		return ""
	}
	return s
}

// list reads an ITF list or set as its elements.
func list(raw any) ([]any, error) {
	if set, ok := raw.(map[string]any); ok {
		raw = set["#set"]
	}
	items, ok := raw.([]any)
	if !ok {
		return nil, fmt.Errorf("%v is no list and no set", raw)
	}
	return items, nil
}

func each[T any](raw any, read func(any) (T, error)) ([]T, error) {
	items, err := list(raw)
	if err != nil {
		return nil, err
	}
	out := make([]T, 0, len(items))
	for _, item := range items {
		v, err := read(item)
		if err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, nil
}

// bySource reads a set of `{src, by}` records, each `by` a set of `{cls, steps}` records, as the
// steps by source and class key.
func bySource[T any](raw any, source func(any) (string, error), class func(any) (string, error), step func(any) (T, error)) (
	map[string]map[string][]T, error) {
	rows, err := list(raw)
	if err != nil {
		return nil, err
	}
	out := map[string]map[string][]T{}
	for _, raw := range rows {
		row, err := record(raw, "a row")
		if err != nil {
			return nil, err
		}
		from, err := source(row["src"])
		if err != nil {
			return nil, err
		}
		pairs, err := list(row["by"])
		if err != nil {
			return nil, err
		}
		out[from] = map[string][]T{}
		for _, raw := range pairs {
			pair, err := record(raw, "a pair of a class and its steps")
			if err != nil {
				return nil, err
			}
			key, err := class(pair["cls"])
			if err != nil {
				return nil, err
			}
			if out[from][key], err = each(pair["steps"], step); err != nil {
				return nil, fmt.Errorf("%s by %s: %w", from, key, err)
			}
		}
	}
	return out, nil
}

// machine reads a machine's part of a dump.
func (r reader) machine(part map[string]any) (*machineView, error) {
	v := &machineView{}
	var err error
	if v.Starts, err = each(part["starts"], r.state); err != nil {
		return nil, fmt.Errorf("starts: %w", err)
	}
	if v.Reach, err = each(part["reach"], r.state); err != nil {
		return nil, fmt.Errorf("reach: %w", err)
	}
	if v.Ends, err = each(part["ends"], r.state); err != nil {
		return nil, fmt.Errorf("ends: %w", err)
	}
	if v.Classes, err = each(part["classes"], r.class); err != nil {
		return nil, fmt.Errorf("classes: %w", err)
	}
	slices.Sort(v.Reach)
	slices.Sort(v.Ends)
	slices.Sort(v.Classes)
	if v.Closed, err = boolean(part["closed"], "closed"); err != nil {
		return nil, err
	}
	if v.Rows, err = bySource(part["rows"], r.state, r.class, r.result); err != nil {
		return nil, fmt.Errorf("rows: %w", err)
	}
	if raw, ok := part["claims"]; ok {
		if v.Claims, err = bySource(raw, r.state, r.class, r.claims); err != nil {
			return nil, fmt.Errorf("claims: %w", err)
		}
	}
	if raw, ok := part["product"]; ok {
		if v.Product, err = r.product(raw); err != nil {
			return nil, fmt.Errorf("product: %w", err)
		}
	}
	return v, nil
}

func (r reader) state(raw any) (string, error) {
	if r.composed != nil {
		return r.composedState(raw)
	}
	v, err := r.x.value(raw, named(r.mm.Decl.GetStateType()))
	return v.Key(), err
}

// class reads a class back as its key: the action's name, and the key of each input.
func (r reader) class(raw any) (string, error) {
	if r.composed != nil {
		return r.composedClass(raw)
	}
	tagged, err := record(raw, "a class")
	if err != nil {
		return "", err
	}
	tag, bindings := text(tagged["tag"]), r.x.bindings[r.index]
	j, err := strconv.Atoi(strings.TrimPrefix(tag, fmt.Sprintf("K%d_", r.index)))
	if err != nil || j < 0 || j >= len(bindings) {
		return "", fmt.Errorf("%q is no class of the machine", tag)
	}
	parts := []string{bindings[j].GetName()}
	for n, p := range bindings[j].GetInputs() {
		inputs, err := record(tagged["value"], "a class's inputs")
		if err != nil {
			return "", err
		}
		v, err := r.x.value(inputs[fmt.Sprintf("i%d", n)], p.GetType())
		if err != nil {
			return "", err
		}
		parts = append(parts, v.Key())
	}
	return strings.Join(parts, "-"), nil
}

// result reads one step record back as keys.
func (r reader) result(raw any) (result, error) {
	if r.composed != nil {
		return r.composedResult(raw)
	}
	decl := r.mm.Decl
	fields, err := record(raw, "a step")
	if err != nil {
		return result{}, err
	}
	outcome, err := r.x.value(fields["f_outcome"], named(decl.GetOutcomeType()))
	if err != nil {
		return result{}, err
	}
	target, err := r.state(fields["f_state"])
	if err != nil {
		return result{}, err
	}
	facts := []string{}
	if decl.GetFactType() != "" {
		if facts, err = each(fields["f_facts"], func(raw any) (string, error) {
			f, err := r.x.value(raw, named(decl.GetFactType()))
			return f.Key(), err
		}); err != nil {
			return result{}, err
		}
	}
	return result{Outcome: outcome.Key(), State: target, Facts: facts, Because: text(fields["f_because"])}, nil
}

// claims reads what each Property of the machine says of one step.
func (r reader) claims(raw any) ([]claimRead, error) {
	fields, err := record(raw, "a step's readings")
	if err != nil {
		return nil, err
	}
	owner := ""
	if r.composed != nil {
		owner = r.composed.decl.GetName()
	} else {
		owner = r.mm.Decl.GetName()
	}
	reads := make([]claimRead, len(r.x.from.properties(owner)))
	for k := range reads {
		read, err := record(fields[fmt.Sprintf("p%d", k)], fmt.Sprintf("the reading of Property %d", k))
		if err != nil {
			return nil, err
		}
		if reads[k].About, err = boolean(read["about"], "whether a Property is about a step"); err != nil {
			return nil, err
		}
		if reads[k].Holds, err = boolean(read["holds"], "whether a Property holds"); err != nil {
			return nil, err
		}
	}
	return reads, nil
}

// product reads the product of the machine and its monitors.
func (r reader) product(raw any) (*productView, error) {
	part, err := record(raw, "the product")
	if err != nil {
		return nil, err
	}
	p := &productView{States: map[string]productState{}}
	if p.Starts, err = each(part["starts"], r.productState); err != nil {
		return nil, err
	}
	if p.Closed, err = boolean(part["closed"], "closed"); err != nil {
		return nil, err
	}
	source := func(raw any) (string, error) {
		ps, err := r.productState(raw)
		p.States[ps.key()] = ps
		return ps.key(), err
	}
	p.Steps, err = bySource(part["edges"], source, r.class, r.productStep)
	return p, err
}

func (r reader) productState(raw any) (productState, error) {
	fields, err := record(raw, "a product state")
	if err != nil {
		return productState{}, err
	}
	state, err := r.state(fields["s"])
	if err != nil {
		return productState{}, err
	}
	mu, err := r.monitors(fields["mu"])
	return productState{State: state, Mu: mu}, err
}

func (r reader) productStep(raw any) (productStep, error) {
	var st productStep
	fields, err := record(raw, "a product step")
	if err != nil {
		return st, err
	}
	if st.Mu, err = r.monitors(fields["mu"]); err != nil {
		return st, err
	}
	if st.Read, err = r.flags(fields["read"]); err != nil {
		return st, err
	}
	st.Viol, err = r.flags(fields["viol"])
	return st, err
}

// monitors reads the monitors' states, one field per monitor in the machine's order.
func (r reader) monitors(raw any) ([]string, error) {
	fields, err := record(raw, "the monitors' states")
	if err != nil {
		return nil, err
	}
	out := make([]string, len(r.mm.Monitors))
	for k, mo := range r.mm.Monitors {
		v, err := r.x.value(fields[fmt.Sprintf("m%d", k)], mo.GetState())
		if err != nil {
			return nil, err
		}
		out[k] = v.Key()
	}
	return out, nil
}

func (r reader) flags(raw any) ([]bool, error) {
	fields, err := record(raw, "the monitors' flags")
	if err != nil {
		return nil, err
	}
	out := make([]bool, len(r.mm.Monitors))
	for k := range r.mm.Monitors {
		if out[k], err = boolean(fields[fmt.Sprintf("m%d", k)], "a monitor's flag"); err != nil {
			return nil, err
		}
	}
	return out, nil
}

// value reads an ITF value back as a value of an IR type.
func (x *QuintExport) value(raw any, t *umpirespb.TypeRef) (umpiremodel.Value, error) {
	switch r := t.GetRef().(type) {
	case *umpirespb.TypeRef_Bool:
		b, err := boolean(raw, fmt.Sprint(raw))
		return umpiremodel.Value{Kind: umpiremodel.BoolValue, Bool: b}, err
	case *umpirespb.TypeRef_IntRange, *umpirespb.TypeRef_Int:
		n, err := integer(raw)
		return umpiremodel.Value{Kind: umpiremodel.IntValue, Int: n}, err
	case *umpirespb.TypeRef_List:
		items, err := each(raw, func(item any) (umpiremodel.Value, error) { return x.value(item, r.List) })
		return umpiremodel.Value{Kind: umpiremodel.ListValue, Items: items}, err
	case *umpirespb.TypeRef_Named:
		return x.declared(raw, r.Named)
	default:
		return umpiremodel.Value{}, errors.New("a channel's contents are not exported")
	}
}

func integer(raw any) (int64, error) {
	if big, ok := raw.(map[string]any); ok {
		raw = big["#bigint"]
	}
	switch num := raw.(type) {
	case string:
		return strconv.ParseInt(num, 10, 64)
	case float64:
		return int64(num), nil
	default:
		return 0, fmt.Errorf("%v is no integer", raw)
	}
}

// declared reads a value of a declared type: a record by its fields, an enum by its variant's tag.
func (x *QuintExport) declared(raw any, name string) (umpiremodel.Value, error) {
	decl, ok := x.from.types[name]
	if !ok {
		return umpiremodel.Value{}, fmt.Errorf("the Model declares no type %s", name)
	}
	if decl.GetRecord() != nil {
		fields, err := x.fields(decl.GetRecord().GetFields(), raw, name)
		return umpiremodel.Value{Kind: umpiremodel.RecordValue, Type: name, Fields: fields}, err
	}
	tagged, err := record(raw, fmt.Sprintf("%v, read as a case of %s,", raw, name))
	if err != nil {
		return umpiremodel.Value{}, err
	}
	variant, ok := x.tags[text(tagged["tag"])]
	if !ok || variant.typ != name {
		return umpiremodel.Value{}, fmt.Errorf("%v is no case of %s", raw, name)
	}
	fields, err := x.fields(variant.enum.GetFields(), tagged["value"], name)
	return umpiremodel.Value{Kind: umpiremodel.EnumValue, Type: name, Case: variant.enum.GetName(), Fields: fields}, err
}

// fields reads a record's or a case's fields in declaration order. A case without fields has none to
// read, whatever its variant carries.
func (x *QuintExport) fields(decls []*umpirespb.Field, raw any, name string) ([]umpiremodel.Value, error) {
	if len(decls) == 0 {
		return nil, nil
	}
	values, err := record(raw, fmt.Sprintf("%v, read as the fields of %s,", raw, name))
	if err != nil {
		return nil, err
	}
	out := make([]umpiremodel.Value, len(decls))
	for n, f := range decls {
		if out[n], err = x.value(values["f_"+plain(f.GetName())], f.GetType()); err != nil {
			return nil, fmt.Errorf("%s: %w", f.GetName(), err)
		}
	}
	return out, nil
}
