package model

// A Query's total is the author's count of its static combinations (model/SEMANTICS.md, Query
// totals): the Scenario machine's or composition's whole state catalog times the scheduled slots
// within the step limit when pinned, or times its action classes, every input assignment of each,
// times the step limit when free. Each is counted before any state is reached or any row evaluated,
// so a state no step reaches and a class disabled everywhere count as any other.

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	umpirespb "go.temporal.io/server/api/umpire/v1"
	"go.temporal.io/server/common/testing/protorequire"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

const totalsDeclaredAt = admLifts + "Declarations.scala:"

// askTotals gives m's machine or composition the Property always, the free Scenario any and the
// Scenario pinned, both from start, and the Queries free of any, at generic:70, and pinned of pinned,
// at generic:71, each limited to the given steps. A machine's pinned Scenario schedules the classes,
// a composition's the keys.
func askTotals(m *umpirespb.Model, machine, state string, start *umpirespb.Value, steps int32, classes []*umpirespb.ActionClass, keys ...string) *umpirespb.Model {
	m.Functions = append(m.Functions, &umpirespb.Function{Name: "generic.always", Position: at(45),
		Params: []*umpirespb.Param{{Name: "after", Type: named(state)}},
		Body:   &umpirespb.Expr{Position: at(45), Kind: &umpirespb.Expr_Literal{Literal: boolValue(true)}}})
	m.Properties = append(m.Properties, &umpirespb.Property{Machine: machine, Name: "always", Position: at(50), Holds: "generic.always"})
	from := func(line int32) *umpirespb.Expr {
		return &umpirespb.Expr{Position: at(line), Kind: &umpirespb.Expr_Literal{Literal: start}}
	}
	m.Scenarios = append(m.Scenarios,
		&umpirespb.Scenario{Machine: machine, Name: "any", Position: at(60), Start: from(60), Free: true},
		&umpirespb.Scenario{Machine: machine, Name: "pinned", Position: at(61), Start: from(61), Actions: classes, Keys: keys})
	for i, q := range [][2]string{{"free", "any"}, {"pinned", "pinned"}} {
		m.Queries = append(m.Queries, &umpirespb.Query{Name: q[0], Position: at(70 + int32(i)), Form: umpirespb.Query_FORM_VERIFY,
			Property: &umpirespb.ClaimRef{Machine: machine, Name: "always"}, Scenario: &umpirespb.ClaimRef{Machine: machine, Name: q[1]},
			Limits: &umpirespb.Limits{Name: "limit", Steps: steps, Actions: steps, Search: 1 << 20}})
	}
	return m
}

// totalsModel is identityModel's machine m over S = {a, a-b}, whose steps are all disabled: a-b is a
// state no step reaches, and each class is disabled in every state. It binds go(p: Y, q: X), of
// 2 × 2 classes, halt(p: S), of 2, and stay, of 1. Its pinned Scenario schedules stay, halt(a-b) and
// go(b-c, a).
func totalsModel(steps int32) *umpirespb.Model {
	m := identityModel("S", "O", "",
		identityAction{name: "go", inputs: []string{"Y", "X"}}, identityAction{name: "halt", inputs: []string{"S"}}, identityAction{name: "stay"})
	return askTotals(m, "m", "S", caseOf("S", "a"), steps, []*umpirespb.ActionClass{
		{Action: "generic.stay"},
		{Action: "generic.halt", Inputs: []*umpirespb.Value{caseOf("S", "a-b")}},
		{Action: "generic.go", Inputs: []*umpirespb.Value{caseOf("Y", "b-c"), caseOf("X", "a")}},
	})
}

func totalOf(t *testing.T, m *umpirespb.Model, query string) Total {
	t.Helper()
	q := admQuery(m, query)
	require.NotNil(t, q, query)
	total, err := QueryTotal(m, q)
	require.NoError(t, err, query)
	return total
}

// declaring is m with each Query asserting the total it counts.
func declaring(t *testing.T, m *umpirespb.Model) *umpirespb.Model {
	t.Helper()
	out := proto.CloneOf(m)
	for _, q := range out.GetQueries() {
		n, ok := totalOf(t, out, q.GetName()).N()
		require.True(t, ok, q.GetName())
		q.Total = wrapperspb.Int64(n)
	}
	return out
}

// requireAdmitsOnlyTheCount sets the Query's total to its count, which Validate admits, and to one
// more, which it refuses at the Query.
func requireAdmitsOnlyTheCount(t *testing.T, m *umpirespb.Model, query string, n int64) {
	t.Helper()
	m = proto.CloneOf(m)
	q := admQuery(m, query)
	q.Total = wrapperspb.Int64(n)
	require.NoError(t, Validate(m), query)
	q.Total = wrapperspb.Int64(n + 1)
	require.ErrorContains(t, Validate(m), fmt.Sprintf("%s:%d: query %s declares a total of %d, and its static combination count is ",
		q.GetPosition().GetFile(), q.GetPosition().GetLine(), query, n+1))
}

// A free Query counts every input assignment of every action the machine binds, summed: go's 2 × 2,
// halt's 2 and stay's 1 are 7 classes, over both states and 3 steps. The table shows a-b unreached
// and no row enabled, and the count is the same.
func TestAFreeTotalCountsEveryInputAssignmentOfEveryAction(t *testing.T) {
	m := totalsModel(3)
	require.NoError(t, Validate(m))
	table := built(t, m)["m"].Table
	require.Equal(t, []string{"a"}, table.Reachable)
	require.Empty(t, table.Rows, "every class is disabled in every state")
	require.Len(t, table.States, 2)
	require.Len(t, table.Actions, 7)

	total := totalOf(t, m, "free")
	require.Equal(t, Total{Free: true, States: count{n: 2}, Classes: count{n: 7}, Steps: 3, Count: count{n: 42}}, total)
	require.Equal(t, "2 states × 7 classes × 3 steps = 42", total.String())
	requireAdmitsOnlyTheCount(t, m, "free", 42)
}

// A pinned Query counts the scheduled slots within the step limit: as many as the steps when the
// schedule is longer, as many as the schedule when the depth is longer, and none when either is
// empty. A free Query of no steps counts none either.
func TestAPinnedTotalCountsStatesTimesScheduledSlots(t *testing.T) {
	for name, c := range map[string]struct {
		steps          int32
		empty          bool
		pinned, free   int64
		pinnedSpelling string
	}{
		"depth shorter than the schedule": {steps: 2, pinned: 4, free: 28,
			pinnedSpelling: "2 states × 2 scheduled slots (the least of 2 steps and 3 scheduled actions) = 4"},
		"depth as long as the schedule": {steps: 3, pinned: 6, free: 42},
		"depth longer than the schedule": {steps: 5, pinned: 6, free: 70,
			pinnedSpelling: "2 states × 3 scheduled slots (the least of 5 steps and 3 scheduled actions) = 6"},
		"no steps": {steps: 0, pinned: 0, free: 0,
			pinnedSpelling: "2 states × 0 scheduled slots (the least of 0 steps and 3 scheduled actions) = 0"},
		"an empty schedule": {steps: 4, empty: true, pinned: 0, free: 56,
			pinnedSpelling: "2 states × 0 scheduled slots (the least of 4 steps and 0 scheduled actions) = 0"},
	} {
		t.Run(name, func(t *testing.T) {
			m := totalsModel(c.steps)
			if c.empty {
				admScenario(m, "m", "pinned").Actions = nil
			}
			require.NoError(t, Validate(m))
			pinned := totalOf(t, m, "pinned")
			require.False(t, pinned.Free)
			require.Equal(t, count{n: 2}, pinned.States)
			require.Equal(t, count{}, pinned.Classes, "a pinned Query counts no classes")
			require.Equal(t, int64(c.steps), pinned.Steps)
			require.Equal(t, count{n: c.pinned}, pinned.Count)
			if c.pinnedSpelling != "" {
				require.Equal(t, c.pinnedSpelling, pinned.String())
			}
			require.Equal(t, count{n: c.free}, totalOf(t, m, "free").Count)
			requireAdmitsOnlyTheCount(t, m, "pinned", c.pinned)
			requireAdmitsOnlyTheCount(t, m, "free", c.free)
		})
	}
}

// A composition counts its own state catalog, its members' states together, and its classes: each
// member's classes no sync takes, and each sync's pairs of the classes it takes. In the declarations
// fixture pair is Store × Store, 2 × 2 states, whose one class is the sync putBoth; detailedPair is
// Store × Disk, 2 × 3 states, with the sync putBoth and disk's own flush and crash.
func TestACompositionTotalCountsItsStateCatalogAndComposedClasses(t *testing.T) {
	m := proto.CloneOf(lifted(t, "declarations"))
	bothPut := admScenario(m, "detailedPair", "bothPut")
	m.Scenarios = append(m.Scenarios, &umpirespb.Scenario{Machine: "detailedPair", Name: "any", Position: bothPut.GetPosition(),
		Start: bothPut.GetStart(), Free: true})
	both := admQuery(m, "bothPut")
	m.Queries = append(m.Queries, &umpirespb.Query{Name: "detailedPair.any", Position: both.GetPosition(), Form: umpirespb.Query_FORM_VERIFY,
		Property: both.GetProperty(), Scenario: &umpirespb.ClaimRef{Machine: "detailedPair", Name: "any"}, Limits: both.GetLimits()})
	require.NoError(t, Validate(m))

	for query, c := range map[string]struct {
		total    Total
		spelling string
	}{
		"keptTogether": {Total{Free: true, States: count{n: 4}, Classes: count{n: 1}, Steps: 2, Count: count{n: 8}},
			"4 states × 1 classes × 2 steps = 8"},
		"detailedPair.any": {Total{Free: true, States: count{n: 6}, Classes: count{n: 3}, Steps: 2, Count: count{n: 36}},
			"6 states × 3 classes × 2 steps = 36"},
		// Pinned by its one class key.
		"bothPut": {Total{States: count{n: 6}, Steps: 2, Schedule: 1, Count: count{n: 6}},
			"6 states × 1 scheduled slots (the least of 2 steps and 1 scheduled actions) = 6"},
	} {
		total := totalOf(t, m, query)
		require.Equal(t, c.total, total, query)
		require.Equal(t, c.spelling, total.String(), query)
		requireAdmitsOnlyTheCount(t, m, query, c.total.Count.n)
	}
}

// compositionModel's p with b_c and c each given an input of 0..9 and synced as both, and m2 also
// binding x, which no sync takes: 10 × 10 pairs and 1 class of its own, over S × S.
func TestACompositionTotalCountsEachSyncsPairsOfInputAssignments(t *testing.T) {
	m := compositionModel("a", "d", "both")
	for _, a := range m.GetActions() {
		a.Inputs = []*umpirespb.Param{{Name: "n", Type: upTo(9)}}
	}
	for _, f := range m.GetFunctions() {
		f.Params = append(f.Params, &umpirespb.Param{Name: "n", Type: upTo(9)})
	}
	m = alsoBinds(m, "x")
	start := &umpirespb.Value{Kind: &umpirespb.Value_Record{Record: &umpirespb.RecordValue{Type: "P",
		Fields: []*umpirespb.Value{caseOf("S", "a"), caseOf("S", "a")}}}}
	m = askTotals(m, "p", "P", start, 2, nil, "both-0-0", "d_x")
	require.NoError(t, Validate(m))

	free := totalOf(t, m, "free")
	require.Equal(t, "4 states × 101 classes × 2 steps = 808", free.String())
	pinned := totalOf(t, m, "pinned")
	require.Equal(t, "4 states × 2 scheduled slots (the least of 2 steps and 2 scheduled actions) = 8", pinned.String())
	requireAdmitsOnlyTheCount(t, m, "free", 808)
	requireAdmitsOnlyTheCount(t, m, "pinned", 8)
}

// A Query read through a refinement counts the Scenario's machine, not the refined product it reads
// the Property through: putStoresThroughDisk runs disk's putThenFlush, 3 disk states × 2 slots, as
// putAccepted does, and not store's 2 states. An author who counted the product is told the count.
func TestATotalReadThroughARefinementCountsTheScenarioMachine(t *testing.T) {
	m := lifted(t, "declarations")
	through := admQuery(m, "putStoresThroughDisk")
	require.True(t, through.GetThrough())
	require.Equal(t, "store", through.GetProperty().GetMachine())
	total := totalOf(t, m, "putStoresThroughDisk")
	require.Equal(t, Total{States: count{n: 3}, Steps: 2, Schedule: 2, Count: count{n: 6}}, total)
	require.Equal(t, totalOf(t, m, "putAccepted"), total)
	require.Equal(t, count{n: 2}, totalOf(t, m, "putStores").States, "store, the refined product, has 2 states")

	m = proto.CloneOf(m)
	admQuery(m, "putStoresThroughDisk").Total = wrapperspb.Int64(4)
	require.EqualError(t, Validate(m), totalsDeclaredAt+"163: query putStoresThroughDisk declares a total of 4, "+
		"and its static combination count is 3 states × 2 scheduled slots (the least of 2 steps and 2 scheduled actions) = 6")
}

// A total that is not the count is refused at its Query, with what it declares, the count and the
// factors; every such Query is reported, not only the first.
func TestATotalThatIsNotTheCountIsRefusedWithItsFactors(t *testing.T) {
	m := declaring(t, lifted(t, "declarations"))
	require.NoError(t, Validate(m))
	admQuery(m, "keptTogether").Total = wrapperspb.Int64(9)
	admQuery(m, "putAccepted").Total = wrapperspb.Int64(7)
	require.EqualError(t, Validate(m), strings.Join([]string{
		totalsDeclaredAt + "166: query keptTogether declares a total of 9, and its static combination count is 4 states × 1 classes × 2 steps = 8",
		totalsDeclaredAt + "161: query putAccepted declares a total of 7, and its static combination count is " +
			"3 states × 2 scheduled slots (the least of 2 steps and 2 scheduled actions) = 6",
	}, "\n"))
}

func TestATotalBelowZeroIsRefused(t *testing.T) {
	m := totalsModel(3)
	admQuery(m, "free").Total = wrapperspb.Int64(-1)
	require.EqualError(t, Validate(m), "generic:70: query free declares a total of -1, below 0: its static combination count is 2 states × 7 classes × 3 steps = 42")
}

// A count past an int64 is refused whatever the total declares, though each factor fits one: they are
// counted, never listed. Two state fields of 0..2^31 are about 2^62 states, past an int64 over 2
// scheduled slots; an input of 0..2^40 is about 2^40 classes, past it over 2 states and 2^30 steps.
func TestATotalPastAnInt64IsRefused(t *testing.T) {
	wideStates := func() *umpirespb.Model {
		m := identityModel("S", "O", "", identityAction{name: "stay"})
		m.Types[3] = &umpirespb.Type{Name: "S", Position: at(4), Shape: &umpirespb.Type_Record{Record: &umpirespb.Record{Fields: []*umpirespb.Field{
			{Name: "x", Type: upTo(1 << 31)}, {Name: "y", Type: upTo(1 << 31)}}}}}
		zero := &umpirespb.Value{Kind: &umpirespb.Value_Record{Record: &umpirespb.RecordValue{Type: "S",
			Fields: []*umpirespb.Value{admIntValue(0), admIntValue(0)}}}}
		m.Machines[0].Starts[0].Kind = &umpirespb.Expr_Literal{Literal: zero}
		return askTotals(m, "m", "S", zero, 2, []*umpirespb.ActionClass{{Action: "generic.stay"}, {Action: "generic.stay"}})
	}
	wideClasses := func() *umpirespb.Model {
		m := identityModel("S", "O", "", identityAction{name: "wide", inputs: []string{"S"}})
		m.Actions[0].Inputs[0].Type = upTo(1 << 40)
		m.Functions[0].Params[1].Type = upTo(1 << 40)
		return askTotals(m, "m", "S", caseOf("S", "a"), 1<<30, nil)
	}
	for name, c := range map[string]struct {
		model   *umpirespb.Model
		query   string
		at      string
		factors string
	}{
		"states":  {wideStates(), "pinned", "generic:71", "4611686022722355201 states × 2 scheduled slots"},
		"classes": {wideClasses(), "free", "generic:70", "2 states × 1099511627777 classes × 1073741824 steps"},
	} {
		t.Run(name, func(t *testing.T) {
			m := c.model
			require.NoError(t, Validate(m), "a Query declaring no total is not counted at admission")
			total := totalOf(t, m, c.query)
			_, fits := total.N()
			require.False(t, fits)
			for _, declared := range []int64{0, 5, 1<<63 - 1} {
				admQuery(m, c.query).Total = wrapperspb.Int64(declared)
				err := Validate(m)
				require.ErrorContains(t, err, c.at+": query "+c.query)
				require.ErrorContains(t, err, c.factors)
				require.ErrorContains(t, err, "= more than 9223372036854775807, more than an int64 holds")
			}
			_, err := WithTotals(m)
			require.ErrorContains(t, err, "more than an int64 holds")
		})
	}
}

// IR lifted before totals existed asserts none, and Validate admits it; RequireTotals, which the lifter
// and whoever generates Cases apply to current IR, refuses each such Query with the count it would
// assert.
func TestAnAbsentTotalIsAdmittedAndNotRequiredOfHistoricalIR(t *testing.T) {
	// The fixture as lifted before Query totals existed.
	m := WithoutTotals(lifted(t, "declarations"))
	require.NoError(t, Validate(m))
	err := RequireTotals(m)
	require.Error(t, err)
	lines := strings.Split(err.Error(), "\n")
	require.Len(t, lines, len(m.GetQueries()), "every Query is reported")
	for _, l := range lines {
		require.Contains(t, l, " declares no total: its static combination count is ")
	}
	require.Contains(t, lines, totalsDeclaredAt+"162: query putStores declares no total: "+
		"its static combination count is 2 states × 1 scheduled slots (the least of 2 steps and 1 scheduled actions) = 2")

	declared := declaring(t, m)
	require.NoError(t, Validate(declared))
	require.NoError(t, RequireTotals(declared))
	admQuery(declared, "putStores").Total = nil
	require.EqualError(t, RequireTotals(declared), totalsDeclaredAt+"162: query putStores declares no total: "+
		"its static combination count is 2 states × 1 scheduled slots (the least of 2 steps and 1 scheduled actions) = 2")
}

// An exploration that changes a Scenario's schedule recounts the candidate it derives: WithTotals
// asserts each candidate Query's count, leaves a Query that asserts none without one, and rewrites
// neither the source nor the Model it is given.
func TestWithTotalsRecountsADerivedSchedule(t *testing.T) {
	source := declaring(t, lifted(t, "declarations"))
	admQuery(source, "putStores").Total = nil
	before := proto.CloneOf(source)

	derived := proto.CloneOf(source)
	admScenario(derived, "disk", "putThenFlush").Actions = admScenario(derived, "disk", "putThenFlush").GetActions()[:1]
	given := proto.CloneOf(derived)
	require.ErrorContains(t, Validate(derived), "query putAccepted declares a total of 6, and its static combination count is "+
		"3 states × 1 scheduled slots (the least of 2 steps and 1 scheduled actions) = 3")

	recounted, err := WithTotals(derived)
	require.NoError(t, err)
	require.NoError(t, Validate(recounted))
	require.Equal(t, int64(3), admQuery(recounted, "putAccepted").GetTotal().GetValue())
	require.Equal(t, int64(3), admQuery(recounted, "putStoresThroughDisk").GetTotal().GetValue())
	require.Equal(t, int64(18), admQuery(recounted, "durableStays").GetTotal().GetValue(), "a Query of another Scenario keeps its count")
	require.Nil(t, admQuery(recounted, "putStores").GetTotal())
	protorequire.ProtoEqual(t, before, source)
	protorequire.ProtoEqual(t, given, derived)
}

// WithoutTotals is the Model with no Query's total and nothing else changed, and leaves its argument alone.
func TestWithoutTotalsClearsOnlyTotals(t *testing.T) {
	m := declaring(t, lifted(t, "declarations"))
	given := proto.CloneOf(m)
	expected := proto.CloneOf(m)
	for _, q := range expected.GetQueries() {
		q.Total = nil
	}
	protorequire.ProtoEqual(t, expected, WithoutTotals(m))
	protorequire.ProtoEqual(t, given, m)
	// The lifted fixture asserts every total, which is its count.
	current := lifted(t, "declarations")
	for _, q := range current.GetQueries() {
		require.NotNil(t, q.GetTotal(), q.GetName())
	}
	protorequire.ProtoEqual(t, current, m)
	protorequire.ProtoEqual(t, expected, WithoutTotals(current))
}

// The documentation's arithmetic, on a current Model. activityProtocol's state is ProtocolState:
// 12 phases × 3 attempt counts (0..2) × 2 × 2 × 2 Timeouts, its scheduleToClose, scheduleToStart and
// startToClose, is 288 states, before any is reached.
//
//	completion: pinned, completed schedules 3 actions under limits three (3 steps):
//	  288 states × min(3, 3) = 288 × 3 = 864
//	activityProduct.pausedIsNotDispatched: generated by the product's capabilities, a free search of
//	  activityProduct, ProductState's 9 phases, by its 11 classes under limits three:
//	  9 × 11 × 3 = 297
//	stoppedWorkerStartsNothing: the composition standaloneActivity, ProtocolState × worker State
//	  (2 phases), 576 states; stoppedBeforeRetry schedules 6 keys under 6 steps: 576 × 6 = 3456
//
// And on the system contract's Model, where a Query still reads the product's law through a
// refinement (the activity's own pauseHolds did, before the capabilities retired it):
//
//	currentAdmission.product.pausedIsNotDispatched: read through the refinement to activityProduct,
//	  and still counted on currentAdmission, 36 states; staleDeliveryAfterPause schedules 3 actions
//	  under limits three: 36 × min(3, 3) = 108
func TestTheActivityModelsTotalsAreStatesTimesSlots(t *testing.T) {
	m, err := Load(filepath.Join("..", "..", "..", "model", "ir", "activity.json"))
	require.NoError(t, err)
	for query, spelling := range map[string]string{
		"completion":                            "288 states × 3 scheduled slots (the least of 3 steps and 3 scheduled actions) = 864",
		"activityProduct.pausedIsNotDispatched": "9 states × 11 classes × 3 steps = 297",
		"stoppedWorkerStartsNothing":            "576 states × 6 scheduled slots (the least of 6 steps and 6 scheduled actions) = 3456",
	} {
		require.Equal(t, spelling, totalOf(t, m, query).String(), query)
	}
	require.False(t, admQuery(m, "activityProduct.pausedIsNotDispatched").GetThrough())
	system, err := Load(activitySystemIR)
	require.NoError(t, err)
	const through = "currentAdmission.product.pausedIsNotDispatched"
	require.Equal(t, "36 states × 3 scheduled slots (the least of 3 steps and 3 scheduled actions) = 108",
		totalOf(t, system, through).String())
	require.True(t, admQuery(system, through).GetThrough())
}
