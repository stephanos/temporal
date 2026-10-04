package cli

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"
)

func TestCampaignSemanticValidationBelongsToRunner(t *testing.T) {
	files := token.NewFileSet()
	file, err := parser.ParseFile(files, "cli.go", nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	for _, rule := range []struct {
		function string
		calls    map[string]int
		forbid   map[string]bool
	}{
		{"resolveExploreStrategy", map[string]int{"ParseSingleBaseSeed": 2}, map[string]bool{"Count": true, "ParseSeeds": true}},
		{"resolveExploreCoverage", map[string]int{"ValidateCoverage": 1}, map[string]bool{"MissingRequiredSemanticProbes": true, "CoverageNone": true, "CoverageChoice": true, "CoverageSemantic": true, "CoverageSemanticChoice": true}},
		{"resolveChoiceTrace", map[string]int{"ValidateChoiceTraceLimit": 1}, nil},
		{"parseCampaignRequest", map[string]int{"ValidateChoiceCoverage": 1}, map[string]bool{"CoverageChoice": true, "CoverageSemanticChoice": true}},
		{"resolveExploreGuidance", map[string]int{"NormalizeCoverage": 1}, nil},
	} {
		t.Run(rule.function, func(t *testing.T) {
			var body *ast.BlockStmt
			for _, declaration := range file.Decls {
				if function, ok := declaration.(*ast.FuncDecl); ok && function.Name.Name == rule.function {
					body = function.Body
				}
			}
			if body == nil {
				t.Fatal("campaign validation function is missing")
			}
			calls := make(map[string]int)
			ast.Inspect(body, func(node ast.Node) bool {
				if comparison, ok := node.(*ast.BinaryExpr); ok && rule.function == "resolveChoiceTrace" {
					ast.Inspect(comparison, func(node ast.Node) bool {
						if selector, ok := node.(*ast.SelectorExpr); ok && (selector.Sel.Name == "MinimumChoiceTraceBytes" || selector.Sel.Name == "MaximumChoiceTraceBytes") {
							t.Errorf("%s: CLI duplicates Runner's trace capacity rule", files.Position(selector.Pos()))
						}
						return true
					})
				}
				if selector, ok := node.(*ast.SelectorExpr); ok && rule.forbid[selector.Sel.Name] {
					t.Errorf("%s: CLI duplicates Runner's semantic responsibility through %s", files.Position(selector.Pos()), selector.Sel.Name)
				}
				var expressions []ast.Expr
				switch value := node.(type) {
				case *ast.IfStmt:
					condition, ok := value.Cond.(*ast.BinaryExpr)
					if !ok || condition.Op != token.NEQ {
						break
					}
					left, leftOK := condition.X.(*ast.Ident)
					right, rightOK := condition.Y.(*ast.Ident)
					if !leftOK || !rightOK || left.Name != "err" || right.Name != "nil" {
						break
					}
					if assignment, ok := value.Init.(*ast.AssignStmt); ok {
						expressions = assignment.Rhs
					}
				case *ast.ReturnStmt:
					expressions = value.Results
				}
				for _, expression := range expressions {
					ast.Inspect(expression, func(node ast.Node) bool {
						if call, ok := node.(*ast.CallExpr); ok {
							if selector, ok := call.Fun.(*ast.SelectorExpr); ok {
								if owner, ok := selector.X.(*ast.Ident); ok && owner.Name == "runner" {
									calls[selector.Sel.Name]++
								}
							}
						}
						return true
					})
				}
				return true
			})
			for name, want := range rule.calls {
				if got := calls[name]; got != want {
					t.Errorf("%s: %d checked Runner.%s calls, want %d in actual validation", rule.function, got, name, want)
				}
			}
		})
	}
}
