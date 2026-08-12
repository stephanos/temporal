//go:build !unix

package safefile

import "os"

func openNoFollow(path string) (*os.File, error) {
	return os.Open(path)
}
