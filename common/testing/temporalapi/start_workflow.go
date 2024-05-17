// The MIT License
//
// Copyright (c) 2022 Temporal Technologies Inc.  All rights reserved.
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

package temporalapi

import (
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/server/common"
	"go.temporal.io/server/common/testing/testvars"
)

type StartWorkflowRequestBuilder struct {
	v *workflowservice.StartWorkflowExecutionRequest
}

func StartWorkflowRequest(tv *testvars.TestVars) *StartWorkflowRequestBuilder {
	return &StartWorkflowRequestBuilder{
		&workflowservice.StartWorkflowExecutionRequest{
			RequestId:    tv.Any().String(),
			Namespace:    tv.Namespace().Name().String(),
			WorkflowId:   tv.WorkflowID(),
			WorkflowType: tv.WorkflowType(),
			TaskQueue:    tv.TaskQueue(),
		},
	}
}

func (s *StartWorkflowRequestBuilder) Namespace(ns string) *StartWorkflowRequestBuilder {
	r := &StartWorkflowRequestBuilder{v: s.Build()}
	r.v.Namespace = ns
	return r
}

func (s *StartWorkflowRequestBuilder) Build() *workflowservice.StartWorkflowExecutionRequest {
	return common.CloneProto(s.v)
}
