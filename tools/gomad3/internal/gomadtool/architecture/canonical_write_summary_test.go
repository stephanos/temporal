package architecture

import (
	"go/ast"
	"go/build"
	"go/token"
	"go/types"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const canonicalWritePackage = "go.temporal.io/server/tools/gomad3/internal/canonicaljson"

// Import the real encoders' complete production inventory and stock dependency
// sources. A separate directory lets hostile witness tests change their copies.
func canonicalWriteFixture(t *testing.T, source string) *Program {
	t.Helper()
	return canonicalWriteFixtureBeforeImport(t, source, nil)
}

func canonicalWriteFixtureBeforeImport(t *testing.T, source string, change func(*Program)) *Program {
	t.Helper()
	program := compressionStockFixture(t)
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
			t.Fatalf("nonstock canonical dependency: %s", path)
		}
		program.Metadata[path] = Package{ImportPath: path, Dir: selected.Dir, Standard: true,
			GoFiles: selected.GoFiles, Imports: selected.Imports, SFiles: selected.SFiles, HFiles: selected.HFiles}
		for _, dependency := range selected.Imports {
			discover(dependency)
		}
	}
	selected, err := context.ImportDir("../../canonicaljson", 0)
	if err != nil {
		t.Fatal(err)
	}
	directory := t.TempDir()
	for _, name := range selected.GoFiles {
		data, err := os.ReadFile(filepath.Join(selected.Dir, name))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(directory, name), data, 0600); err != nil {
			t.Fatal(err)
		}
	}
	program.Metadata[canonicalWritePackage] = Package{ImportPath: canonicalWritePackage, Dir: directory,
		GoFiles: selected.GoFiles, Imports: selected.Imports}
	for _, dependency := range selected.Imports {
		discover(dependency)
	}
	discover("bytes")
	record := filepath.Join(t.TempDir(), "record")
	if err := os.Mkdir(record, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(record, "pure.go"), []byte(source), 0600); err != nil {
		t.Fatal(err)
	}
	path := program.Module + "/record"
	program.Metadata[path] = Package{ImportPath: path, Dir: record, GoFiles: []string{"pure.go"},
		Imports: []string{"bytes", canonicalWritePackage}}
	if change != nil {
		change(program)
	}
	if _, err := program.Import(path); err != nil {
		t.Fatal(err)
	}
	return program
}

func canonicalWriteCall(t *testing.T, program *Program) (*effectAnalysis, *types.Func, []*abstractValue, token.Pos) {
	t.Helper()
	analysis := effectResolverAnalysis(program)
	root := program.Functions[program.Module+"/record.Root"]
	var call *ast.CallExpr
	ast.Inspect(root.declaration.Body, func(node ast.Node) bool {
		if candidate, ok := node.(*ast.CallExpr); ok && functionID(calledObject(root.pkg.Info, candidate.Fun)) == canonicalWritePackage+".WriteCanonicalJSON" {
			call = candidate
		}
		return true
	})
	if call == nil {
		t.Fatal("fixture did not contain canonical write call")
	}
	fn := calledObject(root.pkg.Info, call.Fun)
	args := []*abstractValue{analysis.eval(root.pkg, call.Args[0], environment{}), analysis.eval(root.pkg, call.Args[1], environment{})}
	return analysis, fn, args, call.Pos()
}

const canonicalWriteScalarSource = `package record
import ("bytes"; "go.temporal.io/server/tools/gomad3/internal/canonicaljson")
func Root() error { return canonicaljson.WriteCanonicalJSON(&bytes.Buffer{}, 7) }
`

func TestCanonicalWriteSummaryQualifiedBoundary(t *testing.T) {
	program := canonicalWriteFixture(t, canonicalWriteScalarSource)
	analysis, fn, args, pos := canonicalWriteCall(t, program)
	result, handled := analysis.standard(fn, nil, args, pos)
	if !handled {
		t.Fatal("source-qualified canonical write boundary is absent from production standard routing")
	}
	if result == nil || result.unknown || len(analysis.findings) != 0 {
		t.Fatalf("fresh buffer and callback-free scalar boundary rejected: result=%#v findings=%v", result, analysis.findings)
	}
}

func TestCanonicalWriteSummaryRejectsWitnessChanges(t *testing.T) {
	for _, name := range []string{"source-before-import", "source-after-import", "source-package", "info", "files", "declaration", "function", "nil-function", "selected-files", "imports", "redirect", "standard", "initializer", "body-in-place", "initializer-in-place", "import-in-place", "uses", "defs", "types", "cached-body-in-place", "cached-uses", "cached-defs", "bufio-body-in-place", "bufio-source", "bufio-binding"} {
		t.Run(name, func(t *testing.T) {
			var program *Program
			if name == "source-before-import" {
				program = canonicalWriteFixtureBeforeImport(t, canonicalWriteScalarSource, func(program *Program) {
					filename := filepath.Join(program.Metadata[canonicalWritePackage].Dir, "streaming.go")
					data, err := os.ReadFile(filename)
					if err != nil {
						t.Fatal(err)
					}
					if err := os.WriteFile(filename, append(data, []byte("\nfunc Changed() {}\n")...), 0600); err != nil {
						t.Fatal(err)
					}
				})
			} else {
				program = canonicalWriteFixture(t, canonicalWriteScalarSource)
			}
			analysis, fn, args, pos := canonicalWriteCall(t, program)
			if strings.HasPrefix(name, "cached-") {
				if _, handled := analysis.standard(fn, nil, args, pos); !handled || len(analysis.findings) != 0 {
					t.Fatal("unchanged witness did not qualify before cached tamper")
				}
			}
			pkg := program.Packages[canonicalWritePackage]
			switch strings.TrimPrefix(name, "cached-") {
			case "source-after-import":
				filename := filepath.Join(pkg.Metadata.Dir, "streaming.go")
				if err := os.WriteFile(filename, []byte("package canonicaljson\n"), 0600); err != nil {
					t.Fatal(err)
				}
			case "source-package":
				copy := *pkg
				program.Packages[canonicalWritePackage] = &copy
			case "info":
				copy := *pkg.Info
				pkg.Info = &copy
			case "files":
				pkg.Files = pkg.Files[:len(pkg.Files)-1]
			case "declaration":
				decl := program.Functions[functionID(fn)]
				copy := *decl.declaration
				decl.declaration = &copy
			case "function":
				fn = types.NewFunc(fn.Pos(), fn.Pkg(), fn.Name(), fn.Type().(*types.Signature))
			case "nil-function":
				if analysis.checkCanonicalSummarySource(nil, pos) || len(analysis.findings) == 0 {
					t.Fatal("missing function identity did not fail closed")
				}
				return
			case "selected-files":
				metadata := program.Metadata[canonicalWritePackage]
				metadata.GoFiles = append(append([]string{}, metadata.GoFiles...), "extra.go")
				program.Metadata[canonicalWritePackage] = metadata
			case "imports":
				metadata := program.Metadata[canonicalWritePackage]
				metadata.Imports = nil
				program.Metadata[canonicalWritePackage] = metadata
			case "redirect":
				metadata := program.Metadata[canonicalWritePackage]
				metadata.ImportMap = map[string]string{"bufio": "example.invalid/foreign"}
				program.Metadata[canonicalWritePackage] = metadata
			case "standard":
				pkg.Metadata.Standard = true
			case "initializer":
				if len(pkg.Info.InitOrder) == 0 {
					t.Fatal("actual canonical package has no initializer witness")
				}
				pkg.Info.InitOrder = nil
			case "body-in-place":
				program.Functions[functionID(fn)].declaration.Body.List = nil
			case "initializer-in-place":
				if len(pkg.Info.InitOrder) == 0 {
					t.Fatal("actual canonical package has no initializer witness")
				}
				ast.Inspect(pkg.Info.InitOrder[0].Rhs, func(node ast.Node) bool {
					if identifier, ok := node.(*ast.Ident); ok {
						identifier.Name = "forged"
					}
					return true
				})
			case "import-in-place":
				pkg.Files[0].Imports[0].Path.Value = `"unsafe"`
			case "uses":
				for identifier, original := range pkg.Info.Uses {
					if original == types.Universe.Lookup("nil") {
						continue
					}
					pkg.Info.Uses[identifier] = types.Universe.Lookup("nil")
					break
				}
			case "defs":
				for identifier := range pkg.Info.Defs {
					pkg.Info.Defs[identifier] = types.Universe.Lookup("nil")
					break
				}
			case "types":
				for expression, original := range pkg.Info.Types {
					if original == (types.TypeAndValue{}) {
						continue
					}
					pkg.Info.Types[expression] = types.TypeAndValue{}
					break
				}
			case "bufio-body-in-place":
				program.Functions["bufio.*Writer.Flush"].declaration.Body.List = nil
			case "bufio-source":
				metadata := program.Metadata["bufio"]
				metadata.Dir = t.TempDir()
				program.Metadata["bufio"] = metadata
			case "bufio-binding":
				program.Packages["bufio"].Types = types.NewPackage("bufio", "bufio")
			}
			if _, handled := analysis.standard(fn, nil, args, pos); !handled || len(analysis.findings) == 0 {
				t.Fatal("forged canonical boundary witness accepted or silently fell through")
			}
		})
	}
}

func TestCanonicalWriteSummaryScalarHooks(t *testing.T) {
	for _, test := range []struct {
		name, declaration, value string
		pure                     bool
	}{
		{"uint-helper", "type Value uint64;func marshal(v uint64)[]byte{return []byte(strconv.Quote(strconv.FormatUint(v,10)))};func(v Value)MarshalJSON()([]byte,error){return marshal(uint64(v)),nil}", "Value(7)", true},
		{"signed", "type Value int64;func(v Value)MarshalJSON()([]byte,error){return []byte(strconv.Quote(strconv.FormatInt(int64(v),10))),nil}", "Value(-7)", true},
		{"text", "type Value uint64;func(v Value)MarshalText()([]byte,error){return []byte(strconv.FormatUint(uint64(v),10)),nil}", "Value(7)", true},
		{"global", "var bytesValue=[]byte(`7`);type Value uint64;func(Value)MarshalJSON()([]byte,error){return bytesValue,nil}", "Value(7)", false},
		{"assignment", "type Value uint64;func(v Value)MarshalJSON()([]byte,error){v++;return []byte(strconv.FormatUint(uint64(v),10)),nil}", "Value(7)", false},
		{"pointer-hook", "type Value uint64;func(v *Value)MarshalJSON()([]byte,error){return []byte(strconv.FormatUint(uint64(*v),10)),nil}", "new(Value)", false},
		{"error", "type Failure int;func(Failure)Error()string{return `dirty`};type Value uint64;func(Value)MarshalJSON()([]byte,error){return nil,Failure(1)}", "Value(7)", false},
		{"recursive", "type Value uint64;func marshal(v uint64)[]byte{return marshal(v)};func(v Value)MarshalJSON()([]byte,error){return marshal(uint64(v)),nil}", "Value(7)", false},
		{"closure", "type Value uint64;func(v Value)MarshalJSON()([]byte,error){return func()[]byte{return []byte(strconv.FormatUint(uint64(v),10))}(),nil}", "Value(7)", false},
		{"dynamic-field", "type Value struct{Dynamic any}", "Value{Dynamic:uint64(7)}", false},
		{"iszero-host", "type Value uint64;func(Value)IsZero()bool{os.Getwd();return false};type Holder struct{Value Value `json:\"value,omitzero\"`}", "Holder{Value:7}", false},
		{"iszero-scalar", "type Value uint64;func(v Value)IsZero()bool{return v==0};type Holder struct{Value Value `json:\"value,omitzero\"`}", "Holder{Value:7}", true},
	} {
		t.Run(test.name, func(t *testing.T) {
			program := canonicalWriteFixture(t, "package record\nimport(\"bytes\";\"strconv\";\"os\";\"go.temporal.io/server/tools/gomad3/internal/canonicaljson\")\nvar _=strconv.FormatUint;var _=os.Getwd\n"+test.declaration+"\nfunc Root()error{return canonicaljson.WriteCanonicalJSON(&bytes.Buffer{},"+test.value+")}\n")
			analysis, fn, args, pos := canonicalWriteCall(t, program)
			_, handled := analysis.standard(fn, nil, args, pos)
			if !handled || (len(analysis.findings) == 0) != test.pure {
				t.Fatalf("scalar hook certificate pure=%v handled=%v findings=%v", test.pure, handled, analysis.findings)
			}
		})
	}
}

func TestCanonicalWriteSummaryRejectsUnprovenGraphs(t *testing.T) {
	for _, name := range []string{"writer-unknown", "writer-builtin", "writer-origin", "writer-pointee-origin", "writer-dirty-state", "writer-nil", "value-unknown", "value-nil-type", "value-function"} {
		t.Run(name, func(t *testing.T) {
			program := canonicalWriteFixture(t, canonicalWriteScalarSource)
			analysis, fn, args, pos := canonicalWriteCall(t, program)
			switch name {
			case "writer-unknown":
				args[0].unknown = true
			case "writer-builtin":
				args[0].builtin = true
			case "writer-origin":
				args[0].origin = token.NoPos
			case "writer-pointee-origin":
				args[0].fields["$pointee"].origin = token.NoPos
			case "writer-dirty-state":
				args[0].fields["$pointee"].fields["buf"].unknown = true
			case "writer-nil":
				args[0] = nil
			case "value-unknown":
				args[1].unknown = true
			case "value-nil-type":
				args[1].types = []types.Type{nil}
			case "value-function":
				args[1].functions = []*boundFunction{{}}
			}
			if _, handled := analysis.standard(fn, nil, args, pos); !handled || len(analysis.findings) == 0 {
				t.Fatalf("unproven graph was accepted: handled=%v findings=%v", handled, analysis.findings)
			}
		})
	}
}

func TestCanonicalWriteSummaryRejectsScalarSourceForgery(t *testing.T) {
	for _, name := range []string{"body", "uses", "types", "declaration-copy", "function-entry"} {
		t.Run(name, func(t *testing.T) {
			program := canonicalWriteFixture(t, "package record\nimport(\"bytes\";\"strconv\";\"go.temporal.io/server/tools/gomad3/internal/canonicaljson\")\ntype Value uint64\nfunc(v Value)MarshalJSON()([]byte,error){return []byte(strconv.FormatUint(uint64(v),10)),nil}\nfunc Root()error{return canonicaljson.WriteCanonicalJSON(&bytes.Buffer{},Value(7))}\n")
			analysis, fn, args, pos := canonicalWriteCall(t, program)
			if _, handled := analysis.standard(fn, nil, args, pos); !handled || len(analysis.findings) != 0 {
				t.Fatal("unchanged scalar witness did not qualify before tamper")
			}
			pkg := program.Packages[program.Module+"/record"]
			switch name {
			case "body":
				program.Functions[program.Module+"/record.Value.MarshalJSON"].declaration.Body.List = nil
			case "uses":
				mutated := false
				for identifier, original := range pkg.Info.Uses {
					if identifier.Name == "FormatUint" {
						replacement := types.Universe.Lookup("nil")
						if original == replacement {
							t.Fatal("scalar Uses fixture was already nil before tamper")
						}
						pkg.Info.Uses[identifier] = replacement
						mutated = true
					}
				}
				if !mutated {
					t.Fatal("scalar Uses fixture did not find FormatUint")
				}
			case "types":
				mutated := false
				for expression, original := range pkg.Info.Types {
					if literal, ok := expression.(*ast.BasicLit); ok && literal.Value == "10" {
						if original == (types.TypeAndValue{}) {
							t.Fatal("scalar Types fixture was already empty before tamper")
						}
						pkg.Info.Types[expression] = types.TypeAndValue{}
						mutated = true
					}
				}
				if !mutated {
					t.Fatal("scalar Types fixture did not find the radix literal")
				}
			case "declaration-copy", "function-entry":
				id := program.Module + "/record.Value.MarshalJSON"
				original := program.Functions[id]
				copy := *original.declaration
				if name == "declaration-copy" {
					original.declaration = &copy
				} else {
					program.Functions[id] = &function{object: original.object, declaration: &copy, pkg: original.pkg}
				}
			}
			analysis.standard(fn, nil, args, pos)
			if len(analysis.findings) == 0 {
				t.Fatal("changed scalar parsed source/type-info retained purity authority")
			}
		})
	}
}

func TestCanonicalWriteSummaryRejectsMutableBufferError(t *testing.T) {
	for _, change := range []string{"bytes.ErrTooLarge=Failure(1)", "_ = &bytes.ErrTooLarge"} {
		t.Run(change, func(t *testing.T) {
			program := canonicalWriteFixture(t, "package record\nimport(\"bytes\";\"os\";\"go.temporal.io/server/tools/gomad3/internal/canonicaljson\")\ntype Failure int\nfunc(Failure)Error()string{os.Getwd();return `dirty`}\nfunc Change(){"+change+"}\nfunc Root()error{return canonicaljson.WriteCanonicalJSON(&bytes.Buffer{},[]string{`large`})}\n")
			analysis, fn, args, pos := canonicalWriteCall(t, program)
			if _, handled := analysis.standard(fn, nil, args, pos); !handled || len(analysis.findings) == 0 {
				t.Fatal("application buffer error mutation/escape retained primitive authority")
			}
		})
	}
}
