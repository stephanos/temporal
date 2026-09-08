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

func capabilitySession(t *testing.T, h *Driver, source *testpilotspb.Case, run string) (*Session, testpilot.Coordinate) {
	t.Helper()
	program := proto.CloneOf(source.Program)
	program.Entrypoints = append(program.Entrypoints, &testpilotspb.EntrypointDefinition{EntrypointId: "worker", Activation: &testpilotspb.EntrypointDefinition_Workflow{Workflow: &testpilotspb.WorkflowActivation{}}})
	program.Slots = append(program.Slots, &testpilotspb.SlotDefinition{SlotId: "capability", Content: &testpilotspb.SlotDefinition_OpaqueCapability{OpaqueCapability: &testpilotspb.OpaqueCapabilityType{}}})
	s, err := h.open(t.Context(), run, program)
	require.NoError(t, err)
	return s, testpilot.Coordinate{RunID: run, EntrypointID: "worker", ActivationID: "activation", InstructionID: "publish", Attempt: 1}
}

type capabilityEffectFunc func(context.Context, proto.Message, int64) testpilot.EffectResult

func (capabilityEffectFunc) Accepts(context.Context, *testpilotspb.Instruction, proto.Message) bool {
	return true
}
func (f capabilityEffectFunc) Invoke(ctx context.Context, input proto.Message, limit int64) testpilot.EffectResult {
	return f(ctx, input, limit)
}

func successfulCapabilityEffect() testpilot.CapabilityEffect {
	return capabilityEffectFunc(func(context.Context, proto.Message, int64) testpilot.EffectResult {
		return testpilot.EffectResult{Outcome: &testpilotspb.InstructionOutcome{Status: testpilotspb.INSTRUCTION_OUTCOME_STATUS_SUCCEEDED, ProtocolCode: "ok"}}
	})
}

func capabilityValue() *testpilotspb.Value {
	return &testpilotspb.Value{Value: &testpilotspb.Value_Text{Text: "result"}}
}

func TestOpaqueCapabilityOwnershipAndInvocation(t *testing.T) {
	h, source, _ := fixture(t, "127.0.0.1:1")
	s, origin := capabilitySession(t, h, source, "run")
	var captured proto.Message
	capability, err := s.NewCapability(t.Context(), origin, capabilityEffectFunc(func(_ context.Context, input proto.Message, _ int64) testpilot.EffectResult {
		captured = input
		return successfulCapabilityEffect().Invoke(t.Context(), input, 0)
	}))
	require.NoError(t, err)
	require.NoError(t, s.Publish(t.Context(), origin, "capability", capability))
	require.NoError(t, s.Publish(t.Context(), origin, "capability", capability))
	claim, err := s.Consume(t.Context(), "capability")
	require.NoError(t, err)

	foreign, _ := capabilitySession(t, h, source, "foreign")
	denied, err := foreign.InvokeCapability(t.Context(), coordinate("foreign", "check"), claim, capabilityValue())
	require.Error(t, err)
	require.Nil(t, denied)

	input := capabilityValue()
	handle, err := s.InvokeCapability(t.Context(), coordinate("run", "check"), claim, input)
	require.NoError(t, err)
	input.Value = &testpilotspb.Value_Text{Text: "changed"}
	result, err := handle.Wait(t.Context())
	require.NoError(t, err)
	require.Equal(t, testpilotspb.INSTRUCTION_OUTCOME_STATUS_SUCCEEDED, result.Outcome.Status)
	require.Equal(t, "result", captured.(*testpilotspb.Value).GetText())

	denied, err = s.InvokeCapability(t.Context(), coordinate("run", "check"), claim, capabilityValue())
	require.Error(t, err)
	require.Nil(t, denied)
}

func TestCapabilityClosureAndLimits(t *testing.T) {
	h, source, _ := fixture(t, "127.0.0.1:1")
	s, origin := capabilitySession(t, h, source, "run")
	capability, err := s.NewCapability(t.Context(), origin, successfulCapabilityEffect())
	require.NoError(t, err)
	s.host.profile.ProgramLimits.MaxActivations = 1
	denied, err := s.NewCapability(t.Context(), origin, successfulCapabilityEffect())
	require.ErrorIs(t, err, errCapacity)
	require.Nil(t, denied)
	require.NoError(t, s.Publish(t.Context(), origin, "capability", capability))
	require.NoError(t, s.Close(t.Context()))
	require.Nil(t, capability.(*opaqueCapability).invoke)
	require.Empty(t, s.capabilities)
	require.Empty(t, s.slots)
}

func TestCapabilityQuarantineRetainsCapacityUntilEffectReturns(t *testing.T) {
	h, source, _ := fixture(t, "127.0.0.1:1")
	entered, release := make(chan struct{}), make(chan struct{})
	s, origin := capabilitySession(t, h, source, "run")
	capability, err := s.NewCapability(t.Context(), origin, capabilityEffectFunc(func(ctx context.Context, _ proto.Message, _ int64) testpilot.EffectResult {
		close(entered)
		<-release
		return testpilot.EffectResult{Outcome: &testpilotspb.InstructionOutcome{Status: testpilotspb.INSTRUCTION_OUTCOME_STATUS_CANCELED, ProtocolCode: ctx.Err().Error()}}
	}))
	require.NoError(t, err)
	require.NoError(t, s.Publish(t.Context(), origin, "capability", capability))
	claim, err := s.Consume(t.Context(), "capability")
	require.NoError(t, err)
	h.profile.ProgramLimits.MaxAttempts = 1
	handle, err := s.InvokeCapability(t.Context(), coordinate("run", "check"), claim, capabilityValue())
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

type blockingCapabilityEffect struct {
	entered chan struct{}
	release chan struct{}
}

func (e blockingCapabilityEffect) Accepts(ctx context.Context, _ *testpilotspb.Instruction, _ proto.Message) bool {
	close(e.entered)
	select {
	case <-ctx.Done():
		return false
	case <-e.release:
		return true
	}
}

func (blockingCapabilityEffect) Invoke(context.Context, proto.Message, int64) testpilot.EffectResult {
	return successfulCapabilityEffect().Invoke(context.Background(), capabilityValue(), 0)
}

func TestCapabilityContractCheckDoesNotHoldDriverLock(t *testing.T) {
	h, source, _ := fixture(t, "127.0.0.1:1")
	s, origin := capabilitySession(t, h, source, "run")
	other, _ := capabilitySession(t, h, source, "other")
	effect := blockingCapabilityEffect{entered: make(chan struct{}), release: make(chan struct{})}
	capability, err := s.NewCapability(t.Context(), origin, effect)
	require.NoError(t, err)
	require.NoError(t, s.Publish(t.Context(), origin, "capability", capability))
	claim, err := s.Consume(t.Context(), "capability")
	require.NoError(t, err)
	result := make(chan error, 1)
	go func() {
		_, err := s.InvokeCapability(t.Context(), coordinate("run", "check"), claim, capabilityValue())
		result <- err
	}()
	<-effect.entered
	ctx, cancel := context.WithTimeout(t.Context(), time.Second)
	defer cancel()
	require.NoError(t, other.Close(ctx))
	close(effect.release)
	require.NoError(t, <-result)
}
