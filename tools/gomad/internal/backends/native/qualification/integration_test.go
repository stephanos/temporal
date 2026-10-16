//go:build gomad_integration

package qualification

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestRootMakeWrappers(t *testing.T) {
	repositoryRoot, err := filepath.Abs(filepath.Join("..", "..", "..", "..", "..", ".."))
	if err != nil {
		t.Fatal(err)
	}

	output, status := runMake(t, repositoryRoot, "gomad-run", "GOMADSEED=103", "GOMAD_RUN=./tools/gomad/internal/backends/native/conformance/testdata/clock/main.go", "GOMAD_ARGS=initial")
	if status != 0 || !strings.HasSuffix(strings.TrimSpace(output), "clock initial ok") {
		t.Fatalf("gomad-run status=%d output=%q", status, output)
	}

	moduleRoot := filepath.Join(repositoryRoot, "tools", "gomad")
	taggedGo := "GOMAD_GO=\"" + filepath.Join(moduleRoot, ".toolchain", "bin", "go") + "\" -C \"" + moduleRoot + "\""
	first, status := runMake(t, repositoryRoot, "gomad-test", "GOMADSEED=101", "GOMAD_PACKAGES=go.temporal.io/server/tools/gomad/internal/backends/native/qualification/testdata/tagged", "GOMAD_ARGS=-run=TestTargetRequiresTestDep", taggedGo)
	if status != 0 {
		t.Fatalf("first gomad-test status=%d output=%q", status, first)
	}
	second, status := runMake(t, repositoryRoot, "gomad-test", "GOMADSEED=102", "GOMAD_PACKAGES=go.temporal.io/server/tools/gomad/internal/backends/native/qualification/testdata/tagged", "GOMAD_ARGS=-run=TestTargetRequiresTestDep", taggedGo)
	if status != 0 || strings.Contains(first, "(cached)") || strings.Contains(second, "(cached)") || strings.Contains(first, "[no test files]") || strings.Contains(second, "[no test files]") {
		t.Fatalf("second gomad-test status=%d first=%q second=%q", status, first, second)
	}

	output, status = runMake(t, repositoryRoot, "gomad-test", "GOMADSEED=", "GOMAD_PACKAGES=go.temporal.io/server/tools/gomad/internal/backends/native/qualification/testdata/tagged", "GOMAD_ARGS=-run=TestTargetRequiresTestDep", taggedGo)
	if status != 2 || !strings.Contains(output, "runtime: invalid GOMADSEED") {
		t.Fatalf("empty-seed gomad-test status=%d output=%q", status, output)
	}
}
