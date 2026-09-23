//go:build unix

package cli

import (
	"os"
	"syscall"
)

// openExisting opens a name without following a final symlink, and without blocking on a FIFO
// swapped in after it was inspected.
func openExisting(path string) (*os.File, error) {
	descriptor, err := syscall.Open(path, syscall.O_RDONLY|syscall.O_CLOEXEC|syscall.O_NOFOLLOW|syscall.O_NONBLOCK, 0)
	if err != nil {
		return nil, err
	}
	return os.NewFile(uintptr(descriptor), path), nil
}
