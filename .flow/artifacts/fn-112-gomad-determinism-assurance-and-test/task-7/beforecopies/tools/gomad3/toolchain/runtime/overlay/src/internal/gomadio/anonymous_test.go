// Copyright 2026 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gomadio

import (
	"testing"
	"unsafe"
)

func TestAnonymousMapIsAlignedAndZeroInitialized(t *testing.T) {
	const (
		size      = uintptr(8192)
		alignment = uintptr(65536)
	)
	address := AnonymousMap(size, alignment)
	if address == 0 {
		t.Fatal("AnonymousMap() = 0")
	}
	if address&(alignment-1) != 0 {
		t.Fatalf("AnonymousMap() address %#x is not aligned to %#x", address, alignment)
	}
	for index, value := range unsafe.Slice((*byte)(unsafe.Pointer(address)), int(size)) {
		if value != 0 {
			t.Fatalf("AnonymousMap() byte %d = %d, want 0", index, value)
		}
	}
	if !AnonymousUnmap(address, size) {
		t.Fatal("AnonymousUnmap() = false")
	}
}

func TestAnonymousUnmapRequiresExactMapping(t *testing.T) {
	const size = uintptr(4096)
	address := AnonymousMap(size, 4096)
	if address == 0 {
		t.Fatal("AnonymousMap() = 0")
	}
	if AnonymousUnmap(address+1, size) {
		t.Fatal("AnonymousUnmap() accepted changed address")
	}
	if AnonymousUnmap(address, size+1) {
		t.Fatal("AnonymousUnmap() accepted changed size")
	}
	if !AnonymousUnmap(address, size) {
		t.Fatal("AnonymousUnmap() = false")
	}
	if AnonymousUnmap(address, size) {
		t.Fatal("AnonymousUnmap() accepted an already released mapping")
	}
}

func TestAnonymousMapRejectsInvalidRequests(t *testing.T) {
	for _, request := range []struct {
		size      uintptr
		alignment uintptr
	}{
		{size: 0, alignment: 4096},
		{size: 4096, alignment: 0},
		{size: 4096, alignment: 3},
		{size: 4096, alignment: anonymousMaximumAlignment << 1},
		{size: anonymousMaximumBytes + 1, alignment: 4096},
	} {
		if address := AnonymousMap(request.size, request.alignment); address != 0 {
			t.Fatalf("AnonymousMap(%d, %d) = %#x, want 0", request.size, request.alignment, address)
		}
	}
}

func TestAnonymousAllocatorEnforcesCumulativeBound(t *testing.T) {
	allocator := newAnonymousAllocator(3 * 4096)
	first := allocator.allocate(4096, 4096)
	if first == 0 {
		t.Fatal("first allocate() = 0")
	}
	if second := allocator.allocate(4096, 4096); second != 0 {
		t.Fatalf("second allocate() = %#x, want 0", second)
	}
	if !allocator.release(first, 4096) {
		t.Fatal("release() = false")
	}
	if second := allocator.allocate(4096, 4096); second == 0 {
		t.Fatal("allocate() after release = 0")
	} else if !allocator.release(second, 4096) {
		t.Fatal("second release() = false")
	}
}
