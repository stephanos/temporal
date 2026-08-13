package guide

import (
	"errors"
	"fmt"

	"go.temporal.io/server/tools/gomadv3/internal/filelock"
)

func acquireLock(path string) (*filelock.Lock, error) {
	lock, err := filelock.Try(path)
	switch {
	case errors.Is(err, filelock.ErrSymbolicLink):
		return nil, errors.New("guided corpus lock is a symbolic link")
	case errors.Is(err, filelock.ErrContended):
		return nil, fmt.Errorf("guided corpus is already in use: %w", err)
	case errors.Is(err, filelock.ErrUnsupported):
		return nil, errors.New("guided corpus is unsupported on this host")
	case err != nil:
		return nil, fmt.Errorf("open guided corpus lock: %w", err)
	default:
		return lock, nil
	}
}

func releaseLock(lock *filelock.Lock) error {
	return lock.Release()
}
