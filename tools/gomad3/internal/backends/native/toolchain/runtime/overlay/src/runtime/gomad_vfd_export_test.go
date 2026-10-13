// Copyright 2026 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package runtime

import "unsafe"

func GomadVirtualSemaphoreCase(mode int32, state int, virtual, ready bool) (uintptr, int32, bool, bool) {
	netpollGenericInit()
	pd := pollcache.alloc()
	pd.gomadVirtual = 0
	if virtual {
		pd.gomadVirtual = 1
	}
	semaphore := &pd.rg
	if mode == 'w' {
		semaphore = &pd.wg
	}
	before := netpollWaiters.Load()
	semaphore.Store(uintptr(state))
	if state == 3 {
		semaphore.Store(pdWait)
		if virtual {
			gomadVirtualPollBlockCommit(getg(), unsafe.Pointer(semaphore))
		} else {
			netpollblockcommit(getg(), unsafe.Pointer(semaphore))
		}
	}
	delta := int32(0)
	gp := netpollunblock(pd, mode, ready, &delta)
	after := semaphore.Load()
	netpollAdjustWaiters(delta)
	balanced := before == netpollWaiters.Load()
	pd.rg.Store(pdNil)
	pd.wg.Store(pdNil)
	pollcache.free(pd)
	return after, delta, gp != nil, balanced
}

func GomadVirtualRegistryCase() bool {
	netpollGenericInit()
	pd := pollcache.alloc()
	pd.fd = gomadVirtualFDBase
	pd.gomadVirtual = 7
	pd.fdseq.Store(11)
	lock(&pd.lock)
	pd.closing = false
	pd.rd, pd.wd = 0, 0
	pd.publishInfo()
	unlock(&pd.lock)
	pd.rg.Store(pdWait)
	pd.wg.Store(pdWait)
	previous := gomadVirtualPending
	gomadVirtualPending = func(fd uintptr, token uint64) int32 {
		if fd == gomadVirtualFDBase && token == 7 {
			return 'w'
		}
		return 0
	}
	defer func() { gomadVirtualPending = previous }()
	if gomadVirtualPollOpen(pd) != 0 {
		return false
	}
	ok := pd.rg.Load() == pdWait && pd.wg.Load() == pdReady
	pd.wg.Store(pdWait)
	gomadVirtualReady(pd.fd, 6, 'r'+'w')
	ok = ok && pd.rg.Load() == pdWait && pd.wg.Load() == pdWait
	gomadVirtualReady(pd.fd, 7, 'r'+'w')
	ok = ok && pd.rg.Load() == pdReady && pd.wg.Load() == pdReady
	gomadVirtualReady(pd.fd, 7, 'r'+'w')
	ok = ok && pd.rg.Load() == pdReady && pd.wg.Load() == pdReady
	lock(&pd.lock)
	pd.closing = true
	pd.publishInfo()
	unlock(&pd.lock)
	pd.rg.Store(pdWait)
	gomadVirtualReady(pd.fd, 7, 'r')
	ok = ok && pd.rg.Load() == pdWait
	gomadVirtualPollClose(pd)
	pd.rg.Store(pdNil)
	pd.wg.Store(pdNil)
	pollcache.free(pd)
	gomadVirtualReady(gomadVirtualFDBase, 7, 'r'+'w')
	return ok
}
