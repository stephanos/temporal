package toolchain

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"time"

	"go.temporal.io/server/tools/gomad3/internal/hostfs"
)

type buildLock struct {
	lock *hostfs.Lock
}

func acquireBuildLock(ctx context.Context, path string) (*buildLock, bool, error) {
	waited := false
	for {
		lock, err := hostfs.Try(path)
		switch {
		case err == nil:
			return &buildLock{lock: lock}, waited, nil
		case errors.Is(err, hostfs.ErrSymbolicLink):
			return nil, waited, errors.New("gomad3 build lock is a symbolic link")
		case errors.Is(err, hostfs.ErrUnsupported):
			return nil, waited, fmt.Errorf("gomad3 toolchain build locking is unsupported on %s", runtime.GOOS)
		case !errors.Is(err, hostfs.ErrContended):
			return nil, waited, fmt.Errorf("lock gomad3 build: %w", err)
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
