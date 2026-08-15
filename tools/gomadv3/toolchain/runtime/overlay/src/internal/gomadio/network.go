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
	Port int
}

type Listener struct {
	address  Address
	once     sync.Once
	mu       sync.Mutex
	pending  []*Conn
	closed   bool
	deadline time.Time
	changed  chan struct{}
}

type Conn struct {
	local   Address
	remote  Address
	state   *connState
	peer    *connState
	readMu  sync.Mutex
	writeMu sync.Mutex
	pending []byte
	close   sync.Once
}

type connState struct {
	shared        *connShared
	incoming      [][]byte
	readClosed    bool
	writeClosed   bool
	readDeadline  time.Time
	writeDeadline time.Time
}

type connShared struct {
	sync.Mutex
	changed chan struct{}
}

var networkState = struct {
	sync.Mutex
	listeners        map[int]*Listener
	nextListenerPort int
	nextClientPort   int
}{listeners: make(map[int]*Listener), nextListenerPort: firstListenerPort, nextClientPort: firstClientPort}

func ListenTCP(network string, port int) (*Listener, error) {
	requestedPort := port
	if network != "tcp" && network != "tcp4" || port < 0 || port > maximumPort {
		record("net.listen", networkArguments(network, requestedPort), nil, 0, resultClass(ErrUnsupported), 0, 0)
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
	listener := &Listener{address: Address{Port: port}, changed: make(chan struct{})}
	networkState.listeners[port] = listener
	record("net.listen", networkArguments(network, requestedPort, port), nil, 0, 0, 0, 0)
	return listener, nil
}

func DialTCP(ctx context.Context, network string, port int) (*Conn, error) {
	if network != "tcp" && network != "tcp4" || port <= 0 || port > maximumPort {
		record("net.dial", networkArguments(network, port), nil, 0, resultClass(ErrUnsupported), 0, 0)
		return nil, ErrUnsupported
	}
	if err := ctx.Err(); err != nil {
		record("net.dial", networkArguments(network, port), nil, 0, resultClass(err), 0, 0)
		return nil, err
	}
	networkState.Lock()
	if networkState.nextClientPort > maximumPort {
		networkState.Unlock()
		record("net.dial", networkArguments(network, port), nil, 0, resultClass(ErrResourceExhausted), 0, 0)
		return nil, ErrResourceExhausted
	}
	clientAddress := Address{Port: networkState.nextClientPort}
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

func (connection *Conn) Read(destination []byte) (int, error) {
	connection.readMu.Lock()
	defer connection.readMu.Unlock()
	if len(destination) == 0 {
		record("net.read", networkArguments("tcp", connection.local.Port, connection.remote.Port, 0), nil, 0, 0, 0, 0)
		return 0, nil
	}
	for {
		connection.state.shared.Lock()
		if connection.state.readClosed {
			connection.state.shared.Unlock()
			record("net.read", networkArguments("tcp", connection.local.Port, connection.remote.Port, len(destination)), nil, 0, resultClass(ErrClosed), 0, 0)
			return 0, ErrClosed
		}
		if len(connection.pending) != 0 {
			length := copy(destination, connection.pending)
			connection.pending = connection.pending[length:]
			connection.state.shared.Unlock()
			record("net.read", networkArguments("tcp", connection.local.Port, connection.remote.Port, len(destination)), destination[:length], uint64(length), 0, 0, 0)
			return length, nil
		}
		if len(connection.state.incoming) != 0 {
			connection.pending = connection.state.incoming[0]
			connection.state.incoming = connection.state.incoming[1:]
			connection.state.shared.signal()
			connection.state.shared.Unlock()
			continue
		}
		if connection.peer.writeClosed {
			connection.state.shared.Unlock()
			record("net.read", networkArguments("tcp", connection.local.Port, connection.remote.Port, len(destination)), nil, 0, resultClass(io.EOF), 0, 0)
			return 0, io.EOF
		}
		deadline := connection.state.readDeadline
		if deadlineExpired(deadline) {
			connection.state.shared.Unlock()
			record("net.read", networkArguments("tcp", connection.local.Port, connection.remote.Port, len(destination)), nil, 0, resultClass(os.ErrDeadlineExceeded), 0, 0)
			return 0, os.ErrDeadlineExceeded
		}
		changed := connection.state.shared.changed
		connection.state.shared.Unlock()
		waitForChange(changed, deadline)
	}
}

func (connection *Conn) Write(source []byte) (int, error) {
	connection.writeMu.Lock()
	defer connection.writeMu.Unlock()
	written := 0
	input := source
	for len(source) != 0 {
		length := min(len(source), maximumChunkBytes)
		connection.state.shared.Lock()
		if connection.state.writeClosed || connection.peer.readClosed {
			connection.state.shared.Unlock()
			record("net.write", networkArguments("tcp", connection.local.Port, connection.remote.Port, len(input)), input[:written], uint64(written), resultClass(ErrClosed), 0, 0)
			return written, ErrClosed
		}
		if len(connection.peer.incoming) < maximumPendingChunks {
			connection.peer.incoming = append(connection.peer.incoming, append([]byte(nil), source[:length]...))
			connection.state.shared.signal()
			connection.state.shared.Unlock()
			written += length
			source = source[length:]
			continue
		}
		deadline := connection.state.writeDeadline
		if deadlineExpired(deadline) {
			connection.state.shared.Unlock()
			record("net.write", networkArguments("tcp", connection.local.Port, connection.remote.Port, len(input)), input[:written], uint64(written), resultClass(os.ErrDeadlineExceeded), 0, 0)
			return written, os.ErrDeadlineExceeded
		}
		changed := connection.state.shared.changed
		connection.state.shared.Unlock()
		waitForChange(changed, deadline)
	}
	record("net.write", networkArguments("tcp", connection.local.Port, connection.remote.Port, len(input)), input, uint64(written), 0, 0, 0)
	return written, nil
}

func (connection *Conn) Close() error {
	closed := false
	connection.close.Do(func() {
		closed = true
		connection.state.shared.Lock()
		connection.state.readClosed = true
		connection.state.writeClosed = true
		connection.state.shared.signal()
		connection.state.shared.Unlock()
	})
	if !closed {
		record("net.close", networkArguments("tcp", connection.local.Port, connection.remote.Port), nil, 0, resultClass(ErrClosed), 0, 0)
		return ErrClosed
	}
	record("net.close", networkArguments("tcp", connection.local.Port, connection.remote.Port), nil, 0, 0, 0, 0)
	return nil
}

func (connection *Conn) CloseRead() error {
	connection.state.shared.Lock()
	defer connection.state.shared.Unlock()
	if connection.state.readClosed {
		return ErrClosed
	}
	connection.state.readClosed = true
	connection.state.shared.signal()
	return nil
}

func (connection *Conn) CloseWrite() error {
	connection.state.shared.Lock()
	defer connection.state.shared.Unlock()
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
	connection.state.shared.Lock()
	connection.state.readDeadline = deadline
	connection.state.writeDeadline = deadline
	connection.state.shared.signal()
	connection.state.shared.Unlock()
	return nil
}

func (connection *Conn) SetReadDeadline(deadline time.Time) error {
	connection.state.shared.Lock()
	connection.state.readDeadline = deadline
	connection.state.shared.signal()
	connection.state.shared.Unlock()
	return nil
}

func (connection *Conn) SetWriteDeadline(deadline time.Time) error {
	connection.state.shared.Lock()
	connection.state.writeDeadline = deadline
	connection.state.shared.signal()
	connection.state.shared.Unlock()
	return nil
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
