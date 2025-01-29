package testcore

import (
	"errors"
	"fmt"
	"unicode"

	"github.com/stretchr/testify/require"
	commonpb "go.temporal.io/api/common/v1"
	enumspb "go.temporal.io/api/enums/v1"
	"go.temporal.io/api/serviceerror"
	"go.temporal.io/api/taskqueue/v1"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/server/common/testing/propcheck"
	"pgregory.net/rapid"
)

var (
	// Cluster
	Cluster        = propcheck.NewModel("cluster")
	FrontendClient = propcheck.NewVar(Cluster, "frontend-client", panicGen[workflowservice.WorkflowServiceClient]("frontend client not set"))

	// Namespace
	Namespace     = propcheck.NewModel("namespace")
	NamespaceName = propcheck.NewVar(Namespace, "name", panicGen[string]("namespace name not set")) // TODO

	// Worker
	Worker   = propcheck.NewModel("worker")
	WorkerID = propcheck.NewVar(Worker, "id", rapid.Custom(func(t *rapid.T) string {
		return idGen("worker").Draw(t, "")
	}))

	// Taskqueue
	Taskqueue = propcheck.NewModel("taskqueue")

	// Workflow
	Workflow     = propcheck.NewModel("workflow")
	WorkflowID   = propcheck.NewVar(Workflow, "id", idGen("id"))
	WorkflowType = propcheck.NewVar(Workflow, "type", rapid.Custom(func(t *rapid.T) *commonpb.WorkflowType {
		return &commonpb.WorkflowType{
			Name: idGen("type").Draw(t, ""),
		}
	}))
	WorkflowTaskQueue = propcheck.NewVar(Workflow, "taskqueue", rapid.Custom(func(t *rapid.T) *taskqueue.TaskQueue {
		return &taskqueue.TaskQueue{
			Name: idGen("taskqueue").Draw(t, ""),
		}
	}))
	WorkflowState     = propcheck.NewVar(Workflow, "state", rapid.Just(enumspb.WORKFLOW_EXECUTION_STATUS_UNSPECIFIED))
	WorkflowStateProp = propcheck.NewProp(Workflow, "state-check",
		func(run *propcheck.Run) propcheck.Error {
			// TODO
			return nil
		})
	WorkflowStart = propcheck.NewTransition(Workflow, "start workflow",
		func(run *propcheck.Run) propcheck.Error {
			startReq := &workflowservice.StartWorkflowExecutionRequest{
				Namespace:                NamespaceName.Get(run),
				WorkflowId:               WorkflowID.Get(run),
				WorkflowType:             WorkflowType.Get(run),
				TaskQueue:                WorkflowTaskQueue.Get(run),
				Identity:                 WorkerID.Get(run),
				WorkflowIdReusePolicy:    WorkflowReusePolicySet.OneOf(run),
				WorkflowIdConflictPolicy: WorkflowConflictPolicySet.OneOf(run),
			}
			_, err := FrontendClient.Get(run).StartWorkflowExecution(NewContext(), startReq)

			var alreadyStartedErr *serviceerror.WorkflowExecutionAlreadyStarted
			switch {
			case errors.As(err, &alreadyStartedErr):
				if startReq.WorkflowIdConflictPolicy != enumspb.WORKFLOW_ID_CONFLICT_POLICY_FAIL {
					return run.Fail(fmt.Errorf("received unexpected error: %w", err))
				}
			case err != nil:
				return run.Fail(err)
			}

			WorkflowState.Set(run, enumspb.WORKFLOW_EXECUTION_STATUS_RUNNING)
			return nil
		})
	WorkflowComplete = propcheck.NewTransition(Workflow, "start workflow",
		func(run *propcheck.Run) propcheck.Error {
			if WorkflowState.Get(run) != enumspb.WORKFLOW_EXECUTION_STATUS_RUNNING {
				return run.Invalid("workflow is not running")
			}
			// TODO
			return nil
		})
	WorkflowTerminate = propcheck.NewTransition(Workflow, "terminate workflow",
		func(run *propcheck.Run) propcheck.Error {
			if WorkflowState.Get(run) != enumspb.WORKFLOW_EXECUTION_STATUS_RUNNING {
				// TODO: sometimes actually continue
				return run.Invalid("workflow is not running")
			}

			_, err := FrontendClient.Get(run).TerminateWorkflowExecution(
				NewContext(),
				&workflowservice.TerminateWorkflowExecutionRequest{
					Namespace: NamespaceName.Get(run),
					WorkflowExecution: &commonpb.WorkflowExecution{
						WorkflowId: WorkflowID.Get(run),
					},
					Reason: idGen("reason").Draw(run.T, ""),
				})
			require.NoError(run.T, err)
			return nil
		})
	WorkflowConflictPolicySet = propcheck.NewSet(
		"workflow conflicy policy",
		map[enumspb.WorkflowIdConflictPolicy]int{
			enumspb.WORKFLOW_ID_CONFLICT_POLICY_UNSPECIFIED:        25,
			enumspb.WORKFLOW_ID_CONFLICT_POLICY_TERMINATE_EXISTING: 25,
			enumspb.WORKFLOW_ID_CONFLICT_POLICY_USE_EXISTING:       25,
			enumspb.WORKFLOW_ID_CONFLICT_POLICY_FAIL:               25,
		},
	)
	WorkflowReusePolicySet = propcheck.NewSet(
		"workflow reuse policy",
		map[enumspb.WorkflowIdReusePolicy]int{
			enumspb.WORKFLOW_ID_REUSE_POLICY_UNSPECIFIED:                 20,
			enumspb.WORKFLOW_ID_REUSE_POLICY_ALLOW_DUPLICATE:             20,
			enumspb.WORKFLOW_ID_REUSE_POLICY_ALLOW_DUPLICATE_FAILED_ONLY: 20,
			enumspb.WORKFLOW_ID_REUSE_POLICY_REJECT_DUPLICATE:            20,
			enumspb.WORKFLOW_ID_REUSE_POLICY_TERMINATE_IF_RUNNING:        20,
		},
	)
)

func idGen(prefix string) *rapid.Generator[string] {
	return rapid.Custom(func(t *rapid.T) string {
		return rapid.Just(prefix).Draw(t, "") +
			rapid.StringOfN(rapid.RuneFrom(nil, unicode.ASCII_Hex_Digit), 12, 12, -1).Draw(t, "")
	})
}

func panicGen[T any](msg string) *rapid.Generator[T] {
	return rapid.Custom(func(t *rapid.T) T {
		panic(msg)
	})
}

//	"update-with-start": func(t *rapid.T) {
//		//startReq := startWorkflowReq(tv)
//		//startReq.WorkflowIdConflictPolicy = enumspb.WORKFLOW_ID_CONFLICT_POLICY_USE_EXISTING
//		//updateReq := s.updateWorkflowRequest(tv,
//		//	&updatepb.WaitPolicy{LifecycleStage: enumspb.UPDATE_WORKFLOW_EXECUTION_LIFECYCLE_STAGE_COMPLETED}, "1")
//		//uwsCh := sendUpdateWithStart(testcore.NewContext(), startReq, updateReq)
//		//
//		//_, err := s.TaskPoller.PollAndHandleWorkflowTask(tv,
//		//	func(task *workflowservice.PollWorkflowTaskQueueResponse) (*workflowservice.RespondWorkflowTaskCompletedRequest, error) {
//		//		return &workflowservice.RespondWorkflowTaskCompletedRequest{
//		//			Messages: s.UpdateAcceptCompleteMessages(tv, task.Messages[0], "1"),
//		//		}, nil
//		//	})
//		//s.NoError(err)
//	},
//
//	"update workflow": func(t *rapid.T) {
//		if !running {
//			t.Skip("workflow is not running")
//		}
//		// TODO
//
//	//updateReq := s.updateWorkflowRequest(tv,
//	//	&updatepb.WaitPolicy{LifecycleStage: enumspb.UPDATE_WORKFLOW_EXECUTION_LIFECYCLE_STAGE_COMPLETED}, "1")
//
//	// "Check" step. This is run after every other action. No side effects!
//	"": func(t *rapid.T) {
//		// TODO
//	},
//})
