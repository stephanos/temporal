//go:build windows

package artifact

import "os"

func renameNoReplace(oldPath, newPath string) error {
	return os.Rename(oldPath, newPath)
}
