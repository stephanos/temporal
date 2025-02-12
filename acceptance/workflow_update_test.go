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

package acceptance

import (
	"testing"

	"go.temporal.io/sdk/workflow"
	. "go.temporal.io/server/acceptance/propmodel"
)

//type UpdateSpec struct {
//	server Handle[Server]
//	client Handle[Client]
//	worker Handle[Worker]
//	ns     Handle[Namespace]
//	tq     Handle[TaskQueue]
//}
//
//func (s *UpdateSpec) Setup(run Run) {
//	s.server = StartLocalServer()
//	s.client = StartSDKClient()
//	s.worker = StartSDKWorker()
//	//AwaitAll(s.server, s.client, s.worker)
//
//	//ns := RegisterNamespace(r, NamespaceName.Val(s.Namespace().String()))
//	//tq := NewTaskQueue(ns)
//	//UpdateWithStartSpec.Example(&UpdateSpec{
//	//	Env:    env,
//	//	Run:    r,
//	//	ns:     ns,
//	//	tq:     tq,
//	//	s:      NewCluster(r),
//	//	client: NewSDKClient(ns, ServerAddr.Val(s.FrontendGRPCAddress())),
//	//	worker: NewSDKWorker(ns, ServerAddr.Val(s.FrontendGRPCAddress()), WorkflowDef.Val(WorkflowWithUpdateHandlerDef)),
//	//})
//}
//
//func (s *UpdateSpec) Teardown(run Run) {}
//
//func TestStartAndUpdate(t *testing.T) {
//	RunExample[*UpdateSpec](t, func(spec *UpdateSpec) {
//		// TODO: can I learn from formal verification here?
//		//wf := StartWorkflow(
//		//	//env.tq,
//		//	WorkflowDef.Val(WorkflowWithUpdateHandlerDef))
//		//env.worker.TickWorkflow()
//		//up := StartWorkflowUpdate(wf)
//		//env.worker.TickWorkflow()
//		//up.Await()
//		//up.UpdateWorkflow()
//		//wf.Await() // waits for any terminal state by default
//	})
//}

//TestUpdateWorkflow_TimeoutWithRetryAfterUpdateAdmitted = newSpecCtx(
//	"UpdateWithStartSpec",
//	func(env *UpdateSpec) {
//		wf := StartWorkflow(
//			env.tq,
//			WorkflowRunTimeout.Val(1*time.Second),
//			WorkflowDef.Val(WorkflowWithUpdateHandlerDef))
//		up := StartWorkflowUpdate(wf)
//		up.Await(Admitted, Timeout(1*time.Second))
//		env.worker.TickWorkflow()
//	})

var WorkflowWithUpdateHandlerDef = WorkflowDefinition{
	Type: "workflowFn",
	Func: func(ctx workflow.Context) error {
		var unblocked bool
		if err := workflow.SetUpdateHandler(ctx,
			"updateHandler",
			func(ctx workflow.Context, arg string) error {
				unblocked = true
				return nil
			},
		); err != nil {
			return err
		}
		return workflow.Await(ctx, func() bool { return unblocked })
	},
}

// TODO: fuzzer "driver"
func TestFuzz(t *testing.T) {
	// TODO
}

//func TestGenerate(t *testing.T) {
//	r := NewCodegenRun("workflow_update_gen_test.go", "proptests")
//	ns := NewNamespace(r, NamespaceName.Val("propspec"))
//	tq := NewTaskQueue(ns)
//	UpdateWithStartSpec.Generate(&UpdateSpec{
//		Run:    r,
//		ns:     ns,
//		tq:     tq,
//		client: NewSpyClient(r),
//		worker: NewWorkerSpy(r),
//	})
//}

//func (s *UpdateWithStartSuite) TestUpdateWithStartWorkflow_Dedupe() {
//	s.Verify(func(ns *Namespace) {
//		//tq := NewTaskQueue(ns)
//		//w := NewLocalWorker(tq)
//		//
//		//wf := NewWorkflowExecution(tq, Any(RequestID))
//		//upd := NewWorkflowExecutionUpdate(wf, GenPick(WaitStageSet), UpdateID("dedupe"))
//		//startUpdateWithStart(wf, upd)
//		//
//		//w.AcceptAndCompleteUpdate(upd)
//		//upd.AwaitCompletion()
//	})
//
//	// TODO: distinguish between parameters that are random but don't change the behavior - and the ones that do
//	// TODO: don't mark whether anything can be run in parallel - mark what's a shared resource/non-parallel resource
//
//	// verify has no *meaningful* randomness, every parameter needs to be specified
//	//spec.Verify(s.T(), "retrying", func(r *propgen.Run) propgen.Error {
//	//	// update-with-start workflow
//	//	//		with fixed request_id
//	//	//		with fixed update_id
//	//	//	 	with any of the conflict policies
//	//	//		with wait stage completed or accepted
//	//	// poll result
//	//	// verify it's the same runID
//	//	// verify it's the same outcome
//	//	// poll again and see same result
//	//	//start := StartWorkflowRequest(Set(RequestID()))
//	//	//upd := UpdateWorkflowRequest(Set(UpdateID()))
//	//	//w, u := UpdateWithStart(start, upd) // this is a transition
//	//	//LocalWorker(AcceptAndCompleteUpdate(u))  // this is a transition
//	//	//UpdateWithStart(start, upd)         // this is a transition (again)
//	//	// implicitly verify there's no WFT
//	//	// implicitly verify res1 and res2 are the same
//	//	// TODO: implicitly poll at any time to verify Update poll returns the same result
//	//	return nil
//	//})
//}
