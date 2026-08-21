package protocol

import (
	"bytes"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDefaultAttemptExecutionViewResolvesExportedNexusExperiment(t *testing.T) {
	t.Parallel()

	encoded, err := os.ReadFile("../testdata/nexus-cancellation.json")
	require.NoError(t, err)
	experiment, err := DecodeExperiment(bytes.NewReader(encoded), DefaultDecodeLimit)
	require.NoError(t, err)

	view, found, err := DefaultAttemptExecutionView(experiment)
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, TargetIDNexusCancellation, view.Attempts.Target)
}

func TestAttemptViewSeparatesLiveOutcomeFromAbstractTransitions(t *testing.T) {
	t.Parallel()

	sound, found, err := DefaultFirstOrderView(TargetIDNexusCancellation, "sound")
	require.NoError(t, err)
	require.True(t, found)
	attempts, found, err := DefaultAttemptView(TargetIDNexusCancellation, "sound")
	require.NoError(t, err)
	require.True(t, found)

	path := []AttemptRequest{
		{Action: ActionKindScheduleOperation},
		{Action: ActionKindDispatchTask},
		{Action: ActionKindRequestCancellation},
		{Action: ActionKindCommitCancellation},
		{Action: ActionKindAcquireOwnership},
		{Action: ActionKindRetryTask},
		{Action: ActionKindWorkerReturnsSuccess},
		{Action: ActionKindPersistSuccess},
	}
	replay, err := attempts.Replay(sound, path)
	require.NoError(t, err)
	require.True(t, replay.Accepted)
	require.Contains(t, replay.LiveOnlyActions, ActionKindScheduleOperation)
	require.Contains(t, replay.LiveOnlyActions, ActionKindRetryTask)

	path[len(path)-1].Outcomes = []ActionOutcome{ActionOutcomeApplied}
	replay, err = attempts.Replay(sound, path)
	require.NoError(t, err)
	require.False(t, replay.Accepted)
	require.Equal(t, ActionKindPersistSuccess, replay.RejectedAction)
	require.Equal(t, ActionOutcomeApplied, replay.RejectedOutcome)
}

func TestAttemptViewReplaysEveryDeclaredNonAppliedOutcomeAsGuardedStutter(t *testing.T) {
	t.Parallel()

	view, found, err := DefaultFirstOrderView(TargetIDNexusCancellation, "sound")
	require.NoError(t, err)
	require.True(t, found)
	attempts, found, err := DefaultAttemptView(TargetIDNexusCancellation, "sound")
	require.NoError(t, err)
	require.True(t, found)

	prefix := []ObservedAttempt{
		{Action: ActionKindScheduleOperation, Outcome: ActionOutcomeApplied},
		{Action: ActionKindDispatchTask, Outcome: ActionOutcomeApplied},
		{Action: ActionKindRequestCancellation, Outcome: ActionOutcomeApplied},
		{Action: ActionKindCommitCancellation, Outcome: ActionOutcomeApplied},
		{Action: ActionKindAcquireOwnership, Outcome: ActionOutcomeApplied},
		{Action: ActionKindWorkerReturnsSuccess, Outcome: ActionOutcomeApplied},
	}
	for _, outcome := range []ActionOutcome{
		ActionOutcomeSuppressed,
		ActionOutcomeRejected,
		ActionOutcomeRetried,
		ActionOutcomeFaultIntercepted,
	} {
		replay, err := attempts.ReplayObserved(view, append(prefix, ObservedAttempt{
			Action: ActionKindPersistSuccess, Outcome: outcome,
		}))
		require.NoError(t, err)
		require.True(t, replay.Accepted, outcome)
	}

	replay, err := attempts.ReplayObserved(view, append(prefix, ObservedAttempt{
		Action: ActionKindPersistSuccess, Outcome: ActionOutcomeApplied,
	}))
	require.NoError(t, err)
	require.False(t, replay.Accepted)
}

func TestMutatedAttemptViewAcceptsAppliedStalePersistence(t *testing.T) {
	t.Parallel()

	view, found, err := DefaultFirstOrderView(TargetIDNexusCancellation,
		"stale-completion-guard-removed")
	require.NoError(t, err)
	require.True(t, found)
	attempts, found, err := DefaultAttemptView(TargetIDNexusCancellation,
		"stale-completion-guard-removed")
	require.NoError(t, err)
	require.True(t, found)

	replay, err := attempts.ReplayObserved(view, []ObservedAttempt{
		{Action: ActionKindScheduleOperation, Outcome: ActionOutcomeApplied},
		{Action: ActionKindDispatchTask, Outcome: ActionOutcomeApplied},
		{Action: ActionKindRequestCancellation, Outcome: ActionOutcomeApplied},
		{Action: ActionKindCommitCancellation, Outcome: ActionOutcomeApplied},
		{Action: ActionKindAcquireOwnership, Outcome: ActionOutcomeApplied},
		{Action: ActionKindWorkerReturnsSuccess, Outcome: ActionOutcomeApplied},
		{Action: ActionKindPersistSuccess, Outcome: ActionOutcomeApplied},
	})
	require.NoError(t, err)
	require.True(t, replay.Accepted)
}

func TestAttemptViewRejectsIncompleteOrUnboundMappings(t *testing.T) {
	t.Parallel()

	for _, test := range []struct {
		name       string
		mutate     func(*AttemptView)
		errorMatch string
	}{
		{
			name: "missing action mapping",
			mutate: func(view *AttemptView) {
				view.Attempts = view.Attempts[:len(view.Attempts)-1]
			},
			errorMatch: "do not exactly cover",
		},
		{
			name: "unknown abstract transition",
			mutate: func(view *AttemptView) {
				view.Attempts[0].Outcomes[0].Transitions = []ActionKind{"unknown-transition"}
			},
			errorMatch: "unknown abstract transition",
		},
		{
			name: "mismatched first-order hash",
			mutate: func(view *AttemptView) {
				view.FirstOrderSemanticHash =
					"sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
			},
			errorMatch: "identity does not match",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			firstOrder, found, err := DefaultFirstOrderView(TargetIDNexusCancellation, "sound")
			require.NoError(t, err)
			require.True(t, found)
			view, found, err := DefaultAttemptView(TargetIDNexusCancellation, "sound")
			require.NoError(t, err)
			require.True(t, found)
			encoded, err := view.CanonicalJSON()
			require.NoError(t, err)
			view, err = DecodeAttemptView(bytes.NewReader(encoded), DefaultDecodeLimit)
			require.NoError(t, err)

			test.mutate(&view)
			require.ErrorContains(t, view.ValidateAgainst(firstOrder), test.errorMatch)
		})
	}
}
