//go:build !unix

package cli

import "os"

// openExisting opens a name that was inspected as a regular file; without O_NOFOLLOW the regular
// file check after opening is what refuses a swapped name.
func openExisting(path string) (*os.File, error) {
	return os.Open(path)
}
