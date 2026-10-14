package wasi

import (
	"encoding/binary"
	"fmt"
	"reflect"
	"strconv"
	"testing"

	netmodel "go.temporal.io/server/tools/gomad/internal/model/network"
)

func reviewStreamPair(t *testing.T, e *Environment) (uint64, uint64) {
	t.Helper()
	l := streamCall(t, e, streamFrame(1, 1, 0, 0, 0, 0, nil))
	c := streamCall(t, e, streamFrame(2, 2, 0, 0, binary.BigEndian.Uint32(l[48:]), 0, nil))
	s := streamCall(t, e, streamFrame(3, 3, binary.BigEndian.Uint64(l[16:]), 0, 0, 0, nil))
	return binary.BigEndian.Uint64(c[16:]), binary.BigEndian.Uint64(s[16:])
}

func TestNetworkV2StallTimeoutWireClassification(t *testing.T) {
	config := streamConfig()
	config.Limits.StallNanos = 10
	e := streamEnvironment(t, config)
	client, server := reviewStreamPair(t, e)
	if err := e.networkV2.model.Partition(e.networkV2.client.Node, e.networkV2.server.Node, true); err != nil {
		t.Fatal(err)
	}
	streamCall(t, e, streamFrame(5, 4, client, 0, 0, 1, []byte("x")))
	e.now = 12
	for i, op := range []uint32{4, 5} {
		handle := server
		if op == 5 {
			handle = client
		}
		r := streamCall(t, e, streamFrame(op, uint64(5+i), handle, 0, 0, 1, []byte("x")))
		if got := binary.BigEndian.Uint32(r[44:]); got != 65558 {
			t.Fatalf("terminal stall status=%d want=65558", got)
		}
	}
	if got := networkV2Errno("i/o timeout"); got != 73 {
		t.Fatalf("caller deadline status=%d want=73", got)
	}
}

func TestNetworkV2PendingDialTimeoutAndCallerDeadlineStatuses(t *testing.T) {
	for _, deadline := range []uint64{0, 5} {
		t.Run(strconv.FormatUint(deadline, 10), func(t *testing.T) {
			config := streamConfig()
			config.Limits.StallNanos = 10
			e := streamEnvironment(t, config)
			l := streamCall(t, e, streamFrame(1, 1, 0, 0, 0, 0, nil))
			if err := e.networkV2.model.Partition(e.networkV2.client.Node, e.networkV2.server.Node, true); err != nil {
				t.Fatal(err)
			}
			r := streamCall(t, e, streamFrame(2, 2, 0, deadline, binary.BigEndian.Uint32(l[48:]), 0, nil))
			if binary.BigEndian.Uint32(r[40:]) != 1 || binary.BigEndian.Uint64(r[16:]) != 0 {
				t.Fatal("disabled-link dial did not wait without allocating a connection")
			}
			e.now = 12
			want := uint32(65558)
			if deadline != 0 {
				e.now, want = deadline, 73
			}
			r = streamCall(t, e, streamFrame(12, 2, 0, 0, 0, 0, nil))
			if got := binary.BigEndian.Uint32(r[44:]); got != want {
				t.Fatalf("dial timeout status=%d want=%d", got, want)
			}
			if err := e.networkV2.model.Heal(e.networkV2.client.Node, e.networkV2.server.Node, true); err != nil {
				t.Fatal(err)
			}
			r = streamCall(t, e, streamFrame(2, 3, 0, 0, binary.BigEndian.Uint32(l[48:]), 0, nil))
			if binary.BigEndian.Uint32(r[44:]) != 0 || binary.BigEndian.Uint64(r[16:]) == 0 {
				t.Fatal("listener failed to accept a fresh post-timeout dial")
			}
		})
	}
}

func TestNetworkV2ClockPollLeapPreservesEarlierDialDeadline(t *testing.T) {
	for _, scheduledHeal := range []bool{false, true} {
		t.Run(strconv.FormatBool(scheduledHeal), func(t *testing.T) {
			config := streamConfig()
			config.Limits.StallNanos = 10
			if scheduledHeal {
				config.Faults = []NetworkFaultV2{{Name: "heal", Action: "heal", AtNanos: 100}}
			}
			e := streamEnvironment(t, config)
			l := streamCall(t, e, streamFrame(1, 1, 0, 0, 0, 0, nil))
			if err := e.networkV2.model.Partition(e.networkV2.client.Node, e.networkV2.server.Node, true); err != nil {
				t.Fatal(err)
			}
			r := streamCall(t, e, streamFrame(2, 2, 0, 5, binary.BigEndian.Uint32(l[48:]), 0, nil))
			if binary.BigEndian.Uint32(r[40:]) != 1 {
				t.Fatal("disabled-link dial did not wait")
			}
			clock := call(t, e, "poll_oneoff", `{"subscriptions":[{"userdata":1,"type":0,"clock_id":1,"timeout":100,"precision":0,"flags":1}]}`)
			if clock.Errno != 0 || e.now != 100 {
				t.Fatalf("cooperative poll leap failed: errno=%d now=%d", clock.Errno, e.now)
			}
			r = streamCall(t, e, streamFrame(12, 2, 0, 0, 0, 0, nil))
			if got := binary.BigEndian.Uint32(r[44:]); got != 73 {
				t.Fatalf("later stall overtook earlier caller deadline: status=%d want=73", got)
			}
		})
	}
}

func TestBrokerLeapPreservesEarlierDialDeadline(t *testing.T) {
	for _, scheduledHeal := range []bool{false, true} {
		t.Run(strconv.FormatBool(scheduledHeal), func(t *testing.T) {
			config := brokerTestConfig()
			config.Network = streamConfig()
			config.Network.Limits.StallNanos = 10
			if scheduledHeal {
				config.Faults = []BrokerNetworkFault{{Name: "heal", Action: "heal", From: "a", To: "b", AtNanos: 100, Symmetric: true}}
			}
			b, err := NewBroker(config)
			if err != nil {
				t.Fatal(err)
			}
			client, _ := b.network.domains(b.Session("a"))
			_, server := b.network.domains(b.Session("b"))
			m := b.network.model
			l := m.Dispatch(netmodel.Request{Domain: server, Op: "listen", RequestID: 1})
			if l.Error != "" {
				t.Fatal(l)
			}
			if err := b.ChangeLink("partition", "a", "b", 0, true); err != nil {
				t.Fatal(err)
			}
			if r := m.Dispatch(netmodel.Request{Domain: client, Op: "dial", Host: server.Address, Port: l.Local.Port, Deadline: 5, RequestID: 2}); r.State != "pending" {
				t.Fatal(r)
			}
			if err := b.network.advance(100); err != nil {
				t.Fatal(err)
			}
			if r := m.Dispatch(netmodel.Request{Domain: client, Op: "poll", RequestID: 2}); r.Error != "context deadline exceeded" {
				t.Fatalf("later stall overtook earlier caller deadline: %+v", r)
			}
		})
	}
}

func TestNetworkV2ScheduledPartitionUsesFaultTime(t *testing.T) {
	config := streamConfig()
	config.Limits.StallNanos = 10
	config.Faults = []NetworkFaultV2{{Name: "partition", Action: "partition", AtNanos: 100}}
	e := streamEnvironment(t, config)
	client, _ := reviewStreamPair(t, e)
	if err := e.networkV2.model.Delay(e.networkV2.client.Node, e.networkV2.server.Node, 200, false); err != nil {
		t.Fatal(err)
	}
	streamCall(t, e, streamFrame(5, 4, client, 0, 0, 1, []byte("x")))
	for _, now := range []uint64{100, 110, 111} {
		if err := e.networkV2Advance(now); err != nil {
			t.Fatal(err)
		}
		snapshot, err := e.networkV2.model.Snapshot()
		if err != nil {
			t.Fatal(err)
		}
		if got := snapshot.Connections[0].Reset; got != (now == 111) {
			t.Fatalf("at=%d reset=%v want=%v", now, got, now == 111)
		}
	}
}

func TestNetworkV2ScheduledHealDelayAndExpiryOrdering(t *testing.T) {
	for _, healAt := range []uint64{100, 111, 112} {
		t.Run(strconv.FormatUint(healAt, 10), func(t *testing.T) {
			config := streamConfig()
			config.Limits.StallNanos = 10
			config.Faults = []NetworkFaultV2{{Name: "partition", Action: "partition", AtNanos: 100}, {Name: "heal", Action: "heal", AtNanos: healAt}}
			e := streamEnvironment(t, config)
			client, server := reviewStreamPair(t, e)
			if err := e.networkV2.model.Delay(e.networkV2.client.Node, e.networkV2.server.Node, 200, false); err != nil {
				t.Fatal(err)
			}
			streamCall(t, e, streamFrame(5, 4, client, 0, 0, 1, []byte("x")))
			if err := e.networkV2Advance(healAt); err != nil {
				t.Fatal(err)
			}
			r := e.networkV2.model.Dispatch(netmodel.Request{Domain: e.networkV2.server, Op: "read", Handle: server, RequestID: 5, Count: 1})
			if healAt == 112 {
				if r.Error != "connection timed out" {
					t.Fatalf("earlier expiry erased by later heal: %+v", r)
				}
				return
			}
			if r.State != "pending" {
				t.Fatalf("heal bypassed fresh delay: %+v", r)
			}
			if _, err := e.networkV2.model.Advance(healAt + 199); err != nil {
				t.Fatal(err)
			}
			if r := e.networkV2.model.Dispatch(netmodel.Request{Domain: e.networkV2.server, Op: "poll", RequestID: 5}); r.State != "pending" {
				t.Fatalf("released early: %+v", r)
			}
			if _, err := e.networkV2.model.Advance(healAt + 200); err != nil {
				t.Fatal(err)
			}
			if r := e.networkV2.model.Dispatch(netmodel.Request{Domain: e.networkV2.server, Op: "poll", RequestID: 5}); string(r.Bytes) != "x" || r.Error != "" {
				t.Fatalf("equal-time heal did not preserve bytes: %+v", r)
			}
		})
	}
}

func TestBrokerLinkGroupRejectionIsAtomic(t *testing.T) {
	for _, scheduled := range []bool{false, true} {
		for _, action := range []string{"partition", "heal", "delay"} {
			t.Run(fmt.Sprintf("%s/scheduled=%v", action, scheduled), func(t *testing.T) {
				config := brokerTestConfig()
				config.Network = streamConfig()
				config.Network.Limits.Transitions = 1
				if scheduled {
					config.Faults = []BrokerNetworkFault{{Name: "change", Action: action, From: "a", To: "b", AtNanos: 100, DelayNanos: 10, Symmetric: true}}
				}
				b, err := NewBroker(config)
				if err != nil {
					t.Fatal(err)
				}
				before, err := b.NetworkRecord()
				if err != nil {
					t.Fatal(err)
				}
				if scheduled {
					err = b.network.advance(100)
				} else {
					err = b.ChangeLink(action, "a", "b", 10, true)
				}
				if err != netmodel.ErrResourceExhausted {
					t.Fatalf("%s scheduled=%v capacity error=%v", action, scheduled, err)
				}
				after, err := b.NetworkRecord()
				if err != nil {
					t.Fatal(err)
				}
				if !reflect.DeepEqual(before, after) {
					t.Fatalf("%s scheduled=%v rejected group changed links or history", action, scheduled)
				}
			})
		}
	}
}

func TestBrokerDirectionalGroupChangesPreserveReverseLinks(t *testing.T) {
	config := brokerTestConfig()
	config.Network = streamConfig()
	b, err := NewBroker(config)
	if err != nil {
		t.Fatal(err)
	}
	for _, action := range []string{"partition", "delay", "heal"} {
		if err := b.ChangeLink(action, "a", "b", 10, false); err != nil {
			t.Fatal(err)
		}
		snapshot, err := b.NetworkSnapshot()
		if err != nil {
			t.Fatal(err)
		}
		var changed, reverse int
		for _, link := range snapshot.Links {
			if link.From[0] == 'a' && link.To[0] == 'b' {
				changed++
				if link.Enabled != (action == "heal") || action != "partition" && link.DelayNanos != 10 {
					t.Fatalf("directional %s changed wrong state: %+v", action, link)
				}
			}
			if link.From[0] == 'b' && link.To[0] == 'a' {
				reverse++
				if !link.Enabled || link.DelayNanos != 0 {
					t.Fatalf("directional %s changed reverse link: %+v", action, link)
				}
			}
		}
		if changed != 4 || reverse != 4 {
			t.Fatalf("logical link expansion changed=%d reverse=%d", changed, reverse)
		}
	}
}

func reviewBrokerPair(t *testing.T, b *Broker) (netmodel.Domain, netmodel.Domain, uint64, uint64) {
	t.Helper()
	client, _ := b.network.domains(b.Session("a"))
	_, server := b.network.domains(b.Session("b"))
	m := b.network.model
	l := m.Dispatch(netmodel.Request{Domain: server, Op: "listen", RequestID: 1})
	c := m.Dispatch(netmodel.Request{Domain: client, Op: "dial", Host: server.Address, Port: l.Local.Port, RequestID: 2})
	s := m.Dispatch(netmodel.Request{Domain: server, Op: "accept", Handle: l.Handle, RequestID: 3})
	if l.Error != "" || c.Error != "" || s.Error != "" || c.Handle == 0 || s.Handle == 0 {
		t.Fatalf("broker pair: listen=%+v dial=%+v accept=%+v", l, c, s)
	}
	return client, server, c.Handle, s.Handle
}

func TestBrokerLinkGroupReservationRejectionAndExpiry(t *testing.T) {
	for _, transitions := range []uint64{12, 13} {
		t.Run(strconv.FormatUint(transitions, 10), func(t *testing.T) {
			config := brokerTestConfig()
			config.Network = streamConfig()
			config.Network.Limits.Transitions, config.Network.Limits.StallNanos = transitions, 10
			b, err := NewBroker(config)
			if err != nil {
				t.Fatal(err)
			}
			client, _, handle, _ := reviewBrokerPair(t, b)
			if err := b.ChangeLink("delay", "a", "b", 200, true); err != nil {
				t.Fatal(err)
			}
			m := b.network.model
			if r := m.Dispatch(netmodel.Request{Domain: client, Op: "write", Handle: handle, RequestID: 4, Bytes: []byte("held")}); r.Error != "" || r.Count != 4 {
				t.Fatal(r)
			}
			before, err := b.NetworkRecord()
			if err != nil {
				t.Fatal(err)
			}
			err = b.ChangeLink("partition", "a", "b", 0, true)
			if transitions == 12 {
				if err != netmodel.ErrResourceExhausted {
					t.Fatalf("reservation admission error=%v", err)
				}
				after, err := b.NetworkRecord()
				if err != nil || !reflect.DeepEqual(before, after) || m.NextDeadline() != 0 {
					t.Fatalf("rejection mutated links, bytes, history or expiry: %v", err)
				}
				if err := b.ChangeLink("delay", "a", "b", 200, true); err != nil {
					t.Fatalf("rejected group leaked reservation: %v", err)
				}
				return
			}
			if err != nil || m.NextDeadline() != 12 {
				t.Fatalf("group failed to reserve pair expiry: %v next=%d", err, m.NextDeadline())
			}
			if err := b.ChangeLink("delay", "a", "b", 1, true); err != netmodel.ErrResourceExhausted {
				t.Fatalf("competing action consumed terminal capacity: %v", err)
			}
			if err := b.network.advance(12); err != nil {
				t.Fatal(err)
			}
			record, err := b.NetworkRecord()
			if err != nil || len(record.Transitions) != 13 || !record.Snapshot.Connections[0].Reset {
				t.Fatalf("reserved expiry failed: %v %+v", err, record)
			}
		})
	}
}

func TestBrokerScheduledPartitionAndHealUseFaultTime(t *testing.T) {
	for _, healAt := range []uint64{0, 111, 112} {
		t.Run(strconv.FormatUint(healAt, 10), func(t *testing.T) {
			config := brokerTestConfig()
			config.Network = streamConfig()
			config.Network.Limits.StallNanos = 10
			config.Faults = []BrokerNetworkFault{{Name: "partition", Action: "partition", From: "a", To: "b", AtNanos: 100, Symmetric: true}}
			if healAt != 0 {
				config.Faults = append(config.Faults, BrokerNetworkFault{Name: "heal", Action: "heal", From: "a", To: "b", AtNanos: healAt, Symmetric: true})
			}
			b, err := NewBroker(config)
			if err != nil {
				t.Fatal(err)
			}
			client, server, c, s := reviewBrokerPair(t, b)
			if err := b.ChangeLink("delay", "a", "b", 200, true); err != nil {
				t.Fatal(err)
			}
			m := b.network.model
			if r := m.Dispatch(netmodel.Request{Domain: client, Op: "write", Handle: c, RequestID: 4, Bytes: []byte("x")}); r.Error != "" || r.Count != 1 {
				t.Fatal(r)
			}
			if err := b.network.advance(max(110, healAt)); err != nil {
				t.Fatal(err)
			}
			r := m.Dispatch(netmodel.Request{Domain: server, Op: "read", Handle: s, RequestID: 5, Count: 1})
			if healAt == 112 {
				if r.Error != "connection timed out" {
					t.Fatalf("later heal erased earlier expiry: %+v", r)
				}
				return
			}
			if r.State != "pending" {
				t.Fatalf("partition or heal used stale time: %+v", r)
			}
			if healAt == 0 {
				if m.NextDeadline() != 111 {
					t.Fatalf("partition expiry=%d want=111", m.NextDeadline())
				}
				return
			}
			if err := b.network.advance(310); err != nil {
				t.Fatal(err)
			}
			if r := m.Dispatch(netmodel.Request{Domain: server, Op: "poll", RequestID: 5}); r.State != "pending" {
				t.Fatalf("heal delay released early: %+v", r)
			}
			if err := b.network.advance(311); err != nil {
				t.Fatal(err)
			}
			if r := m.Dispatch(netmodel.Request{Domain: server, Op: "poll", RequestID: 5}); string(r.Bytes) != "x" || r.Error != "" {
				t.Fatalf("equal-time heal lost byte stream: %+v", r)
			}
		})
	}
}
