package architecture

import (
	"go/ast"
	"go/build"
	"go/types"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const canonicalWriterWorldPackage = "go.temporal.io/server/tools/gomad3/world"

// This helper extends the existing fixture without replacing production world
// declarations. Only the bridge is test-owned; writer, errors and scalar hooks
// come from the complete selected world inventory in the checkout under test.
func canonicalWriterTargetFixture(t *testing.T, source, worldBridge string) *Program {
	t.Helper()
	return canonicalWriteFixtureBeforeImport(t, source, func(program *Program) {
		context := build.Default
		context.GOOS, context.GOARCH = "linux", "amd64"
		context.CgoEnabled = false
		context.BuildTags = []string{"test_dep"}
		var discover func(string)
		discover = func(path string) {
			if path == "unsafe" || program.Metadata[path].ImportPath != "" {
				return
			}
			selected, err := context.Import(path, "", 0)
			if err != nil {
				t.Fatal(err)
			}
			if !selected.Goroot || len(selected.CgoFiles) != 0 {
				t.Fatalf("nonstock writer fixture dependency: %s", path)
			}
			program.Metadata[path] = Package{ImportPath: path, Dir: selected.Dir, Standard: true,
				GoFiles: selected.GoFiles, Imports: selected.Imports, SFiles: selected.SFiles, HFiles: selected.HFiles}
			for _, dependency := range selected.Imports {
				discover(dependency)
			}
		}
		record := program.Metadata[program.Module+"/record"]
		record.Imports = []string{"bytes", "io", canonicalWritePackage}
		if strings.Contains(source, `"bufio"`) {
			record.Imports = append(record.Imports, "bufio")
		}
		if worldBridge != "" {
			selected, err := context.ImportDir("../../../world", 0)
			if err != nil {
				t.Fatal(err)
			}
			directory := t.TempDir()
			files := append([]string{}, selected.GoFiles...)
			for _, name := range files {
				data, err := os.ReadFile(filepath.Join(selected.Dir, name))
				if err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(directory, name), data, 0600); err != nil {
					t.Fatal(err)
				}
			}
			const bridgeName = "canonical_writer_target_fixture.go"
			if err := os.WriteFile(filepath.Join(directory, bridgeName), []byte(worldBridge), 0600); err != nil {
				t.Fatal(err)
			}
			files = append(files, bridgeName)
			program.Metadata[canonicalWriterWorldPackage] = Package{ImportPath: canonicalWriterWorldPackage,
				Dir: directory, GoFiles: files, Imports: selected.Imports}
			for _, dependency := range selected.Imports {
				discover(dependency)
			}
			record.Imports = []string{canonicalWriterWorldPackage}
		}
		for _, dependency := range record.Imports {
			discover(dependency)
		}
		program.Metadata[program.Module+"/record"] = record
	})
}

func canonicalWriterAnalyzeRoot(t *testing.T, program *Program) (*effectAnalysis, *abstractValue) {
	t.Helper()
	root := program.Functions[program.Module+"/record.Root"]
	if root == nil {
		t.Fatal("writer fixture Root not imported")
	}
	analysis := effectResolverAnalysis(program)
	// The resolver fixture intentionally indexes only callsites. Writer error
	// globals additionally need the same initializer/selector binding sites as
	// the complete Effects analysis; otherwise resolve sees a bare error slot.
	for _, pkg := range program.Packages {
		if pkg.Metadata.Standard {
			continue
		}
		for _, file := range pkg.Files {
			ast.Inspect(file, func(node ast.Node) bool {
				switch node := node.(type) {
				case *ast.ValueSpec:
					for i, name := range node.Names {
						if i < len(node.Values) {
							analysis.bindings[pkg.Info.Defs[name]] = append(analysis.bindings[pkg.Info.Defs[name]], expressionSite{expression: node.Values[i], pkg: pkg})
						}
					}
				case *ast.AssignStmt:
					for i, target := range node.Lhs {
						if selector, ok := target.(*ast.SelectorExpr); ok && i < len(node.Rhs) {
							if variable, ok := pkg.Info.Uses[selector.Sel].(*types.Var); ok {
								analysis.fields[variable] = append(analysis.fields[variable], expressionSite{expression: node.Rhs[i], pkg: pkg})
							}
						}
					}
				}
				return true
			})
		}
	}
	result := analysis.call(&boundFunction{fn: root, pkg: root.pkg}, nil, root.declaration.Pos())
	return analysis, result
}

const canonicalWriterWorldBridge = `package world
import (
 "bytes"
 "compress/gzip"
 "crypto/sha256"
 "io"
 "go.temporal.io/server/tools/gomad3/internal/canonicaljson"
)
func CanonicalWriterTarget(snapshot Snapshot) error {
 var compressed bytes.Buffer
 compressedOutput := &compressedBoundedWriter{writer: &compressed, limit: 1 << 20, dimension: "compressed-snapshot-gzip-bytes"}
 compressor := gzip.NewWriter(compressedOutput)
 expandedOutput := &compressedBoundedWriter{writer: io.MultiWriter(compressor, sha256.New()), limit: 1 << 22, dimension: "compressed-snapshot-expanded-bytes"}
 if err := canonicaljson.WriteCanonicalJSON(expandedOutput, snapshot); err != nil { return err }
 return compressor.Close()
}
`

func TestCanonicalWriteSummaryActualWorldSnapshotWriterChain(t *testing.T) {
	program := canonicalWriterTargetFixture(t, `package record
import "go.temporal.io/server/tools/gomad3/world"
func Root() error {
 return world.CanonicalWriterTarget(world.Snapshot{
  SchemaVersion: world.SchemaVersion, Now: world.InitialTime,
  Config: world.Config{Seed: world.Seed(7)},
  NextRequestID: world.RequestID(2), NextEventID: world.EventID(2), NextTransition: world.Sequence(2),
  Requests: []world.RequestSnapshot{{ID: world.RequestID(1), Request: world.Request{Kind:"read", Payload:[]byte{1}}, EventID:world.EventID(1)}},
  Events: []world.EventSnapshot{}, Transitions: []world.Transition{},
 })
}
`, canonicalWriterWorldBridge)
	for _, id := range []string{
		canonicalWriterWorldPackage + ".*compressedBoundedWriter.Write",
		canonicalWriterWorldPackage + ".*CapacityError.Error",
		canonicalWriterWorldPackage + ".*CapacityError.Unwrap",
		canonicalWriterWorldPackage + ".Seed.MarshalJSON",
		canonicalWriterWorldPackage + ".LogicalTime.MarshalJSON",
	} {
		if program.Functions[id] == nil {
			t.Fatalf("actual production writer/error/value method absent: %s", id)
		}
	}
	analysis, result := canonicalWriterAnalyzeRoot(t, program)
	if result == nil || result.unknown || len(analysis.findings) != 0 {
		t.Fatalf("actual Snapshot -> bounded -> MultiWriter(gzip -> bounded -> Buffer, sha256) rejected: result=%#v findings=%v", result, analysis.findings)
	}
}

func TestCanonicalWriteSummaryHostileDirectAndForwardedWriters(t *testing.T) {
	for _, test := range []struct {
		name, declarations, fields, write, setup, valueSetup, value, consume, hostDetail string
	}{
		{name: "counter-gated-host", fields: "used int", write: "if w.used > 0 { go host() }; w.used++; return len(p), nil", setup: "used:1", value: "7", hostDetail: "Sink.Write -> goroutine start"},
		{name: "delegate-mutation", fields: "writer io.Writer", setup: "writer:&bytes.Buffer{}", write: "w.writer=&bytes.Buffer{};return w.writer.Write(p)", value: "7"},
		{name: "reference-mutation", fields: "ref *int", setup: "ref:new(int)", write: "*w.ref=9;return len(p),nil", value: "7"},
		{name: "input-mutation", write: "if len(p)>0 {p[0]=9};return len(p),nil", value: "7"},
		{name: "serialized-value-mutation", fields: "value *Value", setup: "value:&value", valueSetup: "value:=Value{Payload:[]byte{1}};", write: "w.value.Payload[0]=9;return len(p),nil", value: "value"},
		{name: "input-escape", declarations: "var escaped []byte", write: "escaped=p;return len(p),nil", value: "7"},
		{name: "delegate-escape", fields: "writer io.Writer", setup: "writer:&bytes.Buffer{}", declarations: "var escapedWriter io.Writer", write: "escapedWriter=w.writer;return len(p),nil", value: "7"},
		{name: "value-escape", fields: "value *Value", setup: "value:&value", valueSetup: "value:=Value{Payload:[]byte{1}};", declarations: "var escapedValue *Value", write: "escapedValue=w.value;return len(p),nil", value: "value"},
		{name: "returned-error-host", declarations: "type Failure struct{};func(Failure)Error()string{go host();return `failure`}", write: "return 0,Failure{}", value: "7", consume: "if err!=nil {_=err.Error()}", hostDetail: "Failure.Error -> goroutine start"},
		{name: "returned-unwrap-host", declarations: "type Failure struct{};func(Failure)Error()string{return `failure`};func(Failure)Unwrap()error{go host();return nil}", write: "return 0,Failure{}", value: "7", consume: "if u,ok:=err.(interface{Unwrap()error});ok{_=u.Unwrap()}", hostDetail: "Failure.Unwrap -> goroutine start"},
		{name: "mutable-short-write-error", declarations: "type Failure struct{};func(Failure)Error()string{go host();return `short`}", setup: "", write: "return len(p)-1,nil", value: "7", consume: "if err!=nil {_=err.Error()}", hostDetail: "Failure.Error -> goroutine start"},
	} {
		for _, forwarded := range []bool{false, true} {
			name := test.name + "/direct"
			writer := "sink"
			if forwarded {
				name = test.name + "/forwarded"
				writer = "&Forward{writer:sink}"
			}
			t.Run(name, func(t *testing.T) {
				prelude := ""
				if test.name == "mutable-short-write-error" {
					prelude = "io.ErrShortWrite=Failure{};"
				}
				imports := `"io";"go.temporal.io/server/tools/gomad3/internal/canonicaljson"`
				if strings.Contains(test.write+test.setup, "bytes.") {
					imports = `"bytes";` + imports
				}
				source := `package record
import(` + imports + `)
type Value struct{Payload []byte}
type Sink struct{` + test.fields + `}
func host(){}
` + test.declarations + `
func(w *Sink)Write(p []byte)(int,error){` + test.write + `}
type Forward struct{writer io.Writer}
func(w *Forward)Write(p []byte)(int,error){return w.writer.Write(p)}
func Root()error{
 ` + prelude + test.valueSetup + `
 sink:=&Sink{` + test.setup + `}
 err:=canonicaljson.WriteCanonicalJSON(` + writer + `,` + test.value + `)
 ` + test.consume + `;return err
}
`
				analysis, _ := canonicalWriterAnalyzeRoot(t, canonicalWriterTargetFixture(t, source, ""))
				if test.hostDetail == "" && !compressionHasFinding(analysis, "unresolved-effect", "") {
					t.Fatalf("unsupported writer ownership/error case silently certified: %v", analysis.findings)
				}
				if test.hostDetail != "" && !compressionHasFinding(analysis, "host-effect", test.hostDetail) {
					t.Fatalf("ordinary writer/concrete returned error host finding lost: want=%s findings=%v", test.hostDetail, analysis.findings)
				}
			})
		}
	}
}

func TestCanonicalWriteSummaryRejectsUnknownAndReusedWriterState(t *testing.T) {
	for name, source := range map[string]string{
		"unknown": `package record
import("bytes";"io";"go.temporal.io/server/tools/gomad3/internal/canonicaljson")
var shared io.Writer
var _=bytes.Buffer{}
func Root()error{return canonicaljson.WriteCanonicalJSON(shared,7)}
`,
		"reused-bufio-with-retained-delegate-and-error": `package record
import("bufio";"io";"go.temporal.io/server/tools/gomad3/internal/canonicaljson")
type Failure struct{}
func(Failure)Error()string{return "retained failure"}
type Sink struct{}
func(*Sink)Write(p []byte)(int,error){return 0,Failure{}}
var _ io.Writer=(*Sink)(nil)
func Root()error{
 writer:=bufio.NewWriter(&Sink{})
 _,_=writer.Write([]byte{1})
 _=writer.Flush()
 return canonicaljson.WriteCanonicalJSON(writer,7)
}
`,
	} {
		t.Run(name, func(t *testing.T) {
			analysis, _ := canonicalWriterAnalyzeRoot(t, canonicalWriterTargetFixture(t, source, ""))
			if !compressionHasFinding(analysis, "unresolved-effect", "") {
				t.Fatalf("unknown/reused writer borrowed fresh constructor authority: %v", analysis.findings)
			}
		})
	}
}

func TestCanonicalWriterInterfaceAssertionsPreserveUnknownAlternatives(t *testing.T) {
	program := canonicalWriterTargetFixture(t, `package record
import("io";"go.temporal.io/server/tools/gomad3/internal/canonicaljson")
var _ io.Writer
var _=canonicaljson.WriteCanonicalJSON
func Root(err error) error { if u,ok:=err.(interface{Unwrap()error});ok {return u.Unwrap()}; return err }
`, "")
	analysis, _ := canonicalWriterAnalyzeRoot(t, program)
	root := program.Functions[program.Module+"/record.Root"]
	parameter := root.object.Type().(*types.Signature).Params().At(0)
	if actual := analysis.resolve(parameter); actual == nil || !actual.unknown {
		t.Fatalf("unknown-interface fixture did not retain its unknown parameter: %#v", actual)
	}
	if !compressionHasFinding(analysis, "unresolved-effect", "unknown dynamic callback receiver") {
		t.Fatalf("interface assertion erased unknown callback alternatives: %v", analysis.findings)
	}
}

func TestCanonicalWriteSummaryRejectsParenthesizedWriterMutations(t *testing.T) {
	for name, bridge := range map[string]string{
		"delegate-assignment": canonicalWriterWorldBridge + `
func replaceCanonicalDelegate(w *compressedBoundedWriter) { (w.writer) = &bytes.Buffer{} }
`,
		"counter-address-escape": canonicalWriterWorldBridge + `
var escapedCanonicalCounter *uint64
func escapeCanonicalCounter(w *compressedBoundedWriter) { escapedCanonicalCounter = &(w.used) }
`,
		"buffer-extra-address": strings.Replace(canonicalWriterWorldBridge,
			"var compressed bytes.Buffer", "var compressed bytes.Buffer; escaped := &(compressed); _ = escaped", 1),
	} {
		t.Run(name, func(t *testing.T) {
			program := canonicalWriterTargetFixture(t, `package record
import "go.temporal.io/server/tools/gomad3/world"
func Root() error { return world.CanonicalWriterTarget(world.Snapshot{}) }
`, bridge)
			analysis, _ := canonicalWriterAnalyzeRoot(t, program)
			if !compressionHasFinding(analysis, "unresolved-effect", "canonical write writer has no fresh constructor/delegate certificate") {
				t.Fatalf("parenthesized mutation/escape borrowed immutable writer authority: %v", analysis.findings)
			}
		})
	}
}
