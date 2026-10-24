// Copyright 2026 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license.

// Package network is gomad's network model behind a nonblocking request
// interface. The simulation in simulation.go owns all stream queues.
package network

import (
	"context"
	"errors"
	"io"
	"math"
	"os"
	"sort"
	"sync"
	"syscall"
	"time"
)

const Profile = "gomad.simulation-network-stream/v3"
const MaximumChunkBytes = 64 << 10
const MaximumPendingOperations = maximumPendingNetworkOperations

type Domain struct {
	Run         uint64 `json:"run"`
	Node        string `json:"node"`
	Incarnation uint64 `json:"incarnation"`
	Address     string `json:"address"`
}

type Config = simulationNetworkConfig
type Limits = simulationLimits
type Node = simulationNodeConfig
type Link = simulationLink
type Record = simulationRecord
type Snapshot = simulationSnapshot
type Transition = simulationTransition

type Request struct {
	Domain    Domain `json:"domain"`
	Op        string `json:"op"`
	Handle    uint64 `json:"handle,omitempty"`
	Host      string `json:"host,omitempty"`
	Port      int    `json:"port,omitempty"`
	Bytes     []byte `json:"bytes,omitempty"`
	Count     uint32 `json:"count,omitempty"`
	Deadline  uint64 `json:"deadline,omitempty"`
	RequestID uint64 `json:"request_id"`
}

type Result struct {
	RequestID uint64  `json:"request_id"`
	State     string  `json:"state"`
	Handle    uint64  `json:"handle,omitempty"`
	Local     Address `json:"local,omitempty"`
	Remote    Address `json:"remote,omitempty"`
	Bytes     []byte  `json:"bytes,omitempty"`
	Count     int     `json:"count,omitempty"`
	Deadline  uint64  `json:"deadline,omitempty"`
	Error     string  `json:"error,omitempty"`
}

type waiting struct {
	Changed  <-chan struct{}
	Deadline time.Time
	Accepted int
	Dial     *simulationDial
}
type pending struct {
	request  Request
	wait     waiting
	result   *Result
	accepted int
}
type ownedHandle struct {
	domain     Domain
	listener   *Listener
	connection *Conn
}

type Model struct {
	mu         sync.Mutex
	network    *simulationNetwork
	handles    map[uint64]ownedHandle
	pending    map[uint64]*pending
	nextHandle uint64
	maxPending uint64
	cancelErr  error
}

func New(run uint64, config Config) (*Model, error) {
	if config.Replay != nil {
		replay := *config.Replay
		replay.Transitions = append([]simulationTransition(nil), replay.Transitions...)
		replay.Snapshot.Nodes = append([]simulationNodeSnapshot(nil), replay.Snapshot.Nodes...)
		replay.Snapshot.Links = append([]simulationLink(nil), replay.Snapshot.Links...)
		replay.Snapshot.Listeners = append([]simulationListenerSnapshot(nil), replay.Snapshot.Listeners...)
		replay.Snapshot.Connections = append([]simulationConnectionSnapshot(nil), replay.Snapshot.Connections...)
		config.Replay = &replay
	}
	n, err := newSimulationNetwork(run, config)
	if err != nil {
		return nil, err
	}
	n.now = time.Unix(0, 0)
	n.clockCeiling = math.MaxInt64
	n.live = make(map[string]Domain)
	return &Model{network: n, handles: make(map[uint64]ownedHandle), pending: make(map[uint64]*pending), maxPending: MaximumPendingOperations}, nil
}

func (m *Model) LimitClockCeiling(ceiling uint64) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.network.finished {
		return syscall.ESTALE
	}
	if ceiling == 0 || ceiling > math.MaxInt64 || uint64(m.network.now.UnixNano()) > ceiling {
		return ErrUnsupported
	}
	if ceiling >= m.network.clockCeiling {
		return nil
	}
	within := func(deadline time.Time) bool { return deadline.IsZero() || uint64(deadline.UnixNano()) <= ceiling }
	for _, pending := range m.pending {
		if pending.result != nil {
			continue
		}
		if pending.request.Deadline > ceiling || !within(pending.wait.Deadline) {
			return ErrUnsupported
		}
	}
	for pending := range m.network.pendingDials {
		if pending.stalled && !within(pending.due) {
			return ErrUnsupported
		}
	}
	for _, pair := range m.network.connections {
		if pair.closed || pair.reset {
			continue
		}
		for _, state := range []*connState{pair.clientState, pair.serverState} {
			if !within(state.readDeadline) || !within(state.writeDeadline) || state.stalled && !within(state.stallDue) {
				return ErrUnsupported
			}
			for _, chunk := range state.incoming {
				if !within(chunk.ready) {
					return ErrUnsupported
				}
			}
		}
	}
	for _, handle := range m.handles {
		if handle.listener != nil {
			listener, ok := handle.listener.implementation.(*simulationListener)
			if !ok || !listener.closed && !within(listener.deadline) {
				return ErrUnsupported
			}
		}
	}
	m.network.clockCeiling = ceiling
	return nil
}

func (m *Model) Register(domain Domain) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	n := m.network.nodes[domain.Node]
	if m.network.finished || domain.Run != m.network.run || domain.Incarnation == 0 || n == nil || domain.Address != n.address {
		return syscall.ESTALE
	}
	if previous, ok := m.network.live[domain.Node]; ok && previous != domain {
		return syscall.ESTALE
	}
	if domain.Incarnation <= n.lastIncarnation {
		return syscall.ESTALE
	}
	m.network.live[domain.Node] = domain
	n.lastIncarnation = domain.Incarnation
	return nil
}

func (n *simulationNetwork) validateEndpoint(endpoint simulationEndpoint) error {
	d := n.current
	if n.finished || d.Run != n.run || d.Node != endpoint.Node || d.Incarnation != endpoint.Incarnation || d.Address != endpoint.Address || n.live[d.Node] != d {
		return syscall.ESTALE
	}
	return nil
}
func (n *simulationNetwork) nowTime() time.Time { return n.now }

func (n *simulationNetwork) stallDeadline(start time.Time, limit uint64) (time.Time, error) {
	due, err := strictNetworkStallDeadline(start, limit)
	if err != nil || uint64(due.UnixNano()) > n.clockCeiling {
		return time.Time{}, ErrUnsupported
	}
	return due, nil
}

func (n *simulationNetwork) readyTime(start time.Time, delay uint64) (time.Time, error) {
	due, err := checkedNetworkReadyTime(start, delay)
	if err != nil || uint64(due.UnixNano()) > n.clockCeiling {
		return time.Time{}, ErrUnsupported
	}
	return due, nil
}
func (n *simulationNetwork) deadlineExpired(deadline time.Time) bool {
	return !deadline.IsZero() && !n.now.Before(deadline)
}
func virtualTime(value uint64) time.Time {
	if value == 0 {
		return time.Time{}
	}
	return time.Unix(0, int64(value))
}
func virtualNanos(value time.Time) uint64 {
	if value.IsZero() {
		return 0
	}
	return uint64(value.UnixNano())
}

// Pending reports whether a live domain owns an uncollected request.
func (m *Model) Pending(domain Domain, id uint64) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	p := m.pending[id]
	return !m.network.finished && m.network.live[domain.Node] == domain && p != nil && p.request.Domain == domain
}

func (m *Model) Dispatch(request Request) Result {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.network.finished {
		return failed(request, syscall.ESTALE)
	}
	if request.Op == "poll" {
		p := m.pending[request.RequestID]
		if p == nil || p.request.Domain != request.Domain {
			return failed(request, syscall.ESTALE)
		}
		if p.result != nil {
			result := *p.result
			delete(m.pending, request.RequestID)
			return result
		}
		return Result{RequestID: request.RequestID, State: "pending", Deadline: virtualNanos(p.wait.Deadline)}
	}
	if request.RequestID == 0 || m.pending[request.RequestID] != nil || request.Deadline > m.network.clockCeiling || request.Count > MaximumChunkBytes || len(request.Bytes) > MaximumChunkBytes {
		return failed(request, ErrUnsupported)
	}
	if m.network.live[request.Domain.Node] != request.Domain {
		return failed(request, syscall.ESTALE)
	}
	if uint64(len(m.pending)) >= m.maxPending {
		return failed(request, ErrResourceExhausted)
	}
	request.Bytes = append([]byte(nil), request.Bytes...)
	result, wait := m.attempt(request)
	if wait != nil {
		if request.Op == "write" {
			request.Bytes = request.Bytes[result.Count:]
		}
		m.pending[request.RequestID] = &pending{request: request, wait: *wait, accepted: result.Count}
	}
	return result
}

func failed(r Request, err error) Result {
	return Result{RequestID: r.RequestID, State: "completed", Error: modelErrorMessage(err)}
}

// The projection owns its complete error vocabulary. Never invoke an error
// interface callback (including the platform syscall message formatter) at the
// pure request boundary. Native structured replay/capacity errors retain their
// detached message directly.
func modelErrorMessage(err error) string {
	if bridge, ok := err.(*simulationBridgeError); ok {
		if bridge == nil {
			return "network model error"
		}
		return bridge.Message
	}
	switch err {
	case nil:
		return ""
	case syscall.ESTALE:
		return "stale file handle"
	case context.Canceled:
		return "context canceled"
	case context.DeadlineExceeded:
		return "context deadline exceeded"
	case os.ErrDeadlineExceeded:
		return "i/o timeout"
	case ErrStallTimeout:
		return "connection timed out"
	case io.EOF:
		return "EOF"
	case ErrAddressInUse:
		return "address already in use"
	case ErrClosed:
		return "use of closed network connection"
	case ErrConnectionRefused:
		return "connection refused"
	case ErrResourceExhausted:
		return "network resources exhausted"
	case ErrUnsupported:
		return "unsupported Gomad network operation"
	case ErrReplayDiverged:
		return "simulation network replay diverged"
	default:
		return "network model error"
	}
}

func (m *Model) attempt(r Request, retained ...*simulationDial) (result Result, blocked *waiting) {
	var pendingDial *simulationDial
	if len(retained) != 0 {
		pendingDial = retained[0]
	}
	result = Result{RequestID: r.RequestID, State: "completed"}
	m.network.current = r.Domain
	defer func() {
		if v := recover(); v != nil {
			wait, ok := v.(waiting)
			if !ok {
				panic(v) //nolint:forbidigo // Only projected waits are transport signals; preserve every unrelated native panic unchanged.
			}
			if deadline := virtualTime(r.Deadline); !deadline.IsZero() && (wait.Deadline.IsZero() || deadline.Before(wait.Deadline)) {
				wait.Deadline = deadline
			}
			if m.network.deadlineExpired(wait.Deadline) {
				result = failed(r, context.DeadlineExceeded)
				return
			}
			result = Result{RequestID: r.RequestID, State: "pending", Deadline: virtualNanos(wait.Deadline), Count: wait.Accepted}
			blocked = &wait
		}
	}()
	if m.network.live[r.Domain.Node] != r.Domain {
		return failed(r, syscall.ESTALE), nil
	}
	h := m.handles[r.Handle]
	if r.Op != "listen" && r.Op != "dial" && (r.Handle == 0 || h.domain != r.Domain) {
		return failed(r, syscall.ESTALE), nil
	}
	var err error
	switch r.Op {
	case "listen":
		if r.Port < 0 || r.Port > 65535 {
			return failed(r, ErrUnsupported), nil
		}
		var listener *Listener
		listener, err = m.network.listen("tcp", r.Host, r.Port, r.Domain)
		if err == nil {
			result.Handle = m.add(ownedHandle{domain: r.Domain, listener: listener})
			result.Local = listener.Address()
		}
	case "dial":
		if r.Port <= 0 || r.Port > 65535 {
			return failed(r, ErrUnsupported), nil
		}
		var conn *Conn
		conn, err = m.network.dial(virtualContext{network: m.network, deadline: virtualTime(r.Deadline), cancelled: m.cancelErr}, "tcp", r.Host, r.Port, r.Domain, pendingDial)
		if err == nil {
			result.Handle = m.add(ownedHandle{domain: r.Domain, connection: conn})
			result.Local = conn.LocalAddress()
			result.Remote = conn.RemoteAddress()
		}
	case "accept":
		if h.listener == nil {
			return failed(r, ErrUnsupported), nil
		}
		var conn *Conn
		conn, err = h.listener.Accept()
		if err == nil {
			result.Handle = m.add(ownedHandle{domain: r.Domain, connection: conn})
			result.Local = conn.LocalAddress()
			result.Remote = conn.RemoteAddress()
		}
	case "read":
		if h.connection == nil {
			return failed(r, ErrUnsupported), nil
		}
		result.Bytes = make([]byte, r.Count)
		result.Count, err = h.connection.Read(result.Bytes)
		result.Bytes = result.Bytes[:result.Count]
	case "write":
		if h.connection == nil {
			return failed(r, ErrUnsupported), nil
		}
		result.Count, err = h.connection.Write(r.Bytes)
	case "close":
		if h.listener != nil {
			err = h.listener.Close()
		} else if h.connection != nil {
			err = h.connection.Close()
		} else {
			err = ErrUnsupported
		}
	case "close_read", "close_write":
		if h.connection == nil {
			return failed(r, ErrUnsupported), nil
		}
		if r.Op == "close_read" {
			err = h.connection.CloseRead()
		} else {
			err = h.connection.CloseWrite()
		}
	case "deadline", "read_deadline", "write_deadline":
		deadline := virtualTime(r.Deadline)
		if h.listener != nil && r.Op == "deadline" {
			err = h.listener.SetDeadline(deadline)
		} else if h.connection == nil {
			err = ErrUnsupported
		} else if r.Op == "deadline" {
			err = h.connection.SetDeadline(deadline)
		} else if r.Op == "read_deadline" {
			err = h.connection.SetReadDeadline(deadline)
		} else {
			err = h.connection.SetWriteDeadline(deadline)
		}
	default:
		err = ErrUnsupported
	}
	if err != nil {
		result.Error = modelErrorMessage(err)
	}
	return result, nil
}
func (m *Model) add(h ownedHandle) uint64 {
	m.nextHandle++
	m.handles[m.nextHandle] = h
	return m.nextHandle
}

type virtualContext struct {
	network   *simulationNetwork
	deadline  time.Time
	cancelled error
}

func (v virtualContext) Deadline() (time.Time, bool) { return v.deadline, !v.deadline.IsZero() }
func (v virtualContext) Done() <-chan struct{}       { return nil }
func (v virtualContext) Err() error {
	if v.cancelled != nil {
		return v.cancelled
	}
	if v.network.deadlineExpired(v.deadline) {
		return context.DeadlineExceeded
	}
	return nil
}
func (v virtualContext) Value(any) any         { return nil }
func deadlineOf(ctx context.Context) time.Time { value, _ := ctx.Deadline(); return value }

func pendingDialDeadline(ctx context.Context, pending *simulationDial) time.Time {
	deadline := deadlineOf(ctx)
	if pending != nil && pending.stalled {
		return earliestDeadline(deadline, pending.due)
	}
	return deadline
}

// Ready settles signaled operations in stable request order without waiting.
func (m *Model) Ready() []uint64 {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.ready()
}
func (m *Model) ready() []uint64 {
	if m.network.finished {
		return nil
	}
	ids := make([]uint64, 0, len(m.pending))
	for id := range m.pending {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	var completed []uint64
	for _, id := range ids {
		p := m.pending[id]
		if p.result != nil {
			completed = append(completed, id)
			continue
		}
		signaled := m.network.deadlineExpired(p.wait.Deadline)
		select {
		case <-p.wait.Changed:
			signaled = true
		default:
		}
		if !signaled {
			continue
		}
		result, wait := m.attempt(p.request, p.wait.Dial)
		if p.request.Op == "write" {
			p.accepted += result.Count
			p.request.Bytes = p.request.Bytes[result.Count:]
			result.Count = p.accepted
		}
		if wait == nil {
			p.result = &result
			completed = append(completed, id)
		} else {
			p.wait = *wait
		}
	}
	return completed
}
func (m *Model) Advance(now uint64) ([]uint64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.advance(now)
}

// AdvanceBefore positions the clock after settling only strictly earlier events.
func (m *Model) AdvanceBefore(now uint64) ([]uint64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.network.finished {
		return nil, syscall.ESTALE
	}
	if now > m.network.clockCeiling || now < uint64(m.network.now.UnixNano()) {
		return nil, errors.New("network clock moved backwards or overflowed")
	}
	var ids []uint64
	for next := m.nextDeadline(); next != 0 && next < now; next = m.nextDeadline() {
		var err error
		ids, err = m.advance(max(next, uint64(m.network.now.UnixNano())))
		if err != nil {
			return nil, err
		}
	}
	if now > uint64(m.network.now.UnixNano()) {
		var err error
		ids, err = m.advance(now - 1)
		if err != nil {
			return nil, err
		}
	}
	return ids, m.position(now)
}

func (m *Model) position(now uint64) error {
	if m.network.finished {
		return syscall.ESTALE
	}
	if now > m.network.clockCeiling || now < uint64(m.network.now.UnixNano()) {
		return errors.New("network clock moved backwards or overflowed")
	}
	m.network.now = time.Unix(0, int64(now))
	return nil
}

func (m *Model) advance(now uint64) ([]uint64, error) {
	if err := m.position(now); err != nil {
		return nil, err
	}
	m.network.Lock()
	dials := make([]*simulationDial, 0, len(m.network.pendingDials))
	for pending := range m.network.pendingDials {
		dials = append(dials, pending)
	}
	sort.Slice(dials, func(i, j int) bool { return dials[i].identity < dials[j].identity })
	for _, pending := range dials {
		if pending.stalled && !m.network.now.Before(pending.due) {
			if err := m.network.expireDialLocked(pending, pending.epoch, pending.due); err != nil {
				m.network.Unlock()
				return nil, err
			}
		}
	}
	identities := make([]uint64, 0, len(m.network.connections))
	for identity := range m.network.connections {
		identities = append(identities, identity)
	}
	sort.Slice(identities, func(i, j int) bool { return identities[i] < identities[j] })
	for _, identity := range identities {
		pair := m.network.connections[identity]
		for _, state := range []*connState{pair.clientState, pair.serverState} {
			if state.stalled && !m.network.now.Before(state.stallDue) {
				if err := m.network.expireStallLocked(pair, state, state.stallEpoch, state.stallDue); err != nil {
					m.network.Unlock()
					return nil, err
				}
			}
		}
	}
	m.network.Unlock()
	return m.ready(), nil
}
func (m *Model) NextDeadline() uint64 {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.nextDeadline()
}

func (m *Model) nextDeadline() uint64 {
	if m.network.finished {
		return 0
	}
	var next uint64
	for pending := range m.network.pendingDials {
		if pending.stalled {
			due := virtualNanos(pending.due)
			if due != 0 && (next == 0 || due < next) {
				next = due
			}
		}
	}
	for _, pair := range m.network.connections {
		for _, state := range []*connState{pair.clientState, pair.serverState} {
			if state.stalled {
				due := virtualNanos(state.stallDue)
				if due != 0 && (next == 0 || due < next) {
					next = due
				}
			}
		}
	}
	for _, p := range m.pending {
		if p.result != nil {
			continue
		}
		d := virtualNanos(p.wait.Deadline)
		if p.request.Op == "dial" {
			d = virtualNanos(pendingDialDeadline(virtualContext{network: m.network, deadline: virtualTime(p.request.Deadline)}, p.wait.Dial))
		}
		if d != 0 && (next == 0 || d < next) {
			next = d
		}
	}
	return next
}
func (m *Model) Cancel(domain Domain, id uint64) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.network.finished {
		return syscall.ESTALE
	}
	p := m.pending[id]
	if p == nil || p.request.Domain != domain {
		return syscall.ESTALE
	}
	result := failed(p.request, context.Canceled)
	if p.request.Op == "dial" && p.result == nil {
		m.cancelErr = context.Canceled
		result, _ = m.attempt(p.request, p.wait.Dial)
		m.cancelErr = nil
	}
	p.result = &result
	return nil
}
func (m *Model) Partition(left, right string, symmetric bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if symmetric {
		return m.network.changePair(left, right, "partition", false, 0)
	}
	return m.network.changeLink(left, right, "disconnect", false, 0)
}

func (m *Model) ChangeGroup(left, right []string, action string, nanos uint64, symmetric bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	kind, enabled := action, action != "partition"
	switch action {
	case "partition":
		nanos = 0
		if !symmetric {
			kind = "disconnect"
		}
	case "heal":
		nanos = 0
		if !symmetric {
			kind = "reconnect"
		}
	case "delay":
		if nanos > math.MaxInt64 {
			return ErrUnsupported
		}
		if !symmetric {
			kind = "directional_delay"
		}
	default:
		return ErrUnsupported
	}
	return m.network.changeLinkGroup(left, right, kind, enabled, nanos, symmetric)
}
func (m *Model) Heal(left, right string, symmetric bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if symmetric {
		return m.network.changePair(left, right, "heal", true, 0)
	}
	return m.network.changeLink(left, right, "reconnect", true, 0)
}
func (m *Model) Delay(left, right string, nanos uint64, symmetric bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if nanos > math.MaxInt64 {
		return ErrUnsupported
	}
	if symmetric {
		return m.network.changePair(left, right, "delay", true, nanos)
	}
	return m.network.changeLink(left, right, "directional_delay", true, nanos)
}
func (m *Model) Revoke(domain Domain, graceful bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.network.live[domain.Node] != domain {
		return syscall.ESTALE
	}
	if err := m.network.revoke(domain, graceful); err != nil {
		return err
	}
	delete(m.network.live, domain.Node)
	for id, p := range m.pending {
		if p.request.Domain == domain {
			delete(m.pending, id)
		}
	}
	return nil
}
func (m *Model) Snapshot() (Snapshot, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.network.Lock()
	defer m.network.Unlock()
	return m.network.snapshotLocked()
}
func (m *Model) Finish() (Record, error) { m.mu.Lock(); defer m.mu.Unlock(); return m.network.finish() }
func (m *Model) Record() (Record, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.network.Lock()
	defer m.network.Unlock()
	transitions := append([]simulationTransition(nil), m.network.transitions...)
	canonicalizeSimulationTransitions(transitions)
	original := m.network.transitions
	m.network.transitions = transitions
	defer func() { m.network.transitions = original }()
	snapshot, err := m.network.snapshotLocked()
	if err != nil {
		return Record{}, err
	}
	return Record{Transitions: transitions, Snapshot: snapshot}, nil
}

// Reset derives the native crash reset of a connection without revoking its
// listener domain. This extension has the separate stream profile identity.
func (m *Model) Reset(handle uint64) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	h := m.handles[handle]
	if h.connection == nil {
		return syscall.ESTALE
	}
	connection, ok := h.connection.implementation.(*simulationConn)
	if !ok {
		return ErrUnsupported
	}
	m.network.Lock()
	if state := m.network.connections[connection.identity]; state != nil && state.reset {
		m.network.Unlock()
		return nil
	}
	pair := m.network.connections[connection.identity]
	pendingBytes := pair.clientState.retainedBytes + pair.serverState.retainedBytes
	var release uint64
	if pair.reserved {
		release = 1
	}
	err := m.network.commitBatchLocked([]simulationTransition{{Kind: "crash", Source: simulationEndpoint{Node: connection.owner.Node, Incarnation: connection.owner.Incarnation, Address: connection.owner.Address}, Count: 1, Bytes: pendingBytes, Outcome: "reset"}}, 0, release)
	m.network.Unlock()
	if err != nil {
		return err
	}
	return m.network.resetConnection(connection.identity)
}
