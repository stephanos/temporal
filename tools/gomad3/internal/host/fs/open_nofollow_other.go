//go:build !unix

package hostfs

import "os"

func openNoFollow(path string) (*os.File, error) {
	return os.Open(path)
}
