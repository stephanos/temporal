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

//type WorkflowRun Model[WorkflowRun]
//
//var (
//	// ==== ModelType
//
//	workflowRunModel = NewModelType[WorkflowRun]()
//
//	// ==== States
//
//	stateWorkflowPreparing   = NewState("Preparing")
//	stateWorkflowStarting    = NewState("Starting")
//	stateWorkflowRunning     = NewState("Running")
//	stateWorkflowCancelled   = NewState("Cancelled")
//	stateWorkflowTerminating = NewState("Terminating")
//	stateWorkflowTerminated  = NewState("Terminated")
//
//	// ==== Events
//
//	StartingWorkflow = NewInitHandler("Starting",
//		func(w WorkflowRun, req *workflowservice.StartWorkflowExecutionRequest) ID {
//			fmt.Println("StartingWorkflow")
//			//tq := taskQueue.Get(m)
//			//ns := ns.Get(tq)
//			//cluster := cluster.Get(ns)
//			//resp, err := workflowServiceClient.Get(cluster).StartWorkflowExecution(NewContext(), startWorkflowRequest(m))
//			//
//			//var alreadyStartedErr *serviceerror.WorkflowExecutionAlreadyStarted
//			//switch {
//			//case errors.As(err, &alreadyStartedErr):
//			//	if WorkflowIdConflictPolicy.Get(m) != enumspb.WORKFLOW_ID_CONFLICT_POLICY_FAIL {
//			//		CancelWorkflow.Update(m, err)
//			//	}
//			//case err != nil:
//			//	CancelWorkflow.Update(m, err)
//			//default:
//			//	WorkflowStarted.Update(m, resp)
//			//}
//			return req.WorkflowId
//		})
//	WorkflowStarted = NewSignalHandler("Started",
//		[]*State{stateWorkflowStarting},
//		func(w WorkflowRun, resp *workflowservice.StartWorkflowExecutionResponse) {
//			require.NotZero(w, resp.RunId)
//			require.True(w, resp.Started)
//
//			WorkflowRunID.Set(w, resp.RunId)
//		})
//	TerminatingWorkflow = NewSignalHandler("Terminating",
//		[]*State{stateWorkflowRunning},
//		func(w WorkflowRun, _ *workflowservice.TerminateWorkflowExecutionRequest) {
//
//		})
//	TerminatedWorkflow = NewSignalHandler("Terminated",
//		[]*State{stateWorkflowTerminating},
//		func(w WorkflowRun, _ *workflowservice.TerminateWorkflowExecutionResponse) {
//
//		})
//	//CancelWorkflow = NewSignalHandler("Cancel",
//	//	func(w WorkflowRun, err error) {
//	//		w.LogError("cancelled: %v\n", err)
//	//	})
//)
//
//// TODO: `Checks` that can be performed at any time
//
//// ==== Actions
//
//var (
//	StartWorkflow = NewInitAction("StartWorkflow",
//		[]ModelVar{
//			ToInput[TaskQueue](),
//			WorkflowID,
//			RequestID,
//			WorkflowType,
//			WorkflowIdConflictPolicy,
//			WorkflowIdReusePolicy,
//			WorkflowRunID,
//		},
//		func(w Readonly[WorkflowRun]) ID {
//			client := Get[Client](w)
//			id := WorkflowID.Get(w) + "/" + UpdateID.Get(w)
//			client.Update(
//				&workflowservice.StartWorkflowExecutionRequest{
//					//Namespace:  NamespaceName.Get(w),
//					WorkflowId: WorkflowID.Get(w),
//					WorkflowType: &commonpb.WorkflowType{
//						Name: WorkflowType.Get(w),
//					},
//					TaskQueue: &taskqueuepb.TaskQueue{
//						Name: TaskQueueName.Get(w),
//					},
//					RequestId:                RequestID.Get(w),
//					WorkflowIdReusePolicy:    WorkflowIdReusePolicy.Get(w),
//					WorkflowIdConflictPolicy: WorkflowIdConflictPolicy.Get(w),
//				})
//			return id
//		})
//
//	//TerminateWorkflow = NewAction("TerminateWorkflow",
//	//	[]ModelVar{},
//	//	func(w Readonly[WorkflowRun]) (*workflowservice.TerminateWorkflowExecutionRequest, error) {
//	//		return &workflowservice.TerminateWorkflowExecutionRequest{
//	//			Namespace: NamespaceName.Get(w),
//	//			WorkflowExecution: &commonpb.WorkflowExecution{
//	//				WorkflowId: WorkflowID.Get(w),
//	//				RunId:      WorkflowRunID.Get(w),
//	//			},
//	//			// ...
//	//		}, nil
//	//	})
//)
//
//// ==== Vars
//
//var (
//	WorkflowID               = NewVar("WorkflowID", Default(GenId("wfid")))
//	WorkflowType             = NewVar("WorkflowType", Default(GenId("type")))
//	RequestID                = NewVar("RequestID", Default(GenId("req")))
//	WorkflowIdConflictPolicy = NewVar("WorkflowIdConflictPolicy", Default(GenPick(WorkflowConflictPolicySet)))
//	WorkflowIdReusePolicy    = NewVar("WorkflowIdReusePolicy", Default(GenPick(WorkflowIdReusePolicySet)))
//	WorkflowRunTimeout       = NewVar[time.Duration]("WorkflowRunTimeout")
//)
//
//// ==== Outputs
//
//var (
//	WorkflowRunID = NewOutput[string]("WorkflowRunID")
//)
//
//// ==== Sets
//
//var (
//	WorkflowConflictPolicySet = []enumspb.WorkflowIdConflictPolicy{
//		enumspb.WORKFLOW_ID_CONFLICT_POLICY_UNSPECIFIED,
//		enumspb.WORKFLOW_ID_CONFLICT_POLICY_TERMINATE_EXISTING,
//		enumspb.WORKFLOW_ID_CONFLICT_POLICY_USE_EXISTING,
//		enumspb.WORKFLOW_ID_CONFLICT_POLICY_FAIL,
//	}
//	WorkflowIdReusePolicySet = []enumspb.WorkflowIdReusePolicy{
//		enumspb.WORKFLOW_ID_REUSE_POLICY_UNSPECIFIED,
//		enumspb.WORKFLOW_ID_REUSE_POLICY_ALLOW_DUPLICATE,
//		enumspb.WORKFLOW_ID_REUSE_POLICY_ALLOW_DUPLICATE_FAILED_ONLY,
//		enumspb.WORKFLOW_ID_REUSE_POLICY_REJECT_DUPLICATE,
//		enumspb.WORKFLOW_ID_REUSE_POLICY_TERMINATE_IF_RUNNING,
//	}
//)

//func NewWorkflowExecution() WorkflowRun {
//	res := &WorkflowRun{}
//	res.ModelType = NewModelType[WorkflowRun](
//		res,
//		stateWorkflowPreparing,
//		[]Transition{
//			NewTransition(stateWorkflowPreparing, StartingWorkflow, stateWorkflowStarting),
//			NewTransition(stateWorkflowStarting, WorkflowStarted, stateWorkflowRunning),
//			//NewTransition(stateWorkflowStarting, CancelWorkflow, stateWorkflowCancelled),
//			NewTransition(stateWorkflowRunning, TerminatingWorkflow, stateWorkflowTerminating),
//			NewTransition(stateWorkflowTerminating, TerminatedWorkflow, stateWorkflowTerminated),
//		},
//	)
//	return res
//}

//func (w WorkflowRun) ID() string {
//	return WorkflowID.Get(w) + "/" + WorkflowRunID.Get(w)
//}
