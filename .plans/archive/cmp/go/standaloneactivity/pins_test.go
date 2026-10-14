package standaloneactivity

import (
	"testing"

	"github.com/stretchr/testify/require"

	"go.temporal.io/umpire/model/umpire"
)

// What the standalone activity Model says, pinned the way the Nexus pins are.

// ### The product machine

func TestActivityProduct(t *testing.T) {
	table := ActivityProduct.Check(t)

	// Nine phases, and the five the design ends on.
	require.Len(t, table.States, 9)
	require.Len(t, table.Ends, 5)
	require.ElementsMatch(t, table.States, table.Reachable)
	require.Nil(t, table.Stuck)
}

// ### The protocol machine

func TestActivityProtocol(t *testing.T) {
	table := ActivityProtocol.Check(t)

	// Twelve phases, three attempt counts and three deadlines, and the five phases the design ends on.
	require.Len(t, table.States, 12*(attemptBound+1)*2*2*2)
	require.Len(t, table.Ends, 5*(attemptBound+1)*2*2*2)

	// A worker cancels only an attempt whose cancel was requested.
	require.Empty(t, protocolAttemptResultStep(ProtocolState{Phase: Started}, AttemptCanceled{}))

	// A retryable failure yields to whatever the caller asked for meanwhile.
	yields := func(phase Phase) Phase {
		steps := protocolAttemptResultStep(ProtocolState{Phase: phase, Attempts: 1}, AttemptFailed{Retryable: true})
		require.Len(t, steps, 1)
		return steps[0].State.Phase
	}
	require.Equal(t, BackingOff, yields(Started))
	require.Equal(t, Canceled, yields(CancelRequested))
	require.Equal(t, Paused, yields(PauseRequested))

	require.Nil(t, table.Stuck)
}

// ### The refinement

func TestActivityProtocolRefinesProduct(t *testing.T) {
	report := ActivityProtocol.Refinement(t)
	require.Nil(t, report.Rejected)
}

// ### The Queries

func TestFunctionalQueriesFind(t *testing.T) {
	for _, q := range StandaloneActivityTests.Queries {
		t.Run(q.Name, func(t *testing.T) {
			answer := q.Answer(t)
			require.True(t, answer.Found, answer.Explanation)
		})
	}
}

func TestVerifyQueriesHold(t *testing.T) {
	for _, q := range []*umpire.Query{terminalHolds, pauseHolds, stoppedWorkerStartsNothing} {
		t.Run(q.Name, func(t *testing.T) {
			answer := q.Answer(t)
			require.True(t, answer.Verified, answer.Explanation)
		})
	}
}

func TestDeclarationsCheck(t *testing.T) {
	umpire.Check(t,
		ActivityProduct, ActivityProtocol, ActivityWorker, StandaloneActivity,
		StandaloneActivityTests, StandaloneActivityCanary, StandaloneActivityExploration,
	)
}
