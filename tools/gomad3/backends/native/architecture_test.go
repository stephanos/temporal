package native

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
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

func TestProcessCommandsOwnModelWireTranslation(t *testing.T) {
	for _, domain := range []string{"gomadio", "gomadfs"} {
		directory := filepath.Join("toolchain/runtime/overlay/src/internal", domain)
		entries, err := os.ReadDir(directory)
		if err != nil {
			t.Fatal(err)
		}
		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") || strings.HasSuffix(entry.Name(), "_test.go") || entry.Name() == "process_commands.go" {
				continue
			}
			name := filepath.Join(directory, entry.Name())
			fset := token.NewFileSet()
			file, err := parser.ParseFile(fset, name, nil, 0)
			if err != nil {
				t.Fatal(err)
			}
			ast.Inspect(file, func(node ast.Node) bool {
				switch value := node.(type) {
				case *ast.ImportSpec:
					if value.Path.Value == `"internal/gomadmodelwire"` {
						t.Errorf("%s: wire translation escaped its owner", fset.Position(value.Pos()))
					}
				case *ast.SelectorExpr:
					if strings.HasPrefix(value.Sel.Name, "String") || strings.HasPrefix(value.Sel.Name, "Int") || strings.HasPrefix(value.Sel.Name, "Uint") {
						if value.Sel.Name == "String1" || value.Sel.Name == "String2" || value.Sel.Name == "Int1" || value.Sel.Name == "Int2" || value.Sel.Name == "Uint1" || value.Sel.Name == "Uint2" {
							t.Errorf("%s: generic slot escaped its owner", fset.Position(value.Pos()))
						}
					}
				}
				return true
			})
		}
	}
}
