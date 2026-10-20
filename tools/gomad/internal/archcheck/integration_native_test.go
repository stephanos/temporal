//go:build gomad_native

package archcheck_test

import (
	"encoding/json"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"

	"go.temporal.io/server/tools/gomad/internal/archcheck"
)

func TestNativeWrapperFixtureSelectorsResolve(t *testing.T) {
	file, err := parser.ParseFile(token.NewFileSet(), "internal/backends/native/qualification/integration_test.go", nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	selectors := 0
	ast.Inspect(file, func(node ast.Node) bool {
		literal, ok := node.(*ast.BasicLit)
		if !ok || literal.Kind != token.STRING {
			return true
		}
		value, err := strconv.Unquote(literal.Value)
		if err != nil {
			t.Fatal(err)
		}
		selector, ok := strings.CutPrefix(value, "GOMAD_PACKAGES=")
		if !ok {
			return true
		}
		selectors++
		command := exec.CommandContext(t.Context(), "go", "list", "-mod=readonly", "-json", "-tags", archcheck.SelectionTags(), selector)
		output, err := command.CombinedOutput()
		if err != nil {
			t.Errorf("resolve native wrapper fixture %q: %v: %s", selector, err, output)
			return true
		}
		var pkg struct{ TestGoFiles, XTestGoFiles []string }
		if err := json.Unmarshal(output, &pkg); err != nil || len(pkg.TestGoFiles)+len(pkg.XTestGoFiles) == 0 {
			t.Errorf("native wrapper fixture %q selects no tagged test files: %v: %s", selector, err, output)
		}
		return true
	})
	if selectors != 3 {
		t.Fatalf("native wrapper fixture selectors = %d, want 3", selectors)
	}
}

func TestHostGatePreservesRelocatedPackageSelectors(t *testing.T) {
	output, err := exec.Command("make", "--no-print-directory", "-n", "-o", "toolchain", "test-host").CombinedOutput()
	if err != nil {
		t.Fatalf("expand host gate: %v: %s", err, output)
	}
	root := t.TempDir()
	tool := filepath.Join(root, ".toolchain", "bin", "go")
	if err := os.MkdirAll(filepath.Dir(tool), 0o755); err != nil {
		t.Fatal(err)
	}
	shim := "#!/bin/sh\nprintf '%s\\n' \"GOMADSEED=${GOMADSEED-unset}\" \"GOMAD_CHILD_SEED=${GOMAD_CHILD_SEED-unset}\" \"GOMAD_STOCK_GO=$GOMAD_STOCK_GO\" \"GOEXPERIMENT=$GOEXPERIMENT\" \"GOWORK=$GOWORK\" \"$@\"\n"
	if err := os.WriteFile(tool, []byte(shim), 0o755); err != nil {
		t.Fatal(err)
	}
	command := exec.Command("sh", "-c", string(output))
	command.Dir = root
	command.Env = append(os.Environ(), "GOMAD_STOCK_GO=/stock/go", "GOMADSEED=17", "GOMAD_CHILD_SEED=23")
	invocation, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("run expanded host gate: %v: %s", err, invocation)
	}
	arguments := strings.Split(strings.TrimSpace(string(invocation)), "\n")
	wantPrefix := []string{"GOMADSEED=unset", "GOMAD_CHILD_SEED=unset", "GOMAD_STOCK_GO=/stock/go", "GOEXPERIMENT=nogreenteagc", "GOWORK=off", "test", "-tags", "test_dep,gomad_native"}
	if len(arguments) < len(wantPrefix) || !slices.Equal(arguments[:len(wantPrefix)], wantPrefix) {
		t.Fatalf("host gate environment and flags = %q, want prefix %q", arguments, wantPrefix)
	}
	for _, selector := range []string{
		"./cmd/...", "./internal/runner/...", "go.temporal.io/server/tools/gomad/internal/backends/native/execution/...", "./internal/model/clock/...",
		"./internal/evidence/qualification/...", "./internal/backends/target/...", "./internal/evidence/record/...", "./internal/evidence/artifact/...", "./internal/model/choice/...",
		"./internal/evidence/artifact/capture/...", "./internal/model/volume/...", "go.temporal.io/server/tools/gomad/internal/backends/native/transcript/...", "./internal/model/world/...",
		"go.temporal.io/server/tools/gomad/internal/backends/native/toolchain/version", "go.temporal.io/server/tools/gomad/internal/backends/native/toolchain/installation",
		"go.temporal.io/server/tools/gomad/internal/backends/native/upgrade/...", "go.temporal.io/server/tools/gomad/internal/backends/native/preparation/...", "go.temporal.io/server/tools/gomad/internal/backends/native/qualification/...",
		"go.temporal.io/server/tools/gomad/internal/backends/native/generation/...", "go.temporal.io/server/tools/gomad/internal/backends/native/build/...", "go.temporal.io/server/tools/gomad/internal/backends/native/cmd/...",
		"go.temporal.io/server/tools/gomad/internal/backends/native/compatibilitypack/...", "go.temporal.io/server/tools/gomad/internal/backends/native/sourceinventory",
		"./internal/host/...", "./internal/archcheck", "./internal/model/iowire", "./internal/model/strictjson",
	} {
		if !slices.Contains(arguments[len(wantPrefix):], selector) {
			t.Errorf("canonical host gate excludes %s", selector)
		}
	}
}
