package authoring_test

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"go.temporal.io/server/tools/umpire/authoring"
)

const (
	walkthroughPath = "model/AUTHORING.md"
	modelPath       = "model/Temporal/Feature/Nexus/Caller/Model.lean"
)

func checkoutRoot(t *testing.T) string {
	t.Helper()
	_, currentFile, _, ok := runtime.Caller(0)
	require.True(t, ok)
	return filepath.Clean(filepath.Join(filepath.Dir(currentFile), "..", "..", ".."))
}

func readCheckedIn(t *testing.T) (markdown, model string) {
	t.Helper()
	root := checkoutRoot(t)
	markdownBytes, err := os.ReadFile(filepath.Join(root, walkthroughPath))
	require.NoError(t, err)
	modelBytes, err := os.ReadFile(filepath.Join(root, modelPath))
	require.NoError(t, err)
	return string(markdownBytes), string(modelBytes)
}

// TestAuthoringWalkthroughQuotesTheModelFile is the drift gate: every block the walkthrough quotes
// is a region of the Model file, and every region but the terminator is quoted.
func TestAuthoringWalkthroughQuotesTheModelFile(t *testing.T) {
	t.Parallel()
	markdown, model := readCheckedIn(t)
	require.NoError(t, authoring.Check(markdown, model))
	blocks, err := authoring.Blocks(markdown)
	require.NoError(t, err)
	require.Contains(t, blocks, "entities")
	require.Contains(t, blocks, "case")
}

// TestAuthoringCheckFailsOnAPlantedMissingMarker renames a marker in the Model file and expects the
// check to name the block that lost its region.
func TestAuthoringCheckFailsOnAPlantedMissingMarker(t *testing.T) {
	t.Parallel()
	markdown, model := readCheckedIn(t)
	planted := strings.Replace(model, "-- authoring: scenarios\n", "-- authoring: paths\n", 1)
	require.NotEqual(t, model, planted)
	err := authoring.Check(markdown, planted)
	require.ErrorContains(t, err, `block "scenarios" names a marker the Model file lacks`)
}

// TestAuthoringCheckFailsOnAPlantedDuplicateMarker repeats a marker in the Model file and expects
// the check to refuse the file before comparing anything.
func TestAuthoringCheckFailsOnAPlantedDuplicateMarker(t *testing.T) {
	t.Parallel()
	markdown, model := readCheckedIn(t)
	planted := strings.Replace(model, "-- authoring: queries\n", "-- authoring: actions\n", 1)
	require.NotEqual(t, model, planted)
	err := authoring.Check(markdown, planted)
	require.ErrorContains(t, err, `marker "actions" appears twice`)
}

// TestAuthoringCheckFailsOnAPlantedEdit changes one byte of a region and expects the check to name
// the block that drifted.
func TestAuthoringCheckFailsOnAPlantedEdit(t *testing.T) {
	t.Parallel()
	markdown, model := readCheckedIn(t)
	planted := strings.Replace(model, "entity workflow\n", "entity workflows\n", 1)
	require.NotEqual(t, model, planted)
	err := authoring.Check(markdown, planted)
	require.ErrorContains(t, err, `block "entities" differs from the Model file's region`)
}

// TestAuthoringCheckFailsOnAnUnquotedRegion adds a region to the Model file that the walkthrough
// does not quote.
func TestAuthoringCheckFailsOnAnUnquotedRegion(t *testing.T) {
	t.Parallel()
	markdown, model := readCheckedIn(t)
	planted := strings.Replace(model, "-- authoring: end\n", "-- authoring: extra\n\n-- authoring: end\n", 1)
	require.NotEqual(t, model, planted)
	err := authoring.Check(markdown, planted)
	require.ErrorContains(t, err, `region "extra" is not quoted by the walkthrough`)
}

// TestAuthoringBlocksRejectAnUnfencedMarker pins the walkthrough's own shape: a marker is followed
// by a fenced Lean block and quoted once.
func TestAuthoringBlocksRejectAnUnfencedMarker(t *testing.T) {
	t.Parallel()
	_, err := authoring.Blocks("<!-- authoring: entities -->\nentity workflow\n")
	require.ErrorContains(t, err, `block "entities" is not followed by a fenced Lean block`)
	_, err = authoring.Blocks("<!-- authoring: a -->\n```lean\nx\n```\n<!-- authoring: a -->\n```lean\nx\n```\n")
	require.ErrorContains(t, err, `block "a" is quoted twice`)
}
