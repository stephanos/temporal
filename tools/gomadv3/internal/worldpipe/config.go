package worldpipe

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
)

const (
	configHeaderBytes      = 32
	maximumInitialSnapshot = 64 << 20
)

var configMagic = [8]byte{'G', 'O', 'M', 'A', 'D', 'W', 'C', 2}

type Config struct {
	TransitionLimit uint64
	Seed            uint64
	ExpectedInitial []byte
}

func Encode(config Config) ([]byte, error) {
	if config.TransitionLimit == 0 {
		return nil, fmt.Errorf("World transition limit must be positive")
	}
	if len(config.ExpectedInitial) > maximumInitialSnapshot {
		return nil, fmt.Errorf("expected initial World snapshot exceeds its bound")
	}
	result := make([]byte, configHeaderBytes+len(config.ExpectedInitial))
	copy(result[:8], configMagic[:])
	binary.BigEndian.PutUint64(result[8:16], config.TransitionLimit)
	binary.BigEndian.PutUint64(result[16:24], config.Seed)
	binary.BigEndian.PutUint64(result[24:32], uint64(len(config.ExpectedInitial)))
	copy(result[configHeaderBytes:], config.ExpectedInitial)
	return result, nil
}

func Read(reader io.Reader) (Config, error) {
	var encoded [configHeaderBytes]byte
	if _, err := io.ReadFull(reader, encoded[:]); err != nil {
		return Config{}, err
	}
	if !bytes.Equal(encoded[:8], configMagic[:]) {
		return Config{}, fmt.Errorf("invalid World child configuration")
	}
	limit := binary.BigEndian.Uint64(encoded[8:])
	if limit == 0 {
		return Config{}, fmt.Errorf("invalid zero World transition limit")
	}
	initialSize := binary.BigEndian.Uint64(encoded[24:])
	if initialSize > maximumInitialSnapshot {
		return Config{}, fmt.Errorf("expected initial World snapshot exceeds its bound")
	}
	config := Config{TransitionLimit: limit, Seed: binary.BigEndian.Uint64(encoded[16:24])}
	config.ExpectedInitial = make([]byte, initialSize)
	if _, err := io.ReadFull(reader, config.ExpectedInitial); err != nil {
		return Config{}, err
	}
	return config, nil
}
