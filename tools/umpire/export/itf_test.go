package export

import (
	"encoding/json"
	"errors"
	"fmt"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	umpirespb "go.temporal.io/server/api/umpire/v1"
	"go.temporal.io/server/tools/umpire/check"
	"go.temporal.io/server/tools/umpire/interp"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/emptypb"
)

// These literal pre-change readers keep successful-path formatting in the oracle.
// Their value traversal is independent of the production value/declared/fields methods.
type legacyITFValue struct{ *QuintExport }

type legacyITFReader struct {
	x     *QuintExport
	index int
	mm    *interp.Machine
	// composed is set for a composition's part, whose states, classes and results are read as the
	// composed table keys them.
	composed *composedExport
}

func (x *QuintExport) legacyReader(i int) legacyITFReader {
	return legacyITFReader{x: x, index: i, mm: x.from.machines[x.Machines[i]]}
}

// machine reads a machine's part of a dump.
func (r legacyITFReader) machine(part map[string]any) (*machineView, error) {
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

func (r legacyITFReader) state(raw any) (string, error) {
	if r.composed != nil {
		return r.composedState(raw)
	}
	v, err := (legacyITFValue{r.x}).value(raw, named(r.mm.Decl.GetStateType()))
	return v.Key(), err
}

// class reads a class back as its key: the action's name, and the key of each input.
func (r legacyITFReader) class(raw any) (string, error) {
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
		v, err := (legacyITFValue{r.x}).value(inputs[fmt.Sprintf("i%d", n)], p.GetType())
		if err != nil {
			return "", err
		}
		parts = append(parts, v.Key())
	}
	return strings.Join(parts, "-"), nil
}

// result reads one step record back as keys.
func (r legacyITFReader) result(raw any) (result, error) {
	if r.composed != nil {
		return r.composedResult(raw)
	}
	decl := r.mm.Decl
	fields, err := record(raw, "a step")
	if err != nil {
		return result{}, err
	}
	outcome, err := (legacyITFValue{r.x}).value(fields["f_outcome"], named(decl.GetOutcomeType()))
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
			f, err := (legacyITFValue{r.x}).value(raw, named(decl.GetFactType()))
			return f.Key(), err
		}); err != nil {
			return result{}, err
		}
	}
	return result{Outcome: outcome.Key(), State: target, Facts: facts, Because: text(fields["f_because"]), Choice: text(fields["f_choice"])}, nil
}

// claims reads what each Property of the machine says of one step.
func (r legacyITFReader) claims(raw any) ([]claimRead, error) {
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
func (r legacyITFReader) product(raw any) (*productView, error) {
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

func (r legacyITFReader) productState(raw any) (productState, error) {
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

func (r legacyITFReader) productStep(raw any) (productStep, error) {
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
func (r legacyITFReader) monitors(raw any) ([]string, error) {
	fields, err := record(raw, "the monitors' states")
	if err != nil {
		return nil, err
	}
	out := make([]string, len(r.mm.Monitors))
	for k, mo := range r.mm.Monitors {
		v, err := (legacyITFValue{r.x}).value(fields[fmt.Sprintf("m%d", k)], mo.GetState())
		if err != nil {
			return nil, err
		}
		out[k] = v.Key()
	}
	return out, nil
}

func (r legacyITFReader) flags(raw any) ([]bool, error) {
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
func (x legacyITFValue) value(raw any, t *umpirespb.TypeRef) (interp.Value, error) {
	switch r := t.GetRef().(type) {
	case *umpirespb.TypeRef_Bool:
		b, err := boolean(raw, fmt.Sprint(raw))
		return interp.Value{Kind: interp.BoolValue, Bool: b}, err
	case *umpirespb.TypeRef_IntRange, *umpirespb.TypeRef_Int:
		n, err := integer(raw)
		return interp.Value{Kind: interp.IntValue, Int: n}, err
	case *umpirespb.TypeRef_List:
		items, err := each(raw, func(item any) (interp.Value, error) { return x.value(item, r.List) })
		return interp.Value{Kind: interp.ListValue, Items: items}, err
	case *umpirespb.TypeRef_Named:
		return x.declared(raw, r.Named)
	default:
		return interp.Value{}, errors.New("a channel's contents are not exported")
	}
}

// declared reads a value of a declared type: a record by its fields, an enum by its variant's tag.
func (x legacyITFValue) declared(raw any, name string) (interp.Value, error) {
	decl, ok := x.from.types[name]
	if !ok {
		return interp.Value{}, fmt.Errorf("the Model declares no type %s", name)
	}
	if decl.GetRecord() != nil {
		fields, err := x.fields(decl.GetRecord().GetFields(), raw, name)
		return interp.Value{Kind: interp.RecordValue, Type: name, Fields: fields}, err
	}
	tagged, err := record(raw, fmt.Sprintf("%v, read as a case of %s,", raw, name))
	if err != nil {
		return interp.Value{}, err
	}
	variant, ok := x.tags[text(tagged["tag"])]
	if !ok || variant.typ != name {
		return interp.Value{}, fmt.Errorf("%v is no case of %s", raw, name)
	}
	fields, err := x.fields(variant.enum.GetFields(), tagged["value"], name)
	return interp.Value{Kind: interp.EnumValue, Type: name, Case: variant.enum.GetName(), Fields: fields}, err
}

// fields reads a record's or a case's fields in declaration order. A case without fields has none to
// read, whatever its variant carries.
func (x legacyITFValue) fields(decls []*umpirespb.Field, raw any, name string) ([]interp.Value, error) {
	if len(decls) == 0 {
		return nil, nil
	}
	values, err := record(raw, fmt.Sprintf("%v, read as the fields of %s,", raw, name))
	if err != nil {
		return nil, err
	}
	out := make([]interp.Value, len(decls))
	for n, f := range decls {
		if out[n], err = x.value(values["f_"+plain(f.GetName())], f.GetType()); err != nil {
			return nil, fmt.Errorf("%s: %w", f.GetName(), err)
		}
	}
	return out, nil
}

func (x *QuintExport) legacyComposedReader(j int) legacyITFReader {
	return legacyITFReader{x: x, index: j, composed: x.composed[j]}
}

// composedState is the key of a composed state: its members' state keys in member order, joined by
// "_".
func (r legacyITFReader) composedState(raw any) (string, error) {
	c := r.composed
	v, err := (legacyITFValue{r.x}).value(raw, named(c.decl.GetStateType()))
	if err != nil {
		return "", err
	}
	fields := r.x.from.types[c.decl.GetStateType()].GetRecord().GetFields()
	parts := make([]string, len(c.members))
	for m, mb := range c.members {
		k := slices.IndexFunc(fields, func(f *umpirespb.Field) bool { return f.GetName() == mb.field })
		if k < 0 {
			return "", fmt.Errorf("no member fills the field %s", mb.field)
		}
		parts[m] = v.Fields[k].Key()
	}
	return strings.Join(parts, "_"), nil
}

// composedClass is the key of a composed class: `<field>_<class>` for a member's own class, and for a
// sync its name followed by each class's inputs.
func (r legacyITFReader) composedClass(raw any) (string, error) {
	c := r.composed
	tagged, err := record(raw, "a composed class")
	if err != nil {
		return "", err
	}
	tag := text(tagged["tag"])
	var kind byte
	var n int
	if _, err := fmt.Sscanf(strings.TrimPrefix(tag, fmt.Sprintf("C%d_", r.index)), "%c%d", &kind, &n); err != nil {
		return "", fmt.Errorf("%q is no class of the composition", tag)
	}
	switch {
	case kind == 'o' && n >= 0 && n < len(c.members):
		key, err := r.x.legacyReader(c.members[n].machine).class(tagged["value"])
		return c.members[n].field + "_" + key, err
	case kind == 's' && n >= 0 && n < len(c.syncs):
		sync := c.syncs[n]
		pair, err := record(tagged["value"], "a sync's classes")
		if err != nil {
			return "", err
		}
		first, err := r.x.legacyReader(c.members[sync.first].machine).class(pair["a"])
		if err != nil {
			return "", err
		}
		second, err := r.x.legacyReader(c.members[sync.second].machine).class(pair["b"])
		return sync.name + strings.TrimPrefix(first, sync.firstAction) + strings.TrimPrefix(second, sync.secondAction), err
	default:
		return "", fmt.Errorf("%q is no class of the composition", tag)
	}
}

// composedResult reads one composed step record: its state, and its outcome and facts as the strings
// a claim reads.
func (r legacyITFReader) composedResult(raw any) (result, error) {
	fields, err := record(raw, "a composed step")
	if err != nil {
		return result{}, err
	}
	target, err := r.composedState(fields["f_state"])
	if err != nil {
		return result{}, err
	}
	facts, err := each(fields["f_facts"], func(raw any) (string, error) {
		f, ok := raw.(string)
		if !ok {
			return "", fmt.Errorf("the composed fact %v is no string", raw)
		}
		return f, nil
	})
	return result{Outcome: text(fields["f_outcome"]), State: target, Facts: facts, Because: text(fields["f_because"]), Choice: text(fields["f_choice"])}, err
}

func (s *Slice) legacyITFAgreement(x *QuintExport, itf []byte) ([]Receipt, error) {
	wanted := make([]string, 0, len(x.Machines)+len(x.Compositions))
	for i := range x.Machines {
		wanted = append(wanted, fmt.Sprintf("m%d", i))
	}
	for j := range x.Compositions {
		wanted = append(wanted, fmt.Sprintf("c%d", j))
	}
	dump, err := indexQuintDump(itf, wanted)
	if err != nil {
		return nil, err
	}
	var receipts []Receipt
	// Every counterexample of one dump is replayed through one interpretation, made afresh for it.
	var fresh *Slice
	replay := func(machine, monitor string, trace *check.Trace) error {
		if fresh == nil {
			var err error
			if fresh, err = Open(s.Model); err != nil {
				return err
			}
		}
		return fresh.replay(machine, monitor, trace)
	}
	for i := range x.Machines {
		key := fmt.Sprintf("m%d", i)
		part, err := dump.part(itf, key)
		if err != nil {
			return nil, err
		}
		compared, err := s.legacyITFMachineReceipts(x, i, map[string]any{key: part}, replay)
		if err != nil {
			return nil, err
		}
		receipts = append(receipts, compared...)
	}
	for j := range x.Compositions {
		key := fmt.Sprintf("c%d", j)
		part, err := dump.part(itf, key)
		if err != nil {
			return nil, err
		}
		compared, err := s.legacyITFCompositionReceipts(x, j, map[string]any{key: part})
		if err != nil {
			return nil, err
		}
		receipts = append(receipts, compared...)
	}
	// What the export leaves out is part of what the comparison says: it is listed, and agrees nothing.
	receipts = append(receipts, x.Unsupported...)
	for i := range receipts {
		receipts[i].Model = s.Name
	}
	return receipts, s.confirm(receipts, true)
}

// legacyITFMachineReceipts compares machine i's part of a dump with Go's reading of the machine.
func (s *Slice) legacyITFMachineReceipts(x *QuintExport, i int, out map[string]any, replay func(machine, monitor string, trace *check.Trace) error) ([]Receipt, error) {
	name := x.Machines[i]
	part, ok := out[fmt.Sprintf("m%d", i)].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("the Quint dump has no part m%d for the machine %s", i, name)
	}
	theirs, err := x.legacyReader(i).machine(part)
	if err != nil {
		return nil, fmt.Errorf("the Quint dump of %s: %w", name, err)
	}
	mm := s.machines[name]
	if mm == nil {
		return nil, fmt.Errorf("the export names the machine %s, which the Model does not declare", name)
	}
	ours, err := s.view(mm)
	if err != nil {
		return nil, err
	}
	receipts := []Receipt{transitions(name, ours, theirs)}
	if ours.Product != nil || theirs.Product != nil {
		receipts = append(receipts, watched(mm, ours, theirs, replay))
	}
	if len(ours.Properties) > 0 || theirs.Claims != nil {
		receipts = append(receipts, claimed(name, ours, theirs))
	}
	return append(receipts, coverage(name, theirs)), nil
}

func TestITFValueReadsPreserveNativeValuesAndRefusals(t *testing.T) {
	b := &umpirespb.TypeRef{Ref: &umpirespb.TypeRef_Bool{Bool: &emptypb.Empty{}}}
	i := &umpirespb.TypeRef{Ref: &umpirespb.TypeRef_Int{Int: &emptypb.Empty{}}}
	bounded := &umpirespb.TypeRef{Ref: &umpirespb.TypeRef_IntRange{IntRange: &umpirespb.IntRange{}}}
	l := &umpirespb.TypeRef{Ref: &umpirespb.TypeRef_List{List: named("R")}}
	fields := []*umpirespb.Field{{Name: "first", Type: b}, {Name: "second", Type: i}}
	cases := []*umpirespb.Case{{Name: "empty"}, {Name: "full", Fields: fields}}
	foreign := &umpirespb.Case{Name: "foreignEmpty"}
	x := &QuintExport{from: &Slice{types: map[string]*umpirespb.Type{
		"R":     {Name: "R", Shape: &umpirespb.Type_Record{Record: &umpirespb.Record{Fields: fields}}},
		"E":     {Name: "E", Shape: &umpirespb.Type_Enum{Enum: &umpirespb.Enum{Cases: cases}}},
		"Other": {Name: "Other", Shape: &umpirespb.Type_Enum{Enum: &umpirespb.Enum{Cases: []*umpirespb.Case{foreign}}}},
		"Empty": {Name: "Empty", Shape: &umpirespb.Type_Record{Record: &umpirespb.Record{}}},
	}}, tags: map[string]variant{"empty": {typ: "E", enum: cases[0]}, "full": {typ: "E", enum: cases[1]}, "foreign": {typ: "Other", enum: foreign}}}
	good := map[string]any{"f_first": true, "f_second": map[string]any{"#bigint": "-9223372036854775808"}, "extra": map[string]any{"escaped\n/雪": []any{false, nil, "<>&"}}}
	var typedNil map[string]any
	for _, specimen := range []struct {
		name string
		typ  *umpirespb.TypeRef
		raw  any
	}{
		{"Boolean", b, true}, {"wrong Boolean", b, "true"}, {"nil Boolean", b, nil},
		{"integer", i, float64(13)}, {"bounded integer", bounded, float64(13)}, {"integer string", i, "17"}, {"integer overflow", i, "9223372036854775808"},
		{"bad integer", i, "雪"}, {"nil integer", i, nil}, {"big integer", i, map[string]any{"#bigint": "-19"}},
		{"record and ignored JSON extras", named("R"), good}, {"record nil", named("R"), nil},
		{"record typed nil", named("R"), typedNil}, {"record array", named("R"), []any{}},
		{"record missing", named("R"), map[string]any{}},
		{"first error order", named("R"), map[string]any{"f_first": "bad", "f_second": "bad"}},
		{"second error", named("R"), map[string]any{"f_first": true, "f_second": "bad"}},
		{"empty record ignores raw", named("Empty"), false},
		{"enum empty ignores payload", named("E"), map[string]any{"tag": "empty", "value": false}},
		{"enum full", named("E"), map[string]any{"tag": "full", "value": good}},
		{"enum wrong fields", named("E"), map[string]any{"tag": "full", "value": nil}},
		{"enum wrong tag", named("E"), map[string]any{"tag": "unknown", "value": good}},
		{"enum other-type tag", named("E"), map[string]any{"tag": "foreign", "value": good}},
		{"enum nontext tag", named("E"), map[string]any{"tag": true}},
		{"enum nil", named("E"), nil}, {"enum typed nil", named("E"), typedNil}, {"enum scalar", named("E"), "雪\n"},
		{"unknown type", named("unknown"), good}, {"wrong type tag", named("R"), map[string]any{"tag": "full"}},
		{"nested list", l, []any{good, good}}, {"nested set", l, map[string]any{"#set": []any{good}}},
		{"empty list", l, []any{}}, {"list first failure", l, []any{nil, good}}, {"wrong list", l, true},
		{"unexported reference", &umpirespb.TypeRef{}, good},
	} {
		t.Run(specimen.name, func(t *testing.T) {
			want, oldErr := (legacyITFValue{x}).value(specimen.raw, specimen.typ)
			got, err := x.value(specimen.raw, specimen.typ)
			require.Equal(t, oldErr, err, "exact native error value/type/unwrap chain")
			require.Equal(t, want, got)
			require.Equal(t, want.Key(), got.Key())
			if specimen.name == "first error order" {
				require.EqualError(t, err, "first: bad is no Boolean")
			}
			if specimen.raw != nil && specimen.name != "record typed nil" && specimen.name != "enum typed nil" {
				encoded, marshalErr := json.Marshal(specimen.raw)
				require.NoError(t, marshalErr)
				var admitted any
				require.NoError(t, json.Unmarshal(encoded, &admitted))
				oldAdmitted, oldErr := (legacyITFValue{x}).value(admitted, specimen.typ)
				currentAdmitted, err := x.value(admitted, specimen.typ)
				require.Equal(t, oldErr, err)
				require.Equal(t, oldAdmitted, currentAdmitted)
			}
		})
	}
	pristine := proto.Clone(x.from.types["R"])
	fields[0].Type = i
	want, oldErr := (legacyITFValue{x}).value(good, named("R"))
	got, err := x.value(good, named("R"))
	require.EqualError(t, err, "first: true is no integer")
	require.Equal(t, oldErr, err)
	require.Equal(t, want, got, "both read the mutated declaration, not cached types")
	fields[0].Type = b
	require.True(t, proto.Equal(pristine, x.from.types["R"]))
}

func TestITFNativeReadersPreserveCompleteReceipts(t *testing.T) {
	for _, fixture := range []struct {
		name string
		open func(*testing.T) *Slice
	}{
		{"monitored record", func(t *testing.T) *Slice { return namedChoices(t, "accepts", "rejects") }},
		{"Nexus composition", func(t *testing.T) *Slice { return openNamed(t, "nexus-workflow") }},
	} {
		t.Run(fixture.name, func(t *testing.T) {
			old := fixture.open(t)
			pristine := proto.Clone(old.Model)
			current := openSlice(t, proto.Clone(old.Model).(*umpirespb.Model))
			current.Name = old.Name
			x, fresh := exported(t, old), exported(t, current)
			require.Equal(t, x.Machines, fresh.Machines)
			require.Equal(t, x.Compositions, fresh.Compositions)
			if fixture.name == "monitored record" {
				require.True(t, slices.ContainsFunc(x.Machines, func(owner string) bool { return len(old.machines[owner].Monitors) > 0 }))
			} else {
				require.NotEmpty(t, x.Compositions)
			}
			parts := dumpPartsOf(t, old, x)
			for _, input := range []struct {
				name   string
				tamper func(string, map[string]any)
			}{
				{"faithful", nil},
				{"open frontier", func(_ string, part map[string]any) { part["closed"] = false }},
				{"typed refusal", func(_ string, part map[string]any) { part["starts"] = false }},
				{"nested state refusal", func(_ string, part map[string]any) { part["starts"] = []any{nil} }},
			} {
				t.Run(input.name, func(t *testing.T) {
					dump := parts.encode(t, input.tamper)
					before := slices.Clone(dump)
					want, oldErr := old.legacyITFAgreement(x, dump)
					got, err := current.QuintAgreement(fresh, dump)
					require.Equal(t, oldErr, err)
					require.Equal(t, want, got, "complete ordered Receipts including replay and confirmation")
					if input.name == "faithful" {
						require.NoError(t, err)
						if fixture.name == "monitored record" {
							require.Len(t, only(t, got, MonitorAgreement, "trustingActivityRecord").Witnesses, 2, "fresh native witness replay is exercised")
						}
					}
					if oldErr == nil {
						require.NotEmpty(t, want)
					}
					require.Equal(t, before, dump)
					require.True(t, proto.Equal(pristine, old.Model))
					require.True(t, proto.Equal(pristine, current.Model))
				})
			}
		})
	}
}

func TestITFDeclaredValuesAvoidUnusedFormatting(t *testing.T) {
	booleanType := &umpirespb.TypeRef{Ref: &umpirespb.TypeRef_Bool{Bool: &emptypb.Empty{}}}
	integerType := &umpirespb.TypeRef{Ref: &umpirespb.TypeRef_Int{Int: &emptypb.Empty{}}}
	listOf := func(element *umpirespb.TypeRef) *umpirespb.TypeRef {
		return &umpirespb.TypeRef{Ref: &umpirespb.TypeRef_List{List: element}}
	}
	leafFields := []*umpirespb.Field{{Name: "enabled", Type: booleanType}, {Name: "count", Type: integerType}}
	active := &umpirespb.Case{Name: "active", Fields: []*umpirespb.Field{{Name: "payload", Type: named("Leaf")}, {Name: "sequence", Type: listOf(integerType)}}}
	idle := &umpirespb.Case{Name: "idle"}
	x := &QuintExport{from: &Slice{types: map[string]*umpirespb.Type{
		"Leaf":   {Name: "Leaf", Shape: &umpirespb.Type_Record{Record: &umpirespb.Record{Fields: leafFields}}},
		"Status": {Name: "Status", Shape: &umpirespb.Type_Enum{Enum: &umpirespb.Enum{Cases: []*umpirespb.Case{active, idle}}}},
		"Envelope": {Name: "Envelope", Shape: &umpirespb.Type_Record{Record: &umpirespb.Record{Fields: []*umpirespb.Field{
			{Name: "state", Type: named("Status")}, {Name: "history", Type: listOf(named("Status"))}, {Name: "allowed", Type: booleanType},
		}}}},
	}}, tags: map[string]variant{"statusActive": {typ: "Status", enum: active}, "statusIdle": {typ: "Status", enum: idle}}}
	leaf := func(enabled bool, count int64) interp.Value {
		return interp.Value{Kind: interp.RecordValue, Type: "Leaf", Fields: []interp.Value{{Kind: interp.BoolValue, Bool: enabled}, {Kind: interp.IntValue, Int: count}}}
	}
	status := func(payload interp.Value, numbers ...int64) interp.Value {
		items := make([]interp.Value, len(numbers))
		for n, number := range numbers {
			items[n] = interp.Value{Kind: interp.IntValue, Int: number}
		}
		return interp.Value{Kind: interp.EnumValue, Type: "Status", Case: "active", Fields: []interp.Value{payload, {Kind: interp.ListValue, Items: items}}}
	}
	first, second := status(leaf(true, 17), 3, 4), status(leaf(false, -9), -1, 0)
	for _, specimen := range []struct {
		name, typ, input string
		want             interp.Value
	}{
		{"record fields", "Leaf", `{"f_enabled":true,"f_count":{"#bigint":"17"}}`, leaf(true, 17)},
		{"zero-field enum tag", "Status", `{"tag":"statusIdle","value":{"#tup":[]}}`, interp.Value{Kind: interp.EnumValue, Type: "Status", Case: "idle"}},
		{"nested declared traversal", "Envelope", `{"f_state":{"tag":"statusActive","value":{"f_payload":{"f_enabled":true,"f_count":{"#bigint":"17"}},"f_sequence":[{"#bigint":"3"},{"#bigint":"4"}]}},"f_history":[{"tag":"statusActive","value":{"f_payload":{"f_enabled":false,"f_count":{"#bigint":"-9"}},"f_sequence":[{"#bigint":"-1"},{"#bigint":"0"}]}}],"f_allowed":false}`, interp.Value{Kind: interp.RecordValue, Type: "Envelope", Fields: []interp.Value{first, {Kind: interp.ListValue, Items: []interp.Value{second}}, {Kind: interp.BoolValue, Bool: false}}}},
	} {
		t.Run(specimen.name, func(t *testing.T) {
			var raw any
			require.NoError(t, json.Unmarshal([]byte(specimen.input), &raw), "only accepted plain JSON values enter the reader")
			canonical, err := json.Marshal(raw)
			require.NoError(t, err)
			pristine := map[string]*umpirespb.Type{}
			for name, decl := range x.from.types {
				pristine[name] = proto.Clone(decl).(*umpirespb.Type)
			}
			read := func(legacy bool) interp.Value {
				var value interp.Value
				var err error
				if legacy {
					value, err = (legacyITFValue{x}).value(raw, named(specimen.typ))
				} else {
					value, err = x.value(raw, named(specimen.typ))
				}
				require.NoError(t, err)
				require.Equal(t, specimen.want, value, "independently authored complete declared Value")
				require.Equal(t, specimen.want.Key(), value.Key())
				return value
			}
			require.Equal(t, read(true), read(false), "literal original and current reader")
			measure := func(legacy bool) float64 {
				return testing.AllocsPerRun(1, func() { value := read(legacy); runtime.KeepAlive(value) })
			}
			original, current := measure(true), measure(false)
			require.Equal(t, specimen.want, read(false), "fresh complete value after measurements")
			after, err := json.Marshal(raw)
			require.NoError(t, err)
			require.Equal(t, canonical, after, "entire accepted graph pristine after every read")
			require.Len(t, x.from.types, len(pristine))
			for name, decl := range pristine {
				require.True(t, proto.Equal(decl, x.from.types[name]), name)
			}
			t.Logf("fully populated %s native allocations: original=%g current=%g", specimen.typ, original, current)
			require.Less(t, current, original*0.9, "avoid unused diagnostics across populated declared fields")
		})
	}
}

// legacyITFCompositionReceipts compares composition j's part of a dump with the composed table the reader's checker
// builds, and with its Properties as the checker binds them.
func (s *Slice) legacyITFCompositionReceipts(x *QuintExport, j int, out map[string]any) ([]Receipt, error) {
	name := x.Compositions[j]
	part, ok := out[fmt.Sprintf("c%d", j)].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("the Quint dump has no part c%d for the composition %s", j, name)
	}
	theirs, err := x.legacyComposedReader(j).machine(part)
	if err != nil {
		return nil, fmt.Errorf("the Quint dump of %s: %w", name, err)
	}
	c, err := s.bound.Composition(name)
	if err != nil {
		return nil, fmt.Errorf("the export names the composition %s, which Go does not build: %w", name, err)
	}
	ours, err := composedView(c)
	if err != nil {
		return nil, err
	}
	receipts := []Receipt{transitions(name, ours, theirs)}
	if len(ours.Properties) > 0 || theirs.Claims != nil {
		receipts = append(receipts, claimed(name, ours, theirs))
	}
	return append(receipts, coverage(name, theirs)), nil
}
