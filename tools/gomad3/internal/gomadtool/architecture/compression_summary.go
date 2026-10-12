package architecture

import (
	"go/ast"
	"go/token"
	"go/types"
)

// Constructor authority is local to the represented pointer location. The
// private state is part of the normal alias/clone/convergence graph, not a
// path-or-native-type permission to skip arbitrary compression calls.
func (a *effectAnalysis) compressionSummary(fn *types.Func, receiver *abstractValue, args []*abstractValue, pos token.Pos) (*abstractValue, bool) {
	if fn == nil || fn.Pkg() == nil || fn.Pkg().Path() != "compress/gzip" {
		return nil, false
	}
	id := functionID(fn)
	switch id {
	case "compress/gzip.NewWriter", "compress/gzip.*Writer.Write", "compress/gzip.*Writer.Flush", "compress/gzip.*Writer.Close", "compress/gzip.*Writer.Reset":
	default:
		return nil, false
	}
	result := modeledResults(fn)
	if !a.checkCompressionSummarySource(fn, pos) {
		return result, true
	}
	if id == "compress/gzip.NewWriter" {
		typ := fn.Type().(*types.Signature).Results().At(0).Type()
		pointer := typ.(*types.Pointer)
		result = addressValue(zeroValue(pointer.Elem()), typ, pos)
		state := &abstractValue{fields: map[string]*abstractValue{}}
		if len(args) != 0 {
			state.fields["writer"] = args[0]
		}
		result.fields["$gzip-state"] = state
		result.gzipOwned = a.bindingDepth == 0
		return result, true
	}
	state := a.compressionState(receiver, fn, pos)
	if state == nil {
		return result, true
	}
	if id == "compress/gzip.*Writer.Reset" {
		if len(args) != 1 || args[0] == nil {
			a.report("unresolved-effect", pos, "compression Reset has unknown writer binding")
		} else {
			// Retaining the old writer/errors is conservative across alias and
			// branch joins; replacing the graph would lose other pointer aliases.
			state.fields["writer"] = join(state.fields["writer"], args[0])
		}
		return result, true
	}
	if a.compressionActive == nil {
		a.compressionActive = map[*abstractValue]bool{}
	}
	if a.compressionActive[state] {
		a.report("unresolved-effect", pos, "recursive compression writer callback mutation")
		return result, true
	}
	a.compressionActive[state] = true
	defer delete(a.compressionActive, state)
	// Native gzip can write more than once even on Close without a prior
	// Write. Saturate represented receiver/capture mutations, not just a
	// single callback whose first invocation might change a future branch.
	for round := 0; round < 256; round++ {
		before := bindingKey([]*abstractValue{state})
		errors := a.compressionWriteCallbacks(state.fields["writer"], pos)
		state.fields["error"] = join(state.fields["error"], errors)
		if a.compressionState(receiver, fn, pos) == nil {
			return result, true
		}
		if bindingKey([]*abstractValue{state}) == before {
			break
		}
		if round == 255 {
			a.report("unresolved-effect", pos, "compression writer callback mutations did not converge")
		}
	}
	// Conservatively retain exported-error uncertainty around modeled writer
	// implementations. Application writes/address escapes cannot acquire a
	// primitive-error permission merely from the standard package name.
	if a.compressionShortWriteMutation() {
		a.report("unresolved-effect", pos, "compression exported short-write error has unproven mutable provenance")
		state.fields["error"] = join(state.fields["error"], valueOf(types.Universe.Lookup("error").Type()))
	}
	if id == "compress/gzip.*Writer.Write" {
		result.fields["1"] = join(result.fields["1"], state.fields["error"])
	} else {
		result = join(result, state.fields["error"])
	}
	return result, true
}

func (a *effectAnalysis) compressionShortWriteMutation() bool {
	native := a.program.Packages["io"].Types.Scope().Lookup("ErrShortWrite")
	return a.sourceObjectMutation(native)
}

func (a *effectAnalysis) sourceObjectMutation(native types.Object) bool {
	mutated := false
	for _, pkg := range a.program.Packages {
		if pkg.Metadata.Standard {
			continue
		}
		for _, file := range pkg.Files {
			ast.Inspect(file, func(node ast.Node) bool {
				if mutated {
					return false
				}
				var expressions []ast.Expr
				switch node := node.(type) {
				case *ast.AssignStmt:
					expressions = node.Lhs
				case *ast.UnaryExpr:
					if node.Op == token.AND {
						expressions = []ast.Expr{node.X}
					}
				}
				for _, expression := range expressions {
					for {
						parenthesized, ok := expression.(*ast.ParenExpr)
						if !ok {
							break
						}
						expression = parenthesized.X
					}
					var object types.Object
					switch expression := expression.(type) {
					case *ast.Ident:
						object = pkg.Info.Uses[expression]
					case *ast.SelectorExpr:
						object = pkg.Info.Uses[expression.Sel]
					}
					if object == native {
						mutated = true
					}
				}
				return true
			})
		}
	}
	return mutated
}

func (a *effectAnalysis) compressionState(receiver *abstractValue, fn *types.Func, pos token.Pos) *abstractValue {
	expected := fn.Type().(*types.Signature).Recv().Type()
	if receiver == nil || receiver.unknown || !receiver.gzipOwned || len(receiver.types) == 0 {
		a.report("unresolved-effect", pos, "compression writer has unproven constructor/location ownership")
		return nil
	}
	for _, typ := range receiver.types {
		if typ == nil || !types.Identical(typ, expected) {
			a.report("unresolved-effect", pos, "compression writer has mixed or retyped ownership")
			return nil
		}
	}
	state := receiver.fields["$gzip-state"]
	if state == nil || state.unknown || state.fields["writer"] == nil {
		a.report("unresolved-effect", pos, "compression writer has unknown mutable state")
		return nil
	}
	return state
}

func (a *effectAnalysis) compressionWriteCallbacks(writer *abstractValue, pos token.Pos) *abstractValue {
	if writer == nil || writer.unknown || len(writer.types) == 0 {
		a.report("unresolved-effect", pos, "compression has unknown output writer callbacks")
	}
	if writer == nil {
		return valueOf(types.Universe.Lookup("error").Type())
	}
	ioPackage := a.program.Packages["io"]
	if ioPackage == nil {
		a.report("unresolved-effect", pos, "compression output interface has no imported identity")
		return valueOf(types.Universe.Lookup("error").Type())
	}
	writerInterface := ioPackage.Types.Scope().Lookup("Writer").Type().Underlying().(*types.Interface)
	stringInterface := ioPackage.Types.Scope().Lookup("StringWriter").Type().Underlying().(*types.Interface)
	var errors *abstractValue
	for _, typ := range writer.types {
		if typ == nil {
			a.report("unresolved-effect", pos, "compression output has unknown concrete type")
			continue
		}
		if _, dynamic := typ.Underlying().(*types.Interface); dynamic || !types.Implements(typ, writerInterface) {
			a.report("unresolved-effect", pos, "compression output has unproven dynamic Writer binding")
			continue
		}
		copy := *writer
		copy.types = []types.Type{typ}
		bytes := valueOf(types.NewSlice(types.Typ[types.Byte]))
		bytes.elements = valueOf(types.Typ[types.Byte])
		errors = join(errors, a.compressionInvokeWriter(&copy, "Write", bytes, pos))
		// Header Name/Comment may invoke io.WriteString. Unknown header text
		// conservatively retains this branch alongside its byte Write fallback.
		if types.Implements(typ, stringInterface) {
			errors = join(errors, a.compressionInvokeWriter(&copy, "WriteString", valueOf(types.Typ[types.String]), pos))
		}
	}
	return errors
}

func (a *effectAnalysis) compressionInvokeWriter(receiver *abstractValue, name string, argument *abstractValue, pos token.Pos) *abstractValue {
	selection := types.NewMethodSet(receiver.types[0]).Lookup(nil, name)
	if selection == nil || len(selection.Index()) != 1 {
		a.report("unresolved-effect", pos, "compression writer callback has unsupported promoted receiver ownership")
		return valueOf(types.Universe.Lookup("error").Type())
	}
	fn := selection.Obj().(*types.Func)
	var returned *abstractValue
	if result, handled := a.standard(fn, receiver, []*abstractValue{argument}, pos); handled {
		returned = result
	} else if declaration := a.program.Functions[functionID(fn)]; declaration != nil && declaration.object == fn {
		returned = a.call(&boundFunction{fn: declaration, pkg: declaration.pkg, receiver: receiver}, []*abstractValue{argument}, pos)
	} else {
		a.report("unresolved-effect", pos, "compression writer callback has no native/source declaration")
	}
	if returned == nil || returned.fields["1"] == nil {
		a.report("unresolved-effect", pos, "compression writer callback has unknown error result")
		return valueOf(types.Universe.Lookup("error").Type())
	}
	return returned.fields["1"]
}
