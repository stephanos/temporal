//// The MIT License
////
//// Copyright (c) 2025 Temporal Technologies Inc.  All rights reserved.
////
//// Copyright (c) 2020 Uber Technologies, Inc.
////
//// Permission is hereby granted, free of charge, to any person obtaining a copy
//// of this software and associated documentation files (the "Software"), to deal
//// in the Software without restriction, including without limitation the rights
//// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
//// copies of the Software, and to permit persons to whom the Software is
//// furnished to do so, subject to the following conditions:
////
//// The above copyright notice and this permission notice shall be included in
//// all copies or substantial portions of the Software.
////
//// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
//// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
//// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
//// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
//// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
//// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
//// THE SOFTWARE.

package older

//import (
//	"fmt"
//
//	enumspb "go.temporal.io/api/enums/v1"
//	"go.temporal.io/api/serviceerror"
//	updatepb "go.temporal.io/api/update/v1"
//	. "go.temporal.io/api/workflowservice/v1"
//	. "go.temporal.io/server/common/proptest2"
//)
//
//var (
//	stateAdmitted  = StateAlive.SubState("admitted")
//	stateAccepted  = StateAlive.SubState("accepted")
//	stateRejected  = StateDead.SubState("rejected")
//	stateCompleted = StateDead.SubState("completed")
//)
//
//var (
//	triggerUpdateAdmitted  Trigger = "admitted"
//	triggerUpdateAccepted  Trigger = "accepted"
//	triggerUpdateRejected  Trigger = "rejected"
//	triggerUpdateCompleted Trigger = "completed"
//)
//
//type WorkflowUpdate struct {
//	Model[*WorkflowUpdate] /// TODO: merge Model and Parent
//	Parent[*Workflow]
//
//	outcome *updatepb.Outcome // TODO: there are no fields; everything is a property
//}
//
//
////var WorkflowUpdateDeduped = NewValProp("workflowUpdateDeduped", func(u *WorkflowUpdate) {})
//
//func newWorkflowUpdate(
//	wf *Workflow,
//	initEvt *UpdateWorkflowExecutionRequest,
//) *WorkflowUpdate {
//	return GetOrCreateModel[*WorkflowUpdate](
//		initEvt,
//		&WorkflowUpdate{
//			Parent: NewParent(wf),
//		},
//
//		// TODO: send events through Root to children; if a filter rejects the input, don't propagate it any further
//		// TODO: input is Req[*type] and Resp[*type,*serviceErrType]
//		WithInputFilter(
//			func(u *WorkflowUpdate, evt *UpdateWorkflowExecutionRequest) bool {
//				return true
//			}),
//
//		WithInputHandler(
//			func(u *WorkflowUpdate, evt *UpdateWorkflowExecutionRequest) error {
//				u.MustSetID(evt.Request.Meta.UpdateId)
//				return nil
//			}),
//		WithInputHandler(
//			func(u *WorkflowUpdate, evt *UpdateWorkflowExecutionResponse) error {
//				switch evt.Stage {
//				case enumspb.UPDATE_WORKFLOW_EXECUTION_LIFECYCLE_STAGE_ADMITTED:
//					return u.Fire(triggerUpdateAdmitted)
//				case enumspb.UPDATE_WORKFLOW_EXECUTION_LIFECYCLE_STAGE_ACCEPTED:
//					return u.Fire(triggerUpdateAccepted)
//				case enumspb.UPDATE_WORKFLOW_EXECUTION_LIFECYCLE_STAGE_COMPLETED:
//					if u.CurrentState() == stateCompleted {
//						if u.GetID() == evt.UpdateRef.UpdateId {
//							return nil
//						}
//					}
//
//					u.outcome = evt.Outcome
//					return u.Fire(triggerUpdateCompleted)
//				default:
//					return fmt.Errorf("unexpected stage: %v", evt.Stage)
//				}
//			}),
//		WithInputHandler(
//			func(u *WorkflowUpdate, evt *serviceerror.NotFound) error {
//				return nil
//			}),
//		WithInputHandler(
//			func(u *WorkflowUpdate, evt *serviceerror.ResourceExhausted) error {
//				return nil
//			}),
//		WithInputHandler(
//			func(u *WorkflowUpdate, evt *serviceerror.WorkflowNotReady) error {
//				return nil
//			}),
//		WithInputHandler(
//			func(u *WorkflowUpdate, evt *serviceerror.DeadlineExceeded) error {
//				return nil
//			}),
//
//		// TODO
//		//WithState[*WorkflowUpdate](AnyState).
//		//	On(func(u *WorkflowUpdate, req *workflowservice.UpdateWorkflowExecutionRequest) {
//		//		u.MustSetID(req.Request.Meta.UpdateId)
//		//	}),
//		//WithState[*WorkflowUpdate](InitState),
//		WithState[*WorkflowUpdate](StateDraft).
//			Allow(stateAdmitted, triggerUpdateAdmitted).
//			Allow(StateInvalid, TriggerInvalidated).
//			Allow(stateAccepted, triggerUpdateAccepted).
//			Allow(stateRejected, triggerUpdateRejected).
//			Allow(stateCompleted, triggerUpdateCompleted),
//		WithState[*WorkflowUpdate](StateAlive).
//			Allow(stateAccepted, triggerUpdateAccepted).
//			Allow(stateRejected, triggerUpdateRejected).
//			Allow(stateCompleted, triggerUpdateCompleted),
//		WithState[*WorkflowUpdate](stateAdmitted).
//			Allow(stateAccepted, triggerUpdateAccepted).
//			Allow(stateRejected, triggerUpdateRejected).
//			Allow(stateCompleted, triggerUpdateCompleted),
//		WithState[*WorkflowUpdate](stateAccepted).
//			Allow(stateCompleted, triggerUpdateCompleted),
//		WithState[*WorkflowUpdate](stateRejected),
//		WithState[*WorkflowUpdate](stateCompleted),
//	)
//}
