package architecture

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"maps"
	"reflect"
	"slices"
	"sort"
	"strings"
	"testing"
)

var bindingKeyAllocationResult string

func TestBindingKeyLegacyEquivalence(t *testing.T) {
	empty := &abstractValue{}
	cycle := &abstractValue{fields: map[string]*abstractValue{}}
	cycle.fields["self"] = cycle
	const emptyKey = `arg0:{false,false,false,"",0;gzip-ownedfalse;nilness0;types[];elements:nil;};`
	for _, tc := range []struct {
		name   string
		values []*abstractValue
		want   string
	}{
		{"no-arguments", nil, ""},
		{"nil", []*abstractValue{nil}, "arg0:nil;"},
		{"empty", []*abstractValue{empty}, emptyKey},
		{"alias", []*abstractValue{empty, empty}, emptyKey + "arg1:ref0;"},
		{"cycle", []*abstractValue{cycle}, `arg0:{false,false,false,"",0;gzip-ownedfalse;nilness0;types[];field"self":ref0;elements:nil;};`},
		{"quoted-flags-types", []*abstractValue{{utc: true, unknown: true, builtin: true,
			text: "\"\n\t\\\x00\xff☃", truth: -17, gzipOwned: true, nilness: -1,
			nilType: types.NewPointer(types.Typ[types.Int]),
			types:   []types.Type{types.Typ[types.String], types.Typ[types.Int]}}},
			`arg0:{true,true,true,"\"\n\t\\\x00\xff☃",-17;gzip-ownedtrue;nilness-1;niltype"*int";types["int" "string"];elements:nil;};`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := bindingKey(tc.values); got != tc.want {
				t.Fatalf("key=%q want=%q", got, tc.want)
			}
			if got := legacyBindingKey(tc.values); got != tc.want {
				t.Fatalf("legacy fixture=%q want=%q", got, tc.want)
			}
		})
	}
	for seed := 0; seed < 32; seed++ {
		t.Run(fmt.Sprintf("bounded-graph-%d", seed), func(t *testing.T) {
			values := bindingKeyFixture(seed)
			unchanged := bindingKeyInputSnapshot(values)
			want := legacyBindingKey(values)
			if got := bindingKey(values); got != want {
				t.Fatalf("key=%q legacy=%q", got, want)
			}
			if !unchanged() || legacyBindingKey(values) != want {
				t.Fatal("key serialization mutated input values, aliases, captures or flags")
			}
		})
	}
}

func bindingKeyInputSnapshot(values []*abstractValue) func() bool {
	valueStates := map[*abstractValue]abstractValue{}
	functionStates := map[*boundFunction]boundFunction{}
	var visit func(*abstractValue)
	visit = func(value *abstractValue) {
		if value == nil {
			return
		}
		if _, seen := valueStates[value]; seen {
			return
		}
		state := *value
		state.types = slices.Clone(value.types)
		state.functions = slices.Clone(value.functions)
		state.fields = maps.Clone(value.fields)
		valueStates[value] = state
		for _, child := range value.fields {
			visit(child)
		}
		visit(value.elements)
		for _, fn := range value.functions {
			if _, seen := functionStates[fn]; seen {
				continue
			}
			state := *fn
			state.capture = maps.Clone(fn.capture)
			functionStates[fn] = state
			visit(fn.receiver)
			for _, captured := range fn.capture {
				visit(captured)
			}
		}
	}
	for _, value := range values {
		visit(value)
	}
	return func() bool {
		for value, state := range valueStates {
			if !reflect.DeepEqual(*value, state) {
				return false
			}
		}
		for fn, state := range functionStates {
			if !reflect.DeepEqual(*fn, state) {
				return false
			}
		}
		return true
	}
}

func TestBindingKeyFormatterAllocations(t *testing.T) {
	for _, tc := range []struct {
		name   string
		values []*abstractValue
	}{
		{"scalar", []*abstractValue{{text: "key", truth: -1, nilness: 1,
			nilType: types.NewPointer(types.Typ[types.Int]), types: []types.Type{types.Typ[types.String]}}}},
		{"bounded-cyclic-graph", bindingKeyFixture(9)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			legacy := testing.AllocsPerRun(100, func() { bindingKeyAllocationResult = legacyBindingKey(tc.values) })
			actual := testing.AllocsPerRun(100, func() { bindingKeyAllocationResult = bindingKey(tc.values) })
			t.Logf("allocations/call current=%g legacy=%g", actual, legacy)
			if actual >= legacy {
				t.Fatalf("formatter allocations=%g must be below measured legacy=%g", actual, legacy)
			}
		})
	}
}

func bindingKeyFixture(seed int) []*abstractValue {
	pkg := types.NewPackage("example.invalid/p\"\\\n", "p")
	signature := types.NewSignatureType(nil, nil, nil, types.NewTuple(), types.NewTuple(), false)
	fn := &function{object: types.NewFunc(token.Pos(99), pkg, "Callback", signature)}
	first := types.NewVar(token.Pos(17), pkg, "first", types.Typ[types.Int])
	last := types.NewVar(token.Pos(33), pkg, "last", types.Typ[types.Int])
	named := types.NewNamed(types.NewTypeName(token.Pos(42), pkg, "Kind", nil), types.Typ[types.Int], nil)
	texts := []string{"", "plain", "\"\n\t\\\x00\xff☃", "e\u0301\u2028"}
	nodes := make([]*abstractValue, 4)
	for i := range nodes {
		truth := seed - i
		if seed == 30 {
			truth = int(^uint(0) >> 1)
		}
		if seed == 31 {
			truth = -int(^uint(0)>>1) - 1
		}
		nodes[i] = &abstractValue{utc: (seed+i)&1 != 0, unknown: (seed+i)&2 != 0,
			builtin: (seed+i)&4 != 0, gzipOwned: (seed+i)&8 != 0, text: texts[(seed+i)%len(texts)],
			truth: truth, nilness: (seed+i)%3 - 1, nilType: types.NewPointer(named), origin: token.Pos(seed + i + 1),
			types: []types.Type{types.Typ[types.String], named, types.Typ[types.Int], types.Typ[types.String]}}
	}
	nodes[0].fields = map[string]*abstractValue{"z\"\n": nodes[2], "a": nodes[1]}
	nodes[1].elements = nodes[2]
	nodes[2].fields = map[string]*abstractValue{"cycle": nodes[0], "alias": nodes[1], "nil": nil}
	nodes[0].elements = nodes[3]
	nodes[3].functions = []*boundFunction{
		{fn: fn, receiver: nodes[1], methodExpression: seed&1 != 0,
			capture: environment{last: nodes[1], first: nodes[0], nil: nodes[2]}},
		{literal: &ast.FuncLit{Type: &ast.FuncType{Func: token.Pos(113)}},
			receiver: nodes[2], capture: environment{first: nodes[3]}},
	}
	return []*abstractValue{nodes[0], nodes[1], nil, nodes[0]}
}

func legacyBindingKey(values []*abstractValue) string {
	var out strings.Builder
	seen := map[*abstractValue]int{}
	var write func(*abstractValue)
	write = func(v *abstractValue) {
		if v == nil {
			out.WriteString("nil;")
			return
		}
		if n, ok := seen[v]; ok {
			fmt.Fprintf(&out, "ref%d;", n)
			return
		}
		seen[v] = len(seen)
		fmt.Fprintf(&out, "{%t,%t,%t,%q,%d;gzip-owned%t;", v.utc, v.unknown, v.builtin, v.text, v.truth, v.gzipOwned)
		fmt.Fprintf(&out, "nilness%d;", v.nilness)
		if v.nilness != 0 && v.nilType != nil {
			fmt.Fprintf(&out, "niltype%q;", types.TypeString(v.nilType, func(p *types.Package) string { return p.Path() }))
		}
		var names []string
		for _, typ := range v.types {
			names = append(names, types.TypeString(typ, func(p *types.Package) string { return p.Path() }))
		}
		sort.Strings(names)
		fmt.Fprintf(&out, "types%q;", names)
		for _, fn := range v.functions {
			fmt.Fprintf(&out, "method-expression%t;", fn.methodExpression)
			if fn.fn != nil {
				out.WriteString(functionID(fn.fn.object))
			} else if fn.literal != nil {
				fmt.Fprintf(&out, "literal%d", fn.literal.Pos())
			}
			write(fn.receiver)
			var objects []types.Object
			for obj := range fn.capture {
				if obj != nil {
					objects = append(objects, obj)
				}
			}
			sort.Slice(objects, func(i, j int) bool { return objects[i].Pos() < objects[j].Pos() })
			for _, obj := range objects {
				fmt.Fprintf(&out, "capture%d:", obj.Pos())
				write(fn.capture[obj])
			}
		}
		var fields []string
		for name := range v.fields {
			fields = append(fields, name)
		}
		sort.Strings(fields)
		for _, name := range fields {
			fmt.Fprintf(&out, "field%q:", name)
			write(v.fields[name])
		}
		out.WriteString("elements:")
		write(v.elements)
		out.WriteString("};")
	}
	for i, v := range values {
		fmt.Fprintf(&out, "arg%d:", i)
		write(v)
	}
	return out.String()
}
