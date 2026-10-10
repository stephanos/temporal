package testpilot

import (
	"errors"

	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/testpilot/casefile"
	"google.golang.org/protobuf/encoding/protojson"
)

func DecodeCaseProtoJSON(encoded []byte) (*testpilotspb.Case, error) {
	if len(encoded) == 0 {
		return nil, errors.New("case ProtoJSON is required")
	}
	versionJSON, err := casefile.JSONVersion(encoded)
	if err != nil {
		return nil, err
	}
	if len(versionJSON) != 0 {
		version := new(testpilotspb.FormatVersion)
		if err := (protojson.UnmarshalOptions{DiscardUnknown: false}).Unmarshal(versionJSON, version); err != nil {
			return nil, err
		}
		if err := casefile.CheckVersion(version.GetMajor(), version.GetMinor()); err != nil {
			return nil, err
		}
	}
	decoded := new(testpilotspb.Case)
	if err := (protojson.UnmarshalOptions{DiscardUnknown: false}).Unmarshal(encoded, decoded); err != nil {
		return nil, err
	}
	if err := casefile.CheckVersion(decoded.GetVersion().GetMajor(), decoded.GetVersion().GetMinor()); err != nil {
		return nil, err
	}
	return decoded, nil
}
