package ir

import (
	"fmt"
	"math"
	"testing"

	celpb "cel.dev/expr"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/dynamicpb"
	"google.golang.org/protobuf/types/known/anypb"
)

func TestCELSnapshotValueWorkBound(t *testing.T) {
	catalog := fixtureCatalog(t)
	typ := boundType(t, catalog, named("fixture.Payload", false))
	bound, err := SnapshotValueWorkBound(typ, 128, DefaultLimits().Fanout)
	require.NoError(t, err)
	require.Greater(t, bound, int64(16*128))
	_, err = SnapshotValueWorkBound(typ, math.MaxInt64, DefaultLimits().Fanout)
	require.ErrorContains(t, err, "overflow")
	_, err = SnapshotValueWorkBound(typ, 128, 0)
	require.Error(t, err)
}

func TestCELSnapshotValueWorkBoundCoversPackedFieldsAndMapSorting(t *testing.T) {
	set := catalogFixture()
	message := set.File[len(set.File)-1].MessageType[0]
	for i := 0; i < 512; i++ {
		message.Field = append(message.Field, &descriptorpb.FieldDescriptorProto{Name: proto.String(fmt.Sprintf("f%d", i)), Number: proto.Int32(int32(i + 11)), Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum()})
	}
	message.Field = append(message.Field,
		&descriptorpb.FieldDescriptorProto{Name: proto.String("packed"), Number: proto.Int32(523), Label: descriptorpb.FieldDescriptorProto_LABEL_REPEATED.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_BOOL.Enum()},
		&descriptorpb.FieldDescriptorProto{Name: proto.String("numbers"), Number: proto.Int32(524), Label: descriptorpb.FieldDescriptorProto_LABEL_REPEATED.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_MESSAGE.Enum(), TypeName: proto.String(".fixture.Payload.NumbersEntry")})
	message.NestedType = append(message.NestedType, &descriptorpb.DescriptorProto{Name: proto.String("NumbersEntry"), Options: &descriptorpb.MessageOptions{MapEntry: proto.Bool(true)}, Field: []*descriptorpb.FieldDescriptorProto{
		{Name: proto.String("key"), Number: proto.Int32(1), Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_INT64.Enum()},
		{Name: proto.String("value"), Number: proto.Int32(2), Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_BOOL.Enum()},
	}})
	catalog, err := NewCatalog(set)
	require.NoError(t, err)
	typ := boundType(t, catalog, named("fixture.Payload", false))
	for _, name := range []string{"wide", "packed", "string map", "numeric map"} {
		t.Run(name, func(t *testing.T) {
			value := dynamicpb.NewMessage(typ.Message())
			fields := typ.Message().Fields()
			switch name {
			case "wide":
				for i := 0; i < 512; i++ {
					value.Set(fields.ByNumber(protoreflect.FieldNumber(i+11)), protoreflect.ValueOfString("x"))
				}
			case "packed":
				list := value.Mutable(fields.ByName("packed")).List()
				for i := 0; i < 10000; i++ {
					list.Append(protoreflect.ValueOfBool(true))
				}
			case "string map":
				items := value.Mutable(fields.ByName("labels")).Map()
				for i := 0; i < 512; i++ {
					items.Set(protoreflect.ValueOfString(fmt.Sprintf("key%d", i)).MapKey(), protoreflect.ValueOfInt64(1))
				}
			case "numeric map":
				items := value.Mutable(fields.ByName("numbers")).Map()
				for i := 0; i < 512; i++ {
					items.Set(protoreflect.ValueOfInt64(int64(i)).MapKey(), protoreflect.ValueOfBool(true))
				}
			default:
				t.Fatalf("unknown native value shape %q", name)
			}
			envelope, err := anypb.New(value)
			require.NoError(t, err)
			source := &celpb.Value{Kind: &celpb.Value_ObjectValue{ObjectValue: envelope}}
			bound, err := SnapshotValueWorkBound(typ, int64(proto.Size(source)), DefaultLimits().Fanout)
			require.NoError(t, err)
			limits := DefaultLimits()
			limits.Work = bound
			result, work, err := SnapshotValue(t.Context(), source, typ, limits)
			require.NoError(t, err)
			require.True(t, proto.Equal(source, result))
			require.LessOrEqual(t, work, bound)
		})
	}
}
