package standaloneactivity

// What the standalone activity Model says, pinned the way .plans/cmp/lean/ActivityPins.lean pins
// it. Each assertion cites the Lean pin it translates. That file was never compiled: Lean refuses the
// 288-state protocol machine, so only its product-machine pins have a Lean answer, compared in
// parity/. Everything else here is Go's own answer.

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.temporal.io/server/model/go/umpire"
)

func table(t *testing.T, m umpire.Model) *umpire.Table {
	t.Helper()
	tb, err := m.Table()
	require.NoError(t, err)
	return tb
}

func answer(t *testing.T, q *umpire.Query) umpire.Answer {
	t.Helper()
	a, err := q.Answer()
	require.NoError(t, err)
	return a
}

func at(phase Phase, attempts Attempts, opts ...func(*ProtocolState)) ProtocolState {
	s := ProtocolState{Phase: phase, Attempts: attempts, ScheduleToClose: Unset, ScheduleToStart: Unset,
		StartToClose: Unset}
	for _, o := range opts {
		o(&s)
	}
	return s
}

func sts(t Timeout) func(*ProtocolState) { return func(s *ProtocolState) { s.ScheduleToStart = t } }
func stc(t Timeout) func(*ProtocolState) { return func(s *ProtocolState) { s.StartToClose = t } }

func phases(steps []ProtocolStep) []Phase {
	var out []Phase
	for _, s := range steps {
		out = append(out, s.State.Phase)
	}
	return out
}

// ### The product machine

func TestProductMachine(t *testing.T) {
	tb := table(t, ActivityProduct)
	// Nine phases, and the five the design ends on. (ActivityPins.lean:21-22)
	require.Len(t, tb.States, 9)
	require.Len(t, tb.Ends, 5)

	// A canceled answer settles only an activity whose cancellation was requested. (:25-26)
	require.Empty(t, attemptResultStep(ProductState{ProductStarted}, CanceledByRun{}))
	require.Equal(t, ProductCanceled, attemptResultStep(ProductState{ProductCancelRequested}, CanceledByRun{})[0].State.Phase)

	// Unlike the Nexus product, a retry is visible: the caller reads scheduled again. (:29-32)
	require.Equal(t, ProductScheduled, attemptResultStep(ProductState{ProductStarted}, Failed{Retryable: true})[0].State.Phase)
	require.Equal(t, ProductCanceled, attemptResultStep(ProductState{ProductCancelRequested}, Failed{Retryable: true})[0].State.Phase)

	// A paused activity is dispatched to no worker: no product row leaves paused for started.
	// (:35-36)
	for _, r := range tb.Rows {
		if r.Source == "paused" {
			for _, res := range r.Results {
				require.NotEqual(t, "started", res.State, r.Key)
			}
		}
	}
	require.Empty(t, tb.Stuck) // (:40)
}

// ### The protocol machine

func TestProtocolMachine(t *testing.T) {
	tb := table(t, ActivityProtocol)
	// Twelve phases, three attempt counts and three deadlines: 288 states, past Lean's bound of 256.
	// (:51-52)
	require.Len(t, tb.States, 12*(AttemptBound+1)*2*2*2)
	require.Len(t, tb.Ends, 5*(AttemptBound+1)*2*2*2)

	// Eight start requests, one attempt start, four answers, four controls, the fault and four
	// timers. (:55)
	require.Len(t, tb.Actions, 8+1+4+4+1+4)
	require.Equal(t, []string{umpire.KeyOf(at(Unstarted, 0))}, tb.Starts) // (:57)

	// A retryable failure backs the attempt off and is read as scheduled again with the count
	// raised. (:60-61)
	require.Equal(t, []ProtocolStep{{Outcome: Accepted, State: at(BackingOff, 1),
		Facts:   []ProtocolFact{StatusScheduled{}, AttemptCountRead{}},
		Because: "a retryable failure backs off; the caller reads scheduled again"}},
		protocolAttemptResultStep(at(Started, 1), Failed{Retryable: true}))

	// Under a cancel request the same failure settles the activity as canceled; under a pause
	// request it lands in paused. (:64-70)
	require.Equal(t, []Phase{PhaseCanceled}, phases(protocolAttemptResultStep(at(CancelRequested, 1), Failed{Retryable: true})))
	require.Equal(t, []Phase{Paused}, phases(protocolAttemptResultStep(at(PauseRequested, 1), Failed{Retryable: true})))

	// A pause of a held attempt is a request; of a scheduled one it takes effect at once. (:73-74)
	require.Equal(t, []Phase{PauseRequested}, phases(protocolControlStep(at(Started, 1), Pause)))
	require.Equal(t, []Phase{Paused}, phases(protocolControlStep(at(Scheduled, 0), Pause)))

	// A control on an activity that is over is not found. (:77-78)
	require.Equal(t, []ProtocolStep{{Outcome: ControlLost, State: at(PhaseCompleted, 1)}},
		protocolControlStep(at(PhaseCompleted, 1), Terminate))

	// Each deadline covers its own span. (:81-85)
	require.Empty(t, startToCloseStep(at(Scheduled, 0, stc(Expires))))
	require.Equal(t, []Phase{TimedOut}, phases(startToCloseStep(at(PauseRequested, 1, stc(Expires)))))
	require.Equal(t, []ProtocolFact{StatusTimedOut{TypeScheduleToStart}},
		scheduleToStartStep(at(BackingOff, 1, sts(Expires)))[0].Facts)

	require.Empty(t, tb.Stuck) // (:87)
}

// ### The refinement

func TestRefinement(t *testing.T) {
	ref, err := ActivityProtocol.Refinement()
	require.NoError(t, err)                                           // (:96)
	require.Len(t, ref.Rows, resultCount(table(t, ActivityProtocol))) // (:97)
	lookup := func(key string) *string {
		for _, r := range ref.Rows {
			if r.Key == key {
				return r.Product
			}
		}
		t.Fatalf("no refinement row %s", key)
		return nil
	}
	// The visible retry is the product's retryable-failure row; the pause request is a stutter; the
	// unpause of a requested pause is a stutter too. (:101-105)
	require.Equal(t, "attemptResult-failed-true", *lookup("started-1-unset-unset-unset-attemptResult-failed-true"))
	require.Nil(t, lookup("started-1-unset-unset-unset-control-pause"))
	require.Nil(t, lookup("pauseRequested-1-unset-unset-unset-control-unpause"))
	// A retryable failure under a pause request is the product's pause. (:108-109)
	require.Equal(t, "control-pause", *lookup("pauseRequested-1-unset-unset-unset-attemptResult-failed-true"))
}

func resultCount(t *umpire.Table) int {
	n := 0
	for _, r := range t.Rows {
		n += len(r.Results)
	}
	return n
}

// ### The Queries

func TestQueries(t *testing.T) {
	// (:113-120)
	for _, q := range FunctionalQueries {
		require.Equal(t, umpire.Found, answer(t, q).Outcome, q.Name)
	}
	// (:121-123)
	require.Equal(t, umpire.VerifiedWithinLimits, answer(t, TerminalHolds).Outcome)
	require.Equal(t, umpire.VerifiedWithinLimits, answer(t, PauseHolds).Outcome)
	stopped := answer(t, StoppedWorkerStartsNothing)
	require.Equal(t, umpire.VerifiedWithinLimits, stopped.Outcome)
	// Not vacuous: the scenario performs an attempt start while the worker polls, so the claim is
	// exercised, not merely never contradicted. (:126)
	require.True(t, stopped.Exercised)
}

// Every declaration passes the checks the Lean elaborator runs, and the canary admits only paths
// whose every step records evidence.
func TestDeclarationsCheck(t *testing.T) {
	require.NoError(t, umpire.Check(ActivityProduct, ActivityProtocol, ActivityWorker, StandaloneActivity,
		StandaloneActivityTests, StandaloneActivityCanary, StandaloneActivityExploration, TerminalHolds,
		PauseHolds, StoppedWorkerStartsNothing))
}

// Lean pins with no Go counterpart: assert_axioms [activityProduct] and [activityProtocol]
// (ActivityPins.lean:38, :89), kernel axiom inventories Go has no kernel for.
