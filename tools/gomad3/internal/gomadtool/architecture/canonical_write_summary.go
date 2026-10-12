package architecture

import (
	"go/ast"
	"go/token"
	"go/types"
)

// This boundary is intentionally narrower than the encoder's runtime API.
// Unsupported values/writers remain unresolved rather than borrowing authority
// from a native name, an interface binding, or a callback-free flag.
func (a *effectAnalysis) canonicalWriteSummary(fn *types.Func, args []*abstractValue, pos token.Pos) (*abstractValue, bool) {
	if fn == nil || fn.Pkg() == nil || fn.Pkg().Path() != canonicalSummaryPath || fn.Name() != "WriteCanonicalJSON" {
		return nil, false
	}
	result := modeledResults(fn)
	if !a.checkCanonicalSummarySource(fn, pos) {
		return result, true
	}
	if len(args) != 2 || !a.canonicalValue(args[1], pos) {
		a.report("unresolved-effect", pos, "canonical write value has no closed, immutable callback certificate")
		return result, true
	}
	admitted := a.canonicalFreshBuffer(fn, args[0], pos) || a.canonicalWriter(args[0], args[1], pos)
	if !admitted {
		a.report("unresolved-effect", pos, "canonical write writer has no fresh constructor/delegate certificate")
	}
	// Arbitrarily large admitted strings/slices can reach bytes.ErrTooLarge.
	// Its exported error binding cannot inherit primitive callback authority if
	// application code replaces it or takes its address.
	if a.sourceObjectMutation(a.program.Packages["bytes"].Types.Scope().Lookup("ErrTooLarge")) {
		a.report("unresolved-effect", pos, "canonical write exported buffer error has unproven mutable provenance")
	}
	// Admission only bounds writer mutation; it does not grant callback purity.
	// Analyze actual operations and preserve concrete errors even on rejected
	// chains, so unresolved admission cannot conceal host callback findings.
	result = join(result, a.compressionWriteCallbacks(args[0], pos))
	shortWrite := a.program.Packages["io"].Types.Scope().Lookup("ErrShortWrite")
	if a.sourceObjectMutation(shortWrite) {
		a.report("unresolved-effect", pos, "canonical write exported short-write error has unproven mutable provenance")
		result = join(result, a.resolve(shortWrite))
	}
	return result, true
}

func (a *effectAnalysis) canonicalFreshBuffer(fn *types.Func, writer *abstractValue, pos token.Pos) bool {
	native := a.program.Packages["bytes"]
	if native == nil || writer == nil || !a.canonicalSourceBinding("bytes") || !a.checkMemorySource("bytes", pos) {
		return false
	}
	buffer := native.Types.Scope().Lookup("Buffer")
	if buffer == nil {
		return false
	}
	expectedType := types.NewPointer(buffer.Type())
	for path, pkg := range a.program.Packages {
		if pkg.Metadata.Standard || a.program.Files.Position(pos).Filename == "" {
			continue
		}
		var call *ast.CallExpr
		for _, file := range pkg.Files {
			if pos < file.Pos() || pos > file.End() {
				continue
			}
			ast.Inspect(file, func(node ast.Node) bool {
				if candidate, ok := node.(*ast.CallExpr); ok && candidate.Pos() == pos {
					call = candidate
					return false
				}
				return call == nil
			})
		}
		if call == nil || calledObject(pkg.Info, call.Fun) != fn.Origin() || len(call.Args) != 2 || call.Ellipsis.IsValid() ||
			!a.canonicalScalarSourceBinding(path) {
			continue
		}
		expression := call.Args[0]
		for {
			parenthesized, ok := expression.(*ast.ParenExpr)
			if !ok {
				break
			}
			expression = parenthesized.X
		}
		address, ok := expression.(*ast.UnaryExpr)
		if !ok || address.Op != token.AND || !types.Identical(pkg.Info.TypeOf(address), expectedType) {
			return false
		}
		literal, ok := address.X.(*ast.CompositeLit)
		if !ok || len(literal.Elts) != 0 || !types.Identical(pkg.Info.TypeOf(literal), buffer.Type()) {
			return false
		}
		if writer.origin != address.Pos() || writer.fields["$pointee"] == nil || writer.fields["$pointee"].origin != literal.Pos() {
			return false
		}
		pointee := zeroValue(buffer.Type())
		pointee.origin = literal.Pos()
		expected := addressValue(pointee, expectedType, address.Pos())
		return bindingKey([]*abstractValue{writer}) == bindingKey([]*abstractValue{expected})
	}
	return false
}
