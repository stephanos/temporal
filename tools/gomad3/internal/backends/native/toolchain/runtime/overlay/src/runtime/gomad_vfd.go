// Copyright 2026 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build unix || (js && wasm) || wasip1 || windows

package runtime

import (
	"internal/runtime/atomic"
	"unsafe"
)

const (
	gomadVirtualFDBase  = 1 << 20
	gomadVirtualFDCount = 1 << 16
)

type gomadVirtualRegistration struct {
	pd       *pollDesc
	token    uint64
	sequence uintptr
}

var gomadVirtualPoll [gomadVirtualFDCount]gomadVirtualRegistration
var gomadVirtualToken func(uintptr) uint64
var gomadVirtualPending func(uintptr, uint64) int32

//go:linkname gomadVirtualInstall
func gomadVirtualInstall(token func(uintptr) uint64, pending func(uintptr, uint64) int32) {
	if gomadVirtualToken != nil {
		throw("runtime: duplicate Gomad descriptor backend")
	}
	gomadVirtualToken = token
	gomadVirtualPending = pending
}

func gomadVirtualOwner(fd uintptr) uint64 {
	if gomadVirtualToken == nil || fd < gomadVirtualFDBase || fd >= gomadVirtualFDBase+gomadVirtualFDCount {
		return 0
	}
	return gomadVirtualToken(fd)
}

func gomadVirtualPollOpen(pd *pollDesc) uintptr {
	lock(&pollcache.lock)
	slot := &gomadVirtualPoll[pd.fd-gomadVirtualFDBase]
	if slot.pd != nil {
		unlock(&pollcache.lock)
		return 9
	}
	*slot = gomadVirtualRegistration{pd, pd.gomadVirtual, pd.fdseq.Load()}
	unlock(&pollcache.lock)
	if gomadVirtualPending != nil {
		gomadVirtualReady(pd.fd, pd.gomadVirtual, gomadVirtualPending(pd.fd, pd.gomadVirtual))
	}
	return 0
}

func gomadVirtualPollClose(pd *pollDesc) {
	lock(&pollcache.lock)
	slot := &gomadVirtualPoll[pd.fd-gomadVirtualFDBase]
	if slot.pd != pd || slot.token != pd.gomadVirtual || slot.sequence != pd.fdseq.Load() {
		throw("runtime: invalid Gomad descriptor registration")
	}
	*slot = gomadVirtualRegistration{}
	unlock(&pollcache.lock)
}

//go:linkname gomadVirtualReady
func gomadVirtualReady(fd uintptr, token uint64, mode int32) {
	if fd < gomadVirtualFDBase || fd >= gomadVirtualFDBase+gomadVirtualFDCount || token == 0 {
		return
	}
	lock(&pollcache.lock)
	slot := &gomadVirtualPoll[fd-gomadVirtualFDBase]
	pd := slot.pd
	if pd == nil || slot.token != token || slot.sequence != pd.fdseq.Load() || pd.info().closing() {
		unlock(&pollcache.lock)
		return
	}
	var reader, writer *g
	delta := int32(0)
	if mode == 'r' || mode == 'r'+'w' {
		reader = netpollunblock(pd, 'r', true, &delta)
	}
	if mode == 'w' || mode == 'r'+'w' {
		writer = netpollunblock(pd, 'w', true, &delta)
	}
	if delta != 0 {
		throw("runtime: Gomad descriptor changed host poll waiters")
	}
	unlock(&pollcache.lock)
	if reader != nil {
		netpollgoready(reader, 0)
	}
	if writer != nil {
		netpollgoready(writer, 0)
	}
}

func gomadVirtualPollBlockCommit(gp *g, semaphore unsafe.Pointer) bool {
	return atomic.Casuintptr((*uintptr)(semaphore), pdWait, uintptr(unsafe.Pointer(gp)))
}
