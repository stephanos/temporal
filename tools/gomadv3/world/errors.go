package world

import (
	"errors"
	"fmt"
)

var (
	ErrInvalidConfig    = errors.New("invalid World config")
	ErrInvalidRequest   = errors.New("invalid World request")
	ErrUnknownRequest   = errors.New("unknown World request")
	ErrRequestState     = errors.New("invalid World request state")
	ErrTimeRegression   = errors.New("World time regression")
	ErrCapacity         = errors.New("World capacity exhausted")
	ErrInvalidSnapshot  = errors.New("invalid World snapshot")
	ErrReplayDivergence = errors.New("World replay divergence")
)

type CapacityError struct {
	Dimension string
	Limit     uint64
	Used      uint64
	Delta     uint64
}

func (err *CapacityError) Error() string {
	return fmt.Sprintf("%v: %s limit=%d used=%d delta=%d", ErrCapacity, err.Dimension, err.Limit, err.Used, err.Delta)
}

func (err *CapacityError) Unwrap() error {
	return ErrCapacity
}

type ReplayDivergenceError struct {
	Index          uint64
	ExpectedKind   string
	ActualKind     string
	Field          string
	ExpectedDigest Digest
	ActualDigest   Digest
}

func (err *ReplayDivergenceError) Error() string {
	return fmt.Sprintf("%v: transition=%d field=%s expected-kind=%s actual-kind=%s expected-digest=%s actual-digest=%s", ErrReplayDivergence, err.Index, err.Field, err.ExpectedKind, err.ActualKind, err.ExpectedDigest, err.ActualDigest)
}

func (err *ReplayDivergenceError) Unwrap() error {
	return ErrReplayDivergence
}
