//go:build !aix && !darwin && !dragonfly && !freebsd && !linux && !netbsd && !openbsd && !solaris

package hostfs

import "os"

type Lock struct{}

func Try(string) (*Lock, error) {
	return nil, ErrUnsupported
}

func TryDescriptor(*os.File) (*Lock, error) {
	return nil, ErrUnsupported
}

func Shared(string) (*Lock, error) {
	return nil, ErrUnsupported
}

func (*Lock) Release() error {
	return nil
}
