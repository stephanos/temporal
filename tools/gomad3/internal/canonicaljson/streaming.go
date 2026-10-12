package canonicaljson

import (
	"bufio"
	"encoding"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"sync"
	"unicode"
	"unicode/utf8"
)

// WriteCanonicalJSON writes canonical bytes without a trailing newline. The
// caller must keep value immutable until it returns and discard output on error.
// Ordinary structs, arrays, strings and byte slices use bounded scratch space;
// maps additionally retain sorted keys. JSON hooks and complex field layouts
// use CanonicalJSON for that subtree and can require subtree-sized storage.
// Invalid UTF-8 is rejected even in shared slice aliases or ignored fields.
func WriteCanonicalJSON(writer io.Writer, value any) error {
	if writer == nil {
		return fmt.Errorf("canonical JSON writer is required")
	}
	root := reflect.ValueOf(value)
	if err := validateStreamingStrings(root, make(map[streamVisit]struct{})); err != nil {
		return err
	}
	//gomad:dynamic reviewed 2026-10-12 reviewed model or record values and in-memory streams constrain callback methods
	stream := canonicalStream{output: bufio.NewWriterSize(writer, 4096), active: make(map[streamVisit]struct{})}
	if err := stream.value(root); err != nil {
		return err
	}
	return stream.output.Flush()
}

type streamVisit struct {
	visit
	length int
}

func streamIdentity(value reflect.Value) (streamVisit, bool) {
	switch value.Kind() {
	case reflect.Pointer, reflect.Map, reflect.Slice:
		if value.IsNil() {
			return streamVisit{}, false
		}
		key := streamVisit{visit: visit{typ: value.Type(), pointer: value.Pointer()}}
		if value.Kind() == reflect.Slice {
			key.length = value.Len()
		}
		return key, true
	default:
		return streamVisit{}, false
	}
}

func validateStreamingStrings(value reflect.Value, active map[streamVisit]struct{}) error {
	if !value.IsValid() {
		return nil
	}
	if key, tracked := streamIdentity(value); tracked {
		if _, exists := active[key]; exists {
			return nil
		}
		active[key] = struct{}{}
		defer delete(active, key)
	}
	switch value.Kind() {
	case reflect.String:
		if !utf8.ValidString(value.String()) {
			return fmt.Errorf("JSON string is not valid UTF-8")
		}
	case reflect.Interface, reflect.Pointer:
		if !value.IsNil() {
			return validateStreamingStrings(value.Elem(), active)
		}
	case reflect.Struct:
		for index := 0; index < value.NumField(); index++ {
			if err := validateStreamingStrings(value.Field(index), active); err != nil {
				return err
			}
		}
	case reflect.Map:
		iterator := value.MapRange()
		for iterator.Next() {
			if err := validateStreamingStrings(iterator.Key(), active); err != nil {
				return err
			}
			if err := validateStreamingStrings(iterator.Value(), active); err != nil {
				return err
			}
		}
	case reflect.Array, reflect.Slice:
		//gomad:dynamic reviewed 2026-10-12 reflect runtime type metadata methods inspect memory only
		if value.Type().Elem().Kind() == reflect.Uint8 {
			return nil
		}
		for index := 0; index < value.Len(); index++ {
			if err := validateStreamingStrings(value.Index(index), active); err != nil {
				return err
			}
		}
	default:
	}
	return nil
}

var (
	streamMarshalerType     = reflect.TypeFor[json.Marshaler]()
	streamTextMarshalerType = reflect.TypeFor[encoding.TextMarshaler]()
	streamNumberType        = reflect.TypeFor[json.Number]()
	streamStructCache       = struct {
		mutex   sync.RWMutex
		entries map[reflect.Type]streamFields
	}{entries: make(map[reflect.Type]streamFields)}
)

func hasStreamHook(typ reflect.Type) bool {
	//gomad:dynamic reviewed 2026-10-12 reflect runtime type metadata methods inspect memory only
	return typ.Implements(streamMarshalerType) || typ.Implements(streamTextMarshalerType)
}

type canonicalStream struct {
	output *bufio.Writer
	active map[streamVisit]struct{}
	err    error
}

func (stream *canonicalStream) text(value string) {
	for len(value) > 0 && stream.err == nil {
		size := min(len(value), 4096)
		_, stream.err = stream.output.WriteString(value[:size])
		value = value[size:]
	}
}

func (stream *canonicalStream) bytes(value []byte) {
	for len(value) > 0 && stream.err == nil {
		size := min(len(value), 4096)
		_, stream.err = stream.output.Write(value[:size])
		value = value[size:]
	}
}

func (stream *canonicalStream) quoted(value string) {
	const hexadecimal = "0123456789abcdef"
	stream.text(`"`)
	start := 0
	for index := 0; index < len(value) && stream.err == nil; index++ {
		character := value[index]
		var escape string
		switch character {
		case '\\', '"':
			escape = "\\" + string(character)
		case '\b':
			escape = `\b`
		case '\f':
			escape = `\f`
		case '\n':
			escape = `\n`
		case '\r':
			escape = `\r`
		case '\t':
			escape = `\t`
		default:
			if character < 0x20 {
				escape = "\\u00" + string([]byte{hexadecimal[character>>4], hexadecimal[character&0xf]})
			} else if character == 0xe2 && index+2 < len(value) && value[index+1] == 0x80 && (value[index+2] == 0xa8 || value[index+2] == 0xa9) {
				stream.text(value[start:index])
				if value[index+2] == 0xa8 {
					stream.text(`\u2028`)
				} else {
					stream.text(`\u2029`)
				}
				index += 2
				start = index + 1
				continue
			}
		}
		if escape != "" {
			stream.text(value[start:index])
			stream.text(escape)
			start = index + 1
		}
	}
	stream.text(value[start:])
	stream.text(`"`)
}

func (stream *canonicalStream) legacy(value reflect.Value) error {
	encoded, err := CanonicalJSON(value.Interface())
	if err != nil {
		return err
	}
	stream.bytes(encoded)
	return stream.err
}

func (stream *canonicalStream) value(value reflect.Value) error {
	if stream.err != nil {
		return stream.err
	}
	if !value.IsValid() {
		stream.text("null")
		return stream.err
	}
	if value.Kind() == reflect.Interface {
		if value.IsNil() {
			stream.text("null")
			return stream.err
		}
		return stream.value(value.Elem())
	}
	//gomad:dynamic reviewed 2026-10-12 reflect runtime type metadata methods inspect memory only
	if value.CanAddr() && hasStreamHook(reflect.PointerTo(value.Type())) {
		return stream.legacy(value.Addr())
	}
	if hasStreamHook(value.Type()) || value.Type() == streamNumberType {
		return stream.legacy(value)
	}
	if key, tracked := streamIdentity(value); tracked {
		if _, exists := stream.active[key]; exists {
			//gomad:dynamic reviewed 2026-10-12 reflect runtime type metadata methods inspect memory only
			return fmt.Errorf("encode JSON: %w", &json.UnsupportedValueError{Value: value, Str: "encountered a cycle via " + value.Type().String()})
		}
		stream.active[key] = struct{}{}
		defer delete(stream.active, key)
	}
	switch value.Kind() {
	case reflect.Pointer:
		if value.IsNil() {
			stream.text("null")
		} else {
			return stream.value(value.Elem())
		}
	case reflect.Bool:
		stream.text(strconv.FormatBool(value.Bool()))
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		var buffer [20]byte
		stream.bytes(strconv.AppendInt(buffer[:0], value.Int(), 10))
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		var buffer [20]byte
		stream.bytes(strconv.AppendUint(buffer[:0], value.Uint(), 10))
	case reflect.String:
		stream.quoted(value.String())
	case reflect.Struct:
		return stream.object(value)
	case reflect.Map:
		return stream.mapping(value)
	case reflect.Slice:
		if value.IsNil() {
			stream.text("null")
			return stream.err
		}
		//gomad:dynamic reviewed 2026-10-12 reflect runtime type metadata methods inspect memory only
		if value.Type().Elem().Kind() == reflect.Uint8 {
			//gomad:dynamic reviewed 2026-10-12 reflect runtime type metadata methods inspect memory only
			if hasStreamHook(reflect.PointerTo(value.Type().Elem())) {
				return stream.legacy(value)
			}
			stream.byteString(value.Bytes())
			return stream.err
		}
		return stream.sequence(value)
	case reflect.Array:
		return stream.sequence(value)
	default:
		return stream.legacy(value)
	}
	return stream.err
}

func (stream *canonicalStream) byteString(value []byte) {
	var buffer [4096]byte
	stream.text(`"`)
	for len(value) > 0 && stream.err == nil {
		size := min(len(value), 3072)
		base64.StdEncoding.Encode(buffer[:], value[:size])
		stream.bytes(buffer[:base64.StdEncoding.EncodedLen(size)])
		value = value[size:]
	}
	stream.text(`"`)
}

func (stream *canonicalStream) sequence(value reflect.Value) error {
	stream.text("[")
	for index := 0; index < value.Len(); index++ {
		if index > 0 {
			stream.text(",")
		}
		if err := stream.value(value.Index(index)); err != nil {
			return err
		}
	}
	stream.text("]")
	return stream.err
}

type streamField struct {
	name      string
	index     int
	omitEmpty bool
}

type streamFields struct {
	fields []streamField
	legacy bool
}

func fieldsForStream(typ reflect.Type) streamFields {
	streamStructCache.mutex.RLock()
	cached, exists := streamStructCache.entries[typ]
	streamStructCache.mutex.RUnlock()
	if exists {
		return cached
	}
	result := streamFields{}
	seen := make(map[string]struct{})
	//gomad:dynamic reviewed 2026-10-12 reflect runtime type metadata methods inspect memory only
	for index := 0; index < typ.NumField(); index++ {
		//gomad:dynamic reviewed 2026-10-12 reflect runtime type metadata methods inspect memory only
		field := typ.Field(index)
		if field.Anonymous {
			result.legacy = true
			break
		}
		if !field.IsExported() || field.Tag.Get("json") == "-" {
			continue
		}
		parts := strings.Split(field.Tag.Get("json"), ",")
		name := parts[0]
		if name == "" {
			name = field.Name
		}
		for _, character := range name {
			if !unicode.IsLetter(character) && !unicode.IsDigit(character) && !strings.ContainsRune("!#$%&()*+-./:;<=>?@[]^_{|}~ ", character) {
				result.legacy = true
			}
		}
		if _, duplicate := seen[name]; duplicate {
			result.legacy = true
		}
		seen[name] = struct{}{}
		entry := streamField{name: name, index: index}
		for _, option := range parts[1:] {
			switch option {
			case "omitempty":
				entry.omitEmpty = true
			case "string", "omitzero":
				result.legacy = true
			default:
			}
		}
		result.fields = append(result.fields, entry)
	}
	sort.Slice(result.fields, func(i, j int) bool { return result.fields[i].name < result.fields[j].name })
	streamStructCache.mutex.Lock()
	actual, exists := streamStructCache.entries[typ]
	if !exists {
		streamStructCache.entries[typ] = result
		actual = result
	}
	streamStructCache.mutex.Unlock()
	return actual
}

func (stream *canonicalStream) object(value reflect.Value) error {
	fields := fieldsForStream(value.Type())
	if fields.legacy {
		if value.CanAddr() {
			return stream.legacy(value.Addr())
		}
		return stream.legacy(value)
	}
	stream.text("{")
	first := true
	for _, field := range fields.fields {
		item := value.Field(field.index)
		if field.omitEmpty && emptyStreamValue(item) {
			continue
		}
		if !first {
			stream.text(",")
		}
		first = false
		stream.quoted(field.name)
		stream.text(":")
		if err := stream.value(item); err != nil {
			return err
		}
	}
	stream.text("}")
	return stream.err
}

func emptyStreamValue(value reflect.Value) bool {
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

type streamMapKey struct {
	text  string
	value reflect.Value
}

func (stream *canonicalStream) mapping(value reflect.Value) error {
	//gomad:dynamic reviewed 2026-10-12 reflect runtime type metadata methods inspect memory only
	keyType := value.Type().Key()
	//gomad:dynamic reviewed 2026-10-12 reflect runtime type metadata methods inspect memory only
	if keyType.Kind() != reflect.String && keyType.Implements(streamTextMarshalerType) {
		return stream.legacy(value)
	}
	//gomad:dynamic reviewed 2026-10-12 reflect runtime type metadata methods inspect memory only
	switch keyType.Kind() {
	case reflect.String, reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
	default:
		return stream.legacy(value)
	}
	if value.IsNil() {
		stream.text("null")
		return stream.err
	}
	keys := make([]streamMapKey, 0, value.Len())
	iterator := value.MapRange()
	for iterator.Next() {
		key := iterator.Key()
		var text string
		switch key.Kind() {
		case reflect.String:
			text = key.String()
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			text = strconv.FormatInt(key.Int(), 10)
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
			text = strconv.FormatUint(key.Uint(), 10)
		default:
			return stream.legacy(value)
		}
		keys = append(keys, streamMapKey{text: text, value: key})
	}
	sort.Slice(keys, func(i, j int) bool { return keys[i].text < keys[j].text })
	stream.text("{")
	for index, key := range keys {
		if index > 0 {
			stream.text(",")
		}
		stream.quoted(key.text)
		stream.text(":")
		if err := stream.value(value.MapIndex(key.value)); err != nil {
			return err
		}
	}
	stream.text("}")
	return stream.err
}
