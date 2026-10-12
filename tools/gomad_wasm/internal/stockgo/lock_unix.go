//go:build unix

package stockgo

import (
	"os"
	"syscall"
)

// lock blocks until this process holds an exclusive advisory lock on path.
// The kernel releases it if the process dies mid-copy.
func lock(path string) (unlock func(), err error) {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, err
	}
	if err := syscall.Flock(int(file.Fd()), syscall.LOCK_EX); err != nil {
		_ = file.Close()
		return nil, err
	}
	return func() {
		_ = syscall.Flock(int(file.Fd()), syscall.LOCK_UN)
		_ = file.Close()
	}, nil
}
