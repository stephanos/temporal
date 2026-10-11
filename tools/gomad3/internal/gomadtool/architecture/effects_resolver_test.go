package architecture

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
)

func effectResolverProgram(t *testing.T, source string) *Program {
	t.Helper()
	const module = "example.invalid/resolver"
	path := module + "/record"
	dir := filepath.Join(t.TempDir(), "record")
	files := token.NewFileSet()
	file, err := parser.ParseFile(files, filepath.Join(dir, "pure.go"), source, 0)
	if err != nil {
		t.Fatal(err)
	}
	info := &types.Info{
		Types: map[ast.Expr]types.TypeAndValue{}, Defs: map[*ast.Ident]types.Object{},
		Uses: map[*ast.Ident]types.Object{}, Selections: map[*ast.SelectorExpr]*types.Selection{},
		Instances: map[*ast.Ident]types.Instance{}, Implicits: map[ast.Node]types.Object{},
	}
	config := types.Config{GoVersion: "go1.27"}
	checked, err := config.Check(path, files, []*ast.File{file}, info)
	if err != nil {
		t.Fatal(err)
	}
	metadata := Package{ImportPath: path, Dir: dir}
	pkg := &SourcePackage{Metadata: metadata, Types: checked, Info: info, Files: []*ast.File{file}}
	program := &Program{Module: module, Files: files, Metadata: map[string]Package{path: metadata}, Packages: map[string]*SourcePackage{path: pkg}, Functions: map[string]*function{}}
	for _, declaration := range file.Decls {
		if declaration, ok := declaration.(*ast.FuncDecl); ok {
			object := info.Defs[declaration.Name].(*types.Func)
			program.Functions[functionID(object)] = &function{object: object, declaration: declaration, pkg: pkg}
		}
	}
	return program
}

func effectResolverAnalysis(program *Program) *effectAnalysis {
	a := &effectAnalysis{
		program: program, active: map[string]bool{}, resolving: map[types.Object]bool{},
		bindings: map[types.Object][]expressionSite{}, fields: map[*types.Var][]expressionSite{},
		calls: map[string][]callSite{}, findings: map[string]Finding{}, typeArguments: map[*types.TypeParam][]types.Type{},
		parameterOwners: indexParameterOwners(program.Functions),
	}
	for _, pkg := range program.Packages {
		for _, file := range pkg.Files {
			ast.Inspect(file, func(node ast.Node) bool {
				if call, ok := node.(*ast.CallExpr); ok {
					if fn := calledObject(pkg.Info, call.Fun); fn != nil {
						a.calls[functionID(fn)] = append(a.calls[functionID(fn)], callSite{call: call, pkg: pkg})
					}
				}
				return true
			})
		}
	}
	return a
}

func effectResolverFunctionIDs(value *abstractValue) []string {
	var ids []string
	if value != nil {
		for _, fn := range value.functions {
			ids = append(ids, functionID(fn.fn.object))
		}
	}
	sort.Strings(ids)
	return ids
}

func originalParameterScan(a *effectAnalysis, object types.Object) *abstractValue {
	var result *abstractValue
	for _, fn := range a.program.Functions {
		if fn.pkg.Types != object.Pkg() {
			continue
		}
		signature := fn.object.Type().(*types.Signature)
		for index := 0; index < signature.Params().Len(); index++ {
			if signature.Params().At(index) == object {
				for _, site := range a.calls[functionID(fn.object)] {
					if index < len(site.call.Args) {
						result = join(result, a.eval(site.pkg, site.call.Args[index], environment{}))
					}
				}
			}
		}
	}
	if result == nil {
		return valueOf(object.Type())
	}
	return result
}

func TestEffectResolverIndexedOwnersPreserveOriginalScan(t *testing.T) {
	program := effectResolverProgram(t, `package record
func Root() { Invoke(clean, dirty); Invoke(dirty, clean) }
func Invoke(first, second func()) { first() }
func clean() {}
func dirty() { go clean() }
`)
	owner := program.Functions["example.invalid/resolver/record.Invoke"]
	param := owner.object.Type().(*types.Signature).Params().At(0)
	analysis := effectResolverAnalysis(program)
	original := originalParameterScan(effectResolverAnalysis(program), param)
	got := analysis.resolve(param)
	want := []string{"example.invalid/resolver/record.clean", "example.invalid/resolver/record.dirty"}
	if ids := effectResolverFunctionIDs(got); !reflect.DeepEqual(ids, want) || !reflect.DeepEqual(ids, effectResolverFunctionIDs(original)) {
		t.Fatalf("indexed=%v original=%v want=%v", ids, effectResolverFunctionIDs(original), want)
	}
	if analysis.bindingDepth != 0 || len(analysis.resolving) != 0 {
		t.Fatal("resolver guards leaked")
	}
	duplicate := types.NewVar(param.Pos(), param.Pkg(), param.Name(), param.Type())
	if value := analysis.resolve(duplicate); len(value.functions) != 0 || !value.unknown {
		t.Fatalf("same-named object matched parameter: %#v", value)
	}
	analysis.resolving[param] = true
	if value := analysis.resolve(param); len(value.functions) != 0 || !value.unknown {
		t.Fatalf("recursive guard lost: %#v", value)
	}
	delete(analysis.resolving, param)
	clean := analysis.calls[functionID(owner.object)][0]
	analysis.bindings[param] = []expressionSite{{expression: clean.call.Args[0], pkg: clean.pkg}}
	if ids := effectResolverFunctionIDs(analysis.resolve(param)); !reflect.DeepEqual(ids, want[:1]) {
		t.Fatalf("explicit binding precedence lost: %v", ids)
	}
}

func TestEffectResolverFrozenIndexPreservesDuplicateOwnership(t *testing.T) {
	program := effectResolverProgram(t, `package record
func Root() { Invoke(clean, dirty) }
func Invoke(first, second func()) { first() }
func clean() {}
func dirty() { go clean() }
`)
	owner := program.Functions["example.invalid/resolver/record.Invoke"]
	param := owner.object.Type().(*types.Signature).Params().At(0)
	alias := *owner
	alias.object = types.NewFunc(owner.object.Pos(), param.Pkg(), "Alias", types.NewSignatureType(nil, nil, nil, types.NewTuple(param, param), types.NewTuple(), false))
	program.Functions[functionID(alias.object)] = &alias
	foreign := alias
	foreign.object = types.NewFunc(owner.object.Pos(), param.Pkg(), "Foreign", types.NewSignatureType(nil, nil, nil, types.NewTuple(param), types.NewTuple(), false))
	foreign.pkg = &SourcePackage{Types: types.NewPackage(param.Pkg().Path(), param.Pkg().Name())}
	program.Functions[functionID(foreign.object)] = &foreign
	analysis := effectResolverAnalysis(program)
	site := analysis.calls[functionID(owner.object)][0]
	analysis.calls[functionID(alias.object)] = []callSite{site, {call: &ast.CallExpr{Args: site.call.Args[:1]}, pkg: site.pkg}}
	analysis.calls[functionID(foreign.object)] = []callSite{{call: &ast.CallExpr{Args: []ast.Expr{site.call.Args[1], site.call.Args[1]}}, pkg: site.pkg}}
	if entries := len(analysis.parameterOwners[param]); entries != 4 {
		t.Fatalf("lost duplicate owners/indices: got %d want 4", entries)
	}
	original := originalParameterScan(analysis, param)
	delete(program.Functions, functionID(owner.object))
	delete(program.Functions, functionID(alias.object))
	delete(program.Functions, functionID(foreign.object))
	got := effectResolverFunctionIDs(analysis.resolve(param))
	want := []string{"example.invalid/resolver/record.clean", "example.invalid/resolver/record.dirty"}
	if !reflect.DeepEqual(got, want) || !reflect.DeepEqual(got, effectResolverFunctionIDs(original)) {
		t.Fatalf("indexed=%v original=%v want=%v", got, effectResolverFunctionIDs(original), want)
	}
	for _, entry := range analysis.parameterOwners[param] {
		if entry.fn == &foreign {
			continue
		}
		analysis.calls[functionID(entry.fn.object)] = nil
	}
	if value := analysis.resolve(param); len(value.functions) != 0 || !value.unknown {
		t.Fatalf("foreign package identity accepted: %#v", value)
	}
}

func TestEffectResolverConservativeCallbackFindings(t *testing.T) {
	for _, tc := range []struct {
		name, calls string
		want        bool
	}{
		{"unselected-dirty", "Invoke(clean, dirty)", false},
		{"second-call-dirty", "Invoke(clean, dirty); Invoke(dirty, clean)", true},
		{"captured-dirty", "factory := func(f func()) func() { return func() { f() } }; Invoke(factory(dirty), clean)", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			program := effectResolverProgram(t, "package record\nfunc Root(){"+tc.calls+"}\nfunc Invoke(first, second func()){first()}\nfunc clean(){}\nfunc dirty(){go clean()}\n")
			found := false
			for _, finding := range program.Effects() {
				if finding.Category == "host-effect" && strings.HasPrefix(finding.Detail, "example.invalid/resolver/record.Root ->") {
					found = true
				}
			}
			if found != tc.want {
				t.Fatalf("Root host effect=%v want=%v", found, tc.want)
			}
		})
	}
}
