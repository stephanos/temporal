//go:build !unix

package romount

import "os"

func hardLinked(os.FileInfo) bool {
	return false
}
