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
	"syscall"
	"time"
	_ "unsafe"

	"internal/gomadmodelwire"
	"internal/gomadsim"
)

const maximumProcessNetworkHandles = 1 << 20

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
	response, err := exchangeProcessNetwork(gomadmodelwire.Request{Model: gomadmodelwire.ModelNetwork, Operation: gomadmodelwire.NetworkListen, String1: network, String2: host, Int1: int64(port)})
	if err != nil {
		return nil, err
	}
	return &Listener{processHandle: response.Handle, address: Address{IP: response.String1, Port: int(response.Int1)}}, nil
}

func processNetworkDial(ctx context.Context, network, host string, port int) (*Conn, error) {
	deadline := int64(0)
	if value, ok := ctx.Deadline(); ok {
		deadline = value.UnixNano()
	}
	response, err := exchangeProcessNetwork(gomadmodelwire.Request{Model: gomadmodelwire.ModelNetwork, Operation: gomadmodelwire.NetworkDial, String1: network, String2: host, Int1: int64(port), Int2: deadline})
	if err != nil {
		return nil, err
	}
	return processNetworkConn(response), nil
}

func processNetworkAccept(listener *Listener) (*Conn, error) {
	response, err := exchangeProcessNetwork(gomadmodelwire.Request{Model: gomadmodelwire.ModelNetwork, Operation: gomadmodelwire.NetworkAccept, Handle: listener.processHandle})
	if err != nil {
		return nil, err
	}
	return processNetworkConn(response), nil
}

func processNetworkListenerClose(listener *Listener) error {
	_, err := exchangeProcessNetwork(gomadmodelwire.Request{Model: gomadmodelwire.ModelNetwork, Operation: gomadmodelwire.NetworkListenerClose, Handle: listener.processHandle})
	return err
}

func processNetworkListenerSetDeadline(listener *Listener, deadline time.Time) error {
	_, err := exchangeProcessNetwork(gomadmodelwire.Request{Model: gomadmodelwire.ModelNetwork, Operation: gomadmodelwire.NetworkListenerSetDeadline, Handle: listener.processHandle, Int1: processNetworkDeadline(deadline)})
	return err
}

func processNetworkConnRead(connection *Conn, destination []byte) (int, error) {
	if len(destination) == 0 {
		return 0, nil
	}
	response, err := exchangeProcessNetwork(gomadmodelwire.Request{Model: gomadmodelwire.ModelNetwork, Operation: gomadmodelwire.NetworkConnRead, Handle: connection.processHandle, Uint1: uint64(len(destination))})
	read := copy(destination, response.Data)
	if read != len(response.Data) || uint64(read) != response.Uint1 {
		return 0, syscall.EIO
	}
	return read, err
}

func processNetworkConnWrite(connection *Conn, source []byte) (int, error) {
	written := 0
	for len(source) != 0 {
		length := min(len(source), maximumChunkBytes)
		response, err := exchangeProcessNetwork(gomadmodelwire.Request{Model: gomadmodelwire.ModelNetwork, Operation: gomadmodelwire.NetworkConnWrite, Handle: connection.processHandle, Data: append([]byte(nil), source[:length]...)})
		if response.Uint1 > uint64(length) {
			return written, syscall.EIO
		}
		written += int(response.Uint1)
		source = source[response.Uint1:]
		if err != nil {
			return written, err
		}
		if response.Uint1 == 0 {
			return written, io.ErrShortWrite
		}
	}
	return written, nil
}

func processNetworkConnOperation(connection *Conn, operation gomadmodelwire.Operation, deadline time.Time) error {
	_, err := exchangeProcessNetwork(gomadmodelwire.Request{Model: gomadmodelwire.ModelNetwork, Operation: operation, Handle: connection.processHandle, Int1: processNetworkDeadline(deadline)})
	return err
}

func processNetworkConn(response gomadmodelwire.Response) *Conn {
	return &Conn{
		processHandle: response.Handle,
		local:         Address{IP: response.String1, Port: int(response.Int1)},
		remote:        Address{IP: response.String2, Port: int(response.Int2)},
	}
}

func processNetworkDeadline(deadline time.Time) int64 {
	if deadline.IsZero() {
		return 0
	}
	return deadline.UnixNano()
}

func exchangeProcessNetwork(request gomadmodelwire.Request) (gomadmodelwire.Response, error) {
	domain, err, handled := gomadsim.CurrentNetworkDomain()
	if !handled || err != nil {
		if err == nil {
			err = syscall.ESTALE
		}
		return gomadmodelwire.Response{}, err
	}
	encoded, err := gomadmodelwire.EncodeRequest(request)
	if err != nil {
		return gomadmodelwire.Response{}, err
	}
	responseBytes, ok := gomadsim.ProcessModelExchange(domain.Node, domain.Incarnation, encoded, gomadmodelwire.MaximumFrameBytes)
	if !ok {
		return gomadmodelwire.Response{}, syscall.EIO
	}
	response, err := gomadmodelwire.DecodeResponse(responseBytes)
	if err != nil {
		return gomadmodelwire.Response{}, err
	}
	return response, decodeProcessNetworkError(response.Error)
}

//go:linkname ProcessSimulationNetworkOperation
func ProcessSimulationNetworkOperation(domainToken uint64, encoded []byte) ([]byte, bool) {
	request, err := gomadmodelwire.DecodeRequest(encoded)
	if err != nil || request.Model != gomadmodelwire.ModelNetwork {
		return nil, false
	}
	domain, ok := gomadsim.DescribeNetworkDomain(domainToken)
	if !ok {
		return encodeProcessNetworkResponse(gomadmodelwire.Response{Error: encodeProcessNetworkError(syscall.ESTALE)})
	}
	response := applyProcessNetworkOperation(domain, request)
	return encodeProcessNetworkResponse(response)
}

func applyProcessNetworkOperation(domain gomadsim.NetworkDomain, request gomadmodelwire.Request) gomadmodelwire.Response {
	switch request.Operation {
	case gomadmodelwire.NetworkListen:
		listener, err := ListenTCP(request.String1, request.String2, int(request.Int1))
		if err != nil {
			return gomadmodelwire.Response{Error: encodeProcessNetworkError(err)}
		}
		handle, err := registerProcessNetworkResource(processNetworkResource{domain: domain.Token, listener: listener})
		if err != nil {
			return gomadmodelwire.Response{Error: encodeProcessNetworkError(errors.Join(err, listener.Close()))}
		}
		return gomadmodelwire.Response{Handle: handle, String1: listener.address.IP, Int1: int64(listener.address.Port)}
	case gomadmodelwire.NetworkDial:
		ctx := context.Background()
		cancel := func() {}
		if request.Int2 != 0 {
			ctx, cancel = context.WithDeadline(ctx, time.Unix(0, request.Int2))
		}
		connection, err := DialTCP(ctx, request.String1, request.String2, int(request.Int1))
		cancel()
		if err != nil {
			return gomadmodelwire.Response{Error: encodeProcessNetworkError(err)}
		}
		return registerProcessNetworkConn(domain.Token, connection)
	case gomadmodelwire.NetworkAccept:
		resource, ok := processNetworkResourceFor(domain.Token, request.Handle, true)
		if !ok {
			return gomadmodelwire.Response{Error: encodeProcessNetworkError(syscall.ESTALE)}
		}
		connection, err := resource.listener.Accept()
		if err != nil {
			return gomadmodelwire.Response{Error: encodeProcessNetworkError(err)}
		}
		return registerProcessNetworkConn(domain.Token, connection)
	case gomadmodelwire.NetworkListenerClose:
		resource, ok := processNetworkResourceFor(domain.Token, request.Handle, true)
		if !ok {
			return gomadmodelwire.Response{Error: encodeProcessNetworkError(syscall.ESTALE)}
		}
		err := resource.listener.Close()
		if err == nil {
			removeProcessNetworkResource(request.Handle)
		}
		return gomadmodelwire.Response{Error: encodeProcessNetworkError(err)}
	case gomadmodelwire.NetworkListenerSetDeadline:
		resource, ok := processNetworkResourceFor(domain.Token, request.Handle, true)
		if !ok {
			return gomadmodelwire.Response{Error: encodeProcessNetworkError(syscall.ESTALE)}
		}
		return gomadmodelwire.Response{Error: encodeProcessNetworkError(resource.listener.SetDeadline(processNetworkTime(request.Int1)))}
	case gomadmodelwire.NetworkConnRead:
		resource, ok := processNetworkResourceFor(domain.Token, request.Handle, false)
		if !ok {
			return gomadmodelwire.Response{Error: encodeProcessNetworkError(syscall.ESTALE)}
		}
		buffer := make([]byte, min(request.Uint1, uint64(maximumChunkBytes)))
		read, err := resource.conn.Read(buffer)
		return gomadmodelwire.Response{Uint1: uint64(read), Data: buffer[:read], Error: encodeProcessNetworkError(err)}
	case gomadmodelwire.NetworkConnWrite:
		resource, ok := processNetworkResourceFor(domain.Token, request.Handle, false)
		if !ok {
			return gomadmodelwire.Response{Error: encodeProcessNetworkError(syscall.ESTALE)}
		}
		written, err := resource.conn.Write(request.Data)
		return gomadmodelwire.Response{Uint1: uint64(written), Error: encodeProcessNetworkError(err)}
	case gomadmodelwire.NetworkConnClose, gomadmodelwire.NetworkConnCloseRead, gomadmodelwire.NetworkConnCloseWrite, gomadmodelwire.NetworkConnSetDeadline, gomadmodelwire.NetworkConnSetReadDeadline, gomadmodelwire.NetworkConnSetWriteDeadline:
		resource, ok := processNetworkResourceFor(domain.Token, request.Handle, false)
		if !ok {
			return gomadmodelwire.Response{Error: encodeProcessNetworkError(syscall.ESTALE)}
		}
		var err error
		switch request.Operation {
		case gomadmodelwire.NetworkConnClose:
			err = resource.conn.Close()
			if err == nil {
				removeProcessNetworkResource(request.Handle)
			}
		case gomadmodelwire.NetworkConnCloseRead:
			err = resource.conn.CloseRead()
		case gomadmodelwire.NetworkConnCloseWrite:
			err = resource.conn.CloseWrite()
		case gomadmodelwire.NetworkConnSetDeadline:
			err = resource.conn.SetDeadline(processNetworkTime(request.Int1))
		case gomadmodelwire.NetworkConnSetReadDeadline:
			err = resource.conn.SetReadDeadline(processNetworkTime(request.Int1))
		case gomadmodelwire.NetworkConnSetWriteDeadline:
			err = resource.conn.SetWriteDeadline(processNetworkTime(request.Int1))
		}
		return gomadmodelwire.Response{Error: encodeProcessNetworkError(err)}
	default:
		return gomadmodelwire.Response{Error: encodeProcessNetworkError(ErrUnsupported)}
	}
}

func registerProcessNetworkConn(domain uint64, connection *Conn) gomadmodelwire.Response {
	handle, err := registerProcessNetworkResource(processNetworkResource{domain: domain, conn: connection})
	if err != nil {
		return gomadmodelwire.Response{Error: encodeProcessNetworkError(errors.Join(err, connection.Close()))}
	}
	return gomadmodelwire.Response{
		Handle: handle, String1: connection.local.IP, Int1: int64(connection.local.Port),
		String2: connection.remote.IP, Int2: int64(connection.remote.Port),
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

func encodeProcessNetworkResponse(response gomadmodelwire.Response) ([]byte, bool) {
	encoded, err := gomadmodelwire.EncodeResponse(response)
	return encoded, err == nil
}

func encodeProcessNetworkError(err error) gomadmodelwire.WireError {
	if err == nil {
		return gomadmodelwire.WireError{}
	}
	result := gomadmodelwire.WireError{Code: gomadmodelwire.ErrorGeneric, Message: err.Error()}
	switch {
	case errors.Is(err, io.EOF):
		result.Code = gomadmodelwire.ErrorEOF
	case errors.Is(err, os.ErrDeadlineExceeded), errors.Is(err, context.DeadlineExceeded):
		result.Code = gomadmodelwire.ErrorDeadline
	case errors.Is(err, context.Canceled):
		result.Code = gomadmodelwire.ErrorCanceled
	case errors.Is(err, ErrAddressInUse):
		result.Code = gomadmodelwire.ErrorAddressInUse
	case errors.Is(err, ErrClosed):
		result.Code = gomadmodelwire.ErrorClosed
	case errors.Is(err, ErrConnectionRefused):
		result.Code = gomadmodelwire.ErrorConnectionRefused
	case errors.Is(err, ErrResourceExhausted):
		result.Code = gomadmodelwire.ErrorResourceExhausted
	case errors.Is(err, ErrUnsupported):
		result.Code = gomadmodelwire.ErrorUnsupported
	case errors.Is(err, syscall.ESTALE):
		result.Code = gomadmodelwire.ErrorESTALE
	}
	return result
}

func decodeProcessNetworkError(source gomadmodelwire.WireError) error {
	switch source.Code {
	case gomadmodelwire.ErrorNone:
		return nil
	case gomadmodelwire.ErrorEOF:
		return io.EOF
	case gomadmodelwire.ErrorDeadline:
		return os.ErrDeadlineExceeded
	case gomadmodelwire.ErrorCanceled:
		return context.Canceled
	case gomadmodelwire.ErrorAddressInUse:
		return ErrAddressInUse
	case gomadmodelwire.ErrorClosed:
		return ErrClosed
	case gomadmodelwire.ErrorConnectionRefused:
		return ErrConnectionRefused
	case gomadmodelwire.ErrorResourceExhausted:
		return ErrResourceExhausted
	case gomadmodelwire.ErrorUnsupported:
		return ErrUnsupported
	case gomadmodelwire.ErrorESTALE:
		return syscall.ESTALE
	default:
		return errors.New(source.Message)
	}
}

func processNetworkTime(nanos int64) time.Time {
	if nanos == 0 {
		return time.Time{}
	}
	return time.Unix(0, nanos)
}
