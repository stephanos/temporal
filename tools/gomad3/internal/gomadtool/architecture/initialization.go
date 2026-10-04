package architecture

import (
	"crypto/sha256"
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

func (a *effectAnalysis) initializers() {
	seen := map[string]bool{}
	var visit func(string, token.Pos)
	visit = func(path string, pos token.Pos) {
		if seen[path] {
			return
		}
		seen[path] = true
		metadata, found := a.program.Metadata[path]
		if !found {
			a.report("unresolved-effect", pos, "initializer has no package metadata: "+path)
			return
		}
		imports := append([]string{}, metadata.Imports...)
		sort.Strings(imports)
		for _, imported := range imports {
			if imported == "C" && metadata.Standard && len(metadata.CgoFiles) != 0 {
				continue
			}
			visit(imported, pos)
		}
		if metadata.Standard {
			a.checkStartupSource(metadata, pos)
			return
		}
		pkg := a.program.Packages[path]
		if pkg == nil {
			a.report("unresolved-effect", pos, "initializer has no typed source: "+path)
			return
		}
		for _, file := range pkg.Files {
			a.root = path + ":initializer"
			for _, declaration := range file.Decls {
				switch declaration := declaration.(type) {
				case *ast.GenDecl:
					for _, specification := range declaration.Specs {
						if specification, ok := specification.(*ast.ValueSpec); ok {
							for _, expression := range specification.Values {
								a.analyze(expression.Pos(), func() { a.eval(pkg, expression, environment{}) })
							}
						}
					}
				case *ast.FuncDecl:
					if declaration.Name.Name == "init" {
						object, ok := pkg.Info.Defs[declaration.Name].(*types.Func)
						if !ok {
							a.report("unresolved-effect", declaration.Pos(), "initializer has no typed function")
							continue
						}
						fn := &function{object: object, declaration: declaration, pkg: pkg}
						a.analyze(declaration.Pos(), func() {
							a.call(&boundFunction{fn: fn, pkg: pkg}, nil, declaration.Pos())
						})
					}
				}
			}
		}
	}
	var paths []string
	for path := range a.program.Packages {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	for _, path := range paths {
		pkg := a.program.Packages[path]
		for _, file := range pkg.Files {
			if !a.program.IsPureFile(a.program.Files.Position(file.Pos()).Filename) {
				continue
			}
			a.root = path + ":initializer"
			for _, spec := range file.Imports {
				imported, err := strconv.Unquote(spec.Path.Value)
				if err != nil {
					a.report("unresolved-effect", spec.Pos(), "invalid initializer import")
					continue
				}
				if mapped := pkg.Metadata.ImportMap[imported]; mapped != "" {
					imported = mapped
				}
				visit(imported, spec.Pos())
			}
			// Importing a package runs every initializer in it, including siblings
			// of a selectively pure file. Its callable siblings are not pure roots.
			visit(path, file.Pos())
		}
	}
}

// Standard-library startup belongs to runtime.main before the model starts.
// This boundary checks exact Go 1.27.1 source identities, not function purity:
// any typed call made by an application/dependency initializer is still traced.
func (a *effectAnalysis) checkStartupSource(metadata Package, pos token.Pos) {
	entries, err := os.ReadDir(metadata.Dir)
	if err != nil {
		a.report("unresolved-effect", pos, "cannot inspect standard startup source: "+metadata.ImportPath)
		return
	}
	digest := sha256.New()
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(metadata.Dir, entry.Name()))
		if err != nil {
			a.report("unresolved-effect", pos, "cannot read standard startup source: "+metadata.ImportPath)
			return
		}
		fmt.Fprintf(digest, "%x  %s\n", sha256.Sum256(data), entry.Name())
	}
	if fmt.Sprintf("%x", digest.Sum(nil)) != startupSources[metadata.ImportPath] {
		a.report("unresolved-effect", pos, "pinned standard startup source changed: "+metadata.ImportPath)
	}
}
