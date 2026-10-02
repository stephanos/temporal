package model

import (
	"os"

	umpirespb "go.temporal.io/server/api/umpire/v1"
	"google.golang.org/protobuf/encoding/protojson"
)

// Load reads a Model in ProtoJSON, rejects fields the schema does not have, and validates it.
func Load(path string) (*umpirespb.Model, error) {
	encoded, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	m := &umpirespb.Model{}
	if err := (protojson.UnmarshalOptions{}).Unmarshal(encoded, m); err != nil {
		return nil, &Error{Position: path, Message: err.Error()}
	}
	if err := Validate(m); err != nil {
		return nil, err
	}
	return m, nil
}
