// Copyright 2026 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package syscall

import (
	"internal/abi"
	"internal/gomadvfd"
	"unsafe"
)

const gomadSocketFlags = 0

func gomadVirtualCreationFlags(fd, flags int) error {
	if flags != 0 {
		return gomadVirtualRefuse("socket.flags")
	}
	return nil
}

func gomadVirtualKeepaliveOption(level, name int) bool {
	return level == IPPROTO_TCP && (name == TCP_KEEPALIVE || name == TCP_KEEPINTVL || name == TCP_KEEPCNT)
}

type gomadLibcEntry struct {
	pc, target uintptr
	op         uint8
}

var gomadLibcEntries = [...]gomadLibcEntry{
	{pc: abi.FuncPCABI0(libc_socket_trampoline), op: gomadOpSocket},
	{pc: abi.FuncPCABI0(libc_bind_trampoline), op: gomadOpBind},
	{pc: abi.FuncPCABI0(libc_connect_trampoline), op: gomadOpConnect},
	{pc: abi.FuncPCABI0(libc_listen_trampoline), op: gomadOpListen},
	{pc: abi.FuncPCABI0(libc_accept_trampoline), op: gomadOpAccept},
	{pc: abi.FuncPCABI0(libc_read_trampoline), op: gomadOpRead},
	{pc: abi.FuncPCABI0(libc_readv_trampoline), op: gomadOpRefused},
	{pc: abi.FuncPCABI0(libc_write_trampoline), op: gomadOpWrite},
	{pc: abi.FuncPCABI0(libc_writev_trampoline), op: gomadOpWritev},
	{pc: abi.FuncPCABI0(libc_close_trampoline), op: gomadOpClose},
	{pc: abi.FuncPCABI0(libc_shutdown_trampoline), op: gomadOpShutdown},
	{pc: abi.FuncPCABI0(libc_fcntl_trampoline), op: gomadOpFcntl},
	{pc: abi.FuncPCABI0(libc_getsockname_trampoline), op: gomadOpSockname},
	{pc: abi.FuncPCABI0(libc_getpeername_trampoline), op: gomadOpPeername},
	{pc: abi.FuncPCABI0(libc_getsockopt_trampoline), op: gomadOpGetsockopt},
	{pc: abi.FuncPCABI0(libc_setsockopt_trampoline), op: gomadOpSetsockopt},
	{pc: abi.FuncPCABI0(libc_socketpair_trampoline), op: gomadOpRefusedSocket},
	{pc: abi.FuncPCABI0(libc_recvfrom_trampoline), op: gomadOpRefused},
	{pc: abi.FuncPCABI0(libc_sendto_trampoline), op: gomadOpRefused},
	{pc: abi.FuncPCABI0(libc_recvmsg_trampoline), op: gomadOpRefused},
	{pc: abi.FuncPCABI0(libc_sendmsg_trampoline), op: gomadOpRefused},
	{pc: abi.FuncPCABI0(libc_ioctl_trampoline), op: gomadOpRefused},
	{pc: abi.FuncPCABI0(libc_dup_trampoline), op: gomadOpRefused},
	{pc: abi.FuncPCABI0(libc_dup2_trampoline), op: gomadOpRefused},
}

func libc_readv_trampoline()

//go:cgo_import_dynamic libc_readv readv "/usr/lib/libSystem.B.dylib"

func init() {
	for i := range gomadLibcEntries {
		gomadLibcEntries[i].target = gomadBranchTarget(gomadLibcEntries[i].pc)
	}
}

//go:nosplit
//go:norace
func gomadBranchTarget(pc uintptr) uintptr {
	if pc == 0 || pc&3 != 0 {
		return 0
	}
	ins := *(*uint32)(unsafe.Pointer(pc))
	if ins&0xfc000000 != 0x14000000 {
		return 0
	}
	return uintptr(int64(pc) + int64(int32(ins<<6)>>6)*4)
}

//go:nosplit
//go:norace
func gomadDarwinOperation(fn uintptr) uint8 {
	target := gomadBranchTarget(fn)
	for i := range gomadLibcEntries {
		entry := &gomadLibcEntries[i]
		if fn == entry.pc && entry.target == 0 {
			if entry.op == gomadOpSocket || entry.op == gomadOpRefusedSocket {
				return gomadOpRefusedSocket
			}
			return gomadOpRefused
		}
		if entry.target != 0 && (fn == entry.pc || fn == entry.target || target == entry.target) {
			return entry.op
		}
	}
	return gomadOpUnknown
}

//go:nosplit
//go:norace
func gomadGeneric(fn, a1, a2, a3, a4, a5, a6 uintptr) (uintptr, uintptr, Errno, bool) {
	if !gomadvfd.Enabled() {
		return 0, 0, 0, false
	}
	op := gomadDarwinOperation(fn)
	if op == gomadOpAccept {
		a4 = 0
	}
	return gomadGenericOperation(op, a1, a2, a3, a4, a5, a6)
}
