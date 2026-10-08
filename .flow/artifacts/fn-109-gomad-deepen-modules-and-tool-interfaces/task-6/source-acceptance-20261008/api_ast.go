package main

import (
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	files, err := filepath.Glob("tools/gomad3/runner/*.go")
	if err != nil { panic(err) }
	variables := []string{}
	seams := map[string]bool{"Preparer": false, "ArtifactReplayer": false}
	entries := map[string]bool{"exploreWith": false, "resumeWith": false, "runCampaignShardWith": false, "createCampaignPlanWith": false, "replayWith": false, "minimizeWith": false}
	publicTypes := []string{}
	parsed := 0
	for _, path := range files {
		if strings.HasSuffix(path, "_test.go") { continue }
		file, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
		if err != nil { panic(err) }
		parsed++
		for _, d := range file.Decls {
			if f, ok := d.(*ast.FuncDecl); ok {
				if _, exists := entries[f.Name.Name]; exists {
					for _, p := range f.Type.Params.List { if t, ok := p.Type.(*ast.Ident); ok && t.Name == "executionDependencies" { entries[f.Name.Name] = true } }
				}
			}
			g, ok := d.(*ast.GenDecl)
			if !ok { continue }
			for _, s := range g.Specs {
				if t, ok := s.(*ast.TypeSpec); ok {
					if t.Name.Name == "Executor" || t.Name.Name == "ReplayExecutor" { panic("removed interface remains") }
					if _, exists := seams[t.Name.Name]; exists { _, seams[t.Name.Name] = t.Type.(*ast.InterfaceType) }
					if t.Name.IsExported() { publicTypes = append(publicTypes, t.Name.Name) }
					if st, ok := t.Type.(*ast.StructType); ok && t.Name.IsExported() { for _, f := range st.Fields.List { for _, n := range f.Names { if n.Name == "Executor" { panic("removed public field remains") } } } }
				}
				if v, ok := s.(*ast.ValueSpec); ok && g.Tok == token.VAR {
					for _, n := range v.Names { variables = append(variables, path+":"+n.Name) }
					ast.Inspect(v, func(n ast.Node) bool { if i, ok := n.(*ast.Ident); ok && (i.Name == "executionRunner" || i.Name == "executionDependencies") { panic("global execution hook remains") }; if _, ok := n.(*ast.FuncType); ok { panic("global function hook requires explicit reconciliation") }; return true })
				}
			}
		}
	}
	for n, ok := range seams { if !ok { panic("missing public seam "+n) } }
	for n, ok := range entries { if !ok { panic("missing private dependencies "+n) } }
	if parsed == 0 { panic("zero source coverage") }
	value := map[string]any{"parsed_production_files": parsed, "public_seams": seams, "private_entrypoints": entries, "package_variables": variables, "public_types": publicTypes, "exported_executor_interfaces": 0, "exported_executor_fields": 0, "global_execution_hooks": 0}
	if err := json.NewEncoder(os.Stdout).Encode(value); err != nil { fmt.Fprintln(os.Stderr, err); os.Exit(1) }
}
