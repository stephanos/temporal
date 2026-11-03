// Package raw follows activity programs over the public WorkflowService API.
package raw

import (
	"context"
	"errors"
	"fmt"
	"time"

	taskqueuepb "go.temporal.io/api/taskqueue/v1"
	"go.temporal.io/api/workflowservice/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"
	"umpire.local/temporal/reference/worker"
)

const Kind = "raw"
const Version = "1"

// Validate checks startup capabilities without polling.
func Validate(program []byte) error { _, err := worker.Decode(program); return err }

// Follow keeps each assignment until its one terminal choice is acknowledged.
// conn and bridge remain caller-owned.
func Follow(ctx context.Context, conn workflowservice.WorkflowServiceClient, program []byte, bridge *worker.Bridge) error {
	table, err := worker.Decode(program)
	if err != nil {
		return err
	}
	if conn == nil || bridge == nil {
		return errors.New("raw executor requires a connection and worker bridge")
	}
	run, cancel := context.WithCancel(ctx)
	joined := make(chan struct{})
	go func() {
		defer close(joined)
		select {
		case <-bridge.Failed():
			cancel()
		case <-bridge.Done():
			cancel()
		case <-run.Done():
		}
	}()
	defer func() { cancel(); <-joined }()
	return followPolls(ctx, run, conn, table, bridge)
}

func followPolls(ctx, run context.Context, conn workflowservice.WorkflowServiceClient, table *worker.Program, bridge *worker.Bridge) error {
	for {
		task, err := conn.PollActivityTaskQueue(run, &workflowservice.PollActivityTaskQueueRequest{Namespace: table.Namespace, TaskQueue: &taskqueuepb.TaskQueue{Name: table.TaskQueue}, Identity: "umpire-raw"})
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			if fault := bridge.Err(); fault != nil {
				return fault
			}
			if status.Code(err) == codes.DeadlineExceeded {
				continue
			}
			return err
		}
		if len(task.GetTaskToken()) == 0 {
			continue
		}
		err = runAssignment(run, conn, table, bridge, task)
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			return err
		}
	}
}

func runAssignment(ctx context.Context, conn workflowservice.WorkflowServiceClient, table *worker.Program, bridge *worker.Bridge, task *workflowservice.PollActivityTaskQueueResponse) error {
	identity := worker.IdentityFromPoll(table.Namespace, table.TaskQueue, task)
	assignment, err := bridge.Bind(identity)
	if err != nil {
		return err
	}
	taskCtx, taskCancel, err := assignmentContext(ctx, task)
	if err != nil {
		return err
	}
	defer taskCancel()
	return followAssignment(assignment.HeartbeatContext(taskCtx), conn, table, assignment, identity)
}

func assignmentContext(ctx context.Context, task *workflowservice.PollActivityTaskQueueResponse) (context.Context, context.CancelFunc, error) {
	var deadline time.Time
	for _, bound := range []struct {
		start   *timestamppb.Timestamp
		timeout *durationpb.Duration
	}{
		{start: task.GetStartedTime(), timeout: task.GetStartToCloseTimeout()},
		{start: task.GetScheduledTime(), timeout: task.GetScheduleToCloseTimeout()},
	} {
		if bound.timeout == nil {
			continue
		}
		if err := bound.timeout.CheckValid(); err != nil {
			return nil, nil, fmt.Errorf("invalid actual task timeout: %w", err)
		}
		timeout := bound.timeout.AsDuration()
		if timeout < 0 {
			return nil, nil, errors.New("negative actual task timeout")
		}
		if timeout == 0 {
			continue
		}
		if err := bound.start.CheckValid(); err != nil {
			return nil, nil, fmt.Errorf("invalid actual task timestamp: %w", err)
		}
		candidate := bound.start.AsTime().Add(timeout)
		if deadline.IsZero() || candidate.Before(deadline) {
			deadline = candidate
		}
	}
	if deadline.IsZero() {
		child, cancel := context.WithCancel(ctx)
		return child, cancel, nil
	}
	child, cancel := context.WithDeadline(ctx, deadline)
	return child, cancel, nil
}

func followAssignment(ctx context.Context, conn workflowservice.WorkflowServiceClient, table *worker.Program, assignment *worker.Assignment, identity worker.TaskIdentity) error {
	history := make([]bool, 0, table.MaxHeartbeats())
	for step := 0; step <= table.MaxHeartbeats(); step++ {
		if err := ctx.Err(); err != nil {
			return err
		}
		message, err := selectedResponse(table, identity, history)
		if err != nil {
			return err
		}
		switch request := message.(type) {
		case *workflowservice.RecordActivityTaskHeartbeatRequest:
			if len(history) >= table.MaxHeartbeats() {
				return errors.New("worker heartbeat depth exhausted")
			}
			bit, err := heartbeat(ctx, conn, assignment, identity, request)
			if err != nil {
				return err
			}
			history = append(history, bit)
		default:
			return respond(ctx, conn, assignment, identity, message)
		}
	}
	return errors.New("worker decision work exhausted")
}

func selectedResponse(table *worker.Program, identity worker.TaskIdentity, history []bool) (proto.Message, error) {
	response := table.Match(identity.Scenario, identity.Attempt, identity.ActivityType, history...)
	if response == nil {
		if len(history) != 0 {
			return nil, errors.New("actual heartbeat history is outside the declared program domain")
		}
		var err error
		response, err = anypb.New(&workflowservice.RespondActivityTaskFailedRequest{Failure: worker.UnprogrammedFailure()})
		if err != nil {
			return nil, err
		}
	}
	return worker.ResponseMessage(response)
}

func heartbeat(ctx context.Context, conn workflowservice.WorkflowServiceClient, assignment *worker.Assignment, identity worker.TaskIdentity, request *workflowservice.RecordActivityTaskHeartbeatRequest) (bool, error) {
	request.Namespace, request.TaskToken, request.Identity = identity.Namespace, identity.Token, "umpire-raw"
	expected, err := anypb.New(request)
	if err != nil {
		return false, err
	}
	wait, err := assignment.BeginHeartbeat()
	if err != nil {
		return false, err
	}
	actual, err := conn.RecordActivityTaskHeartbeat(ctx, request)
	if err != nil {
		return false, err
	}
	if actual == nil {
		return false, errors.New("missing heartbeat acknowledgment")
	}
	completed, err := wait.Wait(ctx)
	if err != nil {
		return false, err
	}
	return worker.HeartbeatResult(expected, completed.RPC())
}

func respond(ctx context.Context, conn workflowservice.WorkflowServiceClient, assignment *worker.Assignment, identity worker.TaskIdentity, message proto.Message) error {
	var method string
	switch request := message.(type) {
	case *workflowservice.RespondActivityTaskCompletedRequest:
		request.Namespace, request.TaskToken, request.Identity = identity.Namespace, identity.Token, "umpire-raw"
		method = workflowservice.WorkflowService_RespondActivityTaskCompleted_FullMethodName
	case *workflowservice.RespondActivityTaskFailedRequest:
		request.Namespace, request.TaskToken, request.Identity = identity.Namespace, identity.Token, "umpire-raw"
		method = workflowservice.WorkflowService_RespondActivityTaskFailed_FullMethodName
	case *workflowservice.RespondActivityTaskCanceledRequest:
		request.Namespace, request.TaskToken, request.Identity = identity.Namespace, identity.Token, "umpire-raw"
		method = workflowservice.WorkflowService_RespondActivityTaskCanceled_FullMethodName
	default:
		return fmt.Errorf("unsupported terminal response %T", message)
	}
	expected, err := anypb.New(message)
	if err != nil {
		return err
	}
	wait, err := assignment.BeginTerminal(method)
	if err != nil {
		return err
	}
	switch request := message.(type) {
	case *workflowservice.RespondActivityTaskCompletedRequest:
		response, err := conn.RespondActivityTaskCompleted(ctx, request)
		if err != nil {
			return err
		}
		if response == nil {
			return errors.New("missing completed acknowledgment")
		}
	case *workflowservice.RespondActivityTaskFailedRequest:
		response, err := conn.RespondActivityTaskFailed(ctx, request)
		if err != nil {
			return err
		}
		if response == nil {
			return errors.New("missing failed acknowledgment")
		}
	case *workflowservice.RespondActivityTaskCanceledRequest:
		response, err := conn.RespondActivityTaskCanceled(ctx, request)
		if err != nil {
			return err
		}
		if response == nil {
			return errors.New("missing canceled acknowledgment")
		}
	default:
		return errors.New("unsupported terminal response")
	}
	completed, err := wait.Wait(ctx)
	if err != nil {
		return err
	}
	return worker.TerminalResult(expected, completed.RPC())
}
