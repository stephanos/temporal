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
	commonpb "go.temporal.io/api/common/v1"
	. "go.temporal.io/api/enums/v1"
	enumspb "go.temporal.io/api/enums/v1"
	updatepb "go.temporal.io/api/update/v1"
	. "go.temporal.io/api/workflowservice/v1"
	. "go.temporal.io/server/common/proptest"
	"go.temporal.io/server/common/proptest/expect"
	"google.golang.org/protobuf/proto"
)

// TODO props:
// - verify speculative task queue was used *when expected* (check persistence writes ...)
// - only one roundtrip to worker if there are no faults
// - server receives correct worker outcome
// - update request message has correct eventId
// - ResetHistoryEventId is zero?
// - poll after completion returns same result (and UpdateRef contains correct RunID, too)
// - on completion: history contains correct events
// - on rejection: history is reset (or not if not possible!)
// - Update is correctly processed when worker *also* sends another command
// - Update is delivered as Normal WFT when it's on the WF's first one
// - when there's a buffered event (signal), then ...
// - TODO (continue from TestUpdateWorkflow_RunningWorkflowTask_NewEmptySpeculativeWorkflowTask_Rejected)

// with Alex
// IF ... THEN ....
// - when receiving a history from a Poll, the last event must be WFTStarted with
//   a scheduledEvent attribute pointing to the previous event, which is WFTScheduled
// - Update received by worker matches requested Update: Update ID, Update Handler etc.
// - eventually you either get an error or an Outcome
// - Update response from worker matches Worker's response data: Update ID etc.
// - when you poll after completion, you get the same outcome
// - if Update is completed, the history will end with
//   WorkflowExecutionUpdateAccepted and WorkflowExecutionUpdateCompleted,
//   where `AcceptedEventId` points to WorkflowExecutionUpdateAccepted,
//   and `AcceptedRequestSequencingEventId` points at its `WorkflowTaskScheduled`
// - if a speculative Update completes successfully, the speculative WFT is part of the
//   history
// - (need different generators for "with RunID" and "without RunID"

// https://github.com/Dicklesworthstone/introduction_to_temporal_logic

// Next
// Until
// Eventually
// Always
// expect.After(expect.NotZero[ExecuteOutcome], expect.Zero[ExecuteOutcome()),

type (
	WorkflowUpdate struct {
		Model[WorkflowUpdate]
		Workflow Scope[Workflow]
	}

	UpdateHandler  string
	WaitStage      UpdateWorkflowExecutionLifecycleStage
	ExecuteOutcome *updatepb.Outcome
	PollOutcome    *updatepb.Outcome
)

func (u *WorkflowUpdate) Id(
	msg proto.Message,
) ID {
	switch t := msg.(type) {
	case *UpdateWorkflowExecutionRequest:
		return ID(t.GetRequest().GetMeta().GetUpdateId())
	case *UpdateWorkflowExecutionResponse:
		return ID(t.GetUpdateRef().UpdateId)
	case *PollWorkflowExecutionUpdateResponse:
		return ID(t.GetUpdateRef().UpdateId)
	case *PollWorkflowTaskQueueResponse:
		// TODO: only scan unprocessed messages
		for _, msg := range t.GetMessages() {
			if id := u.GetID(); id == msg.GetProtocolInstanceId() {
				return ID(id)
			}
		}
		// TODO: scan history events
	case *RespondWorkflowTaskCompletedRequest:
		for _, msg := range t.GetMessages() {
			if id := u.GetID(); id == msg.GetProtocolInstanceId() {
				return ID(id)
			}
		}
	case *RespondWorkflowTaskFailedRequest:
		// TODO
	}
	return ""
}

// TODO: for identifying the update when there's multiple
//func (u *WorkflowUpdate) Id(
//	identity ID,
//	msg proto.Message,
//) bool {
//	return false
//}

func (u *WorkflowUpdate) OnExecuteRequest(
	req *UpdateWorkflowExecutionRequest,
) (
	waitStage WaitStage,
) {
	waitStage = WaitStage(*req.GetWaitPolicy().GetLifecycleStage().Enum())
	return
}

func (u *WorkflowUpdate) OnExecuteResponse(
	resp *UpdateWorkflowExecutionResponse,
) (
	executeOutcome ExecuteOutcome,
) {
	executeOutcome = protoClone(resp.GetOutcome())
	return
}

func (u *WorkflowUpdate) OnPollResponse(
	resp *PollWorkflowExecutionUpdateResponse,
) (
	pollOutcome PollOutcome,
) {
	pollOutcome = protoClone(resp.GetOutcome())
	return
}

func (u *WorkflowUpdate) OnWorkflowTaskPoll(
	resp *PollWorkflowTaskQueueResponse,
) (
	waitStage WaitStage,
) {
	return
}

func (u *WorkflowUpdate) OnWorkflowTaskCompleted(
	req *RespondWorkflowTaskCompletedRequest,
) (
	waitStage WaitStage,
) {
	return
}

//func (u *WorkflowUpdate) State() *StateMachine {
//	return NewStateMachine().
//		Add(StateDraft.SubState("Running"))
//}

// TODO: we need a state machine that is based on a variable (and nothing more; if it needs more data, use a struct)
// then rules/states can have names that can be referenced by other state machines to "tie in";
// for example, the outcome state machine might reference the lifecycle state machine using "guards"/"preconditions"
func (u *WorkflowUpdate) VerifyOutcomes() expect.Rule {
	return expect.NewRule()
}

func GenWorkflowUpdateRequest(
	client Scope[Client],
	namespace Scope[Namespace],
	workflow Scope[Workflow],
	useLatestRunID Gen[LatestRunID],
	waitStage Gen[WaitStage],
	updateID Gen[ID],
	updateHandler Gen[UpdateHandler],
	payload Gen[Payloads],
	header Gen[Header],
) ActionGen[*UpdateWorkflowExecutionRequest] {
	var runID string
	if !useLatestRunID.Next() {
		runID = string(MustGet[RunID](workflow))
	}

	return &UpdateWorkflowExecutionRequest{
		Namespace: namespace.GetID(),
		WorkflowExecution: &commonpb.WorkflowExecution{
			WorkflowId: workflow.GetID(),
			RunId:      runID,
		},
		WaitPolicy: &updatepb.WaitPolicy{
			LifecycleStage: enumspb.UpdateWorkflowExecutionLifecycleStage(waitStage.Next()),
		},
		Request: &updatepb.Request{
			Meta: &updatepb.Meta{
				UpdateId: string(updateID.Next()),
				Identity: client.GetID(),
			},
			Input: &updatepb.Input{
				Name:   string(updateHandler.Next()),
				Header: header.Next(),
				Args:   payload.Next(),
			},
		},
	}
}

func GenWorkflowPollRequest(
	client Scope[Client],
	namespace Scope[Namespace],
	workflow Scope[Workflow],
	workflowUpdate Scope[WorkflowUpdate],
	waitStage Gen[WaitStage],
) ActionGen[*PollWorkflowExecutionUpdateRequest] {
	return &PollWorkflowExecutionUpdateRequest{
		Namespace: namespace.GetID(),
		UpdateRef: &updatepb.UpdateRef{
			WorkflowExecution: &commonpb.WorkflowExecution{
				WorkflowId: workflow.GetID(),
				RunId:      string(MustGet[RunID](workflow)),
			},
			UpdateId: workflowUpdate.GetID(),
		},
		Identity: client.GetID(),
		WaitPolicy: &updatepb.WaitPolicy{
			LifecycleStage: enumspb.UpdateWorkflowExecutionLifecycleStage(waitStage.Next()),
		},
	}
}

// TODO: listen for parent events; such as state change to "dead"
