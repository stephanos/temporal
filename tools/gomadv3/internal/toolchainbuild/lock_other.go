//go:build !aix && !darwin && !dragonfly && !freebsd && !linux && !netbsd && !openbsd && !solaris

package toolchainbuild

import (
	"context"
	"fmt"
	"runtime"
)

type buildLock struct{}

func acquireBuildLock(context.Context, string) (*buildLock, bool, error) {
	return nil, false, fmt.Errorf("gomadv3 toolchain build locking is unsupported on %s", runtime.GOOS)
}

func (*buildLock) release() error {
	return nil
}
