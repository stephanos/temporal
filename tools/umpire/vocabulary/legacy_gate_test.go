package vocabulary_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLegacyVocabularyCommandRejectsRetiredPublicTokens(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		path    string
		content string
		token   string
	}{
		{
			name:    "Lean API",
			path:    "model/Umpire/Fixture.lean",
			content: "def retired : " + "Declaration" + "Id := by sorry\n",
			token:   "Declaration" + "Id",
		},
		{
			name:    "artifact wire version",
			path:    "model/README.md",
			content: "The old format was `umpire-experiment/" + "v1`.\n",
			token:   "umpire-experiment/" + "v1",
		},
		{
			name:    "artifact wire key",
			path:    "tools/umpire/regression/fixture.go",
			content: "package regression\nconst key = \"" + "semantic" + "Identity\"\n",
			token:   "semantic" + "Identity",
		},
		{
			name:    "generated view API",
			path:    "tools/umpire/regression/fixture.go",
			content: "package regression\nfunc check() { " + "Require" + "Projection() }\n",
			token:   "Require" + "Projection",
		},
		{
			name:    "versioned Qualification API",
			path:    "tools/umpire/evaluation/fixture.go",
			content: "package evaluation\ntype retired " + "Qualification" + "ReceiptV4\n",
			token:   "Qualification" + "Receipt",
		},
		{
			name:    "lower camel Qualification wire key",
			path:    "tools/umpire/evaluation/fixture.json",
			content: "{\"" + "qualifi" + "cation" + "Receipt\":{}}\n",
			token:   "Qualification" + "Receipt",
		},
		{
			name:    "versioned Conformance API",
			path:    "tools/umpire/runevaluation/fixture.go",
			content: "package runevaluation\ntype retired " + "Conformance" + "ResultV2\n",
			token:   "Conformance" + "Result",
		},
		{
			name:    "versioned Refinement API",
			path:    "model/Temporal/System/Nexus/Fixture.lean",
			content: "structure " + "Refinement" + "ResultV3 where\n  accepted : Bool\n",
			token:   "Refinement" + "Result",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			repositoryRoot := t.TempDir()
			writeFixture(t, repositoryRoot, test.path, test.content)

			command := legacyVocabularyCommand(t, repositoryRoot)
			output, err := command.CombinedOutput()
			require.Error(t, err)
			require.Contains(t, string(output), filepath.ToSlash(test.path))
			require.Contains(t, string(output), test.token)
		})
	}
}

func TestLegacyVocabularyCommandAllowsOrdinaryEnglishAndExcludedHistory(t *testing.T) {
	t.Parallel()

	repositoryRoot := t.TempDir()
	writeFixture(t, repositoryRoot, "model/README.md", "A projection can refine a bounded engineering approximation without claiming conformance or qualification.\n")
	writeFixture(t, repositoryRoot, "tools/umpire3/history.go", "package umpire3\nconst old = \""+"semantic"+"Identity\"\n")
	writeFixture(t, repositoryRoot, ".flow/memory/history.md", "The old API used "+"Declaration"+"Id.\n")
	writeFixture(t, repositoryRoot, ".flow/specs/fn-18-versioned-umpire-artifact-boundary.json", `{"id":"fn-18-versioned-umpire-artifact-boundary","status":"closed","note":"`+"semantic"+`Identity"}`+"\n")
	writeFixture(t, repositoryRoot, ".flow/specs/fn-18-versioned-umpire-artifact-boundary.md", "Historical "+"Qualification"+"Result.\n")

	command := legacyVocabularyCommand(t, repositoryRoot)
	output, err := command.CombinedOutput()
	require.NoError(t, err, string(output))
	require.Empty(t, output)
}

func legacyVocabularyCommand(t *testing.T, repositoryRoot string) *exec.Cmd {
	t.Helper()

	_, currentFile, _, ok := runtime.Caller(0)
	require.True(t, ok)
	checkoutRoot := filepath.Clean(filepath.Join(filepath.Dir(currentFile), "..", "..", ".."))

	command := exec.Command(
		"go", "run", "./tools/umpire/cmd/umpire-check-legacy-vocabulary",
		"--repository-root", repositoryRoot,
	)
	command.Dir = checkoutRoot
	return command
}

func writeFixture(t *testing.T, repositoryRoot, relativePath, content string) {
	t.Helper()

	path := filepath.Join(repositoryRoot, filepath.FromSlash(relativePath))
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))
}
