//go:build unix && !linux

package execution

import "syscall"

func dup2(oldfd, newfd int) error {
	return syscall.Dup2(oldfd, newfd)
}
