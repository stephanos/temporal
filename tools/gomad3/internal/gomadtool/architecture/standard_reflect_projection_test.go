package architecture

import (
	"go/types"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReflectInterfaceProjectsRepresentedValue(t *testing.T) {
	a := reflectSummaryAnalysis(t, "package record\ntype Value struct{ Field int };func Root(){}\n")
	root := a.program.Functions["example.invalid/resolver/record.Root"]
	pos := root.declaration.Pos()
	represented := valueOf(root.pkg.Types.Scope().Lookup("Value").Type())
	represented.fields = map[string]*abstractValue{"Field": valueOf(types.Typ[types.Int])}
	represented.elements = valueOf(types.Typ[types.String])
	represented.functions = []*boundFunction{{fn: root, pkg: root.pkg}}
	receiver := valueOf(a.program.Packages["reflect"].Types.Scope().Lookup("Value").Type())
	receiver.fields = map[string]*abstractValue{"represented": represented}
	result, handled := a.standard(stockSummaryFunction(t, a, "reflect", "Value", "Interface"), receiver, nil, pos)
	if !handled || result == nil || result.unknown || len(result.types) != 1 || !types.Identical(result.types[0], represented.types[0]) {
		t.Fatalf("Interface lost actual represented type: handled=%v result=%#v", handled, result)
	}
	if result.fields["Field"] != represented.fields["Field"] || result.elements != represented.elements || len(result.functions) != 1 || result.functions[0] != represented.functions[0] {
		t.Fatalf("Interface lost fields/elements/function binding: %#v", result)
	}
	if len(a.findings) != 0 {
		t.Fatalf("known native Interface projection: %v", a.findings)
	}
}

func TestReflectInterfaceUnknownBindingsRemainUnresolved(t *testing.T) {
	for _, name := range []string{"missing", "represented-unknown", "receiver-unknown", "mixed-interface"} {
		t.Run(name, func(t *testing.T) {
			a := reflectSummaryAnalysis(t, "package record\nfunc Root(){}\n")
			pos := a.program.Functions["example.invalid/resolver/record.Root"].declaration.Pos()
			receiver := valueOf(a.program.Packages["reflect"].Types.Scope().Lookup("Value").Type())
			represented := valueOf(types.Typ[types.Bool])
			receiver.fields = map[string]*abstractValue{"represented": represented}
			switch name {
			case "missing":
				delete(receiver.fields, "represented")
			case "represented-unknown":
				represented.unknown = true
			case "receiver-unknown":
				receiver.unknown = true
			case "mixed-interface":
				represented = join(represented, valueOf(types.NewInterfaceType(nil, nil)))
				receiver.fields["represented"] = represented
			}
			result, handled := a.standard(stockSummaryFunction(t, a, "reflect", "Value", "Interface"), receiver, nil, pos)
			if !handled || result == nil || !result.unknown || len(a.findings) == 0 {
				t.Fatalf("unknown binding acquired certainty: handled=%v result=%#v findings=%v", handled, result, a.findings)
			}
			if name == "receiver-unknown" && represented.unknown {
				t.Fatal("projection mutated original represented binding")
			}
		})
	}
}

func TestReflectInterfaceRejectsForgedNativeObject(t *testing.T) {
	a := reflectSummaryAnalysis(t, "package record\nfunc Root(){}\n")
	pos := a.program.Functions["example.invalid/resolver/record.Root"].declaration.Pos()
	receiver := &abstractValue{fields: map[string]*abstractValue{"represented": valueOf(types.Typ[types.Bool])}}
	forged := reflectSummaryFunction("reflect", "Value", "Interface", types.NewInterfaceType(nil, nil))
	if _, handled := a.standard(forged, receiver, nil, pos); handled {
		t.Fatal("foreign same-name reflect object acquired native projection authority")
	}
	native := stockSummaryFunction(t, a, "reflect", "Value", "Interface")
	forged = types.NewFunc(pos, native.Pkg(), native.Name(), native.Type().(*types.Signature))
	if _, handled := a.standard(forged, receiver, nil, pos); handled {
		t.Fatal("forged object inside actual reflect package acquired declaration authority")
	}
}

func TestReflectInterfaceRejectsChangedSourceBinding(t *testing.T) {
	for _, variant := range []string{"nonstandard", "changed-source"} {
		t.Run(variant, func(t *testing.T) {
			a := reflectSummaryAnalysis(t, "package record\nfunc Root(){}\n")
			pos := a.program.Functions["example.invalid/resolver/record.Root"].declaration.Pos()
			fn := stockSummaryFunction(t, a, "reflect", "Value", "Interface")
			metadata := a.program.Metadata["reflect"]
			if variant == "nonstandard" {
				metadata.Standard = false
			} else {
				metadata.Dir = t.TempDir()
				if err := os.WriteFile(filepath.Join(metadata.Dir, "value.go"), []byte("package reflect\n"), 0600); err != nil {
					t.Fatal(err)
				}
			}
			a.program.Metadata["reflect"] = metadata
			receiver := &abstractValue{fields: map[string]*abstractValue{"represented": valueOf(types.Typ[types.Bool])}}
			_, handled := a.standard(fn, receiver, nil, pos)
			if variant == "nonstandard" && handled {
				t.Fatal("nonstandard identity acquired Interface projection authority")
			}
			if variant == "changed-source" && (!handled || len(a.findings) == 0 || a.checkedMemory["reflect"]) {
				t.Fatalf("changed source accepted: handled=%v findings=%v", handled, a.findings)
			}
		})
	}
}

func TestReflectInterfaceASTPreservesDirtyCallbacks(t *testing.T) {
	const module = "example.invalid/reflectprojection"
	program := stockSummaryProgram(t, module, `package record
import ("reflect"; "encoding/json"; "fmt")
type JSON struct{}
func (JSON) MarshalJSON()([]byte,error){go dirty();return nil,nil}
type Text struct{}
func (Text) MarshalText()([]byte,error){go dirty();return nil,nil}
type Failure struct{}
func (Failure) Error()string{go dirty();return "dirty"}
func dirty(){}
func JSONRoot(){_,_ = json.Marshal(reflect.ValueOf(JSON{}).Interface())}
func TextRoot(){_,_ = json.Marshal(reflect.ValueOf(Text{}).Interface())}
func ErrorRoot(){_ = fmt.Sprintf("%v",reflect.ValueOf(Failure{}).Interface())}
func FunctionRoot(){v := reflect.ValueOf(func(){go dirty()}).Interface();v.(func())()}
`)
	findings := program.Effects()
	for _, root := range []string{"JSONRoot", "TextRoot", "ErrorRoot", "FunctionRoot"} {
		found := false
		for _, finding := range findings {
			if strings.HasPrefix(finding.Detail, module+"/record."+root+" ->") && finding.Category == "host-effect" && strings.Contains(finding.Detail, "goroutine start") {
				found = true
			}
		}
		if !found {
			t.Fatalf("Interface projection lost dirty callback for %s: %v", root, findings)
		}
	}
}
