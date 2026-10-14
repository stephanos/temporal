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
// Format 4.0 changed the Testpilot protocol closure; the control's current companion was recorded
// live under it.
func TestWorkflowServiceCatalogIdentityGolden(t *testing.T) {
	catalog, err := temporal.NewWorkflowServiceCatalog()
	require.NoError(t, err)
	require.Equal(t, "4d172cbbfbcb08fa8affe83b112ab710ad8d330328bfe38706909461fd153854", catalog.Identity())

	bytes, err := os.ReadFile("../replay/testdata/nexusCallerControl-forgedCompletion-current-run.json")
	require.NoError(t, err)
	current, err := recordedrun.Decode(bytes)
	require.NoError(t, err)
	require.Equal(t, catalog.Identity(), current.Driver.Catalog)
}
