//go:build windows

package hostfs

import "os"

// RenameNoReplace retains the existing Windows replacement behavior.
func RenameNoReplace(oldPath, newPath string) error {
	return os.Rename(oldPath, newPath)
}
