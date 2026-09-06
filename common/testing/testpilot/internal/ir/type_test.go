package ir

import (
	"math"
	"testing"

	"github.com/stretchr/testify/require"
	testpilotpb "go.temporal.io/server/api/testpilot/v1"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func scalar(kind testpilotpb.ScalarKind) *testpilotpb.ValueType {
	return &testpilotpb.ValueType{Shape: &testpilotpb.ValueType_Singular{Singular: &testpilotpb.SingularType{Type: &testpilotpb.SingularType_Scalar{Scalar: &testpilotpb.ScalarType{Kind: kind}}}}}
}
func named(name string, enum bool) *testpilotpb.ValueType {
	s := &testpilotpb.SingularType{}
	if enum {
		s.Type = &testpilotpb.SingularType_Enumeration{Enumeration: &testpilotpb.NamedType{ProtobufType: name}}
	} else {
		s.Type = &testpilotpb.SingularType_Message{Message: &testpilotpb.NamedType{ProtobufType: name}}
	}
	return &testpilotpb.ValueType{Shape: &testpilotpb.ValueType_Singular{Singular: s}}
}
func text(value string) *testpilotpb.Value {
	return &testpilotpb.Value{Value: &testpilotpb.Value_Text{Text: value}}
}
func signed(value string) *testpilotpb.Value {
	return &testpilotpb.Value{Value: &testpilotpb.Value_SignedInteger{SignedInteger: value}}
}
func unsigned(value string) *testpilotpb.Value {
	return &testpilotpb.Value{Value: &testpilotpb.Value_UnsignedInteger{UnsignedInteger: value}}
}
func boolean(value bool) *testpilotpb.Value {
	return &testpilotpb.Value{Value: &testpilotpb.Value_BoolValue{BoolValue: value}}
}
func fixtureCatalog(t *testing.T) *Catalog {
	t.Helper()
	c, err := NewCatalog(catalogFixture())
	require.NoError(t, err)
	return c
}
func boundType(t *testing.T, c *Catalog, s *testpilotpb.ValueType) Type {
	t.Helper()
	typ, err := c.BindType(s)
	require.NoError(t, err)
	return typ
}

func TestLiteralsPreserveEveryScalarKindAndRange(t *testing.T) {
	c := fixtureCatalog(t)
	tests := []struct {
		kind      testpilotpb.ScalarKind
		good, bad *testpilotpb.Value
	}{
		{testpilotpb.SCALAR_KIND_TEXT, text("hello"), boolean(true)},
		{testpilotpb.SCALAR_KIND_NATURAL, &testpilotpb.Value{Value: &testpilotpb.Value_Natural{Natural: "18446744073709551616"}}, &testpilotpb.Value{Value: &testpilotpb.Value_Natural{Natural: "01"}}},
		{testpilotpb.SCALAR_KIND_BOOLEAN, boolean(false), text("false")},
		{testpilotpb.SCALAR_KIND_BYTES, &testpilotpb.Value{Value: &testpilotpb.Value_BytesValue{BytesValue: []byte{1}}}, text("bytes")},
		{testpilotpb.SCALAR_KIND_INT32, signed("-2147483648"), signed("2147483648")},
		{testpilotpb.SCALAR_KIND_INT64, signed("-9223372036854775808"), signed("9223372036854775808")},
		{testpilotpb.SCALAR_KIND_UINT32, unsigned("4294967295"), unsigned("4294967296")},
		{testpilotpb.SCALAR_KIND_UINT64, unsigned("18446744073709551615"), unsigned("18446744073709551616")},
		{testpilotpb.SCALAR_KIND_SINT32, signed("2147483647"), signed("-2147483649")},
		{testpilotpb.SCALAR_KIND_SINT64, signed("9223372036854775807"), signed("-9223372036854775809")},
		{testpilotpb.SCALAR_KIND_FIXED32, unsigned("0"), unsigned("-1")},
		{testpilotpb.SCALAR_KIND_FIXED64, unsigned("0"), unsigned("+1")},
		{testpilotpb.SCALAR_KIND_SFIXED32, signed("0"), signed("-0")},
		{testpilotpb.SCALAR_KIND_SFIXED64, signed("0"), signed("01")},
		{testpilotpb.SCALAR_KIND_FLOAT, &testpilotpb.Value{Value: &testpilotpb.Value_FloatingPoint{FloatingPoint: 1.25}}, &testpilotpb.Value{Value: &testpilotpb.Value_FloatingPoint{FloatingPoint: math.MaxFloat64}}},
		{testpilotpb.SCALAR_KIND_DOUBLE, &testpilotpb.Value{Value: &testpilotpb.Value_FloatingPoint{FloatingPoint: math.MaxFloat64}}, signed("1")},
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
	anyType := &testpilotpb.ValueType{Shape: &testpilotpb.ValueType_Singular{Singular: &testpilotpb.SingularType{Type: &testpilotpb.SingularType_Any{Any: &testpilotpb.AnyType{}}}}}
	listType := &testpilotpb.ValueType{Shape: &testpilotpb.ValueType_Repeated{Repeated: &testpilotpb.RepeatedType{Element: scalar(testpilotpb.SCALAR_KIND_TEXT).GetSingular()}}}
	mapType := &testpilotpb.ValueType{Shape: &testpilotpb.ValueType_Map{Map: &testpilotpb.MapType{Key: &testpilotpb.ScalarType{Kind: testpilotpb.SCALAR_KIND_TEXT}, Value: enum.GetSingular()}}}
	enumValue := &testpilotpb.Value{Value: &testpilotpb.Value_EnumValue{EnumValue: &testpilotpb.EnumValue{Number: 1}}}
	for _, tt := range []struct {
		name   string
		schema *testpilotpb.ValueType
		value  *testpilotpb.Value
	}{
		{"enum", enum, enumValue},
		{"message", named("google.protobuf.Timestamp", false), &testpilotpb.Value{Value: &testpilotpb.Value_MessageValue{MessageValue: stamp}}},
		{"any", anyType, &testpilotpb.Value{Value: &testpilotpb.Value_MessageValue{MessageValue: &anypb.Any{TypeUrl: "example.invalid/unknown.Payload", Value: []byte{0xff}}}}},
		{"list", listType, &testpilotpb.Value{Value: &testpilotpb.Value_ListValue{ListValue: &testpilotpb.ValueList{Values: []*testpilotpb.Value{text("x")}}}}},
		{"map", mapType, &testpilotpb.Value{Value: &testpilotpb.Value_MapValue{MapValue: &testpilotpb.ValueMap{Entries: []*testpilotpb.ValueMapEntry{{Key: text("x"), Value: enumValue}}}}}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			typ := boundType(t, c, tt.schema)
			require.NoError(t, c.CheckLiteral(tt.value, typ, DefaultLimits()))
			require.Error(t, c.CheckLiteral(boolean(true), typ, DefaultLimits()))
		})
	}
	require.Error(t, c.CheckLiteral(&testpilotpb.Value{Value: &testpilotpb.Value_EnumValue{EnumValue: &testpilotpb.EnumValue{Number: 22}}}, boundType(t, c, enum), DefaultLimits()))
	unknown := &testpilotpb.Value{Value: &testpilotpb.Value_MessageValue{MessageValue: &anypb.Any{TypeUrl: "type.googleapis.com/google.protobuf.Timestamp", Value: []byte{0x18, 1}}}}
	require.Error(t, c.CheckLiteral(unknown, boundType(t, c, named("google.protobuf.Timestamp", false)), DefaultLimits()))
	duplicate := &testpilotpb.Value{Value: &testpilotpb.Value_MapValue{MapValue: &testpilotpb.ValueMap{Entries: []*testpilotpb.ValueMapEntry{{Key: text("x"), Value: enumValue}, {Key: text("x"), Value: enumValue}}}}}
	require.Error(t, c.CheckLiteral(duplicate, boundType(t, c, mapType), DefaultLimits()))
}

func TestTypeAndLiteralRejectMalformedAndBoundedInputs(t *testing.T) {
	c := fixtureCatalog(t)
	for _, schema := range []*testpilotpb.ValueType{nil, {}, scalar(testpilotpb.SCALAR_KIND_UNSPECIFIED), named("missing.Type", false), named("fixture.State", false), {Shape: &testpilotpb.ValueType_Map{Map: &testpilotpb.MapType{Key: &testpilotpb.ScalarType{Kind: testpilotpb.SCALAR_KIND_BYTES}, Value: scalar(testpilotpb.SCALAR_KIND_TEXT).GetSingular()}}}} {
		_, err := c.BindType(schema)
		require.Error(t, err)
	}
	typ := boundType(t, c, scalar(testpilotpb.SCALAR_KIND_TEXT))
	unknown := text("x")
	unknown.ProtoReflect().SetUnknown([]byte{0x78, 1})
	for _, value := range []*testpilotpb.Value{nil, {}, unknown} {
		require.Error(t, c.CheckLiteral(value, typ, DefaultLimits()))
	}
	limits := DefaultLimits()
	limits.Bytes = 1
	require.Error(t, c.CheckLiteral(text("oversized"), typ, limits))
	limits = DefaultLimits()
	limits.Work = math.MaxInt64
	require.Error(t, c.CheckLiteral(text("x"), typ, limits))
	schema := scalar(testpilotpb.SCALAR_KIND_TEXT)
	snapshot := boundType(t, c, schema)
	schema.GetSingular().GetScalar().Kind = testpilotpb.SCALAR_KIND_BYTES
	exported := snapshot.Schema()
	exported.GetSingular().GetScalar().Kind = testpilotpb.SCALAR_KIND_BYTES
	require.True(t, proto.Equal(scalar(testpilotpb.SCALAR_KIND_TEXT), snapshot.Schema()))
}

func TestBinderRejectsCrossedCatalogsAndTypedNilUnions(t *testing.T) {
	c := fixtureCatalog(t)
	otherSource := catalogFixture()
	otherSource.File[2].Service[0].Name = proto.String("Other")
	other, err := NewCatalog(otherSource)
	require.NoError(t, err)
	foreign := boundType(t, other, scalar(testpilotpb.SCALAR_KIND_TEXT))
	require.Error(t, c.CheckLiteral(text("x"), foreign, DefaultLimits()))
	require.Error(t, c.CheckLiteral(text("x"), Type{}, DefaultLimits()))
	_, err = c.BindPath(foreign, &testpilotpb.FieldPath{}, DefaultLimits())
	require.Error(t, err)
	require.NotPanics(t, func() {
		_, err := c.BindType(&testpilotpb.ValueType{Shape: (*testpilotpb.ValueType_Singular)(nil)})
		require.Error(t, err)
	})
	require.NotPanics(t, func() {
		err := c.CheckLiteral(&testpilotpb.Value{Value: (*testpilotpb.Value_Text)(nil)}, boundType(t, c, scalar(testpilotpb.SCALAR_KIND_TEXT)), DefaultLimits())
		require.Error(t, err)
	})
	require.NotPanics(t, func() {
		_, err := c.BindPath(boundType(t, c, named("fixture.Payload", false)), &testpilotpb.FieldPath{Segments: []*testpilotpb.FieldPathSegment{{Field: "text", Selector: (*testpilotpb.FieldPathSegment_Presence)(nil)}}}, DefaultLimits())
		require.Error(t, err)
	})
}

func TestNamedPayloadsRespectCollectionCeilings(t *testing.T) {
	c := fixtureCatalog(t)
	typ := boundType(t, c, named("fixture.Payload", false))
	value := &testpilotpb.Value{Value: &testpilotpb.Value_MessageValue{MessageValue: &anypb.Any{TypeUrl: "type.googleapis.com/fixture.Payload", Value: []byte{0x1a, 0, 0x1a, 0, 0x1a, 0}}}}
	limits := DefaultLimits()
	limits.Fanout = 2
	require.Error(t, c.CheckLiteral(value, typ, limits))
}

func TestMessageWorkIsChargedBeforeDecodingAllFields(t *testing.T) {
	c := fixtureCatalog(t)
	typ := boundType(t, c, named("fixture.Payload", false))
	value := &testpilotpb.Value{Value: &testpilotpb.Value_MessageValue{MessageValue: &anypb.Any{TypeUrl: "type.googleapis.com/fixture.Payload", Value: []byte{0x1a, 0, 0x1a, 0, 0x1a, 0, 0xff}}}}
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
	value := func(wire []byte) *testpilotpb.Value {
		return &testpilotpb.Value{Value: &testpilotpb.Value_MessageValue{MessageValue: &anypb.Any{TypeUrl: "type.googleapis.com/groups.Payload", Value: wire}}}
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
	schema := &testpilotpb.ValueType{Shape: &testpilotpb.ValueType_Singular{Singular: &testpilotpb.SingularType{Type: &testpilotpb.SingularType_Enumeration{Enumeration: &testpilotpb.NamedType{ProtobufType: "temporal.server.api.testpilot.v1.InstructionOutcomeStatus"}}}}}
	typ, err := catalog.BindType(schema)
	require.NoError(t, err)
	for number := int32(1); number <= 5; number++ {
		require.NoError(t, catalog.CheckLiteral(&testpilotpb.Value{Value: &testpilotpb.Value_EnumValue{EnumValue: &testpilotpb.EnumValue{Number: number}}}, typ, DefaultLimits()))
	}
	require.Error(t, catalog.CheckLiteral(&testpilotpb.Value{Value: &testpilotpb.Value_EnumValue{EnumValue: &testpilotpb.EnumValue{Number: 99}}}, typ, DefaultLimits()))
}

func TestIntrinsicRunEventKind(t *testing.T) {
	catalog, err := NewCatalog(&descriptorpb.FileDescriptorSet{})
	require.NoError(t, err)
	typ, err := catalog.BindType(&testpilotpb.ValueType{Shape: &testpilotpb.ValueType_Singular{Singular: &testpilotpb.SingularType{Type: &testpilotpb.SingularType_Enumeration{Enumeration: &testpilotpb.NamedType{ProtobufType: "temporal.server.api.testpilot.v1.RunEventKind"}}}}})
	require.NoError(t, err)
	require.Equal(t, testpilotpb.RunEventKind(0).Descriptor(), typ.Enum())
}
