package record

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"
)

type Uint64String uint64

func (value Uint64String) MarshalJSON() ([]byte, error) {
	return []byte(strconv.Quote(strconv.FormatUint(uint64(value), 10))), nil
}

func (value *Uint64String) UnmarshalJSON(data []byte) error {
	if value == nil {
		return fmt.Errorf("decode decimal string into nil destination")
	}
	if len(data) < 2 || data[0] != '"' || data[len(data)-1] != '"' {
		return fmt.Errorf("decimal integer must be a JSON string")
	}
	var text string
	if err := json.Unmarshal(data, &text); err != nil {
		return fmt.Errorf("decode decimal string: %w", err)
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
	*value = Uint64String(parsed)
	return nil
}

func CanonicalJSON(value any) ([]byte, error) {
	if err := validateStrings(reflect.ValueOf(value), make(map[visit]struct{})); err != nil {
		return nil, err
	}
	var initial bytes.Buffer
	encoder := json.NewEncoder(&initial)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(value); err != nil {
		return nil, fmt.Errorf("encode JSON: %w", err)
	}
	encoded := bytes.TrimSuffix(initial.Bytes(), []byte{'\n'})
	if err := validateJSONStructure(encoded); err != nil {
		return nil, err
	}

	decoder := json.NewDecoder(bytes.NewReader(encoded))
	decoder.UseNumber()
	var decoded any
	if err := decoder.Decode(&decoded); err != nil {
		return nil, fmt.Errorf("decode intermediate JSON: %w", err)
	}
	var canonical bytes.Buffer
	if err := appendCanonical(&canonical, decoded); err != nil {
		return nil, err
	}
	return canonical.Bytes(), nil
}

func StrictDecode(data []byte, destination any) error {
	if !utf8.Valid(data) {
		return fmt.Errorf("JSON is not valid UTF-8")
	}
	if err := validateJSONStructure(data); err != nil {
		return err
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		return fmt.Errorf("decode JSON: %w", err)
	}
	if err := requireEOF(decoder); err != nil {
		return err
	}
	return nil
}

func CanonicalJSONLines(values []any) ([]byte, error) {
	var output bytes.Buffer
	for index, value := range values {
		encoded, err := CanonicalJSON(value)
		if err != nil {
			return nil, fmt.Errorf("encode JSONL entry %d: %w", index, err)
		}
		output.Write(encoded)
		output.WriteByte('\n')
	}
	return output.Bytes(), nil
}

func StrictDecodeJSONLines[T any](data []byte) ([]T, error) {
	if len(data) == 0 {
		return []T{}, nil
	}
	if data[len(data)-1] != '\n' {
		return nil, fmt.Errorf("JSONL is missing its final newline")
	}
	lines := bytes.Split(data[:len(data)-1], []byte{'\n'})
	values := make([]T, 0, len(lines))
	for index, line := range lines {
		if len(line) == 0 {
			return nil, fmt.Errorf("JSONL entry %d is empty", index)
		}
		var value T
		if err := StrictDecode(line, &value); err != nil {
			return nil, fmt.Errorf("decode JSONL entry %d: %w", index, err)
		}
		canonical, err := CanonicalJSON(value)
		if err != nil {
			return nil, fmt.Errorf("canonicalize JSONL entry %d: %w", index, err)
		}
		if !bytes.Equal(line, canonical) {
			return nil, fmt.Errorf("JSONL entry %d is not canonical", index)
		}
		values = append(values, value)
	}
	return values, nil
}

type visit struct {
	typ     reflect.Type
	pointer uintptr
}

func validateStrings(value reflect.Value, visited map[visit]struct{}) error {
	if !value.IsValid() {
		return nil
	}
	switch value.Kind() {
	case reflect.Interface:
		if value.IsNil() {
			return nil
		}
		return validateStrings(value.Elem(), visited)
	case reflect.Pointer:
		if value.IsNil() {
			return nil
		}
		key := visit{typ: value.Type(), pointer: value.Pointer()}
		if _, ok := visited[key]; ok {
			return nil
		}
		visited[key] = struct{}{}
		return validateStrings(value.Elem(), visited)
	case reflect.Map:
		if value.IsNil() {
			return nil
		}
		key := visit{typ: value.Type(), pointer: value.Pointer()}
		if _, ok := visited[key]; ok {
			return nil
		}
		visited[key] = struct{}{}
		iterator := value.MapRange()
		for iterator.Next() {
			if err := validateStrings(iterator.Key(), visited); err != nil {
				return err
			}
			if err := validateStrings(iterator.Value(), visited); err != nil {
				return err
			}
		}
	case reflect.Slice:
		if value.IsNil() {
			return nil
		}
		key := visit{typ: value.Type(), pointer: value.Pointer()}
		if _, ok := visited[key]; ok {
			return nil
		}
		visited[key] = struct{}{}
		for index := 0; index < value.Len(); index++ {
			if err := validateStrings(value.Index(index), visited); err != nil {
				return err
			}
		}
	case reflect.Array:
		for index := 0; index < value.Len(); index++ {
			if err := validateStrings(value.Index(index), visited); err != nil {
				return err
			}
		}
	case reflect.Struct:
		for index := 0; index < value.NumField(); index++ {
			if err := validateStrings(value.Field(index), visited); err != nil {
				return err
			}
		}
	case reflect.String:
		if !utf8.ValidString(value.String()) {
			return fmt.Errorf("JSON string is not valid UTF-8")
		}
	}
	return nil
}

func validateJSONStructure(data []byte) error {
	if !utf8.Valid(data) {
		return fmt.Errorf("JSON is not valid UTF-8")
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	if err := consumeJSONValue(decoder); err != nil {
		return err
	}
	return requireEOF(decoder)
}

func consumeJSONValue(decoder *json.Decoder) error {
	token, err := decoder.Token()
	if err != nil {
		return fmt.Errorf("decode JSON token: %w", err)
	}
	delimiter, isDelimiter := token.(json.Delim)
	if !isDelimiter {
		return nil
	}
	switch delimiter {
	case '{':
		seen := make(map[string]struct{})
		for decoder.More() {
			keyToken, err := decoder.Token()
			if err != nil {
				return fmt.Errorf("decode JSON object key: %w", err)
			}
			key, ok := keyToken.(string)
			if !ok {
				return fmt.Errorf("JSON object key is not a string")
			}
			if _, ok := seen[key]; ok {
				return fmt.Errorf("duplicate JSON object key %q", key)
			}
			seen[key] = struct{}{}
			if err := consumeJSONValue(decoder); err != nil {
				return err
			}
		}
		closing, err := decoder.Token()
		if err != nil || closing != json.Delim('}') {
			return fmt.Errorf("decode JSON object closing delimiter: %w", err)
		}
	case '[':
		for decoder.More() {
			if err := consumeJSONValue(decoder); err != nil {
				return err
			}
		}
		closing, err := decoder.Token()
		if err != nil || closing != json.Delim(']') {
			return fmt.Errorf("decode JSON array closing delimiter: %w", err)
		}
	default:
		return fmt.Errorf("unexpected JSON delimiter %q", delimiter)
	}
	return nil
}

func requireEOF(decoder *json.Decoder) error {
	if token, err := decoder.Token(); err != io.EOF {
		if err != nil {
			return fmt.Errorf("decode trailing JSON token: %w", err)
		}
		return fmt.Errorf("unexpected trailing JSON token %v", token)
	}
	return nil
}

func appendCanonical(output *bytes.Buffer, value any) error {
	switch typed := value.(type) {
	case nil:
		output.WriteString("null")
	case bool:
		if typed {
			output.WriteString("true")
		} else {
			output.WriteString("false")
		}
	case string:
		return appendJSONString(output, typed)
	case json.Number:
		number := string(typed)
		if strings.ContainsAny(number, ".eE") {
			return fmt.Errorf("floating-point JSON values are forbidden")
		}
		if _, err := strconv.ParseInt(number, 10, 64); err != nil {
			if _, unsignedErr := strconv.ParseUint(number, 10, 64); unsignedErr != nil {
				return fmt.Errorf("noncanonical JSON integer %q", number)
			}
		}
		output.WriteString(number)
	case []any:
		output.WriteByte('[')
		for index, item := range typed {
			if index > 0 {
				output.WriteByte(',')
			}
			if err := appendCanonical(output, item); err != nil {
				return err
			}
		}
		output.WriteByte(']')
	case map[string]any:
		keys := make([]string, 0, len(typed))
		for key := range typed {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		output.WriteByte('{')
		for index, key := range keys {
			if index > 0 {
				output.WriteByte(',')
			}
			if err := appendJSONString(output, key); err != nil {
				return err
			}
			output.WriteByte(':')
			if err := appendCanonical(output, typed[key]); err != nil {
				return err
			}
		}
		output.WriteByte('}')
	default:
		return fmt.Errorf("unsupported canonical JSON value %T", value)
	}
	return nil
}

func appendJSONString(output *bytes.Buffer, value string) error {
	if !utf8.ValidString(value) {
		return fmt.Errorf("JSON string is not valid UTF-8")
	}
	var encoded bytes.Buffer
	encoder := json.NewEncoder(&encoded)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(value); err != nil {
		return fmt.Errorf("encode JSON string: %w", err)
	}
	output.Write(bytes.TrimSuffix(encoded.Bytes(), []byte{'\n'}))
	return nil
}
