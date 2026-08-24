package api

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"
)

func TestProjectionAndRenderingPreserveStructuralTypesAndStreamingRPCs(t *testing.T) {
	t.Parallel()
	document, err := buildProjection(basicDescriptorSet(t))
	require.NoError(t, err)
	require.Len(t, document.Services, 1)
	require.False(t, document.Services[0].Methods[0].ClientStreaming)
	require.False(t, document.Services[0].Methods[0].ServerStreaming)
	require.True(t, document.Services[0].Methods[3].ClientStreaming)
	require.True(t, document.Services[0].Methods[3].ServerStreaming)

	model := findMessage(t, document, "fixture.messaging.public.v1.Message")
	require.Len(t, model.Oneofs, 1)
	require.True(t, findField(t, model, "attributes").Map)
	require.Equal(t, "choice", findField(t, model, "text").Oneof)
	require.Empty(t, findField(t, model, "note").Oneof)
	require.True(t, findField(t, model, "note").Presence)
	legacy := findMessage(t, document, "fixture.protobuf.compat.v1.LegacyOptions")
	require.True(t, legacy.Fields[0].Required)
	require.True(t, legacy.Fields[1].HasDefault)
	require.Equal(t, "7", legacy.Fields[1].Default)
	require.True(t, legacy.Fields[2].Packed)

	plan, err := buildLeanPlan(document, fixtureTestConfiguration)
	require.NoError(t, err)
	require.True(t, plan.fields["fixture.messaging.shared.v1.Left.right"].Recursive)
	require.Len(t, plan.Services, 1)

	artifacts, err := generateArtifacts(fixtureTestConfiguration, document)
	require.NoError(t, err)
	require.Equal(t, []string{
		"Fixture/API.lean",
		"Fixture/API/Proto.lean",
		"Fixture/API/Types.lean",
	}, sortedArtifactPaths(artifacts))

	protoModule := string(artifacts[fixtureTestConfiguration.Layout.ProtoPath])
	require.Contains(t, protoModule, "namespace Fixture.API.Proto")
	require.Contains(t, protoModule, "structure Bytes where")
	require.Contains(t, protoModule, "structure MessageRef where")
	require.Contains(t, protoModule, "structure Method (Request Response : Type) where")
	require.NotContains(t, protoModule, "Descriptor")

	types := string(artifacts[fixtureTestConfiguration.Layout.TypesPath])
	require.Contains(t, types, "import Fixture.API.Proto")
	require.Contains(t, types, "namespace Fixture.Messaging.Public.V1")
	require.Contains(t, types, "def stateReady : Message.Nested.State")
	require.Contains(t, types, "inductive Message.Choice")
	require.Contains(t, types, "right : Option Fixture.API.Proto.MessageRef")
	require.Contains(t, types, "note : Option String")
	require.False(t, strings.HasSuffix(types, "\n\n"))

	apiModule := string(artifacts[fixtureTestConfiguration.Layout.APIPath])
	require.Contains(t, apiModule, "import Fixture.API.Proto")
	require.Contains(t, apiModule, "import Fixture.API.Types")
	require.Contains(t, apiModule, "namespace Fixture.Messaging.Internal.V1.MessagingService")
	require.Contains(t, apiModule, "def chat : Fixture.API.Proto.Method Fixture.Messaging.Public.V1.Message Fixture.Messaging.Public.V1.Reply")
	require.Contains(t, apiModule, "clientStreaming := true, serverStreaming := true, deprecated := true")
}

func TestDescriptorMergePlanningAndRenderingAreDeterministic(t *testing.T) {
	t.Parallel()
	set := basicDescriptorSet(t)
	first, err := proto.Marshal(set)
	require.NoError(t, err)
	reversed := proto.Clone(set).(*descriptorpb.FileDescriptorSet)
	slices.Reverse(reversed.File)
	second, err := proto.Marshal(reversed)
	require.NoError(t, err)
	mergedFirst, err := mergeDescriptorInputs([]descriptorInput{mustDescriptorInput(t, "first.pb", first)})
	require.NoError(t, err)
	mergedSecond, err := mergeDescriptorInputs([]descriptorInput{mustDescriptorInput(t, "second.pb", second)})
	require.NoError(t, err)
	firstProjection, err := buildProjection(mergedFirst)
	require.NoError(t, err)
	secondProjection, err := buildProjection(mergedSecond)
	require.NoError(t, err)
	require.Equal(t, firstProjection, secondProjection)
	firstArtifacts, err := generateArtifacts(fixtureTestConfiguration, firstProjection)
	require.NoError(t, err)
	secondArtifacts, err := generateArtifacts(fixtureTestConfiguration, secondProjection)
	require.NoError(t, err)
	require.Equal(t, firstArtifacts, secondArtifacts)
}

func TestPublishArtifactsResetsOwnedOutputsAndPreservesSiblings(t *testing.T) {
	t.Parallel()
	outputRoot := t.TempDir()
	layout := fixtureTestConfiguration.Layout
	artifacts := fixtureArtifacts()
	authoredPath := filepath.Join(outputRoot, "Fixture", "Authored.lean")
	require.NoError(t, os.MkdirAll(filepath.Dir(authoredPath), 0o755))
	require.NoError(t, os.WriteFile(authoredPath, []byte("authored"), 0o600))
	require.NoError(t, os.MkdirAll(filepath.Join(outputRoot, filepath.FromSlash(layout.APIPath), "nested"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(outputRoot, filepath.FromSlash(layout.APIPath), "nested", "stale"), []byte("stale"), 0o600))
	require.NoError(t, os.MkdirAll(filepath.Join(outputRoot, filepath.FromSlash(layout.APIDirectory), "unexpected"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(outputRoot, filepath.FromSlash(layout.APIDirectory), "unexpected", "stale"), []byte("stale"), 0o600))

	require.NoError(t, publishArtifacts(outputRoot, layout, artifacts))
	require.Equal(t, map[string][]byte{
		"Proto.lean": artifacts[layout.ProtoPath],
		"Types.lean": artifacts[layout.TypesPath],
	}, readTree(t, filepath.Join(outputRoot, filepath.FromSlash(layout.APIDirectory))))
	api, err := os.ReadFile(filepath.Join(outputRoot, filepath.FromSlash(layout.APIPath)))
	require.NoError(t, err)
	require.Equal(t, artifacts[layout.APIPath], api)
	authored, err := os.ReadFile(authoredPath)
	require.NoError(t, err)
	require.Equal(t, []byte("authored"), authored)
	require.NoError(t, publishArtifacts(outputRoot, layout, artifacts))
}

func TestPublishArtifactsValidatesCompleteMapBeforeMutation(t *testing.T) {
	t.Parallel()
	outputRoot := t.TempDir()
	layout := fixtureTestConfiguration.Layout
	apiPath := filepath.Join(outputRoot, filepath.FromSlash(layout.APIPath))
	require.NoError(t, os.MkdirAll(filepath.Dir(apiPath), 0o755))
	require.NoError(t, os.WriteFile(apiPath, []byte("stale"), 0o600))
	invalid := fixtureArtifacts()
	invalid[layout.APIDirectory+"/Unexpected.lean"] = []byte("unexpected")

	err := publishArtifacts(outputRoot, layout, invalid)
	require.ErrorContains(t, err, "exactly the three managed artifacts")
	current, readErr := os.ReadFile(apiPath)
	require.NoError(t, readErr)
	require.Equal(t, []byte("stale"), current)
}

func TestPublishArtifactsRejectsInconsistentLayoutBeforeMutation(t *testing.T) {
	t.Parallel()
	outputRoot := t.TempDir()
	layout := fixtureTestConfiguration.Layout
	apiPath := filepath.Join(outputRoot, filepath.FromSlash(layout.APIPath))
	require.NoError(t, os.MkdirAll(filepath.Dir(apiPath), 0o755))
	require.NoError(t, os.WriteFile(apiPath, []byte("stale"), 0o600))
	artifacts := fixtureArtifacts()
	encodedProto := artifacts[layout.ProtoPath]
	delete(artifacts, layout.ProtoPath)
	layout.ProtoPath = "Other/API/Proto.lean"
	artifacts[layout.ProtoPath] = encodedProto

	err := publishArtifacts(outputRoot, layout, artifacts)
	require.ErrorContains(t, err, "output layout is inconsistent")
	current, readErr := os.ReadFile(apiPath)
	require.NoError(t, readErr)
	require.Equal(t, []byte("stale"), current)
}

func TestPublishArtifactsRejectsRootSymlinkEscapingOutputBeforeMutation(t *testing.T) {
	outputRoot := t.TempDir()
	external := t.TempDir()
	stalePath := filepath.Join(external, "API.lean")
	require.NoError(t, os.WriteFile(stalePath, []byte("external"), 0o600))
	if err := os.Symlink(external, filepath.Join(outputRoot, "Fixture")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}

	err := publishArtifacts(outputRoot, fixtureTestConfiguration.Layout, fixtureArtifacts())
	require.ErrorContains(t, err, "outside output root")
	current, readErr := os.ReadFile(stalePath)
	require.NoError(t, readErr)
	require.Equal(t, []byte("external"), current)
}

func TestPublishArtifactsReplacesOwnedLeafSymlinksWithoutFollowingThem(t *testing.T) {
	outputRoot := t.TempDir()
	external := t.TempDir()
	layout := fixtureTestConfiguration.Layout
	require.NoError(t, os.MkdirAll(filepath.Join(outputRoot, "Fixture"), 0o755))
	externalFile := filepath.Join(external, "outside.lean")
	require.NoError(t, os.WriteFile(externalFile, []byte("outside"), 0o600))
	if err := os.Symlink(externalFile, filepath.Join(outputRoot, filepath.FromSlash(layout.APIPath))); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	externalDirectory := filepath.Join(external, "api")
	require.NoError(t, os.MkdirAll(externalDirectory, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(externalDirectory, "outside"), []byte("outside"), 0o600))
	require.NoError(t, os.Symlink(externalDirectory, filepath.Join(outputRoot, filepath.FromSlash(layout.APIDirectory))))

	require.NoError(t, publishArtifacts(outputRoot, layout, fixtureArtifacts()))
	externalContent, err := os.ReadFile(externalFile)
	require.NoError(t, err)
	require.Equal(t, []byte("outside"), externalContent)
	externalTree := readTree(t, externalDirectory)
	require.Equal(t, map[string][]byte{"outside": []byte("outside")}, externalTree)
}

func TestPublishArtifactsUsesDependencyOrderAndNamesFailingPath(t *testing.T) {
	outputRoot := t.TempDir()
	layout := fixtureTestConfiguration.Layout
	var published []string
	err := publishArtifactsWith(outputRoot, layout, fixtureArtifacts(), func(path string, _ []byte) error {
		published = append(published, filepath.ToSlash(path))
		if strings.HasSuffix(filepath.ToSlash(path), layout.TypesPath) {
			return errors.New("injected failure")
		}
		return nil
	})
	require.ErrorContains(t, err, `publish generated artifact "Fixture/API/Types.lean"`)
	require.Len(t, published, 2)
	require.True(t, strings.HasSuffix(published[0], layout.ProtoPath))
	require.True(t, strings.HasSuffix(published[1], layout.TypesPath))
}

func TestRunRejectsMalformedDescriptorBeforeCreatingOutput(t *testing.T) {
	directory := t.TempDir()
	descriptorPath := filepath.Join(directory, "malformed.pb")
	outputRoot := filepath.Join(directory, "output")
	require.NoError(t, os.WriteFile(descriptorPath, []byte("malformed"), 0o600))
	err := Run([]string{
		"--descriptor", descriptorPath,
		"--lean-root", "Fixture",
		"--output-root", outputRoot,
	})
	require.ErrorContains(t, err, "decode descriptor")
	_, statErr := os.Stat(outputRoot)
	require.ErrorIs(t, statErr, os.ErrNotExist)
}

func fixtureArtifacts() map[string][]byte {
	layout := fixtureTestConfiguration.Layout
	return map[string][]byte{
		layout.ProtoPath: []byte("proto"),
		layout.TypesPath: []byte("types"),
		layout.APIPath:   []byte("api"),
	}
}

func basicDescriptorSet(t *testing.T) *descriptorpb.FileDescriptorSet {
	t.Helper()
	encoded, err := os.ReadFile(filepath.Join("testdata", "basic", "input.pb"))
	require.NoError(t, err)
	set := &descriptorpb.FileDescriptorSet{}
	require.NoError(t, proto.Unmarshal(encoded, set))
	return set
}

func findMessage(t *testing.T, document projection, fullName string) messageProjection {
	t.Helper()
	for _, message := range document.Messages {
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
