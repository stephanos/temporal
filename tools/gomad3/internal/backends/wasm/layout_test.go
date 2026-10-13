package wasm

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func nativeToolchainReference(filename string, source any) (string, error) {
	file, err := parser.ParseFile(token.NewFileSet(), filename, source, 0)
	if err != nil {
		return "", err
	}
	if imported, err := nativeImport(file); err != nil || imported != "" {
		return imported, err
	}
	var reference string
	check := func(path string) {
		path = strings.ReplaceAll(path, "\\", "/")
		for _, prefix := range []string{".toolchain/" + "bin/", ".toolchain/" + "builds/", ".toolchain/" + "go"} {
			if strings.Contains(path, prefix) {
				reference = path
			}
		}
	}
	ast.Inspect(file, func(node ast.Node) bool {
		if literal, ok := node.(*ast.BasicLit); ok && literal.Kind == token.STRING {
			value, err := strconv.Unquote(literal.Value)
			if err == nil {
				check(value)
			}
		}
		if call, ok := node.(*ast.CallExpr); ok {
			if selector, ok := call.Fun.(*ast.SelectorExpr); ok && selector.Sel.Name == "Join" {
				var parts []string
				for _, argument := range call.Args {
					if literal, ok := argument.(*ast.BasicLit); ok && literal.Kind == token.STRING {
						value, err := strconv.Unquote(literal.Value)
						if err == nil {
							parts = append(parts, value)
						}
					}
				}
				check(strings.Join(parts, "/"))
			}
		}
		return reference == ""
	})
	return reference, nil
}

// nativeImport reports an import of the native backend or the pre-move native
// toolchain package.
func nativeImport(file *ast.File) (string, error) {
	for _, imported := range file.Imports {
		path, err := strconv.Unquote(imported.Path.Value)
		if err != nil {
			return "", err
		}
		for _, native := range []string{"go.temporal.io/server/tools/gomad3/toolchain", "go.temporal.io/server/tools/gomad3/internal/backends/native"} {
			if path == native || strings.HasPrefix(path, native+"/") {
				return path, nil
			}
		}
	}
	return "", nil
}

func TestWASMTestsDoNotImportNativeBuilder(t *testing.T) {
	err := filepath.WalkDir(".", func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() && (entry.Name() == ".toolchain" || entry.Name() == "wasmhost") {
			return fs.SkipDir
		}
		if !strings.HasSuffix(path, ".go") {
			return nil
		}
		var imported string
		if strings.HasSuffix(path, "_test.go") {
			imported, err = nativeToolchainReference(path, nil)
		} else {
			var file *ast.File
			file, err = parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
			if err == nil {
				imported, err = nativeImport(file)
			}
		}
		if err != nil {
			return err
		}
		if imported != "" {
			t.Errorf("WASM source %s references native backend %s", path, imported)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, source := range []string{
		"package fixture\nvar command = " + strconv.Quote(".toolchain/"+"bin/go"),
		`package fixture; var command = filepath.Join(root, ".toolchain", "bin", "go")`,
	} {
		if reference, err := nativeToolchainReference("forbidden_test.go", source); err != nil || reference == "" {
			t.Fatalf("native binary guard did not reject source %q: %q, %v", source, reference, err)
		}
	}
	for _, path := range []string{"go.temporal.io/server/tools/gomad3/toolchain", "go.temporal.io/server/tools/gomad3/internal/backends/native/toolchain", "go.temporal.io/server/tools/gomad3/internal/backends/native", "go.temporal.io/server/tools/gomad3/internal/backends/native/build"} {
		imported, err := nativeToolchainReference("forbidden_test.go", "package fixture\nimport _ "+strconv.Quote(path))
		if err != nil || imported != path {
			t.Fatalf("native builder guard did not reject %s: %q, %v", path, imported, err)
		}
	}
}

func TestWASMTargetUsesStockGo(t *testing.T) {
	root := "../../.."
	command := exec.Command("make", "--no-print-directory", "-n", "-C", root, "test-wasm", "GOMAD3_STOCK_GO=/stock/bin/go")
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("stock-Go WASM target: %v\n%s", err, output)
	}
	for _, forbidden := range []string{".toolchain/" + "bin/go", "toolchain-build", "make.bash"} {
		if strings.Contains(string(output), forbidden) {
			t.Fatalf("WASM target invokes native builder %q:\n%s", forbidden, output)
		}
	}
	if !strings.Contains(string(output), "/stock/bin/go") || !strings.Contains(string(output), "./internal/backends/wasm/...") {
		t.Fatalf("WASM target does not test the WASM subtree with stock Go:\n%s", output)
	}
}
