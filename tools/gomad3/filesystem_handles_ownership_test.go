package gomad3_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"strings"
	"testing"
)

func TestFilesystemHandlesOwnOneImplementation(t *testing.T) {
	files, err := filepath.Glob("toolchain/runtime/overlay/src/internal/gomadfs/*.go")
	if err != nil {
		t.Fatal(err)
	}
	types := make(map[string]ast.Expr)
	for _, path := range files {
		if strings.HasSuffix(path, "_test.go") {
			continue
		}
		file, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		for _, declaration := range file.Decls {
			if method, ok := declaration.(*ast.FuncDecl); ok && method.Recv != nil {
				if pointer, ok := method.Recv.List[0].Type.(*ast.StarExpr); ok {
					if receiver, ok := pointer.X.(*ast.Ident); ok && (receiver.Name == "Handle" || receiver.Name == "Mapping") {
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
	for name, declaration := range types {
		structure, ok := declaration.(*ast.StructType)
		if !ok {
			continue
		}
		remote, local := false, false
		for _, field := range structure.Fields.List {
			for _, name := range field.Names {
				remote = remote || name.Name == "processHandle"
			}
			if pointer, ok := field.Type.(*ast.StarExpr); ok {
				if name, ok := pointer.X.(*ast.Ident); ok {
					local = local || name.Name == "FS" || name.Name == "node"
				}
			}
		}
		if remote && local {
			t.Errorf("%s combines remote and local filesystem ownership", name)
		}
	}
	for name, operations := range map[string]int{"Handle": 15, "Mapping": 2} {
		structure, ok := types[name].(*ast.StructType)
		if !ok {
			t.Fatalf("%s must be a facade", name)
		}
		implementations := 0
		for _, field := range structure.Fields.List {
			if identifier, ok := field.Type.(*ast.Ident); ok {
				if contract, ok := types[identifier.Name].(*ast.InterfaceType); ok {
					implementations++
					if contract.Methods.NumFields() != operations {
						t.Errorf("%s has %d operations, want %d", name, contract.Methods.NumFields(), operations)
					}
					continue
				}
			}
			t.Errorf("%s owns optional backend state: %v", name, field.Names)
		}
		if implementations != 1 {
			t.Errorf("%s has %d implementation contracts, want one", name, implementations)
		}
	}
}
