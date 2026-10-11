package architecture

import (
	"go/types"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Stock source is listed independently, then imported into the fixture's own
// FileSet. Metadata injection alone must never stand in for parsed source proof.
func stockSummaryMetadata(t *testing.T) []Package {
	t.Helper()
	root := t.TempDir()
	for name, source := range map[string]string{
		"go.mod":  "module example.invalid/stocksummary\n\ngo 1.27.1\n",
		"seed.go": "package seed\nimport (_ \"reflect\"; _ \"fmt\"; _ \"encoding/json\")\n",
	} {
		if err := os.WriteFile(filepath.Join(root, name), []byte(source), 0600); err != nil {
			t.Fatal(err)
		}
	}
	metadata, err := List(root, "go", Platform{"linux", "amd64"}, true)
	if err != nil {
		t.Fatal(err)
	}
	return metadata
}

func importStockSummarySources(t *testing.T, program *Program) {
	t.Helper()
	for _, metadata := range stockSummaryMetadata(t) {
		if metadata.Standard {
			program.Metadata[metadata.ImportPath] = metadata
		}
	}
	for _, path := range []string{"reflect", "fmt", "encoding/json"} {
		if existing := program.Packages[path]; existing != nil && (!existing.Metadata.Standard || existing.Metadata.Dir != program.Metadata[path].Dir) {
			t.Fatalf("fixture cannot replace nonstock imported package %s with stock metadata", path)
		}
		if _, err := program.Import(path); err != nil {
			t.Fatal(err)
		}
	}
}

func stockSummaryFunction(t *testing.T, analysis *effectAnalysis, path, receiver, name string) *types.Func {
	t.Helper()
	pkg := analysis.program.Packages[path]
	if pkg == nil {
		t.Fatalf("stock package %s was not imported", path)
	}
	if receiver == "" {
		fn, ok := pkg.Types.Scope().Lookup(name).(*types.Func)
		if !ok {
			t.Fatalf("stock function %s.%s missing", path, name)
		}
		return fn
	}
	typeName, ok := pkg.Types.Scope().Lookup(strings.TrimPrefix(receiver, "*")).(*types.TypeName)
	if !ok {
		t.Fatalf("stock receiver %s.%s missing", path, receiver)
	}
	typ := typeName.Type()
	if strings.HasPrefix(receiver, "*") {
		typ = types.NewPointer(typ)
	}
	method := types.NewMethodSet(typ).Lookup(nil, name)
	if method == nil {
		t.Fatalf("stock method %s.%s.%s missing", path, receiver, name)
	}
	return method.Obj().(*types.Func)
}

func stockSummaryProgram(t *testing.T, module, source string) *Program {
	t.Helper()
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "record"), 0700); err != nil {
		t.Fatal(err)
	}
	for name, data := range map[string]string{
		"go.mod":         "module " + module + "\n\ngo 1.27.1\n",
		"record/pure.go": source,
	} {
		if err := os.WriteFile(filepath.Join(root, name), []byte(data), 0600); err != nil {
			t.Fatal(err)
		}
	}
	program, err := Load(root, "go", Platform{"linux", "amd64"})
	if err != nil {
		t.Fatal(err)
	}
	importStockSummarySources(t, program)
	return program
}
