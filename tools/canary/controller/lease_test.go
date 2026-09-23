package controller

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	commonpb "go.temporal.io/api/common/v1"
	enumspb "go.temporal.io/api/enums/v1"
	historypb "go.temporal.io/api/history/v1"
	namespacepb "go.temporal.io/api/namespace/v1"
	"go.temporal.io/api/serviceerror"
	workflowpb "go.temporal.io/api/workflow/v1"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/converter"
	"go.temporal.io/server/tools/canary/policy"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const testNamespace = "canary-test"

// fakeRun is one workflow run the fake server holds.
type fakeRun struct {
	runID   string
	status  enumspb.WorkflowExecutionStatus
	started time.Time
	events  []*historypb.HistoryEvent
}

func (r *fakeRun) close(status enumspb.WorkflowExecutionStatus, event *historypb.HistoryEvent) {
	r.status = status
	r.events = append(r.events, event)
}

// fakeServer is the Service a lease and cleanup see: workflow runs by ID, each run's history, and
// failures a test injects per method and workflow ID. It pages history two events at a time.
type fakeServer struct {
	mu       sync.Mutex
	runs     map[string][]*fakeRun
	next     int
	fail     map[string]error
	calls    []string
	terminal map[string]bool
}

func newFakeServer() *fakeServer {
	return &fakeServer{runs: map[string][]*fakeRun{}, fail: map[string]error{}, terminal: map[string]bool{}}
}

func (s *fakeServer) failing(ctx context.Context, method, workflowID string) error {
	s.calls = append(s.calls, method+" "+workflowID)
	if err := ctx.Err(); err != nil {
		return err
	}
	return s.fail[method+" "+workflowID]
}

func (s *fakeServer) latest(workflowID string) *fakeRun {
	runs := s.runs[workflowID]
	if len(runs) == 0 {
		return nil
	}
	return runs[len(runs)-1]
}

func (s *fakeServer) find(execution *commonpb.WorkflowExecution) *fakeRun {
	if execution.GetRunId() == "" {
		return s.latest(execution.GetWorkflowId())
	}
	for _, run := range s.runs[execution.GetWorkflowId()] {
		if run.runID == execution.GetRunId() {
			return run
		}
	}
	return nil
}

// open starts a run directly, as a Run's own workflow start or a hand-started lease would.
func (s *fakeServer) open(workflowID string, started time.Time) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.next++
	run := &fakeRun{runID: fmt.Sprintf("00000000-0000-4000-9000-%012d", s.next), status: enumspb.WORKFLOW_EXECUTION_STATUS_RUNNING, started: started,
		events: []*historypb.HistoryEvent{{EventType: enumspb.EVENT_TYPE_WORKFLOW_EXECUTION_STARTED}}}
	s.runs[workflowID] = append(s.runs[workflowID], run)
	return run.runID
}

func (s *fakeServer) finish(workflowID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.latest(workflowID).close(enumspb.WORKFLOW_EXECUTION_STATUS_COMPLETED, &historypb.HistoryEvent{EventType: enumspb.EVENT_TYPE_WORKFLOW_EXECUTION_COMPLETED})
}

func (s *fakeServer) timeOut(workflowID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.latest(workflowID).close(enumspb.WORKFLOW_EXECUTION_STATUS_TIMED_OUT, &historypb.HistoryEvent{EventType: enumspb.EVENT_TYPE_WORKFLOW_EXECUTION_TIMED_OUT})
}

func (s *fakeServer) StartWorkflowExecution(ctx context.Context, request *workflowservice.StartWorkflowExecutionRequest, _ ...grpc.CallOption) (*workflowservice.StartWorkflowExecutionResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.failing(ctx, "start", request.GetWorkflowId()); err != nil {
		return nil, err
	}
	if latest := s.latest(request.GetWorkflowId()); latest != nil && latest.status == enumspb.WORKFLOW_EXECUTION_STATUS_RUNNING {
		return nil, serviceerror.NewWorkflowExecutionAlreadyStarted("already started", request.GetRequestId(), latest.runID)
	}
	s.next++
	run := &fakeRun{runID: fmt.Sprintf("00000000-0000-4000-9000-%012d", s.next), status: enumspb.WORKFLOW_EXECUTION_STATUS_RUNNING, started: time.Unix(1000, 0),
		events: []*historypb.HistoryEvent{{EventType: enumspb.EVENT_TYPE_WORKFLOW_EXECUTION_STARTED}}}
	s.runs[request.GetWorkflowId()] = append(s.runs[request.GetWorkflowId()], run)
	return &workflowservice.StartWorkflowExecutionResponse{RunId: run.runID, Started: true}, nil
}

func (s *fakeServer) SignalWorkflowExecution(ctx context.Context, request *workflowservice.SignalWorkflowExecutionRequest, _ ...grpc.CallOption) (*workflowservice.SignalWorkflowExecutionResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.failing(ctx, "signal", request.GetWorkflowExecution().GetWorkflowId()); err != nil {
		return nil, err
	}
	run := s.find(request.GetWorkflowExecution())
	if run == nil || run.status != enumspb.WORKFLOW_EXECUTION_STATUS_RUNNING {
		return nil, serviceerror.NewNotFound("workflow execution already completed")
	}
	run.events = append(run.events, &historypb.HistoryEvent{
		EventType: enumspb.EVENT_TYPE_WORKFLOW_EXECUTION_SIGNALED,
		Attributes: &historypb.HistoryEvent_WorkflowExecutionSignaledEventAttributes{WorkflowExecutionSignaledEventAttributes: &historypb.WorkflowExecutionSignaledEventAttributes{
			SignalName: request.GetSignalName(), Input: request.GetInput(),
		}},
	})
	return &workflowservice.SignalWorkflowExecutionResponse{}, nil
}

func (s *fakeServer) TerminateWorkflowExecution(ctx context.Context, request *workflowservice.TerminateWorkflowExecutionRequest, _ ...grpc.CallOption) (*workflowservice.TerminateWorkflowExecutionResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.failing(ctx, "terminate", request.GetWorkflowExecution().GetWorkflowId()); err != nil {
		return nil, err
	}
	run := s.find(request.GetWorkflowExecution())
	if run == nil || run.status != enumspb.WORKFLOW_EXECUTION_STATUS_RUNNING {
		return nil, serviceerror.NewNotFound("workflow execution already completed")
	}
	if s.terminal[request.GetWorkflowExecution().GetWorkflowId()] {
		// A run the server acknowledges terminating but that stays open, as a stuck close would.
		return &workflowservice.TerminateWorkflowExecutionResponse{}, nil
	}
	run.close(enumspb.WORKFLOW_EXECUTION_STATUS_TERMINATED, &historypb.HistoryEvent{
		EventType: enumspb.EVENT_TYPE_WORKFLOW_EXECUTION_TERMINATED,
		Attributes: &historypb.HistoryEvent_WorkflowExecutionTerminatedEventAttributes{WorkflowExecutionTerminatedEventAttributes: &historypb.WorkflowExecutionTerminatedEventAttributes{
			Reason: request.GetReason(),
		}},
	})
	return &workflowservice.TerminateWorkflowExecutionResponse{}, nil
}

func (s *fakeServer) DescribeWorkflowExecution(ctx context.Context, request *workflowservice.DescribeWorkflowExecutionRequest, _ ...grpc.CallOption) (*workflowservice.DescribeWorkflowExecutionResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.failing(ctx, "describe", request.GetExecution().GetWorkflowId()); err != nil {
		return nil, err
	}
	run := s.find(request.GetExecution())
	if run == nil {
		return nil, serviceerror.NewNotFound("workflow not found")
	}
	return &workflowservice.DescribeWorkflowExecutionResponse{WorkflowExecutionInfo: &workflowpb.WorkflowExecutionInfo{
		Execution: &commonpb.WorkflowExecution{WorkflowId: request.GetExecution().GetWorkflowId(), RunId: run.runID},
		Status:    run.status, StartTime: timestamppb.New(run.started),
	}}, nil
}

// DescribeNamespace answers for the one registered namespace the fake serves.
func (s *fakeServer) DescribeNamespace(ctx context.Context, request *workflowservice.DescribeNamespaceRequest, _ ...grpc.CallOption) (*workflowservice.DescribeNamespaceResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.failing(ctx, "describe-namespace", request.GetNamespace()); err != nil {
		return nil, err
	}
	if request.GetNamespace() != testNamespace {
		return nil, serviceerror.NewNamespaceNotFound(request.GetNamespace())
	}
	return &workflowservice.DescribeNamespaceResponse{NamespaceInfo: &namespacepb.NamespaceInfo{
		Name: testNamespace, State: enumspb.NAMESPACE_STATE_REGISTERED,
	}}, nil
}

func (s *fakeServer) GetWorkflowExecutionHistory(ctx context.Context, request *workflowservice.GetWorkflowExecutionHistoryRequest, _ ...grpc.CallOption) (*workflowservice.GetWorkflowExecutionHistoryResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.failing(ctx, "history", request.GetExecution().GetWorkflowId()); err != nil {
		return nil, err
	}
	run := s.find(request.GetExecution())
	if run == nil {
		return nil, serviceerror.NewNotFound("workflow not found")
	}
	if request.GetHistoryEventFilterType() == enumspb.HISTORY_EVENT_FILTER_TYPE_CLOSE_EVENT {
		return &workflowservice.GetWorkflowExecutionHistoryResponse{History: &historypb.History{Events: run.events[len(run.events)-1:]}}, nil
	}
	start := 0
	if token := request.GetNextPageToken(); len(token) > 0 {
		var err error
		if start, err = strconv.Atoi(string(token)); err != nil {
			return nil, serviceerror.NewInvalidArgument("bad page token")
		}
	}
	end := min(start+2, len(run.events))
	response := &workflowservice.GetWorkflowExecutionHistoryResponse{History: &historypb.History{Events: run.events[start:end]}}
	if end < len(run.events) {
		response.NextPageToken = []byte(strconv.Itoa(end))
	}
	return response, nil
}

func target(server *fakeServer) Target {
	return Target{Service: server, Namespace: testNamespace, Identity: "umpire-canary"}
}

func testPolicy(t *testing.T) *policy.Policy {
	t.Helper()
	canary, err := policy.Embedded()
	require.NoError(t, err)
	return canary
}

func noWait(context.Context, time.Duration) error { return nil }

// runID is the nth Run ID in the form Testpilot chooses.
func runID(n int) string { return fmt.Sprintf("testpilot.run.00000000-0000-4000-8000-%012d", n) }

func TestLeaseStateReadsTheLatestRunAndItsCloseReason(t *testing.T) {
	ctx := t.Context()
	server := newFakeServer()
	leaseID := testPolicy(t).Lease.WorkflowID

	observed, err := leaseState(ctx, target(server), leaseID)
	require.NoError(t, err)
	require.Equal(t, LeaseAbsent, observed.State, "a lease never taken is clean")
	require.True(t, observed.State.Clean())

	fence, err := takeLease(ctx, target(server), testPolicy(t).Lease, time.Hour, "request")
	require.NoError(t, err)
	observed, err = leaseState(ctx, target(server), leaseID)
	require.NoError(t, err)
	require.Equal(t, Observed{State: LeaseOpen, RunID: fence.RunID, Started: time.Unix(1000, 0).UTC()}, observed)
	require.False(t, observed.State.Clean())

	for reason, state := range map[string]LeaseState{
		ReasonReleased: LeaseReleased, ReasonReconciled: LeaseReleased,
		"terminated by hand": LeaseClosedOtherwise, "": LeaseClosedOtherwise,
	} {
		server.runs[leaseID] = nil
		fence, err := takeLease(ctx, target(server), testPolicy(t).Lease, time.Hour, "request")
		require.NoError(t, err)
		require.NoError(t, terminate(ctx, target(server), &commonpb.WorkflowExecution{WorkflowId: leaseID, RunId: fence.RunID}, reason))
		observed, err := leaseState(ctx, target(server), leaseID)
		require.NoError(t, err)
		require.Equal(t, state, observed.State, "terminated with %q", reason)
	}

	server.runs[leaseID] = nil
	server.open(leaseID, time.Unix(2000, 0))
	server.timeOut(leaseID)
	observed, err = leaseState(ctx, target(server), leaseID)
	require.NoError(t, err)
	require.Equal(t, LeaseClosedOtherwise, observed.State, "a lease that reached its run timeout is unreconciled")

	server.runs[leaseID] = nil
	server.open(leaseID, time.Unix(2000, 0))
	server.finish(leaseID)
	observed, err = leaseState(ctx, target(server), leaseID)
	require.NoError(t, err)
	require.Equal(t, LeaseClosedOtherwise, observed.State, "a lease closed any way but a canary termination is unreconciled")

	server.fail["describe "+leaseID] = serviceerror.NewUnavailable("down")
	_, err = leaseState(ctx, target(server), leaseID)
	require.Error(t, err, "an unreadable lease is not a clean one")
}

// A lease is taken only when no run is open: a collision names the run that holds it.
func TestTakeLeaseFailsOnAnyOpenRun(t *testing.T) {
	ctx := t.Context()
	server := newFakeServer()
	lease := testPolicy(t).Lease
	fence, err := takeLease(ctx, target(server), lease, time.Hour, "first")
	require.NoError(t, err)
	require.Equal(t, lease.WorkflowID, fence.WorkflowID)

	_, err = takeLease(ctx, target(server), lease, time.Hour, "second")
	var held *LeaseHeldError
	require.ErrorAs(t, err, &held)
	require.Equal(t, fence.RunID, held.RunID)

	server.fail["start "+lease.WorkflowID] = serviceerror.NewUnavailable("down")
	_, err = takeLease(ctx, target(server), lease, time.Hour, "third")
	require.Error(t, err)
	require.NotErrorAs(t, err, &held)
}

// The fence's signals are the list cleanup acts on: sent to the exact fence run, so a stale fence
// fails, and read back in order, once each, across pages.
func TestTheFenceSignalsNameEveryRunOnce(t *testing.T) {
	ctx := t.Context()
	server := newFakeServer()
	lease := testPolicy(t).Lease
	fence, err := takeLease(ctx, target(server), lease, time.Hour, "request")
	require.NoError(t, err)
	for _, id := range []string{runID(1), runID(2), runID(1), runID(3)} {
		require.NoError(t, signalRunOpened(ctx, target(server), fence, id))
	}
	ids, err := fencedIDs(ctx, target(server), fence)
	require.NoError(t, err)
	require.Equal(t, []string{runID(1), runID(2), runID(3)}, ids)

	stale := Fence{WorkflowID: fence.WorkflowID, RunID: "run-stale"}
	require.Error(t, signalRunOpened(ctx, target(server), stale, runID(4)), "a stale fence fails the signal")
	require.NoError(t, terminate(ctx, target(server), &commonpb.WorkflowExecution{WorkflowId: fence.WorkflowID, RunId: fence.RunID}, ReasonReleased))
	require.Error(t, signalRunOpened(ctx, target(server), fence, runID(4)), "a released fence fails the signal")

	other, err := takeLease(ctx, target(server), lease, time.Hour, "next")
	require.NoError(t, err)
	server.mu.Lock()
	server.find(&commonpb.WorkflowExecution{WorkflowId: lease.WorkflowID, RunId: other.RunID}).events = append(
		server.find(&commonpb.WorkflowExecution{WorkflowId: lease.WorkflowID, RunId: other.RunID}).events,
		&historypb.HistoryEvent{EventType: enumspb.EVENT_TYPE_WORKFLOW_EXECUTION_SIGNALED,
			Attributes: &historypb.HistoryEvent_WorkflowExecutionSignaledEventAttributes{WorkflowExecutionSignaledEventAttributes: &historypb.WorkflowExecutionSignaledEventAttributes{
				SignalName: SignalRunOpened,
			}}})
	server.mu.Unlock()
	_, err = fencedIDs(ctx, target(server), other)
	require.ErrorContains(t, err, "names no Testpilot Run", "a fence signal with no ID fails closed")

	require.NoError(t, terminate(ctx, target(server), &commonpb.WorkflowExecution{WorkflowId: lease.WorkflowID, RunId: other.RunID}, ReasonReleased))
	foreign, err := takeLease(ctx, target(server), lease, time.Hour, "foreign")
	require.NoError(t, err)
	require.NoError(t, signalRunOpened(ctx, target(server), foreign, "customer-workflow"))
	_, err = fencedIDs(ctx, target(server), foreign)
	require.ErrorContains(t, err, "names no Testpilot Run", "a signal naming anything but a Testpilot Run fails closed")
}

func TestFencedIDsStopAtTheEventBound(t *testing.T) {
	ctx := t.Context()
	server := newFakeServer()
	fence, err := takeLease(ctx, target(server), testPolicy(t).Lease, time.Hour, "request")
	require.NoError(t, err)
	input, err := converter.GetDefaultDataConverter().ToPayloads(runID(1))
	require.NoError(t, err)
	run := server.find(&commonpb.WorkflowExecution{WorkflowId: fence.WorkflowID, RunId: fence.RunID})
	for len(run.events) <= maxLeaseEvents {
		run.events = append(run.events, &historypb.HistoryEvent{EventType: enumspb.EVENT_TYPE_WORKFLOW_EXECUTION_SIGNALED,
			Attributes: &historypb.HistoryEvent_WorkflowExecutionSignaledEventAttributes{WorkflowExecutionSignaledEventAttributes: &historypb.WorkflowExecutionSignaledEventAttributes{
				SignalName: SignalRunOpened, Input: input,
			}}})
	}
	_, err = fencedIDs(ctx, target(server), fence)
	require.ErrorContains(t, err, "exceeds", "a lease history past the bound is not one the canary made")
}

// A fenced workflow is closed when the server reports it closed, or when it does not find it on
// two reads a pause apart; one found the second time is judged by its status.
func TestWorkflowClosed(t *testing.T) {
	ctx := t.Context()
	server := newFakeServer()
	server.open("running", time.Unix(1, 0))
	server.open("finished", time.Unix(1, 0))
	server.finish("finished")

	closed, err := workflowClosed(ctx, target(server), "running", time.Second, noWait)
	require.NoError(t, err)
	require.False(t, closed)
	closed, err = workflowClosed(ctx, target(server), "finished", time.Second, noWait)
	require.NoError(t, err)
	require.True(t, closed)

	paused := 0
	closed, err = workflowClosed(ctx, target(server), "never-started", time.Second, func(context.Context, time.Duration) error {
		paused++
		return nil
	})
	require.NoError(t, err)
	require.True(t, closed, "not found twice is never started")
	require.Equal(t, 1, paused)

	closed, err = workflowClosed(ctx, target(server), "late", time.Second, func(context.Context, time.Duration) error {
		server.open("late", time.Unix(1, 0))
		return nil
	})
	require.NoError(t, err)
	require.False(t, closed, "a start that lands between the reads is open")

	_, err = workflowClosed(ctx, target(server), "never-started", time.Second, func(context.Context, time.Duration) error {
		return context.Canceled
	})
	require.ErrorIs(t, err, context.Canceled)

	server.fail["describe running"] = serviceerror.NewUnavailable("down")
	_, err = workflowClosed(ctx, target(server), "running", time.Second, noWait)
	require.Error(t, err)
}
