// Copyright 2026 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build (darwin && arm64) || (linux && amd64)

package syscall_test

import (
	"internal/gomadvfd"
	"syscall"
	"testing"
	"unsafe"
)

type gomadStackBackend struct {
	before, after uintptr
	bytes         []byte
	refusals      []string
	limit         int
}

func (*gomadStackBackend) Socket() (any, gomadvfd.Status)                { return 1, gomadvfd.OK }
func (*gomadStackBackend) Attach(any, uintptr, uint64)                   {}
func (*gomadStackBackend) Bind(any, gomadvfd.Address) gomadvfd.Status    { return gomadvfd.OK }
func (*gomadStackBackend) Listen(any, int) gomadvfd.Status               { return gomadvfd.OK }
func (*gomadStackBackend) Connect(any, gomadvfd.Address) gomadvfd.Status { return gomadvfd.OK }
func (*gomadStackBackend) Accept(any) (any, gomadvfd.Address, gomadvfd.Status) {
	return 1, gomadvfd.Address{}, gomadvfd.OK
}
func (*gomadStackBackend) Shutdown(any, int) gomadvfd.Status { return gomadvfd.OK }
func (*gomadStackBackend) Close(any) gomadvfd.Status         { return gomadvfd.OK }
func (*gomadStackBackend) Local(any) (gomadvfd.Address, gomadvfd.Status) {
	return gomadvfd.Address{}, gomadvfd.OK
}
func (*gomadStackBackend) Remote(any) (gomadvfd.Address, gomadvfd.Status) {
	return gomadvfd.Address{}, gomadvfd.OK
}
func (backend *gomadStackBackend) Refuse(operation string) {
	backend.refusals = append(backend.refusals, operation)
}
func (backend *gomadStackBackend) Read(_ any, p []byte) (int, gomadvfd.Status) {
	backend.before = uintptr(unsafe.Pointer(unsafe.SliceData(p)))
	backend.after = gomadGrowStack(64, p)
	return copy(p, "stack-safe"), gomadvfd.OK
}
func (backend *gomadStackBackend) Write(_ any, p []byte) (int, gomadvfd.Status) {
	backend.before = uintptr(unsafe.Pointer(unsafe.SliceData(p)))
	backend.after = gomadGrowStack(64, p)
	n := len(p)
	if backend.limit > 0 {
		n = min(n, backend.limit)
	}
	backend.bytes = append(backend.bytes[:0], p[:n]...)
	return n, gomadvfd.OK
}

func TestGomadWritevBoundsAndPartialProgress(t *testing.T) {
	backend := &gomadStackBackend{limit: 6}
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
	fd, status := gomadvfd.Socket()
	if status != gomadvfd.OK {
		t.Fatal(status)
	}
	t.Cleanup(func() {
		if status := gomadvfd.Close(fd); status != gomadvfd.OK {
			t.Errorf("close descriptor: %v", status)
		}
	})
	a, b := [4]byte{'a', 'b', 'c', 'd'}, [4]byte{'e', 'f', 'g', 'h'}
	vectors := [2]syscall.Iovec{{Base: &a[0], Len: 4}, {Base: &b[0], Len: 4}}
	if n, err := gomadTestWritev(fd, &vectors[0], 2); n != 6 || err != 0 || string(backend.bytes) != "abcdef" {
		t.Fatalf("partial vectors: n=%d errno=%v bytes=%q", n, err, backend.bytes)
	}
	for _, test := range []struct {
		vectors [2]syscall.Iovec
		want    syscall.Errno
	}{
		{vectors: [2]syscall.Iovec{{Len: 1}}, want: syscall.EFAULT},
		{vectors: [2]syscall.Iovec{{Base: &a[0], Len: uint64(^uint(0) >> 1)}, {Base: &b[0], Len: 1}}, want: syscall.EINVAL},
	} {
		if _, err := gomadTestWritev(fd, &test.vectors[0], 2); err != test.want {
			t.Fatalf("errno=%v want=%v", err, test.want)
		}
	}
	backend.limit = 0
	large := make([]byte, (64<<10)+1)
	largeVectors := [1]syscall.Iovec{{Base: &large[0], Len: uint64(len(large))}}
	if n, err := gomadTestWritev(fd, &largeVectors[0], 1); n != 64<<10 || err != 0 || len(backend.bytes) != 64<<10 {
		t.Fatalf("bounded vector copy: n=%d errno=%v copied=%d", n, err, len(backend.bytes))
	}
}

//go:noinline
func gomadGrowStack(depth int, p []byte) uintptr {
	var frame [128]uint64
	for i := range frame {
		frame[i] = uint64(depth + i)
	}
	var address uintptr
	if depth == 0 {
		address = uintptr(unsafe.Pointer(unsafe.SliceData(p)))
	} else {
		address = gomadGrowStack(depth-1, p)
	}
	if frame[depth] != uint64(depth*2) {
		panic("stack frame corrupted")
	}
	return address
}

func TestGomadGenericMovingStack(t *testing.T) {
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
	fd, status := gomadvfd.Socket()
	if status != gomadvfd.OK {
		t.Fatal(status)
	}
	t.Cleanup(func() {
		if status := gomadvfd.Close(fd); status != gomadvfd.OK {
			t.Errorf("close descriptor: %v", status)
		}
	})
	for _, write := range []bool{false, true} {
		for variant := range 4 {
			result := make(chan string, 1)
			go func() {
				var data [64]byte
				data[0], data[63] = 0x5a, 0xa5
				copy(data[1:], "stack-safe")
				n, errno := gomadTestGenericData(fd, unsafe.Pointer(&data[1]), 10, write, variant)
				if errno != 0 || n != 10 {
					result <- "generic transfer failed"
					return
				}
				if backend.before == backend.after {
					result <- "caller buffer did not move; native stack proof inconclusive"
					return
				}
				if string(data[1:11]) != "stack-safe" || data[0] != 0x5a || data[63] != 0xa5 {
					result <- "caller bytes/sentinels corrupted"
					return
				}
				if write && string(backend.bytes) != "stack-safe" {
					result <- "backend bytes corrupted"
					return
				}
				result <- ""
			}()
			if message := <-result; message != "" {
				t.Fatalf("write=%v variant=%d: %s", write, variant, message)
			}
		}
	}
	if _, err := syscall.GomadVirtualSocket(syscall.AF_INET, syscall.SOCK_DGRAM, 0); err != syscall.EOPNOTSUPP || len(backend.refusals) == 0 {
		t.Fatal("UDP was not refused and reported")
	}
}
