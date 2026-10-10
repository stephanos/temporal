package main

import (
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/pluginpb"
)

func TestHelpersOptionalProtocolPreservesGeneratorResponse(t *testing.T) {
	response := &pluginpb.CodeGeneratorResponse{
		File:              []*pluginpb.CodeGeneratorResponse_File{{Name: proto.String("example.go"), Content: proto.String("package example")}},
		SupportedFeatures: proto.Uint64(uint64(pluginpb.CodeGeneratorResponse_FEATURE_SUPPORTS_EDITIONS)),
	}
	encoded, err := proto.Marshal(response)
	require.NoError(t, err)
	result, err := helpersOptionalResponse(encoded)
	require.NoError(t, err)
	var got pluginpb.CodeGeneratorResponse
	require.NoError(t, proto.Unmarshal(result, &got))
	response.SupportedFeatures = proto.Uint64(response.GetSupportedFeatures() | uint64(pluginpb.CodeGeneratorResponse_FEATURE_PROTO3_OPTIONAL))
	require.True(t, proto.Equal(response, &got))
	_, err = helpersOptionalResponse([]byte{0xff})
	require.Error(t, err)
}
