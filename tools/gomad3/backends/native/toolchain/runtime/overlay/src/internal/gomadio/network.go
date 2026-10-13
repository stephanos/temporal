// Copyright 2026 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gomadio

import (
	"context"
	"errors"
	"os"
	"sync"
	"syscall"
	"time"

	"internal/gomadsim"
	"internal/gomadvfd"
	"internal/poll"
)

var (
	ErrAddressInUse      = errors.New("address already in use")
	ErrClosed            = poll.ErrNetClosing
	ErrConnectionRefused = errors.New("connection refused")
	ErrStallTimeout      = syscall.ETIMEDOUT
	ErrResourceExhausted = errors.New("network resources exhausted")
	ErrUnsupported       = errors.New("unsupported Gomad network operation")
	errNetworkWouldBlock = errors.New("Gomad network would block")
)

const (
	firstListenerPort               = 20000
	firstClientPort                 = 40000
	maximumPort                     = 65535
	maximumPendingConns             = 64
	maximumPendingNetworkOperations = 1 << 16
	maximumChunkBytes               = 64 << 10
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

type standaloneConn struct{ pairedConn }
type standaloneListener struct {
	address    Address
	once       sync.Once
	mu         sync.Mutex
	pending    []*Conn
	closed     bool
	deadline   time.Time
	changed    chan struct{}
	descriptor descriptorIdentity
}

var networkState = struct {
	sync.Mutex
	listeners        map[int]*standaloneListener
	boundPorts       map[int]*descriptorSocket
	nextListenerPort int
	nextClientPort   int
}{listeners: make(map[int]*standaloneListener), boundPorts: make(map[int]*descriptorSocket), nextListenerPort: firstListenerPort, nextClientPort: firstClientPort}

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
			if _, found := networkState.listeners[port]; !found && networkState.boundPorts[port] == nil {
				break
			}
		}
		if port == 0 || port > maximumPort {
			record("net.listen", networkArguments(network, requestedPort), nil, 0, resultClass(ErrResourceExhausted), 0, 0)
			return nil, ErrResourceExhausted
		}
	}
	if _, found := networkState.listeners[port]; found || networkState.boundPorts[port] != nil {
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
	clientAddress, err := allocateClientAddressLocked()
	if err != nil {
		networkState.Unlock()
		record("net.dial", networkArguments(network, port), nil, 0, resultClass(ErrResourceExhausted), 0, 0)
		return nil, ErrResourceExhausted
	}
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
			client, err := listener.tryConnectLocked(clientAddress)
			notice := listener.descriptor.notice('r')
			listener.mu.Unlock()
			record("net.dial", networkArguments(network, clientAddress.Port, port), nil, 0, resultClass(err), 0, 0)
			gomadvfd.Notify(notice)
			return client, err
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
		connection, err := listener.tryAcceptLocked()
		if err == nil {
			listener.mu.Unlock()
			record("net.accept", networkArguments("tcp", listener.address.Port, connection.RemoteAddress().Port), nil, 0, 0, 0, 0)
			descriptorBackend.resumeWaiting(listener)
			return connection, nil
		}
		if err == ErrClosed {
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
	descriptorBackend.resumeWaiting(listener)
	gomadvfd.Notify(listener.descriptor.notice('r'), listener.descriptor.notice('w'))
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

func newConnStates() (*connState, *connState) {
	shared := &connShared{changed: make(chan struct{})}
	return &connState{shared: shared}, &connState{shared: shared}
}

func allocateClientAddressLocked() (Address, error) {
	for networkState.nextClientPort <= maximumPort {
		port := networkState.nextClientPort
		networkState.nextClientPort++
		if networkState.listeners[port] == nil && networkState.boundPorts[port] == nil {
			return Address{IP: "127.0.0.1", Port: port}, nil
		}
	}
	return Address{}, ErrResourceExhausted
}

func (listener *standaloneListener) tryConnectLocked(clientAddress Address) (*Conn, error) {
	if listener.closed {
		return nil, ErrConnectionRefused
	}
	if len(listener.pending) == maximumPendingConns {
		return nil, errNetworkWouldBlock
	}
	clientState, serverState := newConnStates()
	client := &Conn{implementation: &standaloneConn{pairedConn{local: clientAddress, remote: listener.address, state: clientState, peer: serverState}}}
	server := &Conn{implementation: &standaloneConn{pairedConn{local: listener.address, remote: clientAddress, state: serverState, peer: clientState}}}
	listener.pending = append(listener.pending, server)
	listener.signal()
	return client, nil
}

func (listener *standaloneListener) tryAcceptLocked() (*Conn, error) {
	if len(listener.pending) != 0 {
		connection := listener.pending[0]
		listener.pending[0] = nil
		listener.pending = listener.pending[1:]
		listener.signal()
		return connection, nil
	}
	if listener.closed {
		return nil, ErrClosed
	}
	return nil, errNetworkWouldBlock
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
