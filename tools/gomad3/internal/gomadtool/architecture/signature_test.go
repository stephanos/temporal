package architecture

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"testing"
)

func signatureProgram(t *testing.T, source string) *Program {
	t.Helper()
	files := token.NewFileSet()
	private := types.NewPackage("example.invalid/api/internal/detail", "detail")
	obj := types.NewTypeName(token.NoPos, private, "Hidden", nil)
	typ := types.NewNamed(obj, types.Typ[types.String], nil)
	private.Scope().Insert(obj)
	private.MarkComplete()
	_ = typ
	file, err := parser.ParseFile(files, "api.go", source, 0)
	if err != nil {
		t.Fatal(err)
	}
	config := types.Config{Importer: signatureImporter{private}}
	pkg, err := config.Check("example.invalid/api", files, []*ast.File{file}, nil)
	if err != nil {
		t.Fatal(err)
	}
	return &Program{Module: "example.invalid/api", Files: files, Packages: map[string]*SourcePackage{pkg.Path(): {Types: pkg}}}
}

type signatureImporter struct{ pkg *types.Package }

func (i signatureImporter) Import(string) (*types.Package, error) { return i.pkg, nil }

func TestSignatureNestedInternalIdentity(t *testing.T) {
	p := signatureProgram(t, `package api; import "example.invalid/api/internal/detail"; type Report struct { Values []map[string]*detail.Hidden }; func Read() Report { return Report{} }`)
	findings := p.PublicSignatures("example.com/consumer")
	if len(findings) == 0 || findings[0].Category != "public-signature" {
		t.Fatalf("nested internal identity escaped: %v", findings)
	}
}

func TestImportableForeignNamedBoundary(t *testing.T) {
	foreign := types.NewPackage("example.com/foreign", "foreign")
	sealed := types.NewPackage("example.com/foreign/internal/seal", "seal")
	privateName := types.NewTypeName(token.NoPos, sealed, "Options", nil)
	private := types.NewNamed(privateName, types.NewStruct(nil, nil), nil)
	publicName := types.NewTypeName(token.NoPos, foreign, "Options", nil)
	public := types.NewAlias(publicName, private)
	p := signatureProgram(t, "package api; type Report struct{}")
	pkg := p.Packages[p.Module].Types
	pkg.Scope().Insert(types.NewVar(token.NoPos, pkg, "Foreign", public))
	if findings := p.PublicSignatures("example.com/consumer"); len(findings) != 0 {
		t.Fatalf("importable foreign API rejected: %v", findings)
	}
	pkg.Scope().Insert(types.NewVar(token.NoPos, pkg, "PrivateForeign", private))
	findings := p.PublicSignatures("example.com/consumer")
	if len(findings) != 1 || findings[0].Category != "public-signature" {
		t.Fatalf("direct foreign internal identity escaped: %v", findings)
	}
}

func TestInternalVisibilityChecksEveryParent(t *testing.T) {
	for _, test := range []struct {
		consumer, imported string
		allowed            bool
	}{
		{"example.com/api/consumer", "example.com/api/internal/detail", true},
		{"example.com/apix/consumer", "example.com/api/internal/detail", false},
		{"example.com/api/consumer", "example.com/api/internal/nested/internal/detail", false},
		{"example.com/api/internal/nested/consumer", "example.com/api/internal/nested/internal/detail", true},
		{"example.com/consumer", "example.com/api/detail", true},
	} {
		if got := CanImport(test.consumer, test.imported); got != test.allowed {
			t.Errorf("CanImport(%q, %q) = %v, want %v", test.consumer, test.imported, got, test.allowed)
		}
	}
}
