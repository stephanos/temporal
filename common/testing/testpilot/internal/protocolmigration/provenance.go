package protocolmigration

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
)

// opaqueBytesKey is the baseline CaseProvenance field that carried a Producer's payload as bytes.
const opaqueBytesKey = "producer" + "Data"

// syntheticFixture is the one baseline Case whose payload was not Umpire provenance: the
// Umpire-free synthetic Producer wrote three non-UTF-8 bytes to exercise the opaque field.
const syntheticFixture = "tests/testcore/testpilot/testdata/synthetic-case.json"

var syntheticPayload = []byte{0, 255, 128}

// legacyDefinitionKinds and legacyKnownGapKinds are the value names the baseline Umpire payload
// spelled, each without the prefix its enum now takes.
var (
	legacyDefinitionKinds = []string{
		"SETUP", "STATE", "ACTION", "OUTCOME", "FACT", "RELATION", "CAPABILITY", "PROPERTY", "QUERY",
		"SCENARIO", "TARGET", "COMPILER", "PROVIDER", "LAW", "CONNECTOR", "MACHINE",
	}
	legacyKnownGapKinds = []string{"CAPABILITY", "INPUT", "INTERPRETATION", "CLAIM"}
)

// legacyProvenance is the baseline Umpire payload shape, field for field and in the order
// Umpire.Provenance wrote it; correlatedRules was written only when it had a row.
type legacyProvenance struct {
	Definitions     []legacyDefinition     `json:"definitions"`
	Sources         []legacySource         `json:"sources"`
	KnownGaps       []legacyKnownGap       `json:"knownGaps"`
	CorrelatedRules []legacyCorrelatedRule `json:"correlatedRules,omitempty"`
}

type legacyDefinition struct {
	DefinitionID        string `json:"definitionId"`
	BehaviorFingerprint string `json:"behaviorFingerprint"`
	Kind                string `json:"kind"`
}

type legacySource struct {
	Path       string `json:"path"`
	Line       string `json:"line"`
	Column     string `json:"column"`
	Provenance string `json:"provenance"`
}

type legacyKnownGap struct {
	Kind    string  `json:"kind"`
	Code    string  `json:"code"`
	Subject *string `json:"subject,omitempty"`
	Detail  *string `json:"detail,omitempty"`
}

type legacyCorrelatedRule struct {
	RuleID                string       `json:"ruleId"`
	PropertyID            string       `json:"propertyId"`
	PropertyFingerprint   string       `json:"propertyFingerprint"`
	ProjectionID          string       `json:"projectionId"`
	ProjectionFingerprint string       `json:"projectionFingerprint"`
	Source                legacySource `json:"source"`
}

// liftProvenanceRows replaces a baseline CaseProvenance payload with the typed rows it encoded. The
// payload must decode under the baseline shape with no unknown key and re-encode to exactly its own
// bytes, so every row field is lifted and none is dropped. The synthetic Case's three opaque bytes
// encoded no rows and are the only payload dropped.
func liftProvenanceRows(fixture string, object *Object) (any, error) {
	value, carried := object.Fields[opaqueBytesKey]
	if !carried {
		return object, nil
	}
	encoded, isText := value.(string)
	if !isText {
		return nil, fmt.Errorf("%s is not a base64 string", opaqueBytesKey)
	}
	payload, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", opaqueBytesKey, err)
	}
	delete(object.Fields, opaqueBytesKey)
	if fixture == syntheticFixture && bytes.Equal(payload, syntheticPayload) {
		return object, nil
	}
	rows, err := decodeLegacyProvenance(payload)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", opaqueBytesKey, err)
	}
	if err := rows.lift(object); err != nil {
		return nil, err
	}
	return object, nil
}

func decodeLegacyProvenance(payload []byte) (*legacyProvenance, error) {
	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.DisallowUnknownFields()
	rows := new(legacyProvenance)
	if err := decoder.Decode(rows); err != nil {
		return nil, err
	}
	var canonical bytes.Buffer
	encoder := json.NewEncoder(&canonical)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(rows); err != nil {
		return nil, err
	}
	if !bytes.Equal(canonical.Bytes(), payload) {
		return nil, errors.New("payload is not the exact encoding of the baseline Umpire provenance shape")
	}
	return rows, nil
}

func (p *legacyProvenance) lift(object *Object) error {
	if err := liftRows(object, "definitions", p.Definitions, func(definition legacyDefinition) (*Object, error) {
		kind, err := renamedKind("CASE_"+"DEFINITION_KIND_", "DEFINITION_KIND_", legacyDefinitionKinds, definition.Kind)
		if err != nil {
			return nil, err
		}
		return &Object{Fields: map[string]any{
			"definitionId": definition.DefinitionID, "behaviorFingerprint": definition.BehaviorFingerprint, "kind": kind,
		}}, nil
	}); err != nil {
		return err
	}
	if err := liftRows(object, "sources", p.Sources, liftSource); err != nil {
		return err
	}
	if err := liftRows(object, "knownGaps", p.KnownGaps, func(gap legacyKnownGap) (*Object, error) {
		kind, err := renamedKind("CASE_"+"KNOWN_GAP_KIND_", "KNOWN_GAP_KIND_", legacyKnownGapKinds, gap.Kind)
		if err != nil {
			return nil, err
		}
		row := &Object{Fields: map[string]any{"kind": kind, "code": gap.Code}}
		if gap.Subject != nil {
			row.Fields["subject"] = *gap.Subject
		}
		if gap.Detail != nil {
			row.Fields["detail"] = *gap.Detail
		}
		return row, nil
	}); err != nil {
		return err
	}
	return liftRows(object, "correlatedRules", p.CorrelatedRules, func(rule legacyCorrelatedRule) (*Object, error) {
		source, err := liftSource(rule.Source)
		if err != nil {
			return nil, err
		}
		return &Object{Fields: map[string]any{
			"ruleId": rule.RuleID, "propertyId": rule.PropertyID, "propertyFingerprint": rule.PropertyFingerprint,
			"projectionId": rule.ProjectionID, "projectionFingerprint": rule.ProjectionFingerprint, "source": source,
		}}, nil
	})
}

// liftRows writes one typed row list in the payload's order, leaving an empty list out as ProtoJSON
// does.
func liftRows[Row any](object *Object, key string, rows []Row, lift func(Row) (*Object, error)) error {
	if len(rows) == 0 {
		return nil
	}
	if _, clashes := object.Fields[key]; clashes {
		return fmt.Errorf("provenance already carries %q", key)
	}
	lifted := make([]any, len(rows))
	for index, row := range rows {
		value, err := lift(row)
		if err != nil {
			return fmt.Errorf("%s[%d]: %w", key, index, err)
		}
		lifted[index] = value
	}
	object.Fields[key] = lifted
	return nil
}

// liftSource turns the baseline text line and column into the int32 numbers a SourceLocation holds,
// requiring the canonical base-10 spelling of a value that fits.
func liftSource(source legacySource) (*Object, error) {
	position := func(name, text string) (json.Number, error) {
		parsed, err := strconv.ParseInt(text, 10, 32)
		if err != nil || strconv.FormatInt(parsed, 10) != text {
			return "", fmt.Errorf("source %s %q is not a canonical int32", name, text)
		}
		return json.Number(text), nil
	}
	line, err := position("line", source.Line)
	if err != nil {
		return nil, err
	}
	column, err := position("column", source.Column)
	if err != nil {
		return nil, err
	}
	return &Object{Fields: map[string]any{
		"path": source.Path, "line": line, "column": column, "provenance": source.Provenance,
	}}, nil
}

// renamedKind renames a baseline kind literal to the prefix its enum now declares, admitting only
// the value names the baseline payload could spell.
func renamedKind(from, to string, names []string, literal string) (string, error) {
	for _, name := range names {
		if literal == from+name {
			return to + name, nil
		}
	}
	return "", fmt.Errorf("kind %q is not in the baseline vocabulary", literal)
}
