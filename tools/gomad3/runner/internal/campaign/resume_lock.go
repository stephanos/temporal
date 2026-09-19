package campaign

import (
	"context"
	"errors"
	"fmt"

	"go.temporal.io/server/tools/gomad3/internal/hostfs"
)

func acquireResumeLock(ctx context.Context, path string) (*hostfs.Lock, error) {
	if err := observeMutation(ctx, mutationCreate, "resume-lock"); err != nil {
		return nil, err
	}
	lock, err := hostfs.Try(path)
	switch {
	case errors.Is(err, hostfs.ErrSymbolicLink):
		return nil, errors.New("resume lock is a symbolic link")
	case errors.Is(err, hostfs.ErrContended):
		return nil, fmt.Errorf("campaign is already being resumed: %w", err)
	case errors.Is(err, hostfs.ErrUnsupported):
		return nil, errors.New("campaign resume is unsupported on this host")
	case err != nil:
		return nil, fmt.Errorf("open resume lock: %w", err)
	default:
		return lock, nil
	}
}

func releaseResumeLock(lock *hostfs.Lock) error {
	return lock.Release()
}
