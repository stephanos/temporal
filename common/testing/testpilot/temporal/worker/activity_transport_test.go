package worker

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	activitypb "go.temporal.io/api/activity/v1"
	commonpb "go.temporal.io/api/common/v1"
	enumspb "go.temporal.io/api/enums/v1"
	failurepb "go.temporal.io/api/failure/v1"
	namespacepb "go.temporal.io/api/namespace/v1"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/converter"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/await"
	"go.temporal.io/server/common/testing/testpilot"
	"go.temporal.io/server/common/testing/testpilot/internal/testsupport"
	"go.temporal.io/server/common/testing/testpilot/internal/testsupport/facadetest"
	"go.temporal.io/server/common/testing/testpilot/temporal/internal/delivery"
	"go.temporal.io/server/common/testing/testpilot/temporal/internal/primitive"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// frontend is the WorkflowService a real SDK worker talks to in these tests: it hands the worker
// the activity tasks a test queues and keeps every answer the worker sends back, so a test reads
// what Temporal is told about an attempt rather than what the activity function returned.
type frontend struct {
	workflowservice.UnimplementedWorkflowServiceServer
	tasks     chan *workflowservice.PollActivityTaskQueueResponse
	completed chan *workflowservice.RespondActivityTaskCompletedRequest
	failed    chan *workflowservice.RespondActivityTaskFailedRequest
	canceled  chan *workflowservice.RespondActivityTaskCanceledRequest
	// closed is closed by the test when the server would report the activity closed.
	closed chan struct{}
	polled chan *workflowservice.PollActivityExecutionRequest
	// expired is how many long polls the server still answers with no outcome, as it does when a
	// poll expires while the activity runs.
	expired atomic.Int32
	// closedRun is the run the server reports the outcome of; unset, the run it was asked about.
	closedRun string
	// heartbeats are the heartbeats the worker sent, and cancelRequested is what the server answers
	// each with: whether the activity's cancellation is requested.
	heartbeats      chan *workflowservice.RecordActivityTaskHeartbeatRequest
	cancelRequested atomic.Bool
}

func (f *frontend) RecordActivityTaskHeartbeat(_ context.Context, request *workflowservice.RecordActivityTaskHeartbeatRequest) (*workflowservice.RecordActivityTaskHeartbeatResponse, error) {
	requested := f.cancelRequested.Load()
	select {
	case f.heartbeats <- request:
	default:
	}
	return &workflowservice.RecordActivityTaskHeartbeatResponse{CancelRequested: requested}, nil
}

// The server long-polls for the activity's outcome and answers once the activity closed.
func (f *frontend) PollActivityExecution(ctx context.Context, request *workflowservice.PollActivityExecutionRequest) (*workflowservice.PollActivityExecutionResponse, error) {
	f.polled <- request
	if f.expired.Add(-1) >= 0 {
		return &workflowservice.PollActivityExecutionResponse{RunId: request.GetRunId()}, nil
	}
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-f.closed:
		runID := request.GetRunId()
		if f.closedRun != "" {
			runID = f.closedRun
		}
		return &workflowservice.PollActivityExecutionResponse{RunId: runID, Outcome: &activitypb.ActivityExecutionOutcome{}}, nil
	}
}

func (*frontend) GetSystemInfo(context.Context, *workflowservice.GetSystemInfoRequest) (*workflowservice.GetSystemInfoResponse, error) {
	return &workflowservice.GetSystemInfoResponse{Capabilities: &workflowservice.GetSystemInfoResponse_Capabilities{}}, nil
}

func (*frontend) DescribeNamespace(_ context.Context, request *workflowservice.DescribeNamespaceRequest) (*workflowservice.DescribeNamespaceResponse, error) {
	return &workflowservice.DescribeNamespaceResponse{NamespaceInfo: &namespacepb.NamespaceInfo{Name: request.GetNamespace(), State: enumspb.NAMESPACE_STATE_REGISTERED}}, nil
}

// The worker long-polls every task kind its registration starts; only activity tasks are served.
func (*frontend) PollWorkflowTaskQueue(ctx context.Context, _ *workflowservice.PollWorkflowTaskQueueRequest) (*workflowservice.PollWorkflowTaskQueueResponse, error) {
	<-ctx.Done()
	return &workflowservice.PollWorkflowTaskQueueResponse{}, nil
}

func (*frontend) PollNexusTaskQueue(ctx context.Context, _ *workflowservice.PollNexusTaskQueueRequest) (*workflowservice.PollNexusTaskQueueResponse, error) {
	<-ctx.Done()
	return &workflowservice.PollNexusTaskQueueResponse{}, nil
}

func (f *frontend) PollActivityTaskQueue(ctx context.Context, _ *workflowservice.PollActivityTaskQueueRequest) (*workflowservice.PollActivityTaskQueueResponse, error) {
	select {
	case <-ctx.Done():
		return &workflowservice.PollActivityTaskQueueResponse{}, nil
	case task := <-f.tasks:
		return task, nil
	}
}

func (f *frontend) RespondActivityTaskCompleted(_ context.Context, request *workflowservice.RespondActivityTaskCompletedRequest) (*workflowservice.RespondActivityTaskCompletedResponse, error) {
	f.completed <- request
	return &workflowservice.RespondActivityTaskCompletedResponse{}, nil
}

func (f *frontend) RespondActivityTaskFailed(_ context.Context, request *workflowservice.RespondActivityTaskFailedRequest) (*workflowservice.RespondActivityTaskFailedResponse, error) {
	f.failed <- request
	return &workflowservice.RespondActivityTaskFailedResponse{}, nil
}

func (f *frontend) RespondActivityTaskCanceled(_ context.Context, request *workflowservice.RespondActivityTaskCanceledRequest) (*workflowservice.RespondActivityTaskCanceledResponse, error) {
	f.canceled <- request
	return &workflowservice.RespondActivityTaskCanceledResponse{}, nil
}

func (*frontend) ShutdownWorker(context.Context, *workflowservice.ShutdownWorkerRequest) (*workflowservice.ShutdownWorkerResponse, error) {
	return &workflowservice.ShutdownWorkerResponse{}, nil
}

// sent is the one answer the worker sent for a task: which of the three responses, and what it
// carried.
type sent struct {
	Response string
	Token    string
	Result   *testpilotspb.Value
	Failure  *failurepb.Failure
}

func (f *frontend) answer(t *testing.T) sent {
	t.Helper()
	timeout, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()
	select {
	case <-timeout.Done():
		require.FailNow(t, "the worker answered no activity task")
		return sent{}
	case request := <-f.completed:
		result := &testpilotspb.Value{}
		require.NoError(t, converter.GetDefaultDataConverter().FromPayloads(request.GetResult(), result))
		return sent{Response: "completed", Token: string(request.GetTaskToken()), Result: result}
	case request := <-f.failed:
		return sent{Response: "failed", Token: string(request.GetTaskToken()), Failure: request.GetFailure()}
	case request := <-f.canceled:
		return sent{Response: "canceled", Token: string(request.GetTaskToken())}
	}
}

func requireAnswer(t *testing.T, want, got sent) {
	t.Helper()
	require.Equal(t, want.Response, got.Response, got)
	require.Equal(t, want.Token, got.Token)
	require.True(t, proto.Equal(want.Result, got.Result), got.Result)
	require.True(t, proto.Equal(want.Failure, got.Failure), got.Failure)
}

func tokenDigest(token string) string {
	digest := sha256.Sum256([]byte(token))
	return hex.EncodeToString(digest[:])
}

// transportSession is a real Driver over a real SDK client and worker, polling the frontend, with
// one open Session whose activity start has been carried. task builds the activity task the server
// would dispatch for an attempt of that start.
func transportSession(t *testing.T, shape func(*testpilotspb.Program)) (*frontend, *Session, func(token string, attempt int32) *workflowservice.PollActivityTaskQueueResponse) {
	t.Helper()
	server := &frontend{
		tasks:     make(chan *workflowservice.PollActivityTaskQueueResponse, 8),
		completed: make(chan *workflowservice.RespondActivityTaskCompletedRequest, 8),
		failed:    make(chan *workflowservice.RespondActivityTaskFailedRequest, 8),
		canceled:  make(chan *workflowservice.RespondActivityTaskCanceledRequest, 8),
		closed:    make(chan struct{}), polled: make(chan *workflowservice.PollActivityExecutionRequest, 8),
		heartbeats: make(chan *workflowservice.RecordActivityTaskHeartbeatRequest, 64),
	}
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	transport := grpc.NewServer()
	workflowservice.RegisterWorkflowServiceServer(transport, server)
	go func() { _ = transport.Serve(listener) }()
	t.Cleanup(transport.Stop)
	sdk, err := client.Dial(client.Options{HostPort: listener.Addr().String(), Namespace: "namespace"})
	require.NoError(t, err)
	t.Cleanup(sdk.Close)

	prepared := preparedActivityFixture(t, shape, func(profile *testpilot.ProfileSpec) {
		profile.Opcodes = append(profile.Opcodes, testpilot.ActivityAttemptWithholding)
	})
	catalog, err := testpilot.NewCatalog(testsupport.DescriptorClosure(workflowservice.File_temporal_api_workflowservice_v1_service_proto))
	require.NoError(t, err)
	host, err := New(Options{
		Profile: testpilot.ProfileSpec{
			Identity: "profile", Catalog: catalog, ProgramLimits: prepared.Limits(),
			EnvironmentBindings: []testpilot.EnvironmentBinding{{ID: "namespace", Value: "namespace"}, {ID: "task-queue", Value: "task-queue"}},
			Roles: []testpilot.RolePolicy{
				{ID: "endpoint", Kind: testpilotspb.ROLE_KIND_ENDPOINT, Methods: []string{primitive.StartWorkflowPath, delivery.StartActivityPath}},
				{ID: "worker", Kind: testpilotspb.ROLE_KIND_WORKER}, {ID: "queue", Kind: testpilotspb.ROLE_KIND_TASK_QUEUE},
			},
		},
		Client: sdk, WorkerRoleID: "worker", WorkerStopTimeout: time.Second,
	})
	require.NoError(t, err)
	session, err := host.OpenSession(t.Context(), "run", prepared, SessionOptions{Bridge: newTestBridge()})
	require.NoError(t, err)
	// A pooled worker outlives its Runs, so the test stops the one it started.
	t.Cleanup(func() {
		for _, group := range host.registry.groups {
			group.worker.Stop()
		}
	})
	_, request := carryActivityStart(t, session, prepared, activityBinding("activity-id"), "activity-run", delivery.TriggerSucceeded)
	task := func(token string, attempt int32) *workflowservice.PollActivityTaskQueueResponse {
		now := timestamppb.Now()
		// The server names a standalone activity's namespace in workflow_namespace
		// (chasm/lib/activity/activity.go), and the SDK reads the activity's namespace from it.
		return &workflowservice.PollActivityTaskQueueResponse{
			TaskToken: []byte(token), WorkflowNamespace: "namespace", ActivityId: "activity-id", ActivityType: &commonpb.ActivityType{Name: "activity-type"}, ActivityRunId: "activity-run",
			Attempt: attempt, Header: request.GetHeader(), ScheduledTime: now, CurrentAttemptScheduledTime: now, StartedTime: now,
			ScheduleToCloseTimeout: durationpb.New(time.Minute), StartToCloseTimeout: durationpb.New(time.Minute),
		}
	}
	return server, session, task
}

// Through a real SDK worker, Temporal is told what the script declares for each attempt: the first
// attempt's failure, as retryable as the script made it, and then the retry's completion. A
// redelivery of the retry is answered with that completion again and settles nothing further.
func TestTransportTemporalIsToldWhatEachDeclaredAttemptDoes(t *testing.T) {
	server, session, task := transportSession(t, retriedActivity)

	// The failure Temporal receives is the one the script wrote, field for field.
	server.tasks <- task("token-1", 1)
	requireAnswer(t, sent{Response: "failed", Token: "token-1", Failure: &failurepb.Failure{
		Message:     "not yet",
		FailureInfo: &failurepb.Failure_ApplicationFailureInfo{ApplicationFailureInfo: &failurepb.ApplicationFailureInfo{Type: "transient"}},
	}}, server.answer(t))
	first, err := settledAttempt(t, session, "reservation-1")
	require.NoError(t, err)
	requireOutcome(t, answered("activity-run", 1, tokenDigest("token-1"), failedRetryable), first)

	completion := sent{Response: "completed", Token: "token-2", Result: textResult("done")}
	server.tasks <- task("token-2", 2)
	requireAnswer(t, completion, server.answer(t))
	second, err := settledAttempt(t, session, "reservation-2")
	require.NoError(t, err)
	requireOutcome(t, answered("activity-run", 2, tokenDigest("token-2"), completed), second)

	server.tasks <- task("token-2", 2)
	requireAnswer(t, completion, server.answer(t))
	require.Empty(t, server.canceled)
	require.NoError(t, session.Close(t.Context()))
}

func TestTransportActivityHeartbeatRunsInItsAttempt(t *testing.T) {
	details := &commonpb.Payloads{Payloads: []*commonpb.Payload{
		{Metadata: map[string][]byte{"encoding": []byte("binary/plain")}, Data: []byte{1, 2, 3}},
		{Metadata: map[string][]byte{"encoding": []byte("json/plain")}, Data: []byte(`"heartbeat"`)},
	}}
	for _, pending := range []bool{false, true} {
		t.Run(map[bool]string{false: "complete", true: "pending then retry"}[pending], func(t *testing.T) {
			server, session, task := transportSession(t, func(program *testpilotspb.Program) {
				standaloneActivity(program)
				program.Entrypoints[0].Instructions[0].Instruction.GetInvokeRpc().RequestAssignments = append(
					program.Entrypoints[0].Instructions[0].Instruction.GetInvokeRpc().RequestAssignments,
					durationAssignment("heartbeat_timeout", 1),
				)
				heartbeat := &testpilotspb.InstructionNode{InstructionId: "heartbeat", Limits: facadetest.Bounds(), Instruction: &testpilotspb.Instruction{Instruction: &testpilotspb.Instruction_ActivityHeartbeat{ActivityHeartbeat: &testpilotspb.ActivityHeartbeat{Details: proto.CloneOf(details)}}}}
				finish := program.Entrypoints[1].Instructions[0]
				program.Entrypoints[1].Instructions = []*testpilotspb.InstructionNode{heartbeat, finish}
				if pending {
					withheld := &testpilotspb.InstructionNode{InstructionId: "pending", Limits: facadetest.Bounds(), Instruction: &testpilotspb.Instruction{Instruction: &testpilotspb.Instruction_ActivityAttemptWithholding{ActivityAttemptWithholding: &testpilotspb.ActivityAttemptWithholding{Mode: testpilotspb.ACTIVITY_WITHHOLDING_MODE_SDK_PENDING}}}}
					program.Entrypoints[1].Instructions = []*testpilotspb.InstructionNode{heartbeat, withheld, finish}
				}
			})
			firstTask := task("token-1", 1)
			firstTask.HeartbeatTimeout = durationpb.New(time.Second)
			server.tasks <- firstTask
			bounded, cancel := context.WithTimeout(t.Context(), 10*time.Second)
			defer cancel()
			select {
			case <-bounded.Done():
				require.FailNow(t, "the worker sent no declared heartbeat")
			case heartbeat := <-server.heartbeats:
				require.Equal(t, []byte("token-1"), heartbeat.GetTaskToken())
				require.True(t, proto.Equal(details, heartbeat.GetDetails()), heartbeat)
			}
			require.NoError(t, session.reservations["reservation-1"].Drain(bounded))
			first, err := settledAttempt(t, session, "reservation-1")
			require.NoError(t, err)
			response := testpilotspb.ACTIVITY_ATTEMPT_RESPONSE_OFFERED_COMPLETED
			if pending {
				response = testpilotspb.ACTIVITY_ATTEMPT_RESPONSE_PENDING
				require.Empty(t, server.completed)
				require.Empty(t, server.failed)
				require.Empty(t, server.canceled)
				replay := delivery.ActivityDelivery{Header: firstTask.Header, Namespace: "namespace", ActivityID: "activity-id", ActivityType: "activity-type", TaskQueue: "task-queue", ActivityRunID: "activity-run", Attempt: 1, DeliveryID: tokenDigest("token-1")}
				result, replayErr := activityWorker(session.host, session.definition).activateActivity(bounded, replay, func(context.Context) (interface{}, error) {
					t.Fatal("a duplicate pending delivery executed the activity again")
					return nil, nil
				})
				require.Nil(t, result)
				require.Same(t, activity.ErrResultPending, replayErr)
				server.tasks <- task("token-2", 2)
				requireAnswer(t, sent{Response: "completed", Token: "token-2", Result: textResult("done")}, server.answer(t))
				second, err := settledAttempt(t, session, "reservation-2")
				require.NoError(t, err)
				requireOutcome(t, answered("activity-run", 2, tokenDigest("token-2"), completed), second)
			} else {
				requireAnswer(t, sent{Response: "completed", Token: "token-1", Result: textResult("done")}, server.answer(t))
				server.tasks <- firstTask
				requireAnswer(t, sent{Response: "completed", Token: "token-1", Result: textResult("done")}, server.answer(t))
			}
			want := answered("activity-run", 1, tokenDigest("token-1"), response)
			want.ActivityAttempt.HeartbeatInvoked = true
			requireOutcome(t, want, first)
			require.NoError(t, session.Close(bounded))
			require.Empty(t, server.heartbeats)
			require.Empty(t, server.completed)
			require.Empty(t, server.failed)
			require.Empty(t, server.canceled)
		})
	}
}

func TestTransportUnarmedPendingIsRefusedBeforeHeartbeat(t *testing.T) {
	server, session, task := transportSession(t, func(program *testpilotspb.Program) {
		standaloneActivity(program)
		start := program.Entrypoints[0].Instructions[0].Instruction.GetInvokeRpc()
		start.RequestAssignments = append(start.RequestAssignments, durationAssignment("heartbeat_timeout", 1))
		program.Entrypoints[1].Instructions = []*testpilotspb.InstructionNode{
			{InstructionId: "heartbeat", Limits: facadetest.Bounds(), Instruction: &testpilotspb.Instruction{Instruction: &testpilotspb.Instruction_ActivityHeartbeat{ActivityHeartbeat: &testpilotspb.ActivityHeartbeat{Details: &commonpb.Payloads{}}}}},
			{InstructionId: "pending", Limits: facadetest.Bounds(), Instruction: &testpilotspb.Instruction{Instruction: &testpilotspb.Instruction_ActivityAttemptWithholding{ActivityAttemptWithholding: &testpilotspb.ActivityAttemptWithholding{Mode: testpilotspb.ACTIVITY_WITHHOLDING_MODE_SDK_PENDING}}}},
		}
	})
	// The request declared a positive heartbeat timeout; the actual delivery did not carry it.
	server.tasks <- task("unarmed-token", 1)
	answer := server.answer(t)
	require.Equal(t, "failed", answer.Response)
	require.Equal(t, "unarmed-token", answer.Token)
	require.Equal(t, activationErrorType, answer.Failure.GetApplicationFailureInfo().GetType())
	bounded, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()
	require.NoError(t, session.reservations["reservation-1"].Drain(bounded))
	outcome, err := settledAttempt(t, session, "reservation-1")
	require.NoError(t, err)
	require.Equal(t, testpilotspb.ACTIVITY_ATTEMPT_RESPONSE_REFUSED, outcome.Outcome.GetActivityAttempt().GetResponse())
	require.False(t, outcome.Outcome.GetActivityAttempt().GetHeartbeatInvoked())
	require.Empty(t, server.heartbeats)
	require.NoError(t, session.Close(bounded))
}

// The worker offers a non-retryable declared failure and then asks the server, by the activity's
// names, for the activity's outcome. The reservation of the second attempt the script declares
// stays reserved while the server has not answered, and settles as not needed once it reports the
// activity closed, instead of waiting for an attempt the server will never send.
func TestTransportTheServerClosingTheActivityReleasesTheLaterAttempts(t *testing.T) {
	server, session, task := transportSession(t, endedByFailureActivity)
	server.expired.Store(1)

	server.tasks <- task("token-1", 1)
	requireAnswer(t, sent{Response: "failed", Token: "token-1", Failure: &failurepb.Failure{
		Message:     "not yet",
		FailureInfo: &failurepb.Failure_ApplicationFailureInfo{ApplicationFailureInfo: &failurepb.ApplicationFailureInfo{Type: "refusal", NonRetryable: true}},
	}}, server.answer(t))
	first, err := settledAttempt(t, session, "reservation-1")
	require.NoError(t, err)
	requireOutcome(t, answered("activity-run", 1, tokenDigest("token-1"), failedNonRetryable), first)
	bounded, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()
	// The first poll expires with no outcome, which says nothing: the worker asks again.
	for range 2 {
		select {
		case <-bounded.Done():
			require.FailNow(t, "the worker did not ask the server for the activity's outcome")
		case poll := <-server.polled:
			require.True(t, proto.Equal(&workflowservice.PollActivityExecutionRequest{Namespace: "namespace", ActivityId: "activity-id", RunId: "activity-run"}, poll), poll)
		}
	}
	raw := session.reservations["reservation-2"]
	require.False(t, raw.settled())

	close(server.closed)
	require.NoError(t, raw.Drain(bounded))
	second, err := settledAttempt(t, session, "reservation-2")
	require.NoError(t, err)
	requireOutcome(t, notNeeded("activity-run"), second)
	require.False(t, raw.consumed)
}

// An outcome the server reports for another run of the activity says nothing about the run that was
// started. The later attempt keeps its reservation, and the Session says the answer was crossed.
func TestTransportAnOutcomeOfAnotherRunReleasesNothing(t *testing.T) {
	server, session, task := transportSession(t, endedByFailureActivity)
	diagnostics := captureDiagnostics(t, session)
	server.closedRun = "another-run"
	close(server.closed)

	server.tasks <- task("token-1", 1)
	require.Equal(t, "failed", server.answer(t).Response)
	bounded, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()
	await.Require(bounded, t, func(t *await.T) {
		require.Equal(t, []diagnosed{{
			Kind: testpilotspb.RUN_DIAGNOSTIC_KIND_INVARIANT, Code: "activity_closure_crossed",
			Detail: `activity run "activity-run": the server reported the outcome of run "another-run"`,
		}}, diagnostics.all())
	}, 10*time.Second, 5*time.Millisecond)
	require.False(t, session.reservations["reservation-2"].settled())
}

// When the Run cancels an attempt under way, Temporal is told the attempt failed, with the Driver's
// own non-retryable application failure, and never that it was canceled: the server asked for no
// cancellation. The reservation records exactly that answer.
func TestTransportACanceledAttemptIsReportedAndRecordedAsFailed(t *testing.T) {
	server, session, task := transportSession(t, standaloneActivity)
	// Holding the script keeps the admitted attempt under way until the Run has canceled it.
	script := &activityScript{}
	session.activityScripts[activityScriptKey{origin: activityOrigin("run"), entrypointID: "activity"}] = script
	script.mu.Lock()
	raw := session.reservations["reservation-1"]

	server.tasks <- task("token-1", 1)
	select {
	case <-raw.bound:
	case got := <-server.failed:
		script.mu.Unlock()
		require.FailNow(t, "the attempt was answered before it was admitted", got)
	}
	require.NoError(t, raw.Cancel(t.Context()))
	script.mu.Unlock()

	got := server.answer(t)
	require.Equal(t, "failed", got.Response)
	require.Equal(t, "token-1", got.Token)
	require.Equal(t, "testpilot worker activation", got.Failure.GetMessage())
	require.True(t, proto.Equal(&failurepb.ApplicationFailureInfo{Type: "umpire_worker", NonRetryable: true}, got.Failure.GetApplicationFailureInfo()), got.Failure)
	require.Equal(t, "context canceled", got.Failure.GetCause().GetMessage())
	require.Empty(t, server.canceled)
	require.Empty(t, server.completed)

	settled, err := settledActivity(t, session)
	require.NoError(t, err)
	requireOutcome(t, refused("activity-run", 1, tokenDigest("token-1"), "context canceled"), settled)
}

// heartbeat waits for a heartbeat of the task from the worker.
func (f *frontend) heartbeat(t *testing.T, token string) {
	t.Helper()
	timeout, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()
	select {
	case <-timeout.Done():
		require.FailNow(t, "the worker sent no heartbeat")
	case request := <-f.heartbeats:
		require.Equal(t, token, string(request.GetTaskToken()))
	}
}

// Through a real SDK worker, an attempt whose instruction answers canceled heartbeats to learn
// whether the server asks for the cancellation, and answers nothing while it does not. Once the
// server's answer to a heartbeat says the cancellation is requested, the SDK tells Temporal the
// attempt is canceled, which it does for no cancellation the server did not ask for, and the
// reservation records that offer. A redelivery of the attempt is answered canceled again, once
// the server has asked it too.
func TestTransportTemporalIsToldAnAttemptIsCanceledOnceTheServerAsked(t *testing.T) {
	server, session, task := transportSession(t, canceledActivity)
	raw := session.reservations["reservation-1"]

	server.tasks <- task("token-1", 1)
	server.heartbeat(t, "token-1")
	require.False(t, raw.settled())
	require.Empty(t, server.canceled)
	require.Empty(t, server.failed)

	server.cancelRequested.Store(true)
	requireAnswer(t, sent{Response: "canceled", Token: "token-1"}, server.answer(t))
	first, err := settledAttempt(t, session, "reservation-1")
	require.NoError(t, err)
	requireOutcome(t, answered("activity-run", 1, tokenDigest("token-1"), canceledAnswer), first)
	// The offer releases nothing: the declared retry stays reserved until the server closes the
	// activity.
	require.False(t, session.reservations["reservation-2"].settled())

	server.tasks <- task("token-1-again", 1)
	requireAnswer(t, sent{Response: "canceled", Token: "token-1-again"}, server.answer(t))
	require.Empty(t, server.failed)
	require.Empty(t, server.completed)
	require.NoError(t, session.Close(t.Context()))
}

// An attempt that waits for a cancellation the server never asks for answers nothing canceled. When
// the Run cancels it, Temporal is told the attempt failed with the Driver's own non-retryable
// failure, and the reservation records that refusal.
func TestTransportACancellationTheServerNeverAsksForIsRefused(t *testing.T) {
	server, session, task := transportSession(t, canceledActivity)
	raw := session.reservations["reservation-1"]

	server.tasks <- task("token-1", 1)
	server.heartbeat(t, "token-1")
	require.NoError(t, raw.Cancel(t.Context()))

	got := server.answer(t)
	require.Equal(t, "failed", got.Response)
	require.Equal(t, "token-1", got.Token)
	require.True(t, proto.Equal(&failurepb.ApplicationFailureInfo{Type: "umpire_worker", NonRetryable: true}, got.Failure.GetApplicationFailureInfo()), got.Failure)
	require.Equal(t, "context canceled", got.Failure.GetCause().GetMessage())
	require.Empty(t, server.canceled)
	require.Empty(t, server.completed)

	settled, err := settledAttempt(t, session, "reservation-1")
	require.NoError(t, err)
	requireOutcome(t, refused("activity-run", 1, tokenDigest("token-1"), "context canceled"), settled)
}
