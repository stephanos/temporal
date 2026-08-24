package translate

import (
	"go/ast"
	"go/token"
	"go/types"
	"strings"

	"github.com/dave/dst"
	"github.com/dave/dst/decorator"
)

type globalsCollector struct {
	typesInfo *types.Info
	astMap    decorator.AstMap
	pkgPath   string

	globalsFields []string
	shouldShare   map[string]bool
	tests         []string

	dontTranslate map[packageSelector]bool
}

func collectGlobalsAndTests(files []*dst.File, astMap decorator.AstMap, pkgPath string, typesInfo *types.Info, dontTranslate map[packageSelector]bool) *globalsCollector {
	c := &globalsCollector{
		typesInfo: typesInfo,
		astMap:    astMap,
		pkgPath:   pkgPath,

		shouldShare: make(map[string]bool),

		dontTranslate: dontTranslate,
	}

	for _, file := range files {
		c.collectGlobals(file)
	}

	return c
}

func (t *globalsCollector) collectGlobalsDecl(genDecl *dst.GenDecl) {
	if genDecl.Tok != token.VAR {
		return
	}
	for _, spec := range genDecl.Specs {
		spec := spec.(*dst.ValueSpec)

		// XXX: handle single/multi assignment gracefully
		// XXX: do initializers in TypesInfo get split?

		for _, name := range spec.Names {
			obj, ok := t.typesInfo.Defs[t.astMap.Nodes[name].(*ast.Ident)]
			if !ok {
				continue
			}

			if _, ok := obj.(*types.Var); !ok {
				continue
			}

			if name.Name == "_" {
				continue
			}

			n := name.Name

			shouldShare := false

			if t.dontTranslate[packageSelector{Pkg: t.pkgPath, Selector: name.Name}] {
				shouldShare = true
			}

			// XXX: reuse existing err globals?
			/*
				if t.pkgPath == "io" && initializer.Lhs[0].Name() == "EOF" {
					rhs = &dst.Ident{
						Path: "io",
						Name: "EOF",
					}
				}
			*/
			/*
				// XXX: this causes problems for errors.Is(statErr, ErrNotExist)
				if typ, ok := t.TypesInfo.Types[initializer.Rhs]; ok {
					if named, ok := typ.Type.(*types.Named); ok && named.Obj().Name() == "error" {
						if name := initializer.Lhs[0].Name(); token.IsExported(name) && !isInternal(t.pkgPath) && !isVendor(t.pkgPath) {
							// XXX: can we go even deeper and rewrite all references / skip the variable altogether?
							rhs = &dst.Ident{
								Name: name,
								Path: t.pkgPath,
							}
						}
					}
				}
			*/

			if len(spec.Values) > 0 {
				if callExpr, ok := spec.Values[0].(*dst.CallExpr); ok {
					if ident, ok := callExpr.Fun.(*dst.Ident); ok && ident.Name == "MustCompile" && ident.Path == "regexp" {
						shouldShare = true
					}
					if ident, ok := callExpr.Fun.(*dst.Ident); ok && ident.Name == "NewReplacer" && ident.Path == "strings" {
						shouldShare = true
					}
				}
				if callExpr, ok := spec.Values[0].(*dst.CallExpr); ok {
					if ident, ok := callExpr.Fun.(*dst.Ident); ok && ident.Name == "New" && ident.Path == "errors" {
						shouldShare = true
					}
				}
			}

			if t.pkgPath == "golang.org/x/net/http2/hpack" && n == "staticTable" {
				shouldShare = true
			}
			if t.pkgPath == "regexp/syntax" && n == "posixGroup" {
				shouldShare = true
			}
			if t.pkgPath == "hash/crc32" && n == "IEEETable" {
				shouldShare = true
			}
			if t.pkgPath == "crypto/internal/edwards25519" {
				shouldShare = true
			}
			if t.pkgPath == "vendor/golang.org/x/net/http2/hpack" && n == "staticTable" {
				shouldShare = true
			}
			if t.pkgPath == "golang.org/x/net/http/httpguts" && n == "badTrailer" {
				shouldShare = true
			}
			if t.pkgPath == "google.golang.org/grpc/internal/transport" && n == "http2ErrConvTab" {
				shouldShare = true
			}
			if t.pkgPath == "google.golang.org/grpc/internal/transport" && n == "HTTPStatusConvTab" {
				shouldShare = true
			}
			if t.pkgPath == "github.com/google/go-cmp/cmp" && n == "randBool" {
				shouldShare = true
			}
			if t.pkgPath == "github.com/google/go-cmp/cmp/internal/diff" && n == "randBool" {
				shouldShare = true
			}
			if t.pkgPath == "html/template" {
				shouldShare = true
			}
			if t.pkgPath == "unicode" {
				shouldShare = true
			}

			t.globalsFields = append(t.globalsFields, name.Name)
			if shouldShare {
				t.shouldShare[name.Name] = true
			}
		}
	}
}

func (t *globalsCollector) collectFuncDecl(funcDecl *dst.FuncDecl) {
	if !strings.HasPrefix(funcDecl.Name.Name, "Test") {
		return
	}
	if funcDecl.Recv != nil {
		return
	}
	if funcDecl.Type.TypeParams != nil {
		return
	}
	if funcDecl.Type.Results != nil {
		return
	}
	if len(funcDecl.Type.Params.List) != 1 {
		return
	}
	param := funcDecl.Type.Params.List[0]
	if len(param.Names) != 1 {
		return
	}
	unaryExpr, ok := param.Type.(*dst.StarExpr)
	if !ok {
		return
	}
	ident, ok := unaryExpr.X.(*dst.Ident)
	if !ok {
		return
	}
	if ident.Path != "testing" || ident.Name != "T" {
		return
	}
	t.tests = append(t.tests, funcDecl.Name.Name)
}

func (t *globalsCollector) collectGlobals(file *dst.File) {
	for _, decl := range file.Decls {
		if genDecl, ok := decl.(*dst.GenDecl); ok {
			t.collectGlobalsDecl(genDecl)
		}
		if funcDecl, ok := decl.(*dst.FuncDecl); ok {
			t.collectFuncDecl(funcDecl)
		}
	}
}
