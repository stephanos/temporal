package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
)

func main() {
	failed := false
	for _, path := range []string{"tools/gomad3/runner/runner.go", "tools/gomad3/runner/runner_local.go"} {
		set := token.NewFileSet()
		file, err := parser.ParseFile(set, path, nil, parser.ParseComments)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			panic(err)
		}
		for _, declaration := range file.Decls {
			function, ok := declaration.(*ast.FuncDecl)
			if !ok || path == "tools/gomad3/runner/runner.go" && function.Name.Name != "runLocal" {
				continue
			}
			lines := set.Position(function.End()).Line - set.Position(function.Pos()).Line + 1
			fmt.Printf("%s %s %d lines\n", path, function.Name.Name, lines)
			if lines > 150 {
				failed = true
			}
		}
	}
	if failed {
		os.Exit(1)
	}
}
