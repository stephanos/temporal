package hostfs

import (
	"errors"
	"os"
)

func SyncRoot(root *os.Root) (retErr error) {
	if root == nil {
		return errors.New("safe file root is required")
	}
	directory, err := root.Open(".")
	if err != nil {
		return err
	}
	defer func() { retErr = errors.Join(retErr, directory.Close()) }()
	return directory.Sync()
}

// SyncDirectory flushes the directory entries of the directory at path.
func SyncDirectory(path string) (retErr error) {
	directory, err := os.Open(path)
	if err != nil {
		return err
	}
	defer func() { retErr = errors.Join(retErr, directory.Close()) }()
	return directory.Sync()
}
