//go:build linux

package hooks

import "github.com/temporalio/gomad/internal/simulation"

func InternalRuntimeSyscallLinux_Syscall6(num, a1, a2, a3, a4, a5, a6 uintptr) (r1, r2, errno uintptr) {
	r1, r2, err := simulation.RawSyscall6(num, a1, a2, a3, a4, a5, a6)
	return r1, r2, uintptr(err)
}
