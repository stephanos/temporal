package api

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"
)

func TestProjectionCoversMessagesOneofsMapsRecursionAndStreamingRPCs(t *testing.T) {
	t.Parallel()

	set := testDescriptorSet()
	projection, err := buildProjection(set)
	require.NoError(t, err)
	require.Len(t, projection.Services, 2)
	require.Equal(t, sourcePublic, projection.Services[0].Source)
	require.False(t, projection.Services[0].Methods[0].ClientStreaming)
	require.False(t, projection.Services[0].Methods[0].ServerStreaming)
	require.True(t, projection.Services[0].Methods[1].ClientStreaming)
	require.True(t, projection.Services[0].Methods[1].ServerStreaming)

	node := findMessage(t, projection, "temporal.api.test.v1.Node")
	require.Len(t, node.Oneofs, 1)
	require.True(t, findField(t, node, "next").Recursive)
	require.True(t, findField(t, node, "attributes").Map)
	require.Equal(t, "choice", findField(t, node, "text").Oneof)

	artifacts, manifest, err := generateArtifacts("go.temporal.io/api", "v1.2.3", nil, projection)
	require.NoError(t, err)
	require.Equal(t, 2, manifest.Services)
	require.Equal(t, 3, manifest.Methods)
	require.Contains(t, string(artifacts["Temporal/Generated/Types.lean"]), "Temporal.Proto.MessageRef")
	require.Contains(t, string(artifacts["Temporal/Generated/Types.lean"]), "inductive Temporal_Api_Test_V1_Node_Choice")
	require.Contains(t, string(artifacts["Temporal/Generated/GRPC/Public.lean"]), "clientStreaming := true, serverStreaming := true")
}

func TestDescriptorMergeIsDeterministicAndRejectsConflicts(t *testing.T) {
	t.Parallel()

	set := testDescriptorSet()
	first, err := proto.Marshal(set)
	require.NoError(t, err)
	reversed := proto.Clone(set).(*descriptorpb.FileDescriptorSet)
	slices.Reverse(reversed.File)
	second, err := proto.Marshal(reversed)
	require.NoError(t, err)
	mergedFirst, err := mergeDescriptorInputs([]descriptorInput{newDescriptorInput("first", "first", first)})
	require.NoError(t, err)
	mergedSecond, err := mergeDescriptorInputs([]descriptorInput{newDescriptorInput("second", "second", second)})
	require.NoError(t, err)
	firstProjection, err := buildProjection(mergedFirst)
	require.NoError(t, err)
	secondProjection, err := buildProjection(mergedSecond)
	require.NoError(t, err)
	require.Equal(t, firstProjection, secondProjection)

	conflicting := proto.Clone(set).(*descriptorpb.FileDescriptorSet)
	conflicting.File[0].Package = proto.String("temporal.api.changed.v1")
	conflictingBytes, err := proto.Marshal(conflicting)
	require.NoError(t, err)
	_, err = mergeDescriptorInputs([]descriptorInput{
		newDescriptorInput("first", "first", first),
		newDescriptorInput("conflicting", "conflicting", conflictingBytes),
	})
	require.ErrorContains(t, err, "conflicting definitions")
}

func TestPublishAndCheckArtifactsDetectDriftAndRemoveOnlyManagedStaleFiles(t *testing.T) {
	t.Parallel()

	outputRoot := t.TempDir()
	projection, err := buildProjection(testDescriptorSet())
	require.NoError(t, err)
	artifacts, manifest, err := generateArtifacts("go.temporal.io/api", "v1.2.3", nil, projection)
	require.NoError(t, err)
	require.NoError(t, publishArtifacts(outputRoot, artifacts, manifest))
	require.NoError(t, checkArtifacts(outputRoot, artifacts))

	typesPath := filepath.Join(outputRoot, "Temporal", "Generated", "Types.lean")
	require.NoError(t, os.WriteFile(typesPath, []byte("stale"), 0o600))
	require.ErrorContains(t, checkArtifacts(outputRoot, artifacts), "Types.lean (stale)")
	require.NoError(t, publishArtifacts(outputRoot, artifacts, manifest))
	require.NoError(t, checkArtifacts(outputRoot, artifacts))
	require.Error(t, validateManagedPath("../authored.lean"))
}

func TestPackageHasTemporalProtoIgnoresCompatibilityDescriptors(t *testing.T) {
	t.Parallel()

	directory := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(directory, "public.pb.go"), []byte("// source: temporal/api/test/v1/message.proto\n"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(directory, "compat.pb.go"), []byte("// source: nexusannotations/v1/options.proto\n"), 0o600))
	found, err := packageHasTemporalProto(directory, "compat.pb.go,public.pb.go")
	require.NoError(t, err)
	require.True(t, found)
	found, err = packageHasTemporalProto(directory, "compat.pb.go")
	require.NoError(t, err)
	require.False(t, found)
}

func findMessage(t *testing.T, projection projection, fullName string) messageProjection {
	t.Helper()
	for _, message := range projection.Messages {
		if message.FullName == fullName {
			return message
		}
	}
	require.FailNow(t, "message not found", fullName)
	return messageProjection{}
}

func findField(t *testing.T, message messageProjection, name string) fieldProjection {
	t.Helper()
	for _, field := range message.Fields {
		if field.Name == name {
			return field
		}
	}
	require.FailNow(t, "field not found", name)
	return fieldProjection{}
}

func testDescriptorSet() *descriptorpb.FileDescriptorSet {
	publicPath := "temporal/api/test/v1/test.proto"
	publicPackage := "temporal.api.test.v1"
	internalPath := "temporal/server/api/test/v1/service.proto"
	internalPackage := "temporal.server.api.test.v1"
	mapEntry := &descriptorpb.DescriptorProto{
		Name: proto.String("AttributesEntry"), Options: &descriptorpb.MessageOptions{MapEntry: proto.Bool(true)},
		Field: []*descriptorpb.FieldDescriptorProto{
			{Name: proto.String("key"), Number: proto.Int32(1), Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum()},
			{Name: proto.String("value"), Number: proto.Int32(2), Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_INT32.Enum()},
		},
	}
	node := &descriptorpb.DescriptorProto{
		Name: proto.String("Node"), OneofDecl: []*descriptorpb.OneofDescriptorProto{{Name: proto.String("choice")}},
		NestedType: []*descriptorpb.DescriptorProto{mapEntry},
		Field: []*descriptorpb.FieldDescriptorProto{
			{Name: proto.String("name"), JsonName: proto.String("name"), Number: proto.Int32(1), Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum()},
			{Name: proto.String("next"), JsonName: proto.String("next"), Number: proto.Int32(2), Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_MESSAGE.Enum(), TypeName: proto.String(".temporal.api.test.v1.Node")},
			{Name: proto.String("labels"), JsonName: proto.String("labels"), Number: proto.Int32(3), Label: descriptorpb.FieldDescriptorProto_LABEL_REPEATED.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum()},
			{Name: proto.String("attributes"), JsonName: proto.String("attributes"), Number: proto.Int32(4), Label: descriptorpb.FieldDescriptorProto_LABEL_REPEATED.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_MESSAGE.Enum(), TypeName: proto.String(".temporal.api.test.v1.Node.AttributesEntry")},
			{Name: proto.String("text"), JsonName: proto.String("text"), Number: proto.Int32(5), Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(), OneofIndex: proto.Int32(0)},
			{Name: proto.String("number"), JsonName: proto.String("number"), Number: proto.Int32(6), Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_INT32.Enum(), OneofIndex: proto.Int32(0)},
		},
	}
	public := &descriptorpb.FileDescriptorProto{
		Name: proto.String(publicPath), Package: proto.String(publicPackage), Syntax: proto.String("proto3"),
		MessageType: []*descriptorpb.DescriptorProto{
			node,
			{Name: proto.String("Request"), Field: []*descriptorpb.FieldDescriptorProto{{Name: proto.String("node"), JsonName: proto.String("node"), Number: proto.Int32(1), Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_MESSAGE.Enum(), TypeName: proto.String(".temporal.api.test.v1.Node")}}},
			{Name: proto.String("Response")},
		},
		EnumType: []*descriptorpb.EnumDescriptorProto{{
			Name: proto.String("State"), Value: []*descriptorpb.EnumValueDescriptorProto{
				{Name: proto.String("STATE_UNSPECIFIED"), Number: proto.Int32(0)},
				{Name: proto.String("STATE_READY"), Number: proto.Int32(1)},
			},
		}},
		Service: []*descriptorpb.ServiceDescriptorProto{{
			Name: proto.String("TestService"), Method: []*descriptorpb.MethodDescriptorProto{
				{Name: proto.String("Call"), InputType: proto.String(".temporal.api.test.v1.Request"), OutputType: proto.String(".temporal.api.test.v1.Response")},
				{Name: proto.String("Stream"), InputType: proto.String(".temporal.api.test.v1.Request"), OutputType: proto.String(".temporal.api.test.v1.Response"), ClientStreaming: proto.Bool(true), ServerStreaming: proto.Bool(true)},
			},
		}},
	}
	internal := &descriptorpb.FileDescriptorProto{
		Name: proto.String(internalPath), Package: proto.String(internalPackage), Syntax: proto.String("proto3"),
		Dependency: []string{publicPath},
		Service: []*descriptorpb.ServiceDescriptorProto{{
			Name: proto.String("InternalService"), Method: []*descriptorpb.MethodDescriptorProto{{
				Name: proto.String("Watch"), InputType: proto.String(".temporal.api.test.v1.Request"), OutputType: proto.String(".temporal.api.test.v1.Response"), ServerStreaming: proto.Bool(true),
			}},
		}},
	}
	return &descriptorpb.FileDescriptorSet{File: []*descriptorpb.FileDescriptorProto{public, internal}}
}
