package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"os"

	"go.temporal.io/server/tools/gomad3/internal/gomadtool/architecture"
)

func main() {
	files := token.NewFileSet()
	file, err := parser.ParseFile(files, "alias.go", "package runner\ntype PublicAlias = string\n", 0)
	if err != nil {
		panic(err)
	}
	oldRejected := false
	ast.Inspect(file, func(node ast.Node) bool {
		if spec, ok := node.(*ast.TypeSpec); ok && spec.Name.IsExported() && spec.Assign.IsValid() {
			oldRejected = true
		}
		return true
	})
	const path = "go.temporal.io/server/tools/gomad3/runner"
	config := types.Config{}
	pkg, err := config.Check(path, files, []*ast.File{file}, nil)
	if err != nil {
		panic(err)
	}
	program := architecture.Program{Module: "go.temporal.io/server/tools/gomad3", Packages: map[string]*architecture.SourcePackage{path: {Types: pkg}}}
	findings := program.PublicSignatures("example.com/gomad-runner-consumer")
	fmt.Printf("input=type PublicAlias = string; original_exported_alias_predicate_rejects=%t; current_public_signature_findings=%d\n", oldRejected, len(findings))
	if oldRejected && len(findings) == 0 {
		fmt.Fprintln(os.Stderr, "FAIL retained alias assertion: current public-signature guard accepts an exported builtin alias rejected by the original test")
		os.Exit(1)
	}
}
