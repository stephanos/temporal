package temporal

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.temporal.io/api/workflowservice/v1"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/testpilot"
	"go.temporal.io/server/common/testing/testpilot/temporal/internal/delivery"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

func TestCompositeSessionKeepsControllerAndWorkerAuthoritySeparate(t *testing.T) {
	controller := &recordingControllerSession{bridge: &recordingBridge{}}
	workers := &recordingWorkerSession{}
	session := newPreparedCompositeSession(controller, workers, preparedConformanceProgram(t))
	origin := testpilot.Coordinate{RunID: "run", EntrypointID: "controller", ActivationID: "activation", InstructionID: "call", Attempt: 1}

	reservations, err := session.Reserve(t.Context(), testpilot.ReservationRequest{Origin: origin, EntrypointID: "workflow", Count: 1})
	require.NoError(t, err)
	require.Len(t, reservations, 1)
	_, err = session.InvokeRPC(t.Context(), origin, "endpoint", nil, nil)
	require.NoError(t, err)
	bridge, err := session.Bridge(t.Context())
	require.NoError(t, err)
	require.Same(t, controller.bridge, bridge)
	require.NoError(t, session.Quarantine(t.Context(), reservations[0]))
	require.NoError(t, session.Close(t.Context()))
	require.Equal(t, 1, workers.reserves)
	require.Equal(t, 1, workers.quarantines)
	require.Zero(t, controller.quarantines)
	require.Equal(t, 1, controller.invocations)
	require.Equal(t, []string{"worker", "controller"}, append(workers.closes, controller.closes...))
}

// A fault is worker authority, so the composite hands it to the worker Session; a Program with no
// worker use has no worker Session and therefore no queue to stop.
func TestCompositeSessionRoutesFaultsToTheWorker(t *testing.T) {
	workers := &recordingWorkerSession{}
	program := preparedConformanceProgram(t)
	session := newPreparedCompositeSession(&recordingControllerSession{bridge: &recordingBridge{}}, workers, program)
	origin := testpilot.Coordinate{RunID: "run", EntrypointID: "controller", ActivationID: "activation", InstructionID: "stop", Attempt: 1}
	_, err := session.InjectFault(t.Context(), origin, "queue", testpilotspb.FAULT_KIND_WORKER_STOP)
	require.NoError(t, err)
	require.Equal(t, 1, workers.faults)

	workerless := newPreparedCompositeSession(&recordingControllerSession{bridge: &recordingBridge{}}, nil, program)
	_, err = workerless.InjectFault(t.Context(), origin, "queue", testpilotspb.FAULT_KIND_WORKER_STOP)
	require.ErrorIs(t, err, ErrInvalid)
}

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
	prepared, err := testpilot.Prepare(source, profile)
	require.NoError(t, err)
	capture := &programCaptureDriver{identity: prepared.Identity()}
	_, _, err = prepared.Run(t.Context(), capture)
	require.ErrorIs(t, err, errProgramCaptured)
	return capture.program
}

var errProgramCaptured = errors.New("prepared Program captured")

type programCaptureDriver struct {
	identity testpilot.DriverIdentity
	program  testpilot.PreparedProgram
}

func (d *programCaptureDriver) Identity(context.Context) (testpilot.DriverIdentity, error) {
	return d.identity, nil
}

func (*programCaptureDriver) Validate(context.Context, testpilot.PreparedProgram) error { return nil }

func (d *programCaptureDriver) Open(_ context.Context, _ string, program testpilot.PreparedProgram) (testpilot.Session, error) {
	d.program = program
	return nil, errProgramCaptured
}

type recordingControllerSession struct {
	bridge      testpilot.HandleBridge
	invocations int
	quarantines int
	closes      []string
}

func (*recordingControllerSession) Reserve(context.Context, testpilot.ReservationRequest) ([]testpilot.ReservationHandle, error) {
	panic("controller cannot reserve worker authority")
}
func (s *recordingControllerSession) InvokeRPC(context.Context, testpilot.Coordinate, string, protoreflect.MethodDescriptor, proto.Message) (testpilot.EffectHandle, error) {
	s.invocations++
	return recordingEffect{}, nil
}
func (s *recordingControllerSession) PollRPC(context.Context, testpilot.Coordinate, string, protoreflect.MethodDescriptor, proto.Message, time.Duration, testpilot.PollPredicate) (testpilot.EffectHandle, error) {
	s.invocations++
	return recordingEffect{}, nil
}
func (*recordingControllerSession) InvokeHandle(context.Context, testpilot.Coordinate, testpilot.OpaqueHandle, proto.Message) (testpilot.EffectHandle, error) {
	return recordingEffect{}, nil
}
func (*recordingControllerSession) InjectFault(context.Context, testpilot.Coordinate, string, testpilotspb.FaultKind) (testpilot.EffectHandle, error) {
	return nil, errors.New("controller Sessions do not realize faults")
}
func (s *recordingControllerSession) Bridge(context.Context) (testpilot.HandleBridge, error) {
	return s.bridge, nil
}
func (s *recordingControllerSession) Quarantine(context.Context, testpilot.EffectHandle) error {
	s.quarantines++
	return nil
}
func (s *recordingControllerSession) Close(context.Context) error {
	s.closes = append(s.closes, "controller")
	return nil
}
func (*recordingControllerSession) Diagnose(context.Context, string, *testpilotspb.RunDiagnostic) error {
	return nil
}

type recordingWorkerSession struct {
	reserves    int
	faults      int
	quarantines int
	closes      []string
}

func (*recordingWorkerSession) PollRPC(context.Context, testpilot.Coordinate, string, protoreflect.MethodDescriptor, proto.Message, time.Duration, testpilot.PollPredicate) (testpilot.EffectHandle, error) {
	return nil, errors.New("worker Sessions do not poll")
}
func (s *recordingWorkerSession) Reserve(context.Context, testpilot.ReservationRequest) ([]testpilot.ReservationHandle, error) {
	s.reserves++
	return []testpilot.ReservationHandle{recordingReservation{}}, nil
}
func (s *recordingWorkerSession) InjectFault(context.Context, testpilot.Coordinate, string, testpilotspb.FaultKind) (testpilot.EffectHandle, error) {
	s.faults++
	return recordingEffect{}, nil
}
func (s *recordingWorkerSession) Close(context.Context) error {
	s.closes = append(s.closes, "worker")
	return nil
}
func (s *recordingWorkerSession) Quarantine(context.Context, testpilot.EffectHandle) error {
	s.quarantines++
	return nil
}
func (*recordingWorkerSession) Diagnose(context.Context, string, *testpilotspb.RunDiagnostic) error {
	return nil
}

func TestWorkerQuarantineCompletionFollowsRawHandle(t *testing.T) {
	raw := &blockingEffect{done: make(chan struct{})}
	completed := make(chan struct{})
	require.NoError(t, quarantineWorkerHandle(t.Context(), raw, func() { close(completed) }))
	select {
	case <-completed:
		t.Fatal("worker quarantine released before the raw handle completed")
	default:
	}
	close(raw.done)
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

func TestCarrierEffectFinalizesWithFreshBoundAndRetries(t *testing.T) {
	terminal := &recordingTerminalCarrier{failures: 1, admissible: true}
	effect := &carrierEffect{
		EffectHandle:   &blockingEffect{done: make(chan struct{})},
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
	done := make(chan struct{})
	close(done)
	terminal := &recordingTerminalCarrier{admissible: true}
	effect := &carrierEffect{
		EffectHandle: &terminalEffect{done: done, result: testpilot.EffectResult{
			Outcome: &testpilotspb.InstructionOutcome{Status: testpilotspb.INSTRUCTION_OUTCOME_STATUS_PROTOCOL_FAILURE},
		}},
		carrier: terminal, cleanupTimeout: time.Second,
	}
	require.NoError(t, effect.Drain(t.Context()))
	require.Equal(t, []delivery.TriggerStatus{delivery.TriggerNonSuccess}, terminal.dispositions)
	require.False(t, terminal.admissible)
}

type recordingTerminalCarrier struct {
	failures       int
	admissible     bool
	dispositions   []delivery.TriggerStatus
	contextErrors  []error
	pinnedResponse *workflowservice.StartWorkflowExecutionResponse
}

func (c *recordingTerminalCarrier) PinStartResponse(ctx context.Context, response *workflowservice.StartWorkflowExecutionResponse) error {
	c.contextErrors = append(c.contextErrors, ctx.Err())
	c.pinnedResponse = proto.CloneOf(response)
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

type terminalEffect struct {
	done   chan struct{}
	result testpilot.EffectResult
}

func (e *terminalEffect) Wait(ctx context.Context) (testpilot.EffectResult, error) {
	select {
	case <-e.done:
		return e.result, nil
	case <-ctx.Done():
		return testpilot.EffectResult{}, ctx.Err()
	}
}
func (*terminalEffect) Cancel(context.Context) error { return nil }
func (e *terminalEffect) Drain(ctx context.Context) error {
	select {
	case <-e.done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

type blockingEffect struct {
	done chan struct{}
}

func (e *blockingEffect) Wait(ctx context.Context) (testpilot.EffectResult, error) {
	select {
	case <-e.done:
		return testpilot.EffectResult{}, errors.New("terminal worker failure")
	case <-ctx.Done():
		return testpilot.EffectResult{}, ctx.Err()
	}
}
func (*blockingEffect) Cancel(context.Context) error { return nil }
func (e *blockingEffect) Drain(ctx context.Context) error {
	_, err := e.Wait(ctx)
	return err
}

type recordingEffect struct{}

func (recordingEffect) Wait(context.Context) (testpilot.EffectResult, error) {
	return testpilot.EffectResult{}, nil
}
func (recordingEffect) Cancel(context.Context) error { return nil }
func (recordingEffect) Drain(context.Context) error  { return nil }

type recordingReservation struct{ recordingEffect }

func (recordingReservation) Identity() testpilot.ReservationIdentity {
	return testpilot.ReservationIdentity{}
}
func (recordingReservation) Consume(context.Context) (testpilot.Coordinate, error) {
	return testpilot.Coordinate{}, nil
}

type recordingBridge struct{}

func (*recordingBridge) Publish(context.Context, testpilot.Coordinate, string, testpilot.OpaqueHandle) error {
	return nil
}
func (*recordingBridge) Await(context.Context, string) error { return nil }
func (*recordingBridge) Consume(context.Context, string) (testpilot.OpaqueHandle, error) {
	return struct{}{}, nil
}
