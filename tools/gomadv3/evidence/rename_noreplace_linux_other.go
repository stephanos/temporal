//go:build linux && !amd64 && !arm64

package evidence

import "fmt"

func renameNoReplace(_, _ string) error {
	return fmt.Errorf("atomic no-replace publication is unsupported on this Linux architecture")
}
