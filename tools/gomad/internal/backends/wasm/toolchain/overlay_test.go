package toolchain

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"go.temporal.io/server/tools/gomad/internal/backends/wasm/internal/stockgo"
)

func TestSelectedGuestStallTimeoutRemainsDistinctFromDeadline(t *testing.T) {
	file, err := parser.ParseFile(token.NewFileSet(), "net_stream_wasm.go.txt", netStreamGlue, 0)
	if err != nil {
		t.Fatal(err)
	}
	var resultSwitch *ast.SwitchStmt
	ast.Inspect(file, func(node ast.Node) bool {
		statement, ok := node.(*ast.SwitchStmt)
		if !ok {
			return true
		}
		if name, ok := statement.Tag.(*ast.Ident); ok && name.Name == "errno" {
			resultSwitch = statement
		}
		return true
	})
	if resultSwitch == nil {
		t.Fatal("selected guest error decoder is missing")
	}
	var decoder bytes.Buffer
	if err := format.Node(&decoder, token.NewFileSet(), resultSwitch); err != nil {
		t.Fatal(err)
	}
	status, err := parser.ParseFile(token.NewFileSet(), "status.go", netStreamStatusGlue, 0)
	if err != nil {
		t.Fatal(err)
	}
	var statusDeclaration bytes.Buffer
	for _, declaration := range status.Decls {
		if err := format.Node(&statusDeclaration, token.NewFileSet(), declaration); err != nil {
			t.Fatal(err)
		}
	}
	source := `package guestcontrol
import ("context"; "errors"; "os"; "syscall"; "testing")
var ErrClosed = os.ErrClosed
var errCanceled = context.Canceled
` + statusDeclaration.String() + `
func decode(errno syscall.Errno) error {` + decoder.String() + `}
func TestTimeoutIdentity(t *testing.T) {
 stall := decode(65558)
 deadline := decode(syscall.ETIMEDOUT)
 if !errors.Is(stall, syscall.ETIMEDOUT) || errors.Is(stall, os.ErrDeadlineExceeded) {
  t.Fatalf("terminal timeout identity lost: %v", stall)
 }
 timeout, ok := stall.(interface { Timeout() bool })
 if !ok || !timeout.Timeout() { t.Fatalf("stall is not a timeout: %v", stall) }
 if !errors.Is(deadline, os.ErrDeadlineExceeded) { t.Fatalf("caller deadline changed: %v", deadline) }
}`
	directory := t.TempDir()
	if err := os.WriteFile(filepath.Join(directory, "decoder_test.go"), []byte(source), 0600); err != nil {
		t.Fatal(err)
	}
	command := exec.Command(filepath.Join(stockRoot(t), "bin/go"), "test", "-count=1", "decoder_test.go")
	command.Dir = directory
	command.Env = append(os.Environ(), "GOROOT=", "GOTOOLCHAIN=local", "GOWORK=off", "GOENV=off", "GOFLAGS=")
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("selected guest error control: %v\n%s", err, output)
	}
}

func TestImplementationIdentityBindsCompiledBuilderAndGlue(t *testing.T) {
	builder, err := os.ReadFile("overlay.go")
	if err != nil {
		t.Fatal(err)
	}
	glue, err := os.ReadFile("runtime_wasm.go.txt")
	if err != nil {
		t.Fatal(err)
	}
	inspectionGlue, err := os.ReadFile("runtime_inspect_wasm.go.txt")
	if err != nil {
		t.Fatal(err)
	}
	netStreamGlue, err := os.ReadFile("net_stream_wasm.go.txt")
	if err != nil {
		t.Fatal(err)
	}
	netStreamStatusGlue, err := os.ReadFile("net_stream_status_generated.go.txt")
	if err != nil {
		t.Fatal(err)
	}
	input := []byte("gomad-wasm-runtime-implementation/v1\x00")
	for _, source := range [][]byte{builder, glue, inspectionGlue, netStreamGlue, netStreamStatusGlue} {
		var size [8]byte
		binary.BigEndian.PutUint64(size[:], uint64(len(source)))
		input = append(input, size[:]...)
		input = append(input, source...)
	}
	digest := sha256.Sum256(input)
	t.Chdir(t.TempDir())
	if got := ImplementationSHA256(); got != "sha256:"+hex.EncodeToString(digest[:]) {
		t.Fatalf("compiled runtime implementation identity = %q", got)
	}
}

func TestOverlayReusesOnlyVerifiedImmutableOutput(t *testing.T) {
	root := stockRoot(t)
	output := t.TempDir()
	first, identity, err := BuildOverlay(root, nativeRoot(t), output)
	if err != nil {
		t.Fatal(err)
	}
	second, repeated, err := BuildOverlay(root, nativeRoot(t), output)
	if err != nil || first != second || identity != repeated {
		t.Fatalf("overlay did not reuse immutable output: %q %q %v", first, second, err)
	}
	data, err := os.ReadFile(first)
	if err != nil {
		t.Fatal(err)
	}
	var manifest struct{ Replace map[string]string }
	if err := json.Unmarshal(data, &manifest); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(manifest.Replace[filepath.Join(root, "src/runtime/proc.go")], []byte("corrupt"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, _, err := BuildOverlay(root, nativeRoot(t), output); err == nil {
		t.Fatal("reused corrupt runtime overlay")
	}
}

func TestOverlayCompilerAliasesRequireSeparatePreparationCaches(t *testing.T) {
	original := stockRoot(t)
	alias := t.TempDir()
	files := []string{"VERSION"}
	for name := range stockRewrites() {
		files = append(files, sourcePath(name))
	}
	for _, name := range files {
		body, err := os.ReadFile(filepath.Join(original, name))
		if err != nil {
			t.Fatal(err)
		}
		path := filepath.Join(alias, name)
		if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, body, 0600); err != nil {
			t.Fatal(err)
		}
	}
	shared := t.TempDir()
	first, identity, err := BuildOverlay(original, nativeRoot(t), shared)
	if err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(first)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := BuildOverlay(alias, nativeRoot(t), shared); err == nil || !strings.Contains(err.Error(), "overlay.json") {
		t.Fatalf("same-content compiler alias reused a descriptor with foreign paths: %v", err)
	}
	second, aliasIdentity, err := BuildOverlay(alias, nativeRoot(t), t.TempDir())
	if err != nil || identity != aliasIdentity {
		t.Fatalf("private alias cache changed runtime identity: %q %q %v", identity, aliasIdentity, err)
	}
	after, err := os.ReadFile(first)
	if err != nil || !bytes.Equal(before, after) {
		t.Fatalf("alias preparation modified original descriptor: %v", err)
	}
	body, err := os.ReadFile(second)
	if err != nil || bytes.Equal(before, body) {
		t.Fatalf("alias descriptor did not preserve its distinct raw paths: %v", err)
	}
	var descriptor struct{ Replace map[string]string }
	if err := json.Unmarshal(body, &descriptor); err != nil {
		t.Fatal(err)
	}
	if descriptor.Replace[filepath.Join(alias, "src/runtime/proc.go")] == "" || descriptor.Replace[filepath.Join(original, "src/runtime/proc.go")] != "" {
		t.Fatal("private descriptor did not bind the actual compiler alias")
	}
}

func stockRoot(t *testing.T) string {
	t.Helper()
	command := os.Getenv("GOMAD_STOCK_GO")
	if command == "" {
		command = "go"
	}
	cmd := exec.Command(command, "env", "GOROOT")
	cmd.Env = append(os.Environ(), "GOROOT=", "GOTOOLCHAIN=local")
	data, err := cmd.Output()
	if err != nil {
		t.Fatal(err)
	}
	return strings.TrimSpace(string(data))
}

func nativeRoot(t *testing.T) string {
	t.Helper()
	root, err := filepath.Abs("../../../model/runtimesource")
	if err != nil {
		t.Fatal(err)
	}
	return root
}

func TestOverlayBuildsLinkedWASMRuntime(t *testing.T) {
	root := stockgo.Root(t)
	overlay, identity, err := BuildOverlay(root, nativeRoot(t), t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if len(identity) != 64 {
		t.Fatalf("runtime identity = %q", identity)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	module := filepath.Join(t.TempDir(), "runtime.wasm")
	command := exec.CommandContext(ctx, filepath.Join(root, "bin/go"), "build", "-trimpath", "-overlay", overlay, "-o", module, "./testdata/choices")
	command.Env = append(os.Environ(), "GOROOT="+root, "GOOS=wasip1", "GOARCH=wasm", "GOEXPERIMENT=nogreenteagc", "GOWORK=off", "GOENV=off", "GOFLAGS=", "GOTOOLCHAIN=local", "CGO_ENABLED=0")
	if data, err := command.CombinedOutput(); err != nil {
		t.Fatalf("build linked runtime: %v\n%s", err, data)
	}
	data, err := os.ReadFile(module)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(data[:8], []byte{'\x00', 'a', 's', 'm', 1, 0, 0, 0}) {
		t.Fatal("not a WASM module")
	}
	for _, name := range []string{"gomad_wasm_v1", "config", "decision", "observation", "finish", "idle"} {
		if !bytes.Contains(data, []byte(name)) {
			t.Fatalf("missing linked runtime import %q", name)
		}
	}
}

func TestOverlayRejectsVersionAndSourceDrift(t *testing.T) {
	root := stockRoot(t)
	for _, kind := range []string{"version", "anchor"} {
		t.Run(kind, func(t *testing.T) {
			copyRoot := t.TempDir()
			for _, directory := range []string{"src/runtime", "src/net"} {
				if err := os.MkdirAll(filepath.Join(copyRoot, directory), 0700); err != nil {
					t.Fatal(err)
				}
			}
			for _, file := range []string{"VERSION", "src/runtime/proc.go", "src/runtime/runtime2.go", "src/runtime/rand.go", "src/runtime/select.go", "src/runtime/time.go", "src/runtime/os_wasip1.go", "src/runtime/lock_wasip1.go", "src/runtime/note_other.go", "src/net/net_fake.go"} {
				data, err := os.ReadFile(filepath.Join(root, file))
				if err != nil {
					t.Fatal(err)
				}
				if kind == "version" && file == "VERSION" {
					data = []byte("go1.27.2\n")
				}
				if kind == "anchor" && file == "src/runtime/proc.go" {
					data = bytes.Replace(data, []byte("randinit() // must run before mallocinit, AlgInit, mcommoninit"), []byte("randinit() // unexpected source"), 1)
				}
				if err := os.WriteFile(filepath.Join(copyRoot, file), data, 0600); err != nil {
					t.Fatal(err)
				}
			}
			if _, _, err := BuildOverlay(copyRoot, nativeRoot(t), t.TempDir()); err == nil {
				t.Fatalf("accepted %s drift", kind)
			}
		})
	}
}

func TestOverlayIdentityBindsOriginalRuntimeAndSharedSemantics(t *testing.T) {
	root := stockRoot(t)
	native := nativeRoot(t)
	first, identity, err := BuildOverlay(root, native, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	second, same, err := BuildOverlay(root, native, t.TempDir())
	if err != nil || same != identity {
		t.Fatalf("output location changed identity: %q %q %v", identity, same, err)
	}
	for _, overlay := range []string{first, second} {
		data, err := os.ReadFile(overlay)
		if err != nil {
			t.Fatal(err)
		}
		var manifest struct{ Replace map[string]string }
		if err := json.Unmarshal(data, &manifest); err != nil {
			t.Fatal(err)
		}
		for _, name := range []string{"gomad_semantics.go", "gomad_choicewire_generated.go"} {
			copied, err := os.ReadFile(manifest.Replace[filepath.Join(root, "src/runtime", name)])
			if err != nil {
				t.Fatal(err)
			}
			original, err := os.ReadFile(filepath.Join(native, name+".txt"))
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(copied, original) {
				t.Fatalf("overlay changed runtime semantics %s bytes", name)
			}
		}
	}
	for _, changedName := range []string{"gomad_semantics.go", "gomad_choicewire_generated.go"} {
		t.Run(changedName, func(t *testing.T) {
			copyRoot := t.TempDir()
			for _, name := range []string{"gomad_semantics.go", "gomad_choicewire_generated.go"} {
				data, err := os.ReadFile(filepath.Join(native, name+".txt"))
				if err != nil {
					t.Fatal(err)
				}
				if name == changedName {
					data = append(data, '\n')
				}
				if err := os.WriteFile(filepath.Join(copyRoot, name+".txt"), data, 0600); err != nil {
					t.Fatal(err)
				}
			}
			output := t.TempDir()
			manifestPath, changed, err := BuildOverlay(root, copyRoot, output)
			if err != nil || changed == identity {
				t.Fatalf("runtime identity did not bind %s: %q %v", changedName, changed, err)
			}
			data, err := os.ReadFile(manifestPath)
			if err != nil {
				t.Fatal(err)
			}
			var manifest struct{ Replace map[string]string }
			if err := json.Unmarshal(data, &manifest); err != nil {
				t.Fatal(err)
			}
			original, err := os.ReadFile(filepath.Join(native, changedName+".txt"))
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(manifest.Replace[filepath.Join(root, "src/runtime", changedName)], original, 0600); err != nil {
				t.Fatal(err)
			}
			if _, _, err := BuildOverlay(root, copyRoot, output); err == nil || !strings.Contains(err.Error(), changedName) {
				t.Fatalf("stale runtime copy not diagnosed: %v", err)
			}
		})
	}
}
