// Package controller runs the production canary's invocation: it reads and takes the lease, runs
// the activity Model through the Runner serially, each Run's Temporal Driver fenced to one
// activity, publishes the Runs, and cleans up exactly what the lease's fence names. The lease is a
// workflow on a task queue no worker polls; its run ID is the fence, and its `run-opened` signals
// are the server-side list of every activity the fence may touch.
package controller

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	commonpb "go.temporal.io/api/common/v1"
	enumspb "go.temporal.io/api/enums/v1"
	"go.temporal.io/api/serviceerror"
	taskqueuepb "go.temporal.io/api/taskqueue/v1"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/converter"
	"go.temporal.io/server/tools/canary/policy"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/durationpb"
)

// The lease's vocabulary: the signal that fences a Run, and the only termination reasons that
// leave the scope clean.
const (
	SignalRunOpened  = "run-opened"
	ReasonReleased   = "umpire-canary: released"
	ReasonReconciled = "umpire-canary: reconciled"
	// ReasonCleanup is the reason a fenced activity is terminated with.
	ReasonCleanup = "umpire-canary: cleanup"
)

// maxLeaseEvents bounds a read of the lease's history: it holds a start, one signal per iteration
// and a close, so anything past this is not a lease the canary made.
const maxLeaseEvents = 1024

// Service is the part of the WorkflowService the lease and cleanup use: all namespace-scoped.
type Service interface {
	DescribeActivityExecution(context.Context, *workflowservice.DescribeActivityExecutionRequest, ...grpc.CallOption) (*workflowservice.DescribeActivityExecutionResponse, error)
	TerminateActivityExecution(context.Context, *workflowservice.TerminateActivityExecutionRequest, ...grpc.CallOption) (*workflowservice.TerminateActivityExecutionResponse, error)
	StartWorkflowExecution(context.Context, *workflowservice.StartWorkflowExecutionRequest, ...grpc.CallOption) (*workflowservice.StartWorkflowExecutionResponse, error)
	SignalWorkflowExecution(context.Context, *workflowservice.SignalWorkflowExecutionRequest, ...grpc.CallOption) (*workflowservice.SignalWorkflowExecutionResponse, error)
	TerminateWorkflowExecution(context.Context, *workflowservice.TerminateWorkflowExecutionRequest, ...grpc.CallOption) (*workflowservice.TerminateWorkflowExecutionResponse, error)
	DescribeWorkflowExecution(context.Context, *workflowservice.DescribeWorkflowExecutionRequest, ...grpc.CallOption) (*workflowservice.DescribeWorkflowExecutionResponse, error)
	GetWorkflowExecutionHistory(context.Context, *workflowservice.GetWorkflowExecutionHistoryRequest, ...grpc.CallOption) (*workflowservice.GetWorkflowExecutionHistoryResponse, error)
}

// Target is where the lease and the fenced activities live, and the identity the canary's own calls
// carry.
type Target struct {
	Service   Service
	Namespace string
	Identity  string
}

// Fence is one lease run: the lease's workflow ID and the run ID the canary holds.
type Fence struct {
	WorkflowID string
	RunID      string
}

// LeaseState is what the lease's latest run says about the scope.
type LeaseState int

const (
	// LeaseAbsent is a lease ID the server does not find: the first run ever, or one whose history
	// aged out. The scope is clean.
	LeaseAbsent LeaseState = iota + 1
	// LeaseReleased is a latest run the canary terminated as released or reconciled. Clean.
	LeaseReleased
	// LeaseOpen is a latest run still open: a live invocation or a lost process. Unreconciled.
	LeaseOpen
	// LeaseClosedOtherwise is a latest run that timed out or closed any way but a canary
	// termination, a manual one included. Unreconciled.
	LeaseClosedOtherwise
)

func (s LeaseState) String() string {
	switch s {
	case LeaseAbsent:
		return "absent"
	case LeaseReleased:
		return "released"
	case LeaseOpen:
		return "open"
	case LeaseClosedOtherwise:
		return "closed-otherwise"
	default:
		return "unknown"
	}
}

// Clean reports whether a new lease may be taken.
func (s LeaseState) Clean() bool { return s == LeaseAbsent || s == LeaseReleased }

// Observed is the lease's latest run as leaseState reads it.
type Observed struct {
	State   LeaseState
	RunID   string
	Started time.Time
}

// LeaseHeldError is a lease start that collided with a run the canary did not take.
type LeaseHeldError struct{ RunID string }

func (e *LeaseHeldError) Error() string {
	return "the canary lease is already held by run " + e.RunID
}

func notFound(err error) bool {
	var missing *serviceerror.NotFound
	return errors.As(err, &missing) || status.Code(err) == codes.NotFound
}

// leaseState reads the lease's latest run and, when it is closed, its close event, since a
// describe reports a termination without its reason. The controller and reconcile share it.
func leaseState(ctx context.Context, target Target, leaseID string) (Observed, error) {
	return leaseRunState(ctx, target, &commonpb.WorkflowExecution{WorkflowId: leaseID})
}

// fenceState is leaseState for one exact lease run, the fence a recovery record names.
func fenceState(ctx context.Context, target Target, fence Fence) (Observed, error) {
	return leaseRunState(ctx, target, &commonpb.WorkflowExecution{WorkflowId: fence.WorkflowID, RunId: fence.RunID})
}

func leaseRunState(ctx context.Context, target Target, execution *commonpb.WorkflowExecution) (Observed, error) {
	leaseID := execution.GetWorkflowId()
	described, err := target.Service.DescribeWorkflowExecution(ctx, &workflowservice.DescribeWorkflowExecutionRequest{
		Namespace: target.Namespace, Execution: execution,
	})
	if notFound(err) {
		return Observed{State: LeaseAbsent}, nil
	}
	if err != nil {
		return Observed{}, fmt.Errorf("describe the canary lease: %w", err)
	}
	info := described.GetWorkflowExecutionInfo()
	observed := Observed{RunID: info.GetExecution().GetRunId(), Started: info.GetStartTime().AsTime()}
	if observed.RunID == "" {
		return Observed{}, errors.New("the canary lease's latest run has no run ID")
	}
	if info.GetStatus() == enumspb.WORKFLOW_EXECUTION_STATUS_RUNNING {
		observed.State = LeaseOpen
		return observed, nil
	}
	closing, err := target.Service.GetWorkflowExecutionHistory(ctx, &workflowservice.GetWorkflowExecutionHistoryRequest{
		Namespace: target.Namespace, Execution: &commonpb.WorkflowExecution{WorkflowId: leaseID, RunId: observed.RunID},
		HistoryEventFilterType: enumspb.HISTORY_EVENT_FILTER_TYPE_CLOSE_EVENT,
	})
	if err != nil {
		return Observed{}, fmt.Errorf("read the canary lease's close event: %w", err)
	}
	events := closing.GetHistory().GetEvents()
	observed.State = LeaseClosedOtherwise
	if len(events) > 0 {
		terminated := events[len(events)-1].GetWorkflowExecutionTerminatedEventAttributes()
		if terminated != nil && (terminated.GetReason() == ReasonReleased || terminated.GetReason() == ReasonReconciled) {
			observed.State = LeaseReleased
		}
	}
	return observed, nil
}

// takeLease starts the lease's run with the policy's ID, type and unpolled queue, failing on any
// open run, and returns its run ID, the fence. A collision is a LeaseHeldError.
func takeLease(ctx context.Context, target Target, lease policy.Lease, timeout time.Duration, requestID string) (Fence, error) {
	started, err := target.Service.StartWorkflowExecution(ctx, &workflowservice.StartWorkflowExecutionRequest{
		Namespace:                target.Namespace,
		WorkflowId:               lease.WorkflowID,
		WorkflowType:             &commonpb.WorkflowType{Name: lease.WorkflowType},
		TaskQueue:                &taskqueuepb.TaskQueue{Name: lease.TaskQueue, Kind: enumspb.TASK_QUEUE_KIND_NORMAL},
		WorkflowRunTimeout:       durationpb.New(timeout),
		WorkflowIdConflictPolicy: enumspb.WORKFLOW_ID_CONFLICT_POLICY_FAIL,
		WorkflowIdReusePolicy:    enumspb.WORKFLOW_ID_REUSE_POLICY_ALLOW_DUPLICATE,
		RequestId:                requestID,
		Identity:                 target.Identity,
	})
	var held *serviceerror.WorkflowExecutionAlreadyStarted
	if errors.As(err, &held) {
		return Fence{}, &LeaseHeldError{RunID: held.RunId}
	}
	if err != nil {
		return Fence{}, fmt.Errorf("take the canary lease: %w", err)
	}
	if started.GetRunId() == "" || !started.GetStarted() {
		return Fence{}, errors.New("the canary lease start returned no new run")
	}
	return Fence{WorkflowID: lease.WorkflowID, RunID: started.GetRunId()}, nil
}

// signalRunOpened records an activity ID in the lease's history before its Run starts, sent to the
// exact fence run, so a stale fence -- a lease run that has closed -- fails the signal.
func signalRunOpened(ctx context.Context, target Target, fence Fence, activityID string) error {
	input, err := converter.GetDefaultDataConverter().ToPayloads(activityID)
	if err != nil {
		return err
	}
	_, err = target.Service.SignalWorkflowExecution(ctx, &workflowservice.SignalWorkflowExecutionRequest{
		Namespace:         target.Namespace,
		WorkflowExecution: &commonpb.WorkflowExecution{WorkflowId: fence.WorkflowID, RunId: fence.RunID},
		SignalName:        SignalRunOpened, Input: input, Identity: target.Identity, RequestId: activityID,
	})
	if err != nil {
		return fmt.Errorf("fence the Run on the canary lease: %w", err)
	}
	return nil
}

// fencedIDs reads every activity ID the fence's `run-opened` signals name, in order, once each.
func fencedIDs(ctx context.Context, target Target, fence Fence) ([]string, error) {
	var ids []string
	var token []byte
	read := 0
	for {
		page, err := target.Service.GetWorkflowExecutionHistory(ctx, &workflowservice.GetWorkflowExecutionHistoryRequest{
			Namespace: target.Namespace, Execution: &commonpb.WorkflowExecution{WorkflowId: fence.WorkflowID, RunId: fence.RunID},
			NextPageToken: token,
		})
		if err != nil {
			return nil, fmt.Errorf("read the canary lease's fence: %w", err)
		}
		for _, event := range page.GetHistory().GetEvents() {
			if read++; read > maxLeaseEvents {
				return nil, fmt.Errorf("the canary lease's history exceeds %d events", maxLeaseEvents)
			}
			signaled := event.GetWorkflowExecutionSignaledEventAttributes()
			if signaled == nil || signaled.GetSignalName() != SignalRunOpened {
				continue
			}
			var id string
			if err := converter.GetDefaultDataConverter().FromPayloads(signaled.GetInput(), &id); err != nil || !IsActivityID(id) {
				// Only the canary's own fence names an activity; anything else signalled to the lease
				// fails closed, so cleanup never acts on an execution the canary did not start.
				return nil, fmt.Errorf("a %s signal on the canary lease names no canary activity", SignalRunOpened)
			}
			if !slices.Contains(ids, id) {
				ids = append(ids, id)
			}
		}
		if token = page.GetNextPageToken(); len(token) == 0 {
			return ids, nil
		}
	}
}

// identityPrefix begins the identity every canary call carries, the invocation's ID after it.
const identityPrefix = "umpire-canary "

// leaseInvocation is the invocation that started a lease run, read from the identity its start
// carries, or "" when the start names no canary invocation.
func leaseInvocation(ctx context.Context, target Target, fence Fence) (string, error) {
	page, err := target.Service.GetWorkflowExecutionHistory(ctx, &workflowservice.GetWorkflowExecutionHistoryRequest{
		Namespace: target.Namespace, Execution: &commonpb.WorkflowExecution{WorkflowId: fence.WorkflowID, RunId: fence.RunID},
		MaximumPageSize: 1,
	})
	if err != nil {
		return "", fmt.Errorf("read the canary lease's start: %w", err)
	}
	events := page.GetHistory().GetEvents()
	if len(events) == 0 {
		return "", nil
	}
	identity := events[0].GetWorkflowExecutionStartedEventAttributes().GetIdentity()
	invocation, ok := strings.CutPrefix(identity, identityPrefix)
	if !ok || strings.Contains(invocation, " ") {
		return "", nil
	}
	return invocation, nil
}

// Wait pauses for d or until ctx ends.
type Wait func(ctx context.Context, d time.Duration) error

func sleep(ctx context.Context, d time.Duration) error {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

// activityClosed reports whether a fenced activity is closed. One the server does not find on two
// reads a pause apart never started -- its start was fenced but never reached the server -- and
// counts as closed. Cleanup and reconcile share it.
func activityClosed(ctx context.Context, target Target, activityID string, pause time.Duration, wait Wait) (bool, error) {
	for attempt := range 2 {
		described, err := target.Service.DescribeActivityExecution(ctx, &workflowservice.DescribeActivityExecutionRequest{
			Namespace: target.Namespace, ActivityId: activityID,
		})
		if notFound(err) {
			if attempt == 0 {
				if err := wait(ctx, pause); err != nil {
					return false, err
				}
			}
			continue
		}
		if err != nil {
			return false, fmt.Errorf("describe a fenced activity: %w", err)
		}
		switch described.GetInfo().GetStatus() {
		case enumspb.ACTIVITY_EXECUTION_STATUS_COMPLETED, enumspb.ACTIVITY_EXECUTION_STATUS_FAILED, enumspb.ACTIVITY_EXECUTION_STATUS_CANCELED,
			enumspb.ACTIVITY_EXECUTION_STATUS_TERMINATED, enumspb.ACTIVITY_EXECUTION_STATUS_TIMED_OUT:
			return true, nil
		default:
			return false, nil
		}
	}
	return true, nil
}

// closeFenced verifies one fenced activity closed, terminating it with reason first when it is
// open, and says whether it is closed and whether it was terminated. Cleanup and reconcile share it.
func closeFenced(ctx context.Context, target Target, id string, pause time.Duration, wait Wait, reason string) (closed, terminated bool, err error) {
	closed, err = activityClosed(ctx, target, id, pause, wait)
	if err != nil || closed {
		return closed, false, err
	}
	_, err = target.Service.TerminateActivityExecution(ctx, &workflowservice.TerminateActivityExecutionRequest{
		Namespace: target.Namespace, ActivityId: id, Reason: reason, Identity: target.Identity,
	})
	if err != nil && !notFound(err) && status.Code(err) != codes.FailedPrecondition {
		return false, false, err
	}
	closed, err = activityClosed(ctx, target, id, pause, wait)
	return closed, true, err
}

// terminate ends one lease run with reason. A run already closed or never started is not an
// error: the caller reads the lease back.
func terminate(ctx context.Context, target Target, execution *commonpb.WorkflowExecution, reason string) error {
	_, err := target.Service.TerminateWorkflowExecution(ctx, &workflowservice.TerminateWorkflowExecutionRequest{
		Namespace: target.Namespace, WorkflowExecution: execution, Reason: reason, Identity: target.Identity,
	})
	if err != nil && !notFound(err) {
		return err
	}
	return nil
}
