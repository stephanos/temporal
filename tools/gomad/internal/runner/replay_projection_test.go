package runner

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"
)

func TestReplayArtifactRequestDoesNotProjectTrace(t *testing.T) {
	t.Parallel()
	file, err := parser.ParseFile(token.NewFileSet(), "replay_operation.go", nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	for _, declaration := range file.Decls {
		function, ok := declaration.(*ast.FuncDecl)
		if !ok || function.Name.Name != "choiceCapabilityForArtifact" {
			continue
		}
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
}
