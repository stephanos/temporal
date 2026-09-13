package ir

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	testpilotspb "go.temporal.io/server/api/testpilot/v1"
)

// pathSegment is one parsed segment of a field path: a protobuf field name (a oneof's name when the
// selector is Oneof) and at most one selector.
type pathSegment struct {
	field    string
	selector Selector
	// member is the selected oneof member's field name.
	member string
	key    pathKey
}

type pathKeyKind uint8

const (
	textKey pathKeyKind = iota + 1
	integerKey
	booleanKey
)

// pathKey is a map key as a path spells it. Its kind is syntactic; binding types it by the map's key
// kind.
type pathKey struct {
	kind pathKeyKind
	// text is the decoded text of a text key, the base-10 digits of an integer key, or true or false.
	text string
}

// parsePath reads text in the field path grammar:
//
//	path     = "" | segment { "." segment }
//	segment  = name [ "<" name ">" | "[*]" | "[" key "]" | "?" ]
//	key      = JSON string | [ "-" ] digit { digit } | "true" | "false"
//	name     = ( letter | "_" ) { letter | digit | "_" }
func parsePath(text string) ([]pathSegment, error) {
	if text == "" {
		return nil, nil
	}
	var segments []pathSegment
	for position := 0; ; {
		segment, next, err := parseSegment(text, position)
		if err != nil {
			return nil, err
		}
		segments = append(segments, segment)
		if next == len(text) {
			return segments, nil
		}
		if text[next] != '.' {
			return nil, fmt.Errorf("unknown selector %q after %s at byte %d", text[next:next+1], segment.field, next)
		}
		position = next + 1
	}
}

func parseSegment(text string, position int) (pathSegment, int, error) {
	name, next := scanName(text, position)
	if name == "" {
		if next == len(text) || text[next] == '.' {
			return pathSegment{}, 0, fmt.Errorf("empty segment at byte %d", position)
		}
		return pathSegment{}, 0, fmt.Errorf("segment at byte %d does not start with a field name", position)
	}
	segment := pathSegment{field: name}
	if next == len(text) {
		return segment, next, nil
	}
	switch text[next] {
	case '<':
		member, end := scanName(text, next+1)
		if member == "" || end == len(text) || text[end] != '>' {
			return pathSegment{}, 0, fmt.Errorf("unterminated oneof member of %s at byte %d", name, next)
		}
		segment.selector, segment.member = Oneof, member
		return segment, end + 1, nil
	case '?':
		segment.selector = Presence
		return segment, next + 1, nil
	case '[':
		return parseBracket(text, next, segment)
	default:
		return segment, next, nil
	}
}

// parseBracket reads a wildcard or map key selector opening at text[open].
func parseBracket(text string, open int, segment pathSegment) (pathSegment, int, error) {
	if open+1 < len(text) && text[open+1] == '"' {
		end, err := scanJSONString(text, open+1)
		if err != nil {
			return pathSegment{}, 0, fmt.Errorf("map key of %s at byte %d: %w", segment.field, open, err)
		}
		var decoded string
		if err := json.Unmarshal([]byte(text[open+1:end]), &decoded); err != nil {
			return pathSegment{}, 0, fmt.Errorf("map key of %s at byte %d is not a JSON string", segment.field, open)
		}
		if end == len(text) || text[end] != ']' {
			return pathSegment{}, 0, fmt.Errorf("unterminated map key of %s at byte %d", segment.field, open)
		}
		segment.selector, segment.key = MapKey, pathKey{kind: textKey, text: decoded}
		return segment, end + 1, nil
	}
	closing := strings.IndexByte(text[open:], ']')
	if closing < 0 {
		return pathSegment{}, 0, fmt.Errorf("unterminated selector of %s at byte %d", segment.field, open)
	}
	token := text[open+1 : open+closing]
	switch {
	case token == "*":
		segment.selector = Wildcard
	case token == "true" || token == "false":
		segment.selector, segment.key = MapKey, pathKey{kind: booleanKey, text: token}
	case isInteger(token):
		segment.selector, segment.key = MapKey, pathKey{kind: integerKey, text: token}
	default:
		return pathSegment{}, 0, fmt.Errorf("unknown selector [%s] of %s", token, segment.field)
	}
	return segment, open + closing + 1, nil
}

func scanName(text string, position int) (string, int) {
	end := position
	for end < len(text) {
		c := text[end]
		if c == '_' || c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || end > position && c >= '0' && c <= '9' {
			end++
			continue
		}
		break
	}
	return text[position:end], end
}

// scanJSONString returns the end of the JSON string opening at text[open], just past its closing
// quote.
func scanJSONString(text string, open int) (int, error) {
	for index := open + 1; index < len(text); index++ {
		switch text[index] {
		case '\\':
			index++
		case '"':
			return index + 1, nil
		default:
		}
	}
	return 0, errors.New("unterminated map key")
}

func isInteger(token string) bool {
	digits := strings.TrimPrefix(token, "-")
	if digits == "" {
		return false
	}
	for index := range len(digits) {
		if digits[index] < '0' || digits[index] > '9' {
			return false
		}
	}
	return true
}

// formatPath spells segments canonically: the one spelling parsePath reads back to them. A text key is
// quoted escaping only the quote, the backslash and control characters, as Testpilot.Authoring.Path
// spells it.
func formatPath(segments []pathSegment) string {
	var out strings.Builder
	for index, segment := range segments {
		if index > 0 {
			out.WriteByte('.')
		}
		out.WriteString(segment.field)
		switch segment.selector {
		case Wildcard:
			out.WriteString("[*]")
		case MapKey:
			out.WriteString("[" + segment.key.String() + "]")
		case Presence:
			out.WriteByte('?')
		case Oneof:
			out.WriteString("<" + segment.member + ">")
		default:
		}
	}
	return out.String()
}

// String spells the key as a path selector holds it.
func (k pathKey) String() string {
	if k.kind != textKey {
		return k.text
	}
	var out strings.Builder
	out.WriteByte('"')
	for _, r := range k.text {
		switch {
		case r == '"':
			out.WriteString(`\"`)
		case r == '\\':
			out.WriteString(`\\`)
		case r == '\n':
			out.WriteString(`\n`)
		case r == '\r':
			out.WriteString(`\r`)
		case r < 0x20:
			fmt.Fprintf(&out, `\u%04x`, r)
		default:
			out.WriteRune(r)
		}
	}
	out.WriteByte('"')
	return out.String()
}

// value types the key by the map's key kind.
func (k pathKey) value(kind testpilotspb.ScalarKind) (*testpilotspb.Value, error) {
	mismatch := invalid(TypeMismatch, "path", "map key "+k.String()+" does not match the map's "+EnumName(kind)+" keys")
	switch kind {
	case testpilotspb.SCALAR_KIND_TEXT:
		if k.kind == textKey {
			return &testpilotspb.Value{Value: &testpilotspb.Value_TextValue{TextValue: k.text}}, nil
		}
	case testpilotspb.SCALAR_KIND_BOOLEAN:
		if k.kind == booleanKey {
			return &testpilotspb.Value{Value: &testpilotspb.Value_BoolValue{BoolValue: k.text == "true"}}, nil
		}
	case testpilotspb.SCALAR_KIND_INT32, testpilotspb.SCALAR_KIND_INT64, testpilotspb.SCALAR_KIND_SINT32, testpilotspb.SCALAR_KIND_SINT64, testpilotspb.SCALAR_KIND_SFIXED32, testpilotspb.SCALAR_KIND_SFIXED64:
		if k.kind == integerKey {
			return &testpilotspb.Value{Value: &testpilotspb.Value_SignedIntegerValue{SignedIntegerValue: k.text}}, nil
		}
	case testpilotspb.SCALAR_KIND_UINT32, testpilotspb.SCALAR_KIND_UINT64, testpilotspb.SCALAR_KIND_FIXED32, testpilotspb.SCALAR_KIND_FIXED64:
		if k.kind == integerKey {
			return &testpilotspb.Value{Value: &testpilotspb.Value_UnsignedIntegerValue{UnsignedIntegerValue: k.text}}, nil
		}
	default:
	}
	return nil, mismatch
}
