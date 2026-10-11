package architecture

import (
	"bytes"
	"fmt"
	"go/token"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func effectRuntimeDiagnosticsProgram(t *testing.T) *Program {
	t.Helper()
	const module = "example.invalid/runtimegateway"
	root := t.TempDir()
	metadata := map[string]Package{}
	for path, source := range map[string]string{
		module + "/record": "package record\nimport \"runtime\"\nfunc Root(){helper()}\nfunc helper(){go runtime.Gateway();runtime.Gateway()}\n",
		"runtime":          "package runtime\nfunc Gateway(){}\n",
	} {
		dir := filepath.Join(root, filepath.Base(path))
		if err := os.MkdirAll(dir, 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "pure.go"), []byte(source), 0600); err != nil {
			t.Fatal(err)
		}
		metadata[path] = Package{ImportPath: path, Dir: dir, GoFiles: []string{"pure.go"}}
	}
	program := &Program{Module: module, Files: token.NewFileSet(), Metadata: metadata, Packages: map[string]*SourcePackage{}, Functions: map[string]*function{}}
	if _, err := program.Import(module + "/record"); err != nil {
		t.Fatal(err)
	}
	return program
}

func TestEffectRuntimeDiagnosticsFirstEntryPreservesFindings(t *testing.T) {
	program := effectRuntimeDiagnosticsProgram(t)
	t.Setenv("GOMAD3_EFFECT_DIAGNOSTICS", "")
	baseline, output := captureEffectDiagnostics(t, program)
	if output != "" {
		t.Fatalf("default runtime diagnostic output: %s", output)
	}
	if len(baseline) != 2 || baseline[0].Category != "host-effect" || baseline[1].Category != "host-effect" {
		t.Fatalf("fixture findings=%v", baseline)
	}
	t.Setenv("GOMAD3_EFFECT_DIAGNOSTICS", "1")
	findings, output := captureEffectDiagnostics(t, program)
	if !reflect.DeepEqual(findings, baseline) {
		t.Fatalf("diagnostics changed findings: %v vs %v", findings, baseline)
	}
	var first string
	count := 0
	for _, line := range strings.Split(output, "\n") {
		if strings.HasPrefix(line, `effect-analysis FIRST_RUNTIME_ENTRY root="example.invalid/runtimegateway/record.Root"`) {
			first = line
			count++
		}
	}
	if count != 1 {
		t.Fatalf("first runtime markers=%d output=%s", count, output)
	}
	for _, fragment := range []string{
		`function="runtime.Gateway"`,
		`chain=["example.invalid/runtimegateway/record.Root" "example.invalid/runtimegateway/record.helper" "runtime.Gateway"]`,
		"/record/pure.go:3:1", "/record/pure.go:4:1", "/runtime/pure.go:2:1", "chain_positions=[", "chain_depth=3", "truncated=false",
	} {
		if !strings.Contains(first, fragment) {
			t.Fatalf("missing %q in %s", fragment, first)
		}
	}
}

func TestEffectRuntimeDiagnosticsBoundedExactEntryAndReset(t *testing.T) {
	program := effectRuntimeDiagnosticsProgram(t)
	root := "example.invalid/runtimegateway/record.Root"
	pos := program.Functions[root].declaration.Pos()
	var output bytes.Buffer
	a := &effectAnalysis{program: program, root: root, findings: map[string]Finding{}, diagnostics: &effectDiagnostics{output: &output}}
	for _, id := range []string{"runtime/metrics.Read", "internal/runtime/atomic.Load", "runtimex.Call", "runtime"} {
		a.diagnosticCall(id, pos)
	}
	if output.Len() != 0 {
		t.Fatalf("non-runtime package matched: %s", output.String())
	}
	for index := 0; index < 140; index++ {
		a.chain = append(a.chain, fmt.Sprintf("chain%03d", index))
	}
	before := append([]string(nil), a.chain...)
	a.diagnosticCall("runtime.First", pos)
	a.diagnosticCall("runtime.Second", pos)
	if !reflect.DeepEqual(a.chain, before) {
		t.Fatal("diagnostic mutated analysis chain")
	}
	if strings.Count(output.String(), "FIRST_RUNTIME_ENTRY") != 1 {
		t.Fatalf("duplicate first marker: %s", output.String())
	}
	for _, fragment := range []string{`chain=["example.invalid/runtimegateway/record.Root" "chain000"`, `"chain126"`, "chain_depth=142", "truncated=true"} {
		if !strings.Contains(output.String(), fragment) {
			t.Fatalf("missing bounded prefix %q", fragment)
		}
	}
	if strings.Contains(output.String(), `"chain127"`) {
		t.Fatal("runtime chain exceeded 128 IDs")
	}
	output.Reset()
	for range 2 {
		a.analyze(pos, func() { a.diagnosticCall("runtime.First", pos); a.diagnosticCall("runtime.Second", pos) })
	}
	if strings.Count(output.String(), "FIRST_RUNTIME_ENTRY") != 2 || len(a.findings) != 0 {
		t.Fatalf("root reset changed findings or first-entry count: %s", output.String())
	}
}
