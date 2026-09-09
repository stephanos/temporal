package worker

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/testpilot"
)

func faultRequirements() []queueRegistration {
	return []queueRegistration{{queue: "queue", workflows: []string{"workflow"}}}
}

// A dedicated group is the whole isolation mechanism: two Runs on the same task queue share one
// pooled worker, and the Run that injects faults must not be sharing the worker it stops.
func TestFaultRunHoldsItsOwnWorkerGroup(t *testing.T) {
	built := 0
	registry := newWorkerRegistry(4, func(string, string, queueRegistration) (managedWorker, error) {
		built++
		return &fakeManagedWorker{start: func() error { return nil }}, nil
	})
	pooledOne, err := registry.acquire(t.Context(), "plain-1", faultRequirements(), false, nil)
	require.NoError(t, err)
	pooledTwo, err := registry.acquire(t.Context(), "plain-2", faultRequirements(), false, nil)
	require.NoError(t, err)
	fault, err := registry.acquire(t.Context(), "fault", faultRequirements(), true, nil)
	require.NoError(t, err)

	// Two pooled Runs share one worker; the fault Run built its own.
	require.Equal(t, 2, built)
	require.Len(t, registry.groups, 2)
	require.NotNil(t, registry.groups["queue"])
	require.NotNil(t, registry.groups[groupKey("fault", "queue", true)])

	pooled := registry.groups["queue"].worker.(*fakeManagedWorker)
	require.NoError(t, fault.stopWorker(t.Context()))
	// The stop reached only the fault Run's own worker.
	require.Equal(t, 1, registry.groups[groupKey("fault", "queue", true)].worker.(*fakeManagedWorker).stops)
	require.Zero(t, pooled.stops)
	require.True(t, registry.groups[groupKey("fault", "queue", true)].stopped)

	require.NoError(t, pooledOne.release(t.Context()))
	require.NoError(t, pooledTwo.release(t.Context()))
	require.NoError(t, fault.release(t.Context()))
	// The dedicated group leaves with its Run; the pooled group survives its last release.
	require.Nil(t, registry.groups[groupKey("fault", "queue", true)])
	require.NotNil(t, registry.groups["queue"])
}

func TestFaultStopAndResumeKeepTheSameRegistration(t *testing.T) {
	var registrations []queueRegistration
	registry := newWorkerRegistry(2, func(_, _ string, registration queueRegistration) (managedWorker, error) {
		registrations = append(registrations, registration)
		return &fakeManagedWorker{start: func() error { return nil }}, nil
	})
	lease, err := registry.acquire(t.Context(), "fault", faultRequirements(), true, nil)
	require.NoError(t, err)
	group := registry.groups[groupKey("fault", "queue", true)]
	first := group.worker

	require.NoError(t, lease.stopWorker(t.Context()))
	require.NoError(t, lease.resumeWorker(t.Context()))
	require.False(t, group.stopped)
	require.NotSame(t, first, group.worker)
	require.Len(t, registrations, 2)
	require.True(t, registrations[0].compatible(registrations[1]))

	// Both transitions are invariants, not idempotent requests.
	require.ErrorIs(t, lease.resumeWorker(t.Context()), ErrRegistrationConflict)
	require.NoError(t, lease.stopWorker(t.Context()))
	require.ErrorIs(t, lease.stopWorker(t.Context()), ErrRegistrationConflict)
	require.NoError(t, lease.release(t.Context()))
}

// A stop the Run asked for is not a Run failure, so the SDK's fatal path stays suppressed for the
// whole window rather than only for the instant the worker was stopping.
func TestFaultStopSuppressesTheFatalPath(t *testing.T) {
	failures := make(chan string, 4)
	registry := newWorkerRegistry(2, func(string, string, queueRegistration) (managedWorker, error) {
		return &fakeManagedWorker{start: func() error { return nil }}, nil
	})
	lease, err := registry.acquire(t.Context(), "fault", faultRequirements(), true, func(queue string, _ error) { failures <- queue })
	require.NoError(t, err)
	key := groupKey("fault", "queue", true)

	require.NoError(t, lease.stopWorker(t.Context()))
	registry.fail(key, errors.New("worker stopped"))
	require.Empty(t, failures)
	require.Nil(t, registry.groups[key].failure)

	require.NoError(t, lease.resumeWorker(t.Context()))
	registry.fail(key, errors.New("real failure"))
	require.Equal(t, "queue", <-failures)
	require.NoError(t, lease.release(t.Context()))
}

func TestFaultStopHonorsTheInstructionDeadline(t *testing.T) {
	release := make(chan struct{})
	t.Cleanup(func() { close(release) })
	registry := newWorkerRegistry(2, func(string, string, queueRegistration) (managedWorker, error) {
		return &blockingManagedWorker{release: release}, nil
	})
	lease, err := registry.acquire(t.Context(), "fault", faultRequirements(), true, nil)
	require.NoError(t, err)
	ctx, cancel := context.WithTimeout(t.Context(), 20*time.Millisecond)
	defer cancel()
	require.ErrorIs(t, lease.stopWorker(ctx), context.DeadlineExceeded)
}

// Release is the cleanup boundary: a stopped worker is resumed before the group goes away, and a
// resume that cannot complete is reported rather than swallowed.
func TestFaultReleaseResumesBeforeReleasing(t *testing.T) {
	for _, tc := range []struct {
		name      string
		startErr  error
		wantError bool
	}{
		{"resumed", nil, false},
		{"resume failed", errors.New("cannot re-register"), true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			starts := 0
			registry := newWorkerRegistry(2, func(string, string, queueRegistration) (managedWorker, error) {
				return &fakeManagedWorker{start: func() error {
					starts++
					if starts > 1 {
						return tc.startErr
					}
					return nil
				}}, nil
			})
			lease, err := registry.acquire(t.Context(), "fault", faultRequirements(), true, nil)
			require.NoError(t, err)
			require.NoError(t, lease.stopWorker(t.Context()))

			err = lease.release(t.Context())
			if tc.wantError {
				require.ErrorIs(t, err, tc.startErr)
				return
			}
			require.NoError(t, err)
			require.Equal(t, 2, starts)
			require.Nil(t, registry.groups[groupKey("fault", "queue", true)])
		})
	}
}

// A pooled lease is not a fault handle at all: nothing outside a dedicated group may be stopped.
func TestFaultTransitionsRequireADedicatedGroup(t *testing.T) {
	registry := newWorkerRegistry(2, func(string, string, queueRegistration) (managedWorker, error) {
		return &fakeManagedWorker{start: func() error { return nil }}, nil
	})
	lease, err := registry.acquire(t.Context(), "plain", faultRequirements(), false, nil)
	require.NoError(t, err)
	require.ErrorIs(t, lease.stopWorker(t.Context()), ErrUnsupportedOperation)
	require.ErrorIs(t, lease.resumeWorker(t.Context()), ErrUnsupportedOperation)
	require.NoError(t, lease.release(t.Context()))
}

// The worker Driver's own Profile validation is what admits the new capability; a Profile that
// authorizes every capability including InjectFault must still build a Driver.
func TestWorkerProfileAdmitsTheFaultCapability(t *testing.T) {
	prepared := preparedSymbolicRuntimeFixture(t)
	profile := symbolicRuntimeDriver(t, prepared.Snapshot().GetLimits()).options.profile
	profile.Capabilities = make([]testpilot.Capability, 0, testpilot.MaxCapability)
	for capability := testpilot.InvokeRPC; capability <= testpilot.MaxCapability; capability++ {
		profile.Capabilities = append(profile.Capabilities, capability)
	}
	require.True(t, validWorkerProfile(profile))
	profile.Capabilities = append(profile.Capabilities, testpilot.InjectFault)
	require.False(t, validWorkerProfile(profile))
}

type blockingManagedWorker struct{ release <-chan struct{} }

func (w *blockingManagedWorker) Start() error { return nil }
func (w *blockingManagedWorker) Stop()        { <-w.release }

// A Program that requests a fault is what makes the Run's group dedicated, so the definition has
// to carry both the flag and the queue the named role resolves to.
func TestPreparedDefinitionCarriesTheDeclaredFaultQueue(t *testing.T) {
	plain := preparedSymbolicRuntimeFixture(t)
	host := symbolicRuntimeDriver(t, plain.Snapshot().GetLimits())
	definition, err := host.prepareDefinition(plain)
	require.NoError(t, err)
	require.False(t, definition.hasFault)
	require.Empty(t, definition.faultQueues)

	withFault := preparedSymbolicRuntimeFixture(t, func(program *testpilotspb.Program) {
		program.Entrypoints[0].Instructions = append(program.Entrypoints[0].Instructions, &testpilotspb.InstructionDefinition{
			InstructionId: "stop",
			Instruction:   &testpilotspb.Instruction{Instruction: &testpilotspb.Instruction_InjectFault{InjectFault: &testpilotspb.InjectFault{RoleId: "queue", Kind: testpilotspb.FAULT_KIND_WORKER_STOP}}},
			Outcome:       runtimeStatusSchema(), Limits: runtimeBounds(),
		})
	}, func(profile *testpilot.ProfileSpec) {
		profile.Capabilities = append(profile.Capabilities, testpilot.InjectFault)
	})
	definition, err = host.prepareDefinition(withFault)
	require.NoError(t, err)
	require.True(t, definition.hasFault)
	require.Equal(t, map[string]string{"queue": "task-queue"}, definition.faultQueues)
}

// The Session is where a fault becomes a Run fact. A transition the Driver cannot make settles as
// a failed instruction outcome plus a Driver invariant diagnostic, so the Run records that the
// fault was requested and not realized and the Verdict is left to the Contract.
func TestSessionInjectFaultReportsUnrealizedTransitions(t *testing.T) {
	prepared := preparedSymbolicRuntimeFixture(t)
	host := symbolicRuntimeDriver(t, prepared.Snapshot().GetLimits())
	registry := newWorkerRegistry(4, func(string, string, queueRegistration) (managedWorker, error) {
		return &fakeManagedWorker{start: func() error { return nil }}, nil
	})
	host.registry = registry
	definition, err := host.prepareDefinition(prepared)
	require.NoError(t, err)
	definition.faultQueues = map[string]string{"queue": "task-queue"}
	definition.hasFault = true

	var diagnostics []*testpilotspb.RunDiagnostic
	session, err := newSession(host, "run", "session-run", definition, SessionOptions{
		Bridge: newTestBridge(),
		Diagnose: func(_ context.Context, _ string, diagnostic *testpilotspb.RunDiagnostic) error {
			diagnostics = append(diagnostics, diagnostic)
			return nil
		},
	})
	require.NoError(t, err)
	lease, err := registry.acquire(t.Context(), "run", definition.registrations, true, nil)
	require.NoError(t, err)
	session.workers = lease
	at := testpilot.Coordinate{RunID: "run", EntrypointID: "controller", ActivationID: "controller-1", InstructionID: "stop", Attempt: 1}

	// A resume with no prior stop is an invariant failure, not a rejected dispatch.
	handle, err := session.InjectFault(t.Context(), at, "queue", testpilotspb.FAULT_KIND_WORKER_RESUME)
	require.NoError(t, err)
	result, err := handle.Wait(t.Context())
	require.NoError(t, err)
	require.Equal(t, testpilotspb.INSTRUCTION_OUTCOME_STATUS_PROTOCOL_NON_SUCCESS, result.Outcome.GetStatus())
	require.Len(t, diagnostics, 1)
	require.Equal(t, testpilotspb.RUN_DIAGNOSTIC_KIND_INVARIANT, diagnostics[0].GetKind())
	require.Equal(t, "fault_not_realized", diagnostics[0].GetCode())

	// The realized transitions settle as succeeded and record nothing further.
	for _, kind := range []testpilotspb.FaultKind{testpilotspb.FAULT_KIND_WORKER_STOP, testpilotspb.FAULT_KIND_WORKER_RESUME} {
		handle, err := session.InjectFault(t.Context(), at, "queue", kind)
		require.NoError(t, err)
		result, err := handle.Wait(t.Context())
		require.NoError(t, err)
		require.Equal(t, testpilotspb.INSTRUCTION_OUTCOME_STATUS_SUCCEEDED, result.Outcome.GetStatus())
	}
	require.Len(t, diagnostics, 1)

	// An undeclared role and an unknown kind are rejected dispatches, never recorded facts.
	_, err = session.InjectFault(t.Context(), at, "other", testpilotspb.FAULT_KIND_WORKER_STOP)
	require.ErrorIs(t, err, ErrInvalid)
	_, err = session.InjectFault(t.Context(), at, "queue", testpilotspb.FAULT_KIND_UNSPECIFIED)
	require.ErrorIs(t, err, ErrInvalid)
	require.NoError(t, lease.release(t.Context()))
}
