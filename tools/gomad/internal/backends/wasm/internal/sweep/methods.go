package sweep

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"
)

// SuiteMethod records a declaration and every isolated unit that covers it.
type SuiteMethod struct {
	Suite string   `json:"suite"`
	Name  string   `json:"name"`
	File  string   `json:"file"`
	Units []string `json:"units"`
}

// SuiteRegistration records a suite's entry point, including nested subtests.
type SuiteRegistration struct {
	Wrapper string `json:"wrapper"`
	Suite   string `json:"suite"`
	Path    string `json:"path"`
	File    string `json:"file"`
}

type suiteInventory struct {
	Tests         []string
	Methods       []SuiteMethod
	Registrations []SuiteRegistration
	files         map[string]string
	unitFiles     map[string][]string
}

type suiteSource struct {
	files     map[string]*ast.File
	types     map[string]*ast.TypeSpec
	typeFiles map[string]*ast.File
	methods   map[string]map[string]*suiteFunction
}

type suiteFunction struct {
	decl *ast.FuncDecl
	file string
	ast  *ast.File
	key  string
}

type promotedMethod struct {
	function  *suiteFunction
	depth     int
	ambiguous bool
}

func testRunPattern(name string) string {
	parts := strings.Split(name, "/")
	for i, part := range parts {
		if part != ".*" {
			parts[i] = "^" + regexp.QuoteMeta(part) + "$"
		}
	}
	return strings.Join(parts, "/")
}

func selectedTest(name, observed string) bool {
	parts, actual := strings.Split(name, "/"), strings.Split(observed, "/")
	if len(parts) != len(actual) {
		return false
	}
	for i, part := range parts {
		if part != ".*" && part != actual[i] {
			return false
		}
	}
	return true
}

func parseSuiteSource(dir string, files []string) (*suiteSource, error) {
	source := &suiteSource{files: map[string]*ast.File{}, types: map[string]*ast.TypeSpec{}, typeFiles: map[string]*ast.File{}, methods: map[string]map[string]*suiteFunction{}}
	for _, name := range files {
		file, err := parser.ParseFile(token.NewFileSet(), filepath.Join(dir, name), nil, parser.SkipObjectResolution)
		if err != nil {
			return nil, fmt.Errorf("parse %s: %w", filepath.Join(dir, name), err)
		}
		source.files[name] = file
		for _, decl := range file.Decls {
			switch decl := decl.(type) {
			case *ast.GenDecl:
				for _, spec := range decl.Specs {
					if typ, ok := spec.(*ast.TypeSpec); ok {
						key := file.Name.Name + "." + typ.Name.Name
						source.types[key] = typ
						source.typeFiles[key] = file
					}
				}
			case *ast.FuncDecl:
				if decl.Recv == nil {
					continue
				}
				receiver, ok := baseType(decl.Recv.List[0].Type).(*ast.Ident)
				if !ok {
					return nil, fmt.Errorf("unresolved method receiver in %s", name)
				}
				key := file.Name.Name + "." + receiver.Name
				if source.methods[key] == nil {
					source.methods[key] = map[string]*suiteFunction{}
				}
				source.methods[key][decl.Name.Name] = &suiteFunction{decl: decl, file: name, ast: file, key: key}
			}
		}
	}
	return source, nil
}

func baseType(expr ast.Expr) ast.Expr {
	for {
		switch typ := expr.(type) {
		case *ast.StarExpr:
			expr = typ.X
		case *ast.IndexExpr:
			expr = typ.X
		case *ast.IndexListExpr:
			expr = typ.X
		case *ast.ParenExpr:
			expr = typ.X
		default:
			return expr
		}
	}
}

func importPath(file *ast.File, alias string) string {
	for _, spec := range file.Imports {
		path, err := strconv.Unquote(spec.Path.Value)
		if err != nil {
			continue
		}
		name := filepath.Base(path)
		if spec.Name != nil {
			name = spec.Name.Name
		}
		if name == alias {
			return path
		}
	}
	return ""
}

type suiteBinding struct {
	typ    ast.Expr
	values []ast.Expr
}

func suiteBindings(fn *ast.FuncDecl) map[*ast.Ident]*suiteBinding {
	bindings := map[*ast.Ident]*suiteBinding{}
	var walk func(ast.Node, []map[string]*suiteBinding)
	lookup := func(scopes []map[string]*suiteBinding, name string) *suiteBinding {
		for i := len(scopes) - 1; i >= 0; i-- {
			if binding, ok := scopes[i][name]; ok {
				return binding
			}
		}
		return nil
	}
	parameters := func(fields *ast.FieldList, scope map[string]*suiteBinding) {
		if fields != nil {
			for _, field := range fields.List {
				for _, name := range field.Names {
					scope[name.Name] = &suiteBinding{typ: field.Type}
				}
			}
		}
	}
	walk = func(node ast.Node, scopes []map[string]*suiteBinding) {
		if node == nil {
			return
		}
		switch node := node.(type) {
		case *ast.BlockStmt, *ast.IfStmt, *ast.ForStmt, *ast.SwitchStmt, *ast.TypeSwitchStmt, *ast.CaseClause, *ast.CommClause:
			scopes = append(slices.Clone(scopes), map[string]*suiteBinding{})
		case *ast.RangeStmt:
			walk(node.X, scopes)
			scope := map[string]*suiteBinding{}
			scopes = append(slices.Clone(scopes), scope)
			for _, expr := range []ast.Expr{node.Key, node.Value} {
				if name, ok := expr.(*ast.Ident); ok {
					binding := lookup(scopes, name.Name)
					if node.Tok == token.DEFINE {
						binding = &suiteBinding{}
						scope[name.Name] = binding
					}
					if binding != nil {
						binding.values = append(binding.values, nil)
						bindings[name] = binding
					}
				}
			}
			walk(node.Body, scopes)
			return
		case *ast.FuncLit:
			scope := map[string]*suiteBinding{}
			parameters(node.Type.Params, scope)
			scopes = append(slices.Clone(scopes), scope)
			for _, statement := range node.Body.List {
				walk(statement, scopes)
			}
			return
		case *ast.AssignStmt:
			for _, value := range node.Rhs {
				walk(value, scopes)
			}
			for i, lhs := range node.Lhs {
				name, ok := lhs.(*ast.Ident)
				if !ok {
					walk(lhs, scopes)
					continue
				}
				scope := scopes[len(scopes)-1]
				binding := lookup(scopes, name.Name)
				if _, local := scope[name.Name]; node.Tok == token.DEFINE && !local {
					binding = &suiteBinding{}
					scope[name.Name] = binding
				}
				if binding != nil {
					var value ast.Expr
					if i < len(node.Rhs) {
						value = node.Rhs[i]
					}
					binding.values = append(binding.values, value)
					bindings[name] = binding
				}
			}
			return
		case *ast.ValueSpec:
			for _, value := range node.Values {
				walk(value, scopes)
			}
			for i, name := range node.Names {
				binding := &suiteBinding{typ: node.Type}
				if i < len(node.Values) {
					binding.values = append(binding.values, node.Values[i])
				}
				scopes[len(scopes)-1][name.Name] = binding
				bindings[name] = binding
			}
			return
		case *ast.Ident:
			if binding := lookup(scopes, node.Name); binding != nil {
				bindings[node] = binding
			}
			return
		}
		ast.Inspect(node, func(child ast.Node) bool {
			if child == node {
				return true
			}
			if child != nil {
				walk(child, scopes)
			}
			return false
		})
	}
	scope := map[string]*suiteBinding{}
	parameters(fn.Recv, scope)
	parameters(fn.Type.Params, scope)
	parameters(fn.Type.Results, scope)
	if fn.Body != nil {
		for _, statement := range fn.Body.List {
			walk(statement, []map[string]*suiteBinding{scope})
		}
	}
	return bindings
}

func suiteArgument(expr ast.Expr, bindings map[*ast.Ident]*suiteBinding, seen map[*suiteBinding]bool) string {
	switch expr := expr.(type) {
	case *ast.UnaryExpr:
		return suiteArgument(expr.X, bindings, seen)
	case *ast.CompositeLit:
		if name, ok := baseType(expr.Type).(*ast.Ident); ok {
			return name.Name
		}
	case *ast.CallExpr:
		if fn, ok := expr.Fun.(*ast.Ident); ok && fn.Name == "new" && len(expr.Args) == 1 {
			if name, ok := baseType(expr.Args[0]).(*ast.Ident); ok {
				return name.Name
			}
		}
	case *ast.Ident:
		binding, ok := bindings[expr]
		if !ok || seen[binding] {
			return ""
		}
		seen[binding] = true
		defer delete(seen, binding)
		name := ""
		if typ, ok := baseType(binding.typ).(*ast.Ident); ok {
			name = typ.Name
		}
		for i, value := range binding.values {
			resolved := suiteArgument(value, bindings, seen)
			if i == 0 && binding.typ == nil {
				name = resolved
			} else if resolved != name {
				return ""
			}
		}
		return name
	}
	return ""
}

type suiteCall struct {
	suite string
	path  string
}

func suiteCalls(fn *suiteFunction) ([]suiteCall, error) {
	var calls []suiteCall
	var failure error
	if fn.decl.Body == nil {
		return nil, nil
	}
	bindings := suiteBindings(fn.decl)
	var walk func(ast.Node, string)
	walk = func(node ast.Node, path string) {
		ast.Inspect(node, func(node ast.Node) bool {
			call, ok := node.(*ast.CallExpr)
			if !ok || failure != nil {
				return failure == nil
			}
			selector, ok := baseType(call.Fun).(*ast.SelectorExpr)
			if !ok {
				return true
			}
			if qualifier, ok := selector.X.(*ast.Ident); ok && bindings[qualifier] == nil {
				imported := importPath(fn.ast, qualifier.Name)
				if (imported == "github.com/stretchr/testify/suite" && selector.Sel.Name == "Run") || (imported == "go.temporal.io/server/common/testing/parallelsuite" && (selector.Sel.Name == "Run" || selector.Sel.Name == "RunLegacySequential")) {
					if len(call.Args) < 2 {
						failure = fmt.Errorf("unresolved suite registration in %s: %s", fn.file, fn.decl.Name.Name)
						return false
					}
					suite := suiteArgument(call.Args[1], bindings, map[*suiteBinding]bool{})
					if suite == "" {
						failure = fmt.Errorf("unresolved suite registration in %s: %s", fn.file, fn.decl.Name.Name)
						return false
					}
					if slices.ContainsFunc(calls, func(previous suiteCall) bool { return previous.path == path }) {
						failure = fmt.Errorf("overlapping suite registrations in %s: %s", fn.file, fn.decl.Name.Name)
						return false
					}
					calls = append(calls, suiteCall{suite: fn.ast.Name.Name + "." + suite, path: path})
					return false
				}
			}
			if selector.Sel.Name == "Run" && len(call.Args) == 2 {
				if callback, ok := call.Args[1].(*ast.FuncLit); ok && callback.Type.Params != nil && len(callback.Type.Params.List) == 1 {
					if typ, ok := baseType(callback.Type.Params.List[0].Type).(*ast.SelectorExpr); ok && typ.Sel.Name == "T" {
						walk(callback.Body, path+"/.*")
						return false
					}
				}
			}
			return true
		})
	}
	if fn.decl.Body != nil {
		walk(fn.decl.Body, "")
	}
	return calls, failure
}

func (source *suiteSource) methodSet(key string, load func(string) (*suiteSource, error), visiting map[string]bool) (map[string]promotedMethod, error) {
	if visiting[key] {
		return nil, fmt.Errorf("cyclic suite embedding %s", key)
	}
	visiting[key] = true
	defer delete(visiting, key)
	methods := map[string]promotedMethod{}
	for name, method := range source.methods[key] {
		methods[name] = promotedMethod{function: method}
	}
	typ, ok := source.types[key]
	if !ok {
		return nil, fmt.Errorf("unresolved suite type %s", key)
	}
	file := source.typeFiles[key]
	var embedded []ast.Expr
	if structure, ok := typ.Type.(*ast.StructType); ok {
		for _, field := range structure.Fields.List {
			if len(field.Names) == 0 {
				embedded = append(embedded, field.Type)
			}
		}
	} else if typ.Assign.IsValid() {
		embedded = append(embedded, typ.Type)
	}
	promoted := map[string]promotedMethod{}
	for _, expr := range embedded {
		var inherited map[string]promotedMethod
		var err error
		switch name := baseType(expr).(type) {
		case *ast.Ident:
			inherited, err = source.methodSet(file.Name.Name+"."+name.Name, load, visiting)
		case *ast.SelectorExpr:
			qualifier, ok := name.X.(*ast.Ident)
			if !ok || load == nil {
				return nil, fmt.Errorf("unresolved embedded suite type in %s", key)
			}
			external, loadErr := load(importPath(file, qualifier.Name))
			if loadErr != nil {
				return nil, loadErr
			}
			found := false
			for externalKey := range external.types {
				if strings.HasSuffix(externalKey, "."+name.Sel.Name) {
					found = true
					inherited, err = external.methodSet(externalKey, load, map[string]bool{})
					break
				}
			}
			if !found {
				return nil, fmt.Errorf("unresolved embedded suite type %s.%s", qualifier.Name, name.Sel.Name)
			}
			for method := range inherited {
				if strings.HasPrefix(method, "Test") {
					return nil, fmt.Errorf("cross-package promoted test method %s.%s requires explicit registration coverage", name.Sel.Name, method)
				}
			}
		}
		if err != nil {
			return nil, err
		}
		for name, method := range inherited {
			if _, direct := methods[name]; direct {
				continue
			}
			method.depth++
			if previous, duplicate := promoted[name]; duplicate {
				if previous.depth < method.depth {
					continue
				}
				if previous.depth == method.depth {
					method.ambiguous = true
				}
			}
			promoted[name] = method
		}
	}
	for name, method := range promoted {
		if method.ambiguous && strings.HasPrefix(name, "Test") {
			return nil, fmt.Errorf("ambiguous promoted method %s.%s", key, name)
		}
		methods[name] = method
	}
	return methods, nil
}

func parseSuiteInventory(dir string, testFiles, goFiles []string, load func(string) (*suiteSource, error)) (suiteInventory, error) {
	inventory := suiteInventory{files: map[string]string{}, unitFiles: map[string][]string{}}
	source, err := parseSuiteSource(dir, append(slices.Clone(testFiles), goFiles...))
	if err != nil {
		return inventory, err
	}
	methodRows := map[*suiteFunction]*SuiteMethod{}
	for _, methods := range source.methods {
		for name, method := range methods {
			if strings.HasPrefix(name, "Test") && slices.Contains(testFiles, method.file) {
				methodRows[method] = &SuiteMethod{Suite: method.key, Name: name, File: method.file}
			}
		}
	}
	var expand func(*suiteFunction, string, string, []string, map[*suiteFunction]bool) ([]string, error)
	expand = func(fn *suiteFunction, path, wrapper string, files []string, active map[*suiteFunction]bool) ([]string, error) {
		if active[fn] {
			return nil, fmt.Errorf("cyclic suite registration %s", path)
		}
		active[fn] = true
		defer delete(active, fn)
		files = append(slices.Clone(files), fn.file)
		calls, err := suiteCalls(fn)
		if err != nil {
			return nil, err
		}
		var units []string
		if len(calls) == 0 {
			inventory.files[path] = fn.file
			inventory.unitFiles[path] = files
			units = append(units, path)
		}
		for _, call := range calls {
			entry := path + call.path
			inventory.Registrations = append(inventory.Registrations, SuiteRegistration{Wrapper: wrapper, Suite: call.suite, Path: entry, File: fn.file})
			methods, err := source.methodSet(call.suite, load, map[string]bool{})
			if err != nil {
				return nil, err
			}
			var names []string
			for name := range methods {
				if strings.HasPrefix(name, "Test") {
					names = append(names, name)
				}
			}
			sort.Strings(names)
			if len(names) == 0 {
				return nil, fmt.Errorf("registered suite %s has no test methods", call.suite)
			}
			for _, name := range names {
				children, err := expand(methods[name].function, entry+"/"+name, wrapper, files, active)
				if err != nil {
					return nil, err
				}
				units = append(units, children...)
			}
		}
		if row, ok := methodRows[fn]; ok {
			row.Units = append(row.Units, units...)
		}
		return units, nil
	}
	for _, name := range testFiles {
		file := source.files[name]
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Recv != nil || !isTestName(fn.Name.Name) || fn.Name.Name == "TestMain" || fn.Type.TypeParams != nil || fn.Type.Params == nil || len(fn.Type.Params.List) != 1 || len(fn.Type.Params.List[0].Names) > 1 {
				continue
			}
			units, err := expand(&suiteFunction{decl: fn, file: name, ast: file}, fn.Name.Name, fn.Name.Name, nil, map[*suiteFunction]bool{})
			if err != nil {
				return inventory, err
			}
			inventory.Tests = append(inventory.Tests, units...)
		}
	}
	for _, row := range methodRows {
		if len(row.Units) == 0 {
			return inventory, fmt.Errorf("unregistered test method %s.%s in %s", row.Suite, row.Name, row.File)
		}
		sort.Strings(row.Units)
		row.Units = slices.Compact(row.Units)
		inventory.Methods = append(inventory.Methods, *row)
	}
	sort.Strings(inventory.Tests)
	inventory.Tests = slices.Compact(inventory.Tests)
	sort.Slice(inventory.Methods, func(i, j int) bool {
		a, b := inventory.Methods[i], inventory.Methods[j]
		return a.Suite+"."+a.Name < b.Suite+"."+b.Name
	})
	sort.Slice(inventory.Registrations, func(i, j int) bool {
		a, b := inventory.Registrations[i], inventory.Registrations[j]
		return a.Path+"/"+a.Suite < b.Path+"/"+b.Suite
	})
	return inventory, nil
}
