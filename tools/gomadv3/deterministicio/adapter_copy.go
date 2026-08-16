package deterministicio

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"go.temporal.io/server/tools/gomadv3/internal/hostfs"
	"go.temporal.io/server/tools/gomadv3/target"
)

const (
	maximumModuleFiles = 5000
	maximumModuleBytes = 512 << 20
)

type AdapterCapacityError = target.AdapterCapacityError

type adapterCopyLimits struct {
	Files int
	Bytes int64
}

var defaultAdapterCopyLimits = adapterCopyLimits{Files: maximumModuleFiles, Bytes: maximumModuleBytes}

func copyAdapterModule(source, destination string, replacements map[string][]byte, limits adapterCopyLimits) error {
	if limits.Files <= 0 || limits.Bytes <= 0 {
		return errors.New("adapter copy limits must be positive")
	}
	if err := os.Mkdir(destination, 0o700); err != nil {
		return fmt.Errorf("create deterministic I/O adapter module: %w", err)
	}
	pending := make(map[string][]byte, len(replacements))
	for relative, contents := range replacements {
		clean, err := validAdapterRelativePath(relative)
		if err != nil {
			return err
		}
		pending[clean] = contents
	}
	files := 0
	bytesCopied := int64(0)
	write := func(relative string, contents []byte) error {
		files++
		if files > limits.Files {
			return &AdapterCapacityError{Resource: "files", Limit: uint64(limits.Files)}
		}
		bytesCopied += int64(len(contents))
		if bytesCopied > limits.Bytes {
			return &AdapterCapacityError{Resource: "bytes", Limit: uint64(limits.Bytes)}
		}
		targetPath := filepath.Join(destination, relative)
		if err := os.MkdirAll(filepath.Dir(targetPath), 0o700); err != nil {
			return fmt.Errorf("create deterministic I/O adapter directory: %w", err)
		}
		return writeExclusive(targetPath, contents)
	}
	err := filepath.WalkDir(source, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		relative, err := filepath.Rel(source, path)
		if err != nil {
			return errors.New("invalid adapter module path")
		}
		if relative == "." {
			return nil
		}
		relative, err = validAdapterRelativePath(relative)
		if err != nil {
			return err
		}
		targetPath := filepath.Join(destination, relative)
		if entry.IsDir() {
			return os.Mkdir(targetPath, 0o700)
		}
		if files >= limits.Files {
			return &AdapterCapacityError{Resource: "files", Limit: uint64(limits.Files)}
		}
		info, err := entry.Info()
		if err != nil || !info.Mode().IsRegular() {
			return errors.New("adapter module contains a non-regular file")
		}
		contents, replaced := pending[relative]
		if replaced {
			delete(pending, relative)
		} else {
			contents, err = readAdapterFile(path, limits.Bytes-bytesCopied)
			if err != nil {
				return err
			}
		}
		return write(relative, contents)
	})
	if err != nil {
		return fmt.Errorf("copy deterministic I/O adapter module: %w", err)
	}
	additions := make([]string, 0, len(pending))
	for relative := range pending {
		additions = append(additions, relative)
	}
	sort.Strings(additions)
	for _, relative := range additions {
		if err := write(relative, pending[relative]); err != nil {
			return fmt.Errorf("copy deterministic I/O adapter module: %w", err)
		}
	}
	return nil
}

func readAdapterFile(path string, maximum int64) (_ []byte, retErr error) {
	file, info, err := hostfs.OpenPath(path)
	if err != nil {
		return nil, err
	}
	defer func() { retErr = errors.Join(retErr, file.Close()) }()
	if info.Size() < 0 || info.Size() > maximum {
		return nil, &AdapterCapacityError{Resource: "bytes", Limit: uint64(maximum)}
	}
	contents, err := io.ReadAll(io.LimitReader(file, maximum+1))
	if err != nil {
		return nil, err
	}
	if int64(len(contents)) > maximum {
		return nil, &AdapterCapacityError{Resource: "bytes", Limit: uint64(maximum)}
	}
	return contents, nil
}

func validAdapterRelativePath(relative string) (string, error) {
	clean := filepath.Clean(relative)
	if relative == "" || filepath.IsAbs(relative) || clean == "." || clean != relative || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("invalid adapter module path %q", relative)
	}
	return clean, nil
}

func writeExclusive(path string, contents []byte) error {
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o400)
	if err != nil {
		return fmt.Errorf("create deterministic I/O adapter file: %w", err)
	}
	if _, err := file.Write(contents); err != nil {
		return errors.Join(fmt.Errorf("write deterministic I/O adapter file: %w", err), file.Close())
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("close deterministic I/O adapter file: %w", err)
	}
	return nil
}

func digestBytes(contents []byte) string {
	digest := sha256.Sum256(contents)
	return "sha256:" + hex.EncodeToString(digest[:])
}
