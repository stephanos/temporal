package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
)

func generateArtifacts(selection descriptorSelection, projection projection) (map[string][]byte, error) {
	lean, err := generateLean(projection)
	if err != nil {
		return nil, err
	}
	descriptorManifest := struct {
		FormatVersion    string              `json:"formatVersion"`
		DescriptorDigest string              `json:"descriptorDigest"`
		Roots            []string            `json:"roots"`
		Deferred         []messageSelection  `json:"deferred"`
		Files            []string            `json:"files"`
		Messages         []messageProjection `json:"messages"`
		Enums            []enumProjection    `json:"enums"`
		Features         featureSet          `json:"features"`
	}{
		FormatVersion: "umpire3/protobuf-descriptors/v1", DescriptorDigest: projection.DescriptorDigest,
		Roots: projection.Roots, Files: projection.Files, Messages: projection.Messages,
		Enums: projection.Enums, Features: projection.Features,
	}
	for _, message := range selection.Messages {
		if message.Status == "deferred" {
			descriptorManifest.Deferred = append(descriptorManifest.Deferred, message)
		}
	}
	descriptors, err := canonicalIndentedJSON(descriptorManifest)
	if err != nil {
		return nil, err
	}
	dispositions, err := canonicalIndentedJSON(struct {
		FormatVersion string              `json:"formatVersion"`
		Messages      []messageProjection `json:"messages"`
	}{FormatVersion: "umpire3/field-dispositions/v1", Messages: projection.Messages})
	if err != nil {
		return nil, err
	}
	fixtures, err := generateFixtures(projection)
	if err != nil {
		return nil, err
	}
	return map[string][]byte{
		leanOutput: lean, descriptorOutput: descriptors, dispositionsOutput: dispositions,
		fixturesOutput: fixtures,
	}, nil
}

func generateLean(projection projection) ([]byte, error) {
	var generated strings.Builder
	generated.WriteString("namespace Umpire3.Temporal.API.Generated\n\n")
	fmt.Fprintf(&generated, "def descriptorHash : String := %q\n\n", projection.DescriptorDigest)
	generated.WriteString("structure RedactedBytes where\n  digest : String\n  size : Nat\n  deriving DecidableEq, Repr\n\n")
	generated.WriteString("structure BoundedMessage where\n  descriptor : String\n  remainingDepth : Nat\n  deriving DecidableEq, Repr\n\n")
	generated.WriteString("structure FieldMetadata where\n  path : String\n  kind : String\n  presence : Bool\n  oneofName : String\n  repeated : Bool\n  mapField : Bool\n  disposition : String\n  deriving DecidableEq, Repr\n\n")
	generated.WriteString("structure FieldDomain where\n  path : String\n  cases : List String\n  deriving DecidableEq, Repr\n\n")
	for _, enum := range projection.Enums {
		fmt.Fprintf(&generated, "structure %s where\n  number : Int\n  deriving DecidableEq, Repr\n\n", enum.LeanName)
		fmt.Fprintf(&generated, "namespace %s\n", enum.LeanName)
		for _, value := range enum.Values {
			fmt.Fprintf(&generated, "def %s : %s := { number := %d }\n", leanEnumName(value.Name), enum.LeanName, value.Number)
		}
		fmt.Fprintf(&generated, "end %s\n\n", enum.LeanName)
	}
	for _, message := range projection.Messages {
		fmt.Fprintf(&generated, "structure %s where\n", message.LeanName)
		if len(message.Fields) == 0 {
			generated.WriteString("  unit : Unit := ()\n")
		}
		for _, field := range message.Fields {
			fmt.Fprintf(&generated, "  %s : %s\n", field.LeanName, field.LeanType)
		}
		generated.WriteString("  deriving Repr\n\n")
	}
	generated.WriteString("def fieldMetadata : List FieldMetadata := [\n")
	for _, message := range projection.Messages {
		for _, field := range message.Fields {
			fmt.Fprintf(&generated, "  { path := %q, kind := %q, presence := %t, oneofName := %q, repeated := %t, mapField := %t, disposition := %q },\n",
				field.FullName, field.Kind, field.Presence, field.Oneof, field.Repeated, field.Map, field.Disposition)
		}
	}
	generated.WriteString("]\n\nend Umpire3.Temporal.API.Generated\n")
	generated.WriteString("\nnamespace Umpire3.Temporal.API.Generated\n\n")
	generated.WriteString("def fieldDomains : List FieldDomain := [\n")
	for _, message := range projection.Messages {
		for _, field := range message.Fields {
			cases := fieldDomainCases(field)
			fmt.Fprintf(&generated, "  { path := %q, cases := [%s] },\n", field.FullName, leanStrings(cases))
		}
	}
	generated.WriteString("]\n\nend Umpire3.Temporal.API.Generated\n")
	return []byte(generated.String()), nil
}

func leanEnumName(value string) string {
	name := goLikeIdentifier(strings.ToLower(value))
	if name == "" {
		return "unknown"
	}
	characters := []rune(name)
	characters[0] = []rune(strings.ToLower(string(characters[0])))[0]
	return string(characters)
}

func generateFixtures(projection projection) ([]byte, error) {
	type fixture struct {
		Identifier string   `json:"identifier"`
		AppliesTo  []string `json:"appliesTo"`
		Expected   string   `json:"expected"`
	}
	fieldsFor := func(predicate func(fieldProjection) bool) []string {
		var result []string
		for _, message := range projection.Messages {
			for _, field := range message.Fields {
				if predicate(field) {
					result = append(result, field.FullName)
				}
			}
		}
		slices.Sort(result)
		return result
	}
	fixtures := []fixture{
		{Identifier: "presence.absent-vs-present-default", AppliesTo: fieldsFor(func(field fieldProjection) bool { return field.Presence }), Expected: "distinct"},
		{Identifier: "enum.unknown-number", AppliesTo: fieldsFor(func(field fieldProjection) bool { return field.Kind == "enum" }), Expected: "preserved"},
		{Identifier: "oneof.last-value-wins", AppliesTo: fieldsFor(func(field fieldProjection) bool { return field.Oneof != "" }), Expected: "replacement-preserved"},
		{Identifier: "map.canonical-key-order", AppliesTo: fieldsFor(func(field fieldProjection) bool { return field.Map }), Expected: "sorted"},
		{Identifier: "repeated.source-order", AppliesTo: fieldsFor(func(field fieldProjection) bool { return field.Repeated }), Expected: "preserved"},
		{Identifier: "bytes.redacted", AppliesTo: fieldsFor(func(field fieldProjection) bool { return field.Kind == "bytes" }), Expected: "sha256-and-size-only"},
		{Identifier: "duration.negative-and-overflow", AppliesTo: []string{"google.protobuf.Duration"}, Expected: "retained-for-interpretation"},
	}
	return canonicalIndentedJSON(struct {
		FormatVersion    string    `json:"formatVersion"`
		DescriptorDigest string    `json:"descriptorDigest"`
		Fixtures         []fixture `json:"fixtures"`
	}{FormatVersion: "umpire3/protobuf-fixtures/v1", DescriptorDigest: projection.DescriptorDigest, Fixtures: fixtures})
}

func canonicalIndentedJSON(value any) ([]byte, error) {
	var encoded bytes.Buffer
	encoder := json.NewEncoder(&encoded)
	encoder.SetIndent("", "  ")
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(value); err != nil {
		return nil, fmt.Errorf("encode generated API artifact: %w", err)
	}
	return encoded.Bytes(), nil
}

func fieldDomainCases(field fieldProjection) []string {
	var result []string
	if field.Presence {
		result = append(result, "absent", "present-default")
	}
	switch field.Kind {
	case "bool":
		result = append(result, "false", "true")
	case "enum":
		result = append(result, "known", "unknown-number")
	case "bytes":
		result = append(result, "empty-digest", "non-empty-digest")
	case "string":
		result = append(result, "empty", "non-empty")
	case "int32", "sint32", "sfixed32", "int64", "sint64", "sfixed64":
		result = append(result, "negative", "zero", "positive", "boundary")
	case "uint32", "fixed32", "uint64", "fixed64":
		result = append(result, "zero", "positive", "boundary")
	default:
		result = append(result, "default", "non-default")
	}
	if field.Oneof != "" {
		result = append(result, "oneof-replacement")
	}
	if field.Map {
		result = append(result, "map-permuted-keys")
	}
	if field.Repeated {
		result = append(result, "empty-list", "ordered-list")
	}
	return result
}

func leanStrings(values []string) string {
	quoted := make([]string, len(values))
	for index, value := range values {
		quoted[index] = fmt.Sprintf("%q", value)
	}
	return strings.Join(quoted, ", ")
}
