package regression

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

const hermeticCITestCommand = "mise exec -- go test -count=1 -tags test_dep ./tools/umpire/temporal/nexus/... -run '^TestHermeticCIPortability$'"

func TestHermeticCIWorkflowDelegatesToOrdinaryPinnedTest(t *testing.T) {
	repositoryRoot, err := filepath.Abs(filepath.Join("..", "..", ".."))
	require.NoError(t, err)

	workflowPath := filepath.Join(repositoryRoot, ".github", "workflows", "umpire.yml")
	workflowBytes, err := os.ReadFile(workflowPath)
	require.NoError(t, err)
	workflow := string(workflowBytes)

	require.Contains(t, workflow, "permissions:\n  contents: read\n")
	require.Contains(t, workflow, "run: "+hermeticCITestCommand)
	require.Contains(t, workflow, "go-version-file: \"go.mod\"")
	require.Contains(t, workflow, "working_directory: model")
	require.Equal(t, 2, strings.Count(workflow, "cache: false"))
	require.Equal(t, 1, strings.Count(workflow, "permissions:"))
	require.Equal(t, 1, strings.Count(workflow, "  contents: read"))
	require.Equal(t, 1, strings.Count(workflow, "        run:"))

	actionPattern := regexp.MustCompile(`(?m)^\s+- uses: ([^@\s]+)@([0-9a-f]{40})(?:\s+#.*)?$`)
	actions := actionPattern.FindAllStringSubmatch(workflow, -1)
	require.Equal(t, strings.Count(workflow, "- uses: "), len(actions))
	require.Equal(t, []string{
		"actions/checkout",
		"actions/setup-go",
		"jdx/mise-action",
	}, actionNames(actions))
	require.NotContains(t, workflow, "actions/cache")
	require.NotContains(t, workflow, "secrets.")
	require.NotContains(t, workflow, "token:")
	require.NotContains(t, workflow, "id-token:")
	require.NotContains(t, workflow, ": write")
	require.NotContains(t, workflow, "\nenv:")
	for _, forbidden := range []string{
		"--checker",
		"--credential",
		"--endpoint",
		"--namespace",
		"--policy",
		"--profile",
		"umpire-qualify",
	} {
		require.NotContains(t, workflow, forbidden)
	}

	workflowMatches, err := filepath.Glob(filepath.Join(repositoryRoot, ".github", "workflows", "*.yml"))
	require.NoError(t, err)
	invocations := 0
	for _, match := range workflowMatches {
		content, err := os.ReadFile(match)
		require.NoError(t, err)
		invocations += strings.Count(string(content), "TestHermeticCIPortability")
	}
	require.Equal(t, 1, invocations)

	makefileBytes, err := os.ReadFile(filepath.Join(repositoryRoot, "Makefile"))
	require.NoError(t, err)
	require.Contains(t, string(makefileBytes),
		"mise exec -- go test -count=1 -tags test_dep \\\n\t\t./tools/umpire/temporal/nexus/... -run '^TestHermeticCIPortability$$'")
}

func actionNames(actions [][]string) []string {
	names := make([]string, len(actions))
	for index, action := range actions {
		names[index] = action[1]
	}
	return names
}
