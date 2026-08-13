package toolchainbuild

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"time"

	"go.temporal.io/server/tools/gomadv3/internal/filelock"
)

type buildLock struct {
	lock *filelock.Lock
}

func acquireBuildLock(ctx context.Context, path string) (*buildLock, bool, error) {
	waited := false
	for {
		lock, err := filelock.Try(path)
		switch {
		case err == nil:
			return &buildLock{lock: lock}, waited, nil
		case errors.Is(err, filelock.ErrSymbolicLink):
			return nil, waited, errors.New("gomadv3 build lock is a symbolic link")
		case errors.Is(err, filelock.ErrUnsupported):
			return nil, waited, fmt.Errorf("gomadv3 toolchain build locking is unsupported on %s", runtime.GOOS)
		case !errors.Is(err, filelock.ErrContended):
			return nil, waited, fmt.Errorf("lock gomadv3 build: %w", err)
		}
		waited = true
		select {
		case <-ctx.Done():
			return nil, waited, ctx.Err()
		case <-time.After(10 * time.Millisecond):
		}
	}
}

func (lock *buildLock) release() error {
	if lock == nil {
		return nil
	}
	return lock.lock.Release()
}
