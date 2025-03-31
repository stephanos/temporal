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
// FITNESS FOR A PARTICULAR PURPOSE AND NGetONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package trigger

import (
	"cmp"

	commonpb "go.temporal.io/api/common/v1"
	enumspb "go.temporal.io/api/enums/v1"
	updatepb "go.temporal.io/api/update/v1"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/server/acceptance/model"
	"go.temporal.io/server/common/testing/stamp"
)

type StartWorkflowUpdate struct {
	Client         *model.WorkflowClient
	Workflow       *model.Workflow
	UseLatestRunID stamp.Gen[bool]
	WaitStage      stamp.GenProvider[WorkflowUpdateWaitStage]
	UpdateID       stamp.GenProvider[stamp.ID]
	UpdateHandler  stamp.Gen[string]
	Payload        Payloads
	Header         Header
}

func (w *StartWorkflowUpdate) Get(s stamp.Seeder) *workflowservice.UpdateWorkflowExecutionRequest {
	var runID string
	if !cmp.Or(w.UseLatestRunID, stamp.GenJust(true)).Get(s) {
		runID = w.Workflow.RunID
	}

	return &workflowservice.UpdateWorkflowExecutionRequest{
		Namespace: w.Workflow.GetScope().GetID(),
		WorkflowExecution: &commonpb.WorkflowExecution{
			WorkflowId: w.Workflow.GetID(),
			RunId:      runID,
		},
		WaitPolicy: &updatepb.WaitPolicy{
			LifecycleStage: enumspb.UpdateWorkflowExecutionLifecycleStage(w.WaitStage.Get(s)),
		},
		Request: &updatepb.Request{
			Meta: &updatepb.Meta{
				UpdateId: string(w.UpdateID.Get(s)),
				Identity: w.Client.GetID(),
			},
			Input: &updatepb.Input{
				Name:   w.UpdateHandler.Get(s),
				Header: w.Header.Get(s),
				Args:   w.Payload.Get(s),
			},
		},
	}
}

type WorkflowUpdateWaitStage enumspb.UpdateWorkflowExecutionLifecycleStage

func (g WorkflowUpdateWaitStage) Gen() stamp.Gen[WorkflowUpdateWaitStage] {
	panic("implement me")
}

type WorkflowUpdateRequest *updatepb.Request
