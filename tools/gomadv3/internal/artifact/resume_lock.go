package artifact

import (
	"errors"
	"fmt"

	"go.temporal.io/server/tools/gomadv3/internal/filelock"
)

func acquireResumeLock(path string) (*filelock.Lock, error) {
	lock, err := filelock.Try(path)
	switch {
	case errors.Is(err, filelock.ErrSymbolicLink):
		return nil, errors.New("resume lock is a symbolic link")
	case errors.Is(err, filelock.ErrContended):
		return nil, fmt.Errorf("batch is already being resumed: %w", err)
	case errors.Is(err, filelock.ErrUnsupported):
		return nil, errors.New("batch resume is unsupported on this host")
	case err != nil:
		return nil, fmt.Errorf("open resume lock: %w", err)
	default:
		return lock, nil
	}
}

func releaseResumeLock(lock *filelock.Lock) error {
	return lock.Release()
}
