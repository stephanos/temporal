package architecture

import (
	"go/types"
	"testing"
)

var jsontextUnsupportedMethods = []struct{ receiver, name string }{
	{"*Encoder", "WriteToken"}, {"*Encoder", "WriteValue"}, {"*Encoder", "Reset"},
	{"*Decoder", "ReadToken"}, {"*Decoder", "ReadValue"}, {"*Decoder", "SkipValue"}, {"*Decoder", "Reset"},
}

func TestJSONtextUnsupportedContextsNeverGrantAuthority(t *testing.T) {
	const module = "example.invalid/jsontextunknown"
	program := stockSummaryProgram(t, module, "package record\nfunc Root(){}\n")
	for _, method := range jsontextUnsupportedMethods {
		t.Run(method.receiver+"."+method.name, func(t *testing.T) {
			for _, context := range []string{"nil", "unknown", "host", "owned-looking"} {
				t.Run(context, func(t *testing.T) {
					analysis := effectResolverAnalysis(program)
					root := program.Functions[module+"/record.Root"]
					fn := stockSummaryFunction(t, analysis, "encoding/json/jsontext", method.receiver, method.name)
					signature := fn.Type().(*types.Signature)
					receiver := valueOf(signature.Recv().Type())
					switch context {
					case "nil":
						receiver = nil
					case "unknown":
						receiver.unknown = true
					case "host":
						file := program.Packages["os"].Types.Scope().Lookup("File").Type()
						receiver.fields = map[string]*abstractValue{"transport": valueOf(types.NewPointer(file))}
					case "owned-looking":
						receiver.builtin = true
						receiver.fields = map[string]*abstractValue{"transport": {builtin: true}}
					}
					var arguments []*abstractValue
					for index := 0; index < signature.Params().Len(); index++ {
						arguments = append(arguments, valueOf(signature.Params().At(index).Type()))
					}
					returned, handled := analysis.standard(fn, receiver, arguments, root.declaration.Pos())
					if !handled || len(analysis.findings) == 0 {
						t.Fatalf("unsupported stream context was not explicitly conservative: handled=%v findings=%v", handled, analysis.findings)
					}
					assertJSONtextUnknownResult(t, returned)
				})
			}
		})
	}
}

func assertJSONtextUnknownResult(t *testing.T, value *abstractValue) {
	t.Helper()
	if value == nil {
		return
	}
	if value.builtin || value.elements != nil || len(value.functions) != 0 {
		t.Fatal("unsupported stream result acquired a trusted implementation/transport binding")
	}
	for _, typ := range value.types {
		if _, ok := typ.Underlying().(*types.Interface); ok && !value.unknown {
			t.Fatal("unsupported stream returned a known builtin error/interface")
		}
	}
	for _, child := range value.fields {
		assertJSONtextUnknownResult(t, child)
	}
}

func TestJSONtextUnsupportedSummaryRejectsForgedMethod(t *testing.T) {
	const module = "example.invalid/jsontextforeign"
	program := stockSummaryProgram(t, module, "package record\nfunc Root(){}\n")
	analysis := effectResolverAnalysis(program)
	root := program.Functions[module+"/record.Root"]
	original := stockSummaryFunction(t, analysis, "encoding/json/jsontext", "*Encoder", "WriteValue")
	for _, pkg := range []*types.Package{original.Pkg(), types.NewPackage("encoding/json/jsontext", "jsontext")} {
		forged := types.NewFunc(original.Pos(), pkg, original.Name(), original.Type().(*types.Signature))
		_, handled := analysis.standard(forged, nil, nil, root.declaration.Pos())
		if handled && len(analysis.findings) == 0 {
			t.Fatal("forged method acquired native unsupported-context handling without source rejection")
		}
	}
}
