package hostfs

import (
	"os"

	internal "go.temporal.io/server/tools/gomad3/internal/hostfs"
)

// ErrSymbolicLink marks a path refused because it is a symbolic link.
var ErrSymbolicLink = internal.ErrSymbolicLink

func ReadBounded(path string, maximum uint64) ([]byte, error) {
	return internal.ReadBounded(path, maximum)
}

// OpenPath opens a single-link regular file without following symbolic links.
func OpenPath(path string) (*os.File, os.FileInfo, error) {
	return internal.OpenPath(path)
}
