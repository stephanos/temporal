package temporal

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	commonpb "go.temporal.io/api/common/v1"
	taskqueuepb "go.temporal.io/api/taskqueue/v1"
	"go.temporal.io/api/workflowservice/v1"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/testpilot"
	"go.temporal.io/server/common/testing/testpilot/internal/testsupport"
	"go.temporal.io/server/common/testing/testpilot/internal/testsupport/facadetest"
	"go.temporal.io/server/common/testing/testpilot/temporal/internal/delivery"
	workerhost "go.temporal.io/server/common/testing/testpilot/temporal/worker"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/dynamicpb"
)

func TestCompositeSessionKeepsControllerAndWorkerAuthoritySeparate(t *testing.T) {
	var closes []string
	closing := func(name string) func(context.Context) error {
		return func(context.Context) error {
			closes = append(closes, name)
			return nil
		}
	}
	controller := &testsupport.Session{HandleBridge: &recordingBridge{}, OnInvokeRPC: completedRPC, OnQuarantine: acceptQuarantine, OnClose: closing("controller")}
	workers := &testsupport.Session{OnReserve: func(context.Context, testpilot.ReservationRequest) ([]testpilot.ReservationHandle, error) {
		return []testpilot.ReservationHandle{testsupport.NewReservation(testpilot.ReservationIdentity{})}, nil
	}, OnQuarantine: acceptQuarantine, OnClose: closing("worker")}
	session := newPreparedCompositeSession(controller, workers, preparedConformanceProgram(t))
	origin := testpilot.Coordinate{RunID: "run", EntrypointID: "controller", ActivationID: "activation", InstructionID: "call", Attempt: 1}

	reservations, err := session.Reserve(t.Context(), testpilot.ReservationRequest{Origin: origin, EntrypointID: "workflow", Count: 1})
	require.NoError(t, err)
	require.Len(t, reservations, 1)
	_, err = session.InvokeRPC(t.Context(), origin, "endpoint", nil, nil)
	require.NoError(t, err)
	bridge, err := session.Bridge(t.Context())
	require.NoError(t, err)
	require.Same(t, controller.HandleBridge, bridge)
	require.NoError(t, session.Quarantine(t.Context(), reservations[0]))
	require.NoError(t, session.Close(t.Context()))
	require.Equal(t, 1, workers.Calls("Reserve"))
	require.Equal(t, 1, workers.Calls("Quarantine"))
	require.Zero(t, controller.Calls("Reserve"))
	require.Zero(t, controller.Calls("Quarantine"))
	require.Equal(t, 1, controller.Calls("InvokeRPC"))
	require.Equal(t, []string{"worker", "controller"}, closes)
}

// A fault is worker authority, so the composite hands it to the worker Session; a Program with no
// worker use has no worker Session and therefore no queue to stop.
func TestCompositeSessionRoutesFaultsToTheWorker(t *testing.T) {
	workers := &testsupport.Session{OnInjectFault: func(context.Context, testpilot.Coordinate, string, testpilotspb.FaultKind) (testpilot.EffectHandle, error) {
		return testsupport.Completed(testpilot.EffectResult{}), nil
	}}
	program := preparedConformanceProgram(t)
	session := newPreparedCompositeSession(&testsupport.Session{HandleBridge: &recordingBridge{}}, workers, program)
	origin := testpilot.Coordinate{RunID: "run", EntrypointID: "controller", ActivationID: "activation", InstructionID: "stop", Attempt: 1}
	_, err := session.InjectFault(t.Context(), origin, "queue", testpilotspb.FAULT_KIND_WORKER_STOP)
	require.NoError(t, err)
	require.Equal(t, 1, workers.Calls("InjectFault"))

	workerless := newPreparedCompositeSession(&testsupport.Session{HandleBridge: &recordingBridge{}}, nil, program)
	_, err = workerless.InjectFault(t.Context(), origin, "queue", testpilotspb.FAULT_KIND_WORKER_STOP)
	require.ErrorIs(t, err, ErrInvalid)
}

func completedRPC(context.Context, testpilot.Coordinate, string, protoreflect.MethodDescriptor, proto.Message) (testpilot.EffectHandle, error) {
	return testsupport.Completed(testpilot.EffectResult{}), nil
}

func acceptQuarantine(context.Context, testpilot.EffectHandle) error { return nil }

// preparedConformanceProgram is the satisfied conformance Case's Program as preparation hands it to
// a Driver. It declares no reservation carrier at the coordinates these tests invoke from.
func preparedConformanceProgram(t *testing.T) testpilot.PreparedProgram {
	t.Helper()
	encoded, err := os.ReadFile(filepath.Join("..", "testdata", "case-runtime-conformance", "satisfied", "case.json"))
	require.NoError(t, err)
	source, err := testpilot.DecodeCaseProtoJSON(encoded)
	require.NoError(t, err)
	catalog, err := NewWorkflowServiceCatalog()
	require.NoError(t, err)
	profile, err := DeriveProfile(source, catalog, Environment{Identity: "composite"})
	require.NoError(t, err)
	// The corpus is written for the facade conformance Profile's instruction defaults.
	profile.InstructionDefaults = testpilot.InstructionDefaults{TimeoutMilliseconds: 10000, MaxAttempts: 1}
	prepared, err := testpilot.Prepare(source, profile)
	require.NoError(t, err)
	return facadetest.Capture(t, prepared)
}

func TestWorkerQuarantineCompletionFollowsRawHandle(t *testing.T) {
	raw := &testsupport.Effect{WaitErr: errors.New("terminal worker failure")}
	completed := make(chan struct{})
	require.NoError(t, quarantineWorkerHandle(t.Context(), raw, func() { close(completed) }))
	select {
	case <-completed:
		t.Fatal("worker quarantine released before the raw handle completed")
	default:
	}
	raw.Complete()
	ctx, cancel := context.WithTimeout(t.Context(), time.Second)
	defer cancel()
	select {
	case <-completed:
	case <-ctx.Done():
		require.NoError(t, ctx.Err())
	}
	canceled, cancelCanceled := context.WithCancel(t.Context())
	cancelCanceled()
	require.ErrorIs(t, quarantineWorkerHandle(canceled, raw, func() {}), context.Canceled)
	require.ErrorIs(t, quarantineWorkerHandle(t.Context(), nil, func() {}), ErrInvalid)
}

// A carried StartWorkflow request names its whole binding; a request that omits a binding field or
// leaves one empty is the composite Driver's invalid input.
func TestCarrierBindingRejectsIncompleteStartRequests(t *testing.T) {
	complete := func() *workflowservice.StartWorkflowExecutionRequest {
		return &workflowservice.StartWorkflowExecutionRequest{Namespace: "namespace", WorkflowId: "workflow-id", WorkflowType: &commonpb.WorkflowType{Name: "workflow-type"}, TaskQueue: &taskqueuepb.TaskQueue{Name: "task-queue"}}
	}
	binding, err := carrierBinding(complete())
	require.NoError(t, err)
	require.Equal(t, delivery.WorkflowBinding{Namespace: "namespace", WorkflowID: "workflow-id", WorkflowType: "workflow-type", TaskQueue: "task-queue"}, binding)

	for name, mutate := range map[string]func(*workflowservice.StartWorkflowExecutionRequest){
		"namespace":             func(request *workflowservice.StartWorkflowExecutionRequest) { request.Namespace = "" },
		"workflow id":           func(request *workflowservice.StartWorkflowExecutionRequest) { request.WorkflowId = "" },
		"workflow type name":    func(request *workflowservice.StartWorkflowExecutionRequest) { request.WorkflowType.Name = "" },
		"task queue name":       func(request *workflowservice.StartWorkflowExecutionRequest) { request.TaskQueue.Name = "" },
		"missing workflow type": func(request *workflowservice.StartWorkflowExecutionRequest) { request.WorkflowType = nil },
	} {
		t.Run(name, func(t *testing.T) {
			request := complete()
			mutate(request)
			_, err := carrierBinding(request)
			require.ErrorIs(t, err, ErrInvalid)
		})
	}
}

func TestCarrierEffectFinalizesWithFreshBoundAndRetries(t *testing.T) {
	terminal := &recordingTerminalCarrier{failures: 1, admissible: true}
	effect := &carrierEffect{
		EffectHandle:   &testsupport.Effect{},
		carrier:        terminal,
		cleanupTimeout: time.Second,
	}
	canceled, cancel := context.WithCancel(t.Context())
	cancel()
	_, err := effect.Wait(canceled)
	require.ErrorIs(t, err, context.Canceled)
	require.ErrorContains(t, err, "injected terminal failure")
	require.Equal(t, []delivery.TriggerStatus{delivery.TriggerUncertain}, terminal.dispositions)
	require.Equal(t, []error{nil}, terminal.contextErrors)
	require.True(t, terminal.admissible)

	require.NoError(t, effect.Cancel(t.Context()))
	require.Equal(t, []delivery.TriggerStatus{delivery.TriggerUncertain, delivery.TriggerUncertain}, terminal.dispositions)
	require.Equal(t, []error{nil, nil}, terminal.contextErrors)
	require.False(t, terminal.admissible)
}

func TestCarrierEffectDrainFinalizesCompletedResult(t *testing.T) {
	terminal := &recordingTerminalCarrier{admissible: true}
	effect := &carrierEffect{
		EffectHandle: testsupport.Completed(testpilot.EffectResult{
			Outcome: &testpilotspb.InstructionOutcome{Status: testpilotspb.INSTRUCTION_OUTCOME_STATUS_PROTOCOL_FAILURE},
		}),
		carrier: terminal, cleanupTimeout: time.Second,
	}
	require.NoError(t, effect.Drain(t.Context()))
	require.Equal(t, []delivery.TriggerStatus{delivery.TriggerNonSuccess}, terminal.dispositions)
	require.False(t, terminal.admissible)
}

type recordingTerminalCarrier struct {
	failures      int
	admissible    bool
	dispositions  []delivery.TriggerStatus
	contextErrors []error
	pinnedRunID   string
}

func (c *recordingTerminalCarrier) PinStartResponse(ctx context.Context, response delivery.StartResponse) error {
	c.contextErrors = append(c.contextErrors, ctx.Err())
	c.pinnedRunID = response.GetRunId()
	return nil
}
func (c *recordingTerminalCarrier) TriggerTerminal(ctx context.Context, disposition delivery.TriggerStatus) (int, error) {
	c.contextErrors = append(c.contextErrors, ctx.Err())
	c.dispositions = append(c.dispositions, disposition)
	if c.failures > 0 {
		c.failures--
		return 0, errors.New("injected terminal failure")
	}
	c.admissible = false
	return 0, nil
}

type recordingBridge struct{}

func (*recordingBridge) Publish(context.Context, testpilot.Coordinate, string, testpilot.OpaqueHandle) error {
	return nil
}
func (*recordingBridge) Await(context.Context, string) error { return nil }
func (*recordingBridge) Consume(context.Context, string) (testpilot.OpaqueHandle, error) {
	return struct{}{}, nil
}

// A carried StartActivityExecution request names its whole binding, as a carried StartWorkflow
// request does.
func TestActivityCarrierBindingRejectsIncompleteStartRequests(t *testing.T) {
	complete := func() *workflowservice.StartActivityExecutionRequest {
		return &workflowservice.StartActivityExecutionRequest{Namespace: "namespace", ActivityId: "activity-id", ActivityType: &commonpb.ActivityType{Name: "activity-type"}, TaskQueue: &taskqueuepb.TaskQueue{Name: "task-queue"}}
	}
	binding, err := activityCarrierBinding(complete())
	require.NoError(t, err)
	require.Equal(t, delivery.ActivityBinding{Namespace: "namespace", ActivityID: "activity-id", ActivityType: "activity-type", TaskQueue: "task-queue"}, binding)

	for name, mutate := range map[string]func(*workflowservice.StartActivityExecutionRequest){
		"namespace":             func(request *workflowservice.StartActivityExecutionRequest) { request.Namespace = "" },
		"activity id":           func(request *workflowservice.StartActivityExecutionRequest) { request.ActivityId = "" },
		"activity type name":    func(request *workflowservice.StartActivityExecutionRequest) { request.ActivityType.Name = "" },
		"task queue name":       func(request *workflowservice.StartActivityExecutionRequest) { request.TaskQueue.Name = "" },
		"missing activity type": func(request *workflowservice.StartActivityExecutionRequest) { request.ActivityType = nil },
	} {
		t.Run(name, func(t *testing.T) {
			request := complete()
			mutate(request)
			_, err := activityCarrierBinding(request)
			require.ErrorIs(t, err, ErrInvalid)
		})
	}
	_, err = activityCarrierBinding(&workflowservice.StartWorkflowExecutionRequest{})
	require.ErrorIs(t, err, ErrInvalid)
}

// The run a carried start started is read from the response of whichever start it was, and from no
// other message.
func TestStartResponseReadsTheRunOfEitherStart(t *testing.T) {
	for name, test := range map[string]struct {
		response proto.Message
		runID    string
	}{
		"a workflow start":            {&workflowservice.StartWorkflowExecutionResponse{RunId: "workflow-run", Started: true}, "workflow-run"},
		"an activity start":           {&workflowservice.StartActivityExecutionResponse{RunId: "activity-run", Started: true}, "activity-run"},
		"a workflow start, no run":    {&workflowservice.StartWorkflowExecutionResponse{Started: true}, ""},
		"an activity start, no run":   {&workflowservice.StartActivityExecutionResponse{Started: true}, ""},
		"another response with a run": {&workflowservice.SignalWithStartWorkflowExecutionResponse{RunId: "signal-run"}, ""},
	} {
		t.Run(name, func(t *testing.T) {
			wire, err := proto.Marshal(test.response)
			require.NoError(t, err)
			dynamicResponse := dynamicpb.NewMessage(test.response.ProtoReflect().Descriptor())
			require.NoError(t, proto.Unmarshal(wire, dynamicResponse))
			started, err := startResponse(dynamicResponse)
			if test.runID == "" {
				require.ErrorIs(t, err, ErrInvalid)
				return
			}
			require.NoError(t, err)
			require.Equal(t, test.runID, started.GetRunId())
		})
	}
	_, err := startResponse(nil)
	require.ErrorIs(t, err, ErrInvalid)
}

// carrierRecorder is a worker Session that records which carrier the composite asks for and
// refuses to make it, so the test sees the request the composite built and no dispatch follows.
type carrierRecorder struct {
	testsupport.Session
	workflows  []delivery.WorkflowBinding
	activities []delivery.ActivityBinding
	handles    [][]testpilot.ReservationHandle
}

var errCarrierRecorded = errors.New("carrier recorded")

func (r *carrierRecorder) CreateCarrier(_ context.Context, _ testpilot.Coordinate, _ testpilot.ReservationCarrierPlan, binding delivery.WorkflowBinding, handles []testpilot.ReservationHandle) (*workerhost.Carrier, error) {
	r.workflows, r.handles = append(r.workflows, binding), append(r.handles, handles)
	return nil, errCarrierRecorded
}

func (r *carrierRecorder) CreateActivityCarrier(_ context.Context, _ testpilot.Coordinate, _ testpilot.ReservationCarrierPlan, binding delivery.ActivityBinding, handles []testpilot.ReservationHandle) (*workerhost.Carrier, error) {
	r.activities, r.handles = append(r.activities, binding), append(r.handles, handles)
	return nil, errCarrierRecorded
}

// The composite hands a carried StartActivityExecution to the worker as an activity carrier, with
// the reservations of that instruction and the binding the request names, and dispatches nothing
// when the worker refuses or the request names no whole binding.
func TestCompositeSessionCarriesAnActivityStartAsAnActivityCarrier(t *testing.T) {
	prepared := facadetest.RuntimeCase(t, facadetest.SyncReply, workerhost.CommandTypes(), func(profile *testpilot.ProfileSpec) {
		profile.Roles[0].Methods = append(profile.Roles[0].Methods, delivery.StartActivityPath)
		profile.Roles[0].ReservationCarriers = append(profile.Roles[0].ReservationCarriers, testpilot.ReservationCarrierPolicy{Method: delivery.StartActivityPath, Shapes: []testpilot.ReservationCarrierShape{{Kind: testpilot.ActivityEntrypoint, MaximumCount: 1}}})
	}, func(program *testpilotspb.Program) {
		program.Entrypoints[0].Instructions[0].GetInstruction().GetInvokeRpc().Method = delivery.StartActivityPath
		program.Entrypoints = append(program.Entrypoints[:1], &testpilotspb.Entrypoint{
			EntrypointId: "activity",
			Activation:   &testpilotspb.Entrypoint_Activity{Activity: &testpilotspb.ActivityActivation{ActivityType: "activity-type", WorkerRoleId: "worker", TaskQueueRoleId: "queue", AttemptNumbering: &testpilotspb.AttemptNumbering{First: 1, OneRun: true}}},
			Instructions: []*testpilotspb.InstructionNode{{InstructionId: "run-attempt", Limits: facadetest.Bounds(), Instruction: &testpilotspb.Instruction{Instruction: &testpilotspb.Instruction_Finish{Finish: &testpilotspb.Finish{Result: facadetest.Text("done")}}}}},
		})
	})
	program := facadetest.Capture(t, prepared)
	method := program.Entrypoints()[0].Instructions()[0].Method()
	origin := testpilot.Coordinate{RunID: "run", EntrypointID: "controller", ActivationID: "activation", InstructionID: "call", Attempt: 1}
	reserved := testsupport.NewReservation(testpilot.ReservationIdentity{Origin: origin, EntrypointID: "activity", ID: "reservation"})
	dynamicRequest := func(request *workflowservice.StartActivityExecutionRequest) proto.Message {
		wire, err := proto.Marshal(request)
		require.NoError(t, err)
		message := dynamicpb.NewMessage(method.Input())
		require.NoError(t, proto.Unmarshal(wire, message))
		return message
	}

	controller := &testsupport.Session{HandleBridge: &recordingBridge{}, OnInvokeRPC: completedRPC}
	workers := &carrierRecorder{}
	workers.OnReserve = func(context.Context, testpilot.ReservationRequest) ([]testpilot.ReservationHandle, error) {
		return []testpilot.ReservationHandle{reserved}, nil
	}
	session := newPreparedCompositeSession(controller, workers, program)
	_, err := session.Reserve(t.Context(), testpilot.ReservationRequest{Origin: origin, EntrypointID: "activity", Count: 1})
	require.NoError(t, err)

	_, err = session.InvokeRPC(t.Context(), origin, "endpoint", method, dynamicRequest(&workflowservice.StartActivityExecutionRequest{Namespace: "namespace", ActivityType: &commonpb.ActivityType{Name: "activity-type"}, TaskQueue: &taskqueuepb.TaskQueue{Name: "task-queue"}}))
	require.ErrorIs(t, err, ErrInvalid)
	require.Empty(t, workers.activities)

	_, err = session.InvokeRPC(t.Context(), origin, "endpoint", method, dynamicRequest(&workflowservice.StartActivityExecutionRequest{Namespace: "namespace", ActivityId: "activity-id", ActivityType: &commonpb.ActivityType{Name: "activity-type"}, TaskQueue: &taskqueuepb.TaskQueue{Name: "task-queue"}}))
	require.ErrorIs(t, err, errCarrierRecorded)
	require.Equal(t, []delivery.ActivityBinding{{Namespace: "namespace", ActivityID: "activity-id", ActivityType: "activity-type", TaskQueue: "task-queue"}}, workers.activities)
	require.Equal(t, [][]testpilot.ReservationHandle{{reserved}}, workers.handles)
	require.Empty(t, workers.workflows)
	require.Zero(t, controller.Calls("InvokeRPC"))
}
