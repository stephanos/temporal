//go:build darwin

package evidence

import (
	"syscall"
	"unsafe"
)

const (
	renameatxNPSyscall = 488
	renameExclusive    = 0x4
)

func renameNoReplace(oldPath, newPath string) error {
	oldPointer, err := syscall.BytePtrFromString(oldPath)
	if err != nil {
		return err
	}
	newPointer, err := syscall.BytePtrFromString(newPath)
	if err != nil {
		return err
	}
	_, _, errno := syscall.Syscall6(
		renameatxNPSyscall,
		^uintptr(1), uintptr(unsafe.Pointer(oldPointer)),
		^uintptr(1), uintptr(unsafe.Pointer(newPointer)),
		renameExclusive, 0,
	)
	if errno != 0 {
		return errno
	}
	return nil
}
