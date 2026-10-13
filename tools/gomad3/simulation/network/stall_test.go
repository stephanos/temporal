package network

import (
	"reflect"
	"testing"
)

func TestModelPartitionHoldsExistingAndNewFIFO(t *testing.T) {
	m, a, b := testModel(t, testLimits())
	_, client, server := pair(t, m, a, b)
	if err := m.Delay(a.Node, b.Node, 10, false); err != nil {
		t.Fatal(err)
	}
	write := func(domain Domain, handle, id uint64, data string) {
		t.Helper()
		if r := m.Dispatch(Request{Domain: domain, Op: "write", Handle: handle, RequestID: id, Bytes: []byte(data)}); r.State != "completed" || r.Error != "" || r.Count != len(data) {
			t.Fatal(r)
		}
	}
	write(a, client, 4, "before")
	if _, err := m.Advance(20); err != nil {
		t.Fatal(err)
	}
	if err := m.Partition(a.Node, b.Node, false); err != nil {
		t.Fatal(err)
	}
	write(a, client, 5, "after")
	write(b, server, 6, "reverse")
	if r := m.Dispatch(Request{Domain: a, Op: "read", Handle: client, RequestID: 7, Count: 32}); string(r.Bytes) != "reverse" {
		t.Fatal(r)
	}
	if r := m.Dispatch(Request{Domain: b, Op: "read", Handle: server, RequestID: 8, Count: 32}); r.State != "pending" {
		t.Fatalf("undelivered bytes crossed disabled link: %+v", r)
	}
	if err := m.Heal(a.Node, b.Node, false); err != nil {
		t.Fatal(err)
	}
	write(a, client, 9, "fresh")
	if _, err := m.Advance(25); err != nil {
		t.Fatal(err)
	}
	if err := m.Heal(a.Node, b.Node, false); err != nil {
		t.Fatal(err)
	}
	if _, err := m.Advance(29); err != nil {
		t.Fatal(err)
	}
	if r := m.Dispatch(Request{Domain: b, Op: "poll", RequestID: 8}); r.State != "pending" {
		t.Fatalf("heal delay bypassed: %+v", r)
	}
	if _, err := m.Advance(30); err != nil {
		t.Fatal(err)
	}
	var received []string
	received = append(received, string(m.Dispatch(Request{Domain: b, Op: "poll", RequestID: 8}).Bytes))
	for id := uint64(10); id < 12; id++ {
		received = append(received, string(m.Dispatch(Request{Domain: b, Op: "read", Handle: server, RequestID: id, Count: 32}).Bytes))
	}
	if !reflect.DeepEqual(received, []string{"before", "after", "fresh"}) {
		t.Fatalf("FIFO release=%q", received)
	}
}

func TestModelReceivedTailSurvivesPartitionAndRepartitionDelaysSuffix(t *testing.T) {
	m, a, b := testModel(t, testLimits())
	_, client, server := pair(t, m, a, b)
	m.Dispatch(Request{Domain: a, Op: "write", Handle: client, RequestID: 4, Bytes: []byte("received")})
	if r := m.Dispatch(Request{Domain: b, Op: "read", Handle: server, RequestID: 5, Count: 2}); string(r.Bytes) != "re" {
		t.Fatal(r)
	}
	if err := m.Delay(a.Node, b.Node, 10, false); err != nil {
		t.Fatal(err)
	}
	m.Dispatch(Request{Domain: a, Op: "write", Handle: client, RequestID: 6, Bytes: []byte("suffix")})
	if err := m.Partition(a.Node, b.Node, false); err != nil {
		t.Fatal(err)
	}
	if r := m.Dispatch(Request{Domain: b, Op: "read", Handle: server, RequestID: 7, Count: 32}); string(r.Bytes) != "ceived" {
		t.Fatalf("received data held: %+v", r)
	}
	if err := m.Heal(a.Node, b.Node, false); err != nil {
		t.Fatal(err)
	}
	if _, err := m.Advance(5); err != nil {
		t.Fatal(err)
	}
	if err := m.Partition(a.Node, b.Node, false); err != nil {
		t.Fatal(err)
	}
	if _, err := m.Advance(20); err != nil {
		t.Fatal(err)
	}
	if err := m.Heal(a.Node, b.Node, false); err != nil {
		t.Fatal(err)
	}
	if r := m.Dispatch(Request{Domain: b, Op: "read", Handle: server, RequestID: 8, Count: 32}); r.State != "pending" || r.Deadline != 30 {
		t.Fatalf("repartition delay=%+v", r)
	}
	if _, err := m.Advance(30); err != nil {
		t.Fatal(err)
	}
	if r := m.Dispatch(Request{Domain: b, Op: "poll", RequestID: 8}); string(r.Bytes) != "suffix" {
		t.Fatal(r)
	}
}

func TestModelHeldAdmissionReservesTerminalHistoryAtomically(t *testing.T) {
	for _, ceiling := range []uint64{5, 6} {
		limits := testLimits()
		limits.Transitions = ceiling
		m, a, b := testModel(t, limits)
		_, client, _ := pair(t, m, a, b)
		if err := m.Partition(a.Node, b.Node, false); err != nil {
			t.Fatal(err)
		}
		before := len(m.network.transitions)
		r := m.Dispatch(Request{Domain: a, Op: "write", Handle: client, RequestID: 4, Bytes: []byte("held")})
		if ceiling == 5 {
			if r.Error != "network resources exhausted" || r.Count != 0 || len(m.network.transitions) != before || m.network.pendingBytes != 0 {
				t.Fatalf("rejected held admission mutated state: %+v", r)
			}
		} else {
			if r.Error != "" || r.Count != 4 {
				t.Fatal(r)
			}
			if err := m.Delay(a.Node, b.Node, 1, false); err != ErrResourceExhausted {
				t.Fatalf("ordinary operation stole terminal slot: %v", err)
			}
			if err := m.Heal(a.Node, b.Node, false); err != nil {
				t.Fatalf("heal failed to release reservation atomically: %v", err)
			}
		}
	}
}

func TestModelPartitionReservationFailureLeavesQueuedBytesAndTopology(t *testing.T) {
	limits := testLimits()
	limits.Transitions = 5
	m, a, b := testModel(t, limits)
	_, client, server := pair(t, m, a, b)
	m.Dispatch(Request{Domain: a, Op: "write", Handle: client, RequestID: 4, Bytes: []byte("queued")})
	if err := m.Partition(a.Node, b.Node, false); err != ErrResourceExhausted {
		t.Fatalf("partition lacked terminal reservation: %v", err)
	}
	if !m.network.links[simulationLinkKey(a.Node, b.Node)].Enabled || len(m.network.transitions) != 4 {
		t.Fatal("failed partition changed topology or history")
	}
	if r := m.Dispatch(Request{Domain: b, Op: "read", Handle: server, RequestID: 5, Count: 32}); string(r.Bytes) != "queued" {
		t.Fatal(r)
	}
}
