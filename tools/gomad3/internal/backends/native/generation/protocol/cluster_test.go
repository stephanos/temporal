package protocol

import (
	"bytes"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"strings"
	"testing"
)

func clusterWireFixtureSource(t *testing.T, source []byte) []byte {
	t.Helper()
	source = bytes.ReplaceAll(source, []byte("history."), nil)
	files := token.NewFileSet()
	file, err := parser.ParseFile(files, "fixture.go", source, 0)
	if err != nil {
		t.Fatal(err)
	}
	names := make(map[string]string, len(clusterWireNames)+10)
	for original, exported := range clusterWireNames {
		names[exported] = original
	}
	for original, lower := range clusterWireTypes {
		names[lower] = original
	}
	for lower, original := range map[string]string{
		"RuntimeLimits": "Limits", "RuntimeNode": "NodeSpec", "RuntimeLink": "LinkSpec",
		"RuntimeVolume": "VolumeSpec", "RuntimeMount": "VolumeMount", "RuntimeReplay": "ReplayPlan",
	} {
		names[lower] = original
	}
	ast.Inspect(file, func(node ast.Node) bool {
		if ident, ok := node.(*ast.Ident); ok {
			if name, ok := names[ident.Name]; ok {
				ident.Name = name
			} else if strings.HasPrefix(ident.Name, "ProcessFrame") {
				ident.Name = "p" + ident.Name[1:]
			}
		}
		return true
	})
	var declarations []ast.Decl
	for _, declaration := range file.Decls {
		if group, ok := declaration.(*ast.GenDecl); ok && group.Tok == token.IMPORT {
			var imports []ast.Spec
			for _, spec := range group.Specs {
				if !strings.Contains(spec.(*ast.ImportSpec).Path.Value, "/cluster/internal/history") {
					imports = append(imports, spec)
				}
			}
			group.Specs = imports
			if len(imports) == 0 {
				continue
			}
		}
		declarations = append(declarations, declaration)
	}
	file.Decls = declarations
	var result bytes.Buffer
	if err := format.Node(&result, files, file); err != nil {
		t.Fatal(err)
	}
	return result.Bytes()
}
