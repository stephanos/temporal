package lower_test

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"go.temporal.io/server/common/testing/testpilot/recordedrun"
	"go.temporal.io/server/tools/umpire/lower"
)

func TestGeneratedCasesAreCheckedIn(t *testing.T) {
	files, err := lower.GenerateCases(filepath.Join("..", "..", "..", "model", "ir"))
	require.NoError(t, err)
	require.NoError(t, lower.SyncCases(filepath.Join("..", "..", "..", "model", "cases"), files, false))
	manifest, err := lower.DecodeManifest(files["manifest.json"])
	require.NoError(t, err)
	lowered := 0
	for _, entry := range manifest.Queries {
		if entry.Standing != lower.Lowered {
			continue
		}
		lowered++
		_, err := recordedrun.CaseIdentity(files[entry.File])
		require.NoError(t, err)
		require.NotNil(t, entry.Expected)
	}
	require.Positive(t, lowered)
}
