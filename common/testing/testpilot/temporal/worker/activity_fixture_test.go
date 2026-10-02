package worker

import (
	"cmp"
	"context"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"
	commonpb "go.temporal.io/api/common/v1"
	failurepb "go.temporal.io/api/failure/v1"
	taskqueuepb "go.temporal.io/api/taskqueue/v1"
	"go.temporal.io/api/workflowservice/v1"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/testpilot"
	"go.temporal.io/server/common/testing/testpilot/internal/testsupport/facadetest"
	"go.temporal.io/server/common/testing/testpilot/temporal/internal/delivery"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/dynamicpb"
)

// activityEntrypoint is the entrypoint an activity realization lowers to (model/scalav2/goir/testpilot):
// an activity activation and the one Finish its attempt performs.
func activityEntrypoint(result *testpilotspb.Expression) *testpilotspb.Entrypoint {
	return &testpilotspb.Entrypoint{
		EntrypointId: "activity",
		Activation:   &testpilotspb.Entrypoint_Activity{Activity: &testpilotspb.ActivityActivation{ActivityType: "activity-type", WorkerRoleId: "worker", TaskQueueRoleId: "queue"}},
		Instructions: []*testpilotspb.InstructionNode{{
			InstructionId: "run-attempt",
			Instruction:   &testpilotspb.Instruction{Instruction: &testpilotspb.Instruction_Finish{Finish: &testpilotspb.Finish{Result: result}}},
			Limits:        facadetest.Bounds(),
		}},
	}
}

// failingAttempt is the instruction that ends its attempt with an application failure of the type.
func failingAttempt(id, failureType string, nonRetryable bool) *testpilotspb.InstructionNode {
	return &testpilotspb.InstructionNode{
		InstructionId: id,
		Instruction: &testpilotspb.Instruction{Instruction: &testpilotspb.Instruction_ActivityAttemptFailure{ActivityAttemptFailure: &testpilotspb.ActivityAttemptFailure{
			Failure: &failurepb.Failure{Message: "not yet", FailureInfo: &failurepb.Failure_ApplicationFailureInfo{ApplicationFailureInfo: &failurepb.ApplicationFailureInfo{Type: failureType, NonRetryable: nonRetryable}}},
		}}},
		Limits: facadetest.Bounds(),
	}
}

// cancelingAttempt is the instruction that answers its attempt as canceled.
func cancelingAttempt(id string) *testpilotspb.InstructionNode {
	return &testpilotspb.InstructionNode{
		InstructionId: id,
		Instruction:   &testpilotspb.Instruction{Instruction: &testpilotspb.Instruction_ActivityAttemptCancellation{ActivityAttemptCancellation: &testpilotspb.ActivityAttemptCancellation{}}},
		Limits:        facadetest.Bounds(),
	}
}

// canceledActivity is standaloneActivity whose script declares two attempts: the first answers a
// requested cancellation, which ends the activity, so the second never comes.
func canceledActivity(program *testpilotspb.Program) {
	standaloneActivity(program)
	script := program.Entrypoints[1]
	script.Instructions = []*testpilotspb.InstructionNode{cancelingAttempt("first-attempt"), script.Instructions[0]}
}

// retriedActivity is standaloneActivity whose script declares two attempts: the first fails
// retryably and the second completes.
func retriedActivity(program *testpilotspb.Program) {
	standaloneActivity(program)
	script := program.Entrypoints[1]
	script.Instructions = []*testpilotspb.InstructionNode{failingAttempt("first-attempt", "transient", false), script.Instructions[0]}
}

// endedByFailureActivity is retriedActivity whose first attempt declares its failure non-retryable:
// that attempt ends the activity, so the second attempt the script declares never comes.
func endedByFailureActivity(program *testpilotspb.Program) {
	standaloneActivity(program)
	script := program.Entrypoints[1]
	script.Instructions = []*testpilotspb.InstructionNode{failingAttempt("first-attempt", "refusal", true), script.Instructions[0]}
}

func startActivityNode() *testpilotspb.InstructionNode {
	return &testpilotspb.InstructionNode{
		InstructionId: "start-activity",
		Instruction: &testpilotspb.Instruction{Instruction: &testpilotspb.Instruction_InvokeRpc{InvokeRpc: &testpilotspb.InvokeRpc{EndpointRoleId: "endpoint", Method: delivery.StartActivityPath, RequestAssignments: []*testpilotspb.RequestAssignment{
			{Target: "namespace", Value: symbolicEnvironment("namespace")},
			{Target: "task_queue.name", Value: symbolicEnvironment("task-queue")},
			{Target: "activity_type.name", Value: facadetest.Text("activity-type")},
			{Target: "activity_id", Value: facadetest.Text("activity-id")},
		}}}},
		Limits: facadetest.Bounds(),
	}
}

func authorizeActivities(profile *testpilot.ProfileSpec) {
	profile.Opcodes = append(profile.Opcodes, testpilot.ActivityAttemptFailure, testpilot.ActivityAttemptCancellation)
	profile.Roles[0].Methods = append(profile.Roles[0].Methods, delivery.StartActivityPath)
	profile.Roles[0].ReservationCarriers = append(profile.Roles[0].ReservationCarriers, testpilot.ReservationCarrierPolicy{Method: delivery.StartActivityPath, Shapes: []testpilot.ReservationCarrierShape{{Kind: testpilot.ActivityEntrypoint, MaximumCount: 8}}})
}

// standaloneActivity makes the runtime fixture a Program whose controller starts one standalone
// activity and nothing else.
func standaloneActivity(program *testpilotspb.Program) {
	program.Entrypoints[0].Instructions = []*testpilotspb.InstructionNode{startActivityNode()}
	program.Entrypoints = []*testpilotspb.Entrypoint{program.Entrypoints[0], activityEntrypoint(facadetest.Text("done"))}
}

// besideWorkflow keeps the runtime fixture's workflow and Nexus handler and adds the activity the
// controller starts after the workflow.
func besideWorkflow(program *testpilotspb.Program) {
	program.Entrypoints[0].Instructions = append(program.Entrypoints[0].Instructions, startActivityNode())
	program.Entrypoints = append(program.Entrypoints, activityEntrypoint(facadetest.Text("done")))
}

// preparedActivityCase prepares the runtime fixture as shape makes it, under a Profile that names
// StartActivityExecution a carrier of activity activations; modifiers then adjust the Program or
// the Profile.
func preparedActivityCase(t *testing.T, shape func(*testpilotspb.Program), modifiers ...any) *testpilot.PreparedCase {
	t.Helper()
	return preparedSymbolicRuntimeCase(t, append([]any{authorizeActivities, shape}, modifiers...)...)
}

func preparedActivityFixture(t *testing.T, shape func(*testpilotspb.Program), modifiers ...any) testpilot.PreparedProgram {
	t.Helper()
	return facadetest.Capture(t, preparedActivityCase(t, shape, modifiers...))
}

func activityBinding(activityID string) delivery.ActivityBinding {
	return delivery.ActivityBinding{Namespace: "namespace", ActivityID: activityID, ActivityType: "activity-type", TaskQueue: "task-queue"}
}

func activityOrigin(runID string) testpilot.Coordinate {
	return testpilot.Coordinate{RunID: runID, EntrypointID: "controller", ActivationID: "controller-1", InstructionID: "start-activity", Attempt: 1}
}

// activityTestSession opens a Session, reserves the activity's activations and carries them in a
// StartActivityExecution request the way the composite Driver does, then settles the start with
// disposition. The request it returns is the one the server would receive.
func activityTestSession(t *testing.T, host *Driver, definition programDefinition, prepared testpilot.PreparedProgram, runID string, binding delivery.ActivityBinding, activityRunID string, disposition delivery.TriggerStatus) (*Session, *Carrier, *workflowservice.StartActivityExecutionRequest) {
	t.Helper()
	session, err := newSession(host, runID, "session-"+runID, definition, SessionOptions{Bridge: newTestBridge()})
	require.NoError(t, err)
	require.NoError(t, host.mu.LockContext(t.Context(), ErrInvalid))
	host.sessions[runID] = session
	host.mu.Unlock()
	carrier, request := carryActivityStart(t, session, prepared, binding, activityRunID, disposition)
	return session, carrier, request
}

// carryActivityStart reserves one activation per attempt the activity's script declares and
// carries them in the start request, as the scheduler and the composite Driver do between them.
func carryActivityStart(t *testing.T, session *Session, prepared testpilot.PreparedProgram, binding delivery.ActivityBinding, activityRunID string, disposition delivery.TriggerStatus) (*Carrier, *workflowservice.StartActivityExecutionRequest) {
	t.Helper()
	origin := activityOrigin(session.runID)
	plan, exists := prepared.ReservationCarrier("controller", "start-activity")
	require.True(t, exists)
	handles, err := session.Reserve(t.Context(), testpilot.ReservationRequest{Origin: origin, EntrypointID: "activity", Count: plan.Reservations[0].Count})
	require.NoError(t, err)
	carrier, err := session.CreateActivityCarrier(t.Context(), origin, plan, binding, handles)
	require.NoError(t, err)
	request := &workflowservice.StartActivityExecutionRequest{Namespace: binding.Namespace, ActivityId: binding.ActivityID, ActivityType: &commonpb.ActivityType{Name: binding.ActivityType}, TaskQueue: &taskqueuepb.TaskQueue{Name: binding.TaskQueue}}
	var start testpilot.InstructionPlan
	for _, instruction := range prepared.Entrypoints()[0].Instructions() {
		if instruction.Source().GetInstructionId() == "start-activity" {
			start = instruction
		}
	}
	method := start.Method()
	wire, err := proto.Marshal(request)
	require.NoError(t, err)
	dynamicRequest := dynamicpb.NewMessage(method.Input())
	require.NoError(t, proto.Unmarshal(wire, dynamicRequest))
	preparedRequest, err := carrier.PrepareRPC(t.Context(), "endpoint", method, dynamicRequest, 64<<10)
	require.NoError(t, err)
	wire, err = proto.Marshal(preparedRequest)
	require.NoError(t, err)
	require.NoError(t, proto.Unmarshal(wire, request))
	if disposition == delivery.TriggerSucceeded {
		require.NoError(t, carrier.PinStartResponse(t.Context(), &workflowservice.StartActivityExecutionResponse{RunId: activityRunID}))
	}
	_, err = carrier.TriggerTerminal(t.Context(), disposition)
	require.NoError(t, err)
	return carrier, request
}

// activityDelivery is the task the worker receives for the activity request started, in the run
// the server gave it: its first attempt, under one delivery identity.
func activityDelivery(request *workflowservice.StartActivityExecutionRequest, activityRunID string) delivery.ActivityDelivery {
	return activityAttempt(request, activityRunID, 1, "delivery-1")
}

func activityAttempt(request *workflowservice.StartActivityExecutionRequest, activityRunID string, attempt int32, deliveryID string) delivery.ActivityDelivery {
	return delivery.ActivityDelivery{Header: request.GetHeader(), Namespace: request.GetNamespace(), ActivityID: request.GetActivityId(), ActivityType: request.GetActivityType().GetName(), TaskQueue: request.GetTaskQueue().GetName(), ActivityRunID: activityRunID, Attempt: attempt, DeliveryID: deliveryID}
}

// answered is the recorded outcome of an attempt that did what its script said: a succeeded
// activation carrying the attempt's identities and what the worker answered Temporal with.
func answered(activityRunID string, attempt int32, deliveryID string, response testpilotspb.ActivityAttemptResponse) *testpilotspb.InstructionOutcome {
	return &testpilotspb.InstructionOutcome{
		Status:          testpilotspb.INSTRUCTION_OUTCOME_STATUS_SUCCEEDED,
		ActivityAttempt: &testpilotspb.ActivityAttempt{ActivityRunId: activityRunID, SdkAttempt: attempt, DeliveryId: deliveryID, Response: response},
	}
}

// refused is the recorded outcome of an attempt the Driver itself failed: the non-retryable
// application failure the worker answered Temporal with, and why.
func refused(activityRunID string, attempt int32, deliveryID, cause string) *testpilotspb.InstructionOutcome {
	return &testpilotspb.InstructionOutcome{
		Status: testpilotspb.INSTRUCTION_OUTCOME_STATUS_SDK_FAILURE, SdkFailureCode: "umpire_worker", Detail: cause,
		ActivityAttempt: &testpilotspb.ActivityAttempt{ActivityRunId: activityRunID, SdkAttempt: attempt, DeliveryId: deliveryID, Response: testpilotspb.ACTIVITY_ATTEMPT_RESPONSE_REFUSED},
	}
}

// notNeeded is the recorded outcome of a reservation released because the server reported the
// activity closed before any attempt was delivered for it: the activity run, and neither an SDK
// attempt nor a delivery.
func notNeeded(activityRunID string) *testpilotspb.InstructionOutcome {
	return &testpilotspb.InstructionOutcome{
		Status:          testpilotspb.INSTRUCTION_OUTCOME_STATUS_CANCELED,
		ActivityAttempt: &testpilotspb.ActivityAttempt{ActivityRunId: activityRunID, Response: testpilotspb.ACTIVITY_ATTEMPT_RESPONSE_NOT_NEEDED},
	}
}

// serverClosure makes the host's evidence that an activity closed a channel the test sends on, as
// the server answering the long poll would, and counts how often the worker asks. What is sent is
// the run the server reports closed: empty, the run it was asked about.
func serverClosure(host *Driver) (chan string, *atomic.Int32) {
	closed, asked := make(chan string, 1), &atomic.Int32{}
	host.options.activityClosed = func(ctx context.Context, _, _, activityRunID string) (string, error) {
		asked.Add(1)
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case reported := <-closed:
			// The server names the run it was asked about unless the test crosses it.
			return cmp.Or(reported, activityRunID), nil
		}
	}
	return closed, asked
}

// countHeartbeats replaces the heartbeat an attempt sends through the SDK with a count of them: a
// delivery these tests hand the worker directly has no SDK activity behind it.
func countHeartbeats(host *Driver) *atomic.Int32 {
	sent := &atomic.Int32{}
	host.options.heartbeat = func(context.Context) { sent.Add(1) }
	return sent
}

// diagnosed is one diagnostic a Session reported to its Run, whole but for the id the Session
// numbers it with.
type diagnosed struct {
	Kind   testpilotspb.RunDiagnosticKind
	Code   string
	Detail string
}

// diagnosticLog is what a Session reported to its Run, in order.
type diagnosticLog struct {
	mu       sync.Mutex
	reported []diagnosed
}

// captureDiagnostics makes the log the Session's diagnostic sink, as the Run's recorder is.
func captureDiagnostics(t *testing.T, session *Session) *diagnosticLog {
	t.Helper()
	log := &diagnosticLog{}
	require.NoError(t, session.mu.LockContext(t.Context(), ErrInvalid))
	defer session.mu.Unlock()
	session.options.Diagnose = func(_ context.Context, _ string, diagnostic *testpilotspb.RunDiagnostic) error {
		log.mu.Lock()
		defer log.mu.Unlock()
		log.reported = append(log.reported, diagnosed{Kind: diagnostic.GetKind(), Code: diagnostic.GetCode(), Detail: diagnostic.GetDetail()})
		return nil
	}
	return log
}

func (l *diagnosticLog) all() []diagnosed {
	l.mu.Lock()
	defer l.mu.Unlock()
	return append([]diagnosed(nil), l.reported...)
}

func (l *diagnosticLog) codes() []string {
	var codes []string
	for _, diagnostic := range l.all() {
		codes = append(codes, diagnostic.Code)
	}
	return codes
}

const (
	completed          = testpilotspb.ACTIVITY_ATTEMPT_RESPONSE_OFFERED_COMPLETED
	failedRetryable    = testpilotspb.ACTIVITY_ATTEMPT_RESPONSE_OFFERED_FAILED_RETRYABLE
	failedNonRetryable = testpilotspb.ACTIVITY_ATTEMPT_RESPONSE_OFFERED_FAILED_NON_RETRYABLE
	canceledAnswer     = testpilotspb.ACTIVITY_ATTEMPT_RESPONSE_OFFERED_CANCELED
)

// settledActivity is the settled outcome of the first attempt's reservation.
func settledActivity(t *testing.T, session *Session) (testpilot.EffectResult, error) {
	t.Helper()
	return settledAttempt(t, session, "reservation-1")
}

// settledAttempt is the settled outcome of one attempt's reservation. An attempt that never
// settles its reservation is a failure, not a test that waits out its timeout.
func settledAttempt(t *testing.T, session *Session, reservationID string) (testpilot.EffectResult, error) {
	t.Helper()
	raw := session.reservations[reservationID]
	require.NotNil(t, raw, reservationID)
	select {
	case <-raw.done:
	default:
		require.FailNow(t, "the activity reservation was not settled", reservationID)
	}
	return raw.Wait(t.Context())
}

func activityWorker(host *Driver, definition programDefinition) *sdkWorkerInterceptor {
	return &sdkWorkerInterceptor{host: host, queue: "task-queue", registration: definition.registrations[0]}
}

// runScript is the activity function the SDK calls once the inbound interceptor admitted the task.
func runScript(host *Driver) func(context.Context) (any, error) {
	return func(ctx context.Context) (any, error) { return host.dynamicActivity(ctx, nil) }
}
