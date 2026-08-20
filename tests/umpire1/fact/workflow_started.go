package fact

import (
	"go.temporal.io/server/common/testing/umpire"
)

// WorkflowStarted represents a workflow being started.
type WorkflowStarted struct {
	Request    *v1.StartWorkflowExecutionRequest
	EntityPath *umpire.EntityPath
}

func (e *WorkflowStarted) Name() string {
	return "WorkflowStarted"
}

func (e *WorkflowStarted) TargetEntity() *umpire.EntityPath {
	return e.EntityPath
}

func (e *WorkflowStarted) ImportRequest(request any) bool {
	req, ok := request.(*v1.StartWorkflowExecutionRequest)
	if !ok || req == nil || req.GetStartRequest().GetWorkflowId() == "" {
		return false
	}
	e.Request = req
	wfID := umpire.NewEntityID(WorkflowType, req.GetStartRequest().GetWorkflowId())
	e.EntityPath = nsPath(req.GetNamespaceId(), wfID)
	return true
}
