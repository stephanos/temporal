// Package raw follows activity programs over the public WorkflowService API.
package raw

import (
	"context"
	"errors"

	taskqueuepb "go.temporal.io/api/taskqueue/v1"
	"go.temporal.io/api/workflowservice/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/anypb"
	"umpire.local/temporal/reference/worker"
)

const Kind = "raw"
const Version = "1"

// Validate checks startup capabilities without polling.
func Validate(program []byte) error { _, err := worker.Decode(program); return err }

// Follow polls and responds until ctx ends. conn is owned and closed by the caller.
func Follow(ctx context.Context, conn workflowservice.WorkflowServiceClient, program []byte) error {
	table, err := worker.Decode(program)
	if err != nil {
		return err
	}
	if conn == nil {
		return errors.New("raw executor requires a connection")
	}
	for {
		task, err := conn.PollActivityTaskQueue(ctx, &workflowservice.PollActivityTaskQueueRequest{Namespace: table.Namespace, TaskQueue: &taskqueuepb.TaskQueue{Name: table.TaskQueue}, Identity: "umpire-raw"})
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			if status.Code(err) == codes.DeadlineExceeded {
				continue
			}
			return err
		}
		if len(task.GetTaskToken()) == 0 {
			continue
		}
		response := table.Match(worker.Scenario(task.GetHeader().GetFields()[worker.ScenarioHeader]), task.GetAttempt(), task.GetActivityType().GetName())
		if err := respond(ctx, conn, table.Namespace, task.GetTaskToken(), response); err != nil {
			if ctx.Err() != nil {
				return nil
			}
			return err
		}
	}
}

func respond(ctx context.Context, conn workflowservice.WorkflowServiceClient, namespace string, token []byte, response *anypb.Any) error {
	if response == nil {
		_, err := conn.RespondActivityTaskFailed(ctx, &workflowservice.RespondActivityTaskFailedRequest{Namespace: namespace, TaskToken: token, Identity: "umpire-raw", Failure: worker.UnprogrammedFailure()})
		return err
	}
	if response.MessageIs(&workflowservice.RespondActivityTaskCompletedRequest{}) {
		request := new(workflowservice.RespondActivityTaskCompletedRequest)
		if err := response.UnmarshalTo(request); err != nil {
			return err
		}
		request.Namespace, request.TaskToken, request.Identity = namespace, token, "umpire-raw"
		_, err := conn.RespondActivityTaskCompleted(ctx, request)
		return err
	}
	request := new(workflowservice.RespondActivityTaskFailedRequest)
	if err := response.UnmarshalTo(request); err != nil {
		return err
	}
	request.Namespace, request.TaskToken, request.Identity = namespace, token, "umpire-raw"
	_, err := conn.RespondActivityTaskFailed(ctx, request)
	return err
}
