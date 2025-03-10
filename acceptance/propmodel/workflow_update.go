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

package propmodel

import (
	"fmt"

	commonpb "go.temporal.io/api/common/v1"
	. "go.temporal.io/api/enums/v1"
	enumspb "go.temporal.io/api/enums/v1"
	updatepb "go.temporal.io/api/update/v1"
	. "go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/server/common/log"
	"go.temporal.io/server/common/log/tag"
	. "go.temporal.io/server/common/proptest"
	"go.temporal.io/server/common/proptest/expect"
	"go.temporal.io/server/common/softassert"
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

const (
	updateStarting WorkflowUpdateLifecycleStage = iota
	updateSent
	updateAccepted
	updateRejected
	updateCompleted
	update
)

type (
	WorkflowUpdate struct {
		Model[WorkflowUpdate]
		Workflow Scope[Workflow]
	}

	UpdateStart struct {
		WaitStage WorkflowUpdateWaitStage
	}
	UpdatePoll struct {
		WaitStage WorkflowUpdateWaitStage
	}

	WorkflowUpdateStarts         map[grpcID]UpdateStart
	WorkflowUpdatePolls          map[grpcID]UpdatePoll
	WorkflowUpdateHandler        string
	WorkflowUpdateLifecycleStage int
	WorkflowUpdateWaitStage      UpdateWorkflowExecutionLifecycleStage
	WorkflowUpdateExecuteOutcome *updatepb.Outcome
	WorkflowUpdatePollOutcome    *updatepb.Outcome
)

// ==== User Action Handlers

func (u *WorkflowUpdate) OnStartRequest(
	reqID grpcID,
	req *UpdateWorkflowExecutionRequest,
	_ incoming,
) (
	id ID,
	updateStarts WorkflowUpdateStarts,
	updateLifecycleStage WorkflowUpdateLifecycleStage,
) {
	id = ID(req.GetRequest().GetMeta().GetUpdateId())
	updateLifecycleStage = updateStarting
	updateStarts = map[grpcID]UpdateStart{
		reqID: {
			WaitStage: WorkflowUpdateWaitStage(*req.GetWaitPolicy().GetLifecycleStage().Enum()),
		},
	}
	return
}

func (u *WorkflowUpdate) OnStartResponse(
	resp *UpdateWorkflowExecutionResponse,
) (
	id ID,
	executeOutcome WorkflowUpdateExecuteOutcome,
) {
	id = ID(resp.GetUpdateRef().GetUpdateId())
	executeOutcome = protoClone(resp.GetOutcome())
	return
}

func (u *WorkflowUpdate) OnPollRequest(
	reqID grpcID,
	req *PollWorkflowExecutionUpdateRequest,
	_ incoming,
) (
	updatePolls WorkflowUpdatePolls,
) {
	updatePolls = map[grpcID]UpdatePoll{
		reqID: {
			WaitStage: WorkflowUpdateWaitStage(*req.GetWaitPolicy().GetLifecycleStage().Enum()),
		},
	}
	return
}

func (u *WorkflowUpdate) OnPollResponse(
	resp *PollWorkflowExecutionUpdateResponse,
) (
	id ID,
	pollOutcome WorkflowUpdatePollOutcome,
	updateLifecycleStage WorkflowUpdateLifecycleStage,
) {
	id = ID(resp.GetUpdateRef().GetUpdateId())
	pollOutcome = protoClone(resp.GetOutcome())
	return
}

// ==== Internal Action Handlers

func (u *WorkflowUpdate) OnWorkflowTaskPoll(
	id ID,
	logger log.Logger,
	req *PollWorkflowTaskQueueRequest,
	resp *PollWorkflowTaskQueueResponse,
) (
	updateLifecycleStage WorkflowUpdateLifecycleStage,
) {
	// TODO: only scan unprocessed messages
	for _, msg := range resp.GetMessages() {
		if string(id) != msg.GetProtocolInstanceId() {
			break
		}
		//body, err := msg.Body.UnmarshalNew()
		//if err != nil {
		//	softassert.Fail(logger, "failed to unmarshal message body", tag.Error(err))
		//	continue
		//}
	}
	return
}

func (u *WorkflowUpdate) OnWorkflowTaskCompleted(
	id ID,
	logger log.Logger,
	req *RespondWorkflowTaskCompletedRequest,
	resp *RespondWorkflowTaskCompletedResponse,
) (
	updateLifecycleStage WorkflowUpdateLifecycleStage,
) {
	for _, msg := range req.GetMessages() {
		if string(id) != msg.GetProtocolInstanceId() {
			break
		}

		body, err := msg.Body.UnmarshalNew()
		if err != nil {
			softassert.Fail(logger, "failed to unmarshal message body", tag.Error(err))
			continue
		}

		switch updMsg := body.(type) {
		case *updatepb.Acceptance:
			//updMsg.AcceptedRequestSequencingEventId
			//updMsg.AcceptedRequestSequencingEventId
			//updMsg.AcceptedRequest
			updateLifecycleStage = updateAccepted
		case *updatepb.Rejection:
			//updMsg.RejectedRequest
			//updMsg.RejectedRequestMessageId
			//updMsg.RejectedRequestSequencingEventId
			updateLifecycleStage = updateRejected
		case *updatepb.Response:
			softassert.That(logger, string(id) == updMsg.Meta.UpdateId, "TODO")
			updateLifecycleStage = updateCompleted
		default:
			softassert.Fail(logger, fmt.Sprintf("unknown message type: %T", body))
		}
	}
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

// ==== Action Generators

func (u *WorkflowUpdate) GenWorkflowUpdateRequest(
	client Scope[Client],
	namespace Scope[Namespace],
	useLatestRunID Gen[LatestRunID],
	waitStage Gen[WorkflowUpdateWaitStage],
	updateID Gen[ID],
	updateHandler Gen[WorkflowUpdateHandler],
	payload Gen[Payloads],
	header Gen[Header],
) ActionGen[*UpdateWorkflowExecutionRequest] {
	var runID string
	if !useLatestRunID.Next() {
		runID = string(MustGet[RunID](u.Workflow))
	}

	return &UpdateWorkflowExecutionRequest{
		Namespace: namespace.GetID(),
		WorkflowExecution: &commonpb.WorkflowExecution{
			WorkflowId: u.Workflow.GetID(),
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

// TODO: generate functions that can be used anywhere to create a new request; using a map as input to allow optional fields
func (u *WorkflowUpdate) GenWorkflowPollRequest(
	client Scope[Client],
	namespace Scope[Namespace],
	workflowUpdate Scope[WorkflowUpdate],
	waitStage Gen[WorkflowUpdateWaitStage],
) ActionGen[*PollWorkflowExecutionUpdateRequest] {
	return &PollWorkflowExecutionUpdateRequest{
		Namespace: namespace.GetID(),
		UpdateRef: &updatepb.UpdateRef{
			WorkflowExecution: &commonpb.WorkflowExecution{
				WorkflowId: u.Workflow.GetID(),
				RunId:      string(MustGet[RunID](u.Workflow)),
			},
			UpdateId: workflowUpdate.GetID(),
		},
		Identity: client.GetID(),
		WaitPolicy: &updatepb.WaitPolicy{
			LifecycleStage: enumspb.UpdateWorkflowExecutionLifecycleStage(waitStage.Next()),
		},
	}
}

func (u *WorkflowUpdateLifecycleStage) String() string {
	switch *u {
	case updateStarting:
		return "Starting"
	case updateSent:
		return "Sent"
	case updateAccepted:
		return "Accepted"
	case updateRejected:
		return "Rejected"
	case updateCompleted:
		return "Completed"
	default:
		panic(fmt.Sprintf("unknown WorkflowUpdateLifecycleStage: %v", *u))
	}
}

// TODO: listen for parent events; such as state change to "dead"
