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

package acceptance

import (
	"testing"

	. "go.temporal.io/server/acceptance/env"
	. "go.temporal.io/server/acceptance/model"
	. "go.temporal.io/server/acceptance/trigger"
	. "go.temporal.io/server/common/testing/stamp"
)

func TestWorkflowUpdate(t *testing.T) {
	t.Parallel()
	cluster := NewCluster(t)

	DescribeScenario(t, func(s *Scenario) {
		tq, client, worker := simpleSetup(cluster)
		wf := Trigger(client, StartWorkflowExecutionRequest{TaskQueue: tq})
		Trigger(worker, RespondToWorkflowTask{WorkflowWorker: worker, Workflow: wf})
		s.WaitFor(wf.HasBeenCreated)
	})
}

func simpleSetup(cluster *Cluster) (*TaskQueue, *WorkflowClient, *WorkflowWorker) {
	ns := Trigger(cluster, CreateNamespace{})
	tq := Trigger(cluster, CreateTaskQueue{Namespace: ns})
	client := Trigger(cluster, CreateWorkflowClient{TaskQueue: tq})
	worker := Trigger(cluster, CreateWorkflowWorker{TaskQueue: tq})
	return tq, client, worker
}

// verify has no *meaningful* randomness, every parameter needs to be specified
//spec.Verify(s.T(), "retrying", func(r *propgen.Run) propgen.Error {
// update-with-start workflow
//		with fixed request_id
//		with fixed update_id
//	 	with any of the conflict policies
//		with wait stage completed or accepted
// poll result
// verify it's the same runID
// verify it's the same outcome
// poll again and see same result
//start := StartWorkflowRequest(Set(RequestID()))
//upd := UpdateWorkflowRequest(Set(UpdateID()))
//w, u := UpdateWithStart(start, upd) // this is a transition
//LocalWorkflowWorker(AcceptAndCompleteUpdate(u))  // this is a transition
//UpdateWithStart(start, upd)         // this is a transition (again)
// implicitly verify there's no WFT
// implicitly verify res1 and res2 are the same
// TODO: implicitly poll at any time to verify Update poll returns the same result
//return nil
//})
