package gomad3_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

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
