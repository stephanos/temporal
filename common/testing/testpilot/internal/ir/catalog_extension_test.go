package ir

import (
	"testing"

	celpb "cel.dev/expr"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protowire"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/dynamicpb"
	"google.golang.org/protobuf/types/known/anypb"
)

// A payload's extension field is decided by the Case's catalog alone: an extension the process links
// for a message of the same name is never consulted, so its bytes stay unknown wire fields.
func TestCatalogPayloadsNeverResolveLinkedExtensions(t *testing.T) {
	holder := &descriptorpb.FileDescriptorProto{
		Name:    proto.String("fixture/extendable.proto"),
		Package: proto.String("fixture.extendable"),
		Syntax:  proto.String("proto2"),
		MessageType: []*descriptorpb.DescriptorProto{{
			Name:           proto.String("Holder"),
			Field:          []*descriptorpb.FieldDescriptorProto{{Name: proto.String("text"), Number: proto.Int32(1), Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum()}},
			ExtensionRange: []*descriptorpb.DescriptorProto_ExtensionRange{{Start: proto.Int32(100), End: proto.Int32(200)}},
		}},
	}
	linked := &descriptorpb.FileDescriptorProto{
		Name:       proto.String("fixture/linked.proto"),
		Package:    proto.String("fixture.extendable"),
		Syntax:     proto.String("proto2"),
		Dependency: []string{"fixture/extendable.proto"},
		Extension: []*descriptorpb.FieldDescriptorProto{{
			Name: proto.String("note"), Number: proto.Int32(100), Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
			Type: descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(), Extendee: proto.String(".fixture.extendable.Holder"),
		}},
	}
	process, err := protodesc.NewFiles(&descriptorpb.FileDescriptorSet{File: []*descriptorpb.FileDescriptorProto{holder, linked}})
	require.NoError(t, err)
	note, err := process.FindDescriptorByName("fixture.extendable.note")
	require.NoError(t, err)
	require.NoError(t, protoregistry.GlobalTypes.RegisterExtension(dynamicpb.NewExtensionType(note.(protoreflect.ExtensionDescriptor))))
	_, err = protoregistry.GlobalTypes.FindExtensionByNumber("fixture.extendable.Holder", 100)
	require.NoError(t, err, "the process links an extension of the payload's message")

	catalog, err := protodesc.NewFiles(&descriptorpb.FileDescriptorSet{File: []*descriptorpb.FileDescriptorProto{holder}})
	require.NoError(t, err)
	found, err := catalog.FindDescriptorByName("fixture.extendable.Holder")
	require.NoError(t, err)
	descriptor := found.(protoreflect.MessageDescriptor)

	extension := protowire.AppendString(protowire.AppendTag(nil, 100, protowire.BytesType), "linked")
	wire := append(protowire.AppendString(protowire.AppendTag(nil, 1, protowire.BytesType), "kept"), extension...)

	message, err := decodeMessage(&celpb.Value{Kind: &celpb.Value_ObjectValue{ObjectValue: &anypb.Any{TypeUrl: "type.googleapis.com/fixture.extendable.Holder", Value: wire}}}, descriptor)
	require.NoError(t, err)
	require.Equal(t, "kept", message.Get(descriptor.Fields().ByName("text")).String())
	require.Equal(t, protoreflect.RawFields(extension), message.GetUnknown())
	message.Range(func(field protoreflect.FieldDescriptor, _ protoreflect.Value) bool {
		require.False(t, field.IsExtension(), "decoded %s through the process's registry", field.FullName())
		return true
	})

	direct := dynamicpb.NewMessage(descriptor)
	require.NoError(t, unmarshalCatalog(wire, direct, protowire.DefaultRecursionLimit))
	require.Equal(t, protoreflect.RawFields(extension), direct.GetUnknown())
}
