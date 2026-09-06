package ir

import (
	"slices"

	testpilotpb "go.temporal.io/server/api/testpilot/v1"
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
	Key      *testpilotpb.Value
}

type Path struct {
	source         Type
	steps          []PathStep
	typ            Type
	absent, fanout bool
	limit          int64
}

func (p *Path) Type() Type        { return p.typ }
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

func (c *Catalog) BindPath(source Type, path *testpilotpb.FieldPath, limits Limits) (*Path, error) {
	if err := limits.validate(); err != nil {
		return nil, err
	}
	if !c.owns(source) {
		return nil, invalid(TypeMismatch, "path", "source type does not belong to this catalog")
	}
	if source.opaque {
		return nil, invalid(Unsupported, "path", "capabilities cannot be inspected")
	}
	if path == nil {
		return nil, invalid(Malformed, "path", "path is required")
	}
	b := budget{limits: limits}
	if err := inspectSurface(path.ProtoReflect(), &b, "path"); err != nil {
		return nil, err
	}
	result := &Path{source: source, typ: source, limit: limits.Fanout}
	current := source
	for i, segment := range path.Segments {
		if err := b.charge(int64(i)+1, 1, 0, "path"); err != nil {
			return nil, err
		}
		step, next, err := c.bindStep(current, segment, i == len(path.Segments)-1, &b)
		if err != nil {
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
			return nil, invalid(TypeMismatch, "path", "fan-out cannot produce nested collections")
		}
		current.schema = &testpilotpb.ValueType{Shape: &testpilotpb.ValueType_Repeated{Repeated: &testpilotpb.RepeatedType{Element: proto.CloneOf(current.schema.GetSingular())}}}
		current.cardinality = Repeated
	}
	result.typ = current
	return result, nil
}

func (c *Catalog) bindStep(current Type, segment *testpilotpb.FieldPathSegment, final bool, b *budget) (PathStep, Type, error) {
	if segment == nil || (segment.Selector != nil && missing(segment.Selector)) {
		return PathStep{}, Type{}, invalid(Malformed, "path", "nil path segment")
	}
	if current.cardinality != Singular || current.message == nil || current.any {
		return PathStep{}, Type{}, invalid(TypeMismatch, "path", "traversal requires a singular unpacked message")
	}
	var field protoreflect.FieldDescriptor
	step := PathStep{}
	if selected, ok := segment.Selector.(*testpilotpb.FieldPathSegment_Oneof); ok {
		group := current.message.Oneofs().ByName(protoreflect.Name(segment.Field))
		if group == nil || selected.Oneof == nil {
			return PathStep{}, Type{}, invalid(Unknown, "path", "unknown oneof group")
		}
		field = group.Fields().ByName(protoreflect.Name(selected.Oneof.SelectedField))
		step.Selector = Oneof
	} else {
		field = current.message.Fields().ByName(protoreflect.Name(segment.Field))
	}
	if field == nil {
		return PathStep{}, Type{}, invalid(Unknown, "path", "unknown field or selected oneof member")
	}
	step.Field = field
	next := c.fieldType(field)
	switch selection := segment.Selector.(type) {
	case nil:
	case *testpilotpb.FieldPathSegment_Oneof:
	case *testpilotpb.FieldPathSegment_Repeated:
		if !field.IsList() {
			return PathStep{}, Type{}, invalid(TypeMismatch, "path", "wildcard requires a repeated field")
		}
		step.Selector = Wildcard
		next = next.Element()
	case *testpilotpb.FieldPathSegment_MapKey:
		if !field.IsMap() || selection.MapKey == nil {
			return PathStep{}, Type{}, invalid(TypeMismatch, "path", "map-key selector requires a map")
		}
		if err := c.checkLiteral(selection.MapKey.Key, c.scalarType(next.key), b, 1); err != nil {
			return PathStep{}, Type{}, err
		}
		step.Selector = MapKey
		step.Key = proto.CloneOf(selection.MapKey.Key)
		next = next.Element()
	case *testpilotpb.FieldPathSegment_Presence:
		if !field.HasPresence() || !final {
			return PathStep{}, Type{}, invalid(TypeMismatch, "path", "presence requires a final presence-bearing field")
		}
		step.Selector = Presence
		next = c.scalarType(testpilotpb.SCALAR_KIND_BOOLEAN)
	default:
		return PathStep{}, Type{}, invalid(Unsupported, "path", "unknown selector")
	}
	return step, next, nil
}

func (c *Catalog) owns(typ Type) bool {
	return typ.schema != nil && typ.catalog != nil && c != nil && c.identity == typ.catalog.identity
}

func (c *Catalog) fieldType(field protoreflect.FieldDescriptor) Type {
	if field.IsMap() {
		value := c.fieldType(field.MapValue())
		value.cardinality = Map
		value.key = scalarKind(field.MapKey().Kind())
		value.schema = &testpilotpb.ValueType{Shape: &testpilotpb.ValueType_Map{Map: &testpilotpb.MapType{Key: &testpilotpb.ScalarType{Kind: value.key}, Value: value.schema.GetSingular()}}}
		return value
	}
	result := c.scalarType(scalarKind(field.Kind()))
	if field.Enum() != nil {
		result.enumeration = field.Enum()
		result.schema.GetSingular().Type = &testpilotpb.SingularType_Enumeration{Enumeration: &testpilotpb.NamedType{ProtobufType: string(field.Enum().FullName())}}
	}
	if field.Message() != nil {
		if field.Message().FullName() == "google.protobuf.Any" {
			result.any = true
			result.schema.GetSingular().Type = &testpilotpb.SingularType_Any{Any: &testpilotpb.AnyType{}}
		} else {
			result.message = field.Message()
			result.schema.GetSingular().Type = &testpilotpb.SingularType_Message{Message: &testpilotpb.NamedType{ProtobufType: string(field.Message().FullName())}}
		}
	}
	if field.IsList() {
		result.cardinality = Repeated
		result.schema = &testpilotpb.ValueType{Shape: &testpilotpb.ValueType_Repeated{Repeated: &testpilotpb.RepeatedType{Element: result.schema.GetSingular()}}}
	}
	return result
}

func scalarKind(kind protoreflect.Kind) testpilotpb.ScalarKind {
	switch kind {
	case protoreflect.StringKind:
		return testpilotpb.SCALAR_KIND_TEXT
	case protoreflect.BoolKind:
		return testpilotpb.SCALAR_KIND_BOOLEAN
	case protoreflect.BytesKind:
		return testpilotpb.SCALAR_KIND_BYTES
	case protoreflect.Int32Kind:
		return testpilotpb.SCALAR_KIND_INT32
	case protoreflect.Int64Kind:
		return testpilotpb.SCALAR_KIND_INT64
	case protoreflect.Uint32Kind:
		return testpilotpb.SCALAR_KIND_UINT32
	case protoreflect.Uint64Kind:
		return testpilotpb.SCALAR_KIND_UINT64
	case protoreflect.Sint32Kind:
		return testpilotpb.SCALAR_KIND_SINT32
	case protoreflect.Sint64Kind:
		return testpilotpb.SCALAR_KIND_SINT64
	case protoreflect.Fixed32Kind:
		return testpilotpb.SCALAR_KIND_FIXED32
	case protoreflect.Fixed64Kind:
		return testpilotpb.SCALAR_KIND_FIXED64
	case protoreflect.Sfixed32Kind:
		return testpilotpb.SCALAR_KIND_SFIXED32
	case protoreflect.Sfixed64Kind:
		return testpilotpb.SCALAR_KIND_SFIXED64
	case protoreflect.FloatKind:
		return testpilotpb.SCALAR_KIND_FLOAT
	case protoreflect.DoubleKind:
		return testpilotpb.SCALAR_KIND_DOUBLE
	default:
		return testpilotpb.SCALAR_KIND_UNSPECIFIED
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
