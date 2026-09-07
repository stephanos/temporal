package ir

import (
	"math"
	"testing"

	"github.com/stretchr/testify/require"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func scalar(kind testpilotspb.ScalarKind) *testpilotspb.ValueType {
	return &testpilotspb.ValueType{Shape: &testpilotspb.ValueType_Singular{Singular: &testpilotspb.SingularType{Type: &testpilotspb.SingularType_Scalar{Scalar: &testpilotspb.ScalarType{Kind: kind}}}}}
}
func named(name string, enum bool) *testpilotspb.ValueType {
	s := &testpilotspb.SingularType{}
	if enum {
		s.Type = &testpilotspb.SingularType_Enumeration{Enumeration: &testpilotspb.NamedType{ProtobufType: name}}
	} else {
		s.Type = &testpilotspb.SingularType_Message{Message: &testpilotspb.NamedType{ProtobufType: name}}
	}
	return &testpilotspb.ValueType{Shape: &testpilotspb.ValueType_Singular{Singular: s}}
}
func text(value string) *testpilotspb.Value {
	return &testpilotspb.Value{Value: &testpilotspb.Value_Text{Text: value}}
}
func signed(value string) *testpilotspb.Value {
	return &testpilotspb.Value{Value: &testpilotspb.Value_SignedInteger{SignedInteger: value}}
}
func unsigned(value string) *testpilotspb.Value {
	return &testpilotspb.Value{Value: &testpilotspb.Value_UnsignedInteger{UnsignedInteger: value}}
}
func boolean(value bool) *testpilotspb.Value {
	return &testpilotspb.Value{Value: &testpilotspb.Value_BoolValue{BoolValue: value}}
}
func fixtureCatalog(t *testing.T) *Catalog {
	t.Helper()
	c, err := NewCatalog(catalogFixture())
	require.NoError(t, err)
	return c
}
func boundType(t *testing.T, c *Catalog, s *testpilotspb.ValueType) Type {
	t.Helper()
	typ, err := c.BindType(s)
	require.NoError(t, err)
	return typ
}

func TestLiteralsPreserveEveryScalarKindAndRange(t *testing.T) {
	c := fixtureCatalog(t)
	tests := []struct {
		kind      testpilotspb.ScalarKind
		good, bad *testpilotspb.Value
	}{
		{testpilotspb.SCALAR_KIND_TEXT, text("hello"), boolean(true)},
		{testpilotspb.SCALAR_KIND_NATURAL, &testpilotspb.Value{Value: &testpilotspb.Value_Natural{Natural: "18446744073709551616"}}, &testpilotspb.Value{Value: &testpilotspb.Value_Natural{Natural: "01"}}},
		{testpilotspb.SCALAR_KIND_BOOLEAN, boolean(false), text("false")},
		{testpilotspb.SCALAR_KIND_BYTES, &testpilotspb.Value{Value: &testpilotspb.Value_BytesValue{BytesValue: []byte{1}}}, text("bytes")},
		{testpilotspb.SCALAR_KIND_INT32, signed("-2147483648"), signed("2147483648")},
		{testpilotspb.SCALAR_KIND_INT64, signed("-9223372036854775808"), signed("9223372036854775808")},
		{testpilotspb.SCALAR_KIND_UINT32, unsigned("4294967295"), unsigned("4294967296")},
		{testpilotspb.SCALAR_KIND_UINT64, unsigned("18446744073709551615"), unsigned("18446744073709551616")},
		{testpilotspb.SCALAR_KIND_SINT32, signed("2147483647"), signed("-2147483649")},
		{testpilotspb.SCALAR_KIND_SINT64, signed("9223372036854775807"), signed("-9223372036854775809")},
		{testpilotspb.SCALAR_KIND_FIXED32, unsigned("0"), unsigned("-1")},
		{testpilotspb.SCALAR_KIND_FIXED64, unsigned("0"), unsigned("+1")},
		{testpilotspb.SCALAR_KIND_SFIXED32, signed("0"), signed("-0")},
		{testpilotspb.SCALAR_KIND_SFIXED64, signed("0"), signed("01")},
		{testpilotspb.SCALAR_KIND_FLOAT, &testpilotspb.Value{Value: &testpilotspb.Value_FloatingPoint{FloatingPoint: 1.25}}, &testpilotspb.Value{Value: &testpilotspb.Value_FloatingPoint{FloatingPoint: math.MaxFloat64}}},
		{testpilotspb.SCALAR_KIND_DOUBLE, &testpilotspb.Value{Value: &testpilotspb.Value_FloatingPoint{FloatingPoint: math.MaxFloat64}}, signed("1")},
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
	anyType := &testpilotspb.ValueType{Shape: &testpilotspb.ValueType_Singular{Singular: &testpilotspb.SingularType{Type: &testpilotspb.SingularType_Any{Any: &testpilotspb.AnyType{}}}}}
	listType := &testpilotspb.ValueType{Shape: &testpilotspb.ValueType_Repeated{Repeated: &testpilotspb.RepeatedType{Element: scalar(testpilotspb.SCALAR_KIND_TEXT).GetSingular()}}}
	mapType := &testpilotspb.ValueType{Shape: &testpilotspb.ValueType_Map{Map: &testpilotspb.MapType{Key: &testpilotspb.ScalarType{Kind: testpilotspb.SCALAR_KIND_TEXT}, Value: enum.GetSingular()}}}
	enumValue := &testpilotspb.Value{Value: &testpilotspb.Value_EnumValue{EnumValue: &testpilotspb.EnumValue{Number: 1}}}
	for _, tt := range []struct {
		name   string
		schema *testpilotspb.ValueType
		value  *testpilotspb.Value
	}{
		{"enum", enum, enumValue},
		{"message", named("google.protobuf.Timestamp", false), &testpilotspb.Value{Value: &testpilotspb.Value_MessageValue{MessageValue: stamp}}},
		{"any", anyType, &testpilotspb.Value{Value: &testpilotspb.Value_MessageValue{MessageValue: &anypb.Any{TypeUrl: "example.invalid/unknown.Payload", Value: []byte{0xff}}}}},
		{"list", listType, &testpilotspb.Value{Value: &testpilotspb.Value_ListValue{ListValue: &testpilotspb.ValueList{Values: []*testpilotspb.Value{text("x")}}}}},
		{"map", mapType, &testpilotspb.Value{Value: &testpilotspb.Value_MapValue{MapValue: &testpilotspb.ValueMap{Entries: []*testpilotspb.ValueMapEntry{{Key: text("x"), Value: enumValue}}}}}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			typ := boundType(t, c, tt.schema)
			require.NoError(t, c.CheckLiteral(tt.value, typ, DefaultLimits()))
			require.Error(t, c.CheckLiteral(boolean(true), typ, DefaultLimits()))
		})
	}
	require.Error(t, c.CheckLiteral(&testpilotspb.Value{Value: &testpilotspb.Value_EnumValue{EnumValue: &testpilotspb.EnumValue{Number: 22}}}, boundType(t, c, enum), DefaultLimits()))
	unknown := &testpilotspb.Value{Value: &testpilotspb.Value_MessageValue{MessageValue: &anypb.Any{TypeUrl: "type.googleapis.com/google.protobuf.Timestamp", Value: []byte{0x18, 1}}}}
	require.Error(t, c.CheckLiteral(unknown, boundType(t, c, named("google.protobuf.Timestamp", false)), DefaultLimits()))
	duplicate := &testpilotspb.Value{Value: &testpilotspb.Value_MapValue{MapValue: &testpilotspb.ValueMap{Entries: []*testpilotspb.ValueMapEntry{{Key: text("x"), Value: enumValue}, {Key: text("x"), Value: enumValue}}}}}
	require.Error(t, c.CheckLiteral(duplicate, boundType(t, c, mapType), DefaultLimits()))
}

func TestTypeAndLiteralRejectMalformedAndBoundedInputs(t *testing.T) {
	c := fixtureCatalog(t)
	for _, schema := range []*testpilotspb.ValueType{nil, {}, scalar(testpilotspb.SCALAR_KIND_UNSPECIFIED), named("missing.Type", false), named("fixture.State", false), {Shape: &testpilotspb.ValueType_Map{Map: &testpilotspb.MapType{Key: &testpilotspb.ScalarType{Kind: testpilotspb.SCALAR_KIND_BYTES}, Value: scalar(testpilotspb.SCALAR_KIND_TEXT).GetSingular()}}}} {
		_, err := c.BindType(schema)
		require.Error(t, err)
	}
	typ := boundType(t, c, scalar(testpilotspb.SCALAR_KIND_TEXT))
	unknown := text("x")
	unknown.ProtoReflect().SetUnknown([]byte{0x78, 1})
	for _, value := range []*testpilotspb.Value{nil, {}, unknown} {
		require.Error(t, c.CheckLiteral(value, typ, DefaultLimits()))
	}
	limits := DefaultLimits()
	limits.Bytes = 1
	require.Error(t, c.CheckLiteral(text("oversized"), typ, limits))
	limits = DefaultLimits()
	limits.Work = math.MaxInt64
	require.Error(t, c.CheckLiteral(text("x"), typ, limits))
	schema := scalar(testpilotspb.SCALAR_KIND_TEXT)
	snapshot := boundType(t, c, schema)
	schema.GetSingular().GetScalar().Kind = testpilotspb.SCALAR_KIND_BYTES
	exported := snapshot.Schema()
	exported.GetSingular().GetScalar().Kind = testpilotspb.SCALAR_KIND_BYTES
	require.True(t, proto.Equal(scalar(testpilotspb.SCALAR_KIND_TEXT), snapshot.Schema()))
}

func TestBinderRejectsCrossedCatalogsAndTypedNilUnions(t *testing.T) {
	c := fixtureCatalog(t)
	otherSource := catalogFixture()
	otherSource.File[2].Service[0].Name = proto.String("Other")
	other, err := NewCatalog(otherSource)
	require.NoError(t, err)
	foreign := boundType(t, other, scalar(testpilotspb.SCALAR_KIND_TEXT))
	require.Error(t, c.CheckLiteral(text("x"), foreign, DefaultLimits()))
	require.Error(t, c.CheckLiteral(text("x"), Type{}, DefaultLimits()))
	_, err = c.BindPath(foreign, &testpilotspb.FieldPath{}, DefaultLimits())
	require.Error(t, err)
	require.NotPanics(t, func() {
		_, err := c.BindType(&testpilotspb.ValueType{Shape: (*testpilotspb.ValueType_Singular)(nil)})
		require.Error(t, err)
	})
	require.NotPanics(t, func() {
		err := c.CheckLiteral(&testpilotspb.Value{Value: (*testpilotspb.Value_Text)(nil)}, boundType(t, c, scalar(testpilotspb.SCALAR_KIND_TEXT)), DefaultLimits())
		require.Error(t, err)
	})
	require.NotPanics(t, func() {
		_, err := c.BindPath(boundType(t, c, named("fixture.Payload", false)), &testpilotspb.FieldPath{Segments: []*testpilotspb.FieldPathSegment{{Field: "text", Selector: (*testpilotspb.FieldPathSegment_Presence)(nil)}}}, DefaultLimits())
		require.Error(t, err)
	})
}

func TestNamedPayloadsRespectCollectionCeilings(t *testing.T) {
	c := fixtureCatalog(t)
	typ := boundType(t, c, named("fixture.Payload", false))
	value := &testpilotspb.Value{Value: &testpilotspb.Value_MessageValue{MessageValue: &anypb.Any{TypeUrl: "type.googleapis.com/fixture.Payload", Value: []byte{0x1a, 0, 0x1a, 0, 0x1a, 0}}}}
	limits := DefaultLimits()
	limits.Fanout = 2
	require.Error(t, c.CheckLiteral(value, typ, limits))
}

func TestMessageWorkIsChargedBeforeDecodingAllFields(t *testing.T) {
	c := fixtureCatalog(t)
	typ := boundType(t, c, named("fixture.Payload", false))
	value := &testpilotspb.Value{Value: &testpilotspb.Value_MessageValue{MessageValue: &anypb.Any{TypeUrl: "type.googleapis.com/fixture.Payload", Value: []byte{0x1a, 0, 0x1a, 0, 0x1a, 0, 0xff}}}}
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
	value := func(wire []byte) *testpilotspb.Value {
		return &testpilotspb.Value{Value: &testpilotspb.Value_MessageValue{MessageValue: &anypb.Any{TypeUrl: "type.googleapis.com/groups.Payload", Value: wire}}}
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
	schema := &testpilotspb.ValueType{Shape: &testpilotspb.ValueType_Singular{Singular: &testpilotspb.SingularType{Type: &testpilotspb.SingularType_Enumeration{Enumeration: &testpilotspb.NamedType{ProtobufType: "temporal.server.api.testpilot.v1.InstructionOutcomeStatus"}}}}}
	typ, err := catalog.BindType(schema)
	require.NoError(t, err)
	for number := int32(1); number <= 5; number++ {
		require.NoError(t, catalog.CheckLiteral(&testpilotspb.Value{Value: &testpilotspb.Value_EnumValue{EnumValue: &testpilotspb.EnumValue{Number: number}}}, typ, DefaultLimits()))
	}
	require.Error(t, catalog.CheckLiteral(&testpilotspb.Value{Value: &testpilotspb.Value_EnumValue{EnumValue: &testpilotspb.EnumValue{Number: 99}}}, typ, DefaultLimits()))
}

func TestIntrinsicRunEventKind(t *testing.T) {
	catalog, err := NewCatalog(&descriptorpb.FileDescriptorSet{})
	require.NoError(t, err)
	typ, err := catalog.BindType(&testpilotspb.ValueType{Shape: &testpilotspb.ValueType_Singular{Singular: &testpilotspb.SingularType{Type: &testpilotspb.SingularType_Enumeration{Enumeration: &testpilotspb.NamedType{ProtobufType: "temporal.server.api.testpilot.v1.RunEventKind"}}}}})
	require.NoError(t, err)
	require.Equal(t, testpilotspb.RunEventKind(0).Descriptor(), typ.Enum())
}
