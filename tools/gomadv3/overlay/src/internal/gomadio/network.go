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
	ErrUnsupported       = errors.New("unsupported Gomad network operation")
)

const (
	firstListenerPort = 20000
	firstClientPort   = 40000
	maximumChunkBytes = 64 << 10
)

type Address struct {
	Port int
}

type Listener struct {
	address  Address
	pending  chan *Conn
	closed   chan struct{}
	once     sync.Once
	mu       sync.Mutex
	deadline time.Time
	wake     chan struct{}
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
	incoming      chan []byte
	readClosed    chan struct{}
	writeClosed   chan struct{}
	closeRead     sync.Once
	closeWrite    sync.Once
	mu            sync.Mutex
	readDeadline  time.Time
	writeDeadline time.Time
	readWake      chan struct{}
	writeWake     chan struct{}
}

var networkState = struct {
	sync.Mutex
	listeners        map[int]*Listener
	nextListenerPort int
	nextClientPort   int
}{listeners: make(map[int]*Listener), nextListenerPort: firstListenerPort, nextClientPort: firstClientPort}

func ListenTCP(network string, port int) (*Listener, error) {
	requestedPort := port
	if network != "tcp" && network != "tcp4" || port < 0 || port > 65535 {
		record("net.listen", networkArguments(network, requestedPort), nil, 0, resultClass(ErrUnsupported), 0, 0)
		return nil, ErrUnsupported
	}
	networkState.Lock()
	defer networkState.Unlock()
	if port == 0 {
		for {
			port = networkState.nextListenerPort
			networkState.nextListenerPort++
			if _, found := networkState.listeners[port]; !found {
				break
			}
		}
	}
	if _, found := networkState.listeners[port]; found {
		record("net.listen", networkArguments(network, requestedPort, port), nil, 0, resultClass(ErrAddressInUse), 0, 0)
		return nil, ErrAddressInUse
	}
	listener := &Listener{address: Address{Port: port}, pending: make(chan *Conn, 64), closed: make(chan struct{}), wake: make(chan struct{}, 1)}
	networkState.listeners[port] = listener
	record("net.listen", networkArguments(network, requestedPort, port), nil, 0, 0, 0, 0)
	return listener, nil
}

func DialTCP(ctx context.Context, network string, port int) (*Conn, error) {
	if network != "tcp" && network != "tcp4" || port <= 0 || port > 65535 {
		record("net.dial", networkArguments(network, port), nil, 0, resultClass(ErrUnsupported), 0, 0)
		return nil, ErrUnsupported
	}
	select {
	case <-ctx.Done():
		record("net.dial", networkArguments(network, port), nil, 0, resultClass(ctx.Err()), 0, 0)
		return nil, ctx.Err()
	default:
	}
	networkState.Lock()
	listener := networkState.listeners[port]
	clientAddress := Address{Port: networkState.nextClientPort}
	networkState.nextClientPort++
	networkState.Unlock()
	if listener == nil {
		record("net.dial", networkArguments(network, clientAddress.Port, port), nil, 0, resultClass(ErrConnectionRefused), 0, 0)
		return nil, ErrConnectionRefused
	}
	clientState := newConnState()
	serverState := newConnState()
	client := &Conn{local: clientAddress, remote: listener.address, state: clientState, peer: serverState}
	server := &Conn{local: listener.address, remote: clientAddress, state: serverState, peer: clientState}
	select {
	case listener.pending <- server:
		record("net.dial", networkArguments(network, clientAddress.Port, port), nil, 0, 0, 0, 0)
		return client, nil
	case <-listener.closed:
		record("net.dial", networkArguments(network, clientAddress.Port, port), nil, 0, resultClass(ErrConnectionRefused), 0, 0)
		return nil, ErrConnectionRefused
	case <-ctx.Done():
		record("net.dial", networkArguments(network, clientAddress.Port, port), nil, 0, resultClass(ctx.Err()), 0, 0)
		return nil, ctx.Err()
	}
}

func (listener *Listener) Accept() (*Conn, error) {
	for {
		listener.mu.Lock()
		deadline := listener.deadline
		wake := listener.wake
		listener.mu.Unlock()
		timer, timeout := deadlineTimer(deadline)
		select {
		case connection := <-listener.pending:
			stopTimer(timer)
			record("net.accept", networkArguments("tcp", listener.address.Port, connection.remote.Port), nil, 0, 0, 0, 0)
			return connection, nil
		case <-listener.closed:
			stopTimer(timer)
			record("net.accept", networkArguments("tcp", listener.address.Port), nil, 0, resultClass(ErrClosed), 0, 0)
			return nil, ErrClosed
		case <-wake:
			stopTimer(timer)
		case <-timeout:
			record("net.accept", networkArguments("tcp", listener.address.Port), nil, 0, resultClass(os.ErrDeadlineExceeded), 0, 0)
			return nil, os.ErrDeadlineExceeded
		}
	}
}

func (listener *Listener) Close() error {
	closed := false
	listener.once.Do(func() {
		closed = true
		networkState.Lock()
		if networkState.listeners[listener.address.Port] == listener {
			delete(networkState.listeners, listener.address.Port)
		}
		networkState.Unlock()
		close(listener.closed)
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
	notify(listener.wake)
	listener.mu.Unlock()
	return nil
}

func newConnState() *connState {
	return &connState{
		incoming: make(chan []byte, 64), readClosed: make(chan struct{}), writeClosed: make(chan struct{}),
		readWake: make(chan struct{}, 1), writeWake: make(chan struct{}, 1),
	}
}

func (connection *Conn) Read(destination []byte) (int, error) {
	connection.readMu.Lock()
	defer connection.readMu.Unlock()
	if len(destination) == 0 {
		record("net.read", networkArguments("tcp", connection.local.Port, connection.remote.Port, 0), nil, 0, 0, 0, 0)
		return 0, nil
	}
	for {
		if len(connection.pending) != 0 {
			length := copy(destination, connection.pending)
			connection.pending = connection.pending[length:]
			record("net.read", networkArguments("tcp", connection.local.Port, connection.remote.Port, len(destination)), destination[:length], uint64(length), 0, 0, 0)
			return length, nil
		}
		select {
		case chunk := <-connection.state.incoming:
			connection.pending = chunk
			continue
		default:
		}
		deadline, wake := connection.state.deadline(true)
		timer, timeout := deadlineTimer(deadline)
		select {
		case chunk := <-connection.state.incoming:
			stopTimer(timer)
			connection.pending = chunk
		case <-connection.state.readClosed:
			stopTimer(timer)
			record("net.read", networkArguments("tcp", connection.local.Port, connection.remote.Port, len(destination)), nil, 0, resultClass(ErrClosed), 0, 0)
			return 0, ErrClosed
		case <-connection.peer.writeClosed:
			stopTimer(timer)
			record("net.read", networkArguments("tcp", connection.local.Port, connection.remote.Port, len(destination)), nil, 0, resultClass(io.EOF), 0, 0)
			return 0, io.EOF
		case <-wake:
			stopTimer(timer)
		case <-timeout:
			record("net.read", networkArguments("tcp", connection.local.Port, connection.remote.Port, len(destination)), nil, 0, resultClass(os.ErrDeadlineExceeded), 0, 0)
			return 0, os.ErrDeadlineExceeded
		}
	}
}

func (connection *Conn) Write(source []byte) (int, error) {
	connection.writeMu.Lock()
	defer connection.writeMu.Unlock()
	written := 0
	input := source
	for len(source) != 0 {
		length := min(len(source), maximumChunkBytes)
		chunk := append([]byte(nil), source[:length]...)
		deadline, wake := connection.state.deadline(false)
		timer, timeout := deadlineTimer(deadline)
		select {
		case connection.peer.incoming <- chunk:
			stopTimer(timer)
			written += length
			source = source[length:]
		case <-connection.state.writeClosed:
			stopTimer(timer)
			record("net.write", networkArguments("tcp", connection.local.Port, connection.remote.Port, len(input)), input[:written], uint64(written), resultClass(ErrClosed), 0, 0)
			return written, ErrClosed
		case <-connection.peer.readClosed:
			stopTimer(timer)
			record("net.write", networkArguments("tcp", connection.local.Port, connection.remote.Port, len(input)), input[:written], uint64(written), resultClass(ErrClosed), 0, 0)
			return written, ErrClosed
		case <-wake:
			stopTimer(timer)
		case <-timeout:
			record("net.write", networkArguments("tcp", connection.local.Port, connection.remote.Port, len(input)), input[:written], uint64(written), resultClass(os.ErrDeadlineExceeded), 0, 0)
			return written, os.ErrDeadlineExceeded
		}
	}
	record("net.write", networkArguments("tcp", connection.local.Port, connection.remote.Port, len(input)), input, uint64(written), 0, 0, 0)
	return written, nil
}

func (connection *Conn) Close() error {
	closed := false
	connection.close.Do(func() {
		closed = true
		connection.closeRead()
		connection.closeWrite()
	})
	if !closed {
		record("net.close", networkArguments("tcp", connection.local.Port, connection.remote.Port), nil, 0, resultClass(ErrClosed), 0, 0)
		return ErrClosed
	}
	record("net.close", networkArguments("tcp", connection.local.Port, connection.remote.Port), nil, 0, 0, 0, 0)
	return nil
}

func (connection *Conn) CloseRead() error {
	if !connection.closeRead() {
		return ErrClosed
	}
	return nil
}

func (connection *Conn) closeRead() bool {
	closed := false
	connection.state.closeRead.Do(func() {
		closed = true
		close(connection.state.readClosed)
	})
	return closed
}

func (connection *Conn) CloseWrite() error {
	if !connection.closeWrite() {
		return ErrClosed
	}
	return nil
}

func (connection *Conn) closeWrite() bool {
	closed := false
	connection.state.closeWrite.Do(func() {
		closed = true
		close(connection.state.writeClosed)
	})
	return closed
}

func (connection *Conn) LocalAddress() Address {
	return connection.local
}

func (connection *Conn) RemoteAddress() Address {
	return connection.remote
}

func (connection *Conn) SetDeadline(deadline time.Time) error {
	connection.state.mu.Lock()
	connection.state.readDeadline = deadline
	connection.state.writeDeadline = deadline
	notify(connection.state.readWake)
	notify(connection.state.writeWake)
	connection.state.mu.Unlock()
	return nil
}

func (connection *Conn) SetReadDeadline(deadline time.Time) error {
	connection.state.mu.Lock()
	connection.state.readDeadline = deadline
	notify(connection.state.readWake)
	connection.state.mu.Unlock()
	return nil
}

func (connection *Conn) SetWriteDeadline(deadline time.Time) error {
	connection.state.mu.Lock()
	connection.state.writeDeadline = deadline
	notify(connection.state.writeWake)
	connection.state.mu.Unlock()
	return nil
}

func (state *connState) deadline(reading bool) (time.Time, <-chan struct{}) {
	state.mu.Lock()
	defer state.mu.Unlock()
	if reading {
		return state.readDeadline, state.readWake
	}
	return state.writeDeadline, state.writeWake
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

func notify(channel chan struct{}) {
	select {
	case channel <- struct{}{}:
	default:
	}
}
