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

package propactor

import (
	. "go.temporal.io/server/common/testing/propactor"
	"go.temporal.io/server/common/testing/propcheck"
	. "go.temporal.io/server/common/testing/propmodel"
	"go.temporal.io/server/common/testing/taskpoller"
	"go.temporal.io/server/common/testing/updateutils"
)

var (
// _ taskpoller.TaskContext = &LocalWorker{}
)

// ==== Inputs

var (
	WorkerIdentity = NewInput[string](Default(propcheck.GenId("id")))
)

type LocalWorker struct {
	Actor
	updateutils.UpdateUtils
	*taskpoller.TaskPoller
}

func NewLocalWorker(
	params ...any,
) *LocalWorker {
	res := &LocalWorker{}
	//res.UpdateUtils = updateutils.New(res.Run())
	//res.TaskPoller = taskpoller.New(
	//	res.Run(),
	//	workflowServiceClient.Get(cluster.Get(ns.Get(taskQueue.Get(res)))),
	//	NamespaceName.Get(ns.Get(taskQueue.Get(res))).String())
	return res
}

//func (w *LocalWorker) ID() string {
//	return w.WorkerIdentity()
//}
//
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
//			}, nil
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
