package temporal_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.temporal.io/server/common/testing/testpilot/temporal"
)

// The Driver catalog identity is part of every recorded Run's identity, so a change to it leaves
// every pinned Run stale. Only a task that changes the wire (the WorkflowService or the Testpilot
// protocol closure) may change this literal, and it re-records the pinned Runs with
// `make umpire-rerecord-pinned-runs` in the same commit. The literal equals the catalog the pinned
// control Run in common/testing/testpilot/replay/testdata was recorded under.
func TestWorkflowServiceCatalogIdentityGolden(t *testing.T) {
	catalog, err := temporal.NewWorkflowServiceCatalog()
	require.NoError(t, err)
	require.Equal(t, "3364057f225cf6fd6116023ef36573038a9acd29f0d13df2da98c76062e9c3c0", catalog.Identity())
}
