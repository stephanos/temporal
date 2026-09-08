package main

import (
	"encoding/hex"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/types/descriptorpb"
)

func TestOperationBindingChangesWithSameNamedSchema(t *testing.T) {
	original := basicDescriptorSet(t)
	document, err := buildProjection(original)
	require.NoError(t, err)
	before, err := generateArtifacts(fixtureTestConfiguration, document)
	require.NoError(t, err)
	changed := proto.CloneOf(original)
	for _, file := range changed.File {
		if file.GetName() == "shared/messaging/v1/types.proto" {
			file.MessageType[0].Field[0].Number = proto.Int32(99)
		}
	}
	document, err = buildProjection(changed)
	require.NoError(t, err)
	after, err := generateArtifacts(fixtureTestConfiguration, document)
	require.NoError(t, err)
	require.NotEqual(t, string(before["Fixture/API.lean"]), string(after["Fixture/API.lean"]),
		"same-named transitive field-number changes must invalidate generated operation bindings")
}

func TestOperationSchemaProvenanceIgnoresDescriptorOrder(t *testing.T) {
	original := basicDescriptorSet(t)
	before, err := buildProjection(original)
	require.NoError(t, err)
	reordered := proto.CloneOf(original)
	slices.Reverse(reordered.File)
	for _, file := range reordered.File {
		slices.Reverse(file.MessageType)
		for _, message := range file.MessageType {
			slices.Reverse(message.Field)
			slices.Reverse(message.NestedType)
		}
		for _, service := range file.Service {
			slices.Reverse(service.Method)
		}
	}
	after, err := buildProjection(reordered)
	require.NoError(t, err)
	require.Equal(t, before.OperationSchemas, after.OperationSchemas)
	beforePlan, err := buildLeanPlan(before, fixtureTestConfiguration)
	require.NoError(t, err)
	afterPlan, err := buildLeanPlan(after, fixtureTestConfiguration)
	require.NoError(t, err)
	var first, second strings.Builder
	renderOperationBindings(&first, beforePlan)
	renderOperationBindings(&second, afterPlan)
	require.Equal(t, first.String(), second.String())
	require.Equal(t, []string{
		"fixture.messaging.public.v1.Message", "fixture.messaging.public.v1.Message.AttributesEntry",
		"fixture.messaging.public.v1.Message.Nested", "fixture.messaging.public.v1.Message.Nested.State",
		"fixture.messaging.shared.v1.Shared", "fixture.protobuf.compat.v1.LegacyOptions",
	}, before.OperationSchemas.Closures["fixture.messaging.public.v1.Message"])
	require.Len(t, before.OperationSchemas.Inputs, len(original.File))
}

func TestOperationSchemaTracksFileOptionsAndRecursiveClosure(t *testing.T) {
	original := basicDescriptorSet(t)
	before, err := buildProjection(original)
	require.NoError(t, err)
	changed := proto.CloneOf(original)
	for _, file := range changed.File {
		if file.GetName() == "shared/messaging/v1/types.proto" {
			file.Options = &descriptorpb.FileOptions{Deprecated: proto.Bool(true)}
		}
	}
	after, err := buildProjection(changed)
	require.NoError(t, err)
	require.NotEqual(t, before.OperationSchemas, after.OperationSchemas)
	require.Equal(t, []string{"fixture.messaging.shared.v1.Left", "fixture.messaging.shared.v1.Right"},
		before.OperationSchemas.Closures["fixture.messaging.shared.v1.Left"])
}

func TestNormalizedSchemaInputsRemainValid(t *testing.T) {
	document, err := buildProjection(basicDescriptorSet(t))
	require.NoError(t, err)
	normalized := &descriptorpb.FileDescriptorSet{}
	for _, input := range document.OperationSchemas.Inputs {
		encoded, err := hex.DecodeString(input.Descriptor)
		require.NoError(t, err)
		file := &descriptorpb.FileDescriptorProto{}
		require.NoError(t, proto.Unmarshal(encoded, file))
		normalized.File = append(normalized.File, file)
	}
	_, err = protodesc.NewFiles(normalized)
	require.NoError(t, err)
}
