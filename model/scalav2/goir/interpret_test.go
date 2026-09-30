package goir

// The lifted fixtures interpreted: choices, presence, bounded channels and holes, each against rows
// worked out by hand from the fixture's Scala source.

import (
	"math"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	modelirspb "go.temporal.io/server/api/modelir/v1"
	"go.temporal.io/server/model/go/umpire"
	"google.golang.org/protobuf/proto"
)

func built(t *testing.T, m *modelirspb.Model) map[string]*Machine {
	t.Helper()
	out, err := Build(m)
	require.NoError(t, err)
	return out
}

func row(t *testing.T, mm *Machine, key string) umpire.Row {
	t.Helper()
	for _, r := range mm.Table.Rows {
		if r.Key == key {
			return r
		}
	}
	require.Failf(t, "no row", "%s has no row %s", mm.Decl.GetName(), key)
	return umpire.Row{}
}

func hasRow(mm *Machine, key string) bool {
	for _, r := range mm.Table.Rows {
		if r.Key == key {
			return true
		}
	}
	return false
}

const again = "the channel delivers the message again"

func TestChoiceKeepsEveryResultInOrder(t *testing.T) {
	mm := built(t, lifted(t, "admission"))["currentAdmission"]
	facts := []string{"statusStarted", "attemptAdmitted"}
	require.Equal(t, umpire.Row{Key: "scheduled-queued-none-attemptStart", Source: "scheduled-queued-none", Action: "attemptStart",
		Results: []umpire.Result{
			{Outcome: "accepted", State: "started-empty-one", Facts: facts},
			{Outcome: "accepted", State: "started-redelivery-one", Facts: facts, Because: "the channel may deliver the message again"},
		}}, row(t, mm, "scheduled-queued-none-attemptStart"))
}

// Transitions are the table's rows as values, in the same order, so a check can evaluate a monitor or
// a Property over the very steps the table keys.
func TestTransitionsAreTheRowsAsValues(t *testing.T) {
	mm := built(t, lifted(t, "admission"))["currentAdmission"]
	require.Len(t, mm.Transitions, len(mm.Table.Rows))
	for i, tr := range mm.Transitions {
		r := mm.Table.Rows[i]
		require.Equal(t, r.Key, tr.Row)
		require.Equal(t, r.Source, tr.Source.Key())
		require.Equal(t, r.Action, tr.Class.Key)
		require.Len(t, tr.Steps, len(r.Results))
		for j, step := range tr.Steps {
			require.Equal(t, StepType, step.Type)
			require.Equal(t, r.Results[j].Outcome, step.Fields[0].Key())
			require.Equal(t, r.Results[j].State, step.Fields[1].Key())
			require.Equal(t, r.Results[j].Because, step.Fields[3].Text)
		}
	}
	redelivered, ok := mm.State("started-redelivery-one")
	require.True(t, ok)
	for _, tr := range mm.Transitions {
		if tr.Row == "scheduled-queued-none-attemptStart" {
			require.True(t, tr.Steps[1].Fields[1].Equal(redelivered))
		}
	}
}

// A refinement that does not hold is the machine's, not the Model's: its table is still built, so a
// check can find the counterexample on it.
func TestARejectedRefinementStaysWithItsMachine(t *testing.T) {
	machines := built(t, lifted(t, "admission"))
	require.NoError(t, machines["currentAdmission"].Rejected)
	require.ErrorContains(t, machines["staleAdmission"].Rejected, "staleAdmission refines activityProduct: the row "+
		"'paused-queued-none-attemptStart' steps from 'paused-queued-none' to 'started-empty-one', which read as 'paused' and 'started'")
	require.NotEmpty(t, machines["staleAdmission"].Table.Rows)
}

func TestPresenceIsAnOrdinaryEnum(t *testing.T) {
	mm := built(t, lifted(t, "presence"))["presence"]
	// Report: unsent, sent × 2 × 2; kept: None, Some × 2; retries 0..2; polls 0..1.
	require.Len(t, mm.Table.States, 5*3*3*2)
	require.Equal(t, []string{"unsent-None-0-0"}, mm.Table.Starts)
	require.Len(t, mm.Table.Ends, 5*2*3*2)
	require.Equal(t, []umpire.Result{{Outcome: "accepted", State: "sent-failed-false-Some-failed-0-0", Facts: []string{}}},
		row(t, mm, "sent-failed-false-None-0-0-keep").Results)
	require.Equal(t, []umpire.Result{{Outcome: "ignored", State: "unsent-None-0-0", Facts: []string{}}},
		row(t, mm, "unsent-None-0-0-forget").Results)
	require.Equal(t, []umpire.Result{{Outcome: "accepted", State: "unsent-None-0-0", Facts: []string{}}},
		row(t, mm, "unsent-Some-succeeded-0-0-forget").Results)
	require.False(t, hasRow(mm, "sent-failed-true-None-0-0-send-failed"), "a retried send is disabled")
	require.True(t, mm.Disabled("sent-failed-true-None-0-0", "send-failed"))
}

func TestChannelCatalogs(t *testing.T) {
	mm := built(t, lifted(t, "channels"))["relay"]
	// heard 3 × wire (1 + 4 + 4², its entries ping-0, ping-1, pong-0, pong-1) × radio (1 + 2).
	require.Len(t, mm.Table.States, 3*21*3)
	require.Equal(t, []string{"nothing-[]-[]", "nothing-[]-[up-0]", "nothing-[]-[down-0]", "nothing-[ping-0]-[]",
		"nothing-[ping-0]-[up-0]"}, mm.Table.States[:5])
	require.Contains(t, mm.Table.States, "signal-[pong-1,ping-0]-[down-0]")
	require.Equal(t, []string{"flash-down", "flash-up", "radioDelivery-down", "radioDelivery-up", "radioLoss-down",
		"radioLoss-up", "talk-ping", "talk-pong", "wireDelivery-ping", "wireDelivery-pong"}, mm.Table.Actions)

	m := proto.Clone(lifted(t, "channels")).(*modelirspb.Model)
	for _, c := range m.GetChannels() {
		if c.GetName() == "radio" {
			c.Capacity = 2
		}
	}
	radio, err := NewInterpreter(m).Members(&modelirspb.TypeRef{Ref: &modelirspb.TypeRef_Channel{Channel: "fixture.channels.Channels$package$.radio"}})
	require.NoError(t, err)
	keys := make([]string, len(radio))
	for i, v := range radio {
		keys[i] = v.Key()
	}
	require.Equal(t, []string{"[]", "[up-0]", "[down-0]", "[up-0,up-0]", "[up-0,down-0]", "[down-0,down-0]"}, keys,
		"an unordered channel holds one value per multiset")
}

func TestChannelSendAndDelivery(t *testing.T) {
	machines := built(t, lifted(t, "channels"))
	relay := machines["relay"]
	require.Equal(t, []umpire.Result{{Outcome: "accepted", State: "nothing-[ping-0]-[]", Facts: []string{}}},
		row(t, relay, "nothing-[]-[]-talk-ping").Results)
	require.Equal(t, []umpire.Result{{Outcome: "accepted", State: "nothing-[ping-0,pong-0]-[]", Facts: []string{}}},
		row(t, relay, "nothing-[ping-0]-[]-talk-pong").Results, "a FIFO send appends")
	require.True(t, relay.Disabled("nothing-[ping-0,pong-0]-[]", "talk-ping"), "the step refuses a full channel")

	// The receiver's step, and again with the message put back first, its acknowledgment lost.
	require.Equal(t, []umpire.Result{
		{Outcome: "accepted", State: "note-[]-[]", Facts: []string{}},
		{Outcome: "accepted", State: "note-[ping-1]-[]", Facts: []string{}, Because: again},
	}, row(t, relay, "nothing-[ping-0]-[]-wireDelivery-ping").Results)
	require.Equal(t, []umpire.Result{
		{Outcome: "accepted", State: "note-[pong-0]-[]", Facts: []string{}},
		{Outcome: "accepted", State: "note-[ping-1,pong-0]-[]", Facts: []string{}, Because: again},
	}, row(t, relay, "nothing-[ping-0,pong-0]-[]-wireDelivery-ping").Results)
	require.Equal(t, []umpire.Result{{Outcome: "accepted", State: "note-[]-[]", Facts: []string{}}},
		row(t, relay, "nothing-[ping-1]-[]-wireDelivery-ping").Results, "a message delivered its duplicates times is not again")
	require.True(t, relay.Disabled("nothing-[pong-0,ping-0]-[]", "wireDelivery-ping"), "a FIFO channel delivers its first message only")
	require.True(t, relay.Disabled("nothing-[pong-0]-[]", "wireDelivery-pong"), "the receiver does not take pong")
	require.True(t, relay.Disabled("nothing-[]-[]", "wireDelivery-ping"), "an empty channel delivers nothing")

	require.Equal(t, []umpire.Result{{Outcome: "accepted", State: "signal-[]-[]", Facts: []string{}}},
		row(t, relay, "nothing-[]-[up-0]-radioDelivery-up").Results)
	require.True(t, relay.Disabled("nothing-[]-[up-0]", "radioDelivery-down"))
	require.Equal(t, []umpire.Result{{Outcome: "dropped", State: "nothing-[]-[]", Facts: []string{}}},
		row(t, relay, "nothing-[]-[up-0]-radioLoss-up").Results)
	require.True(t, relay.Disabled("nothing-[]-[up-0]", "radioLoss-down"))

	tallying := machines["tallying"]
	require.Equal(t, []string{"nothing-[]", "nothing-[0-0]", "nothing-[1-0]", "nothing-[2-0]"}, tallying.Table.States[:4])
	require.Equal(t, []string{"count", "tallyDelivery-0", "tallyDelivery-1", "tallyDelivery-2"}, tallying.Table.Actions)
	require.Equal(t, []string{"nothing-[]", "nothing-[2-0]", "note-[]", "note-[2-0]"}, tallying.Table.Reachable)
	require.Equal(t, []umpire.Result{{Outcome: "accepted", State: "note-[]", Facts: []string{}}},
		row(t, tallying, "nothing-[2-0]-tallyDelivery-2").Results)
}

func TestUnorderedSendKeepsCatalogOrder(t *testing.T) {
	m := lifted(t, "channels")
	at := &modelirspb.Position{File: "generic", Line: 1}
	signal := func(c string) *modelirspb.Value {
		return &modelirspb.Value{Kind: &modelirspb.Value_Enum{Enum: &modelirspb.EnumValue{Type: "fixture.channels.Signal", Case: c}}}
	}
	held := &modelirspb.Value{Kind: &modelirspb.Value_List{List: &modelirspb.ListValue{Items: []*modelirspb.Value{
		{Kind: &modelirspb.Value_Record{Record: &modelirspb.RecordValue{Type: DeliveryType,
			Fields: []*modelirspb.Value{signal("down"), {Kind: &modelirspb.Value_Int{Int: 0}}}}}}}}}}
	send := func(channel string) *modelirspb.Expr {
		return &modelirspb.Expr{Position: at, Kind: &modelirspb.Expr_Inbox{Inbox: &modelirspb.Inbox{Op: modelirspb.Inbox_OP_SEND,
			Channel:  "fixture.channels.Channels$package$." + channel,
			Contents: &modelirspb.Expr{Position: at, Kind: &modelirspb.Expr_Literal{Literal: held}},
			Message:  &modelirspb.Expr{Position: at, Kind: &modelirspb.Expr_Literal{Literal: signal("up")}}}}}
	}
	in := NewInterpreter(m)
	v, err := in.Eval(send("radio"))
	require.NoError(t, err)
	require.Equal(t, "[up-0,down-0]", v.Key(), "unordered: at its catalog position")
	full, err := in.Eval(&modelirspb.Expr{Position: at, Kind: &modelirspb.Expr_Inbox{Inbox: &modelirspb.Inbox{
		Op: modelirspb.Inbox_OP_IS_FULL, Channel: "fixture.channels.Channels$package$.radio",
		Contents: &modelirspb.Expr{Position: at, Kind: &modelirspb.Expr_Literal{Literal: held}}}}})
	require.NoError(t, err)
	require.True(t, full.Bool)
}

// A send the step does not guard lands outside the channel's catalog, and so outside the state domain.
func TestASendToAFullChannelLeavesTheDomain(t *testing.T) {
	m := proto.Clone(lifted(t, "channels")).(*modelirspb.Model)
	talk := function(m, "Channels$package$.talkStep")
	talk.Body = talk.GetBody().GetIf().GetElse()
	_, err := Build(m)
	require.ErrorContains(t, err, "Channels.scala.fixture:55: relay: row nothing-[ping-0,ping-0]-[]-talk-ping lands in "+
		"nothing-[ping-0,ping-0,ping-0]-[], which is outside the state domain")
}

func TestADeclaredHoleIsNeitherARowNorDisabled(t *testing.T) {
	disk := built(t, lifted(t, "declarations"))["disk"]
	require.Equal(t, []string{"crash", "flush", "put"}, disk.Table.Actions)
	var keys []string
	for _, r := range disk.Table.Rows {
		keys = append(keys, r.Key)
	}
	require.Equal(t, []string{"empty-put", "staged-flush"}, keys)
	require.Len(t, disk.Holes, 1)
	h := disk.Holes[0]
	require.Equal(t, []string{"staged-crash", "staged", "crash", "fixture.declarations.Declarations$package$.crashUnmodeled"},
		[]string{h.Row, h.Source, h.Class, h.Hole.ID})
	require.Equal(t, disk.Holes, disk.ReachableHoles())
	require.False(t, disk.Disabled("staged", "crash"), "a hole row is not disabled")
	require.True(t, disk.Disabled("empty", "crash"))
	require.True(t, disk.Disabled("durable", "crash"))
	require.False(t, disk.Disabled("empty", "put"), "a row is not disabled")
	require.False(t, disk.Disabled("empty", "nothing"), "no such class")
}

// flushStep without its wildcard case: the stages it no longer matches are undeclared holes.
func TestAnUnmatchedValueIsAnUndeclaredHole(t *testing.T) {
	m := proto.Clone(lifted(t, "declarations")).(*modelirspb.Model)
	flush := function(m, "Declarations$package$.flushStep")
	cases := flush.GetBody().GetMatch().GetCases()
	flush.GetBody().GetMatch().Cases = cases[:len(cases)-1]
	disk := built(t, m)["disk"]
	var got [][3]string
	for _, h := range disk.Holes {
		got = append(got, [3]string{h.Row, h.Hole.ID, h.Hole.Message})
	}
	require.Equal(t, [][3]string{
		{"empty-flush", "", "no case matches empty"},
		{"staged-crash", "fixture.declarations.Declarations$package$.crashUnmodeled", "reaches the hole crashUnmodeled"},
		{"durable-flush", "", "no case matches durable"},
	}, got)
	require.True(t, hasRow(disk, "staged-flush"))
	// durable is reachable, so its hole is; the state no path reaches has none.
	var reachable []string
	for _, h := range disk.ReachableHoles() {
		reachable = append(reachable, h.Row)
	}
	require.Equal(t, []string{"empty-flush", "staged-crash", "durable-flush"}, reachable)
}

func TestReachableHolesAreOnlyThoseAPathReaches(t *testing.T) {
	m := proto.Clone(lifted(t, "declarations")).(*modelirspb.Model)
	// Without put, nothing leaves empty: the staged crash hole is unreachable.
	for _, mm := range m.GetMachines() {
		if mm.GetName() == "disk" {
			mm.Steps = slicesDelete(mm.GetSteps(), func(b *modelirspb.StepBinding) bool { return strings.HasSuffix(b.GetAction(), ".put") })
		}
	}
	disk := built(t, m)["disk"]
	require.Len(t, disk.Holes, 1)
	require.Empty(t, disk.ReachableHoles())
}

func slicesDelete[T any](xs []T, drop func(T) bool) []T {
	var out []T
	for _, x := range xs {
		if !drop(x) {
			out = append(out, x)
		}
	}
	return out
}

func TestVisibleProjectionOfARefinement(t *testing.T) {
	disk := built(t, lifted(t, "declarations"))["disk"]
	require.NoError(t, disk.Rejected)
	put := "put"
	require.Equal(t, []umpire.RefinementRow{{Key: "empty-put", Product: &put}, {Key: "staged-flush"}}, disk.Refinement)

	for name, visible := range map[string]string{
		"a stutter records a fact the product sees":   "disk.visible",
		"a stutter's outcome is one the product sees": "disk.visibleOutcomes",
	} {
		t.Run(name, func(t *testing.T) {
			m := proto.Clone(lifted(t, "declarations")).(*modelirspb.Model)
			f := function(m, visible)
			f.Body = &modelirspb.Expr{Position: f.GetBody().GetPosition(),
				Kind: &modelirspb.Expr_Literal{Literal: &modelirspb.Value{Kind: &modelirspb.Value_Bool{Bool: true}}}}
			disk := built(t, m)["disk"]
			require.NotEmpty(t, disk.ReachableHoles(), "the reachable crash hole does not erase the rejection")
			require.ErrorContains(t, disk.Rejected, "disk refines store: the row 'staged-flush'")
		})
	}
}

func TestMonitorsAndAssumptionsAreNamedNotApplied(t *testing.T) {
	m := lifted(t, "declarations")
	machines := built(t, m)
	disk := machines["disk"]
	var monitors []string
	for _, mon := range disk.Monitors {
		monitors = append(monitors, mon.GetName())
	}
	require.Equal(t, []string{"storedOnce", "endsDurable", "stagedBeforeDurable"}, monitors)
	require.Len(t, machines["store"].Assumptions, 1)
	require.Equal(t, "storeOpaque", machines["store"].Assumptions[0].GetName())

	unwatched := proto.Clone(m).(*modelirspb.Model)
	for _, mm := range unwatched.GetMachines() {
		mm.Monitors = nil
	}
	require.Equal(t, disk.Table.Rows, built(t, unwatched)["disk"].Table.Rows, "a monitor never disables a row")

	// A monitor's next function, evaluated over a transition's typed step.
	in := NewInterpreter(m)
	put := disk.Transitions[0]
	require.Equal(t, "empty-put", put.Row)
	seen, err := in.Call(disk.Monitors[0].GetNext(), []Value{{Kind: EnumValue, Type: "fixture.declarations.Seen", Case: "never"},
		put.Source, put.Steps[0]}, nil)
	require.NoError(t, err)
	require.Equal(t, "once", seen.Key())
}

// Retries widened tenfold, 0..2 to 0..29: presence's states grow from 90 to 900.
func TestTenfoldInputWithinAndBeyondCeilings(t *testing.T) {
	m := proto.Clone(lifted(t, "presence")).(*modelirspb.Model)
	for _, ty := range m.GetTypes() {
		for _, f := range ty.GetRecord().GetFields() {
			if f.GetName() == "retries" {
				f.GetType().GetIntRange().High = 29
			}
		}
	}
	machines, err := BuildWithin(m, Ceilings{Members: 900, Evaluations: 900 * 5})
	require.NoError(t, err)
	require.Len(t, machines["presence"].Table.States, 900)
	require.Equal(t, Work{States: 900, Classes: 5, Evaluations: 4500}, machines["presence"].Work)

	_, err = BuildWithin(m, Ceilings{Members: 90, Evaluations: 90 * 5})
	var limit *LimitError
	require.ErrorAs(t, err, &limit)
	require.Equal(t, LimitError{Machine: "presence", Resource: "members", Ceiling: 90, Needed: 900}, *limit)

	_, err = BuildWithin(m, Ceilings{Members: 900, Evaluations: 90 * 5})
	require.ErrorAs(t, err, &limit)
	require.Equal(t, LimitError{Machine: "presence", Resource: "evaluations", Ceiling: 450, Needed: 4500}, *limit)
}

// A catalog too large to list is refused from its size alone, before any of it is allocated.
func TestACatalogIsCountedBeforeItIsListed(t *testing.T) {
	m := proto.Clone(lifted(t, "channels")).(*modelirspb.Model)
	for _, c := range m.GetChannels() {
		if c.GetName() == "wire" {
			c.Capacity = 40
		}
	}
	_, err := BuildWithin(m, DefaultCeilings)
	var limit *LimitError
	require.ErrorAs(t, err, &limit)
	require.Equal(t, "members", limit.Resource)
	require.Equal(t, int64(math.MaxInt64), limit.Needed, "4^0 + … + 4^40 exceeds what an int64 counts: the count saturates")
}

// A delivery without its one message input is refused where it is bound, not read past its inputs.
func TestADeliveryMovesOneMessage(t *testing.T) {
	m := proto.Clone(lifted(t, "channels")).(*modelirspb.Model)
	for _, a := range m.GetActions() {
		if a.GetName() == "wireDelivery" {
			a.Inputs = nil
		}
	}
	_, err := Build(m)
	require.ErrorContains(t, err, "Channels.scala.fixture:57: wireDelivery moves one message of "+
		"fixture.channels.Channels$package$.wire, and has 0 inputs")
}

func TestChannelSizeCountsWithoutListing(t *testing.T) {
	for _, c := range []struct {
		name      string
		entries   int64
		capacity  int64
		unordered bool
		want      int64
	}{
		{"no entries", 0, 3, false, 1},
		{"one entry", 1, 5, true, 6},
		{"wire: 1 + 4 + 4²", 4, 2, false, 21},
		{"radio at capacity 2: [], [u], [d], [u,u], [u,d], [d,d]", 2, 2, true, 6},
		{"multisets of 200 from 2: C(202, 200)", 2, 200, true, 20301},
		{"2⁶⁴ lists and more", 2, 64, false, math.MaxInt64},
		{"C(200, 100) multisets", 100, 100, true, math.MaxInt64},
		{"entries already saturated", math.MaxInt64, 1, true, math.MaxInt64},
	} {
		t.Run(c.name, func(t *testing.T) {
			require.Equal(t, c.want, channelSize(c.entries, c.capacity, c.unordered))
		})
	}
}
