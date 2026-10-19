package protocolnative

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"go.temporal.io/server/tools/gomad/internal/model/generation/protocol"
)

var moduleRoot = filepath.Join("..", "..", "..", "..", "..")

func TestNativeSetIsCurrent(t *testing.T) {
	stale, err := Generate(moduleRoot, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(stale) != 0 {
		t.Fatalf("stale native outputs %v; run gomadtool protocol-generate", stale)
	}
}

func TestNativeSetWritesOnlyNativeOutputs(t *testing.T) {
	for _, target := range Targets() {
		if !strings.HasPrefix(target.Path, "internal/backends/native/") {
			t.Errorf("native target writes outside the native tree: %s", target.Path)
		}
	}
	for _, shared := range protocol.SharedTargets() {
		for _, target := range Targets() {
			if target.Path == shared.Path {
				t.Errorf("native and shared sets both write %s", target.Path)
			}
		}
	}
}

func TestParentRootFollowsReplace(t *testing.T) {
	parent, err := ParentRoot(filepath.Join(moduleRoot, "internal", "backends", "native"))
	if err != nil {
		t.Fatal(err)
	}
	want, err := filepath.Abs(moduleRoot)
	if err != nil {
		t.Fatal(err)
	}
	got, err := filepath.Abs(parent)
	if err != nil || got != want {
		t.Fatalf("parent root = %s, %v; want %s", got, err, want)
	}
	root := t.TempDir()
	native := filepath.Join(root, "native")
	elsewhere := filepath.Join(root, "elsewhere")
	for _, directory := range []string{native, elsewhere} {
		if err := os.MkdirAll(directory, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	write := func(path, contents string) {
		t.Helper()
		if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	write(filepath.Join(elsewhere, "go.mod"), "module "+parentModule+"\n\ngo 1.27.1\n")
	write(filepath.Join(native, "go.mod"), "module "+nativeModule+"\n\ngo 1.27.1\n\nreplace "+parentModule+" => ../elsewhere\n")
	if parent, err := ParentRoot(native); err != nil || parent != elsewhere {
		t.Fatalf("relocated parent root = %s, %v; want %s", parent, err, elsewhere)
	}
	write(filepath.Join(native, "go.mod"), "module "+nativeModule+"\n\ngo 1.27.1\n")
	if _, err := ParentRoot(native); err == nil {
		t.Fatal("native module without a parent replace was accepted")
	}
	write(filepath.Join(elsewhere, "go.mod"), "module example.invalid/other\n\ngo 1.27.1\n")
	write(filepath.Join(native, "go.mod"), "module "+nativeModule+"\n\ngo 1.27.1\n\nreplace "+parentModule+" => ../elsewhere\n")
	if _, err := ParentRoot(native); err == nil {
		t.Fatal("replace target declaring another module was accepted")
	}
}

// nativeInputs lists every file the native set reads, plus the module files
// ParentRoot resolves.
func nativeInputs() []string {
	inputs := []string{
		"go.mod",
		"internal/backends/native/go.mod",
		"internal/model/schema/iowire.json",
		"internal/model/schema/modelwire.json",
		"internal/model/schema/timewire.json",
		"internal/model/schema/processwire.json",
		"internal/model/schema/networkwire.json",
		"internal/model/schema/volumewire.json",
		"internal/model/choice/schema/choicewire.json",
		"internal/model/choice/schema/choicewire.go.tmpl",
		"internal/model/choice/schema/choicewire_runtime.go.tmpl",
		"internal/model/choice/trace.go",
		"internal/model/choice/tape.go",
		"internal/model/runtimesource/gomad_semantics.go.txt",
		"internal/backends/native/toolchain/runtime/go1.27.1.patch",
		"internal/backends/native/toolchain/runtime/overlay/src/cmd/compile/internal/base/gomadcap.go",
		"internal/backends/native/toolchain/runtime/overlay/src/cmd/compile/internal/base/gomadguard.go",
		"internal/backends/native/toolchain/runtime/overlay/src/cmd/compile/internal/gomadguard/guard.go",
		"internal/backends/native/toolchain/runtime/overlay/src/cmd/compile/internal/gomadintercept/intercept.go",
		"internal/backends/native/toolchain/runtime/overlay/src/cmd/internal/gomadcap/encode.go",
		"internal/backends/native/toolchain/runtime/overlay/src/cmd/link/internal/ld/gomadcap.go",
		"internal/backends/native/toolchain/runtime/overlay/src/runtime/gomad.go",
		"internal/backends/native/transcript/boundary/manifest.json",
		"internal/backends/native/transcript/boundary_generated.go",
		"internal/backends/native/build/internal/livecap/livecap.go",
		"internal/backends/native/build/internal/livecap/livecap.go.tmpl",
		"internal/backends/native/build/internal/livecap/livecap.json",
		"internal/backends/native/build/internal/livecap/project.go",
	}
	for _, target := range Targets() {
		inputs = append(inputs, target.Template)
	}
	slices.Sort(inputs)
	return slices.Compact(inputs)
}

func TestNativeGenerateWritesAndChecksEveryOutput(t *testing.T) {
	root := t.TempDir()
	for _, relative := range nativeInputs() {
		contents, err := os.ReadFile(filepath.Join(moduleRoot, filepath.FromSlash(relative)))
		if err != nil {
			t.Fatal(err)
		}
		path := filepath.Join(root, filepath.FromSlash(relative))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, contents, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := Generate(root, false); err != nil {
		t.Fatal(err)
	}
	if stale, err := Generate(root, true); err != nil || len(stale) != 0 {
		t.Fatalf("check after generate = %v, %v", stale, err)
	}
	outputs := []string{
		"internal/backends/native/build/internal/livecap/protocol_generated.go",
		"internal/backends/native/toolchain/runtime/overlay/src/cmd/internal/gomadcap/protocol_generated.go",
	}
	for _, target := range Targets() {
		outputs = append(outputs, target.Path)
	}
	for _, relative := range outputs {
		want, err := os.ReadFile(filepath.Join(moduleRoot, filepath.FromSlash(relative)))
		if err != nil {
			t.Fatal(err)
		}
		path := filepath.Join(root, filepath.FromSlash(relative))
		got, err := os.ReadFile(path)
		if err != nil || string(got) != string(want) {
			t.Fatalf("isolated generation of %s differs from the tree: %v", relative, err)
		}
		if err := os.WriteFile(path, append(got, "stale"...), 0o600); err != nil {
			t.Fatal(err)
		}
		stale, err := Generate(root, true)
		if err != nil || !slices.Equal(stale, []string{relative}) {
			t.Fatalf("check after editing %s = %v, %v", relative, stale, err)
		}
		if err := os.WriteFile(path, got, 0o600); err != nil {
			t.Fatal(err)
		}
	}
}
