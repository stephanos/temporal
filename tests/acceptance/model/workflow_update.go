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

package model

import (
	"fmt"

	protocolpb "go.temporal.io/api/protocol/v1"
	updatepb "go.temporal.io/api/update/v1"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/server/common/testing/stamp"
	"go.temporal.io/server/common/testing/testmarker"
)

var (
	WorkflowUpdateAccepted = testmarker.New(
		"WorkflowUpdate.Accepted",
		"Update was accepted by the update validator on the worker.")
	WorkflowUpdateRejected = testmarker.New(
		"WorkflowUpdate.Rejected",
		"Update was rejected by the update validator on the worker.")
	WorkflowUpdateCompletedWithAccepted = testmarker.New(
		"WorkflowUpdate.Completed-with-Accepted",
		"Update was accepted and completed by the update handler on the worker in a single WorkflowTask.")
	WorkflowUpdateCompletedAfterAccepted = testmarker.New(
		"WorkflowUpdate.Completed-after-Accepted",
		"Update was completed by the update handler on the worker in a separate WorkflowTask from the Acceptance.")
	WorkflowUpdateResurrectedFromAcceptance = testmarker.New(
		"WorkflowUpdate.Resurrected-from-Acceptance",
		"Update was resurrected from an Acceptance response from the worker after being lost.")
	WorkflowUpdateResurrectedFromRejection = testmarker.New(
		"WorkflowUpdate.Resurrected-from-Rejection",
		"Update was resurrected from a Rejection response from the worker after being lost.")
)

type (
	WorkflowUpdate struct {
		stamp.Model[*WorkflowUpdate]
		stamp.Scope[*WorkflowExecution]

		Request    *protocolpb.Message
		Acceptance *updatepb.Acceptance
		Failure    *updatepb.Rejection
		Response   *updatepb.Response
	}
)

func (u *WorkflowUpdate) GetWorkflow() *Workflow {
	return u.GetScope().GetScope()
}

func (u *WorkflowUpdate) OnUpdateWorkflowExecution(
	_ IncomingAction[*workflowservice.UpdateWorkflowExecutionRequest],
) {
	// TODO
}

func (u *WorkflowUpdate) OnPollWorkflowTaskQueue(
	_ IncomingAction[*workflowservice.PollWorkflowTaskQueueRequest],
) func(OutgoingAction[*workflowservice.PollWorkflowTaskQueueResponse]) {
	return func(out OutgoingAction[*workflowservice.PollWorkflowTaskQueueResponse]) {
		for _, msg := range u.ownMessages(out.Response.Messages) {
			u.Require.Nil(u.Request, "Request is already set")
			u.Request = msg
		}
		u.Require.NotNil(u.Request, "Request is nil")
	}
}

func (u *WorkflowUpdate) OnRespondWorkflowTaskCompleted(
	in IncomingAction[*workflowservice.RespondWorkflowTaskCompletedRequest],
) {
	for _, msg := range u.ownMessages(in.Request.Messages) {
		body, err := msg.Body.UnmarshalNew()
		if err != nil {
			// TODO: record error
			continue
		}
		switch t := body.(type) {
		case *updatepb.Acceptance:
			u.Acceptance = t
		case *updatepb.Rejection:
			u.Failure = t
		case *updatepb.Response:
			u.Response = t
		default:
			panic(fmt.Sprintf("unknown message type: %T", body))
		}
	}
}

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

// with Alex
// - when receiving a history from a Poll, the last event must be WFTStarted with
//   a scheduledEvent attribute pointing to the previous event, which is WFTScheduled
// - Update received by worker matches requested Update: Update ID, Update Handler etc.
// - eventually you either get an error or an Outcome
// - Update response from worker matches WorkflowWorker's response data: Update ID etc.
// - when you poll after completion, you get the same outcome
// - if Update is completed, the history will end with
//   WorkflowExecutionUpdateAccepted and WorkflowExecutionUpdateCompleted,
//   where `AcceptedEventId` points to WorkflowExecutionUpdateAccepted,
//   and `AcceptedRequestSequencingEventId` points at its `WorkflowTaskScheduled`
// - if a speculative Update completes successfully, the speculative WFT is part of the
//   history
// - (need different generators for "with RunID" and "without RunID"

func (u *WorkflowUpdate) IsDone() stamp.Prop[bool] {
	return stamp.NewProp(u, func(*stamp.PropContext) bool {
		return u.Response != nil || u.Failure != nil
	})
}

//u.Assert.Equal(string(id), updMsg.AcceptedRequest.Meta.UpdateId)
//u.Assert.Equal(string(id), updMsg.AcceptedRequestMessageId)
// TODO: updMsg.AcceptedRequestSequencingEventId

//u.Assert.Equal(string(id), updMsg.RejectedRequest.Meta.UpdateId)
//u.Assert.Equal(string(id), updMsg.RejectedRequestMessageId)
//u.Assert.NotZero(updMsg.Failure)
//u.Assert.NotZero(updMsg.Failure.Message)
// TODO: updMsg.RejectedRequestSequencingEventId

//u.Assert.Equal(string(id), updMsg.Meta.UpdateId)
//u.Assert.NotNil(updMsg.Outcome)

// TODO: listen for parent events; such as state change to "dead"

func (u *WorkflowUpdate) ownMessages(
	messages []*protocolpb.Message,
) []*protocolpb.Message {
	var result []*protocolpb.Message
	for _, msg := range messages {
		if string(u.GetID()) == msg.ProtocolInstanceId {
			result = append(result, msg)
		}
	}
	return result
}
