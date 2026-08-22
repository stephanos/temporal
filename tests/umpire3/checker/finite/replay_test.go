package finite

import (
	"testing"

	"github.com/stretchr/testify/require"
	protocolcatalog "go.temporal.io/server/tests/umpire3/protocol/catalog"
)

func TestFirstOrderViewReplaysCompiledPathsAndClassifiesLiveOnlyActions(t *testing.T) {
	t.Parallel()

	sound, found, err := DefaultFirstOrderView(protocolcatalog.TargetIDNexusCancellation, "sound")
	require.NoError(t, err)
	require.True(t, found)
	mutated, found, err := DefaultFirstOrderView(protocolcatalog.TargetIDNexusCancellation,
		"stale-completion-guard-removed")
	require.NoError(t, err)
	require.True(t, found)
	actions := []protocolcatalog.ActionKind{
		protocolcatalog.ActionKindScheduleOperation,
		protocolcatalog.ActionKindDispatchTask,
		protocolcatalog.ActionKindRequestCancellation,
		protocolcatalog.ActionKindCommitCancellation,
		protocolcatalog.ActionKindAcquireOwnership,
		protocolcatalog.ActionKindWorkerReturnsSuccess,
		protocolcatalog.ActionKindPersistSuccess,
	}

	soundReplay, err := ReplayFirstOrder(sound, actions)
	require.NoError(t, err)
	require.False(t, soundReplay.Accepted)
	require.Equal(t, protocolcatalog.ActionKindPersistSuccess, soundReplay.RejectedAction)
	require.Equal(t, []protocolcatalog.ActionKind{protocolcatalog.ActionKindScheduleOperation}, soundReplay.LiveOnlyActions)

	mutatedReplay, err := ReplayFirstOrder(mutated, actions)
	require.NoError(t, err)
	require.True(t, mutatedReplay.Accepted)
	require.Empty(t, mutatedReplay.RejectedAction)
	require.Equal(t, []protocolcatalog.ActionKind{protocolcatalog.ActionKindScheduleOperation}, mutatedReplay.LiveOnlyActions)
}
