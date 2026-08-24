package target

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"

	"go.temporal.io/server/tools/gomad3/target/internal/livecap"
)

var linkedCapabilityCapacityDiagnostic = regexp.MustCompile(`live capability ([^\r\n]+?) requires ([0-9]+), maximum is ([0-9]+)`)

type UnsupportedCapabilityCapacityError struct {
	Resource string
	Required uint64
	Maximum  uint64
	Err      error
}

func (err *UnsupportedCapabilityCapacityError) Error() string {
	return fmt.Sprintf("unsupported linked capability capacity: %s requires %d, maximum is %d", err.Resource, err.Required, err.Maximum)
}

func (err *UnsupportedCapabilityCapacityError) Unwrap() error {
	return err.Err
}

func IsUnsupportedCapability(err error) bool {
	var finding *UnsupportedCapabilityError
	var capacity *UnsupportedCapabilityCapacityError
	return errors.As(err, &finding) || errors.As(err, &capacity)
}

func linkedCapabilityError(err error) error {
	var capacity *livecap.CapacityError
	if !errors.As(err, &capacity) {
		return err
	}
	return &UnsupportedCapabilityCapacityError{
		Resource: capacity.Resource, Required: capacity.Required, Maximum: capacity.Maximum, Err: err,
	}
}

func linkedCapabilityBuildError(err error, output []byte) error {
	match := linkedCapabilityCapacityDiagnostic.FindSubmatch(output)
	if len(match) == 4 {
		required, requiredErr := strconv.ParseUint(string(match[2]), 10, 64)
		maximum, maximumErr := strconv.ParseUint(string(match[3]), 10, 64)
		if requiredErr == nil && maximumErr == nil {
			return &UnsupportedCapabilityCapacityError{
				Resource: string(match[1]), Required: required, Maximum: maximum,
				Err: fmt.Errorf("%w: %s", err, output),
			}
		}
	}
	return fmt.Errorf("%w: %s", err, output)
}
