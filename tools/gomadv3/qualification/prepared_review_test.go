package qualification

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"go.temporal.io/server/tools/gomadv3/target"
)

func TestPrepareCapabilityReviewOwnsPrivateAdapterLifecycle(t *testing.T) {
	workingDirectory := t.TempDir()
	if err := os.WriteFile(filepath.Join(workingDirectory, "go.mod"), []byte("module example.com/target\n\ngo 1.26.4\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(workingDirectory, "main.go"), []byte("package main\n\nfunc main() {}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	toolchainRoot, err := filepath.Abs(filepath.Join("..", ".toolchain"))
	if err != nil {
		t.Fatal(err)
	}
	prepared, err := PrepareCapabilityReview(context.Background(), target.Spec{
		Kind: target.KindGoRun, Source: ".", WorkingDir: workingDirectory, ToolchainRoot: toolchainRoot,
	})
	if err != nil {
		t.Fatal(err)
	}
	if prepared.Review.Schema != target.CapabilityReviewSchema || prepared.Spec.PreparationRoot == "" {
		t.Fatalf("prepared review = %#v", prepared)
	}
	root := prepared.Spec.PreparationRoot
	if err := prepared.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(root); !os.IsNotExist(err) {
		t.Fatalf("preparation root remained after close: %v", err)
	}
}
