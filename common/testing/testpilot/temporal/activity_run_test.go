package temporal_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	activitypb "go.temporal.io/api/activity/v1"
	enumspb "go.temporal.io/api/enums/v1"
	failurepb "go.temporal.io/api/failure/v1"
	namespacepb "go.temporal.io/api/namespace/v1"
	"go.temporal.io/api/serviceerror"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/client"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/await"
	"go.temporal.io/server/common/testing/testpilot"
	"go.temporal.io/server/common/testing/testpilot/temporal"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// activityServer is the WorkflowService a Run's controller and SDK worker both talk to. It treats a
// standalone activity as the server does: a start dispatches the first attempt, an attempt failed
// retryably is followed by the next one unless the retry policy is exhausted, an attempt that
// completes or fails non-retryably closes the activity, and a long poll for the activity's outcome
// is answered once it closed. It can also lose the first answer it is sent, as a failed send or an
// attempt it already timed out would.
type activityServer struct {
	workflowservice.UnimplementedWorkflowServiceServer
	tasks  chan *workflowservice.PollActivityTaskQueueResponse
	closed chan struct{}
	// loses is what the server does with the first answer: nothing, or it rejects it and then
	// redelivers the same attempt or issues the next.
	loses string
	// noRetry closes the activity on a retryable failure, as an exhausted retry policy does.
	noRetry bool

	mu         sync.Mutex
	start      *workflowservice.StartActivityExecutionRequest
	attempt    int32
	deliveries int
	answers    []string
	// cancelRequested is set once a cancellation of the activity is requested; the server then says
	// so in answer to a heartbeat, and accepts a canceled answer.
	cancelRequested bool
}

const (
	losesAndRedelivers    = "redelivers the attempt"
	losesAndIssuesTheNext = "issues the next attempt"
)

func (*activityServer) GetSystemInfo(context.Context, *workflowservice.GetSystemInfoRequest) (*workflowservice.GetSystemInfoResponse, error) {
	return &workflowservice.GetSystemInfoResponse{Capabilities: &workflowservice.GetSystemInfoResponse_Capabilities{}}, nil
}

func (*activityServer) DescribeNamespace(_ context.Context, request *workflowservice.DescribeNamespaceRequest) (*workflowservice.DescribeNamespaceResponse, error) {
	return &workflowservice.DescribeNamespaceResponse{NamespaceInfo: &namespacepb.NamespaceInfo{Name: request.GetNamespace(), State: enumspb.NAMESPACE_STATE_REGISTERED}}, nil
}

// The worker long-polls every task kind its registration starts; only activity tasks are served.
func (*activityServer) PollWorkflowTaskQueue(ctx context.Context, _ *workflowservice.PollWorkflowTaskQueueRequest) (*workflowservice.PollWorkflowTaskQueueResponse, error) {
	<-ctx.Done()
	return &workflowservice.PollWorkflowTaskQueueResponse{}, nil
}

func (*activityServer) PollNexusTaskQueue(ctx context.Context, _ *workflowservice.PollNexusTaskQueueRequest) (*workflowservice.PollNexusTaskQueueResponse, error) {
	<-ctx.Done()
	return &workflowservice.PollNexusTaskQueueResponse{}, nil
}

func (s *activityServer) PollActivityTaskQueue(ctx context.Context, _ *workflowservice.PollActivityTaskQueueRequest) (*workflowservice.PollActivityTaskQueueResponse, error) {
	select {
	case <-ctx.Done():
		return &workflowservice.PollActivityTaskQueueResponse{}, nil
	case task := <-s.tasks:
		return task, nil
	}
}

func (*activityServer) ShutdownWorker(context.Context, *workflowservice.ShutdownWorkerRequest) (*workflowservice.ShutdownWorkerResponse, error) {
	return &workflowservice.ShutdownWorkerResponse{}, nil
}

// dispatchLocked queues a delivery of the started activity's current attempt, under a token that
// numbers the deliveries.
func (s *activityServer) dispatchLocked() {
	s.deliveries++
	now := timestamppb.Now()
	// The server names a standalone activity's namespace in workflow_namespace
	// (chasm/lib/activity/activity.go), and the SDK reads the activity's namespace from it.
	s.tasks <- &workflowservice.PollActivityTaskQueueResponse{
		TaskToken: fmt.Appendf(nil, "token-%d", s.deliveries), WorkflowNamespace: s.start.GetNamespace(), ActivityId: s.start.GetActivityId(), ActivityType: s.start.GetActivityType(), ActivityRunId: "activity-run",
		Attempt: s.attempt, Header: s.start.GetHeader(), ScheduledTime: now, CurrentAttemptScheduledTime: now, StartedTime: now,
		ScheduleToCloseTimeout: durationpb.New(time.Minute), StartToCloseTimeout: durationpb.New(time.Minute),
	}
}

func (s *activityServer) StartActivityExecution(_ context.Context, request *workflowservice.StartActivityExecutionRequest) (*workflowservice.StartActivityExecutionResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.start = request
	s.attempt = 1
	s.dispatchLocked()
	return &workflowservice.StartActivityExecutionResponse{RunId: "activity-run", Started: true}, nil
}

// lostLocked reports whether the server loses the answer it was just sent, and if so does what
// follows the loss.
func (s *activityServer) lostLocked() bool {
	if s.loses == "" || len(s.answers) > 0 {
		return false
	}
	s.answers = append(s.answers, "lost")
	if s.loses == losesAndIssuesTheNext {
		s.attempt++
	}
	s.dispatchLocked()
	return true
}

func (s *activityServer) PollActivityExecution(ctx context.Context, request *workflowservice.PollActivityExecutionRequest) (*workflowservice.PollActivityExecutionResponse, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-s.closed:
		return &workflowservice.PollActivityExecutionResponse{RunId: request.GetRunId(), Outcome: &activitypb.ActivityExecutionOutcome{}}, nil
	}
}

func (s *activityServer) RespondActivityTaskCompleted(context.Context, *workflowservice.RespondActivityTaskCompletedRequest) (*workflowservice.RespondActivityTaskCompletedResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.lostLocked() {
		return nil, serviceerror.NewNotFound("activity task not found")
	}
	s.answers = append(s.answers, "completed")
	close(s.closed)
	return &workflowservice.RespondActivityTaskCompletedResponse{}, nil
}

func (s *activityServer) RespondActivityTaskFailed(_ context.Context, request *workflowservice.RespondActivityTaskFailedRequest) (*workflowservice.RespondActivityTaskFailedResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.lostLocked() {
		return nil, serviceerror.NewNotFound("activity task not found")
	}
	application := request.GetFailure().GetApplicationFailureInfo()
	s.answers = append(s.answers, "failed:"+application.GetType())
	if application.GetNonRetryable() || s.noRetry {
		close(s.closed)
		return &workflowservice.RespondActivityTaskFailedResponse{}, nil
	}
	s.attempt++
	s.dispatchLocked()
	return &workflowservice.RespondActivityTaskFailedResponse{}, nil
}

func (s *activityServer) RequestCancelActivityExecution(context.Context, *workflowservice.RequestCancelActivityExecutionRequest) (*workflowservice.RequestCancelActivityExecutionResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cancelRequested = true
	return &workflowservice.RequestCancelActivityExecutionResponse{}, nil
}

func (s *activityServer) RecordActivityTaskHeartbeat(context.Context, *workflowservice.RecordActivityTaskHeartbeatRequest) (*workflowservice.RecordActivityTaskHeartbeatResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return &workflowservice.RecordActivityTaskHeartbeatResponse{CancelRequested: s.cancelRequested}, nil
}

// The server accepts a canceled answer only for an activity whose cancellation is requested, and
// the answer closes the activity.
func (s *activityServer) RespondActivityTaskCanceled(context.Context, *workflowservice.RespondActivityTaskCanceledRequest) (*workflowservice.RespondActivityTaskCanceledResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.cancelRequested {
		return nil, serviceerror.NewInvalidArgument("activity task cancellation was not requested")
	}
	s.answers = append(s.answers, "canceled")
	close(s.closed)
	return &workflowservice.RespondActivityTaskCanceledResponse{}, nil
}

func (s *activityServer) told() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]string(nil), s.answers...)
}

func attemptFinish(id string, result *testpilotspb.Expression) *testpilotspb.InstructionNode {
	return &testpilotspb.InstructionNode{InstructionId: id, Instruction: &testpilotspb.Instruction{Instruction: &testpilotspb.Instruction_Finish{Finish: &testpilotspb.Finish{Result: result}}}}
}

func attemptCancellation(id string) *testpilotspb.InstructionNode {
	return &testpilotspb.InstructionNode{InstructionId: id, Instruction: &testpilotspb.Instruction{Instruction: &testpilotspb.Instruction_ActivityAttemptCancellation{ActivityAttemptCancellation: &testpilotspb.ActivityAttemptCancellation{}}}}
}

func attemptFailure(id, failureType string, nonRetryable bool) *testpilotspb.InstructionNode {
	return &testpilotspb.InstructionNode{InstructionId: id, Instruction: &testpilotspb.Instruction{Instruction: &testpilotspb.Instruction_ActivityAttemptFailure{ActivityAttemptFailure: &testpilotspb.ActivityAttemptFailure{
		Failure: &failurepb.Failure{Message: "not yet", FailureInfo: &failurepb.Failure_ApplicationFailureInfo{ApplicationFailureInfo: &failurepb.ApplicationFailureInfo{Type: failureType, NonRetryable: nonRetryable}}},
	}}}}
}

func deliveryOf(token string) string {
	digest := sha256.Sum256([]byte(token))
	return hex.EncodeToString(digest[:])
}

// attemptFact is the reservation outcome of one activity attempt as the Run records it.
func attemptFact(status testpilotspb.InstructionOutcomeStatus, attempt int32, deliveryID string, response testpilotspb.ActivityAttemptResponse) *testpilotspb.InstructionOutcome {
	return &testpilotspb.InstructionOutcome{Status: status, ActivityAttempt: &testpilotspb.ActivityAttempt{ActivityRunId: "activity-run", SdkAttempt: attempt, DeliveryId: deliveryID, Response: response}}
}

// recordedAttempt is one activity attempt event of a Run, whole but for what the clock decides.
type recordedAttempt struct {
	Source  string
	Causes  []string
	Outcome string
}

// activityScriptCase is activityCase's activity alone: the controller starts it and the script
// declares its attempts.
func activityScriptCase(script []*testpilotspb.InstructionNode) *testpilotspb.Case {
	source := activityCase()
	source.Program.Entrypoints[0].Instructions = source.Program.Entrypoints[0].Instructions[1:]
	source.Program.Entrypoints[3].Instructions = script
	source.Program.Entrypoints = []*testpilotspb.Entrypoint{source.Program.Entrypoints[0], source.Program.Entrypoints[3]}
	source.Program.Roles = append(source.Program.Roles[:2:2], source.Program.Roles[3])
	return source
}

// runActivityScript prepares the Case under the Profile derived from it and runs it through
// PreparedCase.Run with the composite Driver and a real SDK worker, against the server.
func runActivityScript(t *testing.T, server *activityServer, source *testpilotspb.Case) (*testpilot.PreparedCase, *testpilotspb.Run, *testpilotspb.Verdict) {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	transport := grpc.NewServer()
	workflowservice.RegisterWorkflowServiceServer(transport, server)
	go func() { _ = transport.Serve(listener) }()
	t.Cleanup(transport.Stop)
	sdk, err := client.Dial(client.Options{HostPort: listener.Addr().String(), Namespace: "namespace"})
	require.NoError(t, err)
	t.Cleanup(sdk.Close)

	catalog, err := temporal.NewWorkflowServiceCatalog()
	require.NoError(t, err)
	profile, err := temporal.DeriveProfile(source, catalog, temporal.Environment{Identity: "activity", Namespace: "namespace", TaskQueue: "task-queue"})
	require.NoError(t, err)
	prepared, err := testpilot.Prepare(source, profile)
	require.NoError(t, err)
	driver, err := temporal.New(temporal.Options{
		Profile:         profile,
		ServerEndpoints: map[string]temporal.Endpoint{"workflow-service": {Target: listener.Addr().String(), Credentials: insecure.NewCredentials()}},
		SDKClient:       sdk, WorkerRoleID: "worker", WorkerStopTimeout: time.Second,
	})
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, driver.Close(context.Background())) })

	bounded, cancel := context.WithTimeout(t.Context(), 30*time.Second)
	defer cancel()
	run, verdict, err := prepared.Run(bounded, driver)
	require.NoError(t, err)
	return prepared, run, verdict
}

// A Run of a Case whose activity script declares its attempts, through PreparedCase.Run, the
// composite Driver, a real SDK worker and a server that retries, closes and loses answers as
// Temporal does. Temporal is told what each attempt's instruction says, and the Run records each
// declared attempt as a typed fact under the start that carried it, in attempt order: the
// activity run, the SDK attempt, the delivery and the answer the worker offered. A declared
// attempt the activity never reaches is recorded as not needed only once the server reports the
// activity closed, after the attempt that closed it and caused by it, and names no SDK attempt.
// An answer the server loses releases nothing: the same attempt redelivered is answered again, and
// the next attempt finds its reservation. An attempt the worker refuses is recorded as refused and
// the Run is incomplete.
func TestRunRecordsWhatEachDeclaredActivityAttemptDid(t *testing.T) {
	succeeded, canceled := testpilotspb.INSTRUCTION_OUTCOME_STATUS_SUCCEEDED, testpilotspb.INSTRUCTION_OUTCOME_STATUS_CANCELED
	completed := testpilotspb.ACTIVITY_ATTEMPT_RESPONSE_OFFERED_COMPLETED
	notNeeded := attemptFact(canceled, 0, "", testpilotspb.ACTIVITY_ATTEMPT_RESPONSE_NOT_NEEDED)
	carried, err := anypb.New(&failurepb.Failure{Message: "a value, not an outcome"})
	require.NoError(t, err)
	disabled := attemptFinish("first-attempt", textLiteral("never"))
	disabled.Guard = &testpilotspb.Expression{Expression: &testpilotspb.Expression_Literal{Literal: &testpilotspb.Value{Value: &testpilotspb.Value_BoolValue{BoolValue: false}}}}
	refusal := attemptFact(testpilotspb.INSTRUCTION_OUTCOME_STATUS_SDK_FAILURE, 1, deliveryOf("token-1"), testpilotspb.ACTIVITY_ATTEMPT_RESPONSE_REFUSED)
	refusal.SdkFailureCode, refusal.Detail = "umpire_worker", "the activity attempt's instruction is disabled"
	const started, first, second, third = "scheduler.g0.n0.a1.started", "scheduler.g0.n0.a1.r0.i0", "scheduler.g0.n0.a1.r0.i1", "scheduler.g0.n0.a1.r0.i2"
	fact := func(source string, outcome *testpilotspb.InstructionOutcome, causes ...string) recordedAttempt {
		return recordedAttempt{Source: source, Causes: append([]string{started}, causes...), Outcome: outcome.String()}
	}

	for name, test := range map[string]struct {
		script []*testpilotspb.InstructionNode
		// cancels makes the controller request the activity's cancellation once it started it.
		cancels     bool
		loses       string
		noRetry     bool
		told        []string
		facts       []recordedAttempt
		disposition testpilotspb.RunDisposition
		diagnostics []string
	}{
		"one attempt completes": {
			script:      []*testpilotspb.InstructionNode{attemptFinish("first-attempt", textLiteral("done"))},
			told:        []string{"completed"},
			facts:       []recordedAttempt{fact(first, attemptFact(succeeded, 1, deliveryOf("token-1"), completed))},
			disposition: testpilotspb.RUN_DISPOSITION_COMPLETED,
		},
		"an attempt fails retryably and its retry completes": {
			script: []*testpilotspb.InstructionNode{attemptFailure("first-attempt", "transient", false), attemptFinish("second-attempt", textLiteral("done"))},
			told:   []string{"failed:transient", "completed"},
			facts: []recordedAttempt{
				fact(first, attemptFact(succeeded, 1, deliveryOf("token-1"), testpilotspb.ACTIVITY_ATTEMPT_RESPONSE_OFFERED_FAILED_RETRYABLE)),
				fact(second, attemptFact(succeeded, 2, deliveryOf("token-2"), completed)),
			},
			disposition: testpilotspb.RUN_DISPOSITION_COMPLETED,
		},
		"a non-retryable failure closes the activity before its declared retry": {
			script: []*testpilotspb.InstructionNode{attemptFailure("first-attempt", "refusal", true), attemptFinish("second-attempt", textLiteral("done"))},
			told:   []string{"failed:refusal"},
			facts: []recordedAttempt{
				fact(first, attemptFact(succeeded, 1, deliveryOf("token-1"), testpilotspb.ACTIVITY_ATTEMPT_RESPONSE_OFFERED_FAILED_NON_RETRYABLE)),
				fact(second, notNeeded, first),
			},
			disposition: testpilotspb.RUN_DISPOSITION_COMPLETED,
		},
		"an early completion closes the activity before its declared retries": {
			script: []*testpilotspb.InstructionNode{attemptFinish("first-attempt", textLiteral("done")), attemptFailure("second-attempt", "transient", false), attemptFinish("third-attempt", textLiteral("late"))},
			told:   []string{"completed"},
			facts: []recordedAttempt{
				fact(first, attemptFact(succeeded, 1, deliveryOf("token-1"), completed)),
				fact(second, notNeeded, first), fact(third, notNeeded, first),
			},
			disposition: testpilotspb.RUN_DISPOSITION_COMPLETED,
		},
		// The server's retry policy, not the failure, ends the activity here.
		"a retryable failure the server does not retry closes the activity": {
			script:  []*testpilotspb.InstructionNode{attemptFailure("first-attempt", "transient", false), attemptFinish("second-attempt", textLiteral("done"))},
			noRetry: true,
			told:    []string{"failed:transient"},
			facts: []recordedAttempt{
				fact(first, attemptFact(succeeded, 1, deliveryOf("token-1"), testpilotspb.ACTIVITY_ATTEMPT_RESPONSE_OFFERED_FAILED_RETRYABLE)),
				fact(second, notNeeded, first),
			},
			disposition: testpilotspb.RUN_DISPOSITION_COMPLETED,
		},
		// The completion never reaches the server, which delivers the same attempt again: it is
		// answered again, recorded once, and only the accepted answer closes the activity.
		"a completion the server loses is offered again on redelivery": {
			script: []*testpilotspb.InstructionNode{attemptFinish("first-attempt", textLiteral("done")), attemptFinish("second-attempt", textLiteral("late"))},
			loses:  losesAndRedelivers,
			told:   []string{"lost", "completed"},
			facts: []recordedAttempt{
				fact(first, attemptFact(succeeded, 1, deliveryOf("token-1"), completed)),
				fact(second, notNeeded, first),
			},
			disposition: testpilotspb.RUN_DISPOSITION_COMPLETED,
		},
		// The server had already timed the first attempt out, so it rejects the completion and
		// issues the second attempt, which runs the second instruction under its own reservation.
		"a completion the server rejects is followed by the next attempt": {
			script: []*testpilotspb.InstructionNode{attemptFinish("first-attempt", textLiteral("one")), attemptFinish("second-attempt", textLiteral("two")), attemptFinish("third-attempt", textLiteral("three"))},
			loses:  losesAndIssuesTheNext,
			told:   []string{"lost", "completed"},
			facts: []recordedAttempt{
				fact(first, attemptFact(succeeded, 1, deliveryOf("token-1"), completed)),
				fact(second, attemptFact(succeeded, 2, deliveryOf("token-2"), completed)),
				fact(third, notNeeded, second),
			},
			disposition: testpilotspb.RUN_DISPOSITION_COMPLETED,
		},
		// A result that happens to be a Temporal failure message is a result.
		"an attempt completes with a failure message as its result": {
			script:      []*testpilotspb.InstructionNode{attemptFinish("first-attempt", &testpilotspb.Expression{Expression: &testpilotspb.Expression_Literal{Literal: &testpilotspb.Value{Value: &testpilotspb.Value_MessageValue{MessageValue: carried}}}})},
			told:        []string{"completed"},
			facts:       []recordedAttempt{fact(first, attemptFact(succeeded, 1, deliveryOf("token-1"), completed))},
			disposition: testpilotspb.RUN_DISPOSITION_COMPLETED,
		},
		// The attempt heartbeats, the server answers that the cancellation is requested, and the
		// worker answers canceled, which the server accepts and closes the activity on.
		"a requested cancellation is answered and closes the activity": {
			script:  []*testpilotspb.InstructionNode{attemptCancellation("first-attempt"), attemptFinish("second-attempt", textLiteral("late"))},
			cancels: true,
			told:    []string{"canceled"},
			facts: []recordedAttempt{
				fact(first, attemptFact(succeeded, 1, deliveryOf("token-1"), testpilotspb.ACTIVITY_ATTEMPT_RESPONSE_OFFERED_CANCELED)),
				fact(second, notNeeded, first),
			},
			disposition: testpilotspb.RUN_DISPOSITION_COMPLETED,
		},
		"the worker refuses an attempt": {
			script:      []*testpilotspb.InstructionNode{disabled},
			told:        []string{"failed:umpire_worker"},
			facts:       []recordedAttempt{fact(first, refusal)},
			disposition: testpilotspb.RUN_DISPOSITION_INCOMPLETE,
			diagnostics: []string{"activation_failed"},
		},
	} {
		t.Run(name, func(t *testing.T) {
			server := &activityServer{tasks: make(chan *workflowservice.PollActivityTaskQueueResponse, 8), closed: make(chan struct{}), loses: test.loses, noRetry: test.noRetry}
			source := activityScriptCase(test.script)
			if test.cancels {
				controller := source.Program.Entrypoints[0]
				controller.Instructions = append(controller.Instructions, &testpilotspb.InstructionNode{
					InstructionId: "request-cancel",
					Instruction: &testpilotspb.Instruction{Instruction: &testpilotspb.Instruction_InvokeRpc{InvokeRpc: &testpilotspb.InvokeRpc{
						EndpointRoleId: "workflow-service", Method: "/temporal.api.workflowservice.v1.WorkflowService/RequestCancelActivityExecution",
						RequestAssignments: []*testpilotspb.RequestAssignment{
							{Target: "namespace", Value: environmentReference("namespace")},
							{Target: "activity_id", Value: textLiteral("activity-id")},
						},
					}}},
				})
			}
			_, run, _ := runActivityScript(t, server, source)

			var facts []recordedAttempt
			for _, event := range run.GetEvents() {
				if event.GetOutcome().GetActivityAttempt() == nil {
					continue
				}
				require.Equal(t, testpilotspb.RUN_EVENT_KIND_DIAGNOSTIC, event.GetKind())
				require.Equal(t, "controller", event.GetCoordinates().GetEntrypointId())
				require.Equal(t, "start-activity", event.GetCoordinates().GetInstructionId())
				facts = append(facts, recordedAttempt{Source: event.GetSourceId(), Causes: event.GetCausalSourceIds(), Outcome: event.GetOutcome().String()})
			}
			require.Equal(t, test.facts, facts)
			// An attempt settles when the worker hands the SDK its answer, which the SDK then sends.
			await.Require(t.Context(), t, func(t *await.T) { require.Equal(t, test.told, server.told()) }, 10*time.Second, 10*time.Millisecond)
			require.Equal(t, test.disposition, run.GetDisposition(), run.GetDiagnostics())
			var diagnostics []string
			for _, diagnostic := range run.GetDiagnostics() {
				diagnostics = append(diagnostics, diagnostic.GetCode())
			}
			require.Equal(t, test.diagnostics, diagnostics)
		})
	}
}
