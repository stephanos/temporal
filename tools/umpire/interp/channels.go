package interp

// Bounded channels, as model/SEMANTICS.md's Channels section defines them: what a channel
// holds, its catalog, the inbox operations, and the rows of its delivery and loss.

import (
	"slices"

	umpirespb "go.temporal.io/server/api/umpire/v1"
)

const redelivered = "the channel delivers the message again"

func (in *Interpreter) channel(id string, at *umpirespb.Position) (*umpirespb.Channel, error) {
	c, ok := in.channels[id]
	if !ok {
		return nil, ErrorAt(at, "no channel %s", id)
	}
	return c, nil
}

func delivery(message Value, redeliveries int64) Value {
	return Value{Kind: RecordValue, Type: deliveryType, Fields: []Value{message, {Kind: IntValue, Int: redeliveries}}}
}

// entries lists every delivery a channel may hold, in catalog order: each message, and for each its
// redeliveries 0 to the channel's duplicates.
func (in *Interpreter) entries(c *umpirespb.Channel) ([]Value, error) {
	messages, err := in.Members(c.GetMessage())
	if err != nil {
		return nil, err
	}
	var out []Value
	for _, m := range messages {
		for r := int64(0); r <= int64(c.GetDuplicates()); r++ {
			out = append(out, delivery(m, r))
		}
	}
	return out, nil
}

// contents lists everything a channel may hold: the empty list, then every list of one entry, of two,
// up to its capacity, the last entry varying fastest; an unordered channel only the lists whose
// entries are in catalog order.
func (in *Interpreter) contents(c *umpirespb.Channel) ([]Value, error) {
	entries, err := in.entries(c)
	if err != nil {
		return nil, err
	}
	out := []Value{{Kind: ListValue}}
	level := [][]int{{}}
	for range c.GetCapacity() {
		var next [][]int
		for _, prefix := range level {
			for i := range entries {
				if c.GetOrder() == umpirespb.Channel_ORDER_UNORDERED && len(prefix) > 0 && i < prefix[len(prefix)-1] {
					continue
				}
				next = append(next, append(slices.Clone(prefix), i))
			}
		}
		for _, list := range next {
			items := make([]Value, len(list))
			for j, i := range list {
				items[j] = entries[i]
			}
			out = append(out, Value{Kind: ListValue, Items: items})
		}
		level = next
	}
	return out, nil
}

// put adds a delivery to what a channel holds: first or last for a FIFO channel, at its catalog
// position for an unordered one.
func (in *Interpreter) put(c *umpirespb.Channel, held []Value, d Value, first bool) ([]Value, error) {
	out := slices.Clone(held)
	if c.GetOrder() != umpirespb.Channel_ORDER_UNORDERED {
		if first {
			return slices.Insert(out, 0, d), nil
		}
		return append(out, d), nil
	}
	entries, err := in.entries(c)
	if err != nil {
		return nil, err
	}
	index := func(v Value) int { return slices.IndexFunc(entries, v.Equal) }
	at := slices.IndexFunc(out, func(h Value) bool { return index(h) > index(d) })
	if at < 0 {
		at = len(out)
	}
	return slices.Insert(out, at, d), nil
}

func (in *Interpreter) inbox(x *umpirespb.Expr, b *umpirespb.Inbox, e *env) (Value, error) {
	c, err := in.channel(b.GetChannel(), x.GetPosition())
	if err != nil {
		return Value{}, err
	}
	held, err := in.eval(b.GetContents(), e)
	if err != nil {
		return Value{}, err
	}
	switch b.GetOp() {
	case umpirespb.Inbox_OP_IS_EMPTY:
		return Value{Kind: BoolValue, Bool: len(held.Items) == 0}, nil
	case umpirespb.Inbox_OP_IS_FULL:
		return Value{Kind: BoolValue, Bool: len(held.Items) >= int(c.GetCapacity())}, nil
	case umpirespb.Inbox_OP_SEND:
		m, err := in.eval(b.GetMessage(), e)
		if err != nil {
			return Value{}, err
		}
		items, err := in.put(c, held.Items, delivery(m, 0), false)
		return Value{Kind: ListValue, Items: items}, err
	default:
		return Value{}, ErrorAt(x.GetPosition(), "unknown inbox operator %v", b.GetOp())
	}
}

// BothRoles says that an action both delivers and loses a channel's messages, which no action does.
func BothRoles(a *umpirespb.Action) string {
	if a.GetDelivers() == a.GetLoses() {
		return "both delivers and loses " + a.GetDelivers()
	}
	return "both delivers " + a.GetDelivers() + " and loses " + a.GetLoses()
}

// holding is the field of a machine's state that holds a channel.
func (in *Interpreter) holding(decl *umpirespb.Machine, channel string, at *umpirespb.Position) (int, error) {
	for i, f := range in.types[decl.GetStateType()].GetRecord().GetFields() {
		if f.GetType().GetChannel() == channel {
			return i, nil
		}
	}
	return 0, ErrorAt(at, "%s: the state %s holds no channel %s", decl.GetName(), decl.GetStateType(), channel)
}

// transfer is the value of a channel's delivery or loss of message m at state s: the bound function's
// steps at s with the entry taken out, and, for a delivery the channel may duplicate, those steps
// again with the entry put back, its acknowledgment lost. No entry to take out disables the pair.
func (in *Interpreter) transfer(decl *umpirespb.Machine, s Value, c Class) ([]Value, error) {
	a := c.Action
	if a.GetDelivers() != "" && a.GetLoses() != "" {
		return nil, ErrorAt(c.at, "%s %s", a.GetName(), BothRoles(a))
	}
	id, delivers := a.GetDelivers(), true
	if id == "" {
		id, delivers = a.GetLoses(), false
	}
	ch, err := in.channel(id, c.at)
	if err != nil {
		return nil, err
	}
	f, err := in.holding(decl, id, c.at)
	if err != nil {
		return nil, err
	}
	if len(c.Inputs) != 1 {
		return nil, ErrorAt(c.at, "%s moves one message of %s, and has %d inputs", a.GetName(), id, len(c.Inputs))
	}
	m := c.Inputs[0]
	held := s.Fields[f].Items
	taken := slices.IndexFunc(held, func(d Value) bool { return d.Fields[0].Equal(m) })
	if delivers && ch.GetOrder() != umpirespb.Channel_ORDER_UNORDERED && taken > 0 {
		taken = -1
	}
	if taken < 0 {
		return nil, nil
	}
	rest := s
	rest.Fields = slices.Clone(s.Fields)
	rest.Fields[f] = Value{Kind: ListValue, Items: slices.Delete(slices.Clone(held), taken, taken+1)}
	result, err := in.Call(c.step, []Value{rest, m}, c.at)
	if err != nil {
		return nil, err
	}
	steps, err := in.stepList(decl, c, result)
	if err != nil {
		return nil, err
	}
	if r := held[taken].Fields[1].Int; delivers && len(steps) > 0 && r < int64(ch.GetDuplicates()) {
		for _, step := range result.Items {
			next := step.Fields[1]
			if len(next.Fields) <= f || next.Fields[f].Kind != ListValue {
				return nil, ErrorAt(c.at, "%s returns a step to %s, which does not hold %s", c.step, next.Key(), id)
			}
			next.Fields = slices.Clone(next.Fields)
			items, err := in.put(ch, next.Fields[f].Items, delivery(m, r+1), true)
			if err != nil {
				return nil, err
			}
			next.Fields[f] = Value{Kind: ListValue, Items: items}
			again := step
			again.Fields = slices.Clone(step.Fields)
			again.Fields[1] = next
			again.Fields[3] = Value{Kind: TextValue, Text: redelivered}
			// The channel's redelivery is no alternative the receiver named, as its explanation is not.
			again.Choice = ""
			steps = append(steps, again)
		}
	}
	return steps, nil
}
