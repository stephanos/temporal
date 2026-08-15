//go:build windows

package evidence

import "os"

func renameNoReplace(oldPath, newPath string) error {
	return os.Rename(oldPath, newPath)
}
