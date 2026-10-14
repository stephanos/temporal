package network

import (
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"math"
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

func TestRequestStallTimeoutMessageIsDistinctFromCallerDeadline(t *testing.T) {
	if got := modelErrorMessage(syscall.ETIMEDOUT); got != "connection timed out" {
		t.Fatalf("stall timeout message=%q", got)
	}
	if got := modelErrorMessage(os.ErrDeadlineExceeded); got != "i/o timeout" {
		t.Fatalf("caller deadline message=%q", got)
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
	return testModelWithClockCeiling(t, limits, math.MaxInt64)
}

func testModelWithClockCeiling(t *testing.T, limits Limits, ceiling uint64) (*Model, Domain, Domain) {
	t.Helper()
	m, err := New(1, Config{Nodes: []Node{{Node: "a", Address: "a"}, {Node: "b", Address: "b"}}, Links: []Link{{From: "a", To: "b", Enabled: true}, {From: "b", To: "a", Enabled: true}}, Limits: limits})
	if err != nil {
		t.Fatal(err)
	}
	if err := m.LimitClockCeiling(ceiling); err != nil {
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
	return Limits{Listeners: 128, Connections: 128, Bytes: 8 << 20, ConnectionBytes: 4 << 20, StallNanos: 10_000_000_000, Transitions: 8192}
}

func TestModelRejectsInvalidExplicitNetworkLimits(t *testing.T) {
	for _, test := range []struct {
		bytes, stall uint64
	}{
		{0, 1}, {(1 << 30) + 1, 1}, {1, 0}, {1, 9223372036854775807}, {^uint64(0), ^uint64(0)},
	} {
		limits := testLimits()
		limits.ConnectionBytes, limits.StallNanos = test.bytes, test.stall
		if model, err := New(1, Config{Limits: limits}); err == nil || model != nil {
			t.Fatalf("invalid limits activated model: bytes=%d stall=%d", test.bytes, test.stall)
		}
	}
}

func TestModelHeldTimeoutReplayAdmission(t *testing.T) {
	source := simulationEndpoint{Node: "a", Incarnation: 1, Address: "A", Port: 2}
	destination := simulationEndpoint{Node: "b", Incarnation: 1, Address: "B", Port: 3}
	valid := simulationTransition{Kind: "reset", Source: source, Destination: destination, Connection: 1, Outcome: "timeout"}
	owner, err := New(1, Config{Limits: testLimits()})
	if err != nil {
		t.Fatal(err)
	}
	owner.network.transitions = []Transition{valid}
	record, err := owner.Record()
	if err != nil {
		t.Fatal(err)
	}
	config := Config{Limits: testLimits(), Replay: &record}
	if model, err := New(1, config); err != nil || model == nil {
		t.Fatalf("valid reset replay rejected: %v", err)
	}
	for _, mutate := range []func(*Transition){
		func(value *Transition) { value.Connection = 0 },
		func(value *Transition) { value.Kind = "unknown" },
		func(value *Transition) { value.Destination.Incarnation = 0 },
		func(value *Transition) { value.Ordinal = 1 },
	} {
		invalid := valid
		mutate(&invalid)
		config.Replay = &Record{Transitions: []Transition{invalid}}
		if model, err := New(1, config); err == nil || model != nil {
			t.Fatalf("invalid replay activated model: %+v", invalid)
		}
	}
}

func TestModelRejectsUnknownTransitionBeforeCommit(t *testing.T) {
	model, err := New(1, Config{Limits: testLimits()})
	if err != nil {
		t.Fatal(err)
	}
	for _, transition := range []Transition{
		{Kind: "unknown", Outcome: "ok"},
		{Kind: "reset", Source: simulationEndpoint{Node: "a", Incarnation: 1, Address: "A", Port: 2}, Destination: simulationEndpoint{Node: "b", Incarnation: 1, Address: "B", Port: 3}, Outcome: "timeout"},
	} {
		if err := model.network.commitTransitionLocked(transition); err == nil || len(model.network.transitions) != 0 {
			t.Fatalf("invalid transition committed: %+v error=%v", transition, err)
		}
	}
}

func TestModelReplayPreflightRejectsUnboundSnapshot(t *testing.T) {
	m, a, b := testModel(t, testLimits())
	_, client, _ := pair(t, m, a, b)
	m.Dispatch(Request{Domain: a, Op: "write", Handle: client, RequestID: 4, Bytes: []byte("actual")})
	record, err := m.Record()
	if err != nil {
		t.Fatal(err)
	}
	config := Config{Nodes: []Node{{Node: "a", Address: "a"}, {Node: "b", Address: "b"}}, Links: []Link{{From: "a", To: "b", Enabled: true}, {From: "b", To: "a", Enabled: true}}, Limits: testLimits(), Replay: &record}
	if _, err := New(2, config); err != nil {
		t.Fatal(err)
	}
	for _, mutate := range []func(*Record){
		func(r *Record) { r.Snapshot.Identity = "sha256:corrupted" },
		func(r *Record) { r.Snapshot.TransitionSHA256 = "sha256:corrupted" },
		func(r *Record) { r.Snapshot.Connections[0].ServerBytes++ },
		func(r *Record) { r.Transitions[len(r.Transitions)-1].Bytes++ },
	} {
		candidate := record
		candidate.Transitions = append([]Transition(nil), record.Transitions...)
		candidate.Snapshot.Connections = append([]simulationConnectionSnapshot(nil), record.Snapshot.Connections...)
		mutate(&candidate)
		config.Replay = &candidate
		if model, err := New(2, config); err == nil || model != nil {
			t.Fatal("unbound replay activated")
		}
	}
}

func TestModelReplayPreflightChecksSealedOccupancyAndOwnsInputs(t *testing.T) {
	limits := testLimits()
	limits.ConnectionBytes, limits.Bytes = 8, 10
	m, a, b := testModel(t, limits)
	_, client, _ := pair(t, m, a, b)
	m.Dispatch(Request{Domain: a, Op: "write", Handle: client, RequestID: 4, Bytes: []byte("actual")})
	record, err := m.Record()
	if err != nil {
		t.Fatal(err)
	}
	config := Config{Limits: limits, Replay: &record}
	for _, mutate := range []func(*Snapshot){
		func(s *Snapshot) { s.Connections[0].ServerBytes = 9 },
		func(s *Snapshot) { s.Connections[0].ClientBytes = 5 },
		func(s *Snapshot) { s.Connections[0].Reset, s.Connections[0].Closed = true, true },
		func(s *Snapshot) { s.Connections[0].Identity = 0 },
		func(s *Snapshot) { s.Nodes[0].NextClientPort = 0 },
		func(s *Snapshot) { s.Links[0].To = s.Links[0].From },
		func(s *Snapshot) { s.Listeners[0].Endpoint.Port = 65536 },
	} {
		candidate := record
		candidate.Snapshot.Nodes = append([]simulationNodeSnapshot(nil), record.Snapshot.Nodes...)
		candidate.Snapshot.Links = append([]Link(nil), record.Snapshot.Links...)
		candidate.Snapshot.Listeners = append([]simulationListenerSnapshot(nil), record.Snapshot.Listeners...)
		candidate.Snapshot.Connections = append([]simulationConnectionSnapshot(nil), record.Snapshot.Connections...)
		mutate(&candidate.Snapshot)
		candidate.Snapshot.Identity = ""
		encoded, err := encodeSimulationSnapshotIdentity(candidate.Snapshot)
		if err != nil {
			t.Fatal(err)
		}
		candidate.Snapshot.Identity = fmt.Sprintf("sha256:%x", sha256.Sum256(append([]byte("gomad3-simulation-network-snapshot/v3\x00"), encoded...)))
		config.Replay = &candidate
		if model, err := New(2, config); err == nil || model != nil {
			t.Fatalf("invalid sealed snapshot activated: %+v", candidate.Snapshot)
		}
	}
	config.Replay = &record
	replay, err := New(2, config)
	if err != nil {
		t.Fatal(err)
	}
	record.Snapshot.Nodes[0].Address = "changed"
	record.Snapshot.Links[0].Enabled = false
	record.Snapshot.Listeners[0].Closed = true
	record.Snapshot.Connections[0].ServerBytes = 0
	if replay.network.replay.Snapshot.Nodes[0].Address == "changed" || !replay.network.replay.Snapshot.Links[0].Enabled || replay.network.replay.Snapshot.Listeners[0].Closed || replay.network.replay.Snapshot.Connections[0].ServerBytes != 6 {
		t.Fatal("replay aliases caller snapshot")
	}
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

func TestPartitionQueuedHoldBlockedDialCancelAndHeal(t *testing.T) {
	m, a, b := testModel(t, testLimits())
	l, c, s := pair(t, m, a, b)
	m.Dispatch(Request{Domain: a, Op: "write", Handle: c, RequestID: 4, Bytes: []byte("queued")})
	if err := m.Partition("a", "b", true); err != nil {
		t.Fatal(err)
	}
	w := m.Dispatch(Request{Domain: a, Op: "write", Handle: c, RequestID: 5, Bytes: []byte("held")})
	if w.Error != "" || w.Count != 4 {
		t.Fatal(w)
	}
	r := m.Dispatch(Request{Domain: b, Op: "read", Handle: s, RequestID: 6, Count: 64})
	if r.State != "pending" {
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
	if ids := m.Ready(); !reflect.DeepEqual(ids, []uint64{6, 8}) {
		t.Fatal(ids)
	}
	if d = m.Dispatch(Request{Domain: a, Op: "poll", RequestID: 8}); d.Error != "" || d.Handle == 0 {
		t.Fatal(d)
	}
	if r = m.Dispatch(Request{Domain: b, Op: "poll", RequestID: 6}); string(r.Bytes) != "queued" {
		t.Fatal(r)
	}
	if r = m.Dispatch(Request{Domain: b, Op: "read", Handle: s, RequestID: 9, Count: 64}); string(r.Bytes) != "held" {
		t.Fatal(r)
	}
}

func TestQueueBackpressureCapacityDeadlineAndStaleDomain(t *testing.T) {
	localLimits := testLimits()
	localLimits.ConnectionBytes = 64
	m, a, b := testModel(t, localLimits)
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
	for id := uint64(81); id < 85; id++ {
		m.Dispatch(Request{Domain: b, Op: "read", Handle: s, RequestID: id, Count: 1})
	}
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
			t.Run("negative_capacity", func(t *testing.T) {
				run := func(replay *Record) Record {
					limits := testLimits()
					limits.Bytes = 3
					m, a, b := testModel(t, limits)
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
					for cycle := uint64(0); cycle < 8; cycle++ {
						random = random*6364136223846793005 + 1
						payload := []byte{byte(random >> 32), byte(random >> 40), byte(random >> 48)}
						id := uint64(10) + cycle*10
						if got, want := m.Dispatch(Request{Domain: a, Op: "write", Handle: c, RequestID: id, Bytes: payload}), (Result{RequestID: id, State: "completed", Count: 3}); !reflect.DeepEqual(got, want) {
							t.Fatalf("generated write: got=%+v want=%+v", got, want)
						}
						stale := a
						stale.Incarnation = 2
						negative := []struct {
							request Request
							error   string
						}{
							{Request{Domain: b, Op: "read", Handle: s, Count: 65537}, "unsupported Gomad network operation"},
							{Request{Domain: stale, Op: "write", Handle: c, Bytes: []byte("x")}, "stale file handle"},
							{Request{Domain: a, Op: "dial", Host: "b", Port: 1}, "connection refused"},
						}
						for step := uint64(0); step < 3; step++ {
							entry := negative[(step+random%3)%3]
							entry.request.RequestID = id + 1 + step
							before, err := m.Snapshot()
							if err != nil {
								t.Fatal(err)
							}
							got := m.Dispatch(entry.request)
							want := Result{RequestID: entry.request.RequestID, State: "completed", Error: entry.error}
							if !reflect.DeepEqual(got, want) {
								t.Fatalf("generated negative: got=%+v want=%+v", got, want)
							}
							if entry.request.Op == "dial" {
								want := Transition{Kind: "dial", Source: simulationEndpoint{Node: "a", Incarnation: 1, Address: "a"}, Destination: simulationEndpoint{Node: "b", Address: "b", Port: 1}, Outcome: "refused"}
								if got := m.network.transitions[len(m.network.transitions)-1]; got != want {
									t.Fatalf("refused transition: got=%+v want=%+v", got, want)
								}
							}
							after, err := m.Snapshot()
							if err != nil {
								t.Fatal(err)
							}
							before.Identity, before.TransitionSHA256 = "", ""
							after.Identity, after.TransitionSHA256 = "", ""
							if !reflect.DeepEqual(before, after) {
								t.Fatal("negative request mutated network state")
							}
						}
						if got, want := m.Dispatch(Request{Domain: a, Op: "write", Handle: c, RequestID: id + 4, Bytes: []byte("x")}), (Result{RequestID: id + 4, State: "completed", Error: "network resources exhausted"}); !reflect.DeepEqual(got, want) {
							t.Fatalf("generated capacity: got=%+v want=%+v", got, want)
						}
						want := Transition{Kind: "write", Source: simulationEndpoint{Node: "a", Incarnation: 1, Address: "a", Port: 40000}, Destination: simulationEndpoint{Node: "b", Incarnation: 1, Address: "b", Port: 20000}, Connection: 1, Bytes: 1, Outcome: "capacity"}
						if got := m.network.transitions[len(m.network.transitions)-1]; got != want {
							t.Fatalf("capacity transition: got=%+v want=%+v", got, want)
						}
						queued, err := m.Snapshot()
						if err != nil {
							t.Fatal(err)
						}
						if queued.NextConnection != 1 || len(queued.Connections) != 1 || queued.Connections[0].ServerBytes != 3 || queued.Connections[0].ClientBytes != 0 || m.network.pendingBytes != 3 || len(m.pending) != 0 {
							t.Fatalf("rejected write changed bounded queue: %+v", queued)
						}
						if got, want := m.Dispatch(Request{Domain: b, Op: "read", Handle: s, RequestID: id + 5, Count: 3}), (Result{RequestID: id + 5, State: "completed", Count: 3, Bytes: payload}); !reflect.DeepEqual(got, want) {
							t.Fatalf("capacity drain: got=%+v want=%+v", got, want)
						}
						if m.network.pendingBytes != 0 || len(m.pending) != 0 {
							t.Fatal("drain did not restore capacity")
						}
					}
					record, err := m.Finish()
					if err != nil {
						t.Fatal(err)
					}
					if len(record.Transitions) != 27 {
						t.Fatalf("generated transition count: got=%d want=27", len(record.Transitions))
					}
					return record
				}
				record := run(nil)
				if replay := run(&record); !reflect.DeepEqual(record, replay) {
					t.Fatal("generated negative/capacity native record replay differs")
				}
			})
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
	if r = m.Dispatch(Request{Domain: b, Op: "poll", RequestID: 6}); r.Error != "i/o timeout" {
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
	localLimits := testLimits()
	localLimits.ConnectionBytes = 64
	m, a, b := testModel(t, localLimits)
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
	var received []byte
	for id := uint64(200); len(received) < 5; id++ {
		r := m.Dispatch(Request{Domain: b, Op: "read", Handle: s, RequestID: id, Count: 64})
		if r.State != "completed" || r.Error != "" {
			t.Fatal(r)
		}
		received = append(received, r.Bytes...)
	}
	if string(received) != "owned" {
		t.Fatalf("pending bytes alias or repeat caller data: %q", received)
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
