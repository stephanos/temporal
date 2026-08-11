//go:build !unix

package target

import "os"

func openNoFollow(path string) (*os.File, error) {
	return os.Open(path)
}
