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

	"internal/gomadsim"
	"internal/poll"
)

var (
	ErrAddressInUse      = errors.New("address already in use")
	ErrClosed            = poll.ErrNetClosing
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

type listenerImplementation interface {
	Accept() (*Conn, error)
	Close() error
	Address() Address
	SetDeadline(time.Time) error
}
type connImplementation interface {
	Read([]byte) (int, error)
	Write([]byte) (int, error)
	Close() error
	CloseRead() error
	CloseWrite() error
	LocalAddress() Address
	RemoteAddress() Address
	SetDeadline(time.Time) error
	SetReadDeadline(time.Time) error
	SetWriteDeadline(time.Time) error
}
type Listener struct{ implementation listenerImplementation }
type Conn struct {
	implementation connImplementation
	readMu         sync.Mutex
	writeMu        sync.Mutex
}

func (listener *Listener) Accept() (*Conn, error) { return listener.implementation.Accept() }
func (listener *Listener) Close() error           { return listener.implementation.Close() }
func (listener *Listener) Address() Address       { return listener.implementation.Address() }
func (listener *Listener) SetDeadline(deadline time.Time) error {
	return listener.implementation.SetDeadline(deadline)
}
func (connection *Conn) Read(destination []byte) (int, error) {
	connection.readMu.Lock()
	defer connection.readMu.Unlock()
	return connection.implementation.Read(destination)
}
func (connection *Conn) Write(source []byte) (int, error) {
	connection.writeMu.Lock()
	defer connection.writeMu.Unlock()
	return connection.implementation.Write(source)
}
func (connection *Conn) Close() error           { return connection.implementation.Close() }
func (connection *Conn) CloseRead() error       { return connection.implementation.CloseRead() }
func (connection *Conn) CloseWrite() error      { return connection.implementation.CloseWrite() }
func (connection *Conn) LocalAddress() Address  { return connection.implementation.LocalAddress() }
func (connection *Conn) RemoteAddress() Address { return connection.implementation.RemoteAddress() }
func (connection *Conn) SetDeadline(deadline time.Time) error {
	return connection.implementation.SetDeadline(deadline)
}
func (connection *Conn) SetReadDeadline(deadline time.Time) error {
	return connection.implementation.SetReadDeadline(deadline)
}
func (connection *Conn) SetWriteDeadline(deadline time.Time) error {
	return connection.implementation.SetWriteDeadline(deadline)
}

type pairedConn struct {
	local   Address
	remote  Address
	state   *connState
	peer    *connState
	pending []byte
	close   sync.Once
}
type standaloneConn struct{ pairedConn }
type standaloneListener struct {
	address  Address
	once     sync.Once
	mu       sync.Mutex
	pending  []*Conn
	closed   bool
	deadline time.Time
	changed  chan struct{}
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
	listeners        map[int]*standaloneListener
	nextListenerPort int
	nextClientPort   int
}{listeners: make(map[int]*standaloneListener), nextListenerPort: firstListenerPort, nextClientPort: firstClientPort}

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
	listener := &standaloneListener{address: Address{IP: "127.0.0.1", Port: port}, changed: make(chan struct{})}
	networkState.listeners[port] = listener
	record("net.listen", networkArguments(network, requestedPort, port), nil, 0, 0, 0, 0)
	return &Listener{implementation: listener}, nil
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
			client := &Conn{implementation: &standaloneConn{pairedConn{local: clientAddress, remote: listener.address, state: clientState, peer: serverState}}}
			server := &Conn{implementation: &standaloneConn{pairedConn{local: listener.address, remote: clientAddress, state: serverState, peer: clientState}}}
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

func (listener *standaloneListener) Accept() (*Conn, error) {
	for {
		listener.mu.Lock()
		if len(listener.pending) != 0 {
			connection := listener.pending[0]
			listener.pending = listener.pending[1:]
			listener.signal()
			listener.mu.Unlock()
			record("net.accept", networkArguments("tcp", listener.address.Port, connection.RemoteAddress().Port), nil, 0, 0, 0, 0)
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

func (listener *standaloneListener) Close() error {
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

func (listener *standaloneListener) Address() Address {
	return listener.address
}

func (listener *standaloneListener) SetDeadline(deadline time.Time) error {
	listener.mu.Lock()
	listener.deadline = deadline
	listener.signal()
	listener.mu.Unlock()
	return nil
}

func (listener *standaloneListener) signal() {
	close(listener.changed)
	listener.changed = make(chan struct{})
}

func (connection *standaloneConn) lockState() {
	connection.state.shared.Lock()
}

func (connection *standaloneConn) unlockState() {
	connection.state.shared.Unlock()
}

func (connection *standaloneConn) Read(destination []byte) (int, error) {
	if len(destination) == 0 {
		record("net.read", networkArguments("tcp", connection.local.Port, connection.remote.Port, 0), nil, 0, 0, 0, 0)
		return 0, nil
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

func (connection *standaloneConn) Write(source []byte) (int, error) {
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
			connection.peer.incoming = append(connection.peer.incoming, networkChunk{bytes: append([]byte(nil), source[:length]...)})
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

func (connection *standaloneConn) Close() error {
	closed := false
	connection.close.Do(func() {
		closed = true
		connection.lockState()

		connection.state.readClosed = true
		connection.state.writeClosed = true
		connection.state.shared.signal()
		connection.unlockState()
	})
	if !closed {
		record("net.close", networkArguments("tcp", connection.local.Port, connection.remote.Port), nil, 0, resultClass(ErrClosed), 0, 0)
		return ErrClosed
	}
	record("net.close", networkArguments("tcp", connection.local.Port, connection.remote.Port), nil, 0, 0, 0, 0)
	return nil
}

func (connection *standaloneConn) CloseRead() error {
	connection.lockState()
	defer connection.unlockState()
	if connection.state.readClosed {
		return ErrClosed
	}
	connection.state.readClosed = true
	connection.state.shared.signal()
	return nil
}

func (connection *standaloneConn) CloseWrite() error {
	connection.lockState()
	defer connection.unlockState()
	if connection.state.writeClosed {
		return ErrClosed
	}
	connection.state.writeClosed = true
	connection.state.shared.signal()
	return nil
}

func (connection *standaloneConn) LocalAddress() Address {
	return connection.local
}

func (connection *standaloneConn) RemoteAddress() Address {
	return connection.remote
}

func (connection *standaloneConn) SetDeadline(deadline time.Time) error {
	connection.lockState()
	connection.state.readDeadline = deadline
	connection.state.writeDeadline = deadline
	connection.state.shared.signal()
	connection.unlockState()
	return nil
}

func (connection *standaloneConn) SetReadDeadline(deadline time.Time) error {
	connection.lockState()
	connection.state.readDeadline = deadline
	connection.state.shared.signal()
	connection.unlockState()
	return nil
}

func (connection *standaloneConn) SetWriteDeadline(deadline time.Time) error {
	connection.lockState()
	connection.state.writeDeadline = deadline
	connection.state.shared.signal()
	connection.unlockState()
	return nil
}

func newConnStates() (*connState, *connState) {
	shared := &connShared{changed: make(chan struct{})}
	return &connState{shared: shared}, &connState{shared: shared}
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
