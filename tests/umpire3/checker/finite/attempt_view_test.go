package finite_test

import (
	"bytes"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"go.temporal.io/server/tests/umpire3/checker/finite"
	protocolcatalog "go.temporal.io/server/tests/umpire3/protocol/catalog"
	protocolchecker "go.temporal.io/server/tests/umpire3/protocol/checker"
	protocolexperiment "go.temporal.io/server/tests/umpire3/protocol/experiment"
)

func TestDefaultAttemptExecutionViewResolvesExportedNexusExperiment(t *testing.T) {
	t.Parallel()

	encoded, err := os.ReadFile("../../testdata/generated/nexus-cancellation.json")
	require.NoError(t, err)
	experiment, err := protocolexperiment.DecodeExperiment(bytes.NewReader(encoded), protocolexperiment.DefaultDecodeLimit)
	require.NoError(t, err)

	view, found, err := finite.DefaultAttemptExecutionView(experiment)
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, protocolcatalog.TargetIDNexusCancellation, view.Attempts.Target)
}

func TestAttemptViewSeparatesLiveOutcomeFromAbstractTransitions(t *testing.T) {
	t.Parallel()

	sound, found, err := finite.DefaultFirstOrderView(protocolcatalog.TargetIDNexusCancellation, "sound")
	require.NoError(t, err)
	require.True(t, found)
	attempts, found, err := finite.DefaultAttemptView(protocolcatalog.TargetIDNexusCancellation, "sound")
	require.NoError(t, err)
	require.True(t, found)

	path := []finite.AttemptRequest{
		{Action: protocolcatalog.ActionKindScheduleOperation},
		{Action: protocolcatalog.ActionKindDispatchTask},
		{Action: protocolcatalog.ActionKindRequestCancellation},
		{Action: protocolcatalog.ActionKindCommitCancellation},
		{Action: protocolcatalog.ActionKindAcquireOwnership},
		{Action: protocolcatalog.ActionKindRetryTask},
		{Action: protocolcatalog.ActionKindWorkerReturnsSuccess},
		{Action: protocolcatalog.ActionKindPersistSuccess},
	}
	replay, err := (finite.AttemptExecutionView{FirstOrder: sound, Attempts: attempts}).Replay(path)
	require.NoError(t, err)
	require.True(t, replay.Accepted)
	require.Contains(t, replay.LiveOnlyActions, protocolcatalog.ActionKindScheduleOperation)
	require.Contains(t, replay.LiveOnlyActions, protocolcatalog.ActionKindRetryTask)

	path[len(path)-1].Outcomes = []protocolexperiment.ActionOutcome{protocolexperiment.ActionOutcomeApplied}
	replay, err = (finite.AttemptExecutionView{FirstOrder: sound, Attempts: attempts}).Replay(path)
	require.NoError(t, err)
	require.False(t, replay.Accepted)
	require.Equal(t, protocolcatalog.ActionKindPersistSuccess, replay.RejectedAction)
	require.Equal(t, protocolexperiment.ActionOutcomeApplied, replay.RejectedOutcome)
}

func TestAttemptViewReplaysEveryDeclaredNonAppliedOutcomeAsGuardedStutter(t *testing.T) {
	t.Parallel()

	view, found, err := finite.DefaultFirstOrderView(protocolcatalog.TargetIDNexusCancellation, "sound")
	require.NoError(t, err)
	require.True(t, found)
	attempts, found, err := finite.DefaultAttemptView(protocolcatalog.TargetIDNexusCancellation, "sound")
	require.NoError(t, err)
	require.True(t, found)

	prefix := []finite.ObservedAttempt{
		{Action: protocolcatalog.ActionKindScheduleOperation, Outcome: protocolexperiment.ActionOutcomeApplied},
		{Action: protocolcatalog.ActionKindDispatchTask, Outcome: protocolexperiment.ActionOutcomeApplied},
		{Action: protocolcatalog.ActionKindRequestCancellation, Outcome: protocolexperiment.ActionOutcomeApplied},
		{Action: protocolcatalog.ActionKindCommitCancellation, Outcome: protocolexperiment.ActionOutcomeApplied},
		{Action: protocolcatalog.ActionKindAcquireOwnership, Outcome: protocolexperiment.ActionOutcomeApplied},
		{Action: protocolcatalog.ActionKindWorkerReturnsSuccess, Outcome: protocolexperiment.ActionOutcomeApplied},
	}
	for _, outcome := range []protocolexperiment.ActionOutcome{
		protocolexperiment.ActionOutcomeSuppressed,
		protocolexperiment.ActionOutcomeRejected,
		protocolexperiment.ActionOutcomeRetried,
		protocolexperiment.ActionOutcomeFaultIntercepted,
	} {
		replay, err := (finite.AttemptExecutionView{FirstOrder: view, Attempts: attempts}).ReplayObserved(append(prefix, finite.ObservedAttempt{
			Action: protocolcatalog.ActionKindPersistSuccess, Outcome: outcome,
		}))
		require.NoError(t, err)
		require.True(t, replay.Accepted, outcome)
	}

	replay, err := (finite.AttemptExecutionView{FirstOrder: view, Attempts: attempts}).ReplayObserved(append(prefix, finite.ObservedAttempt{
		Action: protocolcatalog.ActionKindPersistSuccess, Outcome: protocolexperiment.ActionOutcomeApplied,
	}))
	require.NoError(t, err)
	require.False(t, replay.Accepted)
}

func TestMutatedAttemptViewAcceptsAppliedStalePersistence(t *testing.T) {
	t.Parallel()

	view, found, err := finite.DefaultFirstOrderView(protocolcatalog.TargetIDNexusCancellation,
		"stale-completion-guard-removed")
	require.NoError(t, err)
	require.True(t, found)
	attempts, found, err := finite.DefaultAttemptView(protocolcatalog.TargetIDNexusCancellation,
		"stale-completion-guard-removed")
	require.NoError(t, err)
	require.True(t, found)

	replay, err := (finite.AttemptExecutionView{FirstOrder: view, Attempts: attempts}).ReplayObserved([]finite.ObservedAttempt{
		{Action: protocolcatalog.ActionKindScheduleOperation, Outcome: protocolexperiment.ActionOutcomeApplied},
		{Action: protocolcatalog.ActionKindDispatchTask, Outcome: protocolexperiment.ActionOutcomeApplied},
		{Action: protocolcatalog.ActionKindRequestCancellation, Outcome: protocolexperiment.ActionOutcomeApplied},
		{Action: protocolcatalog.ActionKindCommitCancellation, Outcome: protocolexperiment.ActionOutcomeApplied},
		{Action: protocolcatalog.ActionKindAcquireOwnership, Outcome: protocolexperiment.ActionOutcomeApplied},
		{Action: protocolcatalog.ActionKindWorkerReturnsSuccess, Outcome: protocolexperiment.ActionOutcomeApplied},
		{Action: protocolcatalog.ActionKindPersistSuccess, Outcome: protocolexperiment.ActionOutcomeApplied},
	})
	require.NoError(t, err)
	require.True(t, replay.Accepted)
}

func TestAttemptViewRejectsIncompleteOrUnboundMappings(t *testing.T) {
	t.Parallel()

	for _, test := range []struct {
		name       string
		mutate     func(*protocolchecker.AttemptView)
		errorMatch string
	}{
		{
			name: "missing action mapping",
			mutate: func(view *protocolchecker.AttemptView) {
				view.Attempts = view.Attempts[:len(view.Attempts)-1]
			},
			errorMatch: "do not exactly cover",
		},
		{
			name: "unknown abstract transition",
			mutate: func(view *protocolchecker.AttemptView) {
				view.Attempts[0].Outcomes[0].Transitions = []protocolcatalog.ActionKind{"unknown-transition"}
			},
			errorMatch: "unknown abstract transition",
		},
		{
			name: "mismatched first-order hash",
			mutate: func(view *protocolchecker.AttemptView) {
				view.FirstOrderSemanticHash =
					"sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
			},
			errorMatch: "identity does not match",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			firstOrder, found, err := finite.DefaultFirstOrderView(protocolcatalog.TargetIDNexusCancellation, "sound")
			require.NoError(t, err)
			require.True(t, found)
			view, found, err := finite.DefaultAttemptView(protocolcatalog.TargetIDNexusCancellation, "sound")
			require.NoError(t, err)
			require.True(t, found)
			encoded, err := view.CanonicalJSON()
			require.NoError(t, err)
			view, err = protocolchecker.DecodeAttemptView(bytes.NewReader(encoded), protocolexperiment.DefaultDecodeLimit)
			require.NoError(t, err)

			test.mutate(&view)
			require.ErrorContains(t, view.ValidateAgainst(firstOrder), test.errorMatch)
		})
	}
}
