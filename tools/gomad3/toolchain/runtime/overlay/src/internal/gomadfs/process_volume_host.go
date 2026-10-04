// Copyright 2026 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gomadfs

import (
	"errors"
	"sync"
	"syscall"
	_ "unsafe"

	"internal/gomadsim"
)

type processVolumeResource struct {
	domain  uint64
	handle  *Handle
	mapping *Mapping
}

var processVolumeResources = struct {
	sync.Mutex
	next   uint64
	values map[uint64]processVolumeResource
}{values: make(map[uint64]processVolumeResource)}

//go:linkname ProcessSimulationVolumeOperation
func ProcessSimulationVolumeOperation(domainToken uint64, encoded []byte) ([]byte, bool) {
	request, err := decodeProcessVolumeCommand(encoded)
	if err != nil {
		return nil, false
	}
	if _, ok := gomadsim.DescribeNetworkDomain(domainToken); !ok {
		return encodeProcessVolumeResponse(processVolumeResult{Err: syscall.ESTALE})
	}
	return encodeProcessVolumeResponse(applyProcessVolumeOperation(domainToken, Current(), request))
}

func applyProcessVolumeOperation(domain uint64, filesystem *FS, request processVolumeCommand) processVolumeResult {
	switch request.Operation {
	case processVolumeResolveOp:
		path, base, err := filesystem.Resolve(request.Path)
		return processVolumeResult{Path: path, Base: base, Err: err}
	case processVolumeMkdirOp, processVolumeMkdirAllOp:
		var err error
		if request.Operation == processVolumeMkdirOp {
			err = filesystem.Mkdir(request.Path, request.Mode)
		} else {
			err = filesystem.MkdirAll(request.Path, request.Mode)
		}
		return processVolumeResult{Err: err}
	case processVolumeStatOp:
		entry, err := filesystem.Stat(request.Path)
		return processVolumeEntryResponse(entry, err)
	case processVolumeOpenOp:
		handle, err := filesystem.Open(request.Path, request.OpenFlags, request.Mode)
		if err != nil {
			return processVolumeResult{Err: err}
		}
		resource, err := registerProcessVolumeResource(processVolumeResource{domain: domain, handle: handle})
		if err != nil {
			return processVolumeResult{Err: errors.Join(err, handle.Close())}
		}
		return processVolumeResult{Handle: resource, Path: handle.Path()}
	case processVolumeRenameOp:
		return processVolumeResult{Err: filesystem.Rename(request.Path, request.Destination)}
	case processVolumeRemoveOp:
		return processVolumeResult{Err: filesystem.Remove(request.Path)}
	case processVolumeRemoveAllOp:
		return processVolumeResult{Err: filesystem.RemoveAll(request.Path)}
	case processVolumeChmodOp:
		return processVolumeResult{Err: filesystem.Chmod(request.Path, request.Mode)}
	case processVolumeChtimesOp:
		return processVolumeResult{Err: filesystem.Chtimes(request.Path, request.ModTime)}
	case processVolumeChdirOp:
		return processVolumeResult{Err: filesystem.Chdir(request.Path)}
	case processVolumeGetwdOp:
		return processVolumeResult{Path: filesystem.Getwd()}
	}
	return applyProcessVolumeResourceOperation(domain, request)
}

func applyProcessVolumeResourceOperation(domain uint64, request processVolumeCommand) processVolumeResult {
	resource, ok := processVolumeResourceFor(domain, request.Handle)
	if !ok {
		return processVolumeResult{Err: syscall.ESTALE}
	}
	if resource.handle != nil {
		return applyProcessVolumeHandleOperation(resource, request)
	}
	if resource.mapping != nil {
		return applyProcessVolumeMappingOperation(resource, request)
	}
	return processVolumeResult{Err: syscall.ESTALE}
}

func applyProcessVolumeHandleOperation(resource processVolumeResource, request processVolumeCommand) processVolumeResult {
	handle := resource.handle
	switch request.Operation {
	case processVolumeHandleReadOp, processVolumeHandleReadAtOp:
		buffer := make([]byte, min(request.ReadLength, uint64(maximumProcessVolumeDataBytes)))
		var count int
		var err error
		if request.Operation == processVolumeHandleReadOp {
			count, err = handle.Read(buffer)
		} else {
			count, err = handle.ReadAt(buffer, request.Offset)
		}
		return processVolumeResult{BytesTransferred: uint64(count), Data: buffer[:count], Err: err}
	case processVolumeHandleWriteOp, processVolumeHandleWriteAtOp:
		var count int
		var err error
		if request.Operation == processVolumeHandleWriteOp {
			count, err = handle.Write(request.Data)
		} else {
			count, err = handle.WriteAt(request.Data, request.Offset)
		}
		return processVolumeResult{BytesTransferred: uint64(count), Err: err}
	case processVolumeHandleTruncateOp:
		return processVolumeResult{Err: handle.Truncate(request.Size)}
	case processVolumeHandleChmodOp:
		return processVolumeResult{Err: handle.Chmod(request.Mode)}
	case processVolumeHandleChtimesOp:
		return processVolumeResult{Err: handle.Chtimes(request.ModTime)}
	case processVolumeHandleChdirOp:
		return processVolumeResult{Err: handle.Chdir()}
	case processVolumeHandleSeekOp:
		offset, err := handle.Seek(request.Offset, int(request.Whence))
		return processVolumeResult{Offset: offset, Err: err}
	case processVolumeHandleStatOp:
		entry, err := handle.Stat()
		return processVolumeEntryResponse(entry, err)
	case processVolumeHandleReadDirOp:
		entries, err := handle.ReadDir(int(request.DirectoryCount))
		return processVolumeResult{Entries: entries, Err: err}
	case processVolumeHandleCloseOp:
		err := handle.Close()
		if err == nil {
			removeProcessVolumeResource(request.Handle)
		}
		return processVolumeResult{Err: err}
	case processVolumeHandleSyncOp:
		return processVolumeResult{Err: handle.Sync()}
	case processVolumeHandleMapOp:
		mapping, err := handle.Map(request.Offset, request.MapLength, false)
		if err != nil {
			return processVolumeResult{Err: err}
		}
		handle, err := registerProcessVolumeResource(processVolumeResource{domain: resource.domain, mapping: mapping})
		if err != nil {
			return processVolumeResult{Err: errors.Join(err, mapping.Close())}
		}
		return processVolumeResult{Handle: handle}
	default:
		return processVolumeResult{Err: syscall.ENOTSUP}
	}
}

func applyProcessVolumeMappingOperation(resource processVolumeResource, request processVolumeCommand) processVolumeResult {
	switch request.Operation {
	case processVolumeMappingBytesOp:
		contents, err := resource.mapping.Bytes()
		return processVolumeResult{Data: append([]byte(nil), contents...), Err: err}
	case processVolumeMappingCloseOp:
		err := resource.mapping.Close()
		if err == nil {
			removeProcessVolumeResource(request.Handle)
		}
		return processVolumeResult{Err: err}
	default:
		return processVolumeResult{Err: syscall.ENOTSUP}
	}
}

func processVolumeEntryResponse(entry Entry, err error) processVolumeResult {
	response := processVolumeResult{Err: err}
	if err == nil {
		response.Entries = []Entry{entry}
	}
	return response
}

func registerProcessVolumeResource(resource processVolumeResource) (uint64, error) {
	processVolumeResources.Lock()
	defer processVolumeResources.Unlock()
	if len(processVolumeResources.values) >= maximumHandles {
		return 0, syscall.EMFILE
	}
	processVolumeResources.next++
	if processVolumeResources.next == 0 {
		return 0, syscall.EMFILE
	}
	processVolumeResources.values[processVolumeResources.next] = resource
	return processVolumeResources.next, nil
}

func processVolumeResourceFor(domain, handle uint64) (processVolumeResource, bool) {
	processVolumeResources.Lock()
	defer processVolumeResources.Unlock()
	resource, ok := processVolumeResources.values[handle]
	if !ok || resource.domain != domain {
		return processVolumeResource{}, false
	}
	return resource, true
}

func removeProcessVolumeResource(handle uint64) {
	processVolumeResources.Lock()
	delete(processVolumeResources.values, handle)
	processVolumeResources.Unlock()
}

func revokeProcessVolumeResources(domain uint64) {
	processVolumeResources.Lock()
	for handle, resource := range processVolumeResources.values {
		if resource.domain == domain {
			delete(processVolumeResources.values, handle)
		}
	}
	processVolumeResources.Unlock()
}
