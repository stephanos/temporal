package ir

import (
	"errors"
	"fmt"
	"slices"

	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

type Selector uint8

const (
	Field Selector = iota
	Wildcard
	MapKey
	Presence
	Oneof
)

type PathStep struct {
	Field    protoreflect.FieldDescriptor
	Selector Selector
	Key      *testpilotspb.Value
}

type Path struct {
	source         Type
	steps          []PathStep
	typ            Type
	absent, fanout bool
	limit          int64
	// text is the path's canonical spelling, which identifies it among equal reads.
	text string
}

func (p *Path) Type() Type        { return p.typ }
func (p *Path) Text() string      { return p.text }
func (p *Path) MayBeAbsent() bool { return p.absent }
func (p *Path) Fanout() bool      { return p.fanout }
func (p *Path) Steps() []PathStep {
	steps := slices.Clone(p.steps)
	for i := range steps {
		steps[i].Key = proto.CloneOf(steps[i].Key)
	}
	return steps
}

// CheckFanout charges expansion before allocating or enumerating the next level.
func (p *Path) CheckFanout(current, count int64) (int64, error) {
	if current < 0 || count < 0 || current > p.limit || count > p.limit || (count > 0 && current > p.limit/count) {
		return 0, invalid(LimitExceeded, "path", "fan-out ceiling exceeded")
	}
	return current * count, nil
}

// BindPath parses text in the field path grammar and types it against source. location names where
// text sits in the Case; every rejection is located there and quotes text.
func (c *Catalog) BindPath(source Type, location, text string, limits Limits) (*Path, error) {
	if err := limits.validate(); err != nil {
		return nil, err
	}
	if !c.owns(source) {
		return nil, invalid(TypeMismatch, location, "source type does not belong to this catalog")
	}
	if source.opaque {
		return nil, invalid(Unsupported, location, "capabilities cannot be inspected")
	}
	b := budget{limits: limits}
	if err := b.charge(1, 1, int64(len(text)), location); err != nil {
		return nil, err
	}
	segments, err := parsePath(text)
	if err != nil {
		return nil, pathError(Malformed, location, text, err.Error())
	}
	return c.bindSegments(source, location, text, segments, &b)
}

// bindSegments types parsed segments of text against source; rejections quote the whole text.
func (c *Catalog) bindSegments(source Type, location, text string, segments []pathSegment, b *budget) (*Path, error) {
	result := &Path{source: source, typ: source, limit: b.limits.Fanout, text: formatPath(segments)}
	current := source
	for i, segment := range segments {
		if err := b.charge(int64(i)+1, 1, 0, location); err != nil {
			return nil, err
		}
		step, next, err := c.bindStep(current, segment, i == len(segments)-1, b)
		if err != nil {
			var bound *Error
			if errors.As(err, &bound) {
				return nil, pathError(bound.Category, location, text, bound.Detail)
			}
			return nil, err
		}
		if step.Selector == Wildcard {
			result.fanout = true
		}
		if step.Selector == MapKey {
			result.absent = true
		}
		if step.Selector == Presence {
			result.absent = false
		} else if step.Field.HasPresence() {
			result.absent = true
		}
		result.steps = append(result.steps, step)
		current = next
	}
	if result.fanout {
		if current.cardinality != Singular {
			return nil, pathError(TypeMismatch, location, text, "fan-out cannot produce nested collections")
		}
		current.schema = &testpilotspb.ValueType{Shape: &testpilotspb.ValueType_Repeated{Repeated: &testpilotspb.RepeatedType{Element: proto.CloneOf(current.schema.GetSingular())}}}
		current.cardinality = Repeated
	}
	result.typ = current
	return result, nil
}

func pathError(category ErrorCategory, location, text, detail string) error {
	return invalid(category, location, fmt.Sprintf("path %q: %s", text, detail))
}

func (c *Catalog) bindStep(current Type, segment pathSegment, final bool, b *budget) (PathStep, Type, error) {
	if current.cardinality != Singular || current.message == nil || current.any {
		return PathStep{}, Type{}, invalid(TypeMismatch, "path", "traversal of "+segment.field+" requires a singular unpacked message")
	}
	var field protoreflect.FieldDescriptor
	step := PathStep{}
	if segment.selector == Oneof {
		group := current.message.Oneofs().ByName(protoreflect.Name(segment.field))
		if group == nil {
			return PathStep{}, Type{}, invalid(Unknown, "path", "unknown oneof group "+segment.field)
		}
		field = group.Fields().ByName(protoreflect.Name(segment.member))
		if field == nil {
			return PathStep{}, Type{}, invalid(Unknown, "path", "oneof "+segment.field+" has no member "+segment.member)
		}
	} else {
		field = current.message.Fields().ByName(protoreflect.Name(segment.field))
		if field == nil {
			return PathStep{}, Type{}, invalid(Unknown, "path", "unknown field "+segment.field)
		}
	}
	step.Field = field
	step.Selector = segment.selector
	next := c.fieldType(field)
	switch segment.selector {
	case Field, Oneof:
	case Wildcard:
		if !field.IsList() {
			return PathStep{}, Type{}, invalid(TypeMismatch, "path", "wildcard requires a repeated field, not "+segment.field)
		}
		next = next.Element()
	case MapKey:
		if !field.IsMap() {
			return PathStep{}, Type{}, invalid(TypeMismatch, "path", "map key selector requires a map, not "+segment.field)
		}
		key, err := segment.key.value(next.key)
		if err != nil {
			return PathStep{}, Type{}, err
		}
		if err := c.checkLiteral(key, c.scalarType(next.key), b, 1); err != nil {
			var literal *Error
			if errors.As(err, &literal) && literal.Category == LimitExceeded {
				return PathStep{}, Type{}, err
			}
			return PathStep{}, Type{}, invalid(TypeMismatch, "path", "map key "+segment.key.String()+" is not a canonical "+EnumName(next.key)+" key")
		}
		step.Key = key
		next = next.Element()
	case Presence:
		if !field.HasPresence() || !final {
			return PathStep{}, Type{}, invalid(TypeMismatch, "path", "presence requires a final presence-bearing field, not "+segment.field)
		}
		next = c.scalarType(testpilotspb.SCALAR_KIND_BOOLEAN)
	default:
		return PathStep{}, Type{}, invalid(Unsupported, "path", "unknown selector")
	}
	return step, next, nil
}

func (c *Catalog) owns(typ Type) bool {
	return (typ.schema != nil || typ.opaque) && typ.catalog != nil && c != nil && c.identity == typ.catalog.identity
}

func (c *Catalog) fieldType(field protoreflect.FieldDescriptor) Type {
	if field.IsMap() {
		value := c.fieldType(field.MapValue())
		value.cardinality = Map
		value.key = scalarKind(field.MapKey().Kind())
		value.schema = &testpilotspb.ValueType{Shape: &testpilotspb.ValueType_Map{Map: &testpilotspb.MapType{Key: &testpilotspb.ScalarType{Kind: value.key}, Value: value.schema.GetSingular()}}}
		return value
	}
	result := c.scalarType(scalarKind(field.Kind()))
	if field.Enum() != nil {
		result.enumeration = field.Enum()
		result.schema.GetSingular().Type = &testpilotspb.SingularType_Enumeration{Enumeration: &testpilotspb.NamedType{ProtobufType: string(field.Enum().FullName())}}
	}
	if field.Message() != nil {
		if field.Message().FullName() == "google.protobuf.Any" {
			result.any = true
			result.schema.GetSingular().Type = &testpilotspb.SingularType_Any{Any: &testpilotspb.AnyType{}}
		} else {
			result.message = field.Message()
			result.schema.GetSingular().Type = &testpilotspb.SingularType_Message{Message: &testpilotspb.NamedType{ProtobufType: string(field.Message().FullName())}}
		}
	}
	if field.IsList() {
		result.cardinality = Repeated
		result.schema = &testpilotspb.ValueType{Shape: &testpilotspb.ValueType_Repeated{Repeated: &testpilotspb.RepeatedType{Element: result.schema.GetSingular()}}}
	}
	return result
}

func scalarKind(kind protoreflect.Kind) testpilotspb.ScalarKind {
	switch kind {
	case protoreflect.StringKind:
		return testpilotspb.SCALAR_KIND_TEXT
	case protoreflect.BoolKind:
		return testpilotspb.SCALAR_KIND_BOOLEAN
	case protoreflect.BytesKind:
		return testpilotspb.SCALAR_KIND_BYTES
	case protoreflect.Int32Kind:
		return testpilotspb.SCALAR_KIND_INT32
	case protoreflect.Int64Kind:
		return testpilotspb.SCALAR_KIND_INT64
	case protoreflect.Uint32Kind:
		return testpilotspb.SCALAR_KIND_UINT32
	case protoreflect.Uint64Kind:
		return testpilotspb.SCALAR_KIND_UINT64
	case protoreflect.Sint32Kind:
		return testpilotspb.SCALAR_KIND_SINT32
	case protoreflect.Sint64Kind:
		return testpilotspb.SCALAR_KIND_SINT64
	case protoreflect.Fixed32Kind:
		return testpilotspb.SCALAR_KIND_FIXED32
	case protoreflect.Fixed64Kind:
		return testpilotspb.SCALAR_KIND_FIXED64
	case protoreflect.Sfixed32Kind:
		return testpilotspb.SCALAR_KIND_SFIXED32
	case protoreflect.Sfixed64Kind:
		return testpilotspb.SCALAR_KIND_SFIXED64
	case protoreflect.FloatKind:
		return testpilotspb.SCALAR_KIND_FLOAT
	case protoreflect.DoubleKind:
		return testpilotspb.SCALAR_KIND_DOUBLE
	default:
		return testpilotspb.SCALAR_KIND_UNSPECIFIED
	}
}

func (p *Path) Conflicts(right *Path) bool {
	a, b := p.steps, right.steps
	for i := 0; i < min(len(a), len(b)); i++ {
		if a[i].Field != b[i].Field {
			group := a[i].Field.ContainingOneof()
			return group != nil && group == b[i].Field.ContainingOneof()
		}
		if a[i].Selector == MapKey && b[i].Selector == MapKey && !proto.Equal(a[i].Key, b[i].Key) {
			return false
		}
		if a[i].Selector != b[i].Selector && a[i].Selector != Oneof && b[i].Selector != Oneof {
			return true
		}
	}
	return true
}
