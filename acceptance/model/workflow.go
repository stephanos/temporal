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
	"cmp"

	"go.temporal.io/server/common/log"
	"go.temporal.io/server/common/log/tag"
	"go.temporal.io/server/common/rpc/interceptor/logtags"
	"go.temporal.io/server/common/tasktoken"
	"go.temporal.io/server/common/testing/stamp"
	"google.golang.org/protobuf/proto"
)

type (
	Workflow struct {
		stamp.Model[*Workflow]
		stamp.Scope[*Namespace]
		RunID string
		tags  *logtags.WorkflowTags
	}
	RunID string
)

// ==== Identity Handlers

func (w *Workflow) IdRequest(req RequestMsg, requestPath Method) (id stamp.ID) {
	id, w.RunID = w.extractIDs(req, requestPath)
	return id
}

func (w *Workflow) IdResponse(req RequestMsg, resp ResponseMsg, requestPath Method) (id stamp.ID) {
	id, w.RunID = w.extractIDs(req, requestPath)
	if id == "" || w.RunID == "" {
		respWorkflowID, respRunID := w.extractIDs(resp, requestPath)
		id = cmp.Or(id, respWorkflowID)
		w.RunID = cmp.Or(w.RunID, respRunID)
	}
	return
}

func (w *Workflow) extractIDs(msg proto.Message, requestPath Method) (workflowID stamp.ID, runID string) {
	if w.tags == nil {
		w.tags = logtags.NewWorkflowTags(tasktoken.NewSerializer(), log.NewTestLogger())
	}
	for _, logTag := range w.tags.Extract(msg, string(requestPath)) {
		switch logTag.Key() {
		case tag.WorkflowIDKey:
			workflowID = stamp.ID(logTag.Value().(string))
		case tag.WorkflowRunIDKey:
			runID = logTag.Value().(string)
		}
	}
	return
}

// ==== Action handlers

// ==== Rules & Properties

func (w *Workflow) TestCorrect() stamp.Rule {
	return stamp.RuleExpr(
		w,
		`true`,
	)
}

func (w *Workflow) HasBeenCreated() stamp.Prop[bool] {
	return stamp.PropExpr[bool](
		w,
		`true`,
	)
}
