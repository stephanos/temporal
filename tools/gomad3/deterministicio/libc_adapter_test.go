package deterministicio

import (
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"go.temporal.io/server/tools/gomad3/target"
)

func TestModerncLibcAdapterTemplateHasPinnedIdentity(t *testing.T) {
	if got := digestBytes([]byte(gomadLibcAdapterSource)); got != gomadLibcAdapterSHA256 {
		t.Fatalf("adapter template digest = %q, want %q", got, gomadLibcAdapterSHA256)
	}
}

func TestProfilePreparesPinnedModerncLibcAdapter(t *testing.T) {
	toolchainRoot, err := filepath.Abs(filepath.Join("..", ".toolchain"))
	if err != nil {
		t.Fatal(err)
	}
	command := exec.CommandContext(context.Background(), filepath.Join(toolchainRoot, "bin", "go"), "env", "GOMODCACHE")
	moduleCache, err := command.Output()
	if err != nil {
		t.Fatal(err)
	}
	profile := Default()
	repositoryRoot, err := filepath.Abs(filepath.Join("..", "..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	spec, adapters, err := profile.PrepareBuildAdapters(target.Spec{ToolchainRoot: toolchainRoot, PreparationRoot: t.TempDir(), WorkingDir: repositoryRoot}, strings.TrimSpace(string(moduleCache)))
	if err != nil {
		t.Fatal(err)
	}
	if len(adapters) != 4 {
		t.Fatalf("adapters = %#v", adapters)
	}
	var adapter BuildAdapter
	var projected target.AdapterReplacement
	for index := range adapters {
		if adapters[index].Module == libcModulePath {
			adapter = adapters[index]
			projected = spec.AdapterReplacements[index]
		}
	}
	if adapter.Module == "" || len(spec.AdapterReplacements) != 4 {
		t.Fatalf("adapter replacements = %#v", spec.AdapterReplacements)
	}
	if projected.Original.Path != adapter.Module || projected.Adapter.Path != adapter.Module || projected.ReplacementPath != adapter.ReplacementRoot || projected.ReplacementSourceInventorySHA256 != adapter.ReplacementSourceInventorySHA256 || projected.PreparedSourceSetSHA256 == "" {
		t.Fatalf("adapter projection = %#v, adapter = %#v", projected, adapter)
	}
	if spec.BuildModFile != adapter.BuildModFile || spec.BuildOverlay != "" || adapter.Module != "modernc.org/libc" || adapter.SourceSHA256 != "sha256:46fc04624c96033980a81d8eeb9b4d73daff0c6cae511931456f2c72a75fcb7e" {
		t.Fatalf("adapter = %#v, spec = %#v", adapter, spec)
	}
	replacement, err := os.ReadFile(adapter.Replacement)
	if err != nil {
		t.Fatal(err)
	}
	for _, text := range []string{"gomadOpen", "gomadRead", "gomadWrite", "gomad: unsupported modernc libc host capability: Xsocket"} {
		if !strings.Contains(string(replacement), text) {
			t.Errorf("replacement omitted %q", text)
		}
	}
	digest := sha256.Sum256(replacement)
	if adapter.ReplacementSHA256 != fmt.Sprintf("sha256:%x", digest) {
		t.Fatalf("replacement digest = %q", adapter.ReplacementSHA256)
	}
	adapterBytes, err := os.ReadFile(filepath.Join(adapter.ReplacementRoot, "gomad_darwin.go"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(adapterBytes), "internal/gomadio.Enabled") || strings.Contains(string(adapterBytes), "Temporal") || strings.Contains(string(adapterBytes), "SQLite") {
		t.Fatalf("modernc adapter = %s", adapterBytes)
	}
}

func TestProfileRejectsUnsupportedModerncLibcVersion(t *testing.T) {
	workingDirectory := t.TempDir()
	if err := os.WriteFile(filepath.Join(workingDirectory, "go.mod"), []byte("module example.test\n\ngo 1.26.4\n\nrequire modernc.org/libc v1.72.2\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	_, _, err := Default().PrepareBuildAdapters(target.Spec{PreparationRoot: t.TempDir(), WorkingDir: workingDirectory}, t.TempDir())
	if err == nil || !strings.Contains(err.Error(), "unsupported modernc.org/libc version") {
		t.Fatalf("PrepareBuildOverlay() error = %v", err)
	}
}

func TestProfileWithoutModerncLibcNeedsNoAdapter(t *testing.T) {
	workingDirectory := t.TempDir()
	if err := os.WriteFile(filepath.Join(workingDirectory, "go.mod"), []byte("module example.test\n\ngo 1.26.4\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	spec := target.Spec{WorkingDir: workingDirectory}
	prepared, adapters, err := Default().PrepareBuildAdapters(spec, "")
	if err != nil {
		t.Fatal(err)
	}
	if prepared.BuildModFile != "" || len(adapters) != 0 {
		t.Fatalf("prepared = %#v, adapters = %#v", prepared, adapters)
	}
}
