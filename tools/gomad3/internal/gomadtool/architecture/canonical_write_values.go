package architecture

import (
	"go/ast"
	"go/token"
	"go/types"
)

func canonicalScalar(typ types.Type) bool {
	if typ == nil {
		return false
	}
	basic, ok := typ.Underlying().(*types.Basic)
	return ok && basic.Info()&(types.IsBoolean|types.IsInteger|types.IsFloat|types.IsString) != 0
}

func canonicalInteger(typ types.Type) bool {
	if typ == nil {
		return false
	}
	basic, ok := typ.Underlying().(*types.Basic)
	return ok && basic.Info()&types.IsInteger != 0
}

// Certification is type-stable under writer interleaving: no dynamic slots or
// mutating hooks are admitted, even if their current represented binding is clean.
func (a *effectAnalysis) canonicalValue(v *abstractValue, pos token.Pos) bool {
	seenTypes := map[types.Type]int{}
	activeFunctions := map[*types.Func]bool{}
	var closed func(types.Type) bool
	closed = func(typ types.Type) bool {
		if typ == nil || seenTypes[typ] == 1 {
			return false
		}
		if seenTypes[typ] == 2 {
			return true
		}
		seenTypes[typ] = 1
		for _, methodType := range []types.Type{typ, types.NewPointer(typ)} {
			methods := types.NewMethodSet(methodType)
			for _, name := range []string{"MarshalJSON", "MarshalText", "AppendText", "MarshalJSONTo", "IsZero"} {
				if selection := methods.Lookup(nil, name); selection != nil {
					fn := selection.Obj().(*types.Func)
					signature := fn.Type().(*types.Signature)
					if len(selection.Index()) != 1 || (name != "MarshalJSON" && name != "MarshalText" && name != "IsZero") ||
						!canonicalInteger(signature.Recv().Type()) || !a.jsonCallbackSignature(fn) ||
						!a.canonicalScalarFunction(fn, activeFunctions, pos) {
						return false
					}
				}
			}
		}
		switch underlying := typ.Underlying().(type) {
		case *types.Basic:
			if !canonicalScalar(typ) {
				return false
			}
		case *types.Pointer:
			if !closed(underlying.Elem()) {
				return false
			}
		case *types.Slice:
			if !closed(underlying.Elem()) {
				return false
			}
		case *types.Array:
			if !closed(underlying.Elem()) {
				return false
			}
		case *types.Map:
			if !closed(underlying.Key()) || !closed(underlying.Elem()) {
				return false
			}
		case *types.Struct:
			for index := 0; index < underlying.NumFields(); index++ {
				if !closed(underlying.Field(index).Type()) {
					return false
				}
			}
		default:
			return false
		}
		seenTypes[typ] = 2
		return true
	}
	seen := map[*abstractValue]bool{}
	var graph func(*abstractValue) bool
	graph = func(value *abstractValue) bool {
		if value == nil || seen[value] {
			return true
		}
		seen[value] = true
		if value.unknown || len(value.functions) != 0 {
			return false
		}
		for _, typ := range value.types {
			if !closed(typ) {
				return false
			}
		}
		for _, child := range value.fields {
			if !graph(child) {
				return false
			}
		}
		return graph(value.elements)
	}
	return v != nil && len(v.types) != 0 && graph(v)
}

func (a *effectAnalysis) canonicalScalarFunction(fn *types.Func, active map[*types.Func]bool, pos token.Pos) bool {
	if fn == nil || fn.Pkg() == nil || !a.canonicalScalarSourceBinding(fn.Pkg().Path()) {
		return false
	}
	fn = fn.Origin()
	decl := a.program.Functions[functionID(fn)]
	bound := a.program.canonicalSources[fn.Pkg().Path()].native.functions[functionID(fn)]
	if active[fn] || decl == nil || decl.object != fn || decl.pkg != a.program.Packages[fn.Pkg().Path()] ||
		bound.object != fn || bound.declaration != decl.declaration || decl.pkg.Types != fn.Pkg() ||
		decl.pkg.Info.Defs[decl.declaration.Name] != fn || decl.declaration.Body == nil || len(decl.declaration.Body.List) != 1 {
		return false
	}
	signature := fn.Type().(*types.Signature)
	if signature.Variadic() || signature.TypeParams().Len() != 0 {
		return false
	}
	parameters := map[types.Object]bool{}
	if receiver := signature.Recv(); receiver != nil {
		if !canonicalInteger(receiver.Type()) {
			return false
		}
		parameters[receiver] = true
	}
	for index := 0; index < signature.Params().Len(); index++ {
		parameter := signature.Params().At(index)
		if !canonicalScalar(parameter.Type()) {
			return false
		}
		parameters[parameter] = true
	}
	returned, ok := decl.declaration.Body.List[0].(*ast.ReturnStmt)
	if !ok || len(returned.Results) != signature.Results().Len() || len(returned.Results) == 0 || len(returned.Results) > 2 {
		return false
	}
	if len(returned.Results) == 2 {
		nilError, ok := returned.Results[1].(*ast.Ident)
		if !ok || decl.pkg.Info.Uses[nilError] != types.Universe.Lookup("nil") || !types.Identical(signature.Results().At(1).Type(), types.Universe.Lookup("error").Type()) {
			return false
		}
	}
	active[fn] = true
	defer delete(active, fn)
	var expression func(ast.Expr) bool
	expression = func(expr ast.Expr) bool {
		switch expr := expr.(type) {
		case *ast.ParenExpr:
			return expression(expr.X)
		case *ast.BasicLit:
			return canonicalScalar(decl.pkg.Info.TypeOf(expr))
		case *ast.Ident:
			object := decl.pkg.Info.Uses[expr]
			_, constant := object.(*types.Const)
			return object != nil && (parameters[object] || constant) && canonicalScalar(object.Type())
		case *ast.UnaryExpr:
			return (expr.Op == token.ADD || expr.Op == token.SUB || expr.Op == token.XOR || expr.Op == token.NOT) && expression(expr.X)
		case *ast.BinaryExpr:
			return expression(expr.X) && expression(expr.Y)
		case *ast.CallExpr:
			if expr.Ellipsis.IsValid() {
				return false
			}
			for _, argument := range expr.Args {
				if !expression(argument) {
					return false
				}
			}
			if decl.pkg.Info.Types[expr.Fun].IsType() {
				typ := decl.pkg.Info.TypeOf(expr.Fun)
				if canonicalScalar(typ) {
					return len(expr.Args) == 1
				}
				if typ == nil || len(expr.Args) != 1 || decl.pkg.Info.TypeOf(expr.Args[0]) == nil {
					return false
				}
				slice, ok := typ.Underlying().(*types.Slice)
				if !ok || !types.Identical(slice.Elem(), types.Typ[types.Byte]) || len(expr.Args) != 1 {
					return false
				}
				operand, ok := decl.pkg.Info.TypeOf(expr.Args[0]).Underlying().(*types.Basic)
				return ok && operand.Info()&types.IsString != 0
			}
			called := calledObject(decl.pkg.Info, expr.Fun)
			if called == nil || called.Pkg() == nil || called.Type().(*types.Signature).Recv() != nil {
				return false
			}
			if called.Pkg().Path() == "strconv" {
				name := called.Name()
				native := a.program.Packages["strconv"]
				return (name == "FormatUint" || name == "FormatInt" || name == "Quote") && native != nil &&
					called.Origin() == native.Types.Scope().Lookup(name) && a.canonicalSourceBinding("strconv") && a.checkMemorySource("strconv", pos)
			}
			return called.Pkg() == fn.Pkg() && a.canonicalScalarFunction(called, active, pos)
		}
		return false
	}
	return expression(returned.Results[0])
}
