//go:build unix

package artifact

import (
	"fmt"
	"os"
	"syscall"
)

func validateLinkCount(info os.FileInfo) error {
	if stat, ok := info.Sys().(*syscall.Stat_t); ok && stat.Nlink != 1 {
		return fmt.Errorf("%s has unexpected link count %d", info.Name(), stat.Nlink)
	}
	return nil
}
