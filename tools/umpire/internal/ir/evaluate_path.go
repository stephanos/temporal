package ir

import (
	"math/bits"
	"slices"
	"strconv"
	"strings"

	umpirespb "go.temporal.io/server/api/umpire/v1"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/dynamicpb"
	"google.golang.org/protobuf/types/known/anypb"
)

func decodeMessage(value *umpirespb.Value, descriptor protoreflect.MessageDescriptor) (protoreflect.Message, error) {
	message := dynamicpb.NewMessage(descriptor)
	if err := proto.Unmarshal(value.GetMessageValue().GetValue(), message); err != nil {
		return nil, err
	}
	return message.ProtoReflect(), nil
}
func (r *runtimeExpression) project(path *Path, source *umpirespb.Value, typ Type) (*umpirespb.Value, error) {
	if err := r.charge(int64(proto.Size(source))); err != nil {
		return nil, err
	}
	current := []*umpirespb.Value{source}
	descriptor := typ.message
	for _, step := range path.steps {
		var next []*umpirespb.Value
		for _, value := range current {
			if err := r.charge(1); err != nil {
				return nil, err
			}
			values, err := r.selectBranch(value, descriptor, step, path.limit-int64(len(next)))
			if err != nil {
				return nil, err
			}

			if int64(len(values)) > path.limit-int64(len(next)) {
				return nil, invalid(LimitExceeded, "path", "fan-out ceiling exceeded")
			}
			next = append(next, values...)
		}
		current = next
		descriptor = step.Field.Message()
		if step.Field.IsMap() {
			descriptor = step.Field.MapValue().Message()
		}
	}
	for _, value := range current {
		if value == nil {
			return nil, nil
		}
	}
	if path.fanout {
		return &umpirespb.Value{Value: &umpirespb.Value_ListValue{ListValue: &umpirespb.ValueList{Values: current}}}, nil
	}
	if len(current) == 0 {
		if len(path.steps) > 0 && path.steps[len(path.steps)-1].Selector == Presence {
			return boolValue(false), nil
		}
		return nil, nil
	}
	return current[0], nil
}
func (r *runtimeExpression) selectBranch(value *umpirespb.Value, descriptor protoreflect.MessageDescriptor, step PathStep, remaining int64) ([]*umpirespb.Value, error) {
	if value == nil {
		if step.Selector == Presence {
			return []*umpirespb.Value{boolValue(false)}, nil
		}
		return []*umpirespb.Value{nil}, nil
	}
	if r.copyWork {
		if err := r.charge(int64(proto.Size(value))); err != nil {
			return nil, err
		}
	}
	message, err := decodeMessage(value, descriptor)
	if err != nil {
		return nil, err
	}
	values, err := r.selectField(message, step, remaining)
	if err != nil {
		return nil, err
	}
	if values == nil {
		return []*umpirespb.Value{nil}, nil
	}
	return values, nil
}
func (r *runtimeExpression) selectField(message protoreflect.Message, step PathStep, remaining int64) ([]*umpirespb.Value, error) {
	field := step.Field
	if step.Selector == Presence {
		return []*umpirespb.Value{boolValue(message.Has(field))}, nil
	}
	if field.HasPresence() && !message.Has(field) {
		return nil, nil
	}
	value := message.Get(field)
	if step.Selector == MapKey {
		key, err := mapKey(step.Key, field.MapKey())
		if err != nil {
			return nil, err
		}
		if !value.Map().Has(key) {
			return nil, nil
		}
		result, err := r.fieldValue(value.Map().Get(key), field.MapValue())
		return []*umpirespb.Value{result}, err
	}
	if step.Selector == Wildcard {
		list := value.List()
		if int64(list.Len()) > remaining {
			return nil, invalid(LimitExceeded, "path", "fan-out ceiling exceeded")
		}
		if err := r.charge(int64(list.Len())); err != nil {
			return nil, err
		}
		result := make([]*umpirespb.Value, 0, list.Len())
		for i := 0; i < list.Len(); i++ {
			v, err := r.scalarValue(list.Get(i), field)
			if err != nil {
				return nil, err
			}
			result = append(result, v)
		}
		return result, nil
	}
	result, err := r.fieldValue(value, field)
	return []*umpirespb.Value{result}, err
}
func mapKey(v *umpirespb.Value, field protoreflect.FieldDescriptor) (protoreflect.MapKey, error) {
	switch field.Kind() {
	case protoreflect.StringKind:
		return protoreflect.ValueOfString(v.GetText()).MapKey(), nil
	case protoreflect.BoolKind:
		return protoreflect.ValueOfBool(v.GetBoolValue()).MapKey(), nil
	case protoreflect.Int32Kind, protoreflect.Int64Kind, protoreflect.Sint32Kind, protoreflect.Sint64Kind, protoreflect.Sfixed32Kind, protoreflect.Sfixed64Kind:
		n, err := strconv.ParseInt(v.GetSignedInteger(), 10, 64)
		if field.Kind() == protoreflect.Int32Kind || field.Kind() == protoreflect.Sint32Kind || field.Kind() == protoreflect.Sfixed32Kind {
			return protoreflect.ValueOfInt32(int32(n)).MapKey(), err
		}
		return protoreflect.ValueOfInt64(n).MapKey(), err
	default:
		n, err := strconv.ParseUint(v.GetUnsignedInteger(), 10, 64)
		if field.Kind() == protoreflect.Uint32Kind || field.Kind() == protoreflect.Fixed32Kind {
			return protoreflect.ValueOfUint32(uint32(n)).MapKey(), err
		}
		return protoreflect.ValueOfUint64(n).MapKey(), err
	}
}
func (r *runtimeExpression) fieldValue(v protoreflect.Value, field protoreflect.FieldDescriptor) (*umpirespb.Value, error) {
	if field.IsList() {
		list := v.List()
		if err := r.charge(int64(list.Len())); err != nil {
			return nil, err
		}
		values := make([]*umpirespb.Value, 0, list.Len())
		for i := 0; i < list.Len(); i++ {
			item, err := r.scalarValue(list.Get(i), field)
			if err != nil {
				return nil, err
			}
			values = append(values, item)
		}
		return &umpirespb.Value{Value: &umpirespb.Value_ListValue{ListValue: &umpirespb.ValueList{Values: values}}}, nil
	}
	if field.IsMap() {
		return r.mapValue(v, field)
	}
	return r.scalarValue(v, field)
}
func (r *runtimeExpression) scalarValue(v protoreflect.Value, field protoreflect.FieldDescriptor) (*umpirespb.Value, error) {
	if r.copyWork {
		work := int64(1)
		switch field.Kind() {
		case protoreflect.StringKind:
			work += int64(len(v.String()))
		case protoreflect.BytesKind:
			work += int64(len(v.Bytes()))
		case protoreflect.MessageKind, protoreflect.GroupKind:
			work += 2 * int64(proto.Size(v.Message().Interface()))
		default:
		}
		if err := r.charge(work); err != nil {
			return nil, err
		}
	}

	switch field.Kind() {
	case protoreflect.BoolKind:
		return boolValue(v.Bool()), nil
	case protoreflect.StringKind:
		return &umpirespb.Value{Value: &umpirespb.Value_Text{Text: v.String()}}, nil
	case protoreflect.BytesKind:
		return &umpirespb.Value{Value: &umpirespb.Value_BytesValue{BytesValue: v.Bytes()}}, nil
	case protoreflect.EnumKind:
		return &umpirespb.Value{Value: &umpirespb.Value_EnumValue{EnumValue: &umpirespb.EnumValue{Number: int32(v.Enum())}}}, nil
	case protoreflect.FloatKind, protoreflect.DoubleKind:
		return &umpirespb.Value{Value: &umpirespb.Value_FloatingPoint{FloatingPoint: v.Float()}}, nil
	case protoreflect.Int32Kind, protoreflect.Int64Kind, protoreflect.Sint32Kind, protoreflect.Sint64Kind, protoreflect.Sfixed32Kind, protoreflect.Sfixed64Kind:
		return &umpirespb.Value{Value: &umpirespb.Value_SignedInteger{SignedInteger: strconv.FormatInt(v.Int(), 10)}}, nil
	case protoreflect.Uint32Kind, protoreflect.Uint64Kind, protoreflect.Fixed32Kind, protoreflect.Fixed64Kind:
		return &umpirespb.Value{Value: &umpirespb.Value_UnsignedInteger{UnsignedInteger: strconv.FormatUint(v.Uint(), 10)}}, nil
	case protoreflect.MessageKind, protoreflect.GroupKind:
		if field.Message().FullName() == "google.protobuf.Any" {
			m := v.Message()
			fields := m.Descriptor().Fields()
			return &umpirespb.Value{Value: &umpirespb.Value_MessageValue{MessageValue: &anypb.Any{TypeUrl: m.Get(fields.ByName("type_url")).String(), Value: m.Get(fields.ByName("value")).Bytes()}}}, nil
		}
		bytes, err := proto.MarshalOptions{Deterministic: true}.Marshal(v.Message().Interface())
		if err != nil {
			return nil, err
		}
		return &umpirespb.Value{Value: &umpirespb.Value_MessageValue{MessageValue: &anypb.Any{TypeUrl: "type.googleapis.com/" + string(field.Message().FullName()), Value: bytes}}}, nil
	default:
		return nil, invalid(Unsupported, "path", "unsupported field kind")
	}
}

func sortMapEntries(entries []*umpirespb.ValueMapEntry) {
	type keyedEntry struct {
		key   string
		value *umpirespb.ValueMapEntry
	}
	keyed := make([]keyedEntry, len(entries))
	for i, entry := range entries {
		keyed[i] = keyedEntry{entry.Key.String(), entry}
	}
	slices.SortFunc(keyed, func(a, b keyedEntry) int { return strings.Compare(a.key, b.key) })
	for i, entry := range keyed {
		entries[i] = entry.value
	}
}

func (r *runtimeExpression) mapValue(v protoreflect.Value, field protoreflect.FieldDescriptor) (*umpirespb.Value, error) {
	items := v.Map()
	if err := r.charge(int64(items.Len())); err != nil {
		return nil, err
	}
	entries := make([]*umpirespb.ValueMapEntry, 0, items.Len())
	var resultErr error
	items.Range(func(k protoreflect.MapKey, v protoreflect.Value) bool {
		key, err := r.scalarValue(k.Value(), field.MapKey())
		if err != nil {
			resultErr = err
			return false
		}
		value, err := r.scalarValue(v, field.MapValue())
		if err != nil {
			resultErr = err
			return false
		}
		entries = append(entries, &umpirespb.ValueMapEntry{Key: key, Value: value})
		return true
	})
	if resultErr != nil {
		return nil, resultErr
	}
	if r.copyWork {
		var size int64
		for _, entry := range entries {
			size += int64(proto.Size(entry.Key)) + 1
		}
		if err := r.charge(size * 8 * int64(bits.Len(uint(len(entries)))+1)); err != nil {
			return nil, err
		}
	}
	sortMapEntries(entries)
	return &umpirespb.Value{Value: &umpirespb.Value_MapValue{MapValue: &umpirespb.ValueMap{Entries: entries}}}, nil
}
