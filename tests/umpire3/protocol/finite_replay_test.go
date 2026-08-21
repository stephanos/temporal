package protocol

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFiniteReplayCatalogCoversEveryTargetWithoutSpecializedFirstOrderView(t *testing.T) {
	t.Parallel()

	catalog, err := DefaultFiniteReplayCatalog()
	require.NoError(t, err)
	composition, err := DefaultComposition()
	require.NoError(t, err)
	expected := 0
	for _, target := range composition.Targets {
		if target.Identifier != TargetIDNexusCancellation {
			expected += len(target.Properties)
		}
	}
	require.Len(t, catalog.Targets, expected)
	for _, target := range catalog.Targets {
		require.Equal(t, ResultClassFiniteExhaustive, target.ResultClass)
		require.Equal(t, TrustBadgeCheckedCertificate, target.TrustBadge)
		require.NotEmpty(t, target.Relation.Declaration)
		require.NotContains(t, target.CanonicalModel, "Umpire3.Temporal.System.Umpire3.")
	}
}

func TestFiniteReplaySeparatesAppliedAndNonAppliedAttempts(t *testing.T) {
	t.Parallel()

	catalog, err := DefaultFiniteReplayCatalog()
	require.NoError(t, err)
	view, found := catalog.Target(TargetIDFeatureNexus, PropertyIDNexusOperationClosure)
	require.True(t, found)

	replay, err := view.Replay([]AttemptRequest{{
		Action: ActionKindPersistSuccess, Outcomes: []ActionOutcome{ActionOutcomeApplied},
	}})
	require.NoError(t, err)
	require.False(t, replay.Accepted)
	require.Equal(t, ActionKindPersistSuccess, replay.RejectedAction)

	replay, err = view.Replay([]AttemptRequest{{
		Action: ActionKindPersistSuccess, Outcomes: []ActionOutcome{ActionOutcomeSuppressed},
	}})
	require.NoError(t, err)
	require.True(t, replay.Accepted)
}

func TestFiniteReplayAcceptsNexusClosureActionFromInitialState(t *testing.T) {
	t.Parallel()

	catalog, err := DefaultFiniteReplayCatalog()
	require.NoError(t, err)
	view, found := catalog.Target(TargetIDFeatureNexus, PropertyIDNexusOperationClosure)
	require.True(t, found)

	replay, err := view.Replay([]AttemptRequest{{
		Action: ActionKindCloseNexusOperation, Outcomes: []ActionOutcome{ActionOutcomeApplied},
	}})
	require.NoError(t, err)
	require.True(t, replay.Accepted)
}

func TestFiniteReplayRejectsAnUnboundGraphAction(t *testing.T) {
	catalog, err := DefaultFiniteReplayCatalog()
	require.NoError(t, err)
	catalog.Targets[0].Attempts[0].AppliedPaths[0] = []string{"unknown"}
	require.ErrorContains(t, catalog.Validate(), "unknown graph action")
}
