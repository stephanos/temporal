package model

// The lifted fixtures interpreted: choices, presence, bounded channels and holes, each against rows
// worked out by hand from the fixture's Scala source.

import (
	"math"
	"runtime/debug"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	umpirespb "go.temporal.io/server/api/umpire/v1"
	"google.golang.org/protobuf/proto"
)

func built(t *testing.T, m *umpirespb.Model) map[string]*Machine {
	t.Helper()
	out, err := Build(m)
	require.NoError(t, err)
	return out
}

func row(t *testing.T, mm *Machine, key string) Row {
	t.Helper()
	for _, r := range mm.Table.Rows {
		if r.Key == key {
			return r
		}
	}
	require.Failf(t, "no row", "%s has no row %s", mm.Decl.GetName(), key)
	return Row{}
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
	require.Equal(t, Row{Key: "scheduled-queued-none-attemptStart", Source: "scheduled-queued-none", Action: "attemptStart",
		Results: []Result{
			{Outcome: "accepted", State: "started-empty-one", Facts: facts, Choice: "committed"},
			{Outcome: "accepted", State: "started-redelivery-one", Facts: facts, Because: "the channel may deliver the message again", Choice: "redelivered"},
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
			require.True(t, tr.Steps[1].Fields[1].equal(redelivered))
		}
	}
}

// A refinement that does not hold is the machine's, not the Model's: its table is still built, so a
// check can find the counterexample on it.
func TestARejectedRefinementStaysWithItsMachine(t *testing.T) {
	m := lifted(t, "admission")
	machines := built(t, m)
	_, err := refinementOf(t, m, "currentAdmission")
	require.NoError(t, err)
	_, err = refinementOf(t, m, "staleAdmission")
	require.ErrorContains(t, err, "staleAdmission refines activityProduct: the row "+
		"'paused-queued-none-attemptStart' steps from 'paused-queued-none' to 'started-empty-one', which read as 'paused' and 'started'")
	require.NotEmpty(t, machines["staleAdmission"].Table.Rows)
}

func TestPresenceIsAnOrdinaryEnum(t *testing.T) {
	mm := built(t, lifted(t, "presence"))["presence"]
	// Report: unsent, sent × 2 × 2; kept: None, Some × 2; retries 0..2; polls 0..1.
	require.Len(t, mm.Table.States, 5*3*3*2)
	require.Equal(t, []string{"unsent-None-0-0"}, mm.Table.Starts)
	require.Len(t, mm.Table.Ends, 5*2*3*2)
	require.Equal(t, []Result{{Outcome: "accepted", State: "sent-failed-false-Some-failed-0-0", Facts: []string{}}},
		row(t, mm, "sent-failed-false-None-0-0-keep").Results)
	require.Equal(t, []Result{{Outcome: "ignored", State: "unsent-None-0-0", Facts: []string{}}},
		row(t, mm, "unsent-None-0-0-forget").Results)
	require.Equal(t, []Result{{Outcome: "accepted", State: "unsent-None-0-0", Facts: []string{}}},
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

	m := proto.Clone(lifted(t, "channels")).(*umpirespb.Model)
	for _, c := range m.GetChannels() {
		if c.GetName() == "radio" {
			c.Capacity = 2
		}
	}
	radio, err := NewInterpreter(m).Members(&umpirespb.TypeRef{Ref: &umpirespb.TypeRef_Channel{Channel: "fixture.channels.Channels$package$.radio"}})
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
	require.Equal(t, []Result{{Outcome: "accepted", State: "nothing-[ping-0]-[]", Facts: []string{}}},
		row(t, relay, "nothing-[]-[]-talk-ping").Results)
	require.Equal(t, []Result{{Outcome: "accepted", State: "nothing-[ping-0,pong-0]-[]", Facts: []string{}}},
		row(t, relay, "nothing-[ping-0]-[]-talk-pong").Results, "a FIFO send appends")
	require.True(t, relay.Disabled("nothing-[ping-0,pong-0]-[]", "talk-ping"), "the step refuses a full channel")

	// The receiver's step, and again with the message put back first, its acknowledgment lost.
	require.Equal(t, []Result{
		{Outcome: "accepted", State: "note-[]-[]", Facts: []string{}},
		{Outcome: "accepted", State: "note-[ping-1]-[]", Facts: []string{}, Because: again},
	}, row(t, relay, "nothing-[ping-0]-[]-wireDelivery-ping").Results)
	require.Equal(t, []Result{
		{Outcome: "accepted", State: "note-[pong-0]-[]", Facts: []string{}},
		{Outcome: "accepted", State: "note-[ping-1,pong-0]-[]", Facts: []string{}, Because: again},
	}, row(t, relay, "nothing-[ping-0,pong-0]-[]-wireDelivery-ping").Results)
	require.Equal(t, []Result{{Outcome: "accepted", State: "note-[]-[]", Facts: []string{}}},
		row(t, relay, "nothing-[ping-1]-[]-wireDelivery-ping").Results, "a message delivered its duplicates times is not again")
	require.True(t, relay.Disabled("nothing-[pong-0,ping-0]-[]", "wireDelivery-ping"), "a FIFO channel delivers its first message only")
	require.True(t, relay.Disabled("nothing-[pong-0]-[]", "wireDelivery-pong"), "the receiver does not take pong")
	require.True(t, relay.Disabled("nothing-[]-[]", "wireDelivery-ping"), "an empty channel delivers nothing")

	require.Equal(t, []Result{{Outcome: "accepted", State: "signal-[]-[]", Facts: []string{}}},
		row(t, relay, "nothing-[]-[up-0]-radioDelivery-up").Results)
	require.True(t, relay.Disabled("nothing-[]-[up-0]", "radioDelivery-down"))
	require.Equal(t, []Result{{Outcome: "dropped", State: "nothing-[]-[]", Facts: []string{}}},
		row(t, relay, "nothing-[]-[up-0]-radioLoss-up").Results)
	require.True(t, relay.Disabled("nothing-[]-[up-0]", "radioLoss-down"))

	tallying := machines["tallying"]
	require.Equal(t, []string{"nothing-[]", "nothing-[0-0]", "nothing-[1-0]", "nothing-[2-0]"}, tallying.Table.States[:4])
	require.Equal(t, []string{"count", "tallyDelivery-0", "tallyDelivery-1", "tallyDelivery-2"}, tallying.Table.Actions)
	require.Equal(t, []string{"nothing-[]", "nothing-[2-0]", "note-[]", "note-[2-0]"}, tallying.Table.Reachable)
	require.Equal(t, []Result{{Outcome: "accepted", State: "note-[]", Facts: []string{}}},
		row(t, tallying, "nothing-[2-0]-tallyDelivery-2").Results)
}

func TestUnorderedSendKeepsCatalogOrder(t *testing.T) {
	m := lifted(t, "channels")
	at := &umpirespb.Position{File: "generic", Line: 1}
	signal := func(c string) *umpirespb.Value {
		return &umpirespb.Value{Kind: &umpirespb.Value_Enum{Enum: &umpirespb.EnumValue{Type: "fixture.channels.Signal", Case: c}}}
	}
	held := &umpirespb.Value{Kind: &umpirespb.Value_List{List: &umpirespb.ListValue{Items: []*umpirespb.Value{
		{Kind: &umpirespb.Value_Record{Record: &umpirespb.RecordValue{Type: deliveryType,
			Fields: []*umpirespb.Value{signal("down"), {Kind: &umpirespb.Value_Int{Int: 0}}}}}}}}}}
	send := func(channel string) *umpirespb.Expr {
		return &umpirespb.Expr{Position: at, Kind: &umpirespb.Expr_Inbox{Inbox: &umpirespb.Inbox{Op: umpirespb.Inbox_OP_SEND,
			Channel:  "fixture.channels.Channels$package$." + channel,
			Contents: &umpirespb.Expr{Position: at, Kind: &umpirespb.Expr_Literal{Literal: held}},
			Message:  &umpirespb.Expr{Position: at, Kind: &umpirespb.Expr_Literal{Literal: signal("up")}}}}}
	}
	in := NewInterpreter(m)
	v, err := in.Eval(send("radio"))
	require.NoError(t, err)
	require.Equal(t, "[up-0,down-0]", v.Key(), "unordered: at its catalog position")
	full, err := in.Eval(&umpirespb.Expr{Position: at, Kind: &umpirespb.Expr_Inbox{Inbox: &umpirespb.Inbox{
		Op: umpirespb.Inbox_OP_IS_FULL, Channel: "fixture.channels.Channels$package$.radio",
		Contents: &umpirespb.Expr{Position: at, Kind: &umpirespb.Expr_Literal{Literal: held}}}}})
	require.NoError(t, err)
	require.True(t, full.Bool)
}

// A send the step does not guard lands outside the channel's catalog, and so outside the state domain.
func TestASendToAFullChannelLeavesTheDomain(t *testing.T) {
	m := proto.Clone(lifted(t, "channels")).(*umpirespb.Model)
	talk := function(m, "Channels$package$.talkStep")
	talk.Body = talk.GetBody().GetIf().GetElse()
	_, err := Build(m)
	require.ErrorContains(t, err, "Channels.scala:56: relay: row nothing-[ping-0,ping-0]-[]-talk-ping lands in "+
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
	require.Equal(t, disk.Holes, disk.reachableHoles())
	require.False(t, disk.Disabled("staged", "crash"), "a hole row is not disabled")
	require.True(t, disk.Disabled("empty", "crash"))
	require.True(t, disk.Disabled("durable", "crash"))
	require.False(t, disk.Disabled("empty", "put"), "a row is not disabled")
	require.False(t, disk.Disabled("empty", "nothing"), "no such class")
}

// flushStep without its wildcard case: the stages it no longer matches are undeclared holes.
func TestAnUnmatchedValueIsAnUndeclaredHole(t *testing.T) {
	m := proto.Clone(lifted(t, "declarations")).(*umpirespb.Model)
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
	for _, h := range disk.reachableHoles() {
		reachable = append(reachable, h.Row)
	}
	require.Equal(t, []string{"empty-flush", "staged-crash", "durable-flush"}, reachable)
}

func TestReachableHolesAreOnlyThoseAPathReaches(t *testing.T) {
	m := proto.Clone(lifted(t, "declarations")).(*umpirespb.Model)
	// Without put, nothing leaves empty: the staged crash hole is unreachable.
	for _, mm := range m.GetMachines() {
		if mm.GetName() == "disk" {
			mm.Steps = slicesDelete(mm.GetSteps(), func(b *umpirespb.StepBinding) bool { return strings.HasSuffix(b.GetAction(), ".put") })
		}
	}
	disk := built(t, m)["disk"]
	require.Len(t, disk.Holes, 1)
	require.Empty(t, disk.reachableHoles())
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
	// The reachable crash hole leaves the refinement unknown, not rejected: every row refines.
	rows, err := refinementOf(t, lifted(t, "declarations"), "disk")
	var incomplete *RefinementError
	require.ErrorAs(t, err, &incomplete)
	require.Equal(t, RefinementIncomplete, incomplete.Kind)
	put := "put"
	require.Equal(t, []RefinementRow{{Key: "empty-put", Product: &put}, {Key: "staged-flush"}}, rows)

	for name, visible := range map[string]string{
		"a stutter records a fact the product sees":   "disk.visible",
		"a stutter's outcome is one the product sees": "disk.visibleOutcomes",
	} {
		t.Run(name, func(t *testing.T) {
			m := proto.Clone(lifted(t, "declarations")).(*umpirespb.Model)
			f := function(m, visible)
			f.Body = &umpirespb.Expr{Position: f.GetBody().GetPosition(),
				Kind: &umpirespb.Expr_Literal{Literal: &umpirespb.Value{Kind: &umpirespb.Value_Bool{Bool: true}}}}
			disk := built(t, m)["disk"]
			require.NotEmpty(t, disk.reachableHoles(), "the reachable crash hole does not erase the rejection")
			_, err := refinementOf(t, m, "disk")
			require.ErrorContains(t, err, "disk refines store: the row 'staged-flush'")
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

	unwatched := proto.Clone(m).(*umpirespb.Model)
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
	m := proto.Clone(lifted(t, "presence")).(*umpirespb.Model)
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
	m := proto.Clone(lifted(t, "channels")).(*umpirespb.Model)
	for _, c := range m.GetChannels() {
		if c.GetName() == "wire" {
			c.Capacity = 40
		}
	}
	_, err := BuildWithin(m, defaultCeilings)
	var limit *LimitError
	require.ErrorAs(t, err, &limit)
	require.Equal(t, "members", limit.Resource)
	require.Equal(t, int64(math.MaxInt64), limit.Needed, "4^0 + … + 4^40 exceeds what an int64 counts: the count saturates")
	require.True(t, limit.Overflow)
}

// A delivery without its one message input is refused where it is bound, not read past its inputs.
func TestADeliveryMovesOneMessage(t *testing.T) {
	m := proto.Clone(lifted(t, "channels")).(*umpirespb.Model)
	for _, a := range m.GetActions() {
		if a.GetName() == "wireDelivery" {
			a.Inputs = nil
		}
	}
	_, err := Build(m)
	require.ErrorContains(t, err, "Channels.scala:58: wireDelivery moves one message of "+
		"fixture.channels.Channels$package$.wire, and has 0 inputs")
}

func TestChannelSizeCountsWithoutListing(t *testing.T) {
	for _, c := range []struct {
		name      string
		entries   count
		capacity  int64
		unordered bool
		want      count
	}{
		{"no entries", count{n: 0}, 3, false, count{n: 1}},
		{"one entry", count{n: 1}, 5, true, count{n: 6}},
		{"wire: 1 + 4 + 4²", count{n: 4}, 2, false, count{n: 21}},
		{"radio at capacity 2: [], [u], [d], [u,u], [u,d], [d,d]", count{n: 2}, 2, true, count{n: 6}},
		{"multisets of 200 from 2: C(202, 200)", count{n: 2}, 200, true, count{n: 20301}},
		{"2⁶⁴ lists and more", count{n: 2}, 64, false, overflowed},
		{"C(200, 100) multisets", count{n: 100}, 100, true, overflowed},
		{"entries already at the int64 bound", count{n: math.MaxInt64}, 1, true, overflowed},
		{"entries past int64", overflowed, 1, false, overflowed},
		{"only the empty list of entries past int64", overflowed, 0, false, count{n: 1}},
	} {
		t.Run(c.name, func(t *testing.T) {
			require.Equal(t, c.want, channelSize(c.entries, c.capacity, c.unordered))
		})
	}
}

// Each step result malformed in one way, built without admission: a located error at the binding,
// never a disabled pair or a panic.
func TestAStepResultMustBeStepRecords(t *testing.T) {
	putStep := "fixture.declarations.Declarations$package$.putStep"
	for name, c := range map[string]struct {
		fixture string
		mutate  func(m *umpirespb.Model)
		want    string
	}{
		"not a list": {"declarations", func(m *umpirespb.Model) {
			f := function(m, "Declarations$package$.putStep")
			f.Body = admLiteral(f.GetBody(), &umpirespb.Value{Kind: &umpirespb.Value_Bool{Bool: false}})
		}, "Declarations.scala:103: " + putStep + " returns false, not a list of steps"},
		"a step of two fields": {"declarations", func(m *umpirespb.Model) {
			f := function(m, "Declarations$package$.putStep")
			f.Body = admLiteral(f.GetBody(), &umpirespb.Value{Kind: &umpirespb.Value_List{List: &umpirespb.ListValue{Items: []*umpirespb.Value{
				{Kind: &umpirespb.Value_Record{Record: &umpirespb.RecordValue{Type: StepType, Fields: []*umpirespb.Value{
					admEnum("fixture.declarations.Outcome", "accepted"), admEnum("fixture.declarations.Stage", "staged")}}}}}}}})
		}, "Declarations.scala:103: " + putStep + " returns a step of 2 fields, not 4"},
		"an outcome of another type": {"presence", func(m *umpirespb.Model) {
			f := function(m, "Presence$package$.forgetStep")
			step := f.GetBody().GetMatch().GetCases()[0].GetBody().GetList().GetItems()[0].GetConstruct()
			step.Args[0] = admLiteral(step.GetArgs()[0], admEnum("fixture.presence.Result", "failed"))
		}, "Presence.scala:75: presence: row unsent-Some-succeeded-0-0-forget has outcome failed, which is no fixture.presence.Outcome"},
		"a fact of another type": {"declarations", func(m *umpirespb.Model) {
			f := function(m, "Declarations$package$.putStep")
			facts := f.GetBody().GetMatch().GetCases()[0].GetBody().GetList().GetItems()[0].GetConstruct().GetArgs()[2]
			facts.GetList().Items[0] = admLiteral(facts, admEnum("fixture.declarations.Kept", "held"))
		}, "Declarations.scala:103: disk: row empty-put records held, which is no fixture.declarations.Fact"},
	} {
		t.Run(name, func(t *testing.T) {
			m := proto.Clone(lifted(t, c.fixture)).(*umpirespb.Model)
			c.mutate(m)
			_, err := Build(m)
			require.ErrorContains(t, err, c.want)
		})
	}
}

func TestATransferIsADeliveryOrALossNotBoth(t *testing.T) {
	m := proto.Clone(lifted(t, "channels")).(*umpirespb.Model)
	for _, a := range m.GetActions() {
		if a.GetName() == "radioLoss" {
			a.Delivers = a.GetLoses()
		}
	}
	_, err := Build(m)
	require.ErrorContains(t, err, "Channels.scala:60: radioLoss both delivers and loses fixture.channels.Channels$package$.radio")
}

// keep and forget given an input of 0..59 each: no binding lists more than 60 classes, and together
// with send's two and poll's one they are 123, counted before any is listed.
func TestTheClassesOfAllBindingsAreCountedTogether(t *testing.T) {
	m := proto.Clone(lifted(t, "presence")).(*umpirespb.Model)
	for _, a := range m.GetActions() {
		if a.GetName() == "keep" || a.GetName() == "forget" {
			a.Inputs = []*umpirespb.Param{{Name: "n", Type: &umpirespb.TypeRef{Ref: &umpirespb.TypeRef_IntRange{
				IntRange: &umpirespb.IntRange{High: 59}}}}}
		}
	}
	_, err := BuildWithin(m, Ceilings{Members: 90, Evaluations: 1 << 20})
	var limit *LimitError
	require.ErrorAs(t, err, &limit)
	require.Equal(t, LimitError{Machine: "presence", Resource: "classes", Ceiling: 90, Needed: 123}, *limit)
}

// A count past what an int64 holds overflows every int64 ceiling, the largest included.
func TestACountPastInt64IsRefusedUnderAnyCeiling(t *testing.T) {
	m := proto.Clone(lifted(t, "channels")).(*umpirespb.Model)
	for _, c := range m.GetChannels() {
		if c.GetName() == "wire" {
			c.Capacity = 40
		}
	}
	_, err := BuildWithin(m, Ceilings{Members: math.MaxInt64, Evaluations: math.MaxInt64})
	var limit *LimitError
	require.ErrorAs(t, err, &limit)
	require.Equal(t, LimitError{Machine: "relay", Resource: "members", Ceiling: math.MaxInt64, Needed: math.MaxInt64, Overflow: true}, *limit)
}

// Relay's heard field typed as Relay itself, built without admission: refused, not recursed into.
func TestACatalogThatContainsItselfIsRefused(t *testing.T) {
	defer debug.SetMaxStack(debug.SetMaxStack(64 << 20))
	m := proto.Clone(lifted(t, "channels")).(*umpirespb.Model)
	for _, ty := range m.GetTypes() {
		if ty.GetName() == "fixture.channels.Relay" {
			ty.GetRecord().GetFields()[0].Type = named("fixture.channels.Relay")
		}
	}
	_, err := Build(m)
	require.ErrorContains(t, err, "the catalog of type fixture.channels.Relay contains itself")
}

func TestARangeEndingAtMaxInt64(t *testing.T) {
	in := NewInterpreter(lifted(t, "presence"))
	for _, c := range []struct {
		low, high int64
		want      []int64
	}{
		{math.MaxInt64, math.MaxInt64, []int64{math.MaxInt64}},
		{math.MaxInt64 - 1, math.MaxInt64, []int64{math.MaxInt64 - 1, math.MaxInt64}},
		{math.MinInt64, math.MinInt64 + 1, []int64{math.MinInt64, math.MinInt64 + 1}},
	} {
		members, err := in.Members(&umpirespb.TypeRef{Ref: &umpirespb.TypeRef_IntRange{IntRange: &umpirespb.IntRange{Low: c.low, High: c.high}}})
		require.NoError(t, err)
		var got []int64
		for _, v := range members {
			got = append(got, v.Int)
		}
		require.Equal(t, c.want, got)
	}
}

func TestCountsCarryOverflow(t *testing.T) {
	require.Equal(t, count{}, count{}.times(overflowed), "nothing times anything is nothing")
	require.Equal(t, overflowed, count{n: 2}.times(overflowed))
	require.Equal(t, overflowed, count{n: 1 << 32}.times(count{n: 1 << 31}), "2^63 is past an int64")
	require.Equal(t, count{n: 1 << 62}, count{n: 1 << 31}.times(count{n: 1 << 31}))
	require.Equal(t, overflowed, count{n: math.MaxInt64}.plus(count{n: 1}))
	require.Equal(t, overflowed, count{n: 1}.plus(overflowed))
	require.Equal(t, overflowed, overflowed.plus(count{}), "an overflow plus nothing still overflows")
}

// Members refuses a catalog past its ceiling by itself, as a caller outside Build reads it.
func TestMembersIsBoundedOnItsOwn(t *testing.T) {
	_, err := NewInterpreter(lifted(t, "presence")).Members(&umpirespb.TypeRef{Ref: &umpirespb.TypeRef_IntRange{
		IntRange: &umpirespb.IntRange{High: 1 << 16}}})
	var limit *LimitError
	require.ErrorAs(t, err, &limit)
	require.Equal(t, LimitError{Resource: "members", Ceiling: 1 << 16, Needed: 1<<16 + 1}, *limit)
}

// pollStep's update written as the string "1": its key is a state's, and its value is none.
func TestAStateOfTheRightKeyButAnotherTypeIsOutsideTheDomain(t *testing.T) {
	m := proto.Clone(lifted(t, "presence")).(*umpirespb.Model)
	f := function(m, "Presence$package$.pollStep")
	polls := f.GetBody().GetIf().GetThen().GetList().GetItems()[0].GetConstruct().GetArgs()[1].GetCopy().GetUpdates()[0]
	polls.Value = admLiteral(polls.GetValue(), &umpirespb.Value{Kind: &umpirespb.Value_Text{Text: "1"}})
	_, err := Build(m)
	require.ErrorContains(t, err, "Presence.scala:75: presence: row unsent-None-0-0-poll lands in unsent-None-0-1, "+
		"which is outside the state domain")
}

// ends, visible and visibleOutcomes each made to return 3: a located error, never a state that is no
// end or a fact the product does not see. Build reads ends; the refinement Check reads the two
// others.
func TestEndsAndVisibleMustReturnABoolean(t *testing.T) {
	three := func(at *umpirespb.Expr) *umpirespb.Expr { return admLiteral(at, admIntValue(3)) }
	for name, c := range map[string]struct {
		mutate func(m *umpirespb.Model)
		want   string
	}{
		"ends": {func(m *umpirespb.Model) {
			ends := admMachine(m, "disk").GetEnds().GetLambda()
			ends.Body = three(ends.GetBody())
		}, "disk: ends is 3 at empty, not a Boolean"},
		"visible": {func(m *umpirespb.Model) {
			f := function(m, "disk.visible")
			f.Body = three(f.GetBody())
		}, "disk: disk.visible is 3 for stored, not a Boolean"},
		"visibleOutcomes": {func(m *umpirespb.Model) {
			f := function(m, "disk.visibleOutcomes")
			f.Body = three(f.GetBody())
		}, "disk: disk.visibleOutcomes is 3 for accepted, not a Boolean"},
	} {
		t.Run(name, func(t *testing.T) {
			m := proto.Clone(lifted(t, "declarations")).(*umpirespb.Model)
			c.mutate(m)
			_, err := Build(m)
			if name != "ends" {
				require.NoError(t, err)
				_, err = refinementOf(t, m, "disk")
			}
			var located *Error
			require.ErrorAs(t, err, &located)
			require.NotEmpty(t, located.Position)
			require.Equal(t, c.want, located.Message)
		})
	}
}
