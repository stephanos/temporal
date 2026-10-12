package architecture

import (
	"go/ast"
	"go/types"
	"testing"
)

func TestCanonicalNativeWriterShortWriteInitializerProvenance(t *testing.T) {
	analysis, multi, _, _, pos := canonicalNativeWriterConstructPair(t)
	method := types.NewMethodSet(multi.types[0]).Lookup(nil, "Write").Obj().(*types.Func)
	returned, handled := analysis.canonicalNativeWriterOperation(method, multi, []*abstractValue{valueOf(types.NewSlice(types.Typ[types.Byte]))}, pos)
	if !handled || returned == nil || returned.fields["1"] == nil {
		t.Fatalf("source-bound MultiWriter operation did not return its error graph: %v", analysis.findings)
	}
	errorValue := returned.fields["1"]
	errorString := analysis.program.Packages["errors"].Types.Scope().Lookup("errorString")
	concrete := false
	for _, typ := range errorValue.types {
		concrete = concrete || types.Identical(typ, types.NewPointer(errorString.Type()))
	}
	if errorValue.unknown || !concrete {
		t.Errorf("pristine parsed ErrShortWrite initializer lost its concrete source-bound error: unknown=%v concrete=%v types=%v", errorValue.unknown, concrete, errorValue.types)
	}
	analysis.errorCallbacks(errorValue, nil, pos, map[string]bool{}, "Error", "Is", "As")
	if len(analysis.findings) != 0 {
		t.Fatalf("known native initializer became an unknown application callback: %v", analysis.findings)
	}
}

func TestCanonicalNativeWriterShortWriteRejectsUnprovenInitializer(t *testing.T) {
	for _, name := range []string{"initializer-replacement", "initializer-in-place", "io-uses", "import-redirect", "constructor-declaration", "constructor-body-in-place", "constructor-uses"} {
		t.Run(name, func(t *testing.T) {
			analysis, _, _, _, pos := canonicalNativeWriterConstructPair(t)
			ioPackage, errorsPackage := analysis.program.Packages["io"], analysis.program.Packages["errors"]
			sentinel := ioPackage.Types.Scope().Lookup("ErrShortWrite")
			var initializer *types.Initializer
			for _, candidate := range ioPackage.Info.InitOrder {
				if len(candidate.Lhs) == 1 && candidate.Lhs[0] == sentinel {
					initializer = candidate
				}
			}
			if initializer == nil {
				t.Fatal("actual stock sentinel initializer missing")
			}
			call := initializer.Rhs.(*ast.CallExpr)
			constructor := analysis.program.Functions["errors.New"]
			switch name {
			case "initializer-replacement":
				copy := *call
				initializer.Rhs = &copy
			case "initializer-in-place":
				call.Args[0].(*ast.BasicLit).Value = `"forged"`
			case "io-uses":
				identifier := call.Fun.(*ast.SelectorExpr).Sel
				if ioPackage.Info.Uses[identifier] == types.Universe.Lookup("nil") {
					t.Fatal("Uses tamper would not change the actual initializer")
				}
				ioPackage.Info.Uses[identifier] = types.Universe.Lookup("nil")
			case "import-redirect":
				metadata := analysis.program.Metadata["io"]
				metadata.ImportMap = map[string]string{"errors": "example.invalid/forged/errors"}
				analysis.program.Metadata["io"] = metadata
			case "constructor-declaration":
				copy, declaration := *constructor, *constructor.declaration
				copy.declaration = &declaration
				analysis.program.Functions["errors.New"] = &copy
			case "constructor-body-in-place":
				constructor.declaration.Body.List = nil
			case "constructor-uses":
				changed := false
				ast.Inspect(constructor.declaration.Body, func(node ast.Node) bool {
					identifier, ok := node.(*ast.Ident)
					if ok && identifier.Name == "text" && errorsPackage.Info.Uses[identifier] != nil {
						errorsPackage.Info.Uses[identifier] = types.Universe.Lookup("nil")
						changed = true
					}
					return true
				})
				if !changed {
					t.Fatal("actual constructor parameter use absent")
				}
			}
			value := analysis.canonicalNativeShortWriteError(pos)
			if value == nil || !value.unknown || !compressionHasFinding(analysis, "unresolved-effect", "short-write initializer has unproven source binding") {
				t.Fatalf("forged initializer acquired primitive error authority: value=%#v findings=%v", value, analysis.findings)
			}
		})
	}
	for _, test := range []struct{ name, parameter, declarations, mutation, consume string }{
		{"unknown-assignment", "cause error", "", "io.ErrShortWrite=cause;", ""},
		{"address-escape", "", "var escaped *error", "escaped=&io.ErrShortWrite;", ""},
		{"mutated-unwrap", "", "type Failure struct{};func(Failure)Error()string{return `failure`};func(Failure)Unwrap()error{go host();return nil};func host(){}", "io.ErrShortWrite=Failure{};", "_ = errors.Is(err,io.EOF);"},
	} {
		t.Run(test.name, func(t *testing.T) {
			source := `package record
import("bytes";"io";"errors";"go.temporal.io/server/tools/gomad3/internal/canonicaljson")
var _=canonicaljson.WriteCanonicalJSON
var _=errors.Is
` + test.declarations + "\nfunc Root(" + test.parameter + ")error{" + test.mutation + `writer:=io.MultiWriter(&bytes.Buffer{},&bytes.Buffer{});_,err:=writer.Write([]byte("x"));` + test.consume + `return err}`
			program := canonicalWriteFixtureBeforeImport(t, source, func(program *Program) {
				metadata := program.Metadata[program.Module+"/record"]
				metadata.Imports = []string{"bytes", "io", "errors", canonicalWritePackage}
				program.Metadata[program.Module+"/record"] = metadata
			})
			analysis, returned := canonicalWriterAnalyzeRoot(t, program)
			if !compressionHasFinding(analysis, "unresolved-effect", "exported short-write error has unproven mutable provenance") {
				t.Fatalf("mutable sentinel was certified: %v", analysis.findings)
			}
			if test.name == "mutated-unwrap" {
				if !compressionHasFinding(analysis, "host-effect", "Failure.Unwrap -> goroutine start") {
					t.Fatalf("mutable sentinel lost its concrete Unwrap callback: %v", analysis.findings)
				}
			} else if returned == nil || !returned.unknown {
				t.Fatalf("mutable/unknown sentinel lost uncertainty: returned=%#v findings=%v", returned, analysis.findings)
			}
		})
	}
}
