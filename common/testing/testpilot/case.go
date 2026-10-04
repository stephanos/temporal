package testpilot

import (
	"errors"

	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"google.golang.org/protobuf/encoding/protojson"
)

func DecodeCaseProtoJSON(encoded []byte) (*testpilotspb.Case, error) {
	if len(encoded) == 0 {
		return nil, errors.New("case ProtoJSON is required")
	}
	decoded := new(testpilotspb.Case)
	if err := (protojson.UnmarshalOptions{DiscardUnknown: false}).Unmarshal(encoded, decoded); err != nil {
		return nil, err
	}
	return decoded, nil
}
