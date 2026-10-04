package world

import (
	"fmt"
)

type modelSentinel string

func (err modelSentinel) Error() string { return string(err) }

const (
	invalidConfig           modelSentinel = "invalid World config"
	invalidRequest          modelSentinel = "invalid World request"
	unknownRequest          modelSentinel = "unknown World request"
	invalidRequestState     modelSentinel = "invalid World request state"
	timeRegression          modelSentinel = "World time regression"
	capacityExhausted       modelSentinel = "World capacity exhausted"
	invalidSnapshotSentinel modelSentinel = "invalid World snapshot"
	replayDivergence        modelSentinel = "World replay divergence"
)

var (
	ErrInvalidConfig    error = invalidConfig
	ErrInvalidRequest   error = invalidRequest
	ErrUnknownRequest   error = unknownRequest
	ErrRequestState     error = invalidRequestState
	ErrTimeRegression   error = timeRegression
	ErrCapacity         error = capacityExhausted
	ErrInvalidSnapshot  error = invalidSnapshotSentinel
	ErrReplayDivergence error = replayDivergence
)

type modelError struct {
	detail string
	kind   TerminalKind
	cause  error
}

func (err *modelError) Error() string { return err.detail }
func (err *modelError) Unwrap() error { return err.cause }

func classifiedError(identity modelSentinel, detail string) error {
	terminal, _ := ownedTerminal(identity)
	return &modelError{detail: string(identity) + ": " + detail, kind: terminal.Kind, cause: identity}
}

func modelContextError(prefix string, cause error) error {
	if terminal, ok := ownedTerminal(cause); ok {
		return &modelError{detail: prefix + ": " + terminal.Detail, kind: terminal.Kind, cause: cause}
	}
	return fmt.Errorf("%s: %w", prefix, cause)
}

func ownedTerminal(err error) (Terminal, bool) {
	switch err := err.(type) {
	case modelSentinel:
		kind := TerminalInvalidInput
		switch err {
		case capacityExhausted:
			kind = TerminalCapacity
		case replayDivergence:
			kind = TerminalReplayDivergence
		case invalidConfig, invalidRequest, unknownRequest, invalidRequestState, timeRegression, invalidSnapshotSentinel:
		default:
			return Terminal{}, false
		}
		return Terminal{Kind: kind, Detail: string(err)}, true
	case *CapacityError:
		if err != nil {
			return Terminal{Kind: TerminalCapacity, Detail: err.Error()}, true
		}
	case *ReplayDivergenceError:
		if err != nil {
			return Terminal{Kind: TerminalReplayDivergence, Detail: err.Error()}, true
		}
	case *modelError:
		if err != nil {
			return Terminal{Kind: err.kind, Detail: err.detail}, true
		}
	}
	return Terminal{}, false
}

type CapacityError struct {
	Dimension string
	Limit     uint64
	Used      uint64
	Delta     uint64
}

func (err *CapacityError) Error() string {
	return fmt.Sprintf("%s: %s limit=%d used=%d delta=%d", string(capacityExhausted), err.Dimension, err.Limit, err.Used, err.Delta)
}

func (err *CapacityError) Unwrap() error {
	return capacityExhausted
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
	return fmt.Sprintf("%s: transition=%d field=%s expected-kind=%s actual-kind=%s expected-digest=%s actual-digest=%s", string(replayDivergence), err.Index, err.Field, err.ExpectedKind, err.ActualKind, err.ExpectedDigest, err.ActualDigest)
}

func (err *ReplayDivergenceError) Unwrap() error {
	return replayDivergence
}
