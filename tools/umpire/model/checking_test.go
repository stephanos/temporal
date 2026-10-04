package model

// The admitted IR bound to the reader's private checker: every expectation here is worked out by hand from the
// fixture's Scala source or from a small counter Model built below, and each control is one mutation
// of a Model that otherwise checks.

import (
	"encoding/json"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	umpirespb "go.temporal.io/server/api/umpire/v1"
	umpire "go.temporal.io/server/tools/umpire/model/internal/checker"
	"google.golang.org/protobuf/proto"
)

func checked(t *testing.T, m *umpirespb.Model) *Report {
	t.Helper()
	return Check(m, DefaultScope)
}

// mutated is the fixture with each mutation applied, recounted as an exploration recounts the
// candidate it derives: each Query that asserts a total asserts the derived Model's count.
func mutated(t *testing.T, fixture string, mutate ...func(m *umpirespb.Model)) *umpirespb.Model {
	t.Helper()
	m := proto.Clone(lifted(t, fixture)).(*umpirespb.Model)
	for _, f := range mutate {
		f(m)
	}
	return recounted(t, m)
}

// recounted is m with each Query that asserts a total asserting m's count (WithTotals).
func recounted(t *testing.T, m *umpirespb.Model) *umpirespb.Model {
	t.Helper()
	out, err := WithTotals(m)
	require.NoError(t, err)
	return out
}

// receiptKey spells a receipt's subject, owner, name and part, those it has.
func receiptKey(r Receipt) string {
	parts := slicesDelete([]string{string(r.Subject), r.Key.Owner, r.Key.Name, string(r.Part)}, func(s string) bool { return s == "" })
	return strings.Join(parts, " ")
}

// kinds is every receipt's kind under its subject, owner, name and part.
func kinds(r *Report) map[string]ReceiptKind {
	out := map[string]ReceiptKind{}
	for _, x := range r.Receipts {
		out[receiptKey(x)] = x.Kind
	}
	return out
}

func receiptOf(t *testing.T, r *Report, key string) Receipt {
	t.Helper()
	for _, x := range r.Receipts {
		if receiptKey(x) == key {
			return x
		}
	}
	require.Failf(t, "no receipt", "no receipt %q among %v", key, kinds(r))
	return Receipt{}
}

// holes is a receipt's holes without what provenance and the search's path give them.
func holes(r Receipt) []HoleReach {
	var out []HoleReach
	for _, h := range r.Holes {
		h.Position, h.Prefix = "", nil
		out = append(out, h)
	}
	return out
}

// taken is the action classes a witness takes, in order.
func taken(w *Trace) []string {
	var out []string
	for _, s := range w.Steps {
		out = append(out, s.Action.Value)
	}
	return out
}

const (
	crashHole = admDeclaredPkg + "crashUnmodeled"
	declared  = "fixture.declarations"
)

// noCrash unbinds disk's crash, the one step of the fixture that reaches a hole.
func noCrash(m *umpirespb.Model) {
	disk := admMachine(m, "disk")
	disk.Steps = slicesDelete(disk.GetSteps(), func(b *umpirespb.StepBinding) bool { return strings.HasSuffix(b.GetAction(), ".crash") })
}

func returning(name string, v *umpirespb.Value) func(m *umpirespb.Model) {
	return func(m *umpirespb.Model) {
		f := functionNamed(m, name)
		f.Body = admLiteral(f.GetBody(), v)
	}
}

// functionNamed is the function of exactly this name, where function takes the first of a suffix.
func functionNamed(m *umpirespb.Model, name string) *umpirespb.Function {
	for _, f := range m.GetFunctions() {
		if f.GetName() == name {
			return f
		}
	}
	return nil
}

func boolValue(b bool) *umpirespb.Value {
	return &umpirespb.Value{Kind: &umpirespb.Value_Bool{Bool: b}}
}

func stage(c string) *umpirespb.Value { return admEnum("fixture.declarations.Stage", c) }

// flushTo makes disk's flush of a staged disk land in this stage.
func flushTo(c string) func(m *umpirespb.Model) {
	return func(m *umpirespb.Model) {
		step := function(m, "Declarations$package$.flushStep").GetBody().GetMatch().GetCases()[0].GetBody().GetList().GetItems()[0]
		target := step.GetConstruct().GetArgs()[1].GetConstruct()
		target.Args[0] = admLiteral(target.GetArgs()[0], stage(c))
	}
}

// diskStarts makes disk start in this stage.
func diskStarts(c string) func(m *umpirespb.Model) {
	return func(m *umpirespb.Model) {
		start := admMachine(m, "disk").GetStarts()[0].GetConstruct()
		start.Args[0] = admLiteral(start.GetArgs()[0], stage(c))
	}
}

// storeAlsoStartsHeld gives the opaque store a second start.
func storeAlsoStartsHeld(m *umpirespb.Model) {
	store := admMachine(m, "store")
	held := proto.Clone(store.GetStarts()[0]).(*umpirespb.Expr)
	held.GetConstruct().Args[0] = admLiteral(held.GetConstruct().GetArgs()[0], admEnum("fixture.declarations.Kept", "held"))
	store.Starts = append(store.Starts, held)
}

// putRecordsStaged makes the store's Property claim that a put records staged, which none does.
func putRecordsStaged(m *umpirespb.Model) {
	contains := functionNamed(m, "store.property.putStores").GetBody().GetBinary()
	contains.Left = admLiteral(contains.GetLeft(), admEnum("fixture.declarations.Fact", "staged"))
}

// ---- Queries and monitors -------------------------------------------------------------------------

func TestQueriesAreAnsweredByTheGenericSearch(t *testing.T) {
	r := checked(t, mutated(t, "declarations", noCrash))
	require.Equal(t, map[string]ReceiptKind{
		"refinement disk store":                      Verified,
		"composition detailedPair":                   Verified,
		"query detailedPair bothPut":                 Unsupported,
		"query disk durableStays":                    Verified,
		"query disk putAccepted":                     Verified,
		"query store putStores":                      Found,
		"query disk putStoresThroughDisk":            Verified,
		"query pair keptTogether":                    Verified,
		"progress disk durableEventually deadlock":   Verified,
		"progress disk durableEventually fair-cycle": Verified,
		"progress disk durableEventually deadline":   Verified,
	}, kinds(r))

	found := receiptOf(t, r, "query store putStores")
	require.Equal(t, ClaimKey{Family: declared, Owner: "store", Name: "putStores"}, found.Key)
	require.Equal(t, ClaimKey{Family: declared, Owner: "store", Name: "putStores"}, found.Property)
	require.Equal(t, ClaimKey{Family: declared, Owner: "store", Name: "putOnce"}, found.Scenario)
	require.Equal(t, []string{"nothing-put"}, found.Rows)
	require.Equal(t, Limits{Name: "two", Steps: 2, Actions: 2, Search: 64}, found.Limits)
	require.Equal(t, "fixture.declarations.target.store", found.Target)
	require.NotEmpty(t, found.Position)

	// The store's Property read on the disk's steps: put records stored, which the store names too.
	through := receiptOf(t, r, "query disk putStoresThroughDisk")
	require.Equal(t, ClaimKey{Family: declared, Owner: "store", Name: "putStores"}, through.Property)
	require.Equal(t, ClaimKey{Family: declared, Owner: "disk", Name: "putThenFlush"}, through.Scenario)
	require.True(t, through.Exercised)
	// empty, staged and durable, each once.
	require.Equal(t, []int{3, 2}, []int{through.Explored, through.Expanded})

	// The store's claim made that put records staged, which no put does: the store has no trace of it,
	// and the disk's put, read as the store's, is where it fails.
	r = checked(t, mutated(t, "declarations", noCrash, putRecordsStaged))
	require.Equal(t, NotFound, receiptOf(t, r, "query store putStores").Kind)
	through = receiptOf(t, r, "query disk putStoresThroughDisk")
	require.Equal(t, []any{Counterexample, []string{"empty-put"}}, []any{through.Kind, through.Rows})

	// A disk whose put records nothing is no step of the store: there is no refinement to read through.
	r = checked(t, mutated(t, "declarations", noCrash, func(m *umpirespb.Model) {
		facts := function(m, "Declarations$package$.putStep").GetBody().GetMatch().GetCases()[0].GetBody().GetList().GetItems()[0].GetConstruct().GetArgs()[2]
		facts.GetList().Items = nil
	}))
	through = receiptOf(t, r, "query disk putStoresThroughDisk")
	require.Equal(t, []any{RefinementRejected, umpire.RefinementUnmatched}, []any{through.Kind, through.Failure})
}

// The corrected design passes every Query; the stale one admits a delivery after a pause, admits a
// redelivered message twice, and reopens a completed activity.
func TestTheAdmissionDesignsAreToldApart(t *testing.T) {
	r := checked(t, lifted(t, "admission"))
	require.Equal(t, map[string]ReceiptKind{
		"refinement currentAdmission activityProduct": Verified,
		"refinement staleAdmission activityProduct":   RefinementRejected,

		"query currentAdmission currentAdmission.staleDelivery":                 Verified,
		"query currentAdmission currentAdmission.admittedBeforePause":           Verified,
		"query currentAdmission currentAdmission.duplicateDelivery":             Verified,
		"query currentAdmission currentAdmission.any.notAdmittedWhilePaused":    Verified,
		"query currentAdmission currentAdmission.any.atMostOneActive":           Verified,
		"query currentAdmission currentAdmission.any.terminalStays":             Verified,
		"query currentAdmission currentAdmission.product.pausedIsNotDispatched": Verified,

		"query staleAdmission staleAdmission.staleDelivery":                 Counterexample,
		"query staleAdmission staleAdmission.admittedBeforePause":           Verified,
		"query staleAdmission staleAdmission.duplicateDelivery":             Counterexample,
		"query staleAdmission staleAdmission.any.notAdmittedWhilePaused":    Counterexample,
		"query staleAdmission staleAdmission.any.atMostOneActive":           Counterexample,
		"query staleAdmission staleAdmission.any.terminalStays":             Counterexample,
		"query staleAdmission staleAdmission.product.pausedIsNotDispatched": RefinementRejected,
	}, kinds(r))

	stale := receiptOf(t, r, "query staleAdmission staleAdmission.staleDelivery")
	require.Equal(t, []string{"scheduled-empty-none-dispatch", "scheduled-queued-none-control-pause", "paused-queued-none-attemptStart"}, stale.Rows)
	require.Equal(t, "started-empty-one", stale.Witness.Steps[2].State.Value)

	rejected := receiptOf(t, r, "refinement staleAdmission activityProduct")
	require.Equal(t, umpire.RefinementUnmatched, rejected.Failure)
	require.Equal(t, []string{"dispatch", "control-pause", "attemptStart"}, taken(rejected.Witness))
}

// The counter Model (below) with a self-loop: skipping stays in place, so a state is reached both
// having skipped and not, which a monitor that remembers a skip keeps apart.
func TestMonitorHistoriesStayApart(t *testing.T) {
	const k = 9
	unwatched := receiptOf(t, checked(t, counter{k: k, skip: true}.model()), "query counter counter.all")
	watched := receiptOf(t, checked(t, counter{k: k, skip: true, monitor: true}.model()), "query counter counter.all")
	require.Equal(t, []ReceiptKind{Verified, Verified}, []ReceiptKind{unwatched.Kind, watched.Kind})
	// Unwatched: the start before any step, and each of 0..k after one. Watched: the start, 1..k
	// without a skip, and 0..k after one.
	require.Equal(t, k+2, unwatched.Explored)
	require.Equal(t, 2*k+2, watched.Explored)
	require.Equal(t, []MonitorVerdict{{Name: "sawSkip", Verdict: umpire.MonitorHeld}}, watched.Monitors)
}

func TestAMonitorSuppressesNoBehavior(t *testing.T) {
	const k = 3
	noSkip := func(c counter) Receipt {
		m := c.model()
		admQuery(m, "counter.all").Property.Name = "noSkip"
		return receiptOf(t, checked(t, m), "query counter counter.all")
	}
	unwatched, watched := noSkip(counter{k: k, skip: true}), noSkip(counter{k: k, skip: true, monitor: true})
	require.Equal(t, Counterexample, unwatched.Kind)
	require.Equal(t, []string{"0-skip"}, unwatched.Rows)
	require.Equal(t, []any{unwatched.Kind, unwatched.Rows, unwatched.Witness}, []any{watched.Kind, watched.Rows, watched.Witness},
		"the watched search finds the same violation on the same path")
	require.Empty(t, watched.Monitor, "the Property fails; the monitor takes nothing from it")

	// A violated monitor is a counterexample of its own, and names itself.
	violated := receiptOf(t, checked(t, counter{k: k, skip: true, monitor: true, violatedOnceSeen: true}.model()), "query counter counter.all")
	require.Equal(t, []any{Counterexample, "sawSkip", []string{"0-skip"}}, []any{violated.Kind, violated.Monitor, violated.Rows})
	require.Equal(t, []MonitorVerdict{{Name: "sawSkip", State: "true", Verdict: umpire.MonitorViolated}}, violated.Monitors)

	// The machine's rows are the same watched and unwatched.
	plain := bind(counter{k: k, skip: true}.model(), DefaultScope)
	seen := bind(counter{k: k, skip: true, monitor: true}.model(), DefaultScope)
	require.Equal(t, rowsJSON(t, plain.subject("counter").table), rowsJSON(t, seen.subject("counter").table))
}

func rowsJSON(t *testing.T, table *Table) string {
	t.Helper()
	require.NotNil(t, table)
	encoded, err := json.Marshal(table.Rows)
	require.NoError(t, err)
	return string(encoded)
}

func TestUnsupportedDeclarationsAreListedAndNeverCounted(t *testing.T) {
	m := mutated(t, "declarations", noCrash, func(m *umpirespb.Model) {
		admQuery(m, "putStoresThroughDisk").Form = umpirespb.Query_FORM_FIND
		admProperty(m, "disk", "durableStays").When = &umpirespb.Property_WhenAction{WhenAction: "put"}
	})
	r := checked(t, m)
	var unsupported []string
	for _, x := range r.Unsupported() {
		require.NotEmpty(t, x.Position, "an unsupported declaration is located")
		require.Nil(t, x.Witness)
		unsupported = append(unsupported, receiptKey(x)+": "+x.Explanation)
	}
	require.Equal(t, []string{
		"query detailedPair bothPut: the member back of detailedPair is disk, which names monitors, and whether a member's " +
			"monitors watch a composition is undefined",
		"query disk durableStays: disk.durableStays is a transition Property about some steps only, and a transition " +
			"Property is about every step",
		"query disk putStoresThroughDisk: a find through a refinement is not supported: only a verify reads a Property " +
			"through one",
	}, unsupported)
	for _, x := range r.Checks() {
		require.NotEqual(t, Unsupported, x.Kind)
	}
	require.Len(t, r.Checks(), len(r.Receipts)-3)

	for kind, isCheck := range map[ReceiptKind]bool{
		Verified: true, Found: true, NotFound: true, Counterexample: true, RefinementRejected: true, Incomplete: true,
		LimitReached: true, Unresolved: true,
		Unsupported: false, AdmissionError: false, DeclarationError: false, ResourceLimit: false, ReplayFailed: false,
	} {
		require.Equal(t, isCheck, kind.IsCheck(), kind)
	}
}

// The close/reset designs, against the trace oracles of specimens/nexus.md: N1 pins the permanent
// rejection after a close, N2 the acknowledgment by the original run after a reset, N3 the canceled
// outcome across a reset and N4 the reset after an acknowledgment. The explored counts are the ones
// the specimen records for each search.
func TestTheCloseResetDesignsAreToldApart(t *testing.T) {
	r := checked(t, lifted(t, "closereset"))
	type answer struct {
		Kind     ReceiptKind
		Explored int
		Monitor  string
	}
	got := map[string]answer{}
	for _, x := range r.Receipts {
		require.Equal(t, QuerySubject, x.Subject)
		got[x.Key.Name] = answer{x.Kind, x.Explored, x.Monitor}
	}
	require.Equal(t, map[string]answer{
		// N1: the closed run rejects the completion permanently, and the outcome is lost.
		"rejectAfterClose.closedThenFinished":                  {Counterexample, 7, ""},
		"rejectAfterClose.resetThenDelivered.ackOnlyWhenKept":  {Verified, 6, ""},
		"rejectAfterClose.resetThenDelivered.outcomePreserved": {Verified, 6, ""},
		"rejectAfterClose.canceledAcrossReset":                 {Verified, 7, ""},
		"rejectAfterClose.ackedThenReset":                      {Verified, 8, ""},
		"rejectAfterClose.any.outcomePreserved":                {Counterexample, 61, ""},
		// A permanent rejection is no acknowledgment, so ackOnlyWhenKept holds of every step; the
		// design's retainedOutcome monitor watches the same search, and is violated where N1 loses
		// the outcome.
		"rejectAfterClose.any.ackOnlyWhenKept": {Counterexample, 61, "retainedOutcome"},

		// N2 and N3: after a reset the original run acknowledges, and the successor never learns.
		"ackByOriginal.closedThenFinished":                  {Verified, 9, ""},
		"ackByOriginal.resetThenDelivered.ackOnlyWhenKept":  {Counterexample, 4, ""},
		"ackByOriginal.resetThenDelivered.outcomePreserved": {Counterexample, 4, ""},
		"ackByOriginal.canceledAcrossReset":                 {Counterexample, 5, ""},
		"ackByOriginal.ackedThenReset":                      {Verified, 8, ""},
		"ackByOriginal.any.outcomePreserved":                {Counterexample, 76, ""},
		"ackByOriginal.any.ackOnlyWhenKept":                 {Counterexample, 76, ""},

		// The corrected design: N1′, N2′, N3, N4, and the free search of N5.
		"retainAndRoute.closedThenFinished":                  {Verified, 9, ""},
		"retainAndRoute.resetThenDelivered.ackOnlyWhenKept":  {Verified, 6, ""},
		"retainAndRoute.resetThenDelivered.outcomePreserved": {Verified, 6, ""},
		"retainAndRoute.canceledAcrossReset":                 {Verified, 7, ""},
		"retainAndRoute.ackedThenReset":                      {Verified, 8, ""},
		"retainAndRoute.any.outcomePreserved":                {Verified, 71, ""},
		"retainAndRoute.any.ackOnlyWhenKept":                 {Verified, 71, ""},
	}, got)
	query := func(name string) Receipt {
		machine, _, _ := strings.Cut(name, ".")
		return receiptOf(t, r, "query "+machine+" "+name)
	}
	last := func(x Receipt) string { return x.Witness.Steps[len(x.Witness.Steps)-1].State.Value }

	// N1, steps 1 to 3.
	n1 := query("rejectAfterClose.closedThenFinished")
	require.Equal(t, []string{"open-false-running-none-none-none-callerClose",
		"closed-false-running-none-none-none-handlerFinish-succeeded",
		"closed-false-done-succeeded-inFlight-succeeded-none-none-complete-succeeded"}, n1.Rows)
	require.Equal(t, []string{"accepted", "accepted", "rejectedPermanent"},
		[]string{n1.Witness.Steps[0].Outcome.Value, n1.Witness.Steps[1].Outcome.Value, n1.Witness.Steps[2].Outcome.Value})
	require.Equal(t, "closed-false-done-succeeded-none-none-none", last(n1))
	// The free search returns the same shape, with failed as the outcome.
	free := query("rejectAfterClose.any.outcomePreserved")
	require.Len(t, free.Rows, 3)
	require.Equal(t, "closed-false-done-failed-none-none-none", last(free))
	require.Equal(t, last(free), last(query("rejectAfterClose.any.ackOnlyWhenKept")))

	// N2, steps 1 to 3, for both promises, pinned and free.
	n2 := []string{"open-false-running-none-none-none-handlerFinish-failed",
		"open-false-done-failed-inFlight-failed-none-none-reset",
		"resetOpen-false-done-failed-inFlight-failed-none-none-complete-failed"}
	for _, name := range []string{"ackByOriginal.resetThenDelivered.ackOnlyWhenKept", "ackByOriginal.resetThenDelivered.outcomePreserved",
		"ackByOriginal.any.ackOnlyWhenKept", "ackByOriginal.any.outcomePreserved"} {
		x := query(name)
		require.Equal(t, n2, x.Rows, name)
		require.Equal(t, "resetOpen-false-done-failed-none-none-none", last(x), name)
	}
	// N3: violated at step 4, with the cancel intent kept across the reset.
	n3 := query("ackByOriginal.canceledAcrossReset")
	require.Len(t, n3.Rows, 4)
	require.Equal(t, "resetOpen-true-done-canceled-none-none-none", last(n3))

	// N1′ and N2′: the corrected design retains at the close and reapplies at the reset, and routes a
	// completion that arrives after the reset to the successor.
	corrected := built(t, lifted(t, "closereset"))["retainAndRoute"]
	for key, want := range map[string]Result{
		"closed-false-done-succeeded-inFlight-succeeded-none-none-complete-succeeded": {Outcome: "retained",
			State: "closed-false-done-succeeded-none-pending-succeeded-none", Facts: []string{}, Choice: "taken"},
		"closed-false-done-succeeded-none-pending-succeeded-none-reset": {Outcome: "accepted",
			State: "resetOpen-false-done-succeeded-none-none-successor-succeeded", Facts: []string{}},
		"resetOpen-false-done-failed-inFlight-failed-none-none-complete-failed": {Outcome: "accepted",
			State: "resetOpen-false-done-failed-none-none-successor-failed", Facts: []string{}, Choice: "taken"},
	} {
		require.Equal(t, want, row(t, corrected, key).Results[0], key)
	}
}

// A composition that replaces nothing, of the store and a disk no monitor watches: the crash hole of a
// staged disk is an unknown pair of the composition, which a search that explores it reads.
func TestAMembersHoleIsAnUnknownPairOfTheComposition(t *testing.T) {
	composed := func(steps int32) *Report {
		return checked(t, mutated(t, "declarations", func(m *umpirespb.Model) {
			admMachine(m, "disk").Monitors = nil
			admComposition(m, "detailedPair").GetMembers()[1].Replaces = ""
			both := admScenario(m, "detailedPair", "bothPut")
			m.Scenarios = append(m.Scenarios, &umpirespb.Scenario{Machine: "detailedPair", Name: "any", Position: both.GetPosition(),
				Start: both.GetStart(), Free: true})
			limits := proto.Clone(admQuery(m, "bothPut").GetLimits()).(*umpirespb.Limits)
			limits.Steps = steps
			m.Queries = append(m.Queries, &umpirespb.Query{Name: "frontStaysHeld", Position: both.GetPosition(), Form: umpirespb.Query_FORM_VERIFY,
				Property: &umpirespb.ClaimRef{Machine: "detailedPair", Name: "frontHeld"},
				Scenario: &umpirespb.ClaimRef{Machine: "detailedPair", Name: "any"}, Limits: limits})
		}))
	}
	// Two steps: both put, then the search reads the pairs of a held store and a staged disk.
	r := composed(2)
	explored := receiptOf(t, r, "query detailedPair frontStaysHeld")
	require.Equal(t, Incomplete, explored.Kind)
	require.Equal(t, []HoleReach{{Edge: RowHole, ID: crashHole, Name: "crashUnmodeled", Row: "held_staged-back_crash", Depth: 1}}, holes(explored))
	require.Equal(t, []string{"putBoth"}, taken(explored.Holes[0].Prefix))
	// The pinned find ends where it is found, and reads no pair of that state.
	found := receiptOf(t, r, "query detailedPair bothPut")
	require.Equal(t, Found, found.Kind)
	require.Empty(t, found.Holes)
	require.NotContains(t, kinds(r), "composition detailedPair", "nothing is replaced, so nothing of the composition is checked")

	// One step: the state with the unknown pair is reached, and no pair of it is read.
	bounded := receiptOf(t, composed(1), "query detailedPair frontStaysHeld")
	require.Equal(t, Verified, bounded.Kind)
	require.Empty(t, bounded.Holes)
}

// ---- Compositions ---------------------------------------------------------------------------------

func composedTable(t *testing.T, m *umpirespb.Model, name string) *Table {
	t.Helper()
	b := bind(m, DefaultScope)
	s := b.subject(name)
	require.NoError(t, s.err)
	return s.table
}

func TestComposedStartsAreTheProductOfEveryMemberStart(t *testing.T) {
	require.Equal(t, []string{"nothing_nothing"}, composedTable(t, lifted(t, "declarations"), "pair").Starts)
	detailed := composedTable(t, mutated(t, "declarations", noCrash), "detailedPair")
	require.Equal(t, []string{"nothing_empty"}, detailed.Starts)
	// The disk's reading of its state as the store's is no field of the composed state.
	require.Equal(t, []string{"front", "back_stage"}, detailed.StateFields)

	// With two starts of the store, the last member varies fastest.
	pair := composedTable(t, mutated(t, "declarations", storeAlsoStartsHeld), "pair")
	require.Equal(t, []string{"nothing_nothing", "nothing_held", "held_nothing", "held_held"}, pair.Starts)
	require.Equal(t, []string{"held_held", "held_nothing", "nothing_held", "nothing_nothing"}, pair.States)
	require.Equal(t, []string{"held_held", "nothing_nothing"}, pair.Ends, "the composition's ends, read over the composed state")
	require.Equal(t, []string{"putBoth"}, pair.Actions)
}

// The detailed disk starts only empty, which reads as the store holding nothing: it refines the store,
// and does not stand in for a store that may also start holding.
func TestAReplacementCoversEveryStartOfWhatItReplaces(t *testing.T) {
	r := checked(t, mutated(t, "declarations", noCrash, storeAlsoStartsHeld))
	require.Equal(t, Verified, receiptOf(t, r, "refinement disk store").Kind)
	replaced := receiptOf(t, r, "composition detailedPair")
	require.Equal(t, []any{RefinementRejected, umpire.RefinementInitial}, []any{replaced.Kind, replaced.Failure})
	require.Nil(t, replaced.Witness)
	require.Equal(t, "held", replaced.ProductWitness.Initial.Value)
}

// storeOpaque is made fair for put, and the sync dropped, so each member's put is a class of its own.
// The disk declares storeOpaque itself, so it keeps it where it replaces a store that names it too:
// the composition carries it for the front store's put and for the disk's.
func TestAReplacingMemberKeepsEveryAssumptionItDeclares(t *testing.T) {
	m := mutated(t, "declarations", noCrash, func(m *umpirespb.Model) {
		for _, a := range m.GetAssumptions() {
			if a.GetName() == "storeOpaque" {
				a.Fair = []string{admDeclaredPkg + "put"}
			}
		}
		admMachine(m, "disk").Assumes = []string{admDeclaredPkg + "storeOpaque", admDeclaredPkg + "flushRuns"}
		admComposition(m, "detailedPair").Syncs = nil
		m.Scenarios = slicesDelete(m.GetScenarios(), func(s *umpirespb.Scenario) bool { return s.GetMachine() == "detailedPair" })
		m.Queries = slicesDelete(m.GetQueries(), func(q *umpirespb.Query) bool { return q.GetName() == "bothPut" })
	})
	want := []Assumption{{Name: "storeOpaque", Fair: []string{"front_put", "back_put"}}, {Name: "flushEventuallyRuns", Fair: []string{"back_flush"}}}
	require.Equal(t, want, composedTable(t, m, "detailedPair").Assumptions)
	require.Equal(t, []string{"storeOpaque", "flushEventuallyRuns"}, receiptOf(t, checked(t, m), "composition detailedPair").Assumptions)

	// A member that replaces nothing keeps every assumption of its own.
	admComposition(m, "detailedPair").GetMembers()[1].Replaces = ""
	require.Equal(t, want, composedTable(t, m, "detailedPair").Assumptions)

	// The replaced store's own storeOpaque is not the disk's: a disk that does not declare it brings
	// none, and the composition's is the front store's alone.
	admComposition(m, "detailedPair").GetMembers()[1].Replaces = "store"
	admMachine(m, "disk").Assumes = []string{admDeclaredPkg + "flushRuns"}
	want[0].Fair = []string{"front_put"}
	require.Equal(t, want, composedTable(t, m, "detailedPair").Assumptions)
}

// The counter's composition with its members listed right before left, against the state record's
// left before right: a composed key follows the members, and the state a claim reads the record.
func TestAComposedStateIsKeyedInMemberOrderAndReadInFieldOrder(t *testing.T) {
	m := counter{k: 2}.model()
	m.GetCompositions()[0].Members = reversed(m.GetCompositions()[0].GetMembers())
	b := bind(m, DefaultScope)
	two := b.subject("two")
	require.NoError(t, two.err)
	// The right counter at 0 and the left at 1.
	state, err := two.state("0_1")
	require.NoError(t, err)
	require.Equal(t, "1-0", state.Key(), "the record holds left, then right")
	key, err := two.key(state)
	require.NoError(t, err)
	require.Equal(t, "0_1", key)
}

// ---- Refinement -----------------------------------------------------------------------------------

func TestRefinementControls(t *testing.T) {
	for name, c := range map[string]struct {
		mutate  []func(m *umpirespb.Model)
		kind    ReceiptKind
		failure RefinementFailure
		witness []string
		holes   []HoleReach
	}{
		"an invisible provider step is a stutter": {mutate: []func(m *umpirespb.Model){noCrash}, kind: Verified},
		"a start that reads as no start of the store": {mutate: []func(m *umpirespb.Model){noCrash, diskStarts("staged")},
			kind: RefinementRejected, failure: umpire.RefinementInitial},
		"a stutter that records a fact the store sees": {
			mutate: []func(m *umpirespb.Model){noCrash, returning("disk.visible", boolValue(true))},
			kind:   RefinementRejected, failure: umpire.RefinementVisibleStutter, witness: []string{"put", "flush"}},
		"a stutter whose outcome the store sees": {
			mutate: []func(m *umpirespb.Model){noCrash, returning("disk.visibleOutcomes", boolValue(true))},
			kind:   RefinementRejected, failure: umpire.RefinementVisibleStutter, witness: []string{"put", "flush"}},
		"a reachable hole": {kind: Incomplete, witness: []string{"put"},
			holes: []HoleReach{{Edge: RowHole, ID: crashHole, Name: "crashUnmodeled", Row: "staged-crash"}}},
		"a reachable hole beside a rejection": {mutate: []func(m *umpirespb.Model){returning("disk.visible", boolValue(true))},
			kind: RefinementRejected, failure: umpire.RefinementVisibleStutter, witness: []string{"put", "flush"}},
	} {
		t.Run(name, func(t *testing.T) {
			r := checked(t, mutated(t, "declarations", c.mutate...))
			for _, key := range []string{"refinement disk store", "composition detailedPair"} {
				got := receiptOf(t, r, key)
				require.Equal(t, []any{c.kind, c.failure}, []any{got.Kind, got.Failure}, key)
				require.Equal(t, c.holes, holes(got), key)
				if c.witness != nil {
					require.Equal(t, c.witness, taken(got.Witness), key)
				}
			}
			require.Equal(t, ClaimKey{Family: declared, Owner: "disk", Name: "store"}, receiptOf(t, r, "refinement disk store").Key)
		})
	}
}

// RefineTables asks what the refined machine sees through predicates that return no error, so one the
// Model's function fails on is kept and fails the refinement. Here visible is no Boolean for staged,
// which no step of the disk records once its flush is unbound.
func TestAVisibleFunctionThatCannotBeReadFailsTheRefinement(t *testing.T) {
	m := mutated(t, "declarations", noCrash, func(m *umpirespb.Model) {
		disk := admMachine(m, "disk")
		disk.Steps = slicesDelete(disk.GetSteps(), func(b *umpirespb.StepBinding) bool { return strings.HasSuffix(b.GetAction(), ".flush") })
		visible := functionNamed(m, "disk.visible")
		visible.Body = &umpirespb.Expr{Position: visible.GetBody().GetPosition(), Kind: &umpirespb.Expr_If{If: &umpirespb.If{
			Condition: visible.GetBody(), Then: admLiteral(visible.GetBody(), boolValue(true)), Else: admLiteral(visible.GetBody(), admIntValue(3))}}}
	})
	b := bind(m, DefaultScope)
	disk, store := b.subject("disk"), b.subject("store")
	spec, failed := b.reading(disk, store)
	require.True(t, spec.SeesFact("stored"))
	_, malformed := failed()
	require.NoError(t, malformed)
	require.False(t, spec.SeesFact("staged"))
	_, malformed = failed()
	require.ErrorContains(t, malformed, "disk: disk.visible is 3 for staged, not a Boolean")
	_, _, err := b.refines(disk, store, spec, failed)
	require.ErrorContains(t, err, "disk: disk.visible is 3 for staged, not a Boolean")
}

// A receipt counts the work its check did. A refinement that holds, or that only reachable holes leave
// unknown, read every row; one rejected at a start read none, and the generic check does not say how
// many rows it read before a row it rejects.
func TestARefinementReceiptCountsOnlyTheRowsItRead(t *testing.T) {
	for name, c := range map[string]struct {
		mutate   []func(m *umpirespb.Model)
		kind     ReceiptKind
		explored int
	}{
		"held":                {[]func(m *umpirespb.Model){noCrash}, Verified, 2},
		"left unknown":        {nil, Incomplete, 2},
		"rejected at a start": {[]func(m *umpirespb.Model){noCrash, diskStarts("staged")}, RefinementRejected, 0},
		"rejected at a row":   {[]func(m *umpirespb.Model){noCrash, returning("disk.visible", boolValue(true))}, RefinementRejected, 0},
	} {
		t.Run(name, func(t *testing.T) {
			got := receiptOf(t, checked(t, mutated(t, "declarations", c.mutate...)), "refinement disk store")
			require.Equal(t, []any{c.kind, c.explored, 2}, []any{got.Kind, got.Explored, got.TableRows})
		})
	}
}

// visibleUnknownForStored makes the disk's refinement unable to say whether the store sees stored,
// the fact its put records: the declared hole stands where the answer was.
func visibleUnknownForStored(m *umpirespb.Model) {
	visible := functionNamed(m, "disk.visible")
	at := visible.GetBody()
	visible.Body = &umpirespb.Expr{Position: at.GetPosition(), Kind: &umpirespb.Expr_If{If: &umpirespb.If{
		Condition: at,
		Then:      &umpirespb.Expr{Position: at.GetPosition(), Kind: &umpirespb.Expr_Hole{Hole: crashHole}},
		Else:      admLiteral(at, boolValue(false))}}}
}

// What a refinement names visible is unknown for the fact the disk's put records. A fact the store
// may see only narrows which steps carry a row, so a rejection found with the fact read as unseen
// stands whatever the hole hides; only a refinement found to hold that way is unknown.
func TestAVisibilityHoleErasesNoRejection(t *testing.T) {
	unread := []HoleReach{{Edge: DeclarationHole, ID: crashHole, Name: "crashUnmodeled"}}

	// The flush made to empty the disk, which reads as the store losing what it held: no step of the
	// store, on the row after the put.
	r := checked(t, mutated(t, "declarations", noCrash, visibleUnknownForStored, flushTo("empty")))
	for _, key := range []string{"refinement disk store", "composition detailedPair", "query disk putStoresThroughDisk"} {
		got := receiptOf(t, r, key)
		require.Equal(t, []any{RefinementRejected, umpire.RefinementUnmatched}, []any{got.Kind, got.Failure}, key)
		require.Equal(t, []string{"put", "flush"}, taken(got.Witness), key)
		require.Equal(t, unread, holes(got), key)
	}

	// With no such row every row refines with the fact read as unseen, which the hole leaves unknown.
	r = checked(t, mutated(t, "declarations", noCrash, visibleUnknownForStored))
	for _, key := range []string{"refinement disk store", "composition detailedPair", "query disk putStoresThroughDisk"} {
		got := receiptOf(t, r, key)
		require.Equal(t, Incomplete, got.Kind, key)
		require.Equal(t, unread, holes(got), key)
	}
	// The generic check read both rows of the disk before it accepted them.
	for _, key := range []string{"refinement disk store", "composition detailedPair"} {
		got := receiptOf(t, r, key)
		require.Equal(t, []any{2, 2, "fixture.declarations.target.disk"}, []any{got.Explored, got.TableRows, got.Target}, key)
	}
	// The Query was searched all the same: its Property is read through the map, which the hole is not in.
	through := receiptOf(t, r, "query disk putStoresThroughDisk")
	require.Equal(t, []int{3, 2}, []int{through.Explored, through.Expanded})
	r = checked(t, mutated(t, "declarations", noCrash, visibleUnknownForStored, putRecordsStaged))
	through = receiptOf(t, r, "query disk putStoresThroughDisk")
	require.Equal(t, []any{Counterexample, []string{"empty-put"}}, []any{through.Kind, through.Rows})
	require.Equal(t, unread, holes(through))
}

const secondHole = "generic.second"

// choosing makes a function's body one value where its condition holds and another elsewhere, where a
// value is a hole's id, to reach that hole, or a literal.
func choosing(condition *umpirespb.Expr, then, otherwise any) *umpirespb.Expr {
	value := func(v any) *umpirespb.Expr {
		if id, ok := v.(string); ok {
			return &umpirespb.Expr{Position: at(1), Kind: &umpirespb.Expr_Hole{Hole: id}}
		}
		return expr(v)
	}
	return expr(&umpirespb.If{Condition: condition, Then: value(then), Else: value(otherwise)})
}

func declaringSecondHole(m *umpirespb.Model) {
	m.Holes = append(m.Holes, &umpirespb.Hole{Id: secondHole, Name: "second", Position: at(1)})
}

func stageIs(variable, c string) *umpirespb.Expr {
	return binary(umpirespb.Binary_OP_EQ, field(expr(variable), "stage"), expr(stage(c)))
}

// A declaration read at several values, a hole at the first and something else at a later one: what
// was read after the hole is never lost behind it. A value that is no Boolean is an error of the
// Model wherever it is read, and every hole read is listed.
func TestAnEarlierHoleMasksNothingReadAfterIt(t *testing.T) {
	three := admIntValue(3)
	first := HoleReach{Edge: DeclarationHole, ID: crashHole, Name: "crashUnmodeled"}
	second := HoleReach{Edge: DeclarationHole, ID: secondHole, Name: "second"}
	frontHeld := func() *umpirespb.Expr {
		return binary(umpirespb.Binary_OP_EQ, field(field(expr("p"), "front"), "kept"), expr(admEnum("fixture.declarations.Kept", "held")))
	}
	for name, c := range map[string]struct {
		// declare gives the declaration its body: the hole where the condition holds, and v elsewhere.
		declare  func(m *umpirespb.Model, v any)
		receipts []string
		message  string
	}{
		// The put's stored, then the flush's staged.
		"what a refinement names visible": {func(m *umpirespb.Model, v any) {
			visible := functionNamed(m, "disk.visible")
			visible.Body = choosing(visible.GetBody(), crashHole, v)
		}, []string{"refinement disk store", "composition detailedPair", "query disk putStoresThroughDisk"},
			"disk: disk.visible is 3 for staged, not a Boolean"},
		// The empty disk, then the staged one.
		"a machine's ends": {func(m *umpirespb.Model, v any) {
			admMachine(m, "disk").GetEnds().GetLambda().Body = choosing(stageIs("d", "empty"), crashHole, v)
		}, []string{"machine disk", "refinement disk store", "query disk durableStays", "progress disk durableEventually"},
			"disk: ends is 3 at staged, not a Boolean"},
		// Both stores holding, then neither.
		"a composition's ends": {func(m *umpirespb.Model, v any) {
			admComposition(m, "pair").GetEnds().GetLambda().Body = choosing(frontHeld(), crashHole, v)
		}, []string{"query pair keptTogether"}, "pair: ends is 3 at nothing_nothing, not a Boolean"},
	} {
		t.Run(name, func(t *testing.T) {
			r := checked(t, mutated(t, "declarations", noCrash, func(m *umpirespb.Model) { c.declare(m, three) }))
			for _, key := range c.receipts {
				got := receiptOf(t, r, key)
				require.Equal(t, DeclarationError, got.Kind, key)
				var located *Error
				require.ErrorAs(t, got.Cause, &located, key)
				require.Equal(t, c.message, located.Message, key)
			}
			r = checked(t, mutated(t, "declarations", noCrash, declaringSecondHole, func(m *umpirespb.Model) { c.declare(m, secondHole) }))
			for _, key := range c.receipts {
				got := receiptOf(t, r, key)
				require.Equal(t, Incomplete, got.Kind, key)
				require.Equal(t, []HoleReach{first, second}, holes(got), key)
			}
		})
	}
}

// The same for the functions a search and a progress check read, which the generic checker classes:
// a hole leaves a step or a state unknown and the check goes on, so a value that is no Boolean down
// another branch still fails it, and every hole it reads is listed.
func TestAnEarlierClaimHoleMasksNothingReadAfterIt(t *testing.T) {
	skipped := func() *umpirespb.Expr {
		return binary(umpirespb.Binary_OP_EQ, field(expr("after"), "outcome"), expr(caseOf("O", "skipped")))
	}
	counting := func(v any) *umpirespb.Model {
		m := counter{k: 2, skip: true}.model()
		m.Holes = []*umpirespb.Hole{{Id: crashHole, Name: "crashUnmodeled", Position: at(1)}, {Id: secondHole, Name: "second", Position: at(1)}}
		// A skip, the first row of the start, is where the hole is; a tick is read after it.
		functionNamed(m, "generic.always").Body = choosing(skipped(), crashHole, v)
		return m
	}
	got := receiptOf(t, checked(t, counting(admIntValue(3))), "query counter counter.all")
	require.Equal(t, DeclarationError, got.Kind)
	require.ErrorContains(t, got.Cause, "counter.always: generic.always is 3 for the step into 1, not a Boolean")
	got = receiptOf(t, checked(t, counting(secondHole)), "query counter counter.all")
	require.Equal(t, Incomplete, got.Kind)
	require.Equal(t, []HoleReach{{Edge: ClaimHole, ID: crashHole, Name: "crashUnmodeled", Row: "0-skip"},
		{Edge: ClaimHole, ID: secondHole, Name: "second", Row: "0-tick"}}, holes(got))

	// The disk whose crash takes an empty disk to a durable one, read before the staged disk a put reaches.
	progressing := func(v any) *umpirespb.Model {
		return mutated(t, "declarations", declaringSecondHole, returning("disk.progress.durableEventually.to", boolValue(false)), func(m *umpirespb.Model) {
			crash := function(m, "Declarations$package$.crashStep").GetBody().GetIf()
			crash.GetCondition().GetBinary().Right = admLiteral(crash.GetCondition(), stage("empty"))
			crash.Then = proto.Clone(function(m, "Declarations$package$.flushStep").GetBody().GetMatch().GetCases()[0].GetBody()).(*umpirespb.Expr)
			functionNamed(m, "disk.progress.durableEventually.from").Body = choosing(stageIs("d", "durable"), crashHole,
				choosing(stageIs("d", "staged"), v, boolValue(true)).GetIf())
		})
	}
	failed := receiptOf(t, checked(t, progressing(admIntValue(3))), "progress disk durableEventually")
	require.Equal(t, DeclarationError, failed.Kind)
	require.ErrorContains(t, failed.Cause, "disk.durableEventually: disk.progress.durableEventually.from is 3 at staged, not a Boolean")
	unread := receiptOf(t, checked(t, progressing(secondHole)), "progress disk durableEventually deadline")
	require.Equal(t, Incomplete, unread.Kind)
	require.Equal(t, []HoleReach{{Edge: ClaimHole, ID: crashHole, Name: "crashUnmodeled", Depth: 1},
		{Edge: ClaimHole, ID: secondHole, Name: "second", Depth: 1}}, holes(unread))
}

// withTape makes the detailed pair two detailed providers that each replace a store: the disk, whose
// refinement its crash hole leaves unknown, and a tape, which is the disk with no crash and no
// monitor and with whatever alter does to it.
func withTape(members [2]string, alter func(m *umpirespb.Model, tape *umpirespb.Machine)) func(m *umpirespb.Model) {
	return func(m *umpirespb.Model) {
		tape := proto.Clone(admMachine(m, "disk")).(*umpirespb.Machine)
		tape.Name, tape.Monitors = "tape", nil
		tape.Steps = slicesDelete(tape.GetSteps(), func(b *umpirespb.StepBinding) bool { return strings.HasSuffix(b.GetAction(), ".crash") })
		alter(m, tape)
		m.Machines = append(m.Machines, tape)

		admType(m, "fixture.declarations.DetailedPair").GetRecord().GetFields()[0].Type = named("fixture.declarations.Disk")
		detailed := admComposition(m, "detailedPair")
		detailed.Ends = nil
		detailed.Members = []*umpirespb.Member{{Field: "front", Machine: members[0], Replaces: "store"},
			{Field: "back", Machine: members[1], Replaces: "store"}}
		m.Properties = slicesDelete(m.GetProperties(), func(p *umpirespb.Property) bool { return p.GetMachine() == "detailedPair" })
		m.Scenarios = slicesDelete(m.GetScenarios(), func(s *umpirespb.Scenario) bool { return s.GetMachine() == "detailedPair" })
		m.Queries = slicesDelete(m.GetQueries(), func(q *umpirespb.Query) bool { return q.GetName() == "bothPut" })
	}
}

var memberOrders = map[string][2]string{"the disk first": {"disk", "tape"}, "the tape first": {"tape", "disk"}}

// The tape started staged reads as no start of the store. Its rejection is the composition's result,
// whichever member is first.
func TestAMembersHoleErasesNoRejectionOfAnotherMember(t *testing.T) {
	for name, members := range memberOrders {
		t.Run(name, func(t *testing.T) {
			r := checked(t, mutated(t, "declarations", withTape(members, func(_ *umpirespb.Model, tape *umpirespb.Machine) {
				start := tape.GetStarts()[0].GetConstruct()
				start.Args[0] = admLiteral(start.GetArgs()[0], stage("staged"))
			})))
			require.Equal(t, Incomplete, receiptOf(t, r, "refinement disk store").Kind)
			require.Equal(t, RefinementRejected, receiptOf(t, r, "refinement tape store").Kind)
			got := receiptOf(t, r, "composition detailedPair")
			require.Equal(t, []any{RefinementRejected, umpire.RefinementInitial, "fixture.declarations.target.tape"},
				[]any{got.Kind, got.Failure, got.Target})
		})
	}
}

// A member whose refinement a hole leaves unknown hides no other member's error either, and no other
// member's hole: a tape whose visible function is malformed is the composition's error, and a tape
// whose map is a second hole has its hole listed beside the disk's.
func TestAMembersHoleMasksNoOtherMembersErrorOrHole(t *testing.T) {
	fact := []*umpirespb.Param{{Name: "f", Type: named("fixture.declarations.Fact")}}
	state := []*umpirespb.Param{{Name: "d", Type: named("fixture.declarations.Disk")}}
	crash := HoleReach{Edge: RowHole, ID: crashHole, Name: "crashUnmodeled", Row: "staged-crash"}
	second := HoleReach{Edge: DeclarationHole, ID: secondHole, Name: "second"}
	for name, members := range memberOrders {
		t.Run(name, func(t *testing.T) {
			r := checked(t, mutated(t, "declarations", withTape(members, func(m *umpirespb.Model, tape *umpirespb.Machine) {
				m.Functions = append(m.Functions, &umpirespb.Function{Name: "tape.visible", Position: at(1), Params: fact, Body: expr(admIntValue(3))})
				tape.GetRefines().Visible = "tape.visible"
			})))
			got := receiptOf(t, r, "composition detailedPair")
			require.Equal(t, DeclarationError, got.Kind)
			require.ErrorContains(t, got.Cause, "tape: tape.visible is 3 for stored, not a Boolean")

			r = checked(t, mutated(t, "declarations", declaringSecondHole, withTape(members, func(m *umpirespb.Model, tape *umpirespb.Machine) {
				m.Functions = append(m.Functions, &umpirespb.Function{Name: "tape.stored", Position: at(1), Params: state,
					Body: &umpirespb.Expr{Position: at(1), Kind: &umpirespb.Expr_Hole{Hole: secondHole}}})
				tape.GetRefines().Map = "tape.stored"
			})))
			got = receiptOf(t, r, "composition detailedPair")
			require.Equal(t, Incomplete, got.Kind)
			want := []HoleReach{crash, second}
			if members[0] == "tape" {
				want = []HoleReach{second, crash}
			}
			require.Equal(t, want, holes(got))
		})
	}
}

// Both members' refinements are left unknown by a crash hole, each with the path to it as its witness.
// The replay Model gives the tape a put that does nothing, so only the tape's path does not replay:
// that is the composition's result, an error, whichever member the tape is.
func TestAnyMembersReplayFailureIsTheCompositions(t *testing.T) {
	holed := func(put func(m *umpirespb.Model, tape *umpirespb.Machine)) func(m *umpirespb.Model, tape *umpirespb.Machine) {
		return func(m *umpirespb.Model, tape *umpirespb.Machine) {
			tape.Steps = nil
			for _, b := range admMachine(m, "disk").GetSteps() {
				tape.Steps = append(tape.Steps, proto.Clone(b).(*umpirespb.StepBinding))
			}
			put(m, tape)
		}
	}
	state := []*umpirespb.Param{{Name: "d", Type: named("fixture.declarations.Disk")}}
	for name, members := range memberOrders {
		t.Run(name, func(t *testing.T) {
			m := mutated(t, "declarations", withTape(members, holed(func(*umpirespb.Model, *umpirespb.Machine) {})))
			noTapePut := mutated(t, "declarations", withTape(members, holed(func(m *umpirespb.Model, tape *umpirespb.Machine) {
				m.Functions = append(m.Functions, &umpirespb.Function{Name: "tape.putStep", Position: at(1), Params: state, Body: expr(&umpirespb.ListOf{})})
				tape.GetSteps()[0].Function = "tape.putStep"
			})))
			r := check(m, DefaultScope, noTapePut)
			require.Equal(t, Incomplete, receiptOf(t, r, "refinement disk store").Kind)
			require.Equal(t, ReplayFailed, receiptOf(t, r, "refinement tape store").Kind)
			got := receiptOf(t, r, "composition detailedPair")
			require.Equal(t, ReplayFailed, got.Kind)
			require.ErrorContains(t, got.Cause, "step 1 takes put, which is not enabled at 'empty'")
			require.Equal(t, "fixture.declarations.target.tape", got.Target)

			// Where both paths replay, the composition is incomplete by both holes, and the path of the
			// member that is not first is kept beside the first's.
			both := receiptOf(t, Check(m, DefaultScope), "composition detailedPair")
			require.Equal(t, Incomplete, both.Kind)
			require.Len(t, both.Holes, 1, "the two members reach the one hole at the one row")
			require.Len(t, both.Also, 1)
			require.NotSame(t, both.Witness, both.Also[0].Witness)
		})
	}
}

// The one rule by which several results become one receipt, over every ordered pair of kinds: the
// kind of the higher precedence gives the receipt, the first of two of one precedence, and every
// hole and every witness of both is kept.
func TestFoldKeepsTheKindOfHighestPrecedenceAndEveryHoleAndWitness(t *testing.T) {
	precedence := [][]ReceiptKind{
		{ReplayFailed, AdmissionError, DeclarationError},
		{RefinementRejected, Counterexample},
		{ResourceLimit, LimitReached, Unresolved},
		// A find's witness is realized: a hole another result read takes nothing from it.
		{Found},
		{Incomplete},
		{Unsupported},
		{Verified, NotFound},
	}
	rank := map[ReceiptKind]int{}
	var kinds []ReceiptKind
	for i, level := range precedence {
		for _, k := range level {
			rank[k] = i
			kinds = append(kinds, k)
		}
	}
	require.Len(t, kinds, 13)
	part := func(k ReceiptKind, n string) Receipt {
		return Receipt{Kind: k, Explanation: n, Key: ClaimKey{Name: n}, Holes: []HoleReach{{Edge: RowHole, Name: n, Row: n}},
			Witness: &Trace{Initial: Atom{Value: n}}}
	}
	for _, first := range kinds {
		for _, second := range kinds {
			x, y := part(first, "x"), part(second, "y")
			got := fold(x, y)
			winner, loser := x, y
			if rank[second] < rank[first] {
				winner, loser = y, x
			}
			want := winner
			want.Holes = []HoleReach{x.Holes[0], y.Holes[0]}
			want.Also = []Receipt{loser}
			// A claim found to hold, or found nowhere, on what was read is not established once a hole was read.
			if winner.Kind == Verified || winner.Kind == NotFound {
				want.Kind = Incomplete
			}
			if want.Kind == Incomplete {
				want.Explanation = "the result is incomplete: the check read the hole x at the row 'x', and the hole y at the row 'y'"
			}
			require.Equal(t, want, got, "%s then %s", first, second)
		}
	}

	// A found witness beside a result a hole leaves incomplete, in either order: the witness stands,
	// the hole is listed, and the incomplete result's own path is kept.
	found, unknown := part(Found, "found"), part(Incomplete, "unknown")
	for _, parts := range [][]Receipt{{found, unknown}, {unknown, found}} {
		got := fold(parts...)
		require.Equal(t, []any{Found, found.Witness, []Receipt{unknown}}, []any{got.Kind, got.Witness, got.Also})
		require.ElementsMatch(t, []HoleReach{found.Holes[0], unknown.Holes[0]}, got.Holes)
	}

	// One result is itself, and a result that read no hole keeps its kind and its words.
	alone := Receipt{Kind: Verified, Explanation: "as it was", Explored: 3}
	require.Equal(t, alone, fold(alone))
	// A part that only qualifies another shares its witness, which is then kept once.
	witness := &Trace{Initial: Atom{Value: "w"}}
	failed := fold(Receipt{Kind: Counterexample, Witness: witness}, Receipt{Kind: ReplayFailed, Witness: witness, Explanation: "did not replay"})
	require.Equal(t, Receipt{Kind: ReplayFailed, Witness: witness, Explanation: "did not replay"}, failed)
	// Three results: the last is of the highest precedence, and the holes of all three are kept in order.
	three := fold(part(Incomplete, "x"), part(ResourceLimit, "y"), part(DeclarationError, "z"))
	require.Equal(t, []any{DeclarationError, "z", 3, 2}, []any{three.Kind, three.Key.Name, len(three.Holes), len(three.Also)})
}

// A machine's start, its ends and its evidence are one reading of the machine: a hole in one hides
// neither a hole nor an error of the Model in another.
func TestAMachinesDeclarationsAreReadPastAHole(t *testing.T) {
	hole := func(id string) *umpirespb.Expr {
		return &umpirespb.Expr{Position: at(1), Kind: &umpirespb.Expr_Hole{Hole: id}}
	}
	unknownStart := func(m *umpirespb.Model) { admMachine(m, "disk").Starts[0] = hole(crashHole) }
	first := HoleReach{Edge: DeclarationHole, ID: crashHole, Name: "crashUnmodeled"}
	second := HoleReach{Edge: DeclarationHole, ID: secondHole, Name: "second"}
	for name, c := range map[string]struct {
		mutate func(m *umpirespb.Model)
		kind   ReceiptKind
		holes  []HoleReach
		cause  string
	}{
		"a start hole and the whole ends a hole": {func(m *umpirespb.Model) { admMachine(m, "disk").Ends = hole(secondHole) },
			Incomplete, []HoleReach{first, second}, ""},
		"a start hole and an ends hole at a state": {func(m *umpirespb.Model) {
			admMachine(m, "disk").GetEnds().GetLambda().Body = choosing(stageIs("d", "staged"), secondHole, boolValue(true))
		}, Incomplete, []HoleReach{first, second}, ""},
		"a start hole and an evidence hole": {func(m *umpirespb.Model) {
			evidence := functionNamed(m, admMachine(m, "disk").GetEvidence())
			evidence.Body = hole(secondHole)
		}, Incomplete, []HoleReach{first, second}, ""},
		"a start hole and ends that is no function": {func(m *umpirespb.Model) {
			admMachine(m, "disk").Ends = expr(boolValue(true))
		}, DeclarationError, nil, "not a function"},
		"a start hole and ends that is no Boolean": {func(m *umpirespb.Model) {
			ends := admMachine(m, "disk").GetEnds().GetLambda()
			ends.Body = admLiteral(ends.GetBody(), admIntValue(3))
		}, DeclarationError, nil, "disk: ends is 3 at empty, not a Boolean"},
	} {
		t.Run(name, func(t *testing.T) {
			r := checked(t, mutated(t, "declarations", noCrash, declaringSecondHole, unknownStart, c.mutate))
			got := receiptOf(t, r, "machine disk")
			require.Equal(t, c.kind, got.Kind)
			require.Equal(t, c.holes, holes(got))
			if c.cause != "" {
				require.ErrorContains(t, got.Cause, c.cause)
			}
		})
	}
}

// What a refinement names visible and its map are read in one generic check: a hole in the one hides
// neither a hole nor an error of the Model in the other.
func TestARefinementsMapIsReadPastAVisibilityHole(t *testing.T) {
	stored := admDeclaredPkg + "stored"
	atDurable := func(v any) func(m *umpirespb.Model) {
		return func(m *umpirespb.Model) {
			f := functionNamed(m, stored)
			var then *umpirespb.Expr
			if id, ok := v.(string); ok {
				then = &umpirespb.Expr{Position: at(1), Kind: &umpirespb.Expr_Hole{Hole: id}}
			} else {
				then = expr("d")
			}
			f.Body = expr(&umpirespb.If{Condition: stageIs("d", "durable"), Then: then, Else: f.GetBody()})
		}
	}
	// The put's stored is read on the first row, and the durable disk the flush reaches on the second.
	r := checked(t, mutated(t, "declarations", noCrash, declaringSecondHole, visibleUnknownForStored, atDurable(secondHole)))
	got := receiptOf(t, r, "refinement disk store")
	require.Equal(t, Incomplete, got.Kind)
	require.Equal(t, []HoleReach{{Edge: DeclarationHole, ID: crashHole, Name: "crashUnmodeled"},
		{Edge: DeclarationHole, ID: secondHole, Name: "second"}}, holes(got))

	// The map made to read a durable disk as itself, which is no state of the store.
	r = checked(t, mutated(t, "declarations", noCrash, visibleUnknownForStored, atDurable(nil)))
	got = receiptOf(t, r, "refinement disk store")
	require.Equal(t, DeclarationError, got.Kind)
	require.ErrorContains(t, got.Cause, "disk: "+stored+" reads durable as durable, which is no state of store")
}

// ---- Progress, assumptions and bounds --------------------------------------------------------------

func progressParts(t *testing.T, r *Report) map[ProgressKind]ReceiptKind {
	t.Helper()
	out := map[ProgressKind]ReceiptKind{}
	for _, part := range []ProgressKind{umpire.DeadlockKind, umpire.CycleKind, umpire.DeadlineKind} {
		out[part] = receiptOf(t, r, "progress disk durableEventually "+string(part)).Kind
	}
	return out
}

// to made the empty stage, which nothing after a put reaches.
var neverThere = func(m *umpirespb.Model) {
	to := functionNamed(m, "disk.progress.durableEventually.to").GetBody().GetBinary()
	to.Right = admLiteral(to.GetRight(), stage("empty"))
}

func within(n int32) func(m *umpirespb.Model) {
	return func(m *umpirespb.Model) { m.GetProgress()[0].Within = n }
}

func TestProgressViolationsAreReportedApart(t *testing.T) {
	type parts = map[ProgressKind]ReceiptKind
	for name, c := range map[string]struct {
		mutate  []func(m *umpirespb.Model)
		want    parts
		witness map[ProgressKind][]string
		loop    int
	}{
		// A flushed disk has no step, and is not where the claim leads.
		"deadlock": {[]func(m *umpirespb.Model){noCrash, neverThere},
			parts{umpire.DeadlockKind: Counterexample, umpire.CycleKind: Verified, umpire.DeadlineKind: Verified},
			map[ProgressKind][]string{umpire.DeadlockKind: {"put", "flush"}}, -1},
		// The one step from staged is already the whole deadline.
		"deadline": {[]func(m *umpirespb.Model){noCrash, neverThere, within(1)},
			parts{umpire.DeadlockKind: Counterexample, umpire.CycleKind: Verified, umpire.DeadlineKind: Counterexample},
			map[ProgressKind][]string{umpire.DeadlineKind: {"put", "flush"}}, -1},
		// A flush that leaves the disk staged is taken forever, as its fairness asks: once to return to
		// the staged disk, and once as the fair class the cycle must take.
		"fair cycle": {[]func(m *umpirespb.Model){noCrash, flushTo("staged")},
			parts{umpire.DeadlockKind: Verified, umpire.CycleKind: Counterexample, umpire.DeadlineKind: Counterexample},
			map[ProgressKind][]string{umpire.CycleKind: {"put", "flush", "flush"}}, 1},
		// The same cycle beside the crash hole: the cycle takes the only fair class, so it stands, and
		// what the hole may hide leaves the absence of a deadlock unknown.
		"a cycle beside a hole": {[]func(m *umpirespb.Model){flushTo("staged")},
			parts{umpire.DeadlockKind: Incomplete, umpire.CycleKind: Counterexample, umpire.DeadlineKind: Counterexample},
			map[ProgressKind][]string{umpire.CycleKind: {"put", "flush", "flush"}}, 1},
		// A staged disk whose only pair is the crash hole is not deadlocked: it may have a step.
		"a state whose only pair is a hole": {[]func(m *umpirespb.Model){func(m *umpirespb.Model) {
			disk := admMachine(m, "disk")
			disk.Steps = slicesDelete(disk.GetSteps(), func(b *umpirespb.StepBinding) bool { return strings.HasSuffix(b.GetAction(), ".flush") })
			m.GetProgress()[0].Assumptions = nil
			m.Scenarios = slicesDelete(m.GetScenarios(), func(s *umpirespb.Scenario) bool { return s.GetName() == "putThenFlush" })
			m.Queries = slicesDelete(m.GetQueries(), func(q *umpirespb.Query) bool { return q.GetScenario().GetName() == "putThenFlush" })
		}}, parts{umpire.DeadlockKind: Incomplete, umpire.CycleKind: Incomplete, umpire.DeadlineKind: Incomplete}, nil, -1},
	} {
		t.Run(name, func(t *testing.T) {
			r := checked(t, mutated(t, "declarations", c.mutate...))
			require.Equal(t, c.want, progressParts(t, r))
			for part, want := range c.witness {
				got := receiptOf(t, r, "progress disk durableEventually "+string(part))
				require.Equal(t, want, taken(got.Witness))
				require.Equal(t, c.loop, got.Loop)
			}
		})
	}
}

func TestProgressBoundsAreNotVerdicts(t *testing.T) {
	m := mutated(t, "declarations", noCrash)
	all := func(kind ReceiptKind) map[ProgressKind]ReceiptKind {
		return map[ProgressKind]ReceiptKind{umpire.DeadlockKind: kind, umpire.CycleKind: kind, umpire.DeadlineKind: kind}
	}
	// One step from the start leaves the staged disk's flush unexplored.
	scope := DefaultScope
	scope.Progress = Limits{Name: "one step", Steps: 1, Search: 1 << 10}
	r := Check(m, scope)
	require.Equal(t, all(Unresolved), progressParts(t, r))
	require.Equal(t, scope.Progress, receiptOf(t, r, "progress disk durableEventually deadline").Limits)

	// Two units of work explore empty and staged, and no more.
	scope.Progress = Limits{Name: "two units", Steps: 1 << 10, Search: 2}
	r = Check(m, scope)
	require.Equal(t, all(LimitReached), progressParts(t, r))
	require.Equal(t, 2, receiptOf(t, r, "progress disk durableEventually deadlock").Explored)
}

func TestReceiptsNameTheAssumptionsTheyRelyOn(t *testing.T) {
	r := checked(t, mutated(t, "declarations", noCrash))
	for key, want := range map[string][]string{
		"query store putStores":                    {"storeOpaque"},
		"query pair keptTogether":                  {"storeOpaque"},
		"query disk putAccepted":                   nil,
		"refinement disk store":                    nil,
		"progress disk durableEventually deadline": {"flushEventuallyRuns"},
	} {
		require.Equal(t, want, receiptOf(t, r, key).Assumptions, key)
	}
}

// ---- Holes, disabled pairs, malformed Models and limits --------------------------------------------

func TestAHoleIsIncompleteEvidenceAndNeverDisabled(t *testing.T) {
	crash := []HoleReach{{Edge: RowHole, ID: crashHole, Name: "crashUnmodeled", Row: "staged-crash", Depth: 1}}

	r := checked(t, lifted(t, "declarations"))
	require.Equal(t, map[string]ReceiptKind{
		"refinement disk store":                      Incomplete,
		"composition detailedPair":                   Incomplete,
		"query detailedPair bothPut":                 Unsupported,
		"query disk durableStays":                    Incomplete,
		"query disk putAccepted":                     Verified,
		"query store putStores":                      Found,
		"query disk putStoresThroughDisk":            Incomplete,
		"query pair keptTogether":                    Verified,
		"progress disk durableEventually deadlock":   Incomplete,
		"progress disk durableEventually fair-cycle": Incomplete,
		"progress disk durableEventually deadline":   Incomplete,
	}, kinds(r))

	// The free Scenario explores the staged disk, where a crash is unknown; a crash of an empty or a
	// durable disk is disabled, and is no hole.
	explored := receiptOf(t, r, "query disk durableStays")
	require.Equal(t, crash, holes(explored))
	require.Equal(t, []string{"put"}, taken(explored.Holes[0].Prefix))
	require.Contains(t, explored.Explanation, "the hole crashUnmodeled at the row 'staged-crash'")
	require.Nil(t, explored.Witness)

	// The table the search read holds the pair as unknown: no row, and not one of the disabled pairs.
	b := bind(lifted(t, "declarations"), DefaultScope)
	disk := b.subject("disk").table
	require.Len(t, disk.Unknown, 1)
	require.Equal(t, []string{"staged-crash", "staged", "crash"}, []string{disk.Unknown[0].Row, disk.Unknown[0].Source, disk.Unknown[0].Action})
	require.JSONEq(t, `[{"key":"empty-put","source":"empty","action":"put","results":[{"outcome":"accepted","state":"staged","facts":["stored"]}]},`+
		`{"key":"staged-flush","source":"staged","action":"flush","results":[{"outcome":"deferred","state":"durable","facts":["staged"]}]}]`,
		rowsJSON(t, disk))
	unknownFrom := func(state string) []UnknownPair {
		return slices.DeleteFunc(slices.Clone(disk.Unknown), func(u UnknownPair) bool { return u.Source != state })
	}
	require.Empty(t, unknownFrom("empty"))
	require.Empty(t, unknownFrom("durable"))

	// The pinned Scenario schedules a flush of the staged disk, never a crash: the hole is unexplored.
	require.Empty(t, receiptOf(t, r, "query disk putAccepted").Holes)

	// The Query reads the store's Property through a refinement the reachable hole leaves unknown.
	through := receiptOf(t, r, "query disk putStoresThroughDisk")
	require.Equal(t, []HoleReach{{Edge: RowHole, ID: crashHole, Name: "crashUnmodeled", Row: "staged-crash"}}, holes(through))

	// A violation read through that refinement stands all the same, with the holes beside it.
	violatedThrough := receiptOf(t, checked(t, mutated(t, "declarations", putRecordsStaged)), "query disk putStoresThroughDisk")
	require.Equal(t, []any{Counterexample, []string{"empty-put"}}, []any{violatedThrough.Kind, violatedThrough.Rows})
	require.Equal(t, holes(through), holes(violatedThrough))

	deadline := receiptOf(t, r, "progress disk durableEventually deadline")
	require.Equal(t, crash, holes(deadline))

	// One step explores no successor of the staged disk: the hole is past the bound.
	r = checked(t, mutated(t, "declarations", func(m *umpirespb.Model) { admQuery(m, "durableStays").GetLimits().Steps = 1 }))
	bounded := receiptOf(t, r, "query disk durableStays")
	require.Equal(t, Verified, bounded.Kind)
	require.Empty(t, bounded.Holes)

	// A violation on the first step stands, with the hole the search went on to explore beside it.
	r = checked(t, mutated(t, "declarations", returning("disk.property.durableStays", boolValue(false))))
	violated := receiptOf(t, r, "query disk durableStays")
	require.Equal(t, Counterexample, violated.Kind)
	require.Equal(t, []string{"empty-put"}, violated.Rows)
	require.Equal(t, crash, holes(violated))
}

// A hole inside a Property's function is a step the claim could not be read on: unknown, not false.
func TestAHoleInsideAClaimFunctionIsUnknownEvidence(t *testing.T) {
	reaches := func(function string) func(m *umpirespb.Model) {
		return func(m *umpirespb.Model) {
			f := functionNamed(m, function)
			f.Body = &umpirespb.Expr{Position: f.GetBody().GetPosition(), Kind: &umpirespb.Expr_Hole{Hole: crashHole}}
		}
	}
	claim := func(row string) []HoleReach {
		return []HoleReach{{Edge: ClaimHole, ID: crashHole, Name: "crashUnmodeled", Row: row}}
	}
	for name, c := range map[string]struct {
		function string
		receipt  string
		want     []HoleReach
	}{
		"a Property":          {"disk.property.durableStays", "query disk durableStays", claim("empty-put")},
		"a monitor's next":    {admDeclaredPkg + "countStored", "query disk durableStays", claim("empty-put")},
		"a monitor's verdict": {admDeclaredPkg + "storedOnce.violated", "query disk durableStays", claim("empty-put")},
		"a monitor's point":   {admDeclaredPkg + "stagedBeforeDurable.after", "query disk durableStays", claim("empty-put")},
		// The claim cannot be read at the start, so nothing past it is: each kind of violation is ruled
		// out by nothing, and says so apart.
		"a progress claim's from": {"disk.progress.durableEventually.from", "progress disk durableEventually deadline",
			[]HoleReach{{Edge: ClaimHole, ID: crashHole, Name: "crashUnmodeled"}}},
		"a composition's Property": {"pair.property.keptTogether", "query pair keptTogether", claim("nothing_nothing-putBoth")},
	} {
		t.Run(name, func(t *testing.T) {
			got := receiptOf(t, checked(t, mutated(t, "declarations", noCrash, reaches(c.function))), c.receipt)
			require.Equal(t, Incomplete, got.Kind)
			require.Equal(t, c.want, holes(got))
		})
	}

}

// A hole reached outside any step leaves one machine's table unknown, or one machine's refinement:
// what depends on it is incomplete, and everything else is checked as before.
func TestADeclarationHoleStaysWithWhatDependsOnIt(t *testing.T) {
	outside := []HoleReach{{Edge: DeclarationHole, ID: crashHole, Name: "crashUnmodeled"}}
	hole := func(at *umpirespb.Expr) *umpirespb.Expr {
		return &umpirespb.Expr{Position: at.GetPosition(), Kind: &umpirespb.Expr_Hole{Hole: crashHole}}
	}

	// The disk's ends: the disk has no table. The store, and the pair of two stores, do not read it.
	r := checked(t, mutated(t, "declarations", noCrash, func(m *umpirespb.Model) {
		ends := admMachine(m, "disk").GetEnds().GetLambda()
		ends.Body = hole(ends.GetBody())
	}))
	require.Equal(t, map[string]ReceiptKind{
		"machine disk":                    Incomplete,
		"refinement disk store":           Incomplete,
		"composition detailedPair":        Incomplete,
		"query detailedPair bothPut":      Unsupported,
		"query disk durableStays":         Incomplete,
		"query disk putAccepted":          Incomplete,
		"query disk putStoresThroughDisk": Incomplete,
		"progress disk durableEventually": Incomplete,
		"query store putStores":           Found,
		"query pair keptTogether":         Verified,
	}, kinds(r))
	for _, x := range r.Receipts {
		if x.Kind == Incomplete {
			require.Equal(t, outside, holes(x), receiptKey(x))
			require.NotEmpty(t, x.Position, receiptKey(x))
		}
	}

	// The map of the disk's refinement: the disk has its table, and its own claims are checked.
	r = checked(t, mutated(t, "declarations", noCrash, func(m *umpirespb.Model) {
		stored := functionNamed(m, admDeclaredPkg+"stored")
		stored.Body = hole(stored.GetBody())
	}))
	require.Equal(t, map[string]ReceiptKind{
		"refinement disk store":                      Incomplete,
		"composition detailedPair":                   Incomplete,
		"query detailedPair bothPut":                 Unsupported,
		"query disk putStoresThroughDisk":            Incomplete,
		"query disk durableStays":                    Verified,
		"query disk putAccepted":                     Verified,
		"query store putStores":                      Found,
		"query pair keptTogether":                    Verified,
		"progress disk durableEventually deadlock":   Verified,
		"progress disk durableEventually fair-cycle": Verified,
		"progress disk durableEventually deadline":   Verified,
	}, kinds(r))
	require.Equal(t, outside, holes(receiptOf(t, r, "refinement disk store")))
}

// The corrected admission design's ends made a hole, beside the stale design, whose violations and
// rejected refinement stand exactly as they do without it.
func TestAnUnrelatedDeclarationHoleErasesNoViolation(t *testing.T) {
	stale := func(r *Report) []Receipt {
		return slicesDelete(plain(r), func(x Receipt) bool { return x.Key.Owner != "staleAdmission" })
	}
	want := stale(checked(t, lifted(t, "admission")))
	require.Len(t, want, 8)

	r := checked(t, mutated(t, "admission", func(m *umpirespb.Model) {
		m.Holes = append(m.Holes, &umpirespb.Hole{Id: "generic.unknownEnd", Name: "unknownEnd", Position: at(1)})
		current := admMachine(m, "currentAdmission")
		current.Ends = &umpirespb.Expr{Position: current.GetEnds().GetPosition(), Kind: &umpirespb.Expr_Hole{Hole: "generic.unknownEnd"}}
	}))
	require.Equal(t, want, stale(r))
	require.Equal(t, Counterexample, receiptOf(t, r, "query staleAdmission staleAdmission.staleDelivery").Kind)
	require.Equal(t, RefinementRejected, receiptOf(t, r, "refinement staleAdmission activityProduct").Kind)

	affected := slicesDelete(r.Receipts, func(x Receipt) bool { return x.Key.Owner != "currentAdmission" })
	require.Len(t, affected, 9, "the machine, its refinement and its seven Queries")
	for _, x := range affected {
		require.Equal(t, Incomplete, x.Kind, receiptKey(x))
		require.Equal(t, []HoleReach{{Edge: DeclarationHole, ID: "generic.unknownEnd", Name: "unknownEnd"}}, holes(x), receiptKey(x))
	}
	require.Equal(t, MachineSubject, affected[0].Subject)
}

// The disk's put made to crash an empty disk into a durable one, which has no step, and the claim
// made to lead nowhere and to be unreadable at the staged disk the put reaches: the deadlock down one
// branch stands beside the hole down the other, and each kind of violation has its own receipt.
func TestAProgressPredicateHoleErasesNoViolation(t *testing.T) {
	r := checked(t, mutated(t, "declarations", returning("disk.progress.durableEventually.to", boolValue(false)), func(m *umpirespb.Model) {
		crash := function(m, "Declarations$package$.crashStep").GetBody().GetIf()
		crash.GetCondition().GetBinary().Right = admLiteral(crash.GetCondition(), stage("empty"))
		crash.Then = proto.Clone(function(m, "Declarations$package$.flushStep").GetBody().GetMatch().GetCases()[0].GetBody()).(*umpirespb.Expr)

		from := functionNamed(m, "disk.progress.durableEventually.from")
		staged := proto.Clone(from.GetBody()).(*umpirespb.Expr)
		empty := proto.Clone(from.GetBody()).(*umpirespb.Expr)
		empty.GetBinary().Right = admLiteral(empty, stage("empty"))
		from.Body = &umpirespb.Expr{Position: from.GetBody().GetPosition(), Kind: &umpirespb.Expr_If{If: &umpirespb.If{
			Condition: staged,
			Then:      &umpirespb.Expr{Position: from.GetBody().GetPosition(), Kind: &umpirespb.Expr_Hole{Hole: crashHole}},
			Else:      empty}}}
	}))
	require.Equal(t, map[ProgressKind]ReceiptKind{umpire.DeadlockKind: Counterexample, umpire.CycleKind: Incomplete,
		umpire.DeadlineKind: Incomplete}, progressParts(t, r))
	unread := []HoleReach{{Edge: ClaimHole, ID: crashHole, Name: "crashUnmodeled", Depth: 1}}
	deadlock := receiptOf(t, r, "progress disk durableEventually deadlock")
	require.Equal(t, []string{"crash"}, taken(deadlock.Witness))
	require.Equal(t, unread, holes(deadlock), "the hole is listed beside the violation")
	deadline := receiptOf(t, r, "progress disk durableEventually deadline")
	require.Equal(t, unread, holes(deadline))
	require.Equal(t, []string{"put"}, taken(deadline.Holes[0].Prefix))
	require.Contains(t, deadline.Explanation, "the hole crashUnmodeled in a claim's function at the state 'staged'")
}

// A refinement that is rejected is the same result wherever it is met: as the machine's own receipt,
// as the Query that reads through it, and as the composition whose member it lets down. Each names
// the table it read and the work it took, as the machine's receipt does.
func TestARejectedRefinementIsOneResultWhereverItIsMet(t *testing.T) {
	as := func(base Receipt, x Receipt) Receipt {
		base.Subject, base.Key, base.Position, base.Property, base.Scenario, base.Limits = x.Subject, x.Key, x.Position, x.Property, x.Scenario, x.Limits
		return base
	}
	r := checked(t, lifted(t, "admission"))
	rejected := receiptOf(t, r, "refinement staleAdmission activityProduct")
	require.Equal(t, "temporal.activity.standalone.admission.target.staleAdmission", rejected.Target)
	require.NotEmpty(t, rejected.Fingerprint)
	require.Equal(t, len(built(t, lifted(t, "admission"))["staleAdmission"].Table.Rows), rejected.TableRows)
	require.Zero(t, rejected.Explored, "the generic check does not say how many rows it read before the one it rejects")
	through := receiptOf(t, r, "query staleAdmission staleAdmission.product.pausedIsNotDispatched")
	require.Equal(t, Limits{Name: "three", Steps: 3, Actions: 3, Search: 4096}, through.Limits)
	require.Equal(t, ClaimKey{Family: "temporal.activity.standalone", Owner: "activityProduct", Name: "pausedIsNotDispatched"}, through.Property)
	require.Equal(t, as(rejected, through), through)

	// The disk is given an assumption, so the rejection carries one to compare.
	r = checked(t, mutated(t, "declarations", noCrash, returning("disk.visible", boolValue(true)), func(m *umpirespb.Model) {
		admMachine(m, "disk").Assumes = []string{admDeclaredPkg + "flushRuns"}
	}))
	rejected = receiptOf(t, r, "refinement disk store")
	require.Equal(t, []any{"fixture.declarations.target.disk", []string{"flushEventuallyRuns"}, 2, 0},
		[]any{rejected.Target, rejected.Assumptions, rejected.TableRows, rejected.Explored})
	replaced := receiptOf(t, r, "composition detailedPair")
	require.Equal(t, as(rejected, replaced), replaced)
	through = receiptOf(t, r, "query disk putStoresThroughDisk")
	require.Equal(t, Limits{Name: "two", Steps: 2, Actions: 2, Search: 64}, through.Limits)
	require.Equal(t, as(rejected, through), through)
}

func TestMalformedDeclarationsAreLocatedErrors(t *testing.T) {
	three := admIntValue(3)
	kept := func(c string) *umpirespb.Value { return admEnum("fixture.declarations.Kept", c) }
	for name, c := range map[string]struct {
		mutate  func(m *umpirespb.Model)
		receipt string
		want    string
	}{
		"ends": {func(m *umpirespb.Model) {
			ends := admMachine(m, "disk").GetEnds().GetLambda()
			ends.Body = admLiteral(ends.GetBody(), three)
		}, "machine disk", "disk: ends is 3 at empty, not a Boolean"},
		"visible": {returning("disk.visible", three), "refinement disk store", "disk: disk.visible is 3 for stored, not a Boolean"},
		"visible outcomes": {returning("disk.visibleOutcomes", three), "refinement disk store",
			"disk: disk.visibleOutcomes is 3 for accepted, not a Boolean"},
		"a same-step Property": {returning("disk.property.putAccepted", three), "query disk putAccepted",
			"disk.putAccepted: disk.property.putAccepted is 3 for the step into staged, not a Boolean"},
		"a transition Property": {returning("disk.property.durableStays", three), "query disk durableStays",
			"disk.durableStays: disk.property.durableStays is 3 for the step into staged, not a Boolean"},
		"a Property read through a refinement": {returning("store.property.putStores", three), "query disk putStoresThroughDisk",
			"store.putStores: store.property.putStores is 3 for the step into held, not a Boolean"},
		"a composition's Property": {returning("pair.property.keptTogether", three), "query pair keptTogether",
			"pair.keptTogether: pair.property.keptTogether is 3 for the step into held_held, not a Boolean"},
		"a monitor's next": {returning(admDeclaredPkg+"countStored", kept("held")), "query disk durableStays",
			"monitor storedOnce: " + admDeclaredPkg + "countStored is held after the step into staged, which is outside its states"},
		"a monitor's verdict": {returning(admDeclaredPkg+"storedOnce.violated", three), "query disk durableStays",
			"monitor storedOnce: " + admDeclaredPkg + "storedOnce.violated is 3 at once, not a Boolean"},
		"a monitor's point": {returning(admDeclaredPkg+"stagedBeforeDurable.after", three), "query disk durableStays",
			"monitor stagedBeforeDurable: " + admDeclaredPkg + "stagedBeforeDurable.after is 3 for the step into staged, not a Boolean"},
		"a monitor's initial state": {func(m *umpirespb.Model) {
			mo := admMonitor(m, "storedOnce")
			mo.Initial = admLiteral(mo.GetInitial(), kept("held"))
		}, "query disk durableStays", "monitor storedOnce: its initial state held is outside its states"},
		// The map made the identity: it reads a disk as a disk, which is no state of the store.
		"a refinement's map": {func(m *umpirespb.Model) {
			stored := functionNamed(m, admDeclaredPkg+"stored")
			stored.Body = &umpirespb.Expr{Position: stored.GetBody().GetPosition(), Kind: &umpirespb.Expr_Var{Var: "d"}}
		}, "refinement disk store", "disk: " + admDeclaredPkg + "stored reads empty as empty, which is no state of store"},
		"a progress claim's from": {returning("disk.progress.durableEventually.from", three), "progress disk durableEventually",
			"disk.durableEventually: disk.progress.durableEventually.from is 3 at empty, not a Boolean"},
		"a progress claim's to": {returning("disk.progress.durableEventually.to", three), "progress disk durableEventually",
			"disk.durableEventually: disk.progress.durableEventually.to is 3 at empty, not a Boolean"},
		"a composition's ends": {func(m *umpirespb.Model) {
			ends := admComposition(m, "pair").GetEnds().GetLambda()
			ends.Body = admLiteral(ends.GetBody(), three)
		}, "query pair keptTogether", "pair: ends is 3 at held_held, not a Boolean"},
	} {
		t.Run(name, func(t *testing.T) {
			r := checked(t, mutated(t, "declarations", noCrash, c.mutate))
			got := receiptOf(t, r, c.receipt)
			require.Equal(t, DeclarationError, got.Kind)
			var located *Error
			require.ErrorAs(t, got.Cause, &located)
			require.NotEmpty(t, located.Position)
			require.NotEmpty(t, got.Position, "the receipt points at the declaration, or where the Model is wrong")
			require.Equal(t, c.want, located.Message)
			require.False(t, got.Kind.IsCheck())
		})
	}
}

func TestAnInadmissibleModelIsOnlyAdmissionErrors(t *testing.T) {
	r := checked(t, mutated(t, "declarations", func(m *umpirespb.Model) {
		m.Version = 2
		admQuery(m, "putStores").GetLimits().Steps = -1
	}))
	var got []string
	for _, x := range r.Receipts {
		require.Equal(t, []any{ModelSubject, AdmissionError}, []any{x.Subject, x.Kind})
		var located *Error
		require.ErrorAs(t, x.Cause, &located)
		require.NotEmpty(t, located.Position)
		got = append(got, located.Message)
	}
	require.Equal(t, []string{"version 2 is not a version this reader knows", "query putStores limits steps to -1, below 0"}, got)
	require.Empty(t, r.Checks())
}

// ---- The tenfold probe ----------------------------------------------------------------------------

// counterScope is the least scope that checks the counter to k completely: its k+1 states, the
// (k+1)² states of two of them with two ticks each, of which k(k+1) are enabled for either, and a
// progress check that explores every state and reads the k states short of the top twice.
func counterScope(k int) Scope {
	return Scope{
		Ceilings:    Ceilings{Members: int64(k + 1), Evaluations: int64(k + 1)},
		Compose:     ComposeCeiling{States: int64((k + 1) * (k + 1)), Evaluations: int64(2 * (k + 1) * (k + 1)), Results: int64(2 * k * (k + 1))},
		Progress:    Limits{Name: "counter", Steps: k, Search: 3*k + 1},
		QuerySearch: (k + 1) * (k + 1),
	}
}

// The counter to 9 checks completely within its least scope. The counter to 99, ten times the states,
// checks completely within its own, and within each bound of the smaller one in turn reports that
// bound, with the work it did, and never a result.
func TestTenfoldProbe(t *testing.T) {
	const small, k = 9, 99
	verified := map[string]ReceiptKind{
		"query counter counter.all":              Verified,
		"query two two.all":                      Verified,
		"progress counter reachesTop deadlock":   Verified,
		"progress counter reachesTop fair-cycle": Verified,
		"progress counter reachesTop deadline":   Verified,
	}
	for _, n := range []int{small, k} {
		r := Check(counter{k: int64(n)}.model(), counterScope(n))
		require.Equal(t, verified, kinds(r), n)
		require.Equal(t, []int{n + 1, (n + 1) * (n + 1), 3*n + 1}, []int{receiptOf(t, r, "query counter counter.all").Explored,
			receiptOf(t, r, "query two two.all").Explored, receiptOf(t, r, "progress counter reachesTop deadline").Explored}, n)
	}

	m, sufficient, smaller := counter{k: k}.model(), counterScope(k), counterScope(small)
	t.Run("the smaller search limit", func(t *testing.T) {
		scope := sufficient
		scope.QuerySearch = smaller.QuerySearch
		r := Check(m, scope)
		// The counter's own 100 product states are within it; the composition's 10000 are not.
		require.Equal(t, Verified, receiptOf(t, r, "query counter counter.all").Kind)
		got := receiptOf(t, r, "query two two.all")
		require.Equal(t, []any{LimitReached, 100, 100}, []any{got.Kind, got.Limits.Search, got.Explored})
	})
	t.Run("the smaller composition ceiling", func(t *testing.T) {
		scope := sufficient
		scope.Compose = smaller.Compose
		r := Check(m, scope)
		got := receiptOf(t, r, "query two two.all")
		require.Equal(t, ResourceLimit, got.Kind)
		require.Equal(t, &ResourceBound{Resource: "states", Ceiling: 100, Needed: 101}, got.Bound)
		require.Equal(t, got.Bound, receiptOf(t, r, "composition two").Bound)
		require.Equal(t, Verified, receiptOf(t, r, "query counter counter.all").Kind)
	})
	t.Run("the smaller progress work", func(t *testing.T) {
		scope := sufficient
		scope.Progress.Search = smaller.Progress.Search
		got := receiptOf(t, Check(m, scope), "progress counter reachesTop deadline")
		require.Equal(t, []any{LimitReached, 28, 28}, []any{got.Kind, got.Limits.Search, got.Explored})
	})
	t.Run("the smaller progress depth", func(t *testing.T) {
		scope := sufficient
		scope.Progress.Steps = smaller.Progress.Steps
		got := receiptOf(t, Check(m, scope), "progress counter reachesTop deadline")
		require.Equal(t, []any{Unresolved, 9}, []any{got.Kind, got.Limits.Steps})
	})
	t.Run("the smaller interpretation ceilings", func(t *testing.T) {
		scope := sufficient
		scope.Ceilings = smaller.Ceilings
		r := Check(m, scope)
		// The counter is not interpreted, so neither is anything that reads it.
		require.Equal(t, map[string]ReceiptKind{"machine counter": ResourceLimit, "composition two": ResourceLimit,
			"query counter counter.all": ResourceLimit, "query two two.all": ResourceLimit,
			"progress counter reachesTop": ResourceLimit}, kinds(r))
		for _, x := range r.Receipts {
			require.Equal(t, &ResourceBound{Resource: "members", Ceiling: 10, Needed: 100}, x.Bound, receiptKey(x))
		}
	})
}

// ---- Keys, identity and provenance ----------------------------------------------------------------

// Each design declares notAdmittedWhilePaused, and the two share a Definition ID: one family, one name.
func TestSiblingClaimsStayApart(t *testing.T) {
	r := checked(t, lifted(t, "admission"))
	current := receiptOf(t, r, "query currentAdmission currentAdmission.any.notAdmittedWhilePaused")
	stale := receiptOf(t, r, "query staleAdmission staleAdmission.any.notAdmittedWhilePaused")
	const family = "temporal.activity.standalone.admission"
	require.Equal(t, ClaimKey{Family: family, Owner: "currentAdmission", Name: "notAdmittedWhilePaused"}, current.Property)
	require.Equal(t, ClaimKey{Family: family, Owner: "staleAdmission", Name: "notAdmittedWhilePaused"}, stale.Property)
	require.Equal(t, []ReceiptKind{Verified, Counterexample}, []ReceiptKind{current.Kind, stale.Kind})

	b := bind(lifted(t, "admission"), DefaultScope)
	id := func(machine string) string {
		table := b.subject(machine).table
		return umpire.KeyProperty(table, "notAdmittedWhilePaused", nil, "", nil).PropertyID(table)
	}
	require.Equal(t, family+".property.notAdmittedWhilePaused", id("currentAdmission"))
	require.Equal(t, id("currentAdmission"), id("staleAdmission"))
}

func everyModel(t *testing.T) map[string]*umpirespb.Model {
	t.Helper()
	out := map[string]*umpirespb.Model{"nexus": load(t)}
	for _, name := range []string{"admission", "channels", "closereset", "declarations", "presence"} {
		out[name] = lifted(t, name)
	}
	return out
}

func TestCheckingChangesNoTableIDOrFingerprint(t *testing.T) {
	for name, m := range everyModel(t) {
		t.Run(name, func(t *testing.T) {
			before := built(t, m)
			checked(t, m)
			b := bind(m, DefaultScope)
			for machine, mm := range built(t, m) {
				for _, table := range []*Table{mm.Table, b.subject(machine).table} {
					want := before[machine].Table
					require.Equal(t, want.IDs(), table.IDs(), machine)
					require.Equal(t, want.TargetFingerprint(), table.TargetFingerprint(), machine)
					require.Equal(t, rowsJSON(t, want), rowsJSON(t, table), machine)
					require.Equal(t, [][]string{want.States, want.Starts, want.Ends, want.Reachable, {stuck(want)}},
						[][]string{table.States, table.Starts, table.Ends, table.Reachable, {stuck(table)}}, machine)
				}
			}
		})
	}
}

// plain is a report's receipts without what provenance gives them: positions, and the errors that
// spell them.
func plain(r *Report) []Receipt {
	out := slices.Clone(r.Receipts)
	for i := range out {
		out[i].Position, out[i].Cause = "", nil
		out[i].Holes = holes(r.Receipts[i])
	}
	return out
}

func reversed[T any](xs []T) []T {
	out := slices.Clone(xs)
	slices.Reverse(out)
	return out
}

func TestProvenanceChangesNoResult(t *testing.T) {
	for _, name := range []string{"admission", "declarations"} {
		t.Run(name, func(t *testing.T) {
			want := plain(checked(t, lifted(t, name)))
			require.NotEmpty(t, want)

			moved := mutated(t, name, func(m *umpirespb.Model) {
				m.Source = "elsewhere"
				for _, p := range positions(m) {
					if p.GetFile() != "" {
						p.File, p.Line = "elsewhere/"+p.GetFile(), p.GetLine()+1000
					}
				}
			})
			require.Equal(t, want, plain(checked(t, moved)), "repositioned and relocated")

			reordered := mutated(t, name, func(m *umpirespb.Model) {
				m.Functions, m.Actions, m.Machines = reversed(m.GetFunctions()), reversed(m.GetActions()), reversed(m.GetMachines())
				m.Monitors, m.Assumptions, m.Holes = reversed(m.GetMonitors()), reversed(m.GetAssumptions()), reversed(m.GetHoles())
				m.Compositions, m.Properties, m.Scenarios = reversed(m.GetCompositions()), reversed(m.GetProperties()), reversed(m.GetScenarios())
				m.Queries, m.Progress = reversed(m.GetQueries()), reversed(m.GetProgress())
			})
			require.Equal(t, want, plain(checked(t, reordered)), "reordered")
		})
	}
}

// ---- Replay ---------------------------------------------------------------------------------------

func TestEveryWitnessReplaysThroughAFreshInterpretation(t *testing.T) {
	for name, m := range everyModel(t) {
		t.Run(name, func(t *testing.T) {
			r := checked(t, m)
			b := bind(m, DefaultScope)
			for _, x := range r.Receipts {
				require.NotContains(t, []ReceiptKind{ReplayFailed, AdmissionError, DeclarationError}, x.Kind, receiptKey(x))
				if x.Witness == nil {
					continue
				}
				if x.Subject == CompositionSubject {
					continue
				}
				table := b.subject(x.Key.Owner).table
				require.NotNil(t, table, receiptKey(x))
				require.NoError(t, table.Replay(x.Witness), receiptKey(x))
			}
		})
	}
}

// The stale design's witnesses replayed against the corrected one, which does not take their last step.
func TestARejectedWitnessIsAnError(t *testing.T) {
	m := lifted(t, "admission")
	corrected := mutated(t, "admission", func(m *umpirespb.Model) {
		admMachine(m, "staleAdmission").Steps = admMachine(m, "currentAdmission").GetSteps()
	})
	r := check(m, DefaultScope, corrected)
	stale := receiptOf(t, r, "query staleAdmission staleAdmission.staleDelivery")
	require.Equal(t, ReplayFailed, stale.Kind)
	require.False(t, stale.Kind.IsCheck())
	require.ErrorContains(t, stale.Cause, "the counterexample did not replay")
	require.ErrorContains(t, stale.Cause, "step 3 takes attemptStart")
	require.Equal(t, ReplayFailed, receiptOf(t, r, "refinement staleAdmission activityProduct").Kind)
	// A result with no witness has nothing to replay, and stands.
	require.Equal(t, Verified, receiptOf(t, r, "query staleAdmission staleAdmission.admittedBeforePause").Kind)

	// The path to a hole a check read is a witness too: here replayed against a disk that takes no put.
	noPut := mutated(t, "declarations", func(m *umpirespb.Model) {
		put := function(m, "Declarations$package$.putStep")
		put.Body = admLiteral(put.GetBody(), &umpirespb.Value{Kind: &umpirespb.Value_List{List: &umpirespb.ListValue{}}})
	})
	r = check(lifted(t, "declarations"), DefaultScope, noPut)
	for _, key := range []string{"refinement disk store", "query disk durableStays", "progress disk durableEventually deadline"} {
		got := receiptOf(t, r, key)
		require.Equal(t, ReplayFailed, got.Kind, key)
		require.ErrorContains(t, got.Cause, "the path to a hole did not replay", key)
		require.ErrorContains(t, got.Cause, "step 1 takes put, which is not enabled at 'empty'", key)
	}

	// A progress violation's witness, the path to a flushed disk with no step, against the same disk.
	r = check(mutated(t, "declarations", noCrash, neverThere), DefaultScope, noPut)
	deadlock := receiptOf(t, r, "progress disk durableEventually deadlock")
	require.Equal(t, ReplayFailed, deadlock.Kind)
	require.ErrorContains(t, deadlock.Cause, "the counterexample did not replay")
	require.ErrorContains(t, deadlock.Cause, "step 1 takes put, which is not enabled at 'empty'")
}

// ---- The counter Model ----------------------------------------------------------------------------

// counter is a small Model of no feature: a machine that ticks n from 0 to k, a composition of two of
// them, a Query over each that holds of every step, and a claim that the top follows the start.
type counter struct {
	k int64
	// skip adds a step that stays in place, with its own outcome.
	skip bool
	// monitor watches the machine for a skip.
	monitor          bool
	violatedOnceSeen bool
}

func expr(kind any) *umpirespb.Expr {
	e := &umpirespb.Expr{Position: at(1)}
	switch k := kind.(type) {
	case *umpirespb.Value:
		e.Kind = &umpirespb.Expr_Literal{Literal: k}
	case string:
		e.Kind = &umpirespb.Expr_Var{Var: k}
	case *umpirespb.FieldAccess:
		e.Kind = &umpirespb.Expr_Field{Field: k}
	case *umpirespb.Binary:
		e.Kind = &umpirespb.Expr_Binary{Binary: k}
	case *umpirespb.If:
		e.Kind = &umpirespb.Expr_If{If: k}
	case *umpirespb.Construct:
		e.Kind = &umpirespb.Expr_Construct{Construct: k}
	case *umpirespb.Copy:
		e.Kind = &umpirespb.Expr_Copy{Copy: k}
	case *umpirespb.ListOf:
		e.Kind = &umpirespb.Expr_List{List: k}
	default:
		panic("no such expression")
	}
	return e
}

func field(base *umpirespb.Expr, name string) *umpirespb.Expr {
	return expr(&umpirespb.FieldAccess{Base: base, Field: name})
}

func binary(op umpirespb.Binary_Op, left, right *umpirespb.Expr) *umpirespb.Expr {
	return expr(&umpirespb.Binary{Op: op, Left: left, Right: right})
}

func (c counter) model() *umpirespb.Model {
	const stepType = StepType
	text := func(s string) *umpirespb.Expr { return expr(&umpirespb.Value{Kind: &umpirespb.Value_Text{Text: s}}) }
	step := func(outcome string, state *umpirespb.Expr) *umpirespb.Expr {
		return expr(&umpirespb.ListOf{Items: []*umpirespb.Expr{expr(&umpirespb.Construct{Type: stepType,
			Args: []*umpirespb.Expr{expr(caseOf("O", outcome)), state, expr(&umpirespb.ListOf{}), text("")}})}})
	}
	n := field(expr("s"), "n")
	state := []*umpirespb.Param{{Name: "s", Type: named("Counter")}}
	after := []*umpirespb.Param{{Name: "after", Type: named(stepType)}}
	fn := func(name string, params []*umpirespb.Param, body *umpirespb.Expr) *umpirespb.Function {
		return &umpirespb.Function{Name: name, Position: at(20), Params: params, Body: body}
	}
	zero := &umpirespb.Value{Kind: &umpirespb.Value_Record{Record: &umpirespb.RecordValue{Type: "Counter", Fields: []*umpirespb.Value{admIntValue(0)}}}}
	skipped := binary(umpirespb.Binary_OP_EQ, field(expr("after"), "outcome"), expr(caseOf("O", "skipped")))

	m := &umpirespb.Model{Source: "generic",
		Types: []*umpirespb.Type{
			{Name: "Counter", Position: at(1), Shape: &umpirespb.Type_Record{Record: &umpirespb.Record{Fields: []*umpirespb.Field{
				{Name: "n", Type: upTo(c.k)}}}}},
			enumType("O", 2, "ok", "skipped"),
			{Name: "Two", Position: at(3), Shape: &umpirespb.Type_Record{Record: &umpirespb.Record{Fields: []*umpirespb.Field{
				{Name: "left", Type: named("Counter")}, {Name: "right", Type: named("Counter")}}}}},
		},
		Actions: []*umpirespb.Action{{Id: "generic.tick", Name: "tick", Position: at(10), Party: "generic"}},
		Functions: []*umpirespb.Function{
			fn("generic.tickStep", state, expr(&umpirespb.If{
				Condition: binary(umpirespb.Binary_OP_LT, n, expr(admIntValue(c.k))),
				Then: step("ok", expr(&umpirespb.Copy{Base: expr("s"), Updates: []*umpirespb.NamedExpr{
					{Name: "n", Value: binary(umpirespb.Binary_OP_ADD, n, expr(admIntValue(1)))}}})),
				Else: expr(&umpirespb.ListOf{})})),
			fn("generic.always", after, expr(boolValue(true))),
			fn("generic.noSkip", after, binary(umpirespb.Binary_OP_NE, field(expr("after"), "outcome"), expr(caseOf("O", "skipped")))),
			fn("generic.atZero", state, binary(umpirespb.Binary_OP_EQ, n, expr(admIntValue(0)))),
			fn("generic.atTop", state, binary(umpirespb.Binary_OP_EQ, n, expr(admIntValue(c.k)))),
		},
		Machines: []*umpirespb.Machine{{Family: "generic", Name: "counter", Position: at(30), StateType: "Counter", OutcomeType: "O",
			Starts: []*umpirespb.Expr{expr(zero)},
			Steps:  []*umpirespb.StepBinding{{Action: "generic.tick", Function: "generic.tickStep", Position: at(31)}}}},
		Compositions: []*umpirespb.Composition{{Family: "generic", Name: "two", Position: at(40), StateType: "Two",
			Members: []*umpirespb.Member{{Field: "left", Machine: "counter"}, {Field: "right", Machine: "counter"}}}},
		Properties: []*umpirespb.Property{
			{Machine: "counter", Name: "always", Position: at(50), Holds: "generic.always"},
			{Machine: "counter", Name: "noSkip", Position: at(51), Holds: "generic.noSkip"},
			{Machine: "two", Name: "always", Position: at(52), Holds: "generic.always"},
		},
		Scenarios: []*umpirespb.Scenario{
			{Machine: "counter", Name: "all", Position: at(60), Start: expr(zero), Free: true},
			{Machine: "two", Name: "all", Position: at(61), Free: true, Start: expr(&umpirespb.Value{Kind: &umpirespb.Value_Record{
				Record: &umpirespb.RecordValue{Type: "Two", Fields: []*umpirespb.Value{zero, zero}}}})},
		},
		Queries: []*umpirespb.Query{
			{Name: "counter.all", Position: at(70), Form: umpirespb.Query_FORM_VERIFY,
				Property: &umpirespb.ClaimRef{Machine: "counter", Name: "always"}, Scenario: &umpirespb.ClaimRef{Machine: "counter", Name: "all"},
				Limits: &umpirespb.Limits{Name: "wide", Steps: int32(4 * c.k), Search: 1 << 20}},
			{Name: "two.all", Position: at(71), Form: umpirespb.Query_FORM_VERIFY,
				Property: &umpirespb.ClaimRef{Machine: "two", Name: "always"}, Scenario: &umpirespb.ClaimRef{Machine: "two", Name: "all"},
				Limits: &umpirespb.Limits{Name: "wide", Steps: int32(4 * c.k), Search: 1 << 20}},
		},
		Progress: []*umpirespb.Progress{{Machine: "counter", Name: "reachesTop", Position: at(80),
			From: "generic.atZero", To: "generic.atTop", Within: int32(c.k)}},
	}
	if c.skip {
		m.Actions = append(m.Actions, &umpirespb.Action{Id: "generic.skip", Name: "skip", Position: at(11), Party: "generic"})
		m.Functions = append(m.Functions, fn("generic.skipStep", state, step("skipped", expr("s"))))
		m.Machines[0].Steps = append(m.Machines[0].Steps, &umpirespb.StepBinding{Action: "generic.skip", Function: "generic.skipStep", Position: at(32)})
		// The composition and the progress claim are about the counter that only ticks.
		m.Compositions, m.Progress = nil, nil
		m.Properties, m.Scenarios, m.Queries = m.Properties[:2], m.Scenarios[:1], m.Queries[:1]
	}
	if c.monitor {
		flag := &umpirespb.TypeRef{Ref: &umpirespb.TypeRef_Bool{Bool: &umpirespb.Empty{}}}
		violated := expr(boolValue(false))
		if c.violatedOnceSeen {
			violated = expr("seen")
		}
		m.Functions = append(m.Functions,
			fn("generic.sawSkip.next", []*umpirespb.Param{{Name: "seen", Type: flag}, {Name: "before", Type: named("Counter")},
				{Name: "after", Type: named(stepType)}}, binary(umpirespb.Binary_OP_OR, expr("seen"), skipped)),
			fn("generic.sawSkip.violated", []*umpirespb.Param{{Name: "seen", Type: flag}}, violated))
		m.Monitors = []*umpirespb.Monitor{{Id: "generic.sawSkip", Name: "sawSkip", Position: at(90), State: flag, Initial: expr(boolValue(false)),
			Next: "generic.sawSkip.next", Violated: "generic.sawSkip.violated", Evaluate: &umpirespb.Monitor_EveryStep{EveryStep: &umpirespb.Empty{}}}}
		m.Machines[0].Monitors = []string{"generic.sawSkip"}
	}
	return m
}

func TestTheCounterModelIsAdmitted(t *testing.T) {
	for name, c := range map[string]counter{"ticking": {k: 9}, "skipping": {k: 9, skip: true}, "watched": {k: 9, skip: true, monitor: true}} {
		t.Run(name, func(t *testing.T) {
			m := c.model()
			require.NoError(t, Validate(m))
			require.Len(t, built(t, m)["counter"].Table.Reachable, 10)
		})
	}
}
