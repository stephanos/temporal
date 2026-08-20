package fact

import (
	"go.temporal.io/server/common/testing/umpire"
)

// WorkflowTaskPolled represents a workflow task being polled.
type WorkflowTaskPolled struct {
	Request      *v1.PollWorkflowTaskQueueRequest
	EntityPath   *umpire.EntityPath
	TaskReturned bool
}

func (e *WorkflowTaskPolled) Name() string {
	return "WorkflowTaskPolled"
}

func (e *WorkflowTaskPolled) TargetEntity() *umpire.EntityPath {
	return e.EntityPath
}

func (e *WorkflowTaskPolled) ImportRequest(request any) bool {
	req, ok := request.(*v1.PollWorkflowTaskQueueRequest)
	if !ok || req == nil || req.GetPollRequest().GetTaskQueue().GetName() == "" {
		return false
	}
	e.Request = req
	tqID := umpire.NewEntityID(TaskQueueType, req.GetPollRequest().GetTaskQueue().GetName())
	e.EntityPath = nsPath(req.GetNamespaceId(), tqID)
	return true
}
