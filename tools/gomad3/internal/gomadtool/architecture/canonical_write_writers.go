package architecture

import (
	"go/ast"
	"go/token"
	"go/types"
)

// Admission does not analyze callbacks. It proves that repeated native writes
// cannot replace delegates, mutate the serialized value/input, or enable a new
// callback. The real callbacks and their returned errors are analyzed normally.
func (a *effectAnalysis) canonicalWriter(writer, value *abstractValue, pos token.Pos) bool {
	valueNodes := map[*abstractValue]bool{}
	var collect func(*abstractValue)
	collect = func(v *abstractValue) {
		if v == nil || valueNodes[v] {
			return
		}
		valueNodes[v] = true
		for _, child := range v.fields {
			collect(child)
		}
		collect(v.elements)
	}
	collect(value)
	active := map[*abstractValue]bool{}
	var certify func(*abstractValue) bool
	certify = func(v *abstractValue) bool {
		if v == nil || v.unknown || len(v.types) != 1 || len(v.functions) != 0 || active[v] || valueNodes[v] {
			return false
		}
		active[v] = true
		defer delete(active, v)
		if class, children, owned := a.canonicalNativeWriterDelegates(v, pos); owned {
			if class == "hash" {
				return true
			}
			for _, child := range children {
				if !certify(child) {
					return false
				}
			}
			return true
		}
		pointer, ok := v.types[0].(*types.Pointer)
		if !ok {
			return false
		}
		named, ok := pointer.Elem().(*types.Named)
		if !ok || named.Obj().Pkg() == nil {
			return false
		}
		path := named.Obj().Pkg().Path()
		if path == "compress/gzip" && named.Obj().Name() == "Writer" {
			method := types.NewMethodSet(v.types[0]).Lookup(nil, "Write")
			if method == nil || !a.checkCompressionSummarySource(method.Obj().(*types.Func), pos) {
				return false
			}
			state := a.compressionState(v, method.Obj().(*types.Func), pos)
			return state != nil && certify(state.fields["writer"])
		}
		pkg, literal, _ := a.canonicalWriterAllocation(v)
		if pkg == nil || !a.canonicalScalarSourceBinding(pkg.Metadata.ImportPath) {
			return false
		}
		if path == "bytes" && named.Obj().Name() == "Buffer" {
			if !a.canonicalSourceBinding("bytes") || !a.checkMemorySource("bytes", pos) {
				return false
			}
			if literal != nil && len(literal.Elts) != 0 {
				return false
			}
			expected := addressValue(zeroValue(named), v.types[0], v.origin)
			return bindingKey([]*abstractValue{v}) == bindingKey([]*abstractValue{expected})
		}
		if a.program.Metadata[path].Standard || pkg.Types != named.Obj().Pkg() || literal == nil {
			return false
		}
		structure, ok := named.Underlying().(*types.Struct)
		if !ok || v.fields["$pointee"] == nil {
			return false
		}
		var delegate *types.Var
		methods := map[*types.Func]bool{}
		for _, name := range []string{"Write", "WriteString"} {
			selection := types.NewMethodSet(v.types[0]).Lookup(nil, name)
			if selection == nil {
				if name == "Write" {
					return false
				}
				continue
			}
			fn := selection.Obj().(*types.Func)
			field, ok := a.canonicalWriterBody(fn)
			if len(selection.Index()) != 1 || !ok || (delegate != nil && delegate != field) {
				return false
			}
			delegate = field
			methods[fn] = true
		}
		if delegate == nil {
			return false
		}
		for i := 0; i < structure.NumFields(); i++ {
			field := structure.Field(i)
			child := v.fields[field.Name()]
			if field.Exported() || child == nil || child != v.fields["$pointee"].fields[field.Name()] || valueNodes[child] || child.unknown || len(child.functions) != 0 {
				return false
			}
			if field == delegate {
				if !certify(child) {
					return false
				}
			} else if !canonicalScalar(field.Type()) || field.Type().Underlying().(*types.Basic).Info()&types.IsBoolean != 0 || len(child.fields) != 0 || child.elements != nil {
				return false
			}
			if a.canonicalWriterFieldMutation(field, methods) {
				return false
			}
		}
		return true
	}
	return certify(writer)
}

// Match an actual address expression in a source-bound local function. A
// local zero Buffer is also supported; globals, new, parameters and closures
// cannot borrow this construction witness.
func (a *effectAnalysis) canonicalWriterAllocation(v *abstractValue) (*SourcePackage, *ast.CompositeLit, *ast.FuncDecl) {
	if !v.origin.IsValid() || v.nilness != 1 || !types.Identical(v.nilType, v.types[0]) {
		return nil, nil, nil
	}
	for _, pkg := range a.program.Packages {
		if pkg.Metadata.Standard {
			continue
		}
		for _, file := range pkg.Files {
			if v.origin < file.Pos() || v.origin > file.End() {
				continue
			}
			for _, declaration := range file.Decls {
				fn, ok := declaration.(*ast.FuncDecl)
				if !ok || fn.Body == nil || v.origin < fn.Body.Pos() || v.origin > fn.Body.End() {
					continue
				}
				var address *ast.UnaryExpr
				closure := false
				ast.Inspect(fn.Body, func(node ast.Node) bool {
					if nested, ok := node.(*ast.FuncLit); ok && v.origin >= nested.Pos() && v.origin <= nested.End() {
						closure = true
						return false
					}
					if unary, ok := node.(*ast.UnaryExpr); ok && unary.Pos() == v.origin && unary.Op == token.AND {
						address = unary
						return false
					}
					return true
				})
				if closure || address == nil || !types.Identical(pkg.Info.TypeOf(address), v.types[0]) {
					return nil, nil, nil
				}
				if literal, ok := address.X.(*ast.CompositeLit); ok {
					if v.fields["$pointee"] == nil || v.fields["$pointee"].origin != literal.Pos() {
						return nil, nil, nil
					}
					return pkg, literal, fn
				}
				ident, ok := address.X.(*ast.Ident)
				if !ok {
					return nil, nil, nil
				}
				object := pkg.Info.Uses[ident]
				zero := false
				ast.Inspect(fn.Body, func(node ast.Node) bool {
					if specification, ok := node.(*ast.ValueSpec); ok && len(specification.Values) == 0 {
						for _, name := range specification.Names {
							if pkg.Info.Defs[name] == object {
								zero = true
							}
						}
					}
					return true
				})
				mutated := false
				ast.Inspect(fn.Body, func(node ast.Node) bool {
					var targets []ast.Expr
					switch node := node.(type) {
					case *ast.AssignStmt:
						targets = node.Lhs
					case *ast.UnaryExpr:
						if node.Op == token.AND && node != address {
							targets = []ast.Expr{node.X}
						}
					}
					for _, target := range targets {
						if ident, ok := ast.Unparen(target).(*ast.Ident); ok && pkg.Info.Uses[ident] == object {
							mutated = true
						}
					}
					return !mutated
				})
				if zero && !mutated && v.fields["$pointee"] != nil && !v.fields["$pointee"].origin.IsValid() {
					return pkg, nil, fn
				}
				return nil, nil, nil
			}
		}
	}
	return nil, nil, nil
}

// This grammar is a restricted forwarding body, not arbitrary pure-Go proof.
// It admits scalar bounds/errors and private integer counters, but never a
// receiver/reference replacement, input write, escape or application callback.
func (a *effectAnalysis) canonicalWriterBody(fn *types.Func) (*types.Var, bool) {
	if fn == nil || fn.Pkg() == nil || !a.canonicalScalarSourceBinding(fn.Pkg().Path()) {
		return nil, false
	}
	pkg := a.program.Packages[fn.Pkg().Path()]
	declaration := a.program.Functions[functionID(fn)]
	bound, exists := a.program.canonicalSources[fn.Pkg().Path()].native.functions[functionID(fn)]
	if !exists || declaration == nil || declaration.object != fn || declaration.pkg != pkg || declaration.declaration != bound.declaration || bound.object != fn || declaration.declaration.Body == nil {
		return nil, false
	}
	signature := fn.Type().(*types.Signature)
	if signature.Recv() == nil || signature.Params().Len() != 1 || signature.Results().Len() != 2 || signature.Variadic() {
		return nil, false
	}
	receiver, input := signature.Recv(), signature.Params().At(0)
	locals := map[types.Object]bool{}
	var delegate *types.Var
	field := func(expression ast.Expr) *types.Var {
		selector, ok := expression.(*ast.SelectorExpr)
		if !ok {
			return nil
		}
		ident, ok := selector.X.(*ast.Ident)
		selection := pkg.Info.Selections[selector]
		if !ok || pkg.Info.Uses[ident] != receiver || selection == nil || selection.Kind() != types.FieldVal || len(selection.Index()) != 1 {
			return nil
		}
		object, _ := selection.Obj().(*types.Var)
		return object
	}
	var scalar func(ast.Expr) bool
	scalar = func(expression ast.Expr) bool {
		switch expression := expression.(type) {
		case *ast.ParenExpr:
			return scalar(expression.X)
		case *ast.BasicLit:
			return canonicalScalar(pkg.Info.TypeOf(expression))
		case *ast.Ident:
			object := pkg.Info.Uses[expression]
			_, constant := object.(*types.Const)
			return object == types.Universe.Lookup("nil") || constant || locals[object]
		case *ast.SelectorExpr:
			f := field(expression)
			return f != nil && !f.Exported() && canonicalScalar(f.Type())
		case *ast.BinaryExpr:
			return scalar(expression.X) && scalar(expression.Y)
		case *ast.UnaryExpr:
			return (expression.Op == token.ADD || expression.Op == token.SUB || expression.Op == token.NOT || expression.Op == token.XOR) && scalar(expression.X)
		case *ast.CallExpr:
			if len(expression.Args) != 1 || expression.Ellipsis.IsValid() {
				return false
			}
			if object, ok := expression.Fun.(*ast.Ident); ok && pkg.Info.Uses[object] == types.Universe.Lookup("len") {
				argument, ok := expression.Args[0].(*ast.Ident)
				return ok && pkg.Info.Uses[argument] == input
			}
			return pkg.Info.Types[expression.Fun].IsType() && canonicalScalar(pkg.Info.TypeOf(expression)) && scalar(expression.Args[0])
		}
		return false
	}
	forward := func(expression ast.Expr) bool {
		call, ok := expression.(*ast.CallExpr)
		if !ok || len(call.Args) != 1 || call.Ellipsis.IsValid() {
			return false
		}
		selector, ok := call.Fun.(*ast.SelectorExpr)
		if !ok || selector.Sel.Name != fn.Name() {
			return false
		}
		f := field(selector.X)
		argument, ok := call.Args[0].(*ast.Ident)
		if f == nil || f.Exported() || !ok || pkg.Info.Uses[argument] != input {
			return false
		}
		native := a.program.Packages["io"].Types.Scope().Lookup("Writer").Type()
		if fn.Name() == "WriteString" {
			native = a.program.Packages["io"].Types.Scope().Lookup("StringWriter").Type()
		}
		if !types.Identical(f.Type(), native) || (delegate != nil && delegate != f) {
			return false
		}
		delegate = f
		return true
	}
	errorResult := func(expression ast.Expr) bool {
		if scalar(expression) {
			return true
		}
		address, ok := expression.(*ast.UnaryExpr)
		if !ok || address.Op != token.AND {
			return false
		}
		literal, ok := address.X.(*ast.CompositeLit)
		if !ok {
			return false
		}
		structure, ok := pkg.Info.TypeOf(literal).Underlying().(*types.Struct)
		if !ok {
			return false
		}
		for i := 0; i < structure.NumFields(); i++ {
			if !canonicalScalar(structure.Field(i).Type()) {
				return false
			}
		}
		for _, element := range literal.Elts {
			if pair, ok := element.(*ast.KeyValueExpr); ok {
				element = pair.Value
			}
			if !scalar(element) {
				return false
			}
		}
		return true
	}
	var statements func([]ast.Stmt) bool
	statements = func(body []ast.Stmt) bool {
		for _, statement := range body {
			switch statement := statement.(type) {
			case *ast.IfStmt:
				if statement.Init != nil || !scalar(statement.Cond) || !statements(statement.Body.List) {
					return false
				}
				if statement.Else != nil {
					block, ok := statement.Else.(*ast.BlockStmt)
					if !ok || !statements(block.List) {
						return false
					}
				}
			case *ast.ReturnStmt:
				if len(statement.Results) == 1 && forward(statement.Results[0]) {
					continue
				}
				if len(statement.Results) != 2 || !scalar(statement.Results[0]) || !errorResult(statement.Results[1]) {
					return false
				}
			case *ast.AssignStmt:
				if statement.Tok == token.DEFINE && len(statement.Lhs) == 2 && len(statement.Rhs) == 1 && forward(statement.Rhs[0]) {
					for _, target := range statement.Lhs {
						ident, ok := target.(*ast.Ident)
						if !ok || pkg.Info.Defs[ident] == nil {
							return false
						}
						locals[pkg.Info.Defs[ident]] = true
					}
					continue
				}
				if statement.Tok != token.ADD_ASSIGN || len(statement.Lhs) != 1 || len(statement.Rhs) != 1 {
					return false
				}
				f := field(statement.Lhs[0])
				if f == nil || f.Exported() || !canonicalInteger(f.Type()) || !scalar(statement.Rhs[0]) {
					return false
				}
			default:
				return false
			}
		}
		return true
	}
	valid := statements(declaration.declaration.Body.List)
	return delegate, valid && delegate != nil
}

func (a *effectAnalysis) canonicalWriterFieldMutation(field *types.Var, admitted map[*types.Func]bool) bool {
	for _, pkg := range a.program.Packages {
		if pkg.Metadata.Standard {
			continue
		}
		for _, file := range pkg.Files {
			for _, declaration := range file.Decls {
				if fn, ok := declaration.(*ast.FuncDecl); ok {
					if object, _ := pkg.Info.Defs[fn.Name].(*types.Func); admitted[object] {
						continue
					}
				}
				mutated := false
				ast.Inspect(declaration, func(node ast.Node) bool {
					var expressions []ast.Expr
					switch node := node.(type) {
					case *ast.AssignStmt:
						expressions = node.Lhs
					case *ast.IncDecStmt:
						expressions = []ast.Expr{node.X}
					case *ast.UnaryExpr:
						if node.Op == token.AND {
							expressions = []ast.Expr{node.X}
						}
					}
					for _, expression := range expressions {
						if selector, ok := ast.Unparen(expression).(*ast.SelectorExpr); ok && pkg.Info.Uses[selector.Sel] == field {
							mutated = true
						}
					}
					return !mutated
				})
				if mutated {
					return true
				}
			}
		}
	}
	return false
}
