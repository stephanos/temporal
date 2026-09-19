package artifactv2

import (
	"bytes"
	"encoding/json"
	"errors"
	"strconv"
	"strings"
)

// Natural is an arbitrary-precision canonical JSON natural matching Lean's Nat wire domain.
type Natural string

func NaturalFromUint64(value uint64) Natural {
	return Natural(strconv.FormatUint(value, 10))
}

func (value Natural) String() string {
	return string(value)
}

func (value Natural) IsZero() bool {
	return value == "0"
}

func (value Natural) MarshalJSON() ([]byte, error) {
	if err := validateNaturalBytes([]byte(value)); err != nil {
		return nil, err
	}
	return []byte(value), nil
}

func (value *Natural) UnmarshalJSON(encoded []byte) error {
	if value == nil {
		return errors.New("decode natural into nil pointer")
	}
	if err := validateNaturalBytes(encoded); err != nil {
		return err
	}
	*value = Natural(bytes.Clone(encoded))
	return nil
}

func validateNaturalBytes(encoded []byte) error {
	if len(encoded) == 0 {
		return errors.New("natural is empty")
	}
	if len(encoded) > 1 && encoded[0] == '0' {
		return errors.New("natural has a leading zero")
	}
	for _, character := range encoded {
		if character < '0' || character > '9' {
			return errors.New("natural is not canonical base-10 digits")
		}
	}
	return nil
}

func compareNatural(left, right Natural) int {
	if len(left) != len(right) {
		if len(left) < len(right) {
			return -1
		}
		return 1
	}
	return strings.Compare(string(left), string(right))
}

var _ json.Marshaler = Natural("")
var _ json.Unmarshaler = (*Natural)(nil)
