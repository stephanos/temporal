package hostfs

import (
	"context"
	"errors"
	"io"
	"os"

	internal "go.temporal.io/server/tools/gomad3/internal/hostfs"
)

// ErrSymbolicLink marks a path refused because it is a symbolic link.
var ErrSymbolicLink = internal.ErrSymbolicLink

func ReadBounded(path string, maximum uint64) ([]byte, error) {
	return internal.ReadBounded(path, maximum)
}

// OpenPath opens a single-link regular file without following symbolic links.
func OpenPath(path string) (*os.File, os.FileInfo, error) {
	return internal.OpenPath(path)
}

// LinkCount returns the platform link count when the host exposes it.
func LinkCount(info os.FileInfo) (uint64, bool) {
	return internal.LinkCount(info)
}

// StagedFile is a synced and closed same-directory publication candidate.
type StagedFile struct {
	staged *internal.StagedFile
}

// StageContext fills, syncs and closes a temporary file in directory.
func StageContext(ctx context.Context, directory, pattern string, mode os.FileMode, fill func(context.Context, io.Writer) error) (*StagedFile, error) {
	staged, err := internal.StageContext(ctx, directory, pattern, mode, fill)
	if err != nil {
		return nil, err
	}
	return &StagedFile{staged: staged}, nil
}

// Path returns the staged path for caller-owned validation before publication.
func (staged *StagedFile) Path() string {
	if staged == nil {
		return ""
	}
	return staged.staged.Path()
}

// PublishContext atomically replaces destination with the staged file.
func (staged *StagedFile) PublishContext(ctx context.Context, destination string) (bool, error) {
	if staged == nil {
		return false, errors.New("staged file is required")
	}
	return staged.staged.PublishContext(ctx, destination)
}

// PublishNoReplaceContext atomically publishes only when destination is absent.
func (staged *StagedFile) PublishNoReplaceContext(ctx context.Context, destination string) (bool, error) {
	if staged == nil {
		return false, errors.New("staged file is required")
	}
	return staged.staged.PublishNoReplaceContext(ctx, destination)
}

// Cleanup removes an unpublished stage and is safe to call repeatedly.
func (staged *StagedFile) Cleanup() error {
	if staged == nil {
		return nil
	}
	return staged.staged.Cleanup()
}

// CleanupAndSync removes an unpublished stage and flushes that removal. It is
// safe to call repeatedly.
func (staged *StagedFile) CleanupAndSync() error {
	if staged == nil {
		return nil
	}
	return staged.staged.CleanupAndSync()
}
