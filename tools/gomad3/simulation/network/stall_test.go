package network

import (
	"math"
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

func TestModelAutonomousStallExpiryUsesReservedTerminalSlot(t *testing.T) {
	limits := testLimits()
	limits.StallNanos = 100
	limits.Transitions = 8
	m, a, b := testModel(t, limits)
	_, client, server := pair(t, m, a, b)
	if err := m.Partition(a.Node, b.Node, false); err != nil {
		t.Fatal(err)
	}
	if r := m.Dispatch(Request{Domain: a, Op: "write", Handle: client, RequestID: 4, Bytes: []byte("held")}); r.Count != 4 || r.Error != "" {
		t.Fatal(r)
	}
	for range 2 {
		if err := m.Delay(a.Node, b.Node, 1, false); err != nil {
			t.Fatal(err)
		}
	}
	if err := m.Delay(a.Node, b.Node, 2, false); err != ErrResourceExhausted {
		t.Fatalf("reserved slot stolen: %v", err)
	}
	if len(m.pending) != 0 || m.NextDeadline() != 101 {
		t.Fatalf("returned writer expiry missing: pending=%d next=%d", len(m.pending), m.NextDeadline())
	}
	if _, err := m.Advance(100); err != nil {
		t.Fatal(err)
	}
	if m.network.pendingBytes != 4 {
		t.Fatal("exact limit reset")
	}
	if _, err := m.Advance(101); err != nil {
		t.Fatal(err)
	}
	if m.network.pendingBytes != 0 || len(m.network.transitions) != 8 || m.NextDeadline() != 0 {
		t.Fatal("expiry did not consume reserved slot and free retained bytes")
	}
	for index, h := range []struct {
		domain Domain
		handle uint64
	}{{a, client}, {b, server}} {
		m.Dispatch(Request{Domain: h.domain, Op: "deadline", Handle: h.handle, RequestID: uint64(10 + index), Deadline: 0})
		for _, op := range []string{"read", "write"} {
			r := m.Dispatch(Request{Domain: h.domain, Op: op, Handle: h.handle, RequestID: uint64(20 + index), Count: 1, Bytes: []byte("x")})
			if r.Error != "connection timed out" || r.Count != 0 {
				t.Fatalf("persistent both-end terminal=%+v", r)
			}
		}
	}
}

func TestModelStallOrderingIdleAndEpochInvalidation(t *testing.T) {
	for _, healFirst := range []bool{false, true} {
		limits := testLimits()
		limits.StallNanos = 10
		m, a, b := testModel(t, limits)
		_, client, server := pair(t, m, a, b)
		if err := m.Partition(a.Node, b.Node, false); err != nil {
			t.Fatal(err)
		}
		if m.NextDeadline() != 0 {
			t.Fatal("idle connection armed expiry")
		}
		if _, err := m.Advance(1_000); err != nil {
			t.Fatal(err)
		}
		m.Dispatch(Request{Domain: a, Op: "write", Handle: client, RequestID: 4, Bytes: []byte("held")})
		if _, err := m.Advance(1_010); err != nil {
			t.Fatal(err)
		}
		if err := m.Partition(a.Node, b.Node, false); err != nil {
			t.Fatal(err)
		}
		if m.NextDeadline() != 1_011 {
			t.Fatal("redundant disable restarted interval")
		}
		if healFirst {
			if err := m.Heal(a.Node, b.Node, false); err != nil {
				t.Fatal(err)
			}
		}
		if _, err := m.Advance(1_011); err != nil {
			t.Fatal(err)
		}
		if !healFirst {
			if err := m.Heal(a.Node, b.Node, false); err != nil {
				t.Fatal(err)
			}
		}
		r := m.Dispatch(Request{Domain: b, Op: "read", Handle: server, RequestID: 5, Count: 32})
		if healFirst {
			if string(r.Bytes) != "held" || r.Error != "" {
				t.Fatal(r)
			}
		} else if r.Error != "connection timed out" {
			t.Fatal(r)
		}
	}
	limits := testLimits()
	limits.StallNanos = 10
	m, a, b := testModel(t, limits)
	_, client, _ := pair(t, m, a, b)
	m.Partition(a.Node, b.Node, false)
	m.Dispatch(Request{Domain: a, Op: "write", Handle: client, RequestID: 4, Bytes: []byte("held")})
	m.Advance(5)
	m.Heal(a.Node, b.Node, false)
	m.Partition(a.Node, b.Node, false)
	if m.NextDeadline() != 16 {
		t.Fatalf("replacement epoch deadline=%d", m.NextDeadline())
	}
	m.Advance(11)
	if m.network.pendingBytes != 4 {
		t.Fatal("stale expiry reset replacement epoch")
	}
	m.Advance(16)
	if m.network.pendingBytes != 0 {
		t.Fatal("replacement expiry did not reset")
	}
}

func TestModelStallOverflowRejectsBeforeWriteOrTopologyMutation(t *testing.T) {
	for _, queued := range []bool{false, true} {
		limits := testLimits()
		limits.StallNanos = math.MaxInt64 - 1
		m, a, b := testModel(t, limits)
		_, client, _ := pair(t, m, a, b)
		if queued {
			m.Dispatch(Request{Domain: a, Op: "write", Handle: client, RequestID: 4, Bytes: []byte("queued")})
		}
		if _, err := m.Advance(2); err != nil {
			t.Fatal(err)
		}
		before := len(m.network.transitions)
		if queued {
			if err := m.Partition(a.Node, b.Node, false); err == nil || len(m.network.transitions) != before || !m.network.links[simulationLinkKey(a.Node, b.Node)].Enabled {
				t.Fatal("overflow partition mutated state")
			}
		} else {
			if err := m.Partition(a.Node, b.Node, false); err != nil {
				t.Fatal(err)
			}
			before = len(m.network.transitions)
			if r := m.Dispatch(Request{Domain: a, Op: "write", Handle: client, RequestID: 5, Bytes: []byte("held")}); r.Error == "" || r.Count != 0 || m.network.pendingBytes != 0 || len(m.network.transitions) != before {
				t.Fatalf("overflow write mutated state: %+v", r)
			}
		}
	}
}

func TestModelFullCloseCancelsHeldSuffixButPreservesReceivedTail(t *testing.T) {
	limits := testLimits()
	limits.StallNanos = 10
	m, a, b := testModel(t, limits)
	_, client, server := pair(t, m, a, b)
	m.Dispatch(Request{Domain: a, Op: "write", Handle: client, RequestID: 4, Bytes: []byte("received")})
	if r := m.Dispatch(Request{Domain: b, Op: "read", Handle: server, RequestID: 5, Count: 2}); string(r.Bytes) != "re" {
		t.Fatal(r)
	}
	if err := m.Partition(a.Node, b.Node, false); err != nil {
		t.Fatal(err)
	}
	m.Dispatch(Request{Domain: a, Op: "write", Handle: client, RequestID: 6, Bytes: []byte("held")})
	if r := m.Dispatch(Request{Domain: a, Op: "close", Handle: client, RequestID: 7}); r.Error != "" {
		t.Fatal(r)
	}
	if m.NextDeadline() != 0 || m.network.pendingBytes != 6 {
		t.Fatalf("close retained held suffix/expiry: next=%d bytes=%d", m.NextDeadline(), m.network.pendingBytes)
	}
	if _, err := m.Advance(100); err != nil {
		t.Fatal(err)
	}
	if r := m.Dispatch(Request{Domain: b, Op: "read", Handle: server, RequestID: 8, Count: 32}); string(r.Bytes) != "ceived" || r.Error != "" {
		t.Fatal(r)
	}
	if r := m.Dispatch(Request{Domain: b, Op: "read", Handle: server, RequestID: 9, Count: 32}); r.Error != "EOF" {
		t.Fatal(r)
	}
	if m.network.pendingBytes != 0 {
		t.Fatal("received tail remained charged")
	}
}

func TestModelCloseAndHalfClosePreserveEnabledDelayAndHealedFIFO(t *testing.T) {
	for _, op := range []string{"close", "close_write"} {
		m, a, b := testModel(t, testLimits())
		_, client, server := pair(t, m, a, b)
		if err := m.Delay(a.Node, b.Node, 10, false); err != nil {
			t.Fatal(err)
		}
		m.Dispatch(Request{Domain: a, Op: "write", Handle: client, RequestID: 4, Bytes: []byte("delayed")})
		if r := m.Dispatch(Request{Domain: a, Op: op, Handle: client, RequestID: 5}); r.Error != "" {
			t.Fatal(r)
		}
		if r := m.Dispatch(Request{Domain: b, Op: "read", Handle: server, RequestID: 6, Count: 32}); r.State != "pending" || r.Deadline != 10 {
			t.Fatal(r)
		}
		if _, err := m.Advance(10); err != nil {
			t.Fatal(err)
		}
		if r := m.Dispatch(Request{Domain: b, Op: "poll", RequestID: 6}); string(r.Bytes) != "delayed" {
			t.Fatal(r)
		}
		if r := m.Dispatch(Request{Domain: b, Op: "read", Handle: server, RequestID: 7, Count: 32}); r.Error != "EOF" {
			t.Fatal(r)
		}
	}
	m, a, b := testModel(t, testLimits())
	_, client, server := pair(t, m, a, b)
	if err := m.Partition(a.Node, b.Node, false); err != nil {
		t.Fatal(err)
	}
	m.Dispatch(Request{Domain: a, Op: "write", Handle: client, RequestID: 4, Bytes: []byte("held")})
	if r := m.Dispatch(Request{Domain: a, Op: "close_write", Handle: client, RequestID: 5}); r.Error != "" {
		t.Fatal(r)
	}
	if m.NextDeadline() == 0 {
		t.Fatal("half-close cancelled live held interval")
	}
	if err := m.Heal(a.Node, b.Node, false); err != nil {
		t.Fatal(err)
	}
	if r := m.Dispatch(Request{Domain: b, Op: "read", Handle: server, RequestID: 6, Count: 32}); string(r.Bytes) != "held" {
		t.Fatal(r)
	}
	if r := m.Dispatch(Request{Domain: b, Op: "read", Handle: server, RequestID: 7, Count: 32}); r.Error != "EOF" {
		t.Fatal(r)
	}
}

func TestModelSimultaneousPairExpiryReplay(t *testing.T) {
	limits := testLimits()
	limits.StallNanos = 100
	limits.Transitions = 11
	run := func(replay *Record) Record {
		m, err := New(1, Config{Nodes: []Node{{Node: "a", Address: "a"}, {Node: "b", Address: "b"}}, Links: []Link{{From: "a", To: "b", Enabled: true}, {From: "b", To: "a", Enabled: true}}, Limits: limits, Replay: replay})
		if err != nil {
			t.Fatal(err)
		}
		a := Domain{Run: 1, Node: "a", Address: "a", Incarnation: 1}
		b := Domain{Run: 1, Node: "b", Address: "b", Incarnation: 1}
		for _, domain := range []Domain{a, b} {
			if err := m.Register(domain); err != nil {
				t.Fatal(err)
			}
		}
		_, first, _ := pair(t, m, a, b)
		_, second, _ := pair(t, m, a, b)
		if err := m.Partition(a.Node, b.Node, true); err != nil {
			t.Fatal(err)
		}
		for index, handle := range []uint64{first, second} {
			if r := m.Dispatch(Request{Domain: a, Op: "write", Handle: handle, RequestID: uint64(4 + index), Bytes: []byte("held")}); r.Count != 4 || r.Error != "" {
				t.Fatal(r)
			}
		}
		if m.NextDeadline() != 101 {
			t.Fatal("simultaneous expiry deadline")
		}
		if _, err := m.Advance(101); err != nil {
			t.Fatal(err)
		}
		if m.network.pendingBytes != 0 || m.network.historyReserved != 0 || len(m.network.transitions) != 11 {
			t.Fatal("simultaneous reservations did not settle")
		}
		record, err := m.Record()
		if err != nil {
			t.Fatal(err)
		}
		return record
	}
	first := run(nil)
	for range 8 {
		if replay := run(&first); !reflect.DeepEqual(first, replay) {
			t.Fatal("simultaneous pair expiry replay depends on map order")
		}
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
