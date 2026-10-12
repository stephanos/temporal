package target

import (
	"errors"
	"fmt"
)

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
