//go:build !darwin && !linux && !windows

package evidence

import "fmt"

func renameNoReplace(_, _ string) error {
	return fmt.Errorf("atomic no-replace publication is unsupported on this platform")
}
