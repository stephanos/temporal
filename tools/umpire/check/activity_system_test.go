package check

// The standalone activity's system contract, lifted from model/temporal/features/activity/standalone/{record,withTaskQueue}
// and the shared task queue it composes, model/temporal/foundations/taskqueue, into ir/activity-standalone-record.json and
// checked here through Check alone: the provider checks below are the queue's own. What each test expects is the
// trace oracle of model/specimens/activity.md it names, in the keys of the lifted Model: the
// specimen's supported sketch folds the delivery into the record's state, and this Model keeps the
// record and the queue apart, so a row is keyed by the record's state, or by both members' states.
//
// No expectation here is an explored-state count: the specimen's counts are those of its sketch, which
// lifter/testdata/lifts/Admission.scala is and checking_test.go pins.

import (
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
	umpirespb "go.temporal.io/server/api/umpire/v1"
	umpire "go.temporal.io/server/tools/umpire/internal/engine"
	"go.temporal.io/server/tools/umpire/interp"
	"go.temporal.io/server/tools/umpire/ir"
)

const activitySystemIR = "../../../model/ir/activity-standalone-record.json"

type checkedModel struct {
	model  *umpirespb.Model
	report *Report
	built  map[string]*interp.Machine
}

var activityRecord = sync.OnceValues(func() (*checkedModel, error) {
	m, err := ir.Load(activitySystemIR)
	if err != nil {
		return nil, err
	}
	// The stale design's rejected refinement is its machine's, not an error of the Model.
	return checkedOnce(m)
})

func systemModel(t *testing.T) *checkedModel {
	t.Helper()
	c, err := activityRecord()
	require.NoError(t, err)
	return c
}

// last is the final step of a witness.
func last(t *testing.T, w *Trace) TraceStep {
	t.Helper()
	require.NotNil(t, w)
	require.NotEmpty(t, w.Steps)
	return w.Steps[len(w.Steps)-1]
}

// depth is how many steps the farthest state a table reaches is from a start.
func depth(table *interp.Table) int {
	at := map[string]int{}
	queue := slices.Clone(table.Starts)
	for _, s := range queue {
		at[s] = 0
	}
	deepest := 0
	for len(queue) > 0 {
		s := queue[0]
		queue = queue[1:]
		for _, row := range table.RowsFrom(s) {
			for _, res := range row.Results {
				if _, seen := at[res.State]; !seen {
					at[res.State] = at[s] + 1
					deepest = max(deepest, at[res.State])
					queue = append(queue, res.State)
				}
			}
		}
	}
	return deepest
}

// lastRow is the key of the row a witness's final step takes.
func lastRow(t *testing.T, w *Trace) string {
	t.Helper()
	from := w.Initial.Value
	if n := len(w.Steps); n > 1 {
		from = w.Steps[n-2].State.Value
	}
	return from + "-" + last(t, w).Action.Value
}

func factsOf(s TraceStep) []string {
	var out []string
	for _, f := range s.Facts {
		out = append(out, f.Value)
	}
	return out
}

// plainResults is a row's results without their typed steps.
func plainResults(t *testing.T, table *interp.Table, key string) []interp.Result {
	t.Helper()
	var out []interp.Result
	for _, res := range table.Rows[rowIndex(t, table, key)].Results {
		res.Step = nil
		out = append(out, res)
	}
	return out
}

const (
	commitFails = "the durable update fails: nothing is admitted and the message stays deliverable"
	current     = "query activityRecord activityRecord."
	stale       = "query trustingActivityRecord trustingActivityRecord."
)

// Every declaration of the system contract has one result, and none is left unanswered. The kinds are
// the oracles' verdicts: the corrected design keeps every promise over every queue, the stale design
// breaks each one, the detailed queue stands in for the opaque one, and each violating provider fails.
// The Properties each design and composition receives from its capabilities are verified by the Queries
// they generate, `<machine>.<property>` over a free search of the machine, which answer as the retired
// `<machine>.any.notAdmittedWhilePaused` and `<machine>.any.terminalStays` did. A design's closed
// rejection, which no hand-written Query asked, fails on both: the corrected design meets a delivery
// after the activity timed out as accepted, owing the queue an answer and recording
// admissionRejected, where the Property wants the state kept and notFound; the stale design's free search
// meets the monitor's second admission first.
func TestActivitySystemResults(t *testing.T) {
	report := systemModel(t).report
	require.Empty(t, report.Unsupported())
	want := map[string]ReceiptKind{
		"refinement activitySystem activityProduct":         Verified,
		"refinement activityRecord activityProduct":         Verified,
		"refinement trustingActivityRecord activityProduct": RefinementRejected,

		"refinement taskQueueSystem taskQueueProduct":                    Verified,
		"refinement lossyMatchingQueue taskQueueProductUnderStorageLoss": Verified,
		"refinement forgetfulQueue taskQueueProduct":                     RefinementRejected,
		"refinement volatileQueue taskQueueProduct":                      RefinementRejected,

		"composition recordOverMatching":         Verified,
		"composition trustingRecordOverMatching": Verified,
		"composition recordOverLossyMatching":    Verified,
		"composition recordOverForgetful":        RefinementRejected,
		"composition recordOverVolatile":         RefinementRejected,

		"query activitySystem competingTimers.scheduleToStartFirst": Found,
		"query activitySystem competingTimers.scheduleToCloseFirst": Found,

		// The product's capability Properties, which the system contract's Model carries with the product.
		"query activityProduct activityProduct.terminalStatesAreFinal":    Verified,
		"query activityProduct activityProduct.pausedIsNotDispatched":     Verified,
		"query activityProduct activityProduct.closedIsRejectedUniformly": Verified,

		current + "staleDelivery":                    Verified,
		current + "admittedBeforePause":              Verified,
		current + "duplicateDelivery":                Verified,
		current + "duplicateDelivery.monitored":      Verified,
		current + "startedAfterCompletion.monitored": Verified,
		current + "pausedIsNotDispatched":            Verified,
		current + "any.atMostOneActive":              Verified,
		current + "terminalStatesAreFinal":           Verified,
		current + "scheduleToStartFirst":             Found,
		current + "scheduleToCloseFirst":             Found,
		current + "product.pausedIsNotDispatched":    Verified,

		stale + "staleDelivery":                    Counterexample,
		stale + "admittedBeforePause":              Verified,
		stale + "duplicateDelivery":                Counterexample,
		stale + "duplicateDelivery.monitored":      Counterexample,
		stale + "startedAfterCompletion.monitored": Counterexample,
		stale + "pausedIsNotDispatched":            Counterexample,
		stale + "any.atMostOneActive":              Counterexample,
		stale + "terminalStatesAreFinal":           Counterexample,
		stale + "scheduleToStartFirst":             Found,
		stale + "scheduleToCloseFirst":             Found,
		stale + "product.pausedIsNotDispatched":    RefinementRejected,

		"query lossyMatchingQueue lossyMatchingQueue.storageLoss": Found,
	}
	// A crash cut is found where the message survives the crash and is delivered after it.
	for provider, cuts := range map[string][6]ReceiptKind{
		"taskQueueSystem":    {Found, Found, Found, Found, Verified, Verified},
		"lossyMatchingQueue": {Found, Found, Found, Found, Verified, Counterexample},
		"forgetfulQueue":     {NotFound, NotFound, Found, Found, Verified, Counterexample},
		"volatileQueue":      {Found, Found, NotFound, NotFound, Verified, Counterexample},
	} {
		for i, name := range []string{"crashAfterInvocation", "crashAfterSyncMatch", "crashAfterPersistence",
			"crashAfterDelivery", "crashAfterAcknowledgment", "any.committedStays"} {
			want["query "+provider+" "+provider+"."+name] = cuts[i]
		}
	}
	for composition, kinds := range map[string][7]ReceiptKind{
		"recordOverQueue":         {Verified, Verified, Verified, Verified, Verified, Verified, Verified},
		"trustingRecordOverQueue": {Counterexample, Verified, Counterexample, Verified, Counterexample, Counterexample, Counterexample},
	} {
		for i, name := range []string{"staleDelivery", "admittedBeforePause", "duplicateDelivery", "failedCommit",
			"pausedIsNotDispatched", "any.atMostOneActive", "terminalStatesAreFinal"} {
			want["query "+composition+" "+composition+"."+name] = kinds[i]
		}
	}
	for composition, kinds := range map[string][7]ReceiptKind{
		"recordOverMatching":         {Verified, Verified, Verified, Verified, Verified, Verified, Verified},
		"recordOverLossyMatching":    {Verified, Verified, Verified, Verified, Verified, Verified, Verified},
		"trustingRecordOverMatching": {Counterexample, Verified, Counterexample, Counterexample, Counterexample, Counterexample, Counterexample},
	} {
		for i, name := range []string{"staleDelivery", "admittedBeforePause", "deliveredAgainAfterLostAck",
			"crashAfterAdmissionCommit", "pausedIsNotDispatched", "any.atMostOneActive", "terminalStatesAreFinal"} {
			want["query "+composition+" "+composition+"."+name] = kinds[i]
		}
	}
	require.Equal(t, want, kinds(report))
	for _, r := range report.Receipts {
		t.Logf("%s: %s, %d explored within %s (%d steps), assuming %v", receiptKey(r), r.Kind, r.Explored, r.Limits.Name,
			r.Limits.Steps, r.Assumptions)
	}
}

// A1 and A1': the stale design admits the old message after the pause, and the corrected design
// answers it without admitting anything. A5: read through its refinement, the stale design's product
// Property has no answer but the refinement's rejection, which is no counterexample.
func TestActivityStaleDeliveryAfterPause(t *testing.T) {
	c := systemModel(t)
	found := receiptOf(t, c.report, stale+"staleDelivery")
	require.Equal(t, []string{"dispatch", "pause", "poll"}, taken(found.Witness))
	require.Equal(t, []string{"scheduled-none-settled-dispatch", "scheduled-none-settled-pause",
		"paused-none-settled-poll"}, found.Rows)
	require.Equal(t, "started-one-owed", last(t, found.Witness).State.Value)
	require.Equal(t, []string{"statusStarted", "attemptAdmitted"}, factsOf(last(t, found.Witness)))
	require.Empty(t, found.Monitor, "the Property fails, not a monitor")

	// The free searches of the design and of the queue composition both isolate the Property. The
	// monitor's own generated Query below pins its two-poll counterexample independently.
	free := receiptOf(t, c.report, stale+"pausedIsNotDispatched")
	require.Empty(t, free.Monitor)
	require.Equal(t, []string{"pause", "poll"}, taken(free.Witness))
	overQueue := receiptOf(t, c.report, "query trustingRecordOverQueue trustingRecordOverQueue.pausedIsNotDispatched")
	require.Empty(t, overQueue.Monitor)
	require.Equal(t, []string{"dispatch", "activity_pause", "admit"}, taken(overQueue.Witness))

	// The corrected design's step is a stutter that records nothing the product sees.
	require.Equal(t, []interp.Result{{Outcome: "accepted", State: "paused-none-owed", Facts: []string{"admissionRejected"}}},
		plainResults(t, c.built["activityRecord"].Table, "paused-none-settled-poll"))
	kept := receiptOf(t, c.report, current+"staleDelivery")
	require.True(t, kept.Exercised)
	require.Nil(t, kept.Witness)

	rejected := receiptOf(t, c.report, "refinement trustingActivityRecord activityProduct")
	require.Equal(t, umpire.RefinementUnmatched, rejected.Failure)
	require.Equal(t, "paused-none-settled-poll", lastRow(t, rejected.Witness))
	through := receiptOf(t, c.report, stale+"product.pausedIsNotDispatched")
	require.Equal(t, RefinementRejected, through.Kind)
	require.Equal(t, rejected.Failure, through.Failure)
	require.Equal(t, rejected.Witness, through.Witness)
}

// A2: work admitted before the pause is legal in both designs, and differs from A1 only in which
// commit came first.
func TestActivityAdmittedBeforePause(t *testing.T) {
	c := systemModel(t)
	for _, key := range []string{current + "admittedBeforePause", stale + "admittedBeforePause",
		"query recordOverQueue recordOverQueue.admittedBeforePause", "query trustingRecordOverQueue trustingRecordOverQueue.admittedBeforePause",
		"query recordOverMatching recordOverMatching.admittedBeforePause",
		"query trustingRecordOverMatching trustingRecordOverMatching.admittedBeforePause"} {
		r := receiptOf(t, c.report, key)
		require.Equal(t, Verified, r.Kind, key)
		require.True(t, r.Exercised, key)
	}
	for _, design := range []string{"activityRecord", "trustingActivityRecord"} {
		require.Equal(t, []interp.Result{{Outcome: "accepted", State: "pausedWhileHeld-one-owed", Facts: []string{"statusPaused"}}},
			plainResults(t, c.built[design].Table, "started-one-owed-pause"), design)
	}
}

// A3 and A8: a second delivery of one message. The stale design admits a second attempt; the
// corrected design answers it and admits nothing. The watching monitor counts the admissions from
// what the steps record, and finds the same violation where the Property asked is one the path keeps.
func TestActivityDuplicateDelivery(t *testing.T) {
	c := systemModel(t)
	doubled := receiptOf(t, c.report, stale+"duplicateDelivery")
	require.Equal(t, []string{"dispatch", "poll", "poll"}, taken(doubled.Witness))
	require.Equal(t, "started-two-owed", last(t, doubled.Witness).State.Value)
	require.Equal(t, []string{"poll", "poll"}, taken(receiptOf(t, c.report, stale+"any.atMostOneActive").Witness))

	monitored := receiptOf(t, c.report, stale+"duplicateDelivery.monitored")
	require.Equal(t, "atMostOneActiveAttempt", monitored.Monitor)
	require.Equal(t, []string{"dispatch", "poll", "poll"}, taken(monitored.Witness))
	require.Equal(t, []MonitorVerdict{{Name: "atMostOneActiveAttempt", State: "two", Verdict: umpire.MonitorViolated},
		{Name: "terminalFinality", State: "open", Verdict: umpire.MonitorHeld}}, monitored.Monitors)
	// Both monitors read the corrected design too, and hold on everything its free search reaches.
	require.Equal(t, []MonitorVerdict{{Name: "atMostOneActiveAttempt", Verdict: umpire.MonitorHeld},
		{Name: "terminalFinality", Verdict: umpire.MonitorHeld}}, receiptOf(t, c.report, current+"any.atMostOneActive").Monitors)

	// The stale design's second admission reads as a stutter of the product that records a status the
	// product sees, so the visible projection refuses it too.
	require.Equal(t, []interp.Result{
		{Outcome: "accepted", State: "started-two-owed", Facts: []string{"statusStarted", "attemptAdmitted"}, Choice: "admissionCommits"},
		{Outcome: "accepted", State: "started-one-owed", Facts: []string{"admissionCommitFailed"}, Because: commitFails, Choice: "admissionCommitFails"},
	}, plainResults(t, c.built["trustingActivityRecord"].Table, "started-one-owed-poll"))
	require.Equal(t, []interp.Result{{Outcome: "accepted", State: "started-one-owed", Facts: []string{"admissionRejected"}}},
		plainResults(t, c.built["activityRecord"].Table, "started-one-owed-poll"))

	// Over the queues the second delivery is the interface's own: a message not yet acknowledged.
	over := receiptOf(t, c.report, "query trustingRecordOverQueue trustingRecordOverQueue.duplicateDelivery")
	require.Equal(t, []string{"dispatch", "admit", "admit"}, taken(over.Witness))
	require.Equal(t, "started-two-owed_deliveredTwice", last(t, over.Witness).State.Value)
	require.Equal(t, over.Witness, receiptOf(t, c.report, "query trustingRecordOverQueue trustingRecordOverQueue.any.atMostOneActive").Witness)

	// Over the detailed queue it takes a lost acknowledgment, or a crash before the acknowledgment.
	lost := receiptOf(t, c.report, "query trustingRecordOverMatching trustingRecordOverMatching.deliveredAgainAfterLostAck")
	require.Equal(t, []string{"dispatch", "queue_addActivityTask", "queue_persistTask", "admit", "queue_ackLoss", "admit"},
		taken(lost.Witness))
	require.Equal(t, "started-two-owed_persisted-true-twice", last(t, lost.Witness).State.Value)
	crashed := receiptOf(t, c.report, "query trustingRecordOverMatching trustingRecordOverMatching.crashAfterAdmissionCommit")
	require.Equal(t, []string{"dispatch", "queue_addActivityTask", "queue_syncMatch", "admit", "queue_crash",
		"queue_addActivityTask", "queue_syncMatch", "admit"}, taken(crashed.Witness))

	// The corrected design meets the redelivery with the attempt it committed, and admits no second.
	matching := composedTable(t, c.model, "recordOverMatching")
	require.Equal(t, []interp.Result{{Outcome: "activity_accepted", State: "started-one-owed_persisted-true-twice",
		Facts: []string{"activity_admissionRejected", "queue_delivered"}}},
		plainResults(t, matching, "started-one-owed_persisted-false-once-admit"))
	for _, key := range []string{"deliveredAgainAfterLostAck", "crashAfterAdmissionCommit"} {
		require.True(t, receiptOf(t, c.report, "query recordOverMatching recordOverMatching."+key).Exercised, key)
	}
}

// A4: an activity that is over is started again by the stale design alone. The terminal-finality
// monitor sees it on a path whose Property asks only for one active attempt.
func TestActivityTerminalFinality(t *testing.T) {
	c := systemModel(t)
	reopened := receiptOf(t, c.report, stale+"startedAfterCompletion.monitored")
	require.Equal(t, "terminalFinality", reopened.Monitor)
	require.Equal(t, []string{"dispatch", "poll", "respondCompleted", "poll"}, taken(reopened.Witness))
	require.Equal(t, "started-one-owed", last(t, reopened.Witness).State.Value)
	require.Equal(t, []MonitorVerdict{{Name: "atMostOneActiveAttempt", State: "one", Verdict: umpire.MonitorHeld},
		{Name: "terminalFinality", State: "reopened", Verdict: umpire.MonitorViolated}}, reopened.Monitors)

	// The free searches over a queue, which no monitor reads, end on the step that leaves the end. Each
	// is the one the composition's capabilities generate for terminalStatesAreFinal.
	for _, key := range []string{"query trustingRecordOverQueue trustingRecordOverQueue.terminalStatesAreFinal",
		"query trustingRecordOverMatching trustingRecordOverMatching.terminalStatesAreFinal"} {
		w := receiptOf(t, c.report, key).Witness
		require.NotNil(t, w, key)
		before, after := w.Steps[len(w.Steps)-2].State.Value, last(t, w).State.Value
		require.True(t, strings.HasPrefix(before, "timedOut-") || strings.HasPrefix(before, "completed-"), "%s: %s", key, before)
		require.True(t, strings.HasPrefix(after, "started-"), "%s: %s", key, after)
	}
}

// A6: with both deadlines armed and no attempt started, each may fire first, and the status records
// which did. Both rows leave one state, so neither order is pruned.
func TestActivityCompetingTimers(t *testing.T) {
	c := systemModel(t)
	for query, table := range map[string][2]string{
		"query activitySystem competingTimers.": {"activitySystem", "scheduled-now-0-expires-expires-unset-unset-unlimited"},
		current:                                 {"activityRecord", "scheduled-none-settled"},
		stale:                                   {"trustingActivityRecord", "scheduled-none-settled"},
	} {
		mm := c.built[table[0]]
		over := map[string]string{}
		for _, timer := range []string{"scheduleToStart", "scheduleToClose"} {
			results := plainResults(t, mm.Table, table[1]+"-"+timer)
			require.Len(t, results, 1)
			require.Equal(t, []string{"statusTimedOut-" + timer}, results[0].Facts)
			over[timer] = results[0].State

			found := receiptOf(t, c.report, query+timer+"First")
			require.Equal(t, timer, last(t, found.Witness).Action.Value)
			require.Equal(t, []string{"statusTimedOut-" + timer}, factsOf(last(t, found.Witness)))
		}
		// Once one has fired the activity is over and the other is disabled.
		for _, timer := range []string{"scheduleToStart", "scheduleToClose"} {
			require.True(t, strings.HasPrefix(over[timer], "timedOut-"), over[timer])
			require.True(t, disabled(mm, over[timer], "scheduleToStart"))
			require.True(t, disabled(mm, over[timer], "scheduleToClose"))
		}
	}
}

// A7: a failed commit at admission records no attempt, leaves the activity scheduled, and leaves the
// message with the queue. What sets it apart from the commit is the missing attemptAdmitted.
func TestActivityFailedCommit(t *testing.T) {
	c := systemModel(t)
	require.Equal(t, []interp.Result{
		{Outcome: "activity_accepted", State: "started-one-owed_deliveredOnce",
			Facts: []string{"activity_statusStarted", "activity_attemptAdmitted", "queue_delivered"}},
		{Outcome: "activity_accepted", State: "scheduled-none-settled_deliveredOnce",
			Facts: []string{"activity_admissionCommitFailed", "queue_delivered"}},
	}, plainResults(t, composedTable(t, c.model, "recordOverQueue"), "scheduled-none-settled_committed-admit"))
	require.Equal(t, commitFails, c.built["recordMember"].Table.Rows[rowIndex(t, c.built["recordMember"].Table,
		"scheduled-none-settled-poll")].Results[1].Because)
	for _, composition := range []string{"recordOverQueue", "trustingRecordOverQueue"} {
		require.Equal(t, Verified, receiptOf(t, c.report, "query "+composition+" "+composition+".failedCommit").Kind)
	}
	// A commit that failed owes no answer, so the message cannot be acknowledged: it is delivered again.
	table := composedTable(t, c.model, "recordOverQueue")
	actions := []string{}
	for _, row := range table.RowsFrom("scheduled-none-settled_deliveredOnce") {
		actions = append(actions, row.Action)
	}
	require.Contains(t, actions, "admit")
	require.NotContains(t, actions, "settle")
}

// A9: an ordinary crash loses what is only in memory and nothing committed, at every cut of the route.
// The provider that drops history's task at the invocation, and the one whose crash wipes persisted
// tasks, each lose the message at their cut and nowhere else.
func TestActivityCrashCuts(t *testing.T) {
	c := systemModel(t)
	for _, row := range c.built["taskQueueSystem"].Table.Rows {
		if row.Action != "crash" {
			continue
		}
		for _, res := range row.Results {
			require.Equal(t, strings.HasPrefix(row.Source, "nowhere-"), strings.HasPrefix(res.State, "nowhere-"), row.Key)
			require.Equal(t, []string{"crashed"}, res.Facts, row.Key)
		}
	}
	held := map[string]string{
		"crashAfterInvocation":  "persisted-true-once",
		"crashAfterSyncMatch":   "reserved-true-once",
		"crashAfterPersistence": "persisted-true-once",
		// Delivered before the crash and again after it: twice, and no third time.
		"crashAfterDelivery": "persisted-true-twice",
	}
	for cut, state := range held {
		found := receiptOf(t, c.report, "query taskQueueSystem taskQueueSystem."+cut)
		require.Contains(t, taken(found.Witness), "crash", cut)
		require.Equal(t, state, last(t, found.Witness).State.Value, cut)
	}
	require.True(t, disabled(c.built["taskQueueSystem"], "persisted-false-twice", "deliver"))
	// After the acknowledgment nothing is outstanding, and a crash changes nothing.
	require.Equal(t, []interp.Result{{Outcome: "internal", State: "nowhere-false-never", Facts: []string{"crashed"}}},
		plainResults(t, c.built["taskQueueSystem"].Table, "nowhere-false-never-crash"))

	for provider, witness := range map[string][]string{
		"forgetfulQueue": {"enqueue", "addActivityTask", "crash"},
		"volatileQueue":  {"enqueue", "addActivityTask", "persistTask", "crash"},
	} {
		lost := receiptOf(t, c.report, "query "+provider+" "+provider+".any.committedStays")
		require.Equal(t, witness, taken(lost.Witness), provider)
		require.Equal(t, "nowhere-false-never", last(t, lost.Witness).State.Value, provider)
		require.Equal(t, []string{"crashed"}, factsOf(last(t, lost.Witness)), provider)
	}
}

// The scoped substitution: within the composition the detailed queue stands in for the opaque one,
// because it refines the interface, and the admission promises keep their verdicts over it. A check
// over the opaque queue names the assumption it rests on, and one over the detailed queue does not.
func TestActivityQueueSubstitution(t *testing.T) {
	c := systemModel(t)
	const opaque = "taskQueueProduct.opaque"
	for _, composition := range []string{"recordOverMatching", "trustingRecordOverMatching"} {
		replaced := receiptOf(t, c.report, "composition "+composition)
		require.Equal(t, Verified, replaced.Kind)
		require.NotContains(t, replaced.Assumptions, opaque)
		require.NotEmpty(t, replaced.Target)
		require.NotEmpty(t, replaced.Fingerprint)
	}
	held := receiptOf(t, c.report, "refinement taskQueueSystem taskQueueProduct")
	require.Equal(t, len(c.built["taskQueueSystem"].Table.Rows), held.Explored)
	require.Empty(t, held.Holes)

	for _, name := range []string{"staleDelivery", "admittedBeforePause", "pausedIsNotDispatched", "any.atMostOneActive",
		"terminalStatesAreFinal"} {
		overQueue := receiptOf(t, c.report, "query recordOverQueue recordOverQueue."+name)
		overMatching := receiptOf(t, c.report, "query recordOverMatching recordOverMatching."+name)
		require.Equal(t, Verified, overQueue.Kind, name)
		require.Equal(t, overQueue.Kind, overMatching.Kind, name)
		require.Equal(t, []string{opaque}, overQueue.Assumptions, name)
		require.Empty(t, overMatching.Assumptions, name)
		require.NotZero(t, overMatching.Limits.Steps, name)
		require.Positive(t, overMatching.Explored, name)
	}

	// The stale design's counterexample survives the substitution, step for step.
	overQueue := receiptOf(t, c.report, "query trustingRecordOverQueue trustingRecordOverQueue.staleDelivery")
	require.Equal(t, []string{"dispatch", "activity_pause", "admit"}, taken(overQueue.Witness))
	require.Equal(t, "started-one-owed_deliveredOnce", last(t, overQueue.Witness).State.Value)
	require.Equal(t, overQueue.Witness, receiptOf(t, c.report, "query trustingRecordOverQueue trustingRecordOverQueue.pausedIsNotDispatched").Witness)
	overMatching := receiptOf(t, c.report, "query trustingRecordOverMatching trustingRecordOverMatching.staleDelivery")
	require.Equal(t, []string{"dispatch", "queue_addActivityTask", "queue_persistTask", "activity_pause", "admit"},
		taken(overMatching.Witness))
	require.Equal(t, "started-one-owed_persisted-true-once", last(t, overMatching.Witness).State.Value)
	free := receiptOf(t, c.report, "query trustingRecordOverMatching trustingRecordOverMatching.pausedIsNotDispatched")
	require.Len(t, free.Witness.Steps, 5)
	require.Equal(t, "admit", last(t, free.Witness).Action.Value)
	require.True(t, strings.HasPrefix(free.Witness.Steps[3].State.Value, "paused-none-settled_"))
}

// V1 and V2: each violating provider fails the refinement at its crash, with a witness that replays,
// and the composition it would stand in is rejected with it. Neither is excused by storage loss.
func TestActivityViolatingProviders(t *testing.T) {
	c := systemModel(t)
	for provider, want := range map[string]struct {
		composition string
		witness     []string
		row         string
	}{
		"forgetfulQueue": {"recordOverForgetful", []string{"enqueue", "addActivityTask", "crash"}, "invoked-false-never-crash"},
		"volatileQueue":  {"recordOverVolatile", []string{"enqueue", "addActivityTask", "persistTask", "crash"}, "persisted-false-never-crash"},
	} {
		t.Run(provider, func(t *testing.T) {
			rejected := receiptOf(t, c.report, "refinement "+provider+" taskQueueProduct")
			require.Equal(t, umpire.RefinementUnmatched, rejected.Failure)
			require.Equal(t, want.witness, taken(rejected.Witness))
			require.Equal(t, want.row, lastRow(t, rejected.Witness))
			require.Equal(t, "nowhere-false-never", last(t, rejected.Witness).State.Value)
			require.NoError(t, c.built[provider].Table.Replay(rejected.Witness))
			require.NotContains(t, rejected.Assumptions, "storageLoss")

			composed := receiptOf(t, c.report, "composition "+want.composition)
			require.Equal(t, RefinementRejected, composed.Kind)
			require.Equal(t, rejected.Failure, composed.Failure)
			require.Equal(t, rejected.Witness, composed.Witness)
			require.NotContains(t, composed.Assumptions, "storageLoss")

			// The fault is the ordinary crash: the provider has no storage-loss step to blame.
			table := c.built[provider].Table
			require.Equal(t, table.ActionAtom("crash"), last(t, rejected.Witness).Action)
			require.NotContains(t, table.Actions, "storageLoss")
		})
	}
}

// Storage loss is a transition of its own, which only the machines that assume it have, and every
// result of a check over them names the assumption.
func TestActivityStorageLossIsAssumed(t *testing.T) {
	c := systemModel(t)
	const assumed = "storageLoss"
	lossy := c.built["lossyMatchingQueue"].Table
	require.Contains(t, lossy.Actions, "storageLoss")
	for _, name := range []string{"taskQueueSystem", "forgetfulQueue", "volatileQueue", "taskQueueProduct"} {
		require.NotContains(t, c.built[name].Table.Actions, "storageLoss", name)
	}
	ids := map[string]bool{}
	for _, fault := range []string{"crash", "ackLoss", "storageLoss"} {
		ids[lossy.ActionAtom(fault).ID] = true
	}
	require.Len(t, ids, 3)

	dropped := receiptOf(t, c.report, "query lossyMatchingQueue lossyMatchingQueue.storageLoss")
	require.Equal(t, []string{"enqueue", "addActivityTask", "persistTask", "storageLoss"}, taken(dropped.Witness))
	require.Equal(t, "nowhere-false-never", last(t, dropped.Witness).State.Value)
	require.Equal(t, []string{"storageLost"}, factsOf(last(t, dropped.Witness)))

	// With the fault enabled a committed message is lost by it, and by nothing else.
	lost := receiptOf(t, c.report, "query lossyMatchingQueue lossyMatchingQueue.any.committedStays")
	require.Equal(t, []string{"enqueue", "storageLoss"}, taken(lost.Witness))

	for _, r := range c.report.Receipts {
		over := r.Key.Owner == "lossyMatchingQueue" || r.Key.Owner == "recordOverLossyMatching"
		if over {
			require.Contains(t, r.Assumptions, assumed, receiptKey(r))
		} else {
			require.NotContains(t, r.Assumptions, assumed, receiptKey(r))
		}
	}
	// The interface and the provider that replaces it name the fault's assumption alike. The provider
	// declares it itself, so the composition keeps it; what only the interface assumes, its opaqueness,
	// is none of the composition's.
	replaced := receiptOf(t, c.report, "composition recordOverLossyMatching")
	require.Equal(t, []string{assumed}, replaced.Assumptions)
	declared := map[string][]string{}
	for _, name := range []string{"taskQueueProductUnderStorageLoss", "lossyMatchingQueue"} {
		for _, a := range c.built[name].Assumptions {
			declared[name] = append(declared[name], a.GetName())
		}
	}
	require.Equal(t, map[string][]string{"taskQueueProductUnderStorageLoss": {"taskQueueProduct.opaque", assumed},
		"lossyMatchingQueue": {assumed}}, declared)
	var named int
	for _, a := range c.model.GetAssumptions() {
		if strings.Contains(a.GetName(), "torageLoss") {
			named++
		}
	}
	require.Equal(t, 1, named, "the storage-loss assumption has one name")
}

// What the admission scope leaves out is absent, not unknown: no unpause, cancel or terminate, and
// no failed or canceled answer. No machine of the contract has a hole.
func TestActivitySystemExclusionsAreDisabled(t *testing.T) {
	c := systemModel(t)
	for _, design := range []string{"activityRecord", "trustingActivityRecord", "recordMember", "trustingRecordMember"} {
		mm := c.built[design]
		require.Empty(t, mm.Holes, design)
		for _, class := range []string{"unpause", "requestCancel", "terminate",
			"respondFailed-fatal", "respondFailed-retryable", "respondCanceled"} {
			require.NotContains(t, mm.Table.Actions, class, "%s: %s", design, class)
		}
	}
	for name, mm := range c.built {
		require.Empty(t, mm.Holes, name)
	}
	for _, r := range c.report.Receipts {
		require.Empty(t, r.Holes, receiptKey(r))
		require.NotEqual(t, ReplayFailed, r.Kind, receiptKey(r))
	}
}

// A free search that verifies the corrected design is bounded past the depth of the table it searches,
// so it reads every step of every state the table reaches: the bound cuts nothing off. The free
// searches are the shared `any` Scenario's and those the capabilities generate for each Property.
func TestActivityFreeSearchesReachEveryState(t *testing.T) {
	c := systemModel(t)
	freeScenarios := map[string]bool{}
	for _, s := range c.model.GetScenarios() {
		if s.GetFree() {
			freeScenarios[s.GetMachine()+" "+s.GetName()] = true
		}
	}
	searches := map[string]bool{}
	for _, q := range c.model.GetQueries() {
		if freeScenarios[q.GetScenario().GetMachine()+" "+q.GetScenario().GetName()] {
			searches[q.GetName()] = true
		}
	}
	tables := map[string]*interp.Table{}
	for _, name := range []string{"activityRecord", "taskQueueSystem"} {
		tables[name] = c.built[name].Table
	}
	for _, name := range []string{"recordOverQueue", "recordOverMatching", "recordOverLossyMatching"} {
		tables[name] = composedTable(t, c.model, name)
	}
	free := 0
	for _, r := range c.report.Receipts {
		table := tables[r.Key.Owner]
		if r.Subject != QuerySubject || table == nil || !searches[r.Key.Name] {
			continue
		}
		free++
		require.Equal(t, Verified, r.Kind, receiptKey(r))
		require.Greater(t, r.Limits.Steps, depth(table), receiptKey(r))
		require.GreaterOrEqual(t, r.Explored, len(table.Reachable), receiptKey(r))
		require.Less(t, r.Explored, r.Limits.Search, receiptKey(r))
	}
	require.Equal(t, 13, free)
}
