// Copyright 2026 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gomadio

import (
	"context"
	"errors"
	"io"
	"sync"
	"syscall"
	"time"
	_ "unsafe"

	"internal/gomadsim"
)

const maximumProcessNetworkHandles = 1 << 20

type processListener struct {
	processHandle uint64
	address       Address
}
type processConn struct {
	processHandle uint64
	local         Address
	remote        Address
}

func (listener *processListener) Address() Address     { return listener.address }
func (connection *processConn) LocalAddress() Address  { return connection.local }
func (connection *processConn) RemoteAddress() Address { return connection.remote }
func (connection *processConn) Close() error {
	return processNetworkConnOperation(connection, processNetworkConnCloseOp, time.Time{})
}
func (connection *processConn) CloseRead() error {
	return processNetworkConnOperation(connection, processNetworkConnCloseReadOp, time.Time{})
}
func (connection *processConn) CloseWrite() error {
	return processNetworkConnOperation(connection, processNetworkConnCloseWriteOp, time.Time{})
}
func (connection *processConn) SetDeadline(deadline time.Time) error {
	return processNetworkConnOperation(connection, processNetworkConnSetDeadlineOp, deadline)
}
func (connection *processConn) SetReadDeadline(deadline time.Time) error {
	return processNetworkConnOperation(connection, processNetworkConnSetReadDeadlineOp, deadline)
}
func (connection *processConn) SetWriteDeadline(deadline time.Time) error {
	return processNetworkConnOperation(connection, processNetworkConnSetWriteDeadlineOp, deadline)
}

type processNetworkResource struct {
	domain   uint64
	listener *Listener
	conn     *Conn
}

var processNetworkResources = struct {
	sync.Mutex
	next   uint64
	values map[uint64]processNetworkResource
}{values: make(map[uint64]processNetworkResource)}

func processNetworkListen(network, host string, port int) (*Listener, error) {
	response, err := exchangeProcessNetwork(processNetworkCommand{Operation: processNetworkListenOp, Network: network, Host: host, Port: int64(port)})
	if err != nil {
		return nil, err
	}
	return &Listener{implementation: &processListener{processHandle: response.Handle, address: response.Local}}, nil
}

func processNetworkDial(ctx context.Context, network, host string, port int) (*Conn, error) {
	deadline := int64(0)
	if value, ok := ctx.Deadline(); ok {
		deadline = value.UnixNano()
	}
	response, err := exchangeProcessNetwork(processNetworkCommand{Operation: processNetworkDialOp, Network: network, Host: host, Port: int64(port), DeadlineNanos: deadline})
	if err != nil {
		return nil, err
	}
	return processNetworkConn(response), nil
}

func (listener *processListener) Accept() (*Conn, error) {
	response, err := exchangeProcessNetwork(processNetworkCommand{Operation: processNetworkAcceptOp, Handle: listener.processHandle})
	if err != nil {
		return nil, err
	}
	return processNetworkConn(response), nil
}

func (listener *processListener) Close() error {
	_, err := exchangeProcessNetwork(processNetworkCommand{Operation: processNetworkListenerCloseOp, Handle: listener.processHandle})
	return err
}

func (listener *processListener) SetDeadline(deadline time.Time) error {
	_, err := exchangeProcessNetwork(processNetworkCommand{Operation: processNetworkListenerSetDeadlineOp, Handle: listener.processHandle, DeadlineNanos: processNetworkDeadline(deadline)})
	return err
}

func (connection *processConn) Read(destination []byte) (int, error) {
	if len(destination) == 0 {
		return 0, nil
	}
	response, err := exchangeProcessNetwork(processNetworkCommand{Operation: processNetworkConnReadOp, Handle: connection.processHandle, ReadLength: uint64(len(destination))})
	read := copy(destination, response.Data)
	if read != len(response.Data) || uint64(read) != response.BytesTransferred {
		return 0, syscall.EIO
	}
	return read, err
}

func (connection *processConn) Write(source []byte) (int, error) {
	written := 0
	for len(source) != 0 {
		length := min(len(source), maximumChunkBytes)
		response, err := exchangeProcessNetwork(processNetworkCommand{Operation: processNetworkConnWriteOp, Handle: connection.processHandle, Data: append([]byte(nil), source[:length]...)})
		if response.BytesTransferred > uint64(length) {
			return written, syscall.EIO
		}
		written += int(response.BytesTransferred)
		source = source[response.BytesTransferred:]
		if err != nil {
			return written, err
		}
		if response.BytesTransferred == 0 {
			return written, io.ErrShortWrite
		}
	}
	return written, nil
}

func processNetworkConnOperation(connection *processConn, operation processNetworkOperation, deadline time.Time) error {
	_, err := exchangeProcessNetwork(processNetworkCommand{Operation: operation, Handle: connection.processHandle, DeadlineNanos: processNetworkDeadline(deadline)})
	return err
}

func processNetworkConn(response processNetworkResult) *Conn {
	return &Conn{implementation: &processConn{
		processHandle: response.Handle,
		local:         response.Local,
		remote:        response.Remote,
	}}
}

func processNetworkDeadline(deadline time.Time) int64 {
	if deadline.IsZero() {
		return 0
	}
	return deadline.UnixNano()
}

func exchangeProcessNetwork(request processNetworkCommand) (processNetworkResult, error) {
	domain, err, handled := gomadsim.CurrentNetworkDomain()
	if !handled || err != nil {
		if err == nil {
			err = syscall.ESTALE
		}
		return processNetworkResult{}, err
	}
	encoded, err := encodeProcessNetworkCommand(request)
	if err != nil {
		return processNetworkResult{}, err
	}
	responseBytes, remoteErr, ok := gomadsim.ProcessModelExchange(domain.Node, domain.Incarnation, encoded, maximumProcessNetworkFrameBytes)
	if !ok {
		return processNetworkResult{}, syscall.EIO
	}
	if remoteErr != "" {
		return processNetworkResult{}, errors.New(remoteErr)
	}
	response, err := decodeProcessNetworkResult(responseBytes)
	if err != nil {
		return processNetworkResult{}, err
	}
	return response, response.Err
}

//go:linkname ProcessSimulationNetworkOperation
func ProcessSimulationNetworkOperation(domainToken uint64, encoded []byte) ([]byte, bool) {
	request, err := decodeProcessNetworkCommand(encoded)
	if err != nil {
		return nil, false
	}
	domain, ok := gomadsim.DescribeNetworkDomain(domainToken)
	if !ok {
		return encodeProcessNetworkResponse(processNetworkResult{Err: syscall.ESTALE})
	}
	response := applyProcessNetworkOperation(domain, request)
	return encodeProcessNetworkResponse(response)
}

func applyProcessNetworkOperation(domain gomadsim.NetworkDomain, request processNetworkCommand) processNetworkResult {
	switch request.Operation {
	case processNetworkListenOp:
		listener, err := ListenTCP(request.Network, request.Host, int(request.Port))
		if err != nil {
			return processNetworkResult{Err: err}
		}
		handle, err := registerProcessNetworkResource(processNetworkResource{domain: domain.Token, listener: listener})
		if err != nil {
			return processNetworkResult{Err: errors.Join(err, listener.Close())}
		}
		return processNetworkResult{Handle: handle, Local: listener.Address()}
	case processNetworkDialOp:
		ctx := context.Background()
		cancel := func() {}
		if request.DeadlineNanos != 0 {
			ctx, cancel = context.WithDeadline(ctx, time.Unix(0, request.DeadlineNanos))
		}
		connection, err := DialTCP(ctx, request.Network, request.Host, int(request.Port))
		cancel()
		if err != nil {
			return processNetworkResult{Err: err}
		}
		return registerProcessNetworkConn(domain.Token, connection)
	case processNetworkAcceptOp:
		resource, ok := processNetworkResourceFor(domain.Token, request.Handle, true)
		if !ok {
			return processNetworkResult{Err: syscall.ESTALE}
		}
		connection, err := resource.listener.Accept()
		if err != nil {
			return processNetworkResult{Err: err}
		}
		return registerProcessNetworkConn(domain.Token, connection)
	case processNetworkListenerCloseOp:
		resource, ok := processNetworkResourceFor(domain.Token, request.Handle, true)
		if !ok {
			return processNetworkResult{Err: syscall.ESTALE}
		}
		err := resource.listener.Close()
		if err == nil {
			removeProcessNetworkResource(request.Handle)
		}
		return processNetworkResult{Err: err}
	case processNetworkListenerSetDeadlineOp:
		resource, ok := processNetworkResourceFor(domain.Token, request.Handle, true)
		if !ok {
			return processNetworkResult{Err: syscall.ESTALE}
		}
		return processNetworkResult{Err: resource.listener.SetDeadline(processNetworkTime(request.DeadlineNanos))}
	case processNetworkConnReadOp:
		resource, ok := processNetworkResourceFor(domain.Token, request.Handle, false)
		if !ok {
			return processNetworkResult{Err: syscall.ESTALE}
		}
		buffer := make([]byte, min(request.ReadLength, uint64(maximumChunkBytes)))
		read, err := resource.conn.Read(buffer)
		return processNetworkResult{BytesTransferred: uint64(read), Data: buffer[:read], Err: err}
	case processNetworkConnWriteOp:
		resource, ok := processNetworkResourceFor(domain.Token, request.Handle, false)
		if !ok {
			return processNetworkResult{Err: syscall.ESTALE}
		}
		written, err := resource.conn.Write(request.Data)
		return processNetworkResult{BytesTransferred: uint64(written), Err: err}
	case processNetworkConnCloseOp, processNetworkConnCloseReadOp, processNetworkConnCloseWriteOp, processNetworkConnSetDeadlineOp, processNetworkConnSetReadDeadlineOp, processNetworkConnSetWriteDeadlineOp:
		resource, ok := processNetworkResourceFor(domain.Token, request.Handle, false)
		if !ok {
			return processNetworkResult{Err: syscall.ESTALE}
		}
		var err error
		switch request.Operation {
		case processNetworkConnCloseOp:
			err = resource.conn.Close()
			if err == nil {
				removeProcessNetworkResource(request.Handle)
			}
		case processNetworkConnCloseReadOp:
			err = resource.conn.CloseRead()
		case processNetworkConnCloseWriteOp:
			err = resource.conn.CloseWrite()
		case processNetworkConnSetDeadlineOp:
			err = resource.conn.SetDeadline(processNetworkTime(request.DeadlineNanos))
		case processNetworkConnSetReadDeadlineOp:
			err = resource.conn.SetReadDeadline(processNetworkTime(request.DeadlineNanos))
		case processNetworkConnSetWriteDeadlineOp:
			err = resource.conn.SetWriteDeadline(processNetworkTime(request.DeadlineNanos))
		}
		return processNetworkResult{Err: err}
	default:
		return processNetworkResult{Err: ErrUnsupported}
	}
}

func registerProcessNetworkConn(domain uint64, connection *Conn) processNetworkResult {
	handle, err := registerProcessNetworkResource(processNetworkResource{domain: domain, conn: connection})
	if err != nil {
		return processNetworkResult{Err: errors.Join(err, connection.Close())}
	}
	return processNetworkResult{
		Handle: handle, Local: connection.LocalAddress(), Remote: connection.RemoteAddress(),
	}
}

func registerProcessNetworkResource(resource processNetworkResource) (uint64, error) {
	processNetworkResources.Lock()
	defer processNetworkResources.Unlock()
	if len(processNetworkResources.values) >= maximumProcessNetworkHandles {
		return 0, ErrResourceExhausted
	}
	processNetworkResources.next++
	if processNetworkResources.next == 0 {
		return 0, ErrResourceExhausted
	}
	processNetworkResources.values[processNetworkResources.next] = resource
	return processNetworkResources.next, nil
}

func processNetworkResourceFor(domain, handle uint64, listener bool) (processNetworkResource, bool) {
	processNetworkResources.Lock()
	defer processNetworkResources.Unlock()
	resource, ok := processNetworkResources.values[handle]
	if !ok || resource.domain != domain || listener && resource.listener == nil || !listener && resource.conn == nil {
		return processNetworkResource{}, false
	}
	return resource, true
}

func removeProcessNetworkResource(handle uint64) {
	processNetworkResources.Lock()
	delete(processNetworkResources.values, handle)
	processNetworkResources.Unlock()
}

func revokeProcessNetworkResources(domain uint64) {
	processNetworkResources.Lock()
	for handle, resource := range processNetworkResources.values {
		if resource.domain == domain {
			delete(processNetworkResources.values, handle)
		}
	}
	processNetworkResources.Unlock()
}

func processNetworkTime(nanos int64) time.Time {
	if nanos == 0 {
		return time.Time{}
	}
	return time.Unix(0, nanos)
}
