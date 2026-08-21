// Copyright 2026 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package net

import (
	"context"
	"io"
	"os"
	"strconv"
	"sync"
	"syscall"
	"time"

	"internal/gomadio"
	"internal/gomadtrace"
)

var gomadNetState = struct {
	sync.RWMutex
	connections map[*netFD]*gomadio.Conn
	listeners   map[*netFD]*gomadio.Listener
}{connections: make(map[*netFD]*gomadio.Conn), listeners: make(map[*netFD]*gomadio.Listener)}

func gomadIOEnabled() bool {
	return gomadio.NetworkEnabled()
}

func gomadObserveBoundary(id uint64) {
	gomadtrace.ObserveBoundary(id)
}

func gomadInterceptInterfaces() ([]Interface, error, bool) {
	if !gomadIOEnabled() {
		return nil, nil, false
	}
	return nil, gomadio.ErrUnsupported, true
}

func gomadInterceptResolverLookupIPAddr(_ *Resolver, ctx context.Context, host string) ([]IPAddr, error, bool) {
	if !gomadIOEnabled() {
		return nil, nil, false
	}
	if err := ctx.Err(); err != nil {
		return nil, err, true
	}
	if host != "localhost" {
		return nil, gomadio.ErrUnsupported, true
	}
	return []IPAddr{{IP: IPv4(127, 0, 0, 1)}}, nil, true
}

func gomadInterceptDialContext(dialer *Dialer, ctx context.Context, network, address string) (Conn, error, bool) {
	if !gomadIOEnabled() {
		return nil, nil, false
	}
	ctx, cancel := dialer.dialCtx(ctx)
	defer cancel()
	remoteAddress, err := gomadParseTCPAddress(network, address, false)
	if err != nil {
		return nil, &OpError{Op: "dial", Net: network, Source: dialer.LocalAddr, Addr: nil, Err: err}, true
	}
	localAddress, ok := dialer.LocalAddr.(*TCPAddr)
	if dialer.LocalAddr != nil && !ok {
		return nil, &OpError{Op: "dial", Net: network, Source: dialer.LocalAddr, Addr: remoteAddress, Err: gomadio.ErrUnsupported}, true
	}
	connection, err := dialTCP(ctx, dialer, network, localAddress, remoteAddress)
	if err != nil {
		return nil, err, true
	}
	return connection, err, true
}

func gomadInterceptListen(config *ListenConfig, ctx context.Context, network, address string) (Listener, error, bool) {
	if !gomadIOEnabled() {
		return nil, nil, false
	}
	localAddress, err := gomadParseTCPAddress(network, address, true)
	if err != nil {
		return nil, &OpError{Op: "listen", Net: network, Source: nil, Addr: nil, Err: err}, true
	}
	listener, err := ListenTCP(network, localAddress)
	return listener, err, true
}

func gomadInterceptConnRead(conn *conn, buffer []byte) (int, error, bool) {
	if !conn.ok() {
		return 0, syscall.EINVAL, true
	}
	connection := gomadConnection(conn.fd)
	if connection == nil {
		if !gomadIOEnabled() {
			return 0, nil, false
		}
		return 0, gomadio.ErrUnsupported, true
	}
	read, err := connection.Read(buffer)
	if err != nil && err != io.EOF {
		err = &OpError{Op: "read", Net: "tcp", Source: gomadTCPAddr(connection.LocalAddress()), Addr: gomadTCPAddr(connection.RemoteAddress()), Err: err}
	}
	return read, err, true
}

func gomadInterceptConnWrite(conn *conn, buffer []byte) (int, error, bool) {
	if !conn.ok() {
		return 0, syscall.EINVAL, true
	}
	connection := gomadConnection(conn.fd)
	if connection == nil {
		if !gomadIOEnabled() {
			return 0, nil, false
		}
		return 0, gomadio.ErrUnsupported, true
	}
	written, err := connection.Write(buffer)
	if err != nil {
		err = &OpError{Op: "write", Net: "tcp", Source: gomadTCPAddr(connection.LocalAddress()), Addr: gomadTCPAddr(connection.RemoteAddress()), Err: err}
	}
	return written, err, true
}

func gomadInterceptConnClose(conn *conn) (error, bool) {
	if !conn.ok() {
		return syscall.EINVAL, true
	}
	connection := gomadConnection(conn.fd)
	if connection == nil {
		if !gomadIOEnabled() {
			return nil, false
		}
		return gomadio.ErrUnsupported, true
	}
	return connection.Close(), true
}

func gomadInterceptConnLocalAddr(conn *conn) (Addr, bool) {
	if !conn.ok() {
		return nil, true
	}
	connection := gomadConnection(conn.fd)
	if connection == nil {
		return nil, gomadIOEnabled()
	}
	return gomadTCPAddr(connection.LocalAddress()), true
}

func gomadInterceptConnRemoteAddr(conn *conn) (Addr, bool) {
	if !conn.ok() {
		return nil, true
	}
	connection := gomadConnection(conn.fd)
	if connection == nil {
		return nil, gomadIOEnabled()
	}
	return gomadTCPAddr(connection.RemoteAddress()), true
}

func gomadInterceptConnSetDeadline(conn *conn, deadline time.Time) (error, bool) {
	if !conn.ok() {
		return syscall.EINVAL, true
	}
	connection := gomadConnection(conn.fd)
	if connection == nil {
		if !gomadIOEnabled() {
			return nil, false
		}
		return gomadio.ErrUnsupported, true
	}
	return connection.SetDeadline(deadline), true
}

func gomadInterceptConnSetReadDeadline(conn *conn, deadline time.Time) (error, bool) {
	if !conn.ok() {
		return syscall.EINVAL, true
	}
	connection := gomadConnection(conn.fd)
	if connection == nil {
		if !gomadIOEnabled() {
			return nil, false
		}
		return gomadio.ErrUnsupported, true
	}
	return connection.SetReadDeadline(deadline), true
}

func gomadInterceptConnSetWriteDeadline(conn *conn, deadline time.Time) (error, bool) {
	if !conn.ok() {
		return syscall.EINVAL, true
	}
	connection := gomadConnection(conn.fd)
	if connection == nil {
		if !gomadIOEnabled() {
			return nil, false
		}
		return gomadio.ErrUnsupported, true
	}
	return connection.SetWriteDeadline(deadline), true
}

func gomadInterceptConnSetReadBuffer(conn *conn, _ int) (error, bool) {
	if !conn.ok() {
		return syscall.EINVAL, true
	}
	if gomadConnection(conn.fd) == nil {
		if !gomadIOEnabled() {
			return nil, false
		}
		return gomadio.ErrUnsupported, true
	}
	return nil, true
}

func gomadInterceptConnSetWriteBuffer(conn *conn, _ int) (error, bool) {
	if !conn.ok() {
		return syscall.EINVAL, true
	}
	if gomadConnection(conn.fd) == nil {
		if !gomadIOEnabled() {
			return nil, false
		}
		return gomadio.ErrUnsupported, true
	}
	return nil, true
}

func gomadInterceptConnFile(conn *conn) (*os.File, error, bool) {
	if gomadConnection(conn.fd) == nil {
		if !gomadIOEnabled() {
			return nil, nil, false
		}
		return nil, gomadio.ErrUnsupported, true
	}
	return nil, gomadio.ErrUnsupported, true
}

func gomadInterceptTCPConnSyscallConn(conn *TCPConn) (syscall.RawConn, error, bool) {
	if conn == nil || gomadConnection(conn.fd) == nil {
		if !gomadIOEnabled() {
			return nil, nil, false
		}
		return nil, gomadio.ErrUnsupported, true
	}
	return nil, gomadio.ErrUnsupported, true
}

func gomadInterceptTCPConnCloseRead(conn *TCPConn) (error, bool) {
	if conn == nil {
		return nil, false
	}
	connection := gomadConnection(conn.fd)
	if connection == nil {
		if !gomadIOEnabled() {
			return nil, false
		}
		return gomadio.ErrUnsupported, true
	}
	return connection.CloseRead(), true
}

func gomadInterceptTCPConnCloseWrite(conn *TCPConn) (error, bool) {
	if conn == nil {
		return nil, false
	}
	connection := gomadConnection(conn.fd)
	if connection == nil {
		if !gomadIOEnabled() {
			return nil, false
		}
		return gomadio.ErrUnsupported, true
	}
	return connection.CloseWrite(), true
}

func gomadInterceptTCPConnOption(conn *TCPConn) (error, bool) {
	if conn == nil || gomadConnection(conn.fd) == nil {
		if !gomadIOEnabled() {
			return nil, false
		}
		return gomadio.ErrUnsupported, true
	}
	return nil, true
}

func gomadInterceptTCPConnSetLinger(conn *TCPConn, _ int) (error, bool) {
	return gomadInterceptTCPConnOption(conn)
}

func gomadInterceptTCPConnSetKeepAlive(conn *TCPConn, _ bool) (error, bool) {
	return gomadInterceptTCPConnOption(conn)
}

func gomadInterceptTCPConnSetKeepAlivePeriod(conn *TCPConn, _ time.Duration) (error, bool) {
	return gomadInterceptTCPConnOption(conn)
}

func gomadInterceptTCPConnSetNoDelay(conn *TCPConn, _ bool) (error, bool) {
	return gomadInterceptTCPConnOption(conn)
}

func gomadInterceptTCPConnMultipathTCP(conn *TCPConn) (bool, error, bool) {
	if conn == nil || gomadConnection(conn.fd) == nil {
		if !gomadIOEnabled() {
			return false, nil, false
		}
		return false, gomadio.ErrUnsupported, true
	}
	return false, nil, true
}

func gomadInterceptDialTCP(ctx context.Context, _ *Dialer, network string, local, remote *TCPAddr) (*TCPConn, error, bool) {
	if !gomadIOEnabled() {
		return nil, nil, false
	}
	if remote == nil {
		return nil, &OpError{Op: "dial", Net: network, Source: local.opAddr(), Addr: nil, Err: errMissingAddress}, true
	}
	if local != nil && local.Port != 0 {
		return nil, &OpError{Op: "dial", Net: network, Source: local.opAddr(), Addr: remote, Err: gomadio.ErrUnsupported}, true
	}
	connection, err := gomadDialTCP(ctx, network, remote.IP.String(), remote.Port)
	if err != nil {
		return nil, &OpError{Op: "dial", Net: network, Source: local.opAddr(), Addr: remote, Err: err}, true
	}
	return connection, nil, true
}

func gomadInterceptTCPListenerSyscallConn(listener *TCPListener) (syscall.RawConn, error, bool) {
	if listener == nil || gomadListener(listener.fd) == nil {
		if !gomadIOEnabled() {
			return nil, nil, false
		}
		return nil, gomadio.ErrUnsupported, true
	}
	return nil, gomadio.ErrUnsupported, true
}

func gomadInterceptTCPListenerAcceptTCP(listener *TCPListener) (*TCPConn, error, bool) {
	if !listener.ok() {
		return nil, syscall.EINVAL, true
	}
	modeled := gomadListener(listener.fd)
	if modeled == nil {
		if !gomadIOEnabled() {
			return nil, nil, false
		}
		return nil, gomadio.ErrUnsupported, true
	}
	connection, err := modeled.Accept()
	if err != nil {
		return nil, &OpError{Op: "accept", Net: "tcp", Source: nil, Addr: gomadTCPAddr(modeled.Address()), Err: err}, true
	}
	return gomadTCPConn(connection), nil, true
}

func gomadInterceptTCPListenerAccept(listener *TCPListener) (Conn, error, bool) {
	if !listener.ok() {
		return nil, syscall.EINVAL, true
	}
	if gomadListener(listener.fd) == nil {
		if !gomadIOEnabled() {
			return nil, nil, false
		}
		return nil, gomadio.ErrUnsupported, true
	}
	connection, err := listener.AcceptTCP()
	return connection, err, true
}

func gomadInterceptTCPListenerClose(listener *TCPListener) (error, bool) {
	if !listener.ok() {
		return syscall.EINVAL, true
	}
	modeled := gomadListener(listener.fd)
	if modeled == nil {
		if !gomadIOEnabled() {
			return nil, false
		}
		return gomadio.ErrUnsupported, true
	}
	return modeled.Close(), true
}

func gomadInterceptTCPListenerAddr(listener *TCPListener) (Addr, bool) {
	if listener == nil {
		return nil, false
	}
	modeled := gomadListener(listener.fd)
	if modeled == nil {
		return nil, gomadIOEnabled()
	}
	return gomadTCPAddr(modeled.Address()), true
}

func gomadInterceptTCPListenerSetDeadline(listener *TCPListener, deadline time.Time) (error, bool) {
	if !listener.ok() {
		return syscall.EINVAL, true
	}
	modeled := gomadListener(listener.fd)
	if modeled == nil {
		if !gomadIOEnabled() {
			return nil, false
		}
		return gomadio.ErrUnsupported, true
	}
	return modeled.SetDeadline(deadline), true
}

func gomadInterceptTCPListenerFile(listener *TCPListener) (*os.File, error, bool) {
	if !listener.ok() {
		return nil, syscall.EINVAL, true
	}
	if gomadListener(listener.fd) == nil {
		if !gomadIOEnabled() {
			return nil, nil, false
		}
		return nil, gomadio.ErrUnsupported, true
	}
	return nil, gomadio.ErrUnsupported, true
}

func gomadInterceptListenTCP(network string, local *TCPAddr) (*TCPListener, error, bool) {
	if !gomadIOEnabled() {
		return nil, nil, false
	}
	if local == nil {
		local = &TCPAddr{}
	}
	listener, err := gomadListenTCP(network, local.IP.String(), local.Port)
	if err != nil {
		return nil, &OpError{Op: "listen", Net: network, Source: nil, Addr: local, Err: err}, true
	}
	return listener, nil, true
}

func gomadInterceptTCPConnSetKeepAliveConfig(conn *TCPConn, _ KeepAliveConfig) (error, bool) {
	return gomadInterceptTCPConnOption(conn)
}

func gomadListenTCP(network, host string, port int) (*TCPListener, error) {
	listener, err := gomadio.ListenTCP(network, host, port)
	if err != nil {
		return nil, err
	}
	fd := new(netFD)
	gomadNetState.Lock()
	gomadNetState.listeners[fd] = listener
	gomadNetState.Unlock()
	return &TCPListener{fd: fd}, nil
}

func gomadDialTCP(ctx context.Context, network, host string, port int) (*TCPConn, error) {
	connection, err := gomadio.DialTCP(ctx, network, host, port)
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
	ip := ParseIP(address.IP)
	if ip == nil {
		ip = IPv4(127, 0, 0, 1)
	}
	return &TCPAddr{IP: ip, Port: address.Port}
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
	if host == "localhost" {
		host = "127.0.0.1"
	}
	if host == "" {
		if !listening {
			return nil, gomadio.ErrUnsupported
		}
		host = "0.0.0.0"
	}
	ip := ParseIP(host)
	if ip == nil || ip.To4() == nil {
		return nil, gomadio.ErrUnsupported
	}
	return &TCPAddr{IP: ip.To4(), Port: port}, nil
}
