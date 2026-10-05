// Package sourceinventory owns the bounded source-inventory digest that binds
// an adapter module's source tree. Target capability review verifies adapter
// replacements by this digest and deterministic I/O adapter preparation
// records it, so both consume this one implementation.
package sourceinventory

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"

	"go.temporal.io/server/tools/gomad3/internal/hostfs"
)

const (
	maximumFiles = 5000
	maximumBytes = uint64(512 << 20)
)

// CapacityError reports an inventory that exceeds its file-count or byte
// limit. Consumers map it to their own adapter capacity error.
type CapacityError struct {
	Resource string
	Limit    uint64
}

func (err *CapacityError) Error() string {
	return fmt.Sprintf("adapter module exceeds %s limit %d", err.Resource, err.Limit)
}

// Digest returns the source inventory of the regular files under root, bounded
// to 5000 files and 512 MiB.
func Digest(root string) (string, error) {
	return digest(root, maximumFiles, maximumBytes)
}

func digest(root string, maximumFiles int, maximumBytes uint64) (string, error) {
	if maximumFiles <= 0 || maximumBytes == 0 {
		return "", errors.New("adapter source inventory limits must be positive")
	}
	hasher := sha256.New()
	_, _ = hasher.Write([]byte("gomad3.adapter-source-inventory/v1\x00"))
	files := 0
	total := uint64(0)
	err := filepath.WalkDir(root, func(filePath string, entry fs.DirEntry, visitErr error) error {
		if visitErr != nil {
			return visitErr
		}
		if entry.IsDir() {
			return nil
		}
		if entry.Type()&fs.ModeSymlink != 0 {
			return errors.New("adapter source inventory contains a symbolic link")
		}
		info, err := entry.Info()
		if err != nil || !info.Mode().IsRegular() || info.Size() < 0 {
			return errors.New("adapter source inventory contains a non-regular file")
		}
		files++
		if files > maximumFiles {
			return &CapacityError{Resource: "files", Limit: uint64(maximumFiles)}
		}
		size := uint64(info.Size())
		if size > maximumBytes-total {
			return &CapacityError{Resource: "bytes", Limit: maximumBytes}
		}
		contents, err := hostfs.ReadBounded(filePath, maximumBytes-total)
		if err != nil {
			return err
		}
		total += uint64(len(contents))
		relative, err := filepath.Rel(root, filePath)
		if err != nil || relative == "." || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
			return errors.New("adapter source inventory path is invalid")
		}
		digest := sha256.Sum256(contents)
		_, _ = hasher.Write([]byte(filepath.ToSlash(relative)))
		_, _ = hasher.Write([]byte{0})
		_, _ = fmt.Fprintf(hasher, "sha256:%x", digest)
		_, _ = hasher.Write([]byte{0})
		return nil
	})
	if err != nil {
		return "", err
	}
	if files == 0 {
		return "", errors.New("adapter source inventory is empty")
	}
	return fmt.Sprintf("sha256:%x", hasher.Sum(nil)), nil
}
