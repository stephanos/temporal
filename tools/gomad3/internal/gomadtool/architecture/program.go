package architecture

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"path/filepath"
	"strings"
)

type SourcePackage struct {
	Metadata Package
	Types    *types.Package
	Info     *types.Info
	Files    []*ast.File
}

type Program struct {
	Module    string
	Files     *token.FileSet
	Metadata  map[string]Package
	Packages  map[string]*SourcePackage
	Functions map[string]*function
}

type function struct {
	object      *types.Func
	declaration *ast.FuncDecl
	pkg         *SourcePackage
}

func Load(root, goCommand string, platform Platform) (*Program, error) {
	module, err := Module(root)
	if err != nil {
		return nil, err
	}
	metadata, err := List(root, goCommand, platform, true)
	if err != nil {
		return nil, err
	}
	p := &Program{Module: module, Files: token.NewFileSet(), Metadata: map[string]Package{}, Packages: map[string]*SourcePackage{}, Functions: map[string]*function{}}
	for _, pkg := range metadata {
		p.Metadata[pkg.ImportPath] = pkg
	}
	for _, pkg := range metadata {
		if !Within(pkg.ImportPath, module) || Within(pkg.ImportPath, module+"/toolchain/runtime/overlay") || pkg.ImportPath == module {
			continue
		}
		if _, err := p.Import(pkg.ImportPath); err != nil {
			return nil, err
		}
	}
	return p, nil
}

func (p *Program) Import(path string) (*types.Package, error) {
	if path == "unsafe" {
		return types.Unsafe, nil
	}
	if pkg := p.Packages[path]; pkg != nil {
		return pkg.Types, nil
	}
	metadata, found := p.Metadata[path]
	if !found {
		return nil, fmt.Errorf("package %s has no source metadata", path)
	}
	if metadata.Error != nil {
		return nil, fmt.Errorf("package %s: %s", path, metadata.Error.Err)
	}
	pkg := &SourcePackage{Metadata: metadata, Info: &types.Info{Types: map[ast.Expr]types.TypeAndValue{}, Defs: map[*ast.Ident]types.Object{}, Uses: map[*ast.Ident]types.Object{}, Selections: map[*ast.SelectorExpr]*types.Selection{}, Instances: map[*ast.Ident]types.Instance{}, Implicits: map[ast.Node]types.Object{}}}
	names := append(append([]string{}, metadata.GoFiles...), metadata.CgoFiles...)
	if len(metadata.CgoFiles) > 0 {
		names = metadata.CompiledGoFiles
		if len(names) == 0 {
			return nil, fmt.Errorf("package %s has no compiled cgo source metadata", path)
		}
	}
	for _, name := range names {
		filename := name
		if !filepath.IsAbs(filename) {
			filename = filepath.Join(metadata.Dir, name)
		}
		file, err := parser.ParseFile(p.Files, filename, nil, 0)
		if err != nil {
			return nil, err
		}
		pkg.Files = append(pkg.Files, file)
	}
	config := types.Config{Importer: sourceImporter{p, metadata.ImportMap}, GoVersion: "go1.27"}
	checked, err := config.Check(path, p.Files, pkg.Files, pkg.Info)
	if err != nil {
		return nil, fmt.Errorf("type-check %s: %w", path, err)
	}
	pkg.Types = checked
	p.Packages[path] = pkg
	for _, file := range pkg.Files {
		for _, decl := range file.Decls {
			if decl, ok := decl.(*ast.FuncDecl); ok {
				if obj, ok := pkg.Info.Defs[decl.Name].(*types.Func); ok {
					p.Functions[functionID(obj)] = &function{obj, decl, pkg}
				}
			}
		}
	}
	return checked, nil
}

type sourceImporter struct {
	program *Program
	imports map[string]string
}

func (i sourceImporter) Import(path string) (*types.Package, error) {
	if resolved := i.imports[path]; resolved != "" {
		path = resolved
	}
	return i.program.Import(path)
}

func functionID(fn *types.Func) string {
	if fn == nil {
		return ""
	}
	fn = fn.Origin()
	path := ""
	if fn.Pkg() != nil {
		path = fn.Pkg().Path() + "."
	}
	signature := fn.Type().(*types.Signature)
	if signature.Recv() != nil {
		return path + types.TypeString(signature.Recv().Type(), func(*types.Package) string { return "" }) + "." + fn.Name()
	}
	return path + fn.Name()
}

func (p *Program) PublicSignatures(consumer string) []Finding {
	var findings []Finding
	for path, pkg := range p.Packages {
		if !Within(path, p.Module) || strings.Contains(path, "/internal/") || strings.HasSuffix(path, "/internal") || pkg.Types == nil {
			continue
		}
		for _, name := range pkg.Types.Scope().Names() {
			obj := pkg.Types.Scope().Lookup(name)
			if !obj.Exported() {
				continue
			}
			seen := map[types.Type]bool{}
			var walk func(types.Type, string)
			walk = func(typ types.Type, route string) {
				if typ == nil || seen[typ] {
					return
				}
				seen[typ] = true
				checkName := func(name *types.TypeName) {
					if name.Pkg() != nil && !CanImport(consumer, name.Pkg().Path()) {
						findings = append(findings, Finding{Category: "public-signature", Path: path + "." + route, Detail: "inaccessible type " + name.Pkg().Path() + "." + name.Name()})
					}
				}
				switch typ := typ.(type) {
				case *types.Alias:
					checkName(typ.Obj())
					for i := 0; i < typ.TypeArgs().Len(); i++ {
						walk(typ.TypeArgs().At(i), route)
					}
					if typ.Obj().Pkg() != nil && !Within(typ.Obj().Pkg().Path(), p.Module) {
						break
					}
					walk(typ.Rhs(), route)
					walkTypeParameters(typ.TypeParams(), walk, route)
				case *types.Named:
					checkName(typ.Obj())
					for i := 0; i < typ.TypeArgs().Len(); i++ {
						walk(typ.TypeArgs().At(i), route)
					}
					if typ.Obj().Pkg() != nil && !Within(typ.Obj().Pkg().Path(), p.Module) {
						break
					}
					walk(typ.Underlying(), route)
					walkTypeParameters(typ.TypeParams(), walk, route)
					for _, receiver := range []types.Type{typ, types.NewPointer(typ)} {
						methods := types.NewMethodSet(receiver)
						for i := 0; i < methods.Len(); i++ {
							method := methods.At(i).Obj()
							if method.Exported() {
								walk(method.Type(), route+"."+method.Name())
							}
						}
					}
				case *types.Pointer:
					walk(typ.Elem(), route)
				case *types.Slice:
					walk(typ.Elem(), route)
				case *types.Array:
					walk(typ.Elem(), route)
				case *types.Map:
					walk(typ.Key(), route)
					walk(typ.Elem(), route)
				case *types.Chan:
					walk(typ.Elem(), route)
				case *types.Struct:
					for i := 0; i < typ.NumFields(); i++ {
						field := typ.Field(i)
						if field.Exported() || field.Embedded() {
							walk(field.Type(), route+"."+field.Name())
						}
					}
				case *types.Signature:
					walkTypeParameters(typ.TypeParams(), walk, route)
					walkTypeParameters(typ.RecvTypeParams(), walk, route)
					for _, tuple := range []*types.Tuple{typ.Params(), typ.Results()} {
						if tuple != nil {
							for i := 0; i < tuple.Len(); i++ {
								walk(tuple.At(i).Type(), route)
							}
						}
					}
				case *types.Interface:
					typ.Complete()
					for i := 0; i < typ.NumMethods(); i++ {
						walk(typ.Method(i).Type(), route+"."+typ.Method(i).Name())
					}
					for i := 0; i < typ.NumEmbeddeds(); i++ {
						walk(typ.EmbeddedType(i), route)
					}
				case *types.TypeParam:
					walk(typ.Constraint(), route)
				case *types.Union:
					for i := 0; i < typ.Len(); i++ {
						walk(typ.Term(i).Type(), route)
					}
				case *types.Tuple:
					for i := 0; i < typ.Len(); i++ {
						walk(typ.At(i).Type(), route)
					}
				}
			}
			walk(obj.Type(), name)
		}
	}
	return findings
}

func walkTypeParameters(params *types.TypeParamList, walk func(types.Type, string), route string) {
	if params != nil {
		for i := 0; i < params.Len(); i++ {
			walk(params.At(i).Constraint(), route)
		}
	}
}

func CanImport(consumer, imported string) bool {
	parts := strings.Split(imported, "/")
	for i, part := range parts {
		if part == "internal" {
			parent := strings.Join(parts[:i], "/")
			if parent == "" || !Within(consumer, parent) {
				return false
			}
		}
	}
	return true
}
