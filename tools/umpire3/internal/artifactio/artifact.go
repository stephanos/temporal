package artifactio

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

func Publish(path string, encoded []byte) error {
	if path == "" || filepath.Base(path) == "." {
		return errors.New("artifact path is required")
	}
	directory := filepath.Dir(path)
	if err := os.MkdirAll(directory, 0o700); err != nil {
		return fmt.Errorf("create artifact directory: %w", err)
	}
	temporary, err := os.CreateTemp(directory, ".umpire3-artifact-*")
	if err != nil {
		return fmt.Errorf("create temporary artifact: %w", err)
	}
	temporaryPath := temporary.Name()
	defer func() { _ = os.Remove(temporaryPath) }()
	if err := temporary.Chmod(0o600); err != nil {
		return closeWithError(temporary, fmt.Errorf("protect temporary artifact: %w", err))
	}
	if _, err := temporary.Write(encoded); err != nil {
		return closeWithError(temporary, fmt.Errorf("write temporary artifact: %w", err))
	}
	if err := temporary.Sync(); err != nil {
		return closeWithError(temporary, fmt.Errorf("sync temporary artifact: %w", err))
	}
	if err := temporary.Close(); err != nil {
		return fmt.Errorf("close temporary artifact: %w", err)
	}
	if err := os.Rename(temporaryPath, path); err != nil {
		return fmt.Errorf("publish artifact: %w", err)
	}
	return syncDirectory(directory)
}

func Remove(path string) error {
	if path == "" || filepath.Base(path) == "." {
		return errors.New("artifact path is required")
	}
	if err := os.Remove(path); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("remove artifact: %w", err)
	}
	return syncDirectory(filepath.Dir(path))
}

func syncDirectory(path string) error {
	directory, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open artifact directory: %w", err)
	}
	syncErr := directory.Sync()
	closeErr := directory.Close()
	if syncErr != nil || closeErr != nil {
		return fmt.Errorf("sync artifact directory: %w", errors.Join(syncErr, closeErr))
	}
	return nil
}

func closeWithError(file *os.File, operationErr error) error {
	if closeErr := file.Close(); closeErr != nil {
		return errors.Join(operationErr, closeErr)
	}
	return operationErr
}
