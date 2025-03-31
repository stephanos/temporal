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
	"time"

	commonpb "go.temporal.io/api/common/v1"
	"go.temporal.io/api/taskqueue/v1"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/server/acceptance/model"
	"go.temporal.io/server/common/testing/stamp"
	"google.golang.org/protobuf/types/known/durationpb"
)

type StartWorkflowExecutionRequest struct {
	stamp.TriggerActor[*model.WorkflowClient]
	stamp.TriggerTarget[*model.Workflow]
	TaskQueue                *model.TaskQueue
	Payload                  Payloads
	WorkflowID               stamp.GenProvider[stamp.ID]
	WorkflowType             stamp.Gen[string]
	WorkflowExecutionTimeout stamp.Gen[time.Duration]
	WorkflowRunTimeout       stamp.Gen[time.Duration]
	WorkflowTaskTimeout      stamp.Gen[time.Duration]
	RequestId                stamp.GenProvider[stamp.ID]
}

// TODO: distinguish between parameters that are random but don't change the behavior - and the ones that do
// TODO: can we use the original instead; and for every field that isn't set, generate a reasonable default instead?
func (w StartWorkflowExecutionRequest) Get(s stamp.Seeder) *workflowservice.StartWorkflowExecutionRequest {
	return &workflowservice.StartWorkflowExecutionRequest{
		Namespace:  w.TaskQueue.GetScope().GetID(),
		WorkflowId: string(w.WorkflowID.Get(s)),
		WorkflowType: &commonpb.WorkflowType{
			Name: cmp.Or(w.WorkflowType, stamp.GenName[string]()).Get(s),
		},
		TaskQueue: &taskqueue.TaskQueue{
			Name: w.TaskQueue.GetID(),
		},
		Input:                    w.Payload.Get(s),
		WorkflowExecutionTimeout: durationpb.New(cmp.Or(w.WorkflowExecutionTimeout, stamp.GenJust(time.Duration(0))).Get(s)),
		WorkflowRunTimeout:       durationpb.New(cmp.Or(w.WorkflowRunTimeout, stamp.GenJust(time.Duration(0))).Get(s)),
		WorkflowTaskTimeout:      durationpb.New(cmp.Or(w.WorkflowTaskTimeout, stamp.GenJust(time.Duration(0))).Get(s)),
		Identity:                 w.GetActor().GetID(),
		RequestId:                string(w.RequestId.Get(s)),
	}
}
