package preparation

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"go.temporal.io/server/tools/gomad3/deterministicio"
	"go.temporal.io/server/tools/gomad3/target"
)

func inspectionFixture(t *testing.T, source string) target.Spec {
	t.Helper()
	module := t.TempDir()
	writeTestFile(t, filepath.Join(module, "go.mod"), "module example.com/inspection\n\ngo 1.27.1\n")
	writeTestFile(t, filepath.Join(module, "main.go"), source)
	return target.Spec{Kind: target.KindGoRun, Source: ".", WorkingDir: module, ToolchainRoot: filepath.Join(testModuleRoot(t), ".toolchain")}
}

func TestInspectClosureDoesNotBuildOrLaunch(t *testing.T) {
	spec := inspectionFixture(t, "package main\n\nfunc main() { panic(\"target launched\") }\n")
	shimRoot := t.TempDir()
	if err := os.Mkdir(filepath.Join(shimRoot, "bin"), 0o700); err != nil {
		t.Fatal(err)
	}
	command := "#!/bin/sh\nif [ \"$1\" = build ] || [ \"$1\" = test ]; then echo unexpected-build >&2; exit 97; fi\nexec \"" + filepath.Join(spec.ToolchainRoot, "bin", "go") + "\" \"$@\"\n"
	writeTestFile(t, filepath.Join(shimRoot, "bin", "go"), command)
	if err := os.Chmod(filepath.Join(shimRoot, "bin", "go"), 0o700); err != nil {
		t.Fatal(err)
	}
	spec.ToolchainRoot = shimRoot
	inspected, err := Inspect(context.Background(), spec)
	if err != nil {
		t.Fatal(err)
	}
	if inspected.Review.Schema != target.CapabilityReviewSchema || inspected.Spec.PreparationRoot == "" {
		t.Fatalf("inspection = %#v", inspected)
	}
	if err := inspected.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestInspectLinkedBuildDoesNotLaunchTarget(t *testing.T) {
	sentinel := filepath.Join(t.TempDir(), "launched")
	spec := inspectionFixture(t, "package main\n\nimport \"os\"\n\nfunc main() { if err := os.WriteFile("+strconv.Quote(sentinel)+", []byte(\"launched\"), 0600); err != nil { panic(err) } }\n")
	spec.CapabilityMode = target.CapabilityModeLinked
	inspected, err := Inspect(context.Background(), spec)
	if err != nil {
		t.Fatal(err)
	}
	if inspected.Review.CapabilityManifest == nil || inspected.Review.CapabilityMode != target.CapabilityModeLinked {
		t.Fatalf("linked review = %#v", inspected.Review)
	}
	if _, err := os.Stat(sentinel); !os.IsNotExist(err) {
		t.Fatalf("target launched: %v", err)
	}
	if err := inspected.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestInspectRejectsCallerRootAndPreservesReviewClassificationOnCleanupFailure(t *testing.T) {
	spec := inspectionFixture(t, "package main\n\nfunc main() {}\n")
	spec.PreparationRoot = t.TempDir()
	if _, err := Inspect(context.Background(), spec); err == nil {
		t.Fatal("caller-owned preparation root accepted")
	}
	spec.PreparationRoot = ""
	spec.Source = "-invalid"
	cleanupFailure := errors.New("cleanup failed")
	_, err := inspect(context.Background(), spec, func(path string) error {
		if !strings.Contains(filepath.Base(path), "gomad3-compatibility-review-") {
			t.Fatalf("cleanup path = %q", path)
		}
		if removeErr := os.RemoveAll(path); removeErr != nil {
			t.Fatal(removeErr)
		}
		return cleanupFailure
	})
	if !target.IsInvalidCapabilityReview(err) || !errors.Is(err, cleanupFailure) {
		t.Fatalf("inspection error = %v", err)
	}
}

func TestInspectCloseSurfacesCleanupFailure(t *testing.T) {
	spec := inspectionFixture(t, "package main\n\nfunc main() {}\n")
	cleanupFailure := errors.New("cleanup failed")
	inspected, err := inspect(context.Background(), spec, func(path string) error {
		if err := os.RemoveAll(path); err != nil {
			t.Fatal(err)
		}
		return cleanupFailure
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := inspected.Close(); !errors.Is(err, cleanupFailure) {
		t.Fatalf("Close() = %v", err)
	}
}

func TestInspectPreservesInvalidAdapterSum(t *testing.T) {
	spec := inspectionFixture(t, "package main\n\nfunc main() {}\n")
	module, err := os.ReadFile(filepath.Join(testModuleRoot(t), "deterministicio", "testdata", "sprig", "go.mod"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(spec.WorkingDir, "go.mod"), module, 0o600); err != nil {
		t.Fatal(err)
	}
	_, err = Inspect(context.Background(), spec)
	if !deterministicio.IsInvalidBuildAdapterConfiguration(err) || StageOf(err) != StageAdapters {
		t.Fatalf("inspection error = %v, stage = %s", err, StageOf(err))
	}
}

func TestInspectRetainsUnsupportedClosureClassification(t *testing.T) {
	spec := inspectionFixture(t, "package main\n\nimport \"os/exec\"\n\nfunc main() { _ = exec.Command(\"true\") }\n")
	inspected, err := Inspect(context.Background(), spec)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := inspected.Close(); err != nil {
			t.Fatal(err)
		}
	}()
	if len(inspected.Review.Findings) == 0 {
		t.Fatal("unsupported closure has no finding")
	}
	_, err = target.ReviewCapabilityClosure(context.Background(), inspected.Spec)
	var unsupported *target.UnsupportedCapabilityError
	if !errors.As(err, &unsupported) || unsupported.Capability != "imports os/exec" {
		t.Fatalf("unsupported closure error = %v", err)
	}
}

func TestInspectLinkedRejectsMalformedBuildRecord(t *testing.T) {
	spec := inspectionFixture(t, "package main\n\nfunc main() {}\n")
	spec.CapabilityMode = target.CapabilityModeLinked
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "bin"), 0o700); err != nil {
		t.Fatal(err)
	}
	key, err := os.ReadFile(filepath.Join(spec.ToolchainRoot, "build-key"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "build-key"), key, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(spec.ToolchainRoot, "builds"), filepath.Join(root, "builds")); err != nil {
		t.Fatal(err)
	}
	command := "#!/bin/sh\n" +
		"if [ \"$1\" != build ]; then exec \"" + filepath.Join(spec.ToolchainRoot, "bin", "go") + "\" \"$@\"; fi\n" +
		"\"" + filepath.Join(spec.ToolchainRoot, "bin", "go") + "\" \"$@\" || exit $?\n" +
		"while [ $# -gt 0 ]; do if [ \"$1\" = -o ]; then shift; printf 'malformed linked record' > \"$1\"; exit 0; fi; shift; done\n" +
		"exit 98\n"
	writeTestFile(t, filepath.Join(root, "bin", "go"), command)
	if err := os.Chmod(filepath.Join(root, "bin", "go"), 0o700); err != nil {
		t.Fatal(err)
	}
	spec.ToolchainRoot = root
	_, err = Inspect(context.Background(), spec)
	if err == nil || !strings.Contains(err.Error(), "extract linked target capability manifest") || target.IsInvalidCapabilityReview(err) || target.IsUnsupportedCapability(err) || StageOf(err) != StageReview {
		t.Fatalf("inspection error = %v, stage = %s", err, StageOf(err))
	}
}
