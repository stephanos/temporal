package authoring_test

import (
	"maps"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"go.temporal.io/server/tools/umpire/authoring"
)

const (
	walkthroughPath = "model/lean/AUTHORING.md"
	callerPath      = "model/lean/Temporal/Feature/Nexus/Caller/Model.lean"
	workerPath      = "model/lean/Temporal/Feature/Worker/Model.lean"
	outagePath      = "model/lean/Temporal/Feature/Workflow/Outage/Model.lean"
	controlPath     = "model/lean/Temporal/Feature/Nexus/Control/Model.lean"
)

// modelPaths are the Model files the walkthrough quotes: the caller Model it walks through, and the
// worker entity, the Outage composition and the derived Control machine its composition section
// quotes.
var modelPaths = []string{callerPath, workerPath, outagePath, controlPath}

func checkoutRoot(t *testing.T) string {
	t.Helper()
	_, currentFile, _, ok := runtime.Caller(0)
	require.True(t, ok)
	return filepath.Clean(filepath.Join(filepath.Dir(currentFile), "..", "..", ".."))
}

func readCheckedIn(t *testing.T) (markdown string, models map[string]string) {
	t.Helper()
	root := checkoutRoot(t)
	markdownBytes, err := os.ReadFile(filepath.Join(root, walkthroughPath))
	require.NoError(t, err)
	models = map[string]string{}
	for _, path := range modelPaths {
		modelBytes, err := os.ReadFile(filepath.Join(root, path))
		require.NoError(t, err)
		models[path] = string(modelBytes)
	}
	return string(markdownBytes), models
}

// plant replaces one occurrence of old in the Model file at path and fails if it is absent.
func plant(t *testing.T, models map[string]string, path, old, replacement string) map[string]string {
	t.Helper()
	planted := maps.Clone(models)
	planted[path] = strings.Replace(models[path], old, replacement, 1)
	require.NotEqual(t, models[path], planted[path])
	return planted
}

// TestAuthoringWalkthroughQuotesTheModelFiles is the drift gate: every block the walkthrough quotes
// is a region of one Model file, and every region but each file's terminator is quoted.
func TestAuthoringWalkthroughQuotesTheModelFiles(t *testing.T) {
	t.Parallel()
	markdown, models := readCheckedIn(t)
	require.NoError(t, authoring.Check(markdown, models))
	blocks, err := authoring.Blocks(markdown)
	require.NoError(t, err)
	for _, path := range modelPaths {
		regions, err := authoring.Regions(models[path])
		require.NoError(t, err)
		require.Contains(t, regions, "end", path)
		require.Greater(t, len(regions), 1, "%s quotes no region", path)
		for name := range regions {
			if name != "end" {
				require.Contains(t, blocks, name, path)
			}
		}
	}
}

// TestAuthoringCheckFailsOnAPlantedMissingMarker renames a marker in the Model file and expects the
// check to name the block that lost its region.
func TestAuthoringCheckFailsOnAPlantedMissingMarker(t *testing.T) {
	t.Parallel()
	markdown, models := readCheckedIn(t)
	planted := plant(t, models, callerPath, "-- authoring: scenarios\n", "-- authoring: paths\n")
	err := authoring.Check(markdown, planted)
	require.ErrorContains(t, err, `block "scenarios" names a marker no Model file has`)
}

// TestAuthoringCheckFailsOnAPlantedDuplicateMarker repeats a marker in the Model file and expects
// the check to refuse the file before comparing anything.
func TestAuthoringCheckFailsOnAPlantedDuplicateMarker(t *testing.T) {
	t.Parallel()
	markdown, models := readCheckedIn(t)
	planted := plant(t, models, callerPath, "-- authoring: queries\n", "-- authoring: actions\n")
	err := authoring.Check(markdown, planted)
	require.ErrorContains(t, err, callerPath+`: marker "actions" appears twice`)
}

// TestAuthoringCheckFailsOnAMarkerInTwoFiles gives a region of the worker module a name the caller
// Model already marks: a block is named by its marker alone, so the walkthrough could quote either.
func TestAuthoringCheckFailsOnAMarkerInTwoFiles(t *testing.T) {
	t.Parallel()
	markdown, models := readCheckedIn(t)
	planted := plant(t, models, workerPath, "-- authoring: polling\n", "-- authoring: actions\n")
	err := authoring.Check(markdown, planted)
	require.ErrorContains(t, err, `marker "actions" appears in both `+callerPath+` and `+workerPath)
}

// TestAuthoringCheckFailsOnAPlantedEdit changes one byte of a region in each quoted file and
// expects the check to name the block that drifted and its file.
func TestAuthoringCheckFailsOnAPlantedEdit(t *testing.T) {
	t.Parallel()
	markdown, models := readCheckedIn(t)
	for _, planting := range []struct{ path, old, replacement, block string }{
		{callerPath, "entity workflow\n", "entity workflows\n", "entities"},
		{workerPath, "entity worker\n", "entity workers\n", "worker"},
		{outagePath, "compose workerOutage\n", "compose workerOutages\n", "outage"},
		{controlPath, "  from: pair\n", "  from: pairs\n", "derived"},
	} {
		planted := plant(t, models, planting.path, planting.old, planting.replacement)
		err := authoring.Check(markdown, planted)
		require.ErrorContains(t, err, `block "`+planting.block+`" differs from its region in `+planting.path)
	}
}

// TestAuthoringCheckFailsOnAnUnquotedRegion adds a region to a Model file that the walkthrough
// does not quote.
func TestAuthoringCheckFailsOnAnUnquotedRegion(t *testing.T) {
	t.Parallel()
	markdown, models := readCheckedIn(t)
	planted := plant(t, models, controlPath, "-- authoring: end\n", "-- authoring: extra\n\n-- authoring: end\n")
	err := authoring.Check(markdown, planted)
	require.ErrorContains(t, err, `region "extra" of `+controlPath+` is not quoted by the walkthrough`)
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
