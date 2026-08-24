// Copyright 2026 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package mount

import (
	"io"
	"sync"
	"syscall"
	"unsafe"

	"internal/gomadwire"
)

const (
	requestDescriptor  = 9
	responseDescriptor = 10
)

type Status = gomadwire.MountStatus
type Kind = gomadwire.MountKind
type Child = gomadwire.MountChild
type Entry = gomadwire.MountEntry

const (
	StatusOK        = gomadwire.MountStatusOK
	StatusUnmounted = gomadwire.MountStatusUnmounted
	StatusNotExist  = gomadwire.MountStatusNotExist
	KindFile        = gomadwire.MountKindFile
	KindDirectory   = gomadwire.MountKindDirectory
)

var Default = New(
	descriptorWriter(requestDescriptor),
	descriptorReader(responseDescriptor),
	gomadwire.MountLimits{PathBytes: 4096, FileBytes: 16 << 20, DirectoryEntries: 100_000},
)

//go:linkname runtimeBlockingRead runtime.gomadBlockingRead
func runtimeBlockingRead(int32, unsafe.Pointer, int32) int32

//go:linkname runtimeBlockingWrite runtime.gomadBlockingWrite
func runtimeBlockingWrite(uintptr, unsafe.Pointer, int32) int32

type Client struct {
	mu        sync.Mutex
	requests  io.Writer
	responses io.Reader
	limits    gomadwire.MountLimits
	ordinal   uint64
}

func New(requests io.Writer, responses io.Reader, limits gomadwire.MountLimits) *Client {
	return &Client{requests: requests, responses: responses, limits: limits}
}

func (client *Client) Lookup(path string) (Entry, Status, error) {
	client.mu.Lock()
	defer client.mu.Unlock()
	if uint64(len(path)) > client.limits.PathBytes {
		return gomadwire.MountEntry{}, 0, syscall.ENAMETOOLONG
	}
	ordinal := client.ordinal
	if err := gomadwire.WriteMountLookupRequest(client.requests, gomadwire.MountRequest{Ordinal: ordinal, Path: path}, client.limits); err != nil {
		return gomadwire.MountEntry{}, 0, err
	}
	response, err := gomadwire.ReadMountResponse(client.responses, client.limits)
	if err != nil {
		return gomadwire.MountEntry{}, 0, err
	}
	if response.Ordinal != ordinal {
		return gomadwire.MountEntry{}, 0, syscall.EPROTO
	}
	client.ordinal++
	return response.Entry, response.Status, nil
}

type descriptorWriter int

func (descriptor descriptorWriter) Write(data []byte) (int, error) {
	if len(data) == 0 {
		return 0, nil
	}
	if len(data) > 1<<31-1 {
		return 0, syscall.EOVERFLOW
	}
	written := runtimeBlockingWrite(uintptr(descriptor), unsafe.Pointer(&data[0]), int32(len(data)))
	if written < 0 || int(written) > len(data) {
		return 0, syscall.EIO
	}
	return int(written), nil
}

type descriptorReader int

func (descriptor descriptorReader) Read(data []byte) (int, error) {
	if len(data) == 0 {
		return 0, nil
	}
	if len(data) > 1<<31-1 {
		return 0, syscall.EOVERFLOW
	}
	read := runtimeBlockingRead(int32(descriptor), unsafe.Pointer(&data[0]), int32(len(data)))
	if read < 0 || int(read) > len(data) {
		return 0, syscall.EIO
	}
	if read == 0 {
		return 0, io.EOF
	}
	return int(read), nil
}
