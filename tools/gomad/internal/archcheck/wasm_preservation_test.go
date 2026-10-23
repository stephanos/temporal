package archcheck_test

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

// Retained compiler-overlay support and model oracles must never select the
// removed native Go backend, even when a reference is assembled with Join.
func wasmNativeReference(filename string, source any) (string, error) {
	file, err := parser.ParseFile(token.NewFileSet(), filename, source, 0)
	if err != nil {
		return "", err
	}
	for _, imported := range file.Imports {
		path, err := strconv.Unquote(imported.Path.Value)
		if err != nil {
			return "", err
		}
		if path == modulePath+"/toolchain" || strings.HasPrefix(path, modulePath+"/toolchain/") {
			return path, nil
		}
	}
	var reference string
	check := func(path string) {
		path = strings.ReplaceAll(path, "\\", "/")
		if strings.Contains(path, "/native/") {
			reference = path
		}
		for _, prefix := range []string{".toolchain/bin/", ".toolchain/builds/", ".toolchain/go"} {
			if strings.Contains(path, prefix) {
				reference = path
			}
		}
	}
	ast.Inspect(file, func(node ast.Node) bool {
		if literal, ok := node.(*ast.BasicLit); ok && literal.Kind == token.STRING {
			if value, err := strconv.Unquote(literal.Value); err == nil {
				check(value)
			}
		}
		if call, ok := node.(*ast.CallExpr); ok {
			if selector, ok := call.Fun.(*ast.SelectorExpr); ok && selector.Sel.Name == "Join" {
				var parts []string
				for _, argument := range call.Args {
					if literal, ok := argument.(*ast.BasicLit); ok && literal.Kind == token.STRING {
						if value, err := strconv.Unquote(literal.Value); err == nil {
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

func TestRetainedWASMSourcesDoNotImportNativeBuilder(t *testing.T) {
	if err := filepath.WalkDir("internal/backends/wasm", func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !entry.IsDir() && strings.HasSuffix(path, ".go") {
			reference, err := wasmNativeReference(path, nil)
			if err != nil {
				return err
			}
			if reference != "" {
				t.Errorf("WASM source %s references native backend %s", path, reference)
			}
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	for _, source := range []string{
		"package fixture; var command = " + strconv.Quote(".toolchain/"+"bin/go"),
		`package fixture; var command = filepath.Join(root, ".toolchain", "bin", "go")`,
		`package fixture; var fixture = "../../native/conformance/testdata"`,
		`package fixture; var fixture = filepath.Join("..", "..", "native", "conformance", "testdata")`,
		`package fixture; var command = ".toolchain\\bin\\go"`,
		"package fixture; import _ " + strconv.Quote(modulePath+"/toolchain"),
		"package fixture; import _ " + strconv.Quote(modulePath+"/toolchain/build"),
	} {
		if reference, err := wasmNativeReference("forbidden_test.go", source); err != nil || reference == "" {
			t.Fatalf("native builder guard admitted %q: %q, %v", source, reference, err)
		}
	}
}

func TestNativeWASMTargetRetainsStockGoOracleSelection(t *testing.T) {
	command := exec.CommandContext(t.Context(), "make", "--no-print-directory", "-n", "test-wasm", "GOMAD_STOCK_GO=/stock/bin/go")
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("stock-Go WASM target: %v\n%s", err, output)
	}
	text := string(output)
	for _, forbidden := range []string{".toolchain/" + "bin/go", "toolchain-build", "make.bash"} {
		if strings.Contains(text, forbidden) {
			t.Errorf("WASM target invokes removed native builder %q", forbidden)
		}
	}
	for _, required := range []string{"/stock/bin/go", "cargo +1.99.1 test", "-p gomad", "./internal/backends/wasm/toolchain/..."} {
		if !strings.Contains(text, required) {
			t.Errorf("WASM target lost native engine or retained stock-Go oracle selection %q", required)
		}
	}
}
