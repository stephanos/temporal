package runner

import (
	"os/exec"
	"strings"
	"testing"
)

func TestRunnerHasNoConcreteBackendDependency(t *testing.T) {
	command := exec.CommandContext(t.Context(), "go", "list", "-deps", "-tags", "test_dep", ".")
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("list Runner dependencies: %v: %s", err, output)
	}
	for _, path := range strings.Fields(string(output)) {
		if strings.HasPrefix(path, "go.temporal.io/server/tools/gomad3/backends/") {
			t.Errorf("Runner depends on concrete backend package %s", path)
		}
	}
}
