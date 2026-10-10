package casefile

import (
	"encoding/json"
	"fmt"
)

// CurrentMajor and CurrentMinor are the single admission boundary. The coordinated
// CEL, Duration and evidence migration activates 4.0 only after its consumers land.
const (
	CurrentMajor int32 = 1
	CurrentMinor int32 = 0
	CELMajor     int32 = 4
	CELMinor     int32 = 0
)

// FormatError identifies an unsupported format before its payload is interpreted.
type FormatError struct{ Major, Minor, ExpectedMajor, ExpectedMinor int32 }

func (e *FormatError) Error() string {
	return fmt.Sprintf("unsupported Case version %d.%d; expected %d.%d", e.Major, e.Minor, e.ExpectedMajor, e.ExpectedMinor)
}

// CheckVersion admits exactly the current format, never a compatibility range.
func CheckVersion(major, minor int32) error {
	return checkVersion(major, minor, CurrentMajor, CurrentMinor)
}

func checkVersion(major, minor, expectedMajor, expectedMinor int32) error {
	if major != expectedMajor || minor != expectedMinor {
		return &FormatError{Major: major, Minor: minor, ExpectedMajor: expectedMajor, ExpectedMinor: expectedMinor}
	}
	return nil
}

// JSONVersion extracts the raw version envelope without interpreting the schema
// payload. The owning protobuf decoder decides its exact numeric syntax.
func JSONVersion(encoded []byte) ([]byte, error) {
	var envelope map[string]json.RawMessage
	if err := json.Unmarshal(encoded, &envelope); err != nil {
		return nil, err
	}
	return envelope["version"], nil
}
