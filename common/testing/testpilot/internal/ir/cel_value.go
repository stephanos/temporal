package ir

import (
	"cmp"
	"fmt"
	"math"
	"reflect"
	"strings"

	engine "cel.dev/cel-go/cel"
	"cel.dev/cel-go/common/types"
	"cel.dev/cel-go/common/types/ref"
	"cel.dev/cel-go/common/types/traits"
	celpb "cel.dev/expr"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/known/anypb"
)

func (c *Catalog) celRegistry() (*types.Registry, error) {
	registry, err := types.NewRegistry()
	if err != nil {
		return nil, err
	}
	c.files.RangeFiles(func(file protoreflect.FileDescriptor) bool {
		err = registry.RegisterDescriptor(file)
		return err == nil
	})
	if err != nil {
		return nil, err
	}
	if err = registry.RegisterType(types.NewObjectType("testpilot.OpaqueAny")); err != nil {
		return nil, err
	}
	return registry, nil
}
func celType(typ Type) *engine.Type {
	if typ.cardinality == Repeated {
		return engine.ListType(celType(typ.Element()))
	}
	if typ.cardinality == Map {
		return engine.MapType(celType(typ.catalog.scalarType(typ.key)), celType(typ.Element()))
	}
	if typ.enumeration != nil {
		return engine.IntType
	}
	if typ.message != nil {
		if typ.message.FullName() == "google.protobuf.Duration" {
			return engine.DurationType
		}
		return engine.ObjectType(string(typ.message.FullName()))
	}
	if typ.any {
		return engine.ObjectType("testpilot.OpaqueAny")
	}
	switch typ.scalar {
	case testpilotspb.SCALAR_KIND_TEXT:
		return engine.StringType
	case testpilotspb.SCALAR_KIND_BOOLEAN:
		return engine.BoolType
	case testpilotspb.SCALAR_KIND_BYTES:
		return engine.BytesType
	case testpilotspb.SCALAR_KIND_FLOAT, testpilotspb.SCALAR_KIND_DOUBLE:
		return engine.DoubleType
	case testpilotspb.SCALAR_KIND_UINT32, testpilotspb.SCALAR_KIND_UINT64, testpilotspb.SCALAR_KIND_FIXED32, testpilotspb.SCALAR_KIND_FIXED64:
		return engine.UintType
	default:
		return engine.IntType
	}
}
func constantValue(constant *celpb.Constant) (*celpb.Value, error) {
	switch v := constant.ConstantKind.(type) {
	case *celpb.Constant_BoolValue:
		return &celpb.Value{Kind: &celpb.Value_BoolValue{BoolValue: v.BoolValue}}, nil
	case *celpb.Constant_StringValue:
		return &celpb.Value{Kind: &celpb.Value_StringValue{StringValue: v.StringValue}}, nil
	case *celpb.Constant_BytesValue:
		return &celpb.Value{Kind: &celpb.Value_BytesValue{BytesValue: append([]byte(nil), v.BytesValue...)}}, nil
	case *celpb.Constant_Int64Value:
		return &celpb.Value{Kind: &celpb.Value_Int64Value{Int64Value: v.Int64Value}}, nil
	case *celpb.Constant_Uint64Value:
		return &celpb.Value{Kind: &celpb.Value_Uint64Value{Uint64Value: v.Uint64Value}}, nil
	case *celpb.Constant_DoubleValue:
		return &celpb.Value{Kind: &celpb.Value_DoubleValue{DoubleValue: v.DoubleValue}}, nil
	default:
		return nil, Invalid(Unsupported, "constant", "unsupported CEL constant")
	}
}
func (c *Catalog) literalType(value *celpb.Value) (Type, error) {
	if value == nil {
		return Type{}, Invalid(Malformed, "literal", "missing literal")
	}
	switch v := value.Kind.(type) {
	case *celpb.Value_StringValue:
		return c.scalarType(testpilotspb.SCALAR_KIND_TEXT), nil
	case *celpb.Value_BoolValue:
		return c.scalarType(testpilotspb.SCALAR_KIND_BOOLEAN), nil
	case *celpb.Value_BytesValue:
		return c.scalarType(testpilotspb.SCALAR_KIND_BYTES), nil
	case *celpb.Value_Int64Value:
		return c.scalarType(testpilotspb.SCALAR_KIND_INT64), nil
	case *celpb.Value_Uint64Value:
		return c.scalarType(testpilotspb.SCALAR_KIND_UINT64), nil
	case *celpb.Value_DoubleValue:
		return c.scalarType(testpilotspb.SCALAR_KIND_DOUBLE), nil
	case *celpb.Value_EnumValue:
		return c.BindType(&testpilotspb.ValueType{Shape: &testpilotspb.ValueType_Singular{Singular: &testpilotspb.SingularType{Type: &testpilotspb.SingularType_Enumeration{Enumeration: &testpilotspb.NamedType{ProtobufType: v.EnumValue.GetType()}}}}})
	case *celpb.Value_ObjectValue:
		url := v.ObjectValue.GetTypeUrl()
		name := url[strings.LastIndexByte(url, '/')+1:]
		return c.BindType(&testpilotspb.ValueType{Shape: &testpilotspb.ValueType_Singular{Singular: &testpilotspb.SingularType{Type: &testpilotspb.SingularType_Message{Message: &testpilotspb.NamedType{ProtobufType: name}}}}})
	case *celpb.Value_ListValue:
		element := c.scalarType(testpilotspb.SCALAR_KIND_TEXT)
		if len(v.ListValue.GetValues()) > 0 {
			var err error
			element, err = c.literalType(v.ListValue.Values[0])
			if err != nil {
				return Type{}, err
			}
		}
		if element.cardinality != Singular {
			return Type{}, Invalid(Unsupported, "literal", "nested collection literals are unsupported")
		}
		return c.BindType(&testpilotspb.ValueType{Shape: &testpilotspb.ValueType_Repeated{Repeated: &testpilotspb.RepeatedType{Element: element.Schema().GetSingular()}}})
	case *celpb.Value_MapValue:
		key, element := c.scalarType(testpilotspb.SCALAR_KIND_TEXT), c.scalarType(testpilotspb.SCALAR_KIND_TEXT)
		if len(v.MapValue.GetEntries()) > 0 {
			var err error
			key, err = c.literalType(v.MapValue.Entries[0].GetKey())
			if err != nil {
				return Type{}, err
			}
			element, err = c.literalType(v.MapValue.Entries[0].GetValue())
			if err != nil {
				return Type{}, err
			}
		}
		if element.cardinality != Singular || key.cardinality != Singular {
			return Type{}, Invalid(Unsupported, "literal", "nested collection literals are unsupported")
		}
		return c.BindType(&testpilotspb.ValueType{Shape: &testpilotspb.ValueType_Map{Map: &testpilotspb.MapType{Key: key.Schema().GetSingular().GetScalar(), Value: element.Schema().GetSingular()}}})
	default:
		return Type{}, Invalid(Unsupported, "literal", "unsupported CEL value")
	}
}
func toCEL(value *celpb.Value, typ Type, registry *types.Registry) (ref.Val, error) {
	if value == nil {
		return nil, Invalid(Unavailable, "expression", "missing binding")
	}
	if typ.cardinality == Repeated {
		values := make([]ref.Val, len(value.GetListValue().GetValues()))
		for i, v := range value.GetListValue().Values {
			converted, err := toCEL(v, typ.Element(), registry)
			if err != nil {
				return nil, err
			}
			values[i] = converted
		}
		return types.NewRefValList(registry, values), nil
	}
	if typ.cardinality == Map {
		values := map[ref.Val]ref.Val{}
		for _, entry := range value.GetMapValue().GetEntries() {
			key, err := toCEL(entry.Key, typ.catalog.scalarType(typ.key), registry)
			if err != nil {
				return nil, err
			}
			v, err := toCEL(entry.Value, typ.Element(), registry)
			if err != nil {
				return nil, err
			}
			values[key] = v
		}
		return types.NewRefValMap(registry, values), nil
	}
	if typ.enumeration != nil {
		return types.Int(value.GetEnumValue().GetValue()), nil
	}
	if typ.any {
		return opaqueAny{proto.CloneOf(value.GetObjectValue())}, nil
	}
	if typ.message != nil {
		message, err := decodeMessage(value, typ.message)
		if err != nil {
			return nil, err
		}
		return messageValue{message: message, typ: typ}, nil
	}
	switch v := value.Kind.(type) {
	case *celpb.Value_StringValue:
		return types.String(v.StringValue), nil
	case *celpb.Value_BytesValue:
		return types.Bytes(append([]byte(nil), v.BytesValue...)), nil
	case *celpb.Value_BoolValue:
		return types.Bool(v.BoolValue), nil
	case *celpb.Value_Int64Value:
		return types.Int(v.Int64Value), nil
	case *celpb.Value_Uint64Value:
		return types.Uint(v.Uint64Value), nil
	case *celpb.Value_DoubleValue:
		n := v.DoubleValue
		if typ.scalar == testpilotspb.SCALAR_KIND_FLOAT {
			n = float64(float32(n))
		}
		return types.Double(n), nil
	default:
		return nil, literalMismatch()
	}
}
func fromCEL(value ref.Val, typ Type) (*celpb.Value, error) {
	if types.IsError(value) || types.IsUnknown(value) {
		return nil, Invalid(Unavailable, "expression", fmt.Sprintf("%v", value))
	}
	if typ.cardinality == Repeated {
		list, ok := value.(traits.Lister)
		if !ok {
			return nil, literalMismatch()
		}
		result := &celpb.ListValue{}
		iterator := list.Iterator()
		for iterator.HasNext() == types.True {
			item, err := fromCEL(iterator.Next(), typ.Element())
			if err != nil {
				return nil, err
			}
			result.Values = append(result.Values, item)
		}
		return &celpb.Value{Kind: &celpb.Value_ListValue{ListValue: result}}, nil
	}
	if typ.cardinality == Map {
		values, ok := value.(traits.Mapper)
		if !ok {
			return nil, literalMismatch()
		}
		result := &celpb.MapValue{}
		iterator := values.Iterator()
		for iterator.HasNext() == types.True {
			k := iterator.Next()
			key, err := fromCEL(k, typ.catalog.scalarType(typ.key))
			if err != nil {
				return nil, err
			}
			v, err := fromCEL(values.Get(k), typ.Element())
			if err != nil {
				return nil, err
			}
			result.Entries = append(result.Entries, &celpb.MapValue_Entry{Key: key, Value: v})
		}
		sortMapEntries(result.Entries)
		return &celpb.Value{Kind: &celpb.Value_MapValue{MapValue: result}}, nil
	}
	if typ.enumeration != nil {
		n, ok := value.(types.Int)
		if !ok || int64(n) < math.MinInt32 || int64(n) > math.MaxInt32 {
			return nil, literalMismatch()
		}
		return EnumValue(typ.enumeration, protoreflect.EnumNumber(n)), nil
	}
	if typ.any {
		v, ok := value.(opaqueAny)
		if !ok {
			return nil, literalMismatch()
		}
		return &celpb.Value{Kind: &celpb.Value_ObjectValue{ObjectValue: proto.CloneOf(v.envelope)}}, nil
	}
	if typ.message != nil {
		message, ok := value.Value().(proto.Message)
		if !ok {
			return nil, literalMismatch()
		}
		if !SameMessage(message.ProtoReflect().Descriptor(), typ.message) {
			return nil, literalMismatch()
		}
		wire, err := proto.MarshalOptions{Deterministic: true}.Marshal(message)
		if err != nil {
			return nil, err
		}
		return &celpb.Value{Kind: &celpb.Value_ObjectValue{ObjectValue: &anypb.Any{TypeUrl: "type.googleapis.com/" + string(typ.message.FullName()), Value: wire}}}, nil
	}
	switch v := value.(type) {
	case types.Bool:
		return boolValue(bool(v)), nil
	case types.String:
		return &celpb.Value{Kind: &celpb.Value_StringValue{StringValue: string(v)}}, nil
	case types.Bytes:
		return &celpb.Value{Kind: &celpb.Value_BytesValue{BytesValue: append([]byte(nil), v...)}}, nil
	case types.Int:
		return &celpb.Value{Kind: &celpb.Value_Int64Value{Int64Value: int64(v)}}, nil
	case types.Uint:
		return &celpb.Value{Kind: &celpb.Value_Uint64Value{Uint64Value: uint64(v)}}, nil
	case types.Double:
		return &celpb.Value{Kind: &celpb.Value_DoubleValue{DoubleValue: float64(v)}}, nil
	default:
		return nil, literalMismatch()
	}
}

type opaqueAny struct{ envelope *anypb.Any }
type messageValue struct {
	message protoreflect.Message
	typ     Type
}

func (v messageValue) ConvertToNative(typ reflect.Type) (any, error) {
	if reflect.TypeOf(v.message.Interface()).AssignableTo(typ) {
		return v.message.Interface(), nil
	}
	return nil, fmt.Errorf("catalog message cannot convert to %v", typ)
}
func (v messageValue) ConvertToType(typ ref.Type) ref.Val {
	if typ.TypeName() == v.Type().TypeName() {
		return v
	}
	if typ == types.TypeType {
		return types.NewObjectType(string(v.typ.message.FullName()))
	}
	return types.NewErr("catalog message type conversion is unsupported")
}
func (v messageValue) Equal(other ref.Val) ref.Val {
	if v.typ.message.FullName() == "google.protobuf.Duration" {
		return types.Bool(v.Compare(other) == types.IntZero)
	}
	message, ok := other.Value().(proto.Message)
	return types.Bool(ok && proto.Equal(v.message.Interface(), message))
}
func (v messageValue) Type() ref.Type {
	if v.typ.message.FullName() == "google.protobuf.Duration" {
		return types.DurationType
	}
	return types.NewObjectType(string(v.typ.message.FullName()))
}
func (v messageValue) Compare(other ref.Val) ref.Val {
	right, ok := other.(messageValue)
	if !ok || v.typ.message.FullName() != "google.protobuf.Duration" || right.typ.message.FullName() != "google.protobuf.Duration" {
		return types.NewErr("message ordering is unsupported")
	}
	for _, name := range []protoreflect.Name{"seconds", "nanos"} {
		order := cmp.Compare(v.message.Get(v.typ.message.Fields().ByName(name)).Int(), right.message.Get(right.typ.message.Fields().ByName(name)).Int())
		if order != 0 {
			return types.Int(order)
		}
	}
	return types.IntZero
}
func (v messageValue) Value() any { return v.message.Interface() }

func (v opaqueAny) ConvertToNative(typ reflect.Type) (any, error) {
	if typ == reflect.TypeOf(&anypb.Any{}) {
		return proto.CloneOf(v.envelope), nil
	}
	return nil, fmt.Errorf("opaque Any cannot convert to %v", typ)
}
func (v opaqueAny) ConvertToType(typ ref.Type) ref.Val {
	if typ.TypeName() == v.Type().TypeName() {
		return v
	}
	if typ == types.TypeType {
		return types.NewObjectType("testpilot.OpaqueAny")
	}
	return types.NewErr("opaque Any type conversion is unsupported")
}
func (v opaqueAny) Equal(other ref.Val) ref.Val {
	candidate, ok := other.(opaqueAny)
	if !ok {
		return types.False
	}
	return types.Bool(proto.Equal(v.envelope, candidate.envelope))
}
func (v opaqueAny) Type() ref.Type { return types.NewObjectType("testpilot.OpaqueAny") }
func (v opaqueAny) Value() any     { return v.envelope }
