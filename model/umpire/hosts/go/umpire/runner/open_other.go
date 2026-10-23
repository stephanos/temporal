//go:build !unix

package runner

import "os"

// openExisting opens a name inspected as a regular file; the check after opening refuses a
// swapped name.
func openExisting(path string) (*os.File, error) {
	return os.Open(path)
}
