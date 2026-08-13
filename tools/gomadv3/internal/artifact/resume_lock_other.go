//go:build !aix && !darwin && !dragonfly && !freebsd && !linux && !netbsd && !openbsd && !solaris

package artifact

import (
	"fmt"
	"os"
)

func acquireResumeLock(string) (*os.File, error) {
	return nil, fmt.Errorf("batch resume is unsupported on this host")
}

func releaseResumeLock(file *os.File) error {
	return file.Close()
}
