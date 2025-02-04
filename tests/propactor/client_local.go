// The MIT License
//
// Copyright (c) 2020 Temporal Technologies Inc.  All rights reserved.
//
// Copyright (c) 2020 Uber Technologies, Inc.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package propactor

import (
	commonpb "go.temporal.io/api/common/v1"
	enumspb "go.temporal.io/api/enums/v1"
	taskqueuepb "go.temporal.io/api/taskqueue/v1"
	updatepb "go.temporal.io/api/update/v1"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/server/common/payloads"
	. "go.temporal.io/server/common/testing/propactor"
	. "go.temporal.io/server/tests/propmodel"
)

type LocalClient struct {
	Actor
	workflowServiceClient workflowservice.WorkflowServiceClient
}

func NewLocalClient(
	params ...any,
) *LocalClient {
	return &LocalClient{
		//workflowServiceClient: workflowServiceClient,
	}
}

var StartUpdateWithStart = NewHandler("StartUpdateWithStart",
	func(upd *WorkflowUpdate) func(c *LocalClient) {
		return func(c *LocalClient) {
			//wf := workflowRun.Get(upd)
			//WorkflowBeingStarted.Fire(wf, nil)
			//WorkflowUpdateBeingStarted.Fire(upd, nil)
			//
			//go func() {
			//	resp, err := c.workflowServiceClient.ExecuteMultiOperation(
			//		NewContext(),
			//		&workflowservice.ExecuteMultiOperationRequest{
			//			Namespace: NamespaceName.Get(ns.Get(taskQueue.Get(wf))).String(),
			//			Operations: []*workflowservice.ExecuteMultiOperationRequest_Operation{
			//				{
			//					Operation: &workflowservice.ExecuteMultiOperationRequest_Operation_StartWorkflow{
			//						StartWorkflow: startWorkflowRequest(wf.Model),
			//					},
			//				},
			//				{
			//					Operation: &workflowservice.ExecuteMultiOperationRequest_Operation_UpdateWorkflow{
			//						UpdateWorkflow: startUpdateRequest(upd.Model),
			//					},
			//				},
			//			},
			//		})
			//
			//	if err != nil {
			//		var multiErr *serviceerror.MultiOperationExecution
			//		if errors.As(err, &multiErr) {
			//			CancelWorkflow.Fire(wf, multiErr.OperationErrors()[0])
			//			CancelledWorkflowUpdate.Fire(upd, multiErr.OperationErrors()[0])
			//		} else {
			//			CancelWorkflow.Fire(wf, err)
			//			CancelledWorkflowUpdate.Fire(upd, err)
			//		}
			//		return
			//	}
			//
			//	workflowStarted.Fire(wf,
			//		resp.Responses[0].Response.(*workflowservice.ExecuteMultiOperationResponse_Response_StartWorkflow).StartWorkflow)
			//	startedWorkflowUpdate.Fire(upd,
			//		resp.Responses[1].Response.(*workflowservice.ExecuteMultiOperationResponse_Response_UpdateWorkflow).UpdateWorkflow)
			//}()
		}
	})

func startUpdateRequest(upd *WorkflowUpdate) *workflowservice.UpdateWorkflowExecutionRequest {
	return &workflowservice.UpdateWorkflowExecutionRequest{
		Namespace: NamespaceName.Get(upd).String(),
		WorkflowExecution: &commonpb.WorkflowExecution{
			WorkflowId: WorkflowID.Get(upd),
			RunId:      WorkflowRunID.Get(upd),
		},
		WaitPolicy: &updatepb.WaitPolicy{LifecycleStage: WaitStage.Get(upd)},
		Request: &updatepb.Request{
			Meta: &updatepb.Meta{UpdateId: UpdateID.Get(upd)},
			Input: &updatepb.Input{
				Name: UpdateHandler.Get(upd),
				Args: payloads.EncodeString("args-value-of-" + UpdateID.Get(upd)),
			},
		},
	}
}

func startWorkflowRequest(run *WorkflowRun) *workflowservice.StartWorkflowExecutionRequest {
	return &workflowservice.StartWorkflowExecutionRequest{
		Namespace:    NamespaceName.Get(run).String(),
		WorkflowId:   WorkflowID.Get(run),
		WorkflowType: &commonpb.WorkflowType{Name: WorkflowType.Get(run)},
		TaskQueue: &taskqueuepb.TaskQueue{
			Name: TaskQueueName.Get(run),
			Kind: enumspb.TASK_QUEUE_KIND_NORMAL,
		},
		Identity:                 Identity.Get(run),
		WorkflowIdReusePolicy:    WorkflowIdReusePolicy.Get(run),
		WorkflowIdConflictPolicy: WorkflowIdConflictPolicy.Get(run),
	}
}

//func StartWorkflow(
//	params ...any,
//) *WorkflowRun {
//	wf := NewWorkflowExecution(params...)
//	//startWorkflow.Fire(wf, nil)
//	return wf
//}

// TerminateWorkflow
//_, err := workflowServiceClient.Get(cluster.Get(ns.Get(taskQueue.Get(m)))).TerminateWorkflowExecution(
//NewContext(),
//&workflowservice.TerminateWorkflowExecutionRequest{
//Namespace: NamespaceName.Get(ns.Get(taskQueue.Get(m))).String(),
//WorkflowExecution: &commonpb.WorkflowExecution{
//WorkflowId: WorkflowID.Get(m),
//RunId:      WorkflowRunID.Get(m),
//},
//Reason: "TODO",
//})
//m.NoError(err)
