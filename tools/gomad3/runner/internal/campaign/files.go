package campaign

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"go.temporal.io/server/tools/gomad3/internal/hostfs"
	"go.temporal.io/server/tools/gomad3/record"
)

const maximumManifestBytes = 16 << 20

func readValidatedFile(root *os.Root, path string, mode os.FileMode, maximum uint64) ([]byte, error) {
	file, info, err := hostfs.OpenRoot(root, path)
	if err != nil {
		return nil, err
	}
	if info.Mode().Perm() != mode {
		return nil, errors.Join(fmt.Errorf("%s mode is %#o, want %#o", filepath.Base(path), info.Mode().Perm(), mode), file.Close())
	}
	if info.Size() < 0 || uint64(info.Size()) > maximum {
		return nil, errors.Join(fmt.Errorf("%s size exceeds its bound", filepath.Base(path)), file.Close())
	}
	reader := &io.LimitedReader{R: file, N: int64(maximum)}
	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, errors.Join(err, file.Close())
	}
	if reader.N == 0 {
		var extra [1]byte
		if count, readErr := file.Read(extra[:]); count != 0 || readErr != io.EOF {
			return nil, errors.Join(fmt.Errorf("%s exceeds its bound", filepath.Base(path)), file.Close())
		}
	}
	return data, file.Close()
}

func hashValidatedFile(root *os.Root, path string, mode os.FileMode, expectedSize uint64) (record.SHA256, uint64, error) {
	file, info, err := hostfs.OpenRoot(root, path)
	if err != nil {
		return "", 0, err
	}
	if info.Mode().Perm() != mode || info.Size() < 0 || uint64(info.Size()) != expectedSize {
		return "", 0, errors.Join(fmt.Errorf("%s metadata does not match its manifest", filepath.Base(path)), file.Close())
	}
	hasher := sha256.New()
	reader := &io.LimitedReader{R: file, N: int64(info.Size())}
	size, err := io.Copy(hasher, reader)
	if err != nil {
		return "", 0, errors.Join(err, file.Close())
	}
	var extra [1]byte
	if count, readErr := file.Read(extra[:]); count != 0 || readErr != io.EOF {
		return "", 0, errors.Join(fmt.Errorf("%s changed size while hashing", filepath.Base(path)), file.Close())
	}
	return record.SHA256("sha256:" + hex.EncodeToString(hasher.Sum(nil))), uint64(size), file.Close()
}

func syncDirectoryContext(ctx context.Context, path string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := observeMutation(ctx, mutationDirectorySync, "directory"); err != nil {
		return err
	}
	if err := hostfs.SyncDirectory(path); err != nil {
		return err
	}
	return ctx.Err()
}

func atomicWriteContext(ctx context.Context, path string, data []byte) (retErr error) {
	if err := ctx.Err(); err != nil {
		return err
	}
	directory := filepath.Dir(path)
	if err := observeMutation(ctx, mutationCreate, "atomic-temporary"); err != nil {
		return err
	}
	temporary, err := os.CreateTemp(directory, ".tmp-")
	if err != nil {
		return err
	}
	temporaryPath := temporary.Name()
	keep := true
	defer func() {
		if keep {
			if removeErr := observeMutation(ctx, mutationDelete, "atomic-temporary"); removeErr != nil {
				retErr = errors.Join(retErr, removeErr)
			} else if removeErr := os.Remove(temporaryPath); removeErr != nil && !os.IsNotExist(removeErr) {
				retErr = errors.Join(retErr, removeErr)
			}
		}
	}()
	if err := temporary.Chmod(0o600); err != nil {
		return errors.Join(err, temporary.Close())
	}
	if _, err := temporary.Write(data); err != nil {
		return errors.Join(err, temporary.Close())
	}
	if err := syncFileContext(ctx, temporary, "atomic-temporary"); err != nil {
		return errors.Join(err, temporary.Close())
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := renameContext(ctx, temporaryPath, path, "atomic-publish"); err != nil {
		return err
	}
	keep = false
	return syncDirectoryContext(ctx, directory)
}

func removeCompletedPartialContext(ctx context.Context, path, operation string) error {
	if path == "" {
		return nil
	}
	if err := observeMutation(ctx, mutationDelete, operation); err != nil {
		return err
	}
	if err := os.RemoveAll(path); err != nil {
		return fmt.Errorf("remove completed partial %s: %w", filepath.Base(path), err)
	}
	return nil
}

func makePrivateDirectoriesContext(ctx context.Context, path string) error {
	path = filepath.Clean(path)
	missing := []string{}
	current := path
	for {
		info, err := os.Lstat(current)
		if err == nil {
			if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
				return fmt.Errorf("%s is not a directory", current)
			}
			break
		}
		if !os.IsNotExist(err) {
			return err
		}
		missing = append(missing, current)
		parent := filepath.Dir(current)
		if parent == current {
			return fmt.Errorf("no existing parent for %s", path)
		}
		current = parent
	}
	for index := len(missing) - 1; index >= 0; index-- {
		if err := observeMutation(ctx, mutationCreate, "private-directory"); err != nil {
			return err
		}
		if err := os.Mkdir(missing[index], 0o700); err != nil {
			return err
		}
		if err := os.Chmod(missing[index], 0o700); err != nil {
			return err
		}
	}
	return os.Chmod(path, 0o700)
}
