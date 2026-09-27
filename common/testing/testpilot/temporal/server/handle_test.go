package server

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/testpilot"
	"google.golang.org/protobuf/proto"
)

func handleSession(t *testing.T, h *Driver, source *testpilotspb.Case, run string) (*Session, testpilot.Coordinate) {
	t.Helper()
	program := proto.CloneOf(source.Program)
	program.Entrypoints = append(program.Entrypoints, &testpilotspb.Entrypoint{EntrypointId: "worker", Activation: &testpilotspb.Entrypoint_Workflow{Workflow: &testpilotspb.WorkflowActivation{}}})
	program.Slots = append(program.Slots, &testpilotspb.Slot{SlotId: "handle", Content: &testpilotspb.Slot_OpaqueHandle{OpaqueHandle: &testpilotspb.OpaqueHandleType{}}})
	s, err := h.open(t.Context(), run, program, h.profile.ProgramLimits)
	require.NoError(t, err)
	return s, testpilot.Coordinate{RunID: run, EntrypointID: "worker", ActivationID: "activation", InstructionID: "publish", Attempt: 1}
}

type handleEffectFunc func(context.Context, proto.Message, int64) testpilot.EffectResult

func (handleEffectFunc) Accepts(context.Context, *testpilotspb.Instruction, proto.Message) bool {
	return true
}
func (f handleEffectFunc) Invoke(ctx context.Context, input proto.Message, limit int64) testpilot.EffectResult {
	return f(ctx, input, limit)
}

func successfulHandleEffect() testpilot.HandleEffect {
	return handleEffectFunc(func(context.Context, proto.Message, int64) testpilot.EffectResult {
		return testpilot.EffectResult{Outcome: &testpilotspb.InstructionOutcome{Status: testpilotspb.INSTRUCTION_OUTCOME_STATUS_SUCCEEDED, ProtocolCode: "ok"}}
	})
}

func handleValue() *testpilotspb.Value {
	return &testpilotspb.Value{Value: &testpilotspb.Value_TextValue{TextValue: "result"}}
}

func TestOpaqueHandleOwnershipAndInvocation(t *testing.T) {
	h, source, _ := fixture(t, "127.0.0.1:1")
	s, origin := handleSession(t, h, source, "run")
	var captured proto.Message
	opaque, err := s.NewHandle(t.Context(), origin, handleEffectFunc(func(_ context.Context, input proto.Message, _ int64) testpilot.EffectResult {
		captured = input
		return successfulHandleEffect().Invoke(t.Context(), input, 0)
	}))
	require.NoError(t, err)
	require.NoError(t, s.Publish(t.Context(), origin, "handle", opaque))
	require.NoError(t, s.Publish(t.Context(), origin, "handle", opaque))
	claim, err := s.Consume(t.Context(), "handle")
	require.NoError(t, err)

	foreign, _ := handleSession(t, h, source, "foreign")
	denied, err := foreign.InvokeHandle(t.Context(), coordinate("foreign", "check"), claim, handleValue())
	require.Error(t, err)
	require.Nil(t, denied)

	input := handleValue()
	handle, err := s.InvokeHandle(t.Context(), coordinate("run", "check"), claim, input)
	require.NoError(t, err)
	input.Value = &testpilotspb.Value_TextValue{TextValue: "changed"}
	result, err := handle.Wait(t.Context())
	require.NoError(t, err)
	require.Equal(t, testpilotspb.INSTRUCTION_OUTCOME_STATUS_SUCCEEDED, result.Outcome.Status)
	require.Equal(t, "result", captured.(*testpilotspb.Value).GetTextValue())

	denied, err = s.InvokeHandle(t.Context(), coordinate("run", "check"), claim, handleValue())
	require.Error(t, err)
	require.Nil(t, denied)
}

func TestHandleClosureAndLimits(t *testing.T) {
	h, source, _ := fixture(t, "127.0.0.1:1")
	s, origin := handleSession(t, h, source, "run")
	handle, err := s.NewHandle(t.Context(), origin, successfulHandleEffect())
	require.NoError(t, err)
	s.host.profile.ProgramLimits.MaxActivations = 1
	denied, err := s.NewHandle(t.Context(), origin, successfulHandleEffect())
	require.ErrorIs(t, err, errCapacity)
	require.Nil(t, denied)
	require.NoError(t, s.Publish(t.Context(), origin, "handle", handle))
	require.NoError(t, s.Close(t.Context()))
	require.Nil(t, handle.(*opaqueHandle).invoke)
	require.Empty(t, s.handles)
	require.Empty(t, s.slots)
}

func TestHandleQuarantineRetainsCapacityUntilEffectReturns(t *testing.T) {
	h, source, _ := fixture(t, "127.0.0.1:1")
	entered, release := make(chan struct{}), make(chan struct{})
	s, origin := handleSession(t, h, source, "run")
	opaque, err := s.NewHandle(t.Context(), origin, handleEffectFunc(func(ctx context.Context, _ proto.Message, _ int64) testpilot.EffectResult {
		close(entered)
		<-release
		return testpilot.EffectResult{Outcome: &testpilotspb.InstructionOutcome{Status: testpilotspb.INSTRUCTION_OUTCOME_STATUS_CANCELED, ProtocolCode: ctx.Err().Error()}}
	}))
	require.NoError(t, err)
	require.NoError(t, s.Publish(t.Context(), origin, "handle", opaque))
	claim, err := s.Consume(t.Context(), "handle")
	require.NoError(t, err)
	h.profile.ProgramLimits.MaxAttempts = 1
	handle, err := s.InvokeHandle(t.Context(), coordinate("run", "check"), claim, handleValue())
	require.NoError(t, err)
	<-entered
	require.NoError(t, handle.Cancel(t.Context()))
	ctx, cancel := context.WithTimeout(t.Context(), time.Millisecond)
	defer cancel()
	_, err = handle.Wait(ctx)
	require.ErrorIs(t, err, context.DeadlineExceeded)
	require.ErrorIs(t, handle.Drain(ctx), context.DeadlineExceeded)
	require.NoError(t, s.Quarantine(t.Context(), handle))
	require.NoError(t, s.Close(t.Context()))
	h.mu.Lock()
	require.EqualValues(t, 1, h.effects)
	h.mu.Unlock()
	close(release)
	require.NoError(t, handle.Drain(t.Context()))
	result, err := handle.Wait(t.Context())
	require.NoError(t, err)
	require.Equal(t, testpilotspb.INSTRUCTION_OUTCOME_STATUS_CANCELED, result.Outcome.Status)
	h.mu.Lock()
	require.Zero(t, h.effects)
	h.mu.Unlock()
}

type blockingHandleEffect struct {
	entered chan struct{}
	release chan struct{}
}

func (e blockingHandleEffect) Accepts(ctx context.Context, _ *testpilotspb.Instruction, _ proto.Message) bool {
	close(e.entered)
	select {
	case <-ctx.Done():
		return false
	case <-e.release:
		return true
	}
}

func (blockingHandleEffect) Invoke(context.Context, proto.Message, int64) testpilot.EffectResult {
	return successfulHandleEffect().Invoke(context.Background(), handleValue(), 0)
}

func TestHandleContractCheckDoesNotHoldDriverLock(t *testing.T) {
	h, source, _ := fixture(t, "127.0.0.1:1")
	s, origin := handleSession(t, h, source, "run")
	other, _ := handleSession(t, h, source, "other")
	effect := blockingHandleEffect{entered: make(chan struct{}), release: make(chan struct{})}
	handle, err := s.NewHandle(t.Context(), origin, effect)
	require.NoError(t, err)
	require.NoError(t, s.Publish(t.Context(), origin, "handle", handle))
	claim, err := s.Consume(t.Context(), "handle")
	require.NoError(t, err)
	result := make(chan error, 1)
	go func() {
		_, err := s.InvokeHandle(t.Context(), coordinate("run", "check"), claim, handleValue())
		result <- err
	}()
	<-effect.entered
	ctx, cancel := context.WithTimeout(t.Context(), time.Second)
	defer cancel()
	require.NoError(t, other.Close(ctx))
	close(effect.release)
	require.NoError(t, <-result)
}
