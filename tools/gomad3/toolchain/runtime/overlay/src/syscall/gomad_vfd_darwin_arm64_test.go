// Copyright 2026 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package syscall_test

import (
	"syscall"
	"testing"
	"unsafe"
)

func gomadTestGenericData(fd int, p unsafe.Pointer, count uintptr, write bool, variant int) (uintptr, syscall.Errno) {
	return syscall.GomadTestDarwinData(fd, p, count, write, variant)
}

func gomadTestWritev(fd int, vectors *syscall.Iovec, count uintptr) (uintptr, syscall.Errno) {
	return syscall.GomadTestDarwinWritev(fd, vectors, count)
}

func TestGomadDarwinReadvKnownShape(t *testing.T) {
	if !syscall.GomadTestDarwinReadvKnownShape() {
		t.Fatal("reviewed readv trampoline and target not classified as refused")
	}
}
