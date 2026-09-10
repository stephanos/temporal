package temporal

import (
	"go.temporal.io/api/workflowservice/v1"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/testpilot"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
)

// NewWorkflowServiceCatalog freezes the exact public WorkflowService descriptor closure together
// with the Testpilot protocol closure. A Case that declares a CorrelatedEvidence Observation names a
// protocol message rather than a Temporal API one, so the Driver's catalog has to know both.
func NewWorkflowServiceCatalog() (*testpilot.Catalog, error) {
	descriptors := WorkflowServiceDescriptorSet()
	protocol := descriptorClosure(testpilotspb.File_temporal_server_api_testpilot_v1_run_proto)
	seen := make(map[string]struct{}, len(descriptors.File))
	for _, file := range descriptors.File {
		seen[file.GetName()] = struct{}{}
	}
	for _, file := range protocol.File {
		if _, exists := seen[file.GetName()]; !exists {
			descriptors.File = append(descriptors.File, file)
		}
	}
	return testpilot.NewCatalog(descriptors)
}

// WorkflowServiceDescriptorSet returns the exact public WorkflowService descriptor closure.
func WorkflowServiceDescriptorSet() *descriptorpb.FileDescriptorSet {
	return descriptorClosure(workflowservice.File_temporal_api_workflowservice_v1_service_proto)
}

func descriptorClosure(root protoreflect.FileDescriptor) *descriptorpb.FileDescriptorSet {
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
	add(root)
	return result
}
