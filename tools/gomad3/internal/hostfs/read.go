package hostfs

import (
	"errors"
	"fmt"
	"io"
)

// ReadBounded reads the regular file at path, refusing a symbolic link and a
// file larger than maximum bytes.
func ReadBounded(path string, maximum uint64) (_ []byte, retErr error) {
	file, info, err := OpenPath(path)
	if err != nil {
		if errors.Is(err, ErrSymbolicLink) {
			return nil, fmt.Errorf("%s is not a regular file", path)
		}
		return nil, err
	}
	defer func() { retErr = errors.Join(retErr, file.Close()) }()
	if info.Size() < 0 || uint64(info.Size()) > maximum {
		return nil, fmt.Errorf("%s exceeds its size bound", path)
	}
	data, err := io.ReadAll(io.LimitReader(file, int64(maximum)+1))
	if err != nil {
		return nil, err
	}
	if uint64(len(data)) > maximum {
		return nil, fmt.Errorf("%s exceeds its size bound", path)
	}
	return data, nil
}
