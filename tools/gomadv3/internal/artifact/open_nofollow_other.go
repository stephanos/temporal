//go:build !unix

package artifact

import "os"

func openNoFollow(path string) (*os.File, error) {
	return os.Open(path)
}
