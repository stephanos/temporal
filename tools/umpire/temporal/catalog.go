package temporal

import (
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/server/tools/umpire"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
)

// NewWorkflowServiceCatalog freezes the exact public WorkflowService descriptor closure.
func NewWorkflowServiceCatalog() (*umpire.Catalog, error) {
	return umpire.NewCatalog(descriptorClosure(workflowservice.File_temporal_api_workflowservice_v1_service_proto))
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
