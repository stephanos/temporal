//go:build !unix

package measure

import (
	"errors"
	"os"
)

func linkRetentionPin(*os.Root, *os.Root, string) error {
	return errors.New("measurement retention requires descriptor-relative hardlinks")
}
