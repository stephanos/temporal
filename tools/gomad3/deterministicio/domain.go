package deterministicio

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"unicode/utf8"
)

type Digest string

func (identity Digest) Bytes() ([sha256.Size]byte, error) {
	var decoded [sha256.Size]byte
	const prefix = "sha256:"
	value := string(identity)
	if len(value) != len(prefix)+hex.EncodedLen(len(decoded)) || value[:len(prefix)] != prefix {
		return decoded, fmt.Errorf("invalid SHA-256 %q", value)
	}
	hexValue := value[len(prefix):]
	if _, err := hex.Decode(decoded[:], []byte(hexValue)); err != nil || hex.EncodeToString(decoded[:]) != hexValue {
		return [sha256.Size]byte{}, fmt.Errorf("invalid SHA-256 %q", value)
	}
	return decoded, nil
}

func hashBytes(data []byte) Digest {
	digest := sha256.Sum256(data)
	return Digest("sha256:" + hex.EncodeToString(digest[:]))
}

type Adapter struct {
	Module  string `json:"module"`
	Sum     string `json:"sum"`
	Version string `json:"version"`
}

func canonicalJSON(value any) ([]byte, error) {
	if err := validateCanonicalStrings(value); err != nil {
		return nil, err
	}
	var output bytes.Buffer
	encoder := json.NewEncoder(&output)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(value); err != nil {
		return nil, fmt.Errorf("encode JSON: %w", err)
	}
	return bytes.TrimSuffix(output.Bytes(), []byte{'\n'}), nil
}

func validateCanonicalStrings(value any) error {
	encoded, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("validate JSON strings: %w", err)
	}
	if !utf8.Valid(encoded) {
		return errors.New("JSON is not valid UTF-8")
	}
	return nil
}
