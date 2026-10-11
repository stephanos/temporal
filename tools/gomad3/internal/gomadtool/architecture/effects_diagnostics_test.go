package architecture

import (
	"bytes"
	"errors"
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func effectDiagnosticsProgram(t *testing.T) *Program {
	t.Helper()
	const module = "example.invalid/effectdiagnostics"
	path := module + "/record"
	dir := filepath.Join(t.TempDir(), "record")
	files := token.NewFileSet()
	file, err := parser.ParseFile(files, filepath.Join(dir, "pure.go"), `package record
func Root() { helper() }
func helper() { go leaf() }
func leaf() {}
`, 0)
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
	program := &Program{
		Module: module, Files: files, Metadata: map[string]Package{path: metadata},
		Packages: map[string]*SourcePackage{path: pkg}, Functions: map[string]*function{},
	}
	for _, declaration := range file.Decls {
		if declaration, ok := declaration.(*ast.FuncDecl); ok {
			object := info.Defs[declaration.Name].(*types.Func)
			program.Functions[functionID(object)] = &function{object: object, declaration: declaration, pkg: pkg}
		}
	}
	return program
}

func captureEffectDiagnostics(t *testing.T, program *Program) ([]Finding, string) {
	t.Helper()
	output, err := os.CreateTemp(t.TempDir(), "effect-diagnostics-")
	if err != nil {
		t.Fatal(err)
	}
	original := os.Stderr
	os.Stderr = output
	defer func() {
		os.Stderr = original
		if err := output.Close(); err != nil {
			t.Error(err)
		}
	}()
	findings := program.Effects()
	if _, err := output.Seek(0, io.SeekStart); err != nil {
		t.Fatal(err)
	}
	data, err := io.ReadAll(output)
	if err != nil {
		t.Fatal(err)
	}
	return findings, string(data)
}

func TestEffectDiagnosticsOptInPreservesFindings(t *testing.T) {
	program := effectDiagnosticsProgram(t)
	t.Setenv("GOMAD3_EFFECT_DIAGNOSTICS", "")
	baseline, output := captureEffectDiagnostics(t, program)
	if output != "" {
		t.Fatalf("default analysis wrote diagnostics: %s", output)
	}
	if len(baseline) != 2 || baseline[0].Category != "host-effect" || baseline[1].Category != "host-effect" {
		t.Fatalf("fixture lost transitive goroutine findings: %v", baseline)
	}
	t.Setenv("GOMAD3_EFFECT_DIAGNOSTICS", "true")
	findings, output := captureEffectDiagnostics(t, program)
	if output != "" || !reflect.DeepEqual(findings, baseline) {
		t.Fatalf("non-opt-in changed behavior: %v, %s", findings, output)
	}
	t.Setenv("GOMAD3_EFFECT_DIAGNOSTICS", "1")
	findings, output = captureEffectDiagnostics(t, program)
	if !reflect.DeepEqual(findings, baseline) {
		t.Fatalf("diagnostics changed findings: got %v want %v", findings, baseline)
	}
	for _, fragment := range []string{
		`effect-analysis START root="example.invalid/effectdiagnostics/record.Root"`,
		`effect-analysis END root="example.invalid/effectdiagnostics/record.Root"`,
		"/record/pure.go:2:1", "calls=3", "max_chain=3", "memo_entries=3", "active_context_entries=0",
	} {
		if !strings.Contains(output, fragment) {
			t.Fatalf("missing %q in diagnostics:\n%s", fragment, output)
		}
	}
}

func TestEffectDiagnosticsSamplesAreBounded(t *testing.T) {
	program := effectDiagnosticsProgram(t)
	root := "example.invalid/effectdiagnostics/record.Root"
	pos := program.Functions[root].declaration.Pos()
	var output bytes.Buffer
	a := &effectAnalysis{
		program: program, root: root, chain: []string{"oldest", "older", "one", "two", "three", "four", "five", "six"},
		memo: map[string]*abstractValue{"first": {}, "second": {}}, active: map[string]bool{"active": true},
		diagnostics: &effectDiagnostics{output: &output},
	}
	a.diagnostics.calls = 9998
	a.diagnosticCall(root, pos)
	if output.Len() != 0 {
		t.Fatal("sample emitted before interval")
	}
	a.diagnosticCall(root, pos)
	if strings.Count(output.String(), "effect-analysis SAMPLE ") != 1 {
		t.Fatalf("missing interval sample: %s", output.String())
	}
	for interval := uint64(2); interval <= 257; interval++ {
		a.diagnostics.calls = interval*10000 - 1
		a.diagnosticCall(root, pos)
	}
	if count := strings.Count(output.String(), "effect-analysis SAMPLE "); count != 256 {
		t.Fatalf("sample limit: got %d want 256", count)
	}
	for _, fragment := range []string{"memo_entries=2", "active_context_entries=1", "max_chain=9", `chain=["one" "two" "three" "four" "five" "six"]`, "/record/pure.go:2:1"} {
		if !strings.Contains(output.String(), fragment) {
			t.Fatalf("missing %q in sample", fragment)
		}
	}
	if strings.Contains(output.String(), "oldest") || strings.Contains(output.String(), "older") {
		t.Fatal("sample retained an unbounded chain")
	}
}

type failedEffectDiagnostics struct{}

func (failedEffectDiagnostics) Write([]byte) (int, error) {
	return 0, errors.New("diagnostic output failed")
}

func TestEffectDiagnosticsWriterFailureLeavesFindings(t *testing.T) {
	a := &effectAnalysis{
		findings:    map[string]Finding{"finding": {Category: "host-effect", Detail: "existing finding"}},
		diagnostics: &effectDiagnostics{output: failedEffectDiagnostics{}},
	}
	a.diagnosticLine("diagnostic message\n")
	if a.diagnostics != nil || len(a.findings) != 1 || a.findings["finding"].Detail != "existing finding" {
		t.Fatalf("diagnostic failure changed checker state: %v", a.findings)
	}
	a.diagnosticCall("ignored", token.NoPos)
}
