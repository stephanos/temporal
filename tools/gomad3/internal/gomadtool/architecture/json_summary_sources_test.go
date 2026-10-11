package architecture

import (
	"crypto/sha256"
	"fmt"
	"go/token"
	"go/types"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

type jsonSourceChecker interface {
	checkJSONSummarySource(*types.Func, token.Pos) bool
}

func TestJSONSummarySourceRequiresImportedFunction(t *testing.T) {
	program := &Program{Files: token.NewFileSet(), Metadata: map[string]Package{}, Packages: map[string]*SourcePackage{}, Functions: map[string]*function{}}
	analysis := summarySourceAnalysis(program)
	checker, ok := any(analysis).(jsonSourceChecker)
	if !ok {
		t.Fatal("JSON callback source qualification is unavailable")
	}
	for _, fn := range []*types.Func{nil, types.NewFunc(token.NoPos, types.NewPackage("encoding/json", "json"), "Marshal", types.NewSignatureType(nil, nil, nil, nil, nil, false))} {
		if checker.checkJSONSummarySource(fn, token.NoPos) {
			t.Fatal("JSON summary accepted without an imported declaration")
		}
	}
	if len(analysis.findings) == 0 {
		t.Fatal("source rejection lost its conservative finding")
	}
}

func jsonProfileFixture(t *testing.T) (*Program, *types.Func, map[string]string) {
	t.Helper()
	return jsonProfileFixtureWithMetadata(t, nil)
}

func jsonProfileFixtureWithMetadata(t *testing.T, change func(*Program)) (*Program, *types.Func, map[string]string) {
	t.Helper()
	files := map[string][]string{
		"encoding/json":                    {"v2_encode.go", "v2_decode.go", "v2_stream.go", "v2_options.go", "v2_inject.go", "v2_indent.go", "v2_scanner.go"},
		"encoding/json/v2":                 {"arshal.go", "arshal_any.go", "arshal_default.go", "arshal_embedded.go", "arshal_funcs.go", "arshal_methods.go", "arshal_time.go", "doc.go", "errors.go", "fields.go", "fold.go", "intern.go", "options.go"},
		"encoding/json/jsontext":           {"decode.go", "doc.go", "encode.go", "errors.go", "export.go", "options.go", "pools.go", "quote.go", "state.go", "token.go", "value.go"},
		"encoding/json/internal":           {"internal.go"},
		"encoding/json/internal/jsonflags": {"flags.go"},
		"encoding/json/internal/jsonopts":  {"options.go", "options_format.go"},
		"encoding/json/internal/jsonwire":  {"decode.go", "encode.go", "wire.go"},
	}
	previous := jsonSummarySourcePins
	jsonSummarySourcePins = map[string]map[string]string{}
	t.Cleanup(func() { jsonSummarySourcePins = previous })
	root := t.TempDir()
	program := &Program{Files: token.NewFileSet(), Metadata: map[string]Package{}, Packages: map[string]*SourcePackage{}, Functions: map[string]*function{}}
	filenames := map[string]string{}
	var imports []string
	for path := range files {
		if path != "encoding/json" {
			imports = append(imports, path)
		}
	}
	sort.Strings(imports)
	for path, names := range files {
		directory := filepath.Join(root, path)
		if err := os.MkdirAll(directory, 0700); err != nil {
			t.Fatal(err)
		}
		metadata := Package{ImportPath: path, Dir: directory, Standard: true, GoFiles: append([]string(nil), names...)}
		jsonSummarySourcePins[path] = map[string]string{}
		for i, name := range names {
			source := "package " + filepath.Base(path) + "\n"
			if i == 0 && path != "encoding/json" {
				source += "const Marker = 0\n"
				if path == "encoding/json/jsontext" {
					source += "type Encoder struct{}\ntype Decoder struct{}\ntype Value []byte\nfunc (Value) MarshalJSON()([]byte,error){return nil,nil}\nfunc (*Value) UnmarshalJSON([]byte)error{return nil}\n"
				}
			}
			if i == 0 && path == "encoding/json" {
				for index, dependency := range imports {
					source += fmt.Sprintf("import dependency%d %q\n", index, dependency)
				}
				for index := range imports {
					source += fmt.Sprintf("var _ = dependency%d.Marker\n", index)
					if imports[index] == "encoding/json/jsontext" {
						source += fmt.Sprintf("type RawMessage = dependency%d.Value\n", index)
						source += fmt.Sprintf("type Number string\nfunc (Number) MarshalJSONTo(*dependency%d.Encoder)error{return nil}\nfunc (*Number) UnmarshalJSONFrom(*dependency%d.Decoder)error{return nil}\n", index, index)
					}
				}
				source += "func Marshal(any) ([]byte, error) { return nil, nil }\n"
				metadata.Imports = append([]string(nil), imports...)
			}
			filename := filepath.Join(directory, name)
			if err := os.WriteFile(filename, []byte(source), 0600); err != nil {
				t.Fatal(err)
			}
			filenames[path+"/"+name] = filename
			jsonSummarySourcePins[path][name] = fmt.Sprintf("%x", sha256.Sum256([]byte(source)))
		}
		program.Metadata[path] = metadata
	}
	if change != nil {
		change(program)
	}
	if _, err := program.Import("encoding/json"); err != nil {
		t.Fatal(err)
	}
	return program, program.Packages["encoding/json"].Types.Scope().Lookup("Marshal").(*types.Func), filenames
}

func TestJSONSummarySourceQualifiesNativeAliasOrigin(t *testing.T) {
	program, _, _ := jsonProfileFixture(t)
	alias := program.Packages["encoding/json"].Types.Scope().Lookup("RawMessage").Type()
	for _, name := range []string{"MarshalJSON", "UnmarshalJSON"} {
		fn := types.NewMethodSet(types.NewPointer(alias)).Lookup(nil, name).Obj().(*types.Func)
		analysis := summarySourceAnalysis(program)
		if !analysis.checkJSONSummarySource(fn, token.NoPos) || len(analysis.findings) != 0 {
			t.Fatalf("native alias declaration %s rejected: %v", functionID(fn), analysis.findings)
		}
		foreign := types.NewFunc(token.NoPos, fn.Pkg(), fn.Name(), fn.Type().(*types.Signature))
		if analysis.checkJSONSummarySource(foreign, token.NoPos) || len(analysis.findings) == 0 {
			t.Fatal("foreign alias declaration object accepted")
		}
	}
}

func TestJSONSummarySourceRejectsRedirectedNativeImport(t *testing.T) {
	program, fn, _ := jsonProfileFixtureWithMetadata(t, func(program *Program) {
		if _, err := program.Import("encoding/json/v2"); err != nil {
			t.Fatal(err)
		}
		directory := t.TempDir()
		if err := os.WriteFile(filepath.Join(directory, "foreign.go"), []byte("package v2\nconst Marker = 0\n"), 0600); err != nil {
			t.Fatal(err)
		}
		program.Metadata["foreign.invalid/v2"] = Package{ImportPath: "foreign.invalid/v2", Dir: directory, GoFiles: []string{"foreign.go"}}
		metadata := program.Metadata["encoding/json"]
		metadata.ImportMap = map[string]string{"encoding/json/v2": "foreign.invalid/v2"}
		program.Metadata["encoding/json"] = metadata
	})
	analysis := summarySourceAnalysis(program)
	if analysis.checkJSONSummarySource(fn, token.NoPos) || len(analysis.findings) == 0 {
		t.Fatal("JSON source profile accepted a foreign imported implementation")
	}
}

func TestJSONSummarySourceRejectsEachMissingFile(t *testing.T) {
	program, fn, filenames := jsonProfileFixture(t)
	for path, filename := range filenames {
		t.Run(path, func(t *testing.T) {
			original, err := os.ReadFile(filename)
			if err != nil {
				t.Fatal(err)
			}
			if err := os.Remove(filename); err != nil {
				t.Fatal(err)
			}
			analysis := summarySourceAnalysis(program)
			if analysis.checkJSONSummarySource(fn, token.NoPos) || len(analysis.findings) == 0 {
				t.Fatal("missing transitive JSON implementation accepted")
			}
			if err := os.WriteFile(filename, original, 0600); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestJSONSummarySourceAcceptsParsedProfile(t *testing.T) {
	program, fn, _ := jsonProfileFixture(t)
	analysis := summarySourceAnalysis(program)
	if !analysis.checkJSONSummarySource(fn, token.NoPos) || len(analysis.findings) != 0 {
		t.Fatalf("qualified parsed JSON profile rejected: %v", analysis.findings)
	}
}

func TestJSONSummarySourceRejectsEachChangedFile(t *testing.T) {
	program, fn, filenames := jsonProfileFixture(t)
	for path, filename := range filenames {
		t.Run(path, func(t *testing.T) {
			original, err := os.ReadFile(filename)
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filename, append(append([]byte(nil), original...), []byte("\n// changed\n")...), 0600); err != nil {
				t.Fatal(err)
			}
			analysis := summarySourceAnalysis(program)
			if analysis.checkJSONSummarySource(fn, token.NoPos) || len(analysis.findings) == 0 {
				t.Fatal("changed transitive JSON implementation accepted")
			}
			if err := os.WriteFile(filename, original, 0600); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestJSONSummarySourceRejectsSelectedManifestMutation(t *testing.T) {
	for _, change := range []string{"missing", "extra", "legacy", "mixed", "unknown"} {
		t.Run(change, func(t *testing.T) {
			program, fn, _ := jsonProfileFixture(t)
			metadata := program.Metadata["encoding/json"]
			switch change {
			case "missing":
				metadata.GoFiles = metadata.GoFiles[1:]
			case "extra", "unknown":
				metadata.GoFiles = append(metadata.GoFiles, change+".go")
			case "legacy":
				metadata.GoFiles = []string{"encode.go", "decode.go", "stream.go"}
			case "mixed":
				metadata.GoFiles = append(metadata.GoFiles, "encode.go")
			}
			program.Metadata["encoding/json"] = metadata
			analysis := summarySourceAnalysis(program)
			if analysis.checkJSONSummarySource(fn, token.NoPos) || len(analysis.findings) == 0 {
				t.Fatalf("%s selected profile accepted", change)
			}
		})
	}
}

func TestJSONSummarySourceAnalysisSnapshotLifetime(t *testing.T) {
	program, fn, filenames := jsonProfileFixture(t)
	analysis := summarySourceAnalysis(program)
	if !analysis.checkJSONSummarySource(fn, token.NoPos) {
		t.Fatal("qualified initial profile rejected")
	}
	filename := filenames["encoding/json/v2/arshal_methods.go"]
	if err := os.WriteFile(filename, []byte("package v2\n// changed\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if !analysis.checkJSONSummarySource(fn, token.NoPos) {
		t.Fatal("admitted immutable analysis lost its parsed profile")
	}
	fresh := summarySourceAnalysis(program)
	if fresh.checkJSONSummarySource(fn, token.NoPos) || len(fresh.findings) == 0 {
		t.Fatal("new analysis reused stale disk qualification")
	}
}

func TestJSONSummarySourceFailureIsNotCached(t *testing.T) {
	program, fn, filenames := jsonProfileFixture(t)
	filename := filenames["encoding/json/v2/arshal_methods.go"]
	original, err := os.ReadFile(filename)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filename, []byte(strings.ReplaceAll(string(original), "package v2", "package other")), 0600); err != nil {
		t.Fatal(err)
	}
	analysis := summarySourceAnalysis(program)
	if analysis.checkJSONSummarySource(fn, token.NoPos) {
		t.Fatal("changed profile accepted")
	}
	if err := os.WriteFile(filename, original, 0600); err != nil {
		t.Fatal(err)
	}
	if !analysis.checkJSONSummarySource(fn, token.NoPos) {
		t.Fatal("failed source validation was cached")
	}
}

func TestJSONSummarySourceCachedSuccessRequiresCurrentBinding(t *testing.T) {
	for _, change := range []string{"function", "forged-function-table", "package", "types", "info", "files", "native-imports", "metadata", "source-metadata", "selected-files", "compiled-files", "imports", "import-map", "nonstandard", "pin", "unparsed", "snapshot", "directory"} {
		t.Run(change, func(t *testing.T) {
			program, fn, _ := jsonProfileFixture(t)
			analysis := summarySourceAnalysis(program)
			if !analysis.checkJSONSummarySource(fn, token.NoPos) {
				t.Fatal("initial profile was not qualified")
			}
			pkg := program.Packages["encoding/json"]
			metadata := program.Metadata["encoding/json"]
			switch change {
			case "function", "forged-function-table":
				fn = types.NewFunc(token.NoPos, pkg.Types, "Marshal", fn.Type().(*types.Signature))
				if change == "forged-function-table" {
					copy := *program.Functions[functionID(fn)]
					copy.object = fn
					program.Functions[functionID(fn)] = &copy
				}
			case "package":
				copy := *pkg
				program.Packages["encoding/json"] = &copy
			case "types":
				pkg.Types = types.NewPackage("encoding/json", "json")
			case "info":
				pkg.Info = &types.Info{}
			case "files":
				pkg.Files = pkg.Files[1:]
			case "native-imports":
				pkg.Types.SetImports([]*types.Package{types.NewPackage("encoding/json/v2", "v2")})
			case "metadata":
				metadata.ImportPath = "foreign.invalid/json"
			case "source-metadata":
				pkg.Metadata.Standard = false
			case "selected-files":
				metadata.GoFiles[0] = "foreign.go"
			case "compiled-files":
				metadata.CompiledGoFiles = []string{"foreign.go"}
			case "imports":
				metadata.Imports[0] = "foreign.invalid/json"
			case "import-map":
				metadata.ImportMap = map[string]string{"encoding/json/v2": "foreign.invalid/json"}
			case "nonstandard":
				metadata.Standard = false
			case "pin":
				jsonSummarySourcePins["encoding/json"]["v2_encode.go"] = "changed"
			case "unparsed":
				delete(program.jsonSources["encoding/json"].digests, "v2_encode.go")
			case "snapshot":
				delete(program.jsonSources, "encoding/json")
			case "directory":
				metadata.Dir = t.TempDir()
			}
			program.Metadata["encoding/json"] = metadata
			if analysis.checkJSONSummarySource(fn, token.NoPos) || len(analysis.findings) == 0 {
				t.Fatalf("%s mutation reused a qualified profile", change)
			}
		})
	}
}

func TestJSONSummarySourceCachedSuccessRequiresEveryPackage(t *testing.T) {
	for _, path := range []string{"encoding/json", "encoding/json/v2", "encoding/json/jsontext", "encoding/json/internal", "encoding/json/internal/jsonflags", "encoding/json/internal/jsonopts", "encoding/json/internal/jsonwire"} {
		for _, change := range []string{"package", "types", "source-selection"} {
			t.Run(path+"/"+change, func(t *testing.T) {
				program, fn, _ := jsonProfileFixture(t)
				analysis := summarySourceAnalysis(program)
				if !analysis.checkJSONSummarySource(fn, token.NoPos) {
					t.Fatal("initial profile was not qualified")
				}
				switch change {
				case "package":
					copy := *program.Packages[path]
					program.Packages[path] = &copy
				case "types":
					program.Packages[path].Types = types.NewPackage(path, filepath.Base(path))
				case "source-selection":
					program.Packages[path].Metadata.GoFiles[0] = "unknown.go"
				}
				if analysis.checkJSONSummarySource(fn, token.NoPos) || len(analysis.findings) == 0 {
					t.Fatalf("%s binding mutation reused a qualified profile", path)
				}
			})
		}
	}
}
