//go:build linux

package execution

import "syscall"

func dup2(oldfd, newfd int) error {
	if oldfd == newfd {
		return nil
	}
	return syscall.Dup3(oldfd, newfd, 0)
}
