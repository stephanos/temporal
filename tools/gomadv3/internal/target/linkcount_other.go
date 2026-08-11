//go:build !unix

package target

import "os"

func validateLinkCount(os.FileInfo) error {
	return nil
}
