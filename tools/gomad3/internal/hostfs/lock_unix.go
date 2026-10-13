//go:build aix || darwin || dragonfly || freebsd || linux || netbsd || openbsd || solaris

package hostfs

import (
	"errors"
	"os"
	"syscall"
)

type Lock struct {
	file *os.File
}

// Try takes an exclusive lock on path without waiting.
func Try(path string) (*Lock, error) {
	return lock(path, syscall.LOCK_EX|syscall.LOCK_NB)
}

// Shared waits for a shared lock on path; shared holders exclude a Try
// holder but not each other.
func Shared(path string) (*Lock, error) {
	return lock(path, syscall.LOCK_SH)
}

func lock(path string, how int) (*Lock, error) {
	if info, err := os.Lstat(path); err == nil && info.Mode()&os.ModeSymlink != 0 {
		return nil, ErrSymbolicLink
	} else if err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, err
	}
	if err := file.Chmod(0o600); err != nil {
		return nil, errors.Join(err, file.Close())
	}
	if err := flock(file, how); err != nil {
		return nil, errors.Join(err, file.Close())
	}
	return &Lock{file: file}, nil
}

// TryDescriptor takes an exclusive lock without waiting on a file the caller
// has already opened and validated. On success the Lock owns file and Release
// closes it; on failure the caller still owns file.
func TryDescriptor(file *os.File) (*Lock, error) {
	if file == nil {
		return nil, errors.New("lock descriptor is required")
	}
	if err := flock(file, syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		return nil, err
	}
	return &Lock{file: file}, nil
}

func flock(file *os.File, how int) error {
	err := syscall.Flock(int(file.Fd()), how)
	for errors.Is(err, syscall.EINTR) {
		err = syscall.Flock(int(file.Fd()), how)
	}
	if errors.Is(err, syscall.EWOULDBLOCK) || errors.Is(err, syscall.EAGAIN) {
		err = errors.Join(ErrContended, err)
	}
	return err
}

func (lock *Lock) Release() error {
	if lock == nil || lock.file == nil {
		return nil
	}
	file := lock.file
	lock.file = nil
	return errors.Join(syscall.Flock(int(file.Fd()), syscall.LOCK_UN), file.Close())
}
