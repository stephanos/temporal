package temporal

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

type coverageType struct {
	pkg, name string
}

type coverageFunction struct {
	id      string
	pkg     *coveragePackage
	decl    *ast.FuncDecl
	imports map[string]string
}

type coveragePackage struct {
	path   string
	funcs  map[string]*coverageFunction
	fields map[string]map[string]coverageType
	embeds map[string][]coverageType
}

type coverageIndex struct {
	root, module string
	fset         *token.FileSet
	packages     map[string]*coveragePackage
	unresolved   map[string]bool
}

// ReverseCoverage checks exact catalog targets and reports statically reachable
// validation seams. Unresolved calls are diagnostics, not evidence of coverage.
func ReverseCoverage(repoRoot string, covers []string) ([]string, []string, error) {
	repoRoot, err := filepath.Abs(repoRoot)
	if err != nil {
		return nil, nil, err
	}
	repoRoot, err = filepath.EvalSymlinks(repoRoot)
	if err != nil {
		return nil, nil, err
	}
	body, err := os.ReadFile(filepath.Join(repoRoot, "go.mod"))
	if err != nil {
		return nil, nil, err
	}
	var module string
	for _, line := range strings.Split(string(body), "\n") {
		fields := strings.Fields(line)
		if len(fields) >= 2 && fields[0] == "module" {
			module = strings.Trim(fields[1], `"`)
			break
		}
	}
	if module == "" {
		return nil, nil, fmt.Errorf("repository go.mod has no module identity")
	}
	index := &coverageIndex{root: repoRoot, module: module, fset: token.NewFileSet(), packages: make(map[string]*coveragePackage), unresolved: make(map[string]bool)}
	covered := make(map[string]bool)
	for _, target := range covers {
		pkgPath, name, ok := coverageIdentity(target)
		if !ok {
			return nil, nil, fmt.Errorf("invalid covers target %q", target)
		}
		pkg, err := index.load(pkgPath)
		if err != nil {
			return nil, nil, fmt.Errorf("covers target %s: %w", target, err)
		}
		if pkg == nil || pkg.funcs[name] == nil {
			return nil, nil, fmt.Errorf("covers target %s has no non-test function definition", target)
		}
		covered[target] = true
	}
	roots := []string{
		module + "/chasm/lib/activity.frontendHandler.StartActivityExecution",
		module + "/chasm/lib/activity.frontendHandler.DescribeActivityExecution",
		module + "/chasm/lib/activity.frontendHandler.PollActivityExecution",
		module + "/chasm/lib/activity.frontendHandler.RequestCancelActivityExecution",
		module + "/service/frontend.WorkflowHandler.PollActivityTaskQueue",
		module + "/service/frontend.WorkflowHandler.RespondActivityTaskCompleted",
		module + "/service/frontend.WorkflowHandler.RespondActivityTaskFailed",
		module + "/service/frontend.WorkflowHandler.RespondActivityTaskCanceled",
	}
	uncovered, unresolved, visited := make(map[string]bool), index.unresolved, make(map[string]bool)
	queue := append([]string(nil), roots...)
	queue = append(queue, covers...)
	for len(queue) > 0 {
		id := queue[0]
		queue = queue[1:]
		if visited[id] {
			continue
		}
		visited[id] = true
		pkgPath, name, _ := coverageIdentity(id)
		pkg, err := index.load(pkgPath)
		if err != nil {
			unresolved[id+": "+err.Error()] = true
			continue
		}
		if pkg == nil || pkg.funcs[name] == nil {
			unresolved[id+": no static function definition"] = true
			continue
		}
		fn := pkg.funcs[name]
		if fn.decl.Body == nil {
			unresolved[id+": function has no Go body"] = true
			continue
		}
		if coverageValidationName(fn.decl.Name.Name) && !covered[id] {
			uncovered[id] = true
		}
		variables := make(map[string]coverageType)
		for _, list := range []*ast.FieldList{fn.decl.Recv, fn.decl.Type.Params} {
			if list == nil {
				continue
			}
			for _, field := range list.List {
				for _, name := range field.Names {
					variables[name.Name] = coverageExprType(field.Type, pkgPath, fn.imports)
				}
			}
		}
		ast.Inspect(fn.decl.Body, func(node ast.Node) bool {
			call, ok := node.(*ast.CallExpr)
			if !ok {
				return true
			}
			callName := coverageCallName(call.Fun)
			if (callName == "NewInvalidArgument" || callName == "NewInvalidArgumentf") && !covered[id] {
				uncovered[fmt.Sprintf("%s#inline:%d", id, index.fset.Position(call.Pos()).Line)] = true
			}
			target, resolved := index.resolveCall(fn, variables, call.Fun)
			if resolved {
				queue = append(queue, target)
			} else if coverageValidationName(callName) {
				var rendered bytes.Buffer
				if err := printer.Fprint(&rendered, index.fset, call.Fun); err != nil {
					unresolved[id+": cannot render call: "+err.Error()] = true
				} else {
					unresolved[fmt.Sprintf("%s:%d -> %s", id, index.fset.Position(call.Pos()).Line, rendered.String())] = true
				}
			}
			return true
		})
	}
	return coverageSorted(uncovered), coverageSorted(unresolved), nil
}

func coverageIdentity(id string) (string, string, bool) {
	slash := strings.LastIndex(id, "/")
	dot := strings.Index(id[slash+1:], ".")
	if dot < 1 {
		return "", "", false
	}
	dot += slash + 1
	return id[:dot], id[dot+1:], dot+1 < len(id)
}

func (index *coverageIndex) load(pkgPath string) (*coveragePackage, error) {
	if pkg, exists := index.packages[pkgPath]; exists {
		return pkg, nil
	}
	if pkgPath != index.module && !strings.HasPrefix(pkgPath, index.module+"/") {
		return nil, nil
	}
	rel := strings.TrimPrefix(strings.TrimPrefix(pkgPath, index.module), "/")
	if !filepath.IsLocal(rel) && rel != "" {
		return nil, fmt.Errorf("package path escapes repository: %s", pkgPath)
	}
	dir := filepath.Join(index.root, filepath.FromSlash(rel))
	physicalDir, err := filepath.EvalSymlinks(dir)
	if err != nil {
		return nil, err
	}
	if physicalDir != dir {
		err := fmt.Errorf("package directory identity mismatch for %s: %s resolves to %s", pkgPath, dir, physicalDir)
		index.unresolved[err.Error()] = true
		return nil, err
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	pkg := &coveragePackage{path: pkgPath, funcs: make(map[string]*coverageFunction), fields: make(map[string]map[string]coverageType), embeds: make(map[string][]coverageType)}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") || strings.HasSuffix(entry.Name(), "_test.go") {
			continue
		}
		file, err := parser.ParseFile(index.fset, filepath.Join(dir, entry.Name()), nil, 0)
		if err != nil {
			return nil, err
		}
		imports := make(map[string]string)
		for _, imp := range file.Imports {
			importPath, err := strconv.Unquote(imp.Path.Value)
			if err != nil {
				return nil, err
			}
			name := path.Base(importPath)
			if imp.Name != nil {
				name = imp.Name.Name
			}
			imports[name] = importPath
		}
		for _, declaration := range file.Decls {
			switch declaration := declaration.(type) {
			case *ast.FuncDecl:
				name := declaration.Name.Name
				if declaration.Recv != nil {
					receiver := coverageExprType(declaration.Recv.List[0].Type, pkgPath, imports)
					name = receiver.name + "." + name
				}
				if _, exists := pkg.funcs[name]; exists {
					return nil, fmt.Errorf("ambiguous non-test definition %s.%s", pkgPath, name)
				}
				pkg.funcs[name] = &coverageFunction{id: pkgPath + "." + name, pkg: pkg, decl: declaration, imports: imports}
			case *ast.GenDecl:
				for _, spec := range declaration.Specs {
					typeSpec, ok := spec.(*ast.TypeSpec)
					if !ok {
						continue
					}
					structure, ok := typeSpec.Type.(*ast.StructType)
					if !ok {
						continue
					}
					fields := make(map[string]coverageType)
					for _, field := range structure.Fields.List {
						if len(field.Names) == 0 {
							embedded := coverageExprType(field.Type, pkgPath, imports)
							pkg.embeds[typeSpec.Name.Name] = append(pkg.embeds[typeSpec.Name.Name], embedded)
							fields[embedded.name] = embedded
						}
						for _, name := range field.Names {
							fields[name.Name] = coverageExprType(field.Type, pkgPath, imports)
						}
					}
					pkg.fields[typeSpec.Name.Name] = fields
				}
			}
		}
	}
	index.packages[pkgPath] = pkg
	return pkg, nil
}

func coverageExprType(expr ast.Expr, pkg string, imports map[string]string) coverageType {
	switch expr := expr.(type) {
	case *ast.StarExpr:
		return coverageExprType(expr.X, pkg, imports)
	case *ast.Ident:
		return coverageType{pkg, expr.Name}
	case *ast.SelectorExpr:
		if qualifier, ok := expr.X.(*ast.Ident); ok {
			return coverageType{imports[qualifier.Name], expr.Sel.Name}
		}
	case *ast.IndexExpr:
		return coverageExprType(expr.X, pkg, imports)
	case *ast.IndexListExpr:
		return coverageExprType(expr.X, pkg, imports)
	}
	return coverageType{}
}

func (index *coverageIndex) valueType(fn *coverageFunction, variables map[string]coverageType, expr ast.Expr) coverageType {
	switch expr := expr.(type) {
	case *ast.Ident:
		return variables[expr.Name]
	case *ast.SelectorExpr:
		owner := index.valueType(fn, variables, expr.X)
		pkg, err := index.load(owner.pkg)
		if err == nil && pkg != nil {
			return pkg.fields[owner.name][expr.Sel.Name]
		}
	}
	return coverageType{}
}

func (index *coverageIndex) resolveCall(fn *coverageFunction, variables map[string]coverageType, expr ast.Expr) (string, bool) {
	var pkgPath, name string
	switch expr := expr.(type) {
	case *ast.Ident:
		pkgPath, name = fn.pkg.path, expr.Name
	case *ast.SelectorExpr:
		if qualifier, ok := expr.X.(*ast.Ident); ok && fn.imports[qualifier.Name] != "" {
			pkgPath, name = fn.imports[qualifier.Name], expr.Sel.Name
		} else {
			owner := index.valueType(fn, variables, expr.X)
			return index.resolveMethod(owner, expr.Sel.Name, make(map[coverageType]bool))
		}
	default:
		return "", false
	}
	pkg, err := index.load(pkgPath)
	if err != nil || pkg == nil || pkg.funcs[name] == nil {
		return "", false
	}
	return pkg.funcs[name].id, true
}

func (index *coverageIndex) resolveMethod(owner coverageType, method string, seen map[coverageType]bool) (string, bool) {
	if seen[owner] {
		return "", false
	}
	seen[owner] = true
	pkg, err := index.load(owner.pkg)
	if err != nil || pkg == nil {
		return "", false
	}
	if fn := pkg.funcs[owner.name+"."+method]; fn != nil {
		return fn.id, true
	}
	var target string
	for _, embedded := range pkg.embeds[owner.name] {
		if id, ok := index.resolveMethod(embedded, method, seen); ok {
			if target != "" && target != id {
				return "", false
			}
			target = id
		}
	}
	return target, target != ""
}

func coverageCallName(expr ast.Expr) string {
	switch expr := expr.(type) {
	case *ast.Ident:
		return expr.Name
	case *ast.SelectorExpr:
		return expr.Sel.Name
	}
	return ""
}

func coverageValidationName(name string) bool {
	name = strings.ToLower(name)
	return strings.Contains(name, "validat") || strings.HasPrefix(name, "check") || strings.HasPrefix(name, "normaliz")
}

func coverageSorted(values map[string]bool) []string {
	result := make([]string, 0, len(values))
	for value := range values {
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}
