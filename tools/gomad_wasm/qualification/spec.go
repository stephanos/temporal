package qualification

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	"go.temporal.io/server/tools/gomad3/hostfs"
)

func ReadSpec(path string) (Spec, error) {
	encoded, err := hostfs.ReadBounded(path, 16<<20)
	if err != nil {
		return Spec{}, err
	}
	if err := uniqueJSON(encoded); err != nil {
		return Spec{}, err
	}
	var object map[string]json.RawMessage
	if err := json.Unmarshal(encoded, &object); err != nil {
		return Spec{}, err
	}
	for name := range object {
		if strings.EqualFold(name, "Options") {
			return Spec{}, errors.New("qualification spec Options must be absent; installation owns the pinned backend")
		}
	}
	var spec Spec
	if err := strictDecode(encoded, &spec); err != nil {
		return Spec{}, err
	}
	return spec, nil
}

func strictDecode(encoded []byte, destination any) error {
	if err := uniqueJSON(encoded); err != nil {
		return err
	}
	decoder := json.NewDecoder(bytes.NewReader(encoded))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		return err
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		return errors.New("qualification JSON has trailing data")
	}
	return nil
}

// Names are case-folded because encoding/json matches Go fields that way.
func uniqueJSON(encoded []byte) error {
	decoder := json.NewDecoder(bytes.NewReader(encoded))
	var value func(int) error
	value = func(depth int) error {
		if depth > 128 {
			return errors.New("qualification JSON nesting limit")
		}
		token, err := decoder.Token()
		if err != nil {
			return err
		}
		delimiter, ok := token.(json.Delim)
		if !ok {
			return nil
		}
		switch delimiter {
		case '{':
			seen := map[string]bool{}
			for decoder.More() {
				name, err := decoder.Token()
				if err != nil {
					return err
				}
				key, ok := name.(string)
				if !ok {
					return errors.New("qualification JSON object key is not a string")
				}
				key = strings.ToLower(key)
				if seen[key] {
					return fmt.Errorf("duplicate qualification JSON key %q", key)
				}
				seen[key] = true
				if err := value(depth + 1); err != nil {
					return err
				}
			}
		case '[':
			for decoder.More() {
				if err := value(depth + 1); err != nil {
					return err
				}
			}
		default:
			return errors.New("invalid qualification JSON delimiter")
		}
		_, err = decoder.Token()
		return err
	}
	if err := value(0); err != nil {
		return err
	}
	if _, err := decoder.Token(); err != io.EOF {
		return errors.New("qualification JSON has trailing data")
	}
	return nil
}
