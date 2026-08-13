package safefile

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

func Replace(path string, contents []byte, mode os.FileMode) (retErr error) {
	directory := filepath.Dir(path)
	if err := os.MkdirAll(directory, 0o755); err != nil {
		return fmt.Errorf("create parent directory: %w", err)
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
	if err := temporary.Close(); err != nil {
		return fmt.Errorf("close replacement: %w", err)
	}
	if err := os.Rename(temporaryPath, path); err != nil {
		return fmt.Errorf("publish replacement: %w", err)
	}
	return nil
}
