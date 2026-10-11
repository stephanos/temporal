package architecture

import (
	"go/ast"
	"go/types"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func reflectSummaryFunction(path, receiver, name string, results ...types.Type) *types.Func {
	pkg := types.NewPackage(path, filepath.Base(path))
	var recv *types.Var
	if receiver != "" {
		named := types.NewNamed(types.NewTypeName(0, pkg, strings.TrimPrefix(receiver, "*"), nil), types.NewStruct(nil, nil), nil)
		var typ types.Type = named
		if strings.HasPrefix(receiver, "*") {
			typ = types.NewPointer(named)
		}
		recv = types.NewVar(0, pkg, "receiver", typ)
	}
	vars := make([]*types.Var, len(results))
	for index, typ := range results {
		vars[index] = types.NewVar(0, pkg, "", typ)
	}
	return types.NewFunc(0, pkg, name, types.NewSignatureType(recv, nil, nil, types.NewTuple(), types.NewTuple(vars...), false))
}

func reflectSummaryAnalysis(t *testing.T, source string) *effectAnalysis {
	t.Helper()
	program := effectResolverProgram(t, source)
	reflectSummarySources(t, program)
	return effectResolverAnalysis(program)
}

func reflectSummarySources(t *testing.T, program *Program) {
	t.Helper()
	importStockSummarySources(t, program)
}

var reflectPrimitiveAccessors = []struct {
	name string
	typ  types.Type
}{
	{"Bool", types.Typ[types.Bool]},
	{"Int", types.Typ[types.Int64]},
	{"Uint", types.Typ[types.Uint64]},
	{"CanAddr", types.Typ[types.Bool]},
}

func TestReflectPrimitiveSummaryPinned(t *testing.T) {
	for _, accessor := range reflectPrimitiveAccessors {
		t.Run(accessor.name, func(t *testing.T) {
			a := reflectSummaryAnalysis(t, "package record\nfunc Root(){}\n")
			pos := a.program.Functions["example.invalid/resolver/record.Root"].declaration.Pos()
			represented := valueOf(accessor.typ)
			receiver := &abstractValue{fields: map[string]*abstractValue{"represented": represented}}
			result, handled := a.standard(reflectSummaryFunction("reflect", "Value", accessor.name, accessor.typ), receiver, nil, pos)
			if !handled || result == nil || len(result.types) != 1 || !types.Identical(result.types[0], accessor.typ) {
				t.Fatalf("pinned %s not modeled as %s: handled=%v result=%#v", accessor.name, accessor.typ, handled, result)
			}
			if len(a.findings) != 0 || !a.checkedMemory["reflect"] {
				t.Fatalf("pinned accessor purity/source validation: findings=%v checked=%v", a.findings, a.checkedMemory)
			}
			if result.fields["represented"] != represented {
				t.Fatal("accessor lost represented receiver flow")
			}
		})
	}
}

func TestReflectPrimitiveSummaryRejectsSourceIdentityChanges(t *testing.T) {
	for _, variant := range []string{"nonstandard", "changed-source"} {
		for _, accessor := range reflectPrimitiveAccessors {
			t.Run(variant+"/"+accessor.name, func(t *testing.T) {
				a := reflectSummaryAnalysis(t, "package record\nfunc Root(){}\n")
				root := a.program.Functions["example.invalid/resolver/record.Root"]
				pos := root.declaration.Pos()
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
				fn := reflectSummaryFunction("reflect", "Value", accessor.name, accessor.typ)
				_, handled := a.standard(fn, nil, nil, pos)
				if variant == "nonstandard" {
					if handled {
						t.Fatal("forged nonstandard reflect accepted as a summary")
					}
					// A rejected summary must follow normal source analysis, not become a pure result.
					a.call(&boundFunction{fn: &function{object: fn, declaration: &ast.FuncDecl{Body: nil}, pkg: root.pkg}, pkg: root.pkg}, nil, pos)
				} else if !handled {
					t.Fatal("source-mutation fixture did not exercise the accessor summary")
				}
				if a.checkedMemory["reflect"] || len(a.findings) != 1 {
					t.Fatalf("source rejection findings=%v checked=%v", a.findings, a.checkedMemory)
				}
				for _, finding := range a.findings {
					if finding.Category != "unresolved-effect" || variant == "changed-source" && !strings.Contains(finding.Detail, "pinned memory summary source changed: reflect") {
						t.Fatalf("source identity did not fail closed: %v", finding)
					}
				}
			})
		}
	}
}

func TestReflectAccessorPreservesDirtyImplicitCallbacks(t *testing.T) {
	for _, tc := range []struct {
		name, source, method, path, receiver, summary string
		results                                       []types.Type
	}{
		{"json", "func (Dirty) MarshalJSON()([]byte,error){go sideEffect();return nil,nil}", "MarshalJSON", "encoding/json", "", "Marshal", []types.Type{types.NewSlice(types.Typ[types.Byte]), types.Universe.Lookup("error").Type()}},
		{"text", "func (Dirty) MarshalText()([]byte,error){go sideEffect();return nil,nil}", "MarshalText", "encoding/json", "", "Marshal", []types.Type{types.NewSlice(types.Typ[types.Byte]), types.Universe.Lookup("error").Type()}},
		{"error", "func (Dirty) Error()string{go sideEffect();return \"dirty\"}", "Error", "fmt", "", "Sprintf", []types.Type{types.Typ[types.String]}},
		{"writer", "func (Dirty) Write([]byte)(int,error){go sideEffect();return 0,nil}", "Write", "encoding/json", "*Encoder", "Encode", []types.Type{types.Universe.Lookup("error").Type()}},
		{"reader", "func (Dirty) Read([]byte)(int,error){go sideEffect();return 0,nil}", "Read", "encoding/json", "*Decoder", "Token", []types.Type{types.NewInterfaceType(nil, nil), types.Universe.Lookup("error").Type()}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a := reflectSummaryAnalysis(t, "package record\ntype Dirty struct{}\nfunc Root(){}\nfunc sideEffect(){}\n"+tc.source+"\n")
			root := a.program.Functions["example.invalid/resolver/record.Root"]
			pos := root.declaration.Pos()
			dirty := valueOf(root.pkg.Types.Scope().Lookup("Dirty").Type())
			reflected, handled := a.standard(reflectSummaryFunction("reflect", "", "ValueOf", types.Typ[types.Bool]), nil, []*abstractValue{dirty}, pos)
			if !handled {
				t.Fatal("ValueOf summary missing")
			}
			for _, accessor := range reflectPrimitiveAccessors {
				_, handled = a.standard(reflectSummaryFunction("reflect", "Value", accessor.name, accessor.typ), reflected, nil, pos)
				if !handled || len(a.findings) != 0 {
					t.Fatalf("%s summary unexpectedly invoked an application callback: handled=%v findings=%v", accessor.name, handled, a.findings)
				}
			}
			unwrapped, handled := a.standard(stockSummaryFunction(t, a, "reflect", "Value", "Interface"), reflected, nil, pos)
			if !handled {
				t.Fatal("Interface summary missing")
			}
			value := unwrapped
			args := []*abstractValue{value}
			var receiver *abstractValue
			switch tc.name {
			case "error":
				format := valueOf(types.Typ[types.String])
				format.text = "%v"
				args = []*abstractValue{format, value}
			case "reader", "writer":
				receiver = &abstractValue{fields: map[string]*abstractValue{"transport": value}}
				args = []*abstractValue{valueOf(types.Typ[types.Bool])}
			}
			_, handled = a.standard(stockSummaryFunction(t, a, tc.path, tc.receiver, tc.summary), receiver, args, pos)
			if !handled || len(a.findings) != 1 {
				t.Fatalf("dirty callback lost: handled=%v findings=%v", handled, a.findings)
			}
			for _, finding := range a.findings {
				if finding.Category != "host-effect" || !strings.Contains(finding.Detail, "record.Dirty."+tc.method) || !strings.Contains(finding.Detail, "goroutine start") {
					t.Fatalf("wrong callback finding: %v", finding)
				}
			}
		})
	}
}

func TestReflectPrimitiveASTDoesNotHideDirtyMarshaler(t *testing.T) {
	const module = "example.invalid/reflectsummary"
	program := stockSummaryProgram(t, module, `package record
import ("reflect";"encoding/json")
type Dirty struct{}
func (Dirty) MarshalJSON() ([]byte,error) { go sideEffect();return nil,nil }
func sideEffect() {}
func Root() {
 v := reflect.ValueOf(true)
 _ = v.Bool(); _ = v.Int(); _ = v.Uint(); _ = v.CanAddr()
 _,_ = json.Marshal(reflect.ValueOf(Dirty{}).Interface())
}
`)
	rootFinding := false
	for _, finding := range program.Effects() {
		if strings.HasPrefix(finding.Detail, module+"/record.Root ->") && (finding.Category == "host-effect" || finding.Category == "unresolved-effect") {
			rootFinding = true
			t.Logf("reflected marshaler remains conservative: %s %s", finding.Category, finding.Detail)
		}
	}
	if !rootFinding {
		t.Fatal("primitive reflect summaries made a reflected dirty MarshalJSON callback appear pure")
	}
}
