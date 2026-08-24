package deterministicio

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"
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

type decimal uint64

func (value decimal) MarshalJSON() ([]byte, error) {
	return []byte(strconv.Quote(strconv.FormatUint(uint64(value), 10))), nil
}

func (value *decimal) UnmarshalJSON(data []byte) error {
	if value == nil {
		return errors.New("decode decimal string into nil destination")
	}
	var text string
	if err := json.Unmarshal(data, &text); err != nil {
		return errors.New("decimal integer must be a JSON string")
	}
	if text == "" || len(text) > 1 && text[0] == '0' {
		return fmt.Errorf("invalid canonical decimal string %q", text)
	}
	for _, character := range text {
		if character < '0' || character > '9' {
			return fmt.Errorf("invalid canonical decimal string %q", text)
		}
	}
	parsed, err := strconv.ParseUint(text, 10, 64)
	if err != nil {
		return fmt.Errorf("decimal string out of range: %w", err)
	}
	*value = decimal(parsed)
	return nil
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

func decodeCanonicalJSON(data []byte, destination any) error {
	if !utf8.Valid(data) {
		return errors.New("JSON is not valid UTF-8")
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		return fmt.Errorf("decode JSON: %w", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); err == nil {
		return errors.New("JSON contains trailing data")
	} else if !errors.Is(err, io.EOF) {
		return fmt.Errorf("decode trailing JSON: %w", err)
	}
	canonical, err := canonicalJSON(destination)
	if err != nil {
		return fmt.Errorf("canonicalize JSON: %w", err)
	}
	if !bytes.Equal(data, canonical) {
		return errors.New("JSON is not canonical")
	}
	return nil
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
