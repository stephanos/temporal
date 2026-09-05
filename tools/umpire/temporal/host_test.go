package temporal

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.temporal.io/api/workflowservice/v1"
	umpirespb "go.temporal.io/server/api/umpire/v1"
	"go.temporal.io/server/tools/umpire"
	"go.temporal.io/server/tools/umpire/temporal/internal/delivery"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

func TestCompositeSessionKeepsControllerAndWorkerAuthoritySeparate(t *testing.T) {
	controller := &recordingControllerSession{bridge: &recordingBridge{}}
	workers := &recordingWorkerSession{}
	session := newCompositeSession(controller, workers, umpire.PreparedProgram{})
	origin := umpire.Coordinate{RunID: "run", EntrypointID: "controller", ActivationID: "activation", InstructionID: "call", Attempt: 1}

	reservations, err := session.Reserve(t.Context(), umpire.ReservationRequest{Origin: origin, EntrypointID: "workflow", Count: 1})
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

type recordingControllerSession struct {
	bridge      umpire.CapabilityBridge
	invocations int
	quarantines int
	closes      []string
}

func (*recordingControllerSession) Reserve(context.Context, umpire.ReservationRequest) ([]umpire.ReservationHandle, error) {
	panic("controller cannot reserve worker authority")
}
func (s *recordingControllerSession) InvokeRPC(context.Context, umpire.Coordinate, string, protoreflect.MethodDescriptor, proto.Message) (umpire.EffectHandle, error) {
	s.invocations++
	return recordingEffect{}, nil
}
func (*recordingControllerSession) CompleteNexusOperation(context.Context, umpire.Coordinate, umpire.OpaqueCapability, *umpirespb.Value) (umpire.EffectHandle, error) {
	return recordingEffect{}, nil
}
func (s *recordingControllerSession) Bridge(context.Context) (umpire.CapabilityBridge, error) {
	return s.bridge, nil
}
func (s *recordingControllerSession) Quarantine(context.Context, umpire.EffectHandle) error {
	s.quarantines++
	return nil
}
func (s *recordingControllerSession) Close(context.Context) error {
	s.closes = append(s.closes, "controller")
	return nil
}
func (*recordingControllerSession) Diagnose(context.Context, string, *umpirespb.RunDiagnostic) error {
	return nil
}

type recordingWorkerSession struct {
	reserves    int
	quarantines int
	closes      []string
}

func (s *recordingWorkerSession) Reserve(context.Context, umpire.ReservationRequest) ([]umpire.ReservationHandle, error) {
	s.reserves++
	return []umpire.ReservationHandle{recordingReservation{}}, nil
}
func (s *recordingWorkerSession) Close(context.Context) error {
	s.closes = append(s.closes, "worker")
	return nil
}
func (s *recordingWorkerSession) Quarantine(context.Context, umpire.EffectHandle) error {
	s.quarantines++
	return nil
}
func (*recordingWorkerSession) Diagnose(context.Context, string, *umpirespb.RunDiagnostic) error {
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
	require.Equal(t, []delivery.TriggerDisposition{delivery.TriggerUncertain}, terminal.dispositions)
	require.Equal(t, []error{nil}, terminal.contextErrors)
	require.True(t, terminal.admissible)

	require.NoError(t, effect.Cancel(t.Context()))
	require.Equal(t, []delivery.TriggerDisposition{delivery.TriggerUncertain, delivery.TriggerUncertain}, terminal.dispositions)
	require.Equal(t, []error{nil, nil}, terminal.contextErrors)
	require.False(t, terminal.admissible)
}

func TestCarrierEffectDrainFinalizesCompletedResult(t *testing.T) {
	done := make(chan struct{})
	close(done)
	terminal := &recordingTerminalCarrier{admissible: true}
	effect := &carrierEffect{
		EffectHandle: &terminalEffect{done: done, result: umpire.EffectResult{
			Outcome: &umpirespb.InstructionOutcome{Status: umpirespb.INSTRUCTION_OUTCOME_STATUS_PROTOCOL_NON_SUCCESS},
		}},
		carrier: terminal, cleanupTimeout: time.Second,
	}
	require.NoError(t, effect.Drain(t.Context()))
	require.Equal(t, []delivery.TriggerDisposition{delivery.TriggerNonSuccess}, terminal.dispositions)
	require.False(t, terminal.admissible)
}

type recordingTerminalCarrier struct {
	failures       int
	admissible     bool
	dispositions   []delivery.TriggerDisposition
	contextErrors  []error
	pinnedResponse *workflowservice.StartWorkflowExecutionResponse
}

func (c *recordingTerminalCarrier) PinStartResponse(ctx context.Context, response *workflowservice.StartWorkflowExecutionResponse) error {
	c.contextErrors = append(c.contextErrors, ctx.Err())
	c.pinnedResponse = proto.CloneOf(response)
	return nil
}
func (c *recordingTerminalCarrier) TriggerTerminal(ctx context.Context, disposition delivery.TriggerDisposition) (int, error) {
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
	result umpire.EffectResult
}

func (e *terminalEffect) Wait(ctx context.Context) (umpire.EffectResult, error) {
	select {
	case <-e.done:
		return e.result, nil
	case <-ctx.Done():
		return umpire.EffectResult{}, ctx.Err()
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

func (e *blockingEffect) Wait(ctx context.Context) (umpire.EffectResult, error) {
	select {
	case <-e.done:
		return umpire.EffectResult{}, errors.New("terminal worker failure")
	case <-ctx.Done():
		return umpire.EffectResult{}, ctx.Err()
	}
}
func (*blockingEffect) Cancel(context.Context) error { return nil }
func (e *blockingEffect) Drain(ctx context.Context) error {
	_, err := e.Wait(ctx)
	return err
}

type recordingEffect struct{}

func (recordingEffect) Wait(context.Context) (umpire.EffectResult, error) {
	return umpire.EffectResult{}, nil
}
func (recordingEffect) Cancel(context.Context) error { return nil }
func (recordingEffect) Drain(context.Context) error  { return nil }

type recordingReservation struct{ recordingEffect }

func (recordingReservation) Identity() umpire.ReservationIdentity {
	return umpire.ReservationIdentity{}
}
func (recordingReservation) Consume(context.Context) (umpire.Coordinate, error) {
	return umpire.Coordinate{}, nil
}

type recordingBridge struct{}

func (*recordingBridge) Publish(context.Context, umpire.Coordinate, string, umpire.OpaqueCapability) error {
	return nil
}
func (*recordingBridge) Await(context.Context, string) error { return nil }
func (*recordingBridge) Consume(context.Context, string) (umpire.OpaqueCapability, error) {
	return struct{}{}, nil
}
