//go:build unix

package artifact

import (
	"os"
	"syscall"
)

// linkCount is the number of names the file has, when the platform reports it.
func linkCount(info os.FileInfo) (uint64, bool) {
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return 0, false
	}
	return uint64(stat.Nlink), true
}
