//go:build !aix && !darwin && !dragonfly && !freebsd && !linux && !netbsd && !openbsd && !solaris

package filelock

type Lock struct{}

func Try(string) (*Lock, error) {
	return nil, ErrUnsupported
}

func (*Lock) Release() error {
	return nil
}
