package execution

import (
	"os/exec"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestExecutionImportBoundary(t *testing.T) {
	command := exec.CommandContext(t.Context(), "go", "list", "-tags", "test_dep", "-deps", "go.temporal.io/server/tools/umpire/internal/execution")
	output, err := command.Output()
	require.NoError(t, err)
	for _, dependency := range strings.Fields(string(output)) {
		require.NotEqual(t, "go.temporal.io/server/tools/umpire", dependency)
		for _, prefix := range []string{"go.temporal.io/server/tools/umpire/verification", "go.temporal.io/server/tools/umpire/temporal", "go.temporal.io/sdk"} {
			require.False(t, dependency == prefix || strings.HasPrefix(dependency, prefix+"/"), "forbidden dependency: %s", dependency)
		}
	}
}
