package authoring

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"os"
	"path/filepath"
	"testing"
)

func TestGenerateStandaloneConsumerPackage(t *testing.T) {
	root := t.TempDir()
	request := validRequest()
	approval, err := ApprovalSHA256(request)
	if err != nil {
		t.Fatal(err)
	}
	if err := Generate(root, request, approval); err != nil {
		t.Fatal(err)
	}
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, filepath.Join(root, "packs_generated_test.go"), nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	configuration := types.Config{}
	if _, err := configuration.Check("example.com/consumer/compatibility", fset, []*ast.File{file}, nil); err != nil {
		t.Fatalf("generated standalone consumer package does not compile: %v", err)
	}
}

func TestGenerateRequiresExactApprovalAndCheckDetectsDrift(t *testing.T) {
	root := t.TempDir()
	request := validRequest()
	approval, err := ApprovalSHA256(request)
	if err != nil {
		t.Fatal(err)
	}
	if err := Generate(root, request, "sha256:0000000000000000000000000000000000000000000000000000000000000000"); err == nil {
		t.Fatal("Generate() accepted the wrong approval")
	}
	if _, err := os.Stat(filepath.Join(root, "packs", "example-pack.json")); !os.IsNotExist(err) {
		t.Fatalf("rejected generation published a pack: %v", err)
	}

	if err := Generate(root, request, approval); err != nil {
		t.Fatal(err)
	}
	if err := Check(root); err != nil {
		t.Fatal(err)
	}
	packPath := filepath.Join(root, "packs", "example-pack.json")
	pack, err := os.ReadFile(packPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(packPath, append(pack, '\n'), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := Check(root); err == nil {
		t.Fatal("Check() accepted a modified generated pack")
	}
}

func TestRegenerateUsesOnlyRecordedExactApprovals(t *testing.T) {
	root := t.TempDir()
	request := validRequest()
	approval, err := ApprovalSHA256(request)
	if err != nil {
		t.Fatal(err)
	}
	if err := Generate(root, request, approval); err != nil {
		t.Fatal(err)
	}
	packPath := filepath.Join(root, "packs", request.ID+".json")
	if err := os.WriteFile(packPath, []byte("stale\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := Regenerate(root); err != nil {
		t.Fatal(err)
	}
	if err := Check(root); err != nil {
		t.Fatal(err)
	}
}
