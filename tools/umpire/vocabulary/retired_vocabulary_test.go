package vocabulary_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"

	"go.temporal.io/server/tools/umpire/internal/retiredvocabulary"
)

func TestRetiredVocabularyCommandRejectsRetiredPublicTokens(t *testing.T) {
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
			repositoryRoot := seedScannedSurface(t)
			writeFixture(t, repositoryRoot, test.path, test.content)

			command := retiredVocabularyCommand(t, repositoryRoot)
			output, err := command.CombinedOutput()
			require.Error(t, err)
			require.Contains(t, string(output), filepath.ToSlash(test.path))
			require.Contains(t, string(output), test.token)
		})
	}
}

func TestRetiredVocabularyCommandAllowsOrdinaryEnglishAndExcludedHistory(t *testing.T) {
	t.Parallel()

	repositoryRoot := seedScannedSurface(t)
	writeFixture(t, repositoryRoot, "model/README.md", "A projection can refine a bounded engineering approximation without claiming conformance or qualification.\n")
	writeFixture(t, repositoryRoot, "tools/legacy/history.go", "package legacy\nconst old = \""+"semantic"+"Identity\"\n")
	writeFixture(t, repositoryRoot, ".flow/memory/history.md", "The old API used "+"Declaration"+"Id.\n")
	writeFixture(t, repositoryRoot, ".flow/specs/fn-18-versioned-umpire-artifact-boundary.json", `{"id":"fn-18-versioned-umpire-artifact-boundary","status":"closed","note":"`+"semantic"+`Identity"}`+"\n")
	writeFixture(t, repositoryRoot, ".flow/specs/fn-18-versioned-umpire-artifact-boundary.md", "Historical "+"Qualification"+"Result.\n")

	command := retiredVocabularyCommand(t, repositoryRoot)
	output, err := command.CombinedOutput()
	require.NoError(t, err, string(output))
	require.Empty(t, output)
}

func TestRetiredVocabularyCommandAllowsOnlyCaseBoundsAndCatalogQualifiedLiteral(t *testing.T) {
	t.Parallel()

	for _, test := range []struct {
		name    string
		path    string
		content string
	}{
		{name: "Case encoder", path: "model/Umpire/Case/ProtoJSON.lean", content: "def key := \"" + "bou" + "nds\"\n"},
		{name: "Temporal Case", path: "tests/testcore/testpilot/testdata/get-system-info-case.json", content: "{\"" + "bou" + "nds\":{}}\n"},
		{name: "conformance Case", path: "common/testing/testpilot/testdata/case-runtime-conformance/satisfied/case.json", content: "{\"" + "bou" + "nds\":{}}\n"},
		{name: "catalog validation literal", path: "common/testing/testpilot/internal/ir/catalog.go", content: "package ir\nconst suffix = \"." + "qualified\"\n"},
	} {
		t.Run(test.name, func(t *testing.T) {
			repositoryRoot := seedScannedSurface(t)
			writeFixture(t, repositoryRoot, test.path, test.content)
			output, err := retiredVocabularyCommand(t, repositoryRoot).CombinedOutput()
			require.NoError(t, err, string(output))
			require.Empty(t, output)
		})
	}

	for _, test := range []struct {
		name    string
		path    string
		content string
	}{
		{name: "bounds outside Case schema", path: "tools/umpire/fixture.json", content: "{\"" + "bou" + "nds\":{}}\n"},
		{name: "qualified outside validation", path: "tools/umpire/fixture.go", content: "package umpire\nconst suffix = \"." + "qualified\"\n"},
	} {
		t.Run("reject "+test.name, func(t *testing.T) {
			repositoryRoot := seedScannedSurface(t)
			writeFixture(t, repositoryRoot, test.path, test.content)
			output, err := retiredVocabularyCommand(t, repositoryRoot).CombinedOutput()
			require.Error(t, err)
			require.Contains(t, string(output), filepath.ToSlash(test.path))
		})
	}
}

func TestRetiredVocabularyCommandFailsOnAMissingScannedPath(t *testing.T) {
	t.Parallel()

	for _, test := range []struct {
		name    string
		removed string
	}{
		{name: "required file", removed: "model/Umpire/ARCHITECTURE.md"},
		{name: "scan root", removed: "model/Shared"},
		{name: "open spec record", removed: ".flow/specs/" + retiredvocabulary.DownstreamSpecs()[0] + ".json"},
	} {
		t.Run(test.name, func(t *testing.T) {
			repositoryRoot := seedScannedSurface(t)
			require.NoError(t, os.RemoveAll(filepath.Join(repositoryRoot, filepath.FromSlash(test.removed))))

			output, err := retiredVocabularyCommand(t, repositoryRoot).CombinedOutput()
			require.Error(t, err)
			require.Contains(t, string(output), "scanned path "+test.removed+" does not exist")
		})
	}
}

func TestRetiredVocabularyCommandScansTestpilotAndSharedTrees(t *testing.T) {
	t.Parallel()

	for _, path := range []string{"model/Testpilot/Fixture.lean", "model/Shared/Fixture.lean"} {
		t.Run(path, func(t *testing.T) {
			repositoryRoot := seedScannedSurface(t)
			writeFixture(t, repositoryRoot, path, "structure "+"Projection"+"Record where\n  id : String\n")

			output, err := retiredVocabularyCommand(t, repositoryRoot).CombinedOutput()
			require.Error(t, err)
			require.Contains(t, string(output), path)
			require.Contains(t, string(output), "Projection"+"Record")
		})
	}
}

func retiredVocabularyCommand(t *testing.T, repositoryRoot string) *exec.Cmd {
	t.Helper()

	_, currentFile, _, ok := runtime.Caller(0)
	require.True(t, ok)
	checkoutRoot := filepath.Clean(filepath.Join(filepath.Dir(currentFile), "..", "..", ".."))

	command := exec.Command(
		"go", "run", "./tools/umpire/cmd/umpire-check-retired-vocabulary",
		"--repository-root", repositoryRoot,
	)
	command.Dir = checkoutRoot
	return command
}

// seedScannedSurface builds a temporary repository root that holds every path
// the scan requires, so a test asserts on the token it plants rather than on a
// missing-path error.
func seedScannedSurface(t *testing.T) string {
	t.Helper()

	repositoryRoot := t.TempDir()
	for _, root := range retiredvocabulary.ScanRoots() {
		require.NoError(t, os.MkdirAll(filepath.Join(repositoryRoot, filepath.FromSlash(root)), 0o755))
	}
	for _, path := range retiredvocabulary.RequiredFiles() {
		writeFixture(t, repositoryRoot, path, "")
	}
	for _, specID := range retiredvocabulary.DownstreamSpecs() {
		writeFixture(t, repositoryRoot, ".flow/specs/"+specID+".json", `{"id":"`+specID+`","status":"closed"}`+"\n")
	}
	writeFixture(t, repositoryRoot, ".plans/UMPIRE4_SPEC.md", "# Umpire4\n")
	return repositoryRoot
}

func writeFixture(t *testing.T, repositoryRoot, relativePath, content string) {
	t.Helper()

	path := filepath.Join(repositoryRoot, filepath.FromSlash(relativePath))
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))
}
