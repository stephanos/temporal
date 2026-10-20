package runner

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"
)

func TestReplayArtifactRequestDoesNotProjectTrace(t *testing.T) {
	t.Parallel()
	for _, test := range []struct{ path, function string }{
		{"replay_operation.go", "choiceCapabilityForArtifact"},
		{"../backends/wasm/backend/inspection.go", "standaloneInspection"},
	} {
		t.Run(test.function, func(t *testing.T) {
			t.Parallel()
			file, err := parser.ParseFile(token.NewFileSet(), test.path, nil, 0)
			if err != nil {
				t.Fatal(err)
			}
			found := false
			for _, declaration := range file.Decls {
				function, ok := declaration.(*ast.FuncDecl)
				if !ok || function.Name.Name != test.function {
					continue
				}
				found = true
				ast.Inspect(function.Body, func(node ast.Node) bool {
					call, ok := node.(*ast.CallExpr)
					if !ok {
						return true
					}
					selector, ok := call.Fun.(*ast.SelectorExpr)
					if ok && (selector.Sel.Name == "DecodeStoredTrace" || selector.Sel.Name == "ProjectReplayPlan") {
						t.Errorf("artifact replay calls %s", selector.Sel.Name)
					}
					return true
				})
			}
			if !found {
				t.Fatalf("missing execution request builder %s", test.function)
			}
		})
	}
}
