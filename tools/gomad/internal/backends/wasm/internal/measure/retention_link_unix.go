//go:build unix

package measure

import (
	"errors"
	"os"

	"golang.org/x/sys/unix"
)

func linkRetentionPin(source, destination *os.Root, name string) (retErr error) {
	from, err := source.Open(".")
	if err != nil {
		return err
	}
	defer func() { retErr = errors.Join(retErr, from.Close()) }()
	to, err := destination.Open(".")
	if err != nil {
		return err
	}
	defer func() { retErr = errors.Join(retErr, to.Close()) }()
	return unix.Linkat(int(from.Fd()), ".pin", int(to.Fd()), name, 0)
}
