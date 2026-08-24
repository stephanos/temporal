// Copyright 2026 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gomadfs

import (
	"errors"
	"io"
	"sync"
	"syscall"
	_ "unsafe"

	"internal/gomadmodelwire"
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
	request, err := gomadmodelwire.DecodeRequest(encoded)
	if err != nil || request.Model != gomadmodelwire.ModelVolume {
		return nil, false
	}
	if _, ok := gomadsim.DescribeNetworkDomain(domainToken); !ok {
		return encodeProcessVolumeResponse(gomadmodelwire.Response{Error: encodeProcessVolumeError(syscall.ESTALE)})
	}
	return encodeProcessVolumeResponse(applyProcessVolumeOperation(domainToken, Current(), request))
}

func applyProcessVolumeOperation(domain uint64, filesystem *FS, request gomadmodelwire.Request) gomadmodelwire.Response {
	switch request.Operation {
	case gomadmodelwire.VolumeResolve:
		path, base, err := filesystem.Resolve(request.String1)
		return gomadmodelwire.Response{String1: path, String2: base, Error: encodeProcessVolumeError(err)}
	case gomadmodelwire.VolumeMkdir, gomadmodelwire.VolumeMkdirAll:
		var err error
		if request.Operation == gomadmodelwire.VolumeMkdir {
			err = filesystem.Mkdir(request.String1, uint32(request.Uint1))
		} else {
			err = filesystem.MkdirAll(request.String1, uint32(request.Uint1))
		}
		return gomadmodelwire.Response{Error: encodeProcessVolumeError(err)}
	case gomadmodelwire.VolumeStat:
		entry, err := filesystem.Stat(request.String1)
		return processVolumeEntryResponse(entry, err)
	case gomadmodelwire.VolumeOpen:
		handle, err := filesystem.Open(request.String1, OpenFlags{
			Read: request.Flags&processOpenRead != 0, Write: request.Flags&processOpenWrite != 0,
			Append: request.Flags&processOpenAppend != 0, Create: request.Flags&processOpenCreate != 0,
			Exclusive: request.Flags&processOpenExclusive != 0, Truncate: request.Flags&processOpenTruncate != 0,
		}, uint32(request.Uint1))
		if err != nil {
			return gomadmodelwire.Response{Error: encodeProcessVolumeError(err)}
		}
		resource, err := registerProcessVolumeResource(processVolumeResource{domain: domain, handle: handle})
		if err != nil {
			return gomadmodelwire.Response{Error: encodeProcessVolumeError(errors.Join(err, handle.Close()))}
		}
		return gomadmodelwire.Response{Handle: resource, String1: handle.Path()}
	case gomadmodelwire.VolumeRename:
		return gomadmodelwire.Response{Error: encodeProcessVolumeError(filesystem.Rename(request.String1, request.String2))}
	case gomadmodelwire.VolumeRemove:
		return gomadmodelwire.Response{Error: encodeProcessVolumeError(filesystem.Remove(request.String1))}
	case gomadmodelwire.VolumeRemoveAll:
		return gomadmodelwire.Response{Error: encodeProcessVolumeError(filesystem.RemoveAll(request.String1))}
	case gomadmodelwire.VolumeChmod:
		return gomadmodelwire.Response{Error: encodeProcessVolumeError(filesystem.Chmod(request.String1, uint32(request.Uint1)))}
	case gomadmodelwire.VolumeChtimes:
		return gomadmodelwire.Response{Error: encodeProcessVolumeError(filesystem.Chtimes(request.String1, request.Int1))}
	case gomadmodelwire.VolumeChdir:
		return gomadmodelwire.Response{Error: encodeProcessVolumeError(filesystem.Chdir(request.String1))}
	case gomadmodelwire.VolumeGetwd:
		return gomadmodelwire.Response{String1: filesystem.Getwd()}
	}
	return applyProcessVolumeResourceOperation(domain, request)
}

func applyProcessVolumeResourceOperation(domain uint64, request gomadmodelwire.Request) gomadmodelwire.Response {
	resource, ok := processVolumeResourceFor(domain, request.Handle)
	if !ok {
		return gomadmodelwire.Response{Error: encodeProcessVolumeError(syscall.ESTALE)}
	}
	if resource.handle != nil {
		return applyProcessVolumeHandleOperation(resource, request)
	}
	if resource.mapping != nil {
		return applyProcessVolumeMappingOperation(resource, request)
	}
	return gomadmodelwire.Response{Error: encodeProcessVolumeError(syscall.ESTALE)}
}

func applyProcessVolumeHandleOperation(resource processVolumeResource, request gomadmodelwire.Request) gomadmodelwire.Response {
	handle := resource.handle
	switch request.Operation {
	case gomadmodelwire.VolumeHandleRead, gomadmodelwire.VolumeHandleReadAt:
		buffer := make([]byte, min(request.Uint1, uint64(gomadmodelwire.MaximumDataBytes)))
		var count int
		var err error
		if request.Operation == gomadmodelwire.VolumeHandleRead {
			count, err = handle.Read(buffer)
		} else {
			count, err = handle.ReadAt(buffer, request.Int1)
		}
		return gomadmodelwire.Response{Uint1: uint64(count), Data: buffer[:count], Error: encodeProcessVolumeError(err)}
	case gomadmodelwire.VolumeHandleWrite, gomadmodelwire.VolumeHandleWriteAt:
		var count int
		var err error
		if request.Operation == gomadmodelwire.VolumeHandleWrite {
			count, err = handle.Write(request.Data)
		} else {
			count, err = handle.WriteAt(request.Data, request.Int1)
		}
		return gomadmodelwire.Response{Uint1: uint64(count), Error: encodeProcessVolumeError(err)}
	case gomadmodelwire.VolumeHandleTruncate:
		return gomadmodelwire.Response{Error: encodeProcessVolumeError(handle.Truncate(request.Int1))}
	case gomadmodelwire.VolumeHandleChmod:
		return gomadmodelwire.Response{Error: encodeProcessVolumeError(handle.Chmod(uint32(request.Uint1)))}
	case gomadmodelwire.VolumeHandleChtimes:
		return gomadmodelwire.Response{Error: encodeProcessVolumeError(handle.Chtimes(request.Int1))}
	case gomadmodelwire.VolumeHandleChdir:
		return gomadmodelwire.Response{Error: encodeProcessVolumeError(handle.Chdir())}
	case gomadmodelwire.VolumeHandleSeek:
		offset, err := handle.Seek(request.Int1, int(request.Int2))
		return gomadmodelwire.Response{Int1: offset, Error: encodeProcessVolumeError(err)}
	case gomadmodelwire.VolumeHandleStat:
		entry, err := handle.Stat()
		return processVolumeEntryResponse(entry, err)
	case gomadmodelwire.VolumeHandleReadDir:
		entries, err := handle.ReadDir(int(request.Int1))
		response := gomadmodelwire.Response{Entries: make([]gomadmodelwire.Entry, len(entries)), Error: encodeProcessVolumeError(err)}
		for index := range entries {
			response.Entries[index] = processWireEntry(entries[index])
		}
		return response
	case gomadmodelwire.VolumeHandleClose:
		err := handle.Close()
		if err == nil {
			removeProcessVolumeResource(request.Handle)
		}
		return gomadmodelwire.Response{Error: encodeProcessVolumeError(err)}
	case gomadmodelwire.VolumeHandleSync:
		return gomadmodelwire.Response{Error: encodeProcessVolumeError(handle.Sync())}
	case gomadmodelwire.VolumeHandleMap:
		mapping, err := handle.Map(request.Uint1)
		if err != nil {
			return gomadmodelwire.Response{Error: encodeProcessVolumeError(err)}
		}
		handle, err := registerProcessVolumeResource(processVolumeResource{domain: resource.domain, mapping: mapping})
		if err != nil {
			return gomadmodelwire.Response{Error: encodeProcessVolumeError(errors.Join(err, mapping.Close()))}
		}
		return gomadmodelwire.Response{Handle: handle}
	default:
		return gomadmodelwire.Response{Error: encodeProcessVolumeError(syscall.ENOTSUP)}
	}
}

func applyProcessVolumeMappingOperation(resource processVolumeResource, request gomadmodelwire.Request) gomadmodelwire.Response {
	switch request.Operation {
	case gomadmodelwire.VolumeMappingBytes:
		contents, err := resource.mapping.Bytes()
		return gomadmodelwire.Response{Data: append([]byte(nil), contents...), Error: encodeProcessVolumeError(err)}
	case gomadmodelwire.VolumeMappingClose:
		err := resource.mapping.Close()
		if err == nil {
			removeProcessVolumeResource(request.Handle)
		}
		return gomadmodelwire.Response{Error: encodeProcessVolumeError(err)}
	default:
		return gomadmodelwire.Response{Error: encodeProcessVolumeError(syscall.ENOTSUP)}
	}
}

func processVolumeEntryResponse(entry Entry, err error) gomadmodelwire.Response {
	response := gomadmodelwire.Response{Error: encodeProcessVolumeError(err)}
	if err == nil {
		response.Entries = []gomadmodelwire.Entry{processWireEntry(entry)}
	}
	return response
}

func processWireEntry(entry Entry) gomadmodelwire.Entry {
	return gomadmodelwire.Entry{Name: entry.Name, Mode: entry.Mode, Kind: uint8(entry.Kind), ModTime: entry.ModTime, Data: append([]byte(nil), entry.Data...)}
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

func encodeProcessVolumeResponse(response gomadmodelwire.Response) ([]byte, bool) {
	encoded, err := gomadmodelwire.EncodeResponse(response)
	return encoded, err == nil
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
