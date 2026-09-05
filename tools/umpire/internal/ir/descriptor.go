package ir

import (
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

type messagePair struct {
	source, target protoreflect.MessageDescriptor
}

// Descriptor identity is the common fast path. A bounded worklist checks independently built
// descriptors without rebinding paths or retaining a cache of Host-supplied descriptor identities.
func compatibleMessage(source, target protoreflect.MessageDescriptor, b *budget) error {
	pending := []messagePair{{source, target}}
	seen := map[messagePair]bool{}
	for len(pending) > 0 {
		pair := pending[len(pending)-1]
		pending = pending[:len(pending)-1]
		if pair.source == pair.target || seen[pair] {
			continue
		}
		if err := b.charge(1, 1+int64(len(pair.source.FullName())+len(pair.target.FullName())), 0, "descriptor"); err != nil {
			return err
		}
		seen[pair] = true
		a, z := pair.source, pair.target
		if a.FullName() != z.FullName() || a.ParentFile().Syntax() != z.ParentFile().Syntax() || a.IsMapEntry() != z.IsMapEntry() || a.Fields().Len() != z.Fields().Len() || a.Oneofs().Len() != z.Oneofs().Len() || a.Messages().Len() != z.Messages().Len() || a.Enums().Len() != z.Enums().Len() || a.Extensions().Len() != z.Extensions().Len() {
			return crossedDescriptor()
		}
		if err := compatibleOptions(a.Options(), z.Options(), b); err != nil {
			return err
		}
		children, err := compatibleMembers(a, z, b)
		if err != nil {
			return err
		}
		pending = append(pending, children...)

	}
	return nil
}
func crossedDescriptor() error {
	return invalid(TypeMismatch, "message", "response descriptor differs from pinned schema")
}
func compatibleField(a, z protoreflect.FieldDescriptor, b *budget) error {
	if z == nil {
		return crossedDescriptor()
	}
	if err := b.charge(1, 1+int64(len(a.FullName())+len(z.FullName())+len(a.JSONName())+len(z.JSONName())), 0, "descriptor.field"); err != nil {
		return err
	}
	if a.FullName() != z.FullName() || a.JSONName() != z.JSONName() || a.Number() != z.Number() || a.Kind() != z.Kind() || a.Cardinality() != z.Cardinality() || a.HasPresence() != z.HasPresence() || a.IsPacked() != z.IsPacked() || a.IsMap() != z.IsMap() || a.IsExtension() != z.IsExtension() || a.HasOptionalKeyword() != z.HasOptionalKeyword() || a.HasDefault() != z.HasDefault() {
		return crossedDescriptor()
	}
	x, y := a.ContainingOneof(), z.ContainingOneof()
	if (x == nil) != (y == nil) {
		return crossedDescriptor()
	}
	if x != nil {
		if err := b.charge(1, int64(len(x.FullName())+len(y.FullName())), 0, "descriptor.oneof"); err != nil {
			return err
		}
		if x.FullName() != y.FullName() || x.IsSynthetic() != y.IsSynthetic() {
			return crossedDescriptor()
		}
	}
	if a.Message() == nil && !a.IsList() {
		if err := b.charge(1, defaultWork(a)+defaultWork(z), 0, "descriptor.default"); err != nil {
			return err
		}
		if !a.Default().Equal(z.Default()) {
			return crossedDescriptor()
		}
	}
	if err := compatibleOptions(a.Options(), z.Options(), b); err != nil {
		return err
	}
	if a.Enum() != nil {
		return compatibleEnum(a.Enum(), z.Enum(), b)
	}
	return nil
}
func compatibleEnum(a, z protoreflect.EnumDescriptor, b *budget) error {
	if a == z {
		return nil
	}
	if z == nil {
		return crossedDescriptor()
	}
	if err := b.charge(1, 1+int64(len(a.FullName())+len(z.FullName())), 0, "descriptor.enum"); err != nil {
		return err
	}
	if a.FullName() != z.FullName() || a.IsClosed() != z.IsClosed() || a.Values().Len() != z.Values().Len() {
		return crossedDescriptor()
	}
	if err := compatibleOptions(a.Options(), z.Options(), b); err != nil {
		return err
	}
	for i := 0; i < a.Values().Len(); i++ {
		value := a.Values().Get(i)
		if err := b.charge(1, 1+int64(len(value.Name())), 0, "descriptor.enum"); err != nil {
			return err
		}
		other := z.Values().ByName(value.Name())
		if other == nil || value.Number() != other.Number() {
			return crossedDescriptor()
		}
		if err := compatibleOptions(value.Options(), other.Options(), b); err != nil {
			return err
		}
	}
	return nil
}
func compatibleOptions(a, z proto.Message, b *budget) error {
	fanout := b.limits.Fanout
	b.limits.Fanout = DefaultLimits().Fanout
	defer func() { b.limits.Fanout = fanout }()
	for _, options := range []proto.Message{a, z} {
		if missing(options) {
			continue
		}
		if err := inspectSurface(options.ProtoReflect(), b, "descriptor.options"); err != nil {
			return err
		}
	}
	size := int64(proto.Size(a)) + int64(proto.Size(z))
	if err := b.charge(1, size, 0, "descriptor.options"); err != nil {
		return err
	}
	if size != 0 && !proto.Equal(a, z) {
		return crossedDescriptor()
	}
	return nil
}

func defaultWork(field protoreflect.FieldDescriptor) int64 {
	switch field.Kind() {
	case protoreflect.StringKind:
		return int64(len(field.Default().String()))
	case protoreflect.BytesKind:
		return int64(len(field.Default().Bytes()))
	default:
		return 8
	}
}

func compatibleMembers(a, z protoreflect.MessageDescriptor, b *budget) ([]messagePair, error) {
	var children []messagePair
	for i := 0; i < a.Fields().Len(); i++ {
		field := a.Fields().Get(i)
		other := z.Fields().ByNumber(field.Number())
		if err := compatibleField(field, other, b); err != nil {
			return nil, err
		}
		if field.Message() != nil {
			children = append(children, messagePair{field.Message(), other.Message()})
		}
	}
	for i := 0; i < a.Messages().Len(); i++ {
		if err := b.charge(1, 1, 0, "descriptor"); err != nil {
			return nil, err
		}
		nested := a.Messages().Get(i)
		other := z.Messages().ByName(nested.Name())
		if other == nil {
			return nil, crossedDescriptor()
		}
		children = append(children, messagePair{nested, other})
	}
	for i := 0; i < a.Enums().Len(); i++ {
		enumeration := a.Enums().Get(i)
		if err := compatibleEnum(enumeration, z.Enums().ByName(enumeration.Name()), b); err != nil {
			return nil, err
		}
	}
	for i := 0; i < a.Extensions().Len(); i++ {
		field := a.Extensions().Get(i)
		if err := compatibleField(field, z.Extensions().ByName(field.Name()), b); err != nil {
			return nil, err
		}
	}
	return children, nil
}
