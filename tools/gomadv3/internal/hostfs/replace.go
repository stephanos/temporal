package hostfs

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

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
	temporary, err := os.CreateTemp(directory, ".safefile-*")
	if err != nil {
		return fmt.Errorf("create replacement: %w", err)
	}
	temporaryPath := temporary.Name()
	defer func() {
		if err := os.Remove(temporaryPath); err != nil && !errors.Is(err, os.ErrNotExist) {
			retErr = errors.Join(retErr, fmt.Errorf("remove replacement: %w", err))
		}
	}()
	if err := temporary.Chmod(mode); err != nil {
		return errors.Join(fmt.Errorf("set replacement mode: %w", err), temporary.Close())
	}
	if _, err := temporary.Write(contents); err != nil {
		return errors.Join(fmt.Errorf("write replacement: %w", err), temporary.Close())
	}
	if err := temporary.Sync(); err != nil {
		return errors.Join(fmt.Errorf("sync replacement: %w", err), temporary.Close())
	}
	if err := temporary.Close(); err != nil {
		return fmt.Errorf("close replacement: %w", err)
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := os.Rename(temporaryPath, path); err != nil {
		return fmt.Errorf("publish replacement: %w", err)
	}
	directoryFile, err := os.Open(directory)
	if err != nil {
		return fmt.Errorf("open replacement directory: %w", err)
	}
	return errors.Join(directoryFile.Sync(), directoryFile.Close())
}
