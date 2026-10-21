package toolchain

import (
	"bytes"
	"encoding/json"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"go.temporal.io/server/tools/gomad/internal/backends/wasm/internal/stockgo"
)

func TestOverlayPreservesRuntimeDeclarations(t *testing.T) {
	t.Parallel()
	root := stockgo.Root(t)
	overlay, _, err := BuildOverlay(root, nativeRoot(t), t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(overlay)
	if err != nil {
		t.Fatal(err)
	}
	var manifest struct{ Replace map[string]string }
	if err := json.Unmarshal(data, &manifest); err != nil {
		t.Fatal(err)
	}
	command := exec.Command(filepath.Join(root, "bin/go"), "list", "-json", "-overlay", overlay, "runtime")
	command.Env = append(os.Environ(), "GOROOT="+root, "GOOS=wasip1", "GOARCH=wasm", "GOEXPERIMENT=nogreenteagc", "CGO_ENABLED=0", "GOTOOLCHAIN=local", "GOENV=off", "GOWORK=off", "GOFLAGS=")
	listed, err := command.Output()
	if err != nil {
		t.Fatal(err)
	}
	var pkg struct {
		Dir     string
		GoFiles []string
	}
	if err := json.Unmarshal(listed, &pkg); err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, name := range pkg.GoFiles {
		path := filepath.Join(pkg.Dir, name)
		if replacement, ok := manifest.Replace[path]; ok {
			path = replacement
		}
		got = append(got, runtimeDeclarations(t, path)...)
	}
	slices.Sort(got)
	golden, err := os.ReadFile("testdata/runtime-declarations-go1.27.1.txt")
	if err != nil {
		t.Fatal(err)
	}
	want := strings.Split(strings.TrimSuffix(string(golden), "\n"), "\n")
	if !slices.Equal(got, want) {
		counts := make(map[string]int)
		for _, name := range want {
			counts[name]++
		}
		for _, name := range got {
			counts[name]--
		}
		var missing, unexpected []string
		for name, count := range counts {
			for ; count > 0; count-- {
				missing = append(missing, name)
			}
			for ; count < 0; count++ {
				unexpected = append(unexpected, name)
			}
		}
		slices.Sort(missing)
		slices.Sort(unexpected)
		t.Fatalf("runtime declarations differ: missing %v; unexpected %v", missing, unexpected)
	}
}

func TestOverlayPublishesOnlyDebugDeclarationsInInspection(t *testing.T) {
	t.Parallel()
	root := stockRoot(t)
	output := t.TempDir()
	overlay, _, err := BuildOverlay(root, nativeRoot(t), output)
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(overlay)
	if err != nil {
		t.Fatal(err)
	}
	var manifest struct{ Replace map[string]string }
	if err := json.Unmarshal(data, &manifest); err != nil {
		t.Fatal(err)
	}
	inspection := manifest.Replace[filepath.Join(root, "src/runtime/gomad_inspect_wasm.go")]
	if inspection == "" {
		t.Fatal("inspection runtime source is not published in the overlay")
	}
	for _, source := range []struct {
		path  string
		debug bool
	}{
		{manifest.Replace[filepath.Join(root, "src/runtime/gomad_wasm.go")], false},
		{inspection, true},
	} {
		data, err := os.ReadFile(source.path)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.HasPrefix(data, []byte("//go:build wasip1 && wasm\n")) {
			t.Fatalf("runtime source %s lost its build constraint", source.path)
		}
		for _, declaration := range runtimeDeclarations(t, source.path) {
			name := strings.SplitN(declaration, " ", 2)[1]
			if strings.HasPrefix(name, "gomadDebug") != source.debug {
				t.Fatalf("runtime declaration %s is in the wrong source %s", declaration, source.path)
			}
		}
	}
	if err := os.WriteFile(inspection, []byte("corrupt"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, _, err := BuildOverlay(root, nativeRoot(t), output); err == nil || !strings.Contains(err.Error(), "gomad_inspect_wasm.go") {
		t.Fatalf("changed inspection copy was reused: %v", err)
	}
}

func runtimeDeclarations(t *testing.T, path string) []string {
	t.Helper()
	file, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, declaration := range file.Decls {
		switch declaration := declaration.(type) {
		case *ast.FuncDecl:
			receiver := ""
			if declaration.Recv != nil {
				typeName := declaration.Recv.List[0].Type
				if pointer, ok := typeName.(*ast.StarExpr); ok {
					typeName = pointer.X
				}
				if name, ok := typeName.(*ast.Ident); ok {
					receiver = name.Name + "."
				}
			}
			names = append(names, "func "+receiver+declaration.Name.Name)
		case *ast.GenDecl:
			for _, spec := range declaration.Specs {
				switch spec := spec.(type) {
				case *ast.TypeSpec:
					names = append(names, declaration.Tok.String()+" "+spec.Name.Name)
				case *ast.ValueSpec:
					for _, name := range spec.Names {
						names = append(names, declaration.Tok.String()+" "+name.Name)
					}
				}
			}
		}
	}
	return names
}
