package hostfs

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
)

type stagedHandle interface {
	io.Writer
	Name() string
	Chmod(os.FileMode) error
	Sync() error
	Close() error
}

type stagedOperationError struct {
	operation string
	err       error
}

func (err *stagedOperationError) Error() string { return err.operation + ": " + err.err.Error() }
func (err *stagedOperationError) Unwrap() error { return err.err }

type publicationOperations struct {
	createTemp      func(string, string) (stagedHandle, error)
	rename          func(string, string) error
	renameNoReplace func(string, string) error
	remove          func(string) error
	syncDirectory   func(string) error
}

var defaultPublicationOperations = publicationOperations{
	createTemp: func(directory, pattern string) (stagedHandle, error) {
		return os.CreateTemp(directory, pattern)
	},
	rename:          os.Rename,
	renameNoReplace: RenameNoReplace,
	remove:          os.Remove,
	syncDirectory:   SyncDirectory,
}

// StagedFile is a synced and closed same-directory publication candidate.
type StagedFile struct {
	mu         sync.Mutex
	path       string
	operations publicationOperations
}

// Path returns the staged path for caller-owned validation before publication.
func (staged *StagedFile) Path() string {
	if staged == nil {
		return ""
	}
	staged.mu.Lock()
	defer staged.mu.Unlock()
	return staged.path
}

// StageContext fills, syncs and closes a temporary file in directory.
func StageContext(ctx context.Context, directory, pattern string, mode os.FileMode, fill func(context.Context, io.Writer) error) (_ *StagedFile, retErr error) {
	return stageContext(ctx, directory, pattern, mode, fill, defaultPublicationOperations)
}

func stageContext(ctx context.Context, directory, pattern string, mode os.FileMode, fill func(context.Context, io.Writer) error, operations publicationOperations) (_ *StagedFile, retErr error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	temporary, err := operations.createTemp(directory, pattern)
	if err != nil {
		return nil, fmt.Errorf("create staged file: %w", err)
	}
	staged := &StagedFile{path: temporary.Name(), operations: operations}
	closed := false
	complete := false
	defer func() {
		if complete {
			return
		}
		if !closed {
			closed = true
			if err := temporary.Close(); err != nil {
				retErr = errors.Join(retErr, err)
			}
		}
		if err := staged.Cleanup(); err != nil {
			retErr = errors.Join(retErr, err)
		}
	}()
	if err := temporary.Chmod(mode); err != nil {
		return nil, fmt.Errorf("set staged file mode: %w", err)
	}
	if fill == nil {
		return nil, errors.New("staged file fill is required")
	}
	if err := fill(ctx, temporary); err != nil {
		return nil, &stagedOperationError{operation: "fill staged file", err: err}
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := temporary.Sync(); err != nil {
		return nil, fmt.Errorf("sync staged file: %w", err)
	}
	closed = true
	if err := temporary.Close(); err != nil {
		return nil, fmt.Errorf("close staged file: %w", err)
	}
	complete = true
	return staged, nil
}

// PublishContext atomically replaces destination with the staged file. The
// returned disposition is true once the rename has happened, even if flushing
// the directory reports an error.
func (staged *StagedFile) PublishContext(ctx context.Context, destination string) (bool, error) {
	if staged == nil {
		return false, errors.New("staged file is required")
	}
	return staged.publishContext(ctx, destination, staged.operations.rename)
}

// PublishNoReplaceContext atomically publishes the staged file only when the
// destination is absent.
func (staged *StagedFile) PublishNoReplaceContext(ctx context.Context, destination string) (bool, error) {
	if staged == nil {
		return false, errors.New("staged file is required")
	}
	return staged.publishContext(ctx, destination, staged.operations.renameNoReplace)
}

func (staged *StagedFile) publishContext(ctx context.Context, destination string, rename func(string, string) error) (bool, error) {
	if staged == nil {
		return false, errors.New("staged file is required")
	}
	staged.mu.Lock()
	defer staged.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return false, err
	}
	if staged.path == "" {
		return false, errors.New("staged file is no longer available")
	}
	if filepath.Clean(filepath.Dir(staged.path)) != filepath.Clean(filepath.Dir(destination)) {
		return false, errors.New("staged file and destination must share a directory")
	}
	if err := rename(staged.path, destination); err != nil {
		return false, fmt.Errorf("publish staged file: %w", err)
	}
	staged.path = ""
	if err := staged.operations.syncDirectory(filepath.Dir(destination)); err != nil {
		return true, fmt.Errorf("sync publication directory: %w", err)
	}
	return true, nil
}

// Cleanup removes an unpublished stage. It is safe to call repeatedly.
func (staged *StagedFile) Cleanup() error {
	return staged.cleanup(false)
}

// CleanupAndSync removes an unpublished stage and flushes that removal before
// returning. It is safe to call repeatedly.
func (staged *StagedFile) CleanupAndSync() error {
	return staged.cleanup(true)
}

func (staged *StagedFile) cleanup(syncDirectory bool) error {
	if staged == nil {
		return nil
	}
	staged.mu.Lock()
	defer staged.mu.Unlock()
	if staged.path == "" {
		return nil
	}
	directory := filepath.Dir(staged.path)
	err := staged.operations.remove(staged.path)
	if err == nil {
		staged.path = ""
		if syncDirectory {
			if err := staged.operations.syncDirectory(directory); err != nil {
				return err
			}
		}
		return nil
	}
	if errors.Is(err, os.ErrNotExist) {
		staged.path = ""
		return err
	}
	return err
}

func Replace(path string, contents []byte, mode os.FileMode) (retErr error) {
	return ReplaceContext(context.Background(), path, contents, mode)
}

func ReplaceContext(ctx context.Context, path string, contents []byte, mode os.FileMode) (retErr error) {
	if err := ctx.Err(); err != nil {
		return err
	}
	directory := filepath.Dir(path)
	if err := os.MkdirAll(directory, 0o755); err != nil {
		return fmt.Errorf("create parent directory: %w", err)
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	staged, err := StageContext(ctx, directory, ".safefile-*", mode, func(_ context.Context, writer io.Writer) error {
		_, err := writer.Write(contents)
		return err
	})
	if err != nil {
		return err
	}
	defer func() {
		if err := staged.Cleanup(); err != nil {
			retErr = errors.Join(retErr, err)
		}
	}()
	_, err = staged.PublishContext(ctx, path)
	return err
}
