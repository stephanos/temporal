package architecture

import (
	"go/token"
	"go/types"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type jsonCallbackWalker interface {
	jsonCallbacks(*abstractValue, bool, token.Pos) *abstractValue
}

func TestJSONCallbacksNativeWrapperMemoryLeaves(t *testing.T) {
	program, fn, _ := jsonProfileFixture(t)
	for _, name := range []string{"Number", "RawMessage"} {
		for _, decoding := range []bool{false, true} {
			t.Run(name+map[bool]string{false: "/marshal", true: "/unmarshal"}[decoding], func(t *testing.T) {
				analysis := effectResolverAnalysis(program)
				if !analysis.checkJSONSummarySource(fn, fn.Pos()) {
					t.Fatal("wrapper source profile was not qualified")
				}
				typ := program.Packages["encoding/json"].Types.Scope().Lookup(name).Type()
				if decoding {
					typ = types.NewPointer(typ)
				}
				analysis.jsonCallbacks(valueOf(typ), decoding, fn.Pos())
				if len(analysis.findings) != 0 {
					t.Fatalf("native owned-memory %s leaf rejected: %v", name, analysis.findings)
				}
			})
		}
	}
}

func TestJSONCallbacksNativeMemoryRequiresExactWrapperBinding(t *testing.T) {
	program, wrapper, _ := jsonProfileFixture(t)
	typ := program.Packages["encoding/json"].Types.Scope().Lookup("Number").Type()
	method := types.NewMethodSet(typ).Lookup(nil, "MarshalJSONTo").Obj().(*types.Func)
	analysis := effectResolverAnalysis(program)
	if analysis.jsonNativeMemoryCallback(method, valueOf(typ)) {
		t.Fatal("direct native stream received wrapper-owned memory authority")
	}
	analysis.jsonCallbacks(valueOf(typ), false, wrapper.Pos())
	if len(analysis.findings) == 0 {
		t.Fatal("native stream without a qualified wrapper was silently accepted")
	}
	analysis = effectResolverAnalysis(program)
	if !analysis.checkJSONSummarySource(wrapper, wrapper.Pos()) {
		t.Fatal("wrapper did not qualify")
	}
	foreign := types.NewNamed(types.NewTypeName(token.NoPos, types.NewPackage("encoding/json", "json"), "Number", nil), types.Typ[types.String], nil)
	for name, value := range map[string]*abstractValue{
		"empty":   {},
		"unknown": {types: []types.Type{typ}, unknown: true},
		"mixed":   {types: []types.Type{typ, types.Typ[types.String]}},
		"foreign": {types: []types.Type{foreign}},
	} {
		if analysis.jsonNativeMemoryCallback(method, value) {
			t.Fatalf("%s binding received native memory authority", name)
		}
	}
}

func jsonWalkerSourceFixture(t *testing.T, source string) (*effectAnalysis, *abstractValue, token.Pos) {
	t.Helper()
	program, _, _ := jsonProfileFixture(t)
	const path = "example.invalid/jsonwalker/record"
	program.Module = "example.invalid/jsonwalker"
	directory := t.TempDir()
	if err := os.WriteFile(filepath.Join(directory, "pure.go"), []byte("package record\nimport \"encoding/json/jsontext\"\nvar _ *jsontext.Encoder\nfunc sideEffect(){}\nfunc Root(){}\n"+source), 0600); err != nil {
		t.Fatal(err)
	}
	program.Metadata[path] = Package{ImportPath: path, Dir: directory, GoFiles: []string{"pure.go"}, Imports: []string{"encoding/json/jsontext"}}
	if _, err := program.Import(path); err != nil {
		t.Fatal(err)
	}
	root := program.Functions[path+".Root"]
	return effectResolverAnalysis(program), valueOf(root.pkg.Types.Scope().Lookup("Dirty").Type()), root.declaration.Pos()
}

func TestJSONCallbacksStreamRetainsFallbackAndUnknownAuthority(t *testing.T) {
	for _, decoding := range []bool{false, true} {
		t.Run(map[bool]string{false: "marshal", true: "unmarshal"}[decoding], func(t *testing.T) {
			methods := "func (Dirty) MarshalJSONTo(*jsontext.Encoder)error{return nil}\nfunc (Dirty) MarshalJSON()([]byte,error){go sideEffect();return nil,nil}\nfunc (Leaf) MarshalJSON()([]byte,error){go sideEffect();return nil,nil}\n"
			if decoding {
				methods = "func (Dirty) UnmarshalJSONFrom(*jsontext.Decoder)error{return nil}\nfunc (Dirty) UnmarshalJSON([]byte)error{go sideEffect();return nil}\nfunc (Leaf) UnmarshalJSON([]byte)error{go sideEffect();return nil}\n"
			}
			analysis, receiver, pos := jsonWalkerSourceFixture(t, "type Leaf struct{}\ntype Dirty struct{ Value Leaf }\n"+methods)
			analysis.jsonCallbacks(receiver, decoding, pos)
			lower, child, unknownStream := false, false, false
			for _, finding := range analysis.findings {
				lower = lower || finding.Category == "host-effect" && strings.Contains(finding.Detail, "record.Dirty.")
				child = child || finding.Category == "host-effect" && strings.Contains(finding.Detail, "record.Leaf.")
				unknownStream = unknownStream || finding.Category == "unresolved-effect" && strings.Contains(finding.Detail, "unsupported custom JSONtext stream context")
			}
			if !lower || !child || !unknownStream {
				t.Fatalf("stream fallback/authority lost: lower=%v child=%v unknown=%v findings=%v", lower, child, unknownStream, analysis.findings)
			}
		})
	}
}

func TestJSONCallbacksOmizeroAndIgnoredTags(t *testing.T) {
	program := effectResolverProgram(t, "package record\ntype Field struct{}\ntype Dirty struct{ Active Field `json:\"active,omitzero\"`; Ignored Field `json:\"-\"` }\nfunc (*Field) IsZero()bool{go sideEffect();return false}\nfunc sideEffect(){}\nfunc Root(){}\n")
	root := program.Functions["example.invalid/resolver/record.Root"]
	analysis := effectResolverAnalysis(program)
	analysis.jsonCallbacks(valueOf(root.pkg.Types.Scope().Lookup("Dirty").Type()), false, root.declaration.Pos())
	found := false
	for _, finding := range analysis.findings {
		found = found || finding.Category == "host-effect" && strings.Contains(finding.Detail, "*Field.IsZero -> goroutine start")
	}
	if !found {
		t.Fatalf("pointer-on-copy IsZero was not visited: %v", analysis.findings)
	}
	analysis = effectResolverAnalysis(program)
	ignored := types.NewStruct([]*types.Var{types.NewField(token.NoPos, root.pkg.Types, "Ignored", root.pkg.Types.Scope().Lookup("Field").Type(), false)}, []string{"json:\"-\""})
	analysis.jsonCallbacks(valueOf(ignored), false, root.declaration.Pos())
	if len(analysis.findings) != 0 {
		t.Fatalf("ignored JSON field invoked a callback: %v", analysis.findings)
	}
}

func TestJSONCallbacksMapOrderAndCapturedCallbackRemainConservative(t *testing.T) {
	program := effectResolverProgram(t, "package record\ntype Dirty struct{ Callback func() }\nfunc (v Dirty) MarshalJSON()([]byte,error){v.Callback();return nil,nil}\nfunc captured(){go sideEffect()}\nfunc sideEffect(){}\nfunc Root(){}\n")
	root := program.Functions["example.invalid/resolver/record.Root"]
	captured := program.Functions["example.invalid/resolver/record.captured"]
	value := valueOf(root.pkg.Types.Scope().Lookup("Dirty").Type())
	value.fields = map[string]*abstractValue{"Callback": {functions: []*boundFunction{{fn: captured, pkg: captured.pkg}}}}
	collection := valueOf(types.NewMap(types.Typ[types.String], value.types[0]))
	collection.elements = value
	analysis := effectResolverAnalysis(program)
	analysis.jsonCallbacks(collection, false, root.declaration.Pos())
	order, host := false, false
	for _, finding := range analysis.findings {
		order = order || finding.Category == "unresolved-effect" && strings.Contains(finding.Detail, "JSON map callback order")
		host = host || finding.Category == "host-effect" && strings.Contains(finding.Detail, "record.captured -> goroutine start")
	}
	if !order || !host {
		t.Fatalf("map order/captured callback lost: order=%v host=%v findings=%v", order, host, analysis.findings)
	}
}

func TestJSONCallbacksRejectsEmptyBinding(t *testing.T) {
	program := effectResolverProgram(t, "package record\nfunc Root(){}\n")
	analysis := effectResolverAnalysis(program)
	root := program.Functions["example.invalid/resolver/record.Root"]
	analysis.jsonCallbacks(&abstractValue{}, false, root.declaration.Pos())
	if len(analysis.findings) == 0 {
		t.Fatal("empty JSON receiver binding was accepted")
	}
}

func TestJSONCallbacksInterfaceOnlyBindingIsConservative(t *testing.T) {
	program := effectResolverProgram(t, "package record\ntype Failure interface{error}\nfunc Root(){}\n")
	root := program.Functions["example.invalid/resolver/record.Root"]
	for _, typ := range []types.Type{types.Universe.Lookup("any").Type(), types.Universe.Lookup("error").Type(), root.pkg.Types.Scope().Lookup("Failure").Type()} {
		for _, decoding := range []bool{false, true} {
			analysis := effectResolverAnalysis(program)
			value := valueOf(typ)
			value.unknown = false
			value.builtin = true // Generic modeled/decoded values are not fresh-nil proof.
			analysis.jsonCallbacks(value, decoding, root.declaration.Pos())
			if len(analysis.findings) == 0 {
				t.Fatalf("interface-only %s decoding=%v was accepted", typ, decoding)
			}
		}
	}
	analysis := effectResolverAnalysis(program)
	analysis.jsonCallbacks(valueOf(types.Typ[types.UntypedNil]), false, root.declaration.Pos())
	if len(analysis.findings) != 0 {
		t.Fatalf("proven nil unexpectedly rejected: %v", analysis.findings)
	}
}

func TestJSONCallbacksOmizeroPointerPreservesPointee(t *testing.T) {
	program := effectResolverProgram(t, "package record\ntype Field struct{Callback func()}\ntype Dirty struct{Value *Field `json:\"value,omitzero\"`}\nfunc (f *Field) IsZero()bool{(*f).Callback();return false}\nfunc captured(){go sideEffect()}\nfunc sideEffect(){}\nfunc Root(){}\n")
	root := program.Functions["example.invalid/resolver/record.Root"]
	captured := program.Functions["example.invalid/resolver/record.captured"]
	pointee := valueOf(root.pkg.Types.Scope().Lookup("Field").Type())
	pointee.fields = map[string]*abstractValue{"Callback": {functions: []*boundFunction{{fn: captured, pkg: captured.pkg}}}}
	pointer := &abstractValue{types: []types.Type{types.NewPointer(pointee.types[0])}, fields: map[string]*abstractValue{"$pointee": pointee}}
	value := valueOf(root.pkg.Types.Scope().Lookup("Dirty").Type())
	value.fields = map[string]*abstractValue{"Value": pointer}
	analysis := effectResolverAnalysis(program)
	analysis.jsonCallbacks(value, false, root.declaration.Pos())
	found := false
	for _, finding := range analysis.findings {
		found = found || finding.Category == "host-effect" && strings.Contains(finding.Detail, "record.captured -> goroutine start")
	}
	if !found {
		t.Fatalf("pointer IsZero lost the actual pointee callback: %v", analysis.findings)
	}
	if pointer.fields["$pointee"] != pointee || pointee.fields["Callback"].functions[0].fn != captured {
		t.Fatal("callback traversal mutated the original pointer binding")
	}
}

func TestJSONCallbacksNestedUnorderedRecursionIsConservative(t *testing.T) {
	program := effectResolverProgram(t, "package record\ntype Dirty struct{Next map[string]*Dirty}\nfunc Root(){}\n")
	root := program.Functions["example.invalid/resolver/record.Root"]
	analysis := effectResolverAnalysis(program)
	analysis.jsonCallbacks(valueOf(root.pkg.Types.Scope().Lookup("Dirty").Type()), false, root.declaration.Pos())
	found := false
	for _, finding := range analysis.findings {
		found = found || finding.Category == "unresolved-effect" && strings.Contains(finding.Detail, "recursive JSON callback binding")
	}
	if !found {
		t.Fatalf("nested unordered recursion was accepted: %v", analysis.findings)
	}
}

func TestJSONCallbacksV1Addressability(t *testing.T) {
	for _, tc := range []struct {
		name string
		want bool
	}{
		{"value", false}, {"pointer", true}, {"slice", true}, {"array", false}, {"pointer-array", true}, {"map", false}, {"interface-field", false}, {"interface-slice", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			program := effectResolverProgram(t, "package record\ntype Dirty struct{}\ntype Holder struct{ Value any }\nfunc (*Dirty) MarshalJSON()([]byte,error){go sideEffect();return nil,nil}\nfunc sideEffect(){}\nfunc Root(){}\n")
			root := program.Functions["example.invalid/resolver/record.Root"]
			typ := root.pkg.Types.Scope().Lookup("Dirty").Type()
			value := valueOf(typ)
			switch tc.name {
			case "pointer":
				value = addressValue(value, types.NewPointer(typ), root.declaration.Pos())
			case "slice":
				value = valueOf(types.NewSlice(typ))
			case "array":
				value = valueOf(types.NewArray(typ, 1))
			case "pointer-array":
				value = valueOf(types.NewPointer(types.NewArray(typ, 1)))
			case "map":
				value = valueOf(types.NewMap(types.Typ[types.String], typ))
			case "interface-field":
				holder := valueOf(root.pkg.Types.Scope().Lookup("Holder").Type())
				holder.fields = map[string]*abstractValue{"Value": value}
				value = addressValue(holder, types.NewPointer(holder.types[0]), root.declaration.Pos())
			case "interface-slice":
				collection := valueOf(types.NewSlice(types.Universe.Lookup("any").Type()))
				collection.elements = value
				value = collection
			}
			analysis := effectResolverAnalysis(program)
			analysis.jsonCallbacks(value, false, root.declaration.Pos())
			found := false
			for _, finding := range analysis.findings {
				found = found || finding.Category == "host-effect" && strings.Contains(finding.Detail, "*Dirty.MarshalJSON -> goroutine start")
			}
			if found != tc.want {
				t.Fatalf("pointer callback visited=%v want=%v: %v", found, tc.want, analysis.findings)
			}
		})
	}
}

func TestJSONCallbacksMapKeysAreTextOnly(t *testing.T) {
	program := effectResolverProgram(t, "package record\ntype Dirty struct{}\ntype Key struct{ Value Dirty }\nfunc (Dirty) MarshalJSON()([]byte,error){go sideEffect();return nil,nil}\nfunc sideEffect(){}\nfunc Root(){}\n")
	root := program.Functions["example.invalid/resolver/record.Root"]
	key := root.pkg.Types.Scope().Lookup("Key").Type()
	analysis := effectResolverAnalysis(program)
	analysis.jsonCallbacks(valueOf(types.NewMap(key, types.Typ[types.String])), false, root.declaration.Pos())
	for _, finding := range analysis.findings {
		if finding.Category == "host-effect" {
			t.Fatalf("map-key default traversal invoked a JSON callback: %v", finding)
		}
	}
}

func TestJSONCallbacksLegacyPriorityAndSignature(t *testing.T) {
	for _, tc := range []struct{ name, methods, want, absent string }{
		{"append-text", "func (Dirty) AppendText(b []byte)([]byte,error){go appendEffect();return b,nil}\nfunc (Dirty) MarshalText()([]byte,error){go textEffect();return nil,nil}\n", "Dirty.AppendText", "Dirty.MarshalText"},
		{"json-first", "func (Dirty) MarshalJSON()([]byte,error){go jsonEffect();return nil,nil}\nfunc (Dirty) AppendText(b []byte)([]byte,error){go appendEffect();return b,nil}\n", "Dirty.MarshalJSON", "Dirty.AppendText"},
		{"invalid-signature", "func (Dirty) MarshalJSON()string{go jsonEffect();return \"\"}\nfunc (Dirty) MarshalText()([]byte,error){go textEffect();return nil,nil}\n", "Dirty.MarshalText", "Dirty.MarshalJSON"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			program := effectResolverProgram(t, "package record\ntype Dirty struct{}\nfunc appendEffect(){}\nfunc textEffect(){}\nfunc jsonEffect(){}\nfunc Root(){}\n"+tc.methods)
			analysis := effectResolverAnalysis(program)
			walker, ok := any(analysis).(jsonCallbackWalker)
			if !ok {
				t.Fatal("JSON-specific callback traversal is unavailable")
			}
			root := program.Functions["example.invalid/resolver/record.Root"]
			walker.jsonCallbacks(valueOf(root.pkg.Types.Scope().Lookup("Dirty").Type()), false, root.declaration.Pos())
			found := false
			for _, finding := range analysis.findings {
				found = found || finding.Category == "host-effect" && strings.Contains(finding.Detail, tc.want+" -> goroutine start")
				if finding.Category == "host-effect" && strings.Contains(finding.Detail, tc.absent+" -> goroutine start") {
					t.Fatalf("ineligible or lower-priority callback executed: %v", finding)
				}
			}
			if !found {
				t.Fatalf("eligible callback %s was not visited: %v", tc.want, analysis.findings)
			}
		})
	}
}
