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
	commonpb "go.temporal.io/api/common/v1"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/server/common/testing/testvars"
)

type SignalWorkflowRequestBuilder struct {
	v *workflowservice.SignalWorkflowExecutionRequest
}

func SignalWorkflowRequest(tv *testvars.TestVars) *SignalWorkflowRequestBuilder {
	return &SignalWorkflowRequestBuilder{
		&workflowservice.SignalWorkflowExecutionRequest{
			RequestId:         tv.Any().String(),
			Namespace:         tv.Namespace().Name().String(),
			WorkflowExecution: &commonpb.WorkflowExecution{WorkflowId: tv.WorkflowID(), RunId: tv.RunID()},
			SignalName:        tv.String(),
			Identity:          tv.WorkerIdentity(),
		},
	}
}

func (s *SignalWorkflowRequestBuilder) WorkflowExecution(we *commonpb.WorkflowExecution) *SignalWorkflowRequestBuilder {
	r := &SignalWorkflowRequestBuilder{v: s.Build()}
	r.v.WorkflowExecution = we
	return r
}

func (s *SignalWorkflowRequestBuilder) Namespace(ns string) *SignalWorkflowRequestBuilder {
	r := &SignalWorkflowRequestBuilder{v: s.Build()}
	r.v.Namespace = ns
	return r
}

func (s *SignalWorkflowRequestBuilder) SignalName(n string) *SignalWorkflowRequestBuilder {
	r := &SignalWorkflowRequestBuilder{v: s.Build()}
	r.v.SignalName = n
	return r
}

func (s *SignalWorkflowRequestBuilder) Header(h *commonpb.Header) *SignalWorkflowRequestBuilder {
	r := &SignalWorkflowRequestBuilder{v: s.Build()}
	r.v.Header = h
	return r
}

func (s *SignalWorkflowRequestBuilder) Build() *workflowservice.SignalWorkflowExecutionRequest {
	return s.v
}

func (s *SignalWorkflowRequestBuilder) Input(p *commonpb.Payloads) *SignalWorkflowRequestBuilder {
	r := &SignalWorkflowRequestBuilder{v: s.Build()}
	r.v.Input = p
	return r
}
