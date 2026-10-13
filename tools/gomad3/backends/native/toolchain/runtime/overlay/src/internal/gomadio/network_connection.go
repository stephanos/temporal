// Copyright 2026 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gomadio

import (
	"internal/gomadvfd"
	"io"
	"math"
	"os"
	"sync"
	"time"
)

type pairedConn struct {
	local    Address
	remote   Address
	state    *connState
	peer     *connState
	network  *simulationNetwork
	owner    simulationEndpoint
	target   simulationEndpoint
	identity uint64
}

type connState struct {
	shared        *connShared
	incoming      []networkChunk
	reset         bool
	terminal      error
	pending       []byte
	retainedBytes uint64
	closeCalled   bool
	stalled       bool
	stallStart    time.Time
	stallDue      time.Time
	stallEpoch    uint64
	stallTimer    *time.Timer
	readClosed    bool
	writeClosed   bool
	readDeadline  time.Time
	writeDeadline time.Time
	descriptor    descriptorIdentity
}

type connShared struct {
	sync.Mutex
	changed chan struct{}
	pair    *simulationConnection
}

type networkChunk struct {
	identity    uint64
	connection  uint64
	source      simulationEndpoint
	destination simulationEndpoint
	bytes       []byte
	ready       time.Time
	delayNanos  uint64
}

func (connection *pairedConn) validate() error {
	if connection.network != nil {
		return validateSimulationEndpoint(connection.network, connection.owner)
	}
	return nil
}
func (connection *pairedConn) nowTime() time.Time { return time.Now() }
func (connection *pairedConn) deadlineExpired(deadline time.Time) bool {
	return !deadline.IsZero() && !connection.nowTime().Before(deadline)
}
func (connection *pairedConn) terminalError() error {
	if connection.network != nil && connection.network.divergence != nil {
		return connection.network.divergence
	}
	if connection.state.terminal != nil {
		return connection.state.terminal
	}
	if connection.peer.terminal != nil {
		return connection.peer.terminal
	}
	if connection.state.reset || connection.peer.reset {
		return ErrClosed
	}
	return nil
}

func (connection *pairedConn) lockState() {
	if connection.network != nil {
		connection.network.Lock()
	}
	connection.state.shared.Lock()
}

func (connection *pairedConn) unlockState() {
	connection.state.shared.Unlock()
	if connection.network != nil {
		connection.network.Unlock()
	}
}

func (connection *pairedConn) Read(destination []byte) (int, error) {
	if len(destination) == 0 {
		record("net.read", networkArguments("tcp", connection.local.Port, connection.remote.Port, 0), nil, 0, 0, 0, 0)
		return 0, nil
	}

	if err := connection.validate(); err != nil {
		return 0, err
	}
	for {
		connection.lockState()
		length, err, ready, freed := connection.tryReadLocked(destination)
		notice := connection.peer.descriptor.notice('w')
		if err != errNetworkWouldBlock {
			connection.unlockState()
			record("net.read", networkArguments("tcp", connection.local.Port, connection.remote.Port, len(destination)), destination[:length], uint64(length), resultClass(err), 0, 0)
			if freed {
				gomadvfd.Notify(notice)
			}
			return length, err
		}
		deadline := connection.state.readDeadline
		if connection.deadlineExpired(deadline) {
			connection.unlockState()
			record("net.read", networkArguments("tcp", connection.local.Port, connection.remote.Port, len(destination)), nil, 0, resultClass(os.ErrDeadlineExceeded), 0, 0)
			return 0, os.ErrDeadlineExceeded
		}
		if !ready.IsZero() {
			changed := connection.state.shared.changed
			connection.unlockState()
			waitForChange(changed, earliestDeadline(deadline, ready))
			continue
		}
		changed := connection.state.shared.changed
		connection.unlockState()
		waitForChange(changed, deadline)
	}
}

func (connection *pairedConn) Write(source []byte) (int, error) {
	if err := connection.validate(); err != nil {
		return 0, err
	}
	written := 0
	input := source
	for len(source) != 0 {
		connection.lockState()
		length, err := connection.tryWriteLocked(source)
		notice := connection.peer.descriptor.notice('r')
		if err != errNetworkWouldBlock {
			connection.unlockState()
			written += length
			if length != 0 {
				gomadvfd.Notify(notice)
			}
			if err != nil {
				record("net.write", networkArguments("tcp", connection.local.Port, connection.remote.Port, len(input)), input[:written], uint64(written), resultClass(err), 0, 0)
				return written, err
			}
			source = source[length:]
			continue
		}
		deadline := connection.state.writeDeadline
		if connection.deadlineExpired(deadline) {
			connection.unlockState()
			record("net.write", networkArguments("tcp", connection.local.Port, connection.remote.Port, len(input)), input[:written], uint64(written), resultClass(os.ErrDeadlineExceeded), 0, 0)
			return written, os.ErrDeadlineExceeded
		}
		changed := connection.state.shared.changed
		connection.unlockState()
		waitForChange(changed, deadline)
	}
	record("net.write", networkArguments("tcp", connection.local.Port, connection.remote.Port, len(input)), input, uint64(written), 0, 0, 0)
	return written, nil
}

func (connection *pairedConn) Close() error {
	if err := connection.validate(); err != nil {
		return err
	}
	connection.lockState()
	if connection.state.closeCalled {
		connection.unlockState()
		record("net.close", networkArguments("tcp", connection.local.Port, connection.remote.Port), nil, 0, resultClass(ErrClosed), 0, 0)
		return ErrClosed
	}
	if connection.network != nil {
		var release uint64
		if connection.state.shared.pair.reserved {
			release = 1
		}
		if err := connection.network.commitBatchLocked([]simulationTransition{{Kind: "close", Source: connection.owner, Destination: connection.target, Connection: connection.identity, Outcome: "ok"}}, 0, release); err != nil {
			connection.unlockState()
			return err
		}
		connection.state.shared.pair.reserved = false
		for _, state := range []*connState{connection.state, connection.peer} {
			state.stalled = false
			state.stallEpoch++
			connection.network.armStallTimerLocked(state)
		}
		if !connection.network.links[simulationLinkKey(connection.owner.Node, connection.target.Node)].Enabled {
			connection.network.dropUndeliveredLocked(connection.peer)
		}
	}
	connection.state.closeCalled = true
	connection.shutdownLocked(true, true)
	notices := connection.descriptorNoticesLocked()
	connection.unlockState()
	record("net.close", networkArguments("tcp", connection.local.Port, connection.remote.Port), nil, 0, 0, 0, 0)
	gomadvfd.Notify(notices...)
	return nil
}

func (connection *pairedConn) CloseRead() error {
	if err := connection.validate(); err != nil {
		return err
	}
	connection.lockState()
	if connection.state.readClosed {
		connection.unlockState()
		return ErrClosed
	}
	connection.shutdownLocked(true, false)
	notices := connection.descriptorNoticesLocked()
	connection.unlockState()
	gomadvfd.Notify(notices...)
	return nil
}

func (connection *pairedConn) CloseWrite() error {
	if err := connection.validate(); err != nil {
		return err
	}
	connection.lockState()
	if connection.state.writeClosed {
		connection.unlockState()
		return ErrClosed
	}
	connection.shutdownLocked(false, true)
	notices := connection.descriptorNoticesLocked()
	connection.unlockState()
	gomadvfd.Notify(notices...)
	return nil
}

func (connection *pairedConn) LocalAddress() Address {
	return connection.local
}

func (connection *pairedConn) RemoteAddress() Address {
	return connection.remote
}

func (connection *pairedConn) SetDeadline(deadline time.Time) error {
	if err := connection.validate(); err != nil {
		return err
	}
	connection.lockState()
	connection.state.readDeadline = deadline
	connection.state.writeDeadline = deadline
	connection.state.shared.signal()
	connection.unlockState()
	return nil
}

func (connection *pairedConn) SetReadDeadline(deadline time.Time) error {
	if err := connection.validate(); err != nil {
		return err
	}
	connection.lockState()
	connection.state.readDeadline = deadline
	connection.state.shared.signal()
	connection.unlockState()
	return nil
}

func (connection *pairedConn) SetWriteDeadline(deadline time.Time) error {
	if err := connection.validate(); err != nil {
		return err
	}
	connection.lockState()
	connection.state.writeDeadline = deadline
	connection.state.shared.signal()
	connection.unlockState()
	return nil
}

func (connection *pairedConn) tryReadLocked(destination []byte) (int, error, time.Time, bool) {
	if len(destination) == 0 {
		return 0, nil, time.Time{}, false
	}
	if err := connection.terminalError(); err != nil {
		return 0, err, time.Time{}, false
	}
	if connection.state.readClosed {
		return 0, ErrClosed, time.Time{}, false
	}
	if len(connection.state.pending) == 0 && len(connection.state.incoming) != 0 {
		chunk := connection.state.incoming[0]
		if connection.network != nil && !connection.network.links[simulationLinkKey(chunk.source.Node, chunk.destination.Node)].Enabled {
			return 0, errNetworkWouldBlock, time.Time{}, false
		}
		if connection.nowTime().Before(chunk.ready) {
			return 0, errNetworkWouldBlock, chunk.ready, false
		}
		if connection.network != nil {
			transition := simulationTransition{Kind: "deliver", Source: chunk.source, Destination: chunk.destination, Connection: chunk.connection, Delivery: chunk.identity, Bytes: uint64(len(chunk.bytes)), DelayNanos: chunk.delayNanos, Outcome: "ok", PayloadSHA256: simulationPayloadSHA256(chunk.bytes)}
			if err := connection.network.commitTransitionLocked(transition); err != nil {
				return 0, err, time.Time{}, false
			}
			delete(connection.network.deliveries, chunk.identity)
		}
		connection.state.pending = chunk.bytes
		connection.state.incoming[0] = networkChunk{}
		connection.state.incoming = connection.state.incoming[1:]
	}
	if len(connection.state.pending) != 0 {
		length := copy(destination, connection.state.pending)
		connection.state.pending = connection.state.pending[length:]
		connection.state.retainedBytes -= uint64(length)
		if connection.network != nil {
			connection.network.pendingBytes -= uint64(length)
		}
		connection.state.shared.signal()
		return length, nil, time.Time{}, true
	}
	if connection.peer.writeClosed {
		return 0, io.EOF, time.Time{}, false
	}
	return 0, errNetworkWouldBlock, time.Time{}, false
}

func (connection *pairedConn) tryWriteLocked(source []byte) (int, error) {
	if len(source) == 0 {
		return 0, nil
	}
	if err := connection.terminalError(); err != nil {
		return 0, err
	}
	if connection.state.writeClosed || connection.peer.readClosed {
		return 0, ErrClosed
	}
	capacity := uint64(4 << 20)
	if connection.network != nil {
		capacity = connection.network.limits.ConnectionBytes
	}
	if connection.peer.retainedBytes >= capacity {
		return 0, errNetworkWouldBlock
	}
	length := min(len(source), maximumChunkBytes, int(capacity-connection.peer.retainedBytes))
	chunk := networkChunk{bytes: append([]byte(nil), source[:length]...)}
	if connection.network != nil {
		link, ok := connection.network.links[simulationLinkKey(connection.owner.Node, connection.target.Node)]
		outcome := "ok"
		if !ok || !link.Enabled {
			outcome = "held"
		}
		transition := simulationTransition{Kind: "write", Source: connection.owner, Destination: connection.target, Connection: connection.identity, Bytes: uint64(length), DelayNanos: link.DelayNanos, Outcome: outcome, PayloadSHA256: simulationPayloadSHA256(source[:length])}
		if uint64(len(connection.network.deliveries)) >= connection.network.limits.Deliveries || uint64(length) > connection.network.limits.Bytes-connection.network.pendingBytes {
			transition.Outcome = "capacity"
			if err := connection.network.commitTransitionLocked(transition); err != nil {
				return 0, err
			}
			return 0, ErrResourceExhausted
		}
		delivery := connection.network.nextDelivery + 1
		ready, err := checkedNetworkReadyTime(connection.nowTime(), link.DelayNanos)
		if err != nil {
			return 0, err
		}
		transition.Delivery = delivery
		var reservation uint64
		var due time.Time
		if outcome == "held" && !connection.peer.stalled {
			var err error
			due, err = strictNetworkStallDeadline(connection.nowTime(), connection.network.limits.StallNanos)
			if err != nil {
				return 0, err
			}
			if !connection.state.shared.pair.reserved {
				reservation = 1
			}
		}
		if err := connection.network.commitBatchLocked([]simulationTransition{transition}, reservation, 0); err != nil {
			return 0, err
		}
		connection.network.nextDelivery = delivery
		if outcome == "held" && !connection.peer.stalled {
			connection.peer.stalled = true
			connection.peer.stallStart = connection.nowTime()
			connection.peer.stallDue = due
			connection.peer.stallEpoch++
			connection.state.shared.pair.reserved = true
			connection.network.armStallTimerLocked(connection.peer)
		}
		chunk.identity = delivery
		chunk.connection = connection.identity
		chunk.source = connection.owner
		chunk.destination = connection.target
		chunk.ready = ready
		chunk.delayNanos = link.DelayNanos
		connection.network.deliveries[delivery] = simulationDelivery{identity: delivery, connection: connection.identity, source: connection.owner, destination: connection.target, bytes: uint64(length), delayNanos: link.DelayNanos}
		connection.network.pendingBytes += uint64(length)
	}
	connection.peer.incoming = append(connection.peer.incoming, chunk)
	connection.peer.retainedBytes += uint64(length)
	connection.state.shared.signal()
	return length, nil
}

func (connection *pairedConn) shutdownLocked(read, write bool) {
	if read {
		connection.state.readClosed = true
		if connection.network != nil {
			connection.network.dropStateDeliveriesLocked(connection.state)
		} else {
			connection.state.retainedBytes = 0
			connection.state.incoming = nil
			connection.state.pending = nil
		}
	}
	if write {
		connection.state.writeClosed = true
	}
	connection.state.shared.signal()
}

func (connection *pairedConn) descriptorNoticesLocked() []gomadvfd.Notice {
	return []gomadvfd.Notice{connection.state.descriptor.notice('r'), connection.state.descriptor.notice('w'), connection.peer.descriptor.notice('r'), connection.peer.descriptor.notice('w')}
}

func strictNetworkStallDeadline(start time.Time, limit uint64) (time.Time, error) {
	if limit == 0 || limit > math.MaxInt64-1 || start.UnixNano() < 0 || uint64(start.UnixNano()) > uint64(math.MaxInt64)-limit-1 {
		return time.Time{}, ErrUnsupported
	}
	return start.Add(time.Duration(limit) + 1), nil
}

func checkedNetworkReadyTime(start time.Time, delay uint64) (time.Time, error) {
	if delay > math.MaxInt64 || start.UnixNano() < 0 || uint64(start.UnixNano()) > uint64(math.MaxInt64)-delay {
		return time.Time{}, ErrUnsupported
	}
	return start.Add(time.Duration(delay)), nil
}

func (network *simulationNetwork) nowTime() time.Time { return time.Now() }

func (network *simulationNetwork) armStallTimerLocked(state *connState) {
	if state.stallTimer != nil {
		state.stallTimer.Stop()
		state.stallTimer = nil
	}
	if !state.stalled {
		return
	}
	pair, epoch, due := state.shared.pair, state.stallEpoch, state.stallDue
	state.stallTimer = time.AfterFunc(max(time.Until(due), 0), func() {
		network.Lock()
		pair.clientState.shared.Lock()
		err := network.expireStallLocked(pair, state, epoch, due)
		if err != nil {
			pair.clientState.shared.signal()
			network.signalLocked()
		}
		notices := []gomadvfd.Notice{pair.clientState.descriptor.notice('r'), pair.clientState.descriptor.notice('w'), pair.serverState.descriptor.notice('r'), pair.serverState.descriptor.notice('w')}
		pair.clientState.shared.Unlock()
		network.Unlock()
		gomadvfd.Notify(notices...)
	})
}

func (network *simulationNetwork) dropUndeliveredLocked(state *connState) {
	var discarded uint64
	for _, chunk := range state.incoming {
		discarded += uint64(len(chunk.bytes))
		delete(network.deliveries, chunk.identity)
	}
	state.incoming = nil
	state.retainedBytes -= discarded
	network.pendingBytes -= discarded
}

func (network *simulationNetwork) armDialTimerLocked(pending *simulationDial) {
	if pending.timer != nil {
		pending.timer.Stop()
		pending.timer = nil
	}
	if !pending.stalled {
		return
	}
	epoch, due := pending.epoch, pending.due
	pending.timer = time.AfterFunc(max(time.Until(due), 0), func() {
		network.Lock()
		if err := network.expireDialLocked(pending, epoch, due); err != nil {
			network.signalLocked()
		}
		network.Unlock()
	})
}

func (network *simulationNetwork) finishDialLocked(pending *simulationDial) {
	if pending == nil {
		return
	}
	pending.stalled = false
	pending.reserved = false
	pending.epoch++
	network.armDialTimerLocked(pending)
	delete(network.pendingDials, pending)
}

func (network *simulationNetwork) expireDialLocked(pending *simulationDial, epoch uint64, due time.Time) error {
	if network.finished || !network.pendingDials[pending] || !pending.stalled || !pending.reserved || pending.epoch != epoch || pending.due != due || network.nowTime().Before(due) {
		return nil
	}
	transition := simulationTransition{Kind: "dial", Source: pending.source, Destination: pending.destination, Outcome: "timeout"}
	if err := network.commitBatchLocked([]simulationTransition{transition}, 0, 1); err != nil {
		return err
	}
	pending.terminal = ErrStallTimeout
	network.finishDialLocked(pending)
	network.signalLocked()
	return nil
}
