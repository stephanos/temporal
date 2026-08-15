//go:build !unix

package deterministicio

import "os"

func hardLinked(os.FileInfo) bool {
	return false
}
