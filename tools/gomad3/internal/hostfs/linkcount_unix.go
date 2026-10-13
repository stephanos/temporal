//go:build unix

package hostfs

import (
	"fmt"
	"os"
	"syscall"
)

// LinkCount returns the number of names for info when the host exposes it.
func LinkCount(info os.FileInfo) (uint64, bool) {
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return 0, false
	}
	return uint64(stat.Nlink), true
}

func validateLinkCount(info os.FileInfo) error {
	if links, known := LinkCount(info); known && links != 1 {
		return fmt.Errorf("%s has unexpected link count %d", info.Name(), links)
	}
	return nil
}
