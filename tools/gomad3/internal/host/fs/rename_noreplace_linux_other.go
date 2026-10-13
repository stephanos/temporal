//go:build linux && !amd64 && !arm64

package hostfs

import "fmt"

// RenameNoReplace reports the retained unsupported-architecture behavior.
func RenameNoReplace(_, _ string) error {
	return fmt.Errorf("atomic no-replace publication is unsupported on this Linux architecture")
}
