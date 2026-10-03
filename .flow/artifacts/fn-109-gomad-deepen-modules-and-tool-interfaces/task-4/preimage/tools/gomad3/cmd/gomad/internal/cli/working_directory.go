package cli

import (
	"go.temporal.io/server/tools/gomad3/target"
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
	if err := target.ValidateWorkingDirectory(explicit); err != nil {
		return "", invalidWorkingDirectoryError{err: err}
	}
	return explicit, nil
}
