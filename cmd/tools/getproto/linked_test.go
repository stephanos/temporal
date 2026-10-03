package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
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
	require.NotContains(t, names, "temporal/server/api/umpire/v1/ir.proto")

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
