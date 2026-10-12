package hostfs

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
)

func HashExecutable(path string) (digest string, fileSize uint64, retErr error) {
	file, info, err := OpenPath(path)
	if err != nil {
		if errors.Is(err, ErrSymbolicLink) {
			return "", 0, fmt.Errorf("%s is not a regular executable", path)
		}
		return "", 0, err
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil {
			digest, fileSize = "", 0
			if retErr == nil {
				retErr = closeErr
			} else {
				retErr = errors.Join(retErr, closeErr)
			}
		}
	}()
	if !info.Mode().IsRegular() || info.Mode()&0o111 == 0 {
		return "", 0, fmt.Errorf("%s is not a regular executable", path)
	}
	hasher := sha256.New()
	size, err := io.Copy(hasher, file)
	if err != nil {
		return "", 0, err
	}
	if size < 0 {
		return "", 0, fmt.Errorf("negative target size")
	}
	return "sha256:" + hex.EncodeToString(hasher.Sum(nil)), uint64(size), nil
}
