package testpilot

import (
	"errors"
	"fmt"

	"go.temporal.io/server/common/testing/testpilot/internal/ir"
)

// PreparationErrorCategory is a stable classification of a static admission rejection.
type PreparationErrorCategory string

const (
	PreparationMalformed     PreparationErrorCategory = "malformed"
	PreparationUnknown       PreparationErrorCategory = "unknown"
	PreparationTypeMismatch  PreparationErrorCategory = "type_mismatch"
	PreparationUnavailable   PreparationErrorCategory = "unavailable"
	PreparationUnsupported   PreparationErrorCategory = "unsupported"
	PreparationLimitExceeded PreparationErrorCategory = "limit_exceeded"
)

// PreparationError describes a NewCatalog, Prepare, or Profile binding rejection.
// Path uses the admission input vocabulary and is bounded to 256 bytes.
// Detail is human-readable and is not a stable string API.
// ProtoJSON decoding and runtime Run/Driver failures are outside this contract.
type PreparationError struct {
	Category PreparationErrorCategory
	Path     string
	Detail   string
	cause    error
}

func (e *PreparationError) Error() string {
	if e.cause != nil {
		return e.cause.Error()
	}
	return fmt.Sprintf("%s at %s: %s", e.Category, e.Path, e.Detail)
}

// Unwrap retains the original admission error and its message context.
func (e *PreparationError) Unwrap() error { return e.cause }

func preparationError(err error, path string) error {
	var public *PreparationError
	if errors.As(err, &public) {
		return err
	}
	result := &PreparationError{Category: PreparationMalformed, Path: path, Detail: err.Error(), cause: err}
	var internal *ir.Error
	if errors.As(err, &internal) {
		result.Category = PreparationErrorCategory(internal.Category)
		result.Path = internal.Path
		result.Detail = internal.Detail
	}
	if len(result.Path) > 256 {
		result.Path = result.Path[:256]
	}
	return result
}
