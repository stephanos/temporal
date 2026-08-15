// Copyright 2026 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package mount

import (
	"io"
	"sync"
	"syscall"

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
	return syscall.Write(int(descriptor), data)
}

type descriptorReader int

func (descriptor descriptorReader) Read(data []byte) (int, error) {
	read, err := syscall.Read(int(descriptor), data)
	if read == 0 && err == nil && len(data) != 0 {
		return 0, io.EOF
	}
	return read, err
}
