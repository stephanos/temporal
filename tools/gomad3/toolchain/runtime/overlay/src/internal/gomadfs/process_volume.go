// Copyright 2026 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gomadfs

import (
	"errors"
	"syscall"

	"internal/gomadsim"
)

var processFilesystem = &FS{process: true}

func processResolve(name string) (string, string, error) {
	response, err := exchangeProcessVolume(processVolumeCommand{Operation: processVolumeResolveOp, Path: name})
	return response.Path, response.Base, err
}

func processMkdir(name string, perm uint32, all bool) error {
	operation := processVolumeMkdirOp
	if all {
		operation = processVolumeMkdirAllOp
	}
	_, err := exchangeProcessVolume(processVolumeCommand{Operation: operation, Path: name, Mode: perm})
	return err
}

func processStat(name string) (Entry, error) {
	response, err := exchangeProcessVolume(processVolumeCommand{Operation: processVolumeStatOp, Path: name})
	if err != nil {
		return Entry{}, err
	}
	if len(response.Entries) != 1 {
		return Entry{}, syscall.EIO
	}
	return response.Entries[0], nil
}

func processOpen(name string, flags OpenFlags, perm uint32) (*Handle, error) {
	response, err := exchangeProcessVolume(processVolumeCommand{Operation: processVolumeOpenOp, Path: name, Mode: perm, OpenFlags: flags})
	if err != nil {
		return nil, err
	}
	if response.Handle == 0 || response.Path == "" {
		return nil, syscall.EIO
	}
	return &Handle{fs: processFilesystem, processHandle: response.Handle, name: response.Path}, nil
}

func processPathOperation(command processVolumeCommand) error {
	_, err := exchangeProcessVolume(command)
	return err
}

func processGetwd() string {
	response, err := exchangeProcessVolume(processVolumeCommand{Operation: processVolumeGetwdOp})
	if err != nil {
		return ""
	}
	return response.Path
}

func processHandleRead(handle *Handle, destination []byte, offset int64, at bool) (int, error) {
	if handle.closed {
		return 0, ErrClosed
	}
	operation := processVolumeHandleReadOp
	if at {
		operation = processVolumeHandleReadAtOp
	}
	response, err := exchangeProcessVolume(processVolumeCommand{Operation: operation, Handle: handle.processHandle, Offset: offset, ReadLength: uint64(len(destination))})
	read := copy(destination, response.Data)
	if read != len(response.Data) || uint64(read) != response.BytesTransferred {
		return 0, syscall.EIO
	}
	return read, err
}

func processHandleWrite(handle *Handle, source []byte, offset int64, at bool) (int, error) {
	if handle.closed {
		return 0, ErrClosed
	}
	operation := processVolumeHandleWriteOp
	if at {
		operation = processVolumeHandleWriteAtOp
	}
	response, err := exchangeProcessVolume(processVolumeCommand{Operation: operation, Handle: handle.processHandle, Offset: offset, Data: append([]byte(nil), source...)})
	if response.BytesTransferred > uint64(len(source)) {
		return 0, syscall.EIO
	}
	return int(response.BytesTransferred), err
}

func processHandleOperation(handle *Handle, command processVolumeCommand) (processVolumeResult, error) {
	if handle.closed {
		return processVolumeResult{}, ErrClosed
	}
	command.Handle = handle.processHandle
	return exchangeProcessVolume(command)
}

func processHandleStat(handle *Handle) (Entry, error) {
	response, err := processHandleOperation(handle, processVolumeCommand{Operation: processVolumeHandleStatOp})
	if err != nil {
		return Entry{}, err
	}
	if len(response.Entries) != 1 {
		return Entry{}, syscall.EIO
	}
	return response.Entries[0], nil
}

func processHandleReadDir(handle *Handle, count int) ([]Entry, error) {
	response, err := processHandleOperation(handle, processVolumeCommand{Operation: processVolumeHandleReadDirOp, DirectoryCount: int64(count)})
	if response.Entries == nil {
		response.Entries = []Entry{}
	}
	return response.Entries, err
}

func processHandleMap(handle *Handle, offset int64, length uint64, writable bool) (*Mapping, error) {
	// A process volume serves mapping bytes by copy over the model exchange,
	// so a store through the copy cannot reach the volume; writable mappings
	// stay unsupported there instead of silently detaching.
	if writable {
		return nil, syscall.ENOTSUP
	}
	response, err := processHandleOperation(handle, processVolumeCommand{Operation: processVolumeHandleMapOp, Offset: offset, MapLength: length})
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
		response, err := exchangeProcessVolume(processVolumeCommand{Operation: processVolumeMappingBytesOp, Handle: mapping.processHandle})
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
	_, err := exchangeProcessVolume(processVolumeCommand{Operation: processVolumeMappingCloseOp, Handle: mapping.processHandle})
	if err == nil {
		mapping.closed = true
		mapping.data = nil
	}
	return err
}

func exchangeProcessVolume(request processVolumeCommand) (processVolumeResult, error) {
	domain, err, handled := gomadsim.CurrentNetworkDomain()
	if !handled || err != nil {
		if err == nil {
			err = syscall.ESTALE
		}
		return processVolumeResult{}, err
	}
	encoded, err := encodeProcessVolumeCommand(request)
	if err != nil {
		return processVolumeResult{}, err
	}
	responseBytes, remoteErr, ok := gomadsim.ProcessModelExchange(domain.Node, domain.Incarnation, encoded, maximumProcessVolumeFrameBytes)
	if !ok {
		return processVolumeResult{}, syscall.EIO
	}
	if remoteErr != "" {
		return processVolumeResult{}, errors.New(remoteErr)
	}
	response, err := decodeProcessVolumeResult(responseBytes)
	if err != nil {
		return processVolumeResult{}, err
	}
	return response, response.Err
}
