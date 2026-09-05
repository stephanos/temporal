package ir

import (
	"context"
	"strconv"

	umpirespb "go.temporal.io/server/api/umpire/v1"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/dynamicpb"
)

type Write struct {
	Path  *Path
	Value *umpirespb.Value
}

// BuildRequest never publishes the partially constructed request on failure.
func BuildRequest(ctx context.Context, descriptor protoreflect.MessageDescriptor, writes []Write, limits Limits) (proto.Message, int64, error) {
	b, err := runtimeBudget(ctx, limits)
	if err != nil {
		return nil, 0, err
	}
	if descriptor == nil {
		return nil, 0, invalid(Malformed, "request", "request descriptor required")
	}
	message := dynamicpb.NewMessage(descriptor)
	for i, write := range writes {
		if err = b.charge(1, 1, 0, "request"); err != nil {
			break
		}
		if err = validateWrite(write, writes[:i], descriptor, b); err != nil {
			break
		}
		if err = writePath(message, write.Path, write.Value, b); err != nil {
			break
		}
	}
	if err == nil {
		err = inspect(message.ProtoReflect(), 1, b, "request")
	}
	if err == nil && int64(proto.Size(message)) > limits.Bytes {
		err = invalid(LimitExceeded, "request", "encoded request exceeds byte ceiling")
	}
	if err != nil {
		return nil, b.work, err
	}
	return message, b.work, nil
}
func writePath(message *dynamicpb.Message, path *Path, value *umpirespb.Value, b *budget) error {
	if len(path.steps) == 0 {
		replacement, err := decodeMessage(value, message.Descriptor())
		if err != nil {
			return err
		}
		proto.Merge(message, replacement.Interface())
		return nil
	}
	current := message.ProtoReflect()
	for i, step := range path.steps {
		if err := b.charge(1, 1, 0, "request.path"); err != nil {
			return err
		}
		if step.Selector == Presence || step.Selector == Wildcard {
			return invalid(Unsupported, "request.path", "assignment selector is not writable")
		}
		if i == len(path.steps)-1 {
			return writeField(current, step, value)
		}
		if step.Selector == MapKey {
			key, err := mapKey(step.Key, step.Field.MapKey())
			if err != nil {
				return err
			}
			values := current.Mutable(step.Field).Map()
			if !values.Has(key) {
				values.Set(key, values.NewValue())
			}
			current = values.Get(key).Message()
		} else {
			current = current.Mutable(step.Field).Message()
		}
	}
	return nil
}
func writeField(message protoreflect.Message, step PathStep, value *umpirespb.Value) error {
	field := step.Field
	if step.Selector == MapKey {
		key, err := mapKey(step.Key, field.MapKey())
		if err != nil {
			return err
		}
		item, err := writeScalar(value, field.MapValue())
		if err != nil {
			return err
		}
		message.Mutable(field).Map().Set(key, item)
		return nil
	}
	if field.IsList() {
		list := message.Mutable(field).List()
		for _, v := range value.GetListValue().Values {
			item, err := writeScalar(v, field)
			if err != nil {
				return err
			}
			list.Append(item)
		}
		return nil
	}
	if field.IsMap() {
		values := message.Mutable(field).Map()
		for _, entry := range value.GetMapValue().Entries {
			key, err := mapKey(entry.Key, field.MapKey())
			if err != nil {
				return err
			}
			item, err := writeScalar(entry.Value, field.MapValue())
			if err != nil {
				return err
			}
			values.Set(key, item)
		}
		return nil
	}
	item, err := writeScalar(value, field)
	if err != nil {
		return err
	}
	message.Set(field, item)
	return nil
}
func writeScalar(value *umpirespb.Value, field protoreflect.FieldDescriptor) (protoreflect.Value, error) {
	switch field.Kind() {
	case protoreflect.BoolKind:
		return protoreflect.ValueOfBool(value.GetBoolValue()), nil
	case protoreflect.StringKind:
		return protoreflect.ValueOfString(value.GetText()), nil
	case protoreflect.BytesKind:
		return protoreflect.ValueOfBytes(append([]byte(nil), value.GetBytesValue()...)), nil
	case protoreflect.EnumKind:
		return protoreflect.ValueOfEnum(protoreflect.EnumNumber(value.GetEnumValue().Number)), nil
	case protoreflect.FloatKind:
		return protoreflect.ValueOfFloat32(float32(value.GetFloatingPoint())), nil
	case protoreflect.DoubleKind:
		return protoreflect.ValueOfFloat64(value.GetFloatingPoint()), nil
	case protoreflect.Int32Kind, protoreflect.Sint32Kind, protoreflect.Sfixed32Kind:
		n, err := strconv.ParseInt(value.GetSignedInteger(), 10, 32)
		return protoreflect.ValueOfInt32(int32(n)), err
	case protoreflect.Int64Kind, protoreflect.Sint64Kind, protoreflect.Sfixed64Kind:
		n, err := strconv.ParseInt(value.GetSignedInteger(), 10, 64)
		return protoreflect.ValueOfInt64(n), err
	case protoreflect.Uint32Kind, protoreflect.Fixed32Kind:
		n, err := strconv.ParseUint(value.GetUnsignedInteger(), 10, 32)
		return protoreflect.ValueOfUint32(uint32(n)), err
	case protoreflect.Uint64Kind, protoreflect.Fixed64Kind:
		n, err := strconv.ParseUint(value.GetUnsignedInteger(), 10, 64)
		return protoreflect.ValueOfUint64(n), err
	case protoreflect.MessageKind, protoreflect.GroupKind:
		if field.Message().FullName() == "google.protobuf.Any" {
			m := dynamicpb.NewMessage(field.Message())
			fields := field.Message().Fields()
			v := value.GetMessageValue()
			m.Set(fields.ByName("type_url"), protoreflect.ValueOfString(v.TypeUrl))
			m.Set(fields.ByName("value"), protoreflect.ValueOfBytes(append([]byte(nil), v.Value...)))
			return protoreflect.ValueOfMessage(m), nil
		}
		m, err := decodeMessage(value, field.Message())
		if err != nil {
			return protoreflect.Value{}, err
		}
		return protoreflect.ValueOfMessage(m), nil
	default:
		return protoreflect.Value{}, invalid(Unsupported, "request", "unsupported protobuf field")
	}
}

func conflictWork(left, right *Path) int64 {
	work := int64(1)
	for _, path := range []*Path{left, right} {
		for _, step := range path.steps {
			work += 1 + int64(proto.Size(step.Key))
		}
	}
	return work
}

func validateWrite(write Write, previousWrites []Write, descriptor protoreflect.MessageDescriptor, b *budget) error {
	path := write.Path
	if path == nil || path.source.message != descriptor || path.fanout {
		return invalid(TypeMismatch, "request", "assignment path has wrong source or fanout")
	}
	for _, previous := range previousWrites {
		if err := b.charge(1, conflictWork(previous.Path, path), 0, "request"); err != nil {
			return err
		}
		if previous.Path.Conflicts(path) {
			return invalid(Malformed, "request", "overlapping assignments")
		}
	}
	if err := validateValue(write.Value, path.typ, b); err != nil {
		return err
	}
	return b.charge(1, 2*int64(proto.Size(write.Value)), 0, "request")
}
