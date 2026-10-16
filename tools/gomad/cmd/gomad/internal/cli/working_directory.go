package cli

import (
	"fmt"
	"os"
	"path/filepath"
)

// invalidWorkingDirectoryError marks a --working-dir the caller supplied that
// cannot name a target module root; it is invalid input, not a Runner failure.
type invalidWorkingDirectoryError struct {
	err error
}

func (e invalidWorkingDirectoryError) Error() string {
	return e.err.Error()
}

func (e invalidWorkingDirectoryError) Unwrap() error {
	return e.err
}

// resolveWorkingDirectory returns the explicit --working-dir after validating
// it, or the process working directory when none was given, so a module outside
// the current directory can be targeted without changing directories.
func resolveWorkingDirectory(explicit string, current func() (string, error)) (string, error) {
	if explicit == "" {
		return current()
	}
	if err := validateWorkingDirectory(explicit); err != nil {
		return "", invalidWorkingDirectoryError{err: err}
	}
	return explicit, nil
}

func validateWorkingDirectory(directory string) error {
	if !filepath.IsAbs(directory) || filepath.Clean(directory) != directory {
		return fmt.Errorf("working directory %q must be an absolute, clean path", directory)
	}
	info, err := os.Stat(directory)
	if err != nil {
		return fmt.Errorf("working directory %q: %w", directory, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("working directory %q is not a directory", directory)
	}
	info, err = os.Stat(filepath.Join(directory, "go.mod"))
	if os.IsNotExist(err) {
		return fmt.Errorf("working directory %q is not a module root: it has no go.mod", directory)
	}
	if err != nil {
		return fmt.Errorf("working directory %q: %w", directory, err)
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("working directory %q has a go.mod that is not a regular file", directory)
	}
	return nil
}
