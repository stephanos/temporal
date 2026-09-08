package main

import (
	"fmt"
	"math"
	"slices"
	"strings"

	"google.golang.org/protobuf/reflect/protoreflect"
)

func concreteMessageShape(message protoreflect.MessageDescriptor) string {
	fields := make([]protoreflect.FieldDescriptor, 0, message.Fields().Len())
	for i := 0; i < message.Fields().Len(); i++ {
		fields = append(fields, message.Fields().Get(i))
	}
	slices.SortFunc(fields, func(a, b protoreflect.FieldDescriptor) int { return int(a.Number() - b.Number()) })
	items := make([]string, 0, len(fields))
	for _, field := range fields {
		cardinality := ".singular"
		typ := concreteSingular(field)
		presence := ".optional"
		switch {
		case field.IsMap():
			cardinality = "(.map " + concreteSingular(field.MapKey()) + ")"
			typ = concreteSingular(field.MapValue())
		case field.IsList():
			cardinality = ".repeated"
		case field.ContainingOneof() != nil && !field.ContainingOneof().IsSynthetic():
			presence = fmt.Sprintf("(.oneof %q)", field.ContainingOneof().Name())
		case field.Cardinality() == protoreflect.Required:
			presence = ".required"
		case !field.HasPresence():
			presence = "(.implicit " + concreteDefault(field) + ")"
		default:
		}
		defaultValue := "none"
		if field.HasDefault() {
			defaultValue = "(some " + concreteDefault(field) + ")"
		}
		items = append(items, fmt.Sprintf("⟨%d, %q, %s, %s, %s, %s⟩", field.Number(), field.Name(), typ, cardinality, presence, defaultValue))
	}
	unsupported := "none"
	if message.ExtensionRanges().Len() != 0 || message.Extensions().Len() != 0 {
		unsupported = "(some \"message extensions\")"
	}
	return fmt.Sprintf("(.message [%s] %s)", strings.Join(items, ", "), unsupported)
}

func concreteEnumShape(enum protoreflect.EnumDescriptor) string {
	numbers := make([]int32, 0, enum.Values().Len())
	for i := 0; i < enum.Values().Len(); i++ {
		numbers = append(numbers, int32(enum.Values().Get(i).Number()))
	}
	slices.Sort(numbers)
	numbers = slices.Compact(numbers)
	items := make([]string, 0, len(numbers))
	for _, n := range numbers {
		items = append(items, fmt.Sprint(n))
	}
	return fmt.Sprintf("(.enumeration %t [%s])", enum.IsClosed(), strings.Join(items, ", "))
}

func concreteSingular(field protoreflect.FieldDescriptor) string {
	switch field.Kind() {
	case protoreflect.BoolKind:
		return ".boolean"
	case protoreflect.StringKind:
		return ".text"
	case protoreflect.BytesKind:
		return ".bytes"
	case protoreflect.EnumKind:
		return fmt.Sprintf("(.enumeration %q)", field.Enum().FullName())
	case protoreflect.MessageKind:
		return fmt.Sprintf("(.message %q)", field.Message().FullName())
	case protoreflect.GroupKind:
		return "(.unsupported \"group\")"
	case protoreflect.FloatKind, protoreflect.DoubleKind:
		return fmt.Sprintf("(.floating %t)", field.Kind() == protoreflect.DoubleKind)
	default:
		return fmt.Sprintf("(.integer .%s)", field.Kind())
	}
}

func concreteDefault(field protoreflect.FieldDescriptor) string {
	switch field.Kind() {
	case protoreflect.BoolKind:
		return fmt.Sprintf("(.boolean %t)", field.Default().Bool())
	case protoreflect.StringKind:
		return fmt.Sprintf("(.text %s)", leanString(field.Default().String()))
	case protoreflect.BytesKind:
		items := make([]string, 0, len(field.Default().Bytes()))
		for _, b := range field.Default().Bytes() {
			items = append(items, fmt.Sprint(b))
		}
		return fmt.Sprintf("(.bytes [%s])", strings.Join(items, ", "))
	case protoreflect.EnumKind:
		return fmt.Sprintf("(.enumeration %q (%d))", field.Enum().FullName(), field.Default().Enum())
	case protoreflect.FloatKind:
		return fmt.Sprintf("(.floating false %d)", math.Float32bits(float32(field.Default().Float())))
	case protoreflect.DoubleKind:
		return fmt.Sprintf("(.floating true %d)", math.Float64bits(field.Default().Float()))
	case protoreflect.Uint32Kind, protoreflect.Uint64Kind, protoreflect.Fixed32Kind, protoreflect.Fixed64Kind:
		return fmt.Sprintf("(.integer .%s %d)", field.Kind(), field.Default().Uint())
	default:
		return fmt.Sprintf("(.integer .%s (%d))", field.Kind(), field.Default().Int())
	}
}

func leanString(value string) string {
	var rendered strings.Builder
	rendered.Grow(len(value) + 2)
	rendered.WriteByte('"')
	for _, character := range value {
		switch character {
		case '"':
			rendered.WriteString(`\"`)
		case '\\':
			rendered.WriteString(`\\`)
		case '\n':
			rendered.WriteString(`\n`)
		case '\r':
			rendered.WriteString(`\r`)
		case '\t':
			rendered.WriteString(`\t`)
		default:
			if character < 0x20 || character == 0x7f {
				fmt.Fprintf(&rendered, `\u%04x`, character)
			} else {
				rendered.WriteRune(character)
			}
		}
	}
	rendered.WriteByte('"')
	return rendered.String()
}
