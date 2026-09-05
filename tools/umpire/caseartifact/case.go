// Package caseartifact decodes and canonically packs Lean-produced Case artifacts.
package caseartifact

import (
	"errors"

	umpirespb "go.temporal.io/server/api/umpire/v1"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

func DecodeProtoJSON(encoded []byte) (*umpirespb.Case, error) {
	if len(encoded) == 0 {
		return nil, errors.New("case ProtoJSON is required")
	}
	decoded := new(umpirespb.Case)
	if err := (protojson.UnmarshalOptions{DiscardUnknown: false}).Unmarshal(encoded, decoded); err != nil {
		return nil, err
	}
	return decoded, nil
}

func Pack(encoded []byte) ([]byte, error) {
	decoded, err := DecodeProtoJSON(encoded)
	if err != nil {
		return nil, err
	}
	return (proto.MarshalOptions{Deterministic: true}).Marshal(decoded)
}
