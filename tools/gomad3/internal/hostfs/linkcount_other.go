//go:build !unix

package hostfs

import "os"

// LinkCount reports unavailable link-count metadata on this platform.
func LinkCount(os.FileInfo) (uint64, bool) {
	return 0, false
}

func validateLinkCount(os.FileInfo) error {
	return nil
}
