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
	workflowUpdatePreparing = NewState("Preparing", Init)
	workflowUpdateStarting  = NewState("Starting")
	workflowUpdateRunning   = NewState("Running")
	workflowUpdateAccepted  = NewState("Accepted")
	workflowUpdateRejected  = NewState("Rejected")
	workflowUpdateCompleted = NewState("Completed")
	workflowUpdateCancelled = NewState("Cancelled") // ie never started
)

// ==== Events

var (
	StartingWorkflowUpdate = NewEvent[any]("Starting",
		func(m *Model, _ any) {

		})
	WorkflowUpdateBeingStarted = NewEvent("BeingStarted",
		func(m *Model, resp *workflowservice.UpdateWorkflowExecutionResponse) {
			// nothing to do
		})
	CancelledWorkflowUpdate = NewEvent[error]("Cancel", func(m *Model, err error) {
		m.Errorf("cancelled: %v\n", err)
	})
	StartedWorkflowUpdate = NewEvent("Started",
		func(m *Model, resp *workflowservice.UpdateWorkflowExecutionResponse) {
			switch resp.Stage {
			case enumspb.UPDATE_WORKFLOW_EXECUTION_LIFECYCLE_STAGE_ADMITTED:
				// nothing to do
			case enumspb.UPDATE_WORKFLOW_EXECUTION_LIFECYCLE_STAGE_ACCEPTED:
				AcceptedWorkflowUpdate.Fire(m, nil)
			case enumspb.UPDATE_WORKFLOW_EXECUTION_LIFECYCLE_STAGE_COMPLETED:
				CompletedWorkflowUpdate.Fire(m, resp)
			default:
				panic("unexpected stage")
			}
		})
	AcceptedWorkflowUpdate = NewEvent("Accept",
		func(m *Model, resp *workflowservice.UpdateWorkflowExecutionResponse) {
			// nothing to do
		})
	RejectedWorkflowUpdate = NewEvent("Reject",
		func(m *Model, resp *workflowservice.UpdateWorkflowExecutionResponse) {
			//resp.outcome = resp.Outcome
			//resp.NotZero(resp.outcome.GetFailure())
			// TODO: verify that last WFT is still `WorkflowTaskCompleted`
		})
	CompletedWorkflowUpdate = NewEvent("Complete",
		func(m *Model, resp *workflowservice.UpdateWorkflowExecutionResponse) {
			// nothing to do
		})
)

// ==== Actions

var (
	StartWorkflowUpdate = NewAction("StartWorkflowUpdate", StartingWorkflowUpdate)
)

// ==== Inputs

var (
	UpdateID      = NewInput(Default(GenId("id")))
	UpdatePayload = NewInput(Default(GenId("pl")))
	UpdateHandler = NewInput(Default(GenId("hdl")))
	WaitStage     = NewInput(Default(GenPick(WaitStageSet)))
)

// ==== Sets

var (
	WaitStageSet = []enumspb.UpdateWorkflowExecutionLifecycleStage{
		enumspb.UPDATE_WORKFLOW_EXECUTION_LIFECYCLE_STAGE_ACCEPTED,
		enumspb.UPDATE_WORKFLOW_EXECUTION_LIFECYCLE_STAGE_COMPLETED,
	}
)

type WorkflowUpdate struct {
	*Model
}

func NewWorkflowExecutionUpdate(
	params ...any,
) (res *WorkflowUpdate) {
	return &WorkflowUpdate{
		Model: NewModel(
			"WorkflowUpdate",
			[]Transition{
				NewTransition(workflowUpdatePreparing, StartingWorkflowUpdate, workflowUpdateStarting),
				NewTransition(workflowUpdatePreparing, WorkflowUpdateBeingStarted, workflowUpdateStarting),
				NewTransition(workflowUpdateStarting, CancelledWorkflowUpdate, workflowUpdateCancelled),
				NewTransition(workflowUpdateStarting, StartedWorkflowUpdate, workflowUpdateRunning),
				NewTransition(workflowUpdateRunning, AcceptedWorkflowUpdate, workflowUpdateAccepted),
				NewTransition(workflowUpdateRunning, RejectedWorkflowUpdate, workflowUpdateRejected),
				NewTransition(workflowUpdateRunning, CompletedWorkflowUpdate, workflowUpdateCompleted),
				NewTransition(workflowUpdateAccepted, CompletedWorkflowUpdate, workflowUpdateCompleted),
			},
			[]ModelInput{
				NewInput[*WorkflowRun](),
				UpdateID,
				UpdatePayload,
				UpdateHandler,
				WaitStage,
			},
			params),
	}
}

func (u *WorkflowUpdate) ID() string {
	return WorkflowRunID.Get(u) + "/" + UpdateID.Get(u)
}

func (u *WorkflowUpdate) AwaitCompletion() {
	WaitForState(
		u.Model,
		workflowUpdateCancelled,
		workflowUpdateRejected,
		workflowUpdateCompleted)
}
