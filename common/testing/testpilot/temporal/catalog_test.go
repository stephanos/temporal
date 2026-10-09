package temporal_test

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"go.temporal.io/server/common/testing/testpilot/recordedrun"
	"go.temporal.io/server/common/testing/testpilot/temporal"
)

// The Driver catalog identity is part of every recorded Run's identity, so a change to it leaves
// every pinned Run stale. Only an actual WorkflowService/Testpilot protocol-closure change justifies
// changing this literal; current companions must be truthfully re-recorded, never catalog-relabeled.
// This repairs the inherited f98632f94a→59cb017d30 Testpilot protocol change; the historical control
// Run is preserved beside a fresh current-catalog recording.
func TestWorkflowServiceCatalogIdentityGolden(t *testing.T) {
	catalog, err := temporal.NewWorkflowServiceCatalog()
	require.NoError(t, err)
	require.Equal(t, "12fa287d45f6e40502f05d1c89643a59d612ddd53e7fde025dc90abfb7c744f6", catalog.Identity())

	read := func(path string) recordedrun.Decoded {
		t.Helper()
		bytes, err := os.ReadFile(path)
		require.NoError(t, err)
		decoded, err := recordedrun.Decode(bytes)
		require.NoError(t, err)
		return decoded
	}
	historical := read("../replay/testdata/nexusCallerControl-forgedCompletion-run.json")
	current := read("../replay/testdata/nexusCallerControl-forgedCompletion-current-run.json")
	require.Equal(t, "3364057f225cf6fd6116023ef36573038a9acd29f0d13df2da98c76062e9c3c0", historical.Driver.Catalog)
	require.Equal(t, catalog.Identity(), current.Driver.Catalog)
	require.Equal(t, historical.Case, current.Case)
	require.Equal(t, historical.Driver.Profile, current.Driver.Profile)
	require.Equal(t, historical.Driver.Bindings, current.Driver.Bindings)
	require.NotEqual(t, historical.Run.GetRunId(), current.Run.GetRunId())
}
