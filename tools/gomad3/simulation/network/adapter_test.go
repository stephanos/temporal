package network

import (
	"bytes"
	"context"
	"io"
	"os"
	"reflect"
	"syscall"
	"testing"
)

type callbackError struct{ calls *int }

func (e callbackError) Error() string { *e.calls++; return "arbitrary callback" }

func TestRequestErrorSerializationIsClosedAndPreservesMessages(t *testing.T) {
	if got := modelErrorMessage(syscall.ESTALE); got != "stale file handle" {
		t.Fatalf("stale handle vocabulary changed: %q", got)
	}
	for _, err := range []error{context.Canceled, context.DeadlineExceeded, os.ErrDeadlineExceeded, io.EOF, ErrAddressInUse, ErrClosed, ErrConnectionRefused, ErrResourceExhausted, ErrUnsupported, ErrReplayDiverged} {
		if got, want := modelErrorMessage(err), err.Error(); got != want {
			t.Fatalf("error message changed: got=%q want=%q", got, want)
		}
	}
	calls := 0
	if got := modelErrorMessage(callbackError{calls: &calls}); got != "network model error" || calls != 0 {
		t.Fatalf("unowned callback invoked: got=%q calls=%d", got, calls)
	}
	if got := modelErrorMessage(&simulationBridgeError{Kind: "capacity", Message: "detached capacity"}); got != "detached capacity" {
		t.Fatalf("structured native message changed: %q", got)
	}
}

func TestProjectedWaitBoundaryPreservesUnrelatedNativePanic(t *testing.T) {
	m, _, b := testModel(t, testLimits())
	close(m.network.changed)
	defer func() {
		if recovered := recover(); recovered == nil {
			t.Fatal("native double-close panic was swallowed or converted to a pending result")
		}
	}()
	m.Dispatch(Request{Domain: b, Op: "listen", RequestID: 1})
}

func testModel(t *testing.T, limits Limits) (*Model, Domain, Domain) {
	t.Helper()
	m, err := New(1, Config{Nodes: []Node{{Node: "a", Address: "a"}, {Node: "b", Address: "b"}}, Links: []Link{{From: "a", To: "b", Enabled: true}, {From: "b", To: "a", Enabled: true}}, Limits: limits})
	if err != nil {
		t.Fatal(err)
	}
	a := Domain{Run: 1, Node: "a", Address: "a", Incarnation: 1}
	b := Domain{Run: 1, Node: "b", Address: "b", Incarnation: 1}
	for _, d := range []Domain{a, b} {
		if err = m.Register(d); err != nil {
			t.Fatal(err)
		}
	}
	return m, a, b
}
func testLimits() Limits {
	return Limits{Listeners: 128, Connections: 128, Deliveries: 128, Bytes: 8 << 20, Transitions: 8192}
}
func pair(t *testing.T, m *Model, a, b Domain) (uint64, uint64, uint64) {
	t.Helper()
	l := m.Dispatch(Request{Domain: b, Op: "listen", RequestID: 1})
	c := m.Dispatch(Request{Domain: a, Op: "dial", RequestID: 2, Host: b.Address, Port: l.Local.Port})
	s := m.Dispatch(Request{Domain: b, Op: "accept", RequestID: 3, Handle: l.Handle})
	for _, r := range []Result{l, c, s} {
		if r.Error != "" || r.State != "completed" {
			t.Fatalf("pair: %+v", r)
		}
	}
	return l.Handle, c.Handle, s.Handle
}

func TestStreamDelayDrainHalfClose(t *testing.T) {
	m, a, b := testModel(t, testLimits())
	_, c, s := pair(t, m, a, b)
	if err := m.Delay("a", "b", 10, false); err != nil {
		t.Fatal(err)
	}
	w := m.Dispatch(Request{Domain: a, Op: "write", Handle: c, RequestID: 4, Bytes: []byte("actual stream")})
	if w.Count != 13 || w.Error != "" {
		t.Fatal(w)
	}
	r := m.Dispatch(Request{Domain: b, Op: "read", Handle: s, RequestID: 5, Count: 6})
	if r.State != "pending" || r.Deadline != 10 {
		t.Fatal(r)
	}
	if ids, err := m.Advance(9); err != nil || len(ids) != 0 {
		t.Fatal(ids, err)
	}
	if ids, err := m.Advance(10); err != nil || !reflect.DeepEqual(ids, []uint64{5}) {
		t.Fatal(ids, err)
	}
	r = m.Dispatch(Request{Domain: b, Op: "poll", RequestID: 5})
	if string(r.Bytes) != "actual" {
		t.Fatal(r)
	}
	if x := m.Dispatch(Request{Domain: a, Op: "close_write", Handle: c, RequestID: 6}); x.Error != "" {
		t.Fatal(x)
	}
	r = m.Dispatch(Request{Domain: b, Op: "read", Handle: s, RequestID: 7, Count: 64})
	if string(r.Bytes) != " stream" {
		t.Fatal(r)
	}
	r = m.Dispatch(Request{Domain: b, Op: "read", Handle: s, RequestID: 8, Count: 64})
	if r.Error != "EOF" {
		t.Fatal(r)
	}
}

func TestPartitionQueuedDrainBlockedDialCancelAndHeal(t *testing.T) {
	m, a, b := testModel(t, testLimits())
	l, c, s := pair(t, m, a, b)
	m.Dispatch(Request{Domain: a, Op: "write", Handle: c, RequestID: 4, Bytes: []byte("queued")})
	if err := m.Partition("a", "b", true); err != nil {
		t.Fatal(err)
	}
	w := m.Dispatch(Request{Domain: a, Op: "write", Handle: c, RequestID: 5, Bytes: []byte("dropped")})
	if w.Error != "" || w.Count != 7 {
		t.Fatal(w)
	}
	r := m.Dispatch(Request{Domain: b, Op: "read", Handle: s, RequestID: 6, Count: 64})
	if string(r.Bytes) != "queued" {
		t.Fatal(r)
	}
	d := m.Dispatch(Request{Domain: a, Op: "dial", Host: b.Address, Port: m.handles[l].listener.Address().Port, RequestID: 7, Deadline: 20})
	if d.State != "pending" || d.Deadline != 20 {
		t.Fatal(d)
	}
	if err := m.Cancel(a, 7); err != nil {
		t.Fatal(err)
	}
	d = m.Dispatch(Request{Domain: a, Op: "poll", RequestID: 7})
	if d.Error != "context canceled" {
		t.Fatal(d)
	}
	d = m.Dispatch(Request{Domain: a, Op: "dial", Host: b.Address, Port: m.handles[l].listener.Address().Port, RequestID: 8})
	if d.State != "pending" {
		t.Fatal(d)
	}
	if err := m.Heal("a", "b", true); err != nil {
		t.Fatal(err)
	}
	if ids := m.Ready(); !reflect.DeepEqual(ids, []uint64{8}) {
		t.Fatal(ids)
	}
	if d = m.Dispatch(Request{Domain: a, Op: "poll", RequestID: 8}); d.Error != "" || d.Handle == 0 {
		t.Fatal(d)
	}
}

func TestQueueBackpressureCapacityDeadlineAndStaleDomain(t *testing.T) {
	m, a, b := testModel(t, testLimits())
	_, c, s := pair(t, m, a, b)
	for i := uint64(0); i < 64; i++ {
		r := m.Dispatch(Request{Domain: a, Op: "write", Handle: c, RequestID: 10 + i, Bytes: []byte{byte(i)}})
		if r.Count != 1 || r.Error != "" {
			t.Fatal(r)
		}
	}
	r := m.Dispatch(Request{Domain: a, Op: "write", Handle: c, RequestID: 80, Bytes: []byte("held")})
	if r.State != "pending" {
		t.Fatal(r)
	}
	m.Dispatch(Request{Domain: b, Op: "read", Handle: s, RequestID: 81, Count: 1})
	if ids := m.Ready(); !reflect.DeepEqual(ids, []uint64{80}) {
		t.Fatal(ids)
	}
	r = m.Dispatch(Request{Domain: a, Op: "poll", RequestID: 80})
	if r.Count != 4 {
		t.Fatal(r)
	}
	if err := m.Revoke(a, false); err != nil {
		t.Fatal(err)
	}
	if r = m.Dispatch(Request{Domain: a, Op: "write", Handle: c, RequestID: 82, Bytes: []byte("stale")}); r.Error == "" {
		t.Fatal(r)
	}
	a.Incarnation++
	if err := m.Register(a); err != nil {
		t.Fatal(err)
	}
	if r = m.Dispatch(Request{Domain: a, Op: "write", Handle: c, RequestID: 83, Bytes: []byte("stale")}); r.Error == "" {
		t.Fatal(r)
	}
	limits := testLimits()
	limits.Bytes = 3
	n, x, y := testModel(t, limits)
	_, cc, ss := pair(t, n, x, y)
	if r = n.Dispatch(Request{Domain: x, Op: "write", Handle: cc, RequestID: 4, Bytes: []byte("four")}); r.Error != ErrResourceExhausted.Error() {
		t.Fatal(r)
	}
	n.Dispatch(Request{Domain: y, Op: "read_deadline", Handle: ss, RequestID: 5, Deadline: 10})
	if r = n.Dispatch(Request{Domain: y, Op: "read", Handle: ss, RequestID: 6, Count: 1}); r.State != "pending" {
		t.Fatal(r)
	}
	if _, err := n.Advance(10); err != nil {
		t.Fatal(err)
	}
	if r = n.Dispatch(Request{Domain: y, Op: "poll", RequestID: 6}); r.Error == "" {
		t.Fatal(r)
	}
}

func TestGeneratedOperationSequencesExactNativeReplay(t *testing.T) {
	for seed := uint64(1); seed <= 32; seed++ {
		t.Run(string(rune('A'+seed-1)), func(t *testing.T) {
			run := func(replay *Record) Record {
				m, a, b := testModel(t, testLimits())
				if replay != nil {
					m.network.replay = replay
					m.network.replayLanes = map[string][]simulationTransition{}
					m.network.replayNext = map[string]int{}
					for _, tr := range replay.Transitions {
						lane := simulationTransitionLane(tr)
						m.network.replayLanes[lane] = append(m.network.replayLanes[lane], tr)
					}
				}
				_, c, s := pair(t, m, a, b)
				random := seed
				var expected []byte
				for i := uint64(0); i < 64; i++ {
					random = random*6364136223846793005 + 1
					payload := bytes.Repeat([]byte{byte(random >> 32)}, int(random%127+1))
					w := m.Dispatch(Request{Domain: a, Op: "write", Handle: c, RequestID: 10 + i*2, Bytes: payload})
					if w.Error != "" {
						t.Fatal(w)
					}
					expected = append(expected, payload...)
					r := m.Dispatch(Request{Domain: b, Op: "read", Handle: s, RequestID: 11 + i*2, Count: uint32(len(payload))})
					if !bytes.Equal(r.Bytes, payload) {
						t.Fatal("stream oracle mismatch")
					}
				}
				if len(expected) == 0 {
					t.Fatal("empty generated stream")
				}
				record, err := m.Finish()
				if err != nil {
					t.Fatal(err)
				}
				return record
			}
			record := run(nil)
			if replay := run(&record); !reflect.DeepEqual(record, replay) {
				t.Fatal("native record replay differs")
			}
		})
	}
}

func TestDelayedReadDeadlineAndReset(t *testing.T) {
	m, a, b := testModel(t, testLimits())
	_, c, s := pair(t, m, a, b)
	if err := m.Delay("a", "b", 20, false); err != nil {
		t.Fatal(err)
	}
	m.Dispatch(Request{Domain: b, Op: "read_deadline", Handle: s, RequestID: 4, Deadline: 10})
	m.Dispatch(Request{Domain: a, Op: "write", Handle: c, RequestID: 5, Bytes: []byte("delayed")})
	r := m.Dispatch(Request{Domain: b, Op: "read", Handle: s, RequestID: 6, Count: 64})
	if r.State != "pending" || r.Deadline != 10 {
		t.Fatal(r)
	}
	if _, err := m.Advance(10); err != nil {
		t.Fatal(err)
	}
	if r = m.Dispatch(Request{Domain: b, Op: "poll", RequestID: 6}); r.Error != "context deadline exceeded" {
		t.Fatal(r)
	}
	if err := m.Reset(c); err != nil {
		t.Fatal(err)
	}
	if r = m.Dispatch(Request{Domain: b, Op: "read", Handle: s, RequestID: 7, Count: 64}); r.Error != ErrClosed.Error() {
		t.Fatal(r)
	}
}

func TestPendingRequestOwnsBytesAndCapacityFailures(t *testing.T) {
	m, a, b := testModel(t, testLimits())
	_, c, s := pair(t, m, a, b)
	for i := uint64(0); i < 64; i++ {
		m.Dispatch(Request{Domain: a, Op: "write", Handle: c, RequestID: 10 + i, Bytes: []byte{0}})
	}
	input := []byte("owned")
	if r := m.Dispatch(Request{Domain: a, Op: "write", Handle: c, RequestID: 80, Bytes: input}); r.State != "pending" {
		t.Fatal(r)
	}
	input[0] = 'X'
	for i := uint64(0); i < 64; i++ {
		m.Dispatch(Request{Domain: b, Op: "read", Handle: s, RequestID: 100 + i, Count: 1})
		m.Ready()
	}
	m.Dispatch(Request{Domain: a, Op: "poll", RequestID: 80})
	if r := m.Dispatch(Request{Domain: b, Op: "read", Handle: s, RequestID: 200, Count: 64}); string(r.Bytes) != "owned" {
		t.Fatal("pending bytes alias caller", r)
	}
	limits := testLimits()
	limits.Listeners = 1
	limits.Connections = 1
	limits.Transitions = 32
	n, x, y := testModel(t, limits)
	l, _, _ := pair(t, n, x, y)
	if r := n.Dispatch(Request{Domain: y, Op: "listen", RequestID: 4}); r.Error != ErrResourceExhausted.Error() {
		t.Fatal("listener capacity", r)
	}
	if r := n.Dispatch(Request{Domain: x, Op: "dial", Host: y.Address, Port: n.handles[l].listener.Address().Port, RequestID: 5}); r.Error != ErrResourceExhausted.Error() {
		t.Fatal("connection capacity", r)
	}
	if r := n.Dispatch(Request{Domain: x, Op: "write", Handle: 99, RequestID: 6, Bytes: make([]byte, MaximumChunkBytes+1)}); r.Error != ErrUnsupported.Error() {
		t.Fatal("oversized operation admitted", r)
	}
}

func TestRevokeDiscardsDeadIncarnationReadinessAndFinishRejectsHandles(t *testing.T) {
	m, a, b := testModel(t, testLimits())
	_, c, s := pair(t, m, a, b)
	if r := m.Dispatch(Request{Domain: b, Op: "read", Handle: s, RequestID: 4, Count: 1}); r.State != "pending" {
		t.Fatal(r)
	}
	if err := m.Revoke(b, false); err != nil {
		t.Fatal(err)
	}
	if ready := m.Ready(); len(ready) != 0 {
		t.Fatal("dead incarnation retained readiness", ready)
	}
	if r := m.Dispatch(Request{Domain: b, Op: "poll", RequestID: 4}); r.Error == "" {
		t.Fatal("stale completion admitted", r)
	}
	if _, err := m.Finish(); err != nil {
		t.Fatal(err)
	}
	if r := m.Dispatch(Request{Domain: a, Op: "read", Handle: c, RequestID: 5, Count: 1}); r.Error == "" {
		t.Fatal("terminal handle admitted", r)
	}
	if err := m.Register(Domain{Run: 1, Node: b.Node, Address: b.Address, Incarnation: 2}); err == nil {
		t.Fatal("terminal registration admitted")
	}
}
