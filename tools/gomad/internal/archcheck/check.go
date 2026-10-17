package archcheck

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
)

func Check(root, goCommand string, platform Platform, config Config) ([]Finding, error) {
	inventory, findings, err := discover(root, goCommand, platform, false)
	if err != nil || slices.ContainsFunc(findings, func(f Finding) bool { return f.Category == "package-error" }) {
		return findings, err
	}
	if config.Module == "" {
		return findings, fmt.Errorf("module import path is required")
	}
	if config.PureFiles != nil && !slices.ContainsFunc(inventory.Sources, config.PureFiles) {
		return findings, fmt.Errorf("pure file selector matches no source")
	}
	var packages []Package
	purePackages := map[string]bool{}
	files, names := map[string]map[string]*ast.File{}, map[string]map[string]bool{}
	for _, pkg := range inventory.Packages {
		packages = append(packages, Package{pkg.ImportPath, pkg.Imports})
		owner, count := "", 0
		for name, prefixes := range config.Owners {
			if slices.ContainsFunc(prefixes, func(prefix string) bool { return Within(pkg.ImportPath, prefix) }) {
				owner, count = name, count+1
			}
		}
		if config.Owners != nil && count != 1 && len(pkg.GoFiles)+len(pkg.CgoFiles) != 0 {
			findings = append(findings, Finding{"ownerless", pkg.ImportPath, fmt.Sprintf("matched %d owners", count)})
		}
		for _, imported := range pkg.Imports {
			if !Within(imported, config.Module) {
				continue
			}
			for target, prefixes := range config.Owners {
				if target != owner && slices.ContainsFunc(prefixes, func(prefix string) bool { return Within(imported, prefix) }) && !slices.Contains(config.OwnerEdges[owner], target) {
					findings = append(findings, Finding{"owner-edge", pkg.ImportPath, imported})
				}
			}
			for _, edge := range config.ModuleEdges {
				if Within(pkg.ImportPath, edge.From) && Within(imported, edge.To) {
					findings = append(findings, Finding{"module-edge", pkg.ImportPath, imported})
				}
			}
		}
		files[pkg.ImportPath], names[pkg.ImportPath] = map[string]*ast.File{}, map[string]bool{}
		for _, name := range pkg.IgnoredGoFiles {
			path := filepath.Join(pkg.Dir, name)
			relative, err := filepath.Rel(inventory.root, path)
			if err != nil {
				return findings, err
			}
			relative = filepath.ToSlash(relative)
			if config.PureFiles == nil || !config.PureFiles(relative) {
				continue
			}
			file, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
			if err != nil {
				return findings, err
			}
			for _, spec := range file.Imports {
				imported, err := strconv.Unquote(spec.Path.Value)
				if err != nil {
					return findings, err
				}
				if imported == "C" {
					findings = append(findings, Finding{"pure-syntax", relative, "cgo is forbidden in pure files"})
				}
			}
		}
		sources := append(slices.Clone(pkg.GoFiles), pkg.CgoFiles...)
		pure := len(sources) != 0
		for _, name := range sources {
			path := filepath.Join(pkg.Dir, name)
			relative, err := filepath.Rel(inventory.root, path)
			if err != nil {
				return findings, err
			}
			relative = filepath.ToSlash(relative)
			selected := config.PureFiles != nil && config.PureFiles(relative)
			pure = pure && selected
			if !selected {
				continue
			}
			file, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
			if err != nil {
				return findings, err
			}
			files[pkg.ImportPath][relative] = file
			for name, object := range file.Scope.Objects {
				if object.Kind == ast.Fun || object.Kind == ast.Typ {
					names[pkg.ImportPath][name] = true
				}
			}
		}
		purePackages[pkg.ImportPath] = pure
	}
	for _, detail := range config.CheckImports(packages) {
		findings = append(findings, Finding{"module-edge", config.Module, detail})
	}
	for pkg, sources := range files {
		for relative, file := range sources {
			findings = append(findings, syntax(relative, file, config.PureImports, purePackages, names[pkg])...)
		}
	}
	return ordered(findings), nil
}

func ordered(findings []Finding) []Finding {
	slices.SortFunc(findings, func(a, b Finding) int {
		return strings.Compare(a.Category+"\x00"+a.Path+"\x00"+a.Detail, b.Category+"\x00"+b.Path+"\x00"+b.Detail)
	})
	return slices.Compact(findings)
}

func syntax(path string, file *ast.File, allowed []string, purePackages, pureNames map[string]bool) []Finding {
	var findings []Finding
	add := func(category, detail string) { findings = append(findings, Finding{category, path, detail}) }
	imports := map[string]string{}
	for _, spec := range file.Imports {
		imported, err := strconv.Unquote(spec.Path.Value)
		if err != nil {
			add("pure-import", err.Error())
			continue
		}
		name := filepath.Base(imported)
		if spec.Name != nil {
			name = spec.Name.Name
		}
		imports[name] = imported
		if imported == "C" {
			add("pure-syntax", "cgo is forbidden in pure files")
		} else if !slices.Contains(allowed, imported) && !purePackages[imported] {
			add("pure-import", imported)
		}
		if imported == "syscall" && (name == "." || name == "_") {
			add("pure-import", "syscall requires qualified Errno or E constants")
		}
	}
	isTime := func(node ast.Node) bool {
		found, seen := false, map[ast.Node]bool{}
		var inspect func(ast.Node) bool
		inspect = func(node ast.Node) bool {
			if id, ok := node.(*ast.Ident); ok {
				found = found || (id.Obj == nil && imports[id.Name] == "time")
				if id.Obj != nil {
					if declaration, ok := id.Obj.Decl.(ast.Node); ok && !seen[declaration] {
						seen[declaration] = true
						ast.Inspect(declaration, inspect)
					}
				}
			}
			return !found
		}
		ast.Inspect(node, inspect)
		return found
	}
	ast.Inspect(file, func(node ast.Node) bool {
		switch node := node.(type) {
		case *ast.GoStmt:
			add("pure-syntax", "go statement")
		case *ast.SelectStmt:
			for _, clause := range node.Body.List {
				comm := clause.(*ast.CommClause).Comm
				if comm != nil && isTime(comm) {
					add("pure-syntax", "select on a time channel")
				}
			}
		case *ast.SelectorExpr:
			if id, ok := node.X.(*ast.Ident); ok && imports[id.Name] == "syscall" && node.Sel.Name != "Errno" && (!strings.HasPrefix(node.Sel.Name, "E") || strings.ToUpper(node.Sel.Name) != node.Sel.Name) {
				add("pure-import", "syscall."+node.Sel.Name)
			}
		case *ast.FuncDecl:
			if node.Name.Name != "init" || node.Recv != nil {
				break
			}
			ast.Inspect(node.Body, func(child ast.Node) bool {
				call, ok := child.(*ast.CallExpr)
				if !ok {
					return true
				}
				if id, ok := call.Fun.(*ast.Ident); ok && (pureNames[id.Name] || slices.Contains([]string{"append", "cap", "clear", "complex", "copy", "delete", "imag", "len", "make", "max", "min", "new", "panic", "real", "recover", "bool", "byte", "complex64", "complex128", "float32", "float64", "int", "int8", "int16", "int32", "int64", "rune", "string", "uint", "uint8", "uint16", "uint32", "uint64", "uintptr"}, id.Name)) {
					return true
				}
				if _, ok := call.Fun.(*ast.FuncLit); ok {
					return true
				}
				if sel, ok := call.Fun.(*ast.SelectorExpr); ok {
					if id, ok := sel.X.(*ast.Ident); ok && (slices.Contains(allowed, imports[id.Name]) || purePackages[imports[id.Name]]) {
						return true
					}
				}
				add("pure-syntax", "init call outside the pure import set")
				return true
			})
		}
		return true
	})
	return findings
}
