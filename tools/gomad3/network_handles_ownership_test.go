package gomad3_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"testing"
)

// This guards the ownership boundary: adding a second backend's state to a
// public handle permits combinations that construction cannot make valid.
func TestNetworkHandlesOwnOneImplementation(t *testing.T) {
	files, err := filepath.Glob("toolchain/runtime/overlay/src/internal/gomadio/*.go")
	if err != nil {
		t.Fatal(err)
	}
	types := make(map[string]ast.Expr)
	for _, path := range files {
		if filepath.Ext(path) != ".go" || len(path) >= 8 && path[len(path)-8:] == "_test.go" {
			continue
		}
		file, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		for _, declaration := range file.Decls {
			if method, ok := declaration.(*ast.FuncDecl); ok && method.Recv != nil {
				if pointer, ok := method.Recv.List[0].Type.(*ast.StarExpr); ok {
					if receiver, ok := pointer.X.(*ast.Ident); ok && (receiver.Name == "Listener" || receiver.Name == "Conn") {
						ast.Inspect(method.Body, func(node ast.Node) bool {
							switch node.(type) {
							case *ast.IfStmt, *ast.SwitchStmt, *ast.TypeSwitchStmt, *ast.TypeAssertExpr:
								t.Errorf("%s.%s redispatches after construction", receiver.Name, method.Name.Name)
							}
							return true
						})
					}
				}
			}
			if declaration, ok := declaration.(*ast.GenDecl); ok {
				for _, spec := range declaration.Specs {
					if spec, ok := spec.(*ast.TypeSpec); ok {
						types[spec.Name.Name] = spec.Type
					}
				}
			}
		}
	}
	// An optional-state union must not escape this boundary into a helper.
	for name, declaration := range types {
		structure, ok := declaration.(*ast.StructType)
		if !ok {
			continue
		}
		process, modeled := false, false
		for _, field := range structure.Fields.List {
			for _, fieldName := range field.Names {
				if fieldName.Name == "processHandle" {
					process = true
				}
			}
			fieldType := field.Type
			if pointer, ok := fieldType.(*ast.StarExpr); ok {
				fieldType = pointer.X
			}
			if identifier, ok := fieldType.(*ast.Ident); ok {
				modeled = modeled || identifier.Name == "simulationNetwork" || identifier.Name == "simulationEndpoint" || identifier.Name == "connState" || identifier.Name == "pairedConn"
			}
		}
		if process && modeled {
			t.Errorf("%s combines remote and local modeled state", name)
		}
	}
	for _, name := range []string{"Listener", "Conn"} {
		structure, ok := types[name].(*ast.StructType)
		if !ok {
			t.Fatalf("%s must be a facade", name)
		}
		implementations := 0
		for _, field := range structure.Fields.List {
			identifier, ok := field.Type.(*ast.Ident)
			if ok {
				if contract, ok := types[identifier.Name].(*ast.InterfaceType); ok {
					implementations++
					want := 4
					if name == "Conn" {
						want = 10
					}
					if contract.Methods.NumFields() != want {
						t.Errorf("%s implementation has %d operations, want %d", name, contract.Methods.NumFields(), want)
					}
					continue
				}
			}
			selector, ok := field.Type.(*ast.SelectorExpr)
			if name == "Conn" && ok && selector.Sel.Name == "Mutex" {
				continue
			}
			t.Errorf("%s owns backend state instead of one implementation: %v", name, field.Names)
		}
		if implementations != 1 {
			t.Errorf("%s has %d implementation contracts, want one", name, implementations)
		}
	}
}
