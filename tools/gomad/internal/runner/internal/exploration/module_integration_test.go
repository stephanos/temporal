//go:build integration

package exploration_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func rootSimulationTarget(t *testing.T) (root, toolchainRoot, source string) {
	t.Helper()
	root, err := filepath.Abs(filepath.Join("..", "..", "..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	return root, filepath.Join(root, ".toolchain"), "./internal/model/cluster"
}

func TestRootSimulationTargetResolvesInGomadModule(t *testing.T) {
	root, _, source := rootSimulationTarget(t)
	command := exec.CommandContext(t.Context(), "go", "list", "-mod=readonly", "-tags", "test_dep,gomad_toolchain", "-f", "{{.Module.Path}}|{{.ImportPath}}", source)
	command.Dir = root
	command.Env = append(filteredSimulationBuildEnvironment(os.Environ()), "GOTOOLCHAIN=local", "GOWORK=off", "GOFLAGS=")
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("resolve root simulation target: %v: %s", err, output)
	}
	const expected = "go.temporal.io/server/tools/gomad|go.temporal.io/server/tools/gomad/internal/model/cluster"
	if strings.TrimSpace(string(output)) != expected {
		t.Fatalf("root simulation target ownership = %q, want %q", output, expected)
	}
}

func filteredSimulationBuildEnvironment(environment []string) []string {
	filtered := make([]string, 0, len(environment))
	for _, entry := range environment {
		name, _, found := strings.Cut(entry, "=")
		if !found {
			continue
		}
		switch name {
		case "CGO_ENABLED", "GOEXPERIMENT", "GOMADSEED", "GOMAD_CHILD_SEED", "GOTOOLCHAIN", "GOWORK":
			continue
		}
		filtered = append(filtered, entry)
	}
	return filtered
}
