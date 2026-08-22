package provenance

import (
	"errors"
	"fmt"
	"io"
	"os"

	"go.temporal.io/server/tools/gomadv3/internal/canonicaljson"
	"go.temporal.io/server/tools/gomadv3/internal/hostfs"
)

func Store(path string, value any) ([]byte, error) {
	encoded, err := canonicaljson.CanonicalJSON(value)
	if err != nil {
		return nil, fmt.Errorf("encode provenance: %w", err)
	}
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return nil, fmt.Errorf("write provenance: %w", err)
	}
	if err := file.Chmod(0o600); err != nil {
		return nil, errors.Join(fmt.Errorf("set provenance mode: %w", err), file.Close())
	}
	if _, err := file.Write(encoded); err != nil {
		return nil, errors.Join(fmt.Errorf("write provenance: %w", err), file.Close())
	}
	if err := file.Close(); err != nil {
		return nil, fmt.Errorf("close provenance: %w", err)
	}
	return encoded, nil
}

func Load(path string, maximum uint64, destination any) (_ []byte, retErr error) {
	file, info, err := hostfs.OpenPath(path)
	if err != nil {
		if errors.Is(err, hostfs.ErrSymbolicLink) {
			return nil, fmt.Errorf("%s is not a regular file", path)
		}
		return nil, err
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil {
			retErr = errors.Join(retErr, closeErr)
		}
	}()
	if info.Size() < 0 || uint64(info.Size()) > maximum {
		return nil, fmt.Errorf("provenance file size or type is invalid")
	}
	encoded, err := io.ReadAll(io.LimitReader(file, int64(maximum)+1))
	if err != nil {
		return nil, err
	}
	if uint64(len(encoded)) > maximum {
		return nil, fmt.Errorf("provenance file exceeds %d bytes", maximum)
	}
	if err := canonicaljson.DecodeCanonicalJSON(encoded, destination); err != nil {
		return nil, err
	}
	return encoded, nil
}
