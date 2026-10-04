// Copyright 2026 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gomadio

import (
	"io"
	"os"
	"sync"
	"time"
)

type simulationListener struct {
	address  Address
	owner    simulationEndpoint
	network  *simulationNetwork
	mu       sync.Mutex
	pending  []*simulationConn
	closed   bool
	deadline time.Time
	changed  chan struct{}
}
type simulationConn struct {
	pairedConn
	owner    simulationEndpoint
	target   simulationEndpoint
	network  *simulationNetwork
	identity uint64
}

func (listener *simulationListener) Accept() (*Conn, error) {
	connection, err := listener.network.accept(listener)
	if err != nil {
		record("net.accept", networkArguments("tcp", listener.address.Port), nil, 0, resultClass(err), 0, 0)
		return nil, err
	}
	record("net.accept", networkArguments("tcp", listener.address.Port, connection.remote.Port), nil, 0, 0, 0, 0)
	return &Conn{implementation: connection}, nil
}
func (listener *simulationListener) Close() error {
	err := listener.network.closeListener(listener)
	record("net.listener.close", networkArguments("tcp", listener.address.Port), nil, 0, resultClass(err), 0, 0)
	return err
}
func (listener *simulationListener) Address() Address { return listener.address }
func (listener *simulationListener) SetDeadline(deadline time.Time) error {
	if err := validateSimulationEndpoint(listener.network, listener.owner); err != nil {
		return err
	}
	listener.mu.Lock()
	listener.deadline = deadline
	listener.signal()
	listener.mu.Unlock()
	return nil
}
func (listener *simulationListener) signal() {
	close(listener.changed)
	listener.changed = make(chan struct{})
}
func (connection *simulationConn) lockState() {
	connection.network.Lock()

	connection.state.shared.Lock()
}

func (connection *simulationConn) unlockState() {
	connection.state.shared.Unlock()

	connection.network.Unlock()
}

func (connection *simulationConn) Read(destination []byte) (int, error) {
	if len(destination) == 0 {
		record("net.read", networkArguments("tcp", connection.local.Port, connection.remote.Port, 0), nil, 0, 0, 0, 0)
		return 0, nil
	}

	if err := validateSimulationEndpoint(connection.network, connection.owner); err != nil {
		return 0, err
	}

	for {
		connection.lockState()
		if connection.state.reset {
			connection.unlockState()
			record("net.read", networkArguments("tcp", connection.local.Port, connection.remote.Port, len(destination)), nil, 0, resultClass(ErrClosed), 0, 0)
			return 0, ErrClosed
		}
		if connection.state.readClosed {
			connection.unlockState()
			record("net.read", networkArguments("tcp", connection.local.Port, connection.remote.Port, len(destination)), nil, 0, resultClass(ErrClosed), 0, 0)
			return 0, ErrClosed
		}
		if len(connection.pending) != 0 {
			length := copy(destination, connection.pending)
			connection.pending = connection.pending[length:]
			connection.unlockState()
			record("net.read", networkArguments("tcp", connection.local.Port, connection.remote.Port, len(destination)), destination[:length], uint64(length), 0, 0, 0)
			return length, nil
		}
		if len(connection.state.incoming) != 0 {
			chunk := connection.state.incoming[0]
			deadline := connection.state.readDeadline
			if time.Now().Before(chunk.ready) {
				changed := connection.state.shared.changed
				connection.unlockState()
				waitForChange(changed, earliestDeadline(deadline, chunk.ready))
				continue
			}

			transition := simulationTransition{
				Kind: "deliver", Source: chunk.source, Destination: chunk.destination,
				Connection: chunk.connection, Delivery: chunk.identity, Bytes: uint64(len(chunk.bytes)),
				DelayNanos: chunk.delayNanos, Outcome: "ok", PayloadSHA256: simulationPayloadSHA256(chunk.bytes),
			}
			if err := connection.network.commitTransitionLocked(transition); err != nil {
				connection.unlockState()
				return 0, err
			}
			delete(connection.network.deliveries, chunk.identity)
			connection.network.pendingBytes -= uint64(len(chunk.bytes))

			connection.pending = chunk.bytes
			connection.state.incoming = connection.state.incoming[1:]
			connection.state.shared.signal()
			connection.unlockState()
			continue
		}
		if connection.peer.writeClosed {
			connection.unlockState()
			record("net.read", networkArguments("tcp", connection.local.Port, connection.remote.Port, len(destination)), nil, 0, resultClass(io.EOF), 0, 0)
			return 0, io.EOF
		}
		deadline := connection.state.readDeadline
		if deadlineExpired(deadline) {
			connection.unlockState()
			record("net.read", networkArguments("tcp", connection.local.Port, connection.remote.Port, len(destination)), nil, 0, resultClass(os.ErrDeadlineExceeded), 0, 0)
			return 0, os.ErrDeadlineExceeded
		}
		changed := connection.state.shared.changed
		connection.unlockState()
		waitForChange(changed, deadline)
	}
}

func (connection *simulationConn) Write(source []byte) (int, error) {
	if err := validateSimulationEndpoint(connection.network, connection.owner); err != nil {
		return 0, err
	}

	written := 0
	input := source
	for len(source) != 0 {
		length := min(len(source), maximumChunkBytes)
		connection.lockState()
		if connection.state.reset || connection.peer.reset || connection.state.writeClosed || connection.peer.readClosed {
			connection.unlockState()
			record("net.write", networkArguments("tcp", connection.local.Port, connection.remote.Port, len(input)), input[:written], uint64(written), resultClass(ErrClosed), 0, 0)
			return written, ErrClosed
		}
		if len(connection.peer.incoming) < maximumPendingChunks {
			link, ok := connection.network.links[simulationLinkKey(connection.owner.Node, connection.target.Node)]
			outcome := "ok"
			if !ok || !link.Enabled {
				outcome = "partition_drop"
			}
			if outcome == "ok" && (uint64(len(connection.network.deliveries)) >= connection.network.limits.Deliveries || uint64(length) > connection.network.limits.Bytes-connection.network.pendingBytes) {
				transition := simulationTransition{Kind: "write", Source: connection.owner, Destination: connection.target, Connection: connection.identity, Bytes: uint64(length), DelayNanos: link.DelayNanos, Outcome: "capacity", PayloadSHA256: simulationPayloadSHA256(source[:length])}
				err := connection.network.commitTransitionLocked(transition)
				connection.unlockState()
				if err != nil {
					return written, err
				}
				return written, ErrResourceExhausted
			}
			delivery := connection.network.nextDelivery + 1
			transition := simulationTransition{Kind: "write", Source: connection.owner, Destination: connection.target, Connection: connection.identity, Delivery: delivery, Bytes: uint64(length), DelayNanos: link.DelayNanos, Outcome: outcome, PayloadSHA256: simulationPayloadSHA256(source[:length])}
			if err := connection.network.commitTransitionLocked(transition); err != nil {
				connection.unlockState()
				return written, err
			}
			connection.network.nextDelivery = delivery
			if outcome == "ok" {
				ready := time.Now().Add(time.Duration(link.DelayNanos))
				chunk := networkChunk{identity: delivery, connection: connection.identity, source: connection.owner, destination: connection.target, bytes: append([]byte(nil), source[:length]...), ready: ready, delayNanos: link.DelayNanos}
				connection.peer.incoming = append(connection.peer.incoming, chunk)
				connection.network.deliveries[delivery] = simulationDelivery{identity: delivery, connection: connection.identity, source: connection.owner, destination: connection.target, bytes: uint64(length), delayNanos: link.DelayNanos}
				connection.network.pendingBytes += uint64(length)
			}
			connection.state.shared.signal()
			connection.unlockState()
			written += length
			source = source[length:]
			continue
		}
		deadline := connection.state.writeDeadline
		if deadlineExpired(deadline) {
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

func (connection *simulationConn) Close() error {
	if err := validateSimulationEndpoint(connection.network, connection.owner); err != nil {
		return err
	}

	closed := false
	var closeErr error
	connection.close.Do(func() {
		closed = true
		connection.lockState()

		if err := connection.network.commitTransitionLocked(simulationTransition{Kind: "close", Source: connection.owner, Destination: connection.target, Connection: connection.identity, Outcome: "ok"}); err != nil {
			connection.unlockState()
			closed = false
			closeErr = err
			return
		}

		connection.state.readClosed = true
		connection.state.writeClosed = true
		connection.state.shared.signal()
		connection.unlockState()
	})
	if closeErr != nil {
		return closeErr
	}
	if !closed {
		record("net.close", networkArguments("tcp", connection.local.Port, connection.remote.Port), nil, 0, resultClass(ErrClosed), 0, 0)
		return ErrClosed
	}
	record("net.close", networkArguments("tcp", connection.local.Port, connection.remote.Port), nil, 0, 0, 0, 0)
	return nil
}

func (connection *simulationConn) CloseRead() error {
	if err := validateSimulationEndpoint(connection.network, connection.owner); err != nil {
		return err
	}

	connection.lockState()
	defer connection.unlockState()
	if connection.state.readClosed {
		return ErrClosed
	}
	connection.state.readClosed = true
	connection.state.shared.signal()
	return nil
}

func (connection *simulationConn) CloseWrite() error {
	if err := validateSimulationEndpoint(connection.network, connection.owner); err != nil {
		return err
	}

	connection.lockState()
	defer connection.unlockState()
	if connection.state.writeClosed {
		return ErrClosed
	}
	connection.state.writeClosed = true
	connection.state.shared.signal()
	return nil
}

func (connection *simulationConn) LocalAddress() Address {
	return connection.local
}

func (connection *simulationConn) RemoteAddress() Address {
	return connection.remote
}

func (connection *simulationConn) SetDeadline(deadline time.Time) error {
	if err := validateSimulationEndpoint(connection.network, connection.owner); err != nil {
		return err
	}

	connection.lockState()
	connection.state.readDeadline = deadline
	connection.state.writeDeadline = deadline
	connection.state.shared.signal()
	connection.unlockState()
	return nil
}

func (connection *simulationConn) SetReadDeadline(deadline time.Time) error {
	if err := validateSimulationEndpoint(connection.network, connection.owner); err != nil {
		return err
	}

	connection.lockState()
	connection.state.readDeadline = deadline
	connection.state.shared.signal()
	connection.unlockState()
	return nil
}

func (connection *simulationConn) SetWriteDeadline(deadline time.Time) error {
	if err := validateSimulationEndpoint(connection.network, connection.owner); err != nil {
		return err
	}

	connection.lockState()
	connection.state.writeDeadline = deadline
	connection.state.shared.signal()
	connection.unlockState()
	return nil
}
