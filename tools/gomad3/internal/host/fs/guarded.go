package hostfs

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// OpenRootWritable opens an existing regular, single-link file below root for
// reading and writing, with the same lstat/open identity check as OpenRoot. It
// never creates the file or changes its mode.
func OpenRootWritable(root *os.Root, path string) (*os.File, os.FileInfo, error) {
	if root == nil {
		return nil, nil, errors.New("safe file root is required")
	}
	return openRegular(path, root.Lstat, func(name string) (*os.File, error) {
		return root.OpenFile(name, os.O_RDWR, 0)
	})
}

// OpenRootGrowing opens a regular, single-link file below root read-only like
// OpenRoot, but admits a file that grew between its lstat and open. It suits
// an append-only log read without its writer's lock; a shrunken or replaced
// file is still rejected.
func OpenRootGrowing(root *os.Root, path string) (*os.File, os.FileInfo, error) {
	if root == nil {
		return nil, nil, errors.New("safe file root is required")
	}
	return openRegularSized(path, root.Lstat, root.Open, true)
}

// CreateRootExclusive creates a new regular file below root for reading and
// writing, failing when any object already exists at path. It sets perm on the
// new descriptor, independent of the process umask, and returns the validated
// descriptor metadata. The created file is left in place on a later failure.
func CreateRootExclusive(root *os.Root, path string, perm os.FileMode) (*os.File, os.FileInfo, error) {
	if root == nil {
		return nil, nil, errors.New("safe file root is required")
	}
	file, err := root.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_EXCL, perm)
	if err != nil {
		return nil, nil, err
	}
	if err := file.Chmod(perm); err != nil {
		return nil, nil, errors.Join(err, file.Close())
	}
	info, err := file.Stat()
	if err == nil && (!info.Mode().IsRegular() || info.Mode().Perm() != perm) {
		err = fmt.Errorf("%s is not a new private regular file", filepath.Base(path))
	}
	if err == nil {
		err = validateLinkCount(info)
	}
	if err != nil {
		return nil, nil, errors.Join(err, file.Close())
	}
	return file, info, nil
}

// SingleLink reports an error when info describes a file with more than one
// hard link on hosts that expose link counts.
func SingleLink(info os.FileInfo) error {
	return validateLinkCount(info)
}

// SyncRoot flushes the directory entries of the pinned root directory.
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
