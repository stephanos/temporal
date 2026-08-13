//go:build !aix && !darwin && !dragonfly && !freebsd && !linux && !netbsd && !openbsd && !solaris

package guide

import (
	"fmt"
	"os"
)

func acquireLock(string) (*os.File, error) {
	return nil, fmt.Errorf("guided corpus is unsupported on this host")
}

func releaseLock(file *os.File) error {
	return file.Close()
}
