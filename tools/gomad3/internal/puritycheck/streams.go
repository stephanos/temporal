package puritycheck

import (
	"go/ast"
	"go/token"
	"go/types"
	"slices"
	"strings"
)

var streamWrappers = map[string][]string{"encoding/json": {"Decoder", "Encoder"}, "compress/gzip": {"Reader", "Writer"}, "bufio": {"Reader", "Writer", "Scanner", "ReadWriter"}}

// streamWrapper reports stdlib types whose methods read or write a stream
// supplied at construction, which the checker sees only at the constructor.
func streamWrapper(typ types.Type) bool {
	if ptr, ok := types.Unalias(typ).(*types.Pointer); ok {
		typ = ptr.Elem()
	}
	named, ok := types.Unalias(typ).(*types.Named)
	return ok && named.Obj().Pkg() != nil && slices.Contains(streamWrappers[named.Obj().Pkg().Path()], named.Obj().Name())
}

type parameter struct {
	fn    *types.Func
	index int
}

type streamIndex struct {
	assigned   map[*types.Var][]ast.Expr
	parameters map[*types.Var]parameter
	calls      map[*types.Func][]*ast.CallExpr
	uses       map[*types.Func]int
}

// localStream accepts a direct stdlib wrapper constructor call, a local
// variable whose every assignment is one, or a parameter of an unexported
// function that only pure call sites invoke with one. Other parameters,
// fields and globals carry a stream the constructor check never saw.
func (a *analysis) localStream(pkg *sourcePackage, expr ast.Expr, seen map[*types.Var]bool) bool {
	switch expr := unparen(expr).(type) {
	case *ast.CallExpr:
		fn := staticFunction(pkg, expr.Fun)
		return fn != nil && fn.Signature().Recv() == nil && fn.Pkg() != nil && streamWrappers[fn.Pkg().Path()] != nil && strings.HasPrefix(fn.Name(), "New")
	case *ast.Ident:
		v, ok := pkg.info.ObjectOf(expr).(*types.Var)
		if !ok || v.Parent() == nil || v.Parent() == pkg.types.Scope() {
			return false
		}
		if seen[v] {
			return true
		}
		seen[v] = true
		index := a.streamIndex(pkg)
		assignments, ok := index.assigned[v]
		for _, value := range assignments {
			ok = ok && value != nil && a.localStream(pkg, value, seen)
		}
		if site, found := index.parameters[v]; found {
			calls := index.calls[site.fn]
			ok = ok && len(calls) != 0 && len(calls) == index.uses[site.fn]
			for _, call := range calls {
				ok = ok && len(call.Args) == site.fn.Signature().Params().Len() && a.p.pure(a.p.relative(a.p.fset.Position(call.Pos()).Filename)) && a.localStream(pkg, call.Args[site.index], seen)
			}
		}
		return ok
	}
	return false
}

// streamIndex records, per package, the values assigned to each variable (nil
// for range variables, address-taken variables and parameters not provable
// through their call sites) and the call sites of unexported functions.
func (a *analysis) streamIndex(pkg *sourcePackage) *streamIndex {
	if index := a.streams[pkg]; index != nil {
		return index
	}
	index := &streamIndex{map[*types.Var][]ast.Expr{}, map[*types.Var]parameter{}, map[*types.Func][]*ast.CallExpr{}, map[*types.Func]int{}}
	record := func(ident ast.Expr, value ast.Expr) {
		if ident, ok := ident.(*ast.Ident); ok {
			if v, ok := pkg.info.ObjectOf(ident).(*types.Var); ok {
				index.assigned[v] = append(index.assigned[v], value)
			}
		}
	}
	for _, file := range pkg.files {
		ast.Inspect(file, func(node ast.Node) bool {
			switch node := node.(type) {
			case *ast.FuncDecl:
				fn, _ := pkg.info.Defs[node.Name].(*types.Func)
				if fn == nil || node.Recv != nil || fn.Exported() || fn.Signature().Variadic() {
					return true
				}
				position := 0
				for _, field := range node.Type.Params.List {
					for _, name := range field.Names {
						if v, ok := pkg.info.Defs[name].(*types.Var); ok {
							index.assigned[v], index.parameters[v] = []ast.Expr{}, parameter{fn, position}
						}
						position++
					}
				}
				index.uses[fn] = 0
			case *ast.CallExpr:
				if ident, ok := unparen(node.Fun).(*ast.Ident); ok {
					if fn, ok := pkg.info.Uses[ident].(*types.Func); ok {
						index.calls[fn] = append(index.calls[fn], node)
					}
				}
			case *ast.AssignStmt:
				for i, lhs := range node.Lhs {
					record(lhs, node.Rhs[min(i, len(node.Rhs)-1)])
				}
			case *ast.ValueSpec:
				for i, name := range node.Names {
					if v, ok := pkg.info.Defs[name].(*types.Var); ok && index.assigned[v] == nil {
						index.assigned[v] = []ast.Expr{}
					}
					if len(node.Values) > 0 {
						record(name, node.Values[min(i, len(node.Values)-1)])
					}
				}
			case *ast.Field:
				for _, name := range node.Names {
					if v, _ := pkg.info.Defs[name].(*types.Var); v == nil || index.parameters[v].fn == nil {
						record(name, nil)
					}
				}
			case *ast.RangeStmt:
				record(node.Key, nil)
				record(node.Value, nil)
			case *ast.UnaryExpr:
				if node.Op == token.AND {
					record(unparen(node.X), nil)
				}
			}
			return true
		})
	}
	for _, obj := range pkg.info.Uses {
		if fn, ok := obj.(*types.Func); ok {
			if _, tracked := index.uses[fn]; tracked {
				index.uses[fn]++
			}
		}
	}
	a.streams[pkg] = index
	return index
}
