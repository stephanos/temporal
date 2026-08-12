//go:build !unix

package safefile

import "os"

func validateLinkCount(os.FileInfo) error {
	return nil
}
