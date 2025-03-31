// The MIT License
//
// Copyright (c) 2025 Temporal Technologies Inc.  All rights reserved.
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

package older

import (
	"fmt"
	"time"

	"github.com/pborman/uuid"
	enumspb "go.temporal.io/api/enums/v1"
	sdkclient "go.temporal.io/sdk/client"
	. "go.temporal.io/server/common/proptest1"
)

type Client Model[Client]

var (
	// ==== ModelType

	//clientModel = NewModelType[Server]()

	// ==== States

	clientStarting = NewState("Starting")
	clientStarted  = NewState("Started")

	// ==== Vars

	clientStartInputs = []ModelVar{
		//WorkflowType,
		//ServerAddr,
		//nsInput,
	}

	// ==== Actions

	StartLocalClient = NewInitAction[Client]("StartLocalClient",
		clientStartInputs,
		func(c Readonly[Client]) ID {
			//SDKClientClient.Set(c, client)
			// TODO
			return uuid.New()
		})

	StartSDKClient = NewInitAction[Client]("StartSDKClient",
		clientStartInputs,
		func(c Readonly[Client]) ID {
			id := uuid.New()

			sdkClient, err := sdkclient.Dial(sdkclient.Options{
				//HostPort: ServerAddr.Get(c),
				//Namespace: NamespaceName.Get(nsInput.Get(c)),
			})
			if err != nil {
				return id
			}
			c.Update(sdkClient)

			return id
		})

	StartSpyClient = NewInitAction[Client]("StartSpyClient",
		clientStartInputs,
		func(c Readonly[Client]) ID {
			return uuid.New()
		})

	Send = NewAction[Client]("Send",
		[]ModelVar{},
		func(c Readonly[Client]) any {
			// TODO
			return nil
		},
		RequireState(clientStarted),
		WaitForState(10*time.Second))
)

//func (s *SDKClient) Do(action Action[*SDKClient]) {
//	switch action.Template.String() {
//	case StartWorkflow.String():
//		payload := StartWorkflow.Payload(action.ModelType)
//		options := sdkclient.StartWorkflowOptions{
//			ID:                       payload.WorkflowId,
//			TaskQueue:                payload.TaskQueue.Name,
//			WorkflowIDReusePolicy:    payload.WorkflowIdReusePolicy,
//			WorkflowIDConflictPolicy: payload.WorkflowIdConflictPolicy,
//		}
//		_, err := s.sdkClient.ExecuteWorkflow(NewContext(), options, WorkflowType.Get(s))
//		if err != nil {
//			panic(err)
//		}
//	case StartWorkflowUpdate.String():
//		payload := StartWorkflowUpdate.Payload(action.ModelType)
//		options := sdkclient.UpdateWorkflowOptions{
//			UpdateID:   payload.Request.Meta.UpdateId,
//			WorkflowID: payload.WorkflowExecution.WorkflowId,
//			RunID:      payload.WorkflowExecution.RunId,
//			UpdateName: payload.Request.Input.Name,
//			//Args: nil,
//			WaitForStage: convertWaitStage(payload.WaitPolicy.LifecycleStage),
//			//FirstExecutionRunID
//		}
//		_, err := s.sdkClient.UpdateWorkflow(NewContext(), options)
//		if err != nil {
//			panic(err)
//		}
//	default:
//		panic(fmt.Sprintf("unsupported action: %v", action))
//	}
//}

func convertWaitStage(
	stage enumspb.UpdateWorkflowExecutionLifecycleStage,
) sdkclient.WorkflowUpdateStage {
	switch stage {
	case enumspb.UPDATE_WORKFLOW_EXECUTION_LIFECYCLE_STAGE_UNSPECIFIED:
		return sdkclient.WorkflowUpdateStageUnspecified
	case enumspb.UPDATE_WORKFLOW_EXECUTION_LIFECYCLE_STAGE_ADMITTED:
		return sdkclient.WorkflowUpdateStageAdmitted
	case enumspb.UPDATE_WORKFLOW_EXECUTION_LIFECYCLE_STAGE_ACCEPTED:
		return sdkclient.WorkflowUpdateStageAccepted
	case enumspb.UPDATE_WORKFLOW_EXECUTION_LIFECYCLE_STAGE_COMPLETED:
		return sdkclient.WorkflowUpdateStageCompleted
	default:
		panic(fmt.Sprintf("unexpected stage: %v", stage))
	}
}

//NewTransition(startWorkflowRequest(), StartingWorkflow, stateWorkflowStarting),

//func (l *LocalClient) Do(action Action[*LocalClient]) {
//	panic("implement me")
//}

//var startUpdateWithStart = NewSignalHandler(
//	"startUpdateWithStart",
//	func(c *LocalClient, upd *WorkflowUpdate) {
//		//wf := workflowRun.Get(upd)
//		//WorkflowBeingStarted.Update(wf)
//		//WorkflowUpdateBeingStarted.Update(upd, nil)
//		//
//		//go func() {
//		//	resp, err := c.workflowServiceClient.ExecuteMultiOperation(
//		//		NewContext(),
//		//		&workflowservice.ExecuteMultiOperationRequest{
//		//			Namespace: NamespaceName.Get(ns.Get(taskQueue.Get(wf))).String(),
//		//			Operations: []*workflowservice.ExecuteMultiOperationRequest_Operation{
//		//				{
//		//					Operation: &workflowservice.ExecuteMultiOperationRequest_Operation_StartWorkflow{
//		//						StartWorkflow: startWorkflowRequest(wf.ModelType),
//		//					},
//		//				},
//		//				{
//		//					Operation: &workflowservice.ExecuteMultiOperationRequest_Operation_UpdateWorkflow{
//		//						UpdateWorkflow: startUpdateRequest(upd.ModelType),
//		//					},
//		//				},
//		//			},
//		//		})
//		//
//		//	if err != nil {
//		//		var multiErr *serviceerror.MultiOperationExecution
//		//		if errors.As(err, &multiErr) {
//		//			CancelWorkflow.Update(wf, multiErr.OperationErrors()[0])
//		//			CancelledWorkflowUpdate.Update(upd, multiErr.OperationErrors()[0])
//		//		} else {
//		//			CancelWorkflow.Update(wf, err)
//		//			CancelledWorkflowUpdate.Update(upd, err)
//		//		}
//		//		return
//		//	}
//		//
//		//	workflowStarted.Update(wf,
//		//		resp.Responses[0].Response.(*workflowservice.ExecuteMultiOperationResponse_Response_StartWorkflow).StartWorkflow)
//		//	startedWorkflowUpdate.Update(upd,
//		//		resp.Responses[1].Response.(*workflowservice.ExecuteMultiOperationResponse_Response_UpdateWorkflow).UpdateWorkflow)
//		//}()
//	})
