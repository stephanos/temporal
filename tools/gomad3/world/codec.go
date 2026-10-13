package world

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"unicode/utf8"

	"go.temporal.io/server/tools/gomad3/internal/strictjson"
)

const (
	MaximumSnapshotJSONBytes = 64 << 20
	maximumSnapshotElements  = 1 << 20
	maximumSnapshotNodes     = MaximumSnapshotJSONBytes
)

type worldJSONValue interface {
	Snapshot | ReplayPlan | Transition | Terminal | RequestSnapshot | EventSnapshot | Quiescence | Cancellation | LogicalTime
}

func encodeWorldJSON[T worldJSONValue](value T) ([]byte, error) {
	//gomad:dynamic reviewed 2026-10-13 closed World wire types marshal only caller-owned memory
	initial, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	decoder := json.NewDecoder(bytes.NewReader(initial))
	decoder.UseNumber()
	var normalized any
	//gomad:dynamic reviewed 2026-10-13 stdlib JSON normalization decodes only the just-marshaled World value
	if err := decoder.Decode(&normalized); err != nil {
		return nil, err
	}
	var output bytes.Buffer
	encoder := json.NewEncoder(&output)
	encoder.SetEscapeHTML(false)
	//gomad:dynamic reviewed 2026-10-13 stdlib JSON normalization encodes only its memory-owned intermediate value
	if err := encoder.Encode(normalized); err != nil {
		return nil, err
	}
	return bytes.TrimSuffix(output.Bytes(), []byte{'\n'}), nil
}

func EncodeSnapshot(snapshot Snapshot) ([]byte, error) {
	if _, err := Restore(snapshot, nil); err != nil {
		return nil, err
	}
	if err := validateSnapshotStrings(snapshot); err != nil {
		return nil, err
	}
	encoded, err := encodeWorldJSON(snapshot)
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
		//gomad:dynamic reviewed 2026-10-12 model codec and gzip errors expose memory-only messages
		return Snapshot{}, invalidSnapshot("json.bounds: " + err.Error())
	}
	var snapshot Snapshot
	if err := strictjson.Decode(data, &snapshot); err != nil {
		//gomad:dynamic reviewed 2026-10-12 model codec and gzip errors expose memory-only messages
		return Snapshot{}, invalidSnapshot("json: " + err.Error())
	}
	if _, err := Restore(snapshot, nil); err != nil {
		return Snapshot{}, err
	}
	canonical, err := encodeWorldJSON(snapshot)
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
		//gomad:dynamic reviewed 2026-10-12 reviewed codec and validator errors have memory-only methods
		return fmt.Errorf("config: %w", err)
	}
	if err := validateConfig(config); err != nil {
		return modelContextError("config", err)
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
				//gomad:dynamic reviewed 2026-10-12 reviewed codec and validator errors have memory-only methods
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

func requireJSONEOF(decoder *json.Decoder) error {
	if token, err := decoder.Token(); err != io.EOF {
		if err != nil {
			return err
		}
		//gomad:dynamic reviewed 2026-10-12 reviewed model or record values and in-memory streams constrain callback methods
		return fmt.Errorf("unexpected trailing token %v", token)
	}
	return nil
}
