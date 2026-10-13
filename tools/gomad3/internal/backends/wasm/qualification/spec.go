package qualification

import (
	"encoding/json"
	"errors"
	"strings"

	hostfs "go.temporal.io/server/tools/gomad3/internal/host/fs"
	"go.temporal.io/server/tools/gomad3/internal/model/strictjson"
)

func ReadSpec(path string) (Spec, error) {
	encoded, err := hostfs.ReadBounded(path, 16<<20)
	if err != nil {
		return Spec{}, err
	}
	if err := checkQualificationJSON(encoded); err != nil {
		return Spec{}, err
	}
	var object map[string]json.RawMessage
	if err := strictjson.Decode(encoded, &object); err != nil {
		return Spec{}, err
	}
	for name := range object {
		if strings.EqualFold(name, "Options") {
			return Spec{}, errors.New("qualification spec Options must be absent; installation owns the pinned backend")
		}
	}
	var spec Spec
	if err := strictjson.Decode(encoded, &spec); err != nil {
		return Spec{}, err
	}
	return spec, nil
}

func strictDecode(encoded []byte, destination any) error {
	if err := checkQualificationJSON(encoded); err != nil {
		return err
	}
	return strictjson.Decode(encoded, destination)
}

func checkQualificationJSON(encoded []byte) error {
	err := strictjson.CheckSyntax(encoded, 128)
	if errors.Is(err, strictjson.ErrNestingLimit) {
		return errors.New("qualification JSON nesting limit")
	}
	if errors.Is(err, strictjson.ErrTrailingData) {
		return errors.New("qualification JSON has trailing data")
	}
	return err
}
