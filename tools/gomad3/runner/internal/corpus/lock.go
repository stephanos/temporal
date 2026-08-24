package corpus

import (
	"errors"
	"fmt"

	"go.temporal.io/server/tools/gomad3/internal/hostfs"
)

func acquireLock(path string) (*hostfs.Lock, error) {
	lock, err := hostfs.Try(path)
	switch {
	case errors.Is(err, hostfs.ErrSymbolicLink):
		return nil, errors.New("guided corpus lock is a symbolic link")
	case errors.Is(err, hostfs.ErrContended):
		return nil, fmt.Errorf("guided corpus is already in use: %w", err)
	case errors.Is(err, hostfs.ErrUnsupported):
		return nil, errors.New("guided corpus is unsupported on this host")
	case err != nil:
		return nil, fmt.Errorf("open guided corpus lock: %w", err)
	default:
		return lock, nil
	}
}

func releaseLock(lock *hostfs.Lock) error {
	return lock.Release()
}
