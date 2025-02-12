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
	. "go.temporal.io/api/enums/v1"
	. "go.temporal.io/api/workflowservice/v1"
)

type WorkflowRun struct {
	Model[WorkflowRun, Root]
}

func (u *WorkflowRun) IsAlive() bool {
	return true
}

type WorkflowUpdate struct {
	Model[WorkflowUpdate, *WorkflowRun]
}

// ==== Vars

type (
	ID       = Var[relativeID] // for all models
	ParentID = Var[absoluteID] // for all models

	UpdateID  = Var[string]
	WaitStage = SetVar[UpdateWorkflowExecutionLifecycleStage]
)

// ==== Props

func (u *WorkflowUpdate) IsAlive(
// waitStage WaitStage,
) bool {
	return true
}

// ==== Spec

func (u *WorkflowUpdate) Correctness() Spec {
	panic("not implemented")
}

// ==== Observers

func (u *WorkflowUpdate) OnUpdateRequest(
	req Request[*UpdateWorkflowExecutionRequest],
) bool {
	// (1) check if this is the right update
	if false {
		return false
	}

	// (2) update the vars
	// ...

	Set[ID](u, "test")
	Set[ParentID](u, "test")

	return true
}

func (u *WorkflowUpdate) OnUpdateResponse(
	resp Response[*UpdateWorkflowExecutionRequest, *UpdateWorkflowExecutionResponse],
) bool {
	// (1) check if this is the right update
	if false {
		return false
	}

	// (2) update the vars
	// ...

	return true
}

// ==== Generators

func (u *WorkflowUpdate) GenerateRequest(
	workflowRun Gen[WorkflowRun],
	updateID Gen[UpdateID],
	waitStage Gen[WaitStage],
	// ...
) Gen[*UpdateWorkflowExecutionRequest] {
	panic("not implemented")
}
