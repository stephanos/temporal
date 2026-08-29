// Package artifact owns bounded, exact-byte admission for inert Umpire Artifacts.
package artifact

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"unicode/utf8"
)

type JSONPath string

type CollectionKind string

const (
	CollectionArray  CollectionKind = "array"
	CollectionObject CollectionKind = "object"
)

type Bounds struct {
	CollectionLimit func(JSONPath, CollectionKind) int
	StringLimit     func(JSONPath) int
	PayloadLimit    func([]byte) error
}

type Decoder[T any] struct {
	Format             string
	NestedFormats      []NestedFormat
	Bounds             Bounds
	Validate           func(T) error
	Canonical          func(T) ([]byte, error)
	ProvenanceChecksum func(T) error
	ArtifactChecksum   func(T) error
	Closure            func(T) error
}

type NestedFormat struct {
	Path   JSONPath
	Format string
}

func (d Decoder[T]) Decode(encoded []byte) (T, error) {
	return decodeWithStructuralLimits(d, encoded, standardStructuralLimits)
}

func decodeWithStructuralLimits[T any](d Decoder[T], encoded []byte, limits structuralLimits) (T, error) {
	var zero T
	analysis, err := inspectAdmission[T](d, encoded, limits)
	if err != nil {
		return zero, err
	}
	value, err := decodeTransport(d, encoded)
	if err != nil {
		return zero, err
	}
	if err := verifyCanonical(d, value, encoded, analysis); err != nil {
		return zero, err
	}
	if err := runFinalChecks(d, value); err != nil {
		return zero, err
	}
	return value, nil
}

func inspectAdmission[T any](d Decoder[T], encoded []byte, limits structuralLimits) (jsonAnalysis, error) {
	analysis, err := inspectAdmissionFormat(d, encoded, limits)
	if err != nil {
		return jsonAnalysis{}, err
	}
	if analysis.unknownField {
		return jsonAnalysis{}, wrapAdmission(ErrorUnknownField, errors.New("JSON object contains an unknown field"))
	}
	if analysis.collectionLimit {
		return jsonAnalysis{}, wrapAdmission(ErrorCollectionLimit, errors.New("JSON collection exceeds its limit"))
	}
	if analysis.stringLimit {
		return jsonAnalysis{}, wrapAdmission(ErrorStringLimit, errors.New("decoded JSON string exceeds its byte limit"))
	}
	if d.Bounds.PayloadLimit != nil {
		if err := d.Bounds.PayloadLimit(encoded); err != nil {
			return jsonAnalysis{}, wrapAdmission(ErrorPayloadLimit, err)
		}
	}
	return analysis, nil
}

func inspectAdmissionFormat[T any](d Decoder[T], encoded []byte, limits structuralLimits) (jsonAnalysis, error) {
	if err := checkDocumentBytes(encoded, limits); err != nil {
		return jsonAnalysis{}, err
	}
	if !utf8.Valid(encoded) || !json.Valid(encoded) {
		return jsonAnalysis{}, wrapAdmission(ErrorSyntax, errors.New("document is not one valid UTF-8 JSON value"))
	}
	metrics, err := measureJSON(encoded)
	if err != nil {
		return jsonAnalysis{}, wrapAdmission(ErrorSyntax, err)
	}
	if err := checkStructuralMetrics(metrics, limits); err != nil {
		return jsonAnalysis{}, err
	}
	analysis, err := inspectJSON(encoded, schemaFor[T](), d.NestedFormats, d.Bounds, limits)
	if err != nil {
		return jsonAnalysis{}, wrapAdmission(ErrorSyntax, err)
	}
	if analysis.duplicateKey {
		return jsonAnalysis{}, wrapAdmission(ErrorDuplicateKey, errors.New("JSON object contains a duplicate key"))
	}
	if analysis.caseCollision {
		return jsonAnalysis{}, wrapAdmission(ErrorCaseCollision, errors.New("JSON object key collides under case folding"))
	}
	actualFormat, ok := analysis.format()
	if !ok {
		return jsonAnalysis{}, wrapAdmission(ErrorUnsupportedFormat, errors.New("formatVersion is missing or is not a string"))
	}
	if code := compareFormat(d.Format, actualFormat); code != "" {
		return jsonAnalysis{}, wrapAdmission(code, fmt.Errorf("got %q; expected %q", actualFormat, d.Format))
	}
	for _, expected := range d.NestedFormats {
		actual, ok := analysis.nestedFormat(expected.Path)
		if !ok {
			return jsonAnalysis{}, wrapAdmission(ErrorUnsupportedFormat,
				fmt.Errorf("%s is missing or is not a string", expected.Path))
		}
		if code := compareFormat(expected.Format, actual); code != "" {
			return jsonAnalysis{}, wrapAdmission(code,
				fmt.Errorf("got %q at %s; expected %q", actual, expected.Path, expected.Format))
		}
	}
	return analysis, nil
}

func checkStructuralMetrics(metrics jsonMetrics, limits structuralLimits) error {
	if exceeds(metrics.tokens, limits.tokens) {
		return wrapAdmission(ErrorTokenLimit, fmt.Errorf("document has %d tokens; limit is %d", metrics.tokens, limits.tokens))
	}
	if exceeds(metrics.depth, limits.depth) {
		return wrapAdmission(ErrorDepthLimit, fmt.Errorf("document has depth %d; limit is %d", metrics.depth, limits.depth))
	}
	return nil
}

func decodeTransport[T any](d Decoder[T], encoded []byte) (T, error) {
	var zero T
	decoder := json.NewDecoder(bytes.NewReader(encoded))
	decoder.DisallowUnknownFields()
	decoder.UseNumber()
	var value T
	if err := decoder.Decode(&value); err != nil {
		return zero, wrapAdmission(ErrorMalformedValue, err)
	}
	if err := requireEOF(decoder); err != nil {
		return zero, wrapAdmission(ErrorMalformedValue, err)
	}
	if d.Validate != nil {
		if err := d.Validate(value); err != nil {
			return zero, wrapAdmission(ErrorMalformedValue, err)
		}
	}
	return value, nil
}

func verifyCanonical[T any](d Decoder[T], value T, encoded []byte, analysis jsonAnalysis) error {
	if analysis.noncanonicalValue {
		return wrapAdmission(ErrorNoncanonical, errors.New("JSON number is not a canonical integer"))
	}
	canonical := d.Canonical
	if canonical == nil {
		canonical = func(value T) ([]byte, error) {
			return CanonicalPretty(value)
		}
	}
	canonicalBytes, err := canonical(value)
	if err != nil {
		return wrapAdmission(ErrorMalformedValue, err)
	}
	if !bytes.Equal(encoded, canonicalBytes) {
		return wrapAdmission(ErrorNoncanonical, errors.New("document is not exact deterministic pretty JSON"))
	}
	return nil
}

func runFinalChecks[T any](d Decoder[T], value T) error {
	for _, check := range []struct {
		code ErrorCode
		fn   func(T) error
	}{
		{code: ErrorProvenanceChecksum, fn: d.ProvenanceChecksum},
		{code: ErrorArtifactChecksum, fn: d.ArtifactChecksum},
		{code: ErrorClosure, fn: d.Closure},
	} {
		if check.fn == nil {
			continue
		}
		if err := check.fn(value); err != nil {
			return wrapAdmission(check.code, err)
		}
	}
	return nil
}

func compareFormat(expected, actual string) ErrorCode {
	expectedFamily, expectedVersion, expectedOK := splitFormat(expected)
	actualFamily, actualVersion, actualOK := splitFormat(actual)
	if !expectedOK || !actualOK || actualVersion != expectedVersion {
		return ErrorUnsupportedFormat
	}
	if actualFamily != expectedFamily {
		return ErrorWrongFamily
	}
	return ""
}

func splitFormat(format string) (family string, version string, ok bool) {
	separator := strings.LastIndex(format, "/v")
	if separator <= 0 || separator+2 == len(format) {
		return "", "", false
	}
	version = format[separator+2:]
	for _, character := range version {
		if character < '0' || character > '9' {
			return "", "", false
		}
	}
	return format[:separator], version, true
}

func requireEOF(decoder *json.Decoder) error {
	if token, err := decoder.Token(); err != io.EOF {
		if err != nil {
			return err
		}
		return fmt.Errorf("unexpected trailing JSON token %v", token)
	}
	return nil
}
