package puritycheck

import (
	"go/types"
	"strings"

	"go.temporal.io/server/tools/gomad3/internal/archcheck"
)

func PublicSignatures(root, goCommand string, platform archcheck.Platform, consumer string) ([]Finding, error) {
	p, err := load(root, goCommand, platform, func(string) bool { return true })
	if err != nil {
		return []Finding{{Category: "package-error", Path: root, Detail: err.Error()}}, err
	}
	var findings []Finding
	for path, pkg := range p.packages {
		if !canImport(consumer, path) || strings.Contains(path, "/internal/") || strings.HasSuffix(path, "/internal") {
			continue
		}
		for _, name := range pkg.types.Scope().Names() {
			obj := pkg.types.Scope().Lookup(name)
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
					if name.Pkg() != nil && !canImport(consumer, name.Pkg().Path()) {
						findings = append(findings, Finding{Category: "public-signature", Path: path + "." + route, Detail: "inaccessible type " + name.Pkg().Path() + "." + name.Name()})
					}
				}
				switch typ := typ.(type) {
				case *types.Alias:
					checkName(typ.Obj())
					for i := 0; i < typ.TypeArgs().Len(); i++ {
						walk(typ.TypeArgs().At(i), route)
					}
					if typ.Obj().Pkg() != nil && !within(typ.Obj().Pkg().Path(), p.module) {
						break
					}
					walk(typ.Rhs(), route)
					walkTypeParameters(typ.TypeParams(), walk, route)
				case *types.Named:
					checkName(typ.Obj())
					for i := 0; i < typ.TypeArgs().Len(); i++ {
						walk(typ.TypeArgs().At(i), route)
					}
					if typ.Obj().Pkg() != nil && !within(typ.Obj().Pkg().Path(), p.module) {
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
	sortFindings(findings)
	return findings, nil
}

func walkTypeParameters(params *types.TypeParamList, walk func(types.Type, string), route string) {
	if params != nil {
		for i := 0; i < params.Len(); i++ {
			walk(params.At(i).Constraint(), route)
		}
	}
}

func canImport(consumer, imported string) bool {
	parts := strings.Split(imported, "/")
	for i, part := range parts {
		if part == "internal" && !within(consumer, strings.Join(parts[:i], "/")) {
			return false
		}
	}
	return true
}
