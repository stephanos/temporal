package ir

import (
	"context"
	"fmt"
	"math"
	"testing"

	"github.com/stretchr/testify/require"
	umpirespb "go.temporal.io/server/api/umpire/v1"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/known/anypb"
)

func TestRequestWritesPreservePresenceAndOwnership(t *testing.T) {
	c := fixtureCatalog(t)
	typ := boundType(t, c, named("fixture.Payload", false))
	lookup := fieldPath("labels")
	lookup.Segments[0].Selector = &umpirespb.FieldPathSegment_MapKey{MapKey: &umpirespb.MapKeySelector{Key: text("zero")}}
	var writes []Write
	for _, tc := range []struct {
		path  *umpirespb.FieldPath
		value *umpirespb.Value
	}{
		{fieldPath("child", "optional_text"), text("")}, {fieldPath("success"), text("")}, {lookup, signed("0")},
	} {
		path, err := c.BindPath(typ, tc.path, DefaultLimits())
		require.NoError(t, err)
		writes = append(writes, Write{Path: path, Value: tc.value})
	}
	request, work, err := BuildRequest(context.Background(), typ.Message(), writes, DefaultLimits())
	require.NoError(t, err)
	wire, err := proto.MarshalOptions{Deterministic: true}.Marshal(request)
	require.NoError(t, err)
	require.Equal(t, []byte{0x12, 2, 0x52, 0, 0x22, 8, 0x0a, 4, 'z', 'e', 'r', 'o', 0x10, 0, 0x3a, 0}, wire)
	exact := DefaultLimits()
	exact.Work = work
	exact.Bytes = int64(len(wire))
	_, _, err = BuildRequest(context.Background(), typ.Message(), writes, exact)
	require.NoError(t, err)
	exact.Work--
	_, _, err = BuildRequest(context.Background(), typ.Message(), writes, exact)
	require.Error(t, err)
	exact.Work = work
	exact.Bytes--
	_, _, err = BuildRequest(context.Background(), typ.Message(), writes, exact)
	require.Error(t, err)
	writes[0].Value.Value = &umpirespb.Value_Text{Text: "changed"}
	again, err := proto.MarshalOptions{Deterministic: true}.Marshal(request)
	require.NoError(t, err)
	require.Equal(t, wire, again)
}

func TestRequestRejectsCrossedValuesAndConflictingWrites(t *testing.T) {
	c := fixtureCatalog(t)
	typ := boundType(t, c, named("fixture.Payload", false))
	p, err := c.BindPath(typ, fieldPath("failure"), DefaultLimits())
	require.NoError(t, err)
	for _, v := range []*umpirespb.Value{nil, text("0"), signed("9223372036854775808"), signed("01")} {
		got, _, err := BuildRequest(context.Background(), typ.Message(), []Write{{Path: p, Value: v}}, DefaultLimits())
		require.Error(t, err)
		require.Nil(t, got)
	}
	q, err := c.BindPath(typ, fieldPath("success"), DefaultLimits())
	require.NoError(t, err)
	for _, writes := range [][]Write{{{Path: p, Value: signed("0")}, {Path: p, Value: signed("1")}}, {{Path: p, Value: signed("0")}, {Path: q, Value: text("")}}} {
		got, _, err := BuildRequest(context.Background(), typ.Message(), writes, DefaultLimits())
		require.Error(t, err)
		require.Nil(t, got)
	}
}

func TestRequestNumericWidthsAndCollections(t *testing.T) {
	source := catalogFixture()
	kinds := []descriptorpb.FieldDescriptorProto_Type{descriptorpb.FieldDescriptorProto_TYPE_INT32, descriptorpb.FieldDescriptorProto_TYPE_SINT32, descriptorpb.FieldDescriptorProto_TYPE_SFIXED32, descriptorpb.FieldDescriptorProto_TYPE_INT64, descriptorpb.FieldDescriptorProto_TYPE_SINT64, descriptorpb.FieldDescriptorProto_TYPE_SFIXED64, descriptorpb.FieldDescriptorProto_TYPE_UINT32, descriptorpb.FieldDescriptorProto_TYPE_FIXED32, descriptorpb.FieldDescriptorProto_TYPE_UINT64, descriptorpb.FieldDescriptorProto_TYPE_FIXED64, descriptorpb.FieldDescriptorProto_TYPE_FLOAT, descriptorpb.FieldDescriptorProto_TYPE_DOUBLE, descriptorpb.FieldDescriptorProto_TYPE_BYTES, descriptorpb.FieldDescriptorProto_TYPE_BOOL}
	message := &descriptorpb.DescriptorProto{Name: proto.String("Numbers")}
	for i, kind := range kinds {
		message.Field = append(message.Field, &descriptorpb.FieldDescriptorProto{Name: proto.String(fmt.Sprintf("field%d", i)), Number: proto.Int32(int32(i + 1)), Type: kind.Enum(), Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum()})
	}
	source.File[2].MessageType = append(source.File[2].MessageType, message)
	c, err := NewCatalog(source)
	require.NoError(t, err)
	typ := boundType(t, c, named("fixture.Numbers", false))
	for i, tc := range []struct {
		value, bad *umpirespb.Value
		want       any
	}{
		{signed("-2147483648"), signed("2147483648"), int32(-2147483648)}, {signed("-2147483648"), signed("2147483648"), int32(-2147483648)}, {signed("-2147483648"), signed("2147483648"), int32(-2147483648)},
		{signed("-9223372036854775808"), signed("9223372036854775808"), int64(-9223372036854775808)}, {signed("-9223372036854775808"), signed("9223372036854775808"), int64(-9223372036854775808)}, {signed("-9223372036854775808"), signed("9223372036854775808"), int64(-9223372036854775808)},
		{unsigned("4294967295"), unsigned("4294967296"), uint32(4294967295)}, {unsigned("4294967295"), unsigned("4294967296"), uint32(4294967295)},
		{unsigned("18446744073709551615"), unsigned("18446744073709551616"), uint64(18446744073709551615)}, {unsigned("18446744073709551615"), unsigned("18446744073709551616"), uint64(18446744073709551615)},
		{&umpirespb.Value{Value: &umpirespb.Value_FloatingPoint{FloatingPoint: 0.1}}, &umpirespb.Value{Value: &umpirespb.Value_FloatingPoint{FloatingPoint: math.MaxFloat64}}, float32(0.1)},
		{&umpirespb.Value{Value: &umpirespb.Value_FloatingPoint{FloatingPoint: 0.1}}, text("0.1"), float64(0.1)},
		{&umpirespb.Value{Value: &umpirespb.Value_BytesValue{BytesValue: []byte{1, 2}}}, text("bytes"), []byte{1, 2}}, {boolean(true), signed("1"), true},
	} {
		t.Run(fmt.Sprint(kinds[i]), func(t *testing.T) {
			path, err := c.BindPath(typ, fieldPath(fmt.Sprintf("field%d", i)), DefaultLimits())
			require.NoError(t, err)
			request, _, err := BuildRequest(context.Background(), typ.Message(), []Write{{Path: path, Value: tc.value}}, DefaultLimits())
			require.NoError(t, err)
			actual := request.ProtoReflect().Get(typ.Message().Fields().Get(i)).Interface()
			switch want := tc.want.(type) {
			case float32:
				require.InDelta(t, want, actual, 0.00000001)
			case float64:
				require.InDelta(t, want, actual, 0.00000001)
			default:
				require.Equal(t, want, actual)
			}
			_, _, err = BuildRequest(context.Background(), typ.Message(), []Write{{Path: path, Value: tc.bad}}, DefaultLimits())
			require.Error(t, err)
		})
	}
	payload := boundType(t, c, named("fixture.Payload", false))
	for _, tc := range []struct {
		path  *umpirespb.FieldPath
		value *umpirespb.Value
	}{
		{fieldPath("items"), &umpirespb.Value{Value: &umpirespb.Value_ListValue{ListValue: &umpirespb.ValueList{Values: []*umpirespb.Value{{Value: &umpirespb.Value_MessageValue{MessageValue: &anypb.Any{TypeUrl: "type.googleapis.com/fixture.Payload", Value: []byte{10, 1, 'b'}}}}, {Value: &umpirespb.Value_MessageValue{MessageValue: &anypb.Any{TypeUrl: "type.googleapis.com/fixture.Payload", Value: []byte{10, 1, 'a'}}}}}}}}},
		{fieldPath("labels"), &umpirespb.Value{Value: &umpirespb.Value_MapValue{MapValue: &umpirespb.ValueMap{Entries: []*umpirespb.ValueMapEntry{{Key: text("z"), Value: signed("0")}, {Key: text("a"), Value: signed("1")}}}}}},
		{fieldPath("payload"), &umpirespb.Value{Value: &umpirespb.Value_MessageValue{MessageValue: &anypb.Any{TypeUrl: "type.googleapis.com/custom.Bytes", Value: []byte{1, 2, 3}}}}},
	} {
		path, err := c.BindPath(payload, tc.path, DefaultLimits())
		require.NoError(t, err)
		request, _, err := BuildRequest(context.Background(), payload.Message(), []Write{{Path: path, Value: tc.value}}, DefaultLimits())
		require.NoError(t, err)
		projected, _, err := path.Read(context.Background(), request, DefaultLimits())
		require.NoError(t, err)
		if tc.path.Segments[0].Field == "labels" {
			require.Equal(t, "a", projected.GetMapValue().Entries[0].Key.GetText())
			require.Equal(t, "z", projected.GetMapValue().Entries[1].Key.GetText())
		} else {
			require.True(t, proto.Equal(tc.value, projected))
		}
		before := proto.CloneOf(request)
		proto.Reset(tc.value)
		require.True(t, proto.Equal(before, request))
	}
}
