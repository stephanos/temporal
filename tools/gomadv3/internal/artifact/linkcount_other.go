//go:build !unix

package artifact

import "os"

func validateLinkCount(os.FileInfo) error {
	return nil
}
