//go:build !unix

package hostfs

import "os"

func validateLinkCount(os.FileInfo) error {
	return nil
}
