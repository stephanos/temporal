package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

// resolveCrossFileEnumReferences rewrites references to enum values declared in another file of the
// same generated Go package. protogen trims the enum type prefix from a value only inside the file
// that declares the enum, so a getter in a second file still names the untrimmed constant, for
// example `RunEventKind_RUN_EVENT_KIND_UNSPECIFIED` for `RUN_EVENT_KIND_UNSPECIFIED`.
func resolveCrossFileEnumReferences(root string) error {
	packages := map[string][]string{}
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !entry.IsDir() && strings.HasSuffix(path, ".pb.go") {
			packages[filepath.Dir(path)] = append(packages[filepath.Dir(path)], path)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("error walking generated files in %s: %w", root, err)
	}
	for _, files := range packages {
		if err := resolvePackageEnumReferences(files); err != nil {
			return err
		}
	}
	return nil
}

func resolvePackageEnumReferences(paths []string) error {
	fileSet := token.NewFileSet()
	files := make([]*ast.File, len(paths))
	for index, path := range paths {
		file, err := parser.ParseFile(fileSet, path, nil, parser.SkipObjectResolution)
		if err != nil {
			return fmt.Errorf("error parsing generated file %s: %w", path, err)
		}
		files[index] = file
	}
	declared, trimmed := packageEnumDeclarations(files)
	for index, path := range paths {
		var references []*ast.Ident
		ast.Inspect(files[index], func(node ast.Node) bool {
			if ident, ok := node.(*ast.Ident); ok && !declared[ident.Name] && trimmed[ident.Name] != "" {
				references = append(references, ident)
			}
			return true
		})
		if err := rewriteEnumReferences(fileSet, path, references, trimmed); err != nil {
			return err
		}
	}
	return nil
}

// packageEnumDeclarations returns every top-level name the files declare, and maps the untrimmed
// spelling `Type_VALUE` of each typed constant to the name it was declared with.
func packageEnumDeclarations(files []*ast.File) (map[string]bool, map[string]string) {
	declared := map[string]bool{}
	trimmed := map[string]string{}
	for _, file := range files {
		for _, declaration := range file.Decls {
			general, ok := declaration.(*ast.GenDecl)
			if !ok {
				continue
			}
			for _, spec := range general.Specs {
				recordDeclaration(general.Tok, spec, declared, trimmed)
			}
		}
	}
	return declared, trimmed
}

func recordDeclaration(tok token.Token, spec ast.Spec, declared map[string]bool, trimmed map[string]string) {
	if typeSpec, ok := spec.(*ast.TypeSpec); ok {
		declared[typeSpec.Name.Name] = true
		return
	}
	value, ok := spec.(*ast.ValueSpec)
	if !ok {
		return
	}
	typeName, typed := value.Type.(*ast.Ident)
	for _, name := range value.Names {
		declared[name.Name] = true
		if typed && tok == token.CONST {
			trimmed[typeName.Name+"_"+name.Name] = name.Name
		}
	}
}

func rewriteEnumReferences(fileSet *token.FileSet, path string, references []*ast.Ident, trimmed map[string]string) error {
	if len(references) == 0 {
		return nil
	}
	source, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("error reading generated file %s: %w", path, err)
	}
	// Rewrite from the end so earlier offsets stay valid.
	slices.Reverse(references)
	for _, ident := range references {
		offset := fileSet.Position(ident.Pos()).Offset
		source = slices.Concat(source[:offset], []byte(trimmed[ident.Name]), source[offset+len(ident.Name):])
	}
	if err := os.WriteFile(path, source, 0644); err != nil {
		return fmt.Errorf("error writing generated file %s: %w", path, err)
	}
	return nil
}
