package main

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"
)

func TestMergeDescriptorInputsUsesSemanticEqualityAndLocatorsInConflicts(t *testing.T) {
	firstFile := &descriptorpb.FileDescriptorProto{Name: proto.String("example/model.proto"), Package: proto.String("example.v1")}
	firstSet := marshalDescriptorSet(t, firstFile)
	equivalent := proto.Clone(firstFile).(*descriptorpb.FileDescriptorProto)
	merged, err := mergeDescriptorInputs([]descriptorInput{
		mustDescriptorInput(t, "zeta.pb", firstSet),
		mustDescriptorInput(t, "alpha.pb", marshalDescriptorSet(t, equivalent)),
	})
	require.NoError(t, err)
	require.Len(t, merged.File, 1)

	conflicting := proto.Clone(firstFile).(*descriptorpb.FileDescriptorProto)
	conflicting.Package = proto.String("different.v1")
	_, err = mergeDescriptorInputs([]descriptorInput{
		mustDescriptorInput(t, "public.pb", firstSet),
		mustDescriptorInput(t, "internal.pb", marshalDescriptorSet(t, conflicting)),
	})
	require.ErrorContains(t, err, `descriptor file "example/model.proto"`)
	require.ErrorContains(t, err, `descriptor paths "public.pb" and "internal.pb"`)
}

func TestMergeDescriptorInputsSortsFilesAndRejectsEmptyPaths(t *testing.T) {
	encoded := marshalDescriptorSet(t,
		&descriptorpb.FileDescriptorProto{Name: proto.String("z.proto")},
		&descriptorpb.FileDescriptorProto{Name: proto.String("a.proto")},
	)
	merged, err := mergeDescriptorInputs([]descriptorInput{mustDescriptorInput(t, "input.pb", encoded)})
	require.NoError(t, err)
	paths := []string{merged.File[0].GetName(), merged.File[1].GetName()}
	require.True(t, slices.IsSorted(paths))

	_, err = mergeDescriptorInputs([]descriptorInput{mustDescriptorInput(t, "input.pb", marshalDescriptorSet(t,
		&descriptorpb.FileDescriptorProto{},
	))})
	require.ErrorContains(t, err, "file without a path")
}

func TestDescriptorFileInputPreservesSuppliedLocator(t *testing.T) {
	directory := t.TempDir()
	path := filepath.Join(directory, "input.pb")
	require.NoError(t, os.WriteFile(path, marshalDescriptorSet(t,
		&descriptorpb.FileDescriptorProto{Name: proto.String("example.proto")},
	), 0o600))

	input, err := descriptorFileInput(path, "fixtures/input.pb")
	require.NoError(t, err)
	require.Equal(t, "fixtures/input.pb", input.Locator)
	_, err = descriptorFileInput(filepath.Join(directory, "missing.pb"), "missing.pb")
	require.ErrorContains(t, err, `read descriptor "`)
	malformedPath := filepath.Join(directory, "malformed.pb")
	require.NoError(t, os.WriteFile(malformedPath, []byte("not a descriptor set"), 0o600))
	_, err = descriptorFileInput(malformedPath, "malformed.pb")
	require.ErrorContains(t, err, `decode descriptor "malformed.pb"`)
}

func marshalDescriptorSet(t *testing.T, files ...*descriptorpb.FileDescriptorProto) []byte {
	t.Helper()
	encoded, err := proto.MarshalOptions{Deterministic: true}.Marshal(&descriptorpb.FileDescriptorSet{File: files})
	require.NoError(t, err)
	return encoded
}

func mustDescriptorInput(t *testing.T, locator string, encoded []byte) descriptorInput {
	t.Helper()
	input, err := newDescriptorInput(locator, encoded)
	require.NoError(t, err)
	return input
}
