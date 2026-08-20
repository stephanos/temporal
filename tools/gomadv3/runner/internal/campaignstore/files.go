package campaignstore

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"go.temporal.io/server/tools/gomadv3/evidence"
	"go.temporal.io/server/tools/gomadv3/internal/hostfs"
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

func hashValidatedFile(root *os.Root, path string, mode os.FileMode, expectedSize uint64) (evidence.SHA256, uint64, error) {
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
	return evidence.SHA256("sha256:" + hex.EncodeToString(hasher.Sum(nil))), uint64(size), file.Close()
}

func syncDirectory(path string) (retErr error) {
	directory, err := os.Open(path)
	if err != nil {
		return err
	}
	defer func() {
		retErr = errors.Join(retErr, directory.Close())
	}()
	return directory.Sync()
}

func syncDirectoryContext(ctx context.Context, path string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := observeMutation(ctx, mutationDirectorySync, "directory"); err != nil {
		return err
	}
	if err := syncDirectory(path); err != nil {
		return err
	}
	return ctx.Err()
}
