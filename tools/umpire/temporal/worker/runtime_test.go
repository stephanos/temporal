package worker

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	umpirespb "go.temporal.io/server/api/umpire/v1"
	"go.temporal.io/server/tools/umpire"
	"go.temporal.io/server/tools/umpire/temporal/internal/delivery"
)

func TestRegistrationRejectsIncompatibleQueueBeforeStart(t *testing.T) {
	starts := 0
	registry := newWorkerRegistry(2, func(queue string, registration queueRegistration) (managedWorker, error) {
		return &fakeManagedWorker{start: func() error { starts++; return nil }}, nil
	})

	release, err := registry.acquire(t.Context(), "run-1", []queueRegistration{{queue: "queue", workflows: []string{"workflow"}, nexus: []nexusRegistration{{service: "service", operation: "operation"}}}}, nil)
	require.NoError(t, err)
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		require.NoError(t, release(ctx))
	})
	_, err = registry.acquire(t.Context(), "run-2", []queueRegistration{{queue: "queue", workflows: []string{"other"}, nexus: []nexusRegistration{{service: "service", operation: "operation"}}}}, nil)
	require.ErrorIs(t, err, ErrRegistrationConflict)
	require.Equal(t, 1, starts)
}

func TestRegistrationSharesExactSignatureAndBoundsRetainedStates(t *testing.T) {
	starts := 0
	registry := newWorkerRegistry(1, func(queue string, registration queueRegistration) (managedWorker, error) {
		return &fakeManagedWorker{start: func() error { starts++; return nil }}, nil
	})
	requirements := []queueRegistration{{queue: "queue", workflows: []string{"workflow"}}}
	release1, err := registry.acquire(t.Context(), "run-1", requirements, nil)
	require.NoError(t, err)
	release2, err := registry.acquire(t.Context(), "run-2", requirements, nil)
	require.NoError(t, err)
	require.Equal(t, 1, starts)
	require.NoError(t, release1(t.Context()))
	require.NoError(t, release2(t.Context()))
	_, err = registry.acquire(t.Context(), "run-3", []queueRegistration{{queue: "other", workflows: []string{"workflow"}}}, nil)
	require.ErrorIs(t, err, ErrCapacity)
}

func TestRegistrationWaitAndDuplicateAcquisitionAreContextBounded(t *testing.T) {
	started := make(chan struct{})
	proceed := make(chan struct{})
	registry := newWorkerRegistry(1, func(string, queueRegistration) (managedWorker, error) {
		return &fakeManagedWorker{start: func() error {
			close(started)
			<-proceed
			return nil
		}}, nil
	})
	requirements := []queueRegistration{{queue: "queue", workflows: []string{"workflow"}}}
	acquired := make(chan error, 1)
	go func() {
		_, err := registry.acquire(context.Background(), "run-1", requirements, nil)
		acquired <- err
	}()
	<-started
	canceled, cancel := context.WithCancel(t.Context())
	cancel()
	_, err := registry.acquire(canceled, "run-2", requirements, nil)
	require.ErrorIs(t, err, context.Canceled)
	close(proceed)
	require.NoError(t, <-acquired)
	_, err = registry.acquire(t.Context(), "run-1", requirements, nil)
	require.ErrorIs(t, err, ErrRegistrationConflict)
}

func TestRegistrationFailureStopsOnlyStartedWorkersAndNotifiesDependents(t *testing.T) {
	first := &fakeManagedWorker{start: func() error { return nil }}
	second := &fakeManagedWorker{start: func() error { return errors.New("start failed") }}
	registry := newWorkerRegistry(2, func(queue string, _ queueRegistration) (managedWorker, error) {
		if queue == "a" {
			return first, nil
		}
		return second, nil
	})
	_, err := registry.acquire(t.Context(), "run", []queueRegistration{{queue: "a", workflows: []string{"workflow"}}, {queue: "b", workflows: []string{"workflow"}}}, nil)
	require.EqualError(t, err, "start failed")
	require.Equal(t, 1, first.stops)
	require.Zero(t, second.stops)

	registry = newWorkerRegistry(2, func(string, queueRegistration) (managedWorker, error) {
		return &fakeManagedWorker{start: func() error { return nil }}, nil
	})
	failures := make(chan string, 2)
	_, err = registry.acquire(t.Context(), "run-a", []queueRegistration{{queue: "a", workflows: []string{"workflow"}}}, func(queue string, _ error) { failures <- queue })
	require.NoError(t, err)
	_, err = registry.acquire(t.Context(), "run-b", []queueRegistration{{queue: "b", workflows: []string{"workflow"}}}, func(queue string, _ error) { failures <- queue })
	require.NoError(t, err)
	registry.fail("a", errors.New("fatal"))
	require.Equal(t, "a", <-failures)
	require.Empty(t, failures)
}

func TestRegistrationBuildsEveryWorkerBeforeStartingAny(t *testing.T) {
	starts := 0
	registry := newWorkerRegistry(2, func(queue string, _ queueRegistration) (managedWorker, error) {
		if queue == "b" {
			return nil, errors.New("registration failed")
		}
		return &fakeManagedWorker{start: func() error { starts++; return nil }}, nil
	})
	_, err := registry.acquire(t.Context(), "run", []queueRegistration{{queue: "a", workflows: []string{"workflow"}}, {queue: "b", workflows: []string{"workflow"}}}, nil)
	require.EqualError(t, err, "registration failed")
	require.Zero(t, starts)
}

func TestRegistrationUsesStructuralNexusSignatures(t *testing.T) {
	starts := 0
	registry := newWorkerRegistry(1, func(string, queueRegistration) (managedWorker, error) {
		return &fakeManagedWorker{start: func() error { starts++; return nil }}, nil
	})
	release, err := registry.acquire(t.Context(), "run-a", []queueRegistration{{queue: "queue", nexus: []nexusRegistration{{service: "a/b", operation: "c"}}}}, nil)
	require.NoError(t, err)
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		require.NoError(t, release(ctx))
	})
	_, err = registry.acquire(t.Context(), "run-b", []queueRegistration{{queue: "queue", nexus: []nexusRegistration{{service: "a", operation: "b/c"}}}}, nil)
	require.ErrorIs(t, err, ErrRegistrationConflict)
	require.Equal(t, 1, starts)
}

func TestReservationCancelUsesExactAdmittedIdentity(t *testing.T) {
	identity := umpire.ReservationIdentity{Origin: umpire.Coordinate{RunID: "run", EntrypointID: "controller", ActivationID: "controller-1", InstructionID: "start", Attempt: 1}, EntrypointID: "workflow", ID: "reservation", Ordinal: 0}
	reservation := newReservation(identity)
	coordinate, err := reservation.Consume(t.Context())
	require.NoError(t, err)
	require.Equal(t, "reservation", coordinate.ActivationID)

	canceled := false
	require.NoError(t, reservation.bindCancellation("workflow-id", "temporal-run-id", "", func(ctx context.Context, workflowID, runID, requestID string) error {
		canceled = true
		require.Equal(t, "workflow-id", workflowID)
		require.Equal(t, "temporal-run-id", runID)
		require.Empty(t, requestID)
		return nil
	}))
	require.NoError(t, reservation.Cancel(t.Context()))
	require.True(t, canceled)
}

func TestReservationCancelBeforeAdmissionRetiresWithoutTargetCall(t *testing.T) {
	reservation := newReservation(umpire.ReservationIdentity{Origin: umpire.Coordinate{RunID: "run", EntrypointID: "controller", ActivationID: "controller-1", InstructionID: "start", Attempt: 1}, EntrypointID: "handler", ID: "reservation", Ordinal: 0})
	require.NoError(t, reservation.Cancel(t.Context()))
	_, err := reservation.Consume(t.Context())
	require.ErrorIs(t, err, ErrClosed)
	result, err := reservation.Wait(t.Context())
	require.NoError(t, err)
	require.Equal(t, umpirespb.INSTRUCTION_OUTCOME_STATUS_CANCELED, result.Outcome.GetStatus())
}

func TestActivationValuesOwnValidatedOutcome(t *testing.T) {
	values := newActivationValues("workflow", 8)
	original := &umpirespb.Value{Value: &umpirespb.Value_Text{Text: "result"}}
	values.store("await", &umpire.OutcomeSnapshot{Outcome: &umpirespb.InstructionOutcome{Status: umpirespb.INSTRUCTION_OUTCOME_STATUS_SUCCEEDED, Value: original}, Fields: map[umpirespb.InstructionOutcomeField]*umpirespb.Value{umpirespb.INSTRUCTION_OUTCOME_FIELD_VALUE: original}})
	original.Value = &umpirespb.Value_Text{Text: "mutated"}
	reference := umpire.ValueReference{Kind: umpire.OutcomeReference, Entrypoint: "workflow", ID: "await", Field: int32(umpirespb.INSTRUCTION_OUTCOME_FIELD_VALUE)}
	require.Equal(t, "result", values.lookup(reference).GetText())
	require.Nil(t, values.lookup(umpire.ValueReference{Kind: umpire.SlotReference, ID: "private-capability"}))
}

func TestReservationBindingRejectsCrossedIdentity(t *testing.T) {
	reservation := newReservation(umpire.ReservationIdentity{Origin: umpire.Coordinate{RunID: "run", EntrypointID: "controller", ActivationID: "controller-1", InstructionID: "start", Attempt: 1}, EntrypointID: "handler", ID: "reservation", Ordinal: 0})
	_, err := reservation.Consume(t.Context())
	require.NoError(t, err)
	err = reservation.bindCancellation("workflow", "run", "request", func(context.Context, string, string, string) error { return errors.New("unexpected") })
	require.ErrorIs(t, err, ErrInvalid)
}

func TestReservationCancellationRetriesAndDrainOnlyWaitsForTerminality(t *testing.T) {
	reservation := newReservation(umpire.ReservationIdentity{Origin: umpire.Coordinate{RunID: "run", EntrypointID: "controller", ActivationID: "controller-1", InstructionID: "start", Attempt: 1}, EntrypointID: "workflow", ID: "reservation", Ordinal: 0})
	_, err := reservation.Consume(t.Context())
	require.NoError(t, err)
	attempts := 0
	require.NoError(t, reservation.bindCancellation("workflow", "temporal-run", "", func(context.Context, string, string, string) error {
		attempts++
		if attempts == 1 {
			return errors.New("transient")
		}
		return nil
	}))
	require.EqualError(t, reservation.Cancel(t.Context()), "transient")
	require.NoError(t, reservation.Cancel(t.Context()))
	require.Equal(t, 2, attempts)
	reservation.finish(umpire.EffectResult{}, errors.New("activation failed"))
	require.NoError(t, reservation.Drain(t.Context()))
}

func TestReservationCancelWaitsForExactBinding(t *testing.T) {
	reservation := newReservation(umpire.ReservationIdentity{Origin: umpire.Coordinate{RunID: "run", EntrypointID: "controller", ActivationID: "controller-1", InstructionID: "start", Attempt: 1}, EntrypointID: "workflow", ID: "reservation", Ordinal: 0})
	_, err := reservation.Consume(t.Context())
	require.NoError(t, err)
	canceled := make(chan struct{})
	result := make(chan error, 1)
	go func() { result <- reservation.Cancel(context.Background()) }()
	require.NoError(t, reservation.bindCancellation("workflow", "temporal-run", "", func(_ context.Context, workflowID, runID, requestID string) error {
		require.Equal(t, "workflow", workflowID)
		require.Equal(t, "temporal-run", runID)
		require.Empty(t, requestID)
		close(canceled)
		return nil
	}))
	<-canceled
	require.NoError(t, <-result)
}

func TestSessionCloseRetriesCancellationBeforeRelease(t *testing.T) {
	prepared := preparedRuntimeFixture(t, umpirespb.NEXUS_RESPONSE_KIND_SYNCHRONOUS)
	host, definition := runtimeTestHost(t, prepared)
	session, err := newSession(host, "run", "session", definition, SessionOptions{Bridge: newTestBridge()})
	require.NoError(t, err)
	require.NoError(t, host.mu.lock(t.Context()))
	host.sessions[session.runID] = session
	host.mu.unlock()
	handles, err := session.Reserve(t.Context(), umpire.ReservationRequest{
		Origin:       umpire.Coordinate{RunID: "run", EntrypointID: "controller", ActivationID: "controller", InstructionID: "call", Attempt: 1},
		EntrypointID: "workflow", Count: 1,
	})
	require.NoError(t, err)
	raw := session.reservations[handles[0].Identity().ID]
	_, err = raw.Consume(t.Context())
	require.NoError(t, err)
	attempts := 0
	require.NoError(t, raw.bindCancellation("workflow", "temporal-run", "", func(context.Context, string, string, string) error {
		attempts++
		if attempts == 1 {
			return errors.New("transient")
		}
		return nil
	}))

	require.ErrorIs(t, session.Close(t.Context()), delivery.ErrLifecycle)
	require.Same(t, session, host.sessions[session.runID])
	require.NoError(t, session.Close(t.Context()))
	require.Equal(t, 2, attempts)
	require.Nil(t, host.sessions[session.runID])
}

type fakeManagedWorker struct {
	start func() error
	stops int
}

func (w *fakeManagedWorker) Start() error { return w.start() }
func (w *fakeManagedWorker) Stop()        { w.stops++ }
