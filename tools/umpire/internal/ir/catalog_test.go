package ir

import (
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func catalogFixture() *descriptorpb.FileDescriptorSet {
	optional := descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL
	repeated := descriptorpb.FieldDescriptorProto_LABEL_REPEATED
	field := func(name string, number int32, kind descriptorpb.FieldDescriptorProto_Type, named string) *descriptorpb.FieldDescriptorProto {
		f := &descriptorpb.FieldDescriptorProto{Name: proto.String(name), Number: proto.Int32(number), Label: &optional, Type: &kind}
		if named != "" {
			f.TypeName = proto.String(named)
		}
		return f
	}
	msg := &descriptorpb.DescriptorProto{Name: proto.String("Payload"), Field: []*descriptorpb.FieldDescriptorProto{
		field("text", 1, descriptorpb.FieldDescriptorProto_TYPE_STRING, ""),
		field("child", 2, descriptorpb.FieldDescriptorProto_TYPE_MESSAGE, ".fixture.Payload"),
		field("items", 3, descriptorpb.FieldDescriptorProto_TYPE_MESSAGE, ".fixture.Payload"),
		field("labels", 4, descriptorpb.FieldDescriptorProto_TYPE_MESSAGE, ".fixture.Payload.LabelsEntry"),
		field("when", 5, descriptorpb.FieldDescriptorProto_TYPE_MESSAGE, ".google.protobuf.Timestamp"),
		field("payload", 6, descriptorpb.FieldDescriptorProto_TYPE_MESSAGE, ".google.protobuf.Any"),
		field("success", 7, descriptorpb.FieldDescriptorProto_TYPE_STRING, ""),
		field("failure", 8, descriptorpb.FieldDescriptorProto_TYPE_INT64, ""),
		field("state", 9, descriptorpb.FieldDescriptorProto_TYPE_ENUM, ".fixture.State"),
		field("optional_text", 10, descriptorpb.FieldDescriptorProto_TYPE_STRING, ""),
	}}
	msg.Field[2].Label = &repeated
	msg.Field[3].Label = &repeated
	msg.OneofDecl = []*descriptorpb.OneofDescriptorProto{{Name: proto.String("result")}, {Name: proto.String("_optional_text")}}
	msg.Field[6].OneofIndex = proto.Int32(0)
	msg.Field[7].OneofIndex = proto.Int32(0)
	msg.Field[9].OneofIndex = proto.Int32(1)
	msg.Field[9].Proto3Optional = proto.Bool(true)
	msg.NestedType = []*descriptorpb.DescriptorProto{{Name: proto.String("LabelsEntry"), Options: &descriptorpb.MessageOptions{MapEntry: proto.Bool(true)}, Field: []*descriptorpb.FieldDescriptorProto{field("key", 1, descriptorpb.FieldDescriptorProto_TYPE_STRING, ""), field("value", 2, descriptorpb.FieldDescriptorProto_TYPE_INT64, "")}}}
	return &descriptorpb.FileDescriptorSet{File: []*descriptorpb.FileDescriptorProto{
		protodesc.ToFileDescriptorProto(anypb.File_google_protobuf_any_proto),
		protodesc.ToFileDescriptorProto(timestamppb.File_google_protobuf_timestamp_proto),
		{Name: proto.String("fixture.proto"), Package: proto.String("fixture"), Syntax: proto.String("proto3"), Dependency: []string{"google/protobuf/any.proto", "google/protobuf/timestamp.proto"}, MessageType: []*descriptorpb.DescriptorProto{msg}, EnumType: []*descriptorpb.EnumDescriptorProto{{Name: proto.String("State"), Value: []*descriptorpb.EnumValueDescriptorProto{{Name: proto.String("UNKNOWN"), Number: proto.Int32(0)}, {Name: proto.String("READY"), Number: proto.Int32(1)}}}}, Service: []*descriptorpb.ServiceDescriptorProto{
			{Name: proto.String("Records"), Method: []*descriptorpb.MethodDescriptorProto{{Name: proto.String("Read"), InputType: proto.String(".fixture.Payload"), OutputType: proto.String(".fixture.Payload")}}},
			{Name: proto.String("Clock"), Method: []*descriptorpb.MethodDescriptorProto{{Name: proto.String("Now"), InputType: proto.String(".google.protobuf.Any"), OutputType: proto.String(".google.protobuf.Timestamp")}, {Name: proto.String("Watch"), InputType: proto.String(".fixture.Payload"), OutputType: proto.String(".fixture.Payload"), ServerStreaming: proto.Bool(true)}}},
		}},
	}}
}

func TestCatalogBindsUnaryMethodsAndSnapshotsDescriptors(t *testing.T) {
	source := catalogFixture()
	catalog, err := NewCatalog(source)
	require.NoError(t, err)
	for method, output := range map[string]string{"/fixture.Records/Read": "fixture.Payload", "/fixture.Clock/Now": "google.protobuf.Timestamp"} {
		descriptor, err := catalog.Method(method)
		require.NoError(t, err)
		require.Equal(t, output, string(descriptor.Output().FullName()))
	}
	identity := catalog.Identity()
	source.File[2].Service[0].Method[0].Name = proto.String("Changed")
	_, err = catalog.Method("/fixture.Records/Read")
	require.NoError(t, err)
	require.Equal(t, identity, catalog.Identity())
	for _, method := range []string{"", "fixture.Records.Read", "/fixture.Records/Missing", "/fixture.Clock/Watch"} {
		_, err := catalog.Method(method)
		require.Error(t, err)
	}
}

func TestCatalogRejectsMalformedDescriptorGraphs(t *testing.T) {
	for name, mutate := range map[string]func(*descriptorpb.FileDescriptorSet){
		"unresolved":     func(s *descriptorpb.FileDescriptorSet) { s.File = s.File[2:] },
		"duplicate file": func(s *descriptorpb.FileDescriptorSet) { s.File = append(s.File, proto.CloneOf(s.File[2])) },
		"duplicate method": func(s *descriptorpb.FileDescriptorSet) {
			svc := s.File[2].Service[0]
			svc.Method = append(svc.Method, proto.CloneOf(svc.Method[0]))
		},
		"malformed": func(s *descriptorpb.FileDescriptorSet) { s.File[2].MessageType[0].Field[0].Number = proto.Int32(-1) },
		"nil file":  func(s *descriptorpb.FileDescriptorSet) { s.File = append(s.File, nil) },
	} {
		t.Run(name, func(t *testing.T) {
			source := catalogFixture()
			mutate(source)
			_, err := NewCatalog(source)
			require.Error(t, err)
		})
	}
	_, err := NewCatalog(nil)
	require.Error(t, err)
}

func TestCatalogRejectsConflictingIntrinsicStatus(t *testing.T) {
	source := catalogFixture()
	source.File = append(source.File, &descriptorpb.FileDescriptorProto{Name: proto.String("conflict.proto"), Package: proto.String("temporal.server.api.umpire.v1"), Syntax: proto.String("proto3"), EnumType: []*descriptorpb.EnumDescriptorProto{{Name: proto.String("InstructionOutcomeStatus"), Value: []*descriptorpb.EnumValueDescriptorProto{{Name: proto.String("WRONG"), Number: proto.Int32(0)}}}}})
	_, err := NewCatalog(source)
	require.Error(t, err)
}
