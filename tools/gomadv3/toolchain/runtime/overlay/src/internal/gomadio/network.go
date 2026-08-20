// Copyright 2026 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gomadio

import (
	"context"
	"errors"
	"io"
	"os"
	"sync"
	"time"

	"internal/gomadmodelwire"
	"internal/gomadsim"
)

var (
	ErrAddressInUse      = errors.New("address already in use")
	ErrClosed            = errors.New("use of closed network connection")
	ErrConnectionRefused = errors.New("connection refused")
	ErrResourceExhausted = errors.New("network resources exhausted")
	ErrUnsupported       = errors.New("unsupported Gomad network operation")
)

const (
	firstListenerPort    = 20000
	firstClientPort      = 40000
	maximumPort          = 65535
	maximumPendingConns  = 64
	maximumPendingChunks = 64
	maximumChunkBytes    = 64 << 10
)

type Address struct {
	IP   string
	Port int
}

type Listener struct {
	address       Address
	processHandle uint64
	owner         simulationEndpoint
	network       *simulationNetwork
	once          sync.Once
	mu            sync.Mutex
	pending       []*Conn
	closed        bool
	deadline      time.Time
	changed       chan struct{}
}

type Conn struct {
	local         Address
	remote        Address
	processHandle uint64
	owner         simulationEndpoint
	target        simulationEndpoint
	network       *simulationNetwork
	identity      uint64
	state         *connState
	peer          *connState
	readMu        sync.Mutex
	writeMu       sync.Mutex
	pending       []byte
	close         sync.Once
}

type connState struct {
	shared        *connShared
	incoming      []networkChunk
	reset         bool
	readClosed    bool
	writeClosed   bool
	readDeadline  time.Time
	writeDeadline time.Time
}

type connShared struct {
	sync.Mutex
	changed chan struct{}
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

var networkState = struct {
	sync.Mutex
	listeners        map[int]*Listener
	nextListenerPort int
	nextClientPort   int
}{listeners: make(map[int]*Listener), nextListenerPort: firstListenerPort, nextClientPort: firstClientPort}

func ListenTCP(network, host string, port int) (*Listener, error) {
	requestedPort := port
	if network != "tcp" && network != "tcp4" || port < 0 || port > maximumPort {
		record("net.listen", networkArguments(network, requestedPort), nil, 0, resultClass(ErrUnsupported), 0, 0)
		return nil, ErrUnsupported
	}
	if gomadsim.ProcessRole() == 2 {
		return processNetworkListen(network, host, requestedPort)
	}
	if simulation, domain, err, handled := currentSimulationNetwork(); handled {
		if err != nil {
			return nil, err
		}
		return simulation.listen(network, host, requestedPort, domain)
	}
	if host != "" && host != "127.0.0.1" && host != "0.0.0.0" {
		return nil, ErrUnsupported
	}
	networkState.Lock()
	defer networkState.Unlock()
	if port == 0 {
		for networkState.nextListenerPort <= maximumPort {
			port = networkState.nextListenerPort
			networkState.nextListenerPort++
			if _, found := networkState.listeners[port]; !found {
				break
			}
		}
		if port == 0 || port > maximumPort {
			record("net.listen", networkArguments(network, requestedPort), nil, 0, resultClass(ErrResourceExhausted), 0, 0)
			return nil, ErrResourceExhausted
		}
	}
	if _, found := networkState.listeners[port]; found {
		record("net.listen", networkArguments(network, requestedPort, port), nil, 0, resultClass(ErrAddressInUse), 0, 0)
		return nil, ErrAddressInUse
	}
	listener := &Listener{address: Address{IP: "127.0.0.1", Port: port}, changed: make(chan struct{})}
	networkState.listeners[port] = listener
	record("net.listen", networkArguments(network, requestedPort, port), nil, 0, 0, 0, 0)
	return listener, nil
}

func DialTCP(ctx context.Context, network, host string, port int) (*Conn, error) {
	if network != "tcp" && network != "tcp4" || port <= 0 || port > maximumPort {
		record("net.dial", networkArguments(network, port), nil, 0, resultClass(ErrUnsupported), 0, 0)
		return nil, ErrUnsupported
	}
	if err := ctx.Err(); err != nil {
		record("net.dial", networkArguments(network, port), nil, 0, resultClass(err), 0, 0)
		return nil, err
	}
	if gomadsim.ProcessRole() == 2 {
		return processNetworkDial(ctx, network, host, port)
	}
	if simulation, domain, err, handled := currentSimulationNetwork(); handled {
		if err != nil {
			return nil, err
		}
		return simulation.dial(ctx, network, host, port, domain)
	}
	if host != "" && host != "127.0.0.1" {
		return nil, ErrUnsupported
	}
	networkState.Lock()
	if networkState.nextClientPort > maximumPort {
		networkState.Unlock()
		record("net.dial", networkArguments(network, port), nil, 0, resultClass(ErrResourceExhausted), 0, 0)
		return nil, ErrResourceExhausted
	}
	clientAddress := Address{IP: "127.0.0.1", Port: networkState.nextClientPort}
	networkState.nextClientPort++
	listener := networkState.listeners[port]
	if listener == nil {
		networkState.Unlock()
		record("net.dial", networkArguments(network, clientAddress.Port, port), nil, 0, resultClass(ErrConnectionRefused), 0, 0)
		return nil, ErrConnectionRefused
	}
	listener.mu.Lock()
	networkState.Unlock()
	for {
		if err := ctx.Err(); err != nil {
			listener.mu.Unlock()
			record("net.dial", networkArguments(network, clientAddress.Port, port), nil, 0, resultClass(err), 0, 0)
			return nil, err
		}
		if listener.closed {
			listener.mu.Unlock()
			record("net.dial", networkArguments(network, clientAddress.Port, port), nil, 0, resultClass(ErrConnectionRefused), 0, 0)
			return nil, ErrConnectionRefused
		}
		if len(listener.pending) < maximumPendingConns {
			clientState, serverState := newConnStates()
			client := &Conn{local: clientAddress, remote: listener.address, state: clientState, peer: serverState}
			server := &Conn{local: listener.address, remote: clientAddress, state: serverState, peer: clientState}
			listener.pending = append(listener.pending, server)
			listener.signal()
			listener.mu.Unlock()
			record("net.dial", networkArguments(network, clientAddress.Port, port), nil, 0, 0, 0, 0)
			return client, nil
		}
		changed := listener.changed
		listener.mu.Unlock()
		select {
		case <-changed:
		case <-ctx.Done():
		}
		listener.mu.Lock()
	}
}

func (listener *Listener) Accept() (*Conn, error) {
	if listener.processHandle != 0 {
		return processNetworkAccept(listener)
	}
	if listener.network != nil {
		connection, err := listener.network.accept(listener)
		if err != nil {
			record("net.accept", networkArguments("tcp", listener.address.Port), nil, 0, resultClass(err), 0, 0)
			return nil, err
		}
		record("net.accept", networkArguments("tcp", listener.address.Port, connection.remote.Port), nil, 0, 0, 0, 0)
		return connection, nil
	}
	for {
		listener.mu.Lock()
		if len(listener.pending) != 0 {
			connection := listener.pending[0]
			listener.pending = listener.pending[1:]
			listener.signal()
			listener.mu.Unlock()
			record("net.accept", networkArguments("tcp", listener.address.Port, connection.remote.Port), nil, 0, 0, 0, 0)
			return connection, nil
		}
		if listener.closed {
			listener.mu.Unlock()
			record("net.accept", networkArguments("tcp", listener.address.Port), nil, 0, resultClass(ErrClosed), 0, 0)
			return nil, ErrClosed
		}
		deadline := listener.deadline
		if deadlineExpired(deadline) {
			listener.mu.Unlock()
			record("net.accept", networkArguments("tcp", listener.address.Port), nil, 0, resultClass(os.ErrDeadlineExceeded), 0, 0)
			return nil, os.ErrDeadlineExceeded
		}
		changed := listener.changed
		listener.mu.Unlock()
		waitForChange(changed, deadline)
	}
}

func (listener *Listener) Close() error {
	if listener.processHandle != 0 {
		return processNetworkListenerClose(listener)
	}
	if listener.network != nil {
		err := listener.network.closeListener(listener)
		record("net.listener.close", networkArguments("tcp", listener.address.Port), nil, 0, resultClass(err), 0, 0)
		return err
	}
	closed := false
	listener.once.Do(func() {
		closed = true
		networkState.Lock()
		listener.mu.Lock()
		if networkState.listeners[listener.address.Port] == listener {
			delete(networkState.listeners, listener.address.Port)
		}
		listener.closed = true
		listener.signal()
		listener.mu.Unlock()
		networkState.Unlock()
	})
	if !closed {
		record("net.listener.close", networkArguments("tcp", listener.address.Port), nil, 0, resultClass(ErrClosed), 0, 0)
		return ErrClosed
	}
	record("net.listener.close", networkArguments("tcp", listener.address.Port), nil, 0, 0, 0, 0)
	return nil
}

func (listener *Listener) Address() Address {
	return listener.address
}

func (listener *Listener) SetDeadline(deadline time.Time) error {
	if listener.processHandle != 0 {
		return processNetworkListenerSetDeadline(listener, deadline)
	}
	if listener.network != nil {
		if err := validateSimulationEndpoint(listener.network, listener.owner); err != nil {
			return err
		}
	}
	listener.mu.Lock()
	listener.deadline = deadline
	listener.signal()
	listener.mu.Unlock()
	return nil
}

func (listener *Listener) signal() {
	close(listener.changed)
	listener.changed = make(chan struct{})
}

func newConnStates() (*connState, *connState) {
	shared := &connShared{changed: make(chan struct{})}
	return &connState{shared: shared}, &connState{shared: shared}
}

func (connection *Conn) lockState() {
	if connection.network != nil {
		connection.network.Lock()
	}
	connection.state.shared.Lock()
}

func (connection *Conn) unlockState() {
	connection.state.shared.Unlock()
	if connection.network != nil {
		connection.network.Unlock()
	}
}

func (connection *Conn) Read(destination []byte) (int, error) {
	connection.readMu.Lock()
	defer connection.readMu.Unlock()
	if connection.processHandle != 0 {
		return processNetworkConnRead(connection, destination)
	}
	if len(destination) == 0 {
		record("net.read", networkArguments("tcp", connection.local.Port, connection.remote.Port, 0), nil, 0, 0, 0, 0)
		return 0, nil
	}
	if connection.network != nil {
		if err := validateSimulationEndpoint(connection.network, connection.owner); err != nil {
			return 0, err
		}
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
			if connection.network != nil {
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
			}
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

func (connection *Conn) Write(source []byte) (int, error) {
	connection.writeMu.Lock()
	defer connection.writeMu.Unlock()
	if connection.processHandle != 0 {
		return processNetworkConnWrite(connection, source)
	}
	if connection.network != nil {
		if err := validateSimulationEndpoint(connection.network, connection.owner); err != nil {
			return 0, err
		}
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
			if connection.network == nil {
				connection.peer.incoming = append(connection.peer.incoming, networkChunk{bytes: append([]byte(nil), source[:length]...)})
				connection.state.shared.signal()
				connection.unlockState()
				written += length
				source = source[length:]
				continue
			}
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

func (connection *Conn) Close() error {
	if connection.processHandle != 0 {
		return processNetworkConnOperation(connection, gomadmodelwire.NetworkConnClose, time.Time{})
	}
	if connection.network != nil {
		if err := validateSimulationEndpoint(connection.network, connection.owner); err != nil {
			return err
		}
	}
	closed := false
	var closeErr error
	connection.close.Do(func() {
		closed = true
		connection.lockState()
		if connection.network != nil {
			if err := connection.network.commitTransitionLocked(simulationTransition{Kind: "close", Source: connection.owner, Destination: connection.target, Connection: connection.identity, Outcome: "ok"}); err != nil {
				connection.unlockState()
				closed = false
				closeErr = err
				return
			}
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

func (connection *Conn) CloseRead() error {
	if connection.processHandle != 0 {
		return processNetworkConnOperation(connection, gomadmodelwire.NetworkConnCloseRead, time.Time{})
	}
	if connection.network != nil {
		if err := validateSimulationEndpoint(connection.network, connection.owner); err != nil {
			return err
		}
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

func (connection *Conn) CloseWrite() error {
	if connection.processHandle != 0 {
		return processNetworkConnOperation(connection, gomadmodelwire.NetworkConnCloseWrite, time.Time{})
	}
	if connection.network != nil {
		if err := validateSimulationEndpoint(connection.network, connection.owner); err != nil {
			return err
		}
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

func (connection *Conn) LocalAddress() Address {
	return connection.local
}

func (connection *Conn) RemoteAddress() Address {
	return connection.remote
}

func (connection *Conn) SetDeadline(deadline time.Time) error {
	if connection.processHandle != 0 {
		return processNetworkConnOperation(connection, gomadmodelwire.NetworkConnSetDeadline, deadline)
	}
	if connection.network != nil {
		if err := validateSimulationEndpoint(connection.network, connection.owner); err != nil {
			return err
		}
	}
	connection.lockState()
	connection.state.readDeadline = deadline
	connection.state.writeDeadline = deadline
	connection.state.shared.signal()
	connection.unlockState()
	return nil
}

func (connection *Conn) SetReadDeadline(deadline time.Time) error {
	if connection.processHandle != 0 {
		return processNetworkConnOperation(connection, gomadmodelwire.NetworkConnSetReadDeadline, deadline)
	}
	if connection.network != nil {
		if err := validateSimulationEndpoint(connection.network, connection.owner); err != nil {
			return err
		}
	}
	connection.lockState()
	connection.state.readDeadline = deadline
	connection.state.shared.signal()
	connection.unlockState()
	return nil
}

func (connection *Conn) SetWriteDeadline(deadline time.Time) error {
	if connection.processHandle != 0 {
		return processNetworkConnOperation(connection, gomadmodelwire.NetworkConnSetWriteDeadline, deadline)
	}
	if connection.network != nil {
		if err := validateSimulationEndpoint(connection.network, connection.owner); err != nil {
			return err
		}
	}
	connection.lockState()
	connection.state.writeDeadline = deadline
	connection.state.shared.signal()
	connection.unlockState()
	return nil
}

func earliestDeadline(left, right time.Time) time.Time {
	if left.IsZero() || right.Before(left) {
		return right
	}
	return left
}

func (shared *connShared) signal() {
	close(shared.changed)
	shared.changed = make(chan struct{})
}

func deadlineExpired(deadline time.Time) bool {
	return !deadline.IsZero() && !time.Now().Before(deadline)
}

func waitForChange(changed <-chan struct{}, deadline time.Time) {
	timer, timeout := deadlineTimer(deadline)
	select {
	case <-changed:
		stopTimer(timer)
	case <-timeout:
	}
}

func deadlineTimer(deadline time.Time) (*time.Timer, <-chan time.Time) {
	if deadline.IsZero() {
		return nil, nil
	}
	timer := time.NewTimer(max(time.Until(deadline), 0))
	return timer, timer.C
}

func stopTimer(timer *time.Timer) {
	if timer != nil {
		timer.Stop()
	}
}
