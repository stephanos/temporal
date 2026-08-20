package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"slices"
	"strings"
	"unicode"

	_ "go.temporal.io/api/failure/v1"
	_ "go.temporal.io/api/history/v1"
	_ "go.temporal.io/api/nexus/v1"
	_ "go.temporal.io/api/workflowservice/v1"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/descriptorpb"
)

type featureSet struct {
	Presence bool `json:"presence"`
	Oneof    bool `json:"oneof"`
	Enum     bool `json:"enum"`
	Repeated bool `json:"repeated"`
	Map      bool `json:"map"`
	Nested   bool `json:"nested"`
	Bytes    bool `json:"bytes"`
	Duration bool `json:"duration"`
}

type enumValueProjection struct {
	Name   string `json:"name"`
	Number int32  `json:"number"`
}

type enumProjection struct {
	FullName string                `json:"fullName"`
	LeanName string                `json:"leanName"`
	Values   []enumValueProjection `json:"values"`
}

type fieldProjection struct {
	FullName    string           `json:"fullName"`
	Name        string           `json:"name"`
	LeanName    string           `json:"leanName"`
	Kind        string           `json:"kind"`
	LeanType    string           `json:"leanType"`
	TypeName    string           `json:"typeName,omitempty"`
	Presence    bool             `json:"presence"`
	Oneof       string           `json:"oneof,omitempty"`
	Repeated    bool             `json:"repeated"`
	Map         bool             `json:"map"`
	Recursive   bool             `json:"recursive"`
	Disposition fieldDisposition `json:"disposition"`
}

type messageProjection struct {
	FullName string            `json:"fullName"`
	LeanName string            `json:"leanName"`
	Root     bool              `json:"root"`
	Purpose  string            `json:"purpose"`
	Owner    string            `json:"owner"`
	Fields   []fieldProjection `json:"fields"`
}

type projection struct {
	DescriptorDigest string              `json:"descriptorDigest"`
	Files            []string            `json:"files"`
	Roots            []string            `json:"roots"`
	Messages         []messageProjection `json:"messages"`
	Enums            []enumProjection    `json:"enums"`
	Features         featureSet          `json:"features"`
}

func buildProjection(selection descriptorSelection) (projection, error) {
	selectedByName := make(map[protoreflect.FullName]messageSelection)
	messages := make(map[protoreflect.FullName]protoreflect.MessageDescriptor)
	enums := make(map[protoreflect.FullName]protoreflect.EnumDescriptor)
	var roots []string
	var addMessage func(protoreflect.MessageDescriptor)
	addEnum := func(descriptor protoreflect.EnumDescriptor) {
		enums[descriptor.FullName()] = descriptor
	}
	addMessage = func(descriptor protoreflect.MessageDescriptor) {
		if _, exists := messages[descriptor.FullName()]; exists {
			return
		}
		messages[descriptor.FullName()] = descriptor
		fields := descriptor.Fields()
		for index := 0; index < fields.Len(); index++ {
			field := fields.Get(index)
			if field.IsMap() {
				value := field.MapValue()
				if value.Message() != nil {
					addMessage(value.Message())
				}
				if value.Enum() != nil {
					addEnum(value.Enum())
				}
				continue
			}
			if field.Message() != nil {
				addMessage(field.Message())
			}
			if field.Enum() != nil {
				addEnum(field.Enum())
			}
		}
	}
	for _, selected := range selection.Messages {
		if selected.Status == "deferred" {
			continue
		}
		descriptorType, err := protoregistry.GlobalTypes.FindMessageByName(protoreflect.FullName(selected.FullName))
		if err != nil {
			return projection{}, fmt.Errorf("resolve selected protobuf message %q: %w", selected.FullName, err)
		}
		descriptor := descriptorType.Descriptor()
		for field := range selected.Fields {
			if descriptor.Fields().ByName(protoreflect.Name(field)) == nil {
				return projection{}, fmt.Errorf("selected protobuf message %q has no field %q", selected.FullName, field)
			}
		}
		selectedByName[descriptor.FullName()] = selected
		roots = append(roots, selected.FullName)
		addMessage(descriptor)
	}
	slices.Sort(roots)

	typeNames := make(map[protoreflect.FullName]string, len(messages)+len(enums))
	baseCounts := make(map[string]int)
	for name := range messages {
		baseCounts[baseTypeName(name)]++
	}
	for name := range enums {
		baseCounts[baseTypeName(name)]++
	}
	for name := range messages {
		typeNames[name] = uniqueTypeName(name, baseCounts)
	}
	for name := range enums {
		typeNames[name] = uniqueTypeName(name, baseCounts)
	}

	messageNames := sortedFullNames(messages)
	result := projection{Roots: roots}
	for _, name := range messageNames {
		descriptor := messages[name]
		selected, root := selectedByName[name]
		message := messageProjection{
			FullName: string(name), LeanName: typeNames[name], Root: root,
			Purpose: selected.Purpose, Owner: selected.Owner,
		}
		if !root {
			message.Purpose = "descriptor-dependency"
			message.Owner = "Umpire3.ProtobufProjection"
		}
		if _, nested := descriptor.Parent().(protoreflect.MessageDescriptor); nested {
			result.Features.Nested = true
		}
		fields := descriptor.Fields()
		for index := 0; index < fields.Len(); index++ {
			field := fields.Get(index)
			fieldResult := projectField(field, descriptor, messages, typeNames)
			fieldResult.Disposition = dispositionTransportOnly
			if root {
				fieldResult.Disposition = selected.DefaultFieldDisposition
				if explicit, ok := selected.Fields[string(field.Name())]; ok {
					fieldResult.Disposition = explicit
				}
			}
			message.Fields = append(message.Fields, fieldResult)
			result.Features.Presence = result.Features.Presence || fieldResult.Presence
			result.Features.Oneof = result.Features.Oneof || fieldResult.Oneof != ""
			result.Features.Repeated = result.Features.Repeated || fieldResult.Repeated
			result.Features.Map = result.Features.Map || fieldResult.Map
			result.Features.Bytes = result.Features.Bytes || field.Kind() == protoreflect.BytesKind
			result.Features.Enum = result.Features.Enum || field.Enum() != nil ||
				(field.IsMap() && field.MapValue().Enum() != nil)
			message.Fields[index] = fieldResult
		}
		if descriptor.FullName() == "google.protobuf.Duration" {
			result.Features.Duration = true
		}
		result.Messages = append(result.Messages, message)
	}
	result.Messages = dependencyOrder(result.Messages)
	for _, name := range sortedFullNames(enums) {
		descriptor := enums[name]
		values := descriptor.Values()
		enum := enumProjection{FullName: string(name), LeanName: typeNames[name]}
		for index := 0; index < values.Len(); index++ {
			value := values.Get(index)
			enum.Values = append(enum.Values, enumValueProjection{Name: string(value.Name()), Number: int32(value.Number())})
		}
		result.Enums = append(result.Enums, enum)
	}

	files := make(map[string]protoreflect.FileDescriptor)
	for _, descriptor := range messages {
		files[descriptor.ParentFile().Path()] = descriptor.ParentFile()
	}
	for _, descriptor := range enums {
		files[descriptor.ParentFile().Path()] = descriptor.ParentFile()
	}
	for path := range files {
		result.Files = append(result.Files, path)
	}
	slices.Sort(result.Files)
	descriptorSet := &descriptorpb.FileDescriptorSet{}
	for _, path := range result.Files {
		descriptorSet.File = append(descriptorSet.File, protodesc.ToFileDescriptorProto(files[path]))
	}
	encoded, err := proto.MarshalOptions{Deterministic: true}.Marshal(descriptorSet)
	if err != nil {
		return projection{}, fmt.Errorf("marshal selected descriptor closure: %w", err)
	}
	digest := sha256.Sum256(encoded)
	result.DescriptorDigest = "sha256:" + hex.EncodeToString(digest[:])
	return result, nil
}

func projectField(
	field protoreflect.FieldDescriptor,
	containing protoreflect.MessageDescriptor,
	messages map[protoreflect.FullName]protoreflect.MessageDescriptor,
	typeNames map[protoreflect.FullName]string,
) fieldProjection {
	result := fieldProjection{
		FullName: string(field.FullName()), Name: string(field.Name()), LeanName: leanFieldName(string(field.Name())),
		Kind: field.Kind().String(), Presence: field.HasPresence(), Repeated: field.Cardinality() == protoreflect.Repeated,
		Map: field.IsMap(),
	}
	if oneof := field.ContainingOneof(); oneof != nil {
		result.Oneof = string(oneof.Name())
	}
	if field.IsMap() {
		result.Repeated = false
		valueType := scalarLeanType(field.MapValue(), typeNames)
		if valueMessage := field.MapValue().Message(); valueMessage != nil {
			result.TypeName = string(valueMessage.FullName())
			if reaches(valueMessage.FullName(), containing.FullName(), messages, make(map[protoreflect.FullName]bool)) {
				valueType = "BoundedMessage"
				result.Recursive = true
			}
		} else if valueEnum := field.MapValue().Enum(); valueEnum != nil {
			result.TypeName = string(valueEnum.FullName())
		}
		result.LeanType = "List (" + scalarLeanType(field.MapKey(), typeNames) + " × " + valueType + ")"
		return result
	}
	base := scalarLeanType(field, typeNames)
	if field.Message() != nil {
		result.TypeName = string(field.Message().FullName())
		if reaches(field.Message().FullName(), containing.FullName(), messages, make(map[protoreflect.FullName]bool)) {
			base = "BoundedMessage"
			result.Recursive = true
		}
	} else if field.Enum() != nil {
		result.TypeName = string(field.Enum().FullName())
	}
	if result.Repeated {
		result.LeanType = "List " + parenthesize(base)
	} else if result.Presence {
		result.LeanType = "Option " + parenthesize(base)
	} else {
		result.LeanType = base
	}
	return result
}

func scalarLeanType(field protoreflect.FieldDescriptor, typeNames map[protoreflect.FullName]string) string {
	switch field.Kind() {
	case protoreflect.BoolKind:
		return "Bool"
	case protoreflect.EnumKind:
		return typeNames[field.Enum().FullName()]
	case protoreflect.Int32Kind, protoreflect.Sint32Kind, protoreflect.Sfixed32Kind,
		protoreflect.Int64Kind, protoreflect.Sint64Kind, protoreflect.Sfixed64Kind:
		return "Int"
	case protoreflect.Uint32Kind, protoreflect.Fixed32Kind, protoreflect.Uint64Kind, protoreflect.Fixed64Kind:
		return "Nat"
	case protoreflect.FloatKind, protoreflect.DoubleKind:
		return "Float"
	case protoreflect.StringKind:
		return "String"
	case protoreflect.BytesKind:
		return "RedactedBytes"
	case protoreflect.MessageKind, protoreflect.GroupKind:
		return typeNames[field.Message().FullName()]
	default:
		return ""
	}
}

func reaches(
	from, target protoreflect.FullName,
	messages map[protoreflect.FullName]protoreflect.MessageDescriptor,
	visited map[protoreflect.FullName]bool,
) bool {
	if from == target {
		return true
	}
	if visited[from] {
		return false
	}
	visited[from] = true
	descriptor := messages[from]
	if descriptor == nil {
		return false
	}
	fields := descriptor.Fields()
	for index := 0; index < fields.Len(); index++ {
		field := fields.Get(index)
		var next protoreflect.MessageDescriptor
		if field.IsMap() {
			next = field.MapValue().Message()
		} else {
			next = field.Message()
		}
		if next != nil && reaches(next.FullName(), target, messages, visited) {
			return true
		}
	}
	return false
}

func sortedFullNames[T any](values map[protoreflect.FullName]T) []protoreflect.FullName {
	result := make([]protoreflect.FullName, 0, len(values))
	for name := range values {
		result = append(result, name)
	}
	slices.Sort(result)
	return result
}

func baseTypeName(name protoreflect.FullName) string {
	parts := strings.Split(string(name), ".")
	return goLikeIdentifier(parts[len(parts)-1])
}

func uniqueTypeName(name protoreflect.FullName, counts map[string]int) string {
	base := baseTypeName(name)
	if counts[base] == 1 {
		return base
	}
	return goLikeIdentifier(string(name))
}

func goLikeIdentifier(value string) string {
	parts := strings.FieldsFunc(value, func(character rune) bool {
		return !unicode.IsLetter(character) && !unicode.IsNumber(character)
	})
	var result strings.Builder
	for _, part := range parts {
		characters := []rune(part)
		if len(characters) == 0 {
			continue
		}
		result.WriteRune(unicode.ToUpper(characters[0]))
		result.WriteString(string(characters[1:]))
	}
	return result.String()
}

func leanFieldName(protoName string) string {
	name := goLikeIdentifier(protoName)
	if name != "" {
		characters := []rune(name)
		characters[0] = unicode.ToLower(characters[0])
		name = string(characters)
	}
	reserved := map[string]string{
		"namespace": "namespaceName", "type": "typeName", "match": "matchValue", "where": "whereValue",
		"meta": "metadata",
	}
	if replacement, exists := reserved[name]; exists {
		return replacement
	}
	return name
}

func parenthesize(value string) string {
	if strings.Contains(value, " ") {
		return "(" + value + ")"
	}
	return value
}

func dependencyOrder(messages []messageProjection) []messageProjection {
	byName := make(map[string]messageProjection, len(messages))
	for _, message := range messages {
		byName[message.FullName] = message
	}
	visited := make(map[string]bool, len(messages))
	result := make([]messageProjection, 0, len(messages))
	var visit func(messageProjection)
	visit = func(message messageProjection) {
		if visited[message.FullName] {
			return
		}
		visited[message.FullName] = true
		var dependencies []string
		for _, field := range message.Fields {
			if field.TypeName != "" && !field.Recursive {
				if _, messageDependency := byName[field.TypeName]; messageDependency {
					dependencies = append(dependencies, field.TypeName)
				}
			}
		}
		slices.Sort(dependencies)
		for _, dependency := range dependencies {
			visit(byName[dependency])
		}
		result = append(result, message)
	}
	for _, message := range messages {
		visit(message)
	}
	return result
}
