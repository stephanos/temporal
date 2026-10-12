package hostfs

import (
	"errors"
	"fmt"
	"io"
	"math"
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
	return readBounded(file, path, info.Size(), maximum)
}

func readBounded(reader io.Reader, path string, initialSize int64, maximum uint64) ([]byte, error) {
	if initialSize < 0 || uint64(initialSize) > maximum {
		return nil, fmt.Errorf("%s exceeds its size bound", path)
	}
	if maximum >= math.MaxInt64 || uint64(initialSize) > uint64(int(^uint(0)>>1)) {
		return nil, fmt.Errorf("%s has an unsupported size bound", path)
	}
	data := make([]byte, initialSize)
	n, err := io.ReadFull(reader, data)
	if err == io.EOF || err == io.ErrUnexpectedEOF {
		return data[:n], nil
	}
	if err != nil {
		return nil, err
	}
	remainder, err := io.ReadAll(io.LimitReader(reader, int64(maximum)-initialSize+1))
	if err != nil {
		return nil, err
	}
	if uint64(len(remainder))+uint64(len(data)) > maximum {
		return nil, fmt.Errorf("%s exceeds its size bound", path)
	}
	return append(data, remainder...), nil
}
