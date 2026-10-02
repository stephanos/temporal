package nexuscaller

// What the Caller Model says, pinned the way model/lean/Temporal/Feature/Nexus/Caller/Tests.lean pins
// it. Each assertion cites the Lean pin it translates. Pins about Lean internals with no Go
// counterpart (assert_axioms, the elaborated Property groups, the reference backend's path counts)
// are listed at the end of the file with the reason; the Case pins live with the Case producer.

import (
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"go.temporal.io/server/model/go/umpire"
	"go.temporal.io/server/model/go/worker"
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

// at is a state written the way a reader names one: the phase, and whichever fields are not at the
// value the operation begins with (Tests.lean `at'`).
func at(phase Phase, opts ...func(*ProtocolState)) ProtocolState {
	s := ProtocolState{Phase: phase, ScheduleToClose: Unset, ScheduleToStart: Unset, StartToClose: Unset}
	for _, o := range opts {
		o(&s)
	}
	return s
}

func attempts(n Attempts) func(*ProtocolState) { return func(s *ProtocolState) { s.Attempts = n } }
func sts(t Timeout) func(*ProtocolState)       { return func(s *ProtocolState) { s.ScheduleToStart = t } }
func stc(t Timeout) func(*ProtocolState)       { return func(s *ProtocolState) { s.StartToClose = t } }

// ### The product machine

func TestProductMachine(t *testing.T) {
	tb := table(t, NexusProduct)

	// Six phases, and the four the design ends on. (Tests.lean:23-24)
	require.Len(t, tb.States, 6)
	require.Len(t, tb.Ends, 4)

	// Every action class the machine steps on: six replies, three resolutions, the two faults it
	// cannot see, and the one timer. (:28)
	require.Len(t, tb.Actions, 12)

	// A retryable handler error is invisible here: it is the protocol machine that backs off. (:31)
	require.Empty(t, handlerReplyStep(ProductState{ProductScheduled}, HandlerError{Retryable: true}))

	// What the Model actually reaches: every phase. (:34-36)
	require.Equal(t, []string{"scheduled", "canceled", "failed", "succeeded", "started", "timedOut"}, tb.Reachable)

	// (:40)
	require.Empty(t, tb.Stuck)
}

// ### The protocol machine

func TestProtocolMachine(t *testing.T) {
	tb := table(t, NexusProtocol)

	// Eight phases, three attempt counts and three deadlines, and the four phases the design ends on.
	// (:52-53)
	require.Len(t, tb.States, 8*(AttemptBound+1)*2*2*2)
	require.Len(t, tb.Ends, 4*(AttemptBound+1)*2*2*2)

	// Eight schedule commands, six replies, three resolutions, the two faults and the four timers.
	// The catalog is in canonical order, so it opens on the backoff timer. (:58-59)
	require.Len(t, tb.Actions, 8+6+3+1+1+4)
	require.Equal(t, []string{"backoff", "complete-canceled"}, tb.Actions[:2])

	// The machine begins before the operation exists, with every deadline at its first value. (:62)
	require.Equal(t, []string{umpire.KeyOf(at(Unscheduled))}, tb.Starts)

	// (:65) The machine carries no setup parameter: Go machines have none, so there is nothing to pin.

	// A retryable handler error backs the operation off and raises the attempt count. (:69-70)
	require.Equal(t, []ProtocolStep{{Outcome: Accepted, State: at(BackingOff, attempts(1)),
		Facts: []ProtocolFact{PendingAttemptsRead{}}}},
		protocolHandlerReplyStep(at(Scheduled), HandlerError{Retryable: true}))

	// The count saturates rather than wrapping. (:73-74)
	saturated := protocolHandlerReplyStep(at(Scheduled, attempts(AttemptBound)), HandlerError{Retryable: true})
	require.Len(t, saturated, 1)
	require.Equal(t, Attempts(AttemptBound), saturated[0].State.Attempts)

	// A completion before the start records the Started event first, one after it does not.
	// (:78-81)
	require.Equal(t, []ProtocolFact{NexusOperationStarted{}, NexusOperationCompleted{}},
		protocolCompleteStep(at(BackingOff, attempts(1)), ResolvedSucceeded)[0].Facts)
	require.Equal(t, []ProtocolFact{NexusOperationCompleted{}},
		protocolCompleteStep(at(Started), ResolvedSucceeded)[0].Facts)

	// A completion after the operation is over is not found and changes nothing. (:84-85)
	require.Equal(t, []ProtocolStep{{Outcome: CompletionMissing, State: at(TimedOut)}},
		protocolCompleteStep(at(TimedOut), ResolvedSucceeded))

	// A timer fires only when the schedule command set it, and each covers its own span. (:88-90)
	require.Empty(t, startToCloseStep(at(Scheduled, stc(Expires))))
	require.Equal(t, TimedOut, startToCloseStep(at(Started, stc(Expires)))[0].State.Phase)
	require.Empty(t, scheduleToCloseStep(at(Started)))

	// Which timer fired is recorded. (:93-94)
	require.Equal(t, []ProtocolFact{NexusOperationTimedOut{TypeScheduleToStart}},
		scheduleToStartStep(at(Scheduled, sts(Expires)))[0].Facts)

	// The handler's worker stopping keeps the state and records nothing; the product machine does
	// not see it at all. (:98-100)
	require.Equal(t, []ProtocolStep{{Outcome: Accepted, State: at(Scheduled, sts(Expires))}},
		protocolWorkerStopStep(at(Scheduled, sts(Expires))))
	require.Empty(t, workerStopStep(ProductState{ProductScheduled}))

	// Nothing is stuck. (:103)
	require.Empty(t, tb.Stuck)

	// Not every state is reachable. The Behavior Fingerprint reads the table, so this number is part
	// of the Model's identity. (:108)
	require.Len(t, tb.Reachable, 158)
}

// ### The refinement

func TestRefinement(t *testing.T) {
	ref, err := NexusProtocol.Refinement()
	require.NoError(t, err) // (:118)
	tb := table(t, NexusProtocol)
	require.Len(t, ref.Rows, len(tb.Rows)) // (:119)

	lookup := func(key string) (string, bool) {
		for _, r := range ref.Rows {
			if r.Key == key {
				if r.Product == nil {
					return "", true
				}
				return *r.Product, true
			}
		}
		t.Fatalf("no refinement row %s", key)
		return "", false
	}

	// A reply the product machine sees is that reply's step. A retry it cannot see is a stutter, and
	// so are the schedule command and the backoff timer. (:123-129)
	require.Equal(t, "handlerReply-async", must(lookup("scheduled-0-unset-unset-unset-handlerReply-async")))
	require.Empty(t, must(lookup("scheduled-0-unset-unset-unset-handlerReply-handlerError-true")))
	require.Empty(t, must(lookup("unscheduled-0-unset-unset-unset-schedule-unset-unset-expires")))
	require.Empty(t, must(lookup("backingOff-1-unset-unset-unset-backoff")))

	// A deadline firing is the product's one timer, whichever deadline it was. (:132-133)
	require.Equal(t, "timeout", must(lookup("started-0-unset-unset-expires-startToClose")))

	// A completion before the start records the Started event first and still carries the step.
	// (:136-137)
	require.Equal(t, "complete-succeeded", must(lookup("backingOff-1-unset-unset-unset-complete-succeeded")))

	// The rows the product machine does not see: every schedule command, every retry, every backoff
	// and every worker stop. (:141)
	var stutters []umpire.RefinementRow
	for _, r := range ref.Rows {
		if r.Product == nil {
			stutters = append(stutters, r)
		}
	}
	require.Len(t, stutters, 24*8+24+24+24+192)

	// Stutter invariance: every stutter leaves a phase that reads as scheduled, or is a worker stop.
	// (:153-155)
	for _, r := range stutters {
		phase, _, _ := strings.Cut(r.Key, "-")
		require.True(t, slices.Contains([]string{"unscheduled", "scheduled", "backingOff"}, phase) ||
			strings.HasSuffix(r.Key, "-workerStop"), r.Key)
	}

	// The product state a protocol state reads as is a field named after the product machine.
	// (:159-160)
	require.Equal(t, []string{"phase", "attempts", "scheduleToClose", "scheduleToStart", "startToClose",
		"nexusProduct"}, tb.StateFields)
}

func must(v string, ok bool) string {
	if !ok {
		panic("missing")
	}
	return v
}

// ### The Properties and the Queries

func TestQueries(t *testing.T) {
	// A protocol Scenario names its classed actions with their inputs, and its start by its phase.
	// (:170-172)
	require.Equal(t, "unscheduled-0-unset-unset-unset", AsyncThenSucceeded.Start)
	require.Equal(t, []string{"schedule-unset-unset-unset", "handlerReply-async", "complete-succeeded"},
		AsyncThenSucceeded.Actions)

	// Each functional Query finds its claim on its path. (:176-196)
	for _, q := range FunctionalQueries {
		require.Equal(t, umpire.Found, answer(t, q).Outcome, q.Name)
	}

	// A timer is named like any action, and a Scenario lists it where it fires. (:199-203)
	require.Equal(t, []string{"schedule-unset-unset-unset", "handlerReply-handlerError-true", "backoff",
		"handlerReply-syncSuccess"}, RetriedThenSucceeded.Actions)
	require.Equal(t, []string{"schedule-unset-expires-unset", "workerStop", "scheduleToStart"},
		ScheduleToStartExpires.Actions)

	// The product claim is verified over every trace of the asynchronous path, and keeps its own
	// identity: it is the product Property and no other. (:207-211)
	require.Equal(t, umpire.VerifiedWithinLimits, answer(t, TerminalHolds).Outcome)
	require.Equal(t, "terminalIsFinal", TerminalHolds.Property.Name)
	require.Equal(t, NexusProduct.Name(), TerminalHolds.Property.Machine.Name())
}

// The product Property over every trace of the protocol machine within four. (:228-230)
//
// Both searches verify it; they count different product states. Lean lowers terminalIsFinal into
// four clause groups, one per terminal phase, and its product state keeps a fired bit per group, so
// Veil visits 171 states. The Go monitor keeps one fired bit per Property and visits 111. The count
// is a search statistic: fn-88's own backend differential exempts `explored` from comparison.
func TestTerminalHoldsEverywhere(t *testing.T) {
	everywhere := NexusProtocol.Scenario("everywhere").Starts(unscheduled).Free().
		VerifyRefined("terminalHoldsEverywhere", TerminalIsFinal, NexusProtocol.Via(NexusProduct), Four)
	a := answer(t, everywhere)
	require.Equal(t, umpire.VerifiedWithinLimits, a.Outcome)
	require.Equal(t, 111, a.Explored)
}

// A product Property about an action the protocol machine does not have cannot be read there.
// (:233-245)
func TestProductPropertyOnMissingAction(t *testing.T) {
	timesOut := NexusProduct.Property("timesOut").
		When(timeout.With()).
		Holds(func(s ProductStep) bool { return s.State.Phase == ProductTimedOut })
	q := AsyncThenSucceeded.VerifyRefined("timesOutOnProtocol", timesOut, NexusProtocol.Via(NexusProduct), Three)
	_, err := q.Answer()
	require.EqualError(t, err, "query timesOutOnProtocol: the Property names the action 'timeout' of "+
		"'nexusProduct', and 'nexusProtocol' has no action of that name; a Property on the refined "+
		"machine is read on the refining one through the values of the same name, and a state "+
		"through its map")
}

// ### The sets

func TestSets(t *testing.T) {
	// (:254-256)
	require.Equal(t, umpire.Canary, NexusCallerCanary.Purpose)
	require.Equal(t, map[umpire.Party]umpire.Binding{Caller: umpire.Driven, Handler: umpire.Observed,
		Network: umpire.Observed, worker.Party: umpire.Driven}, NexusCallerCanary.Bindings)
	require.NoError(t, umpire.Check(NexusCallerCanary))

	// A Query whose path takes a silent step carries a capability gap no deployment closes, so a
	// canary naming it is rejected, naming the Query and the gap. (:262-277)
	canaryRetry := &umpire.Set{Name: "canaryRetry", Purpose: umpire.Canary,
		Bindings: NexusCallerCanary.Bindings, Queries: []*umpire.Query{Retry}}
	require.EqualError(t, umpire.Check(canaryRetry),
		"set canaryRetry: retry takes the silent step backoff, a gap no deployment closes")

	// The exploration covers the protocol machine under four. (:282-306)
	require.Equal(t, umpire.Exploratory, NexusCallerExploration.Purpose)
	require.Equal(t, "nexusProtocol", NexusCallerExploration.Machine.Name())
	require.Equal(t, []umpire.CoverageGoal{umpire.CoverRows, umpire.CoverResults, umpire.CoverClassMembers},
		NexusCallerExploration.Cover)
	require.Equal(t, "four", NexusCallerExploration.Budget.Name)
	require.Len(t, table(t, NexusProtocol).Rows, 1152)
	targets, err := NexusCallerExploration.Targets()
	require.NoError(t, err)
	require.Len(t, targets, 885+2+2)
	var kinds, results []string
	var rows int
	for _, tg := range targets {
		if !slices.Contains(kinds, tg.Kind) {
			kinds = append(kinds, tg.Kind)
		}
		switch tg.Kind {
		case "row":
			rows++
		case "result":
			results = append(results, tg.Outcome)
		default:
		}
	}
	require.Equal(t, []string{"row", "result", "classMember"}, kinds)
	require.Equal(t, 885, rows)
	require.LessOrEqual(t, len(targets), Four.Search)
	require.Equal(t, []string{"temporal.nexus.caller.outcome.nexusProtocol.accepted",
		"temporal.nexus.caller.outcome.nexusProtocol.notFound"}, results)
	require.Equal(t, umpire.CoverageTarget{Kind: "classMember",
		Member: "temporal.nexus.caller.action.nexusProtocol.handlerReply-handlerError-false",
		Action: "temporal.nexus.caller.action.handlerReply", Field: "reply",
		Class: "handlerError (retryable := false)", Example: "BadRequest"}, targets[len(targets)-2])
}

// ### The composition

func TestComposition(t *testing.T) {
	// (:435)
	require.Equal(t, []string{"serve", "workerStop"}, table(t, HandlerWorker).Actions)

	tb := table(t, NexusCaller)
	// The composition reaches every reachable protocol state under both worker phases. (:437-439)
	require.Len(t, tb.States, 316)
	stopped := 0
	for _, s := range tb.States {
		if strings.HasSuffix(s, "_stopped") {
			stopped++
		}
	}
	require.Equal(t, 158, stopped)
	require.Len(t, tb.Rows, 1468)
	require.Equal(t, []string{"handlerReply-async", "handlerReply-handlerError-false",
		"handlerReply-handlerError-true", "handlerReply-operationCanceled", "handlerReply-operationFailed",
		"handlerReply-syncSuccess", "operation_backoff", "operation_complete-canceled",
		"operation_complete-failed", "operation_complete-succeeded",
		"operation_schedule-expires-expires-expires", "operation_schedule-expires-expires-unset",
		"operation_schedule-expires-unset-expires", "operation_schedule-expires-unset-unset",
		"operation_schedule-unset-expires-expires", "operation_schedule-unset-expires-unset",
		"operation_schedule-unset-unset-expires", "operation_schedule-unset-unset-unset",
		"operation_scheduleToClose", "operation_scheduleToStart", "operation_startToClose",
		"operation_transportFault", "workerStop"}, tb.Actions) // (:440-450)

	// A reply has a row only where the worker polls. (:453-456)
	replies := 0
	for _, r := range tb.Rows {
		if strings.Contains(r.Key, "-handlerReply") {
			replies++
			require.NotContains(t, r.Key, "_stopped-")
		}
	}
	require.Equal(t, 144, replies)

	// Verified over the path: the one reply comes before the stop. (:470-472)
	require.Equal(t, umpire.VerifiedWithinLimits, answer(t, StoppedWorkerRepliesNothing).Outcome)
}

// Every declaration passes the checks the Lean elaborator runs.
func TestDeclarationsCheck(t *testing.T) {
	require.NoError(t, umpire.Check(NexusProduct, NexusProtocol, HandlerWorker, NexusCaller,
		NexusCallerTests, NexusCallerCanary, NexusCallerExploration, TerminalHolds,
		StoppedWorkerRepliesNothing))
}

// Lean pins with no Go counterpart, and why:
//
//   - assert_axioms [nexusProduct], [nexusProtocol], [nexusProtocol.refines], [handlerWorker,
//     nexusCaller], [nexusCaller.agrees] (:38, :110, :162, :458-459): kernel axiom inventories. Go
//     has no kernel; the equivalent claims are the table, refinement and composition tests above.
//   - terminalIsFinal.names.groups.length == 4 (:166) and repliedByPollingWorker.names.groups
//     (:463-467): the Lean elaborator lowers a predicate into clause groups by evaluating it on
//     every state. Go keeps the predicate as a function, so there are no groups to count.
//   - asyncThenSucceeded.names.setupState / occurrences (:170-172): translated above as the
//     Scenario's start and action keys.
//   - The reference backend's path counts, "4 paths" and "3525 paths" (:223-230): Go has one
//     backend, the breadth-first product search; its product-state counts are pinned above.
//   - nexusCallerCanaryCases white-box gaps, instructionIds and declaredKinds (:257, :310-360):
//     Case production pins. case_test.go checks every produced Case byte for byte against the
//     checked-in fixture, which carries every instruction id, evidence kind and Known Gap these pins
//     read.
