package worker

import (
	"bytes"
	"context"
	"errors"

	genericworker "github.com/temporalio/umpire/hosts/go/worker"
	"go.temporal.io/api/workflowservice/v1"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
	temporalv1 "umpire.local/temporal/umpire/temporal/v1"
)

type activityRoot struct {
	Scenario     string
	Attempt      int32
	ActivityType string
}

func activityKey(key *temporalv1.ActivityTaskKey) genericworker.Key[activityRoot, bool] {
	return genericworker.Key[activityRoot, bool]{Root: activityRoot{Scenario: key.GetScenario(), Attempt: key.GetAttempt(), ActivityType: key.GetActivityType()}, History: key.GetHeartbeatHistory()}
}

func activityCodec() genericworker.ActionCodec[*anypb.Any] {
	return genericworker.ActionCodec[*anypb.Any]{Clone: func(a *anypb.Any) *anypb.Any { return proto.CloneOf(a) }, Admit: func(a *anypb.Any) (genericworker.ActionRole, error) {
		message, err := ResponseMessage(a)
		if err != nil {
			return 0, err
		}
		if _, ok := message.(*workflowservice.RecordActivityTaskHeartbeatRequest); ok {
			return genericworker.Continue, nil
		}
		return genericworker.Finish, nil
	}}
}

// ActivitySession owns one assignment's exact program cursor. Transport and SDK
// lifetimes remain executor-owned; only actual captured completions earn progress.
type ActivitySession struct {
	*genericworker.Session[activityRoot, bool, *anypb.Any, TaskIdentity]
	assignment *Assignment
}
type ActivityTicket = genericworker.Ticket[bool, *anypb.Any]

func (s *ActivitySession) Context(ctx context.Context) context.Context {
	return s.assignment.HeartbeatContext(ctx)
}

// NewSession claims the actual delivery for the program and binds actual task
// identity into immutable expected requests. beforeAdvance is the executor's
// optional notification barrier after a validated actual heartbeat.
func (table *Program) NewSession(bridge *Bridge, identity TaskIdentity, executorIdentity string, beforeAdvance func(context.Context, bool) error) (*ActivitySession, error) {
	if table == nil || table.table == nil || bridge == nil {
		return nil, errors.New("activity session requires a decoded program and bridge")
	}
	if identity.Namespace != table.Namespace || identity.TaskQueue != table.TaskQueue {
		return nil, errors.New("activity session crosses program namespace or queue")
	}
	// Retain neither the caller's token buffer nor its mutable identity storage.
	identity.Token = bytes.Clone(identity.Token)
	fallback, err := anypb.New(&workflowservice.RespondActivityTaskFailedRequest{Failure: UnprogrammedFailure()})
	if err != nil {
		return nil, err
	}
	binding := genericworker.Binding[bool, *anypb.Any]{
		Prepare: func(action *anypb.Any) (genericworker.Prepared, error) {
			return prepareActivity(action, identity, executorIdentity)
		},
		Continue: func(expected *anypb.Any, actual genericworker.Completion) (bool, error) {
			return HeartbeatResult(expected, actual.Capture())
		},
		Finish: func(expected *anypb.Any, actual genericworker.Completion) error {
			return TerminalResult(expected, actual.Capture())
		},
		BeforeAdvance: beforeAdvance,
	}
	root := activityRoot{Scenario: identity.Scenario, Attempt: identity.Attempt, ActivityType: identity.ActivityType}
	assignment, err := bridge.Bind(identity)
	if err != nil {
		return nil, err
	}
	session, err := genericworker.NewSession(table.table, assignment.assignment, root, binding, &fallback)
	if err != nil {
		return nil, err
	}
	return &ActivitySession{Session: session, assignment: assignment}, nil
}

func prepareActivity(action *anypb.Any, identity TaskIdentity, executorIdentity string) (genericworker.Prepared, error) {
	message, err := ResponseMessage(action)
	if err != nil {
		return genericworker.Prepared{}, err
	}
	var method string
	switch request := message.(type) {
	case *workflowservice.RecordActivityTaskHeartbeatRequest:
		request.Namespace, request.TaskToken, request.Identity = identity.Namespace, bytes.Clone(identity.Token), executorIdentity
		method = workflowservice.WorkflowService_RecordActivityTaskHeartbeat_FullMethodName
	case *workflowservice.RespondActivityTaskCompletedRequest:
		request.Namespace, request.TaskToken, request.Identity = identity.Namespace, bytes.Clone(identity.Token), executorIdentity
		method = workflowservice.WorkflowService_RespondActivityTaskCompleted_FullMethodName
	case *workflowservice.RespondActivityTaskFailedRequest:
		request.Namespace, request.TaskToken, request.Identity = identity.Namespace, bytes.Clone(identity.Token), executorIdentity
		method = workflowservice.WorkflowService_RespondActivityTaskFailed_FullMethodName
	case *workflowservice.RespondActivityTaskCanceledRequest:
		request.Namespace, request.TaskToken, request.Identity = identity.Namespace, bytes.Clone(identity.Token), executorIdentity
		method = workflowservice.WorkflowService_RespondActivityTaskCanceled_FullMethodName
	default:
		return genericworker.Prepared{}, errors.New("unsupported activity action")
	}
	request, err := anypb.New(message)
	return genericworker.Prepared{Method: method, Request: request}, err
}
