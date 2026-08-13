//go:build aix || darwin || dragonfly || freebsd || linux || netbsd || openbsd || solaris

package toolchainbuild

import (
	"context"
	"errors"
	"fmt"
	"os"
	"syscall"
	"time"
)

type buildLock struct {
	file *os.File
}

func acquireBuildLock(ctx context.Context, path string) (*buildLock, bool, error) {
	if info, err := os.Lstat(path); err == nil && info.Mode()&os.ModeSymlink != 0 {
		return nil, false, errors.New("gomadv3 build lock is a symbolic link")
	} else if err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, false, err
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, false, fmt.Errorf("open gomadv3 build lock: %w", err)
	}
	waited := false
	for {
		err := syscall.Flock(int(file.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
		if err == nil {
			return &buildLock{file: file}, waited, nil
		}
		if !errors.Is(err, syscall.EWOULDBLOCK) && !errors.Is(err, syscall.EAGAIN) {
			return nil, waited, errors.Join(fmt.Errorf("lock gomadv3 build: %w", err), file.Close())
		}
		waited = true
		select {
		case <-ctx.Done():
			return nil, waited, errors.Join(ctx.Err(), file.Close())
		case <-time.After(10 * time.Millisecond):
		}
	}
}

func (lock *buildLock) release() error {
	if lock == nil || lock.file == nil {
		return nil
	}
	return errors.Join(syscall.Flock(int(lock.file.Fd()), syscall.LOCK_UN), lock.file.Close())
}
