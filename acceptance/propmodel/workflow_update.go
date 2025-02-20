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
	updatepb "go.temporal.io/api/update/v1"
	. "go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/server/common/payloads"
	. "go.temporal.io/server/common/proptest"
	"go.temporal.io/server/common/proptest/expect"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
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

var (
	SingleOutcome = Rule(
		"Update Outcomes never change",
		All(
			expect.SetOnceEventually[ExecuteOutcome](),
			expect.SetOnceEventually[PollOutcome](),
		),
	)
	//SameOutcome = Rule(
	//	"Update Outcomes from execution and polling must only be set once",
	//)
)

func (u *WorkflowUpdate) Id(
	msg proto.Message,
) ID {
	// TODO: UpdateID is not required by the client to set, need to figure out what to do if it's not set
	updateID := findProtoValueByNameType[string](msg, "update_id", protoreflect.StringKind)
	return ID(updateID)
}

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

func (u *WorkflowUpdate) Verify() expect.Rule {
	return SingleOutcome
}

func GenWorkflowUpdateRequest(
	env *Env,
	client Scope[Client],
	namespace Scope[Namespace],
	workflow Scope[Workflow],
	waitStage Gen[WaitStage],
	updateID Gen[ID],
	updateHandler Gen[UpdateHandler],
) ActionGen[*UpdateWorkflowExecutionRequest] {
	return &UpdateWorkflowExecutionRequest{
		Namespace: namespace.GetID(),
		WorkflowExecution: &commonpb.WorkflowExecution{
			WorkflowId: workflow.GetID(),
			RunId:      string(MustGet[RunID](workflow)),
		},
		//WaitPolicy: &updatepb.WaitPolicy{LifecycleStage: waitStage.Next()},
		Request: &updatepb.Request{
			Meta: &updatepb.Meta{
				UpdateId: string(updateID.Next(env)),
				Identity: client.GetID(),
			},
			Input: &updatepb.Input{
				Name: string(updateHandler.Next(env)),
				//Header:
				Args: payloads.EncodeString("args-value-of-" + string(updateID.Next(env))),
			},
		},
	}
}

// TODO: listen for parent events; such as state change to "dead"
