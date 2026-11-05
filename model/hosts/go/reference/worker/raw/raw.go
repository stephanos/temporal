// Package raw follows activity programs over the public WorkflowService API.
package raw

import (
	"context"
	"errors"
	"fmt"
	"time"

	genericworker "github.com/temporalio/umpire/hosts/go/worker"
	taskqueuepb "go.temporal.io/api/taskqueue/v1"
	"go.temporal.io/api/workflowservice/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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
	session, err := table.NewSession(bridge, identity, "umpire-raw", nil)
	if err != nil {
		return err
	}
	taskCtx, taskCancel, err := assignmentContext(ctx, task)
	if err != nil {
		return err
	}
	defer taskCancel()
	return followAssignment(session.Context(taskCtx), conn, session)
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

func followAssignment(ctx context.Context, conn workflowservice.WorkflowServiceClient, session *worker.ActivitySession) error {
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		ticket, err := session.Begin()
		if err != nil {
			return err
		}
		if err := dispatch(ctx, conn, ticket); err != nil {
			return err
		}
		if err := ticket.Wait(ctx); err != nil {
			return err
		}
		if ticket.Role() == genericworker.Finish {
			return nil
		}
	}
}

func dispatch(ctx context.Context, conn workflowservice.WorkflowServiceClient, ticket *worker.ActivityTicket) error {
	message, err := worker.ResponseMessage(ticket.Request())
	if err != nil {
		return err
	}
	switch request := message.(type) {
	case *workflowservice.RecordActivityTaskHeartbeatRequest:
		response, err := conn.RecordActivityTaskHeartbeat(ctx, request)
		if err != nil {
			return err
		}
		if response == nil {
			return errors.New("missing heartbeat acknowledgment")
		}
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
		return fmt.Errorf("unsupported worker response %T", message)
	}
	return nil
}
