package process

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
)

const (
	configHeaderBytes      = 40
	maximumInitialSnapshot = 64 << 20
	maximumReplayPlan      = 64 << 20
)

var configMagic = [8]byte{'G', 'O', 'M', 'A', 'D', 'W', 'C', 3}

type SessionSpec struct {
	TransitionLimit uint64
	Seed            uint64
	ExpectedInitial []byte
	ReplayPlan      []byte
}

func EncodeSessionSpec(config SessionSpec) ([]byte, error) {
	if config.TransitionLimit == 0 {
		return nil, fmt.Errorf("World transition limit must be positive")
	}
	if len(config.ExpectedInitial) > maximumInitialSnapshot {
		return nil, fmt.Errorf("expected initial World snapshot exceeds its bound")
	}
	if len(config.ReplayPlan) > maximumReplayPlan {
		return nil, errors.New("world replay plan exceeds its bound")
	}
	if len(config.ReplayPlan) != 0 && len(config.ExpectedInitial) == 0 {
		return nil, errors.New("world replay plan requires an expected initial snapshot")
	}
	result := make([]byte, configHeaderBytes+len(config.ExpectedInitial)+len(config.ReplayPlan))
	copy(result[:8], configMagic[:])
	binary.BigEndian.PutUint64(result[8:16], config.TransitionLimit)
	binary.BigEndian.PutUint64(result[16:24], config.Seed)
	binary.BigEndian.PutUint64(result[24:32], uint64(len(config.ExpectedInitial)))
	binary.BigEndian.PutUint64(result[32:40], uint64(len(config.ReplayPlan)))
	offset := configHeaderBytes + len(config.ExpectedInitial)
	copy(result[configHeaderBytes:offset], config.ExpectedInitial)
	copy(result[offset:], config.ReplayPlan)
	return result, nil
}

func readSessionSpec(reader io.Reader) (SessionSpec, error) {
	var encoded [configHeaderBytes]byte
	if _, err := io.ReadFull(reader, encoded[:]); err != nil {
		return SessionSpec{}, err
	}
	if !bytes.Equal(encoded[:8], configMagic[:]) {
		return SessionSpec{}, fmt.Errorf("invalid World child configuration")
	}
	limit := binary.BigEndian.Uint64(encoded[8:])
	if limit == 0 {
		return SessionSpec{}, fmt.Errorf("invalid zero World transition limit")
	}
	initialSize := binary.BigEndian.Uint64(encoded[24:])
	if initialSize > maximumInitialSnapshot {
		return SessionSpec{}, fmt.Errorf("expected initial World snapshot exceeds its bound")
	}
	replayPlanSize := binary.BigEndian.Uint64(encoded[32:])
	if replayPlanSize > maximumReplayPlan {
		return SessionSpec{}, errors.New("world replay plan exceeds its bound")
	}
	if replayPlanSize != 0 && initialSize == 0 {
		return SessionSpec{}, errors.New("world replay plan requires an expected initial snapshot")
	}
	config := SessionSpec{TransitionLimit: limit, Seed: binary.BigEndian.Uint64(encoded[16:24])}
	config.ExpectedInitial = make([]byte, initialSize)
	if _, err := io.ReadFull(reader, config.ExpectedInitial); err != nil {
		return SessionSpec{}, err
	}
	config.ReplayPlan = make([]byte, replayPlanSize)
	if _, err := io.ReadFull(reader, config.ReplayPlan); err != nil {
		return SessionSpec{}, err
	}
	return config, nil
}
