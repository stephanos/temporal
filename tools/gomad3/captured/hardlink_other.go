//go:build !unix

package captured

import "os"

func hardLinked(os.FileInfo) bool {
	return false
}
