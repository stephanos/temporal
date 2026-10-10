// Package testsupport holds the descriptor, limit and fake-Driver-side helpers the Testpilot tests
// share. It imports no Testpilot package but contract, so every core package's tests can use it.
package testsupport

import (
	"time"

	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/known/durationpb"
)

// DescriptorClosure is the descriptor set of roots and everything they import, each file once and
// after its imports.
func DescriptorClosure(roots ...protoreflect.FileDescriptor) *descriptorpb.FileDescriptorSet {
	seen := make(map[string]struct{})
	result := &descriptorpb.FileDescriptorSet{}
	var add func(protoreflect.FileDescriptor)
	add = func(file protoreflect.FileDescriptor) {
		if _, exists := seen[file.Path()]; exists {
			return
		}
		seen[file.Path()] = struct{}{}
		imports := file.Imports()
		for index := 0; index < imports.Len(); index++ {
			add(imports.Get(index))
		}
		result.File = append(result.File, protodesc.ToFileDescriptorProto(file))
	}
	for _, root := range roots {
		add(root)
	}
	return result
}

// ProgramLimits is a fresh set of Program limits every field of which admits the test fixtures.
// Callers change the fields their test is about.
func ProgramLimits() *testpilotspb.ProgramLimits {
	return &testpilotspb.ProgramLimits{MaxEntrypoints: 8, MaxNodes: 32, MaxEdges: 64, MaxActivations: 64, MaxAttempts: 32, MaxRunEvents: 256, MaxExpressionDepth: 16, MaxPathFanout: 128, MaxRequestBytes: 4096, MaxResponseBytes: 4096, MaxDuration: durationpb.New(time.Duration(30000) * time.Millisecond), CleanupDuration: durationpb.New(time.Duration(5000) * time.Millisecond), MaxInstructionEmittedEvents: 8, MaxInstructionResponseBytes: 4096}
}
