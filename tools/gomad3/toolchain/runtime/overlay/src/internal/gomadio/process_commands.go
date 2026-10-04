// Copyright 2026 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gomadio

import (
	"context"
	"errors"
	"io"
	"os"
	"syscall"

	"internal/gomadmodelwire"
)

type processNetworkOperation uint16

const (
	processNetworkListenOp               processNetworkOperation = processNetworkOperation(gomadmodelwire.NetworkListen)
	processNetworkDialOp                 processNetworkOperation = processNetworkOperation(gomadmodelwire.NetworkDial)
	processNetworkAcceptOp               processNetworkOperation = processNetworkOperation(gomadmodelwire.NetworkAccept)
	processNetworkListenerCloseOp        processNetworkOperation = processNetworkOperation(gomadmodelwire.NetworkListenerClose)
	processNetworkListenerSetDeadlineOp  processNetworkOperation = processNetworkOperation(gomadmodelwire.NetworkListenerSetDeadline)
	processNetworkConnReadOp             processNetworkOperation = processNetworkOperation(gomadmodelwire.NetworkConnRead)
	processNetworkConnWriteOp            processNetworkOperation = processNetworkOperation(gomadmodelwire.NetworkConnWrite)
	processNetworkConnCloseOp            processNetworkOperation = processNetworkOperation(gomadmodelwire.NetworkConnClose)
	processNetworkConnCloseReadOp        processNetworkOperation = processNetworkOperation(gomadmodelwire.NetworkConnCloseRead)
	processNetworkConnCloseWriteOp       processNetworkOperation = processNetworkOperation(gomadmodelwire.NetworkConnCloseWrite)
	processNetworkConnSetDeadlineOp      processNetworkOperation = processNetworkOperation(gomadmodelwire.NetworkConnSetDeadline)
	processNetworkConnSetReadDeadlineOp  processNetworkOperation = processNetworkOperation(gomadmodelwire.NetworkConnSetReadDeadline)
	processNetworkConnSetWriteDeadlineOp processNetworkOperation = processNetworkOperation(gomadmodelwire.NetworkConnSetWriteDeadline)
	maximumProcessNetworkFrameBytes                              = gomadmodelwire.MaximumFrameBytes
)

type processNetworkCommand struct {
	Operation           processNetworkOperation
	Handle              uint64
	Network, Host       string
	Port, DeadlineNanos int64
	ReadLength          uint64
	Data                []byte
}

type processNetworkResult struct {
	Handle           uint64
	Local, Remote    Address
	BytesTransferred uint64
	Data             []byte
	Err              error
}

func encodeProcessNetworkCommand(command processNetworkCommand) ([]byte, error) {
	request := gomadmodelwire.Request{Model: gomadmodelwire.ModelNetwork, Operation: gomadmodelwire.Operation(command.Operation), Handle: command.Handle}
	switch command.Operation {
	case processNetworkListenOp, processNetworkDialOp:
		request.String1 = command.Network
		request.String2 = command.Host
		request.Int1 = command.Port
		if command.Operation == processNetworkDialOp {
			request.Int2 = command.DeadlineNanos
		}
	case processNetworkListenerSetDeadlineOp, processNetworkConnSetDeadlineOp, processNetworkConnSetReadDeadlineOp, processNetworkConnSetWriteDeadlineOp:
		request.Int1 = command.DeadlineNanos
	case processNetworkConnReadOp:
		request.Uint1 = command.ReadLength
	case processNetworkConnWriteOp:
		request.Data = command.Data
	}
	return gomadmodelwire.EncodeRequest(request)
}

func decodeProcessNetworkCommand(encoded []byte) (processNetworkCommand, error) {
	request, err := gomadmodelwire.DecodeRequest(encoded)
	if err != nil {
		return processNetworkCommand{}, err
	}
	if request.Model != gomadmodelwire.ModelNetwork {
		return processNetworkCommand{}, errors.New("simulation model is not network")
	}
	command := processNetworkCommand{Operation: processNetworkOperation(request.Operation), Handle: request.Handle}
	switch command.Operation {
	case processNetworkListenOp, processNetworkDialOp:
		command.Network = request.String1
		command.Host = request.String2
		command.Port = request.Int1
		if command.Operation == processNetworkDialOp {
			command.DeadlineNanos = request.Int2
		}
	case processNetworkListenerSetDeadlineOp, processNetworkConnSetDeadlineOp, processNetworkConnSetReadDeadlineOp, processNetworkConnSetWriteDeadlineOp:
		command.DeadlineNanos = request.Int1
	case processNetworkConnReadOp:
		command.ReadLength = request.Uint1
	case processNetworkConnWriteOp:
		command.Data = request.Data
	}
	return command, nil
}

func encodeProcessNetworkResponse(result processNetworkResult) ([]byte, bool) {
	encoded, err := gomadmodelwire.EncodeResponse(gomadmodelwire.Response{
		Handle: result.Handle, String1: result.Local.IP, Int1: int64(result.Local.Port), String2: result.Remote.IP, Int2: int64(result.Remote.Port),
		Uint1: result.BytesTransferred, Data: result.Data, Error: encodeProcessNetworkError(result.Err),
	})
	return encoded, err == nil
}

func decodeProcessNetworkResult(encoded []byte) (processNetworkResult, error) {
	response, err := gomadmodelwire.DecodeResponse(encoded)
	if err != nil {
		return processNetworkResult{}, err
	}
	return processNetworkResult{Handle: response.Handle, Local: Address{IP: response.String1, Port: int(response.Int1)}, Remote: Address{IP: response.String2, Port: int(response.Int2)}, BytesTransferred: response.Uint1, Data: response.Data, Err: decodeProcessNetworkError(response.Error)}, nil
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
