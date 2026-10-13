package wasm_test

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"go.temporal.io/server/tools/gomad3/internal/archcheck"
)

const wasmModule = "go.temporal.io/server/tools/gomad3"

func wasmArchitectureConfig() archcheck.Config {
	return archcheck.Config{
		Module:      wasmModule,
		ModuleEdges: []archcheck.ModuleEdge{{From: wasmModule + "/backends/wasm", To: wasmModule + "/backends/native"}},
		PureFiles: func(path string) bool {
			return strings.HasPrefix(path, "wasi/") && strings.HasSuffix(path, ".go") &&
				!strings.HasSuffix(path, "_test.go") && path != "wasi/process.go" && path != "wasi/broker_process.go"
		},
		PureImports: []string{
			"bytes", "crypto/sha256", "embed", "encoding/binary", "encoding/hex", "encoding/json",
			"errors", "fmt", "hash", "io", "io/fs", "math", "path", "slices", "sort", "strconv",
			"strings", "sync", "syscall", "unicode/utf8",
			wasmModule + "/captured", wasmModule + "/choice", wasmModule + "/runner",
			wasmModule + "/backends", wasmModule + "/simulation/network",
			wasmModule + "/volume", wasmModule + "/world",
			wasmModule + "/internal/strictjson",
		},
	}
}

//nolint:staticcheck // Tests execute on the compiling host and require that exact reviewed Go toolchain.
func compiledGoCommand() string { return filepath.Join(runtime.GOROOT(), "bin", "go") }

func TestWASMArchitecture(t *testing.T) {
	goCommand := compiledGoCommand()
	for _, platform := range []archcheck.Platform{{OS: "darwin", Arch: "arm64"}, {OS: "linux", Arch: "amd64"}} {
		t.Run(platform.OS+"/"+platform.Arch, func(t *testing.T) {
			findings, err := archcheck.Check(".", goCommand, platform, wasmArchitectureConfig())
			if err != nil {
				t.Fatal(err)
			}
			for _, finding := range findings {
				t.Errorf("%s %s: %s", finding.Category, finding.Path, finding.Detail)
			}
		})
	}
}

func TestWASMArchitectureRejectsHostImports(t *testing.T) {
	for _, filename := range []string{"clock.go", "process.go", "moved_process.go"} {
		t.Run(filename, func(t *testing.T) {
			root := t.TempDir()
			for name, source := range map[string]string{
				"go.mod":                            "module " + wasmModule + "\n\ngo 1.27.1\n",
				"backends/wasm/wasi/environment.go": "package wasi\n",
				"backends/wasm/wasi/" + filename:    "package wasi\nimport \"os\"\nvar _ = os.Getenv\n",
			} {
				path := filepath.Join(root, name)
				if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(path, []byte(source), 0600); err != nil {
					t.Fatal(err)
				}
			}
			findings, err := archcheck.Check(filepath.Join(root, "backends/wasm"), compiledGoCommand(), archcheck.Platform{OS: "linux", Arch: "amd64"}, wasmArchitectureConfig())
			if err != nil {
				t.Fatal(err)
			}
			if filename == "process.go" {
				if len(findings) != 0 {
					t.Fatalf("transport exemption: %+v", findings)
				}
				return
			}
			if len(findings) != 1 || findings[0].Category != "pure-import" || !strings.Contains(findings[0].Path, filename) || !strings.Contains(findings[0].Detail, "os") {
				t.Fatalf("host import escaped pure boundary: %+v", findings)
			}
		})
	}
}
