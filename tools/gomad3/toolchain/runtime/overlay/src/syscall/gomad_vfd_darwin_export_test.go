// Copyright 2026 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build darwin && arm64

package syscall

import (
	"internal/abi"
	"unsafe"
)

func GomadTestDarwinData(fd int, p unsafe.Pointer, count uintptr, write bool, variant int) (uintptr, Errno) {
	fn := abi.FuncPCABI0(libc_read_trampoline)
	if write {
		fn = abi.FuncPCABI0(libc_write_trampoline)
	}
	var n uintptr
	var err Errno
	switch variant {
	case 0:
		n, _, err = syscall(fn, uintptr(fd), uintptr(p), count)
	case 1:
		n, _, err = syscall6(fn, uintptr(fd), uintptr(p), count, 0, 0, 0)
	case 2:
		n, _, err = rawSyscall(fn, uintptr(fd), uintptr(p), count)
	case 3:
		n, _, err = rawSyscall6(fn, uintptr(fd), uintptr(p), count, 0, 0, 0)
	}
	return n, err
}

func GomadTestDarwinWritev(fd int, vectors *Iovec, count uintptr) (uintptr, Errno) {
	n, _, err := syscall(abi.FuncPCABI0(libc_writev_trampoline), uintptr(fd), uintptr(unsafe.Pointer(vectors)), count)
	return n, err
}

func GomadTestDarwinReadvKnownShape() bool {
	pc := abi.FuncPCABI0(libc_readv_trampoline)
	target := gomadBranchTarget(pc)
	return target != 0 && gomadDarwinOperation(pc) == gomadOpRefused && gomadDarwinOperation(target) == gomadOpRefused
}
