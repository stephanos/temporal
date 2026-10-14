//go:build unix

package publish

import (
	"os"
	"syscall"
)

// openExisting opens a name without following a final symlink, and without blocking on a FIFO
// swapped in after it was inspected.
func openExisting(path string) (*os.File, error) {
	return os.OpenFile(path, os.O_RDONLY|syscall.O_NOFOLLOW|syscall.O_NONBLOCK, 0)
}
