package finite_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.temporal.io/server/tests/umpire3/checker/finite"
	protocolcatalog "go.temporal.io/server/tests/umpire3/protocol/catalog"
	protocolexperiment "go.temporal.io/server/tests/umpire3/protocol/experiment"
)

func TestFiniteReplayCatalogCoversEveryTargetWithoutSpecializedFirstOrderView(t *testing.T) {
	t.Parallel()

	catalog, err := finite.DefaultFiniteReplayCatalog()
	require.NoError(t, err)
	composition, err := protocolcatalog.DefaultComposition()
	require.NoError(t, err)
	expected := 0
	for _, target := range composition.Targets {
		if target.Identifier != protocolcatalog.TargetIDNexusCancellation {
			expected += len(target.Properties)
		}
	}
	require.Len(t, catalog.Targets, expected)
	for _, target := range catalog.Targets {
		require.Equal(t, protocolcatalog.ResultClassFiniteExhaustive, target.ResultClass)
		require.Equal(t, protocolcatalog.TrustBadgeCheckedCertificate, target.TrustBadge)
		require.NotEmpty(t, target.Relation.Declaration)
		require.NotContains(t, target.CanonicalModel, "Umpire3.Temporal.System.Umpire3.")
	}
}

func TestFiniteReplaySeparatesAppliedAndNonAppliedAttempts(t *testing.T) {
	t.Parallel()

	catalog, err := finite.DefaultFiniteReplayCatalog()
	require.NoError(t, err)
	view, found := catalog.Target(protocolcatalog.TargetIDFeatureNexus, protocolcatalog.PropertyIDNexusOperationClosure)
	require.True(t, found)

	replay, err := (finite.AttemptExecutionView{Finite: &view}).Replay([]finite.AttemptRequest{{
		Action: protocolcatalog.ActionKindPersistSuccess, Outcomes: []protocolexperiment.ActionOutcome{protocolexperiment.ActionOutcomeApplied},
	}})
	require.NoError(t, err)
	require.False(t, replay.Accepted)
	require.Equal(t, protocolcatalog.ActionKindPersistSuccess, replay.RejectedAction)

	replay, err = (finite.AttemptExecutionView{Finite: &view}).Replay([]finite.AttemptRequest{{
		Action: protocolcatalog.ActionKindPersistSuccess, Outcomes: []protocolexperiment.ActionOutcome{protocolexperiment.ActionOutcomeSuppressed},
	}})
	require.NoError(t, err)
	require.True(t, replay.Accepted)
}

func TestFiniteReplayAcceptsNexusClosureActionFromInitialState(t *testing.T) {
	t.Parallel()

	catalog, err := finite.DefaultFiniteReplayCatalog()
	require.NoError(t, err)
	view, found := catalog.Target(protocolcatalog.TargetIDFeatureNexus, protocolcatalog.PropertyIDNexusOperationClosure)
	require.True(t, found)

	replay, err := (finite.AttemptExecutionView{Finite: &view}).Replay([]finite.AttemptRequest{{
		Action: protocolcatalog.ActionKindCloseNexusOperation, Outcomes: []protocolexperiment.ActionOutcome{protocolexperiment.ActionOutcomeApplied},
	}})
	require.NoError(t, err)
	require.True(t, replay.Accepted)
}

func TestFiniteReplayRejectsAnUnboundGraphAction(t *testing.T) {
	catalog, err := finite.DefaultFiniteReplayCatalog()
	require.NoError(t, err)
	catalog.Targets[0].Attempts[0].AppliedPaths[0] = []string{"unknown"}
	require.ErrorContains(t, catalog.Validate(), "unknown graph action")
}
