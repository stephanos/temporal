//go:build !unix

package readonlymount

import "os"

func hardLinked(os.FileInfo) bool {
	return false
}
