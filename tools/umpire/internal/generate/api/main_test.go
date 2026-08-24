package api

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"
)

func TestProjectionCoversMessagesOneofsMapsRecursionAndStreamingRPCs(t *testing.T) {
	t.Parallel()
	projection, err := buildProjection(basicDescriptorSet(t), fixtureTestConfiguration.Classify)
	require.NoError(t, err)
	require.Len(t, projection.Services, 1)
	require.Equal(t, sourceInternal, projection.Services[0].Source)
	require.False(t, projection.Services[0].Methods[0].ClientStreaming)
	require.False(t, projection.Services[0].Methods[0].ServerStreaming)
	require.True(t, projection.Services[0].Methods[3].ClientStreaming)
	require.True(t, projection.Services[0].Methods[3].ServerStreaming)

	model := findMessage(t, projection, "fixture.messaging.public.v1.Message")
	require.Len(t, model.Oneofs, 1)
	require.True(t, findField(t, model, "attributes").Map)
	require.Equal(t, "choice", findField(t, model, "text").Oneof)
	require.Empty(t, findField(t, model, "note").Oneof)
	require.True(t, findField(t, model, "note").Presence)
	plan, err := buildLeanPlan(projection, fixtureTestConfiguration)
	require.NoError(t, err)
	require.True(t, plan.fields["fixture.messaging.shared.v1.Left.right"].Recursive)

	artifacts, manifest, err := generateArtifacts(fixtureTestConfiguration, nil, projection)
	require.NoError(t, err)
	require.Equal(t, "umpire/protobuf-lean/v1", manifest.FormatVersion)
	require.Equal(t, 1, manifest.Services)
	require.Equal(t, 4, manifest.Methods)
	types := string(artifacts[fixtureTestConfiguration.Layout.TypesPath])
	require.Contains(t, types, "namespace Fixture.Messaging.Public.V1")
	require.Contains(t, types, "def stateReady : Message.Nested.State")
	require.Contains(t, types, "inductive Message.Choice")
	require.Contains(t, types, "right : Option Fixture.Proto.MessageRef")
	require.Contains(t, types, "note : Option String")
	grpc := string(artifacts[testGRPCPath(fixtureTestConfiguration, sourceInternal)])
	require.Contains(t, grpc, "namespace Fixture.Messaging.Internal.V1.MessagingService")
	require.Contains(t, grpc, "def chat : Fixture.Proto.Method Fixture.Messaging.Public.V1.Message Fixture.Messaging.Public.V1.Reply")
	require.Contains(t, grpc, "clientStreaming := true, serverStreaming := true")
	require.False(t, strings.HasSuffix(types, "\n\n"))
}

func TestDescriptorMergeIsDeterministicAndRejectsConflicts(t *testing.T) {
	t.Parallel()
	set := basicDescriptorSet(t)
	first, err := proto.Marshal(set)
	require.NoError(t, err)
	reversed := proto.Clone(set).(*descriptorpb.FileDescriptorSet)
	slices.Reverse(reversed.File)
	second, err := proto.Marshal(reversed)
	require.NoError(t, err)
	mergedFirst, err := mergeDescriptorInputs([]descriptorInput{mustDescriptorInput(t, "first", "first", first)})
	require.NoError(t, err)
	mergedSecond, err := mergeDescriptorInputs([]descriptorInput{mustDescriptorInput(t, "second", "second", second)})
	require.NoError(t, err)
	firstProjection, err := buildProjection(mergedFirst, fixtureTestConfiguration.Classify)
	require.NoError(t, err)
	secondProjection, err := buildProjection(mergedSecond, fixtureTestConfiguration.Classify)
	require.NoError(t, err)
	require.Equal(t, firstProjection, secondProjection)
	firstArtifacts, _, err := generateArtifacts(fixtureTestConfiguration, nil, firstProjection)
	require.NoError(t, err)
	secondArtifacts, _, err := generateArtifacts(fixtureTestConfiguration, nil, secondProjection)
	require.NoError(t, err)
	require.Equal(t, firstArtifacts, secondArtifacts)

	conflicting := proto.Clone(set).(*descriptorpb.FileDescriptorSet)
	conflicting.File[0].Package = proto.String("changed.v1")
	conflictingBytes, err := proto.Marshal(conflicting)
	require.NoError(t, err)
	_, err = mergeDescriptorInputs([]descriptorInput{
		mustDescriptorInput(t, "first", "first", first),
		mustDescriptorInput(t, "conflicting", "conflicting", conflictingBytes),
	})
	require.ErrorContains(t, err, "conflicting definitions")
}

func TestDescriptorInputDigestIgnoresFileOrder(t *testing.T) {
	t.Parallel()
	set := basicDescriptorSet(t)
	first, err := proto.Marshal(set)
	require.NoError(t, err)
	reversed := proto.Clone(set).(*descriptorpb.FileDescriptorSet)
	slices.Reverse(reversed.File)
	second, err := proto.Marshal(reversed)
	require.NoError(t, err)
	require.Equal(t,
		mustDescriptorInput(t, "first", "first", first).Digest,
		mustDescriptorInput(t, "second", "second", second).Digest,
	)
}

func TestProjectionPreservesDescriptorMetadata(t *testing.T) {
	t.Parallel()
	projection, err := buildProjection(basicDescriptorSet(t), fixtureTestConfiguration.Classify)
	require.NoError(t, err)
	artifacts, _, err := generateArtifacts(fixtureTestConfiguration, nil, projection)
	require.NoError(t, err)

	var document map[string]any
	require.NoError(t, json.Unmarshal(artifacts[fixtureTestConfiguration.Layout.SchemaPath], &document))
	message := findJSONByFullName(t, document["messages"].([]any), "fixture.protobuf.compat.v1.LegacyOptions")
	fields := message["fields"].([]any)
	require.Equal(t, true, fields[0].(map[string]any)["required"])
	require.Equal(t, true, fields[1].(map[string]any)["hasDefault"])
	require.Equal(t, "7", fields[1].(map[string]any)["defaultValue"])
	require.Equal(t, true, fields[2].(map[string]any)["packed"])
	enum := findJSONByFullName(t, document["enums"].([]any), "fixture.messaging.public.v1.Message.Nested.State")
	enumValue := enum["values"].([]any)[1].(map[string]any)
	require.Equal(t, "fixture.messaging.public.v1.Message.Nested.STATE_READY", enumValue["fullName"])
	require.Equal(t, "Fixture.Messaging.Public.V1.Message.Nested.State.stateReady", enumValue["leanName"])
	require.Equal(t, true, enumValue["deprecated"])
	service := document["services"].([]any)[0].(map[string]any)
	require.Equal(t, "Fixture.Protobuf.Compat.V1.LegacyOptions.name", fields[0].(map[string]any)["leanName"])
	method := service["methods"].([]any)[3].(map[string]any)
	require.Equal(t, "Fixture.Messaging.Internal.V1.MessagingService.chat", method["leanName"])
	require.Equal(t, true, method["deprecated"])

	types := string(artifacts[fixtureTestConfiguration.Layout.TypesPath])
	require.Contains(t, types, "name : String")
	require.Contains(t, string(artifacts[testCatalogPath(fixtureTestConfiguration, sourceExternal)]), "path := \"compat/protobuf/v1/options.proto\"")
}

func TestGeneratedArtifactsEmitEmptySourceServiceModules(t *testing.T) {
	t.Parallel()
	projection, err := buildProjection(basicDescriptorSet(t), fixtureTestConfiguration.Classify)
	require.NoError(t, err)
	artifacts, _, err := generateArtifacts(fixtureTestConfiguration, nil, projection)
	require.NoError(t, err)
	externalGRPC := testGRPCPath(fixtureTestConfiguration, sourceExternal)
	require.Contains(t, artifacts, externalGRPC)
	require.Contains(t, string(artifacts[fixtureTestConfiguration.Layout.UmbrellaPath]), "import "+leanModuleName(externalGRPC))
	require.NotContains(t, string(artifacts[externalGRPC]), "MessagingService")
	require.Contains(t, string(artifacts[testGRPCPath(fixtureTestConfiguration, sourceInternal)]), "fixture.messaging.internal.v1.MessagingService.Chat")
}

func TestGeneratedSchemaUsesArraysForCollections(t *testing.T) {
	t.Parallel()
	projection, err := buildProjection(basicDescriptorSet(t), fixtureTestConfiguration.Classify)
	require.NoError(t, err)
	artifacts, _, err := generateArtifacts(fixtureTestConfiguration, nil, projection)
	require.NoError(t, err)
	require.NotContains(t, string(artifacts[fixtureTestConfiguration.Layout.SchemaPath]), ": null")

	var document schemaProjection
	require.NoError(t, json.Unmarshal(artifacts[fixtureTestConfiguration.Layout.SchemaPath], &document))
	message := findSchemaMessage(t, document, "fixture.messaging.public.v1.Message")
	require.Len(t, message.Oneofs, 1)
	require.Equal(t, "Fixture.Messaging.Public.V1.Message.Choice.text", findSchemaField(t, message, "text").LeanName)
	require.Equal(t, "Fixture.Messaging.Public.V1.Message.note", findSchemaField(t, message, "note").LeanName)
	require.Equal(t, []string{"fixture.messaging.public.v1.Message.text", "fixture.messaging.public.v1.Message.number"}, message.Oneofs[0].FieldNames)
}

func TestGeneratedSchemaUsesArraysForEmptyCollections(t *testing.T) {
	t.Parallel()
	artifacts, _, err := generateArtifacts(fixtureTestConfiguration, nil, projection{})
	require.NoError(t, err)
	require.NotContains(t, string(artifacts[fixtureTestConfiguration.Layout.SchemaPath]), ": null")
}

func TestGeneratedManifestUsesArraysForEmptyCollections(t *testing.T) {
	configuration := testGenerationConfig("Empty")
	configuration.Sources = nil
	configuration.Groups = []sourceGroup{"External"}
	artifacts, _, err := generateArtifacts(configuration, nil, projection{})
	require.NoError(t, err)
	manifest := string(artifacts[configuration.Layout.ManifestPath])
	require.NotContains(t, manifest, `"inputs": null`)
	require.NotContains(t, manifest, `"sourceRules": null`)
	require.Contains(t, artifacts, configuration.Layout.CorePath)
}

func TestPublishAndCheckArtifactsDetectDriftAndRemoveOnlyManagedStaleFiles(t *testing.T) {
	t.Parallel()
	outputRoot := t.TempDir()
	projection, err := buildProjection(basicDescriptorSet(t), fixtureTestConfiguration.Classify)
	require.NoError(t, err)
	artifacts, _, err := generateArtifacts(fixtureTestConfiguration, nil, projection)
	require.NoError(t, err)
	require.NoError(t, publishArtifacts(outputRoot, artifacts, fixtureTestConfiguration.Layout))
	require.NoError(t, checkArtifacts(outputRoot, artifacts, fixtureTestConfiguration.Layout, "check"))

	typesPath := filepath.Join(outputRoot, filepath.FromSlash(fixtureTestConfiguration.Layout.TypesPath))
	require.NoError(t, os.WriteFile(typesPath, []byte("stale"), 0o600))
	require.ErrorContains(t, checkArtifacts(outputRoot, artifacts, fixtureTestConfiguration.Layout, "check"), "Types.lean (stale)")
	require.NoError(t, publishArtifacts(outputRoot, artifacts, fixtureTestConfiguration.Layout))
	require.NoError(t, checkArtifacts(outputRoot, artifacts, fixtureTestConfiguration.Layout, "check"))
	require.Error(t, validateManagedPath(fixtureTestConfiguration.Layout, "../authored.lean"))
	require.Error(t, validateManagedPath(fixtureTestConfiguration.Layout, "Other/Generated/stale.lean"))
}

func TestCheckArtifactsDetectsFilesRemovedFromTheGenerator(t *testing.T) {
	t.Parallel()
	outputRoot := t.TempDir()
	projection, err := buildProjection(basicDescriptorSet(t), fixtureTestConfiguration.Classify)
	require.NoError(t, err)
	artifacts, _, err := generateArtifacts(fixtureTestConfiguration, nil, projection)
	require.NoError(t, err)
	require.NoError(t, publishArtifacts(outputRoot, artifacts, fixtureTestConfiguration.Layout))
	withoutExternalGRPC := make(map[string][]byte, len(artifacts)-1)
	for path, encoded := range artifacts {
		if path != testGRPCPath(fixtureTestConfiguration, sourceExternal) {
			withoutExternalGRPC[path] = encoded
		}
	}
	require.ErrorContains(t, checkArtifacts(outputRoot, withoutExternalGRPC, fixtureTestConfiguration.Layout, "check"), "GRPC/External.lean (unexpected)")
}

func TestPublishValidatesEveryManagedPathBeforeMutation(t *testing.T) {
	outputRoot := t.TempDir()
	projection, err := buildProjection(basicDescriptorSet(t), fixtureTestConfiguration.Classify)
	require.NoError(t, err)
	artifacts, manifest, err := generateArtifacts(fixtureTestConfiguration, nil, projection)
	require.NoError(t, err)
	require.NoError(t, publishArtifacts(outputRoot, artifacts, fixtureTestConfiguration.Layout))

	typesPath := filepath.Join(outputRoot, filepath.FromSlash(fixtureTestConfiguration.Layout.TypesPath))
	originalTypes, err := os.ReadFile(typesPath)
	require.NoError(t, err)
	safeStale := fixtureTestConfiguration.Layout.GeneratedPath + "/Stale.lean"
	safeStalePath := filepath.Join(outputRoot, filepath.FromSlash(safeStale))
	require.NoError(t, os.WriteFile(safeStalePath, []byte("stale"), 0o600))
	previous := manifest
	previous.GeneratedFiles = append(previous.GeneratedFiles,
		artifactDigest{Path: safeStale},
		artifactDigest{Path: "../authored.lean"},
	)
	encodedPrevious, err := canonicalIndentedJSON(previous)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(
		filepath.Join(outputRoot, filepath.FromSlash(fixtureTestConfiguration.Layout.ManifestPath)),
		encodedPrevious,
		0o600,
	))

	changed := make(map[string][]byte, len(artifacts))
	for path, encoded := range artifacts {
		changed[path] = slices.Clone(encoded)
	}
	changed[fixtureTestConfiguration.Layout.TypesPath] = []byte("mutated")
	require.ErrorContains(t,
		publishArtifacts(outputRoot, changed, fixtureTestConfiguration.Layout),
		"refuse to remove stale artifact",
	)
	currentTypes, err := os.ReadFile(typesPath)
	require.NoError(t, err)
	require.Equal(t, originalTypes, currentTypes)
	currentStale, err := os.ReadFile(safeStalePath)
	require.NoError(t, err)
	require.Equal(t, []byte("stale"), currentStale)
}

func TestInspectLeavesOutputTreeUntouched(t *testing.T) {
	outputRoot := filepath.Join(t.TempDir(), "untouched")
	var stdout bytes.Buffer
	err := Run(context.Background(), []string{
		"inspect",
		"--descriptor", "fixture=testdata/basic/input.pb",
		"--source", "Public=public/",
		"--default-source", "External",
		"--lean-root", "Fixture",
		"--output-root", outputRoot,
	}, &stdout)
	require.NoError(t, err)
	_, err = os.Stat(outputRoot)
	require.ErrorIs(t, err, os.ErrNotExist)
	var manifest generationManifest
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &manifest))
	require.Equal(t, "Fixture", manifest.LeanRoot)
}

func basicDescriptorSet(t *testing.T) *descriptorpb.FileDescriptorSet {
	t.Helper()
	encoded, err := os.ReadFile(filepath.Join("testdata", "basic", "input.pb"))
	require.NoError(t, err)
	set := &descriptorpb.FileDescriptorSet{}
	require.NoError(t, proto.Unmarshal(encoded, set))
	return set
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

func findSchemaMessage(t *testing.T, projection schemaProjection, fullName string) schemaMessage {
	t.Helper()
	for _, message := range projection.Messages {
		if message.FullName == fullName {
			return message
		}
	}
	require.FailNow(t, "schema message not found", fullName)
	return schemaMessage{}
}

func findSchemaField(t *testing.T, message schemaMessage, name string) schemaField {
	t.Helper()
	for _, field := range message.Fields {
		if field.Name == name {
			return field
		}
	}
	require.FailNow(t, "schema field not found", name)
	return schemaField{}
}

func findJSONByFullName(t *testing.T, values []any, fullName string) map[string]any {
	t.Helper()
	for _, value := range values {
		item := value.(map[string]any)
		if item["fullName"] == fullName {
			return item
		}
	}
	require.FailNow(t, "JSON item not found", fullName)
	return nil
}
