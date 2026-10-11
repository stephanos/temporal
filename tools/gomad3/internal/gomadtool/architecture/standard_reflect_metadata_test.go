package architecture

import (
	"go/token"
	"go/types"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func reflectMetadataAnalysis(t *testing.T) *effectAnalysis {
	t.Helper()
	const module = "example.invalid/reflectmetadata"
	program := stockSummaryProgram(t, module, `package record
import "reflect"
type Hostile struct { reflect.Type }
func (Hostile) Kind() reflect.Kind { go sideEffect(); return 0 }
type Dirty struct{}
func (Dirty) MarshalJSON() ([]byte,error) { go sideEffect(); return nil,nil }
type DirtyText struct{}
func (DirtyText) MarshalText() ([]byte,error) { go sideEffect(); return nil,nil }
type DirtyError struct{}
func (DirtyError) Error() string { go sideEffect(); return "dirty" }
type Data struct { Number int; Value Dirty; Sequence []Dirty }
func Root() {}
func sideEffect() {}
func Native() reflect.Type { return reflect.TypeOf(Data{}) }
func CallKind(t reflect.Type) { _=t.Kind() }
func RootHostile() { CallKind(Hostile{reflect.TypeOf(0)}) }
func RootMixed() { CallKind(reflect.TypeOf(0)); CallKind(Hostile{reflect.TypeOf(0)}) }
func RootUnknown(t reflect.Type) { _=t.Kind() }
func RootArgument() { _=reflect.TypeOf(0).Implements(Hostile{reflect.TypeOf(0)}) }
func RootNative() {
 t:=reflect.TypeFor[Data]()
 _=t.Kind(); _=t.NumField(); _=t.Field(0).Type.Kind(); _=t.String()
 _=t.Implements(reflect.TypeFor[any]())
 _=reflect.PointerTo(t).Elem().Kind()
 _=reflect.TypeOf(map[string][]Dirty{}).Key().Kind()
 _=reflect.ValueOf(Data{}).Type().Kind()
}
`)
	return effectResolverAnalysis(program)
}

func reflectMetadataFunction(a *effectAnalysis, receiver, name string) *types.Func {
	if receiver == "" {
		return a.program.Packages["reflect"].Types.Scope().Lookup(name).(*types.Func)
	}
	return a.program.Functions["reflect."+receiver+"."+name].object
}

func reflectMetadataPosition(a *effectAnalysis) token.Pos {
	return a.program.Functions["example.invalid/reflectmetadata/record.Root"].declaration.Pos()
}

func requireNativeReflectMetadata(t *testing.T, a *effectAnalysis, value *abstractValue) {
	t.Helper()
	native := types.NewPointer(a.program.Packages["reflect"].Types.Scope().Lookup("rtype").Type())
	if value == nil || value.unknown || value.builtin || len(value.types) == 0 {
		t.Fatalf("missing non-builtin native metadata identity: %#v", value)
	}
	for _, typ := range value.types {
		if !types.Identical(typ, native) {
			t.Fatalf("metadata dynamic type=%v want imported %v", typ, native)
		}
	}
}

func TestReflectMetadataProducersPreserveNativeIdentity(t *testing.T) {
	a := reflectMetadataAnalysis(t)
	pos := reflectMetadataPosition(a)
	mapType := types.NewMap(types.Typ[types.String], types.NewSlice(types.Typ[types.Int]))
	typeOf, handled := a.standard(reflectMetadataFunction(a, "", "TypeOf"), nil, []*abstractValue{valueOf(mapType)}, pos)
	if !handled {
		t.Fatal("TypeOf summary missing")
	}
	requireNativeReflectMetadata(t, a, typeOf)
	if typeOf.fields["represented"] != nil || typeOf.fields["$reflected"] == nil || len(typeOf.fields["$reflected"].types) != 1 || !types.Identical(typeOf.fields["$reflected"].types[0], mapType) {
		t.Fatal("TypeOf mixed metadata with ordinary represented values")
	}
	value, handled := a.standard(reflectMetadataFunction(a, "", "ValueOf"), nil, []*abstractValue{valueOf(mapType)}, pos)
	if !handled {
		t.Fatal("ValueOf summary missing")
	}
	valueType, handled := a.standard(reflectMetadataFunction(a, "Value", "Type"), value, nil, pos)
	if !handled {
		t.Fatal("Value.Type summary missing")
	}
	requireNativeReflectMetadata(t, a, valueType)
	typeFor := reflectMetadataFunction(a, "", "TypeFor")
	parameter := typeFor.Type().(*types.Signature).TypeParams().At(0)
	a.typeArguments[parameter] = []types.Type{mapType, types.Typ[types.String]}
	forType, handled := a.standard(typeFor, nil, nil, pos)
	if !handled {
		t.Fatal("TypeFor summary missing")
	}
	requireNativeReflectMetadata(t, a, forType)
	if reflected := forType.fields["$reflected"]; reflected == nil || len(reflected.types) != 2 {
		t.Fatal("TypeFor lost a represented generic type alternative")
	}
	pointer, handled := a.standard(reflectMetadataFunction(a, "", "PointerTo"), nil, []*abstractValue{typeOf}, pos)
	if !handled {
		t.Fatal("PointerTo summary missing")
	}
	requireNativeReflectMetadata(t, a, pointer)
	if reflected := pointer.fields["$reflected"]; reflected == nil || len(reflected.types) != 1 || !types.Identical(reflected.types[0], types.NewPointer(mapType)) {
		t.Fatal("PointerTo lost represented pointer type")
	}
	if len(a.findings) != 0 || !a.checkedMemory["reflect"] {
		t.Fatalf("native metadata source validation: %v", a.findings)
	}
}

func TestReflectMetadataGettersProjectMetadataOnly(t *testing.T) {
	a := reflectMetadataAnalysis(t)
	pos := reflectMetadataPosition(a)
	record := a.program.Packages["example.invalid/reflectmetadata/record"]
	data := record.Types.Scope().Lookup("Data").Type()
	dirty := record.Types.Scope().Lookup("Dirty").Type()
	metadata, _ := a.standard(reflectMetadataFunction(a, "", "TypeOf"), nil, []*abstractValue{valueOf(data)}, pos)
	for _, name := range []string{"Kind", "NumField", "String"} {
		result, handled := a.standard(reflectMetadataFunction(a, "*rtype", name), metadata, nil, pos)
		if !handled || result == nil || result.unknown || result.builtin {
			t.Fatalf("native %s getter missing: handled=%v result=%#v", name, handled, result)
		}
	}
	field, handled := a.standard(reflectMetadataFunction(a, "*rtype", "Field"), metadata, []*abstractValue{valueOf(types.Typ[types.Int])}, pos)
	if !handled || field == nil {
		t.Fatal("native Field metadata summary missing")
	}
	requireNativeReflectMetadata(t, a, field.fields["Type"])
	reflected := field.fields["Type"].fields["$reflected"]
	if reflected == nil || len(reflected.types) != 3 {
		t.Fatal("unknown Field index did not retain every field type")
	}
	mapType := types.NewMap(types.Typ[types.String], types.NewSlice(dirty))
	mapMetadata, _ := a.standard(reflectMetadataFunction(a, "", "TypeOf"), nil, []*abstractValue{valueOf(mapType)}, pos)
	key, handled := a.standard(reflectMetadataFunction(a, "*rtype", "Key"), mapMetadata, nil, pos)
	if !handled {
		t.Fatal("native Key summary missing")
	}
	requireNativeReflectMetadata(t, a, key)
	if !types.Identical(key.fields["$reflected"].types[0], types.Typ[types.String]) {
		t.Fatal("Key metadata retained map type")
	}
	element, handled := a.standard(reflectMetadataFunction(a, "*rtype", "Elem"), mapMetadata, nil, pos)
	if !handled {
		t.Fatal("native Elem summary missing")
	}
	requireNativeReflectMetadata(t, a, element)
	if !types.Identical(element.fields["$reflected"].types[0], types.NewSlice(dirty)) {
		t.Fatal("Elem metadata retained map type")
	}
	_, handled = a.standard(reflectMetadataFunction(a, "*rtype", "Implements"), metadata, []*abstractValue{key}, pos)
	if !handled || len(a.findings) != 0 {
		t.Fatalf("metadata-only lookup invoked Dirty methods: handled=%v findings=%v", handled, a.findings)
	}
	a.implicit(valueOf(dirty), []string{"MarshalJSON"}, nil, pos, map[types.Type]bool{})
	if len(a.findings) == 0 {
		t.Fatal("native metadata summary erased a dirty value callback")
	}
}

func TestReflectMetadataStaticInterfaceAndFunctionTypesRemainKnown(t *testing.T) {
	a := reflectMetadataAnalysis(t)
	pos := reflectMetadataPosition(a)
	key := types.NewInterfaceType(nil, nil)
	element := types.NewSignatureType(nil, nil, nil, types.NewTuple(), types.NewTuple(), false)
	metadata, _ := a.standard(reflectMetadataFunction(a, "", "TypeOf"), nil, []*abstractValue{valueOf(types.NewMap(key, element))}, pos)
	for _, name := range []string{"Key", "Elem"} {
		result, handled := a.standard(reflectMetadataFunction(a, "*rtype", name), metadata, nil, pos)
		if !handled || result == nil || result.fields["$reflected"] == nil || result.fields["$reflected"].unknown {
			t.Fatalf("static %s metadata confused with unknown dynamic value: handled=%v result=%#v", name, handled, result)
		}
	}
}

func TestReflectMetadataDoesNotEraseOriginalValueCallbacks(t *testing.T) {
	for _, tc := range []struct{ name, method string }{
		{"Dirty", "MarshalJSON"},
		{"DirtyText", "MarshalText"},
		{"DirtyError", "Error"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a := reflectMetadataAnalysis(t)
			pos := reflectMetadataPosition(a)
			typ := a.program.Packages["example.invalid/reflectmetadata/record"].Types.Scope().Lookup(tc.name).Type()
			value := valueOf(typ)
			marker := valueOf(types.Typ[types.String])
			value.fields = map[string]*abstractValue{"original": marker}
			metadata, handled := a.standard(reflectMetadataFunction(a, "", "TypeOf"), nil, []*abstractValue{value}, pos)
			if !handled {
				t.Fatal("TypeOf summary missing")
			}
			_, handled = a.standard(reflectMetadataFunction(a, "*rtype", "String"), metadata, nil, pos)
			if !handled || len(a.findings) != 0 || value.fields["original"] != marker || value.builtin || value.unknown || len(value.types) != 1 || !types.Identical(value.types[0], typ) {
				t.Fatalf("metadata altered or consumed original value: value=%#v findings=%v", value, a.findings)
			}
			if represented := metadata.fields["$reflected"]; represented == nil || len(represented.fields) != 0 || len(represented.functions) != 0 {
				t.Fatal("metadata token retained ordinary callback-bearing value data")
			}
			a.implicit(value, []string{tc.method}, nil, pos, map[types.Type]bool{})
			found := false
			for _, finding := range a.findings {
				found = found || finding.Category == "host-effect" && strings.Contains(finding.Detail, "record."+tc.name+"."+tc.method)
			}
			if !found {
				t.Fatalf("metadata token hid original %s callback: %v", tc.method, a.findings)
			}
		})
	}
}

func TestReflectMetadataNativeASTIsPure(t *testing.T) {
	a := reflectMetadataAnalysis(t)
	fn := a.program.Functions["example.invalid/reflectmetadata/record.RootNative"]
	a.root = functionID(fn.object)
	a.call(&boundFunction{fn: fn, pkg: fn.pkg}, nil, fn.declaration.Pos())
	if len(a.findings) != 0 {
		t.Fatalf("native metadata AST findings: %v", a.findings)
	}
}

func TestReflectMetadataRejectsNonNativeAlternatives(t *testing.T) {
	a := reflectMetadataAnalysis(t)
	pos := reflectMetadataPosition(a)
	native := types.NewPointer(a.program.Packages["reflect"].Types.Scope().Lookup("rtype").Type())
	foreignPackage := types.NewPackage("reflect", "reflect")
	foreign := types.NewPointer(types.NewNamed(types.NewTypeName(0, foreignPackage, "rtype", nil), types.NewStruct(nil, nil), nil))
	hostile := a.program.Packages["example.invalid/reflectmetadata/record"].Types.Scope().Lookup("Hostile").Type()
	for _, tc := range []struct {
		name  string
		value *abstractValue
	}{
		{"absent", nil},
		{"unknown-native", &abstractValue{types: []types.Type{native}, unknown: true}},
		{"interface", valueOf(a.program.Packages["reflect"].Types.Scope().Lookup("Type").Type())},
		{"hostile-embedded", valueOf(hostile)},
		{"mixed", &abstractValue{types: []types.Type{native, hostile}}},
		{"different-package-object", valueOf(foreign)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for _, name := range []string{"Kind", "Elem", "Key", "NumField", "Field", "String", "Implements"} {
				if _, handled := a.standard(reflectMetadataFunction(a, "*rtype", name), tc.value, []*abstractValue{valueOf(native)}, pos); handled {
					t.Fatalf("native %s accepted %s receiver", name, tc.name)
				}
			}
			if _, handled := a.standard(reflectMetadataFunction(a, "", "PointerTo"), nil, []*abstractValue{tc.value}, pos); handled {
				t.Fatalf("PointerTo accepted %s metadata", tc.name)
			}
			if _, handled := a.standard(reflectMetadataFunction(a, "*rtype", "Implements"), valueOf(native), []*abstractValue{tc.value}, pos); handled {
				t.Fatalf("Implements accepted %s argument", tc.name)
			}
		})
	}
}

func TestReflectMetadataHostileASTCallbacksRemainVisible(t *testing.T) {
	a := reflectMetadataAnalysis(t)
	for _, name := range []string{"RootHostile", "RootMixed", "RootUnknown", "RootArgument"} {
		fn := a.program.Functions["example.invalid/reflectmetadata/record."+name]
		a.root = functionID(fn.object)
		a.call(&boundFunction{fn: fn, pkg: fn.pkg}, nil, fn.declaration.Pos())
		found := false
		for _, finding := range a.findings {
			if strings.HasPrefix(finding.Detail, a.root+" ->") {
				if name == "RootUnknown" {
					found = found || finding.Category == "unresolved-effect"
				} else {
					found = found || finding.Category == "host-effect" && strings.Contains(finding.Detail, "record.Hostile.Kind")
				}
			}
		}
		if !found {
			t.Fatalf("native metadata hid %s effects: %v", name, a.findings)
		}
	}
}

func TestReflectMetadataRejectsPinAndFunctionIdentityChanges(t *testing.T) {
	for _, variant := range []string{"nonstandard", "nonstandard-import", "changed-source", "different-package-function"} {
		t.Run(variant, func(t *testing.T) {
			a := reflectMetadataAnalysis(t)
			pos := reflectMetadataPosition(a)
			fn := reflectMetadataFunction(a, "", "TypeOf")
			metadata := a.program.Metadata["reflect"]
			switch variant {
			case "nonstandard":
				metadata.Standard = false
			case "nonstandard-import":
				a.program.Packages["reflect"].Metadata.Standard = false
			case "changed-source":
				metadata.Dir = t.TempDir()
				if err := os.WriteFile(filepath.Join(metadata.Dir, "type.go"), []byte("package reflect\n"), 0600); err != nil {
					t.Fatal(err)
				}
			case "different-package-function":
				fn = types.NewFunc(fn.Pos(), types.NewPackage("reflect", "reflect"), fn.Name(), fn.Type().(*types.Signature))
			}
			a.program.Metadata["reflect"] = metadata
			result, handled := a.standard(fn, nil, []*abstractValue{valueOf(types.Typ[types.Int])}, pos)
			if variant == "changed-source" {
				if !handled || len(a.findings) == 0 || a.checkedMemory["reflect"] {
					t.Fatalf("changed native metadata source accepted: handled=%v result=%#v findings=%v", handled, result, a.findings)
				}
			} else if handled {
				t.Fatalf("%s native metadata summary accepted", variant)
			}
		})
	}
}
