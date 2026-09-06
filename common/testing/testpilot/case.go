package testpilot

import (
	"errors"

	testpilotpb "go.temporal.io/server/api/testpilot/v1"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

func DecodeCaseProtoJSON(encoded []byte) (*testpilotpb.Case, error) {
	if len(encoded) == 0 {
		return nil, errors.New("case ProtoJSON is required")
	}
	decoded := new(testpilotpb.Case)
	if err := (protojson.UnmarshalOptions{DiscardUnknown: false}).Unmarshal(encoded, decoded); err != nil {
		return nil, err
	}
	return decoded, nil
}

func PackCaseProtoJSON(encoded []byte) ([]byte, error) {
	decoded, err := DecodeCaseProtoJSON(encoded)
	if err != nil {
		return nil, err
	}
	return (proto.MarshalOptions{Deterministic: true}).Marshal(decoded)
}
