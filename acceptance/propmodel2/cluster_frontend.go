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
	"go.temporal.io/api/workflowservice/v1"
	. "go.temporal.io/server/common/proptest2"
)

type (
	Frontend struct {
		Model[*Frontend]
		Parent[*Cluster]
	}
)

func GetFrontend(
	cluster *Cluster,
	info *handlerInfoData,
) *Frontend {
	extractMethodsFromGRPCService((*workflowservice.WorkflowServiceServer)(nil))

	return GetOrCreateModel[*Frontend](
		info,
		&Frontend{
			Parent: NewParent(cluster),
		},
		WithInputHandler(
			func(f *Frontend, evt *handlerInfoData) error {
				f.MustSetID(evt.identity)
				return nil
			}))
}

func (f *Frontend) handle(
	req any,
) ResponseHandlerProvider {
	// TODO: send req to all models and let them figure out if it's them (parents first)

	switch t := req.(type) {
	case *workflowservice.RegisterNamespaceRequest:
		ns := newNamespace(f.Parent, t)
		return ns
	case *workflowservice.StartWorkflowExecutionRequest:
		ns := Get[*Namespace](&f.Parent, t.Namespace)
		return newWorkflow(ns, t)
	case *workflowservice.UpdateWorkflowExecutionRequest:
		ns := Get[*Namespace](&f.Parent, t.Namespace)
		wf := Get[*Workflow](ns, t.WorkflowExecution.WorkflowId)
		return newWorkflowUpdate(wf, t)
	case *workflowservice.SignalWorkflowExecutionRequest:
		ns := Get[*Namespace](&f.Parent, t.Namespace)
		wf := Get[*Workflow](ns, t.WorkflowExecution.WorkflowId)
		return newWorkflowSignal(wf, t)
	default:
		return nil
	}
}
