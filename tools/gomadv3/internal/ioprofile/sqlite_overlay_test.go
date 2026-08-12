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

func TestProfilePreparesPinnedSQLiteOverlay(t *testing.T) {
	toolchainRoot, err := filepath.Abs(filepath.Join("..", "..", ".toolchain"))
	if err != nil {
		t.Fatal(err)
	}
	command := exec.CommandContext(context.Background(), filepath.Join(toolchainRoot, "bin", "go"), "env", "GOMODCACHE")
	moduleCache, err := command.Output()
	if err != nil {
		t.Fatal(err)
	}
	profile, err := Resolve(TemporalActivityAPIBatchSecurity)
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
	if spec.BuildModFile != overlay.Path || spec.BuildOverlay == "" || overlay.SourceSHA256 != "sha256:49e9d6f24ca24c12a0cd99593655d5236eaf88214d7b1e0fc94a5262c44e5180" {
		t.Fatalf("overlay = %#v, spec = %#v", overlay, spec)
	}
	replacement, err := os.ReadFile(overlay.Replacement)
	if err != nil {
		t.Fatal(err)
	}
	for _, text := range []string{"//go:linkname gomadSQLiteEnabled internal/gomadio.Enabled", "gomadSQLiteRandomness(zBuf, nBuf)", "gomadSQLiteCurrentTime()"} {
		if !strings.Contains(string(replacement), text) {
			t.Errorf("replacement omitted %q", text)
		}
	}
	digest := sha256.Sum256(replacement)
	if overlay.ReplacementSHA256 != fmt.Sprintf("sha256:%x", digest) {
		t.Fatalf("replacement digest = %q", overlay.ReplacementSHA256)
	}
	overlayBytes, err := os.ReadFile(spec.BuildOverlay)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(overlayBytes), "functional_test_base.go") {
		t.Fatalf("Temporal overlay = %s", overlayBytes)
	}
}
