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
	commonpb "go.temporal.io/api/common/v1"
	enumspb "go.temporal.io/api/enums/v1"
	updatepb "go.temporal.io/api/update/v1"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/server/common/payloads"
	. "go.temporal.io/server/common/proptest1"
)

type WorkflowUpdate Model[WorkflowUpdate]

var (
	// ==== ModelType

	workflowUpdateModel = NewModelType[WorkflowUpdate]()

	// ==== States

	workflowUpdatePreparing = NewState("Preparing")
	workflowUpdateStarting  = NewState("Starting")
	workflowUpdateRunning   = NewState("Running")
	workflowUpdateAccepted  = NewState("Accepted")
	workflowUpdateRejected  = NewState("Rejected")
	workflowUpdateCompleted = NewState("Completed")
	workflowUpdateCancelled = NewState("Cancelled") // ie never started

	// ==== Events

	StartingWorkflowUpdate = NewSignalHandler("Starting",
		[]*State{workflowUpdatePreparing},
		func(upd WorkflowUpdate, resp *workflowservice.UpdateWorkflowExecutionResponse) {
		})
	//CancelledWorkflowUpdate = NewSignalHandler("Cancel", func(upd *WorkflowUpdate, err error) {
	//	//upd.LogError("cancelled: %v\n", err)
	//})
	StartedWorkflowUpdate = NewSignalHandler("Started",
		[]*State{workflowUpdateStarting},
		func(upd WorkflowUpdate, resp *workflowservice.UpdateWorkflowExecutionResponse) {
			switch resp.Stage {
			case enumspb.UPDATE_WORKFLOW_EXECUTION_LIFECYCLE_STAGE_ADMITTED:
				// nothing to do
			case enumspb.UPDATE_WORKFLOW_EXECUTION_LIFECYCLE_STAGE_ACCEPTED:
				//AcceptedWorkflowUpdate.Fire(upd, nil)
			case enumspb.UPDATE_WORKFLOW_EXECUTION_LIFECYCLE_STAGE_COMPLETED:
				//CompletedWorkflowUpdate.Fire(upd, resp)
			default:
				panic("unexpected stage")
			}
		})
	AcceptedWorkflowUpdate = NewSignalHandler("Accept",
		[]*State{}, // stateWorkflowRunning
		func(upd WorkflowUpdate, resp *workflowservice.UpdateWorkflowExecutionResponse) {
			// nothing to do
		})
	// TODO: allow multiple *signal* or *query* handlers for the same event
	//RejectedWorkflowUpdate = NewSignalHandler("Reject",
	//	[]*State{stateWorkflowRunning},
	//	func(upd WorkflowUpdate, resp *workflowservice.UpdateWorkflowExecutionResponse) {
	//		//resp.outcome = resp.Outcome
	//		//resp.NotZero(resp.outcome.GetFailure())
	//		// TODO: verify that last WFT is still `WorkflowTaskCompleted`
	//	})
	CompletedWorkflowUpdate = NewSignalHandler("Complete",
		[]*State{workflowUpdateAccepted},
		func(upd WorkflowUpdate, resp *workflowservice.UpdateWorkflowExecutionResponse) {
			// nothing to do
		})

	// ==== Actions

	StartWorkflowUpdate = NewInitAction("StartWorkflowUpdate",
		[]ModelVar{ // TODO: do those need to be specified at all?
			//ToInput[WorkflowRun](),
			UpdateID,
			UpdatePayload,
			UpdateHandler,
			WaitStage,
		},
		func(u Readonly[WorkflowUpdate]) ID {
			client := Get[Client](u)
			//id := WorkflowID.Get(u) + "/" + UpdateID.Get(u)
			client.Update(
				&workflowservice.UpdateWorkflowExecutionRequest{
					//Namespace: NamespaceName.Get(u),
					WorkflowExecution: &commonpb.WorkflowExecution{
						//WorkflowId: WorkflowID.Get(u),
						//RunId:      WorkflowRunID.Get(u), TODO
					},
					WaitPolicy: &updatepb.WaitPolicy{LifecycleStage: WaitStage.Get(u)},
					Request: &updatepb.Request{
						Meta: &updatepb.Meta{UpdateId: UpdateID.Get(u)},
						Input: &updatepb.Input{
							Name: UpdateHandler.Get(u),
							Args: payloads.EncodeString("args-value-of-" + UpdateID.Get(u)),
						},
					},
				})
			panic("TODO")
		})

	// ==== Vars

	// TODO?
	//type UpdateID      = ModelVar[string]
	//type UpdatePayload = ModelVar[string]
	//type UpdateHandler = ModelVar[string]
	//type WaitStage     = ModelVar[enumspb.UpdateWorkflowExecutionLifecycleStage]

	UpdateID      = NewVar("UpdateID", Default(GenId("id")))
	UpdatePayload = NewVar("UpdatePayload", Default(GenId("pl")))
	UpdateHandler = NewVar("UpdateHandler", Default(GenId("hdl")))
	WaitStage     = NewVar("WaitStage", Default(GenPick(WaitStageSet)))

	// ==== Sets

	WaitStageSet = []enumspb.UpdateWorkflowExecutionLifecycleStage{
		enumspb.UPDATE_WORKFLOW_EXECUTION_LIFECYCLE_STAGE_ACCEPTED,
		enumspb.UPDATE_WORKFLOW_EXECUTION_LIFECYCLE_STAGE_COMPLETED,
	}
)

//func NewWorkflowExecutionUpdate() *WorkflowUpdate {
//	res := &WorkflowUpdate{}
//	res.ModelDef = NewModelType[WorkflowUpdate](
//		res,
//		workflowUpdatePreparing,
//		[]Transition{
//			NewTransition(workflowUpdatePreparing, StartingWorkflowUpdate, workflowUpdateStarting),
//			//NewTransition(workflowUpdateStarting, CancelledWorkflowUpdate, workflowUpdateCancelled),
//			NewTransition(workflowUpdateStarting, StartedWorkflowUpdate, workflowUpdateRunning),
//			NewTransition(workflowUpdateRunning, AcceptedWorkflowUpdate, workflowUpdateAccepted),
//			NewTransition(workflowUpdateRunning, RejectedWorkflowUpdate, workflowUpdateRejected),
//			NewTransition(workflowUpdateRunning, CompletedWorkflowUpdate, workflowUpdateCompleted),
//			NewTransition(workflowUpdateAccepted, CompletedWorkflowUpdate, workflowUpdateCompleted),
//		})
//	return res
//}

//func (u *WorkflowUpdate) ID() string {
//	return WorkflowRunID.Get(u) + "/" + UpdateID.Get(u)
//}

//func (u *WorkflowUpdate) AwaitCompletion() {
//	u.Run().WaitForState(
//		u,
//		workflowUpdateCancelled,
//		workflowUpdateRejected,
//		workflowUpdateCompleted)
//}
