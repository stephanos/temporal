//go:build !darwin && !linux && !windows

package hostfs

import "fmt"

// RenameNoReplace reports the retained unsupported-platform behavior.
func RenameNoReplace(_, _ string) error {
	return fmt.Errorf("atomic no-replace publication is unsupported on this platform")
}
