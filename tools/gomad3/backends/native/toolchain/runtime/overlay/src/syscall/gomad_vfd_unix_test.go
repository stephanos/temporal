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

func TestGomadVirtualValidation(t *testing.T) {
	if status := gomadvfd.RegisterBackend(&gomadStackBackend{}); status != gomadvfd.OK {
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
	for _, test := range []struct {
		name string
		call func() error
		want error
	}{
		{"nil read", func() error { _, e := syscall.GomadGenericRead(1<<20, nil, 1); return e }, syscall.EFAULT},
		{"oversized read", func() error { _, e := syscall.GomadGenericRead(1<<20, nil, ^uintptr(0)); return e }, syscall.EINVAL},
		{"nil write", func() error { _, e := syscall.GomadGenericWrite(1<<20, nil, 1); return e }, syscall.EFAULT},
		{"oversized vectors", func() error { _, e := syscall.GomadGenericWritev(1<<20, nil, 1025); return e }, syscall.EINVAL},
		{"nil vectors", func() error { _, e := syscall.GomadGenericWritev(1<<20, nil, 1); return e }, syscall.EFAULT},
		{"short address", func() error { var sa syscall.RawSockaddrAny; return syscall.GomadVirtualBind(1<<20, unsafe.Pointer(&sa), 1) }, syscall.EINVAL},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := test.call(); got != test.want {
				t.Fatalf("got %v, want %v", got, test.want)
			}
		})
	}
	for _, socketType := range []int{syscall.SOCK_DGRAM, syscall.SOCK_RAW} {
		if fd, err := syscall.GomadVirtualSocket(syscall.AF_INET, socketType, 0); fd != -1 || err != syscall.EOPNOTSUPP {
			t.Fatalf("unmodeled socket: fd=%d err=%v", fd, err)
		}
	}
	if fd, err := syscall.GomadVirtualSocket(syscall.AF_INET6, syscall.SOCK_STREAM, 0); fd != -1 || err != syscall.EAFNOSUPPORT {
		t.Fatalf("IPv6 socket: fd=%d err=%v", fd, err)
	}
}
