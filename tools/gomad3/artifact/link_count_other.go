//go:build !unix

package artifact

import "os"

func linkCount(os.FileInfo) (uint64, bool) {
	return 0, false
}
