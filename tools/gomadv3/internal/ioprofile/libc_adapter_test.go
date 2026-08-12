package ioprofile

import (
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"go.temporal.io/server/tools/gomadv3/internal/target"
)

func TestProfilePreparesPinnedModerncLibcAdapter(t *testing.T) {
	toolchainRoot, err := filepath.Abs(filepath.Join("..", "..", ".toolchain"))
	if err != nil {
		t.Fatal(err)
	}
	command := exec.CommandContext(context.Background(), filepath.Join(toolchainRoot, "bin", "go"), "env", "GOMODCACHE")
	moduleCache, err := command.Output()
	if err != nil {
		t.Fatal(err)
	}
	profile, err := Resolve(Deterministic)
	if err != nil {
		t.Fatal(err)
	}
	repositoryRoot, err := filepath.Abs(filepath.Join("..", "..", "..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	spec, overlay, err := profile.PrepareBuildOverlay(target.Spec{ToolchainRoot: toolchainRoot, PreparationRoot: t.TempDir(), WorkingDir: repositoryRoot}, strings.TrimSpace(string(moduleCache)))
	if err != nil {
		t.Fatal(err)
	}
	if spec.BuildModFile != overlay.Path || spec.BuildOverlay != "" || overlay.SourceSHA256 != "sha256:46fc04624c96033980a81d8eeb9b4d73daff0c6cae511931456f2c72a75fcb7e" {
		t.Fatalf("overlay = %#v, spec = %#v", overlay, spec)
	}
	replacement, err := os.ReadFile(overlay.Replacement)
	if err != nil {
		t.Fatal(err)
	}
	for _, text := range []string{"gomadOpen", "gomadRead", "gomadWrite"} {
		if !strings.Contains(string(replacement), text) {
			t.Errorf("replacement omitted %q", text)
		}
	}
	digest := sha256.Sum256(replacement)
	if overlay.ReplacementSHA256 != fmt.Sprintf("sha256:%x", digest) {
		t.Fatalf("replacement digest = %q", overlay.ReplacementSHA256)
	}
	adapterBytes, err := os.ReadFile(filepath.Join(filepath.Dir(overlay.Replacement), "gomad_darwin.go"))
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
	_, _, err := Default().PrepareBuildOverlay(target.Spec{PreparationRoot: t.TempDir(), WorkingDir: workingDirectory}, t.TempDir())
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
	prepared, overlay, err := Default().PrepareBuildOverlay(spec, "")
	if err != nil {
		t.Fatal(err)
	}
	if prepared.BuildModFile != "" || overlay != (BuildOverlay{}) {
		t.Fatalf("prepared = %#v, overlay = %#v", prepared, overlay)
	}
}
