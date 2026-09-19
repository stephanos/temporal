package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"slices"
)

const supportedIndexFormat = "umpire-plan-index/v1"

type planIndex struct {
	Format    string
	Documents []documentEntry
	FlowSpecs []flowSpecEntry
}

type documentEntry struct {
	Path                string
	Lifecycle           string
	Authority           string
	AuthorityParents    []string
	SupersededBy        *string
	AllowedMissingLinks []allowedMissingLink
}

type allowedMissingLink struct {
	Target string
	Reason string
	Anchor *string
}

type flowSpecEntry struct {
	ID               string
	Scope            string
	Disposition      string
	Phase            string
	Status           string
	Ready            bool
	CompletionReview string
	SpecDependencies []string
}

type jsonValue struct {
	kind   string
	object map[string]jsonValue
	array  []jsonValue
	text   string
	truth  bool
}

func parseIndex(encoded []byte) (planIndex, error) {
	root, err := decodeJSON(encoded)
	if err != nil {
		return planIndex{}, err
	}
	if err := requireObject(root, "$", []string{"format", "documents", "flowSpecs"}); err != nil {
		return planIndex{}, err
	}

	format, err := stringField(root, "format", "$")
	if err != nil {
		return planIndex{}, err
	}
	if format != supportedIndexFormat {
		return planIndex{}, fmt.Errorf("$.format: unsupported value %q", format)
	}
	documents, err := documentEntries(root.object["documents"], "$.documents")
	if err != nil {
		return planIndex{}, err
	}
	flowSpecs, err := flowSpecEntries(root.object["flowSpecs"], "$.flowSpecs")
	if err != nil {
		return planIndex{}, err
	}
	return planIndex{Format: format, Documents: documents, FlowSpecs: flowSpecs}, nil
}

func decodeJSON(encoded []byte) (jsonValue, error) {
	decoder := json.NewDecoder(bytes.NewReader(encoded))
	decoder.UseNumber()
	token, err := decoder.Token()
	if err != nil {
		return jsonValue{}, malformedJSONError(err)
	}
	value, err := decodeJSONValue(decoder, token, "$")
	if err != nil {
		return jsonValue{}, err
	}
	token, err = decoder.Token()
	if err == nil {
		return jsonValue{}, fmt.Errorf("decode plan index: unexpected trailing JSON token %v", token)
	}
	if !errors.Is(err, io.EOF) {
		return jsonValue{}, malformedJSONError(err)
	}
	return value, nil
}

func decodeJSONValue(decoder *json.Decoder, token json.Token, valuePath string) (jsonValue, error) {
	switch value := token.(type) {
	case json.Delim:
		return decodeJSONCollection(decoder, value, valuePath)
	case string:
		return jsonValue{kind: "string", text: value}, nil
	case bool:
		return jsonValue{kind: "boolean", truth: value}, nil
	case nil:
		return jsonValue{kind: "null"}, nil
	case json.Number:
		return jsonValue{kind: "number", text: string(value)}, nil
	default:
		return jsonValue{}, fmt.Errorf("decode plan index: unsupported JSON token %T", token)
	}
}

func decodeJSONCollection(decoder *json.Decoder, delimiter json.Delim, valuePath string) (jsonValue, error) {
	switch delimiter {
	case '{':
		return decodeJSONObject(decoder, valuePath)
	case '[':
		return decodeJSONArray(decoder, valuePath)
	default:
		return jsonValue{}, fmt.Errorf("decode plan index: unexpected delimiter %q", delimiter)
	}
}

func decodeJSONObject(decoder *json.Decoder, valuePath string) (jsonValue, error) {
	object := make(map[string]jsonValue)
	for decoder.More() {
		keyToken, err := decoder.Token()
		if err != nil {
			return jsonValue{}, malformedJSONError(err)
		}
		key, ok := keyToken.(string)
		if !ok {
			return jsonValue{}, errors.New("decode plan index: expected object field name")
		}
		if _, exists := object[key]; exists {
			return jsonValue{}, fmt.Errorf("%s: duplicate field %q", valuePath, key)
		}
		childToken, err := decoder.Token()
		if err != nil {
			return jsonValue{}, malformedJSONError(err)
		}
		child, err := decodeJSONValue(decoder, childToken, valuePath+"."+key)
		if err != nil {
			return jsonValue{}, err
		}
		object[key] = child
	}
	if _, err := decoder.Token(); err != nil {
		return jsonValue{}, malformedJSONError(err)
	}
	return jsonValue{kind: "object", object: object}, nil
}

func decodeJSONArray(decoder *json.Decoder, valuePath string) (jsonValue, error) {
	array := make([]jsonValue, 0)
	for index := 0; decoder.More(); index++ {
		childToken, err := decoder.Token()
		if err != nil {
			return jsonValue{}, malformedJSONError(err)
		}
		child, err := decodeJSONValue(decoder, childToken, fmt.Sprintf("%s[%d]", valuePath, index))
		if err != nil {
			return jsonValue{}, err
		}
		array = append(array, child)
	}
	if _, err := decoder.Token(); err != nil {
		return jsonValue{}, malformedJSONError(err)
	}
	return jsonValue{kind: "array", array: array}, nil
}

func malformedJSONError(err error) error {
	if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
		return errors.New("decode plan index: unexpected end of JSON input")
	}
	return fmt.Errorf("decode plan index: %w", err)
}

func documentEntries(value jsonValue, valuePath string) ([]documentEntry, error) {
	if value.kind != "array" {
		return nil, typeError(valuePath, "array", value)
	}
	entries := make([]documentEntry, 0, len(value.array))
	for index, item := range value.array {
		itemPath := fmt.Sprintf("%s[%d]", valuePath, index)
		fields := []string{"path", "lifecycle", "authority", "authorityParents", "supersededBy", "allowedMissingLinks"}
		if err := requireObject(item, itemPath, fields); err != nil {
			return nil, err
		}
		pathValue, err := stringField(item, "path", itemPath)
		if err != nil {
			return nil, err
		}
		lifecycle, err := enumField(item, "lifecycle", itemPath, "active", "reference", "historical", "superseded", "unclassified")
		if err != nil {
			return nil, err
		}
		authority, err := enumField(item, "authority", itemPath, "normative-rules", "delivery-order", "architecture", "scoped-contract", "descriptive", "historical", "unclassified")
		if err != nil {
			return nil, err
		}
		parents, err := stringArray(item.object["authorityParents"], itemPath+".authorityParents")
		if err != nil {
			return nil, err
		}
		supersededBy, err := nullableString(item.object["supersededBy"], itemPath+".supersededBy", false)
		if err != nil {
			return nil, err
		}
		missingLinks, err := missingLinkEntries(item.object["allowedMissingLinks"], itemPath+".allowedMissingLinks")
		if err != nil {
			return nil, err
		}
		entries = append(entries, documentEntry{
			Path: pathValue, Lifecycle: lifecycle, Authority: authority,
			AuthorityParents: parents, SupersededBy: supersededBy, AllowedMissingLinks: missingLinks,
		})
	}
	return entries, nil
}

func missingLinkEntries(value jsonValue, valuePath string) ([]allowedMissingLink, error) {
	if value.kind != "array" {
		return nil, typeError(valuePath, "array", value)
	}
	entries := make([]allowedMissingLink, 0, len(value.array))
	for index, item := range value.array {
		itemPath := fmt.Sprintf("%s[%d]", valuePath, index)
		if err := requireObject(item, itemPath, []string{"target", "reason", "anchor"}); err != nil {
			return nil, err
		}
		target, err := nonemptyStringField(item, "target", itemPath)
		if err != nil {
			return nil, err
		}
		reason, err := nonemptyStringField(item, "reason", itemPath)
		if err != nil {
			return nil, err
		}
		anchor, err := nullableString(item.object["anchor"], itemPath+".anchor", true)
		if err != nil {
			return nil, err
		}
		entries = append(entries, allowedMissingLink{Target: target, Reason: reason, Anchor: anchor})
	}
	return entries, nil
}

func flowSpecEntries(value jsonValue, valuePath string) ([]flowSpecEntry, error) {
	if value.kind != "array" {
		return nil, typeError(valuePath, "array", value)
	}
	entries := make([]flowSpecEntry, 0, len(value.array))
	for index, item := range value.array {
		itemPath := fmt.Sprintf("%s[%d]", valuePath, index)
		fields := []string{"id", "scope", "disposition", "phase", "status", "ready", "completionReview", "specDependencies"}
		if err := requireObject(item, itemPath, fields); err != nil {
			return nil, err
		}
		id, err := stringField(item, "id", itemPath)
		if err != nil {
			return nil, err
		}
		scope, err := enumField(item, "scope", itemPath, "umpire-roadmap", "umpire-support", "other")
		if err != nil {
			return nil, err
		}
		disposition, err := enumField(item, "disposition", itemPath, "retained", "completed-prerequisite", "deferred", "superseded", "out-of-scope", "unclassified")
		if err != nil {
			return nil, err
		}
		phase, err := enumField(item, "phase", itemPath, "p0", "p1", "p2", "p3", "verification", "support", "none")
		if err != nil {
			return nil, err
		}
		status, err := enumField(item, "status", itemPath, "open", "done")
		if err != nil {
			return nil, err
		}
		ready, err := boolField(item, "ready", itemPath)
		if err != nil {
			return nil, err
		}
		completionReview, err := enumField(item, "completionReview", itemPath, "unknown", "ship", "needs_work", "needs_human")
		if err != nil {
			return nil, err
		}
		dependencies, err := stringArray(item.object["specDependencies"], itemPath+".specDependencies")
		if err != nil {
			return nil, err
		}
		entries = append(entries, flowSpecEntry{
			ID: id, Scope: scope, Disposition: disposition, Phase: phase, Status: status,
			Ready: ready, CompletionReview: completionReview, SpecDependencies: dependencies,
		})
	}
	return entries, nil
}

func requireObject(value jsonValue, valuePath string, fields []string) error {
	if value.kind != "object" {
		return typeError(valuePath, "object", value)
	}
	allowed := make(map[string]struct{}, len(fields))
	for _, field := range fields {
		allowed[field] = struct{}{}
	}
	var unknown []string
	for field := range value.object {
		if _, ok := allowed[field]; !ok {
			unknown = append(unknown, field)
		}
	}
	if len(unknown) != 0 {
		slices.Sort(unknown)
		return fmt.Errorf("%s: unknown field %q", valuePath, unknown[0])
	}
	for _, field := range fields {
		if _, ok := value.object[field]; !ok {
			return fmt.Errorf("%s: missing field %q", valuePath, field)
		}
	}
	return nil
}

func stringField(object jsonValue, field, objectPath string) (string, error) {
	value := object.object[field]
	if value.kind != "string" {
		return "", typeError(objectPath+"."+field, "string", value)
	}
	return value.text, nil
}

func nonemptyStringField(object jsonValue, field, objectPath string) (string, error) {
	value, err := stringField(object, field, objectPath)
	if err != nil {
		return "", err
	}
	if value == "" {
		return "", fmt.Errorf("%s.%s: must not be empty", objectPath, field)
	}
	return value, nil
}

func enumField(object jsonValue, field, objectPath string, allowed ...string) (string, error) {
	value, err := stringField(object, field, objectPath)
	if err != nil {
		return "", err
	}
	for _, candidate := range allowed {
		if value == candidate {
			return value, nil
		}
	}
	return "", fmt.Errorf("%s.%s: unsupported value %q", objectPath, field, value)
}

func boolField(object jsonValue, field, objectPath string) (bool, error) {
	value := object.object[field]
	if value.kind != "boolean" {
		return false, typeError(objectPath+"."+field, "boolean", value)
	}
	return value.truth, nil
}

func nullableString(value jsonValue, valuePath string, nonempty bool) (*string, error) {
	if value.kind == "null" {
		return nil, nil
	}
	if value.kind != "string" {
		return nil, typeError(valuePath, "string or null", value)
	}
	if nonempty && value.text == "" {
		return nil, fmt.Errorf("%s: must not be empty", valuePath)
	}
	result := value.text
	return &result, nil
}

func stringArray(value jsonValue, valuePath string) ([]string, error) {
	if value.kind != "array" {
		return nil, typeError(valuePath, "array", value)
	}
	result := make([]string, 0, len(value.array))
	for index, item := range value.array {
		if item.kind != "string" {
			return nil, typeError(fmt.Sprintf("%s[%d]", valuePath, index), "string", item)
		}
		result = append(result, item.text)
	}
	return result, nil
}

func typeError(valuePath, expected string, actual jsonValue) error {
	kind := actual.kind
	if kind == "" {
		kind = "missing"
	}
	return fmt.Errorf("%s: expected %s, got %s", valuePath, expected, kind)
}
