package worker

import (
	"bytes"
	"errors"
	"fmt"

	genericworker "github.com/temporalio/umpire/hosts/go/worker"
	"go.temporal.io/api/workflowservice/v1"
	"google.golang.org/protobuf/proto"
)

// TaskIdentity contains public poll/SDK assignment values, without decoding the token.
type TaskIdentity struct {
	Namespace, TaskQueue, ActivityID, ActivityRunID   string
	WorkflowID, WorkflowRunID, ActivityType, Scenario string
	Attempt                                           int32
	Token                                             []byte
}

func (i TaskIdentity) same(other TaskIdentity) bool {
	return i.Namespace == other.Namespace && i.TaskQueue == other.TaskQueue &&
		i.ActivityID == other.ActivityID && i.ActivityRunID == other.ActivityRunID &&
		i.WorkflowID == other.WorkflowID && i.WorkflowRunID == other.WorkflowRunID &&
		i.ActivityType == other.ActivityType && i.Scenario == other.Scenario &&
		i.Attempt == other.Attempt && bytes.Equal(i.Token, other.Token)
}

// IdentityFromPoll reads assignment identity only from the actual poll request/reply.
func IdentityFromPoll(namespace, taskQueue string, task *workflowservice.PollActivityTaskQueueResponse) TaskIdentity {
	return TaskIdentity{Namespace: namespace, TaskQueue: taskQueue,
		ActivityID: task.GetActivityId(), ActivityRunID: task.GetActivityRunId(),
		WorkflowID: task.GetWorkflowExecution().GetWorkflowId(), WorkflowRunID: task.GetWorkflowExecution().GetRunId(),
		ActivityType: task.GetActivityType().GetName(), Scenario: Scenario(task.GetHeader().GetFields()[ScenarioHeader]),
		Attempt: task.GetAttempt(), Token: bytes.Clone(task.GetTaskToken())}
}

func activityProtocol() genericworker.Protocol[TaskIdentity] {
	return genericworker.Protocol[TaskIdentity]{
		Methods: map[string]genericworker.Method{
			workflowservice.WorkflowService_PollActivityTaskQueue_FullMethodName:        {Role: genericworker.DeliveryRole},
			workflowservice.WorkflowService_RecordActivityTaskHeartbeat_FullMethodName:  {Role: genericworker.Continuing, RequireAssignmentContext: true},
			workflowservice.WorkflowService_RespondActivityTaskCompleted_FullMethodName: {Role: genericworker.Terminal},
			workflowservice.WorkflowService_RespondActivityTaskFailed_FullMethodName:    {Role: genericworker.Terminal},
			workflowservice.WorkflowService_RespondActivityTaskCanceled_FullMethodName:  {Role: genericworker.Terminal},
		},
		Delivery: activityDelivery,
		OperationAddress: func(c genericworker.Capture) (genericworker.Address, error) {
			namespace, token, err := rpcAssignment(c)
			return genericworker.Address{Scope: namespace, Handle: string(token)}, err
		},
		CloneIdentity: func(i TaskIdentity) TaskIdentity { i.Token = bytes.Clone(i.Token); return i },
		SameIdentity:  func(a, b TaskIdentity) bool { return a.same(b) },
	}
}
func activityDelivery(rpc genericworker.Capture) (genericworker.Delivery[TaskIdentity], bool, error) {
	request := new(workflowservice.PollActivityTaskQueueRequest)
	response := new(workflowservice.PollActivityTaskQueueResponse)
	if err := rpc.Request.UnmarshalTo(request); err != nil {
		return genericworker.Delivery[TaskIdentity]{}, false, (fmt.Errorf("poll request: %w", err))
	}
	if err := rpc.Response.UnmarshalTo(response); err != nil {
		return genericworker.Delivery[TaskIdentity]{}, false, (fmt.Errorf("poll response: %w", err))
	}
	if len(response.GetTaskToken()) == 0 {
		return genericworker.Delivery[TaskIdentity]{}, false, nil
	}
	if response.GetWorkflowNamespace() != "" && response.GetWorkflowNamespace() != request.GetNamespace() {
		return genericworker.Delivery[TaskIdentity]{}, false, (errors.New("poll response namespace crosses actual poll request"))
	}
	identity := IdentityFromPoll(request.GetNamespace(), request.GetTaskQueue().GetName(), response)
	if identity.Namespace == "" || identity.TaskQueue == "" || identity.ActivityType == "" || identity.Attempt <= 0 {
		return genericworker.Delivery[TaskIdentity]{}, false, (errors.New("poll assignment identity is incomplete"))
	}
	return genericworker.Delivery[TaskIdentity]{Address: genericworker.Address{Scope: identity.Namespace, Handle: string(identity.Token)}, Identity: identity}, true, nil
}

func rpcAssignment(rpc genericworker.Capture) (string, []byte, error) {
	var namespace string
	var token []byte
	var request proto.Message
	switch rpc.Method {
	case workflowservice.WorkflowService_RecordActivityTaskHeartbeat_FullMethodName:
		request = new(workflowservice.RecordActivityTaskHeartbeatRequest)
	case workflowservice.WorkflowService_RespondActivityTaskCompleted_FullMethodName:
		request = new(workflowservice.RespondActivityTaskCompletedRequest)
	case workflowservice.WorkflowService_RespondActivityTaskFailed_FullMethodName:
		request = new(workflowservice.RespondActivityTaskFailedRequest)
	case workflowservice.WorkflowService_RespondActivityTaskCanceled_FullMethodName:
		request = new(workflowservice.RespondActivityTaskCanceledRequest)
	default:
		return "", nil, errors.New("unsupported worker RPC method")
	}
	if err := rpc.Request.UnmarshalTo(request); err != nil {
		return "", nil, fmt.Errorf("worker RPC request: %w", err)
	}
	fields, ok := request.(interface {
		GetNamespace() string
		GetTaskToken() []byte
	})
	if !ok {
		return "", nil, errors.New("worker request has no assignment identity")
	}
	namespace, token = fields.GetNamespace(), fields.GetTaskToken()
	if namespace == "" || len(token) == 0 {
		return "", nil, errors.New("worker request assignment identity is missing")
	}
	return namespace, token, nil
}
