package gomad3_test

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"go.temporal.io/server/tools/gomad3/internal/archcheck"
	"go.temporal.io/server/tools/gomad3/internal/puritycheck"
)

var pureDependencyFiles = []string{
	"choice/tape.go", "choice/divergence.go", "choice/no_op_select.go",
	"choice/wire.go", "choice/trace.go", "choice/legacy_v1.go", "choice/interrupted.go",
	"choice/limits.go", "choice/internal/wire/wire_generated.go",
	"internal/compatibilitypack/evaluation.go", "internal/compatibilitypack/selection_pure.go",
	"internal/compatibilitypack/schema.go",
	"internal/canonicaljson/canonical.go", "internal/canonicaljson/implementation.go", "internal/canonicaljson/streaming.go",
}

func nativePureFile(path string) bool {
	for _, dependency := range pureDependencyFiles {
		if path == dependency {
			return true
		}
	}
	for _, prefix := range []string{"record/", "world/mailbox/", "runner/internal/exploration/", "backends/native/build/internal/capabilitypolicy/"} {
		if strings.HasPrefix(path, prefix) {
			return true
		}
	}
	if filepath.ToSlash(filepath.Dir(path)) == "world" {
		return true
	}
	return path == "runner/internal/campaign/controller.go" || path == "backends/native/build/capability_evaluation.go" || path == "simulation/clock/simulation_progress.go"
}

//nolint:staticcheck // Tests execute on the compiling host and require that exact reviewed Go toolchain.
func sourceGoCommand() string { return filepath.Join(runtime.GOROOT(), "bin", "go") }

func typedSourcePlatforms() []archcheck.Platform {
	platforms := []archcheck.Platform{{OS: runtime.GOOS, Arch: runtime.GOARCH}}
	if os.Getenv("GOMAD3_ARCH_ALL_PLATFORMS") == "1" || os.Getenv("CI") != "" {
		for _, platform := range []archcheck.Platform{{OS: "darwin", Arch: "arm64"}, {OS: "linux", Arch: "amd64"}} {
			if platform != platforms[0] {
				platforms = append(platforms, platform)
			}
		}
	}
	return platforms
}

func TestPureModulesHaveNoHostEffects(t *testing.T) {
	paths := append([]string{"runner/internal/campaign/controller.go", "backends/native/build/capability_evaluation.go", "simulation/clock/simulation_progress.go"}, pureDependencyFiles...)
	if err := pureFileInventory(".", paths); err != nil {
		t.Fatal(err)
	}
	for _, platform := range typedSourcePlatforms() {
		started := time.Now()
		findings, err := puritycheck.Check(".", sourceGoCommand(), platform, nativePureFile)
		if err != nil {
			t.Fatal(err)
		}
		for _, finding := range findings {
			t.Errorf("%s/%s %s %s: %s", platform.OS, platform.Arch, finding.Category, finding.Path, finding.Detail)
		}
		elapsed := time.Since(started)
		t.Logf("typed pure files on %s/%s in %s", platform.OS, platform.Arch, elapsed)
		if platform.OS == runtime.GOOS && platform.Arch == runtime.GOARCH && elapsed >= 30*time.Second {
			t.Errorf("host pure checker took %s (limit 30s)", elapsed)
		}
	}
}

func pureFileInventory(root string, paths []string) error {
	for _, path := range paths {
		info, err := os.Stat(filepath.Join(root, path))
		if err != nil {
			return fmt.Errorf("required pure file %s: %w", path, err)
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("required pure file %s is not regular source", path)
		}
	}
	return nil
}

func TestPureFileInventoryRejectsMissingAndRenamed(t *testing.T) {
	for _, name := range []string{"missing", "renamed"} {
		t.Run(name, func(t *testing.T) {
			root := t.TempDir()
			paths := []string{"original.go", "dependency.go"}
			for _, path := range paths {
				if err := os.WriteFile(filepath.Join(root, path), []byte("package fixture"), 0600); err != nil {
					t.Fatal(err)
				}
			}
			if err := pureFileInventory(root, paths); err != nil {
				t.Fatal(err)
			}
			if name == "missing" {
				if err := os.Remove(filepath.Join(root, "dependency.go")); err != nil {
					t.Fatal(err)
				}
			} else {
				if err := os.Rename(filepath.Join(root, "dependency.go"), filepath.Join(root, "renamed.go")); err != nil {
					t.Fatal(err)
				}
			}
			if err := pureFileInventory(root, paths); err == nil || !strings.Contains(err.Error(), "dependency.go") {
				t.Fatalf("missing exact dependency admitted: %v", err)
			}
		})
	}
}
