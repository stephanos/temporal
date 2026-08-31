package runevaluation

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"

	"go.temporal.io/server/tools/umpire/internal/artifactv2"
)

const canonicalJSONWriteChunk = 32 << 10

var (
	naturalType         = reflect.TypeFor[artifactv2.Natural]()
	jsonNumberType      = reflect.TypeFor[json.Number]()
	checkerRequestType  = reflect.TypeFor[checkerRequest]()
	checkerResponseType = reflect.TypeFor[checkerResponse]()
)

type canonicalJSONWriter struct {
	writer io.Writer
	depth  int
}

type canonicalJSONField struct {
	name  string
	value reflect.Value
}

func writeCanonicalPrettyJSON(writer io.Writer, value any) error {
	encoder := canonicalJSONWriter{writer: writer}
	if err := encoder.writeValue(reflect.ValueOf(value)); err != nil {
		return err
	}
	return encoder.writeString("\n")
}

func (encoder *canonicalJSONWriter) writeValue(value reflect.Value) error {
	return encoder.writeValueWithObjectOrder(value, false)
}

func (encoder *canonicalJSONWriter) writeValueWithObjectOrder(
	value reflect.Value,
	sortObjectFields bool,
) error {
	for value.IsValid() && (value.Kind() == reflect.Interface || value.Kind() == reflect.Pointer) {
		if value.IsNil() {
			return encoder.writeString("null")
		}
		value = value.Elem()
	}
	if !value.IsValid() {
		return encoder.writeString("null")
	}
	if value.Type() == naturalType {
		return encoder.writeNatural(value.String())
	}
	if value.Type() == jsonNumberType {
		return encoder.writeNumber(value.String())
	}

	switch value.Kind() {
	case reflect.Bool:
		return encoder.writeString(strconv.FormatBool(value.Bool()))
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return encoder.writeString(strconv.FormatInt(value.Int(), 10))
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return encoder.writeString(strconv.FormatUint(value.Uint(), 10))
	case reflect.String:
		return encoder.writeJSONString(value.String())
	case reflect.Struct:
		return encoder.writeStruct(value, sortObjectFields)
	case reflect.Slice, reflect.Array:
		if value.Kind() == reflect.Slice && value.IsNil() {
			return encoder.writeString("null")
		}
		return encoder.writeArray(value, sortObjectFields)
	default:
		return fmt.Errorf("unsupported checker JSON type %s", value.Type())
	}
}

func (encoder *canonicalJSONWriter) writeStruct(value reflect.Value, sortObjectFields bool) error {
	fields := canonicalJSONFields(value)
	if sortObjectFields {
		slices.SortFunc(fields, func(left, right canonicalJSONField) int {
			return strings.Compare(left.name, right.name)
		})
	}
	if len(fields) == 0 {
		return encoder.writeString("{}")
	}
	if err := encoder.writeString("{"); err != nil {
		return err
	}
	encoder.depth++
	for index, field := range fields {
		sortNestedObjectFields := sortObjectFields || checkerProtocolJSONField(value.Type(), field.name)
		if err := encoder.writeStructField(field, index != 0, sortNestedObjectFields); err != nil {
			return err
		}
	}
	encoder.depth--
	if err := encoder.writeIndent(); err != nil {
		return err
	}
	return encoder.writeString("}")
}

func canonicalJSONFields(value reflect.Value) []canonicalJSONField {
	fields := make([]canonicalJSONField, 0, value.NumField())
	for index := 0; index < value.NumField(); index++ {
		field := value.Type().Field(index)
		if field.PkgPath != "" {
			continue
		}
		name, options, _ := strings.Cut(field.Tag.Get("json"), ",")
		if name == "-" {
			continue
		}
		if name == "" {
			name = field.Name
		}
		memberValue := value.Field(index)
		if strings.Contains(options, "omitempty") && canonicalJSONEmptyValue(memberValue) {
			continue
		}
		fields = append(fields, canonicalJSONField{name: name, value: memberValue})
	}
	return fields
}

func checkerProtocolJSONField(owner reflect.Type, name string) bool {
	if owner == checkerRequestType {
		switch name {
		case "phaseOutcomes", "controlAttempts", "sourceClosures", "sources", "facts",
			"runKnownGaps", "rawEvidenceKnownGaps":
			return true
		}
	}
	if owner == checkerResponseType {
		switch name {
		case "implementationLink", "evidenceBackedModelTrace", "evidenceLinks", "dispositions", "diagnostics",
			"observationKnownGaps", "propertyVerdicts", "querySummary", "resultKnownGaps":
			return true
		}
	}
	return false
}

func (encoder *canonicalJSONWriter) writeStructField(
	field canonicalJSONField,
	preceded bool,
	sortObjectFields bool,
) error {
	if preceded {
		if err := encoder.writeString(","); err != nil {
			return err
		}
	}
	if err := encoder.writeIndent(); err != nil {
		return err
	}
	if err := encoder.writeJSONString(field.name); err != nil {
		return err
	}
	if err := encoder.writeString(": "); err != nil {
		return err
	}
	return encoder.writeValueWithObjectOrder(field.value, sortObjectFields)
}

func (encoder *canonicalJSONWriter) writeArray(value reflect.Value, sortObjectFields bool) error {
	if value.Len() == 0 {
		return encoder.writeString("[]")
	}
	if err := encoder.writeString("["); err != nil {
		return err
	}
	encoder.depth++
	for index := 0; index < value.Len(); index++ {
		if index != 0 {
			if err := encoder.writeString(","); err != nil {
				return err
			}
		}
		if err := encoder.writeIndent(); err != nil {
			return err
		}
		if err := encoder.writeValueWithObjectOrder(value.Index(index), sortObjectFields); err != nil {
			return err
		}
	}
	encoder.depth--
	if err := encoder.writeIndent(); err != nil {
		return err
	}
	return encoder.writeString("]")
}

func (encoder *canonicalJSONWriter) writeIndent() error {
	if err := encoder.writeString("\n"); err != nil {
		return err
	}
	for range encoder.depth {
		if err := encoder.writeString("  "); err != nil {
			return err
		}
	}
	return nil
}

func (encoder *canonicalJSONWriter) writeJSONString(value string) error {
	if err := encoder.writeString("\""); err != nil {
		return err
	}
	start := 0
	for index := 0; index < len(value); {
		next, replacement := canonicalJSONStringReplacement(value, index)
		if replacement == "" {
			index = next
			continue
		}
		if err := encoder.writeString(value[start:index]); err != nil {
			return err
		}
		if err := encoder.writeString(replacement); err != nil {
			return err
		}
		index = next
		start = index
	}
	if err := encoder.writeString(value[start:]); err != nil {
		return err
	}
	return encoder.writeString("\"")
}

func canonicalJSONStringReplacement(value string, index int) (int, string) {
	character := value[index]
	if character < utf8.RuneSelf {
		if character >= 0x20 && character != '\\' && character != '"' {
			return index + 1, ""
		}
		return index + 1, canonicalJSONEscape(character)
	}
	decoded, size := utf8.DecodeRuneInString(value[index:])
	if decoded == utf8.RuneError && size == 1 {
		return index + 1, "\ufffd"
	}
	if decoded == '\u2028' {
		return index + size, "\\u2028"
	}
	if decoded == '\u2029' {
		return index + size, "\\u2029"
	}
	return index + size, ""
}

func (encoder *canonicalJSONWriter) writeNatural(value string) error {
	if value == "" || len(value) > 1 && value[0] == '0' {
		return errors.New("checker JSON natural is invalid")
	}
	for _, character := range []byte(value) {
		if character < '0' || character > '9' {
			return errors.New("checker JSON natural is invalid")
		}
	}
	return encoder.writeString(value)
}

func (encoder *canonicalJSONWriter) writeNumber(value string) error {
	digits := strings.TrimPrefix(value, "-")
	if digits == "" || len(digits) > 1 && digits[0] == '0' {
		return errors.New("checker JSON number is invalid")
	}
	for _, character := range []byte(digits) {
		if character < '0' || character > '9' {
			return errors.New("checker JSON number is invalid")
		}
	}
	return encoder.writeString(value)
}

func (encoder *canonicalJSONWriter) writeString(value string) error {
	for len(value) != 0 {
		chunkBytes := min(len(value), canonicalJSONWriteChunk)
		written, err := io.WriteString(encoder.writer, value[:chunkBytes])
		if err != nil {
			return err
		}
		if written != chunkBytes {
			return io.ErrShortWrite
		}
		value = value[chunkBytes:]
	}
	return nil
}

func canonicalJSONEscape(character byte) string {
	switch character {
	case '\\', '"':
		return "\\" + string(character)
	case '\b':
		return "\\b"
	case '\f':
		return "\\f"
	case '\n':
		return "\\n"
	case '\r':
		return "\\r"
	case '\t':
		return "\\t"
	default:
		const hexadecimal = "0123456789abcdef"
		return "\\u00" + string([]byte{hexadecimal[character>>4], hexadecimal[character&0xf]})
	}
}

func canonicalJSONEmptyValue(value reflect.Value) bool {
	switch value.Kind() {
	case reflect.Array, reflect.Map, reflect.Slice, reflect.String:
		return value.Len() == 0
	case reflect.Bool, reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr,
		reflect.Float32, reflect.Float64, reflect.Interface, reflect.Pointer:
		return value.IsZero()
	default:
		return false
	}
}
