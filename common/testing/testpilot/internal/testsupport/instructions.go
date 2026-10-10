package testsupport

import testpilotspb "go.temporal.io/server/api/testpilot/v1"

func LocalReference(reference *testpilotspb.InstructionReference) *testpilotspb.LocalInstructionReference {
	if reference == nil {
		return nil
	}
	return &testpilotspb.LocalInstructionReference{InstructionId: reference.InstructionId}
}

func LocalReferences(references []*testpilotspb.InstructionReference) []*testpilotspb.LocalInstructionReference {
	result := make([]*testpilotspb.LocalInstructionReference, len(references))
	for index, reference := range references {
		result[index] = LocalReference(reference)
	}
	return result
}
