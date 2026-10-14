//go:build gomad3_integration

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

	output, status := runMake(t, repositoryRoot, "gomad3-run", "GOMADSEED=103", "GOMAD3_RUN=./tools/gomad/internal/backends/native/conformance/testdata/clock/main.go", "GOMAD3_ARGS=initial")
	if status != 0 || !strings.HasSuffix(strings.TrimSpace(output), "clock initial ok") {
		t.Fatalf("gomad3-run status=%d output=%q", status, output)
	}

	moduleRoot := filepath.Join(repositoryRoot, "tools", "gomad")
	taggedGo := "GOMAD3_GO=\"" + filepath.Join(moduleRoot, ".toolchain", "bin", "go") + "\" -C \"" + moduleRoot + "\""
	first, status := runMake(t, repositoryRoot, "gomad3-test", "GOMADSEED=101", "GOMAD3_PACKAGES=./backends/native/qualification/testdata/tagged", "GOMAD3_ARGS=-run=TestTargetRequiresTestDep", taggedGo)
	if status != 0 {
		t.Fatalf("first gomad3-test status=%d output=%q", status, first)
	}
	second, status := runMake(t, repositoryRoot, "gomad3-test", "GOMADSEED=102", "GOMAD3_PACKAGES=./backends/native/qualification/testdata/tagged", "GOMAD3_ARGS=-run=TestTargetRequiresTestDep", taggedGo)
	if status != 0 || strings.Contains(first, "(cached)") || strings.Contains(second, "(cached)") || strings.Contains(first, "[no test files]") || strings.Contains(second, "[no test files]") {
		t.Fatalf("second gomad3-test status=%d first=%q second=%q", status, first, second)
	}

	output, status = runMake(t, repositoryRoot, "gomad3-test", "GOMADSEED=", "GOMAD3_PACKAGES=./backends/native/qualification/testdata/tagged", "GOMAD3_ARGS=-run=TestTargetRequiresTestDep", taggedGo)
	if status != 2 || !strings.Contains(output, "runtime: invalid GOMADSEED") {
		t.Fatalf("empty-seed gomad3-test status=%d output=%q", status, output)
	}
}
