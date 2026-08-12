package safefile

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

var ErrSymbolicLink = errors.New("symbolic link")

func OpenPath(path string) (*os.File, os.FileInfo, error) {
	return openRegular(path, os.Lstat, openNoFollow)
}

func OpenRoot(root *os.Root, path string) (*os.File, os.FileInfo, error) {
	if root == nil {
		return nil, nil, errors.New("safe file root is required")
	}
	return openRegular(path, root.Lstat, root.Open)
}

func openRegular(path string, lstat func(string) (os.FileInfo, error), open func(string) (*os.File, error)) (*os.File, os.FileInfo, error) {
	info, err := lstat(path)
	if err != nil {
		return nil, nil, err
	}
	name := filepath.Base(path)
	if info.Mode()&os.ModeSymlink != 0 {
		return nil, nil, fmt.Errorf("%s is a %w", name, ErrSymbolicLink)
	}
	if !info.Mode().IsRegular() {
		return nil, nil, fmt.Errorf("%s is not a regular file", name)
	}
	if err := validateLinkCount(info); err != nil {
		return nil, nil, err
	}
	file, err := open(path)
	if err != nil {
		return nil, nil, err
	}
	openedInfo, err := file.Stat()
	if err != nil || !os.SameFile(info, openedInfo) || openedInfo.Mode() != info.Mode() || openedInfo.Size() != info.Size() {
		return nil, nil, errors.Join(fmt.Errorf("%s changed while opening", name), err, file.Close())
	}
	if err := validateLinkCount(openedInfo); err != nil {
		return nil, nil, errors.Join(err, file.Close())
	}
	return file, openedInfo, nil
}
