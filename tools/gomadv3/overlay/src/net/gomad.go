// Copyright 2026 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package net

import (
	"context"
	"internal/gomadio"
	"strconv"
	"sync"
)

var gomadNetState = struct {
	sync.RWMutex
	connections map[*netFD]*gomadio.Conn
	listeners   map[*netFD]*gomadio.Listener
}{connections: make(map[*netFD]*gomadio.Conn), listeners: make(map[*netFD]*gomadio.Listener)}

func gomadIOEnabled() bool {
	return gomadio.Enabled()
}

func gomadListenTCP(network string, port int) (*TCPListener, error) {
	listener, err := gomadio.ListenTCP(network, port)
	if err != nil {
		return nil, err
	}
	fd := new(netFD)
	gomadNetState.Lock()
	gomadNetState.listeners[fd] = listener
	gomadNetState.Unlock()
	return &TCPListener{fd: fd}, nil
}

func gomadDialTCP(ctx context.Context, network string, port int) (*TCPConn, error) {
	connection, err := gomadio.DialTCP(ctx, network, port)
	if err != nil {
		return nil, err
	}
	return gomadTCPConn(connection), nil
}

func gomadTCPConn(connection *gomadio.Conn) *TCPConn {
	fd := new(netFD)
	gomadNetState.Lock()
	gomadNetState.connections[fd] = connection
	gomadNetState.Unlock()
	return &TCPConn{conn: conn{fd: fd}}
}

func gomadConnection(fd *netFD) *gomadio.Conn {
	gomadNetState.RLock()
	defer gomadNetState.RUnlock()
	return gomadNetState.connections[fd]
}

func gomadListener(fd *netFD) *gomadio.Listener {
	gomadNetState.RLock()
	defer gomadNetState.RUnlock()
	return gomadNetState.listeners[fd]
}

func gomadTCPAddr(address gomadio.Address) *TCPAddr {
	return &TCPAddr{IP: IPv4(127, 0, 0, 1), Port: address.Port}
}

func gomadParseTCPAddress(network, address string, listening bool) (*TCPAddr, error) {
	if network != "tcp" && network != "tcp4" {
		return nil, gomadio.ErrUnsupported
	}
	host, portText, err := SplitHostPort(address)
	if err != nil {
		return nil, gomadio.ErrUnsupported
	}
	port, err := strconv.Atoi(portText)
	if err != nil || port < 0 || port > 65535 || !listening && port == 0 {
		return nil, gomadio.ErrUnsupported
	}
	switch host {
	case "", "localhost", "127.0.0.1", "0.0.0.0":
	default:
		return nil, gomadio.ErrUnsupported
	}
	return &TCPAddr{IP: IPv4(127, 0, 0, 1), Port: port}, nil
}
