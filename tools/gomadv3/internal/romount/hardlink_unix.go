//go:build unix

package romount

import (
	"os"
	"syscall"
)

func hardLinked(info os.FileInfo) bool {
	stat, ok := info.Sys().(*syscall.Stat_t)
	return ok && stat.Nlink > 1
}
