package main

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGenerateNexusAPIIsDeterministicAndDescriptorDerived(t *testing.T) {
	first, err := generateNexusAPI()
	require.NoError(t, err)
	second, err := generateNexusAPI()
	require.NoError(t, err)
	require.True(t, bytes.Equal(first, second))
	require.Contains(t, string(first), "RequestCancelNexusOperationExecutionRequest")
	require.Contains(t, string(first), "namespaceName : String")
	require.Contains(t, string(first), "operationId : String")
	require.Contains(t, string(first), "descriptorHash : String := \"sha256:")
}
