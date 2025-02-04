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

package propmodel

import (
	enumspb "go.temporal.io/api/enums/v1"
	"go.temporal.io/api/workflowservice/v1"
	. "go.temporal.io/server/common/testing/propcheck"
	. "go.temporal.io/server/common/testing/propmodel"
)

// ==== States

var (
	stateWorkflowPreparing   = NewState("Preparing", Init)
	stateWorkflowStarting    = NewState("Starting")
	stateWorkflowRunning     = NewState("Running")
	stateWorkflowCancelled   = NewState("Cancelled")
	stateWorkflowTerminating = NewState("Terminating")
	stateWorkflowTerminated  = NewState("Terminated")
)

// ==== Events

var (
	StartingWorkflow = NewEvent("Starting",
		func(m *Model, _ any) {
			//tq := taskQueue.Get(m)
			//ns := ns.Get(tq)
			//cluster := cluster.Get(ns)
			//resp, err := workflowServiceClient.Get(cluster).StartWorkflowExecution(NewContext(), startWorkflowRequest(m))
			//
			//var alreadyStartedErr *serviceerror.WorkflowExecutionAlreadyStarted
			//switch {
			//case errors.As(err, &alreadyStartedErr):
			//	if WorkflowIdConflictPolicy.Get(m) != enumspb.WORKFLOW_ID_CONFLICT_POLICY_FAIL {
			//		CancelWorkflow.Fire(m, err)
			//	}
			//case err != nil:
			//	CancelWorkflow.Fire(m, err)
			//default:
			//	WorkflowStarted.Fire(m, resp)
			//}
		})
	WorkflowBeingStarted = NewEvent("BeingStarted",
		func(m *Model, _ any) {

		})
	WorkflowStarted = NewEvent("Started",
		func(m *Model, resp *workflowservice.StartWorkflowExecutionResponse) {
			m.NotZero(resp.RunId)
			m.True(resp.Started)

			WorkflowRunID.Set(m, resp.RunId)
		})
	TerminatingWorkflow = NewEvent("Terminating",
		func(m *Model, _ any) {

		})
	TerminatedWorkflow = NewEvent("Terminated",
		func(m *Model, _ any) {

		})
	CancelWorkflow = NewEvent("Cancel",
		func(m *Model, err error) {
			m.Errorf("cancelled: %v\n", err)
		})
)

// TODO: `Checks` that can be performed at any time

// ==== Actions

var (
	// TODO: mark as long-poll
	StartWorkflow     = NewAction("StartingWorkflow", StartingWorkflow)
	TerminateWorkflow = NewAction("TerminateWorkflow", TerminatingWorkflow)
)

// ==== Inputs

var (
	WorkflowID               = NewInput(Default(GenId("wfid")))
	WorkflowType             = NewInput(Default(GenId("type")))
	Identity                 = NewInput(Default(GenId("id")))
	RequestID                = NewInput(Default(GenId("req")))
	WorkflowIdConflictPolicy = NewInput(Default(GenPick(WorkflowConflictPolicySet)))
	WorkflowIdReusePolicy    = NewInput(Default(GenPick(WorkflowIdReusePolicySet)))
)

// ==== Outputs

var (
	WorkflowRunID = NewOutput[string]()
)

// ==== Sets

var (
	WorkflowConflictPolicySet = []enumspb.WorkflowIdConflictPolicy{
		enumspb.WORKFLOW_ID_CONFLICT_POLICY_UNSPECIFIED,
		enumspb.WORKFLOW_ID_CONFLICT_POLICY_TERMINATE_EXISTING,
		enumspb.WORKFLOW_ID_CONFLICT_POLICY_USE_EXISTING,
		enumspb.WORKFLOW_ID_CONFLICT_POLICY_FAIL,
	}
	WorkflowIdReusePolicySet = []enumspb.WorkflowIdReusePolicy{
		enumspb.WORKFLOW_ID_REUSE_POLICY_UNSPECIFIED,
		enumspb.WORKFLOW_ID_REUSE_POLICY_ALLOW_DUPLICATE,
		enumspb.WORKFLOW_ID_REUSE_POLICY_ALLOW_DUPLICATE_FAILED_ONLY,
		enumspb.WORKFLOW_ID_REUSE_POLICY_REJECT_DUPLICATE,
		enumspb.WORKFLOW_ID_REUSE_POLICY_TERMINATE_IF_RUNNING,
	}
)

type WorkflowRun struct {
	*Model
}

func NewWorkflowExecution(
	params ...any,
) (res *WorkflowRun) {
	return &WorkflowRun{
		Model: NewModel(
			"WorkflowExecution",
			[]Transition{
				NewTransition(stateWorkflowPreparing, StartingWorkflow, stateWorkflowStarting),
				NewTransition(stateWorkflowPreparing, WorkflowBeingStarted, stateWorkflowStarting),
				NewTransition(stateWorkflowStarting, WorkflowStarted, stateWorkflowRunning),
				NewTransition(stateWorkflowStarting, CancelWorkflow, stateWorkflowCancelled),
				NewTransition(stateWorkflowRunning, TerminatingWorkflow, stateWorkflowTerminating),
				NewTransition(stateWorkflowTerminating, TerminatedWorkflow, stateWorkflowTerminated),
			},
			[]ModelInput{
				NewInput[*TaskQueue](),
				WorkflowID,
				RequestID,
				WorkflowType,
				Identity,
				WorkflowIdConflictPolicy,
				WorkflowIdReusePolicy,
				WorkflowRunID,
			},
			params,
		),
	}
}

func (w *WorkflowRun) ID() string {
	return WorkflowID.Get(w) + "/" + WorkflowRunID.Get(w)
}
