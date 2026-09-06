package execution

import (
	"os/exec"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPrivateCoreImportBoundary(t *testing.T) {
	for _, packageName := range []string{"ir", "execution", "verification"} {
		command := exec.CommandContext(t.Context(), "go", "list", "-tags", "test_dep", "-deps", "go.temporal.io/server/common/testing/testpilot/internal/"+packageName)
		output, err := command.Output()
		require.NoError(t, err)
		for _, dependency := range strings.Fields(string(output)) {
			for _, prefix := range []string{"go.temporal.io/server/tools/umpire", "go.temporal.io/server/tests", "go.temporal.io/sdk"} {
				require.False(t, dependency == prefix || strings.HasPrefix(dependency, prefix+"/"), "forbidden dependency: %s", dependency)
			}
		}
	}
}
