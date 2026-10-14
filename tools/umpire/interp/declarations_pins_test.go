package interp

// The claims of the Scala framework's own declaration tests that no other test here states, asserted
// over the lifted fixtures: an optional value's catalog, what a channel holds, and what a refinement's
// visible projection reads. .plans/umpire-scala-evaluator-audit.md maps each to the test it replaces.

import (
	"testing"

	"github.com/stretchr/testify/require"
	umpirespb "go.temporal.io/server/api/umpire/v1"
	"google.golang.org/protobuf/proto"
)

func memberKeys(t *testing.T, m *umpirespb.Model, ref *umpirespb.TypeRef) []string {
	t.Helper()
	members, err := NewInterpreter(m).Members(ref)
	require.NoError(t, err)
	keys := make([]string, len(members))
	for i, v := range members {
		keys[i] = v.Key()
	}
	return keys
}

func TestAnOptionalValueIsNoneThenSomeOfEachMember(t *testing.T) {
	require.Equal(t, []string{"None", "Some-succeeded", "Some-failed"},
		memberKeys(t, readLifted(t, "presence"), &umpirespb.TypeRef{Ref: &umpirespb.TypeRef_Named{Named: "scala.Option[fixture.presence.Result]"}}))
}

// inbox is a channel of the channels fixture as it holds the messages sent to it in this order. It
// builds only the probe expression by hand; the Model it is evaluated in comes from the IR.
func inbox(channel string, op umpirespb.Inbox_Op, messages ...*umpirespb.Value) *umpirespb.Expr {
	at := &umpirespb.Position{File: "generic", Line: 1}
	held := &umpirespb.Expr{Position: at, Kind: &umpirespb.Expr_Literal{Literal: &umpirespb.Value{Kind: &umpirespb.Value_List{List: &umpirespb.ListValue{}}}}}
	for _, message := range messages {
		held = &umpirespb.Expr{Position: at, Kind: &umpirespb.Expr_Inbox{Inbox: &umpirespb.Inbox{Op: umpirespb.Inbox_OP_SEND,
			Channel: "fixture.channels." + channel, Contents: held,
			Message: &umpirespb.Expr{Position: at, Kind: &umpirespb.Expr_Literal{Literal: message}}}}}
	}
	if op == umpirespb.Inbox_OP_SEND {
		return held
	}
	return &umpirespb.Expr{Position: at, Kind: &umpirespb.Expr_Inbox{Inbox: &umpirespb.Inbox{Op: op,
		Channel: "fixture.channels." + channel, Contents: held}}}
}

func TestAFIFOChannelHoldsEverySequenceUpToItsCapacityInSendOrder(t *testing.T) {
	m := readLifted(t, "channels")
	wire := memberKeys(t, m, &umpirespb.TypeRef{Ref: &umpirespb.TypeRef_Channel{Channel: "fixture.channels.wire"}})
	// Capacity two, two notes, each held once or redelivered once: 1 + 4 + 4².
	require.Len(t, wire, 1+4+16)
	require.Equal(t, []string{"[]", "[ping-0]", "[ping-1]", "[pong-0]", "[pong-1]", "[ping-0,ping-0]"}, wire[:6])

	note := func(c string) *umpirespb.Value { return admEnum("fixture.channels.Note", c) }
	sent, err := NewInterpreter(m).Eval(inbox("wire", umpirespb.Inbox_OP_SEND, note("pong"), note("ping")))
	require.NoError(t, err)
	require.Equal(t, "[pong-0,ping-0]", sent.Key(), "a FIFO channel keeps send order, not catalog order")
}

func TestAnUnorderedChannelHoldsEachMultisetOnce(t *testing.T) {
	m := proto.Clone(readLifted(t, "channels")).(*umpirespb.Model)
	for _, c := range m.GetChannels() {
		if c.GetName() == "radio" {
			c.Capacity = 2
		}
	}
	signal := func(c string) *umpirespb.Value { return admEnum("fixture.channels.Signal", c) }
	in := NewInterpreter(m)
	eval := func(x *umpirespb.Expr) Value {
		v, err := in.Eval(x)
		require.NoError(t, err)
		return v
	}
	upThenDown := eval(inbox("radio", umpirespb.Inbox_OP_SEND, signal("up"), signal("down")))
	downThenUp := eval(inbox("radio", umpirespb.Inbox_OP_SEND, signal("down"), signal("up")))
	require.Equal(t, "[up-0,down-0]", upThenDown.Key())
	require.Equal(t, upThenDown.Key(), downThenUp.Key(), "the same messages are the same value whatever order they were sent in")
	require.True(t, eval(inbox("radio", umpirespb.Inbox_OP_IS_FULL, signal("down"), signal("down"))).Bool)
	require.False(t, eval(inbox("radio", umpirespb.Inbox_OP_IS_FULL, signal("down"))).Bool)
}
