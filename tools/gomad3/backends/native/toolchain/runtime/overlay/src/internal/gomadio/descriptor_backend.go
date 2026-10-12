// Copyright 2026 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gomadio

import (
	"io"
	"sync"

	"internal/gomadsim"
	"internal/gomadvfd"
)

type descriptorIdentity struct {
	fd         uintptr
	generation uint64
}

func (identity descriptorIdentity) notice(mode int32) gomadvfd.Notice {
	return gomadvfd.Notice{FD: identity.fd, Generation: identity.generation, Mode: mode}
}

type descriptorSocket struct {
	identity      descriptorIdentity
	address       Address
	bound         bool
	closed        bool
	listener      *standaloneListener
	connection    *standaloneConn
	waiting       *standaloneListener
	connectStatus gomadvfd.Status
}

type standaloneDescriptorBackend struct {
	sync.Mutex
	waiting map[*standaloneListener][]*descriptorSocket
}

var descriptorBackend = &standaloneDescriptorBackend{
	waiting: make(map[*standaloneListener][]*descriptorSocket),
}

func init() {
	if gomadvfd.RegisterBackend(descriptorBackend) != gomadvfd.OK {
		panic("gomad3: descriptor backend registration failed")
	}
}

func (backend *standaloneDescriptorBackend) Socket() (any, gomadvfd.Status) {
	if gomadsim.ProcessRole() == 2 {
		return nil, gomadvfd.Refuse("descriptor.process-backend")
	}
	if _, _, _, handled := currentSimulationNetwork(); handled {
		return nil, gomadvfd.Refuse("descriptor.simulation-backend")
	}
	return &descriptorSocket{}, gomadvfd.OK
}

func (backend *standaloneDescriptorBackend) Attach(handle any, fd uintptr, generation uint64) {
	backend.Lock()
	socket := handle.(*descriptorSocket)
	socket.identity = descriptorIdentity{fd, generation}
	if socket.listener != nil {
		socket.listener.mu.Lock()
		socket.listener.descriptor = socket.identity
		socket.listener.mu.Unlock()
	}
	if socket.connection != nil {
		socket.connection.lockState()
		socket.connection.state.descriptor = socket.identity
		socket.connection.unlockState()
	}
	backend.Unlock()
}

func descriptorAddress(address gomadvfd.Address, listen bool) (Address, gomadvfd.Status) {
	if address.Port < 0 || address.Port > maximumPort || !listen && address.Port == 0 {
		return Address{}, gomadvfd.Invalid
	}
	if address.IP != [4]byte{127, 0, 0, 1} && address.IP != [4]byte{} {
		return Address{}, gomadvfd.Unsupported
	}
	return Address{IP: "127.0.0.1", Port: address.Port}, gomadvfd.OK
}

func leafAddress(address Address) gomadvfd.Address {
	return gomadvfd.Address{IP: [4]byte{127, 0, 0, 1}, Port: address.Port}
}

func (backend *standaloneDescriptorBackend) bindLocked(socket *descriptorSocket, address Address) gomadvfd.Status {
	if socket.closed {
		return gomadvfd.Closed
	}
	if socket.bound || socket.connection != nil || socket.waiting != nil {
		return gomadvfd.Invalid
	}
	networkState.Lock()
	defer networkState.Unlock()
	if address.Port == 0 {
		for networkState.nextListenerPort <= maximumPort {
			address.Port = networkState.nextListenerPort
			networkState.nextListenerPort++
			if networkState.listeners[address.Port] == nil && networkState.boundPorts[address.Port] == nil {
				break
			}
		}
		if address.Port == 0 || address.Port > maximumPort || networkState.listeners[address.Port] != nil || networkState.boundPorts[address.Port] != nil {
			return gomadvfd.Capacity
		}
	}
	if networkState.listeners[address.Port] != nil || networkState.boundPorts[address.Port] != nil {
		return gomadvfd.AddressInUse
	}
	socket.address = address
	socket.bound = true
	networkState.boundPorts[address.Port] = socket
	return gomadvfd.OK
}

func (backend *standaloneDescriptorBackend) Bind(handle any, address gomadvfd.Address) gomadvfd.Status {
	local, status := descriptorAddress(address, true)
	if status != gomadvfd.OK {
		return status
	}
	backend.Lock()
	defer backend.Unlock()
	return backend.bindLocked(handle.(*descriptorSocket), local)
}

func (backend *standaloneDescriptorBackend) Listen(handle any, backlog int) gomadvfd.Status {
	backend.Lock()
	defer backend.Unlock()
	socket := handle.(*descriptorSocket)
	if socket.closed {
		return gomadvfd.Closed
	}
	if socket.connection != nil || socket.waiting != nil || backlog < 0 {
		return gomadvfd.Invalid
	}
	if socket.listener != nil {
		return gomadvfd.OK
	}
	if !socket.bound {
		if status := backend.bindLocked(socket, Address{IP: "127.0.0.1"}); status != gomadvfd.OK {
			return status
		}
	}
	networkState.Lock()
	defer networkState.Unlock()
	if networkState.listeners[socket.address.Port] != nil {
		return gomadvfd.AddressInUse
	}
	socket.listener = &standaloneListener{address: socket.address, changed: make(chan struct{}), descriptor: socket.identity}
	networkState.listeners[socket.address.Port] = socket.listener
	return gomadvfd.OK
}

func (backend *standaloneDescriptorBackend) Connect(handle any, address gomadvfd.Address) gomadvfd.Status {
	remote, status := descriptorAddress(address, false)
	if status != gomadvfd.OK {
		return status
	}
	backend.Lock()
	socket := handle.(*descriptorSocket)
	if socket.closed {
		backend.Unlock()
		return gomadvfd.Closed
	}
	if socket.connection != nil || socket.listener != nil {
		backend.Unlock()
		return gomadvfd.Invalid
	}
	if socket.waiting != nil {
		backend.Unlock()
		return gomadvfd.WouldBlock
	}
	networkState.Lock()
	listener := networkState.listeners[remote.Port]
	if listener == nil {
		networkState.Unlock()
		backend.Unlock()
		return gomadvfd.Refused
	}
	if !socket.bound {
		local, err := allocateClientAddressLocked()
		if err != nil {
			networkState.Unlock()
			backend.Unlock()
			return gomadvfd.Capacity
		}
		socket.address = local
	}
	listener.mu.Lock()
	networkState.Unlock()
	client, err := listener.tryConnectLocked(socket.address)
	var notices []gomadvfd.Notice
	switch err {
	case nil:
		socket.connection = client.implementation.(*standaloneConn)
		socket.connection.state.descriptor = socket.identity
		socket.connectStatus = gomadvfd.OK
		notices = append(notices, listener.descriptor.notice('r'), socket.identity.notice('w'))
		status = gomadvfd.OK
	case errNetworkWouldBlock:
		if len(backend.waiting[listener]) == maximumPendingConns {
			status = gomadvfd.Capacity
			break
		}
		backend.waiting[listener] = append(backend.waiting[listener], socket)
		socket.waiting = listener
		socket.connectStatus = gomadvfd.WouldBlock
		status = gomadvfd.WouldBlock
	default:
		status = gomadvfd.Refused
	}
	listener.mu.Unlock()
	backend.Unlock()
	gomadvfd.Notify(notices...)
	return status
}

func (backend *standaloneDescriptorBackend) Accept(handle any) (any, gomadvfd.Address, gomadvfd.Status) {
	backend.Lock()
	socket := handle.(*descriptorSocket)
	if socket.closed {
		backend.Unlock()
		return nil, gomadvfd.Address{}, gomadvfd.Closed
	}
	listener := socket.listener
	if listener == nil {
		backend.Unlock()
		return nil, gomadvfd.Address{}, gomadvfd.Invalid
	}
	listener.mu.Lock()
	connection, err := listener.tryAcceptLocked()
	if err != nil {
		listener.mu.Unlock()
		backend.Unlock()
		return nil, gomadvfd.Address{}, descriptorStatus(err)
	}
	var notices []gomadvfd.Notice
	if waiting := backend.waiting[listener]; len(waiting) != 0 {
		next := waiting[0]
		waiting[0] = nil
		backend.waiting[listener] = waiting[1:]
		client, connectError := listener.tryConnectLocked(next.address)
		next.waiting = nil
		next.connectStatus = descriptorStatus(connectError)
		if connectError == nil {
			next.connection = client.implementation.(*standaloneConn)
			next.connection.state.descriptor = next.identity
			notices = append(notices, listener.descriptor.notice('r'))
		}
		notices = append(notices, next.identity.notice('w'))
	}
	listener.mu.Unlock()
	accepted := &descriptorSocket{connection: connection.implementation.(*standaloneConn)}
	peer := leafAddress(accepted.connection.remote)
	backend.Unlock()
	gomadvfd.Notify(notices...)
	return accepted, peer, gomadvfd.OK
}

func descriptorStatus(err error) gomadvfd.Status {
	switch err {
	case nil:
		return gomadvfd.OK
	case errNetworkWouldBlock:
		return gomadvfd.WouldBlock
	case ErrClosed:
		return gomadvfd.Closed
	case ErrConnectionRefused:
		return gomadvfd.Refused
	case io.EOF:
		return gomadvfd.EndOfStream
	default:
		return gomadvfd.Unsupported
	}
}

func (backend *standaloneDescriptorBackend) resumeWaiting(listener *standaloneListener) {
	backend.Lock()
	listener.mu.Lock()
	var notices []gomadvfd.Notice
	waiting := backend.waiting[listener]
	for len(waiting) != 0 {
		next := waiting[0]
		client, err := listener.tryConnectLocked(next.address)
		if err == errNetworkWouldBlock {
			break
		}
		waiting[0] = nil
		waiting = waiting[1:]
		next.waiting = nil
		next.connectStatus = descriptorStatus(err)
		if err == nil {
			next.connection = client.implementation.(*standaloneConn)
			next.connection.state.descriptor = next.identity
			notices = append(notices, listener.descriptor.notice('r'))
		}
		notices = append(notices, next.identity.notice('w'))
	}
	if len(waiting) == 0 {
		delete(backend.waiting, listener)
	} else {
		backend.waiting[listener] = waiting
	}
	listener.mu.Unlock()
	backend.Unlock()
	gomadvfd.Notify(notices...)
}

func (backend *standaloneDescriptorBackend) Read(handle any, destination []byte) (int, gomadvfd.Status) {
	backend.Lock()
	socket := handle.(*descriptorSocket)
	if socket.closed {
		backend.Unlock()
		return 0, gomadvfd.Closed
	}
	connection := socket.connection
	if connection == nil {
		backend.Unlock()
		return 0, gomadvfd.NotConnected
	}
	connection.lockState()
	n, err, _, freed := connection.tryReadLocked(destination)
	status := descriptorStatus(err)
	if err != nil && connection.state.reset {
		status = gomadvfd.Reset
	}
	notice := connection.peer.descriptor.notice('w')
	connection.unlockState()
	backend.Unlock()
	if freed {
		gomadvfd.Notify(notice)
	}
	return n, status
}

func (backend *standaloneDescriptorBackend) Write(handle any, source []byte) (int, gomadvfd.Status) {
	backend.Lock()
	socket := handle.(*descriptorSocket)
	if socket.closed {
		backend.Unlock()
		return 0, gomadvfd.Closed
	}
	connection := socket.connection
	if connection == nil {
		backend.Unlock()
		return 0, gomadvfd.NotConnected
	}
	connection.lockState()
	n, err := connection.tryWriteLocked(source)
	status := descriptorStatus(err)
	if err == ErrClosed {
		if connection.state.reset || connection.peer.reset {
			status = gomadvfd.Reset
		} else {
			status = gomadvfd.BrokenPipe
		}
	}
	notice := connection.peer.descriptor.notice('r')
	connection.unlockState()
	backend.Unlock()
	if n != 0 {
		gomadvfd.Notify(notice)
	}
	return n, status
}

func (backend *standaloneDescriptorBackend) Shutdown(handle any, how int) gomadvfd.Status {
	if how < 0 || how > 2 {
		return gomadvfd.Invalid
	}
	backend.Lock()
	socket := handle.(*descriptorSocket)
	if socket.closed {
		backend.Unlock()
		return gomadvfd.Closed
	}
	connection := socket.connection
	if connection == nil {
		backend.Unlock()
		return gomadvfd.NotConnected
	}
	connection.lockState()
	connection.shutdownLocked(how == 0 || how == 2, how == 1 || how == 2)
	notices := connection.descriptorNoticesLocked()
	connection.unlockState()
	backend.Unlock()
	gomadvfd.Notify(notices...)
	return gomadvfd.OK
}

func (backend *standaloneDescriptorBackend) Close(handle any) gomadvfd.Status {
	backend.Lock()
	socket := handle.(*descriptorSocket)
	if socket.closed {
		backend.Unlock()
		return gomadvfd.Closed
	}
	socket.closed = true
	networkState.Lock()
	if networkState.boundPorts[socket.address.Port] == socket {
		delete(networkState.boundPorts, socket.address.Port)
	}
	networkState.Unlock()
	notices := []gomadvfd.Notice{socket.identity.notice('r'), socket.identity.notice('w')}
	if listener := socket.waiting; listener != nil {
		waiting := backend.waiting[listener]
		for index, pending := range waiting {
			if pending == socket {
				copy(waiting[index:], waiting[index+1:])
				waiting[len(waiting)-1] = nil
				backend.waiting[listener] = waiting[:len(waiting)-1]
				break
			}
		}
		socket.waiting = nil
	}
	if listener := socket.listener; listener != nil {
		networkState.Lock()
		listener.mu.Lock()
		if networkState.listeners[listener.address.Port] == listener {
			delete(networkState.listeners, listener.address.Port)
		}
		listener.closed = true
		listener.signal()
		for _, pending := range backend.waiting[listener] {
			pending.waiting = nil
			pending.connectStatus = gomadvfd.Refused
			notices = append(notices, pending.identity.notice('r'), pending.identity.notice('w'))
		}
		delete(backend.waiting, listener)
		for _, pending := range listener.pending {
			connection := pending.implementation.(*standaloneConn)
			connection.lockState()
			connection.state.reset = true
			connection.peer.reset = true
			connection.state.shared.signal()
			notices = append(notices, connection.peer.descriptor.notice('r'), connection.peer.descriptor.notice('w'))
			connection.unlockState()
		}
		listener.pending = nil
		listener.mu.Unlock()
		networkState.Unlock()
	}
	if connection := socket.connection; connection != nil {
		connection.lockState()
		connection.shutdownLocked(true, true)
		notices = append(notices, connection.peer.descriptor.notice('r'), connection.peer.descriptor.notice('w'))
		connection.unlockState()
	}
	backend.Unlock()
	gomadvfd.Notify(notices...)
	return gomadvfd.OK
}

func (backend *standaloneDescriptorBackend) Local(handle any) (gomadvfd.Address, gomadvfd.Status) {
	backend.Lock()
	defer backend.Unlock()
	socket := handle.(*descriptorSocket)
	if socket.closed {
		return gomadvfd.Address{}, gomadvfd.Closed
	}
	if socket.connection != nil {
		return leafAddress(socket.connection.local), gomadvfd.OK
	}
	if socket.address.Port != 0 {
		return leafAddress(socket.address), gomadvfd.OK
	}
	return gomadvfd.Address{}, gomadvfd.OK
}

func (backend *standaloneDescriptorBackend) Remote(handle any) (gomadvfd.Address, gomadvfd.Status) {
	backend.Lock()
	defer backend.Unlock()
	socket := handle.(*descriptorSocket)
	if socket.closed {
		return gomadvfd.Address{}, gomadvfd.Closed
	}
	if socket.connection != nil {
		return leafAddress(socket.connection.remote), gomadvfd.OK
	}
	if socket.connectStatus != gomadvfd.OK {
		return gomadvfd.Address{}, socket.connectStatus
	}
	return gomadvfd.Address{}, gomadvfd.NotConnected
}

func (backend *standaloneDescriptorBackend) Refuse(operation string) {
	err := ErrUnsupported
	if operation == "descriptor.capacity" {
		err = ErrResourceExhausted
	}
	record("net.syscall.refused", []byte(operation), nil, 0, resultClass(err), 0, 0)
}
