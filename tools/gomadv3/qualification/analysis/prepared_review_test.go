package analysis

import (
	"context"
	"os"
	"path/filepath"
	"slices"
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
	toolchainRoot, err := filepath.Abs(filepath.Join("..", "..", ".toolchain"))
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

func TestPrepareCapabilityReviewSupportsTemporalBackoffWithGRPCAdapter(t *testing.T) {
	repositoryRoot, err := filepath.Abs(filepath.Join("..", "..", "..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	toolchainRoot, err := filepath.Abs(filepath.Join("..", "..", ".toolchain"))
	if err != nil {
		t.Fatal(err)
	}
	prepared, err := PrepareCapabilityReview(context.Background(), target.Spec{
		Kind: target.KindGoTest, Source: "./common/backoff", WorkingDir: repositoryRoot,
		ToolchainRoot: toolchainRoot, BuildTags: []string{"test_dep"}, CapabilityMode: target.CapabilityModeClosure,
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := prepared.Close(); err != nil {
			t.Fatal(err)
		}
	})
	if len(prepared.Review.Findings) != 0 {
		t.Fatalf("capability findings = %#v", prepared.Review.Findings)
	}
	if len(prepared.Review.Closure.Compatibility) != 0 {
		t.Fatalf("capability compatibility allowances = %#v", prepared.Review.Closure.Compatibility)
	}
	grpcSelected := false
	for _, adapter := range prepared.Adapters {
		if adapter.Module == "google.golang.org/grpc" && adapter.Version == "v1.80.0" {
			grpcSelected = true
		}
	}
	if !grpcSelected {
		t.Fatalf("selected adapters = %#v", prepared.Adapters)
	}
	grpcInternal := false
	for _, pkg := range prepared.Review.Closure.Packages {
		if pkg.ImportPath == "golang.org/x/sys/unix" {
			t.Fatal("rewritten closure retained golang.org/x/sys/unix")
		}
		if pkg.ImportPath != "google.golang.org/grpc/internal" {
			continue
		}
		grpcInternal = true
		if slices.Contains(pkg.Imports, "syscall") || slices.Contains(pkg.Imports, "golang.org/x/sys/unix") || pkg.Module == nil || pkg.Module.Adapter == nil || pkg.Module.Adapter.Adapter.Path != "google.golang.org/grpc" {
			t.Fatalf("rewritten gRPC package = %#v", pkg)
		}
	}
	if !grpcInternal {
		t.Fatal("rewritten closure omitted google.golang.org/grpc/internal")
	}
}
