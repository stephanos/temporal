// Copyright 2026 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gomadfs

import (
	"errors"
	"io"
	"syscall"

	"internal/gomadmodelwire"
	"internal/gomadsim"
)

const (
	processOpenRead uint64 = 1 << iota
	processOpenWrite
	processOpenAppend
	processOpenCreate
	processOpenExclusive
	processOpenTruncate
)

var processFilesystem = &FS{process: true}

func processResolve(name string) (string, string, error) {
	response, err := exchangeProcessVolume(gomadmodelwire.Request{Model: gomadmodelwire.ModelVolume, Operation: gomadmodelwire.VolumeResolve, String1: name})
	return response.String1, response.String2, err
}

func processMkdir(name string, perm uint32, all bool) error {
	operation := gomadmodelwire.VolumeMkdir
	if all {
		operation = gomadmodelwire.VolumeMkdirAll
	}
	_, err := exchangeProcessVolume(gomadmodelwire.Request{Model: gomadmodelwire.ModelVolume, Operation: operation, String1: name, Uint1: uint64(perm)})
	return err
}

func processStat(name string) (Entry, error) {
	response, err := exchangeProcessVolume(gomadmodelwire.Request{Model: gomadmodelwire.ModelVolume, Operation: gomadmodelwire.VolumeStat, String1: name})
	if err != nil {
		return Entry{}, err
	}
	if len(response.Entries) != 1 {
		return Entry{}, syscall.EIO
	}
	return processEntry(response.Entries[0]), nil
}

func processOpen(name string, flags OpenFlags, perm uint32) (*Handle, error) {
	var encodedFlags uint64
	if flags.Read {
		encodedFlags |= processOpenRead
	}
	if flags.Write {
		encodedFlags |= processOpenWrite
	}
	if flags.Append {
		encodedFlags |= processOpenAppend
	}
	if flags.Create {
		encodedFlags |= processOpenCreate
	}
	if flags.Exclusive {
		encodedFlags |= processOpenExclusive
	}
	if flags.Truncate {
		encodedFlags |= processOpenTruncate
	}
	response, err := exchangeProcessVolume(gomadmodelwire.Request{Model: gomadmodelwire.ModelVolume, Operation: gomadmodelwire.VolumeOpen, String1: name, Uint1: uint64(perm), Flags: encodedFlags})
	if err != nil {
		return nil, err
	}
	if response.Handle == 0 || response.String1 == "" {
		return nil, syscall.EIO
	}
	return &Handle{fs: processFilesystem, processHandle: response.Handle, name: response.String1}, nil
}

func processPathOperation(operation gomadmodelwire.Operation, first, second string, integer int64, unsigned uint64) error {
	_, err := exchangeProcessVolume(gomadmodelwire.Request{Model: gomadmodelwire.ModelVolume, Operation: operation, String1: first, String2: second, Int1: integer, Uint1: unsigned})
	return err
}

func processGetwd() string {
	response, err := exchangeProcessVolume(gomadmodelwire.Request{Model: gomadmodelwire.ModelVolume, Operation: gomadmodelwire.VolumeGetwd})
	if err != nil {
		return ""
	}
	return response.String1
}

func processHandleRead(handle *Handle, destination []byte, offset int64, at bool) (int, error) {
	if handle.closed {
		return 0, syscall.EINVAL
	}
	operation := gomadmodelwire.VolumeHandleRead
	if at {
		operation = gomadmodelwire.VolumeHandleReadAt
	}
	response, err := exchangeProcessVolume(gomadmodelwire.Request{Model: gomadmodelwire.ModelVolume, Operation: operation, Handle: handle.processHandle, Int1: offset, Uint1: uint64(len(destination))})
	read := copy(destination, response.Data)
	if read != len(response.Data) || uint64(read) != response.Uint1 {
		return 0, syscall.EIO
	}
	return read, err
}

func processHandleWrite(handle *Handle, source []byte, offset int64, at bool) (int, error) {
	if handle.closed {
		return 0, syscall.EINVAL
	}
	operation := gomadmodelwire.VolumeHandleWrite
	if at {
		operation = gomadmodelwire.VolumeHandleWriteAt
	}
	response, err := exchangeProcessVolume(gomadmodelwire.Request{Model: gomadmodelwire.ModelVolume, Operation: operation, Handle: handle.processHandle, Int1: offset, Data: append([]byte(nil), source...)})
	if response.Uint1 > uint64(len(source)) {
		return 0, syscall.EIO
	}
	return int(response.Uint1), err
}

func processHandleOperation(handle *Handle, operation gomadmodelwire.Operation, first, second int64, unsigned uint64) (gomadmodelwire.Response, error) {
	if handle.closed {
		return gomadmodelwire.Response{}, syscall.EINVAL
	}
	return exchangeProcessVolume(gomadmodelwire.Request{Model: gomadmodelwire.ModelVolume, Operation: operation, Handle: handle.processHandle, Int1: first, Int2: second, Uint1: unsigned})
}

func processHandleStat(handle *Handle) (Entry, error) {
	response, err := processHandleOperation(handle, gomadmodelwire.VolumeHandleStat, 0, 0, 0)
	if err != nil {
		return Entry{}, err
	}
	if len(response.Entries) != 1 {
		return Entry{}, syscall.EIO
	}
	return processEntry(response.Entries[0]), nil
}

func processHandleReadDir(handle *Handle, count int) ([]Entry, error) {
	response, err := processHandleOperation(handle, gomadmodelwire.VolumeHandleReadDir, int64(count), 0, 0)
	entries := make([]Entry, len(response.Entries))
	for index := range response.Entries {
		entries[index] = processEntry(response.Entries[index])
	}
	return entries, err
}

func processHandleMap(handle *Handle, length uint64) (*Mapping, error) {
	response, err := processHandleOperation(handle, gomadmodelwire.VolumeHandleMap, 0, 0, length)
	if err != nil {
		return nil, err
	}
	if response.Handle == 0 {
		return nil, syscall.EIO
	}
	return &Mapping{fs: processFilesystem, processHandle: response.Handle}, nil
}

func processMappingBytes(mapping *Mapping) ([]byte, error) {
	if mapping.closed {
		return nil, syscall.EINVAL
	}
	if mapping.data == nil {
		response, err := exchangeProcessVolume(gomadmodelwire.Request{Model: gomadmodelwire.ModelVolume, Operation: gomadmodelwire.VolumeMappingBytes, Handle: mapping.processHandle})
		if err != nil {
			return nil, err
		}
		mapping.data = append([]byte(nil), response.Data...)
	}
	return mapping.data, nil
}

func processMappingClose(mapping *Mapping) error {
	if mapping.closed {
		return syscall.EINVAL
	}
	_, err := exchangeProcessVolume(gomadmodelwire.Request{Model: gomadmodelwire.ModelVolume, Operation: gomadmodelwire.VolumeMappingClose, Handle: mapping.processHandle})
	if err == nil {
		mapping.closed = true
		mapping.data = nil
	}
	return err
}

func exchangeProcessVolume(request gomadmodelwire.Request) (gomadmodelwire.Response, error) {
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
	responseBytes, remoteErr, ok := gomadsim.ProcessModelExchange(domain.Node, domain.Incarnation, encoded, gomadmodelwire.MaximumFrameBytes)
	if !ok {
		return gomadmodelwire.Response{}, syscall.EIO
	}
	if remoteErr != "" {
		return gomadmodelwire.Response{}, errors.New(remoteErr)
	}
	response, err := gomadmodelwire.DecodeResponse(responseBytes)
	if err != nil {
		return gomadmodelwire.Response{}, err
	}
	return response, decodeProcessVolumeError(response.Error)
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
