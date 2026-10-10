package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/pluginpb"
)

const helpersBackendEnvironment = "TEMPORAL_PROTO_HELPERS_BACKEND"

// The pinned helper plugin emits descriptor-independent serialization methods. Its only missing
// optional-field support is the protoc protocol flag, so preserve its output and advertise that
// capability at this boundary rather than changing the pinned generator or dropping presence.
func helpersOptionalResponse(encoded []byte) ([]byte, error) {
	var response pluginpb.CodeGeneratorResponse
	if err := proto.Unmarshal(encoded, &response); err != nil {
		return nil, fmt.Errorf("helper generator response: %w", err)
	}
	response.SupportedFeatures = proto.Uint64(response.GetSupportedFeatures() | uint64(pluginpb.CodeGeneratorResponse_FEATURE_PROTO3_OPTIONAL))
	return proto.Marshal(&response)
}

func runHelpersBackend(backend string) error {
	request, err := io.ReadAll(os.Stdin)
	if err != nil {
		return err
	}
	command := exec.Command(backend)
	command.Stdin = bytes.NewReader(request)
	command.Stderr = os.Stderr
	encoded, err := command.Output()
	if err != nil {
		return fmt.Errorf("helper generator: %w", err)
	}
	response, err := helpersOptionalResponse(encoded)
	if err != nil {
		return err
	}
	_, err = os.Stdout.Write(response)
	return err
}
