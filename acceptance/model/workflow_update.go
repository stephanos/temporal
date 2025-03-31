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

	commandpb "go.temporal.io/api/command/v1"
	enumspb "go.temporal.io/api/enums/v1"
	"go.temporal.io/api/serviceerror"
	updatepb "go.temporal.io/api/update/v1"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/server/common/testing/stamp"
	"go.temporal.io/server/common/testing/testmarker"
	"google.golang.org/protobuf/proto"
)

// TODO: always run one happy-case base test before going any deeper

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
		stamp.Scope[*Workflow]
	}

	WorkflowUpdateStartRecord struct {
		stamp.OperationID
		*workflowservice.UpdateWorkflowExecutionRequest
	}
	WorkflowUpdatePollRecord struct {
		stamp.OperationID
		*workflowservice.PollWorkflowExecutionUpdateRequest
	}
	WorkflowUpdateOutcomeRecord struct {
		stamp.OperationID
		Outcome *updatepb.Outcome
		Error   serviceerror.ServiceError
	}
	WorkflowUpdateReqRecord struct {
		Request *updatepb.Request
		Error   serviceerror.ServiceError
	}
	WorkflowUpdateRespRecord struct {
		Message proto.Message
		Attrs   *commandpb.ProtocolMessageCommandAttributes
	}
)

// ==== Identity Handlers

func (u *WorkflowUpdate) IdRequest(req RequestMsg) stamp.ID {
	switch t := req.(type) {
	case *workflowservice.UpdateWorkflowExecutionRequest:
		return stamp.ID(t.Request.Meta.UpdateId)
	case *workflowservice.PollWorkflowExecutionUpdateRequest:
		return stamp.ID(t.UpdateRef.UpdateId)
	default:
		return ""
	}
}

// ==== Action Handlers

func (u *WorkflowUpdate) OnStartRequest(
	reqID ActionID,
	req *workflowservice.UpdateWorkflowExecutionRequest,
	_ Incoming,
) {
	u.Record(WorkflowUpdateStartRecord{
		OperationID:                    stamp.OperationID(reqID),
		UpdateWorkflowExecutionRequest: req,
	})
}

func (u *WorkflowUpdate) OnStartResponse(
	reqID ActionID,
	resp *workflowservice.UpdateWorkflowExecutionResponse,
	err serviceerror.ServiceError,
) {
	if err != nil {
		u.Record(WorkflowUpdateOutcomeRecord{
			OperationID: stamp.OperationID(reqID),
			Error:       err,
		})
	} else {
		u.Record(WorkflowUpdateOutcomeRecord{
			OperationID: stamp.OperationID(reqID),
			Outcome:     resp.Outcome,
		})
	}
	return
}

func (u *WorkflowUpdate) OnPollRequest(
	reqID ActionID,
	req *workflowservice.PollWorkflowExecutionUpdateRequest,
	_ Incoming,
) {
	u.Record(WorkflowUpdatePollRecord{
		OperationID:                        stamp.OperationID(reqID),
		PollWorkflowExecutionUpdateRequest: req,
	})
	return
}

func (u *WorkflowUpdate) OnPollResponse(
	reqID ActionID,
	resp *workflowservice.PollWorkflowExecutionUpdateResponse,
	err serviceerror.ServiceError,
) {
	if err != nil {
		u.Record(WorkflowUpdateOutcomeRecord{
			OperationID: stamp.OperationID(reqID),
			Error:       err,
		})
	} else {
		u.Record(WorkflowUpdateOutcomeRecord{
			OperationID: stamp.OperationID(reqID),
			Outcome:     resp.Outcome,
		})
	}
	return
}

func (u *WorkflowUpdate) OnWorkflowTaskPoll(
	req *workflowservice.PollWorkflowTaskQueueRequest,
	resp *workflowservice.PollWorkflowTaskQueueResponse,
	err serviceerror.ServiceError,
) {
	if err != nil {
		u.Record(WorkflowUpdateReqRecord{Error: err})
		return
	}

	// TODO: only scan unprocessed messages
	for _, msg := range resp.Messages {
		if u.GetID() != msg.ProtocolInstanceId {
			break
		}

		body, err := msg.Body.UnmarshalNew()
		if err != nil {
			panic(fmt.Sprintf("failed to unmarshal message body: %v", err))
		}

		switch updMsg := body.(type) {
		case *updatepb.Request:
			u.Record(WorkflowUpdateReqRecord{Request: updMsg})
		default:
			panic(fmt.Sprintf("unknown message type: %T", body))
		}
	}

	return
}

func (u *WorkflowUpdate) OnWorkflowTaskCompleted(
	req *workflowservice.RespondWorkflowTaskCompletedRequest,
	resp *workflowservice.RespondWorkflowTaskCompletedResponse,
	err serviceerror.ServiceError,
) {
	if err != nil {
		// TODO
		return
	}

	var commandAttrs []*commandpb.ProtocolMessageCommandAttributes
	for _, cmd := range req.Commands {
		switch cmd.CommandType {
		case enumspb.COMMAND_TYPE_PROTOCOL_MESSAGE:
			commandAttrs = append(commandAttrs, cmd.GetProtocolMessageCommandAttributes())
		}
	}

	for _, msg := range req.Messages {
		if u.GetID() != msg.ProtocolInstanceId {
			break
		}

		body, err := msg.Body.UnmarshalNew()
		if err != nil {
			panic(fmt.Sprintf("failed to unmarshal message body: %v", err))
		}

		switch updMsg := body.(type) {
		case *updatepb.Acceptance:
			entry := WorkflowUpdateRespRecord{Message: updMsg}
			entry.Attrs, commandAttrs = commandAttrs[0], commandAttrs[1:]
			u.Record(entry)
		case *updatepb.Rejection:
			u.Record(WorkflowUpdateRespRecord{Message: updMsg})
			// NOTE: there is no command for a rejection
		case *updatepb.Response:
			entry := WorkflowUpdateRespRecord{Message: updMsg}
			entry.Attrs, commandAttrs = commandAttrs[0], commandAttrs[1:]
			u.Record(entry)
		default:
			panic(fmt.Sprintf("unknown message type: %T", body))
		}
	}

	return
}

// ==== Rules

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

func (u *WorkflowUpdate) TestCorrect() stamp.Rule {
	return stamp.RuleExpr(
		u,
		`true`,
		stamp.WithExample(
			true,
			&WorkflowUpdateStartRecord{
				UpdateWorkflowExecutionRequest: &workflowservice.UpdateWorkflowExecutionRequest{
					Request: &updatepb.Request{},
				},
			},
		),
	)
}

// TODO: move to rules
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

// ==== Triggers

// TODO: listen for parent events; such as state change to "dead"
