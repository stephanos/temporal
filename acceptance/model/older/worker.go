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

package older

import (
	"github.com/pborman/uuid"
	. "go.temporal.io/server/common/proptest1"
)

type Worker Model[Worker]

var (
	// ==== ModelType

	workerModel = NewModelType[Worker]()

	// ==== States

	workerPreparing = NewState("Preparing")
	workerStarting  = NewState("Starting")

	// ==== Vars

	// TODO: PollerType - Workflow, Activity, Nexus
	WorkerIdentity = NewVar[string]("WorkerIdentity", Default(GenId("id")))

	workerStartInputs = []ModelVar{
		//WorkflowType,
		//ServerAddr,
		//nsInput,
	}

	// ==== Events

	sdkWorkerStarting = NewInitHandler("Starting",
		func(w Worker, _ any) ID {
			//sdkClient, err := sdkclient.Dial(sdkclient.Options{
			//	HostPort:  ServerAddr.Get(w),
			//	Namespace: NamespaceName.Get(nsInput.Get(w)),
			//})
			//if err != nil {
			//	panic(err)
			//}
			//
			//w.worker = worker.New(sdkClient, TaskQueueName.Get(w), worker.Options{})
			//w.worker.RegisterWorkflow(WorkflowDef.Get(w))
			//
			//err = w.worker.Start()
			return uuid.New()
		})

	// ==== Actions

	StartLocalWorker = NewInitAction[Worker]("StartLocalWorker",
		workerStartInputs,
		func(c Readonly[Worker]) ID {
			// TODO
			return uuid.New()
		})

	StartSDKWorker = NewInitAction[Worker]("StartSDKWorker",
		workerStartInputs,
		func(c Readonly[Worker]) ID {
			// TODO
			return uuid.New()
		})

	StartSpyWorker = NewInitAction[Worker]("StartSpyWorker",
		workerStartInputs,
		func(c Readonly[Worker]) ID {
			// TODO
			return uuid.New()
		})
)

//type LocalWorker struct {
//	*ActorDef[LocalWorker]
//	Identity
//	updateutils.UpdateUtils
//	*taskpoller.TaskPoller
//}
//
//func NewLocalWorker(
//	params ...ModelParam,
//) *LocalWorker {
//	res := &LocalWorker{}
//	res.ActorDef = &ActorDef[LocalWorker]{
//		ModelType: NewModelType[LocalWorker](
//			res,
//			workerPreparing,
//			[]Transition{},
//			[]ModelVar{},
//			params,
//		),
//	}
//	return res
//
//	//res.UpdateUtils = updateutils.New(res.Run())
//	//res.TaskPoller = taskpoller.New(
//	//	res.Run(),
//	//	workflowServiceClient.Get(cluster.Get(ns.Get(taskQueue.Get(res)))),
//	//	NamespaceName.Get(ns.Get(taskQueue.Get(res))).String())
//}
//
//func (w *LocalWorker) ID() string {
//	return WorkerIdentity.Get(w)
//}

//func (w *LocalWorker) TaskQueue() *taskqueuepb.TaskQueue {
//	return &taskqueuepb.TaskQueue{Name: TaskQueueName.Get(taskQueue.Get(w))}
//}
//
//func (w *LocalWorker) WorkerIdentity() string {
//	return WorkerIdentity.Get(w)
//}
//
//func (w *LocalWorker) MessageID() string {
//	return "<message_id>"
//}

//func (w *LocalWorker) DrainWorkflowTask() {
//	_, err := w.PollAndHandleWorkflowTask(w, taskpoller.DrainWorkflowTask)
//	w.NoError(err)
//}
//
//func (w *LocalWorker) AcceptAndCompleteUpdate(upd *WorkflowUpdate) {
//	_, err := w.PollAndHandleWorkflowTask(
//		w,
//		func(task *workflowservice.PollWorkflowTaskQueueResponse) (*workflowservice.RespondWorkflowTaskCompletedRequest, error) {
//			return &workflowservice.RespondWorkflowTaskCompletedRequest{
//				Commands: w.UpdateAcceptCompleteCommands(w),
//				Messages: w.UpdateAcceptCompleteMessages(w, task.Messages[0]),
//			}
//		})
//	w.NoError(err)
//}
//
//func (w *LocalWorker) CompleteUpdate(upd *WorkflowUpdate) {
//	_, err := w.PollAndHandleWorkflowTask(
//		w,
//		func(task *workflowservice.PollWorkflowTaskQueueResponse) (*workflowservice.RespondWorkflowTaskCompletedRequest, error) {
//			return &workflowservice.RespondWorkflowTaskCompletedRequest{
//				Commands: w.UpdateCompleteCommands(w),
//				Messages: w.UpdateCompleteMessages(w, task.Messages[0]),
//			}, nil
//		})
//	w.NoError(err)
//}
//
//func (w *LocalWorker) AcceptUpdate(upd *WorkflowUpdate) {
//	_, err := w.PollAndHandleWorkflowTask(
//		w,
//		func(task *workflowservice.PollWorkflowTaskQueueResponse) (*workflowservice.RespondWorkflowTaskCompletedRequest, error) {
//			return &workflowservice.RespondWorkflowTaskCompletedRequest{
//				Commands: w.UpdateAcceptCommands(w),
//				Messages: w.UpdateAcceptMessages(w, task.Messages[0]),
//			}, nil
//		})
//	w.NoError(err)
//}
//
//func (w *LocalWorker) RejectUpdate(upd *WorkflowUpdate) {
//	_, err := w.PollAndHandleWorkflowTask(
//		w,
//		func(task *workflowservice.PollWorkflowTaskQueueResponse) (*workflowservice.RespondWorkflowTaskCompletedRequest, error) {
//			return &workflowservice.RespondWorkflowTaskCompletedRequest{
//				Messages: w.UpdateRejectMessages(w, task.Messages[0]),
//			}, nil
//		})
//	w.NoError(err)
//}
