// Copyright 2026 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package syscall

import "internal/gomadvfd"

const gomadSocketFlags = SOCK_NONBLOCK | SOCK_CLOEXEC
const gomadSysSendmmsg = 307

func gomadVirtualCreationFlags(fd, flags int) error {
	if flags&SOCK_NONBLOCK != 0 {
		if status := gomadvfd.SetFlags(fd, O_NONBLOCK); status != gomadvfd.OK {
			return gomadVirtualErr(status)
		}
	}
	if flags&SOCK_CLOEXEC != 0 {
		return gomadVirtualErr(gomadvfd.SetDescriptorFlags(fd, FD_CLOEXEC))
	}
	return nil
}

func gomadVirtualKeepaliveOption(level, name int) bool {
	return level == IPPROTO_TCP && (name == TCP_KEEPIDLE || name == TCP_KEEPINTVL || name == TCP_KEEPCNT)
}

//go:nosplit
//go:norace
func gomadLinuxOperation(trap uintptr) uint8 {
	switch trap {
	case SYS_SOCKET:
		return gomadOpSocket
	case SYS_BIND:
		return gomadOpBind
	case SYS_CONNECT:
		return gomadOpConnect
	case SYS_LISTEN:
		return gomadOpListen
	case SYS_ACCEPT, SYS_ACCEPT4:
		return gomadOpAccept
	case SYS_READ:
		return gomadOpRead
	case SYS_WRITE:
		return gomadOpWrite
	case SYS_WRITEV:
		return gomadOpWritev
	case SYS_CLOSE:
		return gomadOpClose
	case SYS_SHUTDOWN:
		return gomadOpShutdown
	case SYS_FCNTL:
		return gomadOpFcntl
	case SYS_GETSOCKNAME:
		return gomadOpSockname
	case SYS_GETPEERNAME:
		return gomadOpPeername
	case SYS_GETSOCKOPT:
		return gomadOpGetsockopt
	case SYS_SETSOCKOPT:
		return gomadOpSetsockopt
	case SYS_RECVFROM, SYS_SENDTO, SYS_RECVMSG, SYS_SENDMSG, SYS_RECVMMSG, gomadSysSendmmsg, SYS_READV, SYS_IOCTL, SYS_DUP, SYS_DUP2, SYS_DUP3:
		return gomadOpRefused
	case SYS_SOCKETPAIR:
		return gomadOpRefusedSocket
	}
	return gomadOpUnknown
}

//go:nosplit
//go:norace
func gomadGeneric(trap, a1, a2, a3, a4, a5, a6 uintptr) (uintptr, uintptr, Errno, bool) {
	op := gomadLinuxOperation(trap)
	if trap == SYS_ACCEPT {
		a4 = 0
	}
	return gomadGenericOperation(op, a1, a2, a3, a4, a5, a6)
}
