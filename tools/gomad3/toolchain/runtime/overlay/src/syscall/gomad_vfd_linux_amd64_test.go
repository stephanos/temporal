// Copyright 2026 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package syscall_test

import (
	"internal/gomadvfd"
	"syscall"
	"testing"
	"unsafe"
)

func gomadTestGenericData(fd int, p unsafe.Pointer, count uintptr, write bool, variant int) (uintptr, syscall.Errno) {
	op := uintptr(syscall.SYS_READ)
	if write {
		op = syscall.SYS_WRITE
	}
	var n uintptr
	var err syscall.Errno
	switch variant {
	case 0:
		n, _, err = syscall.Syscall(op, uintptr(fd), uintptr(p), count)
	case 1:
		n, _, err = syscall.Syscall6(op, uintptr(fd), uintptr(p), count, 0, 0, 0)
	case 2:
		n, _, err = syscall.RawSyscall(op, uintptr(fd), uintptr(p), count)
	case 3:
		n, _, err = syscall.RawSyscall6(op, uintptr(fd), uintptr(p), count, 0, 0, 0)
	}
	return n, err
}

func gomadTestWritev(fd int, vectors *syscall.Iovec, count uintptr) (uintptr, syscall.Errno) {
	n, _, err := syscall.Syscall(syscall.SYS_WRITEV, uintptr(fd), uintptr(unsafe.Pointer(vectors)), count)
	return n, err
}

func TestGomadGenericOperationClassification(t *testing.T) {
	backend := &gomadStackBackend{}
	if status := gomadvfd.RegisterBackend(backend); status != gomadvfd.OK {
		t.Fatal(status)
	}
	if status := gomadvfd.SetEnabled(true, nil); status != gomadvfd.OK {
		t.Fatal(status)
	}
	t.Cleanup(func() {
		if status := gomadvfd.SetEnabled(false, nil); status != gomadvfd.OK {
			t.Errorf("disable boundary: %v", status)
		}
	})
	if _, _, _, handled := syscall.GomadGeneric(syscall.SYS_GETPID, 1<<20, 0, 0, 0, 0, 0); handled {
		t.Fatal("non-descriptor argument interpreted as a descriptor")
	}
	fd, status := gomadvfd.Socket()
	if status != gomadvfd.OK {
		t.Fatal(status)
	}
	t.Cleanup(func() {
		if gomadvfd.Owns(fd) {
			if status := gomadvfd.Close(fd); status != gomadvfd.OK {
				t.Errorf("close descriptor: %v", status)
			}
		}
	})
	for _, trap := range []uintptr{syscall.SYS_RECVMMSG, 307} {
		if _, _, errno, handled := syscall.GomadGeneric(trap, uintptr(fd), 0, 0, 0, 0, 0); !handled || errno != syscall.EOPNOTSUPP {
			t.Fatalf("unmodeled message operation: trap=%d handled=%v errno=%v", trap, handled, errno)
		}
	}
	if len(backend.refusals) != 2 {
		t.Fatalf("unmodeled message operation refusals=%v", backend.refusals)
	}
	if status := gomadvfd.Close(fd); status != gomadvfd.OK {
		t.Fatal(status)
	}
	for _, descriptor := range []uintptr{gomadvfd.FirstFD + gomadvfd.MaxDescriptors - 1, uintptr(fd)} {
		for _, trap := range []uintptr{syscall.SYS_READ, syscall.SYS_CLOSE, syscall.SYS_FCNTL} {
			if r1, _, errno, handled := syscall.GomadGeneric(trap, descriptor, ^uintptr(0), 1, 0, 0, 0); !handled || errno != syscall.EBADF || r1 != ^uintptr(0) {
				t.Fatalf("unowned virtual descriptor: fd=%d trap=%d handled=%v errno=%v r1=%d", descriptor, trap, handled, errno, r1)
			}
		}
	}
	if _, _, _, handled := syscall.GomadGeneric(syscall.SYS_READ, 4, 0, 0, 0, 0, 0); handled {
		t.Fatal("reserved descriptor routed")
	}
	if _, _, _, handled := syscall.GomadGeneric(syscall.SYS_READ, gomadvfd.FirstFD+gomadvfd.MaxDescriptors, 0, 0, 0, 0, 0); handled {
		t.Fatal("out-of-range descriptor routed")
	}
	if status := gomadvfd.SetEnabled(false, nil); status != gomadvfd.OK {
		t.Fatal(status)
	}
	if _, _, _, handled := syscall.GomadGeneric(syscall.SYS_READ, uintptr(fd), 0, 1, 0, 0, 0); handled {
		t.Fatal("disabled boundary routed descriptor")
	}
}
