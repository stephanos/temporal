package architecture

import (
	"go/ast"
	"go/types"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func reflectMapIndexFunction() *types.Func {
	return reflectSummaryFunction("reflect", "Value", "MapIndex", types.NewStruct(nil, nil))
}

func TestReflectMapIndexProjectsElementsWithoutMutatingMap(t *testing.T) {
	a := reflectSummaryAnalysis(t, "package record\nfunc Root(){}\n")
	pos := a.program.Functions["example.invalid/resolver/record.Root"].declaration.Pos()
	element := valueOf(types.Typ[types.String])
	element.text = "element"
	mapValue := valueOf(types.NewMap(types.Typ[types.String], types.Typ[types.String]))
	mapValue.elements = element
	mapValue.fields = map[string]*abstractValue{"original": valueOf(types.Typ[types.Int])}
	receiver := &abstractValue{fields: map[string]*abstractValue{"represented": mapValue}}
	result, handled := a.standard(reflectMapIndexFunction(), receiver, nil, pos)
	if !handled || result == nil || result.fields["represented"] != element || len(a.findings) != 0 {
		t.Fatalf("map element projection: handled=%v result=%#v findings=%v", handled, result, a.findings)
	}
	if len(result.fields) != 1 || result.builtin || receiver.fields["represented"] != mapValue || mapValue.elements != element || len(mapValue.fields) != 1 || element.text != "element" {
		t.Fatal("MapIndex treated its result as the original map or mutated the represented map")
	}
	valid, handled := a.standard(reflectSummaryFunction("reflect", "Value", "IsValid", types.Typ[types.Bool]), result, nil, pos)
	if !handled || valid.truth != 0 {
		t.Fatal("MapIndex assumed the requested key existed")
	}
}

func TestReflectMapIndexJoinsEveryFallbackElementType(t *testing.T) {
	a := reflectSummaryAnalysis(t, `package record
type JSON struct{}
type Text struct{}
func (JSON) MarshalJSON()([]byte,error){go sideEffect();return nil,nil}
func (Text) MarshalText()([]byte,error){go sideEffect();return nil,nil}
func sideEffect(){}
func Root(){}
`)
	root := a.program.Functions["example.invalid/resolver/record.Root"]
	jsonType := root.pkg.Types.Scope().Lookup("JSON").Type()
	textType := root.pkg.Types.Scope().Lookup("Text").Type()
	mapValue := &abstractValue{types: []types.Type{types.NewMap(types.Typ[types.String], jsonType), types.NewMap(types.Typ[types.Int], textType)}}
	result, handled := a.standard(reflectMapIndexFunction(), &abstractValue{fields: map[string]*abstractValue{"represented": mapValue}}, nil, root.declaration.Pos())
	if !handled || result == nil || result.fields["represented"] == nil {
		t.Fatalf("map fallback not modeled: handled=%v result=%#v", handled, result)
	}
	projected := result.fields["represented"]
	if len(projected.types) != 2 || !types.Identical(projected.types[0], jsonType) || !types.Identical(projected.types[1], textType) || mapValue.elements != nil || len(mapValue.types) != 2 {
		t.Fatalf("fallback lost map alternatives or mutated original: %#v", projected)
	}
	a.implicit(projected, []string{"MarshalJSON", "MarshalText"}, nil, root.declaration.Pos(), map[types.Type]bool{})
	if len(a.findings) != 2 {
		t.Fatalf("fallback lost a dirty callback alternative: %v", a.findings)
	}
	for _, finding := range a.findings {
		if finding.Category != "host-effect" || !strings.Contains(finding.Detail, "goroutine start") {
			t.Fatalf("wrong fallback callback finding: %v", finding)
		}
	}
}

func TestReflectMapIndexRejectsUnknownMapBindings(t *testing.T) {
	mapType := types.NewMap(types.Typ[types.String], types.Typ[types.Int])
	for _, tc := range []struct {
		name  string
		value *abstractValue
	}{
		{"nil-receiver", nil},
		{"absent", nil},
		{"untyped", &abstractValue{}},
		{"nil-type", &abstractValue{types: []types.Type{nil}}},
		{"nonmap", valueOf(types.Typ[types.Int])},
		{"interface", valueOf(types.NewInterfaceType(nil, nil))},
		{"unknown-map", &abstractValue{types: []types.Type{mapType}, unknown: true}},
		{"nil-map", zeroValue(mapType)},
		{"mixed-map", &abstractValue{types: []types.Type{mapType, types.Typ[types.Int]}}},
		{"unknown-interface-element", valueOf(types.NewMap(types.Typ[types.String], types.NewInterfaceType(nil, nil)))},
		{"unknown-function-element", valueOf(types.NewMap(types.Typ[types.String], types.NewSignatureType(nil, nil, nil, types.NewTuple(), types.NewTuple(), false)))},
		{"invalid-key-type", valueOf(types.NewMap(types.NewSlice(types.Typ[types.Byte]), types.Typ[types.Int]))},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a := reflectSummaryAnalysis(t, "package record\nfunc Root(){}\n")
			pos := a.program.Functions["example.invalid/resolver/record.Root"].declaration.Pos()
			receiver := &abstractValue{fields: map[string]*abstractValue{"represented": tc.value}}
			if tc.name == "nil-receiver" {
				receiver = nil
			}
			result, handled := a.standard(reflectMapIndexFunction(), receiver, nil, pos)
			if !handled || result == nil || len(a.findings) == 0 {
				t.Fatalf("unknown MapIndex binding accepted: handled=%v result=%#v findings=%v", handled, result, a.findings)
			}
			if tc.name == "mixed-map" && (result.fields["represented"] == nil || !types.Identical(result.fields["represented"].types[0], types.Typ[types.Int])) {
				t.Fatal("unknown alternative discarded the known map element")
			}
			for _, finding := range a.findings {
				if finding.Category != "unresolved-effect" {
					t.Fatalf("unknown binding did not remain conservative: %v", finding)
				}
			}
		})
	}
}

func TestReflectMapIndexPreservesCapturedFunctionsAndDirtyValues(t *testing.T) {
	for _, tc := range []struct{ name, source, method string }{
		{"json", "func (Dirty) MarshalJSON()([]byte,error){go sideEffect();return nil,nil}", "MarshalJSON"},
		{"text", "func (Dirty) MarshalText()([]byte,error){go sideEffect();return nil,nil}", "MarshalText"},
		{"error", "func (Dirty) Error()string{go sideEffect();return \"dirty\"}", "Error"},
		{"function", "func factory(f func())func(){return func(){f()}}", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a := reflectSummaryAnalysis(t, "package record\ntype Dirty struct{}\nfunc Root(){}\nfunc sideEffect(){go Root()}\n"+tc.source+"\n")
			root := a.program.Functions["example.invalid/resolver/record.Root"]
			pos := root.declaration.Pos()
			element := valueOf(root.pkg.Types.Scope().Lookup("Dirty").Type())
			elementType := element.types[0]
			if tc.name == "function" {
				factory := a.program.Functions["example.invalid/resolver/record.factory"]
				dirty := a.program.Functions["example.invalid/resolver/record.sideEffect"]
				bound := &abstractValue{types: []types.Type{dirty.object.Type()}, functions: []*boundFunction{{fn: dirty, pkg: dirty.pkg}}}
				signature := factory.object.Type().(*types.Signature)
				literal := factory.declaration.Body.List[0].(*ast.ReturnStmt).Results[0].(*ast.FuncLit)
				element = &abstractValue{functions: []*boundFunction{{literal: literal, pkg: factory.pkg, capture: environment{signature.Params().At(0): bound}}}}
				elementType = signature.Results().At(0).Type()
			}
			mapValue := valueOf(types.NewMap(types.Typ[types.String], elementType))
			mapValue.elements = element
			indexed, handled := a.standard(reflectMapIndexFunction(), &abstractValue{fields: map[string]*abstractValue{"represented": mapValue}}, nil, pos)
			if !handled || indexed == nil || len(a.findings) != 0 {
				t.Fatalf("MapIndex did not preserve callback-valued element: handled=%v findings=%v", handled, a.findings)
			}
			interfaced, handled := a.standard(stockSummaryFunction(t, a, "reflect", "Value", "Interface"), indexed, nil, pos)
			if !handled || len(interfaced.functions) != len(element.functions) || interfaced.unknown != element.unknown {
				t.Fatal("Interface lost MapIndex's represented callback element")
			}
			if tc.method == "" {
				a.invoke(interfaced, nil, pos)
			} else {
				a.implicit(interfaced, []string{tc.method}, nil, pos, map[types.Type]bool{})
			}
			found := false
			for _, finding := range a.findings {
				found = found || finding.Category == "host-effect" && strings.Contains(finding.Detail, "goroutine start")
			}
			if !found {
				t.Fatalf("MapIndex/Interface hid dirty map element: %v", a.findings)
			}
		})
	}
}

func TestReflectMapIndexRejectsChangedSource(t *testing.T) {
	for _, variant := range []string{"nonstandard", "changed-source"} {
		t.Run(variant, func(t *testing.T) {
			a := reflectSummaryAnalysis(t, "package record\nfunc Root(){}\n")
			root := a.program.Functions["example.invalid/resolver/record.Root"]
			pos := root.declaration.Pos()
			metadata := a.program.Metadata["reflect"]
			if variant == "nonstandard" {
				metadata.Standard = false
			} else {
				metadata.Dir = t.TempDir()
				if err := os.WriteFile(filepath.Join(metadata.Dir, "map.go"), []byte("package reflect\n"), 0600); err != nil {
					t.Fatal(err)
				}
			}
			a.program.Metadata["reflect"] = metadata
			mapValue := valueOf(types.NewMap(types.Typ[types.String], types.Typ[types.Int]))
			receiver := &abstractValue{fields: map[string]*abstractValue{"represented": mapValue}}
			_, handled := a.standard(reflectMapIndexFunction(), receiver, nil, pos)
			if variant == "nonstandard" {
				if handled {
					t.Fatal("nonstandard MapIndex summary accepted")
				}
				a.call(&boundFunction{fn: &function{object: reflectMapIndexFunction(), declaration: &ast.FuncDecl{}, pkg: root.pkg}, pkg: root.pkg}, nil, pos)
			} else if !handled {
				t.Fatal("changed-source fixture missed MapIndex summary")
			}
			if a.checkedMemory["reflect"] || len(a.findings) == 0 {
				t.Fatalf("MapIndex source not rejected: findings=%v checked=%v", a.findings, a.checkedMemory)
			}
			for _, finding := range a.findings {
				if finding.Category != "unresolved-effect" || variant == "changed-source" && !strings.Contains(finding.Detail, "pinned memory summary source changed: reflect") {
					t.Fatalf("wrong source rejection: %v", finding)
				}
			}
		})
	}
}

func TestReflectMapIndexASTConsumersRemainConservative(t *testing.T) {
	const module = "example.invalid/reflectmap"
	program := stockSummaryProgram(t, module, `package record
import ("reflect";"encoding/json";"fmt")
type JSON struct{}
type Text struct{}
type Error struct{}
func (JSON) MarshalJSON()([]byte,error){go sideEffect();return nil,nil}
func (Text) MarshalText()([]byte,error){go sideEffect();return nil,nil}
func (Error) Error()string{go sideEffect();return "dirty"}
func sideEffect(){}
func dirty(){go sideEffect()}
func RootJSON(){_,_=json.Marshal(reflect.ValueOf(map[string]JSON{"k":{}}).MapIndex(reflect.ValueOf("k")).Interface())}
func RootText(){_,_=json.Marshal(reflect.ValueOf(map[string]Text{"k":{}}).MapIndex(reflect.ValueOf("k")).Interface())}
func RootError(){_=fmt.Sprintf("%v",reflect.ValueOf(map[string]Error{"k":{}}).MapIndex(reflect.ValueOf("k")).Interface())}
func RootFunction(){reflect.ValueOf(map[string]func(){"k":dirty}).MapIndex(reflect.ValueOf("k")).Interface().(func())()}
func RootMissingKey(){if reflect.ValueOf(map[string]int{}).MapIndex(reflect.ValueOf("missing")).IsValid(){go sideEffect()}else{go dirty()}}
`)
	findings := program.Effects()
	for _, name := range []string{"RootJSON", "RootText", "RootError", "RootFunction", "RootMissingKey"} {
		conservative := false
		missingKeyBranches := 0
		for _, finding := range findings {
			if strings.HasPrefix(finding.Detail, module+"/record."+name+" ->") {
				conservative = conservative || finding.Category == "host-effect" || finding.Category == "unresolved-effect"
				if finding.Category == "host-effect" && strings.HasSuffix(finding.Detail, "/record.RootMissingKey -> goroutine start") {
					missingKeyBranches++
				}
			}
		}
		if !conservative || name == "RootMissingKey" && missingKeyBranches != 2 {
			t.Fatalf("MapIndex accepted %s or assumed key presence: %v", name, findings)
		}
	}
}
