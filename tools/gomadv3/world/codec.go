package world

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"
)

const (
	MaximumSnapshotJSONBytes = 64 << 20
	maximumSnapshotElements  = 1 << 20
	maximumSnapshotNodes     = MaximumSnapshotJSONBytes
)

func EncodeSnapshot(snapshot Snapshot) ([]byte, error) {
	if _, err := Restore(snapshot, nil); err != nil {
		return nil, err
	}
	encoded, err := canonicalJSON(snapshot)
	if err != nil {
		return nil, err
	}
	if len(encoded) > MaximumSnapshotJSONBytes {
		return nil, invalidSnapshot("json.size")
	}
	return encoded, nil
}

func DecodeSnapshot(data []byte) (Snapshot, error) {
	if len(data) > MaximumSnapshotJSONBytes {
		return Snapshot{}, invalidSnapshot("json.size")
	}
	if !utf8.Valid(data) {
		return Snapshot{}, invalidSnapshot("json.utf8")
	}
	if err := preflightSnapshotJSON(data); err != nil {
		return Snapshot{}, invalidSnapshot("json.bounds: " + err.Error())
	}
	if err := validateJSONStructure(data); err != nil {
		return Snapshot{}, invalidSnapshot("json: " + err.Error())
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var snapshot Snapshot
	if err := decoder.Decode(&snapshot); err != nil {
		return Snapshot{}, invalidSnapshot("json: " + err.Error())
	}
	if err := requireJSONEOF(decoder); err != nil {
		return Snapshot{}, invalidSnapshot("json: " + err.Error())
	}
	if _, err := Restore(snapshot, nil); err != nil {
		return Snapshot{}, err
	}
	canonical, err := canonicalJSON(snapshot)
	if err != nil {
		return Snapshot{}, invalidSnapshot("json.canonical")
	}
	if !bytes.Equal(data, canonical) {
		return Snapshot{}, invalidSnapshot("json.canonical")
	}
	return snapshot, nil
}

func preflightSnapshotJSON(data []byte) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	opening, err := decoder.Token()
	if err != nil || opening != json.Delim('{') {
		return fmt.Errorf("invalid top-level object")
	}
	key, err := decoder.Token()
	if err != nil || key != "config" {
		return fmt.Errorf("config must be the first canonical field")
	}
	var config Config
	if err := decoder.Decode(&config); err != nil {
		return fmt.Errorf("config: %w", err)
	}
	if err := validateConfig(config); err != nil {
		return fmt.Errorf("config: %w", err)
	}
	maximumString := uint64(config.Limits.MaxStringBytes)
	encodedPayload := config.Limits.MaxPayloadBytes
	if encodedPayload <= (math.MaxUint64-2)/4*3 {
		encodedPayload = (encodedPayload + 2) / 3 * 4
	} else {
		encodedPayload = math.MaxUint64
	}
	maximumString = max(maximumString, encodedPayload, 64, 20)
	if maximumString <= math.MaxUint64/6 {
		maximumString *= 6
	} else {
		maximumString = math.MaxUint64
	}
	if err := validateJSONStringTokenLengths(data, maximumString); err != nil {
		return err
	}
	elementBudget := snapshotElementBudget(config.Limits)
	nestedArrayLimit := max(config.Limits.MaxRequests, config.Limits.MaxEvents)
	for decoder.More() {
		if err := consumeSnapshotBudget(&elementBudget); err != nil {
			return err
		}
		field, fieldErr := decoder.Token()
		if fieldErr != nil {
			return fieldErr
		}
		name, ok := field.(string)
		if !ok {
			return fmt.Errorf("top-level key is not a string")
		}
		var limit uint64
		switch name {
		case "requests":
			limit = config.Limits.MaxRequests
		case "events":
			limit = config.Limits.MaxEvents
		case "transitions":
			limit = config.Limits.MaxTransitions
		}
		if limit != 0 {
			if err := consumeBoundedJSONArray(decoder, limit, nestedArrayLimit, &elementBudget, 1); err != nil {
				return fmt.Errorf("%s: %w", name, err)
			}
			continue
		}
		if err := consumeBoundedJSONValue(decoder, nestedArrayLimit, &elementBudget, 1); err != nil {
			return err
		}
	}
	closing, err := decoder.Token()
	if err != nil || closing != json.Delim('}') {
		return fmt.Errorf("invalid top-level close")
	}
	return requireJSONEOF(decoder)
}

func snapshotElementBudget(limits Limits) uint64 {
	const (
		fixedSnapshotNodes      = 256
		nodesPerSemanticElement = 128
	)
	elements := cappedAdd(limits.MaxRequests, limits.MaxEvents, maximumSnapshotElements)
	elements = cappedAdd(elements, limits.MaxTransitions, maximumSnapshotElements)
	perTransition := cappedAdd(limits.MaxRequests, limits.MaxEvents, maximumSnapshotElements)
	nested := uint64(maximumSnapshotElements)
	if limits.MaxTransitions <= maximumSnapshotElements && perTransition <= maximumSnapshotElements/limits.MaxTransitions {
		nested = limits.MaxTransitions * perTransition
	}
	elements = cappedAdd(elements, nested, maximumSnapshotElements)
	if elements >= (maximumSnapshotNodes-fixedSnapshotNodes)/nodesPerSemanticElement {
		return maximumSnapshotNodes
	}
	return fixedSnapshotNodes + elements*nodesPerSemanticElement
}

func cappedAdd(left, right, limit uint64) uint64 {
	if left >= limit || right >= limit-left {
		return limit
	}
	return left + right
}

func validateJSONStringTokenLengths(data []byte, maximum uint64) error {
	for index := 0; index < len(data); index++ {
		if data[index] != '"' {
			continue
		}
		start := index
		for index++; index < len(data); index++ {
			switch data[index] {
			case '\\':
				index++
			case '"':
				if uint64(index-start-1) > maximum {
					return fmt.Errorf("string exceeds configured bound")
				}
				goto next
			}
		}
		return fmt.Errorf("unterminated string")
	next:
	}
	return nil
}

func consumeBoundedJSONArray(decoder *json.Decoder, limit, nestedArrayLimit uint64, budget *uint64, depth int) error {
	if err := consumeSnapshotBudget(budget); err != nil {
		return err
	}
	opening, err := decoder.Token()
	if err != nil || opening != json.Delim('[') {
		return fmt.Errorf("value is not an array")
	}
	var count uint64
	for decoder.More() {
		if count >= limit {
			return fmt.Errorf("element count exceeds configured limit")
		}
		count++
		if err := consumeBoundedJSONValue(decoder, nestedArrayLimit, budget, depth+1); err != nil {
			return err
		}
	}
	closing, err := decoder.Token()
	if err != nil || closing != json.Delim(']') {
		return fmt.Errorf("invalid array close")
	}
	return nil
}

func consumeBoundedJSONValue(decoder *json.Decoder, nestedArrayLimit uint64, budget *uint64, depth int) error {
	if depth > 64 {
		return fmt.Errorf("JSON nesting exceeds its bound")
	}
	if err := consumeSnapshotBudget(budget); err != nil {
		return err
	}
	token, err := decoder.Token()
	if err != nil {
		return err
	}
	delimiter, ok := token.(json.Delim)
	if !ok {
		return nil
	}
	var closing json.Delim
	switch delimiter {
	case '{':
		closing = '}'
		for decoder.More() {
			if err := consumeSnapshotBudget(budget); err != nil {
				return err
			}
			if _, err := decoder.Token(); err != nil {
				return err
			}
			if err := consumeBoundedJSONValue(decoder, nestedArrayLimit, budget, depth+1); err != nil {
				return err
			}
		}
	case '[':
		closing = ']'
		var count uint64
		for decoder.More() {
			if count >= nestedArrayLimit {
				return fmt.Errorf("aggregate element count exceeds configured limit")
			}
			count++
			if err := consumeBoundedJSONValue(decoder, nestedArrayLimit, budget, depth+1); err != nil {
				return err
			}
		}
	default:
		return fmt.Errorf("unexpected delimiter %q", delimiter)
	}
	token, err = decoder.Token()
	if err != nil || token != closing {
		return fmt.Errorf("invalid JSON close")
	}
	return nil
}

func consumeSnapshotBudget(budget *uint64) error {
	if *budget == 0 {
		return fmt.Errorf("aggregate JSON node count exceeds configured limit")
	}
	*budget--
	return nil
}

func canonicalJSON(value any) ([]byte, error) {
	var encoded bytes.Buffer
	encoder := json.NewEncoder(&encoded)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(value); err != nil {
		return nil, err
	}
	decoder := json.NewDecoder(bytes.NewReader(bytes.TrimSuffix(encoded.Bytes(), []byte{'\n'})))
	decoder.UseNumber()
	var decoded any
	if err := decoder.Decode(&decoded); err != nil {
		return nil, err
	}
	var canonical bytes.Buffer
	if err := appendCanonicalJSON(&canonical, decoded); err != nil {
		return nil, err
	}
	return canonical.Bytes(), nil
}

func appendCanonicalJSON(output *bytes.Buffer, value any) error {
	switch typed := value.(type) {
	case nil:
		output.WriteString("null")
	case bool:
		output.WriteString(strconv.FormatBool(typed))
	case string:
		return appendCanonicalJSONString(output, typed)
	case json.Number:
		number := string(typed)
		if strings.ContainsAny(number, ".eE") {
			return fmt.Errorf("floating-point JSON value %q", number)
		}
		if _, err := strconv.ParseInt(number, 10, 64); err != nil {
			if _, unsignedErr := strconv.ParseUint(number, 10, 64); unsignedErr != nil {
				return fmt.Errorf("invalid JSON integer %q", number)
			}
		}
		output.WriteString(number)
	case []any:
		output.WriteByte('[')
		for index, item := range typed {
			if index > 0 {
				output.WriteByte(',')
			}
			if err := appendCanonicalJSON(output, item); err != nil {
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
			if err := appendCanonicalJSONString(output, key); err != nil {
				return err
			}
			output.WriteByte(':')
			if err := appendCanonicalJSON(output, typed[key]); err != nil {
				return err
			}
		}
		output.WriteByte('}')
	default:
		return fmt.Errorf("unsupported canonical JSON value %T", value)
	}
	return nil
}

func appendCanonicalJSONString(output *bytes.Buffer, value string) error {
	if !utf8.ValidString(value) {
		return fmt.Errorf("invalid UTF-8 JSON string")
	}
	var encoded bytes.Buffer
	encoder := json.NewEncoder(&encoded)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(value); err != nil {
		return err
	}
	output.Write(bytes.TrimSuffix(encoded.Bytes(), []byte{'\n'}))
	return nil
}

func validateJSONStructure(data []byte) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	if err := consumeJSONValue(decoder); err != nil {
		return err
	}
	return requireJSONEOF(decoder)
}

func consumeJSONValue(decoder *json.Decoder) error {
	token, err := decoder.Token()
	if err != nil {
		return err
	}
	delimiter, isDelimiter := token.(json.Delim)
	if !isDelimiter {
		return nil
	}
	switch delimiter {
	case '{':
		seen := make(map[string]struct{})
		for decoder.More() {
			keyToken, keyErr := decoder.Token()
			if keyErr != nil {
				return keyErr
			}
			key, ok := keyToken.(string)
			if !ok {
				return fmt.Errorf("object key is not a string")
			}
			if _, found := seen[key]; found {
				return fmt.Errorf("duplicate object key %q", key)
			}
			seen[key] = struct{}{}
			if err := consumeJSONValue(decoder); err != nil {
				return err
			}
		}
		closing, closeErr := decoder.Token()
		if closeErr != nil || closing != json.Delim('}') {
			return fmt.Errorf("invalid object close: %w", closeErr)
		}
	case '[':
		for decoder.More() {
			if err := consumeJSONValue(decoder); err != nil {
				return err
			}
		}
		closing, closeErr := decoder.Token()
		if closeErr != nil || closing != json.Delim(']') {
			return fmt.Errorf("invalid array close: %w", closeErr)
		}
	default:
		return fmt.Errorf("unexpected delimiter %q", delimiter)
	}
	return nil
}

func requireJSONEOF(decoder *json.Decoder) error {
	if token, err := decoder.Token(); err != io.EOF {
		if err != nil {
			return err
		}
		return fmt.Errorf("unexpected trailing token %v", token)
	}
	return nil
}
