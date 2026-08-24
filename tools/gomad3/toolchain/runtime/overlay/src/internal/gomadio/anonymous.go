// Copyright 2026 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gomadio

import (
	"sync"
	"unsafe"
)

const (
	anonymousMaximumBytes     = uintptr(256 << 20)
	anonymousMaximumAlignment = uintptr(1 << 20)
)

var anonymousMemory = newAnonymousAllocator(anonymousMaximumBytes)

type anonymousAllocator struct {
	sync.Mutex
	maximum  uintptr
	charged  uintptr
	mappings map[uintptr]anonymousMapping
}

type anonymousMapping struct {
	backing []byte
	size    uintptr
	charge  uintptr
}

func newAnonymousAllocator(maximum uintptr) *anonymousAllocator {
	return &anonymousAllocator{maximum: maximum, mappings: make(map[uintptr]anonymousMapping)}
}

//go:linkname AnonymousMap
func AnonymousMap(size, alignment uintptr) uintptr {
	return anonymousMemory.allocate(size, alignment)
}

func (allocator *anonymousAllocator) allocate(size, alignment uintptr) uintptr {
	if size == 0 || alignment == 0 || alignment > anonymousMaximumAlignment || alignment&(alignment-1) != 0 {
		return 0
	}
	padding := alignment - 1
	if size > ^uintptr(0)-padding {
		return 0
	}
	charge := size + padding
	if charge > uintptr(int(^uint(0)>>1)) {
		return 0
	}
	allocator.Lock()
	defer allocator.Unlock()
	if charge > allocator.maximum-allocator.charged {
		return 0
	}
	backing := make([]byte, int(charge))
	base := uintptr(unsafe.Pointer(unsafe.SliceData(backing)))
	address := (base + padding) &^ padding
	if _, found := allocator.mappings[address]; found {
		return 0
	}
	allocator.mappings[address] = anonymousMapping{backing: backing, size: size, charge: charge}
	allocator.charged += charge
	return address
}

//go:linkname AnonymousUnmap
func AnonymousUnmap(address, size uintptr) bool {
	return anonymousMemory.release(address, size)
}

func (allocator *anonymousAllocator) release(address, size uintptr) bool {
	allocator.Lock()
	defer allocator.Unlock()
	mapping, found := allocator.mappings[address]
	if !found || mapping.size != size {
		return false
	}
	delete(allocator.mappings, address)
	allocator.charged -= mapping.charge
	return true
}
