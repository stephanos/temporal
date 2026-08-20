package main

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
)

const testSelectionPath = "../../model/Temporal/API/selection.json"

func TestProjectionIsDeterministicAndClosedOverSelectedDescriptors(t *testing.T) {
	selection, err := loadSelection(testSelectionPath)
	require.NoError(t, err)
	first, err := buildProjection(selection)
	require.NoError(t, err)
	second, err := buildProjection(selection)
	require.NoError(t, err)

	require.Equal(t, first.DescriptorDigest, second.DescriptorDigest)
	require.Equal(t, first.Messages, second.Messages)
	require.Contains(t, first.Roots, "temporal.api.workflowservice.v1.RequestCancelNexusOperationExecutionRequest")
	require.Contains(t, first.Roots, "temporal.api.workflowservice.v1.StartWorkflowExecutionRequest")
	require.Contains(t, first.Roots, "temporal.api.failure.v1.Failure")
	require.NotContains(t, first.Roots, "temporal.api.history.v1.HistoryEvent")
}

func TestProjectionCoversRequiredProtobufShapes(t *testing.T) {
	selection, err := loadSelection(testSelectionPath)
	require.NoError(t, err)
	projection, err := buildProjection(selection)
	require.NoError(t, err)

	require.True(t, projection.Features.Presence)
	require.True(t, projection.Features.Oneof)
	require.True(t, projection.Features.Enum)
	require.True(t, projection.Features.Repeated)
	require.True(t, projection.Features.Map)
	require.True(t, projection.Features.Nested)
	require.True(t, projection.Features.Bytes)
	require.True(t, projection.Features.Duration)
}

func TestGenerateLeanWireTypesPreservesStructure(t *testing.T) {
	selection, err := loadSelection(testSelectionPath)
	require.NoError(t, err)
	projection, err := buildProjection(selection)
	require.NoError(t, err)
	generated, err := generateLean(projection)
	require.NoError(t, err)

	require.Contains(t, string(generated), "structure RequestCancelNexusOperationExecutionRequest")
	require.Contains(t, string(generated), "structure StartWorkflowExecutionRequest")
	require.Contains(t, string(generated), "Option")
	require.Contains(t, string(generated), "List")
	require.Contains(t, string(generated), "BoundedMessage")
	require.Contains(t, string(generated), "def descriptorHash")
	require.Contains(t, string(generated), "def fieldDomains")
}

func TestGenerateArtifactsIsDeterministicAndFullyDispositioned(t *testing.T) {
	selection, err := loadSelection(testSelectionPath)
	require.NoError(t, err)
	projection, err := buildProjection(selection)
	require.NoError(t, err)
	first, err := generateArtifacts(selection, projection)
	require.NoError(t, err)
	second, err := generateArtifacts(selection, projection)
	require.NoError(t, err)
	require.Equal(t, first, second)
	require.False(t, bytes.Contains(first[dispositionsOutput], []byte(`"disposition":""`)))
}

func TestSelectionRejectsUnknownFieldOverride(t *testing.T) {
	selection, err := loadSelection(testSelectionPath)
	require.NoError(t, err)
	selection.Messages[0].Fields["field_that_does_not_exist"] = dispositionInterpreted

	_, err = buildProjection(selection)
	require.ErrorContains(t, err, "field_that_does_not_exist")
}
