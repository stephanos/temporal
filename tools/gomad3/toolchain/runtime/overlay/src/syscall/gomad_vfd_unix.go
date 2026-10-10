// Copyright 2026 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build (darwin && arm64) || (linux && amd64)

package syscall

import (
	"internal/gomadvfd"
	"unsafe"
)

//go:linkname runtimeVirtualInstall runtime.gomadVirtualInstall
func runtimeVirtualInstall(func(uintptr) uint64, func(uintptr, uint64) int32)

//go:linkname runtimeVirtualReady runtime.gomadVirtualReady
func runtimeVirtualReady(uintptr, uint64, int32)

func init() {
	runtimeVirtualInstall(gomadvfd.Token, gomadvfd.TakeReady)
	gomadvfd.RegisterReady(runtimeVirtualReady)
}

//go:nosplit
//go:norace
func gomadVirtualFD(fd int) bool {
	return gomadvfd.Enabled() && gomadvfd.InRange(uintptr(fd))
}

//go:nosplit
//go:norace
func gomadVirtualEnabled() bool { return gomadvfd.Enabled() }

func gomadVirtualClose(fd int) error           { return gomadVirtualErr(gomadvfd.Close(fd)) }
func gomadVirtualListen(fd, backlog int) error { return gomadVirtualErr(gomadvfd.Listen(fd, backlog)) }
func gomadVirtualShutdown(fd, how int) error   { return gomadVirtualErr(gomadvfd.Shutdown(fd, how)) }
func gomadVirtualReadBytes(fd int, p []byte) (int, error) {
	n, status := gomadvfd.Read(fd, p)
	return n, gomadVirtualErr(status)
}
func gomadVirtualWriteBytes(fd int, p []byte) (int, error) {
	n, status := gomadvfd.Write(fd, p)
	return n, gomadVirtualErr(status)
}

func gomadVirtualErr(status gomadvfd.Status) error {
	switch status {
	case gomadvfd.OK, gomadvfd.EndOfStream:
		return nil
	case gomadvfd.WouldBlock:
		return EAGAIN
	case gomadvfd.Closed:
		return EBADF
	case gomadvfd.Refused:
		return ECONNREFUSED
	case gomadvfd.AddressInUse:
		return EADDRINUSE
	case gomadvfd.Capacity:
		return ENOBUFS
	case gomadvfd.NotConnected:
		return ENOTCONN
	case gomadvfd.BrokenPipe:
		return EPIPE
	case gomadvfd.Reset:
		return ECONNRESET
	case gomadvfd.Invalid:
		return EINVAL
	}
	return EOPNOTSUPP
}

func gomadVirtualRefuse(operation string) error {
	return gomadVirtualErr(gomadvfd.Refuse(operation))
}

func gomadVirtualCleanup(fd int, cause error) error {
	if err := gomadVirtualErr(gomadvfd.Close(fd)); err != nil {
		return err
	}
	return cause
}

func gomadVirtualSocket(domain, typ, proto int) (int, error) {
	if domain != AF_INET {
		gomadvfd.Refuse("socket.family")
		return -1, EAFNOSUPPORT
	}
	flags := typ & gomadSocketFlags
	if typ & ^gomadSocketFlags != SOCK_STREAM || proto != 0 && proto != IPPROTO_TCP {
		return -1, gomadVirtualRefuse("socket.type-or-protocol")
	}
	fd, status := gomadvfd.Socket()
	if status != gomadvfd.OK {
		return -1, gomadVirtualErr(status)
	}
	if err := gomadVirtualCreationFlags(fd, flags); err != nil {
		return -1, gomadVirtualCleanup(fd, err)
	}
	return fd, nil
}

func gomadVirtualAddress(p unsafe.Pointer, length _Socklen) (gomadvfd.Address, error) {
	if uintptr(length) != unsafe.Sizeof(RawSockaddrInet4{}) {
		return gomadvfd.Address{}, EINVAL
	}
	if p == nil {
		return gomadvfd.Address{}, EFAULT
	}
	raw := (*RawSockaddrInet4)(p)
	if raw.Family != AF_INET {
		gomadvfd.Refuse("sockaddr.family")
		return gomadvfd.Address{}, EAFNOSUPPORT
	}
	port := (*[2]byte)(unsafe.Pointer(&raw.Port))
	return gomadvfd.Address{IP: raw.Addr, Port: int(port[0])<<8 | int(port[1])}, nil
}

func gomadVirtualBind(fd int, p unsafe.Pointer, length _Socklen) error {
	address, err := gomadVirtualAddress(p, length)
	if err != nil {
		return err
	}
	return gomadVirtualErr(gomadvfd.Bind(fd, address))
}

func gomadVirtualConnect(fd int, p unsafe.Pointer, length _Socklen) error {
	address, err := gomadVirtualAddress(p, length)
	if err != nil {
		return err
	}
	status := gomadvfd.Connect(fd, address)
	if status == gomadvfd.WouldBlock {
		return EINPROGRESS
	}
	return gomadVirtualErr(status)
}

func gomadGenericAddress(fd int, p unsafe.Pointer, length uintptr, connect bool) error {
	if length != unsafe.Sizeof(RawSockaddrInet4{}) {
		return EINVAL
	}
	if connect {
		return gomadVirtualConnect(fd, p, _Socklen(length))
	}
	return gomadVirtualBind(fd, p, _Socklen(length))
}

func gomadVirtualAddressOutput(address gomadvfd.Address, raw *RawSockaddrAny, length *_Socklen) error {
	if raw == nil || length == nil {
		return EFAULT
	}
	sa := SockaddrInet4{Port: address.Port, Addr: address.IP}
	p, n, err := sa.sockaddr()
	if err != nil {
		return err
	}
	copy(unsafe.Slice((*byte)(unsafe.Pointer(raw)), min(uintptr(*length), uintptr(n))), unsafe.Slice((*byte)(p), uintptr(n)))
	*length = n
	return nil
}

func gomadVirtualAccept(fd int, raw *RawSockaddrAny, length *_Socklen, flags int) (int, error) {
	if flags & ^gomadSocketFlags != 0 {
		return -1, gomadVirtualRefuse("accept.flags")
	}
	if (raw == nil) != (length == nil) {
		return -1, EFAULT
	}
	nfd, address, status := gomadvfd.Accept(fd)
	if status != gomadvfd.OK {
		return -1, gomadVirtualErr(status)
	}
	if err := gomadVirtualCreationFlags(nfd, flags); err != nil {
		return -1, gomadVirtualCleanup(nfd, err)
	}
	if raw != nil {
		if err := gomadVirtualAddressOutput(address, raw, length); err != nil {
			return -1, gomadVirtualCleanup(nfd, err)
		}
	}
	return nfd, nil
}

func gomadVirtualName(fd int, raw *RawSockaddrAny, length *_Socklen, peer bool) error {
	if raw == nil || length == nil {
		return EFAULT
	}
	var address gomadvfd.Address
	var status gomadvfd.Status
	if peer {
		address, status = gomadvfd.Remote(fd)
	} else {
		address, status = gomadvfd.Local(fd)
	}
	if status != gomadvfd.OK {
		return gomadVirtualErr(status)
	}
	return gomadVirtualAddressOutput(address, raw, length)
}

func gomadVirtualGetsockopt(fd, level, name int, p unsafe.Pointer, length *_Socklen) error {
	if p == nil || length == nil {
		return EFAULT
	}
	if *length < 4 {
		return EINVAL
	}
	if !gomadvfd.Owns(fd) {
		return EBADF
	}
	var value int32
	switch {
	case level == SOL_SOCKET && name == SO_TYPE:
		value = SOCK_STREAM
	case level == SOL_SOCKET && name == SO_ERROR:
		_, status := gomadvfd.Remote(fd)
		if status == gomadvfd.WouldBlock {
			value = int32(EINPROGRESS)
		} else if status != gomadvfd.NotConnected {
			if err := gomadVirtualErr(status); err != nil {
				value = int32(err.(Errno))
			}
		}
	default:
		return gomadVirtualRefuse("getsockopt.option")
	}
	*(*int32)(p) = value
	*length = 4
	return nil
}

func gomadVirtualSetsockopt(fd, level, name int, p unsafe.Pointer, length uintptr) error {
	if length != 4 {
		return EINVAL
	}
	if p == nil {
		return EFAULT
	}
	if !gomadvfd.Owns(fd) {
		return EBADF
	}
	value := *(*int32)(p)
	if level == SOL_SOCKET && (name == SO_REUSEADDR || name == SO_KEEPALIVE) || level == IPPROTO_TCP && name == TCP_NODELAY {
		if value != 0 && value != 1 {
			return EINVAL
		}
		return nil
	}
	if gomadVirtualKeepaliveOption(level, name) {
		if value <= 0 || value > 32767 {
			return EINVAL
		}
		return nil
	}
	return gomadVirtualRefuse("setsockopt.option")
}

func gomadVirtualFcntl(fd, command, argument int) (int, error) {
	if !gomadvfd.Owns(fd) {
		return -1, EBADF
	}
	var status gomadvfd.Status
	switch command {
	case F_GETFL:
		flags, status := gomadvfd.Flags(fd)
		return flags | O_RDWR, gomadVirtualErr(status)
	case F_SETFL:
		if argument & ^(O_NONBLOCK|O_RDWR) != 0 {
			return -1, gomadVirtualRefuse("fcntl.status-flags")
		}
		status = gomadvfd.SetFlags(fd, argument&O_NONBLOCK)
	case F_GETFD:
		flags, status := gomadvfd.DescriptorFlags(fd)
		return flags, gomadVirtualErr(status)
	case F_SETFD:
		if argument & ^FD_CLOEXEC != 0 {
			return -1, gomadVirtualRefuse("fcntl.descriptor-flags")
		}
		status = gomadvfd.SetDescriptorFlags(fd, argument)
	default:
		return -1, gomadVirtualRefuse("fcntl.command")
	}
	return 0, gomadVirtualErr(status)
}

func gomadGenericRead(fd int, p unsafe.Pointer, count uintptr) (int, error) {
	if count > uintptr(^uint(0)>>1) {
		return 0, EINVAL
	}
	if p == nil && count != 0 {
		return 0, EFAULT
	}
	n, status := gomadvfd.Read(fd, unsafe.Slice((*byte)(p), int(count)))
	return n, gomadVirtualErr(status)
}

func gomadGenericWrite(fd int, p unsafe.Pointer, count uintptr) (int, error) {
	if count > uintptr(^uint(0)>>1) {
		return 0, EINVAL
	}
	if p == nil && count != 0 {
		return 0, EFAULT
	}
	n, status := gomadvfd.Write(fd, unsafe.Slice((*byte)(p), int(count)))
	return n, gomadVirtualErr(status)
}

func gomadGenericWritev(fd int, p *Iovec, count uintptr) (uintptr, error) {
	if count > 1024 {
		return 0, EINVAL
	}
	if p == nil && count != 0 {
		return 0, EFAULT
	}
	vectors := unsafe.Slice(p, int(count))
	var total uintptr
	for i := range vectors {
		length := uintptr(vectors[i].Len)
		if length > uintptr(^uint(0)>>1) || total > uintptr(^uint(0)>>1)-length {
			return 0, EINVAL
		}
		if vectors[i].Base == nil && length != 0 {
			return 0, EFAULT
		}
		total += length
	}
	if total == 0 {
		_, status := gomadvfd.Write(fd, nil)
		return 0, gomadVirtualErr(status)
	}
	owned := make([]byte, min(total, uintptr(64<<10)))
	var copied int
	for i := range vectors {
		length := min(uintptr(vectors[i].Len), uintptr(len(owned)-copied))
		copied += copy(owned[copied:], unsafe.Slice(vectors[i].Base, int(length)))
		if copied == len(owned) {
			break
		}
	}
	n, status := gomadvfd.Write(fd, owned)
	if n > 0 {
		return uintptr(n), nil
	}
	return 0, gomadVirtualErr(status)
}

func gomadVirtualWrite(fd int, p []byte) (int, error, bool) {
	if !gomadVirtualFD(fd) {
		return 0, nil, false
	}
	n, status := gomadvfd.Write(fd, p)
	return n, gomadVirtualErr(status), true
}

const (
	gomadOpUnknown uint8 = iota
	gomadOpSocket
	gomadOpBind
	gomadOpConnect
	gomadOpListen
	gomadOpAccept
	gomadOpRead
	gomadOpWrite
	gomadOpWritev
	gomadOpClose
	gomadOpShutdown
	gomadOpFcntl
	gomadOpSockname
	gomadOpPeername
	gomadOpGetsockopt
	gomadOpSetsockopt
	gomadOpRefused
	gomadOpRefusedSocket
)

// Raw addresses never cross a splittable call; nested iovec bases remain in
// their tracked caller array until the bounded owned copy is complete.
//
//go:nosplit
//go:norace
func gomadGenericOperation(op uint8, a1, a2, a3, a4, a5, a6 uintptr) (r1, r2 uintptr, errno Errno, handled bool) {
	if op == gomadOpUnknown || !gomadvfd.Enabled() {
		return 0, 0, 0, false
	}
	if op != gomadOpSocket && op != gomadOpRefusedSocket {
		if !gomadvfd.InRange(a1) {
			return 0, 0, 0, false
		}
		if gomadvfd.Token(a1) == 0 {
			return ^uintptr(0), 0, EBADF, true
		}
	}
	var n int
	var err error
	fd := int(a1)
	switch op {
	case gomadOpSocket:
		n, err = gomadVirtualSocket(fd, int(a2), int(a3))
	case gomadOpBind:
		err = gomadGenericAddress(fd, unsafe.Pointer(a2), a3, false)
	case gomadOpConnect:
		err = gomadGenericAddress(fd, unsafe.Pointer(a2), a3, true)
	case gomadOpListen:
		err = gomadVirtualErr(gomadvfd.Listen(fd, int(a2)))
	case gomadOpAccept:
		n, err = gomadVirtualAccept(fd, (*RawSockaddrAny)(unsafe.Pointer(a2)), (*_Socklen)(unsafe.Pointer(a3)), int(a4))
	case gomadOpRead:
		n, err = gomadGenericRead(fd, unsafe.Pointer(a2), a3)
	case gomadOpWrite:
		n, err = gomadGenericWrite(fd, unsafe.Pointer(a2), a3)
	case gomadOpWritev:
		r1, err = gomadGenericWritev(fd, (*Iovec)(unsafe.Pointer(a2)), a3)
	case gomadOpClose:
		err = gomadVirtualErr(gomadvfd.Close(fd))
	case gomadOpShutdown:
		err = gomadVirtualErr(gomadvfd.Shutdown(fd, int(a2)))
	case gomadOpFcntl:
		n, err = gomadVirtualFcntl(fd, int(a2), int(a3))
	case gomadOpSockname, gomadOpPeername:
		err = gomadVirtualName(fd, (*RawSockaddrAny)(unsafe.Pointer(a2)), (*_Socklen)(unsafe.Pointer(a3)), op == gomadOpPeername)
	case gomadOpGetsockopt:
		err = gomadVirtualGetsockopt(fd, int(a2), int(a3), unsafe.Pointer(a4), (*_Socklen)(unsafe.Pointer(a5)))
	case gomadOpSetsockopt:
		err = gomadVirtualSetsockopt(fd, int(a2), int(a3), unsafe.Pointer(a4), a5)
	case gomadOpRefused, gomadOpRefusedSocket:
		err = gomadVirtualRefuse("syscall.operation")
	}
	if err != nil {
		return ^uintptr(0), 0, err.(Errno), true
	}
	if op != gomadOpWritev {
		r1 = uintptr(n)
	}
	return r1, 0, 0, true
}
