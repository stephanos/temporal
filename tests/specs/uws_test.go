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

package specs

import (
	"testing"

	"github.com/stretchr/testify/suite"
	. "go.temporal.io/server/common/testing/propcheck"
	"go.temporal.io/server/common/testing/propspec"
	. "go.temporal.io/server/tests/propactor"
	. "go.temporal.io/server/tests/propmodel"
	"go.temporal.io/server/tests/testcore"
)

var (
	UpdateWithStartSpec = propspec.New(
		func(r Run) {
			_ = NewWorker(r)
			_ = NewClient(r)
			ns := NewNamespace(r)
			tq := NewTaskQueue(ns)
			wf := NewWorkflowExecution(tq)
			upd := NewWorkflowExecutionUpdate(wf)
			for range Repeat(ns, 3) {
				//StartUpdateWithStart(upd)(c)
			}
			//w.AcceptAndCompleteUpdate(upd)
			upd.AwaitCompletion()
		})
)

func TestGenerate(t *testing.T) {
	run := NewGenerateRun()
	//c := NewCluster(run)
	//ns := NewNamespace(c, namespace.Name("propspec"))
	UpdateWithStartSpec.Generate(run)
}

func TestExample(t *testing.T) {
	// HACK
	s := testcore.FunctionalTestBase{Suite: suite.Suite{}}
	s.Suite.SetT(t)
	s.SetupSuite()
	s.SetupTest()

	run := NewExampleRun(t)
	//c := NewCluster(run) // s.FrontendClient()
	//ns := NewNamespace(c, s.Namespace())
	UpdateWithStartSpec.Example(run)
}

func TestFuzz(t *testing.T) {
	// TODO
}

//func CheckUpdateWithStartWorkflow(ns *Namespace) {
//	tq := NewTaskQueue(ns)
//	w := NewLocalWorker(tq)
//	wf := NewWorkflowExecution(tq)
//	upd := NewWorkflowExecutionUpdate(wf)
//	for range Repeat(ns, 3) {
//		StartUpdateWithStart(wf, upd)
//	}
//	w.AcceptAndCompleteUpdate(upd)
//	upd.AwaitCompletion()
//}

//func (s *UpdateWithStartSuite) TestUpdateWithStartWorkflow_Dedupe() {
//	s.Verify(func(ns *Namespace) {
//		//tq := NewTaskQueue(ns)
//		//w := NewLocalWorker(tq)
//		//
//		//wf := NewWorkflowExecution(tq, Any(RequestID))
//		//upd := NewWorkflowExecutionUpdate(wf, GenPick(WaitStageSet), UpdateID("dedupe"))
//		//StartUpdateWithStart(wf, upd)
//		//
//		//w.AcceptAndCompleteUpdate(upd)
//		//upd.AwaitCompletion()
//	})
//
//	// TODO: distinguish between parameters that are random but don't change the behavior - and the ones that do
//	// TODO: don't mark whether anything can be run in parallel - mark what's a shared resource/non-parallel resource
//
//	// verify has no *meaningful* randomness, every parameter needs to be specified
//	//spec.Verify(s.T(), "retrying", func(r *propcheck.Run) propcheck.Error {
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
