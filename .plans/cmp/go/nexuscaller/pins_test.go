package nexuscaller

import (
	"testing"

	"github.com/stretchr/testify/require"

	"go.temporal.io/umpire/model/common"
	"go.temporal.io/umpire/model/umpire"
)

// What the Caller Model says. The claims below are about the tables, because the tables are what
// Search, the Behavior Fingerprint and Contract lowering read. A switch arm that stopped saying what
// it says would fail here. Lean runs these as #guard while it elaborates the Model; Go runs them as a
// white-box test in the Model's package, so `go test ./...` is the edit-to-feedback loop.

// ### The product machine

func TestNexusProduct(t *testing.T) {
	table := NexusProduct.Check(t)

	// Six phases, and the four the design ends on.
	require.Len(t, table.States, 6)
	require.Len(t, table.Ends, 4)

	// Every action class the machine steps on: six replies, three resolutions, the two faults it cannot
	// see, and the one timer.
	require.Len(t, table.ActionKeys, 6+3+1+1+1)

	// A retryable handler error is invisible here: it is the protocol machine that backs off.
	require.Empty(t, handlerReplyStep(ProductState{Phase: ProductScheduled}, HandlerError{Retryable: true}))

	// What the Model actually reaches: every phase.
	require.ElementsMatch(t, table.States, table.Reachable)
	require.Nil(t, table.Stuck)
}

// ### The protocol machine

func TestNexusProtocol(t *testing.T) {
	table := NexusProtocol.Check(t)

	// Eight phases, three attempt counts and three deadlines, and the four phases the design ends on.
	require.Len(t, table.States, 8*(attemptBound+1)*2*2*2)
	require.Len(t, table.Ends, 4*(attemptBound+1)*2*2*2)

	// Every action class: eight schedule commands, one per assignment of the three deadlines, six
	// replies, three resolutions, the two faults and the four timers.
	require.Len(t, table.ActionKeys, 8+6+3+1+1+4)

	// The machine begins before the operation exists, with every deadline at its first value.
	require.Equal(t, []ProtocolState{unscheduled}, NexusProtocol.Starts)

	// A retryable handler error backs the operation off and raises the attempt count. No history event
	// records that, which is why its evidence is the derived pendingAttempts observation.
	require.Equal(t,
		[]ProtocolStep{{
			Outcome: Accepted,
			State:   ProtocolState{Phase: BackingOff, Attempts: 1},
			Facts:   []ProtocolFact{PendingAttempts{}},
		}},
		protocolHandlerReplyStep(ProtocolState{Phase: Scheduled}, HandlerError{Retryable: true}))

	// The count saturates rather than wrapping.
	saturated := protocolHandlerReplyStep(ProtocolState{Phase: Scheduled, Attempts: attemptBound}, HandlerError{Retryable: true})
	require.Len(t, saturated, 1)
	require.EqualValues(t, attemptBound, saturated[0].State.Attempts)

	// The handler's worker stopping keeps the state and records nothing.
	withDeadline := ProtocolState{Phase: Scheduled, ScheduleToStart: common.Expires}
	require.Equal(t, []ProtocolStep{{Outcome: Accepted, State: withDeadline}}, protocolWorkerStopStep(withDeadline))

	// Nothing is stuck: the operation's own phase decides which steps are enabled.
	require.Nil(t, table.Stuck)
}

// ### The refinement

func TestNexusProtocolRefinesProduct(t *testing.T) {
	report := NexusProtocol.Refinement(t)
	require.Nil(t, report.Rejected)
	require.Len(t, report.Rows, len(NexusProtocol.Check(t).Rows))
}

// ### The Queries

func TestFunctionalQueriesFind(t *testing.T) {
	for _, q := range NexusCallerTests.Queries {
		t.Run(q.Name, func(t *testing.T) {
			answer := q.Answer(t)
			require.True(t, answer.Found, answer.Explanation)
		})
	}
}

func TestVerifyQueriesHold(t *testing.T) {
	for _, q := range []*umpire.Query{terminalHolds, stoppedWorkerRepliesNothing} {
		t.Run(q.Name, func(t *testing.T) {
			answer := q.Answer(t)
			require.True(t, answer.Verified, answer.Explanation)
		})
	}
}

// ### Everything else Lean checks at elaboration

func TestDeclarationsCheck(t *testing.T) {
	umpire.Check(t,
		NexusProduct, NexusProtocol, HandlerWorker, NexusCaller,
		NexusCallerTests, NexusCallerCanary, NexusCallerExploration,
	)
}
