package model

// The Nexus caller close and reset designs, lifted from
// features/nexuscaller/system/ClosePolicy.scala into ir/nexus-close.json and checked here through
// Check alone. What each test expects is the trace oracle of model/specimens/nexus.md it names, in
// the keys of the lifted Model. A state key spells caller, cancel intent, handler, channel,
// retained, known: the specimen's six fields, with the cancel intent carrying its principal and the
// handler its receipt of the cancel request.
//
// Every claim here is an authored design promise: no server has a close policy, an operation-level
// retention or a reset that reapplies it. nexus_close_baseline_test.go compares the one behavior the Go
// baseline also has, and lists which claims are which.
//
// No expectation here is an explored-state count: the specimen's counts are those of its sketch, a
// lifter fixture fn-114.10 retired once this Model carried each of its designs.

import (
	"cmp"
	"regexp"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
	umpirespb "go.temporal.io/server/api/umpire/v1"
	umpire "go.temporal.io/server/tools/umpire/model/internal/checker"
	"google.golang.org/protobuf/proto"
)

const nexusCloseIR = "../../../model/ir/nexus-close.json"

var nexusClose = sync.OnceValues(func() (*checkedModel, error) {
	m, err := Load(nexusCloseIR)
	if err != nil {
		return nil, err
	}
	return checkedOnce(m)
})

// checkedOnce is Check within the default scope, with the interpretation its checks read kept beside
// the report: a test that reads rows then interprets the Model no further time, which for a Model of
// large catalogs is most of what a test costs. It is check's own sequence, receipt for receipt, and
// TestCheckedOnceIsCheck holds it to Check. A machine that has no table is an error here as it is of
// Build.
func checkedOnce(m *umpirespb.Model) (*checkedModel, error) {
	if err := Validate(m); err != nil {
		return nil, err
	}
	c := newChecker(m, DefaultScope, m)
	r := &Report{Scope: DefaultScope}
	for _, mm := range m.GetMachines() {
		if err := c.first.failed[mm.GetName()]; err != nil {
			return nil, err
		}
		if mm.GetRefines() != nil {
			r.Receipts = append(r.Receipts, c.refinement(mm))
		}
	}
	for _, composition := range m.GetCompositions() {
		if x, checked := c.composition(composition); checked {
			r.Receipts = append(r.Receipts, x)
		}
	}
	for _, q := range m.GetQueries() {
		r.Receipts = append(r.Receipts, c.query(q))
	}
	for _, p := range m.GetProgress() {
		r.Receipts = append(r.Receipts, c.progress(p)...)
	}
	subjects := []Subject{ModelSubject, MachineSubject, RefinementSubject, CompositionSubject, QuerySubject, ProgressSubject}
	slices.SortStableFunc(r.Receipts, func(x, y Receipt) int {
		return cmp.Or(cmp.Compare(slices.Index(subjects, x.Subject), slices.Index(subjects, y.Subject)),
			strings.Compare(x.Key.Family, y.Key.Family), strings.Compare(x.Key.Owner, y.Key.Owner),
			strings.Compare(x.Key.Name, y.Key.Name))
	})
	return &checkedModel{model: m, report: r, built: c.first.machines}, nil
}

// The report checkedOnce keeps is Check's, whole, on Models with refinements, compositions, monitors,
// Queries and progress claims, and the machines it keeps are Build's.
func TestCheckedOnceIsCheck(t *testing.T) {
	for name, m := range map[string]*umpirespb.Model{"admission": lifted(t, "admission"),
		"declarations": mutated(t, "declarations", noCrash)} {
		t.Run(name, func(t *testing.T) {
			once, err := checkedOnce(m)
			require.NoError(t, err)
			require.Equal(t, plain(Check(m, DefaultScope)), plain(once.report))
			require.NotEmpty(t, once.report.Receipts)
			want := built(t, m)
			require.Len(t, once.built, len(want))
			for machine, mm := range want {
				require.Equal(t, mm.Table.TargetFingerprint(), once.built[machine].Table.TargetFingerprint(), machine)
				require.Equal(t, mm.Table.Rows, once.built[machine].Table.Rows, machine)
			}
		})
	}
	// A machine with no table is Build's error, and no report.
	unread := mutated(t, "admission", func(m *umpirespb.Model) {
		m.Holes = append(m.Holes, &umpirespb.Hole{Id: "generic.unknownEnd", Name: "unknownEnd", Position: at(1)})
		current := admMachine(m, "currentAdmission")
		current.Ends = &umpirespb.Expr{Position: current.GetEnds().GetPosition(), Kind: &umpirespb.Expr_Hole{Hole: "generic.unknownEnd"}}
	})
	_, want := Build(unread)
	require.Error(t, want)
	_, err := checkedOnce(unread)
	require.EqualError(t, err, want.Error())
}

func closeModel(t *testing.T) *checkedModel {
	t.Helper()
	c, err := nexusClose()
	require.NoError(t, err)
	return c
}

// The designs, in the order the tables below list their verdicts.
var (
	// The three policies of the specimen, and the two faulty resets: one forgets the cancel request, one
	// does not reapply what the original run recorded.
	closeDesigns = []string{"rejectAfterClose", "ackByOriginal", "retainAndRoute", "forgetsCancelOnReset", "truncatesOnReset"}
	// The three policies over the channel that redelivers once, with a schedule-to-close deadline.
	closeDeadlineDesigns = []string{"rejectAfterCloseWithDeadline", "ackByOriginalWithDeadline", "retainAndRouteWithDeadline"}
)

const closeBounded = "retainAndRouteBoundedRetry"

// closeVerdict is a Query's kind and the monitor a counterexample names, "" where the Property fails.
type closeVerdict struct {
	Kind    ReceiptKind
	Monitor string
}

// verdicts reads one word per design: V verified, F found, N not found, X a counterexample of the
// Property, and X:<monitor> a counterexample that only a watching monitor reports.
func verdicts(t *testing.T, words string) []closeVerdict {
	t.Helper()
	var out []closeVerdict
	for _, w := range strings.Fields(words) {
		kind, monitor, _ := strings.Cut(w, ":")
		v, ok := map[string]ReceiptKind{"V": Verified, "F": Found, "N": NotFound, "X": Counterexample}[kind]
		require.True(t, ok, w)
		out = append(out, closeVerdict{v, monitor})
	}
	return out
}

func closeQuery(t *testing.T, c *checkedModel, design, name string) Receipt {
	t.Helper()
	return receiptOf(t, c.report, "query "+design+" "+design+"."+name)
}

func closeProgress(t *testing.T, r *Report, design, claim string, part ProgressKind) Receipt {
	t.Helper()
	return receiptOf(t, r, "progress "+design+" "+claim+" "+string(part))
}

// Every declaration has one result, and none is left unanswered or unsupported. The kinds are the
// oracles' verdicts. A design whose monitor is violated answers a free verify with that violation,
// whatever Property the Query asks, so a Property that holds of every step is shown on the corrected
// design, and on a faulty one the receipt names the monitor.
func TestNexusCloseResults(t *testing.T) {
	c := closeModel(t)
	require.Empty(t, c.report.Unsupported())
	want := map[string]closeVerdict{}
	set := func(designs []string, queries map[string]string) {
		for name, words := range queries {
			vs := verdicts(t, words)
			require.Len(t, vs, len(designs), name)
			for i, design := range designs {
				want["query "+design+" "+design+"."+name] = vs[i]
			}
		}
	}
	// rejectAfterClose, ackByOriginal, retainAndRoute, forgetsCancelOnReset, truncatesOnReset.
	set(closeDesigns, map[string]string{
		// N1 and N1': the closed run rejects permanently and the outcome is lost; retained, it is reapplied.
		"closedThenFinished": "X V V V V",
		// N2 and N2': after a reset the original run acknowledges; routed, the successor commits.
		"resetThenDelivered.ackOnlyWhenKept":  "V X V V V",
		"resetThenDelivered.outcomePreserved": "V X V V V",
		// N3 and N7: the canceled outcome crosses the reset, and so must the request that asked for it.
		"canceledAcrossReset": "V X V X:cancelPrincipal V",
		// N4, and its mutation: a reset that does not reapply what the original run recorded.
		"ackedThenReset": "V V V V X",
		// N5 and N6.
		"duplicateCompletion":                 "V V V V V",
		"resetBetweenCommitAndAcknowledgment": "V X V V X:retainedOutcome",
		// The free searches. A permanent rejection is no acknowledgment and a reset is none either, so
		// where those lose the outcome it is the retainedOutcome monitor that says so.
		"any.outcomePreserved":            "X X V X:cancelPrincipal X",
		"any.ackOnlyWhenKept":             "X:retainedOutcome X V X:cancelPrincipal X:retainedOutcome",
		"any.closedHistoryIsFrozen":       "X:retainedOutcome X:retainedOutcome V X:cancelPrincipal X:retainedOutcome",
		"any.handlerEffectIsIrreversible": "X:retainedOutcome X:retainedOutcome V X:cancelPrincipal X:retainedOutcome",
		"any.knownIsTheHandlersOutcome":   "X:retainedOutcome X:retainedOutcome V X:cancelPrincipal X:retainedOutcome",
		"any.knowledgeIsFinal":            "X:retainedOutcome X:retainedOutcome V X:cancelPrincipal X",
		// What stays apart: the frozen history and the detached work, and a cancellation's request,
		// receipt, effect and knowledge.
		"detachedWorkProceeds":   "F F F F F",
		"intentWithoutReceipt":   "F F F F F",
		"receiptWithoutEffect":   "F F F F F",
		"effectWithoutKnowledge": "F F F F F",
		"canceledIsKnown":        "F F F F F",
		// What the Go baseline has too: an open caller accepts a completion.
		"asyncCompletion": "F F F F F",
		"asyncFailure":    "F F F F F",
		// Transient and permanent rejection are told apart, and so are the two resets of N6.
		"transientRejectionAfterClose": "F F F F F",
		"permanentRejectionAfterClose": "F N N N N",
		"lostAfterReset":               "F N N N N",
		"resetAfterRetention":          "N F F F F",
		"resetBeforeRetention":         "F N F F F",
	})
	safety := map[string]string{
		"closedThenFinished":                 "X V V",
		"resetThenDelivered.ackOnlyWhenKept": "V X V",
		"any.outcomePreserved":               "X X V",
		"any.ackOnlyWhenKept":                "X:retainedOutcome X V",
	}
	set(closeDeadlineDesigns, safety)
	// N8: the deadline resolves the wait the lost outcome leaves, and a late completion is dropped.
	set(closeDeadlineDesigns, map[string]string{
		"expiredAfterClosedLoss":  "F N N",
		"expiredAfterResetLoss":   "N F N",
		"any.noUnnecessaryWait":   "X:retainedOutcome X:retainedOutcome V",
		"expiredWhileReported":    "F F F",
		"lateCompletionIsDropped": "F F F",
	})
	for name, words := range safety {
		want["query "+closeBounded+" "+closeBounded+"."+name] = verdicts(t, words)[2]
	}

	// Progress, as deadlock, fair cycle and missed deadline. N1 and N2 are the deadlocks. N9 is the
	// cycle of a channel that retries until acknowledged, which the channel that redelivers once has
	// not. N8: with the deadline no design is stuck. N11: a retained outcome reaches an owner only if a
	// reset runs.
	for claim, parts := range map[string]string{
		"rejectAfterClose outcomeReachesOwner":             "X X X",
		"ackByOriginal outcomeReachesOwner":                "X X X",
		"retainAndRoute outcomeReachesOwner":               "V X X",
		closeBounded + " outcomeReachesOwner":              "V V V",
		"rejectAfterCloseWithDeadline outcomeReachesOwner": "V V V",
		"ackByOriginalWithDeadline outcomeReachesOwner":    "V V V",
		"retainAndRouteWithDeadline outcomeReachesOwner":   "V V V",
		"retainAndRoute retainedReachesOwner":              "V V X",
		"retainAndRoute retainedWaitsWithoutRecovery":      "V X X",
		closeBounded + " retainedReachesOwner":             "V V V",
	} {
		vs := verdicts(t, parts)
		for i, part := range []ProgressKind{umpire.DeadlockKind, umpire.CycleKind, umpire.DeadlineKind} {
			want["progress "+claim+" "+string(part)] = vs[i]
		}
	}

	got := map[string]closeVerdict{}
	for _, r := range c.report.Receipts {
		got[receiptKey(r)] = closeVerdict{r.Kind, r.Monitor}
		require.Empty(t, r.Holes, receiptKey(r))
		t.Logf("%s: %s %s, %d explored within %s (%d steps), assuming %v", receiptKey(r), r.Kind, r.Monitor, r.Explored,
			r.Limits.Name, r.Limits.Steps, r.Assumptions)
	}
	require.Equal(t, want, got)
}

const (
	closeOpened = "open-none-running-none-none-none"
	// The lost outcome of N1 and N2: a successor that owns an operation whose outcome nothing holds.
	closeLost = "resetOpen-none-done-succeeded-none-none-none"
)

func outcomes(w *Trace) []string {
	var out []string
	for _, s := range w.Steps {
		out = append(out, s.Outcome.Value)
	}
	return out
}

// N1, pinned control 1: the handler finishes after the caller closed, the closed run rejects its
// completion permanently, the handler stops reporting, and the reset rebuilds the operation as open.
func TestNexusCloseRejectionAfterCloseLosesTheOutcome(t *testing.T) {
	c := closeModel(t)
	n1 := closeQuery(t, c, "rejectAfterClose", "closedThenFinished")
	rows := []string{closeOpened + "-callerClose",
		"closed-none-running-none-none-none-handlerFinish-succeeded",
		"closed-none-done-succeeded-inFlight-succeeded-none-none-complete-succeeded"}
	require.Equal(t, rows, n1.Rows)
	require.Equal(t, []string{"accepted", "accepted", "rejectedPermanent"}, outcomes(n1.Witness))
	require.Equal(t, "closed-none-done-succeeded-none-none-none", last(t, n1.Witness).State.Value)
	require.Equal(t, []string{"completionDropped"}, factsOf(last(t, n1.Witness)))
	require.Empty(t, n1.Monitor, "the Property fails; the monitor that agrees is not what is reported")

	// Step 4: the reset finds nothing to reapply, and the state it reaches has no step and is no end.
	n1reset := closeQuery(t, c, "rejectAfterClose", "lostAfterReset")
	require.Equal(t, append(rows, "closed-none-done-succeeded-none-none-none-reset"), n1reset.Rows)
	require.Equal(t, closeLost, last(t, n1reset.Witness).State.Value)
	require.Equal(t, []string{"workflowReset"}, factsOf(last(t, n1reset.Witness)))
	table := c.built["rejectAfterClose"].Table
	require.Equal(t, closeLost, stuck(table))
	require.Contains(t, closeStuck(table), closeLost)

	// The free search returns the same shape, with failed as the outcome.
	free := closeQuery(t, c, "rejectAfterClose", "any.outcomePreserved")
	require.Len(t, free.Rows, 3)
	require.Equal(t, "closed-none-done-failed-none-none-none", last(t, free.Witness).State.Value)
	// ackOnlyWhenKept holds of that step: the monitor is what reports the loss on the same path.
	monitored := closeQuery(t, c, "rejectAfterClose", "any.ackOnlyWhenKept")
	require.Equal(t, free.Witness, monitored.Witness)
	require.Equal(t, []MonitorVerdict{
		{Name: "retainedOutcome", State: "true", Verdict: umpire.MonitorViolated},
		{Name: "ownerAcknowledgment", State: "false", Verdict: umpire.MonitorHeld},
		{Name: "singleOutcome", State: "none", Verdict: umpire.MonitorHeld},
		{Name: "cancelPrincipal", State: "nobody", Verdict: umpire.MonitorHeld},
	}, monitored.Monitors)

	// A transient rejection at step 3 keeps the report in flight, and is a different answer.
	transient := closeQuery(t, c, "rejectAfterClose", "transientRejectionAfterClose")
	require.Equal(t, rows, transient.Rows)
	require.Equal(t, "rejectedTransient", last(t, transient.Witness).Outcome.Value)
	require.Equal(t, "closed-none-done-succeeded-inFlight-succeeded-none-none", last(t, transient.Witness).State.Value)
	permanent := closeQuery(t, c, "rejectAfterClose", "permanentRejectionAfterClose")
	require.Equal(t, n1.Witness, permanent.Witness)
}

// N1' and N6, reset after retention: the corrected design answers the closed run's completion
// `retained`, and the reset reapplies it to the successor.
func TestNexusCloseRetainedOutcomeIsReapplied(t *testing.T) {
	c := closeModel(t)
	n1 := closeQuery(t, c, "retainAndRoute", "resetAfterRetention")
	require.Equal(t, []string{"callerClose", "handlerFinish-succeeded", "complete-succeeded", "reset"}, taken(n1.Witness))
	require.Equal(t, []string{"accepted", "accepted", "retained", "accepted"}, outcomes(n1.Witness))
	require.Equal(t, "closed-none-done-succeeded-none-pending-succeeded-none", n1.Witness.Steps[2].State.Value)
	require.Equal(t, []string{"outcomeRetained"}, factsOf(n1.Witness.Steps[2]))
	require.Equal(t, "resetOpen-none-done-succeeded-none-none-successor-succeeded", last(t, n1.Witness).State.Value)
	require.Equal(t, []string{"workflowReset", "outcomeReapplied", "nexusOperationCompleted"}, factsOf(last(t, n1.Witness)))
	require.True(t, closeQuery(t, c, "retainAndRoute", "closedThenFinished").Exercised)
	require.Empty(t, stuck(c.built["retainAndRoute"].Table))
}

// N2, pinned control 2, and N2' with N6's reset before retention: a completion that arrives after the
// reset is acknowledged by the original run and the successor never learns it; routed, it is committed
// by the successor.
func TestNexusCloseAcknowledgmentByTheOriginalRun(t *testing.T) {
	c := closeModel(t)
	rows := []string{closeOpened + "-handlerFinish-failed",
		"open-none-done-failed-inFlight-failed-none-none-reset",
		"resetOpen-none-done-failed-inFlight-failed-none-none-complete-failed"}
	for _, name := range []string{"resetThenDelivered.ackOnlyWhenKept", "resetThenDelivered.outcomePreserved",
		"any.ackOnlyWhenKept", "any.outcomePreserved"} {
		n2 := closeQuery(t, c, "ackByOriginal", name)
		require.Equal(t, rows, n2.Rows, name)
		require.Equal(t, []string{"accepted", "accepted", "accepted"}, outcomes(n2.Witness), name)
		require.Equal(t, "resetOpen-none-done-failed-none-none-none", last(t, n2.Witness).State.Value, name)
		require.Empty(t, factsOf(last(t, n2.Witness)), "no history records the outcome")
		require.Empty(t, n2.Monitor, name)
	}
	// The state it leaves has no step and is no end, as N1's: the table reports that one for both designs.
	require.Contains(t, closeStuck(c.built["ackByOriginal"].Table), "resetOpen-none-done-failed-none-none-none")
	require.Equal(t, closeLost, stuck(c.built["ackByOriginal"].Table))
	// Both monitors read the acknowledgment the same way.
	require.Equal(t, []MonitorVerdict{
		{Name: "retainedOutcome", State: "true", Verdict: umpire.MonitorViolated},
		{Name: "ownerAcknowledgment", State: "true", Verdict: umpire.MonitorViolated},
		{Name: "singleOutcome", State: "none", Verdict: umpire.MonitorHeld},
		{Name: "cancelPrincipal", State: "nobody", Verdict: umpire.MonitorHeld},
	}, closeQuery(t, c, "ackByOriginal", "any.ackOnlyWhenKept").Monitors)

	routed := closeQuery(t, c, "retainAndRoute", "resetBeforeRetention")
	require.Equal(t, rows, routed.Rows)
	require.Equal(t, "resetOpen-none-done-failed-none-none-successor-failed", last(t, routed.Witness).State.Value)
	require.Equal(t, []string{"nexusOperationFailed"}, factsOf(last(t, routed.Witness)))
}

// N3: a canceled outcome crosses the reset. The request, the handler's receipt of it, the handler's
// effect and the caller's knowledge are four steps, and the cancel intent survives the reset in the
// three policies. Receipt is not effect, and effect is not knowledge.
func TestNexusCloseCancellationAcrossReset(t *testing.T) {
	c := closeModel(t)
	path := []string{"requestCancel-callerWorkflow", "deliverCancel", "handlerFinish-canceled", "reset", "complete-canceled"}
	n3 := closeQuery(t, c, "ackByOriginal", "canceledAcrossReset")
	require.Equal(t, path, taken(n3.Witness))
	require.Equal(t, "resetOpen-requested-callerWorkflow-done-canceled-none-none-none", last(t, n3.Witness).State.Value)
	for _, design := range []string{"rejectAfterClose", "ackByOriginal", "retainAndRoute"} {
		require.Equal(t, []Result{{Outcome: "accepted", State: "resetOpen-requested-callerWorkflow-done-canceled-inFlight-canceled-none-none",
			Facts: []string{"workflowReset"}}},
			plainResults(t, c.built[design].Table, "open-requested-callerWorkflow-done-canceled-inFlight-canceled-none-none-reset"), design)
	}
	require.True(t, closeQuery(t, c, "retainAndRoute", "canceledAcrossReset").Exercised)

	table := c.built["retainAndRoute"]
	// The request alone reaches no handler: a canceled result needs the handler's receipt of it.
	requested := closeQuery(t, c, "retainAndRoute", "intentWithoutReceipt")
	require.Equal(t, "open-requested-callerWorkflow-running-none-none-none", last(t, requested.Witness).State.Value)
	require.Equal(t, []string{"cancelRequested-callerWorkflow"}, factsOf(last(t, requested.Witness)))
	require.True(t, table.Disabled("open-requested-callerWorkflow-running-none-none-none", "handlerFinish-canceled"))
	require.True(t, table.Disabled(closeOpened, "handlerFinish-canceled"))
	require.True(t, table.Disabled(closeOpened, "deliverCancel"))
	// A handler that received the request may still succeed.
	received := closeQuery(t, c, "retainAndRoute", "receiptWithoutEffect")
	require.Equal(t, []string{"requestCancel-callerWorkflow", "deliverCancel", "handlerFinish-succeeded"}, taken(received.Witness))
	require.Equal(t, "open-requested-callerWorkflow-cancelReceived-none-none-none", received.Witness.Steps[1].State.Value)
	require.Equal(t, []string{"cancelReceived"}, factsOf(received.Witness.Steps[1]))
	require.Equal(t, "open-requested-callerWorkflow-done-succeeded-inFlight-succeeded-none-none", last(t, received.Witness).State.Value)
	// The handler's canceled effect is no knowledge of the caller's until the completion is committed.
	effect := closeQuery(t, c, "retainAndRoute", "effectWithoutKnowledge")
	require.Equal(t, "open-requested-callerWorkflow-done-canceled-inFlight-canceled-none-none", last(t, effect.Witness).State.Value)
	require.Equal(t, []string{"handlerFinished-canceled"}, factsOf(last(t, effect.Witness)))
	known := closeQuery(t, c, "retainAndRoute", "canceledIsKnown")
	require.Equal(t, "open-requested-callerWorkflow-done-canceled-none-none-original-canceled", last(t, known.Witness).State.Value)
	require.Equal(t, []string{"nexusOperationCanceled"}, factsOf(last(t, known.Witness)))
	// One cancellation, and none once the caller closed or the handler finished.
	for _, state := range []string{"open-requested-callerWorkflow-running-none-none-none", "closed-none-running-none-none-none",
		"open-none-done-succeeded-inFlight-succeeded-none-none"} {
		require.True(t, table.Disabled(state, "requestCancel-callerWorkflow"), state)
	}
}

// N7: a reset that forgets the cancel request leaves the successor owning a canceled outcome it never
// asked for. It is reported by the cancelPrincipal monitor at the reset, and is no loss of the outcome:
// the monitors of the outcome hold on the same path.
func TestNexusClosePrincipalLossIsItsOwnAssessment(t *testing.T) {
	c := closeModel(t)
	n7 := closeQuery(t, c, "forgetsCancelOnReset", "canceledAcrossReset")
	require.Equal(t, []string{"requestCancel-callerWorkflow", "deliverCancel", "handlerFinish-canceled", "reset"}, taken(n7.Witness))
	require.Equal(t, "resetOpen-none-done-canceled-inFlight-canceled-none-none", last(t, n7.Witness).State.Value)
	require.Equal(t, []MonitorVerdict{
		{Name: "retainedOutcome", State: "false", Verdict: umpire.MonitorHeld},
		{Name: "ownerAcknowledgment", State: "false", Verdict: umpire.MonitorHeld},
		{Name: "singleOutcome", State: "none", Verdict: umpire.MonitorHeld},
		{Name: "cancelPrincipal", State: "lost", Verdict: umpire.MonitorViolated},
	}, n7.Monitors)
	// The free search needs only the request and the reset.
	free := closeQuery(t, c, "forgetsCancelOnReset", "any.outcomePreserved")
	require.Equal(t, []string{"requestCancel-callerWorkflow", "reset"}, taken(free.Witness))
	// The successor then commits the canceled outcome with no request in its history.
	require.Equal(t, []Result{{Outcome: "accepted", State: "resetOpen-none-done-canceled-none-none-successor-canceled",
		Facts: []string{"nexusOperationCanceled"}, Choice: "taken"}},
		plainResults(t, c.built["forgetsCancelOnReset"].Table,
			"resetOpen-none-done-canceled-inFlight-canceled-none-none-complete-canceled")[:1])
	// Every design that keeps the request keeps its principal: the monitor holds over their whole search.
	for _, design := range []string{"retainAndRoute", closeBounded, "retainAndRouteWithDeadline"} {
		require.Contains(t, closeQuery(t, c, design, "any.outcomePreserved").Monitors,
			MonitorVerdict{Name: "cancelPrincipal", Verdict: umpire.MonitorHeld}, design)
	}
}

// N4: a reset after the acknowledgment reapplies what the original run recorded. Its mutation, a reset
// that does not, loses the outcome at the reset step.
func TestNexusCloseResetAfterAcknowledgment(t *testing.T) {
	c := closeModel(t)
	for _, design := range []string{"rejectAfterClose", "ackByOriginal", "retainAndRoute"} {
		require.True(t, closeQuery(t, c, design, "ackedThenReset").Exercised, design)
		require.Equal(t, []Result{{Outcome: "accepted", State: "resetOpen-none-done-succeeded-none-none-successor-succeeded",
			Facts: []string{"workflowReset", "outcomeReapplied", "nexusOperationCompleted"}}},
			plainResults(t, c.built[design].Table, "open-none-done-succeeded-none-none-original-succeeded-reset"), design)
	}
	truncated := closeQuery(t, c, "truncatesOnReset", "ackedThenReset")
	require.Equal(t, []string{"handlerFinish-succeeded", "complete-succeeded", "reset"}, taken(truncated.Witness))
	require.Equal(t, closeLost, last(t, truncated.Witness).State.Value)
	require.Empty(t, truncated.Monitor)
	// What the operation retained is still reapplied by that reset: only the original run's record is cut.
	require.Equal(t, Found, closeQuery(t, c, "truncatesOnReset", "resetAfterRetention").Kind)
}

// N5: a lost acknowledgment delivers the completion again. The first delivery commits and records the
// outcome; the second is accepted, changes no knowledge and records nothing.
func TestNexusCloseDuplicateCompletion(t *testing.T) {
	c := closeModel(t)
	table := c.built["retainAndRoute"].Table
	const lostAck = "the acknowledgment is lost"
	require.Equal(t, []Result{
		{Outcome: "accepted", State: "open-none-done-succeeded-none-none-original-succeeded", Facts: []string{"nexusOperationCompleted"},
			Choice: "taken"},
		{Outcome: "rejectedTransient", State: "open-none-done-succeeded-inFlight-succeeded-none-none", Facts: []string{},
			Choice: "rejectedForNow"},
		{Outcome: "accepted", State: "open-none-done-succeeded-inFlight-succeeded-none-original-succeeded",
			Facts: []string{"nexusOperationCompleted"}, Because: lostAck, Choice: "ackLost"},
	}, plainResults(t, table, "open-none-done-succeeded-inFlight-succeeded-none-none-complete-succeeded"))
	require.Equal(t, []Result{
		{Outcome: "accepted", State: "open-none-done-succeeded-none-none-original-succeeded", Facts: []string{}, Choice: "taken"},
		{Outcome: "rejectedTransient", State: "open-none-done-succeeded-inFlight-succeeded-none-original-succeeded", Facts: []string{},
			Choice: "rejectedForNow"},
		{Outcome: "accepted", State: "open-none-done-succeeded-inFlight-succeeded-none-original-succeeded", Facts: []string{},
			Because: lostAck, Choice: "ackLost"},
	}, plainResults(t, table, "open-none-done-succeeded-inFlight-succeeded-none-original-succeeded-complete-succeeded"))
	for _, design := range closeDesigns {
		require.True(t, closeQuery(t, c, design, "duplicateCompletion").Exercised, design)
	}
	// The channel carries the handler's one result: no other completion is deliverable.
	for _, class := range []string{"complete-failed", "complete-canceled"} {
		require.True(t, c.built["retainAndRoute"].Disabled("open-none-done-succeeded-inFlight-succeeded-none-none", class), class)
	}
	// No run records two outcomes on anything the corrected design reaches.
	require.Contains(t, closeQuery(t, c, "retainAndRoute", "any.knowledgeIsFinal").Monitors,
		MonitorVerdict{Name: "singleOutcome", Verdict: umpire.MonitorHeld})
}

// A closed run's history is frozen while the handler's detached work goes on: no step of a closed
// caller changes what it knows or what it asked, and the handler finishes after the close.
func TestNexusCloseFrozenHistoryAndDetachedWork(t *testing.T) {
	c := closeModel(t)
	detached := closeQuery(t, c, "retainAndRoute", "detachedWorkProceeds")
	require.Equal(t, []string{"callerClose", "handlerFinish-succeeded"}, taken(detached.Witness))
	require.Equal(t, []string{"workflowClosed"}, factsOf(detached.Witness.Steps[0]))
	require.Equal(t, "closed-none-done-succeeded-inFlight-succeeded-none-none", last(t, detached.Witness).State.Value)
	for _, name := range []string{"any.closedHistoryIsFrozen", "any.handlerEffectIsIrreversible", "any.knownIsTheHandlersOutcome"} {
		require.True(t, closeQuery(t, c, "retainAndRoute", name).Exercised, name)
	}
	// Read on the rows, for every design: a step that leaves the caller closed leaves its history's
	// fields as they were, and no step undoes the handler's effect.
	for name, mm := range c.built {
		for _, row := range mm.Table.Rows {
			before := strings.Split(row.Source, "-")
			for _, res := range row.Results {
				if before[0] == "closed" && strings.HasPrefix(res.State, "closed-") {
					require.Equal(t, closeHistory(row.Source), closeHistory(res.State), "%s: %s", name, row.Key)
				}
				if done := closeDone.FindString(row.Source); done != "" {
					require.Equal(t, done, closeDone.FindString(res.State), "%s: %s", name, row.Key)
				}
			}
		}
	}
}

var (
	closeDone = regexp.MustCompile(`-done-(succeeded|failed|canceled)-`)
	// A state key's cancel intent and, after the retained field, what the runs know.
	closeIntent    = regexp.MustCompile(`^[a-zA-Z]+-(none|requested-callerWorkflow)-`)
	closeKnowledge = regexp.MustCompile(`-(none|pending-[a-z]+)-(none|expired|original-[a-z]+|successor-[a-z]+)$`)
)

// closeHistory is what a run's history holds of a state: the cancel intent and the knowledge.
func closeHistory(state string) [2]string {
	return [2]string{closeIntent.FindStringSubmatch(state)[1], closeKnowledge.FindStringSubmatch(state)[2]}
}

// N8: with a schedule-to-close deadline the state N1 and N2 are stuck in has a step. The lost outcome
// stays the safety counterexample it was, the timeout that resolves the wait is found with nothing
// owed, and no progress receipt reports a hang. In the corrected design a deadline fires only while
// the report is still owed.
func TestNexusCloseTimeoutResolvesTheLostOutcome(t *testing.T) {
	c := closeModel(t)
	for design, lost := range map[string]struct {
		query string
		path  []string
		state string
	}{
		"rejectAfterCloseWithDeadline": {"expiredAfterClosedLoss",
			[]string{"callerClose", "handlerFinish-succeeded", "complete-succeeded", "reset", "scheduleToClose"},
			"resetOpen-none-done-succeeded-none-none-expired"},
		"ackByOriginalWithDeadline": {"expiredAfterResetLoss",
			[]string{"handlerFinish-failed", "reset", "complete-failed", "scheduleToClose"},
			"resetOpen-none-done-failed-none-none-expired"},
	} {
		expired := closeQuery(t, c, design, lost.query)
		require.Equal(t, lost.path, taken(expired.Witness), design)
		require.Equal(t, lost.state, last(t, expired.Witness).State.Value, design)
		require.Equal(t, []string{"nexusOperationTimedOut"}, factsOf(last(t, expired.Witness)), design)
		require.Empty(t, stuck(c.built[design].Table), design)
		for _, part := range []ProgressKind{umpire.DeadlockKind, umpire.CycleKind, umpire.DeadlineKind} {
			r := closeProgress(t, c.report, design, "outcomeReachesOwner", part)
			require.Equal(t, Verified, r.Kind, "%s %s", design, part)
			require.Nil(t, r.Witness)
			require.True(t, r.Exercised)
			require.Contains(t, r.Assumptions, "scheduleToCloseExpires")
		}
	}
	// The safety violation is the one the design has without the deadline, at the same step.
	for _, design := range []string{"rejectAfterClose", "ackByOriginal"} {
		for _, name := range []string{"closedThenFinished", "resetThenDelivered.ackOnlyWhenKept", "any.outcomePreserved"} {
			timed := closeQuery(t, c, design+"WithDeadline", name)
			require.Equal(t, closeQuery(t, c, design, name).Rows, timed.Rows, "%s %s", design, name)
		}
	}
	// A deadline that fires while the report is in flight is no loss, and the completion that arrives
	// after it is dropped.
	for _, design := range closeDeadlineDesigns {
		owed := closeQuery(t, c, design, "expiredWhileReported")
		require.Equal(t, "open-none-done-succeeded-inFlight-succeeded-none-expired", last(t, owed.Witness).State.Value, design)
		late := closeQuery(t, c, design, "lateCompletionIsDropped")
		require.Equal(t, []string{"handlerFinish-succeeded", "scheduleToClose", "complete-succeeded"}, taken(late.Witness), design)
		require.Equal(t, "rejectedPermanent", last(t, late.Witness).Outcome.Value, design)
		require.Equal(t, "open-none-done-succeeded-none-none-expired", last(t, late.Witness).State.Value, design)
		// A closed run's history is frozen, so no deadline fires in it.
		require.True(t, c.built[design].Disabled("closed-none-done-succeeded-none-none-none", "scheduleToClose"), design)
	}
	require.True(t, closeQuery(t, c, "retainAndRouteWithDeadline", "any.noUnnecessaryWait").Exercised)
	// No design without the deadline has the step.
	for _, design := range append([]string{closeBounded}, closeDesigns...) {
		require.NotContains(t, c.built[design].Table.Actions, "scheduleToClose", design)
	}
}

// N1 and N2 as progress witnesses, and N10: the deadlock is the lost outcome's stuck state, reached
// after the handler finished. A state in which the handler still runs is where a path may end, and no
// prefix that ends there is a counterexample.
func TestNexusCloseDeadlockIsTheLostOutcome(t *testing.T) {
	c := closeModel(t)
	lost := regexp.MustCompile(`^resetOpen-none-done-(succeeded|failed)-none-none-none$`)
	for design, answer := range map[string]string{"rejectAfterClose": "rejectedPermanent", "ackByOriginal": "accepted"} {
		table := c.built[design].Table
		dead := closeProgress(t, c.report, design, "outcomeReachesOwner", umpire.DeadlockKind)
		end := last(t, dead.Witness).State.Value
		require.Regexp(t, lost, end, design)
		require.Empty(t, table.RowsFrom(end), design)
		require.Equal(t, -1, dead.Loop, design)
		require.NoError(t, table.Replay(dead.Witness), design)
		// The step that lost it is the delivery, with the answer the policy gives.
		delivered := slicesIndex(taken(dead.Witness), func(a string) bool { return strings.HasPrefix(a, "complete-") })
		require.GreaterOrEqual(t, delivered, 0, design)
		require.Equal(t, answer, dead.Witness.Steps[delivered].Outcome.Value, design)
		require.Contains(t, closeStuck(table), end, design)
		for _, stuck := range closeStuck(table) {
			require.Regexp(t, closeDone, stuck, "%s: a stuck state is one the handler finished in", design)
		}
	}
	require.Equal(t, Verified, closeProgress(t, c.report, "retainAndRoute", "outcomeReachesOwner", umpire.DeadlockKind).Kind)

	// A prefix that ends while the handler still runs proves nothing: explored no further than the
	// start, every kind is unresolved, on a faulty design too. Explored three steps deep, one short of
	// the stuck state, the deadlock is still unresolved and no hang is claimed.
	only := proto.Clone(c.model).(*umpirespb.Model)
	only.Queries, only.Properties, only.Scenarios = nil, nil, nil
	only.Machines = slicesDelete(only.GetMachines(), func(m *umpirespb.Machine) bool { return m.GetName() != "rejectAfterClose" })
	only.Progress = slicesDelete(only.GetProgress(), func(p *umpirespb.Progress) bool { return p.GetMachine() != "rejectAfterClose" })
	scope := DefaultScope
	scope.Progress = Limits{Name: "the start", Steps: 0, Search: 1 << 20}
	open := Check(only, scope)
	for _, part := range []ProgressKind{umpire.DeadlockKind, umpire.CycleKind, umpire.DeadlineKind} {
		r := closeProgress(t, open, "rejectAfterClose", "outcomeReachesOwner", part)
		require.Equal(t, Unresolved, r.Kind, part)
		require.Nil(t, r.Witness, part)
	}
	scope.Progress = Limits{Name: "three steps", Steps: 3, Search: 1 << 20}
	short := closeProgress(t, Check(only, scope), "rejectAfterClose", "outcomeReachesOwner", umpire.DeadlockKind)
	require.Equal(t, Unresolved, short.Kind)
	require.Nil(t, short.Witness)
	require.Len(t, closeProgress(t, c.report, "rejectAfterClose", "outcomeReachesOwner", umpire.DeadlockKind).Witness.Steps, 4)
}

// closeStuck is every state a table reaches that has no step and is no end.
func closeStuck(table *Table) []string {
	var out []string
	for _, s := range table.Reachable {
		if len(table.RowsFrom(s)) == 0 && !slices.Contains(table.Ends, s) {
			out = append(out, s)
		}
	}
	return out
}

func slicesIndex(xs []string, f func(string) bool) int {
	for i, x := range xs {
		if f(x) {
			return i
		}
	}
	return -1
}

// N9: over the channel that retries until acknowledged, a transient rejection forever is a fair
// non-progress cycle, though the delivery is fair. Over the channel that redelivers once, which the
// machine assumes by name, there is none. Each receipt names the variant by its machine and the
// assumptions it relies on.
func TestNexusCloseFairNonProgressCycle(t *testing.T) {
	c := closeModel(t)
	const retry, delivery = "transientRejectionEventuallyAccepted", "enabledDeliveryAndRecoveryActionsEventuallyRun"
	cycle := closeProgress(t, c.report, "retainAndRoute", "outcomeReachesOwner", umpire.CycleKind)
	require.GreaterOrEqual(t, cycle.Loop, 0)
	require.Greater(t, len(cycle.Witness.Steps), cycle.Loop)
	for _, step := range cycle.Witness.Steps[cycle.Loop:] {
		require.True(t, strings.HasPrefix(step.Action.Value, "complete-"), step.Action.Value)
		require.Equal(t, "rejectedTransient", step.Outcome.Value)
	}
	require.Contains(t, cycle.Assumptions, delivery)
	require.NotContains(t, cycle.Assumptions, retry)
	require.NoError(t, c.built["retainAndRoute"].Table.Replay(cycle.Witness))

	for _, part := range []ProgressKind{umpire.DeadlockKind, umpire.CycleKind, umpire.DeadlineKind} {
		r := closeProgress(t, c.report, closeBounded, "outcomeReachesOwner", part)
		require.Equal(t, Verified, r.Kind, part)
		require.Equal(t, []string{"retentionSurvivesCrash", retry, "handlerReportsUntilAckOrPermanent", delivery}, r.Assumptions, part)
		require.True(t, r.Exercised, part)
	}
	// The redelivery is spent once: a second transient rejection has no row result.
	once := c.built[closeBounded].Table
	require.Equal(t, []Result{{Outcome: "accepted", State: "open-none-done-succeeded-none-none-original-succeeded",
		Facts: []string{"nexusOperationCompleted"}}},
		plainResults(t, once, "open-none-done-succeeded-retried-succeeded-none-none-complete-succeeded"))
	require.Equal(t, "rejectedTransient",
		plainResults(t, once, "open-none-done-succeeded-inFlight-succeeded-none-none-complete-succeeded")[1].Outcome)
}

// N11: an outcome retained for a closed run reaches an owner only when a reset runs. Under the
// recovery assumption, which makes the reset fair, no cycle keeps it waiting; without it the
// redelivered completion is such a cycle. Over the channel that retries until acknowledged the
// deadline is missed either way, and over the one that redelivers once it is met.
func TestNexusCloseRetainedOutcomeNeedsRecovery(t *testing.T) {
	c := closeModel(t)
	const recovery = "currentOwnerEventuallyRecoversAndReappliesRetainedOutcome"
	assumed := closeProgress(t, c.report, "retainAndRoute", "retainedReachesOwner", umpire.CycleKind)
	require.Contains(t, assumed.Assumptions, recovery)
	require.True(t, assumed.Exercised)
	waiting := closeProgress(t, c.report, "retainAndRoute", "retainedWaitsWithoutRecovery", umpire.CycleKind)
	require.NotContains(t, waiting.Assumptions, recovery)
	require.GreaterOrEqual(t, waiting.Loop, 0)
	for _, step := range waiting.Witness.Steps[waiting.Loop:] {
		require.True(t, strings.HasPrefix(step.Action.Value, "complete-"), step.Action.Value)
		require.True(t, strings.HasPrefix(step.State.Value, "closed-"), step.State.Value)
		require.Contains(t, step.State.Value, "-pending-")
	}
	// Retention is safe on that path: the outcome is kept, and only no owner knows it.
	require.Equal(t, Verified, closeQuery(t, c, "retainAndRoute", "any.outcomePreserved").Kind)
	missed := closeProgress(t, c.report, "retainAndRoute", "retainedReachesOwner", umpire.DeadlineKind)
	require.NotContains(t, taken(missed.Witness)[len(missed.Witness.Steps)-2:], "reset")
	for _, part := range []ProgressKind{umpire.DeadlockKind, umpire.CycleKind, umpire.DeadlineKind} {
		r := closeProgress(t, c.report, closeBounded, "retainedReachesOwner", part)
		require.Contains(t, r.Assumptions, recovery, part)
		require.True(t, r.Exercised, part)
	}
}

// Each result names what it relies on: retention where the policy retains, the redelivery bound and
// the deadline where the machine has them, and nothing else.
func TestNexusCloseReceiptsNameTheirAssumptions(t *testing.T) {
	c := closeModel(t)
	const retention, retry, deadline = "retentionSurvivesCrash", "transientRejectionEventuallyAccepted", "scheduleToCloseExpires"
	for design, want := range map[string][]string{
		"rejectAfterClose":             nil,
		"ackByOriginal":                {retention},
		"retainAndRoute":               {retention},
		"forgetsCancelOnReset":         {retention},
		"truncatesOnReset":             {retention},
		closeBounded:                   {retention, retry},
		"rejectAfterCloseWithDeadline": {retry, deadline},
		"ackByOriginalWithDeadline":    {retention, retry, deadline},
		"retainAndRouteWithDeadline":   {retention, retry, deadline},
	} {
		queries := 0
		for _, r := range c.report.Receipts {
			if r.Key.Owner == design && r.Subject == QuerySubject {
				queries++
				require.Equal(t, want, r.Assumptions, receiptKey(r))
			}
		}
		require.Positive(t, queries, design)
	}
}

// A free search that verifies is bounded past the depth of the table it searches, so it reads every
// step of every state the table reaches: the bound cuts nothing off.
func TestNexusCloseFreeSearchesReachEveryState(t *testing.T) {
	c := closeModel(t)
	free := 0
	for _, r := range c.report.Receipts {
		if r.Subject != QuerySubject || !strings.Contains(r.Key.Name, ".any.") || r.Kind != Verified {
			continue
		}
		free++
		table := c.built[r.Key.Owner].Table
		require.Greater(t, r.Limits.Steps, depth(table), receiptKey(r))
		require.GreaterOrEqual(t, r.Explored, len(table.Reachable), receiptKey(r))
		require.Less(t, r.Explored, r.Limits.Search, receiptKey(r))
		require.True(t, r.Exercised, receiptKey(r))
	}
	// Six of the corrected design, two of it over the bounded channel and three with the deadline.
	require.Equal(t, 11, free)
}

// Every witness of a Query replays against the table of its own design, and no result is a replay
// failure.
func TestNexusCloseWitnessesReplay(t *testing.T) {
	c := closeModel(t)
	replayed := 0
	for _, r := range c.report.Receipts {
		require.NotEqual(t, ReplayFailed, r.Kind, receiptKey(r))
		if r.Witness == nil {
			continue
		}
		replayed++
		require.NoError(t, c.built[r.Key.Owner].Table.Replay(r.Witness), receiptKey(r))
	}
	require.Positive(t, replayed)
}

// The pinned Queries whose paths are the specimen's explore the product states its scratch run
// counted: 7 and 9 for N1 and N1', 4 and 6 for N2 and N2', 8 for N4. The canceled path of N3 has one
// step more here, the delivery of the request to the handler, and so one state more than the
// specimen's 5 and 7. The free searches are over more states than the sketch's, and are not compared.
func TestNexusClosePinnedSearchesExploreTheSpecimensStates(t *testing.T) {
	c := closeModel(t)
	got := map[string]int{}
	want := map[string]int{
		"rejectAfterClose.closedThenFinished":                          7,
		"retainAndRoute.closedThenFinished":                            9,
		"ackByOriginal.resetThenDelivered.ackOnlyWhenKept":             4,
		"ackByOriginal.resetThenDelivered.outcomePreserved":            4,
		"retainAndRoute.resetThenDelivered.ackOnlyWhenKept":            6,
		"retainAndRoute.resetThenDelivered.outcomePreserved":           6,
		"ackByOriginal.canceledAcrossReset":                            5 + 1,
		"rejectAfterClose.canceledAcrossReset":                         7 + 1,
		"retainAndRoute.canceledAcrossReset":                           7 + 1,
		"rejectAfterClose.ackedThenReset":                              8,
		"ackByOriginal.ackedThenReset":                                 8,
		"retainAndRoute.ackedThenReset":                                8,
		"rejectAfterCloseWithDeadline.closedThenFinished":              7,
		"retainAndRouteWithDeadline.closedThenFinished":                9,
		"ackByOriginalWithDeadline.resetThenDelivered.ackOnlyWhenKept": 4,
	}
	for name := range want {
		design, query, _ := strings.Cut(name, ".")
		got[name] = closeQuery(t, c, design, query).Explored
	}
	require.Equal(t, want, got)
}
