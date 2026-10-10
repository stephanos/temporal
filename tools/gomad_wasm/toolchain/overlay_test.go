package toolchain

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestImplementationIdentityBindsCompiledBuilderAndGlue(t *testing.T) {
	builder, err := os.ReadFile("overlay.go")
	if err != nil {
		t.Fatal(err)
	}
	glue, err := os.ReadFile("runtime_wasm.go.txt")
	if err != nil {
		t.Fatal(err)
	}
	input := []byte("gomad-wasm-runtime-implementation/v1\x00")
	for _, source := range [][]byte{builder, glue} {
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

func stockRoot(t *testing.T) string {
	t.Helper()
	command := os.Getenv("GOMAD3_STOCK_GO")
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
	root, err := filepath.Abs("../../gomad3/toolchain/runtime/overlay/src/runtime")
	if err != nil {
		t.Fatal(err)
	}
	return root
}

func TestOverlayBuildsLinkedWASMRuntime(t *testing.T) {
	root := filepath.Join(t.TempDir(), "stock-go")
	if err := os.CopyFS(root, os.DirFS(stockRoot(t))); err != nil {
		t.Fatal(err)
	}
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
			if err := os.MkdirAll(filepath.Join(copyRoot, "src/runtime"), 0700); err != nil {
				t.Fatal(err)
			}
			for _, file := range []string{"VERSION", "src/runtime/proc.go", "src/runtime/runtime2.go", "src/runtime/rand.go", "src/runtime/select.go", "src/runtime/time.go", "src/runtime/os_wasip1.go", "src/runtime/lock_wasip1.go", "src/runtime/note_other.go"} {
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

func TestOverlayIdentityBindsOriginalRuntimeAndNativeSemantics(t *testing.T) {
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
		codec, err := os.ReadFile(manifest.Replace[filepath.Join(root, "src/runtime/gomad_choicewire_generated.go")])
		if err != nil {
			t.Fatal(err)
		}
		original, err := os.ReadFile(filepath.Join(native, "gomad_choicewire_generated.go"))
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(codec, original) {
			t.Fatal("overlay changed native codec bytes")
		}
	}
	copyRoot := t.TempDir()
	for _, name := range []string{"gomad.go", "gomad_choicewire_generated.go"} {
		data, err := os.ReadFile(filepath.Join(native, name))
		if err != nil {
			t.Fatal(err)
		}
		if name == "gomad.go" {
			data = bytes.Replace(data, []byte("0x53c5ca59"), []byte("0x53c5ca58"), 1)
		}
		if err := os.WriteFile(filepath.Join(copyRoot, name), data, 0600); err != nil {
			t.Fatal(err)
		}
	}
	_, changed, err := BuildOverlay(root, copyRoot, t.TempDir())
	if err != nil || changed == identity {
		t.Fatalf("changed native seeded semantics did not change identity: %q %v", changed, err)
	}
}
