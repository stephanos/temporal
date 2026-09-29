package set

import (
	"errors"
	"fmt"
	"syscall"
)

// DefaultMinimumFreeBytes is the free space a set run keeps on the artifact
// volume: a seed's retained Campaigns and the go build's temporary link
// output fit under it, so a run stops before it fills the disk for every
// other user of the machine.
const DefaultMinimumFreeBytes = 2 << 30

var ErrLowFreeSpace = errors.New("qualification set stopped: free space on the artifact volume fell below its bound")

// freeBytes reports the space available to this user on path's volume.
func freeBytes(path string) (uint64, error) {
	var stat syscall.Statfs_t
	if err := syscall.Statfs(path, &stat); err != nil {
		return 0, fmt.Errorf("inspect free space of %s: %w", path, err)
	}
	return uint64(stat.Bavail) * uint64(stat.Bsize), nil
}
