package artifact

import (
	"errors"
	"fmt"

	"go.temporal.io/server/tools/umpire/internal/artifactv2"
)

var rawEvidenceV2Decoder = Decoder[artifactv2.RawEvidence]{
	Format: artifactv2.RawEvidenceFormat,
	NestedFormats: []NestedFormat{
		{Path: "$.experiment.formatVersion", Format: artifactv2.ExperimentFormat},
		{Path: "$.runtimeConfiguration.formatVersion", Format: artifactv2.RuntimeConfigurationFormat},
		{Path: "$.run.formatVersion", Format: artifactv2.ExperimentRunFormat},
	},
	Bounds: Bounds{
		CollectionLimit: rawEvidenceV2CollectionLimit,
		StringLimit:     runtimeV2StringLimit,
		PayloadLimit:    rawEvidenceV2PayloadLimit,
	},
	Validate:           artifactv2.ValidateRawEvidence,
	Canonical:          artifactv2.CanonicalRawEvidenceBytes,
	ProvenanceChecksum: artifactv2.VerifyRawEvidenceProvenanceChecksum,
	ArtifactChecksum:   artifactv2.VerifyRawEvidenceArtifactChecksum,
}

// DecodeRawEvidenceV2 admits only the canonical persisted bounded v2 RawEvidence bytes.
func DecodeRawEvidenceV2(encoded []byte) (artifactv2.RawEvidence, error) {
	return rawEvidenceV2Decoder.Decode(encoded)
}

// EncodeRawEvidenceV2 returns the sole canonical persisted v2 RawEvidence representation.
func EncodeRawEvidenceV2(document artifactv2.RawEvidence) ([]byte, error) {
	encoded, err := artifactv2.CanonicalRawEvidenceBytes(document)
	if err != nil {
		return nil, wrapAdmission(ErrorMalformedValue, err)
	}
	if _, err := rawEvidenceV2Decoder.Decode(encoded); err != nil {
		return nil, err
	}
	return encoded, nil
}

// ValidateRawEvidenceV2Closure checks exact input bindings, Run source closure, and control receipts.
func ValidateRawEvidenceV2Closure(
	document artifactv2.RawEvidence,
	experiment artifactv2.Experiment,
	runtimeConfiguration artifactv2.RuntimeConfiguration,
	run artifactv2.ExperimentRun,
) error {
	if err := artifactv2.ValidateRawEvidenceClosure(document, experiment, runtimeConfiguration, run); err != nil {
		return wrapAdmission(ErrorClosure, err)
	}
	return nil
}

func rawEvidenceV2CollectionLimit(path JSONPath, kind CollectionKind) int {
	if kind != CollectionArray {
		return 0
	}
	switch path {
	case "$.sources":
		return MaximumEvidenceSources
	case "$.facts":
		return MaximumEvidenceFacts
	case "$.facts[*].fields":
		return MaximumFieldsPerEvidenceFact
	default:
		return 0
	}
}

func rawEvidenceV2PayloadLimit(encoded []byte) error {
	cursor := jsonCursor{encoded: encoded}
	cursor.skipSpace()
	if cursor.position >= len(encoded) || encoded[cursor.position] != '{' {
		return errors.New("RawEvidence payload root is not an object")
	}
	cursor.position++
	cursor.skipSpace()
	for cursor.position < len(encoded) && encoded[cursor.position] != '}' {
		key, next, err := scanJSONString(encoded, cursor.position)
		if err != nil {
			return err
		}
		cursor.position = next
		cursor.skipSpace()
		if cursor.position >= len(encoded) || encoded[cursor.position] != ':' {
			return errors.New("RawEvidence payload member is missing a colon")
		}
		cursor.position++
		if jsonStringEqualsPlain(encoded, key, "facts") {
			return rawEvidenceV2FactsPayloadLimit(&cursor)
		}
		if err := cursor.skipValue(); err != nil {
			return err
		}
		cursor.skipComma()
	}
	return nil
}

func rawEvidenceV2FactsPayloadLimit(cursor *jsonCursor) error {
	cursor.skipSpace()
	if cursor.position >= len(cursor.encoded) || cursor.encoded[cursor.position] != '[' {
		return errors.New("RawEvidence facts payload is not an array")
	}
	cursor.position++
	cursor.skipSpace()
	aggregate := 0
	for cursor.position < len(cursor.encoded) && cursor.encoded[cursor.position] != ']' {
		factPayload, err := rawEvidenceV2FactPayload(cursor)
		if err != nil {
			return err
		}
		if exceeds(factPayload, MaximumEvidenceFactPayloadBytes) {
			return fmt.Errorf("RawEvidence fact has %d decoded payload bytes; limit is %d",
				factPayload, MaximumEvidenceFactPayloadBytes)
		}
		aggregate += factPayload
		if exceeds(aggregate, MaximumRawEvidencePayloadBytes) {
			return fmt.Errorf("RawEvidence has %d decoded payload bytes; limit is %d",
				aggregate, MaximumRawEvidencePayloadBytes)
		}
		cursor.skipComma()
	}
	return nil
}

func rawEvidenceV2FactPayload(cursor *jsonCursor) (int, error) {
	cursor.skipSpace()
	if cursor.position >= len(cursor.encoded) || cursor.encoded[cursor.position] != '{' {
		return 0, errors.New("RawEvidence fact payload is not an object")
	}
	cursor.position++
	cursor.skipSpace()
	payload := 0
	for cursor.position < len(cursor.encoded) && cursor.encoded[cursor.position] != '}' {
		key, next, err := scanJSONString(cursor.encoded, cursor.position)
		if err != nil {
			return 0, err
		}
		cursor.position = next
		cursor.skipSpace()
		if cursor.position >= len(cursor.encoded) || cursor.encoded[cursor.position] != ':' {
			return 0, errors.New("RawEvidence fact member is missing a colon")
		}
		cursor.position++
		if jsonStringEqualsPlain(cursor.encoded, key, "fields") {
			payload, err = rawEvidenceV2FieldsPayload(cursor)
			if err != nil {
				return 0, err
			}
		} else if err := cursor.skipValue(); err != nil {
			return 0, err
		}
		cursor.skipComma()
	}
	if cursor.position < len(cursor.encoded) {
		cursor.position++
	}
	return payload, nil
}

func rawEvidenceV2FieldsPayload(cursor *jsonCursor) (int, error) {
	cursor.skipSpace()
	if cursor.position >= len(cursor.encoded) || cursor.encoded[cursor.position] != '[' {
		return 0, errors.New("RawEvidence fields payload is not an array")
	}
	cursor.position++
	cursor.skipSpace()
	payload := 0
	for cursor.position < len(cursor.encoded) && cursor.encoded[cursor.position] != ']' {
		fieldPayload, err := rawEvidenceV2FieldPayload(cursor)
		if err != nil {
			return 0, err
		}
		payload += fieldPayload
		cursor.skipComma()
	}
	if cursor.position < len(cursor.encoded) {
		cursor.position++
	}
	return payload, nil
}

func rawEvidenceV2FieldPayload(cursor *jsonCursor) (int, error) {
	cursor.skipSpace()
	if cursor.position >= len(cursor.encoded) || cursor.encoded[cursor.position] != '{' {
		return 0, errors.New("RawEvidence field payload is not an object")
	}
	cursor.position++
	cursor.skipSpace()
	payload := 0
	for cursor.position < len(cursor.encoded) && cursor.encoded[cursor.position] != '}' {
		key, next, err := scanJSONString(cursor.encoded, cursor.position)
		if err != nil {
			return 0, err
		}
		cursor.position = next
		cursor.skipSpace()
		if cursor.position >= len(cursor.encoded) || cursor.encoded[cursor.position] != ':' {
			return 0, errors.New("RawEvidence field member is missing a colon")
		}
		cursor.position++
		if jsonStringEqualsPlain(cursor.encoded, key, "value") {
			payload, err = decodedRawEvidenceValueBytes(cursor)
			if err != nil {
				return 0, err
			}
		} else if err := cursor.skipValue(); err != nil {
			return 0, err
		}
		cursor.skipComma()
	}
	if cursor.position < len(cursor.encoded) {
		cursor.position++
	}
	return payload, nil
}

func decodedRawEvidenceValueBytes(cursor *jsonCursor) (int, error) {
	cursor.skipSpace()
	if cursor.position >= len(cursor.encoded) {
		return 0, errors.New("RawEvidence field value is missing")
	}
	if cursor.encoded[cursor.position] == '"' {
		value, next, err := scanJSONString(cursor.encoded, cursor.position)
		if err != nil {
			return 0, err
		}
		cursor.position = next
		return value.decodedBytes, nil
	}
	if cursor.encoded[cursor.position] == '{' || cursor.encoded[cursor.position] == '[' {
		start := cursor.position
		if err := cursor.skipValue(); err != nil {
			return 0, err
		}
		return cursor.position - start, nil
	}
	start := cursor.position
	for cursor.position < len(cursor.encoded) && !isJSONSeparator(cursor.encoded[cursor.position]) {
		cursor.position++
	}
	value := cursor.encoded[start:cursor.position]
	if string(value) == "null" {
		return 0, nil
	}
	if string(value) == "true" || string(value) == "false" {
		return 1, nil
	}
	return len(value), nil
}
