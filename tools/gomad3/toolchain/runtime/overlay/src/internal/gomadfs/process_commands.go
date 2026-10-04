// Copyright 2026 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gomadfs

import (
	"errors"
	"io"
	"syscall"

	"internal/gomadmodelwire"
)

type processVolumeOperation uint16

const (
	processVolumeResolveOp         processVolumeOperation = processVolumeOperation(gomadmodelwire.VolumeResolve)
	processVolumeMkdirOp           processVolumeOperation = processVolumeOperation(gomadmodelwire.VolumeMkdir)
	processVolumeMkdirAllOp        processVolumeOperation = processVolumeOperation(gomadmodelwire.VolumeMkdirAll)
	processVolumeStatOp            processVolumeOperation = processVolumeOperation(gomadmodelwire.VolumeStat)
	processVolumeOpenOp            processVolumeOperation = processVolumeOperation(gomadmodelwire.VolumeOpen)
	processVolumeRenameOp          processVolumeOperation = processVolumeOperation(gomadmodelwire.VolumeRename)
	processVolumeRemoveOp          processVolumeOperation = processVolumeOperation(gomadmodelwire.VolumeRemove)
	processVolumeRemoveAllOp       processVolumeOperation = processVolumeOperation(gomadmodelwire.VolumeRemoveAll)
	processVolumeChmodOp           processVolumeOperation = processVolumeOperation(gomadmodelwire.VolumeChmod)
	processVolumeChtimesOp         processVolumeOperation = processVolumeOperation(gomadmodelwire.VolumeChtimes)
	processVolumeChdirOp           processVolumeOperation = processVolumeOperation(gomadmodelwire.VolumeChdir)
	processVolumeGetwdOp           processVolumeOperation = processVolumeOperation(gomadmodelwire.VolumeGetwd)
	processVolumeHandleReadOp      processVolumeOperation = processVolumeOperation(gomadmodelwire.VolumeHandleRead)
	processVolumeHandleReadAtOp    processVolumeOperation = processVolumeOperation(gomadmodelwire.VolumeHandleReadAt)
	processVolumeHandleWriteOp     processVolumeOperation = processVolumeOperation(gomadmodelwire.VolumeHandleWrite)
	processVolumeHandleWriteAtOp   processVolumeOperation = processVolumeOperation(gomadmodelwire.VolumeHandleWriteAt)
	processVolumeHandleTruncateOp  processVolumeOperation = processVolumeOperation(gomadmodelwire.VolumeHandleTruncate)
	processVolumeHandleChmodOp     processVolumeOperation = processVolumeOperation(gomadmodelwire.VolumeHandleChmod)
	processVolumeHandleChtimesOp   processVolumeOperation = processVolumeOperation(gomadmodelwire.VolumeHandleChtimes)
	processVolumeHandleChdirOp     processVolumeOperation = processVolumeOperation(gomadmodelwire.VolumeHandleChdir)
	processVolumeHandleSeekOp      processVolumeOperation = processVolumeOperation(gomadmodelwire.VolumeHandleSeek)
	processVolumeHandleStatOp      processVolumeOperation = processVolumeOperation(gomadmodelwire.VolumeHandleStat)
	processVolumeHandleReadDirOp   processVolumeOperation = processVolumeOperation(gomadmodelwire.VolumeHandleReadDir)
	processVolumeHandleCloseOp     processVolumeOperation = processVolumeOperation(gomadmodelwire.VolumeHandleClose)
	processVolumeHandleSyncOp      processVolumeOperation = processVolumeOperation(gomadmodelwire.VolumeHandleSync)
	processVolumeHandleMapOp       processVolumeOperation = processVolumeOperation(gomadmodelwire.VolumeHandleMap)
	processVolumeMappingBytesOp    processVolumeOperation = processVolumeOperation(gomadmodelwire.VolumeMappingBytes)
	processVolumeMappingCloseOp    processVolumeOperation = processVolumeOperation(gomadmodelwire.VolumeMappingClose)
	maximumProcessVolumeFrameBytes                        = gomadmodelwire.MaximumFrameBytes
	maximumProcessVolumeDataBytes                         = gomadmodelwire.MaximumDataBytes
)
const (
	processOpenRead uint64 = 1 << iota
	processOpenWrite
	processOpenAppend
	processOpenCreate
	processOpenExclusive
	processOpenTruncate
)

type processVolumeCommand struct {
	Operation                                     processVolumeOperation
	Handle                                        uint64
	Path, Destination                             string
	Mode                                          uint32
	OpenFlags                                     OpenFlags
	ModTime, Offset, Size, Whence, DirectoryCount int64
	ReadLength, MapLength                         uint64
	Data                                          []byte
}

type processVolumeResult struct {
	Handle           uint64
	Path, Base       string
	Offset           int64
	BytesTransferred uint64
	Data             []byte
	Entries          []Entry
	Err              error
}

func encodeProcessVolumeCommand(command processVolumeCommand) ([]byte, error) {
	request := gomadmodelwire.Request{Model: gomadmodelwire.ModelVolume, Operation: gomadmodelwire.Operation(command.Operation), Handle: command.Handle}
	switch command.Operation {
	case processVolumeResolveOp, processVolumeStatOp, processVolumeRemoveOp, processVolumeRemoveAllOp, processVolumeChdirOp:
		request.String1 = command.Path
	case processVolumeRenameOp:
		request.String1 = command.Path
		request.String2 = command.Destination
	case processVolumeMkdirOp, processVolumeMkdirAllOp, processVolumeChmodOp, processVolumeOpenOp:
		request.String1 = command.Path
		request.Uint1 = uint64(command.Mode)
	case processVolumeChtimesOp:
		request.String1 = command.Path
		request.Int1 = command.ModTime
	case processVolumeHandleReadOp, processVolumeHandleReadAtOp:
		request.Int1 = command.Offset
		request.Uint1 = command.ReadLength
	case processVolumeHandleWriteOp, processVolumeHandleWriteAtOp:
		request.Int1 = command.Offset
		request.Data = command.Data
	case processVolumeHandleTruncateOp:
		request.Int1 = command.Size
	case processVolumeHandleChmodOp:
		request.Uint1 = uint64(command.Mode)
	case processVolumeHandleChtimesOp:
		request.Int1 = command.ModTime
	case processVolumeHandleSeekOp:
		request.Int1 = command.Offset
		request.Int2 = command.Whence
	case processVolumeHandleReadDirOp:
		request.Int1 = command.DirectoryCount
	case processVolumeHandleMapOp:
		request.Int1 = command.Offset
		request.Uint1 = command.MapLength
	}
	if command.Operation == processVolumeOpenOp {
		flags := command.OpenFlags
		if flags.Read {
			request.Flags |= processOpenRead
		}
		if flags.Write {
			request.Flags |= processOpenWrite
		}
		if flags.Append {
			request.Flags |= processOpenAppend
		}
		if flags.Create {
			request.Flags |= processOpenCreate
		}
		if flags.Exclusive {
			request.Flags |= processOpenExclusive
		}
		if flags.Truncate {
			request.Flags |= processOpenTruncate
		}
	}
	return gomadmodelwire.EncodeRequest(request)
}

func decodeProcessVolumeCommand(encoded []byte) (processVolumeCommand, error) {
	request, err := gomadmodelwire.DecodeRequest(encoded)
	if err != nil {
		return processVolumeCommand{}, err
	}
	if request.Model != gomadmodelwire.ModelVolume {
		return processVolumeCommand{}, errors.New("simulation model is not volume")
	}
	command := processVolumeCommand{Operation: processVolumeOperation(request.Operation), Handle: request.Handle}
	switch command.Operation {
	case processVolumeResolveOp, processVolumeStatOp, processVolumeRemoveOp, processVolumeRemoveAllOp, processVolumeChdirOp:
		command.Path = request.String1
	case processVolumeRenameOp:
		command.Path = request.String1
		command.Destination = request.String2
	case processVolumeMkdirOp, processVolumeMkdirAllOp, processVolumeChmodOp, processVolumeOpenOp:
		command.Path = request.String1
		command.Mode = uint32(request.Uint1)
	case processVolumeChtimesOp:
		command.Path = request.String1
		command.ModTime = request.Int1
	case processVolumeHandleReadOp, processVolumeHandleReadAtOp:
		command.Offset = request.Int1
		command.ReadLength = request.Uint1
	case processVolumeHandleWriteOp, processVolumeHandleWriteAtOp:
		command.Offset = request.Int1
		command.Data = request.Data
	case processVolumeHandleTruncateOp:
		command.Size = request.Int1
	case processVolumeHandleChmodOp:
		command.Mode = uint32(request.Uint1)
	case processVolumeHandleChtimesOp:
		command.ModTime = request.Int1
	case processVolumeHandleSeekOp:
		command.Offset = request.Int1
		command.Whence = request.Int2
	case processVolumeHandleReadDirOp:
		command.DirectoryCount = request.Int1
	case processVolumeHandleMapOp:
		command.Offset = request.Int1
		command.MapLength = request.Uint1
	}
	if command.Operation == processVolumeOpenOp {
		command.OpenFlags = OpenFlags{Read: request.Flags&processOpenRead != 0, Write: request.Flags&processOpenWrite != 0, Append: request.Flags&processOpenAppend != 0, Create: request.Flags&processOpenCreate != 0, Exclusive: request.Flags&processOpenExclusive != 0, Truncate: request.Flags&processOpenTruncate != 0}
	}
	return command, nil
}

func encodeProcessVolumeResponse(result processVolumeResult) ([]byte, bool) {
	response := gomadmodelwire.Response{Handle: result.Handle, String1: result.Path, String2: result.Base, Int1: result.Offset, Uint1: result.BytesTransferred, Data: result.Data, Error: encodeProcessVolumeError(result.Err)}
	response.Entries = make([]gomadmodelwire.Entry, len(result.Entries))
	for index := range result.Entries {
		response.Entries[index] = processWireEntry(result.Entries[index])
	}
	encoded, err := gomadmodelwire.EncodeResponse(response)
	return encoded, err == nil
}

func decodeProcessVolumeResult(encoded []byte) (processVolumeResult, error) {
	response, err := gomadmodelwire.DecodeResponse(encoded)
	if err != nil {
		return processVolumeResult{}, err
	}
	result := processVolumeResult{Handle: response.Handle, Path: response.String1, Base: response.String2, Offset: response.Int1, BytesTransferred: response.Uint1, Data: response.Data, Err: decodeProcessVolumeError(response.Error)}
	result.Entries = make([]Entry, len(response.Entries))
	for index := range response.Entries {
		result.Entries[index] = processEntry(response.Entries[index])
	}
	return result, nil
}

func processEntry(entry gomadmodelwire.Entry) Entry {
	return Entry{Name: entry.Name, Mode: entry.Mode, Kind: Kind(entry.Kind), ModTime: entry.ModTime, Data: append([]byte(nil), entry.Data...)}
}

func decodeProcessVolumeError(source gomadmodelwire.WireError) error {
	switch source.Code {
	case gomadmodelwire.ErrorNone:
		return nil
	case gomadmodelwire.ErrorEOF:
		return io.EOF
	case gomadmodelwire.ErrorUnsupported:
		return syscall.ENOTSUP
	case gomadmodelwire.ErrorEINVAL:
		return syscall.EINVAL
	case gomadmodelwire.ErrorEEXIST:
		return syscall.EEXIST
	case gomadmodelwire.ErrorENOENT:
		return syscall.ENOENT
	case gomadmodelwire.ErrorENOTDIR:
		return syscall.ENOTDIR
	case gomadmodelwire.ErrorEISDIR:
		return syscall.EISDIR
	case gomadmodelwire.ErrorEROFS:
		return syscall.EROFS
	case gomadmodelwire.ErrorENOSPC:
		return syscall.ENOSPC
	case gomadmodelwire.ErrorEBADF:
		return syscall.EBADF
	case gomadmodelwire.ErrorENODEV:
		return syscall.ENODEV
	case gomadmodelwire.ErrorESTALE:
		return syscall.ESTALE
	case gomadmodelwire.ErrorENOTEMPTY:
		return syscall.ENOTEMPTY
	case gomadmodelwire.ErrorCapacity:
		return &VolumeCapacityError{Resource: source.Resource, Required: source.Required, Maximum: source.Maximum}
	}
	switch source.Message {
	case syscall.EBUSY.Error():
		return syscall.EBUSY
	case syscall.EFBIG.Error():
		return syscall.EFBIG
	case syscall.EMFILE.Error():
		return syscall.EMFILE
	case syscall.EPROTO.Error():
		return syscall.EPROTO
	case syscall.EXDEV.Error():
		return syscall.EXDEV
	}
	return errors.New(source.Message)
}

func processWireEntry(entry Entry) gomadmodelwire.Entry {
	return gomadmodelwire.Entry{Name: entry.Name, Mode: entry.Mode, Kind: uint8(entry.Kind), ModTime: entry.ModTime, Data: append([]byte(nil), entry.Data...)}
}

func encodeProcessVolumeError(err error) gomadmodelwire.WireError {
	if err == nil {
		return gomadmodelwire.WireError{}
	}
	result := gomadmodelwire.WireError{Code: gomadmodelwire.ErrorGeneric, Message: err.Error()}
	var capacity *VolumeCapacityError
	switch {
	case errors.As(err, &capacity):
		result.Code = gomadmodelwire.ErrorCapacity
		result.Resource = capacity.Resource
		result.Required = capacity.Required
		result.Maximum = capacity.Maximum
	case errors.Is(err, io.EOF):
		result.Code = gomadmodelwire.ErrorEOF
	case errors.Is(err, syscall.ENOTSUP):
		result.Code = gomadmodelwire.ErrorUnsupported
	case errors.Is(err, syscall.EINVAL):
		result.Code = gomadmodelwire.ErrorEINVAL
	case errors.Is(err, syscall.EEXIST):
		result.Code = gomadmodelwire.ErrorEEXIST
	case errors.Is(err, syscall.ENOENT):
		result.Code = gomadmodelwire.ErrorENOENT
	case errors.Is(err, syscall.ENOTDIR):
		result.Code = gomadmodelwire.ErrorENOTDIR
	case errors.Is(err, syscall.EISDIR):
		result.Code = gomadmodelwire.ErrorEISDIR
	case errors.Is(err, syscall.EROFS):
		result.Code = gomadmodelwire.ErrorEROFS
	case errors.Is(err, syscall.ENOSPC):
		result.Code = gomadmodelwire.ErrorENOSPC
	case errors.Is(err, syscall.EBADF):
		result.Code = gomadmodelwire.ErrorEBADF
	case errors.Is(err, syscall.ENODEV):
		result.Code = gomadmodelwire.ErrorENODEV
	case errors.Is(err, syscall.ESTALE):
		result.Code = gomadmodelwire.ErrorESTALE
	case errors.Is(err, syscall.ENOTEMPTY):
		result.Code = gomadmodelwire.ErrorENOTEMPTY
	}
	return result
}
