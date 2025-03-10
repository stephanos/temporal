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

	commandpb "go.temporal.io/api/command/v1"
	commonpb "go.temporal.io/api/common/v1"
	. "go.temporal.io/api/enums/v1"
	enumspb "go.temporal.io/api/enums/v1"
	"go.temporal.io/api/serviceerror"
	updatepb "go.temporal.io/api/update/v1"
	. "go.temporal.io/api/workflowservice/v1"
	. "go.temporal.io/server/common/proptest"
	"go.temporal.io/server/common/proptest/rule"
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
		Model[WorkflowUpdate]
		Workflow Scope[Workflow]
	}

	// TODO: use some kind of append log to track changes?

	WorkflowUpdateStartLog struct {
		OperationID
		*UpdateWorkflowExecutionRequest
	}
	WorkflowUpdatePollLog struct {
		OperationID
		*PollWorkflowExecutionUpdateRequest
	}
	WorkflowUpdateOutcomeLog struct {
		OperationID
		Outcome *updatepb.Outcome
		Error   serviceerror.ServiceError
	}
	WorkflowUpdateReqLog struct {
		Request *updatepb.Request
		Error   serviceerror.ServiceError
	}
	WorkflowUpdateRespLog struct {
		Message proto.Message
		Attrs   *commandpb.ProtocolMessageCommandAttributes
	}

	WorkflowUpdateHandler   string
	WorkflowUpdateWaitStage UpdateWorkflowExecutionLifecycleStage
	WorkflowUpdateRequest   *updatepb.Request
)

// ==== User Action Handlers

func (u *WorkflowUpdate) OnStartRequest(
	reqID grpcID,
	req *UpdateWorkflowExecutionRequest,
	_ incoming,
) (
	id ID,
	log Log,
) {
	id = ID(req.Request.Meta.UpdateId)
	log.Add(WorkflowUpdateStartLog{
		OperationID:                    OperationID(reqID),
		UpdateWorkflowExecutionRequest: req,
	})
	return
}

func (u *WorkflowUpdate) OnStartResponse(
	reqID grpcID,
	resp *UpdateWorkflowExecutionResponse,
	err serviceerror.ServiceError,
) (
	log Log,
) {
	if err != nil {
		log.Add(WorkflowUpdateOutcomeLog{
			OperationID: OperationID(reqID),
			Error:       err,
		})
	} else {
		log.Add(WorkflowUpdateOutcomeLog{
			OperationID: OperationID(reqID),
			Outcome:     resp.Outcome,
		})
	}
	return
}

func (u *WorkflowUpdate) OnPollRequest(
	reqID grpcID,
	req *PollWorkflowExecutionUpdateRequest,
	_ incoming,
) (
	id ID,
	log Log,
) {
	id = ID(req.UpdateRef.UpdateId)
	log.Add(WorkflowUpdatePollLog{
		OperationID:                        OperationID(reqID),
		PollWorkflowExecutionUpdateRequest: req,
	})
	return
}

func (u *WorkflowUpdate) OnPollResponse(
	reqID grpcID,
	resp *PollWorkflowExecutionUpdateResponse,
	err serviceerror.ServiceError,
) (
	log Log,
) {
	if err != nil {
		log.Add(WorkflowUpdateOutcomeLog{
			OperationID: OperationID(reqID),
			Error:       err,
		})
	} else {
		log.Add(WorkflowUpdateOutcomeLog{
			OperationID: OperationID(reqID),
			Outcome:     resp.Outcome,
		})
	}
	return
}

// ==== Internal Action Handlers

func (u *WorkflowUpdate) OnWorkflowTaskPoll(
	id ID,
	req *PollWorkflowTaskQueueRequest,
	resp *PollWorkflowTaskQueueResponse,
	err serviceerror.ServiceError,
) (
	log Log,
) {
	if err != nil {
		log.Add(WorkflowUpdateReqLog{Error: err})
		return
	}

	// TODO: only scan unprocessed messages
	for _, msg := range resp.Messages {
		if string(id) != msg.ProtocolInstanceId {
			break
		}

		body, err := msg.Body.UnmarshalNew()
		u.Assert.NoError(err, "failed to unmarshal message body")

		switch updMsg := body.(type) {
		case *updatepb.Request:
			log.Add(WorkflowUpdateReqLog{Request: updMsg})
		default:
			u.Assert.Fail(fmt.Sprintf("unknown message type: %T", body))
		}
	}

	return
}

func (u *WorkflowUpdate) OnWorkflowTaskCompleted(
	id ID,
	req *RespondWorkflowTaskCompletedRequest,
	resp *RespondWorkflowTaskCompletedResponse,
	err serviceerror.ServiceError,
) (
	log Log,
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
		if string(id) != msg.ProtocolInstanceId {
			break
		}

		body, err := msg.Body.UnmarshalNew()
		u.Assert.NoError(err, "failed to unmarshal message body")

		switch updMsg := body.(type) {
		case *updatepb.Acceptance:
			entry := WorkflowUpdateRespLog{Message: updMsg}
			entry.Attrs, commandAttrs = commandAttrs[0], commandAttrs[1:]
			log.Add(entry)
		case *updatepb.Rejection:
			log.Add(WorkflowUpdateRespLog{Message: updMsg})
			// NOTE: there is no command for a rejection
		case *updatepb.Response:
			entry := WorkflowUpdateRespLog{Message: updMsg}
			entry.Attrs, commandAttrs = commandAttrs[0], commandAttrs[1:]
			log.Add(entry)
		default:
			u.Assert.Fail(fmt.Sprintf("unknown message type: %T", body))
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
// - Update response from worker matches Worker's response data: Update ID etc.
// - when you poll after completion, you get the same outcome
// - if Update is completed, the history will end with
//   WorkflowExecutionUpdateAccepted and WorkflowExecutionUpdateCompleted,
//   where `AcceptedEventId` points to WorkflowExecutionUpdateAccepted,
//   and `AcceptedRequestSequencingEventId` points at its `WorkflowTaskScheduled`
// - if a speculative Update completes successfully, the speculative WFT is part of the
//   history
// - (need different generators for "with RunID" and "without RunID"

// TODO: flag to enforce or not (property can be seen as a var)?
func (u *WorkflowUpdate) IsCorrect() *rule.Rule {
	return rule.NewRule(
		`len(log) > 0`,
		rule.WithExample(),
		rule.WithCounterExample(),
	)
}

// TODO: move to rules
//u.Assert.Equal(string(id), updMsg.AcceptedRequest.Meta.UpdateId)
//u.Assert.Equal(string(id), updMsg.AcceptedRequestMessageId)
// TODO: updMsg.AcceptedRequestSequencingEventId

// TODO: move to rules
//u.Assert.Equal(string(id), updMsg.RejectedRequest.Meta.UpdateId)
//u.Assert.Equal(string(id), updMsg.RejectedRequestMessageId)
//u.Assert.NotZero(updMsg.Failure)
//u.Assert.NotZero(updMsg.Failure.Message)
// TODO: updMsg.RejectedRequestSequencingEventId

// TODO: move to rules
//u.Assert.Equal(string(id), updMsg.Meta.UpdateId)
//u.Assert.NotNil(updMsg.Outcome)

// ==== Action Generators

type GenWorkflowUpdateRequestInput struct {
	client         Scope[Client]
	namespace      Scope[Namespace]
	useLatestRunID Gen[LatestRunID]
	waitStage      Gen[WorkflowUpdateWaitStage]
	updateID       Gen[ID]
	updateHandler  Gen[WorkflowUpdateHandler]
	payload        Gen[Payloads]
	header         Gen[Header]
}

func (u *WorkflowUpdate) GenWorkflowUpdateRequest(
	gen GenWorkflowUpdateRequestInput,
) ActionGen[*UpdateWorkflowExecutionRequest] {
	var runID string
	if !gen.useLatestRunID.Next() {
		runID = string(MustGet[RunID](u.Workflow))
	}
	return &UpdateWorkflowExecutionRequest{
		Namespace: gen.namespace.GetID(),
		WorkflowExecution: &commonpb.WorkflowExecution{
			WorkflowId: u.Workflow.GetID(),
			RunId:      runID,
		},
		WaitPolicy: &updatepb.WaitPolicy{
			LifecycleStage: enumspb.UpdateWorkflowExecutionLifecycleStage(gen.waitStage.Next()),
		},
		Request: &updatepb.Request{
			Meta: &updatepb.Meta{
				UpdateId: string(gen.updateID.Next()),
				Identity: gen.client.GetID(),
			},
			Input: &updatepb.Input{
				Name:   string(gen.updateHandler.Next()),
				Header: gen.header.Next(),
				Args:   gen.payload.Next(),
			},
		},
	}
}

type GenWorkflowPollRequestInput struct {
	client         Scope[Client]
	namespace      Scope[Namespace]
	workflowUpdate Scope[WorkflowUpdate]
	waitStage      Gen[WorkflowUpdateWaitStage]
}

func (u *WorkflowUpdate) GenWorkflowPollRequest(
	gen GenWorkflowPollRequestInput,
) ActionGen[*PollWorkflowExecutionUpdateRequest] {
	return &PollWorkflowExecutionUpdateRequest{
		Namespace: gen.namespace.GetID(),
		UpdateRef: &updatepb.UpdateRef{
			WorkflowExecution: &commonpb.WorkflowExecution{
				WorkflowId: u.Workflow.GetID(),
				RunId:      string(MustGet[RunID](u.Workflow)),
			},
			UpdateId: gen.workflowUpdate.GetID(),
		},
		Identity: gen.client.GetID(),
		WaitPolicy: &updatepb.WaitPolicy{
			LifecycleStage: enumspb.UpdateWorkflowExecutionLifecycleStage(gen.waitStage.Next()),
		},
	}
}

// TODO: listen for parent events; such as state change to "dead"
