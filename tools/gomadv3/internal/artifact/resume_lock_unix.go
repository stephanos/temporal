//go:build aix || darwin || dragonfly || freebsd || linux || netbsd || openbsd || solaris

package artifact

import (
	"errors"
	"fmt"
	"os"
	"syscall"
)

func acquireResumeLock(path string) (*os.File, error) {
	if info, err := os.Lstat(path); err == nil && info.Mode()&os.ModeSymlink != 0 {
		return nil, fmt.Errorf("resume lock is a symbolic link")
	} else if err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, fmt.Errorf("open resume lock: %w", err)
	}
	if err := file.Chmod(0o600); err != nil {
		return nil, errors.Join(err, file.Close())
	}
	if err := syscall.Flock(int(file.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		return nil, errors.Join(fmt.Errorf("batch is already being resumed: %w", err), file.Close())
	}
	return file, nil
}

func releaseResumeLock(file *os.File) error {
	return errors.Join(syscall.Flock(int(file.Fd()), syscall.LOCK_UN), file.Close())
}
