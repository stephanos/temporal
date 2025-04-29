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

package testenv

import (
	"fmt"

	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/server/common/testing/stamp"
	"go.temporal.io/server/tests/acceptance/model"
	"go.temporal.io/server/tests/acceptance/testenv/action"
	"google.golang.org/protobuf/proto"
)

type WorkflowWorker struct {
	stamp.ActorModel[*model.WorkflowWorker]
	client workflowservice.WorkflowServiceClient
}

func newWorkflowWorker(
	c *Cluster,
	tq *model.TaskQueue,
) *WorkflowWorker {
	return &WorkflowWorker{
		ActorModel: stamp.NewActorModel(stamp.Act(c, action.CreateWorkflowWorker{TaskQueue: tq})),
		client:     c.tbase.FrontendClient(),
	}
}

func (w *WorkflowWorker) OnAction(params stamp.ActionParams) error {
	switch t := params.Payload.(type) {
	case proto.Message:
		_, err := issueWorkflowRPC(w.Context(), w.client, t, params.ActID)
		return err
	default:
		panic(fmt.Sprintf("unhandled action %T", t))
	}
	return nil
}
