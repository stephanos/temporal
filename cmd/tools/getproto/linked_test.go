package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/descriptorpb"
)

func TestLinkedModelDescriptors(t *testing.T) {
	input := filepath.Join(t.TempDir(), "api.binpb")
	seed, err := proto.Marshal(&descriptorpb.FileDescriptorSet{})
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(input, seed, 0644))

	first, names, err := linkedModelDescriptors(input)
	require.NoError(t, err)
	second, again, err := linkedModelDescriptors(input)
	require.NoError(t, err)
	require.Equal(t, first, second)
	require.Equal(t, names, again)
	require.Contains(t, names, "temporal/api/workflowservice/v1/service.proto")
	require.Contains(t, names, "temporal/server/api/testpilot/v1/case.proto")
	for _, name := range names {
		require.False(t, isUmpireSchema(name), "%s is generated into ir-scalapb.jar", name)
	}

	var set descriptorpb.FileDescriptorSet
	require.NoError(t, proto.Unmarshal(first, &set))
	require.Greater(t, len(set.File), len(names))
	for i := 1; i < len(set.File); i++ {
		require.Less(t, set.File[i-1].GetName(), set.File[i].GetName())
	}
}

func TestLinkedModelDescriptorsMissingInput(t *testing.T) {
	_, _, err := linkedModelDescriptors(filepath.Join(t.TempDir(), "missing.binpb"))
	require.ErrorContains(t, err, "make proto/api.binpb")
}

func TestLinkedModelDescriptorsRejectInternalChangeWithUnchangedAPI(t *testing.T) {
	input := filepath.Join(t.TempDir(), "api.binpb")
	seed, err := proto.Marshal(&descriptorpb.FileDescriptorSet{})
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(input, seed, 0644))
	linked, _, err := linkedModelDescriptors(input)
	require.NoError(t, err)
	var set descriptorpb.FileDescriptorSet
	require.NoError(t, proto.Unmarshal(linked, &set))
	current := &descriptorpb.FileDescriptorSet{}
	for _, file := range set.File {
		if isModelInternalDescriptor(file.GetName()) {
			current.File = append(current.File, proto.Clone(file).(*descriptorpb.FileDescriptorProto))
		}
	}
	require.NoError(t, checkCurrentInternal(linked, current))
	current.File = append(current.File, &descriptorpb.FileDescriptorProto{Name: proto.String("temporal/server/api/testpilot/v1/new.proto")})
	require.ErrorContains(t, checkCurrentInternal(linked, current), "make protoc")
	current.File = current.File[:len(current.File)-1]
	for _, file := range current.File {
		if file.GetName() == "temporal/server/api/testpilot/v1/case.proto" {
			file.MessageType = append(file.MessageType, &descriptorpb.DescriptorProto{Name: proto.String("NewModelField")})
		}
	}
	unchanged, err := os.ReadFile(input)
	require.NoError(t, err)
	require.Equal(t, seed, unchanged)
	require.ErrorContains(t, checkCurrentInternal(linked, current), "make protoc")
}

// A registry whose IR schema is a root importing a second file of the IR, in place of the IR's own.
func multiFileSchemaRegistry(t *testing.T) *protoregistry.Files {
	t.Helper()
	registry := new(protoregistry.Files)
	protoregistry.GlobalFiles.RangeFiles(func(file protoreflect.FileDescriptor) bool {
		if !isUmpireSchema(file.Path()) {
			require.NoError(t, registry.RegisterFile(file))
		}
		return true
	})
	const pkg = "temporal.server.api.umpire.v1"
	const common = umpireSchemaDirectory + "v1/common.proto"
	for _, file := range []*descriptorpb.FileDescriptorProto{
		{Name: proto.String(common), Package: proto.String(pkg), Syntax: proto.String("proto3"),
			MessageType: []*descriptorpb.DescriptorProto{{Name: proto.String("Position")}}},
		{Name: proto.String(umpireSchemaRoot), Package: proto.String(pkg), Syntax: proto.String("proto3"), Dependency: []string{common},
			MessageType: []*descriptorpb.DescriptorProto{{Name: proto.String("Model"), Field: []*descriptorpb.FieldDescriptorProto{{
				Name: proto.String("position"), Number: proto.Int32(1), Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
				Type: descriptorpb.FieldDescriptorProto_TYPE_MESSAGE.Enum(), TypeName: proto.String("." + pkg + ".Position"), JsonName: proto.String("position")}}}}},
	} {
		descriptor, err := protodesc.NewFile(file, registry)
		require.NoError(t, err)
		require.NoError(t, registry.RegisterFile(descriptor))
	}
	return registry
}

func TestLinkedModelDescriptorsKeepTheWholeSchemaClosureApart(t *testing.T) {
	input := filepath.Join(t.TempDir(), "api.binpb")
	seed, err := proto.Marshal(&descriptorpb.FileDescriptorSet{})
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(input, seed, 0644))
	linked, names, err := linkModelDescriptors(input, multiFileSchemaRegistry(t))
	require.NoError(t, err)
	require.NotContains(t, names, umpireSchemaRoot)
	require.NotContains(t, names, umpireSchemaDirectory+"v1/common.proto")
	require.Contains(t, names, "temporal/server/api/testpilot/v1/case.proto")

	var set descriptorpb.FileDescriptorSet
	require.NoError(t, proto.Unmarshal(linked, &set))
	current := &descriptorpb.FileDescriptorSet{}
	var schema []string
	for _, file := range set.File {
		if isModelInternalDescriptor(file.GetName()) {
			current.File = append(current.File, proto.Clone(file).(*descriptorpb.FileDescriptorProto))
		}
		if isUmpireSchema(file.GetName()) {
			schema = append(schema, file.GetName())
		}
	}
	require.Equal(t, []string{umpireSchemaDirectory + "v1/common.proto", umpireSchemaRoot}, schema)
	require.NoError(t, checkCurrentInternal(linked, current))

	// An imported file changed while the root did not: the linked descriptors are stale.
	for _, file := range current.File {
		if file.GetName() == umpireSchemaDirectory+"v1/common.proto" {
			file.MessageType = append(file.MessageType, &descriptorpb.DescriptorProto{Name: proto.String("Span")})
		}
	}
	require.ErrorContains(t, checkCurrentInternal(linked, current), "linked descriptor "+umpireSchemaDirectory+"v1/common.proto differs")

	// A file the schema newly imports is not linked yet.
	current.File = append(current.File, &descriptorpb.FileDescriptorProto{Name: proto.String(umpireSchemaDirectory + "v1/value.proto")})
	for _, file := range current.File {
		if file.GetName() == umpireSchemaDirectory+"v1/common.proto" {
			file.MessageType = file.MessageType[:len(file.MessageType)-1]
		}
	}
	require.ErrorContains(t, checkCurrentInternal(linked, current), "current internal proto "+umpireSchemaDirectory+"v1/value.proto is absent")
}
