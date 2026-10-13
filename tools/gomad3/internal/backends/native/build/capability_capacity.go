package build

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"

	"go.temporal.io/server/tools/gomad3/internal/backends/native/build/internal/livecap"
	"go.temporal.io/server/tools/gomad3/internal/backends/target"
)

var linkedCapabilityCapacityDiagnostic = regexp.MustCompile(`live capability ([^\r\n]+?) requires ([0-9]+), maximum is ([0-9]+)`)

func linkedCapabilityError(err error) error {
	var capacity *livecap.CapacityError
	if !errors.As(err, &capacity) {
		return err
	}
	return &target.UnsupportedCapabilityCapacityError{
		Resource: capacity.Resource, Required: capacity.Required, Maximum: capacity.Maximum, Err: err,
	}
}

func linkedCapabilityBuildError(err error, output []byte) error {
	match := linkedCapabilityCapacityDiagnostic.FindSubmatch(output)
	if len(match) == 4 {
		required, requiredErr := strconv.ParseUint(string(match[2]), 10, 64)
		maximum, maximumErr := strconv.ParseUint(string(match[3]), 10, 64)
		if requiredErr == nil && maximumErr == nil {
			return &target.UnsupportedCapabilityCapacityError{
				Resource: string(match[1]), Required: required, Maximum: maximum,
				Err: fmt.Errorf("%w: %s", err, output),
			}
		}
	}
	return fmt.Errorf("%w: %s", err, output)
}
