package ir

import (
	"math"
	"testing"

	"github.com/stretchr/testify/require"
	umpirespb "go.temporal.io/server/api/umpire/v1"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func scalar(kind umpirespb.ScalarKind) *umpirespb.ValueType {
	return &umpirespb.ValueType{Shape: &umpirespb.ValueType_Singular{Singular: &umpirespb.SingularType{Type: &umpirespb.SingularType_Scalar{Scalar: &umpirespb.ScalarType{Kind: kind}}}}}
}
func named(name string, enum bool) *umpirespb.ValueType {
	s := &umpirespb.SingularType{}
	if enum {
		s.Type = &umpirespb.SingularType_Enumeration{Enumeration: &umpirespb.NamedType{ProtobufType: name}}
	} else {
		s.Type = &umpirespb.SingularType_Message{Message: &umpirespb.NamedType{ProtobufType: name}}
	}
	return &umpirespb.ValueType{Shape: &umpirespb.ValueType_Singular{Singular: s}}
}
func text(value string) *umpirespb.Value {
	return &umpirespb.Value{Value: &umpirespb.Value_Text{Text: value}}
}
func signed(value string) *umpirespb.Value {
	return &umpirespb.Value{Value: &umpirespb.Value_SignedInteger{SignedInteger: value}}
}
func unsigned(value string) *umpirespb.Value {
	return &umpirespb.Value{Value: &umpirespb.Value_UnsignedInteger{UnsignedInteger: value}}
}
func boolean(value bool) *umpirespb.Value {
	return &umpirespb.Value{Value: &umpirespb.Value_BoolValue{BoolValue: value}}
}
func fixtureCatalog(t *testing.T) *Catalog {
	t.Helper()
	c, err := NewCatalog(catalogFixture())
	require.NoError(t, err)
	return c
}
func boundType(t *testing.T, c *Catalog, s *umpirespb.ValueType) Type {
	t.Helper()
	typ, err := c.BindType(s)
	require.NoError(t, err)
	return typ
}

func TestLiteralsPreserveEveryScalarKindAndRange(t *testing.T) {
	c := fixtureCatalog(t)
	tests := []struct {
		kind      umpirespb.ScalarKind
		good, bad *umpirespb.Value
	}{
		{umpirespb.SCALAR_KIND_TEXT, text("hello"), boolean(true)},
		{umpirespb.SCALAR_KIND_NATURAL, &umpirespb.Value{Value: &umpirespb.Value_Natural{Natural: "18446744073709551616"}}, &umpirespb.Value{Value: &umpirespb.Value_Natural{Natural: "01"}}},
		{umpirespb.SCALAR_KIND_BOOLEAN, boolean(false), text("false")},
		{umpirespb.SCALAR_KIND_BYTES, &umpirespb.Value{Value: &umpirespb.Value_BytesValue{BytesValue: []byte{1}}}, text("bytes")},
		{umpirespb.SCALAR_KIND_INT32, signed("-2147483648"), signed("2147483648")},
		{umpirespb.SCALAR_KIND_INT64, signed("-9223372036854775808"), signed("9223372036854775808")},
		{umpirespb.SCALAR_KIND_UINT32, unsigned("4294967295"), unsigned("4294967296")},
		{umpirespb.SCALAR_KIND_UINT64, unsigned("18446744073709551615"), unsigned("18446744073709551616")},
		{umpirespb.SCALAR_KIND_SINT32, signed("2147483647"), signed("-2147483649")},
		{umpirespb.SCALAR_KIND_SINT64, signed("9223372036854775807"), signed("-9223372036854775809")},
		{umpirespb.SCALAR_KIND_FIXED32, unsigned("0"), unsigned("-1")},
		{umpirespb.SCALAR_KIND_FIXED64, unsigned("0"), unsigned("+1")},
		{umpirespb.SCALAR_KIND_SFIXED32, signed("0"), signed("-0")},
		{umpirespb.SCALAR_KIND_SFIXED64, signed("0"), signed("01")},
		{umpirespb.SCALAR_KIND_FLOAT, &umpirespb.Value{Value: &umpirespb.Value_FloatingPoint{FloatingPoint: 1.25}}, &umpirespb.Value{Value: &umpirespb.Value_FloatingPoint{FloatingPoint: math.MaxFloat64}}},
		{umpirespb.SCALAR_KIND_DOUBLE, &umpirespb.Value{Value: &umpirespb.Value_FloatingPoint{FloatingPoint: math.MaxFloat64}}, signed("1")},
	}
	for _, tt := range tests {
		t.Run(tt.kind.String(), func(t *testing.T) {
			typ := boundType(t, c, scalar(tt.kind))
			require.NoError(t, c.CheckLiteral(tt.good, typ, DefaultLimits()))
			require.Error(t, c.CheckLiteral(tt.bad, typ, DefaultLimits()))
			require.True(t, proto.Equal(scalar(tt.kind), typ.Schema()))
		})
	}
}

func TestNamedCollectionAndAnyLiterals(t *testing.T) {
	c := fixtureCatalog(t)
	stamp, err := anypb.New(&timestamppb.Timestamp{Seconds: 10})
	require.NoError(t, err)
	enum := named("fixture.State", true)
	anyType := &umpirespb.ValueType{Shape: &umpirespb.ValueType_Singular{Singular: &umpirespb.SingularType{Type: &umpirespb.SingularType_Any{Any: &umpirespb.AnyType{}}}}}
	listType := &umpirespb.ValueType{Shape: &umpirespb.ValueType_Repeated{Repeated: &umpirespb.RepeatedType{Element: scalar(umpirespb.SCALAR_KIND_TEXT).GetSingular()}}}
	mapType := &umpirespb.ValueType{Shape: &umpirespb.ValueType_Map{Map: &umpirespb.MapType{Key: &umpirespb.ScalarType{Kind: umpirespb.SCALAR_KIND_TEXT}, Value: enum.GetSingular()}}}
	enumValue := &umpirespb.Value{Value: &umpirespb.Value_EnumValue{EnumValue: &umpirespb.EnumValue{Number: 1}}}
	for _, tt := range []struct {
		name   string
		schema *umpirespb.ValueType
		value  *umpirespb.Value
	}{
		{"enum", enum, enumValue},
		{"message", named("google.protobuf.Timestamp", false), &umpirespb.Value{Value: &umpirespb.Value_MessageValue{MessageValue: stamp}}},
		{"any", anyType, &umpirespb.Value{Value: &umpirespb.Value_MessageValue{MessageValue: &anypb.Any{TypeUrl: "example.invalid/unknown.Payload", Value: []byte{0xff}}}}},
		{"list", listType, &umpirespb.Value{Value: &umpirespb.Value_ListValue{ListValue: &umpirespb.ValueList{Values: []*umpirespb.Value{text("x")}}}}},
		{"map", mapType, &umpirespb.Value{Value: &umpirespb.Value_MapValue{MapValue: &umpirespb.ValueMap{Entries: []*umpirespb.ValueMapEntry{{Key: text("x"), Value: enumValue}}}}}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			typ := boundType(t, c, tt.schema)
			require.NoError(t, c.CheckLiteral(tt.value, typ, DefaultLimits()))
			require.Error(t, c.CheckLiteral(boolean(true), typ, DefaultLimits()))
		})
	}
	require.Error(t, c.CheckLiteral(&umpirespb.Value{Value: &umpirespb.Value_EnumValue{EnumValue: &umpirespb.EnumValue{Number: 22}}}, boundType(t, c, enum), DefaultLimits()))
	unknown := &umpirespb.Value{Value: &umpirespb.Value_MessageValue{MessageValue: &anypb.Any{TypeUrl: "type.googleapis.com/google.protobuf.Timestamp", Value: []byte{0x18, 1}}}}
	require.Error(t, c.CheckLiteral(unknown, boundType(t, c, named("google.protobuf.Timestamp", false)), DefaultLimits()))
	duplicate := &umpirespb.Value{Value: &umpirespb.Value_MapValue{MapValue: &umpirespb.ValueMap{Entries: []*umpirespb.ValueMapEntry{{Key: text("x"), Value: enumValue}, {Key: text("x"), Value: enumValue}}}}}
	require.Error(t, c.CheckLiteral(duplicate, boundType(t, c, mapType), DefaultLimits()))
}

func TestTypeAndLiteralRejectMalformedAndBoundedInputs(t *testing.T) {
	c := fixtureCatalog(t)
	for _, schema := range []*umpirespb.ValueType{nil, {}, scalar(umpirespb.SCALAR_KIND_UNSPECIFIED), named("missing.Type", false), named("fixture.State", false), {Shape: &umpirespb.ValueType_Map{Map: &umpirespb.MapType{Key: &umpirespb.ScalarType{Kind: umpirespb.SCALAR_KIND_BYTES}, Value: scalar(umpirespb.SCALAR_KIND_TEXT).GetSingular()}}}} {
		_, err := c.BindType(schema)
		require.Error(t, err)
	}
	typ := boundType(t, c, scalar(umpirespb.SCALAR_KIND_TEXT))
	unknown := text("x")
	unknown.ProtoReflect().SetUnknown([]byte{0x78, 1})
	for _, value := range []*umpirespb.Value{nil, {}, unknown} {
		require.Error(t, c.CheckLiteral(value, typ, DefaultLimits()))
	}
	limits := DefaultLimits()
	limits.Bytes = 1
	require.Error(t, c.CheckLiteral(text("oversized"), typ, limits))
	limits = DefaultLimits()
	limits.Work = math.MaxInt64
	require.Error(t, c.CheckLiteral(text("x"), typ, limits))
	schema := scalar(umpirespb.SCALAR_KIND_TEXT)
	snapshot := boundType(t, c, schema)
	schema.GetSingular().GetScalar().Kind = umpirespb.SCALAR_KIND_BYTES
	exported := snapshot.Schema()
	exported.GetSingular().GetScalar().Kind = umpirespb.SCALAR_KIND_BYTES
	require.True(t, proto.Equal(scalar(umpirespb.SCALAR_KIND_TEXT), snapshot.Schema()))
}

func TestBinderRejectsCrossedCatalogsAndTypedNilUnions(t *testing.T) {
	c := fixtureCatalog(t)
	otherSource := catalogFixture()
	otherSource.File[2].Service[0].Name = proto.String("Other")
	other, err := NewCatalog(otherSource)
	require.NoError(t, err)
	foreign := boundType(t, other, scalar(umpirespb.SCALAR_KIND_TEXT))
	require.Error(t, c.CheckLiteral(text("x"), foreign, DefaultLimits()))
	require.Error(t, c.CheckLiteral(text("x"), Type{}, DefaultLimits()))
	_, err = c.BindPath(foreign, &umpirespb.FieldPath{}, DefaultLimits())
	require.Error(t, err)
	require.NotPanics(t, func() {
		_, err := c.BindType(&umpirespb.ValueType{Shape: (*umpirespb.ValueType_Singular)(nil)})
		require.Error(t, err)
	})
	require.NotPanics(t, func() {
		err := c.CheckLiteral(&umpirespb.Value{Value: (*umpirespb.Value_Text)(nil)}, boundType(t, c, scalar(umpirespb.SCALAR_KIND_TEXT)), DefaultLimits())
		require.Error(t, err)
	})
	require.NotPanics(t, func() {
		_, err := c.BindPath(boundType(t, c, named("fixture.Payload", false)), &umpirespb.FieldPath{Segments: []*umpirespb.FieldPathSegment{{Field: "text", Selector: (*umpirespb.FieldPathSegment_Presence)(nil)}}}, DefaultLimits())
		require.Error(t, err)
	})
}

func TestNamedPayloadsRespectCollectionCeilings(t *testing.T) {
	c := fixtureCatalog(t)
	typ := boundType(t, c, named("fixture.Payload", false))
	value := &umpirespb.Value{Value: &umpirespb.Value_MessageValue{MessageValue: &anypb.Any{TypeUrl: "type.googleapis.com/fixture.Payload", Value: []byte{0x1a, 0, 0x1a, 0, 0x1a, 0}}}}
	limits := DefaultLimits()
	limits.Fanout = 2
	require.Error(t, c.CheckLiteral(value, typ, limits))
}

func TestMessageWorkIsChargedBeforeDecodingAllFields(t *testing.T) {
	c := fixtureCatalog(t)
	typ := boundType(t, c, named("fixture.Payload", false))
	value := &umpirespb.Value{Value: &umpirespb.Value_MessageValue{MessageValue: &anypb.Any{TypeUrl: "type.googleapis.com/fixture.Payload", Value: []byte{0x1a, 0, 0x1a, 0, 0x1a, 0, 0xff}}}}
	limits := DefaultLimits()
	limits.Work = 6
	var admission *Error
	require.ErrorAs(t, c.CheckLiteral(value, typ, limits), &admission)
	require.Equal(t, LimitExceeded, admission.Category)
}

func TestGroupPayloadsAreScannedUnderTheSameWorkBudget(t *testing.T) {
	source := catalogFixture()
	source.File = append(source.File, &descriptorpb.FileDescriptorProto{Name: proto.String("groups.proto"), Package: proto.String("groups"), Syntax: proto.String("proto2"), MessageType: []*descriptorpb.DescriptorProto{{Name: proto.String("Payload"), Field: []*descriptorpb.FieldDescriptorProto{{Name: proto.String("node"), Number: proto.Int32(1), Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_GROUP.Enum(), TypeName: proto.String(".groups.Payload.Node")}}, NestedType: []*descriptorpb.DescriptorProto{{Name: proto.String("Node"), Field: []*descriptorpb.FieldDescriptorProto{{Name: proto.String("numbers"), Number: proto.Int32(1), Label: descriptorpb.FieldDescriptorProto_LABEL_REPEATED.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_INT32.Enum()}}}}}}})
	c, err := NewCatalog(source)
	require.NoError(t, err)
	typ := boundType(t, c, named("groups.Payload", false))
	value := func(wire []byte) *umpirespb.Value {
		return &umpirespb.Value{Value: &umpirespb.Value_MessageValue{MessageValue: &anypb.Any{TypeUrl: "type.googleapis.com/groups.Payload", Value: wire}}}
	}
	require.NoError(t, c.CheckLiteral(value([]byte{0x0b, 8, 1, 0x0c}), typ, DefaultLimits()))
	require.NoError(t, c.CheckLiteral(value([]byte{0x0b, 0x0a, 2, 1, 2, 0x0c}), typ, DefaultLimits()))
	require.Error(t, c.CheckLiteral(value([]byte{0x0b, 0x0a, 1, 0xff, 0x0c}), typ, DefaultLimits()))
	limits := DefaultLimits()
	limits.Work = 6
	var admission *Error
	require.ErrorAs(t, c.CheckLiteral(value([]byte{0x0b, 8, 1, 8, 2, 0xff}), typ, limits), &admission)
	require.Equal(t, LimitExceeded, admission.Category)
}

func TestIntrinsicOutcomeStatusWithoutHostSchema(t *testing.T) {
	catalog, err := NewCatalog(catalogFixture())
	require.NoError(t, err)
	schema := &umpirespb.ValueType{Shape: &umpirespb.ValueType_Singular{Singular: &umpirespb.SingularType{Type: &umpirespb.SingularType_Enumeration{Enumeration: &umpirespb.NamedType{ProtobufType: "temporal.server.api.umpire.v1.InstructionOutcomeStatus"}}}}}
	typ, err := catalog.BindType(schema)
	require.NoError(t, err)
	for number := int32(1); number <= 5; number++ {
		require.NoError(t, catalog.CheckLiteral(&umpirespb.Value{Value: &umpirespb.Value_EnumValue{EnumValue: &umpirespb.EnumValue{Number: number}}}, typ, DefaultLimits()))
	}
	require.Error(t, catalog.CheckLiteral(&umpirespb.Value{Value: &umpirespb.Value_EnumValue{EnumValue: &umpirespb.EnumValue{Number: 99}}}, typ, DefaultLimits()))
}
