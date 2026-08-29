package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRenderGeneratedRunnerTestMatchesTheCheckedInOrdinaryGoTest(t *testing.T) {
	packageRoot := filepath.Join("..", "..", "temporal", "nexus")
	manifestPath := filepath.Join(
		packageRoot,
		"testdata",
		"caller-closure-input-set",
		"manifest.json",
	)
	input, err := loadGenerationInput(manifestPath, packageRoot)
	require.NoError(t, err)

	generated, err := renderGeneratedTest(input)
	require.NoError(t, err)
	require.Contains(t, string(generated), "context.WithTimeout(context.Background(), 135*time.Second)")
	want, err := os.ReadFile(filepath.Join(packageRoot, generatedTestFileName))
	require.NoError(t, err)
	require.Equal(t, want, generated)
}

func TestRunRegeneratesOnlyTheDeterministicGoTest(t *testing.T) {
	packageRoot := filepath.Join(hostTempDir(t), "nexus")
	fixtureRoot := filepath.Join(packageRoot, "testdata", "caller-closure-input-set")
	copyInputSet(t, fixtureRoot)
	manifestPath := filepath.Join(fixtureRoot, "manifest.json")

	require.NoError(t, run([]string{manifestPath, "--output", packageRoot}))
	first, err := os.ReadFile(filepath.Join(packageRoot, generatedTestFileName))
	require.NoError(t, err)
	require.NoError(t, run([]string{manifestPath, "--output", packageRoot}))
	second, err := os.ReadFile(filepath.Join(packageRoot, generatedTestFileName))
	require.NoError(t, err)
	require.Equal(t, first, second)

	entries, err := os.ReadDir(packageRoot)
	require.NoError(t, err)
	require.Len(t, entries, 2)
	require.Equal(t, generatedTestFileName, entries[0].Name())
	require.Equal(t, "testdata", entries[1].Name())
}

func TestRenderGeneratedRunnerTestQuotesWhitespaceInEmbedPaths(t *testing.T) {
	packageRoot := filepath.Join(hostTempDir(t), "nexus")
	fixtureRoot := filepath.Join(packageRoot, "fixture with space")
	copyInputSet(t, fixtureRoot)
	input, err := loadGenerationInput(filepath.Join(fixtureRoot, "manifest.json"), packageRoot)
	require.NoError(t, err)

	generated, err := renderGeneratedTest(input)
	require.NoError(t, err)
	require.Contains(t, string(generated), `//go:embed "fixture with space/manifest.json"`)
}

func TestValidateEmbedRootRejectsPatternMetacharactersAndControls(t *testing.T) {
	for _, embedRoot := range []string{
		"fixture*",
		"fixture?",
		"fixture[one]",
		`fixture\one`,
		"fixture\nnext",
	} {
		t.Run(strings.ReplaceAll(embedRoot, "/", "-"), func(t *testing.T) {
			require.ErrorContains(t, validateEmbedRoot(embedRoot), "unsafe generated test fixture path")
		})
	}
}

func hostTempDir(t *testing.T) string {
	t.Helper()
	root := filepath.Join("..", "..", "..", "..", ".flow", "tmp")
	require.NoError(t, os.MkdirAll(root, 0o755))
	temporary, err := os.MkdirTemp(root, "fn19-8-generator-")
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, os.RemoveAll(temporary))
	})
	return temporary
}

func TestRunRejectsAnythingButTheGenerationGrammar(t *testing.T) {
	for _, args := range [][]string{
		nil,
		{"manifest.json"},
		{"manifest.json", "--run", "nexus"},
		{"manifest.json", "--output", "nexus", "--run"},
	} {
		require.ErrorContains(t, run(args), "expected <manifest> --output <package>")
	}
}

func copyInputSet(t *testing.T, destination string) {
	t.Helper()
	source := filepath.Join(
		"..", "..", "temporal", "nexus", "testdata", "caller-closure-input-set",
	)
	for _, relative := range []string{
		"manifest.json",
		filepath.Join("artifacts", "experiment.json"),
		filepath.Join("artifacts", "runtime-configuration.json"),
	} {
		require.NoError(t, os.MkdirAll(filepath.Dir(filepath.Join(destination, relative)), 0o755))
		encoded, err := os.ReadFile(filepath.Join(source, relative))
		require.NoError(t, err)
		require.NoError(t, os.WriteFile(filepath.Join(destination, relative), encoded, 0o600))
	}
}
