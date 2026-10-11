package architecture

import (
	"go/types"
	"strings"
	"testing"
)

// These semantic regressions are independent of source-profile qualification.
// A conservative provenance finding is not evidence that a callback was visited.
func TestJSONV2CallbacksRetainDirtyBodies(t *testing.T) {
	const module = "example.invalid/jsoncallbacks"
	for _, tc := range []struct {
		name, source, callback string
	}{
		{"stream", `
func (Dirty) MarshalJSONTo(*jsontext.Encoder) error { go sideEffect();return nil }
`, "Dirty.MarshalJSONTo"},
		{"append-text", `
func (Dirty) AppendText(dst []byte)([]byte,error){go sideEffect();return dst,nil}
`, "Dirty.AppendText"},
		{"omitzero", `
type Field struct{}
type Dirty struct{ Value Field ` + "`json:\"value,omitzero\"`" + ` }
func (*Field) IsZero()bool{go sideEffect();return false}
`, "*Field.IsZero"},
		{"callback-error", `
type Failure struct{}
func (Failure) Is(error)bool{go sideEffect();return false}
func (Failure) Error()string{return "failure"}
func (Dirty) MarshalJSON()([]byte,error){return nil,Failure{}}
`, "Failure.Is"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			source := "package record\nimport \"encoding/json/jsontext\"\nvar _ *jsontext.Encoder\nfunc sideEffect(){}\nfunc Root(){}\n"
			if tc.name != "omitzero" {
				source += "type Dirty struct{}\n"
			}
			program := stockSummaryProgram(t, module, source+tc.source)
			analysis := effectResolverAnalysis(program)
			root := program.Functions[module+"/record.Root"]
			receiver := valueOf(root.pkg.Types.Scope().Lookup("Dirty").Type())
			fn := stockSummaryFunction(t, analysis, "encoding/json", "", "Marshal")
			analysis.standard(fn, nil, []*abstractValue{receiver}, root.declaration.Pos())
			visitedDirtyBody := false
			for _, finding := range analysis.findings {
				visitedDirtyBody = visitedDirtyBody || finding.Category == "host-effect" && strings.Contains(finding.Detail, "/record."+tc.callback+" -> goroutine start")
			}
			if !visitedDirtyBody {
				t.Fatalf("JSON callback body %s was not conservatively visited: %v", tc.callback, analysis.findings)
			}
		})
	}
}

func TestJSONV2DecodeRetainsDiscardedReaderErrorCallback(t *testing.T) {
	const module = "example.invalid/jsonreadererror"
	program := stockSummaryProgram(t, module, `package record
type Reader struct{}
type Failure struct{}
func (Reader) Read([]byte)(int,error){return 0,Failure{}}
func (Failure) Error()string{go sideEffect();return "failure"}
func sideEffect(){}
func Root(){}
`)
	analysis := effectResolverAnalysis(program)
	root := program.Functions[module+"/record.Root"]
	reader := valueOf(root.pkg.Types.Scope().Lookup("Reader").Type())
	constructor := stockSummaryFunction(t, analysis, "encoding/json", "", "NewDecoder")
	decoder, _ := analysis.standard(constructor, nil, []*abstractValue{reader}, root.declaration.Pos())
	decode := stockSummaryFunction(t, analysis, "encoding/json", "*Decoder", "Decode")
	analysis.standard(decode, decoder, []*abstractValue{valueOf(types.NewPointer(types.Typ[types.Int]))}, root.declaration.Pos())
	for _, finding := range analysis.findings {
		if finding.Category == "host-effect" && strings.Contains(finding.Detail, "/record.Failure.Error -> goroutine start") {
			return
		}
	}
	t.Fatalf("Decode discarded a source-reachable reader error callback: %v", analysis.findings)
}

func TestJSONV2NativeAliasesUseProvenWrapperMemory(t *testing.T) {
	const module = "example.invalid/jsonnativealiases"
	program := stockSummaryProgram(t, module, `package record
import "encoding/json"
type Data struct{ Number json.Number; Raw json.RawMessage }
func Root(){}
`)
	for _, decoding := range []bool{false, true} {
		t.Run(map[bool]string{false: "marshal", true: "unmarshal"}[decoding], func(t *testing.T) {
			analysis := effectResolverAnalysis(program)
			root := program.Functions[module+"/record.Root"]
			value := valueOf(root.pkg.Types.Scope().Lookup("Data").Type())
			name := "Marshal"
			args := []*abstractValue{value}
			if decoding {
				name = "Unmarshal"
				args = []*abstractValue{valueOf(types.NewSlice(types.Typ[types.Byte])), addressValue(value, types.NewPointer(value.types[0]), root.declaration.Pos())}
			}
			fn := stockSummaryFunction(t, analysis, "encoding/json", "", name)
			analysis.standard(fn, nil, args, root.declaration.Pos())
			if len(analysis.findings) != 0 {
				t.Fatalf("native JSON aliases in wrapper-owned memory context not qualified: %v", analysis.findings)
			}
		})
	}
}

func TestJSONV2RoutingRejectsForeignNativeDeclaration(t *testing.T) {
	const module = "example.invalid/jsonforeignentry"
	program := stockSummaryProgram(t, module, "package record\nfunc Root(){}\n")
	for _, samePackage := range []bool{false, true} {
		t.Run(map[bool]string{false: "foreign-package", true: "same-package-forged-object"}[samePackage], func(t *testing.T) {
			analysis := effectResolverAnalysis(program)
			root := program.Functions[module+"/record.Root"]
			original := stockSummaryFunction(t, analysis, "encoding/json", "", "Marshal")
			pkg := types.NewPackage("encoding/json", "json")
			if samePackage {
				pkg = original.Pkg()
			}
			forged := types.NewFunc(original.Pos(), pkg, original.Name(), original.Type().(*types.Signature))
			_, handled := analysis.standard(forged, nil, []*abstractValue{valueOf(types.Typ[types.Int])}, root.declaration.Pos())
			if !handled || len(analysis.findings) == 0 {
				t.Fatalf("forged JSON native declaration acquired source/purity permission: handled=%v findings=%v", handled, analysis.findings)
			}
		})
	}
}
