package wasm_test

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"

	"go.temporal.io/server/tools/gomad/internal/archcheck"
	"go.temporal.io/server/tools/gomad/internal/backends/wasm/determinism"
)

const wasmModule = "go.temporal.io/server/tools/gomad"

func wasmArchitectureConfig() archcheck.Config {
	return archcheck.Config{
		Module:      wasmModule,
		PureFiles: func(path string) bool {
			return strings.HasPrefix(path, "wasi/") && strings.HasSuffix(path, ".go") &&
				!strings.HasSuffix(path, "_test.go") && path != "wasi/process.go" && path != "wasi/broker_process.go" &&
				// The one-helper cluster session is a host process transport like process.go.
				path != "wasi/cluster.go"
		},
		PureImports: []string{
			"bytes", "compress/gzip", "crypto/sha256", "embed", "encoding/binary", "encoding/hex", "encoding/json",
			"errors", "fmt", "hash", "io", "io/fs", "math", "path", "slices", "sort", "strconv",
			"strings", "sync", "syscall", "unicode/utf8",
			wasmModule + "/internal/evidence/artifact/capture", wasmModule + "/internal/model/choice", wasmModule + "/internal/runner",
			wasmModule + "/internal/backends", wasmModule + "/internal/model/network",
			wasmModule + "/internal/model/clock",
			wasmModule + "/internal/model/volume", wasmModule + "/internal/model/world",
			wasmModule + "/internal/model/strictjson",
			wasmModule + "/internal/evidence/record/softassert",
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
	for _, filename := range []string{"clock.go", "debug_output.go", "measurement.go", "process.go", "cluster.go", "moved_process.go"} {
		t.Run(filename, func(t *testing.T) {
			root := t.TempDir()
			for name, source := range map[string]string{
				"go.mod": "module " + wasmModule + "\n\ngo 1.27.1\n",
				"internal/backends/wasm/wasi/environment.go": "package wasi\n",
				"internal/backends/wasm/wasi/" + filename:    "package wasi\nimport \"os\"\nvar _ = os.Getenv\n",
			} {
				path := filepath.Join(root, name)
				if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(path, []byte(source), 0600); err != nil {
					t.Fatal(err)
				}
			}
			findings, err := archcheck.Check(filepath.Join(root, "internal/backends/wasm"), compiledGoCommand(), archcheck.Platform{OS: "linux", Arch: "amd64"}, wasmArchitectureConfig())
			if err != nil {
				t.Fatal(err)
			}
			if filename == "process.go" || filename == "cluster.go" {
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

// wasmHostFile selects the Go host packages whose host effects the
// determinism inventory reviews site by site.
func wasmHostFile(path string) bool {
	return (strings.HasPrefix(path, "wasi/") || strings.HasPrefix(path, "cluster/") || strings.HasPrefix(path, "backend/")) &&
		strings.HasSuffix(path, ".go") && !strings.HasSuffix(path, "_test.go")
}

// TestWASMHostEffectSitesAreInventoried extends the purity boundary to the
// host packages: every host clock read, goroutine start, map range and
// unstable sort in wasi, cluster and backend, including the transport files
// exempt from the pure import rule, is a determinism inventory row.
func TestWASMHostEffectSitesAreInventoried(t *testing.T) {
	inventory, err := determinism.Load()
	if err != nil {
		t.Fatal(err)
	}
	sites, err := archcheck.HostEffectSites(".", compiledGoCommand(), archcheck.Platform{OS: runtime.GOOS, Arch: runtime.GOARCH}, wasmHostFile)
	if err != nil {
		t.Fatal(err)
	}
	derived := make([]string, 0, len(sites))
	for _, site := range sites {
		derived = append(derived, fmt.Sprintf("%s|%d", site.Key(), site.Count))
	}
	if err := inventory.Complete("host-go", derived); err != nil {
		t.Fatal(err)
	}
	// The pure-import exemption of the transport files no longer hides their
	// host effects: each one is reviewed as a row like any other site.
	for _, file := range []string{"wasi/process.go", "wasi/cluster.go"} {
		if !slices.ContainsFunc(sites, func(site archcheck.HostSite) bool { return site.File == file }) {
			t.Errorf("exempt transport file %s reports no host-effect site; the scan no longer covers it", file)
		}
	}
	// A new site, or another occurrence of a reviewed one, fails by name.
	added := append(slices.Clone(derived), "wasi/process.go|Run|clock|time.Now|1")
	if err := inventory.Complete("host-go", added); err == nil || !strings.Contains(err.Error(), `"wasi/process.go|Run|clock|time.Now|1" has no inventory row`) {
		t.Fatalf("new host-effect site not reported: %v", err)
	}
}
